# V4-091 Hot-Update Execution Safety Evidence Control Entry Assessment

## Scope

V4-091 assesses the smallest safe control entry for producing `HotUpdateExecutionSafetyEvidenceRecord` after V4-090 added the missioncontrol registry and readiness guard consumption.

This slice is assessment-only. It does not add direct commands, TaskState wrappers, producer helpers, evidence records, pointer mutation, reload/apply behavior, outcomes, promotions, rollbacks, canary records, approval records, LKG records, or V4-092 implementation.

## Live State Inspected

- `docs/FRANK_V4_SPEC.md`
- `docs/maintenance/V4_089_EXPLICIT_QUIESCE_DEPLOY_LOCK_EVIDENCE_ASSESSMENT.md`
- `docs/maintenance/V4_090_HOT_UPDATE_EXECUTION_SAFETY_EVIDENCE_REGISTRY_AFTER.md`
- `internal/missioncontrol/hot_update_execution_safety_evidence.go`
- `internal/missioncontrol/hot_update_execution_readiness.go`
- `internal/agent/loop.go`
- `internal/agent/tools/taskstate.go`
- existing direct-command tests and audit assertions in `internal/agent/loop_processdirect_test.go`

## Current Evidence Surface

V4-090 added `HotUpdateExecutionSafetyEvidenceRecord` with deterministic ID:

```text
hot-update-execution-safety-<hot_update_id>-<job_id>
```

Records are stored under:

```text
runtime_packs/hot_update_execution_safety/<evidence_id>.json
```

The record is scoped to one hot update and one job. It can optionally bind to active-step evidence through:

- `active_step_id`
- `attempt_id`
- `writer_epoch`
- `activation_seq`

The readiness guard now consumes matching evidence for active occupied `live_runtime` jobs. It allows new pointer-switch or reload/apply readiness only when evidence exists, is unexpired, is not stale, has `deploy_lock_state=deploy_unlocked`, and has `quiesce_state=ready`.

The current store helper supports:

- first write
- exact replay with `changed=false`
- divergent duplicate rejection

There is no producer command or TaskState wrapper yet.

## Existing Control Patterns

Existing hot-update direct commands use the direct command path in `internal/agent/loop.go` and delegate to TaskState wrappers:

- `HOT_UPDATE_GATE_RECORD <job_id> <hot_update_id> <candidate_pack_id>`
- `HOT_UPDATE_GATE_FROM_DECISION <job_id> <promotion_decision_id>`
- `HOT_UPDATE_GATE_PHASE <job_id> <hot_update_id> <phase>`
- `HOT_UPDATE_GATE_EXECUTE <job_id> <hot_update_id>`
- `HOT_UPDATE_GATE_RELOAD <job_id> <hot_update_id>`
- `HOT_UPDATE_GATE_FAIL <job_id> <hot_update_id> [reason...]`
- `HOT_UPDATE_OUTCOME_CREATE <job_id> <hot_update_id>`
- `HOT_UPDATE_PROMOTION_CREATE <job_id> <outcome_id>`
- `HOT_UPDATE_LKG_RECERTIFY <job_id> <promotion_id>`

The command regexes are case-insensitive and use snake-case command names. Success responses use created/selected or executed/selected wording. Failure returns an empty response plus error.

TaskState wrappers consistently:

- resolve the mission store root
- validate active or persisted job context
- reject wrong `job_id`
- derive timestamps from the TaskState clock path
- use `operator` for operator-created records
- emit a runtime-control audit event with a snake-case action name
- return the missioncontrol `changed` flag or boolean equivalent

## Producer Decision

The first evidence producer should be a narrow operator direct command plus TaskState wrapper.

It should not be an automatic runtime-control helper first. The repo still lacks durable proof for "no tool call in flight" and "no governed external side effect in flight." Inferring quiesce from active-job occupancy or paused state would recreate the unsafe ambiguity V4-090 was designed to avoid.

It should not be both a direct command and automatic runtime-control producer in V4-092. The first implementation should make one explicit operator assertion path testable before any runtime auto-quiesce producer exists.

It does not need another commandless missioncontrol-only slice. V4-090 already added the missioncontrol registry and guard consumption. V4-092 may add a small missioncontrol construction/update helper to keep TaskState simple, but the implementation slice should expose the existing direct operator path.

## Recommended V4-092 Command

Recommend exactly this command:

```text
HOT_UPDATE_EXECUTION_READY <job_id> <hot_update_id> <ttl_seconds> [reason...]
```

