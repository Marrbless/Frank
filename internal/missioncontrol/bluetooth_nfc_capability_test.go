package missioncontrol

import (
	"strings"
	"testing"
	"time"
)

func TestStoreWorkspaceBluetoothNFCCapabilityExposureCreatesSourceAndCapabilityRecords(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	workspace := t.TempDir()
	if _, err := StoreWorkspaceSharedStorageCapabilityExposure(root, workspace); err != nil {
		t.Fatalf("StoreWorkspaceSharedStorageCapabilityExposure() error = %v", err)
	}

	if _, err := StoreWorkspaceBluetoothNFCCapabilityExposure(root, workspace); err != nil {
		t.Fatalf("StoreWorkspaceBluetoothNFCCapabilityExposure(create) error = %v", err)
	}

	record, err := RequireExposedBluetoothNFCCapabilityRecord(root)
	if err != nil {
		t.Fatalf("RequireExposedBluetoothNFCCapabilityRecord() error = %v", err)
	}
	if record.CapabilityID != BluetoothNFCLocalFileCapabilityID {
		t.Fatalf("CapabilityID = %q, want %q", record.CapabilityID, BluetoothNFCLocalFileCapabilityID)
	}
	if record.Name != BluetoothNFCCapabilityName {
		t.Fatalf("Name = %q, want %q", record.Name, BluetoothNFCCapabilityName)
	}
	if !record.Exposed {
		t.Fatal("Exposed = false, want true")
	}

	source, err := RequireReadableBluetoothNFCSourceRecord(root, workspace)
	if err != nil {
		t.Fatalf("RequireReadableBluetoothNFCSourceRecord() error = %v", err)
	}
	if source.Path != BluetoothNFCLocalFileDefaultPath {
		t.Fatalf("Path = %q, want %q", source.Path, BluetoothNFCLocalFileDefaultPath)
	}
}

func TestStoreWorkspaceBluetoothNFCCapabilityExposureFailsClosedWithoutSharedStorageExposure(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	workspace := t.TempDir()

	_, err := StoreWorkspaceBluetoothNFCCapabilityExposure(root, workspace)
	if err == nil {
		t.Fatal("StoreWorkspaceBluetoothNFCCapabilityExposure() error = nil, want shared_storage rejection")
	}
	if !strings.Contains(err.Error(), `bluetooth_nfc capability requires shared_storage exposure: shared_storage capability requires one committed capability record named "shared_storage"`) {
		t.Fatalf("StoreWorkspaceBluetoothNFCCapabilityExposure() error = %q, want shared_storage rejection", err)
	}
}

func TestRequireApprovedBluetoothNFCCapabilityOnboardingProposalFailsClosed(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 19, 10, 0, 0, 0, time.UTC)
	record := validCapabilityOnboardingProposalRecord(now, func(record *CapabilityOnboardingProposalRecord) {
		record.ProposalID = "proposal-bluetooth-nfc"
		record.CapabilityName = BluetoothNFCCapabilityName
		record.State = CapabilityOnboardingProposalStateProposed
	})
	if err := StoreCapabilityOnboardingProposalRecord(root, record); err != nil {
		t.Fatalf("StoreCapabilityOnboardingProposalRecord() error = %v", err)
	}

	job := Job{
		ID:               "job-bluetooth-nfc",
		MaxAuthority:     AuthorityTierHigh,
		MissionStoreRoot: root,
		Plan: Plan{
			ID: "plan-bluetooth-nfc",
			Steps: []Step{
				{
					ID:                   "build",
					Type:                 StepTypeOneShotCode,
					RequiredCapabilities: []string{BluetoothNFCCapabilityName},
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

	_, err = RequireApprovedBluetoothNFCCapabilityOnboardingProposal(ec)
	if err == nil {
		t.Fatal("RequireApprovedBluetoothNFCCapabilityOnboardingProposal() error = nil, want fail-closed approval rejection")
	}
	if !strings.Contains(err.Error(), `requires approved capability onboarding proposal "proposal-bluetooth-nfc", got state "proposed"`) {
		t.Fatalf("RequireApprovedBluetoothNFCCapabilityOnboardingProposal() error = %q, want approval rejection", err)
	}
}
