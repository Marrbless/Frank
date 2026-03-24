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

func TestFormatOperatorStatusSummaryProducesDeterministicJSONForTerminalRuntime(t *testing.T) {
	t.Parallel()

	formatted, err := FormatOperatorStatusSummary(JobRuntimeState{
		JobID:         "job-1",
		State:         JobStateAborted,
		AbortedReason: RuntimeAbortReasonOperatorCommand,
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
}