Semantics:

- records `deploy_lock_state=deploy_unlocked`
- records `quiesce_state=ready`
- requires explicit positive `ttl_seconds`
- caps `ttl_seconds` at 300 seconds
- optional reason is trimmed free text
- missing reason should default to a deterministic short reason such as `operator asserted hot-update execution readiness`

This is intentionally narrower than:

```text
HOT_UPDATE_EXECUTION_SAFETY_RECORD <job_id> <hot_update_id> <deploy_lock_state> <quiesce_state> <ttl_seconds> [reason...]
```

The broad form would let an operator write arbitrary locked, unknown, or failed states before the system has a concrete producer model for those states. Missing or expired readiness already fails closed, so V4-092 does not need explicit failed/locked evidence to be safe.

This command is also clearer than `HOT_UPDATE_QUIESCE_RECORD` because V4-090 evidence is not only quiesce state; it is combined deploy-lock and quiesce readiness.

## TaskState Wrapper Behavior

Recommended wrapper name:

```go
RecordHotUpdateExecutionReady(jobID, hotUpdateID string, ttlSeconds int, reason string) (bool, error)
```

The wrapper should:

1. Resolve and validate the mission store root.
2. Validate active or persisted job context using the same pattern as existing hot-update wrappers.
3. Reject supplied `job_id` if it does not match the active/persisted job.
4. Require active or persisted runtime control context.
5. Load the current `ActiveJobRecord`.
6. Require the active job exists, holds global occupancy, and matches `job_id`.
7. Load committed `JobRuntimeRecord` and `RuntimeControlRecord` for `job_id`.
8. Require the active job is on `live_runtime`.
9. Require runtime/control active step evidence matches the active job where present.
10. Load the referenced `HotUpdateGateRecord`.
11. Require gate state is execution-relevant: `staged`, `reloading`, or `reload_apply_recovery_needed`.
12. Derive `created_at` from `taskStateTransitionTimestamp(taskStateNowUTC())`.
13. Derive `created_by=operator`.
14. Derive `expires_at=created_at+ttl_seconds`.
15. Populate `active_step_id`, `attempt_id`, `writer_epoch`, and `activation_seq` from active job evidence, not operator arguments.
16. Store or select the deterministic evidence record.
17. Emit audit action `hot_update_execution_ready` on success and failure.

The command should require `job_id` and `hot_update_id`. It should not require active step, attempt, writer epoch, or activation sequence as user-supplied arguments. Those fields are safety bindings and must come from current persisted active-job/runtime evidence.

## Expiry

Use explicit `ttl_seconds`, not an explicit timestamp and not a silent default.

Rationale:

- explicit timestamp is cumbersome and creates parsing/time-zone surface area in the direct command
- a silent default can accidentally create readiness the operator did not understand
- a small explicit TTL makes the assertion time-bounded and operator-visible

Recommended bounds:

- reject `ttl_seconds <= 0`
- reject `ttl_seconds > 300`

V4-092 tests should use the TaskState clock hook to make `created_at` and `expires_at` deterministic.

## Idempotence And Replacement

Exact replay should be byte-stable:

- if the deterministic evidence record already exists, is unexpired, and exactly matches the current active-job binding, readiness state, reason, `created_at`, and `expires_at`, the command returns `changed=false`
- direct command response should say `Selected hot-update execution readiness ...`

V4-092 should also allow narrowly scoped replacement when current evidence is expired.

Reason:

- evidence is current state, not immutable history
- without expired-record replacement, the deterministic ID would make it impossible to refresh readiness for the same `(hot_update_id, job_id)` after TTL expiry

Replacement should be fail-closed and limited:

- expired evidence may be replaced only for the same deterministic `(hot_update_id, job_id)`
- replacement must bind to the current active job evidence
- replacement must write only `deploy_unlocked` plus `ready`
- non-expired divergent evidence must fail closed
- stale active-step, attempt, writer-epoch, or activation-sequence evidence must fail closed
- evidence for a different job or hot update is impossible through the deterministic ID, but direct tests should still prove such records do not satisfy readiness

V4-092 should not add broad arbitrary overwrite semantics for non-expired records.

## Direct Command Responses

On `changed=true`:

```text
Recorded hot-update execution readiness job=<job_id> hot_update=<hot_update_id> expires_at=<rfc3339>.
```

On `changed=false`:

```text
Selected hot-update execution readiness job=<job_id> hot_update=<hot_update_id> expires_at=<rfc3339>.
```

