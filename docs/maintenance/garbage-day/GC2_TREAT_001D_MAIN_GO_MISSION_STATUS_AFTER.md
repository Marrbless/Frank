# GC2-TREAT-001D Main.go Mission Status After

## Git diff summaries

### `git diff --stat`

```text
 cmd/picobot/main.go | 471 ----------------------------------------------------
 1 file changed, 471 deletions(-)
```

### `git diff --numstat`

```text
0	471	cmd/picobot/main.go
```

`git diff` does not include newly added untracked files. The extracted helper file and treatment notes are listed below under files changed.

## Files changed

- `cmd/picobot/main.go`
- `cmd/picobot/main_mission_status.go`
- `docs/maintenance/garbage-day/GC2_TREAT_001D_MAIN_GO_MISSION_STATUS_BEFORE.md`
- `docs/maintenance/garbage-day/GC2_TREAT_001D_MAIN_GO_MISSION_STATUS_AFTER.md`

## Exact functions moved

Moved from `cmd/picobot/main.go` into `cmd/picobot/main_mission_status.go`:

- mission status/assertion support types:
  - `missionStatusSnapshot`
  - `missionStatusFrankZohoSendProofLocator`
  - `missionStatusFrankZohoSendProofVerification`
  - `missionStatusFrankZohoSendProofVerifier`
  - `missionStatusFrankZohoSendProofVerifierFunc`
  - `missionStatusAssertionExpectation`
- mission status/assertion injected vars:
  - `loadGatewayStatusObservation`
  - `loadGatewayStatusObservationFile`
  - `loadMissionStatusObservation`
  - `writeMissionStatusSnapshotAtomic`
  - `newFrankZohoSendProofVerifier`
- mission status/assertion helpers:
  - `loadMissionStatusFrankZohoSendProofFile`
  - `loadMissionStatusFrankZohoVerifiedSendProofFile`
  - `writeMissionStatusSnapshotFromCommand`
  - `missionStatusSnapshotMissionFile`
  - `waitForMissionStatusStepConfirmation`
  - `assertMissionStatusSnapshot`
  - `assertMissionGatewayStatusSnapshot`
  - `waitForMissionStatusAssertion`
  - `waitForMissionGatewayStatusAssertion`
  - `projectGatewayStatusAssertionSnapshot`
  - `newMissionStatusAssertionForStep`
  - `checkMissionStatusAssertion`
  - `equalAllowedToolsExact`
  - `loadMissionStatusSnapshot`
  - `valueOrNilString`
  - `containsString`
  - `writeMissionStatusSnapshot`
  - `writeProjectedMissionStatusSnapshot`
  - `intersectAllowedTools`
  - `removeMissionStatusSnapshot`

## Exact functions intentionally left in `main.go` and why

- `missionStatusCmd`, `missionAssertCmd`, and `missionAssertStepCmd`
  - Left in place because this slice only extracted the helper family, not command wiring or command-surface layout.
- mission bootstrap/runtime hydration helpers such as:
  - `configureMissionBootstrapJob`
  - `loadPersistedMissionRuntime`
  - `loadCommittedMissionRuntime`
  - `loadPersistedMissionRuntimeSnapshot`
  - Left in place because they are runtime/bootstrap surfaces, not the selected status/assertion seam.
- mission step control activation/watch helpers
  - Left in place because they are watcher/control surfaces and explicitly out of scope.
- scheduled-trigger governance helpers
  - Left in place because they are a separate structural seam.
- channels login, memory command, and mission inspect helpers
  - Left untouched because those are separate completed seams.

## Runtime behavior

Runtime behavior was not changed.

- `mission status`, `mission assert`, and `mission assert-step` command names, flags, help text, output text, assertion behavior, and error messages are unchanged.
- No gateway, mission bootstrap, runtime hook, watcher, scheduled-trigger, provider, or MCP behavior was touched.

## Validation commands and results

- `gofmt -w cmd/picobot/main.go cmd/picobot/main_mission_status.go`
  - passed
- `git diff --check`
  - passed
- `go test -count=1 ./cmd/picobot`
  - passed
- `go test -count=1 ./...`
  - passed
- `git status --short --branch`
  - result after slice:
    - `## frank-v3-foundation`
    - ` M cmd/picobot/main.go`
    - `?? cmd/picobot/main_mission_status.go`
    - `?? docs/maintenance/garbage-day/GC2_TREAT_001D_MAIN_GO_MISSION_STATUS_BEFORE.md`
    - `?? docs/maintenance/garbage-day/GC2_TREAT_001D_MAIN_GO_MISSION_STATUS_AFTER.md`

## Deferred next candidates from the main.go assessment

- Scheduled-trigger governance helper extraction
- Mission runtime/bootstrap hook extraction
