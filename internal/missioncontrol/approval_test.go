package missioncontrol

import (
	"testing"
	"time"
)

func TestParsePlainApprovalDecision(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		input    string
		want     ApprovalDecision
		wantOK   bool
		testName string
	}{
		{testName: "yes", input: "yes", want: ApprovalDecisionApprove, wantOK: true},
		{testName: "yes punctuation", input: " Yes! ", want: ApprovalDecisionApprove, wantOK: true},
		{testName: "no", input: "no", want: ApprovalDecisionDeny, wantOK: true},
		{testName: "go ahead unchanged", input: "go ahead", wantOK: false},
	} {
		t.Run(tc.testName, func(t *testing.T) {
			got, ok := ParsePlainApprovalDecision(tc.input)
			if ok != tc.wantOK {
				t.Fatalf("ParsePlainApprovalDecision(%q) ok = %t, want %t", tc.input, ok, tc.wantOK)
			}
			if got != tc.want {
				t.Fatalf("ParsePlainApprovalDecision(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestResolveSinglePendingApprovalRequest(t *testing.T) {
	t.Parallel()

	pending := ApprovalRequest{
		JobID:           "job-1",
		StepID:          "build",
		RequestedAction: ApprovalRequestedActionStepComplete,
		Scope:           ApprovalScopeMissionStep,
		State:           ApprovalStatePending,
	}

	request, ok, err := ResolveSinglePendingApprovalRequest(JobRuntimeState{
		ApprovalRequests: []ApprovalRequest{pending},
	})
	if err != nil {
		t.Fatalf("ResolveSinglePendingApprovalRequest(single) error = %v", err)
	}
	if !ok {
		t.Fatal("ResolveSinglePendingApprovalRequest(single) ok = false, want true")
	}
	if request != pending {
		t.Fatalf("ResolveSinglePendingApprovalRequest(single) = %#v, want %#v", request, pending)
	}

	_, ok, err = ResolveSinglePendingApprovalRequest(JobRuntimeState{})
	if err != nil {
		t.Fatalf("ResolveSinglePendingApprovalRequest(none) error = %v", err)
	}
	if ok {
		t.Fatal("ResolveSinglePendingApprovalRequest(none) ok = true, want false")
	}

	_, ok, err = ResolveSinglePendingApprovalRequest(JobRuntimeState{
		ApprovalRequests: []ApprovalRequest{
			pending,
			{
				JobID:           "job-1",
				StepID:          "other",
				RequestedAction: ApprovalRequestedActionStepComplete,
				Scope:           ApprovalScopeMissionStep,
				State:           ApprovalStatePending,
			},
		},
	})
	if err == nil {
		t.Fatal("ResolveSinglePendingApprovalRequest(multiple) error = nil, want ambiguity failure")
	}
	if !ok {
		t.Fatal("ResolveSinglePendingApprovalRequest(multiple) ok = false, want true")
	}
}

func TestApprovalRequestMatchesStepBinding(t *testing.T) {
	t.Parallel()

	step := Step{
		ID:      "build",
		Type:    StepTypeDiscussion,
		Subtype: StepSubtypeAuthorization,
	}
	request := ApprovalRequest{
		JobID:           "job-1",
		StepID:          "build",
		RequestedAction: ApprovalRequestedActionStepComplete,
		Scope:           ApprovalScopeMissionStep,
		State:           ApprovalStatePending,
	}

	if !ApprovalRequestMatchesStepBinding(request, "job-1", step) {
		t.Fatal("ApprovalRequestMatchesStepBinding() = false, want true")
	}
	if ApprovalRequestMatchesStepBinding(request, "other-job", step) {
		t.Fatal("ApprovalRequestMatchesStepBinding(wrong job) = true, want false")
	}
}

func TestFindReusableApprovalGrantMatchesOneJobAcrossSteps(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC)
	grant, ok := FindReusableApprovalGrant(JobRuntimeState{
		ApprovalRequests: []ApprovalRequest{
			{
				JobID:           "job-1",
				StepID:          "authorize-1",
				RequestedAction: ApprovalRequestedActionStepComplete,
				Scope:           ApprovalScopeOneJob,
				State:           ApprovalStateGranted,
				RequestedAt:     now.Add(-2 * time.Minute),
				ResolvedAt:      now.Add(-90 * time.Second),
			},
		},
		ApprovalGrants: []ApprovalGrant{
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
		},
	}, now, "job-1", Step{
		ID:            "authorize-2",
		Type:          StepTypeDiscussion,
		Subtype:       StepSubtypeAuthorization,
		ApprovalScope: ApprovalScopeOneJob,
	})
	if !ok {
		t.Fatal("FindReusableApprovalGrant() ok = false, want true")
	}
	if grant.StepID != "authorize-1" {
		t.Fatalf("FindReusableApprovalGrant().StepID = %q, want %q", grant.StepID, "authorize-1")
	}
}

func TestFindReusableApprovalGrantRejectsNonGrantedLatestJobScopeState(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		name    string
		request ApprovalRequest
		grant   ApprovalGrant
	}{
		{
			name: "pending request",
			request: ApprovalRequest{
				JobID:           "job-1",
				StepID:          "authorize-2",
				RequestedAction: ApprovalRequestedActionStepComplete,
				Scope:           ApprovalScopeOneJob,
				State:           ApprovalStatePending,
				RequestedAt:     now.Add(-30 * time.Second),
			},
			grant: ApprovalGrant{
				JobID:           "job-1",
				StepID:          "authorize-1",
				RequestedAction: ApprovalRequestedActionStepComplete,
				Scope:           ApprovalScopeOneJob,
				GrantedVia:      ApprovalGrantedViaOperatorCommand,
				State:           ApprovalStateGranted,
				GrantedAt:       now.Add(-2 * time.Minute),
				ExpiresAt:       now.Add(time.Minute),
			},
		},
		{
			name: "revoked grant",
			request: ApprovalRequest{
				JobID:           "job-1",
				StepID:          "authorize-1",
				RequestedAction: ApprovalRequestedActionStepComplete,
				Scope:           ApprovalScopeOneJob,
				State:           ApprovalStateGranted,
				RequestedAt:     now.Add(-2 * time.Minute),
				ResolvedAt:      now.Add(-90 * time.Second),
			},
			grant: ApprovalGrant{
				JobID:           "job-1",
				StepID:          "authorize-1",
				RequestedAction: ApprovalRequestedActionStepComplete,
				Scope:           ApprovalScopeOneJob,
				GrantedVia:      ApprovalGrantedViaOperatorCommand,
				State:           ApprovalStateRevoked,
				GrantedAt:       now.Add(-90 * time.Second),
				RevokedAt:       now.Add(-30 * time.Second),
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, ok := FindReusableApprovalGrant(JobRuntimeState{
				ApprovalRequests: []ApprovalRequest{tc.request},
				ApprovalGrants:   []ApprovalGrant{tc.grant},
			}, now, "job-1", Step{
				ID:            "authorize-2",
				Type:          StepTypeDiscussion,
				Subtype:       StepSubtypeAuthorization,
				ApprovalScope: ApprovalScopeOneJob,
			})
			if ok {
				t.Fatal("FindReusableApprovalGrant() ok = true, want false")
			}
		})
	}
}

func TestFindReusableApprovalGrantRejectsWrongAction(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC)
	_, ok := FindReusableApprovalGrant(JobRuntimeState{
		ApprovalRequests: []ApprovalRequest{
			{
				JobID:           "job-1",
				StepID:          "authorize-1",
				RequestedAction: "different_action",
				Scope:           ApprovalScopeOneJob,
				State:           ApprovalStateGranted,
				RequestedAt:     now.Add(-2 * time.Minute),
				ResolvedAt:      now.Add(-90 * time.Second),
			},
		},
		ApprovalGrants: []ApprovalGrant{
			{
				JobID:           "job-1",
				StepID:          "authorize-1",
				RequestedAction: "different_action",
				Scope:           ApprovalScopeOneJob,
				GrantedVia:      ApprovalGrantedViaOperatorCommand,
				State:           ApprovalStateGranted,
				GrantedAt:       now.Add(-90 * time.Second),
				ExpiresAt:       now.Add(time.Minute),
			},
		},
	}, now, "job-1", Step{
		ID:            "authorize-2",
		Type:          StepTypeDiscussion,
		Subtype:       StepSubtypeAuthorization,
		ApprovalScope: ApprovalScopeOneJob,
	})
	if ok {
		t.Fatal("FindReusableApprovalGrant() ok = true, want false")
	}
}

func TestApplyApprovalDecisionWithSessionStampsRequestAndGrant(t *testing.T) {
	t.Parallel()

	ec := testStepValidationExecutionContext(Step{
		ID:      "discuss",
		Type:    StepTypeDiscussion,
		Subtype: StepSubtypeAuthorization,
	}, JobStateWaitingUser)
	now := time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC)
	ec.Runtime.ApprovalRequests = []ApprovalRequest{
		{
			JobID:           "job-1",
			StepID:          "discuss",
			RequestedAction: ApprovalRequestedActionStepComplete,
			Scope:           ApprovalScopeMissionStep,
			RequestedVia:    ApprovalRequestedViaRuntime,
			State:           ApprovalStatePending,
			RequestedAt:     now.Add(-30 * time.Second),
			ExpiresAt:       now.Add(5 * time.Minute),
		},
	}

	runtime, err := ApplyApprovalDecisionWithSession(ec, now, ApprovalDecisionApprove, ApprovalGrantedViaOperatorCommand, "telegram", "chat-42")
	if err != nil {
		t.Fatalf("ApplyApprovalDecisionWithSession() error = %v", err)
	}
	if len(runtime.ApprovalRequests) != 1 {
		t.Fatalf("ApprovalRequests = %#v, want one request", runtime.ApprovalRequests)
	}
	if runtime.ApprovalRequests[0].SessionChannel != "telegram" || runtime.ApprovalRequests[0].SessionChatID != "chat-42" {
		t.Fatalf("ApprovalRequests[0] session = (%q, %q), want (%q, %q)", runtime.ApprovalRequests[0].SessionChannel, runtime.ApprovalRequests[0].SessionChatID, "telegram", "chat-42")
	}
	if len(runtime.ApprovalGrants) != 1 {
		t.Fatalf("ApprovalGrants = %#v, want one grant", runtime.ApprovalGrants)
	}
	if runtime.ApprovalGrants[0].SessionChannel != "telegram" || runtime.ApprovalGrants[0].SessionChatID != "chat-42" {
		t.Fatalf("ApprovalGrants[0] session = (%q, %q), want (%q, %q)", runtime.ApprovalGrants[0].SessionChannel, runtime.ApprovalGrants[0].SessionChatID, "telegram", "chat-42")
	}
}