On failure:

- return empty response plus error
- preserve existing direct command behavior
- include enough deterministic error context for malformed args, wrong job, missing gate, invalid gate state, invalid TTL, missing active job, non-live-runtime job, stale active evidence, duplicate divergence, and store validation failures

## Audit

Use audit action:

```text
hot_update_execution_ready
```

Emit it on success and failure through the existing TaskState audit path. Failure audit should carry the validation error code when the wrapper returns `missioncontrol.ValidationError`; otherwise it should preserve the existing invalid-runtime fallback.

## Fail-Closed Requirements

V4-092 must fail closed when:

- arguments are missing or malformed
- extra required-position arguments are invalid
- `ttl_seconds` is not an integer
- `ttl_seconds <= 0`
- `ttl_seconds > 300`
- store root is missing
- active/persisted job context is missing
- supplied job does not match active/persisted job
- active job record is missing
- active job record does not match `job_id`
- active job does not hold global occupancy
- committed runtime/control evidence is missing
- active job is not `live_runtime`
- active step, attempt, writer epoch, or activation sequence conflicts with persisted evidence
- hot-update gate is missing
- hot-update gate is not `staged`, `reloading`, or `reload_apply_recovery_needed`
- existing non-expired evidence is divergent
- existing evidence is stale for the current active job

All failures should return an empty direct response plus error.

## Required V4-092 Tests

V4-092 should add focused tests proving:

- command parses `HOT_UPDATE_EXECUTION_READY <job_id> <hot_update_id> <ttl_seconds> [reason...]`
- command rejects missing job ID, hot-update ID, and TTL
- command rejects extra malformed positional fields where applicable
- command rejects non-integer TTL
- command rejects zero, negative, and too-large TTL
- command creates `deploy_unlocked` plus `ready` evidence
- evidence ID is `hot-update-execution-safety-<hot_update_id>-<job_id>`
- evidence binds active step, attempt, writer epoch, and activation sequence from current active job evidence
- `created_by=operator`
- `created_at` comes from the TaskState clock path
- `expires_at=created_at+ttl_seconds`
- created response includes job, hot-update ID, and expiry
- exact replay returns selected response and is byte-stable
- expired evidence can be replaced for the same job/hot-update/current active binding
- non-expired divergent duplicate fails closed with empty response
- wrong job ID fails closed with empty response
- missing active job fails closed with empty response
- non-live-runtime active job fails closed with empty response
- missing runtime/control evidence fails closed with empty response
- stale active-step, attempt, writer-epoch, or activation-sequence evidence fails closed
- missing hot-update gate fails closed with empty response
- invalid gate state fails closed with empty response
- audit action `hot_update_execution_ready` is emitted on created path
- audit action `hot_update_execution_ready` is emitted on selected/idempotent path
- audit action `hot_update_execution_ready` is emitted with error on rejected path
- after evidence creation, `AssessHotUpdateExecutionReadiness` allows pointer-switch readiness for the active live-runtime job
- after evidence creation, `AssessHotUpdateExecutionReadiness` allows reload/apply readiness for the active live-runtime job
- wrapper does not mutate hot-update gates, active runtime-pack pointer, last-known-good pointer, `reload_generation`, runtime records, runtime control records, or active job records
- wrapper does not create outcomes, promotions, rollback records, rollback-apply records, canary records, approval records, or LKG records

## Recommended V4-092 Slice

Recommend exactly one implementation slice:

```text
V4-092 - Hot-Update Execution Ready Control Entry
```

Scope:

- add `HOT_UPDATE_EXECUTION_READY <job_id> <hot_update_id> <ttl_seconds> [reason...]`
- add a TaskState wrapper that validates active/persisted job context and current active live-runtime evidence
- add a small missioncontrol construction/update helper only if needed to keep current-state replacement semantics out of the direct command layer
- write only `deploy_lock_state=deploy_unlocked` and `quiesce_state=ready`
- preserve exact replay and allow only expired-record replacement for refresh
- add direct-command, TaskState, audit, readiness, and invariant tests

Non-goals:

- no broad arbitrary safety-state writer
- no automatic quiesce inference
- no deploy-lock or quiesce producer from tool-call or side-effect tracking
- no pointer switch changes
- no reload/apply changes
- no canary behavior
- no owner approval behavior
- no hot-update outcomes
- no promotions
- no rollbacks
- no rollback-apply records
- no LKG mutation
- no V4-093 work
