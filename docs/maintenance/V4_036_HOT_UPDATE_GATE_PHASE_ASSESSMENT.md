## V4-036 Hot-Update Gate Phase Progression Assessment

### Current checkpoint facts

- branch: `frank-v4-036-hot-update-gate-phase-assessment`
- HEAD: `0bb16cf81a622ecde1c8a8ed143ed47afe8bd6eb`
- tags at HEAD:
  - `frank-v4-035-hot-update-gate-control-entry`
- ahead/behind upstream:
  - `434 0`
- repo green:
  - `git status --short --branch` showed a clean worktree before this memo
  - `go test -count=1 ./...` passed before this memo

### Current hot-update gate truth

Committed hot-update gate truth currently lives in `internal/missioncontrol/hot_update_gate_registry.go`.

Today the gate record already stores:

- `hot_update_id`
- `candidate_pack_id`
- `previous_active_pack_id`
- `rollback_target_pack_id`
- `target_surfaces`
- `surface_classes`
- `reload_mode`
- `compatibility_contract_ref`
- optional bounded references:
  - `eval_evidence_refs`
  - `smoke_check_refs`
  - `canary_ref`
  - `approval_ref`
  - `budget_ref`
- `prepared_at`
- `state`
- `decision`
- `failure_reason`

Current gate creation is bounded and non-executing:

- V4-035 added `EnsureHotUpdateGateRecordFromCandidate(...)`
- V4-035 added direct operator control:
  - `HOT_UPDATE_GATE_RECORD <job_id> <hot_update_id> <candidate_pack_id>`
- V4-034 added read-only operator status exposure:
  - `hot_update_gate_identity`

Current downstream durable records already split terminal or external consequences out of the gate:

- `HotUpdateOutcomeRecord`
- `PromotionRecord`
- `RollbackRecord`

That separation is important: it means the gate record can remain the workflow envelope for staging/progression, while actual applied consequences stay in separate records.

### 1. Minimal phase/state progression that should exist

The smallest safe non-applying progression is:

1. `prepared`
2. `validated`
3. `staged`

Interpretation:

- `prepared`
  - gate exists
  - committed linkage is valid
  - candidate pack and rollback target are identified
- `validated`
  - gate has enough committed review/evidence linkage to say the candidate is eligible for a later decision
  - still no apply, reload, promotion, rollback, or outcome record
- `staged`
  - candidate remains selected and intentionally held as the current gate-ready candidate
  - still non-applying

The existing enum contains many later execution-adjacent states such as `quiescing`, `reloading`, `smoke_testing`, `canarying`, `committed`, `rolled_back`, `failed`, and `aborted`. Those should not be widened in the first non-applying phase slice. They either imply execution orchestration or overlap with separate outcome/promotion/rollback records.

### 2. What stays on the gate record vs separate records

The gate record should keep:

- candidate identity and linkage
- pre-apply compatibility/reload shape
- bounded workflow phase before any live mutation
- operator decision posture while still pre-apply
- pre-apply evidence references
- bounded failure reason only for gate-local validation or staging rejection

The gate record should not become the source of truth for:

- actual apply success or reload execution evidence
- terminal hot-update outcome classification
- promotion details
- rollback details
- runtime pointer mutation history

Those belong in separate durable records:

- `HotUpdateOutcomeRecord` for the first committed outcome decision
- `PromotionRecord` for actual promotion linkage
- `RollbackRecord` for actual rollback linkage

Reason:

- that split is already established in the repo
- it keeps the gate as the pre-apply workflow envelope
- it avoids collapsing all hot-update history into one mutable record

### 3. Narrowest truthful control surface for advancing gate phase

The narrowest truthful control surface is the existing operator direct-command path:

- `AgentLoop.ProcessDirect(...)`
- `TaskState` wrapper
- `missioncontrol` durable helper

Recommended first control entry:

- `HOT_UPDATE_GATE_PHASE <job_id> <hot_update_id> <phase>`

