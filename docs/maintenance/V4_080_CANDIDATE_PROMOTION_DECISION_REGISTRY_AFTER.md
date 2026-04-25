# V4-080 Candidate Promotion Decision Registry After State

## Record Shape

V4-080 adds `CandidatePromotionDecisionRecord` with:

- `record_version`
- `promotion_decision_id`
- `result_id`
- `run_id`
- `candidate_id`
- `eval_suite_id`
- `promotion_policy_id`
- `baseline_pack_id`
- `candidate_pack_id`
- `eligibility_state`
- `decision`
- `reason`
- `created_at`
- `created_by`
- optional `notes`

The only decision value added in this slice is:

- `selected_for_promotion`

The deterministic helper identity is:

`candidate-promotion-decision-<result_id>`

## Storage Path

Candidate promotion decisions are stored under:

`runtime_packs/candidate_promotion_decisions/<promotion_decision_id>.json`

The registry provides normalize, validate, store, load, and list helpers. List output is deterministic by file name.

## Helper Behavior

V4-080 adds:

```go
CreateCandidatePromotionDecisionFromEligibleResult(root, resultID, createdBy string, createdAt time.Time) (CandidatePromotionDecisionRecord, bool, error)
```

The helper uses `EvaluateCandidateResultPromotionEligibility(root, resultID)` as the only eligibility gate. It creates a durable decision only when the derived state is `eligible`.

The helper rejects:

- missing candidate result
- invalid eligibility
- rejected eligibility
- unsupported policy eligibility
- `canary_required`
- `owner_approval_required`
- `canary_and_owner_approval_required`
- missing `created_by`
- zero `created_at`

## Idempotence And Duplicate Behavior

- first write stores the normalized record and returns `changed=true`
- exact helper replay with the same normalized record returns `changed=false`
- divergent duplicate with the same `promotion_decision_id` fails closed
- a different decision record for the same `result_id` fails closed

## Source Authority

Decision creation and record linkage rely on:

- committed `CandidateResultRecord`
- derived `CandidatePromotionEligibilityStatus`
- referenced `PromotionPolicyRecord`
- linked `ImprovementRunRecord`
- linked frozen `EvalSuiteRecord`
- linked baseline and candidate runtime packs

The record does not accept caller-supplied scores, policy fields, eval-suite refs, run refs, or pack refs beyond the committed result/eligibility linkage.

## Read Model

Operator status now includes `candidate_promotion_decision_identity` when decision records exist.

The read model is deterministic, read-only, and surfaces invalid decision records without hiding other registry state. `BuildCommittedMissionStatusSnapshot` includes the identity through the existing status composition path.

## Invariants Preserved

V4-080 does not create `PromotionRecord`, create hot-update gates, create hot-update outcomes, create rollbacks, mutate active runtime-pack pointer, mutate last-known-good pointer, mutate `reload_generation`, execute canaries, request owner approval, run evals, score candidates, add commands, add TaskState wrappers, or start V4-081.
