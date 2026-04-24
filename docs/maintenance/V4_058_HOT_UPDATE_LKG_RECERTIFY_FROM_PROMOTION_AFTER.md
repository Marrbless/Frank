# V4-058 Hot-Update LKG Recertification From Promotion - After

## Implemented Helper

V4-058 adds a missioncontrol-only helper:

`RecertifyLastKnownGoodFromPromotion(root, promotionID, verifiedBy, verifiedAt)`

The helper loads an existing committed `PromotionRecord`, requires `outcome_id`, loads the linked `HotUpdateOutcomeRecord`, and accepts only `outcome_kind = hot_updated`. Existing promotion validation continues to enforce coherent promotion, gate, outcome, pack, and optional candidate/run/result linkage.

## Source Authority Chain

The helper recertifies only through this committed chain:

- `PromotionRecord` is the direct source authority.
- The promotion must link to an existing hot-update outcome.
- The linked outcome must be `hot_updated`.
- The promotion's existing linkage to the originating hot-update gate must remain valid.

The helper does not recertify directly from a hot-update gate and does not recertify directly from a hot-update outcome without a promotion.

## Active Pointer Guard

Before writing last-known-good, the helper loads `active_pointer.json` and requires:

`active_pointer.active_pack_id == promotion.promoted_pack_id`

Missing, invalid, or mismatched active pointers fail closed. The helper does not mutate `active_pointer.json` and does not increment or mutate `reload_generation`.

## Current LKG Replacement Rules

The helper loads `last_known_good_pointer.json` and applies these rules:

- Missing or invalid current LKG rejects.
- First replacement is allowed only when the current LKG pack equals `promotion.previous_active_pack_id`.
- Exact replay is allowed when the current LKG already equals the deterministic recertified pointer.
- If current LKG points to any other pack, the helper fails closed.
- If current LKG already points to the promoted pack but basis, rollback ref, verification actor, or verification time differ from the deterministic pointer, the helper fails closed.

## Deterministic Pointer Mapping

The written `LastKnownGoodRuntimePackPointer` is deterministic:

- `pack_id = PromotionRecord.PromotedPackID`
- `basis = hot_update_promotion:<promotion_id>`
- `verified_at = helper input`
- `verified_by = helper input`
- `rollback_record_ref = hot_update_promotion:<promotion_id>`

## Replay And Invariants

First successful replacement writes only `runtime_packs/last_known_good_pointer.json` and returns `changed=true`. Exact replay returns `changed=false` and leaves bytes unchanged. Divergent current LKG, changed active pointer state, or changed authority linkage fails closed rather than rewriting.

This slice explicitly does not add a direct command, does not add a `TaskState` wrapper, does not mutate the active pointer, does not mutate `reload_generation`, does not create a `HotUpdateOutcomeRecord`, does not create or mutate a `PromotionRecord`, does not mutate hot-update gates, does not add rollback behavior, and does not start V4-059.
