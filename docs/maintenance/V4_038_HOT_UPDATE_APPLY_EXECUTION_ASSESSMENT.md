# V4-038 Hot-Update Apply/Reload Execution Assessment

## Current checkpoint facts

- branch: `frank-v4-038-hot-update-apply-execution-assessment`
- `HEAD`: `c9ec30ce1181ebf19d0399cd7d2171b64543f7dc`
- tags at `HEAD`:
  - `frank-v4-037-hot-update-gate-phase-control`
- ahead/behind `upstream/main`: `436 0`
- initial `git status --short --branch`:

```text
## frank-v4-038-hot-update-apply-execution-assessment
```

- baseline `go test -count=1 ./...`: passed

## Current hot-update gate / pointer / outcome / promotion truth

### Durable truth that exists today

- `HotUpdateGateRecord` is the durable authority for pre-apply hot-update workflow identity and linkage:
  - `hot_update_id`
  - `candidate_pack_id`
  - `previous_active_pack_id`
  - `rollback_target_pack_id`
  - `target_surfaces`
  - `surface_classes`
  - `reload_mode`
  - `compatibility_contract_ref`
  - `prepared_at`
  - `phase_updated_at`
  - `phase_updated_by`
  - `state`
  - `decision`
- gate states currently reachable in code are bounded to:
  - `prepared`
  - `validated`
  - `staged`
- `ActiveRuntimePackPointer` remains the durable authority for committed active selection:
  - `active_pack_id`
  - `previous_active_pack_id`
  - `last_known_good_pack_id`
  - `updated_at`
  - `updated_by`
  - `update_record_ref`
  - `reload_generation`
- `LastKnownGoodRuntimePackPointer` remains separate durable truth and is not touched by the current hot-update lane.
- `HotUpdateOutcomeRecord` already exists as a separate append-only terminal/outcome ledger.
- `PromotionRecord` already exists as a separate append-only promotion ledger linked to gate/outcome truth.

### Control-plane truth that exists today

- gate record creation/select exists through:
  - `HOT_UPDATE_GATE_RECORD <job_id> <hot_update_id> <candidate_pack_id>`
- gate phase progression exists through:
  - `HOT_UPDATE_GATE_PHASE <job_id> <hot_update_id> <phase>`
- read-only status exposure exists through:
  - `STATUS <job_id>`
  - `hot_update_gate_identity`

### Important current absence

- there is no hot-update apply execution helper in `internal/missioncontrol`
- there is no hot-update apply operator command in `internal/agent/loop.go`
- there is no durable execution-attempt model for post-`staged` hot-update behavior
- there is no committed reload/apply convergence helper for candidate packs
- there is no current hot-update outcome creation path driven by gate execution
- there is no promotion handoff driven by gate execution

## 1. Exact authority needed to consume a committed `staged` hot-update gate

The minimum safe authority is:

1. Read authority for:
   - the `HotUpdateGateRecord`
   - the `ActiveRuntimePackPointer`
   - the referenced candidate runtime pack
   - the referenced previous active runtime pack
   - the referenced rollback target runtime pack
2. Write authority for:
   - the `HotUpdateGateRecord`
   - the `ActiveRuntimePackPointer`
3. Runtime execution authority for one bounded convergence mechanism:
   - either a restart-style apply path
   - or a much larger new in-process reload framework

The smallest truthful path is the restart-style path. The repo does not currently expose a reusable in-process surface reload engine. The gate already carries a derived `reload_mode`, but there is no implementation hook behind those modes yet.

Outcome authority and promotion authority are not required for the first execution slice.

## 2. Minimum execution boundary for apply/reload

### Active runtime-pack pointer mutation

This belongs in the first execution slice.

Reason:

- the spec says active content changes only through the hot-update gate
- the durable pointer is the current source of truth for live selection
- consuming `staged` without pointer mutation would not produce a real execution state change

The first slice should switch the active pointer to `candidate_pack_id`.

### `reload_generation` mutation

This also belongs in the first execution slice and must happen exactly once with the pointer switch.

Reason:

- it is already the durable replay key used to signal runtime convergence pressure
- switching the pointer without incrementing `reload_generation` would make restart/reload reconciliation ambiguous

### Reload/apply mechanics

These should not be implemented in the first execution slice.

Reason:

- there is no existing hot-update reload engine
- adding one would widen the slice across runtime behavior, provider/channel coupling, and recovery policy
- the smallest truthful first slice is the same shape chosen for rollback: durable pointer switch plus explicit reload-pending execution state

### Outcome record creation

This should happen later, not in the first execution slice.

Reason:

- `HotUpdateOutcomeRecord` is append-only and already models terminal or decision-complete outcomes
- the first execution slice will not yet know whether the hot update actually converged, passed smoke checks, canary, committed, failed, or rolled back
- writing `hot_updated` too early would overstate runtime truth

### Promotion handoff or non-handoff

Promotion handoff should not occur in the first execution slice.

Reason:

- `PromotionRecord` is a separate durable contract
- promotion semantics include explicit reason, notes, promoted-at timestamp, optional outcome linkage, and last-known-good basis
- pointer switching to a staged candidate is not yet a promotion decision

## 3. What must be atomic vs crash-recoverable

### Must be atomic together

Inside the `active_pointer.json` write, these fields must move together:

- `active_pack_id = candidate_pack_id`
- `previous_active_pack_id = old active pack`
- `update_record_ref = hot_update:<hot_update_id>`
- `reload_generation = old + 1`

That is already one atomic file write under existing store semantics and should remain one write.

### Does not need to be one atomic transaction with the pointer write

- hot-update gate execution-state mutation
- the later reload/apply convergence attempt
- later outcome creation
- later promotion creation

These can be modeled as a crash-recoverable state machine as long as the gate state and pointer truth are durable enough to reconcile.

### Recommended crash-recoverable state machine

The smallest state progression after `staged` is:

1. `staged`
2. `reloading`
3. later explicit result state:
   - `smoke_testing`
   - `canarying`
   - `committed`
   - `failed`
   - `rolled_back`

For the first execution slice, only this transition is needed:

- `staged -> reloading`

and `reloading` should mean:

- pointer has switched to the candidate pack
- `reload_generation` has advanced
- runtime convergence is still pending

## 4. Replay / idempotence rules required

### Entry conditions

- `staged` may start execution
- `reloading` with a matching active pointer should be treated as already-switched / reload-pending
- later committed terminal states should not re-run the pointer switch

### Required replay key

- `hot_update_id` remains the workflow key
- `active_pointer.update_record_ref` should be set to `hot_update:<hot_update_id>`

### Required replay rules

- replay from `staged` may perform the pointer switch once
- replay after successful pointer switch must not:
  - switch the pointer a second time
  - increment `reload_generation` again
  - rewrite `previous_active_pack_id` differently
- replay when the active pointer already targets `candidate_pack_id` and `update_record_ref` already equals `hot_update:<hot_update_id>` should be idempotent
- replay should fail closed if:
  - the gate is missing
  - the gate is not `staged`
  - linked runtime packs are missing
  - the active pointer already changed in a way not attributable to this `hot_update_id`

## 5. Before / during / after invariants

### Before execution

- the gate record exists
- the gate state is exactly `staged`
- the candidate runtime pack exists
- the previous active runtime pack exists
- the rollback target runtime pack exists
- `previous_active_pack_id` on the gate matches the current active pointer `active_pack_id`
- the candidate pack `rollback_target_pack_id` matches the gate `rollback_target_pack_id`

### During the first execution slice

- no outcome record is created
- no promotion record is created
- no rollback record is created
- no last-known-good pointer mutation occurs
- no second candidate selection or new gate record is created
- no evaluator, scoring, autonomy, provider, or channel behavior changes occur

### After the first execution slice succeeds

- `active_pointer.active_pack_id == gate.candidate_pack_id`
- `active_pointer.previous_active_pack_id == old active pack`
- `active_pointer.update_record_ref == hot_update:<hot_update_id>`
- `active_pointer.reload_generation` has increased exactly once
- gate state becomes `reloading`
- `last_known_good_pointer.json` remains unchanged
- there is still no outcome or promotion record

## 6. Should outcome creation happen in the first execution slice or later?

Outcome creation should happen later.

Reason:

- the repo already treats `HotUpdateOutcomeRecord` as a separate append-only decision/result surface
- the first execution slice will only establish that the candidate is now the selected active target and that reload is pending
- outcome kinds such as `hot_updated`, `failed`, `promoted`, or `rolled_back` would overclaim runtime truth before convergence, smoke checks, and later policy steps complete

The first execution slice should therefore stop at durable pointer switch plus `reloading` state, with no outcome write.

## 7. Acceptance tests required for the smallest safe implementation slice

The first implementation slice should have narrow tests for:

1. happy path pointer switch from `staged`
   - active pointer changes from `previous_active_pack_id` to `candidate_pack_id`
2. `reload_generation` increments exactly once
3. `update_record_ref` becomes `hot_update:<hot_update_id>`
4. gate state becomes `reloading`
5. exact replay is idempotent
   - no second pointer switch
   - no second `reload_generation` increment
6. `last_known_good_pointer.json` remains unchanged
7. missing or invalid linkage rejects without pointer mutation
8. invalid starting state rejects without pointer mutation
9. status/read-model remains coherent after pointer switch
   - existing `hot_update_gate_identity` reports the new gate state
   - existing `runtime_pack_identity` reports the switched active pointer

## 8. Explicit non-goals for the first execution slice

- no reload/apply runtime convergence implementation
- no smoke test execution
- no canary execution
- no hot-update outcome record creation
- no promotion record creation
- no rollback behavior changes
- no last-known-good pointer mutation
- no second pointer mutation on replay
- no new provider/channel behavior
- no evaluator execution
- no scoring behavior
- no autonomy changes

## Smallest safe next implementation slice recommendation

The smallest safe next slice is:

1. require a committed `HotUpdateGateRecord` in `staged`
2. validate the linked candidate, previous-active, and rollback-target pack linkage against current pointer truth
3. switch `active_pointer.json` to `candidate_pack_id`
4. increment `reload_generation`
5. set `update_record_ref` to `hot_update:<hot_update_id>`
6. advance the gate state to `reloading`
7. make exact replay idempotent when the pointer already reflects the same `hot_update_id`
8. stop there

This is the smallest truthful first execution slice because it:

- performs the first real hot-update activation mutation
- remains compatible with the spec requirement that active content changes only through the hot-update gate
- does not prematurely claim convergence, outcome, or promotion
- keeps crash recovery legible through pointer truth plus gate phase
