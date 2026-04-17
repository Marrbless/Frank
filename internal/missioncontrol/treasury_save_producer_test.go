package missioncontrol

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestProducePostActiveTreasurySaveSuccessfulExecution(t *testing.T) {
	t.Parallel()

	fixtures := writeExecutionContextFrankRegistryFixtures(t)
	root := fixtures.root
	now := time.Date(2026, 4, 14, 14, 0, 0, 0, time.UTC)
	targetContainer := storeTreasurySaveTargetContainerForTest(t, root, "container-savings-producer", "container-class-savings-producer", now)
	mustStoreTreasuryForMutationTest(t, root, validTreasuryRecord(now, func(record *TreasuryRecord) {
		record.TreasuryID = "treasury-save-producer"
		record.State = TreasuryStateActive
		record.ContainerRefs = []FrankRegistryObjectRef{{
			Kind:     FrankRegistryObjectKindContainer,
			ObjectID: fixtures.container.ContainerID,
		}}
		record.PostActiveSave = &TreasuryPostActiveSave{
			AssetCode:         "USD",
			Amount:            "1.25",
			TargetContainerID: targetContainer.ContainerID,
			SourceRef:         "transfer:reserve-a",
			EvidenceLocator:   "https://evidence.example/save-a",
		}
	}))

	if err := ProducePostActiveTreasurySave(root, WriterLockLease{LeaseHolderID: "holder-1"}, PostActiveTreasurySaveInput{
		TreasuryRef: TreasuryRef{TreasuryID: "treasury-save-producer"},
	}, now.Add(2*time.Minute)); err != nil {
		t.Fatalf("ProducePostActiveTreasurySave() error = %v", err)
	}

	treasury, err := LoadTreasuryRecord(root, "treasury-save-producer")
	if err != nil {
		t.Fatalf("LoadTreasuryRecord() error = %v", err)
	}
	if treasury.PostActiveSave == nil || treasury.PostActiveSave.ConsumedEntryID == "" {
		t.Fatalf("LoadTreasuryRecord().PostActiveSave = %#v, want consumed linkage", treasury.PostActiveSave)
	}
	entry, err := LoadTreasuryLedgerEntry(root, "treasury-save-producer", treasury.PostActiveSave.ConsumedEntryID)
	if err != nil {
		t.Fatalf("LoadTreasuryLedgerEntry() error = %v", err)
	}
	if entry.EntryKind != TreasuryLedgerEntryKindMovement {
		t.Fatalf("LoadTreasuryLedgerEntry().EntryKind = %q, want %q", entry.EntryKind, TreasuryLedgerEntryKindMovement)
	}
}

func TestProducePostActiveTreasurySaveReplayFailsClosed(t *testing.T) {
	t.Parallel()

	fixtures := writeExecutionContextFrankRegistryFixtures(t)
	root := fixtures.root
	now := time.Date(2026, 4, 14, 14, 10, 0, 0, time.UTC)
	targetContainer := storeTreasurySaveTargetContainerForTest(t, root, "container-savings-producer-replay", "container-class-savings-producer-replay", now)
	mustStoreTreasuryForMutationTest(t, root, validTreasuryRecord(now, func(record *TreasuryRecord) {
		record.TreasuryID = "treasury-save-producer-replay"
		record.State = TreasuryStateActive
		record.ContainerRefs = []FrankRegistryObjectRef{{
			Kind:     FrankRegistryObjectKindContainer,
			ObjectID: fixtures.container.ContainerID,
		}}
		record.PostActiveSave = &TreasuryPostActiveSave{
			AssetCode:         "USD",
			Amount:            "1.25",
			TargetContainerID: targetContainer.ContainerID,
			SourceRef:         "transfer:reserve-a",
		}
	}))

	input := PostActiveTreasurySaveInput{TreasuryRef: TreasuryRef{TreasuryID: "treasury-save-producer-replay"}}
	recordedAt := now.Add(2 * time.Minute)
	if err := ProducePostActiveTreasurySave(root, WriterLockLease{LeaseHolderID: "holder-1"}, input, recordedAt); err != nil {
		t.Fatalf("ProducePostActiveTreasurySave(first) error = %v", err)
	}

	err := ProducePostActiveTreasurySave(root, WriterLockLease{LeaseHolderID: "holder-1"}, input, now.Add(3*time.Minute))
	if err == nil {
		t.Fatal("ProducePostActiveTreasurySave(replay) error = nil, want deterministic duplicate rejection")
	}
	if !strings.Contains(err.Error(), `execution context treasury "treasury-save-producer-replay" treasury.post_active_save is already consumed by entry "`) {
		t.Fatalf("ProducePostActiveTreasurySave(replay) error = %q, want consumed replay rejection", err.Error())
	}

	treasury, err := LoadTreasuryRecord(root, input.TreasuryRef.TreasuryID)
	if err != nil {
		t.Fatalf("LoadTreasuryRecord() error = %v", err)
	}
	if treasury.PostActiveSave == nil || treasury.PostActiveSave.ConsumedEntryID == "" {
		t.Fatalf("LoadTreasuryRecord().PostActiveSave = %#v, want consumed linkage", treasury.PostActiveSave)
	}
}

