# V4-053 Hot-Update Promotion Creation Assessment

## Current Branch / HEAD / Tags

- Branch: `frank-v4-053-hot-update-promotion-creation-assessment`
- HEAD: `6b770689f2d6fb83de91e88439814df338f7e697`
- Tags at HEAD:
  - `frank-v4-052-hot-update-outcome-create-control-entry`

## Repo Baseline

- `git status --short --branch --untracked-files=all` at slice start was clean:
  - `## frank-v4-053-hot-update-promotion-creation-assessment`
- Baseline `/usr/local/go/bin/go test -count=1 ./...` passed.

## Scope

This is a docs-only assessment for the smallest safe future implementation slice that creates a `PromotionRecord` from an existing successful `HotUpdateOutcomeRecord`.

No Go code, tests, commands, TaskState wrappers, promotion records, outcome records, active runtime-pack pointer state, `reload_generation`, last-known-good pointer, hot-update gates, or V4-054 work are changed in V4-053.

## Existing PromotionRecord Contract

`PromotionRecord` already exists as an append-only ledger record with these fields:

- `promotion_id`
- `promoted_pack_id`
- `previous_active_pack_id`
- optional `last_known_good_pack_id`
- optional `last_known_good_basis`
- `hot_update_id`
- optional `outcome_id`
- optional `candidate_id`
- optional `run_id`
- optional `candidate_result_id`
- `reason`
- optional `notes`
- `promoted_at`
- `created_at`
- `created_by`

Existing storage behavior:

- `StorePromotionRecord` normalizes records.
- `StorePromotionRecord` validates required fields.
- Exact duplicate replay is idempotent when the stored record deeply equals the proposed record.
- Divergent duplicate with the same `promotion_id` fails closed.
- Listing is deterministic through the existing store JSON list path.
- Read models already expose promotion identity through `promotion_identity`.

Existing linkage validation:

- `promoted_pack_id` must reference an existing runtime pack.
- `previous_active_pack_id` must reference an existing runtime pack.
- `last_known_good_pack_id`, when present, must reference an existing runtime pack and requires `last_known_good_basis`.
- `hot_update_id` must reference an existing `HotUpdateGateRecord`.
- `promoted_pack_id` must match the originating gate `candidate_pack_id`.
- `previous_active_pack_id` must match the originating gate `previous_active_pack_id`.
- `outcome_id`, when present, must reference an existing `HotUpdateOutcomeRecord`.
- The linked outcome must match the promotion `hot_update_id`.
- The linked outcome `candidate_pack_id`, when present, must match `promoted_pack_id`.
- Optional candidate/run/result refs, when present, are validated against existing candidate/run/result records and the gate/outcome linkage.

The existing store does not decide whether an outcome is eligible for promotion. That must be enforced by the future creation helper.

## Existing HotUpdateOutcomeRecord Contract

`HotUpdateOutcomeRecord` already exists as an append-only outcome ledger with:

- deterministic storage under `runtime_packs/hot_update_outcomes`
- `outcome_id`
- `hot_update_id`
- optional candidate/run/result refs
- optional `candidate_pack_id`
- `outcome_kind`
- optional `reason`
- `outcome_at`
- `created_at`
- `created_by`

V4-050 added deterministic outcome creation from committed terminal gates:

- `reload_apply_succeeded` creates `outcome_kind = hot_updated`
- `reload_apply_failed` creates `outcome_kind = failed`
- deterministic outcome ID is `hot-update-outcome-<hot_update_id>`
- `candidate_pack_id` is copied from the gate
- `outcome_at` is copied from the gate `phase_updated_at`
- candidate/run/result refs remain empty unless already provided through some other deterministic linkage

V4-052 exposed that helper through:

- `HOT_UPDATE_OUTCOME_CREATE <job_id> <hot_update_id>`

The direct control entry creates the outcome only. It intentionally does not create promotions, mutate pointers, update last-known-good state, or infer success/failure outside the committed gate and outcome helper.

## Source Authority For Promotion Creation

The future promotion creation helper should use the existing committed `HotUpdateOutcomeRecord` as its source authority.

Recommended helper input:

- `root`
- `outcome_id`
- `created_by`
- `created_at`

The helper should load the outcome by `outcome_id`, validate outcome linkage, and only then load the originating `HotUpdateGateRecord` through `outcome.HotUpdateID`.

The helper must not create a promotion directly from a successful gate without an existing outcome. The outcome ledger is now the explicit checkpoint between reload/apply completion and promotion ledger creation.

## Eligible Outcomes

The smallest safe V4-054 implementation should accept only:

- `outcome_kind = hot_updated`

The helper should reject:

- `failed`
- `kept_staged`
- `discarded`
- `blocked`
- `approval_required`
- `cold_restart_required`
- `canary_applied`
- `promoted`
- `rolled_back`
- `aborted`
- any unknown or future outcome kind not explicitly selected

Failed hot-update outcomes must reject. A failed reload/apply or operator terminal-failure outcome is a durable terminal result, not promotion authority.

## Deterministic Promotion Identity

Recommended deterministic promotion ID:

- `hot-update-promotion-<hot_update_id>`

Rationale:

- It mirrors the V4-050 outcome identity format.
- It is scoped to the hot-update lane.
- A single successful hot-update outcome should produce at most one promotion ledger entry.
- It avoids accepting caller-provided arbitrary promotion IDs in the first helper.

