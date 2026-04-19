package missioncontrol

import (
	"fmt"
	"net/mail"
	"strings"
)

type ResolvedExecutionContextFrankPayPalOnboardingBundle struct {
	Identity      FrankIdentityRecord    `json:"identity"`
	Account       FrankAccountRecord     `json:"account"`
	Provider      PlatformRecord         `json:"provider"`
	ProviderCheck EligibilityCheckRecord `json:"provider_check"`
	AccountClass  PlatformRecord         `json:"account_class"`
	AccountCheck  EligibilityCheckRecord `json:"account_check"`
}

type ResolvedExecutionContextFrankPayPalOnboardingPreflight struct {
	Identity *FrankIdentityRecord `json:"identity,omitempty"`
	Account  *FrankAccountRecord  `json:"account,omitempty"`
}

func DeclaresFrankPayPalOnboarding(step Step) bool {
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

func ResolveExecutionContextFrankPayPalOnboardingBundle(ec ExecutionContext) (ResolvedExecutionContextFrankPayPalOnboardingBundle, bool, error) {
	resolved, err := ResolveExecutionContextFrankRegistryObjectRefs(ec)
	if err != nil {
		return ResolvedExecutionContextFrankPayPalOnboardingBundle{}, false, err
	}

	identities := make([]FrankIdentityRecord, 0, len(resolved.Identities))
	for _, identity := range resolved.Identities {
		if identity.PayPal != nil {
			identities = append(identities, identity)
		}
	}
	accounts := make([]FrankAccountRecord, 0, len(resolved.Accounts))
	for _, account := range resolved.Accounts {
		if account.PayPal != nil {
			accounts = append(accounts, account)
		}
	}

	if len(identities) == 0 && len(accounts) == 0 {
		return ResolvedExecutionContextFrankPayPalOnboardingBundle{}, false, nil
	}
	if len(identities) != 1 {
		return ResolvedExecutionContextFrankPayPalOnboardingBundle{}, false, fmt.Errorf("execution context Frank object refs must resolve exactly one paypal identity, got %d", len(identities))
	}
	if len(accounts) != 1 {
		return ResolvedExecutionContextFrankPayPalOnboardingBundle{}, false, fmt.Errorf("execution context Frank object refs must resolve exactly one paypal account, got %d", len(accounts))
	}

	identity := identities[0]
	account := accounts[0]
	if strings.TrimSpace(account.IdentityID) != strings.TrimSpace(identity.IdentityID) {
		return ResolvedExecutionContextFrankPayPalOnboardingBundle{}, false, fmt.Errorf(
			"execution context paypal account %q must link identity_id %q, got %q",
			account.AccountID,
			identity.IdentityID,
			account.IdentityID,
		)
	}
	if strings.TrimSpace(account.ProviderOrPlatformID) != strings.TrimSpace(identity.ProviderOrPlatformID) {
		return ResolvedExecutionContextFrankPayPalOnboardingBundle{}, false, fmt.Errorf(
			"execution context paypal account %q provider_or_platform_id %q does not match identity %q provider_or_platform_id %q",
			account.AccountID,
			account.ProviderOrPlatformID,
			identity.IdentityID,
			identity.ProviderOrPlatformID,
		)
	}
	if identity.EligibilityTargetRef.Kind != EligibilityTargetKindProvider {
		return ResolvedExecutionContextFrankPayPalOnboardingBundle{}, false, fmt.Errorf(
			"execution context paypal identity %q eligibility_target_ref.kind %q must be %q",
			identity.IdentityID,
			identity.EligibilityTargetRef.Kind,
			EligibilityTargetKindProvider,
		)
	}
	if account.EligibilityTargetRef.Kind != EligibilityTargetKindAccountClass {
		return ResolvedExecutionContextFrankPayPalOnboardingBundle{}, false, fmt.Errorf(
			"execution context paypal account %q eligibility_target_ref.kind %q must be %q",
			account.AccountID,
			account.EligibilityTargetRef.Kind,
			EligibilityTargetKindAccountClass,
		)
	}

	provider, err := LoadPlatformRecord(ec.MissionStoreRoot, identity.EligibilityTargetRef.RegistryID)
	if err != nil {
		return ResolvedExecutionContextFrankPayPalOnboardingBundle{}, false, fmt.Errorf("execution context paypal identity %q provider target %q: %w", identity.IdentityID, identity.EligibilityTargetRef.RegistryID, err)
	}
	if provider.TargetClass != EligibilityTargetKindProvider {
		return ResolvedExecutionContextFrankPayPalOnboardingBundle{}, false, fmt.Errorf("execution context paypal provider %q target_class %q must be %q", provider.PlatformID, provider.TargetClass, EligibilityTargetKindProvider)
	}
	providerCheck, err := LoadEligibilityCheckRecord(ec.MissionStoreRoot, provider.LastCheckID)
	if err != nil {
		return ResolvedExecutionContextFrankPayPalOnboardingBundle{}, false, fmt.Errorf("execution context paypal provider %q last_check_id %q: %w", provider.PlatformID, provider.LastCheckID, err)
	}

	accountClass, err := LoadPlatformRecord(ec.MissionStoreRoot, account.EligibilityTargetRef.RegistryID)
	if err != nil {
		return ResolvedExecutionContextFrankPayPalOnboardingBundle{}, false, fmt.Errorf("execution context paypal account %q account-class target %q: %w", account.AccountID, account.EligibilityTargetRef.RegistryID, err)
	}
	if accountClass.TargetClass != EligibilityTargetKindAccountClass {
		return ResolvedExecutionContextFrankPayPalOnboardingBundle{}, false, fmt.Errorf("execution context paypal account-class %q target_class %q must be %q", accountClass.PlatformID, accountClass.TargetClass, EligibilityTargetKindAccountClass)
	}
	accountCheck, err := LoadEligibilityCheckRecord(ec.MissionStoreRoot, accountClass.LastCheckID)
	if err != nil {
		return ResolvedExecutionContextFrankPayPalOnboardingBundle{}, false, fmt.Errorf("execution context paypal account-class %q last_check_id %q: %w", accountClass.PlatformID, accountClass.LastCheckID, err)
	}

	return ResolvedExecutionContextFrankPayPalOnboardingBundle{
		Identity:      identity,
		Account:       account,
		Provider:      provider,
		ProviderCheck: providerCheck,
		AccountClass:  accountClass,
		AccountCheck:  accountCheck,
	}, true, nil
}

func ResolveExecutionContextFrankPayPalOnboardingPreflight(ec ExecutionContext) (ResolvedExecutionContextFrankPayPalOnboardingPreflight, error) {
	if ec.Step == nil {
		return ResolvedExecutionContextFrankPayPalOnboardingPreflight{}, fmt.Errorf("execution context step is required")
	}
	if !DeclaresFrankPayPalOnboarding(*ec.Step) {
		return ResolvedExecutionContextFrankPayPalOnboardingPreflight{}, nil
	}
	if strings.TrimSpace(ec.MissionStoreRoot) == "" {
		return ResolvedExecutionContextFrankPayPalOnboardingPreflight{}, fmt.Errorf("mission store root is required to resolve Frank object refs")
	}

	bundle, ok, err := ResolveExecutionContextFrankPayPalOnboardingBundle(ec)
	if err != nil {
		return ResolvedExecutionContextFrankPayPalOnboardingPreflight{}, err
	}
	if !ok {
		return ResolvedExecutionContextFrankPayPalOnboardingPreflight{}, nil
	}

	return ResolvedExecutionContextFrankPayPalOnboardingPreflight{
		Identity: &bundle.Identity,
		Account:  &bundle.Account,
	}, nil
}

func normalizeFrankPayPalIdentity(config *FrankPayPalIdentity) *FrankPayPalIdentity {
	if config == nil {
		return nil
	}
	normalized := *config
	normalized.PayPalUserID = strings.TrimSpace(normalized.PayPalUserID)
	normalized.Sub = strings.TrimSpace(normalized.Sub)
	normalized.Email = strings.TrimSpace(normalized.Email)
	normalized.AccountType = strings.TrimSpace(normalized.AccountType)
	return &normalized
}

func normalizeFrankPayPalAccount(config *FrankPayPalAccount) *FrankPayPalAccount {
	if config == nil {
		return nil
	}
	normalized := *config
	normalized.ClientIDEnvVarRef = strings.TrimSpace(normalized.ClientIDEnvVarRef)
	normalized.ClientSecretEnvVarRef = strings.TrimSpace(normalized.ClientSecretEnvVarRef)
	normalized.Environment = strings.ToLower(strings.TrimSpace(normalized.Environment))
	return &normalized
}

func validateFrankPayPalIdentity(config FrankPayPalIdentity) error {
	if email := strings.TrimSpace(config.Email); email != "" {
		parsed, err := mail.ParseAddress(email)
		if err != nil || parsed == nil || parsed.Name != "" || parsed.Address != email {
			return fmt.Errorf("mission store Frank identity paypal.email must be a bare email address when present")
		}
	}
	return nil
}

func validateFrankPayPalAccount(config FrankPayPalAccount) error {
	if environment := strings.ToLower(strings.TrimSpace(config.Environment)); environment != "" && environment != "sandbox" && environment != "live" {
		return fmt.Errorf("mission store Frank account paypal.environment must be %q or %q when present", "sandbox", "live")
	}
	if config.ConfirmedAuthenticated {
		if strings.TrimSpace(config.ClientIDEnvVarRef) == "" {
			return fmt.Errorf("mission store Frank account paypal.confirmed_authenticated requires paypal.client_id_env_var_ref")
		}
		if strings.TrimSpace(config.ClientSecretEnvVarRef) == "" {
			return fmt.Errorf("mission store Frank account paypal.confirmed_authenticated requires paypal.client_secret_env_var_ref")
		}
		if strings.TrimSpace(config.Environment) == "" {
			return fmt.Errorf("mission store Frank account paypal.confirmed_authenticated requires paypal.environment")
		}
	}
	return nil
}

func normalizeResolvedExecutionContextFrankPayPalOnboardingBundle(bundle ResolvedExecutionContextFrankPayPalOnboardingBundle) ResolvedExecutionContextFrankPayPalOnboardingBundle {
	bundle.Identity.ZohoMailbox = normalizeFrankZohoMailboxIdentity(bundle.Identity.ZohoMailbox)
	bundle.Identity.TelegramOwnerControl = normalizeFrankTelegramOwnerControlIdentity(bundle.Identity.TelegramOwnerControl)
	bundle.Identity.SlackOwnerControl = normalizeFrankSlackOwnerControlIdentity(bundle.Identity.SlackOwnerControl)
	bundle.Identity.DiscordOwnerControl = normalizeFrankDiscordOwnerControlIdentity(bundle.Identity.DiscordOwnerControl)
	bundle.Identity.WhatsAppOwnerControl = normalizeFrankWhatsAppOwnerControlIdentity(bundle.Identity.WhatsAppOwnerControl)
	bundle.Identity.GitHub = normalizeFrankGitHubIdentity(bundle.Identity.GitHub)
	bundle.Identity.Stripe = normalizeFrankStripeIdentity(bundle.Identity.Stripe)
	bundle.Identity.PayPal = normalizeFrankPayPalIdentity(bundle.Identity.PayPal)
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
	bundle.Account.PayPal = normalizeFrankPayPalAccount(bundle.Account.PayPal)
	bundle.Account.CreatedAt = bundle.Account.CreatedAt.UTC()
	bundle.Account.UpdatedAt = bundle.Account.UpdatedAt.UTC()

	return bundle
}

func validateResolvedExecutionContextFrankPayPalOnboardingBundle(bundle ResolvedExecutionContextFrankPayPalOnboardingBundle) error {
	if err := ValidateFrankIdentityRecord(bundle.Identity); err != nil {
		return err
	}
	if err := ValidateFrankAccountRecord(bundle.Account); err != nil {
		return err
	}
	if bundle.Identity.PayPal == nil {
		return fmt.Errorf("mission store Frank paypal onboarding requires paypal identity record")
	}
	if bundle.Account.PayPal == nil {
		return fmt.Errorf("mission store Frank paypal onboarding requires paypal account record")
	}
	if NormalizeIdentityMode(bundle.Identity.IdentityMode) != IdentityModeAgentAlias {
		return fmt.Errorf(
			"mission store Frank paypal onboarding identity %q identity_mode %q must be %q",
			bundle.Identity.IdentityID,
			bundle.Identity.IdentityMode,
			IdentityModeAgentAlias,
		)
	}
	if strings.TrimSpace(bundle.Account.IdentityID) != strings.TrimSpace(bundle.Identity.IdentityID) {
		return fmt.Errorf(
			"mission store Frank paypal onboarding account %q must link identity_id %q, got %q",
			bundle.Account.AccountID,
			bundle.Identity.IdentityID,
			bundle.Account.IdentityID,
		)
	}
	if strings.TrimSpace(bundle.Account.ProviderOrPlatformID) != strings.TrimSpace(bundle.Identity.ProviderOrPlatformID) {
		return fmt.Errorf(
			"mission store Frank paypal onboarding account %q provider_or_platform_id %q does not match identity %q provider_or_platform_id %q",
			bundle.Account.AccountID,
			bundle.Account.ProviderOrPlatformID,
			bundle.Identity.IdentityID,
			bundle.Identity.ProviderOrPlatformID,
		)
	}
	if bundle.Provider.TargetClass != EligibilityTargetKindProvider {
		return fmt.Errorf("mission store Frank paypal onboarding provider %q target_class %q must be %q", bundle.Provider.PlatformID, bundle.Provider.TargetClass, EligibilityTargetKindProvider)
	}
	if bundle.AccountClass.TargetClass != EligibilityTargetKindAccountClass {
		return fmt.Errorf("mission store Frank paypal onboarding account-class %q target_class %q must be %q", bundle.AccountClass.PlatformID, bundle.AccountClass.TargetClass, EligibilityTargetKindAccountClass)
	}
	if bundle.Account.PayPal.ConfirmedAuthenticated {
		if strings.TrimSpace(bundle.Identity.PayPal.PayPalUserID) == "" && strings.TrimSpace(bundle.Identity.PayPal.Sub) == "" {
			return fmt.Errorf("mission store Frank paypal onboarding confirmed identity requires paypal.paypal_user_id or paypal.sub")
		}
		if strings.TrimSpace(bundle.Account.PayPal.ClientIDEnvVarRef) == "" {
			return fmt.Errorf("mission store Frank paypal onboarding confirmed account requires paypal.client_id_env_var_ref")
		}
		if strings.TrimSpace(bundle.Account.PayPal.ClientSecretEnvVarRef) == "" {
			return fmt.Errorf("mission store Frank paypal onboarding confirmed account requires paypal.client_secret_env_var_ref")
		}
		if strings.TrimSpace(bundle.Account.PayPal.Environment) == "" {
			return fmt.Errorf("mission store Frank paypal onboarding confirmed account requires paypal.environment")
		}
	}
	return nil
}
