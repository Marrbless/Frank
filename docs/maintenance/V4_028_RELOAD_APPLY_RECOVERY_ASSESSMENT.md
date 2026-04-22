# V4-028 Reload-Apply Recovery Assessment

## Current checkpoint facts

- Repository root: `/mnt/d/pbot/picobot`
- Branch: `frank-v4-028-rollback-apply-recovery-assessment`
- HEAD: `c58103643140338694ac5e85e257f3207f326721`
- Tags at HEAD: `frank-v4-027-rollback-reload-apply-skeleton`
- Ahead/behind `upstream/main`: `426 0`
- Baseline worktree state at assessment start: clean
- Baseline validation at assessment start: `go test -count=1 ./...` passed

## Current rollback / rollback-apply / pointer-switch truth

### Durable rollback authority

- Rollback records remain the sole durable authority for rollback linkage.
- A rollback-apply record links to exactly one committed rollback record through `rollback_id`.
- The rollback record remains the durable source of truth for:
  - `from_pack_id`
  - `target_pack_id`
  - rollback creation metadata

### Durable rollback-apply authority

- Rollback-apply records remain the sole durable authority for rollback-apply workflow identity and phase.
- The current durable rollback-apply phase set is:
  - `recorded`
  - `validated`
  - `ready_to_apply`
  - `pointer_switched_reload_pending`
  - `reload_apply_in_progress`
  - `reload_apply_succeeded`
  - `reload_apply_failed`
- `activation_state` is still constrained to `unchanged`.
- `execution_error` is only valid for terminal `reload_apply_failed`.

### Pointer-switch truth from V4-025

- `ExecuteRollbackApplyPointerSwitch(...)` already performed the first execution slice before V4-027:
  - active runtime-pack pointer switches to the rollback target pack
  - `reload_generation` increments exactly once
  - `update_record_ref` becomes `rollback_apply:<apply_id>`
  - `last_known_good_pointer.json` remains unchanged
- This means a rollback-apply record entering V4-027 reload/apply already has the active pointer durably switched.

### Reload/apply truth from V4-027

- `ExecuteRollbackApplyReloadApply(...)` only starts from `pointer_switched_reload_pending`.
- It validates:
  - linked rollback record exists
  - active pointer matches rollback target
  - `update_record_ref == rollback_apply:<apply_id>`
  - `previous_active_pack_id == rollback.from_pack_id`
  - target pack record exists
- It then:
  1. writes phase `reload_apply_in_progress`
  2. runs the bounded convergence helper
  3. writes terminal phase `reload_apply_succeeded` or `reload_apply_failed`
- Current bounded convergence is intentionally small:
  - re-resolve active runtime-pack pointer
  - resolve active runtime-pack record
  - confirm active pack still equals rollback target
  - confirm `update_record_ref` still equals `rollback_apply:<apply_id>`
- There is no separate durable success receipt beyond the terminal rollback-apply phase write.
- There is no startup reconciliation path and no recovery command yet.

## What state on disk is authoritative after a crash in `reload_apply_in_progress`?

After a crash or restart with a persisted rollback-apply record still in `reload_apply_in_progress`, the authoritative durable facts on disk are only:

- the rollback-apply record still exists with:
  - `phase == reload_apply_in_progress`
  - `activation_state == unchanged`
  - empty `execution_error`
  - most recent phase actor/timestamp metadata pointing at the start of execution
- the active runtime-pack pointer is whatever was already durably switched by V4-025
- `reload_generation` is whatever value resulted from the earlier pointer-switch slice
- `update_record_ref` remains `rollback_apply:<apply_id>` if it has not been overwritten
- `last_known_good_pointer.json` remains unchanged from before rollback apply
- the linked rollback record and referenced runtime-pack records remain the durable linkage authority

There is currently no stronger durable outcome authority than those files.

## What exact evidence can distinguish success, failure, or unknown?

### Evidence that reload actually succeeded before crash

Currently there is no new durable evidence that can prove reload/apply succeeded if the process crashed before the terminal `reload_apply_succeeded` phase write.

The bounded V4-027 convergence helper does not emit a separate success marker, boot receipt, or completion ledger. If the process crashes after the convergence logic effectively succeeded but before the success phase write, disk state is still only:

- active pointer already switched
- rollback-apply phase `reload_apply_in_progress`
- no `execution_error`

That state is indistinguishable from other incomplete outcomes.

### Evidence that reload actually failed before crash

Currently there is no durable evidence that can prove reload/apply failed if the process crashed before the terminal `reload_apply_failed` phase write.

If convergence encountered a failure but the process died before the failure phase and `execution_error` were durably written, the on-disk state is still only:

- active pointer already switched
- rollback-apply phase `reload_apply_in_progress`
- no `execution_error`

That is also indistinguishable from other incomplete outcomes.

### Evidence that outcome is unknown

`reload_apply_in_progress` with no terminal phase and no `execution_error` must therefore be treated as an unknown-outcome state, not as success and not as failure.

That conclusion is grounded in the current code path: there is no durable intermediate receipt between “execution started” and “terminal result recorded”.

## Recovery choice analysis

### Option: mark unknown as failed

This is safe in the sense that it does not over-claim success, but it collapses two materially different realities:

- actual apply failure
- crash after effective convergence but before terminal success write

That would create false negatives and would encode an outcome the system cannot prove.

### Option: retry automatically

This is not the smallest safe first recovery choice.

Automatic retry would need stronger proof that repeating the bounded convergence path is safe in every interrupted state. Today there is no distinct durable marker separating:

- “only pre-terminal bookkeeping happened”
- “effective convergence already happened once”
- “some future broader reload side effect partially happened”

For the current V4-027 skeleton, auto-retry is tempting because the helper is small, but it would bake in recovery semantics before there is durable evidence for future broader reload/apply behavior.

### Option: require explicit operator advancement without recording recovery state

This is safe but weak. It leaves the durable model in `reload_apply_in_progress`, which is a stale execution phase rather than a truthful post-crash state.

That makes operator inspection and later replay rules ambiguous.

### Recommended first recovery choice: record a new explicit recovery-needed state

The smallest safe recovery model is:

- treat persisted `reload_apply_in_progress` after restart as unknown outcome
- add a new explicit durable state such as `reload_apply_recovery_needed`
- transition into that state through a small recovery reconciliation helper
- require explicit operator action for any later retry or failure decision

This is the smallest truthful model because it:

- does not claim success without proof
- does not claim failure without proof
- does not silently retry
- gives later control surfaces a durable, inspectable state that means “pointer already switched; reload outcome unknown; operator or later policy must decide”

## Replay / idempotence rules required on recovery

### Recovery reconciliation rules

- If phase is `reload_apply_in_progress` on startup or explicit recovery inspection:
  - validate that linked rollback record still exists
  - validate that active pointer still matches `rollback_apply:<apply_id>` and the rollback target pack
  - if linkage still matches, transition to `reload_apply_recovery_needed`
  - if linkage no longer matches, fail closed into a terminal invalid-or-failed path with explicit error detail

### Replay rules after recovery

- Replay from `reload_apply_succeeded` remains idempotent and must not redo side effects.
- Replay from `reload_apply_failed` should continue to fail closed unless a later explicit retry slice broadens policy.
- Replay from `reload_apply_recovery_needed` should not happen automatically.
- Any retry from `reload_apply_recovery_needed` should require an explicit operator advancement that either:
  - re-enters `reload_apply_in_progress` and retries bounded convergence, or
  - records terminal failure if the operator chooses not to retry.

### Why not retry directly from `reload_apply_in_progress` after restart?

Because restart needs to normalize stale in-flight state first. `reload_apply_in_progress` means “execution started in a previous process”; after a crash, that state is no longer actively progressing. A recovery-needed normalization step is the cleanest way to make replay explicit and auditable.

## Invariants that must hold during recovery

### Active runtime-pack pointer

- The active pointer is authoritative for which pack is currently selected.
- Recovery must not switch the pointer again.
- Recovery must only inspect whether the pointer still matches:
  - rollback target pack
  - `update_record_ref == rollback_apply:<apply_id>`

### Reload generation

- `reload_generation` must remain unchanged during recovery.
- It was already incremented exactly once during the pointer-switch slice.
- Recovery must not increment it again.

### Rollback-apply phase

- Recovery must preserve phase monotonicity.
- `reload_apply_in_progress` should not persist forever after restart.
- The smallest truthful forward move is to a dedicated recovery-needed state rather than back to `pointer_switched_reload_pending`.

### Execution error

- `execution_error` should remain empty for unknown-outcome recovery-needed state.
- `execution_error` should only be populated when a terminal failure has been durably concluded.
- Recovery should not invent failure text for an unknown outcome.

### Last-known-good pointer

- `last_known_good_pointer.json` must remain unchanged during recovery.
- Recovery is about clarifying the rollback-apply workflow state, not recertifying pack health.

## Smallest safe implementation slice after this assessment

The smallest safe next slice is:

1. Extend rollback-apply phase with a new durable state:
   - `reload_apply_recovery_needed`
2. Add a small recovery reconciliation helper in `internal/missioncontrol` that:
   - consumes `reload_apply_in_progress`
   - validates linked rollback record and current pointer linkage
   - records `reload_apply_recovery_needed` when the outcome is unknown but linkage is still coherent
   - records terminal failure with explicit error only when linkage is provably invalid
3. Add the smallest control or inspection entry needed to invoke that reconciliation explicitly.
4. Do not retry reload/apply in that slice.
5. Do not mutate pointer, `reload_generation`, or `last_known_good_pointer.json` in that slice.

This keeps the first recovery slice focused on truthful state normalization, not on retry policy.

## Acceptance tests required for that next slice

- persisted `reload_apply_in_progress` with coherent pointer linkage normalizes to `reload_apply_recovery_needed`
- recovery normalization does not switch the active pointer again
- recovery normalization does not increment `reload_generation` again
- recovery normalization leaves `last_known_good_pointer.json` unchanged
- invalid linkage while recovering fails closed with durable error detail
- terminal `reload_apply_succeeded` remains idempotent on replay
- terminal `reload_apply_failed` remains closed to replay unless a later retry slice explicitly changes policy

## Explicit non-goals for that first recovery slice

- no automatic reload/apply retry
- no automatic success inference
- no automatic last-known-good recertification
- no second active-pointer mutation
- no second `reload_generation` increment
- no promotion behavior changes
- no evaluator, scoring, autonomy, provider, or channel changes
- no broader reload engine redesign
- no policy for repeated recovery loops beyond the first truthful normalization step

## Recommended conclusion

Persisted `reload_apply_in_progress` is currently an unknown-outcome crash state. The smallest safe recovery model is to recognize that explicitly in the durable workflow state rather than silently retrying, silently failing, or silently succeeding. The next implementation slice should therefore add a dedicated recovery-needed phase and a narrow reconciliation path that normalizes stale in-flight records without mutating pointer or last-known-good state.
