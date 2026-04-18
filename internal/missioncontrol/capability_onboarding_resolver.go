package missioncontrol

import (
	"fmt"
	"strings"
)

func ValidateCapabilityOnboardingProposalRef(ref CapabilityOnboardingProposalRef) error {
	normalized := NormalizeCapabilityOnboardingProposalRef(ref)
	if err := validateCapabilityOnboardingProposalIDValue(normalized.ProposalID); err != nil {
		return fmt.Errorf("capability onboarding proposal ref is invalid: %w", err)
	}
	return nil
}

func ResolveCapabilityOnboardingProposalRef(root string, ref CapabilityOnboardingProposalRef) (CapabilityOnboardingProposalRecord, error) {
	normalized := NormalizeCapabilityOnboardingProposalRef(ref)
	if err := ValidateCapabilityOnboardingProposalRef(normalized); err != nil {
		return CapabilityOnboardingProposalRecord{}, err
	}
	return LoadCapabilityOnboardingProposalRecord(root, normalized.ProposalID)
}

func ResolveExecutionContextCapabilityOnboardingProposal(ec ExecutionContext) (*CapabilityOnboardingProposalRecord, error) {
	if ec.Step == nil {
		return nil, fmt.Errorf("execution context step is required")
	}
	if ec.Step.CapabilityOnboardingProposalRef == nil {
		return nil, nil
	}
	if strings.TrimSpace(ec.MissionStoreRoot) == "" {
		return nil, fmt.Errorf("mission store root is required to resolve capability onboarding proposal refs")
	}

	record, err := ResolveCapabilityOnboardingProposalRef(ec.MissionStoreRoot, *ec.Step.CapabilityOnboardingProposalRef)
	if err != nil {
		return nil, err
	}
	return &record, nil
}

func CapabilityOnboardingProposalStateValidForPlan(state CapabilityOnboardingProposalState) bool {
	switch NormalizeCapabilityOnboardingProposalState(state) {
	case CapabilityOnboardingProposalStateProposed, CapabilityOnboardingProposalStateApproved:
		return true
	default:
		return false
	}
}

type ResolvedExecutionContextCapabilityOnboardingProposalPreflight struct {
	Proposal             *CapabilityOnboardingProposalRecord `json:"proposal,omitempty"`
	RequiredCapabilities []string                            `json:"required_capabilities,omitempty"`
	RequiredDataDomains  []string                            `json:"required_data_domains,omitempty"`
}

func ResolveExecutionContextCapabilityOnboardingProposalPreflight(ec ExecutionContext) (ResolvedExecutionContextCapabilityOnboardingProposalPreflight, error) {
	proposal, err := ResolveExecutionContextCapabilityOnboardingProposal(ec)
	if err != nil {
		return ResolvedExecutionContextCapabilityOnboardingProposalPreflight{}, err
	}
	if proposal == nil {
		return ResolvedExecutionContextCapabilityOnboardingProposalPreflight{}, nil
	}

	proposalCopy := *proposal
	return ResolvedExecutionContextCapabilityOnboardingProposalPreflight{
		Proposal:             &proposalCopy,
		RequiredCapabilities: append([]string(nil), NormalizeStepRequiredCapabilities(ec.Step.RequiredCapabilities)...),
		RequiredDataDomains:  append([]string(nil), NormalizeStepRequiredDataDomains(ec.Step.RequiredDataDomains)...),
	}, nil
}