func TestApprovalRequestContentForAuthorizationStep(t *testing.T) {
	t.Parallel()

	content, ok := approvalRequestContentForStep(Job{MaxAuthority: AuthorityTierHigh}, Step{
		ID:                "build",
		Type:              StepTypeDiscussion,
		Subtype:           StepSubtypeAuthorization,
		RequiredAuthority: AuthorityTierMedium,
	})
	if !ok {
		t.Fatal("approvalRequestContentForStep() ok = false, want true")
	}
	if content.ProposedAction == "" {
		t.Fatal("approvalRequestContentForStep().ProposedAction = empty, want non-empty")
	}
	if content.FallbackIfDenied == "" {
		t.Fatal("approvalRequestContentForStep().FallbackIfDenied = empty, want non-empty")
	}
	if content.AuthorityTier != AuthorityTierMedium {
		t.Fatalf("approvalRequestContentForStep().AuthorityTier = %q, want %q", content.AuthorityTier, AuthorityTierMedium)
	}
	if content.IdentityScope != ApprovalScopeNone || content.PublicScope != ApprovalScopeNone {
		t.Fatalf("approvalRequestContentForStep() scopes = (%q, %q), want (%q, %q)", content.IdentityScope, content.PublicScope, ApprovalScopeNone, ApprovalScopeNone)
	}
	if content.FilesystemEffect != ApprovalEffectNone || content.ProcessEffect != ApprovalEffectNone || content.NetworkEffect != ApprovalEffectNone {
		t.Fatalf("approvalRequestContentForStep() effects = (%q, %q, %q), want all %q", content.FilesystemEffect, content.ProcessEffect, content.NetworkEffect, ApprovalEffectNone)
	}
}

