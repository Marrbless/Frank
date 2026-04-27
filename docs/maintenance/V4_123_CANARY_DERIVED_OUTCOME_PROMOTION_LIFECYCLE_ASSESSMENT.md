# V4-123 - Canary-Derived Outcome/Promotion Lifecycle Assessment

## Scope

V4-123 assesses whether canary-derived gates that pass the V4-122 execution readiness guard can safely use the existing generic outcome, promotion, rollback, rollback-apply, and last-known-good paths, or whether downstream records need additional canary audit lineage.

This slice is docs-only. It does not change Go code, tests, commands, TaskState wrappers, outcome records, promotion records, rollback records, rollback-apply records, runtime pointers, last-known-good pointers, canary authorities, owner approval decisions, source records, `CandidatePromotionDecisionRecord`, `CreateHotUpdateGateFromCandidatePromotionDecision(...)`, or V4-124 implementation.

## Live State Inspected

Starting state:

- branch: `frank-v4-123-canary-derived-outcome-promotion-lifecycle-assessment`
- HEAD: `820bae3d691e792120e29a34d62cfd7c1f5c11c0`
- tag at HEAD: `frank-v4-122-canary-derived-gate-execution-readiness-guard`
- worktree: clean before this memo
- baseline validator: `/usr/local/go/bin/go test -count=1 ./...`

Docs inspected:

- `docs/FRANK_V4_SPEC.md`
- `docs/maintenance/V4_120_CANARY_GATE_LIFECYCLE_CHECKPOINT.md`
- `docs/maintenance/V4_121_CANARY_DERIVED_GATE_EXECUTION_READINESS_ASSESSMENT.md`
- `docs/maintenance/V4_122_CANARY_DERIVED_GATE_EXECUTION_READINESS_GUARD_AFTER.md`

Code paths inspected:

- `HotUpdateGateRecord`
- `AssessHotUpdateCanaryGateExecutionReadiness(...)`
- `CreateHotUpdateOutcomeFromTerminalGate(...)`
- `HotUpdateOutcomeRecord`
- direct command `HOT_UPDATE_OUTCOME_CREATE`
- `TaskState.CreateHotUpdateOutcomeFromTerminalGate(...)`
- `CreatePromotionFromSuccessfulHotUpdateOutcome(...)`
- `PromotionRecord`
- direct command `HOT_UPDATE_PROMOTION_CREATE`
- `TaskState.CreatePromotionFromSuccessfulHotUpdateOutcome(...)`
- rollback registry
- rollback-apply registry
- `RecertifyLastKnownGoodFromPromotion(...)`
- direct command `HOT_UPDATE_LKG_RECERTIFY`
- status/read-model fields for hot-update gate, outcome, promotion, rollback, rollback-apply, and LKG

## V4-122 Guard Baseline

V4-122 added `AssessHotUpdateCanaryGateExecutionReadiness(root, hotUpdateID)` and calls it from:

- phase advancement to `validated` or `staged`,
- pointer switch before active runtime-pack pointer mutation,
- reload/apply before convergence writes.

The guard is canary-specific only when `HotUpdateGateRecord.canary_ref` is non-empty. It revalidates the canary satisfaction authority, selected evidence, fresh canary satisfaction, fresh promotion eligibility, owner approval decision when required, candidate pack, rollback target, active pointer, and present LKG pointer. Normal eligible-only gates remain generic.

This means a canary-derived gate can only reach `reload_apply_succeeded` through the governed V4-122 missioncontrol/direct-command path after passing the canary authority guard at the lifecycle points that precede runtime effects.

## Current Outcome Behavior

`HotUpdateOutcomeRecord` currently stores:

- `outcome_id`
- `hot_update_id`
- optional candidate/run/result refs
- `candidate_pack_id`
- `outcome_kind`
- reason, notes, outcome time, created time, and creator

It does not store `canary_ref` or `approval_ref`.

`CreateHotUpdateOutcomeFromTerminalGate(...)` loads the terminal gate, validates derived gate linkage, and accepts only:

- `reload_apply_succeeded`, producing `outcome_kind=hot_updated`
- `reload_apply_failed`, producing `outcome_kind=failed`

It does not call `AssessHotUpdateCanaryGateExecutionReadiness(...)` and does not branch on `gate.canary_ref`.

`HOT_UPDATE_OUTCOME_CREATE` reaches the helper through `TaskState.CreateHotUpdateOutcomeFromTerminalGate(...)`. TaskState validates job context and reuses the existing outcome creation time on replay. It does not add canary-specific checks.

