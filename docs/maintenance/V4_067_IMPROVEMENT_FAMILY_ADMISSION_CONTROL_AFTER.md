# V4-067 Improvement Family Admission Control After

## Admission Rules Added

The V4 improvement-family boundary is now explicit through helper-backed validation:

- `improve_promptpack`
- `improve_skills`
- `improve_routing_manifest`
- `improve_runtime_extension`
- `evaluate_candidate`
- `promote_candidate`
- `rollback_candidate`
- `improve_topology`
- `propose_source_patch`

For V4 jobs, those families are admitted only when:

- `execution_plane=improvement_workspace`
- `execution_host` is compatible with the improvement workspace

## Execution-Host Compatibility

Compatible improvement-workspace hosts:

- `phone`
- `desktop_dev`
- `workspace`

The first two are directly from the frozen V4 spec. `workspace` is retained as a repo-supported adjacent host value from V4-065.

Known but incompatible hosts, such as `remote_provider`, reject when used by an improvement-family job in `improvement_workspace`.

## Rejection Codes Used

- Improvement family outside `improvement_workspace`: `E_LAB_ONLY_FAMILY`
- Improvement family in `improvement_workspace` with incompatible host: `E_IMPROVEMENT_WORKSPACE_REQUIRED`
- Live family on `improvement_workspace`: `E_EXECUTION_PLANE_INCOMPATIBLE`
- Hot-update family on `improvement_workspace`: `E_EXECUTION_PLANE_INCOMPATIBLE`
- Missing or unknown execution metadata: unchanged from V4-066

`E_IMPROVEMENT_WORKSPACE_REQUIRED` is used for incompatible improvement-workspace hosts because V4-066 did not add a host-specific incompatibility code and this is the closest existing V4 admission code for the workspace boundary.

## Compatibility Behavior

- Pre-V4 jobs still do not require execution metadata.
- No read-model fields were added.
- No commands were added.
- Existing V4 family-to-plane validation remains the single admission path; this slice only makes the improvement-family branch explicit and adds host compatibility.

## Invariants Preserved

- No adaptive lab was implemented.
- No mutation was implemented.
- No eval runs were implemented.
- No prompt-pack or skill-pack registry was implemented.
- No topology gate beyond admission metadata checks was implemented.
- No source-patch artifact policy beyond admission metadata checks was implemented.
- No promotion policy was implemented.
- No canary policy was implemented.
- No deploy lock was implemented.
- No new commands were added.
- No TaskState wrappers were added.
- No runtime-pack pointers were mutated.
- No `active_pointer.json` mutation.
- No `last_known_good_pointer.json` mutation.
- No `reload_generation` mutation.
- No runtime packs, candidates, eval suites, improvement runs, outcomes, promotions, rollbacks, or gates were created or mutated.
- No V4-068 work was started.

## Validation

- `/usr/local/go/bin/gofmt -w internal/missioncontrol/types.go internal/missioncontrol/types_test.go internal/missioncontrol/validate.go internal/missioncontrol/validate_test.go`
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
