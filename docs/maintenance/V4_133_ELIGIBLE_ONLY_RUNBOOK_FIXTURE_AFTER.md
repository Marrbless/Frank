# V4-133 Eligible-Only Runbook Fixture - After

## Live Starting Point

- Previous slice: V4-132 Runbook Drift / Command Help Consistency
- Working branch: `frank-v4-133-v4-completion-gap-rescan-closure`
- Prior commit: `7e6b8f0 test: add V4 runbook command help drift coverage`
- Prior tag: `frank-v4-132-runbook-drift-command-help-consistency`

## Gap Closed

Added a bounded direct-command runbook fixture for the eligible-only hot-update lifecycle. This complements the existing canary runbook fixture and proves the baseline operator path works as a whole instead of only through fragmented command tests.

## Implementation Scope

New test file:

- `internal/agent/loop_processdirect_eligible_runbook_test.go`

New test:

- `TestProcessDirectEligibleRunbookFromDecisionThroughLastKnownGood`

The fixture walks:

```text
HOT_UPDATE_GATE_FROM_DECISION
HOT_UPDATE_GATE_PHASE validated
HOT_UPDATE_GATE_PHASE staged
HOT_UPDATE_EXECUTION_READY
HOT_UPDATE_GATE_EXECUTE
HOT_UPDATE_GATE_RELOAD
HOT_UPDATE_OUTCOME_CREATE
HOT_UPDATE_PROMOTION_CREATE
HOT_UPDATE_LKG_RECERTIFY
STATUS
```

## Assertions Added

The fixture asserts:

- each documented command returns a successful acknowledgement
- active runtime-pack pointer stays byte-stable until `HOT_UPDATE_GATE_EXECUTE`
- active runtime-pack pointer changes to the candidate pack only at execute
- `reload_generation` increments only at execute
- reload, outcome creation, promotion creation, LKG recertification, and status reads do not mutate the active pointer
- LKG remains byte-stable until `HOT_UPDATE_LKG_RECERTIFY`
- LKG recertifies to the candidate pack from the deterministic promotion ref
- the candidate-promotion-decision ledger contains exactly the eligible decision used by the runbook
- final `STATUS <job_id>` surfaces configured gate, outcome, promotion, active runtime pack, and LKG identities
- eligible-only gate, outcome, and promotion keep empty `canary_ref` and `approval_ref`
- no canary requirement, canary evidence, canary satisfaction authority, owner approval request, owner approval decision, rollback, rollback-apply, or runtime approval records are created

## Invariants Preserved

- Production behavior is unchanged.
- Command syntax is unchanged.
- No new command was added.
- No TaskState wrapper changed.
- `CandidatePromotionDecisionRecord` remains eligible-only.
- `CreateHotUpdateGateFromCandidatePromotionDecision(...)` remains unchanged.
- Canary-gate lifecycle behavior is unchanged.
- Natural-language owner approval remains unbound to canary owner approval.
- Rollback and rollback-apply remain generic and are not executed by this fixture.

## Validation Run

Validation for this slice is:

```text
/usr/local/go/bin/gofmt -w internal/agent/loop_processdirect_eligible_runbook_test.go
git diff --check
/usr/local/go/bin/go test -count=1 ./internal/agent
/usr/local/go/bin/go test -count=1 ./internal/missioncontrol
/usr/local/go/bin/go test -count=1 ./cmd/picobot
/usr/local/go/bin/go test -count=1 ./internal/agent/tools
/usr/local/go/bin/go test -count=1 ./...
```

## Risks

The fixture is intentionally bounded. It does not duplicate the full failure matrix from focused registry and direct-command tests, and it does not execute rollback or rollback-apply flows.

## Recommended Next Action

Continue V4 completion by re-scanning for a concrete non-canary status, operator usability, or executable validation gap. Do not continue canary-gate widening unless live inspection identifies a concrete missing safety or audit surface.
