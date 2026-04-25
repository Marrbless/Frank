# V4-076 Candidate Result Promotion-Eligibility After State

## Fields Reused Or Added

V4-076 reuses existing `CandidateResultRecord` linkage fields:

- `run_id`
- `candidate_id`
- `eval_suite_id`
- `baseline_pack_id`
- `candidate_pack_id`
- optional `hot_update_id`
- `baseline_score`
- `train_score`
- `holdout_score`
- `complexity_score`
- `compatibility_score`
- `resource_score`
- `regression_flags`
- `decision`

It adds one optional lightweight reference field:

- `promotion_policy_id`

The field uses `omitempty` JSON behavior and is normalized by trimming whitespace.

## Linkage Behavior

Candidate result storage still requires the linked improvement run to exist. The result `candidate_id`, `eval_suite_id`, `baseline_pack_id`, and `candidate_pack_id` must match the linked run. Existing candidate, eval-suite, runtime-pack, and optional hot-update gate linkage checks remain unchanged.

When `promotion_policy_id` is declared, `StoreCandidateResultRecord` and `LoadCandidateResultRecord` validate that the referenced `PromotionPolicyRecord` exists. Candidate results without `promotion_policy_id` remain backward compatible because earlier records did not have that field.

The candidate-result read model now exposes `promotion_policy_id` through `candidate_result_identity` when present.

## Promotion-Eligibility Reference Behavior

V4-076 records the policy identity needed for later promotion-eligibility checks, but it does not evaluate that policy. Train and holdout scores remain explicit separate fields. The current schema has no separate `promotion_eligible` or `promotable` status; therefore this slice does not add train-only promotability rules. Later slices can evaluate `promotion_policy_id`, train/holdout scores, regression flags, canary evidence, and owner approval against a concrete promotion-policy evaluator.

## Replay And Duplicate Behavior

Exact replay of a normalized candidate result remains idempotent and does not rewrite the record. A divergent duplicate with the same `result_id` still fails closed with the existing repo-style registry error.

## Invariants Preserved

V4-076 does not implement scoring, eval execution, promotion-policy evaluation, candidate promotion decisions, canary execution, canary evidence, deploy locks, adaptive lab execution, prompt-pack registries, skill-pack registries, topology mutation, source-patch application or deployment, commands, TaskState wrappers, or V4-077 work.

It does not mutate runtime packs, eval suites, improvement runs, outcomes, promotions, rollbacks, gates, `active_pointer.json`, `last_known_good_pointer.json`, or `reload_generation` except for test fixtures needed to prove linkage behavior.
