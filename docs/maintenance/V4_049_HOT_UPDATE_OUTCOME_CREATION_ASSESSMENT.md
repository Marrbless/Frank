# V4-049 Hot-Update Outcome Creation Assessment

## Current Branch / HEAD / Tags

- Branch: `frank-v4-049-hot-update-outcome-creation-assessment`
- HEAD: `d32ef62e74f571c94cc637057fa98fb3428ba44d`
- Tags at HEAD:
  - `frank-v4-048-hot-update-gate-observability-read-model`

## Repo Baseline

- `git status --short --branch` at slice start was clean:
  - `## frank-v4-049-hot-update-outcome-creation-assessment`
- Baseline `/usr/local/go/bin/go test -count=1 ./...` passed when rerun outside the sandbox.
- The first sandboxed baseline failed because the sandbox made `/home/omar/.cache/go-build` read-only and blocked loopback socket binding for `httptest`.

## Scope

This is a docs-only assessment for the smallest safe future implementation slice that creates `HotUpdateOutcomeRecord` entries from terminal hot-update gate states.

No code, tests, commands, storage records, pointer behavior, reload generation behavior, or last-known-good behavior are changed in V4-049.

## Existing Hot-Update Gate Authority

The committed `HotUpdateGateRecord` is the current workflow authority for hot-update gate state. It already stores:

- `hot_update_id`
- `candidate_pack_id`
- `previous_active_pack_id`
- `rollback_target_pack_id`
- `target_surfaces`
- `surface_classes`
- `reload_mode`
- `compatibility_contract_ref`
- `prepared_at`
- `phase_updated_at`
- `phase_updated_by`
- `state`
- `decision`
- `failure_reason`

The completed hot-update state-machine slices now cover:

- gate creation/selection
- phase progression
- pointer switch
- reload/apply convergence
- `reload_apply_in_progress -> reload_apply_recovery_needed` normalization
- retry from `reload_apply_recovery_needed`
- terminal failure from `reload_apply_recovery_needed`
- read-only exposure of terminal failure detail and transition metadata

The terminal gate states relevant to the outcome-creation question are therefore durable and observable before any outcome ledger slice is added.

## Existing HotUpdateOutcomeRecord Contract

`HotUpdateOutcomeRecord` already exists as an append-only ledger record with these fields:

- `outcome_id`
- `hot_update_id`
- optional `candidate_id`
- optional `run_id`
- optional `candidate_result_id`
- optional `candidate_pack_id`
- `outcome_kind`
- optional `reason`
- optional `notes`
- `outcome_at`
- `created_at`
- `created_by`

Existing storage behavior:

- `StoreHotUpdateOutcomeRecord` normalizes records.
- `StoreHotUpdateOutcomeRecord` validates required fields.
- `StoreHotUpdateOutcomeRecord` validates gate linkage.
- Exact duplicate replay is idempotent when the stored record deeply equals the proposed record.
- Divergent duplicate with the same `outcome_id` fails closed.
- Listing is deterministic through the existing store JSON list path.
- Read models already expose outcome identity through `hot_update_outcome_identity`.

Existing linkage validation:

- `hot_update_id` must reference an existing `HotUpdateGateRecord`.
- `candidate_pack_id`, when present, must reference an existing runtime pack and match the gate candidate pack.
- `candidate_id`, when present, must reference an existing improvement candidate linked to the gate candidate pack and hot-update ID when set.
- `run_id`, when present, must reference an existing improvement run linked to the gate hot-update ID and candidate pack.
- `candidate_result_id`, when present, must reference an existing candidate result linked to the gate hot-update ID, run, candidate, and candidate pack when those refs are present.

## Eligible Terminal Gate States

The smallest safe outcome-creation slice should only accept committed gate states that are already terminal for the current hot-update state machine:

- `reload_apply_succeeded`
- `reload_apply_failed`

`reload_apply_succeeded` should map to an outcome that records successful hot-update application.

Recommended outcome kind:

- `hot_updated`

`reload_apply_failed` should map to an outcome that records a failed hot-update application.

Recommended outcome kind:

- `failed`

All non-terminal or non-final gate states should fail closed, including:

- `prepared`
- `validated`
- `staged`
- `reloading`
- `reload_apply_in_progress`
- `reload_apply_recovery_needed`
- any broader states not selected by the future slice

`reload_apply_recovery_needed` must not create an outcome because it represents an unknown outcome requiring explicit operator resolution.

## Required Source Authority

The source authority for the future creation helper should be the existing committed `HotUpdateGateRecord`.

The helper should:

- load the gate by `hot_update_id`
- normalize and validate the gate
- validate existing gate linkage
- require the gate state to be one of the selected terminal states
- derive outcome fields from the gate and already-committed linked records only
- never infer a terminal state from runtime pointer state, candidate records, or absence of records

The active runtime-pack pointer may be inspected only if a future policy explicitly chooses to validate attribution for successful outcome creation. It must not be mutated.

## Deterministic Outcome Identity

The safest first implementation should use a deterministic `outcome_id` derived from the gate identity rather than requiring a caller-provided arbitrary ID.

Recommended identity:

- `hot-update-outcome-<hot_update_id>`

Alternative identity:

- `outcome-<hot_update_id>`

The selected format should be documented and tested. The key property is that repeated calls for the same terminal gate derive the same `outcome_id`.

## Deterministic Linkage

The future helper should always populate:

- `hot_update_id` from the gate
- `candidate_pack_id` from `HotUpdateGateRecord.CandidatePackID`
- `outcome_kind` from the terminal gate state
- `outcome_at` from `HotUpdateGateRecord.PhaseUpdatedAt`
- `created_at` from the operator/system transition timestamp passed to the helper
- `created_by` from the caller, probably `operator` for the direct control path or `system` if purely internal

