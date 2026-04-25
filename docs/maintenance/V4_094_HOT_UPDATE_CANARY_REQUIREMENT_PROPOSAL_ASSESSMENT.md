# V4-094 Hot-Update Canary Requirement Proposal Assessment

## Scope

V4-094 assesses the smallest safe durable surface for representing canary-required hot-update policy branches before any canary execution.

This slice is assessment-only. It does not change Go code, tests, commands, TaskState wrappers, candidate promotion decisions, hot-update gates, outcomes, promotions, rollback records, rollback-apply records, active runtime-pack pointer, last-known-good pointer, `reload_generation`, canary evidence, owner approval state, or V4-095 implementation.

## Live State Inspected

Docs inspected:

- `docs/FRANK_V4_SPEC.md`
- `docs/maintenance/V4_078_CANDIDATE_RESULT_PROMOTION_ELIGIBILITY_HELPER_AFTER.md`
- `docs/maintenance/V4_080_CANDIDATE_PROMOTION_DECISION_REGISTRY_AFTER.md`
- `docs/maintenance/V4_090_HOT_UPDATE_EXECUTION_SAFETY_EVIDENCE_REGISTRY_AFTER.md`
- `docs/maintenance/V4_092_HOT_UPDATE_EXECUTION_READY_CONTROL_ENTRY_AFTER.md`
- `docs/maintenance/V4_093_HOT_UPDATE_EXECUTION_SAFETY_LIFECYCLE_CHECKPOINT.md`

Code surfaces inspected:

- candidate result and promotion eligibility helpers
- promotion policy registry
- candidate promotion decision registry
- hot-update gate registry
- hot-update outcome and promotion registries
- rollback and last-known-good registry surfaces
- TaskState hot-update wrappers
- direct command parsing patterns
- operator status/read-model identity patterns

## Current Canary-Related State

`PromotionPolicyRecord` already has canary policy fields:

- `requires_canary`
- `requires_owner_approval`
- `max_canary_duration`

`EvaluateCandidateResultPromotionEligibility(...)` already derives these states:

- `canary_required`
- `owner_approval_required`
- `canary_and_owner_approval_required`

The derived status includes:

- result/run/candidate/eval-suite/policy/baseline-pack/candidate-pack refs
- `canary_required`
- `owner_approval_required`
- blocking reasons and errors

`CreateCandidatePromotionDecisionFromEligibleResult(...)` intentionally creates a `CandidatePromotionDecisionRecord` only when the derived state is `eligible`. The record validator also requires `eligibility_state=eligible` and `decision=selected_for_promotion`.

`CreateHotUpdateGateFromCandidatePromotionDecision(...)` depends on a committed candidate promotion decision, so the current gate path is eligible-only. Canary-required results do not currently become promotion decisions or hot-update gates.

`HotUpdateGateRecord` already has optional `canary_ref`, `approval_ref`, state `canarying`, decision `apply_canary`, and reload mode `canary_reload`, but there is no durable canary requirement, proposal, or evidence registry.

`HotUpdateOutcomeRecord` includes `canary_applied`, but outcome creation currently derives from terminal reload/apply gate state. It is not a canary proposal or canary execution ledger.

`PromotionRecord`, rollback records, rollback-apply records, and last-known-good recertification are downstream ledger/recovery surfaces and should not be used to represent pending canary requirements.

## Decision

The next implementation should add a durable canary requirement/proposal record first.

It should not add canary evidence first. Evidence should prove something that ran; the repo currently has no canary execution path, no canary runner, and no bounded canary workload record. Creating evidence before a requirement/proposal surface would make the authority chain backwards.

It should not add canary execution first. The frozen spec requires train/holdout/canary separation and V4-093 explicitly deferred canary evidence before execution. Executing canaries without a proposal/requirement ledger would widen into opaque runtime behavior.

It should not change `CandidatePromotionDecisionRecord` to accept canary-required states. That record is intentionally eligible-only and feeds direct hot-update gate creation. Overloading it would either permit gate creation too early or require retrofitting every decision consumer with gated-state semantics.

It should not create a `PromotionRecord` for canary-required states. Promotion records are downstream of successful hot-update outcomes, not policy-gate requirements.

It should not add direct commands first. The V4 pattern is missioncontrol registry/read-model first, then TaskState/direct command exposure in a later slice.

## Record Authority

The canary requirement should be authorized by committed source records and a fresh derived eligibility check.

Required source authority:

- committed `CandidateResultRecord`
- derived `CandidatePromotionEligibilityStatus`
- linked `ImprovementRunRecord`
- linked `ImprovementCandidateRecord`
- linked frozen `EvalSuiteRecord`
- referenced `PromotionPolicyRecord`
- baseline `RuntimePackRecord`
- candidate `RuntimePackRecord`

