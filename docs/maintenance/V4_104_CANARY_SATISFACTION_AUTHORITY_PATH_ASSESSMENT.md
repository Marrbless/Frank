# V4-104 Canary Satisfaction Authority Path Assessment

## Scope

V4-104 assesses the smallest safe durable authority path after:

```go
AssessHotUpdateCanarySatisfaction(root, canaryRequirementID)
```

returns either:

- `satisfied`
- `waiting_owner_approval`

This is a docs-only slice. It does not change Go code, tests, commands, TaskState wrappers, canary requirements, canary evidence, canary satisfaction logic, candidate promotion decisions, hot-update gates, owner approval records, outcomes, promotions, rollbacks, rollback-apply records, runtime-pack pointers, last-known-good pointers, `reload_generation`, pointer-switch behavior, reload/apply behavior, or V4-105 implementation.

## Live State Inspected

Docs inspected:

- `docs/FRANK_V4_SPEC.md`
- `docs/maintenance/V4_102_CANARY_PASSED_PROMOTION_GATE_PATH_ASSESSMENT.md`
- `docs/maintenance/V4_103_HOT_UPDATE_CANARY_SATISFACTION_ASSESSMENT_AFTER.md`

Code surfaces inspected:

- `internal/missioncontrol/hot_update_canary_requirement_registry.go`
- `internal/missioncontrol/hot_update_canary_evidence_registry.go`
- `internal/missioncontrol/hot_update_canary_satisfaction.go`
- `internal/missioncontrol/candidate_promotion_decision_registry.go`
- `internal/missioncontrol/hot_update_gate_registry.go`
- `internal/missioncontrol/hot_update_outcome_registry.go`
- `internal/missioncontrol/promotion_registry.go`
- `internal/missioncontrol/rollback_registry.go`
- `internal/missioncontrol/rollback_apply_registry.go`
- `internal/missioncontrol/store_records.go`
- `internal/missioncontrol/status.go`
- `internal/missioncontrol/store_project.go`
- `internal/agent/loop.go`
- `internal/agent/tools/taskstate.go`

## Current Authority Chain

`HotUpdateCanaryRequirementRecord` is the durable record that a candidate result requires canary before promotion. It is created only for derived promotion eligibility states:

- `canary_required`
- `canary_and_owner_approval_required`

It stores stable refs copied from the candidate result and derived eligibility:

- `result_id`
- `run_id`
- `candidate_id`
- `eval_suite_id`
- `promotion_policy_id`
- `baseline_pack_id`
- `candidate_pack_id`

It also stores:

- `required_by_policy=true`
- `owner_approval_required=true` only for `canary_and_owner_approval_required`
- `state=required`

`HotUpdateCanaryEvidenceRecord` is append-only. It records operator-entered evidence states:

- `passed`
- `failed`
- `blocked`
- `expired`

Only `passed` records have `passed=true`. Evidence validates and cross-checks the canary requirement, candidate result, improvement run, improvement candidate, frozen eval suite, promotion policy, baseline runtime pack, candidate runtime pack, and freshly derived eligibility. Fresh eligibility must remain `canary_required` or `canary_and_owner_approval_required`.

`AssessHotUpdateCanarySatisfaction(...)` is read-only. It validates the requirement, selects the newest valid matching evidence by `observed_at` with `canary_evidence_id` as tie-breaker, and returns:

- `satisfied` when selected evidence is `passed` and `owner_approval_required=false`
- `waiting_owner_approval` when selected evidence is `passed` and `owner_approval_required=true`
- `not_satisfied`, `failed`, `blocked`, `expired`, or `invalid` otherwise

The status/read-model surface `hot_update_canary_satisfaction_identity` exposes these assessments but creates no authority record.

## Candidate Promotion Decision Contract

`CandidatePromotionDecisionRecord` should remain strictly eligible-only.

Reasons:

- validation requires `eligibility_state=eligible`
- `CreateCandidatePromotionDecisionFromEligibleResult(...)` re-runs `EvaluateCandidateResultPromotionEligibility(...)`
- linkage validation also re-runs eligibility and rejects anything other than `eligible`
- canary-required policies still derive `canary_required` or `canary_and_owner_approval_required` after passed evidence exists
- broadening this record would erase the difference between "policy did not require canary" and "policy required canary and evidence satisfied it"

