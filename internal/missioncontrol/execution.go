package missioncontrol

import "fmt"

const RejectionCodeUnknownStep RejectionCode = "unknown_step"

func CloneExecutionContext(ec ExecutionContext) ExecutionContext {
	var cloned ExecutionContext
	if ec.Job != nil {
		cloned.Job = CloneJob(ec.Job)
	}
	if ec.Step != nil {
		stepCopy := copyStep(*ec.Step)
		cloned.Step = &stepCopy
	}
	cloned.Runtime = CloneJobRuntimeState(ec.Runtime)
	cloned.MissionStoreRoot = ec.MissionStoreRoot
	cloned.GovernedExternalTargets = cloneAutonomyEligibilityTargetRefs(ec.GovernedExternalTargets)
	return cloned
}

func CloneJob(job *Job) *Job {
	if job == nil {
		return nil
	}
	jobCopy := copyJob(*job)
	return &jobCopy
}

func ResolveExecutionContext(job Job, stepID string) (ExecutionContext, error) {
	if validationErrors := ValidatePlan(job); len(validationErrors) > 0 {
		return ExecutionContext{}, validationErrors[0]
	}

	stepIndex := -1
	for i, step := range job.Plan.Steps {
		if step.ID == stepID {
			stepIndex = i
			break
		}
	}

	if stepIndex == -1 {
		return ExecutionContext{}, ValidationError{
			Code:    RejectionCodeUnknownStep,
			StepID:  stepID,
			Message: fmt.Sprintf(`step %q not found in plan`, stepID),
		}
	}

	jobCopy := copyJob(job)
	stepCopy := normalizedStep(jobCopy.Plan.Steps[stepIndex])
	jobCopy.Plan.Steps[stepIndex] = copyStep(stepCopy)
	return ExecutionContext{
		Job:                     &jobCopy,
		Step:                    &stepCopy,
		GovernedExternalTargets: cloneAutonomyEligibilityTargetRefs(stepCopy.GovernedExternalTargets),
	}, nil
}

func copyJob(job Job) Job {
	jobCopy := job
	jobCopy.AllowedTools = append([]string(nil), job.AllowedTools...)
	jobCopy.SelectedSkills = append([]string(nil), job.SelectedSkills...)
	jobCopy.ModelPolicy = cloneModelPolicy(job.ModelPolicy)
	jobCopy.TargetSurfaces = cloneJobSurfaceRefs(job.TargetSurfaces)
	jobCopy.MutableSurfaces = cloneJobSurfaceRefs(job.MutableSurfaces)
	jobCopy.ImmutableSurfaces = cloneJobSurfaceRefs(job.ImmutableSurfaces)
	jobCopy.Plan = Plan{ID: job.Plan.ID}
	if job.Plan.Steps != nil {
		jobCopy.Plan.Steps = make([]Step, len(job.Plan.Steps))
		for i, step := range job.Plan.Steps {
			jobCopy.Plan.Steps[i] = copyStep(step)
		}
	}
	return jobCopy
}

func copyStep(step Step) Step {
	stepCopy := step
	stepCopy.DependsOn = append([]string(nil), step.DependsOn...)
	stepCopy.AllowedTools = append([]string(nil), step.AllowedTools...)
	stepCopy.SelectedSkills = append([]string(nil), step.SelectedSkills...)
	stepCopy.ModelPolicy = cloneModelPolicy(step.ModelPolicy)
	stepCopy.SuccessCriteria = append([]string(nil), step.SuccessCriteria...)
	stepCopy.LongRunningStartupCommand = append([]string(nil), step.LongRunningStartupCommand...)
	stepCopy.RequiredCapabilities = append([]string(nil), step.RequiredCapabilities...)
	stepCopy.RequiredDataDomains = append([]string(nil), step.RequiredDataDomains...)
	stepCopy.GovernedExternalTargets = cloneAutonomyEligibilityTargetRefs(step.GovernedExternalTargets)
	stepCopy.FrankObjectRefs = cloneFrankRegistryObjectRefs(step.FrankObjectRefs)
	stepCopy.CampaignRef = cloneCampaignRef(step.CampaignRef)
	stepCopy.TreasuryRef = cloneTreasuryRef(step.TreasuryRef)
	stepCopy.CapabilityOnboardingProposalRef = cloneCapabilityOnboardingProposalRef(step.CapabilityOnboardingProposalRef)
	stepCopy.SystemAction = cloneSystemActionSpec(step.SystemAction)
	return stepCopy
}

