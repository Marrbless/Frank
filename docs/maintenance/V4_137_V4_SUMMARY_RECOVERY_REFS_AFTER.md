# V4-137 V4 Summary Recovery Refs - After

## Live starting point

- Previous slice: V4-136 V4 Summary Help / Runbook Drift Coverage
- Working branch: `frank-v4-137-v4-summary-recovery-refs`
- Starting commit: `9d6f1cf551634999724d7d9d3db0f167834e71a3`
- Starting tag: `frank-v4-136-v4-summary-help-drift-coverage`
- Startup worktree: clean

## Gap closed

`v4_summary` now includes compact recovery refs:

- `selected_rollback_id`
- `selected_rollback_apply_id`

These fields are derived from the existing detailed rollback and rollback-apply identity sections. They do not create, mutate, repair, hide, or filter any records.

## Files changed

- `internal/missioncontrol/status.go`
- `internal/missioncontrol/status_v4_summary_test.go`
- `docs/HOT_UPDATE_OPERATOR_RUNBOOK.md`
- `docs/maintenance/V4_137_V4_SUMMARY_RECOVERY_REFS_BEFORE.md`
- `docs/maintenance/V4_137_V4_SUMMARY_RECOVERY_REFS_AFTER.md`

## Tests added

Added `TestOperatorV4SummarySurfacesRecoveryRefs`.

The test proves:

- rollback-apply state remains the compact state when rollback and rollback-apply records are both present
- `selected_rollback_id` is surfaced
- `selected_rollback_apply_id` is surfaced
- existing `has_rollback` and `has_rollback_apply` booleans remain true

## Runbook update

The generic recovery command index now tells operators to inspect:

- `v4_summary.state`
- `v4_summary.selected_rollback_id`
- `v4_summary.selected_rollback_apply_id`
- detailed rollback identity sections as audit authority

## Validation run

Planned final validation:

```text
/usr/local/go/bin/gofmt -w internal/missioncontrol/status.go internal/missioncontrol/status_v4_summary_test.go
git diff --check
/usr/local/go/bin/go test -count=1 ./internal/agent/tools
/usr/local/go/bin/go test -count=1 ./internal/agent
/usr/local/go/bin/go test -count=1 ./internal/missioncontrol
/usr/local/go/bin/go test -count=1 ./cmd/picobot
/usr/local/go/bin/go test -count=1 ./...
```

## Invariants preserved

- No runtime behavior changed.
- No rollback or rollback-apply lifecycle behavior changed.
- No hot-update gate, outcome, promotion, LKG, pointer-switch, or reload/apply behavior changed.
- No parser behavior or command syntax changed.
- No detailed identity section was removed or replaced.
- No canary-gate widening occurred.
- No natural-language owner approval binding was added.
- No dependencies were added.
