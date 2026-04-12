package missioncontrol

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestApplyDefaultTreasuryBootstrapPolicyPermittedCallsRecordFirstTreasuryAcquisitionOnce(t *testing.T) {
	fixtures := writeExecutionContextFrankRegistryFixtures(t)
	root := fixtures.root
	now := time.Date(2026, 4, 11, 12, 45, 0, 0, time.UTC)
	mustStoreTreasuryForMutationTest(t, root, validTreasuryRecord(now, func(record *TreasuryRecord) {
		record.TreasuryID = "treasury-bootstrap-policy-permitted"
		record.State = TreasuryStateBootstrap
		record.ContainerRefs = []FrankRegistryObjectRef{{
			Kind:     FrankRegistryObjectKindContainer,
			ObjectID: fixtures.container.ContainerID,
		}}
	}))

	originalMutation := treasuryBootstrapPolicyMutation
	t.Cleanup(func() { treasuryBootstrapPolicyMutation = originalMutation })

	calls := 0
	var gotRoot string
	var gotLease WriterLockLease
	var gotInput FirstTreasuryAcquisitionInput
	var gotNow time.Time
	treasuryBootstrapPolicyMutation = func(root string, lease WriterLockLease, input FirstTreasuryAcquisitionInput, now time.Time) error {
		calls++
		gotRoot = root
		gotLease = lease
		gotInput = input
		gotNow = now
		return nil
	}

	recordedAt := now.Add(2 * time.Minute)
	if err := ApplyDefaultTreasuryBootstrapPolicy(root, WriterLockLease{LeaseHolderID: "holder-1"}, DefaultTreasuryBootstrapPolicyInput{
		TreasuryRef: TreasuryRef{TreasuryID: "treasury-bootstrap-policy-permitted"},
		EntryID:     "entry-first-value",
		AssetCode:   "USD",
		Amount:      "10.00",
		SourceRef:   "payout:listing-1",
	}, recordedAt); err != nil {
		t.Fatalf("ApplyDefaultTreasuryBootstrapPolicy() error = %v", err)
	}

	if calls != 1 {
		t.Fatalf("bootstrap mutation calls = %d, want 1", calls)
	}
	if gotRoot != root {
		t.Fatalf("bootstrap mutation root = %q, want %q", gotRoot, root)
	}
	if gotLease.LeaseHolderID != "holder-1" {
		t.Fatalf("bootstrap mutation lease = %#v, want holder-1", gotLease)
	}
	if gotInput.TreasuryID != "treasury-bootstrap-policy-permitted" {
		t.Fatalf("bootstrap mutation input = %#v, want treasury-bootstrap-policy-permitted", gotInput)
	}
	if gotInput.EntryID != "entry-first-value" || gotInput.AssetCode != "USD" || gotInput.Amount != "10.00" || gotInput.SourceRef != "payout:listing-1" {
		t.Fatalf("bootstrap mutation input = %#v, want canonical acquisition payload", gotInput)
	}
	if !gotNow.Equal(recordedAt.UTC()) {
		t.Fatalf("bootstrap mutation now = %s, want %s", gotNow, recordedAt.UTC())
	}
}

