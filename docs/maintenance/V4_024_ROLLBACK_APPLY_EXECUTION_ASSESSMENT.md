# V4-024 Rollback Apply Execution Assessment

## Current Checkpoint Facts

- branch: `frank-v4-024-rollback-apply-execution-assessment`
- `HEAD`: `e55110dfef8dc662c89d1cb0d3042f09eb637441`
- tags at `HEAD`: `frank-v4-023-rollback-apply-phase-control-entry1`
- ahead/behind `upstream/main`: `422 0`
- initial `git status --short --branch`:

```text
## frank-v4-024-rollback-apply-execution-assessment
```

- baseline `go test -count=1 ./...`: passed
- V4 rollback lane already committed through:
  - V4-016 rollback record skeleton
  - V4-017 rollback read-model
  - V4-018 rollback control surface
  - V4-019 rollback-apply skeleton
  - V4-020 rollback-apply read-model
  - V4-021 rollback-apply control entry
  - V4-022 rollback-apply phase progression
  - V4-023 rollback-apply phase control entry

## Current Rollback / Rollback-Apply Control-Plane Truth

### Durable records already present

- `RollbackRecord` is the durable authority for rollback linkage:
  - `rollback_id`
  - `from_pack_id`
  - `target_pack_id`
  - optional `last_known_good_pack_id`
  - optional promotion / hot-update / outcome linkage
- `RollbackApplyRecord` is the durable authority for rollback-apply workflow identity:
  - `apply_id`
  - `rollback_id`
  - `phase`
  - `activation_state`
  - `created_at`
  - `created_by`
  - `phase_updated_at`
  - `phase_updated_by`
- `ActiveRuntimePackPointer` is the durable authority for currently selected runtime pack:
  - `active_pack_id`
  - `previous_active_pack_id`
  - `last_known_good_pack_id`
  - `updated_at`
  - `updated_by`
  - `update_record_ref`
  - `reload_generation`
- `LastKnownGoodRuntimePackPointer` is the durable authority for last-known-good pack identity:
  - `pack_id`
  - `basis`
  - `verified_at`
  - `verified_by`
  - `rollback_record_ref`

### Current operator/control surfaces

- rollback creation already exists through the direct command path
- rollback-apply record creation/select already exists through:
  - `ROLLBACK_APPLY_RECORD <job_id> <rollback_id> <apply_id>`
- rollback-apply phase advance already exists through:
  - `ROLLBACK_APPLY_PHASE <job_id> <apply_id> <phase>`
- read-only inspection already exists through:
  - `STATUS <job_id>`

### Current rollback-apply behavioral truth

- rollback-apply is still non-executing
- valid rollback-apply phases are currently bounded to:
  - `recorded`
  - `validated`
  - `ready_to_apply`
- current transition authority is adjacent-only:
  - `recorded -> validated`
  - `validated -> ready_to_apply`
- same-phase replay is idempotent
- `activation_state` is currently invariantly `unchanged`
- no current helper mutates:
  - `active_runtime_pack_pointer`
  - `last_known_good_runtime_pack_pointer`
  - runtime reload state
  - process/apply state

## 1. Minimum Execution Authority Needed

The minimum safe execution authority to consume a committed rollback-apply record is:

1. Read authority for:
   - the `RollbackApplyRecord`
   - the linked `RollbackRecord`
   - the current `ActiveRuntimePackPointer`
   - the referenced runtime-pack records
2. Write authority for:
   - the `RollbackApplyRecord`, or a sibling execution record if execution state is split out
   - the `ActiveRuntimePackPointer`
3. Optional but not required for the first slice:
   - `LastKnownGoodRuntimePackPointer`
   - reload/apply runtime mechanics

The first execution slice does not need promotion authority, evaluator authority, scoring authority, autonomy authority, or provider/channel authority.

The direct command/control plane already exists and is the narrowest truthful authority path. The missing piece is durable execution state plus controlled active-pointer mutation.

## 2. Can Execution Be Safely Split?

### Active runtime-pack pointer mutation

Yes, this can be split out as the first execution slice.

It is the smallest durable mutation that actually consumes a rollback-apply record. It can be made safe if the rollback-apply workflow also records that the pointer has been switched and that reload is still pending.

### Last-known-good pointer updates

This should remain separate from the first execution slice.

Reason:

- rollback apply is restoring an already-authorized target pack, not certifying a new pack as last-known-good
- updating last-known-good on the same slice as pointer mutation creates a larger blast radius and a harder recovery story
- post-rollback verification policy is a separate concern from selecting the rollback target

### Runtime/control-plane status mutation

