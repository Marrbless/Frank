# Garbage Day

Garbage Day is repo maintenance and health work. It is not Frank V4 implementation work, and it is not authorization to change behavior just because a report found a problem.

## Current status

- Phase 1 cleanup passes: completed earlier and reconciled here into a durable before/after surface.
- Round 2 repo diagnosis: completed in this directory as a repo-wide patient chart.
- V4 completion: recorded at `frank-v4-full-spec-complete`.
- Post-V4 kickoff: started in [POST_V4_GARBAGE_DAY_KICKOFF.md](./POST_V4_GARBAGE_DAY_KICKOFF.md).
- Current controller: [POST_V4_GARBAGE_DAY_MATRIX.md](./POST_V4_GARBAGE_DAY_MATRIX.md).
- Treatment: authorized only as bounded, non-destructive maintenance slices from the current controller.

## Documents

- Phase 1 before/after: [PHASE_1_BEFORE_AFTER.md](./PHASE_1_BEFORE_AFTER.md)
- Phase 1 artifact index: [PHASE_1_ARTIFACT_INDEX.md](./PHASE_1_ARTIFACT_INDEX.md)
- Round 2 repo diagnosis: [ROUND_2_REPO_DIAGNOSIS.md](./ROUND_2_REPO_DIAGNOSIS.md)
- Post-V4 kickoff: [POST_V4_GARBAGE_DAY_KICKOFF.md](./POST_V4_GARBAGE_DAY_KICKOFF.md)
- Post-V4 matrix: [POST_V4_GARBAGE_DAY_MATRIX.md](./POST_V4_GARBAGE_DAY_MATRIX.md)

## Guardrails

- Garbage Day is maintenance, cleanup planning, and repo-health assessment work.
- Garbage Day is not Frank V4 behavior work.
- Garbage Day reports are evidence, not automatic authorization to fix.
- Protected V3 surfaces still require explicit treatment selection, bounded slices, and validation before any code change.

## Evidence basis

- Live repo state on `2026-04-19` from `pwd`, `git rev-parse HEAD`, `git status --short --branch`, `git tag --points-at HEAD`, `git remote -v`, `git branch --show-current`, `git branch -vv`, `git log --oneline --decorate -20`, `go version`, `go env`, `go list ./...`, and `go test -count=1 ./...`.
- Raw Phase 1 reports in `docs/maintenance/GARBAGE_DAY_*.md`, indexed in [PHASE_1_ARTIFACT_INDEX.md](./PHASE_1_ARTIFACT_INDEX.md).
