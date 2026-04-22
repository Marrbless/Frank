# V4-026 Reload/Apply Execution Assessment

## Current Checkpoint Facts

- branch: `frank-v4-026-reload-apply-execution-assessment`
- `HEAD`: `2ecef226665b19bc984dc6be4626d2d367203859`
- tags at `HEAD`: `frank-v4-025-rollback-pointer-switch-skeleton`
- ahead/behind `upstream/main`: `424 0`
- initial `git status --short --branch`:

```text
## frank-v4-026-reload-apply-execution-assessment
```

- baseline `go test -count=1 ./...`: passed
- committed rollback lane through V4-025 now provides:
  - rollback record authority
  - rollback-apply workflow identity and phase progression
  - rollback-apply direct control entry
  - pointer-switch execution to `pointer_switched_reload_pending`
  - read-only status exposure for rollback-apply phase and active pointer identity

## Current Rollback / Rollback-Apply / Pointer-Switch Truth

### Durable truth that exists today

- `RollbackRecord` remains the authority for:
  - `rollback_id`
  - `from_pack_id`
  - `target_pack_id`
  - optional `last_known_good_pack_id`
  - optional promotion / hot-update / outcome linkage
- `RollbackApplyRecord` remains the authority for rollback-apply workflow identity and current progression:
  - `apply_id`
  - `rollback_id`
  - `phase`
  - `activation_state`
  - `created_at`
  - `created_by`
  - `phase_updated_at`
  - `phase_updated_by`
- committed rollback-apply phases now include:
  - `recorded`
  - `validated`
  - `ready_to_apply`
  - `pointer_switched_reload_pending`
- `ActiveRuntimePackPointer` remains the authority for committed active selection:
  - `active_pack_id`
  - `previous_active_pack_id`
  - `last_known_good_pack_id`
  - `updated_at`
  - `updated_by`
  - `update_record_ref`
  - `reload_generation`
- `LastKnownGoodRuntimePackPointer` remains independent durable truth and was intentionally left unchanged by V4-025

### Control-plane truth that exists today

- `ROLLBACK_APPLY_EXECUTE <job_id> <apply_id>`:
  - requires `ready_to_apply`
  - switches the active pointer to `rollback.target_pack_id`
  - increments `reload_generation`
  - sets `update_record_ref` to `rollback_apply:<apply_id>`
  - advances rollback-apply phase to `pointer_switched_reload_pending`
  - is idempotent on exact replay
- `STATUS <job_id>` can already show:
  - active runtime-pack identity
  - rollback-apply phase
- current status does not expose:
  - reload/apply attempt state
  - reload/apply success vs failure
  - restart/recovery reconciliation evidence

### Important current absence

- there is still no reusable runtime-pack apply hook in the repo
- there is still no committed reload/apply execution helper
- there is still no durable reload/apply attempt/result model
- there is still no reload success/failure read-model surface beyond the coarse rollback-apply phase

## 1. Exact Authority Needed To Consume `pointer_switched_reload_pending`

The minimum safe authority for the next slice is:

1. Read authority for:
   - the `RollbackApplyRecord`
   - the linked `RollbackRecord`
   - the `ActiveRuntimePackPointer`
   - the referenced target runtime pack
2. Write authority for:
   - the `RollbackApplyRecord` so reload/apply attempt state and outcome can be recorded durably
3. Runtime execution authority for one bounded apply mechanism:
   - either an in-process pack reload hook
   - or a controlled process-restart apply hook
4. Recovery/reconciliation authority on process start:
   - enough to observe a still-pending reload/apply state and reconcile it against the committed active pointer

The current codebase does not expose an in-process pack apply API. Because of that, the smallest truthful reload/apply authority is a controlled restart-style apply path, not a new generic in-process reload engine.

## 2. Minimum Execution Boundary For Reload/Apply

### Already done before this slice

- runtime-pack pointer switch
- `reload_generation` increment
- attribution via `update_record_ref = rollback_apply:<apply_id>`
- durable state `pointer_switched_reload_pending`

### Minimum boundary for the next real slice

The next slice should do only these things:

1. consume `pointer_switched_reload_pending`
2. durably mark that reload/apply has started
3. invoke one bounded reload/apply mechanism
4. durably record success or failure
5. leave `last_known_good_pointer.json` unchanged

### Recommended apply mechanism

The smallest safe real apply mechanism is:

- controlled restart-style apply semantics aligned with spec `process_restart_hot_swap`

