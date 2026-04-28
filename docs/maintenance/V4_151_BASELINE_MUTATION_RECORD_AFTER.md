# V4-151 Baseline Mutation Record After

Branch: `frank-v4-151-baseline-mutation-record`

## Requirement Rows

- `AC-007` moved from `PARTIAL` to `DONE`.

## Implemented

- Added `CandidateMutationRecord` storage under `runtime_packs/candidate_mutations`.
- Candidate mutation records link to improvement run, improvement candidate, eval suite, baseline pack, candidate pack, and a baseline result ref.
- The baseline result ref must be present in the linked candidate's validation basis refs.
- Mutation start time must not precede baseline capture time, and completion time must not precede mutation start time.
- Store/load/list paths are append-only/idempotent and reject divergent duplicate mutation records.

## Validation

- Focused package test: `/usr/local/go/bin/go test -count=1 ./internal/missioncontrol`
- Required full validation: `git diff --check` and `/usr/local/go/bin/go test -count=1 ./...`

No AI call, network call, external service call, active pack mutation, or device side effect was added.