func TestApplyDefaultTreasuryBootstrapPolicyReplayStaysDeterministic(t *testing.T) {
	fixtures := writeExecutionContextFrankRegistryFixtures(t)
	root := fixtures.root
	now := time.Date(2026, 4, 11, 13, 0, 0, 0, time.UTC)
	mustStoreTreasuryForMutationTest(t, root, validTreasuryRecord(now, func(record *TreasuryRecord) {
		record.TreasuryID = "treasury-bootstrap-policy-replay"
		record.State = TreasuryStateBootstrap
		record.ContainerRefs = []FrankRegistryObjectRef{{
			Kind:     FrankRegistryObjectKindContainer,
			ObjectID: fixtures.container.ContainerID,
		}}
	}))

	originalMutation := treasuryBootstrapPolicyMutation
	t.Cleanup(func() { treasuryBootstrapPolicyMutation = originalMutation })

	calls := 0
	treasuryBootstrapPolicyMutation = func(root string, lease WriterLockLease, input FirstTreasuryAcquisitionInput, now time.Time) error {
		calls++
		return RecordFirstTreasuryAcquisition(root, lease, input, now)
	}

	recordedAt := now.Add(time.Minute)
	input := DefaultTreasuryBootstrapPolicyInput{
		TreasuryRef: TreasuryRef{TreasuryID: "treasury-bootstrap-policy-replay"},
		EntryID:     "entry-first-value",
		AssetCode:   "USD",
		Amount:      "10.00",
		SourceRef:   "payout:listing-1",
	}
	if err := ApplyDefaultTreasuryBootstrapPolicy(root, WriterLockLease{LeaseHolderID: "holder-1"}, input, recordedAt); err != nil {
		t.Fatalf("ApplyDefaultTreasuryBootstrapPolicy(first) error = %v", err)
	}

	err := ApplyDefaultTreasuryBootstrapPolicy(root, WriterLockLease{LeaseHolderID: "holder-1"}, input, now.Add(2*time.Minute))
	if err == nil {
		t.Fatal("ApplyDefaultTreasuryBootstrapPolicy(replay) error = nil, want deterministic duplicate rejection")
	}
	if !strings.Contains(err.Error(), `mission store treasury "treasury-bootstrap-policy-replay" default bootstrap policy requires state "bootstrap", got "funded"`) {
		t.Fatalf("ApplyDefaultTreasuryBootstrapPolicy(replay) error = %q, want funded-state replay rejection", err.Error())
	}
	if calls != 1 {
		t.Fatalf("bootstrap mutation calls = %d, want 1 across replay attempts", calls)
	}

	treasury, loadErr := LoadTreasuryRecord(root, "treasury-bootstrap-policy-replay")
	if loadErr != nil {
		t.Fatalf("LoadTreasuryRecord() error = %v", loadErr)
	}
	if treasury.State != TreasuryStateFunded {
		t.Fatalf("LoadTreasuryRecord().State = %q, want %q", treasury.State, TreasuryStateFunded)
	}
	if !treasury.UpdatedAt.Equal(recordedAt.UTC()) {
		t.Fatalf("LoadTreasuryRecord().UpdatedAt = %s, want %s", treasury.UpdatedAt, recordedAt.UTC())
	}

	entries, err := ListTreasuryLedgerEntries(root, "treasury-bootstrap-policy-replay")
	if err != nil {
		t.Fatalf("ListTreasuryLedgerEntries() error = %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("ListTreasuryLedgerEntries() len = %d, want 1", len(entries))
	}
	if entries[0].EntryID != input.EntryID || entries[0].EntryKind != TreasuryLedgerEntryKindAcquisition {
		t.Fatalf("ListTreasuryLedgerEntries() = %#v, want one recorded acquisition entry", entries)
	}
}

func TestApplyDefaultTreasuryBootstrapPolicyFailsClosedOnMissingOrMalformedTreasury(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 11, 13, 10, 0, 0, time.UTC)

	t.Run("missing treasury", func(t *testing.T) {
		t.Parallel()

		root := t.TempDir()
		err := ApplyDefaultTreasuryBootstrapPolicy(root, WriterLockLease{LeaseHolderID: "holder-1"}, DefaultTreasuryBootstrapPolicyInput{
			TreasuryRef: TreasuryRef{TreasuryID: "treasury-missing"},
			EntryID:     "entry-first-value",
			AssetCode:   "USD",
			Amount:      "10.00",
			SourceRef:   "payout:listing-1",
		}, now.Add(time.Minute))
		if err == nil {
			t.Fatal("ApplyDefaultTreasuryBootstrapPolicy() error = nil, want missing treasury rejection")
		}
		if !strings.Contains(err.Error(), ErrTreasuryRecordNotFound.Error()) {
			t.Fatalf("ApplyDefaultTreasuryBootstrapPolicy() error = %q, want missing treasury rejection", err.Error())
		}
	})

	t.Run("malformed treasury", func(t *testing.T) {
		t.Parallel()

		fixtures := writeExecutionContextFrankRegistryFixtures(t)
		root := fixtures.root
		writeMalformedTreasuryRecordForMutationTest(t, root, validTreasuryRecord(now, func(record *TreasuryRecord) {
			record.TreasuryID = "treasury-malformed"
			record.DisplayName = ""
			record.State = TreasuryStateBootstrap
			record.ContainerRefs = []FrankRegistryObjectRef{{
				Kind:     FrankRegistryObjectKindContainer,
				ObjectID: fixtures.container.ContainerID,
			}}
		}))

		err := ApplyDefaultTreasuryBootstrapPolicy(root, WriterLockLease{LeaseHolderID: "holder-1"}, DefaultTreasuryBootstrapPolicyInput{
			TreasuryRef: TreasuryRef{TreasuryID: "treasury-malformed"},
			EntryID:     "entry-first-value",
			AssetCode:   "USD",
			Amount:      "10.00",
			SourceRef:   "payout:listing-1",
		}, now.Add(2*time.Minute))
		if err == nil {
			t.Fatal("ApplyDefaultTreasuryBootstrapPolicy() error = nil, want malformed treasury rejection")
		}
		if !strings.Contains(err.Error(), "mission store treasury display_name is required") {
			t.Fatalf("ApplyDefaultTreasuryBootstrapPolicy() error = %q, want malformed treasury rejection", err.Error())
		}

		assertNoTreasuryLedgerEntries(t, root, "treasury-malformed")
	})
}

