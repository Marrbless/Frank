package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/local/picobot/internal/config"
)

type onboardingChecklistReport struct {
	Complete bool                      `json:"complete"`
	Steps    []onboardingChecklistStep `json:"steps"`
}

type onboardingChecklistStep struct {
	ID     string `json:"id"`
	State  string `json:"state"`
	Detail string `json:"detail,omitempty"`
}

func newOnboardChecklistCmd() *cobra.Command {
	return &cobra.Command{
		Use:          "checklist",
		Short:        "Print read-only onboarding next steps",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			report := buildOnboardingChecklist(config.DefaultConfigPath())
			data, err := json.MarshalIndent(report, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to encode onboarding checklist: %w", err)
			}
			data = append(data, '\n')
			if _, err := cmd.OutOrStdout().Write(data); err != nil {
				return fmt.Errorf("failed to write onboarding checklist: %w", err)
			}
			return nil
		},
	}
}

func buildOnboardingChecklist(configPath string) onboardingChecklistReport {
	cfg, exists, loadErr := loadDiagnosticsConfig(configPath)
	steps := make([]onboardingChecklistStep, 0, 5)

	if loadErr != nil {
		steps = append(steps, onboardingChecklistStep{
			ID:     "config",
			State:  "error",
			Detail: loadErr.Error(),
		})
		return onboardingChecklistReport{Complete: false, Steps: steps}
	}

	if !exists {
		steps = append(steps,
			pendingOnboardingStep("config", "run picobot onboard to create ~/.picobot/config.json"),
			pendingOnboardingStep("workspace", "workspace is created by picobot onboard"),
			pendingOnboardingStep("provider", "set providers.openai.apiKey to a real key"),
			pendingOnboardingStep("owner_channel", "enable one owner channel with a token and allowlist"),
		)
		return onboardingChecklistReport{Complete: false, Steps: steps}
	}

	steps = append(steps, onboardingChecklistStep{
		ID:     "config",
		State:  "done",
		Detail: "config file exists and decoded",
	})
	steps = append(steps, workspaceOnboardingStep(cfg))
	steps = append(steps, providerOnboardingStep(cfg))
	steps = append(steps, ownerChannelOnboardingStep(cfg))

	complete := true
	for _, step := range steps {
		if step.State != "done" {
			complete = false
			break
		}
	}
	return onboardingChecklistReport{Complete: complete, Steps: steps}
}

func pendingOnboardingStep(id string, detail string) onboardingChecklistStep {
	return onboardingChecklistStep{ID: id, State: "pending", Detail: detail}
}

func workspaceOnboardingStep(cfg config.Config) onboardingChecklistStep {
	workspace := strings.TrimSpace(cfg.Agents.Defaults.Workspace)
	if workspace == "" {
		return pendingOnboardingStep("workspace", "agents.defaults.workspace is empty")
	}
	resolved, err := expandDiagnosticsPath(workspace)
	if err != nil {
		return onboardingChecklistStep{ID: "workspace", State: "error", Detail: err.Error()}
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return pendingOnboardingStep("workspace", fmt.Sprintf("workspace directory %q is missing", resolved))
	}
	if !info.IsDir() {
		return onboardingChecklistStep{ID: "workspace", State: "error", Detail: fmt.Sprintf("workspace path %q is not a directory", resolved)}
	}
	return onboardingChecklistStep{ID: "workspace", State: "done", Detail: "workspace directory exists"}
}

func providerOnboardingStep(cfg config.Config) onboardingChecklistStep {
	if cfg.Providers.OpenAI == nil {
		return pendingOnboardingStep("provider", "providers.openai is missing")
	}
	apiKey := strings.TrimSpace(cfg.Providers.OpenAI.APIKey)
	if apiKey == "" || strings.Contains(apiKey, "REPLACE_WITH") {
		return pendingOnboardingStep("provider", "set providers.openai.apiKey to a real key")
	}
	if strings.TrimSpace(cfg.Providers.OpenAI.APIBase) == "" {
		return pendingOnboardingStep("provider", "set providers.openai.apiBase")
	}
	return onboardingChecklistStep{ID: "provider", State: "done", Detail: "OpenAI-compatible provider is configured"}
}

func ownerChannelOnboardingStep(cfg config.Config) onboardingChecklistStep {
	switch {
	case cfg.Channels.Telegram.Enabled && strings.TrimSpace(cfg.Channels.Telegram.Token) != "" && (len(cfg.Channels.Telegram.AllowFrom) > 0 || cfg.Channels.Telegram.OpenMode):
		return onboardingChecklistStep{ID: "owner_channel", State: "done", Detail: "telegram owner channel is configured"}
	case cfg.Channels.Discord.Enabled && strings.TrimSpace(cfg.Channels.Discord.Token) != "" && (len(cfg.Channels.Discord.AllowFrom) > 0 || cfg.Channels.Discord.OpenMode):
		return onboardingChecklistStep{ID: "owner_channel", State: "done", Detail: "discord owner channel is configured"}
	case cfg.Channels.Slack.Enabled && strings.TrimSpace(cfg.Channels.Slack.AppToken) != "" && strings.TrimSpace(cfg.Channels.Slack.BotToken) != "" && (len(cfg.Channels.Slack.AllowUsers) > 0 || cfg.Channels.Slack.OpenUserMode) && (len(cfg.Channels.Slack.AllowChannels) > 0 || cfg.Channels.Slack.OpenChannelMode):
		return onboardingChecklistStep{ID: "owner_channel", State: "done", Detail: "slack owner channel is configured"}
	case cfg.Channels.WhatsApp.Enabled && len(cfg.Channels.WhatsApp.AllowFrom) > 0 || cfg.Channels.WhatsApp.Enabled && cfg.Channels.WhatsApp.OpenMode:
		return onboardingChecklistStep{ID: "owner_channel", State: "done", Detail: "whatsapp owner channel is configured"}
	default:
		return pendingOnboardingStep("owner_channel", "enable Telegram, Discord, Slack, or WhatsApp with token/session and allowlist or explicit open mode")
	}
}
