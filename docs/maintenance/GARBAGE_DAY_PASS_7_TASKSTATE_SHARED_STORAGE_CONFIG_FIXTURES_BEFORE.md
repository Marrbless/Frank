# Garbage Day Pass 7 Before

## repo root
- `/mnt/d/pbot/picobot`

## branch
- `frank-v3-foundation`

## HEAD
- `b66b57eb35a4b51ced40f6c18bedea768bdb094d`

## git status --short --branch
```text
## frank-v3-foundation
```

## Pass 6 status
- Pass 6 is committed or otherwise no longer present as an uncommitted diff.

## line counts
- `internal/agent/tools/taskstate_test.go`: `7399`
- `internal/agent/tools/taskstate_capability_test_helpers_test.go`: `250`

## exact inline shared-storage config setup blocks found
- One remaining inline shared-storage workspace config setup block in `internal/agent/tools/taskstate_test.go`:
  - `TestTaskStateActivateStepSharedStorageCapabilityPathInvokesRealMutation`
- Exact preserved setup steps in that block:
  - `home := t.TempDir()`
  - `configDir := filepath.Join(home, ".picobot")`
  - `os.MkdirAll(configDir, 0o755)`
  - `workspace := filepath.Join(home, "workspace-root")`
  - `configPath := filepath.Join(configDir, "config.json")`
  - `configJSON := fmt.Sprintf(\`{"agents":{"defaults":{"workspace":%q}}}\`, workspace)`
  - `os.WriteFile(configPath, []byte(configJSON), 0o644)`
  - `t.Setenv("HOME", home)`
- Additional shared-storage-related repetition remains in the file around `missioncontrol.StoreWorkspaceSharedStorageCapabilityExposure(...)`, but that is an exposure-fixture seam, not a workspace config setup seam, and is out of scope for this pass.

## exact helpers planned for extraction
- Reuse the existing shared capability helper file and route the inline shared-storage workspace config setup through:
  - `writeTaskStateWorkspaceCapabilityConfigFixture`
- No new production code helpers are planned.
- No scenario names or assertions are planned to change.

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
ok  	github.com/local/picobot/internal/agent/tools	0.012s
```

## baseline validation
- Command:
  - `go test -count=1 ./internal/agent/tools -run 'Test.*(SharedStorage|Capability|Onboarding|Exposure|Proposal|Config)'`
- Result:
  - `ok  	github.com/local/picobot/internal/agent/tools	0.728s`
