# GC3-FIX-001 Zoho Timestamp Stabilization After

Date: 2026-04-21

## git diff --stat

```text
 internal/agent/tools/frank_zoho_send_email_test.go | 266 +++++++++++++++++++++
 internal/agent/tools/taskstate.go                  |  62 +++--
 2 files changed, 311 insertions(+), 17 deletions(-)
```

## git diff --numstat

```text
266	0	internal/agent/tools/frank_zoho_send_email_test.go
45	17	internal/agent/tools/taskstate.go
```

## Files changed

- `internal/agent/tools/taskstate.go`
- `internal/agent/tools/frank_zoho_send_email_test.go`
- `docs/maintenance/garbage-day/GC3_FIX_001_ZOHO_TIMESTAMP_STABILIZATION_BEFORE.md`
- `docs/maintenance/garbage-day/GC3_FIX_001_ZOHO_TIMESTAMP_STABILIZATION_AFTER.md`

## Exact logical paths stabilized

- Zoho inbound reply sync and follow-up preparation path
  - `SyncFrankZohoCampaignInboundReplies`
  - `PrepareFrankZohoCampaignSend`
  - `claimFrankZohoCampaignReplyWorkItem`
  - `transitionFrankZohoCampaignReplyWorkItemResponded`
  - `transitionFrankZohoCampaignReplyWorkItemOnFailure`

- Zoho outbound prepare/send/failure/finalize path
  - `PrepareFrankZohoCampaignSend` sent replay finalize branch
  - `RecordFrankZohoCampaignSend`
  - `RecordFrankZohoCampaignSendFailure`

- Stabilization mechanism
  - taskstate-local clock hook: `taskStateNowUTC`
  - single operation timestamp sampled once per logical path
  - producer-side timestamp clamping to predecessor timestamp floors via `taskStateTransitionTimestamp`

## Exact tests added/changed

- Added:
  - `TestFrankZohoCampaignSendStabilizesTimestampOrderingUnderClockRegression`
  - `TestPrepareFrankZohoCampaignFollowUpDuplicateBlockStabilizesTimestampOrderingUnderClockRegression`
  - `useTaskStateNowSequence` helper

- Existing tests retained without weakening:
  - `TestFrankZohoCampaignSendPrepareFinalizeAndReplaySafety`
  - `TestPrepareFrankZohoCampaignFollowUpBlocksDuplicateUnresolvedAction`

## Validators changed

- No missioncontrol validators were changed.
- Stabilization was done in producer paths only.

## Validation commands and results

- `gofmt -w internal/agent/tools/taskstate.go internal/agent/tools/frank_zoho_send_email_test.go`
  - passed
- `git diff --check`
  - passed with no output
- `go test -count=1 ./internal/agent/tools`
  - passed
  - note: run outside sandbox because `httptest` socket creation is restricted inside the sandbox here
- `go test -count=1 -run TestFrankZohoCampaignSendPrepareFinalizeAndReplaySafety -v ./internal/agent/tools`
  - passed 5/5 repeated runs
- `go test -count=1 -run TestPrepareFrankZohoCampaignFollowUpBlocksDuplicateUnresolvedAction -v ./internal/agent/tools`
  - passed 5/5 repeated runs
- `go test -count=1 ./...`
  - passed 3/3 repeated runs
- `git status --short --branch --untracked-files=all`
  - passed
  - output:
    - `## frank-v3-foundation`
    - ` M internal/agent/tools/frank_zoho_send_email_test.go`
    - ` M internal/agent/tools/taskstate.go`
    - `?? docs/maintenance/garbage-day/GC3_FIX_001_ZOHO_TIMESTAMP_STABILIZATION_AFTER.md`
    - `?? docs/maintenance/garbage-day/GC3_FIX_001_ZOHO_TIMESTAMP_STABILIZATION_BEFORE.md`

## Outcome

- Producer-side Zoho timestamp sampling is now stabilized per logical operation path.
- Ordering-sensitive fields continue to satisfy existing validators without relaxing any validation rules.
