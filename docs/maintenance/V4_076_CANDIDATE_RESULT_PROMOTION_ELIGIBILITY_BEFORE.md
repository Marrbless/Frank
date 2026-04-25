# V4-076 Candidate Result Promotion-Eligibility Before State

## Before-State Gap

V4-075 hardened improvement-run to eval-suite linkage. Candidate results already linked to an improvement run, candidate, eval suite, baseline pack, candidate pack, optional hot-update gate, train score, holdout score, regression flags, and decision. Candidate-result storage was already immutable by `result_id`: exact replay was idempotent and divergent duplicates failed closed.

The remaining V4 promotion-eligibility gap was that candidate results did not carry a promotion policy reference. V4-071 introduced the durable promotion policy registry and V4-072 added job-level `promotion_policy_id`, but candidate result records could not yet declare which policy identity later eligibility checks should use.

## Existing Linkage Behavior

Before this slice, `StoreCandidateResultRecord` already loaded the linked improvement run and required result `candidate_id`, `eval_suite_id`, `baseline_pack_id`, and `candidate_pack_id` to match the run. It also checked candidate, eval-suite, runtime-pack, and optional hot-update gate consistency.

The store-aware candidate-result path did not evaluate scores, did not compute policy outcomes, and did not mutate runtime pointers or promotion records.

## Scope Boundary

This slice is schema/linkage validation only. It must not implement scoring, eval execution, promotion-policy evaluation, candidate promotion decisions, canary execution, adaptive lab execution, commands, TaskState wrappers, or V4-077 work.
