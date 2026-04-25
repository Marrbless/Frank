# V4-087 Hot-Update Execution Readiness Guard After

## Implemented Helper

V4-087 added a read-only missioncontrol guard/read model:

```go
AssessHotUpdateExecutionReadiness(root string, input HotUpdateExecutionReadinessInput) (HotUpdateExecutionReadinessAssessment, error)
```

The helper lives in `internal/missioncontrol/hot_update_execution_readiness.go`.

## Input Shape

`HotUpdateExecutionReadinessInput` carries:

- `Transition`
- `HotUpdateID`
- `CommandJobID`
- `QuiesceState`
- optional `ActiveJob`
- optional `JobRuntime`
- optional `RuntimeControl`

When explicit evidence is not supplied, the helper reads the current store evidence:

- `active_job.json`
- committed job runtime record for the active job
- committed runtime control record for the active job
- hot-update gate record for replay classification when a hot-update id is supplied
- active runtime-pack pointer for pointer-switch replay classification

The helper does not write any store file.

## Output Shape

`HotUpdateExecutionReadinessAssessment` carries:

- transition
- transition class
- hot-update id
- observed gate state
- command job id
- execution-sensitive flag
- ready flag
- rejection code
- reason
- active job considered flag
- active job id
- active job state
- active execution plane
- active mission family
- active step id
- quiesce state
- replay class

## Transition Classification

The guard classifies:

- `prepared_gate_create` as `metadata`, non-execution-sensitive
- `phase_validated` as `metadata`, non-execution-sensitive
- `phase_staged` as `metadata`, non-execution-sensitive
- `pointer_switch` as `execution`, execution-sensitive
- `reload_apply` as `execution`, execution-sensitive
- `terminal_failure` as `metadata_recovery`, non-execution-sensitive
- `outcome_create` as `ledger`, non-execution-sensitive
- `promotion_create` as `ledger`, non-execution-sensitive
- `lkg_recertify` as `outside_hot_update_execution_readiness`, non-execution-sensitive

## Readiness Semantics

The guard is conservative:

- absent active occupied job allows readiness
- active occupied same hot-update control job allows readiness when the active runtime evidence is `hot_update_gate` and the job id matches the command job id
- active occupied `live_runtime` job without explicit quiesce proof blocks with `E_ACTIVE_JOB_DEPLOY_LOCK`
- active occupied `live_runtime` job with explicit failed quiesce blocks with `E_RELOAD_QUIESCE_FAILED`
- active occupied `live_runtime` job with explicit ready quiesce is allowed
- active occupied job with missing execution-plane evidence blocks with `E_ACTIVE_JOB_DEPLOY_LOCK`
- active occupied non-live-runtime job is not treated as unsafe live work by this skeleton

Because the repo still lacks durable deploy-lock and quiesce records, `HotUpdateQuiesceStateNotConfigured` is the default and is distinct from `HotUpdateQuiesceStateFailed`.

## Replay Semantics

The guard checks replay before applying active-job blocking:

- pointer-switch replay is allowed when the gate is already `reloading` and the active runtime-pack pointer already references the candidate pack and deterministic hot-update update record ref
- reload/apply replay is allowed when the gate is already `reload_apply_succeeded`
- reload/apply retry from `reload_apply_recovery_needed` is not replay and remains execution-sensitive

This preserves the expected future behavior that exact replay remains allowed even if an active live job appears after an execution transition already completed.

## Rejection Codes

The guard returns V4 rejection codes on blocking assessments:

- `E_ACTIVE_JOB_DEPLOY_LOCK` for unsafe or unproven active live work
- `E_RELOAD_QUIESCE_FAILED` for explicit quiesce failure

The helper returns these codes in the assessment. It does not currently return command-layer errors for them because V4-087 intentionally does not wire enforcement into TaskState or direct commands.

## Tests

V4-087 added focused missioncontrol tests in `internal/missioncontrol/hot_update_execution_readiness_test.go` covering:

- read-only behavior across hot-update gate, active pointer, LKG pointer, active job, runtime record, runtime control record, and `reload_generation`
- transition classification
- absent active occupied job readiness
- same hot-update control job readiness
- active live-runtime blocking without quiesce proof
- explicit quiesce failure blocking
- pointer-switch replay allowance
- reload/apply success replay allowance
- recovery-needed reload/apply retry blocking

## Invariants Preserved

This slice does not:

- wire enforcement into TaskState
- change direct command behavior
- create deploy-lock records
- create quiesce records
- mutate the active runtime-pack pointer
- mutate the last-known-good pointer
- mutate `reload_generation`
- advance hot-update phases
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
- implement V4-088
