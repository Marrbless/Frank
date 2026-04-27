# V4-136 V4 Summary Help / Runbook Drift Coverage - After

## Live starting point

- Previous slice: V4-135 Status V4 Key Summary
- Working branch: `frank-v4-136-v4-completion-gap-rescan-stop-or-polish`
- Starting commit: `d8898a876d36b65dd01b71125bfc8681164b7143`
- Starting tag: `frank-v4-135-status-v4-key-summary`
- Startup worktree: clean
- Startup validation passed: `/usr/local/go/bin/go test -count=1 ./...`

## Gap found

V4-135 added `v4_summary` to active `STATUS <job_id>` and committed mission status snapshots, and the operator runbook already referenced it in status checklists. The static `HELP V4` / `HELP HOT_UPDATE` output still did not mention `v4_summary`, so the in-band operator help did not expose the new compact status surface.

## Why this is a V4 completion gap

`v4_summary` was added as a status-usability surface for operators. Without static help and drift coverage, operators could run the documented help command and miss the compact summary that answers the common "where is the V4 lifecycle now?" question.

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

- Added `v4_summary` guidance under static help's `STATUS <job_id>` section.
- Added a runbook note that `v4_summary` is the compact orientation surface and detailed identity sections remain audit authority.
- Added static help and help/runbook drift assertions for the `v4_summary` guidance.
- Stabilized the active STATUS / committed snapshot parity fixture by using the store-test pattern of a current UTC timestamp for the projected runtime commit, because the store batch validates writer-lock leases against live time.

No parser, command, status JSON, status loader, hot-update lifecycle, rollback, rollback-apply, LKG, pointer-switch, reload/apply, or runtime behavior changed.

## Tests added/updated

Updated:

- `internal/agent/loop_processdirect_help_test.go`
- `internal/agent/loop_processdirect_help_consistency_test.go`
- `internal/agent/tools/taskstate_status_test.go`

The tests now assert:

- `HELP HOT_UPDATE` includes the `v4_summary` status guidance.
- static help and the runbook both preserve the compact summary / detailed audit authority relationship.
- the status parity fixture no longer expires its projected writer lease before the committed snapshot can be built.

## Validation run

Validation for this slice passed:

```text
/usr/local/go/bin/gofmt -w internal/agent/loop.go internal/agent/loop_processdirect_help_test.go internal/agent/loop_processdirect_help_consistency_test.go
/usr/local/go/bin/go test -count=1 ./internal/agent/tools
/usr/local/go/bin/go test -count=1 ./internal/agent
/usr/local/go/bin/go test -count=1 ./internal/missioncontrol
/usr/local/go/bin/go test -count=1 ./cmd/picobot
git diff --check
/usr/local/go/bin/go test -count=1 ./...
```

## Risks

- Static help remains manually maintained; the drift test now covers this summary cross-link but does not generate help from parser or schema metadata.
- This is a discoverability polish slice only. It does not add any new lifecycle behavior.

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

If validation remains green, stop widening V4 unless a new live, concrete status/help/test/safety gap is found. The current V4 completion surface has status parity, a compact summary, runbook fixtures for eligible and canary paths, and static help/runbook drift coverage for the summary.
