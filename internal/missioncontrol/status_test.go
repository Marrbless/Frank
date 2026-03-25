package missioncontrol

import (
	"encoding/json"
	"testing"
	"time"
)

func TestBuildOperatorStatusSummaryIncludesLatestApprovalForActiveStep(t *testing.T) {
	t.Parallel()

	requestedAt := time.Date(2026, 3, 24, 12, 2, 0, 0, time.UTC)
	expiresAt := time.Date(2026, 3, 24, 12, 5, 0, 0, time.UTC)
	supersededAt := time.Date(2026, 3, 24, 12, 1, 0, 0, time.UTC)
	summary := BuildOperatorStatusSummary(JobRuntimeState{
		JobID:         "job-1",
		State:         JobStateWaitingUser,
		ActiveStepID:  "build",
		WaitingReason: "discussion_authorization",
		ApprovalRequests: []ApprovalRequest{
			{
				JobID:           "job-1",
				StepID:          "draft",
				RequestedAction: ApprovalRequestedActionStepComplete,
				Scope:           ApprovalScopeMissionStep,
				State:           ApprovalStateSuperseded,
				SupersededAt:    supersededAt,
			},
			{
				JobID:           "job-1",
				StepID:          "build",
				RequestedAction: ApprovalRequestedActionStepComplete,
				Scope:           ApprovalScopeMissionStep,
				RequestedVia:    ApprovalRequestedViaRuntime,
				SessionChannel:  "telegram",
				SessionChatID:   "chat-42",
				State:           ApprovalStatePending,
				RequestedAt:     requestedAt,
				ExpiresAt:       expiresAt,
				Content: &ApprovalRequestContent{
					ProposedAction:   "Continue build.",
					WhyNeeded:        "Operator approval is required.",
					AuthorityTier:    AuthorityTierMedium,
					FallbackIfDenied: "Stay waiting.",
				},
			},
		},
	})

	if summary.JobID != "job-1" {
		t.Fatalf("JobID = %q, want %q", summary.JobID, "job-1")
	}
	if summary.State != JobStateWaitingUser {
		t.Fatalf("State = %q, want %q", summary.State, JobStateWaitingUser)
	}
	if summary.ActiveStepID != "build" {
		t.Fatalf("ActiveStepID = %q, want %q", summary.ActiveStepID, "build")
	}
	if summary.WaitingReason != "discussion_authorization" {
		t.Fatalf("WaitingReason = %q, want %q", summary.WaitingReason, "discussion_authorization")
	}
	if summary.ApprovalRequest == nil {
		t.Fatal("ApprovalRequest = nil, want populated summary")
	}
	if summary.ApprovalRequest.StepID != "build" {
		t.Fatalf("ApprovalRequest.StepID = %q, want %q", summary.ApprovalRequest.StepID, "build")
	}
	if summary.ApprovalRequest.State != ApprovalStatePending {
		t.Fatalf("ApprovalRequest.State = %q, want %q", summary.ApprovalRequest.State, ApprovalStatePending)
	}
	if summary.ApprovalRequest.RequestedVia != ApprovalRequestedViaRuntime {
		t.Fatalf("ApprovalRequest.RequestedVia = %q, want %q", summary.ApprovalRequest.RequestedVia, ApprovalRequestedViaRuntime)
	}
	if summary.ApprovalRequest.GrantedVia != "" {
		t.Fatalf("ApprovalRequest.GrantedVia = %q, want empty", summary.ApprovalRequest.GrantedVia)
	}
	if summary.ApprovalRequest.SessionChannel != "telegram" {
		t.Fatalf("ApprovalRequest.SessionChannel = %q, want %q", summary.ApprovalRequest.SessionChannel, "telegram")
	}
	if summary.ApprovalRequest.SessionChatID != "chat-42" {
		t.Fatalf("ApprovalRequest.SessionChatID = %q, want %q", summary.ApprovalRequest.SessionChatID, "chat-42")
	}
	if summary.ApprovalRequest.ProposedAction != "Continue build." {
		t.Fatalf("ApprovalRequest.ProposedAction = %q, want %q", summary.ApprovalRequest.ProposedAction, "Continue build.")
	}
	if summary.ApprovalRequest.WhyNeeded != "Operator approval is required." {
		t.Fatalf("ApprovalRequest.WhyNeeded = %q, want %q", summary.ApprovalRequest.WhyNeeded, "Operator approval is required.")
	}
	if summary.ApprovalRequest.AuthorityTier != AuthorityTierMedium {
		t.Fatalf("ApprovalRequest.AuthorityTier = %q, want %q", summary.ApprovalRequest.AuthorityTier, AuthorityTierMedium)
	}
	if summary.ApprovalRequest.FallbackIfDenied != "Stay waiting." {
		t.Fatalf("ApprovalRequest.FallbackIfDenied = %q, want %q", summary.ApprovalRequest.FallbackIfDenied, "Stay waiting.")
	}
	if summary.ApprovalRequest.ExpiresAt == nil || *summary.ApprovalRequest.ExpiresAt != "2026-03-24T12:05:00Z" {
		t.Fatalf("ApprovalRequest.ExpiresAt = %#v, want RFC3339 time", summary.ApprovalRequest.ExpiresAt)
	}
	if summary.ApprovalRequest.SupersededAt != nil {
		t.Fatalf("ApprovalRequest.SupersededAt = %#v, want nil for active request", summary.ApprovalRequest.SupersededAt)
	}
	if len(summary.ApprovalHistory) != 2 {
		t.Fatalf("ApprovalHistory count = %d, want %d", len(summary.ApprovalHistory), 2)
	}
	if summary.ApprovalHistory[0].StepID != "build" {
		t.Fatalf("ApprovalHistory[0].StepID = %q, want %q", summary.ApprovalHistory[0].StepID, "build")
	}
	if summary.ApprovalHistory[0].RequestedAt == nil || *summary.ApprovalHistory[0].RequestedAt != "2026-03-24T12:02:00Z" {
		t.Fatalf("ApprovalHistory[0].RequestedAt = %#v, want RFC3339 time", summary.ApprovalHistory[0].RequestedAt)
	}
	if summary.ApprovalHistory[1].StepID != "draft" {
		t.Fatalf("ApprovalHistory[1].StepID = %q, want %q", summary.ApprovalHistory[1].StepID, "draft")
	}
}

