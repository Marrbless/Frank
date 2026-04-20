# GD2-TREAT-002 Web / Provider Log-Surface Assessment

Date: 2026-04-19

## 1. Live repo state

- Canonical repo: `/mnt/d/pbot/picobot`
- Branch: `frank-v3-foundation`
- HEAD: `4dd343dd4c1598be09f7b3ae137757ef443bf5c5`
- Tags at HEAD: none
- Ahead/behind `upstream/main`: `366 ahead / 0 behind`
- Baseline test result: `go test -count=1 ./...` passed before this assessment at the above HEAD
- Preconditions checked:
  - branch is `frank-v3-foundation`
  - `HEAD` contains `upstream/main`
  - worktree was clean before this assessment
  - `docs/maintenance/garbage-day/ROUND_2_REPO_DIAGNOSIS.md` exists
  - `docs/maintenance/garbage-day/GD2_TREAT_001_PROGRESS_SUMMARY.md` exists

## 2. Surface inventory

| Package / file | Why it is in scope | Logs? | Data shape at risk | Surface type |
| --- | --- | --- | --- | --- |
| `internal/providers/openai.go` | primary provider HTTP path | yes | raw non-2xx provider body text, provider status | provider-facing, debug/error |
| `internal/providers/factory.go` | provider selection from config | no | config-driven provider selection only | none |
| `internal/channels/discord.go` | inbound/outbound chat logging | yes | sender name, sender ID, channel ID, first 50 chars of content, attachment URL prefix | operator/debug |
| `internal/channels/slack.go` | inbound/outbound chat logging | yes | user ID, channel ID, first 50 chars of content, private attachment URLs, allowlist decisions | operator/debug |
| `internal/channels/telegram.go` | channel HTTP errors and access-control logs | yes | unauthorized user ID, transport errors | operator/debug |
| `internal/channels/whatsapp.go` | channel logging plus wrapped library logger | yes | sender JID, chat JID, first 50 chars of content, unauthorized sender IDs, connected account identifiers, library-supplied log text | operator/debug |
| `internal/channels/whatsapp_stub.go` | lite-build startup notice | yes | availability only | operator/debug |
| `internal/mcp/client.go` | MCP HTTP/stdin transport and error propagation | no direct log calls | raw HTTP response body text, remote tool error text, configured HTTP headers sent on requests, stderr inherited from child process | provider-facing, error propagation |
| `internal/agent/loop.go` | provider/tool/MCP startup and failure logging | yes | channel + sender IDs, provider errors, MCP connect errors, raw tool errors in activity notifications, full tool errors fed back into model loop | operator/debug, user-visible, model-visible |
| `internal/agent/tools/registry.go` | common execution wrapper for all tools | yes | full tool argument JSON, raw tool error text, audit metadata | operator/debug plus audit-like |
| `internal/agent/tools/web.go` | direct web fetch tool | no direct log calls | fetched URL in args via registry log, full response body returned to tool chain | provider-facing through registry/model path |
| `internal/agent/tools/web_search.go` | external search HTTP path | no direct log calls | search query in args via registry log, transport errors, HTTP status | provider-facing through registry/model path |
| `internal/agent/tools/mcp.go` | MCP tool wrapper | no direct log calls | raw MCP tool result or error text propagated to registry/loop | provider-facing through registry/model path |
| `internal/agent/tools/frank_zoho_send_email.go` | provider-facing mail HTTP path used by tool layer | no direct log calls | recipient addresses, subject, body, provider account IDs, original-message URLs, provider status descriptions, raw original message payloads | provider-facing, error propagation, audit-adjacent |
| `internal/config/onboard.go` | default config, secret-handling guidance, config write | no logging | placeholder API key written to config template, safety guidance text | config/bootstrap |
| `internal/config/loader.go` | runtime config load and env overrides | no logging | config and env values loaded without redaction layer | config/bootstrap |
| `cmd/picobot/main.go` | onboarding, startup, direct CLI error display, channel login prompts | yes | config path/workspace path, token prompts echoed to terminal, startup/mission logs, direct CLI `error: %v` output | operator-facing, startup, user-visible |
| `docs/CONFIG.md` | config and MCP header guidance | n/a | bearer-header example, token-shaped config examples, provider API key placeholder | documentation |
| `docs/HOW_TO_START.md` | onboarding guidance | n/a | token-shaped examples and direct editing guidance for provider/channel secrets | documentation |

## 3. Logging call inventory

