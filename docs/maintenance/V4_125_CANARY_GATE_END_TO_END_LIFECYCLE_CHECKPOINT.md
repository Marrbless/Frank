# V4-125 - Canary Gate End-to-End Lifecycle Checkpoint

## Scope

V4-125 is a docs-only checkpoint after V4-124. It summarizes the completed canary-required hot-update path from candidate result through promotion audit lineage, documents the no-owner and owner-approved branches, records what remains intentionally generic, and recommends one next slice.

This slice does not change Go code, tests, commands, TaskState wrappers, records, runtime pointers, last-known-good pointers, `reload_generation`, outcome kinds, rollback behavior, rollback-apply behavior, natural-language owner approval behavior, `CandidatePromotionDecisionRecord`, `CreateHotUpdateGateFromCandidatePromotionDecision(...)`, or V4-126 implementation.

## Live State Inspected

Starting state:

- branch: `frank-v4-125-canary-gate-end-to-end-lifecycle-checkpoint`
- HEAD: `3ed75c54252b5f27b05bec86073ed2db33171d3d`
- tag at HEAD: `frank-v4-124-canary-derived-outcome-promotion-audit-lineage`
- worktree: clean before this memo
- baseline validator: `/usr/local/go/bin/go test -count=1 ./...`

Docs inspected:

- `docs/FRANK_V4_SPEC.md`
- `docs/maintenance/V4_120_CANARY_GATE_LIFECYCLE_CHECKPOINT.md`
- `docs/maintenance/V4_121_CANARY_DERIVED_GATE_EXECUTION_READINESS_ASSESSMENT.md`
- `docs/maintenance/V4_122_CANARY_DERIVED_GATE_EXECUTION_READINESS_GUARD_AFTER.md`
- `docs/maintenance/V4_123_CANARY_DERIVED_OUTCOME_PROMOTION_LIFECYCLE_ASSESSMENT.md`
- `docs/maintenance/V4_124_CANARY_DERIVED_OUTCOME_PROMOTION_AUDIT_LINEAGE_AFTER.md`

Code paths inspected:

- canary requirement, evidence, satisfaction, satisfaction authority, owner approval request, and owner approval decision registries
- canary gate creation helper and direct command
- canary-derived gate execution readiness helper
- hot-update phase, pointer-switch, reload/apply, outcome, promotion, rollback, rollback-apply, and LKG recertification paths
- TaskState wrappers and direct-command routing
- status/read-model surfaces for canary, owner approval, gate, outcome, promotion, rollback, rollback-apply, and LKG identity
- `CandidatePromotionDecisionRecord` and `CreateHotUpdateGateFromCandidatePromotionDecision(...)`

## Completed Slices

| Slice | Result |
| --- | --- |
| V4-095 | Added `HotUpdateCanaryRequirementRecord` for candidate results whose fresh eligibility is `canary_required` or `canary_and_owner_approval_required`. |
| V4-096 | Assessed the requirement command surface. |
| V4-097 | Added `HOT_UPDATE_CANARY_REQUIREMENT_CREATE <job_id> <result_id>`. |
| V4-098 | Assessed durable canary evidence. |
| V4-099 | Added `HotUpdateCanaryEvidenceRecord` with `passed`, `failed`, `blocked`, and `expired` states. |
| V4-100 | Assessed the evidence command surface. |
| V4-101 | Added `HOT_UPDATE_CANARY_EVIDENCE_CREATE <job_id> <canary_requirement_id> <evidence_state> <observed_at> [reason...]`. |
| V4-102 | Assessed the canary-passed promotion gate path. |
| V4-103 | Added read-only `AssessHotUpdateCanarySatisfaction(...)`. |
| V4-104 | Assessed durable canary satisfaction authority. |
| V4-105 | Added `HotUpdateCanarySatisfactionAuthorityRecord`. |
| V4-106 | Assessed the canary satisfaction authority command surface. |
| V4-107 | Added `HOT_UPDATE_CANARY_SATISFACTION_AUTHORITY_CREATE <job_id> <canary_requirement_id>`. |
| V4-108 | Assessed owner approval authority. |
| V4-109 | Added `HotUpdateOwnerApprovalRequestRecord`. |
| V4-110 | Assessed the owner approval request command surface. |
| V4-111 | Added `HOT_UPDATE_OWNER_APPROVAL_REQUEST_CREATE <job_id> <canary_satisfaction_authority_id>`. |
| V4-112 | Assessed owner approval grant/rejection authority. |
| V4-113 | Added `HotUpdateOwnerApprovalDecisionRecord` with terminal `granted` and `rejected` decisions. |
| V4-114 | Assessed the owner approval decision command surface. |
| V4-115 | Added `HOT_UPDATE_OWNER_APPROVAL_DECISION_CREATE <job_id> <owner_approval_request_id> <decision> [reason...]`. |
| V4-116 | Assessed canary owner approval to gate authority. |
| V4-117 | Added `CreateHotUpdateGateFromCanarySatisfactionAuthority(...)` and deterministic canary gate IDs. |
| V4-118 | Assessed the canary gate control entry. |
| V4-119 | Added `HOT_UPDATE_CANARY_GATE_CREATE <job_id> <canary_satisfaction_authority_id> [owner_approval_decision_id]`. |
| V4-120 | Checkpointed prepared canary gate lifecycle and identified downstream execution readiness gaps. |
| V4-121 | Assessed canary-derived gate execution readiness and recommended a shared guard. |
| V4-122 | Added `AssessHotUpdateCanaryGateExecutionReadiness(...)` and wired it into phase, pointer-switch, and reload/apply for canary-derived gates. |
| V4-123 | Assessed outcome, promotion, rollback, rollback-apply, and LKG behavior after guarded execution. |
| V4-124 | Added copied `canary_ref` and `approval_ref` audit lineage to outcomes, promotions, and their read models. |

