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