| File | Function | Log type | Exact category of data at risk | Severity | Confidence |
| --- | --- | --- | --- | --- | --- |
| `internal/providers/openai.go:426-451` | `(*OpenAIProvider).doJSON` | `log.Printf` + returned error | raw remote provider response body; can contain prompt fragments, provider account identifiers, request echo, quota/account text | High | High |
| `internal/agent/loop.go:1409-1413` | agent loop provider call path | `log.Printf("provider error: %v")` | repeats provider error text from `openai.go`, including raw remote body fragments | High | High |
| `internal/agent/loop.go:1585-1587` + `cmd/picobot/main.go:552-557` | `ProcessDirect` + CLI `agent` command | returned error surfaced to stderr | direct user-visible provider internals in CLI mode | High | High |
| `internal/agent/tools/registry.go:184-197` | `(*Registry).Execute` | `log.Printf` | full tool argument JSON for every tool call; includes memory text, web URLs/queries, MCP args, email tool args, file paths, command args | High | High |
| `internal/agent/tools/registry.go:192-194` | `(*Registry).Execute` failure path | `log.Printf` | raw tool error text from MCP servers, Zoho provider status descriptions, other downstream errors | High | High |
| `internal/agent/tools/registry.go:201-209` | `logAuditEvent` / `emitAuditEvent` | `log.Printf` | audit metadata only: job ID, step ID, tool name, code, reason, timestamp | Medium | High |
| `internal/agent/loop.go:1472-1476` | chat loop tool-activity notification | user-facing notification | raw tool error string sent to chat channel when tool activity indicator is enabled | High | High |
| `internal/agent/loop.go:1476` and `1603-1660` | tool result feedback into provider loop | model-visible tool message | raw tool error text and raw successful tool results reintroduced into model conversation | High | High |
| `internal/agent/loop.go:1478-1482`, `1646-1649`; `internal/missioncontrol/step_validation.go:43-47` | successful tool evidence capture | audit-adjacent persistence | full successful tool arguments and result text persisted as runtime validation evidence | High | Medium |
| `internal/mcp/client.go:333-335` | `(*httpTransport).doPost` | returned error | raw HTTP error body from remote MCP server | High | High |
| `internal/mcp/client.go:103-105` | `(*Client).CallTool` | returned error | raw MCP tool error text from remote server content | High | High |
| `internal/agent/loop.go:1121-1140` | MCP startup registration | `log.Printf` | raw MCP connect failure text; may include HTTP body text or child-process stderr context | Medium-High | High |
| `internal/channels/discord.go:149-150` | `(*discordClient).handleMessage` | `log.Printf` | sender display name, sender ID, channel ID, first 50 chars of content, attachment URL prefix if present | High | High |
| `internal/channels/discord.go:108-110` | `(*discordClient).handleMessage` | `log.Printf` | unauthorized username and user ID | Medium | High |
| `internal/channels/slack.go:164`, `210`, `269-280` | `handleMention`, `handleMessage`, `logUnauthorized` | `log.Printf` | user ID, channel ID, first 50 chars of content, private attachment URLs, allowlist decision details | High | High |
| `internal/channels/telegram.go:119-143`, `159-162`, `194-196` | polling / outbound sender | `log.Printf` | unauthorized user ID and raw HTTP transport errors; no message content log | Medium | High |
| `internal/channels/whatsapp.go:61-82`, `121-124`, `301-322` | library logger, startup, inbound handler | `log.Printf` | library-supplied WhatsApp log text, connected account IDs, sender/chat JIDs, first 50 chars of content | High | Medium-High |
| `cmd/picobot/main.go:2995-2999`, `3018-3159` | `promptLine`, interactive setup helpers | terminal output | channel/provider tokens are entered with normal terminal echo; secrets visible on-screen and in terminal capture history | High | High |
| `docs/CONFIG.md:224-248`, `273-333`; `docs/HOW_TO_START.md:295-412`, `500-517` | docs examples | documentation examples | bearer-header example and realistic token-shaped examples normalize plaintext secret handling | Medium | High |

## 4. Sensitive data risk matrix

