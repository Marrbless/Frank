package missioncontrol

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/local/picobot/internal/channels"
	"github.com/local/picobot/internal/config"
)

var readWhatsAppOwnerControlIdentity = channels.ReadWhatsAppAuthenticatedIdentity

// ProduceFrankWhatsAppOwnerControlOnboarding is the single missioncontrol-owned
// execution producer for the WhatsApp owner-control onboarding lane. It reads
// the configured WhatsApp authenticated-device store, confirms account
// identity through provider read-back only, and persists only non-secret
// WhatsApp identifiers into the committed Frank identity/account records.
func ProduceFrankWhatsAppOwnerControlOnboarding(root string, bundle ResolvedExecutionContextFrankWhatsAppOwnerControlOnboardingBundle, now time.Time) error {
	if err := ValidateStoreRoot(root); err != nil {
		return err
	}
	if now.IsZero() {
		now = time.Now().UTC()
	} else {
		now = now.UTC()
	}

	bundle = normalizeResolvedExecutionContextFrankWhatsAppOwnerControlOnboardingBundle(bundle)
	if err := validateResolvedExecutionContextFrankWhatsAppOwnerControlOnboardingBundle(bundle); err != nil {
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
		return fmt.Errorf("whatsapp owner-control onboarding requires readable config: %w", err)
	}
	_, resolvedDBPath, err := resolveWhatsAppOwnerControlConfig(cfg)
	if err != nil {
		return err
	}

	readBack, err := readWhatsAppOwnerControlIdentity(context.Background(), resolvedDBPath)
	if err != nil {
		return fmt.Errorf("whatsapp owner-control onboarding provider read-back failed: %w", err)
	}
	if strings.TrimSpace(readBack.PhoneJID) == "" && strings.TrimSpace(readBack.LIDJID) == "" {
		return fmt.Errorf("whatsapp owner-control onboarding provider read-back did not return phone_jid or lid_jid")
	}
	if strings.TrimSpace(readBack.AuthenticatedDeviceJID) == "" {
		return fmt.Errorf("whatsapp owner-control onboarding provider read-back did not return authenticated device jid")
	}

	committedPhoneJID := ""
	committedLIDJID := ""
	if bundle.Identity.WhatsAppOwnerControl != nil {
		committedPhoneJID = strings.TrimSpace(bundle.Identity.WhatsAppOwnerControl.PhoneJID)
		committedLIDJID = strings.TrimSpace(bundle.Identity.WhatsAppOwnerControl.LIDJID)
	}
	if committedPhoneJID != "" && committedPhoneJID != strings.TrimSpace(readBack.PhoneJID) {
		return fmt.Errorf(
			"whatsapp owner-control identity %q conflicts with provider phone_jid %q",
			bundle.Identity.IdentityID,
			readBack.PhoneJID,
		)
	}
	if committedLIDJID != "" && committedLIDJID != strings.TrimSpace(readBack.LIDJID) {
		return fmt.Errorf(
			"whatsapp owner-control identity %q conflicts with provider lid_jid %q",
			bundle.Identity.IdentityID,
			readBack.LIDJID,
		)
	}

	committedAuthenticatedDeviceJID := ""
	committedAuthStoreRef := ""
	committedConfirmedAuthenticated := false
	if bundle.Account.WhatsAppOwnerControl != nil {
		committedAuthenticatedDeviceJID = strings.TrimSpace(bundle.Account.WhatsAppOwnerControl.AuthenticatedDeviceJID)
		committedAuthStoreRef = strings.TrimSpace(bundle.Account.WhatsAppOwnerControl.AuthStoreRef)
		committedConfirmedAuthenticated = bundle.Account.WhatsAppOwnerControl.ConfirmedAuthenticated
	}
	if committedAuthenticatedDeviceJID != "" && committedAuthenticatedDeviceJID != strings.TrimSpace(readBack.AuthenticatedDeviceJID) {
		return fmt.Errorf(
			"whatsapp owner-control account %q conflicts with provider authenticated_device_jid %q",
			bundle.Account.AccountID,
			readBack.AuthenticatedDeviceJID,
		)
	}
	if committedAuthStoreRef != "" && committedAuthStoreRef != resolvedDBPath {
		return fmt.Errorf(
			"whatsapp owner-control account %q conflicts with configured auth_store_ref %q",
			bundle.Account.AccountID,
			resolvedDBPath,
		)
	}

	updatedIdentity := bundle.Identity
	if updatedIdentity.WhatsAppOwnerControl == nil {
		updatedIdentity.WhatsAppOwnerControl = &FrankWhatsAppOwnerControlIdentity{}
	}
	if updatedIdentity.WhatsAppOwnerControl.PhoneJID == "" {
		updatedIdentity.WhatsAppOwnerControl.PhoneJID = strings.TrimSpace(readBack.PhoneJID)
	}
	if updatedIdentity.WhatsAppOwnerControl.LIDJID == "" {
		updatedIdentity.WhatsAppOwnerControl.LIDJID = strings.TrimSpace(readBack.LIDJID)
	}
	updatedIdentity.UpdatedAt = updatedAtOnOrAfterCreatedAt(updatedIdentity.CreatedAt, now)

	updatedAccount := bundle.Account
	if updatedAccount.WhatsAppOwnerControl == nil {
		updatedAccount.WhatsAppOwnerControl = &FrankWhatsAppOwnerControlAccount{}
	}
	if updatedAccount.WhatsAppOwnerControl.AuthenticatedDeviceJID == "" {
		updatedAccount.WhatsAppOwnerControl.AuthenticatedDeviceJID = strings.TrimSpace(readBack.AuthenticatedDeviceJID)
	}
	if updatedAccount.WhatsAppOwnerControl.AuthStoreRef == "" {
		updatedAccount.WhatsAppOwnerControl.AuthStoreRef = resolvedDBPath
	}
	updatedAccount.WhatsAppOwnerControl.ConfirmedAuthenticated = true
	updatedAccount.UpdatedAt = updatedAtOnOrAfterCreatedAt(updatedAccount.CreatedAt, now)

	if committedConfirmedAuthenticated &&
		strings.TrimSpace(updatedIdentity.WhatsAppOwnerControl.PhoneJID) == committedPhoneJID &&
		strings.TrimSpace(updatedIdentity.WhatsAppOwnerControl.LIDJID) == committedLIDJID &&
		strings.TrimSpace(updatedAccount.WhatsAppOwnerControl.AuthenticatedDeviceJID) == committedAuthenticatedDeviceJID &&
		strings.TrimSpace(updatedAccount.WhatsAppOwnerControl.AuthStoreRef) == committedAuthStoreRef {
		return nil
	}

	if err := StoreFrankIdentityRecord(root, updatedIdentity); err != nil {
		return err
	}
	return StoreFrankAccountRecord(root, updatedAccount)
}

