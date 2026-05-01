# Configuration Reference

Picobot is configured via `~/.picobot/config.json`. Run `picobot onboard` to generate the default config.

`config.json` stores provider credentials and channel tokens in plaintext. Keep it local, restrict filesystem access appropriately, and do not paste live secrets into logs, screenshots, issue reports, or chat transcripts.

## Full Default Config

```json
{
  "agents": {
    "defaults": {
      "workspace": "~/.picobot/workspace",
      "model": "stub-model",
      "maxTokens": 8192,
      "temperature": 0.7,
      "maxToolIterations": 100,
      "heartbeatIntervalS": 60,
      "requestTimeoutS": 60,
      "enableToolActivityIndicator": true
    }
  },
  "mcpServers": {},
  "channels": {
    "telegram": {
      "enabled": false,
      "token": "",
      "allowFrom": []
    },
    "discord": {
      "enabled": false,
      "token": "",
      "allowFrom": []
    },
    "slack": {
      "enabled": false,
      "appToken": "",
      "botToken": "",
      "allowUsers": [],
      "allowChannels": []
    },
    "whatsapp": {
      "enabled": false,
      "dbPath": "",
      "allowFrom": []
    }
  },
  "providers": {
    "openai": {
      "apiKey": "REPLACE_WITH_REAL_API_KEY",
      "apiBase": "https://openrouter.ai/api/v1"
    }
  }
}
```

## Frank Mission-Control Runtime

Frank mission-control startup and operator-control settings are currently CLI-only. They are not stored in `~/.picobot/config.json`.

Use the `gateway` and `mission` commands for the mission-control surface that is already implemented on current HEAD:

- `--mission-required` fails closed unless a governed mission step is active
- `--mission-file` and `--mission-step` bootstrap the active job and step at startup
- `--mission-status-file` writes the operator status snapshot JSON
- `--mission-step-control-file` enables file-watched step switching for `picobot mission set-step`
- `--mission-store-root` points at the durable mission store
- `--mission-resume-approved` is required before resuming a persisted non-terminal runtime after reboot

If `--mission-store-root` is omitted but `--mission-status-file` is set, Picobot derives the durable store root as `<status-file>.store`.

The durable mission store is the persisted Frank runtime surface. It holds the committed mission runtime, audit history, approval records, artifacts, current gateway log segment, and packaged log bundles used by `picobot mission package-logs` and `picobot mission prune-store`.

---

## agents.defaults

Agent behavior settings.

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `workspace` | string | `~/.picobot/workspace` | Path to the agent's workspace directory. Contains bootstrap files, memory, and skills. |
| `model` | string | `stub-model` | Default LLM model to use. Set to a real model like `google/gemini-2.5-flash`. Can be overridden with the `-M` flag. |
| `maxTokens` | int | `8192` | Maximum tokens for LLM responses. |
| `temperature` | float | `0.7` | LLM temperature (0.0 = deterministic, 1.0 = creative). |
| `maxToolIterations` | int | `100` | Maximum number of tool-calling iterations per request. Prevents infinite loops. |
| `heartbeatIntervalS` | int | `60` | How often (in seconds) the heartbeat checks `HEARTBEAT.md` for periodic tasks. Only used in gateway mode. |
| `requestTimeoutS` | int | `60` | HTTP timeout in seconds for each LLM API request. Increase for slow models or poor network conditions. |
| `enableToolActivityIndicator` | bool | `true` | When `true`, sends interim `🤖 Running` / `📢 done` messages to the chat channel as tools are called. Set to `false` for IoT or headless deployments where only the final response should be delivered. |

### Model Priority

The model is resolved in this order:
1. **CLI flag** (`-M` / `--model`)
2. **Config** (`agents.defaults.model`)
3. **Provider default** (fallback)

### Example

```json
{
  "agents": {
    "defaults": {
      "workspace": "/home/user/.picobot/workspace",
      "model": "google/gemini-2.5-flash",
      "maxTokens": 16384,
      "temperature": 0.5,
      "maxToolIterations": 200,
      "heartbeatIntervalS": 120,
      "requestTimeoutS": 120,
      "enableToolActivityIndicator": false
    }
  }
}
```

---

## providers

LLM provider configuration. Picobot uses an OpenAI-compatible API provider.

### providers.openai

