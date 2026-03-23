package missioncontrol

import (
	"testing"
	"time"
)

func TestCompleteRuntimeStepDiscussionSubtypeTransitionsToWaitingUser(t *testing.T) {
	t.Parallel()

	ec := testStepValidationExecutionContext(Step{
		ID:      "discuss",
		Type:    StepTypeDiscussion,
		Subtype: StepSubtypeBlocker,
	}, JobStateRunning)
	now := time.Date(2026, 3, 23, 12, 0, 0, 0, time.UTC)

	runtime, err := CompleteRuntimeStep(ec, now, StepValidationInput{FinalResponse: "Blocked pending approval."})
	if err != nil {
		t.Fatalf("CompleteRuntimeStep() error = %v", err)
	}

	if runtime.State != JobStateWaitingUser {
		t.Fatalf("State = %q, want %q", runtime.State, JobStateWaitingUser)
	}
	if runtime.ActiveStepID != "discuss" {
		t.Fatalf("ActiveStepID = %q, want %q", runtime.ActiveStepID, "discuss")
	}
	if runtime.WaitingReason != "discussion_blocker" {
		t.Fatalf("WaitingReason = %q, want %q", runtime.WaitingReason, "discussion_blocker")
	}
	if len(runtime.CompletedSteps) != 0 {
		t.Fatalf("CompletedSteps = %#v, want empty", runtime.CompletedSteps)
	}
}

func TestStepValidatorKindUsesSpecAlignedWaitUserName(t *testing.T) {
	t.Parallel()

	ec := testStepValidationExecutionContext(Step{
		ID:      "discuss",
		Type:    StepTypeDiscussion,
		Subtype: StepSubtypeAuthorization,
	}, JobStateWaitingUser)

	if kind := stepValidatorKind(ec); kind != StepValidatorKindWaitUser {
		t.Fatalf("stepValidatorKind() = %q, want %q", kind, StepValidatorKindWaitUser)
	}
}

func TestCompleteRuntimeStepDiscussionRejectsSideEffects(t *testing.T) {
	t.Parallel()

	ec := testStepValidationExecutionContext(Step{
		ID:   "discuss",
		Type: StepTypeDiscussion,
	}, JobStateRunning)

	_, err := CompleteRuntimeStep(ec, time.Date(2026, 3, 23, 12, 1, 0, 0, time.UTC), StepValidationInput{
		FinalResponse: "Here is the plan.",
		SuccessfulTools: []RuntimeToolCallEvidence{
			{ToolName: "filesystem", Arguments: map[string]interface{}{"action": "write", "path": "draft.txt"}},
		},
	})
	if err == nil {
		t.Fatal("CompleteRuntimeStep() error = nil, want discussion side-effect rejection")
	}
}

func TestCompleteRuntimeStepOneShotCodePausesAndRecordsCompletion(t *testing.T) {
	t.Parallel()

	ec := testStepValidationExecutionContext(Step{
		ID:   "build",
		Type: StepTypeOneShotCode,
	}, JobStateRunning)
	now := time.Date(2026, 3, 23, 12, 5, 0, 0, time.UTC)

	runtime, err := CompleteRuntimeStep(ec, now, StepValidationInput{
		FinalResponse: "Implemented the change.",
		SuccessfulTools: []RuntimeToolCallEvidence{
			{ToolName: "filesystem", Arguments: map[string]interface{}{"action": "write", "path": "result.txt"}},
			{ToolName: "filesystem", Arguments: map[string]interface{}{"action": "stat", "path": "result.txt"}},
		},
	})
	if err != nil {
		t.Fatalf("CompleteRuntimeStep() error = %v", err)
	}

	if runtime.State != JobStatePaused {
		t.Fatalf("State = %q, want %q", runtime.State, JobStatePaused)
	}
	if runtime.PausedReason != RuntimePauseReasonStepComplete {
		t.Fatalf("PausedReason = %q, want %q", runtime.PausedReason, RuntimePauseReasonStepComplete)
	}
	if runtime.ActiveStepID != "" {
		t.Fatalf("ActiveStepID = %q, want empty", runtime.ActiveStepID)
	}
	if len(runtime.CompletedSteps) != 1 || runtime.CompletedSteps[0].StepID != "build" {
		t.Fatalf("CompletedSteps = %#v, want build completion", runtime.CompletedSteps)
	}
}

func TestCompleteRuntimeStepOneShotCodeRequiresVerificationEvidence(t *testing.T) {
	t.Parallel()

	ec := testStepValidationExecutionContext(Step{
		ID:   "build",
		Type: StepTypeOneShotCode,
	}, JobStateRunning)

	_, err := CompleteRuntimeStep(ec, time.Date(2026, 3, 23, 12, 6, 0, 0, time.UTC), StepValidationInput{
		FinalResponse: "Implemented the change.",
		SuccessfulTools: []RuntimeToolCallEvidence{
			{ToolName: "filesystem", Arguments: map[string]interface{}{"action": "write", "path": "result.txt"}},
		},
	})
	if err == nil {
		t.Fatal("CompleteRuntimeStep() error = nil, want verification evidence failure")
	}
}

