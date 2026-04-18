package tools

import (
	"strings"
	"testing"
	"time"

	"github.com/local/picobot/internal/missioncontrol"
)

func TestTaskStateActivateStepWhatsAppOwnerControlOnboardingInvokesHookWithResolvedBundle(t *testing.T) {
	t.Parallel()

	root, identity, account := writeTaskStateWhatsAppOwnerControlFixtures(t)
	job := testTaskStateJob()
	job.Plan.Steps[0].IdentityMode = missioncontrol.IdentityModeOwnerOnlyControl
	job.Plan.Steps[0].FrankObjectRefs = []missioncontrol.FrankRegistryObjectRef{
		{Kind: missioncontrol.FrankRegistryObjectKindIdentity, ObjectID: identity.IdentityID},
		{Kind: missioncontrol.FrankRegistryObjectKindAccount, ObjectID: account.AccountID},
	}

	state := NewTaskState()
	state.SetMissionStoreRoot(root)

	called := false
	state.whatsAppOwnerControlOnboardingHook = func(gotRoot string, bundle missioncontrol.ResolvedExecutionContextFrankWhatsAppOwnerControlOnboardingBundle, now time.Time) error {
		called = true
		if gotRoot != root {
			t.Fatalf("whatsAppOwnerControlOnboardingHook() root = %q, want %q", gotRoot, root)
		}
		if bundle.Identity.IdentityID != identity.IdentityID {
			t.Fatalf("whatsAppOwnerControlOnboardingHook() identity = %q, want %q", bundle.Identity.IdentityID, identity.IdentityID)
		}
		if bundle.Account.AccountID != account.AccountID {
			t.Fatalf("whatsAppOwnerControlOnboardingHook() account = %q, want %q", bundle.Account.AccountID, account.AccountID)
		}
		return nil
	}

	if err := state.ActivateStep(job, "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}
	if !called {
		t.Fatal("whatsAppOwnerControlOnboardingHook() not called, want invoked for resolved WhatsApp owner-control bundle")
	}
}

func TestTaskStateActivateStepWhatsAppOwnerControlOnboardingFailsClosedWithoutAccount(t *testing.T) {
	t.Parallel()

	root, identity, _ := writeTaskStateWhatsAppOwnerControlFixtures(t)
	job := testTaskStateJob()
	job.Plan.Steps[0].IdentityMode = missioncontrol.IdentityModeOwnerOnlyControl
	job.Plan.Steps[0].FrankObjectRefs = []missioncontrol.FrankRegistryObjectRef{
		{Kind: missioncontrol.FrankRegistryObjectKindIdentity, ObjectID: identity.IdentityID},
	}

	state := NewTaskState()
	state.SetMissionStoreRoot(root)
	state.whatsAppOwnerControlOnboardingHook = func(root string, bundle missioncontrol.ResolvedExecutionContextFrankWhatsAppOwnerControlOnboardingBundle, now time.Time) error {
		t.Fatal("whatsAppOwnerControlOnboardingHook() called without exact linked account")
		return nil
	}

	err := state.ActivateStep(job, "build")
	if err == nil {
		t.Fatal("ActivateStep() error = nil, want missing whatsapp owner-control account rejection")
	}
	if !strings.Contains(err.Error(), "must resolve exactly one whatsapp owner-control account, got 0") {
		t.Fatalf("ActivateStep() error = %q, want missing account rejection", err.Error())
	}
}

func writeTaskStateWhatsAppOwnerControlFixtures(t *testing.T) (string, missioncontrol.FrankIdentityRecord, missioncontrol.FrankAccountRecord) {
	t.Helper()

	root := t.TempDir()
	now := time.Date(2026, 4, 18, 18, 0, 0, 0, time.UTC)
	providerTarget := missioncontrol.AutonomyEligibilityTargetRef{
		Kind:       missioncontrol.EligibilityTargetKindProvider,
		RegistryID: "provider-whatsapp-owner-control",
	}
	accountClassTarget := missioncontrol.AutonomyEligibilityTargetRef{
		Kind:       missioncontrol.EligibilityTargetKindAccountClass,
		RegistryID: "account-class-whatsapp-owner-control",
	}

	writeTaskStateAutonomyEligibilityFixture(t, root, providerTarget, missioncontrol.PlatformRecord{
		PlatformID:       providerTarget.RegistryID,
		PlatformName:     "whatsapp owner-control",
		TargetClass:      providerTarget.Kind,
		EligibilityLabel: missioncontrol.EligibilityLabelAutonomyCompatible,
		LastCheckID:      "check-provider-whatsapp-owner-control",
		Notes:            []string{"registry note"},
		UpdatedAt:        now,
	}, missioncontrol.EligibilityCheckRecord{
		CheckID:                "check-provider-whatsapp-owner-control",
		TargetKind:             providerTarget.Kind,
		TargetName:             "whatsapp owner-control",
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
		PlatformName:     "whatsapp owner-control account",
		TargetClass:      accountClassTarget.Kind,
		EligibilityLabel: missioncontrol.EligibilityLabelAutonomyCompatible,
		LastCheckID:      "check-account-class-whatsapp-owner-control",
		Notes:            []string{"registry note"},
		UpdatedAt:        now.Add(time.Minute),
	}, missioncontrol.EligibilityCheckRecord{
		CheckID:                "check-account-class-whatsapp-owner-control",
		TargetKind:             accountClassTarget.Kind,
		TargetName:             "whatsapp owner-control account",
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
		IdentityID:           "identity-whatsapp-owner-control",
		IdentityKind:         "owner_control_channel",
		DisplayName:          "WhatsApp Owner Control",
		ProviderOrPlatformID: providerTarget.RegistryID,
		WhatsAppOwnerControl: &missioncontrol.FrankWhatsAppOwnerControlIdentity{},
		IdentityMode:         missioncontrol.IdentityModeOwnerOnlyControl,
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
		AccountID:            "account-whatsapp-owner-control",
		AccountKind:          "owner_control_channel",
		Label:                "WhatsApp Owner Control",
		ProviderOrPlatformID: providerTarget.RegistryID,
		WhatsAppOwnerControl: &missioncontrol.FrankWhatsAppOwnerControlAccount{},
		IdentityID:           identity.IdentityID,
		ControlModel:         "owner_controlled",
		RecoveryModel:        "owner_recoverable",
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
