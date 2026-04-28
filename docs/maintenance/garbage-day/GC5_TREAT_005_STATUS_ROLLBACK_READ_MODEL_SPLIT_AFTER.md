# GC5-TREAT-005 Status Rollback Read-Model Split After

Date: 2026-04-28

## Scope

Split one cohesive rollback read-model family out of `internal/missioncontrol/status.go` into a same-package file.

## Changes

- Added `internal/missioncontrol/status_rollback.go`.
- Moved rollback and rollback-apply status identity types, `With*` identity helpers, public loaders, and private rollback loader/record projection helpers.
- Preserved package, exported names, JSON fields, loader behavior, validation order, and linkage checks.

## Size Delta

- `internal/missioncontrol/status.go`: `3634` lines.
- `internal/missioncontrol/status_rollback.go`: `254` lines.

## Validation

- `/usr/local/go/bin/go test -count=1 ./internal/missioncontrol -run 'TestLoadOperatorRollback|TestBuildCommittedMissionStatusSnapshotIncludesRollback|TestBuildOperatorV4SummaryStatus|TestWithV4SummaryIncludesRecoveryIdentities'`
  - Result: PASS (`ok`, 0.429s).
- `/usr/local/go/bin/go test -count=1 ./internal/missioncontrol`
  - Result: PASS (`ok`, 17.912s).

## Notes

- This was a mechanical same-package move only.
- No store paths, JSON formats, status states, or runtime behavior were intentionally changed.