func TestCompleteRuntimeStepWaitingUserRequiresRecognizedInput(t *testing.T) {
	t.Parallel()

	ec := testStepValidationExecutionContext(Step{
		ID:      "discuss",
		Type:    StepTypeDiscussion,
		Subtype: StepSubtypeAuthorization,
	}, JobStateWaitingUser)
	now := time.Date(2026, 3, 23, 12, 10, 0, 0, time.UTC)

	_, err := CompleteRuntimeStep(ec, now, StepValidationInput{UserInput: "thanks"})
	if err == nil {
		t.Fatal("CompleteRuntimeStep() error = nil, want validation failure")
	}

	validationErr, ok := err.(ValidationError)
	if !ok {
		t.Fatalf("CompleteRuntimeStep() error type = %T, want ValidationError", err)
	}
	if validationErr.Code != RejectionCodeStepValidationFailed {
		t.Fatalf("ValidationError.Code = %q, want %q", validationErr.Code, RejectionCodeStepValidationFailed)
	}
}

func TestCompleteRuntimeStepWaitingUserApprovalPausesAndRecordsCompletion(t *testing.T) {
	t.Parallel()

	ec := testStepValidationExecutionContext(Step{
		ID:      "discuss",
		Type:    StepTypeDiscussion,
		Subtype: StepSubtypeAuthorization,
	}, JobStateWaitingUser)
	now := time.Date(2026, 3, 23, 12, 11, 0, 0, time.UTC)

	runtime, err := CompleteRuntimeStep(ec, now, StepValidationInput{UserInput: "approved"})
	if err != nil {
		t.Fatalf("CompleteRuntimeStep() error = %v", err)
	}

	if runtime.State != JobStatePaused {
		t.Fatalf("State = %q, want %q", runtime.State, JobStatePaused)
	}
	if runtime.ActiveStepID != "" {
		t.Fatalf("ActiveStepID = %q, want empty", runtime.ActiveStepID)
	}
	if len(runtime.CompletedSteps) != 1 || runtime.CompletedSteps[0].StepID != "discuss" {
		t.Fatalf("CompletedSteps = %#v, want discuss completion", runtime.CompletedSteps)
	}
}

func TestCompleteRuntimeStepFinalResponseRejectsFalseCompletionClaim(t *testing.T) {
	t.Parallel()

	ec := testStepValidationExecutionContext(Step{
		ID:   "final",
		Type: StepTypeFinalResponse,
	}, JobStateRunning)

	_, err := CompleteRuntimeStep(ec, time.Date(2026, 3, 23, 12, 15, 0, 0, time.UTC), StepValidationInput{
		FinalResponse: "Done",
	})
	if err == nil {
		t.Fatal("CompleteRuntimeStep() error = nil, want false completion claim rejection")
	}

	validationErr, ok := err.(ValidationError)
	if !ok {
		t.Fatalf("CompleteRuntimeStep() error type = %T, want ValidationError", err)
	}
	if validationErr.Code != RejectionCodeFalseCompletionClaim {
		t.Fatalf("ValidationError.Code = %q, want %q", validationErr.Code, RejectionCodeFalseCompletionClaim)
	}
}

func TestCompleteRuntimeStepFinalResponseRejectsMetaOnlyShape(t *testing.T) {
	t.Parallel()

	ec := testStepValidationExecutionContext(Step{
		ID:   "final",
		Type: StepTypeFinalResponse,
	}, JobStateRunning)

	_, err := CompleteRuntimeStep(ec, time.Date(2026, 3, 23, 12, 15, 30, 0, time.UTC), StepValidationInput{
		FinalResponse: "Here is the final answer.",
	})
	if err == nil {
		t.Fatal("CompleteRuntimeStep() error = nil, want meta-only final response rejection")
	}

	validationErr, ok := err.(ValidationError)
	if !ok {
		t.Fatalf("CompleteRuntimeStep() error type = %T, want ValidationError", err)
	}
	if validationErr.Code != RejectionCodeStepValidationFailed {
		t.Fatalf("ValidationError.Code = %q, want %q", validationErr.Code, RejectionCodeStepValidationFailed)
	}
}

func TestCompleteRuntimeStepFinalResponseCompletesJob(t *testing.T) {
	t.Parallel()

	ec := testStepValidationExecutionContext(Step{
		ID:   "final",
		Type: StepTypeFinalResponse,
	}, JobStateRunning)
	now := time.Date(2026, 3, 23, 12, 16, 0, 0, time.UTC)

	runtime, err := CompleteRuntimeStep(ec, now, StepValidationInput{
		FinalResponse: "Here is the final answer with the requested outcome.",
	})
	if err != nil {
		t.Fatalf("CompleteRuntimeStep() error = %v", err)
	}

	if runtime.State != JobStateCompleted {
		t.Fatalf("State = %q, want %q", runtime.State, JobStateCompleted)
	}
	if runtime.ActiveStepID != "" {
		t.Fatalf("ActiveStepID = %q, want empty", runtime.ActiveStepID)
	}
	if len(runtime.CompletedSteps) != 1 || runtime.CompletedSteps[0].StepID != "final" {
		t.Fatalf("CompletedSteps = %#v, want final completion", runtime.CompletedSteps)
	}
}

func testStepValidationExecutionContext(step Step, state JobState) ExecutionContext {
	job := Job{
		ID:           "job-1",
		MaxAuthority: AuthorityTierHigh,
		AllowedTools: []string{"read", "write"},
		Plan: Plan{
			ID:    "plan-1",
			Steps: []Step{step},
		},
	}

	return ExecutionContext{
		Job:  &job,
		Step: &step,
		Runtime: &JobRuntimeState{
			JobID:        job.ID,
			State:        state,
			ActiveStepID: step.ID,
		},
	}
}
