package missioncontrol

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestRecordFirstTreasuryAcquisitionTransitionsBootstrapTreasuryToFundedAndAppendsLedgerEntry(t *testing.T) {
	t.Parallel()

	fixtures := writeExecutionContextFrankRegistryFixtures(t)
	root := fixtures.root
	now := time.Date(2026, 4, 10, 13, 0, 0, 0, time.UTC)
	mustStoreTreasuryForFirstAcquisitionTest(t, root, validTreasuryRecord(now, func(record *TreasuryRecord) {
		record.TreasuryID = "treasury-bootstrap"
		record.ContainerRefs = []FrankRegistryObjectRef{{
			Kind:     FrankRegistryObjectKindContainer,
			ObjectID: fixtures.container.ContainerID,
		}}
	}))

	input := FirstTreasuryAcquisitionInput{
		TreasuryID: "treasury-bootstrap",
		EntryID:    "entry-first-value",
		AssetCode:  "USD",
		Amount:     "10.00",
		SourceRef:  "payout:listing-1",
	}
	recordedAt := now.Add(2 * time.Minute)
	if err := RecordFirstTreasuryAcquisition(root, WriterLockLease{LeaseHolderID: "holder-1"}, input, recordedAt); err != nil {
		t.Fatalf("RecordFirstTreasuryAcquisition() error = %v", err)
	}

	treasury, err := LoadTreasuryRecord(root, input.TreasuryID)
	if err != nil {
		t.Fatalf("LoadTreasuryRecord() error = %v", err)
	}
	if treasury.State != TreasuryStateFunded {
		t.Fatalf("LoadTreasuryRecord().State = %q, want %q", treasury.State, TreasuryStateFunded)
	}
	if !treasury.UpdatedAt.Equal(recordedAt.UTC()) {
		t.Fatalf("LoadTreasuryRecord().UpdatedAt = %s, want %s", treasury.UpdatedAt, recordedAt.UTC())
	}

	entry, err := LoadTreasuryLedgerEntry(root, input.TreasuryID, input.EntryID)
	if err != nil {
		t.Fatalf("LoadTreasuryLedgerEntry() error = %v", err)
	}
	if entry.EntryKind != TreasuryLedgerEntryKindAcquisition {
		t.Fatalf("LoadTreasuryLedgerEntry().EntryKind = %q, want %q", entry.EntryKind, TreasuryLedgerEntryKindAcquisition)
	}
	if entry.AssetCode != input.AssetCode || entry.Amount != input.Amount || entry.SourceRef != input.SourceRef {
		t.Fatalf("LoadTreasuryLedgerEntry() = %#v, want recorded acquisition payload", entry)
	}
	if !entry.CreatedAt.Equal(recordedAt.UTC()) {
		t.Fatalf("LoadTreasuryLedgerEntry().CreatedAt = %s, want %s", entry.CreatedAt, recordedAt.UTC())
	}
}

func TestRecordFirstTreasuryAcquisitionDuplicateBootstrapTransitionFailsClosed(t *testing.T) {
	t.Parallel()

	fixtures := writeExecutionContextFrankRegistryFixtures(t)
	root := fixtures.root
	now := time.Date(2026, 4, 10, 13, 10, 0, 0, time.UTC)
	mustStoreTreasuryForFirstAcquisitionTest(t, root, validTreasuryRecord(now, func(record *TreasuryRecord) {
		record.TreasuryID = "treasury-duplicate"
		record.ContainerRefs = []FrankRegistryObjectRef{{
			Kind:     FrankRegistryObjectKindContainer,
			ObjectID: fixtures.container.ContainerID,
		}}
	}))
	if err := StoreTreasuryLedgerEntry(root, validTreasuryLedgerEntry(now.Add(time.Minute), func(entry *TreasuryLedgerEntry) {
		entry.EntryID = "entry-existing-acquisition"
		entry.TreasuryID = "treasury-duplicate"
	})); err != nil {
		t.Fatalf("StoreTreasuryLedgerEntry() error = %v", err)
	}

	err := RecordFirstTreasuryAcquisition(root, WriterLockLease{LeaseHolderID: "holder-1"}, FirstTreasuryAcquisitionInput{
		TreasuryID: "treasury-duplicate",
		EntryID:    "entry-second-acquisition",
		AssetCode:  "USD",
		Amount:     "20.00",
		SourceRef:  "payout:listing-2",
	}, now.Add(2*time.Minute))
	if err == nil {
		t.Fatal("RecordFirstTreasuryAcquisition() error = nil, want duplicate acquisition rejection")
	}
	if !strings.Contains(err.Error(), `mission store treasury "treasury-duplicate" already has recorded acquisition ledger entry "entry-existing-acquisition"`) {
		t.Fatalf("RecordFirstTreasuryAcquisition() error = %q, want duplicate acquisition rejection", err.Error())
	}

	treasury, err := LoadTreasuryRecord(root, "treasury-duplicate")
	if err != nil {
		t.Fatalf("LoadTreasuryRecord() error = %v", err)
	}
	if treasury.State != TreasuryStateBootstrap {
		t.Fatalf("LoadTreasuryRecord().State = %q, want %q", treasury.State, TreasuryStateBootstrap)
	}
	entries, err := ListTreasuryLedgerEntries(root, "treasury-duplicate")
	if err != nil {
		t.Fatalf("ListTreasuryLedgerEntries() error = %v", err)
	}
	if len(entries) != 1 || entries[0].EntryID != "entry-existing-acquisition" {
		t.Fatalf("ListTreasuryLedgerEntries() = %#v, want one existing acquisition entry", entries)
	}
}

