# V4-047 Hot-Update State Machine Checkpoint

## Current Branch / HEAD / Tags

- Branch: `frank-v4-047-hot-update-state-machine-checkpoint-memo`
- HEAD: `d12f1ae086f9e74f4a36214237e60698fe0ccabc`
- Tags at HEAD:
  - `frank-v4-046-hot-update-terminal-failure-resolution`

## Repo Green Status

- `git status --short --branch` at checkpoint start was clean:
  - `## frank-v4-047-hot-update-state-machine-checkpoint-memo`
- Baseline `/usr/local/go/bin/go test -count=1 ./...` passed when rerun outside the sandbox.
- The first sandboxed baseline failed because the sandbox made `/home/omar/.cache/go-build` read-only and blocked loopback socket binding for `httptest`.

## Completed Hot-Update Slices

- `V4-034`: hot-update gate read model
- `V4-035`: hot-update gate operator control entry
- `V4-036`: hot-update gate phase progression assessment
- `V4-037`: hot-update gate phase control
- `V4-038`: hot-update pointer-switch execution assessment
- `V4-039`: active runtime-pack pointer switch to the hot-update candidate
- `V4-040`: reload/apply convergence assessment
- `V4-041`: bounded reload/apply convergence with durable success/failure recording
- `V4-042`: recovery-needed assessment for interrupted reload/apply
- `V4-043`: recovery-needed normalization from unknown in-progress reload/apply
- `V4-044`: recovery resolution assessment
- `V4-045`: explicit retry from `reload_apply_recovery_needed`
- `V4-046`: explicit terminal-failure resolution from `reload_apply_recovery_needed`

## Current State-Machine Capability

- Durable hot-update gate storage exists through committed `HotUpdateGateRecord` files keyed by `hot_update_id`.
- Read-only operator status exposes hot-update gate identity, candidate pack linkage, previous active pack linkage, rollback target linkage, reload mode, current state, and decision.
- Operator control supports:
  - gate creation or selection through `HOT_UPDATE_GATE_RECORD`
  - phase progression through `HOT_UPDATE_GATE_PHASE`
  - pointer switch execution through `HOT_UPDATE_GATE_EXECUTE`
  - reload/apply convergence through `HOT_UPDATE_GATE_RELOAD`
  - retry from `reload_apply_recovery_needed` through `HOT_UPDATE_GATE_RELOAD`
  - terminal-failure resolution from `reload_apply_recovery_needed` through `HOT_UPDATE_GATE_FAIL`

## State Progression Now Covered

- Gate storage and selection:
  - committed gate record is the workflow authority
  - replay of existing gate selection remains idempotent
- Phase progression:
  - `prepared -> validated -> staged`
  - adjacent progression is explicit and operator driven
- Pointer switch:
  - `staged -> reloading`
  - active runtime-pack pointer switches to the candidate pack
  - `reload_generation` increments only on the pointer switch
  - last-known-good pointer remains unchanged
- Reload/apply convergence:
  - `reloading -> reload_apply_in_progress -> reload_apply_succeeded`
  - convergence failure records `reload_apply_failed` with concrete failure detail
  - replay after `reload_apply_succeeded` remains idempotent
- Recovery-needed normalization:
  - interrupted or persisted `reload_apply_in_progress` can normalize to `reload_apply_recovery_needed`
  - normalization preserves pointer state and last-known-good state
- Retry from recovery-needed:
  - `reload_apply_recovery_needed -> reload_apply_in_progress`
  - retry reuses the same committed hot-update gate
  - retry success records `reload_apply_succeeded`
  - retry failure records `reload_apply_failed`
  - retry does not create new outcome, promotion, gate, or apply records
- Terminal-failure resolution from recovery-needed:
  - `reload_apply_recovery_needed -> reload_apply_failed`
  - reason text is required
  - failure detail is deterministic:
    - `operator_terminal_failure: <reason>`
  - exact replay with the same reason is idempotent
  - different reason after terminal failure fails closed
  - resolution does not mutate active pointer, `reload_generation`, or last-known-good state

## Boundary Invariants

- The hot-update gate record is now sufficient as the workflow authority through the bounded state-machine lifecycle.
- No retry from terminal `reload_apply_failed` exists.
- No automatic retry exists.
- No automatic success inference exists.
- No automatic failure inference exists outside the explicit operator terminal-failure command.
- No new gate or apply record is created during recovery resolution.
- No `HotUpdateOutcomeRecord` is created by the state-machine resolution slices.
- No `PromotionRecord` is created by the state-machine resolution slices.
- No last-known-good mutation or recertification is performed by the state-machine resolution slices.

