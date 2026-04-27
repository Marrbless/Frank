# V4-129 - Canary Gate Completion / Stop-Widening Checkpoint

## Scope

V4-129 is a docs-only checkpoint after V4-128. It records whether the V4 canary-required hot-update path is complete through promotion audit lineage, whether canary-gate lifecycle widening should pause, and what one next slice should happen outside canary-gate implementation.

This slice does not change Go code, tests, commands, command syntax, TaskState wrappers, runtime behavior, record schemas, runtime pointers, last-known-good pointers, `reload_generation`, outcome kinds, rollback behavior, rollback-apply behavior, natural-language owner approval behavior, `CandidatePromotionDecisionRecord`, `CreateHotUpdateGateFromCandidatePromotionDecision(...)`, or V4-130 implementation.

## Live State Inspected

Starting state:

- initial branch: `frank-v4-128-canary-gate-end-to-end-runbook-fixture`
- reconciled branch: `frank-v4-129-canary-gate-completion-stop-widening-checkpoint`
- HEAD: `588fe8e1493612f1116dcfa359e406c0047f84d3`
- tag at HEAD: `frank-v4-128-canary-gate-end-to-end-runbook-fixture`
- worktree: clean before this memo
- startup validator: `/usr/local/go/bin/go test -count=1 ./...`

The expected V4-129 branch did not exist locally, so it was created from the tagged V4-128 commit before editing.

Docs inspected:

- `docs/FRANK_V4_SPEC.md`
- `docs/HOT_UPDATE_OPERATOR_RUNBOOK.md`
- `docs/maintenance/V4_125_CANARY_GATE_END_TO_END_LIFECYCLE_CHECKPOINT.md`
- `docs/maintenance/V4_126_CANARY_GATE_OPERATOR_RUNBOOK_HELP_TEXT_AFTER.md`
- `docs/maintenance/V4_127_CANARY_GATE_END_TO_END_INTEGRATION_FIXTURE_ASSESSMENT.md`
- `docs/maintenance/V4_128_CANARY_GATE_END_TO_END_RUNBOOK_FIXTURE_AFTER.md`

Code and test surfaces inspected:

- `internal/agent/loop_processdirect_canary_runbook_test.go`
- direct-command routing in `internal/agent/loop.go`
- TaskState hot-update wrapper references in `internal/agent/tools/taskstate.go`
- canary gate creation, phase, pointer-switch, reload/apply, and execution-readiness helpers in `internal/missioncontrol/hot_update_gate_registry.go`
- outcome, promotion, and status/read-model lineage surfaces in `internal/missioncontrol/hot_update_outcome_registry.go`, `internal/missioncontrol/promotion_registry.go`, and `internal/missioncontrol/status.go`
- rollback, rollback-apply, and LKG registry entry points in `internal/missioncontrol/rollback_registry.go`, `internal/missioncontrol/rollback_apply_registry.go`, and `internal/missioncontrol/runtime_pack_registry.go`

## Completed Slice Timeline

