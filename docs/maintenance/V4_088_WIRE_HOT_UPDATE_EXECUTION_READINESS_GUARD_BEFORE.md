# V4-088 Wire Hot-Update Execution Readiness Guard Before

## Before-State Gap

V4-087 added `missioncontrol.AssessHotUpdateExecutionReadiness(...)` as a read-only guard/read model, but no TaskState wrapper called it yet.

Before this slice:

- `HOT_UPDATE_GATE_EXECUTE <job_id> <hot_update_id>` could call `TaskState.ExecuteHotUpdateGatePointerSwitch`
- `HOT_UPDATE_GATE_RELOAD <job_id> <hot_update_id>` could call `TaskState.ExecuteHotUpdateGateReloadApply`
- the missioncontrol pointer-switch helper still owned active pointer mutation and `reload_generation` increment
- the missioncontrol reload/apply helper still owned reload/apply convergence
- idempotent pointer-switch replay and reload/apply replay were implemented in missioncontrol
- deploy-lock/quiesce readiness was assessable but not enforced by TaskState

The missing V4-088 behavior was to fail closed before new execution-sensitive transitions while preserving exact replay.

## Intended Wiring

Guard only these TaskState wrappers:

- `TaskState.ExecuteHotUpdateGatePointerSwitch`
- `TaskState.ExecuteHotUpdateGateReloadApply`

Guard only these transitions:

- `pointer_switch`
- `reload_apply`

Do not guard:

- prepared gate creation
- candidate-decision gate creation
- phase advancement to `validated` or `staged`
- terminal failure resolution
- outcome creation
- promotion creation
- LKG recertification

## Error Semantics

When readiness blocks execution, TaskState should return a deterministic `missioncontrol.ValidationError` whose error string includes:

- rejection code
- hot-update id
- transition
- reason

Direct commands should keep the existing behavior: empty response plus returned error.

The guarded wrappers should emit their existing audit actions with the error:

- `hot_update_gate_execute`
- `hot_update_gate_reload`

## Replay Semantics

The wrapper must call the guard before missioncontrol execution helpers, but exact replay must still pass:

- pointer-switch replay after the gate is already `reloading` and the active pointer already references the candidate/hot-update
- reload/apply replay after the gate is already `reload_apply_succeeded`

New attempts must be blocked when readiness is not proven:

- new pointer switch from `staged`
- new reload/apply from `reloading`
- reload/apply retry from `reload_apply_recovery_needed`

## Non-Goals

V4-088 must not:

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
