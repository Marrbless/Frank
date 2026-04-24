# V4-070 Topology Mode Disabled Admission Before

## Before-State Gap

V4-068 added job-level surface declarations and required `improve_topology` jobs to declare a `skill_topology` target or mutable surface. V4-069 did not change topology behavior.

Before this slice, a V4 `improve_topology` job with `execution_plane=improvement_workspace`, a compatible host, immutable surfaces, and a `skill_topology` surface was admitted by validation. The frozen V4 spec requires topology changes to be disabled by default unless topology mode is explicitly enabled.

## Intended Slice

Add the smallest job-level topology-mode admission flag and reject `improve_topology` by default.

## Invariants To Preserve

- No topology mutation.
- No skill-pack registry.
- No adaptive lab execution.
- No eval runs.
- No baseline/train/holdout logic.
- No source-patch deployment.
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
- No V4-071 work.
