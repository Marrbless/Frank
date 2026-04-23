# V4-040 Hot-Update Reload/Apply Convergence Assessment

## Current checkpoint facts

- branch: `frank-v4-040-hot-update-reload-apply-assessment`
- `HEAD`: `f963f3c67b28b996681f16d982b43cd42168c5f5`
- tags at `HEAD`:
  - `frank-v4-039-hot-update-pointer-switch-skeleton`
- ahead/behind `upstream/main`: `438 0`
- initial `git status --short --branch`:

```text
## frank-v4-040-hot-update-reload-apply-assessment
```

- baseline `go test -count=1 ./...`: passed

## Current hot-update gate / pointer / outcome / promotion truth

### Durable truth that exists today

- `HotUpdateGateRecord` is the durable authority for hot-update workflow identity and current gate state:
  - `hot_update_id`
  - `candidate_pack_id`
  - `previous_active_pack_id`
  - `rollback_target_pack_id`
  - `target_surfaces`
  - `surface_classes`
  - `reload_mode`
  - `compatibility_contract_ref`
  - `phase_updated_at`
  - `phase_updated_by`
  - `state`
  - `decision`
- current committed gate states now include at least:
  - `prepared`
  - `validated`
  - `staged`
  - `reloading`
- `ActiveRuntimePackPointer` remains the durable authority for committed active selection:
  - `active_pack_id`
  - `previous_active_pack_id`
  - `last_known_good_pack_id`
  - `updated_at`
  - `updated_by`
  - `update_record_ref`
  - `reload_generation`
- `LastKnownGoodRuntimePackPointer` remains separate durable truth and was intentionally left unchanged by V4-039.
- `HotUpdateOutcomeRecord` exists as a separate append-only outcome ledger.
- `PromotionRecord` exists as a separate append-only promotion ledger.

### Control-plane truth that exists today

- `HOT_UPDATE_GATE_RECORD <job_id> <hot_update_id> <candidate_pack_id>`
  - creates/selects the gate
- `HOT_UPDATE_GATE_PHASE <job_id> <hot_update_id> <phase>`
  - advances `prepared -> validated -> staged`
- `HOT_UPDATE_GATE_EXECUTE <job_id> <hot_update_id>`
  - requires `staged`
  - switches the active pointer to `candidate_pack_id`
  - increments `reload_generation`
  - sets `update_record_ref = hot_update:<hot_update_id>`
  - advances the gate to `reloading`
  - is idempotent on exact replay
- `STATUS <job_id>` can already show:
  - `hot_update_gate_identity`
  - `runtime_pack_identity`

### Important current absence

- there is still no hot-update reload/apply convergence helper
- there is still no durable convergence-attempt/result model beyond gate state `reloading`
- there is still no durable way to distinguish:
  - reload/apply succeeded
  - reload/apply failed
  - reload/apply outcome unknown after crash
- there is still no outcome creation path driven by convergence
- there is still no promotion handoff path driven by convergence

## 1. Exact authority needed to consume a committed gate already in `reloading`

The minimum safe authority for the next slice is:

1. Read authority for:
   - the `HotUpdateGateRecord`
   - the `ActiveRuntimePackPointer`
   - the referenced candidate runtime pack
   - the referenced previous active runtime pack
   - the referenced rollback target runtime pack
2. Write authority for:
   - the `HotUpdateGateRecord`
3. Runtime execution authority for one bounded convergence mechanism:
   - preferably a controlled restart-style convergence path
4. Recovery/reconciliation authority:
   - enough to observe a persisted in-progress convergence attempt after crash/restart and normalize or complete it deterministically

The next slice does not need outcome-write authority or promotion-write authority if it stops short of terminal result recording.

## 2. Durable evidence that can distinguish success, failure, and unknown-after-crash

Today, durable evidence is insufficient.

After V4-039, the repo can prove only:

- the gate reached `reloading`
- the active pointer names the candidate pack
- `reload_generation` already advanced
- `update_record_ref` already identifies `hot_update:<hot_update_id>`
- `last_known_good_pointer.json` remains unchanged

That cannot distinguish:

