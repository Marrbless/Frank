# V4-078 Candidate Result Promotion Eligibility Helper Before State

## Before-State Gap

V4-076 added `promotion_policy_id` to `CandidateResultRecord` and exposed it through candidate-result identity status. V4-077 assessed the smallest safe evaluator and recommended a read-only derived status helper before any durable eligibility ledger, promotion record, canary execution, or owner approval flow.

Before V4-078, candidate results could reference a promotion policy, but the repo did not derive whether a committed result was eligible, rejected, canary-gated, owner-approval-gated, unsupported, or invalid under that policy.

## Existing Inputs

The existing source records before this slice were already sufficient for a conservative derived helper:

- `CandidateResultRecord`
- linked `ImprovementRunRecord`
- linked frozen `EvalSuiteRecord`
- linked baseline and candidate runtime packs
- referenced `PromotionPolicyRecord`

The result score fields were plain `float64`, so missing JSON score keys decoded as zero and required a separate raw JSON presence check before typed evaluation.

## Scope Boundary

This slice must remain read-only. It must not create a `PromotionRecord`, create a durable eligibility record, mutate candidate results, mutate improvement runs, mutate eval suites, mutate promotion policies, mutate runtime-pack pointers, mutate last-known-good pointers, mutate `reload_generation`, execute canaries, request owner approval, run evals, score candidates, add commands, add TaskState wrappers, or start V4-079 work.
