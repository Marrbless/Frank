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
- Telegram, Slack, or another channel restricted to an owner allowlist,
- mission-controlled gateway for unattended operation.

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

```sh
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

Update from the private repo:

```sh
cd ~/Frank
git pull --ff-only
go build -tags lite -ldflags="-s -w" -o picobot ./cmd/picobot
./picobot version
```

Back up `~/.picobot` regularly. It contains local config, memory, skills, mission state, and channel setup.

`~/.picobot/workspace/HEARTBEAT.md` should contain only real scheduled tasks under `## Periodic Tasks`. The default template is ignored by current releases, but keeping the section empty is still the lowest-cost idle mode.
