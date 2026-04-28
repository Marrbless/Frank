# GC4-TREAT-002 ProcessDirect Inspect Split Before

Branch: `frank-garbage-day-gc4-002-processdirect-inspect-split`

## Target

`GC4-002`: split one command family out of `internal/agent/loop_processdirect_test.go`.

## Starting Evidence

- `internal/agent/loop_processdirect_test.go`: `11038` lines at post-V4 kickoff.
- The `INSPECT` command tests were a contiguous family near the operator command tests.
- The family covered:
  - active job summary,
  - wrong job rejection,
  - unknown step rejection,
  - persisted paused runtime inspect,
  - terminal runtime inspect.

## Slice Constraint

Move only the inspect command tests into a same-package test file. Leave shared fixtures in place and preserve test names, assertions, helper calls, command strings, and behavior.
