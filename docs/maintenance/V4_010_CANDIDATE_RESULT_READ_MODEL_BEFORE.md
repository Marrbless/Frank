# V4-010 Candidate Result Read-Model Before

## Branch

- `frank-v4-010-candidate-result-read-model`

## HEAD

- `fc61b9f5b3d053ebcc6917fc4e3131e4837bbd25`

## Tags At HEAD

- `frank-v4-009-candidate-result-scorecard-skeleton`

## Ahead/Behind Upstream

- `408 0`

## git status --short --branch

```text
## frank-v4-010-candidate-result-read-model
```

## Baseline go test -count=1 ./... Result

- `go test -count=1 ./...`
- passed

## Exact Files Planned

- `internal/missioncontrol/status.go`
- `internal/missioncontrol/store_project.go`
- `internal/missioncontrol/status_candidate_result_identity_test.go`
- `internal/agent/tools/taskstate_readout.go`
- `internal/agent/tools/taskstate_status_test.go`
- `docs/maintenance/V4_010_CANDIDATE_RESULT_READ_MODEL_AFTER.md`

## Exact Read-Model Surface Planned

Use the existing read-only `missioncontrol.OperatorStatusSummary` surface and thread it through the existing committed snapshot and taskstate status adapters.

Planned additions:

- top-level `candidate_result_identity` status block on `OperatorStatusSummary`
- deterministic `results[]` entries loaded from committed candidate-result records
- per-result read-only exposure of:
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
  - safe `error` when linkage or stored shape is invalid

Top-level states planned:

- `configured`
- `not_configured`
- `invalid`

Per-result states planned:

- `configured`
- `invalid`

## Exact Non-Goals

- no evaluator execution
- no scoring calculation behavior
- no improvement execution
- no apply/reload behavior
- no promotion workflow
- no rollback workflow
- no autonomy changes
- no provider or channel behavior changes
- no new inspect or command surface beyond the existing status/read-model hook
- no dependency changes
- no cleanup outside this slice
- no commit
