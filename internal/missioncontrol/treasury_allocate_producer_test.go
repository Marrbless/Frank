package missioncontrol

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestProducePostActiveTreasuryAllocateSuccessfulExecution(t *testing.T) {
	t.Parallel()

	fixtures := writeExecutionContextFrankRegistryFixtures(t)
	root := fixtures.root
	now := time.Date(2026, 4, 14, 15, 10, 0, 0, time.UTC)
	mustStoreTreasuryForMutationTest(t, root, validTreasuryRecord(now, func(record *TreasuryRecord) {
		record.TreasuryID = "treasury-allocate-producer"
		record.State = TreasuryStateActive
		record.ContainerRefs = []FrankRegistryObjectRef{{
			Kind:     FrankRegistryObjectKindContainer,
			ObjectID: fixtures.container.ContainerID,
		}}
		record.PostActiveAllocate = &TreasuryPostActiveAllocate{
			AssetCode: "USD",
			Amount:    "1.10",
			SourceContainerRef: FrankRegistryObjectRef{
				Kind:     FrankRegistryObjectKindContainer,
				ObjectID: fixtures.container.ContainerID,
			},
			AllocationTargetRef: "allocation:ops-reserve",
			SourceRef:           "allocate:ops-reserve-a",
		}
	}))

	if err := ProducePostActiveTreasuryAllocate(root, WriterLockLease{LeaseHolderID: "holder-1"}, PostActiveTreasuryAllocateInput{
		TreasuryRef: TreasuryRef{TreasuryID: "treasury-allocate-producer"},
	}, now.Add(2*time.Minute)); err != nil {
		t.Fatalf("ProducePostActiveTreasuryAllocate() error = %v", err)
	}

	treasury, err := LoadTreasuryRecord(root, "treasury-allocate-producer")
	if err != nil {
		t.Fatalf("LoadTreasuryRecord() error = %v", err)
	}
	if treasury.PostActiveAllocate == nil || treasury.PostActiveAllocate.ConsumedEntryID == "" {
		t.Fatalf("LoadTreasuryRecord().PostActiveAllocate = %#v, want consumed linkage", treasury.PostActiveAllocate)
	}
	entry, err := LoadTreasuryLedgerEntry(root, "treasury-allocate-producer", treasury.PostActiveAllocate.ConsumedEntryID)
	if err != nil {
		t.Fatalf("LoadTreasuryLedgerEntry() error = %v", err)
	}
	if entry.EntryKind != TreasuryLedgerEntryKindMovement {
		t.Fatalf("LoadTreasuryLedgerEntry().EntryKind = %q, want %q", entry.EntryKind, TreasuryLedgerEntryKindMovement)
	}
}

func TestProducePostActiveTreasuryAllocateReplayFailsClosed(t *testing.T) {
	t.Parallel()

	fixtures := writeExecutionContextFrankRegistryFixtures(t)
	root := fixtures.root
	now := time.Date(2026, 4, 14, 15, 20, 0, 0, time.UTC)
	mustStoreTreasuryForMutationTest(t, root, validTreasuryRecord(now, func(record *TreasuryRecord) {
		record.TreasuryID = "treasury-allocate-producer-replay"
		record.State = TreasuryStateActive
		record.ContainerRefs = []FrankRegistryObjectRef{{
			Kind:     FrankRegistryObjectKindContainer,
			ObjectID: fixtures.container.ContainerID,
		}}
		record.PostActiveAllocate = &TreasuryPostActiveAllocate{
			AssetCode: "USD",
			Amount:    "1.20",
			SourceContainerRef: FrankRegistryObjectRef{
				Kind:     FrankRegistryObjectKindContainer,
				ObjectID: fixtures.container.ContainerID,
			},
			AllocationTargetRef: "allocation:ops-reserve",
			SourceRef:           "allocate:ops-reserve-b",
		}
	}))

	input := PostActiveTreasuryAllocateInput{TreasuryRef: TreasuryRef{TreasuryID: "treasury-allocate-producer-replay"}}
	if err := ProducePostActiveTreasuryAllocate(root, WriterLockLease{LeaseHolderID: "holder-1"}, input, now.Add(2*time.Minute)); err != nil {
		t.Fatalf("ProducePostActiveTreasuryAllocate(first) error = %v", err)
	}

	err := ProducePostActiveTreasuryAllocate(root, WriterLockLease{LeaseHolderID: "holder-1"}, input, now.Add(3*time.Minute))
	if err == nil {
		t.Fatal("ProducePostActiveTreasuryAllocate(replay) error = nil, want deterministic duplicate rejection")
	}
	if !strings.Contains(err.Error(), `execution context treasury "treasury-allocate-producer-replay" treasury.post_active_allocate is already consumed by entry "`) {
		t.Fatalf("ProducePostActiveTreasuryAllocate(replay) error = %q, want consumed replay rejection", err.Error())
	}
}

