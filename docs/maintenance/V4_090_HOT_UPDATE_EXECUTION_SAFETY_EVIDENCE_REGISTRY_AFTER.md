# V4-090 Hot-Update Execution Safety Evidence Registry After

## Implemented Record

V4-090 adds a combined missioncontrol evidence record:

```go
type HotUpdateExecutionSafetyEvidenceRecord struct {
    RecordVersion   int
    EvidenceID      string
    HotUpdateID     string
    JobID           string
    ActiveStepID    string
    AttemptID       string
    WriterEpoch     uint64
    ActivationSeq   uint64
    DeployLockState HotUpdateDeployLockState
    QuiesceState    HotUpdateQuiesceState
    Reason          string
    CreatedAt       time.Time
    CreatedBy       string
    ExpiresAt       time.Time
}
```

Deploy-lock states:

- `unknown`
- `deploy_locked`
- `deploy_unlocked`

Evidence quiesce states:

- `unknown`
- `ready`
- `failed`

`not_configured` remains an assessment state for missing evidence, not a valid persisted evidence state.

## Storage Path

Evidence records are stored under:

```text
runtime_packs/hot_update_execution_safety/<evidence_id>.json
```

The deterministic evidence ID is:

```text
hot-update-execution-safety-<hot_update_id>-<job_id>
```

V4-090 adds normalize, validate, store, load, current lookup, and list helpers in missioncontrol.

## Validation Behavior

Validation rejects:

- missing or invalid `record_version`
- missing `evidence_id`
- missing `hot_update_id`
- missing `job_id`
- invalid deterministic evidence ID
- missing or invalid `deploy_lock_state`
- missing or invalid `quiesce_state`
- missing `created_at`
- missing `created_by`
- missing `expires_at` for `deploy_unlocked` plus `ready` evidence

Whitespace is normalized before storage. JSON remains deterministic through the existing mission store atomic JSON writer.

## Idempotence And Duplicates

The store helper returns the stored normalized record and a `changed` flag.

- first write returns `changed=true`
- exact replay returns `changed=false` and leaves bytes stable
- divergent duplicate for the same evidence ID fails closed
- list order is deterministic by JSON filename

V4-090 does not add append-only history. The record is a current evidence surface scoped by hot update and job.

## Readiness Guard Consumption

`AssessHotUpdateExecutionReadiness(...)` now consumes current matching evidence for active occupied `live_runtime` jobs.

Replay is still checked before active-job and evidence blocking:

- pointer-switch replay after the active pointer already switched remains allowed without evidence
- reload/apply replay after `reload_apply_succeeded` remains allowed without evidence

For a new pointer switch or reload/apply attempt with an active occupied live-runtime job, readiness is allowed only when:

- matching evidence exists for the assessed hot update and active job
- evidence is unexpired
- evidence deploy-lock state is `deploy_unlocked`
- evidence quiesce state is `ready`
- populated `active_step_id`, `attempt_id`, `writer_epoch`, and `activation_seq` match current active job evidence

The guard blocks with `E_ACTIVE_JOB_DEPLOY_LOCK` for:

- missing evidence
- expired evidence
- mismatched or stale evidence
- deploy-lock state `unknown`
- deploy-lock state `deploy_locked`
- quiesce state `unknown`

The guard blocks with `E_RELOAD_QUIESCE_FAILED` for explicit quiesce failure.

## Read Model

The existing readiness assessment now exposes the consumed evidence where applicable:

- `evidence_id`
- `deploy_lock_state`
- `quiesce_state`
- `evidence_expires_at`
- `evidence_expired`
- `evidence_stale`

No broad operator status formatting or direct command output is added in this slice.

## Tests

V4-090 adds missioncontrol coverage proving:

- required evidence fields are validated
- ready/unlocked evidence requires `expires_at`
- deterministic evidence ID behavior
- store/load/list behavior
- exact replay idempotence and byte stability
- divergent duplicate rejection
- missing, expired, different hot-update, different job, locked, failed, and stale evidence fail closed
- ready evidence allows pointer-switch and reload/apply readiness for active live-runtime work
- pointer-switch and reload/apply replay behavior remains preserved by existing replay tests
- the guard remains read-only over gates, pointers, active job, runtime records, runtime control records, evidence records, and reload generation
- no outcome, promotion, rollback, rollback-apply, canary, approval, or LKG behavior is created

## Invariants Preserved

This slice does not:

- add direct commands
- add TaskState write wrappers
- add evidence producer logic
- create deploy-lock command behavior
- create quiesce observation behavior from tool calls or side effects
- mutate the active runtime-pack pointer
- mutate the last-known-good pointer
- mutate `reload_generation`
- execute pointer switch
- run reload/apply
- create hot-update gates
- create hot-update outcomes
- create promotions
- create rollback records
- create rollback-apply records
- create canary records
- create approval records
- create or mutate LKG records
- implement canary behavior
- implement owner approval behavior
- implement V4-091
