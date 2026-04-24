# V4-065 Execution Plane Skeleton Before

## Branch

- `frank-v4-065-execution-plane-skeleton`

## HEAD

- `0cf26b75ffe240bc18f7426e6e14170e13075b33`

## Tags At HEAD

- `frank-v4-064-v4-spec-gap-assessment`

## Baseline

- Worktree was clean before edits.
- Baseline `/usr/local/go/bin/go test -count=1 ./...` passed.

## Before-State Gap From V4-064

- `docs/maintenance/V4_064_FRANK_V4_SPEC_GAP_ASSESSMENT.md` identified the next safe slice as a job/proposal-level execution-plane skeleton.
- `internal/missioncontrol.Job` did not carry `execution_plane`, `execution_host`, or `mission_family`.
- `ValidatePlan` had no V4 job-level requirement for those fields and no plane/family compatibility guard.
- Existing inspect/status read models exposed job ID, state, authority, allowed tools, and step data, but not V4 execution-plane metadata.
- Similar fields existed on `ImprovementRunRecord`, but that was too late to enforce job admission.

## Planned Scope

- Add job-level `execution_plane`, `execution_host`, and `mission_family` fields.
- Add a V4-only validation boundary so pre-V4 jobs remain backward compatible.
- Add fail-closed validation for known execution planes, known execution hosts, known mission families, and family/plane compatibility.
- Expose the fields through existing inspect/status/read-model surfaces.
- Preserve JSON/storage round-trip behavior without rewriting old records that omit the fields.

## Explicit Non-Goals

- No adaptive lab implementation.
- No improvement mutation.
- No runtime-pack pointer mutation.
- No hot-update behavior change.
- No new commands.
- No TaskState wrappers.
- No rejection-code framework beyond the errors needed by this slice.
- No V4-066 work.
