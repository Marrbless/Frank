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
	if len(runtime.ApprovalRequests) != 0 {
		t.Fatalf("ApprovalRequests = %#v, want empty for blocker discussion", runtime.ApprovalRequests)
	}
	if len(runtime.CompletedSteps) != 0 {
		t.Fatalf("CompletedSteps = %#v, want empty", runtime.CompletedSteps)
	}
}

func TestCompleteRuntimeStepAuthorizationTransitionsToWaitingUserWithPendingApproval(t *testing.T) {
	t.Parallel()

	ec := testStepValidationExecutionContext(Step{
		ID:      "discuss",
		Type:    StepTypeDiscussion,
		Subtype: StepSubtypeAuthorization,
	}, JobStateRunning)
	now := time.Date(2026, 3, 23, 12, 0, 30, 0, time.UTC)

	runtime, err := CompleteRuntimeStep(ec, now, StepValidationInput{FinalResponse: "Need approval before continuing."})
	if err != nil {
		t.Fatalf("CompleteRuntimeStep() error = %v", err)
	}

	if len(runtime.ApprovalRequests) != 1 {
		t.Fatalf("ApprovalRequests = %#v, want one pending request", runtime.ApprovalRequests)
	}
	request := runtime.ApprovalRequests[0]
	if request.State != ApprovalStatePending {
		t.Fatalf("ApprovalRequests[0].State = %q, want %q", request.State, ApprovalStatePending)
	}
	if request.JobID != "job-1" || request.StepID != "discuss" {
		t.Fatalf("ApprovalRequests[0] binding = (%q, %q), want (job-1, discuss)", request.JobID, request.StepID)
	}
	if request.RequestedAction != ApprovalRequestedActionStepComplete {
		t.Fatalf("ApprovalRequests[0].RequestedAction = %q, want %q", request.RequestedAction, ApprovalRequestedActionStepComplete)
	}
	if request.Scope != ApprovalScopeMissionStep {
		t.Fatalf("ApprovalRequests[0].Scope = %q, want %q", request.Scope, ApprovalScopeMissionStep)
	}
	if request.RequestedVia != ApprovalRequestedViaRuntime {
		t.Fatalf("ApprovalRequests[0].RequestedVia = %q, want %q", request.RequestedVia, ApprovalRequestedViaRuntime)
	}
	if request.Content == nil {
		t.Fatal("ApprovalRequests[0].Content = nil, want enriched content")
	}
	if request.Content.ProposedAction == "" {
		t.Fatal("ApprovalRequests[0].Content.ProposedAction = empty, want non-empty")
	}
	if request.Content.FallbackIfDenied == "" {
		t.Fatal("ApprovalRequests[0].Content.FallbackIfDenied = empty, want non-empty")
	}
	if request.Content.AuthorityTier != AuthorityTierHigh {
		t.Fatalf("ApprovalRequests[0].Content.AuthorityTier = %q, want %q", request.Content.AuthorityTier, AuthorityTierHigh)
	}
	if request.Content.FilesystemEffect != ApprovalEffectNone || request.Content.ProcessEffect != ApprovalEffectNone || request.Content.NetworkEffect != ApprovalEffectNone {
		t.Fatalf("ApprovalRequests[0].Content effects = (%q, %q, %q), want all %q", request.Content.FilesystemEffect, request.Content.ProcessEffect, request.Content.NetworkEffect, ApprovalEffectNone)
	}
}

func TestCompleteRuntimeStepWaitUserSubtypeTransitionsToWaitingUser(t *testing.T) {
	t.Parallel()

	ec := testStepValidationExecutionContext(Step{
		ID:      "wait",
		Type:    StepTypeWaitUser,
		Subtype: StepSubtypeBlocker,
	}, JobStateRunning)
	now := time.Date(2026, 3, 23, 12, 0, 45, 0, time.UTC)

	runtime, err := CompleteRuntimeStep(ec, now, StepValidationInput{FinalResponse: "Waiting for your answer."})
	if err != nil {
		t.Fatalf("CompleteRuntimeStep() error = %v", err)
	}
	if runtime.State != JobStateWaitingUser {
		t.Fatalf("State = %q, want %q", runtime.State, JobStateWaitingUser)
	}
	if runtime.ActiveStepID != "wait" {
		t.Fatalf("ActiveStepID = %q, want %q", runtime.ActiveStepID, "wait")
	}
	if runtime.WaitingReason != "wait_user_blocker" {
		t.Fatalf("WaitingReason = %q, want %q", runtime.WaitingReason, "wait_user_blocker")
	}
	if len(runtime.ApprovalRequests) != 0 {
		t.Fatalf("ApprovalRequests = %#v, want empty for blocker wait_user step", runtime.ApprovalRequests)
	}
}

func TestCompleteRuntimeStepWaitUserAuthorizationTransitionsToWaitingUserWithPendingApproval(t *testing.T) {
	t.Parallel()

	ec := testStepValidationExecutionContext(Step{
		ID:      "wait",
		Type:    StepTypeWaitUser,
		Subtype: StepSubtypeAuthorization,
	}, JobStateRunning)
	now := time.Date(2026, 3, 23, 12, 1, 0, 0, time.UTC)

	runtime, err := CompleteRuntimeStep(ec, now, StepValidationInput{FinalResponse: "Need approval before continuing."})
	if err != nil {
		t.Fatalf("CompleteRuntimeStep() error = %v", err)
	}
	if runtime.State != JobStateWaitingUser {
		t.Fatalf("State = %q, want %q", runtime.State, JobStateWaitingUser)
	}
	if runtime.WaitingReason != "wait_user_authorization" {
		t.Fatalf("WaitingReason = %q, want %q", runtime.WaitingReason, "wait_user_authorization")
	}
	if len(runtime.ApprovalRequests) != 1 {
		t.Fatalf("ApprovalRequests = %#v, want one pending request", runtime.ApprovalRequests)
	}
	if runtime.ApprovalRequests[0].State != ApprovalStatePending {
		t.Fatalf("ApprovalRequests[0].State = %q, want %q", runtime.ApprovalRequests[0].State, ApprovalStatePending)
	}
}

