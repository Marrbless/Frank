# V4-150 Restart Active Pack Loader After

Branch: `frank-v4-150-restart-active-pack-loader`

## Requirement Rows

- `AC-005` moved from `PARTIAL` to `DONE`.

## Implemented

- Added `CommittedActiveRuntimePackLoad` and `LoadCommittedActiveRuntimePackForRestart`.
- Restart/read loading now resolves the durable active pointer, active runtime-pack record, and all required prompt, skill, manifest, and extension component metadata.
- `ResolveActiveRuntimePackComponents` now delegates through the committed active restart/read loader.
- Rollback reload/apply convergence now requires the target active pack's component metadata, not only the target pack record.

## Validation

- Focused package test: `/usr/local/go/bin/go test -count=1 ./internal/missioncontrol`
- Required full validation: `git diff --check` and `/usr/local/go/bin/go test -count=1 ./...`

No process restart implementation, real plugin hot reload, network call, external service call, or phone hardware dependency was added.
