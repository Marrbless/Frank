# V4-157 Policy Surface Scanner Before

Branch: `frank-v4-157-policy-surface-scanner`

## Requirement Rows

- `AC-024` was `PARTIAL`.

## Observed Gap

- `E_POLICY_MUTATION_FORBIDDEN` existed as a rejection code.
- Generic immutable-surface checks existed for job and component admission.
- Package imports rejected declared authority grants.
- Candidate package/import and extension declarations did not share a deterministic scanner for frozen authority, approval, autonomy, treasury, or campaign policy surfaces.

## Intended Slice

- Add a small shared frozen-policy-surface declaration scanner.
- Reject package import declared surfaces that target frozen policy surfaces.
- Reject extension pack declarations that target frozen policy surfaces through permissions, tool permission refs, or external side effects.
- Reject hot-update component admission when declared surfaces target frozen policy surfaces.
- Preserve candidate-only package/import behavior and existing hot-update gate flow.
