# Garbage Day Phase 1 Before/After

## Visual TLDR

| BEFORE PHASE 1 | AFTER PHASE 1 |
| --- | --- |
| large `TaskState` production/test files | `TaskState` readout cluster extracted |
| repeated helper and fixture blocks | repeated capability fixtures consolidated |
| duplicate atomic-write style plumbing | shared-storage fixtures consolidated |
| no clear Garbage Day doc index | malformed treasury helper deduped |
| tests passing but repo health still poor | tests still passing, but large files still remain |

Citations: `GARBAGE_DAY_BASELINE.md`, `GARBAGE_DAY_AFTER.md`, `GARBAGE_DAY_PASS_2_TASKSTATE_ASSESSMENT.md`, `GARBAGE_DAY_PASS_3_TASKSTATE_READOUT_AFTER.md`, `GARBAGE_DAY_PASS_4_TASKSTATE_READOUT_TEST_HELPERS_AFTER.md`, `GARBAGE_DAY_PASS_5_TASKSTATE_CAPABILITY_PROPOSAL_FIXTURES_AFTER.md`, `GARBAGE_DAY_PASS_6_TASKSTATE_CAPABILITY_CONFIG_FIXTURES_AFTER.md`, `GARBAGE_DAY_PASS_7_TASKSTATE_SHARED_STORAGE_CONFIG_FIXTURES_AFTER.md`, `GARBAGE_DAY_PASS_8_TASKSTATE_SHARED_STORAGE_EXPOSURE_FIXTURES_AFTER.md`, `GARBAGE_DAY_SUMMARY.md`.

## Before/After Table

| Area | Before | After | Source artifacts |
| --- | --- | --- | --- |
| Repo package count | `14` packages | `14` packages | `GARBAGE_DAY_BASELINE.md`, live `go list ./...` |
| Tracked file count | `253` | `272` | `GARBAGE_DAY_BASELINE.md`, live `git ls-files \| wc -l`, `GARBAGE_DAY_SUMMARY.md` |
| Tracked Go file count | `214` | `217` | `GARBAGE_DAY_BASELINE.md`, live `git ls-files '*.go' \| wc -l`, `GARBAGE_DAY_SUMMARY.md` |
| Atomic-write duplication | duplicate private atomic writers in `cmd/picobot/main.go` and `internal/missioncontrol/store_snapshot.go` | replaced with shared store-layer writer | `GARBAGE_DAY_BASELINE.md`, `GARBAGE_DAY_AFTER.md` |
| TaskState readout production code | readout logic embedded inside `internal/agent/tools/taskstate.go` | readout cluster moved to `internal/agent/tools/taskstate_readout.go` | `GARBAGE_DAY_PASS_2_TASKSTATE_ASSESSMENT.md`, `GARBAGE_DAY_PASS_3_TASKSTATE_READOUT_BEFORE.md`, `GARBAGE_DAY_PASS_3_TASKSTATE_READOUT_AFTER.md` |
| Readout malformed treasury helper | duplicate helper in two test files | shared helper file added | `GARBAGE_DAY_PASS_4_TASKSTATE_READOUT_TEST_HELPERS_BEFORE.md`, `GARBAGE_DAY_PASS_4_TASKSTATE_READOUT_TEST_HELPERS_AFTER.md` |
| Capability proposal fixtures | repeated proposal writers inline in `taskstate_test.go` | shared proposal helper file introduced | `GARBAGE_DAY_PASS_5_TASKSTATE_CAPABILITY_PROPOSAL_FIXTURES_BEFORE.md`, `GARBAGE_DAY_PASS_5_TASKSTATE_CAPABILITY_PROPOSAL_FIXTURES_AFTER.md` |
| Capability config fixtures | repeated workspace config writers inline in `taskstate_test.go` | shared config helper introduced | `GARBAGE_DAY_PASS_6_TASKSTATE_CAPABILITY_CONFIG_FIXTURES_BEFORE.md`, `GARBAGE_DAY_PASS_6_TASKSTATE_CAPABILITY_CONFIG_FIXTURES_AFTER.md` |
| Shared-storage config fixtures | one remaining inline shared-storage config block | routed through wrapper helper | `GARBAGE_DAY_PASS_7_TASKSTATE_SHARED_STORAGE_CONFIG_FIXTURES_BEFORE.md`, `GARBAGE_DAY_PASS_7_TASKSTATE_SHARED_STORAGE_CONFIG_FIXTURES_AFTER.md` |
| Shared-storage exposure fixtures | repeated `StoreWorkspaceSharedStorageCapabilityExposure(root, workspace)` blocks | thin shared helper introduced | `GARBAGE_DAY_PASS_8_TASKSTATE_SHARED_STORAGE_EXPOSURE_FIXTURES_BEFORE.md`, `GARBAGE_DAY_PASS_8_TASKSTATE_SHARED_STORAGE_EXPOSURE_FIXTURES_AFTER.md` |
| Garbage Day docs surface | flat raw reports, no durable index | indexed and consolidated under `docs/maintenance/garbage-day/` | `GARBAGE_DAY_SUMMARY.md`, this directory |

