package missioncontrol

import "testing"

func TestAppendAuditHistoryCanonicalizesFrozenV2ErrorCodes(t *testing.T) {
	t.Parallel()

	history := AppendAuditHistory(nil, AuditEvent{
		JobID:    "job-1",
		StepID:   "step-1",
		ToolName: "read",
		Allowed:  false,
		Code:     RejectionCodeToolNotAllowed,
		Reason:   "tool is not allowed by step tool scope",
	})
	if got := history[0].Code; got != RejectionCode("E_INVALID_ACTION_FOR_STEP") {
		t.Fatalf("AppendAuditHistory()[0].Code = %q, want %q", got, RejectionCode("E_INVALID_ACTION_FOR_STEP"))
	}

	history = AppendAuditHistory(nil, AuditEvent{
		JobID:    "job-1",
		StepID:   "step-1",
		ToolName: "resume",
		Allowed:  false,
		Code:     RejectionCodeResumeApprovalRequired,
		Reason:   "runtime resume requires explicit approval",
	})
	if got := history[0].Code; got != RejectionCode("E_RESUME_REQUIRES_APPROVAL") {
		t.Fatalf("AppendAuditHistory()[0].Code = %q, want %q", got, RejectionCode("E_RESUME_REQUIRES_APPROVAL"))
	}
}

func TestAppendAuditHistoryCanonicalizesInvalidRuntimeStateByReason(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		reason string
		want   RejectionCode
	}{
		{
			name:   "no active step",
			reason: "runtime execution requires an active step",
			want:   RejectionCode("E_NO_ACTIVE_STEP"),
		},
		{
			name:   "aborted",
			reason: "job is aborted and cannot resume",
			want:   RejectionCode("E_ABORTED"),
		},
		{
			name:   "out of order fallback",
			reason: "runtime active step \"build\" does not match control step \"final\"",
			want:   RejectionCode("E_STEP_OUT_OF_ORDER"),
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			history := AppendAuditHistory(nil, AuditEvent{
				JobID:    "job-1",
				StepID:   "step-1",
				ToolName: "resume",
				Allowed:  false,
				Code:     RejectionCodeInvalidRuntimeState,
				Reason:   tc.reason,
			})
			if got := history[0].Code; got != tc.want {
				t.Fatalf("AppendAuditHistory()[0].Code = %q, want %q", got, tc.want)
			}
		})
	}
}
