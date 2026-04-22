# V4-025 Rollback Pointer Switch Before

- branch: `frank-v4-025-rollback-pointer-switch-skeleton`
- `HEAD`: `f98e2c8aa191c8943d712f42a458711eebaad1d0`
- tags at `HEAD`: none
- ahead/behind `upstream/main`: `423 0`
- `git status --short --branch`:

```text
## frank-v4-025-rollback-pointer-switch-skeleton
```

- baseline `go test -count=1 ./...`: passed

## Exact Files Planned

- `internal/missioncontrol/rollback_apply_registry.go`
- `internal/missioncontrol/rollback_apply_registry_test.go`
- `internal/agent/tools/taskstate.go`
- `internal/agent/loop.go`
- `internal/agent/loop_processdirect_test.go`
- `docs/maintenance/V4_025_ROLLBACK_POINTER_SWITCH_BEFORE.md`
- `docs/maintenance/V4_025_ROLLBACK_POINTER_SWITCH_AFTER.md`

## Exact State Transitions Planned

- rollback-apply workflow phase:
  - `ready_to_apply -> pointer_switched_reload_pending`
- active runtime-pack pointer on successful first execution:
  - `active_pack_id: <rollback.from_pack_id> -> <rollback.target_pack_id>`
  - `previous_active_pack_id: <current active> -> <prior active pack>`
  - `reload_generation: n -> n+1`
  - `update_record_ref: <previous ref> -> rollback_apply:<apply_id>`
- exact replay after `pointer_switched_reload_pending`:
  - no second pointer mutation
  - no second `reload_generation` increment
  - no last-known-good mutation

## Exact Non-Goals

- no reload or apply mechanics
- no last-known-good pointer mutation
- no broader rollback workflow semantics beyond `pointer_switched_reload_pending`
- no promotion behavior changes
- no evaluator execution
- no scoring behavior changes
- no autonomy changes
- no provider or channel behavior changes
- no cleanup outside this slice
- no dependency changes
- no commit
