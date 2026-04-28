# V4-164 Autonomy Owner Pause Before

Branch: `frank-v4-164-autonomy-owner-pause`

## Matrix Row

- Requirement: `AC-037`
- Status before slice: `MISSING`
- Gap: owner approval gates existed for hot-update lifecycle records, but no durable autonomy owner-pause record stopped autonomy-originated hot-update proposals.

## Intended Slice

Add deterministic local owner pause behavior:

- owner-pause records with explicit authority refs,
- rejection of natural-language approval binding,
- hot-update proposal blocking with `E_AUTONOMY_PAUSED`,
- status/read-model surfacing.

Manual/operator hot-update controls and non-hot autonomous proposals are not changed.