Reason:

- the repo has no existing in-process runtime-pack application hook
- implementing generic `pack_reload`, `skill_reload`, or `extension_reload` would require inventing new reload plumbing first
- the spec already says a restart loads the committed active pack unless recovery mode chooses last-known-good
- restart-based apply is smaller than a new partial reload framework

### Success/failure recording boundary

Success/failure recording must be durable and separate from pointer truth.

The active pointer already says which pack should become active. The next slice must add durable execution truth answering whether runtime convergence to that pack:

- has not started
- is in progress
- completed
- failed

### Last-known-good boundary

No last-known-good recertification is required in the first reload/apply slice.

Rollback apply is restoring a target that was already selected by committed rollback authority. The first real reload/apply slice only needs to converge runtime behavior to the already-switched active pointer.

## 3. What Must Be Atomic Vs Crash-Recoverable

### Does not need to be atomic

- writing reload/apply attempt state
- performing the restart-style apply action
- writing reload/apply success/failure state

These cannot be one atomic operation across a process boundary.

### Must behave as one crash-recoverable state machine

The next slice needs this durable progression:

1. `pointer_switched_reload_pending`
2. `reload_apply_in_progress`
3. terminal outcome:
   - `reload_apply_succeeded`
   - or `reload_apply_failed`

### Why `reload_apply_in_progress` is required

Without an explicit in-progress state, a crash cannot distinguish:

- never started
- started and crashed before recording outcome
- started, partially converged, and then crashed

That ambiguity is too large once the side effect crosses the process boundary.

## 4. Replay / Idempotence Rules Required

### Replay-safe entry conditions

- if phase is `pointer_switched_reload_pending`, execution may start
- if phase is `reload_apply_in_progress`, recovery logic must reconcile instead of blindly reissuing the side effect
- if phase is `reload_apply_succeeded`, return idempotent already-applied success
- if phase is `reload_apply_failed`, retry must be explicit and policy-bounded, not implicit

### Required replay key

- `apply_id` remains the stable execution key
- `update_record_ref = rollback_apply:<apply_id>` remains the pointer-side join key

### Recovery/replay rules

- never rewrite the active pointer again for the same `apply_id`
- never increment `reload_generation` again during reload/apply replay
- if process starts and finds:
  - active pointer attributed to `rollback_apply:<apply_id>`
  - rollback-apply phase `reload_apply_in_progress`
  then recovery should reconcile that in-progress attempt before any new retry path
- retry after explicit failure should reuse the same switched pointer state and create only a new reload/apply attempt, not a new pointer switch

## 5. Required State Transitions After Success / Failure / Crash

### On reload/apply success

Recommended transition:

- `pointer_switched_reload_pending -> reload_apply_in_progress -> reload_apply_succeeded`

Result:

- active pointer remains on rollback target
- runtime convergence is durably acknowledged
- last-known-good remains unchanged

### On reload/apply failure

Recommended transition:

- `pointer_switched_reload_pending -> reload_apply_in_progress -> reload_apply_failed`

Result:

- active pointer still names the rollback target pack unless a later explicit recovery slice changes it
- failure is durable and inspectable
- retry is explicit

### On process crash before completion

Recommended transition model:

- durable phase remains `reload_apply_in_progress`
- process-start recovery reconciles that state against the current committed active pointer and resumes failure/success handling

The crash case should not silently rewrite the pointer or mutate last-known-good.

## 6. Last-Known-Good Decision Analysis

Last-known-good should remain unchanged in the first reload/apply slice.

Reason:

- V4-025 explicitly preserved it
- replay rules forbid silently losing or fabricating last-known-good truth
- reload/apply success does not itself prove explicit later recertification
- recertification is a separate verification/policy question, not a prerequisite for converging runtime to the already-selected rollback target

No change to `last_known_good_pointer.json` is required for the first real reload/apply slice.

## 7. Acceptance Tests Required For The First Real Execution Slice

The next implementation slice should add narrow tests for:

1. happy-path consume of `pointer_switched_reload_pending`
   - writes `reload_apply_in_progress`
   - invokes the bounded restart/apply path
   - records `reload_apply_succeeded`
2. idempotent replay after success
   - second execution returns already-applied success
   - does not rewrite the pointer
   - does not increment `reload_generation`
3. explicit failure recording
   - bounded apply hook fails
   - rollback-apply becomes `reload_apply_failed`
   - pointer remains unchanged from the V4-025 switched state
