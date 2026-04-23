Branch: `frank-v4-044-hot-update-recovery-resolution-assessment`
HEAD: `83b44c47ba9f9afa5fc049d5a8be13b5f9bf1cec`
Tags at HEAD: `frank-v4-043-hot-update-recovery-needed`
Ahead/behind `upstream/main`: `442 0`
Baseline `go test -count=1 ./...`: passed

# V4-044 Hot-Update Recovery Resolution Assessment

## Current checkpoint facts

- V4-039 added the first hot-update pointer-switch execution slice. A committed staged gate can switch the active runtime-pack pointer to `candidate_pack_id`, increment `reload_generation`, stamp `update_record_ref=hot_update:<hot_update_id>`, and advance to `reloading`.
- V4-041 added bounded post-pointer convergence states on the gate: `reload_apply_in_progress`, `reload_apply_succeeded`, and `reload_apply_failed`.
- V4-043 added `reload_apply_recovery_needed` plus bounded normalization from persisted `reload_apply_in_progress` when crash/restart leaves the outcome unknown.
- The current hot-update workflow authority is still the committed `HotUpdateGateRecord`. No `HotUpdateOutcomeRecord` or `PromotionRecord` is created during pointer switch, convergence, or recovery normalization.
- The current operator execution surfaces already exist:
  - `HOT_UPDATE_GATE_EXECUTE <job_id> <hot_update_id>`
  - `HOT_UPDATE_GATE_RELOAD <job_id> <hot_update_id>`
- Current durable runtime evidence after the pointer switch remains:
  - active pointer on `candidate_pack_id`
  - `reload_generation` already incremented once
  - `update_record_ref=hot_update:<hot_update_id>`
  - `last_known_good_pointer.json` unchanged

## Current recovery truth

- `reload_apply_recovery_needed` means the system has enough durable evidence to know that:
  - the gate had already been selected and the active pointer had already switched earlier
  - the current active pointer is still attributed to the same `hot_update_id`
  - convergence outcome was not durably proven as success or terminal failure
- The gate remains the only truthful workflow identity for operator recovery handling.
- `HotUpdateOutcomeRecord` remains a separate terminal-result ledger and should not be fabricated to resolve recovery ambiguity.
- `PromotionRecord` remains a separate handoff ledger and should not be fabricated during recovery resolution.

## Recovery options

### Option 1: Explicit retry on the same `hot_update_id`

Description:
- Allow an operator to retry reload/apply convergence from `reload_apply_recovery_needed` using the same committed gate.
- Revalidate active-pointer attribution and gate linkage.
- Clear stale failure detail if present.
- Transition back into `reload_apply_in_progress`.
- Reuse the existing bounded convergence helper from V4-041.

Expected value:
- Smallest continuation of the existing state machine.
- Preserves one workflow identity and one durable gate record.
- Reuses the existing control surface and convergence path.
- Avoids inventing parallel authority or terminal semantics before necessary.

Risk:
- Retry still depends on the same bounded convergence path and therefore still inherits any crash ambiguity that later recovery slices may need to normalize again.
- Requires strict validation so that retry cannot run against a mismatched pointer attribution.

Confidence:
- High.

Why now or not now:
- This is the smallest safe first implementation slice. It directly resolves the recovery-needed state without broadening the authority model.

### Option 2: Explicit terminal failure from `reload_apply_recovery_needed`

Description:
- Allow an operator to mark the gate terminally failed from `reload_apply_recovery_needed`, with explicit operator-supplied reason text written durably on the gate.

Expected value:
- Gives operators a deterministic way to stop the workflow when retry is not desired.
- Produces a clear terminal gate state.

Risk:
- It is a policy decision, not the smallest continuation of the already-started execution path.
- It resolves ambiguity by operator declaration rather than by reusing the existing execution path.
- If added first, it may encourage terminal closure before the system supports the smallest retry path.

Confidence:
- Medium.

Why now or not now:
- Reasonable later slice, but not the smallest first slice. It broadens workflow resolution semantics before the simplest retry path is available.

### Option 3: Create a new gate/apply record instead of retry

Description:
- Treat recovery-needed as requiring a new durable workflow identity rather than continuing the existing one.

Expected value:
- Could isolate the retried attempt under a new identifier.

Risk:
- Breaks the current authority model, where the original `HotUpdateGateRecord` is the workflow authority for pointer switch and convergence.
- Introduces duplicate identities for the same already-switched pointer state.
- Requires additional linkage semantics between old and new records.
- Larger and less truthful than same-record retry.

Confidence:
- High that this is not the right first slice.

Why now or not now:
- Not now. This is materially broader than required and not justified by the current code or state-machine shape.

## Smallest safe first implementation slice

The smallest safe first implementation slice is:

- explicit retry on the same `hot_update_id` from `reload_apply_recovery_needed`
- using the existing committed `HotUpdateGateRecord` as the only workflow authority
- using the existing direct operator reload command as the control entry

