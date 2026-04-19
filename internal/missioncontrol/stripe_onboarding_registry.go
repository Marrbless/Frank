package missioncontrol

import (
	"fmt"
	"strings"
)

type ResolvedExecutionContextFrankStripeOnboardingBundle struct {
	Identity      FrankIdentityRecord    `json:"identity"`
	Account       FrankAccountRecord     `json:"account"`
	Provider      PlatformRecord         `json:"provider"`
	ProviderCheck EligibilityCheckRecord `json:"provider_check"`
	AccountClass  PlatformRecord         `json:"account_class"`
	AccountCheck  EligibilityCheckRecord `json:"account_check"`
}

type ResolvedExecutionContextFrankStripeOnboardingPreflight struct {
	Identity *FrankIdentityRecord `json:"identity,omitempty"`
	Account  *FrankAccountRecord  `json:"account,omitempty"`
}

func DeclaresFrankStripeOnboarding(step Step) bool {
	if NormalizeIdentityMode(step.IdentityMode) != IdentityModeAgentAlias {
		return false
	}

	hasIdentityRef := false
	hasAccountRef := false
	for _, ref := range step.FrankObjectRefs {
		switch NormalizeFrankRegistryObjectKind(ref.Kind) {
		case FrankRegistryObjectKindIdentity:
			hasIdentityRef = true
		case FrankRegistryObjectKindAccount:
			hasAccountRef = true
		}
	}
	return hasIdentityRef && hasAccountRef
}

