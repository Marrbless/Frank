# Garbage Campaign Phase 1 Complete

Date: 2026-04-20

## Live State

- Canonical repo: `/mnt/d/pbot/picobot`
- Current branch: `frank-v3-foundation`
- Current HEAD: `f81567cbe7224de2b908449aa142bb54884eaebc`
- Tags at HEAD:
  - `frank-garbage-campaign-002c-clean`
- Ahead/behind `upstream/main`: `371 ahead / 0 behind`
- Repo green at this checkpoint: yes
  - Evidence: `go test -count=1 ./...` passed at this `HEAD`

## Completed Lanes

### GD2-TREAT-009

- Status: complete
- Major risks removed:
  - removed the repo-behind-upstream state for `upstream/main`
  - removed unresolved integration overlap risk in `cmd/picobot/main.go`, `internal/agent/loop.go`, and `docs/HOW_TO_START.md`
  - landed upstream MCP support and tool-activity gating without weakening Frank mission-control, approval, treasury, capability, or owner-control semantics
  - removed ambiguity about whether upstream MCP/JSON changes could coexist with current Frank V3 protected surfaces

### GD2-TREAT-001A through 001D

- Status: complete
- Major risks removed:
  - removed session-path traversal and filename-collision risk from external `channel:chatID` values
  - removed the startup rehydration gap where session files existed on disk but were not loaded
  - removed fail-open startup behavior for malformed session files
  - removed the crash window where session and memory overwrites could leave truncated or zero-length files
  - removed the false-success remember path that claimed persistence succeeded when it had failed
  - removed undurable append behavior that wrote without explicit file sync and ignored close failure
  - removed the in-process long-memory read-modify-write race that could lose concurrent appends inside one `MemoryStore`
  - removed immediate retry duplication at the file tail for the same append request

### GD2-TREAT-002A through 002C

- Status: complete
- Major risks removed:
  - removed raw non-2xx provider body logging
  - removed surfaced provider and MCP error payloads that could include prompt fragments, secrets, provider account data, or remote error bodies
  - removed raw tool failure notifications and model-visible tool error payloads in the treated paths
  - removed raw `ProcessDirect` provider error exposure
  - removed raw tool-argument key exposure from registry logs and tool-activity notifications
  - removed inbound Slack, Discord, and WhatsApp raw content fragments from logs
  - removed Slack and Discord attachment-URL exposure from inbound logs
  - replaced token-shaped generated/default/examples with unmistakable placeholders in onboarding/config/docs
  - added secret-safe token entry in interactive Telegram, Discord, and Slack login flows on supported terminals
  - added explicit documentation that config stores credentials in plaintext and that live secrets should not be pasted into logs, screenshots, issue reports, or chat transcripts

## Phase 1 Closure

- The hygiene, logging, onboarding, and durability cluster targeted by `009`, `001A–001D`, and `002A–002C` is closed.
- This does not mean the repo is structurally clean.
- It means the bounded safety/hygiene cluster selected for this campaign phase has been completed and validated on the canonical branch.

## Phase 2 Opening

- The next Garbage Campaign phase is structural anti-slop cleanup.
- The focus shifts away from the closed hygiene/safety cluster and toward the chronic structural disease clusters already identified in Round 2:
  - overgrown protected-surface files
  - duplicated runtime/treasury/capability/test scaffolding
  - stringly typed and catch-all interface surfaces
  - docs and surface-truth drift

## V4 Status

- Frank V4 remains postponed.
- V4 planning or substrate work should stay deferred pending explicit structural campaign decisions about what to clean next and how far to take the anti-slop phase before opening a V4 branch.