## Outcome Assessment

Generic outcome creation can safely consume terminal canary-derived gates after V4-122.

Outcome creation is an accounting step for a terminal execution state. It should not be blocked merely because a canary source record drifts after pointer switch or reload/apply. Blocking it would risk losing the durable result of an already attempted hot update, which conflicts with the V4 requirement that every attempt has an explicit outcome.

Outcome creation should not re-run `AssessHotUpdateCanaryGateExecutionReadiness(...)`. The guard is appropriate before phase, pointer switch, and reload/apply because those steps either make a gate executable or perform execution effects. After a gate is terminal, the important invariant is recording what happened, not re-proving that source canary authority still looks fresh.

Successful canary-derived outcomes should remain `outcome_kind=hot_updated` for now. `HotUpdateOutcomeKindCanaryApplied` exists, but the live path does not use it for terminal hot-update success, and switching canary-derived success to `canary_applied` would currently break generic promotion creation because `CreatePromotionFromSuccessfulHotUpdateOutcome(...)` only accepts `hot_updated`. The enum alone is not enough reason to introduce a canary-specific success kind.

Failed canary-derived outcomes should remain `outcome_kind=failed`. Failure is a terminal hot-update outcome independent of whether the gate was canary-derived.

## Current Promotion Behavior

`PromotionRecord` currently stores:

- `promotion_id`
- `promoted_pack_id`
- `previous_active_pack_id`
- optional LKG fields
- `hot_update_id`
- `outcome_id`
- optional candidate/run/result refs
- reason, notes, promoted time, created time, and creator

It does not store `canary_ref` or `approval_ref`.

`CreatePromotionFromSuccessfulHotUpdateOutcome(...)` loads a successful outcome, requires `outcome_kind=hot_updated`, loads the linked gate, verifies candidate pack and previous-active linkage, builds a deterministic promotion ID from the hot update ID, and stores a generic promotion. It does not call `AssessHotUpdateCanaryGateExecutionReadiness(...)`.

`HOT_UPDATE_PROMOTION_CREATE` reaches the helper through `TaskState.CreatePromotionFromSuccessfulHotUpdateOutcome(...)`. TaskState validates job context and replay timestamps. It does not add canary-specific checks.

## Promotion Assessment

Generic promotion creation can safely consume successful canary-derived outcomes after V4-122, provided promotion remains a downstream record of a successful terminal hot-update outcome rather than a second authority gate.

Promotion creation should not re-run canary gate readiness. If source canary evidence, authority, or owner approval records drift after a successful terminal outcome, promotion creation should still be able to preserve the durable promotion record for the pack that was already activated successfully. Revalidating canary authority at promotion time would create the same bad failure mode as outcome revalidation: it could prevent recording downstream truth after irreversible execution.

The current generic linkage checks are still relevant and should remain:

- outcome kind must be `hot_updated`;
- outcome hot update ID must match the gate;
- outcome candidate pack must match the gate and promotion;
- promoted pack must match the gate candidate pack;
- previous active pack must match the gate previous active pack;
- optional candidate/run/result refs must match linked records when present.

## Audit Lineage Gap

The main downstream gap is audit parity, not execution authority.

`HotUpdateGateRecord` already carries `canary_ref` and `approval_ref`, and `hot_update_gate_identity` exposes those fields. However, `HotUpdateOutcomeRecord`, `PromotionRecord`, `hot_update_outcome_identity`, and `promotion_identity` do not copy or expose those refs.

That means a reviewer can recover canary lineage by joining outcome or promotion back to the gate through `hot_update_id`, but the downstream records themselves do not carry immutable canary/owner-approval lineage. If a status surface or exported record stream is viewed without the gate record, the canary/owner approval authority is easy to lose.

The V4 spec lists `approval_ref` and `canary_ref` among minimum hot-update and promotion record fields. The live gate record satisfies that for the hot-update gate, but promotion records do not yet satisfy direct audit parity.

## Rollback And Rollback-Apply Assessment

Rollback and rollback-apply should remain generic.

Rollback records derive from promotions and record the packs to restore. Rollback linkage validates promotion, gate, outcome, from-pack, target-pack, and optional LKG consistency. Rollback-apply then switches the pointer back to the target pack and records reload/apply convergence through its own workflow.

These recovery paths should not be blocked by stale canary evidence, stale canary authority, or owner approval drift. Once a canary-derived update has become active or partially active, rollback authority should be based on safe restoration of previous active or last-known-good packs. Canary drift may explain why an update should not have executed, but it must not prevent recovery.

