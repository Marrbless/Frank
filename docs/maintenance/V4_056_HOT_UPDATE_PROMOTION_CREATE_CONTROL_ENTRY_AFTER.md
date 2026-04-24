# V4-056 Hot-Update Promotion Create Control Entry - After

## Implemented Command

V4-056 adds direct operator command support for:

- `HOT_UPDATE_PROMOTION_CREATE <job_id> <outcome_id>`

The command is parsed in the existing direct operator command path and calls a new TaskState wrapper:

- `(*TaskState).CreatePromotionFromSuccessfulHotUpdateOutcome(jobID, outcomeID string) (bool, error)`

## TaskState Wrapper Behavior

The wrapper follows the existing hot-update control pattern:

- validates the mission store root
- validates `job_id` against the active execution context or persisted runtime control context
- derives `now` with `taskStateTransitionTimestamp(taskStateNowUTC())`
- calls `missioncontrol.CreatePromotionFromSuccessfulHotUpdateOutcome(root, outcomeID, "operator", now)`
- emits audit action `hot_update_promotion_create`
- returns the missioncontrol `changed` flag unchanged

For exact command replay, the wrapper preserves the existing deterministic promotion file by retrying with the stored promotion `created_at` only when the deterministic promotion already exists. Divergent duplicates and helper fail-closed errors still reject.

## Direct Response Semantics

Successful creation returns:

- `Created hot-update promotion job=<job_id> outcome=<outcome_id>.`

Exact replay returns:

- `Selected hot-update promotion job=<job_id> outcome=<outcome_id>.`

Failures return an empty direct response plus the underlying error, consistent with existing direct command behavior.

## Failure Behavior

The command preserves V4-054 helper failures for missing outcomes, non-`hot_updated` outcomes, failed outcomes, missing originating gates, invalid outcome/gate linkage, empty `candidate_pack_id`, missing or unresolved `previous_active_pack_id`, divergent deterministic promotions, existing promotions for the same `hot_update_id` under a different `promotion_id`, and existing promotions for the same `outcome_id` under a different `promotion_id`.

Wrong `job_id` rejects through existing TaskState job validation.

## Status / Read Model

After successful creation, `STATUS <job_id>` surfaces the promotion through existing `promotion_identity` output. The status read model exposes the deterministic promotion ID, promoted pack, previous active pack, source hot-update ID, source outcome ID, reason, promoted timestamp, created timestamp, and created actor.

No separate status command was added.

## Preserved Invariants

This slice does not create a `HotUpdateOutcomeRecord`, does not mutate the active runtime-pack pointer, does not mutate `reload_generation`, does not mutate or recertify `last_known_good_pointer.json`, does not mutate hot-update gates, does not promote directly from a gate without an outcome, does not accept manual promotion IDs, reasons, promoted pack refs, previous active pack refs, last-known-good fields, candidate/run/result refs, timestamps, or actors, and does not start V4-057.
