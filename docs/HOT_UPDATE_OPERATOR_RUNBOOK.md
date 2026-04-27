# Hot-Update Operator Runbook

This runbook covers the completed V4 hot-update lifecycle from gate creation through last-known-good recertification. It includes both the eligible-only path and the canary-required path through promotion audit lineage.

Use it during live operator work. Keep the sequence explicit. Later ledger and recertification steps are not automatic side effects of earlier commands.

## Direct-Command Help

Use either static help command when you need the current V4 hot-update command index from the operator channel:

```text
HOT_UPDATE_HELP
HELP HOT_UPDATE
HELP V4
```

The help response is read-only. It does not require an active mission step, does not create records, and points back to this runbook for the full workflow.

## Placeholders

- `<job_id>`: active mission job id
- `<hot_update_id>`: durable hot-update workflow id
- `<candidate_pack_id>`: runtime pack selected for hot update
- `<outcome_id>`: usually `hot-update-outcome-<hot_update_id>`
- `<promotion_id>`: usually `hot-update-promotion-<hot_update_id>`
- `<result_id>`: committed candidate result id
- `<canary_requirement_id>`: deterministic canary requirement id for the candidate result
- `<canary_satisfaction_authority_id>`: deterministic authority id from the requirement and selected evidence
- `<owner_approval_request_id>`: deterministic owner approval request id for the authority
- `<owner_approval_decision_id>`: deterministic owner approval decision id for the request
- `<observed_at>`: RFC3339 or RFC3339Nano canary evidence timestamp
- `<reason>`: non-empty operator reason text

## High-Level Path Summary

The canary-required hot-update path is complete through promotion audit lineage. Canary-derived outcomes and promotions preserve `canary_ref`, and owner-approved canary outcomes/promotions preserve both `canary_ref` and `approval_ref`.

The eligible-only path remains separate:

- `CandidatePromotionDecisionRecord` remains strictly `eligibility_state=eligible`.
- `CreateHotUpdateGateFromCandidatePromotionDecision(...)` remains unchanged.
- Do not use the eligible-only candidate promotion decision path for `canary_required` or `canary_and_owner_approval_required` results.

## Canary-Required Branch Overview

### No-Owner-Approval Branch

```text
candidate result + policy
-> canary requirement
-> passed canary evidence
-> canary satisfaction=satisfied
-> canary satisfaction authority=authorized
-> prepared canary gate with canary_ref and empty approval_ref
-> guarded phase/execute/reload
-> outcome with copied canary_ref
-> promotion with copied canary_ref
```

### Owner-Approved Branch

```text
candidate result + policy
-> canary requirement
-> passed canary evidence
-> canary satisfaction=waiting_owner_approval
-> canary satisfaction authority=waiting_owner_approval
-> owner approval request=requested
-> owner approval decision=granted
-> prepared canary gate with canary_ref and approval_ref
-> guarded phase/execute/reload
-> outcome with copied canary_ref and approval_ref
-> promotion with copied canary_ref and approval_ref
```

### Rejected Owner Approval

```text
owner approval decision=rejected
```

A rejected owner approval decision records durable terminal rejection authority. It does not authorize `HOT_UPDATE_CANARY_GATE_CREATE`, must not be treated as approval, and must not authorize execution. Natural-language aliases such as `yes`, `no`, `approve`, `deny`, `approved`, and `denied` are intentionally not bound to this canary owner-approval path.

## Canary Command Sequence

### No-Owner-Approval Branch

