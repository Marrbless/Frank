# GC4-TREAT-002 ProcessDirect Inspect Split After

Branch: `frank-garbage-day-gc4-002-processdirect-inspect-split`

## Completed

- Added `internal/agent/loop_processdirect_inspect_test.go`.
- Moved the five `TestProcessDirectInspectCommand...` tests out of the process-direct omnibus.
- Left shared helpers in `loop_processdirect_test.go`.
- Preserved test names, command strings, assertions, package, and runtime fixtures.

## Resulting Shape

- `internal/agent/loop_processdirect_test.go`: `10873` lines after the split.
- `internal/agent/loop_processdirect_inspect_test.go`: `177` lines.

## Validation

- `git diff --check`
- `/usr/local/go/bin/go test -count=1 ./internal/agent -run 'TestProcessDirectInspectCommand'`
- `/usr/local/go/bin/go test -count=1 ./internal/agent`
- `/usr/local/go/bin/go test -count=1 ./...`

## Remaining

The next cleanup candidate is `GC4-003`: reassess `internal/agent/tools/taskstate.go` and select the smallest same-package extraction that avoids runtime persistence-core risk.
