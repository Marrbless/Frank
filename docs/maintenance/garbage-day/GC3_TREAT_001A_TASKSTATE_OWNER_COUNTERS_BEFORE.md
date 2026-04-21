# GC3-TREAT-001A TaskState Owner Counters Before

Date: 2026-04-20

- Branch: `frank-v3-foundation`
- HEAD: `8476e3f9a7a87f460c32742694953b60f231fc87`
- Tags at HEAD: none
- Ahead/behind `upstream/main`: `391 ahead / 0 behind`
- `git status --short --branch` at start:
  - `## frank-v3-foundation`
- Baseline `go test -count=1 ./...` result:
  - passed

## Exact functions selected for extraction

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

## Exact non-goals

- do not change runtime behavior
- do not change persistence or hydration logic
- do not touch approval / waiting-user / runtime-control parity code
- do not touch capability exposure applier logic
- do not touch treasury / campaign / onboarding activation logic
- do not touch Zoho outbound or reply-work-item lifecycle code
- do not perform broader TaskState cleanup beyond this wrapper family

## Expected destination file

- `internal/agent/tools/taskstate_owner_counters.go`
