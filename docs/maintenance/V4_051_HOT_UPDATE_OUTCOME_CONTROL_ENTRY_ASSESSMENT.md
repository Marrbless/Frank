# V4-051 Hot-Update Outcome Control Entry Assessment

## Current Branch / HEAD / Tags

- Branch: `frank-v4-051-hot-update-outcome-control-entry-assessment`
- HEAD: `ca4e530e7caf1c3f07badf9c02d086d74b97654d`
- Tags at HEAD:
  - `frank-v4-050-hot-update-outcome-from-terminal-gates`

## Repo Baseline

- `git status --short --branch --untracked-files=all` at slice start was clean:
  - `## frank-v4-051-hot-update-outcome-control-entry-assessment`
- Baseline `/usr/local/go/bin/go test -count=1 ./...` passed.

## Scope

This is a docs-only assessment for the smallest safe operator/control entry that can invoke:

- `missioncontrol.CreateHotUpdateOutcomeFromTerminalGate(root, hotUpdateID, createdBy, createdAt)`

No Go code, tests, commands, records, runtime pointers, reload generation, last-known-good pointers, hot-update gates, promotions, or V4-052 work are changed in V4-051.

## Existing Direct Operator Command Path

The current direct operator command path is in `internal/agent/loop.go`.

Hot-update gate commands are parsed with regexes and handled in `ProcessDirect` before provider fallback:

- `HOT_UPDATE_GATE_RECORD <job_id> <hot_update_id> <candidate_pack_id>`
- `HOT_UPDATE_GATE_PHASE <job_id> <hot_update_id> <phase>`
- `HOT_UPDATE_GATE_EXECUTE <job_id> <hot_update_id>`
- `HOT_UPDATE_GATE_RELOAD <job_id> <hot_update_id>`
- `HOT_UPDATE_GATE_FAIL <job_id> <hot_update_id> <reason...>`

The command handlers call `TaskState` wrappers and return deterministic acknowledgements:

- changed path: `Recorded`, `Advanced`, `Executed`, or `Resolved`
- idempotent/select path: `Selected`
- failure path: empty response plus returned error

This is the right surface for the future outcome command because it is already the operator-facing hot-update control lane.

## Existing TaskState Wrapper Pattern

Current hot-update direct commands go through `internal/agent/tools/taskstate.go`.

The wrappers consistently:

- derive `now` from `taskStateTransitionTimestamp(taskStateNowUTC())`
- clone execution/runtime state under lock
- read the mission store root from `TaskState`
- validate the mission store root
- require the command `job_id` to match the active or persisted runtime job
- require active execution context or persisted runtime control context
- call the missioncontrol helper with `createdBy` / `updatedBy` set to `operator`
- emit a runtime control audit event with the command action name
- return the missioncontrol `changed` flag unchanged

Because the direct command path already depends on `TaskState` for job binding, mission store root resolution, timestamps, and audit events, V4-052 should add a small TaskState wrapper even though V4-050 itself correctly stayed missioncontrol-only.

## V4-050 Helper Contract

`CreateHotUpdateOutcomeFromTerminalGate` already provides the storage semantics V4-052 should expose.

Accepted terminal states:

- `reload_apply_succeeded`
- `reload_apply_failed`

Rejected states:

- `prepared`
- `validated`
- `staged`
- `reloading`
- `reload_apply_in_progress`
- `reload_apply_recovery_needed`
- any other non-selected state

Deterministic outcome identity:

- `hot-update-outcome-<hot_update_id>`

Success mapping:

- `outcome_kind`: `hot_updated`
- `reason`: `hot update reload/apply succeeded`

Failure mapping:

- `outcome_kind`: `failed`
- `reason`: copied from `HotUpdateGateRecord.FailureReason`
- empty or whitespace-only failure reason fails closed

Replay behavior:

- first creation writes the outcome and returns `changed=true`
- exact replay returns `changed=false`
- divergent duplicate with the same `outcome_id` fails closed
- existing outcome for the same `hot_update_id` with a different `outcome_id` fails closed
- gate changes that would derive a different outcome fail closed

## Recommended V4-052 Command Shape

Recommended command:

- `HOT_UPDATE_OUTCOME_CREATE <job_id> <hot_update_id>`

Rationale:

- It follows the existing uppercase direct operator command convention.
- It keeps the hot-update lane prefix.
- It names the created ledger object directly.
- It does not ask the operator for an outcome ID, kind, reason, pack ID, or timestamp because V4-050 derives those from the committed gate and helper input.

Required arguments:

- `job_id`: must match the active job or persisted runtime control job, using the same validation as existing hot-update TaskState wrappers.
- `hot_update_id`: identifies the committed terminal hot-update gate to resolve into an outcome.

Rejected arguments:

- caller-provided `outcome_id`
- caller-provided outcome kind
- caller-provided reason
- candidate/run/result refs
- pack refs
- timestamps

Those fields must remain derived, not operator-authored, for this first control entry.

## Recommended V4-052 Implementation Shape

