package missioncontrol

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/local/picobot/internal/channels"
	"github.com/local/picobot/internal/config"
)

var readSlackOwnerControlIdentity = channels.ReadSlackAuthIdentity

// ProduceFrankSlackOwnerControlOnboarding is the single missioncontrol-owned
// execution producer for the Slack owner-control onboarding lane. It reads the
// configured Slack owner-control channel, confirms bot identity through
// provider read-back only, and persists only non-secret Slack identifiers into
// the committed Frank identity/account records.
func ProduceFrankSlackOwnerControlOnboarding(root string, bundle ResolvedExecutionContextFrankSlackOwnerControlOnboardingBundle, now time.Time) error {
	if err := ValidateStoreRoot(root); err != nil {
		return err
	}
	if now.IsZero() {
		now = time.Now().UTC()
	} else {
		now = now.UTC()
	}

	bundle = normalizeResolvedExecutionContextFrankSlackOwnerControlOnboardingBundle(bundle)
	if err := validateResolvedExecutionContextFrankSlackOwnerControlOnboardingBundle(bundle); err != nil {
		return err
	}
	if _, err := RequireAutonomyEligibleTarget(root, bundle.Identity.EligibilityTargetRef); err != nil {
		return err
	}
	if _, err := RequireAutonomyEligibleTarget(root, bundle.Account.EligibilityTargetRef); err != nil {
		return err
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("slack owner-control onboarding requires readable config: %w", err)
	}
	channelCfg, err := resolveSlackOwnerControlConfig(cfg)
	if err != nil {
		return err
	}

	readBack, err := readSlackOwnerControlIdentity(context.Background(), channelCfg.BotToken)
	if err != nil {
		return fmt.Errorf("slack owner-control onboarding provider read-back failed: %w", err)
	}

	committedTeamID := ""
	committedUserID := ""
	if bundle.Identity.SlackOwnerControl != nil {
		committedTeamID = strings.TrimSpace(bundle.Identity.SlackOwnerControl.TeamID)
		committedUserID = strings.TrimSpace(bundle.Identity.SlackOwnerControl.UserID)
	}
	if committedTeamID != "" && committedTeamID != readBack.TeamID {
		return fmt.Errorf(
			"slack owner-control identity %q conflicts with provider team id %q",
			bundle.Identity.IdentityID,
			readBack.TeamID,
		)
	}
	if committedUserID != "" && committedUserID != readBack.UserID {
		return fmt.Errorf(
			"slack owner-control identity %q conflicts with provider user id %q",
			bundle.Identity.IdentityID,
			readBack.UserID,
		)
	}

	committedBotID := ""
	if bundle.Account.SlackOwnerControl != nil {
		committedBotID = strings.TrimSpace(bundle.Account.SlackOwnerControl.BotID)
	}
	if committedBotID != "" && committedBotID != readBack.BotID {
		return fmt.Errorf(
			"slack owner-control account %q conflicts with provider bot id %q",
			bundle.Account.AccountID,
			readBack.BotID,
		)
	}

	if committedTeamID == readBack.TeamID && committedUserID == readBack.UserID && committedBotID == readBack.BotID {
		return nil
	}

	updatedIdentity := bundle.Identity
	if updatedIdentity.SlackOwnerControl == nil {
		updatedIdentity.SlackOwnerControl = &FrankSlackOwnerControlIdentity{}
	}
	updatedIdentity.SlackOwnerControl.TeamID = readBack.TeamID
	updatedIdentity.SlackOwnerControl.UserID = readBack.UserID
	updatedIdentity.UpdatedAt = updatedAtOnOrAfterCreatedAt(updatedIdentity.CreatedAt, now)

	updatedAccount := bundle.Account
	if updatedAccount.SlackOwnerControl == nil {
		updatedAccount.SlackOwnerControl = &FrankSlackOwnerControlAccount{}
	}
	updatedAccount.SlackOwnerControl.BotID = readBack.BotID
	updatedAccount.UpdatedAt = updatedAtOnOrAfterCreatedAt(updatedAccount.CreatedAt, now)

	if err := StoreFrankIdentityRecord(root, updatedIdentity); err != nil {
		return err
	}
	return StoreFrankAccountRecord(root, updatedAccount)
}

func resolveSlackOwnerControlConfig(cfg config.Config) (config.SlackConfig, error) {
	channelCfg := cfg.Channels.Slack
	if !channelCfg.Enabled {
		return config.SlackConfig{}, fmt.Errorf("slack owner-control onboarding requires configured slack channel enabled")
	}
	if strings.TrimSpace(channelCfg.AppToken) == "" {
		return config.SlackConfig{}, fmt.Errorf("slack owner-control onboarding requires configured slack app token")
	}
	if strings.TrimSpace(channelCfg.BotToken) == "" {
		return config.SlackConfig{}, fmt.Errorf("slack owner-control onboarding requires configured slack bot token")
	}

	allowUsers := make([]string, 0, len(channelCfg.AllowUsers))
	for _, raw := range channelCfg.AllowUsers {
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" {
			continue
		}
		allowUsers = append(allowUsers, trimmed)
	}
	if len(allowUsers) != 1 {
		return config.SlackConfig{}, fmt.Errorf("slack owner-control onboarding requires exactly one configured slack allowUsers user id, got %d", len(allowUsers))
	}
	channelCfg.AllowUsers = allowUsers
	return channelCfg, nil
}
