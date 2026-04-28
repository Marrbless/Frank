# V4-152 Eval Suite Content Freeze Before

Branch: `frank-v4-152-eval-suite-content-freeze`

## Requirement Rows

- `AC-008` was `PARTIAL`.

## Observed Gap

- Eval suites were required to be `frozen_for_run`.
- The deterministic local eval runner loaded the frozen eval suite before producing candidate results.
- Eval-suite records did not carry content identities for rubric, train corpus, holdout corpus, evaluator, or a frozen content copy ref.

## Intended Slice

- Add deterministic content identity fields to eval-suite records.
- Require SHA-256 identities for rubric, train corpus, holdout corpus, evaluator, and a frozen content copy ref.
- Preserve existing local deterministic runner behavior and avoid external eval calls or mutable evaluator behavior.
