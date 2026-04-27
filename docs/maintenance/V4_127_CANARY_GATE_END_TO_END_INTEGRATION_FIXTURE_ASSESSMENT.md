# V4-127 Canary Gate End-to-End Integration Fixture Assessment

## Scope

V4-127 is a docs-only assessment of whether the completed canary-required hot-update lifecycle should gain one named runbook-style integration fixture. It does not change Go code, tests, command syntax, direct-command parsing, TaskState wrappers, runtime behavior, rollback behavior, LKG behavior, `CandidatePromotionDecisionRecord`, `CreateHotUpdateGateFromCandidatePromotionDecision(...)`, or V4-128 implementation.

## Live State Inspected

Starting state after branch reconciliation:

- branch: `frank-v4-127-canary-gate-end-to-end-integration-fixture-assessment`
- HEAD: `5bc691e6c868fd59a25b5a9624a8ad4f5951cd09`
- tag at HEAD: `frank-v4-126-canary-gate-operator-runbook-help-text`
- worktree: clean before this memo
- startup validator: `/usr/local/go/bin/go test -count=1 ./...`

Docs inspected:

- `docs/FRANK_V4_SPEC.md`
- `docs/HOT_UPDATE_OPERATOR_RUNBOOK.md`
- `docs/maintenance/V4_125_CANARY_GATE_END_TO_END_LIFECYCLE_CHECKPOINT.md`
- `docs/maintenance/V4_126_CANARY_GATE_OPERATOR_RUNBOOK_HELP_TEXT_AFTER.md`

Test/code surfaces inspected:

- `internal/agent/loop_processdirect_test.go`
- `internal/missioncontrol/hot_update_canary_requirement_registry_test.go`
- `internal/missioncontrol/hot_update_canary_evidence_registry_test.go`
- `internal/missioncontrol/hot_update_canary_satisfaction_test.go`
- `internal/missioncontrol/hot_update_canary_satisfaction_authority_registry_test.go`
- `internal/missioncontrol/hot_update_owner_approval_request_registry_test.go`
- `internal/missioncontrol/hot_update_owner_approval_decision_registry_test.go`
- `internal/missioncontrol/hot_update_gate_registry_test.go`
- `internal/missioncontrol/hot_update_outcome_registry_test.go`
- `internal/missioncontrol/promotion_registry_test.go`
- `internal/missioncontrol/rollback_registry_test.go`
- `internal/missioncontrol/rollback_apply_registry_test.go`
- status/read-model tests for canary requirement, evidence, satisfaction, satisfaction authority, owner approval request/decision, gate, outcome, promotion, rollback, rollback-apply, runtime pack, and candidate promotion decision identities

## Current Focused Coverage

The focused registry and direct-command coverage is strong:

- canary requirement creation, replay, linked-record validation, stale eligibility rejection, and no downstream side effects
- canary evidence creation for `passed`, `failed`, `blocked`, and `expired`, replay, ordering, linked-record validation, and no downstream side effects
- read-only canary satisfaction assessment, deterministic evidence selection, stale eligibility rejection, and read-only behavior
- canary satisfaction authority creation for `authorized` and `waiting_owner_approval`, replay, stale satisfaction/eligibility rejection, and no downstream side effects
- owner approval request and decision creation, replay, granted/rejected states, stale satisfaction/eligibility rejection, exact decision-token parsing, alias rejection, and no downstream side effects
- canary-derived gate creation for no-owner and owner-approved branches, rejected owner approval blocking, owner branch mismatch rejection, replay/duplicate behavior, and source/runtime preservation
- canary-derived execution readiness for no-owner and owner-approved branches, stale authority, owner approval drift, active pointer drift, missing rollback target, invalid present LKG pointer, and read-only behavior
- lifecycle guard wiring for phase, pointer switch, and reload/apply, including stale-authority fail-closed behavior before mutation
- outcome and promotion canary lineage propagation, replay, divergent duplicate rejection, and no post-terminal canary reauthorization
- status/read-model exposure for every canary, owner approval, gate, outcome, promotion, rollback, rollback-apply, and runtime identity
- generic rollback, rollback-apply, and LKG behavior

