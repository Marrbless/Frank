# V4-136 V4 Summary Help / Runbook Drift Coverage - Before

## Live starting point

- Previous slice: V4-135 Status V4 Key Summary
- Working branch: `frank-v4-136-v4-completion-gap-rescan-stop-or-polish`
- Starting commit: `d8898a876d36b65dd01b71125bfc8681164b7143`
- Starting tag: `frank-v4-135-status-v4-key-summary`
- Startup worktree: clean
- Startup validation passed: `/usr/local/go/bin/go test -count=1 ./...`

## Gap found

V4-135 added the compact read-only `v4_summary` and included it in active `STATUS <job_id>` and committed mission status snapshots. The operator runbook now tells operators to inspect `v4_summary`, and eligible/canary runbook fixtures assert summary signals.

The static `HELP V4` / `HELP HOT_UPDATE` text still only named `STATUS <job_id>` and did not mention `v4_summary` or the rule that detailed identity sections remain the audit authority. The help/runbook drift test also did not lock that status-usability cross-link.

## Why this is a V4 completion gap

The summary was added specifically to make completed V4 lifecycle state usable from the operator channel. If the in-band help surface does not point operators from `STATUS <job_id>` to `v4_summary`, the new status summary can drift into an under-discoverable JSON field even though it is present and tested.

## Files inspected

- `docs/FRANK_V4_SPEC.md`
- `docs/HOT_UPDATE_OPERATOR_RUNBOOK.md`
- `docs/maintenance/V4_130_V4_END_TO_END_LIFECYCLE_HANDOFF_CHECKPOINT.md`
- `docs/maintenance/V4_131_OPERATOR_DIRECT_COMMAND_HELP_SURFACE_AFTER.md`
- `docs/maintenance/V4_132_RUNBOOK_DRIFT_COMMAND_HELP_CONSISTENCY_AFTER.md`
- `docs/maintenance/V4_133_ELIGIBLE_ONLY_RUNBOOK_FIXTURE_AFTER.md`
- `docs/maintenance/V4_134_ACTIVE_STATUS_COMMITTED_SNAPSHOT_PARITY_AFTER.md`
- `docs/maintenance/V4_135_STATUS_V4_KEY_SUMMARY_AFTER.md`
- `internal/agent/tools/taskstate_readout.go`
- `internal/agent/tools/taskstate_status_test.go`
- `internal/missioncontrol/status.go`
- `internal/missioncontrol/status_v4_summary_test.go`
- `internal/missioncontrol/store_project.go`
- `internal/agent/loop.go`
- `internal/agent/loop_processdirect_help_test.go`
- `internal/agent/loop_processdirect_help_consistency_test.go`
- `internal/agent/loop_processdirect_eligible_runbook_test.go`
- `internal/agent/loop_processdirect_canary_runbook_test.go`

## Implementation scope

Planned smallest closure:

- add a `v4_summary` pointer under static help's `STATUS <job_id>` section
- mirror the same guidance in the operator runbook
- add help/runbook drift assertions so future edits keep the cross-link

No runtime behavior, parser behavior, STATUS JSON shape, status loader, hot-update lifecycle, rollback, rollback-apply, LKG, pointer-switch, or reload/apply behavior should change.

## Tests added/updated

Planned:

- update `TestProcessDirectHotUpdateHelpListsV4LifecycleCommands`
- update `TestProcessDirectHotUpdateHelpRunbookAndParserStayConsistent`

## Validation run

Planned validation:

```text
/usr/local/go/bin/gofmt -w <changed Go files>
git diff --check
/usr/local/go/bin/go test -count=1 ./internal/agent/tools
/usr/local/go/bin/go test -count=1 ./internal/agent
/usr/local/go/bin/go test -count=1 ./internal/missioncontrol
/usr/local/go/bin/go test -count=1 ./cmd/picobot
/usr/local/go/bin/go test -count=1 ./...
```

## Risks

- Static help remains hand-maintained; this slice only adds drift coverage for the new summary guidance.
- Help text is intentionally concise and does not replace the detailed runbook checklist.

## Explicit non-goals

- No canary-gate widening.
- No natural-language owner approval binding.
- No broadening of `CandidatePromotionDecisionRecord`.
- No change to `CreateHotUpdateGateFromCandidatePromotionDecision(...)`.
- No outcome-kind changes.
- No automatic canary telemetry.
- No rollback, rollback-apply, LKG, pointer-switch, reload/apply, or runtime behavior changes.
- No detailed STATUS identity removal or replacement.
- No new dependencies.
- No V4-137 work.

## Recommended next action

After this slice, run a fresh validation suite. If no further concrete status/help/test gap appears, pause V4 widening rather than inventing broader policy work.
