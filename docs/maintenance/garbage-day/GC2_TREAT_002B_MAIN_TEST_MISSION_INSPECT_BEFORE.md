GC2-TREAT-002B main_test mission inspect split before

- branch: `frank-v3-foundation`
- head: `65789d6a32a8a92437d60670d6e082c477a39100`
- tags at head: `frank-garbage-campaign-002a-main-test-scheduled-trigger-clean`
- ahead/behind upstream: `384 ahead / 0 behind`
- git status --short --branch:

```text
## frank-v3-foundation
```

- baseline `go test -count=1 ./...` result: failed before edits
  - unchanged pre-existing failure surface: `github.com/local/picobot/internal/agent`
  - observed failing test at baseline: `TestAgentRememberFailsClosedWhenAppendTodayFails`

Exact tests selected for movement

- `TestMissionInspectCommandWithValidFilePrintsExpectedSummary`
- `TestMissionInspectCommandWithStepIDReturnsExactlyOneResolvedStep`
- `TestMissionInspectCommandWithStepIDIncludesResolvedEffectiveAllowedTools`
- `TestMissionInspectCommandTreasuryPreflightZeroRefPathUnchanged`
- `TestMissionInspectCommandTreasuryStepSurfacesResolvedTreasuryPreflight`
- `TestMissionInspectCommandCampaignStepSurfacesResolvedCampaignPreflight`
- `TestMissionInspectCommandZohoMailboxBootstrapStepSurfacesResolvedPreflight`
- `TestMissionInspectCommandTelegramOwnerControlStepSurfacesResolvedPreflight`
- `TestMissionInspectCommandTreasuryPreflightInvalidContainerStateFailsClosed`
- `TestMissionInspectCommandMissingCampaignFailsClosed`
- `TestMissionInspectCommandWithUnknownStepReturnsClearError`
- `TestMissionInspectCommandWithoutStepIDPreservesExistingBehavior`
- `TestMissionInspectCommandNotificationsCapabilityReturnsCommittedRecord`
- `TestMissionInspectCommandNotificationsCapabilityRequiresStoreRoot`
- `TestMissionInspectCommandNotificationsCapabilityRejectsStepWithoutRequirement`
- `TestMissionInspectCommandSharedStorageCapabilityReturnsCommittedRecord`
- `TestMissionInspectCommandSharedStorageCapabilityRequiresStoreRoot`
- `TestMissionInspectCommandSharedStorageCapabilityRejectsStepWithoutRequirement`
- `TestMissionInspectCommandContactsCapabilityReturnsCommittedRecordAndSource`
- `TestMissionInspectCommandContactsCapabilityRequiresStoreRoot`
- `TestMissionInspectCommandContactsCapabilityRejectsStepWithoutRequirement`
- `TestMissionInspectCommandLocationCapabilityReturnsCommittedRecordAndSource`
- `TestMissionInspectCommandLocationCapabilityRequiresStoreRoot`
- `TestMissionInspectCommandLocationCapabilityRejectsStepWithoutRequirement`
- `TestMissionInspectCommandCameraCapabilityReturnsCommittedRecordAndSource`
- `TestMissionInspectCommandCameraCapabilityRequiresStoreRoot`
- `TestMissionInspectCommandCameraCapabilityRejectsStepWithoutRequirement`
- `TestMissionInspectCommandMicrophoneCapabilityReturnsCommittedRecordAndSource`
- `TestMissionInspectCommandMicrophoneCapabilityRequiresStoreRoot`
- `TestMissionInspectCommandMicrophoneCapabilityRejectsStepWithoutRequirement`
- `TestMissionInspectCommandSMSPhoneCapabilityReturnsCommittedRecordAndSource`
- `TestMissionInspectCommandSMSPhoneCapabilityRequiresStoreRoot`
- `TestMissionInspectCommandSMSPhoneCapabilityRejectsStepWithoutRequirement`
- `TestMissionInspectCommandBluetoothNFCCapabilityReturnsCommittedRecordAndSource`
- `TestMissionInspectCommandBluetoothNFCCapabilityRequiresStoreRoot`
- `TestMissionInspectCommandBluetoothNFCCapabilityRejectsStepWithoutRequirement`
- `TestMissionInspectCommandBroadAppControlCapabilityReturnsCommittedRecordAndSource`
- `TestMissionInspectCommandBroadAppControlCapabilityRequiresStoreRoot`
- `TestMissionInspectCommandBroadAppControlCapabilityRejectsStepWithoutRequirement`
- `TestMissionInspectCommandSuccessCriteriaZeroValuePreservesExistingBehavior`
- `TestMissionInspectCommandWithZeroToolStepPrintsEmptyEffectiveAllowedTools`
- `TestMissionInspectCommandWithMissingFileReturnsError`
- `TestMissionInspectCommandWithInvalidJSONReturnsError`
- `TestMissionInspectCommandWithInvalidMissionReturnsValidationError`

Exact helpers selected for movement

- `writeMalformedTreasuryRecordForMainTest`
- `writeMissionInspectNotificationsCapabilityFixtures`
- `writeMissionInspectSharedStorageCapabilityFixtures`
- `writeMissionInspectContactsCapabilityFixtures`
- `writeMissionInspectLocationCapabilityFixtures`
- `writeMissionInspectCameraCapabilityFixtures`
- `writeMissionInspectMicrophoneCapabilityFixtures`
- `writeMissionInspectSMSPhoneCapabilityFixtures`
- `writeMissionInspectBluetoothNFCCapabilityFixtures`
- `writeMissionInspectBroadAppControlCapabilityFixtures`
- `writeMissionInspectTreasuryFixtures`
- `writeMissionInspectZohoMailboxBootstrapFixtures`
- `writeMissionInspectTelegramOwnerControlFixtures`
- `mustStoreMissionInspectCampaignFixture`
- `writeMissionInspectEligibilityFixture`

Exact non-goals

- no production code changes
- no changes to prompt/channel, agent CLI, mission status/assertion, mission set-step, mission bootstrap/runtime, watcher/operator-control, package/prune, or other `main_test.go` families
- no shared helper extraction beyond mission-inspect-local helper movement
- no V4 work
- no dependency changes
- no test weakening or deletion

Expected destination file

- `cmd/picobot/main_mission_inspect_test.go`