func TestCompleteRuntimeStepLongRunningCodePausesAfterBuildAndValidation(t *testing.T) {
	t.Parallel()

	ec := testStepValidationExecutionContext(Step{
		ID:                        "build",
		Type:                      StepTypeLongRunningCode,
		LongRunningStartupCommand: []string{"npm", "start"},
		LongRunningArtifactPath:   "service.bin",
	}, JobStateRunning)
	now := time.Date(2026, 3, 23, 12, 1, 30, 0, time.UTC)

	runtime, err := CompleteRuntimeStep(ec, now, StepValidationInput{
		FinalResponse: "Built the service and verified the artifact.",
		SuccessfulTools: []RuntimeToolCallEvidence{
			{ToolName: "filesystem", Arguments: map[string]interface{}{"action": "write", "path": "service.bin"}},
			{ToolName: "filesystem", Arguments: map[string]interface{}{"action": "stat", "path": "service.bin"}, Result: "exists=true kind=file"},
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
	if len(runtime.CompletedSteps) != 1 || runtime.CompletedSteps[0].StepID != "build" {
		t.Fatalf("CompletedSteps = %#v, want build completion", runtime.CompletedSteps)
	}
}

func TestCompleteRuntimeStepLongRunningCodeRejectsStartEvidence(t *testing.T) {
	t.Parallel()

	ec := testStepValidationExecutionContext(Step{
		ID:                        "build",
		Type:                      StepTypeLongRunningCode,
		LongRunningStartupCommand: []string{"npm", "start"},
		LongRunningArtifactPath:   "service.bin",
	}, JobStateRunning)

	_, err := CompleteRuntimeStep(ec, time.Date(2026, 3, 23, 12, 2, 0, 0, time.UTC), StepValidationInput{
		FinalResponse: "Built and started the service.",
		SuccessfulTools: []RuntimeToolCallEvidence{
			{ToolName: "filesystem", Arguments: map[string]interface{}{"action": "write", "path": "service.bin"}},
			{ToolName: "filesystem", Arguments: map[string]interface{}{"action": "stat", "path": "service.bin"}},
			{ToolName: "exec", Arguments: map[string]interface{}{"cmd": []string{"npm", "start"}}},
		},
	})
	if err == nil {
		t.Fatal("CompleteRuntimeStep() error = nil, want long-running start rejection")
	}
	validationErr, ok := err.(ValidationError)
	if !ok {
		t.Fatalf("CompleteRuntimeStep() error = %T, want ValidationError", err)
	}
	if validationErr.Code != RejectionCodeLongRunningStartForbidden {
		t.Fatalf("ValidationError.Code = %q, want %q", validationErr.Code, RejectionCodeLongRunningStartForbidden)
	}
}

func TestValidateLongRunningCodeCompletionRejectsMissingStartupMetadata(t *testing.T) {
	t.Parallel()

	err := validateLongRunningCodeCompletion(Step{
		ID:                      "build",
		Type:                    StepTypeLongRunningCode,
		LongRunningArtifactPath: "service.bin",
	}, []RuntimeToolCallEvidence{
		{ToolName: "filesystem", Arguments: map[string]interface{}{"action": "write", "path": "service.bin"}},
		{ToolName: "filesystem", Arguments: map[string]interface{}{"action": "stat", "path": "service.bin"}, Result: "exists=true kind=file"},
	})
	if err == nil {
		t.Fatal("validateLongRunningCodeCompletion() error = nil, want missing startup metadata rejection")
	}
	validationErr, ok := err.(ValidationError)
	if !ok {
		t.Fatalf("validateLongRunningCodeCompletion() error = %T, want ValidationError", err)
	}
	if validationErr.Code != RejectionCodeStepValidationFailed {
		t.Fatalf("ValidationError.Code = %q, want %q", validationErr.Code, RejectionCodeStepValidationFailed)
	}
	if validationErr.Message != "long_running_code completion requires explicit startup command metadata" {
		t.Fatalf("ValidationError.Message = %q, want startup metadata failure", validationErr.Message)
	}
}

func TestCompleteRuntimeStepLongRunningCodeRequiresDeclaredArtifactPathEvidence(t *testing.T) {
	t.Parallel()

	ec := testStepValidationExecutionContext(Step{
		ID:                        "build",
		Type:                      StepTypeLongRunningCode,
		LongRunningStartupCommand: []string{"npm", "start"},
		LongRunningArtifactPath:   "service.bin",
	}, JobStateRunning)

	_, err := CompleteRuntimeStep(ec, time.Date(2026, 3, 23, 12, 3, 0, 0, time.UTC), StepValidationInput{
		FinalResponse: "Built another file instead.",
		SuccessfulTools: []RuntimeToolCallEvidence{
			{ToolName: "filesystem", Arguments: map[string]interface{}{"action": "write", "path": "wrong.bin"}},
			{ToolName: "filesystem", Arguments: map[string]interface{}{"action": "stat", "path": "wrong.bin"}, Result: "exists=true kind=file"},
		},
	})
	if err == nil {
		t.Fatal("CompleteRuntimeStep() error = nil, want declared artifact contract failure")
	}
	validationErr, ok := err.(ValidationError)
	if !ok {
		t.Fatalf("CompleteRuntimeStep() error = %T, want ValidationError", err)
	}
	if validationErr.Code != RejectionCodeStepValidationFailed {
		t.Fatalf("ValidationError.Code = %q, want %q", validationErr.Code, RejectionCodeStepValidationFailed)
	}
	if validationErr.Message != `long_running_code completion requires writing "service.bin"` {
		t.Fatalf("ValidationError.Message = %q, want exact artifact path failure", validationErr.Message)
	}
}

func TestCompleteRuntimeStepRejectsCompletedReplayMarkerAtPrimitiveBoundary(t *testing.T) {
	t.Parallel()

	ec := testStepValidationExecutionContext(Step{
		ID:              "build",
		Type:            StepTypeOneShotCode,
		SuccessCriteria: []string{"write code and validate it"},
	}, JobStateRunning)
	ec.Runtime.CompletedSteps = []RuntimeStepRecord{
		{StepID: "build", At: time.Date(2026, 3, 27, 12, 0, 0, 0, time.UTC)},
	}

	_, err := CompleteRuntimeStep(ec, time.Date(2026, 3, 27, 12, 5, 0, 0, time.UTC), StepValidationInput{
		FinalResponse: "Wrote and validated the code.",
		SuccessfulTools: []RuntimeToolCallEvidence{
			{ToolName: "filesystem", Arguments: map[string]interface{}{"action": "write", "path": "main.go"}},
			{ToolName: "exec", Arguments: map[string]interface{}{"cmd": []string{"go", "test", "./..."}}, Result: "ok"},
		},
	})
	if err == nil {
		t.Fatal("CompleteRuntimeStep() error = nil, want replay rejection")
	}
	validationErr, ok := err.(ValidationError)
	if !ok {
		t.Fatalf("CompleteRuntimeStep() error = %T, want ValidationError", err)
	}
	if validationErr.Code != RejectionCodeInvalidRuntimeState {
		t.Fatalf("ValidationError.Code = %q, want %q", validationErr.Code, RejectionCodeInvalidRuntimeState)
	}
	if validationErr.StepID != "build" {
		t.Fatalf("ValidationError.StepID = %q, want %q", validationErr.StepID, "build")
	}
}

func TestCompleteRuntimeStepAuthorizationBindsReusableOneJobGrantInSameJob(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 25, 12, 1, 0, 0, time.UTC)
	ec := testStepValidationExecutionContextForJob(Job{
		ID:           "job-1",
		MaxAuthority: AuthorityTierHigh,
		AllowedTools: []string{"read", "write"},
		Plan: Plan{
			ID: "plan-1",
			Steps: []Step{
				{ID: "authorize-1", Type: StepTypeDiscussion, Subtype: StepSubtypeAuthorization, ApprovalScope: ApprovalScopeOneJob},
				{ID: "authorize-2", Type: StepTypeDiscussion, Subtype: StepSubtypeAuthorization, ApprovalScope: ApprovalScopeOneJob},
			},
		},
	}, "authorize-2", JobStateRunning)
	ec.Runtime.ApprovalRequests = []ApprovalRequest{
		{
			JobID:           "job-1",
			StepID:          "authorize-1",
			RequestedAction: ApprovalRequestedActionStepComplete,
			Scope:           ApprovalScopeOneJob,
			State:           ApprovalStateGranted,
			RequestedAt:     now.Add(-2 * time.Minute),
			ResolvedAt:      now.Add(-90 * time.Second),
		},
	}
	ec.Runtime.ApprovalGrants = []ApprovalGrant{
		{
			JobID:           "job-1",
			StepID:          "authorize-1",
			RequestedAction: ApprovalRequestedActionStepComplete,
			Scope:           ApprovalScopeOneJob,
			GrantedVia:      ApprovalGrantedViaOperatorCommand,
			SessionChannel:  "telegram",
			SessionChatID:   "chat-42",
			State:           ApprovalStateGranted,
			GrantedAt:       now.Add(-90 * time.Second),
			ExpiresAt:       now.Add(time.Minute),
		},
	}

	runtime, err := CompleteRuntimeStep(ec, now, StepValidationInput{FinalResponse: "Still need approval context."})
	if err != nil {
		t.Fatalf("CompleteRuntimeStep() error = %v", err)
	}
	if runtime.State != JobStatePaused {
		t.Fatalf("State = %q, want %q", runtime.State, JobStatePaused)
	}
	if len(runtime.CompletedSteps) != 1 || runtime.CompletedSteps[0].StepID != "authorize-2" {
		t.Fatalf("CompletedSteps = %#v, want authorize-2 completion", runtime.CompletedSteps)
	}
	if len(runtime.ApprovalRequests) != 2 || runtime.ApprovalRequests[1].State != ApprovalStateGranted {
		t.Fatalf("ApprovalRequests = %#v, want reused one_job approval recorded as granted", runtime.ApprovalRequests)
	}
	if runtime.ApprovalRequests[1].StepID != "authorize-2" {
		t.Fatalf("ApprovalRequests[1].StepID = %q, want %q", runtime.ApprovalRequests[1].StepID, "authorize-2")
	}
	if runtime.ApprovalRequests[1].GrantedVia != ApprovalGrantedViaOperatorCommand {
		t.Fatalf("ApprovalRequests[1].GrantedVia = %q, want %q", runtime.ApprovalRequests[1].GrantedVia, ApprovalGrantedViaOperatorCommand)
	}
	if runtime.ApprovalRequests[1].SessionChannel != "telegram" || runtime.ApprovalRequests[1].SessionChatID != "chat-42" {
		t.Fatalf("ApprovalRequests[1] session = (%q, %q), want (%q, %q)", runtime.ApprovalRequests[1].SessionChannel, runtime.ApprovalRequests[1].SessionChatID, "telegram", "chat-42")
	}
	if len(runtime.ApprovalGrants) != 1 {
		t.Fatalf("ApprovalGrants = %#v, want original reusable grant only", runtime.ApprovalGrants)
	}
}

func TestCompleteRuntimeStepAuthorizationBindsReusableOneSessionGrantInSameSession(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 25, 12, 1, 0, 0, time.UTC)
	ec := testStepValidationExecutionContextForJob(Job{
		ID:           "job-1",
		MaxAuthority: AuthorityTierHigh,
		AllowedTools: []string{"read", "write"},
		Plan: Plan{
			ID: "plan-1",
			Steps: []Step{
				{ID: "authorize-1", Type: StepTypeDiscussion, Subtype: StepSubtypeAuthorization, ApprovalScope: ApprovalScopeOneSession},
				{ID: "authorize-2", Type: StepTypeDiscussion, Subtype: StepSubtypeAuthorization, ApprovalScope: ApprovalScopeOneSession},
			},
		},
	}, "authorize-2", JobStateRunning)
	ec.Runtime.ApprovalRequests = []ApprovalRequest{
		{
			JobID:           "job-1",
			StepID:          "authorize-1",
			RequestedAction: ApprovalRequestedActionStepComplete,
			Scope:           ApprovalScopeOneSession,
			SessionChannel:  "telegram",
			SessionChatID:   "chat-42",
			State:           ApprovalStateGranted,
			RequestedAt:     now.Add(-2 * time.Minute),
			ResolvedAt:      now.Add(-90 * time.Second),
		},
	}
	ec.Runtime.ApprovalGrants = []ApprovalGrant{
		{
			JobID:           "job-1",
			StepID:          "authorize-1",
			RequestedAction: ApprovalRequestedActionStepComplete,
			Scope:           ApprovalScopeOneSession,
			GrantedVia:      ApprovalGrantedViaOperatorCommand,
			SessionChannel:  "telegram",
			SessionChatID:   "chat-42",
			State:           ApprovalStateGranted,
			GrantedAt:       now.Add(-90 * time.Second),
			ExpiresAt:       now.Add(time.Minute),
		},
	}

	runtime, err := CompleteRuntimeStep(ec, now, StepValidationInput{
		FinalResponse:  "Still need approval context.",
		SessionChannel: "telegram",
		SessionChatID:  "chat-42",
	})
	if err != nil {
		t.Fatalf("CompleteRuntimeStep() error = %v", err)
	}
	if runtime.State != JobStatePaused {
		t.Fatalf("State = %q, want %q", runtime.State, JobStatePaused)
	}
	if len(runtime.CompletedSteps) != 1 || runtime.CompletedSteps[0].StepID != "authorize-2" {
		t.Fatalf("CompletedSteps = %#v, want authorize-2 completion", runtime.CompletedSteps)
	}
	if len(runtime.ApprovalRequests) != 2 || runtime.ApprovalRequests[1].State != ApprovalStateGranted {
		t.Fatalf("ApprovalRequests = %#v, want reused one_session approval recorded as granted", runtime.ApprovalRequests)
	}
	if runtime.ApprovalRequests[1].SessionChannel != "telegram" || runtime.ApprovalRequests[1].SessionChatID != "chat-42" {
		t.Fatalf("ApprovalRequests[1] session = (%q, %q), want (%q, %q)", runtime.ApprovalRequests[1].SessionChannel, runtime.ApprovalRequests[1].SessionChatID, "telegram", "chat-42")
	}
	if len(runtime.ApprovalGrants) != 1 {
		t.Fatalf("ApprovalGrants = %#v, want original reusable grant only", runtime.ApprovalGrants)
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

func TestCompleteRuntimeStepStaticArtifactPausesWhenExactFileAndStructureAreValidated(t *testing.T) {
	t.Parallel()

	ec := testStepValidationExecutionContext(Step{
		ID:              "artifact",
		Type:            StepTypeStaticArtifact,
		SuccessCriteria: []string{"Write `report.json` as valid JSON."},
	}, JobStateRunning)
	now := time.Date(2026, 3, 23, 12, 5, 30, 0, time.UTC)

	runtime, err := CompleteRuntimeStep(ec, now, StepValidationInput{
		FinalResponse: "Created the artifact.",
		SuccessfulTools: []RuntimeToolCallEvidence{
			{ToolName: "filesystem", Arguments: map[string]interface{}{"action": "write", "path": "report.json"}, Result: "written"},
			{ToolName: "filesystem", Arguments: map[string]interface{}{"action": "stat", "path": "report.json"}, Result: "exists=true\nkind=file\nname=report.json\nsize=17\n"},
			{ToolName: "filesystem", Arguments: map[string]interface{}{"action": "read", "path": "report.json"}, Result: "{\n  \"ok\": true\n}\n"},
		},
	})
	if err != nil {
		t.Fatalf("CompleteRuntimeStep() error = %v", err)
	}

	if runtime.State != JobStatePaused {
		t.Fatalf("State = %q, want %q", runtime.State, JobStatePaused)
	}
	if len(runtime.CompletedSteps) != 1 || runtime.CompletedSteps[0].StepID != "artifact" {
		t.Fatalf("CompletedSteps = %#v, want artifact completion", runtime.CompletedSteps)
	}
}

func TestCompleteRuntimeStepStaticArtifactRequiresStructureCheck(t *testing.T) {
	t.Parallel()

	ec := testStepValidationExecutionContext(Step{
		ID:              "artifact",
		Type:            StepTypeStaticArtifact,
		SuccessCriteria: []string{"Write `report.json` as valid JSON."},
	}, JobStateRunning)

	_, err := CompleteRuntimeStep(ec, time.Date(2026, 3, 23, 12, 5, 45, 0, time.UTC), StepValidationInput{
		FinalResponse: "Created the artifact.",
		SuccessfulTools: []RuntimeToolCallEvidence{
			{ToolName: "filesystem", Arguments: map[string]interface{}{"action": "write", "path": "report.json"}, Result: "written"},
			{ToolName: "filesystem", Arguments: map[string]interface{}{"action": "stat", "path": "report.json"}, Result: "exists=true\nkind=file\nname=report.json\nsize=17\n"},
		},
	})
	if err == nil {
		t.Fatal("CompleteRuntimeStep() error = nil, want missing structure check rejection")
	}
}

func TestCompleteRuntimeStepStaticArtifactRejectsInvalidStructure(t *testing.T) {
	t.Parallel()

	ec := testStepValidationExecutionContext(Step{
		ID:              "artifact",
		Type:            StepTypeStaticArtifact,
		SuccessCriteria: []string{"Write `report.json` as valid JSON."},
	}, JobStateRunning)

	_, err := CompleteRuntimeStep(ec, time.Date(2026, 3, 23, 12, 6, 0, 0, time.UTC), StepValidationInput{
		FinalResponse: "Created the artifact.",
		SuccessfulTools: []RuntimeToolCallEvidence{
			{ToolName: "filesystem", Arguments: map[string]interface{}{"action": "write", "path": "report.json"}, Result: "written"},
			{ToolName: "filesystem", Arguments: map[string]interface{}{"action": "stat", "path": "report.json"}, Result: "exists=true\nkind=file\nname=report.json\nsize=12\n"},
			{ToolName: "filesystem", Arguments: map[string]interface{}{"action": "read", "path": "report.json"}, Result: "{not-json}\n"},
		},
	})
	if err == nil {
		t.Fatal("CompleteRuntimeStep() error = nil, want invalid structure rejection")
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
	ec.Runtime.ApprovalRequests = []ApprovalRequest{
		{
			JobID:           "job-1",
			StepID:          "discuss",
			RequestedAction: ApprovalRequestedActionStepComplete,
			Scope:           ApprovalScopeMissionStep,
			RequestedVia:    ApprovalRequestedViaRuntime,
			State:           ApprovalStatePending,
			RequestedAt:     time.Date(2026, 3, 23, 12, 10, 30, 0, time.UTC),
		},
	}

	runtime, err := CompleteRuntimeStep(ec, now, StepValidationInput{
		UserInput:     "APPROVE job-1 discuss",
		UserInputKind: WaitingUserInputApproval,
		ApprovalVia:   ApprovalGrantedViaOperatorCommand,
	})
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
	if len(runtime.ApprovalRequests) != 1 || runtime.ApprovalRequests[0].State != ApprovalStateGranted {
		t.Fatalf("ApprovalRequests = %#v, want granted request", runtime.ApprovalRequests)
	}
	if runtime.ApprovalRequests[0].GrantedVia != ApprovalGrantedViaOperatorCommand {
		t.Fatalf("ApprovalRequests[0].GrantedVia = %q, want %q", runtime.ApprovalRequests[0].GrantedVia, ApprovalGrantedViaOperatorCommand)
	}
	if len(runtime.ApprovalGrants) != 1 || runtime.ApprovalGrants[0].State != ApprovalStateGranted {
		t.Fatalf("ApprovalGrants = %#v, want one granted record", runtime.ApprovalGrants)
	}
	if runtime.ApprovalGrants[0].ExpiresAt != ec.Runtime.ApprovalRequests[0].ExpiresAt {
		t.Fatalf("ApprovalGrants[0].ExpiresAt = %v, want %v", runtime.ApprovalGrants[0].ExpiresAt, ec.Runtime.ApprovalRequests[0].ExpiresAt)
	}
}

func TestCompleteRuntimeStepWaitingUserPendingApprovalRejectsFreeFormApproval(t *testing.T) {
	t.Parallel()

	ec := testStepValidationExecutionContext(Step{
		ID:      "discuss",
		Type:    StepTypeDiscussion,
		Subtype: StepSubtypeAuthorization,
	}, JobStateWaitingUser)
	ec.Runtime.ApprovalRequests = []ApprovalRequest{
		{
			JobID:           "job-1",
			StepID:          "discuss",
			RequestedAction: ApprovalRequestedActionStepComplete,
			Scope:           ApprovalScopeMissionStep,
			RequestedVia:    ApprovalRequestedViaRuntime,
			State:           ApprovalStatePending,
			RequestedAt:     time.Date(2026, 3, 23, 12, 10, 30, 0, time.UTC),
		},
	}

	_, err := CompleteRuntimeStep(ec, time.Date(2026, 3, 23, 12, 11, 15, 0, time.UTC), StepValidationInput{
		UserInput:     "approved",
		UserInputKind: WaitingUserInputApproval,
	})
	if err == nil {
		t.Fatal("CompleteRuntimeStep() error = nil, want explicit operator command failure")
	}
}

func TestCompleteRuntimeStepDiscussionAuthorizationStampsApprovalExpiry(t *testing.T) {
	t.Parallel()

	ec := testStepValidationExecutionContext(Step{
		ID:      "discuss",
		Type:    StepTypeDiscussion,
		Subtype: StepSubtypeAuthorization,
	}, JobStateRunning)
	now := time.Date(2026, 3, 23, 12, 10, 30, 0, time.UTC)

	runtime, err := CompleteRuntimeStep(ec, now, StepValidationInput{
		FinalResponse: "Need approval before continuing.",
	})
	if err != nil {
		t.Fatalf("CompleteRuntimeStep() error = %v", err)
	}

	if runtime.State != JobStateWaitingUser {
		t.Fatalf("State = %q, want %q", runtime.State, JobStateWaitingUser)
	}
	if len(runtime.ApprovalRequests) != 1 {
		t.Fatalf("len(ApprovalRequests) = %d, want 1", len(runtime.ApprovalRequests))
	}
	if runtime.ApprovalRequests[0].ExpiresAt != now.Add(defaultApprovalRequestTTL) {
		t.Fatalf("ApprovalRequests[0].ExpiresAt = %v, want %v", runtime.ApprovalRequests[0].ExpiresAt, now.Add(defaultApprovalRequestTTL))
	}
}

func TestCompleteRuntimeStepAuthorizationDoesNotBindOneJobGrantAcrossJobs(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 25, 12, 2, 0, 0, time.UTC)
	ec := testStepValidationExecutionContext(Step{
		ID:            "authorize-2",
		Type:          StepTypeDiscussion,
		Subtype:       StepSubtypeAuthorization,
		ApprovalScope: ApprovalScopeOneJob,
	}, JobStateRunning)
	ec.Runtime.ApprovalRequests = []ApprovalRequest{
		{
			JobID:           "other-job",
			StepID:          "authorize-1",
			RequestedAction: ApprovalRequestedActionStepComplete,
			Scope:           ApprovalScopeOneJob,
			State:           ApprovalStateGranted,
			RequestedAt:     now.Add(-2 * time.Minute),
			ResolvedAt:      now.Add(-90 * time.Second),
		},
	}
	ec.Runtime.ApprovalGrants = []ApprovalGrant{
		{
			JobID:           "other-job",
			StepID:          "authorize-1",
			RequestedAction: ApprovalRequestedActionStepComplete,
			Scope:           ApprovalScopeOneJob,
			GrantedVia:      ApprovalGrantedViaOperatorCommand,
			State:           ApprovalStateGranted,
			GrantedAt:       now.Add(-90 * time.Second),
			ExpiresAt:       now.Add(time.Minute),
		},
	}

	runtime, err := CompleteRuntimeStep(ec, now, StepValidationInput{FinalResponse: "Need approval before continuing."})
	if err != nil {
		t.Fatalf("CompleteRuntimeStep() error = %v", err)
	}
	if runtime.State != JobStateWaitingUser {
		t.Fatalf("State = %q, want %q", runtime.State, JobStateWaitingUser)
	}
	if len(runtime.ApprovalRequests) != 2 || runtime.ApprovalRequests[1].State != ApprovalStatePending {
		t.Fatalf("ApprovalRequests = %#v, want a new pending approval for the active job", runtime.ApprovalRequests)
	}
	if runtime.ApprovalRequests[1].JobID != "job-1" {
		t.Fatalf("ApprovalRequests[1].JobID = %q, want %q", runtime.ApprovalRequests[1].JobID, "job-1")
	}
}

func TestCompleteRuntimeStepAuthorizationDoesNotBindOneSessionGrantAcrossSessions(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 25, 12, 2, 0, 0, time.UTC)
	ec := testStepValidationExecutionContext(Step{
		ID:            "authorize-2",
		Type:          StepTypeDiscussion,
		Subtype:       StepSubtypeAuthorization,
		ApprovalScope: ApprovalScopeOneSession,
	}, JobStateRunning)
	ec.Runtime.ApprovalRequests = []ApprovalRequest{
		{
			JobID:           "job-1",
			StepID:          "authorize-1",
			RequestedAction: ApprovalRequestedActionStepComplete,
			Scope:           ApprovalScopeOneSession,
			SessionChannel:  "telegram",
			SessionChatID:   "chat-42",
			State:           ApprovalStateGranted,
			RequestedAt:     now.Add(-2 * time.Minute),
			ResolvedAt:      now.Add(-90 * time.Second),
		},
	}
	ec.Runtime.ApprovalGrants = []ApprovalGrant{
		{
			JobID:           "job-1",
			StepID:          "authorize-1",
			RequestedAction: ApprovalRequestedActionStepComplete,
			Scope:           ApprovalScopeOneSession,
			GrantedVia:      ApprovalGrantedViaOperatorCommand,
			SessionChannel:  "telegram",
			SessionChatID:   "chat-42",
			State:           ApprovalStateGranted,
			GrantedAt:       now.Add(-90 * time.Second),
			ExpiresAt:       now.Add(time.Minute),
		},
	}

	runtime, err := CompleteRuntimeStep(ec, now, StepValidationInput{
		FinalResponse:  "Need approval before continuing.",
		SessionChannel: "slack",
		SessionChatID:  "C123::171234",
	})
	if err != nil {
		t.Fatalf("CompleteRuntimeStep() error = %v", err)
	}
	if runtime.State != JobStateWaitingUser {
		t.Fatalf("State = %q, want %q", runtime.State, JobStateWaitingUser)
	}
	if len(runtime.ApprovalRequests) != 2 || runtime.ApprovalRequests[1].State != ApprovalStatePending {
		t.Fatalf("ApprovalRequests = %#v, want a new pending approval for the current session", runtime.ApprovalRequests)
	}
	if runtime.ApprovalRequests[1].StepID != "authorize-2" {
		t.Fatalf("ApprovalRequests[1].StepID = %q, want %q", runtime.ApprovalRequests[1].StepID, "authorize-2")
	}
}

func TestCompleteRuntimeStepAuthorizationDoesNotBindOneSessionGrantAcrossJobs(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 25, 12, 2, 0, 0, time.UTC)
	ec := testStepValidationExecutionContext(Step{
		ID:            "authorize-2",
		Type:          StepTypeDiscussion,
		Subtype:       StepSubtypeAuthorization,
		ApprovalScope: ApprovalScopeOneSession,
	}, JobStateRunning)
	ec.Runtime.ApprovalRequests = []ApprovalRequest{
		{
			JobID:           "other-job",
			StepID:          "authorize-1",
			RequestedAction: ApprovalRequestedActionStepComplete,
			Scope:           ApprovalScopeOneSession,
			SessionChannel:  "telegram",
			SessionChatID:   "chat-42",
			State:           ApprovalStateGranted,
			RequestedAt:     now.Add(-2 * time.Minute),
			ResolvedAt:      now.Add(-90 * time.Second),
		},
	}
	ec.Runtime.ApprovalGrants = []ApprovalGrant{
		{
			JobID:           "other-job",
			StepID:          "authorize-1",
			RequestedAction: ApprovalRequestedActionStepComplete,
			Scope:           ApprovalScopeOneSession,
			GrantedVia:      ApprovalGrantedViaOperatorCommand,
			SessionChannel:  "telegram",
			SessionChatID:   "chat-42",
			State:           ApprovalStateGranted,
			GrantedAt:       now.Add(-90 * time.Second),
			ExpiresAt:       now.Add(time.Minute),
		},
	}

	runtime, err := CompleteRuntimeStep(ec, now, StepValidationInput{
		FinalResponse:  "Need approval before continuing.",
		SessionChannel: "telegram",
		SessionChatID:  "chat-42",
	})
	if err != nil {
		t.Fatalf("CompleteRuntimeStep() error = %v", err)
	}
	if runtime.State != JobStateWaitingUser {
		t.Fatalf("State = %q, want %q", runtime.State, JobStateWaitingUser)
	}
	if len(runtime.ApprovalRequests) != 2 || runtime.ApprovalRequests[1].State != ApprovalStatePending {
		t.Fatalf("ApprovalRequests = %#v, want a new pending approval for the active job", runtime.ApprovalRequests)
	}
	if runtime.ApprovalRequests[1].JobID != "job-1" {
		t.Fatalf("ApprovalRequests[1].JobID = %q, want %q", runtime.ApprovalRequests[1].JobID, "job-1")
	}
}

func TestCompleteRuntimeStepAuthorizationDoesNotBindExpiredOneJobGrant(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 25, 12, 3, 0, 0, time.UTC)
	ec := testStepValidationExecutionContext(Step{
		ID:            "authorize-2",
		Type:          StepTypeDiscussion,
		Subtype:       StepSubtypeAuthorization,
		ApprovalScope: ApprovalScopeOneJob,
	}, JobStateRunning)
	ec.Runtime.ApprovalRequests = []ApprovalRequest{
		{
			JobID:           "job-1",
			StepID:          "authorize-1",
			RequestedAction: ApprovalRequestedActionStepComplete,
			Scope:           ApprovalScopeOneJob,
			State:           ApprovalStateGranted,
			RequestedAt:     now.Add(-2 * time.Minute),
			ResolvedAt:      now.Add(-90 * time.Second),
		},
	}
	ec.Runtime.ApprovalGrants = []ApprovalGrant{
		{
			JobID:           "job-1",
			StepID:          "authorize-1",
			RequestedAction: ApprovalRequestedActionStepComplete,
			Scope:           ApprovalScopeOneJob,
			GrantedVia:      ApprovalGrantedViaOperatorCommand,
			State:           ApprovalStateGranted,
			GrantedAt:       now.Add(-90 * time.Second),
			ExpiresAt:       now.Add(-time.Second),
		},
	}

	runtime, err := CompleteRuntimeStep(ec, now, StepValidationInput{FinalResponse: "Need approval before continuing."})
	if err != nil {
		t.Fatalf("CompleteRuntimeStep() error = %v", err)
	}
	if runtime.State != JobStateWaitingUser {
		t.Fatalf("State = %q, want %q", runtime.State, JobStateWaitingUser)
	}
	if len(runtime.ApprovalRequests) != 2 || runtime.ApprovalRequests[1].State != ApprovalStatePending {
		t.Fatalf("ApprovalRequests = %#v, want a fresh pending approval", runtime.ApprovalRequests)
	}
}

