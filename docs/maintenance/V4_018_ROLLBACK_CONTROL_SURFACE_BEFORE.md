Branch

- `frank-v4-018-rollback-control-surface`

HEAD

- `1217601228ba51523fabe6ff3412ad3c7baa1370`

Tags At HEAD

- `frank-v4-017-rollback-read-model`

Ahead/Behind Upstream

- `416 0`

Git Status --short --branch

```text
## frank-v4-018-rollback-control-surface
```

Baseline `go test -count=1 ./...` Result

- `PASS`

Exact Files Planned

- `internal/agent/loop.go`
- `internal/agent/tools/taskstate.go`
- `internal/agent/loop_processdirect_test.go`
- `docs/maintenance/V4_018_ROLLBACK_CONTROL_SURFACE_AFTER.md`

Exact Surface Chosen

- Existing operator direct control-command surface in `internal/agent/loop.go`
- New record-only command form: `rollback_record <job_id> <promotion_id> <rollback_id>`
- `TaskState` helper in `internal/agent/tools/taskstate.go` that derives a rollback record from an existing committed promotion record and stores it through the existing rollback ledger contract
- Existing `STATUS <job_id>` read-only surface for inspecting the resulting rollback identity through the already-committed rollback read-model

Exact Non-Goals

- No rollback apply behavior
- No runtime pack activation changes
- No provider or channel behavior changes beyond the new operator command parse/ack path
- No evaluator or scoring behavior
- No autonomy behavior changes
- No promotion workflow changes
- No rollback target-selection workflow beyond deterministic proposal derivation from the promotion record
- No dependency changes
- No cleanup outside this bounded control-plane slice
