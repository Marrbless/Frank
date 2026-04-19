# GD2-TREAT-001 Session and Memory Durability Hardening Assessment

Date: 2026-04-19

## 1. Live repo state

- Canonical repo: `/mnt/d/pbot/picobot`
- Branch: `frank-v3-foundation`
- HEAD: `36cc7d07e36a129967dcc9ab722f47338236373c`
- Tags at HEAD: `frank-v3-foundation-upstream-sync-gd2`
- Ahead/behind `upstream/main`: `359 ahead / 0 behind` from `git rev-list --left-right --count HEAD...upstream/main`
- `go test -count=1 ./...`: passed
- Preconditions:
  - `docs/maintenance/garbage-day/ROUND_2_REPO_DIAGNOSIS.md` exists
  - HEAD contains `upstream/main`
  - worktree was clean before this assessment

## 2. File/package map

### Session package

- `internal/session/manager.go`

### Memory package

- Persistence-relevant: `internal/agent/memory/store.go`
- Non-persistence package files reviewed for scope boundary: `ranker.go`, `llm_ranker.go`
- Package tests reviewed:
  - `internal/agent/memory/store_test.go`
  - `internal/agent/memory/store_persistence_test.go`
  - `internal/agent/memory/ranker_test.go`
  - `internal/agent/memory/llm_ranker_test.go`
  - `internal/agent/memory/llm_ranker_logging_test.go`
  - `internal/agent/memory/llm_ranker_provider_integration_test.go`
- Note: checked-in `internal/agent/memory/*.md` files are repo artifacts, not the runtime persistence target. Runtime memory writes go to `<workspace>/memory/`.

### Relevant tool files

- `internal/agent/tools/memory.go`
- `internal/agent/tools/write_memory.go`

### Relevant loop file

- `internal/agent/loop.go`

### Relevant missioncontrol durability helpers

- `internal/missioncontrol/store_fs.go`
- `internal/missioncontrol/store_snapshot.go`

### Relevant tests

- `internal/agent/tools/memory_test.go`
- `internal/agent/tools/write_memory_test.go`
- `internal/agent/loop_remember_test.go`
- `internal/agent/loop_write_memory_test.go`
- `internal/agent/loop_processdirect_test.go`
- `internal/missioncontrol/store_fs_test.go`
- `internal/missioncontrol/store_snapshot_test.go`
- Session coverage gap: there are no `internal/session/*_test.go` files

## 3. Persistence surface inventory

