# V4-159 Active Pack Mutation Guard After

Branch: `frank-v4-159-active-pack-mutation-guard`

## Requirement Rows

- `AC-029` moved from `PARTIAL` to `DONE`.

## Implemented

- Added a shared active-pack ad hoc mutation guard.
- Raw runtime component storage now rejects new component records referenced by the committed active pack.
- Candidate component records not referenced by the active pack remain allowed for staged hot-update content.
- Package imports now reject candidate-only imports when `candidate_pack_id` is the current active pack.
- Both rejection paths use `E_ACTIVE_PACK_ADHOC_MUTATION_FORBIDDEN`.

## Validation

- Focused package test: `/usr/local/go/bin/go test -count=1 ./internal/missioncontrol`
- Required full validation: `git diff --check` and `/usr/local/go/bin/go test -count=1 ./...`

No direct active pointer write path, active pack edit path, external side effect, network call, or device action was added.