func TestCompleteRuntimeStepAuthorizationDoesNotBindExpiredOneSessionGrant(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 25, 12, 3, 0, 0, time.UTC)
	ec := testStepValidationExecutionContext(Step{
		ID:            "authorize-2",
		Type:          StepTypeDiscussion,
		Subtype:       StepSubtypeAuthorization,
		ApprovalScope: ApprovalScopeOneSession,
	}, JobStateRunning)
	ec.Runtime.ApprovalRequests = []ApprovalRequest{
		{
			JobID:           "job-1",
			StepID:          "authorize-1",
			RequestedAction: ApprovalRequestedActionStepComplete,
			Scope:           ApprovalScopeOneSession,
			SessionChannel:  "telegram",
			SessionChatID:   "chat-42",
			State:           ApprovalStateGranted,
			RequestedAt:     now.Add(-2 * time.Minute),
			ResolvedAt:      now.Add(-90 * time.Second),
		},
	}
	ec.Runtime.ApprovalGrants = []ApprovalGrant{
		{
			JobID:           "job-1",
			StepID:          "authorize-1",
			RequestedAction: ApprovalRequestedActionStepComplete,
			Scope:           ApprovalScopeOneSession,
			GrantedVia:      ApprovalGrantedViaOperatorCommand,
			SessionChannel:  "telegram",
			SessionChatID:   "chat-42",
			State:           ApprovalStateGranted,
			GrantedAt:       now.Add(-90 * time.Second),
			ExpiresAt:       now.Add(-time.Second),
		},
	}

	runtime, err := CompleteRuntimeStep(ec, now, StepValidationInput{
		FinalResponse:  "Need approval before continuing.",
		SessionChannel: "telegram",
		SessionChatID:  "chat-42",
	})
	if err != nil {
		t.Fatalf("CompleteRuntimeStep() error = %v", err)
	}
	if runtime.State != JobStateWaitingUser {
		t.Fatalf("State = %q, want %q", runtime.State, JobStateWaitingUser)
	}
	if len(runtime.ApprovalRequests) != 2 || runtime.ApprovalRequests[1].State != ApprovalStatePending {
		t.Fatalf("ApprovalRequests = %#v, want a fresh pending approval", runtime.ApprovalRequests)
	}
}

