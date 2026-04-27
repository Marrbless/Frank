# V4-135 Status V4 Key Summary - Before

## Live starting point

- Branch: `frank-v4-135-v4-completion-gap-rescan-status-summary`
- Starting commit: `2b80da293985fb68951c4f16f4bb8abdf4513415`
- Starting tag: `frank-v4-134-active-status-committed-snapshot-parity`
- Startup validation: `/usr/local/go/bin/go test -count=1 ./...` passed before edits.

## Gap found

Active `STATUS <job_id>` and committed mission status snapshots now carry the detailed V4 identity surfaces, including promotion policy and candidate promotion decision identity. The status surface is complete but noisy: an operator must inspect many detailed identity sections to answer the common V4 lifecycle questions.

There is no compact additive `v4_summary` in `OperatorStatusSummary`.

## Why this is a V4 completion gap

The frozen V4 spec requires durable, read-only operational awareness for hot-update, rollback, and runtime-pack state. The detailed read model satisfies audit depth, but the lack of a small key summary leaves the operator-complete status surface harder to use after the lifecycle is complete.

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

Planned scope is one additive read-only status slice:

- Add `v4_summary` to `OperatorStatusSummary`.
- Derive it only from existing status/read-model identity data already loaded into the summary.
- Include it in active `STATUS <job_id>` and committed mission status snapshots.
- Keep detailed identity fields unchanged.
- Update the runbook to tell operators where to inspect the compact summary.

## Tests added/updated

Planned coverage:

- Empty store summary is deterministic and `not_configured`.
- Invalid detailed identity status contributes to invalid state/count/warnings.
- Eligible-only runbook final status includes candidate decision, gate, outcome, promotion, active pack, and LKG summary signals.
- Canary runbook final status includes canary authority and owner approval lineage summary signals.
- Active STATUS and committed snapshot carry matching `v4_summary`.

## Validation run

To run after implementation:

- `/usr/local/go/bin/gofmt -w` on changed Go files
- `git diff --check`
- `/usr/local/go/bin/go test -count=1 ./internal/agent/tools`
- `/usr/local/go/bin/go test -count=1 ./internal/agent`
- `/usr/local/go/bin/go test -count=1 ./internal/missioncontrol`
- `/usr/local/go/bin/go test -count=1 ./cmd/picobot`
- `/usr/local/go/bin/go test -count=1 ./...`

## Risks

- Summary state names must not imply new V4 policy.
- The summary must not hide invalid records or replace detailed identity surfaces.
- The summary must remain read-only and must not introduce extra status scans beyond existing composition.

## Explicit non-goals

- No canary-gate widening.
- No natural-language owner approval binding.
- No broadening of `CandidatePromotionDecisionRecord`.
- No change to `CreateHotUpdateGateFromCandidatePromotionDecision(...)`.
- No outcome-kind changes.
- No automatic canary telemetry.
- No rollback, rollback-apply, LKG, pointer-switch, or reload/apply behavior changes.
- No new dependencies.
- No V4-136 work.

## Recommended next action

Implement the compact summary as an additive read-only field, validate it against active and committed status paths, then re-scan for remaining concrete V4 completion gaps.
