# V4-131 Operator Direct Command Help Surface - After

## Live Starting Point

- Branch at start: `frank-v4-130-v4-end-to-end-lifecycle-handoff-checkpoint`
- Working branch: `frank-v4-131-v4-completion-gap-closure`
- HEAD: `1cf170c025a0ef08e8f91d83d8f7b5b9286d6d99`
- Tag at HEAD: `frank-v4-130-v4-end-to-end-lifecycle-handoff-checkpoint`
- Startup worktree: clean
- Startup validator: `/usr/local/go/bin/go test -count=1 ./...` passed

## Gap Closed

V4 now has an in-band, read-only hot-update command help surface:

```text
HOT_UPDATE_HELP
HELP HOT_UPDATE
HELP V4
```

The help response lists the completed V4 hot-update command groups and points operators to `docs/HOT_UPDATE_OPERATOR_RUNBOOK.md`.

## Why This Is A V4 Completion Gap

The V4 lifecycle is command-rich. V4-130 identified command discoverability as a top residual operator risk after lifecycle safety, canary authority, audit lineage, runbook, and E2E canary fixture work were complete. A static direct-command help surface closes the smallest concrete operator-completeness gap without changing lifecycle behavior.

## Files Changed

- `internal/agent/loop.go`
- `internal/agent/loop_processdirect_help_test.go`
- `docs/HOT_UPDATE_OPERATOR_RUNBOOK.md`
- `docs/maintenance/V4_131_OPERATOR_DIRECT_COMMAND_HELP_SURFACE_BEFORE.md`
- `docs/maintenance/V4_131_OPERATOR_DIRECT_COMMAND_HELP_SURFACE_AFTER.md`

## Implementation Scope

`ProcessDirect` now recognizes static hot-update help commands before mission state, TaskState command handling, natural-language approval handling, or provider execution.

The help output includes:

- runbook pointer
- `STATUS <job_id>`
- eligible-only hot-update command group
- canary-required hot-update command group
- rollback, rollback-apply, and LKG recovery command group
- invariants that `CandidatePromotionDecisionRecord` remains eligible-only, canary owner approval is exact durable `granted`/`rejected`, canary-derived gates are guarded, canary audit lineage is preserved, and rollback/LKG remain generic

## Tests Added

`TestProcessDirectHotUpdateHelpListsV4LifecycleCommands` proves:

- `HELP HOT_UPDATE` returns static V4 help
- `HOT_UPDATE_HELP` returns the same response
- representative eligible-only, canary-required, rollback-apply, outcome/promotion, LKG, and invariant text is present
- the provider is not called for the static help response

## Validation Run

Validation for this slice is:

```text
/usr/local/go/bin/gofmt -w internal/agent/loop.go internal/agent/loop_processdirect_help_test.go
git diff --check
/usr/local/go/bin/go test -count=1 ./internal/agent
/usr/local/go/bin/go test -count=1 ./internal/missioncontrol
/usr/local/go/bin/go test -count=1 ./cmd/picobot
/usr/local/go/bin/go test -count=1 ./internal/agent/tools
/usr/local/go/bin/go test -count=1 ./...
```

## Invariants Preserved

- Existing command syntax is unchanged.
- Existing command semantics are unchanged.
- No hot-update, outcome, promotion, rollback, rollback-apply, LKG, pointer-switch, reload/apply, or runtime behavior changed.
- No records are created by help.
- No TaskState wrappers were added.
- No missioncontrol records or schemas changed.
- No canary-gate widening occurred.
- Natural-language owner approval remains unbound from canary owner approval.
- `CandidatePromotionDecisionRecord` remains eligible-only.

## Risks

- The static help text must be maintained with future direct-command additions.
- This improves command discoverability but does not shorten the operator workflow or add an automated workflow engine.

## Recommended Next Action

Pause further canary-gate widening. If continuing V4 completion, prefer a similarly narrow non-canary completion gap such as a runbook drift test that checks documented V4 command names against the direct-command parser.
