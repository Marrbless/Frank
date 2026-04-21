# V4-005 Candidate Read-Model After

## git diff --stat

```text
 internal/agent/tools/taskstate_readout.go     |   1 +
 internal/agent/tools/taskstate_status_test.go |  96 ++++++++++++++++++-
 internal/missioncontrol/status.go             | 128 ++++++++++++++++++++++++++
 internal/missioncontrol/store_project.go      |   1 +
 4 files changed, 223 insertions(+), 3 deletions(-)
```

`git diff` does not include new untracked files. This slice also added:

- `internal/missioncontrol/status_improvement_candidate_identity_test.go`
- `docs/maintenance/V4_005_CANDIDATE_READ_MODEL_BEFORE.md`
- `docs/maintenance/V4_005_CANDIDATE_READ_MODEL_AFTER.md`

## git diff --numstat

```text
1	0	internal/agent/tools/taskstate_readout.go
93	3	internal/agent/tools/taskstate_status_test.go
128	0	internal/missioncontrol/status.go
1	0	internal/missioncontrol/store_project.go
```

## Files Changed

- `internal/missioncontrol/status.go`
- `internal/missioncontrol/store_project.go`
- `internal/missioncontrol/status_improvement_candidate_identity_test.go`
- `internal/agent/tools/taskstate_readout.go`
- `internal/agent/tools/taskstate_status_test.go`
- `docs/maintenance/V4_005_CANDIDATE_READ_MODEL_BEFORE.md`
- `docs/maintenance/V4_005_CANDIDATE_READ_MODEL_AFTER.md`

## Exact Read-Model Fields Added

Added `improvement_candidate_identity` to `missioncontrol.OperatorStatusSummary`.

Added nested read-only fields:

- `improvement_candidate_identity.state`
- `improvement_candidate_identity.candidates[]`
- `improvement_candidate_identity.candidates[].state`
- `improvement_candidate_identity.candidates[].candidate_id`
- `improvement_candidate_identity.candidates[].baseline_pack_id`
- `improvement_candidate_identity.candidates[].candidate_pack_id`
- `improvement_candidate_identity.candidates[].source_workspace_ref`
- `improvement_candidate_identity.candidates[].source_summary`
- `improvement_candidate_identity.candidates[].validation_basis_refs`
- `improvement_candidate_identity.candidates[].hot_update_id`
- `improvement_candidate_identity.candidates[].created_at`
- `improvement_candidate_identity.candidates[].created_by`
- `improvement_candidate_identity.candidates[].error`

State values used by this slice:

- `configured`
- `not_configured`
- `invalid`

## Exact Helpers Added

- `WithImprovementCandidateIdentity`
- `LoadOperatorImprovementCandidateIdentityStatus`

Private/local helpers:

- `loadOperatorImprovementCandidateStatuses`
- `loadOperatorImprovementCandidateStatus`
- `operatorImprovementCandidateStatusFromRecord`

## Exact Tests Added

Added in `internal/missioncontrol/status_improvement_candidate_identity_test.go`:

- `TestLoadOperatorImprovementCandidateIdentityStatusConfigured`
- `TestLoadOperatorImprovementCandidateIdentityStatusNotConfigured`
- `TestLoadOperatorImprovementCandidateIdentityStatusInvalidMissingLinkedRefs`
- `TestBuildCommittedMissionStatusSnapshotIncludesImprovementCandidateIdentity`

Changed in `internal/agent/tools/taskstate_status_test.go`:

- `TestTaskStateOperatorStatusSurfacesImprovementCandidateIdentity`
- updated existing JSON-envelope assertions in:
  - `TestTaskStateOperatorStatusActiveExecutionContextSurfacesFrankZohoMailboxBootstrapPreflight`
  - `TestTaskStateOperatorStatusSurfacesCampaignZohoEmailSendGateOnPersistedPath`
  - `TestTaskStateOperatorStatusActiveAndPersistedPathsPreserveAdapterBoundaryContract`

## Validation Commands And Results

- `gofmt -w internal/missioncontrol/status.go internal/missioncontrol/store_project.go internal/missioncontrol/status_improvement_candidate_identity_test.go internal/agent/tools/taskstate_readout.go internal/agent/tools/taskstate_status_test.go`
  - passed
- `git diff --check`
  - passed
- `go test -count=1 ./internal/missioncontrol`
  - passed
- `go test -count=1 ./...`
  - passed
- `git status --short --branch`
  - final:

```text
## frank-v4-001-pack-registry-foundation
 M internal/agent/tools/taskstate_readout.go
 M internal/agent/tools/taskstate_status_test.go
 M internal/missioncontrol/status.go
 M internal/missioncontrol/store_project.go
?? docs/maintenance/V4_005_CANDIDATE_READ_MODEL_AFTER.md
?? docs/maintenance/V4_005_CANDIDATE_READ_MODEL_BEFORE.md
?? internal/missioncontrol/status_improvement_candidate_identity_test.go
```

## Deferred Next V4 Candidates

- improvement candidate inspect/read-model expansion only if a later slice needs per-candidate deep detail beyond status summary
- append-only improvement run / ledger skeleton linked from candidate records
- eval-suite record read-model exposure after the immutable eval contract exists
- promotion and rollback durable records without apply or autonomy behavior
