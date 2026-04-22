Branch: `frank-v4-011-eval-suite-read-model`
HEAD: `d8bfd438f8f024e34686171edf4e5cd94cfeadca`

## Files changed

- `internal/missioncontrol/status.go`
- `internal/missioncontrol/store_project.go`
- `internal/missioncontrol/status_eval_suite_identity_test.go`
- `internal/agent/tools/taskstate_readout.go`
- `internal/agent/tools/taskstate_status_test.go`
- `docs/maintenance/V4_011_EVAL_SUITE_READ_MODEL_BEFORE.md`
- `docs/maintenance/V4_011_EVAL_SUITE_READ_MODEL_AFTER.md`

## Exact read-model fields added

- Added top-level status block `eval_suite_identity` to `OperatorStatusSummary`.
- Added top-level fields:
  - `state`
  - `suites`
- Added per-suite fields:
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
  - `error`

## Exact helpers added

- `WithEvalSuiteIdentity(summary OperatorStatusSummary, root string) OperatorStatusSummary`
- `LoadOperatorEvalSuiteIdentityStatus(root string) OperatorEvalSuiteIdentityStatus`
- private loaders:
  - `loadOperatorEvalSuiteStatuses`
  - `loadOperatorEvalSuiteStatus`
  - `operatorEvalSuiteStatusFromRecord`

## Exact behavior added

- Status now exposes committed eval-suite identity and linkage through the existing read-only operator status surface.
- Committed mission-status snapshots now carry `runtime_summary.eval_suite_identity`.
- Taskstate operator status now carries `eval_suite_identity` whenever a mission store root is present.
- Missing eval-suite storage returns top-level `not_configured`.
- Invalid stored shape or broken linkage returns top-level `invalid` and per-suite `error` details without executing any evaluator behavior.

## Exact tests added

- `TestLoadOperatorEvalSuiteIdentityStatusConfigured`
- `TestLoadOperatorEvalSuiteIdentityStatusNotConfigured`
- `TestLoadOperatorEvalSuiteIdentityStatusInvalidMissingLinkedRefs`
- `TestBuildCommittedMissionStatusSnapshotIncludesEvalSuiteIdentity`
- `TestTaskStateOperatorStatusSurfacesEvalSuiteIdentity`

## Validation commands and results

- `gofmt -w internal/missioncontrol/status.go internal/missioncontrol/store_project.go internal/missioncontrol/status_eval_suite_identity_test.go internal/agent/tools/taskstate_readout.go internal/agent/tools/taskstate_status_test.go`
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
## frank-v4-011-eval-suite-read-model
 M internal/agent/tools/taskstate_readout.go
 M internal/agent/tools/taskstate_status_test.go
 M internal/missioncontrol/status.go
 M internal/missioncontrol/store_project.go
?? docs/maintenance/V4_011_EVAL_SUITE_READ_MODEL_AFTER.md
?? docs/maintenance/V4_011_EVAL_SUITE_READ_MODEL_BEFORE.md
?? internal/missioncontrol/status_eval_suite_identity_test.go
```

## Deferred next V4 candidates

- evaluator execution and scoring behavior
- improvement execution control surfaces
- promotion workflow
- rollback workflow
- deeper inspect expansion only if a later slice needs more than identity and linkage
