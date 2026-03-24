package missioncontrol

import (
	"reflect"
	"testing"
	"time"
)

func TestSetJobRuntimeActiveStepStartsRunningRuntime(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 23, 12, 0, 0, 0, time.UTC)
	job := testExecutionJob()

	runtime, err := SetJobRuntimeActiveStep(job, nil, "build", now)
	if err != nil {
		t.Fatalf("SetJobRuntimeActiveStep() error = %v", err)
	}

	if runtime.JobID != job.ID {
		t.Fatalf("JobID = %q, want %q", runtime.JobID, job.ID)
	}
	if runtime.State != JobStateRunning {
		t.Fatalf("State = %q, want %q", runtime.State, JobStateRunning)
	}
	if runtime.ActiveStepID != "build" {
		t.Fatalf("ActiveStepID = %q, want %q", runtime.ActiveStepID, "build")
	}
	if runtime.CreatedAt != now || runtime.UpdatedAt != now || runtime.StartedAt != now || runtime.ActiveStepAt != now {
		t.Fatalf("timestamps = %#v, want all primary timestamps set to %v", runtime, now)
	}
}

func TestTransitionJobRuntimeCompletedRequiresValidation(t *testing.T) {
	t.Parallel()

	current := JobRuntimeState{
		JobID:        "job-1",
		State:        JobStateRunning,
		ActiveStepID: "build",
		CreatedAt:    time.Date(2026, 3, 23, 11, 0, 0, 0, time.UTC),
	}

	_, err := TransitionJobRuntime(current, JobStateCompleted, time.Date(2026, 3, 23, 12, 0, 0, 0, time.UTC), RuntimeTransitionOptions{})
	if err == nil {
		t.Fatal("TransitionJobRuntime() error = nil, want validation failure")
	}

	validationErr, ok := err.(ValidationError)
	if !ok {
		t.Fatalf("TransitionJobRuntime() error type = %T, want ValidationError", err)
	}
	if validationErr.Code != RejectionCodeValidationRequired {
		t.Fatalf("ValidationError.Code = %q, want %q", validationErr.Code, RejectionCodeValidationRequired)
	}
}

func TestResumeJobRuntimeAfterBootRequiresApproval(t *testing.T) {
	t.Parallel()

	_, err := ResumeJobRuntimeAfterBoot(JobRuntimeState{
		JobID:        "job-1",
		State:        JobStateRunning,
		ActiveStepID: "build",
	}, time.Date(2026, 3, 23, 12, 0, 0, 0, time.UTC), false)
	if err == nil {
		t.Fatal("ResumeJobRuntimeAfterBoot() error = nil, want approval failure")
	}

	validationErr, ok := err.(ValidationError)
	if !ok {
		t.Fatalf("ResumeJobRuntimeAfterBoot() error type = %T, want ValidationError", err)
	}
	if validationErr.Code != RejectionCodeResumeApprovalRequired {
		t.Fatalf("ValidationError.Code = %q, want %q", validationErr.Code, RejectionCodeResumeApprovalRequired)
	}
}

func TestValidateRuntimeExecutionWaitingUserDenied(t *testing.T) {
	t.Parallel()

	err := ValidateRuntimeExecution(ExecutionContext{
		Job:  &Job{ID: "job-1"},
		Step: &Step{ID: "build"},
		Runtime: &JobRuntimeState{
			JobID:        "job-1",
			State:        JobStateWaitingUser,
			ActiveStepID: "build",
		},
	})
	if err == nil {
		t.Fatal("ValidateRuntimeExecution() error = nil, want waiting_user denial")
	}

	validationErr, ok := err.(ValidationError)
	if !ok {
		t.Fatalf("ValidateRuntimeExecution() error type = %T, want ValidationError", err)
	}
	if validationErr.Code != RejectionCodeWaitingUser {
		t.Fatalf("ValidationError.Code = %q, want %q", validationErr.Code, RejectionCodeWaitingUser)
	}
}

func TestResolveExecutionContextWithRuntimeIncludesIndependentRuntimeCopy(t *testing.T) {
	t.Parallel()

	job := testExecutionJob()
	runtime := JobRuntimeState{
		JobID:        job.ID,
		State:        JobStateRunning,
		ActiveStepID: "build",
		CompletedSteps: []RuntimeStepRecord{
			{StepID: "draft", At: time.Date(2026, 3, 23, 11, 0, 0, 0, time.UTC)},
		},
		ApprovalRequests: []ApprovalRequest{
			{
				JobID:           job.ID,
				StepID:          "build",
				RequestedAction: ApprovalRequestedActionStepComplete,
				Scope:           ApprovalScopeMissionStep,
				RequestedVia:    ApprovalRequestedViaRuntime,
				State:           ApprovalStatePending,
				RequestedAt:     time.Date(2026, 3, 23, 11, 30, 0, 0, time.UTC),
			},
		},
		ApprovalGrants: []ApprovalGrant{
			{
				JobID:           job.ID,
				StepID:          "draft",
				RequestedAction: ApprovalRequestedActionStepComplete,
				Scope:           ApprovalScopeMissionStep,
				GrantedVia:      ApprovalGrantedViaOperatorCommand,
				State:           ApprovalStateGranted,
				GrantedAt:       time.Date(2026, 3, 23, 11, 45, 0, 0, time.UTC),
			},
		},
	}

	ec, err := ResolveExecutionContextWithRuntime(job, runtime)
	if err != nil {
		t.Fatalf("ResolveExecutionContextWithRuntime() error = %v", err)
	}

	if ec.Runtime == nil {
		t.Fatal("ResolveExecutionContextWithRuntime().Runtime = nil, want non-nil")
	}
	if !reflect.DeepEqual(*ec.Runtime, runtime) {
		t.Fatalf("ResolveExecutionContextWithRuntime().Runtime = %#v, want %#v", *ec.Runtime, runtime)
	}

	ec.Runtime.CompletedSteps[0].StepID = "mutated"
	ec.Runtime.ApprovalRequests[0].StepID = "mutated-request"
	ec.Runtime.ApprovalGrants[0].StepID = "mutated-grant"
	if runtime.CompletedSteps[0].StepID != "draft" {
		t.Fatalf("original runtime step = %q, want %q", runtime.CompletedSteps[0].StepID, "draft")
	}
	if runtime.ApprovalRequests[0].StepID != "build" {
		t.Fatalf("original approval request step = %q, want %q", runtime.ApprovalRequests[0].StepID, "build")
	}
	if runtime.ApprovalGrants[0].StepID != "draft" {
		t.Fatalf("original approval grant step = %q, want %q", runtime.ApprovalGrants[0].StepID, "draft")
	}
}
