# Docker Deployment

Run Picobot as a Docker container — one command to start.

## Quick Start

### Option 1: Docker Compose (Recommended)

```sh
# 1. Create .env with your API key and settings
nano docker/.env

# 2. Start
docker compose -f docker/docker-compose.yml up -d

# 3. Check logs
docker compose -f docker/docker-compose.yml logs -f
```

### Option 2: Docker Run

```sh
# Build the image
docker build -f docker/Dockerfile -t picobot .

# Run with environment variables
docker run -d \
  --name picobot \
  --restart unless-stopped \
  -e OPENAI_API_KEY="sk-or-v1-YOUR_KEY" \
  -e OPENAI_API_BASE="https://openrouter.ai/api/v1" \
  -e PICOBOT_MODEL="openrouter/free" \
  -e PICOBOT_MAX_TOKENS=8192 \
  -e PICOBOT_MAX_TOOL_ITERATIONS=100 \
  -e PICOBOT_ENABLE_TOOL_ACTIVITY_INDICATOR=true \
  -e TELEGRAM_BOT_TOKEN="123456:ABC..." \
  -e TELEGRAM_ALLOW_FROM="8881234567" \
  -e DISCORD_BOT_TOKEN="MTIzNDU2..." \
  -e DISCORD_ALLOW_FROM="123456789012345678" \
  -e SLACK_APP_TOKEN="xapp-1-..." \
  -e SLACK_BOT_TOKEN="xoxb-..." \
  -e SLACK_ALLOW_USERS="U0123456789" \
  -e SLACK_ALLOW_CHANNELS="C0123456789" \
  -v ./picobot-data:/home/picobot/.picobot \
  picobot
```

## Environment Variables

### Channel Access Policy

Docker environment overrides preserve Picobot's fail-closed channel policy. Setting
a channel token enables that channel, but gateway startup still fails until you
also provide allowlist IDs or mount a config file that explicitly sets the
matching open-mode field:

- Telegram: `TELEGRAM_ALLOW_FROM` or `channels.telegram.openMode=true`
- Discord: `DISCORD_ALLOW_FROM` or `channels.discord.openMode=true`
- Slack: `SLACK_ALLOW_USERS` and `SLACK_ALLOW_CHANNELS`, or
  `channels.slack.openUserMode=true` and `channels.slack.openChannelMode=true`

There is no Docker environment shortcut for open mode. Prefer allowlists for
unattended containers.

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `OPENAI_API_KEY` | Yes | — | OpenAI-compatible API key (OpenRouter, OpenAI, etc.) |
| `OPENAI_API_BASE` | No | `https://openrouter.ai/api/v1` | OpenAI-compatible API base URL |
| `PICOBOT_MODEL` | No | `google/gemini-2.5-flash` | LLM model to use |
| `PICOBOT_MAX_TOKENS` | No | `8192` | Maximum tokens for LLM responses |
| `PICOBOT_MAX_TOOL_ITERATIONS` | No | `100` | Maximum tool iterations per request |
| `PICOBOT_ENABLE_TOOL_ACTIVITY_INDICATOR` | No | `true` | Send `🤖 Running` / `📢 done` progress messages during tool calls. Set to `false` for IoT or headless deployments |
| `TELEGRAM_BOT_TOKEN` | No | — | Telegram bot token from @BotFather |
| `TELEGRAM_ALLOW_FROM` | If Telegram token is set | — | Comma-separated Telegram user IDs. Required unless mounted config sets `channels.telegram.openMode=true` |
| `DISCORD_BOT_TOKEN` | No | — | Discord bot token from Developer Portal |
| `DISCORD_ALLOW_FROM` | If Discord token is set | — | Comma-separated Discord user IDs. Required unless mounted config sets `channels.discord.openMode=true` |
| `SLACK_APP_TOKEN` | No | — | Slack App-Level Token (`xapp-...`), also enables the channel |
| `SLACK_BOT_TOKEN` | No | — | Slack Bot Token (`xoxb-...`), also enables the channel |
| `SLACK_ALLOW_USERS` | If Slack tokens are set | — | Comma-separated Slack user IDs allowed to chat. Required unless mounted config sets `channels.slack.openUserMode=true` |
| `SLACK_ALLOW_CHANNELS` | If Slack tokens are set | — | Comma-separated Slack channel IDs allowed. Required unless mounted config sets `channels.slack.openChannelMode=true`; DMs ignore this list after startup |

## Data Persistence

All data is stored in the `picobot-data` Docker volume:
- `config.json` — configuration
- `workspace/` — bootstrap files, memory, skills

Data persists across container restarts and rebuilds.
