# Tool Permission Manifest

Date: 2026-04-30

This manifest records the built-in agent tool surface exposed by `internal/agent/loop.go` and the enforcement model in `internal/agent/tools/registry.go` and `internal/missioncontrol/guard.go`.

## Enforcement Model

- Tool names in mission `allowed_tools` are exact model-visible names.
- `AgentLoop` registers local tools in `NewAgentLoop`; MCP tools are registered dynamically as `mcp_<server>_<tool>`.
- `Registry.DefinitionsForExecutionContext` exposes only tools allowed by the active job and step. If mission mode is required and there is no execution context, no tools are exposed.
- `DefaultToolGuard` rejects execution when runtime state is invalid, the step requires approval, the step authority exceeds the job max authority, the tool is outside job or step `allowed_tools`, campaign readiness is missing, governed external identity mode is wrong, or autonomy eligibility checks fail.
- The code does not assign a static authority tier to each tool. The "Recommended minimum authority" column below is operator guidance for mission authors, not an automatic enforcement rule.

## Registered Local Tools

| Tool | Registered | Recommended minimum authority | External side effect | Durable write behavior |
| --- | --- | --- | --- | --- |
| `message` | Always | Low | Sends one outbound message to the current channel/chat. | No direct file write; mission runtime budget/accounting may be updated by the loop after owner-facing messages. |
| `frank_zoho_send_email` | Always | High | Sends email through Zoho Mail and may fetch provider mailbox proof, replies, or bounce evidence during campaign reconciliation. | Updates campaign runtime state through `TaskState` when run inside `AgentLoop`: prepared/sent/failed outbound actions, send receipts, and reply-work transitions. Durable persistence depends on the mission runtime persistence hook. |
| `frank_zoho_manage_reply_work_item` | Always | Medium | No provider send; mutates local campaign reply-work state for a committed inbound reply. | Updates mission runtime reply-work items through `TaskState`; durable persistence depends on the mission runtime persistence hook. |
| `filesystem` | Always | Medium | No network or process launch. | `read`, `list`, and `stat` are read-only. `write` creates parent directories and writes files under the workspace-rooted `os.Root`; writes to `projects/current` require `frank_new_project` first. |
| `exec` | Always | High | Runs local processes and native `frank_*` helpers; `frank_sshd` can start or stop `sshd`. | Called programs can mutate the workspace. Native `frank_new_project` creates or archives `projects/current`; `frank_finish` helpers may move stray workspace files into `projects/current`. Shell interpreters, string commands, `python -c`, dangerous program names, absolute paths, `~`, and `..` args are rejected. |
| `web` | Always | Low | Performs an HTTP GET for a supplied URL. | No durable writes. |
| `web_search` | Always | Low | Performs an HTTP GET against DuckDuckGo Instant Answer API. | No durable writes. |
| `cron` | Only when `AgentLoop` has a scheduler | Medium | Schedules, lists, or cancels in-process reminders/tasks. Fired gateway reminders re-enter the agent through governed mission routing. | The tool mutates the in-memory scheduler. Gateway deferred-trigger handling can persist deferred trigger records under the mission store when a mission store root is configured. |
| `write_memory` | Always | Medium | No network. | Appends or overwrites workspace memory files for today's note or `MEMORY.md`; heartbeat/status noise is rejected. |
| `list_memory` | Always | Low | No network. | Read-only list of workspace memory files. |
| `read_memory` | Always | Low | No network. | Read-only access to today's note, `MEMORY.md`, or a dated daily note. |
| `edit_memory` | Always | Medium | No network. | Rewrites a memory file by exact find/replace; heartbeat/status replacement content is skipped. |
| `delete_memory` | Always | Medium | No network. | Deletes a dated daily memory file. `MEMORY.md` is protected from this tool. |
| `create_skill` | Always | High | No network. | Creates `skills/<name>/SKILL.md` under the workspace with prompt-only frontmatter. |
| `list_skills` | Always | Low | No network. | Read-only listing of skills under `skills/`. |
| `read_skill` | Always | Low | No network. | Read-only access to `skills/<name>/SKILL.md`. |
| `delete_skill` | Always | High | No network. | Removes `skills/<name>` recursively under the workspace-rooted `os.Root`. |

## Dynamic And Dormant Tools

| Tool family | Registered | Recommended minimum authority | Notes |
| --- | --- | --- | --- |
| `mcp_<server>_<tool>` | When configured MCP servers connect successfully | Depends on server tool side effects | The local registry only wraps the remote MCP schema and call. Mission authors must document and allow each MCP tool by exact generated name. Treat unknown MCP write/network capabilities as High until proven otherwise. |
| `spawn` | Not registered by `NewAgentLoop` | Medium if enabled later | The current implementation is a stub acknowledgement. Registering it would widen the model-visible tool surface and should come with tests and an updated manifest row. |

For MCP configuration and authorization details, see [CONFIG.md](../CONFIG.md#mcpservers).

## Update Rule

When adding, removing, or renaming a built-in tool:

1. Update the `NewAgentLoop` registration block or MCP naming rule.
2. Update this manifest with side effects, durable write behavior, and a recommended minimum authority.
3. Add or update tests for registry exposure, mission `allowed_tools`, and any durable writes.
