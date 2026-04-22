# V4-027 Rollback Reload/Apply After

## `git diff --stat`

```text
 internal/agent/loop.go                             |  15 +
 internal/agent/loop_processdirect_test.go          | 186 ++++++++++++
 internal/agent/tools/taskstate.go                  |  89 ++++++
 internal/missioncontrol/rollback_apply_registry.go | 207 ++++++++++++-
 .../missioncontrol/rollback_apply_registry_test.go | 335 +++++++++++++++++++++
 5 files changed, 831 insertions(+), 1 deletion(-)
```

Note: tracked `git diff --stat` output does not include new untracked maintenance reports created in this slice.

## `git diff --numstat`

```text
15	0	internal/agent/loop.go
186	0	internal/agent/loop_processdirect_test.go
89	0	internal/agent/tools/taskstate.go
206	1	internal/missioncontrol/rollback_apply_registry.go
335	0	internal/missioncontrol/rollback_apply_registry_test.go
```

Note: tracked `git diff --numstat` output does not include new untracked maintenance reports created in this slice.

## Files Changed

- `docs/maintenance/V4_027_ROLLBACK_RELOAD_APPLY_BEFORE.md`
- `docs/maintenance/V4_027_ROLLBACK_RELOAD_APPLY_AFTER.md`
- `internal/missioncontrol/rollback_apply_registry.go`
- `internal/missioncontrol/rollback_apply_registry_test.go`
- `internal/agent/tools/taskstate.go`
- `internal/agent/loop.go`
- `internal/agent/loop_processdirect_test.go`

## Exact Helpers / State Transitions Added

- rollback-apply phases added:
  - `reload_apply_in_progress`
  - `reload_apply_succeeded`
  - `reload_apply_failed`
- rollback-apply durable field added:
  - `execution_error`
- validation behavior added:
  - `execution_error` is required only for `reload_apply_failed`
  - `execution_error` is rejected for all non-failed phases
- durable missioncontrol helper added:
  - `ExecuteRollbackApplyReloadApply(root, applyID, updatedBy, updatedAt)`
- internal missioncontrol helper added for testable convergence:
  - `executeRollbackApplyReloadApplyWithConvergence(...)`
- bounded convergence helper added:
  - `rollbackApplyRestartStyleConvergence(...)`
  - current skeleton behavior re-resolves the already-switched active runtime-pack pointer and target pack record as the bounded restart-style convergence path
- helper behavior:
  - requires existing rollback-apply record in `pointer_switched_reload_pending`
  - validates linked rollback record
  - validates already-switched active pointer linkage:
    - `active_pack_id == rollback.target_pack_id`
    - `previous_active_pack_id == rollback.from_pack_id`
    - `update_record_ref == rollback_apply:<apply_id>`
  - transitions `pointer_switched_reload_pending -> reload_apply_in_progress`
  - runs bounded convergence
  - on success transitions `reload_apply_in_progress -> reload_apply_succeeded`
  - on failure transitions `reload_apply_in_progress -> reload_apply_failed`
  - stores error detail in `execution_error` on failure
  - replay from `reload_apply_succeeded` is idempotent
  - replay from `reload_apply_failed` fails closed
- taskstate control entry added:
  - `(*TaskState).ExecuteRollbackApplyReloadApply(jobID, applyID string) (bool, error)`
- direct operator command added:
  - `ROLLBACK_APPLY_RELOAD <job_id> <apply_id>`

## Exact Tests Added

- `internal/missioncontrol/rollback_apply_registry_test.go`
  - `TestExecuteRollbackApplyReloadApplyHappyPathPreservesPointerAndLastKnownGood`
  - `TestExecuteRollbackApplyReloadApplyRecordsFailureWithoutMutatingPointer`
  - `TestExecuteRollbackApplyReloadApplyReplayAfterSuccessIsIdempotent`
  - `TestExecuteRollbackApplyReloadApplyRejectsInvalidStartingPhase`
- `internal/agent/loop_processdirect_test.go`
  - `TestProcessDirectRollbackApplyReloadCommandSucceedsWithoutSecondPointerMutation`
  - `TestProcessDirectRollbackApplyReloadCommandRejectsInvalidStartingPhase`

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

- no `last_known_good_pointer.json` mutation was implemented
- no second pointer switch was implemented
- no second `reload_generation` increment was implemented
- no promotion behavior changes were implemented
- no evaluator execution, scoring, autonomy, provider, or channel behavior changes were implemented beyond the bounded restart-style convergence skeleton

## Deferred Next V4 Candidates

- richer read-model exposure for reload/apply failure detail if operator inspection needs `execution_error` through `STATUS`
- recovery handling for persisted `reload_apply_in_progress` after crash or interrupted execution
- explicit retry semantics for `reload_apply_failed` if a later slice decides to permit bounded operator retry
