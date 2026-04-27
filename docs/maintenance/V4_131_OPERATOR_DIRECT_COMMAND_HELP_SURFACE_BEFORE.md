# V4-131 Operator Direct Command Help Surface - Before

## Live Starting Point

- Branch at start: `frank-v4-130-v4-end-to-end-lifecycle-handoff-checkpoint`
- Created working branch: `frank-v4-131-v4-completion-gap-closure`
- HEAD: `1cf170c025a0ef08e8f91d83d8f7b5b9286d6d99`
- Tag at HEAD: `frank-v4-130-v4-end-to-end-lifecycle-handoff-checkpoint`
- Startup worktree: clean
- Startup validator: `/usr/local/go/bin/go test -count=1 ./...` passed

## Gap Found

V4-130 ranked the long operator workflow and command discoverability as the top residual risks after canary-gate stop-widening. Live inspection confirmed that:

- `docs/HOT_UPDATE_OPERATOR_RUNBOOK.md` documents the complete V4 hot-update workflow.
- `internal/agent/loop.go` exposes many direct commands for eligible-only, canary-required, rollback, rollback-apply, outcome, promotion, and LKG flows.
- No central in-band direct-command help surface existed for the V4 hot-update command set.

## Why This Is A V4 Completion Gap

The implemented lifecycle is broad enough that operators must otherwise reconstruct command spelling and ordering from the runbook or maintenance memos. That is an operator-completeness gap, not a missing safety policy: the lifecycle behavior is complete, but the operator channel lacks a compact discoverability surface for the completed command groups.

## Files Inspected

- `docs/FRANK_V4_SPEC.md`
- `docs/HOT_UPDATE_OPERATOR_RUNBOOK.md`
- `docs/maintenance/V4_130_V4_END_TO_END_LIFECYCLE_HANDOFF_CHECKPOINT.md`
- `internal/agent/loop.go`
- `internal/agent/loop_processdirect_test.go`
- `internal/agent/loop_processdirect_canary_runbook_test.go`

## Implementation Scope

The intended scope is a read-only direct-command help response for V4 hot-update operators:

- add a static `HOT_UPDATE_HELP` / `HELP HOT_UPDATE` parser surface
- list completed eligible-only, canary-required, rollback/rollback-apply, outcome/promotion, and LKG command groups
- point operators to `docs/HOT_UPDATE_OPERATOR_RUNBOOK.md`
- add focused tests that the help surface is static and does not call the provider
- update the runbook to mention the help commands

## Tests Planned

- Add a focused `internal/agent` test proving `HELP HOT_UPDATE` and `HOT_UPDATE_HELP` return the same static help output.
- Assert representative V4 command groups and important invariants are present.
- Assert the provider is not called for the static help response.

## Explicit Non-Goals

- No hot-update lifecycle behavior change.
- No command syntax change for existing commands.
- No missioncontrol schema or validation change.
- No TaskState wrapper change.
- No canary-gate widening.
- No natural-language owner approval binding.
- No `CandidatePromotionDecisionRecord` broadening.
- No rollback, rollback-apply, LKG, pointer-switch, reload/apply, outcome, or promotion behavior change.
- No V4-132 implementation.

## Risks

- Help text can drift from future command parser changes unless later slices update it with the parser.
- This closes discoverability, not workflow length.
