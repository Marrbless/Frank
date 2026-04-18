package missioncontrol

import (
	"errors"
	"fmt"
	"strings"
)

const (
	NotificationsCapabilityName              = "notifications"
	NotificationsTelegramCapabilityID        = "notifications-telegram"
	NotificationsTelegramCapabilityValidator = "telegram owner-control channel confirmed"
	NotificationsTelegramCapabilityNotes     = "notifications capability exposed through the configured Telegram owner-control channel"
)

func StepRequiresNotificationsCapability(step Step) bool {
	for _, capability := range NormalizeStepRequiredCapabilities(step.RequiredCapabilities) {
		if capability == NotificationsCapabilityName {
			return true
		}
	}
	return false
}

func ResolveNotificationsCapabilityRecord(root string) (*CapabilityRecord, error) {
	record, err := ResolveCapabilityRecordByName(root, NotificationsCapabilityName)
	if err != nil {
		return nil, err
	}
	recordCopy := record
	return &recordCopy, nil
}

func RequireExposedNotificationsCapabilityRecord(root string) (*CapabilityRecord, error) {
	record, err := ResolveNotificationsCapabilityRecord(root)
	if err != nil {
		if errors.Is(err, ErrCapabilityRecordNotFound) {
			return nil, fmt.Errorf(`notifications capability requires one committed capability record named %q`, NotificationsCapabilityName)
		}
		return nil, err
	}
	if !record.Exposed {
		return nil, fmt.Errorf("notifications capability record %q is not exposed", record.CapabilityID)
	}
	return record, nil
}

func RequireApprovedNotificationsCapabilityOnboardingProposal(ec ExecutionContext) (*CapabilityOnboardingProposalRecord, error) {
	if ec.Step == nil {
		return nil, fmt.Errorf("execution context step is required")
	}
	if !StepRequiresNotificationsCapability(*ec.Step) {
		return nil, nil
	}

	proposal, err := ResolveExecutionContextCapabilityOnboardingProposal(ec)
	if err != nil {
		return nil, err
	}
	if proposal == nil {
		return nil, fmt.Errorf("execution context notifications capability requires capability onboarding proposal ref")
	}
	if strings.TrimSpace(proposal.CapabilityName) != NotificationsCapabilityName {
		return nil, fmt.Errorf(
			"execution context notifications capability requires capability onboarding proposal for %q, got %q",
			NotificationsCapabilityName,
			proposal.CapabilityName,
		)
	}
	if NormalizeCapabilityOnboardingProposalState(proposal.State) != CapabilityOnboardingProposalStateApproved {
		return nil, fmt.Errorf(
			"execution context notifications capability requires approved capability onboarding proposal %q, got state %q",
			proposal.ProposalID,
			proposal.State,
		)
	}

	proposalCopy := *proposal
	return &proposalCopy, nil
}

func StoreTelegramNotificationsCapabilityExposure(root string) (CapabilityRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return CapabilityRecord{}, err
	}

	existing, err := ResolveNotificationsCapabilityRecord(root)
	switch {
	case err == nil:
		if strings.TrimSpace(existing.CapabilityID) != NotificationsTelegramCapabilityID {
			return CapabilityRecord{}, fmt.Errorf(
				"notifications capability record %q is not telegram-specific",
				existing.CapabilityID,
			)
		}
	case errors.Is(err, ErrCapabilityRecordNotFound):
		existing = nil
	case err != nil:
		return CapabilityRecord{}, err
	}

	record := CapabilityRecord{
		CapabilityID:  NotificationsTelegramCapabilityID,
		Class:         NotificationsCapabilityName,
		Name:          NotificationsCapabilityName,
		Exposed:       true,
		AuthorityTier: AuthorityTierMedium,
		Validator:     NotificationsTelegramCapabilityValidator,
		Notes:         NotificationsTelegramCapabilityNotes,
	}
	if existing != nil {
		record.RecordVersion = existing.RecordVersion
	}
	if err := StoreCapabilityRecord(root, record); err != nil {
		return CapabilityRecord{}, err
	}
	return LoadCapabilityRecord(root, NotificationsTelegramCapabilityID)
}
