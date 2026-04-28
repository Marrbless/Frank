# Maintenance Artifact Retention Policy

This policy distinguishes durable maintenance evidence from pruneable scratch. It is a cleanup rule for `docs/maintenance` and `docs/maintenance/garbage-day`; it does not authorize deletion by itself.

## Retain By Default

Keep these artifacts tracked and reachable:

- spec completion matrices, final completion notes, and checkpoint decisions
- before/after treatment notes tied to code, test, or docs slices
- validation evidence that records exact commands, package scope, and results
- diagnosis reports that contain source references, counts, risks, or treatment decisions
- artifact indexes that preserve provenance for older raw reports
- controller documents that define active or historical cleanup scope

Retained artifacts may be consolidated or moved only when a replacement index names the original paths, explains what evidence was carried forward, and records the validation used for the move.

## Pruneable Scratch

An artifact is pruneable only when all of these are true:

- it is local scratch, generated output, or a convenience rollup rather than source evidence
- it is untracked, or a tracked replacement explicitly preserves every unique fact needed for audit
- `rg` finds no live controller, matrix, README, spec, or treatment note depending on the path
- it is not the only place recording validation commands, branch/head facts, or risk decisions
- the pruning slice names the exact files and receives explicit human approval when deletion is involved

Deletion must not be bundled with behavior changes, code refactors, schema changes, or safety-policy changes.

## Archive Instead Of Delete

Prefer an archive move over deletion when a tracked artifact has historical value but is no longer part of the primary reading path. An archive move requires:

- a destination under the same maintenance area, such as `docs/maintenance/garbage-day/archive-*`
- an index update that maps old paths to new paths
- link/reference updates for any primary docs that still point at the moved files
- `git diff --check`

Do not archive V4 completion evidence until a dedicated V4 evidence index exists and the move is approved as its own slice.

## Current Classification

| Path family | Classification | Rationale | Current action |
| --- | --- | --- | --- |
| `docs/maintenance/V4_*.md` | retained evidence | V4 completion history, slice receipts, validation commands, and audit checkpoints | keep in place |
| `docs/maintenance/V4_FULL_SPEC_COMPLETION_*.md` | retained controller/final evidence | records full-spec closure and matrix state | keep in place |
| `docs/maintenance/GARBAGE_DAY_*.md` | retained raw Phase 1 evidence | indexed by `PHASE_1_ARTIFACT_INDEX.md`; some facts are compressed elsewhere but raw details remain useful | keep in place; archive only in a later approved slice |
| `docs/maintenance/garbage-day/*.md` | retained controller and treatment evidence | active Garbage Day route, matrices, diagnosis, and before/after receipts | keep in place |
| future generated logs, temporary exports, and convenience rollups | candidate scratch | may duplicate retained evidence or be local-only output | evaluate with the prune gate before any deletion |

## Prune Gate

Before deleting or moving a maintenance artifact:

1. List the exact paths and their tracked/untracked status.
2. Search references with `rg` and record the result.
3. Identify the retained replacement, index, or reason no replacement is needed.
4. Confirm the artifact is not unique validation, branch/head, risk, or decision evidence.
5. Keep deletion separate from implementation or behavior work.
6. Run `git diff --check`.
7. Record the decision in a before/after note.

If any step is uncertain, retain the artifact.
