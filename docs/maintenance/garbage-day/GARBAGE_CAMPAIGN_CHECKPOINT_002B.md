# Garbage Campaign Checkpoint After GD2-TREAT-002B

Date: 2026-04-20

## Live State

- Canonical repo: `/mnt/d/pbot/picobot`
- Current branch: `frank-v3-foundation`
- Current HEAD: `7415eb20696a5c94db7f7275d8b4dd224230451c`
- Ahead/behind `upstream/main`: `369 ahead / 0 behind`
- Worktree state at checkpoint: clean
- Repo green at checkpoint: yes
  - Evidence: `go test -count=1 ./...` passed at the above `HEAD`

## Completed Campaign Lanes

### 009. Upstream integration

- Status: complete
- Evidence:
  - `docs/maintenance/garbage-day/GD2_TREAT_009_UPSTREAM_INTEGRATION_ASSESSMENT.md`
  - `docs/maintenance/garbage-day/GD2_TREAT_009B_UPSTREAM_INTEGRATION_RESOLUTION.md`
  - `docs/maintenance/garbage-day/GD2_TREAT_009C_UPSTREAM_INTEGRATION_VERIFY.md`
- Exact major risks removed:
  - removed the repo-behind-upstream state for `upstream/main`
  - removed unresolved overlap risk around `cmd/picobot/main.go`, `internal/agent/loop.go`, and `docs/HOW_TO_START.md`
  - landed upstream MCP support and tool-activity gating without weakening Frank mission-control, approval, treasury, capability, or owner-control semantics
  - removed ambiguity about whether upstream JSON/MCP changes could coexist with Frank V3 protected surfaces

### 001A. Session path hardening and startup rehydration

- Status: complete
- Exact major risks removed:
  - removed session-path traversal and filename-collision risk from external `channel:chatID` values
  - removed the startup rehydration gap where session files existed on disk but were not loaded
  - removed fail-open startup behavior for malformed session files

### 001B. Atomic session and memory overwrite writes

- Status: complete
- Exact major risks removed:
  - removed the crash window where session overwrites or memory overwrites could leave truncated or zero-length files
  - aligned those overwrite paths with temp-file + sync + rename durability semantics already used in missioncontrol

### 001C. Append durability and remember fail-closed behavior

- Status: complete
- Exact major risks removed:
  - removed the false-success remember path that claimed persistence succeeded when it had failed
  - removed undurable append behavior that wrote without explicit file sync and ignored close failure

### 001D. Long-memory append replay and concurrency hardening

- Status: complete
- Exact major risks removed:
  - removed the in-process long-memory read-modify-write race that could lose concurrent appends inside one `MemoryStore`
  - removed immediate retry duplication at the file tail for the same append request

### 002A. Provider and MCP error-surface redaction

- Status: complete
- Exact major risks removed:
  - removed raw non-2xx provider body logging
  - removed surfaced provider and MCP error payloads that could include prompt fragments, secrets, provider account data, or remote error bodies
  - removed raw tool failure notifications and model-visible tool error payloads in the treated paths
  - removed raw `ProcessDirect` provider error exposure

### 002B. Tool-arg and channel-content log minimization

- Status: complete
- Exact major risks removed:
  - removed raw tool-argument key exposure from registry logs and tool-activity notifications
  - removed inbound Slack, Discord, and WhatsApp raw content fragments from logs
  - removed Slack and Discord attachment-URL exposure from inbound logs
  - narrowed treated log surfaces to structural summaries that preserve operator signal without raw content

## Major Risks Removed So Far

- The branch is now upstream-contained and not carrying unresolved integration debt.
- The session and memory layer is materially harder to corrupt or silently lie about.
- The highest-severity provider and MCP error-surface leakage has been reduced.
- The most obvious remaining raw tool-arg and inbound channel-content log noise has been reduced.
- MCP and tool-activity support now exist on this branch without expanding Frank authority semantics.

## Remaining Major Disease Clusters From Round 2

### 1. Secret-safe onboarding and docs hygiene

- Still open as `GD2-TREAT-002C`.
- Main remaining issues:
  - provider and channel secrets are still entered with normal terminal echo during onboarding
  - docs still normalize plaintext secret handling more than they should
  - startup/debug omission policy is not yet documented as a clean operator contract

### 2. Overgrown protected-surface files

- Still open and still the strongest structural maintenance risk.
- Main hotspots remain:
  - `cmd/picobot/main.go`
  - `internal/agent/loop.go`
  - `internal/agent/tools/taskstate.go`
  - treasury-heavy `internal/missioncontrol/*` files

### 3. Duplicated runtime, treasury, capability, and test scaffolding

- Still open.
- The diagnosis still points to repeated helper families and very large tests as a chronic regression amplifier.

### 4. Docs and surface-truth drift

- Still open.
- The repo still carries mixed Picobot-generic language, Frank V3 policy surfaces, and V4 intent documents without one short canonical routing layer.
- `spawn` surface truth drift is still unresolved in the Round 2 backlog.

### 5. Remaining attack-surface and stringly-typed design debt

