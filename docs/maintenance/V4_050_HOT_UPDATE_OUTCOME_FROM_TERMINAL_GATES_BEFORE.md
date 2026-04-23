# V4-050 Hot-Update Outcome From Terminal Gates Before

## Before-State Gap

After V4-049, hot-update gates had durable terminal states and read-model visibility, but there was no bounded missioncontrol helper that created a `HotUpdateOutcomeRecord` directly from an existing committed terminal `HotUpdateGateRecord`.

The outcome ledger storage already existed and enforced append-only duplicate behavior, but callers still had to construct outcome records by hand. That left the terminal-gate-to-outcome mapping undocumented in code and untested as a single deterministic operation.

## Existing Authority

The committed `HotUpdateGateRecord` was already the workflow authority for:

- `hot_update_id`
- `candidate_pack_id`
- terminal state
- `failure_reason`
- `phase_updated_at`

The active runtime-pack pointer, reload generation, and last-known-good pointer were not outcome authority for this slice.

## Required Implementation Boundary

V4-050 was constrained to ledger-only missioncontrol behavior:

- no public operator command
- no TaskState wrapper
- no `PromotionRecord` creation
- no active runtime-pack pointer mutation
- no `reload_generation` mutation
- no `last_known_good_pointer.json` mutation
- no hot-update gate mutation
- no new hot-update gate creation
- no terminal-state inference from pointer state
- no V4-051 work
