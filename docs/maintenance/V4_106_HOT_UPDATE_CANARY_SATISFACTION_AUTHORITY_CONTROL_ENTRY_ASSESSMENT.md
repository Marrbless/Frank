# V4-106 Hot-Update Canary Satisfaction Authority Control Entry Assessment

## Scope

V4-106 assesses the smallest safe operator control entry for creating a durable:

```go
HotUpdateCanarySatisfactionAuthorityRecord
```

through the existing direct-command and `TaskState` path.

This is a docs-only slice. It does not change Go code, tests, direct commands, TaskState wrappers, canary requirements, canary evidence, canary satisfaction logic, authority records, owner approval records, candidate promotion decisions, hot-update gates, outcomes, promotions, rollbacks, rollback-apply records, runtime-pack pointers, last-known-good pointers, `reload_generation`, pointer-switch behavior, reload/apply behavior, or V4-107 implementation.

## Live State Inspected

Docs inspected:

- `docs/FRANK_V4_SPEC.md`
- `docs/maintenance/V4_104_CANARY_SATISFACTION_AUTHORITY_PATH_ASSESSMENT.md`
- `docs/maintenance/V4_105_HOT_UPDATE_CANARY_SATISFACTION_AUTHORITY_REGISTRY_AFTER.md`
- `docs/HOT_UPDATE_OPERATOR_RUNBOOK.md`

Code surfaces inspected:

- `internal/missioncontrol/hot_update_canary_satisfaction_authority_registry.go`
- `internal/missioncontrol/hot_update_canary_satisfaction.go`
- `internal/missioncontrol/hot_update_canary_requirement_registry.go`
- `internal/missioncontrol/hot_update_canary_evidence_registry.go`
- `internal/missioncontrol/status.go`
- `internal/missioncontrol/store_project.go`
- `internal/missioncontrol/candidate_promotion_decision_registry.go`
- `internal/missioncontrol/hot_update_gate_registry.go`
- `internal/missioncontrol/hot_update_outcome_registry.go`
- `internal/missioncontrol/promotion_registry.go`
- `internal/missioncontrol/rollback_registry.go`
- `internal/missioncontrol/rollback_apply_registry.go`
- `internal/missioncontrol/audit.go`
- `internal/missioncontrol/store_records.go`
- `internal/agent/loop.go`
- `internal/agent/tools/taskstate.go`
- `internal/agent/loop_processdirect_test.go`

## Current Authority Surface

V4-105 implemented the durable authority helper:

```go
CreateHotUpdateCanarySatisfactionAuthorityFromRequirement(root, canaryRequirementID, createdBy string, createdAt time.Time) (HotUpdateCanarySatisfactionAuthorityRecord, bool, error)
```

The helper is intentionally store-root-only. It accepts only a canary requirement ID plus operator metadata, calls:

```go
AssessHotUpdateCanarySatisfaction(root, canaryRequirementID)
```

and creates authority only when the assessment is:

- `satisfied`, producing `state=authorized`
- `waiting_owner_approval`, producing `state=waiting_owner_approval`

The selected evidence is not caller-supplied. It is selected by the read-only satisfaction helper as the newest valid matching evidence by `observed_at`, with `canary_evidence_id` as the tie-breaker. The selected evidence must be `passed`.

The deterministic authority ID is:

```text
hot-update-canary-satisfaction-authority-<canary_requirement_id>-<selected_canary_evidence_id>
```

The record is immutable once written. Exact replay returns the existing record with `changed=false`; divergent duplicates fail closed.

## Existing Control Path Pattern

Existing V4 direct commands use explicit durable-object names:

- `HOT_UPDATE_CANARY_REQUIREMENT_CREATE <job_id> <result_id>`
- `HOT_UPDATE_CANARY_EVIDENCE_CREATE <job_id> <canary_requirement_id> <evidence_state> <observed_at> [reason]`
- `HOT_UPDATE_GATE_FROM_DECISION <job_id> <promotion_decision_id>`
- `HOT_UPDATE_OUTCOME_CREATE <job_id> <hot_update_id>`
- `HOT_UPDATE_PROMOTION_CREATE <job_id> <outcome_id>`
- `HOT_UPDATE_LKG_RECERTIFY <job_id> <promotion_id>`

The `TaskState` wrappers all:

- validate the mission store root
- require active or persisted mission control context
- reject a supplied `job_id` that does not match active/persisted job state
- derive `created_by` as `operator`
- derive timestamps from `taskStateTransitionTimestamp(taskStateNowUTC())`
- reuse existing creation timestamps when deterministic replay needs byte-stable idempotence
- emit an operator audit action through `emitRuntimeControlAuditEvent(...)`
- return an empty direct-command response plus error on failure

## Command Shape Decision

V4-107 should expose authority creation as a direct operator command now.

The exact command should be:

```text
HOT_UPDATE_CANARY_SATISFACTION_AUTHORITY_CREATE <job_id> <canary_requirement_id>
```

This is preferred over:

```text
HOT_UPDATE_CANARY_AUTHORITY_CREATE <job_id> <canary_requirement_id>
```

because the repo's V4 command names preserve the durable object role. The durable type and read-model identity are specifically canary **satisfaction** authority, not a generic canary authority. Shortening the command would make future canary authority surfaces harder to distinguish.

The source authority argument should be `canary_requirement_id`, not selected canary evidence ID and not authority ID.

Reasons:

- the missioncontrol helper already accepts `canaryRequirementID`
- `AssessHotUpdateCanarySatisfaction(...)` owns selected-evidence choice
- caller-supplied evidence ID would duplicate and potentially conflict with the freshness/tie-break selection rule
- caller-supplied authority ID would duplicate deterministic ID derivation and create an unnecessary replay footgun

The command should still require `job_id` even though the missioncontrol helper is store-root-only. The direct-command surface is a governed operator-control path, so it must remain bound to active or persisted job context and audit history.

Malformed command behavior should mirror nearby commands:

```text
HOT_UPDATE_CANARY_SATISFACTION_AUTHORITY_CREATE requires job_id and canary_requirement_id
```

## TaskState Wrapper Decision

V4-107 should add exactly this wrapper:

```go
func (s *TaskState) CreateHotUpdateCanarySatisfactionAuthorityFromRequirement(jobID string, canaryRequirementID string) (missioncontrol.HotUpdateCanarySatisfactionAuthorityRecord, bool, error)
```

Wrapper behavior should be:

1. Set:

   ```go
   const action = "hot_update_canary_satisfaction_authority_create"
   ```

2. Derive:

   ```go
   now := taskStateTransitionTimestamp(taskStateNowUTC())
   ```

3. Snapshot the same control context used by the existing canary requirement/evidence wrappers:

   - `executionContext`
   - `runtimeControl`
   - `runtimeState`
   - `missionStoreRoot`

4. Validate `missionStoreRoot` with `missioncontrol.ValidateStoreRoot(root)`.

5. Validate active or persisted job context using the existing wrapper pattern:

   - if an active execution context exists, require non-nil job, step, and runtime
   - reject when `ec.Job.ID != jobID`
   - otherwise require persisted runtime state
   - reject when `runtimeState.JobID != jobID`
   - require persisted runtime control context when active step state requires it
   - reject when `control.JobID != jobID`

6. Before calling the create helper, derive the deterministic authority ID from the current selected evidence:

   ```go
   assessment, err := missioncontrol.AssessHotUpdateCanarySatisfaction(root, canaryRequirementID)
   ```

   Require `assessment.State == "configured"`, `assessment.SatisfactionState` in `satisfied` or `waiting_owner_approval`, and non-empty `assessment.SelectedCanaryEvidenceID`. If not, fail closed and emit the audit rejection.

7. Compute:

   ```go
   authorityID := missioncontrol.HotUpdateCanarySatisfactionAuthorityIDFromRequirementEvidence(
       assessment.CanaryRequirementID,
       assessment.SelectedCanaryEvidenceID,
   )
   ```

8. Set `createdAt := now`. If `missioncontrol.LoadHotUpdateCanarySatisfactionAuthorityRecord(root, authorityID)` succeeds, reuse `existing.CreatedAt`. If loading returns `ErrHotUpdateCanarySatisfactionAuthorityRecordNotFound`, continue with `now`. If loading returns any other error, fail closed.

9. Call:

   ```go
   record, changed, err := missioncontrol.CreateHotUpdateCanarySatisfactionAuthorityFromRequirement(root, canaryRequirementID, "operator", createdAt)
   ```

10. Emit audit action `hot_update_canary_satisfaction_authority_create` with applied/rejected result through `emitRuntimeControlAuditEvent(...)`.

11. Return the `record` and `changed` flag from the missioncontrol helper.

`created_by` should always be `operator`. `created_at` should come from the existing TaskState timestamp path and must reuse an existing record's `CreatedAt` for exact replay stability.

## Response Semantics

