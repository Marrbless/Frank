# V4-089 Explicit Quiesce / Deploy-Lock Evidence Surface Assessment

## Scope

V4-089 assesses the smallest safe evidence surface for explicit hot-update deploy-lock and quiesce readiness now that V4-088 wires `AssessHotUpdateExecutionReadiness(...)` into the TaskState execution paths for pointer switch and reload/apply.

This slice is assessment-only. It does not add deploy-lock records, quiesce records, commands, TaskState wrappers, enforcement changes, pointer mutation, reload/apply behavior, outcomes, promotions, rollbacks, canary records, approval records, LKG records, or V4-090 implementation.

## Live State Inspected

- `docs/FRANK_V4_SPEC.md`
- `docs/maintenance/V4_086_HOT_UPDATE_DEPLOY_LOCK_QUIESCE_ENFORCEMENT_ASSESSMENT.md`
- `docs/maintenance/V4_087_HOT_UPDATE_EXECUTION_READINESS_GUARD_AFTER.md`
- `docs/maintenance/V4_088_WIRE_HOT_UPDATE_EXECUTION_READINESS_GUARD_AFTER.md`
- `internal/missioncontrol/hot_update_execution_readiness.go`
- `internal/agent/tools/taskstate.go`
- `internal/missioncontrol/store_active_job.go`
- `internal/missioncontrol/store_records.go`
- existing hot-update gate, outcome, active-job, runtime, runtime-control, and status/read-model registry patterns

## Current Execution Readiness Surface

`AssessHotUpdateExecutionReadiness(root, input)` currently classifies hot-update transitions and reads execution evidence without mutating store state.

Execution-sensitive transitions:

- `pointer_switch`
- `reload_apply`

Non-execution-sensitive or out-of-scope transitions:

- `prepared_gate_create`
- `phase_validated`
- `phase_staged`
- `terminal_failure`
- `outcome_create`
- `promotion_create`
- `lkg_recertify`

The current input can carry explicit `QuiesceState`, `ActiveJob`, `JobRuntime`, and `RuntimeControl`, but V4-088 TaskState wiring only passes transition, hot-update ID, and command job ID. If an active occupied live-runtime job exists and no explicit quiesce proof is provided, the guard fails closed with `E_ACTIVE_JOB_DEPLOY_LOCK`. If explicit failed quiesce is supplied, it fails with `E_RELOAD_QUIESCE_FAILED`.

Replay is evaluated before live-job blocking:

- pointer-switch replay is allowed when the gate is already `reloading` and the active runtime-pack pointer already references the candidate pack through the hot-update update record
- reload/apply replay is allowed when the gate is already `reload_apply_succeeded`

This preserves idempotent replay even if a lock or live job appears after the transition already applied.

## Existing Runtime Evidence

The repo has enough runtime records to identify active occupancy, but not enough to prove quiesce readiness.

Current durable records:

- `ActiveJobRecord` in `active_job.json` identifies the active job, state, active step, lease holder, lease expiry, update time, and activation sequence.
- `HoldsGlobalActiveJobOccupancy` treats `running`, `waiting_user`, and `paused` as occupied.
- `JobRuntimeRecord` carries execution plane, mission family, active step, state, surface declarations, policy/evidence refs, and timestamps.
- `RuntimeControlRecord` carries committed control context for the active step, including execution plane, mission family, max authority, allowed tools, and step definition.

Missing durable evidence:

- no deploy-lock record
- no quiesce readiness record
- no in-flight tool call marker
- no governed external side-effect in-flight marker
- no lease-scoped proof that an occupied live-runtime job is safely paused between atomic actions
- no status/read-model block exposing hot-update execution safety evidence

The current guard therefore correctly treats occupied live-runtime work as unsafe unless explicit proof is supplied through input.

## Evidence Surface Decision

V4-090 should add one combined missioncontrol record/read-model skeleton, not separate deploy-lock and quiesce records and not a direct command first.

Recommended slice:

**V4-090 - Hot-Update Execution Safety Evidence Registry Skeleton**

Add a missioncontrol-only evidence registry and update the readiness guard to consume the current matching evidence. Do not add direct commands, TaskState write paths, automatic execution, canary behavior, approval behavior, or pointer/reload mutations.

Recommended record name:

`HotUpdateExecutionSafetyEvidenceRecord`

A combined record is the smallest safe fit because deploy-lock and quiesce readiness must be interpreted atomically for one active job and one hot update. Separate deploy-lock and quiesce records would introduce ordering and conflict semantics before the repo has any producer for the evidence.

