package missioncontrol

import "fmt"

// RequireExecutionContextCampaignReadiness is the single missioncontrol-owned
// campaign readiness surface for current V3 control-plane work. It consumes the
// read-only campaign preflight, preserves zero-ref behavior, and fails closed
// when a declared campaign is not execution-ready.
func RequireExecutionContextCampaignReadiness(ec ExecutionContext) error {
	preflight, err := ResolveExecutionContextCampaignPreflight(ec)
	if err != nil {
		return err
	}
	if preflight.Campaign == nil {
		return nil
	}

	campaign := preflight.Campaign
	if campaign.State != CampaignStateActive {
		return ValidationError{
			Code:    RejectionCodeInvalidRuntimeState,
			StepID:  campaignReadinessStepID(ec),
			Message: fmt.Sprintf("campaign readiness requires state %q; got %q", CampaignStateActive, campaign.State),
		}
	}
	if campaign.IdentityMode != IdentityModeAgentAlias {
		return ValidationError{
			Code:    RejectionCodeInvalidRuntimeState,
			StepID:  campaignReadinessStepID(ec),
			Message: fmt.Sprintf("campaign readiness requires identity_mode %q; got %q", IdentityModeAgentAlias, campaign.IdentityMode),
		}
	}

	for _, target := range campaign.GovernedExternalTargets {
		if _, err := RequireAutonomyEligibleTarget(ec.MissionStoreRoot, target); err != nil {
			return err
		}
	}
	for _, identity := range preflight.Identities {
		if _, err := RequireAutonomyEligibleTarget(ec.MissionStoreRoot, identity.EligibilityTargetRef); err != nil {
			return err
		}
	}
	for _, account := range preflight.Accounts {
		if _, err := RequireAutonomyEligibleTarget(ec.MissionStoreRoot, account.EligibilityTargetRef); err != nil {
			return err
		}
	}
	for _, container := range preflight.Containers {
		if _, err := RequireAutonomyEligibleTarget(ec.MissionStoreRoot, container.EligibilityTargetRef); err != nil {
			return err
		}
	}

	return nil
}

func campaignReadinessStepID(ec ExecutionContext) string {
	if ec.Step == nil {
		return ""
	}
	return ec.Step.ID
}
