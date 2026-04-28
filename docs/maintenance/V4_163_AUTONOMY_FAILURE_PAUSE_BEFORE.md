# V4-163 Autonomy Failure Pause Before

Branch: `frank-v4-163-autonomy-failure-pause`

## Matrix Row

- Requirement: `AC-036`
- Status before slice: `MISSING`
- Gap: autonomy wake cycles and budgets existed, but repeated failed autonomous work had no durable pause state.

## Intended Slice

Add deterministic local repeated-failure handling:

- wake-cycle failure records,
- repeated-failure pause records keyed by budget,
- threshold evaluation using `max_failed_attempts_before_pause`,
- proposal blocking with `E_REPEATED_FAILURE_PAUSE`,
- status/read-model surfacing.

No process supervision, external side effects, real device behavior, or hot-update crash handling is in scope.
