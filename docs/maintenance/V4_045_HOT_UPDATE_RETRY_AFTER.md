## V4-045 Hot-Update Retry After

### git diff --stat

```text
 internal/agent/loop_processdirect_test.go          | 145 +++++++++++++++
 .../missioncontrol/hot_update_gate_registry.go     |   3 +-
 .../hot_update_gate_registry_test.go               | 194 +++++++++++++++++++++
 3 files changed, 341 insertions(+), 1 deletion(-)
```

### git diff --numstat

```text
145	0	internal/agent/loop_processdirect_test.go
2	1	internal/missioncontrol/hot_update_gate_registry.go
194	0	internal/missioncontrol/hot_update_gate_registry_test.go
```

### files changed

- `internal/missioncontrol/hot_update_gate_registry.go`
- `internal/missioncontrol/hot_update_gate_registry_test.go`
- `internal/agent/loop_processdirect_test.go`
- `docs/maintenance/V4_045_HOT_UPDATE_RETRY_BEFORE.md`
- `docs/maintenance/V4_045_HOT_UPDATE_RETRY_AFTER.md`

### exact helpers/state transitions added

- widened `ExecuteHotUpdateGateReloadApply(...)` to permit retry from `reload_apply_recovery_needed`
- preserved existing `reload_apply_failed` closed behavior
- preserved existing `reload_apply_in_progress` recovery-required behavior
- preserved bounded convergence reuse through `hotUpdateGateRestartStyleConvergence(...)`
- retry start clears stale `FailureReason` and rewrites the same gate to `reload_apply_in_progress`
- retry success rewrites the same gate to `reload_apply_succeeded`
- retry failure rewrites the same gate to `reload_apply_failed` with fresh failure detail

### exact tests added

- `internal/missioncontrol/hot_update_gate_registry_test.go`
  - `TestExecuteHotUpdateGateReloadApplyRetryFromRecoveryNeededSucceedsWithoutSecondPointerMutation`
  - `TestExecuteHotUpdateGateReloadApplyRetryFromRecoveryNeededRecordsFailureAndClearsFailureReasonOnStart`
  - `storeHotUpdateRecoveryNeededFixture(...)`
- `internal/agent/loop_processdirect_test.go`
  - `TestProcessDirectHotUpdateGateReloadCommandRetriesFromRecoveryNeeded`

### validation commands and results

- `gofmt -w internal/missioncontrol/hot_update_gate_registry.go internal/missioncontrol/hot_update_gate_registry_test.go internal/agent/loop_processdirect_test.go`
  - pass
- `git diff --check`
  - pass
- `go test -count=1 ./internal/missioncontrol -run 'TestExecuteHotUpdateGateReloadApply|TestReconcileHotUpdateGateRecoveryNeeded'`
  - pass
- `go test -count=1 ./internal/agent -run 'TestProcessDirectHotUpdateGateReloadCommand'`
  - pass
- `go test -count=1 ./internal/missioncontrol`
  - pass
- `go test -count=1 ./internal/agent`
  - pass
- `go test -count=1 ./internal/agent/tools`
  - pass
- `go test -count=1 ./...`
  - pass
- `git status --short --branch`
  - pass
  - `## frank-v4-045-hot-update-retry`
  - ` M internal/agent/loop_processdirect_test.go`
  - ` M internal/missioncontrol/hot_update_gate_registry.go`
  - ` M internal/missioncontrol/hot_update_gate_registry_test.go`
  - `?? docs/maintenance/V4_045_HOT_UPDATE_RETRY_AFTER.md`
  - `?? docs/maintenance/V4_045_HOT_UPDATE_RETRY_BEFORE.md`

### scope boundary statements

- no second pointer switch or `reload_generation` increment was implemented
- no `last_known_good_pointer.json` mutation was implemented
- no `HotUpdateOutcomeRecord` or `PromotionRecord` was created
- no new hot-update gate/apply record was created

### deferred next V4 candidates

- explicit terminal-failure resolution action from `reload_apply_recovery_needed`
- later terminal-result slice for `HotUpdateOutcomeRecord` creation and any promotion handoff
- any broader recovery policy only if a later checkpoint explicitly selects it
