## GC3-TREAT-001C Before

- Branch: `frank-v3-foundation`
- HEAD: `55779753bb8ac5d5c06cc7522723e87cdcb683dd`
- Tags at HEAD:
  - `frank-garbage-campaign-gc3-fix-001-zoho-timestamps-clean`
- Ahead/behind `upstream/main`: `395 0`
- `git status --short --branch`:
  - `## frank-v3-foundation`
- Baseline `go test -count=1 ./...` result:
  - passed

### Selected Extraction Region

Move the capability exposure applier family out of `internal/agent/tools/taskstate.go` into `internal/agent/tools/taskstate_capability_exposure.go`:

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

### Exact Non-Goals

- No runtime behavior changes.
- No changes to persistence or hydration core.
- No changes to approval, waiting-user, or runtime-control parity paths.
- No changes to treasury, campaign readiness, onboarding activation, or Zoho lifecycle code.
- No V4 work.
- No broad cleanup or dependency changes.
- No test weakening.

### Expected Destination File

- `internal/agent/tools/taskstate_capability_exposure.go`
