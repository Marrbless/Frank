# GC4-TREAT-003 TaskState Budget Split Before

Branch: `frank-garbage-day-gc4-003-taskstate-budget-split`

## Target

`GC4-003`: reduce `internal/agent/tools/taskstate.go` by extracting a small protected runtime/control cluster.

## Current Reassessment

`taskstate.go` still contains several high-risk clusters:

- runtime persistence/projection internals,
- approval and reboot-safe operator control,
- treasury activation,
- hot-update and rollback control,
- Frank Zoho work-item transitions.

Those clusters are not the safest first post-V4 TaskState move.

## Selected Slice

Move only the runtime budget enforcement pair:

- `EnforceUnattendedWallClockBudget`
- `RecordFailedToolAction`

Why this slice:

- It is small and coherent.
- It is protected runtime/control behavior, but not persistence-core, approval, treasury, or hot-update gate logic.
- It is already covered by focused TaskState budget tests and process-direct budget tests.

## Slice Constraint

Same-package move only. Preserve method names, return values, runtime state mutations, audit semantics, and call sites.
