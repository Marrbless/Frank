# V4-112 Hot-Update Owner Approval Grant/Rejection Authority Assessment

## Scope

V4-112 assesses the smallest safe durable authority path for terminal owner approval decisions after a `HotUpdateOwnerApprovalRequestRecord` exists.

This is a docs-only slice. It does not change Go code, tests, direct commands, `TaskState` wrappers, owner approval request records, owner approval grants, owner approval rejections, expiration records, natural-language approval binding, runtime approval records, canary satisfaction authority records, candidate promotion decisions, hot-update gates, outcomes, promotions, rollbacks, rollback-apply records, runtime-pack pointers, last-known-good pointers, `reload_generation`, pointer-switch behavior, reload/apply behavior, or V4-113 implementation.

## Live State Inspected

Docs inspected:

- `docs/FRANK_V4_SPEC.md`
- `docs/maintenance/V4_108_HOT_UPDATE_OWNER_APPROVAL_AUTHORITY_PATH_ASSESSMENT.md`
- `docs/maintenance/V4_109_HOT_UPDATE_OWNER_APPROVAL_REQUEST_REGISTRY_AFTER.md`
- `docs/maintenance/V4_110_HOT_UPDATE_OWNER_APPROVAL_REQUEST_CONTROL_ENTRY_ASSESSMENT.md`
- `docs/maintenance/V4_111_HOT_UPDATE_OWNER_APPROVAL_REQUEST_CONTROL_ENTRY_AFTER.md`

Code surfaces inspected:

- `internal/missioncontrol/approval.go`
- `internal/missioncontrol/store_records.go`
- `internal/missioncontrol/store_hydrate.go`
- `internal/missioncontrol/status.go`
- `internal/missioncontrol/store_project.go`
- `internal/missioncontrol/hot_update_owner_approval_request_registry.go`
- `internal/missioncontrol/hot_update_canary_satisfaction_authority_registry.go`
- `internal/missioncontrol/hot_update_canary_satisfaction.go`
- `internal/missioncontrol/hot_update_gate_registry.go`
- `internal/missioncontrol/candidate_promotion_decision_registry.go`
- `internal/missioncontrol/hot_update_outcome_registry.go`
- `internal/missioncontrol/promotion_registry.go`
- `internal/missioncontrol/rollback_registry.go`
- `internal/missioncontrol/rollback_apply_registry.go`
- `internal/missioncontrol/runtime_pack_registry.go`
- `internal/agent/loop.go`
- `internal/agent/tools/taskstate.go`
- `internal/agent/tools/taskstate_readout.go`

## Current Owner Approval Request Surface

V4-109 added immutable owner approval request authority:

```go
type HotUpdateOwnerApprovalRequestRecord struct {
    RecordVersion int
    OwnerApprovalRequestID string
    CanarySatisfactionAuthorityID string
    CanaryRequirementID string
    SelectedCanaryEvidenceID string
    ResultID string
    RunID string
    CandidateID string
    EvalSuiteID string
    PromotionPolicyID string
    BaselinePackID string
    CandidatePackID string
    AuthorityState HotUpdateCanarySatisfactionAuthorityState
    SatisfactionState HotUpdateCanarySatisfactionState
    OwnerApprovalRequired bool
    State HotUpdateOwnerApprovalRequestState
    Reason string
    CreatedAt time.Time
    CreatedBy string
}
```

The only stored request state is:

```text
requested
```

Records are stored under:

```text
runtime_packs/hot_update_owner_approval_requests/<owner_approval_request_id>.json
```

with deterministic ID:

```text
hot-update-owner-approval-request-<canary_satisfaction_authority_id>
```

`CreateHotUpdateOwnerApprovalRequestFromCanarySatisfactionAuthority(...)` accepts only the canary satisfaction authority ID, `created_by`, and `created_at`. It loads the committed authority, requires:

