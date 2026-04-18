package missioncontrol

import (
	"fmt"
	"net/mail"
	"strings"
)

type ResolvedExecutionContextFrankGitHubOnboardingBundle struct {
	Identity      FrankIdentityRecord    `json:"identity"`
	Account       FrankAccountRecord     `json:"account"`
	Provider      PlatformRecord         `json:"provider"`
	ProviderCheck EligibilityCheckRecord `json:"provider_check"`
	AccountClass  PlatformRecord         `json:"account_class"`
	AccountCheck  EligibilityCheckRecord `json:"account_check"`
}

type ResolvedExecutionContextFrankGitHubOnboardingPreflight struct {
	Identity *FrankIdentityRecord `json:"identity,omitempty"`
	Account  *FrankAccountRecord  `json:"account,omitempty"`
}

func DeclaresFrankGitHubOnboarding(step Step) bool {
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

func ResolveExecutionContextFrankGitHubOnboardingBundle(ec ExecutionContext) (ResolvedExecutionContextFrankGitHubOnboardingBundle, bool, error) {
	resolved, err := ResolveExecutionContextFrankRegistryObjectRefs(ec)
	if err != nil {
		return ResolvedExecutionContextFrankGitHubOnboardingBundle{}, false, err
	}

	identities := make([]FrankIdentityRecord, 0, len(resolved.Identities))
	for _, identity := range resolved.Identities {
		if identity.GitHub != nil {
			identities = append(identities, identity)
		}
	}
	accounts := make([]FrankAccountRecord, 0, len(resolved.Accounts))
	for _, account := range resolved.Accounts {
		if account.GitHub != nil {
			accounts = append(accounts, account)
		}
	}

	if len(identities) == 0 && len(accounts) == 0 {
		return ResolvedExecutionContextFrankGitHubOnboardingBundle{}, false, nil
	}
	if len(identities) != 1 {
		return ResolvedExecutionContextFrankGitHubOnboardingBundle{}, false, fmt.Errorf("execution context Frank object refs must resolve exactly one github identity, got %d", len(identities))
	}
	if len(accounts) != 1 {
		return ResolvedExecutionContextFrankGitHubOnboardingBundle{}, false, fmt.Errorf("execution context Frank object refs must resolve exactly one github account, got %d", len(accounts))
	}

	identity := identities[0]
	account := accounts[0]
	if strings.TrimSpace(account.IdentityID) != strings.TrimSpace(identity.IdentityID) {
		return ResolvedExecutionContextFrankGitHubOnboardingBundle{}, false, fmt.Errorf(
			"execution context github account %q must link identity_id %q, got %q",
			account.AccountID,
			identity.IdentityID,
			account.IdentityID,
		)
	}
	if strings.TrimSpace(account.ProviderOrPlatformID) != strings.TrimSpace(identity.ProviderOrPlatformID) {
		return ResolvedExecutionContextFrankGitHubOnboardingBundle{}, false, fmt.Errorf(
			"execution context github account %q provider_or_platform_id %q does not match identity %q provider_or_platform_id %q",
			account.AccountID,
			account.ProviderOrPlatformID,
			identity.IdentityID,
			identity.ProviderOrPlatformID,
		)
	}
	if identity.EligibilityTargetRef.Kind != EligibilityTargetKindProvider {
		return ResolvedExecutionContextFrankGitHubOnboardingBundle{}, false, fmt.Errorf(
			"execution context github identity %q eligibility_target_ref.kind %q must be %q",
			identity.IdentityID,
			identity.EligibilityTargetRef.Kind,
			EligibilityTargetKindProvider,
		)
	}
	if account.EligibilityTargetRef.Kind != EligibilityTargetKindAccountClass {
		return ResolvedExecutionContextFrankGitHubOnboardingBundle{}, false, fmt.Errorf(
			"execution context github account %q eligibility_target_ref.kind %q must be %q",
			account.AccountID,
			account.EligibilityTargetRef.Kind,
			EligibilityTargetKindAccountClass,
		)
	}

	provider, err := LoadPlatformRecord(ec.MissionStoreRoot, identity.EligibilityTargetRef.RegistryID)
	if err != nil {
		return ResolvedExecutionContextFrankGitHubOnboardingBundle{}, false, fmt.Errorf("execution context github identity %q provider target %q: %w", identity.IdentityID, identity.EligibilityTargetRef.RegistryID, err)
	}
	if provider.TargetClass != EligibilityTargetKindProvider {
		return ResolvedExecutionContextFrankGitHubOnboardingBundle{}, false, fmt.Errorf("execution context github provider %q target_class %q must be %q", provider.PlatformID, provider.TargetClass, EligibilityTargetKindProvider)
	}
	providerCheck, err := LoadEligibilityCheckRecord(ec.MissionStoreRoot, provider.LastCheckID)
	if err != nil {
		return ResolvedExecutionContextFrankGitHubOnboardingBundle{}, false, fmt.Errorf("execution context github provider %q last_check_id %q: %w", provider.PlatformID, provider.LastCheckID, err)
	}

	accountClass, err := LoadPlatformRecord(ec.MissionStoreRoot, account.EligibilityTargetRef.RegistryID)
	if err != nil {
		return ResolvedExecutionContextFrankGitHubOnboardingBundle{}, false, fmt.Errorf("execution context github account %q account-class target %q: %w", account.AccountID, account.EligibilityTargetRef.RegistryID, err)
	}
	if accountClass.TargetClass != EligibilityTargetKindAccountClass {
		return ResolvedExecutionContextFrankGitHubOnboardingBundle{}, false, fmt.Errorf("execution context github account-class %q target_class %q must be %q", accountClass.PlatformID, accountClass.TargetClass, EligibilityTargetKindAccountClass)
	}
	accountCheck, err := LoadEligibilityCheckRecord(ec.MissionStoreRoot, accountClass.LastCheckID)
	if err != nil {
		return ResolvedExecutionContextFrankGitHubOnboardingBundle{}, false, fmt.Errorf("execution context github account-class %q last_check_id %q: %w", accountClass.PlatformID, accountClass.LastCheckID, err)
	}

	return ResolvedExecutionContextFrankGitHubOnboardingBundle{
		Identity:      identity,
		Account:       account,
		Provider:      provider,
		ProviderCheck: providerCheck,
		AccountClass:  accountClass,
		AccountCheck:  accountCheck,
	}, true, nil
}

func ResolveExecutionContextFrankGitHubOnboardingPreflight(ec ExecutionContext) (ResolvedExecutionContextFrankGitHubOnboardingPreflight, error) {
	if ec.Step == nil {
		return ResolvedExecutionContextFrankGitHubOnboardingPreflight{}, fmt.Errorf("execution context step is required")
	}
	if !DeclaresFrankGitHubOnboarding(*ec.Step) {
		return ResolvedExecutionContextFrankGitHubOnboardingPreflight{}, nil
	}
	if strings.TrimSpace(ec.MissionStoreRoot) == "" {
		return ResolvedExecutionContextFrankGitHubOnboardingPreflight{}, fmt.Errorf("mission store root is required to resolve Frank object refs")
	}

	bundle, ok, err := ResolveExecutionContextFrankGitHubOnboardingBundle(ec)
	if err != nil {
		return ResolvedExecutionContextFrankGitHubOnboardingPreflight{}, err
	}
	if !ok {
		return ResolvedExecutionContextFrankGitHubOnboardingPreflight{}, nil
	}

	return ResolvedExecutionContextFrankGitHubOnboardingPreflight{
		Identity: &bundle.Identity,
		Account:  &bundle.Account,
	}, nil
}

func normalizeFrankGitHubIdentity(config *FrankGitHubIdentity) *FrankGitHubIdentity {
	if config == nil {
		return nil
	}
	normalized := *config
	normalized.GitHubUserID = strings.TrimSpace(normalized.GitHubUserID)
	normalized.Login = strings.TrimSpace(normalized.Login)
	normalized.NodeID = strings.TrimSpace(normalized.NodeID)
	normalized.PrimaryVerifiedEmail = strings.TrimSpace(normalized.PrimaryVerifiedEmail)
	return &normalized
}

func normalizeFrankGitHubAccount(config *FrankGitHubAccount) *FrankGitHubAccount {
	if config == nil {
		return nil
	}
	normalized := *config
	normalized.TokenEnvVarRef = strings.TrimSpace(normalized.TokenEnvVarRef)
	return &normalized
}

func validateFrankGitHubIdentity(config FrankGitHubIdentity) error {
	if githubUserID := strings.TrimSpace(config.GitHubUserID); githubUserID != "" && !isDigitsOnly(githubUserID) {
		return fmt.Errorf("mission store Frank identity github.github_user_id must be numeric when present")
	}
	if email := strings.TrimSpace(config.PrimaryVerifiedEmail); email != "" {
		parsed, err := mail.ParseAddress(email)
		if err != nil || parsed == nil || parsed.Name != "" || parsed.Address != email {
			return fmt.Errorf("mission store Frank identity github.primary_verified_email must be a bare email address when present")
		}
	}
	return nil
}

func validateFrankGitHubAccount(config FrankGitHubAccount) error {
	if config.ConfirmedAuthenticated && strings.TrimSpace(config.TokenEnvVarRef) == "" {
		return fmt.Errorf("mission store Frank account github.confirmed_authenticated requires github.token_env_var_ref")
	}
	return nil
}

func normalizeResolvedExecutionContextFrankGitHubOnboardingBundle(bundle ResolvedExecutionContextFrankGitHubOnboardingBundle) ResolvedExecutionContextFrankGitHubOnboardingBundle {
	bundle.Identity.ZohoMailbox = normalizeFrankZohoMailboxIdentity(bundle.Identity.ZohoMailbox)
	bundle.Identity.TelegramOwnerControl = normalizeFrankTelegramOwnerControlIdentity(bundle.Identity.TelegramOwnerControl)
	bundle.Identity.SlackOwnerControl = normalizeFrankSlackOwnerControlIdentity(bundle.Identity.SlackOwnerControl)
	bundle.Identity.DiscordOwnerControl = normalizeFrankDiscordOwnerControlIdentity(bundle.Identity.DiscordOwnerControl)
	bundle.Identity.WhatsAppOwnerControl = normalizeFrankWhatsAppOwnerControlIdentity(bundle.Identity.WhatsAppOwnerControl)
	bundle.Identity.GitHub = normalizeFrankGitHubIdentity(bundle.Identity.GitHub)
	bundle.Identity.IdentityMode = NormalizeIdentityMode(bundle.Identity.IdentityMode)
	bundle.Identity.CreatedAt = bundle.Identity.CreatedAt.UTC()
	bundle.Identity.UpdatedAt = bundle.Identity.UpdatedAt.UTC()

	bundle.Account.ZohoMailbox = normalizeFrankZohoMailboxAccount(bundle.Account.ZohoMailbox)
	bundle.Account.TelegramOwnerControl = normalizeFrankTelegramOwnerControlAccount(bundle.Account.TelegramOwnerControl)
	bundle.Account.SlackOwnerControl = normalizeFrankSlackOwnerControlAccount(bundle.Account.SlackOwnerControl)
	bundle.Account.DiscordOwnerControl = normalizeFrankDiscordOwnerControlAccount(bundle.Account.DiscordOwnerControl)
	bundle.Account.WhatsAppOwnerControl = normalizeFrankWhatsAppOwnerControlAccount(bundle.Account.WhatsAppOwnerControl)
	bundle.Account.GitHub = normalizeFrankGitHubAccount(bundle.Account.GitHub)
	bundle.Account.CreatedAt = bundle.Account.CreatedAt.UTC()
	bundle.Account.UpdatedAt = bundle.Account.UpdatedAt.UTC()

	return bundle
}

func validateResolvedExecutionContextFrankGitHubOnboardingBundle(bundle ResolvedExecutionContextFrankGitHubOnboardingBundle) error {
	if err := ValidateFrankIdentityRecord(bundle.Identity); err != nil {
		return err
	}
	if err := ValidateFrankAccountRecord(bundle.Account); err != nil {
		return err
	}
	if bundle.Identity.GitHub == nil {
		return fmt.Errorf("mission store Frank github onboarding requires github identity record")
	}
	if bundle.Account.GitHub == nil {
		return fmt.Errorf("mission store Frank github onboarding requires github account record")
	}
	if NormalizeIdentityMode(bundle.Identity.IdentityMode) != IdentityModeAgentAlias {
		return fmt.Errorf(
			"mission store Frank github onboarding identity %q identity_mode %q must be %q",
			bundle.Identity.IdentityID,
			bundle.Identity.IdentityMode,
			IdentityModeAgentAlias,
		)
	}
	if strings.TrimSpace(bundle.Account.IdentityID) != strings.TrimSpace(bundle.Identity.IdentityID) {
		return fmt.Errorf(
			"mission store Frank github onboarding account %q must link identity_id %q, got %q",
			bundle.Account.AccountID,
			bundle.Identity.IdentityID,
			bundle.Account.IdentityID,
		)
	}
	if strings.TrimSpace(bundle.Account.ProviderOrPlatformID) != strings.TrimSpace(bundle.Identity.ProviderOrPlatformID) {
		return fmt.Errorf(
			"mission store Frank github onboarding account %q provider_or_platform_id %q does not match identity %q provider_or_platform_id %q",
			bundle.Account.AccountID,
			bundle.Account.ProviderOrPlatformID,
			bundle.Identity.IdentityID,
			bundle.Identity.ProviderOrPlatformID,
		)
	}
	if bundle.Provider.TargetClass != EligibilityTargetKindProvider {
		return fmt.Errorf("mission store Frank github onboarding provider %q target_class %q must be %q", bundle.Provider.PlatformID, bundle.Provider.TargetClass, EligibilityTargetKindProvider)
	}
	if bundle.AccountClass.TargetClass != EligibilityTargetKindAccountClass {
		return fmt.Errorf("mission store Frank github onboarding account-class %q target_class %q must be %q", bundle.AccountClass.PlatformID, bundle.AccountClass.TargetClass, EligibilityTargetKindAccountClass)
	}
	if bundle.Account.GitHub.ConfirmedAuthenticated {
		if strings.TrimSpace(bundle.Identity.GitHub.GitHubUserID) == "" || strings.TrimSpace(bundle.Identity.GitHub.Login) == "" {
			return fmt.Errorf("mission store Frank github onboarding confirmed identity requires github.github_user_id and github.login")
		}
		if strings.TrimSpace(bundle.Account.GitHub.TokenEnvVarRef) == "" {
			return fmt.Errorf("mission store Frank github onboarding confirmed account requires github.token_env_var_ref")
		}
	}
	return nil
}
