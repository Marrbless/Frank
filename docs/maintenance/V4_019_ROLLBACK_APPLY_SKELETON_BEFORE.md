# V4-019 Rollback Apply Skeleton Before

## Branch

`frank-v4-019-rollback-apply-skeleton`

## HEAD

`cbe2768c754cc4870d16ab44d5159a041d8d7871`

## Tags At HEAD

- `frank-v4-018-rollback-control-surface`

## Ahead / Behind `upstream/main`

- ahead: `417`
- behind: `0`

## git status --short --branch

```text
## frank-v4-019-rollback-apply-skeleton
```

## Baseline `go test -count=1 ./...` Result

```text
ok  	github.com/local/picobot/cmd/picobot	13.713s
?   	github.com/local/picobot/embeds	[no test files]
ok  	github.com/local/picobot/internal/agent	0.456s
ok  	github.com/local/picobot/internal/agent/memory	0.122s
ok  	github.com/local/picobot/internal/agent/skills	0.007s
ok  	github.com/local/picobot/internal/agent/tools	14.073s
ok  	github.com/local/picobot/internal/channels	0.586s
?   	github.com/local/picobot/internal/chat	[no test files]
ok  	github.com/local/picobot/internal/config	0.007s
ok  	github.com/local/picobot/internal/cron	2.305s
?   	github.com/local/picobot/internal/heartbeat	[no test files]
ok  	github.com/local/picobot/internal/mcp	0.030s
ok  	github.com/local/picobot/internal/missioncontrol	9.255s
ok  	github.com/local/picobot/internal/providers	0.025s
ok  	github.com/local/picobot/internal/session	0.088s
```

## Exact Files Planned

- `docs/maintenance/V4_019_ROLLBACK_APPLY_SKELETON_BEFORE.md`
- `docs/maintenance/V4_019_ROLLBACK_APPLY_SKELETON_AFTER.md`
- `internal/missioncontrol/rollback_apply_registry.go`
- `internal/missioncontrol/rollback_apply_registry_test.go`

## Exact Workflow / State Shapes Planned

- `RollbackApplyRef`
  - `apply_id`
- `RollbackApplyPhase`
  - bounded pre-execution workflow phases only:
  - `recorded`
  - `validated`
  - `ready_to_apply`
- `RollbackApplyActivationState`
  - explicit runtime activation semantics:
  - `unchanged`
- `RollbackApplyRecord`
  - `record_version`
  - `apply_id`
  - `rollback_id`
  - `phase`
  - `activation_state`
  - `requested_at`
  - `created_at`
  - `created_by`
- missioncontrol helpers
  - store/load/list helpers following the existing immutable registry pattern
  - creation helper that consumes an already committed rollback record and stores a rollback-apply record in `phase=recorded` with `activation_state=unchanged`
  - linkage validation that fails closed when the referenced rollback record is missing or invalid

## Exact Non-Goals

- mutating the active runtime-pack pointer
- reload or apply execution behavior
- promotion workflow changes
- evaluator execution
- scoring behavior
- autonomy changes
- provider or channel behavior changes
- read-model expansion outside the new registry surface
- dependency changes
- cleanup outside this slice
