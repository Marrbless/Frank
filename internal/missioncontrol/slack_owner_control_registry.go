package missioncontrol

import (
	"fmt"
	"strings"
)

type ResolvedExecutionContextFrankSlackOwnerControlOnboardingBundle struct {
	Identity      FrankIdentityRecord    `json:"identity"`
	Account       FrankAccountRecord     `json:"account"`
	Provider      PlatformRecord         `json:"provider"`
	ProviderCheck EligibilityCheckRecord `json:"provider_check"`
	AccountClass  PlatformRecord         `json:"account_class"`
	AccountCheck  EligibilityCheckRecord `json:"account_check"`
}

type ResolvedExecutionContextFrankSlackOwnerControlOnboardingPreflight struct {
	Identity *FrankIdentityRecord `json:"identity,omitempty"`
	Account  *FrankAccountRecord  `json:"account,omitempty"`
}

func DeclaresFrankSlackOwnerControlOnboarding(step Step) bool {
	if NormalizeIdentityMode(step.IdentityMode) != IdentityModeOwnerOnlyControl {
		return false
	}
	return len(step.FrankObjectRefs) > 0
}

func ResolveExecutionContextFrankSlackOwnerControlOnboardingBundle(ec ExecutionContext) (ResolvedExecutionContextFrankSlackOwnerControlOnboardingBundle, bool, error) {
	resolved, err := ResolveExecutionContextFrankRegistryObjectRefs(ec)
	if err != nil {
		return ResolvedExecutionContextFrankSlackOwnerControlOnboardingBundle{}, false, err
	}

	identities := make([]FrankIdentityRecord, 0, len(resolved.Identities))
	for _, identity := range resolved.Identities {
		if identity.SlackOwnerControl != nil {
			identities = append(identities, identity)
		}
	}
	accounts := make([]FrankAccountRecord, 0, len(resolved.Accounts))
	for _, account := range resolved.Accounts {
		if account.SlackOwnerControl != nil {
			accounts = append(accounts, account)
		}
	}

	if len(identities) == 0 && len(accounts) == 0 {
		return ResolvedExecutionContextFrankSlackOwnerControlOnboardingBundle{}, false, nil
	}
	if len(identities) != 1 {
		return ResolvedExecutionContextFrankSlackOwnerControlOnboardingBundle{}, false, fmt.Errorf("execution context Frank object refs must resolve exactly one slack owner-control identity, got %d", len(identities))
	}
	if len(accounts) != 1 {
		return ResolvedExecutionContextFrankSlackOwnerControlOnboardingBundle{}, false, fmt.Errorf("execution context Frank object refs must resolve exactly one slack owner-control account, got %d", len(accounts))
	}

	identity := identities[0]
	account := accounts[0]
	if strings.TrimSpace(account.IdentityID) != strings.TrimSpace(identity.IdentityID) {
		return ResolvedExecutionContextFrankSlackOwnerControlOnboardingBundle{}, false, fmt.Errorf(
			"execution context slack owner-control account %q must link identity_id %q, got %q",
			account.AccountID,
			identity.IdentityID,
			account.IdentityID,
		)
	}
	if strings.TrimSpace(account.ProviderOrPlatformID) != strings.TrimSpace(identity.ProviderOrPlatformID) {
		return ResolvedExecutionContextFrankSlackOwnerControlOnboardingBundle{}, false, fmt.Errorf(
			"execution context slack owner-control account %q provider_or_platform_id %q does not match identity %q provider_or_platform_id %q",
			account.AccountID,
			account.ProviderOrPlatformID,
			identity.IdentityID,
			identity.ProviderOrPlatformID,
		)
	}
	if identity.EligibilityTargetRef.Kind != EligibilityTargetKindProvider {
		return ResolvedExecutionContextFrankSlackOwnerControlOnboardingBundle{}, false, fmt.Errorf(
			"execution context slack owner-control identity %q eligibility_target_ref.kind %q must be %q",
			identity.IdentityID,
			identity.EligibilityTargetRef.Kind,
			EligibilityTargetKindProvider,
		)
	}
	if account.EligibilityTargetRef.Kind != EligibilityTargetKindAccountClass {
		return ResolvedExecutionContextFrankSlackOwnerControlOnboardingBundle{}, false, fmt.Errorf(
			"execution context slack owner-control account %q eligibility_target_ref.kind %q must be %q",
			account.AccountID,
			account.EligibilityTargetRef.Kind,
			EligibilityTargetKindAccountClass,
		)
	}

	provider, err := LoadPlatformRecord(ec.MissionStoreRoot, identity.EligibilityTargetRef.RegistryID)
	if err != nil {
		return ResolvedExecutionContextFrankSlackOwnerControlOnboardingBundle{}, false, fmt.Errorf("execution context slack owner-control identity %q provider target %q: %w", identity.IdentityID, identity.EligibilityTargetRef.RegistryID, err)
	}
	if provider.TargetClass != EligibilityTargetKindProvider {
		return ResolvedExecutionContextFrankSlackOwnerControlOnboardingBundle{}, false, fmt.Errorf("execution context slack owner-control provider %q target_class %q must be %q", provider.PlatformID, provider.TargetClass, EligibilityTargetKindProvider)
	}
	providerCheck, err := LoadEligibilityCheckRecord(ec.MissionStoreRoot, provider.LastCheckID)
	if err != nil {
		return ResolvedExecutionContextFrankSlackOwnerControlOnboardingBundle{}, false, fmt.Errorf("execution context slack owner-control provider %q last_check_id %q: %w", provider.PlatformID, provider.LastCheckID, err)
	}

	accountClass, err := LoadPlatformRecord(ec.MissionStoreRoot, account.EligibilityTargetRef.RegistryID)
	if err != nil {
		return ResolvedExecutionContextFrankSlackOwnerControlOnboardingBundle{}, false, fmt.Errorf("execution context slack owner-control account %q account-class target %q: %w", account.AccountID, account.EligibilityTargetRef.RegistryID, err)
	}
	if accountClass.TargetClass != EligibilityTargetKindAccountClass {
		return ResolvedExecutionContextFrankSlackOwnerControlOnboardingBundle{}, false, fmt.Errorf("execution context slack owner-control account-class %q target_class %q must be %q", accountClass.PlatformID, accountClass.TargetClass, EligibilityTargetKindAccountClass)
	}
	accountCheck, err := LoadEligibilityCheckRecord(ec.MissionStoreRoot, accountClass.LastCheckID)
	if err != nil {
		return ResolvedExecutionContextFrankSlackOwnerControlOnboardingBundle{}, false, fmt.Errorf("execution context slack owner-control account-class %q last_check_id %q: %w", accountClass.PlatformID, accountClass.LastCheckID, err)
	}

	return ResolvedExecutionContextFrankSlackOwnerControlOnboardingBundle{
		Identity:      identity,
		Account:       account,
		Provider:      provider,
		ProviderCheck: providerCheck,
		AccountClass:  accountClass,
		AccountCheck:  accountCheck,
	}, true, nil
}