func normalizedStep(step Step) Step {
	stepCopy := copyStep(step)
	stepCopy.IdentityMode = NormalizeIdentityMode(stepCopy.IdentityMode)
	stepCopy.RequiredCapabilities = NormalizeStepRequiredCapabilities(stepCopy.RequiredCapabilities)
	stepCopy.RequiredDataDomains = NormalizeStepRequiredDataDomains(stepCopy.RequiredDataDomains)
	stepCopy.FrankObjectRefs = normalizeFrankRegistryObjectRefs(stepCopy.FrankObjectRefs)
	stepCopy.CampaignRef = normalizeCampaignRefPtr(stepCopy.CampaignRef)
	stepCopy.TreasuryRef = normalizeTreasuryRefPtr(stepCopy.TreasuryRef)
	stepCopy.CapabilityOnboardingProposalRef = normalizeCapabilityOnboardingProposalRefPtr(stepCopy.CapabilityOnboardingProposalRef)
	return stepCopy
}

func cloneAutonomyEligibilityTargetRefs(targets []AutonomyEligibilityTargetRef) []AutonomyEligibilityTargetRef {
	if len(targets) == 0 {
		return nil
	}

	cloned := make([]AutonomyEligibilityTargetRef, len(targets))
	copy(cloned, targets)
	return cloned
}

func cloneFrankRegistryObjectRefs(refs []FrankRegistryObjectRef) []FrankRegistryObjectRef {
	if len(refs) == 0 {
		return nil
	}

	cloned := make([]FrankRegistryObjectRef, len(refs))
	copy(cloned, refs)
	return cloned
}

func cloneJobSurfaceRefs(refs []JobSurfaceRef) []JobSurfaceRef {
	if len(refs) == 0 {
		return nil
	}

	cloned := make([]JobSurfaceRef, len(refs))
	copy(cloned, refs)
	return cloned
}

func normalizeFrankRegistryObjectRefs(refs []FrankRegistryObjectRef) []FrankRegistryObjectRef {
	if len(refs) == 0 {
		return nil
	}

	normalized := make([]FrankRegistryObjectRef, len(refs))
	for i, ref := range refs {
		normalized[i] = NormalizeFrankRegistryObjectRef(ref)
	}
	return normalized
}

func cloneCampaignRef(ref *CampaignRef) *CampaignRef {
	if ref == nil {
		return nil
	}

	cloned := *ref
	return &cloned
}

func normalizeCampaignRefPtr(ref *CampaignRef) *CampaignRef {
	if ref == nil {
		return nil
	}

	normalized := NormalizeCampaignRef(*ref)
	return &normalized
}

func cloneTreasuryRef(ref *TreasuryRef) *TreasuryRef {
	if ref == nil {
		return nil
	}

	cloned := *ref
	return &cloned
}

func normalizeTreasuryRefPtr(ref *TreasuryRef) *TreasuryRef {
	if ref == nil {
		return nil
	}

	normalized := NormalizeTreasuryRef(*ref)
	return &normalized
}

func cloneCapabilityOnboardingProposalRef(ref *CapabilityOnboardingProposalRef) *CapabilityOnboardingProposalRef {
	if ref == nil {
		return nil
	}

	cloned := *ref
	return &cloned
}

func normalizeCapabilityOnboardingProposalRefPtr(ref *CapabilityOnboardingProposalRef) *CapabilityOnboardingProposalRef {
	if ref == nil {
		return nil
	}

	normalized := NormalizeCapabilityOnboardingProposalRef(*ref)
	return &normalized
}

func cloneModelPolicy(policy *ModelPolicy) *ModelPolicy {
	if policy == nil {
		return nil
	}
	cloned := *policy
	cloned.AllowedModels = append([]string(nil), policy.AllowedModels...)
	cloned.RequiredCapabilities.SupportsTools = cloneBoolPtr(policy.RequiredCapabilities.SupportsTools)
	cloned.RequiredCapabilities.Local = cloneBoolPtr(policy.RequiredCapabilities.Local)
	cloned.RequiredCapabilities.Offline = cloneBoolPtr(policy.RequiredCapabilities.Offline)
	cloned.RequiredCapabilities.SupportsResponsesAPI = cloneBoolPtr(policy.RequiredCapabilities.SupportsResponsesAPI)
	cloned.AllowFallback = cloneBoolPtr(policy.AllowFallback)
	cloned.AllowCloud = cloneBoolPtr(policy.AllowCloud)
	return &cloned
}

func cloneBoolPtr(value *bool) *bool {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}
