# V4-134 Active STATUS / Committed Snapshot Parity - Before

## Live Starting Point

- Working branch: `frank-v4-134-v4-completion-gap-rescan-status-usability`
- HEAD at start: `f4719ee36143ef999948bd2c367deba0e401304e`
- Tag at HEAD: `frank-v4-133-eligible-only-runbook-fixture`
- Startup worktree: clean
- Startup validator: `/usr/local/go/bin/go test -count=1 ./...` passed

## Gap Found

Live inspection found a concrete STATUS parity gap: `BuildCommittedMissionStatusSnapshot(...)` attaches `promotion_policy_identity` and `candidate_promotion_decision_identity`, but active `STATUS <job_id>` through `TaskState.OperatorStatus(...)` does not.

## Why This Is A V4 Completion Gap

V4 operators need active `STATUS <job_id>` to show the same completed hot-update identity surfaces available in committed mission status snapshots. The eligible-only hot-update path depends on promotion policy and candidate-promotion-decision records; hiding those records from active STATUS makes the read model incomplete during live operation even though the records and committed snapshot projection already exist.

## Files Inspected

- `docs/FRANK_V4_SPEC.md`
- `docs/HOT_UPDATE_OPERATOR_RUNBOOK.md`
- `docs/maintenance/V4_130_V4_END_TO_END_LIFECYCLE_HANDOFF_CHECKPOINT.md`
- `docs/maintenance/V4_131_OPERATOR_DIRECT_COMMAND_HELP_SURFACE_AFTER.md`
- `docs/maintenance/V4_132_RUNBOOK_DRIFT_COMMAND_HELP_CONSISTENCY_AFTER.md`
- `docs/maintenance/V4_133_ELIGIBLE_ONLY_RUNBOOK_FIXTURE_AFTER.md`
- `internal/agent/loop.go`
- `internal/agent/tools/taskstate_readout.go`
- `internal/agent/tools/taskstate_status_test.go`
- `internal/missioncontrol/status.go`
- `internal/missioncontrol/store_project.go`
- `internal/missioncontrol/status_promotion_policy_identity_test.go`
- `internal/missioncontrol/status_candidate_promotion_decision_identity_test.go`
- `internal/agent/loop_processdirect_eligible_runbook_test.go`
- `internal/agent/loop_processdirect_canary_runbook_test.go`

## Implementation Scope

Planned scope is one read-only status composition fix:

- add `promotion_policy_identity` to active `STATUS <job_id>`
- add `candidate_promotion_decision_identity` to active `STATUS <job_id>`
- add focused tests proving active STATUS and committed snapshots expose those same V4 surfaces
- update the operator runbook eligible-only checklist to mention those status sections

## Tests Added Or Updated

Planned:

- add active STATUS parity coverage in `internal/agent/tools/taskstate_status_test.go`

## Validation Run

Planned:

```text
git diff --check
/usr/local/go/bin/go test -count=1 ./internal/agent
/usr/local/go/bin/go test -count=1 ./internal/missioncontrol
/usr/local/go/bin/go test -count=1 ./cmd/picobot
/usr/local/go/bin/go test -count=1 ./internal/agent/tools
/usr/local/go/bin/go test -count=1 ./...
```

## Risks

- Active STATUS JSON becomes additively larger.
- Invalid records will remain visible as invalid through the existing identity loaders.

## Explicit Non-Goals

- No hot-update behavior changes.
- No pointer-switch, reload/apply, rollback, rollback-apply, or LKG behavior changes.
- No canary-gate widening.
- No natural-language owner approval binding.
- No `CandidatePromotionDecisionRecord` schema broadening.
- No change to `CreateHotUpdateGateFromCandidatePromotionDecision(...)`.
- No new dependencies.
- No compact `v4_summary` field in this slice.

## Recommended Next Action

Implement the active STATUS parity fix and validate it with focused package tests plus the full test suite.
