# V4-149 Package Donor Import Before

Branch: `frank-v4-149-package-donor-import`

## Matrix Rows

- `AC-032` is `MISSING`: package authority rejection code exists, but no donor/package import path exists.
- `SF-006` is `MISSING`: package/donor imports do not yet preserve provenance as candidate-only content.
- `AC-027` is `PARTIAL`: package/donor authority checks remain the missing external-guardrail piece after extension gate hardening.

## Intended Slice

Add a local deterministic package import registry that records donor/package content as candidate-only content with provenance, content identity, candidate linkage, shape validation, and fail-closed authority grant rejection.

## Non-Goals

- Do not activate imported package content.
- Do not add network, external service, real donor fetch, or device side effects.
- Do not invent approval authority for package-declared authority grants.
