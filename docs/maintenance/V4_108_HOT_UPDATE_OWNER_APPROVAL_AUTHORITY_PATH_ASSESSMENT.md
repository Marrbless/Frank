# V4-108 Hot-Update Owner Approval Authority Path Assessment

## Scope

V4-108 assesses the smallest safe durable owner-approval authority path after a hot-update canary satisfaction authority exists with:

```text
state=waiting_owner_approval
owner_approval_required=true
```

This is a docs-only slice. It does not change Go code, tests, direct commands, TaskState wrappers, canary satisfaction authority records, owner approval records, candidate promotion decisions, hot-update gates, outcomes, promotions, rollbacks, rollback-apply records, runtime-pack pointers, last-known-good pointers, `reload_generation`, pointer-switch behavior, reload/apply behavior, or V4-109 implementation.

## Live State Inspected

Docs inspected:

- `docs/FRANK_V4_SPEC.md`
- `docs/maintenance/V4_104_CANARY_SATISFACTION_AUTHORITY_PATH_ASSESSMENT.md`
- `docs/maintenance/V4_105_HOT_UPDATE_CANARY_SATISFACTION_AUTHORITY_REGISTRY_AFTER.md`
- `docs/maintenance/V4_106_HOT_UPDATE_CANARY_SATISFACTION_AUTHORITY_CONTROL_ENTRY_ASSESSMENT.md`
- `docs/maintenance/V4_107_HOT_UPDATE_CANARY_SATISFACTION_AUTHORITY_CONTROL_ENTRY_AFTER.md`

Code surfaces inspected:

- `internal/missioncontrol/approval.go`
- `internal/missioncontrol/store_records.go`
- `internal/missioncontrol/store_types.go`
- `internal/missioncontrol/store_mutate.go`
- `internal/missioncontrol/store_hydrate.go`
- `internal/missioncontrol/status.go`
- `internal/missioncontrol/store_project.go`
- `internal/missioncontrol/hot_update_canary_satisfaction_authority_registry.go`
- `internal/missioncontrol/hot_update_canary_satisfaction.go`
- `internal/missioncontrol/hot_update_canary_requirement_registry.go`
- `internal/missioncontrol/hot_update_canary_evidence_registry.go`
- `internal/missioncontrol/candidate_promotion_decision_registry.go`
- `internal/missioncontrol/hot_update_gate_registry.go`
- `internal/missioncontrol/hot_update_outcome_registry.go`
- `internal/missioncontrol/promotion_registry.go`
- `internal/missioncontrol/rollback_registry.go`
- `internal/missioncontrol/rollback_apply_registry.go`
- `internal/agent/loop.go`
- `internal/agent/tools/taskstate.go`
- `internal/agent/tools/taskstate_readout.go`

## Current Authority Path

V4-107 exposes:

```text
HOT_UPDATE_CANARY_SATISFACTION_AUTHORITY_CREATE <job_id> <canary_requirement_id>
```

through:

```go
CreateHotUpdateCanarySatisfactionAuthorityFromRequirement(jobID string, canaryRequirementID string)
```

The wrapper validates active or persisted job context, assesses current canary satisfaction, reuses an existing authority `CreatedAt` for exact replay, emits audit action `hot_update_canary_satisfaction_authority_create`, and calls:

```go
missioncontrol.CreateHotUpdateCanarySatisfactionAuthorityFromRequirement(root, canaryRequirementID, "operator", createdAt)
```

For owner-approval-required canary requirements with selected passed canary evidence, the helper creates or selects:

```text
authority_state=waiting_owner_approval
owner_approval_required=true
satisfaction_state=waiting_owner_approval
```

That record proves canary satisfaction plus an unresolved owner approval requirement. It does not create owner approval requests, candidate promotion decisions, hot-update gates, outcomes, promotions, rollbacks, rollback-apply records, pointer mutations, or reload/apply changes.

## Existing Approval Surface Assessment

Existing `ApprovalRequestRecord` and `ApprovalGrantRecord` are not sufficient as the durable owner-approval authority for canary-gated hot updates.

They are job/runtime-step scoped:

- stored under `jobs/<job_id>/approvals/requests` and `jobs/<job_id>/approvals/grants`
- projected from `JobRuntimeState.ApprovalRequests` and `JobRuntimeState.ApprovalGrants`
- bound by `job_id`, `step_id`, `requested_action`, and `scope`
- hydrated back into the active job runtime and runtime-control status path
- reusable across mission steps when approval scope allows it

They do not carry stable linkage to:

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

Using them directly would make owner approval depend on a runtime-step binding instead of the frozen canary satisfaction authority. That would be weaker than the V4-105/V4-107 source-ref copying and replay model, and would leave hot-update gate creation with no stable authority record to validate.

