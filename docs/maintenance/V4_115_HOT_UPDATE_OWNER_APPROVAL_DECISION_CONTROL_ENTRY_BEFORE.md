# V4-115 Hot-Update Owner Approval Decision Control Entry Before

## Before-State Gap From V4-114

V4-113 added the durable `HotUpdateOwnerApprovalDecisionRecord` registry and read model, and V4-114 assessed the governed operator control entry needed to create those records. Before V4-115, operators could see owner approval decision identity status, but there was no direct command or `TaskState` wrapper to create/select a decision through the governed control path.

## Existing Durable Authority

The source helper already existed:

```go
CreateHotUpdateOwnerApprovalDecisionFromRequest(root, ownerApprovalRequestID string, decision HotUpdateOwnerApprovalDecision, decidedBy string, decidedAt time.Time, reason string) (HotUpdateOwnerApprovalDecisionRecord, bool, error)
```

It consumes `owner_approval_request_id`, derives:

```text
hot-update-owner-approval-decision-<owner_approval_request_id>
```

and stores records under:

```text
runtime_packs/hot_update_owner_approval_decisions/<sha256(owner_approval_decision_id)>/record.json
```

Supported terminal decisions were already:

```text
granted
rejected
```

## Missing Control Surface

The missing surface was:

```text
HOT_UPDATE_OWNER_APPROVAL_DECISION_CREATE <job_id> <owner_approval_request_id> <decision> [reason...]
```

with a `TaskState` wrapper that validates the active or persisted job context, resolves the mission store root, derives `decided_by=operator`, derives `decided_at` from the TaskState timestamp path, reuses existing `DecidedAt` for exact replay, applies a deterministic default reason when omitted, emits audit, and calls the existing missioncontrol helper.

## Invariants To Preserve

V4-115 must preserve:

- no natural-language owner approval binding
- no runtime `ApprovalRequestRecord` mutation
- no runtime `ApprovalGrantRecord` mutation
- no separate grant/rejection registries
- no owner approval expiration record
- no candidate promotion decision for canary-required states
- no broadening of `CandidatePromotionDecisionRecord`
- no hot-update gate for canary-required states
- no `approval_ref` population
- no `canary_ref` population
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
- no V4-116 work