func TestBuildOperatorStatusSummaryIncludesGrantedApprovalBindingMetadata(t *testing.T) {
	t.Parallel()

	summary := BuildOperatorStatusSummary(JobRuntimeState{
		JobID:        "job-1",
		State:        JobStatePaused,
		ActiveStepID: "build",
		ApprovalRequests: []ApprovalRequest{
			{
				JobID:           "job-1",
				StepID:          "build",
				RequestedAction: ApprovalRequestedActionStepComplete,
				Scope:           ApprovalScopeOneSession,
				RequestedVia:    ApprovalRequestedViaRuntime,
				GrantedVia:      ApprovalGrantedViaOperatorReply,
				SessionChannel:  "slack",
				SessionChatID:   "C123::171234",
				State:           ApprovalStateGranted,
			},
		},
	})

	if summary.ApprovalRequest == nil {
		t.Fatal("ApprovalRequest = nil, want populated summary")
	}
	if summary.ApprovalRequest.RequestedVia != ApprovalRequestedViaRuntime {
		t.Fatalf("ApprovalRequest.RequestedVia = %q, want %q", summary.ApprovalRequest.RequestedVia, ApprovalRequestedViaRuntime)
	}
	if summary.ApprovalRequest.GrantedVia != ApprovalGrantedViaOperatorReply {
		t.Fatalf("ApprovalRequest.GrantedVia = %q, want %q", summary.ApprovalRequest.GrantedVia, ApprovalGrantedViaOperatorReply)
	}
	if summary.ApprovalRequest.SessionChannel != "slack" {
		t.Fatalf("ApprovalRequest.SessionChannel = %q, want %q", summary.ApprovalRequest.SessionChannel, "slack")
	}
	if summary.ApprovalRequest.SessionChatID != "C123::171234" {
		t.Fatalf("ApprovalRequest.SessionChatID = %q, want %q", summary.ApprovalRequest.SessionChatID, "C123::171234")
	}
}

