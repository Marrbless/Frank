package missioncontrol

import (
	"fmt"
	"strings"
)

type ResolvedExecutionContextFrankDiscordOwnerControlOnboardingBundle struct {
	Identity      FrankIdentityRecord    `json:"identity"`
	Account       FrankAccountRecord     `json:"account"`
	Provider      PlatformRecord         `json:"provider"`
	ProviderCheck EligibilityCheckRecord `json:"provider_check"`
	AccountClass  PlatformRecord         `json:"account_class"`
	AccountCheck  EligibilityCheckRecord `json:"account_check"`
}

type ResolvedExecutionContextFrankDiscordOwnerControlOnboardingPreflight struct {
	Identity *FrankIdentityRecord `json:"identity,omitempty"`
	Account  *FrankAccountRecord  `json:"account,omitempty"`
}

func DeclaresFrankDiscordOwnerControlOnboarding(step Step) bool {
	if NormalizeIdentityMode(step.IdentityMode) != IdentityModeOwnerOnlyControl {
		return false
	}
	return len(step.FrankObjectRefs) > 0
}

func ResolveExecutionContextFrankDiscordOwnerControlOnboardingBundle(ec ExecutionContext) (ResolvedExecutionContextFrankDiscordOwnerControlOnboardingBundle, bool, error) {
	resolved, err := ResolveExecutionContextFrankRegistryObjectRefs(ec)
	if err != nil {
		return ResolvedExecutionContextFrankDiscordOwnerControlOnboardingBundle{}, false, err
	}

	identities := make([]FrankIdentityRecord, 0, len(resolved.Identities))
	for _, identity := range resolved.Identities {
		if identity.DiscordOwnerControl != nil {
			identities = append(identities, identity)
		}
	}
	accounts := make([]FrankAccountRecord, 0, len(resolved.Accounts))
	for _, account := range resolved.Accounts {
		if account.DiscordOwnerControl != nil {
			accounts = append(accounts, account)
		}
	}

	if len(identities) == 0 && len(accounts) == 0 {
		return ResolvedExecutionContextFrankDiscordOwnerControlOnboardingBundle{}, false, nil
	}
	if len(identities) != 1 {
		return ResolvedExecutionContextFrankDiscordOwnerControlOnboardingBundle{}, false, fmt.Errorf("execution context Frank object refs must resolve exactly one discord owner-control identity, got %d", len(identities))
	}
	if len(accounts) != 1 {
		return ResolvedExecutionContextFrankDiscordOwnerControlOnboardingBundle{}, false, fmt.Errorf("execution context Frank object refs must resolve exactly one discord owner-control account, got %d", len(accounts))
	}

	identity := identities[0]
	account := accounts[0]
	if strings.TrimSpace(account.IdentityID) != strings.TrimSpace(identity.IdentityID) {
		return ResolvedExecutionContextFrankDiscordOwnerControlOnboardingBundle{}, false, fmt.Errorf(
			"execution context discord owner-control account %q must link identity_id %q, got %q",
			account.AccountID,
			identity.IdentityID,
			account.IdentityID,
		)
	}
	if strings.TrimSpace(account.ProviderOrPlatformID) != strings.TrimSpace(identity.ProviderOrPlatformID) {
		return ResolvedExecutionContextFrankDiscordOwnerControlOnboardingBundle{}, false, fmt.Errorf(
			"execution context discord owner-control account %q provider_or_platform_id %q does not match identity %q provider_or_platform_id %q",
			account.AccountID,
			account.ProviderOrPlatformID,
			identity.IdentityID,
			identity.ProviderOrPlatformID,
		)
	}
	if identity.EligibilityTargetRef.Kind != EligibilityTargetKindProvider {
		return ResolvedExecutionContextFrankDiscordOwnerControlOnboardingBundle{}, false, fmt.Errorf(
			"execution context discord owner-control identity %q eligibility_target_ref.kind %q must be %q",
			identity.IdentityID,
			identity.EligibilityTargetRef.Kind,
			EligibilityTargetKindProvider,
		)
	}
	if account.EligibilityTargetRef.Kind != EligibilityTargetKindAccountClass {
		return ResolvedExecutionContextFrankDiscordOwnerControlOnboardingBundle{}, false, fmt.Errorf(
			"execution context discord owner-control account %q eligibility_target_ref.kind %q must be %q",
			account.AccountID,
			account.EligibilityTargetRef.Kind,
			EligibilityTargetKindAccountClass,
		)
	}

	provider, err := LoadPlatformRecord(ec.MissionStoreRoot, identity.EligibilityTargetRef.RegistryID)
	if err != nil {
		return ResolvedExecutionContextFrankDiscordOwnerControlOnboardingBundle{}, false, fmt.Errorf("execution context discord owner-control identity %q provider target %q: %w", identity.IdentityID, identity.EligibilityTargetRef.RegistryID, err)
	}
	if provider.TargetClass != EligibilityTargetKindProvider {
		return ResolvedExecutionContextFrankDiscordOwnerControlOnboardingBundle{}, false, fmt.Errorf("execution context discord owner-control provider %q target_class %q must be %q", provider.PlatformID, provider.TargetClass, EligibilityTargetKindProvider)
	}
	providerCheck, err := LoadEligibilityCheckRecord(ec.MissionStoreRoot, provider.LastCheckID)
	if err != nil {
		return ResolvedExecutionContextFrankDiscordOwnerControlOnboardingBundle{}, false, fmt.Errorf("execution context discord owner-control provider %q last_check_id %q: %w", provider.PlatformID, provider.LastCheckID, err)
	}

	accountClass, err := LoadPlatformRecord(ec.MissionStoreRoot, account.EligibilityTargetRef.RegistryID)
	if err != nil {
		return ResolvedExecutionContextFrankDiscordOwnerControlOnboardingBundle{}, false, fmt.Errorf("execution context discord owner-control account %q account-class target %q: %w", account.AccountID, account.EligibilityTargetRef.RegistryID, err)
	}
	if accountClass.TargetClass != EligibilityTargetKindAccountClass {
		return ResolvedExecutionContextFrankDiscordOwnerControlOnboardingBundle{}, false, fmt.Errorf("execution context discord owner-control account-class %q target_class %q must be %q", accountClass.PlatformID, accountClass.TargetClass, EligibilityTargetKindAccountClass)
	}
	accountCheck, err := LoadEligibilityCheckRecord(ec.MissionStoreRoot, accountClass.LastCheckID)
	if err != nil {
		return ResolvedExecutionContextFrankDiscordOwnerControlOnboardingBundle{}, false, fmt.Errorf("execution context discord owner-control account-class %q last_check_id %q: %w", accountClass.PlatformID, accountClass.LastCheckID, err)
	}

	return ResolvedExecutionContextFrankDiscordOwnerControlOnboardingBundle{
		Identity:      identity,
		Account:       account,
		Provider:      provider,
		ProviderCheck: providerCheck,
		AccountClass:  accountClass,
		AccountCheck:  accountCheck,
	}, true, nil
}

