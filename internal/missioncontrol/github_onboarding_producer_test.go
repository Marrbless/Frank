package missioncontrol

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestResolveExecutionContextFrankGitHubOnboardingBundleResolvesExactLinkedBundle(t *testing.T) {
	t.Parallel()

	fixtures := writeGitHubOnboardingFixtures(t)
	ec := testExecutionContextWithFrankObjectRefs(t, []FrankRegistryObjectRef{
		{Kind: FrankRegistryObjectKindIdentity, ObjectID: fixtures.identity.IdentityID},
		{Kind: FrankRegistryObjectKindAccount, ObjectID: fixtures.account.AccountID},
	})
	ec.MissionStoreRoot = fixtures.root

	got, ok, err := ResolveExecutionContextFrankGitHubOnboardingBundle(ec)
	if err != nil {
		t.Fatalf("ResolveExecutionContextFrankGitHubOnboardingBundle() error = %v", err)
	}
	if !ok {
		t.Fatal("ResolveExecutionContextFrankGitHubOnboardingBundle() ok = false, want true")
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

func TestResolveExecutionContextFrankGitHubOnboardingBundleFailsClosedOnMissingAccount(t *testing.T) {
	t.Parallel()

	fixtures := writeGitHubOnboardingFixtures(t)
	ec := testExecutionContextWithFrankObjectRefs(t, []FrankRegistryObjectRef{
		{Kind: FrankRegistryObjectKindIdentity, ObjectID: fixtures.identity.IdentityID},
	})
	ec.MissionStoreRoot = fixtures.root

	_, _, err := ResolveExecutionContextFrankGitHubOnboardingBundle(ec)
	if err == nil {
		t.Fatal("ResolveExecutionContextFrankGitHubOnboardingBundle() error = nil, want missing account rejection")
	}
	if !strings.Contains(err.Error(), "must resolve exactly one github account, got 0") {
		t.Fatalf("ResolveExecutionContextFrankGitHubOnboardingBundle() error = %q, want missing account rejection", err.Error())
	}
}

func TestProduceFrankGitHubOnboardingPersistsConfirmedIdentifiersAndReplaysDeterministically(t *testing.T) {
	fixtures := writeGitHubOnboardingFixtures(t)
	t.Setenv("PICOBOT_GITHUB_TOKEN", "github-token")

	originalReadUser := readGitHubAuthenticatedUser
	originalReadEmail := readGitHubPrimaryVerifiedEmail
	defer func() {
		readGitHubAuthenticatedUser = originalReadUser
		readGitHubPrimaryVerifiedEmail = originalReadEmail
	}()
	readGitHubAuthenticatedUser = func(ctx context.Context, token string) (GitHubAuthenticatedUser, error) {
		if token != "github-token" {
			t.Fatalf("readGitHubAuthenticatedUser() token = %q, want %q", token, "github-token")
		}
		return GitHubAuthenticatedUser{
			GitHubUserID: "1234567",
			Login:        "frank-bot",
			NodeID:       "MDQ6VXNlcjEyMzQ1Njc=",
		}, nil
	}
	readGitHubPrimaryVerifiedEmail = func(ctx context.Context, token string) (string, bool, error) {
		if token != "github-token" {
			t.Fatalf("readGitHubPrimaryVerifiedEmail() token = %q, want %q", token, "github-token")
		}
		return "frank@example.com", true, nil
	}

	ec := testExecutionContextWithFrankObjectRefs(t, []FrankRegistryObjectRef{
		{Kind: FrankRegistryObjectKindIdentity, ObjectID: fixtures.identity.IdentityID},
		{Kind: FrankRegistryObjectKindAccount, ObjectID: fixtures.account.AccountID},
	})
	ec.MissionStoreRoot = fixtures.root
	bundle, ok, err := ResolveExecutionContextFrankGitHubOnboardingBundle(ec)
	if err != nil {
		t.Fatalf("ResolveExecutionContextFrankGitHubOnboardingBundle() error = %v", err)
	}
	if !ok {
		t.Fatal("ResolveExecutionContextFrankGitHubOnboardingBundle() ok = false, want true")
	}

	firstNow := time.Date(2026, 4, 18, 19, 0, 0, 0, time.UTC)
	if err := ProduceFrankGitHubOnboarding(fixtures.root, bundle, firstNow); err != nil {
		t.Fatalf("ProduceFrankGitHubOnboarding(first) error = %v", err)
	}

	identity, err := LoadFrankIdentityRecord(fixtures.root, fixtures.identity.IdentityID)
	if err != nil {
		t.Fatalf("LoadFrankIdentityRecord() error = %v", err)
	}
	account, err := LoadFrankAccountRecord(fixtures.root, fixtures.account.AccountID)
	if err != nil {
		t.Fatalf("LoadFrankAccountRecord() error = %v", err)
	}
	if identity.GitHub == nil || identity.GitHub.GitHubUserID != "1234567" || identity.GitHub.Login != "frank-bot" || identity.GitHub.NodeID != "MDQ6VXNlcjEyMzQ1Njc=" || identity.GitHub.PrimaryVerifiedEmail != "frank@example.com" {
		t.Fatalf("Identity.GitHub = %#v, want github identity fields persisted", identity.GitHub)
	}
	if account.GitHub == nil || account.GitHub.TokenEnvVarRef != "PICOBOT_GITHUB_TOKEN" || !account.GitHub.ConfirmedAuthenticated {
		t.Fatalf("Account.GitHub = %#v, want token ref and confirmation persisted", account.GitHub)
	}
	firstIdentityUpdatedAt := identity.UpdatedAt
	firstAccountUpdatedAt := account.UpdatedAt

	bundle, ok, err = ResolveExecutionContextFrankGitHubOnboardingBundle(ec)
	if err != nil {
		t.Fatalf("ResolveExecutionContextFrankGitHubOnboardingBundle(replay) error = %v", err)
	}
	if !ok {
		t.Fatal("ResolveExecutionContextFrankGitHubOnboardingBundle(replay) ok = false, want true")
	}

	if err := ProduceFrankGitHubOnboarding(fixtures.root, bundle, firstNow.Add(time.Minute)); err != nil {
		t.Fatalf("ProduceFrankGitHubOnboarding(replay) error = %v, want deterministic no-op success", err)
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

type gitHubOnboardingFixtures struct {
	root               string
	providerTarget     AutonomyEligibilityTargetRef
	accountClassTarget AutonomyEligibilityTargetRef
	identity           FrankIdentityRecord
	account            FrankAccountRecord
}

func writeGitHubOnboardingFixtures(t *testing.T) gitHubOnboardingFixtures {
	t.Helper()

	root := t.TempDir()
	now := time.Date(2026, 4, 18, 18, 0, 0, 0, time.UTC)
	providerTarget := AutonomyEligibilityTargetRef{
		Kind:       EligibilityTargetKindProvider,
		RegistryID: "provider-github",
	}
	accountClassTarget := AutonomyEligibilityTargetRef{
		Kind:       EligibilityTargetKindAccountClass,
		RegistryID: "account-class-github",
	}

	writeFrankRegistryEligibilityFixture(t, root, providerTarget, EligibilityLabelAutonomyCompatible, "github", "check-provider-github", now)
	writeFrankRegistryEligibilityFixture(t, root, accountClassTarget, EligibilityLabelAutonomyCompatible, "github account", "check-account-class-github", now.Add(time.Minute))

	identity := FrankIdentityRecord{
		RecordVersion:        StoreRecordVersion,
		IdentityID:           "identity-github",
		IdentityKind:         "platform_identity",
		DisplayName:          "Frank GitHub",
		ProviderOrPlatformID: providerTarget.RegistryID,
		GitHub:               &FrankGitHubIdentity{},
		IdentityMode:         IdentityModeAgentAlias,
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
		AccountID:            "account-github",
		AccountKind:          "platform_account",
		Label:                "Frank GitHub Account",
		ProviderOrPlatformID: providerTarget.RegistryID,
		GitHub: &FrankGitHubAccount{
			TokenEnvVarRef: "PICOBOT_GITHUB_TOKEN",
		},
		IdentityID:           identity.IdentityID,
		ControlModel:         "agent_managed",
		RecoveryModel:        "env_ref_recoverable",
		State:                "candidate",
		EligibilityTargetRef: accountClassTarget,
		CreatedAt:            now.Add(2 * time.Minute),
		UpdatedAt:            now.Add(3 * time.Minute),
	}
	if err := StoreFrankAccountRecord(root, account); err != nil {
		t.Fatalf("StoreFrankAccountRecord() error = %v", err)
	}

	return gitHubOnboardingFixtures{
		root:               root,
		providerTarget:     providerTarget,
		accountClassTarget: accountClassTarget,
		identity:           identity,
		account:            account,
	}
}
