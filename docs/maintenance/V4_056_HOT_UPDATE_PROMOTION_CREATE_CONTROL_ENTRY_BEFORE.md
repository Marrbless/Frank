# V4-056 Hot-Update Promotion Create Control Entry - Before

## Gap

After V4-054, missioncontrol could derive a deterministic `PromotionRecord` from an existing committed successful `HotUpdateOutcomeRecord`, but operators had no direct command for invoking that helper through the existing runtime control path.

The hot-update lane already had direct commands for gate creation, phase progression, pointer switch, reload/apply, terminal failure, and outcome creation. Promotion creation was still helper-only, so operators had to rely on code-level access instead of the established direct command surface.

## Existing Authority

The committed `HotUpdateOutcomeRecord` is the source authority for promotion creation. The originating `HotUpdateGateRecord` supplies the previous active pack linkage through `outcome.hot_update_id`.

The future command must not promote directly from a gate without an outcome and must not accept caller-supplied promotion fields.

## Required Control Shape

The smallest safe command shape is:

- `HOT_UPDATE_PROMOTION_CREATE <job_id> <outcome_id>`

Required behavior:

- validate `job_id` through the same active or persisted runtime control pattern as existing hot-update commands
- resolve the mission store root through TaskState
- derive the timestamp through the existing TaskState timestamp helper
- call `CreatePromotionFromSuccessfulHotUpdateOutcome(root, outcomeID, "operator", now)`
- emit runtime control audit action `hot_update_promotion_create`
- return deterministic `Created...` or `Selected...` acknowledgements

## Non-Goals

V4-056 must not create hot-update outcomes, mutate the active runtime-pack pointer, mutate `reload_generation`, mutate or recertify `last_known_good_pointer.json`, mutate hot-update gates, accept manual promotion fields, broaden policy/authorization, or start V4-057.