```text
HOT_UPDATE_CANARY_REQUIREMENT_CREATE <job_id> <result_id>
HOT_UPDATE_CANARY_EVIDENCE_CREATE <job_id> <canary_requirement_id> passed <observed_at> [reason...]
HOT_UPDATE_CANARY_SATISFACTION_AUTHORITY_CREATE <job_id> <canary_requirement_id>
HOT_UPDATE_CANARY_GATE_CREATE <job_id> <canary_satisfaction_authority_id>
HOT_UPDATE_GATE_PHASE <job_id> <hot_update_id> validated
HOT_UPDATE_GATE_PHASE <job_id> <hot_update_id> staged
HOT_UPDATE_GATE_EXECUTE <job_id> <hot_update_id>
HOT_UPDATE_GATE_RELOAD <job_id> <hot_update_id>
HOT_UPDATE_OUTCOME_CREATE <job_id> <hot_update_id>
HOT_UPDATE_PROMOTION_CREATE <job_id> <outcome_id>
```

After gate creation, `hot_update_gate_identity` must show `canary_ref=<canary_satisfaction_authority_id>` and empty `approval_ref`.

After outcome creation, `hot_update_outcome_identity` must show copied `canary_ref` and empty `approval_ref`.

After promotion creation, `promotion_identity` must show copied `canary_ref` and empty `approval_ref`.

### Owner-Approved Branch

```text
HOT_UPDATE_CANARY_REQUIREMENT_CREATE <job_id> <result_id>
HOT_UPDATE_CANARY_EVIDENCE_CREATE <job_id> <canary_requirement_id> passed <observed_at> [reason...]
HOT_UPDATE_CANARY_SATISFACTION_AUTHORITY_CREATE <job_id> <canary_requirement_id>
HOT_UPDATE_OWNER_APPROVAL_REQUEST_CREATE <job_id> <canary_satisfaction_authority_id>
HOT_UPDATE_OWNER_APPROVAL_DECISION_CREATE <job_id> <owner_approval_request_id> granted [reason...]
HOT_UPDATE_CANARY_GATE_CREATE <job_id> <canary_satisfaction_authority_id> <owner_approval_decision_id>
HOT_UPDATE_GATE_PHASE <job_id> <hot_update_id> validated
HOT_UPDATE_GATE_PHASE <job_id> <hot_update_id> staged
HOT_UPDATE_GATE_EXECUTE <job_id> <hot_update_id>
HOT_UPDATE_GATE_RELOAD <job_id> <hot_update_id>
HOT_UPDATE_OUTCOME_CREATE <job_id> <hot_update_id>
HOT_UPDATE_PROMOTION_CREATE <job_id> <outcome_id>
```

After gate creation, `hot_update_gate_identity` must show `canary_ref=<canary_satisfaction_authority_id>` and `approval_ref=<owner_approval_decision_id>`.

After outcome creation, `hot_update_outcome_identity` must show copied `canary_ref` and copied `approval_ref`.

After promotion creation, `promotion_identity` must show copied `canary_ref` and copied `approval_ref`.

### Rejected Owner Approval Sequence

```text
HOT_UPDATE_OWNER_APPROVAL_DECISION_CREATE <job_id> <owner_approval_request_id> rejected [reason...]
```

This is a terminal blocker for canary gate creation and execution authority. Do not pass the rejected decision id to `HOT_UPDATE_CANARY_GATE_CREATE`.

## Canary Status Checklist

Run:

```text
STATUS <job_id>
```

Inspect these sections as the canary path progresses:

- `hot_update_canary_requirement_identity`: requirement exists, is valid/configured, and refers to the expected candidate result and policy.
- `hot_update_canary_evidence_identity`: selected evidence exists, is valid/configured, and records `evidence_state=passed`.
- `hot_update_canary_satisfaction_identity`: no-owner branch should show `satisfaction_state=satisfied`; owner-approved branch should show `satisfaction_state=waiting_owner_approval`.
- `hot_update_canary_satisfaction_authority_identity`: no-owner branch should show `state=authorized`; owner-approved branch should show `state=waiting_owner_approval`.
- `hot_update_owner_approval_request_identity`: owner-approved branch should show `state=requested`.
- `hot_update_owner_approval_decision_identity`: owner-approved branch should show `decision=granted`; rejected branch should show `decision=rejected` and stop.
- `hot_update_gate_identity`: canary-derived gates must show `canary_ref`; owner-approved gates must also show `approval_ref`.
- `hot_update_outcome_identity`: canary-derived outcomes must show copied `canary_ref`; owner-approved outcomes must also show copied `approval_ref`.
- `promotion_identity`: canary-derived promotions must show copied `canary_ref`; owner-approved promotions must also show copied `approval_ref`.
- `rollback_identity`: generic recovery records, if any.
- `rollback_apply_identity`: generic rollback-apply records, if any.
- `runtime_pack_identity.active`: active runtime-pack pointer and `reload_generation`.
- `runtime_pack_identity.last_known_good`: current LKG pointer and basis.

