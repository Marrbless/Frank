## V4-039 Hot-Update Pointer Switch Before

- branch: `frank-v4-039-hot-update-pointer-switch-skeleton`
- HEAD: `93a4eb678ff5b9b964467b06be236cb867e49c20`
- tags at HEAD:
  - `frank-v4-037-hot-update-gate-phase-control`
- ahead/behind upstream:
  - `437 0`
- git status --short --branch:
  - `## frank-v4-039-hot-update-pointer-switch-skeleton`
- baseline `go test -count=1 ./...` result:
  - passed

### Exact files planned

- `internal/missioncontrol/hot_update_gate_registry.go`
- `internal/missioncontrol/hot_update_gate_registry_test.go`
- `internal/agent/tools/taskstate.go`
- `internal/agent/loop.go`
- `internal/agent/loop_processdirect_test.go`
- `docs/maintenance/V4_039_HOT_UPDATE_POINTER_SWITCH_BEFORE.md`
- `docs/maintenance/V4_039_HOT_UPDATE_POINTER_SWITCH_AFTER.md`

### Exact state transitions planned

- add the smallest hot-update gate execution helper with:
  - required start state `staged`
  - success transition `staged -> reloading`
  - exact replay in `reloading` is idempotent when the active pointer already matches `hot_update:<hot_update_id>`
- active pointer mutation on success:
  - `active_pack_id -> candidate_pack_id`
  - `previous_active_pack_id -> gate.previous_active_pack_id`
  - `update_record_ref -> hot_update:<hot_update_id>`
  - `reload_generation += 1`
- recovery-safe replay path:
  - if pointer already matches the candidate pack and `update_record_ref` already matches `hot_update:<hot_update_id>`, persist `reloading` without incrementing again

### Exact non-goals

- no reload/apply convergence mechanics
- no smoke/canary execution
- no `HotUpdateOutcomeRecord` creation
- no `PromotionRecord` creation
- no `last_known_good_pointer.json` mutation
- no rollback behavior changes
- no evaluator execution
- no scoring behavior
- no autonomy changes
- no provider/channel behavior changes
- no cleanup outside this slice
- no dependency changes
- no commit
