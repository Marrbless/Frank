# V4-074 Eval Suite Immutability Hardening After State

## Store Behavior After

`StoreEvalSuiteRecord` now treats `eval_suite_id` as immutable by ID:

- first write stores the normalized record
- exact replay of the same normalized record is idempotent and returns without rewriting
- divergent duplicate writes with the same `eval_suite_id` fail closed with a repo-style registry error
- deterministic load/list behavior is preserved
- `frozen_for_run=true` remains required

The divergent duplicate error is a store-layer registry error, not a `ValidationError`, so it does not emit `E_EVAL_IMMUTABLE`. V4 rejection codes remain attached to validation/admission surfaces; this slice hardens registry write semantics directly.

## Invariants Preserved

V4-074 does not change `EvalSuiteRecord` schema. Evaluator, rubric, train corpus, and holdout corpus refs are preserved exactly after normalization.

It does not implement eval execution, candidate scoring, baseline/train/holdout result registries, promotion-policy evaluation, canary enforcement, canary evidence, deploy locks, adaptive lab execution, prompt-pack registries, skill-pack registries, topology mutation, source-patch application or deployment, commands, TaskState wrappers, or V4-075 work.

It does not mutate runtime packs, candidates, improvement runs, candidate results, outcomes, promotions, rollbacks, hot-update gates, `active_pointer.json`, `last_known_good_pointer.json`, or `reload_generation`.