| Slice | Result |
| --- | --- |
| V4-095 | Added durable canary requirement records for candidate results with canary-required eligibility. |
| V4-096 | Assessed the canary requirement command surface. |
| V4-097 | Added `HOT_UPDATE_CANARY_REQUIREMENT_CREATE`. |
| V4-098 | Assessed durable canary evidence. |
| V4-099 | Added durable canary evidence records with passed, failed, blocked, and expired states. |
| V4-100 | Assessed the canary evidence command surface. |
| V4-101 | Added `HOT_UPDATE_CANARY_EVIDENCE_CREATE`. |
| V4-102 | Assessed the canary-passed promotion gate path. |
| V4-103 | Added read-only `AssessHotUpdateCanarySatisfaction(...)`. |
| V4-104 | Assessed durable canary satisfaction authority. |
| V4-105 | Added canary satisfaction authority records. |
| V4-106 | Assessed the canary satisfaction authority command surface. |
| V4-107 | Added `HOT_UPDATE_CANARY_SATISFACTION_AUTHORITY_CREATE`. |
| V4-108 | Assessed owner approval authority for canary-required results. |
| V4-109 | Added owner approval request records. |
| V4-110 | Assessed the owner approval request command surface. |
| V4-111 | Added `HOT_UPDATE_OWNER_APPROVAL_REQUEST_CREATE`. |
| V4-112 | Assessed owner approval granted/rejected authority. |
| V4-113 | Added owner approval decision records with terminal `granted` and `rejected` states. |
| V4-114 | Assessed the owner approval decision command surface. |
| V4-115 | Added `HOT_UPDATE_OWNER_APPROVAL_DECISION_CREATE`. |
| V4-116 | Assessed canary owner approval to gate authority. |
| V4-117 | Added canary satisfaction authority to prepared hot-update gate creation. |
| V4-118 | Assessed the canary gate direct-command surface. |
| V4-119 | Added `HOT_UPDATE_CANARY_GATE_CREATE`. |
| V4-120 | Checkpointed prepared canary gate lifecycle and identified execution readiness gaps. |
| V4-121 | Assessed canary-derived gate execution readiness. |
| V4-122 | Added read-only canary gate execution readiness and wired it into phase, pointer-switch, and reload/apply lifecycle points. |
| V4-123 | Assessed downstream outcome, promotion, rollback, rollback-apply, and LKG behavior after guarded execution. |
| V4-124 | Propagated `canary_ref` and `approval_ref` into hot-update outcomes, promotions, and read models. |
| V4-125 | Checkpointed the end-to-end canary lifecycle through promotion audit lineage. |
| V4-126 | Updated the operator runbook for canary gate operation, status checks, and fail-closed cases. |
| V4-127 | Assessed the need for a bounded direct-command end-to-end runbook fixture. |
| V4-128 | Added the direct-command runbook fixture for no-owner, owner-approved, and rejected-owner blocker paths. |

## Current Complete Authority Chain

The completed authority arc is:

```text
candidate result + promotion policy
-> canary requirement
-> selected canary evidence
-> canary satisfaction assessment
-> canary satisfaction authority
-> optional owner approval request
-> optional owner approval decision
-> prepared canary hot-update gate
```

This chain provides durable authority records for every required decision before prepared canary gate creation:

- candidate result and promotion policy establish the evaluated candidate and canary/owner-approval requirement;
- canary requirement records the result requiring canary handling;
- canary evidence records the operator-supplied canary observation;
- canary satisfaction assessment reads the current requirement/evidence/policy/result state without mutation;
- canary satisfaction authority records `authorized` for no-owner branches or `waiting_owner_approval` for owner-required branches;
- owner approval request and decision records are durable only for the owner-required branch;
- prepared canary gates carry `canary_ref` and optional `approval_ref`.

No missing canary-gate authority record remains before promotion. Owner approval is not a substitute for passed canary evidence, and rejected owner approval is not execution authority.

## Current Complete Command Chain

The governed direct-command chain is:

```text
HOT_UPDATE_CANARY_REQUIREMENT_CREATE <job_id> <result_id>
HOT_UPDATE_CANARY_EVIDENCE_CREATE <job_id> <canary_requirement_id> <evidence_state> <observed_at> [reason...]
HOT_UPDATE_CANARY_SATISFACTION_AUTHORITY_CREATE <job_id> <canary_requirement_id>
HOT_UPDATE_OWNER_APPROVAL_REQUEST_CREATE <job_id> <canary_satisfaction_authority_id>
HOT_UPDATE_OWNER_APPROVAL_DECISION_CREATE <job_id> <owner_approval_request_id> <decision> [reason...]
HOT_UPDATE_CANARY_GATE_CREATE <job_id> <canary_satisfaction_authority_id> [owner_approval_decision_id]
HOT_UPDATE_GATE_PHASE <job_id> <hot_update_id> validated
HOT_UPDATE_GATE_PHASE <job_id> <hot_update_id> staged
HOT_UPDATE_GATE_EXECUTE <job_id> <hot_update_id>
HOT_UPDATE_GATE_RELOAD <job_id> <hot_update_id>
HOT_UPDATE_OUTCOME_CREATE <job_id> <hot_update_id>
HOT_UPDATE_PROMOTION_CREATE <job_id> <outcome_id>
```

