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
		{name: "running to completed", from: JobStateRunning, to: JobStateCompleted},
		{name: "running to rejected", from: JobStateRunning, to: JobStateRejected},
		{name: "completed to completed", from: JobStateCompleted, to: JobStateCompleted},
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
		{name: "completed to running", from: JobStateCompleted, to: JobStateRunning},
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