- `authority_state=waiting_owner_approval`
- `satisfaction_state=waiting_owner_approval`
- `owner_approval_required=true`
- non-empty selected canary evidence ID

and cross-checks:

- canary satisfaction authority
- canary requirement
- selected passed canary evidence
- fresh canary satisfaction assessment
- candidate result
- improvement run
- improvement candidate
- frozen eval suite
- promotion policy
- baseline runtime pack
- candidate runtime pack
- freshly derived candidate promotion eligibility

Fresh canary satisfaction must still be:

```text
waiting_owner_approval
```

Fresh eligibility must still be:

```text
canary_and_owner_approval_required
```

Exact replay returns the existing record with `changed=false`; divergent duplicates fail closed. The request helper does not mutate the request after create/select and does not create downstream hot-update records.

V4-111 exposed this through:

```text
HOT_UPDATE_OWNER_APPROVAL_REQUEST_CREATE <job_id> <canary_satisfaction_authority_id>
```

with audit action:

```text
hot_update_owner_approval_request_create
```

That command creates or selects only the request record. It does not create owner approval grants, rejections, candidate promotion decisions, hot-update gates, outcomes, promotions, rollbacks, rollback-apply records, pointer changes, or reload/apply changes.

## Existing Runtime Approval Surface Assessment

Existing runtime approvals are useful for naming and UX, but they are not sufficient as canonical hot-update owner approval authority.

`ApprovalRequestRecord` and `ApprovalGrantRecord` are job/runtime-step scoped:

- `ApprovalRequestRecord` carries `request_id`, `job_id`, `step_id`, `requested_action`, `scope`, `state`, `requested_at`, `resolved_at`, and runtime/session metadata.
- `ApprovalGrantRecord` carries `grant_id`, `request_id`, `job_id`, `step_id`, `requested_action`, `scope`, `state`, `granted_at`, and runtime/session metadata.
- Committed request/grant records are stored under per-job approval directories and resolved by committed job runtime `applied_seq`.
- Hydration projects them back into `JobRuntimeState.ApprovalRequests` and `JobRuntimeState.ApprovalGrants`.
- Operator status surfaces them through `approval_request` and `approval_history`.

They do not carry stable hot-update authority refs:

- `owner_approval_request_id`
- `canary_satisfaction_authority_id`
- `canary_requirement_id`
- `selected_canary_evidence_id`
- `result_id`
- `run_id`
- `candidate_id`
- `eval_suite_id`
- `promotion_policy_id`
- `baseline_pack_id`
- `candidate_pack_id`

Using `ApprovalGrantRecord` as the canonical V4 hot-update owner approval would couple terminal update authority to a runtime step and would leave `HotUpdateGateRecord.approval_ref` without a stable, replay-checkable owner approval record. Runtime approvals should remain separate from the canary-gated hot-update owner approval ledger.

## Natural-Language Approval Assessment

The natural-language path intentionally binds only plain `yes` and `no`.

`ParsePlainApprovalDecision(...)` accepts `yes` and `no`. `ApplyNaturalApprovalDecision(...)` resolves exactly one pending runtime approval request from the active execution context or persisted runtime/control context. It then calls `ApplyApprovalDecision(...)` with `operator_reply`.

The path fails or declines to handle when:

- input is not plain `yes` or `no`
- no runtime approval request is pending
- multiple runtime approval requests are pending
- runtime state is not `waiting_user`
- the request does not match the active job and step

This path should not create hot-update owner approval grant/rejection authority now. Binding `yes` or `no` to terminal hot-update owner decisions before a durable decision registry and read model exist would conflate runtime approval UX with V4 hot-update authority and would risk ambiguous approval if both runtime and hot-update approval surfaces are present.

## Gate And Downstream Assessment

`HotUpdateGateRecord` already has fields that can carry future canary and owner approval refs:

```go
CanaryRef   string `json:"canary_ref,omitempty"`
ApprovalRef string `json:"approval_ref,omitempty"`
```