The no-owner branch omits the owner approval request/decision commands and omits `owner_approval_decision_id` from `HOT_UPDATE_CANARY_GATE_CREATE`.

The owner-approved branch uses exact durable decision token `granted`.

The rejected-owner branch uses exact durable decision token `rejected` and stops. It must not proceed to gate creation.

No remaining canary-gate command surface is required for completion through promotion audit lineage.

## Current Complete Read-Model Chain

`STATUS <job_id>` exposes the current read-model chain:

- `hot_update_canary_requirement_identity`
- `hot_update_canary_evidence_identity`
- `hot_update_canary_satisfaction_identity`
- `hot_update_canary_satisfaction_authority_identity`
- `hot_update_owner_approval_request_identity`
- `hot_update_owner_approval_decision_identity`
- `hot_update_gate_identity` with `canary_ref` and `approval_ref`
- `hot_update_outcome_identity` with copied `canary_ref` and `approval_ref`
- `promotion_identity` with copied `canary_ref` and `approval_ref`
- `rollback_identity`
- `rollback_apply_identity`
- `runtime_pack_identity.active`
- `runtime_pack_identity.last_known_good`

Invalid source records remain visible as invalid status entries instead of being repaired or hidden. Read models remain read-only.

No remaining canary-gate read-model surface is required for completion through promotion audit lineage.

## Execution Guard Coverage

The execution arc is:

```text
prepared canary gate
-> guarded phase advancement
-> guarded pointer switch
-> guarded reload/apply
-> terminal outcome
-> promotion
```

`AssessHotUpdateCanaryGateExecutionReadiness(root, hotUpdateID)` is shared and read-only. It is invoked for canary-derived gates before:

- phase advancement to `validated` or `staged`;
- pointer switch;
- reload/apply convergence writes.

For no-owner canary gates, the guard requires the gate `canary_ref` to match the canary satisfaction authority, `approval_ref` to be empty, authority state to be `authorized`, fresh satisfaction to remain `satisfied`, fresh eligibility to remain `canary_required`, selected evidence to remain selected and passed, the active pointer to remain at the expected baseline, the candidate pack and rollback target to load, and any present LKG pointer to be valid.

For owner-approved canary gates, the guard requires the gate `canary_ref` to match the authority, `approval_ref` to be present, authority state to be `waiting_owner_approval`, fresh satisfaction to remain `waiting_owner_approval`, fresh eligibility to remain `canary_and_owner_approval_required`, the owner approval decision to load, match copied refs, and be exactly `granted`, plus the same runtime-pack, active-pointer, rollback-target, and present-LKG checks.

Rejected, missing, stale, or mismatched owner approval decisions fail closed. Missing or stale canary authority fails closed. Source records are not mutated by readiness checks.

## Outcome / Promotion Audit Lineage Coverage

The audit lineage arc is:

```text
gate canary_ref / approval_ref
-> outcome copied canary_ref / approval_ref
-> promotion copied canary_ref / approval_ref
```

`CreateHotUpdateOutcomeFromTerminalGate(...)` remains generic after guarded terminal execution and does not re-run canary execution readiness. It records terminal gate state so successful canary-derived outcomes remain `outcome_kind=hot_updated`, failed canary-derived outcomes remain `outcome_kind=failed`, and `canary_applied` is not used by this path.

