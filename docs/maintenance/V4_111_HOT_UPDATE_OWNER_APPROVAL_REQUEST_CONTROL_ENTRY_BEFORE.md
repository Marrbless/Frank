# V4-111 Hot-Update Owner Approval Request Control Entry Before

## Before-State Gap

V4-109 added the durable `HotUpdateOwnerApprovalRequestRecord` registry and the read-only `hot_update_owner_approval_request_identity` status surface.

V4-110 assessed the smallest safe operator control entry and selected:

```text
HOT_UPDATE_OWNER_APPROVAL_REQUEST_CREATE <job_id> <canary_satisfaction_authority_id>
```

Before V4-111, the missioncontrol helper existed:

```go
CreateHotUpdateOwnerApprovalRequestFromCanarySatisfactionAuthority(root, canarySatisfactionAuthorityID, createdBy string, createdAt time.Time) (HotUpdateOwnerApprovalRequestRecord, bool, error)
```

but no direct operator command or `TaskState` wrapper exposed it through the governed control path.

## Required Slice

V4-111 should add only:

- the direct command parser for `HOT_UPDATE_OWNER_APPROVAL_REQUEST_CREATE <job_id> <canary_satisfaction_authority_id>`
- a `TaskState` wrapper around the V4-109 missioncontrol helper
- created/selected response text
- audit action `hot_update_owner_approval_request_create`
- focused tests proving create, replay, fail-closed, audit, status, and no downstream mutation behavior

## Expected TaskState Wrapper Behavior

The wrapper should:

- return the zero record, `false`, and `nil` when `TaskState` is nil
- derive `now` from `taskStateTransitionTimestamp(taskStateNowUTC())`
- clone execution context, runtime control, runtime state, and mission store root under lock
- build audit context from active execution context or persisted runtime/control context
- validate the mission store root
- validate active execution context when present
- otherwise validate persisted runtime state and runtime control context
- reject wrong `job_id` before helper invocation
- normalize and validate `canary_satisfaction_authority_id`
- derive deterministic request ID with `HotUpdateOwnerApprovalRequestIDFromCanarySatisfactionAuthority(...)`
- reuse existing request `CreatedAt` when the deterministic request record already loads
- fail closed if the deterministic request record load returns anything other than not-found
- call the missioncontrol helper with `created_by=operator`
- emit audit action `hot_update_owner_approval_request_create` on success, idempotent selection, and failure
- return the missioncontrol record and changed flag

## Invariants To Preserve

V4-111 must not create owner approval grants, owner approval rejections, owner approval expiration records, runtime `ApprovalRequestRecord` or `ApprovalGrantRecord` mutations, natural-language approval binding, candidate promotion decisions, hot-update gates, canary evidence, outcomes, promotions, rollbacks, rollback-apply records, last-known-good records, pointer mutations, `reload_generation` mutation, pointer-switch changes, reload/apply changes, or V4-112 work.
