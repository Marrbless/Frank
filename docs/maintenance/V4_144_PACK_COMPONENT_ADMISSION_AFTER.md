# V4-144 Pack Component Admission After

Branch: `frank-v4-144-pack-component-admission`

## Requirement Rows

- `AC-021` moved from `PARTIAL` to `DONE`.
- `AC-029` remains `PARTIAL` because raw component/package import paths are not yet gate-bound.
- `AC-024` remains `PARTIAL` because policy-surface content scanners are still later work.

## Implemented

- Extended runtime pack component metadata with optional `surface_class`, `declared_surfaces`, and `hot_reloadable`.
- Added `AssessRuntimePackComponentAdmissionForHotUpdate`.
- Added `RequireRuntimePackComponentAdmissionForHotUpdate`.
- Admission now blocks:
  - missing hot-update gate context,
  - missing surface class,
  - surface class not declared by candidate pack or gate,
  - missing declared surfaces,
  - declared surfaces outside candidate mutable surfaces,
  - declared surfaces outside gate target surfaces,
  - declared surfaces listed as candidate immutable surfaces,
  - non-hot-reloadable component metadata.

## Validation

- Focused package test: `/usr/local/go/bin/go test -count=1 ./internal/missioncontrol`
- Required full validation: `git diff --check` and `/usr/local/go/bin/go test -count=1 ./...`

No content activation, network call, external service call, or extension permission widening approval was added.
