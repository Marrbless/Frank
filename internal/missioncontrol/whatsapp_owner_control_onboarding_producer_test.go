package missioncontrol

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/local/picobot/internal/channels"
	"github.com/local/picobot/internal/config"
)

func TestResolveExecutionContextFrankWhatsAppOwnerControlOnboardingBundleResolvesExactLinkedBundle(t *testing.T) {
	t.Parallel()

	fixtures := writeWhatsAppOwnerControlOnboardingFixtures(t)
	ec := testExecutionContextWithFrankObjectRefs(t, []FrankRegistryObjectRef{
		{Kind: FrankRegistryObjectKindIdentity, ObjectID: fixtures.identity.IdentityID},
		{Kind: FrankRegistryObjectKindAccount, ObjectID: fixtures.account.AccountID},
	})
	ec.MissionStoreRoot = fixtures.root

	got, ok, err := ResolveExecutionContextFrankWhatsAppOwnerControlOnboardingBundle(ec)
	if err != nil {
		t.Fatalf("ResolveExecutionContextFrankWhatsAppOwnerControlOnboardingBundle() error = %v", err)
	}
	if !ok {
		t.Fatal("ResolveExecutionContextFrankWhatsAppOwnerControlOnboardingBundle() ok = false, want true")
	}
	if got.Identity.IdentityID != fixtures.identity.IdentityID {
		t.Fatalf("Identity.IdentityID = %q, want %q", got.Identity.IdentityID, fixtures.identity.IdentityID)
	}
	if got.Account.AccountID != fixtures.account.AccountID {
		t.Fatalf("Account.AccountID = %q, want %q", got.Account.AccountID, fixtures.account.AccountID)
	}
	if got.Provider.PlatformID != fixtures.providerTarget.RegistryID {
		t.Fatalf("Provider.PlatformID = %q, want %q", got.Provider.PlatformID, fixtures.providerTarget.RegistryID)
	}
	if got.AccountClass.PlatformID != fixtures.accountClassTarget.RegistryID {
		t.Fatalf("AccountClass.PlatformID = %q, want %q", got.AccountClass.PlatformID, fixtures.accountClassTarget.RegistryID)
	}
}

func TestResolveExecutionContextFrankWhatsAppOwnerControlOnboardingBundleFailsClosedOnMissingAccount(t *testing.T) {
	t.Parallel()

	fixtures := writeWhatsAppOwnerControlOnboardingFixtures(t)
	ec := testExecutionContextWithFrankObjectRefs(t, []FrankRegistryObjectRef{
		{Kind: FrankRegistryObjectKindIdentity, ObjectID: fixtures.identity.IdentityID},
	})
	ec.MissionStoreRoot = fixtures.root

	_, _, err := ResolveExecutionContextFrankWhatsAppOwnerControlOnboardingBundle(ec)
	if err == nil {
		t.Fatal("ResolveExecutionContextFrankWhatsAppOwnerControlOnboardingBundle() error = nil, want missing account rejection")
	}
	if !strings.Contains(err.Error(), "must resolve exactly one whatsapp owner-control account, got 0") {
		t.Fatalf("ResolveExecutionContextFrankWhatsAppOwnerControlOnboardingBundle() error = %q, want missing account rejection", err.Error())
	}
}