No canary-specific rollback or rollback-apply policy is recommended for V4-124.

## LKG Assessment

`RecertifyLastKnownGoodFromPromotion(...)` consumes a promotion and linked successful outcome, requires `outcome_kind=hot_updated`, verifies the active pointer is on the promoted pack, and then updates the LKG pointer with a deterministic `hot_update_promotion:<promotion_id>` basis.

LKG recertification does not need a canary-specific authority recheck. Its safety question is whether the promoted pack is active and linked to a successful hot-update outcome. Re-running canary readiness at LKG time would risk blocking recertification because source evidence drifted after a successful apply, which is not the right recovery or audit behavior.

Additional canary audit fields on LKG are not recommended as the next slice. LKG points to the promoted pack and promotion basis; if promotion carries canary lineage, LKG can inherit audit lineage by following `rollback_record_ref` / basis to the promotion.

## Status And Read Models

Current status surfaces:

- `hot_update_gate_identity` includes `canary_ref` and `approval_ref`.
- `hot_update_outcome_identity` does not include `canary_ref` or `approval_ref`.
- `promotion_identity` does not include `canary_ref` or `approval_ref`.
- `rollback_identity` does not include canary or approval refs.
- `rollback_apply_identity` does not include canary or approval refs.
- runtime pack identity exposes active and LKG pointer state, not canary lineage.

The smallest read-model improvement is to expose copied canary/approval refs on outcome and promotion statuses after those records gain fields. Rollback, rollback-apply, and LKG can remain generic and derive lineage by following promotion/outcome/gate refs.

## Recommendation

V4-124 should be a small missioncontrol registry/read-model implementation slice, not another docs-only assessment.

Recommended V4-124:

```text
V4-124 - Canary-Derived Outcome/Promotion Audit Lineage Propagation
```

Smallest safe scope:

1. Add `CanaryRef` and `ApprovalRef` to `HotUpdateOutcomeRecord`.
2. Copy those fields from the linked `HotUpdateGateRecord` in `CreateHotUpdateOutcomeFromTerminalGate(...)`.
3. Add `CanaryRef` and `ApprovalRef` to `PromotionRecord`.
4. Copy those fields from the linked outcome, or from the gate with a consistency check if outcome fields are present.
5. Add validation/linkage checks so promotion can fail closed when outcome/gate copied refs disagree.
6. Expose `canary_ref` and `approval_ref` in `OperatorHotUpdateOutcomeStatus` and `OperatorPromotionStatus`.
7. Add focused tests for canary-derived and non-canary records, replay stability, status/read-model output, and no changes to rollback, rollback-apply, LKG, commands, or TaskState wrappers.

V4-124 should not re-run canary readiness from outcome, promotion, rollback, rollback-apply, or LKG creation. It should only preserve immutable lineage from the gate into downstream audit records.

## Explicit Answers

After V4-122, generic outcome creation can safely consume terminal canary-derived gates.

`HotUpdateOutcomeRecord` should copy `canary_ref` and `approval_ref` from the gate for audit parity.

Outcome creation should not re-run `AssessHotUpdateCanaryGateExecutionReadiness(...)`; terminal gate state after guarded execution is enough to record outcome.

Successful canary-derived outcomes should remain `outcome_kind=hot_updated`; `canary_applied` exists but is not necessary and would complicate generic promotion.

Failed canary-derived outcomes should remain `outcome_kind=failed`.

Generic promotion creation can safely consume successful canary-derived outcomes.

`PromotionRecord` should copy `canary_ref` and `approval_ref` from the outcome/gate for audit parity.

Promotion creation should not re-run canary gate readiness; it should rely on successful terminal outcome and linkage.

Rollback and rollback-apply do not need canary-specific changes.

LKG recertification does not need canary-specific policy or direct canary audit fields in the next slice.

V4-124 should be a small audit-field propagation implementation slice in missioncontrol registries and read models.

## Explicit Non-Goals

V4-123 does not:

- change Go code or tests
- add commands
- add TaskState wrappers
- create outcomes
- create promotions
- create rollbacks
- create rollback-apply records
- mutate active runtime-pack pointer
- mutate last-known-good pointer
- mutate `reload_generation`
- mutate canary satisfaction authority
- mutate owner approval decision
- mutate source records
- broaden `CandidatePromotionDecisionRecord`
- create candidate promotion decisions for canary-required states
- change `CreateHotUpdateGateFromCandidatePromotionDecision(...)`
- implement V4-124
