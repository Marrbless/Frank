# GD2-TREAT-001B Atomic Session Memory Writes After

Date: 2026-04-19

## `git diff --stat`

```text
 internal/agent/memory/store.go           |  10 ++-
 internal/agent/memory/store_test.go      | 114 +++++++++++++++++++++++++++++++
 internal/missioncontrol/store_fs.go      |  13 ++++
 internal/missioncontrol/store_fs_test.go |  17 +++++
 internal/session/manager.go              |  15 +++-
 internal/session/manager_test.go         |  92 +++++++++++++++++++++++++
 6 files changed, 258 insertions(+), 3 deletions(-)
```

Note: plain `git diff --stat` does not list new untracked docs. The new report files are listed explicitly below.

## `git diff --numstat`

```text
8	2	internal/agent/memory/store.go
114	0	internal/agent/memory/store_test.go
13	0	internal/missioncontrol/store_fs.go
17	0	internal/missioncontrol/store_fs_test.go
14	1	internal/session/manager.go
92	0	internal/session/manager_test.go
```

Note: plain `git diff --numstat` does not list new untracked docs. The new report files are listed explicitly below.

## Files changed

- `internal/session/manager.go`
- `internal/session/manager_test.go`
- `internal/agent/memory/store.go`
- `internal/agent/memory/store_test.go`
- `internal/missioncontrol/store_fs.go`
- `internal/missioncontrol/store_fs_test.go`
- `docs/maintenance/garbage-day/GD2_TREAT_001B_ATOMIC_SESSION_MEMORY_WRITES_BEFORE.md` (new)
- `docs/maintenance/garbage-day/GD2_TREAT_001B_ATOMIC_SESSION_MEMORY_WRITES_AFTER.md` (new)

## Exact atomic write helper/path used

- Reused the existing missioncontrol atomic temp-file pattern in:
  - `internal/missioncontrol/store_fs.go`
- Added the smallest shared wrapper needed to preserve existing overwrite file mode:
  - `WriteStoreFileAtomicMode(path string, data []byte, mode os.FileMode) error`
- Wrapper behavior:
  - create temp file in target directory
  - optionally `Chmod` temp file to requested mode
  - write bytes
  - `Sync` temp file
  - close temp file
  - `Rename` over target path
  - `Sync` parent directory
  - remove temp file on error
- Existing `WriteStoreFileAtomic` behavior remains available and unchanged for existing callers.

## Exact overwrite writes converted

### Session overwrite writes converted

- `internal/session/manager.go`
  - `SessionManager.Save`
  - old: `os.WriteFile(fpath, b, 0644)`
  - new: package-local seam `sessionWriteFileAtomic`, defaulting to `missioncontrol.WriteStoreFileAtomicMode(fpath, b, 0o644)`

### Memory overwrite writes converted

- `internal/agent/memory/store.go`
  - `WriteLongTerm`
  - old: `os.WriteFile(path, []byte(content), 0o644)`
  - new: package-local seam `memoryWriteFileAtomic`, defaulting to `missioncontrol.WriteStoreFileAtomicMode(path, []byte(content), 0o644)`
- `internal/agent/memory/store.go`
  - `WriteFile`
  - old: `os.WriteFile(filepath.Join(s.memoryDir, name), []byte(content), 0o644)`
  - new: `memoryWriteFileAtomic(...)` with the same `0o644` target mode

## Exact append writes intentionally left alone

- `internal/agent/memory/store.go`
  - `AppendToday`
  - still uses append open/write semantics
- Higher-layer append flows intentionally unchanged:
  - `write_memory` append-to-long behavior
  - remember shortcut behavior
  - daily note append durability

## Exact tests added/changed

### `internal/session/manager_test.go`

- Added `TestSessionManagerSaveUsesAtomicWriter`
- Added `TestSessionManagerSaveAtomicFailurePreservesExistingFile`
- Added `TestSessionManagerLoadAllIgnoresAtomicTempFiles`
- Existing 001A tests remained in place and still pass:
  - round-trip load
  - traversal/separator safety
  - distinct-key non-collision
  - missing directory load
  - explicit corrupted JSON failure
  - legacy compatibility behavior

### `internal/agent/memory/store_test.go`

- Added `TestWriteLongTermUsesAtomicWriter`
- Added `TestWriteFileUsesAtomicWriter`
- Added `TestWriteLongTermAtomicFailurePreservesExistingContent`
- Added `TestWriteFileAtomicFailurePreservesExistingContent`

### `internal/missioncontrol/store_fs_test.go`

- Added `TestWriteStoreFileAtomicModePreservesRequestedFileMode`

## Validation commands and results

- `gofmt -w internal/session/manager.go internal/session/manager_test.go internal/agent/memory/store.go internal/agent/memory/store_test.go internal/missioncontrol/store_fs.go internal/missioncontrol/store_fs_test.go`
  - passed
- `git diff --check`
  - passed
- `go test -count=1 ./internal/session`
  - passed
- `go test -count=1 ./internal/agent/memory`
  - passed
- `go test -count=1 ./internal/missioncontrol`
  - passed
- `go test -count=1 ./internal/agent`
  - passed
- `go test -count=1 ./internal/agent/tools`
  - passed
- `go test -count=1 ./...`
  - passed
- `git status --short --branch`
  - only intended files for this slice were modified/new

## Remaining risks

- Append-style writes are still not made durable in this slice.
- Session and memory overwrite callers now use atomic replacement, but startup/read paths can still observe temp-file artifacts from interrupted writes unless they explicitly ignore them.
  - Session rehydration now ignores `.json.tmp-*` atomic temp files.
  - Memory file enumeration/read paths were not widened in this slice because overwrite persistence there does not scan the directory during startup.
- Long-memory append semantics in higher layers are still read-modify-write and replay-prone; this slice intentionally did not redesign that path.

## Deferred work

- `GD2-TREAT-001C`: append durability
- remember fail-closed behavior
- replay-safe long-memory append behavior
