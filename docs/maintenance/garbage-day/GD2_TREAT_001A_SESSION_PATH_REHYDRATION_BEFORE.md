# GD2-TREAT-001A Session Path Rehydration Before

Date: 2026-04-19

## Live repo state

- Branch: `frank-v3-foundation`
- HEAD: `42f53a197b1d41d47f476c8344a8ad1f3124f76e`
- Tags at HEAD: none
- Ahead/behind `upstream/main`: `360 ahead / 0 behind`
- `git status --short --branch`:

```text
## frank-v3-foundation
```

- Baseline `go test -count=1 ./...`: passed

## Current session files/functions inspected

- `internal/session/manager.go`
  - `NewSessionManager`
  - `GetOrCreate`
  - `Save`
  - `LoadAll`
- `internal/agent/loop.go`
  - startup session manager construction at `NewAgentLoop`
  - session `GetOrCreate` / `Save` call sites during message handling
- `internal/agent/loop_test.go`
  - current focused loop construction/process test surface

## Current filename derivation behavior

- Session files are written under:
  - `filepath.Join(workspace, "sessions")`
- Final filename is currently derived directly from the raw logical key:
  - `filepath.Join(path, s.Key+".json")`
- The logical key is currently built from external operator session identifiers:
  - `msg.Channel + ":" + msg.ChatID`
- Current implications:
  - raw separators and traversal segments can influence the derived filename
  - no dedicated encoding/sanitization exists
  - no collision defense exists beyond raw string equality

## Current startup `LoadAll` behavior

- `SessionManager.LoadAll()` exists in `internal/session/manager.go`
- `NewAgentLoop` currently calls `session.NewSessionManager(workspace)` only
- No startup call to `LoadAll()` is present in the current construction path
- Saved session files therefore are not rehydrated into memory on restart
- Current corruption handling inside `LoadAll()` is silent skip:
  - unreadable file: skipped
  - invalid JSON: skipped
  - `MkdirAll` error is ignored

## Exact implementation plan

1. Add a deterministic, collision-safe filename encoding for session keys in `internal/session/manager.go`.
2. Centralize session path derivation behind helper(s) so all writes target only files inside `filepath.Join(workspace, "sessions")`.
3. Preserve the raw logical session key inside the JSON record; only the filename changes.
4. Treat empty session keys as invalid and return a clear error before writing.
5. Keep backward compatibility by continuing to load legacy session files that already exist inside `sessions/`, based on their JSON content rather than the legacy filename format.
6. Make `LoadAll()` explicit about corruption or unreadable session-file failures instead of silently swallowing them.
7. Call `LoadAll()` during `NewAgentLoop` startup and surface failure using the existing startup failure style used in that constructor.

## Exact tests planned

### `internal/session/manager_test.go`

- save/load round trip with a normal safe key
- traversal-like key cannot escape `sessions/`
- separator-heavy key cannot escape `sessions/`
- two distinct keys produce different filenames and distinct persisted sessions
- missing `sessions/` directory loads as empty without fatal error
- corrupted session file produces explicit failure
- legacy session file already inside `sessions/` still loads by JSON content

### `internal/agent/loop_test.go`

- startup rehydration test proving `NewAgentLoop` loads saved session state before use

## Explicit non-goals

- no atomic-write migration for session files in this slice
- no changes to memory persistence
- no changes to remember append behavior
- no MCP changes
- no V4 work
- no broad cleanup or unrelated refactors
