# V4-071 Promotion Policy Registry Skeleton After State

## Fields Added

V4-071 adds `PromotionPolicyRecord` with:

- `promotion_policy_id`
- `requires_holdout_pass`
- `requires_canary`
- `requires_owner_approval`
- `allows_autonomous_hot_update`
- `allowed_surface_classes`
- `epsilon_rule`
- `regression_rule`
- `compatibility_rule`
- `resource_rule`
- `max_canary_duration`
- `forbidden_surface_changes`
- `created_at`
- `created_by`
- optional `notes`

## Storage Path And Registry Behavior

Promotion policies are stored under:

`runtime_packs/promotion_policies/<promotion_policy_id>.json`

The registry provides normalize, validate, store, load, and list helpers. `StorePromotionPolicyRecord` normalizes IDs, rules, metadata, `allowed_surface_classes`, and `forbidden_surface_changes`; list output is deterministic by file name.

## Validation And Idempotence

Validation rejects missing `promotion_policy_id`, missing required rules, missing allowed surface classes, missing forbidden surface changes, missing `created_at`, missing `created_by`, and invalid or non-positive `max_canary_duration`.

Exact replay of an existing normalized record is idempotent. A divergent duplicate write with the same `promotion_policy_id` fails closed.

## Read Model

The operator status read model now includes `promotion_policy_identity` when status snapshots attach registry identities. It exposes configured policy identities and policy metadata deterministically. Invalid policy records are surfaced as invalid read-model entries.

## Compatibility

No existing job is required to reference a promotion policy in this slice. Existing immutable surface declarations may name policy refs, but V4-071 only creates the registry skeleton and does not enforce linkage from jobs or improvement records.

## Invariants Preserved

V4-071 does not implement promotion-policy evaluation, canary execution, candidate promotion decisions, adaptive lab execution, eval runs, topology mutation, source-patch deployment, deploy locks, commands, TaskState wrappers, or V4-072 work.

It does not mutate runtime packs, candidates, eval suites, improvement runs, outcomes, promotions, rollbacks, hot-update gates, `active_pointer.json`, `last_known_good_pointer.json`, or `reload_generation`. The only new durable records are promotion policy records.
