package missioncontrol

import (
	"fmt"
	"strings"
)

type ResolvedExecutionContextFrankWhatsAppOwnerControlOnboardingBundle struct {
	Identity      FrankIdentityRecord    `json:"identity"`
	Account       FrankAccountRecord     `json:"account"`
	Provider      PlatformRecord         `json:"provider"`
	ProviderCheck EligibilityCheckRecord `json:"provider_check"`
	AccountClass  PlatformRecord         `json:"account_class"`
	AccountCheck  EligibilityCheckRecord `json:"account_check"`
}

type ResolvedExecutionContextFrankWhatsAppOwnerControlOnboardingPreflight struct {
	Identity *FrankIdentityRecord `json:"identity,omitempty"`
	Account  *FrankAccountRecord  `json:"account,omitempty"`
}

func DeclaresFrankWhatsAppOwnerControlOnboarding(step Step) bool {
	if NormalizeIdentityMode(step.IdentityMode) != IdentityModeOwnerOnlyControl {
		return false
	}
	return len(step.FrankObjectRefs) > 0
}

func ResolveExecutionContextFrankWhatsAppOwnerControlOnboardingBundle(ec ExecutionContext) (ResolvedExecutionContextFrankWhatsAppOwnerControlOnboardingBundle, bool, error) {
	resolved, err := ResolveExecutionContextFrankRegistryObjectRefs(ec)
	if err != nil {
		return ResolvedExecutionContextFrankWhatsAppOwnerControlOnboardingBundle{}, false, err
	}

	identities := make([]FrankIdentityRecord, 0, len(resolved.Identities))
	for _, identity := range resolved.Identities {
		if identity.WhatsAppOwnerControl != nil {
			identities = append(identities, identity)
		}
	}
	accounts := make([]FrankAccountRecord, 0, len(resolved.Accounts))
	for _, account := range resolved.Accounts {
		if account.WhatsAppOwnerControl != nil {
			accounts = append(accounts, account)
		}
	}

	if len(identities) == 0 && len(accounts) == 0 {
		return ResolvedExecutionContextFrankWhatsAppOwnerControlOnboardingBundle{}, false, nil
	}
	if len(identities) != 1 {
		return ResolvedExecutionContextFrankWhatsAppOwnerControlOnboardingBundle{}, false, fmt.Errorf("execution context Frank object refs must resolve exactly one whatsapp owner-control identity, got %d", len(identities))
	}
	if len(accounts) != 1 {
		return ResolvedExecutionContextFrankWhatsAppOwnerControlOnboardingBundle{}, false, fmt.Errorf("execution context Frank object refs must resolve exactly one whatsapp owner-control account, got %d", len(accounts))
	}

	identity := identities[0]
	account := accounts[0]
	if strings.TrimSpace(account.IdentityID) != strings.TrimSpace(identity.IdentityID) {
		return ResolvedExecutionContextFrankWhatsAppOwnerControlOnboardingBundle{}, false, fmt.Errorf(
			"execution context whatsapp owner-control account %q must link identity_id %q, got %q",
			account.AccountID,
			identity.IdentityID,
			account.IdentityID,
		)
	}
	if strings.TrimSpace(account.ProviderOrPlatformID) != strings.TrimSpace(identity.ProviderOrPlatformID) {
		return ResolvedExecutionContextFrankWhatsAppOwnerControlOnboardingBundle{}, false, fmt.Errorf(
			"execution context whatsapp owner-control account %q provider_or_platform_id %q does not match identity %q provider_or_platform_id %q",
			account.AccountID,
			account.ProviderOrPlatformID,
			identity.IdentityID,
			identity.ProviderOrPlatformID,
		)
	}
	if identity.EligibilityTargetRef.Kind != EligibilityTargetKindProvider {
		return ResolvedExecutionContextFrankWhatsAppOwnerControlOnboardingBundle{}, false, fmt.Errorf(
			"execution context whatsapp owner-control identity %q eligibility_target_ref.kind %q must be %q",
			identity.IdentityID,
			identity.EligibilityTargetRef.Kind,
			EligibilityTargetKindProvider,
		)
	}
	if account.EligibilityTargetRef.Kind != EligibilityTargetKindAccountClass {
		return ResolvedExecutionContextFrankWhatsAppOwnerControlOnboardingBundle{}, false, fmt.Errorf(
			"execution context whatsapp owner-control account %q eligibility_target_ref.kind %q must be %q",
			account.AccountID,
			account.EligibilityTargetRef.Kind,
			EligibilityTargetKindAccountClass,
		)
	}

	provider, err := LoadPlatformRecord(ec.MissionStoreRoot, identity.EligibilityTargetRef.RegistryID)
	if err != nil {
		return ResolvedExecutionContextFrankWhatsAppOwnerControlOnboardingBundle{}, false, fmt.Errorf("execution context whatsapp owner-control identity %q provider target %q: %w", identity.IdentityID, identity.EligibilityTargetRef.RegistryID, err)
	}
	if provider.TargetClass != EligibilityTargetKindProvider {
		return ResolvedExecutionContextFrankWhatsAppOwnerControlOnboardingBundle{}, false, fmt.Errorf("execution context whatsapp owner-control provider %q target_class %q must be %q", provider.PlatformID, provider.TargetClass, EligibilityTargetKindProvider)
	}
	providerCheck, err := LoadEligibilityCheckRecord(ec.MissionStoreRoot, provider.LastCheckID)
	if err != nil {
		return ResolvedExecutionContextFrankWhatsAppOwnerControlOnboardingBundle{}, false, fmt.Errorf("execution context whatsapp owner-control provider %q last_check_id %q: %w", provider.PlatformID, provider.LastCheckID, err)
	}

	accountClass, err := LoadPlatformRecord(ec.MissionStoreRoot, account.EligibilityTargetRef.RegistryID)
	if err != nil {
		return ResolvedExecutionContextFrankWhatsAppOwnerControlOnboardingBundle{}, false, fmt.Errorf("execution context whatsapp owner-control account %q account-class target %q: %w", account.AccountID, account.EligibilityTargetRef.RegistryID, err)
	}
	if accountClass.TargetClass != EligibilityTargetKindAccountClass {
		return ResolvedExecutionContextFrankWhatsAppOwnerControlOnboardingBundle{}, false, fmt.Errorf("execution context whatsapp owner-control account-class %q target_class %q must be %q", accountClass.PlatformID, accountClass.TargetClass, EligibilityTargetKindAccountClass)
	}
	accountCheck, err := LoadEligibilityCheckRecord(ec.MissionStoreRoot, accountClass.LastCheckID)
	if err != nil {
		return ResolvedExecutionContextFrankWhatsAppOwnerControlOnboardingBundle{}, false, fmt.Errorf("execution context whatsapp owner-control account-class %q last_check_id %q: %w", accountClass.PlatformID, accountClass.LastCheckID, err)
	}

	return ResolvedExecutionContextFrankWhatsAppOwnerControlOnboardingBundle{
		Identity:      identity,
		Account:       account,
		Provider:      provider,
		ProviderCheck: providerCheck,
		AccountClass:  accountClass,
		AccountCheck:  accountCheck,
	}, true, nil
}

