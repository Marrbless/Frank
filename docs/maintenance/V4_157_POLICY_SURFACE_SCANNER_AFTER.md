# V4-157 Policy Surface Scanner After

Branch: `frank-v4-157-policy-surface-scanner`

## Requirement Rows

- `AC-024` moved from `PARTIAL` to `DONE`.

## Implemented

- Added a shared deterministic frozen-policy-surface scanner for authority, approval, autonomy, treasury, and campaign policy declarations.
- Package imports now reject declared surfaces that target frozen policy surfaces with `E_POLICY_MUTATION_FORBIDDEN`.
- Runtime extension pack validation now rejects policy-surface declarations in permissions, tool permission refs, and external side effects.
- Hot-update component admission now rejects policy-surface declared surfaces before generic mutable/immutable-surface checks.

## Validation

- Focused package test: `/usr/local/go/bin/go test -count=1 ./internal/missioncontrol`
- Required full validation: `git diff --check` and `/usr/local/go/bin/go test -count=1 ./...`

No activation path, authority grant, external service call, network call, device action, or policy mutation lane was added.
