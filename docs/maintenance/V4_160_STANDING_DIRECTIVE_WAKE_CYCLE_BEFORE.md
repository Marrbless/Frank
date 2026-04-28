# V4-160 Standing Directive Wake Cycle Before

Branch: `frank-v4-160-standing-directive-wake-cycle`

## Matrix Row

- Requirement: `AC-033`
- Status before slice: `MISSING`
- Gap: no durable standing directive or wake-cycle records existed for local autonomous scheduling.

## Intended Slice

Add deterministic local records for:

- standing directives with due interval schedules and autonomy scope refs,
- wake-cycle mission proposal outcomes for due directives,
- fail-closed rejection for not-due, retired, paused, or disallowed selections.

No external service, network, phone hardware, owner-approval binding, or hot-update activation behavior is in scope.