func TestBuildOperatorStatusSummaryIncludesRevokedApprovalState(t *testing.T) {
	t.Parallel()

	revokedAt := time.Date(2026, 3, 24, 12, 9, 0, 0, time.UTC)
	summary := BuildOperatorStatusSummary(JobRuntimeState{
		JobID:        "job-1",
		State:        JobStateRunning,
		ActiveStepID: "build",
		ApprovalRequests: []ApprovalRequest{
			{
				JobID:           "job-1",
				StepID:          "build",
				RequestedAction: ApprovalRequestedActionStepComplete,
				Scope:           ApprovalScopeOneJob,
				RequestedVia:    ApprovalRequestedViaRuntime,
				GrantedVia:      ApprovalGrantedViaOperatorCommand,
				SessionChannel:  "cli",
				SessionChatID:   "direct",
				State:           ApprovalStateRevoked,
			},
		},
		ApprovalGrants: []ApprovalGrant{
			{
				JobID:           "job-1",
				StepID:          "build",
				RequestedAction: ApprovalRequestedActionStepComplete,
				Scope:           ApprovalScopeOneJob,
				GrantedVia:      ApprovalGrantedViaOperatorCommand,
				SessionChannel:  "cli",
				SessionChatID:   "direct",
				State:           ApprovalStateRevoked,
				RevokedAt:       revokedAt,
			},
		},
	})

	if summary.ApprovalRequest == nil {
		t.Fatal("ApprovalRequest = nil, want populated summary")
	}
	if summary.ApprovalRequest.State != ApprovalStateRevoked {
		t.Fatalf("ApprovalRequest.State = %q, want %q", summary.ApprovalRequest.State, ApprovalStateRevoked)
	}
	if summary.ApprovalRequest.GrantedVia != ApprovalGrantedViaOperatorCommand {
		t.Fatalf("ApprovalRequest.GrantedVia = %q, want %q", summary.ApprovalRequest.GrantedVia, ApprovalGrantedViaOperatorCommand)
	}
	if len(summary.ApprovalHistory) != 1 {
		t.Fatalf("ApprovalHistory count = %d, want 1", len(summary.ApprovalHistory))
	}
	if summary.ApprovalHistory[0].RevokedAt == nil || *summary.ApprovalHistory[0].RevokedAt != "2026-03-24T12:09:00Z" {
		t.Fatalf("ApprovalHistory[0].RevokedAt = %#v, want RFC3339 time", summary.ApprovalHistory[0].RevokedAt)
	}
}

func TestBuildOperatorStatusSummaryIncludesBoundedDeterministicApprovalHistory(t *testing.T) {
	t.Parallel()

	requests := make([]ApprovalRequest, 0, OperatorStatusApprovalHistoryLimit+2)
	for i := 0; i < OperatorStatusApprovalHistoryLimit+2; i++ {
		requests = append(requests, ApprovalRequest{
			JobID:           "job-1",
			StepID:          "step-" + string(rune('a'+i)),
			RequestedAction: ApprovalRequestedActionStepComplete,
			Scope:           ApprovalScopeMissionStep,
			RequestedVia:    ApprovalRequestedViaRuntime,
			State:           ApprovalStatePending,
			RequestedAt:     time.Date(2026, 3, 24, 12, i, 0, 0, time.UTC),
		})
	}

	summary := BuildOperatorStatusSummary(JobRuntimeState{
		JobID:            "job-1",
		State:            JobStateWaitingUser,
		ApprovalRequests: requests,
	})

	if len(summary.ApprovalHistory) != OperatorStatusApprovalHistoryLimit {
		t.Fatalf("ApprovalHistory count = %d, want %d", len(summary.ApprovalHistory), OperatorStatusApprovalHistoryLimit)
	}
	for i, wantStep := range []string{"step-g", "step-f", "step-e", "step-d", "step-c"} {
		if summary.ApprovalHistory[i].StepID != wantStep {
			t.Fatalf("ApprovalHistory[%d].StepID = %q, want %q", i, summary.ApprovalHistory[i].StepID, wantStep)
		}
	}
}

