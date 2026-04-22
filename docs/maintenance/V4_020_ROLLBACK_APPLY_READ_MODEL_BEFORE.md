# V4-020 Rollback Apply Read-Model Before

## Branch

- `frank-v4-020-rollback-apply-read-model`

## HEAD

- `b3f6329489e5872dd8635452dd36d2717c94edd8`

## Tags At HEAD

- `frank-v4-019-rollback-apply-skeleton`

## Ahead/Behind `upstream/main`

- ahead `418`
- behind `0`

## `git status --short --branch`

```text
## frank-v4-020-rollback-apply-read-model
```

## Baseline `go test -count=1 ./...` Result

- passed

## Exact Files Planned

- `internal/missioncontrol/status.go`
- `internal/missioncontrol/store_project.go`
- `internal/missioncontrol/status_rollback_apply_identity_test.go`
- `internal/agent/tools/taskstate_readout.go`
- `internal/agent/tools/taskstate_status_test.go`
- `docs/maintenance/V4_020_ROLLBACK_APPLY_READ_MODEL_AFTER.md`

## Exact Read-Model Surface Chosen

- existing operator status identity projection carried by:
  - `internal/missioncontrol/status.go`
  - committed snapshot `runtime_summary` via `BuildCommittedMissionStatusSnapshot`
  - existing taskstate operator-status adapter in `internal/agent/tools/taskstate_readout.go`
- no inspect-specific shape will be added because the current inspect surface is step-focused and does not already expose the V4 identity family

## Exact Non-Goals

- no rollback apply behavior
- no active runtime-pack pointer mutation
- no promotion behavior changes
- no evaluator execution
- no scoring behavior
- no autonomy changes
- no provider/channel behavior changes
- no dependency changes
- no cleanup outside this slice
- no commit