It also has canary-related state and decision values:

- `canarying`
- `apply_canary`
- `canary_reload`

The implemented specialized gate helper remains:

```go
CreateHotUpdateGateFromCandidatePromotionDecision(root, promotionDecisionID, createdBy string, createdAt time.Time)
```

It consumes `CandidatePromotionDecisionRecord`, which is still eligible-only:

```text
eligibility_state=eligible
decision=selected_for_promotion
```

That contract must remain intact. Owner approval grant/rejection must not broaden `CandidatePromotionDecisionRecord` to accept `canary_required` or `canary_and_owner_approval_required`, and it must not create a hot-update gate directly.

Hot-update outcome, promotion, rollback, rollback-apply, active runtime-pack pointer, last-known-good pointer, and reload-generation surfaces are downstream of gate execution. Terminal owner approval decisions should not create or mutate any of them.

## Decision Record Shape

V4-113 should add one immutable terminal decision registry, not separate grant and rejection registries.

Recommended record:

```go
type HotUpdateOwnerApprovalDecisionRecord struct {
    RecordVersion int
    OwnerApprovalDecisionID string
    OwnerApprovalRequestID string
    CanarySatisfactionAuthorityID string
    CanaryRequirementID string
    SelectedCanaryEvidenceID string
    ResultID string
    RunID string
    CandidateID string
    EvalSuiteID string
    PromotionPolicyID string
    BaselinePackID string
    CandidatePackID string
    RequestState HotUpdateOwnerApprovalRequestState
    AuthorityState HotUpdateCanarySatisfactionAuthorityState
    SatisfactionState HotUpdateCanarySatisfactionState
    OwnerApprovalRequired bool
    Decision HotUpdateOwnerApprovalDecision
    Reason string
    DecidedAt time.Time
    DecidedBy string
}
```

Recommended decision values:

```text
granted
rejected
```

A single terminal decision record is preferable because there must be at most one terminal owner decision for a request. Separate grant and rejection record families would require extra cross-family exclusion logic to prevent both:

```text
hot-update-owner-approval-grant-<owner_approval_request_id>
hot-update-owner-approval-rejection-<owner_approval_request_id>
```

from coexisting. A single record with a decision enum makes the exclusion the natural storage invariant.

## Deterministic IDs

Recommended terminal decision ID:

```text
hot-update-owner-approval-decision-<owner_approval_request_id>
```

This differs from a decision-suffixed ID such as:

```text
hot-update-owner-approval-decision-<owner_approval_request_id>-<decision>
```

because the decision-suffixed form would allow two deterministic IDs for one request. V4 needs one terminal owner approval decision per request. The decision should be data in the record, not a second namespace dimension.

If a future implementation needs a grant-only ref for `HotUpdateGateRecord.approval_ref`, it should point to the terminal decision record only when:

```text
decision=granted
```

A rejection decision is terminal evidence to block later gate creation, not an approval ref for a gate.

## Source Authority And Cross-Checks

The terminal decision should consume:

```text
owner_approval_request_id
```

as its source authority.

It should not consume only `canary_satisfaction_authority_id`, `canary_requirement_id`, selected evidence ID, candidate result ID, or a future hot-update gate ID. The request record is the durable handoff from canary satisfaction authority to owner approval. The terminal decision should copy source refs from the request and then cross-check the same upstream authority chain.

V4-113 should load and cross-check:

- `HotUpdateOwnerApprovalRequestRecord`
- `HotUpdateCanarySatisfactionAuthorityRecord`
- `HotUpdateCanaryRequirementRecord`
- selected `HotUpdateCanaryEvidenceRecord`
- fresh canary satisfaction assessment
- committed `CandidateResultRecord`
- linked `ImprovementRunRecord`
- linked `ImprovementCandidateRecord`
- frozen `EvalSuiteRecord`
- referenced `PromotionPolicyRecord`
- baseline `RuntimePackRecord`
- candidate `RuntimePackRecord`
- freshly derived `CandidatePromotionEligibilityStatus`

