# PII Durable Field Inventory

Date: 2026-04-30

This inventory marks durable Picobot fields that can contain personal data,
provider identifiers, user/channel identifiers, message text, or credential
material. It is a minimization review aid, not a retention or deletion policy.

## Scope

Included:

- user-controlled config at `~/.picobot/config.json`,
- workspace session and memory files,
- mission-store JSON records, status snapshots, scheduler deferrals, and logs,
- provider-owned local data paths that Picobot config points at.

Excluded:

- transient in-memory chat hub messages,
- test fixtures except where they define a durable schema,
- third-party provider systems outside local Picobot files.

## Inventory

| Durable surface | Path or producer | Fields or content | Data class | Current minimization notes |
| --- | --- | --- | --- | --- |
| Config JSON | `internal/config/schema.go`, `~/.picobot/config.json` | `providers.openai.apiKey`, `channels.*.token`, Slack `appToken`/`botToken`, MCP `headers` | Credentials/secrets | Config is saved `0600`; startup warns on permissive secret-bearing config files. |
| Config JSON | `internal/config/schema.go`, `~/.picobot/config.json` | Telegram/Discord/WhatsApp `allowFrom`, Slack `allowUsers`, Slack `allowChannels`, WhatsApp `dbPath`, MCP `url`/`command`/`args` | User IDs, channel IDs, local provider path, endpoint metadata | Required for authorization/routing. Values are plaintext config, not redacted inside config. |
| Workspace sessions | `internal/session/manager.go`, `<workspace>/sessions/*.json` | `Session.Key`, `Session.History[]` | Chat/session IDs and message text | History is capped at 50 messages and filenames are base64url encoded, but JSON content remains plaintext. |
| Workspace memory | `internal/agent/memory/store.go`, `<workspace>/memory/MEMORY.md`, `<workspace>/memory/YYYY-MM-DD.md` | Free-form memory text and dated notes | Message text, user facts, arbitrary PII | Heartbeat/status noise is rejected by `write_memory`; otherwise content is operator/model supplied plaintext. |
| Chat hub routed to tools | `internal/chat/chat.go`, `internal/agent/tools/message.go` | `Inbound.SenderID`, `Inbound.ChatID`, `Inbound.Content`, `Outbound.ChatID`, `Outbound.Content` | User IDs, channel IDs, message text | Hub messages are in-memory, but downstream session, memory, logs, and mission approval records can persist selected values. |
| Mission status snapshot | `internal/missioncontrol/store_project.go` | `mission_file`, `runtime`, `runtime_summary`, `runtime_control`, `allowed_tools`, IDs/refs | Runtime metadata, paths, provider/status projections | Operator-facing JSON intentionally repeats committed runtime data; status projections may expose provider locators. |
| Mission runtime/control records | `internal/missioncontrol/store_records.go`, `internal/missioncontrol/types.go`, `internal/missioncontrol/runtime.go` | job/step IDs, `ExecutionHost`, surface refs, `Step.SuccessCriteria`, `LongRunningStartupCommand`, artifact paths, waiting/pause/failure reasons | Runtime metadata, paths, operator-authored text | Required for replay and inspection. Free-text reason fields can include sensitive details if callers provide them. |
| Approval records | `internal/missioncontrol/approval.go`, `internal/missioncontrol/store_records.go` | `SessionChannel`, `SessionChatID`, `RequestedAction`, `Scope`, `Reason`, `ApprovalRequestContent.*` | Chat IDs, authorization text, side-effect descriptions | Approval status projections include only selected fields, but persisted request/grant JSON keeps full approval content. |
| Audit records | `internal/missioncontrol/audit.go`, `internal/missioncontrol/store_records.go` | `ToolName`, `Reason`, `Code`, job/step IDs, timestamps | Tool/action metadata and free-form rejection text | Audit is capped in runtime state but durable audit event files may persist each committed event. Avoid putting raw message bodies in `Reason`. |
| Artifacts | `internal/missioncontrol/store_records.go` | `Path`, `VerificationCommand`, `VerificationOutput` | Filesystem paths, command output | Verification output can contain command stdout/stderr and should be treated as arbitrary operator/workspace text. |
| Current and packaged logs | `internal/missioncontrol/store_logs.go`, `cmd/picobot/main.go` | `logs/current.log`, `log_packages/*/gateway.log` | Application log text | Channel adapters hash user/chat IDs and omit raw inbound content, but generic log lines can still contain text supplied by other packages or errors. |
| Deferred scheduler records | `cmd/picobot/main.go`, `internal/missioncontrol/deferred_scheduler_status.go` | `Name`, `Message`, `Channel`, `ChatID`, `FireAt`, `DeferredAt` | Reminder text and chat IDs | Status projection omits channel/chat ID, but durable records keep them so deferred jobs can route later. |
| Campaign registry | `internal/missioncontrol/campaign_registry.go` | `DisplayName`, `Objective`, `StopConditions`, `ComplianceChecks`, Zoho `to`/`cc`/`bcc` | Campaign text and recipient email addresses | Provider-specific addressing is durable only for the approved Zoho campaign-email lane. |
| Campaign outbound records | `internal/missioncontrol/store_records.go`, `internal/missioncontrol/campaign_zoho_email_outbound_actions.go` | `ProviderAccountID`, `FromAddress`, `FromDisplayName`, `Addressing`, `Subject`, `BodySHA256`, provider/MIME/message IDs, `OriginalMessageURL`, failure descriptions | Email addresses, provider IDs, subject text, provider diagnostics | Body content is minimized to hash plus format; subject and recipient fields remain plaintext for audit and proof. |
| Frank Zoho send receipts | `internal/missioncontrol/store_records.go` | `ProviderAccountID`, `FromAddress`, `FromDisplayName`, provider/MIME/mail IDs, `OriginalMessageURL` | Email addresses and provider locators | Receipt stores proof locators, not email body. |
| Frank Zoho inbound replies | `internal/missioncontrol/store_frank_zoho_inbound_replies.go` | `ProviderAccountID`, provider/MIME/mail IDs, `InReplyTo`, `References`, `FromAddress`, `FromDisplayName`, `Subject`, `OriginalMessageURL` | Email addresses, provider locators, subject text | Reply body is not stored in the durable record; subject and sender metadata remain plaintext. |
| Frank Zoho bounce evidence | `internal/missioncontrol/store_frank_zoho_bounce_evidence.go` | provider/MIME/mail IDs, original provider IDs, `FinalRecipient`, `DiagnosticCode`, `OriginalMessageURL` | Email addresses, provider locators, mail diagnostics | Diagnostic code can contain provider-supplied text and should be treated as sensitive. |
| Identity/account registry | `internal/missioncontrol/identity_registry.go` | display names, provider/platform IDs, Zoho mailbox address/display name, Telegram owner user ID, Telegram bot user ID, organization/account IDs, env var names | User IDs, provider IDs, email addresses, identity labels | Secret values are referenced by env var name where present; env var names themselves can reveal provider/account purpose. |
| WhatsApp provider store | `internal/channels/whatsapp.go`, config `channels.whatsapp.dbPath` | SQLite auth/session data owned by `whatsmeow` | Provider session/auth data and contact/JID metadata | Picobot config points at this durable provider DB; schema is owned by the dependency and should be treated as sensitive. |

## Minimization Observations

- Channel logs currently minimize inbound message body logging to character and attachment counts, and hash user/chat IDs before writing standard logs.
- Email outbound body content is not persisted in mission records; `body_sha256` records only a content hash.
- Session history and memory files are the highest-risk plaintext text surfaces because they intentionally preserve conversational content.
- Mission status snapshots duplicate selected runtime/provider fields for operator inspection; do not treat them as sanitized exports.
- Provider identifiers, message IDs, mail IDs, MIME IDs, and `original_message_url` values are linkable provider locators even when they do not look like human-readable PII.

## Open Policy Decisions

- Retention windows for session history files, memory notes, mission audit records, logs, and provider proof records.
- Whether operator status snapshots should redact or hash `SessionChatID`, provider locators, or email addresses for any export mode.
- Whether memory writes need an explicit PII class, deletion workflow, or owner approval when content resembles credentials or third-party personal data.
- Whether provider-owned local stores such as WhatsApp SQLite data need a documented backup/exclusion rule.
