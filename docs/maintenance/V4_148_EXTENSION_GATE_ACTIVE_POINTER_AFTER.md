# V4-148 Extension Gate Active Pointer After

Branch: `frank-v4-148-extension-gate-active-pointer`

## Requirement Rows

- `AC-031` remains `DONE`.
- `AC-027` remains `PARTIAL` until package/donor authority checks are implemented.

## Implemented

- Direct hot-update gate admission now verifies that `previous_active_pack_id` matches the current active runtime-pack pointer before extension permission assessment trusts the baseline.
- Added a regression test where a gate attempts to use a fake previous pack with the candidate extension ref while the active pointer still names another pack.
- Rejected mismatched gates are not persisted.

## Validation

- Focused package test: `/usr/local/go/bin/go test -count=1 ./internal/missioncontrol`
- Required full validation: `git diff --check` and `/usr/local/go/bin/go test -count=1 ./...`

No approval authority, network call, external service call, device side effect, or real plugin hot reload was added.
