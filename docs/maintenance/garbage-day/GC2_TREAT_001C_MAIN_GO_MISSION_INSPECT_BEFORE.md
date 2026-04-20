# GC2-TREAT-001C Main.go Mission Inspect Before

- Branch: `frank-v3-foundation`
- HEAD: `0e0de854d3b98ab883b18a79e8e146dab2993861`
- Tags at HEAD: `frank-garbage-campaign-001b-maingo-clean`
- Ahead/behind `upstream/main`: `379 ahead / 0 behind`
- `git status --short --branch`:
  - `## frank-v3-foundation`
- Baseline `go test -count=1 ./...` result: passed

## Exact functions/regions selected for extraction

- mission inspect read-model types:
  - `missionInspectSummary`
  - `missionInspectNotificationsCapability`
  - `missionInspectSharedStorageCapability`
  - `missionInspectContactsCapability`
  - `missionInspectLocationCapability`
  - `missionInspectCameraCapability`
  - `missionInspectMicrophoneCapability`
  - `missionInspectSMSPhoneCapability`
  - `missionInspectBluetoothNFCCapability`
  - `missionInspectBroadAppControlCapability`
- mission inspect read-model helpers:
  - `newMissionInspectSummary`
  - `newMissionInspectNotificationsCapability`
  - `newMissionInspectSharedStorageCapability`
  - `newMissionInspectContactsCapability`
  - `newMissionInspectLocationCapability`
  - `newMissionInspectCameraCapability`
  - `newMissionInspectMicrophoneCapability`
  - `newMissionInspectSMSPhoneCapability`
  - `newMissionInspectBluetoothNFCCapability`
  - `newMissionInspectBroadAppControlCapability`

Selected source region:

- `cmd/picobot/main.go:1754-2235`

## Exact non-goals

- Do not change the `mission inspect` command shape, flags, help text, output text, or error messages.
- Do not change gateway boot behavior.
- Do not change mission bootstrap/runtime hooks.
- Do not change mission step control watcher behavior.
- Do not change scheduled-trigger governance helpers.
- Do not change mission status/assertion helpers other than import/wiring adjacency if strictly required.
- Do not change channels login or memory command behavior.
- Do not widen the slice into broader mission-control cleanup.

## Expected destination file

- `cmd/picobot/main_mission_inspect.go`
