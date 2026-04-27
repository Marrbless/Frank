# V4-119 Hot-Update Canary Gate Control Entry Before

## Before-State Gap

V4-117 added the missioncontrol helper:

```go
CreateHotUpdateGateFromCanarySatisfactionAuthority(
    root string,
    canarySatisfactionAuthorityID string,
    ownerApprovalDecisionID string,
    createdBy string,
    createdAt time.Time,
) (HotUpdateGateRecord, bool, error)
```

V4-118 assessed the control path and selected a direct operator command as the smallest safe next slice. Before V4-119, operators could create gates from eligible candidate promotion decisions, but there was no governed direct command or TaskState wrapper for the canary-required authority chain.

## Required Control Surface

The selected command shape is:

```text
HOT_UPDATE_CANARY_GATE_CREATE <job_id> <canary_satisfaction_authority_id> [owner_approval_decision_id]
```

The command uses `canary_satisfaction_authority_id` as source authority. The optional `owner_approval_decision_id` is supplied only for owner-approval-required branches and remains empty for authorized no-owner-approval branches.

## Required Wrapper Behavior

The TaskState wrapper must:

- validate active or persisted job context using existing direct-command patterns;
- reject a supplied `job_id` that does not match the active or persisted job;
- resolve and validate the mission store root;
- normalize and validate `canary_satisfaction_authority_id`;
- normalize and validate non-empty `owner_approval_decision_id`;
- derive `created_by="operator"`;
- derive `created_at` from the TaskState timestamp path;
- derive the deterministic gate ID with `HotUpdateGateIDFromCanarySatisfactionAuthority`;
- reuse an existing deterministic gate `PreparedAt` for replay stability;
- fail closed if an existing deterministic gate cannot be loaded;
- call the V4-117 missioncontrol helper;
- emit `hot_update_canary_gate_create` audit events on created, selected, and rejected paths.

## Invariants To Preserve

V4-119 must not execute gates, advance gate phase, pointer-switch, reload/apply, create outcomes, create promotions, create rollbacks, create rollback-apply records, mutate active runtime-pack pointer, mutate last-known-good pointer, mutate `reload_generation`, mutate canary satisfaction authority or owner approval decision records, mutate source records, broaden `CandidatePromotionDecisionRecord`, create candidate promotion decisions for canary-required states, or implement V4-120.
