# GC5-TREAT-002 ProcessDirect Rollback Apply Split After

Branch: `frank-garbage-day-gc4-005-maintenance-artifact-retention`

## Completed

- Added `internal/agent/loop_processdirect_rollback_apply_test.go`.
- Moved the contiguous `ROLLBACK_APPLY_*` process-direct command tests out of `loop_processdirect_test.go`.
- Preserved test names, command strings, assertions, helper usage, and package visibility.

## Resulting Shape

- `internal/agent/loop_processdirect_test.go`: `9847` lines after the split.
- `internal/agent/loop_processdirect_rollback_apply_test.go`: `1038` lines.

## Validation

- `/usr/local/go/bin/go test -count=1 ./internal/agent -run 'TestProcessDirectRollbackApply'`
- `/usr/local/go/bin/go test -count=1 ./internal/agent`

## Remaining

Continue with `GC5-003` from `REPO_WIDE_GARBAGE_CAMPAIGN_MATRIX.md`.