## Current Authority Chain

The completed authority chain is:

```text
candidate result + promotion policy
-> canary requirement
-> selected canary evidence
-> read-only canary satisfaction assessment
-> canary satisfaction authority
-> optional owner approval request
-> optional owner approval decision
-> canary-derived prepared hot-update gate
-> canary-derived execution readiness guard at phase/execute/reload
-> terminal hot-update outcome with copied lineage
-> promotion with copied lineage
```

The chain is split deliberately from the eligible-only path. `CandidatePromotionDecisionRecord` remains strictly `eligibility_state=eligible`, and `CreateHotUpdateGateFromCandidatePromotionDecision(...)` remains unchanged.

## No-Owner-Approval Branch

The no-owner branch is now complete through promotion audit lineage:

```text
candidate result + policy
-> canary requirement
-> passed canary evidence
-> canary satisfaction=satisfied
-> canary satisfaction authority=authorized
-> prepared canary gate with canary_ref and empty approval_ref
-> guarded phase/execute/reload
-> outcome with copied canary_ref
-> promotion with copied canary_ref
```

The gate helper requires the caller to omit `owner_approval_decision_id` for this branch. Supplying an owner approval decision for an `authorized` no-owner authority fails closed.

After V4-124, successful terminal outcomes remain `outcome_kind=hot_updated`, failed terminal outcomes remain `outcome_kind=failed`, and the copied outcome/promotion lineage carries `canary_ref` with empty `approval_ref`.

## Owner-Approval Branch

The owner-approved branch is now complete through promotion audit lineage:

```text
candidate result + policy
-> canary requirement
-> passed canary evidence
-> canary satisfaction=waiting_owner_approval
-> canary satisfaction authority=waiting_owner_approval
-> owner approval request=requested
-> owner approval decision=granted
-> prepared canary gate with canary_ref and approval_ref
-> guarded phase/execute/reload
-> outcome with copied canary_ref and approval_ref
-> promotion with copied canary_ref and approval_ref
```

The canary gate helper requires a committed owner approval decision whose copied refs match the authority and whose `decision` is exactly `granted`.

Owner approval does not replace passed canary evidence. The branch still requires passed selected evidence and a fresh satisfaction state of `waiting_owner_approval`.

## Rejected Owner Approval

```text
owner approval decision=rejected
```

is durable terminal authority for the owner approval request, but it is a terminal blocker for canary gate creation and never authorizes execution.

The gate helper rejects rejected decisions at creation time. The V4-122 canary execution readiness guard also rejects rejected owner approval decisions if a canary-derived gate's source authority is later made stale or mismatched. Natural-language approval aliases remain unbound; only exact `granted` and `rejected` command tokens are accepted.

## Current Command Chain

The canary-required command chain is:

```text
HOT_UPDATE_CANARY_REQUIREMENT_CREATE <job_id> <result_id>
HOT_UPDATE_CANARY_EVIDENCE_CREATE <job_id> <canary_requirement_id> <evidence_state> <observed_at> [reason...]
HOT_UPDATE_CANARY_SATISFACTION_AUTHORITY_CREATE <job_id> <canary_requirement_id>
HOT_UPDATE_OWNER_APPROVAL_REQUEST_CREATE <job_id> <canary_satisfaction_authority_id>
HOT_UPDATE_OWNER_APPROVAL_DECISION_CREATE <job_id> <owner_approval_request_id> <decision> [reason...]
HOT_UPDATE_CANARY_GATE_CREATE <job_id> <canary_satisfaction_authority_id> [owner_approval_decision_id]
HOT_UPDATE_GATE_PHASE <job_id> <hot_update_id> validated
HOT_UPDATE_GATE_PHASE <job_id> <hot_update_id> staged
HOT_UPDATE_GATE_EXECUTE <job_id> <hot_update_id>
HOT_UPDATE_GATE_RELOAD <job_id> <hot_update_id>
HOT_UPDATE_OUTCOME_CREATE <job_id> <hot_update_id>
HOT_UPDATE_PROMOTION_CREATE <job_id> <outcome_id>
```

`HOT_UPDATE_LKG_RECERTIFY <job_id> <promotion_id>` remains available after promotion, and rollback / rollback-apply commands remain generic recovery controls.

No additional command surface is required for canary path completion through promotion audit lineage.

## Current Read-Model Chain

The completed read-model chain exposes:

- `hot_update_canary_requirement_identity`
- `hot_update_canary_evidence_identity`
- `hot_update_canary_satisfaction_identity`
- `hot_update_canary_satisfaction_authority_identity`
- `hot_update_owner_approval_request_identity`
- `hot_update_owner_approval_decision_identity`
- `hot_update_gate_identity` with `canary_ref` and `approval_ref`
- `hot_update_outcome_identity` with copied `canary_ref` and `approval_ref`
- `promotion_identity` with copied `canary_ref` and `approval_ref`
- `rollback_identity`
- `rollback_apply_identity`
- runtime pack active/LKG identity

Invalid records remain visible as invalid statuses without hiding valid records. Read models remain read-only and do not repair records.

No missing read-model surface blocks canary-required completion through promotion audit lineage.

## Execution Guard Summary

`AssessHotUpdateCanaryGateExecutionReadiness(root, hotUpdateID)` is shared and read-only.

For non-canary gates, it returns ready/not-applicable and does not require canary records.

For no-owner canary gates, it requires:

- gate `canary_ref` equals the authority ID
- empty `approval_ref`
- authority `state=authorized`
- `owner_approval_required=false`
- fresh satisfaction remains `satisfied`
- fresh eligibility remains `canary_required`
- selected evidence still loads, is still selected, and is passed
- source refs, runtime packs, active pointer context, rollback target, and present LKG pointer remain valid

For owner-approved canary gates, it requires:

- gate `canary_ref` equals the authority ID
- non-empty `approval_ref`
- authority `state=waiting_owner_approval`
- `owner_approval_required=true`
- fresh satisfaction remains `waiting_owner_approval`
- fresh eligibility remains `canary_and_owner_approval_required`
- owner approval decision loads, matches copied refs, and has `decision=granted`
- rejected, missing, stale, or mismatched owner approval decisions fail closed

The guard is called before canary-derived phase advancement to `validated` or `staged`, before pointer switch, and before reload/apply convergence writes.

## Outcome And Promotion Lineage

`CreateHotUpdateOutcomeFromTerminalGate(...)` remains generic after guarded execution and does not re-run canary readiness. It records terminal state:

- `reload_apply_succeeded` -> `outcome_kind=hot_updated`
- `reload_apply_failed` -> `outcome_kind=failed`

For canary-derived gates, it copies `canary_ref` and `approval_ref` from the linked gate into `HotUpdateOutcomeRecord`.

`CreatePromotionFromSuccessfulHotUpdateOutcome(...)` remains generic after a successful terminal outcome and does not re-run canary readiness. It copies `canary_ref` and `approval_ref` from the linked outcome into `PromotionRecord` and fails closed if outcome/gate/promotion lineage diverges.

This preserves downstream audit lineage without blocking outcome or promotion creation because canary source records drift after terminal execution.

## Rollback, Rollback-Apply, And LKG

Rollback, rollback-apply, and LKG are intentionally generic after promotion.