Reason:

- this matches the already-accepted rollback-apply pattern
- it reuses the same operator control plane added in V4-018 and extended through V4-035
- it keeps phase authority in `missioncontrol`, not in the loop parser

The durable transition helper should live in `internal/missioncontrol`, and the direct command should only invoke it.

### 4. Invariants before, during, and after each transition

Before any transition:

- the gate record must already exist
- the gate linkage must still validate:
  - `candidate_pack_id`
  - `previous_active_pack_id`
  - `rollback_target_pack_id`
- the job binding must match the active or persisted mission-control context

During each transition:

- no active runtime-pack pointer mutation
- no `reload_generation` change
- no `last_known_good_pointer.json` mutation
- no creation of outcome, promotion, or rollback records
- no apply or reload mechanics

After each successful transition:

- gate linkage remains valid
- the gate remains the same durable authority record
- the phase change is durable
- unchanged fields remain byte-stable except for explicit transition metadata

If transition metadata is added, the required invariants should also include:

- `phase_updated_at` present
- `phase_updated_by` present
- `phase_updated_at >= prepared_at`

### 5. Idempotence / replay rules required

Minimum safe replay model:

- same-phase replay is idempotent and returns “selected” / unchanged
- adjacent forward transition is allowed once
- skipped transitions fail closed
- regressive transitions fail closed

Recommended first order:

- `prepared -> validated`
- `validated -> staged`

Recommended explicit rule:

- `staged -> staged` is idempotent
- `validated -> validated` is idempotent
- `prepared -> prepared` is idempotent
- `prepared -> staged` is invalid
- `staged -> validated` is invalid

This is the same bounded adjacent progression rule already accepted for rollback-apply.

### 6. Failure modes needing explicit representation vs fail-closed rejection

Explicit representation needed in the first slice:

- none beyond the existing `state` progression and optional gate-local transition metadata

Fail-closed rejection is sufficient for:

- missing gate record
- invalid requested next phase
- missing linked runtime packs
- mismatched or stale linkage
- non-adjacent transition attempts
- regressive transition attempts

`failure_reason` does not need to become a general transition-error ledger in the first slice. Using it for every rejected command would blur the boundary between operator command failure and durable workflow state.

For the first slice, rejected transitions should remain command errors, not durable state mutations.

### 7. Smallest safe implementation slice after this assessment

Recommended next implementation slice:

1. add durable transition metadata to `HotUpdateGateRecord`
   - `phase_updated_at`
   - `phase_updated_by`
2. normalize older gate records by backfilling those fields from `prepared_at`
3. add a bounded durable helper in `missioncontrol`
   - adjacent progression only
   - idempotent same-phase replay
4. add one direct operator control entry
   - `HOT_UPDATE_GATE_PHASE <job_id> <hot_update_id> <phase>`
5. limit supported phases in that slice to:
   - `validated`
   - `staged`

That is the smallest safe slice because it:

- adds durable workflow progression
- does not imply apply behavior
- does not overload outcome/promotion semantics
- stays aligned with the rollback-apply progression pattern already accepted on this branch line

### 8. Explicit non-goals for that first slice

- no apply or reload execution
- no promotion behavior
- no rollback behavior
- no outcome record creation
- no active pointer mutation
- no `reload_generation` change
- no `last_known_good_pointer.json` mutation
- no evaluator execution
- no scoring behavior
- no autonomy changes
- no provider or channel changes
- no execution-adjacent gate phases such as:
  - `quiescing`
  - `reloading`
  - `smoke_testing`
  - `canarying`
  - `committed`
  - `rolled_back`
  - `failed`
  - `aborted`

### Recommendation

The smallest justified next slice is:

- bounded gate phase progression for `prepared -> validated -> staged`
- with explicit transition metadata
- using the existing direct operator command path

That closes the remaining obvious workflow gap in hot-update gate state without crossing into apply/reload or duplicating responsibility already owned by outcome, promotion, or rollback records.
