# V4-006 Eval Suite Skeleton Before

## Branch

- `frank-v4-001-pack-registry-foundation`

## HEAD

- `f1d7887aa8d7f51e324151444de356eb3b1badb0`

## Tags At HEAD

- `frank-v4-005-candidate-read-model`

## Ahead/Behind Upstream

- `403 0`

## git status --short --branch

```text
## frank-v4-001-pack-registry-foundation
```

## Baseline `go test -count=1 ./...` Result

- passed

## Exact Files Planned

- `internal/missioncontrol/eval_suite_registry.go`
- `internal/missioncontrol/eval_suite_registry_test.go`
- `docs/maintenance/V4_006_EVAL_SUITE_SKELETON_BEFORE.md`
- `docs/maintenance/V4_006_EVAL_SUITE_SKELETON_AFTER.md`

## Exact Record / Storage Shapes Planned

### `EvalSuiteRef`

- same-package durable ref wrapper with canonical `eval_suite_id`

### `EvalSuiteRecord`

- bounded immutable eval-suite record with:
  - `record_version`
  - `eval_suite_id`
  - `rubric_ref`
  - `train_corpus_ref`
  - `holdout_corpus_ref`
  - `evaluator_ref`
  - `negative_case_count`
  - `boundary_case_count`
  - `frozen_for_run`
  - `candidate_id`
  - `baseline_pack_id`
  - `candidate_pack_id`
  - `created_at`
  - `created_by`

### Planned Storage / Validation Helpers

- eval-suite directory and path helpers under the existing V4 mission-store surface
- normalize / validate helpers for eval-suite ref and eval-suite record
- store / load / list helpers for eval-suite records
- optional linkage validation against:
  - `ImprovementCandidateRecord` for `candidate_id`
  - `RuntimePackRecord` for `baseline_pack_id` and `candidate_pack_id`
- consistency validation where applicable:
  - `frozen_for_run` must be true for stored immutable suite records
  - if `candidate_id` is present, any stored `baseline_pack_id` and `candidate_pack_id` must match the linked candidate record

## Exact Non-Goals

- no evaluator execution
- no scoring behavior
- no improvement execution
- no apply or reload behavior
- no promotion workflow
- no rollback workflow
- no autonomy changes
- no provider or channel behavior changes
- no operator read-model exposure in this slice
- no mutation of runtime-pack, hot-update, or candidate storage contracts
- no cleanup outside this slice
