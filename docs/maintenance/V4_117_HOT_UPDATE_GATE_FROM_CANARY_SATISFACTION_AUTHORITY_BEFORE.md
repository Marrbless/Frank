# V4-117 - Hot-Update Gate From Canary Satisfaction Authority Helper Before

## Before-State Gap

V4-116 established that canary-required candidate results cannot safely use the existing `CandidatePromotionDecisionRecord` gate path because that record remains strictly eligible-only. The live gate registry had `canary_ref` and `approval_ref` fields, but no missioncontrol helper could create a prepared `HotUpdateGateRecord` from the durable canary satisfaction authority chain.

The existing helper:

```go
CreateHotUpdateGateFromCandidatePromotionDecision(root, promotionDecisionID, createdBy string, createdAt time.Time) (HotUpdateGateRecord, bool, error)
```

remained correct only for `eligibility_state=eligible` candidate promotion decisions. It did not support:

- `canary_required`
- `canary_and_owner_approval_required`
- `HotUpdateCanarySatisfactionAuthorityRecord` source authority
- `HotUpdateOwnerApprovalDecisionRecord` granted owner approval authority

## Required Next Slice

V4-117 needed to add only a missioncontrol helper and tests in the existing hot-update gate registry.

The intended helper surface was:

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

The deterministic gate ID needed to be:

```text
hot-update-canary-gate-<sha256(canary_satisfaction_authority_id)>
```

## Required Authority Behavior

The no-owner-approval branch needed:

- canary satisfaction authority `state=authorized`
- `owner_approval_required=false`
- `satisfaction_state=satisfied`
- fresh canary satisfaction still `satisfied`
- fresh promotion eligibility still `canary_required`
- empty owner approval decision ID

The owner-approved branch needed:

- canary satisfaction authority `state=waiting_owner_approval`
- `owner_approval_required=true`
- `satisfaction_state=waiting_owner_approval`
- non-empty owner approval decision ID
- loaded owner approval decision with `decision=granted`
- fresh canary satisfaction still `waiting_owner_approval`
- fresh promotion eligibility still `canary_and_owner_approval_required`

Both branches needed passed selected evidence, source-record cross-checks, active pointer baseline validation, candidate rollback target validation, and last-known-good tolerance matching the existing gate helper.

## Invariants To Preserve

V4-117 must not add a direct command, TaskState wrapper, natural-language binding, candidate promotion decision creation or broadening, outcomes, promotions, rollbacks, rollback-apply records, active pointer mutations, last-known-good mutations, `reload_generation` changes, pointer-switch behavior changes, reload/apply changes, or V4-118 work.
