# V4-103 Hot-Update Canary Satisfaction Assessment Before

## Before-State Gap

V4-102 established that passed canary evidence must not silently convert a canary-required candidate result into the normal `eligible` promotion-decision path. The repo already had durable canary requirements and append-only canary evidence, but no read-only authority helper that answered whether a specific requirement was satisfied by valid passed evidence.

Before V4-103:

- `HotUpdateCanaryRequirementRecord` represented a policy-required canary.
- `HotUpdateCanaryEvidenceRecord` represented operator-recorded canary observations.
- status exposed requirement and evidence identities separately.
- no helper selected the latest valid evidence for a requirement.
- no status/read-model identity exposed `satisfied`, `waiting_owner_approval`, `failed`, `blocked`, `expired`, `not_satisfied`, or `invalid` canary satisfaction state.

## Invariants Entering The Slice

This slice starts with the V4-102 decision that `CandidatePromotionDecisionRecord` remains strictly for derived `eligible` results. Canary-required states continue to require distinct authority before any later promotion/gate path can proceed.

V4-103 must not create durable satisfaction records, direct commands, TaskState wrappers, candidate promotion decisions, hot-update gates, owner approval requests, canary evidence, outcomes, promotions, rollbacks, rollback-apply records, runtime-pack pointer mutations, last-known-good mutations, `reload_generation` changes, pointer-switch changes, reload/apply changes, or V4-104 behavior.
