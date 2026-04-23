## V4-048 Hot-Update Gate Observability Before

- branch: `frank-v4-048-hot-update-gate-observability-read-model`
- HEAD: `16ee00a2e6ba9505e96bf16d721ed21263c79ff4`
- tag at HEAD:
  - `frank-v4-047-hot-update-state-machine-checkpoint`
- git status --short --branch at slice start:
  - `## frank-v4-048-hot-update-gate-observability-read-model`
- baseline `/usr/local/go/bin/go test -count=1 ./...` result:
  - pass when rerun outside the sandbox with Go build-cache and loopback socket access
  - initial sandboxed run failed because `/home/omar/.cache/go-build` was read-only and `httptest` could not bind loopback sockets

## Before-State Observability Gap

- V4-046 added deterministic hot-update terminal-failure resolution from `reload_apply_recovery_needed`.
- The committed `HotUpdateGateRecord` already persisted:
  - `failure_reason`
  - `phase_updated_at`
  - `phase_updated_by`
- The existing hot-update gate read model/status projection surfaced identity, linkage, prepared time, state, and decision.
- Operators still could not see terminal failure detail or phase transition metadata cleanly through the status/read-model surface without inspecting raw store JSON.

## Planned Read-Only Fields

- Add `failure_reason` to the existing hot-update gate status object.
- Add `phase_updated_at` to the existing hot-update gate status object.
- Add `phase_updated_by` to the existing hot-update gate status object.
- Preserve deterministic ordering through the existing sorted gate loader.
- Keep all changes read-only and projection-only.

## Explicit Non-Goals

- no new operator commands
- no new workflow states
- no new storage records
- no `HotUpdateOutcomeRecord` creation
- no `PromotionRecord` creation
- no active runtime-pack pointer mutation
- no `reload_generation` mutation
- no `last_known_good_pointer.json` mutation
- no retry behavior changes
- no terminal-failure behavior changes
- no automatic success or failure inference
- no V4-049 work
