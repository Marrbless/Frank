# Garbage Day Pass 8 Before

## repo root
- `/mnt/d/pbot/picobot`

## branch
- `frank-v3-foundation`

## HEAD
- `6c4534f1dc5f27dd973b6d55b2c44080cac126a9`

## git status --short --branch
```text
## frank-v3-foundation
```

## Pass 7 status
- Pass 7 is committed or otherwise no longer present as an uncommitted diff.

## line counts
- `internal/agent/tools/taskstate_test.go`: `7388`
- `internal/agent/tools/taskstate_capability_test_helpers_test.go`: `256`

## exact repeated shared-storage exposure-store setup blocks found
- Repeated identical setup block in `internal/agent/tools/taskstate_test.go`:
  - `if _, err := missioncontrol.StoreWorkspaceSharedStorageCapabilityExposure(root, workspace); err != nil {`
  - `\tt.Fatalf("StoreWorkspaceSharedStorageCapabilityExposure() error = %v", err)`
  - `}`
- Exact repeated locations found for the `root, workspace` variant:
  - around lines `5440`, `5483`, `5545`, `5597`, `5640`, `5702`, `5754`, `5797`, `5859`, `5911`, `5954`, `6016`, `6068`, `6111`, `6173`, `6225`, `6268`, `6330`, `6382`, `6425`, `6487`
- Separate non-target variant also found:
  - hook-local `missioncontrol.StoreWorkspaceSharedStorageCapabilityExposure(root, filepath.Join(t.TempDir(), "workspace"))`
  - this is not part of the repeated `root, workspace` fixture seam and is planned to remain untouched in this pass

## exact helpers planned for extraction
- Add one shared test-only helper to `internal/agent/tools/taskstate_capability_test_helpers_test.go`:
  - `storeTaskStateSharedStorageCapabilityExposure`
- Planned helper behavior:
  - accept `t`, `root`, and `workspace`
  - call `missioncontrol.StoreWorkspaceSharedStorageCapabilityExposure(root, workspace)`
  - preserve the exact fatal-on-error message and behavior
- Replace only the repeated `root, workspace` setup blocks in `internal/agent/tools/taskstate_test.go`

## targeted test names
From:
`go test -list 'Test.*(SharedStorage|Capability|Onboarding|Exposure|Proposal|Config)' ./internal/agent/tools`

```text
TestTaskStateActivateStepTelegramOwnerControlOnboardingInvokesHookWithResolvedBundle
TestTaskStateActivateStepTelegramOwnerControlOnboardingFailsClosedWithoutAccount
TestTaskStateActivateStepNotificationsCapabilityPathCallsHookOnce
TestTaskStateActivateStepNotificationsCapabilityPathInvokesRealMutation
TestTaskStateActivateStepNotificationsCapabilityRequiresApprovedProposal
TestTaskStateActivateStepNotificationsCapabilityFailsClosedWithoutExposedRecord
TestTaskStateActivateStepSharedStorageCapabilityPathCallsHookOnce
TestTaskStateActivateStepSharedStorageCapabilityPathInvokesRealMutation
TestTaskStateActivateStepSharedStorageCapabilityRequiresApprovedProposal
TestTaskStateActivateStepSharedStorageCapabilityFailsClosedWithoutExposedRecord
TestTaskStateActivateStepContactsCapabilityPathCallsHookOnce
TestTaskStateActivateStepContactsCapabilityPathInvokesRealMutation
TestTaskStateActivateStepContactsCapabilityRequiresApprovedProposal
TestTaskStateActivateStepContactsCapabilityFailsClosedWithoutExposedRecord
TestTaskStateActivateStepContactsCapabilityFailsClosedWithoutSharedStorageExposure
TestTaskStateActivateStepLocationCapabilityPathCallsHookOnce
TestTaskStateActivateStepLocationCapabilityPathInvokesRealMutation
TestTaskStateActivateStepLocationCapabilityRequiresApprovedProposal
TestTaskStateActivateStepLocationCapabilityFailsClosedWithoutExposedRecord
TestTaskStateActivateStepLocationCapabilityFailsClosedWithoutSharedStorageExposure
TestTaskStateActivateStepCameraCapabilityPathCallsHookOnce
TestTaskStateActivateStepCameraCapabilityPathInvokesRealMutation
TestTaskStateActivateStepCameraCapabilityRequiresApprovedProposal
TestTaskStateActivateStepCameraCapabilityFailsClosedWithoutExposedRecord
TestTaskStateActivateStepCameraCapabilityFailsClosedWithoutSharedStorageExposure
TestTaskStateActivateStepMicrophoneCapabilityPathCallsHookOnce
TestTaskStateActivateStepMicrophoneCapabilityPathInvokesRealMutation
TestTaskStateActivateStepMicrophoneCapabilityRequiresApprovedProposal
TestTaskStateActivateStepMicrophoneCapabilityFailsClosedWithoutExposedRecord
TestTaskStateActivateStepMicrophoneCapabilityFailsClosedWithoutSharedStorageExposure
TestTaskStateActivateStepSMSPhoneCapabilityPathCallsHookOnce
TestTaskStateActivateStepSMSPhoneCapabilityPathInvokesRealMutation
TestTaskStateActivateStepSMSPhoneCapabilityRequiresApprovedProposal
TestTaskStateActivateStepSMSPhoneCapabilityFailsClosedWithoutExposedRecord
TestTaskStateActivateStepSMSPhoneCapabilityFailsClosedWithoutSharedStorageExposure
TestTaskStateActivateStepBluetoothNFCCapabilityPathCallsHookOnce
TestTaskStateActivateStepBluetoothNFCCapabilityPathInvokesRealMutation
TestTaskStateActivateStepBluetoothNFCCapabilityRequiresApprovedProposal
TestTaskStateActivateStepBluetoothNFCCapabilityFailsClosedWithoutExposedRecord
TestTaskStateActivateStepBluetoothNFCCapabilityFailsClosedWithoutSharedStorageExposure
TestTaskStateActivateStepBroadAppControlCapabilityPathCallsHookOnce
TestTaskStateActivateStepBroadAppControlCapabilityPathInvokesRealMutation
TestTaskStateActivateStepBroadAppControlCapabilityRequiresApprovedProposal
TestTaskStateActivateStepBroadAppControlCapabilityFailsClosedWithoutExposedRecord
TestTaskStateActivateStepBroadAppControlCapabilityFailsClosedWithoutSharedStorageExposure
ok  	github.com/local/picobot/internal/agent/tools	0.009s
```

## baseline validation
- Command:
  - `go test -count=1 ./internal/agent/tools -run 'Test.*(SharedStorage|Capability|Onboarding|Exposure|Proposal|Config)'`
- Result:
  - `ok  	github.com/local/picobot/internal/agent/tools	0.728s`