func TestApprovalRequestContentForAuthorizationStepFallsBackToJobAuthority(t *testing.T) {
	t.Parallel()

	content, ok := approvalRequestContentForStep(Job{MaxAuthority: AuthorityTierHigh}, Step{
		ID:      "build",
		Type:    StepTypeDiscussion,
		Subtype: StepSubtypeAuthorization,
	})
	if !ok {
		t.Fatal("approvalRequestContentForStep() ok = false, want true")
	}
	if content.AuthorityTier != AuthorityTierHigh {
		t.Fatalf("approvalRequestContentForStep().AuthorityTier = %q, want %q", content.AuthorityTier, AuthorityTierHigh)
	}
}

func TestRefreshApprovalRequestsExpiresElapsedPendingRequest(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 24, 15, 0, 0, 0, time.UTC)
	runtime, changed := RefreshApprovalRequests(JobRuntimeState{
		ApprovalRequests: []ApprovalRequest{
			{
				JobID:           "job-1",
				StepID:          "build",
				RequestedAction: ApprovalRequestedActionStepComplete,
				Scope:           ApprovalScopeMissionStep,
				State:           ApprovalStatePending,
				RequestedAt:     now.Add(-2 * time.Minute),
				ExpiresAt:       now.Add(-1 * time.Minute),
			},
		},
	}, now)
	if !changed {
		t.Fatal("RefreshApprovalRequests() changed = false, want true")
	}
	if runtime.ApprovalRequests[0].State != ApprovalStateExpired {
		t.Fatalf("ApprovalRequests[0].State = %q, want %q", runtime.ApprovalRequests[0].State, ApprovalStateExpired)
	}
	if runtime.ApprovalRequests[0].ResolvedAt != now.Add(-1*time.Minute) {
		t.Fatalf("ApprovalRequests[0].ResolvedAt = %v, want %v", runtime.ApprovalRequests[0].ResolvedAt, now.Add(-1*time.Minute))
	}
}

