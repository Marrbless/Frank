# V4-095 Hot-Update Canary Requirement Registry After State

## Record Shape

V4-095 adds `HotUpdateCanaryRequirementRecord`:

- `record_version`
- `canary_requirement_id`
- `result_id`
- `run_id`
- `candidate_id`
- `eval_suite_id`
- `promotion_policy_id`
- `baseline_pack_id`
- `candidate_pack_id`
- `eligibility_state`
- `required_by_policy`
- `owner_approval_required`
- `state`
- `reason`
- `created_at`
- `created_by`

The initial and only V4-095 requirement state is:

- `required`

## Storage Path And Deterministic ID

Records are stored under:

```text
runtime_packs/hot_update_canary_requirements/<canary_requirement_id>.json
```

The deterministic ID helper is:

```text
hot-update-canary-requirement-<result_id>
```

Validation rejects any record whose `canary_requirement_id` does not match the deterministic ID for its `result_id`.

## Validation Behavior

Validation rejects:

- missing or invalid `record_version`
- missing `canary_requirement_id`
- missing `result_id`
- missing `run_id`
- missing `candidate_id`
- missing `eval_suite_id`
- missing `promotion_policy_id`
- missing `baseline_pack_id`
- missing `candidate_pack_id`
- invalid `eligibility_state`
- eligibility states other than `canary_required` or `canary_and_owner_approval_required`
- `required_by_policy=false`
- any `state` other than `required`
- missing `reason`
- zero `created_at`
- missing `created_by`
- deterministic ID mismatch

## Source Authority Records

Creation and linkage validation rely on committed source authority:

- `CandidateResultRecord`
- `ImprovementRunRecord`
- `ImprovementCandidateRecord`
- frozen `EvalSuiteRecord`
- `PromotionPolicyRecord`
- baseline `RuntimePackRecord`
- candidate `RuntimePackRecord`
- freshly derived `CandidatePromotionEligibilityStatus`

The helper does not accept caller-supplied scores, policy fields, eval refs, run refs, candidate refs, or runtime-pack refs beyond the committed candidate result and derived eligibility linkage.

## Creation Helper Behavior

V4-095 adds:

```go
CreateHotUpdateCanaryRequirementFromCandidateResult(root, resultID, createdBy string, createdAt time.Time) (HotUpdateCanaryRequirementRecord, bool, error)
```

The helper:

- loads the committed candidate result
- re-runs `EvaluateCandidateResultPromotionEligibility(root, resultID)`
- requires derived state `canary_required` or `canary_and_owner_approval_required`
- loads and cross-checks the linked run, candidate, frozen eval suite, promotion policy, baseline pack, and candidate pack
- copies stable refs from the derived eligibility/source records
- sets `required_by_policy=true`
- sets `owner_approval_required=true` only for `canary_and_owner_approval_required`
- sets `state=required`
- sets reason to `candidate result requires canary before promotion`
- uses caller-supplied `created_at` and `created_by`
- rejects missing `created_by`
- rejects zero `created_at`

The helper rejects `eligible`, `owner_approval_required`, `rejected`, `unsupported_policy`, and `invalid` derived eligibility states.

## Idempotence And Duplicate Behavior

- first write stores the normalized record and returns `changed=true`
- exact replay returns `changed=false`
- exact replay is byte-stable
- divergent duplicate for the same `canary_requirement_id` fails closed
- a second requirement for the same `result_id` fails closed unless it is the exact same normalized record
- list order is deterministic by file name
- derived eligibility changing away from canary-required fails closed

## Status / Read Model

V4-095 adds the read-only operator identity surface:

```text
hot_update_canary_requirement_identity
```

Minimum status fields are surfaced:

- `state`
- `canary_requirement_id`
- `result_id`
- `run_id`
- `candidate_id`
- `eval_suite_id`
- `promotion_policy_id`
- `baseline_pack_id`
- `candidate_pack_id`
- `eligibility_state`
- `required_by_policy`
- `owner_approval_required`
- `requirement_state`
- `reason`
- `created_at`
- `created_by`
- `error`

The read model surfaces:

- `not_configured` when no requirement records exist
- `configured` for valid requirement records
- `invalid` records without hiding other valid records

`BuildCommittedMissionStatusSnapshot` includes the identity through the existing status composition path. The read model does not mutate records.

## Invariants Preserved

V4-095 does not:

- execute canaries
- create canary evidence
- request owner approval
- create owner approval proposal records
- create candidate promotion decisions for canary-required states
- create hot-update gates for canary-required states
- mutate `CandidateResultRecord`
- mutate `PromotionPolicyRecord`
- mutate runtime packs
- mutate active runtime-pack pointer
- mutate last-known-good pointer
- mutate `reload_generation`
- create outcomes
- create promotions
- create rollbacks
- create rollback-apply records
- add direct commands
- add TaskState wrappers
- change pointer-switch or reload/apply behavior
- broaden promotion policy grammar
- implement V4-096
