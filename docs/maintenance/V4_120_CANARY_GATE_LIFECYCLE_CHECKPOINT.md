# V4-120 Canary Gate Lifecycle Checkpoint

## Scope

V4-120 is a docs-only checkpoint after V4-119. It summarizes the canary/owner-approval authority path completed so far, inspects the downstream hot-update lifecycle surfaces, and recommends the next smallest safe slice. It does not change Go code, tests, commands, TaskState wrappers, records, runtime pointers, reload/apply behavior, outcomes, promotions, rollbacks, rollback-apply records, last-known-good state, or V4-121 implementation.

## Completed Authority Chain

The complete prepared-gate authority chain is now:

```text
candidate result + policy
-> canary requirement
-> canary evidence
-> canary satisfaction assessment
-> canary satisfaction authority
-> optional owner approval request
-> optional owner approval decision
-> prepared canary hot-update gate
```

This preserves the existing `CandidatePromotionDecisionRecord` contract as strictly eligible-only. Canary-required candidate results do not create candidate promotion decisions, and `CreateHotUpdateGateFromCandidatePromotionDecision(...)` remains unchanged.

## Slice Summary

V4-095 added `HotUpdateCanaryRequirementRecord`, stored under `runtime_packs/hot_update_canary_requirements/`, with deterministic ID `hot-update-canary-requirement-<result_id>`. It captures a candidate result whose fresh promotion eligibility is `canary_required` or `canary_and_owner_approval_required`.

V4-097 exposed requirement creation through:

```text
HOT_UPDATE_CANARY_REQUIREMENT_CREATE <job_id> <result_id>
```

The TaskState wrapper validates job context, reuses `CreatedAt` for replay, emits `hot_update_canary_requirement_create`, and does not create evidence, owner approval, candidate promotion decisions, gates, outcomes, promotions, or pointer mutations.

V4-099 added `HotUpdateCanaryEvidenceRecord`, stored under `runtime_packs/hot_update_canary_evidence/`, with deterministic ID based on requirement ID and `observed_at`. Evidence states are `passed`, `failed`, `blocked`, and `expired`; only `evidence_state=passed` stores `passed=true`.

V4-101 exposed evidence creation through:

```text
HOT_UPDATE_CANARY_EVIDENCE_CREATE <job_id> <canary_requirement_id> <evidence_state> <observed_at> [reason...]
```

The command records durable operator evidence but does not execute canaries or advance promotion/gate state.

V4-103 added read-only `AssessHotUpdateCanarySatisfaction(...)`. It selects matching valid evidence deterministically, reports `satisfied`, `waiting_owner_approval`, `failed`, `blocked`, `expired`, `not_satisfied`, or `invalid`, and does not write records.

V4-105 added `HotUpdateCanarySatisfactionAuthorityRecord`, stored under `runtime_packs/hot_update_canary_satisfaction_authorities/`. It durably records either `state=authorized` with `owner_approval_required=false` and `satisfaction_state=satisfied`, or `state=waiting_owner_approval` with `owner_approval_required=true` and `satisfaction_state=waiting_owner_approval`.

V4-107 exposed authority creation through:

```text
HOT_UPDATE_CANARY_SATISFACTION_AUTHORITY_CREATE <job_id> <canary_requirement_id>
```

The command consumes the fresh read-only satisfaction assessment, reuses `CreatedAt` for replay, and stops before owner approval or gate creation.

V4-109 added `HotUpdateOwnerApprovalRequestRecord`, stored under `runtime_packs/hot_update_owner_approval_requests/`, with deterministic ID `hot-update-owner-approval-request-<canary_satisfaction_authority_id>`. The only stored request state is `requested`.

V4-111 exposed owner approval request creation through:

```text
HOT_UPDATE_OWNER_APPROVAL_REQUEST_CREATE <job_id> <canary_satisfaction_authority_id>
```

The command only accepts `waiting_owner_approval` canary satisfaction authority records and does not use runtime `ApprovalRequestRecord` / `ApprovalGrantRecord`.

V4-113 added `HotUpdateOwnerApprovalDecisionRecord`, stored under `runtime_packs/hot_update_owner_approval_decisions/` using a hash-keyed directory layout. The deterministic decision ID is `hot-update-owner-approval-decision-<owner_approval_request_id>`, and the only terminal decisions are `granted` and `rejected`.

V4-115 exposed terminal decision creation through:

```text
HOT_UPDATE_OWNER_APPROVAL_DECISION_CREATE <job_id> <owner_approval_request_id> <decision> [reason...]
```

Only exact `granted` and `rejected` are accepted. Natural-language aliases such as `yes`, `no`, `approve`, and `deny` fail closed.

