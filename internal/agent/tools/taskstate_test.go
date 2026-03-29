package tools

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/local/picobot/internal/missioncontrol"
)

func TestTaskStateActivateStepStoresValidExecutionContext(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	job := testTaskStateJob()

	if err := state.ActivateStep(job, "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}

	ec, ok := state.ExecutionContext()
	if !ok {
		t.Fatal("ExecutionContext() ok = false, want true")
	}

	if ec.Job == nil {
		t.Fatal("ExecutionContext().Job = nil, want non-nil")
	}

	if ec.Step == nil {
		t.Fatal("ExecutionContext().Step = nil, want non-nil")
	}

	if ec.Job.ID != job.ID {
		t.Fatalf("ExecutionContext().Job.ID = %q, want %q", ec.Job.ID, job.ID)
	}

	if ec.Step.ID != "build" {
		t.Fatalf("ExecutionContext().Step.ID = %q, want %q", ec.Step.ID, "build")
	}
	if ec.Runtime == nil {
		t.Fatal("ExecutionContext().Runtime = nil, want non-nil")
	}
	if ec.Runtime.State != missioncontrol.JobStateRunning {
		t.Fatalf("ExecutionContext().Runtime.State = %q, want %q", ec.Runtime.State, missioncontrol.JobStateRunning)
	}
	if ec.Runtime.ActiveStepID != "build" {
		t.Fatalf("ExecutionContext().Runtime.ActiveStepID = %q, want %q", ec.Runtime.ActiveStepID, "build")
	}
}

func TestTaskStateActivateStepInvalidPlanDoesNotOverwriteExistingContext(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	original := missioncontrol.ExecutionContext{
		Job:  &missioncontrol.Job{ID: "existing-job"},
		Step: &missioncontrol.Step{ID: "existing-step"},
	}
	state.SetExecutionContext(original)

	err := state.ActivateStep(missioncontrol.Job{
		ID:           "job-1",
		MaxAuthority: missioncontrol.AuthorityTierHigh,
		AllowedTools: []string{"read"},
		Plan:         missioncontrol.Plan{ID: "plan-1"},
	}, "build")
	if err == nil {
		t.Fatal("ActivateStep() error = nil, want validation error")
	}

	got, ok := state.ExecutionContext()
	if !ok {
		t.Fatal("ExecutionContext() ok = false, want true")
	}

	if !reflect.DeepEqual(got, original) {
		t.Fatalf("ExecutionContext() = %#v, want original %#v", got, original)
	}
}

func TestTaskStateActivateStepUnknownStepDoesNotOverwriteExistingContext(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	original := missioncontrol.ExecutionContext{
		Job:  &missioncontrol.Job{ID: "existing-job"},
		Step: &missioncontrol.Step{ID: "existing-step"},
	}
	state.SetExecutionContext(original)

	err := state.ActivateStep(testTaskStateJob(), "missing")
	if err == nil {
		t.Fatal("ActivateStep() error = nil, want unknown step error")
	}

	validationErr, ok := err.(missioncontrol.ValidationError)
	if !ok {
		t.Fatalf("ActivateStep() error type = %T, want ValidationError", err)
	}

	if validationErr.Code != missioncontrol.RejectionCodeUnknownStep {
		t.Fatalf("ValidationError.Code = %q, want %q", validationErr.Code, missioncontrol.RejectionCodeUnknownStep)
	}

	got, ok := state.ExecutionContext()
	if !ok {
		t.Fatal("ExecutionContext() ok = false, want true")
	}

	if !reflect.DeepEqual(got, original) {
		t.Fatalf("ExecutionContext() = %#v, want original %#v", got, original)
	}
}

func TestTaskStateExecutionContextReturnsIndependentSnapshot(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	if err := state.ActivateStep(testTaskStateJob(), "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}

	ec, ok := state.ExecutionContext()
	if !ok {
		t.Fatal("ExecutionContext() ok = false, want true")
	}

	ec.Job.AllowedTools[0] = "mutated-job-tool"
	ec.Job.Plan.Steps[0].AllowedTools[0] = "mutated-plan-step-tool"
	ec.Step.AllowedTools[0] = "mutated-step-tool"
	ec.Runtime.ActiveStepID = "mutated-runtime-step"

	stored, ok := state.ExecutionContext()
	if !ok {
		t.Fatal("ExecutionContext() ok = false, want true")
	}

	if stored.Job.AllowedTools[0] != "read" {
		t.Fatalf("stored Job.AllowedTools[0] = %q, want %q", stored.Job.AllowedTools[0], "read")
	}

	if stored.Job.Plan.Steps[0].AllowedTools[0] != "read" {
		t.Fatalf("stored Job.Plan.Steps[0].AllowedTools[0] = %q, want %q", stored.Job.Plan.Steps[0].AllowedTools[0], "read")
	}

	if stored.Step.AllowedTools[0] != "read" {
		t.Fatalf("stored Step.AllowedTools[0] = %q, want %q", stored.Step.AllowedTools[0], "read")
	}
	if stored.Runtime == nil {
		t.Fatal("stored Runtime = nil, want non-nil")
	}
	if stored.Runtime.ActiveStepID != "build" {
		t.Fatalf("stored Runtime.ActiveStepID = %q, want %q", stored.Runtime.ActiveStepID, "build")
	}
}

func TestTaskStateClearExecutionContextPreservesDurableRuntimeState(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	if err := state.ActivateStep(testTaskStateJob(), "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}

	state.ClearExecutionContext()

	got, ok := state.ExecutionContext()
	if ok {
		t.Fatalf("ExecutionContext() ok = true, want false with context %#v", got)
	}

	if !reflect.DeepEqual(got, missioncontrol.ExecutionContext{}) {
		t.Fatalf("ExecutionContext() = %#v, want zero value", got)
	}
	runtime, ok := state.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want durable runtime after clear")
	}
	if runtime.State != missioncontrol.JobStateRunning {
		t.Fatalf("MissionRuntimeState().State = %q, want %q", runtime.State, missioncontrol.JobStateRunning)
	}
	if runtime.ActiveStepID != "build" {
		t.Fatalf("MissionRuntimeState().ActiveStepID = %q, want %q", runtime.ActiveStepID, "build")
	}
}

func TestTaskStateResumeRuntimeStoresExecutionContext(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	job := testTaskStateJob()
	runtimeState := missioncontrol.JobRuntimeState{
		JobID:        job.ID,
		State:        missioncontrol.JobStateRunning,
		ActiveStepID: "build",
	}

	if err := state.ResumeRuntime(job, runtimeState, nil, true); err != nil {
		t.Fatalf("ResumeRuntime() error = %v", err)
	}

	ec, ok := state.ExecutionContext()
	if !ok {
		t.Fatal("ExecutionContext() ok = false, want true")
	}
	if ec.Runtime == nil {
		t.Fatal("ExecutionContext().Runtime = nil, want non-nil")
	}
	if ec.Runtime.ActiveStepID != "build" {
		t.Fatalf("ExecutionContext().Runtime.ActiveStepID = %q, want %q", ec.Runtime.ActiveStepID, "build")
	}
}

func TestTaskStateResumeRuntimeRejectsCompletedActiveStepReplay(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	job := testTaskStateJob()
	runtimeState := missioncontrol.JobRuntimeState{
		JobID:        job.ID,
		State:        missioncontrol.JobStatePaused,
		ActiveStepID: "build",
		CompletedSteps: []missioncontrol.RuntimeStepRecord{
			{StepID: "build", At: time.Date(2026, 3, 27, 12, 0, 0, 0, time.UTC)},
		},
	}

	err := state.ResumeRuntime(job, runtimeState, nil, true)
	if err == nil {
		t.Fatal("ResumeRuntime() error = nil, want completed-step replay rejection")
	}

	validationErr, ok := err.(missioncontrol.ValidationError)
	if !ok {
		t.Fatalf("ResumeRuntime() error type = %T, want ValidationError", err)
	}
	if validationErr.Code != missioncontrol.RejectionCodeInvalidRuntimeState {
		t.Fatalf("ValidationError.Code = %q, want %q", validationErr.Code, missioncontrol.RejectionCodeInvalidRuntimeState)
	}
	if _, ok := state.ExecutionContext(); ok {
		t.Fatal("ExecutionContext() ok = true, want false after rejected replay resume")
	}
}

func TestTaskStateActivateStepRejectsPreviouslyCompletedStepReplay(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	job := testTaskStateJob()
	if err := state.HydrateRuntimeControl(job, missioncontrol.JobRuntimeState{
		JobID:        job.ID,
		State:        missioncontrol.JobStatePaused,
		ActiveStepID: "final",
		PausedReason: missioncontrol.RuntimePauseReasonOperatorCommand,
		CompletedSteps: []missioncontrol.RuntimeStepRecord{
			{StepID: "build", At: time.Date(2026, 3, 27, 12, 0, 0, 0, time.UTC)},
		},
	}, nil); err != nil {
		t.Fatalf("HydrateRuntimeControl() setup error = %v", err)
	}

	err := state.ActivateStep(job, "build")
	if err == nil {
		t.Fatal("ActivateStep() error = nil, want completed-step replay rejection")
	}

	validationErr, ok := err.(missioncontrol.ValidationError)
	if !ok {
		t.Fatalf("ActivateStep() error type = %T, want ValidationError", err)
	}
	if validationErr.Code != missioncontrol.RejectionCodeInvalidRuntimeState {
		t.Fatalf("ValidationError.Code = %q, want %q", validationErr.Code, missioncontrol.RejectionCodeInvalidRuntimeState)
	}
}

func TestTaskStateActivateStepRejectsPreviouslyFailedStepReplay(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	job := testTaskStateJob()
	if err := state.HydrateRuntimeControl(job, missioncontrol.JobRuntimeState{
		JobID:        job.ID,
		State:        missioncontrol.JobStatePaused,
		ActiveStepID: "final",
		PausedReason: missioncontrol.RuntimePauseReasonOperatorCommand,
		FailedSteps: []missioncontrol.RuntimeStepRecord{
			{StepID: "build", Reason: "validator failed", At: time.Date(2026, 3, 27, 12, 0, 0, 0, time.UTC)},
		},
	}, nil); err != nil {
		t.Fatalf("HydrateRuntimeControl() setup error = %v", err)
	}

	err := state.ActivateStep(job, "build")
	if err == nil {
		t.Fatal("ActivateStep() error = nil, want failed-step replay rejection")
	}

	validationErr, ok := err.(missioncontrol.ValidationError)
	if !ok {
		t.Fatalf("ActivateStep() error type = %T, want ValidationError", err)
	}
	if validationErr.Code != missioncontrol.RejectionCodeInvalidRuntimeState {
		t.Fatalf("ValidationError.Code = %q, want %q", validationErr.Code, missioncontrol.RejectionCodeInvalidRuntimeState)
	}
}

func TestTaskStateApplyStepOutputPausesCompletedOneShotCodeStep(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	if err := state.ActivateStep(testTaskStateJob(), "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}

	if err := state.ApplyStepOutput("Implemented the change.", []missioncontrol.RuntimeToolCallEvidence{
		{ToolName: "filesystem", Arguments: map[string]interface{}{"action": "write", "path": "result.txt"}},
		{ToolName: "filesystem", Arguments: map[string]interface{}{"action": "stat", "path": "result.txt"}},
	}); err != nil {
		t.Fatalf("ApplyStepOutput() error = %v", err)
	}

	if _, ok := state.ExecutionContext(); ok {
		t.Fatal("ExecutionContext() ok = true, want false after completed step pause")
	}

	runtime, ok := state.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if runtime.State != missioncontrol.JobStatePaused {
		t.Fatalf("MissionRuntimeState().State = %q, want %q", runtime.State, missioncontrol.JobStatePaused)
	}
	if len(runtime.CompletedSteps) != 1 || runtime.CompletedSteps[0].StepID != "build" {
		t.Fatalf("MissionRuntimeState().CompletedSteps = %#v, want build completion", runtime.CompletedSteps)
	}
}

func TestTaskStateApplyStepOutputPausesForUnattendedWallClockBudget(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	job := testTaskStateJob()
	now := time.Now().UTC()
	if err := state.ActivateStep(job, "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}

	ec, ok := state.ExecutionContext()
	if !ok || ec.Runtime == nil {
		t.Fatalf("ExecutionContext() = (%#v, %t), want active runtime", ec, ok)
	}
	ec.Runtime.CreatedAt = now.Add(-5 * time.Hour)
	ec.Runtime.UpdatedAt = now.Add(-2 * time.Minute)
	ec.Runtime.StartedAt = now.Add(-5 * time.Hour)
	ec.Runtime.ActiveStepAt = now.Add(-2 * time.Minute)
	state.SetExecutionContext(ec)

	if err := state.ApplyStepOutput("Implemented the change.", []missioncontrol.RuntimeToolCallEvidence{
		{ToolName: "filesystem", Arguments: map[string]interface{}{"action": "write", "path": "result.txt"}},
	}); err != nil {
		t.Fatalf("ApplyStepOutput() error = %v", err)
	}

	ec, ok = state.ExecutionContext()
	if !ok {
		t.Fatal("ExecutionContext() ok = false, want paused execution context")
	}
	if ec.Runtime == nil || ec.Runtime.State != missioncontrol.JobStatePaused {
		t.Fatalf("ExecutionContext().Runtime = %#v, want paused runtime", ec.Runtime)
	}

	runtime, ok := state.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if runtime.PausedReason != missioncontrol.RuntimePauseReasonBudgetExhausted {
		t.Fatalf("MissionRuntimeState().PausedReason = %q, want %q", runtime.PausedReason, missioncontrol.RuntimePauseReasonBudgetExhausted)
	}
	if runtime.BudgetBlocker == nil {
		t.Fatal("MissionRuntimeState().BudgetBlocker = nil, want blocker")
	}
	if runtime.BudgetBlocker.Ceiling != "unattended_wall_clock" {
		t.Fatalf("MissionRuntimeState().BudgetBlocker.Ceiling = %q, want %q", runtime.BudgetBlocker.Ceiling, "unattended_wall_clock")
	}
	if len(runtime.CompletedSteps) != 0 {
		t.Fatalf("MissionRuntimeState().CompletedSteps = %#v, want empty after budget pause", runtime.CompletedSteps)
	}
	if len(runtime.AuditHistory) != 1 || runtime.AuditHistory[0].ToolName != "budget_exhausted" {
		t.Fatalf("MissionRuntimeState().AuditHistory = %#v, want one budget_exhausted event", runtime.AuditHistory)
	}
}

func TestTaskStateEnforceUnattendedWallClockBudgetPausesRunningExecution(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	job := testTaskStateJob()
	now := time.Now().UTC()
	if err := state.ActivateStep(job, "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}

	ec, ok := state.ExecutionContext()
	if !ok || ec.Runtime == nil {
		t.Fatalf("ExecutionContext() = (%#v, %t), want active runtime", ec, ok)
	}
	ec.Runtime.CreatedAt = now.Add(-5 * time.Hour)
	ec.Runtime.UpdatedAt = now.Add(-1 * time.Minute)
	ec.Runtime.StartedAt = now.Add(-5 * time.Hour)
	ec.Runtime.ActiveStepAt = now.Add(-1 * time.Minute)
	state.SetExecutionContext(ec)

	exhausted, err := state.EnforceUnattendedWallClockBudget()
	if err != nil {
		t.Fatalf("EnforceUnattendedWallClockBudget() error = %v", err)
	}
	if !exhausted {
		t.Fatal("EnforceUnattendedWallClockBudget() exhausted = false, want true")
	}

	runtime, ok := state.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if runtime.State != missioncontrol.JobStatePaused {
		t.Fatalf("MissionRuntimeState().State = %q, want %q", runtime.State, missioncontrol.JobStatePaused)
	}
	if runtime.BudgetBlocker == nil || runtime.BudgetBlocker.Ceiling != "unattended_wall_clock" {
		t.Fatalf("MissionRuntimeState().BudgetBlocker = %#v, want unattended wall-clock blocker", runtime.BudgetBlocker)
	}
}

func TestTaskStateRecordFailedToolActionPausesAtBudget(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	job := testTaskStateJob()
	if err := state.ActivateStep(job, "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}

	var exhausted bool
	var err error
	for i := 0; i < 5; i++ {
		exhausted, err = state.RecordFailedToolAction("message", "message tool: 'content' argument required")
		if err != nil {
			t.Fatalf("RecordFailedToolAction() step %d error = %v", i, err)
		}
	}

	if !exhausted {
		t.Fatal("RecordFailedToolAction() exhausted = false, want true on threshold")
	}

	runtime, ok := state.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if runtime.State != missioncontrol.JobStatePaused {
		t.Fatalf("MissionRuntimeState().State = %q, want %q", runtime.State, missioncontrol.JobStatePaused)
	}
	if runtime.BudgetBlocker == nil || runtime.BudgetBlocker.Ceiling != "failed_actions" {
		t.Fatalf("MissionRuntimeState().BudgetBlocker = %#v, want failed_actions blocker", runtime.BudgetBlocker)
	}
	if len(runtime.AuditHistory) != 6 {
		t.Fatalf("MissionRuntimeState().AuditHistory count = %d, want 6", len(runtime.AuditHistory))
	}
}

func TestTaskStateApplyStepOutputPausesCompletedStaticArtifactStep(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	job := testTaskStateJob()
	job.AllowedTools = []string{"filesystem", "read"}
	job.Plan.Steps[0] = missioncontrol.Step{
		ID:              "build",
		Type:            missioncontrol.StepTypeStaticArtifact,
		AllowedTools:    []string{"filesystem"},
		SuccessCriteria: []string{"Write `report.json` as valid JSON."},
	}

	if err := state.ActivateStep(job, "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}

	if err := state.ApplyStepOutput("Created report.json.", []missioncontrol.RuntimeToolCallEvidence{
		{ToolName: "filesystem", Arguments: map[string]interface{}{"action": "write", "path": "report.json"}, Result: "written"},
		{ToolName: "filesystem", Arguments: map[string]interface{}{"action": "stat", "path": "report.json"}, Result: "exists=true\nkind=file\nname=report.json\nsize=17\n"},
		{ToolName: "filesystem", Arguments: map[string]interface{}{"action": "read", "path": "report.json"}, Result: "{\n  \"ok\": true\n}\n"},
	}); err != nil {
		t.Fatalf("ApplyStepOutput() error = %v", err)
	}

	if _, ok := state.ExecutionContext(); ok {
		t.Fatal("ExecutionContext() ok = true, want false after completed static_artifact pause")
	}

	runtime, ok := state.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if runtime.State != missioncontrol.JobStatePaused {
		t.Fatalf("MissionRuntimeState().State = %q, want %q", runtime.State, missioncontrol.JobStatePaused)
	}
	if len(runtime.CompletedSteps) != 1 || runtime.CompletedSteps[0].StepID != "build" {
		t.Fatalf("MissionRuntimeState().CompletedSteps = %#v, want build completion", runtime.CompletedSteps)
	}
}

func TestTaskStateApplyStepOutputTransitionsDiscussionSubtypeToWaitingUser(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	job := testTaskStateJob()
	job.Plan.Steps[0] = missioncontrol.Step{
		ID:      "build",
		Type:    missioncontrol.StepTypeDiscussion,
		Subtype: missioncontrol.StepSubtypeAuthorization,
	}
	job.Plan.Steps[1] = missioncontrol.Step{
		ID:        "final",
		Type:      missioncontrol.StepTypeFinalResponse,
		DependsOn: []string{"build"},
	}

	if err := state.ActivateStep(job, "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}

	if err := state.ApplyStepOutput("Need approval before continuing.", nil); err != nil {
		t.Fatalf("ApplyStepOutput() error = %v", err)
	}

	ec, ok := state.ExecutionContext()
	if !ok {
		t.Fatal("ExecutionContext() ok = false, want true")
	}
	if ec.Runtime == nil {
		t.Fatal("ExecutionContext().Runtime = nil, want non-nil")
	}
	if ec.Runtime.State != missioncontrol.JobStateWaitingUser {
		t.Fatalf("ExecutionContext().Runtime.State = %q, want %q", ec.Runtime.State, missioncontrol.JobStateWaitingUser)
	}
}

func TestTaskStatePauseRuntimePausesActiveStepWithoutCompletion(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	if err := state.ActivateStep(testTaskStateJob(), "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}

	if err := state.PauseRuntime("job-1"); err != nil {
		t.Fatalf("PauseRuntime() error = %v", err)
	}

	ec, ok := state.ExecutionContext()
	if !ok {
		t.Fatal("ExecutionContext() ok = false, want paused execution context")
	}
	if ec.Runtime == nil {
		t.Fatal("ExecutionContext().Runtime = nil, want non-nil")
	}
	if ec.Runtime.State != missioncontrol.JobStatePaused {
		t.Fatalf("ExecutionContext().Runtime.State = %q, want %q", ec.Runtime.State, missioncontrol.JobStatePaused)
	}
	if ec.Runtime.ActiveStepID != "build" {
		t.Fatalf("ExecutionContext().Runtime.ActiveStepID = %q, want %q", ec.Runtime.ActiveStepID, "build")
	}
	if len(ec.Runtime.CompletedSteps) != 0 {
		t.Fatalf("ExecutionContext().Runtime.CompletedSteps = %#v, want empty", ec.Runtime.CompletedSteps)
	}
}

func TestTaskStatePauseRuntimeRequiresActiveExecutionContextAfterTeardown(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	if err := state.ActivateStep(testTaskStateJob(), "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}

	state.ClearExecutionContext()

	err := state.PauseRuntime("job-1")
	if err == nil {
		t.Fatal("PauseRuntime() error = nil, want active-step failure")
	}

	validationErr, ok := err.(missioncontrol.ValidationError)
	if !ok {
		t.Fatalf("PauseRuntime() error type = %T, want ValidationError", err)
	}
	if validationErr.Code != missioncontrol.RejectionCodeInvalidRuntimeState {
		t.Fatalf("ValidationError.Code = %q, want %q", validationErr.Code, missioncontrol.RejectionCodeInvalidRuntimeState)
	}

	runtime, ok := state.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want preserved runtime")
	}
	if runtime.State != missioncontrol.JobStateRunning {
		t.Fatalf("MissionRuntimeState().State = %q, want %q", runtime.State, missioncontrol.JobStateRunning)
	}
	if runtime.ActiveStepID != "build" {
		t.Fatalf("MissionRuntimeState().ActiveStepID = %q, want %q", runtime.ActiveStepID, "build")
	}
}

func TestTaskStateResumeRuntimeControlRequiresPausedState(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	if err := state.ActivateStep(testTaskStateJob(), "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}

	err := state.ResumeRuntimeControl("job-1")
	if err == nil {
		t.Fatal("ResumeRuntimeControl() error = nil, want paused-state failure")
	}

	validationErr, ok := err.(missioncontrol.ValidationError)
	if !ok {
		t.Fatalf("ResumeRuntimeControl() error type = %T, want ValidationError", err)
	}
	if validationErr.Code != missioncontrol.RejectionCodeInvalidRuntimeState {
		t.Fatalf("ValidationError.Code = %q, want %q", validationErr.Code, missioncontrol.RejectionCodeInvalidRuntimeState)
	}
}

func TestTaskStateHydrateRuntimeControlResumesPausedRuntimeAfterRehydration(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	job := testTaskStateJob()
	runtime := missioncontrol.JobRuntimeState{
		JobID:        job.ID,
		State:        missioncontrol.JobStatePaused,
		ActiveStepID: "build",
		PausedReason: missioncontrol.RuntimePauseReasonOperatorCommand,
	}

	if err := state.HydrateRuntimeControl(job, runtime, nil); err != nil {
		t.Fatalf("HydrateRuntimeControl() error = %v", err)
	}
	if _, ok := state.ExecutionContext(); ok {
		t.Fatal("ExecutionContext() ok = true, want false after rehydration")
	}
	if err := state.ResumeRuntimeControl("job-1"); err != nil {
		t.Fatalf("ResumeRuntimeControl() error = %v", err)
	}

	ec, ok := state.ExecutionContext()
	if !ok {
		t.Fatal("ExecutionContext() ok = false, want restored context")
	}
	if ec.Runtime == nil || ec.Runtime.State != missioncontrol.JobStateRunning {
		t.Fatalf("ExecutionContext().Runtime = %#v, want running runtime", ec.Runtime)
	}
	if ec.Step == nil || ec.Step.ID != "build" {
		t.Fatalf("ExecutionContext().Step = %#v, want build", ec.Step)
	}
}

func TestTaskStateResumeRuntimeControlRejectsCompletedActiveStepReplayAfterRehydration(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	job := testTaskStateJob()
	runtime := missioncontrol.JobRuntimeState{
		JobID:        job.ID,
		State:        missioncontrol.JobStatePaused,
		ActiveStepID: "build",
		PausedReason: missioncontrol.RuntimePauseReasonOperatorCommand,
		CompletedSteps: []missioncontrol.RuntimeStepRecord{
			{StepID: "build", At: time.Date(2026, 3, 27, 12, 0, 0, 0, time.UTC)},
		},
	}

	if err := state.HydrateRuntimeControl(job, runtime, nil); err != nil {
		t.Fatalf("HydrateRuntimeControl() error = %v", err)
	}

	err := state.ResumeRuntimeControl(job.ID)
	if err == nil {
		t.Fatal("ResumeRuntimeControl() error = nil, want completed-step replay rejection")
	}

	validationErr, ok := err.(missioncontrol.ValidationError)
	if !ok {
		t.Fatalf("ResumeRuntimeControl() error type = %T, want ValidationError", err)
	}
	if validationErr.Code != missioncontrol.RejectionCodeInvalidRuntimeState {
		t.Fatalf("ValidationError.Code = %q, want %q", validationErr.Code, missioncontrol.RejectionCodeInvalidRuntimeState)
	}
	if _, ok := state.ExecutionContext(); ok {
		t.Fatal("ExecutionContext() ok = true, want no active context after rejected replay resume")
	}
}

func TestTaskStateResumeRuntimePreservesReusableOneJobApprovalAfterReboot(t *testing.T) {
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
					ID:            "authorize-1",
					Type:          missioncontrol.StepTypeDiscussion,
					Subtype:       missioncontrol.StepSubtypeAuthorization,
					ApprovalScope: missioncontrol.ApprovalScopeOneJob,
				},
				{
					ID:            "authorize-2",
					Type:          missioncontrol.StepTypeDiscussion,
					Subtype:       missioncontrol.StepSubtypeAuthorization,
					ApprovalScope: missioncontrol.ApprovalScopeOneJob,
					DependsOn:     []string{"authorize-1"},
				},
				{
					ID:        "final",
					Type:      missioncontrol.StepTypeFinalResponse,
					DependsOn: []string{"authorize-2"},
				},
			},
		},
	}
	now := time.Now().UTC()
	runtimeState := missioncontrol.JobRuntimeState{
		JobID:        job.ID,
		State:        missioncontrol.JobStateRunning,
		ActiveStepID: "authorize-2",
		ApprovalRequests: []missioncontrol.ApprovalRequest{
			{
				JobID:           job.ID,
				StepID:          "authorize-1",
				RequestedAction: missioncontrol.ApprovalRequestedActionStepComplete,
				Scope:           missioncontrol.ApprovalScopeOneJob,
				State:           missioncontrol.ApprovalStateGranted,
				RequestedAt:     now.Add(-2 * time.Minute),
				ResolvedAt:      now.Add(-90 * time.Second),
			},
		},
		ApprovalGrants: []missioncontrol.ApprovalGrant{
			{
				JobID:           job.ID,
				StepID:          "authorize-1",
				RequestedAction: missioncontrol.ApprovalRequestedActionStepComplete,
				Scope:           missioncontrol.ApprovalScopeOneJob,
				GrantedVia:      missioncontrol.ApprovalGrantedViaOperatorCommand,
				State:           missioncontrol.ApprovalStateGranted,
				GrantedAt:       now.Add(-90 * time.Second),
				ExpiresAt:       now.Add(time.Minute),
			},
		},
	}

	if err := state.ResumeRuntime(job, runtimeState, nil, true); err != nil {
		t.Fatalf("ResumeRuntime() error = %v", err)
	}
	if err := state.ApplyStepOutput("Need approval before continuing.", nil); err != nil {
		t.Fatalf("ApplyStepOutput() error = %v", err)
	}

	if _, ok := state.ExecutionContext(); ok {
		t.Fatal("ExecutionContext() ok = true, want false after completion")
	}
	runtime, ok := state.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if runtime.State != missioncontrol.JobStatePaused {
		t.Fatalf("MissionRuntimeState().State = %q, want %q", runtime.State, missioncontrol.JobStatePaused)
	}
	if len(runtime.CompletedSteps) != 1 || runtime.CompletedSteps[0].StepID != "authorize-2" {
		t.Fatalf("MissionRuntimeState().CompletedSteps = %#v, want authorize-2 completion", runtime.CompletedSteps)
	}
	if len(runtime.ApprovalRequests) != 2 || runtime.ApprovalRequests[1].State != missioncontrol.ApprovalStateGranted {
		t.Fatalf("MissionRuntimeState().ApprovalRequests = %#v, want reused one_job approval recorded", runtime.ApprovalRequests)
	}
}

func TestTaskStateResumeRuntimePreservesReusableOneSessionApprovalAfterReboot(t *testing.T) {
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
					ID:            "authorize-1",
					Type:          missioncontrol.StepTypeDiscussion,
					Subtype:       missioncontrol.StepSubtypeAuthorization,
					ApprovalScope: missioncontrol.ApprovalScopeOneSession,
				},
				{
					ID:            "authorize-2",
					Type:          missioncontrol.StepTypeDiscussion,
					Subtype:       missioncontrol.StepSubtypeAuthorization,
					ApprovalScope: missioncontrol.ApprovalScopeOneSession,
					DependsOn:     []string{"authorize-1"},
				},
				{
					ID:        "final",
					Type:      missioncontrol.StepTypeFinalResponse,
					DependsOn: []string{"authorize-2"},
				},
			},
		},
	}
	now := time.Now().UTC()
	runtimeState := missioncontrol.JobRuntimeState{
		JobID:        job.ID,
		State:        missioncontrol.JobStateRunning,
		ActiveStepID: "authorize-2",
		ApprovalRequests: []missioncontrol.ApprovalRequest{
			{
				JobID:           job.ID,
				StepID:          "authorize-1",
				RequestedAction: missioncontrol.ApprovalRequestedActionStepComplete,
				Scope:           missioncontrol.ApprovalScopeOneSession,
				SessionChannel:  "telegram",
				SessionChatID:   "chat-42",
				State:           missioncontrol.ApprovalStateGranted,
				RequestedAt:     now.Add(-2 * time.Minute),
				ResolvedAt:      now.Add(-90 * time.Second),
			},
		},
		ApprovalGrants: []missioncontrol.ApprovalGrant{
			{
				JobID:           job.ID,
				StepID:          "authorize-1",
				RequestedAction: missioncontrol.ApprovalRequestedActionStepComplete,
				Scope:           missioncontrol.ApprovalScopeOneSession,
				GrantedVia:      missioncontrol.ApprovalGrantedViaOperatorCommand,
				SessionChannel:  "telegram",
				SessionChatID:   "chat-42",
				State:           missioncontrol.ApprovalStateGranted,
				GrantedAt:       now.Add(-90 * time.Second),
				ExpiresAt:       now.Add(time.Minute),
			},
		},
	}

	state.SetOperatorSession("telegram", "chat-42")
	if err := state.ResumeRuntime(job, runtimeState, nil, true); err != nil {
		t.Fatalf("ResumeRuntime() error = %v", err)
	}
	if err := state.ApplyStepOutput("Need approval before continuing.", nil); err != nil {
		t.Fatalf("ApplyStepOutput() error = %v", err)
	}

	if _, ok := state.ExecutionContext(); ok {
		t.Fatal("ExecutionContext() ok = true, want false after completion")
	}
	runtime, ok := state.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if runtime.State != missioncontrol.JobStatePaused {
		t.Fatalf("MissionRuntimeState().State = %q, want %q", runtime.State, missioncontrol.JobStatePaused)
	}
	if len(runtime.CompletedSteps) != 1 || runtime.CompletedSteps[0].StepID != "authorize-2" {
		t.Fatalf("MissionRuntimeState().CompletedSteps = %#v, want authorize-2 completion", runtime.CompletedSteps)
	}
	if len(runtime.ApprovalRequests) != 2 || runtime.ApprovalRequests[1].State != missioncontrol.ApprovalStateGranted {
		t.Fatalf("MissionRuntimeState().ApprovalRequests = %#v, want reused one_session approval recorded", runtime.ApprovalRequests)
	}
	if runtime.ApprovalRequests[1].SessionChannel != "telegram" || runtime.ApprovalRequests[1].SessionChatID != "chat-42" {
		t.Fatalf("MissionRuntimeState().ApprovalRequests[1] session = (%q, %q), want (%q, %q)", runtime.ApprovalRequests[1].SessionChannel, runtime.ApprovalRequests[1].SessionChatID, "telegram", "chat-42")
	}
}

func TestTaskStateResumeRuntimeUsesDeterministicLatestReusableApprovalAfterReboot(t *testing.T) {
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
					ID:            "authorize-a",
					Type:          missioncontrol.StepTypeDiscussion,
					Subtype:       missioncontrol.StepSubtypeAuthorization,
					ApprovalScope: missioncontrol.ApprovalScopeOneJob,
				},
				{
					ID:            "authorize-b",
					Type:          missioncontrol.StepTypeDiscussion,
					Subtype:       missioncontrol.StepSubtypeAuthorization,
					ApprovalScope: missioncontrol.ApprovalScopeOneJob,
					DependsOn:     []string{"authorize-a"},
				},
				{
					ID:            "authorize-c",
					Type:          missioncontrol.StepTypeDiscussion,
					Subtype:       missioncontrol.StepSubtypeAuthorization,
					ApprovalScope: missioncontrol.ApprovalScopeOneJob,
					DependsOn:     []string{"authorize-b"},
				},
				{
					ID:        "final",
					Type:      missioncontrol.StepTypeFinalResponse,
					DependsOn: []string{"authorize-c"},
				},
			},
		},
	}
	now := time.Now().UTC()
	runtimeState := missioncontrol.JobRuntimeState{
		JobID:        job.ID,
		State:        missioncontrol.JobStateRunning,
		ActiveStepID: "authorize-c",
		ApprovalRequests: []missioncontrol.ApprovalRequest{
			{
				JobID:           job.ID,
				StepID:          "authorize-b",
				RequestedAction: missioncontrol.ApprovalRequestedActionStepComplete,
				Scope:           missioncontrol.ApprovalScopeOneJob,
				State:           missioncontrol.ApprovalStateGranted,
				RequestedAt:     now.Add(-90 * time.Second),
				ResolvedAt:      now.Add(-time.Minute),
			},
			{
				JobID:           job.ID,
				StepID:          "authorize-a",
				RequestedAction: missioncontrol.ApprovalRequestedActionStepComplete,
				Scope:           missioncontrol.ApprovalScopeOneJob,
				State:           missioncontrol.ApprovalStateGranted,
				RequestedAt:     now.Add(-3 * time.Minute),
				ResolvedAt:      now.Add(-2 * time.Minute),
			},
		},
		ApprovalGrants: []missioncontrol.ApprovalGrant{
			{
				JobID:           job.ID,
				StepID:          "authorize-b",
				RequestedAction: missioncontrol.ApprovalRequestedActionStepComplete,
				Scope:           missioncontrol.ApprovalScopeOneJob,
				GrantedVia:      missioncontrol.ApprovalGrantedViaOperatorReply,
				State:           missioncontrol.ApprovalStateGranted,
				GrantedAt:       now.Add(-time.Minute),
				ExpiresAt:       now.Add(time.Minute),
			},
			{
				JobID:           job.ID,
				StepID:          "authorize-a",
				RequestedAction: missioncontrol.ApprovalRequestedActionStepComplete,
				Scope:           missioncontrol.ApprovalScopeOneJob,
				GrantedVia:      missioncontrol.ApprovalGrantedViaOperatorCommand,
				State:           missioncontrol.ApprovalStateGranted,
				GrantedAt:       now.Add(-2 * time.Minute),
				ExpiresAt:       now.Add(time.Minute),
			},
		},
	}

	if err := state.ResumeRuntime(job, runtimeState, nil, true); err != nil {
		t.Fatalf("ResumeRuntime() error = %v", err)
	}
	if err := state.ApplyStepOutput("Need approval before continuing.", nil); err != nil {
		t.Fatalf("ApplyStepOutput() error = %v", err)
	}

	runtime, ok := state.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if len(runtime.ApprovalRequests) != 3 {
		t.Fatalf("MissionRuntimeState().ApprovalRequests = %#v, want three approval records", runtime.ApprovalRequests)
	}
	if runtime.ApprovalRequests[2].StepID != "authorize-c" {
		t.Fatalf("MissionRuntimeState().ApprovalRequests[2].StepID = %q, want %q", runtime.ApprovalRequests[2].StepID, "authorize-c")
	}
	if runtime.ApprovalRequests[2].GrantedVia != missioncontrol.ApprovalGrantedViaOperatorReply {
		t.Fatalf("MissionRuntimeState().ApprovalRequests[2].GrantedVia = %q, want %q", runtime.ApprovalRequests[2].GrantedVia, missioncontrol.ApprovalGrantedViaOperatorReply)
	}
}

