# V4-078 Candidate Result Promotion Eligibility Helper After State

## Helper And Status Shape

V4-078 adds the read-only helper:

```go
EvaluateCandidateResultPromotionEligibility(root, resultID string) (CandidatePromotionEligibilityStatus, error)
```

It returns `CandidatePromotionEligibilityStatus` with:

- `state`
- `result_id`
- `run_id`
- `candidate_id`
- `eval_suite_id`
- `promotion_policy_id`
- `baseline_pack_id`
- `candidate_pack_id`
- `decision`
- `blocking_reasons`
- `canary_required`
- `owner_approval_required`
- `error`

Supported states are:

- `eligible`
- `canary_required`
- `owner_approval_required`
- `canary_and_owner_approval_required`
- `rejected`
- `unsupported_policy`
- `invalid`

## Source Authority

The helper loads the committed `CandidateResultRecord` through `LoadCandidateResultRecord`, preserving existing store-aware linkage to the improvement run, eval suite, runtime packs, optional hot-update gate, and optional promotion policy. It then loads the referenced `PromotionPolicyRecord`.

The helper does not accept caller-supplied scores, policies, eval suites, runs, or packs.

## Supported Policy Grammar

Only this narrow grammar is supported:

- `epsilon_rule`: `epsilon <= <non-negative-float>`
- `regression_rule`: `no_regression_flags` or `holdout_regression <= 0`
- `compatibility_rule`: `compatibility_score >= <float-between-0-and-1>`
- `resource_rule`: `resource_score >= <float-between-0-and-1>`

Malformed, unsupported, or ambiguous policy strings produce `unsupported_policy`.

## Score Presence Handling

Because candidate-result scores are plain `float64`, the helper performs a read-only raw JSON key presence check before typed evaluation. These keys are required:

- `baseline_score`
- `train_score`
- `holdout_score`
- `complexity_score`
- `compatibility_score`
- `resource_score`

Missing score keys produce `invalid`. Non-finite or otherwise invalid score JSON also produces `invalid` through the existing load/validation path.

## Derived Status Behavior

The helper requires:

- non-empty `promotion_policy_id`
- loaded referenced promotion policy
- decision `keep` for positive eligibility
- holdout improvement satisfying the epsilon rule
- no train-only promotion
- regression flags allowed by policy
- compatibility score satisfying policy
- resource score satisfying policy

If the policy requires canary, the helper returns `canary_required` rather than promoting. If the policy requires owner approval, it returns `owner_approval_required`. If both are required, it returns `canary_and_owner_approval_required`.

The helper does not implement complexity tie-breaks because the current schema has a single `complexity_score`, not separate baseline and candidate complexity evidence.

## Read Model Exposure

`OperatorCandidateResultStatus` now includes optional `promotion_eligibility`. Candidate-result identity status derives this value for each configured candidate result using the same store root and deterministic result filename order.

Invalid or unsupported eligibility status is surfaced on the result without adding commands.

## Fail-Closed Behavior

The helper returns fail-closed derived status for:

- missing candidate result
- invalid candidate result
- missing linked improvement run
- missing linked eval suite
- eval suite not frozen
- train and holdout corpus refs not distinct
- missing `promotion_policy_id`
- missing referenced promotion policy
- missing required score JSON keys
- non-finite or invalid score JSON
- decision other than `keep`
- train-only improvement
- regression flags under a rejecting rule
- unsupported or malformed epsilon rule
- unsupported or malformed regression rule
- unsupported or malformed compatibility rule
- unsupported or malformed resource rule

## Invariants Preserved

V4-078 does not create a `PromotionRecord`, create a durable eligibility record, mutate `CandidateResultRecord`, mutate `ImprovementRunRecord`, mutate `EvalSuiteRecord`, mutate `PromotionPolicyRecord`, mutate active runtime-pack pointers, mutate last-known-good pointers, mutate `reload_generation`, execute canaries, request owner approval, run evals, score candidates, add commands, add TaskState wrappers, or start V4-079 work.
