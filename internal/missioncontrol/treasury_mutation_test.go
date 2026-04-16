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
	mustStoreTreasuryForMutationTest(t, root, validTreasuryRecord(now, func(record *TreasuryRecord) {
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
	mustStoreTreasuryForMutationTest(t, root, validTreasuryRecord(now, func(record *TreasuryRecord) {
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
	mustStoreTreasuryForMutationTest(t, root, validTreasuryRecord(now, func(record *TreasuryRecord) {
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
	mustStoreTreasuryForMutationTest(t, root, validTreasuryRecord(now.Add(2*time.Minute), func(record *TreasuryRecord) {
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
	mustStoreTreasuryForMutationTest(t, root, validTreasuryRecord(now, func(record *TreasuryRecord) {
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
	mustStoreTreasuryForMutationTest(t, root, validTreasuryRecord(now, func(record *TreasuryRecord) {
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

func TestActivateFundedTreasuryTransitionsFundedTreasuryToActive(t *testing.T) {
	t.Parallel()

	fixtures := writeExecutionContextFrankRegistryFixtures(t)
	root := fixtures.root
	now := time.Date(2026, 4, 10, 14, 0, 0, 0, time.UTC)
	mustStoreTreasuryForMutationTest(t, root, validTreasuryRecord(now, func(record *TreasuryRecord) {
		record.TreasuryID = "treasury-funded"
		record.State = TreasuryStateFunded
		record.ContainerRefs = []FrankRegistryObjectRef{{
			Kind:     FrankRegistryObjectKindContainer,
			ObjectID: fixtures.container.ContainerID,
		}}
	}))

	activatedAt := now.Add(2 * time.Minute)
	if err := ActivateFundedTreasury(root, WriterLockLease{LeaseHolderID: "holder-1"}, ActivateFundedTreasuryInput{
		TreasuryID: "treasury-funded",
	}, activatedAt); err != nil {
		t.Fatalf("ActivateFundedTreasury() error = %v", err)
	}

	treasury, err := LoadTreasuryRecord(root, "treasury-funded")
	if err != nil {
		t.Fatalf("LoadTreasuryRecord() error = %v", err)
	}
	if treasury.State != TreasuryStateActive {
		t.Fatalf("LoadTreasuryRecord().State = %q, want %q", treasury.State, TreasuryStateActive)
	}
	if !treasury.UpdatedAt.Equal(activatedAt.UTC()) {
		t.Fatalf("LoadTreasuryRecord().UpdatedAt = %s, want %s", treasury.UpdatedAt, activatedAt.UTC())
	}

	assertNoTreasuryLedgerEntries(t, root, "treasury-funded")
}

func TestActivateFundedTreasuryDuplicateActivationFailsClosed(t *testing.T) {
	t.Parallel()

	fixtures := writeExecutionContextFrankRegistryFixtures(t)
	root := fixtures.root
	now := time.Date(2026, 4, 10, 14, 10, 0, 0, time.UTC)
	mustStoreTreasuryForMutationTest(t, root, validTreasuryRecord(now, func(record *TreasuryRecord) {
		record.TreasuryID = "treasury-duplicate-activation"
		record.State = TreasuryStateFunded
		record.ContainerRefs = []FrankRegistryObjectRef{{
			Kind:     FrankRegistryObjectKindContainer,
			ObjectID: fixtures.container.ContainerID,
		}}
	}))

	activeAt := now.Add(time.Minute)
	if err := ActivateFundedTreasury(root, WriterLockLease{LeaseHolderID: "holder-1"}, ActivateFundedTreasuryInput{
		TreasuryID: "treasury-duplicate-activation",
	}, activeAt); err != nil {
		t.Fatalf("ActivateFundedTreasury(first) error = %v", err)
	}

	err := ActivateFundedTreasury(root, WriterLockLease{LeaseHolderID: "holder-1"}, ActivateFundedTreasuryInput{
		TreasuryID: "treasury-duplicate-activation",
	}, now.Add(2*time.Minute))
	if err == nil {
		t.Fatal("ActivateFundedTreasury() error = nil, want duplicate activation rejection")
	}
	if !strings.Contains(err.Error(), `mission store treasury "treasury-duplicate-activation" activation requires state "funded", got "active"`) {
		t.Fatalf("ActivateFundedTreasury() error = %q, want duplicate activation rejection", err.Error())
	}

	treasury, loadErr := LoadTreasuryRecord(root, "treasury-duplicate-activation")
	if loadErr != nil {
		t.Fatalf("LoadTreasuryRecord() error = %v", loadErr)
	}
	if treasury.State != TreasuryStateActive {
		t.Fatalf("LoadTreasuryRecord().State = %q, want %q", treasury.State, TreasuryStateActive)
	}
	if !treasury.UpdatedAt.Equal(activeAt.UTC()) {
		t.Fatalf("LoadTreasuryRecord().UpdatedAt = %s, want unchanged %s", treasury.UpdatedAt, activeAt.UTC())
	}

	assertNoTreasuryLedgerEntries(t, root, "treasury-duplicate-activation")
}

func TestActivateFundedTreasuryFailsClosedWithoutActiveContainer(t *testing.T) {
	t.Parallel()

	root := writeExecutionContextFrankRegistryFixtures(t).root
	now := time.Date(2026, 4, 10, 14, 20, 0, 0, time.UTC)
	writeMalformedTreasuryRecordForMutationTest(t, root, validTreasuryRecord(now, func(record *TreasuryRecord) {
		record.TreasuryID = "treasury-no-active-container"
		record.State = TreasuryStateFunded
		record.ContainerRefs = nil
	}))

	err := ActivateFundedTreasury(root, WriterLockLease{LeaseHolderID: "holder-1"}, ActivateFundedTreasuryInput{
		TreasuryID: "treasury-no-active-container",
	}, now.Add(2*time.Minute))
	if err == nil {
		t.Fatal("ActivateFundedTreasury() error = nil, want missing active-container rejection")
	}
	if !strings.Contains(err.Error(), `mission store treasury state "funded" requires exactly one active_container_id derivable from container_refs`) {
		t.Fatalf("ActivateFundedTreasury() error = %q, want missing active-container rejection", err.Error())
	}

	assertNoTreasuryLedgerEntries(t, root, "treasury-no-active-container")
}

func TestActivateFundedTreasuryFailsClosedWithAmbiguousActiveContainer(t *testing.T) {
	t.Parallel()

	fixtures := writeExecutionContextFrankRegistryFixtures(t)
	root := fixtures.root
	now := time.Date(2026, 4, 10, 14, 30, 0, 0, time.UTC)
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
		record.TreasuryID = "treasury-ambiguous-activation"
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

	err := ActivateFundedTreasury(root, WriterLockLease{LeaseHolderID: "holder-1"}, ActivateFundedTreasuryInput{
		TreasuryID: "treasury-ambiguous-activation",
	}, now.Add(3*time.Minute))
	if err == nil {
		t.Fatal("ActivateFundedTreasury() error = nil, want ambiguous active-container rejection")
	}
	if !strings.Contains(err.Error(), `mission store treasury state "funded" requires exactly one active_container_id derivable from container_refs`) {
		t.Fatalf("ActivateFundedTreasury() error = %q, want ambiguous active-container rejection", err.Error())
	}

	assertNoTreasuryLedgerEntries(t, root, "treasury-ambiguous-activation")
}

func TestActivateFundedTreasuryFailsClosedOutsideFundedState(t *testing.T) {
	t.Parallel()

	fixtures := writeExecutionContextFrankRegistryFixtures(t)
	root := fixtures.root
	now := time.Date(2026, 4, 10, 14, 40, 0, 0, time.UTC)
	mustStoreTreasuryForMutationTest(t, root, validTreasuryRecord(now, func(record *TreasuryRecord) {
		record.TreasuryID = "treasury-bootstrap-activation"
		record.State = TreasuryStateBootstrap
		record.ContainerRefs = []FrankRegistryObjectRef{{
			Kind:     FrankRegistryObjectKindContainer,
			ObjectID: fixtures.container.ContainerID,
		}}
	}))

	err := ActivateFundedTreasury(root, WriterLockLease{LeaseHolderID: "holder-1"}, ActivateFundedTreasuryInput{
		TreasuryID: "treasury-bootstrap-activation",
	}, now.Add(2*time.Minute))
	if err == nil {
		t.Fatal("ActivateFundedTreasury() error = nil, want invalid state rejection")
	}
	if !strings.Contains(err.Error(), `mission store treasury "treasury-bootstrap-activation" activation requires state "funded", got "bootstrap"`) {
		t.Fatalf("ActivateFundedTreasury() error = %q, want invalid state rejection", err.Error())
	}

	assertNoTreasuryLedgerEntries(t, root, "treasury-bootstrap-activation")
}

func TestActivateFundedTreasuryPreservesTreasuryPreflightAndActivePolicyReadModel(t *testing.T) {
	t.Parallel()

	fixtures := writeExecutionContextFrankRegistryFixtures(t)
	root := fixtures.root
	now := time.Date(2026, 4, 10, 14, 50, 0, 0, time.UTC)
	mustStoreTreasuryForMutationTest(t, root, validTreasuryRecord(now, func(record *TreasuryRecord) {
		record.TreasuryID = "treasury-read-model"
		record.State = TreasuryStateFunded
		record.ContainerRefs = []FrankRegistryObjectRef{{
			Kind:     FrankRegistryObjectKindContainer,
			ObjectID: fixtures.container.ContainerID,
		}}
	}))

	activatedAt := now.Add(2 * time.Minute)
	if err := ActivateFundedTreasury(root, WriterLockLease{LeaseHolderID: "holder-1"}, ActivateFundedTreasuryInput{
		TreasuryID: "treasury-read-model",
	}, activatedAt); err != nil {
		t.Fatalf("ActivateFundedTreasury() error = %v", err)
	}

	job := testExecutionJob()
	job.Plan.Steps[0].TreasuryRef = &TreasuryRef{TreasuryID: "treasury-read-model"}
	ec, err := ResolveExecutionContext(job, "build")
	if err != nil {
		t.Fatalf("ResolveExecutionContext() error = %v", err)
	}
	ec.MissionStoreRoot = root

	preflight, err := ResolveExecutionContextTreasuryPreflight(ec)
	if err != nil {
		t.Fatalf("ResolveExecutionContextTreasuryPreflight() error = %v", err)
	}
	if preflight.Treasury == nil {
		t.Fatal("ResolveExecutionContextTreasuryPreflight().Treasury = nil, want activated treasury")
	}
	if len(preflight.Containers) != 1 || !reflect.DeepEqual(preflight.Containers[0], fixtures.container) {
		t.Fatalf("ResolveExecutionContextTreasuryPreflight().Containers = %#v, want [%#v]", preflight.Containers, fixtures.container)
	}

	view := preflight.Treasury.AsObjectView()
	wantPermitted, wantForbidden := DefaultTreasuryTransactionPolicy(TreasuryStateActive)
	if preflight.Treasury.State != TreasuryStateActive {
		t.Fatalf("ResolveExecutionContextTreasuryPreflight().Treasury.State = %q, want %q", preflight.Treasury.State, TreasuryStateActive)
	}
	if !reflect.DeepEqual(view.PermittedTransactionClasses, wantPermitted) {
		t.Fatalf("TreasuryRecord.AsObjectView().PermittedTransactionClasses = %#v, want %#v", view.PermittedTransactionClasses, wantPermitted)
	}
	if !reflect.DeepEqual(view.ForbiddenTransactionClasses, wantForbidden) {
		t.Fatalf("TreasuryRecord.AsObjectView().ForbiddenTransactionClasses = %#v, want %#v", view.ForbiddenTransactionClasses, wantForbidden)
	}
	if view.ActiveContainerID != fixtures.container.ContainerID {
		t.Fatalf("TreasuryRecord.AsObjectView().ActiveContainerID = %q, want %q", view.ActiveContainerID, fixtures.container.ContainerID)
	}

	assertNoTreasuryLedgerEntries(t, root, "treasury-read-model")
}

func TestRecordPostBootstrapTreasuryAcquisitionAppendsLedgerEntryAndConsumesCommittedBlock(t *testing.T) {
	t.Parallel()

	fixtures := writeExecutionContextFrankRegistryFixtures(t)
	root := fixtures.root
	now := time.Date(2026, 4, 10, 15, 10, 0, 0, time.UTC)
	mustStoreTreasuryForMutationTest(t, root, validTreasuryRecord(now, func(record *TreasuryRecord) {
		record.TreasuryID = "treasury-post-bootstrap"
		record.State = TreasuryStateActive
		record.ContainerRefs = []FrankRegistryObjectRef{{
			Kind:     FrankRegistryObjectKindContainer,
			ObjectID: fixtures.container.ContainerID,
		}}
		record.PostBootstrapAcquisition = &TreasuryPostBootstrapAcquisition{
			AssetCode:       "USD",
			Amount:          "2.25",
			SourceRef:       "payout:listing-2",
			EvidenceLocator: "https://evidence.example/payout-2",
			ConfirmedAt:     now.Add(time.Minute),
		}
	}))

	recordedAt := now.Add(2 * time.Minute)
	if err := RecordPostBootstrapTreasuryAcquisition(root, WriterLockLease{LeaseHolderID: "holder-1"}, PostBootstrapTreasuryAcquisitionInput{
		TreasuryID: "treasury-post-bootstrap",
	}, recordedAt); err != nil {
		t.Fatalf("RecordPostBootstrapTreasuryAcquisition() error = %v", err)
	}

	treasury, err := LoadTreasuryRecord(root, "treasury-post-bootstrap")
	if err != nil {
		t.Fatalf("LoadTreasuryRecord() error = %v", err)
	}
	if treasury.State != TreasuryStateActive {
		t.Fatalf("LoadTreasuryRecord().State = %q, want %q", treasury.State, TreasuryStateActive)
	}
	if treasury.PostBootstrapAcquisition == nil {
		t.Fatal("LoadTreasuryRecord().PostBootstrapAcquisition = nil, want consumed block")
	}
	if !treasury.UpdatedAt.Equal(recordedAt.UTC()) {
		t.Fatalf("LoadTreasuryRecord().UpdatedAt = %s, want %s", treasury.UpdatedAt, recordedAt.UTC())
	}
	entryID := treasury.PostBootstrapAcquisition.ConsumedEntryID
	if entryID == "" {
		t.Fatal("LoadTreasuryRecord().PostBootstrapAcquisition.ConsumedEntryID = empty, want committed linkage")
	}

	entry, err := LoadTreasuryLedgerEntry(root, "treasury-post-bootstrap", entryID)
	if err != nil {
		t.Fatalf("LoadTreasuryLedgerEntry() error = %v", err)
	}
	if entry.EntryKind != TreasuryLedgerEntryKindAcquisition {
		t.Fatalf("LoadTreasuryLedgerEntry().EntryKind = %q, want %q", entry.EntryKind, TreasuryLedgerEntryKindAcquisition)
	}
	if entry.AssetCode != "USD" || entry.Amount != "2.25" || entry.SourceRef != "payout:listing-2" {
		t.Fatalf("LoadTreasuryLedgerEntry() = %#v, want committed post-bootstrap acquisition payload", entry)
	}
	if !entry.CreatedAt.Equal(recordedAt.UTC()) {
		t.Fatalf("LoadTreasuryLedgerEntry().CreatedAt = %s, want %s", entry.CreatedAt, recordedAt.UTC())
	}
}

func TestRecordPostBootstrapTreasuryAcquisitionReplayFailsClosedAfterCommittedConsumption(t *testing.T) {
	t.Parallel()

	fixtures := writeExecutionContextFrankRegistryFixtures(t)
	root := fixtures.root
	now := time.Date(2026, 4, 10, 15, 20, 0, 0, time.UTC)
	mustStoreTreasuryForMutationTest(t, root, validTreasuryRecord(now, func(record *TreasuryRecord) {
		record.TreasuryID = "treasury-post-bootstrap-replay"
		record.State = TreasuryStateActive
		record.ContainerRefs = []FrankRegistryObjectRef{{
			Kind:     FrankRegistryObjectKindContainer,
			ObjectID: fixtures.container.ContainerID,
		}}
		record.PostBootstrapAcquisition = &TreasuryPostBootstrapAcquisition{
			AssetCode:       "USD",
			Amount:          "3.00",
			SourceRef:       "payout:listing-3",
			EvidenceLocator: "https://evidence.example/payout-3",
			ConfirmedAt:     now.Add(time.Minute),
		}
	}))

	input := PostBootstrapTreasuryAcquisitionInput{TreasuryID: "treasury-post-bootstrap-replay"}
	if err := RecordPostBootstrapTreasuryAcquisition(root, WriterLockLease{LeaseHolderID: "holder-1"}, input, now.Add(2*time.Minute)); err != nil {
		t.Fatalf("RecordPostBootstrapTreasuryAcquisition(first) error = %v", err)
	}

	err := RecordPostBootstrapTreasuryAcquisition(root, WriterLockLease{LeaseHolderID: "holder-1"}, input, now.Add(3*time.Minute))
	if err == nil {
		t.Fatal("RecordPostBootstrapTreasuryAcquisition(replay) error = nil, want deterministic consumed rejection")
	}
	if !strings.Contains(err.Error(), `mission store treasury "treasury-post-bootstrap-replay" post-bootstrap acquisition already consumed by entry "`) {
		t.Fatalf("RecordPostBootstrapTreasuryAcquisition(replay) error = %q, want consumed rejection", err.Error())
	}

	entries, err := ListTreasuryLedgerEntries(root, "treasury-post-bootstrap-replay")
	if err != nil {
		t.Fatalf("ListTreasuryLedgerEntries() error = %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("ListTreasuryLedgerEntries() len = %d, want 1", len(entries))
	}
}

func TestRecordPostBootstrapTreasuryAcquisitionFailsClosedOnDerivedEntryAmbiguity(t *testing.T) {
	t.Parallel()

	fixtures := writeExecutionContextFrankRegistryFixtures(t)
	root := fixtures.root
	now := time.Date(2026, 4, 10, 15, 30, 0, 0, time.UTC)
	block := TreasuryPostBootstrapAcquisition{
		AssetCode:       "USD",
		Amount:          "4.00",
		SourceRef:       "payout:listing-4",
		EvidenceLocator: "https://evidence.example/payout-4",
		ConfirmedAt:     now.Add(time.Minute),
	}
	mustStoreTreasuryForMutationTest(t, root, validTreasuryRecord(now, func(record *TreasuryRecord) {
		record.TreasuryID = "treasury-post-bootstrap-ambiguous"
		record.State = TreasuryStateActive
		record.ContainerRefs = []FrankRegistryObjectRef{{
			Kind:     FrankRegistryObjectKindContainer,
			ObjectID: fixtures.container.ContainerID,
		}}
		record.PostBootstrapAcquisition = &block
	}))
	derivedEntryID := derivePostBootstrapTreasuryAcquisitionEntryID("treasury-post-bootstrap-ambiguous", block)
	if err := StoreTreasuryLedgerEntry(root, validTreasuryLedgerEntry(now.Add(2*time.Minute), func(entry *TreasuryLedgerEntry) {
		entry.EntryID = derivedEntryID
		entry.TreasuryID = "treasury-post-bootstrap-ambiguous"
		entry.AssetCode = block.AssetCode
		entry.Amount = block.Amount
		entry.SourceRef = block.SourceRef
	})); err != nil {
		t.Fatalf("StoreTreasuryLedgerEntry() error = %v", err)
	}

	err := RecordPostBootstrapTreasuryAcquisition(root, WriterLockLease{LeaseHolderID: "holder-1"}, PostBootstrapTreasuryAcquisitionInput{
		TreasuryID: "treasury-post-bootstrap-ambiguous",
	}, now.Add(3*time.Minute))
	if err == nil {
		t.Fatal("RecordPostBootstrapTreasuryAcquisition() error = nil, want ambiguous existing-entry rejection")
	}
	if !strings.Contains(err.Error(), `mission store treasury "treasury-post-bootstrap-ambiguous" post-bootstrap acquisition derived entry "`) ||
		!strings.Contains(err.Error(), `already exists without committed consumed_entry_id`) {
		t.Fatalf("RecordPostBootstrapTreasuryAcquisition() error = %q, want ambiguous derived-entry rejection", err.Error())
	}

	treasury, err := LoadTreasuryRecord(root, "treasury-post-bootstrap-ambiguous")
	if err != nil {
		t.Fatalf("LoadTreasuryRecord() error = %v", err)
	}
	if treasury.PostBootstrapAcquisition == nil || treasury.PostBootstrapAcquisition.ConsumedEntryID != "" {
		t.Fatalf("LoadTreasuryRecord().PostBootstrapAcquisition = %#v, want unconsumed block preserved", treasury.PostBootstrapAcquisition)
	}
}

func TestRecordPostBootstrapTreasuryAcquisitionFailsClosedWithoutCommittedBlock(t *testing.T) {
	t.Parallel()

	fixtures := writeExecutionContextFrankRegistryFixtures(t)
	root := fixtures.root
	now := time.Date(2026, 4, 10, 15, 40, 0, 0, time.UTC)
	mustStoreTreasuryForMutationTest(t, root, validTreasuryRecord(now, func(record *TreasuryRecord) {
		record.TreasuryID = "treasury-post-bootstrap-missing-block"
		record.State = TreasuryStateActive
		record.ContainerRefs = []FrankRegistryObjectRef{{
			Kind:     FrankRegistryObjectKindContainer,
			ObjectID: fixtures.container.ContainerID,
		}}
		record.PostBootstrapAcquisition = nil
	}))

	err := RecordPostBootstrapTreasuryAcquisition(root, WriterLockLease{LeaseHolderID: "holder-1"}, PostBootstrapTreasuryAcquisitionInput{
		TreasuryID: "treasury-post-bootstrap-missing-block",
	}, now.Add(2*time.Minute))
	if err == nil {
		t.Fatal("RecordPostBootstrapTreasuryAcquisition() error = nil, want missing block rejection")
	}
	if !strings.Contains(err.Error(), `mission store treasury "treasury-post-bootstrap-missing-block" post-bootstrap acquisition requires committed treasury.post_bootstrap_acquisition`) {
		t.Fatalf("RecordPostBootstrapTreasuryAcquisition() error = %q, want missing block rejection", err.Error())
	}

	assertNoTreasuryLedgerEntries(t, root, "treasury-post-bootstrap-missing-block")
}

func TestRecordPostBootstrapTreasuryAcquisitionFailsClosedOutsideActiveState(t *testing.T) {
	t.Parallel()

	fixtures := writeExecutionContextFrankRegistryFixtures(t)
	root := fixtures.root
	now := time.Date(2026, 4, 10, 15, 50, 0, 0, time.UTC)
	mustStoreTreasuryForMutationTest(t, root, validTreasuryRecord(now, func(record *TreasuryRecord) {
		record.TreasuryID = "treasury-post-bootstrap-funded"
		record.State = TreasuryStateFunded
		record.ContainerRefs = []FrankRegistryObjectRef{{
			Kind:     FrankRegistryObjectKindContainer,
			ObjectID: fixtures.container.ContainerID,
		}}
		record.PostBootstrapAcquisition = nil
	}))

	err := RecordPostBootstrapTreasuryAcquisition(root, WriterLockLease{LeaseHolderID: "holder-1"}, PostBootstrapTreasuryAcquisitionInput{
		TreasuryID: "treasury-post-bootstrap-funded",
	}, now.Add(2*time.Minute))
	if err == nil {
		t.Fatal("RecordPostBootstrapTreasuryAcquisition() error = nil, want invalid state rejection")
	}
	if !strings.Contains(err.Error(), `mission store treasury "treasury-post-bootstrap-funded" post-bootstrap acquisition requires state "active", got "funded"`) {
		t.Fatalf("RecordPostBootstrapTreasuryAcquisition() error = %q, want invalid state rejection", err.Error())
	}

	assertNoTreasuryLedgerEntries(t, root, "treasury-post-bootstrap-funded")
}

func mustStoreTreasuryForMutationTest(t *testing.T, root string, record TreasuryRecord) {
	t.Helper()

	if err := StoreTreasuryRecord(root, record); err != nil {
		t.Fatalf("StoreTreasuryRecord() error = %v", err)
	}
}

func writeMalformedTreasuryRecordForMutationTest(t *testing.T, root string, treasury TreasuryRecord) {
	t.Helper()

	treasury = normalizeTreasuryRecord(treasury)
	payload := map[string]any{
		"record_version":   normalizeRecordVersion(treasury.RecordVersion),
		"treasury_id":      treasury.TreasuryID,
		"display_name":     treasury.DisplayName,
		"state":            string(treasury.State),
		"zero_seed_policy": string(treasury.ZeroSeedPolicy),
		"created_at":       treasury.CreatedAt,
		"updated_at":       treasury.UpdatedAt,
	}
	if len(treasury.ContainerRefs) > 0 {
		refs := make([]map[string]any, 0, len(treasury.ContainerRefs))
		for _, ref := range treasury.ContainerRefs {
			refs = append(refs, map[string]any{
				"kind":      string(ref.Kind),
				"object_id": ref.ObjectID,
			})
		}
		payload["container_refs"] = refs
	}
	if err := WriteStoreJSONAtomic(StoreTreasuryPath(root, treasury.TreasuryID), payload); err != nil {
		t.Fatalf("WriteStoreJSONAtomic() error = %v", err)
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
