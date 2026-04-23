# V4-042 Hot-Update Reload/Apply Recovery Assessment

## Current checkpoint facts

- branch: `frank-v4-042-hot-update-recovery-assessment`
- `HEAD`: `0d3a02a1fae864d22053b41469edbfa75796ffda`
- tags at `HEAD`:
  - `frank-v4-041-hot-update-reload-apply-skeleton`
- ahead/behind `upstream/main`: `440 0`
- initial `git status --short --branch`:

```text
## frank-v4-042-hot-update-recovery-assessment
```

- baseline `go test -count=1 ./...`: passed

## Current hot-update recovery truth

### Durable truth that exists today

- `HotUpdateGateRecord` is still the only durable workflow authority for the hot-update lane.
- current committed gate states now include:
  - `prepared`
  - `validated`
  - `staged`
  - `reloading`
  - `reload_apply_in_progress`
  - `reload_apply_succeeded`
  - `reload_apply_failed`
- `ExecuteHotUpdateGateReloadApply(...)` currently:
  - requires `reloading`
  - validates active pointer attribution and gate linkage
  - writes `reload_apply_in_progress`
  - runs one bounded restart-style convergence check
  - writes either `reload_apply_succeeded` or `reload_apply_failed`
- `ActiveRuntimePackPointer` remains separate durable truth:
  - `active_pack_id`
  - `previous_active_pack_id`
  - `last_known_good_pack_id`
  - `update_record_ref`
  - `reload_generation`
- `LastKnownGoodRuntimePackPointer` remains separate durable truth and is intentionally untouched by V4-039 through V4-041.
- `HotUpdateOutcomeRecord` and `PromotionRecord` still exist as separate append-only ledgers and are not written by V4-041.

### Important current absence

- there is no hot-update recovery-specific gate state
- there is no recovery reconciliation helper for persisted `reload_apply_in_progress`
- there is no durable receipt between:
  - `reload_apply_in_progress`
  - and terminal `reload_apply_succeeded` / `reload_apply_failed`
- there is no safe automatic way to infer whether a crash in `reload_apply_in_progress` means:
  - success before crash
  - failure before crash
  - or genuinely unknown outcome

## 1. What exact on-disk evidence is authoritative after a crash in `reload_apply_in_progress`?

After a crash in `reload_apply_in_progress`, the authoritative on-disk evidence is limited to:

- the `HotUpdateGateRecord`
  - `hot_update_id`
  - `candidate_pack_id`
  - `previous_active_pack_id`
  - `rollback_target_pack_id`
  - `state = reload_apply_in_progress`
  - `phase_updated_at`
  - `phase_updated_by`
  - `failure_reason` still empty
- the `ActiveRuntimePackPointer`
  - `active_pack_id` already set to `candidate_pack_id`
  - `previous_active_pack_id` already set
  - `update_record_ref = hot_update:<hot_update_id>`
  - `reload_generation` already incremented by the earlier pointer-switch slice
- the `LastKnownGoodRuntimePackPointer`
  - unchanged from its pre-hot-update state
- absence of any `HotUpdateOutcomeRecord` or `PromotionRecord` for this gate

That evidence is authoritative for:

- the pointer switch already happened earlier
- the gate entered convergence
- the active pointer is still attributed to this hot update
- last-known-good has not been recertified
- no terminal outcome or promotion has been durably recorded

## 2. What can distinguish success, failure, and unknown?

Today, not enough durable evidence exists to distinguish these reliably.

### Can prove success?

Only if the gate already reached `reload_apply_succeeded`.

If the gate is still `reload_apply_in_progress`, success is not proven.

### Can prove failure?

Only if the gate already reached `reload_apply_failed` with a durable `failure_reason`.

If the gate is still `reload_apply_in_progress`, failure is not proven.

### Can prove unknown?

Yes. A persisted `reload_apply_in_progress` after crash/restart is an unknown-outcome state.

Reason:

- the bounded restart-style convergence path in V4-041 does not emit any separate durable receipt before terminal gate-state write
- a crash can happen:
  - after entering `reload_apply_in_progress`
  - before writing `reload_apply_succeeded`
  - before writing `reload_apply_failed`

So `reload_apply_in_progress` after restart means:

```text
pointer already switched + convergence started + terminal outcome not durably recorded
```

and nothing more.

## 3. What should the first recovery slice do?

The smallest safe first recovery slice should add a new recovery-needed gate state and require explicit operator recovery action later.