| Surface | Data | Path source | Trust level | Method | Atomic | Temp+rename+fsync | Error handling | Replay / idempotence | Tests |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| `internal/session/manager.go:44-58` `SessionManager.Save` | session key + trimmed history JSON | `filepath.Join(workspace, "sessions", s.Key+".json")` | `workspace` trusted; `s.Key` derived from inbound `channel:chatID` and is not sanitized | `json.MarshalIndent` + `os.WriteFile(..., 0644)` after `os.MkdirAll(..., 0755)` | No | No | returns write/mkdir/marshal errors to caller | overwrites whole file; same payload is idempotent, but crash can leave truncated/partial file | none |
| `internal/session/manager.go:61-84` `SessionManager.LoadAll` | all session JSON files under workspace | `filepath.Join(workspace, "sessions")` | trusted root, but every file in dir is accepted | `os.ReadDir` + `os.ReadFile` + `json.Unmarshal` | N/A read path | N/A | `MkdirAll` error ignored; unreadable/corrupt files silently skipped | rehydrate only if called; no call sites found in scope | none |
| `internal/agent/memory/store.go:40-53` `NewMemoryStoreWithWorkspace` | memory dir bootstrap | `workspace + "/memory"` | workspace trusted | `os.MkdirAll(..., 0755)` | N/A | No | `MkdirAll` error ignored | repeated calls harmless | indirect only |
| `internal/agent/memory/store.go:122-131` `ReadLongTerm` | `MEMORY.md` text | `<workspace>/memory/MEMORY.md` | trusted workspace; fixed filename | `os.ReadFile` | N/A | N/A | returns `""` on not found; returns read errors otherwise | idempotent | `store_persistence_test.go` |
| `internal/agent/memory/store.go:135-140` `WriteLongTerm` | overwrite long-term memory text | `<workspace>/memory/MEMORY.md` | trusted workspace; fixed filename | `os.WriteFile(..., 0644)` after `MkdirAll` | No | No | returns errors | overwrite is idempotent only if caller sends same full content | `store_persistence_test.go`, `write_memory_test.go`, `memory_test.go` |
| `internal/agent/memory/store.go:144-154` `ReadToday` | today's note text | `<workspace>/memory/YYYY-MM-DD.md` | trusted workspace; fixed date-derived filename | `os.ReadFile` | N/A | N/A | returns `""` on not found; returns read errors otherwise | idempotent | `store_persistence_test.go` |
| `internal/agent/memory/store.go:158-170` `AppendToday` | timestamped line append to today's note | `<workspace>/memory/YYYY-MM-DD.md` | trusted workspace; fixed date-derived filename | `os.OpenFile(O_CREATE|O_APPEND|O_WRONLY, 0644)` + `fmt.Fprintf` | No | No | returns open/write errors; close error discarded | not idempotent; retries duplicate entries | `store_persistence_test.go`, `write_memory_test.go`, `loop_remember_test.go`, `loop_write_memory_test.go` |
| `internal/agent/memory/store.go:174-192` `GetRecentMemories` | recent daily note contents | fixed date-derived files under memory dir | trusted workspace | repeated `os.ReadFile` | N/A | N/A | missing files skipped; other read errors returned | idempotent | `store_persistence_test.go` |
| `internal/agent/memory/store.go:210-224` `ListFiles` | filenames under memory dir | memory dir | trusted workspace | `os.ReadDir` | N/A | N/A | missing dir returns empty list; other errors returned | idempotent | `memory_test.go` |
| `internal/agent/memory/store.go:230-241` `ReadFile` | validated memory file contents | `<workspace>/memory/<validated>` | validated `MEMORY.md` or `YYYY-MM-DD.md` only | `os.ReadFile` | N/A | N/A | invalid name rejected; missing file returns `""` | idempotent | `memory_test.go` |
| `internal/agent/memory/store.go:246-253` `WriteFile` | overwrite validated memory file | `<workspace>/memory/<validated>` | validated `MEMORY.md` or `YYYY-MM-DD.md` only | `os.WriteFile(..., 0644)` after `MkdirAll` | No | No | invalid name rejected; write/mkdir errors returned | overwrite idempotent only if same full content | `memory_test.go` |
| `internal/agent/memory/store.go:258-272` `DeleteFile` | delete dated note | `<workspace>/memory/YYYY-MM-DD.md` | validated dated filename only | `os.Remove` | OS-level rename semantics not used | No | not-found normalized to explicit error | repeated delete returns error, not idempotent | `memory_test.go` |
| `internal/agent/tools/write_memory.go:74-125` `WriteMemoryTool.Execute` target `today` | user/tool-supplied content | fixed today note path via `AppendToday` | target validated; content untrusted | delegates to `AppendToday` | No | No | propagates store error except heartbeat filter is special-cased | retries duplicate note lines | `write_memory_test.go`, `loop_write_memory_test.go`, `loop_processdirect_test.go` |
| `internal/agent/tools/write_memory.go:107-117` `WriteMemoryTool.Execute` target `long` append | previous `MEMORY.md` + new content | fixed long-term path via `ReadLongTerm` then `WriteLongTerm` | target validated; content untrusted | read-modify-write | No | No | propagates read/write errors | not idempotent; retries duplicate; concurrent writers can lose updates | `write_memory_test.go` |
| `internal/agent/tools/write_memory.go:118-122` `WriteMemoryTool.Execute` target `long` overwrite | full new long-term content | fixed long-term path | target validated; content untrusted | delegates to `WriteLongTerm` | No | No | propagates write errors | idempotent only for same content | `write_memory_test.go` |
| `internal/agent/tools/memory.go:148-177` `EditMemoryTool.Execute` | read/replace/write validated memory file | validated target via `resolveMemoryTarget` and `ReadFile` / `WriteFile` | target validated; replacement text untrusted | whole-file read/replace/overwrite | No | No | text-not-found returns explicit error; store errors propagated; heartbeat-like replacement silently returns success with empty response | not idempotent; concurrent writes can lose updates | `memory_test.go` |
| `internal/agent/tools/memory.go:95-111` `ReadMemoryTool.Execute` | read validated memory file | validated target | validated | delegates to `ReadFile` | N/A | N/A | missing or empty file collapsed to friendly string | idempotent | `memory_test.go` |
| `internal/agent/tools/memory.go:209-221` `DeleteMemoryTool.Execute` | delete dated note | validated target | validated | delegates to `DeleteFile` | N/A | N/A | explicit validation and delete errors | not idempotent | `memory_test.go` |
| `internal/agent/loop.go:1340-1364` remember shortcut | today's note append + user-facing confirmation | fixed today note path via `AppendToday` | note text untrusted; path fixed | direct append from loop | No | No | append error only logged; user still receives `"OK, I've remembered that."` | retries duplicate note lines | `loop_remember_test.go` |
| `internal/agent/loop.go:1297-1303`, `1317-1323`, `1356-1362`, `1512-1517` | session transcript save after command/approval/remember/final response | session key derived from external channel/chat ID | untrusted session key | direct `SessionManager.Save` call | No | No | save errors logged and swallowed | repeated save of same session state is idempotent; writes are not crash-safe | no direct session tests |
| `internal/agent/loop.go:1385`, `1566` | memory context reads for prompt building | fixed memory paths via store methods | trusted workspace; fixed filenames | `GetMemoryContext()` | N/A | N/A | returned error ignored, so agent continues without memory context | idempotent | indirect coverage only |
| `internal/missioncontrol/store_fs.go:17-65` and `internal/missioncontrol/store_snapshot.go:9-33` | reference atomic helper available in repo | arbitrary caller path | depends on caller | temp file in same dir + `Sync` + `Rename` + parent dir sync | Yes | Yes | errors returned, tested fail-closed | overwrite idempotence depends on caller | `store_fs_test.go`, `store_snapshot_test.go` |