Do not claim canary path evidence exists unless the relevant status identity surfaces configured valid records.

## Canary Guard Behavior

Canary-derived gates are guarded by readiness checks before:

- phase advancement to `validated` or `staged`
- pointer switch
- reload/apply

Guard failures fail closed and should preserve runtime/source state. The guard revalidates selected evidence, fresh satisfaction, fresh promotion eligibility, source ref linkage, active pointer context, rollback target, present LKG pointer validity, and owner approval decision `granted` when required.

## Canary Outcome And Promotion Behavior

Outcome creation does not re-run canary readiness after terminal execution. It records the terminal gate result:

- successful canary-derived outcomes remain `outcome_kind = hot_updated`
- failed canary-derived outcomes remain `outcome_kind = failed`
- `canary_applied` is not used by this path

Promotion creation does not re-run canary readiness after a successful terminal outcome.

Outcome and promotion records preserve audit lineage by copying `canary_ref` and `approval_ref`. Missing `canary_ref` on a canary-derived outcome or promotion is not normal. Missing `approval_ref` is normal only for the no-owner branch.

## Canary Rollback, Rollback-Apply, And LKG

Rollback and rollback-apply remain generic recovery flows. LKG recertification remains generic after promotion.

Rollback, rollback-apply, and LKG should not be blocked by stale canary evidence or owner-approval drift after execution. Operators can follow promotion -> outcome -> gate refs for canary and owner approval lineage.

## Canary Operator Warnings

- Do not use the eligible-only candidate promotion decision path for canary-required results.
- Do not use natural-language approval as owner approval authority.
- Do not supply an owner approval decision ID for the no-owner branch.
- Do not attempt canary gate creation with a rejected owner approval decision.
- Do not treat missing `canary_ref` or `approval_ref` on canary-derived outcome/promotion records as normal.
- Do not claim canary path evidence exists unless the relevant status identity shows configured valid records.

## Canary Troubleshooting

| Symptom | Likely cause | Operator action |
| --- | --- | --- |
| Stale satisfaction | Newer or changed evidence no longer yields expected satisfaction. | Reinspect `hot_update_canary_satisfaction_identity`; record fresh valid evidence or stop. |
| Stale promotion eligibility | Candidate result or policy no longer evaluates to the expected canary state. | Reinspect candidate result and policy; do not force gate execution. |
| Missing selected evidence | Authority references evidence that does not load or no longer matches. | Stop; recreate the authority only after valid evidence exists. |
| Missing owner approval decision | Owner-required branch has no committed decision. | Create a decision with exact `granted` or stop if rejected. |
| Rejected owner approval decision | Decision is terminal `rejected`. | Stop; do not create or execute a canary gate. |
| Mismatched approval refs | Decision copied refs do not match the authority/request chain. | Stop and inspect owner approval request/decision identity. |
| Active pointer drift | Active pack no longer matches the expected baseline or switched candidate context. | Stop and inspect `runtime_pack_identity.active`. |
| Missing rollback target | Candidate rollback target or pack is absent. | Stop; repair candidate/runtime-pack lineage before gate execution. |
| Invalid present LKG pointer | LKG pointer exists but points to invalid/missing pack data. | Stop and repair or recertify through valid generic flow. |
| Divergent duplicate record | Existing deterministic record differs from requested replay. | Stop; inspect the existing record and do not overwrite. |
| Command run against wrong job ID | `<job_id>` does not match active/persisted job context. | Use the active mission job id from `STATUS <job_id>`. |

