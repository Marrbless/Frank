# Android Phone Deployment

This guide is for running Frank/Picobot on an Android phone as a long-running agent appliance.

Target device for v1.0: Samsung Galaxy S21 Ultra or another ARM64 Android device.

## Security Model

Recommended baseline:

- no SIM in the agent phone,
- Tailscale installed on the agent phone,
- Android always-on VPN enabled for Tailscale,
- "block connections without VPN" enabled when available,
- a controlled exit node or firewall for internet egress,
- private GitHub access through `gh` or SSH,
- Telegram, Slack, WhatsApp, or another channel restricted to an owner allowlist,
- mission-controlled gateway for unattended operation.

Channel setup now fails closed for production gateway startup: an enabled channel must have a non-empty allowlist, or the config must record an explicit open-mode acknowledgement such as `openMode: true`. Use open mode only for deliberate public/testing deployments.

Do not paste provider API keys, Telegram tokens, SSH private keys, or GitHub tokens into shared logs, screenshots, issues, or chat transcripts.

## Install Termux

Install Termux from F-Droid or the official Termux GitHub releases. Do not use the abandoned Play Store build, and do not mix Termux app/add-on sources because Android package signatures differ.

Optional:

- Termux:Boot, from the same source as Termux, if you want automatic startup after reboot.

On the phone, disable battery restrictions for:

- Termux,
- Tailscale,
- any channel app used for control.

For real 24/7 operation, keep the phone plugged in and keep the device cool.

## Install Dependencies

In Termux:

```sh
pkg update
pkg upgrade
pkg install git gh golang nano tmux
```

Authenticate GitHub access to the private Frank repo:

```sh
gh auth login
gh auth status
```

## Clone

```sh
gh repo clone Marrbless/Frank ~/Frank
cd ~/Frank
```

If you use SSH instead:

```sh
git clone git@github.com:Marrbless/Frank.git ~/Frank
cd ~/Frank
```

## Build

Preferred on-phone build:

```sh
go build -tags lite -ldflags="-s -w" -o picobot ./cmd/picobot
./picobot version
```

The lite build omits WhatsApp support and is the recommended first phone build. Use the full build only if you specifically need WhatsApp on the agent phone:

```sh
go build -ldflags="-s -w" -o picobot ./cmd/picobot
```

## Onboard

```sh
./picobot onboard
nano ~/.picobot/config.json
```

Set at least:

- `providers.openai.apiKey`
- `providers.openai.apiBase`
- `agents.defaults.model`
- one channel token and owner allowlist, usually Telegram first

Test a one-shot call:

```sh
./picobot agent -m "hello from the phone"
```

## Start Gateway

Basic gateway:

```sh
tmux new -s frank
./picobot gateway
```

Detach from tmux with `Ctrl-b`, then `d`.

Resume:

```sh
tmux attach -t frank
```

## Mission-Control Gateway

Mission-control runtime settings are CLI flags, not `config.json`.

Use a valid mission file and explicit durable paths:

```sh
mkdir -p ~/.picobot/frank

./picobot gateway \
  --mission-required \
  --mission-file ~/.picobot/frank/mission.json \
  --mission-step discussion \
  --mission-status-file ~/.picobot/frank/mission-status.json \
  --mission-step-control-file ~/.picobot/frank/mission-step-control.json \
  --mission-store-root ~/.picobot/frank/mission-store
```

If `mission.json` does not exist yet, start with the basic gateway and create a minimal governed mission before enabling `--mission-required`.

Inspect status:

```sh
./picobot mission status --status-file ~/.picobot/frank/mission-status.json
```

## Autostart

After the manual gateway path is proven stable, use Termux:Boot.

Create:

```sh
mkdir -p ~/.termux/boot
nano ~/.termux/boot/start-frank
```

Example script:

```sh picobot-check:shell-syntax
#!/data/data/com.termux/files/usr/bin/sh
cd "$HOME/Frank" || exit 1
tmux new-session -d -s frank './picobot gateway'
```

Then:

```sh
chmod +x ~/.termux/boot/start-frank
```

For mission-control mode, replace the gateway command with the explicit `--mission-*` command after the mission file and store paths are stable.

## Maintenance

Update from the private repo and restart the bot process without rebooting Android:

```sh
cd ~/Frank
scripts/termux/update-and-restart-frank
```

The updater is transactional:

1. Optionally runs `git pull --ff-only`.
2. Builds a side-by-side candidate binary.
3. Runs a candidate smoke check with `picobot version`.
4. Preserves the previous binary and gateway command under `.termux-frank-backup/`.
5. Switches the binary only after build and smoke checks pass.
6. Restarts only the configured tmux session.
7. Verifies the tmux session is alive before declaring success.
8. Rolls back to the preserved binary and command if restart or health checks fail.