- reload/apply actually succeeded before crash
- reload/apply actually failed before crash
- outcome is unknown because the process crashed mid-convergence

To distinguish those cases durably, the next lane needs explicit convergence attempt/result state on the gate record.

## 3. Should the first convergence slice add explicit states?

Yes.

The smallest safe model is to add explicit convergence states on the gate record:

- `reload_apply_in_progress`
- `reload_apply_succeeded`
- `reload_apply_failed`

The first convergence slice does not need `reload_apply_recovery_needed` yet if it can fail closed by leaving `reload_apply_in_progress` as the unknown-outcome crash state. But that means a later recovery slice will still be required.

Recommended minimal stance:

- first convergence slice adds:
  - `reload_apply_in_progress`
  - `reload_apply_succeeded`
  - `reload_apply_failed`
- later recovery slice may add:
  - `reload_apply_recovery_needed`

Reason:

- `reload_apply_in_progress` is the smallest necessary crash marker
- `reload_apply_succeeded` and `reload_apply_failed` are the smallest durable terminal execution truths
- `reload_apply_recovery_needed` is useful, but not required for the first convergence slice if crash-recovery handling is explicitly deferred

## 4. Minimum execution boundary for convergence

### Gate state mutation

This belongs in the first convergence slice.

The next slice should:

1. consume `reloading`
2. write `reload_apply_in_progress`
3. invoke one bounded convergence mechanism
4. write either:
   - `reload_apply_succeeded`
   - or `reload_apply_failed`

### Reload/apply mechanics

This also belongs in the first convergence slice, but only as one bounded convergence mechanism.

The smallest truthful mechanism is restart-style convergence aligned with the spec’s `process_restart_hot_swap` direction, not a new generic in-process reload framework.

Reason:

- the repo still has no reusable in-process pack apply API
- the spec explicitly names `process_restart_hot_swap`
- restart-style convergence is smaller than designing a generic `soft_reload`/`skill_reload`/`extension_reload` engine

### Outcome record creation

This should not belong in the first convergence slice.

Reason:

- `HotUpdateOutcomeRecord` is append-only terminal-result truth
- the first convergence slice should establish runtime convergence only
- smoke check, canary, commit, rollback, and promotion decisions still remain separate concerns

### Promotion handoff or non-handoff

Promotion handoff should also remain out of scope for the first convergence slice.

Reason:

- `PromotionRecord` is a separate durable contract
- promotion requires explicit reason/basis semantics beyond “candidate converged successfully”
- convergence is not itself a promotion decision

## 5. Which pieces must be atomic vs modeled as a crash-recoverable state machine?

### Already atomic enough today

The pointer-switch write is already done and should not be repeated:

- `active_pack_id`
- `previous_active_pack_id`
- `update_record_ref`
- `reload_generation`

That should remain untouched by the convergence slice.

### Not atomic and must be modeled as a crash-recoverable state machine

- gate convergence start marker
- runtime restart/apply side effect
- gate success/failure write

These cannot be one atomic transaction across a process boundary.

The minimum crash-recoverable state machine is:

1. `reloading`
2. `reload_apply_in_progress`
3. terminal result:
   - `reload_apply_succeeded`
   - or `reload_apply_failed`

If the process crashes after step 2 but before step 3, the gate remains `reload_apply_in_progress` and later recovery logic must reconcile it.

## 6. Replay / idempotence rules required after partial success or partial failure

### Entry rules

- if gate state is `reloading`, convergence may start
- if gate state is `reload_apply_in_progress`, blind replay should not occur without explicit recovery handling
- if gate state is `reload_apply_succeeded`, return idempotent already-applied success
- if gate state is `reload_apply_failed`, retry should be explicit and policy-bounded, not implicit

### Required replay invariants

- never switch the pointer again
- never increment `reload_generation` again
- never rewrite `update_record_ref` away from `hot_update:<hot_update_id>` during retry/replay
- never mutate `last_known_good_pointer.json` during convergence replay

### Partial-success / partial-failure rules