V4-117 added:

```go
HotUpdateGateIDFromCanarySatisfactionAuthority(canarySatisfactionAuthorityID string) string

CreateHotUpdateGateFromCanarySatisfactionAuthority(
    root string,
    canarySatisfactionAuthorityID string,
    ownerApprovalDecisionID string,
    createdBy string,
    createdAt time.Time,
) (HotUpdateGateRecord, bool, error)
```

The helper creates/selects only `state=prepared`, `decision=keep_staged` gates. It derives the gate ID as `hot-update-canary-gate-<sha256(canary_satisfaction_authority_id)>`, sets `canary_ref=<canary_satisfaction_authority_id>`, and sets `approval_ref=<owner_approval_decision_id>` only for granted owner-approval branches.

V4-119 exposed that helper through:

```text
HOT_UPDATE_CANARY_GATE_CREATE <job_id> <canary_satisfaction_authority_id> [owner_approval_decision_id]
```

The TaskState wrapper validates job context, normalizes refs, reuses existing gate `PreparedAt` for replay, emits `hot_update_canary_gate_create`, and leaves execution/reload/outcome/promotion behavior unchanged.

## Branches

### No-Owner-Approval Branch

```text
canary_required
-> passed canary evidence
-> satisfaction=satisfied
-> authority=authorized
-> prepared gate with canary_ref and empty approval_ref
```

The V4-117/V4-119 gate path requires the caller to omit `owner_approval_decision_id`. Supplying one fails closed to avoid ambiguous authority.

### Owner-Approval Branch

```text
canary_and_owner_approval_required
-> passed canary evidence
-> satisfaction=waiting_owner_approval
-> authority=waiting_owner_approval
-> owner approval request
-> owner approval decision=granted
-> prepared gate with canary_ref and approval_ref
```

The gate helper requires a committed owner approval decision whose copied refs match the canary satisfaction authority and whose decision is exactly `granted`.

### Rejected Owner Approval

```text
owner approval decision=rejected
```

is a terminal blocker for canary gate creation. It is durable terminal authority for the request but is never accepted as `approval_ref` for a hot-update canary gate.

## Status And Read Models

The chain now exposes read-only operator identity surfaces for:

- `hot_update_canary_requirement_identity`
- `hot_update_canary_evidence_identity`
- `hot_update_canary_satisfaction_identity`
- `hot_update_canary_satisfaction_authority_identity`
- `hot_update_owner_approval_request_identity`
- `hot_update_owner_approval_decision_identity`
- `hot_update_gate_identity`

Invalid records are surfaced without hiding valid records. `hot_update_gate_identity` includes `canary_ref` and `approval_ref` for canary-derived gates. The read-model surfaces are read-only and do not mutate source records, runtime packs, active pointers, last-known-good pointers, or `reload_generation`.

## Downstream Lifecycle Inspection

`AdvanceHotUpdateGatePhase(...)` handles any valid `HotUpdateGateRecord` generically through `prepared -> validated -> staged`. It validates gate/runtime-pack linkage, but it does not inspect or revalidate `canary_ref` or `approval_ref`.

`ExecuteHotUpdateGatePointerSwitch(...)` requires `state=staged`, validates runtime-pack linkage, verifies the active pointer is still on `previous_active_pack_id`, then switches to `candidate_pack_id`, sets `update_record_ref=hot_update:<hot_update_id>`, increments `reload_generation`, and moves the gate to `reloading`. TaskState wraps it with `AssessHotUpdateExecutionReadiness(...)` for execution-sensitive deploy-lock checks. The pointer-switch logic does not inspect or revalidate canary satisfaction authority or owner approval decision refs.

`ExecuteHotUpdateGateReloadApply(...)` treats canary-derived gates the same as ordinary gates. It requires the active pointer to point at the candidate pack with the expected `update_record_ref` and `previous_active_pack_id`, runs the current restart-style convergence check, and moves to `reload_apply_succeeded` or `reload_apply_failed`. It does not branch on `canary_ref`, `approval_ref`, or `reload_mode=canary_reload`.

`CreateHotUpdateOutcomeFromTerminalGate(...)` currently creates normal outcomes from terminal gate states: `reload_apply_succeeded` becomes `outcome_kind=hot_updated`, and `reload_apply_failed` becomes `outcome_kind=failed`. It does not create canary-specific outcome kinds for canary-derived gates.

`CreatePromotionFromSuccessfulHotUpdateOutcome(...)` currently consumes any successful `outcome_kind=hot_updated` outcome and the linked gate. It checks pack and gate linkage, but it does not distinguish canary-derived gates or revalidate canary/owner-approval authority.