`CreatePromotionFromSuccessfulHotUpdateOutcome(...)` remains generic after a successful terminal outcome and does not re-run canary execution readiness. It copies lineage from the linked outcome and cross-checks outcome/gate/promotion lineage consistency.

This preserves durable downstream audit lineage without blocking outcome or promotion creation because source canary records drift after irreversible execution.

## Operator Runbook Coverage

`docs/HOT_UPDATE_OPERATOR_RUNBOOK.md` documents:

- the completed canary-required path through promotion audit lineage;
- the separate eligible-only path;
- exact no-owner and owner-approved command sequences;
- rejected owner approval as terminal rejection authority;
- `STATUS <job_id>` checklist sections;
- canary-derived gate guard behavior;
- outcome and promotion lineage behavior;
- generic rollback, rollback-apply, and LKG positioning;
- warnings against natural-language owner approval binding and against using the eligible-only candidate promotion decision path for canary-required results;
- troubleshooting for stale satisfaction, stale promotion eligibility, missing evidence, missing/mismatched owner approval, active pointer drift, missing rollback target, invalid present LKG pointer, divergent duplicates, and wrong job IDs.

The runbook is sufficient for an operator to execute the canary path without reconstructing it from maintenance memos.

## End-to-End Runbook Fixture Coverage

`internal/agent/loop_processdirect_canary_runbook_test.go` provides the executable direct-command runbook fixture from V4-128.

It covers:

- no-owner canary branch from requirement through promotion;
- owner-approved canary branch from requirement through promotion;
- rejected-owner blocker behavior for gate creation;
- `STATUS <job_id>` unmarshaled into `missioncontrol.OperatorStatusSummary`;
- configured valid canary requirement, evidence, satisfaction, satisfaction authority, owner approval request/decision when applicable, gate, outcome, and promotion identities;
- `canary_ref` propagation through gate, outcome, and promotion;
- `approval_ref` propagation through gate, outcome, and promotion for the owner-approved branch;
- `outcome_kind=hot_updated`;
- deterministic outcome and promotion linkage;
- no candidate promotion decision for canary-required branches;
- no rollback or rollback-apply records;
- no natural-language runtime approval request/grant records;
- active pointer mutation only at `HOT_UPDATE_GATE_EXECUTE`;
- `reload_generation` increment only at `HOT_UPDATE_GATE_EXECUTE`;
- absent or byte-stable LKG pointer when LKG recertification is not invoked.

The fixture is intentionally bounded and is not a full fail-closed matrix. Focused tests continue to cover the broader stale/missing/mismatched cases.

## No-Owner Branch

The complete no-owner branch is:

```text
candidate result + policy
-> canary requirement
-> passed canary evidence
-> canary satisfaction=satisfied
-> canary satisfaction authority=authorized
-> prepared canary gate with canary_ref and empty approval_ref
-> guarded phase/execute/reload
-> outcome with copied canary_ref
-> promotion with copied canary_ref
```

This branch is complete through promotion audit lineage. Supplying an owner approval decision ID for this branch is not required and should fail the branch contract.

## Owner-Approved Branch

The complete owner-approved branch is:

```text
candidate result + policy
-> canary requirement
-> passed canary evidence
-> canary satisfaction=waiting_owner_approval
-> canary satisfaction authority=waiting_owner_approval
-> owner approval request=requested
-> owner approval decision=granted
-> prepared canary gate with canary_ref and approval_ref
-> guarded phase/execute/reload
-> outcome with copied canary_ref and approval_ref
-> promotion with copied canary_ref and approval_ref
```

This branch is complete through promotion audit lineage. The owner approval decision must be exact durable `granted` authority, must match the request/authority refs, and does not replace passed canary evidence.

## Rejected-Owner Branch

```text
owner approval decision=rejected
```

means:

- durable terminal rejection authority;
- terminal blocker for canary gate creation;
- never execution authority;
- not equivalent to approval;
- not bindable through natural-language `yes`, `no`, `approve`, `deny`, `approved`, or `denied` aliases.