func TestBuildOperatorStatusSummaryIncludesDeterministicRecentAuditSubset(t *testing.T) {
	t.Parallel()

	history := AppendAuditHistory(nil, AuditEvent{JobID: "job-1", StepID: "build", ToolName: "write_memory", Allowed: true, Timestamp: time.Date(2026, 3, 24, 12, 0, 0, 0, time.UTC)})
	history = AppendAuditHistory(history, AuditEvent{JobID: "job-1", StepID: "build", ToolName: "pause", Allowed: true, Timestamp: time.Date(2026, 3, 24, 12, 1, 0, 0, time.UTC)})
	history = AppendAuditHistory(history, AuditEvent{JobID: "job-1", StepID: "build", ToolName: "resume", Allowed: true, Timestamp: time.Date(2026, 3, 24, 12, 2, 0, 0, time.UTC)})
	history = AppendAuditHistory(history, AuditEvent{JobID: "job-1", StepID: "build", ToolName: "abort", Allowed: false, Code: RejectionCodeInvalidRuntimeState, Timestamp: time.Date(2026, 3, 24, 12, 3, 0, 0, time.UTC)})
	history = AppendAuditHistory(history, AuditEvent{JobID: "job-1", StepID: "final", ToolName: "status", Allowed: true, Timestamp: time.Date(2026, 3, 24, 12, 4, 0, 0, time.UTC)})
	history = AppendAuditHistory(history, AuditEvent{JobID: "job-1", StepID: "final", ToolName: "set_step", Allowed: true, Timestamp: time.Date(2026, 3, 24, 12, 5, 0, 0, time.UTC)})

	runtime := JobRuntimeState{
		JobID:        "job-1",
		State:        JobStatePaused,
		AuditHistory: history,
	}

	summary := BuildOperatorStatusSummary(runtime)
	if len(summary.RecentAudit) != OperatorStatusRecentAuditLimit {
		t.Fatalf("RecentAudit count = %d, want %d", len(summary.RecentAudit), OperatorStatusRecentAuditLimit)
	}

	for i, want := range []struct {
		action string
		stepID string
		class  AuditActionClass
		result AuditResult
		code   RejectionCode
		at     string
	}{
		{action: "set_step", stepID: "final", class: AuditActionClassOperatorCommand, result: AuditResultApplied, at: "2026-03-24T12:05:00Z"},
		{action: "status", stepID: "final", class: AuditActionClassOperatorCommand, result: AuditResultApplied, at: "2026-03-24T12:04:00Z"},
		{action: "abort", stepID: "build", class: AuditActionClassOperatorCommand, result: AuditResultRejected, code: RejectionCodeInvalidRuntimeState, at: "2026-03-24T12:03:00Z"},
		{action: "resume", stepID: "build", class: AuditActionClassOperatorCommand, result: AuditResultApplied, at: "2026-03-24T12:02:00Z"},
		{action: "pause", stepID: "build", class: AuditActionClassOperatorCommand, result: AuditResultApplied, at: "2026-03-24T12:01:00Z"},
	} {
		got := summary.RecentAudit[i]
		if got.EventID == "" {
			t.Fatalf("RecentAudit[%d].EventID = empty, want deterministic id", i)
		}
		if got.JobID != "job-1" {
			t.Fatalf("RecentAudit[%d].JobID = %q, want %q", i, got.JobID, "job-1")
		}
		if got.StepID != want.stepID {
			t.Fatalf("RecentAudit[%d].StepID = %q, want %q", i, got.StepID, want.stepID)
		}
		if got.Action != want.action {
			t.Fatalf("RecentAudit[%d].Action = %q, want %q", i, got.Action, want.action)
		}
		if got.ActionClass != want.class {
			t.Fatalf("RecentAudit[%d].ActionClass = %q, want %q", i, got.ActionClass, want.class)
		}
		if got.Result != want.result {
			t.Fatalf("RecentAudit[%d].Result = %q, want %q", i, got.Result, want.result)
		}
		if got.Code != want.code {
			t.Fatalf("RecentAudit[%d].Code = %q, want %q", i, got.Code, want.code)
		}
		if got.Timestamp != want.at {
			t.Fatalf("RecentAudit[%d].Timestamp = %q, want %q", i, got.Timestamp, want.at)
		}
	}
}

