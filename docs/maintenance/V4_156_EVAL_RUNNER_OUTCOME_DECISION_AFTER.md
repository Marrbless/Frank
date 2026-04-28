# V4-156 Eval Runner Outcome Decision After

Branch: `frank-v4-156-eval-runner-outcome-decision`

## Requirement Rows

- `AC-020` moved from `PARTIAL` to `DONE`.

## Implemented

- Added `ImprovementAttemptOutcomeRecord` under `runtime_packs/improvement_attempt_outcomes`.
- Added `CreateImprovementAttemptOutcomeFromCandidateResult`.
- Wired `RunLocalDeterministicEval` to create terminal attempt outcomes.
- Eligible `keep` outcomes create the existing candidate promotion decision record.
- Holdout/policy failures create `blocked` outcomes without promotion decisions.
- Explicit `discard` runner decisions create `discard` outcomes without promotion decisions.

## Validation

- Focused package test: `/usr/local/go/bin/go test -count=1 ./internal/missioncontrol`
- Required full validation: `git diff --check` and `/usr/local/go/bin/go test -count=1 ./...`

No activation path, hot-update execution, promotion execution, network call, AI call, external service call, or device side effect was added.
