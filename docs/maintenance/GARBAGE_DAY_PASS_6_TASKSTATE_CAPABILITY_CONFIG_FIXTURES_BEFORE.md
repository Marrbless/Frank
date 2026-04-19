# Garbage Day Pass 6 Before

## repo root
- `/mnt/d/pbot/picobot`

## branch
- `frank-v3-foundation`

## HEAD
- `5896c8ecfc6869abedf0f085b53334b1670ed485`

## git status --short --branch
```text
## frank-v3-foundation
```

## Pass 5 status
- Pass 5 is committed or otherwise no longer present as an uncommitted diff.

## line counts
- `internal/agent/tools/taskstate_test.go`: `7525`
- `internal/agent/tools/taskstate_capability_test_helpers_test.go`: `187`

## exact capability config fixture functions or repeated blocks found
- Exact duplicate config fixture wrappers in `internal/agent/tools/taskstate_test.go`:
  - `writeTaskStateContactsCapabilityConfigFixture`
  - `writeTaskStateLocationCapabilityConfigFixture`
  - `writeTaskStateCameraCapabilityConfigFixture`
  - `writeTaskStateMicrophoneCapabilityConfigFixture`
  - `writeTaskStateSMSPhoneCapabilityConfigFixture`
  - `writeTaskStateBluetoothNFCCapabilityConfigFixture`
  - `writeTaskStateBroadAppControlCapabilityConfigFixture`
- Each duplicate wrapper performs the same steps:
  - `home := t.TempDir()`
  - create `filepath.Join(home, ".picobot")`
  - set `workspace := filepath.Join(home, "workspace-root")`
  - write `config.json` with exact JSON `{"agents":{"defaults":{"workspace":%q}}}`
  - `t.Setenv("HOME", home)`
  - return `workspace`
- Similar-but-separate config setup blocks also exist elsewhere in `taskstate_test.go`:
  - inline shared-storage config setup in `TestTaskStateActivateStepSharedStorageCapabilityPathInvokesRealMutation`
  - inline notifications Telegram config setup in `TestTaskStateActivateStepNotificationsCapabilityPathInvokesRealMutation`
- Those blocks are not part of the initial smallest safe extraction unless the consolidation naturally reuses the identical workspace-config helper without widening behavior risk.

## exact helpers planned for extraction
- Add shared test-only helper in `internal/agent/tools/taskstate_capability_test_helpers_test.go`:
  - `writeTaskStateWorkspaceCapabilityConfigFixture`
- Preserve scenario-readable wrapper names by moving these wrappers into the shared helper file and having them delegate:
  - `writeTaskStateContactsCapabilityConfigFixture`
  - `writeTaskStateLocationCapabilityConfigFixture`
  - `writeTaskStateCameraCapabilityConfigFixture`
  - `writeTaskStateMicrophoneCapabilityConfigFixture`
  - `writeTaskStateSMSPhoneCapabilityConfigFixture`
  - `writeTaskStateBluetoothNFCCapabilityConfigFixture`
  - `writeTaskStateBroadAppControlCapabilityConfigFixture`

## targeted test names
From:
`go test -list 'Test.*(Capability|Onboarding|Exposure|Proposal|Config)' ./internal/agent/tools`

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
ok  	github.com/local/picobot/internal/agent/tools	0.010s
```

## baseline validation
- Command:
  - `go test -count=1 ./internal/agent/tools -run 'Test.*(Capability|Onboarding|Exposure|Proposal|Config)'`
- Result:
  - `ok  	github.com/local/picobot/internal/agent/tools	0.726s`
