# V4-130 - V4 End-to-End Lifecycle Handoff Checkpoint

## 1. Scope

V4-130 is a docs-only handoff checkpoint after V4-129. It tells the next AI/operator where the V4 hot-update lifecycle stands end to end, what is complete, what remains intentionally generic or deferred, and what one next slice should happen.

This slice does not change Go code, tests, commands, command syntax, TaskState wrappers, schemas, runtime behavior, rollback/LKG behavior, active pointer behavior, reload/apply behavior, outcome kinds, natural-language owner approval behavior, `CandidatePromotionDecisionRecord`, `CreateHotUpdateGateFromCandidatePromotionDecision(...)`, or V4-131 implementation.

## 2. Live State Inspected

Startup state:

- initial branch: `frank-v4-129-canary-gate-completion-stop-widening-checkpoint`
- reconciled branch: `frank-v4-130-v4-end-to-end-lifecycle-handoff-checkpoint`
- HEAD: `60a1f123f4233d96ea8b20500b22245a442e5d42`
- tag at HEAD: `frank-v4-129-canary-gate-completion-stop-widening-checkpoint`
- worktree: clean before this memo
- startup validator: `/usr/local/go/bin/go test -count=1 ./...` passed

The expected V4-130 branch did not exist locally, so it was created from the tagged V4-129 commit before editing.

Key docs inspected:

- `docs/FRANK_V4_SPEC.md`
- `docs/HOT_UPDATE_OPERATOR_RUNBOOK.md`
- `docs/maintenance/V4_061_HOT_UPDATE_END_TO_END_LIFECYCLE_CHECKPOINT.md`
- `docs/maintenance/V4_085_CANDIDATE_DECISION_HOT_UPDATE_GATE_LIFECYCLE_CHECKPOINT.md`
- `docs/maintenance/V4_093_HOT_UPDATE_EXECUTION_SAFETY_LIFECYCLE_CHECKPOINT.md`
- `docs/maintenance/V4_120_CANARY_GATE_LIFECYCLE_CHECKPOINT.md`
- `docs/maintenance/V4_125_CANARY_GATE_END_TO_END_LIFECYCLE_CHECKPOINT.md`
- `docs/maintenance/V4_126_CANARY_GATE_OPERATOR_RUNBOOK_HELP_TEXT_AFTER.md`
- `docs/maintenance/V4_128_CANARY_GATE_END_TO_END_RUNBOOK_FIXTURE_AFTER.md`
- `docs/maintenance/V4_129_CANARY_GATE_COMPLETION_STOP_WIDENING_CHECKPOINT.md`

Key code/test surfaces inspected:

- `internal/missioncontrol/hot_update_gate_registry.go`
- `internal/missioncontrol/hot_update_execution_readiness.go`
- `internal/missioncontrol/hot_update_outcome_registry.go`
- `internal/missioncontrol/promotion_registry.go`
- `internal/missioncontrol/runtime_pack_registry.go`
- `internal/missioncontrol/rollback_registry.go`
- `internal/missioncontrol/rollback_apply_registry.go`
- `internal/missioncontrol/status.go`
- `internal/agent/loop.go`
- `internal/agent/tools/taskstate.go`
- `internal/agent/loop_processdirect_canary_runbook_test.go`

## 3. Completed V4 Arcs

### Hot-Update Gate Lifecycle Arc

The eligible-only hot-update lifecycle is implemented and validated:

```text
runtime pack / candidate pack
-> candidate result / promotion policy
-> promotion eligibility
-> candidate promotion decision for eligible-only results
-> prepared hot-update gate
-> execution readiness / deploy lock / quiesce evidence
-> phase advancement
-> pointer switch
-> reload/apply
-> terminal outcome
-> promotion
-> optional rollback / rollback-apply
-> optional LKG recertification
```

The lifecycle is explicit and stepwise. Gate creation, phase movement, pointer switch, reload/apply, outcome creation, promotion creation, rollback, rollback-apply, and LKG recertification are separate operator-controlled ledger or transition steps.

### Canary-Required Authority Arc

The canary-required authority arc is implemented:

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

The authority chain has durable records for every pre-gate decision. Owner approval is optional by policy branch and is never a substitute for passed canary evidence.

### Canary-Required Execution Arc

The canary-required execution arc is implemented:

```text
prepared canary gate
-> guarded phase advancement
-> guarded pointer switch
-> guarded reload/apply
-> terminal outcome
-> promotion
```