`RecertifyLastKnownGoodFromPromotion(...)` currently consumes a promotion and successful hot-update outcome, requires the active pointer to match the promoted pack, and then updates the last-known-good pointer. It does not apply canary-specific policy.

Rollback and rollback-apply registries remain generic recovery paths. No canary-specific rollback policy has been assessed or implemented.

## Checkpoint Answers

Existing gate phase logic can structurally handle a canary-derived prepared gate because V4-117 stores a normal `HotUpdateGateRecord` with `state=prepared`, `decision=keep_staged`, candidate pack refs, previous active pack refs, rollback target, reload mode, `canary_ref`, and optional `approval_ref`. However, phase advancement does not revalidate the canary/owner-approval authority chain, so downstream safety should not be assumed complete from structure alone.

Existing `HOT_UPDATE_GATE_EXECUTE` / pointer-switch logic likely needs a canary-specific assessment before use with canary-derived gates. It is protected by generic execution readiness, state, active pointer, and runtime-pack linkage checks, but it does not inspect `canary_ref` or `approval_ref` or re-confirm that the durable canary/owner-approval authority is still valid at execution time.

Reload/apply treats canary-derived gates like normal hot-update gates. That may be mechanically compatible because V4-117 deliberately derives the normal `reload_mode`, but it should remain unextended until an execution-readiness or lifecycle assessment decides whether canary-derived gates require extra authority checks at reload/apply time.

Outcome creation should remain normal `hot_updated` / `failed` for now. The current implementation does not need canary-specific outcome kinds to represent a successful or failed reload/apply, but it also does not prove that canary-derived gates should be allowed to reach outcome creation without a prior execution assessment.

Promotion creation should not yet be assumed safe for canary-derived gate/outcome records. Before promotion consumes a canary-derived successful outcome exactly like an eligible-only gate outcome, a canary-specific promotion authority assessment should decide whether promotion must re-check `canary_ref`, `approval_ref`, and the preserved canary/owner-approval chain.

Rollback and LKG recertification do not currently have canary-specific policy. LKG recertification depends on promotion/outcome/gate linkage and active pointer state, not canary refs. A canary-specific policy may be unnecessary, but it should be explicitly assessed after execution/outcome/promotion behavior is settled.

## Invariants Preserved

The V4-095 through V4-119 chain preserves:

- no candidate promotion decisions for canary-required states;
- no broadening of `CandidatePromotionDecisionRecord`;
- no changes to `CreateHotUpdateGateFromCandidatePromotionDecision(...)`;
- no use of owner approval as a substitute for passed canary evidence;
- no use of rejected owner approval as gate authority;
- no natural-language owner approval binding;
- no runtime `ApprovalRequestRecord` / `ApprovalGrantRecord` mutation;
- no source-record mutation after create/select helpers;
- no active runtime-pack pointer mutation before explicit gate execution;
- no last-known-good pointer mutation;
- no `reload_generation` mutation before explicit gate execution;
- no reload/apply behavior change;
- no outcome, promotion, rollback, or rollback-apply creation before explicit downstream commands.

## Top Integration Risks

1. Canary-derived prepared gates can now be created, but execution paths do not revalidate `canary_ref` / `approval_ref` authority before pointer switch.
2. Phase advancement can move a canary-derived gate to `staged` using only generic gate linkage checks.
3. Outcome and promotion helpers are generic and may accept canary-derived gate lineage once reload/apply succeeds, without a canary-specific promotion authority check.
4. Freshness can drift after prepared gate creation if source records are manually corrupted or replaced; later lifecycle paths currently rely on the immutable prepared gate record rather than re-reading the full canary authority chain.
5. LKG recertification may be mechanically correct but has not been assessed for canary-specific policy expectations.

## Next Smallest Safe Slice

The next slice should be a docs-only execution readiness assessment:

```text
V4-121 - Canary-Derived Hot-Update Gate Execution Readiness Assessment
```

That assessment should decide whether `HOT_UPDATE_GATE_PHASE`, `HOT_UPDATE_GATE_EXECUTE`, and `HOT_UPDATE_GATE_RELOAD` need canary-derived gate guards that revalidate `canary_ref`, `approval_ref`, owner approval `granted`, rejected-decision blocking, fresh canary satisfaction, and fresh promotion eligibility before any pointer switch or reload/apply. It should also decide whether outcome/promotion/LKG can remain generic after guarded execution, or whether separate outcome/promotion authority assessments are required.

V4 should not be claimed complete at this checkpoint. The prepared-gate authority path is implemented, but downstream execution, outcome, promotion, rollback, and LKG behavior for canary-derived gates has only been inspected here and remains intentionally deferred.
