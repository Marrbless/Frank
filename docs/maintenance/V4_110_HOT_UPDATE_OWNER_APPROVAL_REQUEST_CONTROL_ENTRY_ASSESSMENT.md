# V4-110 Hot-Update Owner Approval Request Control Entry Assessment

## Scope

V4-110 assesses the smallest safe operator control entry for creating a durable:

```go
HotUpdateOwnerApprovalRequestRecord
```

through the existing direct-command and `TaskState` path.

This is a docs-only slice. It does not change Go code, tests, direct commands, `TaskState` wrappers, owner approval request records, owner approval grants, owner approval rejections, owner approval expiration records, natural-language approval binding, canary satisfaction authority records, candidate promotion decisions, hot-update gates, outcomes, promotions, rollbacks, rollback-apply records, runtime-pack pointers, last-known-good pointers, `reload_generation`, pointer-switch behavior, reload/apply behavior, or V4-111 implementation.

## Live State Inspected

Docs inspected:

- `docs/FRANK_V4_SPEC.md`
- `docs/maintenance/V4_108_HOT_UPDATE_OWNER_APPROVAL_AUTHORITY_PATH_ASSESSMENT.md`
- `docs/maintenance/V4_109_HOT_UPDATE_OWNER_APPROVAL_REQUEST_REGISTRY_AFTER.md`
- `docs/maintenance/V4_106_HOT_UPDATE_CANARY_SATISFACTION_AUTHORITY_CONTROL_ENTRY_ASSESSMENT.md`
- `docs/maintenance/V4_107_HOT_UPDATE_CANARY_SATISFACTION_AUTHORITY_CONTROL_ENTRY_AFTER.md`

Code surfaces inspected:

- `internal/missioncontrol/hot_update_owner_approval_request_registry.go`
- `internal/missioncontrol/hot_update_owner_approval_request_registry_test.go`
- `internal/missioncontrol/hot_update_canary_satisfaction_authority_registry.go`
- `internal/missioncontrol/hot_update_canary_satisfaction.go`
- `internal/missioncontrol/approval.go`
- `internal/missioncontrol/store_records.go`
- `internal/missioncontrol/status.go`
- `internal/missioncontrol/store_project.go`
- `internal/missioncontrol/hot_update_gate_registry.go`
- `internal/missioncontrol/hot_update_outcome_registry.go`
- `internal/missioncontrol/promotion_registry.go`
- `internal/missioncontrol/rollback_registry.go`
- `internal/missioncontrol/rollback_apply_registry.go`
- `internal/agent/loop.go`
- `internal/agent/tools/taskstate.go`
- `internal/agent/tools/taskstate_readout.go`
- `internal/agent/loop_processdirect_test.go`

## Current Request Registry Surface

V4-109 implemented the durable helper:

```go
CreateHotUpdateOwnerApprovalRequestFromCanarySatisfactionAuthority(root, canarySatisfactionAuthorityID, createdBy string, createdAt time.Time) (HotUpdateOwnerApprovalRequestRecord, bool, error)
```

The helper is intentionally store-root-only. It accepts only:

- `canary_satisfaction_authority_id`
- `created_by`
- `created_at`

It loads the committed `HotUpdateCanarySatisfactionAuthorityRecord`, requires:

- `state=waiting_owner_approval`
- `owner_approval_required=true`
- `satisfaction_state=waiting_owner_approval`
- non-empty `selected_canary_evidence_id`

and then cross-checks the authority, canary requirement, selected passed canary evidence, fresh canary satisfaction assessment, candidate result, improvement run, improvement candidate, frozen eval suite, promotion policy, baseline runtime pack, candidate runtime pack, and freshly derived candidate promotion eligibility.

Fresh canary satisfaction must remain:

```text
waiting_owner_approval
```

Fresh eligibility must remain:

```text
canary_and_owner_approval_required
```

The deterministic request ID is:

```text
hot-update-owner-approval-request-<canary_satisfaction_authority_id>
```

The record is immutable once written. Exact replay returns the existing record with `changed=false`; divergent duplicates fail closed.

## Existing Control Path Pattern

Existing V4 direct commands use explicit durable-object names:

- `HOT_UPDATE_CANARY_REQUIREMENT_CREATE <job_id> <result_id>`
- `HOT_UPDATE_CANARY_EVIDENCE_CREATE <job_id> <canary_requirement_id> <evidence_state> <observed_at> [reason]`
- `HOT_UPDATE_CANARY_SATISFACTION_AUTHORITY_CREATE <job_id> <canary_requirement_id>`
- `HOT_UPDATE_GATE_FROM_DECISION <job_id> <promotion_decision_id>`
- `HOT_UPDATE_OUTCOME_CREATE <job_id> <hot_update_id>`
- `HOT_UPDATE_PROMOTION_CREATE <job_id> <outcome_id>`
- `HOT_UPDATE_LKG_RECERTIFY <job_id> <promotion_id>`

The adjacent `TaskState` wrappers all:

- validate the mission store root
- require active or persisted mission control context
- reject a supplied `job_id` that does not match active/persisted job state
- derive `created_by` as `operator`
- derive timestamps from `taskStateTransitionTimestamp(taskStateNowUTC())`
- reuse existing creation timestamps when deterministic replay needs byte-stable idempotence
- emit an operator audit action through `emitRuntimeControlAuditEvent(...)`
- return an empty direct-command response plus error on failure

The existing runtime `ApprovalRequestRecord` and `ApprovalGrantRecord` surfaces remain useful for naming/UX reference only. They are job/runtime-step scoped, projected from `JobRuntimeState`, hydrated into runtime control status, and shown through `approval_history`. They should not be reused as the hot-update owner approval request authority and should not be updated by V4-111.

## Command Shape Decision

V4-111 should expose owner approval request creation as a direct operator command now.

The exact command should be:

```text
HOT_UPDATE_OWNER_APPROVAL_REQUEST_CREATE <job_id> <canary_satisfaction_authority_id>
```

This is preferred over:

```text
HOT_UPDATE_OWNER_APPROVAL_CREATE <job_id> <canary_satisfaction_authority_id>
```

because the implemented durable surface is a request record, not a grant, rejection, expiration, or generic approval decision. The longer command follows the repo's current V4 pattern of preserving the durable object role in the command name, as seen with `HOT_UPDATE_CANARY_SATISFACTION_AUTHORITY_CREATE` and the registry identity name `hot_update_owner_approval_request_identity`.

The source authority argument should be:

```text
canary_satisfaction_authority_id
```

It should not be `owner_approval_request_id`, because that ID is deterministic and derived from the authority ID. Accepting it would duplicate source selection and create a replay footgun.

It should not be `canary_requirement_id`, because a requirement can have multiple canary evidence records and multiple canary satisfaction authority records over time. V4-109 deliberately made the request bind to the committed `waiting_owner_approval` canary satisfaction authority and copy its selected evidence and candidate refs.

The command should still require `job_id` even though the missioncontrol helper is store-root-only. The direct-command surface is governed operator control, so it must remain bound to active or persisted job context and audit history.

Malformed command behavior should mirror nearby commands:

```text
HOT_UPDATE_OWNER_APPROVAL_REQUEST_CREATE requires job_id and canary_satisfaction_authority_id
```

## TaskState Wrapper Decision

V4-111 should add exactly this wrapper:

```go
func (s *TaskState) CreateHotUpdateOwnerApprovalRequestFromCanarySatisfactionAuthority(jobID string, canarySatisfactionAuthorityID string) (missioncontrol.HotUpdateOwnerApprovalRequestRecord, bool, error)
```

Wrapper behavior should be:

1. Return the zero record, `false`, and `nil` when `TaskState` is nil, matching nearby wrappers.

2. Set:

   ```go
   const action = "hot_update_owner_approval_request_create"
   ```

3. Derive:

   ```go
   now := taskStateTransitionTimestamp(taskStateNowUTC())
   ```

4. Snapshot the same control context used by existing V4 wrappers:

   - `executionContext`
   - `runtimeControl`
   - `runtimeState`
   - `missionStoreRoot`

5. Validate `missionStoreRoot` with `missioncontrol.ValidateStoreRoot(root)`.

6. Validate active or persisted job context using the existing wrapper pattern:

   - if an active execution context exists, require non-nil job, step, and runtime
   - reject when `ec.Job.ID != jobID`
   - otherwise require persisted runtime state
   - reject when `runtimeState.JobID != jobID`
   - require persisted runtime control context when active context is absent
   - reject when `control.JobID != jobID`

7. Normalize and validate `canarySatisfactionAuthorityID` by deriving the deterministic request ID:

   ```go
   requestID := missioncontrol.HotUpdateOwnerApprovalRequestIDFromCanarySatisfactionAuthority(canarySatisfactionAuthorityID)
   ```

   The missioncontrol helper should remain the source authority validator. The wrapper only needs the deterministic request ID for replay timestamp reuse.

8. Set `createdAt := now`. If:

   ```go
   missioncontrol.LoadHotUpdateOwnerApprovalRequestRecord(root, requestID)
   ```

   succeeds, reuse `existing.CreatedAt`. If loading returns `ErrHotUpdateOwnerApprovalRequestRecordNotFound`, continue with `now`. If loading returns any other error, fail closed.

9. Call:

   ```go
   record, changed, err := missioncontrol.CreateHotUpdateOwnerApprovalRequestFromCanarySatisfactionAuthority(root, canarySatisfactionAuthorityID, "operator", createdAt)
   ```

10. Emit audit action `hot_update_owner_approval_request_create` with applied/rejected result through `emitRuntimeControlAuditEvent(...)`.

11. Return the `record` and `changed` flag from the missioncontrol helper.

`created_by` should always be `operator`. `created_at` should come from the existing `TaskState` timestamp path and must reuse an existing request record's `CreatedAt` for exact replay stability.

## Response Semantics

On success, the direct command should include `request_state`, `authority_state`, and `owner_approval_required`.

Created response:

