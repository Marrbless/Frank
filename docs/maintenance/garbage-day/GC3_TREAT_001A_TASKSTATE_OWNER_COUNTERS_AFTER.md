# GC3-TREAT-001A TaskState Owner Counters After

Date: 2026-04-20

## git diff --stat

Tracked diff:

```text
 internal/agent/tools/taskstate.go | 334 --------------------------------------
 1 file changed, 334 deletions(-)
```

Untracked extracted file diffstat:

```text
 .../agent/tools/taskstate_owner_counters.go        | 341 +++++++++++++++++++++
 1 file changed, 341 insertions(+)
```

## git diff --numstat

Tracked diff:

```text
0	334	internal/agent/tools/taskstate.go
```

Untracked extracted file numstat:

```text
341	0	/dev/null => internal/agent/tools/taskstate_owner_counters.go
```

## Files changed

- `internal/agent/tools/taskstate.go`
- `internal/agent/tools/taskstate_owner_counters.go`
- `docs/maintenance/garbage-day/GC3_TREAT_001A_TASKSTATE_OWNER_COUNTERS_BEFORE.md`
- `docs/maintenance/garbage-day/GC3_TREAT_001A_TASKSTATE_OWNER_COUNTERS_AFTER.md`

## Exact functions moved

- `RecordOwnerFacingMessage`
- `RecordOwnerFacingCheckIn`
- `RecordOwnerFacingDailySummary`
- `RecordOwnerFacingApprovalRequest`
- `RecordOwnerFacingCompletion`
- `RecordOwnerFacingWaitingUser`
- `RecordOwnerFacingBudgetPause`
- `RecordOwnerFacingDenyAck`
- `RecordOwnerFacingPauseAck`
- `RecordOwnerFacingSetStepAck`
- `RecordOwnerFacingRevokeApprovalAck`
- `RecordOwnerFacingResumeAck`

## Exact functions intentionally left in taskstate.go and why

- `ApplyWaitingUserInput`
  - start of the protected approval / waiting-user / runtime-control parity zone
- `ApplyNaturalApprovalDecision`
- `ApplyApprovalDecision`
- `RevokeApproval`
- `PauseRuntime`
- `ResumeRuntimeControl`
- `AbortRuntime`
  - all are part of the protected runtime-control and approval path and were intentionally left untouched
- `storeRuntimeStateLocked`
- `persistPreparedRuntimeStateLocked`
- `persistHydratedRuntimeStateLocked`
- `hydrateRuntimeControlLocked`
- `projectRuntimeStateLocked`
- `applyRuntimeControl`
  - all are part of the protected persistence / hydration / projection core
- capability appliers, treasury/campaign/onboarding activation helpers, and Zoho outbound/reply-work-item helpers
  - all are outside this seam and remained in `taskstate.go`

## Runtime behavior

- Runtime behavior was not changed.
- The move was same-package only and preserved outputs, error handling, locking, runtime persistence calls, and notification behavior exactly.

## Validation commands and results

- `gofmt -w internal/agent/tools/taskstate.go internal/agent/tools/taskstate_owner_counters.go`
  - passed
- `git diff --check`
  - passed
- `go test -count=1 ./internal/agent/tools`
  - passed
- `go test -count=1 ./...`
  - first run hit a non-reproducing failure in `TestFrankZohoCampaignSendPrepareFinalizeAndReplaySafety`
  - immediate rerun passed
- `git status --short --branch`
  - `## frank-v3-foundation`
  - ` M internal/agent/tools/taskstate.go`
  - `?? docs/maintenance/garbage-day/GC3_TREAT_001A_TASKSTATE_OWNER_COUNTERS_BEFORE.md`
  - `?? docs/maintenance/garbage-day/GC3_TREAT_001A_TASKSTATE_OWNER_COUNTERS_AFTER.md`
  - `?? internal/agent/tools/taskstate_owner_counters.go`

## Deferred next candidates from the TaskState assessment

- `GC3-TREAT-001B` inspect-family test split
- `GC3-TREAT-001C` capability exposure applier extraction
- `GC3-TREAT-001D` runtime persistence-core extraction
- `GC3-TREAT-001E` approval / reboot-safe control cleanup
