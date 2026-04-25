# V4-087 Hot-Update Execution Readiness Guard Before

## Before-State Gap

V4-086 found that the repo had the hot-update execution helpers and the V4 rejection codes, but no shared read-only deploy-lock/quiesce readiness surface.

Existing behavior before this slice:

- prepared hot-update gates could be created from candidate promotion decisions
- hot-update gates could be advanced to `validated` and `staged`
- pointer switch could mutate the active runtime-pack pointer and increment `reload_generation`
- reload/apply could run convergence from `reloading` or `reload_apply_recovery_needed`
- terminal failure, outcome creation, promotion creation, and LKG recertification had their own ledger/certification helpers
- direct commands and TaskState wrappers existed for the current hot-update lifecycle helpers

The missing piece was a single missioncontrol read model that could answer whether a transition is execution-sensitive and whether deploy-lock/quiesce evidence allows that execution transition.

## Constraints

V4-087 is a read-only skeleton slice.

It must not:

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
- create hot-update gates, outcomes, promotions, rollback records, rollback-apply records, canary records, approval records, or LKG records
- implement canary behavior
- implement owner approval behavior
- implement V4-088

## Intended Shape

Add a missioncontrol helper:

```go
AssessHotUpdateExecutionReadiness(root string, input HotUpdateExecutionReadinessInput) (HotUpdateExecutionReadinessAssessment, error)
```

The helper should classify the requested hot-update transition, inspect available active job/runtime/control evidence, detect replay-safe execution transitions where possible, and return a typed assessment without writing any files.

The input should carry:

- transition
- hot-update id
- command job id when relevant
- quiesce state placeholder
- optional active job, job runtime, and runtime control evidence

The assessment should carry:

- transition and transition class
- hot-update id and gate state when observed
- execution-sensitive flag
- ready flag
- rejection code
- reason
- active job id/state/execution-plane/mission-family/active-step when considered
- quiesce state
- replay classification

## Conservative Semantics

Initial semantics should be conservative because no durable deploy-lock or quiesce records exist yet:

- prepared gate creation is metadata and non-execution-sensitive
- phase advancement to `validated` and `staged` is metadata and non-execution-sensitive
- pointer switch is execution-sensitive
- reload/apply from `reloading` is execution-sensitive
- reload/apply retry from `reload_apply_recovery_needed` is execution-sensitive
- terminal failure resolution is metadata/recovery and not blocked
- outcome creation is ledger-only and not blocked
- promotion creation is ledger-only and not blocked
- LKG recertification is outside hot-update execution readiness enforcement
- absent active occupied job allows readiness
- active occupied same hot-update control job does not count as unsafe live work by itself
- active occupied `live_runtime` job without explicit quiesce proof returns a blocking assessment with `E_ACTIVE_JOB_DEPLOY_LOCK`
- explicit failed quiesce returns a blocking assessment with `E_RELOAD_QUIESCE_FAILED`
- exact pointer-switch replay after the pointer already switched is allowed
- exact reload/apply replay after success is allowed