- if the runtime already restarted onto the candidate but success was not durably recorded, the next slice must not fake success without explicit durable evidence
- if convergence failed and `reload_apply_failed` was durably written, replay should fail closed unless a later explicit retry slice is added
- if convergence crashed in `reload_apply_in_progress`, later recovery should reconcile rather than starting a second pointer-switch path

## 7. Should `HotUpdateOutcomeRecord` creation belong in the first convergence slice?

No. It should belong in a later terminal-result slice.

Reason:

- the current outcome ledger models explicit end states like `hot_updated`, `failed`, `promoted`, `rolled_back`
- the first convergence slice should not overclaim more than “runtime convergence succeeded/failed”
- smoke-test, canary, commit, rollback, and promotion policy are still separate downstream decisions

The first convergence slice should therefore stop at durable convergence truth on the gate record, with no outcome write.

## 8. Before / during / after invariants

### Before convergence

- the gate record exists
- the gate state is exactly `reloading`
- the active pointer already targets `candidate_pack_id`
- `update_record_ref == hot_update:<hot_update_id>`
- `reload_generation` already reflects the earlier pointer-switch slice
- candidate pack linkage is still valid
- previous-active and rollback-target linkage is still valid

### During convergence

- no second pointer switch
- no second `reload_generation` increment
- no `last_known_good_pointer.json` mutation
- no `HotUpdateOutcomeRecord` creation
- no `PromotionRecord` creation
- the gate remains the sole authority for convergence-attempt truth

### After convergence success

- pointer remains on the candidate pack
- `reload_generation` remains unchanged from the pre-convergence value
- `last_known_good_pointer.json` remains unchanged
- gate transitions to `reload_apply_succeeded`
- outcome/promotion truth still does not exist yet

### After convergence failure

- pointer remains on the candidate pack unless a later recovery/rollback slice changes it
- `reload_generation` remains unchanged
- `last_known_good_pointer.json` remains unchanged
- gate transitions to `reload_apply_failed`
- outcome/promotion truth still does not exist yet

### Authority boundaries

- gate record remains authority for pre-terminal execution state
- outcome record remains authority for later terminal result classification
- promotion record remains authority for later promotion semantics

## 9. Acceptance tests required for the smallest safe implementation slice

The first convergence slice should have narrow tests for:

1. happy path from `reloading` to `reload_apply_succeeded`
   - writes `reload_apply_in_progress`
   - invokes the bounded restart/apply path
   - records `reload_apply_succeeded`
2. failure path from `reloading` to `reload_apply_failed`
3. exact replay after success is idempotent
4. no second pointer switch
5. no second `reload_generation` increment
6. `last_known_good_pointer.json` remains unchanged
7. invalid starting phase rejects
8. invalid pointer attribution rejects
9. no `HotUpdateOutcomeRecord` created
10. no `PromotionRecord` created
11. existing status/read-model remains coherent after convergence result

## 10. Explicit non-goals for the first convergence slice

- no second pointer mutation
- no second `reload_generation` increment
- no `last_known_good_pointer.json` mutation
- no `HotUpdateOutcomeRecord` creation
- no `PromotionRecord` creation
- no smoke-test execution
- no canary execution
- no commit/promotion decision
- no rollback or recovery policy implementation
- no provider/channel behavior changes beyond the bounded restart-style convergence path
- no evaluator execution
- no scoring behavior
- no autonomy changes

## Smallest safe next implementation slice recommendation

The smallest safe next slice is:

1. consume a committed `HotUpdateGateRecord` in `reloading`
2. add gate states:
   - `reload_apply_in_progress`
   - `reload_apply_succeeded`
   - `reload_apply_failed`
3. write `reload_apply_in_progress`
4. invoke one bounded restart-style convergence path
5. write `reload_apply_succeeded` or `reload_apply_failed`
6. leave pointer, `reload_generation`, `last_known_good`, outcome, and promotion otherwise unchanged

This is the smallest truthful convergence slice because it:

- adds the first real runtime-convergence behavior after the pointer switch
- records durable success/failure truth without overclaiming terminal hot-update outcome
- preserves the already-selected pointer state
- keeps later outcome/promotion/rollback semantics in separate slices
