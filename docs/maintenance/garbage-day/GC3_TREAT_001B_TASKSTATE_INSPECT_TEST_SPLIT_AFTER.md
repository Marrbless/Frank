# GC3-TREAT-001B TaskState Inspect Test Split After

Date: 2026-04-21

- Branch: `frank-v3-foundation`
- HEAD: `551c4be1f0b86c1cb6128d98cdb82040bd5f2823`
- Scope completed from resumed dirty working set: yes
- Production behavior changed: no

## Files changed

- `internal/agent/tools/taskstate_test.go`
- `internal/agent/tools/taskstate_inspect_test.go`
- `docs/maintenance/garbage-day/GC3_TREAT_001B_TASKSTATE_INSPECT_TEST_SPLIT_BEFORE.md`
- `docs/maintenance/garbage-day/GC3_TREAT_001B_TASKSTATE_INSPECT_TEST_SPLIT_AFTER.md`

## git diff --stat

```text
 internal/agent/tools/taskstate_test.go | 500 ---------------------------------
 1 file changed, 500 deletions(-)
```

## git diff --numstat

```text
0	500	internal/agent/tools/taskstate_test.go
```

## Exact tests moved

- `TestTaskStateOperatorInspectWithoutValidatedPlanReturnsDeterministicError`
- `TestTaskStateOperatorInspectActiveExecutionContextZeroTreasuryRefPathUnchanged`
- `TestTaskStateOperatorInspectActiveExecutionContextSurfacesResolvedTreasuryPreflight`
- `TestTaskStateOperatorInspectActiveExecutionContextSurfacesResolvedCampaignPreflight`
- `TestTaskStateOperatorInspectSurfacesCampaignZohoEmailAddressing`
- `TestTaskStateOperatorInspectActiveAndPersistedPathsPreserveAdapterBoundaryContract`
- `TestTaskStateOperatorInspectUsesPersistedInspectablePlanWithoutMissionJob`
- `TestTaskStateOperatorInspectPersistedInspectablePlanPathUnchangedForTreasurySteps`
- `TestTaskStateOperatorInspectPersistedInspectablePlanWrongJobDoesNotBind`
- `TestTaskStateOperatorInspectPersistedInspectablePlanRejectsInvalidStep`
- `TestTaskStateOperatorInspectTerminalRuntimeUsesPersistedInspectablePlanWithoutMissionJob`

## Exact helpers moved

- None

## Tests intentionally left in taskstate_test.go and why

- No `TestTaskStateOperatorInspect...` tests were left behind in `internal/agent/tools/taskstate_test.go`.
- Adjacent tests such as `TestTaskStateApplyWaitingUserInputDoesNotCompleteDeniedApproval` and `TestTaskStateActivateStepMissingCampaignFailsClosed` were intentionally left because they belong to different TaskState families.
- Shared private helpers such as `testTaskStateJob`, `writeTaskStateTreasuryFixtures`, `mustStoreTaskStateCampaignFixture`, JSON readout helpers, and adapter-boundary assertions were intentionally left because they are reused by non-OperatorInspect tests and did not need movement to keep the inspect family coherent.

## Validation commands and results

- `gofmt -w internal/agent/tools/taskstate_test.go internal/agent/tools/taskstate_inspect_test.go`
  - passed
- `git diff --check`
  - passed with no output
- `go test -count=1 ./internal/agent/tools`
  - passed: `ok  	github.com/local/picobot/internal/agent/tools	13.328s`
- `go test -count=1 ./...`
  - passed
  - summary:
    - `ok  	github.com/local/picobot/cmd/picobot	14.122s`
    - `ok  	github.com/local/picobot/internal/agent	0.493s`
    - `ok  	github.com/local/picobot/internal/agent/memory	0.130s`
    - `ok  	github.com/local/picobot/internal/agent/skills	0.047s`
    - `ok  	github.com/local/picobot/internal/agent/tools	13.848s`
    - `ok  	github.com/local/picobot/internal/channels	0.631s`
    - `ok  	github.com/local/picobot/internal/config	0.014s`
    - `ok  	github.com/local/picobot/internal/cron	2.305s`
    - `ok  	github.com/local/picobot/internal/mcp	0.029s`
    - `ok  	github.com/local/picobot/internal/missioncontrol	9.512s`
    - `ok  	github.com/local/picobot/internal/providers	0.026s`
    - `ok  	github.com/local/picobot/internal/session	0.096s`
    - `?   	github.com/local/picobot/embeds	[no test files]`
    - `?   	github.com/local/picobot/internal/chat	[no test files]`
    - `?   	github.com/local/picobot/internal/heartbeat	[no test files]`
- `git status --short --branch --untracked-files=all`
  - passed
  - output:
    - `## frank-v3-foundation`
    - ` M internal/agent/tools/taskstate_test.go`
    - `?? docs/maintenance/garbage-day/GC3_TREAT_001B_TASKSTATE_INSPECT_TEST_SPLIT_AFTER.md`
    - `?? docs/maintenance/garbage-day/GC3_TREAT_001B_TASKSTATE_INSPECT_TEST_SPLIT_BEFORE.md`
    - `?? internal/agent/tools/taskstate_inspect_test.go`