Recommended answer:

- do not mark unknown as failed automatically
- do not retry automatically
- do not infer success automatically
- add a new state:
  - `reload_apply_recovery_needed`

Reason:

- auto-fail would overclaim a result not durably proven
- auto-retry would repeat execution from an ambiguous post-crash state
- auto-success would overclaim runtime convergence without proof
- a recovery-needed state preserves the exact truth: outcome is unknown and needs explicit resolution

So the first recovery slice should normalize:

```text
reload_apply_in_progress -> reload_apply_recovery_needed
```

only when pointer attribution and gate linkage are still coherent.

## 4. What replay / idempotence rules are required on recovery?

The recovery slice needs strict replay rules:

### For normalization

- `reload_apply_in_progress -> reload_apply_recovery_needed` should be allowed once
- replay on `reload_apply_recovery_needed` should be idempotent

### For invalid linkage

- if active pointer attribution no longer matches `hot_update:<hot_update_id>`, fail closed
- if active pointer no longer points at `candidate_pack_id`, fail closed
- if candidate / previous-active / rollback-target linkage is broken, fail closed

### For later operator resolution

Those later slices should not be part of V4-042, but the recovery assessment must preserve space for them:

- explicit retry on the same `hot_update_id`
- explicit terminal failure decision

Neither should happen implicitly during recovery normalization.

## 5. What invariants must remain fixed during recovery?

These invariants must remain unchanged throughout the first recovery slice:

- no second active pointer switch
- no second `reload_generation` increment
- no `last_known_good_pointer.json` mutation
- no `HotUpdateOutcomeRecord` creation
- no `PromotionRecord` creation
- no change to `update_record_ref`
- no rewrite of candidate / previous-active / rollback-target linkage

Recovery normalization should be gate-state-only plus validation.

## 6. Should outcome creation remain deferred until after recovery semantics are explicit?

Yes.

`HotUpdateOutcomeRecord` creation should remain deferred until after recovery semantics are explicit and terminal runtime truth is known.

Reason:

- outcome records are append-only terminal-result truth
- `reload_apply_recovery_needed` is not terminal success or terminal failure
- creating an outcome during ambiguity would either:
  - overstate success
  - overstate failure
  - or invent a new outcome semantic too early

Promotion creation must remain deferred for the same reason.

## 7. What acceptance tests are required for the smallest safe recovery slice?

The smallest safe recovery slice should add narrow tests for:

1. `reload_apply_in_progress` normalizes to `reload_apply_recovery_needed` when pointer attribution and gate linkage remain coherent
2. replay on `reload_apply_recovery_needed` is idempotent
3. invalid active-pointer attribution rejects without mutating pointer state
4. invalid candidate / previous-active / rollback-target linkage rejects without mutating pointer state
5. `reload_generation` remains unchanged during normalization
6. `last_known_good_pointer.json` remains unchanged during normalization
7. no `HotUpdateOutcomeRecord` is created during normalization
8. no `PromotionRecord` is created during normalization
9. status/read-model coherence remains truthful after normalization

If an operator control entry is later added for recovery normalization, one integration test should verify that the existing direct-command/status path reports `reload_apply_recovery_needed` without changing pointer state.

## 8. Explicit non-goals for the first recovery slice

The first recovery slice should explicitly not do any of the following:

- no second pointer switch
- no second `reload_generation` increment
- no `last_known_good_pointer.json` mutation
- no automatic retry
- no automatic success inference
- no automatic terminal failure inference
- no `HotUpdateOutcomeRecord` creation
- no `PromotionRecord` creation
- no smoke/canary execution
- no rollback execution changes
- no provider/channel/evaluator/scoring/autonomy changes

## Recommended smallest safe next implementation slice

The smallest safe next slice after this assessment is:

1. add gate state:
   - `reload_apply_recovery_needed`
2. add a bounded reconciliation helper in `internal/missioncontrol` that:
   - consumes persisted `reload_apply_in_progress`
   - validates active pointer attribution and gate linkage
   - normalizes to `reload_apply_recovery_needed` when outcome is still unknown
3. keep pointer state, `reload_generation`, last-known-good, outcome records, and promotion records unchanged
4. defer explicit retry and explicit terminal-failure resolution to later operator-driven slices

That is the smallest truthful recovery model because it records uncertainty without inventing execution that did not happen and without erasing the evidence that convergence was interrupted.
