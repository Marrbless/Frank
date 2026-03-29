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

func TestBuildInspectablePlanContextPreservesStaticArtifactContractMetadata(t *testing.T) {
	t.Parallel()

	job := Job{
		ID:           "job-1",
		SpecVersion:  JobSpecVersionV2,
		MaxAuthority: AuthorityTierHigh,
		AllowedTools: []string{"write", "read"},
		Plan: Plan{
			ID: "plan-1",
			Steps: []Step{
				{
					ID:                   "artifact",
					Type:                 StepTypeStaticArtifact,
					SuccessCriteria:      []string{"write a report"},
					StaticArtifactPath:   "report.json",
					StaticArtifactFormat: "json",
				},
				{
					ID:        "final",
					Type:      StepTypeFinalResponse,
					DependsOn: []string{"artifact"},
				},
			},
		},
	}

	plan, err := BuildInspectablePlanContext(job)
	if err != nil {
		t.Fatalf("BuildInspectablePlanContext() error = %v", err)
	}
	if plan.Steps[0].StaticArtifactPath != "report.json" {
		t.Fatalf("Steps[0].StaticArtifactPath = %q, want %q", plan.Steps[0].StaticArtifactPath, "report.json")
	}
	if plan.Steps[0].StaticArtifactFormat != "json" {
		t.Fatalf("Steps[0].StaticArtifactFormat = %q, want %q", plan.Steps[0].StaticArtifactFormat, "json")
	}
}

