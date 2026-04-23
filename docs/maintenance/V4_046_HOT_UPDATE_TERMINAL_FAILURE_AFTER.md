## V4-046 Hot-Update Terminal Failure After

### git diff --stat

```text
 internal/agent/loop.go                             |  16 ++
 internal/agent/loop_processdirect_test.go          | 214 +++++++++++++++++++++
 internal/agent/tools/taskstate.go                  |  89 +++++++++
 .../missioncontrol/hot_update_gate_registry.go     |  76 ++++++++
 .../hot_update_gate_registry_test.go               | 209 ++++++++++++++++++++
 5 files changed, 604 insertions(+)
```

### git diff --numstat

```text
16	0	internal/agent/loop.go
214	0	internal/agent/loop_processdirect_test.go
89	0	internal/agent/tools/taskstate.go
76	0	internal/missioncontrol/hot_update_gate_registry.go
209	0	internal/missioncontrol/hot_update_gate_registry_test.go
```

### Files Changed

- `internal/missioncontrol/hot_update_gate_registry.go`
- `internal/missioncontrol/hot_update_gate_registry_test.go`
- `internal/agent/tools/taskstate.go`
- `internal/agent/loop.go`
- `internal/agent/loop_processdirect_test.go`
- `docs/maintenance/V4_046_HOT_UPDATE_TERMINAL_FAILURE_BEFORE.md`
- `docs/maintenance/V4_046_HOT_UPDATE_TERMINAL_FAILURE_AFTER.md`

### Implemented Terminal-Failure Behavior

- Added `ResolveHotUpdateGateTerminalFailure(root, hotUpdateID, reason, updatedBy, updatedAt)`.
- Added deterministic operator failure detail formatting:
  - `operator_terminal_failure: <reason>`
- Added one explicit operator-driven transition:
  - `reload_apply_recovery_needed -> reload_apply_failed`
- Required non-empty reason text after trimming whitespace.
- Allowed exact replay of the same terminal-failure decision and reason to return idempotently when the gate is already `reload_apply_failed` with the same deterministic `failure_reason`.
- Failed closed when a different reason is submitted after terminal failure.
- Added the matching `TaskState` wrapper:
  - `ResolveHotUpdateGateTerminalFailure(jobID, hotUpdateID, reason)`
- Added the matching direct operator command:
  - `HOT_UPDATE_GATE_FAIL <job_id> <hot_update_id> <reason...>`
- Preserved `HOT_UPDATE_GATE_RELOAD` retry behavior from V4-045 unchanged.
- Kept the current committed hot-update gate record as the sole workflow authority.

### Invariants Preserved

- No active runtime-pack pointer mutation was implemented.
- No `reload_generation` increment was implemented.
- No `last_known_good_pointer.json` mutation was implemented.
- No `HotUpdateOutcomeRecord` creation was implemented.
- No `PromotionRecord` creation was implemented.
- No new hot-update gate or apply record was implemented.
- No automatic retry was implemented.
- No automatic success inference was implemented.
- No automatic failure inference outside `HOT_UPDATE_GATE_FAIL` was implemented.

### Tests Added

- `TestResolveHotUpdateGateTerminalFailureFromRecoveryNeededPreservesCommittedState`
- `TestResolveHotUpdateGateTerminalFailureRequiresReasonAndReplayIsIdempotent`
- `TestResolveHotUpdateGateTerminalFailureRejectsNonRecoveryNeededStates`
- `TestProcessDirectHotUpdateGateFailCommandResolvesRecoveryNeededTerminalFailure`
- `TestProcessDirectHotUpdateGateFailCommandRequiresReasonAndRejectsInvalidStartingPhase`

### Validation Commands and Results

- `/usr/local/go/bin/gofmt -w internal/agent/loop.go internal/agent/loop_processdirect_test.go internal/agent/tools/taskstate.go internal/missioncontrol/hot_update_gate_registry.go internal/missioncontrol/hot_update_gate_registry_test.go`
  - pass
- `git diff --check`
  - pass
- `/usr/local/go/bin/go test -count=1 ./internal/missioncontrol`
  - pass
- `/usr/local/go/bin/go test -count=1 ./internal/agent/tools`
  - pass
- `/usr/local/go/bin/go test -count=1 ./internal/agent`
  - pass
- `/usr/local/go/bin/go test -count=1 ./cmd/picobot`
  - pass
- `/usr/local/go/bin/go test -count=1 ./...`
  - pass

### Notes

- The first sandboxed baseline `go test -count=1 ./...` failed because the sandbox made `/home/omar/.cache/go-build` read-only and blocked loopback socket binding for `httptest`.
- The required Go validation commands were rerun outside the sandbox with the same `/usr/local/go/bin/go` toolchain and passed.
