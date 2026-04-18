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

func TestResolveExecutionContextFrankTelegramOwnerControlOnboardingBundleResolvesExactLinkedBundle(t *testing.T) {
	t.Parallel()

	fixtures := writeTelegramOwnerControlOnboardingFixtures(t)
	ec := testExecutionContextWithFrankObjectRefs(t, []FrankRegistryObjectRef{
		{Kind: FrankRegistryObjectKindIdentity, ObjectID: fixtures.identity.IdentityID},
		{Kind: FrankRegistryObjectKindAccount, ObjectID: fixtures.account.AccountID},
	})
	ec.MissionStoreRoot = fixtures.root

	got, ok, err := ResolveExecutionContextFrankTelegramOwnerControlOnboardingBundle(ec)
	if err != nil {
		t.Fatalf("ResolveExecutionContextFrankTelegramOwnerControlOnboardingBundle() error = %v", err)
	}
	if !ok {
		t.Fatal("ResolveExecutionContextFrankTelegramOwnerControlOnboardingBundle() ok = false, want true")
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

func TestResolveExecutionContextFrankTelegramOwnerControlOnboardingBundleFailsClosedOnMissingAccount(t *testing.T) {
	t.Parallel()

	fixtures := writeTelegramOwnerControlOnboardingFixtures(t)
	ec := testExecutionContextWithFrankObjectRefs(t, []FrankRegistryObjectRef{
		{Kind: FrankRegistryObjectKindIdentity, ObjectID: fixtures.identity.IdentityID},
	})
	ec.MissionStoreRoot = fixtures.root

	_, _, err := ResolveExecutionContextFrankTelegramOwnerControlOnboardingBundle(ec)
	if err == nil {
		t.Fatal("ResolveExecutionContextFrankTelegramOwnerControlOnboardingBundle() error = nil, want missing account rejection")
	}
	if !strings.Contains(err.Error(), "must resolve exactly one telegram owner-control account, got 0") {
		t.Fatalf("ResolveExecutionContextFrankTelegramOwnerControlOnboardingBundle() error = %q, want missing account rejection", err.Error())
	}
}

func TestProduceFrankTelegramOwnerControlOnboardingPersistsConfirmedIdentifiersAndReplaysDeterministically(t *testing.T) {
	fixtures := writeTelegramOwnerControlOnboardingFixtures(t)
	configureTelegramOwnerControlTestConfig(t, "999111222", filepath.Join(t.TempDir(), "workspace"))

	originalRead := readTelegramOwnerControlBotIdentity
	defer func() { readTelegramOwnerControlBotIdentity = originalRead }()
	readTelegramOwnerControlBotIdentity = func(ctx context.Context, token string) (channels.TelegramBotIdentity, error) {
		if token != "telegram-token" {
			t.Fatalf("readTelegramOwnerControlBotIdentity() token = %q, want %q", token, "telegram-token")
		}
		return channels.TelegramBotIdentity{BotUserID: "777888999", Username: "frank_owner_bot"}, nil
	}

	ec := testExecutionContextWithFrankObjectRefs(t, []FrankRegistryObjectRef{
		{Kind: FrankRegistryObjectKindIdentity, ObjectID: fixtures.identity.IdentityID},
		{Kind: FrankRegistryObjectKindAccount, ObjectID: fixtures.account.AccountID},
	})
	ec.MissionStoreRoot = fixtures.root
	bundle, ok, err := ResolveExecutionContextFrankTelegramOwnerControlOnboardingBundle(ec)
	if err != nil {
		t.Fatalf("ResolveExecutionContextFrankTelegramOwnerControlOnboardingBundle() error = %v", err)
	}
	if !ok {
		t.Fatal("ResolveExecutionContextFrankTelegramOwnerControlOnboardingBundle() ok = false, want true")
	}

	firstNow := time.Date(2026, 4, 18, 19, 0, 0, 0, time.UTC)
	if err := ProduceFrankTelegramOwnerControlOnboarding(fixtures.root, bundle, firstNow); err != nil {
		t.Fatalf("ProduceFrankTelegramOwnerControlOnboarding(first) error = %v", err)
	}

	identity, err := LoadFrankIdentityRecord(fixtures.root, fixtures.identity.IdentityID)
	if err != nil {
		t.Fatalf("LoadFrankIdentityRecord() error = %v", err)
	}
	account, err := LoadFrankAccountRecord(fixtures.root, fixtures.account.AccountID)
	if err != nil {
		t.Fatalf("LoadFrankAccountRecord() error = %v", err)
	}
	if identity.TelegramOwnerControl == nil || identity.TelegramOwnerControl.OwnerUserID != "999111222" {
		t.Fatalf("Identity.TelegramOwnerControl = %#v, want owner_user_id %q", identity.TelegramOwnerControl, "999111222")
	}
	if account.TelegramOwnerControl == nil || account.TelegramOwnerControl.BotUserID != "777888999" {
		t.Fatalf("Account.TelegramOwnerControl = %#v, want bot_user_id %q", account.TelegramOwnerControl, "777888999")
	}
	firstIdentityUpdatedAt := identity.UpdatedAt
	firstAccountUpdatedAt := account.UpdatedAt

	bundle, ok, err = ResolveExecutionContextFrankTelegramOwnerControlOnboardingBundle(ec)
	if err != nil {
		t.Fatalf("ResolveExecutionContextFrankTelegramOwnerControlOnboardingBundle(replay) error = %v", err)
	}
	if !ok {
		t.Fatal("ResolveExecutionContextFrankTelegramOwnerControlOnboardingBundle(replay) ok = false, want true")
	}

	if err := ProduceFrankTelegramOwnerControlOnboarding(fixtures.root, bundle, firstNow.Add(time.Minute)); err != nil {
		t.Fatalf("ProduceFrankTelegramOwnerControlOnboarding(replay) error = %v, want deterministic no-op success", err)
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

type telegramOwnerControlOnboardingFixtures struct {
	root               string
	providerTarget     AutonomyEligibilityTargetRef
	accountClassTarget AutonomyEligibilityTargetRef
	identity           FrankIdentityRecord
	account            FrankAccountRecord
}

func writeTelegramOwnerControlOnboardingFixtures(t *testing.T) telegramOwnerControlOnboardingFixtures {
	t.Helper()

	root := t.TempDir()
	now := time.Date(2026, 4, 18, 18, 0, 0, 0, time.UTC)
	providerTarget := AutonomyEligibilityTargetRef{
		Kind:       EligibilityTargetKindProvider,
		RegistryID: "provider-telegram-owner-control",
	}
	accountClassTarget := AutonomyEligibilityTargetRef{
		Kind:       EligibilityTargetKindAccountClass,
		RegistryID: "account-class-telegram-owner-control",
	}

	writeFrankRegistryEligibilityFixture(t, root, providerTarget, EligibilityLabelAutonomyCompatible, "telegram owner-control", "check-provider-telegram", now)
	writeFrankRegistryEligibilityFixture(t, root, accountClassTarget, EligibilityLabelAutonomyCompatible, "telegram owner-control account", "check-account-class-telegram", now.Add(time.Minute))

	identity := FrankIdentityRecord{
		RecordVersion:        StoreRecordVersion,
		IdentityID:           "identity-telegram-owner-control",
		IdentityKind:         "owner_control_channel",
		DisplayName:          "Telegram Owner Control",
		ProviderOrPlatformID: providerTarget.RegistryID,
		TelegramOwnerControl: &FrankTelegramOwnerControlIdentity{},
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
		AccountID:            "account-telegram-owner-control",
		AccountKind:          "owner_control_channel",
		Label:                "Telegram Owner Control",
		ProviderOrPlatformID: providerTarget.RegistryID,
		TelegramOwnerControl: &FrankTelegramOwnerControlAccount{},
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

	return telegramOwnerControlOnboardingFixtures{
		root:               root,
		providerTarget:     providerTarget,
		accountClassTarget: accountClassTarget,
		identity:           identity,
		account:            account,
	}
}

func configureTelegramOwnerControlTestConfig(t *testing.T, ownerUserID string, workspace string) {
	t.Helper()

	home := t.TempDir()
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.Workspace = workspace
	cfg.Channels.Telegram.Enabled = true
	cfg.Channels.Telegram.Token = "telegram-token"
	cfg.Channels.Telegram.AllowFrom = []string{ownerUserID}
	if err := config.SaveConfig(cfg, filepath.Join(home, ".picobot", "config.json")); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}
	t.Setenv("HOME", home)
}