Canary-derived gates use the shared canary execution-readiness guard before phase advancement to `validated` or `staged`, pointer switch, and reload/apply convergence writes.

### Canary Audit Lineage Arc

The audit lineage arc is implemented:

```text
gate canary_ref / approval_ref
-> outcome copied canary_ref / approval_ref
-> promotion copied canary_ref / approval_ref
```

Outcome and promotion creation remain generic after guarded terminal execution, but downstream records preserve immutable canary/owner approval lineage.

### Operator / Runbook / Test Arc

The operator arc is implemented:

```text
docs/HOT_UPDATE_OPERATOR_RUNBOOK.md
-> direct-command canary sequence
-> STATUS checklist
-> V4-128 executable runbook fixture
```

The runbook covers eligible-only and canary-required paths. The V4-128 direct-command fixture proves the no-owner, owner-approved, and rejected-owner blocker paths through the operator command surface.

## 4. Completed Slice Timeline

Early V4 foundation through V4-059 built runtime-pack identity, improvement candidate/run/result records, hot-update gate/read-model/control, rollback and rollback-apply state machines, outcome creation, promotion creation, and LKG recertification helpers/control entries.

| Slice | Result |
| --- | --- |
| V4-060 | Added `HOT_UPDATE_LKG_RECERTIFY` control entry. |
| V4-061 | Checkpointed hot-update lifecycle from gate through LKG recertification. |
| V4-062 | Assessed operator UX/help for the completed hot-update lifecycle. |
| V4-063 | Added the hot-update operator runbook. |
| V4-064 | Assessed Frank V4 frozen spec gaps. |
| V4-065 | Added execution-plane / execution-host / mission-family skeleton. |
| V4-066 | Added V4 rejection code skeleton. |
| V4-067 | Added improvement-family admission control. |
| V4-068 | Added improvement target/mutable/immutable surface declarations. |
| V4-069 | Enforced source-patch artifact-only admission. |
| V4-070 | Gated topology mode by default. |
| V4-071 | Added promotion policy registry skeleton. |
| V4-072 | Added promotion policy reference admission. |
| V4-073 | Added baseline/train/holdout evidence references. |
| V4-074 | Hardened eval-suite immutability. |
| V4-075 | Added improvement-run evidence linkage. |
| V4-076 | Added candidate-result promotion eligibility refs. |
| V4-077 | Assessed promotion policy eligibility evaluation. |
| V4-078 | Added candidate result promotion eligibility helper. |
| V4-079 | Assessed promotion eligibility to candidate decision. |
| V4-080 | Added `CandidatePromotionDecisionRecord`. |
| V4-081 | Assessed candidate decision to hot-update gate. |
| V4-082 | Added hot-update gate creation from candidate promotion decision. |
| V4-083 | Assessed candidate-decision gate command surface. |
| V4-084 | Added `HOT_UPDATE_GATE_FROM_DECISION`. |
| V4-085 | Checkpointed eligible candidate decision to prepared hot-update gate lifecycle. |
| V4-086 | Assessed deploy-lock/quiesce enforcement. |
| V4-087 | Added hot-update execution readiness guard. |
| V4-088 | Wired execution readiness into pointer switch and reload/apply. |
| V4-089 | Assessed explicit quiesce/deploy-lock evidence. |
| V4-090 | Added hot-update execution safety evidence registry. |
| V4-091 | Assessed execution safety evidence command surface. |
| V4-092 | Added `HOT_UPDATE_EXECUTION_READY`. |
| V4-093 | Checkpointed hot-update execution safety lifecycle. |
| V4-094 | Assessed canary requirement proposal path. |
| V4-095 | Added durable canary requirement records. |
| V4-096 | Assessed canary requirement command surface. |
| V4-097 | Added `HOT_UPDATE_CANARY_REQUIREMENT_CREATE`. |
| V4-098 | Assessed durable canary evidence. |
| V4-099 | Added canary evidence records. |
| V4-100 | Assessed canary evidence command surface. |
| V4-101 | Added `HOT_UPDATE_CANARY_EVIDENCE_CREATE`. |
| V4-102 | Assessed canary-passed promotion gate path. |
| V4-103 | Added read-only `AssessHotUpdateCanarySatisfaction(...)`. |
| V4-104 | Assessed durable canary satisfaction authority. |
| V4-105 | Added canary satisfaction authority records. |
| V4-106 | Assessed canary satisfaction authority command surface. |
| V4-107 | Added `HOT_UPDATE_CANARY_SATISFACTION_AUTHORITY_CREATE`. |
| V4-108 | Assessed owner approval authority for canary-required results. |
| V4-109 | Added owner approval request records. |
| V4-110 | Assessed owner approval request command surface. |
| V4-111 | Added `HOT_UPDATE_OWNER_APPROVAL_REQUEST_CREATE`. |
| V4-112 | Assessed owner approval grant/rejection authority. |
| V4-113 | Added owner approval decision records with `granted` and `rejected`. |
| V4-114 | Assessed owner approval decision command surface. |
| V4-115 | Added `HOT_UPDATE_OWNER_APPROVAL_DECISION_CREATE`. |
| V4-116 | Assessed canary owner approval to gate authority. |
| V4-117 | Added canary satisfaction authority to prepared hot-update gate helper. |
| V4-118 | Assessed canary gate direct-command surface. |
| V4-119 | Added `HOT_UPDATE_CANARY_GATE_CREATE`. |
| V4-120 | Checkpointed prepared canary gate lifecycle and downstream gaps. |
| V4-121 | Assessed canary-derived gate execution readiness. |
| V4-122 | Added canary gate execution readiness and lifecycle guard wiring. |
| V4-123 | Assessed downstream outcome/promotion/rollback/LKG behavior. |
| V4-124 | Propagated canary/approval lineage into outcome, promotion, and read models. |
| V4-125 | Checkpointed canary lifecycle through promotion audit lineage. |
| V4-126 | Updated operator runbook/help text for canary gate operation. |
| V4-127 | Assessed need for an end-to-end direct-command fixture. |
| V4-128 | Added the canary runbook fixture. |
| V4-129 | Checkpointed canary-gate completion and stop-widening decision. |

