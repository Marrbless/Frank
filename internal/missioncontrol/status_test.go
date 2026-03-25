package missioncontrol

import (
	"encoding/json"
	"testing"
	"time"
)

func TestBuildOperatorStatusSummaryIncludesLatestApprovalForActiveStep(t *testing.T) {
	t.Parallel()

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
				State:           ApprovalStatePending,
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
}

func TestBuildOperatorStatusSummaryIncludesDeterministicRecentAuditSubset(t *testing.T) {
	t.Parallel()

	runtime := JobRuntimeState{
		JobID: "job-1",
		State: JobStatePaused,
		AuditHistory: []AuditEvent{
			{JobID: "job-1", StepID: "build", ToolName: "write_memory", Allowed: true, Timestamp: time.Date(2026, 3, 24, 12, 0, 0, 0, time.UTC)},
			{JobID: "job-1", StepID: "build", ToolName: "pause", Allowed: true, Timestamp: time.Date(2026, 3, 24, 12, 1, 0, 0, time.UTC)},
			{JobID: "job-1", StepID: "build", ToolName: "resume", Allowed: true, Timestamp: time.Date(2026, 3, 24, 12, 2, 0, 0, time.UTC)},
			{JobID: "job-1", StepID: "build", ToolName: "abort", Allowed: false, Code: RejectionCodeInvalidRuntimeState, Timestamp: time.Date(2026, 3, 24, 12, 3, 0, 0, time.UTC)},
			{JobID: "job-1", StepID: "final", ToolName: "status", Allowed: true, Timestamp: time.Date(2026, 3, 24, 12, 4, 0, 0, time.UTC)},
			{JobID: "job-1", StepID: "final", ToolName: "set_step", Allowed: true, Timestamp: time.Date(2026, 3, 24, 12, 5, 0, 0, time.UTC)},
		},
	}

	summary := BuildOperatorStatusSummary(runtime)
	if len(summary.RecentAudit) != OperatorStatusRecentAuditLimit {
		t.Fatalf("RecentAudit count = %d, want %d", len(summary.RecentAudit), OperatorStatusRecentAuditLimit)
	}

	for i, want := range []struct {
		action string
		stepID string
		code   RejectionCode
		at     string
	}{
		{action: "set_step", stepID: "final", at: "2026-03-24T12:05:00Z"},
		{action: "status", stepID: "final", at: "2026-03-24T12:04:00Z"},
		{action: "abort", stepID: "build", code: RejectionCodeInvalidRuntimeState, at: "2026-03-24T12:03:00Z"},
		{action: "resume", stepID: "build", at: "2026-03-24T12:02:00Z"},
		{action: "pause", stepID: "build", at: "2026-03-24T12:01:00Z"},
	} {
		got := summary.RecentAudit[i]
		if got.JobID != "job-1" {
			t.Fatalf("RecentAudit[%d].JobID = %q, want %q", i, got.JobID, "job-1")
		}
		if got.StepID != want.stepID {
			t.Fatalf("RecentAudit[%d].StepID = %q, want %q", i, got.StepID, want.stepID)
		}
		if got.Action != want.action {
			t.Fatalf("RecentAudit[%d].Action = %q, want %q", i, got.Action, want.action)
		}
		if got.Code != want.code {
			t.Fatalf("RecentAudit[%d].Code = %q, want %q", i, got.Code, want.code)
		}
		if got.Timestamp != want.at {
			t.Fatalf("RecentAudit[%d].Timestamp = %q, want %q", i, got.Timestamp, want.at)
		}
	}
}

func TestFormatOperatorStatusSummaryProducesDeterministicJSONForTerminalRuntime(t *testing.T) {
	t.Parallel()

	formatted, err := FormatOperatorStatusSummary(JobRuntimeState{
		JobID:         "job-1",
		State:         JobStateAborted,
		AbortedReason: RuntimeAbortReasonOperatorCommand,
		AuditHistory: []AuditEvent{
			{
				JobID:     "job-1",
				StepID:    "build",
				ToolName:  "abort",
				Allowed:   true,
				Timestamp: time.Date(2026, 3, 24, 12, 8, 0, 0, time.UTC),
			},
		},
		ApprovalRequests: []ApprovalRequest{
			{
				JobID:           "job-1",
				StepID:          "build",
				RequestedAction: ApprovalRequestedActionStepComplete,
				Scope:           ApprovalScopeMissionStep,
				State:           ApprovalStateSuperseded,
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
	if entry["action"] != "abort" {
		t.Fatalf("recent_audit[0].action = %#v, want %q", entry["action"], "abort")
	}
	if entry["timestamp"] != "2026-03-24T12:08:00Z" {
		t.Fatalf("recent_audit[0].timestamp = %#v, want %q", entry["timestamp"], "2026-03-24T12:08:00Z")
	}
}