func TestProduceFrankWhatsAppOwnerControlOnboardingPersistsConfirmedIdentifiersAndReplaysDeterministically(t *testing.T) {
	fixtures := writeWhatsAppOwnerControlOnboardingFixtures(t)
	dbPath := filepath.Join(t.TempDir(), "whatsapp.db")
	configureWhatsAppOwnerControlTestConfig(t, "15551234567", dbPath, filepath.Join(t.TempDir(), "workspace"))
	resolvedDBPath, err := filepath.Abs(dbPath)
	if err != nil {
		t.Fatalf("filepath.Abs() error = %v", err)
	}

	originalRead := readWhatsAppOwnerControlIdentity
	defer func() { readWhatsAppOwnerControlIdentity = originalRead }()
	readWhatsAppOwnerControlIdentity = func(ctx context.Context, gotDBPath string) (channels.WhatsAppAuthenticatedIdentity, error) {
		if gotDBPath != resolvedDBPath {
			t.Fatalf("readWhatsAppOwnerControlIdentity() dbPath = %q, want %q", gotDBPath, resolvedDBPath)
		}
		return channels.WhatsAppAuthenticatedIdentity{
			PhoneJID:               "15551234567@s.whatsapp.net",
			LIDJID:                 "169032883908635@lid",
			AuthenticatedDeviceJID: "15551234567@s.whatsapp.net",
		}, nil
	}

	ec := testExecutionContextWithFrankObjectRefs(t, []FrankRegistryObjectRef{
		{Kind: FrankRegistryObjectKindIdentity, ObjectID: fixtures.identity.IdentityID},
		{Kind: FrankRegistryObjectKindAccount, ObjectID: fixtures.account.AccountID},
	})
	ec.MissionStoreRoot = fixtures.root
	bundle, ok, err := ResolveExecutionContextFrankWhatsAppOwnerControlOnboardingBundle(ec)
	if err != nil {
		t.Fatalf("ResolveExecutionContextFrankWhatsAppOwnerControlOnboardingBundle() error = %v", err)
	}
	if !ok {
		t.Fatal("ResolveExecutionContextFrankWhatsAppOwnerControlOnboardingBundle() ok = false, want true")
	}

	firstNow := time.Date(2026, 4, 18, 19, 0, 0, 0, time.UTC)
	if err := ProduceFrankWhatsAppOwnerControlOnboarding(fixtures.root, bundle, firstNow); err != nil {
		t.Fatalf("ProduceFrankWhatsAppOwnerControlOnboarding(first) error = %v", err)
	}

	identity, err := LoadFrankIdentityRecord(fixtures.root, fixtures.identity.IdentityID)
	if err != nil {
		t.Fatalf("LoadFrankIdentityRecord() error = %v", err)
	}
	account, err := LoadFrankAccountRecord(fixtures.root, fixtures.account.AccountID)
	if err != nil {
		t.Fatalf("LoadFrankAccountRecord() error = %v", err)
	}
	if identity.WhatsAppOwnerControl == nil || identity.WhatsAppOwnerControl.PhoneJID != "15551234567@s.whatsapp.net" || identity.WhatsAppOwnerControl.LIDJID != "169032883908635@lid" {
		t.Fatalf("Identity.WhatsAppOwnerControl = %#v, want phone_jid and lid_jid persisted", identity.WhatsAppOwnerControl)
	}
	if account.WhatsAppOwnerControl == nil || account.WhatsAppOwnerControl.AuthenticatedDeviceJID != "15551234567@s.whatsapp.net" || account.WhatsAppOwnerControl.AuthStoreRef != resolvedDBPath || !account.WhatsAppOwnerControl.ConfirmedAuthenticated {
		t.Fatalf("Account.WhatsAppOwnerControl = %#v, want authenticated device jid, auth_store_ref, and confirmation persisted", account.WhatsAppOwnerControl)
	}
	firstIdentityUpdatedAt := identity.UpdatedAt
	firstAccountUpdatedAt := account.UpdatedAt

	bundle, ok, err = ResolveExecutionContextFrankWhatsAppOwnerControlOnboardingBundle(ec)
	if err != nil {
		t.Fatalf("ResolveExecutionContextFrankWhatsAppOwnerControlOnboardingBundle(replay) error = %v", err)
	}
	if !ok {
		t.Fatal("ResolveExecutionContextFrankWhatsAppOwnerControlOnboardingBundle(replay) ok = false, want true")
	}

	if err := ProduceFrankWhatsAppOwnerControlOnboarding(fixtures.root, bundle, firstNow.Add(time.Minute)); err != nil {
		t.Fatalf("ProduceFrankWhatsAppOwnerControlOnboarding(replay) error = %v, want deterministic no-op success", err)
	}

	identity, err = LoadFrankIdentityRecord(fixtures.root, fixtures.identity.IdentityID)
	if err != nil {
		t.Fatalf("LoadFrankIdentityRecord(replay) error = %v", err)
	}
	account, err = LoadFrankAccountRecord(fixtures.root, fixtures.account.AccountID)
	if err != nil {
		t.Fatalf("LoadFrankAccountRecord(replay) error = %v", err)
	}
	if !identity.UpdatedAt.Equal(firstIdentityUpdatedAt) {
		t.Fatalf("Identity.UpdatedAt after replay = %v, want %v", identity.UpdatedAt, firstIdentityUpdatedAt)
	}
	if !account.UpdatedAt.Equal(firstAccountUpdatedAt) {
		t.Fatalf("Account.UpdatedAt after replay = %v, want %v", account.UpdatedAt, firstAccountUpdatedAt)
	}
}