Connect to any OpenAI-compatible API service (OpenAI, OpenRouter, Ollama, etc.).

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `apiKey` | string | *(required)* | Your API key. Get OpenRouter keys at https://openrouter.ai/keys |
| `apiBase` | string | `https://openrouter.ai/api/v1` | API base URL. Use `https://api.openai.com/v1` for OpenAI, `http://localhost:11434/v1` for local Ollama, or any compatible endpoint. |

```json
{
  "providers": {
    "openai": {
      "apiKey": "<REPLACE_WITH_REAL_PROVIDER_API_KEY>",
      "apiBase": "https://openrouter.ai/api/v1"
    }
  }
}
```

**Examples:**

OpenAI:

```json
{
  "providers": {
    "openai": {
      "apiKey": "<REPLACE_WITH_OPENAI_API_KEY>",
      "apiBase": "https://api.openai.com/v1"
    }
  }
}
```

Local Ollama (no API key needed):

```json
{
  "providers": {
    "openai": {
      "apiKey": "not-needed",
      "apiBase": "http://localhost:11434/v1"
    }
  }
}
```

### Provider Fallback

If no valid provider is configured, Picobot uses a **Stub** provider (echoes back your message, for testing).

---

## Frank V5 Model Control Plane

Frank V5 adds a model registry and deterministic resolver while preserving the legacy `providers.openai` configuration.

Operator inspection commands:

```sh
picobot models list
picobot models inspect local_fast
picobot models route --model best --requires-tools
picobot models route --local
picobot models health
picobot models health local_fast
```

V5 does not download models, start Ollama, or start llama.cpp. Local runtimes remain external processes, usually managed in a separate Termux or tmux session. Runtime routing uses named providers and named model profiles; fallback is preflight-only and does not retry an agent turn after a provider request has started. See [FRANK_V5_MODEL_CONTROL_SPEC.md](FRANK_V5_MODEL_CONTROL_SPEC.md).

### Example V5 model registry

