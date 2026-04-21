# V4-008 Improvement Run Read-Model Before

## Branch

- `frank-v4-007-improvement-run-ledger-skeleton`

## HEAD

- `c927deeb993031146cd1a31cec23009dad053c42`

## Tags At HEAD

- `frank-v4-007-improvement-run-ledger-skeleton`

## Ahead/Behind Upstream

- `405 0`

## git status --short --branch

```text
## frank-v4-007-improvement-run-ledger-skeleton
```

## Baseline go test -count=1 ./... Result

- `go test -count=1 ./...`
- passed

## Exact Files Planned

- `internal/missioncontrol/status.go`
- `internal/missioncontrol/store_project.go`
- `internal/missioncontrol/status_improvement_run_identity_test.go`
- `internal/agent/tools/taskstate_readout.go`
- `internal/agent/tools/taskstate_status_test.go`
- `docs/maintenance/V4_008_IMPROVEMENT_RUN_READ_MODEL_AFTER.md`

## Exact Read-Model Surface Planned

Use the existing read-only `missioncontrol.OperatorStatusSummary` surface and thread it through the existing committed snapshot and taskstate status adapters.

Planned additions:

- top-level `improvement_run_identity` status block on `OperatorStatusSummary`
- deterministic `runs[]` entries loaded from committed improvement-run ledger records
- per-run read-only exposure of:
  - `run_id`
  - `candidate_id`
  - `eval_suite_id`
  - `baseline_pack_id`
  - `candidate_pack_id`
  - `hot_update_id`
  - `created_at`
  - `completed_at`
  - `created_by`
  - safe `error` when linkage or stored shape is invalid
- helper path:
  - `WithImprovementRunIdentity`
  - `LoadOperatorImprovementRunIdentityStatus`
  - private safe loaders that mark invalid linkage without mutating storage

Top-level states planned:

- `configured`
- `not_configured`
- `invalid`

Per-run states planned:

- `configured`
- `invalid`

## Exact Non-Goals

- no improvement execution
- no evaluator execution
- no scoring behavior
- no hot-update apply or reload behavior
- no promotion workflow
- no rollback workflow
- no autonomy changes
- no provider or channel behavior changes
- no storage-schema expansion beyond read-only status exposure needs
- no dependency changes
- no commit