## Eligible-Only Command Sequence

### 1. Record Or Select The Gate

For eligible-only candidate promotion decisions, prefer the decision-derived gate command:

```text
HOT_UPDATE_GATE_FROM_DECISION <job_id> <promotion_decision_id>
```

`HOT_UPDATE_GATE_RECORD` remains supported for operator-controlled hot-update workflows that already have an explicit hot-update id and candidate pack id:

```text
HOT_UPDATE_GATE_RECORD <job_id> <hot_update_id> <candidate_pack_id>
```

Expected responses:

- `Recorded hot-update gate job=<job_id> hot_update=<hot_update_id>.`
- `Selected hot-update gate job=<job_id> hot_update=<hot_update_id>.`

Status check:

```text
STATUS <job_id>
```

Confirm:

- `hot_update_gate_identity.state = configured`
- gate has expected `hot_update_id`
- gate has expected `candidate_pack_id`
- gate has expected `previous_active_pack_id`
- gate has rollback target and reload metadata

### 2. Record Execution Readiness

When deploy-lock or quiesce evidence is required, record bounded execution readiness before pointer switch or reload/apply:

```text
HOT_UPDATE_EXECUTION_READY <job_id> <hot_update_id> <ttl_seconds> [reason...]
```

Status check:

```text
STATUS <job_id>
```

Confirm:

- readiness evidence exists for the expected hot-update id
- readiness has not expired before phase advancement, pointer switch, or reload/apply

### 3. Validate And Stage

```text
HOT_UPDATE_GATE_PHASE <job_id> <hot_update_id> validated
HOT_UPDATE_GATE_PHASE <job_id> <hot_update_id> staged
```

Expected responses:

- `Advanced hot-update gate job=<job_id> hot_update=<hot_update_id> phase=validated.`
- `Advanced hot-update gate job=<job_id> hot_update=<hot_update_id> phase=staged.`
- replay may return `Selected...` when the requested phase is already selected

Status check:

```text
STATUS <job_id>
```

Confirm:

- gate state reaches `validated`, then `staged`
- `phase_updated_at` is present
- `phase_updated_by = operator`
- active runtime pack is unchanged
- last-known-good is unchanged

### 4. Execute Pointer Switch

```text
HOT_UPDATE_GATE_EXECUTE <job_id> <hot_update_id>
```

Expected responses:

- `Executed hot-update gate job=<job_id> hot_update=<hot_update_id>.`
- `Selected hot-update gate execution job=<job_id> hot_update=<hot_update_id>.`

Status check:

```text
STATUS <job_id>
```

Confirm:

- gate state is `reloading`
- `runtime_pack_identity.active.active_pack_id = <candidate_pack_id>`
- `runtime_pack_identity.active.previous_active_pack_id` names the prior active pack
- `reload_generation` changed only as part of this pointer switch
- `runtime_pack_identity.last_known_good.pack_id` still names the previous LKG

### 5. Reload/Apply

```text
HOT_UPDATE_GATE_RELOAD <job_id> <hot_update_id>
```

Expected responses:

- `Executed hot-update reload/apply job=<job_id> hot_update=<hot_update_id>.`
- `Selected hot-update reload/apply job=<job_id> hot_update=<hot_update_id>.`

Status check:

```text
STATUS <job_id>
```

Confirm one of:

- success: gate state is `reload_apply_succeeded`
- failure: gate state is `reload_apply_failed` and `failure_reason` is present
- interrupted/unknown: gate state is `reload_apply_recovery_needed`

Also confirm:

- active pointer is not switched again
- `reload_generation` is not incremented again
- no outcome is created automatically
- no promotion is created automatically
- LKG is not recertified automatically

