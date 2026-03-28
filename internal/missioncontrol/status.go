package missioncontrol

import (
	"encoding/json"
	"sort"
	"time"
)

const OperatorStatusRecentAuditLimit = 5
const OperatorStatusApprovalHistoryLimit = 5
const OperatorStatusArtifactLimit = 5

type OperatorStatusSummary struct {
	JobID           string                         `json:"job_id"`
	State           JobState                       `json:"state"`
	ActiveStepID    string                         `json:"active_step_id,omitempty"`
	AllowedTools    []string                       `json:"allowed_tools,omitempty"`
	WaitingReason   string                         `json:"waiting_reason,omitempty"`
	WaitingAt       *string                        `json:"waiting_at,omitempty"`
	PausedReason    string                         `json:"paused_reason,omitempty"`
	PausedAt        *string                        `json:"paused_at,omitempty"`
	AbortedReason   string                         `json:"aborted_reason,omitempty"`
	FailedStepID    string                         `json:"failed_step_id,omitempty"`
	FailureReason   string                         `json:"failure_reason,omitempty"`
	FailedAt        *string                        `json:"failed_at,omitempty"`
	ApprovalRequest *OperatorApprovalRequestStatus `json:"approval_request,omitempty"`
	ApprovalHistory []OperatorApprovalHistoryEntry `json:"approval_history,omitempty"`
	RecentAudit     []OperatorRecentAuditStatus    `json:"recent_audit,omitempty"`
	Artifacts       []OperatorArtifactStatus       `json:"artifacts,omitempty"`
	Truncation      *OperatorStatusTruncation      `json:"truncation,omitempty"`
}

type OperatorStatusTruncation struct {
	ApprovalHistoryOmitted int `json:"approval_history_omitted,omitempty"`
	RecentAuditOmitted     int `json:"recent_audit_omitted,omitempty"`
	ArtifactsOmitted       int `json:"artifacts_omitted,omitempty"`
}

type OperatorArtifactStatus struct {
	StepID   string   `json:"step_id"`
	StepType StepType `json:"step_type"`
	Path     string   `json:"path"`
	State    string   `json:"state,omitempty"`
}

type OperatorApprovalRequestStatus struct {
	State            ApprovalState `json:"state"`
	StepID           string        `json:"step_id"`
	RequestedAction  string        `json:"requested_action"`
	Scope            string        `json:"scope"`
	RequestedVia     string        `json:"requested_via,omitempty"`
	GrantedVia       string        `json:"granted_via,omitempty"`
	SessionChannel   string        `json:"session_channel,omitempty"`
	SessionChatID    string        `json:"session_chat_id,omitempty"`
	ProposedAction   string        `json:"proposed_action,omitempty"`
	WhyNeeded        string        `json:"why_needed,omitempty"`
	AuthorityTier    AuthorityTier `json:"authority_tier,omitempty"`
	FallbackIfDenied string        `json:"fallback_if_denied,omitempty"`
	ExpiresAt        *string       `json:"expires_at,omitempty"`
	SupersededAt     *string       `json:"superseded_at,omitempty"`
}

type OperatorRecentAuditStatus struct {
	EventID     string           `json:"event_id,omitempty"`
	JobID       string           `json:"job_id"`
	StepID      string           `json:"step_id,omitempty"`
	Action      string           `json:"action"`
	ActionClass AuditActionClass `json:"action_class,omitempty"`
	Result      AuditResult      `json:"result,omitempty"`
	Allowed     bool             `json:"allowed"`
	Code        RejectionCode    `json:"error_code,omitempty"`
	Timestamp   string           `json:"timestamp"`
}

type OperatorApprovalHistoryEntry struct {
	StepID          string  `json:"step_id"`
	RequestedAction string  `json:"requested_action"`
	Scope           string  `json:"scope"`
	State           string  `json:"state"`
	RequestedVia    string  `json:"requested_via"`
	GrantedVia      string  `json:"granted_via"`
	SessionChannel  string  `json:"session_channel"`
	SessionChatID   string  `json:"session_chat_id"`
	RequestedAt     *string `json:"requested_at"`
	ResolvedAt      *string `json:"resolved_at"`
	ExpiresAt       *string `json:"expires_at"`
	RevokedAt       *string `json:"revoked_at,omitempty"`
}

func BuildOperatorStatusSummary(runtime JobRuntimeState) OperatorStatusSummary {
	return buildOperatorStatusSummary(runtime, nil)
}

