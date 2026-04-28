# V4-142 Runtime Pack Component Loader Before

Branch: `frank-v4-142-runtime-pack-component-loader`

## Matrix Rows

- `SF-002` is `MISSING`: runtime pack records contain prompt, skill, manifest, and extension refs, but those refs do not resolve to first-class local content metadata records.
- `SF-003` remains `PARTIAL`: reload/apply currently verifies pointer convergence and increments reload generation, but does not yet refresh component metadata through loaders.
- `AC-005`, `AC-021`, `AC-029`, and `AC-030` remain `PARTIAL` because content surfaces are not yet first-class loadable pack records.

## Slice

Add the smallest local deterministic pack component registry/loader foundation for prompt, skill, manifest, and extension pack metadata:

- durable component records keyed by component kind and ID,
- required content ref, SHA-256 identity, provenance, source summary, creation metadata,
- idempotent exact replay and divergent duplicate rejection,
- runtime pack component resolver for the four component refs,
- active runtime pack component resolver that starts from the committed active pointer.

This slice does not activate component content and does not implement real plugin hot reload.
