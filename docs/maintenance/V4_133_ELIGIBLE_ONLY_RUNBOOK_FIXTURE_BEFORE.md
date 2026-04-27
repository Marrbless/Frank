# V4-133 Eligible-Only Runbook Fixture - Before

## Live Starting Point

- Previous slice: V4-132 Runbook Drift / Command Help Consistency
- Expected branch: `frank-v4-133-v4-completion-gap-rescan-closure`
- Expected prior commit: `7e6b8f0 test: add V4 runbook command help drift coverage`
- Expected prior tag: `frank-v4-132-runbook-drift-command-help-consistency`

## Gap Found

V4 canary-required hot-update flow has a bounded direct-command runbook fixture that proves the documented operator sequence works through promotion audit lineage.

The eligible-only hot-update path has focused direct-command and registry coverage, plus help/runbook drift coverage, but no single bounded runbook-style direct-command fixture that walks the documented eligible-only sequence as an operator would use it.

## Why This Is A V4 Completion Gap

The eligible-only path remains the baseline hot-update lifecycle:

candidate promotion decision -> prepared hot-update gate -> execution readiness -> phase advancement -> pointer switch -> reload/apply -> terminal outcome -> promotion -> optional LKG recertification.

Fragmented unit coverage proves individual commands, but it does not catch command-order drift, status composition drift, or accidental side effects across the full eligible-only operator path.

## Files Inspected

- `docs/FRANK_V4_SPEC.md`
- `docs/HOT_UPDATE_OPERATOR_RUNBOOK.md`
- `docs/maintenance/V4_130_V4_END_TO_END_LIFECYCLE_HANDOFF_CHECKPOINT.md`
- `docs/maintenance/V4_131_OPERATOR_DIRECT_COMMAND_HELP_SURFACE_AFTER.md`
- `docs/maintenance/V4_132_RUNBOOK_DRIFT_COMMAND_HELP_CONSISTENCY_AFTER.md`
- `internal/agent/loop.go`
- `internal/agent/loop_processdirect_test.go`
- `internal/agent/loop_processdirect_canary_runbook_test.go`
- `internal/agent/loop_processdirect_help_test.go`
- `internal/agent/loop_processdirect_help_consistency_test.go`
- `internal/missioncontrol/status.go`

## Planned Scope

Add one deterministic direct-command fixture for the eligible-only operator path. The fixture should:

- start from an existing eligible `CandidatePromotionDecisionRecord`
- create the gate with `HOT_UPDATE_GATE_FROM_DECISION`
- advance through validated and staged phases
- record execution readiness with `HOT_UPDATE_EXECUTION_READY`
- execute pointer switch and reload/apply
- create outcome and promotion
- recertify LKG from the promotion
- assert the candidate-promotion-decision ledger and final `STATUS <job_id>` downstream identity surfaces
- assert no canary authority, rollback, rollback-apply, or runtime approval records are created

## Explicit Non-Goals

- No production code changes.
- No command syntax changes.
- No new commands.
- No TaskState wrapper changes.
- No canary-gate widening.
- No natural-language owner approval binding.
- No `CandidatePromotionDecisionRecord` broadening.
- No rollback or rollback-apply execution.
- No V4-134 work.