### Atomic helper usage conclusion

- I found no call from `internal/session`, `internal/agent/memory`, `internal/agent/tools/memory.go`, `internal/agent/tools/write_memory.go`, or the reviewed `internal/agent/loop.go` paths into `missioncontrol.WriteStoreFileAtomic` or `WriteStoreJSONAtomic`.
- The durable temp-file + fsync + rename pattern exists in-repo, but these session/memory surfaces do not use it.

## 4. Durability risks

### High

#### R1. Session files are persisted but not reloaded on startup

- Confidence: High
- Evidence:
  - `internal/agent/loop.go:1096` creates `session.NewSessionManager(workspace)`
  - `internal/session/manager.go:61-84` implements `LoadAll`
  - repo search found no `LoadAll()` call sites in scope
- Affected files/functions:
  - `internal/agent/loop.go`
  - `internal/session/manager.go`
- Why it matters:
  - Session history is written to disk, but the current startup path never rehydrates it.
  - After restart, the on-disk session archive does not restore conversational context, so durability is weaker than the persistence surface implies.
- Smallest safe treatment:
  - Load sessions during agent startup, fail or warn clearly on load failure, and add restart rehydration coverage.

#### R2. Session file path is derived from unsanitized external session identifiers

- Confidence: Medium
- Evidence:
  - `internal/session/manager.go:53` uses `filepath.Join(path, s.Key+".json")`
  - `internal/agent/loop.go:1298`, `1318`, `1357`, `1382` build `s.Key` from `msg.Channel + ":" + msg.ChatID`
