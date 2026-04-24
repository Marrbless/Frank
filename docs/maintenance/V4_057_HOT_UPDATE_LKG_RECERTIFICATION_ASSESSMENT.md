# V4-057 Hot-Update Last-Known-Good Recertification Assessment

## Current Branch / HEAD / Tags

- Branch: `frank-v4-057-hot-update-lkg-recertification-assessment`
- HEAD: `285f0ff4cc965bc0ae38f870fbe31e4d91718899`
- Tags at HEAD:
  - `frank-v4-056-hot-update-promotion-create-control-entry`

## Repo Baseline

- `git status --short --branch --untracked-files=all` at slice start was clean:
  - `## frank-v4-057-hot-update-lkg-recertification-assessment`
- Baseline `/usr/local/go/bin/go test -count=1 ./...` passed when run with normal test permissions.
- The first sandboxed baseline attempt failed because the Go build cache was read-only and `httptest` could not open local sockets.

## Scope

This is a docs-only assessment for the smallest safe future slice that can recertify a promoted hot-update runtime pack as last-known-good.

No Go code, tests, commands, TaskState wrappers, last-known-good pointer files, active runtime-pack pointers, `reload_generation`, hot-update outcomes, promotions, hot-update gates, rollback behavior, policy/authorization rules, or V4-058 work are changed in V4-057.

## Existing Runtime Pointer Contracts

Runtime pack state is split across two pointer records:

- `active_pointer.json` stores `active_pack_id`, optional `previous_active_pack_id`, optional `last_known_good_pack_id`, `updated_at`, `updated_by`, `update_record_ref`, and `reload_generation`.
- `last_known_good_pointer.json` stores `pack_id`, `basis`, `verified_at`, `verified_by`, and `rollback_record_ref`.

`StoreActiveRuntimePackPointer` validates:

- active pack exists
- previous active pack exists when present
- active-pointer `last_known_good_pack_id` exists when present
- updated timestamp, actor, and update record ref are present

`StoreLastKnownGoodRuntimePackPointer` validates:

- LKG pack exists
- basis is non-empty
- verified timestamp is non-zero
- verified actor is non-empty
- rollback record ref is non-empty

There is no append-only LKG recertification ledger today. The canonical mutable LKG target is `runtime_packs/last_known_good_pointer.json`.

## Active Pointer And Reload Generation Semantics

The active pointer is the workflow authority for which runtime pack is currently active. Hot-update pointer switch mutates `active_pointer.json` and increments `reload_generation`. Reload/apply and later ledger steps preserve that active pointer and do not increment `reload_generation`.

LKG recertification should not change active runtime selection. It should not increment `reload_generation`, because no reload/apply or active-pack switch is happening. The active pointer should be read as a guard, not used as a mutation target.

Recommended future rule:

- Require `active_pointer.active_pack_id == promotion.promoted_pack_id`.
- Fail closed when the active pointer is missing, invalid, or points at any pack other than the promoted pack.
- Do not mutate `active_pointer.json`.
- Do not mutate `reload_generation`.

This keeps recertification from blessing a pack that is no longer active.

## Existing Promotion Authority

`PromotionRecord` already links:

- `promotion_id`
- `promoted_pack_id`
- `previous_active_pack_id`
- `hot_update_id`
- optional `outcome_id`
- optional candidate/run/result refs
- promotion reason and timestamps

Existing promotion validation already checks:

- promoted pack exists
- previous active pack exists
- hot-update gate exists
- promoted pack matches gate candidate pack
- previous active pack matches gate previous active pack
- linked outcome, when present, exists and matches promotion hot-update linkage
- optional candidate/run/result refs pass existing linkage validation

V4-054 created deterministic promotions only from committed successful hot-update outcomes:

- source authority: committed `HotUpdateOutcomeRecord`
- accepted outcome kind: `hot_updated`
- promotion ID: `hot-update-promotion-<hot_update_id>`
- promoted pack: outcome candidate pack
- previous active pack: originating gate previous active pack
- reason: `hot update outcome promoted`

V4-056 exposed that helper through:

- `HOT_UPDATE_PROMOTION_CREATE <job_id> <outcome_id>`

That command creates the promotion only. It intentionally does not mutate LKG state.

## Source Authority For LKG Recertification

The future recertification source authority should be an existing committed `PromotionRecord`.

Required source checks:

- promotion exists
- promotion is valid under existing promotion validation and linkage checks
- promotion has `outcome_id`
- linked outcome exists
- linked outcome has `outcome_kind = hot_updated`
- promotion `promoted_pack_id` is non-empty and validated
- promotion `previous_active_pack_id` is non-empty and validated
- promotion `promoted_pack_id` matches the linked outcome candidate pack when present
- promotion `hot_update_id` matches the linked outcome hot-update ID

The helper must not recertify from a hot-update gate directly. The required chain is:

1. committed successful hot-update outcome
2. committed promotion derived from that outcome
3. explicit LKG recertification from the promotion

## Decision And Evidence Requirement

Outcome and promotion timestamps are necessary evidence, but they are not sufficient by themselves to mutate last-known-good state.

Recommended rule:

- LKG recertification requires an explicit recertification decision surface.
- Promotion creation must not automatically recertify LKG.
- Status/read-model observation must not automatically recertify LKG.
- A future helper may implement the deterministic mutation, but it must not be called automatically from promotion creation.

