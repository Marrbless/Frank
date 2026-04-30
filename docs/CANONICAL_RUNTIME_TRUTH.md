# Canonical Runtime Truth

If you are doing repo work, runtime cleanup, operator-surface review, or future Frank planning, start here first.

## Current canonical truth

- Live repo branch and live shell state win over older handoffs, pasted summaries, and stale docs.
- The current canonical Frank runtime/control truth is the code and docs on branch `main`.
- The durable implemented runtime today is the Picobot gateway plus Frank mission-control surface:
  - CLI-driven startup and operator control
  - durable mission store
  - status snapshot and step-control files
  - hot-update, rollback, runtime-pack, canary, approval, and LKG records
  - optional channels, MCP, memory, skills, and scheduler support

## What is current implementation truth

Treat these as the current operator/runtime truth on this branch:

- [../START_HERE_OPERATOR.md](../START_HERE_OPERATOR.md)
- [HOW_TO_START.md](./HOW_TO_START.md)
- [CONFIG.md](./CONFIG.md)
- [HOT_UPDATE_OPERATOR_RUNBOOK.md](./HOT_UPDATE_OPERATOR_RUNBOOK.md)
- [ANDROID_PHONE_DEPLOYMENT.md](./ANDROID_PHONE_DEPLOYMENT.md)
- current code under `cmd/picobot/` and `internal/missioncontrol/`
- current code under `internal/agent/` and `internal/agent/tools/`

## What is not current implementation truth

- Older branch-specific maintenance artifacts are historical evidence unless the live repo state confirms they still describe current behavior.
- [FRANK_DEV_WORKFLOW.md](./FRANK_DEV_WORKFLOW.md) is a historical workflow doc for an older desktop-centric handoff model.
- Spec language in [FRANK_V4_SPEC.md](./FRANK_V4_SPEC.md) is current only where implemented by live code and validated by current tests/runbooks.

## Deployment target vs implementation truth

- Current implementation truth is host-neutral Go runtime behavior on `main` plus documented phone deployment paths.
- Phone deployment docs describe the intended private runtime target, but real-device evidence must still be recorded for phone-only operational claims.
- Do not treat historical phone-only target language as proof that a specific device has been validated after a new change.

## Older workflow docs

- [FRANK_DEV_WORKFLOW.md](./FRANK_DEV_WORKFLOW.md) is historical workflow guidance for an older desktop-centric handoff model.
- Keep it for provenance, but do not treat it as the current canonical runtime-truth document.

## Practical routing

- If you are operating or verifying the current runtime, start with [HOW_TO_START.md](./HOW_TO_START.md) and [CONFIG.md](./CONFIG.md).
- If you are doing hot-update operations, use [HOT_UPDATE_OPERATOR_RUNBOOK.md](./HOT_UPDATE_OPERATOR_RUNBOOK.md).
- If you are doing repo cleanup or structural work, use the live branch state on `main` plus [maintenance/CURRENT.md](./maintenance/CURRENT.md).
- If a doc conflicts with the live repo branch and current shell evidence, the live repo branch and current shell evidence win.
