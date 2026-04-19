package tools

import (
	"strings"
	"testing"
	"time"

	"github.com/local/picobot/internal/missioncontrol"
)

func TestTaskStateActivateStepStripeOnboardingInvokesHookWithResolvedBundle(t *testing.T) {
	t.Parallel()

	root, identity, account := writeTaskStateStripeFixtures(t)
	job := testTaskStateJob()
	job.Plan.Steps[0].IdentityMode = missioncontrol.IdentityModeAgentAlias
	job.Plan.Steps[0].FrankObjectRefs = []missioncontrol.FrankRegistryObjectRef{
		{Kind: missioncontrol.FrankRegistryObjectKindIdentity, ObjectID: identity.IdentityID},
		{Kind: missioncontrol.FrankRegistryObjectKindAccount, ObjectID: account.AccountID},
	}

	state := NewTaskState()
	state.SetMissionStoreRoot(root)

	called := false
	state.stripeOnboardingHook = func(gotRoot string, bundle missioncontrol.ResolvedExecutionContextFrankStripeOnboardingBundle, now time.Time) error {
		called = true
		if gotRoot != root {
			t.Fatalf("stripeOnboardingHook() root = %q, want %q", gotRoot, root)
		}
		if bundle.Identity.IdentityID != identity.IdentityID {
			t.Fatalf("stripeOnboardingHook() identity = %q, want %q", bundle.Identity.IdentityID, identity.IdentityID)
		}
		if bundle.Account.AccountID != account.AccountID {
			t.Fatalf("stripeOnboardingHook() account = %q, want %q", bundle.Account.AccountID, account.AccountID)
		}
		return nil
	}

	if err := state.ActivateStep(job, "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}
	if !called {
		t.Fatal("stripeOnboardingHook() not called, want invoked for resolved Stripe bundle")
	}
}

func TestTaskStateActivateStepStripeOnboardingFailsClosedWithoutAccount(t *testing.T) {
	t.Parallel()

	root, identity, _ := writeTaskStateStripeFixtures(t)
	job := testTaskStateJob()
	job.Plan.Steps[0].IdentityMode = missioncontrol.IdentityModeAgentAlias
	job.Plan.Steps[0].FrankObjectRefs = []missioncontrol.FrankRegistryObjectRef{
		{Kind: missioncontrol.FrankRegistryObjectKindIdentity, ObjectID: identity.IdentityID},
		{Kind: missioncontrol.FrankRegistryObjectKindAccount, ObjectID: "account-missing"},
	}

	state := NewTaskState()
	state.SetMissionStoreRoot(root)
	state.stripeOnboardingHook = func(root string, bundle missioncontrol.ResolvedExecutionContextFrankStripeOnboardingBundle, now time.Time) error {
		t.Fatal("stripeOnboardingHook() called without exact linked account")
		return nil
	}

	err := state.ActivateStep(job, "build")
	if err == nil {
		t.Fatal("ActivateStep() error = nil, want missing stripe account rejection")
	}
	if !strings.Contains(err.Error(), `resolve Frank object ref kind "account" object_id "account-missing": mission store Frank account record not found`) {
		t.Fatalf("ActivateStep() error = %q, want missing account rejection", err.Error())
	}
}

func writeTaskStateStripeFixtures(t *testing.T) (string, missioncontrol.FrankIdentityRecord, missioncontrol.FrankAccountRecord) {
	t.Helper()

	root := t.TempDir()
	now := time.Date(2026, 4, 18, 20, 0, 0, 0, time.UTC)
	providerTarget := missioncontrol.AutonomyEligibilityTargetRef{
		Kind:       missioncontrol.EligibilityTargetKindProvider,
		RegistryID: "provider-stripe",
	}
	accountClassTarget := missioncontrol.AutonomyEligibilityTargetRef{
		Kind:       missioncontrol.EligibilityTargetKindAccountClass,
		RegistryID: "account-class-stripe",
	}

	writeTaskStateAutonomyEligibilityFixture(t, root, providerTarget, missioncontrol.PlatformRecord{
		PlatformID:       providerTarget.RegistryID,
		PlatformName:     "stripe",
		TargetClass:      providerTarget.Kind,
		EligibilityLabel: missioncontrol.EligibilityLabelAutonomyCompatible,
		LastCheckID:      "check-provider-stripe",
		Notes:            []string{"registry note"},
		UpdatedAt:        now,
	}, missioncontrol.EligibilityCheckRecord{
		CheckID:                "check-provider-stripe",
		TargetKind:             providerTarget.Kind,
		TargetName:             "stripe",
		CanCreateWithoutOwner:  true,
		CanOnboardWithoutOwner: true,
		CanControlAsAgent:      true,
		CanRecoverAsAgent:      true,
		RulesAsObservedOK:      true,
		Label:                  missioncontrol.EligibilityLabelAutonomyCompatible,
		Reasons:                []string{"autonomy-compatible"},
		CheckedAt:              now,
	})
	writeTaskStateAutonomyEligibilityFixture(t, root, accountClassTarget, missioncontrol.PlatformRecord{
		PlatformID:       accountClassTarget.RegistryID,
		PlatformName:     "stripe account",
		TargetClass:      accountClassTarget.Kind,
		EligibilityLabel: missioncontrol.EligibilityLabelAutonomyCompatible,
		LastCheckID:      "check-account-class-stripe",
		Notes:            []string{"registry note"},
		UpdatedAt:        now.Add(time.Minute),
	}, missioncontrol.EligibilityCheckRecord{
		CheckID:                "check-account-class-stripe",
		TargetKind:             accountClassTarget.Kind,
		TargetName:             "stripe account",
		CanCreateWithoutOwner:  true,
		CanOnboardWithoutOwner: true,
		CanControlAsAgent:      true,
		CanRecoverAsAgent:      true,
		RulesAsObservedOK:      true,
		Label:                  missioncontrol.EligibilityLabelAutonomyCompatible,
		Reasons:                []string{"autonomy-compatible"},
		CheckedAt:              now.Add(time.Minute),
	})

	identity := missioncontrol.FrankIdentityRecord{
		RecordVersion:        missioncontrol.StoreRecordVersion,
		IdentityID:           "identity-stripe",
		IdentityKind:         "platform_identity",
		DisplayName:          "Frank Stripe",
		ProviderOrPlatformID: providerTarget.RegistryID,
		Stripe:               &missioncontrol.FrankStripeIdentity{},
		IdentityMode:         missioncontrol.IdentityModeAgentAlias,
		State:                "candidate",
		EligibilityTargetRef: providerTarget,
		CreatedAt:            now,
		UpdatedAt:            now.Add(time.Minute),
	}
	if err := missioncontrol.StoreFrankIdentityRecord(root, identity); err != nil {
		t.Fatalf("StoreFrankIdentityRecord() error = %v", err)
	}

	account := missioncontrol.FrankAccountRecord{
		RecordVersion:        missioncontrol.StoreRecordVersion,
		AccountID:            "account-stripe",
		AccountKind:          "platform_account",
		Label:                "Frank Stripe Account",
		ProviderOrPlatformID: providerTarget.RegistryID,
		Stripe: &missioncontrol.FrankStripeAccount{
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
	if err := missioncontrol.StoreFrankAccountRecord(root, account); err != nil {
		t.Fatalf("StoreFrankAccountRecord() error = %v", err)
	}

	return root, identity, account
}
