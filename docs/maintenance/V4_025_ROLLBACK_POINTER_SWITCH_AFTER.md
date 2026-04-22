# V4-025 Rollback Pointer Switch After

## `git diff --stat`

```text
 internal/agent/loop.go                             |  15 +
 internal/agent/loop_processdirect_test.go          | 198 +++++++++++++
 internal/agent/tools/taskstate.go                  |  89 ++++++
 internal/missioncontrol/rollback_apply_registry.go | 134 ++++++++-
 .../missioncontrol/rollback_apply_registry_test.go | 307 +++++++++++++++++++++
 5 files changed, 739 insertions(+), 4 deletions(-)
```

Note: tracked `git diff --stat` output does not include new untracked maintenance reports created in this slice.

## `git diff --numstat`

```text
15	0	internal/agent/loop.go
198	0	internal/agent/loop_processdirect_test.go
89	0	internal/agent/tools/taskstate.go
130	4	internal/missioncontrol/rollback_apply_registry.go
307	0	internal/missioncontrol/rollback_apply_registry_test.go
```

Note: tracked `git diff --numstat` output does not include new untracked maintenance reports created in this slice.

## Files Changed

- `docs/maintenance/V4_025_ROLLBACK_POINTER_SWITCH_BEFORE.md`
- `docs/maintenance/V4_025_ROLLBACK_POINTER_SWITCH_AFTER.md`
- `internal/missioncontrol/rollback_apply_registry.go`
- `internal/missioncontrol/rollback_apply_registry_test.go`
- `internal/agent/tools/taskstate.go`
- `internal/agent/loop.go`
- `internal/agent/loop_processdirect_test.go`

## Exact Helpers / State Transitions Added

- rollback-apply phase added:
  - `pointer_switched_reload_pending`
- durable missioncontrol helper added:
  - `ExecuteRollbackApplyPointerSwitch(root, applyID, updatedBy, updatedAt)`
- helper behavior:
  - requires existing rollback-apply record
  - requires phase `ready_to_apply`
  - loads and validates linked rollback record
  - loads active runtime-pack pointer
  - switches `active_pack_id` to the rollback target pack
  - sets `previous_active_pack_id` to the pre-switch active pack
  - preserves `last_known_good_pack_id`
  - increments `reload_generation` exactly once on pointer mutation
  - sets `update_record_ref` to `rollback_apply:<apply_id>`
  - advances rollback-apply phase to `pointer_switched_reload_pending`
  - returns idempotent select behavior on exact replay
  - reconciles a replay where the pointer was already switched by the same apply id before the phase write completed
- helper added:
  - `rollbackApplyPointerUpdateRecordRef(applyID string) string`
- taskstate control entry added:
  - `(*TaskState).ExecuteRollbackApplyPointerSwitch(jobID, applyID string) (bool, error)`
- direct operator command added:
  - `ROLLBACK_APPLY_EXECUTE <job_id> <apply_id>`
- exact state transitions added:
  - rollback-apply workflow:
    - `ready_to_apply -> pointer_switched_reload_pending`
  - active runtime-pack pointer:
    - `active_pack_id: rollback.from_pack_id -> rollback.target_pack_id`
    - `previous_active_pack_id: prior active pack retained`
    - `reload_generation: n -> n+1`
    - `update_record_ref: -> rollback_apply:<apply_id>`

## Exact Tests Added

- `internal/missioncontrol/rollback_apply_registry_test.go`
  - `TestExecuteRollbackApplyPointerSwitchHappyPathPreservesLastKnownGood`
  - `TestExecuteRollbackApplyPointerSwitchReplayIsIdempotent`
  - `TestExecuteRollbackApplyPointerSwitchRejectsInvalidPhaseAndMissingRollbackWithoutPointerMutation`
- `internal/agent/loop_processdirect_test.go`
  - `TestProcessDirectRollbackApplyExecuteCommandSwitchesPointerAndIsReplaySafe`
  - `TestProcessDirectRollbackApplyExecuteCommandRejectsInvalidPhaseWithoutPointerMutation`

## Validation Commands And Results

- `gofmt -w internal/missioncontrol/rollback_apply_registry.go internal/missioncontrol/rollback_apply_registry_test.go internal/agent/tools/taskstate.go internal/agent/loop.go internal/agent/loop_processdirect_test.go`
  - passed
- `git diff --check`
  - passed
- `go test -count=1 ./internal/missioncontrol`
  - passed
- `go test -count=1 ./internal/agent`
  - passed
- `go test -count=1 ./internal/agent/tools`
  - passed
- `go test -count=1 ./...`
  - passed

## Explicit Boundaries Preserved

- reload/apply was not implemented
- last-known-good was left unchanged
- `last_known_good_pointer.json` was not mutated by the new execution helper
- no promotion behavior changes were implemented
- no evaluator execution, scoring, autonomy, provider, or channel behavior changes were implemented

## Deferred Next V4 Candidates

- read-only status exposure for rollback-apply execution progress beyond phase text if operators need explicit reload-pending wording outside the phase field
- execution slice that consumes `pointer_switched_reload_pending` and performs bounded reload/apply mechanics
- recovery policy for crashes or reload failure after pointer switch
