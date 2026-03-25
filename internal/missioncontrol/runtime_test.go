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

func TestBuildRuntimeControlContextCapturesMinimalStepBinding(t *testing.T) {
	t.Parallel()

	job := testExecutionJob()

	control, err := BuildRuntimeControlContext(job, "build")
	if err != nil {
		t.Fatalf("BuildRuntimeControlContext() error = %v", err)
	}
	if control.JobID != job.ID {
		t.Fatalf("JobID = %q, want %q", control.JobID, job.ID)
	}
	if control.Step.ID != "build" {
		t.Fatalf("Step.ID = %q, want %q", control.Step.ID, "build")
	}
	if !reflect.DeepEqual(control.AllowedTools, job.AllowedTools) {
		t.Fatalf("AllowedTools = %#v, want %#v", control.AllowedTools, job.AllowedTools)
	}
}

func TestBuildInspectablePlanContextCapturesValidatedPlan(t *testing.T) {
	t.Parallel()

	job := testExecutionJob()

	plan, err := BuildInspectablePlanContext(job)
	if err != nil {
		t.Fatalf("BuildInspectablePlanContext() error = %v", err)
	}
	if plan.MaxAuthority != job.MaxAuthority {
		t.Fatalf("MaxAuthority = %q, want %q", plan.MaxAuthority, job.MaxAuthority)
	}
	if !reflect.DeepEqual(plan.AllowedTools, job.AllowedTools) {
		t.Fatalf("AllowedTools = %#v, want %#v", plan.AllowedTools, job.AllowedTools)
	}
	if len(plan.Steps) != len(job.Plan.Steps) {
		t.Fatalf("len(Steps) = %d, want %d", len(plan.Steps), len(job.Plan.Steps))
	}
	if plan.Steps[1].ID != "final" {
		t.Fatalf("Steps[1].ID = %q, want %q", plan.Steps[1].ID, "final")
	}
}

func TestResolveExecutionContextWithRuntimeControlReconstructsExecutionContext(t *testing.T) {
	t.Parallel()

	job := testExecutionJob()
	control, err := BuildRuntimeControlContext(job, "build")
	if err != nil {
		t.Fatalf("BuildRuntimeControlContext() error = %v", err)
	}

	runtime := JobRuntimeState{
		JobID:        job.ID,
		State:        JobStateRunning,
		ActiveStepID: "build",
	}
	ec, err := ResolveExecutionContextWithRuntimeControl(control, runtime)
	if err != nil {
		t.Fatalf("ResolveExecutionContextWithRuntimeControl() error = %v", err)
	}
	if ec.Job == nil || ec.Job.ID != job.ID {
		t.Fatalf("ExecutionContext.Job = %#v, want job %q", ec.Job, job.ID)
	}
	if ec.Step == nil || ec.Step.ID != "build" {
		t.Fatalf("ExecutionContext.Step = %#v, want build", ec.Step)
	}
	if ec.Runtime == nil || ec.Runtime.State != JobStateRunning {
		t.Fatalf("ExecutionContext.Runtime = %#v, want running runtime", ec.Runtime)
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

func TestPauseJobRuntimeDoesNotCompleteActiveStep(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 24, 12, 0, 0, 0, time.UTC)
	runtime, err := PauseJobRuntime(JobRuntimeState{
		JobID:        "job-1",
		State:        JobStateRunning,
		ActiveStepID: "build",
		CreatedAt:    time.Date(2026, 3, 24, 11, 0, 0, 0, time.UTC),
		StartedAt:    time.Date(2026, 3, 24, 11, 0, 0, 0, time.UTC),
		ActiveStepAt: time.Date(2026, 3, 24, 11, 30, 0, 0, time.UTC),
	}, now)
	if err != nil {
		t.Fatalf("PauseJobRuntime() error = %v", err)
	}

	if runtime.State != JobStatePaused {
		t.Fatalf("State = %q, want %q", runtime.State, JobStatePaused)
	}
	if runtime.ActiveStepID != "build" {
		t.Fatalf("ActiveStepID = %q, want %q", runtime.ActiveStepID, "build")
	}
	if runtime.PausedReason != RuntimePauseReasonOperatorCommand {
		t.Fatalf("PausedReason = %q, want %q", runtime.PausedReason, RuntimePauseReasonOperatorCommand)
	}
	if len(runtime.CompletedSteps) != 0 {
		t.Fatalf("CompletedSteps = %#v, want empty", runtime.CompletedSteps)
	}
}

func TestResumePausedJobRuntimeRequiresPausedState(t *testing.T) {
	t.Parallel()

	_, err := ResumePausedJobRuntime(JobRuntimeState{
		JobID:        "job-1",
		State:        JobStateWaitingUser,
		ActiveStepID: "build",
	}, time.Date(2026, 3, 24, 12, 0, 0, 0, time.UTC))
	if err == nil {
		t.Fatal("ResumePausedJobRuntime() error = nil, want paused-state failure")
	}

	validationErr, ok := err.(ValidationError)
	if !ok {
		t.Fatalf("ResumePausedJobRuntime() error type = %T, want ValidationError", err)
	}
	if validationErr.Code != RejectionCodeInvalidRuntimeState {
		t.Fatalf("ValidationError.Code = %q, want %q", validationErr.Code, RejectionCodeInvalidRuntimeState)
	}
}

func TestAbortJobRuntimeTransitionsToTerminalAbortedState(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 24, 12, 0, 0, 0, time.UTC)
	runtime, err := AbortJobRuntime(JobRuntimeState{
		JobID:        "job-1",
		State:        JobStatePaused,
		ActiveStepID: "build",
		PausedReason: RuntimePauseReasonOperatorCommand,
		PausedAt:     time.Date(2026, 3, 24, 11, 45, 0, 0, time.UTC),
	}, now)
	if err != nil {
		t.Fatalf("AbortJobRuntime() error = %v", err)
	}

	if runtime.State != JobStateAborted {
		t.Fatalf("State = %q, want %q", runtime.State, JobStateAborted)
	}
	if runtime.AbortedReason != RuntimeAbortReasonOperatorCommand {
		t.Fatalf("AbortedReason = %q, want %q", runtime.AbortedReason, RuntimeAbortReasonOperatorCommand)
	}
	if runtime.AbortedAt != now {
		t.Fatalf("AbortedAt = %v, want %v", runtime.AbortedAt, now)
	}
	if runtime.ActiveStepID != "" {
		t.Fatalf("ActiveStepID = %q, want empty", runtime.ActiveStepID)
	}
	if !IsTerminalJobState(runtime.State) {
		t.Fatalf("IsTerminalJobState(%q) = false, want true", runtime.State)
	}
}