| Data class | Entry surface | Logged / exposed where | Current guard | Suspected weakness | Severity | Treatment candidate |
| --- | --- | --- | --- | --- | --- | --- |
| Auth headers | MCP HTTP config headers; provider auth headers; Zoho auth header | not logged directly, but MCP HTTP error bodies and downstream tool/provider errors can leak adjacent auth context | no explicit header logging in code | raw remote error bodies and raw tool errors are surfaced without redaction; generic tool arg logging makes header-bearing tools risky if added later | High | `GD2-TREAT-002A` |
| API keys / tokens / secrets | onboarding prompts, config file, provider config, channel config | interactive token prompts echo to terminal; docs show token-shaped examples; provider/MCP error paths may echo remote auth failures | empty-token validation only; docs warn about secrecy in a few places | no secret input masking; no centralized redaction; docs normalize plaintext secret editing | High | `GD2-TREAT-002C` |
| Cookies / session values | future MCP HTTP headers, fetched web content, remote error bodies | no direct cookie logs found | none beyond absence of explicit cookie logging | raw remote body propagation would leak cookie/session text if provider/server echoes it | Medium | `GD2-TREAT-002A` |
| Provider account identifiers | Zoho provider account ID, message ID, original message URL; OpenAI account/quota identifiers | `frank_zoho_send_email` errors propagated into registry/loop; `openai.go` logs raw body; adjacent operator status projections carry provider fields | IDs are not intentionally redacted | raw provider descriptions and URLs can surface to logs, chat notifications, CLI stderr, and model loop | High | `GD2-TREAT-002A` |
| Chat contents | inbound channel messages | Discord/Slack/WhatsApp log first 50 chars; Slack/Discord append attachment URLs before truncation | truncation to 50 chars in three channels; Telegram does not log message text | first 50 chars still enough to leak secrets, names, prompt text, or URLs; private attachment URLs often begin at the front and defeat truncation value | High | `GD2-TREAT-002B` |
| Memory contents | `write_memory`, `edit_memory`, `read_memory`, long-memory tools | `registry.go` logs raw tool arg JSON; successful tool evidence persists args/results | none | durable personal notes and remembered facts are logged and may also persist as step evidence | High | `GD2-TREAT-002B` |
| Email bodies / subjects / recipients | `frank_zoho_send_email` tool args and provider errors | raw tool arg logging in registry; tool errors logged and optionally pushed to user via activity notifications; successful tools may persist result text | none | recipients, subject, body, reply-thread identifiers, and original-message data can cross debug, user-visible, and audit-adjacent channels | High | `GD2-TREAT-002B` then `002A` |
| Attachments / URLs | Slack/Discord attachment URLs; web tool URLs; Zoho original-message URLs | channel message logs; registry arg logs; operator status projections adjacent to scoped surface | truncate(content, 50) only on channel logs | private Slack/Discord URLs and Zoho URLs remain identifying and may be directly usable if logged | High | `GD2-TREAT-002B` |
| Model prompts / tool args / web content | provider request bodies, tool args, tool results, fetched content | `openai.go` raw non-2xx body logging; registry raw args logging; loop replays raw tool errors/results back into model | none | remote providers can echo submitted prompt text; generic registry logging makes all tools high-risk; tool results can contain fetched web pages or provider mail bodies | High | `GD2-TREAT-002A` and `002B` |
| Personally identifying information | sender names, user IDs, channel IDs, chat JIDs, phone-linked IDs | Discord/Slack/Telegram/WhatsApp logs; onboarding prompts for allowlist IDs; WhatsApp connected account logs | Telegram avoids content logging; some channels only log IDs on unauthorized access | identifiers are still durable PII in logs; WhatsApp and Discord include stronger identity detail than needed for normal ops | Medium-High | `GD2-TREAT-002B` |

## 5. Audit-vs-debug separation

### Current state

- The repo does have a real audit concept in missioncontrol:
  - `internal/missioncontrol/audit.go:39-50`
  - `internal/agent/tools/registry.go:201-209`
- The audit line itself is relatively narrow:
  - job ID
  - step ID
  - tool name
  - allow/deny
  - code
  - reason
  - timestamp
- The separation is weak in practice because audit lines and debug/error lines all go through the same process logger.

### Audit logs that include too much data

- I did not find a scoped audit log call that directly dumps full tool args or full tool results.
- The adjacent risk is not the audit line itself; it is the nearby debug line in the same execution wrapper:
  - `internal/agent/tools/registry.go:185-186` logs raw tool args
  - `internal/agent/tools/registry.go:193` logs raw tool errors
- In effect, the same tool execution emits:
  - one narrow audit line
  - plus one broad debug line that defeats the narrowness

### Debug logs that should be redacted

- `internal/providers/openai.go:445`
  - should not log raw remote response body text
- `internal/agent/tools/registry.go:186`
  - should not log raw tool argument JSON
