package missioncontrol

import (
	"strings"
	"testing"
	"time"
)

func TestStoreWorkspaceCameraCapabilityExposureCreatesSourceAndCapabilityRecords(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	workspace := t.TempDir()
	if _, err := StoreWorkspaceSharedStorageCapabilityExposure(root, workspace); err != nil {
		t.Fatalf("StoreWorkspaceSharedStorageCapabilityExposure() error = %v", err)
	}

	if _, err := StoreWorkspaceCameraCapabilityExposure(root, workspace); err != nil {
		t.Fatalf("StoreWorkspaceCameraCapabilityExposure(create) error = %v", err)
	}

	record, err := RequireExposedCameraCapabilityRecord(root)
	if err != nil {
		t.Fatalf("RequireExposedCameraCapabilityRecord() error = %v", err)
	}
	if record.CapabilityID != CameraLocalFileCapabilityID {
		t.Fatalf("CapabilityID = %q, want %q", record.CapabilityID, CameraLocalFileCapabilityID)
	}
	if record.Name != CameraCapabilityName {
		t.Fatalf("Name = %q, want %q", record.Name, CameraCapabilityName)
	}
	if !record.Exposed {
		t.Fatal("Exposed = false, want true")
	}

	source, err := RequireReadableCameraSourceRecord(root, workspace)
	if err != nil {
		t.Fatalf("RequireReadableCameraSourceRecord() error = %v", err)
	}
	if source.Path != CameraLocalFileDefaultPath {
		t.Fatalf("Path = %q, want %q", source.Path, CameraLocalFileDefaultPath)
	}
}

func TestStoreWorkspaceCameraCapabilityExposureFailsClosedWithoutSharedStorageExposure(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	workspace := t.TempDir()

	_, err := StoreWorkspaceCameraCapabilityExposure(root, workspace)
	if err == nil {
		t.Fatal("StoreWorkspaceCameraCapabilityExposure() error = nil, want shared_storage rejection")
	}
	if !strings.Contains(err.Error(), `camera capability requires shared_storage exposure: shared_storage capability requires one committed capability record named "shared_storage"`) {
		t.Fatalf("StoreWorkspaceCameraCapabilityExposure() error = %q, want shared_storage rejection", err)
	}
}

func TestRequireApprovedCameraCapabilityOnboardingProposalFailsClosed(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 19, 2, 0, 0, 0, time.UTC)
	record := validCapabilityOnboardingProposalRecord(now, func(record *CapabilityOnboardingProposalRecord) {
		record.ProposalID = "proposal-camera"
		record.CapabilityName = CameraCapabilityName
		record.State = CapabilityOnboardingProposalStateProposed
	})
	if err := StoreCapabilityOnboardingProposalRecord(root, record); err != nil {
		t.Fatalf("StoreCapabilityOnboardingProposalRecord() error = %v", err)
	}

	job := Job{
		ID:               "job-camera",
		MaxAuthority:     AuthorityTierHigh,
		MissionStoreRoot: root,
		Plan: Plan{
			ID: "plan-camera",
			Steps: []Step{
				{
					ID:                   "build",
					Type:                 StepTypeOneShotCode,
					RequiredCapabilities: []string{CameraCapabilityName},
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

	_, err = RequireApprovedCameraCapabilityOnboardingProposal(ec)
	if err == nil {
		t.Fatal("RequireApprovedCameraCapabilityOnboardingProposal() error = nil, want fail-closed approval rejection")
	}
	if !strings.Contains(err.Error(), `requires approved capability onboarding proposal "proposal-camera", got state "proposed"`) {
		t.Fatalf("RequireApprovedCameraCapabilityOnboardingProposal() error = %q, want approval rejection", err)
	}
}
