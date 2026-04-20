# GD2-TREAT-001C Memory Append Remember After

Date: 2026-04-19

## `git diff --stat`

```text
 internal/agent/loop.go               |  6 +++--
 internal/agent/loop_remember_test.go | 47 ++++++++++++++++++++++++++++++++++++
 internal/agent/memory/store.go       | 19 ++++++++++++---
 internal/agent/memory/store_test.go  | 18 ++++++++++++++
 4 files changed, 85 insertions(+), 5 deletions(-)
```

Note: plain `git diff --stat` does not include new untracked docs. The new report files are listed explicitly below.

## `git diff --numstat`

```text
4	2	internal/agent/loop.go
47	0	internal/agent/loop_remember_test.go
16	3	internal/agent/memory/store.go
18	0	internal/agent/memory/store_test.go
```

Note: plain `git diff --numstat` does not include new untracked docs. The new report files are listed explicitly below.

## Files changed

- `internal/agent/memory/store.go`
- `internal/agent/memory/store_test.go`
- `internal/agent/loop.go`
- `internal/agent/loop_remember_test.go`
- `docs/maintenance/garbage-day/GD2_TREAT_001C_MEMORY_APPEND_REMEMBER_BEFORE.md` (new)
- `docs/maintenance/garbage-day/GD2_TREAT_001C_MEMORY_APPEND_REMEMBER_AFTER.md` (new)

## Exact append durability changes

- `internal/agent/memory/store.go`
  - added `memorySyncFile` seam defaulting to `(*os.File).Sync`
  - `AppendToday` now:
    - writes the appended line
    - explicitly `Sync`s the file
    - returns sync failure if syncing fails
    - closes the file explicitly and returns close failure if it occurs
  - removed the old deferred close that discarded close error
- File format was preserved:
  - appended line format remains `[%RFC3339 timestamp%] text\n`
- Overwrite-style persistence paths from 001B were left untouched.

## Exact remember failure behavior changes

- `internal/agent/loop.go`
  - remember shortcut still returns `"OK, I've remembered that."` on successful `AppendToday`
  - if `AppendToday` fails:
    - it no longer sends `"OK, I've remembered that."`
    - it now sends `"I couldn't remember that because saving memory failed."`
    - the same failure response is recorded in session history instead of a false success
- No unrelated loop behavior was changed.

## Exact tests added/changed

### `internal/agent/memory/store_test.go`

- Added `TestAppendTodaySyncFailureReturnsError`
  - proves append durability failure is surfaced as an error via the new sync seam

### `internal/agent/loop_remember_test.go`

- Existing success-path remember test still passes unchanged in behavior
- Added `TestAgentRememberFailsClosedWhenAppendTodayFails`
  - blocks the memory directory path so `AppendToday` fails deterministically
  - verifies failure response is emitted
  - verifies false success text is not recorded in session history

## Remaining replay/concurrency risks

- Long-memory append replay/lost-update risk remains.
- Current long-memory append behavior lives in `internal/agent/tools/write_memory.go` as read-modify-write:
  - read current `MEMORY.md`
  - concatenate new content
  - overwrite file
- That path was outside the allowed edit surface for this slice, so no concurrency hardening was applied here.
- A future treatment should address:
  - duplicate/retry append behavior
  - concurrent append lost updates
  - explicit idempotence expectations

## Validation commands and results

- `gofmt -w internal/agent/memory/store.go internal/agent/memory/store_test.go internal/agent/loop.go internal/agent/loop_remember_test.go`
  - passed
- `git diff --check`
  - passed
- `go test -count=1 ./internal/agent/memory`
  - passed
- `go test -count=1 ./internal/agent/tools`
  - passed
- `go test -count=1 ./internal/agent`
  - passed
- `go test -count=1 ./...`
  - passed
- `git status --short --branch`
  - only intended files for this slice were modified/new

## Remaining risks and deferred work

- Daily append now syncs and fails clearly, but append behavior still does not use a higher-level append journal or dedupe strategy.
- Long-memory append replay/lost-update hardening remains deferred because its implementation path was outside the allowed edit surface for this slice.
- Deferred work:
  - long-memory append replay-safe behavior
  - explicit concurrency hardening for long-memory append
  - any further append durability treatment beyond file sync semantics
