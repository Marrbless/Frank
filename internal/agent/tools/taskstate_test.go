package tools

import (
	"reflect"
	"testing"

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

func TestTaskStateClearExecutionContextRemovesActiveContext(t *testing.T) {
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
	if _, ok := state.MissionRuntimeState(); ok {
		t.Fatal("MissionRuntimeState() ok = true, want false after clear")
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

	if err := state.ResumeRuntime(job, runtimeState, true); err != nil {
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

func TestTaskStateApplyWaitingUserInputPausesCompletedStep(t *testing.T) {
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

	inputKind, err := state.ApplyWaitingUserInput("approved")
	if err != nil {
		t.Fatalf("ApplyWaitingUserInput() error = %v", err)
	}
	if inputKind != missioncontrol.WaitingUserInputApproval {
		t.Fatalf("ApplyWaitingUserInput() kind = %q, want %q", inputKind, missioncontrol.WaitingUserInputApproval)
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
