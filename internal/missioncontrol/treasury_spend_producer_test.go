package missioncontrol

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestProducePostActiveTreasurySpendSuccessfulExecution(t *testing.T) {
	t.Parallel()

	fixtures := writeExecutionContextFrankRegistryFixtures(t)
	root := fixtures.root
	now := time.Date(2026, 4, 14, 15, 20, 0, 0, time.UTC)
	mustStoreTreasuryForMutationTest(t, root, validTreasuryRecord(now, func(record *TreasuryRecord) {
		record.TreasuryID = "treasury-spend-producer"
		record.State = TreasuryStateActive
		record.ContainerRefs = []FrankRegistryObjectRef{{
			Kind:     FrankRegistryObjectKindContainer,
			ObjectID: fixtures.container.ContainerID,
		}}
		record.PostActiveSpend = &TreasuryPostActiveSpend{
			AssetCode: "USD",
			Amount:    "0.75",
			SourceContainerRef: FrankRegistryObjectRef{
				Kind:     FrankRegistryObjectKindContainer,
				ObjectID: fixtures.container.ContainerID,
			},
			TargetRef:       "vendor:domain-renewal",
			SourceRef:       "spend:domain-renewal-a",
			EvidenceLocator: "https://evidence.example/spend-a",
		}
	}))

	if err := ProducePostActiveTreasurySpend(root, WriterLockLease{LeaseHolderID: "holder-1"}, PostActiveTreasurySpendInput{
		TreasuryRef: TreasuryRef{TreasuryID: "treasury-spend-producer"},
	}, now.Add(2*time.Minute)); err != nil {
		t.Fatalf("ProducePostActiveTreasurySpend() error = %v", err)
	}

	treasury, err := LoadTreasuryRecord(root, "treasury-spend-producer")
	if err != nil {
		t.Fatalf("LoadTreasuryRecord() error = %v", err)
	}
	if treasury.PostActiveSpend == nil || treasury.PostActiveSpend.ConsumedEntryID == "" {
		t.Fatalf("LoadTreasuryRecord().PostActiveSpend = %#v, want consumed linkage", treasury.PostActiveSpend)
	}
	entry, err := LoadTreasuryLedgerEntry(root, "treasury-spend-producer", treasury.PostActiveSpend.ConsumedEntryID)
	if err != nil {
		t.Fatalf("LoadTreasuryLedgerEntry() error = %v", err)
	}
	if entry.EntryKind != TreasuryLedgerEntryKindDisposition {
		t.Fatalf("LoadTreasuryLedgerEntry().EntryKind = %q, want %q", entry.EntryKind, TreasuryLedgerEntryKindDisposition)
	}
}

func TestProducePostActiveTreasurySpendReplayFailsClosed(t *testing.T) {
	t.Parallel()

	fixtures := writeExecutionContextFrankRegistryFixtures(t)
	root := fixtures.root
	now := time.Date(2026, 4, 14, 15, 30, 0, 0, time.UTC)
	mustStoreTreasuryForMutationTest(t, root, validTreasuryRecord(now, func(record *TreasuryRecord) {
		record.TreasuryID = "treasury-spend-producer-replay"
		record.State = TreasuryStateActive
		record.ContainerRefs = []FrankRegistryObjectRef{{
			Kind:     FrankRegistryObjectKindContainer,
			ObjectID: fixtures.container.ContainerID,
		}}
		record.PostActiveSpend = &TreasuryPostActiveSpend{
			AssetCode: "USD",
			Amount:    "0.80",
			SourceContainerRef: FrankRegistryObjectRef{
				Kind:     FrankRegistryObjectKindContainer,
				ObjectID: fixtures.container.ContainerID,
			},
			TargetRef: "vendor:domain-renewal",
			SourceRef: "spend:domain-renewal-b",
		}
	}))

	input := PostActiveTreasurySpendInput{TreasuryRef: TreasuryRef{TreasuryID: "treasury-spend-producer-replay"}}
	if err := ProducePostActiveTreasurySpend(root, WriterLockLease{LeaseHolderID: "holder-1"}, input, now.Add(2*time.Minute)); err != nil {
		t.Fatalf("ProducePostActiveTreasurySpend(first) error = %v", err)
	}

	err := ProducePostActiveTreasurySpend(root, WriterLockLease{LeaseHolderID: "holder-1"}, input, now.Add(3*time.Minute))
	if err == nil {
		t.Fatal("ProducePostActiveTreasurySpend(replay) error = nil, want deterministic duplicate rejection")
	}
	if !strings.Contains(err.Error(), `execution context treasury "treasury-spend-producer-replay" treasury.post_active_spend is already consumed by entry "`) {
		t.Fatalf("ProducePostActiveTreasurySpend(replay) error = %q, want consumed replay rejection", err.Error())
	}

	treasury, err := LoadTreasuryRecord(root, input.TreasuryRef.TreasuryID)
	if err != nil {
		t.Fatalf("LoadTreasuryRecord() error = %v", err)
	}
	if treasury.PostActiveSpend == nil || treasury.PostActiveSpend.ConsumedEntryID == "" {
		t.Fatalf("LoadTreasuryRecord().PostActiveSpend = %#v, want consumed linkage", treasury.PostActiveSpend)
	}
}