func TestTaskStateRevokeApprovalPreventsOneJobReuse(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	job := testReusableApprovalJob(missioncontrol.ApprovalScopeOneJob)
	now := time.Now().UTC()
	runtimeState := missioncontrol.JobRuntimeState{
		JobID:        job.ID,
		State:        missioncontrol.JobStateRunning,
		ActiveStepID: "authorize-2",
		ApprovalRequests: []missioncontrol.ApprovalRequest{
			{
				JobID:           job.ID,
				StepID:          "authorize-1",
				RequestedAction: missioncontrol.ApprovalRequestedActionStepComplete,
				Scope:           missioncontrol.ApprovalScopeOneJob,
				RequestedVia:    missioncontrol.ApprovalRequestedViaRuntime,
				GrantedVia:      missioncontrol.ApprovalGrantedViaOperatorCommand,
				State:           missioncontrol.ApprovalStateGranted,
				RequestedAt:     now.Add(-2 * time.Minute),
				ResolvedAt:      now.Add(-90 * time.Second),
			},
		},
		ApprovalGrants: []missioncontrol.ApprovalGrant{
			{
				JobID:           job.ID,
				StepID:          "authorize-1",
				RequestedAction: missioncontrol.ApprovalRequestedActionStepComplete,
				Scope:           missioncontrol.ApprovalScopeOneJob,
				GrantedVia:      missioncontrol.ApprovalGrantedViaOperatorCommand,
				State:           missioncontrol.ApprovalStateGranted,
				GrantedAt:       now.Add(-90 * time.Second),
				ExpiresAt:       now.Add(time.Minute),
			},
		},
	}

	if err := state.ResumeRuntime(job, runtimeState, nil, true); err != nil {
		t.Fatalf("ResumeRuntime() error = %v", err)
	}
	if err := state.RevokeApproval(job.ID, "authorize-2"); err != nil {
		t.Fatalf("RevokeApproval() error = %v", err)
	}
	if err := state.ApplyStepOutput("Need approval before continuing.", nil); err != nil {
		t.Fatalf("ApplyStepOutput() error = %v", err)
	}

	runtime, ok := state.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if runtime.State != missioncontrol.JobStateWaitingUser {
		t.Fatalf("MissionRuntimeState().State = %q, want %q", runtime.State, missioncontrol.JobStateWaitingUser)
	}
	if len(runtime.ApprovalRequests) != 2 || runtime.ApprovalRequests[0].State != missioncontrol.ApprovalStateRevoked || runtime.ApprovalRequests[1].State != missioncontrol.ApprovalStatePending {
		t.Fatalf("MissionRuntimeState().ApprovalRequests = %#v, want revoked then pending approvals", runtime.ApprovalRequests)
	}
	if runtime.ApprovalRequests[0].RevokedAt.IsZero() {
		t.Fatalf("MissionRuntimeState().ApprovalRequests[0].RevokedAt = %v, want stamped revoke time", runtime.ApprovalRequests[0].RevokedAt)
	}
	if len(runtime.ApprovalGrants) != 1 || runtime.ApprovalGrants[0].State != missioncontrol.ApprovalStateRevoked {
		t.Fatalf("MissionRuntimeState().ApprovalGrants = %#v, want one revoked approval grant", runtime.ApprovalGrants)
	}
}