Existing runtime approval notifications, natural-language approval handling, and `approval_history` status should remain available for ordinary mission step approvals, but they should not be reused as the canonical canary-gated hot-update owner approval ledger.

## Owner Approval Binding Decision

Owner approval should be tied primarily to:

```text
canary_satisfaction_authority_id
```

The owner approval record should copy the supporting refs from the authority record:

- `canary_requirement_id`
- `selected_canary_evidence_id`
- `result_id`
- `run_id`
- `candidate_id`
- `eval_suite_id`
- `promotion_policy_id`
- `baseline_pack_id`
- `candidate_pack_id`

It should not be tied primarily to a future hot-update gate. For `waiting_owner_approval`, owner approval authority must exist before any specialized gate helper can safely create a gate. A gate may later copy an `approval_ref`, but the gate should consume approval authority, not define it.

It should not use canary evidence alone as the source authority. Passed canary evidence proves the canary branch only; it does not prove owner approval. It should also not use `candidate_result_id` alone, because the candidate result can be associated with multiple policy-derived authority branches.

## Registry Shape Decision

V4 should add a dedicated missioncontrol registry/read-model for hot-update owner approval requests before adding a direct command.

Recommended request record:

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
    AuthorityState string
    SatisfactionState string
    OwnerApprovalRequired bool
    State string
    Reason string
    CreatedAt time.Time
    CreatedBy string
}
```

The minimum stored state for V4-109 should be:

```text
requested
```

Future granted/rejected/expired handling should be ledger-style, not current-state mutation on the request record:

- requested: immutable `HotUpdateOwnerApprovalRequestRecord`
- granted: future immutable grant/decision record linked to request ID
- rejected: future immutable grant/decision record linked to request ID
- expired: read-model/readiness result derived from policy time and record age, or a future immutable expiration decision if the repo needs a durable terminal record
- invalid: read-model status for malformed or stale records, not a normal stored success state

This keeps the request record byte-stable and leaves terminal owner decisions to a separate authority surface. It also avoids silently turning the existing runtime approval current-state model into a hot-update gate authority.

If V4 later needs an expiration deadline, it should be introduced with an explicit policy source and copied into the request record. V4-109 should not invent an expiration policy if live promotion/hot-update policy records do not provide one.

## Assessment Answers

- Existing `ApprovalRequestRecord` and `ApprovalGrantRecord` are too job/runtime-step scoped to be the canonical owner-approval authority for canary-gated hot updates. They may remain useful for ordinary runtime approvals, notifications, and `approval_history`, but they lack stable canary satisfaction authority linkage.
- Owner approval should bind to `canary_satisfaction_authority_id` and copy the authority's canary requirement, selected canary evidence, candidate result, run, candidate, eval suite, promotion policy, baseline pack, and candidate pack refs.
- Owner approval request/proposal authority should be created before any specialized hot-update gate helper for `waiting_owner_approval`; a future gate should consume approval authority, not define it.
- Owner approval should be ledger-style. The request record should be immutable, and later grant/reject/expiration decisions should be separate immutable records or explicit read-model derivations rather than current-state mutation of the request.
- The minimal V4-109 request fields are the deterministic request ID, source canary satisfaction authority ID, all copied source refs, authority/satisfaction/owner-approval state, request state, reason, `created_at`, and `created_by`.
- Stored V4-109 state should be only `requested`. Future `granted` and `rejected` should be separate decision records; `expired` should require an explicit policy basis before becoming durable; `invalid` should be read-model state for malformed or stale records.
- The request ID should be `hot-update-owner-approval-request-<canary_satisfaction_authority_id>`. Future terminal decisions should reserve deterministic IDs derived from `owner_approval_request_id`.
- Exact replay should return the existing normalized request with `changed=false`; divergent deterministic duplicates, invalid existing records, and stale source authority should fail closed.
- Status exposure should be a read-only missioncontrol identity, `hot_update_owner_approval_request_identity`, included through the existing `STATUS <job_id>` readout path.
- V4-109 should be missioncontrol registry/read-model first, not direct command first. Another docs-only assessment is not needed before implementing that skeleton.

## Deterministic IDs

Recommended request ID:

```text
hot-update-owner-approval-request-<canary_satisfaction_authority_id>
```

Recommended future grant/decision IDs:

```text
hot-update-owner-approval-grant-<owner_approval_request_id>
hot-update-owner-approval-rejection-<owner_approval_request_id>
```

V4-109 should implement only the request ID. Future grant/rejection IDs should be reserved in the assessment so a later slice can remain deterministic without mutating the request record.

## Replay And Duplicate Behavior

The V4-109 request helper should be exact-replay stable:

- load and validate the canary satisfaction authority
- require `state=waiting_owner_approval`
- require `owner_approval_required=true`
- require `satisfaction_state=waiting_owner_approval`
- copy all authority refs into the request record
- derive `owner_approval_request_id` from the authority ID
- if the same normalized request already exists, return it with `changed=false`
- if a deterministic request exists with divergent content, fail closed
- if the deterministic request cannot be loaded or validated, fail closed
- if the authority record cannot be loaded or validated, fail closed

The helper should not mutate the canary satisfaction authority, canary requirement, canary evidence, candidate result, promotion policy, runtime packs, active pointer, last-known-good pointer, or reload generation.

## Status And Read Model

V4-109 should add missioncontrol read-model exposure alongside the existing status identities, likely:

```json
"hot_update_owner_approval_request_identity": { ... }
```

The identity should surface:

- `configured`
- `not_configured`
- `invalid`

Minimum request status fields:

- `state`
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
- `authority_state`
- `satisfaction_state`
- `owner_approval_required`
- `request_state`
- `reason`
- `created_at`
- `created_by`
- `error`

Invalid records should be surfaced without hiding valid records. The read model must remain read-only.

`STATUS <job_id>` should include the identity once the readout path is wired, but V4-109 should not add a separate status command.

## Gate And Downstream Assessment

`CandidatePromotionDecisionRecord` intentionally accepts only:

```text
eligibility_state=eligible
decision=selected_for_promotion
```

The existing `CreateCandidatePromotionDecisionFromEligibleResult(...)` and `CreateHotUpdateGateFromCandidatePromotionDecision(...)` path is therefore for the no-canary/no-owner-approval eligible branch. It must not be broadened to admit `canary_required` or `canary_and_owner_approval_required`.

`HotUpdateGateRecord` already has schema capacity for:

- `canary_ref`
- `approval_ref`
- `state=canarying`
- `decision=apply_canary`
- `reload_mode=canary_reload`

Those fields are not enough by themselves. They are refs on a gate record, not durable proof that canary evidence and owner approval authority were both satisfied. A future specialized gate helper can consume canary satisfaction authority plus owner approval authority and populate `canary_ref` and `approval_ref`, but V4-109 should not create gates.

No outcome, promotion, rollback, rollback-apply, active pointer, last-known-good pointer, or reload/apply path should be introduced before the explicit owner approval authority exists.

## Fail-Closed Requirements

V4-109 must fail closed for:

- missing mission store root
- invalid `canary_satisfaction_authority_id`
- missing canary satisfaction authority
- invalid canary satisfaction authority
- authority state other than `waiting_owner_approval`
- `owner_approval_required=false`
- satisfaction state other than `waiting_owner_approval`
- missing selected canary evidence ID
- selected evidence not linked to the authority
- selected evidence no longer validating as passed evidence
- missing canary requirement
- invalid canary requirement
- stale or mismatched canary requirement, evidence, assessment, candidate result, run, candidate, eval suite, promotion policy, baseline pack, or candidate pack refs
- fresh canary satisfaction assessment no longer being `waiting_owner_approval`
- fresh eligibility changing away from `canary_and_owner_approval_required`
- existing deterministic request record that fails to load or validate
- divergent duplicate request record
- any attempt to treat owner approval request creation as an approval grant
- any attempt to create candidate promotion decisions, hot-update gates, outcomes, promotions, rollbacks, rollback-apply records, pointer mutations, last-known-good changes, `reload_generation` changes, or reload/apply behavior changes

## Recommended V4-109 Slice

Recommend exactly one V4-109 implementation slice:

```text
V4-109 - Hot-Update Owner Approval Request Registry And Read Model
```

V4-109 should implement only a missioncontrol registry/read-model skeleton for durable owner approval requests.

Expected implementation surface:

- `HotUpdateOwnerApprovalRequestRecord`
- deterministic request ID helper
- normalize/validate helpers
- store/load/list helpers under a runtime-pack hot-update owner approval request directory
- create/select helper from `canary_satisfaction_authority_id`
- linkage validation against canary satisfaction authority and copied source refs
- exact replay returns `changed=false`
- divergent duplicates fail closed
- read-model identity loaded by `STATUS <job_id>`
- focused missioncontrol and status tests
- before/after maintenance docs

V4-109 should not add a direct command. A direct command should wait until the durable request registry and read model prove the record shape, replay behavior, and invalid-record exposure.

V4-109 should not add grant/reject commands, natural-language owner approval binding, specialized hot-update gate creation, candidate promotion decision changes, outcome/promotion/rollback/LKG creation, pointer mutation, reload/apply behavior, or V4-110 work.

## Invariants Preserved

V4-108 preserves:

- no Go code changes
- no test changes
- no direct commands
- no TaskState wrappers
- no owner approval request creation
- no owner approval grant creation
- no owner approval proposal record creation
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
- no promotion policy mutation
- no runtime pack mutation
- no active runtime-pack pointer mutation
- no last-known-good pointer mutation
- no `reload_generation` mutation
- no pointer-switch behavior change
- no reload/apply behavior change
- no V4-109 implementation
