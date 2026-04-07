package missioncontrol

import (
	"errors"
	"reflect"
	"testing"
	"time"
)

func TestEligibilityCheckRecordRoundTripAndList(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 6, 14, 15, 0, 0, time.FixedZone("offset", -4*60*60))

	checkB := EligibilityCheckRecord{
		CheckID:                     "check-b",
		TargetKind:                  EligibilityTargetKindPlatform,
		TargetName:                  "forum.example",
		CanCreateWithoutOwner:       true,
		CanOnboardWithoutOwner:      true,
		CanControlAsAgent:           true,
		CanRecoverAsAgent:           true,
		RequiresHumanOnlyStep:       false,
		RequiresOwnerOnlySecretOrID: false,
		RulesAsObservedOK:           true,
		Label:                       EligibilityLabelAutonomyCompatible,
		Reasons:                     []string{"self-service signup", "agent-managed recovery"},
		CheckedAt:                   now,
	}
	if err := StoreEligibilityCheckRecord(root, checkB); err != nil {
		t.Fatalf("StoreEligibilityCheckRecord(check-b) error = %v", err)
	}

	checkA := EligibilityCheckRecord{
		CheckID:                     "check-a",
		TargetKind:                  EligibilityTargetKindTreasuryContainerClass,
		TargetName:                  "kyc-exchange-wallet",
		CanCreateWithoutOwner:       false,
		CanOnboardWithoutOwner:      false,
		CanControlAsAgent:           false,
		CanRecoverAsAgent:           false,
		RequiresHumanOnlyStep:       true,
		RequiresOwnerOnlySecretOrID: true,
		RulesAsObservedOK:           false,
		Label:                       EligibilityLabelIneligible,
		Reasons:                     []string{"requires human KYC"},
		CheckedAt:                   now.Add(time.Minute),
	}
	if err := StoreEligibilityCheckRecord(root, checkA); err != nil {
		t.Fatalf("StoreEligibilityCheckRecord(check-a) error = %v", err)
	}

	got, err := LoadEligibilityCheckRecord(root, "check-a")
	if err != nil {
		t.Fatalf("LoadEligibilityCheckRecord() error = %v", err)
	}

	checkA.RecordVersion = StoreRecordVersion
	checkA.CheckedAt = checkA.CheckedAt.UTC()
	if !reflect.DeepEqual(got, checkA) {
		t.Fatalf("LoadEligibilityCheckRecord() = %#v, want %#v", got, checkA)
	}

	records, err := ListEligibilityCheckRecords(root)
	if err != nil {
		t.Fatalf("ListEligibilityCheckRecords() error = %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("ListEligibilityCheckRecords() len = %d, want 2", len(records))
	}
	if records[0].CheckID != "check-a" || records[1].CheckID != "check-b" {
		t.Fatalf("ListEligibilityCheckRecords() ids = [%q %q], want [check-a check-b]", records[0].CheckID, records[1].CheckID)
	}
}

func TestPlatformRecordRoundTripAndList(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 6, 18, 30, 0, 0, time.FixedZone("offset", 2*60*60))

	if err := StorePlatformRecord(root, PlatformRecord{
		PlatformID:       "platform-b",
		PlatformName:     "wallet-balance",
		TargetClass:      EligibilityTargetKindTreasuryContainerClass,
		EligibilityLabel: EligibilityLabelAutonomyCompatible,
		LastCheckID:      "check-b",
		Notes:            []string{"self-custodial"},
		UpdatedAt:        now,
	}); err != nil {
		t.Fatalf("StorePlatformRecord(platform-b) error = %v", err)
	}

	want := PlatformRecord{
		PlatformID:       "platform-a",
		PlatformName:     "marketplace-seller-balance",
		TargetClass:      EligibilityTargetKindPlatform,
		EligibilityLabel: EligibilityLabelHumanGated,
		LastCheckID:      "check-a",
		Notes:            []string{"manual onboarding observed"},
		UpdatedAt:        now.Add(time.Minute),
	}
	if err := StorePlatformRecord(root, want); err != nil {
		t.Fatalf("StorePlatformRecord(platform-a) error = %v", err)
	}

	got, err := LoadPlatformRecord(root, "platform-a")
	if err != nil {
		t.Fatalf("LoadPlatformRecord() error = %v", err)
	}

	want.RecordVersion = StoreRecordVersion
	want.UpdatedAt = want.UpdatedAt.UTC()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("LoadPlatformRecord() = %#v, want %#v", got, want)
	}

	records, err := ListPlatformRecords(root)
	if err != nil {
		t.Fatalf("ListPlatformRecords() error = %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("ListPlatformRecords() len = %d, want 2", len(records))
	}
	if records[0].PlatformID != "platform-a" || records[1].PlatformID != "platform-b" {
		t.Fatalf("ListPlatformRecords() ids = [%q %q], want [platform-a platform-b]", records[0].PlatformID, records[1].PlatformID)
	}
}

func TestEligibilityRegistryNotFoundAndValidation(t *testing.T) {
	t.Parallel()

	root := t.TempDir()

	if _, err := LoadEligibilityCheckRecord(root, "missing"); !errors.Is(err, ErrEligibilityCheckRecordNotFound) {
		t.Fatalf("LoadEligibilityCheckRecord(missing) error = %v, want %v", err, ErrEligibilityCheckRecordNotFound)
	}
	if _, err := LoadPlatformRecord(root, "missing"); !errors.Is(err, ErrPlatformRecordNotFound) {
		t.Fatalf("LoadPlatformRecord(missing) error = %v, want %v", err, ErrPlatformRecordNotFound)
	}

	err := StoreEligibilityCheckRecord(root, EligibilityCheckRecord{
		CheckID:    "bad-check",
		TargetKind: EligibilityTargetKind("unknown"),
		TargetName: "bad",
		Label:      EligibilityLabelAutonomyCompatible,
		Reasons:    []string{"note"},
		CheckedAt:  time.Now(),
	})
	if err == nil || err.Error() != `mission store eligibility check target_kind "unknown" is invalid` {
		t.Fatalf("StoreEligibilityCheckRecord(invalid) error = %v, want invalid target_kind", err)
	}

	err = StorePlatformRecord(root, PlatformRecord{
		PlatformID:       "bad-platform",
		PlatformName:     "bad",
		TargetClass:      EligibilityTargetKindPlatform,
		EligibilityLabel: EligibilityLabel("unknown"),
		LastCheckID:      "check-1",
		Notes:            []string{"note"},
		UpdatedAt:        time.Now(),
	})
	if err == nil || err.Error() != `mission store platform record eligibility_label "unknown" is invalid` {
		t.Fatalf("StorePlatformRecord(invalid) error = %v, want invalid eligibility_label", err)
	}
}
