# V4-068 Improvement Target Surface Skeleton Before

## Before-State Gap

V4-064 identified missing job/proposal-level target-surface enforcement as one of the remaining controlled self-improvement gaps. V4-065 added job-level `execution_plane`, `execution_host`, and `mission_family`, V4-066 added V4 rejection-code constants, and V4-067 made improvement-family admission explicit.

Before this slice, V4 improvement-family jobs could be admitted to `improvement_workspace` without declaring which surfaces were mutable or which evaluator/policy/baseline surfaces were immutable for the run. Runtime packs and hot-update gates already had surface fields, but the job/read-model boundary did not.

## Intended Slice

Add the smallest schema and validation skeleton for job-level V4 improvement surface declarations:

- target surfaces
- mutable surfaces
- immutable surfaces

This slice is only a declaration and validation skeleton. It does not execute improvement runs or mutate any pack, gate, pointer, or ledger.

## Invariants To Preserve

- No adaptive lab execution.
- No mutation of improvement records, runtime packs, candidates, eval suites, outcomes, promotions, rollbacks, or gates.
- No runtime-pack pointer mutation.
- No `active_pointer.json` mutation.
- No `last_known_good_pointer.json` mutation.
- No `reload_generation` mutation.
- No hot-update behavior change.
- No new commands.
- No TaskState wrappers.
- No prompt-pack or skill-pack registry.
- No topology enablement.
- No source-patch artifact policy enforcement beyond declaration checks.
- No promotion policy, canary policy, deploy lock, or V4-069 work.
