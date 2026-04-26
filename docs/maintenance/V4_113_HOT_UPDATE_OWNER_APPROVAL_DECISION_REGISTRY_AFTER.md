# V4-113 Hot-Update Owner Approval Decision Registry After

## Implemented Surface

V4-113 adds a durable missioncontrol terminal owner approval decision record:

```go
type HotUpdateOwnerApprovalDecisionRecord struct {
    RecordVersion int
    OwnerApprovalDecisionID string
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
    RequestState HotUpdateOwnerApprovalRequestState
    AuthorityState HotUpdateCanarySatisfactionAuthorityState
    SatisfactionState HotUpdateCanarySatisfactionState
    OwnerApprovalRequired bool
    Decision HotUpdateOwnerApprovalDecision
    Reason string
    DecidedAt time.Time
    DecidedBy string
}
```

Supported terminal decision values are:

```text
granted
rejected
```

V4-113 intentionally uses one decision record with a `decision` value rather than separate grant and rejection registries, so a request can have only one terminal decision.

## Storage Path

Records are stored under:

```text
runtime_packs/hot_update_owner_approval_decisions/<sha256(owner_approval_decision_id)>/record.json
```

The hash-keyed directory layout preserves the exact deterministic decision ID inside the record while keeping path segments and atomic temp filenames short enough for filesystems that reject very long `<id>.json.tmp-*` names.

## Deterministic ID

The deterministic ID is:

```text
hot-update-owner-approval-decision-<owner_approval_request_id>
```

The helper is:

```go
HotUpdateOwnerApprovalDecisionIDFromRequest(ownerApprovalRequestID string) string
```

The decision value is record data, not part of the ID. This preserves the invariant that one owner approval request can have at most one terminal owner approval decision.

## Validation Behavior

Validation rejects:

- missing or invalid `record_version`
- missing `owner_approval_decision_id`
- missing `owner_approval_request_id`
- missing copied source refs
- invalid `request_state`
- `request_state` other than `requested`
- invalid `authority_state`
- `authority_state` other than `waiting_owner_approval`
- invalid `satisfaction_state`
- `satisfaction_state` other than `waiting_owner_approval`
- `owner_approval_required=false`
- invalid `decision`
- any decision other than `granted` or `rejected`
- missing `reason`
- zero `decided_at`
- missing `decided_by`
- deterministic decision ID mismatch

## Source Authority Records

Creation and linkage validation consume `owner_approval_request_id` and cross-check through the existing owner approval request authority chain:

- `HotUpdateOwnerApprovalRequestRecord`
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

Required states remain:

```text
request_state=requested
authority_state=waiting_owner_approval
satisfaction_state=waiting_owner_approval
owner_approval_required=true
fresh canary satisfaction=waiting_owner_approval
fresh eligibility=canary_and_owner_approval_required
```

Selected canary evidence must still be passed. Owner approval does not substitute for canary satisfaction.

## Creation Helper Behavior

V4-113 adds:

```go
CreateHotUpdateOwnerApprovalDecisionFromRequest(root, ownerApprovalRequestID string, decision HotUpdateOwnerApprovalDecision, decidedBy string, decidedAt time.Time, reason string) (HotUpdateOwnerApprovalDecisionRecord, bool, error)
```

The helper:

- validates the store root
- rejects missing `owner_approval_request_id`
- rejects invalid `decision`
- rejects missing `decided_by`
- rejects zero `decided_at`
- rejects missing `reason`
- loads the committed owner approval request
- requires `request_state=requested`
- copies stable refs from the request
- sets `decision=granted` or `decision=rejected`
- does not accept caller-supplied source refs beyond request ID, decision, `decided_by`, `decided_at`, and reason
- does not create a hot-update gate
- does not mutate the owner approval request

## Granted Vs Rejected

Both `granted` and `rejected` are terminal owner approval decisions for the request.

A future hot-update gate helper may consume only a `granted` decision as `approval_ref`. A `rejected` decision is terminal blocking authority and must not be treated as approval for a gate.

## Replay And Duplicate Behavior

Storage behavior:

- first write stores the normalized record and returns `changed=true`
- exact replay returns the existing record and `changed=false`
- exact replay is byte-stable
- divergent duplicate for the same `owner_approval_decision_id` fails closed
- any second terminal decision for the same `owner_approval_request_id` fails closed
- a different decision for a request that already has a terminal decision fails closed
- invalid existing deterministic decision records fail closed through load/list helpers and remain visible in status
- list order is deterministic by storage directory name

## Status And Read Model

Mission status now includes:

```json
"hot_update_owner_approval_decision_identity": { ... }
```

The identity surfaces:

- `configured`
- `not_configured`
- `invalid`

Minimum status fields are:

- `state`
- `owner_approval_decision_id`
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
- `request_state`
- `authority_state`
- `satisfaction_state`
- `owner_approval_required`
- `decision`
- `reason`
- `decided_at`
- `decided_by`
- `error`

Invalid decision records are surfaced without hiding valid records. The read model does not mutate records.

Committed mission status snapshots include the owner approval decision identity through the same composition path as the existing V4 hot-update identities. `STATUS <job_id>` includes the identity through the existing readout adapter; no separate status command was added.

## Invariants Preserved

V4-113 preserves:

- no direct command
- no `TaskState` wrapper
- no natural-language owner approval binding
- no runtime `ApprovalRequestRecord` mutation
- no runtime `ApprovalGrantRecord` mutation
- no separate grant/rejection registries
- no owner approval expiration record
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
- no owner approval request mutation
- no owner approval decision mutation after create/select
- no promotion policy mutation
- no runtime pack mutation
- no active runtime-pack pointer mutation
- no last-known-good pointer mutation
- no `reload_generation` mutation
- no pointer-switch behavior change
- no reload/apply behavior change
- no V4-114 work