Direct-command tests also cover the individual operator commands in `internal/agent/loop_processdirect_test.go`, including `HOT_UPDATE_CANARY_REQUIREMENT_CREATE`, `HOT_UPDATE_CANARY_EVIDENCE_CREATE`, `HOT_UPDATE_CANARY_SATISFACTION_AUTHORITY_CREATE`, `HOT_UPDATE_OWNER_APPROVAL_REQUEST_CREATE`, `HOT_UPDATE_OWNER_APPROVAL_DECISION_CREATE`, `HOT_UPDATE_CANARY_GATE_CREATE`, `HOT_UPDATE_GATE_PHASE`, `HOT_UPDATE_GATE_EXECUTE`, `HOT_UPDATE_GATE_RELOAD`, `HOT_UPDATE_OUTCOME_CREATE`, and `HOT_UPDATE_PROMOTION_CREATE`.

## Current Gap

The coverage is fragmented by design. It proves each authority record, guard, command, status surface, and downstream ledger behavior, but it does not provide a single named test that proves the V4-126 runbook command sequence works as a whole.

The missing evidence is not another fail-closed matrix. The gap is a bounded operator-path fixture that executes the no-owner and owner-approved command chains end-to-end through promotion audit lineage using direct commands, then checks the key read-model and side-effect invariants that an operator relies on.

## Fixture Style Decision

The next fixture should be direct-command based, not missioncontrol-only.

Reasoning:

- V4-126 is an operator runbook update, so the highest-value regression test is the runbook command surface.
- Missioncontrol helpers already have focused registry and lifecycle coverage.
- A missioncontrol-only fixture would duplicate existing helper coverage and would not prove command parsing, TaskState wrappers, audit wrappers, or status JSON composition.
- A combined direct-command plus missioncontrol integration fixture would be too broad for the smallest next slice.

The fixture should live in a new dedicated test file:

```text
internal/agent/loop_processdirect_canary_runbook_test.go
```

This keeps `internal/agent/loop_processdirect_test.go` from growing further while preserving access to same-package helpers such as `newLoopHotUpdateOutcomeAgent`, `writeLoopHotUpdateCanaryRequirementFixtures`, and related test utilities.

## Branch Coverage Recommendation

V4-128 should add two direct-command runbook tests:

1. no-owner canary branch from requirement through promotion
2. owner-approved canary branch from requirement through promotion

Both tests should use fixed timestamps and deterministic IDs. The tests should execute the same command order documented in `docs/HOT_UPDATE_OPERATOR_RUNBOOK.md`:

No-owner branch:

```text
HOT_UPDATE_CANARY_REQUIREMENT_CREATE <job_id> <result_id>
HOT_UPDATE_CANARY_EVIDENCE_CREATE <job_id> <canary_requirement_id> passed <observed_at> [reason...]
HOT_UPDATE_CANARY_SATISFACTION_AUTHORITY_CREATE <job_id> <canary_requirement_id>
HOT_UPDATE_CANARY_GATE_CREATE <job_id> <canary_satisfaction_authority_id>
HOT_UPDATE_GATE_PHASE <job_id> <hot_update_id> validated
HOT_UPDATE_GATE_PHASE <job_id> <hot_update_id> staged
HOT_UPDATE_GATE_EXECUTE <job_id> <hot_update_id>
HOT_UPDATE_GATE_RELOAD <job_id> <hot_update_id>
HOT_UPDATE_OUTCOME_CREATE <job_id> <hot_update_id>
HOT_UPDATE_PROMOTION_CREATE <job_id> <outcome_id>
```

Owner-approved branch:

