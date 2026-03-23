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
