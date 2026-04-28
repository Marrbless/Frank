# V4-146 Extension Blocker Status Before

Branch: `frank-v4-146-extension-blocker-status`

## Matrix Rows

- `SF-005` is `PARTIAL`: extension permission and compatibility assessment exists, but promotion/status reads do not surface it.
- `AC-031` is `PARTIAL`: extension permission widening checks exist but are not part of candidate promotion eligibility.
- `AC-027` remains `PARTIAL`: extension guardrails exist but need to affect promotion/hot-update eligibility.

## Slice

Wire extension permission assessment into candidate promotion eligibility and candidate-result status reads:

- compare baseline and candidate extension pack refs,
- reject eligibility when extension permissions widen,
- reject eligibility when compatibility contracts mismatch,
- include the assessment in `promotion_eligibility`.

This slice does not add approval authority and does not add real extension hot reload.
