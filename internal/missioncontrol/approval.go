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
	ApprovalScopeOneStep                = ApprovalScopeMissionStep
	ApprovalScopeOneJob                 = "one_job"
	ApprovalScopeOneSession             = "one_session"
	ApprovalRequestedViaRuntime         = "runtime_waiting_user"
	ApprovalGrantedViaOperatorCommand   = "operator_command"
	ApprovalGrantedViaOperatorReply     = "operator_reply"
	defaultApprovalRequestTTL           = 5 * time.Minute
	maxPendingApprovalRequestsPerJob    = 3
)

const (
	ApprovalScopeNone  = "none"
	ApprovalEffectNone = "none"
)

type ApprovalRequestContent struct {
	ProposedAction   string        `json:"proposed_action,omitempty"`
	WhyNeeded        string        `json:"why_needed,omitempty"`
	AuthorityTier    AuthorityTier `json:"authority_tier,omitempty"`
	IdentityScope    string        `json:"identity_scope,omitempty"`
	PublicScope      string        `json:"public_scope,omitempty"`
	FilesystemEffect string        `json:"filesystem_effect,omitempty"`
	ProcessEffect    string        `json:"process_effect,omitempty"`
	NetworkEffect    string        `json:"network_effect,omitempty"`
	FallbackIfDenied string        `json:"fallback_if_denied,omitempty"`
}

type ApprovalRequest struct {
	JobID           string                  `json:"job_id"`
	StepID          string                  `json:"step_id"`
	RequestedAction string                  `json:"requested_action"`
	Scope           string                  `json:"scope"`
	Content         *ApprovalRequestContent `json:"content,omitempty"`
	RequestedVia    string                  `json:"requested_via"`
	GrantedVia      string                  `json:"granted_via,omitempty"`
	SessionChannel  string                  `json:"session_channel,omitempty"`
	SessionChatID   string                  `json:"session_chat_id,omitempty"`
	State           ApprovalState           `json:"state"`
	Reason          string                  `json:"reason,omitempty"`
	RequestedAt     time.Time               `json:"requested_at,omitempty"`
	ExpiresAt       time.Time               `json:"expires_at,omitempty"`
	ResolvedAt      time.Time               `json:"resolved_at,omitempty"`
	SupersededAt    time.Time               `json:"superseded_at,omitempty"`
	RevokedAt       time.Time               `json:"revoked_at,omitempty"`
}

type ApprovalGrant struct {
	JobID           string        `json:"job_id"`
	StepID          string        `json:"step_id"`
	RequestedAction string        `json:"requested_action"`
	Scope           string        `json:"scope"`
	GrantedVia      string        `json:"granted_via"`
	SessionChannel  string        `json:"session_channel,omitempty"`
	SessionChatID   string        `json:"session_chat_id,omitempty"`
	State           ApprovalState `json:"state"`
	GrantedAt       time.Time     `json:"granted_at,omitempty"`
	ExpiresAt       time.Time     `json:"expires_at,omitempty"`
	RevokedAt       time.Time     `json:"revoked_at,omitempty"`
}

func approvalBindingForStep(step Step) (string, string, bool) {
	if step.Subtype != StepSubtypeAuthorization {
		return "", "", false
	}
	switch step.Type {
	case StepTypeDiscussion, StepTypeWaitUser:
	default:
		return "", "", false
	}
	scope := normalizeApprovalScope(step.ApprovalScope)
	if scope == "" {
		scope = ApprovalScopeOneStep
	}
	return ApprovalRequestedActionStepComplete, scope, true
}

