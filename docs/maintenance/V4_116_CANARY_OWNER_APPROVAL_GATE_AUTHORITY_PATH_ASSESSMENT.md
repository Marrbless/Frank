# V4-116 - Canary/Owner-Approval Gate Authority Path Assessment

## Scope

V4-116 assesses the smallest safe specialized hot-update gate authority path for canary-required candidate results after the V4 canary satisfaction authority, owner approval request, and owner approval decision chain exists.

This slice is assessment-only. It does not change Go code, tests, commands, TaskState wrappers, hot-update gates, candidate promotion decisions, canary authorities, owner approval records, outcomes, promotions, rollbacks, rollback-apply records, runtime-pack pointers, last-known-good pointers, `reload_generation`, pointer-switch behavior, reload/apply behavior, or V4-117 implementation.

## Live State Inspected

Docs inspected:

- `docs/FRANK_V4_SPEC.md`
- `docs/maintenance/V4_104_CANARY_SATISFACTION_AUTHORITY_PATH_ASSESSMENT.md`
- `docs/maintenance/V4_105_HOT_UPDATE_CANARY_SATISFACTION_AUTHORITY_REGISTRY_AFTER.md`
- `docs/maintenance/V4_107_HOT_UPDATE_CANARY_SATISFACTION_AUTHORITY_CONTROL_ENTRY_AFTER.md`
- `docs/maintenance/V4_108_HOT_UPDATE_OWNER_APPROVAL_AUTHORITY_PATH_ASSESSMENT.md`
- `docs/maintenance/V4_109_HOT_UPDATE_OWNER_APPROVAL_REQUEST_REGISTRY_AFTER.md`
- `docs/maintenance/V4_111_HOT_UPDATE_OWNER_APPROVAL_REQUEST_CONTROL_ENTRY_AFTER.md`
- `docs/maintenance/V4_112_HOT_UPDATE_OWNER_APPROVAL_GRANT_REJECTION_AUTHORITY_ASSESSMENT.md`
- `docs/maintenance/V4_113_HOT_UPDATE_OWNER_APPROVAL_DECISION_REGISTRY_AFTER.md`
- `docs/maintenance/V4_115_HOT_UPDATE_OWNER_APPROVAL_DECISION_CONTROL_ENTRY_AFTER.md`

Code inspected:

- `internal/missioncontrol/hot_update_gate_registry.go`
- `internal/missioncontrol/candidate_promotion_decision_registry.go`
- `internal/missioncontrol/hot_update_canary_requirement_registry.go`
- `internal/missioncontrol/hot_update_canary_evidence_registry.go`
- `internal/missioncontrol/hot_update_canary_satisfaction.go`
- `internal/missioncontrol/hot_update_canary_satisfaction_authority_registry.go`
- `internal/missioncontrol/hot_update_owner_approval_request_registry.go`
- `internal/missioncontrol/hot_update_owner_approval_decision_registry.go`
- `internal/missioncontrol/hot_update_outcome_registry.go`
- `internal/missioncontrol/promotion_registry.go`
- `internal/missioncontrol/rollback_registry.go`
- `internal/missioncontrol/rollback_apply_registry.go`
- `internal/missioncontrol/runtime_pack_registry.go`
- `internal/missioncontrol/status.go`
- `internal/agent/loop.go`
- `internal/agent/tools/taskstate.go`
- `internal/agent/tools/taskstate_readout.go`

## Current Gate Contract

`HotUpdateGateRecord` already has source-ref capacity for canary and owner approval authority:

```go
CanaryRef   string `json:"canary_ref,omitempty"`
ApprovalRef string `json:"approval_ref,omitempty"`
```

The gate also has canary-related schema values:

- state `canarying`
- decision `apply_canary`
- reload mode `canary_reload`

However, the implemented creation helper is still:

```go
CreateHotUpdateGateFromCandidatePromotionDecision(root, promotionDecisionID, createdBy string, createdAt time.Time) (HotUpdateGateRecord, bool, error)
```

That helper consumes a committed `CandidatePromotionDecisionRecord`, requires:

```text
decision=selected_for_promotion
eligibility_state=eligible
```

then re-runs candidate promotion eligibility and again requires `eligible`. It loads and cross-checks candidate result, improvement run, improvement candidate, frozen eval suite, promotion policy, baseline pack, candidate pack, active runtime-pack pointer, candidate rollback target pack, and optional last-known-good pointer. It creates a gate with:

```text
state=prepared
decision=keep_staged
reload_mode=<derived from candidate mutable surfaces>
```

