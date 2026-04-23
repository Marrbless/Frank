## V4-048 Hot-Update Gate Observability After

### Files Changed

- `internal/missioncontrol/status.go`
- `internal/missioncontrol/status_hot_update_gate_identity_test.go`
- `internal/agent/loop_processdirect_test.go`
- `docs/maintenance/V4_048_HOT_UPDATE_GATE_OBSERVABILITY_BEFORE.md`
- `docs/maintenance/V4_048_HOT_UPDATE_GATE_OBSERVABILITY_AFTER.md`

### Exact Read-Only Fields Surfaced

- Added `failure_reason` to `OperatorHotUpdateGateStatus`.
- Added `phase_updated_at` to `OperatorHotUpdateGateStatus`.
- Added `phase_updated_by` to `OperatorHotUpdateGateStatus`.
- These fields are projected from existing `HotUpdateGateRecord` storage only.
- No storage schema, workflow, or command behavior changed.

### Implemented Observability Behavior

- Hot-update gate identity/status output now shows deterministic terminal failure detail when `failure_reason` is present.
- Hot-update gate identity/status output now shows transition timestamp and actor metadata when `phase_updated_at` and `phase_updated_by` are present.
- Existing gate ordering remains deterministic because the loader still sorts gate filenames before projection.
- Existing status surfaces are reused:
  - committed mission status snapshot
  - `TaskState`/direct `STATUS` readout
  - hot-update gate identity read model

### Tests Added Or Expanded

- Added missioncontrol read-model coverage proving:
  - terminal failure detail is visible
  - phase transition metadata is visible
  - output is deterministic across repeated loads
  - active runtime-pack pointer bytes are unchanged
  - `reload_generation` is unchanged
  - `last_known_good_pointer.json` bytes are unchanged
  - no `HotUpdateOutcomeRecord` is created
  - no `PromotionRecord` is created
  - no new hot-update gate record is created
- Expanded direct `STATUS` coverage proving:
  - terminal failure detail is visible through the operator readout
  - phase transition metadata is visible through the operator readout

### Invariants Preserved

- No new operator command was added.
- No workflow state was added.
- No storage record type was added.
- No `HotUpdateOutcomeRecord` was created.
- No `PromotionRecord` was created.
- No active runtime-pack pointer mutation was implemented.
- No `reload_generation` mutation was implemented.
- No `last_known_good_pointer.json` mutation was implemented.
- No retry behavior was changed.
- No terminal-failure behavior was changed.
- No automatic success or failure inference was added.

### Validation Commands

- `/usr/local/go/bin/gofmt -w internal/missioncontrol/status.go internal/missioncontrol/status_hot_update_gate_identity_test.go internal/agent/loop_processdirect_test.go internal/agent/tools/taskstate.go internal/agent/tools/taskstate_test.go internal/agent/loop.go internal/missioncontrol/hot_update_gate_registry.go internal/missioncontrol/hot_update_gate_registry_test.go`
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

- Go validation was run outside the sandbox where necessary because the sandbox blocks the Go build cache and loopback sockets used by existing tests.
- This slice intentionally stops at read-only observability polish and does not start V4-049.
