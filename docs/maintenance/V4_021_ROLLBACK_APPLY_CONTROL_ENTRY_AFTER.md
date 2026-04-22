# V4-021 Rollback-Apply Control Entry After

## `git diff --stat`

```text
 internal/agent/loop.go                             |  16 +++
 internal/agent/loop_processdirect_test.go          | 157 +++++++++++++++++++++
 internal/agent/tools/taskstate.go                  |  89 ++++++++++++
 internal/missioncontrol/rollback_apply_registry.go |  36 +++++
 .../missioncontrol/rollback_apply_registry_test.go |  80 +++++++++++
 5 files changed, 378 insertions(+)
```

Note: tracked `git diff --stat` output does not include new untracked files created in this slice.

## `git diff --numstat`

```text
16	0	internal/agent/loop.go
157	0	internal/agent/loop_processdirect_test.go
89	0	internal/agent/tools/taskstate.go
36	0	internal/missioncontrol/rollback_apply_registry.go
80	0	internal/missioncontrol/rollback_apply_registry_test.go
```

Note: tracked `git diff --numstat` output does not include new untracked files created in this slice.

## Files Changed

- `docs/maintenance/V4_021_ROLLBACK_APPLY_CONTROL_ENTRY_BEFORE.md`
- `docs/maintenance/V4_021_ROLLBACK_APPLY_CONTROL_ENTRY_AFTER.md`
- `internal/missioncontrol/rollback_apply_registry.go`
- `internal/missioncontrol/rollback_apply_registry_test.go`
- `internal/agent/tools/taskstate.go`
- `internal/agent/loop.go`
- `internal/agent/loop_processdirect_test.go`

## Exact Control-Surface Fields / Helpers Added

- direct operator command:
  - `ROLLBACK_APPLY_RECORD <job_id> <rollback_id> <apply_id>`
- `rollbackApplyRecordCommandRE` in `internal/agent/loop.go`
- `(*TaskState).EnsureRollbackApplyRecord(jobID, rollbackID, applyID string) (bool, error)`
- `missioncontrol.EnsureRollbackApplyRecordFromRollback(root, applyID, rollbackID, createdBy, requestedAt)`
- create-or-select behavior:
  - creates a rollback-apply record from an existing committed rollback when absent
  - selects the existing immutable rollback-apply record when the same `apply_id` already points at the same `rollback_id`
  - rejects mismatched existing `apply_id` to `rollback_id` linkage
- existing `STATUS <job_id>` surface reused unchanged for post-entry read-model inspection

## Exact Tests Added

- `TestEnsureRollbackApplyRecordFromRollbackCreatesOrSelectsExistingMatch`
- `TestEnsureRollbackApplyRecordFromRollbackRejectsMismatchedExistingRollback`
- `TestProcessDirectRollbackApplyRecordCommandCreatesOrSelectsWorkflowAndPreservesActiveRuntimePackPointer`
- `TestProcessDirectRollbackApplyRecordCommandFailsClosedWhenRollbackIsMissing`

## Validation Commands And Results

- `gofmt -w internal/missioncontrol/rollback_apply_registry.go internal/missioncontrol/rollback_apply_registry_test.go internal/agent/tools/taskstate.go internal/agent/loop.go internal/agent/loop_processdirect_test.go`
  - passed
- `git diff --check`
  - passed
- `go test -count=1 ./internal/agent`
  - passed
- `go test -count=1 ./internal/agent/tools`
  - passed
- `go test -count=1 ./internal/missioncontrol`
  - passed
- `go test -count=1 ./...`
  - passed
- `git status --short --branch`
  - current status after validation and report writing:

```text
## frank-v4-021-rollback-apply-control-entry
 M internal/agent/loop.go
 M internal/agent/loop_processdirect_test.go
 M internal/agent/tools/taskstate.go
 M internal/missioncontrol/rollback_apply_registry.go
 M internal/missioncontrol/rollback_apply_registry_test.go
?? docs/maintenance/V4_021_ROLLBACK_APPLY_CONTROL_ENTRY_AFTER.md
?? docs/maintenance/V4_021_ROLLBACK_APPLY_CONTROL_ENTRY_BEFORE.md
```

## Explicit No-Execution Statement

- No rollback apply behavior was implemented.
- No active runtime-pack pointer mutation was implemented.
- No promotion behavior changes, evaluator execution, scoring behavior, or autonomy changes were implemented.
- The new control surface only creates or selects durable rollback-apply workflow records using existing rollback and rollback-apply storage contracts.

## Deferred Next V4 Candidates

- rollback-apply phase progression beyond `recorded` in a separate non-executing slice
- rollback-apply execution behavior that mutates activation state or runtime-pack pointers in a later slice
- any broader operator inspect exposure only if a later slice needs rollback-apply control context beyond existing status/runtime-summary surfaces
