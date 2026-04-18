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

func TestResolveExecutionContextFrankDiscordOwnerControlOnboardingBundleResolvesExactLinkedBundle(t *testing.T) {
	t.Parallel()

	fixtures := writeDiscordOwnerControlOnboardingFixtures(t)
	ec := testExecutionContextWithFrankObjectRefs(t, []FrankRegistryObjectRef{
		{Kind: FrankRegistryObjectKindIdentity, ObjectID: fixtures.identity.IdentityID},
		{Kind: FrankRegistryObjectKindAccount, ObjectID: fixtures.account.AccountID},
	})
	ec.MissionStoreRoot = fixtures.root

	got, ok, err := ResolveExecutionContextFrankDiscordOwnerControlOnboardingBundle(ec)
	if err != nil {
		t.Fatalf("ResolveExecutionContextFrankDiscordOwnerControlOnboardingBundle() error = %v", err)
	}
	if !ok {
		t.Fatal("ResolveExecutionContextFrankDiscordOwnerControlOnboardingBundle() ok = false, want true")
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

func TestResolveExecutionContextFrankDiscordOwnerControlOnboardingBundleFailsClosedOnMissingAccount(t *testing.T) {
	t.Parallel()

	fixtures := writeDiscordOwnerControlOnboardingFixtures(t)
	ec := testExecutionContextWithFrankObjectRefs(t, []FrankRegistryObjectRef{
		{Kind: FrankRegistryObjectKindIdentity, ObjectID: fixtures.identity.IdentityID},
	})
	ec.MissionStoreRoot = fixtures.root

	_, _, err := ResolveExecutionContextFrankDiscordOwnerControlOnboardingBundle(ec)
	if err == nil {
		t.Fatal("ResolveExecutionContextFrankDiscordOwnerControlOnboardingBundle() error = nil, want missing account rejection")
	}
	if !strings.Contains(err.Error(), "must resolve exactly one discord owner-control account, got 0") {
		t.Fatalf("ResolveExecutionContextFrankDiscordOwnerControlOnboardingBundle() error = %q, want missing account rejection", err.Error())
	}
}

func TestProduceFrankDiscordOwnerControlOnboardingPersistsConfirmedIdentifiersAndReplaysDeterministically(t *testing.T) {
	fixtures := writeDiscordOwnerControlOnboardingFixtures(t)
	configureDiscordOwnerControlTestConfig(t, "123456789012345678", filepath.Join(t.TempDir(), "workspace"))

	originalRead := readDiscordOwnerControlBotIdentity
	defer func() { readDiscordOwnerControlBotIdentity = originalRead }()
	readDiscordOwnerControlBotIdentity = func(ctx context.Context, token string) (channels.DiscordBotIdentity, error) {
		if token != "discord-bot-token" {
			t.Fatalf("readDiscordOwnerControlBotIdentity() token = %q, want %q", token, "discord-bot-token")
		}
		return channels.DiscordBotIdentity{
			BotUserID:     "777888999000111222",
			Username:      "frankbot",
			GlobalName:    "Frank",
			Discriminator: "1234",
		}, nil
	}

	ec := testExecutionContextWithFrankObjectRefs(t, []FrankRegistryObjectRef{
		{Kind: FrankRegistryObjectKindIdentity, ObjectID: fixtures.identity.IdentityID},
		{Kind: FrankRegistryObjectKindAccount, ObjectID: fixtures.account.AccountID},
	})
	ec.MissionStoreRoot = fixtures.root
	bundle, ok, err := ResolveExecutionContextFrankDiscordOwnerControlOnboardingBundle(ec)
	if err != nil {
		t.Fatalf("ResolveExecutionContextFrankDiscordOwnerControlOnboardingBundle() error = %v", err)
	}
	if !ok {
		t.Fatal("ResolveExecutionContextFrankDiscordOwnerControlOnboardingBundle() ok = false, want true")
	}

	firstNow := time.Date(2026, 4, 18, 19, 0, 0, 0, time.UTC)
	if err := ProduceFrankDiscordOwnerControlOnboarding(fixtures.root, bundle, firstNow); err != nil {
		t.Fatalf("ProduceFrankDiscordOwnerControlOnboarding(first) error = %v", err)
	}

	identity, err := LoadFrankIdentityRecord(fixtures.root, fixtures.identity.IdentityID)
	if err != nil {
		t.Fatalf("LoadFrankIdentityRecord() error = %v", err)
	}
	account, err := LoadFrankAccountRecord(fixtures.root, fixtures.account.AccountID)
	if err != nil {
		t.Fatalf("LoadFrankAccountRecord() error = %v", err)
	}
	if identity.DiscordOwnerControl == nil || identity.DiscordOwnerControl.Username != "frankbot" || identity.DiscordOwnerControl.GlobalName != "Frank" || identity.DiscordOwnerControl.Discriminator != "1234" {
		t.Fatalf("Identity.DiscordOwnerControl = %#v, want username/global_name/discriminator persisted", identity.DiscordOwnerControl)
	}
	if account.DiscordOwnerControl == nil || account.DiscordOwnerControl.BotUserID != "777888999000111222" {
		t.Fatalf("Account.DiscordOwnerControl = %#v, want bot_user_id %q", account.DiscordOwnerControl, "777888999000111222")
	}
	firstIdentityUpdatedAt := identity.UpdatedAt
	firstAccountUpdatedAt := account.UpdatedAt

	bundle, ok, err = ResolveExecutionContextFrankDiscordOwnerControlOnboardingBundle(ec)
	if err != nil {
		t.Fatalf("ResolveExecutionContextFrankDiscordOwnerControlOnboardingBundle(replay) error = %v", err)
	}
	if !ok {
		t.Fatal("ResolveExecutionContextFrankDiscordOwnerControlOnboardingBundle(replay) ok = false, want true")
	}

	if err := ProduceFrankDiscordOwnerControlOnboarding(fixtures.root, bundle, firstNow.Add(time.Minute)); err != nil {
		t.Fatalf("ProduceFrankDiscordOwnerControlOnboarding(replay) error = %v, want deterministic no-op success", err)
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

type discordOwnerControlOnboardingFixtures struct {
	root               string
	providerTarget     AutonomyEligibilityTargetRef
	accountClassTarget AutonomyEligibilityTargetRef
	identity           FrankIdentityRecord
	account            FrankAccountRecord
}

func writeDiscordOwnerControlOnboardingFixtures(t *testing.T) discordOwnerControlOnboardingFixtures {
	t.Helper()

	root := t.TempDir()
	now := time.Date(2026, 4, 18, 18, 0, 0, 0, time.UTC)
	providerTarget := AutonomyEligibilityTargetRef{
		Kind:       EligibilityTargetKindProvider,
		RegistryID: "provider-discord-owner-control",
	}
	accountClassTarget := AutonomyEligibilityTargetRef{
		Kind:       EligibilityTargetKindAccountClass,
		RegistryID: "account-class-discord-owner-control",
	}

	writeFrankRegistryEligibilityFixture(t, root, providerTarget, EligibilityLabelAutonomyCompatible, "discord owner-control", "check-provider-discord", now)
	writeFrankRegistryEligibilityFixture(t, root, accountClassTarget, EligibilityLabelAutonomyCompatible, "discord owner-control account", "check-account-class-discord", now.Add(time.Minute))

	identity := FrankIdentityRecord{
		RecordVersion:        StoreRecordVersion,
		IdentityID:           "identity-discord-owner-control",
		IdentityKind:         "owner_control_channel",
		DisplayName:          "Discord Owner Control",
		ProviderOrPlatformID: providerTarget.RegistryID,
		DiscordOwnerControl:  &FrankDiscordOwnerControlIdentity{},
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
		AccountID:            "account-discord-owner-control",
		AccountKind:          "owner_control_channel",
		Label:                "Discord Owner Control",
		ProviderOrPlatformID: providerTarget.RegistryID,
		DiscordOwnerControl:  &FrankDiscordOwnerControlAccount{},
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

	return discordOwnerControlOnboardingFixtures{
		root:               root,
		providerTarget:     providerTarget,
		accountClassTarget: accountClassTarget,
		identity:           identity,
		account:            account,
	}
}

func configureDiscordOwnerControlTestConfig(t *testing.T, ownerUserID string, workspace string) {
	t.Helper()

	home := t.TempDir()
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.Workspace = workspace
	cfg.Channels.Discord.Enabled = true
	cfg.Channels.Discord.Token = "discord-bot-token"
	cfg.Channels.Discord.AllowFrom = []string{ownerUserID}
	if err := config.SaveConfig(cfg, filepath.Join(home, ".picobot", "config.json")); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}
	t.Setenv("HOME", home)
}
