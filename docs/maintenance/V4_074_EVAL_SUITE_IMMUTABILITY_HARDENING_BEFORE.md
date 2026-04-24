# V4-074 Eval Suite Immutability Hardening Before State

## Spec Gap

V4-064 identified eval-suite immutability as partial: `EvalSuiteRecord` already required evaluator, rubric, train corpus, holdout corpus, and `frozen_for_run=true`, but `StoreEvalSuiteRecord` still wrote directly to the eval-suite JSON path.

That meant a second store call with the same `eval_suite_id` could silently replace the frozen evaluator, rubric, train corpus, or holdout corpus refs. V4-073 added job-level baseline/train/holdout evidence references, but it did not harden the existing eval-suite registry.

## Store Behavior Before

Before V4-074:

- first write stored the normalized record
- exact replay rewrote the same JSON bytes in practice
- divergent duplicate writes with the same `eval_suite_id` could overwrite the existing frozen suite
- list/load order was deterministic
- `frozen_for_run=true` was already required

## Constraints For This Slice

V4-074 is registry immutability hardening only. It must not implement eval execution, candidate scoring, baseline/train/holdout result registries, promotion-policy evaluation, canary enforcement, adaptive lab behavior, commands, TaskState wrappers, or V4-075 work.
