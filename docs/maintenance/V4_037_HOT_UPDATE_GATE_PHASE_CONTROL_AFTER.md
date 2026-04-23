## V4-037 Hot-Update Gate Phase Control After

### git diff --stat

```text
 internal/agent/loop.go                             |  16 ++
 internal/agent/loop_processdirect_test.go          |  96 +++++++++
 internal/agent/tools/taskstate.go                  |  89 +++++++++
 .../missioncontrol/hot_update_gate_registry.go     |  96 +++++++++
 .../hot_update_gate_registry_test.go               | 215 +++++++++++++++++++++
 5 files changed, 512 insertions(+)
```

### git diff --numstat

```text
16	0	internal/agent/loop.go
96	0	internal/agent/loop_processdirect_test.go
89	0	internal/agent/tools/taskstate.go
96	0	internal/missioncontrol/hot_update_gate_registry.go
215	0	internal/missioncontrol/hot_update_gate_registry_test.go
```

### Files changed

- `internal/missioncontrol/hot_update_gate_registry.go`
- `internal/missioncontrol/hot_update_gate_registry_test.go`
- `internal/agent/tools/taskstate.go`
- `internal/agent/loop.go`
- `internal/agent/loop_processdirect_test.go`
- `docs/maintenance/V4_037_HOT_UPDATE_GATE_PHASE_CONTROL_BEFORE.md`
- `docs/maintenance/V4_037_HOT_UPDATE_GATE_PHASE_CONTROL_AFTER.md`

### Exact helpers/fields added

- added durable gate transition metadata:
  - `phase_updated_at`
  - `phase_updated_by`
- added normalization/backfill for older gate records:
  - `phase_updated_at` defaults to `prepared_at`
  - `phase_updated_by` defaults to `operator`
- added durable phase helper:
  - `AdvanceHotUpdateGatePhase(root, hotUpdateID, nextState, updatedBy, updatedAt)`
- added adjacent phase validation helpers:
  - `isValidHotUpdateGatePhaseStartState(...)`
  - `isValidHotUpdateGateAdjacentTransition(...)`
- added taskstate wrapper:
  - `AdvanceHotUpdateGatePhase(jobID, hotUpdateID, phase)`
- added direct operator command:
  - `HOT_UPDATE_GATE_PHASE <job_id> <hot_update_id> <phase>`

### Exact tests added

- `internal/missioncontrol/hot_update_gate_registry_test.go`
  - `TestAdvanceHotUpdateGatePhaseValidProgressionAndPreservesActiveRuntimePackPointer`
  - `TestAdvanceHotUpdateGatePhaseIsIdempotentForSamePhase`
  - `TestAdvanceHotUpdateGatePhaseRejectsSkippedRegressiveAndInvalidStartingTransitions`
- `internal/agent/loop_processdirect_test.go`
  - `TestProcessDirectHotUpdateGatePhaseCommandAdvancesGateAndPreservesActiveRuntimePackPointer`

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

### Scope boundary statement

No apply/reload behavior, outcome creation, promotion behavior, rollback behavior, or runtime-pack pointer mutation was implemented in this slice.

### Deferred next V4 candidates

- read-only exposure of gate transition metadata if later operator slices require it
- bounded apply-adjacent hot-update gate execution only if a later checkpoint explicitly selects it
- any later hot-update outcome or promotion linkage only in separate slices
