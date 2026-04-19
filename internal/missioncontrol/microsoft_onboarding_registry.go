package missioncontrol

import (
	"fmt"
	"strings"
)

type ResolvedExecutionContextFrankMicrosoftOnboardingBundle struct {
	Identity      FrankIdentityRecord    `json:"identity"`
	Account       FrankAccountRecord     `json:"account"`
	Provider      PlatformRecord         `json:"provider"`
	ProviderCheck EligibilityCheckRecord `json:"provider_check"`
	AccountClass  PlatformRecord         `json:"account_class"`
	AccountCheck  EligibilityCheckRecord `json:"account_check"`
}

type ResolvedExecutionContextFrankMicrosoftOnboardingPreflight struct {
	Identity *FrankIdentityRecord `json:"identity,omitempty"`
	Account  *FrankAccountRecord  `json:"account,omitempty"`
}

func DeclaresFrankMicrosoftOnboarding(step Step) bool {
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

func ResolveExecutionContextFrankMicrosoftOnboardingBundle(ec ExecutionContext) (ResolvedExecutionContextFrankMicrosoftOnboardingBundle, bool, error) {
	resolved, err := ResolveExecutionContextFrankRegistryObjectRefs(ec)
	if err != nil {
		return ResolvedExecutionContextFrankMicrosoftOnboardingBundle{}, false, err
	}

	identities := make([]FrankIdentityRecord, 0, len(resolved.Identities))
	for _, identity := range resolved.Identities {
		if identity.Microsoft != nil {
			identities = append(identities, identity)
		}
	}
	accounts := make([]FrankAccountRecord, 0, len(resolved.Accounts))
	for _, account := range resolved.Accounts {
		if account.Microsoft != nil {
			accounts = append(accounts, account)
		}
	}

	if len(identities) == 0 && len(accounts) == 0 {
		return ResolvedExecutionContextFrankMicrosoftOnboardingBundle{}, false, nil
	}
	if len(identities) != 1 {
		return ResolvedExecutionContextFrankMicrosoftOnboardingBundle{}, false, fmt.Errorf("execution context Frank object refs must resolve exactly one microsoft identity, got %d", len(identities))
	}
	if len(accounts) != 1 {
		return ResolvedExecutionContextFrankMicrosoftOnboardingBundle{}, false, fmt.Errorf("execution context Frank object refs must resolve exactly one microsoft account, got %d", len(accounts))
	}

	identity := identities[0]
	account := accounts[0]
	if strings.TrimSpace(account.IdentityID) != strings.TrimSpace(identity.IdentityID) {
		return ResolvedExecutionContextFrankMicrosoftOnboardingBundle{}, false, fmt.Errorf(
			"execution context microsoft account %q must link identity_id %q, got %q",
			account.AccountID,
			identity.IdentityID,
			account.IdentityID,
		)
	}
	if strings.TrimSpace(account.ProviderOrPlatformID) != strings.TrimSpace(identity.ProviderOrPlatformID) {
		return ResolvedExecutionContextFrankMicrosoftOnboardingBundle{}, false, fmt.Errorf(
			"execution context microsoft account %q provider_or_platform_id %q does not match identity %q provider_or_platform_id %q",
			account.AccountID,
			account.ProviderOrPlatformID,
			identity.IdentityID,
			identity.ProviderOrPlatformID,
		)
	}
	if identity.EligibilityTargetRef.Kind != EligibilityTargetKindProvider {
		return ResolvedExecutionContextFrankMicrosoftOnboardingBundle{}, false, fmt.Errorf(
			"execution context microsoft identity %q eligibility_target_ref.kind %q must be %q",
			identity.IdentityID,
			identity.EligibilityTargetRef.Kind,
			EligibilityTargetKindProvider,
		)
	}
	if account.EligibilityTargetRef.Kind != EligibilityTargetKindAccountClass {
		return ResolvedExecutionContextFrankMicrosoftOnboardingBundle{}, false, fmt.Errorf(
			"execution context microsoft account %q eligibility_target_ref.kind %q must be %q",
			account.AccountID,
			account.EligibilityTargetRef.Kind,
			EligibilityTargetKindAccountClass,
		)
	}

	provider, err := LoadPlatformRecord(ec.MissionStoreRoot, identity.EligibilityTargetRef.RegistryID)
	if err != nil {
		return ResolvedExecutionContextFrankMicrosoftOnboardingBundle{}, false, fmt.Errorf("execution context microsoft identity %q provider target %q: %w", identity.IdentityID, identity.EligibilityTargetRef.RegistryID, err)
	}
	if provider.TargetClass != EligibilityTargetKindProvider {
		return ResolvedExecutionContextFrankMicrosoftOnboardingBundle{}, false, fmt.Errorf("execution context microsoft provider %q target_class %q must be %q", provider.PlatformID, provider.TargetClass, EligibilityTargetKindProvider)
	}
	providerCheck, err := LoadEligibilityCheckRecord(ec.MissionStoreRoot, provider.LastCheckID)
	if err != nil {
		return ResolvedExecutionContextFrankMicrosoftOnboardingBundle{}, false, fmt.Errorf("execution context microsoft provider %q last_check_id %q: %w", provider.PlatformID, provider.LastCheckID, err)
	}

	accountClass, err := LoadPlatformRecord(ec.MissionStoreRoot, account.EligibilityTargetRef.RegistryID)
	if err != nil {
		return ResolvedExecutionContextFrankMicrosoftOnboardingBundle{}, false, fmt.Errorf("execution context microsoft account %q account-class target %q: %w", account.AccountID, account.EligibilityTargetRef.RegistryID, err)
	}
	if accountClass.TargetClass != EligibilityTargetKindAccountClass {
		return ResolvedExecutionContextFrankMicrosoftOnboardingBundle{}, false, fmt.Errorf("execution context microsoft account-class %q target_class %q must be %q", accountClass.PlatformID, accountClass.TargetClass, EligibilityTargetKindAccountClass)
	}
	accountCheck, err := LoadEligibilityCheckRecord(ec.MissionStoreRoot, accountClass.LastCheckID)
	if err != nil {
		return ResolvedExecutionContextFrankMicrosoftOnboardingBundle{}, false, fmt.Errorf("execution context microsoft account-class %q last_check_id %q: %w", accountClass.PlatformID, accountClass.LastCheckID, err)
	}

	return ResolvedExecutionContextFrankMicrosoftOnboardingBundle{
		Identity:      identity,
		Account:       account,
		Provider:      provider,
		ProviderCheck: providerCheck,
		AccountClass:  accountClass,
		AccountCheck:  accountCheck,
	}, true, nil
}

func ResolveExecutionContextFrankMicrosoftOnboardingPreflight(ec ExecutionContext) (ResolvedExecutionContextFrankMicrosoftOnboardingPreflight, error) {
	if ec.Step == nil {
		return ResolvedExecutionContextFrankMicrosoftOnboardingPreflight{}, fmt.Errorf("execution context step is required")
	}
	if !DeclaresFrankMicrosoftOnboarding(*ec.Step) {
		return ResolvedExecutionContextFrankMicrosoftOnboardingPreflight{}, nil
	}
	if strings.TrimSpace(ec.MissionStoreRoot) == "" {
		return ResolvedExecutionContextFrankMicrosoftOnboardingPreflight{}, fmt.Errorf("mission store root is required to resolve Frank object refs")
	}

	bundle, ok, err := ResolveExecutionContextFrankMicrosoftOnboardingBundle(ec)
	if err != nil {
		return ResolvedExecutionContextFrankMicrosoftOnboardingPreflight{}, err
	}
	if !ok {
		return ResolvedExecutionContextFrankMicrosoftOnboardingPreflight{}, nil
	}

	return ResolvedExecutionContextFrankMicrosoftOnboardingPreflight{
		Identity: &bundle.Identity,
		Account:  &bundle.Account,
	}, nil
}

func normalizeResolvedExecutionContextFrankMicrosoftOnboardingBundle(bundle ResolvedExecutionContextFrankMicrosoftOnboardingBundle) ResolvedExecutionContextFrankMicrosoftOnboardingBundle {
	bundle.Identity.ZohoMailbox = normalizeFrankZohoMailboxIdentity(bundle.Identity.ZohoMailbox)
	bundle.Identity.TelegramOwnerControl = normalizeFrankTelegramOwnerControlIdentity(bundle.Identity.TelegramOwnerControl)
	bundle.Identity.SlackOwnerControl = normalizeFrankSlackOwnerControlIdentity(bundle.Identity.SlackOwnerControl)
	bundle.Identity.DiscordOwnerControl = normalizeFrankDiscordOwnerControlIdentity(bundle.Identity.DiscordOwnerControl)
	bundle.Identity.WhatsAppOwnerControl = normalizeFrankWhatsAppOwnerControlIdentity(bundle.Identity.WhatsAppOwnerControl)
	bundle.Identity.GitHub = normalizeFrankGitHubIdentity(bundle.Identity.GitHub)
	bundle.Identity.Stripe = normalizeFrankStripeIdentity(bundle.Identity.Stripe)
	bundle.Identity.PayPal = normalizeFrankPayPalIdentity(bundle.Identity.PayPal)
	bundle.Identity.Google = normalizeFrankGoogleIdentity(bundle.Identity.Google)
	bundle.Identity.Microsoft = normalizeFrankMicrosoftIdentity(bundle.Identity.Microsoft)
	bundle.Identity.LinkedIn = normalizeFrankLinkedInIdentity(bundle.Identity.LinkedIn)
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
	bundle.Account.Microsoft = normalizeFrankMicrosoftAccount(bundle.Account.Microsoft)
	bundle.Account.LinkedIn = normalizeFrankLinkedInAccount(bundle.Account.LinkedIn)
	bundle.Account.CreatedAt = bundle.Account.CreatedAt.UTC()
	bundle.Account.UpdatedAt = bundle.Account.UpdatedAt.UTC()

	return bundle
}

func validateResolvedExecutionContextFrankMicrosoftOnboardingBundle(bundle ResolvedExecutionContextFrankMicrosoftOnboardingBundle) error {
	if err := ValidateFrankIdentityRecord(bundle.Identity); err != nil {
		return err
	}
	if err := ValidateFrankAccountRecord(bundle.Account); err != nil {
		return err
	}
	if bundle.Identity.Microsoft == nil {
		return fmt.Errorf("mission store Frank microsoft onboarding requires microsoft identity record")
	}
	if bundle.Account.Microsoft == nil {
		return fmt.Errorf("mission store Frank microsoft onboarding requires microsoft account record")
	}
	if NormalizeIdentityMode(bundle.Identity.IdentityMode) != IdentityModeAgentAlias {
		return fmt.Errorf(
			"mission store Frank microsoft onboarding identity %q identity_mode %q must be %q",
			bundle.Identity.IdentityID,
			bundle.Identity.IdentityMode,
			IdentityModeAgentAlias,
		)
	}
	if strings.TrimSpace(bundle.Account.IdentityID) != strings.TrimSpace(bundle.Identity.IdentityID) {
		return fmt.Errorf(
			"mission store Frank microsoft onboarding account %q must link identity_id %q, got %q",
			bundle.Account.AccountID,
			bundle.Identity.IdentityID,
			bundle.Account.IdentityID,
		)
	}
	if strings.TrimSpace(bundle.Account.ProviderOrPlatformID) != strings.TrimSpace(bundle.Identity.ProviderOrPlatformID) {
		return fmt.Errorf(
			"mission store Frank microsoft onboarding account %q provider_or_platform_id %q does not match identity %q provider_or_platform_id %q",
			bundle.Account.AccountID,
			bundle.Account.ProviderOrPlatformID,
			bundle.Identity.IdentityID,
			bundle.Identity.ProviderOrPlatformID,
		)
	}
	return nil
}
