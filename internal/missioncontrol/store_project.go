package missioncontrol

import (
	"fmt"
	"strings"
	"time"
)

type MissionStatusSnapshot struct {
	MissionRequired   bool             `json:"mission_required"`
	Active            bool             `json:"active"`
	MissionFile       string           `json:"mission_file"`
	JobID             string           `json:"job_id"`
	StepID            string           `json:"step_id"`
	StepType          string           `json:"step_type"`
	RequiredAuthority AuthorityTier    `json:"required_authority"`
	RequiresApproval  bool             `json:"requires_approval"`
	AllowedTools      []string         `json:"allowed_tools"`
	Runtime           *JobRuntimeState `json:"runtime,omitempty"`
	// RuntimeSummary is a convenience projection of direct OperatorStatus JSON.
	// Provider-specific Zoho fields may be carried through incidentally here, but
	// only direct OperatorStatus and direct OperatorInspect remain frozen
	// operator-read contract surfaces for those fields on this branch line.
	RuntimeSummary *OperatorStatusSummary `json:"runtime_summary,omitempty"`
	RuntimeControl *RuntimeControlContext `json:"runtime_control,omitempty"`
	UpdatedAt      string                 `json:"updated_at"`
}

type MissionStatusSnapshotOptions struct {
	MissionRequired bool
	MissionFile     string
	UpdatedAt       time.Time
}

func BuildCommittedMissionStatusSnapshot(root, jobID string, opts MissionStatusSnapshotOptions) (MissionStatusSnapshot, error) {
	if strings.TrimSpace(root) == "" {
		return MissionStatusSnapshot{}, fmt.Errorf("committed mission status snapshot requires a durable store root")
	}
	if strings.TrimSpace(jobID) == "" {
		return MissionStatusSnapshot{}, fmt.Errorf("committed mission status snapshot requires a job_id")
	}

	now := opts.UpdatedAt
	if now.IsZero() {
		now = time.Now().UTC()
	} else {
		now = now.UTC()
	}

	runtimeState, err := HydrateCommittedJobRuntimeState(root, jobID, now)
	if err != nil {
		return MissionStatusSnapshot{}, err
	}
	runtimeControl, err := HydrateCommittedRuntimeControlContext(root, jobID, now)
	if err != nil {
		return MissionStatusSnapshot{}, err
	}

	snapshot := MissionStatusSnapshot{
		MissionRequired: opts.MissionRequired,
		MissionFile:     opts.MissionFile,
		JobID:           runtimeState.JobID,
		StepID:          runtimeState.ActiveStepID,
		StepType:        "",
		AllowedTools:    []string{},
		Runtime:         CloneJobRuntimeState(&runtimeState),
		RuntimeControl:  CloneRuntimeControlContext(runtimeControl),
		UpdatedAt:       now.Format(time.RFC3339Nano),
	}
	snapshot.Active = runtimeState.State == JobStateRunning && runtimeState.ActiveStepID != ""

	if stepSummary, ok, err := buildCommittedMissionStatusStepSummary(runtimeState, runtimeControl); err != nil {
		return MissionStatusSnapshot{}, err
	} else if ok {
		snapshot.StepType = string(stepSummary.StepType)
		snapshot.RequiredAuthority = stepSummary.RequiredAuthority
		snapshot.RequiresApproval = stepSummary.RequiresApproval
		snapshot.AllowedTools = append([]string(nil), stepSummary.EffectiveAllowedTools...)
	}

	summary := BuildOperatorStatusSummaryWithAllowedTools(runtimeState, snapshot.AllowedTools)
	campaignGate, err := buildCommittedMissionStatusCampaignZohoEmailSendGate(root, runtimeState, runtimeControl)
	if err != nil {
		return MissionStatusSnapshot{}, err
	}
	summary.CampaignZohoEmailSendGate = campaignGate
	deferredSchedulerTriggers, err := LoadDeferredSchedulerTriggerStatuses(root)
	if err != nil {
		return MissionStatusSnapshot{}, err
	}
	summary = WithDeferredSchedulerTriggers(summary, deferredSchedulerTriggers)
	summary = WithRuntimePackIdentity(summary, root)
	summary = WithImprovementCandidateIdentity(summary, root)
	summary = WithEvalSuiteIdentity(summary, root)
	summary = WithPromotionPolicyIdentity(summary, root)
	summary = WithImprovementRunIdentity(summary, root)
	summary = WithCandidateResultIdentity(summary, root)
	summary = WithCandidatePromotionDecisionIdentity(summary, root)
	summary = WithHotUpdateGateIdentity(summary, root)
	summary = WithHotUpdateOutcomeIdentity(summary, root)
	summary = WithPromotionIdentity(summary, root)
	summary = WithRollbackIdentity(summary, root)
	summary = WithRollbackApplyIdentity(summary, root)
	snapshot.RuntimeSummary = &summary

	return snapshot, nil
}

func buildCommittedMissionStatusStepSummary(runtime JobRuntimeState, control *RuntimeControlContext) (InspectStep, bool, error) {
	if runtime.ActiveStepID == "" {
		return InspectStep{}, false, nil
	}
	if control != nil {
		summary, err := NewInspectSummaryFromControl(*control, runtime.ActiveStepID)
		if err != nil {
			return InspectStep{}, false, err
		}
		if len(summary.Steps) == 0 {
			return InspectStep{}, false, nil
		}
		return summary.Steps[0], true, nil
	}
	if runtime.InspectablePlan == nil {
		return InspectStep{}, false, fmt.Errorf("committed runtime active step %q is missing inspectable plan metadata", runtime.ActiveStepID)
	}
	summary, err := NewInspectSummaryFromInspectablePlan(runtime.JobID, runtime.InspectablePlan, runtime.ActiveStepID)
	if err != nil {
		return InspectStep{}, false, err
	}
	if len(summary.Steps) == 0 {
		return InspectStep{}, false, nil
	}
	return summary.Steps[0], true, nil
}

func buildCommittedMissionStatusCampaignZohoEmailSendGate(root string, runtime JobRuntimeState, control *RuntimeControlContext) (*CampaignZohoEmailSendGateDecision, error) {
	step, ok, err := committedMissionStatusActiveStep(runtime, control)
	if err != nil || !ok || step.CampaignRef == nil {
		return nil, err
	}

	ec := ExecutionContext{
		Step:             &step,
		MissionStoreRoot: root,
	}
	preflight, err := ResolveExecutionContextCampaignPreflight(ec)
	if err != nil {
		return nil, err
	}
	if preflight.Campaign == nil {
		return nil, nil
	}

	decision, err := LoadCommittedCampaignZohoEmailSendGateDecision(root, *preflight.Campaign)
	if err != nil {
		return &CampaignZohoEmailSendGateDecision{
			CampaignID: strings.TrimSpace(preflight.Campaign.CampaignID),
			Allowed:    false,
			Halted:     false,
			Reason:     err.Error(),
		}, nil
	}
	return &decision, nil
}

func committedMissionStatusActiveStep(runtime JobRuntimeState, control *RuntimeControlContext) (Step, bool, error) {
	if runtime.ActiveStepID == "" {
		return Step{}, false, nil
	}
	if control != nil && strings.TrimSpace(control.Step.ID) == runtime.ActiveStepID {
		return copyStep(control.Step), true, nil
	}
	if runtime.InspectablePlan == nil {
		return Step{}, false, nil
	}
	for _, step := range runtime.InspectablePlan.Steps {
		if strings.TrimSpace(step.ID) == runtime.ActiveStepID {
			return copyStep(step), true, nil
		}
	}
	return Step{}, false, nil
}
