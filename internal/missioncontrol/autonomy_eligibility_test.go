package missioncontrol

import (
	"errors"
	"reflect"
	"testing"
	"time"
)

func TestRequireAutonomyEligibleTargetEligibleProviderPasses(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 6, 12, 0, 0, 0, time.UTC)
	target := AutonomyEligibilityTargetRef{
		Kind:       EligibilityTargetKindProvider,
		RegistryID: "provider-mail",
	}
	writeAutonomyEligibilityFixture(t, root, target, PlatformRecord{
		PlatformID:       target.RegistryID,
		PlatformName:     "mail.example",
		TargetClass:      target.Kind,
		EligibilityLabel: EligibilityLabelAutonomyCompatible,
		LastCheckID:      "check-provider-mail",
		Notes:            []string{"registry note"},
		UpdatedAt:        now,
	}, EligibilityCheckRecord{
		CheckID:                     "check-provider-mail",
		TargetKind:                  target.Kind,
		TargetName:                  "mail.example",
		CanCreateWithoutOwner:       true,
		CanOnboardWithoutOwner:      true,
		CanControlAsAgent:           true,
		CanRecoverAsAgent:           true,
		RequiresHumanOnlyStep:       false,
		RequiresOwnerOnlySecretOrID: false,
		RulesAsObservedOK:           true,
		Label:                       EligibilityLabelAutonomyCompatible,
		Reasons:                     []string{"operator-reviewed"},
		CheckedAt:                   now,
	})

	result, err := RequireAutonomyEligibleTarget(root, target)
	if err != nil {
		t.Fatalf("RequireAutonomyEligibleTarget() error = %v", err)
	}
	if result.Decision != AutonomyEligibilityDecisionEligible {
		t.Fatalf("Decision = %q, want %q", result.Decision, AutonomyEligibilityDecisionEligible)
	}
	if result.CheckID != "check-provider-mail" {
		t.Fatalf("CheckID = %q, want %q", result.CheckID, "check-provider-mail")
	}
	if len(result.Reasons) != 0 {
		t.Fatalf("Reasons = %#v, want empty for eligible target", result.Reasons)
	}
}

func TestRequireAutonomyEligibleTargetIneligibleProviderFailsWithCanonicalReason(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 6, 12, 30, 0, 0, time.UTC)
	target := AutonomyEligibilityTargetRef{
		Kind:       EligibilityTargetKindProvider,
		RegistryID: "provider-human-id",
	}
	writeAutonomyEligibilityFixture(t, root, target, PlatformRecord{
		PlatformID:       target.RegistryID,
		PlatformName:     "human-id.example",
		TargetClass:      target.Kind,
		EligibilityLabel: EligibilityLabelIneligible,
		LastCheckID:      "check-provider-human-id",
		Notes:            []string{"registry note"},
		UpdatedAt:        now,
	}, EligibilityCheckRecord{
		CheckID:                     "check-provider-human-id",
		TargetKind:                  target.Kind,
		TargetName:                  "human-id.example",
		CanCreateWithoutOwner:       false,
		CanOnboardWithoutOwner:      false,
		CanControlAsAgent:           false,
		CanRecoverAsAgent:           false,
		RequiresHumanOnlyStep:       true,
		RequiresOwnerOnlySecretOrID: true,
		RulesAsObservedOK:           false,
		Label:                       EligibilityLabelIneligible,
		Reasons:                     []string{string(AutonomyEligibilityReasonOwnerIdentityRequired)},
		CheckedAt:                   now,
	})

	result, err := RequireAutonomyEligibleTarget(root, target)
	if !errors.Is(err, ErrAutonomyEligibleTargetRequired) {
		t.Fatalf("RequireAutonomyEligibleTarget() error = %v, want %v", err, ErrAutonomyEligibleTargetRequired)
	}
	if result.Decision != AutonomyEligibilityDecisionIneligible {
		t.Fatalf("Decision = %q, want %q", result.Decision, AutonomyEligibilityDecisionIneligible)
	}
	wantReasons := []AutonomyEligibilityReason{AutonomyEligibilityReasonOwnerIdentityRequired}
	if !reflect.DeepEqual(result.Reasons, wantReasons) {
		t.Fatalf("Reasons = %#v, want %#v", result.Reasons, wantReasons)
	}
}

