# Garbage Day Baseline

## Facts
- Repo root: `/mnt/d/pbot/picobot`
- HEAD: `32e189ebaa168d607c6363596361dfe5e7c5c02a`
- Branch/status: `frank-v3-foundation`; dirty baseline with untracked `docs/FRANK_V4_SPEC.md`
- Tags at HEAD: `frank-v3-foundation-coherent-32e189e`, `frank-v3-telegram-only-provider-onboarding-32e189e`
- `docs/FRANK_V4_SPEC.md` exists: yes
- Baseline `go test -count=1 ./...` passed: yes
- Dirty files already present before cleanup:
  - `docs/FRANK_V4_SPEC.md` (untracked)

## Code shape
- Go package count: `14`
- Tracked file count: `253`
- Tracked Go file count: `214`
- Top 20 largest Go files by line count:
  - `10959` `cmd/picobot/main_test.go`
  - `7763` `internal/agent/tools/taskstate_test.go`
  - `3614` `internal/agent/tools/taskstate.go`
  - `3454` `internal/missioncontrol/treasury_registry_test.go`
  - `3213` `cmd/picobot/main.go`
  - `2728` `internal/agent/tools/frank_zoho_send_email_test.go`
  - `2717` `internal/agent/loop_processdirect_test.go`
  - `1955` `internal/agent/loop_checkin_test.go`
  - `1917` `internal/missioncontrol/identity_registry_test.go`
  - `1898` `internal/missioncontrol/runtime_test.go`
  - `1894` `internal/missioncontrol/status_test.go`
  - `1783` `internal/missioncontrol/step_validation_test.go`
  - `1741` `internal/missioncontrol/treasury_registry.go`
  - `1727` `internal/agent/loop.go`
  - `1708` `internal/missioncontrol/treasury_mutation_test.go`
  - `1558` `internal/missioncontrol/treasury_mutation.go`
  - `1553` `internal/agent/tools/taskstate_status_test.go`
  - `1373` `internal/missioncontrol/store_records.go`
  - `1315` `internal/missioncontrol/runtime.go`
  - `1256` `internal/agent/tools/frank_zoho_send_email.go`
- Directories with the most Go files:
  - `136` `internal/missioncontrol`
  - `28` `internal/agent/tools`
  - `11` `internal/agent`
  - `9` `internal/channels`
  - `9` `internal/agent/memory`
  - `7` `internal/providers`
  - `4` `internal/config`
  - `2` `internal/cron`
  - `2` `internal/agent/skills`
  - `2` `cmd/picobot`

## Garbage candidates
- Unused/dead-code candidates from tools or clear reference checks:
  - `cmd/picobot/main.go:1859` `writeJSONBytesAtomic` is a private single-call helper duplicating atomic write logic already provided by `internal/missioncontrol/store_fs.go`
  - `internal/missioncontrol/store_snapshot.go:40` `writeMissionStatusSnapshotFileAtomic` duplicates the store-layer atomic temp-file writer and has no direct references outside the default test hook wiring
- Commented-out code blocks:
  - `internal/agent/tools/memory.go:159` contains a commented-out `return` path that no longer documents behavior and leaves the live silent-skip branch harder to read
- Duplicated helpers/constants/string literals that are safe to consolidate:
  - Atomic JSON/file-write plumbing is duplicated in `cmd/picobot/main.go` and `internal/missioncontrol/store_snapshot.go`; both are local/private and mechanically replaceable with existing store-layer helpers
- Stale files that no longer appear connected to current V3/V4 truth:
  - No safe delete candidates proven from import/reference checks in this pass
- TODO/FIXME/HACK/XXX inventory, grouped by directory:
  - `docs`
    - `docs/HOW_TO_START.md:401`
    - `docs/CONFIG.md:205`
  - No Go-source TODO/FIXME/HACK/XXX hits were found
- Suspicious low-quality habits:
  - `panic` hit is limited to a test helper in `internal/missioncontrol/step_validation_test.go`; not a production-code issue
  - `fmt.Print*` hits are concentrated in CLI/user-onboarding flows in `cmd/picobot/main.go` and interactive WhatsApp setup in `internal/channels/whatsapp.go`; these are not treated as accidental debugging output without stronger evidence
  - Broad `map[string]interface{}` / `interface{}` use is widespread in tool/provider plumbing, chat metadata, and tests; this is real technical debt but not safe Garbage Day scope without a dedicated behavior lane
  - Large files exist in mission-control, tooling, and CLI tests; size alone is not enough evidence for cleanup-only surgery in this pass

## Protected surfaces
- `docs/FRANK_V4_SPEC.md` and the rest of `docs/` specs/handover material
- Zoho campaign email lane:
  - `internal/agent/tools/frank_zoho_*`
  - `internal/missioncontrol/store_frank_zoho_*`
  - `internal/missioncontrol/campaign_*`
- Zoho mailbox/bootstrap and completed outreach surfaces already wired through existing V3 flows
- Treasury lifecycle and transaction surfaces:
  - `internal/missioncontrol/treasury_*`
  - related status/preflight projections
- Capability onboarding and capability exposure:
  - `internal/missioncontrol/capability_*`
  - local/shared-storage-backed capability records
- Telegram owner-control onboarding and current channel wiring
- Mission-control semantics and current runtime/control-plane contracts

## Cleanup plan
- Remove the stale commented-out branch in `internal/agent/tools/memory.go` and keep the live silent-skip behavior explicit
- Replace `cmd/picobot/main.go`'s private atomic file-write helper with the existing store-layer atomic writer, preserving current error prefixes
- Replace `internal/missioncontrol/store_snapshot.go`'s duplicate atomic file-write implementation with the existing store-layer atomic writer via the existing injectable writer hook
- Run `gofmt` on touched Go files, then `git diff --check`, `go test -count=1 ./...`, and the targeted package tests requested by the prompt
- Explicit non-goals:
  - No new Frank V4 runtime behavior
  - No mission-control semantic changes
  - No provider/onboarding widening beyond current V3
  - No dependency additions
  - No architectural rewrites or broad interface-typing churn
