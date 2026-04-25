# V4-093 Hot-Update Execution Safety Lifecycle Checkpoint

## Facts

V4-085 established that the controlled candidate-to-prepared-hot-update-gate path was complete, while execution still needed deploy-lock and quiesce safety.

V4-086 assessed the live hot-update lifecycle and classified only pointer switch and reload/apply as execution-sensitive transitions. It explicitly left prepared gate creation, phase advancement to `validated` or `staged`, terminal failure resolution, outcome creation, promotion creation, and LKG recertification outside the first deploy-lock/quiesce enforcement point.

V4-087 added the read-only missioncontrol guard:

```go
AssessHotUpdateExecutionReadiness(root string, input HotUpdateExecutionReadinessInput) (HotUpdateExecutionReadinessAssessment, error)
```

The guard classifies transitions, preserves replay-first behavior, blocks unproven active `live_runtime` work with `E_ACTIVE_JOB_DEPLOY_LOCK`, and blocks explicit quiesce failure with `E_RELOAD_QUIESCE_FAILED`.

V4-088 wired the guard into:

- `TaskState.ExecuteHotUpdateGatePointerSwitch`
- `TaskState.ExecuteHotUpdateGateReloadApply`

The direct commands `HOT_UPDATE_GATE_EXECUTE` and `HOT_UPDATE_GATE_RELOAD` now fail closed with empty response plus error before a new execution-sensitive transition when readiness is not proven. Exact pointer-switch and reload/apply replay remains allowed after the transition was already applied.

V4-089 assessed the missing evidence surface and selected explicit evidence over inference from coarse active-job occupancy.

V4-090 added `HotUpdateExecutionSafetyEvidenceRecord` under:

```text
runtime_packs/hot_update_execution_safety/<evidence_id>.json
```

with deterministic ID:

```text
hot-update-execution-safety-<hot_update_id>-<job_id>
```

The readiness guard now consumes current matching unexpired evidence for active `live_runtime` work and requires `deploy_lock_state=deploy_unlocked`, `quiesce_state=ready`, and current active-job binding.

V4-091 selected the narrow operator producer shape.

V4-092 added:

```text
HOT_UPDATE_EXECUTION_READY <job_id> <hot_update_id> <ttl_seconds> [reason...]
```

and `TaskState.RecordHotUpdateExecutionReady(...)`, which writes only short-lived `deploy_unlocked` plus `ready` evidence, with TTL capped at 300 seconds, `created_by=operator`, active-job binding fields from current evidence, exact replay selection, expired replacement, and audit action `hot_update_execution_ready`.

## Complete

The hot-update execution-safety path is complete enough for operator-controlled pointer-switch and reload/apply paths.

The current completed surface includes:

- read-only readiness assessment for hot-update transitions
- transition classification that distinguishes execution-sensitive work from metadata and ledger work
- TaskState enforcement before new pointer switch and reload/apply attempts
- replay-first behavior for already-applied pointer switch and reload/apply
- explicit short-lived deploy-lock/quiesce evidence
- direct operator command for readiness evidence
- fail-closed active `live_runtime` behavior when matching readiness evidence is absent, expired, stale, locked, unknown, or failed
- direct command empty-response-plus-error behavior for blocked execution attempts
- audit coverage for readiness evidence and blocked execution wrappers

This closes the universal execution-safety blocker identified in V4-085: a prepared/staged gate can no longer be newly switched or reloaded through the TaskState operator path during active live work unless explicit current readiness evidence exists.

## Still Deferred

Fully automatic execution is still blocked by policy-branch surfaces, not by the deploy-lock/quiesce guard itself.

Still deferred:

- canary-required branch
- owner-approval-required branch
- combined canary-and-owner-approval branch
- canary requirement/proposal records
- canary evidence records before any canary execution
- owner approval request/proposal records for hot-update policy
- automatic quiesce observation from tool-call or governed side-effect trackers
- append-only deploy-lock audit ledger beyond the current scoped evidence surface
- richer operator status formatting for execution-safety evidence
- final V4 handover / stop-widening checkpoint

