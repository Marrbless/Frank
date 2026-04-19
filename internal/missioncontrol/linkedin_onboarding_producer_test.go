package missioncontrol

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestResolveExecutionContextFrankLinkedInOnboardingBundleResolvesExactLinkedBundle(t *testing.T) {
	t.Parallel()

	fixtures := writeLinkedInOnboardingFixtures(t)
	ec := testExecutionContextWithFrankObjectRefs(t, []FrankRegistryObjectRef{
		{Kind: FrankRegistryObjectKindIdentity, ObjectID: fixtures.identity.IdentityID},
		{Kind: FrankRegistryObjectKindAccount, ObjectID: fixtures.account.AccountID},
	})
	ec.MissionStoreRoot = fixtures.root

	got, ok, err := ResolveExecutionContextFrankLinkedInOnboardingBundle(ec)
	if err != nil {
		t.Fatalf("ResolveExecutionContextFrankLinkedInOnboardingBundle() error = %v", err)
	}
	if !ok {
		t.Fatal("ResolveExecutionContextFrankLinkedInOnboardingBundle() ok = false, want true")
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

func TestResolveExecutionContextFrankLinkedInOnboardingBundleFailsClosedOnMissingAccount(t *testing.T) {
	t.Parallel()

	fixtures := writeLinkedInOnboardingFixtures(t)
	ec := testExecutionContextWithFrankObjectRefs(t, []FrankRegistryObjectRef{
		{Kind: FrankRegistryObjectKindIdentity, ObjectID: fixtures.identity.IdentityID},
	})
	ec.MissionStoreRoot = fixtures.root

	_, _, err := ResolveExecutionContextFrankLinkedInOnboardingBundle(ec)
	if err == nil {
		t.Fatal("ResolveExecutionContextFrankLinkedInOnboardingBundle() error = nil, want missing account rejection")
	}
	if !strings.Contains(err.Error(), "must resolve exactly one linkedin account, got 0") {
		t.Fatalf("ResolveExecutionContextFrankLinkedInOnboardingBundle() error = %q, want missing account rejection", err.Error())
	}
}

func TestProduceFrankLinkedInOnboardingPersistsConfirmedIdentifiersAndReplaysDeterministically(t *testing.T) {
	fixtures := writeLinkedInOnboardingFixtures(t)
	t.Setenv("PICOBOT_LINKEDIN_ACCESS_TOKEN", "linkedin-access-token")

	originalRead := readLinkedInUserInfo
	defer func() { readLinkedInUserInfo = originalRead }()
	readLinkedInUserInfo = func(ctx context.Context, accessToken string) (LinkedInUserInfoReadback, error) {
		if accessToken != "linkedin-access-token" {
			t.Fatalf("readLinkedInUserInfo() accessToken = %q, want %q", accessToken, "linkedin-access-token")
		}
		return LinkedInUserInfoReadback{
			LinkedInSub:   "linkedin-sub-123456",
			Name:          "Frank LinkedIn",
			PictureURL:    "https://example.test/frank-linkedin.png",
			Email:         "frank@linkedin.example",
			EmailVerified: boolPtr(true),
		}, nil
	}

	ec := testExecutionContextWithFrankObjectRefs(t, []FrankRegistryObjectRef{
		{Kind: FrankRegistryObjectKindIdentity, ObjectID: fixtures.identity.IdentityID},
		{Kind: FrankRegistryObjectKindAccount, ObjectID: fixtures.account.AccountID},
	})
	ec.MissionStoreRoot = fixtures.root
	bundle, ok, err := ResolveExecutionContextFrankLinkedInOnboardingBundle(ec)
	if err != nil {
		t.Fatalf("ResolveExecutionContextFrankLinkedInOnboardingBundle() error = %v", err)
	}
	if !ok {
		t.Fatal("ResolveExecutionContextFrankLinkedInOnboardingBundle() ok = false, want true")
	}

	firstNow := time.Date(2026, 4, 19, 3, 0, 0, 0, time.UTC)
	if err := ProduceFrankLinkedInOnboarding(fixtures.root, bundle, firstNow); err != nil {
		t.Fatalf("ProduceFrankLinkedInOnboarding(first) error = %v", err)
	}

	identity, err := LoadFrankIdentityRecord(fixtures.root, fixtures.identity.IdentityID)
	if err != nil {
		t.Fatalf("LoadFrankIdentityRecord() error = %v", err)
	}
	account, err := LoadFrankAccountRecord(fixtures.root, fixtures.account.AccountID)
	if err != nil {
		t.Fatalf("LoadFrankAccountRecord() error = %v", err)
	}
	if identity.LinkedIn == nil || identity.LinkedIn.LinkedInSub != "linkedin-sub-123456" || identity.LinkedIn.Name != "Frank LinkedIn" || identity.LinkedIn.PictureURL != "https://example.test/frank-linkedin.png" || identity.LinkedIn.Email != "frank@linkedin.example" {
		t.Fatalf("Identity.LinkedIn = %#v, want linkedin identity fields persisted", identity.LinkedIn)
	}
	if identity.LinkedIn.EmailVerified == nil || !*identity.LinkedIn.EmailVerified {
		t.Fatalf("Identity.LinkedIn.EmailVerified = %#v, want true", identity.LinkedIn.EmailVerified)
	}
	if account.LinkedIn == nil || account.LinkedIn.OAuthAccessTokenEnvVarRef != "PICOBOT_LINKEDIN_ACCESS_TOKEN" || !account.LinkedIn.ConfirmedAuthenticated {
		t.Fatalf("Account.LinkedIn = %#v, want access-token ref and confirmation persisted", account.LinkedIn)
	}
	firstIdentityUpdatedAt := identity.UpdatedAt
	firstAccountUpdatedAt := account.UpdatedAt

	bundle, ok, err = ResolveExecutionContextFrankLinkedInOnboardingBundle(ec)
	if err != nil {
		t.Fatalf("ResolveExecutionContextFrankLinkedInOnboardingBundle(replay) error = %v", err)
	}
	if !ok {
		t.Fatal("ResolveExecutionContextFrankLinkedInOnboardingBundle(replay) ok = false, want true")
	}

	if err := ProduceFrankLinkedInOnboarding(fixtures.root, bundle, firstNow.Add(time.Minute)); err != nil {
		t.Fatalf("ProduceFrankLinkedInOnboarding(replay) error = %v, want deterministic no-op success", err)
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

type linkedInOnboardingFixtures struct {
	root               string
	providerTarget     AutonomyEligibilityTargetRef
	accountClassTarget AutonomyEligibilityTargetRef
	identity           FrankIdentityRecord
	account            FrankAccountRecord
}

func writeLinkedInOnboardingFixtures(t *testing.T) linkedInOnboardingFixtures {
	t.Helper()

	root := t.TempDir()
	now := time.Date(2026, 4, 19, 0, 0, 0, 0, time.UTC)
	providerTarget := AutonomyEligibilityTargetRef{
		Kind:       EligibilityTargetKindProvider,
		RegistryID: "provider-linkedin",
	}
	accountClassTarget := AutonomyEligibilityTargetRef{
		Kind:       EligibilityTargetKindAccountClass,
		RegistryID: "account-class-linkedin",
	}

	writeFrankRegistryEligibilityFixture(t, root, providerTarget, EligibilityLabelAutonomyCompatible, "linkedin", "check-provider-linkedin", now)
	writeFrankRegistryEligibilityFixture(t, root, accountClassTarget, EligibilityLabelAutonomyCompatible, "linkedin account", "check-account-class-linkedin", now.Add(time.Minute))

	identity := FrankIdentityRecord{
		RecordVersion:        StoreRecordVersion,
		IdentityID:           "identity-linkedin",
		IdentityKind:         "platform_identity",
		DisplayName:          "Frank LinkedIn",
		ProviderOrPlatformID: providerTarget.RegistryID,
		LinkedIn:             &FrankLinkedInIdentity{},
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
		AccountID:            "account-linkedin",
		AccountKind:          "platform_account",
		Label:                "Frank LinkedIn Account",
		ProviderOrPlatformID: providerTarget.RegistryID,
		LinkedIn: &FrankLinkedInAccount{
			OAuthAccessTokenEnvVarRef: "PICOBOT_LINKEDIN_ACCESS_TOKEN",
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

	return linkedInOnboardingFixtures{
		root:               root,
		providerTarget:     providerTarget,
		accountClassTarget: accountClassTarget,
		identity:           identity,
		account:            account,
	}
}
