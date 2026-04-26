# V4-101 Hot-Update Canary Evidence Control Entry After

## Implemented Surface

V4-101 adds the direct operator command:

```text
HOT_UPDATE_CANARY_EVIDENCE_CREATE <job_id> <canary_requirement_id> <evidence_state> <observed_at> [reason...]
```

The command is parsed in `internal/agent/loop.go` and delegates to the TaskState wrapper:

```go
func (s *TaskState) CreateHotUpdateCanaryEvidenceFromRequirement(jobID string, canaryRequirementID string, state missioncontrol.HotUpdateCanaryEvidenceState, observedAt time.Time, reason string) (missioncontrol.HotUpdateCanaryEvidenceRecord, bool, error)
```

The wrapper calls:

```go
missioncontrol.CreateHotUpdateCanaryEvidenceFromRequirement(root, canaryRequirementID, state, observedAt, "operator", createdAt, reason)
```

No canary execution, canary execution automation, canary execution proposal, owner approval request, owner approval proposal, candidate promotion decision, hot-update gate, outcome, promotion, rollback, rollback-apply record, active pointer mutation, last-known-good mutation, or reload/apply change is created by this command.

## Command Parsing And Evidence State

The command accepts:

- `job_id`
- `canary_requirement_id`
- `evidence_state`
- `observed_at`
- optional trailing free-text `reason`

Valid evidence states are:

- `passed`
- `failed`
- `blocked`
- `expired`

The command does not accept `passed` as an argument. The registry infers `passed=true` only when `evidence_state=passed`; all other evidence states store `passed=false`.

If the reason is omitted or whitespace, TaskState supplies:

```text
operator recorded hot-update canary evidence <evidence_state>
```

Malformed commands with missing required arguments reject with:

```text
HOT_UPDATE_CANARY_EVIDENCE_CREATE requires job_id, canary_requirement_id, evidence_state, observed_at, and optional reason
```

## Observed Time And Deterministic ID

`observed_at` is parsed as RFC3339 or RFC3339Nano and normalized to UTC through the missioncontrol helper path. It is required because the durable evidence ID is deterministic:

```text
hot-update-canary-evidence-<canary_requirement_id>-<observed_at_utc_compact>
```

Supplying `observed_at` keeps exact direct-command replay stable and preserves append-only multiple-attempt behavior. Multiple evidence records for the same requirement remain allowed only when `observed_at` differs.

## TaskState Wrapper Behavior

The wrapper:

- returns zero record, `false`, `nil` when called on nil TaskState
- derives `now` from `taskStateTransitionTimestamp(taskStateNowUTC())`
- clones execution context, runtime control, runtime state, and mission store root under lock
- validates the mission store root
- validates the active execution context when present
- otherwise validates persisted runtime state and persisted runtime control context
- rejects a supplied `job_id` that does not match the active or persisted job context
- validates non-zero `observed_at`
- validates `evidence_state` as one of `passed`, `failed`, `blocked`, or `expired`
- trims `reason` and applies the deterministic default when empty
- uses `created_by="operator"`
- derives the deterministic canary evidence ID from `canary_requirement_id` and `observed_at`
- reuses an existing deterministic evidence record's `created_at` when it loads successfully
- fails closed if the existing deterministic evidence record returns anything other than not-found
- returns the missioncontrol record and changed flag

The `created_at` reuse makes an exact direct-command replay byte-stable. Without reuse, TaskState would generate a fresh timestamp and the registry would treat the replay as a divergent duplicate for the same evidence ID.

## Response Semantics

Created path:

```text
Created hot-update canary evidence job=<job_id> canary_requirement=<canary_requirement_id> canary_evidence=<canary_evidence_id> evidence_state=<state> passed=<bool>.
```

Selected/idempotent path:

```text
Selected hot-update canary evidence job=<job_id> canary_requirement=<canary_requirement_id> canary_evidence=<canary_evidence_id> evidence_state=<state> passed=<bool>.
```

Failure path returns an empty direct-command response plus the error.

## Audit Behavior

The wrapper emits runtime-control audit action:

```text
hot_update_canary_evidence_create
```

The audit action is emitted on created, selected/idempotent, and wrapper failure paths. Success and idempotent selection are allowed. Failure paths are rejected and use the existing audit code mapping for TaskState validation errors and missioncontrol helper errors.

## Fail-Closed Behavior

The command and wrapper fail closed for:

- missing or malformed command arguments
- invalid `observed_at`
- invalid evidence state
- missing mission store root
- missing active and persisted runtime context
- wrong `job_id`
- missing canary requirement
- invalid canary requirement
- canary requirement not in `state=required`
- missing candidate result
- missing linked improvement run
- missing linked improvement candidate
- missing or unfrozen eval suite
- missing promotion policy
- missing baseline runtime pack
- missing candidate runtime pack
- stale derived eligibility away from `canary_required` or `canary_and_owner_approval_required`
- divergent duplicate evidence ID
- existing deterministic evidence record that does not validate or load

Failed, blocked, and expired evidence are durable records but do not satisfy a canary requirement in this slice.

## Status And Read Model

`STATUS <job_id>` includes the existing V4-099 read-only identity surface:

```json
"hot_update_canary_evidence_identity": { ... }
```

TaskState status readout now composes this identity beside the canary requirement identity. The read model surfaces configured, not-configured, and invalid evidence records through the missioncontrol status helpers. It is read-only and does not mutate evidence, requirements, source records, runtime packs, pointers, last-known-good state, or `reload_generation`.

## Invariants Preserved

V4-101 preserves these non-goals:

- no canary execution
- no canary execution automation
- no canary execution proposal records
- no owner approval request
- no owner approval proposal records
- no candidate promotion decision for canary-required states
- no hot-update gate for canary-required states
- no outcome, promotion, rollback, rollback-apply, or last-known-good creation
- no candidate result mutation
- no canary requirement mutation
- no promotion policy mutation
- no runtime pack mutation
- no active runtime-pack pointer mutation
- no last-known-good pointer mutation
- no `reload_generation` mutation
- no pointer-switch behavior change
- no reload/apply behavior change
- no V4-102 work
