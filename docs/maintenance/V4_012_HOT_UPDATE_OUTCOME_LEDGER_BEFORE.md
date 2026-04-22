# V4-012 Hot-Update Outcome Ledger Before

## Branch

- `frank-v4-012-hot-update-outcome-ledger`

## HEAD

- `f801099223a53958036e5dc2015a2c8262b23298`

## Tags At HEAD

- `frank-v4-011-eval-suite-read-model`

## Ahead/Behind Upstream

- `410 0`

## git status --short --branch

```text
## frank-v4-012-hot-update-outcome-ledger
```

## Baseline go test -count=1 ./... Result

- `go test -count=1 ./...`
- passed

## Exact Files Planned

- `internal/missioncontrol/hot_update_outcome_registry.go`
- `internal/missioncontrol/hot_update_outcome_registry_test.go`
- `docs/maintenance/V4_012_HOT_UPDATE_OUTCOME_LEDGER_BEFORE.md`
- `docs/maintenance/V4_012_HOT_UPDATE_OUTCOME_LEDGER_AFTER.md`

## Exact Record/Storage Shapes Planned

Bounded storage-only record and helpers in `internal/missioncontrol`:

- `HotUpdateOutcomeKind`
- `HotUpdateOutcomeRef`
- `HotUpdateOutcomeRecord`
- `ErrHotUpdateOutcomeRecordNotFound`
- `StoreHotUpdateOutcomesDir`
- `StoreHotUpdateOutcomePath`
- `NormalizeHotUpdateOutcomeRef`
- `NormalizeHotUpdateOutcomeRecord`
- `HotUpdateOutcomeGateRef`
- optional ref helpers where applicable:
  - `HotUpdateOutcomeImprovementCandidateRef`
  - `HotUpdateOutcomeImprovementRunRef`
  - `HotUpdateOutcomeCandidateResultRef`
  - `HotUpdateOutcomeCandidatePackRef`
- `ValidateHotUpdateOutcomeRef`
- `ValidateHotUpdateOutcomeRecord`
- `StoreHotUpdateOutcomeRecord`
- `LoadHotUpdateOutcomeRecord`
- `ListHotUpdateOutcomeRecords`

Planned record fields:

- `record_version`
- `outcome_id`
- `hot_update_id`
- optional `candidate_id`
- optional `run_id`
- optional `candidate_result_id`
- optional `candidate_pack_id`
- `outcome_kind`
- optional `reason`
- optional `notes`
- `outcome_at`
- `created_at`
- `created_by`

Planned storage/replay contract:

- one immutable JSON file per `outcome_id` under `runtime_packs/hot_update_outcomes/`
- exact replay is idempotent
- divergent duplicate by the same `outcome_id` is rejected fail-closed
- listing is deterministic and sorted by filename / outcome id

Planned linkage checks:

- `hot_update_id` must resolve to an existing hot-update gate
- if `candidate_pack_id` is present, it must resolve and match the gate candidate pack
- if `candidate_id` is present, it must resolve and, when pack or hot-update linkage exists on the candidate, match
- if `run_id` is present, it must resolve and, when present, match `candidate_id`, `candidate_pack_id`, and `hot_update_id`
- if `candidate_result_id` is present, it must resolve and, when present, match `candidate_id`, `run_id`, `candidate_pack_id`, and `hot_update_id`

## Exact Non-Goals

- no apply/reload behavior
- no promotion workflow
- no rollback workflow
- no evaluator execution
- no scoring calculation behavior
- no autonomy changes
- no provider/channel behavior changes
- no read-model or inspect exposure unless forced by storage implementation
- no dependency changes
- no cleanup outside this slice
- no commit
