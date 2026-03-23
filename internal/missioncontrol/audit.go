package missioncontrol

import "time"

type RejectionCode string

const (
	RejectionCodeApprovalRequired       RejectionCode = "approval_required"
	RejectionCodeAuthorityExceeded      RejectionCode = "authority_exceeded"
	RejectionCodeToolNotAllowed         RejectionCode = "tool_not_allowed"
	RejectionCodeMissionContextRequired RejectionCode = "mission_context_required"
)

type AuditEvent struct {
	JobID     string        `json:"job_id"`
	StepID    string        `json:"step_id"`
	ToolName  string        `json:"proposed_action"`
	Allowed   bool          `json:"allowed"`
	Code      RejectionCode `json:"error_code,omitempty"`
	Reason    string        `json:"reason,omitempty"`
	Timestamp time.Time     `json:"timestamp"`
}

type AuditEmitter interface {
	EmitAuditEvent(event AuditEvent)
}
