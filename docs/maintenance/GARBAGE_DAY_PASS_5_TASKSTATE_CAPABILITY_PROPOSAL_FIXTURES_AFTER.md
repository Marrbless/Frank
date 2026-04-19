# Garbage Day Pass 5 After

## git diff --stat
```text
 internal/agent/tools/taskstate_test.go | 216 ---------------------------------
 1 file changed, 216 deletions(-)
```

Note: `git diff --stat` excludes the new untracked helper file until it is staged. The new helper file added in this pass is `internal/agent/tools/taskstate_capability_test_helpers_test.go`.

## git diff --numstat
```text
0	216	internal/agent/tools/taskstate_test.go
```

## files changed
- modified: `internal/agent/tools/taskstate_test.go`
- added (untracked): `internal/agent/tools/taskstate_capability_test_helpers_test.go`
- added (untracked): `docs/maintenance/GARBAGE_DAY_PASS_5_TASKSTATE_CAPABILITY_PROPOSAL_FIXTURES_AFTER.md`
- retained from baseline: `docs/maintenance/GARBAGE_DAY_PASS_5_TASKSTATE_CAPABILITY_PROPOSAL_FIXTURES_BEFORE.md`

## before/after line counts
- `internal/agent/tools/taskstate_test.go`: `7741 -> 7525`
- `internal/agent/tools/taskstate_capability_test_helpers_test.go`: `absent -> 187`

## exact helpers moved or introduced
- Introduced shared helper:
  - `writeTaskStateCapabilityProposalFixture`
- Introduced shared fixture spec:
  - `taskStateCapabilityProposalFixtureSpec`
- Moved these repeated proposal fixture writers out of `taskstate_test.go` and into `taskstate_capability_test_helpers_test.go` without renaming:
  - `writeTaskStateNotificationsCapabilityProposalFixture`
  - `writeTaskStateSharedStorageCapabilityProposalFixture`
  - `writeTaskStateContactsCapabilityProposalFixture`
  - `writeTaskStateLocationCapabilityProposalFixture`
  - `writeTaskStateCameraCapabilityProposalFixture`
  - `writeTaskStateMicrophoneCapabilityProposalFixture`
  - `writeTaskStateSMSPhoneCapabilityProposalFixture`
  - `writeTaskStateBluetoothNFCCapabilityProposalFixture`
  - `writeTaskStateBroadAppControlCapabilityProposalFixture`

## exact fixture records preserved
- Each wrapper still writes a `missioncontrol.CapabilityOnboardingProposalRecord` with the same per-scenario values for:
  - `ProposalID`
  - `CapabilityName`
  - `WhyNeeded`
  - `MissionFamilies`
  - `Risks`
  - `Validators`
  - `KillSwitch`
  - `DataAccessed`
  - `ApprovalRequired` (`true` in every moved fixture)
  - `CreatedAt`
  - `State` (still passed through from each test call site)
- The helper preserves the same storage path behavior:
  - `t.TempDir()`
  - `missioncontrol.StoreCapabilityOnboardingProposalRecord(root, record)`
  - fatal-on-error behavior via `t.Fatalf("StoreCapabilityOnboardingProposalRecord() error = %v", err)`

## exact assertions preserved
- No test scenario names changed.
- No assertion text changed.
- No acceptance/rejection expectations changed.
- No committed record shape assertions changed.
- No capability exposure, onboarding, or proposal validation assertions changed.
- The only edited test file content was deletion of duplicate proposal-fixture bodies from `taskstate_test.go`; all assertions remain in place at their original call sites.

## repeated fixture blocks intentionally left alone
- Left all `writeTaskState*CapabilityConfigFixture` helpers in `taskstate_test.go`.
- Reason:
  - they are a separate config-fixture seam, not proposal-fixture writing
  - consolidating them in the same pass would widen scope beyond the requested smallest safe test-only change
  - they are more coupled to per-capability workspace/config setup than the proposal record writers

## behavior preservation notes
- This pass touched test-only Go files only.
- No production Go files were modified.
- Wrapper helper names were preserved to avoid hiding scenario intent in the tests.
- The shared helper only centralizes repeated `CapabilityOnboardingProposalRecord` construction and storage; it does not change the values written by any scenario.

## risks / deferred cleanup
- The capability config-fixture helpers remain repetitive and are a plausible future cleanup slice.
- The shared helper assumes all moved proposal fixtures continue to require `ApprovalRequired: true`; that matches every moved fixture today and is preserved explicitly.
- Because the new helper file is untracked, `git diff --stat` and `git diff --numstat` underreport the full unstaged change set until the file is staged. The file content was still validated by `go test`.

## validation commands and results
- `gofmt -w internal/agent/tools/taskstate_test.go internal/agent/tools/taskstate_capability_test_helpers_test.go`
  - result: passed
- `git diff --check`
  - result: passed
- `go test -count=1 ./internal/agent/tools -run 'Test.*(Capability|Onboarding|Exposure|Proposal)'`
  - result: `ok  	github.com/local/picobot/internal/agent/tools	0.764s`
- `go test -count=1 ./internal/agent/tools`
  - result: `ok  	github.com/local/picobot/internal/agent/tools	8.859s`
- `go test -count=1 ./...`
  - result: passed
  - representative tail:
    - `ok  	github.com/local/picobot/cmd/picobot	14.405s`
    - `ok  	github.com/local/picobot/internal/agent	0.320s`
    - `ok  	github.com/local/picobot/internal/agent/tools	14.004s`
    - `ok  	github.com/local/picobot/internal/missioncontrol	9.965s`
