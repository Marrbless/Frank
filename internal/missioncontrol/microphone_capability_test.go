package missioncontrol

import (
	"strings"
	"testing"
	"time"
)

func TestStoreWorkspaceMicrophoneCapabilityExposureCreatesSourceAndCapabilityRecords(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	workspace := t.TempDir()
	if _, err := StoreWorkspaceSharedStorageCapabilityExposure(root, workspace); err != nil {
		t.Fatalf("StoreWorkspaceSharedStorageCapabilityExposure() error = %v", err)
	}

	if _, err := StoreWorkspaceMicrophoneCapabilityExposure(root, workspace); err != nil {
		t.Fatalf("StoreWorkspaceMicrophoneCapabilityExposure(create) error = %v", err)
	}

	record, err := RequireExposedMicrophoneCapabilityRecord(root)
	if err != nil {
		t.Fatalf("RequireExposedMicrophoneCapabilityRecord() error = %v", err)
	}
	if record.CapabilityID != MicrophoneLocalFileCapabilityID {
		t.Fatalf("CapabilityID = %q, want %q", record.CapabilityID, MicrophoneLocalFileCapabilityID)
	}
	if record.Name != MicrophoneCapabilityName {
		t.Fatalf("Name = %q, want %q", record.Name, MicrophoneCapabilityName)
	}
	if !record.Exposed {
		t.Fatal("Exposed = false, want true")
	}

	source, err := RequireReadableMicrophoneSourceRecord(root, workspace)
	if err != nil {
		t.Fatalf("RequireReadableMicrophoneSourceRecord() error = %v", err)
	}
	if source.Path != MicrophoneLocalFileDefaultPath {
		t.Fatalf("Path = %q, want %q", source.Path, MicrophoneLocalFileDefaultPath)
	}
}

func TestStoreWorkspaceMicrophoneCapabilityExposureFailsClosedWithoutSharedStorageExposure(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	workspace := t.TempDir()

	_, err := StoreWorkspaceMicrophoneCapabilityExposure(root, workspace)
	if err == nil {
		t.Fatal("StoreWorkspaceMicrophoneCapabilityExposure() error = nil, want shared_storage rejection")
	}
	if !strings.Contains(err.Error(), `microphone capability requires shared_storage exposure: shared_storage capability requires one committed capability record named "shared_storage"`) {
		t.Fatalf("StoreWorkspaceMicrophoneCapabilityExposure() error = %q, want shared_storage rejection", err)
	}
}

func TestRequireApprovedMicrophoneCapabilityOnboardingProposalFailsClosed(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 19, 4, 0, 0, 0, time.UTC)
	record := validCapabilityOnboardingProposalRecord(now, func(record *CapabilityOnboardingProposalRecord) {
		record.ProposalID = "proposal-microphone"
		record.CapabilityName = MicrophoneCapabilityName
		record.State = CapabilityOnboardingProposalStateProposed
	})
	if err := StoreCapabilityOnboardingProposalRecord(root, record); err != nil {
		t.Fatalf("StoreCapabilityOnboardingProposalRecord() error = %v", err)
	}

	job := Job{
		ID:               "job-microphone",
		MaxAuthority:     AuthorityTierHigh,
		MissionStoreRoot: root,
		Plan: Plan{
			ID: "plan-microphone",
			Steps: []Step{
				{
					ID:                   "build",
					Type:                 StepTypeOneShotCode,
					RequiredCapabilities: []string{MicrophoneCapabilityName},
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

	_, err = RequireApprovedMicrophoneCapabilityOnboardingProposal(ec)
	if err == nil {
		t.Fatal("RequireApprovedMicrophoneCapabilityOnboardingProposal() error = nil, want fail-closed approval rejection")
	}
	if !strings.Contains(err.Error(), `requires approved capability onboarding proposal "proposal-microphone", got state "proposed"`) {
		t.Fatalf("RequireApprovedMicrophoneCapabilityOnboardingProposal() error = %q, want approval rejection", err)
	}
}
