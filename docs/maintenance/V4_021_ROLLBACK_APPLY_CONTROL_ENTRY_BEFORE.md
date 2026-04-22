# V4-021 Rollback-Apply Control Entry Before

## Branch

- `frank-v4-021-rollback-apply-control-entry`

## HEAD

- `39c6979250b1d7c22900f8250387ec970f0a7964`

## Tags At HEAD

- `frank-v4-020-rollback-apply-read-model`

## Ahead/Behind `upstream/main`

- ahead `419`
- behind `0`

## `git status --short --branch`

```text
## frank-v4-021-rollback-apply-control-entry
```

## Baseline `go test -count=1 ./...` Result

- passed

## Exact Files Planned

- `internal/missioncontrol/rollback_apply_registry.go`
- `internal/missioncontrol/rollback_apply_registry_test.go`
- `internal/agent/tools/taskstate.go`
- `internal/agent/loop.go`
- `internal/agent/loop_processdirect_test.go`
- `docs/maintenance/V4_021_ROLLBACK_APPLY_CONTROL_ENTRY_AFTER.md`

## Exact Control Surface Chosen

- existing direct operator command path in:
  - `internal/agent/loop.go`
  - `internal/agent/tools/taskstate.go`
- planned sibling command to `ROLLBACK_RECORD`:
  - `ROLLBACK_APPLY_RECORD <job_id> <rollback_id> <apply_id>`
- behavior scope:
  - create a rollback-apply record from an existing committed rollback record when absent
  - select the existing immutable rollback-apply record when the same `apply_id` already resolves to the same `rollback_id`
  - fail closed when rollback linkage is missing or mismatched
- existing `STATUS <job_id>` surface reused unchanged to verify read-model coherence after control entry

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
