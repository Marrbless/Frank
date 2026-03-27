package missioncontrol

import (
	"reflect"
	"testing"
	"time"
)

func TestValidatePlanAcceptsSystemActionStepWithExplicitMetadata(t *testing.T) {
	t.Parallel()

	job := testV2Job([]Step{
		{
			ID:           "status-service",
			Type:         StepTypeSystemAction,
			AllowedTools: []string{"exec"},
			SystemAction: &SystemAction{
				Kind:      SystemActionKindService,
				Operation: SystemActionOperationStatus,
				Target:    "demo-service",
				Command:   []string{"echo", "status demo-service"},
				PostState: &SystemActionPostState{
					Command:         []string{"echo", "demo-service state=running"},
					SuccessContains: []string{"state=running"},
				},
			},
		},
		{ID: "final", Type: StepTypeFinalResponse, DependsOn: []string{"status-service"}},
	})
	job.AllowedTools = []string{"exec", "read", "write"}

	if errors := ValidatePlan(job); len(errors) != 0 {
		t.Fatalf("ValidatePlan() = %#v, want no errors", errors)
	}
}

func TestValidatePlanRejectsSystemActionWithoutV2SpecVersion(t *testing.T) {
	t.Parallel()

	job := testJob([]Step{
		{
			ID:           "status-service",
			Type:         StepTypeSystemAction,
			AllowedTools: []string{"exec"},
			SystemAction: &SystemAction{
				Kind:      SystemActionKindService,
				Operation: SystemActionOperationStatus,
				Target:    "demo-service",
				Command:   []string{"echo", "status demo-service"},
				PostState: &SystemActionPostState{
					Command:         []string{"echo", "demo-service state=running"},
					SuccessContains: []string{"state=running"},
				},
			},
		},
		{ID: "final", Type: StepTypeFinalResponse, DependsOn: []string{"status-service"}},
	})
	job.AllowedTools = []string{"exec", "read", "write"}

	errors := ValidatePlan(job)
	want := []ValidationError{{
		Code:    RejectionCodeInvalidStepType,
		StepID:  "status-service",
		Message: `step type "system_action" requires job spec_version frank_v2`,
	}}
	if !reflect.DeepEqual(errors, want) {
		t.Fatalf("ValidatePlan() = %#v, want %#v", errors, want)
	}
}

func TestValidatePlanRejectsSystemActionWithoutMetadata(t *testing.T) {
	t.Parallel()

	job := testV2Job([]Step{
		{ID: "status-service", Type: StepTypeSystemAction, AllowedTools: []string{"exec"}},
		{ID: "final", Type: StepTypeFinalResponse, DependsOn: []string{"status-service"}},
	})
	job.AllowedTools = []string{"exec", "read", "write"}

	errors := ValidatePlan(job)
	want := []ValidationError{{
		Code:    RejectionCodeInvalidStepType,
		StepID:  "status-service",
		Message: "system_action step requires explicit system_action metadata",
	}}
	if !reflect.DeepEqual(errors, want) {
		t.Fatalf("ValidatePlan() = %#v, want %#v", errors, want)
	}
}

func TestValidatePlanRejectsSystemActionWithoutPostStateContract(t *testing.T) {
	t.Parallel()

	job := testV2Job([]Step{
		{
			ID:           "status-service",
			Type:         StepTypeSystemAction,
			AllowedTools: []string{"exec"},
			SystemAction: &SystemAction{
				Kind:      SystemActionKindService,
				Operation: SystemActionOperationStatus,
				Target:    "demo-service",
				Command:   []string{"echo", "status demo-service"},
			},
		},
		{ID: "final", Type: StepTypeFinalResponse, DependsOn: []string{"status-service"}},
	})
	job.AllowedTools = []string{"exec", "read", "write"}

	errors := ValidatePlan(job)
	want := []ValidationError{{
		Code:    RejectionCodeInvalidStepType,
		StepID:  "status-service",
		Message: "system_action step requires explicit post_state verification contract",
	}}
	if !reflect.DeepEqual(errors, want) {
		t.Fatalf("ValidatePlan() = %#v, want %#v", errors, want)
	}
}

