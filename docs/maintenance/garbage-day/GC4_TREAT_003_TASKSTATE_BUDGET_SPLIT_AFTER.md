# GC4-TREAT-003 TaskState Budget Split After

Branch: `frank-garbage-day-gc4-003-taskstate-budget-split`

## Completed

- Added `internal/agent/tools/taskstate_runtime_budget.go`.
- Moved `EnforceUnattendedWallClockBudget` and `RecordFailedToolAction` out of `taskstate.go`.
- Preserved same-package visibility and existing call sites.
- Avoided approval, persistence, treasury, hot-update, rollback, and Frank Zoho transition internals.

## Resulting Shape

- `internal/agent/tools/taskstate.go`: `4726` lines after the split.
- `internal/agent/tools/taskstate_runtime_budget.go`: `61` lines.

## Validation

- `git diff --check`
- `/usr/local/go/bin/go test -count=1 ./internal/agent/tools -run 'TestTaskStateEnforceUnattendedWallClockBudget|TestTaskStateRecordFailedToolAction|TestTaskStateApplyStepOutputPausesForUnattendedWallClockBudget'`
- `/usr/local/go/bin/go test -count=1 ./internal/agent/tools`
- `/usr/local/go/bin/go test -count=1 ./...`

## Remaining

The next cleanup candidate is `GC4-004`: split one runtime bootstrap command subfamily from `cmd/picobot/main_runtime_bootstrap_test.go`.