## Core Versus Polish

V4 core now complete:

- candidate result eligibility derivation
- durable candidate promotion decision for eligible results
- prepared hot-update gate creation from a candidate decision
- direct command for candidate-decision gate creation
- read-only hot-update execution readiness assessment
- TaskState deploy-lock/quiesce enforcement for pointer switch and reload/apply
- explicit short-lived execution-safety evidence
- direct command for operator readiness evidence

Remaining V4 core:

- canary requirement/proposal surface for `canary_required`
- owner approval request/proposal surface for `owner_approval_required`
- combined handling for `canary_and_owner_approval_required`
- final handover once remaining core branches are either implemented or explicitly deferred

Later polish:

- automatic quiesce observation from in-flight tool-call and side-effect trackers
- append-only deploy-lock/quiesce audit history
- richer status/read-model formatting
- broader command aliases
- canary execution automation after records and evidence are grounded
- owner approval UX smoothing after request/proposal records are grounded

## Answers

The hot-update execution-safety path is complete enough for operator-controlled pointer-switch and reload/apply paths. New execution-sensitive transitions now require either no unsafe active occupied job or matching unexpired readiness evidence, and exact replay remains stable.

Fully automatic execution remains blocked by policy branches that intentionally have no durable proposal/evidence records yet: canary-required and owner-approval-required promotion states. The system can now execute safely when an operator records readiness, but it still cannot satisfy canary or owner-approval requirements.

V4 should next focus on canary gates, not pointer-switch/reload changes and not final handover. The execution-safety path has no concrete live gap requiring another pointer-switch or reload slice. Canary proposal/evidence records should precede canary execution, and canary evidence can also become an input to later owner-approval decisions.

Owner approval gates remain V4 core, but they should follow the first canary requirement/proposal surface unless live assessment proves an existing approval registry can be reused more directly. Final handover should wait until canary and owner-approval branches are either implemented or explicitly deferred by a dedicated checkpoint.

## Recommended V4-094 Slice

Recommend exactly one next slice:

```text
V4-094 — Hot-Update Canary Requirement Proposal Assessment
```

Scope should be docs-only assessment.

It should inspect:

- promotion eligibility states `canary_required` and `canary_and_owner_approval_required`
- `PromotionPolicyRecord` canary fields
- candidate result, promotion decision, hot-update gate, outcome, promotion, rollback, and LKG registries
- any existing canary refs on hot-update gate records
- direct command and TaskState hot-update patterns
- status/read-model surfaces that would need to expose pending canary requirements

It should decide the smallest future implementation for durable canary requirement/proposal records before any canary execution. It should not implement canary execution, owner approval, pointer switch, reload/apply changes, outcomes, promotions, rollback behavior, or LKG mutation.

Rationale:

- deploy-lock/quiesce is now grounded for operator-controlled execution
- the next unresolved V4 core branch is policy-required canary handling
- the frozen spec requires train/holdout/canary separation and references canary policy before autonomous hot promotion
- canary execution without a proposal/evidence record would widen into opaque runtime behavior
- owner approval should not be used as a substitute for missing canary evidence when policy says canary is required
- final handover before canary/approval branches are grounded would overstate V4 completion

## Must Not Widen

V4-094 must not accidentally widen into:

- pointer-switch or reload/apply behavior changes
- automatic quiesce inference
- deploy-lock ledger implementation
- canary execution
- owner approval request creation
- promotion decision creation for canary-required results
- hot-update gate creation for canary-required results
- outcome creation
- promotion creation
- rollback or rollback-apply creation
- LKG mutation
- active runtime-pack pointer mutation
- last-known-good pointer mutation
- `reload_generation` mutation
- broad operator status redesign
- final V4 completion claims

## Checkpoint Answer

The execution-safety path from staged/reloading hot-update gate to guarded operator-controlled pointer switch and reload/apply is now complete. The next safest V4 work is to ground the canary-required policy branch with proposal/evidence records before any canary execution or final V4 handover.
