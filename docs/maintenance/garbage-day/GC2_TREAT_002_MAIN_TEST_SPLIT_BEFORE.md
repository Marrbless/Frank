# GC2-TREAT-002 Main Test Split Before

- Branch: `frank-v3-foundation`
- HEAD: `ce7fcba0e83f1816b9025ac3604d138b889fbf1d`
- Tags at HEAD: none
- Ahead/behind `upstream/main`: `382 ahead / 0 behind`
- `git status --short --branch`:
  - `## frank-v3-foundation`
- Baseline `go test -count=1 ./...` result: passed

## Selected first split seam

- Move the `memory` CLI test family from `cmd/picobot/main_test.go` into `cmd/picobot/main_memory_test.go`
- Tests selected:
  - `TestMemoryCLI_ReadAppendWriteRecent`
  - `TestMemoryCLI_Rank`

## Non-goals

- Do not split mission bootstrap/runtime persistence families in this slice.
- Do not split scheduled-trigger tests in this slice.
- Do not extract shared fixtures/helpers in this slice.
- Do not change any production code or test behavior.
