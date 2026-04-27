# V4-137 V4 Summary Recovery Refs - Before

## Live starting point

- Previous slice: V4-136 V4 Summary Help / Runbook Drift Coverage
- Working branch: `frank-v4-137-v4-summary-recovery-refs`
- Starting commit: `9d6f1cf551634999724d7d9d3db0f167834e71a3`
- Starting tag: `frank-v4-136-v4-summary-help-drift-coverage`
- Startup worktree: clean
- Startup validation passed in V4-136: `/usr/local/go/bin/go test -count=1 ./...`

## Gap found

V4-135 added compact `v4_summary` status and V4-136 exposed it through help/runbook drift coverage. The compact summary includes selected ids for the gate/outcome/promotion path, but recovery state only exposed booleans:

- `has_rollback`
- `has_rollback_apply`

When `v4_summary.state` is `rollback_recorded` or `rollback_apply_recorded`, operators still have to inspect detailed identity sections just to identify the selected rollback or rollback-apply record.

## Why this is a V4 completion gap

Rollback and rollback-apply are part of the completed V4 recovery lifecycle. The compact status summary should orient operators to the current recovery record ids the same way it already orients them to selected hot-update, outcome, and promotion ids.

Detailed `rollback_identity` and `rollback_apply_identity` remain the audit authority for linkage, phase, activation state, invalid records, and errors.

## Planned implementation scope

- Add `selected_rollback_id` to `OperatorV4SummaryStatus`.
- Add `selected_rollback_apply_id` to `OperatorV4SummaryStatus`.
- Derive both fields read-only from existing detailed rollback and rollback-apply identity sections.
- Add focused missioncontrol summary coverage.
- Update the operator runbook generic recovery status checklist.

## Explicit non-goals

- No runtime behavior changes.
- No rollback or rollback-apply lifecycle changes.
- No pointer-switch, reload/apply, LKG, outcome, promotion, or hot-update behavior changes.
- No parser or command syntax changes.
- No record schema changes outside the additive status JSON fields.
- No canary-gate widening.
- No natural-language owner approval binding.
- No new dependencies.
