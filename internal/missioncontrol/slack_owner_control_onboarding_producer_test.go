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

func TestResolveExecutionContextFrankSlackOwnerControlOnboardingBundleResolvesExactLinkedBundle(t *testing.T) {
	t.Parallel()

	fixtures := writeSlackOwnerControlOnboardingFixtures(t)
	ec := testExecutionContextWithFrankObjectRefs(t, []FrankRegistryObjectRef{
		{Kind: FrankRegistryObjectKindIdentity, ObjectID: fixtures.identity.IdentityID},
		{Kind: FrankRegistryObjectKindAccount, ObjectID: fixtures.account.AccountID},
	})
	ec.MissionStoreRoot = fixtures.root

	got, ok, err := ResolveExecutionContextFrankSlackOwnerControlOnboardingBundle(ec)
	if err != nil {
		t.Fatalf("ResolveExecutionContextFrankSlackOwnerControlOnboardingBundle() error = %v", err)
	}
	if !ok {
		t.Fatal("ResolveExecutionContextFrankSlackOwnerControlOnboardingBundle() ok = false, want true")
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

func TestResolveExecutionContextFrankSlackOwnerControlOnboardingBundleFailsClosedOnMissingAccount(t *testing.T) {
	t.Parallel()

	fixtures := writeSlackOwnerControlOnboardingFixtures(t)
	ec := testExecutionContextWithFrankObjectRefs(t, []FrankRegistryObjectRef{
		{Kind: FrankRegistryObjectKindIdentity, ObjectID: fixtures.identity.IdentityID},
	})
	ec.MissionStoreRoot = fixtures.root

	_, _, err := ResolveExecutionContextFrankSlackOwnerControlOnboardingBundle(ec)
	if err == nil {
		t.Fatal("ResolveExecutionContextFrankSlackOwnerControlOnboardingBundle() error = nil, want missing account rejection")
	}
	if !strings.Contains(err.Error(), "must resolve exactly one slack owner-control account, got 0") {
		t.Fatalf("ResolveExecutionContextFrankSlackOwnerControlOnboardingBundle() error = %q, want missing account rejection", err.Error())
	}
}

func TestProduceFrankSlackOwnerControlOnboardingPersistsConfirmedIdentifiersAndReplaysDeterministically(t *testing.T) {
	fixtures := writeSlackOwnerControlOnboardingFixtures(t)
	configureSlackOwnerControlTestConfig(t, "UOWNER1", filepath.Join(t.TempDir(), "workspace"))

	originalRead := readSlackOwnerControlIdentity
	defer func() { readSlackOwnerControlIdentity = originalRead }()
	readSlackOwnerControlIdentity = func(ctx context.Context, botToken string) (channels.SlackAuthIdentity, error) {
		if botToken != "xoxb-slack-bot-token" {
			t.Fatalf("readSlackOwnerControlIdentity() botToken = %q, want %q", botToken, "xoxb-slack-bot-token")
		}
		return channels.SlackAuthIdentity{
			TeamID: "T123456",
			UserID: "U234567",
			BotID:  "B345678",
		}, nil
	}

	ec := testExecutionContextWithFrankObjectRefs(t, []FrankRegistryObjectRef{
		{Kind: FrankRegistryObjectKindIdentity, ObjectID: fixtures.identity.IdentityID},
		{Kind: FrankRegistryObjectKindAccount, ObjectID: fixtures.account.AccountID},
	})
	ec.MissionStoreRoot = fixtures.root
	bundle, ok, err := ResolveExecutionContextFrankSlackOwnerControlOnboardingBundle(ec)
	if err != nil {
		t.Fatalf("ResolveExecutionContextFrankSlackOwnerControlOnboardingBundle() error = %v", err)
	}
	if !ok {
		t.Fatal("ResolveExecutionContextFrankSlackOwnerControlOnboardingBundle() ok = false, want true")
	}

	firstNow := time.Date(2026, 4, 18, 19, 0, 0, 0, time.UTC)
	if err := ProduceFrankSlackOwnerControlOnboarding(fixtures.root, bundle, firstNow); err != nil {
		t.Fatalf("ProduceFrankSlackOwnerControlOnboarding(first) error = %v", err)
	}

	identity, err := LoadFrankIdentityRecord(fixtures.root, fixtures.identity.IdentityID)
	if err != nil {
		t.Fatalf("LoadFrankIdentityRecord() error = %v", err)
	}
	account, err := LoadFrankAccountRecord(fixtures.root, fixtures.account.AccountID)
	if err != nil {
		t.Fatalf("LoadFrankAccountRecord() error = %v", err)
	}
	if identity.SlackOwnerControl == nil || identity.SlackOwnerControl.TeamID != "T123456" || identity.SlackOwnerControl.UserID != "U234567" {
		t.Fatalf("Identity.SlackOwnerControl = %#v, want team_id %q user_id %q", identity.SlackOwnerControl, "T123456", "U234567")
	}
	if account.SlackOwnerControl == nil || account.SlackOwnerControl.BotID != "B345678" {
		t.Fatalf("Account.SlackOwnerControl = %#v, want bot_id %q", account.SlackOwnerControl, "B345678")
	}
	firstIdentityUpdatedAt := identity.UpdatedAt
	firstAccountUpdatedAt := account.UpdatedAt

	bundle, ok, err = ResolveExecutionContextFrankSlackOwnerControlOnboardingBundle(ec)
	if err != nil {
		t.Fatalf("ResolveExecutionContextFrankSlackOwnerControlOnboardingBundle(replay) error = %v", err)
	}
	if !ok {
		t.Fatal("ResolveExecutionContextFrankSlackOwnerControlOnboardingBundle(replay) ok = false, want true")
	}

	if err := ProduceFrankSlackOwnerControlOnboarding(fixtures.root, bundle, firstNow.Add(time.Minute)); err != nil {
		t.Fatalf("ProduceFrankSlackOwnerControlOnboarding(replay) error = %v, want deterministic no-op success", err)
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

type slackOwnerControlOnboardingFixtures struct {
	root               string
	providerTarget     AutonomyEligibilityTargetRef
	accountClassTarget AutonomyEligibilityTargetRef
	identity           FrankIdentityRecord
	account            FrankAccountRecord
}

func writeSlackOwnerControlOnboardingFixtures(t *testing.T) slackOwnerControlOnboardingFixtures {
	t.Helper()

	root := t.TempDir()
	now := time.Date(2026, 4, 18, 18, 0, 0, 0, time.UTC)
	providerTarget := AutonomyEligibilityTargetRef{
		Kind:       EligibilityTargetKindProvider,
		RegistryID: "provider-slack-owner-control",
	}
	accountClassTarget := AutonomyEligibilityTargetRef{
		Kind:       EligibilityTargetKindAccountClass,
		RegistryID: "account-class-slack-owner-control",
	}

	writeFrankRegistryEligibilityFixture(t, root, providerTarget, EligibilityLabelAutonomyCompatible, "slack owner-control", "check-provider-slack", now)
	writeFrankRegistryEligibilityFixture(t, root, accountClassTarget, EligibilityLabelAutonomyCompatible, "slack owner-control account", "check-account-class-slack", now.Add(time.Minute))

	identity := FrankIdentityRecord{
		RecordVersion:        StoreRecordVersion,
		IdentityID:           "identity-slack-owner-control",
		IdentityKind:         "owner_control_channel",
		DisplayName:          "Slack Owner Control",
		ProviderOrPlatformID: providerTarget.RegistryID,
		SlackOwnerControl:    &FrankSlackOwnerControlIdentity{},
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
		AccountID:            "account-slack-owner-control",
		AccountKind:          "owner_control_channel",
		Label:                "Slack Owner Control",
		ProviderOrPlatformID: providerTarget.RegistryID,
		SlackOwnerControl:    &FrankSlackOwnerControlAccount{},
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

	return slackOwnerControlOnboardingFixtures{
		root:               root,
		providerTarget:     providerTarget,
		accountClassTarget: accountClassTarget,
		identity:           identity,
		account:            account,
	}
}

func configureSlackOwnerControlTestConfig(t *testing.T, ownerUserID string, workspace string) {
	t.Helper()

	home := t.TempDir()
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.Workspace = workspace
	cfg.Channels.Slack.Enabled = true
	cfg.Channels.Slack.AppToken = "xapp-slack-app-token"
	cfg.Channels.Slack.BotToken = "xoxb-slack-bot-token"
	cfg.Channels.Slack.AllowUsers = []string{ownerUserID}
	if err := config.SaveConfig(cfg, filepath.Join(home, ".picobot", "config.json")); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}
	t.Setenv("HOME", home)
}
