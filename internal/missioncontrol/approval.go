package missioncontrol

import (
	"fmt"
	"strings"
	"time"
)

type ApprovalState string

const (
	ApprovalStatePending    ApprovalState = "pending"
	ApprovalStateGranted    ApprovalState = "granted"
	ApprovalStateDenied     ApprovalState = "denied"
	ApprovalStateExpired    ApprovalState = "expired"
	ApprovalStateSuperseded ApprovalState = "superseded"
	ApprovalStateRevoked    ApprovalState = "revoked"
)

type ApprovalDecision string

const (
	ApprovalDecisionApprove ApprovalDecision = "approve"
	ApprovalDecisionDeny    ApprovalDecision = "deny"
)

const (
	ApprovalRequestedActionStepComplete = "step_complete"
	ApprovalScopeMissionStep            = "mission_step"
	ApprovalRequestedViaRuntime         = "runtime_waiting_user"
	ApprovalGrantedViaOperatorCommand   = "operator_command"
	ApprovalGrantedViaOperatorReply     = "operator_reply"
	defaultApprovalRequestTTL           = 5 * time.Minute
)

type ApprovalRequest struct {
	JobID           string        `json:"job_id"`
	StepID          string        `json:"step_id"`
	RequestedAction string        `json:"requested_action"`
	Scope           string        `json:"scope"`
	RequestedVia    string        `json:"requested_via"`
	GrantedVia      string        `json:"granted_via,omitempty"`
	State           ApprovalState `json:"state"`
	Reason          string        `json:"reason,omitempty"`
	RequestedAt     time.Time     `json:"requested_at,omitempty"`
	ExpiresAt       time.Time     `json:"expires_at,omitempty"`
	ResolvedAt      time.Time     `json:"resolved_at,omitempty"`
	SupersededAt    time.Time     `json:"superseded_at,omitempty"`
}

type ApprovalGrant struct {
	JobID           string        `json:"job_id"`
	StepID          string        `json:"step_id"`
	RequestedAction string        `json:"requested_action"`
	Scope           string        `json:"scope"`
	GrantedVia      string        `json:"granted_via"`
	State           ApprovalState `json:"state"`
	GrantedAt       time.Time     `json:"granted_at,omitempty"`
	RevokedAt       time.Time     `json:"revoked_at,omitempty"`
}

func approvalBindingForStep(step Step) (string, string, bool) {
	if step.Type != StepTypeDiscussion || step.Subtype != StepSubtypeAuthorization {
		return "", "", false
	}
	return ApprovalRequestedActionStepComplete, ApprovalScopeMissionStep, true
}

func ParsePlainApprovalDecision(input string) (ApprovalDecision, bool) {
	normalized := strings.ToLower(strings.TrimSpace(input))
	normalized = strings.Trim(normalized, " \t\r\n.!?")
	switch normalized {
	case "yes":
		return ApprovalDecisionApprove, true
	case "no":
		return ApprovalDecisionDeny, true
	default:
		return "", false
	}
}

func ResolveSinglePendingApprovalRequest(runtime JobRuntimeState) (ApprovalRequest, bool, error) {
	pending := make([]ApprovalRequest, 0, 1)
	for _, request := range runtime.ApprovalRequests {
		if request.State != ApprovalStatePending {
			continue
		}
		pending = append(pending, request)
		if len(pending) > 1 {
			return ApprovalRequest{}, true, ValidationError{
				Code:    RejectionCodeStepValidationFailed,
				Message: "plain yes/no approval is ambiguous because multiple pending approval requests exist",
			}
		}
	}
	if len(pending) == 0 {
		return ApprovalRequest{}, false, nil
	}
	return pending[0], true, nil
}

func ApprovalRequestMatchesStepBinding(request ApprovalRequest, jobID string, step Step) bool {
	requestedAction, scope, ok := approvalBindingForStep(step)
	if !ok {
		return false
	}
	if request.JobID != jobID || request.StepID != step.ID {
		return false
	}
	return request.RequestedAction == requestedAction && request.Scope == scope
}

func hasPendingApprovalRequest(runtime *JobRuntimeState, jobID, stepID, requestedAction, scope string) bool {
	if runtime == nil {
		return false
	}

	_, ok := findPendingApprovalRequest(runtime.ApprovalRequests, jobID, stepID, requestedAction, scope)
	return ok
}

func findLatestApprovalRequest(requests []ApprovalRequest, jobID, stepID, requestedAction, scope string) (int, bool) {
	for i := len(requests) - 1; i >= 0; i-- {
		request := requests[i]
		if request.JobID != jobID || request.StepID != stepID {
			continue
		}
		if request.RequestedAction != requestedAction || request.Scope != scope {
			continue
		}
		return i, true
	}

	return 0, false
}

func RefreshApprovalRequests(current JobRuntimeState, now time.Time) (JobRuntimeState, bool) {
	next := *CloneJobRuntimeState(&current)
	changed := false
	for i := range next.ApprovalRequests {
		request := &next.ApprovalRequests[i]
		if request.State != ApprovalStatePending {
			continue
		}
		if request.ExpiresAt.IsZero() || request.ExpiresAt.After(now) {
			continue
		}
		request.State = ApprovalStateExpired
		if request.ResolvedAt.IsZero() {
			request.ResolvedAt = request.ExpiresAt
			if request.ResolvedAt.IsZero() {
				request.ResolvedAt = now
			}
		}
		request.GrantedVia = ""
		request.SupersededAt = time.Time{}
		changed = true
	}
	if changed {
		next.UpdatedAt = now
	}
	return next, changed
}

func NormalizeHydratedApprovalRequests(current JobRuntimeState, now time.Time) (JobRuntimeState, bool) {
	if current.State != JobStateWaitingUser {
		return *CloneJobRuntimeState(&current), false
	}
	return RefreshApprovalRequests(current, now)
}