## Before/After Line Counts

| File | Before | After | Source artifacts |
| --- | --- | --- | --- |
| `internal/agent/tools/taskstate.go` | `3614` | `3343` | `GARBAGE_DAY_BASELINE.md`, `GARBAGE_DAY_PASS_3_TASKSTATE_READOUT_AFTER.md`, live `wc -l` |
| `internal/agent/tools/taskstate_readout.go` | absent | `282` | `GARBAGE_DAY_PASS_3_TASKSTATE_READOUT_BEFORE.md`, `GARBAGE_DAY_PASS_3_TASKSTATE_READOUT_AFTER.md`, live `wc -l` |
| `internal/agent/tools/taskstate_test.go` | `7763` | `7346` | `GARBAGE_DAY_BASELINE.md`, `GARBAGE_DAY_PASS_4_TASKSTATE_READOUT_TEST_HELPERS_AFTER.md`, `GARBAGE_DAY_PASS_5_TASKSTATE_CAPABILITY_PROPOSAL_FIXTURES_AFTER.md`, `GARBAGE_DAY_PASS_6_TASKSTATE_CAPABILITY_CONFIG_FIXTURES_AFTER.md`, `GARBAGE_DAY_PASS_7_TASKSTATE_SHARED_STORAGE_CONFIG_FIXTURES_AFTER.md`, `GARBAGE_DAY_PASS_8_TASKSTATE_SHARED_STORAGE_EXPOSURE_FIXTURES_AFTER.md`, live `wc -l` |
| `internal/agent/tools/taskstate_status_test.go` | `1553` | `1531` | `GARBAGE_DAY_BASELINE.md`, `GARBAGE_DAY_PASS_4_TASKSTATE_READOUT_TEST_HELPERS_AFTER.md`, live `wc -l` |
| `internal/agent/tools/taskstate_readout_test_helpers_test.go` | absent | `29` | `GARBAGE_DAY_PASS_4_TASKSTATE_READOUT_TEST_HELPERS_BEFORE.md`, `GARBAGE_DAY_PASS_4_TASKSTATE_READOUT_TEST_HELPERS_AFTER.md`, live `wc -l` |
| `internal/agent/tools/taskstate_capability_test_helpers_test.go` | absent | `264` | `GARBAGE_DAY_PASS_5_TASKSTATE_CAPABILITY_PROPOSAL_FIXTURES_AFTER.md`, `GARBAGE_DAY_PASS_6_TASKSTATE_CAPABILITY_CONFIG_FIXTURES_AFTER.md`, `GARBAGE_DAY_PASS_7_TASKSTATE_SHARED_STORAGE_CONFIG_FIXTURES_AFTER.md`, `GARBAGE_DAY_PASS_8_TASKSTATE_SHARED_STORAGE_EXPOSURE_FIXTURES_AFTER.md`, live `wc -l` |

## Package / File / Go-File Count Changes

| Metric | Before | After | Evidence |
| --- | --- | --- | --- |
| Packages | `14` | `14` | `GARBAGE_DAY_BASELINE.md`, live `go list ./...` |
| Tracked files | `253` | `272` | `GARBAGE_DAY_BASELINE.md`, `GARBAGE_DAY_SUMMARY.md`, live `git ls-files \| wc -l` |
| Tracked Go files | `214` | `217` | `GARBAGE_DAY_BASELINE.md`, `GARBAGE_DAY_SUMMARY.md`, live `git ls-files '*.go' \| wc -l` |

## Production Files Changed In Phase 1

- `cmd/picobot/main.go`
- `internal/agent/tools/memory.go`
- `internal/missioncontrol/store_snapshot.go`
- `internal/agent/tools/taskstate.go`
- `internal/agent/tools/taskstate_readout.go`

Citations: `GARBAGE_DAY_AFTER.md`, `GARBAGE_DAY_SUMMARY.md`.

## Test / Helper Files Changed In Phase 1

- `internal/agent/tools/taskstate_test.go`
- `internal/agent/tools/taskstate_status_test.go`
- `internal/agent/tools/taskstate_readout_test_helpers_test.go`
- `internal/agent/tools/taskstate_capability_test_helpers_test.go`

