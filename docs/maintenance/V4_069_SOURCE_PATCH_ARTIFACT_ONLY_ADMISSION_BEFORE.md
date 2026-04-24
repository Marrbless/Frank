# V4-069 Source-Patch Artifact-Only Admission Before

## Before-State Gap From V4-068

V4-068 added job-level surface declarations and required `propose_source_patch` jobs to include a `source_patch_artifact` target or mutable surface class.

Before this slice, a V4 `propose_source_patch` job could still declare `source_patch_artifact` alongside another target or mutable class such as `prompt_pack`, `skill`, `manifest_entry`, or `skill_topology`. That left the source-patch family’s artifact-only boundary under-specified at admission time.

## Intended Slice

Add validation-only admission hardening so `propose_source_patch` can declare patch artifacts but cannot declare direct runtime-source mutation/deployment surfaces.

## Invariants To Preserve

- No source patch artifact generation.
- No patch application.
- No source deployment.
- No runtime-source mutation.
- No adaptive lab execution.
- No eval runs.
- No topology enablement.
- No promotion policy.
- No canary policy.
- No deploy lock.
- No commands.
- No TaskState wrappers.
- No runtime-pack pointer mutation.
- No `active_pointer.json` mutation.
- No `last_known_good_pointer.json` mutation.
- No `reload_generation` mutation.
- No runtime packs, candidates, eval suites, improvement runs, outcomes, promotions, rollbacks, or gates created or mutated.
- No V4-070 work.
