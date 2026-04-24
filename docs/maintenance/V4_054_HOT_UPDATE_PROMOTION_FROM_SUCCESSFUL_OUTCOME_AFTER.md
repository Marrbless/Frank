# V4-054 Hot-Update Promotion From Successful Outcome - After

## Implemented Helper

V4-054 adds `CreatePromotionFromSuccessfulHotUpdateOutcome(root, outcomeID, createdBy, createdAt)` in missioncontrol.

The helper loads the committed `HotUpdateOutcomeRecord` by `outcome_id`, accepts only `outcome_kind = hot_updated`, loads the originating `HotUpdateGateRecord` through `outcome.hot_update_id`, and creates exactly one deterministic `PromotionRecord` when the linkage is valid.

## Deterministic Mapping

The derived promotion uses:

- `promotion_id = hot-update-promotion-<hot_update_id>`
- `promoted_pack_id = outcome.candidate_pack_id`
- `previous_active_pack_id = gate.previous_active_pack_id`
- `hot_update_id = outcome.hot_update_id`
- `outcome_id = outcome.outcome_id`
- `promoted_at = outcome.outcome_at`
- `created_at = helper input`
- `created_by = helper input`
- `reason = hot update outcome promoted`

Optional `candidate_id`, `run_id`, and `candidate_result_id` are copied only when present on the outcome and still pass existing promotion linkage validation. Last-known-good promotion fields remain empty.

## Fail-Closed Behavior

The helper rejects missing outcomes, invalid outcome linkage, missing originating gates, non-`hot_updated` outcomes, empty outcome `candidate_pack_id`, outcome/gate candidate pack mismatch, missing or unresolved gate `previous_active_pack_id`, divergent deterministic duplicates, existing promotions for the same `hot_update_id` under another `promotion_id`, and existing promotions for the same `outcome_id` under another `promotion_id`.

Exact replay of the same derived promotion is idempotent and returns `changed=false`. If the linked outcome or gate changes such that the derived promotion differs, the deterministic duplicate fails closed rather than rewriting.

## Preserved Invariants

This slice is ledger-only. It does not add a direct command or TaskState wrapper, does not mutate the active runtime-pack pointer, does not mutate `reload_generation`, does not mutate or recertify `last_known_good_pointer.json`, does not create a hot-update outcome, does not mutate hot-update gates, does not promote directly from gate success without an outcome, and does not start V4-055.
