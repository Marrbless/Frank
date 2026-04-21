# V4-007 Improvement Run Ledger Before

## Branch

- `frank-v4-007-improvement-run-ledger-skeleton`

## HEAD

- `aecd0ab5cedd3808a6ac95be72615ee3bcccfcd0`

## Tags At HEAD

- `frank-v4-006-eval-suite-skeleton`

## Ahead/Behind Upstream

- `404 0`

## git status --short --branch

```text
## frank-v4-007-improvement-run-ledger-skeleton
```

## Baseline `go test -count=1 ./...` Result

- passed

## Exact Files Planned

- `internal/missioncontrol/improvement_run_registry.go`
- `internal/missioncontrol/improvement_run_registry_test.go`
- `docs/maintenance/V4_007_IMPROVEMENT_RUN_LEDGER_BEFORE.md`
- `docs/maintenance/V4_007_IMPROVEMENT_RUN_LEDGER_AFTER.md`

## Exact Record / Storage Shapes Planned

### `ImprovementRunRef`

- same-package durable ref wrapper with canonical `run_id`

### `ImprovementRunRecord`

- bounded append-only improvement-run record with:
  - `record_version`
  - `run_id`
  - `objective`
  - `execution_plane`
  - `execution_host`
  - `mission_family`
  - `target_type`
  - `target_ref`
  - `surface_class`
  - `candidate_id`
  - `eval_suite_id`
  - `baseline_pack_id`
  - `candidate_pack_id`
  - `hot_update_id`
  - `state`
  - `decision`
  - `created_at`
  - `completed_at`
  - `stop_reason`
  - `created_by`

### Planned Storage / Validation Helpers

- improvement-run directory and path helpers under the existing V4 mission-store surface
- normalize / validate helpers for run ref and run record
- append-only store / load / list helpers for run records
- optional linkage validation against:
  - `ImprovementCandidateRecord` for `candidate_id`
  - `EvalSuiteRecord` for `eval_suite_id`
  - `RuntimePackRecord` for `baseline_pack_id` and `candidate_pack_id`
  - `HotUpdateGateRecord` for optional `hot_update_id`
- consistency validation where applicable:
  - linked candidate and eval-suite records must agree with stored baseline/candidate pack refs
  - if `hot_update_id` is present, the linked gate must agree with stored candidate/baseline pack refs
  - append-only semantics reject divergent rewrites for an existing `run_id` while allowing identical replay

## Exact Non-Goals

- no improvement execution
- no evaluator execution or scoring
- no apply or reload behavior
- no promotion workflow
- no rollback workflow
- no autonomy changes
- no provider or channel behavior changes
- no operator read-model exposure in this slice
- no mutation of runtime-pack, hot-update, candidate, or eval-suite storage contracts
- no cleanup outside this slice