The helper should populate these optional refs only when they can be deterministically resolved from existing committed records:

- `candidate_id`
- `run_id`
- `candidate_result_id`

Because `HotUpdateGateRecord` does not itself store candidate/run/result IDs, the first implementation should not invent those refs. It can either:

- leave them empty in V4-050, relying on `hot_update_id` and `candidate_pack_id` linkage; or
- resolve them only when exactly one coherent existing candidate/run/result chain is linked to the same `hot_update_id` and candidate pack.

The smaller and safer V4-050 choice is to leave candidate/run/result refs empty unless an existing deterministic resolver already exists and is trivial to reuse.

## Failure Detail Mapping

For failed terminal gates:

- source field: `HotUpdateGateRecord.FailureReason`
- outcome field: `HotUpdateOutcomeRecord.Reason`

The reason should preserve deterministic terminal-failure detail exactly when present, including:

- `operator_terminal_failure: <reason>`

For convergence failures, the concrete stored `failure_reason` should be copied as the outcome reason.

For failed gates with empty `failure_reason`, the helper should either:

- fail closed because terminal failure detail is missing; or
- use a deterministic fallback such as `hot update reload/apply failed`.

The stricter first implementation should fail closed on empty `failure_reason` for `reload_apply_failed`, because V4-048 already exposes this field and terminal-failure/convergence paths are expected to populate it.

For successful terminal gates:

- `reason` can be empty, or a deterministic fixed reason can be used.

Recommended first implementation:

- successful reason: `hot update reload/apply succeeded`

This avoids an empty reason while keeping output deterministic.

## Idempotence And Replay Rules

The future helper should preserve the existing append-only outcome contract:

- Exact replay with the same derived `outcome_id` and identical fields returns idempotently.
- If the existing outcome deeply equals the derived record, the helper returns already-resolved/no-change.
- If an outcome exists for the same `outcome_id` but diverges in any field, fail closed.
- If any existing outcome already links to the same `hot_update_id` but has a different `outcome_id`, fail closed unless a future migration explicitly permits legacy aliases.
- If the gate state has changed after outcome creation and the derived record would differ, fail closed rather than rewriting the outcome.

Recommended changed flag semantics:

- created outcome: `changed = true`
- exact replay / already-present same outcome: `changed = false`
- divergent duplicate: error
- non-terminal gate: error

## Already-Present Outcome Handling

An already-present matching outcome counts as already resolved.

The future helper should load/list existing outcomes before writing so it can distinguish:

- no outcome exists for the gate: create derived outcome
- exactly matching derived outcome exists: return idempotently
- any divergent existing outcome for the same identity or hot-update ID exists: fail closed

This keeps outcome creation append-only and prevents duplicate ledgers for the same terminal gate.

## Command Surface Recommendation

V4-050 should not add a public operator command unless necessary.

Preferred first implementation:

- add a small missioncontrol helper such as `EnsureHotUpdateOutcomeFromTerminalGate(root, hotUpdateID, createdBy, createdAt)`
- add tests around the helper
- optionally expose it through the existing `TaskState` direct operator path only if there is already a clear adjacent operator workflow need

Because V4-049 is assessing storage/ledger creation, not operator ergonomics, the smallest V4-050 should stay in `missioncontrol` unless the implementation proves an existing command path is required.

## Non-Goals For V4-050

- no promotion creation
- no last-known-good mutation
- no active runtime-pack pointer mutation
- no `reload_generation` mutation
- no new gate creation
- no terminal-state inference beyond the committed gate
- no retry behavior
- no terminal-failure behavior
- no automatic success/failure inference from runtime pointer state
- no rollback creation
- no policy/authorization expansion

## Risks To Control

- Duplicating outcomes for the same hot update would weaken ledger authority.
- Inferring terminal state from pointer state would bypass the committed gate authority.
- Creating promotion records in the same slice would combine two ledgers and make replay semantics harder to reason about.
- Mutating last-known-good in the same slice would turn outcome recording into recertification, which needs separate evidence and policy.
- Copying candidate/run/result refs without deterministic resolution could create misleading linkage.

## Recommended Smallest V4-050 Slice

Implement `HotUpdateOutcomeRecord` creation from terminal committed hot-update gates in `missioncontrol`.

Recommended exact scope:

- add `EnsureHotUpdateOutcomeFromTerminalGate(root, hotUpdateID, createdBy, createdAt)`
- accept only `reload_apply_succeeded` and `reload_apply_failed`
- derive deterministic `outcome_id` from `hot_update_id`
- set `hot_update_id`, `candidate_pack_id`, `outcome_kind`, `reason`, `outcome_at`, `created_at`, and `created_by`
- leave `candidate_id`, `run_id`, and `candidate_result_id` empty unless deterministic existing linkage is already trivial and unambiguous
- use the existing `StoreHotUpdateOutcomeRecord` append-only semantics
- treat exact replay as idempotent
- fail closed on divergent duplicates or non-terminal gates
- prove no promotion, pointer, reload generation, last-known-good, or gate mutations occur

Recommended tests:

- successful terminal gate creates `hot_updated` outcome
- failed terminal gate creates `failed` outcome with copied failure detail
- exact replay is idempotent and does not rewrite the file
- divergent duplicate fails closed
- existing outcome for same hot update but different identity fails closed
- non-terminal states reject outcome creation
- no `PromotionRecord` is created
- active runtime-pack pointer bytes remain unchanged
- `reload_generation` remains unchanged
- `last_known_good_pointer.json` bytes remain unchanged
- gate record bytes remain unchanged

Recommendation:

- V4-050 is appropriate if the project wants to proceed from state-machine completion into the outcome ledger.
- Keep V4-050 ledger-only. Promotion and last-known-good recertification should remain separate later slices.