- Affected files/functions:
  - `internal/session/manager.go`
  - `internal/agent/loop.go`
- Why it matters:
  - If a channel or chat ID contains path separators, `..`, or an absolute-path prefix, the save target can escape the intended `sessions/` directory.
  - Impact ranges from failed saves to arbitrary file overwrite within the agent's filesystem permissions.
- Smallest safe treatment:
  - Replace raw filename derivation with a safe encoding or strict basename validation and add rejection tests for traversal/absolute path inputs.

#### R3. Overwrite-style session and memory writes are not crash-safe despite an existing atomic helper in-repo

- Confidence: High
- Evidence:
  - `internal/session/manager.go:54-58`
  - `internal/agent/memory/store.go:135-140`
  - `internal/agent/memory/store.go:246-253`
  - contrast with `internal/missioncontrol/store_fs.go:26-65`
- Affected files/functions:
  - `internal/session/manager.go` `Save`
  - `internal/agent/memory/store.go` `WriteLongTerm`, `WriteFile`
  - `internal/agent/tools/write_memory.go` long-memory append path
  - `internal/agent/tools/memory.go` edit path
- Why it matters:
  - Power loss, process crash, disk-full, or interrupted overwrite can leave truncated or zero-length files.
  - Session JSON corruption then cascades into silent session loss because `LoadAll` skips unreadable JSON.
- Smallest safe treatment:
  - Route overwrite writes through a shared atomic writer with temp-file, fsync, rename, and parent-dir sync semantics, then add failure-mode tests.

### Medium

#### R4. Daily note append path is not durable and the remember shortcut acknowledges success even when persistence fails

- Confidence: High
- Evidence:
  - `internal/agent/memory/store.go:164-170` appends without `Sync`
  - `internal/agent/loop.go:1346-1349` logs append failure but still replies `"OK, I've remembered that."`
- Affected files/functions:
  - `internal/agent/memory/store.go`
  - `internal/agent/loop.go`
- Why it matters:
  - The highest-volume write path can lose the last append on crash.
  - More importantly, the user can be told memory was saved when it was not.
- Smallest safe treatment:
  - Sync the file after append and make the remember shortcut fail closed with an explicit persistence error message.

#### R5. Long-term append is read-modify-write and is not replay-safe

- Confidence: High
- Evidence:
  - `internal/agent/tools/write_memory.go:107-117`
- Affected files/functions:
  - `internal/agent/tools/write_memory.go`
  - `internal/agent/memory/store.go`
- Why it matters:
  - Retries duplicate entries.
  - Concurrent writers can lose updates because both readers can start from the same old file contents and then overwrite each other.
- Smallest safe treatment:
  - Replace long-memory append mode with an append-oriented write path or serialize it with a process-local lock plus an idempotence strategy for retries.

#### R6. Read failures and corrupted session files are handled fail-open or silently skipped

- Confidence: High
- Evidence:
  - `internal/session/manager.go:65` ignores `MkdirAll` error
  - `internal/session/manager.go:74-80` silently skips unreadable or invalid JSON files
  - `internal/agent/loop.go:1385`, `1566` discard `GetMemoryContext()` errors
- Affected files/functions:
  - `internal/session/manager.go`
  - `internal/agent/loop.go`
- Why it matters:
  - Operators get no signal that persisted state was unreadable or omitted.
  - Silent degradation makes recovery and incident triage harder.
- Smallest safe treatment:
  - Aggregate and surface load failures, define a clear corrupted-file policy, and stop discarding memory-context read errors silently.

#### R7. Session and memory files are created world-readable

- Confidence: High
- Evidence:
  - `internal/session/manager.go:50`, `58`
  - `internal/agent/memory/store.go:52`, `136`, `164`, `250`, `253`
- Affected files/functions:
  - session and memory write/bootstrap paths
