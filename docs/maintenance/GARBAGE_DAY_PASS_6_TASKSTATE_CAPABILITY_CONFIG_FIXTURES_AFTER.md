# Garbage Day Pass 6 After

## git diff --stat
```text
 .../taskstate_capability_test_helpers_test.go      |  63 +++++++++++
 internal/agent/tools/taskstate_test.go             | 126 ---------------------
 2 files changed, 63 insertions(+), 126 deletions(-)
```

## git diff --numstat
```text
63	0	internal/agent/tools/taskstate_capability_test_helpers_test.go
0	126	internal/agent/tools/taskstate_test.go
```

## files changed
- modified: `internal/agent/tools/taskstate_capability_test_helpers_test.go`
- modified: `internal/agent/tools/taskstate_test.go`
- added (untracked): `docs/maintenance/GARBAGE_DAY_PASS_6_TASKSTATE_CAPABILITY_CONFIG_FIXTURES_BEFORE.md`
- added (untracked): `docs/maintenance/GARBAGE_DAY_PASS_6_TASKSTATE_CAPABILITY_CONFIG_FIXTURES_AFTER.md`

## before/after line counts
- `internal/agent/tools/taskstate_test.go`: `7525 -> 7399`
- `internal/agent/tools/taskstate_capability_test_helpers_test.go`: `187 -> 250`

## exact helpers moved or introduced
- Introduced shared config helper:
  - `writeTaskStateWorkspaceCapabilityConfigFixture`
- Moved these config fixture wrappers out of `internal/agent/tools/taskstate_test.go` into `internal/agent/tools/taskstate_capability_test_helpers_test.go` without renaming:
  - `writeTaskStateContactsCapabilityConfigFixture`
  - `writeTaskStateLocationCapabilityConfigFixture`
  - `writeTaskStateCameraCapabilityConfigFixture`
  - `writeTaskStateMicrophoneCapabilityConfigFixture`
  - `writeTaskStateSMSPhoneCapabilityConfigFixture`
  - `writeTaskStateBluetoothNFCCapabilityConfigFixture`
  - `writeTaskStateBroadAppControlCapabilityConfigFixture`

## exact fixture records preserved
- The shared config helper preserves the exact setup written by every moved wrapper:
  - `home := t.TempDir()`
  - `configDir := filepath.Join(home, ".picobot")`
  - `os.MkdirAll(configDir, 0o755)`
  - `workspace := filepath.Join(home, "workspace-root")`
  - `configPath := filepath.Join(configDir, "config.json")`
  - `configJSON := fmt.Sprintf(\`{"agents":{"defaults":{"workspace":%q}}}\`, workspace)`
  - `os.WriteFile(configPath, []byte(configJSON), 0o644)`
  - `t.Setenv("HOME", home)`
  - returns `workspace`
- The wrapper names and return values remain identical at every test call site.

## exact assertions preserved
- No test scenario names changed.
- No assertion text changed.
- No acceptance/rejection expectations changed.
- No capability onboarding, proposal, config validation, or exposure assertions changed.
- The only modifications in `internal/agent/tools/taskstate_test.go` were deletion of duplicated config-fixture helper bodies.

## repeated fixture blocks intentionally left alone and why
- Left the inline Telegram notifications config setup in `TestTaskStateActivateStepNotificationsCapabilityPathInvokesRealMutation`.
  - Reason: it writes a different config shape (`channels.telegram`) and is not the same workspace-defaults fixture seam.
- Left the inline shared-storage config setup in `TestTaskStateActivateStepSharedStorageCapabilityPathInvokesRealMutation`.
  - Reason: it is similar to the moved workspace helper, but converting inline test body setup in this pass would widen the slice beyond the smallest safe helper extraction.
- Left the earlier separate config helper blocks elsewhere in `taskstate_test.go`.
  - Reason: they are adjacent but outside the capability-config wrapper cluster targeted in this pass and should be evaluated as a separate cleanup slice.

## risks / deferred cleanup
- The inline shared-storage workspace config setup is still duplicated and is a plausible next cleanup candidate.
- Additional config-related helpers elsewhere in `taskstate_test.go` may be reducible, but they were not touched here to avoid widening scope.
- This pass assumes all moved capability-config wrappers should remain identical; that matches the pre-change code exactly.

## validation commands and results
- `gofmt -w internal/agent/tools/taskstate_test.go internal/agent/tools/taskstate_capability_test_helpers_test.go`
  - result: passed
- `git diff --check`
  - result: passed
- `go test -count=1 ./internal/agent/tools -run 'Test.*(Capability|Onboarding|Exposure|Proposal|Config)'`
  - result: `ok  	github.com/local/picobot/internal/agent/tools	0.718s`
- `go test -count=1 ./internal/agent/tools`
  - result: `ok  	github.com/local/picobot/internal/agent/tools	8.540s`
- `go test -count=1 ./...`
  - result: passed
  - representative tail:
    - `ok  	github.com/local/picobot/cmd/picobot	13.803s`
    - `ok  	github.com/local/picobot/internal/agent	0.321s`
    - `ok  	github.com/local/picobot/internal/agent/tools	13.302s`
    - `ok  	github.com/local/picobot/internal/missioncontrol	9.433s`