func TestApplyDefaultTreasuryBootstrapPolicyFailsClosedWithoutActiveContainer(t *testing.T) {
	t.Parallel()

	root := writeExecutionContextFrankRegistryFixtures(t).root
	now := time.Date(2026, 4, 11, 13, 15, 0, 0, time.UTC)
	writeMalformedTreasuryRecordForMutationTest(t, root, validTreasuryRecord(now, func(record *TreasuryRecord) {
		record.TreasuryID = "treasury-bootstrap-policy-no-container"
		record.State = TreasuryStateBootstrap
		record.ContainerRefs = nil
	}))

	err := ApplyDefaultTreasuryBootstrapPolicy(root, WriterLockLease{LeaseHolderID: "holder-1"}, DefaultTreasuryBootstrapPolicyInput{
		TreasuryRef: TreasuryRef{TreasuryID: "treasury-bootstrap-policy-no-container"},
		EntryID:     "entry-first-value",
		AssetCode:   "USD",
		Amount:      "10.00",
		SourceRef:   "payout:listing-1",
	}, now.Add(2*time.Minute))
	if err == nil {
		t.Fatal("ApplyDefaultTreasuryBootstrapPolicy() error = nil, want missing active-container rejection")
	}
	if !strings.Contains(err.Error(), `mission store treasury "treasury-bootstrap-policy-no-container" default bootstrap policy requires exactly one active_container_id derivable from container_refs`) {
		t.Fatalf("ApplyDefaultTreasuryBootstrapPolicy() error = %q, want missing active-container rejection", err.Error())
	}

	assertNoTreasuryLedgerEntries(t, root, "treasury-bootstrap-policy-no-container")
}

func TestApplyDefaultTreasuryBootstrapPolicyFailsClosedWithAmbiguousActiveContainer(t *testing.T) {
	t.Parallel()

	fixtures := writeExecutionContextFrankRegistryFixtures(t)
	root := fixtures.root
	now := time.Date(2026, 4, 11, 13, 30, 0, 0, time.UTC)
	target := AutonomyEligibilityTargetRef{
		Kind:       EligibilityTargetKindTreasuryContainerClass,
		RegistryID: "container-class-wallet-2",
	}
	writeFrankRegistryEligibilityFixture(t, root, target, EligibilityLabelAutonomyCompatible, "container-class-wallet-2", "check-container-class-wallet-2", now)
	secondContainer := FrankContainerRecord{
		RecordVersion:        StoreRecordVersion,
		ContainerID:          "container-wallet-2",
		ContainerKind:        "wallet",
		Label:                "Secondary Wallet",
		ContainerClassID:     "container-class-wallet-2",
		State:                "active",
		EligibilityTargetRef: target,
		CreatedAt:            now.UTC(),
		UpdatedAt:            now.Add(time.Minute).UTC(),
	}
	if err := StoreFrankContainerRecord(root, secondContainer); err != nil {
		t.Fatalf("StoreFrankContainerRecord() error = %v", err)
	}
	writeMalformedTreasuryRecordForMutationTest(t, root, validTreasuryRecord(now.Add(2*time.Minute), func(record *TreasuryRecord) {
		record.TreasuryID = "treasury-bootstrap-policy-ambiguous"
		record.State = TreasuryStateBootstrap
		record.ContainerRefs = []FrankRegistryObjectRef{
			{
				Kind:     FrankRegistryObjectKindContainer,
				ObjectID: fixtures.container.ContainerID,
			},
			{
				Kind:     FrankRegistryObjectKindContainer,
				ObjectID: secondContainer.ContainerID,
			},
		}
	}))

	err := ApplyDefaultTreasuryBootstrapPolicy(root, WriterLockLease{LeaseHolderID: "holder-1"}, DefaultTreasuryBootstrapPolicyInput{
		TreasuryRef: TreasuryRef{TreasuryID: "treasury-bootstrap-policy-ambiguous"},
		EntryID:     "entry-first-value",
		AssetCode:   "USD",
		Amount:      "10.00",
		SourceRef:   "payout:listing-1",
	}, now.Add(3*time.Minute))
	if err == nil {
		t.Fatal("ApplyDefaultTreasuryBootstrapPolicy() error = nil, want ambiguous active-container rejection")
	}
	if !strings.Contains(err.Error(), `mission store treasury "treasury-bootstrap-policy-ambiguous" default bootstrap policy requires exactly one active_container_id derivable from container_refs`) {
		t.Fatalf("ApplyDefaultTreasuryBootstrapPolicy() error = %q, want ambiguous active-container rejection", err.Error())
	}

	assertNoTreasuryLedgerEntries(t, root, "treasury-bootstrap-policy-ambiguous")
}