Alternative:

- `promotion-<hot_update_id>`

The longer `hot-update-promotion-<hot_update_id>` is more explicit and less likely to collide with older generic promotion IDs.

## Required Derived Linkage

The future helper should derive:

- `promotion_id`: `hot-update-promotion-<hot_update_id>`
- `promoted_pack_id`: `HotUpdateOutcomeRecord.CandidatePackID`
- `previous_active_pack_id`: `HotUpdateGateRecord.PreviousActivePackID`
- `hot_update_id`: `HotUpdateOutcomeRecord.HotUpdateID`
- `outcome_id`: `HotUpdateOutcomeRecord.OutcomeID`
- `promoted_at`: `HotUpdateOutcomeRecord.OutcomeAt`
- `created_at`: helper input
- `created_by`: helper input
- `reason`: deterministic fixed reason, recommended `hot update outcome promoted`

Optional refs should be copied only when already present on the outcome:

- `candidate_id`
- `run_id`
- `candidate_result_id`

Because V4-050 outcomes leave those refs empty unless deterministically known, V4-054 should not invent them.

`last_known_good_pack_id` and `last_known_good_basis` should remain empty in V4-054 unless the slice explicitly expands into recertification. Promotion ledger creation is not the same as last-known-good mutation.

## Required Fail-Closed Checks

The helper should fail closed when:

- the outcome is missing
- the outcome linkage is invalid
- the originating gate is missing
- `outcome_kind` is not `hot_updated`
- `candidate_pack_id` is empty
- the outcome candidate pack does not match the gate candidate pack
- the gate previous active pack is empty or missing
- an existing promotion with the deterministic `promotion_id` diverges from the derived record
- any existing promotion for the same `hot_update_id` has a different `promotion_id`
- any existing promotion for the same `outcome_id` has a different `promotion_id`
- optional candidate/run/result refs are present but fail existing promotion linkage validation

The helper should not infer promotion eligibility from active pointer state, status output, or the absence of failure records.

## Replay And Idempotence

First creation should write the promotion and return `changed=true`.

Exact replay of the same derived promotion should return `changed=false` and leave the existing promotion file unchanged.

Divergent duplicate with the same deterministic `promotion_id` should fail closed through the existing append-only store behavior.

Existing promotion for the same `hot_update_id` or `outcome_id` under a different `promotion_id` should fail closed before writing. This prevents duplicate promotion ledgers for the same successful hot-update outcome.

If the linked outcome or gate changes after promotion creation such that the derived promotion would differ, the helper should fail closed rather than rewriting the promotion.

## Read-Model Expectations

Existing status/read-model surfaces should be enough for V4-054:

- `promotion_identity` already exposes promotion records.
- `hot_update_outcome_identity` already exposes the source outcome.
- `hot_update_gate_identity` already exposes the originating gate.

After helper success, status should show the created promotion in `promotion_identity` with:

- deterministic `promotion_id`
- promoted pack
- previous active pack
- hot-update ID
- outcome ID
- optional candidate/run/result refs when present
- reason
- promoted/created timestamps
- created actor

No separate status command is needed unless implementation proves the existing read model is insufficient.

## Non-Goals For V4-054

- no direct operator command
- no TaskState wrapper
- no active runtime-pack pointer mutation
- no `reload_generation` mutation
- no `last_known_good_pointer.json` mutation
- no last-known-good recertification
- no new outcome creation
- no hot-update gate mutation
- no automatic promotion directly from gate success without an outcome
- no terminal-state inference outside the committed outcome
- no policy or authorization broadening
- no manual promotion IDs
- no manual promoted pack refs
- no manual previous active pack refs
- no manual candidate/run/result refs

## Recommended V4-054 Implementation Slice

Add a missioncontrol helper, likely:

- `CreatePromotionFromSuccessfulHotUpdateOutcome(root, outcomeID, createdBy string, createdAt time.Time) (PromotionRecord, bool, error)`

The helper should:

- load the committed outcome by `outcome_id`
- require `outcome_kind = hot_updated`
- load the originating gate through `outcome.HotUpdateID`
- derive `promotion_id = hot-update-promotion-<hot_update_id>`
- derive pack linkage from outcome and gate
- copy optional candidate/run/result refs from the outcome only when present
- leave last-known-good fields empty
- use `promoted_at = outcome.OutcomeAt`
- use helper input for `created_at` and `created_by`
- check for existing promotions by deterministic ID, hot-update ID, and outcome ID before writing
- call `StorePromotionRecord`
- return `changed=true` for first write and `changed=false` for exact replay

Focused tests should prove:

- successful `hot_updated` outcome creates a promotion
- failed outcome rejects
- non-`hot_updated` outcomes reject
- missing outcome rejects
- exact replay is idempotent and does not rewrite
- divergent deterministic duplicate fails closed
- existing promotion for same `hot_update_id` under a different ID fails closed
- existing promotion for same `outcome_id` under a different ID fails closed
- optional refs are copied only when already present
- no active pointer mutation
- no `reload_generation` mutation
- no LKG mutation
- no outcome mutation
- no gate mutation
- no new outcome or gate record creation

This keeps V4-054 as a ledger-only missioncontrol implementation slice, with operator control deferred to a later explicit slice if needed.
