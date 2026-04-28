# V4-162 Autonomy Budget Enforcement Before

Branch: `frank-v4-162-autonomy-budget-enforcement`

## Matrix Row

- Requirement: `AC-035`
- Status before slice: `MISSING`
- Gap: standing directives could propose bounded work, but no durable autonomy budget record or local budget stop decision existed.

## Intended Slice

Add deterministic local autonomy budget enforcement:

- budget records with the V4 minimum fields,
- local debit assessment against wake-cycle records,
- budgeted standing-directive proposals,
- durable blocked wake cycles with `E_AUTONOMY_BUDGET_EXCEEDED`,
- status/read-model surfacing.

No real spend, external action, scheduler, phone hardware, or network behavior is in scope.