For the first implementation slice, the explicit decision can be represented by the caller invoking a dedicated missioncontrol helper with `verified_by` and `verified_at`. A later direct command can expose that helper to operators through the existing TaskState control path.

## Deterministic Mutation Target

The future mutation target should be only:

- `runtime_packs/last_known_good_pointer.json`

Recommended derived pointer:

- `pack_id = PromotionRecord.PromotedPackID`
- `basis = hot_update_promotion:<promotion_id>`
- `verified_at = helper input`
- `verified_by = helper input`
- `rollback_record_ref = hot_update_promotion:<promotion_id>`

The duplicated ref in `basis` and `rollback_record_ref` is intentional for the first slice because `LastKnownGoodRuntimePackPointer` already requires both fields and there is no dedicated recertification record yet.

Do not mutate:

- `active_pointer.json`
- `reload_generation`
- `PromotionRecord`
- `HotUpdateOutcomeRecord`
- `HotUpdateGateRecord`

## Current LKG Handling

Recommended future rule:

- If `last_known_good_pointer.json` already exactly points to the promoted pack with the deterministic basis/ref and same verification metadata, return idempotently.
- If it points to the promoted pack with the deterministic basis/ref but different verification metadata, treat it as already selected for command replay only when the control wrapper intentionally reuses the stored `verified_at`; otherwise helper-level divergent replay should fail closed.
- If it points to `promotion.previous_active_pack_id`, allow replacement only because the committed promotion explicitly proves the transition from previous active pack to promoted pack.
- If it points to any other pack, fail closed.
- If the LKG pointer is missing or invalid, fail closed in the first slice. Recertification is not bootstrap.

This keeps the first slice from silently overwriting unrelated LKG state.

## Replay And Idempotence

First creation should write `last_known_good_pointer.json` and return `changed=true`.

Exact replay with the same derived LKG pointer should return `changed=false` and leave bytes unchanged.

Divergent current LKG should fail closed unless it is the expected previous active pack from the promotion.

Active pointer mismatch should fail closed:

- if active pointer is missing or invalid
- if `active_pack_id != promotion.promoted_pack_id`

If the promotion, outcome, gate, or active pointer changes after recertification such that the derived or guarded state differs, the helper should fail closed rather than rewriting.

## Durable Ledger Record Decision

A dedicated recertification ledger record would be useful later for audit history, multi-stage evidence, and richer policy. It is not required for the smallest safe first slice.

For V4-058, the LKG pointer write itself is sufficient if:

- mutation is deterministic
- basis/ref names the promotion ID
- the helper is idempotent
- all source linkage is validated
- all non-target files are byte-stable in tests

If stronger audit is needed later, add a recertification ledger before broadening policy or automation.

## Read-Model Expectations

Existing status should surface the result through the runtime pack identity read model:

- `last_known_good.pack_id` should become the promoted pack.
- `last_known_good.basis` should show `hot_update_promotion:<promotion_id>`.
- `last_known_good.verified_at` should match the helper input or selected stored value on replay.
- `last_known_good.verified_by` should be the actor.

Promotion status should remain unchanged. The promotion record has optional LKG fields, but V4-058 should not mutate the promotion to fill them.

One read-model caveat remains: `active_pointer.json` also has an optional `last_known_good_pack_id`. If V4-058 mutates only `last_known_good_pointer.json`, status surfaces that read both records may temporarily show active-pointer metadata that still names the previous LKG. That should be treated as a read-model polish issue, not a reason to mutate active pointer in the first recertification slice.

## Recommended V4-058 Implementation Slice

The smallest safe V4-058 slice is a missioncontrol-only helper, for example:

- `RecertifyLastKnownGoodFromPromotion(root, promotionID, verifiedBy string, verifiedAt time.Time) (LastKnownGoodRuntimePackPointer, bool, error)`

Recommended behavior:

- load and validate the promotion by `promotion_id`
- require linked successful hot-update outcome
- validate promotion/outcome/gate linkage through existing loaders
- load active pointer and require `active_pack_id == promotion.promoted_pack_id`
- load current LKG pointer
- allow replacement only when current LKG pack is `promotion.previous_active_pack_id`
- allow exact replay when current LKG already equals the deterministic pointer
- write only `last_known_good_pointer.json`
- return `changed=true` on first replacement and `changed=false` on exact replay

Recommended V4-058 tests:

- successful promotion recertifies promoted pack as LKG
- failed or missing linked outcome rejects
- promotion missing outcome rejects
- active pointer mismatch rejects
- missing active pointer rejects
- missing current LKG rejects
- current LKG not equal to previous active or promoted pack rejects
- exact replay is idempotent and byte-stable
- divergent existing LKG fails closed
- active pointer bytes are unchanged
- `reload_generation` is unchanged
- promotion bytes are unchanged
- hot-update outcome bytes are unchanged
- hot-update gate bytes are unchanged
- no new outcome or promotion is created

Do not add a direct command in V4-058 unless the implementation decision intentionally combines helper and operator control. If the helper-only pattern continues, a later control-entry slice should expose an operator command such as:

- `HOT_UPDATE_LKG_RECERTIFY <job_id> <promotion_id>`

## Non-Goals For V4-058

- no active runtime-pack pointer mutation
- no `reload_generation` mutation
- no hot-update outcome creation
- no promotion creation
- no promotion mutation
- no hot-update gate mutation
- no rollback behavior
- no automatic recertification from promotion creation
- no status-triggered recertification
- no broad policy/authorization change
