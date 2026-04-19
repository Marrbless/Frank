package missioncontrol

import "encoding/json"

type InspectSummary struct {
	JobID        string        `json:"job_id"`
	MaxAuthority AuthorityTier `json:"max_authority"`
	AllowedTools []string      `json:"allowed_tools"`
	Steps        []InspectStep `json:"steps"`
}

type InspectStep struct {
	StepID                                       string                                                                `json:"step_id"`
	StepType                                     StepType                                                              `json:"step_type"`
	DependsOn                                    []string                                                              `json:"depends_on"`
	RequiredAuthority                            AuthorityTier                                                         `json:"required_authority"`
	AllowedTools                                 []string                                                              `json:"allowed_tools"`
	SuccessCriteria                              []string                                                              `json:"success_criteria"`
	EffectiveAllowedTools                        []string                                                              `json:"effective_allowed_tools"`
	RequiresApproval                             bool                                                                  `json:"requires_approval"`
	CampaignPreflight                            *ResolvedExecutionContextCampaignPreflight                            `json:"campaign_preflight,omitempty"`
	TreasuryPreflight                            *ResolvedExecutionContextTreasuryPreflight                            `json:"treasury_preflight,omitempty"`
	FrankZohoMailboxBootstrapPreflight           *ResolvedExecutionContextFrankZohoMailboxBootstrapPreflight           `json:"frank_zoho_mailbox_bootstrap_preflight,omitempty"`
	FrankTelegramOwnerControlOnboardingPreflight *ResolvedExecutionContextFrankTelegramOwnerControlOnboardingPreflight `json:"frank_telegram_owner_control_onboarding_preflight,omitempty"`
	FrankSlackOwnerControlOnboardingPreflight    *ResolvedExecutionContextFrankSlackOwnerControlOnboardingPreflight    `json:"frank_slack_owner_control_onboarding_preflight,omitempty"`
	FrankDiscordOwnerControlOnboardingPreflight  *ResolvedExecutionContextFrankDiscordOwnerControlOnboardingPreflight  `json:"frank_discord_owner_control_onboarding_preflight,omitempty"`
	FrankWhatsAppOwnerControlOnboardingPreflight *ResolvedExecutionContextFrankWhatsAppOwnerControlOnboardingPreflight `json:"frank_whatsapp_owner_control_onboarding_preflight,omitempty"`
	FrankGitHubOnboardingPreflight               *ResolvedExecutionContextFrankGitHubOnboardingPreflight               `json:"frank_github_onboarding_preflight,omitempty"`
	FrankStripeOnboardingPreflight               *ResolvedExecutionContextFrankStripeOnboardingPreflight               `json:"frank_stripe_onboarding_preflight,omitempty"`
	FrankPayPalOnboardingPreflight               *ResolvedExecutionContextFrankPayPalOnboardingPreflight               `json:"frank_paypal_onboarding_preflight,omitempty"`
	FrankGoogleOnboardingPreflight               *ResolvedExecutionContextFrankGoogleOnboardingPreflight               `json:"frank_google_onboarding_preflight,omitempty"`
	FrankLinkedInOnboardingPreflight             *ResolvedExecutionContextFrankLinkedInOnboardingPreflight             `json:"frank_linkedin_onboarding_preflight,omitempty"`
	CapabilityOnboardingProposalPreflight        *ResolvedExecutionContextCapabilityOnboardingProposalPreflight        `json:"capability_onboarding_proposal_preflight,omitempty"`
}

func NewInspectSummary(job Job, stepID string) (InspectSummary, error) {
	return newInspectSummary(job, stepID, func(step Step, ec ExecutionContext) (InspectStep, error) {
		return newInspectStepSummary(step, ec), nil
	})
}

func NewInspectSummaryWithTreasuryPreflight(job Job, stepID string, storeRoot string) (InspectSummary, error) {
	return NewInspectSummaryWithCampaignAndTreasuryPreflight(job, stepID, storeRoot)
}

