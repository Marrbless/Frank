# V4-146 Extension Blocker Status After

Branch: `frank-v4-146-extension-blocker-status`

## Requirement Rows

- `SF-005` moved from `PARTIAL` to `DONE`.
- `AC-031` remains `PARTIAL` until hot-update gate admission directly enforces extension assessments.
- `AC-027` remains `PARTIAL` until direct hot-update gate admission and package authority checks are complete.

## Implemented

- Added `extension_permission_assessment` to candidate promotion eligibility.
- Candidate promotion eligibility now compares baseline and candidate extension pack refs.
- Eligibility rejects candidates when extension permission assessment finds:
  - permission widening,
  - new external side-effect tools,
  - compatibility contract mismatch.
- Candidate-result status reads now surface the same structured blocker assessment.
- Existing agent/tool promotion fixtures now seed deterministic no-widening extension metadata so legacy eligible candidates stay eligible while missing/widened extension metadata still fails closed.

## Validation

- Focused package test: `/usr/local/go/bin/go test -count=1 ./internal/missioncontrol`
- Regression package tests: `/usr/local/go/bin/go test -count=1 ./internal/agent` and `/usr/local/go/bin/go test -count=1 ./internal/agent/tools`
- Required full validation: `git diff --check` and `/usr/local/go/bin/go test -count=1 ./...`

No approval authority, external service call, network call, or real plugin hot reload was added.
