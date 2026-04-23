# V4-030 Rollback Recovery Resolution Assessment

## Current checkpoint facts

- Repository root: `/mnt/d/pbot/picobot`
- Branch: `frank-v4-030-rollback-recovery-resolution-assessment`
- HEAD: `87a6a90d9a3189b40b2dcca8f5dca5708427d463`
- Tags at HEAD: `frank-v4-029-rollback-apply-recovery-needed`
- Ahead/behind `upstream/main`: `428 0`
- Baseline worktree state at assessment start: clean
- Baseline validation at assessment start: `go test -count=1 ./...` passed

## Current rollback-apply recovery-needed truth

- Rollback records remain the sole durable authority for rollback linkage and target pack identity.
- Rollback-apply records remain the sole durable authority for rollback-apply workflow identity and phase.
- The current durable rollback-apply phases are:
  - `recorded`
  - `validated`
  - `ready_to_apply`
  - `pointer_switched_reload_pending`
  - `reload_apply_in_progress`
  - `reload_apply_recovery_needed`
  - `reload_apply_succeeded`
  - `reload_apply_failed`
- `ReconcileRollbackApplyRecoveryNeeded(...)` already normalizes persisted unknown-outcome crash state:
  - `reload_apply_in_progress -> reload_apply_recovery_needed`
  - only when rollback linkage and active-pointer linkage are still coherent
  - without mutating the active pointer, `reload_generation`, or `last_known_good_pointer.json`
- There is currently no operator-facing resolution step for `reload_apply_recovery_needed`.
- The existing operator direct-command path already exposes:
  - `ROLLBACK_APPLY_RECORD`
  - `ROLLBACK_APPLY_PHASE`
  - `ROLLBACK_APPLY_EXECUTE`
  - `ROLLBACK_APPLY_RELOAD`
- The existing reload/apply helper still refuses `reload_apply_recovery_needed`; it only accepts `pointer_switched_reload_pending`.

## 1. What exact recovery options should exist?

Three operator-visible resolution choices are reasonable in principle.

### Option A: explicit retry of reload/apply on the same `apply_id`

This should exist.

Reason:

- the active pointer has already been switched for this same rollback-apply workflow
- `update_record_ref` already binds the active pointer to `rollback_apply:<apply_id>`
- retrying on the same `apply_id` preserves workflow identity, audit continuity, and replay semantics
- it avoids creating competing workflow records for the same already-switched pointer state

### Option B: explicit terminal failure decision

This should exist eventually, but it is not the smallest safe first slice.

Reason:

- a human may decide that retrying is unsafe or no longer desired
- the control plane needs a way to convert `reload_apply_recovery_needed` into a terminal state deliberately
- that decision should be auditable and should include operator-provided failure reason text

### Option C: creation of a new apply record instead of retry

This should not be the first or preferred recovery-resolution path.

Reason:

- the active pointer already points at the rollback target under the existing `apply_id`
- creating a new apply record would fork durable workflow identity away from the pointer state already on disk
- that would weaken the authority model by making one pointer mutation correspond to multiple rollback-apply records

If this ever exists, it should be treated as a separate workflow-authoring decision, not as recovery resolution for the existing record.

## 2. Which option is the smallest safe first implementation slice?

The smallest safe first implementation slice is:

- explicit retry of reload/apply on the same `apply_id` from `reload_apply_recovery_needed`

Why this is the smallest safe slice:

- it continues the existing workflow instead of creating a new one
- it reuses the already-existing reload/apply command semantics
- it does not require a new pointer mutation path
- it keeps durable authority centered on the existing rollback record and rollback-apply record
- it can be implemented by widening one already-existing execution helper rather than adding a second operator concept immediately

The first slice should not include explicit operator terminal failure resolution yet. That is a valid later slice, but retry is the narrower continuation of the already-started workflow.

## 3. What control surface should drive that resolution?

The narrowest truthful operator-driven path is:

1. keep execution semantics in `internal/missioncontrol`
2. reuse the existing `TaskState.ExecuteRollbackApplyReloadApply(...)` wrapper
3. reuse the existing operator direct-command `ROLLBACK_APPLY_RELOAD <job_id> <apply_id>`

That means:

- `missioncontrol` owns the durable retry rules
- `taskstate` remains a thin validation and audit wrapper
- `loop.go` keeps using the existing direct command without inventing a new operator verb for the first slice

This is narrower than adding a brand-new command because the operator concept already exists: “retry reload/apply”.

## 4. What idempotence rules must hold for retry?

### Starting-phase rules