func ResolveExecutionContextFrankDiscordOwnerControlOnboardingPreflight(ec ExecutionContext) (ResolvedExecutionContextFrankDiscordOwnerControlOnboardingPreflight, error) {
	if ec.Step == nil {
		return ResolvedExecutionContextFrankDiscordOwnerControlOnboardingPreflight{}, fmt.Errorf("execution context step is required")
	}
	if !DeclaresFrankDiscordOwnerControlOnboarding(*ec.Step) {
		return ResolvedExecutionContextFrankDiscordOwnerControlOnboardingPreflight{}, nil
	}
	if strings.TrimSpace(ec.MissionStoreRoot) == "" {
		return ResolvedExecutionContextFrankDiscordOwnerControlOnboardingPreflight{}, fmt.Errorf("mission store root is required to resolve Frank object refs")
	}

	bundle, ok, err := ResolveExecutionContextFrankDiscordOwnerControlOnboardingBundle(ec)
	if err != nil {
		return ResolvedExecutionContextFrankDiscordOwnerControlOnboardingPreflight{}, err
	}
	if !ok {
		return ResolvedExecutionContextFrankDiscordOwnerControlOnboardingPreflight{}, nil
	}

	return ResolvedExecutionContextFrankDiscordOwnerControlOnboardingPreflight{
		Identity: &bundle.Identity,
		Account:  &bundle.Account,
	}, nil
}

func normalizeFrankDiscordOwnerControlIdentity(config *FrankDiscordOwnerControlIdentity) *FrankDiscordOwnerControlIdentity {
	if config == nil {
		return nil
	}
	normalized := *config
	normalized.Username = strings.TrimSpace(normalized.Username)
	normalized.GlobalName = strings.TrimSpace(normalized.GlobalName)
	normalized.Discriminator = strings.TrimSpace(normalized.Discriminator)
	return &normalized
}

func normalizeFrankDiscordOwnerControlAccount(config *FrankDiscordOwnerControlAccount) *FrankDiscordOwnerControlAccount {
	if config == nil {
		return nil
	}
	normalized := *config
	normalized.BotUserID = strings.TrimSpace(normalized.BotUserID)
	return &normalized
}

func validateFrankDiscordOwnerControlIdentity(config FrankDiscordOwnerControlIdentity) error {
	_ = config
	return nil
}