On success, the direct command should include `owner_approval_required=<bool>` and `authority_state=<state>`.

Created response:

```text
Created hot-update canary satisfaction authority job=<job_id> canary_requirement=<canary_requirement_id> authority=<canary_satisfaction_authority_id> authority_state=<state> owner_approval_required=<bool>.
```

Idempotent selected response:

```text
Selected hot-update canary satisfaction authority job=<job_id> canary_requirement=<canary_requirement_id> authority=<canary_satisfaction_authority_id> authority_state=<state> owner_approval_required=<bool>.
```

Failure response should remain the existing direct-command pattern:

```text
<empty response> + error
```

## Owner Approval Assessment

`waiting_owner_approval` should allow creation of the canary satisfaction authority record, but it must not authorize a hot-update gate.

V4-107 should not create owner approval requests.

Reasons:

- the V4-105 authority record already represents the canary-satisfied fact separately from gate authority
- existing approval records are job/runtime approval records, not canary-specific proposal records with copied requirement/evidence/result/policy refs
- creating approval requests would widen V4-107 from "control entry for authority creation" into a new approval workflow
- V4-104 already separated the future sequence into authority first, then owner approval proposal/request, then specialized canary gate helper

For `state=waiting_owner_approval`, the command should return the created/selected authority record with:

```text
authority_state=waiting_owner_approval owner_approval_required=true
```

No approval request, hot-update gate, outcome, promotion, rollback, or pointer mutation should occur.

## Fail-Closed Behavior

Wrong job ID should fail before helper invocation with the existing validation style:

```text
operator command does not match the active job
```

and `RejectionCodeStepValidationFailed`.

Missing persisted context should fail with the existing style:

```text
operator command requires an active mission step
operator command requires persisted mission control context
```

Missing, invalid, or stale canary requirement/evidence/satisfaction should fail closed through the wrapper precheck or the missioncontrol helper. V4-107 should fail closed for:

- invalid store root
- no active/persisted job context
- supplied `job_id` not matching active/persisted job
- invalid `canary_requirement_id`
- missing canary requirement
- invalid canary requirement
- canary requirement state not `required`
- stale derived eligibility away from `canary_required` or `canary_and_owner_approval_required`
- no valid selected canary evidence
- selected evidence not `passed`
- latest selected evidence is `failed`, `blocked`, or `expired`
- satisfaction assessment state not `configured`
- satisfaction state not `satisfied` or `waiting_owner_approval`
- missing candidate result
- missing linked improvement run
- missing linked improvement candidate
- missing or non-linkable eval suite
- missing promotion policy
- missing baseline runtime pack
- missing candidate runtime pack
- copied ref mismatch across requirement, evidence, assessment, candidate result, and fresh eligibility
- existing deterministic authority record that fails to load or validate
- divergent duplicate authority record
- any attempted downstream gate authorization when `authority_state=waiting_owner_approval`

The following must remain fail-closed and out of scope for V4-107:

- owner approval request/proposal creation
- candidate promotion decisions for canary-required states
- hot-update gates for canary-required states
- canary execution
- canary evidence creation
- outcomes
- promotions
- rollbacks
- rollback-apply records
- candidate result mutation
- canary requirement mutation
- canary evidence mutation
- promotion policy mutation
- runtime pack mutation
- active runtime-pack pointer mutation
- last-known-good pointer mutation
- `reload_generation` mutation
- pointer-switch behavior change
- reload/apply behavior change

## Recommended V4-107 Slice

Recommend exactly one V4-107 implementation slice:

```text
V4-107 - Hot-Update Canary Satisfaction Authority Direct Command
```

V4-107 should implement only:

- `hotUpdateCanarySatisfactionAuthorityCreateCommandRE`
- malformed command detection for `HOT_UPDATE_CANARY_SATISFACTION_AUTHORITY_CREATE`
- `AgentLoop.processOperatorCommand(...)` handling for:

  ```text
  HOT_UPDATE_CANARY_SATISFACTION_AUTHORITY_CREATE <job_id> <canary_requirement_id>
  ```

- `TaskState.CreateHotUpdateCanarySatisfactionAuthorityFromRequirement(jobID, canaryRequirementID string) (...)`
- focused process-direct tests for created, selected replay, malformed arguments, wrong job ID, invalid/missing/stale source state, `waiting_owner_approval`, audit action, read-model visibility, replay `created_at` stability, and absence of downstream records

V4-107 should not implement owner approval requests or any gate/promotion/rollback path.
