GC2-TREAT-002B main_test mission inspect split after

- branch: `frank-v3-foundation`
- head: `65789d6a32a8a92437d60670d6e082c477a39100`
- production behavior changed: no

Git diff --stat

```text
 cmd/picobot/main_test.go | 2363 +++-------------------------------------------
 1 file changed, 154 insertions(+), 2209 deletions(-)
```

Untracked moved test file diff stat

```text
 .../picobot/main_mission_inspect_test.go           | 2070 ++++++++++++++++++++
 1 file changed, 2070 insertions(+)
```

Git diff --numstat

```text
154	2209	cmd/picobot/main_test.go
```

Untracked moved test file diff numstat

```text
2070	0	/dev/null => cmd/picobot/main_mission_inspect_test.go
```

Files changed

- `cmd/picobot/main_test.go`
- `cmd/picobot/main_mission_inspect_test.go`
- `docs/maintenance/garbage-day/GC2_TREAT_002B_MAIN_TEST_MISSION_INSPECT_BEFORE.md`
- `docs/maintenance/garbage-day/GC2_TREAT_002B_MAIN_TEST_MISSION_INSPECT_AFTER.md`

Exact tests moved

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

Exact helpers moved

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

Exact tests intentionally left in `main_test.go` and why

- Prompt/channel tests stayed because they belong to operator-input/onboarding behavior, not mission inspect read-model coverage.
- Agent CLI tests stayed because they exercise a separate CLI surface and share no inspect fixtures.
- Mission status/assertion and mission set-step tests stayed because they share runtime-status helpers and assertion semantics.
- Mission bootstrap, runtime persistence, watcher, and operator-control tests stayed because they cover heavier protected runtime-truth surfaces and share overlapping mission-state fixtures.
- Package, prune, and gateway mission store logging tests stayed because they are not part of the mission inspect read-model family.
- Shared generic helpers such as `assertMainJSONObjectKeys`, `testMissionBootstrapJob`, and `writeMissionBootstrapJobFile` stayed because they are used across multiple non-inspect families.

Validation commands and results

- `gofmt -w cmd/picobot/main_test.go cmd/picobot/main_mission_inspect_test.go` -> passed
- `git diff --check` -> passed
- `go test -count=1 ./cmd/picobot` -> passed
- `go test -count=1 ./...` -> passed

Deferred next candidates from the `main_test` split assessment

- Mission status/assertion test family split
- Mission bootstrap/runtime/watcher/operator-control family split
