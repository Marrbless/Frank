# GD2-TREAT-001B Atomic Session Memory Writes Before

Date: 2026-04-19

## Live repo state

- Branch: `frank-v3-foundation`
- HEAD: `c417f2128d7088de9b620ddc5020b6ce0b7cbab3`
- Tags at HEAD: none
- Ahead/behind `upstream/main`: `362 ahead / 0 behind`
- `git status --short --branch`:

```text
## frank-v3-foundation
```

- Baseline `go test -count=1 ./...`: passed

## Overwrite write surfaces found

### Session overwrite writes

- `internal/session/manager.go`
  - `SessionManager.Save`
  - current behavior: `json.MarshalIndent` then `os.WriteFile(...)`

### Memory overwrite writes

- `internal/agent/memory/store.go`
  - `WriteLongTerm`
  - `WriteFile`
  - current behavior: `os.WriteFile(...)`

## Append write surfaces explicitly excluded

- `internal/agent/memory/store.go`
  - `AppendToday`
  - current behavior: `os.OpenFile(..., O_CREATE|O_APPEND|O_WRONLY)` plus `fmt.Fprintf`
- Excluded from this slice:
  - daily note append durability
  - remember shortcut behavior
  - replay-safe long-memory append semantics in higher layers

## Existing atomic helper/pattern selected for reuse

- Reuse `internal/missioncontrol/store_fs.go`
  - `WriteStoreFileAtomic(path string, data []byte) error`
- Existing behavior already matches the required pattern:
  - temp file created in target directory
  - file write
  - temp-file `Sync`
  - temp-file close before rename
  - `Rename` over target
  - parent directory `Sync`
  - temp-file cleanup on error
- Current expectation:
  - no new helper should be necessary
  - no change to `internal/missioncontrol/store_fs.go` should be necessary unless tests show package-boundary friction

## Exact implementation plan

1. Replace `SessionManager.Save` plain overwrite with `missioncontrol.WriteStoreFileAtomic` after JSON marshaling.
2. Replace `MemoryStore.WriteLongTerm` plain overwrite with `missioncontrol.WriteStoreFileAtomic`.
3. Replace `MemoryStore.WriteFile` plain overwrite with `missioncontrol.WriteStoreFileAtomic`.
4. Preserve current content bytes and current file permissions behavior as closely as possible within the existing helper semantics.
5. Leave append paths alone.
6. Add focused tests proving overwrite paths now use atomic replacement semantics or fail safely under injected helper failure.

## Exact tests planned

### `internal/session/manager_test.go`

- existing round-trip and path-hardening tests remain green
- new test: session `Save` uses atomic helper path and writes expected bytes
- new test: injected atomic helper failure leaves no partial/corrupt target replacement if testable

### `internal/agent/memory/store_test.go` / `store_persistence_test.go`

- new test: `WriteLongTerm` uses atomic helper path
- new test: `WriteFile` uses atomic helper path
- new test: injected atomic helper failure does not replace existing target contents
- existing long-term/today persistence tests remain green

### `internal/missioncontrol/store_fs_test.go`

- only if helper exposure/wrapping changes become necessary
- currently not planned

## Explicit non-goals

- no append-path durability changes
- no remember fail-closed changes
- no replay-safe long-memory append redesign
- no session key encoding changes
- no startup rehydration changes except incidental test compatibility
- no MCP changes
- no V4 work
- no broad cleanup
