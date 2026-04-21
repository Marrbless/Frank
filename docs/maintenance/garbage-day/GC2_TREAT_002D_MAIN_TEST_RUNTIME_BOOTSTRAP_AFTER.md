## GC2-TREAT-002D After

### git diff --stat

Tracked diff:

```text
cmd/picobot/main_test.go | 6554 +---------------------------------------------
1 file changed, 8 insertions(+), 6546 deletions(-)
```

Untracked new file evidence:

```text
.../picobot/main_runtime_bootstrap_test.go | 6555 ++++++++++++++++++++
1 file changed, 6555 insertions(+)
```

### git diff --numstat

Tracked diff:

```text
8	6546	cmd/picobot/main_test.go
```

Untracked new file evidence:

```text
6555	0	/dev/null => cmd/picobot/main_runtime_bootstrap_test.go
```

### Files Changed

- `cmd/picobot/main_test.go`
- `cmd/picobot/main_runtime_bootstrap_test.go`
- `docs/maintenance/garbage-day/GC2_TREAT_002D_MAIN_TEST_RUNTIME_BOOTSTRAP_BEFORE.md`
- `docs/maintenance/garbage-day/GC2_TREAT_002D_MAIN_TEST_RUNTIME_BOOTSTRAP_AFTER.md`

### Exact Tests Moved

Moved from `cmd/picobot/main_test.go` to `cmd/picobot/main_runtime_bootstrap_test.go`:

- all `TestMissionSetStepCommand...`
- all `TestConfigureMissionBootstrap...`
- all `TestWriteMissionStatusSnapshot...`
- `TestWriteProjectedMissionStatusSnapshotIncludesCommittedRuntimeSummaryTruncation`
- `TestStartupAndRuntimeChangeDurableProjectionUseSameSharedBuilder`
- `TestMissionStatusSnapshotWritePersistsAuditHistory`
- all `TestMissionStatusBootstrap...`
- all `TestMissionStatusRuntimeChangeHook...`
- all `TestApplyMissionStepControlFile...`
- all `TestRestoreMissionStepControlFileOnStartup...`
- all `TestWatchMissionStepControlFile...`
- all `TestMissionOperatorSetStepCommand...`
- all `TestResolveMissionStoreRoot...`
- all `TestLoadPersistedMissionRuntime...`
- all `TestMissionStatusRuntimePersistence...`
- `TestConfigureMissionBootstrapJobAcceptsV2LongRunningCodeMissionFile`

### Exact Helpers Moved

Moved with the heavyweight family into `cmd/picobot/main_runtime_bootstrap_test.go`:

- `newMissionBootstrapTestCommand`
- `newMissionBootstrapTestLoop`
- `writeCommittedMissionBootstrapJobRuntimeRecord`
- `writeCommittedMissionBootstrapRuntimeControlRecord`
- `writeCommittedMissionBootstrapActiveJobRecord`
- `seedIncoherentMissionStore`
- `assertMissionStatusSnapshotMatchesCommittedDurableState`
- `cloneMissionBootstrapStep`
- `configureMissionBootstrapJobForStartupTest`
- `missionStatusFixedResponseProvider`
- `(*missionStatusFixedResponseProvider).Chat`
- `(*missionStatusFixedResponseProvider).GetDefaultModel`
- `writeMissionBootstrapJobFile`
- `runtimeControlForBootstrapStep`
- `expectedAuthorizationApprovalContent`
- `mustReadFile`
- `writeMissionStepControlFile`
- `writeMissionStatusSnapshotFile`
- `testMissionBootstrapJob`
- `readMissionStatusSnapshotFile`
- `readMissionStepControlFile`
- `assertMissionStepControlFileMissing`
- `assertNoAtomicTempFiles`
- `captureStandardLogger`

### Exact Tests Intentionally Left In `main_test.go` And Why

Left in `cmd/picobot/main_test.go` because they are not part of the heavyweight bootstrap/runtime/watcher/operator-control family:

- `TestPromptSecretFallsBackToReaderWhenNotTerminal`
- `TestPromptSecretUsesHiddenInputWhenTerminalAvailable`
- `TestAgentCLI_ModelFlag`
- `TestMissionPackageLogsCommandReturnsStableSummary`
- `TestMissionPackageLogsCommandPrunesExpiredPackagesAfterSuccessfulPackaging`
- `TestMissionPruneStoreCommandReturnsStableSummary`
- `TestConfigureGatewayMissionStoreLoggingPrunesExpiredPackagesAfterStartupPackaging`
- `TestMissionPruneStoreCommandReturnsStableNoOpSummary`
- `TestConfigureGatewayMissionStoreLoggingRoutesStdlibLoggerIntoActiveSegment`
- `TestConfigureGatewayMissionStoreLoggingWithoutStoreRootPreservesExistingLoggerBehavior`
- `TestRemoveMissionStatusSnapshotRemovesFile`

### Runtime Behavior

- Production behavior was not changed.
- Test meaning, fixture semantics, and assertions were preserved.

### Validation Commands And Results

- `gofmt -w cmd/picobot/main_test.go cmd/picobot/main_runtime_bootstrap_test.go`
  - passed
- `git diff --check`
  - passed
- `go test -count=1 ./cmd/picobot`
  - passed
- `go test -count=1 ./...`
  - passed

### main_test De-omnibus Status

- Yes: this completes the planned `cmd/picobot/main_test.go` de-omnibus campaign.
- `cmd/picobot/main_test.go` is now the small residual test surface for prompt/config and package/store logging coverage.
- The remaining large test mass now lives in a dedicated file, `cmd/picobot/main_runtime_bootstrap_test.go`, which is a future structural target if deeper runtime-family decomposition is desired.
