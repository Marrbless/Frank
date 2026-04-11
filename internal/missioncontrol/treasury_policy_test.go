package missioncontrol

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestApplyDefaultTreasuryActivationPolicyPermittedCallsActivateFundedTreasuryOnce(t *testing.T) {
	fixtures := writeExecutionContextFrankRegistryFixtures(t)
	root := fixtures.root
	now := time.Date(2026, 4, 11, 13, 30, 0, 0, time.UTC)
	mustStoreTreasuryForMutationTest(t, root, validTreasuryRecord(now, func(record *TreasuryRecord) {
		record.TreasuryID = "treasury-policy-permitted"
		record.State = TreasuryStateFunded
		record.ContainerRefs = []FrankRegistryObjectRef{{
			Kind:     FrankRegistryObjectKindContainer,
			ObjectID: fixtures.container.ContainerID,
		}}
	}))

	originalMutation := treasuryActivationPolicyMutation
	t.Cleanup(func() { treasuryActivationPolicyMutation = originalMutation })

	calls := 0
	var gotRoot string
	var gotLease WriterLockLease
	var gotInput ActivateFundedTreasuryInput
	var gotNow time.Time
	treasuryActivationPolicyMutation = func(root string, lease WriterLockLease, input ActivateFundedTreasuryInput, now time.Time) error {
		calls++
		gotRoot = root
		gotLease = lease
		gotInput = input
		gotNow = now
		return nil
	}

	activatedAt := now.Add(2 * time.Minute)
	if err := ApplyDefaultTreasuryActivationPolicy(root, WriterLockLease{LeaseHolderID: "holder-1"}, DefaultTreasuryActivationPolicyInput{
		TreasuryRef: TreasuryRef{TreasuryID: "treasury-policy-permitted"},
	}, activatedAt); err != nil {
		t.Fatalf("ApplyDefaultTreasuryActivationPolicy() error = %v", err)
	}

	if calls != 1 {
		t.Fatalf("activation mutation calls = %d, want 1", calls)
	}
	if gotRoot != root {
		t.Fatalf("activation mutation root = %q, want %q", gotRoot, root)
	}
	if gotLease.LeaseHolderID != "holder-1" {
		t.Fatalf("activation mutation lease = %#v, want holder-1", gotLease)
	}
	if gotInput.TreasuryID != "treasury-policy-permitted" {
		t.Fatalf("activation mutation input = %#v, want treasury-policy-permitted", gotInput)
	}
	if !gotNow.Equal(activatedAt.UTC()) {
		t.Fatalf("activation mutation now = %s, want %s", gotNow, activatedAt.UTC())
	}
}

func TestApplyDefaultTreasuryActivationPolicyDisallowedDoesNotActivate(t *testing.T) {
	root := t.TempDir()
	now := time.Date(2026, 4, 11, 13, 45, 0, 0, time.UTC)
	target := AutonomyEligibilityTargetRef{
		Kind:       EligibilityTargetKindTreasuryContainerClass,
		RegistryID: "container-class-human-wallet",
	}
	writeFrankRegistryEligibilityFixture(t, root, target, EligibilityLabelIneligible, "container-class-human-wallet", "check-container-class-human-wallet", now)

	container := FrankContainerRecord{
		RecordVersion:        StoreRecordVersion,
		ContainerID:          "container-human-wallet",
		ContainerKind:        "wallet",
		Label:                "Human Wallet",
		ContainerClassID:     "container-class-human-wallet",
		State:                "candidate",
		EligibilityTargetRef: target,
		CreatedAt:            now.UTC(),
		UpdatedAt:            now.Add(time.Minute).UTC(),
	}
	if err := StoreFrankContainerRecord(root, container); err != nil {
		t.Fatalf("StoreFrankContainerRecord() error = %v", err)
	}
	mustStoreTreasuryForMutationTest(t, root, validTreasuryRecord(now.Add(2*time.Minute), func(record *TreasuryRecord) {
		record.TreasuryID = "treasury-policy-disallowed"
		record.State = TreasuryStateFunded
		record.ContainerRefs = []FrankRegistryObjectRef{{
			Kind:     FrankRegistryObjectKindContainer,
			ObjectID: container.ContainerID,
		}}
	}))

	originalMutation := treasuryActivationPolicyMutation
	t.Cleanup(func() { treasuryActivationPolicyMutation = originalMutation })

	calls := 0
	treasuryActivationPolicyMutation = func(string, WriterLockLease, ActivateFundedTreasuryInput, time.Time) error {
		calls++
		return nil
	}

	_, wantErr := RequireAutonomyEligibleTarget(root, target)
	if wantErr == nil {
		t.Fatal("RequireAutonomyEligibleTarget() error = nil, want ineligible treasury container policy rejection")
	}

	err := ApplyDefaultTreasuryActivationPolicy(root, WriterLockLease{LeaseHolderID: "holder-1"}, DefaultTreasuryActivationPolicyInput{
		TreasuryRef: TreasuryRef{TreasuryID: "treasury-policy-disallowed"},
	}, now.Add(3*time.Minute))
	if err == nil {
		t.Fatal("ApplyDefaultTreasuryActivationPolicy() error = nil, want policy rejection")
	}
	if err.Error() != wantErr.Error() {
		t.Fatalf("ApplyDefaultTreasuryActivationPolicy() error = %q, want %q", err.Error(), wantErr.Error())
	}
	if calls != 0 {
		t.Fatalf("activation mutation calls = %d, want 0", calls)
	}

	treasury, loadErr := LoadTreasuryRecord(root, "treasury-policy-disallowed")
	if loadErr != nil {
		t.Fatalf("LoadTreasuryRecord() error = %v", loadErr)
	}
	if treasury.State != TreasuryStateFunded {
		t.Fatalf("LoadTreasuryRecord().State = %q, want %q", treasury.State, TreasuryStateFunded)
	}
}