func TestProducePostActiveTreasurySaveFailsClosedOnIneligibleTargetContainer(t *testing.T) {
	t.Parallel()

	fixtures := writeExecutionContextFrankRegistryFixtures(t)
	root := fixtures.root
	now := time.Date(2026, 4, 14, 14, 20, 0, 0, time.UTC)
	target := AutonomyEligibilityTargetRef{
		Kind:       EligibilityTargetKindTreasuryContainerClass,
		RegistryID: "container-class-savings-ineligible",
	}
	writeFrankRegistryEligibilityFixture(t, root, target, EligibilityLabelIneligible, "container-class-savings-ineligible", "check-container-class-savings-ineligible", now)
	targetContainer := FrankContainerRecord{
		RecordVersion:        StoreRecordVersion,
		ContainerID:          "container-savings-ineligible",
		ContainerKind:        "wallet",
		Label:                "Savings Wallet",
		ContainerClassID:     target.RegistryID,
		State:                "active",
		EligibilityTargetRef: target,
		CreatedAt:            now.UTC(),
		UpdatedAt:            now.Add(time.Minute).UTC(),
	}
	if err := StoreFrankContainerRecord(root, targetContainer); err != nil {
		t.Fatalf("StoreFrankContainerRecord() error = %v", err)
	}
	mustStoreTreasuryForMutationTest(t, root, validTreasuryRecord(now, func(record *TreasuryRecord) {
		record.TreasuryID = "treasury-save-producer-ineligible"
		record.State = TreasuryStateActive
		record.ContainerRefs = []FrankRegistryObjectRef{{
			Kind:     FrankRegistryObjectKindContainer,
			ObjectID: fixtures.container.ContainerID,
		}}
		record.PostActiveSave = &TreasuryPostActiveSave{
			AssetCode:         "USD",
			Amount:            "1.25",
			TargetContainerID: targetContainer.ContainerID,
			SourceRef:         "transfer:reserve-a",
		}
	}))

	err := ProducePostActiveTreasurySave(root, WriterLockLease{LeaseHolderID: "holder-1"}, PostActiveTreasurySaveInput{
		TreasuryRef: TreasuryRef{TreasuryID: "treasury-save-producer-ineligible"},
	}, now.Add(2*time.Minute))
	if err == nil {
		t.Fatal("ProducePostActiveTreasurySave() error = nil, want ineligible target rejection")
	}
	if !strings.Contains(err.Error(), target.RegistryID) {
		t.Fatalf("ProducePostActiveTreasurySave() error = %q, want target eligibility rejection", err.Error())
	}

	assertNoTreasuryLedgerEntries(t, root, "treasury-save-producer-ineligible")
}

func TestProducePostActiveTreasurySavePreservesInspectTreasuryPreflightContract(t *testing.T) {
	t.Parallel()

	fixtures := writeExecutionContextFrankRegistryFixtures(t)
	root := fixtures.root
	now := time.Date(2026, 4, 14, 14, 30, 0, 0, time.UTC)
	targetContainer := storeTreasurySaveTargetContainerForTest(t, root, "container-savings-producer-readout", "container-class-savings-producer-readout", now)
	mustStoreTreasuryForMutationTest(t, root, validTreasuryRecord(now, func(record *TreasuryRecord) {
		record.TreasuryID = "treasury-save-producer-readout"
		record.State = TreasuryStateActive
		record.ContainerRefs = []FrankRegistryObjectRef{{
			Kind:     FrankRegistryObjectKindContainer,
			ObjectID: fixtures.container.ContainerID,
		}}
		record.PostActiveSave = &TreasuryPostActiveSave{
			AssetCode:         "USD",
			Amount:            "1.25",
			TargetContainerID: targetContainer.ContainerID,
			SourceRef:         "transfer:reserve-a",
		}
	}))

	if err := ProducePostActiveTreasurySave(root, WriterLockLease{LeaseHolderID: "holder-1"}, PostActiveTreasurySaveInput{
		TreasuryRef: TreasuryRef{TreasuryID: "treasury-save-producer-readout"},
	}, now.Add(time.Minute)); err != nil {
		t.Fatalf("ProducePostActiveTreasurySave() error = %v", err)
	}

	job := testExecutionJob()
	job.Plan.Steps[0].TreasuryRef = &TreasuryRef{TreasuryID: "treasury-save-producer-readout"}
	summary, err := NewInspectSummaryWithTreasuryPreflight(job, "build", root)
	if err != nil {
		t.Fatalf("NewInspectSummaryWithTreasuryPreflight() error = %v", err)
	}
	if len(summary.Steps) != 1 || summary.Steps[0].TreasuryPreflight == nil || summary.Steps[0].TreasuryPreflight.Treasury == nil {
		t.Fatalf("InspectSummary.Steps = %#v, want one treasury preflight step", summary.Steps)
	}
	if summary.Steps[0].TreasuryPreflight.Treasury.PostActiveSave == nil {
		t.Fatalf("InspectSummary treasury post_active_save = %#v, want present save block", summary.Steps[0].TreasuryPreflight.Treasury)
	}
	if !reflect.DeepEqual(summary.Steps[0].TreasuryPreflight.Containers, []FrankContainerRecord{fixtures.container}) {
		t.Fatalf("InspectSummary treasury containers = %#v, want [%#v]", summary.Steps[0].TreasuryPreflight.Containers, fixtures.container)
	}
}
