# GC4-TREAT-001 Status Autonomy Split After

Branch: `frank-garbage-day-gc4-001-status-autonomy-split`

## Completed

- Added `internal/missioncontrol/status_autonomy.go`.
- Moved the V4 autonomy identity status types, wrapper, loaders, record adapters, and last-error helpers out of `status.go`.
- Preserved same-package visibility and exported names.
- Did not change JSON field names, status states, store paths, validation calls, or read-model linkage behavior.

## Validation

- `git diff --check`
- `/usr/local/go/bin/go test -count=1 ./internal/missioncontrol -run 'TestLoadOperatorAutonomyIdentityStatus|TestBuildCommittedMissionStatusSnapshotIncludesAutonomyIdentity|TestOperatorV4Summary'`
- `/usr/local/go/bin/go test -count=1 ./internal/missioncontrol`
- `/usr/local/go/bin/go test -count=1 ./...`

## Remaining

The next cleanup candidate remains `GC4-002`: split one command family out of `internal/agent/loop_processdirect_test.go`.
