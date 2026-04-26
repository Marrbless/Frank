# V4-118 - Hot-Update Canary Gate Control Entry Assessment

## Scope

V4-118 assesses the smallest safe TaskState/direct-command surface for calling the V4-117 missioncontrol helper:

```go
CreateHotUpdateGateFromCanarySatisfactionAuthority(
    root string,
    canarySatisfactionAuthorityID string,
    ownerApprovalDecisionID string,
    createdBy string,
    createdAt time.Time,
) (HotUpdateGateRecord, bool, error)
```

This slice is assessment-only. It does not change Go code, tests, commands, TaskState wrappers, hot-update gates, candidate promotion decisions, canary authorities, owner approval decisions, outcomes, promotions, rollbacks, rollback-apply records, active runtime-pack pointers, last-known-good pointers, `reload_generation`, pointer-switch behavior, reload/apply behavior, or V4-119 implementation.

## Live State Inspected

Docs inspected:

- `docs/FRANK_V4_SPEC.md`
- `docs/maintenance/V4_116_CANARY_OWNER_APPROVAL_GATE_AUTHORITY_PATH_ASSESSMENT.md`
- `docs/maintenance/V4_117_HOT_UPDATE_GATE_FROM_CANARY_SATISFACTION_AUTHORITY_AFTER.md`

Code inspected:

- `internal/missioncontrol/hot_update_gate_registry.go`
- `internal/missioncontrol/status.go`
- `internal/agent/loop.go`
- `internal/agent/tools/taskstate.go`
- `internal/agent/tools/taskstate_readout.go`
- existing direct command and TaskState patterns for:
  - `HOT_UPDATE_GATE_FROM_DECISION`
  - `HOT_UPDATE_CANARY_SATISFACTION_AUTHORITY_CREATE`
  - `HOT_UPDATE_OWNER_APPROVAL_DECISION_CREATE`
- adjacent hot-update outcome, promotion, rollback, rollback-apply, active pointer, LKG, and reload/apply surfaces

## Existing Helper Contract

V4-117 added a dedicated canary-required gate helper while preserving the existing eligible-only path:

```go
HotUpdateGateIDFromCanarySatisfactionAuthority(canarySatisfactionAuthorityID string) string

CreateHotUpdateGateFromCanarySatisfactionAuthority(
    root string,
    canarySatisfactionAuthorityID string,
    ownerApprovalDecisionID string,
    createdBy string,
    createdAt time.Time,
) (HotUpdateGateRecord, bool, error)
```

The helper creates/selects only `state=prepared`, `decision=keep_staged` gates. It sets:

```text
canary_ref=<canary_satisfaction_authority_id>
approval_ref="" for no-owner-approval branch
approval_ref=<owner_approval_decision_id> for owner-approved branch
```

It does not create candidate promotion decisions, does not call `CreateHotUpdateGateFromCandidatePromotionDecision(...)`, does not execute gates, and does not mutate active pointer, LKG pointer, or `reload_generation`.

## Command Shape Decision

V4-119 should add exactly one direct command:

```text
HOT_UPDATE_CANARY_GATE_CREATE <job_id> <canary_satisfaction_authority_id> [owner_approval_decision_id]
```

This is preferred over:

```text
HOT_UPDATE_GATE_FROM_CANARY_AUTHORITY <job_id> <canary_satisfaction_authority_id> [owner_approval_decision_id]
```

Reasons:

- the command creates a distinct canary-gated `HotUpdateGateRecord`, so `_CREATE` matches the current V4 authority-control naming pattern
- the source authority is already explicit in the second argument name
- the optional owner approval decision is part of the authority chain, so a `FROM_CANARY_AUTHORITY` name would understate the owner-approved branch
- the audit action can cleanly mirror the command as `hot_update_canary_gate_create`

The command should require `job_id` because all adjacent direct operator commands validate active or persisted job context before writing missioncontrol records.

## Source Authority Arguments

The command should use `canary_satisfaction_authority_id` as the source authority. It should not accept raw canary evidence, canary requirement ID, owner approval request ID, owner approval decision ID alone, candidate result ID, or future hot-update ID as the primary source.

`owner_approval_decision_id` should be optional at the command surface:

- omitted for `state=authorized`, `owner_approval_required=false`
- required by the missioncontrol helper for `state=waiting_owner_approval`, `owner_approval_required=true`
- rejected by the missioncontrol helper when supplied for the no-owner-approval branch

The wrapper should not pre-create, infer, or search for owner approval decisions. The operator must supply the decision ID for the owner-approved branch.

## TaskState Wrapper Decision

V4-119 should add a TaskState wrapper equivalent to:

```go
func (s *TaskState) CreateHotUpdateGateFromCanarySatisfactionAuthority(
    jobID string,
    canarySatisfactionAuthorityID string,
    ownerApprovalDecisionID string,
) (missioncontrol.HotUpdateGateRecord, bool, error)
```

Wrapper behavior should mirror the V4-111 and V4-115 durable authority controls:

- return zero record, `false`, `nil` when `s == nil`, consistent with local wrapper conventions
- derive `now` from `taskStateTransitionTimestamp(taskStateNowUTC())`
- clone execution context, runtime control, runtime state, and mission store root under lock
- build audit execution context from active execution context or persisted runtime/control context
- validate mission store root
- validate active execution context when present:
  - job exists
  - step exists
  - runtime exists
  - supplied `job_id` matches active execution context job
- otherwise validate persisted runtime state and persisted runtime control context:
  - runtime state exists
  - runtime control context exists
  - supplied `job_id` matches persisted runtime state
  - supplied `job_id` matches persisted runtime control
