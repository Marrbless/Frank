package missioncontrol

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestProducePostActiveTreasuryTransferSuccessfulExecution(t *testing.T) {
	t.Parallel()

	fixtures := writeExecutionContextFrankRegistryFixtures(t)
	root := fixtures.root
	now := time.Date(2026, 4, 14, 14, 40, 0, 0, time.UTC)
	targetContainer := storeTreasurySaveTargetContainerForTest(t, root, "container-vault-producer", "container-class-vault-producer", now)
	mustStoreTreasuryForMutationTest(t, root, validTreasuryRecord(now, func(record *TreasuryRecord) {
		record.TreasuryID = "treasury-transfer-producer"
		record.State = TreasuryStateActive
		record.ContainerRefs = []FrankRegistryObjectRef{{
			Kind:     FrankRegistryObjectKindContainer,
			ObjectID: fixtures.container.ContainerID,
		}}
		record.PostActiveTransfer = &TreasuryPostActiveTransfer{
			AssetCode: "USD",
			Amount:    "1.25",
			SourceContainerRef: FrankRegistryObjectRef{
				Kind:     FrankRegistryObjectKindContainer,
				ObjectID: fixtures.container.ContainerID,
			},
			TargetContainerRef: FrankRegistryObjectRef{
				Kind:     FrankRegistryObjectKindContainer,
				ObjectID: targetContainer.ContainerID,
			},
			SourceRef:       "transfer:rebalance-a",
			EvidenceLocator: "https://evidence.example/transfer-a",
		}
	}))

	if err := ProducePostActiveTreasuryTransfer(root, WriterLockLease{LeaseHolderID: "holder-1"}, PostActiveTreasuryTransferInput{
		TreasuryRef: TreasuryRef{TreasuryID: "treasury-transfer-producer"},
	}, now.Add(2*time.Minute)); err != nil {
		t.Fatalf("ProducePostActiveTreasuryTransfer() error = %v", err)
	}

	treasury, err := LoadTreasuryRecord(root, "treasury-transfer-producer")
	if err != nil {
		t.Fatalf("LoadTreasuryRecord() error = %v", err)
	}
	if treasury.PostActiveTransfer == nil || treasury.PostActiveTransfer.ConsumedEntryID == "" {
		t.Fatalf("LoadTreasuryRecord().PostActiveTransfer = %#v, want consumed linkage", treasury.PostActiveTransfer)
	}
	entry, err := LoadTreasuryLedgerEntry(root, "treasury-transfer-producer", treasury.PostActiveTransfer.ConsumedEntryID)
	if err != nil {
		t.Fatalf("LoadTreasuryLedgerEntry() error = %v", err)
	}
	if entry.EntryKind != TreasuryLedgerEntryKindMovement {
		t.Fatalf("LoadTreasuryLedgerEntry().EntryKind = %q, want %q", entry.EntryKind, TreasuryLedgerEntryKindMovement)
	}
}

func TestProducePostActiveTreasuryTransferReplayFailsClosed(t *testing.T) {
	t.Parallel()

	fixtures := writeExecutionContextFrankRegistryFixtures(t)
	root := fixtures.root
	now := time.Date(2026, 4, 14, 14, 50, 0, 0, time.UTC)
	targetContainer := storeTreasurySaveTargetContainerForTest(t, root, "container-vault-producer-replay", "container-class-vault-producer-replay", now)
	mustStoreTreasuryForMutationTest(t, root, validTreasuryRecord(now, func(record *TreasuryRecord) {
		record.TreasuryID = "treasury-transfer-producer-replay"
		record.State = TreasuryStateActive
		record.ContainerRefs = []FrankRegistryObjectRef{{
			Kind:     FrankRegistryObjectKindContainer,
			ObjectID: fixtures.container.ContainerID,
		}}
		record.PostActiveTransfer = &TreasuryPostActiveTransfer{
			AssetCode: "USD",
			Amount:    "1.25",
			SourceContainerRef: FrankRegistryObjectRef{
				Kind:     FrankRegistryObjectKindContainer,
				ObjectID: fixtures.container.ContainerID,
			},
			TargetContainerRef: FrankRegistryObjectRef{
				Kind:     FrankRegistryObjectKindContainer,
				ObjectID: targetContainer.ContainerID,
			},
			SourceRef: "transfer:rebalance-a",
		}
	}))

	input := PostActiveTreasuryTransferInput{TreasuryRef: TreasuryRef{TreasuryID: "treasury-transfer-producer-replay"}}
	if err := ProducePostActiveTreasuryTransfer(root, WriterLockLease{LeaseHolderID: "holder-1"}, input, now.Add(2*time.Minute)); err != nil {
		t.Fatalf("ProducePostActiveTreasuryTransfer(first) error = %v", err)
	}

	err := ProducePostActiveTreasuryTransfer(root, WriterLockLease{LeaseHolderID: "holder-1"}, input, now.Add(3*time.Minute))
	if err == nil {
		t.Fatal("ProducePostActiveTreasuryTransfer(replay) error = nil, want deterministic duplicate rejection")
	}
	if !strings.Contains(err.Error(), `execution context treasury "treasury-transfer-producer-replay" treasury.post_active_transfer is already consumed by entry "`) {
		t.Fatalf("ProducePostActiveTreasuryTransfer(replay) error = %q, want consumed replay rejection", err.Error())
	}
}