func TestRequireAutonomyEligibleTargetUnknownFailsClosed(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	target := AutonomyEligibilityTargetRef{
		Kind:       EligibilityTargetKindProvider,
		RegistryID: "missing-provider",
	}

	result, err := EvaluateAutonomyEligibility(root, target)
	if err != nil {
		t.Fatalf("EvaluateAutonomyEligibility() error = %v", err)
	}
	if result.Decision != AutonomyEligibilityDecisionUnknown {
		t.Fatalf("Decision = %q, want %q", result.Decision, AutonomyEligibilityDecisionUnknown)
	}

	result, err = RequireAutonomyEligibleTarget(root, target)
	if !errors.Is(err, ErrAutonomyEligibleTargetRequired) {
		t.Fatalf("RequireAutonomyEligibleTarget() error = %v, want %v", err, ErrAutonomyEligibleTargetRequired)
	}
	if result.Decision != AutonomyEligibilityDecisionUnknown {
		t.Fatalf("Decision = %q, want %q", result.Decision, AutonomyEligibilityDecisionUnknown)
	}
}

func TestEvaluateAutonomyEligibilitySharedSurfaceAcrossTargetKinds(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		kind EligibilityTargetKind
	}{
		{name: "platform", kind: EligibilityTargetKindPlatform},
		{name: "account-class", kind: EligibilityTargetKindAccountClass},
		{name: "container-class", kind: EligibilityTargetKindTreasuryContainerClass},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			now := time.Date(2026, 4, 6, 13, 0, 0, 0, time.UTC)
			target := AutonomyEligibilityTargetRef{
				Kind:       tc.kind,
				RegistryID: "target-" + string(tc.kind),
			}
			writeAutonomyEligibilityFixture(t, root, target, PlatformRecord{
				PlatformID:       target.RegistryID,
				PlatformName:     "name-" + string(tc.kind),
				TargetClass:      target.Kind,
				EligibilityLabel: EligibilityLabelAutonomyCompatible,
				LastCheckID:      "check-" + string(tc.kind),
				Notes:            []string{"registry note"},
				UpdatedAt:        now,
			}, EligibilityCheckRecord{
				CheckID:                     "check-" + string(tc.kind),
				TargetKind:                  target.Kind,
				TargetName:                  "name-" + string(tc.kind),
				CanCreateWithoutOwner:       true,
				CanOnboardWithoutOwner:      true,
				CanControlAsAgent:           true,
				CanRecoverAsAgent:           true,
				RequiresHumanOnlyStep:       false,
				RequiresOwnerOnlySecretOrID: false,
				RulesAsObservedOK:           true,
				Label:                       EligibilityLabelAutonomyCompatible,
				Reasons:                     []string{"operator-reviewed"},
				CheckedAt:                   now,
			})

			result, err := EvaluateAutonomyEligibility(root, target)
			if err != nil {
				t.Fatalf("EvaluateAutonomyEligibility() error = %v", err)
			}
			if result.Decision != AutonomyEligibilityDecisionEligible {
				t.Fatalf("Decision = %q, want %q", result.Decision, AutonomyEligibilityDecisionEligible)
			}
		})
	}
}

