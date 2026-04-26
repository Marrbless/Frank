# V4-107 Hot-Update Canary Satisfaction Authority Control Entry After

## Implemented Surface

V4-107 adds the direct operator command:

```text
HOT_UPDATE_CANARY_SATISFACTION_AUTHORITY_CREATE <job_id> <canary_requirement_id>
```

It exposes the existing V4-105 helper:

```go
missioncontrol.CreateHotUpdateCanarySatisfactionAuthorityFromRequirement(root, canaryRequirementID, "operator", createdAt)
```

through a `TaskState` wrapper:

```go
CreateHotUpdateCanarySatisfactionAuthorityFromRequirement(jobID string, canaryRequirementID string)
```

## TaskState Wrapper Behavior

The wrapper:

- returns the zero record, `false`, and `nil` when `TaskState` is nil
- uses audit action `hot_update_canary_satisfaction_authority_create`
- derives `now` from `taskStateTransitionTimestamp(taskStateNowUTC())`
- clones execution context, runtime control, runtime state, and mission store root under lock
- builds audit context from active execution context, or from persisted runtime/control context when active context is absent
- validates the mission store root
- validates active execution context when present
- validates persisted runtime state and runtime control context when active context is absent
- rejects supplied `job_id` when it does not match active or persisted job context
- reads `missioncontrol.AssessHotUpdateCanarySatisfaction(root, canaryRequirementID)`
- requires configured assessment state
- requires satisfaction state `satisfied` or `waiting_owner_approval`
- requires non-empty selected canary evidence ID
- derives the deterministic authority ID from canary requirement ID and selected canary evidence ID
- reuses existing authority `CreatedAt` when the deterministic authority record already loads
- fails closed if an existing deterministic authority record returns anything other than not-found
- calls the missioncontrol helper with `created_by=operator`
- emits audit on created, selected, and rejected paths
- returns the missioncontrol record and changed flag

## CreatedAt Replay Reuse

Exact replay is byte-stable. The wrapper first derives the deterministic authority ID from the current selected canary evidence. If that authority record already loads successfully, the wrapper passes the existing `CreatedAt` back into the helper.

If the record is absent, the wrapper uses the current TaskState timestamp. If loading fails for any other reason, the command fails closed before attempting creation.

## Response Semantics

Created:

```text
Created hot-update canary satisfaction authority job=<job_id> canary_requirement=<canary_requirement_id> authority=<canary_satisfaction_authority_id> authority_state=<state> owner_approval_required=<bool>.
```

Selected:

```text
Selected hot-update canary satisfaction authority job=<job_id> canary_requirement=<canary_requirement_id> authority=<canary_satisfaction_authority_id> authority_state=<state> owner_approval_required=<bool>.
```

Malformed:

```text
HOT_UPDATE_CANARY_SATISFACTION_AUTHORITY_CREATE requires job_id and canary_requirement_id
```

Failures return the existing direct-command pattern: empty response plus surfaced error.

## Waiting Owner Approval Behavior

For `waiting_owner_approval`, the command creates or selects an authority with:

```text
authority_state=waiting_owner_approval
owner_approval_required=true
satisfaction_state=waiting_owner_approval
```

This records that canary satisfaction exists but does not authorize gate creation. V4-107 does not create owner approval requests or proposal records.

## Audit Behavior

All command paths emit `hot_update_canary_satisfaction_authority_create`:

- created authority: allowed/applied audit
- selected idempotent replay: allowed/applied audit
- rejected fail-closed path: rejected audit with the existing validation code style

## Fail-Closed Behavior

The command fails closed for:

- malformed arguments
- missing mission store root
- missing active/persisted mission context
- wrong `job_id`
- invalid or missing canary requirement
- invalid canary requirement state
- no selected valid passed evidence
- latest selected evidence `failed`, `blocked`, or `expired`
- satisfaction assessment not configured
- satisfaction state other than `satisfied` or `waiting_owner_approval`
- missing candidate result
- missing linked improvement run
- missing linked improvement candidate
- missing or non-linkable eval suite
- missing promotion policy
- missing baseline runtime pack
- missing candidate runtime pack
- copied ref mismatch across requirement, evidence, assessment, candidate result, and fresh eligibility
- stale derived eligibility away from canary-required states
- existing deterministic authority record that fails to load or validate
- divergent duplicate authority records

## Status / Read Model

No separate status command was added. Existing:

```text
STATUS <job_id>
```

surfaces created authorities through:

```json
"hot_update_canary_satisfaction_authority_identity"
```

## Invariants Preserved

V4-107 preserves:

- no owner approval request
- no owner approval proposal record
- no candidate promotion decision for canary-required states
- no broadening of `CandidatePromotionDecisionRecord`
- no hot-update gate for canary-required states
- no canary execution
- no canary evidence creation by this command
- no outcome creation
- no promotion creation
- no rollback creation
- no rollback-apply creation
- no candidate result mutation
- no canary requirement mutation
- no canary evidence mutation
- no canary satisfaction authority mutation after create/select
- no promotion policy mutation
- no runtime pack mutation
- no active runtime-pack pointer mutation
- no last-known-good pointer mutation
- no `reload_generation` mutation
- no pointer-switch behavior change
- no reload/apply behavior change
- no V4-108 work