func ResolveExecutionContextFrankStripeOnboardingBundle(ec ExecutionContext) (ResolvedExecutionContextFrankStripeOnboardingBundle, bool, error) {
	resolved, err := ResolveExecutionContextFrankRegistryObjectRefs(ec)
	if err != nil {
		return ResolvedExecutionContextFrankStripeOnboardingBundle{}, false, err
	}

	identities := make([]FrankIdentityRecord, 0, len(resolved.Identities))
	for _, identity := range resolved.Identities {
		if identity.Stripe != nil {
			identities = append(identities, identity)
		}
	}
	accounts := make([]FrankAccountRecord, 0, len(resolved.Accounts))
	for _, account := range resolved.Accounts {
		if account.Stripe != nil {
			accounts = append(accounts, account)
		}
	}

	if len(identities) == 0 && len(accounts) == 0 {
		return ResolvedExecutionContextFrankStripeOnboardingBundle{}, false, nil
	}
	if len(identities) != 1 {
		return ResolvedExecutionContextFrankStripeOnboardingBundle{}, false, fmt.Errorf("execution context Frank object refs must resolve exactly one stripe identity, got %d", len(identities))
	}
	if len(accounts) != 1 {
		return ResolvedExecutionContextFrankStripeOnboardingBundle{}, false, fmt.Errorf("execution context Frank object refs must resolve exactly one stripe account, got %d", len(accounts))
	}

	identity := identities[0]
	account := accounts[0]
	if strings.TrimSpace(account.IdentityID) != strings.TrimSpace(identity.IdentityID) {
		return ResolvedExecutionContextFrankStripeOnboardingBundle{}, false, fmt.Errorf(
			"execution context stripe account %q must link identity_id %q, got %q",
			account.AccountID,
			identity.IdentityID,
			account.IdentityID,
		)
	}
	if strings.TrimSpace(account.ProviderOrPlatformID) != strings.TrimSpace(identity.ProviderOrPlatformID) {
		return ResolvedExecutionContextFrankStripeOnboardingBundle{}, false, fmt.Errorf(
			"execution context stripe account %q provider_or_platform_id %q does not match identity %q provider_or_platform_id %q",
			account.AccountID,
			account.ProviderOrPlatformID,
			identity.IdentityID,
			identity.ProviderOrPlatformID,
		)
	}
	if identity.EligibilityTargetRef.Kind != EligibilityTargetKindProvider {
		return ResolvedExecutionContextFrankStripeOnboardingBundle{}, false, fmt.Errorf(
			"execution context stripe identity %q eligibility_target_ref.kind %q must be %q",
			identity.IdentityID,
			identity.EligibilityTargetRef.Kind,
			EligibilityTargetKindProvider,
		)
	}
	if account.EligibilityTargetRef.Kind != EligibilityTargetKindAccountClass {
		return ResolvedExecutionContextFrankStripeOnboardingBundle{}, false, fmt.Errorf(
			"execution context stripe account %q eligibility_target_ref.kind %q must be %q",
			account.AccountID,
			account.EligibilityTargetRef.Kind,
			EligibilityTargetKindAccountClass,
		)
	}

	provider, err := LoadPlatformRecord(ec.MissionStoreRoot, identity.EligibilityTargetRef.RegistryID)
	if err != nil {
		return ResolvedExecutionContextFrankStripeOnboardingBundle{}, false, fmt.Errorf("execution context stripe identity %q provider target %q: %w", identity.IdentityID, identity.EligibilityTargetRef.RegistryID, err)
	}
	if provider.TargetClass != EligibilityTargetKindProvider {
		return ResolvedExecutionContextFrankStripeOnboardingBundle{}, false, fmt.Errorf("execution context stripe provider %q target_class %q must be %q", provider.PlatformID, provider.TargetClass, EligibilityTargetKindProvider)
	}
	providerCheck, err := LoadEligibilityCheckRecord(ec.MissionStoreRoot, provider.LastCheckID)
	if err != nil {
		return ResolvedExecutionContextFrankStripeOnboardingBundle{}, false, fmt.Errorf("execution context stripe provider %q last_check_id %q: %w", provider.PlatformID, provider.LastCheckID, err)
	}

	accountClass, err := LoadPlatformRecord(ec.MissionStoreRoot, account.EligibilityTargetRef.RegistryID)
	if err != nil {
		return ResolvedExecutionContextFrankStripeOnboardingBundle{}, false, fmt.Errorf("execution context stripe account %q account-class target %q: %w", account.AccountID, account.EligibilityTargetRef.RegistryID, err)
	}
	if accountClass.TargetClass != EligibilityTargetKindAccountClass {
		return ResolvedExecutionContextFrankStripeOnboardingBundle{}, false, fmt.Errorf("execution context stripe account-class %q target_class %q must be %q", accountClass.PlatformID, accountClass.TargetClass, EligibilityTargetKindAccountClass)
	}
	accountCheck, err := LoadEligibilityCheckRecord(ec.MissionStoreRoot, accountClass.LastCheckID)
	if err != nil {
		return ResolvedExecutionContextFrankStripeOnboardingBundle{}, false, fmt.Errorf("execution context stripe account-class %q last_check_id %q: %w", accountClass.PlatformID, accountClass.LastCheckID, err)
	}

	return ResolvedExecutionContextFrankStripeOnboardingBundle{
		Identity:      identity,
		Account:       account,
		Provider:      provider,
		ProviderCheck: providerCheck,
		AccountClass:  accountClass,
		AccountCheck:  accountCheck,
	}, true, nil
}

func ResolveExecutionContextFrankStripeOnboardingPreflight(ec ExecutionContext) (ResolvedExecutionContextFrankStripeOnboardingPreflight, error) {
	if ec.Step == nil {
		return ResolvedExecutionContextFrankStripeOnboardingPreflight{}, fmt.Errorf("execution context step is required")
	}
	if !DeclaresFrankStripeOnboarding(*ec.Step) {
		return ResolvedExecutionContextFrankStripeOnboardingPreflight{}, nil
	}
	if strings.TrimSpace(ec.MissionStoreRoot) == "" {
		return ResolvedExecutionContextFrankStripeOnboardingPreflight{}, fmt.Errorf("mission store root is required to resolve Frank object refs")
	}

	bundle, ok, err := ResolveExecutionContextFrankStripeOnboardingBundle(ec)
	if err != nil {
		return ResolvedExecutionContextFrankStripeOnboardingPreflight{}, err
	}
	if !ok {
		return ResolvedExecutionContextFrankStripeOnboardingPreflight{}, nil
	}

	return ResolvedExecutionContextFrankStripeOnboardingPreflight{
		Identity: &bundle.Identity,
		Account:  &bundle.Account,
	}, nil
}

