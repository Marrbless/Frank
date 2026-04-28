# GC4-TREAT-004 Bootstrap Store Root Split Before

Branch: `frank-garbage-day-gc4-004-bootstrap-store-root-split`

## Target

`GC4-004`: split one runtime bootstrap command subfamily from `cmd/picobot/main_runtime_bootstrap_test.go`.

## Starting Evidence

- `cmd/picobot/main_runtime_bootstrap_test.go`: `6555` lines at post-V4 kickoff.
- The mission store-root resolution tests were a small, self-contained subfamily using the existing bootstrap test command helper.
- The subfamily covered:
  - explicit mission store root,
  - fallback from status file,
  - empty result without inputs.

## Slice Constraint

Move only the mission store-root resolution tests. Preserve test names, assertions, helper use, and command flag behavior. Do not touch runtime persistence, control-file watcher, approval, or durable bootstrap tests.
