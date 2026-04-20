# GD2-TREAT-001D Long Memory Append After

Date: 2026-04-19

## `git diff --stat`

```text
 internal/agent/memory/store.go            | 26 +++++++++++++
 internal/agent/memory/store_test.go       | 56 +++++++++++++++++++++++++++
 internal/agent/tools/write_memory.go      |  8 +---
 internal/agent/tools/write_memory_test.go | 63 +++++++++++++++++++++++++++++++
 4 files changed, 146 insertions(+), 7 deletions(-)
```

Note: plain `git diff --stat` does not include new untracked docs. The new report files are listed explicitly below.

## `git diff --numstat`

```text
26	0	internal/agent/memory/store.go
56	0	internal/agent/memory/store_test.go
1	7	internal/agent/tools/write_memory.go
63	0	internal/agent/tools/write_memory_test.go
```

Note: plain `git diff --numstat` does not include new untracked docs. The new report files are listed explicitly below.

## Files changed

- `internal/agent/tools/write_memory.go`
- `internal/agent/tools/write_memory_test.go`
- `internal/agent/memory/store.go`
- `internal/agent/memory/store_test.go`
- `docs/maintenance/garbage-day/GD2_TREAT_001D_LONG_MEMORY_APPEND_BEFORE.md` (new)
- `docs/maintenance/garbage-day/GD2_TREAT_001D_LONG_MEMORY_APPEND_AFTER.md` (new)

## Exact replay/concurrency hardening added

- Added `MemoryStore.AppendLongTerm(content string)` in `internal/agent/memory/store.go`.
- `AppendLongTerm` now:
  - creates `memory/` if needed
  - serializes long-memory append work with the store mutex
  - reads current `MEMORY.md`
  - computes the trailing append segment as `"\n" + content`
  - treats an existing identical trailing segment as already-applied and returns success without rewriting
  - otherwise writes the updated content back using the existing 001B atomic overwrite path via `memoryWriteFileAtomic`
- `internal/agent/tools/write_memory.go` no longer performs its own tool-layer read-modify-write for `target == "long"` and `append == true`.
- The tool now delegates long-memory append semantics to `mem.AppendLongTerm(content)`.

## Exact idempotence behavior defined

- Defined rule:
  - a retried long-memory append with the same `content` is idempotent if the current `MEMORY.md` already ends with the exact trailing segment `"\n" + content`
  - in that case, no duplicate tail append is written
- Consequence:
  - the same append request retried immediately after success does not duplicate the same trailing content
  - concurrent appends through the same `MemoryStore` instance no longer lose updates due to tool-layer read-modify-write races
- Intentionally not defined as global dedupe:
  - if the same content appears earlier in the file but not at the tail, a new append is still allowed
  - if different content is appended in between, a later append of the same content is treated as a new append

## Exact tests added/changed

### `internal/agent/memory/store_test.go`

- Added `TestAppendLongTermRetryIsIdempotentAtTail`
- Added `TestAppendLongTermConcurrentAppendsPreserveBothValues`

### `internal/agent/tools/write_memory_test.go`

- Added `TestWriteMemoryTool_LongAppendRetryIsIdempotentAtTail`
- Added `TestWriteMemoryTool_LongAppendConcurrentCallsPreserveBothValues`
- Existing write-memory success and overwrite behavior tests still pass

## Remaining risks

- Cross-process concurrency is still not hardened:
  - the new protection serializes appends within the same `MemoryStore` instance only
  - separate processes or separate store instances writing the same `MEMORY.md` concurrently can still race
- Retry dedupe is intentionally narrow:
  - it only suppresses a duplicate retry when the target file already ends with the exact trailing append segment
  - it does not attempt semantic dedupe of equivalent content with whitespace or formatting differences
- There is still no request ID or append journal:
  - idempotence is content-tail based, not operation-ID based

## Validation commands and results

- `gofmt -w internal/agent/tools/write_memory.go internal/agent/tools/write_memory_test.go internal/agent/memory/store.go internal/agent/memory/store_test.go`
  - passed
- `git diff --check`
  - passed
- `go test -count=1 ./internal/agent/tools`
  - passed
- `go test -count=1 ./internal/agent/memory`
  - passed
- `go test -count=1 ./internal/agent`
  - passed
- `go test -count=1 ./...`
  - passed
- `git status --short --branch`
  - only intended files for this slice were modified/new