func TestCompleteRuntimeStepSystemActionRecordsVerifiedStateAndRollback(t *testing.T) {
	t.Parallel()

	job := testV2Job([]Step{
		{
			ID:           "start-service",
			Type:         StepTypeSystemAction,
			AllowedTools: []string{"exec"},
			SystemAction: &SystemAction{
				Kind:      SystemActionKindService,
				Operation: SystemActionOperationStart,
				Target:    "demo-service",
				Command:   []string{"democtl", "start", "demo-service"},
				PostState: &SystemActionPostState{
					Command:         []string{"democtl", "status", "demo-service"},
					SuccessContains: []string{"state=running"},
					FailureContains: []string{"state=stopped"},
				},
				Rollback: &SystemActionRollback{
					Command: []string{"democtl", "stop", "demo-service"},
					Reason:  "stop the service if later validation fails",
				},
			},
		},
	})
	job.AllowedTools = []string{"exec", "read", "write"}

	ec := testStepValidationExecutionContextForJob(job, "start-service", JobStateRunning)
	now := time.Date(2026, 3, 26, 14, 0, 0, 0, time.UTC)

	runtime, err := CompleteRuntimeStep(ec, now, StepValidationInput{
		FinalResponse: "Started the local service and verified it is running.",
		SuccessfulTools: []RuntimeToolCallEvidence{
			{ToolName: "exec", Arguments: map[string]interface{}{"cmd": []string{"democtl", "start", "demo-service"}}, Result: "started"},
			{ToolName: "exec", Arguments: map[string]interface{}{"cmd": []string{"democtl", "status", "demo-service"}}, Result: "demo-service state=running pid=1234"},
		},
	})
	if err != nil {
		t.Fatalf("CompleteRuntimeStep() error = %v", err)
	}
	if runtime.State != JobStatePaused {
		t.Fatalf("State = %q, want %q", runtime.State, JobStatePaused)
	}
	if len(runtime.CompletedSteps) != 1 {
		t.Fatalf("CompletedSteps = %#v, want one completed system_action", runtime.CompletedSteps)
	}
	record := runtime.CompletedSteps[0]
	if record.StepID != "start-service" {
		t.Fatalf("CompletedSteps[0].StepID = %q, want %q", record.StepID, "start-service")
	}
	if record.ResultingState == nil {
		t.Fatal("CompletedSteps[0].ResultingState = nil, want durable resulting state")
	}
	if record.ResultingState.State != "running" {
		t.Fatalf("CompletedSteps[0].ResultingState.State = %q, want %q", record.ResultingState.State, "running")
	}
	if record.ResultingState.Target != "demo-service" {
		t.Fatalf("CompletedSteps[0].ResultingState.Target = %q, want %q", record.ResultingState.Target, "demo-service")
	}
	if record.ResultingState.VerificationOutput != "demo-service state=running pid=1234" {
		t.Fatalf("CompletedSteps[0].ResultingState.VerificationOutput = %q, want verified status output", record.ResultingState.VerificationOutput)
	}
	if record.Rollback == nil {
		t.Fatal("CompletedSteps[0].Rollback = nil, want durable rollback record")
	}
	if !record.Rollback.Available {
		t.Fatalf("CompletedSteps[0].Rollback.Available = false, want true")
	}
	if !reflect.DeepEqual(record.Rollback.Command, []string{"democtl", "stop", "demo-service"}) {
		t.Fatalf("CompletedSteps[0].Rollback.Command = %#v, want stop command", record.Rollback.Command)
	}
}