func TestCompleteRuntimeStepAuthorizationDoesNotBindNonGrantedOneJobRequest(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 25, 12, 4, 0, 0, time.UTC)
	for _, tc := range []struct {
		name  string
		state ApprovalState
	}{
		{name: "pending", state: ApprovalStatePending},
		{name: "denied", state: ApprovalStateDenied},
		{name: "superseded", state: ApprovalStateSuperseded},
		{name: "revoked", state: ApprovalStateRevoked},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ec := testStepValidationExecutionContext(Step{
				ID:            "authorize-2",
				Type:          StepTypeDiscussion,
				Subtype:       StepSubtypeAuthorization,
				ApprovalScope: ApprovalScopeOneJob,
			}, JobStateRunning)
			ec.Runtime.ApprovalRequests = []ApprovalRequest{
				{
					JobID:           "job-1",
					StepID:          "authorize-1",
					RequestedAction: ApprovalRequestedActionStepComplete,
					Scope:           ApprovalScopeOneJob,
					State:           tc.state,
					RequestedAt:     now.Add(-2 * time.Minute),
				},
			}
			if tc.state != ApprovalStatePending {
				ec.Runtime.ApprovalRequests[0].ResolvedAt = now.Add(-90 * time.Second)
			}
			ec.Runtime.ApprovalGrants = []ApprovalGrant{
				{
					JobID:           "job-1",
					StepID:          "authorize-1",
					RequestedAction: ApprovalRequestedActionStepComplete,
					Scope:           ApprovalScopeOneJob,
					GrantedVia:      ApprovalGrantedViaOperatorCommand,
					State:           ApprovalStateGranted,
					GrantedAt:       now.Add(-90 * time.Second),
					ExpiresAt:       now.Add(time.Minute),
				},
			}

			runtime, err := CompleteRuntimeStep(ec, now, StepValidationInput{FinalResponse: "Need approval before continuing."})
			if err != nil {
				t.Fatalf("CompleteRuntimeStep() error = %v", err)
			}
			if runtime.State != JobStateWaitingUser {
				t.Fatalf("State = %q, want %q", runtime.State, JobStateWaitingUser)
			}
			if len(runtime.ApprovalRequests) != 2 || runtime.ApprovalRequests[1].State != ApprovalStatePending {
				t.Fatalf("ApprovalRequests = %#v, want a fresh pending approval", runtime.ApprovalRequests)
			}
		})
	}
}

