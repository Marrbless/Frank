## V4-037 Hot-Update Gate Phase Control Before

- branch: `frank-v4-037-hot-update-gate-phase-control`
- HEAD: `6c3d1f98dfe91eae285ebe3634cf9d0b780d77f5`
- tags at HEAD:
  - none
- ahead/behind upstream:
  - `435 0`
- git status --short --branch:
  - `## frank-v4-037-hot-update-gate-phase-control`
- baseline `go test -count=1 ./...` result:
  - passed

### Exact files planned

- `internal/missioncontrol/hot_update_gate_registry.go`
- `internal/missioncontrol/hot_update_gate_registry_test.go`
- `internal/agent/tools/taskstate.go`
- `internal/agent/loop.go`
- `internal/agent/loop_processdirect_test.go`
- `docs/maintenance/V4_037_HOT_UPDATE_GATE_PHASE_CONTROL_BEFORE.md`
- `docs/maintenance/V4_037_HOT_UPDATE_GATE_PHASE_CONTROL_AFTER.md`

### Exact transition model planned

- extend `HotUpdateGateRecord` with:
  - `phase_updated_at`
  - `phase_updated_by`
- normalize older records by backfilling transition metadata from `prepared_at`
- add the smallest durable phase helper for hot-update gates with:
  - same-phase replay idempotent
  - allowed adjacent forward transitions only:
    - `prepared -> validated`
    - `validated -> staged`
  - rejected skipped transition:
    - `prepared -> staged`
  - rejected regressive transitions including:
    - `validated -> prepared`
    - `staged -> validated`
- reuse the existing direct operator command surface with a minimal new command:
  - `HOT_UPDATE_GATE_PHASE <job_id> <hot_update_id> <phase>`

### Exact non-goals

- no apply/reload behavior
- no outcome creation
- no promotion behavior
- no rollback behavior
- no runtime-pack pointer mutation
- no evaluator execution
- no scoring behavior
- no autonomy changes
- no provider/channel behavior changes
- no dependency changes
- no cleanup outside this slice
- no commit