func TestRequireAutonomyEligibleTargetConflictingOrMalformedRegistryFailsClosed(t *testing.T) {
	t.Parallel()

	t.Run("conflicting labels", func(t *testing.T) {
		t.Parallel()

		root := t.TempDir()
		now := time.Date(2026, 4, 6, 14, 0, 0, 0, time.UTC)
		target := AutonomyEligibilityTargetRef{
			Kind:       EligibilityTargetKindProvider,
			RegistryID: "provider-conflict",
		}
		if err := StorePlatformRecord(root, PlatformRecord{
			PlatformID:       target.RegistryID,
			PlatformName:     "conflict.example",
			TargetClass:      target.Kind,
			EligibilityLabel: EligibilityLabelAutonomyCompatible,
			LastCheckID:      "check-conflict",
			Notes:            []string{"registry note"},
			UpdatedAt:        now,
		}); err != nil {
			t.Fatalf("StorePlatformRecord() error = %v", err)
		}
		if err := StoreEligibilityCheckRecord(root, EligibilityCheckRecord{
			CheckID:                     "check-conflict",
			TargetKind:                  target.Kind,
			TargetName:                  "conflict.example",
			CanCreateWithoutOwner:       false,
			CanOnboardWithoutOwner:      false,
			CanControlAsAgent:           false,
			CanRecoverAsAgent:           false,
			RequiresHumanOnlyStep:       true,
			RequiresOwnerOnlySecretOrID: false,
			RulesAsObservedOK:           false,
			Label:                       EligibilityLabelIneligible,
			Reasons:                     []string{string(AutonomyEligibilityReasonManualHumanCompletionRequired)},
			CheckedAt:                   now,
		}); err != nil {
			t.Fatalf("StoreEligibilityCheckRecord() error = %v", err)
		}

		_, err := RequireAutonomyEligibleTarget(root, target)
		if err == nil {
			t.Fatal("RequireAutonomyEligibleTarget() error = nil, want conflict failure")
		}
	})

	t.Run("malformed reason code", func(t *testing.T) {
		t.Parallel()

		root := t.TempDir()
		now := time.Date(2026, 4, 6, 14, 5, 0, 0, time.UTC)
		target := AutonomyEligibilityTargetRef{
			Kind:       EligibilityTargetKindPlatform,
			RegistryID: "platform-bad-reason",
		}
		if err := StorePlatformRecord(root, PlatformRecord{
			PlatformID:       target.RegistryID,
			PlatformName:     "bad-reason.example",
			TargetClass:      target.Kind,
			EligibilityLabel: EligibilityLabelIneligible,
			LastCheckID:      "check-bad-reason",
			Notes:            []string{"registry note"},
			UpdatedAt:        now,
		}); err != nil {
			t.Fatalf("StorePlatformRecord() error = %v", err)
		}
		if err := StoreEligibilityCheckRecord(root, EligibilityCheckRecord{
			CheckID:                     "check-bad-reason",
			TargetKind:                  target.Kind,
			TargetName:                  "bad-reason.example",
			CanCreateWithoutOwner:       false,
			CanOnboardWithoutOwner:      false,
			CanControlAsAgent:           false,
			CanRecoverAsAgent:           false,
			RequiresHumanOnlyStep:       true,
			RequiresOwnerOnlySecretOrID: false,
			RulesAsObservedOK:           false,
			Label:                       EligibilityLabelIneligible,
			Reasons:                     []string{"unsupported_reason_code"},
			CheckedAt:                   now,
		}); err != nil {
			t.Fatalf("StoreEligibilityCheckRecord() error = %v", err)
		}

		_, err := RequireAutonomyEligibleTarget(root, target)
		if err == nil {
			t.Fatal("RequireAutonomyEligibleTarget() error = nil, want malformed registry failure")
		}
	})
}

func TestAutonomyEligibilityReasonCodesStable(t *testing.T) {
	t.Parallel()

	cases := map[AutonomyEligibilityReason]string{
		AutonomyEligibilityReasonOwnerIdentityRequired:              "owner_identity_required",
		AutonomyEligibilityReasonOwnerLegalPersonhoodRequired:       "owner_legal_personhood_required",
		AutonomyEligibilityReasonOwnerPaymentMethodRequired:         "owner_payment_method_required",
		AutonomyEligibilityReasonManualHumanCompletionRequired:      "manual_human_completion_required",
		AutonomyEligibilityReasonHumanGatedKYCOrCustodialOnboarding: "human_gated_kyc_or_custodial_onboarding",
		AutonomyEligibilityReasonHiddenOwnerInfrastructureRequired:  "hidden_owner_infrastructure_required",
		AutonomyEligibilityReasonNotAutonomyCompatible:              "not_autonomy_compatible",
	}

	for got, want := range cases {
		if string(got) != want {
			t.Fatalf("reason code = %q, want %q", got, want)
		}
	}
}

func writeAutonomyEligibilityFixture(t *testing.T, root string, target AutonomyEligibilityTargetRef, record PlatformRecord, check EligibilityCheckRecord) {
	t.Helper()

	if err := StorePlatformRecord(root, record); err != nil {
		t.Fatalf("StorePlatformRecord(%s) error = %v", target.RegistryID, err)
	}
	if err := StoreEligibilityCheckRecord(root, check); err != nil {
		t.Fatalf("StoreEligibilityCheckRecord(%s) error = %v", check.CheckID, err)
	}
}
