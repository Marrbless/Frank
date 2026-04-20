## GD2-TREAT-002C After

Date: 2026-04-20

### Live state after treatment

- Canonical repo: `/mnt/d/pbot/picobot`
- Branch: `frank-v3-foundation`
- Base HEAD before treatment: `98d55a50b587d51fb5acc20629e254e25da68af9`
- Upstream containment during treatment: `0 behind upstream/main`
- Commit action: none

### Implemented changes

1. `cmd/picobot/main.go`
   - Added `promptSecret(...)` to hide token entry on supported terminals via `golang.org/x/term`.
   - Preserved the existing line-reader path for non-terminal environments so scripted use and tests keep working.
   - Switched Telegram, Discord, and Slack token prompts to the secret-safe helper.
   - Added concise operator-facing text that token input is hidden on supported terminals.

2. `internal/config/onboard.go`
   - Replaced the default provider placeholder `sk-or-v1-REPLACE_ME` with the non-token-shaped placeholder `REPLACE_WITH_REAL_API_KEY`.

3. `README.md`
   - Reworked quick-start secret examples to use shell environment variables rather than inline literal values in `docker run` / `docker compose` examples.
   - Added concise warning text not to paste live secrets into logs, screenshots, issue reports, or chat transcripts.
   - Replaced config-example secret values with unmistakable placeholders.
   - Documented that `picobot channels login` hides token input on supported terminals.

4. `docs/CONFIG.md`
   - Added a top-level note that `config.json` stores credentials in plaintext and should be kept local.
   - Replaced token-shaped provider, channel, and MCP header examples with unmistakable placeholders.
   - Clarified that the interactive channel login flow hides token entry on supported terminals.

5. `docs/HOW_TO_START.md`
   - Replaced token-shaped provider, Telegram, Discord, and Slack examples with placeholders.
   - Tightened onboarding guidance to avoid encouraging live-secret pasting into screenshots, logs, or chats.
   - Documented that interactive channel login hides token input on supported terminals.

### Focused tests added or updated

- `cmd/picobot/main_test.go`
  - added coverage for `promptSecret(...)` fallback behavior in non-terminal mode
  - added coverage for the hidden-input path when terminal support is available
- `internal/config/onboard_test.go`
  - updated placeholder expectation for generated config
  - asserts the default placeholder is no longer token-shaped

### Validation

- `gofmt -w cmd/picobot/main.go cmd/picobot/main_test.go internal/config/onboard.go internal/config/onboard_test.go`
- `git diff --check`
  - passed
- `go test -count=1 ./internal/config`
  - passed
- `go test -count=1 ./cmd/picobot`
  - passed
- `go test -count=1 ./...`
  - passed

### Residual scope intentionally left untouched

- No V4 work.
- No broad docs reconciliation outside the selected secret/onboarding surfaces.
- No provider onboarding expansion.
- No treasury, campaign, capability, approval, or runtime semantic changes.
- No MCP feature expansion.
- No dependency changes.
