package missioncontrol

import (
	"reflect"
	"testing"
	"time"
)

func TestProducePostSuspendTreasuryResumeSuccessfulExecution(t *testing.T) {
	t.Parallel()

	fixtures := writeExecutionContextFrankRegistryFixtures(t)
	root := fixtures.root
	now := time.Date(2026, 4, 14, 16, 10, 0, 0, time.UTC)
	mustStoreTreasuryForMutationTest(t, root, validTreasuryRecord(now, func(record *TreasuryRecord) {
		record.TreasuryID = "treasury-resume-producer"
		record.State = TreasuryStateSuspended
		record.ContainerRefs = []FrankRegistryObjectRef{{
			Kind:     FrankRegistryObjectKindContainer,
			ObjectID: fixtures.container.ContainerID,
		}}
		record.PostActiveSuspend = &TreasuryPostActiveSuspend{
			Reason:               "risk:manual-review-required",
			SourceRef:            "suspend:risk-review-a",
			ConsumedTransitionID: "transition-suspend-a",
		}
		record.PostSuspendResume = &TreasuryPostSuspendResume{
			Reason:    "ops:manual-clear",
			SourceRef: "resume:manual-clear-a",
		}
	}))

	if err := ProducePostSuspendTreasuryResume(root, WriterLockLease{LeaseHolderID: "holder-1"}, PostSuspendTreasuryResumeInput{
		TreasuryRef: TreasuryRef{TreasuryID: "treasury-resume-producer"},
	}, now.Add(2*time.Minute)); err != nil {
		t.Fatalf("ProducePostSuspendTreasuryResume() error = %v", err)
	}

	treasury, err := LoadTreasuryRecord(root, "treasury-resume-producer")
	if err != nil {
		t.Fatalf("LoadTreasuryRecord() error = %v", err)
	}
	if treasury.State != TreasuryStateActive {
		t.Fatalf("LoadTreasuryRecord().State = %q, want %q", treasury.State, TreasuryStateActive)
	}
	if treasury.PostSuspendResume == nil || treasury.PostSuspendResume.ConsumedTransitionID == "" {
		t.Fatalf("LoadTreasuryRecord().PostSuspendResume = %#v, want consumed linkage", treasury.PostSuspendResume)
	}
	if treasury.PostActiveSuspend != nil {
		t.Fatalf("LoadTreasuryRecord().PostActiveSuspend = %#v, want cleared suspend block", treasury.PostActiveSuspend)
	}
}

func TestProducePostSuspendTreasuryResumeReplayIsNoOpAfterCommittedResume(t *testing.T) {
	t.Parallel()

	fixtures := writeExecutionContextFrankRegistryFixtures(t)
	root := fixtures.root
	now := time.Date(2026, 4, 14, 16, 15, 0, 0, time.UTC)
	mustStoreTreasuryForMutationTest(t, root, validTreasuryRecord(now, func(record *TreasuryRecord) {
		record.TreasuryID = "treasury-resume-producer-replay"
		record.State = TreasuryStateSuspended
		record.ContainerRefs = []FrankRegistryObjectRef{{
			Kind:     FrankRegistryObjectKindContainer,
			ObjectID: fixtures.container.ContainerID,
		}}
		record.PostActiveSuspend = &TreasuryPostActiveSuspend{
			Reason:               "risk:manual-review-required",
			SourceRef:            "suspend:risk-review-b",
			ConsumedTransitionID: "transition-suspend-b",
		}
		record.PostSuspendResume = &TreasuryPostSuspendResume{
			Reason:    "ops:manual-clear",
			SourceRef: "resume:manual-clear-b",
		}
	}))

	input := PostSuspendTreasuryResumeInput{TreasuryRef: TreasuryRef{TreasuryID: "treasury-resume-producer-replay"}}
	if err := ProducePostSuspendTreasuryResume(root, WriterLockLease{LeaseHolderID: "holder-1"}, input, now.Add(2*time.Minute)); err != nil {
		t.Fatalf("ProducePostSuspendTreasuryResume(first) error = %v", err)
	}
	firstTreasury, err := LoadTreasuryRecord(root, "treasury-resume-producer-replay")
	if err != nil {
		t.Fatalf("LoadTreasuryRecord(first) error = %v", err)
	}

	if err := ProducePostSuspendTreasuryResume(root, WriterLockLease{LeaseHolderID: "holder-1"}, input, now.Add(3*time.Minute)); err != nil {
		t.Fatalf("ProducePostSuspendTreasuryResume(replay) error = %v, want active no-op", err)
	}
	secondTreasury, err := LoadTreasuryRecord(root, "treasury-resume-producer-replay")
	if err != nil {
		t.Fatalf("LoadTreasuryRecord(second) error = %v", err)
	}
	if !reflect.DeepEqual(secondTreasury, firstTreasury) {
		t.Fatalf("LoadTreasuryRecord(second) = %#v, want unchanged %#v", secondTreasury, firstTreasury)
	}
}

