# V4-079 Promotion Eligibility To Promotion Decision Assessment

## Facts

- This is a docs-only assessment slice after V4-078.
- V4-078 added the read-only helper `EvaluateCandidateResultPromotionEligibility(root, resultID string) (CandidatePromotionEligibilityStatus, error)`.
- The helper derives `promotion_eligibility` for candidate-result identity status from committed store records.
- Supported eligibility states are:
  - `eligible`
  - `canary_required`
  - `owner_approval_required`
  - `canary_and_owner_approval_required`
  - `rejected`
  - `unsupported_policy`
  - `invalid`
- Current `PromotionRecord` is not a general candidate decision record. It represents actual promotion from a successful hot-update outcome and requires `hot_update_id`, promoted and previous-active pack IDs, and store-aware hot-update linkage.
- Current rollback and last-known-good helpers operate after a promotion or rollback path exists. They are not authority for creating a candidate promotion decision.

## Existing Source Authority

Future decision creation should use only committed store/read-model authority:

- committed `CandidateResultRecord`
- derived `CandidatePromotionEligibilityStatus`
- referenced `PromotionPolicyRecord`
- linked `ImprovementRunRecord`
- linked frozen `EvalSuiteRecord`
- linked baseline and candidate runtime packs as immutable linkage evidence

The future helper should not accept caller-supplied scores, policy fields, run records, eval-suite records, or pack refs. The candidate result ID and mission store root are enough to derive all authority if linkage is valid.

## Decision On Next Durable Surface

V4-080 should create a durable `CandidatePromotionDecisionRecord` registry skeleton.

This is the smallest safe next implementation because:

- V4-078 already makes positive eligibility deterministic and read-only.
- The frozen spec requires a separate promotion or hot-update decision after train and holdout evidence.
- Existing `PromotionRecord` is too late in the pipeline because it represents an actual hot-update promotion, not a pre-promotion decision.
- A durable decision record can preserve operator/audit intent without mutating active pointers, last-known-good pointers, reload generation, hot-update gates, canary state, or approval state.
- Canary and owner-approval requirements should not be silently collapsed into promotion decisions.

The decision record should be created only for `promotion_eligibility.state == "eligible"` in the first implementation slice.

## Valid Future Transitions

- `eligible` -> create durable `CandidatePromotionDecisionRecord`.
- `canary_required` -> create no promotion decision yet; future work should add a canary requirement/proposal record.
- `owner_approval_required` -> create no promotion decision yet; future work should add an owner-approval request/proposal record.
- `canary_and_owner_approval_required` -> create no promotion decision yet; future work should add both gate requirements or a combined gate record.
- `rejected` -> fail closed, no promotion decision.
- `unsupported_policy` -> fail closed, no promotion decision.
- `invalid` -> fail closed, no promotion decision.

Eligibility should remain read-only for gated, rejected, unsupported, and invalid states until the corresponding gate or correction surfaces exist.

## Recommended V4-080 Slice

Recommended next slice:

**V4-080 - Durable Candidate Promotion Decision Registry Skeleton**

Scope:

- add `CandidatePromotionDecisionRecord`
- add normalize, validate, store, load, and list helpers
- add a helper that derives a decision from an eligible candidate result, likely:

```go
CreateCandidatePromotionDecisionFromEligibleResult(root, resultID, createdBy string, createdAt time.Time) (CandidatePromotionDecisionRecord, bool, error)
```

- use `EvaluateCandidateResultPromotionEligibility(root, resultID)` as the decision gate
- reject any state other than `eligible`
- keep exact replay idempotent
- fail closed on divergent duplicates
- expose decision identity through the existing operator status/read-model pattern only if the repo-consistent identity-status surface is small

## Decision Record Fields

Minimum recommended fields:

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

Recommended `decision` value for the first slice:

- `selected_for_promotion`

The record should not contain active pointer paths, last-known-good pointer fields, reload generation values, canary outcome, owner approval evidence, or mutable score overrides.

## Storage And Idempotence

Recommended storage path:

`runtime_packs/candidate_promotion_decisions/<promotion_decision_id>.json`

Recommended duplicate behavior:

- first write stores the normalized record
- exact replay of the same normalized record is idempotent
- divergent duplicate with the same `promotion_decision_id` fails closed
- V4-080 should also fail closed if a different decision record already references the same `result_id`, unless it is the exact same normalized record