func TestCompleteRuntimeStepSystemActionRecordsAlreadyPresentWhenPostStateAlreadyVerifies(t *testing.T) {
	t.Parallel()

	job := testV2Job([]Step{
		{
			ID:           "start-service",
			Type:         StepTypeSystemAction,
			AllowedTools: []string{"exec"},
			SystemAction: &SystemAction{
				Kind:      SystemActionKindService,
				Operation: SystemActionOperationStart,
				Target:    "demo-service",
				Command:   []string{"democtl", "start", "demo-service"},
				PostState: &SystemActionPostState{
					Command:         []string{"democtl", "status", "demo-service"},
					SuccessContains: []string{"state=running"},
					FailureContains: []string{"state=stopped"},
				},
				Rollback: &SystemActionRollback{
					Command: []string{"democtl", "stop", "demo-service"},
					Reason:  "stop the service if later validation fails",
				},
			},
		},
	})
	job.AllowedTools = []string{"exec", "read", "write"}

	ec := testStepValidationExecutionContextForJob(job, "start-service", JobStateRunning)
	now := time.Date(2026, 3, 27, 15, 0, 0, 0, time.UTC)

	runtime, err := CompleteRuntimeStep(ec, now, StepValidationInput{
		FinalResponse: "The service was already running, so I verified the post-state without reissuing start.",
		SuccessfulTools: []RuntimeToolCallEvidence{
			{ToolName: "exec", Arguments: map[string]interface{}{"cmd": []string{"democtl", "status", "demo-service"}}, Result: "demo-service state=running pid=1234"},
		},
	})
	if err != nil {
		t.Fatalf("CompleteRuntimeStep() error = %v", err)
	}
	if runtime.State != JobStatePaused {
		t.Fatalf("State = %q, want %q", runtime.State, JobStatePaused)
	}
	if len(runtime.CompletedSteps) != 1 {
		t.Fatalf("CompletedSteps = %#v, want one completed system_action", runtime.CompletedSteps)
	}
	record := runtime.CompletedSteps[0]
	if record.StepID != "start-service" {
		t.Fatalf("CompletedSteps[0].StepID = %q, want %q", record.StepID, "start-service")
	}
	if record.ResultingState == nil {
		t.Fatal("CompletedSteps[0].ResultingState = nil, want durable already_present state")
	}
	if record.ResultingState.Kind != string(StepTypeSystemAction) {
		t.Fatalf("CompletedSteps[0].ResultingState.Kind = %q, want %q", record.ResultingState.Kind, StepTypeSystemAction)
	}
	if record.ResultingState.State != "already_present" {
		t.Fatalf("CompletedSteps[0].ResultingState.State = %q, want %q", record.ResultingState.State, "already_present")
	}
	if record.ResultingState.Target != "demo-service" {
		t.Fatalf("CompletedSteps[0].ResultingState.Target = %q, want %q", record.ResultingState.Target, "demo-service")
	}
	if !reflect.DeepEqual(record.ResultingState.ActionCommand, []string{"democtl", "start", "demo-service"}) {
		t.Fatalf("CompletedSteps[0].ResultingState.ActionCommand = %#v, want start command", record.ResultingState.ActionCommand)
	}
	if !reflect.DeepEqual(record.ResultingState.VerificationCommand, []string{"democtl", "status", "demo-service"}) {
		t.Fatalf("CompletedSteps[0].ResultingState.VerificationCommand = %#v, want status command", record.ResultingState.VerificationCommand)
	}
	if record.Rollback != nil {
		t.Fatalf("CompletedSteps[0].Rollback = %#v, want nil when no new action was applied", record.Rollback)
	}
}

