# GC5-TREAT-008 Direct-Write Audit

Date: 2026-04-28

## Scope

Audit non-test direct file writes in `cmd` and `internal` before replacing any of them with atomic store helpers.

## Command

`rg -n "os\\.WriteFile|ioutil\\.WriteFile" --glob '*.go' --glob '!**/*_test.go' cmd internal`

## Findings

No non-test direct writes were found under `cmd`.

The remaining direct writes are not mission-store record writes:

| file | site | classification | decision |
| --- | --- | --- | --- |
| `internal/config/onboard.go` | `SaveConfig` | config bootstrap | Keep direct write for now; preserves config mode `0o640` and config path semantics. |
| `internal/config/onboard.go` | workspace bootstrap files | workspace seed output | Keep direct write; creates missing default docs only and preserves existing user edits. |
| `internal/config/onboard.go` | `memory/MEMORY.md` seed | workspace seed output | Keep direct write; creates missing default memory file only. |
| `internal/config/onboard.go` | embedded skill extraction | workspace seed output | Keep direct write; skips existing files and preserves embedded file bytes. |
| `internal/agent/tools/exec.go` | `.project_name` | local operator workspace marker | Keep direct write; scoped to `projects/current` setup, not mission-store state. |
| `internal/missioncontrol/*_capability.go` | local source file seeds | capability workspace seed output | Keep direct writes; source records and capability records already use `WriteStoreJSONAtomic`. |

Capability source seed writes reviewed:

- `camera_capability.go`: empty `camera/current_image.jpg`
- `contacts_capability.go`: `[]\n` at `contacts/contacts.json`
- `location_capability.go`: `{}\n` at `location/current_location.json`
- `microphone_capability.go`: empty `microphone/current_audio.wav`
- `sms_phone_capability.go`: `{}\n` at `sms_phone/current_source.json`
- `bluetooth_nfc_capability.go`: `{}\n` at `bluetooth_nfc/current_source.json`
- `broad_app_control_capability.go`: `{}\n` at `broad_app_control/current_source.json`

## Decision

No direct-write replacements are authorized by GC5-008.

The direct writes are local bootstrap or seed-file writes with intentional mode, skip-existing, or payload behavior. Mission-store JSON records in these flows already use `WriteStoreJSONAtomic`. Replacing the audited writes would be behavior work and should be done only with targeted tests for overwrite behavior, permissions, file content, and failure handling.

## Future Candidate

If direct writes are revisited, start with a small helper for "write missing seed file" that preserves:

- parent directory mode,
- file mode,
- skip-existing behavior,
- seed payload bytes,
- capability-specific error text.

Do not apply mission-store atomic helpers to arbitrary workspace seed files without proving the rename behavior is acceptable on the target filesystem.

## Validation

- `rg -n "os\\.WriteFile|ioutil\\.WriteFile" --glob '*.go' --glob '!**/*_test.go' cmd internal`
  - Result: 12 direct write sites, all classified above.
- `sed -n '1,220p' internal/config/onboard.go`
  - Result: config/workspace bootstrap semantics reviewed.
- `sed -n '360,400p' internal/agent/tools/exec.go`
  - Result: `.project_name` local marker semantics reviewed.
- Capability source initializer evidence was cross-checked with GC5-006.

No code changes were made for this row.