- normalize and validate `canary_satisfaction_authority_id`
- trim `owner_approval_decision_id`; if non-empty, normalize and validate it as an owner approval decision ref
- derive deterministic gate ID using `missioncontrol.HotUpdateGateIDFromCanarySatisfactionAuthority(canarySatisfactionAuthorityID)`
- set `createdAt := now`
- if the deterministic gate already loads successfully, reuse `existing.PreparedAt` for replay stability
- if loading returns `ErrHotUpdateGateRecordNotFound`, continue with `createdAt := now`
- if loading returns any other error, fail closed
- call `missioncontrol.CreateHotUpdateGateFromCanarySatisfactionAuthority(root, canarySatisfactionAuthorityID, ownerApprovalDecisionID, "operator", createdAt)`
- emit audit on success, idempotent selection, and failure
- return the missioncontrol record and changed flag

## PreparedAt Replay Reuse

The wrapper should reuse an existing deterministic gate's `PreparedAt` when `LoadHotUpdateGateRecord(root, hotUpdateID)` succeeds. This matches the current `CreateHotUpdateGateFromCandidatePromotionDecision` TaskState wrapper pattern, which reuses existing `PreparedAt` before calling the missioncontrol helper.

If the existing deterministic gate is invalid, stale, malformed, or fails to load for any reason other than not-found, the wrapper must fail closed and emit a rejected audit event. It must not overwrite or repair the gate.

Exact replay should therefore be byte-stable when the same normalized canary satisfaction authority ID, owner approval decision ID, `created_by=operator`, and reused `PreparedAt` match the stored record.

## Audit Action

V4-119 should emit:

```text
hot_update_canary_gate_create
```

The audit result should be applied on both created and selected/idempotent paths, and rejected on failure. This follows the recent V4 command controls for canary satisfaction authority creation and owner approval decision creation.

## Response Semantics

Created response:

```text
Created hot-update canary gate job=<job_id> canary_satisfaction_authority=<canary_satisfaction_authority_id> hot_update=<hot_update_id> canary_ref=<canary_ref> approval_ref=<approval_ref>.
```

Selected response:

```text
Selected hot-update canary gate job=<job_id> canary_satisfaction_authority=<canary_satisfaction_authority_id> hot_update=<hot_update_id> canary_ref=<canary_ref> approval_ref=<approval_ref>.
```

The response should include `canary_ref` and `approval_ref` because those fields distinguish the no-owner-approval branch from the owner-approved branch and are the durable downstream source references.

Malformed command error:

```text
HOT_UPDATE_CANARY_GATE_CREATE requires job_id, canary_satisfaction_authority_id, and optional owner_approval_decision_id
```

The parser should reject missing arguments and extra arguments beyond the optional owner approval decision ID.

## Status And Read-Model Behavior

No separate status command is needed. `STATUS <job_id>` already composes `hot_update_gate_identity` through the existing readout path.

The created gate should appear in the existing hot-update gate identity read model. V4-119 should not change status/read-model structure unless a test proves that `canary_ref` or `approval_ref` is not surfaced where needed. If those refs are missing from the current gate status shape, V4-119 may add read-only status fields for them as part of the same command slice, because the command response and canary-gate authority auditability depend on those source refs being visible.

The read model must remain read-only and must not repair invalid gates.

## Fail-Closed Behavior

V4-119 should fail closed for:

- missing or malformed command arguments
- wrong `job_id`
- missing mission store root
- missing active execution context and missing persisted runtime context
- missing persisted runtime control context when local wrapper patterns require it
- invalid `canary_satisfaction_authority_id`
- missing canary satisfaction authority
- invalid or stale canary satisfaction authority
- authority state other than `authorized` or `waiting_owner_approval`
- `owner_approval_required=false` with a supplied owner approval decision ID
- `owner_approval_required=true` without a supplied owner approval decision ID
- missing, invalid, rejected, stale, or mismatched owner approval decision
- stale fresh canary satisfaction
- stale fresh promotion eligibility
- selected canary evidence missing, non-passed, stale, or mismatched
- missing or mismatched canary requirement
- missing or mismatched candidate result, improvement run, improvement candidate, eval suite, promotion policy, baseline pack, candidate pack, active pointer, rollback target pack
- active pointer not equal to the authority baseline pack
- invalid present LKG pointer
- existing deterministic gate that fails to load
- divergent duplicate gate

All these failures should emit rejected audit action `hot_update_canary_gate_create` and return an empty direct-command response plus error.

## Explicit Non-Goals For V4-119

V4-119 should not execute the gate, advance gate phase, pointer-switch, reload/apply, create a hot-update outcome, create a promotion, create a rollback, create a rollback-apply record, mutate active runtime-pack pointer, mutate last-known-good pointer, mutate `reload_generation`, mutate canary satisfaction authority, mutate owner approval decision, mutate source records, broaden `CandidatePromotionDecisionRecord`, create candidate promotion decisions for canary-required states, change `CreateHotUpdateGateFromCandidatePromotionDecision(...)`, or implement V4-120.

## Recommended V4-119 Slice

Implement exactly:

```text
V4-119 - Hot-Update Canary Gate Control Entry
```

Scope:

- add `HOT_UPDATE_CANARY_GATE_CREATE <job_id> <canary_satisfaction_authority_id> [owner_approval_decision_id]`
- add the TaskState wrapper around `CreateHotUpdateGateFromCanarySatisfactionAuthority(...)`
- reuse existing `PreparedAt` for deterministic replay
- emit `hot_update_canary_gate_create` audit events
- return created vs selected responses with `canary_ref` and `approval_ref`
- prove `STATUS <job_id>` surfaces the created gate through `hot_update_gate_identity`
- keep all execution, outcome, promotion, rollback, LKG, pointer, and reload/apply behavior untouched
