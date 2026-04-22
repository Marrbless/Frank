Branch: `frank-v4-013-hot-update-outcome-read-model`
HEAD: `367f0b87453230b7e0d6041a365149e26fba6d6d`

## git diff --stat

```text
 internal/agent/tools/taskstate_readout.go     |   1 +
 internal/agent/tools/taskstate_status_test.go | 211 +++++++++++++++++++++++++-
 internal/missioncontrol/status.go             | 133 ++++++++++++++++
 internal/missioncontrol/store_project.go      |   1 +
 4 files changed, 343 insertions(+), 3 deletions(-)
```

## git diff --numstat

```text
1	0	internal/agent/tools/taskstate_readout.go
208	3	internal/agent/tools/taskstate_status_test.go
133	0	internal/missioncontrol/status.go
1	0	internal/missioncontrol/store_project.go
```

## Files changed

- `internal/missioncontrol/status.go`
- `internal/missioncontrol/store_project.go`
- `internal/missioncontrol/status_hot_update_outcome_identity_test.go`
- `internal/agent/tools/taskstate_readout.go`
- `internal/agent/tools/taskstate_status_test.go`
- `docs/maintenance/V4_013_HOT_UPDATE_OUTCOME_READ_MODEL_BEFORE.md`
- `docs/maintenance/V4_013_HOT_UPDATE_OUTCOME_READ_MODEL_AFTER.md`

## Exact read-model fields added

- Added top-level status block `hot_update_outcome_identity` to `OperatorStatusSummary`.
- Added top-level fields:
  - `state`
  - `outcomes`
- Added per-outcome fields:
  - `state`
  - `outcome_id`
  - `hot_update_id`
  - `candidate_id`
  - `run_id`
  - `candidate_result_id`
  - `candidate_pack_id`
  - `outcome_kind`
  - `reason`
  - `notes`
  - `outcome_at`
  - `created_at`
  - `created_by`
  - `error`

## Exact helpers added

- `WithHotUpdateOutcomeIdentity(summary OperatorStatusSummary, root string) OperatorStatusSummary`
- `LoadOperatorHotUpdateOutcomeIdentityStatus(root string) OperatorHotUpdateOutcomeIdentityStatus`
- private loaders:
  - `loadOperatorHotUpdateOutcomeStatuses`
  - `loadOperatorHotUpdateOutcomeStatus`
  - `operatorHotUpdateOutcomeStatusFromRecord`

## Exact tests added

- `TestLoadOperatorHotUpdateOutcomeIdentityStatusConfigured`
- `TestLoadOperatorHotUpdateOutcomeIdentityStatusNotConfigured`
- `TestLoadOperatorHotUpdateOutcomeIdentityStatusInvalidMissingLinkedRefs`
- `TestBuildCommittedMissionStatusSnapshotIncludesHotUpdateOutcomeIdentity`
- `TestTaskStateOperatorStatusSurfacesHotUpdateOutcomeIdentity`

## Validation commands and results

- `gofmt -w internal/missioncontrol/status.go internal/missioncontrol/store_project.go internal/missioncontrol/status_hot_update_outcome_identity_test.go internal/agent/tools/taskstate_readout.go internal/agent/tools/taskstate_status_test.go`
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
## frank-v4-013-hot-update-outcome-read-model
 M internal/agent/tools/taskstate_readout.go
 M internal/agent/tools/taskstate_status_test.go
 M internal/missioncontrol/status.go
 M internal/missioncontrol/store_project.go
?? docs/maintenance/V4_013_HOT_UPDATE_OUTCOME_READ_MODEL_AFTER.md
?? docs/maintenance/V4_013_HOT_UPDATE_OUTCOME_READ_MODEL_BEFORE.md
?? internal/missioncontrol/status_hot_update_outcome_identity_test.go
```

## Deferred next V4 candidates

- promotion record skeleton without apply behavior
- rollback record skeleton without apply behavior
- deeper outcome inspect expansion only if a later slice needs more than identity and linkage
- apply/reload-facing control surfaces only in a later slice, still separate from evaluator/scoring behavior
