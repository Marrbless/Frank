# V4-097 Hot-Update Canary Requirement Control Entry After

## Implemented Surface

V4-097 adds the direct operator command:

```text
HOT_UPDATE_CANARY_REQUIREMENT_CREATE <job_id> <result_id>
```

The command is parsed in `internal/agent/loop.go` and delegates to the TaskState wrapper:

```go
func (s *TaskState) CreateHotUpdateCanaryRequirementFromCandidateResult(jobID string, resultID string) (missioncontrol.HotUpdateCanaryRequirementRecord, bool, error)
```

The wrapper calls:

```go
missioncontrol.CreateHotUpdateCanaryRequirementFromCandidateResult(root, resultID, "operator", createdAt)
```

No canary execution, canary evidence, owner approval request, candidate promotion decision, or hot-update gate is created by this command.

## Source Authority And ID Behavior

The command accepts `result_id`, not a caller-supplied canary requirement ID. The durable requirement ID remains deterministic:

```text
hot-update-canary-requirement-<result_id>
```

The created record is stored by the V4-095 registry under:

```text
runtime_packs/hot_update_canary_requirements/<canary_requirement_id>.json
```

The registry continues to derive eligibility from the committed `CandidateResultRecord` and its linked source records: improvement run, improvement candidate, frozen eval suite, promotion policy, baseline runtime pack, and candidate runtime pack.

## TaskState Wrapper Behavior

The wrapper:

- returns zero record, `false`, `nil` when called on nil TaskState
- derives `now` from `taskStateTransitionTimestamp(taskStateNowUTC())`
- clones execution context, runtime control, runtime state, and mission store root under lock
- validates the mission store root
- validates the active execution context when present
- otherwise validates persisted runtime state and persisted runtime control context
- rejects a supplied `job_id` that does not match the active or persisted job context
- uses `created_by="operator"`
- derives the deterministic canary requirement ID from `result_id`
- reuses an existing deterministic record's `created_at` when it loads successfully
- fails closed if the existing deterministic record returns anything other than not-found
- returns the missioncontrol record and changed flag

The `created_at` reuse makes an exact direct-command replay byte-stable. Without reuse, TaskState would generate a fresh timestamp and the registry would correctly treat the replay as a divergent duplicate.

## Response Semantics

Created path:

```text
Created hot-update canary requirement job=<job_id> result=<result_id> canary_requirement=<canary_requirement_id> owner_approval_required=<bool>.
```

Selected/idempotent path:

```text
Selected hot-update canary requirement job=<job_id> result=<result_id> canary_requirement=<canary_requirement_id> owner_approval_required=<bool>.
```

Failure path returns an empty direct-command response plus the error.

Malformed commands with missing or extra arguments reject with:

```text
HOT_UPDATE_CANARY_REQUIREMENT_CREATE requires job_id and result_id
```

## Owner Approval Behavior

For `canary_required`, the command creates/selects the canary requirement and returns:

```text
owner_approval_required=false
```

For `canary_and_owner_approval_required`, the command creates/selects the canary requirement and returns:

```text
owner_approval_required=true
```

It does not create an owner approval request or proposal. Owner approval control remains future work.

## Audit Behavior

The wrapper emits runtime-control audit action:

```text
hot_update_canary_requirement_create
```

The audit action is emitted on created, selected/idempotent, and failure paths. Success and idempotent selection are allowed. Failure paths are rejected and use the existing audit code mapping for TaskState validation errors and helper errors.

## Fail-Closed Behavior

The command fails closed for:

- missing or malformed command arguments
- missing mission store root
- missing active and persisted runtime context
- wrong `job_id`
- missing candidate result
- invalid candidate result
- missing linked improvement run
- missing linked improvement candidate
- missing or unfrozen eval suite
- missing promotion policy
- missing baseline runtime pack
- missing candidate runtime pack
- derived eligibility `eligible`
- derived eligibility `owner_approval_required`
- derived eligibility `rejected`
- derived eligibility `unsupported_policy`
- derived eligibility `invalid`
- stale eligibility that changes away from a canary-required state during replay
- divergent duplicate requirement
- invalid existing deterministic requirement
- another requirement for the same result under a different ID

Accepted derived eligibility states remain only `canary_required` and `canary_and_owner_approval_required`.

## Status And Read Model

`STATUS <job_id>` now includes the existing V4-095 read-only identity surface:

```json
"hot_update_canary_requirement_identity": { ... }
```

The read model surfaces configured, not-configured, and invalid records through the existing missioncontrol status helpers. The status path is read-only and does not mutate canary requirement records or source records.

## Invariants Preserved

V4-097 preserves these non-goals:

- no canary execution
- no canary evidence creation
- no owner approval request or owner approval proposal record
- no candidate promotion decision for canary-required states
- no hot-update gate for canary-required states
- no outcome, promotion, rollback, rollback-apply, or last-known-good creation
- no candidate result mutation
- no promotion policy mutation
- no runtime pack mutation
- no active runtime-pack pointer mutation
- no last-known-good pointer mutation
- no `reload_generation` mutation
- no reload/apply behavior change
- no promotion policy grammar broadening
- no canary execution readiness logic
- no owner approval control entry
- no V4-098 work
