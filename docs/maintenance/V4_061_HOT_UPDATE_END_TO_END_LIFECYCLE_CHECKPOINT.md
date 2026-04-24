# V4-061 Hot-Update End-to-End Lifecycle Checkpoint

## Scope

V4-061 is a docs-only checkpoint for the completed V4 hot-update lifecycle from committed gate authority through last-known-good recertification.

This slice does not change Go code, tests, commands, TaskState wrappers, `active_pointer.json`, `last_known_good_pointer.json`, `reload_generation`, hot-update outcomes, promotions, hot-update gates, rollback behavior, policy, or V4-062 scope.

## Current Baseline

- Branch: `frank-v4-061-hot-update-end-to-end-lifecycle-checkpoint`
- HEAD: `006767b9365d909c31dda54be1dc8e69a666be0c`
- Tag at HEAD:
  - `frank-v4-060-hot-update-lkg-recertify-control-entry`
- Starting worktree:
  - clean
- Starting validation:
  - `/usr/local/go/bin/go test -count=1 ./...` passed

## Completed Hot-Update Lifecycle

### Gate Storage / Read-Model / Control

The lifecycle starts with committed `HotUpdateGateRecord` storage keyed by `hot_update_id`. The gate is the workflow authority for candidate pack identity, previous active pack identity, rollback target, reload mode, target surfaces, state, decision, failure detail, and phase transition metadata.

Operator status exposes hot-update gate identity through the existing read model. Direct control exists through `HOT_UPDATE_GATE_RECORD <job_id> <hot_update_id> <candidate_pack_id>`, which creates or selects the committed gate without mutating active pointer state.

### Phase Progression

Phase progression is explicit and operator-driven:

- `prepared -> validated`
- `validated -> staged`

The `HOT_UPDATE_GATE_PHASE <job_id> <hot_update_id> <phase>` command advances only valid adjacent phases and preserves active pointer, reload generation, LKG pointer, outcomes, and promotions.

### Pointer Switch

The pointer switch executes only from a staged gate through:

`HOT_UPDATE_GATE_EXECUTE <job_id> <hot_update_id>`

This switches `active_pointer.json` to the candidate pack, records the previous active pack, and increments `reload_generation`. It does not mutate `last_known_good_pointer.json`, create outcomes, create promotions, or infer success.

### Reload/Apply Convergence

Reload/apply convergence is controlled through:

`HOT_UPDATE_GATE_RELOAD <job_id> <hot_update_id>`

The command advances reload/apply work and records terminal success or failure on the committed gate. Successful convergence records `reload_apply_succeeded`. Failed convergence records `reload_apply_failed` with deterministic failure detail from the reload/apply path. It does not mutate the active pointer after the switch, does not increment `reload_generation`, and does not mutate LKG.

### Recovery-Needed Normalization

Unknown or interrupted `reload_apply_in_progress` state can normalize to `reload_apply_recovery_needed`. This creates a stable operator-visible recovery state without inferring success or terminal failure.

Normalization preserves:

- active pointer bytes
- `reload_generation`
- `last_known_good_pointer.json`
- hot-update outcome records
- promotion records
- gate identity

### Retry From Recovery-Needed

The same `HOT_UPDATE_GATE_RELOAD` command supports explicit retry from `reload_apply_recovery_needed`.

Retry reuses the existing committed gate as the sole workflow authority. It does not create a new gate or apply record, does not automatically infer success, and does not mutate LKG. Retry either converges to `reload_apply_succeeded` or records a bounded failure.

### Terminal-Failure Resolution

Terminal failure from recovery-needed is controlled through:

`HOT_UPDATE_GATE_FAIL <job_id> <hot_update_id> <reason>`

The operator must provide non-empty reason text. The stored failure detail is deterministic:

`operator_terminal_failure: <reason>`

Exact replay with the same reason is idempotent. A different reason after terminal failure fails closed. The command preserves active pointer state, `reload_generation`, LKG pointer, outcomes, promotions, and gate identity outside the explicit state/failure transition.

### Gate Observability Polish

The gate read model now surfaces terminal failure detail and relevant transition metadata, including phase update actor/time fields, so operators can inspect status without reading raw store files.

