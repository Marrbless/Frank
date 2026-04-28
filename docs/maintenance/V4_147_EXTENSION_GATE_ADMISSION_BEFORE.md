# V4-147 Extension Gate Admission Before

Branch: `frank-v4-147-extension-gate-admission`

## Matrix Rows

- `AC-031` is `PARTIAL`: extension widening is assessed in candidate-result promotion eligibility/status reads, but direct hot-update gate creation can still be requested from a candidate pack.
- `AC-027` is `PARTIAL`: live external guardrails surface candidate-result blockers, but gate admission still needs direct extension blocker enforcement.

## Intended Slice

Wire extension permission widening assessment into hot-update gate admission for direct candidate hot updates. A candidate pack whose extension ref differs from the previous active pack must fail closed when the extension manifest widens permissions, adds external side-effect tools, changes compatibility contracts, or is missing.

## Non-Goals

- Do not invent approval authority for permission widening.
- Do not implement real plugin hot reload.
- Do not add network/device side effects.
