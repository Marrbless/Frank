# V4-153 Workspace Crash Record After

Branch: `frank-v4-153-workspace-crash-record`

## Requirement Rows

- `AC-015` moved from `PARTIAL` to `DONE`.
- `AC-017` evidence was updated to include the new append-only workspace run record family.

## Implemented

- Added `ImprovementWorkspaceRunRecord` under `runtime_packs/improvement_workspace_runs`.
- Added deterministic crash/failure outcomes for local workspace runner failures.
- Stored active-pointer snapshots at start and completion.
- Required crash/failure records to keep the active-pointer start and completion snapshots identical.
- Store-time validation rejects records whose completion snapshot does not match the current committed active pointer.
- Storage is idempotent for replay and rejects divergent duplicates.

## Validation

- Focused package test: `/usr/local/go/bin/go test -count=1 ./internal/missioncontrol`
- Required full validation: `git diff --check` and `/usr/local/go/bin/go test -count=1 ./...`

No real process supervision, phone hardware, active-pointer mutation path, network call, external service call, or device side effect was added.