func BuildOperatorStatusSummaryWithAllowedTools(runtime JobRuntimeState, allowedTools []string) OperatorStatusSummary {
	return buildOperatorStatusSummary(runtime, allowedTools)
}

func FormatOperatorStatusSummary(runtime JobRuntimeState) (string, error) {
	return formatOperatorStatusSummary(buildOperatorStatusSummary(runtime, nil))
}

func FormatOperatorStatusSummaryWithAllowedTools(runtime JobRuntimeState, allowedTools []string) (string, error) {
	return formatOperatorStatusSummary(buildOperatorStatusSummary(runtime, allowedTools))
}

func EffectiveAllowedTools(job *Job, step *Step) []string {
	if job == nil {
		return nil
	}

	jobTools := make(map[string]struct{}, len(job.AllowedTools))
	for _, toolName := range job.AllowedTools {
		jobTools[toolName] = struct{}{}
	}

	allowed := make([]string, 0, len(jobTools))
	if step == nil || len(step.AllowedTools) == 0 {
		for toolName := range jobTools {
			allowed = append(allowed, toolName)
		}
		sort.Strings(allowed)
		return allowed
	}

	for _, toolName := range step.AllowedTools {
		if _, ok := jobTools[toolName]; ok {
			allowed = append(allowed, toolName)
			delete(jobTools, toolName)
		}
	}
	sort.Strings(allowed)
	return allowed
}

