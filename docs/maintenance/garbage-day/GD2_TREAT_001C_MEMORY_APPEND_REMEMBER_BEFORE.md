# GD2-TREAT-001C Memory Append Remember Before

Date: 2026-04-19

## Live repo state

- Branch: `frank-v3-foundation`
- HEAD: `cf487e929da41f93c321d2b0c2c7d3d57989cf53`
- Tags at HEAD: none
- Ahead/behind `upstream/main`: `363 ahead / 0 behind`
- `git status --short --branch`:

```text
## frank-v3-foundation
```

- Baseline `go test -count=1 ./...`: passed

## Append write surfaces found

### Daily memory append

- `internal/agent/memory/store.go`
  - `AppendToday`
  - current behavior:
    - `os.OpenFile(..., O_CREATE|O_APPEND|O_WRONLY, 0o644)`
    - `fmt.Fprintf(...)`
    - deferred close with close error ignored
    - no explicit `Sync`

### Remember shortcut path

- `internal/agent/loop.go`
  - remember regex shortcut path
  - current behavior:
    - calls `a.memory.AppendToday(note)`
    - logs error if append fails
    - still sends `"OK, I've remembered that."`
    - still records success text into session history

### Long-memory read-modify-write append

- implementation path exists in `internal/agent/tools/write_memory.go`
  - `ReadLongTerm()` then `WriteLongTerm(...)`
- This is a replay/lost-update risk surface, but it is outside the allowed edit surface for this slice.
- Result for this slice:
  - assess and document remaining risk
  - do not widen changes into that file

## Current remember failure behavior

- Fail-open.
- If `AppendToday` returns an error in the remember shortcut path, the loop still tells the user memory was saved:
  - outbound response remains `"OK, I've remembered that."`
  - session history also stores the same success text
- This is the main user-visible correctness bug in scope.

## Exact implementation plan

1. Harden `AppendToday` in `internal/agent/memory/store.go`:
   - add an explicit file sync step after the append write
   - stop discarding close error if no earlier error occurred
   - avoid logging memory contents
2. Add a narrow test seam for append sync failure so failure handling is executable rather than inferred.
3. Update the remember shortcut path in `internal/agent/loop.go` only:
   - if `AppendToday` fails, send a clear failure response
   - do not send `"OK, I've remembered that."`
   - do not record a false success in session history
4. Leave unrelated loop, provider, runtime, MCP, capability, treasury, campaign, approval, and V4 behavior untouched.
5. Document that long-memory append replay/lost-update risk remains deferred because its implementation path is outside the allowed edit surface for this slice.

## Exact tests planned

### `internal/agent/memory/store_test.go` / `store_persistence_test.go`

- append success still writes readable daily memory
- injected append durability failure returns an error
- no content-format regression for successful append

### `internal/agent/loop_remember_test.go`

- existing remember success path still returns `"OK, I've remembered that."`
- new failure-path test:
  - injected `AppendToday` failure produces a clear failure response
  - success text is not emitted
  - no false remembered content appears in persisted daily memory

## Explicit non-goals

- no change to session encoding or startup rehydration
- no change to overwrite atomic-write behavior from 001B
- no change to `write_memory` long-memory append implementation
- no full concurrency redesign for long-memory append
- no MCP changes
- no V4 work
- no broad cleanup