func NewInspectSummaryWithCampaignAndTreasuryPreflight(job Job, stepID string, storeRoot string) (InspectSummary, error) {
	job.MissionStoreRoot = storeRoot
	return newInspectSummary(job, stepID, func(step Step, ec ExecutionContext) (InspectStep, error) {
		ec.MissionStoreRoot = storeRoot
		summary := newInspectStepSummary(step, ec)
		campaignPreflight, err := ResolveExecutionContextCampaignPreflight(ec)
		if err != nil {
			return InspectStep{}, err
		}
		if campaignPreflight.Campaign != nil {
			summary.CampaignPreflight = &campaignPreflight
		}
		preflight, err := ResolveExecutionContextTreasuryPreflight(ec)
		if err != nil {
			return InspectStep{}, err
		}
		if preflight.Treasury != nil {
			summary.TreasuryPreflight = &preflight
		}
		bootstrapPreflight, err := ResolveExecutionContextFrankZohoMailboxBootstrapPreflight(ec)
		if err != nil {
			return InspectStep{}, err
		}
		if bootstrapPreflight.Identity != nil && bootstrapPreflight.Account != nil {
			summary.FrankZohoMailboxBootstrapPreflight = &bootstrapPreflight
		}
		telegramOwnerControlPreflight, err := ResolveExecutionContextFrankTelegramOwnerControlOnboardingPreflight(ec)
		if err != nil {
			return InspectStep{}, err
		}
		if telegramOwnerControlPreflight.Identity != nil && telegramOwnerControlPreflight.Account != nil {
			summary.FrankTelegramOwnerControlOnboardingPreflight = &telegramOwnerControlPreflight
		}
		slackOwnerControlPreflight, err := ResolveExecutionContextFrankSlackOwnerControlOnboardingPreflight(ec)
		if err != nil {
			return InspectStep{}, err
		}
		if slackOwnerControlPreflight.Identity != nil && slackOwnerControlPreflight.Account != nil {
			summary.FrankSlackOwnerControlOnboardingPreflight = &slackOwnerControlPreflight
		}
		discordOwnerControlPreflight, err := ResolveExecutionContextFrankDiscordOwnerControlOnboardingPreflight(ec)
		if err != nil {
			return InspectStep{}, err
		}
		if discordOwnerControlPreflight.Identity != nil && discordOwnerControlPreflight.Account != nil {
			summary.FrankDiscordOwnerControlOnboardingPreflight = &discordOwnerControlPreflight
		}
		whatsAppOwnerControlPreflight, err := ResolveExecutionContextFrankWhatsAppOwnerControlOnboardingPreflight(ec)
		if err != nil {
			return InspectStep{}, err
		}
		if whatsAppOwnerControlPreflight.Identity != nil && whatsAppOwnerControlPreflight.Account != nil {
			summary.FrankWhatsAppOwnerControlOnboardingPreflight = &whatsAppOwnerControlPreflight
		}
		gitHubPreflight, err := ResolveExecutionContextFrankGitHubOnboardingPreflight(ec)
		if err != nil {
			return InspectStep{}, err
		}
		if gitHubPreflight.Identity != nil && gitHubPreflight.Account != nil {
			summary.FrankGitHubOnboardingPreflight = &gitHubPreflight
		}
		stripePreflight, err := ResolveExecutionContextFrankStripeOnboardingPreflight(ec)
		if err != nil {
			return InspectStep{}, err
		}
		if stripePreflight.Identity != nil && stripePreflight.Account != nil {
			summary.FrankStripeOnboardingPreflight = &stripePreflight
		}
		payPalPreflight, err := ResolveExecutionContextFrankPayPalOnboardingPreflight(ec)
		if err != nil {
			return InspectStep{}, err
		}
		if payPalPreflight.Identity != nil && payPalPreflight.Account != nil {
			summary.FrankPayPalOnboardingPreflight = &payPalPreflight
		}
		googlePreflight, err := ResolveExecutionContextFrankGoogleOnboardingPreflight(ec)
		if err != nil {
			return InspectStep{}, err
		}
		if googlePreflight.Identity != nil && googlePreflight.Account != nil {
			summary.FrankGoogleOnboardingPreflight = &googlePreflight
		}
		linkedInPreflight, err := ResolveExecutionContextFrankLinkedInOnboardingPreflight(ec)
		if err != nil {
			return InspectStep{}, err
		}
		if linkedInPreflight.Identity != nil && linkedInPreflight.Account != nil {
			summary.FrankLinkedInOnboardingPreflight = &linkedInPreflight
		}
		capabilityPreflight, err := ResolveExecutionContextCapabilityOnboardingProposalPreflight(ec)
		if err != nil {
			return InspectStep{}, err
		}
		if capabilityPreflight.Proposal != nil {
			summary.CapabilityOnboardingProposalPreflight = &capabilityPreflight
		}
		return summary, nil
	})
}

