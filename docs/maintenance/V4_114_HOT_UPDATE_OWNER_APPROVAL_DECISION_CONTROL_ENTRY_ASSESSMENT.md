# V4-114 - Hot-Update Owner Approval Decision Control Entry Assessment

## Scope

V4-114 assessed the smallest safe governed operator control surface for creating terminal hot-update owner approval decisions after V4-113 added the durable missioncontrol decision registry and read model.

This slice is assessment-only. It does not add Go code, tests, direct commands, TaskState wrappers, natural-language approval binding, hot-update gates, candidate promotion decisions, outcomes, promotions, rollbacks, last-known-good records, pointer mutations, reload/apply behavior, or V4-115 work.

## Live State Inspected

- `docs/FRANK_V4_SPEC.md`
- `docs/maintenance/V4_112_HOT_UPDATE_OWNER_APPROVAL_GRANT_REJECTION_AUTHORITY_ASSESSMENT.md`
- `docs/maintenance/V4_113_HOT_UPDATE_OWNER_APPROVAL_DECISION_REGISTRY_AFTER.md`
- `internal/missioncontrol/hot_update_owner_approval_decision_registry.go`
- `internal/missioncontrol/hot_update_owner_approval_request_registry.go`
- `internal/missioncontrol/status.go`
- `internal/missioncontrol/store_project.go`
- `internal/agent/tools/taskstate_readout.go`
- `internal/agent/loop.go`
- `internal/agent/tools/taskstate.go`
- `internal/missioncontrol/store_records.go`
- `internal/missioncontrol/hot_update_gate_registry.go`

## Current Authority Surface

V4-113 introduced one terminal owner approval decision record:

```go
type HotUpdateOwnerApprovalDecisionRecord struct {
    RecordVersion                 int
    OwnerApprovalDecisionID       string
    OwnerApprovalRequestID        string
    CanarySatisfactionAuthorityID string
    CanaryRequirementID           string
    SelectedCanaryEvidenceID      string
    ResultID                      string
    RunID                         string
    CandidateID                   string
    EvalSuiteID                   string
    PromotionPolicyID             string
    BaselinePackID                string
    CandidatePackID               string
    RequestState                  HotUpdateOwnerApprovalRequestState
    AuthorityState                HotUpdateCanarySatisfactionAuthorityState
    SatisfactionState             HotUpdateCanarySatisfactionState
    OwnerApprovalRequired         bool
    Decision                      HotUpdateOwnerApprovalDecision
    Reason                        string
    DecidedAt                     time.Time
    DecidedBy                     string
}
```

The only valid terminal decision values are `granted` and `rejected`. The deterministic public ID is:

```text
hot-update-owner-approval-decision-<owner_approval_request_id>
```

The physical storage path uses the existing V4-113 hashed directory convention:

```text
runtime_packs/hot_update_owner_approval_decisions/<sha256(owner_approval_decision_id)>/record.json
```

`CreateHotUpdateOwnerApprovalDecisionFromRequest(root, ownerApprovalRequestID, decision, decidedBy, decidedAt, reason)` is the source helper. It validates the store root, request ID, decision, `decided_by`, `decided_at`, and reason; loads the committed `HotUpdateOwnerApprovalRequestRecord`; requires `request_state=requested`; copies source refs from the request; revalidates the request and linked canary/candidate authority chain through store validation; and stores the immutable decision. Exact replay returns `changed=false`; divergent duplicates and any second terminal decision for the same request fail closed.

The read model already exposes `hot_update_owner_approval_decision_identity` in committed mission status snapshots and active `STATUS <job_id>` readout. Invalid decision records are surfaced without hiding valid records.

## Runtime Approval Records

Existing `ApprovalRequestRecord` and `ApprovalGrantRecord` remain job/runtime-step approval history records. They require job ID, step ID, sequence, requested action, scope, runtime approval state, and runtime approval timestamps. They do not carry stable hot-update owner approval request linkage, canary satisfaction authority linkage, selected canary evidence linkage, or candidate result lineage.

They are useful for naming and operator UX only. They must not become canonical hot-update owner approval authority.

## Natural-Language Approval Handling

Natural-language `yes`/`no` handling currently flows through `ApplyNaturalApprovalDecision`, `ParsePlainApprovalDecision`, `ResolveSinglePendingApprovalRequest`, and `ApplyApprovalDecision`. That path binds plain operator replies only to an ordinary pending runtime approval request when exactly one eligible pending runtime request is in scope.

Hot-update owner approval decisions are a separate durable authority surface with a different source ID and cross-check chain. V4-115 must not bind natural-language `yes`/`no` to hot-update owner approval decisions. That binding should remain deferred until the durable direct command has shipped and a later assessment proves unambiguous scoping against both runtime approvals and hot-update owner approval requests.

## Command Shape Decision

V4-115 should add exactly one direct command:

```text
HOT_UPDATE_OWNER_APPROVAL_DECISION_CREATE <job_id> <owner_approval_request_id> <decision> [reason...]
```

The command should use `owner_approval_request_id` as the source authority argument.

Reasons:

- The V4-113 registry consumes `owner_approval_request_id` and derives the decision ID deterministically.
- `owner_approval_decision_id` is derived output, not source authority.
- `canary_satisfaction_authority_id` is one step too early; V4-109/V4-111 intentionally inserted the owner approval request as the durable handoff surface.
- A single command with `decision=granted|rejected` matches the single decision registry. Separate `HOT_UPDATE_OWNER_APPROVAL_GRANT` and `HOT_UPDATE_OWNER_APPROVAL_REJECT` commands would imply separate terminal surfaces and duplicate parsing/audit behavior that the V4-112 assessment rejected.

The command must require `job_id`, even though the missioncontrol helper is store-root-only, because direct operator commands are governed through active or persisted TaskState context, audit emission, and job binding.

The decision argument must be exactly `granted` or `rejected`. Aliases such as `approve`, `approved`, `deny`, `yes`, and `no` should fail closed in V4-115.

The optional reason should be trimmed. When omitted, the wrapper should use a deterministic default:

```text
hot-update owner approval decision granted
hot-update owner approval decision rejected
```

## TaskState Wrapper Decision

V4-115 should add a TaskState wrapper equivalent to:

```go
func (s *TaskState) CreateHotUpdateOwnerApprovalDecisionFromRequest(
    jobID string,
    ownerApprovalRequestID string,
    decision missioncontrol.HotUpdateOwnerApprovalDecision,
    reason string,
) (missioncontrol.HotUpdateOwnerApprovalDecisionRecord, bool, error)
```

Recommended behavior:

1. Return zero record, `false`, `nil` when `s == nil`, matching local wrapper convention.
2. Use audit action `hot_update_owner_approval_decision_create`.
3. Derive `now` from `taskStateTransitionTimestamp(taskStateNowUTC())`.
4. Clone execution context, runtime control context, runtime state, and mission store root under lock.
5. Build the audit execution context from the active execution context when present; otherwise use the persisted runtime/control audit context.
6. Validate the mission store root with `missioncontrol.ValidateStoreRoot`.
7. If active execution context is present, require job, step, and runtime; reject when supplied `job_id` does not match the active job.
8. Otherwise require persisted runtime state and persisted runtime control context; reject when supplied `job_id` does not match either persisted job binding.
9. Normalize/validate `owner_approval_request_id` through the missioncontrol ref helper before deriving the decision ID.
10. Require `decision` to be exactly `granted` or `rejected`.
11. Trim reason and substitute the deterministic decision-specific default only when reason is omitted.
12. Derive `owner_approval_decision_id` with `missioncontrol.HotUpdateOwnerApprovalDecisionIDFromRequest(ownerApprovalRequestID)`.
13. Set `decidedAt := now`.
14. If the deterministic decision record loads successfully, reuse `existing.DecidedAt` for replay stability.
15. If loading returns `ErrHotUpdateOwnerApprovalDecisionRecordNotFound`, continue with `decidedAt := now`.
16. If loading returns any other error, fail closed and emit rejected audit.
17. Call `missioncontrol.CreateHotUpdateOwnerApprovalDecisionFromRequest(root, ownerApprovalRequestID, decision, "operator", decidedAt, reason)`.
18. Emit `hot_update_owner_approval_decision_create` audit on created, selected, and failure paths.
19. Return the missioncontrol record and `changed` flag.

Exact replay should mean the same normalized request ID, decision, reason, decided_by, and reused decided_at. If an existing decision was created with a custom reason, replaying without that same reason should fail closed as a divergent duplicate rather than silently changing or broadening selection semantics.

## Direct Command Response

Malformed or missing arguments should return an empty response plus an argument error. Recommended error:

```text
HOT_UPDATE_OWNER_APPROVAL_DECISION_CREATE requires job_id, owner_approval_request_id, decision, and optional reason
```

Invalid decision values that parse as the third argument should fail closed through the wrapper/helper with the missioncontrol invalid decision error.

Created response:

```text
Created hot-update owner approval decision job=<job_id> owner_approval_request=<owner_approval_request_id> owner_approval_decision=<owner_approval_decision_id> decision=<decision>.
```

Selected response:

```text
Selected hot-update owner approval decision job=<job_id> owner_approval_request=<owner_approval_request_id> owner_approval_decision=<owner_approval_decision_id> decision=<decision>.
```

The response should include `decision=<granted|rejected>` because a single command handles both terminal outcomes.

## Audit Behavior

The audit action should be:

```text
hot_update_owner_approval_decision_create
```

Audit should be emitted on:

- created path with `Allowed=true`
- selected/idempotent path with `Allowed=true`
- failure path with `Allowed=false`

