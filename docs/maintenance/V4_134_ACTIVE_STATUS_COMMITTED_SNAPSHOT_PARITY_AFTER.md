# V4-134 Active STATUS / Committed Snapshot Parity - After

## Live Starting Point

- Working branch: `frank-v4-134-v4-completion-gap-rescan-status-usability`
- HEAD at start: `f4719ee36143ef999948bd2c367deba0e401304e`
- Tag at HEAD: `frank-v4-133-eligible-only-runbook-fixture`
- Startup worktree: clean
- Startup validator: `/usr/local/go/bin/go test -count=1 ./...` passed

## Gap Found

Active `STATUS <job_id>` through `TaskState.OperatorStatus(...)` omitted two V4 identity surfaces that committed mission status snapshots already exposed:

- `promotion_policy_identity`
- `candidate_promotion_decision_identity`

## Why This Is A V4 Completion Gap

The eligible-only V4 hot-update path starts from promotion policy evaluation and a durable candidate-promotion decision. Operators could inspect those records in committed snapshots, but live active STATUS did not show them. That made the live read model incomplete and forced operators to use lower-level ledger inspection during the active workflow.

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

Changed active STATUS composition only:

- `formatOperatorStatusReadoutWithDeferredSchedulerTriggers(...)` now attaches `promotion_policy_identity`
- `formatOperatorStatusReadoutWithDeferredSchedulerTriggers(...)` now attaches `candidate_promotion_decision_identity`
- existing detailed identity sections remain unchanged
- invalid records still surface through the existing status loaders
- no records are mutated, filtered, repaired, created, or deleted by STATUS

The operator runbook eligible-only status checklist now points operators to these two sections before inspecting the derived hot-update gate.

## Tests Added Or Updated

Updated:

- `internal/agent/tools/taskstate_status_test.go`

Added:

- `TestTaskStateOperatorStatusMatchesCommittedSnapshotForPromotionPolicyAndDecisionIdentity`

The test builds a deterministic eligible-only V4 fixture and proves active `TaskState.OperatorStatus(...)` matches `BuildCommittedMissionStatusSnapshot(...)` for:

- `promotion_policy_identity`
- `candidate_promotion_decision_identity`

## Validation Run

Validation run for this slice:

```text
/usr/local/go/bin/gofmt -w internal/agent/tools/taskstate_readout.go internal/agent/tools/taskstate_status_test.go  # passed
git diff --check  # passed
/usr/local/go/bin/go test -count=1 ./internal/agent/tools  # passed
/usr/local/go/bin/go test -count=1 ./internal/agent  # passed
/usr/local/go/bin/go test -count=1 ./internal/missioncontrol  # passed
/usr/local/go/bin/go test -count=1 ./cmd/picobot  # passed
/usr/local/go/bin/go test -count=1 ./...  # passed on rerun
```

One full-suite attempt before the passing rerun hit existing time-order failures in `internal/agent` hot-update phase tests (`phase_updated_at must not precede prepared_at` / `created_at`). The immediately preceding focused `internal/agent` run passed, and the subsequent full-suite rerun passed.

## Risks

- Active STATUS JSON is additively larger.
- This closes parity for two omitted V4 identity surfaces, but it does not add a compact `v4_summary`.

## Explicit Non-Goals

- No hot-update execution behavior changed.
- No pointer-switch, reload/apply, rollback, rollback-apply, or LKG behavior changed.
- No canary-gate widening.
- No natural-language owner approval binding.
- No `CandidatePromotionDecisionRecord` schema broadening.
- No change to `CreateHotUpdateGateFromCandidatePromotionDecision(...)`.
- No outcome kind changes.
- No automatic canary telemetry.
- No new dependencies.
- No V4-135 work.

## Recommended Next Action

Run a fresh V4 completion re-scan after this branch is reviewed. If no higher-priority concrete gap appears, the next status-usability slice can consider a compact additive `v4_summary` built from the now-parity-complete detailed identity surfaces.
