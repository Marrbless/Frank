# V4-158 Canary Scope Truth After

Branch: `frank-v4-158-canary-scope-truth`

## Requirement Rows

- `AC-025` moved from `PARTIAL` to `DONE`.

## Implemented

- Canary requirements now store deterministic `canary_scope_job_refs` and `canary_scope_surfaces`.
- Canary evidence now stores `evidence_source`, `automatic_traffic_exercised`, `exercised_job_refs`, and `exercised_surfaces`.
- Existing local canary evidence creation records `operator_recorded` evidence and explicitly does not claim automatic traffic exercise.
- Evidence linkage rejects exercised jobs or surfaces outside the declared canary scope.
- Requirement, evidence, and satisfaction status/read models surface canary scope and evidence-source truth.

## Validation

- Focused package test: `/usr/local/go/bin/go test -count=1 ./internal/missioncontrol`
- Required full validation: `git diff --check` and `/usr/local/go/bin/go test -count=1 ./...`

No automatic traffic router, live canary traffic, external side effect, network call, or device action was added.
