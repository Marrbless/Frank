package missioncontrol

import (
	"strings"
	"testing"
	"time"
)

func TestStoreWorkspaceContactsCapabilityExposureCreatesSourceAndCapabilityRecords(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	workspace := t.TempDir()
	if _, err := StoreWorkspaceSharedStorageCapabilityExposure(root, workspace); err != nil {
		t.Fatalf("StoreWorkspaceSharedStorageCapabilityExposure() error = %v", err)
	}

	if _, err := StoreWorkspaceContactsCapabilityExposure(root, workspace); err != nil {
		t.Fatalf("StoreWorkspaceContactsCapabilityExposure(create) error = %v", err)
	}

	record, err := RequireExposedContactsCapabilityRecord(root)
	if err != nil {
		t.Fatalf("RequireExposedContactsCapabilityRecord() error = %v", err)
	}
	if record.CapabilityID != ContactsLocalFileCapabilityID {
		t.Fatalf("CapabilityID = %q, want %q", record.CapabilityID, ContactsLocalFileCapabilityID)
	}
	if record.Name != ContactsCapabilityName {
		t.Fatalf("Name = %q, want %q", record.Name, ContactsCapabilityName)
	}
	if !record.Exposed {
		t.Fatal("Exposed = false, want true")
	}

	source, err := RequireReadableContactsSourceRecord(root, workspace)
	if err != nil {
		t.Fatalf("RequireReadableContactsSourceRecord() error = %v", err)
	}
	if source.Path != ContactsLocalFileDefaultPath {
		t.Fatalf("Path = %q, want %q", source.Path, ContactsLocalFileDefaultPath)
	}
}

func TestStoreWorkspaceContactsCapabilityExposureFailsClosedWithoutSharedStorageExposure(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	workspace := t.TempDir()

	_, err := StoreWorkspaceContactsCapabilityExposure(root, workspace)
	if err == nil {
		t.Fatal("StoreWorkspaceContactsCapabilityExposure() error = nil, want shared_storage rejection")
	}
	if !strings.Contains(err.Error(), `contacts capability requires shared_storage exposure: shared_storage capability requires one committed capability record named "shared_storage"`) {
		t.Fatalf("StoreWorkspaceContactsCapabilityExposure() error = %q, want shared_storage rejection", err)
	}
}

func TestRequireApprovedContactsCapabilityOnboardingProposalFailsClosed(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 18, 21, 0, 0, 0, time.UTC)
	record := validCapabilityOnboardingProposalRecord(now, func(record *CapabilityOnboardingProposalRecord) {
		record.ProposalID = "proposal-contacts"
		record.CapabilityName = ContactsCapabilityName
		record.State = CapabilityOnboardingProposalStateProposed
	})
	if err := StoreCapabilityOnboardingProposalRecord(root, record); err != nil {
		t.Fatalf("StoreCapabilityOnboardingProposalRecord() error = %v", err)
	}

	job := Job{
		ID:               "job-contacts",
		MaxAuthority:     AuthorityTierHigh,
		MissionStoreRoot: root,
		Plan: Plan{
			ID: "plan-contacts",
			Steps: []Step{
				{
					ID:                   "build",
					Type:                 StepTypeOneShotCode,
					RequiredCapabilities: []string{ContactsCapabilityName},
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

	_, err = RequireApprovedContactsCapabilityOnboardingProposal(ec)
	if err == nil {
		t.Fatal("RequireApprovedContactsCapabilityOnboardingProposal() error = nil, want fail-closed approval rejection")
	}
	if !strings.Contains(err.Error(), `requires approved capability onboarding proposal "proposal-contacts", got state "proposed"`) {
		t.Fatalf("RequireApprovedContactsCapabilityOnboardingProposal() error = %q, want approval rejection", err)
	}
}
