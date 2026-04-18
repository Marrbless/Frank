package missioncontrol

import (
	"strings"
	"testing"
	"time"
)

func TestStoreWorkspaceBroadAppControlCapabilityExposureCreatesSourceAndCapabilityRecords(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	workspace := t.TempDir()
	if _, err := StoreWorkspaceSharedStorageCapabilityExposure(root, workspace); err != nil {
		t.Fatalf("StoreWorkspaceSharedStorageCapabilityExposure() error = %v", err)
	}

	if _, err := StoreWorkspaceBroadAppControlCapabilityExposure(root, workspace); err != nil {
		t.Fatalf("StoreWorkspaceBroadAppControlCapabilityExposure(create) error = %v", err)
	}

	record, err := RequireExposedBroadAppControlCapabilityRecord(root)
	if err != nil {
		t.Fatalf("RequireExposedBroadAppControlCapabilityRecord() error = %v", err)
	}
	if record.CapabilityID != BroadAppControlLocalFileCapabilityID {
		t.Fatalf("CapabilityID = %q, want %q", record.CapabilityID, BroadAppControlLocalFileCapabilityID)
	}
	if record.Name != BroadAppControlCapabilityName {
		t.Fatalf("Name = %q, want %q", record.Name, BroadAppControlCapabilityName)
	}
	if !record.Exposed {
		t.Fatal("Exposed = false, want true")
	}

	source, err := RequireReadableBroadAppControlSourceRecord(root, workspace)
	if err != nil {
		t.Fatalf("RequireReadableBroadAppControlSourceRecord() error = %v", err)
	}
	if source.Path != BroadAppControlLocalFileDefaultPath {
		t.Fatalf("Path = %q, want %q", source.Path, BroadAppControlLocalFileDefaultPath)
	}
}

func TestStoreWorkspaceBroadAppControlCapabilityExposureFailsClosedWithoutSharedStorageExposure(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	workspace := t.TempDir()

	_, err := StoreWorkspaceBroadAppControlCapabilityExposure(root, workspace)
	if err == nil {
		t.Fatal("StoreWorkspaceBroadAppControlCapabilityExposure() error = nil, want shared_storage rejection")
	}
	if !strings.Contains(err.Error(), `broad_app_control capability requires shared_storage exposure: shared_storage capability requires one committed capability record named "shared_storage"`) {
		t.Fatalf("StoreWorkspaceBroadAppControlCapabilityExposure() error = %q, want shared_storage rejection", err)
	}
}

func TestRequireApprovedBroadAppControlCapabilityOnboardingProposalFailsClosed(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 19, 10, 0, 0, 0, time.UTC)
	record := validCapabilityOnboardingProposalRecord(now, func(record *CapabilityOnboardingProposalRecord) {
		record.ProposalID = "proposal-broad-app-control"
		record.CapabilityName = BroadAppControlCapabilityName
		record.State = CapabilityOnboardingProposalStateProposed
	})
	if err := StoreCapabilityOnboardingProposalRecord(root, record); err != nil {
		t.Fatalf("StoreCapabilityOnboardingProposalRecord() error = %v", err)
	}

	job := Job{
		ID:               "job-broad-app-control",
		MaxAuthority:     AuthorityTierHigh,
		MissionStoreRoot: root,
		Plan: Plan{
			ID: "plan-broad-app-control",
			Steps: []Step{
				{
					ID:                   "build",
					Type:                 StepTypeOneShotCode,
					RequiredCapabilities: []string{BroadAppControlCapabilityName},
					CapabilityOnboardingProposalRef: &CapabilityOnboardingProposalRef{
						ProposalID: record.ProposalID,
					},
				},
				{
					ID:        "final",
					Type:      StepTypeFinalResponse,
					DependsOn: []string{"build"},
				},
			},
		},
	}

	ec, err := ResolveExecutionContext(job, "build")
	if err != nil {
		t.Fatalf("ResolveExecutionContext() error = %v", err)
	}
	ec.MissionStoreRoot = root

	_, err = RequireApprovedBroadAppControlCapabilityOnboardingProposal(ec)
	if err == nil {
		t.Fatal("RequireApprovedBroadAppControlCapabilityOnboardingProposal() error = nil, want fail-closed approval rejection")
	}
	if !strings.Contains(err.Error(), `requires approved capability onboarding proposal "proposal-broad-app-control", got state "proposed"`) {
		t.Fatalf("RequireApprovedBroadAppControlCapabilityOnboardingProposal() error = %q, want approval rejection", err)
	}
}
