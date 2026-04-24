# V4-073 Baseline / Holdout Evidence Reference After State

## Fields Added

V4-073 adds these job-level evidence reference fields with `omitempty` JSON behavior:

- `baseline_ref`
- `train_ref`
- `holdout_ref`

They flow through the same V4 metadata path as execution-plane fields, target surfaces, topology mode, and `promotion_policy_id`:

- `Job`
- `InspectablePlanContext`
- `JobRuntimeState`
- `RuntimeControlContext`
- `JobRuntimeRecord`
- `RuntimeControlRecord`
- `InspectSummary`
- `OperatorStatusSummary`

## Validation Behavior

For `spec_version=frank_v4` improvement-family jobs:

- missing or whitespace-only `baseline_ref` rejects with `E_BASELINE_REQUIRED`
- missing or whitespace-only `train_ref` rejects with `E_TRAIN_REQUIRED`
- missing or whitespace-only `holdout_ref` rejects with `E_HOLDOUT_REQUIRED`
- `train_ref == holdout_ref` rejects with `E_MUTATION_SCOPE_VIOLATION`

`baseline_ref == train_ref` and `baseline_ref == holdout_ref` are not rejected in this slice because the repo does not yet have a stable baseline/train/holdout artifact registry or path convention that proves these must be distinct references. Train/holdout separation is explicit in the frozen spec and is enforced here.

Non-improvement V4 jobs are not required to declare these evidence refs. Pre-V4 jobs remain backward compatible.

## Read-Model / Status Exposure

`baseline_ref`, `train_ref`, and `holdout_ref` are exposed deterministically through inspect summaries, operator status summaries, inspectable plan snapshots, runtime state records, and runtime control records.

## Invariants Preserved

V4-073 does not implement eval execution, candidate scoring, promotion-policy evaluation, holdout enforcement beyond required reference declaration, canary enforcement, owner approval, canary evidence, deploy locks, adaptive lab execution, prompt-pack registries, skill-pack registries, topology mutation, source-patch application or deployment, commands, TaskState wrappers, or V4-074 work.

It does not mutate `active_pointer.json`, `last_known_good_pointer.json`, or `reload_generation`. It does not create runtime packs, candidates, eval suites, improvement runs, outcomes, promotions, rollbacks, or gates except test fixtures where needed.