func ResolveExecutionContextFrankWhatsAppOwnerControlOnboardingPreflight(ec ExecutionContext) (ResolvedExecutionContextFrankWhatsAppOwnerControlOnboardingPreflight, error) {
	if ec.Step == nil {
		return ResolvedExecutionContextFrankWhatsAppOwnerControlOnboardingPreflight{}, fmt.Errorf("execution context step is required")
	}
	if !DeclaresFrankWhatsAppOwnerControlOnboarding(*ec.Step) {
		return ResolvedExecutionContextFrankWhatsAppOwnerControlOnboardingPreflight{}, nil
	}
	if strings.TrimSpace(ec.MissionStoreRoot) == "" {
		return ResolvedExecutionContextFrankWhatsAppOwnerControlOnboardingPreflight{}, fmt.Errorf("mission store root is required to resolve Frank object refs")
	}

	bundle, ok, err := ResolveExecutionContextFrankWhatsAppOwnerControlOnboardingBundle(ec)
	if err != nil {
		return ResolvedExecutionContextFrankWhatsAppOwnerControlOnboardingPreflight{}, err
	}
	if !ok {
		return ResolvedExecutionContextFrankWhatsAppOwnerControlOnboardingPreflight{}, nil
	}

	return ResolvedExecutionContextFrankWhatsAppOwnerControlOnboardingPreflight{
		Identity: &bundle.Identity,
		Account:  &bundle.Account,
	}, nil
}

