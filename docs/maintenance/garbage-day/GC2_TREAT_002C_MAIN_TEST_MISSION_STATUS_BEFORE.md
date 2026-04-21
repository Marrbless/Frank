GC2-TREAT-002C main_test mission status family split before

- branch: `frank-v3-foundation`
- head: `feb79a0c487bdc70c55a82364aa583ccee977de6`
- tags at head: `frank-garbage-campaign-002b-main-test-inspect-clean`
- ahead/behind upstream: `385 ahead / 0 behind`
- git status --short --branch:

```text
## frank-v3-foundation
```

- baseline `go test -count=1 ./...` result: passed

Exact tests selected for movement

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

Exact helpers selected for movement

- none
- status snapshot, control-file, and proof-verifier test helpers remain in `main_test.go` because they are shared with the heavier bootstrap/runtime/watcher/operator-control family

Exact non-goals

- no production code changes
- no changes to prompt/channel, agent CLI, mission inspect, mission set-step, package/prune, gateway logging, or bootstrap/runtime/watcher/operator-control test families
- no shared helper extraction beyond what is strictly necessary for this family move
- no V4 work
- no dependency changes
- no test weakening or deletion

Expected destination file

- `cmd/picobot/main_mission_status_test.go`
