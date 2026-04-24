# V4-060 Hot-Update LKG Recertify Control Entry - After

## Command Shape

V4-060 adds the direct operator command:

`HOT_UPDATE_LKG_RECERTIFY <job_id> <promotion_id>`

The command accepts only the job id and promotion id. It does not accept manual pack ids, previous active pack ids, outcome ids, hot-update ids, basis values, rollback refs, verification timestamps, verification actors, reload generation values, or active pointer fields.

## TaskState Wrapper Behavior

The command dispatches through:

`(*TaskState).RecertifyLastKnownGoodFromPromotion(jobID, promotionID string) (bool, error)`

The wrapper follows the existing hot-update direct-control pattern:

- Validates the mission store root.
- Requires the active execution context job or persisted runtime control job to match `job_id`.
- Derives the first timestamp through the existing TaskState timestamp helper.
- Calls `missioncontrol.RecertifyLastKnownGoodFromPromotion(root, promotionID, "operator", now)`.
- Emits runtime-control audit action `hot_update_lkg_recertify`.
- Returns the missioncontrol `changed` flag unchanged.

## Replay Timestamp Reuse

First execution uses the current TaskState timestamp. Exact command replay must not fail just because a new wrapper timestamp was derived.

If the helper rejects because the current LKG already points to the promoted pack but differs from deterministic recertification, the wrapper loads the committed promotion and current `last_known_good_pointer.json`. It retries only when the current LKG has:

- `pack_id = promotion.promoted_pack_id`
- `basis = hot_update_promotion:<promotion_id>`
- `rollback_record_ref = hot_update_promotion:<promotion_id>`

The retry uses the stored LKG `verified_at`. The wrapper does not retry with arbitrary timestamps and does not retry unrelated fail-closed errors.

## Direct Responses

Successful first mutation returns:

`Recertified hot-update last-known-good job=<job_id> promotion=<promotion_id>.`

Exact replay returns:

`Selected hot-update last-known-good job=<job_id> promotion=<promotion_id>.`

Failures return an empty direct response with the underlying error, matching existing direct command behavior.

## Failure Behavior

The command preserves V4-058 helper failures:

- Missing promotion rejects.
- Promotion without `outcome_id` rejects.
- Linked outcome missing rejects.
- Linked outcome not `hot_updated` rejects.
- Active pointer missing rejects.
- Active pointer mismatch rejects.
- Current LKG missing rejects.
- Current LKG not equal to previous active or promoted pack rejects.
- Divergent existing LKG rejects.
- Wrong `job_id` rejects through existing TaskState validation.

## Status Expectation

After successful recertification, `STATUS <job_id>` shows the recertified pointer through the existing read model:

- `runtime_pack_identity.last_known_good.state = configured`
- `runtime_pack_identity.last_known_good.pack_id = <promotion.promoted_pack_id>`
- `runtime_pack_identity.last_known_good.basis = hot_update_promotion:<promotion_id>`
- `runtime_pack_identity.last_known_good.verified_at = <stored verification timestamp>`

No new status section was added.

## Invariants Preserved

This slice does not mutate `active_pointer.json`, does not mutate `reload_generation`, does not create `HotUpdateOutcomeRecord`, does not create or mutate `PromotionRecord`, does not mutate `HotUpdateGateRecord`, does not add rollback behavior, does not add manual LKG fields, does not recertify directly from a gate, does not recertify directly from an outcome without a promotion, and does not start V4-061.