func TestRecordFirstTreasuryAcquisitionFailsClosedWithoutActiveContainer(t *testing.T) {
	t.Parallel()

	root := writeExecutionContextFrankRegistryFixtures(t).root
	now := time.Date(2026, 4, 10, 13, 20, 0, 0, time.UTC)
	mustStoreTreasuryForFirstAcquisitionTest(t, root, validTreasuryRecord(now, func(record *TreasuryRecord) {
		record.TreasuryID = "treasury-no-container"
		record.ContainerRefs = nil
	}))

	err := RecordFirstTreasuryAcquisition(root, WriterLockLease{LeaseHolderID: "holder-1"}, FirstTreasuryAcquisitionInput{
		TreasuryID: "treasury-no-container",
		EntryID:    "entry-no-container",
		AssetCode:  "USD",
		Amount:     "10.00",
		SourceRef:  "payout:listing-1",
	}, now.Add(2*time.Minute))
	if err == nil {
		t.Fatal("RecordFirstTreasuryAcquisition() error = nil, want missing active-container rejection")
	}
	if !strings.Contains(err.Error(), `mission store treasury ledger entry treasury_id "treasury-no-container" has no active treasury container`) {
		t.Fatalf("RecordFirstTreasuryAcquisition() error = %q, want missing active-container rejection", err.Error())
	}

	assertNoTreasuryLedgerEntries(t, root, "treasury-no-container")
}

