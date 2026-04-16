package missioncontrol

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestProduceFirstValueTreasuryBootstrapSuccessfulExecution(t *testing.T) {
	t.Parallel()

	fixtures := writeExecutionContextFrankRegistryFixtures(t)
	root := fixtures.root
	now := time.Date(2026, 4, 13, 13, 0, 0, 0, time.UTC)
	mustStoreTreasuryForMutationTest(t, root, validTreasuryRecord(now, func(record *TreasuryRecord) {
		record.TreasuryID = "treasury-bootstrap-producer"
		record.State = TreasuryStateBootstrap
		record.ContainerRefs = []FrankRegistryObjectRef{{
			Kind:     FrankRegistryObjectKindContainer,
			ObjectID: fixtures.container.ContainerID,
		}}
	}))

	recordedAt := now.Add(2 * time.Minute)
	if err := RecordFirstTreasuryAcquisition(root, WriterLockLease{LeaseHolderID: "holder-1"}, FirstTreasuryAcquisitionInput{
		TreasuryID: "treasury-bootstrap-producer",
		EntryID:    "entry-first-value",
		AssetCode:  "USD",
		Amount:     "10.00",
		SourceRef:  "payout:listing-1",
	}, recordedAt); err != nil {
		t.Fatalf("RecordFirstTreasuryAcquisition() error = %v", err)
	}

	input := FirstValueTreasuryBootstrapInput{
		TreasuryRef: TreasuryRef{TreasuryID: "treasury-bootstrap-producer"},
		EntryID:     "entry-first-value",
	}
	activatedAt := now.Add(3 * time.Minute)
	if err := ProduceFirstValueTreasuryBootstrap(root, WriterLockLease{LeaseHolderID: "holder-1"}, input, activatedAt); err != nil {
		t.Fatalf("ProduceFirstValueTreasuryBootstrap() error = %v", err)
	}

	treasury, err := LoadTreasuryRecord(root, input.TreasuryRef.TreasuryID)
	if err != nil {
		t.Fatalf("LoadTreasuryRecord() error = %v", err)
	}
	if treasury.State != TreasuryStateActive {
		t.Fatalf("LoadTreasuryRecord().State = %q, want %q", treasury.State, TreasuryStateActive)
	}
	if !treasury.UpdatedAt.Equal(activatedAt.UTC()) {
		t.Fatalf("LoadTreasuryRecord().UpdatedAt = %s, want %s", treasury.UpdatedAt, activatedAt.UTC())
	}

	entries, err := ListTreasuryLedgerEntries(root, input.TreasuryRef.TreasuryID)
	if err != nil {
		t.Fatalf("ListTreasuryLedgerEntries() error = %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("ListTreasuryLedgerEntries() len = %d, want 1", len(entries))
	}
	if entries[0].EntryID != input.EntryID || entries[0].EntryKind != TreasuryLedgerEntryKindAcquisition {
		t.Fatalf("ListTreasuryLedgerEntries() = %#v, want one acquisition entry", entries)
	}
	if entries[0].AssetCode != "USD" || entries[0].Amount != "10.00" || entries[0].SourceRef != "payout:listing-1" {
		t.Fatalf("ListTreasuryLedgerEntries()[0] = %#v, want canonical recorded acquisition payload", entries[0])
	}
	if !entries[0].CreatedAt.Equal(recordedAt.UTC()) {
		t.Fatalf("ListTreasuryLedgerEntries()[0].CreatedAt = %s, want %s", entries[0].CreatedAt, recordedAt.UTC())
	}
}

func TestProduceFirstValueTreasuryBootstrapReplayFailsClosed(t *testing.T) {
	t.Parallel()

	fixtures := writeExecutionContextFrankRegistryFixtures(t)
	root := fixtures.root
	now := time.Date(2026, 4, 13, 13, 10, 0, 0, time.UTC)
	mustStoreTreasuryForMutationTest(t, root, validTreasuryRecord(now, func(record *TreasuryRecord) {
		record.TreasuryID = "treasury-bootstrap-producer-replay"
		record.State = TreasuryStateBootstrap
		record.ContainerRefs = []FrankRegistryObjectRef{{
			Kind:     FrankRegistryObjectKindContainer,
			ObjectID: fixtures.container.ContainerID,
		}}
	}))

	recordedAt := now.Add(2 * time.Minute)
	if err := RecordFirstTreasuryAcquisition(root, WriterLockLease{LeaseHolderID: "holder-1"}, FirstTreasuryAcquisitionInput{
		TreasuryID: "treasury-bootstrap-producer-replay",
		EntryID:    "entry-first-value",
		AssetCode:  "USD",
		Amount:     "10.00",
		SourceRef:  "payout:listing-1",
	}, recordedAt); err != nil {
		t.Fatalf("RecordFirstTreasuryAcquisition() error = %v", err)
	}

	input := FirstValueTreasuryBootstrapInput{
		TreasuryRef: TreasuryRef{TreasuryID: "treasury-bootstrap-producer-replay"},
		EntryID:     "entry-first-value",
	}
	if err := ProduceFirstValueTreasuryBootstrap(root, WriterLockLease{LeaseHolderID: "holder-1"}, input, recordedAt); err != nil {
		t.Fatalf("ProduceFirstValueTreasuryBootstrap(first) error = %v", err)
	}

	err := ProduceFirstValueTreasuryBootstrap(root, WriterLockLease{LeaseHolderID: "holder-1"}, input, now.Add(3*time.Minute))
	if err == nil {
		t.Fatal("ProduceFirstValueTreasuryBootstrap(replay) error = nil, want deterministic duplicate rejection")
	}
	if !strings.Contains(err.Error(), `mission store treasury "treasury-bootstrap-producer-replay" first-value bootstrap producer requires state "funded", got "active"`) {
		t.Fatalf("ProduceFirstValueTreasuryBootstrap(replay) error = %q, want active-state replay rejection", err.Error())
	}

	entries, err := ListTreasuryLedgerEntries(root, input.TreasuryRef.TreasuryID)
	if err != nil {
		t.Fatalf("ListTreasuryLedgerEntries() error = %v", err)
	}
	if len(entries) != 1 || entries[0].EntryID != input.EntryID {
		t.Fatalf("ListTreasuryLedgerEntries() = %#v, want one recorded first-value entry", entries)
	}
}

func TestProduceFirstValueTreasuryBootstrapFailsClosedWithoutActiveContainer(t *testing.T) {
	t.Parallel()

	root := writeExecutionContextFrankRegistryFixtures(t).root
	now := time.Date(2026, 4, 13, 13, 20, 0, 0, time.UTC)
	writeMalformedTreasuryRecordForMutationTest(t, root, validTreasuryRecord(now, func(record *TreasuryRecord) {
		record.TreasuryID = "treasury-bootstrap-producer-no-container"
		record.State = TreasuryStateFunded
		record.ContainerRefs = nil
	}))

	err := ProduceFirstValueTreasuryBootstrap(root, WriterLockLease{LeaseHolderID: "holder-1"}, FirstValueTreasuryBootstrapInput{
		TreasuryRef: TreasuryRef{TreasuryID: "treasury-bootstrap-producer-no-container"},
		EntryID:     "entry-first-value",
	}, now.Add(2*time.Minute))
	if err == nil {
		t.Fatal("ProduceFirstValueTreasuryBootstrap() error = nil, want missing active-container rejection")
	}
	if !strings.Contains(err.Error(), `mission store treasury state "funded" requires exactly one active_container_id derivable from container_refs`) {
		t.Fatalf("ProduceFirstValueTreasuryBootstrap() error = %q, want missing active-container rejection", err.Error())
	}

	assertNoTreasuryLedgerEntries(t, root, "treasury-bootstrap-producer-no-container")
}

func TestProduceFirstValueTreasuryBootstrapFailsClosedWithAmbiguousActiveContainer(t *testing.T) {
	t.Parallel()

	fixtures := writeExecutionContextFrankRegistryFixtures(t)
	root := fixtures.root
	now := time.Date(2026, 4, 13, 13, 30, 0, 0, time.UTC)
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
		record.TreasuryID = "treasury-bootstrap-producer-ambiguous"
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

	err := ProduceFirstValueTreasuryBootstrap(root, WriterLockLease{LeaseHolderID: "holder-1"}, FirstValueTreasuryBootstrapInput{
		TreasuryRef: TreasuryRef{TreasuryID: "treasury-bootstrap-producer-ambiguous"},
		EntryID:     "entry-first-value",
	}, now.Add(3*time.Minute))
	if err == nil {
		t.Fatal("ProduceFirstValueTreasuryBootstrap() error = nil, want ambiguous active-container rejection")
	}
	if !strings.Contains(err.Error(), `mission store treasury state "funded" requires exactly one active_container_id derivable from container_refs`) {
		t.Fatalf("ProduceFirstValueTreasuryBootstrap() error = %q, want ambiguous active-container rejection", err.Error())
	}

	assertNoTreasuryLedgerEntries(t, root, "treasury-bootstrap-producer-ambiguous")
}

func TestProduceFirstValueTreasuryBootstrapFailsClosedOutsideFundedState(t *testing.T) {
	t.Parallel()

	fixtures := writeExecutionContextFrankRegistryFixtures(t)
	root := fixtures.root
	now := time.Date(2026, 4, 13, 13, 40, 0, 0, time.UTC)
	mustStoreTreasuryForMutationTest(t, root, validTreasuryRecord(now, func(record *TreasuryRecord) {
		record.TreasuryID = "treasury-bootstrap-producer-bootstrap"
		record.State = TreasuryStateBootstrap
		record.ContainerRefs = []FrankRegistryObjectRef{{
			Kind:     FrankRegistryObjectKindContainer,
			ObjectID: fixtures.container.ContainerID,
		}}
	}))

	err := ProduceFirstValueTreasuryBootstrap(root, WriterLockLease{LeaseHolderID: "holder-1"}, FirstValueTreasuryBootstrapInput{
		TreasuryRef: TreasuryRef{TreasuryID: "treasury-bootstrap-producer-bootstrap"},
		EntryID:     "entry-first-value",
	}, now.Add(2*time.Minute))
	if err == nil {
		t.Fatal("ProduceFirstValueTreasuryBootstrap() error = nil, want invalid bootstrap-state rejection")
	}
	if !strings.Contains(err.Error(), `mission store treasury "treasury-bootstrap-producer-bootstrap" first-value bootstrap producer requires state "funded", got "bootstrap"`) {
		t.Fatalf("ProduceFirstValueTreasuryBootstrap() error = %q, want invalid bootstrap-state rejection", err.Error())
	}

	assertNoTreasuryLedgerEntries(t, root, "treasury-bootstrap-producer-bootstrap")
}

func TestProduceFirstValueTreasuryBootstrapFailsClosedWithoutCommittedAcquisitionEntry(t *testing.T) {
	t.Parallel()

	fixtures := writeExecutionContextFrankRegistryFixtures(t)
	root := fixtures.root
	now := time.Date(2026, 4, 13, 13, 45, 0, 0, time.UTC)
	mustStoreTreasuryForMutationTest(t, root, validTreasuryRecord(now, func(record *TreasuryRecord) {
		record.TreasuryID = "treasury-bootstrap-producer-missing-entry"
		record.State = TreasuryStateFunded
		record.ContainerRefs = []FrankRegistryObjectRef{{
			Kind:     FrankRegistryObjectKindContainer,
			ObjectID: fixtures.container.ContainerID,
		}}
	}))

	err := ProduceFirstValueTreasuryBootstrap(root, WriterLockLease{LeaseHolderID: "holder-1"}, FirstValueTreasuryBootstrapInput{
		TreasuryRef: TreasuryRef{TreasuryID: "treasury-bootstrap-producer-missing-entry"},
		EntryID:     "entry-first-value",
	}, now.Add(2*time.Minute))
	if err == nil {
		t.Fatal("ProduceFirstValueTreasuryBootstrap() error = nil, want committed acquisition rejection")
	}
	if !strings.Contains(err.Error(), `mission store treasury "treasury-bootstrap-producer-missing-entry" first-value bootstrap producer requires committed acquisition ledger entry "entry-first-value"`) {
		t.Fatalf("ProduceFirstValueTreasuryBootstrap() error = %q, want committed acquisition rejection", err.Error())
	}
}

func TestProduceFirstValueTreasuryBootstrapPreservesInspectTreasuryPreflightContract(t *testing.T) {
	t.Parallel()

	fixtures := writeExecutionContextFrankRegistryFixtures(t)
	root := fixtures.root
	now := time.Date(2026, 4, 13, 13, 50, 0, 0, time.UTC)
	mustStoreTreasuryForMutationTest(t, root, validTreasuryRecord(now, func(record *TreasuryRecord) {
		record.TreasuryID = "treasury-bootstrap-producer-readout"
		record.State = TreasuryStateBootstrap
		record.ContainerRefs = []FrankRegistryObjectRef{{
			Kind:     FrankRegistryObjectKindContainer,
			ObjectID: fixtures.container.ContainerID,
		}}
	}))

	recordedAt := now.Add(time.Minute)
	if err := RecordFirstTreasuryAcquisition(root, WriterLockLease{LeaseHolderID: "holder-1"}, FirstTreasuryAcquisitionInput{
		TreasuryID: "treasury-bootstrap-producer-readout",
		EntryID:    "entry-first-value",
		AssetCode:  "USD",
		Amount:     "10.00",
		SourceRef:  "payout:listing-1",
	}, recordedAt); err != nil {
		t.Fatalf("RecordFirstTreasuryAcquisition() error = %v", err)
	}

	if err := ProduceFirstValueTreasuryBootstrap(root, WriterLockLease{LeaseHolderID: "holder-1"}, FirstValueTreasuryBootstrapInput{
		TreasuryRef: TreasuryRef{TreasuryID: "treasury-bootstrap-producer-readout"},
		EntryID:     "entry-first-value",
	}, now.Add(2*time.Minute)); err != nil {
		t.Fatalf("ProduceFirstValueTreasuryBootstrap() error = %v", err)
	}

	job := testExecutionJob()
	job.Plan.Steps[0].TreasuryRef = &TreasuryRef{TreasuryID: "treasury-bootstrap-producer-readout"}
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
