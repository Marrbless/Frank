package missioncontrol

import (
	"reflect"
	"testing"
)

func TestValidatePlanEmptyPlan(t *testing.T) {
	t.Parallel()

	errors := ValidatePlan(testJob(nil))

	want := []ValidationError{
		{
			Code:    RejectionCodeMissingTerminalFinalStep,
			Message: "plan must include a terminal final_response step",
		},
	}

	if !reflect.DeepEqual(errors, want) {
		t.Fatalf("ValidatePlan() = %#v, want %#v", errors, want)
	}
}

func TestValidatePlanDuplicateStepIDs(t *testing.T) {
	t.Parallel()

	errors := ValidatePlan(testJob([]Step{
		{ID: "dup", Type: StepTypeDiscussion},
		{ID: "dup", Type: StepTypeStaticArtifact},
		{ID: "final", Type: StepTypeFinalResponse},
	}))

	want := []ValidationError{
		{
			Code:    RejectionCodeDuplicateStepID,
			StepID:  "dup",
			Message: "duplicate step ID also used at index 0",
		},
	}

	if !reflect.DeepEqual(errors, want) {
		t.Fatalf("ValidatePlan() = %#v, want %#v", errors, want)
	}
}

func TestValidatePlanMissingDependency(t *testing.T) {
	t.Parallel()

	errors := ValidatePlan(testJob([]Step{
		{ID: "draft", Type: StepTypeDiscussion, DependsOn: []string{"missing"}},
		{ID: "final", Type: StepTypeFinalResponse},
	}))

	want := []ValidationError{
		{
			Code:    RejectionCodeMissingDependencyTarget,
			StepID:  "draft",
			Message: "missing dependency target: missing",
		},
	}

	if !reflect.DeepEqual(errors, want) {
		t.Fatalf("ValidatePlan() = %#v, want %#v", errors, want)
	}
}

func TestValidatePlanCycle(t *testing.T) {
	t.Parallel()

	errors := ValidatePlan(testJob([]Step{
		{ID: "a", Type: StepTypeDiscussion, DependsOn: []string{"b"}},
		{ID: "b", Type: StepTypeOneShotCode, DependsOn: []string{"a"}},
		{ID: "final", Type: StepTypeFinalResponse},
	}))

	want := []ValidationError{
		{
			Code:    RejectionCodeDependencyCycle,
			StepID:  "a",
			Message: "dependency cycle detected",
		},
	}

	if !reflect.DeepEqual(errors, want) {
		t.Fatalf("ValidatePlan() = %#v, want %#v", errors, want)
	}
}

func TestValidatePlanMissingFinalResponse(t *testing.T) {
	t.Parallel()

	errors := ValidatePlan(testJob([]Step{
		{ID: "draft", Type: StepTypeDiscussion},
	}))

	want := []ValidationError{
		{
			Code:    RejectionCodeMissingTerminalFinalStep,
			Message: "plan must include a terminal final_response step",
		},
	}

	if !reflect.DeepEqual(errors, want) {
		t.Fatalf("ValidatePlan() = %#v, want %#v", errors, want)
	}
}

func TestValidatePlanInvalidStepType(t *testing.T) {
	t.Parallel()

	errors := ValidatePlan(testJob([]Step{
		{ID: "draft", Type: StepType(""), DependsOn: nil},
		{ID: "final", Type: StepTypeFinalResponse},
	}))

	want := []ValidationError{
		{
			Code:    RejectionCodeInvalidStepType,
			StepID:  "draft",
			Message: "step type must be one of discussion, static_artifact, one_shot_code, wait_user, final_response",
		},
	}

	if !reflect.DeepEqual(errors, want) {
		t.Fatalf("ValidatePlan() = %#v, want %#v", errors, want)
	}
}

func TestValidatePlanAcceptsWaitUserStepWithSubtype(t *testing.T) {
	t.Parallel()

	errors := ValidatePlan(testJob([]Step{
		{ID: "hold", Type: StepTypeWaitUser, Subtype: StepSubtypeDefinition},
		{ID: "final", Type: StepTypeFinalResponse, DependsOn: []string{"hold"}},
	}))
	if len(errors) != 0 {
		t.Fatalf("ValidatePlan() = %#v, want no errors", errors)
	}
}

