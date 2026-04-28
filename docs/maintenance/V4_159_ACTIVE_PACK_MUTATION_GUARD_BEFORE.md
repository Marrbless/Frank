# V4-159 Active Pack Mutation Guard Before

Branch: `frank-v4-159-active-pack-mutation-guard`

## Requirement Rows

- `AC-029` was `PARTIAL`.

## Observed Gap

- Active pointer changes already flowed through hot-update and rollback records.
- Active component reads used the committed active pointer.
- Component admission required a hot-update gate context.
- Raw component storage could still backfill a component referenced by the committed active pack.
- Package imports could be written against a candidate pack that was already the active pack.

## Intended Slice

- Reject raw component writes when the committed active pack references the component.
- Reject package imports whose candidate pack is the current active pack.
- Use existing `E_ACTIVE_PACK_ADHOC_MUTATION_FORBIDDEN`.
- Preserve inactive candidate component/package paths for staged hot updates.