Failure audit should preserve existing `emitRuntimeControlAuditEvent` behavior: validation errors carry their rejection code, and non-validation failures default to invalid runtime state.

## Wrong Job And Stale Source Failures

Wrong job ID should fail using the existing operator command style:

```text
operator command does not match the active job
```

with `RejectionCodeStepValidationFailed` where the active or persisted context exists but the job binding differs.

Missing active execution context plus missing persisted runtime context should fail with:

```text
operator command requires an active mission step
```

Missing persisted mission control context in the persisted path should fail with:

```text
operator command requires persisted mission control context
```

Missing, invalid, stale, or mismatched owner approval request authority should fail closed through the V4-113 missioncontrol helper and its validation/linkage chain.

## Downstream Gate Sequencing

`HotUpdateGateRecord` already has `canary_ref` and `approval_ref` fields, and gate decisions include `apply_canary`, `require_approval`, and other gate states such as `canarying`. Existing gate creation from candidate promotion decisions still requires eligible-only candidate promotion authority. V4-115 must not create hot-update gates and must not populate `approval_ref`.

The downstream sequence should remain:

1. Create terminal owner approval decision through V4-115.
2. Assess a specialized hot-update gate authority path that can consume a granted owner approval decision as `approval_ref` and the canary satisfaction authority/evidence as `canary_ref`.
3. Keep rejected decisions terminal blockers.

Owner approval must never substitute for passed canary evidence or canary satisfaction.

## Assessment Answers

- Terminal owner approval decision creation should be exposed as a direct operator command in V4-115 because the durable registry/read model exists and the next missing surface is governed operator entry.
- The command should use `owner_approval_request_id` as source authority.
- The command should carry `decision=granted|rejected` as an argument instead of separate grant/reject commands.
- The command should require `job_id` for governed TaskState validation and audit binding.
- `decided_by` should be `"operator"`.
- `decided_at` should come from the TaskState timestamp path and reuse the existing deterministic decision record's `DecidedAt` on replay.
- The audit action should be `hot_update_owner_approval_decision_create`.
- Created vs selected responses should include job ID, owner approval request ID, owner approval decision ID, and decision.
- V4-115 should not bind natural-language `yes`/`no`.
- V4-115 should not create hot-update gates for granted decisions.
- Wrong job ID should fail through existing TaskState job mismatch validation and return an empty direct-command response plus error.
- Missing, invalid, or stale owner approval requests should fail closed through missioncontrol helper validation and be audited as rejected.

## Must Remain Fail-Closed

V4-115 must fail closed for:

- malformed command arguments
- missing mission store root
- missing active execution context and missing persisted runtime context
- missing persisted runtime control context on the persisted path
- wrong job ID
- missing or invalid `owner_approval_request_id`
- invalid decision value
- missing or invalid existing deterministic decision record
- existing deterministic decision load errors other than not found
- divergent duplicate decision record
- any second terminal decision for the same request
- different decision for a request that already has a terminal decision
- missing or invalid owner approval request
- owner approval request state other than `requested`
- authority state other than `waiting_owner_approval`
- `owner_approval_required=false`
- satisfaction state other than `waiting_owner_approval`
- missing, non-passed, stale, or mismatched selected canary evidence
- missing or mismatched canary requirement
- missing or mismatched canary satisfaction authority
- missing candidate result, run, candidate, eval suite, promotion policy, baseline pack, or candidate pack
- fresh canary satisfaction away from `waiting_owner_approval`
- fresh eligibility away from `canary_and_owner_approval_required`
- copied ref mismatch across request, authority, requirement, evidence, assessment, candidate result, and fresh eligibility

## Recommended V4-115 Slice

V4-115 should implement exactly:

- direct command parsing for `HOT_UPDATE_OWNER_APPROVAL_DECISION_CREATE <job_id> <owner_approval_request_id> <decision> [reason...]`
- TaskState wrapper around `missioncontrol.CreateHotUpdateOwnerApprovalDecisionFromRequest`
- decided-at replay reuse by loading the deterministic decision record first
- deterministic default reason when no reason is supplied
- audit action `hot_update_owner_approval_decision_create`
- created and selected response strings
- focused tests for command parsing, created/selected responses, replay byte stability, malformed args, wrong job ID, invalid/stale authority, duplicate decisions, audit success/failure, status readout, and no mutation of unrelated runtime/hot-update surfaces
- V4-115 BEFORE/AFTER maintenance docs

V4-115 must not implement natural-language owner approval binding, runtime approval mutation, separate grant/rejection registries, owner approval expiration records, candidate promotion decisions, hot-update gates, canary execution/evidence creation, outcomes, promotions, rollbacks, rollback-apply records, active pointer mutation, last-known-good mutation, reload generation mutation, pointer-switch behavior, reload/apply behavior, or V4-116 work.