func TestCompleteRuntimeStepAuthorizationDoesNotBindNonGrantedOneSessionRequest(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 25, 12, 4, 0, 0, time.UTC)
	for _, tc := range []struct {
		name  string
		state ApprovalState
	}{
		{name: "pending", state: ApprovalStatePending},
		{name: "denied", state: ApprovalStateDenied},
		{name: "expired", state: ApprovalStateExpired},
		{name: "superseded", state: ApprovalStateSuperseded},
		{name: "revoked", state: ApprovalStateRevoked},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ec := testStepValidationExecutionContext(Step{
				ID:            "authorize-2",
				Type:          StepTypeDiscussion,
				Subtype:       StepSubtypeAuthorization,
				ApprovalScope: ApprovalScopeOneSession,
			}, JobStateRunning)
			ec.Runtime.ApprovalRequests = []ApprovalRequest{
				{
					JobID:           "job-1",
					StepID:          "authorize-1",
					RequestedAction: ApprovalRequestedActionStepComplete,
					Scope:           ApprovalScopeOneSession,
					SessionChannel:  "telegram",
					SessionChatID:   "chat-42",
					State:           tc.state,
					RequestedAt:     now.Add(-2 * time.Minute),
				},
			}
			if tc.state != ApprovalStatePending {
				ec.Runtime.ApprovalRequests[0].ResolvedAt = now.Add(-90 * time.Second)
			}
			ec.Runtime.ApprovalGrants = []ApprovalGrant{
				{
					JobID:           "job-1",
					StepID:          "authorize-1",
					RequestedAction: ApprovalRequestedActionStepComplete,
					Scope:           ApprovalScopeOneSession,
					GrantedVia:      ApprovalGrantedViaOperatorCommand,
					SessionChannel:  "telegram",
					SessionChatID:   "chat-42",
					State:           ApprovalStateGranted,
					GrantedAt:       now.Add(-90 * time.Second),
					ExpiresAt:       now.Add(time.Minute),
				},
			}

			runtime, err := CompleteRuntimeStep(ec, now, StepValidationInput{
				FinalResponse:  "Need approval before continuing.",
				SessionChannel: "telegram",
				SessionChatID:  "chat-42",
			})
			if err != nil {
				t.Fatalf("CompleteRuntimeStep() error = %v", err)
			}
			if runtime.State != JobStateWaitingUser {
				t.Fatalf("State = %q, want %q", runtime.State, JobStateWaitingUser)
			}
			if len(runtime.ApprovalRequests) != 2 || runtime.ApprovalRequests[1].State != ApprovalStatePending {
				t.Fatalf("ApprovalRequests = %#v, want a fresh pending approval", runtime.ApprovalRequests)
			}
		})
	}
}

