# V4-031 Rollback-Apply Retry After

## git diff --stat

```text
 internal/agent/loop_processdirect_test.go          | 113 +++++++++++++++++
 internal/missioncontrol/rollback_apply_registry.go |   3 +-
 .../missioncontrol/rollback_apply_registry_test.go | 139 +++++++++++++++++++++
 3 files changed, 254 insertions(+), 1 deletion(-)
```

## git diff --numstat

```text
113	0	internal/agent/loop_processdirect_test.go
2	1	internal/missioncontrol/rollback_apply_registry.go
139	0	internal/missioncontrol/rollback_apply_registry_test.go
```

## Files changed

- `internal/missioncontrol/rollback_apply_registry.go`
- `internal/missioncontrol/rollback_apply_registry_test.go`
- `internal/agent/loop_processdirect_test.go`
- `docs/maintenance/V4_031_ROLLBACK_APPLY_RETRY_BEFORE.md`
- `docs/maintenance/V4_031_ROLLBACK_APPLY_RETRY_AFTER.md`

## Exact helpers/state transitions added

- Widened `ExecuteRollbackApplyReloadApply(...)` to accept retry from:
  - `reload_apply_recovery_needed`
- Preserved the existing retry flow on the same rollback-apply record:
  - `reload_apply_recovery_needed -> reload_apply_in_progress`
  - `reload_apply_in_progress -> reload_apply_succeeded`
  - `reload_apply_in_progress -> reload_apply_failed`
- Kept the existing `ROLLBACK_APPLY_RELOAD <job_id> <apply_id>` control entry unchanged; the retry path works through the existing wrapper and command.
- Kept `reload_apply_succeeded` replay idempotent and kept `reload_apply_failed` behavior unchanged.

## Exact tests added

- `TestExecuteRollbackApplyReloadApplyRetryFromRecoveryNeededSucceedsWithoutSecondPointerMutation`
- `TestExecuteRollbackApplyReloadApplyRetryFromRecoveryNeededRecordsFailureAndClearsExecutionErrorOnStart`
- `storeRollbackApplyRecoveryNeededFixture`
- `TestProcessDirectRollbackApplyReloadCommandRetriesFromRecoveryNeeded`

## Validation commands and results

- `gofmt -w internal/missioncontrol/rollback_apply_registry.go internal/missioncontrol/rollback_apply_registry_test.go internal/agent/loop_processdirect_test.go` : passed
- `git diff --check` : passed
- `go test -count=1 ./internal/missioncontrol` : passed
- `go test -count=1 ./internal/agent` : passed
- `go test -count=1 ./internal/agent/tools` : passed
- `go test -count=1 ./...` : passed
- `git status --short --branch` : refreshed after report write below

## Explicit non-implementation statements

- No second pointer switch or `reload_generation` increment was implemented.
- No `last_known_good_pointer.json` mutation was implemented.
- No new rollback-apply record was created.

## Deferred next V4 candidates

- explicit operator-driven terminal-failure resolution from `reload_apply_recovery_needed`
- richer read-only exposure of retry/recovery execution detail if later checkpoints need it
- any broader retry policy from terminal `reload_apply_failed` only if a later checkpoint explicitly broadens that behavior
