package main

import (
	"fmt"

	"github.com/local/picobot/internal/config"
)

func requireAllowlistOrOpen(surface string, allowed []string, openMode bool) error {
	if len(allowed) > 0 || openMode {
		return nil
	}
	return fmt.Errorf("%s allowlist is empty; configure allowed IDs or set explicit open mode", surface)
}

func validateConfigForStartup(cfg config.Config) []error {
	var errs []error
	if cfg.Channels.Telegram.Enabled {
		if err := requireAllowlistOrOpen("telegram", cfg.Channels.Telegram.AllowFrom, cfg.Channels.Telegram.OpenMode); err != nil {
			errs = append(errs, err)
		}
	}
	if cfg.Channels.Discord.Enabled {
		if err := requireAllowlistOrOpen("discord", cfg.Channels.Discord.AllowFrom, cfg.Channels.Discord.OpenMode); err != nil {
			errs = append(errs, err)
		}
	}
	if cfg.Channels.Slack.Enabled {
		if err := requireAllowlistOrOpen("slack users", cfg.Channels.Slack.AllowUsers, cfg.Channels.Slack.OpenUserMode); err != nil {
			errs = append(errs, err)
		}
		if err := requireAllowlistOrOpen("slack channels", cfg.Channels.Slack.AllowChannels, cfg.Channels.Slack.OpenChannelMode); err != nil {
			errs = append(errs, err)
		}
	}
	if cfg.Channels.WhatsApp.Enabled {
		if err := requireAllowlistOrOpen("whatsapp", cfg.Channels.WhatsApp.AllowFrom, cfg.Channels.WhatsApp.OpenMode); err != nil {
			errs = append(errs, err)
		}
	}
	return errs
}
