# V4-132 Runbook Drift / Command Help Consistency - After

## Live Starting Point

- Branch at start: `frank-v4-131-v4-completion-gap-closure`
- Working branch: `frank-v4-132-runbook-drift-command-help-consistency`
- HEAD: `460840bd34c3443a05601e23396f1c48b21b8b0f`
- Tag at HEAD: `frank-v4-131-operator-direct-command-help-surface`
- Startup worktree: clean
- Startup validator: `/usr/local/go/bin/go test -count=1 ./...` passed

## Gap Closed

V4 now has executable drift coverage tying together:

- live parser command regexes in `internal/agent/loop.go`
- static hot-update help output
- `docs/HOT_UPDATE_OPERATOR_RUNBOOK.md`

Two small drift items were corrected:

- static help now lists `HOT_UPDATE_HELP`, `HELP HOT_UPDATE`, and `HELP V4`
- the eligible-only runbook sequence now documents `HOT_UPDATE_GATE_FROM_DECISION <job_id> <promotion_decision_id>` before the still-supported `HOT_UPDATE_GATE_RECORD` path
- the runbook now names `HOT_UPDATE_EXECUTION_READY` and the generic rollback/rollback-apply direct commands in explicit command blocks

## Files Changed

- `internal/agent/loop.go`
- `internal/agent/loop_processdirect_help_consistency_test.go`
- `docs/HOT_UPDATE_OPERATOR_RUNBOOK.md`
- `docs/maintenance/V4_132_RUNBOOK_DRIFT_COMMAND_HELP_CONSISTENCY_BEFORE.md`
- `docs/maintenance/V4_132_RUNBOOK_DRIFT_COMMAND_HELP_CONSISTENCY_AFTER.md`

## Canonical Command List Strategy

The new test defines a canonical V4 direct-command list from live parser surfaces:

- `STATUS`
- `HOT_UPDATE_HELP`
- `HELP HOT_UPDATE`
- `HELP V4`
- eligible-only hot-update commands
- execution readiness, phase, execute, reload, fail, outcome, promotion, and LKG commands
- canary requirement/evidence/satisfaction-authority/owner-approval/canary-gate commands
- rollback and rollback-apply commands

Each command has a safe sample string matched against the same-package parser regex. The test does not execute lifecycle commands.

## Consistency Asserted

`TestProcessDirectHotUpdateHelpRunbookAndParserStayConsistent` asserts:

- each canonical command sample matches its live parser regex
- `HOT_UPDATE_HELP`, `HELP HOT_UPDATE`, and `HELP V4` all return equivalent static help
- help aliases do not call the provider
- help aliases do not create files in a temp mission store
- static help includes every canonical command name
- `docs/HOT_UPDATE_OPERATOR_RUNBOOK.md` includes every canonical command name
- help and runbook both preserve key invariants:
  - `CandidatePromotionDecisionRecord` remains eligible-only
  - canary owner approval uses exact `granted` / `rejected`
  - natural-language aliases are not canary owner approval authority
  - canary-derived gates are guarded
  - outcome/promotion preserve `canary_ref` / `approval_ref`
  - rollback, rollback-apply, and LKG remain generic

## Intentionally Not Asserted

- No lifecycle commands are executed.
- No hot-update, canary, owner approval, outcome, promotion, rollback, rollback-apply, or LKG records are created.
- The test does not validate every argument combination.
- The test does not replace the existing focused lifecycle and runbook fixtures.
- The test does not bind natural-language owner approval.
- The test does not broaden `CandidatePromotionDecisionRecord`.

## Validation Run

Validation for this slice is:

```text
/usr/local/go/bin/gofmt -w internal/agent/loop.go internal/agent/loop_processdirect_help_consistency_test.go
git diff --check
/usr/local/go/bin/go test -count=1 ./internal/agent
/usr/local/go/bin/go test -count=1 ./internal/missioncontrol
/usr/local/go/bin/go test -count=1 ./cmd/picobot
/usr/local/go/bin/go test -count=1 ./internal/agent/tools
/usr/local/go/bin/go test -count=1 ./...
```

## Invariants Preserved

- Existing command syntax is unchanged.
- No commands were added or removed.
- No parser behavior changed beyond static help text content.
- No TaskState wrappers changed.
- No missioncontrol behavior changed.
- No runtime behavior changed.
- No canary-gate widening occurred.
- Rollback, rollback-apply, LKG, pointer-switch, reload/apply, outcome, and promotion behavior are unchanged.
- Natural-language owner approval remains separate from canary owner approval authority.
- `CandidatePromotionDecisionRecord` remains eligible-only.

## Residual Drift Risk

The canonical command list is intentionally explicit. Future V4 command additions should update parser, help, runbook, and this test together. This catches omission drift, but it does not generate help from parser metadata.

## Recommended Next Action

Pause canary-gate widening. If continuing V4 completion, pick the next concrete non-canary operator-completeness or status-usability gap from live inspection rather than adding another handoff memo.