- Why it matters:
  - Session transcripts and long-term notes can contain operator instructions, personal notes, or secrets.
  - `0755` directories and `0644` files are permissive on shared hosts.
- Smallest safe treatment:
  - Tighten defaults to private-only perms (`0700` dirs, `0600` files) or a documented secure equivalent.

### Low

#### R8. Session and memory writes are symlink-sensitive

- Confidence: Medium
- Evidence:
  - all writes use `os.WriteFile`, `os.OpenFile`, and `filepath.Join` without no-follow protections
- Affected files/functions:
  - `internal/session/manager.go`
  - `internal/agent/memory/store.go`
- Why it matters:
  - If an attacker or local process can place symlinks under the workspace, writes can be redirected outside the intended tree.
- Smallest safe treatment:
  - Rebase these writes on a rooted filesystem/no-follow strategy or explicit `Lstat` validation before create/overwrite.

## 5. Security/path risks

### Path traversal

- Session path handling is unsafe today.
- `SessionManager.Save` trusts `s.Key` as part of a filename, and `s.Key` comes from external session identifiers in `loop.go`.
- Memory tool target handling is substantially safer:
  - `resolveMemoryTarget` only permits `today`, `long`, or `YYYY-MM-DD`
  - `ReadFile` / `WriteFile` / `DeleteFile` re-validate to `MEMORY.md` or `YYYY-MM-DD.md`

### Unsafe file permissions

- Session JSON and memory markdown are created with `0644`.
- Session and memory directories are created with `0755`.
- This is too permissive for conversation history and durable notes.

### Unsafe directory creation

- Session and memory directories are auto-created.
- Directory creation itself is straightforward, but the mode is permissive and `NewMemoryStoreWithWorkspace` ignores `MkdirAll` failure entirely.

### Symlink-sensitive writes

- Relevant for both session and memory surfaces.
- None of the scoped writes defend against symlinks or hard-link surprises.

### Accidental secret or PII persistence

- The memory system intentionally persists arbitrary user/tool content in plaintext markdown.
- There is a heartbeat-noise filter, but there is no redaction or secret classifier.
- Combined with `0644` perms, accidental sensitive persistence is a real exposure path.

### Logging of memory/session content

- I did not find direct `log.Printf` of session history or memory note contents in the scoped files.
- However, `internal/agent/loop.go:1424-1427` serializes raw tool arguments into user-visible activity notifications. For `write_memory`, that can echo sensitive note content back into the operator-facing channel.

### Untrusted input reaching file paths

- Yes for session files through `msg.Channel` / `msg.ChatID`.
- No material traversal path found for memory file names because tool targets are validated before file access.

## 6. Error handling risks

### Ignored errors

- `internal/agent/memory/store.go:52` ignores memory directory creation failure.
- `internal/session/manager.go:65` ignores `MkdirAll` error during load.
- `internal/agent/loop.go:1385`, `1566` ignore `GetMemoryContext()` errors.
- `internal/agent/memory/store.go:168` ignores file close error after append.

### Partial write risks

- Session JSON and overwrite-style memory writes are plain `os.WriteFile` writes without temp-file swap or fsync.
- Daily note appends are unsynced appends.

### JSON encode/decode handling

- Session JSON encode errors are returned from `Save`.
- Session JSON decode errors in `LoadAll` are silently skipped.
- No scoped session load path uses `DisallowUnknownFields`; malformed or drifted files are either skipped or, more importantly, never loaded because startup does not call `LoadAll`.

### Corrupted file behavior

- Corrupted session files are silently ignored if `LoadAll` is ever called.
- Corrupted memory markdown is treated as plain text; there is no structural corruption detector.

### Missing rejection codes / unclear errors

- Memory tool validation messages are reasonably explicit.
- The remember shortcut is not explicit on failure because it acknowledges success even when append fails.
- Session save failures are only logged; the user receives no persistence warning.

### Fail-open vs fail-closed