For mission-control mode, create a local phone-only environment file so the restart script always uses the same gateway flags:

```sh
cd ~/Frank
nano .termux-frank.env
```

Example:

```sh picobot-check:shell-syntax
PICOBOT_SESSION=frank
PICOBOT_GATEWAY_CMD='./picobot gateway --mission-required --mission-file ~/.picobot/frank/mission.json --mission-step discussion --mission-status-file ~/.picobot/frank/mission-status.json --mission-step-control-file ~/.picobot/frank/mission-step-control.json --mission-store-root ~/.picobot/frank/mission-store'
PICOBOT_MISSION_STATUS_FILE=~/.picobot/frank/mission-status.json
PICOBOT_MISSION_ASSERT_ARGS='--active'
PICOBOT_TRANSCRIPT_LOG=.termux-frank-backup/update.log
```

If you already ran `git pull --ff-only` and only need to rebuild/restart:

```sh
PICOBOT_SKIP_PULL=1 scripts/termux/update-and-restart-frank
```

Preview command order without changing the binary or tmux session:

```sh
scripts/termux/update-and-restart-frank --dry-run
```

Rollback to the last preserved binary and gateway command:

```sh
scripts/termux/update-and-restart-frank --rollback
```

Rollback is also available with:

```sh
PICOBOT_ROLLBACK=1 scripts/termux/update-and-restart-frank
```

Post-update success criteria:

- command exits `0`
- output ends with `update complete`
- `.termux-frank-backup/update.log` records the build, switch, restart, health, and rollback stages without dumping environment variables
- `tmux has-session -t frank` succeeds, or succeeds for your configured `PICOBOT_SESSION`
- if `PICOBOT_MISSION_STATUS_FILE` is set and exists, `picobot mission status` and `picobot mission assert` pass with `PICOBOT_MISSION_ASSERT_ARGS`

Rollback triggers:

- build failure
- candidate smoke-check failure
- tmux restart failure
- post-start tmux health failure
- mission status/assert failure when a status file is configured

Expected transcript shapes:

The snippets below are examples of the operator-visible shape. They are not a
claim of live phone proof; real proof should include the exact device command,
exit code, and `.termux-frank-backup/update.log` excerpt from that run. The
script masks the configured gateway command as `<gateway command>` in transcript
output so local secrets are not dumped.

Successful update:

```text
stage: start binary=picobot candidate=picobot.next.12345 backup=.termux-frank-backup/picobot.previous session=frank dry_run=0 rollback=0
stage: pull
+ git pull --ff-only
stage: build candidate=picobot.next.12345 tags=lite
+ go build -tags lite -ldflags=-s -w -o picobot.next.12345 ./cmd/picobot
+ chmod 755 picobot.next.12345
stage: smoke candidate=picobot.next.12345
stage: preserve previous=picobot backup=.termux-frank-backup/picobot.previous
+ mkdir -p .termux-frank-backup
+ write .termux-frank-backup/session
+ write .termux-frank-backup/gateway-command
+ cp -p picobot .termux-frank-backup/picobot.previous
stage: switch candidate=picobot.next.12345 binary=picobot
+ mv picobot.next.12345 picobot
stage: restart session=frank
+ tmux new-session -d -s frank <gateway command>
stage: health session=frank
+ tmux ls
frank: 1 windows
stage: complete binary=picobot backup=.termux-frank-backup/picobot.previous
update complete
```

Manual rollback:

```text
stage: rollback binary=picobot backup=.termux-frank-backup/picobot.previous session=frank
rolling back: manual rollback requested
+ cp -p .termux-frank-backup/picobot.previous picobot
+ tmux kill-session -t frank
+ tmux new-session -d -s frank <gateway command>
```

Restart failure with automatic rollback:

```text
stage: restart session=frank
+ tmux new-session -d -s frank <gateway command>
rolling back: restart failed
+ cp -p .termux-frank-backup/picobot.previous picobot
+ tmux new-session -d -s frank <gateway command>
```

If the script is not executable after cloning, run:

```sh
chmod +x scripts/termux/update-and-restart-frank
```

Back up `~/.picobot` regularly. It contains local config, memory, skills, mission state, and channel setup.

`~/.picobot/workspace/HEARTBEAT.md` should contain only real scheduled tasks under `## Periodic Tasks`. The default template is ignored by current releases, but keeping the section empty is still the lowest-cost idle mode.
