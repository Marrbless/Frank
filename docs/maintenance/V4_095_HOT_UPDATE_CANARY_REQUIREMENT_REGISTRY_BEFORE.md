# V4-095 Hot-Update Canary Requirement Registry Before State

## Gap From V4-094

V4-094 selected the next safe slice: add a durable canary requirement/proposal registry before any canary execution, canary evidence, owner approval request, candidate promotion decision, or hot-update gate is created for canary-required eligibility states.

Before V4-095, the live system can derive:

- `canary_required`
- `canary_and_owner_approval_required`

from committed candidate results and promotion policies, but there is no durable record representing that canary policy requirement.

## Existing Authority

The existing source authority records are:

- committed `CandidateResultRecord`
- linked `ImprovementRunRecord`
- linked `ImprovementCandidateRecord`
- frozen `EvalSuiteRecord`
- referenced `PromotionPolicyRecord`
- baseline `RuntimePackRecord`
- candidate `RuntimePackRecord`
- freshly derived `CandidatePromotionEligibilityStatus`

`CandidatePromotionDecisionRecord` remains eligible-only and rejects canary-required states. `HotUpdateGateRecord` creation currently depends on a committed eligible candidate promotion decision.

## Required Skeleton

The missing skeleton is a missioncontrol registry/read-model only:

- durable canary requirement record
- deterministic ID helper
- normalize, validate, store, load, and list helpers
- creation helper from candidate result
- read-only operator status identity block
- committed mission status snapshot inclusion through existing status composition

## Constraints

V4-095 must preserve these invariants:

- no canary execution
- no canary evidence
- no owner approval request or owner approval proposal record
- no candidate promotion decision for canary-required states
- no hot-update gate for canary-required states
- no outcome, promotion, rollback, rollback-apply, or last-known-good creation
- no active runtime-pack pointer, last-known-good pointer, runtime pack, candidate result, promotion policy, or `reload_generation` mutation
- no direct command surface
- no TaskState wrapper
- no pointer-switch or reload/apply behavior change
- no promotion policy grammar broadening
- no V4-096 work