### 6. Resolve Recovery-Needed

If the gate is `reload_apply_recovery_needed`, choose one path.

Retry:

```text
HOT_UPDATE_GATE_RELOAD <job_id> <hot_update_id>
```

Terminal failure:

```text
HOT_UPDATE_GATE_FAIL <job_id> <hot_update_id> <reason>
```

Expected terminal-failure responses:

- `Resolved hot-update terminal failure job=<job_id> hot_update=<hot_update_id>.`
- `Selected hot-update terminal failure job=<job_id> hot_update=<hot_update_id>.`

Status check:

```text
STATUS <job_id>
```

Confirm:

- retry success reaches `reload_apply_succeeded`
- retry failure reaches `reload_apply_failed`
- terminal failure stores `failure_reason = operator_terminal_failure: <reason>`
- terminal failure requires the same reason for replay

### 7. Create Outcome

```text
HOT_UPDATE_OUTCOME_CREATE <job_id> <hot_update_id>
```

Expected responses:

- `Created hot-update outcome job=<job_id> hot_update=<hot_update_id>.`
- `Selected hot-update outcome job=<job_id> hot_update=<hot_update_id>.`

Status check:

```text
STATUS <job_id>
```

Confirm:

- `hot_update_outcome_identity.state = configured`
- outcome id is `hot-update-outcome-<hot_update_id>`
- successful gate creates `outcome_kind = hot_updated`
- failed gate creates `outcome_kind = failed`
- failed outcome reason copies gate failure detail
- no promotion is created automatically

### 8. Create Promotion

```text
HOT_UPDATE_PROMOTION_CREATE <job_id> <outcome_id>
```

Expected responses:

- `Created hot-update promotion job=<job_id> outcome=<outcome_id>.`
- `Selected hot-update promotion job=<job_id> outcome=<outcome_id>.`

Status check:

```text
STATUS <job_id>
```

Confirm:

- `promotion_identity.state = configured`
- promotion id is `hot-update-promotion-<hot_update_id>`
- `promoted_pack_id` matches the outcome candidate pack
- `previous_active_pack_id` matches the gate previous active pack
- `outcome_id` links to the successful outcome
- `hot_update_id` links to the gate
- LKG is not recertified automatically

### 9. Recertify Last-Known-Good

```text
HOT_UPDATE_LKG_RECERTIFY <job_id> <promotion_id>
```

Expected responses:

- `Recertified hot-update last-known-good job=<job_id> promotion=<promotion_id>.`
- `Selected hot-update last-known-good job=<job_id> promotion=<promotion_id>.`

Status check:

```text
STATUS <job_id>
```

Confirm:

- `runtime_pack_identity.last_known_good.state = configured`
- `runtime_pack_identity.last_known_good.pack_id = promotion.promoted_pack_id`
- `runtime_pack_identity.last_known_good.basis = hot_update_promotion:<promotion_id>`
- `runtime_pack_identity.last_known_good.verified_at` is present
- active pointer is unchanged
- `reload_generation` is unchanged
- promotion record is unchanged
- outcome record is unchanged
- gate record is unchanged

## Generic Recovery Command Index

Rollback, rollback-apply, and LKG recovery remain generic. Use these commands only when the operator is intentionally entering the generic recovery flow:

```text
ROLLBACK_RECORD <job_id> <promotion_id> <rollback_id>
ROLLBACK_APPLY_RECORD <job_id> <rollback_id> <apply_id>
ROLLBACK_APPLY_PHASE <job_id> <apply_id> <phase>
ROLLBACK_APPLY_EXECUTE <job_id> <apply_id>
ROLLBACK_APPLY_RELOAD <job_id> <apply_id>
ROLLBACK_APPLY_FAIL <job_id> <apply_id> [reason...]
HOT_UPDATE_LKG_RECERTIFY <job_id> <promotion_id>
```

## Response Patterns

