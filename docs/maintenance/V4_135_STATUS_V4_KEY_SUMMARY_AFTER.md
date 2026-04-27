# V4-135 Status V4 Key Summary - After

## Live starting point

- Branch: `frank-v4-135-v4-completion-gap-rescan-status-summary`
- Starting commit: `2b80da293985fb68951c4f16f4bb8abdf4513415`
- Starting tag: `frank-v4-134-active-status-committed-snapshot-parity`
- Startup validation passed: `/usr/local/go/bin/go test -count=1 ./...`

## Gap found

The detailed V4 status/read-model surfaces were complete enough after V4-134, but operator usability still required reading many detailed identity sections to determine current V4 lifecycle state. There was no compact additive `v4_summary` field in `OperatorStatusSummary`.

## Why this is a V4 completion gap

V4 requires read-only operational awareness for hot-update, rollback, rollback-apply, last-known-good, and identity lineage. Detailed identity surfaces preserve audit depth, but a compact key summary closes the status-completeness gap for the common operator question: "where is this V4 lifecycle now?"

## Files inspected

- `docs/FRANK_V4_SPEC.md`
- `docs/HOT_UPDATE_OPERATOR_RUNBOOK.md`
- `docs/maintenance/V4_130_V4_END_TO_END_LIFECYCLE_HANDOFF_CHECKPOINT.md`
- `docs/maintenance/V4_131_OPERATOR_DIRECT_COMMAND_HELP_SURFACE_AFTER.md`
- `docs/maintenance/V4_132_RUNBOOK_DRIFT_COMMAND_HELP_CONSISTENCY_AFTER.md`
- `docs/maintenance/V4_133_ELIGIBLE_ONLY_RUNBOOK_FIXTURE_AFTER.md`
- `docs/maintenance/V4_134_ACTIVE_STATUS_COMMITTED_SNAPSHOT_PARITY_AFTER.md`
- `internal/missioncontrol/status.go`
- `internal/missioncontrol/store_project.go`
- `internal/agent/tools/taskstate_readout.go`
- `internal/agent/tools/taskstate_status_test.go`
- `internal/agent/loop.go`
- `internal/agent/loop_processdirect_help_test.go`
- `internal/agent/loop_processdirect_help_consistency_test.go`
- `internal/agent/loop_processdirect_eligible_runbook_test.go`
- `internal/agent/loop_processdirect_canary_runbook_test.go`

## Implementation scope

- Added `OperatorStatusSummary.v4_summary`.
- Added `OperatorV4SummaryStatus`.
- Added read-only derivation through `BuildOperatorV4SummaryStatus` and `WithV4Summary`.
- Included `v4_summary` in active `STATUS <job_id>`.
- Included `v4_summary` in committed mission status snapshots.
- Updated the hot-update runbook status checklist to mention the compact summary.

The summary is derived only from the detailed status identities already attached to `OperatorStatusSummary`. It does not create, repair, filter, delete, or mutate records.

## Tests added/updated

- Added `internal/missioncontrol/status_v4_summary_test.go`.
- Updated active/committed status parity coverage in `internal/agent/tools/taskstate_status_test.go`.
- Updated active STATUS JSON key assertions for the additive `v4_summary`.
- Updated eligible-only runbook coverage to assert candidate decision, selected gate/outcome/promotion, active pack, LKG, and clean no-canary/no-rollback summary signals.
- Updated canary runbook coverage to assert selected gate/outcome/promotion, canary authority, owner-approval boolean behavior, and clean no-rollback summary signals.

## Validation run

- `/usr/local/go/bin/gofmt -w` on changed Go files: passed.
- `git diff --check`: passed.
- `/usr/local/go/bin/go test -count=1 ./internal/agent/tools`: passed.
- `/usr/local/go/bin/go test -count=1 ./internal/agent`: passed.
- `/usr/local/go/bin/go test -count=1 ./internal/missioncontrol`: passed.
- `/usr/local/go/bin/go test -count=1 ./cmd/picobot`: passed.
- `/usr/local/go/bin/go test -count=1 ./...`: passed.

## Risks

- `v4_summary.state` is intentionally a compact status classification, not a new policy engine.
- The selected id fields are deterministic projections from the already-sorted detailed status read models, not a new timestamp-based "latest" selector.
- Detailed identity sections remain the audit authority for full lineage and invalid-record details.

## Explicit non-goals

- No canary-gate widening.
- No natural-language owner approval binding.
- No broadening of `CandidatePromotionDecisionRecord`.
- No change to `CreateHotUpdateGateFromCandidatePromotionDecision(...)`.
- No outcome-kind changes.
- No automatic canary telemetry.
- No rollback, rollback-apply, LKG, pointer-switch, or reload/apply behavior changes.
- No detailed STATUS identity removal or replacement.
- No new dependencies.
- No V4-136 work.

## Recommended next action

Run a fresh V4 completion re-scan. If no higher-priority concrete gap appears, prefer a small targeted status/help/test slice or stop with a no-op completion report rather than inventing broader V4 policy work.
