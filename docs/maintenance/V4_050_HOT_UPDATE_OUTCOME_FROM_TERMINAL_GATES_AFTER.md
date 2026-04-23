# V4-050 Hot-Update Outcome From Terminal Gates After

## Implemented Helper Behavior

V4-050 adds a missioncontrol helper that creates a `HotUpdateOutcomeRecord` from an existing committed terminal `HotUpdateGateRecord`.

The committed gate remains the source authority. The helper loads the gate by `hot_update_id`, validates existing gate linkage, accepts only supported terminal states, derives one deterministic outcome, and writes it through the existing append-only outcome registry.

## Eligible Terminal States

The helper accepts only:

- `reload_apply_succeeded`
- `reload_apply_failed`

The helper rejects non-terminal states, including:

- `prepared`
- `validated`
- `staged`
- `reloading`
- `reload_apply_in_progress`
- `reload_apply_recovery_needed`

`reload_apply_recovery_needed` still requires explicit operator resolution before an outcome can be created.

## Deterministic Outcome Mapping

The outcome identity is:

- `hot-update-outcome-<hot_update_id>`

For `reload_apply_succeeded`:

- `outcome_kind`: `hot_updated`
- `reason`: `hot update reload/apply succeeded`

For `reload_apply_failed`:

- `outcome_kind`: `failed`
- `reason`: copied from `HotUpdateGateRecord.FailureReason`

A failed gate with empty or whitespace-only `failure_reason` fails closed.

The helper always populates:

- `hot_update_id` from the gate
- `candidate_pack_id` from the gate
- `outcome_at` from `HotUpdateGateRecord.PhaseUpdatedAt`
- `created_at` from helper input
- `created_by` from helper input

For V4-050, these optional refs remain empty because the gate does not store them directly:

- `candidate_id`
- `run_id`
- `candidate_result_id`

## Replay And Idempotence

First creation writes the outcome and reports `changed=true`.

Exact replay of the same derived outcome reports `changed=false` and leaves the existing outcome unchanged.

A divergent existing outcome with the same deterministic `outcome_id` fails closed.

Any existing outcome for the same `hot_update_id` with a different `outcome_id` fails closed.

If the committed gate later changes so the derived outcome would differ, the helper fails closed instead of rewriting the existing outcome.

## Invariants Preserved

V4-050 does not create promotions, mutate the active runtime-pack pointer, increment `reload_generation`, mutate `last_known_good_pointer.json`, mutate hot-update gates, create new gates, add commands, or start V4-051.

The implementation creates only the deterministic hot-update outcome ledger record when the committed terminal gate permits it.
