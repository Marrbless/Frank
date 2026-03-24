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

	runtime, ok := state.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if runtime.State != missioncontrol.JobStateRunning {
		t.Fatalf("MissionRuntimeState().State = %q, want unchanged %q", runtime.State, missioncontrol.JobStateRunning)
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
}

func TestTaskStateOperatorStatusReportsTerminalRuntimeDeterministically(t *testing.T) {
	t.Parallel()

	for _, runtime := range []missioncontrol.JobRuntimeState{
		{JobID: "job-1", State: missioncontrol.JobStateAborted, AbortedReason: missioncontrol.RuntimeAbortReasonOperatorCommand},
		{JobID: "job-1", State: missioncontrol.JobStateCompleted},
		{JobID: "job-1", State: missioncontrol.JobStateFailed},
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