func newInspectSummary(job Job, stepID string, buildStep func(Step, ExecutionContext) (InspectStep, error)) (InspectSummary, error) {
	summary := InspectSummary{
		JobID:        job.ID,
		MaxAuthority: job.MaxAuthority,
		AllowedTools: append([]string(nil), job.AllowedTools...),
	}

	if stepID != "" {
		ec, err := ResolveExecutionContext(job, stepID)
		if err != nil {
			return InspectSummary{}, err
		}

		stepSummary, err := buildStep(*ec.Step, ec)
		if err != nil {
			return InspectSummary{}, err
		}
		summary.Steps = append(summary.Steps, stepSummary)
		return summary, nil
	}

	summary.Steps = make([]InspectStep, 0, len(job.Plan.Steps))
	for _, step := range job.Plan.Steps {
		ec, err := ResolveExecutionContext(job, step.ID)
		if err != nil {
			return InspectSummary{}, err
		}

		stepSummary, err := buildStep(step, ec)
		if err != nil {
			return InspectSummary{}, err
		}
		summary.Steps = append(summary.Steps, stepSummary)
	}

	return summary, nil
}

func NewInspectSummaryFromControl(control RuntimeControlContext, stepID string) (InspectSummary, error) {
	if stepID != "" && control.Step.ID != stepID {
		return InspectSummary{}, ValidationError{
			Code:    RejectionCodeUnknownStep,
			StepID:  stepID,
			Message: `step "` + stepID + `" not found in plan`,
		}
	}

	job := Job{
		ID:           control.JobID,
		MaxAuthority: control.MaxAuthority,
		AllowedTools: append([]string(nil), control.AllowedTools...),
	}
	step := copyStep(control.Step)

	return InspectSummary{
		JobID:        job.ID,
		MaxAuthority: job.MaxAuthority,
		AllowedTools: append([]string(nil), job.AllowedTools...),
		Steps:        []InspectStep{newInspectStepSummary(step, ExecutionContext{Job: &job, Step: &step})},
	}, nil
}

func NewInspectSummaryFromInspectablePlan(jobID string, plan *InspectablePlanContext, stepID string) (InspectSummary, error) {
	if plan == nil {
		return InspectSummary{}, ValidationError{
			Code:    RejectionCodeInvalidRuntimeState,
			Message: "inspect command requires validated mission plan",
		}
	}

	job := Job{
		ID:           jobID,
		MaxAuthority: plan.MaxAuthority,
		AllowedTools: append([]string(nil), plan.AllowedTools...),
		Plan: Plan{
			Steps: make([]Step, len(plan.Steps)),
		},
	}
	for i, step := range plan.Steps {
		job.Plan.Steps[i] = copyStep(step)
	}

	return NewInspectSummary(job, stepID)
}

func FormatInspectSummary(summary InspectSummary) (string, error) {
	data, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return "", err
	}
	return string(append(data, '\n')), nil
}

func newInspectStepSummary(step Step, ec ExecutionContext) InspectStep {
	return InspectStep{
		StepID:                step.ID,
		StepType:              step.Type,
		DependsOn:             append([]string(nil), step.DependsOn...),
		RequiredAuthority:     step.RequiredAuthority,
		AllowedTools:          append([]string(nil), step.AllowedTools...),
		SuccessCriteria:       append([]string(nil), step.SuccessCriteria...),
		EffectiveAllowedTools: EffectiveAllowedTools(ec.Job, ec.Step),
		RequiresApproval:      step.RequiresApproval,
	}
}
