# V4-062 Hot-Update Operator UX / Help Text Assessment

## Scope

V4-062 assesses the smallest useful operator-facing UX/help improvement for the completed V4 hot-update lifecycle.

This slice is docs-only. It does not change Go code, tests, commands, TaskState wrappers, runtime pointers, reload generation, outcomes, promotions, hot-update gates, automation, policy, authorization, or V4-063 scope.

## Current Baseline

- Branch: `frank-v4-062-hot-update-operator-ux-help-assessment`
- HEAD: `ce1be7cc70ecc36e70f0a88649262d490b20882e`
- Tag at HEAD:
  - `frank-v4-061-hot-update-end-to-end-lifecycle-checkpoint`
- Starting worktree:
  - clean
- Starting validation:
  - `/usr/local/go/bin/go test -count=1 ./...` passed

## Surfaces Inspected

Existing operator-facing command/help surfaces are split across:

- `README.md` and `docs/HOW_TO_START.md` for general runtime setup and operator status paths.
- `docs/FRANK_V4_SPEC.md` for broad V4 concepts, hot-update requirements, and operator-control intent.
- `docs/maintenance/*` for slice-level records, before/after notes, and checkpoint memos.
- `internal/agent/loop.go` for direct command parsing.
- `internal/agent/loop_processdirect_test.go` for executable command behavior and response expectations.

There is no dedicated static runbook for the completed hot-update lifecycle. There is also no obvious low-risk in-product help command or static direct-help registry for these operator commands. The command surface is currently discoverable through tests, specs, and maintenance memos rather than through a concise operator sequence document.

## Completed Direct Command Surface

The completed lifecycle uses these direct commands:

- `HOT_UPDATE_GATE_RECORD <job_id> <hot_update_id> <candidate_pack_id>`
- `HOT_UPDATE_GATE_PHASE <job_id> <hot_update_id> validated`
- `HOT_UPDATE_GATE_PHASE <job_id> <hot_update_id> staged`
- `HOT_UPDATE_GATE_EXECUTE <job_id> <hot_update_id>`
- `HOT_UPDATE_GATE_RELOAD <job_id> <hot_update_id>`
- `HOT_UPDATE_GATE_FAIL <job_id> <hot_update_id> <reason>`
- `HOT_UPDATE_OUTCOME_CREATE <job_id> <hot_update_id>`
- `HOT_UPDATE_PROMOTION_CREATE <job_id> <outcome_id>`
- `HOT_UPDATE_LKG_RECERTIFY <job_id> <promotion_id>`
- `STATUS <job_id>`

These commands are intentionally explicit. None of the later ledger or recertification steps are automatic side effects of earlier steps.

## Recommended Operator Command Sequence

### 1. Record Or Select The Gate

Command:

`HOT_UPDATE_GATE_RECORD <job_id> <hot_update_id> <candidate_pack_id>`

Expected response:

- first create: `Recorded hot-update gate job=<job_id> hot_update=<hot_update_id>.`
- replay/select: `Selected hot-update gate job=<job_id> hot_update=<hot_update_id>.`

Status check:

- `STATUS <job_id>`
- confirm `hot_update_gate_identity.state = configured`
- confirm the gate lists `hot_update_id`, `candidate_pack_id`, `previous_active_pack_id`, rollback target, reload mode, and initial state

### 2. Validate And Stage

Commands:

- `HOT_UPDATE_GATE_PHASE <job_id> <hot_update_id> validated`
- `HOT_UPDATE_GATE_PHASE <job_id> <hot_update_id> staged`

Status check:

- confirm gate `state` moves to `validated`, then `staged`
- confirm `phase_updated_at` and `phase_updated_by`
- confirm active pointer and LKG status are unchanged

### 3. Execute Pointer Switch

Command:

`HOT_UPDATE_GATE_EXECUTE <job_id> <hot_update_id>`

Status check:

- confirm gate moves to `reloading`
- confirm `runtime_pack_identity.active.active_pack_id = <candidate_pack_id>`
- confirm `runtime_pack_identity.active.previous_active_pack_id` names the prior active pack
- confirm `reload_generation` changed only as part of this pointer switch
- confirm `runtime_pack_identity.last_known_good.pack_id` still names the previous LKG

### 4. Reload/Apply

Command:

`HOT_UPDATE_GATE_RELOAD <job_id> <hot_update_id>`

Status check:

- success path: confirm gate `state = reload_apply_succeeded`
- failure path: confirm gate `state = reload_apply_failed` and inspect `failure_reason`
- interrupted path: confirm gate can normalize to `reload_apply_recovery_needed`
- confirm no outcome, promotion, or LKG recertification was created automatically

### 5. Resolve Recovery-Needed

If the gate is `reload_apply_recovery_needed`, the operator has two explicit choices.

Retry:

`HOT_UPDATE_GATE_RELOAD <job_id> <hot_update_id>`

Terminal failure:

`HOT_UPDATE_GATE_FAIL <job_id> <hot_update_id> <reason>`

Status check:

- retry success should reach `reload_apply_succeeded`
- retry failure should reach `reload_apply_failed`
- terminal failure should store `failure_reason = operator_terminal_failure: <reason>`
- exact terminal-failure replay with the same reason should select idempotently
- terminal-failure replay with a different reason should fail closed

### 6. Create Outcome

Command:

`HOT_UPDATE_OUTCOME_CREATE <job_id> <hot_update_id>`

Status check:

