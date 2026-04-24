# V4-070 Topology Mode Disabled Admission After

## Field Added

Added job-level `topology_mode_enabled` as a boolean flag.

The flag is carried through the same V4 job metadata/read-model path as the execution-plane and surface declarations:

- `Job`
- `InspectablePlanContext`
- `JobRuntimeState`
- `RuntimeControlContext`
- `JobRuntimeRecord`
- `RuntimeControlRecord`
- `InspectSummary`
- `OperatorStatusSummary`

## Validation Behavior

For `spec_version=frank_v4` jobs with `mission_family=improve_topology`:

- Existing V4-067 validation still requires `execution_plane=improvement_workspace`.
- Existing V4-067 validation still requires a workspace-compatible execution host.
- Existing V4-068 validation still requires a `skill_topology` target or mutable surface.
- `topology_mode_enabled=false` or omitted rejects with `E_TOPOLOGY_CHANGE_DISABLED`.
- `topology_mode_enabled=true` may pass admission when all existing plane, host, and surface validation also passes.

No add/remove/split/merge topology operation was implemented.

## Validation Ordering

The validator preserves existing admission ordering:

- Wrong execution plane rejects through the existing family/plane validator before topology-mode validation runs.
- In `improvement_workspace`, missing `skill_topology` target/mutable surface is reported before disabled topology mode.
- With `topology_mode_enabled=true`, missing `skill_topology` still rejects through the existing V4-068 surface rule.

## Rejection Code Used

- Disabled or omitted topology mode for `improve_topology`: `E_TOPOLOGY_CHANGE_DISABLED`

## Compatibility Behavior

- Pre-V4 jobs remain backward compatible.
- Non-topology improvement families are not affected by the absence of `topology_mode_enabled`.
- New JSON fields use `omitempty`; existing records without the field are not rewritten solely by this slice.
- Existing runtime-pack, hot-update gate, improvement-run, and source-patch records are unchanged.

## Invariants Preserved

- No topology mutation was implemented.
- No skill-pack registry was implemented.
- No adaptive lab was implemented.
- No eval runs were implemented.
- No baseline/train/holdout logic was implemented.
- No source-patch deployment was implemented.
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
- No V4-071 work was started.

## Validation

Validation was run after implementation with the commands listed in the V4-070 handoff.
