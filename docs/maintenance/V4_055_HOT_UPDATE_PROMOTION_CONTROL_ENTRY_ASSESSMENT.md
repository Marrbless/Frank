# V4-055 Hot-Update Promotion Control Entry Assessment

## Current Branch / HEAD / Tags

- Branch: `frank-v4-055-hot-update-promotion-control-entry-assessment`
- HEAD: `655f3b909d38263d4bbaab5f8eae72bb49577d51`
- Tags at HEAD:
  - `frank-v4-054-hot-update-promotion-from-successful-outcome`

## Repo Baseline

- `git status --short --branch --untracked-files=all` at slice start was clean:
  - `## frank-v4-055-hot-update-promotion-control-entry-assessment`
- Baseline `/usr/local/go/bin/go test -count=1 ./...` passed when run with normal test permissions.
- The first sandboxed baseline attempt failed because the Go build cache was read-only and `httptest` could not open local sockets.

## Scope

This is a docs-only assessment for the smallest safe operator/control entry that can invoke:

- `missioncontrol.CreatePromotionFromSuccessfulHotUpdateOutcome(root, outcomeID, createdBy, createdAt)`

No Go code, tests, commands, TaskState wrappers, promotion records, outcome records, active runtime-pack pointer state, `reload_generation`, last-known-good pointer, last-known-good recertification, hot-update gates, or V4-056 work are changed in V4-055.

## Existing Direct Operator Command Path

The current direct operator command path is in `internal/agent/loop.go`.

Hot-update commands are parsed with regexes and handled in `processOperatorCommand` before provider fallback:

- `HOT_UPDATE_GATE_RECORD <job_id> <hot_update_id> <candidate_pack_id>`
- `HOT_UPDATE_GATE_PHASE <job_id> <hot_update_id> <phase>`
- `HOT_UPDATE_GATE_EXECUTE <job_id> <hot_update_id>`
- `HOT_UPDATE_GATE_RELOAD <job_id> <hot_update_id>`
- `HOT_UPDATE_GATE_FAIL <job_id> <hot_update_id> <reason...>`
- `HOT_UPDATE_OUTCOME_CREATE <job_id> <hot_update_id>`

The command handlers call TaskState wrappers and return deterministic acknowledgements:

- changed path: `Recorded`, `Advanced`, `Executed`, `Resolved`, or `Created`
- idempotent/select path: `Selected`
- failure path: empty response plus returned error

This is the right surface for the future promotion command because promotion creation is the next operator-driven ledger step after `HOT_UPDATE_OUTCOME_CREATE`.

## Existing TaskState Wrapper Pattern

Current hot-update direct commands go through `internal/agent/tools/taskstate.go`.

The wrappers consistently:

- derive `now` from `taskStateTransitionTimestamp(taskStateNowUTC())`
- clone execution/runtime state under lock
- read the mission store root from `TaskState`
- validate the mission store root
- require the command `job_id` to match the active job or persisted runtime control job
- require an active execution context or persisted runtime control context
- call the missioncontrol helper with actor `operator`
- emit a runtime control audit event with the command action name
- return the missioncontrol `changed` flag unchanged

Because the direct command path already depends on TaskState for job binding, mission store root resolution, timestamps, and audit events, V4-056 should add a small TaskState wrapper even though V4-054 itself correctly stayed missioncontrol-only.

## V4-054 Helper Contract

`CreatePromotionFromSuccessfulHotUpdateOutcome` already provides the storage semantics V4-056 should expose.

Accepted outcome:

- `outcome_kind = hot_updated`

Rejected outcomes:

- `failed`
- `kept_staged`
- `discarded`
- `blocked`
- `approval_required`
- `cold_restart_required`
- `canary_applied`
- `promoted`
- `rolled_back`
- `aborted`
- unknown or future outcome kinds

Deterministic promotion identity:

- `hot-update-promotion-<hot_update_id>`

Derived mapping:

