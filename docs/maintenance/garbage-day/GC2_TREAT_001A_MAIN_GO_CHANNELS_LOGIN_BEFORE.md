# GC2-TREAT-001A Main.go Channels Login Before

- Branch: `frank-v3-foundation`
- HEAD: `2ad2e2530237eb19998294bbab04a41f610f142e`
- Tags at HEAD: none
- Ahead/behind `upstream/main`: `377 ahead / 0 behind`
- `git status --short --branch`:
  - `## frank-v3-foundation`
- Baseline `go test -count=1 ./...` result: passed

## Exact functions/regions selected for extraction

- `promptLine`
- `promptSecret`
- `parseAllowFrom`
- `setupTelegramInteractive`
- `setupDiscordInteractive`
- `setupSlackInteractive`
- `setupWhatsAppInteractive`
- the `channels login` command block inside `NewRootCmd` that calls those helpers

Selected source regions:

- `cmd/picobot/main.go:459-507`
- `cmd/picobot/main.go:2999-3219`

## Exact non-goals

- Do not change gateway boot behavior.
- Do not change mission bootstrap/runtime hooks.
- Do not change mission status projection/assertion logic.
- Do not change provider or MCP runtime behavior.
- Do not change onboarding authority model.
- Do not change prompts, hidden-secret input behavior, config writes, or current error messages.
- Do not widen the slice into memory commands, scheduled trigger helpers, or mission-control extraction.

## Expected destination file

- `cmd/picobot/main_channels_login.go`
