package missioncontrol

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/local/picobot/internal/channels"
	"github.com/local/picobot/internal/config"
)

var readDiscordOwnerControlBotIdentity = channels.ReadDiscordBotIdentity

// ProduceFrankDiscordOwnerControlOnboarding is the single missioncontrol-owned
// execution producer for the Discord owner-control onboarding lane. It reads
// the configured Discord owner-control channel, confirms bot identity through
// provider read-back only, and persists only non-secret Discord identifiers
// into the committed Frank identity/account records.
func ProduceFrankDiscordOwnerControlOnboarding(root string, bundle ResolvedExecutionContextFrankDiscordOwnerControlOnboardingBundle, now time.Time) error {
	if err := ValidateStoreRoot(root); err != nil {
		return err
	}
	if now.IsZero() {
		now = time.Now().UTC()
	} else {
		now = now.UTC()
	}

	bundle = normalizeResolvedExecutionContextFrankDiscordOwnerControlOnboardingBundle(bundle)
	if err := validateResolvedExecutionContextFrankDiscordOwnerControlOnboardingBundle(bundle); err != nil {
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
		return fmt.Errorf("discord owner-control onboarding requires readable config: %w", err)
	}
	channelCfg, err := resolveDiscordOwnerControlConfig(cfg)
	if err != nil {
		return err
	}

	readBack, err := readDiscordOwnerControlBotIdentity(context.Background(), channelCfg.Token)
	if err != nil {
		return fmt.Errorf("discord owner-control onboarding provider read-back failed: %w", err)
	}

	committedBotUserID := ""
	if bundle.Account.DiscordOwnerControl != nil {
		committedBotUserID = strings.TrimSpace(bundle.Account.DiscordOwnerControl.BotUserID)
	}
	if committedBotUserID != "" && committedBotUserID != readBack.BotUserID {
		return fmt.Errorf(
			"discord owner-control account %q conflicts with provider bot user id %q",
			bundle.Account.AccountID,
			readBack.BotUserID,
		)
	}

	committedUsername := ""
	committedGlobalName := ""
	committedDiscriminator := ""
	if bundle.Identity.DiscordOwnerControl != nil {
		committedUsername = strings.TrimSpace(bundle.Identity.DiscordOwnerControl.Username)
		committedGlobalName = strings.TrimSpace(bundle.Identity.DiscordOwnerControl.GlobalName)
		committedDiscriminator = strings.TrimSpace(bundle.Identity.DiscordOwnerControl.Discriminator)
	}
	if committedUsername != "" && committedUsername != readBack.Username {
		return fmt.Errorf(
			"discord owner-control identity %q conflicts with provider username %q",
			bundle.Identity.IdentityID,
			readBack.Username,
		)
	}
	if committedGlobalName != "" && committedGlobalName != readBack.GlobalName {
		return fmt.Errorf(
			"discord owner-control identity %q conflicts with provider global_name %q",
			bundle.Identity.IdentityID,
			readBack.GlobalName,
		)
	}
	if committedDiscriminator != "" && committedDiscriminator != readBack.Discriminator {
		return fmt.Errorf(
			"discord owner-control identity %q conflicts with provider discriminator %q",
			bundle.Identity.IdentityID,
			readBack.Discriminator,
		)
	}

	if committedBotUserID == readBack.BotUserID &&
		committedUsername == readBack.Username &&
		committedGlobalName == readBack.GlobalName &&
		committedDiscriminator == readBack.Discriminator {
		return nil
	}

	updatedIdentity := bundle.Identity
	if updatedIdentity.DiscordOwnerControl == nil {
		updatedIdentity.DiscordOwnerControl = &FrankDiscordOwnerControlIdentity{}
	}
	updatedIdentity.DiscordOwnerControl.Username = readBack.Username
	updatedIdentity.DiscordOwnerControl.GlobalName = readBack.GlobalName
	updatedIdentity.DiscordOwnerControl.Discriminator = readBack.Discriminator
	updatedIdentity.UpdatedAt = updatedAtOnOrAfterCreatedAt(updatedIdentity.CreatedAt, now)

	updatedAccount := bundle.Account
	if updatedAccount.DiscordOwnerControl == nil {
		updatedAccount.DiscordOwnerControl = &FrankDiscordOwnerControlAccount{}
	}
	updatedAccount.DiscordOwnerControl.BotUserID = readBack.BotUserID
	updatedAccount.UpdatedAt = updatedAtOnOrAfterCreatedAt(updatedAccount.CreatedAt, now)

	if err := StoreFrankIdentityRecord(root, updatedIdentity); err != nil {
		return err
	}
	return StoreFrankAccountRecord(root, updatedAccount)
}

func resolveDiscordOwnerControlConfig(cfg config.Config) (config.DiscordConfig, error) {
	channelCfg := cfg.Channels.Discord
	if !channelCfg.Enabled {
		return config.DiscordConfig{}, fmt.Errorf("discord owner-control onboarding requires configured discord channel enabled")
	}
	if strings.TrimSpace(channelCfg.Token) == "" {
		return config.DiscordConfig{}, fmt.Errorf("discord owner-control onboarding requires configured discord token")
	}

	allowFrom := make([]string, 0, len(channelCfg.AllowFrom))
	for _, raw := range channelCfg.AllowFrom {
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" {
			continue
		}
		if !isDigitsOnly(trimmed) {
			return config.DiscordConfig{}, fmt.Errorf("discord owner-control onboarding allowFrom entry %q must be numeric", trimmed)
		}
		allowFrom = append(allowFrom, trimmed)
	}
	if len(allowFrom) != 1 {
		return config.DiscordConfig{}, fmt.Errorf("discord owner-control onboarding requires exactly one configured discord allowFrom user id, got %d", len(allowFrom))
	}
	channelCfg.AllowFrom = allowFrom
	return channelCfg, nil
}
