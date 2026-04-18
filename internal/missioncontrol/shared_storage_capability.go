package missioncontrol

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/local/picobot/internal/config"
)

const (
	SharedStorageCapabilityName               = "shared_storage"
	SharedStorageWorkspaceCapabilityID        = "shared_storage-workspace"
	SharedStorageWorkspaceCapabilityValidator = "configured workspace root initialized and writable"
	SharedStorageWorkspaceCapabilityNotes     = "shared_storage capability exposed through the configured local workspace root"
)

func StepRequiresSharedStorageCapability(step Step) bool {
	for _, capability := range NormalizeStepRequiredCapabilities(step.RequiredCapabilities) {
		if capability == SharedStorageCapabilityName {
			return true
		}
	}
	return false
}

func ResolveSharedStorageCapabilityRecord(root string) (*CapabilityRecord, error) {
	record, err := ResolveCapabilityRecordByName(root, SharedStorageCapabilityName)
	if err != nil {
		return nil, err
	}
	recordCopy := record
	return &recordCopy, nil
}

func RequireExposedSharedStorageCapabilityRecord(root string) (*CapabilityRecord, error) {
	record, err := ResolveSharedStorageCapabilityRecord(root)
	if err != nil {
		if errors.Is(err, ErrCapabilityRecordNotFound) {
			return nil, fmt.Errorf(`shared_storage capability requires one committed capability record named %q`, SharedStorageCapabilityName)
		}
		return nil, err
	}
	if !record.Exposed {
		return nil, fmt.Errorf("shared_storage capability record %q is not exposed", record.CapabilityID)
	}
	return record, nil
}

func RequireApprovedSharedStorageCapabilityOnboardingProposal(ec ExecutionContext) (*CapabilityOnboardingProposalRecord, error) {
	if ec.Step == nil {
		return nil, fmt.Errorf("execution context step is required")
	}
	if !StepRequiresSharedStorageCapability(*ec.Step) {
		return nil, nil
	}

	proposal, err := ResolveExecutionContextCapabilityOnboardingProposal(ec)
	if err != nil {
		return nil, err
	}
	if proposal == nil {
		return nil, fmt.Errorf("execution context shared_storage capability requires capability onboarding proposal ref")
	}
	if strings.TrimSpace(proposal.CapabilityName) != SharedStorageCapabilityName {
		return nil, fmt.Errorf(
			"execution context shared_storage capability requires capability onboarding proposal for %q, got %q",
			SharedStorageCapabilityName,
			proposal.CapabilityName,
		)
	}
	if NormalizeCapabilityOnboardingProposalState(proposal.State) != CapabilityOnboardingProposalStateApproved {
		return nil, fmt.Errorf(
			"execution context shared_storage capability requires approved capability onboarding proposal %q, got state %q",
			proposal.ProposalID,
			proposal.State,
		)
	}

	proposalCopy := *proposal
	return &proposalCopy, nil
}

func StoreWorkspaceSharedStorageCapabilityExposure(root string, workspacePath string) (CapabilityRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return CapabilityRecord{}, err
	}

	resolvedWorkspacePath, err := resolveSharedStorageWorkspacePath(workspacePath)
	if err != nil {
		return CapabilityRecord{}, err
	}
	if err := config.InitializeWorkspace(resolvedWorkspacePath); err != nil {
		return CapabilityRecord{}, fmt.Errorf("shared_storage capability exposure requires initialized workspace root: %w", err)
	}

	existing, err := ResolveSharedStorageCapabilityRecord(root)
	switch {
	case err == nil:
		if strings.TrimSpace(existing.CapabilityID) != SharedStorageWorkspaceCapabilityID {
			return CapabilityRecord{}, fmt.Errorf(
				"shared_storage capability record %q is not workspace-specific",
				existing.CapabilityID,
			)
		}
	case errors.Is(err, ErrCapabilityRecordNotFound):
		existing = nil
	case err != nil:
		return CapabilityRecord{}, err
	}

	record := CapabilityRecord{
		CapabilityID:  SharedStorageWorkspaceCapabilityID,
		Class:         SharedStorageCapabilityName,
		Name:          SharedStorageCapabilityName,
		Exposed:       true,
		AuthorityTier: AuthorityTierMedium,
		Validator:     SharedStorageWorkspaceCapabilityValidator,
		Notes:         SharedStorageWorkspaceCapabilityNotes,
	}
	if existing != nil {
		record.RecordVersion = existing.RecordVersion
	}
	if err := StoreCapabilityRecord(root, record); err != nil {
		return CapabilityRecord{}, err
	}
	return LoadCapabilityRecord(root, SharedStorageWorkspaceCapabilityID)
}

func resolveSharedStorageWorkspacePath(workspacePath string) (string, error) {
	trimmed := strings.TrimSpace(workspacePath)
	if trimmed == "" {
		return "", fmt.Errorf("shared_storage capability exposure requires configured workspace root")
	}

	if trimmed == "~" || strings.HasPrefix(trimmed, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("shared_storage capability exposure requires resolvable home directory: %w", err)
		}
		if trimmed == "~" {
			trimmed = home
		} else {
			trimmed = filepath.Join(home, trimmed[2:])
		}
	}

	resolved, err := filepath.Abs(trimmed)
	if err != nil {
		return "", fmt.Errorf("shared_storage capability exposure requires resolvable workspace root: %w", err)
	}
	return resolved, nil
}