It derives its gate ID from the promotion decision ID:

```text
hot-update-<promotion_decision_id>
```

This helper should remain unchanged. It is the no-canary/no-owner-approval eligible path.

## Candidate Promotion Decision Contract

`CandidatePromotionDecisionRecord` remains strictly eligible-only. Validation requires `eligibility_state=eligible`, and `CreateCandidatePromotionDecisionFromEligibleResult(...)` re-derives eligibility before writing. The existing gate helper also rejects any decision whose derived eligibility is not `eligible`.

Canary-required states are deliberately not eligible:

- `canary_required`
- `canary_and_owner_approval_required`

V4-117 must not broaden `CandidatePromotionDecisionRecord` and must not create candidate promotion decisions for canary-required states. The canary-required gate path needs a separate helper that consumes the canary/owner-approval authority chain directly.

## Existing Canary And Owner-Approval Authority Chain

The durable canary chain is:

1. `HotUpdateCanaryRequirementRecord`
2. `HotUpdateCanaryEvidenceRecord`
3. read-only `HotUpdateCanarySatisfactionAssessment`
4. `HotUpdateCanarySatisfactionAuthorityRecord`

`HotUpdateCanarySatisfactionAuthorityRecord` has two successful authority branches:

```text
state=authorized
owner_approval_required=false
satisfaction_state=satisfied
```

and:

```text
state=waiting_owner_approval
owner_approval_required=true
satisfaction_state=waiting_owner_approval
```

The durable owner approval chain is:

1. `HotUpdateOwnerApprovalRequestRecord`
2. `HotUpdateOwnerApprovalDecisionRecord`

The decision record supports:

```text
decision=granted
decision=rejected
```

A granted decision is the only owner-approval terminal authority that can unblock gate creation. A rejected decision is a terminal blocker and must not be treated as approval.

## Source Authority Decision

Canary-required hot-update gate creation should consume the durable canary satisfaction authority record, not raw canary evidence.

Reasons:

- raw evidence proves only an observation, not the current selected evidence or the policy-derived satisfaction state
- the authority record freezes selected passed evidence and copied source refs
- the helper can re-run fresh satisfaction and eligibility before gate creation
- `canary_ref` can point to one durable authority ID rather than an evidence ID that lacks owner-approval state

For the no-owner-approval branch, a canary satisfaction authority with:

```text
state=authorized
owner_approval_required=false
satisfaction_state=satisfied
```

is sufficient source authority for a specialized gate helper, provided fresh canary satisfaction still derives `satisfied`, selected evidence remains passed, fresh eligibility remains `canary_required`, the active runtime pack still matches the baseline pack, and rollback target validation passes.

For the owner-approval-required branch, a canary satisfaction authority with:

```text
state=waiting_owner_approval
owner_approval_required=true
satisfaction_state=waiting_owner_approval
```

is not sufficient by itself. The helper must require a `HotUpdateOwnerApprovalDecisionRecord` whose:

```text
decision=granted
owner_approval_required=true
request_state=requested
authority_state=waiting_owner_approval
satisfaction_state=waiting_owner_approval
```

and whose copied refs match the canary satisfaction authority. A `decision=rejected` record must fail closed as a terminal blocker.

## Intermediate Record Decision

V4-117 should not add a separate durable canary-gate proposal/authority record before creating `HotUpdateGateRecord`.

Reasons:

- `HotUpdateGateRecord` is already the durable gate authority record.
- It already has `canary_ref` and `approval_ref` fields.
- The current gate registry already implements replay, duplicate, source linkage, active pointer, and rollback target validation patterns.
- A new intermediate record would duplicate the frozen canary/approval refs that already exist in `HotUpdateCanarySatisfactionAuthorityRecord` and `HotUpdateOwnerApprovalDecisionRecord` without adding a needed state transition.

A read-only readiness helper is also not the smallest next slice by itself. Fresh readiness should be implemented inside the creation helper and tested there. A standalone read-only helper can be factored later if repeated command/read-model surfaces need it.

## Recommended Helper Shape

V4-117 should add a missioncontrol helper in the existing hot-update gate registry:

```go
CreateHotUpdateGateFromCanarySatisfactionAuthority(
    root string,
    canarySatisfactionAuthorityID string,
    ownerApprovalDecisionID string,
    createdBy string,
    createdAt time.Time,
) (HotUpdateGateRecord, bool, error)
```

`ownerApprovalDecisionID` should be required only when the canary satisfaction authority has `owner_approval_required=true`. It should be empty when `owner_approval_required=false`.