func TestCompleteRuntimeStepSystemActionStatusStillRequiresExplicitCommandExecution(t *testing.T) {
	t.Parallel()

	job := testV2Job([]Step{
		{
			ID:           "status-service",
			Type:         StepTypeSystemAction,
			AllowedTools: []string{"exec"},
			SystemAction: &SystemAction{
				Kind:      SystemActionKindService,
				Operation: SystemActionOperationStatus,
				Target:    "demo-service",
				Command:   []string{"democtl", "show", "demo-service"},
				PostState: &SystemActionPostState{
					Command:         []string{"democtl", "status", "demo-service"},
					SuccessContains: []string{"state=running"},
				},
			},
		},
	})
	job.AllowedTools = []string{"exec", "read", "write"}

	ec := testStepValidationExecutionContextForJob(job, "status-service", JobStateRunning)

	_, err := CompleteRuntimeStep(ec, time.Date(2026, 3, 27, 15, 1, 0, 0, time.UTC), StepValidationInput{
		FinalResponse: "Verified the service state from the post-state command only.",
		SuccessfulTools: []RuntimeToolCallEvidence{
			{ToolName: "exec", Arguments: map[string]interface{}{"cmd": []string{"democtl", "status", "demo-service"}}, Result: "demo-service state=running pid=1234"},
		},
	})
	if err == nil {
		t.Fatal("CompleteRuntimeStep() error = nil, want explicit action execution rejection")
	}
	validationErr, ok := err.(ValidationError)
	if !ok {
		t.Fatalf("CompleteRuntimeStep() error = %T, want ValidationError", err)
	}
	if validationErr.Code != RejectionCodeStepValidationFailed {
		t.Fatalf("ValidationError.Code = %q, want %q", validationErr.Code, RejectionCodeStepValidationFailed)
	}
	if validationErr.Message != `system_action completion requires executing "democtl show demo-service"` {
		t.Fatalf("ValidationError.Message = %q, want explicit action execution failure", validationErr.Message)
	}
}

func TestCompleteRuntimeStepSystemActionRejectsMissingPostStateVerification(t *testing.T) {
	t.Parallel()

	job := testV2Job([]Step{
		{
			ID:           "start-service",
			Type:         StepTypeSystemAction,
			AllowedTools: []string{"exec"},
			SystemAction: &SystemAction{
				Kind:      SystemActionKindService,
				Operation: SystemActionOperationStart,
				Target:    "demo-service",
				Command:   []string{"democtl", "start", "demo-service"},
				PostState: &SystemActionPostState{
					Command:         []string{"democtl", "status", "demo-service"},
					SuccessContains: []string{"state=running"},
				},
			},
		},
	})
	job.AllowedTools = []string{"exec", "read", "write"}

	ec := testStepValidationExecutionContextForJob(job, "start-service", JobStateRunning)

	_, err := CompleteRuntimeStep(ec, time.Date(2026, 3, 26, 14, 1, 0, 0, time.UTC), StepValidationInput{
		FinalResponse: "Issued the start command.",
		SuccessfulTools: []RuntimeToolCallEvidence{
			{ToolName: "exec", Arguments: map[string]interface{}{"cmd": []string{"democtl", "start", "demo-service"}}, Result: "started"},
		},
	})
	if err == nil {
		t.Fatal("CompleteRuntimeStep() error = nil, want missing post-state verification rejection")
	}
	validationErr, ok := err.(ValidationError)
	if !ok {
		t.Fatalf("CompleteRuntimeStep() error = %T, want ValidationError", err)
	}
	if validationErr.Code != RejectionCodeStepValidationFailed {
		t.Fatalf("ValidationError.Code = %q, want %q", validationErr.Code, RejectionCodeStepValidationFailed)
	}
	if validationErr.Message != `system_action completion requires verification command "democtl status demo-service"` {
		t.Fatalf("ValidationError.Message = %q, want verification-command failure", validationErr.Message)
	}
}

func TestCompleteRuntimeStepSystemActionRejectsFailingPostStateVerification(t *testing.T) {
	t.Parallel()

	job := testV2Job([]Step{
		{
			ID:           "stop-service",
			Type:         StepTypeSystemAction,
			AllowedTools: []string{"exec"},
			SystemAction: &SystemAction{
				Kind:      SystemActionKindProcess,
				Operation: SystemActionOperationStop,
				Target:    "demo-process",
				Command:   []string{"democtl", "stop", "demo-process"},
				PostState: &SystemActionPostState{
					Command:         []string{"democtl", "status", "demo-process"},
					SuccessContains: []string{"state=stopped"},
					FailureContains: []string{"state=running"},
				},
			},
		},
	})
	job.AllowedTools = []string{"exec", "read", "write"}

	ec := testStepValidationExecutionContextForJob(job, "stop-service", JobStateRunning)

	_, err := CompleteRuntimeStep(ec, time.Date(2026, 3, 26, 14, 2, 0, 0, time.UTC), StepValidationInput{
		FinalResponse: "Stopped the process.",
		SuccessfulTools: []RuntimeToolCallEvidence{
			{ToolName: "exec", Arguments: map[string]interface{}{"cmd": []string{"democtl", "stop", "demo-process"}}, Result: "stopped"},
			{ToolName: "exec", Arguments: map[string]interface{}{"cmd": []string{"democtl", "status", "demo-process"}}, Result: "demo-process state=running pid=1234"},
		},
	})
	if err == nil {
		t.Fatal("CompleteRuntimeStep() error = nil, want failing post-state verification rejection")
	}
	validationErr, ok := err.(ValidationError)
	if !ok {
		t.Fatalf("CompleteRuntimeStep() error = %T, want ValidationError", err)
	}
	if validationErr.Code != RejectionCodeStepValidationFailed {
		t.Fatalf("ValidationError.Code = %q, want %q", validationErr.Code, RejectionCodeStepValidationFailed)
	}
	if validationErr.Message != `system_action post-state verification for "demo-process" is missing "state=stopped"` {
		t.Fatalf("ValidationError.Message = %q, want missing expected state failure", validationErr.Message)
	}
}