func ResolveExecutionContextFrankSlackOwnerControlOnboardingPreflight(ec ExecutionContext) (ResolvedExecutionContextFrankSlackOwnerControlOnboardingPreflight, error) {
	if ec.Step == nil {
		return ResolvedExecutionContextFrankSlackOwnerControlOnboardingPreflight{}, fmt.Errorf("execution context step is required")
	}
	if !DeclaresFrankSlackOwnerControlOnboarding(*ec.Step) {
		return ResolvedExecutionContextFrankSlackOwnerControlOnboardingPreflight{}, nil
	}
	if strings.TrimSpace(ec.MissionStoreRoot) == "" {
		return ResolvedExecutionContextFrankSlackOwnerControlOnboardingPreflight{}, fmt.Errorf("mission store root is required to resolve Frank object refs")
	}

	bundle, ok, err := ResolveExecutionContextFrankSlackOwnerControlOnboardingBundle(ec)
	if err != nil {
		return ResolvedExecutionContextFrankSlackOwnerControlOnboardingPreflight{}, err
	}
	if !ok {
		return ResolvedExecutionContextFrankSlackOwnerControlOnboardingPreflight{}, nil
	}

	return ResolvedExecutionContextFrankSlackOwnerControlOnboardingPreflight{
		Identity: &bundle.Identity,
		Account:  &bundle.Account,
	}, nil
}

func normalizeFrankSlackOwnerControlIdentity(config *FrankSlackOwnerControlIdentity) *FrankSlackOwnerControlIdentity {
	if config == nil {
		return nil
	}
	normalized := *config
	normalized.TeamID = strings.TrimSpace(normalized.TeamID)
	normalized.UserID = strings.TrimSpace(normalized.UserID)
	return &normalized
}

func normalizeFrankSlackOwnerControlAccount(config *FrankSlackOwnerControlAccount) *FrankSlackOwnerControlAccount {
	if config == nil {
		return nil
	}
	normalized := *config
	normalized.BotID = strings.TrimSpace(normalized.BotID)
	return &normalized
}

