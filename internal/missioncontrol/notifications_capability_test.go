package missioncontrol

import (
	"strings"
	"testing"
	"time"
)

func TestStoreTelegramNotificationsCapabilityExposureCreatesAndUpdatesCommittedRecord(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if _, err := StoreTelegramNotificationsCapabilityExposure(root); err != nil {
		t.Fatalf("StoreTelegramNotificationsCapabilityExposure(create) error = %v", err)
	}

	record, err := RequireExposedNotificationsCapabilityRecord(root)
	if err != nil {
		t.Fatalf("RequireExposedNotificationsCapabilityRecord() error = %v", err)
	}
	if record.CapabilityID != NotificationsTelegramCapabilityID {
		t.Fatalf("CapabilityID = %q, want %q", record.CapabilityID, NotificationsTelegramCapabilityID)
	}
	if record.Name != NotificationsCapabilityName {
		t.Fatalf("Name = %q, want %q", record.Name, NotificationsCapabilityName)
	}
	if !record.Exposed {
		t.Fatal("Exposed = false, want true")
	}

	if err := StoreCapabilityRecord(root, CapabilityRecord{
		CapabilityID:  NotificationsTelegramCapabilityID,
		Class:         NotificationsCapabilityName,
		Name:          NotificationsCapabilityName,
		Exposed:       false,
		AuthorityTier: AuthorityTierMedium,
		Validator:     NotificationsTelegramCapabilityValidator,
		Notes:         "notifications capability temporarily unexposed",
	}); err != nil {
		t.Fatalf("StoreCapabilityRecord(unexposed) error = %v", err)
	}

	updated, err := StoreTelegramNotificationsCapabilityExposure(root)
	if err != nil {
		t.Fatalf("StoreTelegramNotificationsCapabilityExposure(update) error = %v", err)
	}
	if !updated.Exposed {
		t.Fatal("updated Exposed = false, want true")
	}
	if updated.Notes != NotificationsTelegramCapabilityNotes {
		t.Fatalf("updated Notes = %q, want %q", updated.Notes, NotificationsTelegramCapabilityNotes)
	}
}

func TestStoreTelegramNotificationsCapabilityExposureFailsClosedOnNonTelegramRecord(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if err := StoreCapabilityRecord(root, CapabilityRecord{
		CapabilityID:  "notifications-discord",
		Class:         NotificationsCapabilityName,
		Name:          NotificationsCapabilityName,
		Exposed:       true,
		AuthorityTier: AuthorityTierMedium,
		Validator:     "discord owner-control channel confirmed",
		Notes:         "notifications capability exposed through discord",
	}); err != nil {
		t.Fatalf("StoreCapabilityRecord() error = %v", err)
	}

	_, err := StoreTelegramNotificationsCapabilityExposure(root)
	if err == nil {
		t.Fatal("StoreTelegramNotificationsCapabilityExposure() error = nil, want fail-closed non-telegram rejection")
	}
	if !strings.Contains(err.Error(), `notifications capability record "notifications-discord" is not telegram-specific`) {
		t.Fatalf("StoreTelegramNotificationsCapabilityExposure() error = %q, want non-telegram rejection", err)
	}
}

func TestRequireApprovedNotificationsCapabilityOnboardingProposalFailsClosed(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 18, 15, 0, 0, 0, time.UTC)
	record := validCapabilityOnboardingProposalRecord(now, func(record *CapabilityOnboardingProposalRecord) {
		record.ProposalID = "proposal-notifications"
		record.CapabilityName = NotificationsCapabilityName
		record.State = CapabilityOnboardingProposalStateProposed
	})
	if err := StoreCapabilityOnboardingProposalRecord(root, record); err != nil {
		t.Fatalf("StoreCapabilityOnboardingProposalRecord() error = %v", err)
	}

	job := Job{
		ID:               "job-notifications",
		MaxAuthority:     AuthorityTierHigh,
		MissionStoreRoot: root,
		Plan: Plan{
			ID: "plan-notifications",
			Steps: []Step{
				{
					ID:                   "build",
					Type:                 StepTypeOneShotCode,
					RequiredCapabilities: []string{NotificationsCapabilityName},
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

	_, err = RequireApprovedNotificationsCapabilityOnboardingProposal(ec)
	if err == nil {
		t.Fatal("RequireApprovedNotificationsCapabilityOnboardingProposal() error = nil, want fail-closed approval rejection")
	}
	if !strings.Contains(err.Error(), `requires approved capability onboarding proposal "proposal-notifications", got state "proposed"`) {
		t.Fatalf("RequireApprovedNotificationsCapabilityOnboardingProposal() error = %q, want approval rejection", err)
	}
}
