package tools

import (
	"strings"
	"testing"
	"time"

	"github.com/local/picobot/internal/missioncontrol"
)

func TestTaskStateActivateStepPayPalOnboardingInvokesHookWithResolvedBundle(t *testing.T) {
	t.Parallel()

	root, identity, account := writeTaskStatePayPalFixtures(t)
	job := testTaskStateJob()
	job.Plan.Steps[0].IdentityMode = missioncontrol.IdentityModeAgentAlias
	job.Plan.Steps[0].FrankObjectRefs = []missioncontrol.FrankRegistryObjectRef{
		{Kind: missioncontrol.FrankRegistryObjectKindIdentity, ObjectID: identity.IdentityID},
		{Kind: missioncontrol.FrankRegistryObjectKindAccount, ObjectID: account.AccountID},
	}

	state := NewTaskState()
	state.SetMissionStoreRoot(root)

	called := false
	state.payPalOnboardingHook = func(gotRoot string, bundle missioncontrol.ResolvedExecutionContextFrankPayPalOnboardingBundle, now time.Time) error {
		called = true
		if gotRoot != root {
			t.Fatalf("payPalOnboardingHook() root = %q, want %q", gotRoot, root)
		}
		if bundle.Identity.IdentityID != identity.IdentityID {
			t.Fatalf("payPalOnboardingHook() identity = %q, want %q", bundle.Identity.IdentityID, identity.IdentityID)
		}
		if bundle.Account.AccountID != account.AccountID {
			t.Fatalf("payPalOnboardingHook() account = %q, want %q", bundle.Account.AccountID, account.AccountID)
		}
		return nil
	}

	if err := state.ActivateStep(job, "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}
	if !called {
		t.Fatal("payPalOnboardingHook() not called, want invoked for resolved PayPal bundle")
	}
}

func TestTaskStateActivateStepPayPalOnboardingFailsClosedWithoutAccount(t *testing.T) {
	t.Parallel()

	root, identity, _ := writeTaskStatePayPalFixtures(t)
	job := testTaskStateJob()
	job.Plan.Steps[0].IdentityMode = missioncontrol.IdentityModeAgentAlias
	job.Plan.Steps[0].FrankObjectRefs = []missioncontrol.FrankRegistryObjectRef{
		{Kind: missioncontrol.FrankRegistryObjectKindIdentity, ObjectID: identity.IdentityID},
		{Kind: missioncontrol.FrankRegistryObjectKindAccount, ObjectID: "account-missing"},
	}

	state := NewTaskState()
	state.SetMissionStoreRoot(root)
	state.payPalOnboardingHook = func(root string, bundle missioncontrol.ResolvedExecutionContextFrankPayPalOnboardingBundle, now time.Time) error {
		t.Fatal("payPalOnboardingHook() called without exact linked account")
		return nil
	}

	err := state.ActivateStep(job, "build")
	if err == nil {
		t.Fatal("ActivateStep() error = nil, want missing paypal account rejection")
	}
	if !strings.Contains(err.Error(), `resolve Frank object ref kind "account" object_id "account-missing": mission store Frank account record not found`) {
		t.Fatalf("ActivateStep() error = %q, want missing account rejection", err.Error())
	}
}

func writeTaskStatePayPalFixtures(t *testing.T) (string, missioncontrol.FrankIdentityRecord, missioncontrol.FrankAccountRecord) {
	t.Helper()

	root := t.TempDir()
	now := time.Date(2026, 4, 18, 22, 0, 0, 0, time.UTC)
	providerTarget := missioncontrol.AutonomyEligibilityTargetRef{
		Kind:       missioncontrol.EligibilityTargetKindProvider,
		RegistryID: "provider-paypal",
	}
	accountClassTarget := missioncontrol.AutonomyEligibilityTargetRef{
		Kind:       missioncontrol.EligibilityTargetKindAccountClass,
		RegistryID: "account-class-paypal",
	}

	writeTaskStateAutonomyEligibilityFixture(t, root, providerTarget, missioncontrol.PlatformRecord{
		PlatformID:       providerTarget.RegistryID,
		PlatformName:     "paypal",
		TargetClass:      providerTarget.Kind,
		EligibilityLabel: missioncontrol.EligibilityLabelAutonomyCompatible,
		LastCheckID:      "check-provider-paypal",
		Notes:            []string{"registry note"},
		UpdatedAt:        now,
	}, missioncontrol.EligibilityCheckRecord{
		CheckID:                "check-provider-paypal",
		TargetKind:             providerTarget.Kind,
		TargetName:             "paypal",
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
		PlatformName:     "paypal account",
		TargetClass:      accountClassTarget.Kind,
		EligibilityLabel: missioncontrol.EligibilityLabelAutonomyCompatible,
		LastCheckID:      "check-account-class-paypal",
		Notes:            []string{"registry note"},
		UpdatedAt:        now.Add(time.Minute),
	}, missioncontrol.EligibilityCheckRecord{
		CheckID:                "check-account-class-paypal",
		TargetKind:             accountClassTarget.Kind,
		TargetName:             "paypal account",
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
		IdentityID:           "identity-paypal",
		IdentityKind:         "platform_identity",
		DisplayName:          "Frank PayPal",
		ProviderOrPlatformID: providerTarget.RegistryID,
		PayPal:               &missioncontrol.FrankPayPalIdentity{},
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
		AccountID:            "account-paypal",
		AccountKind:          "platform_account",
		Label:                "Frank PayPal Account",
		ProviderOrPlatformID: providerTarget.RegistryID,
		PayPal: &missioncontrol.FrankPayPalAccount{
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
	if err := missioncontrol.StoreFrankAccountRecord(root, account); err != nil {
		t.Fatalf("StoreFrankAccountRecord() error = %v", err)
	}

	return root, identity, account
}