This should allow:

- `reload_apply_recovery_needed -> reload_apply_in_progress -> reload_apply_succeeded`
- `reload_apply_recovery_needed -> reload_apply_in_progress -> reload_apply_failed`

This should not add:

- a new gate record
- a new outcome record
- a promotion record
- a second pointer switch
- a second `reload_generation` increment

## Narrowest truthful control surface

The narrowest truthful control surface is the existing operator direct-command path:

- `internal/agent/loop.go`
- `internal/agent/tools/taskstate.go`
- `internal/missioncontrol` durable transition helper

Recommended operator entry:

- keep using `HOT_UPDATE_GATE_RELOAD <job_id> <hot_update_id>`

Rationale:

- that command already truthfully means “run the bounded reload/apply convergence step for this gate”
- widening the existing helper to accept `reload_apply_recovery_needed` is smaller than creating a new command
- `missioncontrol` should remain the transition authority, with `TaskState` only acting as a thin wrapper

## Idempotence and replay rules

Required replay rules for the first retry slice:

1. Retry is allowed only from `reload_apply_recovery_needed`.
2. Retry must revalidate:
   - active pointer attribution still points at `hot_update:<hot_update_id>`
   - active pointer target still matches `candidate_pack_id`
   - gate linkage to `candidate_pack_id`, `previous_active_pack_id`, and `rollback_target_pack_id` remains coherent
3. On retry start:
   - clear stale durable failure detail on the gate
   - transition to `reload_apply_in_progress`
4. Then reuse the existing bounded convergence helper.
5. Replay after `reload_apply_succeeded` remains idempotent and must not redo side effects.
6. Replay after `reload_apply_failed` stays closed in the first retry slice.
7. Invalid attribution or broken linkage must fail closed without mutating pointer state or ledger state.

## Invariants that must remain unchanged during recovery resolution

The following must remain fixed during the first recovery-resolution slice:

- no second active pointer switch
- no second `reload_generation` increment
- no `last_known_good_pointer.json` mutation
- no `HotUpdateOutcomeRecord` creation
- no `PromotionRecord` creation
- no new gate creation
- no rollback-state changes

Authority boundaries must remain:

- `HotUpdateGateRecord` is the sole workflow authority for retrying the same hot update
- `HotUpdateOutcomeRecord` remains reserved for later terminal-result handling
- `PromotionRecord` remains reserved for later promotion handoff handling

## Error detail: preserve vs clear

Recommended error-detail policy:

- `reload_apply_recovery_needed` should preserve whatever durable ambiguity or prior failure detail is already present on the gate.
- Starting an explicit retry from `reload_apply_recovery_needed` should clear stale failure detail before re-entering `reload_apply_in_progress`.
- If the retried convergence fails again, write fresh failure detail on the gate.
- If a later explicit terminal-failure action is added, it should require operator-supplied reason text and write deterministic operator-authored failure detail rather than reusing stale convergence text.

## Acceptance tests for the first implementation slice

The first recovery-resolution implementation slice should add narrow tests for:

1. Happy-path retry from `reload_apply_recovery_needed` to `reload_apply_succeeded`.
2. Failure-path retry from `reload_apply_recovery_needed` to `reload_apply_failed`.
3. Stale durable failure detail cleared on retry start.
4. Existing `HOT_UPDATE_GATE_RELOAD <job_id> <hot_update_id>` command works for the retry path.
5. Replay after `reload_apply_succeeded` remains idempotent.
6. No second active pointer mutation occurs.
7. No second `reload_generation` increment occurs.
8. No `last_known_good_pointer.json` mutation occurs.
9. No `HotUpdateOutcomeRecord` or `PromotionRecord` is created.
10. Invalid starting phase still rejects.
11. Broken active-pointer attribution or gate linkage rejects without mutation.

## Explicit non-goals for the first implementation slice

- no new operator command
- no new gate or apply record
- no automatic retry
- no automatic success inference
- no automatic terminal-failure inference
- no terminal-failure operator action in the same slice
- no outcome creation
- no promotion creation
- no `last_known_good_pointer.json` mutation
- no second pointer switch
- no second `reload_generation` increment
- no rollback, evaluator, scoring, autonomy, provider, or channel changes

## Recommendation

Recommended next direction:

- implement explicit retry on the same `hot_update_id` from `reload_apply_recovery_needed`
- keep using `HOT_UPDATE_GATE_RELOAD <job_id> <hot_update_id>`
- widen only the durable `missioncontrol` helper plus the already-existing wrapper path as needed

Rationale:

- It is the smallest truthful continuation of the current hot-update state machine.
- It preserves the current authority model.
- It is smaller and safer than inventing a second workflow identity.
- It keeps terminal-failure policy as a later, clearly separate operator decision.
