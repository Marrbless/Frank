# V4-149 Package Donor Import After

Branch: `frank-v4-149-package-donor-import`

## Requirement Rows

- `AC-027` moved from `PARTIAL` to `DONE`.
- `AC-032` moved from `MISSING` to `DONE`.
- `SF-006` moved from `MISSING` to `DONE`.

## Implemented

- Added `PackageImportRecord` storage under `runtime_packs/package_imports`.
- Package imports must link to an existing improvement candidate and its candidate runtime pack.
- Package imports require source package ref, local content ref, SHA-256 content identity, content kinds, declared surfaces, provenance, source summary, and writer metadata.
- Package imports must remain `activation_state=candidate_only`.
- Package imports that declare provider, spending, owner-control, campaign, or any other authority grant are rejected with `E_PACKAGE_AUTHORITY_GRANT_FORBIDDEN` before a record is written.
- Store/load/list paths are append-only/idempotent and reject divergent duplicates.

## Validation

- Focused package test: `/usr/local/go/bin/go test -count=1 ./internal/missioncontrol`
- Required full validation: `git diff --check` and `/usr/local/go/bin/go test -count=1 ./...`

No network call, external service call, real donor fetch, active package activation, or approval authority was added.