```text
Created hot-update owner approval request job=<job_id> canary_satisfaction_authority=<canary_satisfaction_authority_id> owner_approval_request=<owner_approval_request_id> request_state=requested authority_state=waiting_owner_approval owner_approval_required=true.
```

Idempotent selected response:

```text
Selected hot-update owner approval request job=<job_id> canary_satisfaction_authority=<canary_satisfaction_authority_id> owner_approval_request=<owner_approval_request_id> request_state=requested authority_state=waiting_owner_approval owner_approval_required=true.
```

Failure response should remain the existing direct-command pattern:

```text
<empty response> + error
```

`STATUS <job_id>` should continue to surface the request through:

```json
"hot_update_owner_approval_request_identity"
```

No separate status command is needed.

## Audit Behavior

V4-111 should emit:

```text
hot_update_owner_approval_request_create
```

for created, selected, and rejected paths.

Allowed created and selected paths should emit applied audit events. Rejected paths should emit rejected audit events with the existing validation code style from `emitRuntimeControlAuditEvent(...)`.

This audit action is separate from runtime `approval_history`. The command creates a durable hot-update request record; it does not create or resolve a runtime `ApprovalRequestRecord` or `ApprovalGrantRecord`.

## Fail-Closed Behavior

Wrong `job_id` should fail before helper invocation with the existing validation style:

```text
operator command does not match the active job
```

and `RejectionCodeStepValidationFailed`.

Missing persisted context should fail with the existing style:

```text
operator command requires an active mission step
operator command requires persisted mission control context
```

Missing, invalid, or stale canary satisfaction authority should fail through the missioncontrol helper and be audited as rejected. That includes:

- missing mission store root
- malformed `canary_satisfaction_authority_id`
- missing authority record
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

The wrapper should not catch and soften these helper failures. Existing invalid/stale records must remain visible in the read-model and must block the command rather than being overwritten.

## Explicit Non-Goals For V4-111

V4-111 should not create approval grants, rejections, or expiration records.

V4-111 should not bind natural-language owner approval. A plain `yes` or `no` should continue to bind only to the existing runtime approval path when exactly one runtime approval request is in scope.

V4-111 should not create candidate promotion decisions, broaden `CandidatePromotionDecisionRecord`, create hot-update gates, execute canaries, create canary evidence, create outcomes, create promotions, create rollbacks, create rollback-apply records, mutate candidate results, mutate canary requirements, mutate canary evidence, mutate canary satisfaction authority records, mutate owner approval request records after create/select, mutate promotion policies, mutate runtime packs, mutate active runtime-pack pointers, mutate last-known-good pointers, mutate `reload_generation`, or change pointer-switch/reload/apply behavior.

## Assessment Answers

- Owner approval request creation should be exposed as a direct operator command now because V4-109 added the durable registry/read-model and the next missing path is governed operator creation through the existing control surface.
- The command should use `canary_satisfaction_authority_id` as source authority.
- The command should require `job_id` even though the missioncontrol helper is store-root-only, because direct commands are governed by active or persisted job context and audit.
- The exact `TaskState` wrapper should be `CreateHotUpdateOwnerApprovalRequestFromCanarySatisfactionAuthority(jobID, canarySatisfactionAuthorityID string)`.
- `created_by` should be `operator`; `created_at` should be derived from `taskStateTransitionTimestamp(taskStateNowUTC())`.
- Exact replay should derive the deterministic request ID from the authority ID, load the existing request, and reuse its `CreatedAt` before calling the missioncontrol helper.
- The audit action should be `hot_update_owner_approval_request_create`.
- Created and selected responses should return the created/selected request ID and include `request_state=requested`, `authority_state=waiting_owner_approval`, and `owner_approval_required=true`.
- V4-111 should not create approval grants or rejections.
- V4-111 should not bind natural-language owner approval.
- Wrong job ID should fail closed before helper invocation using the existing active/persisted job mismatch validation.
- Missing, invalid, or stale canary satisfaction authority should fail closed through the existing missioncontrol helper and surface an empty direct-command response plus error.
- The existing runtime approval records are not the source authority for this command and should not be written by V4-111.

## Recommended V4-111 Slice

Implement exactly:

```text
V4-111 - Hot-Update Owner Approval Request Control Entry
```

V4-111 should add:

- direct command parser and malformed-command handling for `HOT_UPDATE_OWNER_APPROVAL_REQUEST_CREATE <job_id> <canary_satisfaction_authority_id>`
- `TaskState.CreateHotUpdateOwnerApprovalRequestFromCanarySatisfactionAuthority(...)`
- audit action `hot_update_owner_approval_request_create`
- created/selected direct-command responses with request, authority, request state, authority state, and owner approval required fields
- focused tests for create, select/replay stability, status readout, malformed arguments, wrong job ID, missing/invalid/stale authority, stale satisfaction, stale eligibility, invalid existing deterministic request, divergent duplicates, and no downstream side effects
- V4-111 before/after maintenance docs

V4-111 should stop there. The next branch after V4-111 can assess owner approval grant/rejection authority, but V4-111 should create only the request control entry.

## Invariants Preserved

The recommended V4-111 path preserves:

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
- no V4-111 implementation in this slice
