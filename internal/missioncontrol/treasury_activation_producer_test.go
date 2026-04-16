package missioncontrol

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestProduceFundedTreasuryActivationSuccessfulExecution(t *testing.T) {
	t.Parallel()

	fixtures := writeExecutionContextFrankRegistryFixtures(t)
	root := fixtures.root
	now := time.Date(2026, 4, 14, 13, 0, 0, 0, time.UTC)
	mustStoreTreasuryForMutationTest(t, root, validTreasuryRecord(now, func(record *TreasuryRecord) {
		record.TreasuryID = "treasury-activation-producer"
		record.State = TreasuryStateFunded
		record.ContainerRefs = []FrankRegistryObjectRef{{
			Kind:     FrankRegistryObjectKindContainer,
			ObjectID: fixtures.container.ContainerID,
		}}
	}))

	activatedAt := now.Add(2 * time.Minute)
	if err := ProduceFundedTreasuryActivation(root, WriterLockLease{LeaseHolderID: "holder-1"}, DefaultTreasuryActivationPolicyInput{
		TreasuryRef: TreasuryRef{TreasuryID: "treasury-activation-producer"},
	}, activatedAt); err != nil {
		t.Fatalf("ProduceFundedTreasuryActivation() error = %v", err)
	}

	treasury, err := LoadTreasuryRecord(root, "treasury-activation-producer")
	if err != nil {
		t.Fatalf("LoadTreasuryRecord() error = %v", err)
	}
	if treasury.State != TreasuryStateActive {
		t.Fatalf("LoadTreasuryRecord().State = %q, want %q", treasury.State, TreasuryStateActive)
	}
	if !treasury.UpdatedAt.Equal(activatedAt.UTC()) {
		t.Fatalf("LoadTreasuryRecord().UpdatedAt = %s, want %s", treasury.UpdatedAt, activatedAt.UTC())
	}

	assertNoTreasuryLedgerEntries(t, root, "treasury-activation-producer")
}

func TestProduceFundedTreasuryActivationReplayFailsClosed(t *testing.T) {
	t.Parallel()

	fixtures := writeExecutionContextFrankRegistryFixtures(t)
	root := fixtures.root
	now := time.Date(2026, 4, 14, 13, 10, 0, 0, time.UTC)
	mustStoreTreasuryForMutationTest(t, root, validTreasuryRecord(now, func(record *TreasuryRecord) {
		record.TreasuryID = "treasury-activation-producer-replay"
		record.State = TreasuryStateFunded
		record.ContainerRefs = []FrankRegistryObjectRef{{
			Kind:     FrankRegistryObjectKindContainer,
			ObjectID: fixtures.container.ContainerID,
		}}
	}))

	input := DefaultTreasuryActivationPolicyInput{
		TreasuryRef: TreasuryRef{TreasuryID: "treasury-activation-producer-replay"},
	}
	activatedAt := now.Add(2 * time.Minute)
	if err := ProduceFundedTreasuryActivation(root, WriterLockLease{LeaseHolderID: "holder-1"}, input, activatedAt); err != nil {
		t.Fatalf("ProduceFundedTreasuryActivation(first) error = %v", err)
	}

	err := ProduceFundedTreasuryActivation(root, WriterLockLease{LeaseHolderID: "holder-1"}, input, now.Add(3*time.Minute))
	if err == nil {
		t.Fatal("ProduceFundedTreasuryActivation(replay) error = nil, want deterministic duplicate rejection")
	}
	if !strings.Contains(err.Error(), `mission store treasury "treasury-activation-producer-replay" default activation policy requires state "funded", got "active"`) {
		t.Fatalf("ProduceFundedTreasuryActivation(replay) error = %q, want active-state replay rejection", err.Error())
	}

	treasury, err := LoadTreasuryRecord(root, input.TreasuryRef.TreasuryID)
	if err != nil {
		t.Fatalf("LoadTreasuryRecord() error = %v", err)
	}
	if treasury.State != TreasuryStateActive {
		t.Fatalf("LoadTreasuryRecord().State = %q, want %q", treasury.State, TreasuryStateActive)
	}

	assertNoTreasuryLedgerEntries(t, root, input.TreasuryRef.TreasuryID)
}

