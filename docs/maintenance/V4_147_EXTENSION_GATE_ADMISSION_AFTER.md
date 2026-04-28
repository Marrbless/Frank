# V4-147 Extension Gate Admission After

Branch: `frank-v4-147-extension-gate-admission`

## Requirement Rows

- `AC-031` moved from `PARTIAL` to `DONE`.
- `AC-027` remains `PARTIAL` until package/donor authority checks are implemented.

## Implemented

- Hot-update gate storage now compares the previous active runtime pack extension ref with the candidate runtime pack extension ref.
- Direct gate admission rejects extension permission widening, new external-side-effect tools, compatibility contract changes, and missing extension metadata.
- Gate replay/linkage validation re-runs the same extension admission check so stale or corrupted extension metadata fails closed.
- Runtime-pack test fixtures now seed deterministic extension pack metadata for valid unchanged-extension paths.

## Validation

- Focused package test: `/usr/local/go/bin/go test -count=1 ./internal/missioncontrol`
- Required full validation: `git diff --check` and `/usr/local/go/bin/go test -count=1 ./...`

No approval authority, network call, external service call, device side effect, or real plugin hot reload was added.