func ExpireActiveApprovalRequest(current JobRuntimeState, now time.Time, jobID, stepID, requestedAction, scope string) (JobRuntimeState, bool) {
	next, _ := RefreshApprovalRequests(current, now)
	requestIndex, ok := findPendingApprovalRequest(next.ApprovalRequests, jobID, stepID, requestedAction, scope)
	if !ok {
		return next, false
	}

	next.ApprovalRequests[requestIndex].State = ApprovalStateExpired
	next.ApprovalRequests[requestIndex].GrantedVia = ""
	next.ApprovalRequests[requestIndex].ExpiresAt = now
	next.ApprovalRequests[requestIndex].ResolvedAt = now
	next.ApprovalRequests[requestIndex].SupersededAt = time.Time{}
	next.UpdatedAt = now
	return next, true
}

func appendPendingApprovalRequest(current JobRuntimeState, now time.Time, request ApprovalRequest) JobRuntimeState {
	next := *CloneJobRuntimeState(&current)
	for i := range next.ApprovalRequests {
		if !approvalRequestsShareBinding(next.ApprovalRequests[i], request) {
			continue
		}
		if next.ApprovalRequests[i].State != ApprovalStatePending {
			continue
		}
		next.ApprovalRequests[i].State = ApprovalStateSuperseded
		next.ApprovalRequests[i].GrantedVia = ""
		next.ApprovalRequests[i].ResolvedAt = now
		next.ApprovalRequests[i].SupersededAt = now
	}

	request.State = ApprovalStatePending
	request.GrantedVia = ""
	request.RequestedAt = now
	if request.ExpiresAt.IsZero() {
		request.ExpiresAt = now.Add(defaultApprovalRequestTTL)
	}
	request.SupersededAt = time.Time{}
	request.ResolvedAt = time.Time{}
	next.ApprovalRequests = append(next.ApprovalRequests, request)
	next.UpdatedAt = now
	return next
}

func ApplyApprovalDecision(ec ExecutionContext, now time.Time, decision ApprovalDecision, via string) (JobRuntimeState, error) {
	if ec.Job == nil || ec.Step == nil || ec.Runtime == nil {
		return JobRuntimeState{}, ValidationError{
			Code:    RejectionCodeInvalidRuntimeState,
			Message: "approval decision requires active job, step, and runtime state",
		}
	}
	if ec.Runtime.State != JobStateWaitingUser {
		return JobRuntimeState{}, ValidationError{
			Code:    RejectionCodeInvalidRuntimeState,
			StepID:  ec.Step.ID,
			Message: fmt.Sprintf("approval decision requires waiting_user runtime state, got %q", ec.Runtime.State),
		}
	}
	if strings.TrimSpace(via) == "" {
		return JobRuntimeState{}, ValidationError{
			Code:    RejectionCodeStepValidationFailed,
			StepID:  ec.Step.ID,
			Message: "approval decision requires granted_via",
		}
	}

	requestedAction, scope, ok := approvalBindingForStep(*ec.Step)
	if !ok {
		return JobRuntimeState{}, ValidationError{
			Code:    RejectionCodeStepValidationFailed,
			StepID:  ec.Step.ID,
			Message: "step does not define an approval binding",
		}
	}

	next := *CloneJobRuntimeState(ec.Runtime)
	next, _ = RefreshApprovalRequests(next, now)
	requestIndex, ok := findPendingApprovalRequest(next.ApprovalRequests, ec.Job.ID, ec.Step.ID, requestedAction, scope)
	if !ok {
		return JobRuntimeState{}, ValidationError{
			Code:    RejectionCodeStepValidationFailed,
			StepID:  ec.Step.ID,
			Message: "no pending approval request matches the active job and step",
		}
	}

	next.ApprovalRequests[requestIndex].ResolvedAt = now
	switch decision {
	case ApprovalDecisionApprove:
		next.ApprovalRequests[requestIndex].State = ApprovalStateGranted
		next.ApprovalRequests[requestIndex].GrantedVia = via
		next.ApprovalGrants = append(next.ApprovalGrants, ApprovalGrant{
			JobID:           ec.Job.ID,
			StepID:          ec.Step.ID,
			RequestedAction: requestedAction,
			Scope:           scope,
			GrantedVia:      via,
			State:           ApprovalStateGranted,
			GrantedAt:       now,
		})

		return pauseAfterValidatedCompletion(ExecutionContext{
			Job:     ec.Job,
			Step:    ec.Step,
			Runtime: &next,
		}, now)
	case ApprovalDecisionDeny:
		next.ApprovalRequests[requestIndex].State = ApprovalStateDenied
		next.UpdatedAt = now
		return next, nil
	default:
		return JobRuntimeState{}, ValidationError{
			Code:    RejectionCodeStepValidationFailed,
			StepID:  ec.Step.ID,
			Message: fmt.Sprintf("unsupported approval decision %q", decision),
		}
	}
}

func findPendingApprovalRequest(requests []ApprovalRequest, jobID, stepID, requestedAction, scope string) (int, bool) {
	for i := len(requests) - 1; i >= 0; i-- {
		request := requests[i]
		if request.JobID != jobID || request.StepID != stepID {
			continue
		}
		if request.RequestedAction != requestedAction || request.Scope != scope {
			continue
		}
		if request.State != ApprovalStatePending {
			continue
		}
		return i, true
	}

	return 0, false
}

func approvalRequestsShareBinding(left ApprovalRequest, right ApprovalRequest) bool {
	return left.JobID == right.JobID &&
		left.StepID == right.StepID &&
		left.RequestedAction == right.RequestedAction &&
		left.Scope == right.Scope
}
