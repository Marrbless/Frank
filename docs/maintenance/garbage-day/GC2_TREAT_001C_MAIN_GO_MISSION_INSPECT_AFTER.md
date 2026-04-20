# GC2-TREAT-001C Main.go Mission Inspect After

## Git diff summaries

### `git diff --stat`

```text
 cmd/picobot/main.go | 327 ----------------------------------------------------
 1 file changed, 327 deletions(-)
```

### `git diff --numstat`

```text
0	327	cmd/picobot/main.go
```

`git diff` does not include newly added untracked files. The extracted helper file and treatment notes are listed below under files changed.

## Files changed

- `cmd/picobot/main.go`
- `cmd/picobot/main_mission_inspect.go`
- `docs/maintenance/garbage-day/GC2_TREAT_001C_MAIN_GO_MISSION_INSPECT_BEFORE.md`
- `docs/maintenance/garbage-day/GC2_TREAT_001C_MAIN_GO_MISSION_INSPECT_AFTER.md`

## Exact functions moved

Moved from `cmd/picobot/main.go` into `cmd/picobot/main_mission_inspect.go`:

- inspect read-model types:
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
- inspect read-model helpers:
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

## Exact functions intentionally left in `main.go` and why

- the `missionInspectCmd` command block
  - Left in place because this slice was limited to the read-model helper family, not command wiring or command-surface redesign.
- mission bootstrap/runtime hooks
  - Left in place because they are protected runtime-truth zones and out of scope.
- mission step control activation/watch helpers
  - Left in place because they are watcher/runtime surfaces and explicitly out of scope.
- scheduled-trigger governance helpers
  - Left in place because they are a separate structural seam.
- mission status/assertion helpers
  - Left in place because they are adjacent but more semantically sensitive operator-truth surfaces.

## Runtime behavior

Runtime behavior was not changed.

- `mission inspect` command names, flags, help text, output text, read-model behavior, and error messages are unchanged.
- No gateway, mission bootstrap, mission runtime hook, watcher, scheduled-trigger, provider, or MCP behavior was touched.

## Validation commands and results

- `gofmt -w cmd/picobot/main.go cmd/picobot/main_mission_inspect.go`
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
    - `?? cmd/picobot/main_mission_inspect.go`
    - `?? docs/maintenance/garbage-day/GC2_TREAT_001C_MAIN_GO_MISSION_INSPECT_BEFORE.md`
    - `?? docs/maintenance/garbage-day/GC2_TREAT_001C_MAIN_GO_MISSION_INSPECT_AFTER.md`

## Deferred next candidates from the main.go assessment

- Scheduled-trigger governance helper extraction
- Mission status/assertion helper extraction
- Mission runtime/bootstrap hook extraction
