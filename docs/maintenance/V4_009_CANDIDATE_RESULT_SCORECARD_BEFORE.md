# V4-009 Candidate Result Scorecard Before

## Branch

- `frank-v4-009-candidate-result-scorecard-skeleton`

## HEAD

- `0fd33aa6c5a3697fc6b4a9259e6ba71ca37fd747`

## Tags At HEAD

- `frank-v4-block-001-writer-lease-stable`

## Ahead/Behind Upstream

- `407 0`

## git status --short --branch

```text
## frank-v4-009-candidate-result-scorecard-skeleton
```

## Baseline go test -count=1 ./... Result

- `go test -count=1 ./...`
- passed

## Exact Files Planned

- `internal/missioncontrol/candidate_result_registry.go`
- `internal/missioncontrol/candidate_result_registry_test.go`
- `docs/maintenance/V4_009_CANDIDATE_RESULT_SCORECARD_AFTER.md`

## Exact Record / Storage Shapes Planned

Planned durable record:

- `CandidateResultRef`
- `CandidateResultRecord`

Planned storage path:

- `runtime_packs/candidate_results/<result_id>.json`

Planned record shape:

- `record_version`
- `result_id`
- `run_id`
- `candidate_id`
- `eval_suite_id`
- `baseline_pack_id`
- `candidate_pack_id`
- optional `hot_update_id`
- recorded score fields as stored values only:
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

Planned helper surface:

- `StoreCandidateResultsDir`
- `StoreCandidateResultPath`
- `NormalizeCandidateResultRef`
- `NormalizeCandidateResultRecord`
- `CandidateResultImprovementRunRef`
- `CandidateResultImprovementCandidateRef`
- `CandidateResultEvalSuiteRef`
- `CandidateResultBaselinePackRef`
- `CandidateResultCandidatePackRef`
- `CandidateResultHotUpdateGateRef`
- `ValidateCandidateResultRef`
- `ValidateCandidateResultRecord`
- `StoreCandidateResultRecord`
- `LoadCandidateResultRecord`
- `ListCandidateResultRecords`

Planned replay semantics:

- immutable / append-only by `result_id`
- exact replay of the same normalized record is a no-op
- divergent second write for the same `result_id` is rejected

Planned linkage validation:

- `run_id` resolves to committed `ImprovementRunRecord`
- `candidate_id`, `eval_suite_id`, `baseline_pack_id`, `candidate_pack_id`, and optional `hot_update_id` validate as refs
- linked refs resolve and match the committed improvement-run linkage where applicable

## Exact Non-Goals

- no evaluator execution
- no scoring calculation behavior
- no improvement execution
- no apply/reload behavior
- no promotion workflow
- no rollback workflow
- no autonomy changes
- no provider/channel behavior changes
- no read-model exposure unless unexpectedly required
- no dependency changes
- no cleanup outside this slice
- no commit
