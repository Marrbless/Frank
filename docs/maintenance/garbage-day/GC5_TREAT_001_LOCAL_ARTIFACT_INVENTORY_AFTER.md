# GC5-TREAT-001 Local Artifact Inventory After

Branch: `frank-garbage-day-gc4-005-maintenance-artifact-retention`

## Completed

- Inventoried ignored local runtime/build artifacts without deleting files.
- Confirmed the tracked worktree was clean before this docs-only inventory.
- Left all operator-local state in place.

## Inventory

| Path | Status | Size | Count |
| --- | --- | ---: | ---: |
| `.codex` | ignored | `0` | `1` file |
| `internal/agent/memory` | ignored markdown memory files | `336K` | `38` files |
| `internal/agent/sessions` | ignored session data | `4.0K` | `2` files |
| `missions` | ignored mission data | `4.0K` | `1` file |
| `picobot` | ignored local binary | `33M` | `1` file |

## Decision

No deletion was performed. Deleting any of these paths requires a separate explicit approval naming the exact paths.

## Validation

- `git status --short --branch`
- `git status --short --branch --ignored`
- `du -sh .codex internal/agent/memory internal/agent/sessions missions picobot`
- `find internal/agent/memory -maxdepth 1 -type f -name '*.md' | wc -l`
- `find internal/agent/sessions -type f | wc -l`
- `find missions -type f | wc -l`
- `find .codex -type f | wc -l`

## Remaining

Continue with `GC5-002` from `REPO_WIDE_GARBAGE_CAMPAIGN_MATRIX.md`.
