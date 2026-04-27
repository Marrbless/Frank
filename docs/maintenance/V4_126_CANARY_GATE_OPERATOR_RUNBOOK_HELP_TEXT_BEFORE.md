# V4-126 Canary Gate Operator Runbook And Help Text Before

## Scope

V4-126 updates operator documentation for the completed canary-required hot-update lifecycle. This slice is documentation/help text only: it does not change command syntax, direct-command parsing, TaskState wrappers, missioncontrol validation, record schemas, runtime behavior, rollback behavior, LKG behavior, or tests.

## Live State Inspected

The slice starts from V4-125:

- commit: `77c77499f1cbc0ba0eeb7880d68eba20f0ed5306`
- tag: `frank-v4-125-canary-gate-end-to-end-lifecycle-checkpoint`
- branch: `frank-v4-126-canary-gate-operator-runbook-help-text`

The startup validation passed with:

```text
/usr/local/go/bin/go test -count=1 ./...
```

The worktree was clean before the V4-126 documentation changes.

## Authority Sources Read

- `docs/FRANK_V4_SPEC.md`
- `docs/maintenance/V4_125_CANARY_GATE_END_TO_END_LIFECYCLE_CHECKPOINT.md`
- `docs/maintenance/V4_124_CANARY_DERIVED_OUTCOME_PROMOTION_AUDIT_LINEAGE_AFTER.md`

The checkpoint establishes that the canary-required path is complete through promotion audit lineage, that rollback/rollback-apply/LKG remain generic, that natural-language owner approval is intentionally not bound, and that the eligible-only candidate-promotion-decision path remains separate.

## Operator Surface Inspected

The existing operator runbook is:

- `docs/HOT_UPDATE_OPERATOR_RUNBOOK.md`

Live direct-command and TaskState code paths expose the canary-required lifecycle commands:

- `HOT_UPDATE_CANARY_REQUIREMENT_CREATE`
- `HOT_UPDATE_CANARY_EVIDENCE_CREATE`
- `HOT_UPDATE_CANARY_SATISFACTION_AUTHORITY_CREATE`
- `HOT_UPDATE_OWNER_APPROVAL_REQUEST_CREATE`
- `HOT_UPDATE_OWNER_APPROVAL_DECISION_CREATE`
- `HOT_UPDATE_CANARY_GATE_CREATE`
- `HOT_UPDATE_GATE_PHASE`
- `HOT_UPDATE_GATE_EXECUTE`
- `HOT_UPDATE_GATE_RELOAD`
- `HOT_UPDATE_OUTCOME_CREATE`
- `HOT_UPDATE_PROMOTION_CREATE`

Live status/read-model fields expose the required identity surfaces:

- `hot_update_canary_requirement_identity`
- `hot_update_canary_evidence_identity`
- `hot_update_canary_satisfaction_identity`
- `hot_update_canary_satisfaction_authority_identity`
- `hot_update_owner_approval_request_identity`
- `hot_update_owner_approval_decision_identity`
- `hot_update_gate_identity`
- `hot_update_outcome_identity`
- `promotion_identity`
- `rollback_identity`
- `rollback_apply_identity`
- `runtime_pack_identity.active`
- `runtime_pack_identity.last_known_good`

## Before-State Gap

Before V4-126, operators could reconstruct the canary-required path from maintenance memos and live code, but the hot-update runbook did not provide a single operational sequence covering:

- no-owner canary branch
- owner-approved canary branch
- rejected owner approval behavior
- status/read-model checkpoints
- fail-closed guard behavior
- outcome/promotion audit-lineage expectations
- intentionally generic rollback, rollback-apply, and LKG behavior

## Help Text Surface

No central direct-command help command or generated operator command summary was found for these operator commands. Cobra CLI help exists for the binary surface, but the direct-command hot-update lifecycle is documented through the runbook. V4-126 therefore keeps code untouched and updates the existing runbook plus this maintenance record.

## Non-Goals

V4-126 does not add commands, change command syntax, add TaskState wrappers, mutate records, create runtime records, execute gates, pointer-switch, reload/apply, create outcomes, create promotions, create rollbacks, create rollback-apply records, mutate active/LKG pointers, bind natural-language owner approval, broaden `CandidatePromotionDecisionRecord`, or change `CreateHotUpdateGateFromCandidatePromotionDecision(...)`.
