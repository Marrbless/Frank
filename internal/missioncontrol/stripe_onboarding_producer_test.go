package missioncontrol

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestResolveExecutionContextFrankStripeOnboardingBundleResolvesExactLinkedBundle(t *testing.T) {
	t.Parallel()

	fixtures := writeStripeOnboardingFixtures(t)
	ec := testExecutionContextWithFrankObjectRefs(t, []FrankRegistryObjectRef{
		{Kind: FrankRegistryObjectKindIdentity, ObjectID: fixtures.identity.IdentityID},
		{Kind: FrankRegistryObjectKindAccount, ObjectID: fixtures.account.AccountID},
	})
	ec.MissionStoreRoot = fixtures.root

	got, ok, err := ResolveExecutionContextFrankStripeOnboardingBundle(ec)
	if err != nil {
		t.Fatalf("ResolveExecutionContextFrankStripeOnboardingBundle() error = %v", err)
	}
	if !ok {
		t.Fatal("ResolveExecutionContextFrankStripeOnboardingBundle() ok = false, want true")
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

func TestResolveExecutionContextFrankStripeOnboardingBundleFailsClosedOnMissingAccount(t *testing.T) {
	t.Parallel()

	fixtures := writeStripeOnboardingFixtures(t)
	ec := testExecutionContextWithFrankObjectRefs(t, []FrankRegistryObjectRef{
		{Kind: FrankRegistryObjectKindIdentity, ObjectID: fixtures.identity.IdentityID},
	})
	ec.MissionStoreRoot = fixtures.root

	_, _, err := ResolveExecutionContextFrankStripeOnboardingBundle(ec)
	if err == nil {
		t.Fatal("ResolveExecutionContextFrankStripeOnboardingBundle() error = nil, want missing account rejection")
	}
	if !strings.Contains(err.Error(), "must resolve exactly one stripe account, got 0") {
		t.Fatalf("ResolveExecutionContextFrankStripeOnboardingBundle() error = %q, want missing account rejection", err.Error())
	}
}

func TestProduceFrankStripeOnboardingPersistsConfirmedIdentifiersAndReplaysDeterministically(t *testing.T) {
	fixtures := writeStripeOnboardingFixtures(t)
	t.Setenv("PICOBOT_STRIPE_SECRET_KEY", "sk_test_123")

	originalRead := readStripeAccount
	defer func() { readStripeAccount = originalRead }()
	readStripeAccount = func(ctx context.Context, apiKey string) (StripeReadbackAccount, error) {
		if apiKey != "sk_test_123" {
			t.Fatalf("readStripeAccount() apiKey = %q, want %q", apiKey, "sk_test_123")
		}
		return StripeReadbackAccount{
			StripeAccountID:            "acct_123456789",
			BusinessProfileName:        "Frank Labs",
			DashboardDisplayName:       "Frank",
			Country:                    "US",
			DefaultCurrency:            "usd",
			ChargesEnabled:             boolPtr(true),
			PayoutsEnabled:             boolPtr(false),
			DetailsSubmitted:           boolPtr(true),
			RequirementsDisabledReason: "requirements.past_due",
			Livemode:                   boolPtr(false),
		}, nil
	}

	ec := testExecutionContextWithFrankObjectRefs(t, []FrankRegistryObjectRef{
		{Kind: FrankRegistryObjectKindIdentity, ObjectID: fixtures.identity.IdentityID},
		{Kind: FrankRegistryObjectKindAccount, ObjectID: fixtures.account.AccountID},
	})
	ec.MissionStoreRoot = fixtures.root
	bundle, ok, err := ResolveExecutionContextFrankStripeOnboardingBundle(ec)
	if err != nil {
		t.Fatalf("ResolveExecutionContextFrankStripeOnboardingBundle() error = %v", err)
	}
	if !ok {
		t.Fatal("ResolveExecutionContextFrankStripeOnboardingBundle() ok = false, want true")
	}

	firstNow := time.Date(2026, 4, 18, 21, 0, 0, 0, time.UTC)
	if err := ProduceFrankStripeOnboarding(fixtures.root, bundle, firstNow); err != nil {
		t.Fatalf("ProduceFrankStripeOnboarding(first) error = %v", err)
	}

	identity, err := LoadFrankIdentityRecord(fixtures.root, fixtures.identity.IdentityID)
	if err != nil {
		t.Fatalf("LoadFrankIdentityRecord() error = %v", err)
	}
	account, err := LoadFrankAccountRecord(fixtures.root, fixtures.account.AccountID)
	if err != nil {
		t.Fatalf("LoadFrankAccountRecord() error = %v", err)
	}
	if identity.Stripe == nil || identity.Stripe.StripeAccountID != "acct_123456789" || identity.Stripe.BusinessProfileName != "Frank Labs" || identity.Stripe.DashboardDisplayName != "Frank" || identity.Stripe.Country != "US" || identity.Stripe.DefaultCurrency != "usd" {
		t.Fatalf("Identity.Stripe = %#v, want stripe identity fields persisted", identity.Stripe)
	}
	if account.Stripe == nil || account.Stripe.SecretKeyEnvVarRef != "PICOBOT_STRIPE_SECRET_KEY" || !account.Stripe.ConfirmedAuthenticated {
		t.Fatalf("Account.Stripe = %#v, want secret key ref and confirmation persisted", account.Stripe)
	}
	if account.Stripe.ChargesEnabled == nil || !*account.Stripe.ChargesEnabled {
		t.Fatalf("Account.Stripe.ChargesEnabled = %#v, want true", account.Stripe.ChargesEnabled)
	}
	if account.Stripe.PayoutsEnabled == nil || *account.Stripe.PayoutsEnabled {
		t.Fatalf("Account.Stripe.PayoutsEnabled = %#v, want false", account.Stripe.PayoutsEnabled)
	}
	if account.Stripe.DetailsSubmitted == nil || !*account.Stripe.DetailsSubmitted {
		t.Fatalf("Account.Stripe.DetailsSubmitted = %#v, want true", account.Stripe.DetailsSubmitted)
	}
	if account.Stripe.RequirementsDisabledReason != "requirements.past_due" {
		t.Fatalf("Account.Stripe.RequirementsDisabledReason = %q, want %q", account.Stripe.RequirementsDisabledReason, "requirements.past_due")
	}
	if account.Stripe.Livemode == nil || *account.Stripe.Livemode {
		t.Fatalf("Account.Stripe.Livemode = %#v, want false", account.Stripe.Livemode)
	}
	firstIdentityUpdatedAt := identity.UpdatedAt
	firstAccountUpdatedAt := account.UpdatedAt

	bundle, ok, err = ResolveExecutionContextFrankStripeOnboardingBundle(ec)
	if err != nil {
		t.Fatalf("ResolveExecutionContextFrankStripeOnboardingBundle(replay) error = %v", err)
	}
	if !ok {
		t.Fatal("ResolveExecutionContextFrankStripeOnboardingBundle(replay) ok = false, want true")
	}

	if err := ProduceFrankStripeOnboarding(fixtures.root, bundle, firstNow.Add(time.Minute)); err != nil {
		t.Fatalf("ProduceFrankStripeOnboarding(replay) error = %v, want deterministic no-op success", err)
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

type stripeOnboardingFixtures struct {
	root               string
	providerTarget     AutonomyEligibilityTargetRef
	accountClassTarget AutonomyEligibilityTargetRef
	identity           FrankIdentityRecord
	account            FrankAccountRecord
}

func writeStripeOnboardingFixtures(t *testing.T) stripeOnboardingFixtures {
	t.Helper()

	root := t.TempDir()
	now := time.Date(2026, 4, 18, 20, 0, 0, 0, time.UTC)
	providerTarget := AutonomyEligibilityTargetRef{
		Kind:       EligibilityTargetKindProvider,
		RegistryID: "provider-stripe",
	}
	accountClassTarget := AutonomyEligibilityTargetRef{
		Kind:       EligibilityTargetKindAccountClass,
		RegistryID: "account-class-stripe",
	}

	writeFrankRegistryEligibilityFixture(t, root, providerTarget, EligibilityLabelAutonomyCompatible, "stripe", "check-provider-stripe", now)
	writeFrankRegistryEligibilityFixture(t, root, accountClassTarget, EligibilityLabelAutonomyCompatible, "stripe account", "check-account-class-stripe", now.Add(time.Minute))

	identity := FrankIdentityRecord{
		RecordVersion:        StoreRecordVersion,
		IdentityID:           "identity-stripe",
		IdentityKind:         "platform_identity",
		DisplayName:          "Frank Stripe",
		ProviderOrPlatformID: providerTarget.RegistryID,
		Stripe:               &FrankStripeIdentity{},
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
		AccountID:            "account-stripe",
		AccountKind:          "platform_account",
		Label:                "Frank Stripe Account",
		ProviderOrPlatformID: providerTarget.RegistryID,
		Stripe: &FrankStripeAccount{
			SecretKeyEnvVarRef: "PICOBOT_STRIPE_SECRET_KEY",
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

	return stripeOnboardingFixtures{
		root:               root,
		providerTarget:     providerTarget,
		accountClassTarget: accountClassTarget,
		identity:           identity,
		account:            account,
	}
}

func boolPtr(value bool) *bool {
	return &value
}
