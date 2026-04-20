# GC2-TREAT-006C Runtime-Truth Routing After

Date: 2026-04-20

## Diff summary

### `git diff --stat`

```text
 README.md                  | 5 +++--
 docs/FRANK_DEV_WORKFLOW.md | 2 ++
 docs/HOW_TO_START.md       | 2 ++
 3 files changed, 7 insertions(+), 2 deletions(-)
```

### `git diff --numstat`

```text
3	2	README.md
2	0	docs/FRANK_DEV_WORKFLOW.md
2	0	docs/HOW_TO_START.md
```

Note: the new file `docs/CANONICAL_RUNTIME_TRUTH.md` is untracked, so it does not appear in `git diff --stat` yet.

## Files changed

- `README.md`
- `docs/FRANK_DEV_WORKFLOW.md`
- `docs/HOW_TO_START.md`
- `docs/CANONICAL_RUNTIME_TRUTH.md`
- `docs/maintenance/garbage-day/GC2_TREAT_006C_RUNTIME_TRUTH_ROUTING_BEFORE.md`

## Exact routing corrections made

- Added new canonical routing note:
  - `docs/CANONICAL_RUNTIME_TRUTH.md`
  - states that live repo branch and live shell state win
  - states that `frank-v3-foundation` is the current canonical runtime/control truth
  - distinguishes current implementation truth from future V4 target truth
  - gives explicit “start here” routing for repo work, runtime verification, and cleanup
- Added a top-level historical-note banner to `docs/FRANK_DEV_WORKFLOW.md`
  - stops it from masquerading as current canonical runtime-truth
  - routes readers to `docs/CANONICAL_RUNTIME_TRUTH.md`
- Added minimal docs routing in `README.md`
  - added `CANONICAL_RUNTIME_TRUTH.md` to the docs list
- Added minimal operator/repo routing in `docs/HOW_TO_START.md`
  - points repo work and truth checks to `docs/CANONICAL_RUNTIME_TRUTH.md`

## Runtime semantics

- No code changed.
- No runtime behavior changed.
- No V4 implementation work was done.

## Validation commands and results

- `git diff --check`
  - passed
- `go test -count=1 ./...`
  - passed
- `git status --short --branch`
  - after validation and before writing this after note:
    - `## frank-v3-foundation`
    - ` M README.md`
    - ` M docs/FRANK_DEV_WORKFLOW.md`
    - ` M docs/HOW_TO_START.md`
    - `?? docs/CANONICAL_RUNTIME_TRUTH.md`
    - `?? docs/maintenance/garbage-day/GC2_TREAT_006C_RUNTIME_TRUTH_ROUTING_BEFORE.md`
