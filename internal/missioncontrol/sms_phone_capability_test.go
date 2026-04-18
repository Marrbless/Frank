package missioncontrol

import (
	"strings"
	"testing"
	"time"
)

func TestStoreWorkspaceSMSPhoneCapabilityExposureCreatesSourceAndCapabilityRecords(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	workspace := t.TempDir()
	if _, err := StoreWorkspaceSharedStorageCapabilityExposure(root, workspace); err != nil {
		t.Fatalf("StoreWorkspaceSharedStorageCapabilityExposure() error = %v", err)
	}

	if _, err := StoreWorkspaceSMSPhoneCapabilityExposure(root, workspace); err != nil {
		t.Fatalf("StoreWorkspaceSMSPhoneCapabilityExposure(create) error = %v", err)
	}

	record, err := RequireExposedSMSPhoneCapabilityRecord(root)
	if err != nil {
		t.Fatalf("RequireExposedSMSPhoneCapabilityRecord() error = %v", err)
	}
	if record.CapabilityID != SMSPhoneLocalFileCapabilityID {
		t.Fatalf("CapabilityID = %q, want %q", record.CapabilityID, SMSPhoneLocalFileCapabilityID)
	}
	if record.Name != SMSPhoneCapabilityName {
		t.Fatalf("Name = %q, want %q", record.Name, SMSPhoneCapabilityName)
	}
	if !record.Exposed {
		t.Fatal("Exposed = false, want true")
	}

	source, err := RequireReadableSMSPhoneSourceRecord(root, workspace)
	if err != nil {
		t.Fatalf("RequireReadableSMSPhoneSourceRecord() error = %v", err)
	}
	if source.Path != SMSPhoneLocalFileDefaultPath {
		t.Fatalf("Path = %q, want %q", source.Path, SMSPhoneLocalFileDefaultPath)
	}
}

func TestStoreWorkspaceSMSPhoneCapabilityExposureFailsClosedWithoutSharedStorageExposure(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	workspace := t.TempDir()

	_, err := StoreWorkspaceSMSPhoneCapabilityExposure(root, workspace)
	if err == nil {
		t.Fatal("StoreWorkspaceSMSPhoneCapabilityExposure() error = nil, want shared_storage rejection")
	}
	if !strings.Contains(err.Error(), `sms_phone capability requires shared_storage exposure: shared_storage capability requires one committed capability record named "shared_storage"`) {
		t.Fatalf("StoreWorkspaceSMSPhoneCapabilityExposure() error = %q, want shared_storage rejection", err)
	}
}

func TestRequireApprovedSMSPhoneCapabilityOnboardingProposalFailsClosed(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 19, 7, 0, 0, 0, time.UTC)
	record := validCapabilityOnboardingProposalRecord(now, func(record *CapabilityOnboardingProposalRecord) {
		record.ProposalID = "proposal-sms-phone"
		record.CapabilityName = SMSPhoneCapabilityName
		record.State = CapabilityOnboardingProposalStateProposed
	})
	if err := StoreCapabilityOnboardingProposalRecord(root, record); err != nil {
		t.Fatalf("StoreCapabilityOnboardingProposalRecord() error = %v", err)
	}

	job := Job{
		ID:               "job-sms-phone",
		MaxAuthority:     AuthorityTierHigh,
		MissionStoreRoot: root,
		Plan: Plan{
			ID: "plan-sms-phone",
			Steps: []Step{
				{
					ID:                   "build",
					Type:                 StepTypeOneShotCode,
					RequiredCapabilities: []string{SMSPhoneCapabilityName},
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

	_, err = RequireApprovedSMSPhoneCapabilityOnboardingProposal(ec)
	if err == nil {
		t.Fatal("RequireApprovedSMSPhoneCapabilityOnboardingProposal() error = nil, want fail-closed approval rejection")
	}
	if !strings.Contains(err.Error(), `requires approved capability onboarding proposal "proposal-sms-phone", got state "proposed"`) {
		t.Fatalf("RequireApprovedSMSPhoneCapabilityOnboardingProposal() error = %q, want approval rejection", err)
	}
}