This cannot be omitted.

It does not have to be physically atomic with the pointer file, but it must be durably correlated with pointer mutation. A crash-safe implementation needs enough execution state to distinguish:

- ready but not started
- executing before pointer switch
- pointer switched but reload not yet completed
- completed
- failed

Without that state, replay after partial mutation is ambiguous.

### Reload/apply mechanics

This should remain separate from the first execution slice.

Reason:

- pointer mutation and process reload cannot be truly atomic across the process boundary
- the spec already treats restart and replay as explicit recovery surfaces
- a pointer-only first slice is easier to validate and easier to fail closed

### Atomicity recommendation

These do not need to be one atomic filesystem transaction:

- rollback-apply execution state mutation
- active pointer mutation
- later reload/apply
- later last-known-good update

But these do need to behave as one crash-recoverable state machine:

1. durable execution start marker
2. durable active pointer switch
3. durable `reload_pending` marker
4. later reload/apply completion or recovery action

## 3. Invariants Before, During, And After Rollback Apply

### Before execution

- the rollback-apply record exists
- the rollback-apply record phase is exactly `ready_to_apply`
- the linked rollback record exists
- the linked rollback `target_pack_id` exists
- the linked rollback `from_pack_id` exists
- the current active pack is either:
  - exactly the rollback `from_pack_id`, or
  - exactly the rollback `target_pack_id` for idempotent replay / already-applied handling
- the active pointer is valid and loadable
- no conflicting rollback-apply execution is already in progress for the same apply id

### During execution

- at most one durable pointer switch may be attributed to a given `apply_id`
- `update_record_ref` on the active pointer must identify rollback-apply execution, not promotion or bootstrap
- `reload_generation` must increase exactly once if the active pointer is changed
- `previous_active_pack_id` must preserve the pre-switch active pack
- `last_known_good_pack_id` on the active pointer must not be silently dropped
- execution must fail closed if current active state no longer matches rollback linkage in a way that cannot be explained by replay

### After the first execution slice succeeds

- the active pointer selects the rollback target pack
- the rollback-apply workflow is no longer ambiguous about whether the pointer switch happened
- reload/apply is still pending, not implied complete
- last-known-good durable truth remains unchanged
- no evaluator, scoring, promotion, autonomy, or provider behavior has changed

## 4. Idempotence / Replay Model

The minimum replay model should be:

### Safe replay states

- `ready_to_apply` with no execution marker:
  - execution may start
- execution marker says `pointer_switched_reload_pending` and active pointer already targets the rollback pack for this `apply_id`:
  - return idempotent success
  - do not rewrite the pointer again
  - do not increment `reload_generation` again
- execution marker says `completed`:
  - return already applied

### Required replay keys

- `apply_id` remains the stable workflow key
- `update_record_ref` on the active pointer should include the `apply_id`, for example `rollback_apply:<apply_id>`
- execution state must be able to reconcile from either:
  - rollback-apply execution fields
  - or the active pointer’s `update_record_ref`

### Fail-closed replay cases

- rollback-apply record exists but linked rollback record is missing or invalid
- execution record claims switched state but active pointer does not match target pack
- active pointer matches target pack but `update_record_ref` points to a different workflow and the state cannot be explained by replay
- current active pack is neither `from_pack_id` nor `target_pack_id`

## 5. Failure Windows If Mutation Happens But Reload/Apply Does Not

| Failure window | Durable state after crash | Runtime state after crash | Required recovery posture |
|---|---|---|---|
| crash before any execution marker | rollback-apply still `ready_to_apply` | old pack still running | safe retry |
| crash after execution-start marker, before pointer write | apply marked started, pointer still old | old pack still running | safe retry, same apply id |
| crash after pointer write, before explicit reload-pending marker | pointer may already target rollback pack | runtime may still be old pack | reconcile from `update_record_ref`; convert to `pointer_switched_reload_pending` |
| crash after pointer switch and reload-generation increment, before reload | control plane says new active pack | runtime may still be old pack | do not re-switch; expose reload pending |
| reload attempt fails after pointer switch | control plane selects rollback target, reload incomplete | runtime may be old, mixed, or restarting | explicit failure state; recovery policy decides retry or reverse rollback |
| repeated crash on target pack after reload | pointer already switched | runtime unstable on target pack | quarantine / subsequent recovery policy, not part of first slice |

The most important boundary is this one:

- pointer mutation is durable truth
- reload/apply is runtime convergence

Those are not the same event. The first execution slice must expose that difference explicitly.

## 6. Acceptance Tests Required For The First Execution Slice

The first execution slice should add tests for:

