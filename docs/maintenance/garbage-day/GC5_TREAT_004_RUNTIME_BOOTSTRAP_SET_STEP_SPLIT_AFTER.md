# GC5-TREAT-004 Runtime Bootstrap Set-Step Split After

Branch: `frank-garbage-day-gc4-005-maintenance-artifact-retention`

## Completed

- Added `cmd/picobot/main_runtime_bootstrap_set_step_test.go`.
- Moved set-step, mission step control-file, watcher, and operator set-step tests out of `main_runtime_bootstrap_test.go`.
- Left approval, durable runtime hydration, persistence, and non-set-step bootstrap tests in the existing runtime bootstrap file.
- Preserved test names, assertions, helper usage, command flags, and command strings.

## Resulting Shape

- `cmd/picobot/main_runtime_bootstrap_test.go`: `4407` lines after the split.
- `cmd/picobot/main_runtime_bootstrap_set_step_test.go`: `2118` lines.

## Validation

- `/usr/local/go/bin/go test -count=1 ./cmd/picobot -run 'TestMissionSetStep|TestApplyMissionStepControl|TestRestoreMissionStepControl|TestWatchMissionStepControl|TestMissionOperatorSetStep'`
- `/usr/local/go/bin/go test -count=1 ./cmd/picobot`

## Remaining

Continue with `GC5-005` from `REPO_WIDE_GARBAGE_CAMPAIGN_MATRIX.md`.