4. crash/recovery reconciliation
   - durable state starts at `reload_apply_in_progress`
   - recovery path reconciles without a second pointer switch
5. invalid-state rejection
   - missing rollback-apply record
   - phase not `pointer_switched_reload_pending`
   - pointer `update_record_ref` mismatched to `rollback_apply:<apply_id>`
   - pointer active pack mismatched to rollback target
6. last-known-good remains byte-for-byte unchanged
7. status coherence
   - active pointer still reports rollback target
   - rollback-apply read model reports the new reload/apply phase accurately

## Reload / Apply Boundary Analysis

### Recommended boundary for the first real slice

Use rollback-apply phase for the smallest adjacent extension:

- `pointer_switched_reload_pending`
- `reload_apply_in_progress`
- `reload_apply_succeeded`
- `reload_apply_failed`

This is not ideal long-term separation, but it is the smallest adjacent extension to current committed V4-025 truth. Introducing a brand-new sibling execution ledger first would enlarge the slice without clear evidence that the current phase model is insufficient for the first restart-style apply implementation.

### Why not generic in-process reload first

- no existing runtime-pack apply hook exists
- no current pack-component cache invalidation hook exists
- no current extension/tool reload orchestrator exists
- restart is already a spec-recognized convergence mechanism

## Crash / Recovery State-Machine Analysis

The critical recovery invariant is:

- pointer truth and runtime convergence truth are different

Current committed state already says which pack should become active. The first reload/apply slice must make recovery deterministic when runtime convergence is interrupted.

Minimum recovery rules:

- if phase is `pointer_switched_reload_pending`, start the reload/apply attempt
- if phase is `reload_apply_in_progress`, recover that attempt before allowing a new one
- if phase is `reload_apply_failed`, require explicit retry command
- if phase is `reload_apply_succeeded`, no-op on replay

## Idempotence / Replay Model

- stable key: `apply_id`
- stable pointer attribution: `rollback_apply:<apply_id>`
- pointer mutation is never replayed in this slice
- success replay is a no-op
- failure replay is explicit, not automatic
- crash recovery reconciles `reload_apply_in_progress` first

## Last-Known-Good Decision Analysis

- keep `last_known_good_pointer.json` unchanged
- do not infer recertification from reload/apply completion
- defer any recertification to a later policy/verification slice

## Failure-Mode Matrix

| Failure window | Durable state | Runtime state | Required posture |
|---|---|---|---|
| before attempt start | `pointer_switched_reload_pending` | runtime may still be old pack | safe start |
| after writing `reload_apply_in_progress`, before side effect | `reload_apply_in_progress` | runtime still old pack | recover in-progress attempt |
| crash during restart/apply | `reload_apply_in_progress` | runtime may be restarting or old pack | reconcile on process start, no second pointer switch |
| apply hook returns failure | `reload_apply_failed` | runtime not converged to target | explicit retry or later recovery slice |
| apply hook returns success | `reload_apply_succeeded` | runtime converged to target | no-op on replay |
| status/read-model lacks finer detail | active pointer and rollback-apply phase visible | operator cannot see attempt detail | acceptable for first slice if new phases are surfaced truthfully |

## Smallest Safe Next Implementation Slice

Recommended next implementation slice:

1. extend rollback-apply phase model with:
   - `reload_apply_in_progress`
   - `reload_apply_succeeded`
   - `reload_apply_failed`
2. add a minimal control entry on the existing direct-command path to consume `pointer_switched_reload_pending`
3. use a bounded restart-style apply hook as the actual convergence mechanism
4. record success or failure durably on the same rollback-apply record
5. leave the active pointer unchanged
6. leave `last_known_good_pointer.json` unchanged

This is the smallest safe real reload/apply slice because it:

- consumes the committed V4-025 pending state directly
- introduces no second pointer mutation path
- uses a spec-aligned real convergence mechanism
- keeps recovery logic inspectable and replay-safe
- avoids inventing a broader in-process reload framework first

## Explicit Non-Goals For That First Execution Slice

- no new pointer mutation logic
- no second `reload_generation` increment
- no last-known-good recertification
- no promotion behavior changes
- no evaluator or scoring behavior
- no autonomy changes
- no provider/channel changes
- no generic hot-update engine
- no broad pack-component reload framework beyond the bounded restart-style apply mechanism
- no recovery-policy rollback or quarantine automation beyond durable failure recording
