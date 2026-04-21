# GC3 Block: Zoho Timestamp Flake Diagnosis

Date: 2026-04-21

- Branch: `frank-v3-foundation`
- HEAD: `2555730f39f1480bbd900ca13d4fc8e21fb26377`

## Reconciliation evidence

- `git status --short --branch`
  - `## frank-v3-foundation`
  - `?? docs/maintenance/garbage-day/GC3_BLOCK_ZOHO_REPLY_WORK_ITEM_DUPLICATE_DIAGNOSIS.md`
- `git rev-parse HEAD`
  - `2555730f39f1480bbd900ca13d4fc8e21fb26377`

## Requested test runs

- `go test -count=1 -run TestFrankZohoCampaignSendPrepareFinalizeAndReplaySafety -v ./internal/agent/tools`
  - passed
- same test repeated 5 times with `-count=1`
  - all 5 passed
- `go test -count=1 -run TestPrepareFrankZohoCampaignFollowUpBlocksDuplicateUnresolvedAction -v ./internal/agent/tools`
  - passed
- same test repeated 5 times with `-count=1`
  - all 5 passed
- `go test -count=1 ./internal/agent/tools`
  - passed

## Whether each failure reproduced

- `TestFrankZohoCampaignSendPrepareFinalizeAndReplaySafety`
  - did not reproduce
- `TestPrepareFrankZohoCampaignFollowUpBlocksDuplicateUnresolvedAction`
  - did not reproduce

## Whether the package failure appears flaky or deterministic

- Current evidence points to flaky or environment-sensitive, not deterministic.
- Both previously reported failure strings map to timestamp-order validators.
- Neither failure reproduced across:
  - 6 isolated runs of the send prepare/finalize/replay test
  - 6 isolated runs of the duplicate follow-up test
  - 1 full `./internal/agent/tools` package run

## Exact assertion locations

- `TestFrankZohoCampaignSendPrepareFinalizeAndReplaySafety`
  - file: `internal/agent/tools/frank_zoho_send_email_test.go`
  - primary assertion that would fail on the observed wrong error:
    - line 529
    - `t.Fatalf("PrepareFrankZohoCampaignSend(first) error = %v", err)`
  - same-family replay assertion surface:
    - line 567
    - `t.Fatalf("RecordFrankZohoCampaignSend() error = %v", err)`
    - line 615
    - `t.Fatalf("PrepareFrankZohoCampaignSend(sent replay) error = %v", err)`

- `TestPrepareFrankZohoCampaignFollowUpBlocksDuplicateUnresolvedAction`
  - file: `internal/agent/tools/frank_zoho_send_email_test.go`
  - exact wrong-error assertion:
    - line 2060
    - `t.Fatalf("PrepareFrankZohoCampaignSend(duplicate) error = %q, want unresolved follow-up block", err)`

## Exact functions and records responsible for prepared_at / sent_at

- Prepared timestamp producer path:
  - `internal/agent/tools/taskstate.go:1765`
    - `buildFrankZohoPreparedCampaignAction(ec, args, time.Now().UTC())`
  - `internal/missioncontrol/campaign_zoho_email_outbound_actions.go:215`
    - `BuildCampaignZohoEmailOutboundPreparedAction`
    - sets `PreparedAt = preparedAt.UTC()`

- Sent timestamp producer path:
  - `internal/agent/tools/taskstate.go:1780`
    - `missioncontrol.BuildCampaignZohoEmailOutboundSentAction(prepared, receipt, time.Now().UTC())`
  - `internal/missioncontrol/campaign_zoho_email_outbound_actions.go:253`
    - `BuildCampaignZohoEmailOutboundSentAction`
    - sets `SentAt = sentAt.UTC()`

- Outbound action validator path:
  - `internal/missioncontrol/campaign_zoho_email_outbound_actions.go:84`
    - `ValidateCampaignZohoEmailOutboundAction`
  - sent-state ordering check:
    - line 150-151
    - `sent_at must be on or after prepared_at`
  - verified-state ordering check:
    - line 172-176
    - `sent_at must be on or after prepared_at`
    - `verified_at must be on or after sent_at`

- Outbound action record validator path:
  - `internal/missioncontrol/store_records.go:450`
    - `ValidateCampaignZohoEmailOutboundActionRecord`
    - reconstructs the runtime action and calls `ValidateCampaignZohoEmailOutboundAction`

## Exact functions and records responsible for created_at / updated_at

- Reply work item open producer path:
  - `internal/agent/tools/taskstate.go:1577`
    - `missioncontrol.LoadMissingCommittedCampaignZohoEmailReplyWorkItems(..., time.Now().UTC())`
  - `internal/missioncontrol/campaign_zoho_email_reply_triage.go:16`
    - `DeriveMissingCampaignZohoEmailReplyWorkItems`
  - `internal/missioncontrol/campaign_zoho_email_reply_work_items.go:104`
    - `BuildCampaignZohoEmailReplyWorkItemOpen`
    - sets:
      - `CreatedAt = now.UTC()`
      - `UpdatedAt = now.UTC()`

