package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/local/picobot/internal/config"
)

func newChannelsAllowlistCmd() *cobra.Command {
	allowlistCmd := &cobra.Command{
		Use:   "allowlist",
		Short: "List, add, or remove channel allowlist entries",
	}

	listCmd := &cobra.Command{
		Use:   "list <scope>",
		Short: "List allowlist entries for a channel scope",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, _, err := loadChannelAllowlistConfig()
			if err != nil {
				return err
			}
			entries, scope, err := channelAllowlistEntries(&cfg, args[0])
			if err != nil {
				return err
			}
			for _, entry := range *entries {
				fmt.Fprintln(cmd.OutOrStdout(), entry)
			}
			warnOpenModeForAllowlistScope(cmd, scope)
			return nil
		},
	}

	addCmd := &cobra.Command{
		Use:   "add <scope> <id> [id...]",
		Short: "Add IDs to a channel allowlist",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, cfgPath, err := loadChannelAllowlistConfig()
			if err != nil {
				return err
			}
			entries, scope, err := channelAllowlistEntries(&cfg, args[0])
			if err != nil {
				return err
			}
			*entries = addAllowlistEntries(*entries, args[1:])
			if err := config.SaveConfig(cfg, cfgPath); err != nil {
				return fmt.Errorf("failed to save config: %w", err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%s allowlist now has %d entr%s\n", scope.name, len(*entries), pluralY(len(*entries)))
			warnOpenModeForAllowlistScope(cmd, scope)
			return nil
		},
	}

	removeCmd := &cobra.Command{
		Use:   "remove <scope> <id> [id...]",
		Short: "Remove IDs from a channel allowlist",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, cfgPath, err := loadChannelAllowlistConfig()
			if err != nil {
				return err
			}
			entries, scope, err := channelAllowlistEntries(&cfg, args[0])
			if err != nil {
				return err
			}
			*entries = removeAllowlistEntries(*entries, args[1:])
			if err := config.SaveConfig(cfg, cfgPath); err != nil {
				return fmt.Errorf("failed to save config: %w", err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%s allowlist now has %d entr%s\n", scope.name, len(*entries), pluralY(len(*entries)))
			warnOpenModeForAllowlistScope(cmd, scope)
			return nil
		},
	}

	allowlistCmd.AddCommand(listCmd, addCmd, removeCmd)
	return allowlistCmd
}

type channelAllowlistScope struct {
	name     string
	openMode bool
}

func loadChannelAllowlistConfig() (config.Config, string, error) {
	cfgPath, _, err := config.ResolveDefaultPaths()
	if err != nil {
		return config.Config{}, "", fmt.Errorf("failed to resolve config path: %w", err)
	}
	cfg, err := config.LoadConfig()
	if err != nil {
		return config.Config{}, "", fmt.Errorf("failed to load config: %w", err)
	}
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		cfg = config.DefaultConfig()
	} else if err != nil {
		return config.Config{}, "", fmt.Errorf("failed to inspect config path: %w", err)
	}
	return cfg, cfgPath, nil
}

func channelAllowlistEntries(cfg *config.Config, scope string) (*[]string, channelAllowlistScope, error) {
	switch normalizeAllowlistScope(scope) {
	case "telegram":
		return &cfg.Channels.Telegram.AllowFrom, channelAllowlistScope{name: "telegram", openMode: cfg.Channels.Telegram.OpenMode}, nil
	case "discord":
		return &cfg.Channels.Discord.AllowFrom, channelAllowlistScope{name: "discord", openMode: cfg.Channels.Discord.OpenMode}, nil
	case "whatsapp":
		return &cfg.Channels.WhatsApp.AllowFrom, channelAllowlistScope{name: "whatsapp", openMode: cfg.Channels.WhatsApp.OpenMode}, nil
	case "slack-users":
		return &cfg.Channels.Slack.AllowUsers, channelAllowlistScope{name: "slack-users", openMode: cfg.Channels.Slack.OpenUserMode}, nil
	case "slack-channels":
		return &cfg.Channels.Slack.AllowChannels, channelAllowlistScope{name: "slack-channels", openMode: cfg.Channels.Slack.OpenChannelMode}, nil
	default:
		return nil, channelAllowlistScope{}, fmt.Errorf("unknown allowlist scope %q (valid: telegram, discord, whatsapp, slack-users, slack-channels)", scope)
	}
}

func normalizeAllowlistScope(scope string) string {
	normalized := strings.ToLower(strings.TrimSpace(scope))
	normalized = strings.ReplaceAll(normalized, "_", "-")
	normalized = strings.ReplaceAll(normalized, ":", "-")
	switch normalized {
	case "slack-user", "slack-users", "slack-allow-users":
		return "slack-users"
	case "slack-channel", "slack-channels", "slack-allow-channels":
		return "slack-channels"
	default:
		return normalized
	}
}

func addAllowlistEntries(existing []string, additions []string) []string {
	seen := make(map[string]struct{}, len(existing)+len(additions))
	out := make([]string, 0, len(existing)+len(additions))
	for _, entry := range existing {
		trimmed := strings.TrimSpace(entry)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	for _, entry := range additions {
		trimmed := strings.TrimSpace(entry)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	if out == nil {
		return []string{}
	}
	return out
}

func removeAllowlistEntries(existing []string, removals []string) []string {
	remove := make(map[string]struct{}, len(removals))
	for _, entry := range removals {
		trimmed := strings.TrimSpace(entry)
		if trimmed != "" {
			remove[trimmed] = struct{}{}
		}
	}
	out := make([]string, 0, len(existing))
	for _, entry := range existing {
		trimmed := strings.TrimSpace(entry)
		if trimmed == "" {
			continue
		}
		if _, ok := remove[trimmed]; ok {
			continue
		}
		out = append(out, trimmed)
	}
	if out == nil {
		return []string{}
	}
	return out
}

func warnOpenModeForAllowlistScope(cmd *cobra.Command, scope channelAllowlistScope) {
	if scope.openMode {
		fmt.Fprintf(cmd.ErrOrStderr(), "warning: %s open mode is enabled; allowlist edits will not restrict access until open mode is disabled\n", scope.name)
	}
}

func pluralY(count int) string {
	if count == 1 {
		return "y"
	}
	return "ies"
}
