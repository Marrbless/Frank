package missioncontrol

import "testing"

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
