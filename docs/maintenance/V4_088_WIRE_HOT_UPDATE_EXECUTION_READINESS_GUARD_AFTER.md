# V4-088 Wire Hot-Update Execution Readiness Guard After

## Implemented Wiring

V4-088 wires the V4-087 read-only readiness guard into TaskState execution-sensitive hot-update wrappers.

Guarded wrappers:

- `TaskState.ExecuteHotUpdateGatePointerSwitch`
- `TaskState.ExecuteHotUpdateGateReloadApply`

The direct command names and parsing are unchanged:

- `HOT_UPDATE_GATE_EXECUTE <job_id> <hot_update_id>`
- `HOT_UPDATE_GATE_RELOAD <job_id> <hot_update_id>`

## Guarded Transitions

`TaskState.ExecuteHotUpdateGatePointerSwitch` now calls:

```go
missioncontrol.AssessHotUpdateExecutionReadiness(root, missioncontrol.HotUpdateExecutionReadinessInput{
    Transition:  missioncontrol.HotUpdateExecutionTransitionPointerSwitch,
    HotUpdateID:  hotUpdateID,
    CommandJobID: jobID,
})
```

`TaskState.ExecuteHotUpdateGateReloadApply` now calls the same helper with:

```go
Transition: missioncontrol.HotUpdateExecutionTransitionReloadApply
```

If the assessment is not ready, the wrapper fails closed before calling the missioncontrol execution helper.

## Unguarded Transitions

V4-088 does not guard:

- prepared gate creation
- candidate-decision gate creation
- phase advancement to `validated` or `staged`
- terminal failure resolution
- outcome creation
- promotion creation
- LKG recertification

Those paths keep their existing TaskState and direct command behavior.

## Error Behavior

Blocked readiness returns a deterministic `missioncontrol.ValidationError`.

The error includes:

- `E_ACTIVE_JOB_DEPLOY_LOCK` or `E_RELOAD_QUIESCE_FAILED`
- `hot_update_id=<id>`
- `transition=<transition>`
- `reason=<assessment reason>`

Direct command behavior remains unchanged: the command returns an empty response plus the error.

Audit behavior uses the existing wrapper action names:

- blocked pointer switch emits `hot_update_gate_execute` with the readiness error
- blocked reload/apply emits `hot_update_gate_reload` with the readiness error

## Replay Behavior

Exact replay remains allowed because the guard classifies replay before active-job blocking:

- `HOT_UPDATE_GATE_EXECUTE` still selects idempotently after the pointer already switched, even if active live work appears later
- `HOT_UPDATE_GATE_RELOAD` still selects idempotently after reload/apply already succeeded, even if active live work appears later

New execution attempts are blocked when readiness is not proven:

- pointer switch from `staged`
- reload/apply from `reloading`
- reload/apply retry from `reload_apply_recovery_needed`

## Tests

V4-088 adds focused direct-command coverage proving:

- `HOT_UPDATE_GATE_EXECUTE` fails closed with empty response when readiness blocks a new pointer switch
- blocked pointer switch errors include `E_ACTIVE_JOB_DEPLOY_LOCK`, hot-update id, transition, and reason context
- blocked pointer switch emits `hot_update_gate_execute` audit with the rejection code
- blocked pointer switch does not mutate the gate, active pointer, last-known-good pointer, `reload_generation`, outcomes, promotions, rollback records, or rollback-apply records
- pointer-switch replay still succeeds after active live work appears later
- `HOT_UPDATE_GATE_RELOAD` fails closed with empty response when readiness blocks a new reload/apply
- blocked reload/apply errors include `E_ACTIVE_JOB_DEPLOY_LOCK`, hot-update id, transition, and reason context
- blocked reload/apply emits `hot_update_gate_reload` audit with the rejection code
- blocked reload/apply does not mutate the gate, active pointer, last-known-good pointer, `reload_generation`, outcomes, promotions, rollback records, or rollback-apply records
- reload/apply retry from `reload_apply_recovery_needed` is blocked when readiness is not proven
- reload/apply replay still succeeds after active live work appears later

Existing direct-command coverage continues to prove phase advancement, terminal failure, outcome creation, promotion creation, and LKG recertification behavior is unchanged.

## Invariants Preserved

This slice does not:

- add direct commands
- add command names
- change direct command parsing
- create deploy-lock records
- create quiesce records
- mutate active runtime-pack pointer outside the existing pointer-switch helper
- mutate last-known-good pointer
- mutate `reload_generation` outside the existing pointer-switch helper
- create hot-update gates
- create outcomes
- create promotions
- create rollbacks
- create rollback-apply records
- create canary records
- create approval records
- create or mutate LKG records
- execute canaries
- request owner approval
- implement V4-089
