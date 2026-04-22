# V4-027 Rollback Reload/Apply Before

- branch: `frank-v4-027-rollback-reload-apply-skeleton`
- `HEAD`: `0cb2e6210fca69f7177435061a3149a34024800a`
- tags at `HEAD`: none
- ahead/behind `upstream/main`: `425 0`
- `git status --short --branch`:

```text
## frank-v4-027-rollback-reload-apply-skeleton
```

- baseline `go test -count=1 ./...`: passed

## Exact Files Planned

- `internal/missioncontrol/rollback_apply_registry.go`
- `internal/missioncontrol/rollback_apply_registry_test.go`
- `internal/agent/tools/taskstate.go`
- `internal/agent/loop.go`
- `internal/agent/loop_processdirect_test.go`
- `docs/maintenance/V4_027_ROLLBACK_RELOAD_APPLY_BEFORE.md`
- `docs/maintenance/V4_027_ROLLBACK_RELOAD_APPLY_AFTER.md`

## Exact Phase Transitions Planned

- reload/apply execution:
  - `pointer_switched_reload_pending -> reload_apply_in_progress -> reload_apply_succeeded`
  - `pointer_switched_reload_pending -> reload_apply_in_progress -> reload_apply_failed`
- replay behavior:
  - `reload_apply_succeeded` is idempotent and does not redo side effects
  - `reload_apply_failed` fails closed
- strict preserved invariants:
  - no second active-pointer switch
  - no second `reload_generation` increment
  - no last-known-good mutation

## Exact Non-Goals

- no second pointer mutation
- no second `reload_generation` increment
- no `last_known_good_pointer.json` mutation
- no promotion behavior changes
- no evaluator execution
- no scoring behavior
- no autonomy changes
- no provider/channel changes outside the minimum bounded restart-style convergence path
- no cleanup outside this slice
- no dependency changes
- no commit
