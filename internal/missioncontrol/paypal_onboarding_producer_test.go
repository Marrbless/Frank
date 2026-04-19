package missioncontrol

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestResolveExecutionContextFrankPayPalOnboardingBundleResolvesExactLinkedBundle(t *testing.T) {
	t.Parallel()

	fixtures := writePayPalOnboardingFixtures(t)
	ec := testExecutionContextWithFrankObjectRefs(t, []FrankRegistryObjectRef{
		{Kind: FrankRegistryObjectKindIdentity, ObjectID: fixtures.identity.IdentityID},
		{Kind: FrankRegistryObjectKindAccount, ObjectID: fixtures.account.AccountID},
	})
	ec.MissionStoreRoot = fixtures.root

	got, ok, err := ResolveExecutionContextFrankPayPalOnboardingBundle(ec)
	if err != nil {
		t.Fatalf("ResolveExecutionContextFrankPayPalOnboardingBundle() error = %v", err)
	}
	if !ok {
		t.Fatal("ResolveExecutionContextFrankPayPalOnboardingBundle() ok = false, want true")
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

func TestResolveExecutionContextFrankPayPalOnboardingBundleFailsClosedOnMissingAccount(t *testing.T) {
	t.Parallel()

	fixtures := writePayPalOnboardingFixtures(t)
	ec := testExecutionContextWithFrankObjectRefs(t, []FrankRegistryObjectRef{
		{Kind: FrankRegistryObjectKindIdentity, ObjectID: fixtures.identity.IdentityID},
	})
	ec.MissionStoreRoot = fixtures.root

	_, _, err := ResolveExecutionContextFrankPayPalOnboardingBundle(ec)
	if err == nil {
		t.Fatal("ResolveExecutionContextFrankPayPalOnboardingBundle() error = nil, want missing account rejection")
	}
	if !strings.Contains(err.Error(), "must resolve exactly one paypal account, got 0") {
		t.Fatalf("ResolveExecutionContextFrankPayPalOnboardingBundle() error = %q, want missing account rejection", err.Error())
	}
}

func TestProduceFrankPayPalOnboardingPersistsConfirmedIdentifiersAndReplaysDeterministically(t *testing.T) {
	fixtures := writePayPalOnboardingFixtures(t)
	t.Setenv("PICOBOT_PAYPAL_CLIENT_ID", "paypal-client-id")
	t.Setenv("PICOBOT_PAYPAL_CLIENT_SECRET", "paypal-client-secret")

	originalReadAccessToken := readPayPalAccessToken
	originalReadUserInfo := readPayPalUserInfo
	defer func() {
		readPayPalAccessToken = originalReadAccessToken
		readPayPalUserInfo = originalReadUserInfo
	}()
	readPayPalAccessToken = func(ctx context.Context, environment, clientID, clientSecret string) (string, string, error) {
		if environment != "sandbox" {
			t.Fatalf("readPayPalAccessToken() environment = %q, want %q", environment, "sandbox")
		}
		if clientID != "paypal-client-id" {
			t.Fatalf("readPayPalAccessToken() clientID = %q, want %q", clientID, "paypal-client-id")
		}
		if clientSecret != "paypal-client-secret" {
			t.Fatalf("readPayPalAccessToken() clientSecret = %q, want %q", clientSecret, "paypal-client-secret")
		}
		return "paypal-access-token", "", nil
	}
	readPayPalUserInfo = func(ctx context.Context, environment, accessToken string) (PayPalUserInfoReadback, error) {
		if environment != "sandbox" {
			t.Fatalf("readPayPalUserInfo() environment = %q, want %q", environment, "sandbox")
		}
		if accessToken != "paypal-access-token" {
			t.Fatalf("readPayPalUserInfo() accessToken = %q, want %q", accessToken, "paypal-access-token")
		}
		return PayPalUserInfoReadback{
			PayPalUserID:    "https://www.paypal.com/webapps/auth/identity/user/123456",
			Sub:             "sub-123456",
			Email:           "frank@paypal.example",
			EmailVerified:   paypalBoolPtr(true),
			VerifiedAccount: paypalBoolPtr(true),
			AccountType:     "BUSINESS",
		}, nil
	}

	ec := testExecutionContextWithFrankObjectRefs(t, []FrankRegistryObjectRef{
		{Kind: FrankRegistryObjectKindIdentity, ObjectID: fixtures.identity.IdentityID},
		{Kind: FrankRegistryObjectKindAccount, ObjectID: fixtures.account.AccountID},
	})
	ec.MissionStoreRoot = fixtures.root
	bundle, ok, err := ResolveExecutionContextFrankPayPalOnboardingBundle(ec)
	if err != nil {
		t.Fatalf("ResolveExecutionContextFrankPayPalOnboardingBundle() error = %v", err)
	}
	if !ok {
		t.Fatal("ResolveExecutionContextFrankPayPalOnboardingBundle() ok = false, want true")
	}

	firstNow := time.Date(2026, 4, 19, 0, 0, 0, 0, time.UTC)
	if err := ProduceFrankPayPalOnboarding(fixtures.root, bundle, firstNow); err != nil {
		t.Fatalf("ProduceFrankPayPalOnboarding(first) error = %v", err)
	}

	identity, err := LoadFrankIdentityRecord(fixtures.root, fixtures.identity.IdentityID)
	if err != nil {
		t.Fatalf("LoadFrankIdentityRecord() error = %v", err)
	}
	account, err := LoadFrankAccountRecord(fixtures.root, fixtures.account.AccountID)
	if err != nil {
		t.Fatalf("LoadFrankAccountRecord() error = %v", err)
	}
	if identity.PayPal == nil || identity.PayPal.PayPalUserID != "https://www.paypal.com/webapps/auth/identity/user/123456" || identity.PayPal.Sub != "sub-123456" || identity.PayPal.Email != "frank@paypal.example" || identity.PayPal.AccountType != "BUSINESS" {
		t.Fatalf("Identity.PayPal = %#v, want paypal identity fields persisted", identity.PayPal)
	}
	if identity.PayPal.EmailVerified == nil || !*identity.PayPal.EmailVerified {
		t.Fatalf("Identity.PayPal.EmailVerified = %#v, want true", identity.PayPal.EmailVerified)
	}
	if identity.PayPal.VerifiedAccount == nil || !*identity.PayPal.VerifiedAccount {
		t.Fatalf("Identity.PayPal.VerifiedAccount = %#v, want true", identity.PayPal.VerifiedAccount)
	}
	if account.PayPal == nil || account.PayPal.ClientIDEnvVarRef != "PICOBOT_PAYPAL_CLIENT_ID" || account.PayPal.ClientSecretEnvVarRef != "PICOBOT_PAYPAL_CLIENT_SECRET" || account.PayPal.Environment != "sandbox" || !account.PayPal.ConfirmedAuthenticated {
		t.Fatalf("Account.PayPal = %#v, want env refs, environment, and confirmation persisted", account.PayPal)
	}
	firstIdentityUpdatedAt := identity.UpdatedAt
	firstAccountUpdatedAt := account.UpdatedAt

	bundle, ok, err = ResolveExecutionContextFrankPayPalOnboardingBundle(ec)
	if err != nil {
		t.Fatalf("ResolveExecutionContextFrankPayPalOnboardingBundle(replay) error = %v", err)
	}
	if !ok {
		t.Fatal("ResolveExecutionContextFrankPayPalOnboardingBundle(replay) ok = false, want true")
	}

	if err := ProduceFrankPayPalOnboarding(fixtures.root, bundle, firstNow.Add(time.Minute)); err != nil {
		t.Fatalf("ProduceFrankPayPalOnboarding(replay) error = %v, want deterministic no-op success", err)
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

type payPalOnboardingFixtures struct {
	root               string
	providerTarget     AutonomyEligibilityTargetRef
	accountClassTarget AutonomyEligibilityTargetRef
	identity           FrankIdentityRecord
	account            FrankAccountRecord
}

func writePayPalOnboardingFixtures(t *testing.T) payPalOnboardingFixtures {
	t.Helper()

	root := t.TempDir()
	now := time.Date(2026, 4, 18, 22, 0, 0, 0, time.UTC)
	providerTarget := AutonomyEligibilityTargetRef{
		Kind:       EligibilityTargetKindProvider,
		RegistryID: "provider-paypal",
	}
	accountClassTarget := AutonomyEligibilityTargetRef{
		Kind:       EligibilityTargetKindAccountClass,
		RegistryID: "account-class-paypal",
	}

	writeFrankRegistryEligibilityFixture(t, root, providerTarget, EligibilityLabelAutonomyCompatible, "paypal", "check-provider-paypal", now)
	writeFrankRegistryEligibilityFixture(t, root, accountClassTarget, EligibilityLabelAutonomyCompatible, "paypal account", "check-account-class-paypal", now.Add(time.Minute))

	identity := FrankIdentityRecord{
		RecordVersion:        StoreRecordVersion,
		IdentityID:           "identity-paypal",
		IdentityKind:         "platform_identity",
		DisplayName:          "Frank PayPal",
		ProviderOrPlatformID: providerTarget.RegistryID,
		PayPal:               &FrankPayPalIdentity{},
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
		AccountID:            "account-paypal",
		AccountKind:          "platform_account",
		Label:                "Frank PayPal Account",
		ProviderOrPlatformID: providerTarget.RegistryID,
		PayPal: &FrankPayPalAccount{
			ClientIDEnvVarRef:     "PICOBOT_PAYPAL_CLIENT_ID",
			ClientSecretEnvVarRef: "PICOBOT_PAYPAL_CLIENT_SECRET",
			Environment:           "sandbox",
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

	return payPalOnboardingFixtures{
		root:               root,
		providerTarget:     providerTarget,
		accountClassTarget: accountClassTarget,
		identity:           identity,
		account:            account,
	}
}

func paypalBoolPtr(value bool) *bool {
	return &value
}
