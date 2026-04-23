# V4-032 Rollback Terminal-Failure After

## git diff --stat

```text
 internal/agent/loop.go                             |  16 ++
 internal/agent/loop_processdirect_test.go          | 183 +++++++++++++++++++++
 internal/agent/tools/taskstate.go                  |  89 ++++++++++
 internal/missioncontrol/rollback_apply_registry.go |  76 +++++++++
 .../missioncontrol/rollback_apply_registry_test.go | 163 ++++++++++++++++++
 5 files changed, 527 insertions(+)
```

## git diff --numstat

```text
16	0	internal/agent/loop.go
183	0	internal/agent/loop_processdirect_test.go
89	0	internal/agent/tools/taskstate.go
76	0	internal/missioncontrol/rollback_apply_registry.go
163	0	internal/missioncontrol/rollback_apply_registry_test.go
```

## Files changed

- `internal/missioncontrol/rollback_apply_registry.go`
- `internal/missioncontrol/rollback_apply_registry_test.go`
- `internal/agent/tools/taskstate.go`
- `internal/agent/loop.go`
- `internal/agent/loop_processdirect_test.go`
- `docs/maintenance/V4_032_ROLLBACK_TERMINAL_FAILURE_BEFORE.md`
- `docs/maintenance/V4_032_ROLLBACK_TERMINAL_FAILURE_AFTER.md`

## Exact helpers/state transitions added

- Added `ResolveRollbackApplyTerminalFailure(root, applyID, reason, updatedBy, updatedAt)` in `internal/missioncontrol`.
- Added deterministic operator failure detail formatting through:
  - `operator_terminal_failure: <reason>`
- Added one new explicit operator-driven transition:
  - `reload_apply_recovery_needed -> reload_apply_failed`
- Allowed exact replay of the same terminal-failure decision to return idempotently when the record is already in `reload_apply_failed` with the same deterministic `execution_error`.
- Added the matching taskstate wrapper:
  - `ResolveRollbackApplyTerminalFailure(jobID, applyID, reason)`
- Added the matching direct-command path:
  - `ROLLBACK_APPLY_FAIL <job_id> <apply_id> <reason...>`

## Exact tests added

- `TestResolveRollbackApplyTerminalFailureFromRecoveryNeededPreservesPointerState`
- `TestResolveRollbackApplyTerminalFailureRequiresReasonAndReplayIsIdempotent`
- `TestResolveRollbackApplyTerminalFailureRejectsInvalidStartingPhase`
- `TestProcessDirectRollbackApplyFailCommandResolvesRecoveryNeededAndPreservesStatus`
- `TestProcessDirectRollbackApplyFailCommandRequiresReasonAndRejectsInvalidStartingPhase`

## Validation commands and results

- `gofmt -w internal/missioncontrol/rollback_apply_registry.go internal/missioncontrol/rollback_apply_registry_test.go internal/agent/tools/taskstate.go internal/agent/loop.go internal/agent/loop_processdirect_test.go` : passed
- `git diff --check` : passed
- `go test -count=1 ./internal/agent` : passed
- `go test -count=1 ./internal/agent/tools` : passed
- `go test -count=1 ./internal/missioncontrol` : passed
- `go test -count=1 ./...` : passed
- `git status --short --branch` : refreshed after report write below

## Explicit non-implementation statements

- No pointer mutation was implemented.
- No `reload_generation` increment was implemented.
- No `last_known_good_pointer.json` mutation was implemented.

## Deferred next V4 candidates

- expose deterministic terminal failure detail through read-only status if later checkpoints need operator visibility into `execution_error`
- consider whether later slices need explicit retry-from-terminal-failure policy, which remains out of scope now
- any broader workflow resolution semantics beyond the single operator-driven terminal-failure action