## Is The State Machine Complete Enough To Stop Widening?

Yes. The hot-update state machine is complete enough to stop widening core workflow behavior.

The lane now has a coherent bounded lifecycle:

- identity and durable gate authority
- read-only gate status
- explicit operator control entry
- phase progression
- pointer switch
- reload/apply convergence
- recovery-needed normalization for unknown in-progress reload/apply
- explicit retry from recovery-needed
- explicit terminal failure from recovery-needed

The remaining gaps are important, but they are not missing state-machine transitions. They are ledger, observability, policy, and certification concerns that should be handled as separate explicit slices rather than by continuing to widen the state machine.

## Remaining Deferred Areas

### Richer Observability / Read-Model Polish

- Operator status does not yet expose all terminal-failure detail in a polished way.
- Transition metadata such as `phase_updated_at` and `phase_updated_by` may need clearer read-only presentation.
- This is read-only surface polish, not new workflow semantics.

### Outcome Ledger Creation

- There is still no slice that creates `HotUpdateOutcomeRecord` entries from terminal hot-update gate results.
- Outcome creation should be append-only and explicit.
- It should not be smuggled into retry, recovery, or terminal-failure resolution.

### Promotion Record Creation

- There is still no promotion handoff from successful hot-update terminal outcome to `PromotionRecord`.
- Promotion creation should be a distinct ledger/control slice with its own linkage and replay rules.
- It should not be inferred automatically from state-machine convergence without an explicit selected policy.

### Last-Known-Good Mutation / Recertification

- The state machine intentionally preserves `last_known_good_pointer.json` during pointer switch, reload/apply, recovery, retry, and terminal-failure resolution.
- Any last-known-good mutation should require a separate recertification model.
- That future model should define evidence requirements, timing, rollback semantics, and idempotence independently.

### Broader Policy / Authorization

- The current command surface is operator-driven but intentionally narrow.
- Broader approval policy, authorization rules, retry policy from terminal failure, or automatic orchestration remain out of scope.
- Those should be selected only by an explicit future checkpoint, not as a side effect of this state-machine work.

## Option Comparison

### 1. Stop Widening The Hot-Update State Machine

- Expected value:
  - high
  - preserves a disciplined, auditable lifecycle
  - avoids turning terminal states into policy ambiguity
- Risk:
  - low
  - operators may still need raw store inspection for some detail until read-model polish lands
- Recommendation:
  - yes
  - this is the correct boundary for core state-machine behavior

### 2. Add Read-Only Observability Polish

- Expected value:
  - medium
  - improves operator understanding of failure and transition context
  - lowest-risk adjacent slice if another hot-update slice is selected
- Risk:
  - low
  - output shape may widen and require tests for deterministic JSON/readout behavior
- Recommendation:
  - best smallest next slice if continuing in this area
  - should stay read-only

### 3. Add Outcome Ledger Creation

- Expected value:
  - high
  - starts the durable result ledger needed for later promotion decisions
- Risk:
  - medium
  - must avoid duplicating outcomes or inferring policy incorrectly
- Recommendation:
  - valid later slice
  - should follow a focused assessment or spec if linkage/replay rules are not already fully settled

### 4. Add Promotion Creation

- Expected value:
  - high eventually
  - connects successful hot-update outcomes to promotion identity
- Risk:
  - medium-high if done before outcome ledger rules are complete
- Recommendation:
  - do not do before outcome ledger creation is explicitly selected

### 5. Add Last-Known-Good Recertification

- Expected value:
  - high eventually
  - closes the operational loop after a hot update proves safe
- Risk:
  - high if mixed into state-machine or promotion slices
- Recommendation:
  - defer until evidence and recertification policy are explicit

## Recommended Smallest Next Slice

Recommend the smallest next slice as read-only observability/read-model polish for hot-update gates.

Suggested scope:

- expose deterministic terminal failure detail and transition metadata on the existing hot-update gate read model/status surface
- keep it read-only
- do not create outcomes
- do not create promotions
- do not mutate active pointer, `reload_generation`, or last-known-good state
- do not add new commands

If the project wants to pivot away from hot-update, that is also justified now because core state-machine widening is complete enough to stop. If it stays in hot-update for one more slice, observability polish is safer and smaller than outcome or promotion ledger work.