func TestCompleteRuntimeStepWaitingUserPendingApprovalDoesNotLeakAcrossSteps(t *testing.T) {
	t.Parallel()

	ec := testStepValidationExecutionContext(Step{
		ID:      "discuss",
		Type:    StepTypeDiscussion,
		Subtype: StepSubtypeAuthorization,
	}, JobStateWaitingUser)
	ec.Runtime.ApprovalRequests = []ApprovalRequest{
		{
			JobID:           "job-1",
			StepID:          "other-step",
			RequestedAction: ApprovalRequestedActionStepComplete,
			Scope:           ApprovalScopeMissionStep,
			RequestedVia:    ApprovalRequestedViaRuntime,
			State:           ApprovalStatePending,
			RequestedAt:     time.Date(2026, 3, 23, 12, 10, 30, 0, time.UTC),
		},
	}

	runtime, err := CompleteRuntimeStep(ec, time.Date(2026, 3, 23, 12, 11, 30, 0, time.UTC), StepValidationInput{
		UserInput: "approved",
	})
	if err != nil {
		t.Fatalf("CompleteRuntimeStep() error = %v", err)
	}
	if runtime.State != JobStatePaused {
		t.Fatalf("State = %q, want %q", runtime.State, JobStatePaused)
	}
}

func TestCompleteRuntimeStepWaitingUserDenyRecordsDeniedApproval(t *testing.T) {
	t.Parallel()

	ec := testStepValidationExecutionContext(Step{
		ID:      "discuss",
		Type:    StepTypeDiscussion,
		Subtype: StepSubtypeAuthorization,
	}, JobStateWaitingUser)
	now := time.Date(2026, 3, 23, 12, 11, 45, 0, time.UTC)
	ec.Runtime.ApprovalRequests = []ApprovalRequest{
		{
			JobID:           "job-1",
			StepID:          "discuss",
			RequestedAction: ApprovalRequestedActionStepComplete,
			Scope:           ApprovalScopeMissionStep,
			RequestedVia:    ApprovalRequestedViaRuntime,
			State:           ApprovalStatePending,
			RequestedAt:     time.Date(2026, 3, 23, 12, 10, 30, 0, time.UTC),
		},
	}

	runtime, err := CompleteRuntimeStep(ec, now, StepValidationInput{
		UserInput:     "DENY job-1 discuss",
		UserInputKind: WaitingUserInputRejection,
		ApprovalVia:   ApprovalGrantedViaOperatorCommand,
	})
	if err != nil {
		t.Fatalf("CompleteRuntimeStep() error = %v", err)
	}
	if runtime.State != JobStateWaitingUser {
		t.Fatalf("State = %q, want %q", runtime.State, JobStateWaitingUser)
	}
	if len(runtime.CompletedSteps) != 0 {
		t.Fatalf("CompletedSteps = %#v, want empty", runtime.CompletedSteps)
	}
	if len(runtime.ApprovalRequests) != 1 || runtime.ApprovalRequests[0].State != ApprovalStateDenied {
		t.Fatalf("ApprovalRequests = %#v, want denied request", runtime.ApprovalRequests)
	}
	if len(runtime.ApprovalGrants) != 0 {
		t.Fatalf("ApprovalGrants = %#v, want empty", runtime.ApprovalGrants)
	}
}