func validateFrankDiscordOwnerControlAccount(config FrankDiscordOwnerControlAccount) error {
	if botUserID := strings.TrimSpace(config.BotUserID); botUserID != "" && !isDigitsOnly(botUserID) {
		return fmt.Errorf("mission store Frank account discord_owner_control.bot_user_id must be numeric when present")
	}
	return nil
}

func normalizeResolvedExecutionContextFrankDiscordOwnerControlOnboardingBundle(bundle ResolvedExecutionContextFrankDiscordOwnerControlOnboardingBundle) ResolvedExecutionContextFrankDiscordOwnerControlOnboardingBundle {
	bundle.Identity.ZohoMailbox = normalizeFrankZohoMailboxIdentity(bundle.Identity.ZohoMailbox)
	bundle.Identity.TelegramOwnerControl = normalizeFrankTelegramOwnerControlIdentity(bundle.Identity.TelegramOwnerControl)
	bundle.Identity.SlackOwnerControl = normalizeFrankSlackOwnerControlIdentity(bundle.Identity.SlackOwnerControl)
	bundle.Identity.DiscordOwnerControl = normalizeFrankDiscordOwnerControlIdentity(bundle.Identity.DiscordOwnerControl)
	bundle.Identity.IdentityMode = NormalizeIdentityMode(bundle.Identity.IdentityMode)
	bundle.Identity.CreatedAt = bundle.Identity.CreatedAt.UTC()
	bundle.Identity.UpdatedAt = bundle.Identity.UpdatedAt.UTC()

	bundle.Account.ZohoMailbox = normalizeFrankZohoMailboxAccount(bundle.Account.ZohoMailbox)
	bundle.Account.TelegramOwnerControl = normalizeFrankTelegramOwnerControlAccount(bundle.Account.TelegramOwnerControl)
	bundle.Account.SlackOwnerControl = normalizeFrankSlackOwnerControlAccount(bundle.Account.SlackOwnerControl)
	bundle.Account.DiscordOwnerControl = normalizeFrankDiscordOwnerControlAccount(bundle.Account.DiscordOwnerControl)
	bundle.Account.CreatedAt = bundle.Account.CreatedAt.UTC()
	bundle.Account.UpdatedAt = bundle.Account.UpdatedAt.UTC()

	return bundle
}

func validateResolvedExecutionContextFrankDiscordOwnerControlOnboardingBundle(bundle ResolvedExecutionContextFrankDiscordOwnerControlOnboardingBundle) error {
	if err := ValidateFrankIdentityRecord(bundle.Identity); err != nil {
		return err
	}
	if err := ValidateFrankAccountRecord(bundle.Account); err != nil {
		return err
	}
	if bundle.Identity.DiscordOwnerControl == nil {
		return fmt.Errorf("mission store Frank discord owner-control onboarding requires discord_owner_control identity record")
	}
	if bundle.Account.DiscordOwnerControl == nil {
		return fmt.Errorf("mission store Frank discord owner-control onboarding requires discord_owner_control account record")
	}
	if NormalizeIdentityMode(bundle.Identity.IdentityMode) != IdentityModeOwnerOnlyControl {
		return fmt.Errorf(
			"mission store Frank discord owner-control onboarding identity %q identity_mode %q must be %q",
			bundle.Identity.IdentityID,
			bundle.Identity.IdentityMode,
			IdentityModeOwnerOnlyControl,
		)
	}
	if strings.TrimSpace(bundle.Account.IdentityID) != strings.TrimSpace(bundle.Identity.IdentityID) {
		return fmt.Errorf(
			"mission store Frank discord owner-control onboarding account %q must link identity_id %q, got %q",
			bundle.Account.AccountID,
			bundle.Identity.IdentityID,
			bundle.Account.IdentityID,
		)
	}
	if strings.TrimSpace(bundle.Account.ProviderOrPlatformID) != strings.TrimSpace(bundle.Identity.ProviderOrPlatformID) {
		return fmt.Errorf(
			"mission store Frank discord owner-control onboarding account %q provider_or_platform_id %q does not match identity %q provider_or_platform_id %q",
			bundle.Account.AccountID,
			bundle.Account.ProviderOrPlatformID,
			bundle.Identity.IdentityID,
			bundle.Identity.ProviderOrPlatformID,
		)
	}
	if bundle.Provider.TargetClass != EligibilityTargetKindProvider {
		return fmt.Errorf("mission store Frank discord owner-control onboarding provider %q target_class %q must be %q", bundle.Provider.PlatformID, bundle.Provider.TargetClass, EligibilityTargetKindProvider)
	}
	if bundle.AccountClass.TargetClass != EligibilityTargetKindAccountClass {
		return fmt.Errorf("mission store Frank discord owner-control onboarding account-class %q target_class %q must be %q", bundle.AccountClass.PlatformID, bundle.AccountClass.TargetClass, EligibilityTargetKindAccountClass)
	}
	return nil
}