```text
HOT_UPDATE_CANARY_REQUIREMENT_CREATE <job_id> <result_id>
HOT_UPDATE_CANARY_EVIDENCE_CREATE <job_id> <canary_requirement_id> passed <observed_at> [reason...]
HOT_UPDATE_CANARY_SATISFACTION_AUTHORITY_CREATE <job_id> <canary_requirement_id>
HOT_UPDATE_OWNER_APPROVAL_REQUEST_CREATE <job_id> <canary_satisfaction_authority_id>
HOT_UPDATE_OWNER_APPROVAL_DECISION_CREATE <job_id> <owner_approval_request_id> granted [reason...]
HOT_UPDATE_CANARY_GATE_CREATE <job_id> <canary_satisfaction_authority_id> <owner_approval_decision_id>
HOT_UPDATE_GATE_PHASE <job_id> <hot_update_id> validated
HOT_UPDATE_GATE_PHASE <job_id> <hot_update_id> staged
HOT_UPDATE_GATE_EXECUTE <job_id> <hot_update_id>
HOT_UPDATE_GATE_RELOAD <job_id> <hot_update_id>
HOT_UPDATE_OUTCOME_CREATE <job_id> <hot_update_id>
HOT_UPDATE_PROMOTION_CREATE <job_id> <outcome_id>
```

The fixture should stop at promotion audit lineage. It should not execute `HOT_UPDATE_LKG_RECERTIFY`, create rollback records, or create rollback-apply records. Rollback and LKG are intentionally generic and already covered separately; including them would blur the canary runbook fixture into a recovery-flow test.

## Rejected Owner Approval

V4-128 should include a small rejected-owner approval blocker assertion if it fits cleanly in the same dedicated file:

```text
HOT_UPDATE_OWNER_APPROVAL_DECISION_CREATE <job_id> <owner_approval_request_id> rejected [reason...]
HOT_UPDATE_CANARY_GATE_CREATE <job_id> <canary_satisfaction_authority_id> <owner_approval_decision_id>
```

Expected result:

- the rejected decision is recorded
- canary gate creation fails closed
- no gate, outcome, promotion, rollback, rollback-apply, candidate promotion decision, active pointer mutation, LKG mutation, or unexpected `reload_generation` mutation occurs

This assertion should be small and should not grow into another fail-closed matrix, because focused tests already cover rejected and mismatched owner approval in depth.

## Assertions To Include

Each branch fixture should assert:

- command responses are successful in the documented order
- `STATUS <job_id>` JSON exposes the expected canary requirement, evidence, satisfaction, satisfaction authority, gate, outcome, and promotion identity sections
- no-owner gate/outcome/promotion carry `canary_ref` and empty `approval_ref`
- owner-approved gate/outcome/promotion carry both `canary_ref` and `approval_ref`
- outcome kind remains `hot_updated`
- promotion links the deterministic outcome and hot-update IDs
- `ListCandidatePromotionDecisionRecords` remains empty
- `ListRollbackRecords` remains empty
- `ListRollbackApplyRecords` remains empty
- LKG pointer is not created or mutated by the canary runbook fixture unless the fixture deliberately seeds an LKG pointer for readiness; if seeded, it must remain byte-stable
- active pointer mutates only at `HOT_UPDATE_GATE_EXECUTE`
- `reload_generation` increments only at `HOT_UPDATE_GATE_EXECUTE` and is not changed by reload, outcome, promotion, or status reads

The fixture should not reassert every fail-closed branch. It should rely on existing focused tests for stale satisfaction, stale eligibility, missing selected evidence, missing owner approval decision, mismatched owner refs, active pointer drift, missing rollback target, invalid present LKG pointer, divergent duplicates, and natural-language approval aliases.

## Natural-Language Owner Approval

The E2E fixture should use only exact `granted` and `rejected` owner approval decision tokens. Natural-language aliases such as `yes`, `no`, `approve`, `deny`, `approved`, and `denied` are already covered by focused direct-command tests and should not be mixed into the happy-path runbook fixture.

If V4-128 includes the small rejected-owner blocker assertion, it can also assert that no committed approval request/grant records from the older natural-language approval path are created. It should not attempt to bind natural-language approval to the canary path.

## Determinism

Use deterministic inputs:

