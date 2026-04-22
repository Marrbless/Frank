# V4-022 Rollback-Apply Phase Progression Before

## Branch

- `frank-v4-022-rollback-apply-phase-progression`

## HEAD

- `ac3bcb1371fccd5467025a86e9d2b7aac4998014`

## Tags At HEAD

- `frank-v4-021-rollback-apply-control-entry`

## Ahead/Behind `upstream/main`

- ahead `420`
- behind `0`

## `git status --short --branch`

```text
## frank-v4-022-rollback-apply-phase-progression
```

## Baseline `go test -count=1 ./...` Result

- passed

## Exact Files Planned

- `internal/missioncontrol/rollback_apply_registry.go`
- `internal/missioncontrol/rollback_apply_registry_test.go`
- `docs/maintenance/V4_022_ROLLBACK_APPLY_PHASE_PROGRESSION_AFTER.md`

## Exact Workflow / Phase Changes Planned

- extend durable rollback-apply records with backward-compatible phase-transition metadata:
  - `phase_updated_at`
  - `phase_updated_by`
- backfill legacy committed rollback-apply records by normalizing missing transition metadata to the original create metadata
- add one non-executing phase-transition helper over existing rollback-apply records:
  - adjacent progression only
  - `recorded -> validated`
  - `validated -> ready_to_apply`
  - same-phase replay treated as idempotent
  - skipped or regressive transitions rejected
- preserve explicit invariant that `activation_state` remains `unchanged`

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
