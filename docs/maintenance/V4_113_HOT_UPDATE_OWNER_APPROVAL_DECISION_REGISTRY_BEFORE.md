# V4-113 Hot-Update Owner Approval Decision Registry Before

## Before-State Gap From V4-112

V4-112 assessed the terminal owner approval authority path after a durable `HotUpdateOwnerApprovalRequestRecord` exists. The repo had:

- immutable owner approval request records under `runtime_packs/hot_update_owner_approval_requests/`
- `HOT_UPDATE_OWNER_APPROVAL_REQUEST_CREATE <job_id> <canary_satisfaction_authority_id>`
- status/read-model exposure through `hot_update_owner_approval_request_identity`
- canary satisfaction authority records that can be `waiting_owner_approval`

The missing surface was a durable terminal owner approval decision ledger. Existing runtime `ApprovalRequestRecord` and `ApprovalGrantRecord` remained job/runtime-step scoped and did not carry stable canary satisfaction/request linkage. They were not safe as canonical hot-update owner approval authority.

## Required Slice

V4-113 must add a missioncontrol registry/read-model skeleton only:

- one terminal owner approval decision record family
- deterministic decision IDs derived from `owner_approval_request_id`
- validation, storage, load, list, and create/select helpers
- read-only status identity `hot_update_owner_approval_decision_identity`

## Explicit Non-Goals

V4-113 must not add direct commands, `TaskState` wrappers, natural-language approval binding, runtime approval mutation, separate grant/rejection registries, expiration records, candidate promotion decisions, hot-update gates, outcomes, promotions, rollbacks, rollback-apply records, runtime-pack pointer changes, last-known-good changes, reload/apply changes, or V4-114 work.