This helper shape handles both canary-required branches without broadening the eligible-only promotion decision path:

- no-owner-approval branch: canary satisfaction authority `authorized`
- owner-approval branch: canary satisfaction authority `waiting_owner_approval` plus owner approval decision `granted`

## Deterministic Gate ID

V4-117 should add a deterministic gate ID helper derived from the canary satisfaction authority ID, not from owner approval decision ID:

```go
HotUpdateGateIDFromCanarySatisfactionAuthority(canarySatisfactionAuthorityID string) string
```

Recommended public ID:

```text
hot-update-canary-gate-<sha256(canary_satisfaction_authority_id)>
```

The hash should be full-length hex or another repo-consistent fixed-length deterministic encoding.

Rationale:

- the canary satisfaction authority is the common source for both branches
- owner approval is optional, so deriving from owner approval decision ID would not work for `owner_approval_required=false`
- deriving from raw authority ID would create very long filenames because canary authority IDs include requirement and evidence IDs
- `StoreHotUpdateGatePath` currently stores gates as `<hot_update_id>.json`, so the gate ID should stay short enough for filesystem limits

The gate record should preserve the exact source authority through `canary_ref`; the shortened gate ID should not be the only source trace.

## Gate Record Fields

For V4-117, the helper should write only prepared gates:

```text
state=prepared
decision=keep_staged
```

The helper should not set `state=canarying`, should not set `decision=apply_canary`, and should not use `reload_mode=canary_reload` merely because the source was canary-required. The canary has already been represented by durable canary evidence and canary satisfaction authority. Existing phase, pointer-switch, reload/apply, outcome, promotion, rollback, and LKG flows should remain separate downstream paths.

`reload_mode` should be derived from the candidate runtime pack's mutable surfaces using the existing gate behavior:

- `skills` -> `skill_reload`
- `extensions` -> `extension_reload`
- otherwise `soft_reload`

The helper should set:

```text
canary_ref=<canary_satisfaction_authority_id>
```

For no-owner-approval branch:

```text
approval_ref=""
```

For owner-approved branch:

```text
approval_ref=<owner_approval_decision_id>
```

The helper should copy candidate and rollback fields from the same runtime-pack sources as `buildHotUpdateGateRecordFromCandidate(...)`:

- `candidate_pack_id`
- `previous_active_pack_id`
- `rollback_target_pack_id`
- `target_surfaces`
- `surface_classes`
- `compatibility_contract_ref`
- `prepared_at`
- `phase_updated_at`
- `phase_updated_by`

## Source Records To Load And Cross-Check

V4-117 should load and cross-check:

- `HotUpdateCanarySatisfactionAuthorityRecord`
- `HotUpdateCanaryRequirementRecord`
- selected `HotUpdateCanaryEvidenceRecord`
- fresh `AssessHotUpdateCanarySatisfaction(...)`
- committed `CandidateResultRecord`
- linked `ImprovementRunRecord`
- linked `ImprovementCandidateRecord`
- frozen `EvalSuiteRecord`
- referenced `PromotionPolicyRecord`
- baseline `RuntimePackRecord`
- candidate `RuntimePackRecord`
- freshly derived `CandidatePromotionEligibilityStatus`
- `ActiveRuntimePackPointer`
- candidate pack rollback target `RuntimePackRecord`
- `LastKnownGoodRuntimePackPointer` only with the same tolerance as the current gate helper: missing is allowed, invalid load errors fail closed
- `HotUpdateOwnerApprovalRequestRecord` when owner approval is required
- `HotUpdateOwnerApprovalDecisionRecord` when owner approval is required

For no-owner-approval branch, require:

```text
authority_state=authorized
owner_approval_required=false
satisfaction_state=satisfied
fresh satisfaction=satisfied
fresh eligibility=canary_required
selected evidence passed=true
active pointer active_pack_id=<baseline_pack_id>
```

For owner-approval-required branch, require:

```text
authority_state=waiting_owner_approval
owner_approval_required=true
satisfaction_state=waiting_owner_approval
owner approval decision=granted
fresh satisfaction=waiting_owner_approval
fresh eligibility=canary_and_owner_approval_required
selected evidence passed=true
active pointer active_pack_id=<baseline_pack_id>
```

Also require the owner approval decision and its request linkage to match the same:

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

## Replay And Duplicate Behavior

V4-117 should follow the current gate helper pattern:

- first write stores the normalized prepared gate and returns `changed=true`
- exact replay returns the existing gate and `changed=false`
- exact replay is byte-stable when the same normalized refs and `created_at` are used
- divergent duplicate for the deterministic gate ID fails closed
- existing deterministic gate with a different candidate pack fails closed
- an existing deterministic gate whose `canary_ref` or `approval_ref` does not match the source authority fails closed

If a direct command is added later, its TaskState wrapper should reuse an existing gate `PreparedAt` for replay stability. V4-117 should be helper-only, so it should accept caller-supplied `createdAt` and rely on exact replay tests at the missioncontrol layer.

## Fail-Closed Requirements

V4-117 must fail closed for:

- missing or invalid store root
- missing or invalid `canary_satisfaction_authority_id`
- missing `created_by`
- zero `created_at`
- missing canary satisfaction authority
- invalid canary satisfaction authority
- raw canary evidence used as the only source authority
- authority state other than `authorized` or `waiting_owner_approval`
- `authorized` authority with `owner_approval_required=true`
- `authorized` authority with satisfaction state other than `satisfied`
- `waiting_owner_approval` authority without a granted owner approval decision
- `waiting_owner_approval` authority with a rejected owner approval decision
- owner approval decision whose copied refs do not match the authority
- owner approval decision for a different request or authority
- owner approval decision state other than `granted`
- selected canary evidence missing, non-passed, stale, or mismatched
- fresh canary satisfaction away from the authority's required branch
- fresh eligibility away from `canary_required` or `canary_and_owner_approval_required` as appropriate
- missing candidate result
- missing linked improvement run
- missing linked improvement candidate
- missing or non-linkable eval suite
- missing promotion policy
- missing baseline runtime pack
- missing candidate runtime pack
- candidate pack missing `rollback_target_pack_id`
- missing rollback target runtime pack
- active runtime-pack pointer missing
- active pointer not equal to the baseline pack
- invalid last-known-good pointer when present
- divergent duplicate gate
- pre-existing gate for the same source authority with different source refs

V4-117 must not create outcomes, promotions, rollbacks, rollback-apply records, active pointer mutations, last-known-good pointer mutations, or `reload_generation` changes.

## Assessment Answers

- Canary-required gate creation should consume durable canary satisfaction authority, not raw canary evidence.
- No-owner-approval `state=authorized` canary satisfaction authority is sufficient for a specialized helper only when fresh satisfaction and fresh eligibility still match and rollback/active-pointer checks pass.
- Owner-approval-required branches must require a `HotUpdateOwnerApprovalDecisionRecord` with `decision=granted`.
- `decision=rejected` is a terminal blocker for gate creation.
- `CandidatePromotionDecisionRecord` must remain strictly eligible-only.
- `CreateHotUpdateGateFromCandidatePromotionDecision(...)` should remain unchanged.
- The deterministic canary-required gate ID should derive from canary satisfaction authority ID, using a fixed-length hash to avoid long gate filenames.
- `canary_ref` should point to the exact canary satisfaction authority ID.
- `approval_ref` should point to the exact owner approval decision ID when owner approval is required and granted; otherwise it should be empty.
- Initial gate state should be `prepared`.
- Initial gate decision should be `keep_staged`.
- Initial reload mode should be derived from the candidate pack's mutable surfaces, not forced to `canary_reload`.
- The helper should create only prepared gates; execution, reload, canarying, outcomes, promotions, rollback, LKG, pointer switch, and reload/apply stay downstream.
- V4-117 should be missioncontrol helper/registry first, not direct command first.

## Recommended V4-117 Slice

Recommend exactly one V4-117 implementation slice:

```text
V4-117 - Hot-Update Gate From Canary Satisfaction Authority Helper
```

V4-117 should add a missioncontrol helper in `internal/missioncontrol/hot_update_gate_registry.go` plus focused missioncontrol tests. It should create/select a prepared `HotUpdateGateRecord` from:

- authorized canary satisfaction authority with no owner approval requirement, or
- waiting-owner-approval canary satisfaction authority plus granted owner approval decision

It should populate `canary_ref` and `approval_ref`, derive a short deterministic gate ID from the canary satisfaction authority ID, validate all source refs, enforce fresh canary/eligibility state, require active pointer and rollback target safety, preserve exact replay, and fail closed on duplicates or rejected/missing owner approval.

V4-117 should not add a direct command, TaskState wrapper, natural-language binding, candidate promotion decision changes, outcome/promotion/rollback/LKG creation, active pointer mutation, `reload_generation` mutation, pointer-switch change, reload/apply change, or V4-118 work.
