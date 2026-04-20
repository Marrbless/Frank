# Canonical Runtime Truth

If you are doing repo work, runtime cleanup, operator-surface review, or future Frank planning, start here first.

## Current canonical truth

- Live repo branch and live shell state win over older handoffs, pasted summaries, and stale docs.
- The current canonical Frank runtime/control truth is the code and docs on branch `frank-v3-foundation`.
- The durable implemented runtime today is the Frank V3-style Picobot gateway plus mission-control surface:
  - CLI-driven startup and operator control
  - durable mission store
  - status snapshot and step-control files
  - optional channels, MCP, memory, skills, and scheduler support

## What is current implementation truth

Treat these as the current operator/runtime truth on this branch:

- [HOW_TO_START.md](./HOW_TO_START.md)
- [CONFIG.md](./CONFIG.md)
- current code under `cmd/picobot/` and `internal/missioncontrol/`

## What is not current implementation truth

- [FRANK_V4_SPEC.md](./FRANK_V4_SPEC.md) is a future-target spec.
- It describes intended V4 deployment and authority boundaries, not the currently implemented runtime substrate on `frank-v3-foundation`.
- In particular, phone-resident improvement workspace, candidate packs, hot-update gate, and rollback-pack runtime truth are not implemented merely because they are specified there.

## Deployment target vs implementation truth

- Current implementation truth is host-neutral Go runtime behavior on this branch.
- Future deployment target truth may be narrower.
- Do not treat future phone-only target language as proof that the current repo already implements phone-resident V4 runtime surfaces.

## Older workflow docs

- [FRANK_DEV_WORKFLOW.md](./FRANK_DEV_WORKFLOW.md) is historical workflow guidance for an older desktop-centric handoff model.
- Keep it for provenance, but do not treat it as the current canonical runtime-truth document.

## Practical routing

- If you are operating or verifying the current runtime, start with [HOW_TO_START.md](./HOW_TO_START.md) and [CONFIG.md](./CONFIG.md).
- If you are doing repo cleanup or structural work, use the live branch state on `frank-v3-foundation` plus the current maintenance notes under `docs/maintenance/garbage-day/`.
- If a doc conflicts with the live repo branch and current shell evidence, the live repo branch and current shell evidence win.
