GC2-TREAT-002C main_test mission status family split after

- branch: `frank-v3-foundation`
- head: `feb79a0c487bdc70c55a82364aa583ccee977de6`
- production behavior changed: no

Git diff --stat

```text
 cmd/picobot/main_test.go | 1429 ----------------------------------------------
 1 file changed, 1429 deletions(-)
```

Untracked moved test file diff stat

```text
 .../picobot/main_mission_status_test.go            | 1443 ++++++++++++++++++++
 1 file changed, 1443 insertions(+)
```

Git diff --numstat

```text
0	1429	cmd/picobot/main_test.go
```

Untracked moved test file diff numstat

```text
1443	0	/dev/null => cmd/picobot/main_mission_status_test.go
```

Files changed

- `cmd/picobot/main_test.go`
- `cmd/picobot/main_mission_status_test.go`
- `docs/maintenance/garbage-day/GC2_TREAT_002C_MAIN_TEST_MISSION_STATUS_BEFORE.md`
- `docs/maintenance/garbage-day/GC2_TREAT_002C_MAIN_TEST_MISSION_STATUS_AFTER.md`

Exact tests moved

- `TestMissionStatusCommandWithValidFilePrintsExpectedJSON`
- `TestMissionStatusCommandWithActiveStepFieldsPrintsExpectedJSON`
- `TestMissionStatusCommandUsesSharedObservationReader`
- `TestMissionStatusCommandPrintsCanonicalGatewayStatusJSON`
- `TestMissionStatusCommandReturnsFrankZohoSendProofLocatorsFromRuntimeSummary`
- `TestMissionStatusCommandVerifiesFrankZohoSendProofFromRuntimeSummary`
- `TestMissionStatusCommandWithMissingFileReturnsError`
- `TestMissionStatusCommandWithInvalidFileReturnsError`
- `TestMissionAssertCommandWithValidStatusFileAndNoConditionsSucceeds`
- `TestMissionAssertCommandOneShotJobIDMatchSucceeds`
- `TestMissionAssertCommandOneShotStepIDMismatchFailsClearly`
- `TestMissionAssertCommandOneShotActiveMismatchFailsClearly`
- `TestMissionAssertCommandOneShotStepTypeMatchSucceeds`
- `TestMissionAssertCommandOneShotStepTypeMismatchFailsClearly`
- `TestMissionAssertCommandOneShotRequiredAuthorityMatchSucceeds`
- `TestMissionAssertCommandOneShotRequiredAuthorityMismatchFailsClearly`
- `TestMissionAssertCommandOneShotRequiresApprovalSucceedsWhenTrue`
- `TestMissionAssertCommandOneShotRequiresApprovalFailsClearlyWhenFalse`
- `TestMissionAssertCommandOneShotNoRequiresApprovalSucceedsWhenFalse`
- `TestMissionAssertCommandOneShotNoToolsSucceedsForEmptyAllowedTools`
- `TestMissionAssertCommandOneShotNoToolsFailsClearlyWhenToolsArePresent`
- `TestMissionAssertCommandOneShotHasToolSucceedsWhenToolIsPresent`
- `TestMissionAssertCommandOneShotHasToolFailsClearlyWhenToolIsAbsent`
- `TestMissionAssertCommandOneShotExactToolSucceedsWhenAllowedToolsExactlyMatch`
- `TestMissionAssertCommandOneShotExactToolFailsClearlyWhenAllowedToolsDoNotExactlyMatch`
- `TestMissionAssertCommandWaitSucceedsWhenStatusFileChangesBeforeTimeout`
- `TestMissionAssertCommandWaitSucceedsWhenAllowedToolsChangeBeforeTimeout`
- `TestMissionAssertCommandWaitSucceedsWhenAllowedToolsExactlyMatchBeforeTimeout`
- `TestMissionAssertCommandWaitTimesOutWhenValuesNeverMatch`
- `TestMissionAssertCommandWithMissingStatusFileReturnsClearError`
- `TestMissionAssertCommandWithInvalidJSONReturnsClearError`
- `TestMissionAssertCommandUsesSharedGatewayObservationReader`
- `TestMissionAssertStepCommandUsesSharedGatewayObservationReader`
- `TestMissionAssertCommandNoToolsAndHasToolReturnsClearArgumentError`
- `TestMissionAssertCommandNoToolsAndExactToolReturnsClearArgumentError`
- `TestMissionAssertCommandHasToolAndExactToolReturnsClearArgumentError`
- `TestMissionAssertCommandRequiresApprovalAndNoRequiresApprovalReturnsClearArgumentError`
- `TestMissionAssertStepCommandSucceedsWhenStatusMatchesMissionStep`
- `TestMissionAssertStepCommandSucceedsForZeroToolStepWhenStatusAllowedToolsIsNil`
- `TestMissionAssertStepCommandFailsClearlyWhenAllowedToolsDoNotExactlyMatch`
- `TestMissionAssertStepCommandUnknownStepReturnsClearError`
- `TestMissionAssertStepCommandWaitSucceedsWhenStatusChangesBeforeTimeout`
- `TestMissionAssertStepCommandWithInvalidMissionReturnsValidationError`

Exact helpers moved

- none
- shared snapshot/control/proof-verifier test helpers stayed in `main_test.go` because they are also used by the heavier bootstrap/runtime/watcher/operator-control family

Exact tests intentionally left in `main_test.go` and why

- Mission set-step tests stayed because they exercise control-file write and wait behavior, not the status/assertion family, and they continue to share status/control helpers with later runtime-sensitive tests.
- Mission package-logs, prune-store, and gateway mission-store logging tests stayed because they are a separate operational storage/logging family and were interleaved between the early and late mission status/assert blocks.
- Prompt/channel and agent CLI tests stayed because they cover different operator-facing surfaces with no mission status/assertion overlap.
- Bootstrap/runtime/watcher/operator-control tests stayed because they are the remaining heavier protected family from the split assessment and still share the reused mission status helpers.

Validation commands and results

- `gofmt -w cmd/picobot/main_test.go cmd/picobot/main_mission_status_test.go` -> passed
- `git diff --check` -> passed
- `go test -count=1 ./cmd/picobot` -> passed
- `go test -count=1 ./...` -> passed

Deferred next candidates from the `main_test` split assessment

- Mission bootstrap/runtime/watcher/operator-control family split
