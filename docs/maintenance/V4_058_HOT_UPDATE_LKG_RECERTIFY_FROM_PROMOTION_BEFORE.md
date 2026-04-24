# V4-058 Hot-Update LKG Recertification From Promotion - Before

## Gap

After V4-056, the hot-update lane could create a `PromotionRecord` from a successful `HotUpdateOutcomeRecord`, but there was no bounded missioncontrol helper for recertifying the promoted runtime pack as last-known-good.

Operators and future control entries had durable gate, outcome, and promotion authority, but the final `last_known_good_pointer.json` mutation still lacked a deterministic helper with explicit guards.

## Existing Authority Chain

Before this slice, the committed records already established the required source chain:

- `HotUpdateGateRecord` recorded the candidate pack, previous active pack, and terminal reload/apply state.
- `HotUpdateOutcomeRecord` recorded the successful hot-update outcome and candidate pack linkage.
- `PromotionRecord` recorded the promoted pack, previous active pack, hot-update id, and outcome id.
- `active_pointer.json` recorded the current active runtime pack and `reload_generation`.
- `last_known_good_pointer.json` recorded the current last-known-good pack.

## Missing Behavior

There was no helper that:

- Loaded a committed `PromotionRecord` as the sole recertification authority.
- Required a linked successful hot-update outcome.
- Required the active pointer to still point at the promoted pack.
- Replaced the current last-known-good pointer only when it still pointed at the promotion's previous active pack.
- Produced an idempotent deterministic last-known-good pointer for exact replay.

## Preserved Boundaries

This slice was scoped to missioncontrol helper behavior only. It was not intended to add a direct command, add a `TaskState` wrapper, mutate `active_pointer.json`, increment or mutate `reload_generation`, create hot-update outcomes, create or mutate promotions, mutate hot-update gates, add rollback behavior, or start V4-059.
