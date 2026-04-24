# V4-059 Hot-Update LKG Recertification Control Entry Assessment

## Scope

V4-059 assesses the smallest safe operator/control entry for invoking the V4-058 missioncontrol helper:

`RecertifyLastKnownGoodFromPromotion(root, promotionID, verifiedBy, verifiedAt)`

This slice is docs-only. It does not change Go code, tests, commands, TaskState wrappers, pointers, ledgers, hot-update gates, rollback behavior, or policy.

## Existing Surfaces Inspected

The direct operator command path in `internal/agent/loop.go` already parses hot-update control commands with regexes and dispatches to `TaskState` wrappers. Recent examples are:

- `HOT_UPDATE_OUTCOME_CREATE <job_id> <hot_update_id>`
- `HOT_UPDATE_PROMOTION_CREATE <job_id> <outcome_id>`

The corresponding wrappers in `internal/agent/tools/taskstate.go` validate the active or persisted runtime control context, resolve the mission store root, derive timestamps with the existing TaskState timestamp helper, call the missioncontrol helper with actor `operator`, emit runtime control audit events, and return the missioncontrol `changed` flag unchanged.

V4-058 added `RecertifyLastKnownGoodFromPromotion(...)` in missioncontrol. The helper writes only `runtime_packs/last_known_good_pointer.json`, requires an existing committed promotion, requires a linked `hot_updated` outcome, requires active pointer equality with the promoted pack, and fails closed unless the current LKG is either the previous active pack or the exact deterministic recertified pointer.

The existing status/read-model surface already includes `runtime_pack_identity.last_known_good` with `state`, `pack_id`, `basis`, and `verified_at`. Promotion status also exposes `promotion_identity.promotions[]` with `promotion_id`, `promoted_pack_id`, `previous_active_pack_id`, `hot_update_id`, and `outcome_id`.

## Recommended V4-060 Command

Add one direct operator command:

`HOT_UPDATE_LKG_RECERTIFY <job_id> <promotion_id>`

Required arguments:

- `job_id`: must match the active execution context job or persisted runtime control job.
- `promotion_id`: identifies the existing committed `PromotionRecord` to use as the source authority.

Rejected/manual arguments:

- No manual pack id.
- No manual previous active pack id.
- No manual outcome id.
- No manual hot-update id.
- No manual basis.
- No manual rollback record ref.
- No manual verified_at.
- No manual verified_by.
- No manual reload_generation.
- No manual active pointer fields.

## TaskState Wrapper Shape

Add a small wrapper, likely:

`(*TaskState).RecertifyLastKnownGoodFromPromotion(jobID, promotionID string) (bool, error)`

The wrapper should follow the existing hot-update outcome/promotion control pattern:

- Validate mission store root with `missioncontrol.ValidateStoreRoot`.
- If an execution context is present, require `ec.Job`, `ec.Step`, and `ec.Runtime`, and require `ec.Job.ID == jobID`.
- If using persisted runtime state/control, require runtime state, require `runtimeState.JobID == jobID`, require persisted runtime control context, and require `control.JobID == jobID`.
- Derive `now` using `taskStateTransitionTimestamp(taskStateNowUTC())`.
- Call `missioncontrol.RecertifyLastKnownGoodFromPromotion(root, promotionID, "operator", now)`.
- Emit runtime control audit action `hot_update_lkg_recertify` on success and failure.
- Return the missioncontrol `changed` flag unchanged.

## Direct Response Semantics

Recommended direct command responses:

- `changed=true`: `Recertified hot-update last-known-good job=<job_id> promotion=<promotion_id>.`
- `changed=false`: `Selected hot-update last-known-good job=<job_id> promotion=<promotion_id>.`
- failure: empty response plus returned error, matching existing direct command behavior.

## Replay Semantics

Command replay should be idempotent. V4-058 makes helper replay deterministic only when the same `verified_at` is supplied, because `verified_at` is part of the deterministic LKG pointer.

For V4-060, the TaskState wrapper should mirror the V4-052/V4-056 replay pattern:

