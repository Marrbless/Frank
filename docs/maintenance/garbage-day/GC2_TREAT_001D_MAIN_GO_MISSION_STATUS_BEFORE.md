# GC2-TREAT-001D Main.go Mission Status Before

- Branch: `frank-v3-foundation`
- HEAD: `cb127e9782e9a3b9b4456c5a51cdd3a3b2d6ed51`
- Tags at HEAD: `frank-garbage-campaign-001c-maingo-clean`
- Ahead/behind `upstream/main`: `380 ahead / 0 behind`
- `git status --short --branch`:
  - `## frank-v3-foundation`
- Baseline `go test -count=1 ./...` result: passed

## Exact functions/regions selected for extraction

- mission status/assertion support types:
  - `missionStatusSnapshot`
  - `missionStatusFrankZohoSendProofLocator`
  - `missionStatusFrankZohoSendProofVerification`
  - `missionStatusFrankZohoSendProofVerifier`
  - `missionStatusFrankZohoSendProofVerifierFunc`
  - `missionStatusAssertionExpectation`
- mission status/assertion helper functions:
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
- closely adjacent injected vars needed by that family:
  - `loadGatewayStatusObservation`
  - `loadGatewayStatusObservationFile`
  - `loadMissionStatusObservation`
  - `writeMissionStatusSnapshotAtomic`
  - `newFrankZohoSendProofVerifier`

Selected source regions:

- `cmd/picobot/main.go:1688-1761`
- `cmd/picobot/main.go:1764-2394`

## Exact non-goals

- Do not change `mission status`, `mission assert`, or `mission assert-step` command shapes, flags, help text, output text, assertion behavior, or error messages.
- Do not change gateway boot behavior.
- Do not change mission bootstrap/runtime hooks.
- Do not change mission step control watcher/control behavior.
- Do not change scheduled-trigger governance helpers.
- Do not change channels login, memory command, or mission inspect behavior except import/wiring adjacency if strictly needed.
- Do not widen the slice into broader mission runtime cleanup.

## Expected destination file

- `cmd/picobot/main_mission_status.go`
