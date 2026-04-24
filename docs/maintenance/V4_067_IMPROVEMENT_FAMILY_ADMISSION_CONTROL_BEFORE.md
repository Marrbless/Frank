# V4-067 Improvement Family Admission Control Before

## Branch

- Started on `frank-v4-066-v4-rejection-code-skeleton`.
- Created and switched to `frank-v4-067-improvement-family-admission-control` before edits.

## HEAD

- `731099579f1e867474cbc1c3a6873563a6d43ac4`

## Tags At HEAD

- `frank-v4-066-v4-rejection-code-skeleton`

## Baseline

- Worktree was clean before edits.
- Baseline `/usr/local/go/bin/go test -count=1 ./...` passed.

## Before-State Gap From V4-064 / V4-065 / V4-066

- V4-064 identified job/proposal-level execution metadata and improvement-family admission as prerequisites for controlled self-improvement.
- V4-065 added job-level `execution_plane`, `execution_host`, and `mission_family`, plus broad family-to-plane validation.
- V4-066 mapped those validators to deterministic V4 rejection codes.
- The live validation path still lacked explicit helper names/tests for improvement-family admission and did not check whether an improvement-family job's host was compatible with `improvement_workspace`.

## Planned Scope

- Add a named helper that identifies V4 improvement mission families.
- Add a named helper for improvement-workspace-compatible hosts.
- Admit improvement-family V4 jobs only when:
  - `execution_plane=improvement_workspace`
  - `execution_host` is compatible with the improvement workspace
- Keep pre-V4 behavior backward compatible.

## Explicit Non-Goals

- No adaptive lab.
- No mutation.
- No eval runs.
- No prompt-pack or skill-pack registry.
- No topology gate beyond admission metadata checks.
- No source-patch artifact policy beyond admission metadata checks.
- No promotion policy.
- No canary policy.
- No deploy lock.
- No new commands.
- No TaskState wrappers.
- No V4-068 work.