func TestCompleteRuntimeStepWaitingUserTimeoutExpiresPendingApproval(t *testing.T) {
	t.Parallel()

	ec := testStepValidationExecutionContext(Step{
		ID:      "discuss",
		Type:    StepTypeDiscussion,
		Subtype: StepSubtypeAuthorization,
	}, JobStateWaitingUser)
	now := time.Date(2026, 3, 23, 12, 11, 50, 0, time.UTC)
	ec.Runtime.ApprovalRequests = []ApprovalRequest{
		{
			JobID:           "job-1",
			StepID:          "discuss",
			RequestedAction: ApprovalRequestedActionStepComplete,
			Scope:           ApprovalScopeMissionStep,
			RequestedVia:    ApprovalRequestedViaRuntime,
			State:           ApprovalStatePending,
			RequestedAt:     time.Date(2026, 3, 23, 12, 10, 30, 0, time.UTC),
		},
	}

	runtime, err := CompleteRuntimeStep(ec, now, StepValidationInput{
		UserInput:     "timeout",
		UserInputKind: WaitingUserInputTimeout,
	})
	if err != nil {
		t.Fatalf("CompleteRuntimeStep() error = %v", err)
	}
	if runtime.State != JobStateWaitingUser {
		t.Fatalf("State = %q, want %q", runtime.State, JobStateWaitingUser)
	}
	if len(runtime.ApprovalRequests) != 1 || runtime.ApprovalRequests[0].State != ApprovalStateExpired {
		t.Fatalf("ApprovalRequests = %#v, want one expired approval", runtime.ApprovalRequests)
	}
	if runtime.ApprovalRequests[0].ExpiresAt != now {
		t.Fatalf("ApprovalRequests[0].ExpiresAt = %v, want %v", runtime.ApprovalRequests[0].ExpiresAt, now)
	}
}

