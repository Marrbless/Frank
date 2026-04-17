package missioncontrol

import (
	"reflect"
	"testing"
	"time"
)

func TestProducePostActiveTreasurySuspendSuccessfulExecution(t *testing.T) {
	t.Parallel()

	fixtures := writeExecutionContextFrankRegistryFixtures(t)
	root := fixtures.root
	now := time.Date(2026, 4, 14, 15, 0, 0, 0, time.UTC)
	mustStoreTreasuryForMutationTest(t, root, validTreasuryRecord(now, func(record *TreasuryRecord) {
		record.TreasuryID = "treasury-suspend-producer"
		record.State = TreasuryStateActive
		record.ContainerRefs = []FrankRegistryObjectRef{{
			Kind:     FrankRegistryObjectKindContainer,
			ObjectID: fixtures.container.ContainerID,
		}}
		record.PostActiveSuspend = &TreasuryPostActiveSuspend{
			Reason:    "risk:manual-review-required",
			SourceRef: "suspend:risk-review-a",
		}
	}))

	if err := ProducePostActiveTreasurySuspend(root, WriterLockLease{LeaseHolderID: "holder-1"}, PostActiveTreasurySuspendInput{
		TreasuryRef: TreasuryRef{TreasuryID: "treasury-suspend-producer"},
	}, now.Add(2*time.Minute)); err != nil {
		t.Fatalf("ProducePostActiveTreasurySuspend() error = %v", err)
	}

	treasury, err := LoadTreasuryRecord(root, "treasury-suspend-producer")
	if err != nil {
		t.Fatalf("LoadTreasuryRecord() error = %v", err)
	}
	if treasury.State != TreasuryStateSuspended {
		t.Fatalf("LoadTreasuryRecord().State = %q, want %q", treasury.State, TreasuryStateSuspended)
	}
	if treasury.PostActiveSuspend == nil || treasury.PostActiveSuspend.ConsumedTransitionID == "" {
		t.Fatalf("LoadTreasuryRecord().PostActiveSuspend = %#v, want consumed linkage", treasury.PostActiveSuspend)
	}
}

func TestProducePostActiveTreasurySuspendReplayIsNoOpAfterCommittedSuspension(t *testing.T) {
	t.Parallel()

	fixtures := writeExecutionContextFrankRegistryFixtures(t)
	root := fixtures.root
	now := time.Date(2026, 4, 14, 15, 5, 0, 0, time.UTC)
	mustStoreTreasuryForMutationTest(t, root, validTreasuryRecord(now, func(record *TreasuryRecord) {
		record.TreasuryID = "treasury-suspend-producer-replay"
		record.State = TreasuryStateActive
		record.ContainerRefs = []FrankRegistryObjectRef{{
			Kind:     FrankRegistryObjectKindContainer,
			ObjectID: fixtures.container.ContainerID,
		}}
		record.PostActiveSuspend = &TreasuryPostActiveSuspend{
			Reason:    "risk:manual-review-required",
			SourceRef: "suspend:risk-review-b",
		}
	}))

	input := PostActiveTreasurySuspendInput{TreasuryRef: TreasuryRef{TreasuryID: "treasury-suspend-producer-replay"}}
	if err := ProducePostActiveTreasurySuspend(root, WriterLockLease{LeaseHolderID: "holder-1"}, input, now.Add(2*time.Minute)); err != nil {
		t.Fatalf("ProducePostActiveTreasurySuspend(first) error = %v", err)
	}
	firstTreasury, err := LoadTreasuryRecord(root, "treasury-suspend-producer-replay")
	if err != nil {
		t.Fatalf("LoadTreasuryRecord(first) error = %v", err)
	}

	if err := ProducePostActiveTreasurySuspend(root, WriterLockLease{LeaseHolderID: "holder-1"}, input, now.Add(3*time.Minute)); err != nil {
		t.Fatalf("ProducePostActiveTreasurySuspend(replay) error = %v, want suspended no-op", err)
	}
	secondTreasury, err := LoadTreasuryRecord(root, "treasury-suspend-producer-replay")
	if err != nil {
		t.Fatalf("LoadTreasuryRecord(second) error = %v", err)
	}
	if !reflect.DeepEqual(secondTreasury, firstTreasury) {
		t.Fatalf("LoadTreasuryRecord(second) = %#v, want unchanged %#v", secondTreasury, firstTreasury)
	}
}

func TestProducePostActiveTreasurySuspendPreservesInspectTreasuryPreflightContract(t *testing.T) {
	t.Parallel()

	fixtures := writeExecutionContextFrankRegistryFixtures(t)
	root := fixtures.root
	now := time.Date(2026, 4, 14, 15, 15, 0, 0, time.UTC)
	record := validTreasuryRecord(now, func(record *TreasuryRecord) {
		record.TreasuryID = "treasury-suspend-producer-readout"
		record.State = TreasuryStateActive
		record.ContainerRefs = []FrankRegistryObjectRef{{
			Kind:     FrankRegistryObjectKindContainer,
			ObjectID: fixtures.container.ContainerID,
		}}
		record.PostActiveSuspend = &TreasuryPostActiveSuspend{
			Reason:    "risk:manual-review-required",
			SourceRef: "suspend:risk-review-c",
		}
	})
	mustStoreTreasuryForMutationTest(t, root, record)

	if err := ProducePostActiveTreasurySuspend(root, WriterLockLease{LeaseHolderID: "holder-1"}, PostActiveTreasurySuspendInput{
		TreasuryRef: TreasuryRef{TreasuryID: "treasury-suspend-producer-readout"},
	}, now.Add(2*time.Minute)); err != nil {
		t.Fatalf("ProducePostActiveTreasurySuspend() error = %v", err)
	}

	job := testExecutionJob()
	job.Plan.Steps[0].TreasuryRef = &TreasuryRef{TreasuryID: "treasury-suspend-producer-readout"}
	summary, err := NewInspectSummaryWithTreasuryPreflight(job, "build", root)
	if err != nil {
		t.Fatalf("NewInspectSummaryWithTreasuryPreflight() error = %v", err)
	}
	if len(summary.Steps) != 1 || summary.Steps[0].TreasuryPreflight == nil || summary.Steps[0].TreasuryPreflight.Treasury == nil {
		t.Fatalf("InspectSummary.Steps = %#v, want one treasury preflight step", summary.Steps)
	}
	if summary.Steps[0].TreasuryPreflight.Treasury.PostActiveSuspend == nil {
		t.Fatalf("InspectSummary treasury post_active_suspend = %#v, want present suspend block", summary.Steps[0].TreasuryPreflight.Treasury)
	}
	if summary.Steps[0].TreasuryPreflight.Treasury.State != TreasuryStateSuspended {
		t.Fatalf("InspectSummary treasury state = %q, want %q", summary.Steps[0].TreasuryPreflight.Treasury.State, TreasuryStateSuspended)
	}
	if !reflect.DeepEqual(summary.Steps[0].TreasuryPreflight.Containers, []FrankContainerRecord{fixtures.container}) {
		t.Fatalf("InspectSummary treasury containers = %#v, want [%#v]", summary.Steps[0].TreasuryPreflight.Containers, fixtures.container)
	}
}
