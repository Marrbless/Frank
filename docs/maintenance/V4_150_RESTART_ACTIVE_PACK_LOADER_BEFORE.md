# V4-150 Restart Active Pack Loader Before

Branch: `frank-v4-150-restart-active-pack-loader`

## Requirement Rows

- `AC-005` was `PARTIAL`.

## Observed Gap

- Hot-update reload/apply already resolved prompt, skill, manifest, and extension component metadata from the committed active pointer after pointer switch.
- A named restart/read path for the committed active pack outside hot-update reload/apply was still missing.
- Rollback reload convergence still resolved only the target runtime-pack record and did not require target pack component metadata.

## Intended Slice

- Add a deterministic local restart/read loader over the committed active runtime-pack pointer.
- Require the active pack's prompt, skill, manifest, and extension component metadata.
- Preserve pointer convergence behavior and avoid process restart, real plugin reload, network calls, external services, or phone hardware requirements.