func approvalRequestContentForStep(job Job, step Step) (ApprovalRequestContent, bool) {
	if step.Subtype != StepSubtypeAuthorization {
		return ApprovalRequestContent{}, false
	}
	switch step.Type {
	case StepTypeDiscussion, StepTypeWaitUser:
	default:
		return ApprovalRequestContent{}, false
	}

	authorityTier := step.RequiredAuthority
	if authorityTier == "" {
		authorityTier = job.MaxAuthority
	}

	return ApprovalRequestContent{
		ProposedAction:   "Complete the authorization discussion step and continue to the next mission step.",
		WhyNeeded:        "This step asks the operator to explicitly approve continuation before the mission can proceed.",
		AuthorityTier:    authorityTier,
		IdentityScope:    ApprovalScopeNone,
		PublicScope:      ApprovalScopeNone,
		FilesystemEffect: ApprovalEffectNone,
		ProcessEffect:    ApprovalEffectNone,
		NetworkEffect:    ApprovalEffectNone,
		FallbackIfDenied: "Keep the mission in waiting_user and require an explicit follow-up decision before proceeding.",
	}, true
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
	return request.RequestedAction == requestedAction && normalizeApprovalScope(request.Scope) == normalizeApprovalScope(scope)
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
		if !approvalRequestMatchesBinding(request, jobID, stepID, requestedAction, scope) {
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
		request.RevokedAt = time.Time{}
		changed = true
	}
	if changed {
		next.UpdatedAt = now
	}
	return next, changed
}

func NormalizeHydratedApprovalRequests(current JobRuntimeState, now time.Time) (JobRuntimeState, bool) {
	next := *CloneJobRuntimeState(&current)
	changed := false
	if current.State == JobStateWaitingUser {
		var refreshed bool
		next, refreshed = RefreshApprovalRequests(current, now)
		changed = refreshed
	}
	if normalizeLegacyRevokedApprovalRequests(&next) {
		changed = true
		next.UpdatedAt = now
	}
	return next, changed
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
	next.ApprovalRequests[requestIndex].RevokedAt = time.Time{}
	next.UpdatedAt = now
	return next, true
}

func appendPendingApprovalRequest(current JobRuntimeState, now time.Time, request ApprovalRequest) JobRuntimeState {
	next := *CloneJobRuntimeState(&current)
	request.Scope = normalizeApprovalScope(request.Scope)
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
		next.ApprovalRequests[i].RevokedAt = time.Time{}
	}

	request.State = ApprovalStatePending
	request.GrantedVia = ""
	request.RequestedAt = now
	if request.ExpiresAt.IsZero() {
		request.ExpiresAt = now.Add(defaultApprovalRequestTTL)
	}
	request.SupersededAt = time.Time{}
	request.ResolvedAt = time.Time{}
	request.RevokedAt = time.Time{}
	next.ApprovalRequests = append(next.ApprovalRequests, request)
	next.UpdatedAt = now
	return next
}

func appendPendingApprovalRequestWithinBudget(current JobRuntimeState, now time.Time, request ApprovalRequest) (JobRuntimeState, bool, error) {
	next, _ := RefreshApprovalRequests(current, now)
	observed := countPendingApprovalRequests(next, request.JobID)
	if observed >= maxPendingApprovalRequestsPerJob {
		paused, err := PauseJobRuntimeForBudgetExhaustion(next, now, RuntimeBudgetBlockerRecord{
			Ceiling:  "pending_approvals",
			Limit:    maxPendingApprovalRequestsPerJob,
			Observed: observed,
			Message:  "pending approval request budget exhausted",
		})
		return paused, true, err
	}

	return appendPendingApprovalRequest(next, now, request), false, nil
}

func countPendingApprovalRequests(runtime JobRuntimeState, jobID string) int {
	if strings.TrimSpace(jobID) == "" {
		return 0
	}

	count := 0
	for _, request := range runtime.ApprovalRequests {
		if request.JobID != jobID || request.State != ApprovalStatePending {
			continue
		}
		count++
	}
	return count
}

func ApplyApprovalDecision(ec ExecutionContext, now time.Time, decision ApprovalDecision, via string) (JobRuntimeState, error) {
	return ApplyApprovalDecisionWithSession(ec, now, decision, via, "", "")
}