- retry must only be newly allowed from `reload_apply_recovery_needed`
- retry must continue to be rejected from:
  - `recorded`
  - `validated`
  - `ready_to_apply`
  - `reload_apply_failed`
- replay from `reload_apply_succeeded` must remain idempotent and must not redo side effects

### Retry execution rules

- issuing retry from `reload_apply_recovery_needed` should:
  - revalidate rollback linkage
  - revalidate active-pointer linkage to the same `apply_id`
  - clear stale unknown-outcome state
  - re-enter `reload_apply_in_progress`
  - re-run the bounded reload/apply convergence path

### Post-retry replay rules

- if retry succeeds, reissuing the same command must remain a no-op through the existing `reload_apply_succeeded` behavior
- if retry crashes again after writing `reload_apply_in_progress`, recovery normalization should again move it back to `reload_apply_recovery_needed`
- if retry reaches terminal `reload_apply_failed`, replay should remain closed unless a later checkpoint explicitly broadens retry policy from terminal failure

## 5. What invariants must remain unchanged during recovery resolution?

The following must remain unchanged for the first retry slice.

### Active runtime-pack pointer

- no second pointer switch
- active pointer must continue to reference the rollback target pack
- `update_record_ref` must continue to reference `rollback_apply:<apply_id>`

### Reload generation

- `reload_generation` must not increment again
- the only generation increment for this rollback-apply workflow remains the V4-025 pointer-switch slice

### Last-known-good pointer

- `last_known_good_pointer.json` must remain unchanged
- retry resolution is about converging the already-selected rollback target, not recertifying it

### Rollback/apply authority model

- rollback record remains the source of truth for target linkage
- rollback-apply record remains the source of truth for workflow identity and phase
- no new apply record should be created in the first resolution slice

## 6. What error details must be preserved vs cleared on retry or terminal failure?

### On retry from `reload_apply_recovery_needed`

- `execution_error` should be empty before re-entering `reload_apply_in_progress`
- retry should not invent or preserve terminal failure text for an unknown-outcome recovery state
- if a future implementation later adds recovery metadata, that should be a separate field, not overloaded into `execution_error`

### On terminal failure decision

- if a later slice adds explicit operator terminal failure resolution, it should require explicit operator-supplied reason text
- that text should be written into `execution_error`
- the stored message should clearly indicate that failure was operator-resolved rather than convergence-detected, for example by using a structured prefix or clearly phrased message

### On retry failure

- if retry runs and the bounded convergence path fails again, the resulting terminal `reload_apply_failed` should continue to populate `execution_error` with that concrete convergence failure

## 7. What acceptance tests are required for the first implementation slice?

The first implementation slice should add narrow tests for:

- `ROLLBACK_APPLY_RELOAD` or the underlying missioncontrol helper accepts `reload_apply_recovery_needed` as a valid retry starting phase
- happy-path retry from `reload_apply_recovery_needed` reaches `reload_apply_succeeded`
- retry re-enters `reload_apply_in_progress` internally and clears any stale unknown-outcome state before success/failure recording
- invalid rollback linkage still rejects without pointer mutation
- invalid active-pointer linkage still rejects without pointer mutation
- active pointer remains unchanged during retry resolution
- `reload_generation` remains unchanged during retry resolution
- `last_known_good_pointer.json` remains unchanged during retry resolution
- replay after successful retry is idempotent
- crash-or-retry-loop behavior remains compatible with V4-029 normalization:
  - a retried execution that crashes again after writing `reload_apply_in_progress` can still be normalized back to `reload_apply_recovery_needed`

## 8. What are the explicit non-goals for that slice?

- no new apply-record creation path
- no second pointer mutation
- no second `reload_generation` increment
- no `last_known_good_pointer.json` mutation
- no automatic terminal failure decision
- no new promotion behavior
- no evaluator, scoring, autonomy, provider, or channel changes
- no broader reload engine redesign
- no expansion of terminal-failure retry policy beyond `reload_apply_recovery_needed`

## Recommended conclusion

The smallest safe operator-driven resolution model is to reuse the existing reload/apply command path and allow explicit retry on the same `apply_id` from `reload_apply_recovery_needed`. The durable logic should live in `missioncontrol`, the existing `TaskState.ExecuteRollbackApplyReloadApply(...)` wrapper should remain the thin control-plane adapter, and `ROLLBACK_APPLY_RELOAD <job_id> <apply_id>` should remain the operator-facing verb. Explicit terminal failure resolution is valid, but it should be deferred until after same-record retry exists, because retry is the narrower continuation of the already authoritative rollback-apply workflow.
