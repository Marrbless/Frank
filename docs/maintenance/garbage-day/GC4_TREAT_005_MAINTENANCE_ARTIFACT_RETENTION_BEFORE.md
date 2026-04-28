# GC4-TREAT-005 Maintenance Artifact Retention Before

Branch: `frank-garbage-day-gc4-005-maintenance-artifact-retention`

## Target

`GC4-005`: distinguish retained Garbage Day and maintenance evidence from pruneable scratch without deleting files.

## Starting Evidence

- `docs/maintenance` contains `287` `V4_*.md` files.
- `docs/maintenance` contains `302` top-level files.
- `docs/maintenance/garbage-day` contains `80` files.
- `POST_V4_GARBAGE_DAY_MATRIX.md` has one remaining `MISSING` row: `GC4-005`.
- `PHASE_1_ARTIFACT_INDEX.md` already classifies older raw Phase 1 Garbage Day reports as retained audit detail or later archive/delete candidates.
- `POST_V4_GARBAGE_DAY_KICKOFF.md` explicitly says not to delete V4 evidence or maintenance history.

## Slice Constraint

Add only documentation that defines retention/prune rules. Do not delete, move, archive, or rewrite existing evidence artifacts in this slice.
