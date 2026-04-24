# V4-068 Improvement Target Surface Skeleton After

## Fields / Structs Added

Added job-level `JobSurfaceRef` declarations:

- `class`
- `ref`

Added these fields where job metadata is already carried or surfaced:

- `target_surfaces`
- `mutable_surfaces`
- `immutable_surfaces`

The fields are present on `Job`, `InspectablePlanContext`, `JobRuntimeState`, `RuntimeControlContext`, `JobRuntimeRecord`, `RuntimeControlRecord`, `InspectSummary`, and `OperatorStatusSummary`.

## Surface Classes Supported

- `prompt_pack`
- `skill`
- `manifest_entry`
- `skill_topology`
- `source_patch_artifact`

## Validation Behavior

For `spec_version=frank_v4` improvement-family jobs admitted to `execution_plane=improvement_workspace` with a workspace-compatible host:

- At least one `target_surfaces` or `mutable_surfaces` entry is required.
- `immutable_surfaces` is required.
- Empty or whitespace-only `ref` values reject.
- Duplicate surface refs reject deterministically within each declared surface field.
- Unknown or missing surface classes reject.
- `improve_topology` requires a declared `skill_topology` target/mutable surface class.
- `propose_source_patch` requires a declared `source_patch_artifact` target/mutable surface class.

Non-improvement V4 jobs are not forced to declare these surfaces. Pre-V4 jobs remain backward compatible.

## Rejection Codes Used

- Missing target/mutable surface declaration: `E_MUTATION_SCOPE_VIOLATION`
- Empty or duplicate target/mutable refs: `E_MUTATION_SCOPE_VIOLATION`
- Missing immutable surfaces: `E_FORBIDDEN_SURFACE_CHANGE`
- Empty or duplicate immutable refs: `E_FORBIDDEN_SURFACE_CHANGE`
- Missing or unknown surface class: `E_SURFACE_CLASS_REQUIRED`
- Missing topology target class for `improve_topology`: `E_MUTATION_SCOPE_VIOLATION`
- Missing source-patch artifact target class for `propose_source_patch`: `E_RUNTIME_SOURCE_MUTATION_FORBIDDEN`

## Read-Model / Status Exposure

Existing inspect/status/read-model surfaces now expose `target_surfaces`, `mutable_surfaces`, and `immutable_surfaces` where V4 job metadata is already surfaced. JSON output remains deterministic through existing struct-ordered marshaling.

## Compatibility Behavior

- New fields use `omitempty`; existing records without these fields are not rewritten solely by this slice.
- Validation is scoped to V4 improvement-family jobs only after the V4-067 execution-plane and host admission boundary is satisfied.
- Existing runtime-pack, hot-update gate, and improvement-run surface fields were not changed.

## Invariants Preserved

- No adaptive lab was implemented.
- No mutation was implemented.
- No eval runs were implemented.
- No prompt-pack or skill-pack registry was implemented.
- No topology enablement was implemented.
- No source-patch artifact application or deployment policy was implemented beyond declaration checks.
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
- No V4-069 work was started.

## Validation

Validation was run after implementation with the commands listed in the V4-068 handoff.
