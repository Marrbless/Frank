# V4-072 Promotion Policy Reference Admission Before State

## Spec Gap

V4-071 added a durable `PromotionPolicyRecord` registry under `runtime_packs/promotion_policies`, but V4 jobs did not yet carry a job-level promotion policy reference.

That left improvement-family admission unable to prove which frozen promotion policy would govern later promotion, hot-update, canary, holdout, or owner-approval decisions. Existing immutable surface declarations could name policy-like refs, but the job schema and read models had no dedicated `promotion_policy_id` field and no admission check for policy registration.

## Missing Field

The missing job-level field was:

- `promotion_policy_id`

## Constraints For This Slice

V4-072 is admission and reference validation only. It must add the reference and prove it flows through existing metadata, storage, inspect, and status surfaces without evaluating the policy.

It must not enforce holdout pass, canary requirements, owner approval, deploy locks, baseline/train/holdout behavior, adaptive lab execution, or V4-073 work.