- `promoted_pack_id`: copied from `HotUpdateOutcomeRecord.CandidatePackID`
- `previous_active_pack_id`: copied from `HotUpdateGateRecord.PreviousActivePackID`
- `hot_update_id`: copied from `HotUpdateOutcomeRecord.HotUpdateID`
- `outcome_id`: copied from `HotUpdateOutcomeRecord.OutcomeID`
- optional `candidate_id`, `run_id`, and `candidate_result_id`: copied only when present on the outcome
- `promoted_at`: copied from `HotUpdateOutcomeRecord.OutcomeAt`
- `created_at`: helper input
- `created_by`: helper input
- `reason`: `hot update outcome promoted`

Replay behavior:

- first creation writes the promotion and returns `changed=true`
- exact replay returns `changed=false`
- divergent duplicate with the deterministic `promotion_id` fails closed
- existing promotion for the same `hot_update_id` under a different `promotion_id` fails closed
- existing promotion for the same `outcome_id` under a different `promotion_id` fails closed
- if linked outcome/gate data changes such that the derived promotion differs, the helper fails closed rather than rewriting

## Recommended V4-056 Command Shape

Recommended command:

- `HOT_UPDATE_PROMOTION_CREATE <job_id> <outcome_id>`

Rationale:

- It follows the existing uppercase direct operator command convention.
- It keeps the hot-update lane prefix.
- It names the created ledger object directly.
- It consumes the committed outcome checkpoint created by V4-052.
- It avoids caller-provided promotion IDs, pack refs, timestamps, reasons, or optional linkage.

Required arguments:

- `job_id`: must match the active job or persisted runtime control job, using the same validation as existing hot-update TaskState wrappers.
- `outcome_id`: identifies the committed successful hot-update outcome to resolve into a promotion.

Rejected/manual arguments:

- caller-provided `promotion_id`
- caller-provided promotion kind or status
- caller-provided reason
- caller-provided `promoted_pack_id`
- caller-provided `previous_active_pack_id`
- caller-provided last-known-good pack or basis
- caller-provided candidate/run/result refs
- caller-provided timestamps
- caller-provided actor

Those fields must remain derived from the committed outcome, the originating gate, and the TaskState timestamp/actor pattern.

## Recommended V4-056 Implementation Shape

Smallest safe code surface:

- Add `hotUpdatePromotionCreateCommandRE` in `internal/agent/loop.go`.
- Dispatch it next to `HOT_UPDATE_OUTCOME_CREATE`.
- Add `(*TaskState).CreatePromotionFromSuccessfulHotUpdateOutcome(jobID, outcomeID string) (bool, error)`.
- Use the same active/persisted runtime validation pattern as existing hot-update TaskState wrappers.
- Resolve the mission store root from TaskState.
- Derive `now` with `taskStateTransitionTimestamp(taskStateNowUTC())`.
- Call `missioncontrol.CreatePromotionFromSuccessfulHotUpdateOutcome(root, outcomeID, "operator", now)`.
- Emit runtime control audit action `hot_update_promotion_create`.
- Return deterministic direct responses:
  - changed: `Created hot-update promotion job=<job_id> outcome=<outcome_id>.`
  - idempotent: `Selected hot-update promotion job=<job_id> outcome=<outcome_id>.`

No new public API outside the direct command and TaskState wrapper is needed.

## Failure Behavior To Preserve

V4-056 should pass through fail-closed helper and TaskState errors without manufacturing success.

Expected failure cases:

- Missing outcome: error from `LoadHotUpdateOutcomeRecord`, response `""`.
- Non-`hot_updated` outcome: error containing `does not permit promotion creation`, response `""`.
- Failed outcome: same non-`hot_updated` rejection path, response `""`.
- Missing originating gate: error from loading the gate through `outcome.hot_update_id`, response `""`.
- Invalid outcome/gate linkage: helper/linkage validation error, response `""`.
- Empty `candidate_pack_id`: error containing `candidate_pack_id is required for promotion creation`, response `""`.
- Missing or unresolved `previous_active_pack_id`: error containing previous-active-pack context, response `""`.
- Divergent deterministic promotion: error containing `promotion "<promotion_id>" already exists`, response `""`.
- Existing promotion for same `hot_update_id` but different `promotion_id`: error containing `hot_update_id "<hot_update_id>" already exists as "<promotion_id>"`, response `""`.
- Existing promotion for same `outcome_id` but different `promotion_id`: error containing `outcome_id "<outcome_id>" already exists as "<promotion_id>"`, response `""`.
- Wrong `job_id`: same TaskState validation behavior as existing hot-update direct commands, response `""`.

