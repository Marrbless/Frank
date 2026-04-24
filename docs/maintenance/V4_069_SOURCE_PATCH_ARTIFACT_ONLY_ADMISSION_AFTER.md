# V4-069 Source-Patch Artifact-Only Admission After

## Source-Patch Artifact-Only Admission Rules

For `spec_version=frank_v4` jobs with `mission_family=propose_source_patch`:

- Existing V4-067 validation still requires `execution_plane=improvement_workspace`.
- Existing V4-068 validation still requires at least one target or mutable surface with class `source_patch_artifact`.
- Every declared `target_surfaces` entry must use class `source_patch_artifact`.
- Every declared `mutable_surfaces` entry must use class `source_patch_artifact`.
- `immutable_surfaces` remain allowed for evaluator, rubric, baseline, and policy refs using already-supported classes.

No path-shape heuristic was added because job surface refs do not yet have a stable repo-wide artifact-path convention. Admission is class-based in this slice.

## Classes Rejected For `propose_source_patch`

The following target/mutable classes now reject for `propose_source_patch`:

- `prompt_pack`
- `skill`
- `manifest_entry`
- `skill_topology`

`source_patch_artifact` behavior for other mission families was not changed.

## Rejection Codes Used

- Direct target/mutable source mutation class on `propose_source_patch`: `E_RUNTIME_SOURCE_MUTATION_FORBIDDEN`
- Missing required `source_patch_artifact` target/mutable class: existing V4-068 `E_RUNTIME_SOURCE_MUTATION_FORBIDDEN`
- Missing target/mutable declarations and other generic surface declaration errors: unchanged from V4-068

## Compatibility Behavior

- Pre-V4 jobs remain backward compatible.
- Non-`propose_source_patch` improvement families are not affected by the artifact-only class restriction.
- Existing read-model/status surfaces are unchanged.
- Existing runtime-pack, hot-update gate, and improvement-run records are unchanged.

## Invariants Preserved

- No patch generation was implemented.
- No patch application was implemented.
- No source deployment was implemented.
- No runtime-source mutation was implemented.
- No adaptive lab was implemented.
- No eval runs were implemented.
- No topology enablement was implemented.
- No promotion policy was implemented.
- No canary policy was implemented.
- No deploy lock was implemented.
- No commands were added.
- No TaskState wrappers were added.
- No runtime-pack pointers were mutated.
- No `active_pointer.json` mutation.
- No `last_known_good_pointer.json` mutation.
- No `reload_generation` mutation.
- No runtime packs, candidates, eval suites, improvement runs, outcomes, promotions, rollbacks, or gates were created or mutated.
- No V4-070 work was started.

## Validation

Validation was run after implementation with the commands listed in the V4-069 handoff.
