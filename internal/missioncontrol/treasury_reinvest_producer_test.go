package missioncontrol

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestProducePostActiveTreasuryReinvestSuccessfulExecution(t *testing.T) {
	t.Parallel()

	fixtures := writeExecutionContextFrankRegistryFixtures(t)
	root := fixtures.root
	now := time.Date(2026, 4, 14, 15, 50, 0, 0, time.UTC)
	targetContainer := storeTreasurySaveTargetContainerForTest(t, root, "container-investment-producer", "container-class-investment-producer", now)
	mustStoreTreasuryForMutationTest(t, root, validTreasuryRecord(now, func(record *TreasuryRecord) {
		record.TreasuryID = "treasury-reinvest-producer"
		record.State = TreasuryStateActive
		record.ContainerRefs = []FrankRegistryObjectRef{{
			Kind:     FrankRegistryObjectKindContainer,
			ObjectID: fixtures.container.ContainerID,
		}}
		record.PostActiveReinvest = &TreasuryPostActiveReinvest{
			SourceAssetCode: "USD",
			SourceAmount:    "0.75",
			TargetAssetCode: "BTC",
			TargetAmount:    "0.00001000",
			SourceContainerRef: FrankRegistryObjectRef{
				Kind:     FrankRegistryObjectKindContainer,
				ObjectID: fixtures.container.ContainerID,
			},
			TargetContainerRef: FrankRegistryObjectRef{
				Kind:     FrankRegistryObjectKindContainer,
				ObjectID: targetContainer.ContainerID,
			},
			SourceRef:       "trade:reinvest-a",
			EvidenceLocator: "https://evidence.example/reinvest-a",
			ConfirmedAt:     now.Add(time.Minute),
		}
	}))

	if err := ProducePostActiveTreasuryReinvest(root, WriterLockLease{LeaseHolderID: "holder-1"}, PostActiveTreasuryReinvestInput{
		TreasuryRef: TreasuryRef{TreasuryID: "treasury-reinvest-producer"},
	}, now.Add(2*time.Minute)); err != nil {
		t.Fatalf("ProducePostActiveTreasuryReinvest() error = %v", err)
	}

	treasury, err := LoadTreasuryRecord(root, "treasury-reinvest-producer")
	if err != nil {
		t.Fatalf("LoadTreasuryRecord() error = %v", err)
	}
	if treasury.PostActiveReinvest == nil || treasury.PostActiveReinvest.ConsumedEntryID == "" {
		t.Fatalf("LoadTreasuryRecord().PostActiveReinvest = %#v, want consumed linkage", treasury.PostActiveReinvest)
	}
	entries, err := ListTreasuryLedgerEntries(root, "treasury-reinvest-producer")
	if err != nil {
		t.Fatalf("ListTreasuryLedgerEntries() error = %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("ListTreasuryLedgerEntries() len = %d, want 2", len(entries))
	}
}

func TestProducePostActiveTreasuryReinvestReplayFailsClosed(t *testing.T) {
	t.Parallel()

	fixtures := writeExecutionContextFrankRegistryFixtures(t)
	root := fixtures.root
	now := time.Date(2026, 4, 14, 16, 0, 0, 0, time.UTC)
	targetContainer := storeTreasurySaveTargetContainerForTest(t, root, "container-investment-producer-replay", "container-class-investment-producer-replay", now)
	mustStoreTreasuryForMutationTest(t, root, validTreasuryRecord(now, func(record *TreasuryRecord) {
		record.TreasuryID = "treasury-reinvest-producer-replay"
		record.State = TreasuryStateActive
		record.ContainerRefs = []FrankRegistryObjectRef{{
			Kind:     FrankRegistryObjectKindContainer,
			ObjectID: fixtures.container.ContainerID,
		}}
		record.PostActiveReinvest = &TreasuryPostActiveReinvest{
			SourceAssetCode: "USD",
			SourceAmount:    "0.80",
			TargetAssetCode: "BTC",
			TargetAmount:    "0.00001100",
			SourceContainerRef: FrankRegistryObjectRef{
				Kind:     FrankRegistryObjectKindContainer,
				ObjectID: fixtures.container.ContainerID,
			},
			TargetContainerRef: FrankRegistryObjectRef{
				Kind:     FrankRegistryObjectKindContainer,
				ObjectID: targetContainer.ContainerID,
			},
			SourceRef:       "trade:reinvest-b",
			EvidenceLocator: "https://evidence.example/reinvest-b",
			ConfirmedAt:     now.Add(time.Minute),
		}
	}))

	input := PostActiveTreasuryReinvestInput{TreasuryRef: TreasuryRef{TreasuryID: "treasury-reinvest-producer-replay"}}
	if err := ProducePostActiveTreasuryReinvest(root, WriterLockLease{LeaseHolderID: "holder-1"}, input, now.Add(2*time.Minute)); err != nil {
		t.Fatalf("ProducePostActiveTreasuryReinvest(first) error = %v", err)
	}

	err := ProducePostActiveTreasuryReinvest(root, WriterLockLease{LeaseHolderID: "holder-1"}, input, now.Add(3*time.Minute))
	if err == nil {
		t.Fatal("ProducePostActiveTreasuryReinvest(replay) error = nil, want deterministic duplicate rejection")
	}
	if !strings.Contains(err.Error(), `execution context treasury "treasury-reinvest-producer-replay" treasury.post_active_reinvest is already consumed by entry "`) {
		t.Fatalf("ProducePostActiveTreasuryReinvest(replay) error = %q, want consumed replay rejection", err.Error())
	}
}

func TestProducePostActiveTreasuryReinvestFailsClosedOnIneligibleTargetContainer(t *testing.T) {
	t.Parallel()

	fixtures := writeExecutionContextFrankRegistryFixtures(t)
	root := fixtures.root
	now := time.Date(2026, 4, 14, 16, 10, 0, 0, time.UTC)
	target := AutonomyEligibilityTargetRef{
		Kind:       EligibilityTargetKindTreasuryContainerClass,
		RegistryID: "container-class-investment-ineligible",
	}
	writeFrankRegistryEligibilityFixture(t, root, target, EligibilityLabelIneligible, "container-class-investment-ineligible", "check-container-class-investment-ineligible", now)
	targetContainer := FrankContainerRecord{
		RecordVersion:        StoreRecordVersion,
		ContainerID:          "container-investment-ineligible",
		ContainerKind:        "wallet",
		Label:                "Investment Wallet",
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
		record.TreasuryID = "treasury-reinvest-producer-ineligible"
		record.State = TreasuryStateActive
		record.ContainerRefs = []FrankRegistryObjectRef{{
			Kind:     FrankRegistryObjectKindContainer,
			ObjectID: fixtures.container.ContainerID,
		}}
		record.PostActiveReinvest = &TreasuryPostActiveReinvest{
			SourceAssetCode: "USD",
			SourceAmount:    "0.80",
			TargetAssetCode: "BTC",
			TargetAmount:    "0.00001100",
			SourceContainerRef: FrankRegistryObjectRef{
				Kind:     FrankRegistryObjectKindContainer,
				ObjectID: fixtures.container.ContainerID,
			},
			TargetContainerRef: FrankRegistryObjectRef{
				Kind:     FrankRegistryObjectKindContainer,
				ObjectID: targetContainer.ContainerID,
			},
			SourceRef:       "trade:reinvest-c",
			EvidenceLocator: "https://evidence.example/reinvest-c",
			ConfirmedAt:     now.Add(time.Minute),
		}
	}))

	err := ProducePostActiveTreasuryReinvest(root, WriterLockLease{LeaseHolderID: "holder-1"}, PostActiveTreasuryReinvestInput{
		TreasuryRef: TreasuryRef{TreasuryID: "treasury-reinvest-producer-ineligible"},
	}, now.Add(2*time.Minute))
	if err == nil {
		t.Fatal("ProducePostActiveTreasuryReinvest() error = nil, want target eligibility rejection")
	}
	if !strings.Contains(err.Error(), `autonomy eligibility target "container-class-investment-ineligible" is not autonomy-compatible`) {
		t.Fatalf("ProducePostActiveTreasuryReinvest() error = %q, want target eligibility rejection", err.Error())
	}
	assertNoTreasuryLedgerEntries(t, root, "treasury-reinvest-producer-ineligible")
}