func TestFormatOperatorStatusSummaryWithAllowedToolsUsesSortedUniqueIntersection(t *testing.T) {
	t.Parallel()

	formatted, err := FormatOperatorStatusSummaryWithAllowedTools(
		JobRuntimeState{
			JobID:        "job-1",
			State:        JobStateRunning,
			ActiveStepID: "build",
		},
		EffectiveAllowedTools(
			&Job{AllowedTools: []string{"write", "read", "read", "shell"}},
			&Step{AllowedTools: []string{"shell", "read", "read", "missing"}},
		),
	)
	if err != nil {
		t.Fatalf("FormatOperatorStatusSummaryWithAllowedTools() error = %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal([]byte(formatted), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	allowedTools, ok := got["allowed_tools"].([]any)
	if !ok || len(allowedTools) != 2 {
		t.Fatalf("allowed_tools = %#v, want two entries", got["allowed_tools"])
	}
	if allowedTools[0] != "read" || allowedTools[1] != "shell" {
		t.Fatalf("allowed_tools = %#v, want [read shell]", allowedTools)
	}
}

func TestFormatOperatorStatusSummaryProducesDeterministicJSONForTerminalRuntime(t *testing.T) {
	t.Parallel()

	terminalAudit := AppendAuditHistory(nil, AuditEvent{
		JobID:     "job-1",
		StepID:    "build",
		ToolName:  "abort",
		Allowed:   true,
		Timestamp: time.Date(2026, 3, 24, 12, 8, 0, 0, time.UTC),
	})[0]

	formatted, err := FormatOperatorStatusSummary(JobRuntimeState{
		JobID:         "job-1",
		State:         JobStateAborted,
		AbortedReason: RuntimeAbortReasonOperatorCommand,
		AuditHistory:  []AuditEvent{terminalAudit},
		ApprovalRequests: []ApprovalRequest{
			{
				JobID:           "job-1",
				StepID:          "build",
				RequestedAction: ApprovalRequestedActionStepComplete,
				Scope:           ApprovalScopeMissionStep,
				State:           ApprovalStateSuperseded,
				RequestedAt:     time.Date(2026, 3, 24, 12, 6, 0, 0, time.UTC),
				ResolvedAt:      time.Date(2026, 3, 24, 12, 7, 0, 0, time.UTC),
				ExpiresAt:       time.Date(2026, 3, 24, 12, 9, 0, 0, time.UTC),
				SupersededAt:    time.Date(2026, 3, 24, 12, 7, 0, 0, time.UTC),
			},
		},
	})
	if err != nil {
		t.Fatalf("FormatOperatorStatusSummary() error = %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal([]byte(formatted), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if got["job_id"] != "job-1" {
		t.Fatalf("job_id = %#v, want %q", got["job_id"], "job-1")
	}
	if got["state"] != string(JobStateAborted) {
		t.Fatalf("state = %#v, want %q", got["state"], JobStateAborted)
	}
	if got["aborted_reason"] != RuntimeAbortReasonOperatorCommand {
		t.Fatalf("aborted_reason = %#v, want %q", got["aborted_reason"], RuntimeAbortReasonOperatorCommand)
	}

	request, ok := got["approval_request"].(map[string]any)
	if !ok {
		t.Fatalf("approval_request = %#v, want object", got["approval_request"])
	}
	if request["state"] != string(ApprovalStateSuperseded) {
		t.Fatalf("approval_request.state = %#v, want %q", request["state"], ApprovalStateSuperseded)
	}
	if request["superseded_at"] != "2026-03-24T12:07:00Z" {
		t.Fatalf("approval_request.superseded_at = %#v, want RFC3339 time", request["superseded_at"])
	}
	approvalHistory, ok := got["approval_history"].([]any)
	if !ok || len(approvalHistory) != 1 {
		t.Fatalf("approval_history = %#v, want one approval entry", got["approval_history"])
	}
	historyEntry, ok := approvalHistory[0].(map[string]any)
	if !ok {
		t.Fatalf("approval_history[0] = %#v, want object", approvalHistory[0])
	}
	if historyEntry["state"] != string(ApprovalStateSuperseded) {
		t.Fatalf("approval_history[0].state = %#v, want %q", historyEntry["state"], ApprovalStateSuperseded)
	}
	if historyEntry["requested_at"] != "2026-03-24T12:06:00Z" {
		t.Fatalf("approval_history[0].requested_at = %#v, want %q", historyEntry["requested_at"], "2026-03-24T12:06:00Z")
	}
	if historyEntry["resolved_at"] != "2026-03-24T12:07:00Z" {
		t.Fatalf("approval_history[0].resolved_at = %#v, want %q", historyEntry["resolved_at"], "2026-03-24T12:07:00Z")
	}
	if historyEntry["expires_at"] != "2026-03-24T12:09:00Z" {
		t.Fatalf("approval_history[0].expires_at = %#v, want %q", historyEntry["expires_at"], "2026-03-24T12:09:00Z")
	}

	recentAudit, ok := got["recent_audit"].([]any)
	if !ok || len(recentAudit) != 1 {
		t.Fatalf("recent_audit = %#v, want one audit entry", got["recent_audit"])
	}
	entry, ok := recentAudit[0].(map[string]any)
	if !ok {
		t.Fatalf("recent_audit[0] = %#v, want object", recentAudit[0])
	}
	if entry["job_id"] != "job-1" {
		t.Fatalf("recent_audit[0].job_id = %#v, want %q", entry["job_id"], "job-1")
	}
	if entry["event_id"] != terminalAudit.EventID {
		t.Fatalf("recent_audit[0].event_id = %#v, want %q", entry["event_id"], terminalAudit.EventID)
	}
	if entry["action"] != "abort" {
		t.Fatalf("recent_audit[0].action = %#v, want %q", entry["action"], "abort")
	}
	if entry["action_class"] != string(AuditActionClassOperatorCommand) {
		t.Fatalf("recent_audit[0].action_class = %#v, want %q", entry["action_class"], AuditActionClassOperatorCommand)
	}
	if entry["result"] != string(AuditResultApplied) {
		t.Fatalf("recent_audit[0].result = %#v, want %q", entry["result"], AuditResultApplied)
	}
	if entry["timestamp"] != "2026-03-24T12:08:00Z" {
		t.Fatalf("recent_audit[0].timestamp = %#v, want %q", entry["timestamp"], "2026-03-24T12:08:00Z")
	}
	if _, ok := got["allowed_tools"]; ok {
		t.Fatalf("allowed_tools = %#v, want omitted without active or persisted control context", got["allowed_tools"])
	}
}
