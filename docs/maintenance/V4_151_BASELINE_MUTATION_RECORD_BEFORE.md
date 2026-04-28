# V4-151 Baseline Mutation Record Before

Branch: `frank-v4-151-baseline-mutation-record`

## Requirement Rows

- `AC-007` was `PARTIAL`.

## Observed Gap

- Candidate/result and local eval records carried baseline and candidate pack refs.
- The deterministic eval runner bound results to the improvement run and frozen eval suite.
- There was no durable candidate mutation record proving that baseline evidence was captured before mutation started.

## Intended Slice

- Add an append-only candidate mutation record linked to improvement run, candidate, eval suite, baseline pack, and candidate pack.
- Require a durable baseline result ref and reject mutation records whose start time precedes baseline capture time.
- Keep the slice local and deterministic with no AI calls, network calls, external services, active pack mutation, or device side effects.
