# V4-138 V4 Summary Recovery Status Coverage - After

## Gap closed

Direct-command recovery status tests now assert the compact `v4_summary` recovery refs introduced in V4-137.

## Files changed

- `internal/agent/loop_processdirect_test.go`
- `docs/maintenance/V4_138_V4_SUMMARY_RECOVERY_STATUS_COVERAGE_BEFORE.md`
- `docs/maintenance/V4_138_V4_SUMMARY_RECOVERY_STATUS_COVERAGE_AFTER.md`

## Test coverage added

Updated `TestProcessDirectRollbackApplyRecordCommandCreatesOrSelectsWorkflowAndPreservesActiveRuntimePackPointer` to assert that `STATUS <job_id>` returns:

- `v4_summary.state = rollback_apply_recorded`
- `v4_summary.selected_rollback_id = rollback-1`
- `v4_summary.selected_rollback_apply_id = apply-1`
- `has_rollback = true`
- `has_rollback_apply = true`

Updated `TestProcessDirectRollbackApplyPhaseCommandAdvancesWorkflowAndPreservesActiveRuntimePackPointer` to assert the same selected recovery refs after phase advancement.

## Validation run

Planned final validation:

```text
/usr/local/go/bin/gofmt -w internal/agent/loop_processdirect_test.go
git diff --check
/usr/local/go/bin/go test -count=1 ./internal/agent/tools
/usr/local/go/bin/go test -count=1 ./internal/agent
/usr/local/go/bin/go test -count=1 ./internal/missioncontrol
/usr/local/go/bin/go test -count=1 ./cmd/picobot
/usr/local/go/bin/go test -count=1 ./...
```

## Invariants preserved

- Production code is unchanged.
- Status schema is unchanged.
- Command syntax is unchanged.
- Rollback, rollback-apply, LKG, pointer-switch, reload/apply, outcome, and promotion behavior are unchanged.
- No canary-gate behavior changed.