```json
{
  "agents": {
    "defaults": {
      "workspace": "~/.picobot/workspace",
      "model": "cloud_reasoning",
      "maxTokens": 8192,
      "temperature": 0.7,
      "maxToolIterations": 100,
      "heartbeatIntervalS": 60,
      "requestTimeoutS": 60,
      "enableToolActivityIndicator": true
    }
  },
  "providers": {
    "openai": {
      "type": "openai_compatible",
      "apiKey": "REPLACE_WITH_REAL_PROVIDER_API_KEY",
      "apiBase": "https://api.openai.com/v1"
    },
    "openrouter": {
      "type": "openai_compatible",
      "apiKey": "REPLACE_WITH_REAL_PROVIDER_API_KEY",
      "apiBase": "https://openrouter.ai/api/v1"
    },
    "llamacpp_phone": {
      "type": "openai_compatible",
      "apiKey": "not-needed",
      "apiBase": "http://127.0.0.1:8080/v1"
    },
    "ollama_phone": {
      "type": "openai_compatible",
      "apiKey": "ollama",
      "apiBase": "http://127.0.0.1:11434/v1"
    }
  },
  "models": {
    "local_fast": {
      "provider": "llamacpp_phone",
      "providerModel": "qwen3-1.7b-q8_0",
      "displayName": "Qwen3 1.7B phone local",
      "capabilities": {
        "local": true,
        "offline": true,
        "supportsTools": false,
        "supportsStreaming": false,
        "supportsResponsesAPI": false,
        "supportsVision": false,
        "supportsAudio": false,
        "contextTokens": 4096,
        "maxOutputTokens": 1024,
        "authorityTier": "low",
        "costTier": "free",
        "latencyTier": "slow"
      },
      "request": {
        "maxTokens": 1024,
        "temperature": 0.3,
        "timeoutS": 300,
        "useResponses": false,
        "reasoningEffort": ""
      }
    },
    "ollama_chat": {
      "provider": "ollama_phone",
      "providerModel": "qwen3:1.7b",
      "displayName": "Ollama phone chat",
      "capabilities": {
        "local": true,
        "offline": true,
        "supportsTools": false,
        "supportsStreaming": false,
        "supportsResponsesAPI": false,
        "supportsVision": false,
        "supportsAudio": false,
        "contextTokens": 4096,
        "maxOutputTokens": 1024,
        "authorityTier": "low",
        "costTier": "free",
        "latencyTier": "slow"
      },
      "request": {
        "maxTokens": 1024,
        "temperature": 0.3,
        "timeoutS": 300,
        "useResponses": false,
        "reasoningEffort": ""
      }
    },
    "cloud_reasoning": {
      "provider": "openrouter",
      "providerModel": "google/gemini-2.5-flash",
      "displayName": "Cloud reasoning default",
      "capabilities": {
        "local": false,
        "offline": false,
        "supportsTools": true,
        "supportsStreaming": false,
        "supportsResponsesAPI": false,
        "supportsVision": false,
        "supportsAudio": false,
        "contextTokens": 1000000,
        "maxOutputTokens": 8192,
        "authorityTier": "high",
        "costTier": "standard",
        "latencyTier": "normal"
      },
      "request": {
        "maxTokens": 8192,
        "temperature": 0.5,
        "timeoutS": 120,
        "useResponses": false,
        "reasoningEffort": ""
      }
    },
    "cloud_openai": {
      "provider": "openai",
      "providerModel": "gpt-4.1-mini",
      "displayName": "OpenAI cloud fallback",
      "capabilities": {
        "local": false,
        "offline": false,
        "supportsTools": true,
        "supportsStreaming": false,
        "supportsResponsesAPI": true,
        "supportsVision": false,
        "supportsAudio": false,
        "contextTokens": 128000,
        "maxOutputTokens": 8192,
        "authorityTier": "high",
        "costTier": "standard",
        "latencyTier": "normal"
      },
      "request": {
        "maxTokens": 8192,
        "temperature": 0.3,
        "timeoutS": 120,
        "useResponses": true,
        "reasoningEffort": ""
      }
    }
  },
  "modelAliases": {
    "default": "cloud_reasoning",
    "phone": "local_fast",
    "local": "local_fast",
    "ollama": "ollama_chat",
    "best": "cloud_reasoning",
    "openai": "cloud_openai"
  },
  "modelRouting": {
    "defaultModel": "cloud_reasoning",
    "localPreferredModel": "local_fast",
    "fallbacks": {
      "local_fast": ["cloud_reasoning"],
      "ollama_chat": ["cloud_reasoning"],
      "cloud_reasoning": [],
      "cloud_openai": []
    },
    "allowCloudFallbackFromLocal": false,
    "allowLowerAuthorityFallback": false
  },
  "localRuntimes": {
    "llamacpp_phone": {
      "kind": "external_http",
      "provider": "llamacpp_phone",
      "expectedBaseURL": "http://127.0.0.1:8080/v1",
      "startCommand": "",
      "healthURL": "http://127.0.0.1:8080/health",
      "notes": "Started separately in Termux/tmux."
    },
    "ollama_phone": {
      "kind": "external_http",
      "provider": "ollama_phone",
      "expectedBaseURL": "http://127.0.0.1:11434/v1",
      "startCommand": "",
      "healthURL": "http://127.0.0.1:11434/api/tags",
      "notes": "Started separately by Ollama."
    }
  }
}
```

Model and provider refs are normalized by trimming whitespace and lowercasing. They may contain only `a-z`, `0-9`, `_`, and `-`, and must not contain path separators. Alias chains are rejected. A local model profile defaults to low authority and no tool support if those fields are omitted.

Existing configs that omit `models` keep legacy behavior: `agents.defaults.model` remains the raw provider model sent to `providers.openai`, and `-M some-provider-model` still works as a raw provider model override on the legacy provider.

### V5 runtime notes

- `providerModel` is the string sent to the selected backend.
- `request.maxTokens`, `request.temperature`, `request.timeoutS`, `request.useResponses`, and `request.reasoningEffort` override the global defaults for that model profile.
- `supportsTools: false` causes zero tool schemas to be sent to the provider.
- Low and medium authority models receive only the tool schemas allowed by the current V5 authority gate; high authority remains bounded by mission `allowed_tools` and approvals.
- `localRuntimes.*.healthURL` is used for local preflight readiness checks. The status JSON may show cached `model_health` values, but status reads do not perform network health probes.
- Cloud fallback from a local model is denied unless `modelRouting.allowCloudFallbackFromLocal` or mission policy explicitly allows it.
- Runtime fallback is preflight-only by default. Frank does not automatically replay a turn after a provider request starts.

