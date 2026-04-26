# V4-109 Hot-Update Owner Approval Request Registry After

## Implemented Surface

V4-109 adds a durable missioncontrol owner approval request record:

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

The only stored request state is:

```text
requested
```

## Storage Path

Records are stored under:

```text
runtime_packs/hot_update_owner_approval_requests/<owner_approval_request_id>.json
```

## Deterministic ID

The deterministic ID is:

```text
hot-update-owner-approval-request-<canary_satisfaction_authority_id>
```

The helper is:

```go
HotUpdateOwnerApprovalRequestIDFromCanarySatisfactionAuthority(canarySatisfactionAuthorityID string) string
```

## Validation Behavior

Validation rejects:

- missing or invalid `record_version`
- missing `owner_approval_request_id`
- missing `canary_satisfaction_authority_id`
- missing copied source refs
- invalid `authority_state`
- `authority_state` other than `waiting_owner_approval`
- invalid `satisfaction_state`
- `satisfaction_state` other than `waiting_owner_approval`
- `owner_approval_required=false`
- invalid request `state`
- `state` other than `requested`
- missing `reason`
- zero `created_at`
- missing `created_by`
- deterministic ID mismatch

## Source Authority Records

Creation and linkage validation cross-check:

- `HotUpdateCanarySatisfactionAuthorityRecord`
- `HotUpdateCanaryRequirementRecord`
- selected `HotUpdateCanaryEvidenceRecord`
- fresh canary satisfaction assessment
- committed `CandidateResultRecord`
- linked `ImprovementRunRecord`
- linked `ImprovementCandidateRecord`
- frozen `EvalSuiteRecord`
- referenced `PromotionPolicyRecord`
- baseline `RuntimePackRecord`
- candidate `RuntimePackRecord`
- freshly derived `CandidatePromotionEligibilityStatus`

Fresh canary satisfaction must still be:

```text
waiting_owner_approval
```

Fresh eligibility must remain:

```text
canary_and_owner_approval_required
```

## Creation Helper Behavior

V4-109 adds:

```go
CreateHotUpdateOwnerApprovalRequestFromCanarySatisfactionAuthority(root, canarySatisfactionAuthorityID, createdBy string, createdAt time.Time) (HotUpdateOwnerApprovalRequestRecord, bool, error)
```

The helper:

- validates the store root
- rejects missing `created_by`
- rejects zero `created_at`
- loads the committed canary satisfaction authority
- requires `state=waiting_owner_approval`
- requires `owner_approval_required=true`
- requires `satisfaction_state=waiting_owner_approval`
- requires a non-empty selected canary evidence ID
- copies all source refs from the canary satisfaction authority
- sets `state=requested`
- sets reason `hot-update owner approval requested after canary satisfaction`
- does not accept caller-supplied source refs beyond authority ID, `created_by`, and `created_at`
- does not mutate the canary satisfaction authority

## Replay And Duplicate Behavior

Storage behavior:

- first write stores the normalized record and returns `changed=true`
- exact replay returns the existing record and `changed=false`
- exact replay is byte-stable
- divergent duplicate for the same `owner_approval_request_id` fails closed
- another request for the same `canary_satisfaction_authority_id` under a different ID fails closed
- list order is deterministic by filename

## Status And Read Model

Mission status now includes:

```json
"hot_update_owner_approval_request_identity": { ... }
```

The identity surfaces:

- `configured`
- `not_configured`
- `invalid`

Minimum status fields are:

- `state`
- `owner_approval_request_id`
- `canary_satisfaction_authority_id`
- `canary_requirement_id`
- `selected_canary_evidence_id`
- `result_id`
- `run_id`
- `candidate_id`
- `eval_suite_id`
- `promotion_policy_id`
- `baseline_pack_id`
- `candidate_pack_id`
- `authority_state`
- `satisfaction_state`
- `owner_approval_required`
- `request_state`
- `reason`
- `created_at`
- `created_by`
- `error`

Invalid request records are surfaced without hiding valid records. The read model does not mutate records.

Committed mission status snapshots include the owner approval request identity through the same composition path as the existing V4 hot-update identities. `STATUS <job_id>` includes the identity through the existing readout adapter; no separate status command was added.

## Invariants Preserved

V4-109 preserves:

- no direct command
- no TaskState mutation wrapper
- no owner approval grant creation
- no owner approval rejection creation
- no owner approval expiration record creation
- no natural-language owner approval binding
- no candidate promotion decision for canary-required states
- no broadening of `CandidatePromotionDecisionRecord`
- no hot-update gate for canary-required states
- no canary execution
- no canary evidence creation
- no outcome creation
- no promotion creation
- no rollback creation
- no rollback-apply creation
- no candidate result mutation
- no canary requirement mutation
- no canary evidence mutation
- no canary satisfaction authority mutation
- no owner approval request mutation after create/select
- no promotion policy mutation
- no runtime pack mutation
- no active runtime-pack pointer mutation
- no last-known-good pointer mutation
- no `reload_generation` mutation
- no pointer-switch behavior change
- no reload/apply behavior change
- no V4-110 work
