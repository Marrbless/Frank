# V4-066 Rejection Code Skeleton Before

## Branch

- Started on `frank-v4-065-execution-plane-skeleton`.
- Created and switched to `frank-v4-066-v4-rejection-code-skeleton` before edits.

## HEAD

- `3cafde8fbe5f7cf7cb5afa151c100f0e28467371`

## Tags At HEAD

- `frank-v4-065-execution-plane-skeleton`

## Baseline

- Worktree was clean before edits.
- Baseline `/usr/local/go/bin/go test -count=1 ./...` passed.

## Before-State Gap From V4-064 / V4-065

- `docs/maintenance/V4_064_FRANK_V4_SPEC_GAP_ASSESSMENT.md` identified V4 machine-readable `E_*` rejection codes as missing and said they should follow concrete execution-plane validators.
- V4-065 added `execution_plane`, `execution_host`, and `mission_family` job metadata plus V4-only validation.
- V4-065 validation still emitted repo-local internal codes:
  - `invalid_execution_plane`
  - `invalid_execution_host`
  - `invalid_mission_family`
- The frozen spec requires explicit auditable V4 rejection codes for improvement, hot-update, promotion, rollback, canary, runtime-source, policy, autonomy, and package-authority failure classes.

## Planned Scope

- Add a repo-consistent skeleton of V4 `RejectionCode` constants with exact `E_*` string values.
- Map only the existing V4-065 execution-plane/host/family validators to deterministic V4 codes.
- Preserve existing pre-V4 validation behavior and messages.

## Explicit Non-Goals

- No adaptive lab behavior.
- No improvement mutation.
- No runtime-pack pointer mutation.
- No hot-update behavior change.
- No new commands.
- No TaskState wrappers.
- No topology, canary, deploy-lock, source-patch, promotion-policy, prompt-pack registry, or skill-pack registry enforcement.
- No V4-067 work.
