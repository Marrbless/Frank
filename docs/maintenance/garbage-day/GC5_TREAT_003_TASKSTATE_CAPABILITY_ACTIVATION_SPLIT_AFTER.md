# GC5-TREAT-003 TaskState Capability Activation Split After

Branch: `frank-garbage-day-gc4-005-maintenance-artifact-retention`

## Completed

- Added `internal/agent/tools/taskstate_capability_activation_test.go`.
- Moved the contiguous TaskState capability activation tests out of `taskstate_test.go`.
- Preserved test names, required-capability setup, hook assertions, fail-closed assertions, and shared helper usage.

## Resulting Shape

- `internal/agent/tools/taskstate_test.go`: `6013` lines after the split.
- `internal/agent/tools/taskstate_capability_activation_test.go`: `1305` lines.

## Validation

- `/usr/local/go/bin/go test -count=1 ./internal/agent/tools -run 'TestTaskStateActivateStep.*Capability'`
- `/usr/local/go/bin/go test -count=1 ./internal/agent/tools`

## Remaining

Continue with `GC5-004` from `REPO_WIDE_GARBAGE_CAMPAIGN_MATRIX.md`.
