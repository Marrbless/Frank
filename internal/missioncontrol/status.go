package missioncontrol

import (
	"encoding/json"
	"time"
)

type OperatorStatusSummary struct {
	JobID           string                         `json:"job_id"`
	State           JobState                       `json:"state"`
	ActiveStepID    string                         `json:"active_step_id,omitempty"`
	WaitingReason   string                         `json:"waiting_reason,omitempty"`
	PausedReason    string                         `json:"paused_reason,omitempty"`
	AbortedReason   string                         `json:"aborted_reason,omitempty"`
	ApprovalRequest *OperatorApprovalRequestStatus `json:"approval_request,omitempty"`
}

type OperatorApprovalRequestStatus struct {
	State            ApprovalState `json:"state"`
	StepID           string        `json:"step_id"`
	RequestedAction  string        `json:"requested_action"`
	Scope            string        `json:"scope"`
	ProposedAction   string        `json:"proposed_action,omitempty"`
	WhyNeeded        string        `json:"why_needed,omitempty"`
	AuthorityTier    AuthorityTier `json:"authority_tier,omitempty"`
	FallbackIfDenied string        `json:"fallback_if_denied,omitempty"`
	ExpiresAt        *string       `json:"expires_at,omitempty"`
	SupersededAt     *string       `json:"superseded_at,omitempty"`
}

func BuildOperatorStatusSummary(runtime JobRuntimeState) OperatorStatusSummary {
	summary := OperatorStatusSummary{
		JobID:         runtime.JobID,
		State:         runtime.State,
		ActiveStepID:  runtime.ActiveStepID,
		WaitingReason: runtime.WaitingReason,
		PausedReason:  runtime.PausedReason,
		AbortedReason: runtime.AbortedReason,
	}

	if request, ok := selectOperatorStatusApprovalRequest(runtime); ok {
		status := OperatorApprovalRequestStatus{
			State:           request.State,
			StepID:          request.StepID,
			RequestedAction: request.RequestedAction,
			Scope:           request.Scope,
		}
		if request.Content != nil {
			status.ProposedAction = request.Content.ProposedAction
			status.WhyNeeded = request.Content.WhyNeeded
			status.AuthorityTier = request.Content.AuthorityTier
			status.FallbackIfDenied = request.Content.FallbackIfDenied
		}
		if !request.ExpiresAt.IsZero() {
			expiresAt := request.ExpiresAt.UTC().Format(time.RFC3339Nano)
			status.ExpiresAt = &expiresAt
		}
		if !request.SupersededAt.IsZero() {
			supersededAt := request.SupersededAt.UTC().Format(time.RFC3339Nano)
			status.SupersededAt = &supersededAt
		}
		summary.ApprovalRequest = &status
	}

	return summary
}

func FormatOperatorStatusSummary(runtime JobRuntimeState) (string, error) {
	data, err := json.MarshalIndent(BuildOperatorStatusSummary(runtime), "", "  ")
	if err != nil {
		return "", err
	}
	return string(append(data, '\n')), nil
}

func selectOperatorStatusApprovalRequest(runtime JobRuntimeState) (ApprovalRequest, bool) {
	var fallback *ApprovalRequest
	for i := len(runtime.ApprovalRequests) - 1; i >= 0; i-- {
		request := runtime.ApprovalRequests[i]
		if runtime.JobID != "" && request.JobID != runtime.JobID {
			continue
		}
		if runtime.ActiveStepID != "" && request.StepID == runtime.ActiveStepID {
			return request, true
		}
		if fallback == nil {
			candidate := request
			fallback = &candidate
		}
	}
	if fallback == nil {
		return ApprovalRequest{}, false
	}
	return *fallback, true
}
