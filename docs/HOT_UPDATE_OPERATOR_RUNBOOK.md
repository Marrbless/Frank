# Hot-Update Operator Runbook

This runbook covers the completed V4 hot-update lifecycle from gate creation through last-known-good recertification.

Use it during live operator work. Keep the sequence explicit. Later ledger and recertification steps are not automatic side effects of earlier commands.

## Placeholders

- `<job_id>`: active mission job id
- `<hot_update_id>`: durable hot-update workflow id
- `<candidate_pack_id>`: runtime pack selected for hot update
- `<outcome_id>`: usually `hot-update-outcome-<hot_update_id>`
- `<promotion_id>`: usually `hot-update-promotion-<hot_update_id>`
- `<reason>`: non-empty operator reason text

## Command Sequence

### 1. Record Or Select The Gate

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

### 2. Validate And Stage

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

### 3. Execute Pointer Switch

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

### 4. Reload/Apply

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

### 5. Resolve Recovery-Needed

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

### 6. Create Outcome

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

### 7. Create Promotion

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

### 8. Recertify Last-Known-Good

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
- No automation or orchestration is implied by this runbook.

## Practical Live Checklist

1. Run `STATUS <job_id>` before starting and identify the current active and LKG packs.
2. Record the gate.
3. Validate and stage.
4. Execute the pointer switch.
5. Run reload/apply.
6. If recovery-needed, explicitly retry or terminally fail.
7. Create the outcome only after a terminal gate state exists.
8. Create the promotion only from a successful `hot_updated` outcome.
9. Recertify LKG only from the promotion.
10. Run `STATUS <job_id>` after each major step and verify the expected read-model fields.
