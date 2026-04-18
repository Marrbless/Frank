package missioncontrol

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/local/picobot/internal/channels"
	"github.com/local/picobot/internal/config"
)

var readTelegramOwnerControlBotIdentity = channels.ReadTelegramBotIdentity

// ProduceFrankTelegramOwnerControlOnboarding is the single missioncontrol-owned
// execution producer for the Telegram owner-control onboarding lane. It reads
// the configured Telegram owner-control channel, confirms bot identity through
// provider read-back only, and persists only non-secret Telegram identifiers
// into the committed Frank identity/account records.
func ProduceFrankTelegramOwnerControlOnboarding(root string, bundle ResolvedExecutionContextFrankTelegramOwnerControlOnboardingBundle, now time.Time) error {
	if err := ValidateStoreRoot(root); err != nil {
		return err
	}
	if now.IsZero() {
		now = time.Now().UTC()
	} else {
		now = now.UTC()
	}

	bundle = normalizeResolvedExecutionContextFrankTelegramOwnerControlOnboardingBundle(bundle)
	if err := validateResolvedExecutionContextFrankTelegramOwnerControlOnboardingBundle(bundle); err != nil {
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
		return fmt.Errorf("telegram owner-control onboarding requires readable config: %w", err)
	}
	channelCfg, ownerUserID, err := resolveTelegramOwnerControlConfig(cfg)
	if err != nil {
		return err
	}

	readBack, err := readTelegramOwnerControlBotIdentity(context.Background(), strings.TrimSpace(channelCfg.Token))
	if err != nil {
		return fmt.Errorf("telegram owner-control onboarding provider read-back failed: %w", err)
	}

	committedOwnerUserID := ""
	if bundle.Identity.TelegramOwnerControl != nil {
		committedOwnerUserID = strings.TrimSpace(bundle.Identity.TelegramOwnerControl.OwnerUserID)
	}
	if committedOwnerUserID != "" && committedOwnerUserID != ownerUserID {
		return fmt.Errorf(
			"telegram owner-control identity %q conflicts with configured owner user id %q",
			bundle.Identity.IdentityID,
			ownerUserID,
		)
	}

	committedBotUserID := ""
	if bundle.Account.TelegramOwnerControl != nil {
		committedBotUserID = strings.TrimSpace(bundle.Account.TelegramOwnerControl.BotUserID)
	}
	if committedBotUserID != "" && committedBotUserID != strings.TrimSpace(readBack.BotUserID) {
		return fmt.Errorf(
			"telegram owner-control account %q conflicts with provider bot user id %q",
			bundle.Account.AccountID,
			readBack.BotUserID,
		)
	}

	if committedOwnerUserID == ownerUserID && committedBotUserID == strings.TrimSpace(readBack.BotUserID) {
		return nil
	}

	updatedIdentity := bundle.Identity
	if updatedIdentity.TelegramOwnerControl == nil {
		updatedIdentity.TelegramOwnerControl = &FrankTelegramOwnerControlIdentity{}
	}
	updatedIdentity.TelegramOwnerControl.OwnerUserID = ownerUserID
	updatedIdentity.UpdatedAt = updatedAtOnOrAfterCreatedAt(updatedIdentity.CreatedAt, now)

	updatedAccount := bundle.Account
	if updatedAccount.TelegramOwnerControl == nil {
		updatedAccount.TelegramOwnerControl = &FrankTelegramOwnerControlAccount{}
	}
	updatedAccount.TelegramOwnerControl.BotUserID = strings.TrimSpace(readBack.BotUserID)
	updatedAccount.UpdatedAt = updatedAtOnOrAfterCreatedAt(updatedAccount.CreatedAt, now)

	if err := StoreFrankIdentityRecord(root, updatedIdentity); err != nil {
		return err
	}
	return StoreFrankAccountRecord(root, updatedAccount)
}

func normalizeResolvedExecutionContextFrankTelegramOwnerControlOnboardingBundle(bundle ResolvedExecutionContextFrankTelegramOwnerControlOnboardingBundle) ResolvedExecutionContextFrankTelegramOwnerControlOnboardingBundle {
	bundle.Identity.ZohoMailbox = normalizeFrankZohoMailboxIdentity(bundle.Identity.ZohoMailbox)
	bundle.Identity.TelegramOwnerControl = normalizeFrankTelegramOwnerControlIdentity(bundle.Identity.TelegramOwnerControl)
	bundle.Identity.IdentityMode = NormalizeIdentityMode(bundle.Identity.IdentityMode)
	bundle.Identity.CreatedAt = bundle.Identity.CreatedAt.UTC()
	bundle.Identity.UpdatedAt = bundle.Identity.UpdatedAt.UTC()

	bundle.Account.ZohoMailbox = normalizeFrankZohoMailboxAccount(bundle.Account.ZohoMailbox)
	bundle.Account.TelegramOwnerControl = normalizeFrankTelegramOwnerControlAccount(bundle.Account.TelegramOwnerControl)
	bundle.Account.CreatedAt = bundle.Account.CreatedAt.UTC()
	bundle.Account.UpdatedAt = bundle.Account.UpdatedAt.UTC()

	return bundle
}

