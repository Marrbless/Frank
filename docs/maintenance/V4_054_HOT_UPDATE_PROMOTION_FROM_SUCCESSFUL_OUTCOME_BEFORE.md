# V4-054 Hot-Update Promotion From Successful Outcome - Before

## Gap

After V4-052, operators could create a durable `HotUpdateOutcomeRecord` from a terminal hot-update gate, but the promotion ledger still had no bounded missioncontrol helper for deriving a `PromotionRecord` from that successful hot-update outcome.

The existing promotion store could persist promotion records, but callers had to assemble promotion identity, pack linkage, outcome linkage, timestamps, and reasons themselves. That left the hot-update lane without a single deterministic ledger-only path from:

1. committed successful hot-update outcome
2. originating committed hot-update gate
3. derived promotion ledger record

## Required Source Authority

The future helper must use the committed `HotUpdateOutcomeRecord` as source authority and load it by `outcome_id`. The originating `HotUpdateGateRecord` must be loaded through `outcome.hot_update_id` only to supply gate-owned linkage such as `previous_active_pack_id` and to verify candidate pack consistency.

The helper must not promote directly from a gate without an existing outcome.

## Expected Eligible Outcome

Only `outcome_kind = hot_updated` is eligible.

All other current outcome kinds must fail closed:

- `failed`
- `kept_staged`
- `discarded`
- `blocked`
- `approval_required`
- `cold_restart_required`
- `canary_applied`
- `promoted`
- `rolled_back`
- `aborted`
- unknown future kinds

## Non-Goals

This slice must not add a direct command, TaskState wrapper, active runtime-pack pointer mutation, `reload_generation` mutation, last-known-good pointer mutation, last-known-good recertification, outcome creation, gate mutation, automatic promotion directly from gate success, policy broadening, manual promotion IDs, manual pack refs, or V4-055 work.