- Still open.
- `exec`, `filesystem`, `web`, channel ingress, Zoho/provider boundaries, and broad `map[string]interface{}` payloads remain structurally wide surfaces.

### 6. V4 readiness gap

- Still open by design.
- The diagnosis still says the V4-specific substrate is missing:
  - isolated improvement workspace
  - explicit mutable-target policy
  - pack/candidate registry
  - hot-update gate
  - rollback ledger and last-known-good pointer

### 7. Residual sub-risks inside completed lanes

- 001 residual:
  - cross-process long-memory coordination is still open
  - long-memory idempotence is still tail-based rather than operation-ID-based
- 002 residual:
  - unauthorized-access logs still include identity details
  - runtime validation evidence still stores full successful tool arguments and results

## What Still Looks AI-Sloppy

- Giant files are still acting as catch-all authority sinks instead of clean boundaries.
- Giant tests are still carrying broad fixture/setup burdens instead of smaller command-family or subsystem seams.
- Too much behavior is still routed through generic payload maps and stringly-typed interfaces.
- The repo still presents more than one truth surface about what it currently is.
- Some cleanup candidates are still obvious “agent left it half-generic” symptoms:
  - `spawn` truth drift
  - mixed Picobot/Frank language
  - broad config/docs surfaces that are not clearly routed
- The codebase is healthier, but it is not yet aesthetically or structurally “clean senior-human deliberate” in the big runtime and CLI surfaces.

## What Is Now Materially Healthier

- The repo is synchronized with upstream `main` and green on the canonical branch.
- Session and memory durability semantics are substantially stronger than before Round 2 treatment work.
- The logging story is materially less reckless:
  - raw provider body exposure is reduced
  - raw MCP/tool failure payload exposure is reduced
  - raw inbound message-fragment logging is reduced
  - raw tool-argument semantics are reduced in treated paths
- Frank V3 protected surfaces survived upstream integration without being casually weakened.
- The campaign now has a credible evidence trail with before/after artifacts instead of vague “cleanup happened” claims.

## Proposed “AI-Slop-Free Enough” Exit Gate

This is not “the repo is clean.” It is the narrower gate for “safe enough to stop Garbage Campaign and move to V4 planning without carrying obvious unresolved slop from Round 2.”

1. Canonical branch is green and `0 behind upstream/main`.
2. 009, 001A-001D, 002A, 002B, and 002C are complete.
3. No known raw provider/MCP/tool/channel-content leakage remains on the default operator/onboarding paths covered by Round 2.
4. Secret prompts are non-echoing and docs use unmistakable placeholders rather than token-shaped examples.
5. There is one short operator-facing routing note that states current canonical runtime truth and points to the right docs.
6. Known surface-truth mismatch like `spawn` is either removed from public/operator docs or intentionally implemented.

If those six conditions are met, the repo can reasonably be called “AI-slop-free enough” for a pause, even though deeper structural cleanup would still remain available.

## Options From Here

### 1. Do 002C next

- What it means:
  - finish the remaining secret/onboarding/docs hygiene slice
  - then reassess against the exit gate above
- Benefits:
  - closes the last major open item inside the 002 log/onboarding disease cluster
  - gives a cleaner stopping point before V4 planning
  - is still a bounded slice rather than a sprawling refactor
- Costs:
  - touches `cmd/picobot/main.go` and public docs again
  - small but real risk of CLI/docs churn

### 2. Stop Garbage Campaign and start V4 planning now

- What it means:
  - freeze cleanup here
  - open a planning/design branch for V4 substrate work without more repo-health treatment first
- Benefits:
  - preserves momentum if the real bottleneck is V4 definition rather than more cleanup
  - avoids turning Garbage Day into an endless beautification loop
- Costs:
  - accepts that secret-safe onboarding/docs hygiene is still not done
  - leaves current docs/surface-truth slop in place while starting a new planning layer

### 3. Continue a deeper structural cleanup campaign

- What it means:
  - go beyond 002C into bigger Round 2 structural lanes such as docs reconciliation, spawn truth cleanup, CLI decomposition, test splitting, TaskState helper extraction, or treasury family extraction
- Benefits:
  - directly attacks the biggest chronic AI-sloppy symptoms
  - could leave the repo genuinely cleaner before any V4 scaffold work
- Costs:
  - much higher time and regression budget
  - higher collision risk with protected V3 surfaces
  - easy to accidentally turn maintenance into open-ended redesign

## Recommendation

- Recommended path: Option 1, do `002C` next, then stop Garbage Campaign if the exit gate is met.

### Rationale

- 002C is the smallest remaining high-signal cleanup slice that still belongs to the disease cluster already in motion.
- Stopping now is defensible, but it would leave one obvious “we cleaned the logs but still echo secrets and normalize plaintext tokens” wart on the table.
- Going deeper than 002C before a fresh human decision is not the best trade right now because the remaining large wins are structural, slower, and much closer to protected V3 semantics.
- In short:
  - `002C` is the clean ending
  - stopping now is acceptable but slightly premature
  - deeper structural cleanup should be a separate consciously chosen campaign, not an automatic continuation