func validateResolvedExecutionContextFrankTelegramOwnerControlOnboardingBundle(bundle ResolvedExecutionContextFrankTelegramOwnerControlOnboardingBundle) error {
	if err := ValidateFrankIdentityRecord(bundle.Identity); err != nil {
		return err
	}
	if err := ValidateFrankAccountRecord(bundle.Account); err != nil {
		return err
	}
	if bundle.Identity.TelegramOwnerControl == nil {
		return fmt.Errorf("mission store Frank telegram owner-control onboarding requires telegram_owner_control identity record")
	}
	if bundle.Account.TelegramOwnerControl == nil {
		return fmt.Errorf("mission store Frank telegram owner-control onboarding requires telegram_owner_control account record")
	}
	if NormalizeIdentityMode(bundle.Identity.IdentityMode) != IdentityModeOwnerOnlyControl {
		return fmt.Errorf(
			"mission store Frank telegram owner-control onboarding identity %q identity_mode %q must be %q",
			bundle.Identity.IdentityID,
			bundle.Identity.IdentityMode,
			IdentityModeOwnerOnlyControl,
		)
	}
	if strings.TrimSpace(bundle.Account.IdentityID) != strings.TrimSpace(bundle.Identity.IdentityID) {
		return fmt.Errorf(
			"mission store Frank telegram owner-control onboarding account %q must link identity_id %q, got %q",
			bundle.Account.AccountID,
			bundle.Identity.IdentityID,
			bundle.Account.IdentityID,
		)
	}
	if strings.TrimSpace(bundle.Account.ProviderOrPlatformID) != strings.TrimSpace(bundle.Identity.ProviderOrPlatformID) {
		return fmt.Errorf(
			"mission store Frank telegram owner-control onboarding account %q provider_or_platform_id %q does not match identity %q provider_or_platform_id %q",
			bundle.Account.AccountID,
			bundle.Account.ProviderOrPlatformID,
			bundle.Identity.IdentityID,
			bundle.Identity.ProviderOrPlatformID,
		)
	}
	if bundle.Provider.TargetClass != EligibilityTargetKindProvider {
		return fmt.Errorf("mission store Frank telegram owner-control onboarding provider %q target_class %q must be %q", bundle.Provider.PlatformID, bundle.Provider.TargetClass, EligibilityTargetKindProvider)
	}
	if bundle.AccountClass.TargetClass != EligibilityTargetKindAccountClass {
		return fmt.Errorf("mission store Frank telegram owner-control onboarding account-class %q target_class %q must be %q", bundle.AccountClass.PlatformID, bundle.AccountClass.TargetClass, EligibilityTargetKindAccountClass)
	}
	return nil
}

func resolveTelegramOwnerControlConfig(cfg config.Config) (config.TelegramConfig, string, error) {
	channelCfg := cfg.Channels.Telegram
	if !channelCfg.Enabled {
		return config.TelegramConfig{}, "", fmt.Errorf("telegram owner-control onboarding requires configured telegram channel enabled")
	}
	if strings.TrimSpace(channelCfg.Token) == "" {
		return config.TelegramConfig{}, "", fmt.Errorf("telegram owner-control onboarding requires configured telegram token")
	}

	allowFrom := make([]string, 0, len(channelCfg.AllowFrom))
	for _, raw := range channelCfg.AllowFrom {
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" {
			continue
		}
		if !isDigitsOnly(trimmed) {
			return config.TelegramConfig{}, "", fmt.Errorf("telegram owner-control onboarding allowFrom entry %q must be numeric", trimmed)
		}
		allowFrom = append(allowFrom, trimmed)
	}
	if len(allowFrom) != 1 {
		return config.TelegramConfig{}, "", fmt.Errorf("telegram owner-control onboarding requires exactly one configured telegram allowFrom user id, got %d", len(allowFrom))
	}
	channelCfg.AllowFrom = allowFrom
	return channelCfg, allowFrom[0], nil
}

func updatedAtOnOrAfterCreatedAt(createdAt, now time.Time) time.Time {
	if now.Before(createdAt) {
		return createdAt
	}
	return now
}