func resolveWhatsAppOwnerControlConfig(cfg config.Config) (config.WhatsAppConfig, string, error) {
	channelCfg := cfg.Channels.WhatsApp
	if !channelCfg.Enabled {
		return config.WhatsAppConfig{}, "", fmt.Errorf("whatsapp owner-control onboarding requires configured whatsapp channel enabled")
	}
	if strings.TrimSpace(channelCfg.DBPath) == "" {
		return config.WhatsAppConfig{}, "", fmt.Errorf("whatsapp owner-control onboarding requires configured whatsapp db path")
	}

	allowFrom := make([]string, 0, len(channelCfg.AllowFrom))
	for _, raw := range channelCfg.AllowFrom {
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" {
			continue
		}
		if !isDigitsOnly(trimmed) {
			return config.WhatsAppConfig{}, "", fmt.Errorf("whatsapp owner-control onboarding allowFrom entry %q must be numeric", trimmed)
		}
		allowFrom = append(allowFrom, trimmed)
	}
	if len(allowFrom) != 1 {
		return config.WhatsAppConfig{}, "", fmt.Errorf("whatsapp owner-control onboarding requires exactly one configured whatsapp allowFrom user id, got %d", len(allowFrom))
	}
	channelCfg.AllowFrom = allowFrom

	resolvedDBPath, err := resolveWhatsAppOwnerControlDBPath(channelCfg.DBPath)
	if err != nil {
		return config.WhatsAppConfig{}, "", err
	}
	channelCfg.DBPath = resolvedDBPath
	return channelCfg, resolvedDBPath, nil
}

func resolveWhatsAppOwnerControlDBPath(dbPath string) (string, error) {
	trimmed := strings.TrimSpace(dbPath)
	if trimmed == "" {
		return "", fmt.Errorf("whatsapp owner-control onboarding requires configured whatsapp db path")
	}
	if trimmed == "~" || strings.HasPrefix(trimmed, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("whatsapp owner-control onboarding requires resolvable home directory: %w", err)
		}
		if trimmed == "~" {
			trimmed = home
		} else {
			trimmed = filepath.Join(home, trimmed[2:])
		}
	}
	resolved, err := filepath.Abs(trimmed)
	if err != nil {
		return "", fmt.Errorf("whatsapp owner-control onboarding requires resolvable whatsapp db path: %w", err)
	}
	return resolved, nil
}
