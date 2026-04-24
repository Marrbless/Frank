# V4-071 Promotion Policy Registry Skeleton Before State

## Spec Gap

V4-064 identified frozen Frank V4 governance surfaces that were still missing durable skeletons. V4-068 through V4-070 made improvement target surfaces, source-patch artifact-only admission, and disabled-by-default topology admission explicit, but no durable `PromotionPolicy` registry existed.

The frozen spec requires a `PromotionPolicy` rule-set identity for later promotion and hot-update decisions. Before V4-071, the repo had promotion outcome/application records, but not a standalone promotion-policy record that could be stored, listed, replayed, and exposed through the operator read model.

## Missing Fields

No registry record existed for:

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

## Constraints For This Slice

V4-071 is schema, storage, and read-model only. It must not evaluate a policy, decide candidate promotion, execute canaries, enforce holdout pass, mutate runtime-pack pointers, add commands, add TaskState wrappers, or start V4-072.