Citations: `GARBAGE_DAY_PASS_4_TASKSTATE_READOUT_TEST_HELPERS_AFTER.md`, `GARBAGE_DAY_PASS_5_TASKSTATE_CAPABILITY_PROPOSAL_FIXTURES_AFTER.md`, `GARBAGE_DAY_PASS_6_TASKSTATE_CAPABILITY_CONFIG_FIXTURES_AFTER.md`, `GARBAGE_DAY_PASS_7_TASKSTATE_SHARED_STORAGE_CONFIG_FIXTURES_AFTER.md`, `GARBAGE_DAY_PASS_8_TASKSTATE_SHARED_STORAGE_EXPOSURE_FIXTURES_AFTER.md`, `GARBAGE_DAY_SUMMARY.md`.

## Docs Added In Phase 1

- `docs/maintenance/GARBAGE_DAY_BASELINE.md`
- `docs/maintenance/GARBAGE_DAY_AFTER.md`
- `docs/maintenance/GARBAGE_DAY_PASS_2_TASKSTATE_ASSESSMENT.md`
- `docs/maintenance/GARBAGE_DAY_PASS_3_TASKSTATE_READOUT_BEFORE.md`
- `docs/maintenance/GARBAGE_DAY_PASS_3_TASKSTATE_READOUT_AFTER.md`
- `docs/maintenance/GARBAGE_DAY_PASS_4_TASKSTATE_READOUT_TEST_HELPERS_BEFORE.md`
- `docs/maintenance/GARBAGE_DAY_PASS_4_TASKSTATE_READOUT_TEST_HELPERS_AFTER.md`
- `docs/maintenance/GARBAGE_DAY_PASS_5_TASKSTATE_CAPABILITY_PROPOSAL_FIXTURES_BEFORE.md`
- `docs/maintenance/GARBAGE_DAY_PASS_5_TASKSTATE_CAPABILITY_PROPOSAL_FIXTURES_AFTER.md`
- `docs/maintenance/GARBAGE_DAY_PASS_6_TASKSTATE_CAPABILITY_CONFIG_FIXTURES_BEFORE.md`
- `docs/maintenance/GARBAGE_DAY_PASS_6_TASKSTATE_CAPABILITY_CONFIG_FIXTURES_AFTER.md`
- `docs/maintenance/GARBAGE_DAY_PASS_7_TASKSTATE_SHARED_STORAGE_CONFIG_FIXTURES_BEFORE.md`
- `docs/maintenance/GARBAGE_DAY_PASS_7_TASKSTATE_SHARED_STORAGE_CONFIG_FIXTURES_AFTER.md`
- `docs/maintenance/GARBAGE_DAY_PASS_8_TASKSTATE_SHARED_STORAGE_EXPOSURE_FIXTURES_BEFORE.md`
- `docs/maintenance/GARBAGE_DAY_PASS_8_TASKSTATE_SHARED_STORAGE_EXPOSURE_FIXTURES_AFTER.md`
- `docs/maintenance/GARBAGE_DAY_SUMMARY.md` (`reconciled as present but currently untracked in live repo state`)

Citations: live `git ls-files docs/maintenance`, live `git status --short --branch`, `GARBAGE_DAY_SUMMARY.md`.

## Cleanup Themes By Pass

| Pass | Theme | Net effect | Source artifacts |
| --- | --- | --- | --- |
| Baseline / original cleanup | remove duplicate atomic-write plumbing and stale commented code | simplified three production files without widening behavior | `GARBAGE_DAY_BASELINE.md`, `GARBAGE_DAY_AFTER.md` |
| Pass 2 | assess `TaskState` and choose safe seams | identified readout extraction as the safest production slice | `GARBAGE_DAY_PASS_2_TASKSTATE_ASSESSMENT.md` |
| Pass 3 | extract operator readout cluster | split read-only readout logic into `taskstate_readout.go` | `GARBAGE_DAY_PASS_3_TASKSTATE_READOUT_BEFORE.md`, `GARBAGE_DAY_PASS_3_TASKSTATE_READOUT_AFTER.md` |
| Pass 4 | dedupe readout test helper | introduced shared malformed treasury helper | `GARBAGE_DAY_PASS_4_TASKSTATE_READOUT_TEST_HELPERS_BEFORE.md`, `GARBAGE_DAY_PASS_4_TASKSTATE_READOUT_TEST_HELPERS_AFTER.md` |
| Pass 5 | consolidate capability proposal fixtures | removed `216` duplicated lines from `taskstate_test.go` into shared helper file | `GARBAGE_DAY_PASS_5_TASKSTATE_CAPABILITY_PROPOSAL_FIXTURES_BEFORE.md`, `GARBAGE_DAY_PASS_5_TASKSTATE_CAPABILITY_PROPOSAL_FIXTURES_AFTER.md` |
| Pass 6 | consolidate capability config fixtures | moved repeated workspace config setup into shared helper | `GARBAGE_DAY_PASS_6_TASKSTATE_CAPABILITY_CONFIG_FIXTURES_BEFORE.md`, `GARBAGE_DAY_PASS_6_TASKSTATE_CAPABILITY_CONFIG_FIXTURES_AFTER.md` |
| Pass 7 | consolidate shared-storage config fixture | removed the last inline shared-storage config block | `GARBAGE_DAY_PASS_7_TASKSTATE_SHARED_STORAGE_CONFIG_FIXTURES_BEFORE.md`, `GARBAGE_DAY_PASS_7_TASKSTATE_SHARED_STORAGE_CONFIG_FIXTURES_AFTER.md` |
| Pass 8 | consolidate shared-storage exposure fixture | replaced `21` repeated exposure-store setup blocks with one helper | `GARBAGE_DAY_PASS_8_TASKSTATE_SHARED_STORAGE_EXPOSURE_FIXTURES_BEFORE.md`, `GARBAGE_DAY_PASS_8_TASKSTATE_SHARED_STORAGE_EXPOSURE_FIXTURES_AFTER.md` |

