## V4-039 Hot-Update Pointer Switch After

### git diff --stat

```text
 internal/agent/loop.go                             |  15 ++
 internal/agent/loop_processdirect_test.go          | 122 ++++++++++++
 internal/agent/tools/taskstate.go                  |  89 +++++++++
 .../missioncontrol/hot_update_gate_registry.go     | 133 +++++++++++++
 .../hot_update_gate_registry_test.go               | 218 +++++++++++++++++++++
 5 files changed, 577 insertions(+)
```

### git diff --numstat

```text
15	0	internal/agent/loop.go
122	0	internal/agent/loop_processdirect_test.go
89	0	internal/agent/tools/taskstate.go
133	0	internal/missioncontrol/hot_update_gate_registry.go
218	0	internal/missioncontrol/hot_update_gate_registry_test.go
```

### Files changed

- `internal/missioncontrol/hot_update_gate_registry.go`
- `internal/missioncontrol/hot_update_gate_registry_test.go`
- `internal/agent/tools/taskstate.go`
- `internal/agent/loop.go`
- `internal/agent/loop_processdirect_test.go`
- `docs/maintenance/V4_039_HOT_UPDATE_POINTER_SWITCH_BEFORE.md`
- `docs/maintenance/V4_039_HOT_UPDATE_POINTER_SWITCH_AFTER.md`

### Exact helpers/state transitions added

- added durable hot-update execution helper:
  - `ExecuteHotUpdateGatePointerSwitch(root, hotUpdateID, updatedBy, updatedAt)`
- added durable execution-linkage validation helper:
  - `validateHotUpdateGateExecutionLinkage(root, record)`
- added deterministic pointer attribution helper:
  - `hotUpdateGatePointerUpdateRecordRef(hotUpdateID)`
- added taskstate wrapper:
  - `ExecuteHotUpdateGatePointerSwitch(jobID, hotUpdateID)`
- added direct operator command:
  - `HOT_UPDATE_GATE_EXECUTE <job_id> <hot_update_id>`
- added bounded execution transition:
  - `staged -> reloading`
- added exact replay handling:
  - `reloading` with matching pointer and `update_record_ref=hot_update:<hot_update_id>` returns idempotently without a second increment

### Exact tests added

- `internal/missioncontrol/hot_update_gate_registry_test.go`
  - `TestExecuteHotUpdateGatePointerSwitchSwitchesActivePointerAndIsReplaySafe`
  - `TestExecuteHotUpdateGatePointerSwitchRejectsInvalidStateAndBrokenLinkageWithoutPointerMutation`
- `internal/agent/loop_processdirect_test.go`
  - `TestProcessDirectHotUpdateGateExecuteCommandSwitchesPointerAndIsReplaySafe`

### Validation commands and results

- `gofmt -w internal/missioncontrol/hot_update_gate_registry.go internal/missioncontrol/hot_update_gate_registry_test.go internal/agent/tools/taskstate.go internal/agent/loop.go internal/agent/loop_processdirect_test.go`
  - passed
- `git diff --check`
  - passed
- `go test -count=1 ./internal/missioncontrol`
  - passed
- `go test -count=1 ./internal/agent`
  - passed
- `go test -count=1 ./internal/agent/tools`
  - passed
- `go test -count=1 ./...`
  - passed

### Scope boundary statements

Reload/apply convergence was not implemented in this slice.

No `HotUpdateOutcomeRecord` or `PromotionRecord` was created in this slice.

`last_known_good_pointer.json` was left unchanged in this slice.

### Deferred next V4 candidates

- add the first bounded hot-update reload/apply convergence slice after `reloading`
- decide whether hot-update recovery normalization is needed if a future slice introduces in-progress convergence ambiguity
- add outcome creation only when a later slice can truthfully record a terminal or decision-complete hot-update result