This was read-only polish. It did not add commands, states, storage records, outcome creation, promotion creation, pointer mutation, reload generation mutation, or LKG mutation.

### Outcome Creation Helper

Missioncontrol can create a deterministic `HotUpdateOutcomeRecord` from an existing committed terminal hot-update gate.

Eligible gate states are:

- `reload_apply_succeeded`
- `reload_apply_failed`

The deterministic outcome id is:

`hot-update-outcome-<hot_update_id>`

Successful gates map to `outcome_kind = hot_updated` with reason `hot update reload/apply succeeded`. Failed gates map to `outcome_kind = failed` and copy the gate failure detail. Exact replay is idempotent. Divergent duplicates fail closed. The helper does not create promotions, mutate pointers, mutate reload generation, mutate LKG, create gates, or mutate gates.

### Outcome Creation Control Entry

The outcome helper is exposed through:

`HOT_UPDATE_OUTCOME_CREATE <job_id> <hot_update_id>`

The TaskState wrapper validates active or persisted job context, resolves the mission store root, derives timestamps through the existing helper, calls the missioncontrol helper with actor `operator`, emits audit action `hot_update_outcome_create`, and returns the helper `changed` flag.

Status exposes created outcomes through `hot_update_outcome_identity`.

### Promotion Creation Helper

Missioncontrol can create a deterministic `PromotionRecord` from an existing committed successful hot-update outcome.

Eligible outcome kind is:

`hot_updated`

Failed, blocked, discarded, approval-required, cold-restart, canary, rollback, abort, unknown, or future outcome kinds reject. The deterministic promotion id is:

`hot-update-promotion-<hot_update_id>`

The promotion copies promoted pack from the outcome candidate pack, previous active pack from the originating gate, hot-update id, outcome id, optional candidate/run/result refs when present, and reason `hot update outcome promoted`. Exact replay is idempotent. Divergent duplicates fail closed. The helper does not mutate active pointer, reload generation, LKG, outcomes, or gates.

### Promotion Creation Control Entry

The promotion helper is exposed through:

`HOT_UPDATE_PROMOTION_CREATE <job_id> <outcome_id>`

The TaskState wrapper follows the existing direct-control pattern, emits audit action `hot_update_promotion_create`, and returns the missioncontrol `changed` flag. Status exposes created promotions through `promotion_identity`.

### LKG Recertification Helper

Missioncontrol can recertify a promoted hot-update runtime pack as last-known-good from an existing committed promotion:

`RecertifyLastKnownGoodFromPromotion(root, promotionID, verifiedBy, verifiedAt)`

The promotion is the direct source authority. It must have `outcome_id`, the linked outcome must exist, the linked outcome must be `hot_updated`, and the existing promotion/gate/outcome linkage must remain valid.

The active pointer is a guard:

`active_pointer.active_pack_id == promotion.promoted_pack_id`

The helper writes only `runtime_packs/last_known_good_pointer.json`. The deterministic pointer maps:

- `pack_id = promotion.promoted_pack_id`
- `basis = hot_update_promotion:<promotion_id>`
- `verified_at = helper input`
- `verified_by = helper input`
- `rollback_record_ref = hot_update_promotion:<promotion_id>`

Current LKG handling is fail-closed:

- missing or invalid current LKG rejects
- current LKG equal to `promotion.previous_active_pack_id` may be replaced
- exact deterministic replay returns unchanged
- current LKG already on the promoted pack with divergent metadata rejects at helper level
- current LKG on any unrelated pack rejects

The helper does not mutate active pointer, reload generation, outcomes, promotions, gates, or rollback state.

### LKG Recertification Control Entry

The helper is exposed through:

`HOT_UPDATE_LKG_RECERTIFY <job_id> <promotion_id>`

The TaskState wrapper validates active or persisted job context, resolves the mission store root, derives the initial timestamp through the existing helper, calls missioncontrol with actor `operator`, emits audit action `hot_update_lkg_recertify`, and returns the helper `changed` flag.

Command replay is idempotent. Because `verified_at` is part of the deterministic LKG pointer, the wrapper may reuse the stored LKG `verified_at` only when the current LKG already has:

- `pack_id = promotion.promoted_pack_id`
- `basis = hot_update_promotion:<promotion_id>`
- `rollback_record_ref = hot_update_promotion:<promotion_id>`

It does not accept manual timestamp or manual LKG fields. Status exposes the recertified result through `runtime_pack_identity.last_known_good`.

## End-to-End Completion Assessment

The V4 hot-update end-to-end lifecycle is now complete enough to stop widening.

The lane has a coherent explicit path:

1. record/select a committed gate
2. advance gate phases
3. switch the active pointer to the candidate pack
4. converge reload/apply
5. normalize interrupted reload/apply to recovery-needed
6. retry or terminally fail from recovery-needed
7. expose gate details in status
8. create an outcome from terminal gate authority
9. create a promotion from successful outcome authority
10. recertify LKG from promotion authority
11. confirm via existing status read models

The lifecycle now separates state-machine transitions, ledger creation, promotion selection, and LKG recertification into explicit operator-driven steps. That separation is the right boundary: further changes should be selected as polish, policy, automation, or UX slices rather than as more core lifecycle widening.

## Preserved Invariants

- No automatic retry.
- No automatic success inference.
- No automatic failure inference outside explicit terminal-failure command.
- No automatic outcome creation from gate state changes.
- No automatic promotion creation from outcome creation.
- No automatic LKG recertification from promotion creation.
- No LKG recertification directly from gates.
- No LKG recertification directly from outcomes without promotions.
- No active pointer mutation after the pointer-switch step.
- No `reload_generation` mutation after the pointer-switch step.
- No last-known-good mutation before explicit recertification.
- No promotion mutation during LKG recertification.
- No gate mutation during outcome, promotion, or LKG ledger/control steps.

## Remaining Deferred Areas

### Read-Model Polish Around LKG Metadata

`active_pointer.json` has optional `last_known_good_pack_id` metadata, while `last_known_good_pointer.json` is the canonical LKG pointer mutated by recertification. After LKG recertification, status can show active pointer metadata that still reflects the previous LKG while `runtime_pack_identity.last_known_good` correctly reflects the recertified pack.

This should be treated as read-model polish, not a reason to mutate `active_pointer.json` during recertification.

### Dedicated LKG Recertification Ledger

The current LKG pointer write is deterministic and names the promotion in both basis and rollback record ref. A dedicated append-only recertification ledger may be useful later for richer audit, evidence, approval, or policy workflows.

It is not required for the completed V4 lifecycle.

### Broader Approval / Policy / Authorization

The lane is operator-driven through existing direct command checks. Broader approval requirements, authorization tiers, policy gates, or multi-actor recertification remain deferred.

Those should be selected explicitly and not inferred from successful lifecycle completion.

### Automation / Orchestration

The current lifecycle is explicit and stepwise. Automation could later chain outcome creation, promotion creation, or LKG recertification, but that would introduce policy decisions.

Any orchestration should remain opt-in and should preserve the existing fail-closed helper semantics.

### Rollback Integration Polish

Rollback surfaces already exist separately. Future polish may clarify how hot-update promotion and LKG recertification records should be referenced by rollback UX and status surfaces.

No new rollback behavior is required to consider the hot-update lifecycle complete.

### Operator UX Docs / Help Text

The direct command surface is now larger. Operator-facing help text or runbook documentation could improve discoverability and reduce command sequencing mistakes.

This should be documentation/UX work, not a behavior change.

## Recommended Smallest Next Slice

The smallest useful next slice is operator UX documentation/help text for the completed hot-update lifecycle.

Suggested scope:

- document the command sequence from gate record through LKG recertification
- document expected status fields after each major step
- document fail-closed cases and replay behavior
- keep it docs/read-only or help-text-only
- do not add new workflow states
- do not add automation
- do not mutate pointers, ledgers, gates, or policy

If code work is preferred instead, the smallest safe code slice is read-model polish clarifying `active_pointer.last_known_good_pack_id` metadata versus canonical `last_known_good_pointer.json`. That slice should remain read-only and must not mutate active pointer state.