- `internal/agent/tools/registry.go:193`
  - should not log raw downstream error text without redaction/classification
- `internal/channels/discord.go:150`, `internal/channels/slack.go:164`, `internal/channels/slack.go:210`, `internal/channels/whatsapp.go:322`
  - should not log message content fragments by default
- `internal/channels/slack.go:280`, `internal/channels/discord.go:109`, `internal/channels/whatsapp.go:301`
  - should minimize identifier detail for unauthorized access logs
- `internal/channels/whatsapp.go:61-82`
  - third-party library text currently flows into the standard logger with no redaction boundary

### User-visible errors that may leak provider internals

- `internal/agent/loop.go:1472-1475`
  - tool activity indicator sends raw tool error text to the chat channel
- `internal/agent/loop.go:1644-1659`
  - raw tool errors are reintroduced into the model conversation as `(tool error) ...`
- `internal/agent/loop.go:1585-1587` plus `cmd/picobot/main.go:552-557`
  - direct CLI mode prints raw provider errors to stderr

### Tests that normalize unsafe logging patterns

- `internal/agent/tools/registry_test.go:201-313`, `365-399`
  - explicitly asserts audit log lines are emitted
  - this is good coverage for audit presence, but it does not test redaction of args or errors
- `cmd/picobot/main_test.go:4405-4419`, `7239-7250`, `9477-9491`
  - explicitly asserts mission control success logs include job IDs, step IDs, control paths, and status paths
  - these are operationally useful, but they also normalize rich operational logging as test-approved behavior
- No scoped tests assert:
  - provider error-body redaction
  - tool-arg redaction
  - channel-content redaction
  - secret input masking during onboarding

## 6. Provider / web / MCP-specific risks

### Provider request/response logging

- `internal/providers/openai.go` sends full message history, memory context, tool definitions, and tool-call arguments to the provider.
- The request body is not logged directly.
- The response failure path is still high-risk because:
  - `doJSON` logs the full non-2xx body at `openai.go:445`
  - `doJSON` returns that full body again inside the error at `openai.go:449`
  - `loop.go` logs the returned error again at `1411`
  - direct CLI mode returns the raw provider error upward at `1585-1587`
- Net effect:
  - one remote failure can produce duplicated sensitive fragments in process logs and in CLI-visible stderr

### Web fetch / search / tool logging

- `internal/agent/tools/web.go`
  - no local logging
  - still risky because `registry.go` logs raw `url` args for every call
  - returns full body regardless of status, so downstream errors can become ambiguous, but that is a correctness issue more than a log-only issue
- `internal/agent/tools/web_search.go`
  - no local logging
  - query terms are still logged via `registry.go`
  - transport errors are wrapped and then logged by `registry.go` if execution fails

### MCP request/response logging

- `internal/mcp/client.go` does not emit process logs directly.
- The main risk is raw response propagation:
  - `HTTP %d: %s` at `client.go:335`
  - `tool error: %s` at `client.go:104`
- Those strings flow into:
  - `internal/agent/loop.go:1133` during MCP startup failures
  - `internal/agent/tools/registry.go:193` during tool failures
  - `internal/agent/loop.go:1472-1476` if tool activity notifications are enabled
- The HTTP transport also applies configured custom headers from config at `client.go:313-315`.
  - I did not find the headers themselves being logged.
  - I did find no redaction boundary if a remote server echoes them or adjacent secret context in an error body.

### Config onboarding or startup logs

- `cmd/picobot/main.go:2996-2999` uses plain terminal input for secret prompts.
  - channel tokens are echoed visibly while typed.
- `cmd/picobot/main.go:450` prints config path and workspace path only.
  - low sensitivity, but still operator-facing.
- `internal/config/onboard.go` and `internal/config/loader.go`
  - do not log secrets
  - do store and load plaintext credentials from config as designed on this branch

### Retry / error logs with payload fragments

- OpenAI:
  - raw body text on non-2xx
- MCP:
  - raw HTTP body text
  - raw tool error text
- Zoho mail tool:
  - remote status descriptions are preserved in returned errors
  - those then flow into registry logs and tool-activity notifications
- Channel adapters:
  - no retries dump full payloads, but regular inbound logs already include truncated content and IDs

## 7. Test coverage

### Existing tests that already protect part of the surface

- `internal/agent/tools/registry_test.go`
  - verifies audit lines are emitted exactly once
  - verifies no full `event={...}` payload dump is repeated in audit logs