func normalizeFrankStripeIdentity(config *FrankStripeIdentity) *FrankStripeIdentity {
	if config == nil {
		return nil
	}
	normalized := *config
	normalized.StripeAccountID = strings.TrimSpace(normalized.StripeAccountID)
	normalized.BusinessProfileName = strings.TrimSpace(normalized.BusinessProfileName)
	normalized.DashboardDisplayName = strings.TrimSpace(normalized.DashboardDisplayName)
	normalized.Country = strings.ToUpper(strings.TrimSpace(normalized.Country))
	normalized.DefaultCurrency = strings.ToLower(strings.TrimSpace(normalized.DefaultCurrency))
	return &normalized
}

func normalizeFrankStripeAccount(config *FrankStripeAccount) *FrankStripeAccount {
	if config == nil {
		return nil
	}
	normalized := *config
	normalized.SecretKeyEnvVarRef = strings.TrimSpace(normalized.SecretKeyEnvVarRef)
	normalized.RequirementsDisabledReason = strings.TrimSpace(normalized.RequirementsDisabledReason)
	return &normalized
}

func validateFrankStripeIdentity(config FrankStripeIdentity) error {
	if accountID := strings.TrimSpace(config.StripeAccountID); accountID != "" && !strings.HasPrefix(accountID, "acct_") {
		return fmt.Errorf("mission store Frank identity stripe.stripe_account_id must start with %q when present", "acct_")
	}
	return nil
}

func validateFrankStripeAccount(config FrankStripeAccount) error {
	if config.ConfirmedAuthenticated && strings.TrimSpace(config.SecretKeyEnvVarRef) == "" {
		return fmt.Errorf("mission store Frank account stripe.confirmed_authenticated requires stripe.secret_key_env_var_ref")
	}
	return nil
}

func normalizeResolvedExecutionContextFrankStripeOnboardingBundle(bundle ResolvedExecutionContextFrankStripeOnboardingBundle) ResolvedExecutionContextFrankStripeOnboardingBundle {
	bundle.Identity.ZohoMailbox = normalizeFrankZohoMailboxIdentity(bundle.Identity.ZohoMailbox)
	bundle.Identity.TelegramOwnerControl = normalizeFrankTelegramOwnerControlIdentity(bundle.Identity.TelegramOwnerControl)
	bundle.Identity.SlackOwnerControl = normalizeFrankSlackOwnerControlIdentity(bundle.Identity.SlackOwnerControl)
	bundle.Identity.DiscordOwnerControl = normalizeFrankDiscordOwnerControlIdentity(bundle.Identity.DiscordOwnerControl)
	bundle.Identity.WhatsAppOwnerControl = normalizeFrankWhatsAppOwnerControlIdentity(bundle.Identity.WhatsAppOwnerControl)
	bundle.Identity.GitHub = normalizeFrankGitHubIdentity(bundle.Identity.GitHub)
	bundle.Identity.Stripe = normalizeFrankStripeIdentity(bundle.Identity.Stripe)
	bundle.Identity.IdentityMode = NormalizeIdentityMode(bundle.Identity.IdentityMode)
	bundle.Identity.CreatedAt = bundle.Identity.CreatedAt.UTC()
	bundle.Identity.UpdatedAt = bundle.Identity.UpdatedAt.UTC()

	bundle.Account.ZohoMailbox = normalizeFrankZohoMailboxAccount(bundle.Account.ZohoMailbox)
	bundle.Account.TelegramOwnerControl = normalizeFrankTelegramOwnerControlAccount(bundle.Account.TelegramOwnerControl)
	bundle.Account.SlackOwnerControl = normalizeFrankSlackOwnerControlAccount(bundle.Account.SlackOwnerControl)
	bundle.Account.DiscordOwnerControl = normalizeFrankDiscordOwnerControlAccount(bundle.Account.DiscordOwnerControl)
	bundle.Account.WhatsAppOwnerControl = normalizeFrankWhatsAppOwnerControlAccount(bundle.Account.WhatsAppOwnerControl)
	bundle.Account.GitHub = normalizeFrankGitHubAccount(bundle.Account.GitHub)
	bundle.Account.Stripe = normalizeFrankStripeAccount(bundle.Account.Stripe)
	bundle.Account.CreatedAt = bundle.Account.CreatedAt.UTC()
	bundle.Account.UpdatedAt = bundle.Account.UpdatedAt.UTC()

	return bundle
}