func TestProducePostActiveTreasurySpendPreservesInspectTreasuryPreflightContract(t *testing.T) {
	t.Parallel()

	fixtures := writeExecutionContextFrankRegistryFixtures(t)
	root := fixtures.root
	now := time.Date(2026, 4, 14, 15, 40, 0, 0, time.UTC)
	record := validTreasuryRecord(now, func(record *TreasuryRecord) {
		record.TreasuryID = "treasury-spend-producer-readout"
		record.State = TreasuryStateActive
		record.ContainerRefs = []FrankRegistryObjectRef{{
			Kind:     FrankRegistryObjectKindContainer,
			ObjectID: fixtures.container.ContainerID,
		}}
		record.PostActiveSpend = &TreasuryPostActiveSpend{
			AssetCode: "USD",
			Amount:    "0.90",
			SourceContainerRef: FrankRegistryObjectRef{
				Kind:     FrankRegistryObjectKindContainer,
				ObjectID: fixtures.container.ContainerID,
			},
			TargetRef:       "vendor:domain-renewal",
			SourceRef:       "spend:domain-renewal-c",
			EvidenceLocator: "https://evidence.example/spend-c",
		}
	})
	mustStoreTreasuryForMutationTest(t, root, record)

	if err := ProducePostActiveTreasurySpend(root, WriterLockLease{LeaseHolderID: "holder-1"}, PostActiveTreasurySpendInput{
		TreasuryRef: TreasuryRef{TreasuryID: "treasury-spend-producer-readout"},
	}, now.Add(2*time.Minute)); err != nil {
		t.Fatalf("ProducePostActiveTreasurySpend() error = %v", err)
	}

	job := testExecutionJob()
	job.Plan.Steps[0].TreasuryRef = &TreasuryRef{TreasuryID: "treasury-spend-producer-readout"}
	summary, err := NewInspectSummaryWithTreasuryPreflight(job, "build", root)
	if err != nil {
		t.Fatalf("NewInspectSummaryWithTreasuryPreflight() error = %v", err)
	}
	if len(summary.Steps) != 1 || summary.Steps[0].TreasuryPreflight == nil || summary.Steps[0].TreasuryPreflight.Treasury == nil {
		t.Fatalf("InspectSummary.Steps = %#v, want one treasury preflight step", summary.Steps)
	}
	if summary.Steps[0].TreasuryPreflight.Treasury.PostActiveSpend == nil {
		t.Fatalf("InspectSummary treasury post_active_spend = %#v, want present spend block", summary.Steps[0].TreasuryPreflight.Treasury)
	}
	if !reflect.DeepEqual(summary.Steps[0].TreasuryPreflight.Containers, []FrankContainerRecord{fixtures.container}) {
		t.Fatalf("InspectSummary treasury containers = %#v, want [%#v]", summary.Steps[0].TreasuryPreflight.Containers, fixtures.container)
	}
}