Required states at creation time:

- request `state=requested`
- authority `state=waiting_owner_approval`
- satisfaction `state=waiting_owner_approval`
- `owner_approval_required=true`
- selected evidence remains passed
- fresh canary satisfaction remains `waiting_owner_approval`
- fresh eligibility remains `canary_and_owner_approval_required`
- decision is either `granted` or `rejected`

The helper should reject caller-supplied source refs other than `owner_approval_request_id`, `decision`, `decided_by`, `decided_at`, and `reason`. All hot-update source refs should be copied from the loaded request.

## Replay And Duplicate Behavior

The terminal decision registry should be immutable and exact-replay stable:

- first write stores the normalized record and returns `changed=true`
- exact replay returns the existing record and `changed=false`
- exact replay is byte-stable
- a divergent duplicate for the same `owner_approval_decision_id` fails closed
- any second decision for the same `owner_approval_request_id` fails closed
- a different decision for a request that already has a terminal decision fails closed
- invalid existing deterministic decision records fail closed and remain visible in status
- stale request, stale authority, stale canary satisfaction, stale eligibility, missing selected evidence, or mismatched source refs fail closed

Grant and rejection are terminal. V4-113 should not add `expired` unless an explicit expiration policy source exists. If expiration is needed later, it should be a separately assessed policy-backed terminal decision or read-model readiness result.

## Status And Read Model

V4-113 should add a read-only status identity:

```json
"hot_update_owner_approval_decision_identity": { ... }
```

It should follow the existing V4 identity pattern:

- `configured`
- `not_configured`
- `invalid`

Minimum status fields:

- `state`
- `owner_approval_decision_id`
- `owner_approval_request_id`
- `canary_satisfaction_authority_id`
- `canary_requirement_id`
- `selected_canary_evidence_id`
- `result_id`
- `run_id`
- `candidate_id`
- `eval_suite_id`
- `promotion_policy_id`
- `baseline_pack_id`
- `candidate_pack_id`
- `request_state`
- `authority_state`
- `satisfaction_state`
- `owner_approval_required`
- `decision`
- `reason`
- `decided_at`
- `decided_by`
- `error`

Invalid records must be surfaced without hiding valid records. The read model must be read-only and must not mutate owner approval requests, canary records, candidate records, promotion policy records, runtime packs, gates, outcomes, promotions, rollbacks, rollback-apply records, active pointers, last-known-good pointers, or `reload_generation`.

Committed mission status snapshots and `STATUS <job_id>` should include the identity through the same composition path as the existing V4 hot-update identities. V4-113 should not add a separate status command.

## Assessment Answers

- Terminal owner approval decisions should be one immutable `HotUpdateOwnerApprovalDecisionRecord` with `decision=granted|rejected`, not separate grant/rejection registries.
- The deterministic ID should derive from `owner_approval_request_id` as `hot-update-owner-approval-decision-<owner_approval_request_id>`.
- The terminal decision record should consume `owner_approval_request_id` as source authority and copy all hot-update refs from the request.
- Grant/rejection authority should be created before any specialized hot-update gate helper. Future gate helpers should consume a granted terminal decision; they should not define owner approval.
- Natural-language `yes`/`no` should not create these records now. It should remain bound only to runtime approval requests until the durable decision surface and direct command are implemented and ambiguity rules are explicitly reassessed.
- Source records to load and cross-check are the owner approval request, canary satisfaction authority, canary requirement, selected passed canary evidence, fresh canary satisfaction assessment, candidate result, improvement run, improvement candidate, frozen eval suite, promotion policy, baseline runtime pack, candidate runtime pack, and fresh promotion eligibility.
- Stored decision states should be `granted` and `rejected`. `expired` should not be introduced without an explicit policy source. `invalid` should remain a read-model state for malformed or stale records.
- Exact replay should return `changed=false` and be byte-stable. Divergent duplicates and second terminal decisions for the same request should fail closed.
- Status exposure should be read-only `hot_update_owner_approval_decision_identity`, included in committed mission status snapshots and existing `STATUS <job_id>` readout composition.
- V4-113 should be missioncontrol registry/read-model first, not direct command first.

