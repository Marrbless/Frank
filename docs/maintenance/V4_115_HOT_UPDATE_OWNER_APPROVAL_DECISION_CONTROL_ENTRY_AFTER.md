# V4-115 Hot-Update Owner Approval Decision Control Entry After

## Implemented Surface

V4-115 exposes the existing V4-113 decision helper through the governed direct operator path:

```text
HOT_UPDATE_OWNER_APPROVAL_DECISION_CREATE <job_id> <owner_approval_request_id> <decision> [reason...]
```

Allowed decision values are exactly:

```text
granted
rejected
```

Aliases such as `approve`, `approved`, `deny`, `denied`, `yes`, `no`, and any other value fail closed.

## TaskState Wrapper

V4-115 adds:

```go
func (s *TaskState) CreateHotUpdateOwnerApprovalDecisionFromRequest(
    jobID string,
    ownerApprovalRequestID string,
    decision missioncontrol.HotUpdateOwnerApprovalDecision,
    reason string,
) (missioncontrol.HotUpdateOwnerApprovalDecisionRecord, bool, error)
```

The wrapper:

- returns zero record, `false`, `nil` when `s == nil`
- uses audit action `hot_update_owner_approval_decision_create`
- derives `now` from `taskStateTransitionTimestamp(taskStateNowUTC())`
- clones execution context, runtime control context, runtime state, and mission store root under lock
- builds audit context from active execution context or persisted runtime/control context
- validates the mission store root
- validates active job/step/runtime context when present
- otherwise validates persisted runtime state and runtime control context
- rejects a supplied `job_id` that does not match active or persisted job binding
- normalizes and validates `owner_approval_request_id`
- requires decision to be exactly `granted` or `rejected`
- trims reason
- applies a deterministic default reason when reason is omitted
- derives the deterministic decision ID with `HotUpdateOwnerApprovalDecisionIDFromRequest`
- loads an existing deterministic decision record and reuses its `DecidedAt` for replay stability
- fails closed on deterministic decision load errors other than not-found
- calls `missioncontrol.CreateHotUpdateOwnerApprovalDecisionFromRequest(root, ownerApprovalRequestID, decision, "operator", decidedAt, reason)`
- emits audit on created, selected/idempotent, and rejected failure paths
- returns the missioncontrol record and `changed` flag

## DecidedAt Replay Reuse

On first create, `decided_at` is the TaskState timestamp. On exact replay, the wrapper loads the deterministic existing decision record and reuses `existing.DecidedAt` before calling the missioncontrol helper. This keeps exact replay byte-stable when the normalized request ID, decision, reason/default reason, `decided_by`, and reused `decided_at` match.

If an existing decision was created with a custom reason, replaying the command without that same reason uses the deterministic default reason and fails closed as a divergent duplicate.

## Default Reason Behavior

When no reason is supplied, the wrapper uses:

```text
hot-update owner approval decision granted
hot-update owner approval decision rejected
```

Caller-supplied reasons are trimmed and must replay exactly after normalization.

## Response Semantics

Created response:

```text
Created hot-update owner approval decision job=<job_id> owner_approval_request=<owner_approval_request_id> owner_approval_decision=<owner_approval_decision_id> decision=<decision>.
```

Selected response:

```text
Selected hot-update owner approval decision job=<job_id> owner_approval_request=<owner_approval_request_id> owner_approval_decision=<owner_approval_decision_id> decision=<decision>.
```

Malformed command error:

```text
HOT_UPDATE_OWNER_APPROVAL_DECISION_CREATE requires job_id, owner_approval_request_id, decision, and optional reason
```

## Audit Behavior

The direct command emits `hot_update_owner_approval_decision_create`.

Audit is emitted for:

- created path with `Allowed=true`
- selected/idempotent path with `Allowed=true`
- failure path with `Allowed=false`

Failure audit preserves existing runtime-control audit behavior and carries existing validation rejection codes where missioncontrol supplies them.

## Fail-Closed Behavior

The command fails closed for:

- missing or malformed command args
- invalid decision value
- aliases such as `yes`, `no`, `approve`, and `deny`
- missing mission store root
- missing active execution context and missing persisted runtime context
- missing persisted runtime control context on the persisted path
- wrong `job_id`
- missing or invalid `owner_approval_request_id`
- missing owner approval request
- invalid owner approval request
- owner approval request state other than `requested`
- authority state other than `waiting_owner_approval`
- `owner_approval_required=false`
- satisfaction state other than `waiting_owner_approval`
- selected canary evidence missing, non-passed, stale, or mismatched
- missing or mismatched canary requirement
- missing or mismatched canary satisfaction authority
- missing candidate result, run, candidate, eval suite, promotion policy, baseline pack, or candidate pack
- fresh canary satisfaction away from `waiting_owner_approval`
- fresh eligibility away from `canary_and_owner_approval_required`
- copied ref mismatch across request, authority, requirement, evidence, assessment, candidate result, and fresh eligibility
- existing deterministic decision records that fail to load or validate
- divergent duplicate decision records
- a second terminal decision for the same request
- a different decision for a request that already has a terminal decision

## Status And Read Model

No separate status command was added. Existing `STATUS <job_id>` readout surfaces created decisions through:

```json
"hot_update_owner_approval_decision_identity": { ... }
```

The read model remains read-only and continues to surface invalid decision records without hiding valid records.

## Invariants Preserved

V4-115 preserves:

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