func TestAppendPendingApprovalRequestSupersedesOlderMatchingPendingRequest(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 24, 15, 5, 0, 0, time.UTC)
	runtime := appendPendingApprovalRequest(JobRuntimeState{
		ApprovalRequests: []ApprovalRequest{
			{
				JobID:           "job-1",
				StepID:          "build",
				RequestedAction: ApprovalRequestedActionStepComplete,
				Scope:           ApprovalScopeMissionStep,
				State:           ApprovalStatePending,
				RequestedAt:     now.Add(-2 * time.Minute),
			},
		},
	}, now, ApprovalRequest{
		JobID:           "job-1",
		StepID:          "build",
		RequestedAction: ApprovalRequestedActionStepComplete,
		Scope:           ApprovalScopeMissionStep,
		RequestedVia:    ApprovalRequestedViaRuntime,
	})
	if len(runtime.ApprovalRequests) != 2 {
		t.Fatalf("len(ApprovalRequests) = %d, want 2", len(runtime.ApprovalRequests))
	}
	if runtime.ApprovalRequests[0].State != ApprovalStateSuperseded {
		t.Fatalf("ApprovalRequests[0].State = %q, want %q", runtime.ApprovalRequests[0].State, ApprovalStateSuperseded)
	}
	if runtime.ApprovalRequests[0].SupersededAt != now {
		t.Fatalf("ApprovalRequests[0].SupersededAt = %v, want %v", runtime.ApprovalRequests[0].SupersededAt, now)
	}
	if runtime.ApprovalRequests[1].State != ApprovalStatePending {
		t.Fatalf("ApprovalRequests[1].State = %q, want %q", runtime.ApprovalRequests[1].State, ApprovalStatePending)
	}
	if runtime.ApprovalRequests[1].ExpiresAt != now.Add(defaultApprovalRequestTTL) {
		t.Fatalf("ApprovalRequests[1].ExpiresAt = %v, want %v", runtime.ApprovalRequests[1].ExpiresAt, now.Add(defaultApprovalRequestTTL))
	}
}

func TestAppendPendingApprovalRequestSupersedesOlderPendingOneJobRequestAcrossSteps(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 24, 15, 5, 0, 0, time.UTC)
	runtime := appendPendingApprovalRequest(JobRuntimeState{
		ApprovalRequests: []ApprovalRequest{
			{
				JobID:           "job-1",
				StepID:          "authorize-1",
				RequestedAction: ApprovalRequestedActionStepComplete,
				Scope:           ApprovalScopeOneJob,
				State:           ApprovalStatePending,
				RequestedAt:     now.Add(-2 * time.Minute),
			},
		},
	}, now, ApprovalRequest{
		JobID:           "job-1",
		StepID:          "authorize-2",
		RequestedAction: ApprovalRequestedActionStepComplete,
		Scope:           ApprovalScopeOneJob,
		RequestedVia:    ApprovalRequestedViaRuntime,
	})

	if len(runtime.ApprovalRequests) != 2 {
		t.Fatalf("len(ApprovalRequests) = %d, want 2", len(runtime.ApprovalRequests))
	}
	if runtime.ApprovalRequests[0].State != ApprovalStateSuperseded {
		t.Fatalf("ApprovalRequests[0].State = %q, want %q", runtime.ApprovalRequests[0].State, ApprovalStateSuperseded)
	}
	if runtime.ApprovalRequests[1].State != ApprovalStatePending {
		t.Fatalf("ApprovalRequests[1].State = %q, want %q", runtime.ApprovalRequests[1].State, ApprovalStatePending)
	}
	if runtime.ApprovalRequests[1].StepID != "authorize-2" {
		t.Fatalf("ApprovalRequests[1].StepID = %q, want %q", runtime.ApprovalRequests[1].StepID, "authorize-2")
	}
}
