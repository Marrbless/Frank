package tools

import (
	"encoding/json"
	"strings"

	"github.com/local/picobot/internal/missioncontrol"
)

// Readout-only operator surfaces live here to keep status/inspect adapters
// separate from TaskState's mutation and persistence paths.

func (s *TaskState) OperatorStatus(jobID string) (string, error) {
	if s == nil {
		return "", missioncontrol.ValidationError{
			Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
			Message: "operator command requires an active mission step",
		}
	}

	s.mu.Lock()
	ec := missioncontrol.CloneExecutionContext(s.executionContext)
	hasExecutionContext := s.hasExecutionContext
	control := missioncontrol.CloneRuntimeControlContext(&s.runtimeControl)
	hasRuntimeControl := s.hasRuntimeControl
	runtimeState := missioncontrol.CloneJobRuntimeState(&s.runtimeState)
	hasRuntimeState := s.hasRuntimeState
	missionStoreRoot := strings.TrimSpace(s.missionStoreRoot)
	s.mu.Unlock()

	if hasExecutionContext && ec.Runtime != nil {
		if ec.Job != nil && ec.Job.ID != jobID {
			return "", missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeStepValidationFailed,
				Message: "operator command does not match the active job",
			}
		}
		if ec.Runtime.JobID != "" && ec.Runtime.JobID != jobID {
			return "", missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeStepValidationFailed,
				Message: "operator command does not match the active job",
			}
		}
		campaignPreflight, treasuryPreflight, bootstrapPreflight, err := resolveExecutionContextCampaignAndTreasuryAndFrankZohoMailboxBootstrapPreflight(ec)
		if err != nil {
			return "", err
		}
		summary, err := missioncontrol.FormatOperatorStatusSummaryWithAllowedToolsAndCampaignAndTreasuryAndFrankZohoMailboxBootstrapPreflight(
			*ec.Runtime,
			missioncontrol.EffectiveAllowedTools(ec.Job, ec.Step),
			campaignPreflight,
			treasuryPreflight,
			bootstrapPreflight,
		)
		if err != nil {
			return "", err
		}
		return formatOperatorStatusReadoutWithDeferredSchedulerTriggers(summary, ec.MissionStoreRoot)
	}

	if !hasRuntimeState || runtimeState == nil {
		return "", missioncontrol.ValidationError{
			Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
			Message: "operator command requires an active mission step",
		}
	}
	if runtimeState.JobID != jobID {
		return "", missioncontrol.ValidationError{
			Code:    missioncontrol.RejectionCodeStepValidationFailed,
			Message: "operator command does not match the active job",
		}
	}

	var allowedTools []string
	if hasRuntimeControl && control != nil {
		allowedTools = missioncontrol.EffectiveAllowedTools(&missioncontrol.Job{AllowedTools: append([]string(nil), control.AllowedTools...)}, &control.Step)
	}
	summary := missioncontrol.BuildOperatorStatusSummaryWithAllowedTools(*runtimeState, allowedTools)
	if gate := persistedTaskStateCampaignZohoEmailSendGate(missionStoreRoot, runtimeState, control); gate != nil {
		summary.CampaignZohoEmailSendGate = gate
	}
	data, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return "", err
	}
	return formatOperatorStatusReadoutWithDeferredSchedulerTriggers(string(append(data, '\n')), missionStoreRoot)
}

func persistedTaskStateCampaignZohoEmailSendGate(missionStoreRoot string, runtime *missioncontrol.JobRuntimeState, control *missioncontrol.RuntimeControlContext) *missioncontrol.CampaignZohoEmailSendGateDecision {
	missionStoreRoot = strings.TrimSpace(missionStoreRoot)
	if missionStoreRoot == "" || runtime == nil || control == nil {
		return nil
	}
	if strings.TrimSpace(runtime.ActiveStepID) == "" || strings.TrimSpace(control.Step.ID) != strings.TrimSpace(runtime.ActiveStepID) {
		return nil
	}
	if control.Step.CampaignRef == nil {
		return nil
	}

	ec := missioncontrol.ExecutionContext{
		Step:             &control.Step,
		MissionStoreRoot: missionStoreRoot,
	}
	preflight, err := missioncontrol.ResolveExecutionContextCampaignPreflight(ec)
	if err != nil {
		return &missioncontrol.CampaignZohoEmailSendGateDecision{
			CampaignID: strings.TrimSpace(control.Step.CampaignRef.CampaignID),
			Allowed:    false,
			Halted:     false,
			Reason:     err.Error(),
		}
	}
	if preflight.Campaign == nil {
		return nil
	}

	decision, err := missioncontrol.LoadCommittedCampaignZohoEmailSendGateDecision(missionStoreRoot, *preflight.Campaign)
	if err != nil {
		return &missioncontrol.CampaignZohoEmailSendGateDecision{
			CampaignID: strings.TrimSpace(preflight.Campaign.CampaignID),
			Allowed:    false,
			Halted:     false,
			Reason:     err.Error(),
		}
	}
	return &decision
}

