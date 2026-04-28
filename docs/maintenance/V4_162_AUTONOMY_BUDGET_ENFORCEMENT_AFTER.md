# V4-162 Autonomy Budget Enforcement After

Branch: `frank-v4-162-autonomy-budget-enforcement`

## Completed

- Added `AutonomyBudgetRecord` storage under `autonomy/budgets`.
- Added deterministic budget assessment over same-day wake-cycle debits.
- Made standing-directive mission proposals record a candidate-mutation debit.
- Added durable blocked wake cycles with `E_AUTONOMY_BUDGET_EXCEEDED` when the budget is exhausted.
- Added budget and last-budget-exceeded fields to `autonomy_identity`.

## Validation

- `/usr/local/go/bin/go test -count=1 ./internal/missioncontrol`

## Remaining

- `AC-036`: repeated autonomy failure pause.
- `AC-037`: owner pause record that blocks autonomy-originated hot-update proposals.