func TestCompleteRuntimeStepWaitingUserDeniedApprovalRejectsLaterFreeFormInput(t *testing.T) {
	t.Parallel()

	ec := testStepValidationExecutionContext(Step{
		ID:      "discuss",
		Type:    StepTypeDiscussion,
		Subtype: StepSubtypeAuthorization,
	}, JobStateWaitingUser)
	ec.Runtime.ApprovalRequests = []ApprovalRequest{
		{
			JobID:           "job-1",
			StepID:          "discuss",
			RequestedAction: ApprovalRequestedActionStepComplete,
			Scope:           ApprovalScopeMissionStep,
			RequestedVia:    ApprovalRequestedViaRuntime,
			State:           ApprovalStateDenied,
			RequestedAt:     time.Date(2026, 3, 23, 12, 10, 30, 0, time.UTC),
			ResolvedAt:      time.Date(2026, 3, 23, 12, 11, 45, 0, time.UTC),
		},
	}

	_, err := CompleteRuntimeStep(ec, time.Date(2026, 3, 23, 12, 12, 0, 0, time.UTC), StepValidationInput{
		UserInput: "go ahead",
	})
	if err == nil {
		t.Fatal("CompleteRuntimeStep() error = nil, want denied approval failure")
	}

	validationErr, ok := err.(ValidationError)
	if !ok {
		t.Fatalf("CompleteRuntimeStep() error type = %T, want ValidationError", err)
	}
	if validationErr.Code != RejectionCodeStepValidationFailed {
		t.Fatalf("ValidationError.Code = %q, want %q", validationErr.Code, RejectionCodeStepValidationFailed)
	}
}

func TestCompleteRuntimeStepWaitingUserExpiredApprovalRejectsLaterFreeFormInput(t *testing.T) {
	t.Parallel()

	ec := testStepValidationExecutionContext(Step{
		ID:      "discuss",
		Type:    StepTypeDiscussion,
		Subtype: StepSubtypeAuthorization,
	}, JobStateWaitingUser)
	ec.Runtime.ApprovalRequests = []ApprovalRequest{
		{
			JobID:           "job-1",
			StepID:          "discuss",
			RequestedAction: ApprovalRequestedActionStepComplete,
			Scope:           ApprovalScopeMissionStep,
			RequestedVia:    ApprovalRequestedViaRuntime,
			State:           ApprovalStateExpired,
			RequestedAt:     time.Date(2026, 3, 23, 12, 10, 30, 0, time.UTC),
			ExpiresAt:       time.Date(2026, 3, 23, 12, 11, 45, 0, time.UTC),
			ResolvedAt:      time.Date(2026, 3, 23, 12, 11, 45, 0, time.UTC),
		},
	}

	_, err := CompleteRuntimeStep(ec, time.Date(2026, 3, 23, 12, 12, 0, 0, time.UTC), StepValidationInput{
		UserInput: "go ahead",
	})
	if err == nil {
		t.Fatal("CompleteRuntimeStep() error = nil, want expired approval failure")
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
	steps := []Step{step}
	if step.Type != StepTypeFinalResponse || step.ID != "final" {
		steps = append(steps, Step{
			ID:        "final",
			Type:      StepTypeFinalResponse,
			DependsOn: []string{step.ID},
		})
	}

	return testStepValidationExecutionContextForJob(Job{
		ID:           "job-1",
		MaxAuthority: AuthorityTierHigh,
		AllowedTools: []string{"read", "write"},
		SpecVersion:  specVersionForTestSteps(steps),
		Plan: Plan{
			ID:    "plan-1",
			Steps: steps,
		},
	}, step.ID, state)
}

func testStepValidationExecutionContextForJob(job Job, stepID string, state JobState) ExecutionContext {
	if len(job.Plan.Steps) > 0 {
		hasTerminalFinal := false
		for _, planStep := range job.Plan.Steps {
			if planStep.Type == StepTypeFinalResponse {
				hasTerminalFinal = true
				break
			}
		}
		if !hasTerminalFinal {
			job.Plan.Steps = append(job.Plan.Steps, Step{
				ID:        "final",
				Type:      StepTypeFinalResponse,
				DependsOn: []string{job.Plan.Steps[len(job.Plan.Steps)-1].ID},
			})
		}
	}

	ec, err := ResolveExecutionContext(job, stepID)
	if err != nil {
		panic(err)
	}

	return ExecutionContext{
		Job:  ec.Job,
		Step: ec.Step,
		Runtime: &JobRuntimeState{
			JobID:        job.ID,
			State:        state,
			ActiveStepID: stepID,
		},
	}
}

func specVersionForTestSteps(steps []Step) string {
	for _, step := range steps {
		if isV2OnlyStepType(step.Type) {
			return JobSpecVersionV2
		}
	}
	return ""
}
