# V4-013 Hot-Update Outcome Read-Model Before

## Facts

- Canonical repo: `/mnt/d/pbot/picobot`
- Current branch: `frank-v4-013-hot-update-outcome-read-model`
- Current HEAD: `367f0b87453230b7e0d6041a365149e26fba6d6d`
- Ahead/behind `upstream/main`: `411 0`
- `git status --short --branch` was clean:

```text
## frank-v4-013-hot-update-outcome-read-model
```

- Required docs confirmed present:
  - `docs/maintenance/V4_ENTRY_DECISION.md`
  - `docs/maintenance/V4_012_HOT_UPDATE_OUTCOME_LEDGER_AFTER.md`
  - `docs/maintenance/V4_010_CANDIDATE_RESULT_READ_MODEL_AFTER.md`
  - `docs/maintenance/V4_008_IMPROVEMENT_RUN_READ_MODEL_AFTER.md`
- Baseline validator:
  - `go test -count=1 ./...`
  - passed

## Assumptions

- The narrowest truthful read-only hook is the existing `missioncontrol.OperatorStatusSummary` status surface, because V4-008, V4-010, and V4-011 already expose adjacent committed identity/linkage there and thread that surface through both committed snapshot and taskstate status adapters.
- This slice should add a top-level `hot_update_outcome_identity` block rather than expanding inspect output or creating a new command surface.
- Existing read-only adapters in `internal/missioncontrol/store_project.go` and `internal/agent/tools/taskstate_readout.go` should be updated only enough to carry the new status block through unchanged.

## Exact Files Planned

- `internal/missioncontrol/status.go`
- `internal/missioncontrol/store_project.go`
- `internal/missioncontrol/status_hot_update_outcome_identity_test.go`
- `internal/agent/tools/taskstate_readout.go`
- `internal/agent/tools/taskstate_status_test.go`
- `docs/maintenance/V4_013_HOT_UPDATE_OUTCOME_READ_MODEL_AFTER.md`

## Exact Read-Model Surface Planned

Use the existing read-only `missioncontrol.OperatorStatusSummary` surface and expose committed hot-update outcome identity/linkage in a single top-level `hot_update_outcome_identity` block.

Planned per-outcome fields:

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
- safe `error` when stored shape or linkage is invalid

Top-level states planned:

- `configured`
- `not_configured`
- `invalid`

Per-outcome states planned:

- `configured`
- `invalid`

## Allowed Edit Surface

- `internal/missioncontrol/`
- existing read-only status adapter hook in `internal/agent/tools/taskstate_readout.go`
- adjacent tests
- this checkpoint's before/after notes in `docs/maintenance/`

## No-Touch / Non-Goals

- no apply/reload behavior
- no promotion workflow
- no rollback workflow
- no evaluator execution
- no scoring calculation behavior
- no autonomy changes
- no provider/channel behavior changes
- no dependency changes
- no cleanup outside this slice
- no commit
