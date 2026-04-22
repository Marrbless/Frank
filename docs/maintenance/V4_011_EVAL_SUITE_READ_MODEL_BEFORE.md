# V4-011 Eval-Suite Read-Model Before

## Facts

- Canonical repo: `/mnt/d/pbot/picobot`
- Current branch: `frank-v4-011-eval-suite-read-model`
- Current HEAD: `d8bfd438f8f024e34686171edf4e5cd94cfeadca`
- Ahead/behind `upstream/main`: `409 0`
- `git status --short --branch` was clean:

```text
## frank-v4-011-eval-suite-read-model
```

- Required docs confirmed present:
  - `docs/maintenance/V4_ENTRY_DECISION.md`
  - `docs/maintenance/V4_006_EVAL_SUITE_SKELETON_AFTER.md`
  - `docs/maintenance/V4_007_IMPROVEMENT_RUN_LEDGER_AFTER.md`
  - `docs/maintenance/V4_008_IMPROVEMENT_RUN_READ_MODEL_AFTER.md`
  - `docs/maintenance/V4_009_CANDIDATE_RESULT_SCORECARD_AFTER.md`
  - `docs/maintenance/V4_010_CANDIDATE_RESULT_READ_MODEL_AFTER.md`
- Baseline validator:
  - `go test -count=1 ./...`
  - passed

## Assumptions

- The narrowest truthful read-only hook is the existing `missioncontrol.OperatorStatusSummary` status surface, because V4-004, V4-005, V4-008, and V4-010 already expose adjacent committed identity/linkage there and thread that surface into both committed snapshot and taskstate status adapters.
- This slice should add a top-level `eval_suite_identity` block rather than expanding inspect output or creating a new command surface.
- Existing read-only adapters in `internal/missioncontrol/store_project.go` and `internal/agent/tools/taskstate_readout.go` should be updated only enough to carry the new status block through unchanged.

## Exact Files Planned

- `internal/missioncontrol/status.go`
- `internal/missioncontrol/store_project.go`
- `internal/missioncontrol/status_eval_suite_identity_test.go`
- `internal/agent/tools/taskstate_readout.go`
- `internal/agent/tools/taskstate_status_test.go`
- `docs/maintenance/V4_011_EVAL_SUITE_READ_MODEL_AFTER.md`

## Exact Read-Model Surface Planned

Use the existing read-only `missioncontrol.OperatorStatusSummary` surface and expose committed eval-suite identity/linkage in a single top-level `eval_suite_identity` block.

Planned per-suite fields:

- `state`
- `eval_suite_id`
- `candidate_id`
- `baseline_pack_id`
- `candidate_pack_id`
- `rubric_ref`
- `train_corpus_ref`
- `holdout_corpus_ref`
- `evaluator_ref`
- `frozen_for_run`
- `created_at`
- `created_by`
- safe `error` when stored shape or linkage is invalid

Top-level states planned:

- `configured`
- `not_configured`
- `invalid`

Per-suite states planned:

- `configured`
- `invalid`

## Allowed Edit Surface

- `internal/missioncontrol/`
- existing read-only status adapter hook in `internal/agent/tools/taskstate_readout.go`
- adjacent tests
- this checkpoint's before/after notes in `docs/maintenance/`

## No-Touch / Non-Goals

- no evaluator execution
- no scoring behavior
- no improvement execution
- no apply/reload behavior
- no promotion workflow
- no rollback workflow
- no autonomy changes
- no provider/channel behavior changes
- no dependency changes
- no cleanup outside this slice
- no commit
