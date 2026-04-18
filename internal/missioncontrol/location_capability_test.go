package missioncontrol

import (
	"strings"
	"testing"
	"time"
)

func TestStoreWorkspaceLocationCapabilityExposureCreatesSourceAndCapabilityRecords(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	workspace := t.TempDir()
	if _, err := StoreWorkspaceSharedStorageCapabilityExposure(root, workspace); err != nil {
		t.Fatalf("StoreWorkspaceSharedStorageCapabilityExposure() error = %v", err)
	}

	if _, err := StoreWorkspaceLocationCapabilityExposure(root, workspace); err != nil {
		t.Fatalf("StoreWorkspaceLocationCapabilityExposure(create) error = %v", err)
	}

	record, err := RequireExposedLocationCapabilityRecord(root)
	if err != nil {
		t.Fatalf("RequireExposedLocationCapabilityRecord() error = %v", err)
	}
	if record.CapabilityID != LocationLocalFileCapabilityID {
		t.Fatalf("CapabilityID = %q, want %q", record.CapabilityID, LocationLocalFileCapabilityID)
	}
	if record.Name != LocationCapabilityName {
		t.Fatalf("Name = %q, want %q", record.Name, LocationCapabilityName)
	}
	if !record.Exposed {
		t.Fatal("Exposed = false, want true")
	}

	source, err := RequireReadableLocationSourceRecord(root, workspace)
	if err != nil {
		t.Fatalf("RequireReadableLocationSourceRecord() error = %v", err)
	}
	if source.Path != LocationLocalFileDefaultPath {
		t.Fatalf("Path = %q, want %q", source.Path, LocationLocalFileDefaultPath)
	}
}

func TestStoreWorkspaceLocationCapabilityExposureFailsClosedWithoutSharedStorageExposure(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	workspace := t.TempDir()

	_, err := StoreWorkspaceLocationCapabilityExposure(root, workspace)
	if err == nil {
		t.Fatal("StoreWorkspaceLocationCapabilityExposure() error = nil, want shared_storage rejection")
	}
	if !strings.Contains(err.Error(), `location capability requires shared_storage exposure: shared_storage capability requires one committed capability record named "shared_storage"`) {
		t.Fatalf("StoreWorkspaceLocationCapabilityExposure() error = %q, want shared_storage rejection", err)
	}
}

func TestRequireApprovedLocationCapabilityOnboardingProposalFailsClosed(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 19, 0, 0, 0, 0, time.UTC)
	record := validCapabilityOnboardingProposalRecord(now, func(record *CapabilityOnboardingProposalRecord) {
		record.ProposalID = "proposal-location"
		record.CapabilityName = LocationCapabilityName
		record.State = CapabilityOnboardingProposalStateProposed
	})
	if err := StoreCapabilityOnboardingProposalRecord(root, record); err != nil {
		t.Fatalf("StoreCapabilityOnboardingProposalRecord() error = %v", err)
	}

	job := Job{
		ID:               "job-location",
		MaxAuthority:     AuthorityTierHigh,
		MissionStoreRoot: root,
		Plan: Plan{
			ID: "plan-location",
			Steps: []Step{
				{
					ID:                   "build",
					Type:                 StepTypeOneShotCode,
					RequiredCapabilities: []string{LocationCapabilityName},
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

	_, err = RequireApprovedLocationCapabilityOnboardingProposal(ec)
	if err == nil {
		t.Fatal("RequireApprovedLocationCapabilityOnboardingProposal() error = nil, want fail-closed approval rejection")
	}
	if !strings.Contains(err.Error(), `requires approved capability onboarding proposal "proposal-location", got state "proposed"`) {
		t.Fatalf("RequireApprovedLocationCapabilityOnboardingProposal() error = %q, want approval rejection", err)
	}
}
