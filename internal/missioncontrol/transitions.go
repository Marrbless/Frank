package missioncontrol

import "fmt"

const RejectionCodeInvalidJobTransition RejectionCode = "invalid_job_transition"

func CanTransitionJob(from, to JobState) bool {
	switch from {
	case JobStatePending:
		return to == JobStateRunning || to == JobStateRejected
	case JobStateRunning:
		return to == JobStateWaitingUser || to == JobStatePaused || to == JobStateCompleted || to == JobStateFailed || to == JobStateRejected
	case JobStateWaitingUser:
		return to == JobStateRunning || to == JobStatePaused || to == JobStateFailed || to == JobStateRejected
	case JobStatePaused:
		return to == JobStateRunning || to == JobStateFailed || to == JobStateRejected
	case JobStateCompleted:
		return to == JobStateCompleted
	case JobStateFailed:
		return to == JobStateFailed
	case JobStateRejected:
		return to == JobStateRejected
	default:
		return false
	}
}

func ValidateJobTransition(from, to JobState) error {
	if CanTransitionJob(from, to) {
		return nil
	}

	return ValidationError{
		Code:    RejectionCodeInvalidJobTransition,
		Message: fmt.Sprintf("invalid job transition from %q to %q", from, to),
	}
}
