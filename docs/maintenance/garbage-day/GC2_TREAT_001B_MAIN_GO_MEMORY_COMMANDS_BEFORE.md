# GC2-TREAT-001B Main.go Memory Commands Before

- Branch: `frank-v3-foundation`
- HEAD: `581f762a9835dd64d2ff7b4f5432eb043a7c0dff`
- Tags at HEAD: `frank-garbage-campaign-001a-maingo-clean`
- Ahead/behind `upstream/main`: `378 ahead / 0 behind`
- `git status --short --branch`:
  - `## frank-v3-foundation`
- Baseline `go test -count=1 ./...` result: passed

## Exact functions/regions selected for extraction

- the `memory` command builder subtree inside `NewRootCmd`
- memory subcommands:
  - `read`
  - `append`
  - `write`
  - `recent`
  - `rank`
- closely adjacent private helper construction only if needed to keep that subtree coherent

Selected source region:

- `cmd/picobot/main.go:1312-1514`

## Exact non-goals

- Do not change gateway boot behavior.
- Do not change mission bootstrap/runtime hooks.
- Do not change mission inspect read-model helpers.
- Do not change mission status projection/assertion logic.
- Do not change scheduled-trigger governance helpers.
- Do not change channels login behavior beyond import/wiring stability.
- Do not change command names, flags, help text, workspace/config resolution behavior, or current error messages.
- Do not widen the slice into broader CLI cleanup.

## Expected destination file

- `cmd/picobot/main_memory_commands.go`
