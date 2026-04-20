## GD2-TREAT-002B Before

Date: 2026-04-20

### Live state

- Canonical repo: `/mnt/d/pbot/picobot`
- Branch: `frank-v3-foundation`
- HEAD: `a17eaf7997a65853d1b8a3c00411b5e6ef3c7572`
- Ahead/behind `upstream/main`: `368 ahead / 0 behind`
- Worktree: clean before treatment
- Required assessment doc: `docs/maintenance/garbage-day/GD2_TREAT_002_WEB_PROVIDER_LOG_SURFACE_ASSESSMENT.md` present
- Baseline validation: `go test -count=1 ./...` passed at this HEAD before any edits

### In-scope raw exposure surfaces

1. `internal/agent/tools/registry.go`
   - `SummarizeToolArguments` still logs raw argument key names.
   - Current operator log shape exposes tool-specific semantics such as `authorization`, `query`, `content`, `subject`, or `body`, even when values are redacted.

2. `internal/agent/loop.go`
   - Tool-activity notifications reuse `SummarizeToolArguments`, so raw argument key names are also surfaced back to the interactive channel when activity indicators are enabled.

3. `internal/channels/slack.go`
   - Inbound mention and DM logs still include truncated raw message text, which can expose secrets, prompt text, and attachment URLs.

4. `internal/channels/discord.go`
   - Inbound message logs still include truncated raw message text and inline attachment URLs.

5. `internal/channels/whatsapp.go`
   - Inbound message logs still include truncated raw message text.

### Selected bounded slice

- Replace raw tool-argument key logging with a narrower structural summary that preserves operator signal without exposing argument names or values.
- Update tool-activity notifications to inherit the same narrower summary.
- Remove raw inbound content fragments from Slack, Discord, and WhatsApp logs while keeping minimal operator-useful context such as source identifiers and message length.
- Add focused tests proving the new behavior does not emit raw content or raw tool-argument semantics on the treated surfaces.

### Explicit non-actions for this slice

- No V4 work.
- No broad cleanup outside the identified surfaces above.
- No provider onboarding or docs hygiene beyond this required treatment note.
- No treasury, campaign, capability, or approval semantic changes.
- No MCP feature expansion.
- No dependency changes.
- No commit.