- Session save: fail-open to the user; reply still goes out.
- Remember shortcut memory write: fail-open to the user; reply still says it was remembered.
- Prompt memory context load: fail-open; agent continues without memory context.
- Write-memory tool execution: mostly fail-closed because store errors propagate back into tool execution.

## 7. Test gaps

### Missing or weak tests by scenario

- Corrupted JSON:
  - no session `LoadAll` tests for malformed JSON, unknown fields, or partial JSON
- Partial writes:
  - no session/memory tests that simulate short writes, fsync failures, or interrupted overwrite
- Missing directory:
  - no session tests for missing `sessions/`
  - no tests for `NewMemoryStoreWithWorkspace` bootstrap failure or later recovery
- Permission denied:
  - no scoped tests for unwritable `memory/` or `sessions/`
- Duplicate / replay write:
  - no tests for duplicate `AppendToday`
  - no tests for long-memory append retry duplication or lost-update behavior
- Concurrent writes:
  - no tests for concurrent `WriteLongTerm`, `WriteFile`, `EditMemoryTool`, or `SessionManager.Save`
- Path traversal / invalid path:
  - no tests that malicious session keys cannot escape `sessions/`
  - memory target validation has tests, but session filename validation has none because there is no validation
- Empty state:
  - basic empty memory reads are covered
  - empty session load/store behavior is not covered
- Large state:
  - no tests for large `MEMORY.md`, large session history payloads, or trimming behavior under large inputs

### Existing coverage summary

- Memory happy-path and basic validation coverage exists.
- Loop coverage exists for successful remember and `write_memory` flows.
- Missioncontrol atomic helper coverage exists.
- Session durability coverage is effectively absent.

## 8. Treatment backlog

### GD2-TREAT-001A

- Name: Session path hardening and restart rehydration
- Files touched:
  - `internal/session/manager.go`
  - `internal/agent/loop.go`
  - new session tests
- Exact smallest safe change:
  - sanitize or encode session keys before deriving filenames
  - call `LoadAll` during agent startup
  - surface startup load failure clearly
- Tests required:
  - restart rehydrate test
  - invalid/traversal session key rejection test
  - corrupted session file behavior test
- Risk:
  - High
- Must happen before V4:
  - Yes

### GD2-TREAT-001B

- Name: Atomic overwrite writes for session and memory files
- Files touched:
  - `internal/session/manager.go`
  - `internal/agent/memory/store.go`
  - possibly a shared atomic helper package or a safely extracted helper from missioncontrol
  - related tests
- Exact smallest safe change:
  - replace direct overwrite writes with temp-file + fsync + rename + parent-dir sync semantics
  - preserve current file formats and outward behavior
- Tests required:
  - write failure / fail-closed tests
  - parent-dir sync failure test
  - overwrite preserves previous good file on failure test
- Risk:
  - High
- Must happen before V4:
  - Yes

### GD2-TREAT-001C

- Name: Append durability, failure signaling, and replay-safe memory appends
- Files touched:
  - `internal/agent/memory/store.go`
  - `internal/agent/tools/write_memory.go`
  - `internal/agent/loop.go`
  - related tests
- Exact smallest safe change:
  - make `AppendToday` durable enough for this storage model
  - stop acknowledging remember-success when append fails
  - remove read-modify-write long-memory append or serialize it with explicit replay behavior
- Tests required:
  - permission denied on remember shortcut
  - duplicate/retry append behavior
  - concurrent append/write behavior
  - large note append
- Risk:
  - Medium
- Must happen before V4:
  - Yes

## Bottom line

- The repo already contains a stronger atomic persistence pattern in `internal/missioncontrol/store_fs.go`, but session and memory persistence have not adopted it.
- The most important findings are:
  - session persistence is currently not reloaded on startup
  - session filenames are derived from unsanitized external identifiers
  - overwrite writes are not atomic or fsynced
  - the remember shortcut can claim success when persistence failed