func TestRecordFirstTreasuryAcquisitionFailsClosedWithAmbiguousActiveContainer(t *testing.T) {
	t.Parallel()

	fixtures := writeExecutionContextFrankRegistryFixtures(t)
	root := fixtures.root
	now := time.Date(2026, 4, 10, 13, 30, 0, 0, time.UTC)
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
	mustStoreTreasuryForFirstAcquisitionTest(t, root, validTreasuryRecord(now.Add(2*time.Minute), func(record *TreasuryRecord) {
		record.TreasuryID = "treasury-ambiguous-container"
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

	err := RecordFirstTreasuryAcquisition(root, WriterLockLease{LeaseHolderID: "holder-1"}, FirstTreasuryAcquisitionInput{
		TreasuryID: "treasury-ambiguous-container",
		EntryID:    "entry-ambiguous-container",
		AssetCode:  "USD",
		Amount:     "10.00",
		SourceRef:  "payout:listing-1",
	}, now.Add(3*time.Minute))
	if err == nil {
		t.Fatal("RecordFirstTreasuryAcquisition() error = nil, want ambiguous active-container rejection")
	}
	if !strings.Contains(err.Error(), `mission store treasury ledger entry treasury_id "treasury-ambiguous-container" has ambiguous active treasury container across 2 container_refs`) {
		t.Fatalf("RecordFirstTreasuryAcquisition() error = %q, want ambiguous active-container rejection", err.Error())
	}

	assertNoTreasuryLedgerEntries(t, root, "treasury-ambiguous-container")
}

func TestRecordFirstTreasuryAcquisitionFailsClosedOutsideBootstrapState(t *testing.T) {
	t.Parallel()

	fixtures := writeExecutionContextFrankRegistryFixtures(t)
	root := fixtures.root
	now := time.Date(2026, 4, 10, 13, 40, 0, 0, time.UTC)
	mustStoreTreasuryForFirstAcquisitionTest(t, root, validTreasuryRecord(now, func(record *TreasuryRecord) {
		record.TreasuryID = "treasury-funded"
		record.State = TreasuryStateFunded
		record.ContainerRefs = []FrankRegistryObjectRef{{
			Kind:     FrankRegistryObjectKindContainer,
			ObjectID: fixtures.container.ContainerID,
		}}
	}))

	err := RecordFirstTreasuryAcquisition(root, WriterLockLease{LeaseHolderID: "holder-1"}, FirstTreasuryAcquisitionInput{
		TreasuryID: "treasury-funded",
		EntryID:    "entry-funded",
		AssetCode:  "USD",
		Amount:     "10.00",
		SourceRef:  "payout:listing-1",
	}, now.Add(2*time.Minute))
	if err == nil {
		t.Fatal("RecordFirstTreasuryAcquisition() error = nil, want invalid state rejection")
	}
	if !strings.Contains(err.Error(), `mission store treasury "treasury-funded" first acquisition requires state "bootstrap", got "funded"`) {
		t.Fatalf("RecordFirstTreasuryAcquisition() error = %q, want invalid state rejection", err.Error())
	}

	assertNoTreasuryLedgerEntries(t, root, "treasury-funded")
}

func TestRecordFirstTreasuryAcquisitionPreservesTreasuryAndLedgerContractViews(t *testing.T) {
	t.Parallel()

	fixtures := writeExecutionContextFrankRegistryFixtures(t)
	root := fixtures.root
	now := time.Date(2026, 4, 10, 13, 50, 0, 0, time.UTC)
	mustStoreTreasuryForFirstAcquisitionTest(t, root, validTreasuryRecord(now, func(record *TreasuryRecord) {
		record.TreasuryID = "treasury-contract-view"
		record.ContainerRefs = []FrankRegistryObjectRef{{
			Kind:     FrankRegistryObjectKindContainer,
			ObjectID: fixtures.container.ContainerID,
		}}
	}))

	input := FirstTreasuryAcquisitionInput{
		TreasuryID: "treasury-contract-view",
		EntryID:    "entry-contract-view",
		AssetCode:  "USD",
		Amount:     "10.00",
		SourceRef:  "payout:listing-1",
	}
	recordedAt := now.Add(2 * time.Minute)
	if err := RecordFirstTreasuryAcquisition(root, WriterLockLease{LeaseHolderID: "holder-1"}, input, recordedAt); err != nil {
		t.Fatalf("RecordFirstTreasuryAcquisition() error = %v", err)
	}

	treasury, err := LoadTreasuryRecord(root, input.TreasuryID)
	if err != nil {
		t.Fatalf("LoadTreasuryRecord() error = %v", err)
	}
	view := treasury.AsObjectView()
	wantPermitted, wantForbidden := DefaultTreasuryTransactionPolicy(TreasuryStateFunded)
	if view.ActiveContainerID != fixtures.container.ContainerID {
		t.Fatalf("TreasuryRecord.AsObjectView().ActiveContainerID = %q, want %q", view.ActiveContainerID, fixtures.container.ContainerID)
	}
	if !reflect.DeepEqual(view.PermittedTransactionClasses, wantPermitted) {
		t.Fatalf("TreasuryRecord.AsObjectView().PermittedTransactionClasses = %#v, want %#v", view.PermittedTransactionClasses, wantPermitted)
	}
	if !reflect.DeepEqual(view.ForbiddenTransactionClasses, wantForbidden) {
		t.Fatalf("TreasuryRecord.AsObjectView().ForbiddenTransactionClasses = %#v, want %#v", view.ForbiddenTransactionClasses, wantForbidden)
	}
	if view.LedgerRef != treasury.TreasuryID {
		t.Fatalf("TreasuryRecord.AsObjectView().LedgerRef = %q, want %q", view.LedgerRef, treasury.TreasuryID)
	}

	entry, err := LoadTreasuryLedgerEntry(root, input.TreasuryID, input.EntryID)
	if err != nil {
		t.Fatalf("LoadTreasuryLedgerEntry() error = %v", err)
	}
	entryView, err := ResolveTreasuryLedgerEntryObjectView(root, entry)
	if err != nil {
		t.Fatalf("ResolveTreasuryLedgerEntryObjectView() error = %v", err)
	}
	if entryView.ContainerID != fixtures.container.ContainerID || entryView.EntryClass != TreasuryLedgerEntryKindAcquisition {
		t.Fatalf("ResolveTreasuryLedgerEntryObjectView() = %#v, want canonical acquisition contract fields", entryView)
	}
	if entryView.Direction != TreasuryLedgerDirectionInflow || entryView.Status != TreasuryLedgerStatusRecorded {
		t.Fatalf("ResolveTreasuryLedgerEntryObjectView() = %#v, want inflow/recorded adapter contract", entryView)
	}
}

func mustStoreTreasuryForFirstAcquisitionTest(t *testing.T, root string, record TreasuryRecord) {
	t.Helper()

	if err := StoreTreasuryRecord(root, record); err != nil {
		t.Fatalf("StoreTreasuryRecord() error = %v", err)
	}
}

func assertNoTreasuryLedgerEntries(t *testing.T, root, treasuryID string) {
	t.Helper()

	entries, err := ListTreasuryLedgerEntries(root, treasuryID)
	if err != nil {
		t.Fatalf("ListTreasuryLedgerEntries() error = %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("ListTreasuryLedgerEntries() = %#v, want no entries", entries)
	}
}
