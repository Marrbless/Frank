# GC4-TREAT-004 Bootstrap Store Root Split After

Branch: `frank-garbage-day-gc4-004-bootstrap-store-root-split`

## Completed

- Added `cmd/picobot/main_runtime_bootstrap_store_root_test.go`.
- Moved the three `TestResolveMissionStoreRoot...` tests out of the runtime bootstrap omnibus.
- Left `newMissionBootstrapTestCommand` and shared helpers in the existing bootstrap test file.
- Preserved test names, assertions, flag setup, and missioncontrol parity checks.

## Resulting Shape

- `cmd/picobot/main_runtime_bootstrap_test.go`: `6510` lines after the split.
- `cmd/picobot/main_runtime_bootstrap_store_root_test.go`: `52` lines.

## Validation

- `git diff --check`
- `/usr/local/go/bin/go test -count=1 ./cmd/picobot -run 'TestResolveMissionStoreRoot'`
- `/usr/local/go/bin/go test -count=1 ./cmd/picobot`
- `/usr/local/go/bin/go test -count=1 ./...`

## Remaining

The final matrix row is `GC4-005`: define maintenance artifact retention/prune policy without deleting files.