The command must not infer success from a gate, active pointer state, status output, or absence of a failure record. The committed outcome is the control entry source.

## Read-Only / Status Expectations After Creation

Existing status/read-model surfaces should be sufficient:

- `STATUS <job_id>` already includes `promotion_identity` through TaskState status readout.
- `LoadOperatorPromotionIdentityStatus` lists promotion records deterministically.
- Promotion status exposes `promotion_id`, promoted pack, previous active pack, last-known-good fields when present, hot-update ID, outcome ID, optional candidate/run/result refs, reason, notes, promoted/created timestamps, created actor, and validation error state.
- `hot_update_outcome_identity` continues to expose the source outcome.
- `hot_update_gate_identity` continues to expose the originating gate.

V4-056 should test that `STATUS <job_id>` surfaces the newly created deterministic promotion in `promotion_identity`. It should not add a separate status command unless the existing status read model proves insufficient during implementation.

## Required V4-056 Tests

Direct command tests in `internal/agent/loop_processdirect_test.go` should prove:

- `HOT_UPDATE_PROMOTION_CREATE <job_id> <outcome_id>` creates a promotion from a successful `hot_updated` outcome.
- Exact replay returns the `Selected hot-update promotion...` acknowledgement and does not rewrite the promotion file.
- Missing outcome rejects with empty direct response and no promotion record.
- Non-`hot_updated` outcome kinds reject with empty direct response and no promotion record.
- Failed outcome rejects with empty direct response and no promotion record.
- Missing originating gate rejects with empty direct response and no promotion record.
- Invalid outcome/gate linkage rejects with empty direct response and no promotion record.
- Empty `candidate_pack_id` rejects with empty direct response and no promotion record.
- Missing or unresolved `previous_active_pack_id` rejects with empty direct response and no promotion record.
- Divergent deterministic promotion rejects with empty direct response.
- Existing promotion for same `hot_update_id` but different `promotion_id` rejects with empty direct response.
- Existing promotion for same `outcome_id` but different `promotion_id` rejects with empty direct response.
- Wrong `job_id` rejects through existing TaskState job validation.
- Successful creation does not create a `HotUpdateOutcomeRecord`.
- Active runtime-pack pointer bytes are unchanged.
- `reload_generation` is unchanged.
- `last_known_good_pointer.json` bytes are unchanged.
- Hot-update gate bytes are unchanged.
- Hot-update outcome bytes are unchanged.
- `STATUS <job_id>` includes the created promotion in `promotion_identity`.

TaskState-focused tests in `internal/agent/tools/taskstate_test.go` should be added only if the direct command tests do not already lock the wrapper semantics. If added, keep them narrow around job validation, audit event action name `hot_update_promotion_create`, changed flag propagation, and side-effect invariants.

Missioncontrol tests do not need broad expansion in V4-056 because V4-054 already covered the helper behavior directly.

## Non-Goals For V4-056

- no new promotion mapping
- no direct promotion from a gate without an outcome
- no `HotUpdateOutcomeRecord` creation
- no active runtime-pack pointer mutation
- no `reload_generation` mutation
- no `last_known_good_pointer.json` mutation
- no last-known-good recertification
- no hot-update gate mutation
- no policy or authorization broadening beyond existing TaskState direct command checks
- no manual promotion IDs
- no manual promoted pack refs
- no manual previous active pack refs
- no manual candidate/run/result refs
- no V4-057 work

## Recommendation

The smallest safe future slice is V4-056: add `HOT_UPDATE_PROMOTION_CREATE <job_id> <outcome_id>` to the existing direct operator command path, backed by a minimal TaskState wrapper that calls `CreatePromotionFromSuccessfulHotUpdateOutcome`.

Do not add a separate command family, public API, or manual promotion parameters. The committed outcome plus V4-054 helper already define the authoritative promotion mapping.