func TestApplyDefaultTreasuryActivationPolicyReplayStaysDeterministic(t *testing.T) {
	fixtures := writeExecutionContextFrankRegistryFixtures(t)
	root := fixtures.root
	now := time.Date(2026, 4, 11, 14, 0, 0, 0, time.UTC)
	mustStoreTreasuryForMutationTest(t, root, validTreasuryRecord(now, func(record *TreasuryRecord) {
		record.TreasuryID = "treasury-policy-replay"
		record.State = TreasuryStateFunded
		record.ContainerRefs = []FrankRegistryObjectRef{{
			Kind:     FrankRegistryObjectKindContainer,
			ObjectID: fixtures.container.ContainerID,
		}}
	}))

	originalMutation := treasuryActivationPolicyMutation
	t.Cleanup(func() { treasuryActivationPolicyMutation = originalMutation })

	calls := 0
	treasuryActivationPolicyMutation = func(root string, lease WriterLockLease, input ActivateFundedTreasuryInput, now time.Time) error {
		calls++
		return ActivateFundedTreasury(root, lease, input, now)
	}

	activatedAt := now.Add(time.Minute)
	if err := ApplyDefaultTreasuryActivationPolicy(root, WriterLockLease{LeaseHolderID: "holder-1"}, DefaultTreasuryActivationPolicyInput{
		TreasuryRef: TreasuryRef{TreasuryID: "treasury-policy-replay"},
	}, activatedAt); err != nil {
		t.Fatalf("ApplyDefaultTreasuryActivationPolicy(first) error = %v", err)
	}

	err := ApplyDefaultTreasuryActivationPolicy(root, WriterLockLease{LeaseHolderID: "holder-1"}, DefaultTreasuryActivationPolicyInput{
		TreasuryRef: TreasuryRef{TreasuryID: "treasury-policy-replay"},
	}, now.Add(2*time.Minute))
	if err == nil {
		t.Fatal("ApplyDefaultTreasuryActivationPolicy(replay) error = nil, want deterministic duplicate rejection")
	}
	if !strings.Contains(err.Error(), `mission store treasury "treasury-policy-replay" default activation policy requires state "funded", got "active"`) {
		t.Fatalf("ApplyDefaultTreasuryActivationPolicy(replay) error = %q, want active-state replay rejection", err.Error())
	}
	if calls != 1 {
		t.Fatalf("activation mutation calls = %d, want 1 across replay attempts", calls)
	}

	treasury, loadErr := LoadTreasuryRecord(root, "treasury-policy-replay")
	if loadErr != nil {
		t.Fatalf("LoadTreasuryRecord() error = %v", loadErr)
	}
	if treasury.State != TreasuryStateActive {
		t.Fatalf("LoadTreasuryRecord().State = %q, want %q", treasury.State, TreasuryStateActive)
	}
	if !treasury.UpdatedAt.Equal(activatedAt.UTC()) {
		t.Fatalf("LoadTreasuryRecord().UpdatedAt = %s, want %s", treasury.UpdatedAt, activatedAt.UTC())
	}

	assertNoTreasuryLedgerEntries(t, root, "treasury-policy-replay")
}

func TestApplyDefaultTreasuryActivationPolicyPreservesInspectTreasuryPreflightContract(t *testing.T) {
	t.Parallel()

	fixtures := writeExecutionContextFrankRegistryFixtures(t)
	root := fixtures.root
	now := time.Date(2026, 4, 11, 14, 15, 0, 0, time.UTC)
	mustStoreTreasuryForMutationTest(t, root, validTreasuryRecord(now, func(record *TreasuryRecord) {
		record.TreasuryID = "treasury-policy-readout"
		record.State = TreasuryStateFunded
		record.ContainerRefs = []FrankRegistryObjectRef{{
			Kind:     FrankRegistryObjectKindContainer,
			ObjectID: fixtures.container.ContainerID,
		}}
	}))

	if err := ApplyDefaultTreasuryActivationPolicy(root, WriterLockLease{LeaseHolderID: "holder-1"}, DefaultTreasuryActivationPolicyInput{
		TreasuryRef: TreasuryRef{TreasuryID: "treasury-policy-readout"},
	}, now.Add(time.Minute)); err != nil {
		t.Fatalf("ApplyDefaultTreasuryActivationPolicy() error = %v", err)
	}

	job := testExecutionJob()
	job.Plan.Steps[0].TreasuryRef = &TreasuryRef{TreasuryID: "treasury-policy-readout"}
	summary, err := NewInspectSummaryWithTreasuryPreflight(job, "build", root)
	if err != nil {
		t.Fatalf("NewInspectSummaryWithTreasuryPreflight() error = %v", err)
	}
	if len(summary.Steps) != 1 || summary.Steps[0].TreasuryPreflight == nil || summary.Steps[0].TreasuryPreflight.Treasury == nil {
		t.Fatalf("InspectSummary.Steps = %#v, want one treasury preflight step", summary.Steps)
	}
	if summary.Steps[0].TreasuryPreflight.Treasury.State != TreasuryStateActive {
		t.Fatalf("InspectSummary treasury state = %q, want %q", summary.Steps[0].TreasuryPreflight.Treasury.State, TreasuryStateActive)
	}
	if !reflect.DeepEqual(summary.Steps[0].TreasuryPreflight.Containers, []FrankContainerRecord{fixtures.container}) {
		t.Fatalf("InspectSummary treasury containers = %#v, want [%#v]", summary.Steps[0].TreasuryPreflight.Containers, fixtures.container)
	}
}