func formatOperatorStatusReadoutWithDeferredSchedulerTriggers(summary string, missionStoreRoot string) (string, error) {
	missionStoreRoot = strings.TrimSpace(missionStoreRoot)
	if missionStoreRoot == "" {
		return summary, nil
	}

	var statusSummary missioncontrol.OperatorStatusSummary
	if err := json.Unmarshal([]byte(summary), &statusSummary); err != nil {
		return "", err
	}
	statusSummary = missioncontrol.WithRuntimePackIdentity(statusSummary, missionStoreRoot)

	deferred, err := missioncontrol.LoadDeferredSchedulerTriggerStatuses(missionStoreRoot)
	if err == nil && len(deferred) > 0 {
		statusSummary = missioncontrol.WithDeferredSchedulerTriggers(statusSummary, deferred)
	}

	data, err := json.MarshalIndent(statusSummary, "", "  ")
	if err != nil {
		return "", err
	}
	return string(append(data, '\n')), nil
}

func resolveExecutionContextCampaignAndTreasuryAndFrankZohoMailboxBootstrapPreflight(ec missioncontrol.ExecutionContext) (*missioncontrol.ResolvedExecutionContextCampaignPreflight, *missioncontrol.ResolvedExecutionContextTreasuryPreflight, *missioncontrol.ResolvedExecutionContextFrankZohoMailboxBootstrapPreflight, error) {
	var campaignPreflight *missioncontrol.ResolvedExecutionContextCampaignPreflight
	if ec.Step != nil && ec.Step.CampaignRef != nil {
		resolved, err := missioncontrol.ResolveExecutionContextCampaignPreflight(ec)
		if err != nil {
			return nil, nil, nil, err
		}
		campaignPreflight = &resolved
	}

	var treasuryPreflight *missioncontrol.ResolvedExecutionContextTreasuryPreflight
	if ec.Step != nil && ec.Step.TreasuryRef != nil {
		resolved, err := missioncontrol.ResolveExecutionContextTreasuryPreflight(ec)
		if err != nil {
			return nil, nil, nil, err
		}
		treasuryPreflight = &resolved
	}

	var bootstrapPreflight *missioncontrol.ResolvedExecutionContextFrankZohoMailboxBootstrapPreflight
	if ec.Step != nil && missioncontrol.DeclaresFrankZohoMailboxBootstrap(*ec.Step) {
		resolved, err := missioncontrol.ResolveExecutionContextFrankZohoMailboxBootstrapPreflight(ec)
		if err != nil {
			return nil, nil, nil, err
		}
		if resolved.Identity != nil && resolved.Account != nil {
			bootstrapPreflight = &resolved
		}
	}

	return campaignPreflight, treasuryPreflight, bootstrapPreflight, nil
}

func (s *TaskState) OperatorInspect(jobID string, stepID string) (string, error) {
	if s == nil {
		return "", missioncontrol.ValidationError{
			Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
			Message: "operator command requires an active mission step",
		}
	}

	s.mu.Lock()
	ec := missioncontrol.CloneExecutionContext(s.executionContext)
	hasExecutionContext := s.hasExecutionContext
	missionJob := missioncontrol.CloneJob(&s.missionJob)
	hasMissionJob := s.hasMissionJob
	control := missioncontrol.CloneRuntimeControlContext(&s.runtimeControl)
	hasRuntimeControl := s.hasRuntimeControl
	runtimeState := missioncontrol.CloneJobRuntimeState(&s.runtimeState)
	hasRuntimeState := s.hasRuntimeState
	s.mu.Unlock()

	if hasExecutionContext && ec.Runtime != nil {
		if ec.Job == nil {
			return "", missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
				Message: "operator command requires an active mission step",
			}
		}
		if ec.Job.ID != jobID {
			return "", missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeStepValidationFailed,
				Message: "operator command does not match the active job",
			}
		}
		if ec.Runtime.JobID != "" && ec.Runtime.JobID != jobID {
			return "", missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeStepValidationFailed,
				Message: "operator command does not match the active job",
			}
		}

		summary, err := missioncontrol.NewInspectSummaryWithCampaignAndTreasuryPreflight(*ec.Job, stepID, ec.MissionStoreRoot)
		if err != nil {
			return "", err
		}
		return missioncontrol.FormatInspectSummary(summary)
	}

	if !hasRuntimeState || runtimeState == nil {
		return "", missioncontrol.ValidationError{
			Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
			Message: "operator command requires an active mission step",
		}
	}
	if runtimeState.JobID != jobID {
		return "", missioncontrol.ValidationError{
			Code:    missioncontrol.RejectionCodeStepValidationFailed,
			Message: "operator command does not match the active job",
		}
	}
	if hasMissionJob && missionJob != nil {
		if missionJob.ID != "" && missionJob.ID != jobID {
			return "", missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeStepValidationFailed,
				Message: "operator command does not match the active job",
			}
		}
		summary, err := missioncontrol.NewInspectSummary(*missionJob, stepID)
		if err != nil {
			return "", err
		}
		return missioncontrol.FormatInspectSummary(summary)
	}
	if runtimeState.InspectablePlan != nil {
		summary, err := missioncontrol.NewInspectSummaryFromInspectablePlan(runtimeState.JobID, runtimeState.InspectablePlan, stepID)
		if err != nil {
			return "", err
		}
		return missioncontrol.FormatInspectSummary(summary)
	}
	if !hasRuntimeControl || control == nil {
		return "", missioncontrol.ValidationError{
			Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
			Message: "inspect command requires validated mission plan",
		}
	}
	if control.JobID != "" && control.JobID != jobID {
		return "", missioncontrol.ValidationError{
			Code:    missioncontrol.RejectionCodeStepValidationFailed,
			Message: "operator command does not match the active job",
		}
	}

	return "", missioncontrol.ValidationError{
		Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
		Message: "inspect command requires validated mission plan",
	}
}