- Reply work item claim/update producer path:
  - `internal/agent/tools/taskstate.go:1643`
    - `now := time.Now().UTC()`
  - `internal/agent/tools/taskstate.go:1715`
    - `claimFrankZohoCampaignReplyWorkItem(ec, *ec.Runtime, action, now)`
  - `internal/missioncontrol/campaign_zoho_email_reply_work_items.go:133`
    - `BuildCampaignZohoEmailReplyWorkItemClaimed`
    - preserves existing `CreatedAt`
    - sets `UpdatedAt = updatedAt.UTC()`

- Reply work item validator path:
  - `internal/missioncontrol/campaign_zoho_email_reply_work_items.go:43`
    - `ValidateCampaignZohoEmailReplyWorkItem`
  - ordering check:
    - line 65-66
    - `updated_at must be on or after created_at`

- Reply work item record validator path:
  - `internal/missioncontrol/store_records.go:299`
    - `ValidateCampaignZohoEmailReplyWorkItemRecord`
    - reconstructs the runtime work item and calls `ValidateCampaignZohoEmailReplyWorkItem`

## Diagnosis

- Both reported failures fit the same broader family:
  - a logical Zoho operation samples wall clock more than once
  - a later timestamped transition validates against an earlier field
  - if the later sample is earlier than the earlier sample, validation fails before the intended behavioral assertion is reached

- Outbound action family:
  - `RecordFrankZohoCampaignSend` currently calls:
    - `buildFrankZohoPreparedCampaignAction(..., time.Now().UTC())`
    - then `BuildCampaignZohoEmailOutboundSentAction(..., time.Now().UTC())`
  - if the second sample is earlier than the first, `sent_at < prepared_at` is possible

- Reply work item family:
  - `PrepareFrankZohoCampaignSend` currently uses:
    - one `time.Now().UTC()` in sync/derive of missing open reply work items
    - another `time.Now().UTC()` later for follow-up claim/update
  - if the claim/update timestamp is earlier than the open-item timestamp, `updated_at < created_at` is possible

- That makes the two observed failures part of one timestamp-order flake family, not two unrelated bugs.

## Whether the GC3 test split plausibly changed only scheduling or changed semantics

- It plausibly changed scheduling only.
- GC3-TREAT-001B moved same-package tests into a dedicated file and did not change production code.
- It does not plausibly change Zoho semantics directly.
- It could increase or reshuffle `t.Parallel()` scheduling in `internal/agent/tools`, which can expose a latent timing bug sooner.
- The observed validator strings still point to production timestamp sampling, not to changed test semantics.

## Smallest safe fix candidate

- Do not change behavior broadly.
- Stabilize each logical Zoho operation path by sampling one timestamp once and reusing it for every ordering-related state transition in that operation.

- Smallest candidate slices:
  - in `RecordFrankZohoCampaignSend`
    - sample one `now := time.Now().UTC()`
    - reuse it for:
      - `buildFrankZohoPreparedCampaignAction`
      - `BuildCampaignZohoEmailOutboundSentAction`
  - in `PrepareFrankZohoCampaignSend`
    - sample one stable `now`
    - reuse it for:
      - missing reply-work-item derivation in the sync path when this call depends on it
      - `buildFrankZohoCampaignSendIntent`
      - `claimFrankZohoCampaignReplyWorkItem`
      - any reply-work-item state transitions in that same invocation

- This is the smallest safe fix because it:
  - targets only the identified timestamp-order path
  - does not weaken tests
  - does not change unrelated behavior
  - directly eliminates the identified ordering hazard

## Required regression tests

- Keep existing probes:
  - `go test -count=1 -run TestFrankZohoCampaignSendPrepareFinalizeAndReplaySafety -v ./internal/agent/tools`
  - `go test -count=1 -run TestPrepareFrankZohoCampaignFollowUpBlocksDuplicateUnresolvedAction -v ./internal/agent/tools`
  - `go test -count=1 ./internal/agent/tools`

- Add focused regression coverage if a fix is applied:
  - one narrow test proving `RecordFrankZohoCampaignSend` cannot trip `sent_at < prepared_at` from multiple clock samples within one logical send-finalize operation
  - one narrow test proving the follow-up duplicate-block path cannot trip `updated_at < created_at` when missing reply work items are derived and claimed in the same logical operation

## Bottom line

- Current `HEAD` did not reproduce either failure under the requested probes.
- The package is green for the exact commands requested here.
- The most credible root cause remains a latent multi-`time.Now()` ordering hazard in Zoho operation paths.
- No code fix was applied in this turn because the failure family did not reproduce under the requested gate.