- fixed `time.Date(...)` values in fixtures
- fixed RFC3339/RFC3339Nano `observed_at` strings for canary evidence commands
- repo helper ID functions for expected IDs, including `HotUpdateCanaryRequirementIDFromResult`, `HotUpdateCanaryEvidenceIDFromRequirementObservedAt`, `HotUpdateCanarySatisfactionAuthorityIDFromRequirementEvidence` if needed, `HotUpdateOwnerApprovalRequestIDFromCanarySatisfactionAuthority`, `HotUpdateOwnerApprovalDecisionIDFromRequest`, `HotUpdateGateIDFromCanarySatisfactionAuthority`, `HotUpdateOutcomeIDFromHotUpdate`, and `PromotionIDFromHotUpdate`
- status JSON unmarshaled into `missioncontrol.OperatorStatusSummary` rather than substring-only checks

Avoid hardcoding hash-derived canary gate IDs where helper functions can derive them.

## Runtime And Brittleness Controls

To keep V4-128 bounded:

- add one dedicated direct-command test file
- reuse existing same-package fixture helpers instead of building a new fixture framework
- use at most two happy-path tests plus one small rejected-owner blocker test
- avoid sleeps, network, provider calls, and real-time timestamps
- check status at major milestones, not after every single command
- keep detailed fail-closed matrices in the focused tests that already exist
- stop at promotion audit lineage

## Assessment Answers

Focused coverage across V4-095 through V4-126 is strong but not enough to close the runbook evidence gap. One named runbook-style fixture is worth adding now.

The fixture should be direct-command based. Missioncontrol-only coverage is already strong and should not be duplicated in the next slice.

The fixture should live in `internal/agent/loop_processdirect_canary_runbook_test.go`, a new dedicated file in the existing `internal/agent` test package.

It should walk the no-owner branch from requirement through promotion.

It should walk the owner-approved branch from requirement through promotion.

It should include a small rejected-owner-approval terminal blocker assertion if it remains small and does not become another matrix.

It should not include rollback, rollback-apply, or LKG recertification. It should assert that those records/pointers are not created or mutated unless explicitly seeded for readiness.

It should assert status/read-model surfaces at major milestones and final promotion, not after every single command.

It should assert `canary_ref` and `approval_ref` propagation through gate, outcome, and promotion.

It should assert no `CandidatePromotionDecisionRecord` is created for canary-required results.

It should not add another natural-language approval alias matrix. Existing focused direct-command tests already cover alias rejection. The E2E fixture should demonstrate exact durable `granted` / `rejected` tokens only.

It should assert no rollback/LKG side effects unless explicitly invoked. Since the fixture should stop at promotion, rollback and rollback-apply record counts should remain zero, and LKG should remain absent or byte-stable if seeded.

Deterministic timestamps and IDs should be handled with fixed timestamps and existing ID helper functions.

The fixture should avoid excessive runtime and brittleness by using direct commands against in-memory/tempdir fixtures, no provider calls, no sleeps, status checks at major milestones, and limited assertions focused on the runbook contract.

## Recommended V4-128 Slice

The next slice should be exactly:

```text
V4-128 - Canary Gate End-to-End Runbook Fixture
```

Recommended scope:

- add `internal/agent/loop_processdirect_canary_runbook_test.go`
- add one direct-command runbook test for the no-owner canary branch from requirement through promotion
- add one direct-command runbook test for the owner-approved canary branch from requirement through promotion
- add one small rejected-owner approval blocker assertion if it fits in the same file without broadening into a matrix
- assert status/read-model canary lineage through gate, outcome, and promotion
- assert no candidate promotion decision is created
- assert no rollback or rollback-apply records are created
- assert no LKG recertification occurs
- assert active pointer and `reload_generation` change only at the intended pointer switch
- do not add commands, change syntax, change runtime behavior, bind natural-language owner approval, broaden `CandidatePromotionDecisionRecord`, or change rollback/LKG behavior

This is the smallest safe next slice because it converts the V4-126 runbook from documentation into a compact executable regression without changing production behavior or duplicating the existing fail-closed coverage.

## Explicit Non-Goals

V4-127 does not:

- change Go code or tests
- add integration tests
- add commands
- add TaskState wrappers
- change command syntax
- change runtime behavior
- bind natural-language owner approval
- broaden `CandidatePromotionDecisionRecord`
- change `CreateHotUpdateGateFromCandidatePromotionDecision(...)`
- change outcome kinds
- change rollback, rollback-apply, LKG, pointer-switch, or reload/apply behavior
- implement V4-128
