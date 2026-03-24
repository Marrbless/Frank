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

	state.ClearExecutionContext()

	handled, resp, err := state.ApplyNaturalApprovalDecision("no")
	if err != nil {
		t.Fatalf("ApplyNaturalApprovalDecision(no) error = %v", err)
	}
	if !handled {
		t.Fatal("ApplyNaturalApprovalDecision(no) handled = false, want true")
	}
	if resp != "Denied job=job-1 step=build." {
		t.Fatalf("ApplyNaturalApprovalDecision(no) response = %q, want denial acknowledgement", resp)
	}

	runtime, ok := state.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if runtime.State != missioncontrol.JobStateWaitingUser {
		t.Fatalf("MissionRuntimeState().State = %q, want %q", runtime.State, missioncontrol.JobStateWaitingUser)
	}
	if len(runtime.ApprovalRequests) != 1 || runtime.ApprovalRequests[0].State != missioncontrol.ApprovalStateDenied {
		t.Fatalf("MissionRuntimeState().ApprovalRequests = %#v, want one denied approval", runtime.ApprovalRequests)
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
