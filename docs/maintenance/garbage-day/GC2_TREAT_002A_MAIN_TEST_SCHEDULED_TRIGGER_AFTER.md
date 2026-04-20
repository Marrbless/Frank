GC2-TREAT-002A main_test scheduled-trigger governance split after

- branch: `frank-v3-foundation`
- head: `9fe99aa1041b8c40c8987120e1adc0f4a82e19eb`
- production behavior changed: no

Git diff --stat

```text
 cmd/picobot/main_test.go | 457 -----------------------------------------------
 1 file changed, 457 deletions(-)
```

Untracked moved test file diff stat

```text
 .../picobot/main_scheduled_trigger_test.go         | 470 +++++++++++++++++++++
 1 file changed, 470 insertions(+)
```

Git diff --numstat

```text
0	457	cmd/picobot/main_test.go
```

Untracked moved test file diff numstat

```text
470	0	/dev/null => cmd/picobot/main_scheduled_trigger_test.go
```

Files changed

- `cmd/picobot/main_test.go`
- `cmd/picobot/main_scheduled_trigger_test.go`
- `docs/maintenance/garbage-day/GC2_TREAT_002A_MAIN_TEST_SCHEDULED_TRIGGER_BEFORE.md`
- `docs/maintenance/garbage-day/GC2_TREAT_002A_MAIN_TEST_SCHEDULED_TRIGGER_AFTER.md`

Exact tests moved

- `TestRouteScheduledTriggerThroughGovernedJobCompletesMissionBoundReminder`
- `TestRouteScheduledTriggerThroughGovernedJobRejectsWhileAnotherMissionIsRunning`
- `TestGovernedScheduledTriggerDeferrerRecordsBlockedTriggerOnce`
- `TestGovernedScheduledTriggerDeferrerDeduplicatesReplay`
- `TestGovernedScheduledTriggerDeferrerDrainsDeferredTriggerThroughOrdinaryGovernedPath`

Exact helpers moved

- `scheduledTriggerTestProvider`
- `(*scheduledTriggerTestProvider).Chat`
- `(*scheduledTriggerTestProvider).GetDefaultModel`
- `installScheduledTriggerTestPersistence`
- `testBlockingMissionJob`

Exact tests intentionally left in `main_test.go` and why

- Prompt/channel tests stayed because they belong to the operator-input/onboarding family, not scheduled-trigger governance.
- Agent CLI tests stayed because they are a separate small CLI surface and do not share scheduled-trigger fixtures.
- Mission inspect tests stayed because they depend on a large dedicated read-model fixture graph.
- Mission status, assert, assert-step, set-step, bootstrap, runtime, watcher, and operator-control tests stayed because they exercise higher-risk runtime-truth surfaces and share overlapping mission-state helpers.
- Package, prune, and gateway mission store logging tests stayed because they are not part of the scheduled-trigger family and do not depend on the moved helper cluster.

Validation commands and results

- `gofmt -w cmd/picobot/main_test.go cmd/picobot/main_scheduled_trigger_test.go` -> passed
- `git diff --check` -> passed
- `go test -count=1 ./cmd/picobot` -> passed
- `go test -count=1 ./...` -> passed

Deferred next candidates from the `main_test` split assessment

- Mission inspect test family split
- Mission status/assertion test family split
- Mission bootstrap/runtime/watcher/operator-control family split
