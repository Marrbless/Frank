# GD2-TREAT-001 Progress Summary

Date: 2026-04-19

## Live state at pause

- Canonical repo: `/mnt/d/pbot/picobot`
- Current branch: `frank-v3-foundation`
- Current HEAD: `1f8705addba4451b9993d9fe6bf72c44106dd991`
- Ahead/behind `upstream/main`: `365 ahead / 0 behind`
- Test result: `go test -count=1 ./...` passed at this HEAD

## Treatment sequence

### 001A. Session path hardening + startup rehydration

- Landed across:
  - `1bb7f5e` `fix: harden session paths and reload sessions on startup`
  - `c417f21` `fix: harden session paths and reload sessions on startup`
- Files changed:
  - `internal/session/manager.go`
  - `internal/session/manager_test.go`
  - `internal/agent/loop.go`
  - `internal/agent/loop_test.go`
  - `docs/maintenance/garbage-day/GD2_TREAT_001A_SESSION_PATH_REHYDRATION_BEFORE.md`
  - `docs/maintenance/garbage-day/GD2_TREAT_001A_SESSION_PATH_REHYDRATION_AFTER.md`
- Exact risk removed:
  - removed the session-path traversal and filename-collision risk from deriving the on-disk filename directly from external `channel:chatID` values
  - removed the restart rehydration gap where session files were written but not loaded on startup
  - removed the fail-open startup behavior for malformed session files by making startup surface load failure instead of silently continuing

### 001B. Atomic overwrite writes

- Landed in:
  - `cf487e9` `fix: use atomic writes for session and memory overwrites`
- Files changed:
  - `internal/session/manager.go`
  - `internal/session/manager_test.go`
  - `internal/agent/memory/store.go`
  - `internal/agent/memory/store_test.go`
  - `internal/missioncontrol/store_fs.go`
  - `internal/missioncontrol/store_fs_test.go`
  - `docs/maintenance/garbage-day/GD2_TREAT_001B_ATOMIC_SESSION_MEMORY_WRITES_BEFORE.md`
  - `docs/maintenance/garbage-day/GD2_TREAT_001B_ATOMIC_SESSION_MEMORY_WRITES_AFTER.md`
- Exact risk removed:
  - removed the crash-window where session overwrites, long-memory overwrites, and validated memory-file overwrites could leave truncated or zero-length files because they used plain `os.WriteFile`
  - aligned those overwrite paths with temp-file + `Sync` + `Rename` + parent-dir sync semantics already used in missioncontrol

### 001C. Append durability + remember fail-closed behavior

- Landed in:
  - `31a11ca` `fix: fail closed when remember persistence fails`
- Files changed:
  - `internal/agent/memory/store.go`
  - `internal/agent/memory/store_test.go`
  - `internal/agent/loop.go`
  - `internal/agent/loop_remember_test.go`
  - `docs/maintenance/garbage-day/GD2_TREAT_001C_MEMORY_APPEND_REMEMBER_BEFORE.md`
  - `docs/maintenance/garbage-day/GD2_TREAT_001C_MEMORY_APPEND_REMEMBER_AFTER.md`
- Exact risk removed:
  - removed the false-success remember path that told the operator `"OK, I've remembered that."` even when persistence failed
  - removed the undurable daily-note append behavior that wrote without an explicit file `Sync` and discarded close errors

### 001D. Long-memory append replay/concurrency hardening

- Landed in:
  - `1f8705a` `fix: harden long-memory append retries and concurrency`
- Files changed:
  - `internal/agent/tools/write_memory.go`
  - `internal/agent/tools/write_memory_test.go`
  - `internal/agent/memory/store.go`
  - `internal/agent/memory/store_test.go`
  - `docs/maintenance/garbage-day/GD2_TREAT_001D_LONG_MEMORY_APPEND_BEFORE.md`
  - `docs/maintenance/garbage-day/GD2_TREAT_001D_LONG_MEMORY_APPEND_AFTER.md`
- Exact risk removed:
  - removed the tool-layer long-memory read-modify-write race within a single `MemoryStore` instance that could lose concurrent appends
  - removed immediate retry duplication at the file tail for the same append request
- Tail-idempotence semantics in 001D:
  - idempotence is intentionally narrow and tail-based only
  - a retried append is suppressed only when `MEMORY.md` already ends with the exact trailing segment `"\n" + content`
  - this is not global dedupe; the same content can still be appended again if it is no longer the current tail or if different content was appended in between

## Remaining durability-cluster risk

- Cross-process long-memory coordination is still open.
- The 001D mutex only serializes appends inside one `MemoryStore` instance in one process; separate processes or separate store instances can still race on the same `memory/MEMORY.md`.
- Long-memory idempotence is still content-tail based, not operation-ID based; there is still no request ID or append journal.
- Tail-idempotence is exact-string matching only; equivalent content with different whitespace or formatting is not deduped.

## Recommendation

- From the Round 2 diagnosis, the next disease cluster should be `GD2-TREAT-002`:
  - web and provider log-surface hardening in `internal/agent/tools/web.go` and `internal/providers/openai.go`
- Reason:
  - `GD2-TREAT-009` upstream integration work is already complete
  - `GD2-TREAT-001A` through `001D` now cover the planned session/memory durability subcluster
  - `GD2-TREAT-002` is the next explicitly recommended high-risk cluster in the diagnosis
