# Garbage Day Pass 8 After

## git diff --stat
```text
 .../taskstate_capability_test_helpers_test.go      |  8 +++
 internal/agent/tools/taskstate_test.go             | 84 ++++++----------------
 2 files changed, 29 insertions(+), 63 deletions(-)
```

## git diff --numstat
```text
8	0	internal/agent/tools/taskstate_capability_test_helpers_test.go
21	63	internal/agent/tools/taskstate_test.go
```

## files changed
- modified: `internal/agent/tools/taskstate_capability_test_helpers_test.go`
- modified: `internal/agent/tools/taskstate_test.go`
- added (untracked): `docs/maintenance/GARBAGE_DAY_PASS_8_TASKSTATE_SHARED_STORAGE_EXPOSURE_FIXTURES_BEFORE.md`
- added (untracked): `docs/maintenance/GARBAGE_DAY_PASS_8_TASKSTATE_SHARED_STORAGE_EXPOSURE_FIXTURES_AFTER.md`

## before/after line counts
- `internal/agent/tools/taskstate_test.go`: `7388 -> 7346`
- `internal/agent/tools/taskstate_capability_test_helpers_test.go`: `256 -> 264`

## exact helpers moved or introduced
- Introduced shared exposure helper:
  - `storeTaskStateSharedStorageCapabilityExposure`
- Reused existing per-capability config helpers unchanged.
- Replaced the repeated direct setup blocks in `internal/agent/tools/taskstate_test.go` with helper calls in these tests:
  - `TestTaskStateActivateStepContactsCapabilityPathCallsHookOnce`
  - `TestTaskStateActivateStepContactsCapabilityPathInvokesRealMutation`
  - `TestTaskStateActivateStepContactsCapabilityFailsClosedWithoutExposedRecord`
  - `TestTaskStateActivateStepLocationCapabilityPathCallsHookOnce`
  - `TestTaskStateActivateStepLocationCapabilityPathInvokesRealMutation`
  - `TestTaskStateActivateStepLocationCapabilityFailsClosedWithoutExposedRecord`
  - `TestTaskStateActivateStepCameraCapabilityPathCallsHookOnce`
  - `TestTaskStateActivateStepCameraCapabilityPathInvokesRealMutation`
  - `TestTaskStateActivateStepCameraCapabilityFailsClosedWithoutExposedRecord`
  - `TestTaskStateActivateStepMicrophoneCapabilityPathCallsHookOnce`
  - `TestTaskStateActivateStepMicrophoneCapabilityPathInvokesRealMutation`
  - `TestTaskStateActivateStepMicrophoneCapabilityFailsClosedWithoutExposedRecord`
  - `TestTaskStateActivateStepSMSPhoneCapabilityPathCallsHookOnce`
  - `TestTaskStateActivateStepSMSPhoneCapabilityPathInvokesRealMutation`
  - `TestTaskStateActivateStepSMSPhoneCapabilityFailsClosedWithoutExposedRecord`
  - `TestTaskStateActivateStepBluetoothNFCCapabilityPathCallsHookOnce`
  - `TestTaskStateActivateStepBluetoothNFCCapabilityPathInvokesRealMutation`
  - `TestTaskStateActivateStepBluetoothNFCCapabilityFailsClosedWithoutExposedRecord`
  - `TestTaskStateActivateStepBroadAppControlCapabilityPathCallsHookOnce`
  - `TestTaskStateActivateStepBroadAppControlCapabilityPathInvokesRealMutation`
  - `TestTaskStateActivateStepBroadAppControlCapabilityFailsClosedWithoutExposedRecord`

## exact exposure fixture records preserved
- The new helper preserves the exact prior setup behavior:
  - call `missioncontrol.StoreWorkspaceSharedStorageCapabilityExposure(root, workspace)`
  - use the same `root` argument value already built by each test
  - use the same `workspace` value already returned by each config fixture helper
  - preserve the exact fatal-on-error behavior:
    - `t.Fatalf("StoreWorkspaceSharedStorageCapabilityExposure() error = %v", err)`
- Because the helper is a direct wrapper around the existing missioncontrol call, the committed shared-storage exposure record shape remains unchanged.

## exact assertions preserved
- No test scenario names changed.
- No assertion text changed.
- No acceptance/rejection expectations changed.
- No shared-storage capability semantics changed.
- No capability onboarding or capability exposure assertions changed.
- All existing assertions that depend on shared-storage exposure being present still execute from the same call sites, with the same `root` and `workspace` values.

## any repeated shared-storage setup intentionally left alone and why
- Left the hook-local `missioncontrol.StoreWorkspaceSharedStorageCapabilityExposure(root, filepath.Join(t.TempDir(), "workspace"))` call in `TestTaskStateActivateStepSharedStorageCapabilityPathCallsHookOnce`.
  - Reason: it is not the repeated `root, workspace` fixture shape used across the larger cluster; it intentionally builds a fresh temp path inline inside the hook.
- Left non-shared-storage capability exposure setup helpers and stores alone.
  - Reason: those are separate exposure seams, not this shared-storage precondition seam.

## risks / deferred cleanup
- The hook-local shared-storage exposure store remains a plausible future cleanup candidate if you want a helper that accepts ad hoc workspace paths.
- Additional capability-specific exposure-store setup may still be reducible, but that would be a broader fixture consolidation pass and was not touched here.
- This pass depends on the helper remaining a thin wrapper; any future behavior added inside it would affect all 21 tests together.

## validation commands and results
- `gofmt -w internal/agent/tools/taskstate_test.go internal/agent/tools/taskstate_capability_test_helpers_test.go`
  - result: passed
- `git diff --check`
  - result: passed
- `go test -count=1 ./internal/agent/tools -run 'Test.*(SharedStorage|Capability|Onboarding|Exposure|Proposal|Config)'`
  - result: `ok  	github.com/local/picobot/internal/agent/tools	0.740s`
- `go test -count=1 ./internal/agent/tools`
  - result: `ok  	github.com/local/picobot/internal/agent/tools	8.719s`
- `go test -count=1 ./...`
  - result: passed
  - representative tail:
    - `ok  	github.com/local/picobot/cmd/picobot	14.275s`
    - `ok  	github.com/local/picobot/internal/agent	0.283s`
    - `ok  	github.com/local/picobot/internal/agent/tools	13.810s`
    - `ok  	github.com/local/picobot/internal/missioncontrol	9.735s`