Smallest safe code surface:

- Add `hotUpdateOutcomeCreateCommandRE` in `internal/agent/loop.go`.
- Dispatch it next to the existing `HOT_UPDATE_GATE_*` commands.
- Add `(*TaskState).CreateHotUpdateOutcomeFromTerminalGate(jobID, hotUpdateID string) (bool, error)`.
- Use the same active/persisted runtime validation pattern as the existing hot-update gate wrappers.
- Call `missioncontrol.CreateHotUpdateOutcomeFromTerminalGate(root, hotUpdateID, "operator", now)`.
- Emit runtime control audit action `hot_update_outcome_create`.
- Return deterministic direct responses:
  - changed: `Created hot-update outcome job=<job_id> hot_update=<hot_update_id>.`
  - idempotent: `Selected hot-update outcome job=<job_id> hot_update=<hot_update_id>.`

No new public API outside the direct command and TaskState wrapper is needed.

## Failure Behavior To Preserve

V4-052 should pass through fail-closed helper errors without manufacturing success.

Expected failure cases:

- Missing gate: error from `LoadHotUpdateGateRecord`, response `""`.
- Non-terminal gate: error containing `does not permit outcome creation`, response `""`.
- `reload_apply_failed` with empty `failure_reason`: error containing `failure_reason is required for outcome creation`, response `""`.
- Divergent existing outcome with deterministic `outcome_id`: error containing `hot-update outcome "<outcome_id>" already exists`, response `""`.
- Existing outcome for same `hot_update_id` but different `outcome_id`: error containing `hot_update_id "<hot_update_id>" already exists as "<outcome_id>"`, response `""`.
- Invalid/mismatched job context: same TaskState validation behavior as existing hot-update direct commands.

The command must not infer terminal state from active pointer state, absence of files, or status output.

## Read-Only / Status Expectations After Creation

After successful creation, existing read models should be sufficient:

- `STATUS <job_id>` already includes `hot_update_outcome_identity` through `TaskState` status readout.
- `LoadOperatorHotUpdateOutcomeIdentityStatus` lists outcome records deterministically.
- Outcome status exposes `outcome_id`, `hot_update_id`, optional refs, `candidate_pack_id`, `outcome_kind`, `reason`, `notes`, `outcome_at`, `created_at`, and `created_by`.

V4-052 should test that `STATUS <job_id>` surfaces the newly created deterministic outcome. It should not add a separate status command unless the existing status read model proves insufficient during implementation.

## Required V4-052 Tests

Direct command tests in `internal/agent/loop_processdirect_test.go` should prove:

- `HOT_UPDATE_OUTCOME_CREATE <job_id> <hot_update_id>` creates a `hot_updated` outcome from `reload_apply_succeeded`.
- `HOT_UPDATE_OUTCOME_CREATE <job_id> <hot_update_id>` creates a `failed` outcome from `reload_apply_failed` and copies failure detail.
- Exact replay returns the `Selected hot-update outcome...` acknowledgement and does not rewrite the outcome file.
- Non-terminal states reject with empty direct response and no outcome record.
- Missing gate rejects with empty direct response and no outcome record.
- Failed terminal gate with empty `failure_reason` rejects with empty direct response and no outcome record.
- Divergent duplicate same `outcome_id` rejects with empty direct response.
- Existing outcome for same `hot_update_id` but different `outcome_id` rejects with empty direct response.
- Wrong `job_id` rejects through existing TaskState job validation.
- Successful creation does not create a `PromotionRecord`.
- Active runtime-pack pointer bytes are unchanged.
- `reload_generation` is unchanged.
- `last_known_good_pointer.json` bytes are unchanged.
- Hot-update gate bytes are unchanged.
- No new hot-update gate record is created.
- `STATUS <job_id>` includes the created outcome in `hot_update_outcome_identity`.

TaskState-focused tests in `internal/agent/tools/taskstate_test.go` or the adjacent status/control test file should be added only if the direct command tests do not already lock the wrapper semantics. If added, keep them narrow around job validation, audit event action name, changed flag propagation, and side-effect invariants.

Missioncontrol tests do not need broad expansion in V4-052 because V4-050 already covered the helper behavior directly.

## Non-Goals For V4-052

- no promotion creation
- no active runtime-pack pointer mutation
- no `reload_generation` mutation
- no `last_known_good_pointer.json` mutation
- no hot-update gate mutation
- no new hot-update gate creation
- no terminal-state inference outside the committed gate
- no automatic success or failure inference
- no new outcome mapping
- no policy or authorization broadening beyond the existing TaskState direct command checks
- no V4-053 work

## Recommendation

The smallest safe future slice is V4-052: add `HOT_UPDATE_OUTCOME_CREATE <job_id> <hot_update_id>` to the existing direct operator command path, backed by a minimal TaskState wrapper that calls `CreateHotUpdateOutcomeFromTerminalGate`.

Do not add a separate command family, public API, or manual outcome parameters. The committed gate plus V4-050 helper already define the authoritative outcome mapping.
