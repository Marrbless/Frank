# GC2-TREAT-006C Runtime-Truth Routing Before

Date: 2026-04-20

## Live checkpoint

- Branch: `frank-v3-foundation`
- HEAD: `34b19768d4ece5ddc2c578d0ba5a84c7f04b2371`
- Tags at HEAD:
  - `frank-garbage-campaign-006b-clean`
- Ahead/behind `upstream/main`: `375 ahead / 0 behind`
- `git status --short --branch`:
  - `## frank-v3-foundation`
- Baseline `go test -count=1 ./...` result:
  - passed

## Exact stale authority/workflow truth surfaces found

- `docs/FRANK_DEV_WORKFLOW.md`
  - presents desktop as current authoritative Frank development surface
  - presents `mission-control-v1` as current canonical Frank branch
  - presents desktop-first promotion flow as current operating truth, not historical workflow guidance
- `README.md`
  - links to `FRANK_DEV_WORKFLOW.md` in the docs list
  - does not provide a clear “current canonical runtime/repo truth starts here” routing note
- `docs/HOW_TO_START.md`
  - accurately documents current runtime/operator behavior, but does not clearly route repo work or future cleanup work to a canonical truth note

## Planned files

- `docs/CANONICAL_RUNTIME_TRUTH.md`
  - new short routing note for current implementation/runtime/repo truth
- `docs/FRANK_DEV_WORKFLOW.md`
  - add a top-level note that it is not the current canonical runtime-truth document and route readers to the new note
- `README.md`
  - add a minimal docs link to the canonical runtime-truth note
- `docs/HOW_TO_START.md`
  - add a minimal routing link near the Frank mission-controlled gateway/operator surface
- `docs/maintenance/garbage-day/GC2_TREAT_006C_RUNTIME_TRUTH_ROUTING_AFTER.md`

## Non-goals

- Do not change code
- Do not change runtime behavior
- Do not implement V4
- Do not perform broad docs cleanup
- Do not rewrite the entire workflow document
- Do not reconcile every historical Frank spec in this slice
- Do not add dependencies
- Do not commit
