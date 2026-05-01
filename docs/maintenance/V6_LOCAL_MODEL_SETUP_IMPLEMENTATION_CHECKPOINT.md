# V6 Local Model Setup Implementation Checkpoint

Date: 2026-05-01

## Status

Frank V6 Local Model Setup Wizard is implemented for all safe, testable slices.

Implemented:

- V6-001 preset catalog, typed `EnvSnapshot`, pure planner, dry-run CLI.
- V6-002 interactive wizard shell, approval UI, preset list/inspect commands.
- V6-003 transactional config writer with backup, validation, and idempotence.
- V6-004 side-effect-free runtime/platform detector.
- V6-005 fakeable executor and Ollama manual-required/install framework.
- V6-006 llama.cpp register-existing path.
- V6-007 manifest-gated download framework with fake checksum tests.
- V6-008 deterministic Termux:Boot script generation.
- V6-009 metadata-only/no-prompt readiness checks.
- V6-010 Android/Termux operator docs.
- V6-011 end-to-end fake setup acceptance tests.
- V6-012 final acceptance validation.

## Human Input Still Required

Automatic real downloads remain blocked until reviewed manifests exist for:

- approved Ollama install/package paths by platform,
- approved llama.cpp binary sources,
- approved GGUF/model sources.

Each real manifest must include source URL, immutable version or release id,
checksum, size, license notes, platform, architecture, install command, and
safety notes.

## Validation Evidence

Commands run:

```sh
git diff --check
/usr/local/go/bin/go test -count=1 ./internal/modelsetup
/usr/local/go/bin/go test -count=1 ./cmd/picobot
make test-scripts
/usr/local/go/bin/go test -count=1 ./...
```

Results:

- `git diff --check`: passed.
- `go test ./internal/modelsetup`: passed.
- `go test ./cmd/picobot`: passed.
- `make test-scripts`: passed.
- `go test ./...`: passed.

## Safety Notes

- No real installer URLs, model URLs, versions, checksums, or API keys were added.
- Tests use fake command runners, fake manifests, temp dirs, and `httptest`.
- Local model presets default to `supportsTools=false`, `authorityTier=low`,
  cloud fallback disabled, and `127.0.0.1` binding.
- Cloud presets create stubs only and do not collect or print raw keys.
- Readiness checks are metadata-only and do not send prompts, tool arguments,
  Authorization headers, or request bodies.
