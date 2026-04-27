# V4-128 Canary Gate End-to-End Runbook Fixture - After

## Scope

V4-128 adds a bounded direct-command runbook fixture and maintenance notes for the completed canary-required hot-update path. Production code is unchanged.

Dedicated test file:

`internal/agent/loop_processdirect_canary_runbook_test.go`

## No-Owner Branch Fixture

`TestProcessDirectCanaryRunbookNoOwnerBranchThroughPromotion` executes the documented no-owner sequence:

1. `HOT_UPDATE_CANARY_REQUIREMENT_CREATE`
2. `HOT_UPDATE_CANARY_EVIDENCE_CREATE ... passed ...`
3. `HOT_UPDATE_CANARY_SATISFACTION_AUTHORITY_CREATE`
4. `HOT_UPDATE_CANARY_GATE_CREATE`
5. `HOT_UPDATE_GATE_PHASE ... validated`
6. `HOT_UPDATE_GATE_PHASE ... staged`
7. `HOT_UPDATE_GATE_EXECUTE`
8. `HOT_UPDATE_GATE_RELOAD`
9. `HOT_UPDATE_OUTCOME_CREATE`
10. `HOT_UPDATE_PROMOTION_CREATE`

The fixture asserts configured status identities for canary requirement, evidence, satisfaction, satisfaction authority, gate, outcome, and promotion. It verifies that the gate, outcome, and promotion carry the canary satisfaction authority ID as `canary_ref` and keep `approval_ref` empty. The outcome remains `outcome_kind=hot_updated`, and the promotion links the deterministic outcome and hot-update IDs.

## Owner-Approved Branch Fixture

`TestProcessDirectCanaryRunbookOwnerApprovedBranchThroughPromotion` executes the documented owner-approved sequence:

1. `HOT_UPDATE_CANARY_REQUIREMENT_CREATE`
2. `HOT_UPDATE_CANARY_EVIDENCE_CREATE ... passed ...`
3. `HOT_UPDATE_CANARY_SATISFACTION_AUTHORITY_CREATE`
4. `HOT_UPDATE_OWNER_APPROVAL_REQUEST_CREATE`
5. `HOT_UPDATE_OWNER_APPROVAL_DECISION_CREATE ... granted ...`
6. `HOT_UPDATE_CANARY_GATE_CREATE ... <owner_approval_decision_id>`
7. `HOT_UPDATE_GATE_PHASE ... validated`
8. `HOT_UPDATE_GATE_PHASE ... staged`
9. `HOT_UPDATE_GATE_EXECUTE`
10. `HOT_UPDATE_GATE_RELOAD`
11. `HOT_UPDATE_OUTCOME_CREATE`
12. `HOT_UPDATE_PROMOTION_CREATE`

The fixture asserts configured status identities for the canary path plus owner approval request and owner approval decision. It verifies that the owner decision is exactly `granted`, and that gate, outcome, and promotion copy both `canary_ref` and `approval_ref`.

## Rejected-Owner Blocker

`TestProcessDirectCanaryRunbookRejectedOwnerApprovalBlocksGate` records a rejected owner approval decision and then attempts `HOT_UPDATE_CANARY_GATE_CREATE` with that decision. The command fails closed, no hot-update gate is created, and no outcome, promotion, rollback, rollback-apply, candidate promotion decision, natural-language runtime approval request, or natural-language runtime approval grant record is created.

## Determinism

The tests use fixed RFC3339Nano observed timestamps and derive IDs through the existing deterministic helpers:

- `HotUpdateCanaryRequirementIDFromResult`
- `HotUpdateCanaryEvidenceIDFromRequirementObservedAt`
- `HotUpdateCanarySatisfactionAuthorityIDFromRequirementEvidence`
- `HotUpdateOwnerApprovalRequestIDFromCanarySatisfactionAuthority`
- `HotUpdateOwnerApprovalDecisionIDFromRequest`
- `HotUpdateGateIDFromCanarySatisfactionAuthority`

Outcome and promotion IDs follow the live deterministic conventions already used by the direct-command tests: `hot-update-outcome-<hot_update_id>` and `hot-update-promotion-<hot_update_id>`.

## Side-Effect Invariants

The fixture asserts that active pointer bytes stay unchanged until `HOT_UPDATE_GATE_EXECUTE`, then change to the candidate pack with exactly one `reload_generation` increment. It then verifies that reload, outcome creation, promotion creation, and status reads do not further mutate the active pointer or `reload_generation`.

The fixture also asserts that the LKG pointer is absent or byte-stable, rollback records are absent, rollback-apply records are absent, and candidate promotion decision records are absent. LKG recertification is intentionally excluded because the V4 canary path is complete through promotion audit lineage and rollback/LKG remain generic post-promotion flows.

## Preserved Invariants

V4-128 preserves:

- no production behavior change;
- no command syntax change;
- no new command;
- no new TaskState wrapper;
- no natural-language approval binding;
- no `CandidatePromotionDecisionRecord` broadening;
- no change to `CreateHotUpdateGateFromCandidatePromotionDecision(...)`;
- no rollback, rollback-apply, or LKG behavior change;
- no pointer mutation outside existing successful gate execution;
- no `reload_generation` mutation outside existing successful gate execution;
- no outcome kind change;
- no V4-129 work.