func normalizeFrankWhatsAppOwnerControlIdentity(config *FrankWhatsAppOwnerControlIdentity) *FrankWhatsAppOwnerControlIdentity {
	if config == nil {
		return nil
	}
	normalized := *config
	normalized.PhoneJID = strings.TrimSpace(normalized.PhoneJID)
	normalized.LIDJID = strings.TrimSpace(normalized.LIDJID)
	normalized.PushName = strings.TrimSpace(normalized.PushName)
	return &normalized
}

func normalizeFrankWhatsAppOwnerControlAccount(config *FrankWhatsAppOwnerControlAccount) *FrankWhatsAppOwnerControlAccount {
	if config == nil {
		return nil
	}
	normalized := *config
	normalized.AuthenticatedDeviceJID = strings.TrimSpace(normalized.AuthenticatedDeviceJID)
	normalized.AuthStoreRef = strings.TrimSpace(normalized.AuthStoreRef)
	return &normalized
}

func validateFrankWhatsAppOwnerControlIdentity(config FrankWhatsAppOwnerControlIdentity) error {
	if phoneJID := strings.TrimSpace(config.PhoneJID); phoneJID != "" && !strings.Contains(phoneJID, "@") {
		return fmt.Errorf("mission store Frank identity whatsapp_owner_control.phone_jid must be a full jid when present")
	}
	if lidJID := strings.TrimSpace(config.LIDJID); lidJID != "" && !strings.Contains(lidJID, "@") {
		return fmt.Errorf("mission store Frank identity whatsapp_owner_control.lid_jid must be a full jid when present")
	}
	return nil
}

func validateFrankWhatsAppOwnerControlAccount(config FrankWhatsAppOwnerControlAccount) error {
	if authenticatedDeviceJID := strings.TrimSpace(config.AuthenticatedDeviceJID); authenticatedDeviceJID != "" && !strings.Contains(authenticatedDeviceJID, "@") {
		return fmt.Errorf("mission store Frank account whatsapp_owner_control.authenticated_device_jid must be a full jid when present")
	}
	if config.ConfirmedAuthenticated && strings.TrimSpace(config.AuthenticatedDeviceJID) == "" {
		return fmt.Errorf("mission store Frank account whatsapp_owner_control.confirmed_authenticated requires whatsapp_owner_control.authenticated_device_jid")
	}
	return nil
}