The helper must re-run `EvaluateCandidateResultPromotionEligibility(root, resultID)` and require state:

- `canary_required`, or
- `canary_and_owner_approval_required`

It must cross-check the record fields against the derived status and linked records. It must not accept caller-supplied result/run/candidate/policy/pack authority.

## Where To Tie The Requirement

The durable canary requirement should be tied first to the candidate result and policy authority, not to a hot-update gate or promotion decision.

Reasoning:

- canary requirement is derived before a promotion decision exists
- current promotion decisions are eligible-only
- hot-update gate creation currently requires an eligible promotion decision
- canary-required results need a durable holding/proposal record before they can satisfy canary policy
- tying first to result ID preserves the same source authority as eligibility and avoids creating gates for unsatisfied policy branches

The record should also copy stable refs needed for later linkage:

- result ID
- run ID
- candidate ID
- eval suite ID
- promotion policy ID
- baseline pack ID
- candidate pack ID
- eligibility state
- canary-required flag
- owner-approval-required flag

The future canary evidence record can then reference this canary requirement ID.

## Creation Before Gate

Canary requirement/proposal records should be created before hot-update gate creation.

Current hot-update gate creation is the path for selected, eligible candidates. Canary-required candidates are not selected yet; they are blocked on evidence. Creating a prepared hot-update gate before the canary requirement is satisfied would blur the distinction between "candidate requires a canary" and "candidate is ready to stage/apply."

A later slice may decide whether satisfied canary evidence converts the result into an eligible follow-up decision, produces a gated decision type, or allows a gate with `canary_ref`. V4-095 should not decide execution or conversion behavior beyond preserving enough refs for that future step.

## Ledger Or Current State

V4-095 should add an immutable requirement/proposal ledger, not an overwrite-style current-state record.

Reasoning:

- the source authority is a committed candidate result plus policy
- the requirement is a durable policy fact at the time of derivation
- exact replay can be byte-stable
- divergent duplicate should fail closed
- satisfaction/failure evidence should be separate records rather than mutating the original requirement

The initial record state should be `required`.

Potential later states such as `satisfied`, `failed`, `waived`, `expired`, or `blocked` should not be mutable states in V4-095 unless live implementation proves an existing local pattern for state transitions is needed. Prefer separate evidence/decision records later.

## Minimal Record Shape

Recommended V4-095 record:

```go
type HotUpdateCanaryRequirementRecord struct {
    RecordVersion int
    CanaryRequirementID string
    ResultID string
    RunID string
    CandidateID string
    EvalSuiteID string
    PromotionPolicyID string
    BaselinePackID string
    CandidatePackID string
    EligibilityState string
    RequiredByPolicy bool
    OwnerApprovalRequired bool
    State string
    Reason string
    CreatedAt time.Time
    CreatedBy string
}
```

Recommended deterministic ID:

```text
hot-update-canary-requirement-<result_id>
```

Recommended storage path:

```text
runtime_packs/hot_update_canary_requirements/<canary_requirement_id>.json
```

Recommended initial state:

```text
required
```

`RequiredByPolicy` should be true. `OwnerApprovalRequired` should be true only when eligibility is `canary_and_owner_approval_required`.

The record should not include a hot-update ID, outcome ID, promotion ID, rollback ID, active pointer fields, last-known-good fields, `reload_generation`, canary execution output, owner approval proof, or mutable score overrides.

## Idempotence And Duplicate Behavior

The missioncontrol helper should be shaped like the existing registry helpers:

```go
CreateHotUpdateCanaryRequirementFromCandidateResult(root, resultID, createdBy string, createdAt time.Time) (HotUpdateCanaryRequirementRecord, bool, error)
```

Behavior:

- first write stores the normalized record and returns `changed=true`
- exact replay returns `changed=false`
- byte-stable replay is required
- divergent duplicate for the same `canary_requirement_id` fails closed
- another canary requirement for the same `result_id` fails closed
- derived eligibility changing away from `canary_required` or `canary_and_owner_approval_required` fails closed
- missing linked records or linkage drift fails closed

The helper should reject:

- `eligible`
- `owner_approval_required`
- `rejected`
- `unsupported_policy`
- `invalid`

Those states are not canary requirement authority.

## Canary Plus Owner Approval

`canary_and_owner_approval_required` should produce one canary requirement record with `OwnerApprovalRequired=true`.

It should not create an owner-approval request in V4-095. The canary requirement record should preserve that owner approval remains required after canary evidence, but owner-approval records should be a later dedicated surface.

