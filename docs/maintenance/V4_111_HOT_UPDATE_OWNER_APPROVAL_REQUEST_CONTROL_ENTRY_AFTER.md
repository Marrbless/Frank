# V4-111 Hot-Update Owner Approval Request Control Entry After

## Implemented Surface

V4-111 adds the direct operator command:

```text
HOT_UPDATE_OWNER_APPROVAL_REQUEST_CREATE <job_id> <canary_satisfaction_authority_id>
```

It exposes the existing V4-109 helper:

```go
missioncontrol.CreateHotUpdateOwnerApprovalRequestFromCanarySatisfactionAuthority(root, canarySatisfactionAuthorityID, "operator", createdAt)
```

through a `TaskState` wrapper:

```go
CreateHotUpdateOwnerApprovalRequestFromCanarySatisfactionAuthority(jobID string, canarySatisfactionAuthorityID string)
```

## TaskState Wrapper Behavior

The wrapper:

- returns the zero record, `false`, and `nil` when `TaskState` is nil
- uses audit action `hot_update_owner_approval_request_create`
- derives `now` from `taskStateTransitionTimestamp(taskStateNowUTC())`
- clones execution context, runtime control, runtime state, and mission store root under lock
- builds audit context from active execution context, or from persisted runtime/control context when active context is absent
- validates the mission store root
- validates active execution context when present
- validates persisted runtime state and runtime control context when active context is absent
- rejects supplied `job_id` when it does not match active or persisted job context
- normalizes and validates `canary_satisfaction_authority_id`
- derives deterministic request ID from canary satisfaction authority ID
- reuses existing request `CreatedAt` when the deterministic request record already loads
- fails closed if an existing deterministic request record returns anything other than not-found
- calls the missioncontrol helper with `created_by=operator`
- emits audit on created, selected, and rejected paths
- returns the missioncontrol record and changed flag

## CreatedAt Replay Reuse

Exact replay is byte-stable. The wrapper derives:

```text
hot-update-owner-approval-request-<canary_satisfaction_authority_id>
```

and loads that request before calling the helper. If the request already loads successfully, the wrapper passes the existing `CreatedAt` back into the helper.

If the record is absent, the wrapper uses the current `TaskState` timestamp. If loading fails for any other reason, the command fails closed before attempting creation.

## Response Semantics

Created:

```text
Created hot-update owner approval request job=<job_id> canary_satisfaction_authority=<canary_satisfaction_authority_id> owner_approval_request=<owner_approval_request_id> request_state=requested authority_state=waiting_owner_approval owner_approval_required=true.
```

Selected:

```text
Selected hot-update owner approval request job=<job_id> canary_satisfaction_authority=<canary_satisfaction_authority_id> owner_approval_request=<owner_approval_request_id> request_state=requested authority_state=waiting_owner_approval owner_approval_required=true.
```

Malformed:

```text
HOT_UPDATE_OWNER_APPROVAL_REQUEST_CREATE requires job_id and canary_satisfaction_authority_id
```

Failures return the existing direct-command pattern: empty response plus surfaced error.

## Audit Behavior

All command paths emit `hot_update_owner_approval_request_create`:

- created request: allowed/applied audit
- selected idempotent replay: allowed/applied audit
- rejected fail-closed path: rejected audit with the existing validation code style

This audit action is distinct from runtime `approval_history`. V4-111 does not create or resolve runtime `ApprovalRequestRecord` or `ApprovalGrantRecord`.

## Fail-Closed Behavior

The command fails closed for:

- malformed arguments
- missing mission store root
- missing active/persisted mission context
- wrong `job_id`
- invalid `canary_satisfaction_authority_id`
- missing canary satisfaction authority
- invalid canary satisfaction authority
- authority state other than `waiting_owner_approval`
- `owner_approval_required=false`
- satisfaction state other than `waiting_owner_approval`
- missing selected canary evidence ID
- missing or invalid canary requirement
- selected canary evidence missing, non-passed, stale, or mismatched
- missing candidate result
- missing linked improvement run
- missing linked improvement candidate
- missing or non-linkable eval suite
- missing promotion policy
- missing baseline runtime pack
- missing candidate runtime pack
- fresh canary satisfaction away from `waiting_owner_approval`
- fresh eligibility away from `canary_and_owner_approval_required`
- copied ref mismatch across authority, requirement, evidence, assessment, candidate result, and fresh eligibility
- existing deterministic request record that fails to load or validate
- divergent duplicate request record
- another request for the same authority under a different ID

Invalid/stale request records remain visible through the read-model and are not overwritten.

## Status / Read Model

No separate status command was added. Existing:

```text
STATUS <job_id>
```

surfaces created owner approval requests through:

```json
"hot_update_owner_approval_request_identity"
```

## Invariants Preserved

V4-111 preserves:

- no owner approval grant creation
- no owner approval rejection creation
- no owner approval expiration record creation
- no natural-language owner approval binding
- no runtime `ApprovalRequestRecord` mutation
- no runtime `ApprovalGrantRecord` mutation
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
- no V4-112 work