## 5. Command Surface Handoff

### Eligible-Only Hot-Update Path

Current major commands:

```text
HOT_UPDATE_GATE_FROM_DECISION <job_id> <promotion_decision_id>
HOT_UPDATE_GATE_RECORD <job_id> <hot_update_id> <candidate_pack_id>
HOT_UPDATE_EXECUTION_READY <job_id> <hot_update_id> <ttl_seconds> [reason...]
HOT_UPDATE_GATE_PHASE <job_id> <hot_update_id> validated
HOT_UPDATE_GATE_PHASE <job_id> <hot_update_id> staged
HOT_UPDATE_GATE_EXECUTE <job_id> <hot_update_id>
HOT_UPDATE_GATE_RELOAD <job_id> <hot_update_id>
HOT_UPDATE_GATE_FAIL <job_id> <hot_update_id> <reason>
HOT_UPDATE_OUTCOME_CREATE <job_id> <hot_update_id>
HOT_UPDATE_PROMOTION_CREATE <job_id> <outcome_id>
HOT_UPDATE_LKG_RECERTIFY <job_id> <promotion_id>
ROLLBACK_RECORD <job_id> <promotion_id> <rollback_id>
ROLLBACK_APPLY_RECORD <job_id> <rollback_id> <apply_id>
ROLLBACK_APPLY_PHASE <job_id> <apply_id> <phase>
ROLLBACK_APPLY_EXECUTE <job_id> <apply_id>
ROLLBACK_APPLY_RELOAD <job_id> <apply_id>
ROLLBACK_APPLY_FAIL <job_id> <apply_id> <reason>
```

`HOT_UPDATE_GATE_FROM_DECISION` is the eligible-only path from a durable `CandidatePromotionDecisionRecord` to a prepared gate. `HOT_UPDATE_GATE_RECORD` remains supported as direct gate creation/selection for operator-controlled hot-update workflows.

### Canary-Required Path

Current canary command chain:

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

Natural-language approval aliases remain intentionally unbound for canary owner approval. Only exact durable owner approval decisions `granted` and `rejected` participate in the canary path.

## 6. Read-Model / STATUS Handoff

`STATUS <job_id>` currently exposes these V4-relevant identity surfaces:

- `runtime_pack_identity.active`
- `runtime_pack_identity.last_known_good`
- `improvement_candidate_identity`
- `eval_suite_identity`
- `promotion_policy_identity`
- `improvement_run_identity`
- `candidate_result_identity`
- `candidate_promotion_decision_identity`
- `hot_update_canary_requirement_identity`
- `hot_update_canary_evidence_identity`
- `hot_update_canary_satisfaction_identity`
- `hot_update_canary_satisfaction_authority_identity`
- `hot_update_owner_approval_request_identity`
- `hot_update_owner_approval_decision_identity`
- `hot_update_gate_identity`
- `hot_update_outcome_identity`
- `promotion_identity`
- `rollback_identity`
- `rollback_apply_identity`

Status also still includes broader V3/V4 operator surfaces such as campaign preflight, treasury preflight, provider/mailbox bootstrap preflight, budget blockers, approval request/history, recent audit, artifacts, campaign outbound/reply work, Frank inbound replies, campaign send gate, send proof, deferred scheduler triggers, and truncation metadata.

Invalid V4 records remain visible as invalid read-model entries and are not repaired by status reads.

## 7. Safety Guard Handoff

Current safety guard surfaces:

- `AssessHotUpdateExecutionReadiness(...)` gates execution-sensitive hot-update transitions with deploy-lock and quiesce checks.
- `HOT_UPDATE_EXECUTION_READY` records short-lived explicit deploy-unlocked/quiesce-ready evidence.
- `TaskState.ExecuteHotUpdateGatePointerSwitch(...)` and `TaskState.ExecuteHotUpdateGateReloadApply(...)` preserve replay-first behavior while blocking new unsafe execution attempts.
- `AssessHotUpdateCanaryGateExecutionReadiness(...)` revalidates canary-derived gate authority before canary phase advancement, pointer switch, and reload/apply.
- Owner-required canary gates require exact `decision=granted`; rejected owner approval is durable terminal rejection and never execution authority.
- Pointer switch remains the only happy-path active pointer mutation point and increments `reload_generation` only there.
- Reload/apply does not switch the active pointer again and does not increment `reload_generation`.
- Outcome creation, promotion creation, rollback recording, rollback-apply recording, and LKG recertification are explicit steps rather than automatic side effects.
- Deterministic IDs and timestamp reuse preserve byte-stable replay for deterministic records and wrappers.
- Divergent duplicate records fail closed.

## 8. Canary Gate Stop-Widening Decision

V4-129 decision stands: canary-gate lifecycle widening should pause now.

The canary-required path is complete through promotion audit lineage. Rollback, rollback-apply, and LKG remain intentionally generic. No remaining canary-gate authority, command, read-model, or test gap justifies more canary-gate implementation work.

Do not continue canary-gate widening unless a concrete missing safety or audit surface is identified from live code and tests.

## 9. Deferred / Intentionally Not Implemented

| Deferred item | Classification | Handoff position |
| --- | --- | --- |
| Automatic canary execution / telemetry collection | Possible future policy choice | Current canary evidence is operator-recorded and durable; no missing safety surface before promotion. |
| Natural-language owner approval binding | Explicit non-goal | Durable exact `granted` / `rejected` canary owner approval remains separate from runtime natural-language approval. |
| Canary-specific rollback policy | Possible future policy choice | Rollback remains generic recovery and should not be blocked by later canary drift. |
| Canary-specific rollback-apply policy | Possible future policy choice | Rollback-apply remains generic recovery and should not be canary reauthorization. |
| Canary-specific LKG recertification policy | Possible future policy choice | LKG recertification remains generic after promotion. |
| `canary_applied` outcome kind usage for this path | Possible future policy choice | This path intentionally keeps `hot_updated` and `failed` outcome kinds. |
| Direct canary fields on rollback, rollback-apply, or LKG records | Possible future audit choice | Current lineage is indirect through promotion -> outcome -> gate. |
| Broadening `CandidatePromotionDecisionRecord` | Explicit non-goal | Candidate promotion decisions remain strictly eligible-only. |
| Canary-required candidate promotion decisions | Explicit non-goal | Canary-required results use canary requirement/evidence/authority records instead. |
| Owner approval as a substitute for passed canary evidence | Explicit non-goal | Owner approval can only follow passed canary evidence when policy requires owner approval. |
| Rejected owner approval as execution authority | Explicit non-goal | Rejected owner approval is terminal rejection authority. |
| Automatic operator sequencing / workflow engine for the full command chain | Operator ergonomics gap | The workflow is explicit and validated; automation would be a separate UX/policy choice. |

No item in this table is a true missing safety surface for the completed V4 lifecycle through canary promotion audit lineage.

## 10. Residual Risks

Ranked residual risks:

1. The operator workflow is long despite runbook coverage. This is the highest usability risk and may lead to sequencing mistakes.
2. Many direct commands are hard to discover without the runbook. There is no central command help surface for the full V4 flow.
3. `STATUS <job_id>` can be large/noisy. It exposes the needed surfaces but may be difficult to scan during live operation.
4. The V4-128 E2E fixture is bounded, not a full matrix. Focused tests cover fail-closed cases, but the runbook fixture intentionally covers only main branches and a small rejected-owner blocker.
5. Rollback/LKG canary lineage is indirect through promotion/outcome/gate. This is acceptable now but can slow audits.
6. The unused `canary_applied` enum may confuse future contributors.
7. Natural-language approval remains separate from durable canary owner approval. This is intentional but easy to misunderstand.
8. Manual corruption can make historical read models invalid. Status surfaces invalidity but does not repair source records.
9. Canary telemetry is operator-recorded, not automatically collected.

The residual risk profile argues for handoff/export and operator/status polish, not more canary-gate implementation.

## 11. Recommended Next Slice

Recommend exactly one next slice:

```text
V4-131 - V4 Handoff Export / AI Continuation Memo
```

Why this is the smallest safe next slice:

- V4 canary gate widening should pause.
- V4 now has a broad implemented hot-update lifecycle and a deep maintenance-doc stack.
- A compact continuation memo would let future AI/operator sessions resume without rereading every V4 slice from V4-001 onward.
- It is lower risk than operator behavior changes and does not invent more canary-specific policy.

Why it is not more canary-gate implementation:

- V4-129 found no concrete canary-gate safety, authority, command, read-model, or test gap.
- Canary-required execution is guarded through promotion audit lineage.
- Rollback/LKG are intentionally generic.

Recommended V4-131 type: docs-only.

Expected changed files:

- `docs/maintenance/V4_131_V4_HANDOFF_EXPORT_AI_CONTINUATION_MEMO.md`

Explicit V4-131 non-goals:

- no Go code changes;
- no test changes;
- no command changes;
- no TaskState wrappers;
- no runtime behavior changes;
- no canary-gate widening;
- no rollback/LKG behavior changes;
- no natural-language owner approval binding;
- no `CandidatePromotionDecisionRecord` broadening.

## 12. Handoff Summary For Next AI

```text
Current branch:
  frank-v4-130-v4-end-to-end-lifecycle-handoff-checkpoint

Current base:
  HEAD 60a1f123f4233d96ea8b20500b22245a442e5d42
  tag  frank-v4-129-canary-gate-completion-stop-widening-checkpoint

Completed canary path:
  candidate result + policy
  -> canary requirement
  -> selected passed canary evidence
  -> canary satisfaction
  -> canary satisfaction authority
  -> optional owner approval request/decision
  -> prepared canary gate
  -> guarded phase/execute/reload
  -> outcome with copied canary_ref / approval_ref
  -> promotion with copied canary_ref / approval_ref

Validate repo:
  git diff --name-only
  git diff --check
  /usr/local/go/bin/go test -count=1 ./...

Read first:
  docs/FRANK_V4_SPEC.md
  docs/HOT_UPDATE_OPERATOR_RUNBOOK.md
  docs/maintenance/V4_061_HOT_UPDATE_END_TO_END_LIFECYCLE_CHECKPOINT.md
  docs/maintenance/V4_085_CANDIDATE_DECISION_HOT_UPDATE_GATE_LIFECYCLE_CHECKPOINT.md
  docs/maintenance/V4_093_HOT_UPDATE_EXECUTION_SAFETY_LIFECYCLE_CHECKPOINT.md
  docs/maintenance/V4_125_CANARY_GATE_END_TO_END_LIFECYCLE_CHECKPOINT.md
  docs/maintenance/V4_129_CANARY_GATE_COMPLETION_STOP_WIDENING_CHECKPOINT.md
  docs/maintenance/V4_130_V4_END_TO_END_LIFECYCLE_HANDOFF_CHECKPOINT.md

Key risks:
  long operator workflow
  command discoverability
  noisy status surface
  bounded E2E fixture
  indirect rollback/LKG lineage
  unused canary_applied enum
  manual corruption surfaces invalid read models

Recommended next action:
  V4-131 - V4 Handoff Export / AI Continuation Memo

Do not:
  continue canary-gate widening unless a concrete missing safety/audit surface is identified.
```