The secondary `result_id` uniqueness check is worth including in V4-080 because a candidate result should not acquire multiple conflicting promotion decisions.

## Relationship To Existing PromotionRecord

`PromotionRecord` is downstream of hot-update success. It currently requires:

- promoted pack ID
- previous active pack ID
- hot-update ID
- optional hot-update outcome ID
- optional candidate, run, and candidate-result linkage

It validates linkage to hot-update gates, outcomes, candidates, runs, candidate results, and runtime packs. Creating a `PromotionRecord` too early would imply actual promotion/hot-update completion semantics that V4-079 explicitly does not implement.

`CandidatePromotionDecisionRecord` should be an upstream immutable intent/proposal record. It can later become authority for:

- a hot-update gate proposal,
- a canary requirement,
- an owner-approval request,
- or a later promotion creation helper after all gates pass.

It must not be treated as active runtime promotion.

## Answers To Required Questions

### Should Promotion Eligibility Remain Read-Only Until Canary/Approval Surfaces Exist?

Eligibility should remain read-only for canary-required and owner-approval-required results. An eligible result with no remaining gates may produce a durable decision skeleton because that record still does not promote, hot-update, or mutate runtime state.

### Should An Eligible Result Produce A Durable CandidatePromotionDecisionRecord Before Actual Pack Promotion?

Yes. The durable record is the next smallest safe artifact between derived eligibility and actual promotion. It records that a committed eligible result was selected for future promotion handling, while leaving all runtime mutation to later slices.

### What Fields Would That Decision Record Need?

The first record needs stable identity, all source-linkage IDs copied from the candidate result and derived status, the selected decision value, reason, creation metadata, and optional notes. It should not duplicate scores or policy rules unless a later slice adds an explicit digest/snapshot requirement.

### How Should Idempotence And Divergent Duplicates Work?

Exact replay should be idempotent. Divergent duplicate writes by `promotion_decision_id` should fail closed. A second decision for the same `result_id` should also fail closed unless it is an exact replay of the same normalized decision.

### Should Decisions Be Allowed For Canary-Required Or Owner-Approval-Required Results?

No. Those states mean the candidate is blocked on explicit gates. V4-080 should reject them and defer separate canary and approval surfaces to later slices.

### How Does This Relate To Existing PromotionRecord Used By Hot-Update Promotions?

Existing `PromotionRecord` should remain the durable record for an actual hot-update promotion after a successful hot-update outcome. A candidate promotion decision is earlier, read from eligibility, and must not satisfy or replace the hot-update outcome, canary, approval, rollback, or last-known-good requirements.

### What Must Not Be Mutated By The Decision Helper?

The future decision helper must not mutate:

- `CandidateResultRecord`
- `ImprovementRunRecord`
- `EvalSuiteRecord`
- `PromotionPolicyRecord`
- runtime pack records
- hot-update gates
- hot-update outcomes
- promotions
- rollbacks
- active runtime-pack pointer
- last-known-good pointer
- `reload_generation`
- canary evidence
- owner approval evidence

It also must not run evals, score candidates, execute canaries, request owner approval, add commands, add TaskState wrappers, or create `PromotionRecord`.

## Recommended Tests For V4-080

Focused tests should prove:

- eligible result creates a durable decision record
- exact replay is idempotent
- divergent duplicate `promotion_decision_id` fails closed
- second decision for the same `result_id` fails closed
- missing candidate result fails closed
- invalid eligibility state fails closed
- rejected eligibility state fails closed
- unsupported policy eligibility state fails closed
- canary-required eligibility state fails closed without creating a decision
- owner-approval-required eligibility state fails closed without creating a decision
- canary-and-owner-approval-required eligibility state fails closed without creating a decision
- stored decision loads and lists deterministically
- no promotion records, hot-update gates, outcomes, rollbacks, runtime-pack pointers, last-known-good pointer, or `reload_generation` are mutated

## Invariants Preserved

This V4-079 assessment does not change Go code, tests, commands, TaskState wrappers, candidate results, improvement runs, eval suites, promotion policies, promotion decisions, `PromotionRecord`, hot-update gates, canary state, owner approval state, runtime-pack pointers, last-known-good pointer, `reload_generation`, eval execution, scoring, or V4-080 implementation.
