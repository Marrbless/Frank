package tools

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/local/picobot/internal/missioncontrol"
)

func TestTaskStateOperatorStatusReturnsDeterministicSummaryForActiveRuntime(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	if err := state.ActivateStep(testTaskStateJob(), "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}

	summary, err := state.OperatorStatus("job-1")
	if err != nil {
		t.Fatalf("OperatorStatus() error = %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal([]byte(summary), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if got["job_id"] != "job-1" {
		t.Fatalf("job_id = %#v, want %q", got["job_id"], "job-1")
	}
	if got["state"] != string(missioncontrol.JobStateRunning) {
		t.Fatalf("state = %#v, want %q", got["state"], missioncontrol.JobStateRunning)
	}
	if got["active_step_id"] != "build" {
		t.Fatalf("active_step_id = %#v, want %q", got["active_step_id"], "build")
	}
	allowedTools, ok := got["allowed_tools"].([]any)
	if !ok || len(allowedTools) != 1 || allowedTools[0] != "read" {
		t.Fatalf("allowed_tools = %#v, want [%q]", got["allowed_tools"], "read")
	}

	runtime, ok := state.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if runtime.State != missioncontrol.JobStateRunning {
		t.Fatalf("MissionRuntimeState().State = %q, want unchanged %q", runtime.State, missioncontrol.JobStateRunning)
	}
}

func TestTaskStateOperatorStatusIncludesRecentAuditForActiveRuntime(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	if err := state.ActivateStep(testTaskStateJob(), "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}
	state.EmitAuditEvent(missioncontrol.AuditEvent{
		JobID:     "job-1",
		StepID:    "build",
		ToolName:  "write_memory",
		Allowed:   true,
		Timestamp: time.Date(2026, 3, 24, 12, 0, 0, 0, time.UTC),
	})
	state.EmitAuditEvent(missioncontrol.AuditEvent{
		JobID:     "job-1",
		StepID:    "build",
		ToolName:  "pause",
		Allowed:   false,
		Code:      missioncontrol.RejectionCodeInvalidRuntimeState,
		Timestamp: time.Date(2026, 3, 24, 12, 1, 0, 0, time.UTC),
	})
	expected := missioncontrol.AppendAuditHistory(nil, missioncontrol.AuditEvent{
		JobID:     "job-1",
		StepID:    "build",
		ToolName:  "pause",
		Allowed:   false,
		Code:      missioncontrol.RejectionCodeInvalidRuntimeState,
		Timestamp: time.Date(2026, 3, 24, 12, 1, 0, 0, time.UTC),
	})[0]

	summary, err := state.OperatorStatus("job-1")
	if err != nil {
		t.Fatalf("OperatorStatus() error = %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal([]byte(summary), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	recentAudit, ok := got["recent_audit"].([]any)
	if !ok || len(recentAudit) != 2 {
		t.Fatalf("recent_audit = %#v, want two audit entries", got["recent_audit"])
	}
	first, ok := recentAudit[0].(map[string]any)
	if !ok {
		t.Fatalf("recent_audit[0] = %#v, want object", recentAudit[0])
	}
	if first["action"] != "pause" {
		t.Fatalf("recent_audit[0].action = %#v, want %q", first["action"], "pause")
	}
	if first["event_id"] != expected.EventID {
		t.Fatalf("recent_audit[0].event_id = %#v, want %q", first["event_id"], expected.EventID)
	}
	if first["action_class"] != string(expected.ActionClass) {
		t.Fatalf("recent_audit[0].action_class = %#v, want %q", first["action_class"], expected.ActionClass)
	}
	if first["result"] != string(expected.Result) {
		t.Fatalf("recent_audit[0].result = %#v, want %q", first["result"], expected.Result)
	}
	if first["error_code"] != string(missioncontrol.RejectionCodeInvalidRuntimeState) {
		t.Fatalf("recent_audit[0].error_code = %#v, want %q", first["error_code"], missioncontrol.RejectionCodeInvalidRuntimeState)
	}
	if first["timestamp"] != "2026-03-24T12:01:00Z" {
		t.Fatalf("recent_audit[0].timestamp = %#v, want %q", first["timestamp"], "2026-03-24T12:01:00Z")
	}
}

func TestTaskStateOperatorStatusReturnsApprovalSummaryForPersistedWaitingRuntime(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	job := missioncontrol.Job{
		ID:           "job-1",
		MaxAuthority: missioncontrol.AuthorityTierHigh,
		AllowedTools: []string{"read"},
		Plan: missioncontrol.Plan{
			ID: "plan-1",
			Steps: []missioncontrol.Step{
				{
					ID:      "build",
					Type:    missioncontrol.StepTypeDiscussion,
					Subtype: missioncontrol.StepSubtypeAuthorization,
				},
				{
					ID:        "final",
					Type:      missioncontrol.StepTypeFinalResponse,
					DependsOn: []string{"build"},
				},
			},
		},
	}
	runtime := missioncontrol.JobRuntimeState{
		JobID:         "job-1",
		State:         missioncontrol.JobStateWaitingUser,
		ActiveStepID:  "build",
		WaitingReason: "discussion_authorization",
		ApprovalRequests: []missioncontrol.ApprovalRequest{
			{
				JobID:           "job-1",
				StepID:          "build",
				RequestedAction: missioncontrol.ApprovalRequestedActionStepComplete,
				Scope:           missioncontrol.ApprovalScopeMissionStep,
				State:           missioncontrol.ApprovalStatePending,
				ExpiresAt:       time.Date(2026, 3, 24, 12, 5, 0, 0, time.UTC),
				Content: &missioncontrol.ApprovalRequestContent{
					ProposedAction:   "Continue build.",
					WhyNeeded:        "Operator approval is required.",
					AuthorityTier:    missioncontrol.AuthorityTierMedium,
					FallbackIfDenied: "Stay waiting.",
				},
			},
		},
	}
	control, err := missioncontrol.BuildRuntimeControlContext(job, "build")
	if err != nil {
		t.Fatalf("BuildRuntimeControlContext() error = %v", err)
	}
	if err := state.HydrateRuntimeControl(job, runtime, &control); err != nil {
		t.Fatalf("HydrateRuntimeControl() error = %v", err)
	}
	state.ClearExecutionContext()

	summary, err := state.OperatorStatus("job-1")
	if err != nil {
		t.Fatalf("OperatorStatus() error = %v", err)
	}
	for _, want := range []string{
		`"state": "waiting_user"`,
		`"waiting_reason": "discussion_authorization"`,
		`"requested_action": "step_complete"`,
		`"scope": "mission_step"`,
		`"proposed_action": "Continue build."`,
		`"why_needed": "Operator approval is required."`,
		`"authority_tier": "medium"`,
		`"fallback_if_denied": "Stay waiting."`,
		`"expires_at": "2026-03-24T12:05:00Z"`,
	} {
		if !strings.Contains(summary, want) {
			t.Fatalf("OperatorStatus() = %q, want substring %q", summary, want)
		}
	}
}

func TestTaskStateOperatorStatusReturnsPausedReason(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	job := testTaskStateJob()
	runtime := missioncontrol.JobRuntimeState{
		JobID:        "job-1",
		State:        missioncontrol.JobStatePaused,
		ActiveStepID: "build",
		PausedReason: missioncontrol.RuntimePauseReasonOperatorCommand,
	}
	if err := state.HydrateRuntimeControl(job, runtime, nil); err != nil {
		t.Fatalf("HydrateRuntimeControl() error = %v", err)
	}
	state.ClearExecutionContext()

	summary, err := state.OperatorStatus("job-1")
	if err != nil {
		t.Fatalf("OperatorStatus() error = %v", err)
	}
	if !strings.Contains(summary, `"paused_reason": "operator_command"`) {
		t.Fatalf("OperatorStatus() = %q, want paused reason", summary)
	}
	if !strings.Contains(summary, `"allowed_tools": [`) || !strings.Contains(summary, `"read"`) {
		t.Fatalf("OperatorStatus() = %q, want effective allowed tools", summary)
	}
}

func TestTaskStateOperatorStatusReturnsRecentAuditForPersistedRuntime(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	job := testTaskStateJob()
	persistedHistory := missioncontrol.AppendAuditHistory(nil, missioncontrol.AuditEvent{
		JobID:     "job-1",
		StepID:    "build",
		ToolName:  "write_memory",
		Allowed:   true,
		Timestamp: time.Date(2026, 3, 24, 12, 0, 0, 0, time.UTC),
	})
	persistedHistory = missioncontrol.AppendAuditHistory(persistedHistory, missioncontrol.AuditEvent{
		JobID:     "job-1",
		StepID:    "build",
		ToolName:  "pause",
		Allowed:   true,
		Timestamp: time.Date(2026, 3, 24, 12, 1, 0, 0, time.UTC),
	})
	runtime := missioncontrol.JobRuntimeState{
		JobID:        "job-1",
		State:        missioncontrol.JobStatePaused,
		ActiveStepID: "build",
		PausedReason: missioncontrol.RuntimePauseReasonOperatorCommand,
		AuditHistory: persistedHistory,
	}
	if err := state.HydrateRuntimeControl(job, runtime, nil); err != nil {
		t.Fatalf("HydrateRuntimeControl() error = %v", err)
	}
	state.ClearExecutionContext()

	summary, err := state.OperatorStatus("job-1")
	if err != nil {
		t.Fatalf("OperatorStatus() error = %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal([]byte(summary), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	recentAudit, ok := got["recent_audit"].([]any)
	if !ok || len(recentAudit) != 2 {
		t.Fatalf("recent_audit = %#v, want two audit entries", got["recent_audit"])
	}
	first := recentAudit[0].(map[string]any)
	second := recentAudit[1].(map[string]any)
	if first["action"] != "pause" || second["action"] != "write_memory" {
		t.Fatalf("recent_audit actions = (%#v, %#v), want (%q, %q)", first["action"], second["action"], "pause", "write_memory")
	}
	if first["event_id"] != persistedHistory[1].EventID || second["event_id"] != persistedHistory[0].EventID {
		t.Fatalf("recent_audit event_ids = (%#v, %#v), want (%q, %q)", first["event_id"], second["event_id"], persistedHistory[1].EventID, persistedHistory[0].EventID)
	}
	if first["action_class"] != string(persistedHistory[1].ActionClass) || second["action_class"] != string(persistedHistory[0].ActionClass) {
		t.Fatalf("recent_audit action_class = (%#v, %#v), want (%q, %q)", first["action_class"], second["action_class"], persistedHistory[1].ActionClass, persistedHistory[0].ActionClass)
	}
	if first["result"] != string(persistedHistory[1].Result) || second["result"] != string(persistedHistory[0].Result) {
		t.Fatalf("recent_audit result = (%#v, %#v), want (%q, %q)", first["result"], second["result"], persistedHistory[1].Result, persistedHistory[0].Result)
	}
	allowedTools, ok := got["allowed_tools"].([]any)
	if !ok || len(allowedTools) != 1 || allowedTools[0] != "read" {
		t.Fatalf("allowed_tools = %#v, want [%q]", got["allowed_tools"], "read")
	}
}

func TestTaskStateOperatorStatusReportsTerminalRuntimeDeterministically(t *testing.T) {
	t.Parallel()

	for _, runtime := range []missioncontrol.JobRuntimeState{
		{
			JobID:         "job-1",
			State:         missioncontrol.JobStateAborted,
			AbortedReason: missioncontrol.RuntimeAbortReasonOperatorCommand,
			AuditHistory: []missioncontrol.AuditEvent{
				{
					JobID:     "job-1",
					StepID:    "build",
					ToolName:  "abort",
					Allowed:   true,
					Timestamp: time.Date(2026, 3, 24, 12, 0, 0, 0, time.UTC),
				},
			},
		},
		{
			JobID: "job-1",
			State: missioncontrol.JobStateCompleted,
			AuditHistory: []missioncontrol.AuditEvent{
				{
					JobID:     "job-1",
					StepID:    "final",
					ToolName:  "status",
					Allowed:   true,
					Timestamp: time.Date(2026, 3, 24, 12, 1, 0, 0, time.UTC),
				},
			},
		},
		{
			JobID: "job-1",
			State: missioncontrol.JobStateFailed,
			AuditHistory: []missioncontrol.AuditEvent{
				{
					JobID:     "job-1",
					StepID:    "build",
					ToolName:  "pause",
					Allowed:   false,
					Code:      missioncontrol.RejectionCodeInvalidRuntimeState,
					Timestamp: time.Date(2026, 3, 24, 12, 2, 0, 0, time.UTC),
				},
			},
		},
	} {
		runtime := runtime
		t.Run(string(runtime.State), func(t *testing.T) {
			state := NewTaskState()
			if err := state.HydrateRuntimeControl(testTaskStateJob(), runtime, nil); err != nil {
				t.Fatalf("HydrateRuntimeControl() error = %v", err)
			}

			summary, err := state.OperatorStatus("job-1")
			if err != nil {
				t.Fatalf("OperatorStatus() error = %v", err)
			}
			if !strings.Contains(summary, `"state": "`+string(runtime.State)+`"`) {
				t.Fatalf("OperatorStatus() = %q, want state %q", summary, runtime.State)
			}
			if !strings.Contains(summary, `"recent_audit": [`) {
				t.Fatalf("OperatorStatus() = %q, want recent audit block", summary)
			}
			if strings.Contains(summary, `"allowed_tools":`) {
				t.Fatalf("OperatorStatus() = %q, want allowed_tools omitted without control context", summary)
			}
		})
	}
}

func TestTaskStateOperatorStatusWrongJobDoesNotBind(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	if err := state.ActivateStep(testTaskStateJob(), "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}

	_, err := state.OperatorStatus("other-job")
	if err == nil {
		t.Fatal("OperatorStatus(other-job) error = nil, want mismatch failure")
	}
	if !strings.Contains(err.Error(), "does not match the active job") {
		t.Fatalf("OperatorStatus(other-job) error = %q, want job mismatch", err)
	}
}