- `Recorded...`: a durable gate/workflow record was created.
- `Advanced...`: a gate phase changed.
- `Executed...`: an execution step ran or converged.
- `Resolved...`: an operator terminal-failure resolution was applied.
- `Created...`: a ledger record was created.
- `Recertified...`: `last_known_good_pointer.json` was updated.
- `Selected...`: the requested state or deterministic record already existed and was selected idempotently.
- empty response plus error: the command failed closed.

## Replay And Idempotence

- Replaying `HOT_UPDATE_GATE_RECORD` selects the existing matching gate.
- Replaying phase commands is safe only for compatible current state; invalid progression fails closed.
- Replaying `HOT_UPDATE_GATE_EXECUTE` after pointer switch does not increment `reload_generation` again.
- Replaying `HOT_UPDATE_GATE_RELOAD` after terminal convergence selects compatible existing reload/apply state.
- Replaying `HOT_UPDATE_GATE_FAIL` requires the same reason; a different reason fails closed.
- Replaying `HOT_UPDATE_OUTCOME_CREATE` selects the deterministic matching outcome.
- Replaying `HOT_UPDATE_PROMOTION_CREATE` selects the deterministic matching promotion.
- Replaying `HOT_UPDATE_LKG_RECERTIFY` selects the deterministic recertified LKG using the stored verification timestamp.
- Divergent duplicate outcomes, promotions, or LKG state fail closed.

## Common Fail-Closed Cases

- Wrong `<job_id>` does not match the active or persisted mission job.
- Missing gate blocks phase, execute, reload, and outcome creation.
- Non-adjacent phase progression rejects.
- Pointer switch rejects unless the gate is staged and pack linkage is valid.
- Reload/apply failure does not create an outcome or promotion automatically.
- `reload_apply_recovery_needed` does not retry or fail without an explicit command.
- Terminal failure requires non-empty `<reason>`.
- Terminal failure replay with a different reason rejects.
- Outcome creation rejects non-terminal gates.
- Failed outcome creation rejects empty gate failure detail.
- Promotion creation rejects failed or non-`hot_updated` outcomes.
- LKG recertification rejects missing promotion.
- LKG recertification rejects promotion without `outcome_id`.
- LKG recertification rejects linked outcome missing or not `hot_updated`.
- LKG recertification rejects active pointer missing or mismatched.
- LKG recertification rejects missing current LKG.
- LKG recertification rejects unrelated current LKG.
- LKG recertification rejects divergent existing promoted-pack LKG.

## Preserved Invariants

- No automatic outcome creation.
- No automatic promotion creation.
- No automatic LKG recertification.
- No LKG recertification directly from a gate.
- No LKG recertification directly from an outcome without a promotion.
- No active pointer mutation after the pointer switch.
- No `reload_generation` mutation after the pointer switch.
- No manual outcome fields.
- No manual promotion fields.
- No manual LKG fields.
- No manual active pointer fields.
- No manual reload generation fields.
- No natural-language owner approval binding for canary-required gates.
- No eligible-only candidate promotion decision for canary-required results.
- No canary readiness re-run from outcome or promotion creation after terminal execution.
- No canary-specific rollback, rollback-apply, or LKG policy in this path.
- No automation or orchestration is implied by this runbook.

## Practical Live Checklist

1. Run `STATUS <job_id>` before starting and identify the current active and LKG packs.
2. Choose the eligible-only path or the canary-required path from the candidate result and policy.
3. For canary-required results, create requirement/evidence/authority and owner approval only when the branch requires it.
4. Create the hot-update gate through the selected path.
5. Validate and stage.
6. Execute the pointer switch.
7. Run reload/apply.
8. If recovery-needed, explicitly retry or terminally fail.
9. Create the outcome only after a terminal gate state exists.
10. Create the promotion only from a successful `hot_updated` outcome.
11. Recertify LKG only from the promotion.
12. Run `STATUS <job_id>` after each major step and verify the expected read-model fields.
