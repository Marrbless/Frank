# GC3-FIX-001 Zoho Timestamp Stabilization Before

Date: 2026-04-21

- Branch: `frank-v3-foundation`
- HEAD: `beed364f6317bccbb021d5d7fe8ec7706ce3f6f9`

## git status --short --branch

```text
## frank-v3-foundation
```

## Exact failing tests observed

- Historical/intermittent failures in:
  - `TestFrankZohoCampaignSendPrepareFinalizeAndReplaySafety`
  - `TestPrepareFrankZohoCampaignFollowUpBlocksDuplicateUnresolvedAction`
- Current reconcile run results:
  - `go test -count=1 -run TestFrankZohoCampaignSendPrepareFinalizeAndReplaySafety -v ./internal/agent/tools`
    - passed
  - `go test -count=1 -run TestPrepareFrankZohoCampaignFollowUpBlocksDuplicateUnresolvedAction -v ./internal/agent/tools`
    - passed
  - `go test -count=1 ./internal/agent/tools`
    - passed outside sandbox after rerun to avoid `httptest` socket restrictions

## Exact producer paths

- Reply work item creation/claim/update path
  - `internal/agent/tools/taskstate.go`
    - `SyncFrankZohoCampaignInboundReplies`
      - derives missing reply work items via `LoadMissingCommittedCampaignZohoEmailReplyWorkItems(..., time.Now().UTC())`
    - `PrepareFrankZohoCampaignSend`
      - samples `now := time.Now().UTC()` after sync
      - claims reply work items with that separate timestamp
    - `claimFrankZohoCampaignReplyWorkItem`
    - `transitionFrankZohoCampaignReplyWorkItemResponded`
    - `transitionFrankZohoCampaignReplyWorkItemOnFailure`

- Outbound prepared/sent/verified path
  - `internal/agent/tools/taskstate.go`
    - `PrepareFrankZohoCampaignSend`
      - samples `now := time.Now().UTC()` for prepared/finalize flow
    - `RecordFrankZohoCampaignSend`
      - rebuilds prepared action with `time.Now().UTC()`
      - builds sent action with a second `time.Now().UTC()`
    - `RecordFrankZohoCampaignSendFailure`
      - rebuilds prepared action with `time.Now().UTC()`
      - builds failed action with another `time.Now().UTC()`
      - reopens reply work item on failure with another `time.Now().UTC()`

## Exact validator paths

- `internal/missioncontrol/campaign_zoho_email_outbound_actions.go`
  - `ValidateCampaignZohoEmailOutboundAction`
  - guards:
    - `sent_at must be on or after prepared_at`
    - `verified_at must be on or after sent_at`

- `internal/missioncontrol/campaign_zoho_email_reply_work_items.go`
  - `ValidateCampaignZohoEmailReplyWorkItem`
  - guard:
    - `updated_at must be on or after created_at`

## Exact fix plan

- Add one taskstate-local clock hook to make ordering behavior executable in tests.
- Stabilize timestamp sampling in producer paths only:
  - use one stable timestamp for each logical Zoho operation path
  - thread that timestamp through sync/prepare/claim/finalize paths instead of resampling wall clock
- Clamp transition timestamps to the predecessor timestamp floor where needed so producer-side invariants remain valid even if the wall clock moves backward between related operations.
- Add narrow regression coverage in `internal/agent/tools/frank_zoho_send_email_test.go` for:
  - outbound prepare/send/replay timestamp ordering under simulated clock regression
  - duplicate follow-up blocking path under simulated clock regression

## Exact non-goals

- do not change missioncontrol validators unless producer-only stabilization proves insufficient
- do not weaken or delete tests
- do not change unrelated TaskState behavior
- do not continue GC3 structural cleanup
- do not implement V4
- do not add dependencies
- do not commit
