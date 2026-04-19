# Garbage Day Pass 7 After

## git diff --stat
```text
 .../agent/tools/taskstate_capability_test_helpers_test.go   |  6 ++++++
 internal/agent/tools/taskstate_test.go                      | 13 +------------
 2 files changed, 7 insertions(+), 12 deletions(-)
```

## git diff --numstat
```text
6	0	internal/agent/tools/taskstate_capability_test_helpers_test.go
1	12	internal/agent/tools/taskstate_test.go
```

## files changed
- modified: `internal/agent/tools/taskstate_capability_test_helpers_test.go`
- modified: `internal/agent/tools/taskstate_test.go`
- added (untracked): `docs/maintenance/GARBAGE_DAY_PASS_7_TASKSTATE_SHARED_STORAGE_CONFIG_FIXTURES_BEFORE.md`
- added (untracked): `docs/maintenance/GARBAGE_DAY_PASS_7_TASKSTATE_SHARED_STORAGE_CONFIG_FIXTURES_AFTER.md`

## before/after line counts
- `internal/agent/tools/taskstate_test.go`: `7399 -> 7388`
- `internal/agent/tools/taskstate_capability_test_helpers_test.go`: `250 -> 256`

## exact helpers moved or introduced
- Introduced wrapper helper:
  - `writeTaskStateSharedStorageCapabilityConfigFixture`
- Reused existing shared helper:
  - `writeTaskStateWorkspaceCapabilityConfigFixture`
- Replaced the inline workspace config block in:
  - `TestTaskStateActivateStepSharedStorageCapabilityPathInvokesRealMutation`
  - with a call to `writeTaskStateSharedStorageCapabilityConfigFixture(t)`

## exact config fixture records preserved
- The shared-storage wrapper preserves the exact prior config fixture behavior by delegating to `writeTaskStateWorkspaceCapabilityConfigFixture`, which still performs:
  - `home := t.TempDir()`
  - `configDir := filepath.Join(home, ".picobot")`
  - `os.MkdirAll(configDir, 0o755)`
  - `workspace := filepath.Join(home, "workspace-root")`
  - `configPath := filepath.Join(configDir, "config.json")`
  - `configJSON := fmt.Sprintf(\`{"agents":{"defaults":{"workspace":%q}}}\`, workspace)`
  - `os.WriteFile(configPath, []byte(configJSON), 0o644)`
  - `t.Setenv("HOME", home)`
  - return `workspace`
- That preserves:
  - exact config file path
  - exact JSON shape
  - exact HOME override behavior
  - exact workspace root behavior
  - exact fatal-on-error behavior

## exact assertions preserved
- No test scenario names changed.
- No assertion text changed.
- No acceptance/rejection expectations changed.
- No shared-storage capability semantics changed.
- No capability onboarding or exposure assertions changed.
- In `TestTaskStateActivateStepSharedStorageCapabilityPathInvokesRealMutation`, the post-activation assertions remain identical:
  - exposed shared-storage record exists
  - `record.Validator == missioncontrol.SharedStorageWorkspaceCapabilityValidator`
  - `os.Stat(filepath.Join(workspace, "SOUL.md"))` succeeds

## any repeated shared-storage setup intentionally left alone and why
- Left the repeated `missioncontrol.StoreWorkspaceSharedStorageCapabilityExposure(root, workspace)` setup blocks in other tests.
  - Reason: that is a shared-storage exposure-fixture seam, not a workspace config setup seam, and consolidating it would widen this pass beyond the requested smallest safe slice.
- Left the Telegram notifications inline config setup alone.
  - Reason: it writes a different config shape and is not shared-storage workspace config setup.

## risks / deferred cleanup
- The repeated shared-storage exposure-store setup remains a plausible future cleanup slice.
- There are still broader capability/readiness fixtures in `taskstate_test.go`, but this pass intentionally stopped after removing the last inline shared-storage workspace config block.
- The new wrapper adds only naming clarity; behavior still depends on the generic workspace config helper remaining byte-for-byte stable.

## validation commands and results
- `gofmt -w internal/agent/tools/taskstate_test.go internal/agent/tools/taskstate_capability_test_helpers_test.go`
  - result: passed
- `git diff --check`
  - result: passed
- `go test -count=1 ./internal/agent/tools -run 'Test.*(SharedStorage|Capability|Onboarding|Exposure|Proposal|Config)'`
  - result: `ok  	github.com/local/picobot/internal/agent/tools	0.694s`
- `go test -count=1 ./internal/agent/tools`
  - result: `ok  	github.com/local/picobot/internal/agent/tools	8.403s`
- `go test -count=1 ./...`
  - result: passed
  - representative tail:
    - `ok  	github.com/local/picobot/cmd/picobot	13.890s`
    - `ok  	github.com/local/picobot/internal/agent	0.302s`
    - `ok  	github.com/local/picobot/internal/agent/tools	13.368s`
    - `ok  	github.com/local/picobot/internal/missioncontrol	9.386s`