func TestProduceFundedTreasuryActivationFailsClosedWithoutActiveContainer(t *testing.T) {
	t.Parallel()

	root := writeExecutionContextFrankRegistryFixtures(t).root
	now := time.Date(2026, 4, 14, 13, 20, 0, 0, time.UTC)
	writeMalformedTreasuryRecordForMutationTest(t, root, validTreasuryRecord(now, func(record *TreasuryRecord) {
		record.TreasuryID = "treasury-activation-producer-no-container"
		record.State = TreasuryStateFunded
		record.ContainerRefs = nil
	}))

	err := ProduceFundedTreasuryActivation(root, WriterLockLease{LeaseHolderID: "holder-1"}, DefaultTreasuryActivationPolicyInput{
		TreasuryRef: TreasuryRef{TreasuryID: "treasury-activation-producer-no-container"},
	}, now.Add(2*time.Minute))
	if err == nil {
		t.Fatal("ProduceFundedTreasuryActivation() error = nil, want missing active-container rejection")
	}
	if !strings.Contains(err.Error(), `mission store treasury state "funded" requires exactly one active_container_id derivable from container_refs`) {
		t.Fatalf("ProduceFundedTreasuryActivation() error = %q, want missing active-container rejection", err.Error())
	}

	assertNoTreasuryLedgerEntries(t, root, "treasury-activation-producer-no-container")
}

func TestProduceFundedTreasuryActivationFailsClosedWithAmbiguousActiveContainer(t *testing.T) {
	t.Parallel()

	fixtures := writeExecutionContextFrankRegistryFixtures(t)
	root := fixtures.root
	now := time.Date(2026, 4, 14, 13, 30, 0, 0, time.UTC)
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
		record.TreasuryID = "treasury-activation-producer-ambiguous"
		record.State = TreasuryStateFunded
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

	err := ProduceFundedTreasuryActivation(root, WriterLockLease{LeaseHolderID: "holder-1"}, DefaultTreasuryActivationPolicyInput{
		TreasuryRef: TreasuryRef{TreasuryID: "treasury-activation-producer-ambiguous"},
	}, now.Add(3*time.Minute))
	if err == nil {
		t.Fatal("ProduceFundedTreasuryActivation() error = nil, want ambiguous active-container rejection")
	}
	if !strings.Contains(err.Error(), `mission store treasury state "funded" requires exactly one active_container_id derivable from container_refs`) {
		t.Fatalf("ProduceFundedTreasuryActivation() error = %q, want ambiguous active-container rejection", err.Error())
	}

	assertNoTreasuryLedgerEntries(t, root, "treasury-activation-producer-ambiguous")
}

func TestProduceFundedTreasuryActivationFailsClosedOutsideFundedState(t *testing.T) {
	t.Parallel()

	fixtures := writeExecutionContextFrankRegistryFixtures(t)
	root := fixtures.root
	now := time.Date(2026, 4, 14, 13, 40, 0, 0, time.UTC)
	mustStoreTreasuryForMutationTest(t, root, validTreasuryRecord(now, func(record *TreasuryRecord) {
		record.TreasuryID = "treasury-activation-producer-active"
		record.State = TreasuryStateActive
		record.ContainerRefs = []FrankRegistryObjectRef{{
			Kind:     FrankRegistryObjectKindContainer,
			ObjectID: fixtures.container.ContainerID,
		}}
	}))

	err := ProduceFundedTreasuryActivation(root, WriterLockLease{LeaseHolderID: "holder-1"}, DefaultTreasuryActivationPolicyInput{
		TreasuryRef: TreasuryRef{TreasuryID: "treasury-activation-producer-active"},
	}, now.Add(2*time.Minute))
	if err == nil {
		t.Fatal("ProduceFundedTreasuryActivation() error = nil, want invalid activation-state rejection")
	}
	if !strings.Contains(err.Error(), `mission store treasury "treasury-activation-producer-active" default activation policy requires state "funded", got "active"`) {
		t.Fatalf("ProduceFundedTreasuryActivation() error = %q, want invalid activation-state rejection", err.Error())
	}

	assertNoTreasuryLedgerEntries(t, root, "treasury-activation-producer-active")
}

func TestProduceFundedTreasuryActivationPreservesInspectTreasuryPreflightContract(t *testing.T) {
	t.Parallel()

	fixtures := writeExecutionContextFrankRegistryFixtures(t)
	root := fixtures.root
	now := time.Date(2026, 4, 14, 13, 50, 0, 0, time.UTC)
	mustStoreTreasuryForMutationTest(t, root, validTreasuryRecord(now, func(record *TreasuryRecord) {
		record.TreasuryID = "treasury-activation-producer-readout"
		record.State = TreasuryStateFunded
		record.ContainerRefs = []FrankRegistryObjectRef{{
			Kind:     FrankRegistryObjectKindContainer,
			ObjectID: fixtures.container.ContainerID,
		}}
	}))

	if err := ProduceFundedTreasuryActivation(root, WriterLockLease{LeaseHolderID: "holder-1"}, DefaultTreasuryActivationPolicyInput{
		TreasuryRef: TreasuryRef{TreasuryID: "treasury-activation-producer-readout"},
	}, now.Add(time.Minute)); err != nil {
		t.Fatalf("ProduceFundedTreasuryActivation() error = %v", err)
	}

	job := testExecutionJob()
	job.Plan.Steps[0].TreasuryRef = &TreasuryRef{TreasuryID: "treasury-activation-producer-readout"}
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
