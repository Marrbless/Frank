# V4-143 Reload Component Refresh After

Branch: `frank-v4-143-reload-component-refresh`

## Requirement Rows

- `SF-003` moved from `PARTIAL` to `DONE`.
- `AC-030` moved from `PARTIAL` to `DONE`.
- `AC-005` remains `PARTIAL` because the non-gate restart/runtime read path still needs to require active component resolution.

## Implemented

- `hotUpdateGateRestartStyleConvergence` now calls `ResolveActiveRuntimePackComponents`.
- Reload/apply still verifies the active pointer points to the candidate pack and has the expected gate update ref.
- Missing or invalid active prompt, skill, manifest, or extension component metadata now fails reload/apply.
- Normal runtime-pack test fixtures seed deterministic local component records so existing hot-update paths continue to model valid local metadata.
- Added a reload/apply failure test proving missing candidate component metadata records a failed phase without mutating the active pointer or last-known-good pointer.

## Validation

- Focused package test: `/usr/local/go/bin/go test -count=1 ./internal/missioncontrol`
- Required full validation: `git diff --check` and `/usr/local/go/bin/go test -count=1 ./...`

No activation bypass, network call, external service call, or real plugin hot reload was added.
