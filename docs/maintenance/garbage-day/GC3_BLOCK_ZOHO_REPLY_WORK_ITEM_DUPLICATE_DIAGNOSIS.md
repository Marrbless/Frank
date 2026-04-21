# GC3 Block: Zoho Reply Work Item Duplicate Diagnosis

Date: 2026-04-21

- Branch: `frank-v3-foundation`
- HEAD: `2555730f39f1480bbd900ca13d4fc8e21fb26377`

## Reconciliation evidence

- `git status --short --branch`
  - `## frank-v3-foundation`
- `git rev-parse HEAD`
  - `2555730f39f1480bbd900ca13d4fc8e21fb26377`
- `go test -count=1 -run TestPrepareFrankZohoCampaignFollowUpBlocksDuplicateUnresolvedAction -v ./internal/agent/tools`
  - passed
- repeated 5 times with `-count=1`
  - all 5 passed
- `go test -count=1 ./internal/agent/tools`
  - passed

## Whether the failure reproduced

- No. The named test did not reproduce in isolation.
- The package-level `./internal/agent/tools` run also passed during this diagnosis.

## Whether it appears flaky

- Yes, provisionally.
- The reported failure mode was not reproducible in 6 isolated executions of the target test plus 1 full package run.
- The observed error string points to a time-order validation failure that is not the intended assertion for the test, which is consistent with a latent clock/order edge rather than a stable duplicate-action logic failure.

## Exact failing assertion location

- Test: `TestPrepareFrankZohoCampaignFollowUpBlocksDuplicateUnresolvedAction`
- File: `internal/agent/tools/frank_zoho_send_email_test.go`
- Assertion reached by the reported failure:
  - line 2060
  - `t.Fatalf("PrepareFrankZohoCampaignSend(duplicate) error = %q, want unresolved follow-up block", err)`
- The expected error substring there is:
  - `"already has unresolved follow-up action"`

## Exact created_at / updated_at producer path

- Missing committed reply work items are derived during sync:
  - `internal/agent/tools/taskstate.go`
    - `PrepareFrankZohoCampaignSend` triggers sync when `inbound_reply_id` is present.
    - `SyncFrankZohoCampaignInboundReplies` calls `missioncontrol.LoadMissingCommittedCampaignZohoEmailReplyWorkItems(..., time.Now().UTC())`.
  - `internal/missioncontrol/campaign_zoho_email_reply_triage.go`
    - `LoadMissingCommittedCampaignZohoEmailReplyWorkItems`
    - `DeriveMissingCampaignZohoEmailReplyWorkItems`
  - `internal/missioncontrol/campaign_zoho_email_reply_work_items.go`
    - `BuildCampaignZohoEmailReplyWorkItemOpen`
    - producer sets:
      - `CreatedAt = now.UTC()`
      - `UpdatedAt = now.UTC()`

- Claimed follow-up work item transition path:
  - `internal/agent/tools/taskstate.go`
    - `PrepareFrankZohoCampaignSend` samples another `now := time.Now().UTC()` at line 1643
    - then calls `claimFrankZohoCampaignReplyWorkItem(..., now)` at line 1715
  - `internal/missioncontrol/campaign_zoho_email_reply_work_items.go`
    - `BuildCampaignZohoEmailReplyWorkItemClaimed`
    - producer preserves existing `CreatedAt`
    - producer sets:
      - `UpdatedAt = updatedAt.UTC()`

## Exact created_at / updated_at validator path

- Runtime/item validator:
  - `internal/missioncontrol/campaign_zoho_email_reply_work_items.go:43-101`
  - `ValidateCampaignZohoEmailReplyWorkItem`
  - ordering check:
    - line 65-66
    - `if normalized.UpdatedAt.Before(normalized.CreatedAt) { return fmt.Errorf("mission runtime campaign zoho email reply work item updated_at must be on or after created_at") }`

- Store-record validator path:
  - `internal/missioncontrol/store_records.go:299-327`
  - `ValidateCampaignZohoEmailReplyWorkItemRecord`
  - reconstructs a runtime item record and calls `ValidateCampaignZohoEmailReplyWorkItem`

- Load/list validation path:
  - `internal/missioncontrol/store_campaign_zoho_email_reply_work_items.go`
    - `LoadCampaignZohoEmailReplyWorkItemRecord`
    - `loadCampaignZohoEmailReplyWorkItemRecordFile`
    - `ListCommittedCampaignZohoEmailReplyWorkItemRecords`
    - `ListCommittedAllCampaignZohoEmailReplyWorkItemRecords`

## Diagnosis

- The target test’s intended failure path is:
  1. build or load the reply work item for the inbound reply
  2. claim that work item for the new prepared follow-up action
  3. list committed follow-up actions by `inbound_reply_id`
  4. reject because another prepared follow-up action already exists

- The observed wrong error string can only occur earlier, before step 4, when a reply work item fails timestamp validation.

- The most plausible code-level cause is an intra-call timestamp ordering hazard inside `PrepareFrankZohoCampaignSend`:
  - sync derives a missing open reply work item using one `time.Now().UTC()` sample
  - later in the same call, claim uses a second independent `time.Now().UTC()` sample
  - if the second wall-clock sample is earlier than the first, `BuildCampaignZohoEmailReplyWorkItemClaimed` fails because it preserves `CreatedAt` and sets a smaller `UpdatedAt`

- This is consistent with the reported error string and does not require any production logic change from GC3-TREAT-001B.

## Whether GC3-TREAT-001B could plausibly affect ordering/shared state

- Very unlikely as a direct cause.
- GC3-TREAT-001B was a same-package test-file split only; it did not change production code.
- The moved `TaskState OperatorInspect` tests:
  - use their own temp store roots
  - do not modify the Zoho reply reader globals
  - do not participate in the duplicate-follow-up logic

- The only plausible indirect effect is package scheduling pressure:
  - the new file adds more `t.Parallel()` tests to `internal/agent/tools`
  - changed scheduling can expose latent package-level test fragility

- Even with that caveat, the specific observed error is better explained by timestamp sampling inside the Zoho path than by shared fixtures from the GC3 split.

## Smallest safe fix candidate

- Use one stable timestamp for the full `PrepareFrankZohoCampaignSend` follow-up preparation path and thread it through the reply-work-item derivation/claim transitions.
- Concretely:
  - sample `now` once near the start of `PrepareFrankZohoCampaignSend`
  - pass that same `now` into the sync path that derives missing reply work items
  - reuse the same `now` for `buildFrankZohoCampaignSendIntent`, claim, and any follow-up work-item transitions in that invocation

- Why this is the smallest safe fix:
  - it preserves behavior
  - it does not weaken tests
  - it directly removes the only identified `created_at > updated_at` edge within this call chain
  - it does not require V4 or broader cleanup

## Required tests

- Existing targeted regression checks to keep:
  - `go test -count=1 -run TestPrepareFrankZohoCampaignFollowUpBlocksDuplicateUnresolvedAction -v ./internal/agent/tools`
  - `go test -count=1 ./internal/agent/tools`

- Additional focused regression test recommended if a fix is applied:
  - a narrow test proving follow-up duplicate blocking still wins even when reply-work-item derivation and claim happen in the same call path
  - a narrow test proving reply-work-item transitions never fail `updated_at >= created_at` because of multiple `time.Now()` samples inside one operation

## Bottom line

- Current evidence says:
  - failure did not reproduce
  - it appears flaky or environment-sensitive
  - the GC3-TREAT-001B test split is not a plausible primary cause
  - the most credible root cause is independent wall-clock sampling inside the Zoho reply-work-item preparation path
