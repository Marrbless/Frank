package missioncontrol

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestResolveExecutionContextFrankGoogleOnboardingBundleResolvesExactLinkedBundle(t *testing.T) {
	t.Parallel()

	fixtures := writeGoogleOnboardingFixtures(t)
	ec := testExecutionContextWithFrankObjectRefs(t, []FrankRegistryObjectRef{
		{Kind: FrankRegistryObjectKindIdentity, ObjectID: fixtures.identity.IdentityID},
		{Kind: FrankRegistryObjectKindAccount, ObjectID: fixtures.account.AccountID},
	})
	ec.MissionStoreRoot = fixtures.root

	got, ok, err := ResolveExecutionContextFrankGoogleOnboardingBundle(ec)
	if err != nil {
		t.Fatalf("ResolveExecutionContextFrankGoogleOnboardingBundle() error = %v", err)
	}
	if !ok {
		t.Fatal("ResolveExecutionContextFrankGoogleOnboardingBundle() ok = false, want true")
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

func TestResolveExecutionContextFrankGoogleOnboardingBundleFailsClosedOnMissingAccount(t *testing.T) {
	t.Parallel()

	fixtures := writeGoogleOnboardingFixtures(t)
	ec := testExecutionContextWithFrankObjectRefs(t, []FrankRegistryObjectRef{
		{Kind: FrankRegistryObjectKindIdentity, ObjectID: fixtures.identity.IdentityID},
	})
	ec.MissionStoreRoot = fixtures.root

	_, _, err := ResolveExecutionContextFrankGoogleOnboardingBundle(ec)
	if err == nil {
		t.Fatal("ResolveExecutionContextFrankGoogleOnboardingBundle() error = nil, want missing account rejection")
	}
	if !strings.Contains(err.Error(), "must resolve exactly one google account, got 0") {
		t.Fatalf("ResolveExecutionContextFrankGoogleOnboardingBundle() error = %q, want missing account rejection", err.Error())
	}
}

func TestProduceFrankGoogleOnboardingPersistsConfirmedIdentifiersAndReplaysDeterministically(t *testing.T) {
	fixtures := writeGoogleOnboardingFixtures(t)
	t.Setenv("PICOBOT_GOOGLE_ACCESS_TOKEN", "google-access-token")

	originalRead := readGoogleUserInfo
	defer func() { readGoogleUserInfo = originalRead }()
	readGoogleUserInfo = func(ctx context.Context, accessToken string) (GoogleUserInfoReadback, error) {
		if accessToken != "google-access-token" {
			t.Fatalf("readGoogleUserInfo() accessToken = %q, want %q", accessToken, "google-access-token")
		}
		return GoogleUserInfoReadback{
			GoogleSub:     "google-sub-123456",
			Email:         "frank@google.example",
			EmailVerified: boolPtr(true),
			Name:          "Frank Bot",
			PictureURL:    "https://example.test/frank.png",
		}, nil
	}

	ec := testExecutionContextWithFrankObjectRefs(t, []FrankRegistryObjectRef{
		{Kind: FrankRegistryObjectKindIdentity, ObjectID: fixtures.identity.IdentityID},
		{Kind: FrankRegistryObjectKindAccount, ObjectID: fixtures.account.AccountID},
	})
	ec.MissionStoreRoot = fixtures.root
	bundle, ok, err := ResolveExecutionContextFrankGoogleOnboardingBundle(ec)
	if err != nil {
		t.Fatalf("ResolveExecutionContextFrankGoogleOnboardingBundle() error = %v", err)
	}
	if !ok {
		t.Fatal("ResolveExecutionContextFrankGoogleOnboardingBundle() ok = false, want true")
	}

	firstNow := time.Date(2026, 4, 19, 1, 0, 0, 0, time.UTC)
	if err := ProduceFrankGoogleOnboarding(fixtures.root, bundle, firstNow); err != nil {
		t.Fatalf("ProduceFrankGoogleOnboarding(first) error = %v", err)
	}

	identity, err := LoadFrankIdentityRecord(fixtures.root, fixtures.identity.IdentityID)
	if err != nil {
		t.Fatalf("LoadFrankIdentityRecord() error = %v", err)
	}
	account, err := LoadFrankAccountRecord(fixtures.root, fixtures.account.AccountID)
	if err != nil {
		t.Fatalf("LoadFrankAccountRecord() error = %v", err)
	}
	if identity.Google == nil || identity.Google.GoogleSub != "google-sub-123456" || identity.Google.Email != "frank@google.example" || identity.Google.Name != "Frank Bot" || identity.Google.PictureURL != "https://example.test/frank.png" {
		t.Fatalf("Identity.Google = %#v, want google identity fields persisted", identity.Google)
	}
	if identity.Google.EmailVerified == nil || !*identity.Google.EmailVerified {
		t.Fatalf("Identity.Google.EmailVerified = %#v, want true", identity.Google.EmailVerified)
	}
	if account.Google == nil || account.Google.OAuthAccessTokenEnvVarRef != "PICOBOT_GOOGLE_ACCESS_TOKEN" || !account.Google.ConfirmedAuthenticated {
		t.Fatalf("Account.Google = %#v, want access-token ref and confirmation persisted", account.Google)
	}
	firstIdentityUpdatedAt := identity.UpdatedAt
	firstAccountUpdatedAt := account.UpdatedAt

	bundle, ok, err = ResolveExecutionContextFrankGoogleOnboardingBundle(ec)
	if err != nil {
		t.Fatalf("ResolveExecutionContextFrankGoogleOnboardingBundle(replay) error = %v", err)
	}
	if !ok {
		t.Fatal("ResolveExecutionContextFrankGoogleOnboardingBundle(replay) ok = false, want true")
	}

	if err := ProduceFrankGoogleOnboarding(fixtures.root, bundle, firstNow.Add(time.Minute)); err != nil {
		t.Fatalf("ProduceFrankGoogleOnboarding(replay) error = %v, want deterministic no-op success", err)
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

type googleOnboardingFixtures struct {
	root               string
	providerTarget     AutonomyEligibilityTargetRef
	accountClassTarget AutonomyEligibilityTargetRef
	identity           FrankIdentityRecord
	account            FrankAccountRecord
}

func writeGoogleOnboardingFixtures(t *testing.T) googleOnboardingFixtures {
	t.Helper()

	root := t.TempDir()
	now := time.Date(2026, 4, 18, 23, 0, 0, 0, time.UTC)
	providerTarget := AutonomyEligibilityTargetRef{
		Kind:       EligibilityTargetKindProvider,
		RegistryID: "provider-google",
	}
	accountClassTarget := AutonomyEligibilityTargetRef{
		Kind:       EligibilityTargetKindAccountClass,
		RegistryID: "account-class-google",
	}

	writeFrankRegistryEligibilityFixture(t, root, providerTarget, EligibilityLabelAutonomyCompatible, "google", "check-provider-google", now)
	writeFrankRegistryEligibilityFixture(t, root, accountClassTarget, EligibilityLabelAutonomyCompatible, "google account", "check-account-class-google", now.Add(time.Minute))

	identity := FrankIdentityRecord{
		RecordVersion:        StoreRecordVersion,
		IdentityID:           "identity-google",
		IdentityKind:         "platform_identity",
		DisplayName:          "Frank Google",
		ProviderOrPlatformID: providerTarget.RegistryID,
		Google:               &FrankGoogleIdentity{},
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
		AccountID:            "account-google",
		AccountKind:          "platform_account",
		Label:                "Frank Google Account",
		ProviderOrPlatformID: providerTarget.RegistryID,
		Google: &FrankGoogleAccount{
			OAuthAccessTokenEnvVarRef: "PICOBOT_GOOGLE_ACCESS_TOKEN",
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

	return googleOnboardingFixtures{
		root:               root,
		providerTarget:     providerTarget,
		accountClassTarget: accountClassTarget,
		identity:           identity,
		account:            account,
	}
}
