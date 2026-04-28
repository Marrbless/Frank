# V4-143 Reload Component Refresh Before

Branch: `frank-v4-143-reload-component-refresh`

## Matrix Rows

- `SF-003` is `PARTIAL`: reload/apply re-resolves active pointer and increments reload generation; V4-142 provides component loaders/resolvers but reload/apply does not require them.
- `AC-030` is `PARTIAL`: Pi-like reload is gate-bound, but local component metadata refresh is not part of reload/apply convergence yet.
- `AC-005` is `PARTIAL`: active component resolution exists, but restart/reload-style paths do not fail closed on missing component metadata.

## Slice

Wire the existing hot-update reload/apply convergence through `ResolveActiveRuntimePackComponents` after pointer convergence. Missing or invalid prompt, skill, manifest, or extension component records should cause reload/apply failure without mutating the active pointer or last-known-good pointer.

This slice preserves the existing pointer convergence behavior and does not implement real plugin hot reload.