Rejected owner approval is complete as a stop path. It does not need more canary-gate lifecycle widening.

## Rollback / Rollback-Apply / LKG Position

Rollback, rollback-apply, and LKG recertification remain intentionally generic and are sufficient for now.

Reasoning:

- rollback and rollback-apply are recovery paths, not canary reauthorization paths;
- they must remain available even if canary evidence, satisfaction authority, or owner approval records drift after execution;
- promotion now carries copied canary/approval audit lineage through outcome and gate refs;
- operators can follow promotion -> outcome -> gate to inspect canary authority;
- V4-128 proves the canary runbook does not create rollback, rollback-apply, or LKG side effects unless those generic flows are explicitly invoked.

No canary-specific rollback policy, rollback-apply policy, LKG policy, or direct canary fields on rollback/LKG records are required before calling the canary gate path complete through promotion audit lineage.

## Explicitly Deferred / Still Not Implemented

The following remain intentionally not implemented:

- automatic canary execution / telemetry collection;
- natural-language owner approval binding;
- canary-specific rollback policy;
- canary-specific rollback-apply policy;
- canary-specific LKG recertification policy;
- use of `canary_applied` outcome kind for this path;
- direct canary fields on rollback, rollback-apply, or LKG records;
- broadening of `CandidatePromotionDecisionRecord`;
- canary-required candidate promotion decisions;
- owner approval as a substitute for passed canary evidence;
- rejected owner approval as execution authority.

These are not missing blockers for the canary gate path through promotion audit lineage. They are either explicit non-goals, future policy choices, or intentionally generic recovery surfaces.

## Stop-Widening Decision

V4 canary-gate lifecycle widening should pause now.

Evidence supports stopping:

- the canary-required path is complete through promotion audit lineage;
- durable authority records exist for each required pre-gate decision;
- governed direct commands exist for the operator sequence;
- status/read-model surfaces exist for each authority and downstream ledger surface;
- execution readiness guards run before phase, pointer switch, and reload/apply for canary-derived gates;
- outcome and promotion preserve canary/owner approval audit lineage;
- the operator runbook documents both branches and fail-closed behavior;
- V4-128 added an executable direct-command runbook fixture;
- rollback, rollback-apply, and LKG are intentionally generic and sufficient for now;
- natural-language owner approval remains intentionally unbound;
- `CandidatePromotionDecisionRecord` remains strictly eligible-only;
- `CreateHotUpdateGateFromCandidatePromotionDecision(...)` remains unchanged.

No concrete missing safety, audit, command, read-model, or test surface remains that justifies more canary-gate implementation work before moving on.

## Top Residual Risks

- The operator workflow is long despite runbook and fixture coverage.
- The V4-128 fixture is bounded and does not replace the focused fail-closed matrix.
- Rollback/LKG lineage remains indirect through promotion, outcome, and gate rather than copied directly onto rollback/LKG records.
- The existing `canary_applied` outcome kind remains unused by this path and could confuse future contributors.
- Natural-language approval remains intentionally separate from durable canary owner approval.
- Manual corruption of source records can make historical read models invalid, although invalid records remain visible.
- Canary telemetry is externally/operator recorded rather than automatically collected.

These risks are acceptable for stopping canary-gate widening because they do not represent missing authority before execution or missing lineage through promotion.

## Recommendation

Recommend exactly one next slice:

```text
V4-130 - V4 End-to-End Lifecycle Handoff Checkpoint
```

Rationale:

- canary-gate implementation should pause rather than continue widening;
- the next useful work is a broader V4 handoff checkpoint that places the completed canary-gate lifecycle beside the rest of the V4 hot-update/runtime-pack lifecycle;
- this avoids inventing additional canary-specific policy work without a concrete safety or audit gap.

V4-130 should be docs-only unless live inspection identifies a separate, non-canary missing surface. It should not implement new canary-gate behavior.