## Recommended V4-113 Slice

V4-113 should implement:

```text
Hot-Update Owner Approval Decision Registry And Read Model
```

Expected V4-113 implementation surface:

- `HotUpdateOwnerApprovalDecisionRecord`
- `HotUpdateOwnerApprovalDecision` values `granted` and `rejected`
- deterministic helper `HotUpdateOwnerApprovalDecisionIDFromRequest(ownerApprovalRequestID string) string`
- normalize, validate, store, load, and list helpers
- creation helper equivalent to:

```go
CreateHotUpdateOwnerApprovalDecisionFromRequest(root, ownerApprovalRequestID string, decision HotUpdateOwnerApprovalDecision, decidedBy string, decidedAt time.Time, reason string) (HotUpdateOwnerApprovalDecisionRecord, bool, error)
```

- read-only identity `hot_update_owner_approval_decision_identity`
- committed mission status snapshot and `STATUS <job_id>` readout composition, if consistent with the existing identity path

V4-113 should not add a direct command, `TaskState` wrapper, natural-language binding, hot-update gate helper, candidate promotion decision creation, outcome creation, promotion creation, rollback creation, rollback-apply creation, pointer mutation, LKG mutation, reload/apply change, or V4-114 work.

## Fail-Closed Requirements

V4-113 must fail closed for:

- missing or invalid store root
- missing `owner_approval_request_id`
- invalid or missing request record
- request state other than `requested`
- invalid or missing canary satisfaction authority
- authority state other than `waiting_owner_approval`
- `owner_approval_required=false`
- satisfaction state other than `waiting_owner_approval`
- missing selected canary evidence ID
- missing, non-passed, stale, or mismatched selected canary evidence
- missing or invalid canary requirement
- missing candidate result
- missing linked improvement run
- missing linked improvement candidate
- missing or non-linkable eval suite
- missing promotion policy
- missing baseline runtime pack
- missing candidate runtime pack
- fresh canary satisfaction away from `waiting_owner_approval`
- fresh eligibility away from `canary_and_owner_approval_required`
- copied ref mismatch across request, authority, requirement, evidence, assessment, candidate result, and fresh eligibility
- invalid decision value
- missing reason
- zero `decided_at`
- missing `decided_by`
- deterministic decision ID mismatch
- divergent duplicate decision record
- any second terminal decision for the same request
- existing invalid deterministic decision record

Owner approval must not substitute for passed canary evidence. A granted decision proves only that the owner approved the already canary-satisfied, owner-approval-required path captured by the request record.

## Invariants Preserved

This assessment preserves:

- no Go code changes
- no test changes
- no direct command
- no `TaskState` wrapper
- no owner approval grant creation
- no owner approval rejection creation
- no owner approval expiration record creation
- no natural-language owner approval binding
- no runtime `ApprovalRequestRecord` mutation
- no runtime `ApprovalGrantRecord` mutation
- no candidate promotion decision for canary-required states
- no broadening of `CandidatePromotionDecisionRecord`
- no hot-update gate for canary-required states
- no canary execution
- no canary evidence creation
- no outcome creation
- no promotion creation
- no rollback creation
- no rollback-apply creation
- no candidate result mutation
- no canary requirement mutation
- no canary evidence mutation
- no canary satisfaction authority mutation
- no owner approval request mutation
- no promotion policy mutation
- no runtime pack mutation
- no active runtime-pack pointer mutation
- no last-known-good pointer mutation
- no `reload_generation` mutation
- no pointer-switch behavior change
- no reload/apply behavior change
- no V4-113 implementation
