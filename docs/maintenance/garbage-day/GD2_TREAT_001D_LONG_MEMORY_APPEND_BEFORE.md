# GD2-TREAT-001D Long Memory Append Before

Date: 2026-04-19

## Live repo state

- Branch: `frank-v3-foundation`
- HEAD: `31a11ca20d45086c62198324ef116a936b77dd9e`
- Tags at HEAD: none
- Ahead/behind `upstream/main`: `364 ahead / 0 behind`
- `git status --short --branch`:

```text
## frank-v3-foundation
```

- Baseline `go test -count=1 ./...`: passed

## Exact long-memory append path inventory

- File: `internal/agent/tools/write_memory.go`
- Current append path for `target == "long"` and `append == true`:
  1. `prev, err := w.mem.ReadLongTerm()`
  2. `new := prev + "\n" + content`
  3. `err := w.mem.WriteLongTerm(new)`
  4. return `"appended to long-term memory"`

## Exact risk statement

- Lost update risk:
  - two callers can read the same old `MEMORY.md` contents and both write different derived results
  - last writer wins, so one append can be lost
- Replay/duplicate risk:
  - if the same append request is retried, the same content is appended again
  - there is no request identity or idempotence policy today
- Ambiguity:
  - current behavior does not distinguish intended repeated content from a duplicate retry

## Exact implementation plan

1. Add the smallest helper in `internal/agent/memory/store.go` needed for same-process replay/concurrency hardening:
   - serialize long-memory append work with the store mutex
   - read current `MEMORY.md`
   - define a narrow idempotence rule for retries
   - write updated content with existing 001B atomic overwrite behavior
2. Change `internal/agent/tools/write_memory.go` long-memory append mode to use that helper instead of manual read-modify-write.
3. Keep user-facing success strings unchanged.
4. Preserve the existing long-memory content format unless a minimal change is required.

## Exact tests planned

### `internal/agent/tools/write_memory_test.go`

- normal long-memory append success still works
- repeated identical append retry is idempotent under the defined rule
- concurrent long-memory append calls do not lose either appended value

### `internal/agent/memory/store_test.go`

- helper-level coverage if a small store helper is added
- explicit helper idempotence behavior

## Explicit non-goals

- no loop changes
- no session changes
- no overwrite atomic-write redesign
- no cross-process locking design
- no MCP changes
- no V4 work
- no broad cleanup