1. successful execution from `ready_to_apply`
   - existing rollback-apply record
   - active pack initially equals rollback `from_pack_id`
   - active pointer changes to rollback `target_pack_id`
   - `previous_active_pack_id` becomes prior active
   - `reload_generation` increments once
   - `update_record_ref` is attributed to the rollback-apply workflow
   - last-known-good pointer remains byte-for-byte unchanged
2. idempotent replay after pointer switch
   - repeating the same execute entry does not re-switch
   - does not increment `reload_generation` again
   - returns deterministic already-present / reload-pending acknowledgement
3. invalid active-state rejection
   - current active pack matches neither rollback `from_pack_id` nor `target_pack_id`
   - command fails closed
4. missing linkage rejection
   - missing rollback record
   - missing target pack
   - invalid rollback-apply record
5. status/read-model coherence after pointer switch
   - `STATUS <job_id>` shows the rollback-apply workflow still linked to the same rollback
   - runtime-pack identity shows the switched active pack
   - execution/read-model clearly indicates reload is still pending
6. no last-known-good mutation
   - `last_known_good_pointer.json` does not change
7. no reload/apply side effects
   - no runtime reload function or process-apply path is invoked in the first slice

## Activation Mutation Boundaries

The first execution slice should mutate only:

- active runtime-pack selection
- rollback-apply execution state sufficient to describe pointer switch vs reload pending

It should not mutate:

- last-known-good certification
- promotion records
- hot-update outcomes
- evaluator state
- scoring state
- autonomy state
- provider/channel behavior

Recommended execution-state boundary:

- keep workflow phase as the authority for operator progression into execution readiness
- add separate execution state for runtime mutation progress

Reason:

- current `phase` already models workflow control
- current `activation_state` was introduced under a strict `unchanged` invariant
- overloading `phase` to mean both operator gating and runtime mutation would collapse two different truths

## Last-Known-Good Update Boundaries

The first execution slice should not update the durable last-known-good pointer.

It may preserve the existing active-pointer `last_known_good_pack_id` field, but it should not recertify or rewrite `last_known_good_pointer.json`.

Recommended rule:

- rollback apply may consume the rollback target and switch active selection
- post-rollback stability or verification may later decide whether last-known-good metadata changes

This keeps rollback execution narrower and preserves the spec rule that replay must not silently lose last-known-good truth.

## Reload / Apply Boundary Analysis

Reload/apply should not be in the first execution slice.

Why:

- reload is a separate failure boundary from pointer mutation
- process restart, skill reload, extension reload, and soft reload have materially different recovery stories
- the current codebase has durable pointer records but no committed rollback-apply execution ledger for partial reload progress

Recommended staged boundary:

1. consume `ready_to_apply`
2. durably switch the active pointer
3. record `reload_pending`
4. stop

Later slices can decide whether rollback apply uses:

- `pack_reload`
- `process_restart_hot_swap`
- a narrower reload mode for allowed surfaces

## 7. Smallest Safe First Implementation Slice

### Recommendation

The smallest safe first execution slice is:

- add a rollback-apply execute control entry on the existing direct-command path
- require an existing `RollbackApplyRecord` in phase `ready_to_apply`
- durably switch `ActiveRuntimePackPointer` to the rollback target pack
- increment `reload_generation`
- set `update_record_ref` to the rollback-apply workflow identity
- record rollback-apply execution state as `pointer_switched_reload_pending`
- preserve `LastKnownGoodRuntimePackPointer` unchanged
- do not perform any runtime reload, process restart, or pack-application mechanics yet

### Why this is the smallest safe slice

- it is the first slice that actually consumes rollback-apply authority and mutates activation/runtime-pack truth
- it keeps mutation within already-existing durable truth surfaces
- it makes replay tractable
- it avoids pretending reload/apply is atomic when it is not
- it keeps the recovery story inspectable through durable state

## Explicit Non-Goals For The First Execution Slice

- no runtime reload or process restart
- no reload success confirmation
- no last-known-good pointer rewrite
- no automatic rollback of a failed rollback-apply attempt
- no quarantine policy
- no promotion behavior changes
- no evaluator execution
- no scoring behavior
- no autonomy behavior changes
- no provider/channel behavior changes
- no attempt to make process reload atomic with pointer mutation

## Recommended Follow-On Slice Order

1. pointer-switch execution slice with `reload_pending`
2. read-model/status exposure for rollback-apply execution state
3. explicit reload/apply mechanics that consume `reload_pending`
4. recovery policy for crash or reload failure after pointer switch
5. only then evaluate whether any last-known-good recertification is needed
