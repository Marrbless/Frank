# V4-072 Promotion Policy Reference Admission After State

## Field Added

V4-072 adds job-level `promotion_policy_id` with `omitempty` JSON behavior.

The field now flows through the same V4 job metadata path as execution-plane and surface declarations:

- `Job`
- `InspectablePlanContext`
- `JobRuntimeState`
- `RuntimeControlContext`
- `JobRuntimeRecord`
- `RuntimeControlRecord`
- `InspectSummary`
- `OperatorStatusSummary`

## Syntactic Validation

For `spec_version=frank_v4` improvement-family jobs, `promotion_policy_id` is required.

Missing or whitespace-only values reject with:

`E_PROMOTION_POLICY_REQUIRED`

When no mission store root is available, `ValidatePlan(job)` performs syntactic validation only. A syntactically valid `promotion_policy_id` admits the job if all other V4 metadata and surface validation passes.

## Store-Aware Validation

When `job.MissionStoreRoot` is set, `ValidatePlan(job)` also validates that the referenced `PromotionPolicyRecord` exists in `runtime_packs/promotion_policies`.

Missing or invalid registry references reject with:

`E_PROMOTION_POLICY_REQUIRED`

This slice does not evaluate policy rules or enforce linkage through immutable surfaces.

## Read-Model / Status Exposure

`promotion_policy_id` is exposed deterministically through inspect summaries, operator status summaries, inspectable plan snapshots, runtime state records, and runtime control records.

The existing V4-071 `promotion_policy_identity` status remains available and deterministic.

## Compatibility

Non-improvement V4 jobs are not required to declare `promotion_policy_id` yet. Pre-V4 jobs remain backward compatible. Existing records without the field are not rewritten solely by this slice.

## Invariants Preserved

V4-072 does not implement promotion-policy evaluation, holdout enforcement, canary enforcement, owner-approval enforcement, canary evidence, deploy locks, adaptive lab execution, eval runs, baseline/train/holdout logic, prompt-pack registries, skill-pack registries, topology mutation, source-patch application or deployment, commands, TaskState wrappers, or V4-073 work.

It does not mutate `active_pointer.json`, `last_known_good_pointer.json`, or `reload_generation`. It does not create runtime packs, candidates, eval suites, improvement runs, outcomes, promotions, rollbacks, or gates except test fixtures where needed.