Rollback derives restoration authority from promotion, hot-update outcome/gate linkage, from-pack, target-pack, and optional LKG consistency. Rollback-apply then performs the recovery pointer switch and reload/apply workflow. These paths should not be blocked by stale canary evidence, stale canary satisfaction authority, or owner approval drift after execution.

`RecertifyLastKnownGoodFromPromotion(...)` consumes a promotion and linked successful outcome, requires the active pointer to match the promoted pack, and records the promotion as LKG basis. It does not need a canary-specific reauthorization because promotion now carries copied canary/approval audit lineage.

Generic rollback/LKG behavior is sufficient for now. A future rollback/LKG policy assessment is not required before the canary-required path can be considered complete through promotion audit lineage.

## Checkpoint Answers

The canary-required path is now complete from candidate result through promotion audit lineage. The no-owner branch carries `canary_ref`; the owner-approved branch carries both `canary_ref` and `approval_ref`.

Rollback, rollback-apply, and LKG are intentionally generic after promotion. They are recovery and recertification paths, not canary authority reauthorization points.

There are no remaining missing authority records before promotion. The chain has requirement, evidence, satisfaction assessment, satisfaction authority, optional owner approval request/decision, guarded gate execution, outcome, and promotion lineage.

There are no remaining missing read-model surfaces required for canary path completion through promotion. Outcome and promotion identities now expose copied canary/approval refs.

There are no remaining command surfaces required for canary path completion through promotion. Operator ergonomics may still need help/runbook updates, but the command surface exists.

Natural-language owner approval is still intentionally not bound to this path. Exact durable `granted` / `rejected` owner approval decisions remain the authority surface.

`CandidatePromotionDecisionRecord` is still strictly eligible-only.

`CreateHotUpdateGateFromCandidatePromotionDecision(...)` is still unchanged and remains the eligible-only gate path.

Existing rollback/LKG paths do not need an immediate future policy assessment; generic behavior is sufficient for now because promotion carries audit lineage and recovery must not be blocked by later canary drift.

## Intentionally Not Implemented

V4 still does not implement:

- natural-language owner approval aliases
- automatic canary execution or telemetry collection
- canary-specific outcome kinds for successful hot updates
- canary-specific rollback policy
- canary-specific rollback-apply policy
- canary-specific LKG recertification policy
- direct canary fields on rollback, rollback-apply, or LKG records
- command syntax changes
- new TaskState wrappers
- broadening of `CandidatePromotionDecisionRecord`
- candidate promotion decisions for canary-required states
- owner approval as a substitute for passed canary evidence
- rejected owner approval as any form of execution authority

## Top Residual Risks

1. Operator workflow complexity is now high. The command chain is complete but long, and the help/runbook surface can lag the implemented path.
2. End-to-end integration evidence is split across focused registry/direct-command tests rather than a single named runbook-style fixture.
3. Rollback/LKG lineage is indirect through promotion and outcome, which is intentional but requires operators to follow refs across records.
4. Existing `canary_applied` outcome kind remains unused for this path; this is intentional, but future work should avoid accidentally treating the enum as required authority.
5. Manual corruption of source records after terminal execution can still make historical canary records invalid, even though outcome/promotion creation intentionally preserves terminal accounting instead of re-authorizing.

## Recommendation

The next slice should be exactly:

```text
V4-126 - Canary Gate Operator Runbook And Help Text Update
```

Scope recommendation:

- document the complete no-owner and owner-approved command sequences
- document rejected owner approval as terminal blocker
- document which status/read-model surfaces to inspect at each step
- document that outcome/promotion creation does not re-run canary readiness after terminal execution
- document that rollback, rollback-apply, and LKG remain generic
- do not add commands, change syntax, change Go behavior, or widen policy

This is the smallest safe next slice because the lifecycle is now functionally complete through promotion audit lineage, and the highest immediate risk is operator misuse of a long command path rather than a missing authority record.

## Explicit Non-Goals

V4-125 does not:

- change Go code or tests
- add commands
- add TaskState wrappers
- create records
- mutate runtime pointers
- mutate LKG
- mutate `reload_generation`
- change outcome kinds
- change rollback or rollback-apply behavior
- bind natural-language owner approval
- broaden `CandidatePromotionDecisionRecord`
- change `CreateHotUpdateGateFromCandidatePromotionDecision(...)`
- implement V4-126