func TestTaskStateRevokeApprovalPreventsOneSessionReuse(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	job := testReusableApprovalJob(missioncontrol.ApprovalScopeOneSession)
	now := time.Now().UTC()
	runtimeState := missioncontrol.JobRuntimeState{
		JobID:        job.ID,
		State:        missioncontrol.JobStateRunning,
		ActiveStepID: "authorize-2",
		ApprovalRequests: []missioncontrol.ApprovalRequest{
			{
				JobID:           job.ID,
				StepID:          "authorize-1",
				RequestedAction: missioncontrol.ApprovalRequestedActionStepComplete,
				Scope:           missioncontrol.ApprovalScopeOneSession,
				RequestedVia:    missioncontrol.ApprovalRequestedViaRuntime,
				GrantedVia:      missioncontrol.ApprovalGrantedViaOperatorCommand,
				SessionChannel:  "telegram",
				SessionChatID:   "chat-42",
				State:           missioncontrol.ApprovalStateGranted,
				RequestedAt:     now.Add(-2 * time.Minute),
				ResolvedAt:      now.Add(-90 * time.Second),
			},
		},
		ApprovalGrants: []missioncontrol.ApprovalGrant{
			{
				JobID:           job.ID,
				StepID:          "authorize-1",
				RequestedAction: missioncontrol.ApprovalRequestedActionStepComplete,
				Scope:           missioncontrol.ApprovalScopeOneSession,
				GrantedVia:      missioncontrol.ApprovalGrantedViaOperatorCommand,
				SessionChannel:  "telegram",
				SessionChatID:   "chat-42",
				State:           missioncontrol.ApprovalStateGranted,
				GrantedAt:       now.Add(-90 * time.Second),
				ExpiresAt:       now.Add(time.Minute),
			},
		},
	}

	state.SetOperatorSession("telegram", "chat-42")
	if err := state.ResumeRuntime(job, runtimeState, nil, true); err != nil {
		t.Fatalf("ResumeRuntime() error = %v", err)
	}
	if err := state.RevokeApproval(job.ID, "authorize-2"); err != nil {
		t.Fatalf("RevokeApproval() error = %v", err)
	}
	if err := state.ApplyStepOutput("Need approval before continuing.", nil); err != nil {
		t.Fatalf("ApplyStepOutput() error = %v", err)
	}

	runtime, ok := state.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if len(runtime.ApprovalRequests) != 2 || runtime.ApprovalRequests[0].State != missioncontrol.ApprovalStateRevoked || runtime.ApprovalRequests[1].State != missioncontrol.ApprovalStatePending {
		t.Fatalf("MissionRuntimeState().ApprovalRequests = %#v, want revoked then pending approvals", runtime.ApprovalRequests)
	}
	if runtime.ApprovalRequests[0].RevokedAt.IsZero() {
		t.Fatalf("MissionRuntimeState().ApprovalRequests[0].RevokedAt = %v, want stamped revoke time", runtime.ApprovalRequests[0].RevokedAt)
	}
	if len(runtime.ApprovalGrants) != 1 || runtime.ApprovalGrants[0].State != missioncontrol.ApprovalStateRevoked {
		t.Fatalf("MissionRuntimeState().ApprovalGrants = %#v, want one revoked approval grant", runtime.ApprovalGrants)
	}
}