## Validation Summary

- Original cleanup pass validation: `git diff --check`, `go test -count=1 ./...`, plus targeted package tests all passed.
- Pass 3 validation: `go test -count=1 ./internal/agent/tools -run 'TestTaskStateOperator(Status|Inspect)'`, `go test -count=1 ./internal/agent/tools`, `go test -count=1 ./...` all passed.
- Passes 4 through 8 validation: each pass ran `gofmt`, `git diff --check`, targeted `internal/agent/tools` runs, package tests, and `go test -count=1 ./...`; all passed.
- Live reconciliation validation on `2026-04-19`: `go test -count=1 ./...` passed before this documentation cleanup.

Citations: `GARBAGE_DAY_AFTER.md`, `GARBAGE_DAY_PASS_3_TASKSTATE_READOUT_AFTER.md`, `GARBAGE_DAY_PASS_4_TASKSTATE_READOUT_TEST_HELPERS_AFTER.md`, `GARBAGE_DAY_PASS_5_TASKSTATE_CAPABILITY_PROPOSAL_FIXTURES_AFTER.md`, `GARBAGE_DAY_PASS_6_TASKSTATE_CAPABILITY_CONFIG_FIXTURES_AFTER.md`, `GARBAGE_DAY_PASS_7_TASKSTATE_SHARED_STORAGE_CONFIG_FIXTURES_AFTER.md`, `GARBAGE_DAY_PASS_8_TASKSTATE_SHARED_STORAGE_EXPOSURE_FIXTURES_AFTER.md`, live `go test -count=1 ./...`.

## Remaining Garbage After Phase 1

- `cmd/picobot/main_test.go` remains `10959` lines.
- `internal/agent/tools/taskstate_test.go` remains `7346` lines.
- `internal/agent/tools/taskstate.go` remains `3343` lines even after the readout extraction.
- `cmd/picobot/main.go` remains `3182` lines.
- `internal/missioncontrol/treasury_registry_test.go` remains `3454` lines.
- Repo health improved at the `TaskState` readout and fixture seam, but the repo still carries large protected V3 surfaces, broad `map[string]interface{}` usage, and a fragmented maintenance report set that needed this documentation pass.

Citations: `GARBAGE_DAY_SUMMARY.md`, live `wc -l`, live repo scans in Round 2.

## What Improved

- Duplicate atomic-write helpers were removed from production code.
- The safest read-only `TaskState` seam was split out into its own production file.
- Repeated test helper and fixture blocks were consolidated without changing scenario names or assertions.
- The raw Phase 1 reports now have an explicit index and a durable summary surface in `docs/maintenance/garbage-day/`.

## What Did Not Improve

- Package count did not change.
- The largest production and test files are still large enough to be review hazards.
- Protected V3 surfaces such as treasury, Zoho, approval, and runtime mutation paths were intentionally not decomposed here.
- Tests passing after Phase 1 did not mean overall repo health was clean.

## What Should Not Be Inferred

- Phase 1 did not authorize broad code cleanup.
- Phase 1 did not authorize Frank V4 implementation.
- Phase 1 did not prove that protected V3 surfaces are now safe to refactor casually.
- Phase 1 did not mean repo diagnosis was complete; Round 2 exists for that purpose.

## Notes On Source Quality

- All required Phase 1 source artifacts were present during this reconciliation.
- `docs/maintenance/GARBAGE_DAY_SUMMARY.md` exists in the live worktree but is currently untracked; facts taken from it should be read as report evidence, not as tracked git history.
- No values in this document were invented from missing artifacts, so no reconstructed values were needed.
