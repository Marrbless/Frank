# V4-119 Hot-Update Canary Gate Control Entry After

## Implemented Surface

V4-119 exposes the V4-117 canary-gate helper through the governed operator path:

```text
HOT_UPDATE_CANARY_GATE_CREATE <job_id> <canary_satisfaction_authority_id> [owner_approval_decision_id]
```

The command calls:

```go
func (s *TaskState) CreateHotUpdateGateFromCanarySatisfactionAuthority(
    jobID string,
    canarySatisfactionAuthorityID string,
    ownerApprovalDecisionID string,
) (missioncontrol.HotUpdateGateRecord, bool, error)
```

The wrapper validates the active or persisted job context, validates the mission store root, normalizes and validates the canary satisfaction authority ref, normalizes and validates a non-empty owner approval decision ref, derives `created_by="operator"`, derives `created_at` from the TaskState transition timestamp path, reuses an existing deterministic gate `PreparedAt` for exact replay, and calls `missioncontrol.CreateHotUpdateGateFromCanarySatisfactionAuthority`.

## Response Semantics

Created response:

```text
Created hot-update canary gate job=<job_id> canary_satisfaction_authority=<canary_satisfaction_authority_id> hot_update=<hot_update_id> canary_ref=<canary_ref> approval_ref=<approval_ref>.
```

Selected response:

```text
Selected hot-update canary gate job=<job_id> canary_satisfaction_authority=<canary_satisfaction_authority_id> hot_update=<hot_update_id> canary_ref=<canary_ref> approval_ref=<approval_ref>.
```

Malformed commands return an empty response with:

```text
HOT_UPDATE_CANARY_GATE_CREATE requires job_id, canary_satisfaction_authority_id, and optional owner_approval_decision_id
```

## Audit And Replay

The wrapper emits `hot_update_canary_gate_create` audit events for created, selected/idempotent, and rejected paths. Exact replay reuses the stored `PreparedAt` from the deterministic gate record before calling the missioncontrol helper, preserving byte-stable replay when the normalized source refs and `created_by` match.

## Fail-Closed Behavior

The control entry fails closed for malformed args, wrong job ID, missing mission store root, missing active or persisted runtime context, invalid or stale canary satisfaction authority, no-owner branches with supplied owner approval decision IDs, owner-required branches without owner approval decision IDs, missing/rejected/stale/mismatched owner approval decisions, stale fresh canary satisfaction, stale fresh promotion eligibility, selected evidence failures, missing or mismatched source records, missing or mismatched active pointer, missing rollback target, invalid present LKG pointer, invalid deterministic gate loads, and divergent duplicate gates.

## Status Read Model

`STATUS <job_id>` continues to surface gates through `hot_update_gate_identity`. The read-only gate status now includes `canary_ref` and `approval_ref` so operators can see the canary satisfaction authority and owner approval decision authority consumed by canary-derived gates.

## Invariants Preserved

V4-119 does not execute gates, advance gate phase, pointer-switch, reload/apply, create hot-update outcomes, create promotions, create rollbacks, create rollback-apply records, mutate active runtime-pack pointer, mutate last-known-good pointer, mutate `reload_generation`, mutate canary satisfaction authority, mutate owner approval decision, mutate source records, broaden `CandidatePromotionDecisionRecord`, create candidate promotion decisions for canary-required states, change `CreateHotUpdateGateFromCandidatePromotionDecision`, or implement V4-120.
