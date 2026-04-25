# V4-099 Hot-Update Canary Evidence Registry Before State

## Gap From V4-098

V4-098 assessed the durable canary evidence surface and selected a missioncontrol registry/read-model skeleton as the next implementation slice.

Before V4-099, the repo had:

- `HotUpdateCanaryRequirementRecord` as the durable policy fact for candidate results whose derived promotion eligibility is `canary_required` or `canary_and_owner_approval_required`
- `HOT_UPDATE_CANARY_REQUIREMENT_CREATE <job_id> <result_id>` as the operator control entry for creating/selecting that requirement
- status/read-model support for `hot_update_canary_requirement_identity`
- canary-adjacent hot-update gate fields such as `canary_ref`, `canarying`, `apply_canary`, and `canary_reload`
- outcome kind `canary_applied`

The missing surface was a durable append-only evidence record that could represent an observed canary result against a committed canary requirement before any later promotion, gate, owner approval, or execution automation path consumes it.

## Required V4-099 Scope

V4-099 must add only the missioncontrol canary evidence registry/read-model skeleton:

- durable evidence record type
- deterministic evidence ID helper
- normalize, validate, store, load, and list helpers
- creation helper scoped to committed `HotUpdateCanaryRequirementRecord`
- read-only `hot_update_canary_evidence_identity` status surface
- committed mission status snapshot composition if consistent with existing status patterns
- focused missioncontrol tests

## Source Authority Records

Canary evidence must be authorized by a committed, valid `HotUpdateCanaryRequirementRecord`.

The evidence helper must load and cross-check:

- `HotUpdateCanaryRequirementRecord`
- committed `CandidateResultRecord`
- linked `ImprovementRunRecord`
- linked `ImprovementCandidateRecord`
- frozen `EvalSuiteRecord`
- referenced `PromotionPolicyRecord`
- baseline `RuntimePackRecord`
- candidate `RuntimePackRecord`
- freshly derived `CandidatePromotionEligibilityStatus`

The source requirement must remain in `state=required`, and the derived eligibility must still be:

- `canary_required`, or
- `canary_and_owner_approval_required`

## Invariants To Preserve

V4-099 must not:

- execute canaries
- create canary execution automation
- create canary execution proposal records
- add a direct command
- add a TaskState wrapper
- request owner approval
- create owner approval proposal records
- create candidate promotion decisions for canary-required states
- create hot-update gates for canary-required states
- create outcomes
- create promotions
- create rollbacks
- create rollback-apply records
- mutate candidate results
- mutate canary requirements
- mutate promotion policies
- mutate runtime packs
- mutate active runtime-pack pointer
- mutate last-known-good pointer
- mutate `reload_generation`
- change pointer-switch behavior
- change reload/apply behavior
- implement V4-100
