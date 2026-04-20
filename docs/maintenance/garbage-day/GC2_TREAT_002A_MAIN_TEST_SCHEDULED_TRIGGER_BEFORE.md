GC2-TREAT-002A main_test scheduled-trigger governance split before

- branch: `frank-v3-foundation`
- head: `9fe99aa1041b8c40c8987120e1adc0f4a82e19eb`
- tags at head: `frank-garbage-campaign-002-main-test-memory-clean`
- ahead/behind upstream: `383 ahead / 0 behind`
- git status --short --branch:

```text
## frank-v3-foundation
```

- baseline `go test -count=1 ./...` result: passed

Exact tests selected for movement

- `TestRouteScheduledTriggerThroughGovernedJobCompletesMissionBoundReminder`
- `TestRouteScheduledTriggerThroughGovernedJobRejectsWhileAnotherMissionIsRunning`
- `TestGovernedScheduledTriggerDeferrerRecordsBlockedTriggerOnce`
- `TestGovernedScheduledTriggerDeferrerDeduplicatesReplay`
- `TestGovernedScheduledTriggerDeferrerDrainsDeferredTriggerThroughOrdinaryGovernedPath`

Exact helpers selected for movement

- `scheduledTriggerTestProvider`
- `(*scheduledTriggerTestProvider).Chat`
- `(*scheduledTriggerTestProvider).GetDefaultModel`
- `installScheduledTriggerTestPersistence`
- `testBlockingMissionJob`

Exact non-goals

- no production code changes
- no changes to mission inspect, mission status/assertion, mission bootstrap/runtime, watcher/operator-control, or other `main_test.go` families
- no shared helper extraction beyond the strictly local scheduled-trigger helper cluster
- no V4 work
- no dependency changes
- no test weakening or deletion

Expected destination file

- `cmd/picobot/main_scheduled_trigger_test.go`