Recommended storage model:

- job-scoped and hot-update-scoped current state
- deterministic ID: `hot-update-execution-safety-<hot_update_id>-<job_id>`
- overwrite-style current evidence with exact replay idempotence
- fail closed on divergent duplicate writes unless an explicit update helper validates a state transition

Rationale:

- Quiesce evidence is time-sensitive current state, closer to `active_job.json` and runtime-control state than to immutable promotion/outcome ledgers.
- The guard needs one current answer: whether this hot update may proceed while this job occupies the live runtime.
- Immutable append-only history can be added later if audit requirements demand it. The current repo already has runtime audit events for operator actions; V4-090 should not widen into audit-ledger design.

## Minimal Record Fields

Recommended minimum:

```go
type HotUpdateExecutionSafetyEvidenceRecord struct {
    RecordVersion int       `json:"record_version"`
    EvidenceID    string    `json:"evidence_id"`
    HotUpdateID   string    `json:"hot_update_id"`
    JobID         string    `json:"job_id"`
    ActiveStepID  string    `json:"active_step_id,omitempty"`
    AttemptID     string    `json:"attempt_id,omitempty"`
    WriterEpoch   uint64    `json:"writer_epoch,omitempty"`
    ActivationSeq uint64    `json:"activation_seq,omitempty"`

    DeployLockState string `json:"deploy_lock_state"`
    QuiesceState    string `json:"quiesce_state"`
    Reason          string `json:"reason,omitempty"`

    CreatedAt time.Time `json:"created_at"`
    CreatedBy string    `json:"created_by"`
    ExpiresAt time.Time `json:"expires_at"`
}
```

Recommended initial state values:

- deploy-lock states: `unknown`, `deploy_locked`, `deploy_unlocked`
- quiesce states: `unknown`, `ready`, `failed`

The existing assessment output can continue to report `not_configured` when no evidence record exists and `unknown` when evidence exists but does not prove readiness.

## Actor And Update Model

V4-090 should provide missioncontrol helpers only. The actor should be persisted as `created_by`, but no direct operator command should be added yet.

Allowed future writers:

- operator command, after the registry semantics are proven
- trusted runtime-control/quiesce helper, after a concrete quiesce observation path exists

V4-090 should not infer safe quiesce from active-job occupancy alone. It should accept only explicit evidence that matches the active job and hot update.

## Expiry And Staleness

Evidence must be fail-closed when:

- no record exists
- the record is expired
- `ExpiresAt` is zero for a ready/unlocked record
- the record hot-update ID differs from the assessed hot update
- the record job ID differs from the active occupied job or command job
- the record active step, writer epoch, or activation sequence conflicts with current active-job/runtime evidence when those fields are present
- deploy-lock state is `deploy_locked` or `unknown`
- quiesce state is `failed` or `unknown`
- validation cannot load or validate the record

Ready evidence should be short-lived and explicit. Expired ready evidence must not permit new pointer-switch or reload/apply attempts. Idempotent replay remains allowed because the guard already checks replay before evidence and active-job blocking.

## Guard Consumption

V4-090 should update `AssessHotUpdateExecutionReadiness` in missioncontrol to:

1. Preserve transition classification and replay-first behavior.
2. Load active job, job runtime, and runtime control as it does today.
3. When the active occupied job is the same hot-update control job, continue allowing it.
4. When the active occupied job is `live_runtime`, load the matching current `HotUpdateExecutionSafetyEvidenceRecord`.
5. Allow readiness only when the evidence is unexpired, scoped to the same hot update and job, deploy-lock state is `deploy_unlocked`, and quiesce state is `ready`.
6. Return `E_ACTIVE_JOB_DEPLOY_LOCK` for missing, expired, mismatched, unknown, or locked deploy-lock evidence.
7. Return `E_RELOAD_QUIESCE_FAILED` for explicit quiesce failure.

The assessment should include evidence identity/state in its output if added fields stay small and do not require a separate status design. At minimum, it should set `QuiesceState` based on the consumed record so direct command errors remain deterministic.

## Status / Read-Model Exposure

The first read-model surface should be the readiness assessment itself. It already returns the transition, hot-update ID, active job, active execution plane, quiesce state, readiness, rejection code, reason, and replay class.

After the registry and guard consumption are proven, a later slice can add operator status summary exposure. The eventual status block should be hot-update oriented and include:

- evidence ID
- hot-update ID
- job ID
- deploy-lock state
- quiesce state
- expiry
- stale/expired classification
- last reason

Do not add broad status formatting in V4-090 unless it is required to validate the registry. The smallest useful V4-090 surface is missioncontrol storage plus guard consumption tests.

## Idempotence

Record write idempotence:

- exact replay of the same evidence bytes returns `changed=false`
- divergent duplicate for the deterministic evidence ID fails closed unless a narrowly scoped update helper intentionally replaces current evidence
- deterministic timestamps should be supplied by the caller, not generated inside missioncontrol

Execution replay idempotence:

- pointer-switch replay after the pointer already switched remains allowed even if evidence is missing, expired, or locked
- reload/apply replay after `reload_apply_succeeded` remains allowed even if evidence is missing, expired, or locked
- new pointer-switch or reload/apply attempts require current matching evidence when active live-runtime work occupies the deploy lock

## Direct Command Ordering

The direct operator control surface should come after the missioncontrol evidence registry and guard consumption.

Reason:

- the repo already has TaskState enforcement wired to the read-only guard
- there is no durable evidence contract yet
- adding a command before the registry would force command semantics to define storage behavior implicitly
- missioncontrol tests can prove validation, idempotence, expiry, mismatch, and fail-closed behavior without widening operator command parsing

## Required V4-090 Tests

V4-090 should add focused missioncontrol tests proving:

- evidence record validation rejects missing record version, evidence ID, hot-update ID, job ID, deploy-lock state, quiesce state, created_at, created_by, and required expiry for ready evidence
- deterministic evidence ID is `hot-update-execution-safety-<hot_update_id>-<job_id>`
- exact evidence replay returns `changed=false` and leaves bytes stable
- divergent duplicate evidence fails closed
- missing evidence keeps active live-runtime execution blocked with `E_ACTIVE_JOB_DEPLOY_LOCK`
- expired evidence blocks with `E_ACTIVE_JOB_DEPLOY_LOCK`
- evidence for a different hot update blocks with `E_ACTIVE_JOB_DEPLOY_LOCK`
- evidence for a different job blocks with `E_ACTIVE_JOB_DEPLOY_LOCK`
- deploy_locked evidence blocks with `E_ACTIVE_JOB_DEPLOY_LOCK`
- deploy_unlocked plus quiesce_failed blocks with `E_RELOAD_QUIESCE_FAILED`
- deploy_unlocked plus quiesce_ready allows pointer-switch readiness for an active live-runtime job
- deploy_unlocked plus quiesce_ready allows reload/apply readiness for an active live-runtime job
- active step, writer epoch, or activation sequence mismatch blocks when those fields are present
- pointer-switch replay after already applied still succeeds without evidence
- reload/apply replay after success still succeeds without evidence
- guard remains read-only and does not mutate gates, active runtime-pack pointer, last-known-good pointer, runtime records, active job, or reload generation
- V4-090 creates no outcomes, promotions, rollback records, rollback-apply records, canary records, approval records, or LKG records

## V4 Core Versus Later Polish

V4 core:

- explicit evidence record/read-model for deploy-lock and quiesce readiness
- readiness guard consumption of current matching evidence
- fail-closed behavior for absent, expired, mismatched, locked, failed, or unknown evidence
- replay-first idempotence preservation

Later polish:

- direct command to create/update evidence
- richer operator status summary formatting
- append-only audit ledger for evidence history
- automatic quiesce observation from tool-call and side-effect trackers
- canary and owner-approval records

## Recommendation

Implement exactly one next slice:

**V4-090 - Hot-Update Execution Safety Evidence Registry Skeleton**

Scope:

- add a combined `HotUpdateExecutionSafetyEvidenceRecord`
- add missioncontrol validation, store/load helper, deterministic ID helper, and current evidence read-model
- update `AssessHotUpdateExecutionReadiness` to consume matching unexpired evidence for active live-runtime jobs
- add missioncontrol tests for validation, idempotence, expiry, mismatch, fail-closed behavior, ready/failed state behavior, replay preservation, and read-only invariants

Non-goals for V4-090:

- no direct commands
- no TaskState write wrappers
- no deploy-lock or quiesce producer
- no automatic pointer switch
- no active pointer mutation outside existing execution helpers
- no reload generation mutation outside existing execution helpers
- no last-known-good mutation
- no outcomes, promotions, rollbacks, rollback-apply records, canary records, approval records, or LKG records
- no canary or owner-approval implementation
