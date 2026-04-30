# Autonomous Review Brief Template

Use this after a matrix row has validation evidence and before asking for human
review.

## Row

- Row ID:
- Branch/HEAD:
- Scope:

## Diff Summary

- Files changed:
- Behavior changed:
- Behavior intentionally preserved:

## Validation Evidence

| Command | Result | Notes |
|---------|--------|-------|
| `...` | passed/failed/blocked | exact blocker or relevant output |

## Reviewer Focus

- Regression risks:
- Operator-facing text/schema risks:
- Security or privacy risks:
- Missing evidence:

## Remaining Risk

State residual risk plainly. If evidence is missing, mark it as a real risk and
name the next validator or human action needed.