func TestTaskStateRevokeApprovalDoesNotAffectDifferentSession(t *testing.T) {
	t.Parallel()

	job := testReusableApprovalJob(missioncontrol.ApprovalScopeOneSession)
	now := time.Now().UTC()
	runtimeState := missioncontrol.JobRuntimeState{
		JobID:        job.ID,
		State:        missioncontrol.JobStateRunning,
		ActiveStepID: "authorize-2",
		ApprovalRequests: []missioncontrol.ApprovalRequest{
			{
				JobID:           job.ID,
				StepID:          "authorize-1",
				RequestedAction: missioncontrol.ApprovalRequestedActionStepComplete,
				Scope:           missioncontrol.ApprovalScopeOneSession,
				RequestedVia:    missioncontrol.ApprovalRequestedViaRuntime,
				GrantedVia:      missioncontrol.ApprovalGrantedViaOperatorCommand,
				SessionChannel:  "telegram",
				SessionChatID:   "chat-42",
				State:           missioncontrol.ApprovalStateGranted,
				RequestedAt:     now.Add(-2 * time.Minute),
				ResolvedAt:      now.Add(-90 * time.Second),
			},
		},
		ApprovalGrants: []missioncontrol.ApprovalGrant{
			{
				JobID:           job.ID,
				StepID:          "authorize-1",
				RequestedAction: missioncontrol.ApprovalRequestedActionStepComplete,
				Scope:           missioncontrol.ApprovalScopeOneSession,
				GrantedVia:      missioncontrol.ApprovalGrantedViaOperatorCommand,
				SessionChannel:  "telegram",
				SessionChatID:   "chat-42",
				State:           missioncontrol.ApprovalStateGranted,
				GrantedAt:       now.Add(-90 * time.Second),
				ExpiresAt:       now.Add(time.Minute),
			},
		},
	}

	state := NewTaskState()
	state.SetOperatorSession("slack", "C123::171234")
	if err := state.ResumeRuntime(job, runtimeState, nil, true); err != nil {
		t.Fatalf("ResumeRuntime() error = %v", err)
	}
	if err := state.RevokeApproval(job.ID, "authorize-2"); err == nil {
		t.Fatal("RevokeApproval() error = nil, want session mismatch failure")
	}

	reuseState := NewTaskState()
	reuseState.SetOperatorSession("telegram", "chat-42")
	if err := reuseState.ResumeRuntime(job, runtimeState, nil, true); err != nil {
		t.Fatalf("ResumeRuntime() error = %v", err)
	}
	if err := reuseState.ApplyStepOutput("Need approval before continuing.", nil); err != nil {
		t.Fatalf("ApplyStepOutput() error = %v", err)
	}

	runtime, ok := reuseState.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if runtime.State != missioncontrol.JobStatePaused {
		t.Fatalf("MissionRuntimeState().State = %q, want %q", runtime.State, missioncontrol.JobStatePaused)
	}
	if len(runtime.ApprovalRequests) != 2 || runtime.ApprovalRequests[1].State != missioncontrol.ApprovalStateGranted {
		t.Fatalf("MissionRuntimeState().ApprovalRequests = %#v, want reused one_session approval recorded", runtime.ApprovalRequests)
	}
}

func TestTaskStateRevokeApprovalWrongJobOrStepDoesNotBind(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	job := testReusableApprovalJob(missioncontrol.ApprovalScopeOneJob)
	now := time.Now().UTC()
	runtimeState := missioncontrol.JobRuntimeState{
		JobID:        job.ID,
		State:        missioncontrol.JobStateRunning,
		ActiveStepID: "authorize-2",
		ApprovalRequests: []missioncontrol.ApprovalRequest{
			{
				JobID:           job.ID,
				StepID:          "authorize-1",
				RequestedAction: missioncontrol.ApprovalRequestedActionStepComplete,
				Scope:           missioncontrol.ApprovalScopeOneJob,
				RequestedVia:    missioncontrol.ApprovalRequestedViaRuntime,
				GrantedVia:      missioncontrol.ApprovalGrantedViaOperatorCommand,
				State:           missioncontrol.ApprovalStateGranted,
				RequestedAt:     now.Add(-2 * time.Minute),
				ResolvedAt:      now.Add(-90 * time.Second),
			},
		},
		ApprovalGrants: []missioncontrol.ApprovalGrant{
			{
				JobID:           job.ID,
				StepID:          "authorize-1",
				RequestedAction: missioncontrol.ApprovalRequestedActionStepComplete,
				Scope:           missioncontrol.ApprovalScopeOneJob,
				GrantedVia:      missioncontrol.ApprovalGrantedViaOperatorCommand,
				State:           missioncontrol.ApprovalStateGranted,
				GrantedAt:       now.Add(-90 * time.Second),
				ExpiresAt:       now.Add(time.Minute),
			},
		},
	}

	if err := state.ResumeRuntime(job, runtimeState, nil, true); err != nil {
		t.Fatalf("ResumeRuntime() error = %v", err)
	}
	if err := state.RevokeApproval("other-job", "authorize-2"); err == nil {
		t.Fatal("RevokeApproval(wrong job) error = nil, want mismatch failure")
	}
	if err := state.RevokeApproval(job.ID, "authorize-1"); err == nil {
		t.Fatal("RevokeApproval(wrong step) error = nil, want mismatch failure")
	}

	runtime, ok := state.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if len(runtime.ApprovalRequests) != 1 || runtime.ApprovalRequests[0].State != missioncontrol.ApprovalStateGranted {
		t.Fatalf("MissionRuntimeState().ApprovalRequests = %#v, want unchanged granted approval", runtime.ApprovalRequests)
	}
	if len(runtime.ApprovalGrants) != 1 || runtime.ApprovalGrants[0].State != missioncontrol.ApprovalStateGranted {
		t.Fatalf("MissionRuntimeState().ApprovalGrants = %#v, want unchanged granted approval", runtime.ApprovalGrants)
	}
}

func TestTaskStatePersistedRuntimeRevocationHonorsRevokedApprovalState(t *testing.T) {
	t.Parallel()

	job := missioncontrol.Job{
		ID:           "job-1",
		MaxAuthority: missioncontrol.AuthorityTierHigh,
		AllowedTools: []string{"read"},
		Plan: missioncontrol.Plan{
			ID: "plan-1",
			Steps: []missioncontrol.Step{
				{
					ID:            "authorize-1",
					Type:          missioncontrol.StepTypeDiscussion,
					Subtype:       missioncontrol.StepSubtypeAuthorization,
					ApprovalScope: missioncontrol.ApprovalScopeOneJob,
				},
				{
					ID:            "authorize-2",
					Type:          missioncontrol.StepTypeDiscussion,
					Subtype:       missioncontrol.StepSubtypeAuthorization,
					ApprovalScope: missioncontrol.ApprovalScopeOneJob,
					DependsOn:     []string{"authorize-1"},
				},
				{
					ID:        "final",
					Type:      missioncontrol.StepTypeFinalResponse,
					DependsOn: []string{"authorize-2"},
				},
			},
		},
	}
	control, err := missioncontrol.BuildRuntimeControlContext(job, "authorize-2")
	if err != nil {
		t.Fatalf("BuildRuntimeControlContext() error = %v", err)
	}
	now := time.Now().UTC()
	runtimeState := missioncontrol.JobRuntimeState{
		JobID:        job.ID,
		State:        missioncontrol.JobStateRunning,
		ActiveStepID: "authorize-2",
		ApprovalRequests: []missioncontrol.ApprovalRequest{
			{
				JobID:           job.ID,
				StepID:          "authorize-1",
				RequestedAction: missioncontrol.ApprovalRequestedActionStepComplete,
				Scope:           missioncontrol.ApprovalScopeOneJob,
				RequestedVia:    missioncontrol.ApprovalRequestedViaRuntime,
				GrantedVia:      missioncontrol.ApprovalGrantedViaOperatorCommand,
				State:           missioncontrol.ApprovalStateGranted,
				RequestedAt:     now.Add(-2 * time.Minute),
				ResolvedAt:      now.Add(-90 * time.Second),
			},
		},
		ApprovalGrants: []missioncontrol.ApprovalGrant{
			{
				JobID:           job.ID,
				StepID:          "authorize-1",
				RequestedAction: missioncontrol.ApprovalRequestedActionStepComplete,
				Scope:           missioncontrol.ApprovalScopeOneJob,
				GrantedVia:      missioncontrol.ApprovalGrantedViaOperatorCommand,
				State:           missioncontrol.ApprovalStateGranted,
				GrantedAt:       now.Add(-90 * time.Second),
				ExpiresAt:       now.Add(time.Minute),
			},
		},
	}

	state := NewTaskState()
	if err := state.HydrateRuntimeControl(job, runtimeState, &control); err != nil {
		t.Fatalf("HydrateRuntimeControl() error = %v", err)
	}
	if err := state.RevokeApproval(job.ID, "authorize-2"); err != nil {
		t.Fatalf("RevokeApproval() error = %v", err)
	}

	revokedRuntime, ok := state.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if len(revokedRuntime.ApprovalGrants) != 1 || revokedRuntime.ApprovalGrants[0].State != missioncontrol.ApprovalStateRevoked {
		t.Fatalf("MissionRuntimeState().ApprovalGrants = %#v, want one revoked approval grant", revokedRuntime.ApprovalGrants)
	}

	resumed := NewTaskState()
	if err := resumed.ResumeRuntime(job, revokedRuntime, nil, true); err != nil {
		t.Fatalf("ResumeRuntime() error = %v", err)
	}
	if err := resumed.ApplyStepOutput("Need approval before continuing.", nil); err != nil {
		t.Fatalf("ApplyStepOutput() error = %v", err)
	}

	runtime, ok := resumed.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if runtime.State != missioncontrol.JobStateWaitingUser {
		t.Fatalf("MissionRuntimeState().State = %q, want %q", runtime.State, missioncontrol.JobStateWaitingUser)
	}
	if len(runtime.ApprovalRequests) != 2 || runtime.ApprovalRequests[1].State != missioncontrol.ApprovalStatePending {
		t.Fatalf("MissionRuntimeState().ApprovalRequests = %#v, want revoked then pending approvals after reboot-safe resume", runtime.ApprovalRequests)
	}
}

