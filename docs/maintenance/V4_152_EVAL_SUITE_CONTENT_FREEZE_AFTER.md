# V4-152 Eval Suite Content Freeze After

Branch: `frank-v4-152-eval-suite-content-freeze`

## Requirement Rows

- `AC-008` moved from `PARTIAL` to `DONE`.

## Implemented

- Added eval-suite content identity fields:
  - `rubric_sha256`
  - `train_corpus_sha256`
  - `holdout_corpus_sha256`
  - `evaluator_sha256`
  - `frozen_content_ref`
- Eval-suite storage validates required SHA-256 identities and rejects train/holdout content identity collapse.
- Divergent duplicate eval-suite storage still fails closed.
- Candidate-result linkage fails closed if the referenced eval-suite content identity is corrupted.
- Operator eval-suite status/read models surface frozen content identities for rubric, train corpus, holdout corpus, evaluator, and frozen content refs.

## Validation

- Focused package test: `/usr/local/go/bin/go test -count=1 ./internal/missioncontrol`
- Required full validation: `git diff --check` and `/usr/local/go/bin/go test -count=1 ./...`

No external evaluator call, network call, AI call, service dependency, or evaluator mutation path was added.