func TestValidatePlanRejectsWaitUserStepWithoutSubtype(t *testing.T) {
	t.Parallel()

	errors := ValidatePlan(testJob([]Step{
		{ID: "hold", Type: StepTypeWaitUser},
		{ID: "final", Type: StepTypeFinalResponse, DependsOn: []string{"hold"}},
	}))

	want := []ValidationError{
		{
			Code:    RejectionCodeInvalidStepType,
			StepID:  "hold",
			Message: "wait_user step requires blocker, authorization, or definition subtype",
		},
	}
	if !reflect.DeepEqual(errors, want) {
		t.Fatalf("ValidatePlan() = %#v, want %#v", errors, want)
	}
}

func TestValidatePlanAuthorityExceeded(t *testing.T) {
	t.Parallel()

	job := testJob([]Step{
		{ID: "draft", Type: StepTypeDiscussion, RequiredAuthority: AuthorityTierHigh},
		{ID: "final", Type: StepTypeFinalResponse},
	})
	job.MaxAuthority = AuthorityTierLow

	errors := ValidatePlan(job)

	want := []ValidationError{
		{
			Code:    RejectionCodeAuthorityExceeded,
			StepID:  "draft",
			Message: "step required authority exceeds job max authority",
		},
	}

	if !reflect.DeepEqual(errors, want) {
		t.Fatalf("ValidatePlan() = %#v, want %#v", errors, want)
	}
}

func TestValidatePlanDisallowedStepTool(t *testing.T) {
	t.Parallel()

	job := testJob([]Step{
		{ID: "draft", Type: StepTypeDiscussion, AllowedTools: []string{"write"}},
		{ID: "final", Type: StepTypeFinalResponse},
	})
	job.AllowedTools = []string{"read"}

	errors := ValidatePlan(job)

	want := []ValidationError{
		{
			Code:    RejectionCodeToolNotAllowed,
			StepID:  "draft",
			Message: "step tool is not allowed by job tool scope: write",
		},
	}

	if !reflect.DeepEqual(errors, want) {
		t.Fatalf("ValidatePlan() = %#v, want %#v", errors, want)
	}
}

func TestValidatePlanValidPlan(t *testing.T) {
	t.Parallel()

	errors := ValidatePlan(testJob([]Step{
		{ID: "discuss", Type: StepTypeDiscussion, RequiredAuthority: AuthorityTierLow, AllowedTools: []string{"read"}},
		{ID: "build", Type: StepTypeOneShotCode, DependsOn: []string{"discuss"}, RequiredAuthority: AuthorityTierMedium, AllowedTools: []string{"read", "write"}},
		{ID: "final", Type: StepTypeFinalResponse, DependsOn: []string{"build"}},
	}))

	if len(errors) != 0 {
		t.Fatalf("ValidatePlan() = %#v, want no errors", errors)
	}
}

func TestValidatePlanErrorOrdering(t *testing.T) {
	t.Parallel()

	job := testJob([]Step{
		{ID: "dup", Type: StepTypeDiscussion, DependsOn: []string{"missing"}, RequiredAuthority: AuthorityTierHigh, AllowedTools: []string{"write"}},
		{ID: "dup", Type: StepTypeDiscussion},
		{ID: "cycle-a", Type: StepTypeDiscussion, DependsOn: []string{"cycle-b"}},
		{ID: "cycle-b", Type: StepTypeOneShotCode, DependsOn: []string{"cycle-a"}},
		{ID: "bad-type", Type: StepType("bogus")},
	})
	job.MaxAuthority = AuthorityTierLow
	job.AllowedTools = []string{"read"}

	errors := ValidatePlan(job)

	wantCodes := []RejectionCode{
		RejectionCodeDuplicateStepID,
		RejectionCodeMissingDependencyTarget,
		RejectionCodeDependencyCycle,
		RejectionCodeMissingTerminalFinalStep,
		RejectionCodeInvalidStepType,
		RejectionCodeAuthorityExceeded,
		RejectionCodeToolNotAllowed,
	}

	if len(errors) != len(wantCodes) {
		t.Fatalf("ValidatePlan() returned %d errors, want %d: %#v", len(errors), len(wantCodes), errors)
	}

	for i, wantCode := range wantCodes {
		if errors[i].Code != wantCode {
			t.Fatalf("error[%d].Code = %q, want %q; errors = %#v", i, errors[i].Code, wantCode, errors)
		}
	}
}

func testJob(steps []Step) Job {
	return Job{
		ID:           "job-1",
		MaxAuthority: AuthorityTierMedium,
		AllowedTools: []string{"read", "write"},
		Plan: Plan{
			ID:    "plan-1",
			Steps: steps,
		},
	}
}