func TestProducePostActiveTreasuryTransferFailsClosedOnIneligibleTargetContainer(t *testing.T) {
	t.Parallel()

	fixtures := writeExecutionContextFrankRegistryFixtures(t)
	root := fixtures.root
	now := time.Date(2026, 4, 14, 15, 0, 0, 0, time.UTC)
	target := AutonomyEligibilityTargetRef{
		Kind:       EligibilityTargetKindTreasuryContainerClass,
		RegistryID: "container-class-vault-ineligible",
	}
	writeFrankRegistryEligibilityFixture(t, root, target, EligibilityLabelIneligible, "container-class-vault-ineligible", "check-container-class-vault-ineligible", now)
	targetContainer := FrankContainerRecord{
		RecordVersion:        StoreRecordVersion,
		ContainerID:          "container-vault-ineligible",
		ContainerKind:        "wallet",
		Label:                "Vault Wallet",
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
		record.TreasuryID = "treasury-transfer-producer-ineligible"
		record.State = TreasuryStateActive
		record.ContainerRefs = []FrankRegistryObjectRef{{
			Kind:     FrankRegistryObjectKindContainer,
			ObjectID: fixtures.container.ContainerID,
		}}
		record.PostActiveTransfer = &TreasuryPostActiveTransfer{
			AssetCode: "USD",
			Amount:    "1.25",
			SourceContainerRef: FrankRegistryObjectRef{
				Kind:     FrankRegistryObjectKindContainer,
				ObjectID: fixtures.container.ContainerID,
			},
			TargetContainerRef: FrankRegistryObjectRef{
				Kind:     FrankRegistryObjectKindContainer,
				ObjectID: targetContainer.ContainerID,
			},
			SourceRef: "transfer:rebalance-a",
		}
	}))

	err := ProducePostActiveTreasuryTransfer(root, WriterLockLease{LeaseHolderID: "holder-1"}, PostActiveTreasuryTransferInput{
		TreasuryRef: TreasuryRef{TreasuryID: "treasury-transfer-producer-ineligible"},
	}, now.Add(2*time.Minute))
	if err == nil {
		t.Fatal("ProducePostActiveTreasuryTransfer() error = nil, want ineligible target rejection")
	}
	if !strings.Contains(err.Error(), target.RegistryID) {
		t.Fatalf("ProducePostActiveTreasuryTransfer() error = %q, want target eligibility rejection", err.Error())
	}

	assertNoTreasuryLedgerEntries(t, root, "treasury-transfer-producer-ineligible")
}

func TestProducePostActiveTreasuryTransferPreservesInspectTreasuryPreflightContract(t *testing.T) {
	t.Parallel()

	fixtures := writeExecutionContextFrankRegistryFixtures(t)
	root := fixtures.root
	now := time.Date(2026, 4, 14, 15, 10, 0, 0, time.UTC)
	targetContainer := storeTreasurySaveTargetContainerForTest(t, root, "container-vault-producer-readout", "container-class-vault-producer-readout", now)
	mustStoreTreasuryForMutationTest(t, root, validTreasuryRecord(now, func(record *TreasuryRecord) {
		record.TreasuryID = "treasury-transfer-producer-readout"
		record.State = TreasuryStateActive
		record.ContainerRefs = []FrankRegistryObjectRef{{
			Kind:     FrankRegistryObjectKindContainer,
			ObjectID: fixtures.container.ContainerID,
		}}
		record.PostActiveTransfer = &TreasuryPostActiveTransfer{
			AssetCode: "USD",
			Amount:    "1.25",
			SourceContainerRef: FrankRegistryObjectRef{
				Kind:     FrankRegistryObjectKindContainer,
				ObjectID: fixtures.container.ContainerID,
			},
			TargetContainerRef: FrankRegistryObjectRef{
				Kind:     FrankRegistryObjectKindContainer,
				ObjectID: targetContainer.ContainerID,
			},
			SourceRef: "transfer:rebalance-a",
		}
	}))

	if err := ProducePostActiveTreasuryTransfer(root, WriterLockLease{LeaseHolderID: "holder-1"}, PostActiveTreasuryTransferInput{
		TreasuryRef: TreasuryRef{TreasuryID: "treasury-transfer-producer-readout"},
	}, now.Add(time.Minute)); err != nil {
		t.Fatalf("ProducePostActiveTreasuryTransfer() error = %v", err)
	}

	job := testExecutionJob()
	job.Plan.Steps[0].TreasuryRef = &TreasuryRef{TreasuryID: "treasury-transfer-producer-readout"}
	summary, err := NewInspectSummaryWithTreasuryPreflight(job, "build", root)
	if err != nil {
		t.Fatalf("NewInspectSummaryWithTreasuryPreflight() error = %v", err)
	}
	if len(summary.Steps) != 1 || summary.Steps[0].TreasuryPreflight == nil || summary.Steps[0].TreasuryPreflight.Treasury == nil {
		t.Fatalf("InspectSummary.Steps = %#v, want one treasury preflight step", summary.Steps)
	}
	if summary.Steps[0].TreasuryPreflight.Treasury.PostActiveTransfer == nil {
		t.Fatalf("InspectSummary treasury post_active_transfer = %#v, want present transfer block", summary.Steps[0].TreasuryPreflight.Treasury)
	}
	if !reflect.DeepEqual(summary.Steps[0].TreasuryPreflight.Containers, []FrankContainerRecord{fixtures.container}) {
		t.Fatalf("InspectSummary treasury containers = %#v, want [%#v]", summary.Steps[0].TreasuryPreflight.Containers, fixtures.container)
	}
}
