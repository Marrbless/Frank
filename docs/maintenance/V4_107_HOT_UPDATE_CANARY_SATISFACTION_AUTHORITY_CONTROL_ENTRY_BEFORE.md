# V4-107 Hot-Update Canary Satisfaction Authority Control Entry Before

## Before-State Gap

V4-105 added the durable `HotUpdateCanarySatisfactionAuthorityRecord` registry and helper:

```go
CreateHotUpdateCanarySatisfactionAuthorityFromRequirement(root, canaryRequirementID, createdBy string, createdAt time.Time)
```

V4-106 assessed the smallest safe operator control entry and selected this command shape:

```text
HOT_UPDATE_CANARY_SATISFACTION_AUTHORITY_CREATE <job_id> <canary_requirement_id>
```

Before V4-107, the helper exists only in `missioncontrol`. There is no direct command and no `TaskState` wrapper binding authority creation to active or persisted job context, operator audit, and replay-stable timestamps.

## Required Control Shape

V4-107 should expose only the existing helper through the existing direct operator command path:

```text
HOT_UPDATE_CANARY_SATISFACTION_AUTHORITY_CREATE <job_id> <canary_requirement_id>
```

The command should call a TaskState wrapper equivalent to:

```go
CreateHotUpdateCanarySatisfactionAuthorityFromRequirement(jobID string, canaryRequirementID string)
```

The wrapper should:

- validate active or persisted job context
- reject a supplied `job_id` that does not match the active/persisted job
- validate the mission store root
- assess canary satisfaction from the requirement
- require `satisfied` or `waiting_owner_approval`
- derive the deterministic authority ID from selected evidence
- reuse existing `CreatedAt` when the deterministic authority already exists
- call the V4-105 missioncontrol helper with `created_by=operator`
- emit audit action `hot_update_canary_satisfaction_authority_create`
- return the created/selected record and changed flag

## Response Semantics

Created:

```text
Created hot-update canary satisfaction authority job=<job_id> canary_requirement=<canary_requirement_id> authority=<canary_satisfaction_authority_id> authority_state=<state> owner_approval_required=<bool>.
```

Selected:

```text
Selected hot-update canary satisfaction authority job=<job_id> canary_requirement=<canary_requirement_id> authority=<canary_satisfaction_authority_id> authority_state=<state> owner_approval_required=<bool>.
```

Malformed command:

```text
HOT_UPDATE_CANARY_SATISFACTION_AUTHORITY_CREATE requires job_id and canary_requirement_id
```

## Waiting Owner Approval Behavior

`waiting_owner_approval` may create or select a durable canary satisfaction authority with:

```text
authority_state=waiting_owner_approval owner_approval_required=true
```

It must not create owner approval requests, owner approval proposal records, candidate promotion decisions, hot-update gates, outcomes, promotions, rollbacks, rollback-apply records, pointer mutations, or reload/apply changes.

## Invariants To Preserve

V4-107 must preserve:

- no owner approval request
- no owner approval proposal record
- no candidate promotion decision for canary-required states
- no broadening of `CandidatePromotionDecisionRecord`
- no hot-update gate for canary-required states
- no canary execution
- no canary evidence creation
- no outcome, promotion, rollback, rollback-apply, or last-known-good creation
- no candidate result mutation
- no canary requirement mutation
- no canary evidence mutation
- no authority mutation after create/select
- no promotion policy mutation
- no runtime pack mutation
- no active runtime-pack pointer mutation
- no last-known-good pointer mutation
- no `reload_generation` mutation
- no pointer-switch behavior change
- no reload/apply behavior change
- no V4-108 work
