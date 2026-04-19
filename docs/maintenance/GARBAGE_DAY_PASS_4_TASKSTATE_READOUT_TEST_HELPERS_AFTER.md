# Garbage Day Pass 4 TaskState Readout Test Helpers After

## Git Diff Stat
```text
 internal/agent/tools/taskstate_status_test.go | 22 ----------------------
 internal/agent/tools/taskstate_test.go        | 22 ----------------------
 2 files changed, 44 deletions(-)
```

## Git Diff Numstat
```text
0	22	internal/agent/tools/taskstate_status_test.go
0	22	internal/agent/tools/taskstate_test.go
```

## Files Changed
- Modified tracked test files:
  - `internal/agent/tools/taskstate_test.go`
  - `internal/agent/tools/taskstate_status_test.go`
- Added same-package test helper file:
  - `internal/agent/tools/taskstate_readout_test_helpers_test.go`
- Report artifacts added in this pass:
  - `docs/maintenance/GARBAGE_DAY_PASS_4_TASKSTATE_READOUT_TEST_HELPERS_BEFORE.md`
  - `docs/maintenance/GARBAGE_DAY_PASS_4_TASKSTATE_READOUT_TEST_HELPERS_AFTER.md`

## Before / After Line Counts
- `internal/agent/tools/taskstate_test.go`
  - before: `7763`
  - after: `7741`
- `internal/agent/tools/taskstate_status_test.go`
  - before: `1553`
  - after: `1531`
- `internal/agent/tools/taskstate_readout_test_helpers_test.go`
  - before: not present
  - after: `29`

## Exact Helpers Moved
- Moved into `internal/agent/tools/taskstate_readout_test_helpers_test.go`:
  - `writeMalformedTreasuryRecordForTaskStateReadoutTest`
- Removed duplicated local definitions:
  - `writeMalformedTreasuryRecordForTaskStateTest`
  - `writeMalformedTreasuryRecordForTaskStateStatusTest`

## Exact Assertions Preserved
- No test assertions changed.
- No JSON key expectations changed.
- No active-vs-persisted parity assertions changed.
- No readout adapter boundary assertions changed.
- No scenario names changed.
- No test coverage was removed or skipped.

## Helpers Intentionally Left Duplicated Or Unmoved
- `writeDeferredSchedulerTriggerForTaskStateStatusTest` was left in `taskstate_status_test.go`.
  - Reason: it is status-specific, not duplicated across the two target files, and moving it would add churn without reducing duplication.
- Existing shared readout helpers such as `mustTaskStateReadoutJSON`, `mustTaskStateJSONObject`, `mustTaskStateJSONArray`, `assertTaskStateReadoutAdapterBoundary`, `assertTaskStateJSONObjectKeys`, and the `assertTaskStateResolved*JSONEnvelope` helpers were left where they already are.
  - Reason: they were already package-shared rather than duplicated in both target files, so this pass did not need to move them.

## Risks / Deferred Cleanup
- `git diff --stat` and `git diff --numstat` do not show untracked additions, so the new helper file does not appear there until staged.
- This pass intentionally stopped at the single exact duplicate helper. Broader helper reshaping remains deferred because it would be organizational churn rather than duplication removal.
- A future test-only pass could decide whether status-only helpers like deferred-trigger fixture setup belong in the shared helper file, but there is no evidence pressure for that yet.

## Validation Commands And Results
- `gofmt -w internal/agent/tools/taskstate_test.go internal/agent/tools/taskstate_status_test.go internal/agent/tools/taskstate_readout_test_helpers_test.go`
  - result: completed successfully
- `git diff --check`
  - result: clean
- `go test -count=1 ./internal/agent/tools -run 'TestTaskStateOperator(Status|Inspect)'`
  - result: `ok  	github.com/local/picobot/internal/agent/tools	0.393s`
- `go test -count=1 ./internal/agent/tools`
  - result: `ok  	github.com/local/picobot/internal/agent/tools	8.657s`
- `go test -count=1 ./...`
  - result: passed across the repo
