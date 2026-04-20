GC2-TREAT-002 main_test split after

- branch: `frank-v3-foundation`
- head: `ce7fcba0e83f1816b9025ac3604d138b889fbf1d`
- runtime behavior changed: no

Git diff --stat

```text
 cmd/picobot/main_test.go | 113 -----------------------------------------------
 1 file changed, 113 deletions(-)
```

Untracked moved test file diff stat

```text
 /dev/null => cmd/picobot/main_memory_test.go | 124 +++++++++++++++++++++++++++
 1 file changed, 124 insertions(+)
```

Git diff --numstat

```text
0	113	cmd/picobot/main_test.go
```

Untracked moved test file diff numstat

```text
124	0	/dev/null => cmd/picobot/main_memory_test.go
```

Files changed

- `cmd/picobot/main_test.go`
- `cmd/picobot/main_memory_test.go`
- `docs/maintenance/garbage-day/GC2_TREAT_002_MAIN_TEST_SPLIT_ASSESSMENT.md`
- `docs/maintenance/garbage-day/GC2_TREAT_002_MAIN_TEST_SPLIT_BEFORE.md`
- `docs/maintenance/garbage-day/GC2_TREAT_002_MAIN_TEST_SPLIT_AFTER.md`

Exact tests moved

- `TestMemoryCLI_ReadAppendWriteRecent`
- `TestMemoryCLI_Rank`

Tests and helpers intentionally left in `main_test.go`

- Scheduled-trigger governance tests and their persistence helpers stayed together because they share custom stores and governance-specific fixture wiring.
- Mission inspect tests stayed together because they depend on a large cluster of read-model capability fixtures and JSON assertion helpers.
- Mission status, mission assert, mission assert-step, mission set-step, mission bootstrap, mission runtime, mission watcher, and operator control tests stayed together because they exercise protected runtime-truth surfaces and share overlapping gateway/bootstrap/status helpers.
- Package, prune, and command-log tests stayed in place because they are adjacent to mission runtime/operator flows and were not needed for the safest first split.

Validation commands and results

- `gofmt -w cmd/picobot/main_test.go cmd/picobot/main_memory_test.go` -> passed
- `git diff --check` -> passed
- `go test -count=1 ./cmd/picobot` -> passed
- `go test -count=1 ./...` -> passed

Deferred next candidates from the `main_test.go` assessment

- Scheduled-trigger governance test family split
- Mission inspect test family split
- Mission status/assertion test family split
- Mission bootstrap/runtime/watcher test family split