func TestBuildInspectablePlanContextPreservesOneShotCodeContractMetadata(t *testing.T) {
	t.Parallel()

	job := Job{
		ID:           "job-1",
		SpecVersion:  JobSpecVersionV2,
		MaxAuthority: AuthorityTierHigh,
		AllowedTools: []string{"write", "read"},
		Plan: Plan{
			ID: "plan-1",
			Steps: []Step{
				{
					ID:                  "build",
					Type:                StepTypeOneShotCode,
					SuccessCriteria:     []string{"write code"},
					OneShotArtifactPath: "main.go",
				},
				{
					ID:        "final",
					Type:      StepTypeFinalResponse,
					DependsOn: []string{"build"},
				},
			},
		},
	}

	plan, err := BuildInspectablePlanContext(job)
	if err != nil {
		t.Fatalf("BuildInspectablePlanContext() error = %v", err)
	}
	if plan.Steps[0].OneShotArtifactPath != "main.go" {
		t.Fatalf("Steps[0].OneShotArtifactPath = %q, want %q", plan.Steps[0].OneShotArtifactPath, "main.go")
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

func TestResolveExecutionContextWithRuntimeControlRejectsCompletedActiveStepReplay(t *testing.T) {
	t.Parallel()

	job := testExecutionJob()
	control, err := BuildRuntimeControlContext(job, "build")
	if err != nil {
		t.Fatalf("BuildRuntimeControlContext() error = %v", err)
	}

	_, err = ResolveExecutionContextWithRuntimeControl(control, JobRuntimeState{
		JobID:        job.ID,
		State:        JobStateRunning,
		ActiveStepID: "build",
		CompletedSteps: []RuntimeStepRecord{
			{StepID: "build", At: time.Date(2026, 3, 27, 12, 0, 0, 0, time.UTC)},
		},
	})
	if err == nil {
		t.Fatal("ResolveExecutionContextWithRuntimeControl() error = nil, want completed-step replay rejection")
	}

	validationErr, ok := err.(ValidationError)
	if !ok {
		t.Fatalf("ResolveExecutionContextWithRuntimeControl() error type = %T, want ValidationError", err)
	}
	if validationErr.Code != RejectionCodeInvalidRuntimeState {
		t.Fatalf("ValidationError.Code = %q, want %q", validationErr.Code, RejectionCodeInvalidRuntimeState)
	}
	if validationErr.StepID != "build" {
		t.Fatalf("ValidationError.StepID = %q, want %q", validationErr.StepID, "build")
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

func TestTransitionJobRuntimeRejectsCompletedReplayMarkerWhenRecordingCompletion(t *testing.T) {
	t.Parallel()

	current := JobRuntimeState{
		JobID:        "job-1",
		State:        JobStateRunning,
		ActiveStepID: "build",
		CompletedSteps: []RuntimeStepRecord{
			{StepID: "build", At: time.Date(2026, 3, 27, 12, 0, 0, 0, time.UTC)},
		},
	}

	_, err := TransitionJobRuntime(current, JobStateCompleted, time.Date(2026, 3, 27, 12, 5, 0, 0, time.UTC), RuntimeTransitionOptions{
		validationResult: &stepValidationResult{recordCompletion: true},
	})
	if err == nil {
		t.Fatal("TransitionJobRuntime() error = nil, want completed-step replay rejection")
	}

	validationErr, ok := err.(ValidationError)
	if !ok {
		t.Fatalf("TransitionJobRuntime() error type = %T, want ValidationError", err)
	}
	if validationErr.Code != RejectionCodeInvalidRuntimeState {
		t.Fatalf("ValidationError.Code = %q, want %q", validationErr.Code, RejectionCodeInvalidRuntimeState)
	}
	if validationErr.StepID != "build" {
		t.Fatalf("ValidationError.StepID = %q, want %q", validationErr.StepID, "build")
	}
}

func TestTransitionJobRuntimeRejectsFailedReplayMarkerWhenRecordingFailure(t *testing.T) {
	t.Parallel()

	current := JobRuntimeState{
		JobID:        "job-1",
		State:        JobStateRunning,
		ActiveStepID: "build",
		FailedSteps: []RuntimeStepRecord{
			{StepID: "build", Reason: "validator failed", At: time.Date(2026, 3, 27, 12, 0, 0, 0, time.UTC)},
		},
	}

	_, err := TransitionJobRuntime(current, JobStateFailed, time.Date(2026, 3, 27, 12, 5, 0, 0, time.UTC), RuntimeTransitionOptions{
		FailureReason: "validator failed",
	})
	if err == nil {
		t.Fatal("TransitionJobRuntime() error = nil, want failed-step replay rejection")
	}

	validationErr, ok := err.(ValidationError)
	if !ok {
		t.Fatalf("TransitionJobRuntime() error type = %T, want ValidationError", err)
	}
	if validationErr.Code != RejectionCodeInvalidRuntimeState {
		t.Fatalf("ValidationError.Code = %q, want %q", validationErr.Code, RejectionCodeInvalidRuntimeState)
	}
	if validationErr.StepID != "build" {
		t.Fatalf("ValidationError.StepID = %q, want %q", validationErr.StepID, "build")
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

func TestResumeJobRuntimeAfterBootRejectsCompletedActiveStepReplay(t *testing.T) {
	t.Parallel()

	_, err := ResumeJobRuntimeAfterBoot(JobRuntimeState{
		JobID:        "job-1",
		State:        JobStatePaused,
		ActiveStepID: "build",
		CompletedSteps: []RuntimeStepRecord{
			{StepID: "build", At: time.Date(2026, 3, 27, 12, 0, 0, 0, time.UTC)},
		},
	}, time.Date(2026, 3, 27, 12, 5, 0, 0, time.UTC), true)
	if err == nil {
		t.Fatal("ResumeJobRuntimeAfterBoot() error = nil, want completed-step replay rejection")
	}

	validationErr, ok := err.(ValidationError)
	if !ok {
		t.Fatalf("ResumeJobRuntimeAfterBoot() error type = %T, want ValidationError", err)
	}
	if validationErr.Code != RejectionCodeInvalidRuntimeState {
		t.Fatalf("ValidationError.Code = %q, want %q", validationErr.Code, RejectionCodeInvalidRuntimeState)
	}
	if validationErr.StepID != "build" {
		t.Fatalf("ValidationError.StepID = %q, want %q", validationErr.StepID, "build")
	}
}

func TestResumeJobRuntimeAfterBootRejectsFailedActiveStepReplay(t *testing.T) {
	t.Parallel()

	_, err := ResumeJobRuntimeAfterBoot(JobRuntimeState{
		JobID:        "job-1",
		State:        JobStatePaused,
		ActiveStepID: "build",
		FailedSteps: []RuntimeStepRecord{
			{StepID: "build", At: time.Date(2026, 3, 27, 12, 0, 0, 0, time.UTC)},
		},
	}, time.Date(2026, 3, 27, 12, 5, 0, 0, time.UTC), true)
	if err == nil {
		t.Fatal("ResumeJobRuntimeAfterBoot() error = nil, want failed-step replay rejection")
	}

	validationErr, ok := err.(ValidationError)
	if !ok {
		t.Fatalf("ResumeJobRuntimeAfterBoot() error type = %T, want ValidationError", err)
	}
	if validationErr.Code != RejectionCodeInvalidRuntimeState {
		t.Fatalf("ValidationError.Code = %q, want %q", validationErr.Code, RejectionCodeInvalidRuntimeState)
	}
	if validationErr.StepID != "build" {
		t.Fatalf("ValidationError.StepID = %q, want %q", validationErr.StepID, "build")
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

func TestPauseJobRuntimeForBudgetExhaustionPersistsBudgetBlockerAndAudit(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 28, 12, 0, 0, 0, time.UTC)
	runtime, err := PauseJobRuntimeForBudgetExhaustion(JobRuntimeState{
		JobID:         "job-1",
		State:         JobStateWaitingUser,
		ActiveStepID:  "final",
		WaitingReason: "discussion_authorization",
		WaitingAt:     time.Date(2026, 3, 28, 11, 58, 0, 0, time.UTC),
	}, now, RuntimeBudgetBlockerRecord{
		Ceiling:  "owner_messages",
		Limit:    20,
		Observed: 20,
		Message:  "owner-facing message budget exhausted",
	})
	if err != nil {
		t.Fatalf("PauseJobRuntimeForBudgetExhaustion() error = %v", err)
	}

	if runtime.State != JobStatePaused {
		t.Fatalf("State = %q, want %q", runtime.State, JobStatePaused)
	}
	if runtime.PausedReason != RuntimePauseReasonBudgetExhausted {
		t.Fatalf("PausedReason = %q, want %q", runtime.PausedReason, RuntimePauseReasonBudgetExhausted)
	}
	if runtime.WaitingReason != "" {
		t.Fatalf("WaitingReason = %q, want empty after budget pause", runtime.WaitingReason)
	}
	if runtime.BudgetBlocker == nil {
		t.Fatal("BudgetBlocker = nil, want persisted blocker")
	}
	if runtime.BudgetBlocker.Ceiling != "owner_messages" {
		t.Fatalf("BudgetBlocker.Ceiling = %q, want %q", runtime.BudgetBlocker.Ceiling, "owner_messages")
	}
	if runtime.BudgetBlocker.Limit != 20 || runtime.BudgetBlocker.Observed != 20 {
		t.Fatalf("BudgetBlocker limits = %#v, want limit=20 observed=20", runtime.BudgetBlocker)
	}
	if runtime.BudgetBlocker.Message != "owner-facing message budget exhausted" {
		t.Fatalf("BudgetBlocker.Message = %q, want exact blocker message", runtime.BudgetBlocker.Message)
	}
	if runtime.BudgetBlocker.TriggeredAt != now {
		t.Fatalf("BudgetBlocker.TriggeredAt = %v, want %v", runtime.BudgetBlocker.TriggeredAt, now)
	}
	if len(runtime.AuditHistory) != 1 {
		t.Fatalf("AuditHistory count = %d, want 1", len(runtime.AuditHistory))
	}
	if runtime.AuditHistory[0].ToolName != "budget_exhausted" {
		t.Fatalf("AuditHistory[0].ToolName = %q, want %q", runtime.AuditHistory[0].ToolName, "budget_exhausted")
	}
	if runtime.AuditHistory[0].ActionClass != AuditActionClassRuntime {
		t.Fatalf("AuditHistory[0].ActionClass = %q, want %q", runtime.AuditHistory[0].ActionClass, AuditActionClassRuntime)
	}
	if runtime.AuditHistory[0].Result != AuditResultApplied {
		t.Fatalf("AuditHistory[0].Result = %q, want %q", runtime.AuditHistory[0].Result, AuditResultApplied)
	}
	if !runtime.AuditHistory[0].Allowed {
		t.Fatal("AuditHistory[0].Allowed = false, want true")
	}
	if runtime.AuditHistory[0].StepID != "final" {
		t.Fatalf("AuditHistory[0].StepID = %q, want %q", runtime.AuditHistory[0].StepID, "final")
	}
}

func TestPauseJobRuntimeForUnattendedWallClockPausesAtCeiling(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 28, 12, 0, 0, 0, time.UTC)
	runtime, exhausted, err := PauseJobRuntimeForUnattendedWallClock(JobRuntimeState{
		JobID:        "job-1",
		State:        JobStateRunning,
		ActiveStepID: "build",
		CreatedAt:    now.Add(-5 * time.Hour),
		StartedAt:    now.Add(-5 * time.Hour),
		ActiveStepAt: now.Add(-30 * time.Minute),
	}, now)
	if err != nil {
		t.Fatalf("PauseJobRuntimeForUnattendedWallClock() error = %v", err)
	}
	if !exhausted {
		t.Fatal("PauseJobRuntimeForUnattendedWallClock() exhausted = false, want true")
	}
	if runtime.State != JobStatePaused {
		t.Fatalf("State = %q, want %q", runtime.State, JobStatePaused)
	}
	if runtime.PausedReason != RuntimePauseReasonBudgetExhausted {
		t.Fatalf("PausedReason = %q, want %q", runtime.PausedReason, RuntimePauseReasonBudgetExhausted)
	}
	if runtime.BudgetBlocker == nil {
		t.Fatal("BudgetBlocker = nil, want persisted blocker")
	}
	if runtime.BudgetBlocker.Ceiling != unattendedWallClockBudgetCeiling {
		t.Fatalf("BudgetBlocker.Ceiling = %q, want %q", runtime.BudgetBlocker.Ceiling, unattendedWallClockBudgetCeiling)
	}
	if runtime.BudgetBlocker.Limit != maxUnattendedWallClockPerJobInMinutes {
		t.Fatalf("BudgetBlocker.Limit = %d, want %d", runtime.BudgetBlocker.Limit, maxUnattendedWallClockPerJobInMinutes)
	}
	if runtime.BudgetBlocker.Observed != 300 {
		t.Fatalf("BudgetBlocker.Observed = %d, want %d", runtime.BudgetBlocker.Observed, 300)
	}
	if runtime.BudgetBlocker.Message != "unattended wall-clock budget exhausted" {
		t.Fatalf("BudgetBlocker.Message = %q, want exact budget message", runtime.BudgetBlocker.Message)
	}
	if len(runtime.CompletedSteps) != 0 {
		t.Fatalf("CompletedSteps = %#v, want empty", runtime.CompletedSteps)
	}
}

func TestRecordFailedToolActionPausesAtCeiling(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 28, 12, 0, 0, 0, time.UTC)
	runtime := JobRuntimeState{
		JobID:        "job-1",
		State:        JobStateRunning,
		ActiveStepID: "build",
	}

	var exhausted bool
	var err error
	for i := 0; i < maxFailedActionsBeforePause; i++ {
		runtime, exhausted, err = RecordFailedToolAction(runtime, now.Add(time.Duration(i)*time.Minute), "message", "message tool: 'content' argument required")
		if err != nil {
			t.Fatalf("RecordFailedToolAction() step %d error = %v", i, err)
		}
	}

	if !exhausted {
		t.Fatal("RecordFailedToolAction() exhausted = false, want true on threshold")
	}
	if runtime.State != JobStatePaused {
		t.Fatalf("State = %q, want %q", runtime.State, JobStatePaused)
	}
	if runtime.BudgetBlocker == nil || runtime.BudgetBlocker.Ceiling != failedActionsBudgetCeiling {
		t.Fatalf("BudgetBlocker = %#v, want failed_actions blocker", runtime.BudgetBlocker)
	}
	if runtime.BudgetBlocker.Observed != maxFailedActionsBeforePause {
		t.Fatalf("BudgetBlocker.Observed = %d, want %d", runtime.BudgetBlocker.Observed, maxFailedActionsBeforePause)
	}
	if len(runtime.AuditHistory) != maxFailedActionsBeforePause+1 {
		t.Fatalf("AuditHistory count = %d, want %d including budget event", len(runtime.AuditHistory), maxFailedActionsBeforePause+1)
	}
	if runtime.AuditHistory[maxFailedActionsBeforePause].ToolName != "budget_exhausted" {
		t.Fatalf("final audit event tool = %q, want budget_exhausted", runtime.AuditHistory[maxFailedActionsBeforePause].ToolName)
	}
}

func TestRecordOwnerFacingSetStepAckPausesAtCeiling(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 29, 12, 0, 0, 0, time.UTC)
	runtime := JobRuntimeState{
		JobID:        "job-1",
		State:        JobStateRunning,
		ActiveStepID: "final",
	}
	for i := 0; i < 19; i++ {
		next, exhausted, err := RecordOwnerFacingMessage(runtime, now.Add(time.Duration(i)*time.Second))
		if err != nil {
			t.Fatalf("RecordOwnerFacingMessage() step %d error = %v", i, err)
		}
		if exhausted {
			t.Fatalf("RecordOwnerFacingMessage() step %d exhausted = true, want false before set-step acknowledgement", i)
		}
		runtime = next
	}

	runtime, exhausted, err := RecordOwnerFacingSetStepAck(runtime, now.Add(19*time.Second))
	if err != nil {
		t.Fatalf("RecordOwnerFacingSetStepAck() error = %v", err)
	}
	if !exhausted {
		t.Fatal("RecordOwnerFacingSetStepAck() exhausted = false, want true at threshold")
	}
	if runtime.State != JobStatePaused {
		t.Fatalf("State = %q, want %q", runtime.State, JobStatePaused)
	}
	if runtime.ActiveStepID != "final" {
		t.Fatalf("ActiveStepID = %q, want %q", runtime.ActiveStepID, "final")
	}
	if runtime.BudgetBlocker == nil || runtime.BudgetBlocker.Ceiling != ownerMessagesBudgetCeiling {
		t.Fatalf("BudgetBlocker = %#v, want owner_messages blocker", runtime.BudgetBlocker)
	}
	if got := runtime.AuditHistory[len(runtime.AuditHistory)-2].ToolName; got != ownerFacingSetStepAckAction {
		t.Fatalf("penultimate audit tool = %q, want %q", got, ownerFacingSetStepAckAction)
	}
	if got := runtime.AuditHistory[len(runtime.AuditHistory)-1].ToolName; got != "budget_exhausted" {
		t.Fatalf("last audit tool = %q, want %q", got, "budget_exhausted")
	}
}

func TestRecordOwnerFacingMessagePausesAtCeiling(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 28, 12, 10, 0, 0, time.UTC)
	runtime := JobRuntimeState{
		JobID:        "job-1",
		State:        JobStateRunning,
		ActiveStepID: "build",
	}

	var exhausted bool
	var err error
	for i := 0; i < maxOwnerFacingMessagesPerJob; i++ {
		runtime, exhausted, err = RecordOwnerFacingMessage(runtime, now.Add(time.Duration(i)*time.Minute))
		if err != nil {
			t.Fatalf("RecordOwnerFacingMessage() step %d error = %v", i, err)
		}
	}

	if !exhausted {
		t.Fatal("RecordOwnerFacingMessage() exhausted = false, want true on threshold")
	}
	if runtime.State != JobStatePaused {
		t.Fatalf("State = %q, want %q", runtime.State, JobStatePaused)
	}
	if runtime.BudgetBlocker == nil || runtime.BudgetBlocker.Ceiling != ownerMessagesBudgetCeiling {
		t.Fatalf("BudgetBlocker = %#v, want owner_messages blocker", runtime.BudgetBlocker)
	}
	if runtime.BudgetBlocker.Observed != maxOwnerFacingMessagesPerJob {
		t.Fatalf("BudgetBlocker.Observed = %d, want %d", runtime.BudgetBlocker.Observed, maxOwnerFacingMessagesPerJob)
	}
	if len(runtime.AuditHistory) != maxOwnerFacingMessagesPerJob+1 {
		t.Fatalf("AuditHistory count = %d, want %d including budget event", len(runtime.AuditHistory), maxOwnerFacingMessagesPerJob+1)
	}
	if runtime.AuditHistory[maxOwnerFacingMessagesPerJob].ToolName != "budget_exhausted" {
		t.Fatalf("final audit event tool = %q, want budget_exhausted", runtime.AuditHistory[maxOwnerFacingMessagesPerJob].ToolName)
	}
}

func TestRecordOwnerFacingStepOutputPausesAtCeilingWithPriorMessages(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 29, 9, 0, 0, 0, time.UTC)
	runtime := JobRuntimeState{
		JobID:        "job-1",
		State:        JobStateRunning,
		ActiveStepID: "build",
	}

	for i := 0; i < maxOwnerFacingMessagesPerJob-1; i++ {
		var err error
		runtime, _, err = RecordOwnerFacingMessage(runtime, now.Add(time.Duration(i)*time.Minute))
		if err != nil {
			t.Fatalf("RecordOwnerFacingMessage() step %d error = %v", i, err)
		}
	}

	runtime, exhausted, err := RecordOwnerFacingStepOutput(runtime, now.Add(30*time.Minute))
	if err != nil {
		t.Fatalf("RecordOwnerFacingStepOutput() error = %v", err)
	}
	if !exhausted {
		t.Fatal("RecordOwnerFacingStepOutput() exhausted = false, want true at threshold")
	}
	if runtime.State != JobStatePaused {
		t.Fatalf("State = %q, want %q", runtime.State, JobStatePaused)
	}
	if runtime.BudgetBlocker == nil || runtime.BudgetBlocker.Ceiling != ownerMessagesBudgetCeiling {
		t.Fatalf("BudgetBlocker = %#v, want owner_messages blocker", runtime.BudgetBlocker)
	}
	if runtime.BudgetBlocker.Observed != maxOwnerFacingMessagesPerJob {
		t.Fatalf("BudgetBlocker.Observed = %d, want %d", runtime.BudgetBlocker.Observed, maxOwnerFacingMessagesPerJob)
	}
	if len(runtime.AuditHistory) != maxOwnerFacingMessagesPerJob+1 {
		t.Fatalf("AuditHistory count = %d, want %d including budget event", len(runtime.AuditHistory), maxOwnerFacingMessagesPerJob+1)
	}
	if runtime.AuditHistory[maxOwnerFacingMessagesPerJob-1].ToolName != ownerFacingStepOutputAction {
		t.Fatalf("step-output audit tool = %q, want %q", runtime.AuditHistory[maxOwnerFacingMessagesPerJob-1].ToolName, ownerFacingStepOutputAction)
	}
	if runtime.AuditHistory[maxOwnerFacingMessagesPerJob-1].ActionClass != AuditActionClassRuntime {
		t.Fatalf("step-output audit class = %q, want %q", runtime.AuditHistory[maxOwnerFacingMessagesPerJob-1].ActionClass, AuditActionClassRuntime)
	}
	if runtime.AuditHistory[maxOwnerFacingMessagesPerJob].ToolName != "budget_exhausted" {
		t.Fatalf("final audit event tool = %q, want budget_exhausted", runtime.AuditHistory[maxOwnerFacingMessagesPerJob].ToolName)
	}
}

func TestResumePausedJobRuntimeClearsBudgetBlocker(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 28, 12, 5, 0, 0, time.UTC)
	runtime, err := ResumePausedJobRuntime(JobRuntimeState{
		JobID:        "job-1",
		State:        JobStatePaused,
		ActiveStepID: "build",
		PausedReason: RuntimePauseReasonBudgetExhausted,
		PausedAt:     time.Date(2026, 3, 28, 12, 0, 0, 0, time.UTC),
		BudgetBlocker: &RuntimeBudgetBlockerRecord{
			Ceiling:     "owner_messages",
			Limit:       20,
			Observed:    20,
			Message:     "owner-facing message budget exhausted",
			TriggeredAt: time.Date(2026, 3, 28, 12, 0, 0, 0, time.UTC),
		},
	}, now)
	if err != nil {
		t.Fatalf("ResumePausedJobRuntime() error = %v", err)
	}

	if runtime.State != JobStateRunning {
		t.Fatalf("State = %q, want %q", runtime.State, JobStateRunning)
	}
	if runtime.BudgetBlocker != nil {
		t.Fatalf("BudgetBlocker = %#v, want nil after resume", runtime.BudgetBlocker)
	}
}

func TestSetJobRuntimeActiveStepRejectsPreviouslyCompletedStepReplay(t *testing.T) {
	t.Parallel()

	job := testExecutionJob()
	_, err := SetJobRuntimeActiveStep(job, &JobRuntimeState{
		JobID:        job.ID,
		State:        JobStatePaused,
		ActiveStepID: "final",
		CompletedSteps: []RuntimeStepRecord{
			{StepID: "build", At: time.Date(2026, 3, 27, 12, 0, 0, 0, time.UTC)},
		},
	}, "build", time.Date(2026, 3, 27, 12, 5, 0, 0, time.UTC))
	if err == nil {
		t.Fatal("SetJobRuntimeActiveStep() error = nil, want completed-step replay rejection")
	}

	validationErr, ok := err.(ValidationError)
	if !ok {
		t.Fatalf("SetJobRuntimeActiveStep() error type = %T, want ValidationError", err)
	}
	if validationErr.Code != RejectionCodeInvalidRuntimeState {
		t.Fatalf("ValidationError.Code = %q, want %q", validationErr.Code, RejectionCodeInvalidRuntimeState)
	}
	if validationErr.StepID != "build" {
		t.Fatalf("ValidationError.StepID = %q, want %q", validationErr.StepID, "build")
	}
}

func TestSetJobRuntimeActiveStepRejectsPreviouslyFailedStepReplay(t *testing.T) {
	t.Parallel()

	job := testExecutionJob()
	_, err := SetJobRuntimeActiveStep(job, &JobRuntimeState{
		JobID:        job.ID,
		State:        JobStatePaused,
		ActiveStepID: "final",
		FailedSteps: []RuntimeStepRecord{
			{StepID: "build", Reason: "validator failed", At: time.Date(2026, 3, 27, 12, 0, 0, 0, time.UTC)},
		},
	}, "build", time.Date(2026, 3, 27, 12, 5, 0, 0, time.UTC))
	if err == nil {
		t.Fatal("SetJobRuntimeActiveStep() error = nil, want failed-step replay rejection")
	}

	validationErr, ok := err.(ValidationError)
	if !ok {
		t.Fatalf("SetJobRuntimeActiveStep() error type = %T, want ValidationError", err)
	}
	if validationErr.Code != RejectionCodeInvalidRuntimeState {
		t.Fatalf("ValidationError.Code = %q, want %q", validationErr.Code, RejectionCodeInvalidRuntimeState)
	}
	if validationErr.StepID != "build" {
		t.Fatalf("ValidationError.StepID = %q, want %q", validationErr.StepID, "build")
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

func TestResumePausedJobRuntimeRejectsCompletedActiveStepReplay(t *testing.T) {
	t.Parallel()

	_, err := ResumePausedJobRuntime(JobRuntimeState{
		JobID:        "job-1",
		State:        JobStatePaused,
		ActiveStepID: "build",
		CompletedSteps: []RuntimeStepRecord{
			{StepID: "build", At: time.Date(2026, 3, 27, 12, 0, 0, 0, time.UTC)},
		},
	}, time.Date(2026, 3, 27, 12, 5, 0, 0, time.UTC))
	if err == nil {
		t.Fatal("ResumePausedJobRuntime() error = nil, want completed-step replay rejection")
	}

	validationErr, ok := err.(ValidationError)
	if !ok {
		t.Fatalf("ResumePausedJobRuntime() error type = %T, want ValidationError", err)
	}
	if validationErr.Code != RejectionCodeInvalidRuntimeState {
		t.Fatalf("ValidationError.Code = %q, want %q", validationErr.Code, RejectionCodeInvalidRuntimeState)
	}
	if validationErr.StepID != "build" {
		t.Fatalf("ValidationError.StepID = %q, want %q", validationErr.StepID, "build")
	}
}

func TestResumePausedJobRuntimeRejectsFailedActiveStepReplay(t *testing.T) {
	t.Parallel()

	_, err := ResumePausedJobRuntime(JobRuntimeState{
		JobID:        "job-1",
		State:        JobStatePaused,
		ActiveStepID: "build",
		FailedSteps: []RuntimeStepRecord{
			{StepID: "build", At: time.Date(2026, 3, 27, 12, 0, 0, 0, time.UTC)},
		},
	}, time.Date(2026, 3, 27, 12, 5, 0, 0, time.UTC))
	if err == nil {
		t.Fatal("ResumePausedJobRuntime() error = nil, want failed-step replay rejection")
	}

	validationErr, ok := err.(ValidationError)
	if !ok {
		t.Fatalf("ResumePausedJobRuntime() error type = %T, want ValidationError", err)
	}
	if validationErr.Code != RejectionCodeInvalidRuntimeState {
		t.Fatalf("ValidationError.Code = %q, want %q", validationErr.Code, RejectionCodeInvalidRuntimeState)
	}
	if validationErr.StepID != "build" {
		t.Fatalf("ValidationError.StepID = %q, want %q", validationErr.StepID, "build")
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

func TestHasCompletedRuntimeStepMatchesRecordedCompletion(t *testing.T) {
	t.Parallel()

	runtime := JobRuntimeState{
		JobID:        "job-1",
		State:        JobStatePaused,
		ActiveStepID: "final",
		CompletedSteps: []RuntimeStepRecord{
			{StepID: "build", At: time.Date(2026, 3, 24, 12, 30, 0, 0, time.UTC)},
		},
	}

	if !HasCompletedRuntimeStep(runtime, "build") {
		t.Fatal("HasCompletedRuntimeStep(build) = false, want true")
	}
	if HasCompletedRuntimeStep(runtime, "final") {
		t.Fatal("HasCompletedRuntimeStep(final) = true, want false")
	}
}

func TestHasFailedRuntimeStepMatchesRecordedFailure(t *testing.T) {
	t.Parallel()

	runtime := JobRuntimeState{
		JobID:        "job-1",
		State:        JobStatePaused,
		ActiveStepID: "final",
		FailedSteps: []RuntimeStepRecord{
			{StepID: "build", Reason: "validator failed", At: time.Date(2026, 3, 24, 12, 30, 0, 0, time.UTC)},
		},
	}

	if !HasFailedRuntimeStep(runtime, "build") {
		t.Fatal("HasFailedRuntimeStep(build) = false, want true")
	}
	if HasFailedRuntimeStep(runtime, "final") {
		t.Fatal("HasFailedRuntimeStep(final) = true, want false")
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

func TestValidateRuntimeExecutionRejectsCompletedActiveStepReplay(t *testing.T) {
	t.Parallel()

	err := ValidateRuntimeExecution(ExecutionContext{
		Job:  &Job{ID: "job-1"},
		Step: &Step{ID: "build"},
		Runtime: &JobRuntimeState{
			JobID:        "job-1",
			State:        JobStateRunning,
			ActiveStepID: "build",
			CompletedSteps: []RuntimeStepRecord{
				{StepID: "build", At: time.Date(2026, 3, 27, 12, 0, 0, 0, time.UTC)},
			},
		},
	})
	if err == nil {
		t.Fatal("ValidateRuntimeExecution() error = nil, want completed-step replay rejection")
	}

	validationErr, ok := err.(ValidationError)
	if !ok {
		t.Fatalf("ValidateRuntimeExecution() error type = %T, want ValidationError", err)
	}
	if validationErr.Code != RejectionCodeInvalidRuntimeState {
		t.Fatalf("ValidationError.Code = %q, want %q", validationErr.Code, RejectionCodeInvalidRuntimeState)
	}
	if validationErr.StepID != "build" {
		t.Fatalf("ValidationError.StepID = %q, want %q", validationErr.StepID, "build")
	}
}

func TestResolveExecutionContextWithRuntimeIncludesIndependentRuntimeCopy(t *testing.T) {
	t.Parallel()

	job := testExecutionJob()
	runtime := JobRuntimeState{
		JobID:        job.ID,
		State:        JobStateRunning,
		ActiveStepID: "build",
		BudgetBlocker: &RuntimeBudgetBlockerRecord{
			Ceiling:     "owner_messages",
			Limit:       20,
			Observed:    20,
			Message:     "owner-facing message budget exhausted",
			TriggeredAt: time.Date(2026, 3, 23, 11, 20, 0, 0, time.UTC),
		},
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
	ec.Runtime.BudgetBlocker.Ceiling = "mutated-budget"
	ec.Runtime.ApprovalRequests[0].StepID = "mutated-request"
	ec.Runtime.ApprovalRequests[0].Content.ProposedAction = "mutated-content"
	ec.Runtime.ApprovalGrants[0].StepID = "mutated-grant"
	if runtime.CompletedSteps[0].StepID != "draft" {
		t.Fatalf("original runtime step = %q, want %q", runtime.CompletedSteps[0].StepID, "draft")
	}
	if runtime.BudgetBlocker == nil || runtime.BudgetBlocker.Ceiling != "owner_messages" {
		t.Fatalf("original runtime budget blocker = %#v, want preserved blocker", runtime.BudgetBlocker)
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

func TestResolveExecutionContextWithRuntimeRejectsFailedActiveStepReplay(t *testing.T) {
	t.Parallel()

	job := testExecutionJob()

	_, err := ResolveExecutionContextWithRuntime(job, JobRuntimeState{
		JobID:        job.ID,
		State:        JobStateRunning,
		ActiveStepID: "build",
		FailedSteps: []RuntimeStepRecord{
			{StepID: "build", At: time.Date(2026, 3, 27, 12, 0, 0, 0, time.UTC)},
		},
	})
	if err == nil {
		t.Fatal("ResolveExecutionContextWithRuntime() error = nil, want failed-step replay rejection")
	}

	validationErr, ok := err.(ValidationError)
	if !ok {
		t.Fatalf("ResolveExecutionContextWithRuntime() error type = %T, want ValidationError", err)
	}
	if validationErr.Code != RejectionCodeInvalidRuntimeState {
		t.Fatalf("ValidationError.Code = %q, want %q", validationErr.Code, RejectionCodeInvalidRuntimeState)
	}
	if validationErr.StepID != "build" {
		t.Fatalf("ValidationError.StepID = %q, want %q", validationErr.StepID, "build")
	}
}