func TestRuntimeControlRejectsTerminalStatesDeterministically(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 24, 12, 30, 0, 0, time.UTC)
	testCases := []struct {
		name string
		run  func() error
		want RejectionCode
	}{
		{
			name: "resume completed",
			run: func() error {
				_, err := ResumePausedJobRuntime(JobRuntimeState{
					JobID: "job-1",
					State: JobStateCompleted,
				}, now)
				return err
			},
			want: RejectionCodeInvalidRuntimeState,
		},
		{
			name: "resume failed",
			run: func() error {
				_, err := ResumePausedJobRuntime(JobRuntimeState{
					JobID: "job-1",
					State: JobStateFailed,
				}, now)
				return err
			},
			want: RejectionCodeInvalidRuntimeState,
		},
		{
			name: "resume aborted",
			run: func() error {
				_, err := ResumePausedJobRuntime(JobRuntimeState{
					JobID: "job-1",
					State: JobStateAborted,
				}, now)
				return err
			},
			want: RejectionCodeInvalidRuntimeState,
		},
		{
			name: "abort completed",
			run: func() error {
				_, err := AbortJobRuntime(JobRuntimeState{
					JobID: "job-1",
					State: JobStateCompleted,
				}, now)
				return err
			},
			want: RejectionCodeInvalidRuntimeState,
		},
		{
			name: "abort failed",
			run: func() error {
				_, err := AbortJobRuntime(JobRuntimeState{
					JobID: "job-1",
					State: JobStateFailed,
				}, now)
				return err
			},
			want: RejectionCodeInvalidRuntimeState,
		},
		{
			name: "abort aborted",
			run: func() error {
				_, err := AbortJobRuntime(JobRuntimeState{
					JobID: "job-1",
					State: JobStateAborted,
				}, now)
				return err
			},
			want: RejectionCodeInvalidRuntimeState,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.run()
			if err == nil {
				t.Fatal("runtime control error = nil, want deterministic rejection")
			}

			validationErr, ok := err.(ValidationError)
			if !ok {
				t.Fatalf("runtime control error type = %T, want ValidationError", err)
			}
			if validationErr.Code != tc.want {
				t.Fatalf("ValidationError.Code = %q, want %q", validationErr.Code, tc.want)
			}
		})
	}
}

func TestSetJobRuntimeActiveStepRejectsAbortedRuntime(t *testing.T) {
	t.Parallel()

	job := testExecutionJob()
	_, err := SetJobRuntimeActiveStep(job, &JobRuntimeState{
		JobID: job.ID,
		State: JobStateAborted,
	}, "build", time.Date(2026, 3, 24, 12, 45, 0, 0, time.UTC))
	if err == nil {
		t.Fatal("SetJobRuntimeActiveStep() error = nil, want aborted-state rejection")
	}

	validationErr, ok := err.(ValidationError)
	if !ok {
		t.Fatalf("SetJobRuntimeActiveStep() error type = %T, want ValidationError", err)
	}
	if validationErr.Code != RejectionCodeInvalidJobTransition {
		t.Fatalf("ValidationError.Code = %q, want %q", validationErr.Code, RejectionCodeInvalidJobTransition)
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
				Content: &ApprovalRequestContent{
					ProposedAction: "Complete the authorization discussion step and continue to the next mission step.",
				},
				RequestedVia: ApprovalRequestedViaRuntime,
				State:        ApprovalStatePending,
				RequestedAt:  time.Date(2026, 3, 23, 11, 30, 0, 0, time.UTC),
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
	ec.Runtime.ApprovalRequests[0].Content.ProposedAction = "mutated-content"
	ec.Runtime.ApprovalGrants[0].StepID = "mutated-grant"
	if runtime.CompletedSteps[0].StepID != "draft" {
		t.Fatalf("original runtime step = %q, want %q", runtime.CompletedSteps[0].StepID, "draft")
	}
	if runtime.ApprovalRequests[0].StepID != "build" {
		t.Fatalf("original approval request step = %q, want %q", runtime.ApprovalRequests[0].StepID, "build")
	}
	if runtime.ApprovalRequests[0].Content == nil || runtime.ApprovalRequests[0].Content.ProposedAction != "Complete the authorization discussion step and continue to the next mission step." {
		t.Fatalf("original approval request content = %#v, want preserved content", runtime.ApprovalRequests[0].Content)
	}
	if runtime.ApprovalGrants[0].StepID != "draft" {
		t.Fatalf("original approval grant step = %q, want %q", runtime.ApprovalGrants[0].StepID, "draft")
	}
}
