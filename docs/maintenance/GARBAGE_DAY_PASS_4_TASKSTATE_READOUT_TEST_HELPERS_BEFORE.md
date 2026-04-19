# Garbage Day Pass 4 TaskState Readout Test Helpers Before

## Repo
- repo root: `/mnt/d/pbot/picobot`
- branch: `frank-v3-foundation`
- HEAD: `f9232bd6ed4badf11053285d8717a8d9e4b24f36`

## Git Status
```text
## frank-v3-foundation
```

## Pass 3 Commit State
- Pass 3 is committed or otherwise no longer present as an uncommitted worktree diff.
- Evidence: `internal/agent/tools/taskstate_readout.go` already exists in the tree and `git status --short --branch` is clean before Pass 4.

## Line Counts
- `internal/agent/tools/taskstate_test.go`: `7763`
- `internal/agent/tools/taskstate_status_test.go`: `1553`
- `internal/agent/tools/taskstate_readout_test_helpers_test.go`: not present before extraction

## Exact Duplicate Helper Candidates
- Exact duplicate:
  - `writeMalformedTreasuryRecordForTaskStateTest`
  - `writeMalformedTreasuryRecordForTaskStateStatusTest`
- Not duplicate, so not planned for consolidation in this pass:
  - `writeDeferredSchedulerTriggerForTaskStateStatusTest`
  - `mustTaskStateReadoutJSON`
  - `mustTaskStateJSONObject`
  - `mustTaskStateJSONArray`
  - `assertTaskStateReadoutAdapterBoundary`
  - `assertTaskStateJSONObjectKeys`
  - `assertTaskStateResolved*JSONEnvelope`
  - These are already shared package-wide rather than duplicated in both target files.

## Exact Helpers Planned For Extraction
- Move the duplicated malformed treasury writer into:
  - `internal/agent/tools/taskstate_readout_test_helpers_test.go`
- Planned shared helper:
  - `writeMalformedTreasuryRecordForTaskStateReadoutTest`

## Baseline Validation
- command: `go test -count=1 ./internal/agent/tools -run 'TestTaskStateOperator(Status|Inspect)'`
- result: `ok  	github.com/local/picobot/internal/agent/tools	0.350s`
