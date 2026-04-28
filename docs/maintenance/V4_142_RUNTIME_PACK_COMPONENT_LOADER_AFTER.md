# V4-142 Runtime Pack Component Loader After

Branch: `frank-v4-142-runtime-pack-component-loader`

## Requirement Rows

- `SF-002` moved from `MISSING` to `DONE`.
- `AC-005`, `AC-021`, `AC-029`, `AC-030`, and `SF-003` remain `PARTIAL` with new loader evidence and narrower next slices.

## Implemented

- Added `RuntimePackComponentRecord` for first-class local prompt, skill, manifest, and extension pack metadata.
- Added required content ref, SHA-256 content identity, provenance, source summary, creation metadata, and optional parent component identity.
- Added store/load/list helpers with exact replay idempotence and divergent duplicate rejection.
- Added `ResolveRuntimePackComponents` for a runtime pack's four component refs.
- Added `ResolveActiveRuntimePackComponents` so local callers can resolve component metadata from the committed active pointer only.

## Validation

- Focused package test: `/usr/local/go/bin/go test -count=1 ./internal/missioncontrol`
- Required full validation: `git diff --check` and `/usr/local/go/bin/go test -count=1 ./...`

No activation path, network call, external service call, or real plugin hot reload was added.
