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