func TestTaskStateResumeRuntimeControlResumesPausedRuntimeAfterExecutionContextTeardown(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	if err := state.ActivateStep(testTaskStateJob(), "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}
	if err := state.PauseRuntime("job-1"); err != nil {
		t.Fatalf("PauseRuntime() error = %v", err)
	}

	state.ClearExecutionContext()

	if _, ok := state.ExecutionContext(); ok {
		t.Fatal("ExecutionContext() ok = true, want false after teardown")
	}
	if err := state.ResumeRuntimeControl("job-1"); err != nil {
		t.Fatalf("ResumeRuntimeControl() error = %v", err)
	}

	ec, ok := state.ExecutionContext()
	if !ok {
		t.Fatal("ExecutionContext() ok = false, want restored active context")
	}
	if ec.Runtime == nil {
		t.Fatal("ExecutionContext().Runtime = nil, want non-nil")
	}
	if ec.Runtime.State != missioncontrol.JobStateRunning {
		t.Fatalf("ExecutionContext().Runtime.State = %q, want %q", ec.Runtime.State, missioncontrol.JobStateRunning)
	}
	if ec.Runtime.ActiveStepID != "build" {
		t.Fatalf("ExecutionContext().Runtime.ActiveStepID = %q, want %q", ec.Runtime.ActiveStepID, "build")
	}
}

func TestTaskStateResumeRuntimeControlDoesNotBypassPendingApproval(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	job := testTaskStateJob()
	job.Plan.Steps[0] = missioncontrol.Step{
		ID:      "build",
		Type:    missioncontrol.StepTypeDiscussion,
		Subtype: missioncontrol.StepSubtypeAuthorization,
	}

	if err := state.ActivateStep(job, "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}
	if err := state.ApplyStepOutput("Waiting for approval.", nil); err != nil {
		t.Fatalf("ApplyStepOutput() error = %v", err)
	}

	err := state.ResumeRuntimeControl("job-1")
	if err == nil {
		t.Fatal("ResumeRuntimeControl() error = nil, want waiting_user failure")
	}

	ec, ok := state.ExecutionContext()
	if !ok {
		t.Fatal("ExecutionContext() ok = false, want waiting execution context")
	}
	if ec.Runtime == nil || ec.Runtime.State != missioncontrol.JobStateWaitingUser {
		t.Fatalf("ExecutionContext().Runtime = %#v, want waiting_user runtime", ec.Runtime)
	}
	if len(ec.Runtime.ApprovalRequests) != 1 || ec.Runtime.ApprovalRequests[0].State != missioncontrol.ApprovalStatePending {
		t.Fatalf("ExecutionContext().Runtime.ApprovalRequests = %#v, want one pending approval", ec.Runtime.ApprovalRequests)
	}
}

func TestTaskStateResumeRuntimeControlDoesNotBypassDeniedApproval(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	job := testTaskStateJob()
	job.Plan.Steps[0] = missioncontrol.Step{
		ID:      "build",
		Type:    missioncontrol.StepTypeDiscussion,
		Subtype: missioncontrol.StepSubtypeAuthorization,
	}

	if err := state.ActivateStep(job, "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}
	if err := state.ApplyStepOutput("Waiting for approval.", nil); err != nil {
		t.Fatalf("ApplyStepOutput() error = %v", err)
	}
	if err := state.ApplyApprovalDecision("job-1", "build", missioncontrol.ApprovalDecisionDeny, missioncontrol.ApprovalGrantedViaOperatorCommand); err != nil {
		t.Fatalf("ApplyApprovalDecision() error = %v", err)
	}

	err := state.ResumeRuntimeControl("job-1")
	if err == nil {
		t.Fatal("ResumeRuntimeControl() error = nil, want waiting_user failure")
	}

	ec, ok := state.ExecutionContext()
	if !ok {
		t.Fatal("ExecutionContext() ok = false, want waiting execution context")
	}
	if ec.Runtime == nil || ec.Runtime.State != missioncontrol.JobStateWaitingUser {
		t.Fatalf("ExecutionContext().Runtime = %#v, want waiting_user runtime", ec.Runtime)
	}
	if len(ec.Runtime.ApprovalRequests) != 1 || ec.Runtime.ApprovalRequests[0].State != missioncontrol.ApprovalStateDenied {
		t.Fatalf("ExecutionContext().Runtime.ApprovalRequests = %#v, want one denied approval", ec.Runtime.ApprovalRequests)
	}
}

func TestTaskStateResumeRuntimeControlWrongJobDoesNotBindAfterExecutionContextTeardown(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	if err := state.ActivateStep(testTaskStateJob(), "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}
	if err := state.PauseRuntime("job-1"); err != nil {
		t.Fatalf("PauseRuntime() error = %v", err)
	}

	state.ClearExecutionContext()

	err := state.ResumeRuntimeControl("other-job")
	if err == nil {
		t.Fatal("ResumeRuntimeControl() error = nil, want job mismatch failure")
	}

	validationErr, ok := err.(missioncontrol.ValidationError)
	if !ok {
		t.Fatalf("ResumeRuntimeControl() error type = %T, want ValidationError", err)
	}
	if validationErr.Code != missioncontrol.RejectionCodeStepValidationFailed {
		t.Fatalf("ValidationError.Code = %q, want %q", validationErr.Code, missioncontrol.RejectionCodeStepValidationFailed)
	}

	if _, ok := state.ExecutionContext(); ok {
		t.Fatal("ExecutionContext() ok = true, want false after wrong-job rejection")
	}
	runtime, ok := state.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want durable paused runtime")
	}
	if runtime.State != missioncontrol.JobStatePaused {
		t.Fatalf("MissionRuntimeState().State = %q, want %q", runtime.State, missioncontrol.JobStatePaused)
	}
	if runtime.ActiveStepID != "build" {
		t.Fatalf("MissionRuntimeState().ActiveStepID = %q, want %q", runtime.ActiveStepID, "build")
	}
}

func TestTaskStateHydrateRuntimeControlWrongJobDoesNotBindAfterRehydration(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	job := testTaskStateJob()
	runtime := missioncontrol.JobRuntimeState{
		JobID:        job.ID,
		State:        missioncontrol.JobStatePaused,
		ActiveStepID: "build",
		PausedReason: missioncontrol.RuntimePauseReasonOperatorCommand,
	}

	if err := state.HydrateRuntimeControl(job, runtime, nil); err != nil {
		t.Fatalf("HydrateRuntimeControl() error = %v", err)
	}

	err := state.ResumeRuntimeControl("other-job")
	if err == nil {
		t.Fatal("ResumeRuntimeControl() error = nil, want job mismatch failure")
	}

	validationErr, ok := err.(missioncontrol.ValidationError)
	if !ok {
		t.Fatalf("ResumeRuntimeControl() error type = %T, want ValidationError", err)
	}
	if validationErr.Code != missioncontrol.RejectionCodeStepValidationFailed {
		t.Fatalf("ValidationError.Code = %q, want %q", validationErr.Code, missioncontrol.RejectionCodeStepValidationFailed)
	}
}

func TestTaskStateAbortRuntimeTransitionsToAborted(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	if err := state.ActivateStep(testTaskStateJob(), "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}
	if err := state.PauseRuntime("job-1"); err != nil {
		t.Fatalf("PauseRuntime() error = %v", err)
	}

	if err := state.AbortRuntime("job-1"); err != nil {
		t.Fatalf("AbortRuntime() error = %v", err)
	}

	if _, ok := state.ExecutionContext(); ok {
		t.Fatal("ExecutionContext() ok = true, want false after abort")
	}

	runtime, ok := state.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if runtime.State != missioncontrol.JobStateAborted {
		t.Fatalf("MissionRuntimeState().State = %q, want %q", runtime.State, missioncontrol.JobStateAborted)
	}
	if runtime.AbortedReason != missioncontrol.RuntimeAbortReasonOperatorCommand {
		t.Fatalf("MissionRuntimeState().AbortedReason = %q, want %q", runtime.AbortedReason, missioncontrol.RuntimeAbortReasonOperatorCommand)
	}
}

func TestTaskStateHydrateRuntimeControlAbortsWaitingUserRuntimeAfterRehydration(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	job := testTaskStateJob()
	job.Plan.Steps[0] = missioncontrol.Step{
		ID:      "build",
		Type:    missioncontrol.StepTypeDiscussion,
		Subtype: missioncontrol.StepSubtypeAuthorization,
	}
	runtime := missioncontrol.JobRuntimeState{
		JobID:         job.ID,
		State:         missioncontrol.JobStateWaitingUser,
		ActiveStepID:  "build",
		WaitingReason: "awaiting operator input",
		ApprovalRequests: []missioncontrol.ApprovalRequest{
			{
				JobID:           job.ID,
				StepID:          "build",
				RequestedAction: missioncontrol.ApprovalRequestedActionStepComplete,
				Scope:           missioncontrol.ApprovalScopeMissionStep,
				State:           missioncontrol.ApprovalStatePending,
			},
		},
	}

	if err := state.HydrateRuntimeControl(job, runtime, nil); err != nil {
		t.Fatalf("HydrateRuntimeControl() error = %v", err)
	}
	if err := state.AbortRuntime("job-1"); err != nil {
		t.Fatalf("AbortRuntime() error = %v", err)
	}

	if _, ok := state.ExecutionContext(); ok {
		t.Fatal("ExecutionContext() ok = true, want false after abort")
	}
	got, ok := state.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if got.State != missioncontrol.JobStateAborted {
		t.Fatalf("MissionRuntimeState().State = %q, want %q", got.State, missioncontrol.JobStateAborted)
	}
}

func TestTaskStateAbortRuntimeAbortsWaitingUserRuntimeAfterExecutionContextTeardown(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	job := testTaskStateJob()
	job.Plan.Steps[0] = missioncontrol.Step{
		ID:      "build",
		Type:    missioncontrol.StepTypeDiscussion,
		Subtype: missioncontrol.StepSubtypeAuthorization,
	}

	if err := state.ActivateStep(job, "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}
	if err := state.ApplyStepOutput("Waiting for approval.", nil); err != nil {
		t.Fatalf("ApplyStepOutput() error = %v", err)
	}

	state.ClearExecutionContext()

	if err := state.AbortRuntime("job-1"); err != nil {
		t.Fatalf("AbortRuntime() error = %v", err)
	}

	if _, ok := state.ExecutionContext(); ok {
		t.Fatal("ExecutionContext() ok = true, want false after abort")
	}
	runtime, ok := state.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if runtime.State != missioncontrol.JobStateAborted {
		t.Fatalf("MissionRuntimeState().State = %q, want %q", runtime.State, missioncontrol.JobStateAborted)
	}
	if runtime.AbortedReason != missioncontrol.RuntimeAbortReasonOperatorCommand {
		t.Fatalf("MissionRuntimeState().AbortedReason = %q, want %q", runtime.AbortedReason, missioncontrol.RuntimeAbortReasonOperatorCommand)
	}
}

func TestTaskStateHydrateRuntimeControlRejectsTerminalOperatorCommands(t *testing.T) {
	t.Parallel()

	for _, stateValue := range []missioncontrol.JobState{
		missioncontrol.JobStateCompleted,
		missioncontrol.JobStateFailed,
		missioncontrol.JobStateAborted,
	} {
		stateValue := stateValue
		t.Run(string(stateValue), func(t *testing.T) {
			t.Parallel()

			state := NewTaskState()
			job := testTaskStateJob()
			runtime := missioncontrol.JobRuntimeState{
				JobID: job.ID,
				State: stateValue,
			}

			if err := state.HydrateRuntimeControl(job, runtime, nil); err != nil {
				t.Fatalf("HydrateRuntimeControl() error = %v", err)
			}

			for _, run := range []struct {
				name string
				fn   func() error
			}{
				{name: "resume", fn: func() error { return state.ResumeRuntimeControl(job.ID) }},
				{name: "abort", fn: func() error { return state.AbortRuntime(job.ID) }},
			} {
				err := run.fn()
				if err == nil {
					t.Fatalf("%s error = nil, want invalid runtime state", run.name)
				}
				validationErr, ok := err.(missioncontrol.ValidationError)
				if !ok {
					t.Fatalf("%s error type = %T, want ValidationError", run.name, err)
				}
				if validationErr.Code != missioncontrol.RejectionCodeInvalidRuntimeState {
					t.Fatalf("%s ValidationError.Code = %q, want %q", run.name, validationErr.Code, missioncontrol.RejectionCodeInvalidRuntimeState)
				}
			}
		})
	}
}

func TestTaskStateApplyApprovalDecisionPausesCompletedStep(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	job := testTaskStateJob()
	job.Plan.Steps[0] = missioncontrol.Step{
		ID:      "build",
		Type:    missioncontrol.StepTypeDiscussion,
		Subtype: missioncontrol.StepSubtypeAuthorization,
	}
	job.Plan.Steps[1] = missioncontrol.Step{
		ID:        "final",
		Type:      missioncontrol.StepTypeFinalResponse,
		DependsOn: []string{"build"},
	}

	if err := state.ActivateStep(job, "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}
	if err := state.ApplyStepOutput("Waiting for approval.", nil); err != nil {
		t.Fatalf("ApplyStepOutput() error = %v", err)
	}

	if err := state.ApplyApprovalDecision("job-1", "build", missioncontrol.ApprovalDecisionApprove, missioncontrol.ApprovalGrantedViaOperatorCommand); err != nil {
		t.Fatalf("ApplyApprovalDecision() error = %v", err)
	}
	inputKind, err := state.ApplyWaitingUserInput("approved")
	if err != nil {
		t.Fatalf("ApplyWaitingUserInput() error = %v", err)
	}
	if inputKind != missioncontrol.WaitingUserInputNone {
		t.Fatalf("ApplyWaitingUserInput() kind = %q, want %q after approval completion", inputKind, missioncontrol.WaitingUserInputNone)
	}
	if _, ok := state.ExecutionContext(); ok {
		t.Fatal("ExecutionContext() ok = true, want false after approval completion")
	}

	runtime, ok := state.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if runtime.State != missioncontrol.JobStatePaused {
		t.Fatalf("MissionRuntimeState().State = %q, want %q", runtime.State, missioncontrol.JobStatePaused)
	}
	if len(runtime.CompletedSteps) != 1 || runtime.CompletedSteps[0].StepID != "build" {
		t.Fatalf("MissionRuntimeState().CompletedSteps = %#v, want build completion", runtime.CompletedSteps)
	}
	if len(runtime.ApprovalGrants) != 1 || runtime.ApprovalGrants[0].State != missioncontrol.ApprovalStateGranted {
		t.Fatalf("MissionRuntimeState().ApprovalGrants = %#v, want one granted approval", runtime.ApprovalGrants)
	}
}

func TestTaskStateApplyApprovalDecisionUsesPersistedRuntimeControlAfterExecutionContextTeardown(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	job := testTaskStateJob()
	job.Plan.Steps[0] = missioncontrol.Step{
		ID:      "build",
		Type:    missioncontrol.StepTypeDiscussion,
		Subtype: missioncontrol.StepSubtypeAuthorization,
	}
	job.Plan.Steps[1] = missioncontrol.Step{
		ID:        "final",
		Type:      missioncontrol.StepTypeFinalResponse,
		DependsOn: []string{"build"},
	}

	if err := state.ActivateStep(job, "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}
	if err := state.ApplyStepOutput("Waiting for approval.", nil); err != nil {
		t.Fatalf("ApplyStepOutput() error = %v", err)
	}

	state.SetOperatorSession("telegram", "chat-42")
	state.ClearExecutionContext()

	if err := state.ApplyApprovalDecision("job-1", "build", missioncontrol.ApprovalDecisionApprove, missioncontrol.ApprovalGrantedViaOperatorCommand); err != nil {
		t.Fatalf("ApplyApprovalDecision() error = %v", err)
	}

	if _, ok := state.ExecutionContext(); ok {
		t.Fatal("ExecutionContext() ok = true, want false after reboot-safe approval completion")
	}

	runtime, ok := state.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if runtime.State != missioncontrol.JobStatePaused {
		t.Fatalf("MissionRuntimeState().State = %q, want %q", runtime.State, missioncontrol.JobStatePaused)
	}
	if len(runtime.CompletedSteps) != 1 || runtime.CompletedSteps[0].StepID != "build" {
		t.Fatalf("MissionRuntimeState().CompletedSteps = %#v, want build completion", runtime.CompletedSteps)
	}
	if len(runtime.ApprovalGrants) != 1 || runtime.ApprovalGrants[0].State != missioncontrol.ApprovalStateGranted {
		t.Fatalf("MissionRuntimeState().ApprovalGrants = %#v, want one granted approval", runtime.ApprovalGrants)
	}
	if runtime.ApprovalRequests[0].SessionChannel != "telegram" || runtime.ApprovalRequests[0].SessionChatID != "chat-42" {
		t.Fatalf("MissionRuntimeState().ApprovalRequests[0] session = (%q, %q), want (%q, %q)", runtime.ApprovalRequests[0].SessionChannel, runtime.ApprovalRequests[0].SessionChatID, "telegram", "chat-42")
	}
	if runtime.ApprovalGrants[0].SessionChannel != "telegram" || runtime.ApprovalGrants[0].SessionChatID != "chat-42" {
		t.Fatalf("MissionRuntimeState().ApprovalGrants[0] session = (%q, %q), want (%q, %q)", runtime.ApprovalGrants[0].SessionChannel, runtime.ApprovalGrants[0].SessionChatID, "telegram", "chat-42")
	}
}

func TestTaskStateApplyNaturalApprovalDecisionApprovesSinglePendingRequest(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	job := testTaskStateJob()
	job.Plan.Steps[0] = missioncontrol.Step{
		ID:      "build",
		Type:    missioncontrol.StepTypeDiscussion,
		Subtype: missioncontrol.StepSubtypeAuthorization,
	}

	if err := state.ActivateStep(job, "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}
	if err := state.ApplyStepOutput("Waiting for approval.", nil); err != nil {
		t.Fatalf("ApplyStepOutput() error = %v", err)
	}

	handled, resp, err := state.ApplyNaturalApprovalDecision("yes")
	if err != nil {
		t.Fatalf("ApplyNaturalApprovalDecision(yes) error = %v", err)
	}
	if !handled {
		t.Fatal("ApplyNaturalApprovalDecision(yes) handled = false, want true")
	}
	if resp != "Approved job=job-1 step=build." {
		t.Fatalf("ApplyNaturalApprovalDecision(yes) response = %q, want approval acknowledgement", resp)
	}

	runtime, ok := state.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if runtime.State != missioncontrol.JobStatePaused {
		t.Fatalf("MissionRuntimeState().State = %q, want %q", runtime.State, missioncontrol.JobStatePaused)
	}
}

func TestTaskStateApplyNaturalApprovalDecisionRejectsAmbiguousPendingRequests(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	job := testTaskStateJob()
	job.Plan.Steps[0] = missioncontrol.Step{
		ID:      "build",
		Type:    missioncontrol.StepTypeDiscussion,
		Subtype: missioncontrol.StepSubtypeAuthorization,
	}

	if err := state.ActivateStep(job, "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}
	if err := state.ApplyStepOutput("Waiting for approval.", nil); err != nil {
		t.Fatalf("ApplyStepOutput() error = %v", err)
	}

	state.mu.Lock()
	state.executionContext.Runtime.ApprovalRequests = append(state.executionContext.Runtime.ApprovalRequests, missioncontrol.ApprovalRequest{
		JobID:           "job-1",
		StepID:          "other-step",
		RequestedAction: missioncontrol.ApprovalRequestedActionStepComplete,
		Scope:           missioncontrol.ApprovalScopeMissionStep,
		State:           missioncontrol.ApprovalStatePending,
	})
	state.runtimeState.ApprovalRequests = append(state.runtimeState.ApprovalRequests, missioncontrol.ApprovalRequest{
		JobID:           "job-1",
		StepID:          "other-step",
		RequestedAction: missioncontrol.ApprovalRequestedActionStepComplete,
		Scope:           missioncontrol.ApprovalScopeMissionStep,
		State:           missioncontrol.ApprovalStatePending,
	})
	state.mu.Unlock()

	handled, _, err := state.ApplyNaturalApprovalDecision("yes")
	if err == nil {
		t.Fatal("ApplyNaturalApprovalDecision(yes) error = nil, want ambiguity failure")
	}
	if !handled {
		t.Fatal("ApplyNaturalApprovalDecision(yes) handled = false, want true")
	}

	runtime, ok := state.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if runtime.State != missioncontrol.JobStateWaitingUser {
		t.Fatalf("MissionRuntimeState().State = %q, want %q", runtime.State, missioncontrol.JobStateWaitingUser)
	}
	if len(runtime.ApprovalRequests) != 2 {
		t.Fatalf("MissionRuntimeState().ApprovalRequests = %#v, want two pending approvals", runtime.ApprovalRequests)
	}
}

func TestTaskStateApplyNaturalApprovalDecisionUsesPersistedRuntimeControlAfterExecutionContextTeardown(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	job := testTaskStateJob()
	job.Plan.Steps[0] = missioncontrol.Step{
		ID:      "build",
		Type:    missioncontrol.StepTypeDiscussion,
		Subtype: missioncontrol.StepSubtypeAuthorization,
	}

	if err := state.ActivateStep(job, "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}
	if err := state.ApplyStepOutput("Waiting for approval.", nil); err != nil {
		t.Fatalf("ApplyStepOutput() error = %v", err)
	}

	state.SetOperatorSession("slack", "C123::171234")
	state.ClearExecutionContext()

	handled, resp, err := state.ApplyNaturalApprovalDecision("yes")
	if err != nil {
		t.Fatalf("ApplyNaturalApprovalDecision(yes) error = %v", err)
	}
	if !handled {
		t.Fatal("ApplyNaturalApprovalDecision(yes) handled = false, want true")
	}
	if resp != "Approved job=job-1 step=build." {
		t.Fatalf("ApplyNaturalApprovalDecision(yes) response = %q, want approval acknowledgement", resp)
	}

	runtime, ok := state.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if runtime.State != missioncontrol.JobStatePaused {
		t.Fatalf("MissionRuntimeState().State = %q, want %q", runtime.State, missioncontrol.JobStatePaused)
	}
	if len(runtime.ApprovalRequests) != 1 || runtime.ApprovalRequests[0].State != missioncontrol.ApprovalStateGranted {
		t.Fatalf("MissionRuntimeState().ApprovalRequests = %#v, want one granted approval", runtime.ApprovalRequests)
	}
	if len(runtime.ApprovalGrants) != 1 || runtime.ApprovalGrants[0].State != missioncontrol.ApprovalStateGranted {
		t.Fatalf("MissionRuntimeState().ApprovalGrants = %#v, want one granted approval", runtime.ApprovalGrants)
	}
	if runtime.ApprovalRequests[0].SessionChannel != "slack" || runtime.ApprovalRequests[0].SessionChatID != "C123::171234" {
		t.Fatalf("MissionRuntimeState().ApprovalRequests[0] session = (%q, %q), want (%q, %q)", runtime.ApprovalRequests[0].SessionChannel, runtime.ApprovalRequests[0].SessionChatID, "slack", "C123::171234")
	}
	if runtime.ApprovalGrants[0].SessionChannel != "slack" || runtime.ApprovalGrants[0].SessionChatID != "C123::171234" {
		t.Fatalf("MissionRuntimeState().ApprovalGrants[0] session = (%q, %q), want (%q, %q)", runtime.ApprovalGrants[0].SessionChannel, runtime.ApprovalGrants[0].SessionChatID, "slack", "C123::171234")
	}
}

func TestTaskStateApplyNaturalApprovalDecisionDoesNotBindExpiredPendingRequest(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	job := testTaskStateJob()
	job.Plan.Steps[0] = missioncontrol.Step{
		ID:      "build",
		Type:    missioncontrol.StepTypeDiscussion,
		Subtype: missioncontrol.StepSubtypeAuthorization,
	}

	if err := state.ActivateStep(job, "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}
	if err := state.ApplyStepOutput("Waiting for approval.", nil); err != nil {
		t.Fatalf("ApplyStepOutput() error = %v", err)
	}

	state.mu.Lock()
	expiredAt := time.Now().Add(-1 * time.Minute)
	state.executionContext.Runtime.ApprovalRequests[0].ExpiresAt = expiredAt
	state.runtimeState.ApprovalRequests[0].ExpiresAt = expiredAt
	state.mu.Unlock()

	for _, input := range []string{"yes", "no"} {
		handled, _, err := state.ApplyNaturalApprovalDecision(input)
		if err != nil {
			t.Fatalf("ApplyNaturalApprovalDecision(%q) error = %v", input, err)
		}
		if handled {
			t.Fatalf("ApplyNaturalApprovalDecision(%q) handled = true, want false", input)
		}
	}

	runtime, ok := state.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if len(runtime.ApprovalRequests) != 1 || runtime.ApprovalRequests[0].State != missioncontrol.ApprovalStateExpired {
		t.Fatalf("MissionRuntimeState().ApprovalRequests = %#v, want one expired approval", runtime.ApprovalRequests)
	}
}

func TestTaskStateApplyNaturalApprovalDecisionDoesNotBindSupersededRequest(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	job := testTaskStateJob()
	job.Plan.Steps[0] = missioncontrol.Step{
		ID:      "build",
		Type:    missioncontrol.StepTypeDiscussion,
		Subtype: missioncontrol.StepSubtypeAuthorization,
	}

	if err := state.ActivateStep(job, "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}
	if err := state.ApplyStepOutput("Waiting for approval.", nil); err != nil {
		t.Fatalf("ApplyStepOutput() error = %v", err)
	}

	now := time.Now()
	state.mu.Lock()
	state.executionContext.Runtime.ApprovalRequests = append(state.executionContext.Runtime.ApprovalRequests, missioncontrol.ApprovalRequest{
		JobID:           "job-1",
		StepID:          "build",
		RequestedAction: missioncontrol.ApprovalRequestedActionStepComplete,
		Scope:           missioncontrol.ApprovalScopeMissionStep,
		State:           missioncontrol.ApprovalStatePending,
		RequestedAt:     now,
	})
	state.runtimeState.ApprovalRequests = append(state.runtimeState.ApprovalRequests, missioncontrol.ApprovalRequest{
		JobID:           "job-1",
		StepID:          "build",
		RequestedAction: missioncontrol.ApprovalRequestedActionStepComplete,
		Scope:           missioncontrol.ApprovalScopeMissionStep,
		State:           missioncontrol.ApprovalStatePending,
		RequestedAt:     now,
	})
	state.executionContext.Runtime.ApprovalRequests[0].State = missioncontrol.ApprovalStateSuperseded
	state.executionContext.Runtime.ApprovalRequests[0].SupersededAt = now
	state.runtimeState.ApprovalRequests[0].State = missioncontrol.ApprovalStateSuperseded
	state.runtimeState.ApprovalRequests[0].SupersededAt = now
	state.mu.Unlock()

	handled, resp, err := state.ApplyNaturalApprovalDecision("yes")
	if err != nil {
		t.Fatalf("ApplyNaturalApprovalDecision(yes) error = %v", err)
	}
	if !handled {
		t.Fatal("ApplyNaturalApprovalDecision(yes) handled = false, want true")
	}
	if resp != "Approved job=job-1 step=build." {
		t.Fatalf("ApplyNaturalApprovalDecision(yes) response = %q, want approval acknowledgement", resp)
	}

	runtime, ok := state.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if len(runtime.ApprovalRequests) != 2 || runtime.ApprovalRequests[0].State != missioncontrol.ApprovalStateSuperseded {
		t.Fatalf("MissionRuntimeState().ApprovalRequests = %#v, want leading superseded request", runtime.ApprovalRequests)
	}
}

func TestTaskStateApplyApprovalDecisionBindsOnlyLatestValidRequest(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	job := testTaskStateJob()
	job.Plan.Steps[0] = missioncontrol.Step{
		ID:      "build",
		Type:    missioncontrol.StepTypeDiscussion,
		Subtype: missioncontrol.StepSubtypeAuthorization,
	}

	if err := state.ActivateStep(job, "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}
	if err := state.ApplyStepOutput("Waiting for approval.", nil); err != nil {
		t.Fatalf("ApplyStepOutput() error = %v", err)
	}

	now := time.Now()
	if state.executionContext.Runtime.ApprovalRequests[0].ExpiresAt.IsZero() {
		t.Fatalf("executionContext approval request = %#v, want stamped expires_at", state.executionContext.Runtime.ApprovalRequests)
	}
	if state.runtimeState.ApprovalRequests[0].ExpiresAt.IsZero() {
		t.Fatalf("runtimeState approval request = %#v, want stamped expires_at", state.runtimeState.ApprovalRequests)
	}
	state.mu.Lock()
	state.executionContext.Runtime.ApprovalRequests[0].State = missioncontrol.ApprovalStateSuperseded
	state.executionContext.Runtime.ApprovalRequests[0].SupersededAt = now
	state.runtimeState.ApprovalRequests[0].State = missioncontrol.ApprovalStateSuperseded
	state.runtimeState.ApprovalRequests[0].SupersededAt = now
	newRequest := missioncontrol.ApprovalRequest{
		JobID:           "job-1",
		StepID:          "build",
		RequestedAction: missioncontrol.ApprovalRequestedActionStepComplete,
		Scope:           missioncontrol.ApprovalScopeMissionStep,
		State:           missioncontrol.ApprovalStatePending,
		RequestedAt:     now.Add(time.Second),
	}
	state.executionContext.Runtime.ApprovalRequests = append(state.executionContext.Runtime.ApprovalRequests, newRequest)
	state.runtimeState.ApprovalRequests = append(state.runtimeState.ApprovalRequests, newRequest)
	state.mu.Unlock()

	if err := state.ApplyApprovalDecision("job-1", "build", missioncontrol.ApprovalDecisionApprove, missioncontrol.ApprovalGrantedViaOperatorCommand); err != nil {
		t.Fatalf("ApplyApprovalDecision() error = %v", err)
	}

	runtime, ok := state.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if len(runtime.ApprovalRequests) != 2 {
		t.Fatalf("len(ApprovalRequests) = %d, want 2", len(runtime.ApprovalRequests))
	}
	if runtime.ApprovalRequests[0].State != missioncontrol.ApprovalStateSuperseded {
		t.Fatalf("ApprovalRequests[0].State = %q, want %q", runtime.ApprovalRequests[0].State, missioncontrol.ApprovalStateSuperseded)
	}
	if runtime.ApprovalRequests[1].State != missioncontrol.ApprovalStateGranted {
		t.Fatalf("ApprovalRequests[1].State = %q, want %q", runtime.ApprovalRequests[1].State, missioncontrol.ApprovalStateGranted)
	}
}

func TestTaskStateApplyNaturalApprovalDecisionDoesNotBindWrongStep(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	job := testTaskStateJob()
	job.Plan.Steps[0] = missioncontrol.Step{
		ID:      "build",
		Type:    missioncontrol.StepTypeDiscussion,
		Subtype: missioncontrol.StepSubtypeAuthorization,
	}

	if err := state.ActivateStep(job, "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}
	if err := state.ApplyStepOutput("Waiting for approval.", nil); err != nil {
		t.Fatalf("ApplyStepOutput() error = %v", err)
	}

	state.mu.Lock()
	state.executionContext.Runtime.ApprovalRequests[0].StepID = "other-step"
	state.runtimeState.ApprovalRequests[0].StepID = "other-step"
	state.mu.Unlock()

	handled, _, err := state.ApplyNaturalApprovalDecision("yes")
	if err == nil {
		t.Fatal("ApplyNaturalApprovalDecision(yes) error = nil, want mismatch failure")
	}
	if !handled {
		t.Fatal("ApplyNaturalApprovalDecision(yes) handled = false, want true")
	}

	runtime, ok := state.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if runtime.State != missioncontrol.JobStateWaitingUser {
		t.Fatalf("MissionRuntimeState().State = %q, want %q", runtime.State, missioncontrol.JobStateWaitingUser)
	}
	if len(runtime.CompletedSteps) != 0 {
		t.Fatalf("MissionRuntimeState().CompletedSteps = %#v, want empty", runtime.CompletedSteps)
	}
}

func TestTaskStateApplyNaturalApprovalDecisionRejectsTerminalPersistedRuntime(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	if err := state.HydrateRuntimeControl(missioncontrol.Job{ID: "job-1"}, missioncontrol.JobRuntimeState{
		JobID: "job-1",
		State: missioncontrol.JobStateCompleted,
	}, nil); err != nil {
		t.Fatalf("HydrateRuntimeControl() error = %v", err)
	}

	handled, _, err := state.ApplyNaturalApprovalDecision("yes")
	if err == nil {
		t.Fatal("ApplyNaturalApprovalDecision(yes) error = nil, want terminal-state rejection")
	}
	if !handled {
		t.Fatal("ApplyNaturalApprovalDecision(yes) handled = false, want true")
	}
}

func TestTaskStateHydrateRuntimeControlExpiresElapsedApprovalImmediately(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	hookCalls := 0
	state.SetRuntimeChangeHook(func() {
		hookCalls++
	})

	expiredAt := time.Now().Add(-1 * time.Minute)
	if err := state.HydrateRuntimeControl(missioncontrol.Job{ID: "job-1"}, missioncontrol.JobRuntimeState{
		JobID:         "job-1",
		State:         missioncontrol.JobStateWaitingUser,
		ActiveStepID:  "build",
		WaitingReason: "awaiting operator input",
		ApprovalRequests: []missioncontrol.ApprovalRequest{
			{
				JobID:           "job-1",
				StepID:          "build",
				RequestedAction: missioncontrol.ApprovalRequestedActionStepComplete,
				Scope:           missioncontrol.ApprovalScopeMissionStep,
				RequestedVia:    missioncontrol.ApprovalRequestedViaRuntime,
				State:           missioncontrol.ApprovalStatePending,
				RequestedAt:     expiredAt.Add(-1 * time.Minute),
				ExpiresAt:       expiredAt,
			},
		},
	}, &missioncontrol.RuntimeControlContext{
		JobID: "job-1",
		Step: missioncontrol.Step{
			ID:      "build",
			Type:    missioncontrol.StepTypeDiscussion,
			Subtype: missioncontrol.StepSubtypeAuthorization,
		},
	}); err != nil {
		t.Fatalf("HydrateRuntimeControl() error = %v", err)
	}

	runtime, ok := state.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if len(runtime.ApprovalRequests) != 1 || runtime.ApprovalRequests[0].State != missioncontrol.ApprovalStateExpired {
		t.Fatalf("MissionRuntimeState().ApprovalRequests = %#v, want one expired approval", runtime.ApprovalRequests)
	}
	if runtime.ApprovalRequests[0].ResolvedAt != expiredAt {
		t.Fatalf("MissionRuntimeState().ApprovalRequests[0].ResolvedAt = %v, want %v", runtime.ApprovalRequests[0].ResolvedAt, expiredAt)
	}
	if hookCalls != 1 {
		t.Fatalf("runtime change hook calls = %d, want 1", hookCalls)
	}
}

func TestTaskStateHydrateRuntimeControlLeavesTerminalRuntimeUnchanged(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	hookCalls := 0
	state.SetRuntimeChangeHook(func() {
		hookCalls++
	})

	expiredAt := time.Now().Add(-1 * time.Minute)
	if err := state.HydrateRuntimeControl(missioncontrol.Job{ID: "job-1"}, missioncontrol.JobRuntimeState{
		JobID: "job-1",
		State: missioncontrol.JobStateCompleted,
		ApprovalRequests: []missioncontrol.ApprovalRequest{
			{
				JobID:           "job-1",
				StepID:          "build",
				RequestedAction: missioncontrol.ApprovalRequestedActionStepComplete,
				Scope:           missioncontrol.ApprovalScopeMissionStep,
				State:           missioncontrol.ApprovalStatePending,
				ExpiresAt:       expiredAt,
			},
		},
	}, nil); err != nil {
		t.Fatalf("HydrateRuntimeControl() error = %v", err)
	}

	runtime, ok := state.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if len(runtime.ApprovalRequests) != 1 || runtime.ApprovalRequests[0].State != missioncontrol.ApprovalStatePending {
		t.Fatalf("MissionRuntimeState().ApprovalRequests = %#v, want unchanged terminal approval request", runtime.ApprovalRequests)
	}
	if hookCalls != 0 {
		t.Fatalf("runtime change hook calls = %d, want 0", hookCalls)
	}
}

func TestTaskStateEmitAuditEventPersistsIntoRuntimeHistoryAndTruncatesDeterministically(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	if err := state.ActivateStep(testTaskStateJob(), "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}

	total := missioncontrol.AuditHistoryCap + 2
	for i := 0; i < total; i++ {
		state.EmitAuditEvent(missioncontrol.AuditEvent{
			JobID:     "job-1",
			StepID:    "build",
			ToolName:  fmt.Sprintf("command-%02d", i),
			Allowed:   true,
			Timestamp: time.Date(2026, 3, 24, 12, 0, i, 0, time.UTC),
		})
	}

	audits := state.AuditEvents()
	if len(audits) != missioncontrol.AuditHistoryCap {
		t.Fatalf("AuditEvents() count = %d, want %d", len(audits), missioncontrol.AuditHistoryCap)
	}

	runtime, ok := state.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if len(runtime.AuditHistory) != missioncontrol.AuditHistoryCap {
		t.Fatalf("MissionRuntimeState().AuditHistory count = %d, want %d", len(runtime.AuditHistory), missioncontrol.AuditHistoryCap)
	}

	for i := 0; i < missioncontrol.AuditHistoryCap; i++ {
		want := fmt.Sprintf("command-%02d", i+2)
		if audits[i].ToolName != want {
			t.Fatalf("AuditEvents()[%d].ToolName = %q, want %q", i, audits[i].ToolName, want)
		}
		if audits[i].EventID == "" {
			t.Fatalf("AuditEvents()[%d].EventID = empty, want deterministic id", i)
		}
		if audits[i].ActionClass != missioncontrol.AuditActionClassToolCall {
			t.Fatalf("AuditEvents()[%d].ActionClass = %q, want %q", i, audits[i].ActionClass, missioncontrol.AuditActionClassToolCall)
		}
		if audits[i].Result != missioncontrol.AuditResultAllowed {
			t.Fatalf("AuditEvents()[%d].Result = %q, want %q", i, audits[i].Result, missioncontrol.AuditResultAllowed)
		}
		if runtime.AuditHistory[i].ToolName != want {
			t.Fatalf("MissionRuntimeState().AuditHistory[%d].ToolName = %q, want %q", i, runtime.AuditHistory[i].ToolName, want)
		}
	}
}

func TestTaskStateHydrateRuntimeControlRestoresAuditHistoryWithoutDuplication(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	job := testTaskStateJob()
	runtime := missioncontrol.JobRuntimeState{
		JobID:        "job-1",
		State:        missioncontrol.JobStatePaused,
		ActiveStepID: "build",
		PausedReason: missioncontrol.RuntimePauseReasonOperatorCommand,
		AuditHistory: []missioncontrol.AuditEvent{
			{
				JobID:     "job-1",
				StepID:    "build",
				ToolName:  "pause",
				Allowed:   true,
				Timestamp: time.Date(2026, 3, 24, 12, 0, 0, 0, time.UTC),
			},
		},
	}

	if err := state.HydrateRuntimeControl(job, runtime, nil); err != nil {
		t.Fatalf("HydrateRuntimeControl() error = %v", err)
	}
	state.ClearExecutionContext()
	if err := state.HydrateRuntimeControl(job, runtime, nil); err != nil {
		t.Fatalf("HydrateRuntimeControl() second error = %v", err)
	}

	audits := state.AuditEvents()
	if len(audits) != 1 {
		t.Fatalf("AuditEvents() count = %d, want 1", len(audits))
	}
	if audits[0].ToolName != "pause" {
		t.Fatalf("AuditEvents()[0].ToolName = %q, want %q", audits[0].ToolName, "pause")
	}

	gotRuntime, ok := state.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if len(gotRuntime.AuditHistory) != 1 {
		t.Fatalf("MissionRuntimeState().AuditHistory count = %d, want 1", len(gotRuntime.AuditHistory))
	}
	expectedAudit := missioncontrol.AppendAuditHistory(nil, missioncontrol.AuditEvent{
		JobID:     "job-1",
		StepID:    "build",
		ToolName:  "pause",
		Allowed:   true,
		Timestamp: time.Date(2026, 3, 24, 12, 0, 0, 0, time.UTC),
	})[0]
	if gotRuntime.AuditHistory[0] != expectedAudit {
		t.Fatalf("MissionRuntimeState().AuditHistory[0] = %#v, want %#v", gotRuntime.AuditHistory[0], expectedAudit)
	}
}

func TestTaskStateHydrateRuntimeControlNormalizesLegacyRevokedAtOnlyOnce(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	hookCalls := 0
	state.SetRuntimeChangeHook(func() {
		hookCalls++
	})

	job := testTaskStateJob()
	revokedAt := time.Date(2026, 3, 24, 12, 1, 0, 0, time.UTC)
	runtime := missioncontrol.JobRuntimeState{
		JobID:        "job-1",
		State:        missioncontrol.JobStatePaused,
		ActiveStepID: "build",
		ApprovalRequests: []missioncontrol.ApprovalRequest{
			{
				JobID:           "job-1",
				StepID:          "build",
				RequestedAction: missioncontrol.ApprovalRequestedActionStepComplete,
				Scope:           missioncontrol.ApprovalScopeOneJob,
				RequestedVia:    missioncontrol.ApprovalRequestedViaRuntime,
				GrantedVia:      missioncontrol.ApprovalGrantedViaOperatorCommand,
				State:           missioncontrol.ApprovalStateRevoked,
			},
		},
		ApprovalGrants: []missioncontrol.ApprovalGrant{
			{
				JobID:           "job-1",
				StepID:          "build",
				RequestedAction: missioncontrol.ApprovalRequestedActionStepComplete,
				Scope:           missioncontrol.ApprovalScopeOneJob,
				GrantedVia:      missioncontrol.ApprovalGrantedViaOperatorCommand,
				State:           missioncontrol.ApprovalStateRevoked,
				RevokedAt:       revokedAt,
			},
		},
	}

	if err := state.HydrateRuntimeControl(job, runtime, nil); err != nil {
		t.Fatalf("HydrateRuntimeControl() error = %v", err)
	}
	gotRuntime, ok := state.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if gotRuntime.ApprovalRequests[0].RevokedAt != revokedAt {
		t.Fatalf("MissionRuntimeState().ApprovalRequests[0].RevokedAt = %v, want %v", gotRuntime.ApprovalRequests[0].RevokedAt, revokedAt)
	}
	if hookCalls != 1 {
		t.Fatalf("runtime change hook calls after first hydrate = %d, want 1", hookCalls)
	}

	state.ClearExecutionContext()
	if hookCalls != 2 {
		t.Fatalf("runtime change hook calls after ClearExecutionContext() = %d, want 2", hookCalls)
	}
	hookCallsAfterClear := hookCalls
	if err := state.HydrateRuntimeControl(job, gotRuntime, nil); err != nil {
		t.Fatalf("HydrateRuntimeControl() second error = %v", err)
	}
	if hookCalls != hookCallsAfterClear {
		t.Fatalf("runtime change hook calls after second hydrate = %d, want unchanged %d", hookCalls, hookCallsAfterClear)
	}
}

func TestTaskStateHydrateRuntimeControlApplyApprovalDecisionRejectsExpiredRequest(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	expiredAt := time.Now().Add(-1 * time.Minute)
	if err := state.HydrateRuntimeControl(missioncontrol.Job{ID: "job-1"}, missioncontrol.JobRuntimeState{
		JobID:         "job-1",
		State:         missioncontrol.JobStateWaitingUser,
		ActiveStepID:  "build",
		WaitingReason: "awaiting operator input",
		ApprovalRequests: []missioncontrol.ApprovalRequest{
			{
				JobID:           "job-1",
				StepID:          "build",
				RequestedAction: missioncontrol.ApprovalRequestedActionStepComplete,
				Scope:           missioncontrol.ApprovalScopeMissionStep,
				RequestedVia:    missioncontrol.ApprovalRequestedViaRuntime,
				State:           missioncontrol.ApprovalStatePending,
				ExpiresAt:       expiredAt,
			},
		},
	}, &missioncontrol.RuntimeControlContext{
		JobID: "job-1",
		Step: missioncontrol.Step{
			ID:      "build",
			Type:    missioncontrol.StepTypeDiscussion,
			Subtype: missioncontrol.StepSubtypeAuthorization,
		},
	}); err != nil {
		t.Fatalf("HydrateRuntimeControl() error = %v", err)
	}

	err := state.ApplyApprovalDecision("job-1", "build", missioncontrol.ApprovalDecisionApprove, missioncontrol.ApprovalGrantedViaOperatorCommand)
	if err == nil {
		t.Fatal("ApplyApprovalDecision() error = nil, want expired approval failure")
	}

	runtime, ok := state.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if len(runtime.ApprovalRequests) != 1 || runtime.ApprovalRequests[0].State != missioncontrol.ApprovalStateExpired {
		t.Fatalf("MissionRuntimeState().ApprovalRequests = %#v, want one expired approval", runtime.ApprovalRequests)
	}
}

func TestTaskStateApplyApprovalDecisionDenyAfterExecutionContextTeardownBlocksLaterFreeFormInput(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	job := testTaskStateJob()
	job.Plan.Steps[0] = missioncontrol.Step{
		ID:      "build",
		Type:    missioncontrol.StepTypeDiscussion,
		Subtype: missioncontrol.StepSubtypeAuthorization,
	}
	job.Plan.Steps[1] = missioncontrol.Step{
		ID:        "final",
		Type:      missioncontrol.StepTypeFinalResponse,
		DependsOn: []string{"build"},
	}

	if err := state.ActivateStep(job, "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}
	if err := state.ApplyStepOutput("Waiting for approval.", nil); err != nil {
		t.Fatalf("ApplyStepOutput() error = %v", err)
	}

	state.ClearExecutionContext()

	if err := state.ApplyApprovalDecision("job-1", "build", missioncontrol.ApprovalDecisionDeny, missioncontrol.ApprovalGrantedViaOperatorCommand); err != nil {
		t.Fatalf("ApplyApprovalDecision() error = %v", err)
	}

	inputKind, err := state.ApplyWaitingUserInput("approved")
	if err != nil {
		t.Fatalf("ApplyWaitingUserInput() error = %v", err)
	}
	if inputKind != missioncontrol.WaitingUserInputNone {
		t.Fatalf("ApplyWaitingUserInput() kind = %q, want %q without execution context", inputKind, missioncontrol.WaitingUserInputNone)
	}

	runtime, ok := state.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if runtime.State != missioncontrol.JobStateWaitingUser {
		t.Fatalf("MissionRuntimeState().State = %q, want %q", runtime.State, missioncontrol.JobStateWaitingUser)
	}
	if len(runtime.CompletedSteps) != 0 {
		t.Fatalf("MissionRuntimeState().CompletedSteps = %#v, want empty", runtime.CompletedSteps)
	}
	if len(runtime.ApprovalRequests) != 1 || runtime.ApprovalRequests[0].State != missioncontrol.ApprovalStateDenied {
		t.Fatalf("MissionRuntimeState().ApprovalRequests = %#v, want one denied approval", runtime.ApprovalRequests)
	}
}

func TestTaskStateApplyWaitingUserInputDoesNotBindPendingApproval(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	job := testTaskStateJob()
	job.Plan.Steps[0] = missioncontrol.Step{
		ID:      "build",
		Type:    missioncontrol.StepTypeDiscussion,
		Subtype: missioncontrol.StepSubtypeAuthorization,
	}
	job.Plan.Steps[1] = missioncontrol.Step{
		ID:        "final",
		Type:      missioncontrol.StepTypeFinalResponse,
		DependsOn: []string{"build"},
	}

	if err := state.ActivateStep(job, "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}
	if err := state.ApplyStepOutput("Waiting for approval.", nil); err != nil {
		t.Fatalf("ApplyStepOutput() error = %v", err)
	}

	_, err := state.ApplyWaitingUserInput("approved")
	if err == nil {
		t.Fatal("ApplyWaitingUserInput() error = nil, want explicit operator approval failure")
	}

	ec, ok := state.ExecutionContext()
	if !ok {
		t.Fatal("ExecutionContext() ok = false, want waiting execution context")
	}
	if ec.Runtime == nil || ec.Runtime.State != missioncontrol.JobStateWaitingUser {
		t.Fatalf("ExecutionContext().Runtime = %#v, want waiting_user runtime", ec.Runtime)
	}
	if len(ec.Runtime.ApprovalRequests) != 1 || ec.Runtime.ApprovalRequests[0].State != missioncontrol.ApprovalStatePending {
		t.Fatalf("ExecutionContext().Runtime.ApprovalRequests = %#v, want one pending approval", ec.Runtime.ApprovalRequests)
	}
}

func TestTaskStateApplyApprovalDecisionWrongBindingAfterExecutionContextTeardownDoesNotMutateRuntime(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	job := testTaskStateJob()
	job.Plan.Steps[0] = missioncontrol.Step{
		ID:      "build",
		Type:    missioncontrol.StepTypeDiscussion,
		Subtype: missioncontrol.StepSubtypeAuthorization,
	}
	job.Plan.Steps[1] = missioncontrol.Step{
		ID:        "final",
		Type:      missioncontrol.StepTypeFinalResponse,
		DependsOn: []string{"build"},
	}

	if err := state.ActivateStep(job, "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}
	if err := state.ApplyStepOutput("Waiting for approval.", nil); err != nil {
		t.Fatalf("ApplyStepOutput() error = %v", err)
	}

	state.ClearExecutionContext()
	state.SetOperatorSession("telegram", "chat-99")

	for _, tc := range []struct {
		name   string
		jobID  string
		stepID string
	}{
		{name: "wrong job", jobID: "other-job", stepID: "build"},
		{name: "wrong step", jobID: "job-1", stepID: "other-step"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := state.ApplyApprovalDecision(tc.jobID, tc.stepID, missioncontrol.ApprovalDecisionApprove, missioncontrol.ApprovalGrantedViaOperatorCommand)
			if err == nil {
				t.Fatal("ApplyApprovalDecision() error = nil, want mismatch failure")
			}
		})
	}

	runtime, ok := state.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if runtime.State != missioncontrol.JobStateWaitingUser {
		t.Fatalf("MissionRuntimeState().State = %q, want %q", runtime.State, missioncontrol.JobStateWaitingUser)
	}
	if len(runtime.ApprovalRequests) != 1 || runtime.ApprovalRequests[0].State != missioncontrol.ApprovalStatePending {
		t.Fatalf("MissionRuntimeState().ApprovalRequests = %#v, want one pending approval", runtime.ApprovalRequests)
	}
	if runtime.ApprovalRequests[0].SessionChannel != "" || runtime.ApprovalRequests[0].SessionChatID != "" {
		t.Fatalf("MissionRuntimeState().ApprovalRequests[0] session = (%q, %q), want empty session on non-binding mismatch", runtime.ApprovalRequests[0].SessionChannel, runtime.ApprovalRequests[0].SessionChatID)
	}
}

func TestTaskStateApplyWaitingUserInputDoesNotCompleteDeniedApproval(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	job := testTaskStateJob()
	job.Plan.Steps[0] = missioncontrol.Step{
		ID:      "build",
		Type:    missioncontrol.StepTypeDiscussion,
		Subtype: missioncontrol.StepSubtypeAuthorization,
	}
	job.Plan.Steps[1] = missioncontrol.Step{
		ID:        "final",
		Type:      missioncontrol.StepTypeFinalResponse,
		DependsOn: []string{"build"},
	}

	if err := state.ActivateStep(job, "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}
	if err := state.ApplyStepOutput("Waiting for approval.", nil); err != nil {
		t.Fatalf("ApplyStepOutput() error = %v", err)
	}
	if err := state.ApplyApprovalDecision("job-1", "build", missioncontrol.ApprovalDecisionDeny, missioncontrol.ApprovalGrantedViaOperatorCommand); err != nil {
		t.Fatalf("ApplyApprovalDecision() error = %v", err)
	}

	_, err := state.ApplyWaitingUserInput("go ahead")
	if err == nil {
		t.Fatal("ApplyWaitingUserInput() error = nil, want denied approval failure")
	}

	ec, ok := state.ExecutionContext()
	if !ok {
		t.Fatal("ExecutionContext() ok = false, want waiting execution context")
	}
	if ec.Runtime == nil || ec.Runtime.State != missioncontrol.JobStateWaitingUser {
		t.Fatalf("ExecutionContext().Runtime = %#v, want waiting_user runtime", ec.Runtime)
	}
	if len(ec.Runtime.CompletedSteps) != 0 {
		t.Fatalf("ExecutionContext().Runtime.CompletedSteps = %#v, want empty", ec.Runtime.CompletedSteps)
	}
	if len(ec.Runtime.ApprovalRequests) != 1 || ec.Runtime.ApprovalRequests[0].State != missioncontrol.ApprovalStateDenied {
		t.Fatalf("ExecutionContext().Runtime.ApprovalRequests = %#v, want one denied approval", ec.Runtime.ApprovalRequests)
	}
}

func TestTaskStateOperatorInspectWithoutValidatedPlanReturnsDeterministicError(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	control, err := missioncontrol.BuildRuntimeControlContext(testTaskStateJob(), "build")
	if err != nil {
		t.Fatalf("BuildRuntimeControlContext() error = %v", err)
	}

	state.runtimeState = missioncontrol.JobRuntimeState{
		JobID:        "job-1",
		State:        missioncontrol.JobStatePaused,
		ActiveStepID: "build",
		PausedReason: missioncontrol.RuntimePauseReasonOperatorCommand,
	}
	state.hasRuntimeState = true
	state.runtimeControl = control
	state.hasRuntimeControl = true

	_, err = state.OperatorInspect("job-1", "build")
	if err == nil {
		t.Fatal("OperatorInspect() error = nil, want missing-plan failure")
	}
	if !strings.Contains(err.Error(), string(missioncontrol.RejectionCodeInvalidRuntimeState)) {
		t.Fatalf("OperatorInspect() error = %q, want invalid_runtime_state code", err)
	}
	if !strings.Contains(err.Error(), "inspect command requires validated mission plan") {
		t.Fatalf("OperatorInspect() error = %q, want missing validated plan message", err)
	}
}

func TestTaskStateOperatorInspectActiveExecutionContextPreservesValidatedPlanBehavior(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	if err := state.ActivateStep(testTaskStateJob(), "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}

	got, err := state.OperatorInspect("job-1", "final")
	if err != nil {
		t.Fatalf("OperatorInspect() error = %v", err)
	}

	var summary missioncontrol.InspectSummary
	if err := json.Unmarshal([]byte(got), &summary); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if len(summary.Steps) != 1 || summary.Steps[0].StepID != "final" {
		t.Fatalf("Steps = %#v, want one final step", summary.Steps)
	}
}

func TestTaskStateOperatorInspectUsesPersistedInspectablePlanWithoutMissionJob(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	job := testTaskStateJob()
	inspectablePlan, err := missioncontrol.BuildInspectablePlanContext(job)
	if err != nil {
		t.Fatalf("BuildInspectablePlanContext() error = %v", err)
	}
	control, err := missioncontrol.BuildRuntimeControlContext(job, "build")
	if err != nil {
		t.Fatalf("BuildRuntimeControlContext() error = %v", err)
	}

	state.runtimeState = missioncontrol.JobRuntimeState{
		JobID:           "job-1",
		State:           missioncontrol.JobStatePaused,
		ActiveStepID:    "build",
		InspectablePlan: &inspectablePlan,
		PausedReason:    missioncontrol.RuntimePauseReasonOperatorCommand,
	}
	state.hasRuntimeState = true
	state.runtimeControl = control
	state.hasRuntimeControl = true

	got, err := state.OperatorInspect("job-1", "final")
	if err != nil {
		t.Fatalf("OperatorInspect() error = %v", err)
	}

	var summary missioncontrol.InspectSummary
	if err := json.Unmarshal([]byte(got), &summary); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if summary.JobID != "job-1" {
		t.Fatalf("JobID = %q, want %q", summary.JobID, "job-1")
	}
	if len(summary.Steps) != 1 || summary.Steps[0].StepID != "final" {
		t.Fatalf("Steps = %#v, want one final step", summary.Steps)
	}
	if !reflect.DeepEqual(summary.Steps[0].EffectiveAllowedTools, []string{"read"}) {
		t.Fatalf("EffectiveAllowedTools = %#v, want %#v", summary.Steps[0].EffectiveAllowedTools, []string{"read"})
	}
}

func TestTaskStateOperatorInspectPersistedInspectablePlanWrongJobDoesNotBind(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	inspectablePlan, err := missioncontrol.BuildInspectablePlanContext(testTaskStateJob())
	if err != nil {
		t.Fatalf("BuildInspectablePlanContext() error = %v", err)
	}

	state.runtimeState = missioncontrol.JobRuntimeState{
		JobID:           "job-1",
		State:           missioncontrol.JobStatePaused,
		ActiveStepID:    "build",
		InspectablePlan: &inspectablePlan,
		PausedReason:    missioncontrol.RuntimePauseReasonOperatorCommand,
	}
	state.hasRuntimeState = true

	_, err = state.OperatorInspect("other-job", "final")
	if err == nil {
		t.Fatal("OperatorInspect() error = nil, want mismatch failure")
	}
	if !strings.Contains(err.Error(), "does not match the active job") {
		t.Fatalf("OperatorInspect() error = %q, want job mismatch", err)
	}
}

func TestTaskStateOperatorInspectPersistedInspectablePlanRejectsInvalidStep(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	inspectablePlan, err := missioncontrol.BuildInspectablePlanContext(testTaskStateJob())
	if err != nil {
		t.Fatalf("BuildInspectablePlanContext() error = %v", err)
	}

	state.runtimeState = missioncontrol.JobRuntimeState{
		JobID:           "job-1",
		State:           missioncontrol.JobStatePaused,
		ActiveStepID:    "build",
		InspectablePlan: &inspectablePlan,
		PausedReason:    missioncontrol.RuntimePauseReasonOperatorCommand,
	}
	state.hasRuntimeState = true

	_, err = state.OperatorInspect("job-1", "missing")
	if err == nil {
		t.Fatal("OperatorInspect() error = nil, want unknown-step failure")
	}
	if !strings.Contains(err.Error(), string(missioncontrol.RejectionCodeUnknownStep)) {
		t.Fatalf("OperatorInspect() error = %q, want unknown_step code", err)
	}
	if !strings.Contains(err.Error(), `step "missing" not found in plan`) {
		t.Fatalf("OperatorInspect() error = %q, want missing-step message", err)
	}
}

func TestTaskStateOperatorInspectTerminalRuntimeUsesPersistedInspectablePlanWithoutMissionJob(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	inspectablePlan, err := missioncontrol.BuildInspectablePlanContext(testTaskStateJob())
	if err != nil {
		t.Fatalf("BuildInspectablePlanContext() error = %v", err)
	}

	state.runtimeState = missioncontrol.JobRuntimeState{
		JobID:           "job-1",
		State:           missioncontrol.JobStateCompleted,
		InspectablePlan: &inspectablePlan,
		CompletedAt:     time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC),
	}
	state.hasRuntimeState = true

	got, err := state.OperatorInspect("job-1", "final")
	if err != nil {
		t.Fatalf("OperatorInspect() error = %v", err)
	}

	var summary missioncontrol.InspectSummary
	if err := json.Unmarshal([]byte(got), &summary); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if len(summary.Steps) != 1 || summary.Steps[0].StepID != "final" {
		t.Fatalf("Steps = %#v, want one final step", summary.Steps)
	}
}

func testTaskStateJob() missioncontrol.Job {
	return missioncontrol.Job{
		ID:           "job-1",
		MaxAuthority: missioncontrol.AuthorityTierHigh,
		AllowedTools: []string{"read", "write"},
		Plan: missioncontrol.Plan{
			ID: "plan-1",
			Steps: []missioncontrol.Step{
				{
					ID:                "build",
					Type:              missioncontrol.StepTypeOneShotCode,
					RequiredAuthority: missioncontrol.AuthorityTierMedium,
					AllowedTools:      []string{"read"},
					SuccessCriteria:   []string{"produce code"},
				},
				{
					ID:           "final",
					Type:         missioncontrol.StepTypeFinalResponse,
					DependsOn:    []string{"build"},
					AllowedTools: []string{"read"},
				},
			},
		},
	}
}

func testReusableApprovalJob(scope string) missioncontrol.Job {
	return missioncontrol.Job{
		ID:           "job-1",
		MaxAuthority: missioncontrol.AuthorityTierHigh,
		AllowedTools: []string{"read"},
		Plan: missioncontrol.Plan{
			ID: "plan-1",
			Steps: []missioncontrol.Step{
				{
					ID:            "authorize-1",
					Type:          missioncontrol.StepTypeDiscussion,
					Subtype:       missioncontrol.StepSubtypeAuthorization,
					ApprovalScope: scope,
				},
				{
					ID:            "authorize-2",
					Type:          missioncontrol.StepTypeDiscussion,
					Subtype:       missioncontrol.StepSubtypeAuthorization,
					ApprovalScope: scope,
					DependsOn:     []string{"authorize-1"},
				},
				{
					ID:        "final",
					Type:      missioncontrol.StepTypeFinalResponse,
					DependsOn: []string{"authorize-2"},
				},
			},
		},
	}
}