### Local runtime commands

Start llama.cpp separately, then point `llamacpp_phone` at its OpenAI-compatible server:

```sh
llama-server \
  -m "$HOME/models/qwen3-1.7b-q8_0.gguf" \
  --host 127.0.0.1 \
  --port 8080
```

Start Ollama separately, then point `ollama_phone` at `http://127.0.0.1:11434/v1`:

```sh
ollama serve
ollama pull qwen3:1.7b
```

After the model server is running:

```sh
picobot models list
picobot models inspect phone
picobot models route --model phone --local
picobot models health phone
```

---

## mcpServers

Connect external [MCP (Model Context Protocol)](https://modelcontextprotocol.io) servers to give the agent additional tools. Each entry is a named server that exposes one or more tools, which are registered automatically at startup under the name `mcp_{server}_{tool}`.

Two transports are supported:

| Transport | When to use | Required fields |
|-----------|-------------|------------------|
| **Stdio** | Local process (npx, uvx, binary, docker) | `command` + `args` |
| **HTTP** | Remote or hosted MCP server | `url` (+ optional `headers`) |

### Stdio transport (command + args)

Picobot spawns the process and communicates over stdin/stdout. This works with any MCP server that supports the stdio transport.

```json
{
  "mcpServers": {
    "via-npx": {
      "command": "npx",
      "args": ["-y", "@some/mcp-server"]
    }
  }
}
```

**Common patterns:**

```json
{
  "mcpServers": {
    "via-npx": {
      "command": "npx",
      "args": ["-y", "@some/mcp-server"]
    },
    "via-uvx": {
      "command": "uvx",
      "args": ["some-mcp-server"]
    },
    "via-binary": {
      "command": "/usr/local/bin/my-mcp-server",
      "args": ["--some-flag"]
    },
    "via-docker": {
      "command": "docker",
      "args": ["run", "--rm", "-i", "mcp/some-image"]
    }
  }
}
```

> **Docker note:** Always include `-i` (interactive) in the `args`. Without it, Docker closes stdin immediately and the MCP handshake fails.

### HTTP transport (url + headers)

For MCP servers accessible over HTTP (Streamable HTTP or SSE). Supports bearer tokens and custom headers.

```json
{
  "mcpServers": {
    "via-remote": {
      "url": "https://mcp.example.com/mcp",
      "headers": {
        "Authorization": "Bearer <REPLACE_WITH_MCP_TOKEN>"
      }
    }
  }
}
```

Keep real header values out of pasted logs and screenshots. If you need to share a config snippet, replace secrets with placeholders like the example above.

### MCPServerConfig fields

| Field | Type | Description |
|-------|------|-------------|
| `command` | string | Executable to spawn (for stdio transport). Can be a name on `$PATH` or an absolute path. |
| `args` | string[] | Arguments passed to the command. |
| `url` | string | HTTP endpoint for the MCP server (for HTTP transport). |
| `headers` | object | HTTP headers to attach to every request (e.g. `Authorization`). |

Only one transport is used per server: if both `command` and `url` are set, `command` takes precedence.

### Tool naming

Each MCP tool is registered in the agent's tool registry as `mcp_{server}_{tool}`. For example, a server named `via-npx` exposing a tool `some-action` becomes `mcp_via-npx_some-action`. The agent sees and calls it like any built-in tool.

### Authorization, side effects, and logging

MCP is disabled by default because the generated config starts with an empty `mcpServers` map. Adding a server expands the model-visible tool surface only after that server connects and returns a tool list.

MCP tools use the same mission guard as local tools:

- Mission `allowed_tools` must name each MCP tool exactly, for example `mcp_via-npx_some-action`.
- Step-level `allowed_tools` further narrow the job-level list when present.
- Step approval, runtime validity, campaign readiness, governed identity mode, and autonomy eligibility checks still run before tool execution.

Picobot does not inspect a remote MCP server deeply enough to prove whether a tool is read-only, writes durable state, calls a network, or performs account actions. Treat any MCP tool with unknown side effects as high authority until the server's own documentation and local testing prove a narrower scope. Record approved MCP tools in [maintenance/TOOL_PERMISSION_MANIFEST.md](maintenance/TOOL_PERMISSION_MANIFEST.md) or a mission-specific permission note before using them in unattended work.

MCP logging is intentionally summarized on the local side. Startup failures are logged and the agent continues with other tools. Runtime tool logs use argument counts/types rather than raw argument keys or values, and MCP tool failures are surfaced as summarized MCP failures instead of raw remote payloads when they pass through the local registry.

### Startup behaviour

- Servers are connected when the agent starts (`gateway` or `agent` command).
- If a server fails to connect (process not found, network error, handshake failure), picobot **logs the error and continues** — other servers and built-in tools are unaffected.
- All MCP connections are cleanly shut down when the gateway exits.

---

## channels

Chat channel integrations. Supports Telegram, Discord, Slack, and WhatsApp.

Validate channel safety without starting network services:

```sh
picobot config validate
```

The command fails if any enabled channel has an empty allowlist without an explicit open-mode acknowledgement.
On POSIX systems, it also warns when `~/.picobot/config.json` contains plaintext credentials and is readable by group or other users. New configs are written with owner-only permissions.

Edit allowlists without manually editing JSON:

```sh
picobot channels allowlist list telegram
picobot channels allowlist add telegram 8881234567
picobot channels allowlist remove telegram 8881234567
picobot channels allowlist add slack-users U0123456789
picobot channels allowlist add slack-channels C0123456789
```

Valid allowlist scopes are `telegram`, `discord`, `whatsapp`, `slack-users`, and `slack-channels`. These commands do not disable open mode; if open mode is enabled, Picobot warns that allowlist edits will not restrict access until open mode is disabled.

### channels.telegram

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `enabled` | bool | `false` | Set to `true` to start the Telegram bot. |
| `token` | string | `""` | Your Telegram Bot token from [@BotFather](https://t.me/BotFather). |
| `allowFrom` | string[] | `[]` | List of allowed Telegram user IDs. Gateway startup fails closed when this is empty unless `openMode` is explicitly true. |
| `openMode` | bool | `false` | Explicit acknowledgement that every Telegram user may message the bot. Prefer `allowFrom` for unattended deployments. |

```json
{
  "channels": {
    "telegram": {
      "enabled": true,
      "token": "<REPLACE_WITH_TELEGRAM_BOT_TOKEN>",
      "allowFrom": ["8881234567"],
      "openMode": false
    }
  }
}
```

### channels.discord

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `enabled` | bool | `false` | Set to `true` to start the Discord bot. |
| `token` | string | `""` | Your Discord Bot token from the [Developer Portal](https://discord.com/developers/applications). |
| `allowFrom` | string[] | `[]` | List of allowed Discord user IDs. Gateway startup fails closed when this is empty unless `openMode` is explicitly true. |
| `openMode` | bool | `false` | Explicit acknowledgement that every Discord user may message the bot. Prefer `allowFrom` for unattended deployments. |

```json
{
  "channels": {
    "discord": {
      "enabled": true,
      "token": "<REPLACE_WITH_DISCORD_BOT_TOKEN>",
      "allowFrom": ["123456789012345678"],
      "openMode": false
    }
  }
}
```

The Discord bot uses the Gateway WebSocket API for receiving messages and the REST API for sending. In servers, the bot responds when **mentioned** (`@botname`) or when a message is a **reply** to the bot. In DMs, the bot responds to all messages.

**Required Bot Permissions:**
- Send Messages
- Read Message History

**Required Privileged Intents (enable in Developer Portal → Bot):**
- Message Content Intent

### channels.slack

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `enabled` | bool | `false` | Set to `true` to start the Slack bot. |
| `appToken` | string | `""` | Slack App-Level Token (Socket Mode), starts with `xapp-`. |
| `botToken` | string | `""` | Slack Bot Token, starts with `xoxb-`. |
| `allowUsers` | string[] | `[]` | List of allowed Slack user IDs. Gateway startup fails closed when this is empty unless `openUserMode` is explicitly true. |
| `allowChannels` | string[] | `[]` | List of allowed Slack channel IDs (C..., G..., D...). Gateway startup fails closed when this is empty unless `openChannelMode` is explicitly true. DMs ignore this list after startup. |
| `openUserMode` | bool | `false` | Explicit acknowledgement that every Slack user may message the bot. |
| `openChannelMode` | bool | `false` | Explicit acknowledgement that every Slack channel may reach the bot when mentioned. |

```json
{
  "channels": {
    "slack": {
      "enabled": true,
      "appToken": "<REPLACE_WITH_SLACK_APP_TOKEN>",
      "botToken": "<REPLACE_WITH_SLACK_BOT_TOKEN>",
      "allowUsers": ["U0123456789"],
      "allowChannels": ["C0123456789"],
      "openUserMode": false,
      "openChannelMode": false
    }
  }
}
```

The Slack bot uses Socket Mode. In channels, the bot responds only when mentioned. In DMs, the bot responds to all messages from allowed users and ignores `allowChannels`. Thread replies are preserved when the inbound message is in a thread.

### channels.whatsapp

Uses a personal WhatsApp account (via [whatsmeow](https://go.mau.fi/whatsmeow)) rather than a dedicated bot account. Only direct messages are handled — group messages are ignored.

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `enabled` | bool | `false` | Set to `true` to start the WhatsApp channel. |
| `dbPath` | string | `~/.picobot/whatsapp.db` | Path to the SQLite session database. Created automatically by `picobot channels login`. |
| `allowFrom` | string[] | `[]` | List of **LID numbers** allowed to send messages. Gateway startup fails closed when this is empty unless `openMode` is explicitly true. See below. |
| `openMode` | bool | `false` | Explicit acknowledgement that every WhatsApp sender may message the bot. Prefer `allowFrom` for unattended deployments. |

```json
{
  "channels": {
    "whatsapp": {
      "enabled": true,
      "dbPath": "~/.picobot/whatsapp.db",
      "allowFrom": ["12345678901234"],
      "openMode": false
    }
  }
}
```

**One-time setup:** Link your phone by running:
```
picobot channels login
```
Select **4) WhatsApp**. The setup asks for allowed sender IDs or an explicit `OPEN` acknowledgement, then shows a QR code. In WhatsApp on your phone: **Settings → Linked Devices → Link a Device**. The session is saved to `dbPath` — no QR code is needed on subsequent starts. The config is updated automatically.

#### Finding your LID for allowFrom

Modern WhatsApp accounts use an internal **LID** (Linked ID) — a numeric identifier that is different from the phone number. Picobot routes messages using LIDs, so `allowFrom` must contain LID numbers, not phone numbers.

**How to find your LID:**

Start the gateway after pairing and check the startup log:

```
whatsapp: connected as 85298765432 (LID: 12345678901234)
```

The number after `LID:` is this device's own LID. To find the LID of another person you want to allow, ask them to send you a message, then check the picobot log:

```
whatsapp: dropped message from unauthorized sender 99999999999@lid (add '99999999999' to allowFrom to permit)
```

The number in the log is the sender's LID. Add that number to `allowFrom`.

**Examples:**

| Scenario | `allowFrom` value |
|----------|-------------------|
| Allow only yourself (Notes to Self) | `[]` *(self-chat is always allowed regardless)* |
| Allow one other person | `["12345678901234"]` |
| Allow multiple people | `["12345678901234", "99999999999"]` |
| Allow everyone | `[]` plus `"openMode": true` |

> **Why not phone numbers?** Newer WhatsApp accounts use LID-based addressing internally. If you put a phone number in `allowFrom`, messages from that person will be silently dropped because WhatsApp delivers them with a LID, not the phone number.

> **Self-chat (Notes to Self):** Your own messages to yourself always bypass the `allowFrom` list — no entry needed.

> **Note:** Unlike Telegram/Discord bots, WhatsApp uses a personal phone number. Messages are sent and received from that number.

---

## Docker Environment Variables

When running with Docker, Picobot reads `~/.picobot/config.json` and then the container entrypoint applies a small set of environment overrides. There is no separate Docker-only remapping layer.

Docker channel overrides preserve the same fail-closed startup checks as normal config. If a token environment variable enables a channel, provide the matching allowlist environment variables or mount a config file that explicitly sets the relevant open-mode field. There is no Docker environment shortcut for `openMode`, `openUserMode`, or `openChannelMode`.

| Environment Variable | Config Path | Description |
|---------------------|-------------|-------------|
| `OPENAI_API_KEY` | `providers.openai.apiKey` | OpenAI-compatible API key |
| `OPENAI_API_BASE` | `providers.openai.apiBase` | OpenAI-compatible API base URL |
| `PICOBOT_MODEL` | `agents.defaults.model` | LLM model to use |
| `PICOBOT_MAX_TOKENS` | `agents.defaults.maxTokens` | Maximum tokens for LLM responses |
| `PICOBOT_MAX_TOOL_ITERATIONS` | `agents.defaults.maxToolIterations` | Maximum tool iterations per request |
| `PICOBOT_ENABLE_TOOL_ACTIVITY_INDICATOR` | `agents.defaults.enableToolActivityIndicator` | Show or hide tool activity progress messages |
| `TELEGRAM_BOT_TOKEN` | `channels.telegram.enabled`, `channels.telegram.token` | Enables Telegram and sets the bot token |
| `TELEGRAM_ALLOW_FROM` | `channels.telegram.allowFrom` | Comma-separated Telegram user IDs. Required with `TELEGRAM_BOT_TOKEN` unless config sets `channels.telegram.openMode=true` |
| `DISCORD_BOT_TOKEN` | `channels.discord.enabled`, `channels.discord.token` | Enables Discord and sets the bot token |
| `DISCORD_ALLOW_FROM` | `channels.discord.allowFrom` | Comma-separated Discord user IDs. Required with `DISCORD_BOT_TOKEN` unless config sets `channels.discord.openMode=true` |
| `SLACK_APP_TOKEN` | `channels.slack.enabled`, `channels.slack.appToken` | Enables Slack and sets the app-level token |
| `SLACK_BOT_TOKEN` | `channels.slack.enabled`, `channels.slack.botToken` | Enables Slack and sets the bot token |
| `SLACK_ALLOW_USERS` | `channels.slack.allowUsers` | Comma-separated Slack user IDs. Required when Slack is enabled unless config sets `channels.slack.openUserMode=true` |
| `SLACK_ALLOW_CHANNELS` | `channels.slack.allowChannels` | Comma-separated Slack channel IDs. Required when Slack is enabled unless config sets `channels.slack.openChannelMode=true` |

Provider credentials and channel tokens may also be set in `~/.picobot/config.json` or through the relevant interactive onboarding/login flows. `picobot channels login` hides token entry on supported terminals.

---

## Workspace Files

The workspace directory (default `~/.picobot/workspace`) contains files that shape agent behavior:

| File | Purpose | Who edits |
|------|---------|-----------|
| `SOUL.md` | Agent personality, values, communication style | You (once) |
| `AGENTS.md` | Agent instructions, rules, guidelines | You (once) |
| `USER.md` | Your profile — name, timezone, preferences | You (once) |
| `TOOLS.md` | Tool reference documentation | You (once) |
| `HEARTBEAT.md` | Explicit periodic tasks checked every `heartbeatIntervalS` seconds | You / Agent |
| `memory/MEMORY.md` | Long-term memory | Agent (via write_memory tool) |
| `memory/YYYY-MM-DD.md` | Daily notes | Agent (via write_memory tool) |
| `skills/` | Skill packages | Agent (via skill tools) or you manually |

`HEARTBEAT.md` is task-driven. The gateway reads explicit task lines from the `## Periodic Tasks` section and ignores comments, headings, and the default onboarding template. Leave the section empty when no scheduled work should run.

---

## Example: Minimal Production Config

```json
{
  "agents": {
    "defaults": {
      "workspace": "/home/user/.picobot/workspace",
      "model": "openrouter/free",
      "maxTokens": 8192,
      "temperature": 0.7,
      "maxToolIterations": 200,
      "heartbeatIntervalS": 60
    }
  },
  "mcpServers": {
    "via-npx": {
      "command": "npx",
      "args": ["-y", "@some/mcp-server"]
    }
  },
  "channels": {
    "telegram": {
      "enabled": true,
      "token": "<REPLACE_WITH_TELEGRAM_BOT_TOKEN>",
      "allowFrom": ["YOUR_TELEGRAM_USER_ID"]
    },
    "discord": {
      "enabled": true,
      "token": "<REPLACE_WITH_DISCORD_BOT_TOKEN>",
      "allowFrom": ["YOUR_DISCORD_USER_ID"]
    }
  },
  "providers": {
    "openai": {
      "apiKey": "<REPLACE_WITH_REAL_PROVIDER_API_KEY>",
      "apiBase": "https://openrouter.ai/api/v1"
    }
  }
}
```
