package missioncontrol

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestResolveExecutionContextFrankMicrosoftOnboardingBundleResolvesExactLinkedBundle(t *testing.T) {
	t.Parallel()

	fixtures := writeMicrosoftOnboardingFixtures(t)
	ec := testExecutionContextWithFrankObjectRefs(t, []FrankRegistryObjectRef{
		{Kind: FrankRegistryObjectKindIdentity, ObjectID: fixtures.identity.IdentityID},
		{Kind: FrankRegistryObjectKindAccount, ObjectID: fixtures.account.AccountID},
	})
	ec.MissionStoreRoot = fixtures.root

	got, ok, err := ResolveExecutionContextFrankMicrosoftOnboardingBundle(ec)
	if err != nil {
		t.Fatalf("ResolveExecutionContextFrankMicrosoftOnboardingBundle() error = %v", err)
	}
	if !ok {
		t.Fatal("ResolveExecutionContextFrankMicrosoftOnboardingBundle() ok = false, want true")
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

func TestResolveExecutionContextFrankMicrosoftOnboardingBundleFailsClosedOnMissingAccount(t *testing.T) {
	t.Parallel()

	fixtures := writeMicrosoftOnboardingFixtures(t)
	ec := testExecutionContextWithFrankObjectRefs(t, []FrankRegistryObjectRef{
		{Kind: FrankRegistryObjectKindIdentity, ObjectID: fixtures.identity.IdentityID},
	})
	ec.MissionStoreRoot = fixtures.root

	_, _, err := ResolveExecutionContextFrankMicrosoftOnboardingBundle(ec)
	if err == nil {
		t.Fatal("ResolveExecutionContextFrankMicrosoftOnboardingBundle() error = nil, want missing account rejection")
	}
	if !strings.Contains(err.Error(), "must resolve exactly one microsoft account, got 0") {
		t.Fatalf("ResolveExecutionContextFrankMicrosoftOnboardingBundle() error = %q, want missing account rejection", err.Error())
	}
}

func TestProduceFrankMicrosoftOnboardingPersistsConfirmedIdentifiersAndReplaysDeterministically(t *testing.T) {
	fixtures := writeMicrosoftOnboardingFixtures(t)
	t.Setenv("PICOBOT_MICROSOFT_ACCESS_TOKEN", "microsoft-access-token")

	originalRead := readMicrosoftUserInfo
	defer func() { readMicrosoftUserInfo = originalRead }()
	readMicrosoftUserInfo = func(ctx context.Context, accessToken string) (MicrosoftUserInfoReadback, error) {
		if accessToken != "microsoft-access-token" {
			t.Fatalf("readMicrosoftUserInfo() accessToken = %q, want %q", accessToken, "microsoft-access-token")
		}
		return MicrosoftUserInfoReadback{
			MicrosoftSub:      "microsoft-sub-123456",
			MicrosoftOID:      "microsoft-oid-7890",
			TenantID:          "tenant-4567",
			Email:             "frank@microsoft.example",
			PreferredUsername: "frank@microsoft.example",
			EmailVerified:     boolPtr(true),
			DisplayName:       "Frank Microsoft",
		}, nil
	}

	ec := testExecutionContextWithFrankObjectRefs(t, []FrankRegistryObjectRef{
		{Kind: FrankRegistryObjectKindIdentity, ObjectID: fixtures.identity.IdentityID},
		{Kind: FrankRegistryObjectKindAccount, ObjectID: fixtures.account.AccountID},
	})
	ec.MissionStoreRoot = fixtures.root
	bundle, ok, err := ResolveExecutionContextFrankMicrosoftOnboardingBundle(ec)
	if err != nil {
		t.Fatalf("ResolveExecutionContextFrankMicrosoftOnboardingBundle() error = %v", err)
	}
	if !ok {
		t.Fatal("ResolveExecutionContextFrankMicrosoftOnboardingBundle() ok = false, want true")
	}

	firstNow := time.Date(2026, 4, 19, 4, 0, 0, 0, time.UTC)
	if err := ProduceFrankMicrosoftOnboarding(fixtures.root, bundle, firstNow); err != nil {
		t.Fatalf("ProduceFrankMicrosoftOnboarding(first) error = %v", err)
	}

	identity, err := LoadFrankIdentityRecord(fixtures.root, fixtures.identity.IdentityID)
	if err != nil {
		t.Fatalf("LoadFrankIdentityRecord() error = %v", err)
	}
	account, err := LoadFrankAccountRecord(fixtures.root, fixtures.account.AccountID)
	if err != nil {
		t.Fatalf("LoadFrankAccountRecord() error = %v", err)
	}
	if identity.Microsoft == nil ||
		identity.Microsoft.MicrosoftSub != "microsoft-sub-123456" ||
		identity.Microsoft.MicrosoftOID != "microsoft-oid-7890" ||
		identity.Microsoft.TenantID != "tenant-4567" ||
		identity.Microsoft.Email != "frank@microsoft.example" ||
		identity.Microsoft.PreferredUsername != "frank@microsoft.example" ||
		identity.Microsoft.DisplayName != "Frank Microsoft" {
		t.Fatalf("Identity.Microsoft = %#v, want microsoft identity fields persisted", identity.Microsoft)
	}
	if identity.Microsoft.EmailVerified == nil || !*identity.Microsoft.EmailVerified {
		t.Fatalf("Identity.Microsoft.EmailVerified = %#v, want true", identity.Microsoft.EmailVerified)
	}
	if account.Microsoft == nil || account.Microsoft.OAuthAccessTokenEnvVarRef != "PICOBOT_MICROSOFT_ACCESS_TOKEN" || !account.Microsoft.ConfirmedAuthenticated {
		t.Fatalf("Account.Microsoft = %#v, want access-token ref and confirmation persisted", account.Microsoft)
	}
	firstIdentityUpdatedAt := identity.UpdatedAt
	firstAccountUpdatedAt := account.UpdatedAt

	bundle, ok, err = ResolveExecutionContextFrankMicrosoftOnboardingBundle(ec)
	if err != nil {
		t.Fatalf("ResolveExecutionContextFrankMicrosoftOnboardingBundle(replay) error = %v", err)
	}
	if !ok {
		t.Fatal("ResolveExecutionContextFrankMicrosoftOnboardingBundle(replay) ok = false, want true")
	}

	if err := ProduceFrankMicrosoftOnboarding(fixtures.root, bundle, firstNow.Add(time.Minute)); err != nil {
		t.Fatalf("ProduceFrankMicrosoftOnboarding(replay) error = %v, want deterministic no-op success", err)
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

func TestProduceFrankMicrosoftOnboardingFailsClosedOnConflictingStableIdentifier(t *testing.T) {
	fixtures := writeMicrosoftOnboardingFixtures(t)
	t.Setenv("PICOBOT_MICROSOFT_ACCESS_TOKEN", "microsoft-access-token")

	identity, err := LoadFrankIdentityRecord(fixtures.root, fixtures.identity.IdentityID)
	if err != nil {
		t.Fatalf("LoadFrankIdentityRecord() error = %v", err)
	}
	identity.Microsoft = &FrankMicrosoftIdentity{MicrosoftSub: "committed-sub"}
	if err := StoreFrankIdentityRecord(fixtures.root, identity); err != nil {
		t.Fatalf("StoreFrankIdentityRecord() error = %v", err)
	}

	originalRead := readMicrosoftUserInfo
	defer func() { readMicrosoftUserInfo = originalRead }()
	readMicrosoftUserInfo = func(ctx context.Context, accessToken string) (MicrosoftUserInfoReadback, error) {
		return MicrosoftUserInfoReadback{MicrosoftSub: "provider-sub"}, nil
	}

	ec := testExecutionContextWithFrankObjectRefs(t, []FrankRegistryObjectRef{
		{Kind: FrankRegistryObjectKindIdentity, ObjectID: fixtures.identity.IdentityID},
		{Kind: FrankRegistryObjectKindAccount, ObjectID: fixtures.account.AccountID},
	})
	ec.MissionStoreRoot = fixtures.root
	bundle, ok, err := ResolveExecutionContextFrankMicrosoftOnboardingBundle(ec)
	if err != nil {
		t.Fatalf("ResolveExecutionContextFrankMicrosoftOnboardingBundle() error = %v", err)
	}
	if !ok {
		t.Fatal("ResolveExecutionContextFrankMicrosoftOnboardingBundle() ok = false, want true")
	}

	err = ProduceFrankMicrosoftOnboarding(fixtures.root, bundle, time.Date(2026, 4, 19, 4, 30, 0, 0, time.UTC))
	if err == nil {
		t.Fatal("ProduceFrankMicrosoftOnboarding() error = nil, want conflict rejection")
	}
	if !strings.Contains(err.Error(), `conflicts with provider microsoft_sub "provider-sub"`) {
		t.Fatalf("ProduceFrankMicrosoftOnboarding() error = %q, want microsoft_sub conflict", err.Error())
	}
}

type microsoftOnboardingFixtures struct {
	root               string
	providerTarget     AutonomyEligibilityTargetRef
	accountClassTarget AutonomyEligibilityTargetRef
	identity           FrankIdentityRecord
	account            FrankAccountRecord
}

func writeMicrosoftOnboardingFixtures(t *testing.T) microsoftOnboardingFixtures {
	t.Helper()

	root := t.TempDir()
	now := time.Date(2026, 4, 19, 1, 0, 0, 0, time.UTC)
	providerTarget := AutonomyEligibilityTargetRef{
		Kind:       EligibilityTargetKindProvider,
		RegistryID: "provider-microsoft",
	}
	accountClassTarget := AutonomyEligibilityTargetRef{
		Kind:       EligibilityTargetKindAccountClass,
		RegistryID: "account-class-microsoft",
	}

	writeFrankRegistryEligibilityFixture(t, root, providerTarget, EligibilityLabelAutonomyCompatible, "microsoft", "check-provider-microsoft", now)
	writeFrankRegistryEligibilityFixture(t, root, accountClassTarget, EligibilityLabelAutonomyCompatible, "microsoft account", "check-account-class-microsoft", now.Add(time.Minute))

	identity := FrankIdentityRecord{
		RecordVersion:        StoreRecordVersion,
		IdentityID:           "identity-microsoft",
		IdentityKind:         "platform_identity",
		DisplayName:          "Frank Microsoft",
		ProviderOrPlatformID: providerTarget.RegistryID,
		Microsoft:            &FrankMicrosoftIdentity{},
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
		AccountID:            "account-microsoft",
		AccountKind:          "platform_account",
		Label:                "Frank Microsoft Account",
		ProviderOrPlatformID: providerTarget.RegistryID,
		Microsoft: &FrankMicrosoftAccount{
			OAuthAccessTokenEnvVarRef: "PICOBOT_MICROSOFT_ACCESS_TOKEN",
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

	return microsoftOnboardingFixtures{
		root:               root,
		providerTarget:     providerTarget,
		accountClassTarget: accountClassTarget,
		identity:           identity,
		account:            account,
	}
}