func TestProducePostActiveTreasuryAllocatePreservesInspectTreasuryPreflightContract(t *testing.T) {
	t.Parallel()

	fixtures := writeExecutionContextFrankRegistryFixtures(t)
	root := fixtures.root
	now := time.Date(2026, 4, 14, 15, 30, 0, 0, time.UTC)
	record := validTreasuryRecord(now, func(record *TreasuryRecord) {
		record.TreasuryID = "treasury-allocate-producer-readout"
		record.State = TreasuryStateActive
		record.ContainerRefs = []FrankRegistryObjectRef{{
			Kind:     FrankRegistryObjectKindContainer,
			ObjectID: fixtures.container.ContainerID,
		}}
		record.PostActiveAllocate = &TreasuryPostActiveAllocate{
			AssetCode: "USD",
			Amount:    "1.30",
			SourceContainerRef: FrankRegistryObjectRef{
				Kind:     FrankRegistryObjectKindContainer,
				ObjectID: fixtures.container.ContainerID,
			},
			AllocationTargetRef: "allocation:ops-reserve",
			SourceRef:           "allocate:ops-reserve-c",
		}
	})
	mustStoreTreasuryForMutationTest(t, root, record)

	if err := ProducePostActiveTreasuryAllocate(root, WriterLockLease{LeaseHolderID: "holder-1"}, PostActiveTreasuryAllocateInput{
		TreasuryRef: TreasuryRef{TreasuryID: "treasury-allocate-producer-readout"},
	}, now.Add(2*time.Minute)); err != nil {
		t.Fatalf("ProducePostActiveTreasuryAllocate() error = %v", err)
	}

	job := testExecutionJob()
	job.Plan.Steps[0].TreasuryRef = &TreasuryRef{TreasuryID: "treasury-allocate-producer-readout"}
	summary, err := NewInspectSummaryWithTreasuryPreflight(job, "build", root)
	if err != nil {
		t.Fatalf("NewInspectSummaryWithTreasuryPreflight() error = %v", err)
	}
	if len(summary.Steps) != 1 || summary.Steps[0].TreasuryPreflight == nil || summary.Steps[0].TreasuryPreflight.Treasury == nil {
		t.Fatalf("InspectSummary.Steps = %#v, want one treasury preflight step", summary.Steps)
	}
	if summary.Steps[0].TreasuryPreflight.Treasury.PostActiveAllocate == nil {
		t.Fatalf("InspectSummary treasury post_active_allocate = %#v, want present allocate block", summary.Steps[0].TreasuryPreflight.Treasury)
	}
	if !reflect.DeepEqual(summary.Steps[0].TreasuryPreflight.Containers, []FrankContainerRecord{fixtures.container}) {
		t.Fatalf("InspectSummary treasury containers = %#v, want [%#v]", summary.Steps[0].TreasuryPreflight.Containers, fixtures.container)
	}
}
