package missioncontrol

import (
	"reflect"
	"testing"
)

func TestCloneExecutionContextDeepCopiesNestedData(t *testing.T) {
	t.Parallel()

	original := ExecutionContext{
		Job: &Job{
			ID:           "job-1",
			MaxAuthority: AuthorityTierHigh,
			AllowedTools: []string{"read", "write"},
			Plan: Plan{
				ID: "plan-1",
				Steps: []Step{
					{
						ID:                "build",
						Type:              StepTypeOneShotCode,
						DependsOn:         []string{"prep"},
						RequiredAuthority: AuthorityTierMedium,
						AllowedTools:      []string{"read"},
						RequiresApproval:  true,
						SuccessCriteria:   []string{"produce code"},
					},
				},
			},
		},
		Step: &Step{
			ID:                "build",
			Type:              StepTypeOneShotCode,
			DependsOn:         []string{"prep"},
			RequiredAuthority: AuthorityTierMedium,
			AllowedTools:      []string{"read"},
			RequiresApproval:  true,
			SuccessCriteria:   []string{"produce code"},
		},
	}

	cloned := CloneExecutionContext(original)
	if !reflect.DeepEqual(cloned, original) {
		t.Fatalf("CloneExecutionContext() = %#v, want %#v", cloned, original)
	}

	cloned.Job.AllowedTools[0] = "mutated-job-tool"
	cloned.Job.Plan.Steps[0].DependsOn[0] = "mutated-dependency"
	cloned.Job.Plan.Steps[0].AllowedTools[0] = "mutated-step-tool"
	cloned.Step.SuccessCriteria[0] = "mutated-success"

	if original.Job.AllowedTools[0] != "read" {
		t.Fatalf("original Job.AllowedTools[0] = %q, want %q", original.Job.AllowedTools[0], "read")
	}

	if original.Job.Plan.Steps[0].DependsOn[0] != "prep" {
		t.Fatalf("original Job.Plan.Steps[0].DependsOn[0] = %q, want %q", original.Job.Plan.Steps[0].DependsOn[0], "prep")
	}

	if original.Job.Plan.Steps[0].AllowedTools[0] != "read" {
		t.Fatalf("original Job.Plan.Steps[0].AllowedTools[0] = %q, want %q", original.Job.Plan.Steps[0].AllowedTools[0], "read")
	}

	if original.Step.SuccessCriteria[0] != "produce code" {
		t.Fatalf("original Step.SuccessCriteria[0] = %q, want %q", original.Step.SuccessCriteria[0], "produce code")
	}
}

func TestCloneExecutionContextZeroValue(t *testing.T) {
	t.Parallel()

	if got := CloneExecutionContext(ExecutionContext{}); !reflect.DeepEqual(got, ExecutionContext{}) {
		t.Fatalf("CloneExecutionContext() = %#v, want zero value", got)
	}
}

func TestResolveExecutionContextValidPlan(t *testing.T) {
	t.Parallel()

	job := testExecutionJob()

	ec, err := ResolveExecutionContext(job, "build")
	if err != nil {
		t.Fatalf("ResolveExecutionContext() error = %v", err)
	}

	if ec.Job == nil {
		t.Fatal("ResolveExecutionContext().Job = nil, want non-nil")
	}

	if ec.Step == nil {
		t.Fatal("ResolveExecutionContext().Step = nil, want non-nil")
	}

	if ec.Job.ID != job.ID {
		t.Fatalf("ResolveExecutionContext().Job.ID = %q, want %q", ec.Job.ID, job.ID)
	}

	if ec.Step.ID != "build" {
		t.Fatalf("ResolveExecutionContext().Step.ID = %q, want %q", ec.Step.ID, "build")
	}
}

func TestResolveExecutionContextInvalidPlanReturnsFirstValidationError(t *testing.T) {
	t.Parallel()

	job := Job{
		ID:           "job-1",
		MaxAuthority: AuthorityTierHigh,
		AllowedTools: []string{"read"},
		Plan: Plan{
			ID: "plan-1",
		},
	}

	want := ValidatePlan(job)[0]

	_, err := ResolveExecutionContext(job, "missing")
	if err == nil {
		t.Fatal("ResolveExecutionContext() error = nil, want ValidationError")
	}

	got, ok := err.(ValidationError)
	if !ok {
		t.Fatalf("ResolveExecutionContext() error type = %T, want ValidationError", err)
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ResolveExecutionContext() error = %#v, want %#v", got, want)
	}
}

func TestResolveExecutionContextMissingStep(t *testing.T) {
	t.Parallel()

	job := testExecutionJob()

	_, err := ResolveExecutionContext(job, "missing")
	if err == nil {
		t.Fatal("ResolveExecutionContext() error = nil, want ValidationError")
	}

	got, ok := err.(ValidationError)
	if !ok {
		t.Fatalf("ResolveExecutionContext() error type = %T, want ValidationError", err)
	}

	want := ValidationError{
		Code:    RejectionCodeUnknownStep,
		StepID:  "missing",
		Message: `step "missing" not found in plan`,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ResolveExecutionContext() error = %#v, want %#v", got, want)
	}
}

func TestResolveExecutionContextReturnsIndependentCopies(t *testing.T) {
	t.Parallel()

	job := testExecutionJob()

	ec, err := ResolveExecutionContext(job, "build")
	if err != nil {
		t.Fatalf("ResolveExecutionContext() error = %v", err)
	}

	job.AllowedTools[0] = "mutated-job-tool"
	job.Plan.Steps[0].ID = "mutated-step-id"
	job.Plan.Steps[0].AllowedTools[0] = "mutated-step-tool"
	job.Plan.Steps[0].SuccessCriteria[0] = "mutated-success"

	if ec.Job.AllowedTools[0] != "read" {
		t.Fatalf("ResolveExecutionContext().Job.AllowedTools[0] = %q, want %q", ec.Job.AllowedTools[0], "read")
	}

	if ec.Step.ID != "build" {
		t.Fatalf("ResolveExecutionContext().Step.ID = %q, want %q", ec.Step.ID, "build")
	}

	if ec.Step.AllowedTools[0] != "read" {
		t.Fatalf("ResolveExecutionContext().Step.AllowedTools[0] = %q, want %q", ec.Step.AllowedTools[0], "read")
	}

	if ec.Step.SuccessCriteria[0] != "produce code" {
		t.Fatalf("ResolveExecutionContext().Step.SuccessCriteria[0] = %q, want %q", ec.Step.SuccessCriteria[0], "produce code")
	}
}

func testExecutionJob() Job {
	return Job{
		ID:           "job-1",
		MaxAuthority: AuthorityTierHigh,
		AllowedTools: []string{"read", "write"},
		Plan: Plan{
			ID: "plan-1",
			Steps: []Step{
				{
					ID:                "build",
					Type:              StepTypeOneShotCode,
					RequiredAuthority: AuthorityTierMedium,
					AllowedTools:      []string{"read"},
					SuccessCriteria:   []string{"produce code"},
				},
				{
					ID:           "final",
					Type:         StepTypeFinalResponse,
					DependsOn:    []string{"build"},
					AllowedTools: []string{"read"},
				},
			},
		},
	}
}