This avoids using owner approval as a substitute for required canary evidence and avoids creating approval requests before the canary branch is grounded.

## Status / Read Model

V4-095 should add a read-only identity surface following existing status patterns.

Recommended status block:

```text
hot_update_canary_requirement_identity
```

It should list deterministic requirement records, expose invalid records without hiding other records, and be included in `BuildCommittedMissionStatusSnapshot` through the same `With...Identity` composition pattern used by candidate results, promotion decisions, hot-update gates, outcomes, promotions, rollbacks, and rollback applies.

Minimum status fields:

- state
- canary requirement ID
- result/run/candidate/eval-suite/policy refs
- baseline and candidate pack refs
- eligibility state
- required-by-policy flag
- owner-approval-required flag
- requirement state
- reason
- created_at
- created_by
- error

No direct command output is needed in V4-095.

## Recommended V4-095 Slice

Recommend exactly one next slice:

```text
V4-095 — Hot-Update Canary Requirement Registry Skeleton
```

Scope:

- add `HotUpdateCanaryRequirementRecord`
- add deterministic ID helper
- add normalize, validate, store, load, list helpers
- add `CreateHotUpdateCanaryRequirementFromCandidateResult(...)`
- add read-only operator status identity surface
- add focused missioncontrol tests
- add before/after maintenance docs

Do not add TaskState wrappers or direct commands in V4-095.

Rationale:

- this is the smallest durable surface that advances the canary-required branch
- it preserves the eligible-only candidate promotion decision contract
- it avoids creating hot-update gates before canary policy is satisfied
- it keeps canary execution and canary evidence separate
- it gives later slices a stable ID to reference from evidence, gates, approvals, and status

## Required V4-095 Tests

V4-095 should require tests proving:

- validation rejects missing record version
- validation rejects missing canary requirement ID
- validation rejects missing result ID, run ID, candidate ID, eval suite ID, promotion policy ID, baseline pack ID, and candidate pack ID
- validation rejects invalid eligibility state
- validation requires `required_by_policy=true`
- validation requires state `required`
- validation rejects missing reason
- validation rejects missing `created_at`
- validation rejects missing `created_by`
- deterministic ID is `hot-update-canary-requirement-<result_id>`
- helper creates a requirement for `canary_required`
- helper creates a requirement for `canary_and_owner_approval_required` with `owner_approval_required=true`
- helper rejects `eligible`
- helper rejects `owner_approval_required`
- helper rejects `rejected`, `unsupported_policy`, and `invalid`
- helper loads and cross-checks committed result, run, candidate, frozen eval suite, promotion policy, baseline pack, and candidate pack records
- helper re-derives eligibility and fails closed if it changes away from canary-required
- exact replay returns `changed=false` and leaves bytes stable
- divergent duplicate fails closed
- another requirement for the same result fails closed
- list order is deterministic
- read model surfaces configured, not-configured, and invalid states
- read model is read-only and does not mutate candidate results, promotion policies, promotion decisions, hot-update gates, outcomes, promotions, rollbacks, rollback-apply records, runtime-pack pointers, last-known-good pointer, or `reload_generation`

## Must Remain Fail-Closed

V4-095 must fail closed for:

- missing candidate result
- invalid candidate result
- missing linked improvement run
- missing linked improvement candidate
- missing or unfrozen eval suite
- missing promotion policy
- missing baseline or candidate runtime pack
- eligibility states other than `canary_required` or `canary_and_owner_approval_required`
- mismatched record fields versus derived eligibility
- duplicate divergent records
- stale derived eligibility

## Non-Goals For V4-095

V4-095 must not:

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
- claim V4 complete

## Assessment Answer

Canary-required candidate results should create a separate durable canary requirement/proposal record first, not a `CandidatePromotionDecisionRecord`. `CandidatePromotionDecisionRecord` is intentionally eligible-only and should remain the authority for selected candidates that can proceed to prepared hot-update gate creation.

The canary requirement should be authorized by the committed candidate result and its linked run, candidate, frozen eval suite, promotion policy, baseline pack, candidate pack, and freshly derived `canary_required` or `canary_and_owner_approval_required` eligibility.

The record should be tied primarily to candidate result identity and policy authority. It should be created before hot-update gate creation, stored as an immutable ledger with exact replay and divergent duplicate rejection, and exposed through a read-only status identity surface.

The next safe implementation is V4-095, a missioncontrol registry/read-model skeleton only. Direct commands, TaskState wrappers, canary evidence, canary execution, owner approval, hot-update gate creation, outcomes, promotions, rollbacks, LKG mutation, and pointer/reload changes remain deferred.
