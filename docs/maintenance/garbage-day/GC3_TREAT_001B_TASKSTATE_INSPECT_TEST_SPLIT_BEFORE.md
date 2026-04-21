# GC3-TREAT-001B TaskState Inspect Test Split Before

Date: 2026-04-21

- Branch: `frank-v3-foundation`
- HEAD: `551c4be1f0b86c1cb6128d98cdb82040bd5f2823`
- Tags at HEAD:
  - `frank-garbage-campaign-gc3-001a-taskstate-counters-clean`
- Ahead/behind `upstream/main`: `392 ahead / 0 behind`
- `git status --short --branch` at start:
  - `## frank-v3-foundation`
  - ` M internal/agent/tools/taskstate_test.go`
  - `?? docs/maintenance/garbage-day/GC3_TREAT_001B_TASKSTATE_INSPECT_TEST_SPLIT_BEFORE.md`
  - `?? internal/agent/tools/taskstate_inspect_test.go`
- Baseline `go test -count=1 ./...` result:
  - passed

## Exact tests selected for movement

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

## Helper movement decision

- No inspect-only private helpers were selected for movement.
- Shared helpers such as `testTaskStateJob`, JSON envelope readers, fixture writers, and adapter-boundary assertions remain in `internal/agent/tools/taskstate_test.go`.

## Exact non-goals

- do not change production code
- do not change test behavior, fixtures, or assertions
- do not move non-`OperatorInspect` TaskState test families
- do not rename tests except for any tiny mechanical adaptation required by the file split
- do not perform broader cleanup or V4 work

## Expected destination file

- `internal/agent/tools/taskstate_inspect_test.go`
