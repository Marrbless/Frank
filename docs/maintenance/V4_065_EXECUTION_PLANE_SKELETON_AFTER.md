# V4-065 Execution Plane Skeleton After

## Fields Added

- `Job.ExecutionPlane` as `execution_plane,omitempty`
- `Job.ExecutionHost` as `execution_host,omitempty`
- `Job.MissionFamily` as `mission_family,omitempty`

The same metadata is carried through runtime/read-model structs that already surface job metadata:

- `InspectablePlanContext`
- `JobRuntimeState`
- `RuntimeControlContext`
- `JobRuntimeRecord`
- `RuntimeControlRecord`
- `InspectSummary`
- `OperatorStatusSummary`

## Validation Rules Added

- V4 jobs use `spec_version=frank_v4`.
- For V4 jobs, `execution_plane`, `execution_host`, and `mission_family` are required.
- Unknown execution planes fail closed.
- Unknown execution hosts fail closed.
- Unknown mission families fail closed.
- Live and external V3/V4 families require `execution_plane=live_runtime`.
- Improvement families require `execution_plane=improvement_workspace`.
- Hot-update families require `execution_plane=hot_update_gate`.

## Compatibility Behavior

- Pre-V4 jobs are not required to declare the new fields.
- New JSON fields use `omitempty`; existing jobs and runtime records without the fields are not rewritten solely by this slice.
- Existing V2 validation remains unchanged except for unaffected enum additions.

## Read-Model / Status Exposure

- `mission inspect` summary data now includes `execution_plane`, `execution_host`, and `mission_family` when present.
- Operator status summary data now includes the same fields when present.
- Status falls back to `InspectablePlanContext` metadata when runtime-level metadata is absent but a validated plan snapshot carries it.
- JSON output remains deterministic through existing struct-ordered `json.MarshalIndent` formatting.

## Invariants Preserved

- No adaptive lab was implemented.
- No improvement records, candidates, eval suites, outcomes, promotions, rollbacks, or gates were created or mutated.
- No runtime-pack pointers were mutated.
- No `active_pointer.json` mutation.
- No `last_known_good_pointer.json` mutation.
- No `reload_generation` mutation.
- No hot-update behavior change.
- No new commands.
- No TaskState wrappers.
- No source-patch, topology, deploy-lock, canary, promotion-policy, prompt-pack registry, or skill-pack registry work.
- No V4-066 work.

## Validation

- `/usr/local/go/bin/gofmt -w internal/missioncontrol/types.go internal/missioncontrol/validate.go internal/missioncontrol/validate_test.go internal/missioncontrol/status.go internal/missioncontrol/status_test.go internal/missioncontrol/*status*_test.go internal/missioncontrol/runtime.go internal/missioncontrol/inspect.go internal/missioncontrol/inspect_test.go internal/missioncontrol/store_records.go internal/missioncontrol/store_records_test.go internal/missioncontrol/store_mutate.go internal/missioncontrol/store_hydrate.go internal/missioncontrol/types_test.go`
  - passed
- `git diff --check`
  - passed
- `/usr/local/go/bin/go test -count=1 ./internal/missioncontrol`
  - passed
- `/usr/local/go/bin/go test -count=1 ./cmd/picobot`
  - passed
- `/usr/local/go/bin/go test -count=1 ./internal/agent/tools`
  - passed
- `/usr/local/go/bin/go test -count=1 ./internal/agent`
  - passed
- `/usr/local/go/bin/go test -count=1 ./...`
  - passed
