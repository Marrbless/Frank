# V4-023 Rollback-Apply Phase Control Entry Before

## Branch

- `frank-v4-023-rollback-apply-phase-control-entry`

## HEAD

- `e229c08b7f74199e798ab2bff870b2da12dda16d`

## Tags At HEAD

- `frank-v4-022-rollback-apply-phase-progression`

## Ahead/Behind `upstream/main`

- ahead `421`
- behind `0`

## `git status --short --branch`

```text
## frank-v4-023-rollback-apply-phase-control-entry
```

## Baseline `go test -count=1 ./...` Result

- passed

## Exact Files Planned

- `internal/agent/loop.go`
- `internal/agent/tools/taskstate.go`
- `internal/agent/loop_processdirect_test.go`
- `docs/maintenance/V4_023_ROLLBACK_APPLY_PHASE_CONTROL_ENTRY_AFTER.md`

## Exact Control Surface Chosen

- existing direct operator command path in:
  - `internal/agent/loop.go`
  - `internal/agent/tools/taskstate.go`
- planned sibling command:
  - `ROLLBACK_APPLY_PHASE <job_id> <apply_id> <phase>`
- behavior scope:
  - advance phase on an existing committed rollback-apply record by `apply_id`
  - use committed rollback-apply record plus existing durable phase helper as authority
  - fail closed on missing records, invalid transitions, or invalid phase values
  - reuse existing `STATUS <job_id>` surface to verify read-model coherence after advancement

## Exact Non-Goals

- no rollback apply behavior
- no active runtime-pack pointer mutation
- no promotion behavior changes
- no evaluator execution
- no scoring behavior
- no autonomy changes
- no provider/channel behavior changes
- no cleanup outside this slice
- no dependency changes
- no commit