- confirm `hot_update_outcome_identity.state = configured`
- success gate creates deterministic outcome id `hot-update-outcome-<hot_update_id>`
- success gate maps to `outcome_kind = hot_updated`
- failed gate maps to `outcome_kind = failed`
- failed outcome reason copies deterministic gate failure detail
- confirm no promotion was created automatically

### 7. Create Promotion

Command:

`HOT_UPDATE_PROMOTION_CREATE <job_id> <outcome_id>`

Status check:

- confirm `promotion_identity.state = configured`
- confirm promotion id `hot-update-promotion-<hot_update_id>`
- confirm `promoted_pack_id = outcome.candidate_pack_id`
- confirm `previous_active_pack_id = gate.previous_active_pack_id`
- confirm `outcome_id` and `hot_update_id` linkage
- confirm LKG did not change automatically

### 8. Recertify Last-Known-Good

Command:

`HOT_UPDATE_LKG_RECERTIFY <job_id> <promotion_id>`

Status check:

- confirm `runtime_pack_identity.last_known_good.state = configured`
- confirm `runtime_pack_identity.last_known_good.pack_id = promotion.promoted_pack_id`
- confirm `runtime_pack_identity.last_known_good.basis = hot_update_promotion:<promotion_id>`
- confirm `runtime_pack_identity.last_known_good.verified_at` is present
- confirm active pointer remains unchanged
- confirm `reload_generation` remains unchanged
- confirm promotion, outcome, and gate records are not mutated

## Replay Behavior To Document

The runbook/help text should explicitly state:

- Replaying `HOT_UPDATE_GATE_RECORD` selects the existing gate when it matches.
- Replaying valid adjacent phase commands is idempotent only when the stored phase already matches the selected phase; divergent progression fails closed.
- Replaying `HOT_UPDATE_GATE_EXECUTE` after the pointer switch selects the existing execution state and does not increment `reload_generation` again.
- Replaying `HOT_UPDATE_GATE_RELOAD` after terminal success or compatible in-progress/recovery resolution should not create new records or mutate unrelated state.
- Replaying `HOT_UPDATE_GATE_FAIL` requires the same reason; a different reason fails closed.
- Replaying `HOT_UPDATE_OUTCOME_CREATE` selects the deterministic outcome when the existing outcome matches; divergent duplicates fail closed.
- Replaying `HOT_UPDATE_PROMOTION_CREATE` selects the deterministic promotion when the existing promotion matches; divergent duplicates fail closed.
- Replaying `HOT_UPDATE_LKG_RECERTIFY` selects the already recertified LKG when the stored LKG has the deterministic pack, basis, rollback ref, and stored timestamp.
- `STATUS <job_id>` is the read-only confirmation command after each major step.

## Fail-Closed Cases To Document

The runbook/help text should call out common fail-closed cases:

- Wrong `job_id` rejects through active or persisted TaskState validation.
- Missing or invalid mission store root rejects.
- Missing gate rejects phase, execute, reload, outcome creation, or promotion linkage steps.
- Non-adjacent gate phase progression rejects.
- Pointer switch rejects unless the gate is staged and candidate/rollback linkage is valid.
- Reload/apply failure does not imply outcome or promotion creation.
- Recovery-needed does not retry or terminally fail without explicit command.
- Terminal failure requires non-empty reason text.
- Outcome creation rejects non-terminal gates.
- Failed gate outcome creation rejects empty failure detail.
- Promotion creation rejects failed or non-`hot_updated` outcomes.
- LKG recertification rejects missing promotion, promotion without outcome id, linked outcome missing, linked outcome not `hot_updated`, active pointer missing or mismatched, missing current LKG, unrelated current LKG, and divergent existing LKG.
- No command accepts manual timestamps, manual LKG fields, manual promotion fields, manual outcome fields, or manual pointer fields.

## V4-063 Options

### Option A: Docs / Runbook Only

Create a dedicated operator runbook, likely:

`docs/HOT_UPDATE_OPERATOR_RUNBOOK.md`

Contents:

- command sequence
- expected responses
- status checks
- replay behavior
- fail-closed cases
- invariants and non-goals

Risk:

- low
- no code or test changes
- immediately improves discoverability

Recommendation:

- preferred V4-063 slice

### Option B: CLI / Direct Help Text Only

Add or expose static help text through a direct operator command or CLI help surface.

Risk:

- medium
- there is no existing obvious static direct-help registry for these commands
- adding a help command would be a new direct command, which should be treated as behavior and tested

Recommendation:

- defer unless a later slice explicitly selects a small static help surface

### Option C: Both Docs And Help Text

Add a runbook and a help surface in one slice.

Risk:

- higher than necessary
- mixes documentation with behavior
- increases test and review scope

Recommendation:

- do not select for the smallest next slice

## Recommended V4-063 Slice

Recommend V4-063 as docs/runbook only:

`V4-063 - Hot-Update Operator Runbook`

Create a dedicated operator runbook that documents the completed hot-update lifecycle command sequence and status checks. Keep it read-only and docs-only.

Recommended changed file:

- `docs/HOT_UPDATE_OPERATOR_RUNBOOK.md`

Hard boundaries for V4-063:

- no Go code changes
- no tests
- no commands
- no TaskState wrappers
- no pointer mutation
- no ledger creation or mutation
- no gate mutation
- no automation
- no policy/authorization broadening

If the project later wants in-product help, do a separate assessment or implementation slice after the runbook exists as the stable source text.
