package missioncontrol

import "time"

type RejectionCode string

const (
	RejectionCodeApprovalRequired  RejectionCode = "approval_required"
	RejectionCodeAuthorityExceeded RejectionCode = "authority_exceeded"
	RejectionCodeToolNotAllowed    RejectionCode = "tool_not_allowed"
)

type AuditEvent struct {
	JobID     string        `json:"job_id"`
	StepID    string        `json:"step_id"`
	ToolName  string        `json:"tool_name"`
	Allowed   bool          `json:"allowed"`
	Code      RejectionCode `json:"code"`
	Reason    string        `json:"reason"`
	Timestamp time.Time     `json:"timestamp"`
}