func validateResolvedExecutionContextFrankStripeOnboardingBundle(bundle ResolvedExecutionContextFrankStripeOnboardingBundle) error {
	if err := ValidateFrankIdentityRecord(bundle.Identity); err != nil {
		return err
	}
	if err := ValidateFrankAccountRecord(bundle.Account); err != nil {
		return err
	}
	if bundle.Identity.Stripe == nil {
		return fmt.Errorf("mission store Frank stripe onboarding requires stripe identity record")
	}
	if bundle.Account.Stripe == nil {
		return fmt.Errorf("mission store Frank stripe onboarding requires stripe account record")
	}
	if NormalizeIdentityMode(bundle.Identity.IdentityMode) != IdentityModeAgentAlias {
		return fmt.Errorf(
			"mission store Frank stripe onboarding identity %q identity_mode %q must be %q",
			bundle.Identity.IdentityID,
			bundle.Identity.IdentityMode,
			IdentityModeAgentAlias,
		)
	}
	if strings.TrimSpace(bundle.Account.IdentityID) != strings.TrimSpace(bundle.Identity.IdentityID) {
		return fmt.Errorf(
			"mission store Frank stripe onboarding account %q must link identity_id %q, got %q",
			bundle.Account.AccountID,
			bundle.Identity.IdentityID,
			bundle.Account.IdentityID,
		)
	}
	if strings.TrimSpace(bundle.Account.ProviderOrPlatformID) != strings.TrimSpace(bundle.Identity.ProviderOrPlatformID) {
		return fmt.Errorf(
			"mission store Frank stripe onboarding account %q provider_or_platform_id %q does not match identity %q provider_or_platform_id %q",
			bundle.Account.AccountID,
			bundle.Account.ProviderOrPlatformID,
			bundle.Identity.IdentityID,
			bundle.Identity.ProviderOrPlatformID,
		)
	}
	if bundle.Provider.TargetClass != EligibilityTargetKindProvider {
		return fmt.Errorf("mission store Frank stripe onboarding provider %q target_class %q must be %q", bundle.Provider.PlatformID, bundle.Provider.TargetClass, EligibilityTargetKindProvider)
	}
	if bundle.AccountClass.TargetClass != EligibilityTargetKindAccountClass {
		return fmt.Errorf("mission store Frank stripe onboarding account-class %q target_class %q must be %q", bundle.AccountClass.PlatformID, bundle.AccountClass.TargetClass, EligibilityTargetKindAccountClass)
	}
	if bundle.Account.Stripe.ConfirmedAuthenticated {
		if strings.TrimSpace(bundle.Identity.Stripe.StripeAccountID) == "" {
			return fmt.Errorf("mission store Frank stripe onboarding confirmed identity requires stripe.stripe_account_id")
		}
		if strings.TrimSpace(bundle.Account.Stripe.SecretKeyEnvVarRef) == "" {
			return fmt.Errorf("mission store Frank stripe onboarding confirmed account requires stripe.secret_key_env_var_ref")
		}
	}
	return nil
}
