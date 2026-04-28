# V4-153 Workspace Crash Record Before

Branch: `frank-v4-153-workspace-crash-record`

## Requirement Rows

- `AC-015` was `PARTIAL`.

## Observed Gap

- Candidate and improvement records were separate from the active runtime pointer.
- Hot-update gates controlled active pointer changes.
- There was no local deterministic workspace-run failure/crash record proving that a failed improvement workspace attempt left the committed active pointer unchanged.

## Intended Slice

- Add an append-only improvement workspace run record for local crash/failure outcomes.
- Record active-pointer snapshots at workspace start and completion.
- Reject records when the completion snapshot differs from the current active pointer.
- Preserve the active runtime pointer during failure recording.
