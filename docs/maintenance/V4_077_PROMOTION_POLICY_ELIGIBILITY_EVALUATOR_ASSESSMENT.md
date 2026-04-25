# V4-077 Promotion Policy Eligibility Evaluator Assessment

## Facts

- This is a docs-only assessment slice for the future V4 promotion-policy eligibility evaluator.
- The live starting point is V4-076, where `CandidateResultRecord` gained optional `promotion_policy_id` and store-aware validation that the referenced `PromotionPolicyRecord` exists when declared.
- Current candidate-result storage already validates linkage to:
  - `ImprovementRunRecord` through `run_id`
  - `EvalSuiteRecord` through `eval_suite_id`
  - baseline runtime pack through `baseline_pack_id`
  - candidate runtime pack through `candidate_pack_id`
  - optional hot-update gate through `hot_update_id`
  - optional promotion policy through `promotion_policy_id`
- `EvalSuiteRecord` is frozen by validation through `frozen_for_run=true` and train/holdout corpus refs must be distinct.
- `PromotionPolicyRecord` exists as a durable registry record with string rule fields. Those rule strings are not yet parsed or evaluated.

## Existing Source Authority

The future evaluator should derive eligibility only from committed store records:

- committed `CandidateResultRecord`
- linked `ImprovementRunRecord`
- linked frozen `EvalSuiteRecord`
- referenced `PromotionPolicyRecord`
- linked baseline and candidate runtime pack records only as linkage evidence, not as mutable inputs

The evaluator should not accept caller-supplied score, policy, eval-suite, or run overrides. The store root and candidate result id are enough to find all authority records if the result is well linked.

## Recommended Future Helper

Recommended V4-078 helper:

```go
func EvaluateCandidateResultPromotionEligibility(root, resultID string) (CandidatePromotionEligibilityStatus, error)
```

Recommended status shape:

```go
type CandidatePromotionEligibilityStatus struct {
	State             string   `json:"state"`
	ResultID          string   `json:"result_id,omitempty"`
	RunID             string   `json:"run_id,omitempty"`
	CandidateID       string   `json:"candidate_id,omitempty"`
	EvalSuiteID       string   `json:"eval_suite_id,omitempty"`
	PromotionPolicyID string   `json:"promotion_policy_id,omitempty"`
	BaselinePackID    string   `json:"baseline_pack_id,omitempty"`
	CandidatePackID   string   `json:"candidate_pack_id,omitempty"`
	Decision          string   `json:"decision,omitempty"`
	BlockingReasons   []string `json:"blocking_reasons,omitempty"`
	CanaryRequired    bool     `json:"canary_required,omitempty"`
	OwnerApprovalRequired bool `json:"owner_approval_required,omitempty"`
	Error             string   `json:"error,omitempty"`
}
```

Recommended states:

- `eligible`
- `canary_required`
- `owner_approval_required`
- `canary_and_owner_approval_required`
- `rejected`
- `unsupported_policy`
- `invalid`

The helper should be read-only and deterministic. It should not create a `PromotionRecord`, mutate runtime-pack pointers, mutate last-known-good state, write canary evidence, request owner approval, or run evaluations.

## Durable Record vs Derived Status

V4-078 should create a derived read-only status, not a durable eligibility record.

Reasoning:

- The repo already has immutable candidate results and promotion policies.
- Promotion eligibility is a deterministic read over committed records at this stage.
- Durable eligibility records would create another append-only artifact before promotion/canary/owner-approval semantics are implemented.
- A derived status can be exposed in the candidate-result read model without inventing replay or duplicate-write behavior.

If a later slice adds a durable eligibility ledger, it should be immutable by eligibility id, exact replay should be idempotent, and divergent duplicate writes should fail closed like candidate results and improvement runs.

## Required Inputs

The evaluator should require:

- non-empty `promotion_policy_id`
- finite `baseline_score`
- finite `train_score`
- finite `holdout_score`
- finite `complexity_score`
- finite `compatibility_score`
- finite `resource_score`
- `regression_flags`
- `decision`
- coherent candidate-result to improvement-run linkage
- coherent improvement-run to frozen eval-suite linkage
- coherent baseline and candidate pack linkage

Current score fields are plain `float64`, so a missing JSON score currently decodes to `0`. V4-078 should either add a small raw-JSON presence check for required score keys before typed evaluation or explicitly document that it cannot distinguish a missing score from a literal zero. The safer option is a read-only raw-presence check in the evaluator helper.

## Eligibility Rules To Implement First

The smallest safe V4-078 rule set should be deliberately conservative:

1. Load the candidate result through `LoadCandidateResultRecord` so existing candidate/run/eval-suite/policy/runtime-pack linkage checks run first.
2. Require `promotion_policy_id` on the candidate result.
3. Load the referenced `PromotionPolicyRecord`.
4. Require the linked eval suite to be frozen and train/holdout corpus refs to remain distinct.
5. Require score fields to be present and finite.
6. Require the result decision to be `keep` before eligibility can be positive.
7. Reject train-only wins. If `train_score` improves but `holdout_score` does not satisfy the policy threshold, the status is `rejected`.
8. If `requires_holdout_pass=true`, require holdout to pass the epsilon rule.
9. Apply `regression_rule` to `regression_flags`; for a rejecting rule, non-empty regression flags other than an explicit neutral sentinel such as `none` should reject.
10. Apply `compatibility_rule` and `resource_rule` only if V4-078 implements a narrow documented grammar. Unsupported rule strings must produce `unsupported_policy`, not silent eligibility.
11. Treat `requires_canary=true` as a gate status (`canary_required`), not as promotion eligibility.
12. Treat `requires_owner_approval=true` as a gate status (`owner_approval_required`), not as promotion eligibility.

Train-only wins are insufficient for general promotion. A future exception for cosmetic or explicitly non-holdout policy classes needs an explicit policy field or grammar; it should not be inferred from `requires_holdout_pass=false` alone in V4-078.

## Rule Grammar Recommendation

Current policy rules are strings. V4-078 should not try to evaluate arbitrary prose.

Recommended minimal supported grammar:

- `epsilon_rule`: `epsilon <= <non-negative-float>`
- `regression_rule`: `no_regression_flags` or `holdout_regression <= 0`
- `compatibility_rule`: `compatibility_score >= <float-between-0-and-1>`
- `resource_rule`: `resource_score >= <float-between-0-and-1>`

Unsupported, malformed, or semantically ambiguous rules should fail closed with `unsupported_policy`.

Complexity should only affect eligibility if a supported future policy explicitly permits a holdout tie with lower complexity. The current result schema has one `complexity_score`, not separate baseline and candidate complexity scores, so V4-078 should not implement tie-with-lower-complexity unless the repo first adds enough evidence to compare complexity safely.

## Fail-Closed Behavior

The evaluator should fail closed for:

- missing candidate result
- invalid candidate result
- missing linked improvement run
- linked run mismatch
- missing linked eval suite
- eval suite not frozen
- train and holdout corpus refs not distinct
- missing promotion policy id
- missing referenced promotion policy
- missing required score JSON fields
- non-finite scores
- missing holdout score
- result decision other than `keep`
- train-only improvement
- regression flags under a rejecting regression rule
- unsupported or malformed epsilon rule
- unsupported or malformed regression rule
- unsupported or malformed compatibility rule
- unsupported or malformed resource rule
- policy requiring canary without canary evidence, which should return `canary_required`
- policy requiring owner approval without approval evidence, which should return `owner_approval_required`

Store-layer errors should remain repo-style errors. V4 `E_*` rejection codes should only be used if the evaluator is later wired into an existing `ValidationError` path.

## Read-Model Expectations

If V4-078 implements the derived status, the least surprising read-model exposure is to extend candidate-result identity status:

- add optional `promotion_eligibility` to `OperatorCandidateResultStatus`
- compute it from the same store root used by `LoadOperatorCandidateResultIdentityStatus`
- keep output deterministic by result filename order
- surface invalid or unsupported policy status without hiding the candidate result itself

No new command is needed. Operator status can show whether a candidate result is eligible, rejected, requires canary, requires owner approval, or is blocked by unsupported policy grammar.

## Tests Recommended For V4-078

Focused tests should prove:

- missing candidate result returns a fail-closed error/status
- missing `promotion_policy_id` rejects
- missing referenced policy rejects
- linked run/eval-suite mismatches still reject through existing loaders
- unsupported policy rule strings produce `unsupported_policy`
- finite scores with a valid holdout improvement and no regression flags return `eligible` when no canary or owner approval is required
- train-only improvement returns `rejected`
- regression flags under a rejecting rule return `rejected`
- `requires_canary=true` returns `canary_required`
- `requires_owner_approval=true` returns `owner_approval_required`
- both canary and owner approval return `canary_and_owner_approval_required`
- candidate-result identity read model exposes derived eligibility deterministically if read-model exposure is implemented

## Exact V4-078 Recommendation

Recommended next slice:

**V4-078 - Candidate Result Promotion Eligibility Derived Status Helper**

Scope:

- add `CandidatePromotionEligibilityStatus`
- add `EvaluateCandidateResultPromotionEligibility(root, resultID string) (CandidatePromotionEligibilityStatus, error)`
- parse only the narrow documented policy grammar
- derive status from committed candidate result, linked run, frozen eval suite, and referenced policy
- expose the derived status through existing candidate-result identity read model
- add focused tests for eligible, rejected, canary-required, owner-approval-required, unsupported-policy, and fail-closed linkage cases

Non-goals for V4-078:

- no scoring
- no eval execution
- no promotion record creation
- no candidate promotion decision
- no canary execution
- no owner approval request
- no runtime-pack pointer mutation
- no last-known-good pointer mutation
- no reload generation mutation
- no commands
- no TaskState wrappers
