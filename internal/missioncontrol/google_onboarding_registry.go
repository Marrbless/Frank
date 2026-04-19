package missioncontrol

import (
	"fmt"
	"net/mail"
	"strings"
)

type ResolvedExecutionContextFrankGoogleOnboardingBundle struct {
	Identity      FrankIdentityRecord    `json:"identity"`
	Account       FrankAccountRecord     `json:"account"`
	Provider      PlatformRecord         `json:"provider"`
	ProviderCheck EligibilityCheckRecord `json:"provider_check"`
	AccountClass  PlatformRecord         `json:"account_class"`
	AccountCheck  EligibilityCheckRecord `json:"account_check"`
}

type ResolvedExecutionContextFrankGoogleOnboardingPreflight struct {
	Identity *FrankIdentityRecord `json:"identity,omitempty"`
	Account  *FrankAccountRecord  `json:"account,omitempty"`
}

func DeclaresFrankGoogleOnboarding(step Step) bool {
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

func ResolveExecutionContextFrankGoogleOnboardingBundle(ec ExecutionContext) (ResolvedExecutionContextFrankGoogleOnboardingBundle, bool, error) {
	resolved, err := ResolveExecutionContextFrankRegistryObjectRefs(ec)
	if err != nil {
		return ResolvedExecutionContextFrankGoogleOnboardingBundle{}, false, err
	}

	identities := make([]FrankIdentityRecord, 0, len(resolved.Identities))
	for _, identity := range resolved.Identities {
		if identity.Google != nil {
			identities = append(identities, identity)
		}
	}
	accounts := make([]FrankAccountRecord, 0, len(resolved.Accounts))
	for _, account := range resolved.Accounts {
		if account.Google != nil {
			accounts = append(accounts, account)
		}
	}

	if len(identities) == 0 && len(accounts) == 0 {
		return ResolvedExecutionContextFrankGoogleOnboardingBundle{}, false, nil
	}
	if len(identities) != 1 {
		return ResolvedExecutionContextFrankGoogleOnboardingBundle{}, false, fmt.Errorf("execution context Frank object refs must resolve exactly one google identity, got %d", len(identities))
	}
	if len(accounts) != 1 {
		return ResolvedExecutionContextFrankGoogleOnboardingBundle{}, false, fmt.Errorf("execution context Frank object refs must resolve exactly one google account, got %d", len(accounts))
	}

	identity := identities[0]
	account := accounts[0]
	if strings.TrimSpace(account.IdentityID) != strings.TrimSpace(identity.IdentityID) {
		return ResolvedExecutionContextFrankGoogleOnboardingBundle{}, false, fmt.Errorf(
			"execution context google account %q must link identity_id %q, got %q",
			account.AccountID,
			identity.IdentityID,
			account.IdentityID,
		)
	}
	if strings.TrimSpace(account.ProviderOrPlatformID) != strings.TrimSpace(identity.ProviderOrPlatformID) {
		return ResolvedExecutionContextFrankGoogleOnboardingBundle{}, false, fmt.Errorf(
			"execution context google account %q provider_or_platform_id %q does not match identity %q provider_or_platform_id %q",
			account.AccountID,
			account.ProviderOrPlatformID,
			identity.IdentityID,
			identity.ProviderOrPlatformID,
		)
	}
	if identity.EligibilityTargetRef.Kind != EligibilityTargetKindProvider {
		return ResolvedExecutionContextFrankGoogleOnboardingBundle{}, false, fmt.Errorf(
			"execution context google identity %q eligibility_target_ref.kind %q must be %q",
			identity.IdentityID,
			identity.EligibilityTargetRef.Kind,
			EligibilityTargetKindProvider,
		)
	}
	if account.EligibilityTargetRef.Kind != EligibilityTargetKindAccountClass {
		return ResolvedExecutionContextFrankGoogleOnboardingBundle{}, false, fmt.Errorf(
			"execution context google account %q eligibility_target_ref.kind %q must be %q",
			account.AccountID,
			account.EligibilityTargetRef.Kind,
			EligibilityTargetKindAccountClass,
		)
	}

	provider, err := LoadPlatformRecord(ec.MissionStoreRoot, identity.EligibilityTargetRef.RegistryID)
	if err != nil {
		return ResolvedExecutionContextFrankGoogleOnboardingBundle{}, false, fmt.Errorf("execution context google identity %q provider target %q: %w", identity.IdentityID, identity.EligibilityTargetRef.RegistryID, err)
	}
	if provider.TargetClass != EligibilityTargetKindProvider {
		return ResolvedExecutionContextFrankGoogleOnboardingBundle{}, false, fmt.Errorf("execution context google provider %q target_class %q must be %q", provider.PlatformID, provider.TargetClass, EligibilityTargetKindProvider)
	}
	providerCheck, err := LoadEligibilityCheckRecord(ec.MissionStoreRoot, provider.LastCheckID)
	if err != nil {
		return ResolvedExecutionContextFrankGoogleOnboardingBundle{}, false, fmt.Errorf("execution context google provider %q last_check_id %q: %w", provider.PlatformID, provider.LastCheckID, err)
	}

	accountClass, err := LoadPlatformRecord(ec.MissionStoreRoot, account.EligibilityTargetRef.RegistryID)
	if err != nil {
		return ResolvedExecutionContextFrankGoogleOnboardingBundle{}, false, fmt.Errorf("execution context google account %q account-class target %q: %w", account.AccountID, account.EligibilityTargetRef.RegistryID, err)
	}
	if accountClass.TargetClass != EligibilityTargetKindAccountClass {
		return ResolvedExecutionContextFrankGoogleOnboardingBundle{}, false, fmt.Errorf("execution context google account-class %q target_class %q must be %q", accountClass.PlatformID, accountClass.TargetClass, EligibilityTargetKindAccountClass)
	}
	accountCheck, err := LoadEligibilityCheckRecord(ec.MissionStoreRoot, accountClass.LastCheckID)
	if err != nil {
		return ResolvedExecutionContextFrankGoogleOnboardingBundle{}, false, fmt.Errorf("execution context google account-class %q last_check_id %q: %w", accountClass.PlatformID, accountClass.LastCheckID, err)
	}

	return ResolvedExecutionContextFrankGoogleOnboardingBundle{
		Identity:      identity,
		Account:       account,
		Provider:      provider,
		ProviderCheck: providerCheck,
		AccountClass:  accountClass,
		AccountCheck:  accountCheck,
	}, true, nil
}

func ResolveExecutionContextFrankGoogleOnboardingPreflight(ec ExecutionContext) (ResolvedExecutionContextFrankGoogleOnboardingPreflight, error) {
	if ec.Step == nil {
		return ResolvedExecutionContextFrankGoogleOnboardingPreflight{}, fmt.Errorf("execution context step is required")
	}
	if !DeclaresFrankGoogleOnboarding(*ec.Step) {
		return ResolvedExecutionContextFrankGoogleOnboardingPreflight{}, nil
	}
	if strings.TrimSpace(ec.MissionStoreRoot) == "" {
		return ResolvedExecutionContextFrankGoogleOnboardingPreflight{}, fmt.Errorf("mission store root is required to resolve Frank object refs")
	}

	bundle, ok, err := ResolveExecutionContextFrankGoogleOnboardingBundle(ec)
	if err != nil {
		return ResolvedExecutionContextFrankGoogleOnboardingPreflight{}, err
	}
	if !ok {
		return ResolvedExecutionContextFrankGoogleOnboardingPreflight{}, nil
	}

	return ResolvedExecutionContextFrankGoogleOnboardingPreflight{
		Identity: &bundle.Identity,
		Account:  &bundle.Account,
	}, nil
}

func normalizeFrankGoogleIdentity(config *FrankGoogleIdentity) *FrankGoogleIdentity {
	if config == nil {
		return nil
	}
	normalized := *config
	normalized.GoogleSub = strings.TrimSpace(normalized.GoogleSub)
	normalized.Email = strings.TrimSpace(normalized.Email)
	normalized.Name = strings.TrimSpace(normalized.Name)
	normalized.PictureURL = strings.TrimSpace(normalized.PictureURL)
	return &normalized
}

func normalizeFrankGoogleAccount(config *FrankGoogleAccount) *FrankGoogleAccount {
	if config == nil {
		return nil
	}
	normalized := *config
	normalized.OAuthClientIDEnvVarRef = strings.TrimSpace(normalized.OAuthClientIDEnvVarRef)
	normalized.OAuthAccessTokenEnvVarRef = strings.TrimSpace(normalized.OAuthAccessTokenEnvVarRef)
	return &normalized
}

func validateFrankGoogleIdentity(config FrankGoogleIdentity) error {
	if email := strings.TrimSpace(config.Email); email != "" {
		parsed, err := mail.ParseAddress(email)
		if err != nil || parsed == nil || parsed.Name != "" || parsed.Address != email {
			return fmt.Errorf("mission store Frank identity google.email must be a bare email address when present")
		}
	}
	return nil
}

func validateFrankGoogleAccount(config FrankGoogleAccount) error {
	if config.ConfirmedAuthenticated && strings.TrimSpace(config.OAuthAccessTokenEnvVarRef) == "" {
		return fmt.Errorf("mission store Frank account google.confirmed_authenticated requires google.oauth_access_token_env_var_ref")
	}
	return nil
}

func normalizeResolvedExecutionContextFrankGoogleOnboardingBundle(bundle ResolvedExecutionContextFrankGoogleOnboardingBundle) ResolvedExecutionContextFrankGoogleOnboardingBundle {
	bundle.Identity.ZohoMailbox = normalizeFrankZohoMailboxIdentity(bundle.Identity.ZohoMailbox)
	bundle.Identity.TelegramOwnerControl = normalizeFrankTelegramOwnerControlIdentity(bundle.Identity.TelegramOwnerControl)
	bundle.Identity.SlackOwnerControl = normalizeFrankSlackOwnerControlIdentity(bundle.Identity.SlackOwnerControl)
	bundle.Identity.DiscordOwnerControl = normalizeFrankDiscordOwnerControlIdentity(bundle.Identity.DiscordOwnerControl)
	bundle.Identity.WhatsAppOwnerControl = normalizeFrankWhatsAppOwnerControlIdentity(bundle.Identity.WhatsAppOwnerControl)
	bundle.Identity.GitHub = normalizeFrankGitHubIdentity(bundle.Identity.GitHub)
	bundle.Identity.Stripe = normalizeFrankStripeIdentity(bundle.Identity.Stripe)
	bundle.Identity.PayPal = normalizeFrankPayPalIdentity(bundle.Identity.PayPal)
	bundle.Identity.Google = normalizeFrankGoogleIdentity(bundle.Identity.Google)
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
	bundle.Account.Google = normalizeFrankGoogleAccount(bundle.Account.Google)
	bundle.Account.CreatedAt = bundle.Account.CreatedAt.UTC()
	bundle.Account.UpdatedAt = bundle.Account.UpdatedAt.UTC()

	return bundle
}

func validateResolvedExecutionContextFrankGoogleOnboardingBundle(bundle ResolvedExecutionContextFrankGoogleOnboardingBundle) error {
	if err := ValidateFrankIdentityRecord(bundle.Identity); err != nil {
		return err
	}
	if err := ValidateFrankAccountRecord(bundle.Account); err != nil {
		return err
	}
	if bundle.Identity.Google == nil {
		return fmt.Errorf("mission store Frank google onboarding requires google identity record")
	}
	if bundle.Account.Google == nil {
		return fmt.Errorf("mission store Frank google onboarding requires google account record")
	}
	if NormalizeIdentityMode(bundle.Identity.IdentityMode) != IdentityModeAgentAlias {
		return fmt.Errorf(
			"mission store Frank google onboarding identity %q identity_mode %q must be %q",
			bundle.Identity.IdentityID,
			bundle.Identity.IdentityMode,
			IdentityModeAgentAlias,
		)
	}
	if strings.TrimSpace(bundle.Account.IdentityID) != strings.TrimSpace(bundle.Identity.IdentityID) {
		return fmt.Errorf(
			"mission store Frank google onboarding account %q must link identity_id %q, got %q",
			bundle.Account.AccountID,
			bundle.Identity.IdentityID,
			bundle.Account.IdentityID,
		)
	}
	if strings.TrimSpace(bundle.Account.ProviderOrPlatformID) != strings.TrimSpace(bundle.Identity.ProviderOrPlatformID) {
		return fmt.Errorf(
			"mission store Frank google onboarding account %q provider_or_platform_id %q does not match identity %q provider_or_platform_id %q",
			bundle.Account.AccountID,
			bundle.Account.ProviderOrPlatformID,
			bundle.Identity.IdentityID,
			bundle.Identity.ProviderOrPlatformID,
		)
	}
	if bundle.Provider.TargetClass != EligibilityTargetKindProvider {
		return fmt.Errorf("mission store Frank google onboarding provider %q target_class %q must be %q", bundle.Provider.PlatformID, bundle.Provider.TargetClass, EligibilityTargetKindProvider)
	}
	if bundle.AccountClass.TargetClass != EligibilityTargetKindAccountClass {
		return fmt.Errorf("mission store Frank google onboarding account-class %q target_class %q must be %q", bundle.AccountClass.PlatformID, bundle.AccountClass.TargetClass, EligibilityTargetKindAccountClass)
	}
	if bundle.Account.Google.ConfirmedAuthenticated {
		if strings.TrimSpace(bundle.Identity.Google.GoogleSub) == "" {
			return fmt.Errorf("mission store Frank google onboarding confirmed identity requires google.google_sub")
		}
		if strings.TrimSpace(bundle.Account.Google.OAuthAccessTokenEnvVarRef) == "" {
			return fmt.Errorf("mission store Frank google onboarding confirmed account requires google.oauth_access_token_env_var_ref")
		}
	}
	return nil
}