type whatsAppOwnerControlOnboardingFixtures struct {
	root               string
	providerTarget     AutonomyEligibilityTargetRef
	accountClassTarget AutonomyEligibilityTargetRef
	identity           FrankIdentityRecord
	account            FrankAccountRecord
}

func writeWhatsAppOwnerControlOnboardingFixtures(t *testing.T) whatsAppOwnerControlOnboardingFixtures {
	t.Helper()

	root := t.TempDir()
	now := time.Date(2026, 4, 18, 18, 0, 0, 0, time.UTC)
	providerTarget := AutonomyEligibilityTargetRef{
		Kind:       EligibilityTargetKindProvider,
		RegistryID: "provider-whatsapp-owner-control",
	}
	accountClassTarget := AutonomyEligibilityTargetRef{
		Kind:       EligibilityTargetKindAccountClass,
		RegistryID: "account-class-whatsapp-owner-control",
	}

	writeFrankRegistryEligibilityFixture(t, root, providerTarget, EligibilityLabelAutonomyCompatible, "whatsapp owner-control", "check-provider-whatsapp", now)
	writeFrankRegistryEligibilityFixture(t, root, accountClassTarget, EligibilityLabelAutonomyCompatible, "whatsapp owner-control account", "check-account-class-whatsapp", now.Add(time.Minute))

	identity := FrankIdentityRecord{
		RecordVersion:        StoreRecordVersion,
		IdentityID:           "identity-whatsapp-owner-control",
		IdentityKind:         "owner_control_channel",
		DisplayName:          "WhatsApp Owner Control",
		ProviderOrPlatformID: providerTarget.RegistryID,
		WhatsAppOwnerControl: &FrankWhatsAppOwnerControlIdentity{},
		IdentityMode:         IdentityModeOwnerOnlyControl,
		State:                "candidate",
		EligibilityTargetRef: providerTarget,
		CreatedAt:            now,
		UpdatedAt:            now.Add(time.Minute),
	}
	if err := StoreFrankIdentityRecord(root, identity); err != nil {
		t.Fatalf("StoreFrankIdentityRecord() error = %v", err)
	}

	account := FrankAccountRecord{
		RecordVersion:        StoreRecordVersion,
		AccountID:            "account-whatsapp-owner-control",
		AccountKind:          "owner_control_channel",
		Label:                "WhatsApp Owner Control",
		ProviderOrPlatformID: providerTarget.RegistryID,
		WhatsAppOwnerControl: &FrankWhatsAppOwnerControlAccount{},
		IdentityID:           identity.IdentityID,
		ControlModel:         "owner_controlled",
		RecoveryModel:        "owner_recoverable",
		State:                "candidate",
		EligibilityTargetRef: accountClassTarget,
		CreatedAt:            now.Add(2 * time.Minute),
		UpdatedAt:            now.Add(3 * time.Minute),
	}
	if err := StoreFrankAccountRecord(root, account); err != nil {
		t.Fatalf("StoreFrankAccountRecord() error = %v", err)
	}

	return whatsAppOwnerControlOnboardingFixtures{
		root:               root,
		providerTarget:     providerTarget,
		accountClassTarget: accountClassTarget,
		identity:           identity,
		account:            account,
	}
}

func configureWhatsAppOwnerControlTestConfig(t *testing.T, ownerUserID string, dbPath string, workspace string) {
	t.Helper()

	home := t.TempDir()
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.Workspace = workspace
	cfg.Channels.WhatsApp.Enabled = true
	cfg.Channels.WhatsApp.DBPath = dbPath
	cfg.Channels.WhatsApp.AllowFrom = []string{ownerUserID}
	if err := config.SaveConfig(cfg, filepath.Join(home, ".picobot", "config.json")); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}
	t.Setenv("HOME", home)
}