- First call uses the current TaskState timestamp.
- If the helper rejects because the current LKG already points to the promoted pack but differs from deterministic recertification, the wrapper may load `last_known_good_pointer.json`.
- If the loaded pointer has `pack_id = promotion.promoted_pack_id`, `basis = hot_update_promotion:<promotion_id>`, and `rollback_record_ref = hot_update_promotion:<promotion_id>`, retry the helper with the stored pointer's `verified_at`.
- Do not retry with arbitrary timestamps or for other failure modes.
- Do not hide fail-closed helper errors when the stored pointer does not match the deterministic basis/ref identity.

This preserves exact command replay without adding a manual timestamp argument.

## Failure Behavior To Preserve

V4-060 should preserve the V4-058 helper failure behavior and existing TaskState job validation:

- Missing promotion rejects.
- Promotion without `outcome_id` rejects.
- Linked outcome missing rejects.
- Linked outcome not `hot_updated` rejects.
- Active pointer missing rejects.
- Active pointer mismatch rejects.
- Current LKG missing rejects.
- Current LKG not equal to the promotion's previous active pack or promoted pack rejects.
- Divergent existing LKG rejects.
- Wrong `job_id` rejects through existing TaskState validation.

All failures should return an empty direct response and no additional mutation beyond audit event recording.

## Read-Only Status Expectations

After a successful V4-060 command, `STATUS <job_id>` should show the recertified pointer through existing read-model fields:

- `runtime_pack_identity.last_known_good.state = configured`
- `runtime_pack_identity.last_known_good.pack_id = <promotion.promoted_pack_id>`
- `runtime_pack_identity.last_known_good.basis = hot_update_promotion:<promotion_id>`
- `runtime_pack_identity.last_known_good.verified_at = <wrapper timestamp or replayed stored timestamp>`
- `promotion_identity.promotions[]` should continue to show the source promotion and outcome linkage.

No new status section is required for the smallest slice.

## Required V4-060 Tests

Direct command tests should prove:

- `HOT_UPDATE_LKG_RECERTIFY <job_id> <promotion_id>` recertifies LKG from a successful hot-update promotion.
- Exact command replay returns the selected/idempotent acknowledgement and leaves `last_known_good_pointer.json` byte-stable.
- Missing promotion rejects with empty response and no LKG mutation.
- Promotion without `outcome_id` rejects with empty response and no LKG mutation.
- Linked outcome missing rejects with empty response and no LKG mutation.
- Linked outcome not `hot_updated` rejects with empty response and no LKG mutation.
- Active pointer missing rejects with empty response and no LKG mutation.
- Active pointer mismatch rejects with empty response and no LKG mutation.
- Current LKG missing rejects with empty response.
- Current LKG not equal to previous active or promoted pack rejects with empty response.
- Divergent existing LKG rejects with empty response.
- Wrong `job_id` rejects through existing TaskState validation.
- Successful recertification does not mutate `active_pointer.json`.
- `reload_generation` is unchanged.
- Promotion bytes are unchanged.
- Hot-update outcome bytes are unchanged.
- Hot-update gate bytes are unchanged.
- No `HotUpdateOutcomeRecord` is created.
- No `PromotionRecord` is created or mutated.
- `STATUS <job_id>` shows the recertified LKG in `runtime_pack_identity.last_known_good`.

TaskState-focused tests should be added only if direct command tests do not fully lock:

- Active/persisted job validation.
- Audit event action name `hot_update_lkg_recertify`.
- Changed flag propagation.
- Replay timestamp reuse for exact idempotence.

## Non-Goals For V4-060

V4-060 should not add a public API beyond the direct command and TaskState wrapper. It should not mutate `active_pointer.json`, mutate `reload_generation`, create hot-update outcomes, create or mutate promotions, mutate hot-update gates, add rollback behavior, recertify directly from a gate, recertify directly from an outcome without a promotion, add manual field arguments, or broaden policy/authorization beyond existing TaskState direct command checks.

## Recommendation

The smallest safe V4-060 slice is a direct command plus TaskState wrapper over `RecertifyLastKnownGoodFromPromotion(...)`, with replay timestamp reuse from the already-written deterministic LKG pointer. This keeps authority on the committed promotion/outcome/gate chain, avoids manual operator-supplied data, and uses the existing `STATUS` runtime-pack identity read model for confirmation.