func TestCompleteRuntimeStepSystemActionStartCanReuseLongRunningCodeStartupBoundary(t *testing.T) {
	t.Parallel()

	job := testV2Job([]Step{
		{
			ID:                        "build-service",
			Type:                      StepTypeLongRunningCode,
			AllowedTools:              []string{"exec", "read", "write"},
			LongRunningStartupCommand: []string{"npm", "start"},
			LongRunningArtifactPath:   "dist/service.js",
			SuccessCriteria:           []string{"Build dist/service.js and record npm start."},
		},
		{
			ID:           "start-service",
			Type:         StepTypeSystemAction,
			AllowedTools: []string{"exec"},
			DependsOn:    []string{"build-service"},
			SystemAction: &SystemAction{
				Kind:         SystemActionKindService,
				Operation:    SystemActionOperationStart,
				Target:       "dist/service.js",
				SourceStepID: "build-service",
				PostState: &SystemActionPostState{
					Command:         []string{"pgrep", "-f", "dist/service.js"},
					SuccessContains: []string{"4242"},
				},
				Rollback: &SystemActionRollback{
					Command: []string{"pkill", "-f", "dist/service.js"},
				},
			},
		},
	})
	job.AllowedTools = []string{"exec", "read", "write"}

	ec := testStepValidationExecutionContextForJob(job, "start-service", JobStateRunning)
	runtime, err := CompleteRuntimeStep(ec, time.Date(2026, 3, 26, 14, 3, 0, 0, time.UTC), StepValidationInput{
		FinalResponse: "Started the built service from the recorded startup contract.",
		SuccessfulTools: []RuntimeToolCallEvidence{
			{ToolName: "exec", Arguments: map[string]interface{}{"cmd": []string{"npm", "start"}}, Result: "started"},
			{ToolName: "exec", Arguments: map[string]interface{}{"cmd": []string{"pgrep", "-f", "dist/service.js"}}, Result: "4242"},
		},
	})
	if err != nil {
		t.Fatalf("CompleteRuntimeStep() error = %v", err)
	}
	if len(runtime.CompletedSteps) != 1 {
		t.Fatalf("CompletedSteps = %#v, want one completed system_action step", runtime.CompletedSteps)
	}
	if runtime.CompletedSteps[0].ResultingState == nil {
		t.Fatal("CompletedSteps[0].ResultingState = nil, want durable resulting state")
	}
	if !reflect.DeepEqual(runtime.CompletedSteps[0].ResultingState.ActionCommand, []string{"npm", "start"}) {
		t.Fatalf("CompletedSteps[0].ResultingState.ActionCommand = %#v, want long_running_code startup command", runtime.CompletedSteps[0].ResultingState.ActionCommand)
	}
	if runtime.CompletedSteps[0].ResultingState.SourceStepID != "build-service" {
		t.Fatalf("CompletedSteps[0].ResultingState.SourceStepID = %q, want %q", runtime.CompletedSteps[0].ResultingState.SourceStepID, "build-service")
	}
}
