# V4-008 Improvement Run Read-Model After

## git diff --stat

```text
 internal/agent/tools/taskstate_readout.go     |   1 +
 internal/agent/tools/taskstate_status_test.go | 164 +++++++++++++++++++++++++-
 internal/missioncontrol/status.go             | 127 ++++++++++++++++++++
 internal/missioncontrol/store_project.go      |   1 +
 4 files changed, 290 insertions(+), 3 deletions(-)
```

`git diff --stat` does not include new untracked files. This slice also added:

- `internal/missioncontrol/status_improvement_run_identity_test.go`
- `docs/maintenance/V4_008_IMPROVEMENT_RUN_READ_MODEL_BEFORE.md`
- `docs/maintenance/V4_008_IMPROVEMENT_RUN_READ_MODEL_AFTER.md`

## git diff --numstat

```text
1	0	internal/agent/tools/taskstate_readout.go
161	3	internal/agent/tools/taskstate_status_test.go
127	0	internal/missioncontrol/status.go
1	0	internal/missioncontrol/store_project.go
```

## Files Changed

- `internal/missioncontrol/status.go`
- `internal/missioncontrol/store_project.go`
- `internal/missioncontrol/status_improvement_run_identity_test.go`
- `internal/agent/tools/taskstate_readout.go`
- `internal/agent/tools/taskstate_status_test.go`
- `docs/maintenance/V4_008_IMPROVEMENT_RUN_READ_MODEL_BEFORE.md`
- `docs/maintenance/V4_008_IMPROVEMENT_RUN_READ_MODEL_AFTER.md`

## Exact Read-Model Fields Added

Added `improvement_run_identity` to `missioncontrol.OperatorStatusSummary`.

Added nested read-only fields:

- `improvement_run_identity.state`
- `improvement_run_identity.runs[]`
- `improvement_run_identity.runs[].state`
- `improvement_run_identity.runs[].run_id`
- `improvement_run_identity.runs[].candidate_id`
- `improvement_run_identity.runs[].eval_suite_id`
- `improvement_run_identity.runs[].baseline_pack_id`
- `improvement_run_identity.runs[].candidate_pack_id`
- `improvement_run_identity.runs[].hot_update_id`
- `improvement_run_identity.runs[].created_at`
- `improvement_run_identity.runs[].completed_at`
- `improvement_run_identity.runs[].created_by`
- `improvement_run_identity.runs[].error`

State values used by this slice:

- `configured`
- `not_configured`
- `invalid`

## Exact Helpers Added

- `WithImprovementRunIdentity`
- `LoadOperatorImprovementRunIdentityStatus`

Private/local helpers:

- `loadOperatorImprovementRunStatuses`
- `loadOperatorImprovementRunStatus`
- `operatorImprovementRunStatusFromRecord`

## Exact Tests Added

Added in `internal/missioncontrol/status_improvement_run_identity_test.go`:

- `TestLoadOperatorImprovementRunIdentityStatusConfigured`
- `TestLoadOperatorImprovementRunIdentityStatusNotConfigured`
- `TestLoadOperatorImprovementRunIdentityStatusInvalidMissingLinkedRefs`
- `TestBuildCommittedMissionStatusSnapshotIncludesImprovementRunIdentity`

Changed in `internal/agent/tools/taskstate_status_test.go`:

- `TestTaskStateOperatorStatusSurfacesImprovementRunIdentity`
- updated existing JSON-envelope assertions in:
  - `TestTaskStateOperatorStatusActiveExecutionContextSurfacesFrankZohoMailboxBootstrapPreflight`
  - `TestTaskStateOperatorStatusSurfacesCampaignZohoEmailSendGateOnPersistedPath`
  - `TestTaskStateOperatorStatusActiveAndPersistedPathsPreserveAdapterBoundaryContract`

## Validation Commands And Results

- `gofmt -w internal/missioncontrol/status.go internal/missioncontrol/store_project.go internal/missioncontrol/status_improvement_run_identity_test.go internal/agent/tools/taskstate_readout.go internal/agent/tools/taskstate_status_test.go`
  - passed
- `git diff --check`
  - passed
- `go test -count=1 ./internal/missioncontrol`
  - passed
- `go test -count=1 ./...`
  - passed

## Deferred Next V4 Candidates

- eval-suite read-model exposure if operators need immutable suite visibility on status surfaces
- improvement result or scorecard record skeleton without evaluator execution
- run-history inspect expansion only if a later slice needs more than identity and linkage
- promotion and rollback durable records without apply or autonomy behavior