func buildOperatorStatusSummary(runtime JobRuntimeState, allowedTools []string) OperatorStatusSummary {
	summary := OperatorStatusSummary{
		JobID:         runtime.JobID,
		State:         runtime.State,
		ActiveStepID:  runtime.ActiveStepID,
		WaitingReason: runtime.WaitingReason,
		WaitingAt:     formatOperatorStatusTime(runtime.WaitingAt),
		PausedReason:  runtime.PausedReason,
		PausedAt:      formatOperatorStatusTime(runtime.PausedAt),
		AbortedReason: runtime.AbortedReason,
	}
	if runtime.State == JobStateFailed {
		if record, ok := selectOperatorStatusLatestFailedStep(runtime); ok {
			summary.FailedStepID = record.StepID
			summary.FailureReason = record.Reason
		}
		summary.FailedAt = selectOperatorStatusFailedAt(runtime)
	}
	if allowedTools != nil {
		summary.AllowedTools = append([]string(nil), allowedTools...)
	}

	if request, ok := selectOperatorStatusApprovalRequest(runtime); ok {
		status := OperatorApprovalRequestStatus{
			State:           request.State,
			StepID:          request.StepID,
			RequestedAction: request.RequestedAction,
			Scope:           request.Scope,
			RequestedVia:    request.RequestedVia,
			GrantedVia:      request.GrantedVia,
			SessionChannel:  request.SessionChannel,
			SessionChatID:   request.SessionChatID,
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
	summary.ApprovalHistory = selectOperatorStatusApprovalHistory(runtime)
	summary.RecentAudit = selectOperatorStatusRecentAudit(runtime)
	summary.Artifacts = selectOperatorStatusArtifacts(runtime)
	summary.Truncation = buildOperatorStatusTruncation(runtime, len(summary.ApprovalHistory), len(summary.RecentAudit), len(summary.Artifacts))

	return summary
}

func selectOperatorStatusLatestFailedStep(runtime JobRuntimeState) (RuntimeStepRecord, bool) {
	if len(runtime.FailedSteps) == 0 {
		return RuntimeStepRecord{}, false
	}
	return cloneRuntimeStepRecord(runtime.FailedSteps[len(runtime.FailedSteps)-1]), true
}

func selectOperatorStatusFailedAt(runtime JobRuntimeState) *string {
	if failedAt := formatOperatorStatusTime(runtime.FailedAt); failedAt != nil {
		return failedAt
	}
	if record, ok := selectOperatorStatusLatestFailedStep(runtime); ok {
		return formatOperatorStatusTime(record.At)
	}
	return nil
}

func formatOperatorStatusSummary(summary OperatorStatusSummary) (string, error) {
	data, err := json.MarshalIndent(summary, "", "  ")
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

func selectOperatorStatusRecentAudit(runtime JobRuntimeState) []OperatorRecentAuditStatus {
	if len(runtime.AuditHistory) == 0 {
		return nil
	}

	count := OperatorStatusRecentAuditLimit
	if len(runtime.AuditHistory) < count {
		count = len(runtime.AuditHistory)
	}

	recent := make([]OperatorRecentAuditStatus, 0, count)
	for i := len(runtime.AuditHistory) - 1; i >= len(runtime.AuditHistory)-count; i-- {
		event := normalizeAuditEvent(runtime.AuditHistory[i])
		recent = append(recent, OperatorRecentAuditStatus{
			EventID:     event.EventID,
			JobID:       event.JobID,
			StepID:      event.StepID,
			Action:      event.ToolName,
			ActionClass: event.ActionClass,
			Result:      event.Result,
			Allowed:     event.Allowed,
			Code:        event.Code,
			Timestamp:   event.Timestamp.UTC().Format(time.RFC3339Nano),
		})
	}

	return recent
}

func selectOperatorStatusApprovalHistory(runtime JobRuntimeState) []OperatorApprovalHistoryEntry {
	if len(runtime.ApprovalRequests) == 0 {
		return nil
	}

	history := make([]OperatorApprovalHistoryEntry, 0, minInt(len(runtime.ApprovalRequests), OperatorStatusApprovalHistoryLimit))
	for i := len(runtime.ApprovalRequests) - 1; i >= 0 && len(history) < OperatorStatusApprovalHistoryLimit; i-- {
		request := runtime.ApprovalRequests[i]
		if runtime.JobID != "" && request.JobID != runtime.JobID {
			continue
		}

		entry := OperatorApprovalHistoryEntry{
			StepID:          request.StepID,
			RequestedAction: request.RequestedAction,
			Scope:           request.Scope,
			State:           string(request.State),
			RequestedVia:    request.RequestedVia,
			GrantedVia:      request.GrantedVia,
			SessionChannel:  request.SessionChannel,
			SessionChatID:   request.SessionChatID,
			RequestedAt:     formatOperatorStatusTime(request.RequestedAt),
			ResolvedAt:      formatOperatorStatusTime(request.ResolvedAt),
			ExpiresAt:       formatOperatorStatusTime(request.ExpiresAt),
		}
		if revokedAt := findOperatorStatusApprovalRevokedAt(runtime.ApprovalGrants, request); revokedAt != nil {
			entry.RevokedAt = revokedAt
		}

		history = append(history, entry)
	}

	if len(history) == 0 {
		return nil
	}
	return history
}

func selectOperatorStatusArtifacts(runtime JobRuntimeState) []OperatorArtifactStatus {
	candidates := collectOperatorStatusArtifactCandidates(runtime)
	if len(candidates) == 0 {
		return nil
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		left := candidates[i]
		right := candidates[j]
		if left.planIndex != right.planIndex {
			return left.planIndex < right.planIndex
		}
		if left.Path != right.Path {
			return left.Path < right.Path
		}
		if left.StepID != right.StepID {
			return left.StepID < right.StepID
		}
		return left.StepType < right.StepType
	})

	if len(candidates) > OperatorStatusArtifactLimit {
		candidates = candidates[:OperatorStatusArtifactLimit]
	}

	artifacts := make([]OperatorArtifactStatus, len(candidates))
	for i, candidate := range candidates {
		artifacts[i] = OperatorArtifactStatus{
			StepID:   candidate.StepID,
			StepType: candidate.StepType,
			Path:     candidate.Path,
			State:    candidate.State,
		}
	}
	return artifacts
}

type operatorArtifactCandidate struct {
	StepID    string
	StepType  StepType
	Path      string
	State     string
	planIndex int
}

func collectOperatorStatusArtifactCandidates(runtime JobRuntimeState) []operatorArtifactCandidate {
	if len(runtime.CompletedSteps) == 0 {
		return nil
	}

	stepByID := make(map[string]Step, len(runtime.CompletedSteps))
	stepOrderIndex := make(map[string]int, len(runtime.CompletedSteps))
	defaultPlanIndex := len(runtime.CompletedSteps) + 1
	if runtime.InspectablePlan != nil {
		for i, step := range runtime.InspectablePlan.Steps {
			stepByID[step.ID] = copyStep(step)
			stepOrderIndex[step.ID] = i
		}
		defaultPlanIndex = len(runtime.InspectablePlan.Steps) + 1
	}

	seen := make(map[string]struct{}, len(runtime.CompletedSteps))
	candidates := make([]operatorArtifactCandidate, 0, len(runtime.CompletedSteps))
	for i := len(runtime.CompletedSteps) - 1; i >= 0; i-- {
		record := runtime.CompletedSteps[i]
		step, hasStep := stepByID[record.StepID]

		stepType, path, ok := operatorStatusArtifactRecord(record, step, hasStep)
		if !ok {
			continue
		}

		key := string(stepType) + "\x1f" + record.StepID + "\x1f" + path
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}

		state := ""
		if record.ResultingState != nil {
			state = record.ResultingState.State
		}

		planIndex := defaultPlanIndex
		if index, ok := stepOrderIndex[record.StepID]; ok {
			planIndex = index
		}
		candidates = append(candidates, operatorArtifactCandidate{
			StepID:    record.StepID,
			StepType:  stepType,
			Path:      path,
			State:     state,
			planIndex: planIndex,
		})
	}

	return candidates
}

func operatorStatusArtifactRecord(record RuntimeStepRecord, step Step, hasStep bool) (StepType, string, bool) {
	if hasStep {
		if path, ok := operatorStatusArtifactPathForStep(step); ok {
			if record.ResultingState != nil && isOperatorStatusArtifactStepType(StepType(record.ResultingState.Kind)) {
				if target := cleanedArtifactPath(record.ResultingState.Target); target != "" {
					return step.Type, target, true
				}
			}
			return step.Type, path, true
		}
	}

	if record.ResultingState == nil {
		return "", "", false
	}

	stepType := StepType(record.ResultingState.Kind)
	if !isOperatorStatusArtifactStepType(stepType) {
		return "", "", false
	}
	path := cleanedArtifactPath(record.ResultingState.Target)
	if path == "" {
		return "", "", false
	}
	return stepType, path, true
}

func operatorStatusArtifactPathForStep(step Step) (string, bool) {
	switch step.Type {
	case StepTypeStaticArtifact:
		if path := staticArtifactPath(step); path != "" {
			return path, true
		}
	case StepTypeOneShotCode:
		if path := oneShotArtifactPath(step); path != "" {
			return path, true
		}
	case StepTypeLongRunningCode:
		if path := longRunningArtifactPath(step); path != "" {
			return path, true
		}
	}
	return "", false
}

func isOperatorStatusArtifactStepType(stepType StepType) bool {
	switch stepType {
	case StepTypeStaticArtifact, StepTypeOneShotCode, StepTypeLongRunningCode:
		return true
	default:
		return false
	}
}

func buildOperatorStatusTruncation(runtime JobRuntimeState, shownApprovalHistory int, shownRecentAudit int, shownArtifacts int) *OperatorStatusTruncation {
	truncation := OperatorStatusTruncation{}

	approvalHistoryTotal := countOperatorStatusApprovalHistory(runtime)
	if approvalHistoryTotal > shownApprovalHistory {
		truncation.ApprovalHistoryOmitted = approvalHistoryTotal - shownApprovalHistory
	}
	if len(runtime.AuditHistory) > shownRecentAudit {
		truncation.RecentAuditOmitted = len(runtime.AuditHistory) - shownRecentAudit
	}
	if artifactsTotal := len(collectOperatorStatusArtifactCandidates(runtime)); artifactsTotal > shownArtifacts {
		truncation.ArtifactsOmitted = artifactsTotal - shownArtifacts
	}
	if truncation.ApprovalHistoryOmitted == 0 && truncation.RecentAuditOmitted == 0 && truncation.ArtifactsOmitted == 0 {
		return nil
	}
	return &truncation
}

func countOperatorStatusApprovalHistory(runtime JobRuntimeState) int {
	count := 0
	for _, request := range runtime.ApprovalRequests {
		if runtime.JobID != "" && request.JobID != runtime.JobID {
			continue
		}
		count++
	}
	return count
}

func findOperatorStatusApprovalRevokedAt(grants []ApprovalGrant, request ApprovalRequest) *string {
	if request.State != ApprovalStateRevoked {
		return nil
	}
	if revokedAt := formatOperatorStatusTime(legacyApprovalRequestRevokedAt(request, grants)); revokedAt != nil {
		return revokedAt
	}
	return nil
}

func formatOperatorStatusTime(at time.Time) *string {
	if at.IsZero() {
		return nil
	}
	formatted := at.UTC().Format(time.RFC3339Nano)
	return &formatted
}

func minInt(left, right int) int {
	if left < right {
		return left
	}
	return right
}