- `internal/channels/slack_test.go`
  - validates token prefix checks and attachment handling behavior
- `internal/channels/discord_test.go`, `telegram_test.go`, `whatsapp_test.go`
  - validate transport behavior and allowlist behavior
- `internal/mcp/client_test.go`, `internal/agent/tools/mcp_test.go`
  - validate transport behavior and error propagation
- `internal/providers/openai_test.go`
  - validates parsing behavior only

### Coverage gaps

- No scoped test verifies provider error redaction.
- No scoped test verifies MCP HTTP/body redaction.
- No scoped test verifies raw tool arg logging is suppressed or redacted.
- No scoped test verifies raw tool errors are not sent through chat activity notifications.
- No scoped test verifies channel content logs are absent or redacted.
- No scoped test verifies attachment URLs are not logged.
- No scoped test verifies onboarding secret prompts disable terminal echo.
- No scoped test verifies direct CLI provider errors are sanitized before user display.
- No scoped test verifies `EnableToolActivityIndicator=true` is safe from secret-bearing tool errors.

## 8. Treatment backlog

### GD2-TREAT-002A

- Name: Provider and MCP error-surface redaction
- Exact files:
  - `internal/providers/openai.go`
  - `internal/mcp/client.go`
  - `internal/agent/loop.go`
  - `internal/agent/tools/registry.go`
- Exact smallest safe slice:
  - stop logging raw remote response bodies
  - stop returning raw remote body text in surfaced provider/MCP errors by default
  - replace raw error logging with status/code-only plus local correlation context
  - stop sending raw tool/provider error text through tool-activity notifications
- Severity: High
- Confidence: High
- Tests required:
  - provider non-2xx redaction test
  - MCP HTTP non-200 redaction test
  - MCP tool error redaction test
  - loop tool-activity notification sanitization test
  - direct CLI/provider error sanitization test
- Before V4: yes

### GD2-TREAT-002B

- Name: Tool-arg and channel-content log minimization
- Exact files:
  - `internal/agent/tools/registry.go`
  - `internal/channels/discord.go`
  - `internal/channels/slack.go`
  - `internal/channels/telegram.go`
  - `internal/channels/whatsapp.go`
- Exact smallest safe slice:
  - replace raw `argsJSON` logging with per-tool safe summaries
  - remove content-fragment logging from inbound channel logs by default
  - stop logging private attachment URLs
  - reduce unauthorized-access logs to minimal identity/context needed for ops
  - keep audit lines narrow and separate from debug lines
- Severity: High
- Confidence: High
- Tests required:
  - registry log-redaction tests for memory/web/MCP/email-style args
  - channel log tests proving content and attachment URLs are not emitted
  - unauthorized-log minimization tests
- Before V4: yes

### GD2-TREAT-002C

- Name: Secret-safe onboarding and docs hygiene
- Exact files:
  - `cmd/picobot/main.go`
  - `internal/config/onboard.go`
  - `docs/CONFIG.md`
  - `docs/HOW_TO_START.md`
- Exact smallest safe slice:
  - make secret prompts non-echoing for provider/channel tokens
  - document that config contains plaintext secrets and should not be copied into logs
  - replace token-shaped examples with unmistakable placeholders
  - document what startup/debug logs intentionally omit after 002A/002B
- Severity: Medium-High
- Confidence: High
- Tests required:
  - prompt helper tests for non-echoing secret entry
  - docs regression check for placeholder examples only
- Before V4: yes

## 9. Explicit non-actions

- Do not redesign the web tool’s HTTP semantics in this treatment.
  - `internal/agent/tools/web.go` returning bodies for non-2xx is a real issue, but it is a behavior/correctness change separate from log-surface hardening.
- Do not refactor the missioncontrol audit/store/readout model in this slice.
  - The protected V3 operator surface is larger than the selected treatment; only the narrow log-surfaces feeding into it should move first.
- Do not broaden this assessment into `exec`, `filesystem`, or unrelated taskstate cleanup.
  - Those are real exposure surfaces, but they are outside the selected `GD2-TREAT-002` web/provider cluster.
- Do not change third-party WhatsApp library behavior beyond the local logger adapter boundary when implementation begins.
  - The smallest safe slice is to harden what the repo forwards, not to fork or redesign the library.
- Do not start V4 logging or observability work from this document.
  - The selected treatment is branch-line hardening of current V3/V2 surfaces only.

## Validation

- `git diff --check`
  - passed
- `go test -count=1 ./...`
  - passed