func normalizeResolvedExecutionContextFrankWhatsAppOwnerControlOnboardingBundle(bundle ResolvedExecutionContextFrankWhatsAppOwnerControlOnboardingBundle) ResolvedExecutionContextFrankWhatsAppOwnerControlOnboardingBundle {
	bundle.Identity.ZohoMailbox = normalizeFrankZohoMailboxIdentity(bundle.Identity.ZohoMailbox)
	bundle.Identity.TelegramOwnerControl = normalizeFrankTelegramOwnerControlIdentity(bundle.Identity.TelegramOwnerControl)
	bundle.Identity.SlackOwnerControl = normalizeFrankSlackOwnerControlIdentity(bundle.Identity.SlackOwnerControl)
	bundle.Identity.DiscordOwnerControl = normalizeFrankDiscordOwnerControlIdentity(bundle.Identity.DiscordOwnerControl)
	bundle.Identity.WhatsAppOwnerControl = normalizeFrankWhatsAppOwnerControlIdentity(bundle.Identity.WhatsAppOwnerControl)
	bundle.Identity.IdentityMode = NormalizeIdentityMode(bundle.Identity.IdentityMode)
	bundle.Identity.CreatedAt = bundle.Identity.CreatedAt.UTC()
	bundle.Identity.UpdatedAt = bundle.Identity.UpdatedAt.UTC()

	bundle.Account.ZohoMailbox = normalizeFrankZohoMailboxAccount(bundle.Account.ZohoMailbox)
	bundle.Account.TelegramOwnerControl = normalizeFrankTelegramOwnerControlAccount(bundle.Account.TelegramOwnerControl)
	bundle.Account.SlackOwnerControl = normalizeFrankSlackOwnerControlAccount(bundle.Account.SlackOwnerControl)
	bundle.Account.DiscordOwnerControl = normalizeFrankDiscordOwnerControlAccount(bundle.Account.DiscordOwnerControl)
	bundle.Account.WhatsAppOwnerControl = normalizeFrankWhatsAppOwnerControlAccount(bundle.Account.WhatsAppOwnerControl)
	bundle.Account.CreatedAt = bundle.Account.CreatedAt.UTC()
	bundle.Account.UpdatedAt = bundle.Account.UpdatedAt.UTC()

	return bundle
}

func validateResolvedExecutionContextFrankWhatsAppOwnerControlOnboardingBundle(bundle ResolvedExecutionContextFrankWhatsAppOwnerControlOnboardingBundle) error {
	if err := ValidateFrankIdentityRecord(bundle.Identity); err != nil {
		return err
	}
	if err := ValidateFrankAccountRecord(bundle.Account); err != nil {
		return err
	}
	if bundle.Identity.WhatsAppOwnerControl == nil {
		return fmt.Errorf("mission store Frank whatsapp owner-control onboarding requires whatsapp_owner_control identity record")
	}
	if bundle.Account.WhatsAppOwnerControl == nil {
		return fmt.Errorf("mission store Frank whatsapp owner-control onboarding requires whatsapp_owner_control account record")
	}
	if NormalizeIdentityMode(bundle.Identity.IdentityMode) != IdentityModeOwnerOnlyControl {
		return fmt.Errorf(
			"mission store Frank whatsapp owner-control onboarding identity %q identity_mode %q must be %q",
			bundle.Identity.IdentityID,
			bundle.Identity.IdentityMode,
			IdentityModeOwnerOnlyControl,
		)
	}
	if strings.TrimSpace(bundle.Account.IdentityID) != strings.TrimSpace(bundle.Identity.IdentityID) {
		return fmt.Errorf(
			"mission store Frank whatsapp owner-control onboarding account %q must link identity_id %q, got %q",
			bundle.Account.AccountID,
			bundle.Identity.IdentityID,
			bundle.Account.IdentityID,
		)
	}
	if strings.TrimSpace(bundle.Account.ProviderOrPlatformID) != strings.TrimSpace(bundle.Identity.ProviderOrPlatformID) {
		return fmt.Errorf(
			"mission store Frank whatsapp owner-control onboarding account %q provider_or_platform_id %q does not match identity %q provider_or_platform_id %q",
			bundle.Account.AccountID,
			bundle.Account.ProviderOrPlatformID,
			bundle.Identity.IdentityID,
			bundle.Identity.ProviderOrPlatformID,
		)
	}
	if bundle.Provider.TargetClass != EligibilityTargetKindProvider {
		return fmt.Errorf("mission store Frank whatsapp owner-control onboarding provider %q target_class %q must be %q", bundle.Provider.PlatformID, bundle.Provider.TargetClass, EligibilityTargetKindProvider)
	}
	if bundle.AccountClass.TargetClass != EligibilityTargetKindAccountClass {
		return fmt.Errorf("mission store Frank whatsapp owner-control onboarding account-class %q target_class %q must be %q", bundle.AccountClass.PlatformID, bundle.AccountClass.TargetClass, EligibilityTargetKindAccountClass)
	}
	return nil
}