func TestProducePostSuspendTreasuryResumePreservesInspectTreasuryPreflightContract(t *testing.T) {
	t.Parallel()

	fixtures := writeExecutionContextFrankRegistryFixtures(t)
	root := fixtures.root
	now := time.Date(2026, 4, 14, 16, 20, 0, 0, time.UTC)
	record := validTreasuryRecord(now, func(record *TreasuryRecord) {
		record.TreasuryID = "treasury-resume-producer-readout"
		record.State = TreasuryStateSuspended
		record.ContainerRefs = []FrankRegistryObjectRef{{
			Kind:     FrankRegistryObjectKindContainer,
			ObjectID: fixtures.container.ContainerID,
		}}
		record.PostActiveSuspend = &TreasuryPostActiveSuspend{
			Reason:               "risk:manual-review-required",
			SourceRef:            "suspend:risk-review-c",
			ConsumedTransitionID: "transition-suspend-c",
		}
		record.PostSuspendResume = &TreasuryPostSuspendResume{
			Reason:    "ops:manual-clear",
			SourceRef: "resume:manual-clear-c",
		}
	})
	mustStoreTreasuryForMutationTest(t, root, record)

	if err := ProducePostSuspendTreasuryResume(root, WriterLockLease{LeaseHolderID: "holder-1"}, PostSuspendTreasuryResumeInput{
		TreasuryRef: TreasuryRef{TreasuryID: "treasury-resume-producer-readout"},
	}, now.Add(2*time.Minute)); err != nil {
		t.Fatalf("ProducePostSuspendTreasuryResume() error = %v", err)
	}

	job := testExecutionJob()
	job.Plan.Steps[0].TreasuryRef = &TreasuryRef{TreasuryID: "treasury-resume-producer-readout"}
	summary, err := NewInspectSummaryWithTreasuryPreflight(job, "build", root)
	if err != nil {
		t.Fatalf("NewInspectSummaryWithTreasuryPreflight() error = %v", err)
	}
	if len(summary.Steps) != 1 || summary.Steps[0].TreasuryPreflight == nil || summary.Steps[0].TreasuryPreflight.Treasury == nil {
		t.Fatalf("InspectSummary.Steps = %#v, want one treasury preflight step", summary.Steps)
	}
	if summary.Steps[0].TreasuryPreflight.Treasury.PostSuspendResume == nil {
		t.Fatalf("InspectSummary treasury post_suspend_resume = %#v, want present resume block", summary.Steps[0].TreasuryPreflight.Treasury)
	}
	if summary.Steps[0].TreasuryPreflight.Treasury.State != TreasuryStateActive {
		t.Fatalf("InspectSummary treasury state = %q, want %q", summary.Steps[0].TreasuryPreflight.Treasury.State, TreasuryStateActive)
	}
	if summary.Steps[0].TreasuryPreflight.Treasury.PostActiveSuspend != nil {
		t.Fatalf("InspectSummary treasury post_active_suspend = %#v, want cleared suspend block", summary.Steps[0].TreasuryPreflight.Treasury.PostActiveSuspend)
	}
	if !reflect.DeepEqual(summary.Steps[0].TreasuryPreflight.Containers, []FrankContainerRecord{fixtures.container}) {
		t.Fatalf("InspectSummary treasury containers = %#v, want [%#v]", summary.Steps[0].TreasuryPreflight.Containers, fixtures.container)
	}
}
