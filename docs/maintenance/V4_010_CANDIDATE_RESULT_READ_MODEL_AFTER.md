Branch: `frank-v4-010-candidate-result-read-model`
HEAD: `fc61b9f5b3d053ebcc6917fc4e3131e4837bbd25`

## git diff --stat

```text
 internal/agent/tools/taskstate_readout.go     |   1 +
 internal/agent/tools/taskstate_status_test.go | 192 +++++++++++++++++++++++++-
 internal/missioncontrol/status.go             | 145 +++++++++++++++++++
 internal/missioncontrol/store_project.go      |   1 +
 4 files changed, 336 insertions(+), 3 deletions(-)
```

## git diff --numstat

```text
1	0	internal/agent/tools/taskstate_readout.go
189	3	internal/agent/tools/taskstate_status_test.go
145	0	internal/missioncontrol/status.go
1	0	internal/missioncontrol/store_project.go
```

## Files changed

- `internal/missioncontrol/status.go`
- `internal/missioncontrol/store_project.go`
- `internal/missioncontrol/status_candidate_result_identity_test.go`
- `internal/agent/tools/taskstate_readout.go`
- `internal/agent/tools/taskstate_status_test.go`
- `docs/maintenance/V4_010_CANDIDATE_RESULT_READ_MODEL_BEFORE.md`
- `docs/maintenance/V4_010_CANDIDATE_RESULT_READ_MODEL_AFTER.md`

## Exact read-model fields added

- Added top-level status block `candidate_result_identity` to `OperatorStatusSummary`.
- Added top-level fields:
  - `state`
  - `results`
- Added per-result fields:
  - `state`
  - `result_id`
  - `run_id`
  - `candidate_id`
  - `eval_suite_id`
  - `baseline_pack_id`
  - `candidate_pack_id`
  - `hot_update_id`
  - `baseline_score`
  - `train_score`
  - `holdout_score`
  - `complexity_score`
  - `compatibility_score`
  - `resource_score`
  - `regression_flags`
  - `decision`
  - `notes`
  - `created_at`
  - `created_by`
  - `error`

## Exact helpers added

- `WithCandidateResultIdentity(summary OperatorStatusSummary, root string) OperatorStatusSummary`
- `LoadOperatorCandidateResultIdentityStatus(root string) OperatorCandidateResultIdentityStatus`
- private loaders:
  - `loadOperatorCandidateResultStatuses`
  - `loadOperatorCandidateResultStatus`
  - `operatorCandidateResultStatusFromRecord`

## Exact tests added

- `TestLoadOperatorCandidateResultIdentityStatusConfigured`
- `TestLoadOperatorCandidateResultIdentityStatusNotConfigured`
- `TestLoadOperatorCandidateResultIdentityStatusInvalidMissingLinkedRefs`
- `TestBuildCommittedMissionStatusSnapshotIncludesCandidateResultIdentity`
- `TestTaskStateOperatorStatusSurfacesCandidateResultIdentity`

## Validation commands and results

- `gofmt -w internal/missioncontrol/status.go internal/missioncontrol/store_project.go internal/missioncontrol/status_candidate_result_identity_test.go internal/agent/tools/taskstate_readout.go internal/agent/tools/taskstate_status_test.go`
  - passed
- `git diff --check`
  - passed
- `go test -count=1 ./internal/missioncontrol`
  - passed
- `go test -count=1 ./...`
  - passed
- `git status --short --branch`
  - result:

```text
## frank-v4-010-candidate-result-read-model
 M internal/agent/tools/taskstate_readout.go
 M internal/agent/tools/taskstate_status_test.go
 M internal/missioncontrol/status.go
 M internal/missioncontrol/store_project.go
?? docs/maintenance/V4_010_CANDIDATE_RESULT_READ_MODEL_AFTER.md
?? docs/maintenance/V4_010_CANDIDATE_RESULT_READ_MODEL_BEFORE.md
?? internal/missioncontrol/status_candidate_result_identity_test.go
```

## Deferred next V4 candidates

- Eval-suite read-model / inspect exposure
- Promotion record skeleton without apply behavior
- Rollback record skeleton without apply behavior
- Deeper candidate-result inspect surfaces only if later operator controls need more than identity and linkage
