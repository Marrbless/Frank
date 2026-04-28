# V4-144 Pack Component Admission Before

Branch: `frank-v4-144-pack-component-admission`

## Matrix Rows

- `AC-021` is `PARTIAL`: component records/loaders exist, but candidate component surfaces are not checked against pack/gate mutable surfaces.
- `AC-029` is `PARTIAL`: active pointer writes are gated, but component content admission does not yet require a hot-update gate context.
- `AC-024` remains `PARTIAL`: policy-surface mutation is rejected at plan metadata level, but content-level scanners are still later work.

## Slice

Add deterministic local component admission assessment for prompt, skill, manifest, and extension pack metadata:

- component records may declare surface class, declared mutable surfaces, and hot-reloadability,
- admission runs in hot-update gate context,
- missing gate/candidate/component records fail closed,
- undeclared mutable surfaces are blocked,
- immutable-surface touches are blocked,
- non-hot-reloadable component content is blocked.

This slice does not activate content and does not add extension permission widening policy.