func TestApplyDefaultTreasuryBootstrapPolicyFailsClosedOutsideBootstrapState(t *testing.T) {
	t.Parallel()

	fixtures := writeExecutionContextFrankRegistryFixtures(t)
	root := fixtures.root
	now := time.Date(2026, 4, 11, 13, 45, 0, 0, time.UTC)
	mustStoreTreasuryForMutationTest(t, root, validTreasuryRecord(now, func(record *TreasuryRecord) {
		record.TreasuryID = "treasury-bootstrap-policy-funded"
		record.State = TreasuryStateFunded
		record.ContainerRefs = []FrankRegistryObjectRef{{
			Kind:     FrankRegistryObjectKindContainer,
			ObjectID: fixtures.container.ContainerID,
		}}
	}))

	err := ApplyDefaultTreasuryBootstrapPolicy(root, WriterLockLease{LeaseHolderID: "holder-1"}, DefaultTreasuryBootstrapPolicyInput{
		TreasuryRef: TreasuryRef{TreasuryID: "treasury-bootstrap-policy-funded"},
		EntryID:     "entry-first-value",
		AssetCode:   "USD",
		Amount:      "10.00",
		SourceRef:   "payout:listing-1",
	}, now.Add(2*time.Minute))
	if err == nil {
		t.Fatal("ApplyDefaultTreasuryBootstrapPolicy() error = nil, want invalid bootstrap-state rejection")
	}
	if !strings.Contains(err.Error(), `mission store treasury "treasury-bootstrap-policy-funded" default bootstrap policy requires state "bootstrap", got "funded"`) {
		t.Fatalf("ApplyDefaultTreasuryBootstrapPolicy() error = %q, want invalid bootstrap-state rejection", err.Error())
	}

	assertNoTreasuryLedgerEntries(t, root, "treasury-bootstrap-policy-funded")
}

func TestApplyDefaultTreasuryBootstrapPolicyPreservesInspectTreasuryPreflightContract(t *testing.T) {
	t.Parallel()

	fixtures := writeExecutionContextFrankRegistryFixtures(t)
	root := fixtures.root
	now := time.Date(2026, 4, 11, 14, 0, 0, 0, time.UTC)
	mustStoreTreasuryForMutationTest(t, root, validTreasuryRecord(now, func(record *TreasuryRecord) {
		record.TreasuryID = "treasury-bootstrap-policy-readout"
		record.State = TreasuryStateBootstrap
		record.ContainerRefs = []FrankRegistryObjectRef{{
			Kind:     FrankRegistryObjectKindContainer,
			ObjectID: fixtures.container.ContainerID,
		}}
	}))

	if err := ApplyDefaultTreasuryBootstrapPolicy(root, WriterLockLease{LeaseHolderID: "holder-1"}, DefaultTreasuryBootstrapPolicyInput{
		TreasuryRef: TreasuryRef{TreasuryID: "treasury-bootstrap-policy-readout"},
		EntryID:     "entry-first-value",
		AssetCode:   "USD",
		Amount:      "10.00",
		SourceRef:   "payout:listing-1",
	}, now.Add(time.Minute)); err != nil {
		t.Fatalf("ApplyDefaultTreasuryBootstrapPolicy() error = %v", err)
	}

	job := testExecutionJob()
	job.Plan.Steps[0].TreasuryRef = &TreasuryRef{TreasuryID: "treasury-bootstrap-policy-readout"}
	summary, err := NewInspectSummaryWithTreasuryPreflight(job, "build", root)
	if err != nil {
		t.Fatalf("NewInspectSummaryWithTreasuryPreflight() error = %v", err)
	}
	if len(summary.Steps) != 1 || summary.Steps[0].TreasuryPreflight == nil || summary.Steps[0].TreasuryPreflight.Treasury == nil {
		t.Fatalf("InspectSummary.Steps = %#v, want one treasury preflight step", summary.Steps)
	}
	if summary.Steps[0].TreasuryPreflight.Treasury.State != TreasuryStateFunded {
		t.Fatalf("InspectSummary treasury state = %q, want %q", summary.Steps[0].TreasuryPreflight.Treasury.State, TreasuryStateFunded)
	}
	if !reflect.DeepEqual(summary.Steps[0].TreasuryPreflight.Containers, []FrankContainerRecord{fixtures.container}) {
		t.Fatalf("InspectSummary treasury containers = %#v, want [%#v]", summary.Steps[0].TreasuryPreflight.Containers, fixtures.container)
	}
}

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
