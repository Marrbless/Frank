# V4-132 Runbook Drift / Command Help Consistency - Before

## Live Starting Point

- Branch at start: `frank-v4-131-v4-completion-gap-closure`
- Working branch: `frank-v4-132-runbook-drift-command-help-consistency`
- HEAD: `460840bd34c3443a05601e23396f1c48b21b8b0f`
- Tag at HEAD: `frank-v4-131-operator-direct-command-help-surface`
- Startup worktree: clean
- Startup validator: `/usr/local/go/bin/go test -count=1 ./...` passed

## Gap Found

V4-131 added a static in-band hot-update help surface. Static help creates a drift risk: future parser changes can add, remove, or rename command surfaces without matching updates to `HOT_UPDATE_HELP` or `docs/HOT_UPDATE_OPERATOR_RUNBOOK.md`.

Live inspection found two small existing drift items:

- the parser accepted `HOT_UPDATE_HELP`, `HELP HOT_UPDATE`, and `HELP V4`, and the runbook documented them, but the static help output did not list those aliases
- `HOT_UPDATE_GATE_FROM_DECISION` was implemented and listed in static help, but the eligible-only runbook sequence still documented only `HOT_UPDATE_GATE_RECORD`
- `HOT_UPDATE_EXECUTION_READY` and the generic rollback/rollback-apply direct-command names were implemented and listed in static help, but the runbook did not name them as direct commands

## Why This Is A V4 Completion Gap

The V4 hot-update operator surface is complete but command-rich. Once command discovery moved in-band, V4 needs executable coverage that keeps three surfaces aligned:

- live direct-command parser regexes in `internal/agent/loop.go`
- static help output from `HOT_UPDATE_HELP`
- `docs/HOT_UPDATE_OPERATOR_RUNBOOK.md`

Without that coverage, command discoverability can silently regress even when lifecycle behavior remains correct.

## Files Inspected

- `docs/FRANK_V4_SPEC.md`
- `docs/HOT_UPDATE_OPERATOR_RUNBOOK.md`
- `docs/maintenance/V4_130_V4_END_TO_END_LIFECYCLE_HANDOFF_CHECKPOINT.md`
- `docs/maintenance/V4_131_OPERATOR_DIRECT_COMMAND_HELP_SURFACE_AFTER.md`
- `internal/agent/loop.go`
- `internal/agent/loop_processdirect_help_test.go`
- `internal/agent/loop_processdirect_test.go`
- `internal/agent/loop_processdirect_canary_runbook_test.go`

## Implementation Scope

The intended scope is tests and docs, with only a tiny static help text correction for the discovered alias drift:

- add a focused same-package consistency test for parser regexes, static help, and runbook command names
- assert help aliases are static, equivalent, provider-free, and do not mutate a temp mission store
- assert key V4 invariants remain present in both help and runbook
- document the drift test and residual risk

## Tests Planned

- Add `internal/agent/loop_processdirect_help_consistency_test.go`.
- Use same-package access to direct-command regex variables for parser consistency.
- Avoid executing lifecycle commands or creating hot-update records.
- Read `docs/HOT_UPDATE_OPERATOR_RUNBOOK.md` directly.
- Use static help aliases only.

## Explicit Non-Goals

- No command syntax changes.
- No new direct commands.
- No TaskState wrapper changes.
- No missioncontrol behavior changes.
- No runtime behavior changes.
- No hot-update record creation in the drift test.
- No gate execution, pointer switch, reload/apply, outcome, promotion, rollback, rollback-apply, or LKG execution.
- No canary-gate widening.
- No natural-language owner approval binding.
- No `CandidatePromotionDecisionRecord` broadening.
- No V4-133 implementation.

## Risks

- The canonical command list remains test-maintained; future command additions must update it intentionally.
- This prevents command-name and invariant drift, but does not validate every argument shape or lifecycle behavior.
