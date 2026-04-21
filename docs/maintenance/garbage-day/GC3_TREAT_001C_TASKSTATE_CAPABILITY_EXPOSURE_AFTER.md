## GC3-TREAT-001C After

### git diff --stat

```text
 internal/agent/tools/taskstate.go | 649 --------------------------------------
 1 file changed, 649 deletions(-)
```

Note: `git diff --stat` does not include untracked files. The extracted production file and treatment docs are listed below under files changed.

### git diff --numstat

```text
0	649	internal/agent/tools/taskstate.go
```

### Files Changed

- `internal/agent/tools/taskstate.go`
- `internal/agent/tools/taskstate_capability_exposure.go`
- `docs/maintenance/garbage-day/GC3_TREAT_001C_TASKSTATE_CAPABILITY_EXPOSURE_BEFORE.md`
- `docs/maintenance/garbage-day/GC3_TREAT_001C_TASKSTATE_CAPABILITY_EXPOSURE_AFTER.md`

### Exact Functions Moved

- `(*TaskState).applyNotificationsCapabilityForStep`
- `defaultNotificationsCapabilityExposureHook`
- `(*TaskState).applySharedStorageCapabilityForStep`
- `defaultSharedStorageCapabilityExposureHook`
- `(*TaskState).applyContactsCapabilityForStep`
- `defaultContactsCapabilityExposureHook`
- `(*TaskState).applyLocationCapabilityForStep`
- `defaultLocationCapabilityExposureHook`
- `(*TaskState).applyCameraCapabilityForStep`
- `defaultCameraCapabilityExposureHook`
- `(*TaskState).applyMicrophoneCapabilityForStep`
- `defaultMicrophoneCapabilityExposureHook`
- `(*TaskState).applySMSPhoneCapabilityForStep`
- `defaultSMSPhoneCapabilityExposureHook`
- `(*TaskState).applyBluetoothNFCCapabilityForStep`
- `defaultBluetoothNFCCapabilityExposureHook`
- `(*TaskState).applyBroadAppControlCapabilityForStep`
- `defaultBroadAppControlCapabilityExposureHook`

### Functions Intentionally Left In `taskstate.go`

- `(*TaskState).ActivateStep`
  - Left in place as the central activation/orchestration front door that wires capability exposure alongside campaign readiness, onboarding, treasury, runtime activation, and persistence.
- `NewTaskState`
  - Left in place because hook field wiring belongs with `TaskState` construction.
- `TaskState` hook fields for capability exposure
  - Left in place because they are part of the central state object shape, not the extracted applier implementation.
- `(*TaskState).applyCampaignReadinessGuardForStep`
- `(*TaskState).applyZohoMailboxBootstrapForStep`
- `(*TaskState).applyTelegramOwnerControlOnboardingForStep`
- `(*TaskState).applyTreasuryExecutionForStep`
  - Left in place because they are explicitly protected mutation zones and were out of scope for this treatment.

### Tests Added or Changed

- None.
- Existing capability exposure tests in `internal/agent/tools/taskstate_test.go` were preserved unchanged because same-package extraction did not require mechanical test updates.

### Runtime Behavior

- Runtime behavior was not changed.
- This treatment only moved the capability exposure applier family into a dedicated same-package production file.

### Validators Changed

- No validators or missioncontrol persistence logic were changed.

### Validation Commands And Results

- `gofmt -w internal/agent/tools/taskstate.go internal/agent/tools/taskstate_capability_exposure.go`
  - passed
- `git diff --check`
  - passed
- `go test -count=1 ./internal/agent/tools`
  - passed
- `go test -count=1 ./...`
  - passed
- `git status --short --branch`
  - final:
    - `## frank-v3-foundation`
    - ` M internal/agent/tools/taskstate.go`
    - `?? docs/maintenance/garbage-day/GC3_TREAT_001C_TASKSTATE_CAPABILITY_EXPOSURE_AFTER.md`
    - `?? docs/maintenance/garbage-day/GC3_TREAT_001C_TASKSTATE_CAPABILITY_EXPOSURE_BEFORE.md`
    - `?? internal/agent/tools/taskstate_capability_exposure.go`

### Deferred Next Candidates From The TaskState Assessment

- `GC3-TREAT-001D` runtime persistence-core extraction
- `GC3-TREAT-001E` approval / reboot-safe control cleanup