func ApplyApprovalDecisionWithSession(ec ExecutionContext, now time.Time, decision ApprovalDecision, via string, sessionChannel string, sessionChatID string) (JobRuntimeState, error) {
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
	next.ApprovalRequests[requestIndex].SessionChannel = strings.TrimSpace(sessionChannel)
	next.ApprovalRequests[requestIndex].SessionChatID = strings.TrimSpace(sessionChatID)
	next.ApprovalRequests[requestIndex].RevokedAt = time.Time{}
	switch decision {
	case ApprovalDecisionApprove:
		next.ApprovalRequests[requestIndex].State = ApprovalStateGranted
		next.ApprovalRequests[requestIndex].GrantedVia = via
		next.ApprovalGrants = append(next.ApprovalGrants, ApprovalGrant{
			JobID:           ec.Job.ID,
			StepID:          ec.Step.ID,
			RequestedAction: requestedAction,
			Scope:           normalizeApprovalScope(scope),
			GrantedVia:      via,
			SessionChannel:  next.ApprovalRequests[requestIndex].SessionChannel,
			SessionChatID:   next.ApprovalRequests[requestIndex].SessionChatID,
			State:           ApprovalStateGranted,
			GrantedAt:       now,
			ExpiresAt:       next.ApprovalRequests[requestIndex].ExpiresAt,
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

func RevokeLatestApprovalGrantWithSession(ec ExecutionContext, now time.Time, sessionChannel string, sessionChatID string) (JobRuntimeState, error) {
	if ec.Job == nil || ec.Step == nil || ec.Runtime == nil {
		return JobRuntimeState{}, ValidationError{
			Code:    RejectionCodeInvalidRuntimeState,
			Message: "approval revocation requires active job, step, and runtime state",
		}
	}
	if ec.Runtime.ActiveStepID == "" || ec.Runtime.ActiveStepID != ec.Step.ID {
		return JobRuntimeState{}, ValidationError{
			Code:    RejectionCodeInvalidRuntimeState,
			StepID:  ec.Step.ID,
			Message: "approval revocation requires an active step",
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
	sessionChannel = strings.TrimSpace(sessionChannel)
	sessionChatID = strings.TrimSpace(sessionChatID)

	requestIndex, ok := findLatestReusableApprovalRequest(next.ApprovalRequests, ec.Job.ID, ec.Step.ID, requestedAction, scope, sessionChannel, sessionChatID)
	if !ok || next.ApprovalRequests[requestIndex].State != ApprovalStateGranted {
		return JobRuntimeState{}, ValidationError{
			Code:    RejectionCodeStepValidationFailed,
			StepID:  ec.Step.ID,
			Message: "no granted approval matches the active job and step",
		}
	}

	grantIndex, ok := findLatestReusableApprovalGrant(next.ApprovalGrants, ec.Job.ID, ec.Step.ID, requestedAction, scope, sessionChannel, sessionChatID)
	if !ok || next.ApprovalGrants[grantIndex].State != ApprovalStateGranted {
		return JobRuntimeState{}, ValidationError{
			Code:    RejectionCodeStepValidationFailed,
			StepID:  ec.Step.ID,
			Message: "no granted approval matches the active job and step",
		}
	}

	request := &next.ApprovalRequests[requestIndex]
	request.State = ApprovalStateRevoked
	request.ResolvedAt = now
	request.SupersededAt = time.Time{}
	request.RevokedAt = now

	grant := &next.ApprovalGrants[grantIndex]
	grant.State = ApprovalStateRevoked
	grant.RevokedAt = now

	next.UpdatedAt = now
	return next, nil
}

func findPendingApprovalRequest(requests []ApprovalRequest, jobID, stepID, requestedAction, scope string) (int, bool) {
	for i := len(requests) - 1; i >= 0; i-- {
		request := requests[i]
		if request.JobID != jobID || request.StepID != stepID {
			continue
		}
		if request.RequestedAction != requestedAction || normalizeApprovalScope(request.Scope) != normalizeApprovalScope(scope) {
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
	return approvalBindingScopeMatches(left.JobID, left.StepID, left.RequestedAction, left.Scope, right.JobID, right.StepID, right.RequestedAction, right.Scope)
}

func FindReusableApprovalGrant(runtime JobRuntimeState, now time.Time, jobID string, step Step, sessionChannel string, sessionChatID string) (ApprovalGrant, bool) {
	requestedAction, scope, ok := approvalBindingForStep(step)
	scope = normalizeApprovalScope(scope)
	if !ok || (scope != ApprovalScopeOneJob && scope != ApprovalScopeOneSession) {
		return ApprovalGrant{}, false
	}
	sessionChannel = strings.TrimSpace(sessionChannel)
	sessionChatID = strings.TrimSpace(sessionChatID)
	if scope == ApprovalScopeOneSession && (sessionChannel == "" || sessionChatID == "") {
		return ApprovalGrant{}, false
	}

	if requestIndex, ok := findLatestReusableApprovalRequest(runtime.ApprovalRequests, jobID, step.ID, requestedAction, scope, sessionChannel, sessionChatID); ok {
		switch runtime.ApprovalRequests[requestIndex].State {
		case ApprovalStatePending, ApprovalStateDenied, ApprovalStateExpired, ApprovalStateSuperseded, ApprovalStateRevoked:
			return ApprovalGrant{}, false
		}
	}

	grantIndex, ok := findLatestReusableApprovalGrant(runtime.ApprovalGrants, jobID, step.ID, requestedAction, scope, sessionChannel, sessionChatID)
	if !ok {
		return ApprovalGrant{}, false
	}
	grant := runtime.ApprovalGrants[grantIndex]
	if grant.State != ApprovalStateGranted {
		return ApprovalGrant{}, false
	}
	if !grant.RevokedAt.IsZero() && !grant.RevokedAt.After(now) {
		return ApprovalGrant{}, false
	}
	if !grant.ExpiresAt.IsZero() && !grant.ExpiresAt.After(now) {
		return ApprovalGrant{}, false
	}
	return grant, true
}

func appendGrantedApprovalRequest(current JobRuntimeState, now time.Time, request ApprovalRequest) JobRuntimeState {
	next := *CloneJobRuntimeState(&current)
	request.Scope = normalizeApprovalScope(request.Scope)
	request.State = ApprovalStateGranted
	request.RequestedAt = now
	request.ResolvedAt = now
	request.SupersededAt = time.Time{}
	request.RevokedAt = time.Time{}
	next.ApprovalRequests = append(next.ApprovalRequests, request)
	next.UpdatedAt = now
	return next
}

func normalizeLegacyRevokedApprovalRequests(runtime *JobRuntimeState) bool {
	if runtime == nil || len(runtime.ApprovalRequests) == 0 || len(runtime.ApprovalGrants) == 0 {
		return false
	}

	changed := false
	for i := range runtime.ApprovalRequests {
		request := &runtime.ApprovalRequests[i]
		if request.State != ApprovalStateRevoked || !request.RevokedAt.IsZero() {
			continue
		}
		revokedAt := legacyApprovalRequestRevokedAt(*request, runtime.ApprovalGrants)
		if revokedAt.IsZero() {
			continue
		}
		request.RevokedAt = revokedAt
		changed = true
	}
	return changed
}

func legacyApprovalRequestRevokedAt(request ApprovalRequest, grants []ApprovalGrant) time.Time {
	if request.State != ApprovalStateRevoked {
		return time.Time{}
	}
	if !request.RevokedAt.IsZero() {
		return request.RevokedAt
	}

	for i := len(grants) - 1; i >= 0; i-- {
		grant := grants[i]
		if grant.State != ApprovalStateRevoked || grant.RevokedAt.IsZero() {
			continue
		}
		if !approvalReusableBindingMatches(
			request.JobID,
			request.StepID,
			request.RequestedAction,
			request.Scope,
			request.SessionChannel,
			request.SessionChatID,
			grant.JobID,
			grant.StepID,
			grant.RequestedAction,
			grant.Scope,
			grant.SessionChannel,
			grant.SessionChatID,
		) {
			continue
		}
		return grant.RevokedAt
	}

	return time.Time{}
}

func normalizeApprovalScope(scope string) string {
	switch strings.ToLower(strings.TrimSpace(scope)) {
	case "", ApprovalScopeMissionStep, "one_step":
		return ApprovalScopeMissionStep
	case ApprovalScopeOneJob:
		return ApprovalScopeOneJob
	case ApprovalScopeOneSession:
		return ApprovalScopeOneSession
	default:
		return scope
	}
}

func approvalBindingScopeMatches(leftJobID, leftStepID, leftAction, leftScope, rightJobID, rightStepID, rightAction, rightScope string) bool {
	leftScope = normalizeApprovalScope(leftScope)
	rightScope = normalizeApprovalScope(rightScope)
	if leftJobID != rightJobID || leftAction != rightAction || leftScope != rightScope {
		return false
	}
	if leftScope == ApprovalScopeOneJob {
		return true
	}
	return leftStepID == rightStepID
}

func approvalReusableBindingMatches(leftJobID, leftStepID, leftAction, leftScope, leftSessionChannel, leftSessionChatID, rightJobID, rightStepID, rightAction, rightScope, rightSessionChannel, rightSessionChatID string) bool {
	leftScope = normalizeApprovalScope(leftScope)
	rightScope = normalizeApprovalScope(rightScope)
	if leftJobID != rightJobID || leftAction != rightAction || leftScope != rightScope {
		return false
	}
	switch leftScope {
	case ApprovalScopeOneJob:
		return true
	case ApprovalScopeOneSession:
		return approvalSessionMatches(leftSessionChannel, leftSessionChatID, rightSessionChannel, rightSessionChatID)
	default:
		return leftStepID == rightStepID
	}
}

func approvalSessionMatches(leftChannel, leftChatID, rightChannel, rightChatID string) bool {
	leftChannel = strings.TrimSpace(leftChannel)
	leftChatID = strings.TrimSpace(leftChatID)
	rightChannel = strings.TrimSpace(rightChannel)
	rightChatID = strings.TrimSpace(rightChatID)
	if leftChannel == "" || leftChatID == "" || rightChannel == "" || rightChatID == "" {
		return false
	}
	return leftChannel == rightChannel && leftChatID == rightChatID
}

func approvalRequestMatchesBinding(request ApprovalRequest, jobID, stepID, requestedAction, scope string) bool {
	return approvalBindingScopeMatches(request.JobID, request.StepID, request.RequestedAction, request.Scope, jobID, stepID, requestedAction, scope)
}

func approvalGrantMatchesBinding(grant ApprovalGrant, jobID, stepID, requestedAction, scope string) bool {
	return approvalBindingScopeMatches(grant.JobID, grant.StepID, grant.RequestedAction, grant.Scope, jobID, stepID, requestedAction, scope)
}

func approvalRequestMatchesReusableBinding(request ApprovalRequest, jobID, stepID, requestedAction, scope, sessionChannel, sessionChatID string) bool {
	return approvalReusableBindingMatches(request.JobID, request.StepID, request.RequestedAction, request.Scope, request.SessionChannel, request.SessionChatID, jobID, stepID, requestedAction, scope, sessionChannel, sessionChatID)
}

func approvalGrantMatchesReusableBinding(grant ApprovalGrant, jobID, stepID, requestedAction, scope, sessionChannel, sessionChatID string) bool {
	return approvalReusableBindingMatches(grant.JobID, grant.StepID, grant.RequestedAction, grant.Scope, grant.SessionChannel, grant.SessionChatID, jobID, stepID, requestedAction, scope, sessionChannel, sessionChatID)
}

func findLatestApprovalGrant(grants []ApprovalGrant, jobID, stepID, requestedAction, scope string) (int, bool) {
	for i := len(grants) - 1; i >= 0; i-- {
		if !approvalGrantMatchesBinding(grants[i], jobID, stepID, requestedAction, scope) {
			continue
		}
		return i, true
	}
	return 0, false
}

func findLatestReusableApprovalRequest(requests []ApprovalRequest, jobID, stepID, requestedAction, scope, sessionChannel, sessionChatID string) (int, bool) {
	bestIndex := -1
	for i, request := range requests {
		if !approvalRequestMatchesReusableBinding(request, jobID, stepID, requestedAction, scope, sessionChannel, sessionChatID) {
			continue
		}
		if bestIndex < 0 || reusableApprovalRequestSortsAfter(request, requests[bestIndex]) {
			bestIndex = i
		}
	}
	if bestIndex < 0 {
		return 0, false
	}
	return bestIndex, true
}

func findLatestReusableApprovalGrant(grants []ApprovalGrant, jobID, stepID, requestedAction, scope, sessionChannel, sessionChatID string) (int, bool) {
	bestIndex := -1
	for i, grant := range grants {
		if !approvalGrantMatchesReusableBinding(grant, jobID, stepID, requestedAction, scope, sessionChannel, sessionChatID) {
			continue
		}
		if bestIndex < 0 || reusableApprovalGrantSortsAfter(grant, grants[bestIndex]) {
			bestIndex = i
		}
	}
	if bestIndex < 0 {
		return 0, false
	}
	return bestIndex, true
}

func reusableApprovalRequestSortsAfter(left ApprovalRequest, right ApprovalRequest) bool {
	leftAt := approvalRequestDecisionTime(left)
	rightAt := approvalRequestDecisionTime(right)
	if leftAt.After(rightAt) {
		return true
	}
	if rightAt.After(leftAt) {
		return false
	}
	return approvalRequestStableKey(left) > approvalRequestStableKey(right)
}

func reusableApprovalGrantSortsAfter(left ApprovalGrant, right ApprovalGrant) bool {
	leftAt := approvalGrantDecisionTime(left)
	rightAt := approvalGrantDecisionTime(right)
	if leftAt.After(rightAt) {
		return true
	}
	if rightAt.After(leftAt) {
		return false
	}
	return approvalGrantStableKey(left) > approvalGrantStableKey(right)
}

func approvalRequestDecisionTime(request ApprovalRequest) time.Time {
	if !request.ResolvedAt.IsZero() {
		return request.ResolvedAt
	}
	return request.RequestedAt
}

func approvalGrantDecisionTime(grant ApprovalGrant) time.Time {
	if !grant.RevokedAt.IsZero() {
		return grant.RevokedAt
	}
	return grant.GrantedAt
}

func approvalRequestStableKey(request ApprovalRequest) string {
	return strings.Join([]string{
		request.JobID,
		request.StepID,
		request.RequestedAction,
		normalizeApprovalScope(request.Scope),
		strings.TrimSpace(request.SessionChannel),
		strings.TrimSpace(request.SessionChatID),
		string(request.State),
		request.RequestedVia,
		request.GrantedVia,
		approvalRequestDecisionTime(request).UTC().Format(time.RFC3339Nano),
		request.RequestedAt.UTC().Format(time.RFC3339Nano),
		request.ResolvedAt.UTC().Format(time.RFC3339Nano),
	}, "\x00")
}

func approvalGrantStableKey(grant ApprovalGrant) string {
	return strings.Join([]string{
		grant.JobID,
		grant.StepID,
		grant.RequestedAction,
		normalizeApprovalScope(grant.Scope),
		strings.TrimSpace(grant.SessionChannel),
		strings.TrimSpace(grant.SessionChatID),
		string(grant.State),
		grant.GrantedVia,
		approvalGrantDecisionTime(grant).UTC().Format(time.RFC3339Nano),
		grant.GrantedAt.UTC().Format(time.RFC3339Nano),
		grant.RevokedAt.UTC().Format(time.RFC3339Nano),
	}, "\x00")
}