Changing this contract would widen an existing authority record that is already used as the source for normal hot-update gate creation. That is not the smallest safe next step.

## Hot-Update Gate Contract

`CreateHotUpdateGateFromCandidatePromotionDecision(...)` should not consume raw canary evidence or the read-only satisfaction helper directly in the next slice.

Reasons:

- the helper is grounded in a committed `CandidatePromotionDecisionRecord`
- it requires selected-for-promotion and `eligible`
- it re-runs candidate result eligibility and requires `eligible`
- it loads and cross-checks run, candidate, frozen eval suite, promotion policy, baseline pack, candidate pack, active pointer, rollback target, and optional last-known-good pointer
- its deterministic gate ID is derived from the promotion decision ID

`HotUpdateGateRecord` already has canary-adjacent fields:

- `canary_ref`
- `approval_ref`
- state `canarying`
- decision `apply_canary`
- reload mode `canary_reload`

Those fields are schema capacity, not sufficient authority. A future specialized canary gate helper should consume a durable canary satisfaction authority record, not raw evidence or a transient read-only assessment.

## Owner Approval Surface

The existing approval records are job/runtime approval records:

- `ApprovalRequestRecord`
- `ApprovalGrantRecord`

They are stored under job-scoped runtime-control history, use sequence numbers, and hydrate into `JobRuntimeState` approval requests/grants. They are well-suited for mission-step approval mechanics, but they are not yet a durable canary-specific owner approval proposal tied to canary requirement, selected canary evidence, candidate result, and promotion policy refs.

For `waiting_owner_approval`, passed canary evidence proves only the canary branch. It must not create a gate and must not be treated as owner approval. Owner approval request/proposal creation should wait until there is an explicit canary satisfaction authority record that records the selected evidence and copied source refs.

## Outcome, Promotion, Rollback, And LKG Surfaces

Outcome, promotion, rollback, rollback-apply, and last-known-good surfaces are downstream execution/ledger surfaces:

- `HotUpdateOutcomeRecord` records terminal gate outcomes, including `canary_applied`, but depends on a gate.
- `PromotionRecord` depends on hot-update gate/outcome authority and active pack state.
- `RollbackRecord` and `RollbackApplyRecord` depend on promotion/hot-update failure or explicit rollback authority.
- Last-known-good and active runtime-pack pointer mutations belong to hot-update execution or rollback paths.

None of these should be used to represent canary satisfaction authority. V4-105 should not create outcomes, promotions, rollbacks, rollback-apply records, pointer mutations, or `reload_generation` changes.

## Assessment

The next missing durable object is not an owner approval request and not a hot-update gate. It is a canary-specific satisfaction authority record that freezes the currently selected passed evidence and the read-only satisfaction result.

This record should sit between:

```text
canary requirement + selected passed evidence + satisfaction assessment
```

and later:

```text
owner approval proposal/request, or specialized canary hot-update gate helper
```

It should be a missioncontrol registry/helper first, not a direct command. The command surface should come only after the durable record shape, validation, idempotence, status exposure, and fail-closed behavior are proven.

## Recommended V4-105 Slice

Recommend exactly one V4-105 implementation slice:

```text
V4-105 - Hot-Update Canary Satisfaction Authority Registry Skeleton
```

V4-105 should add a durable missioncontrol registry/read-model helper for a new canary-specific authority record. A repo-consistent name would be:

```go
type HotUpdateCanarySatisfactionAuthorityRecord struct {
    RecordVersion int
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
    EligibilityState string
    OwnerApprovalRequired bool
    SatisfactionState string
    State string
    Reason string
    CreatedAt time.Time
    CreatedBy string
}
```

Suggested deterministic ID:

```text
hot-update-canary-satisfaction-authority-<canary_requirement_id>-<selected_canary_evidence_id>
```

Suggested storage path:

```text
runtime_packs/hot_update_canary_satisfaction_authorities/<canary_satisfaction_authority_id>.json
```

Suggested helper:

```go
CreateHotUpdateCanarySatisfactionAuthorityFromRequirement(root, canaryRequirementID, createdBy string, createdAt time.Time) (HotUpdateCanarySatisfactionAuthorityRecord, bool, error)
```

The helper should call `AssessHotUpdateCanarySatisfaction(...)` and accept only:

- `satisfied`
- `waiting_owner_approval`

For `satisfied`, the durable record should represent canary satisfaction authority ready for a later gate-authority path:

```text
state=authorized
owner_approval_required=false
satisfaction_state=satisfied
```

For `waiting_owner_approval`, the durable record should freeze the canary-satisfied fact but must not authorize gate creation:

```text
state=waiting_owner_approval
owner_approval_required=true
satisfaction_state=waiting_owner_approval
```

This split lets a later owner-approval proposal/request consume a stable source authority instead of recomputing directly from raw evidence.

## Source Authority To Cross-Check

V4-105 should load and cross-check:

- `HotUpdateCanaryRequirementRecord`
- selected `HotUpdateCanaryEvidenceRecord`
- `AssessHotUpdateCanarySatisfaction(...)`
- committed `CandidateResultRecord`
- linked `ImprovementRunRecord`
- linked `ImprovementCandidateRecord`
- frozen `EvalSuiteRecord`
- referenced `PromotionPolicyRecord`
- baseline `RuntimePackRecord`
- candidate `RuntimePackRecord`
- freshly derived `CandidatePromotionEligibilityStatus`

Fresh eligibility must remain:

- `canary_required`, or
- `canary_and_owner_approval_required`

The copied refs must match across the requirement, selected evidence, assessment, candidate result, and derived eligibility.

## Idempotence And Duplicate Behavior

The authority record should be immutable once written.

V4-105 should implement:

- first write stores the normalized record and returns `changed=true`
- exact replay returns the existing record and `changed=false`
- exact replay is byte-stable
- divergent duplicate for the same `canary_satisfaction_authority_id` fails closed
- another authority record for the same `(canary_requirement_id, selected_canary_evidence_id)` under a different ID fails closed
- if newer evidence appears later, a new deterministic authority ID may be created only for the newly selected passed evidence
- if the selected latest valid evidence changes away from `passed`, creation/replay fails closed
- if fresh eligibility changes away from canary-required states, creation/replay fails closed

## Status And Read Model

V4-105 should add a read-only identity surface, for example:

```text
hot_update_canary_satisfaction_authority_identity
```

It should surface:

- `configured`
- `not_configured`
- `invalid`

Minimum status fields should include:

- `state`
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
- `eligibility_state`
- `owner_approval_required`
- `satisfaction_state`
- `authority_state`
- `reason`
- `created_at`
- `created_by`
- `error`

Invalid records should be surfaced without hiding valid records. The read model must not mutate any records.

## Future Slices After V4-105

After V4-105, the likely sequence is:

1. Assess or implement a control entry for creating the canary satisfaction authority record.
2. Add an owner approval proposal/request path that consumes `state=waiting_owner_approval` authority records.
3. Add a specialized canary hot-update gate helper that consumes either `state=authorized` authority records or a later owner-approved authority record.

The specialized gate helper should not consume raw evidence directly. It should consume durable authority that already selected and froze the evidence/source refs.

## Fail-Closed Requirements

The future authority helper should fail closed for:

- missing or invalid canary requirement
- missing selected canary evidence
- selected evidence not `passed`
- satisfaction state other than `satisfied` or `waiting_owner_approval`
- missing candidate result
- missing linked improvement run
- missing linked improvement candidate
- missing or unfrozen eval suite
- missing promotion policy
- missing baseline runtime pack
- missing candidate runtime pack
- stale derived eligibility away from `canary_required` or `canary_and_owner_approval_required`
- copied ref mismatch across requirement, evidence, assessment, candidate result, or derived eligibility
- divergent duplicate authority record
- existing authority record that does not validate or load
- owner approval required when attempting to treat the record as gate-authorized

## Non-Goals Preserved

V4-104 recommends no implementation in this slice and preserves:

- no Go code changes
- no test changes
- no direct commands
- no TaskState wrappers
- no canary-satisfied authority records
- no owner approval requests
- no owner approval proposal records
- no candidate promotion decisions for canary-required states
- no broadening of `CandidatePromotionDecisionRecord`
- no hot-update gates for canary-required states
- no canary execution
- no canary evidence creation
- no outcomes
- no promotions
- no rollbacks
- no rollback-apply records
- no candidate result mutation
- no canary requirement mutation
- no canary evidence mutation
- no promotion policy mutation
- no runtime pack mutation
- no active runtime-pack pointer mutation
- no last-known-good pointer mutation
- no `reload_generation` mutation
- no pointer-switch behavior change
- no reload/apply behavior change
- no V4-105 work
