package missioncontrol

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestStoreWorkspaceSharedStorageCapabilityExposureCreatesWorkspaceAndUpdatesCommittedRecord(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	workspace := filepath.Join(t.TempDir(), "workspace")
	if _, err := StoreWorkspaceSharedStorageCapabilityExposure(root, workspace); err != nil {
		t.Fatalf("StoreWorkspaceSharedStorageCapabilityExposure(create) error = %v", err)
	}

	record, err := RequireExposedSharedStorageCapabilityRecord(root)
	if err != nil {
		t.Fatalf("RequireExposedSharedStorageCapabilityRecord() error = %v", err)
	}
	if record.CapabilityID != SharedStorageWorkspaceCapabilityID {
		t.Fatalf("CapabilityID = %q, want %q", record.CapabilityID, SharedStorageWorkspaceCapabilityID)
	}
	if record.Name != SharedStorageCapabilityName {
		t.Fatalf("Name = %q, want %q", record.Name, SharedStorageCapabilityName)
	}
	if !record.Exposed {
		t.Fatal("Exposed = false, want true")
	}
	if _, err := os.Stat(filepath.Join(workspace, "SOUL.md")); err != nil {
		t.Fatalf("Stat(SOUL.md) error = %v", err)
	}

	if err := StoreCapabilityRecord(root, CapabilityRecord{
		CapabilityID:  SharedStorageWorkspaceCapabilityID,
		Class:         SharedStorageCapabilityName,
		Name:          SharedStorageCapabilityName,
		Exposed:       false,
		AuthorityTier: AuthorityTierMedium,
		Validator:     SharedStorageWorkspaceCapabilityValidator,
		Notes:         "shared_storage capability temporarily unexposed",
	}); err != nil {
		t.Fatalf("StoreCapabilityRecord(unexposed) error = %v", err)
	}

	updated, err := StoreWorkspaceSharedStorageCapabilityExposure(root, workspace)
	if err != nil {
		t.Fatalf("StoreWorkspaceSharedStorageCapabilityExposure(update) error = %v", err)
	}
	if !updated.Exposed {
		t.Fatal("updated Exposed = false, want true")
	}
	if updated.Notes != SharedStorageWorkspaceCapabilityNotes {
		t.Fatalf("updated Notes = %q, want %q", updated.Notes, SharedStorageWorkspaceCapabilityNotes)
	}
}

func TestStoreWorkspaceSharedStorageCapabilityExposureFailsClosedOnNonWorkspaceRecord(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if err := StoreCapabilityRecord(root, CapabilityRecord{
		CapabilityID:  "shared_storage-remote",
		Class:         SharedStorageCapabilityName,
		Name:          SharedStorageCapabilityName,
		Exposed:       true,
		AuthorityTier: AuthorityTierMedium,
		Validator:     "remote storage shared confirmed",
		Notes:         "shared_storage capability exposed through remote storage",
	}); err != nil {
		t.Fatalf("StoreCapabilityRecord() error = %v", err)
	}

	_, err := StoreWorkspaceSharedStorageCapabilityExposure(root, filepath.Join(t.TempDir(), "workspace"))
	if err == nil {
		t.Fatal("StoreWorkspaceSharedStorageCapabilityExposure() error = nil, want fail-closed non-workspace rejection")
	}
	if !strings.Contains(err.Error(), `shared_storage capability record "shared_storage-remote" is not workspace-specific`) {
		t.Fatalf("StoreWorkspaceSharedStorageCapabilityExposure() error = %q, want non-workspace rejection", err)
	}
}

func TestRequireApprovedSharedStorageCapabilityOnboardingProposalFailsClosed(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 18, 19, 0, 0, 0, time.UTC)
	record := validCapabilityOnboardingProposalRecord(now, func(record *CapabilityOnboardingProposalRecord) {
		record.ProposalID = "proposal-shared-storage"
		record.CapabilityName = SharedStorageCapabilityName
		record.State = CapabilityOnboardingProposalStateProposed
	})
	if err := StoreCapabilityOnboardingProposalRecord(root, record); err != nil {
		t.Fatalf("StoreCapabilityOnboardingProposalRecord() error = %v", err)
	}

	job := Job{
		ID:               "job-shared-storage",
		MaxAuthority:     AuthorityTierHigh,
		MissionStoreRoot: root,
		Plan: Plan{
			ID: "plan-shared-storage",
			Steps: []Step{
				{
					ID:                   "build",
					Type:                 StepTypeOneShotCode,
					RequiredCapabilities: []string{SharedStorageCapabilityName},
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

	_, err = RequireApprovedSharedStorageCapabilityOnboardingProposal(ec)
	if err == nil {
		t.Fatal("RequireApprovedSharedStorageCapabilityOnboardingProposal() error = nil, want fail-closed approval rejection")
	}
	if !strings.Contains(err.Error(), `requires approved capability onboarding proposal "proposal-shared-storage", got state "proposed"`) {
		t.Fatalf("RequireApprovedSharedStorageCapabilityOnboardingProposal() error = %q, want approval rejection", err)
	}
}