func validateFrankSlackOwnerControlIdentity(config FrankSlackOwnerControlIdentity) error {
	if teamID := strings.TrimSpace(config.TeamID); teamID == "" && strings.TrimSpace(config.UserID) != "" {
		return fmt.Errorf("mission store Frank identity slack_owner_control.user_id requires slack_owner_control.team_id")
	}
	if userID := strings.TrimSpace(config.UserID); userID == "" && strings.TrimSpace(config.TeamID) != "" {
		return fmt.Errorf("mission store Frank identity slack_owner_control.team_id requires slack_owner_control.user_id")
	}
	return nil
}

func validateFrankSlackOwnerControlAccount(config FrankSlackOwnerControlAccount) error {
	_ = config
	return nil
}

func normalizeResolvedExecutionContextFrankSlackOwnerControlOnboardingBundle(bundle ResolvedExecutionContextFrankSlackOwnerControlOnboardingBundle) ResolvedExecutionContextFrankSlackOwnerControlOnboardingBundle {
	bundle.Identity.ZohoMailbox = normalizeFrankZohoMailboxIdentity(bundle.Identity.ZohoMailbox)
	bundle.Identity.TelegramOwnerControl = normalizeFrankTelegramOwnerControlIdentity(bundle.Identity.TelegramOwnerControl)
	bundle.Identity.SlackOwnerControl = normalizeFrankSlackOwnerControlIdentity(bundle.Identity.SlackOwnerControl)
	bundle.Identity.IdentityMode = NormalizeIdentityMode(bundle.Identity.IdentityMode)
	bundle.Identity.CreatedAt = bundle.Identity.CreatedAt.UTC()
	bundle.Identity.UpdatedAt = bundle.Identity.UpdatedAt.UTC()

	bundle.Account.ZohoMailbox = normalizeFrankZohoMailboxAccount(bundle.Account.ZohoMailbox)
	bundle.Account.TelegramOwnerControl = normalizeFrankTelegramOwnerControlAccount(bundle.Account.TelegramOwnerControl)
	bundle.Account.SlackOwnerControl = normalizeFrankSlackOwnerControlAccount(bundle.Account.SlackOwnerControl)
	bundle.Account.CreatedAt = bundle.Account.CreatedAt.UTC()
	bundle.Account.UpdatedAt = bundle.Account.UpdatedAt.UTC()

	return bundle
}

func validateResolvedExecutionContextFrankSlackOwnerControlOnboardingBundle(bundle ResolvedExecutionContextFrankSlackOwnerControlOnboardingBundle) error {
	if err := ValidateFrankIdentityRecord(bundle.Identity); err != nil {
		return err
	}
	if err := ValidateFrankAccountRecord(bundle.Account); err != nil {
		return err
	}
	if bundle.Identity.SlackOwnerControl == nil {
		return fmt.Errorf("mission store Frank slack owner-control onboarding requires slack_owner_control identity record")
	}
	if bundle.Account.SlackOwnerControl == nil {
		return fmt.Errorf("mission store Frank slack owner-control onboarding requires slack_owner_control account record")
	}
	if NormalizeIdentityMode(bundle.Identity.IdentityMode) != IdentityModeOwnerOnlyControl {
		return fmt.Errorf(
			"mission store Frank slack owner-control onboarding identity %q identity_mode %q must be %q",
			bundle.Identity.IdentityID,
			bundle.Identity.IdentityMode,
			IdentityModeOwnerOnlyControl,
		)
	}
	if strings.TrimSpace(bundle.Account.IdentityID) != strings.TrimSpace(bundle.Identity.IdentityID) {
		return fmt.Errorf(
			"mission store Frank slack owner-control onboarding account %q must link identity_id %q, got %q",
			bundle.Account.AccountID,
			bundle.Identity.IdentityID,
			bundle.Account.IdentityID,
		)
	}
	if strings.TrimSpace(bundle.Account.ProviderOrPlatformID) != strings.TrimSpace(bundle.Identity.ProviderOrPlatformID) {
		return fmt.Errorf(
			"mission store Frank slack owner-control onboarding account %q provider_or_platform_id %q does not match identity %q provider_or_platform_id %q",
			bundle.Account.AccountID,
			bundle.Account.ProviderOrPlatformID,
			bundle.Identity.IdentityID,
			bundle.Identity.ProviderOrPlatformID,
		)
	}
	if bundle.Provider.TargetClass != EligibilityTargetKindProvider {
		return fmt.Errorf("mission store Frank slack owner-control onboarding provider %q target_class %q must be %q", bundle.Provider.PlatformID, bundle.Provider.TargetClass, EligibilityTargetKindProvider)
	}
	if bundle.AccountClass.TargetClass != EligibilityTargetKindAccountClass {
		return fmt.Errorf("mission store Frank slack owner-control onboarding account-class %q target_class %q must be %q", bundle.AccountClass.PlatformID, bundle.AccountClass.TargetClass, EligibilityTargetKindAccountClass)
	}
	return nil
}