func TestProducePostActiveTreasuryReinvestPreservesInspectTreasuryPreflightContract(t *testing.T) {
	t.Parallel()

	fixtures := writeExecutionContextFrankRegistryFixtures(t)
	root := fixtures.root
	now := time.Date(2026, 4, 14, 16, 20, 0, 0, time.UTC)
	targetContainer := storeTreasurySaveTargetContainerForTest(t, root, "container-investment-producer-readout", "container-class-investment-producer-readout", now)
	mustStoreTreasuryForMutationTest(t, root, validTreasuryRecord(now, func(record *TreasuryRecord) {
		record.TreasuryID = "treasury-reinvest-producer-readout"
		record.State = TreasuryStateActive
		record.ContainerRefs = []FrankRegistryObjectRef{{
			Kind:     FrankRegistryObjectKindContainer,
			ObjectID: fixtures.container.ContainerID,
		}}
		record.PostActiveReinvest = &TreasuryPostActiveReinvest{
			SourceAssetCode: "USD",
			SourceAmount:    "0.90",
			TargetAssetCode: "BTC",
			TargetAmount:    "0.00001250",
			SourceContainerRef: FrankRegistryObjectRef{
				Kind:     FrankRegistryObjectKindContainer,
				ObjectID: fixtures.container.ContainerID,
			},
			TargetContainerRef: FrankRegistryObjectRef{
				Kind:     FrankRegistryObjectKindContainer,
				ObjectID: targetContainer.ContainerID,
			},
			SourceRef:       "trade:reinvest-d",
			EvidenceLocator: "https://evidence.example/reinvest-d",
			ConfirmedAt:     now.Add(time.Minute),
		}
	}))

	if err := ProducePostActiveTreasuryReinvest(root, WriterLockLease{LeaseHolderID: "holder-1"}, PostActiveTreasuryReinvestInput{
		TreasuryRef: TreasuryRef{TreasuryID: "treasury-reinvest-producer-readout"},
	}, now.Add(2*time.Minute)); err != nil {
		t.Fatalf("ProducePostActiveTreasuryReinvest() error = %v", err)
	}

	job := testExecutionJob()
	job.Plan.Steps[0].TreasuryRef = &TreasuryRef{TreasuryID: "treasury-reinvest-producer-readout"}
	summary, err := NewInspectSummaryWithTreasuryPreflight(job, "build", root)
	if err != nil {
		t.Fatalf("NewInspectSummaryWithTreasuryPreflight() error = %v", err)
	}
	if len(summary.Steps) != 1 || summary.Steps[0].TreasuryPreflight == nil || summary.Steps[0].TreasuryPreflight.Treasury == nil {
		t.Fatalf("InspectSummary.Steps = %#v, want one treasury preflight step", summary.Steps)
	}
	if summary.Steps[0].TreasuryPreflight.Treasury.PostActiveReinvest == nil {
		t.Fatalf("InspectSummary treasury post_active_reinvest = %#v, want present reinvest block", summary.Steps[0].TreasuryPreflight.Treasury)
	}
	if !reflect.DeepEqual(summary.Steps[0].TreasuryPreflight.Containers, []FrankContainerRecord{fixtures.container}) {
		t.Fatalf("InspectSummary treasury containers = %#v, want [%#v]", summary.Steps[0].TreasuryPreflight.Containers, fixtures.container)
	}
}
