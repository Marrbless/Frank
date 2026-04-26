# V4-117 - Hot-Update Gate From Canary Satisfaction Authority Helper After

## Implemented Helper

V4-117 adds a missioncontrol helper in `internal/missioncontrol/hot_update_gate_registry.go`:

```go
HotUpdateGateIDFromCanarySatisfactionAuthority(canarySatisfactionAuthorityID string) string

CreateHotUpdateGateFromCanarySatisfactionAuthority(
    root string,
    canarySatisfactionAuthorityID string,
    ownerApprovalDecisionID string,
    createdBy string,
    createdAt time.Time,
) (HotUpdateGateRecord, bool, error)
```

The deterministic gate ID is:

```text
hot-update-canary-gate-<sha256(canary_satisfaction_authority_id)>
```

The hash keeps the gate filename bounded while the gate record preserves the exact source authority in:

```text
canary_ref=<canary_satisfaction_authority_id>
```

## No-Owner-Approval Branch

The helper creates/selects a prepared gate when the canary satisfaction authority has:

```text
state=authorized
owner_approval_required=false
satisfaction_state=satisfied
```

The caller must provide an empty `ownerApprovalDecisionID`. A non-empty owner approval decision ID fails closed to avoid ambiguous authority.

The helper re-loads and cross-checks the requirement, selected passed evidence, candidate result, improvement run, improvement candidate, eval suite, promotion policy, baseline runtime pack, candidate runtime pack, fresh canary satisfaction assessment, and fresh promotion eligibility. Fresh eligibility must remain `canary_required`.

## Owner-Approved Branch

The helper creates/selects a prepared gate when the canary satisfaction authority has:

```text
state=waiting_owner_approval
owner_approval_required=true
satisfaction_state=waiting_owner_approval
```

The caller must provide a committed owner approval decision. That decision must have:

```text
decision=granted
```

and its copied refs must match the canary satisfaction authority. `decision=rejected`, a missing decision, or a decision for another authority fails closed. Fresh canary satisfaction must remain `waiting_owner_approval`, and fresh eligibility must remain `canary_and_owner_approval_required`.

The gate records:

```text
approval_ref=<owner_approval_decision_id>
```

For the no-owner branch, `approval_ref` remains empty.

## Gate Shape

V4-117 creates only prepared gates:

```text
state=prepared
decision=keep_staged
```

It does not set `state=canarying`, does not set `decision=apply_canary`, and does not force `reload_mode=canary_reload`.

`reload_mode` is derived from the candidate runtime pack's mutable surfaces using the existing gate behavior:

- `skills` -> `skill_reload`
- `extensions` -> `extension_reload`
- otherwise `soft_reload`

The gate copies target surfaces, surface classes, and compatibility contract refs from the candidate runtime pack through the same build path as existing gate creation. `previous_active_pack_id` comes from the active pointer and must match the authority baseline pack. `rollback_target_pack_id` comes from the candidate pack and must load successfully.

## Active Pointer, Rollback Target, And LKG

The helper requires:

- active runtime-pack pointer exists
- active pointer `active_pack_id` equals the authority baseline pack
- candidate pack has a non-empty `rollback_target_pack_id`
- rollback target runtime pack loads

The helper allows a missing last-known-good pointer, matching the current candidate-promotion gate helper. If a last-known-good pointer is present but invalid, the helper fails closed.

## Replay And Duplicates

The first write stores the normalized prepared gate and returns `changed=true`.

Exact replay with the same normalized refs and `createdAt` returns `changed=false` and leaves bytes stable.

Divergent duplicates for the deterministic gate ID fail closed, including mismatched candidate pack, `canary_ref`, `approval_ref`, phase timestamps, reload/source fields, state, or decision.

## Invariants Preserved

V4-117 does not add direct commands, TaskState wrappers, natural-language approval binding, candidate promotion decision changes, candidate promotion decisions for canary-required states, outcomes, promotions, rollbacks, rollback-apply records, active pointer mutations, last-known-good mutations, `reload_generation` changes, pointer-switch changes, reload/apply changes, or V4-118 work.
