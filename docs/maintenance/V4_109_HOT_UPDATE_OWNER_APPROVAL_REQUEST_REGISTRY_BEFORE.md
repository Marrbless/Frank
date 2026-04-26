# V4-109 Hot-Update Owner Approval Request Registry Before

## Before-State Gap

V4-108 found that existing runtime approval records are not sufficient as canary-gated hot-update owner approval authority. `ApprovalRequestRecord` and `ApprovalGrantRecord` are scoped to job/runtime-step approval flow and do not carry stable refs to the canary satisfaction authority, canary requirement, selected canary evidence, candidate result, promotion policy, baseline pack, and candidate pack.

The current durable V4 chain can record canary satisfaction authority:

```text
HotUpdateCanarySatisfactionAuthorityRecord
state=waiting_owner_approval
owner_approval_required=true
satisfaction_state=waiting_owner_approval
```

That record proves the canary branch is satisfied and that owner approval is still required. It does not create an owner approval request, grant, rejection, hot-update gate, outcome, promotion, rollback, last-known-good change, pointer mutation, or reload/apply change.

## Required V4-109 Surface

V4-109 should add only a missioncontrol registry/read-model skeleton for durable hot-update owner approval requests.

Expected record shape:

```go
type HotUpdateOwnerApprovalRequestRecord struct {
    RecordVersion int
    OwnerApprovalRequestID string
    CanarySatisfactionAuthorityID string
    CanaryRequirementID string
    SelectedCanaryEvidenceID string
    ResultID string
    RunID string
    CandidateID string
    EvalSuiteID string
    PromotionPolicyID string
    BaselinePackID string
    CandidatePackID string
    AuthorityState HotUpdateCanarySatisfactionAuthorityState
    SatisfactionState HotUpdateCanarySatisfactionState
    OwnerApprovalRequired bool
    State HotUpdateOwnerApprovalRequestState
    Reason string
    CreatedAt time.Time
    CreatedBy string
}
```

Storage path:

```text
runtime_packs/hot_update_owner_approval_requests/<owner_approval_request_id>.json
```

Deterministic ID:

```text
hot-update-owner-approval-request-<canary_satisfaction_authority_id>
```

The only stored request state in this slice should be:

```text
requested
```

## Source Authority Records

Creation should consume only:

- `canary_satisfaction_authority_id`
- `created_by`
- `created_at`

The helper should load and cross-check:

- `HotUpdateCanarySatisfactionAuthorityRecord`
- `HotUpdateCanaryRequirementRecord`
- selected `HotUpdateCanaryEvidenceRecord`
- fresh `AssessHotUpdateCanarySatisfaction(...)`
- `CandidateResultRecord`
- linked `ImprovementRunRecord`
- linked `ImprovementCandidateRecord`
- frozen `EvalSuiteRecord`
- `PromotionPolicyRecord`
- baseline `RuntimePackRecord`
- candidate `RuntimePackRecord`
- freshly derived `CandidatePromotionEligibilityStatus`

Fresh canary satisfaction must still be `waiting_owner_approval`. Fresh eligibility must remain `canary_and_owner_approval_required`.

## Invariants To Preserve

V4-109 must not add direct commands, TaskState mutation wrappers, owner approval grants, owner approval rejections, owner approval expiration records, natural-language approval binding, candidate promotion decisions, hot-update gates, outcomes, promotions, rollbacks, rollback-apply records, active pointer mutation, last-known-good mutation, `reload_generation` mutation, pointer-switch changes, reload/apply changes, or V4-110 work.
