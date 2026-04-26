# V4-105 Hot-Update Canary Satisfaction Authority Registry After

## Implemented Surface

V4-105 adds a durable missioncontrol authority record:

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
    SatisfactionState HotUpdateCanarySatisfactionState
    State HotUpdateCanarySatisfactionAuthorityState
    Reason string
    CreatedAt time.Time
    CreatedBy string
}
```

Authority states are:

- `authorized`
- `waiting_owner_approval`

## Storage Path

Records are stored under:

```text
runtime_packs/hot_update_canary_satisfaction_authorities/<canary_satisfaction_authority_id>.json
```

## Deterministic ID

The deterministic ID is:

```text
hot-update-canary-satisfaction-authority-<canary_requirement_id>-<selected_canary_evidence_id>
```

The helper is:

```go
HotUpdateCanarySatisfactionAuthorityIDFromRequirementEvidence(canaryRequirementID, selectedCanaryEvidenceID string) string
```

## Validation Behavior

Validation rejects:

- missing or invalid `record_version`
- missing `canary_satisfaction_authority_id`
- missing `canary_requirement_id`
- missing `selected_canary_evidence_id`
- missing copied source refs
- invalid `eligibility_state`
- invalid `satisfaction_state`
- invalid authority `state`
- missing `reason`
- zero `created_at`
- missing `created_by`
- deterministic ID mismatch

`eligibility_state` must be one of:

- `canary_required`
- `canary_and_owner_approval_required`

`satisfaction_state` must be one of:

- `satisfied`
- `waiting_owner_approval`

The authority state is tied to the satisfaction state:

- `satisfaction_state=satisfied` requires `state=authorized` and `owner_approval_required=false`
- `satisfaction_state=waiting_owner_approval` requires `state=waiting_owner_approval` and `owner_approval_required=true`

## Source Authority Records

Creation and linkage validation cross-check:

- `HotUpdateCanaryRequirementRecord`
- selected `HotUpdateCanaryEvidenceRecord`
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

The selected evidence must be `passed` and must match the requirement/source refs.

## Creation Helper Behavior

V4-105 adds:

```go
CreateHotUpdateCanarySatisfactionAuthorityFromRequirement(root, canaryRequirementID, createdBy string, createdAt time.Time) (HotUpdateCanarySatisfactionAuthorityRecord, bool, error)
```

The helper:

- validates the store root
- rejects missing `created_by`
- rejects zero `created_at`
- loads the committed canary requirement
- calls `AssessHotUpdateCanarySatisfaction(root, canaryRequirementID)`
- accepts only `satisfied` and `waiting_owner_approval`
- rejects `not_satisfied`, `failed`, `blocked`, `expired`, and `invalid`
- copies stable refs from the assessment/source records
- uses the assessment's selected passed evidence ID
- does not accept caller-supplied source refs beyond requirement ID, `created_by`, and `created_at`

For `satisfied`, it writes:

```text
state=authorized
owner_approval_required=false
satisfaction_state=satisfied
```

For `waiting_owner_approval`, it writes:

```text
state=waiting_owner_approval
owner_approval_required=true
satisfaction_state=waiting_owner_approval
```

## Idempotence And Duplicate Behavior

Storage behavior:

- first write stores the normalized record and returns `changed=true`
- exact replay returns `changed=false`
- exact replay is byte-stable
- divergent duplicate for the same `canary_satisfaction_authority_id` fails closed
- another authority record for the same requirement/evidence under a different ID fails closed
- list order is deterministic by filename

If newer passed evidence becomes the selected evidence, a new deterministic authority ID can be created for that newly selected evidence. If the newest selected valid evidence changes away from `passed`, creation fails closed. If fresh eligibility changes away from canary-required states, creation fails closed.

## Status And Read Model

Mission status now includes:

```json
"hot_update_canary_satisfaction_authority_identity": { ... }
```

The identity surfaces:

- `configured`
- `not_configured`
- `invalid`

Minimum status fields are:

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

Invalid authority records are surfaced without hiding valid records. The read model does not mutate records.

## Invariants Preserved

V4-105 preserves:

- no direct command
- no TaskState wrapper
- no owner approval request
- no owner approval proposal record
- no candidate promotion decision for canary-required states
- no broadening of `CandidatePromotionDecisionRecord`
- no hot-update gate for canary-required states
- no canary execution
- no canary evidence creation
- no outcome, promotion, rollback, rollback-apply, or last-known-good creation
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
- no V4-106 work
