# Write Boundaries

This document classifies durable writes in the repo. It is guidance for future cleanup and does not authorize broad rewrites by itself.

## Categories

| Category | Examples | Preferred write path | Notes |
| --- | --- | --- | --- |
| Mission store records | jobs, runtime state, audit events, approvals, hot-update, rollback, runtime-pack records | `missioncontrol.WriteStoreJSONAtomic` or a typed store helper that calls it | These are durable runtime truth and should be atomic. |
| Mission store files | projected status snapshots, packaged logs, control-adjacent files | `missioncontrol.WriteStoreFileAtomic` or typed store helper | Preserve existing file mode and path validation semantics. |
| Session and memory | `internal/session`, `internal/agent/memory` | atomic file helpers where practical | These are user/runtime state, but not every file has the same schema or permissions as mission store records. |
| Config bootstrap | `~/.picobot/config.json` | config-specific save helper | Config may contain plaintext credentials and intentionally uses restrictive file mode. Do not route blindly through mission-store helpers. |
| Workspace seed files | `AGENTS.md`, `SOUL.md`, `TOOLS.md`, initial memory file, embedded skill extraction | bootstrap-specific file creation | These are first-run seeds. Preserve "do not overwrite user edits" behavior. |
| Generated artifacts | local project files, imported skills, capability source placeholders | owning module helper | Validate path containment and overwrite policy at the module interface. |
| Runtime logs | current gateway log and packaged log bundles | store log helpers | Logs are operational evidence and can contain sensitive metadata; keep log rotation and packaging semantics centralized. |
| Tests and fixtures | malformed JSON, temp status files, fake stores | direct test writes are acceptable | Test writes should stay local to `t.TempDir()` or equivalent fixtures. |

## Rules

1. Runtime truth must not be written with an unreviewed direct `os.WriteFile`.
2. Do not convert config writes to mission-store writes without checking file permissions and secret-handling behavior.
3. Do not convert workspace seed writes if it would overwrite existing user files.
4. Do not combine write-path conversions with schema changes.
5. When replacing a direct write, add or preserve a test that proves path, mode, overwrite, and error behavior.
6. Treat append-only records, pointer switches, rollback records, and audit logs as protected surfaces.

## Session And Memory Expectations

Session and memory files are durable operator/user context, but they are not mission-store records and do not share the mission-store JSON schema contract.

- `internal/session` persists JSON session history below `workspace/sessions/`. `Save` must use the session atomic writer, must not leave root-level legacy files, and must preserve the previous session file when the atomic writer fails.
- `internal/session` may still read legacy session files inside `workspace/sessions/` for compatibility, but encoded filenames win when both encoded and legacy records exist for the same key.
- `internal/agent/memory` persists `memory/MEMORY.md` and dated memory notes. `WriteLongTerm` and `WriteFile` must use the memory atomic writer and preserve previous content when the atomic writer fails.
- `AppendToday` is append-oriented rather than replace-oriented; its durability expectation is explicit file sync/error propagation, not JSON record validation.

Current executable coverage:

- `go test -count=1 ./internal/session`
- `go test -count=1 ./internal/agent/memory`

## Current Non-Test Direct Write Hotspots

Known direct-write families from the 2026-04-30 assessment:

- `internal/config/onboard.go`: config and workspace bootstrap files.
- `internal/agent/tools/exec.go`: project metadata under the workspace.
- `internal/agent/skills/importer.go`: imported skill files.
- phone capability source initializers in `internal/missioncontrol/*_capability.go`.

These are not automatically wrong. Each should be audited against the categories above before conversion.

## Validation

For write-boundary changes, run the focused package tests first, then:

```sh
go vet ./...
go test -count=1 ./...
go test -count=1 -tags lite ./...
```

If the change affects shell scripts or phone deployment, also run:

```sh
sh scripts/termux/test-update-and-restart-frank
```
