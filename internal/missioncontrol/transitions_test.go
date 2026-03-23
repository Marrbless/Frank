package missioncontrol

import "testing"

func TestCanTransitionJobAllowed(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		from JobState
		to   JobState
	}{
		{name: "pending to running", from: JobStatePending, to: JobStateRunning},
		{name: "pending to rejected", from: JobStatePending, to: JobStateRejected},
		{name: "running to waiting_user", from: JobStateRunning, to: JobStateWaitingUser},
		{name: "running to paused", from: JobStateRunning, to: JobStatePaused},
		{name: "running to completed", from: JobStateRunning, to: JobStateCompleted},
		{name: "running to failed", from: JobStateRunning, to: JobStateFailed},
		{name: "running to rejected", from: JobStateRunning, to: JobStateRejected},
		{name: "waiting_user to running", from: JobStateWaitingUser, to: JobStateRunning},
		{name: "waiting_user to paused", from: JobStateWaitingUser, to: JobStatePaused},
		{name: "waiting_user to failed", from: JobStateWaitingUser, to: JobStateFailed},
		{name: "paused to running", from: JobStatePaused, to: JobStateRunning},
		{name: "paused to failed", from: JobStatePaused, to: JobStateFailed},
		{name: "completed to completed", from: JobStateCompleted, to: JobStateCompleted},
		{name: "failed to failed", from: JobStateFailed, to: JobStateFailed},
		{name: "rejected to rejected", from: JobStateRejected, to: JobStateRejected},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if !CanTransitionJob(tc.from, tc.to) {
				t.Fatalf("CanTransitionJob(%q, %q) = false, want true", tc.from, tc.to)
			}

			if err := ValidateJobTransition(tc.from, tc.to); err != nil {
				t.Fatalf("ValidateJobTransition(%q, %q) error = %v, want nil", tc.from, tc.to, err)
			}
		})
	}
}

func TestCanTransitionJobInvalid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		from JobState
		to   JobState
	}{
		{name: "pending to completed", from: JobStatePending, to: JobStateCompleted},
		{name: "running to pending", from: JobStateRunning, to: JobStatePending},
		{name: "waiting_user to completed", from: JobStateWaitingUser, to: JobStateCompleted},
		{name: "paused to completed", from: JobStatePaused, to: JobStateCompleted},
		{name: "completed to running", from: JobStateCompleted, to: JobStateRunning},
		{name: "failed to running", from: JobStateFailed, to: JobStateRunning},
		{name: "rejected to pending", from: JobStateRejected, to: JobStatePending},
		{name: "unknown from state", from: JobState("unknown"), to: JobStateRunning},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if CanTransitionJob(tc.from, tc.to) {
				t.Fatalf("CanTransitionJob(%q, %q) = true, want false", tc.from, tc.to)
			}

			err := ValidateJobTransition(tc.from, tc.to)
			if err == nil {
				t.Fatalf("ValidateJobTransition(%q, %q) error = nil, want ValidationError", tc.from, tc.to)
			}

			validationErr, ok := err.(ValidationError)
			if !ok {
				t.Fatalf("ValidateJobTransition(%q, %q) error type = %T, want ValidationError", tc.from, tc.to, err)
			}

			if validationErr.Code != RejectionCodeInvalidJobTransition {
				t.Fatalf("ValidationError.Code = %q, want %q", validationErr.Code, RejectionCodeInvalidJobTransition)
			}

			wantMessage := "invalid job transition from " + `"` + string(tc.from) + `"` + " to " + `"` + string(tc.to) + `"`
			if validationErr.Message != wantMessage {
				t.Fatalf("ValidationError.Message = %q, want %q", validationErr.Message, wantMessage)
			}
		})
	}
}
