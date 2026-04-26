# V4-105 Hot-Update Canary Satisfaction Authority Registry Before

## Before-State Gap

V4-104 selected the next durable authority step after `AssessHotUpdateCanarySatisfaction(...)` returns `satisfied` or `waiting_owner_approval`.

Before V4-105:

- `HotUpdateCanaryRequirementRecord` recorded that policy requires canary.
- `HotUpdateCanaryEvidenceRecord` recorded append-only canary observations.
- `AssessHotUpdateCanarySatisfaction(...)` selected the newest valid evidence and reported `satisfied` or `waiting_owner_approval`.
- `hot_update_canary_satisfaction_identity` exposed the read-only assessment.
- no durable authority record froze the selected passed evidence and satisfaction result.

Without that record, later owner-approval and canary gate work would have to consume raw evidence or a transient read-only assessment directly.

## Required Invariants

V4-105 must preserve the V4-104 authority split:

- `CandidatePromotionDecisionRecord` remains strictly `eligible`-only.
- canary-required results do not create normal candidate promotion decisions.
- raw canary evidence does not create hot-update gates.
- waiting-owner-approval states do not request approval yet.

This slice must not add direct commands, TaskState wrappers, owner approval requests/proposals, candidate promotion decisions, hot-update gates, canary execution, canary evidence, outcomes, promotions, rollbacks, rollback-apply records, pointer mutations, last-known-good changes, `reload_generation` changes, reload/apply behavior, or V4-106 work.
