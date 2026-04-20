## GD2-TREAT-002C Before

Date: 2026-04-20

### Live state

- Canonical repo: `/mnt/d/pbot/picobot`
- Branch: `frank-v3-foundation`
- HEAD: `98d55a50b587d51fb5acc20629e254e25da68af9`
- Ahead/behind `upstream/main`: `370 ahead / 0 behind`
- Worktree: clean before treatment
- Required docs present:
  - `docs/maintenance/garbage-day/GD2_TREAT_002_WEB_PROVIDER_LOG_SURFACE_ASSESSMENT.md`
  - `docs/maintenance/garbage-day/GARBAGE_CAMPAIGN_CHECKPOINT_002B.md`
- Baseline validation: `go test -count=1 ./...` passed before edits

### Remaining secret-unsafe onboarding and docs surfaces in scope

1. `cmd/picobot/main.go`
   - Interactive channel login still reads tokens with visible terminal echo via `promptLine`.
   - This affects Telegram, Discord, and Slack secret entry.

2. `internal/config/onboard.go`
   - Default provider placeholder still looks token-shaped: `sk-or-v1-REPLACE_ME`.
   - This unnecessarily normalizes real-key formatting inside generated config.

3. `docs/HOW_TO_START.md`
   - Tells operators to replace placeholders with actual keys directly in config examples.
   - Includes token-shaped Telegram, Discord, and Slack examples.
   - Includes guidance that can encourage copying live tokens into editors, screenshots, or transcripts.

4. `docs/CONFIG.md`
   - Default/full examples still include token-shaped API key and channel token examples.
   - MCP header example still uses a raw bearer token literal shape.

5. `README.md`
   - Docker and config examples still present secret values inline as literal-looking command/config arguments.
   - Quick-start/config text does not clearly warn against pasting live secrets into logs, issues, or chats.

### Selected bounded slice

- Add a secret-input prompt helper so interactive token entry is hidden on real terminals while preserving current non-terminal/test behavior.
- Replace token-shaped default/example placeholders with unmistakable non-secret placeholders.
- Add concise operator guidance in docs that:
  - config contains plaintext credentials
  - live secrets should not be pasted into logs, screenshots, issues, or chat transcripts
  - environment variables or local config editing should avoid embedding raw secrets in recorded commands where possible

### Explicit non-actions

- No V4 work.
- No broad README/docs reconciliation beyond the secret/onboarding hygiene lines above.
- No provider onboarding expansion.
- No treasury, campaign, capability, approval, or runtime semantic changes.
- No MCP feature expansion.
- No dependency changes.
- No commit.
