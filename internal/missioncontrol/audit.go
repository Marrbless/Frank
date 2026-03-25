package missioncontrol

import (
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"strings"
	"time"
)

type RejectionCode string

type AuditActionClass string

type AuditResult string

const AuditHistoryCap = 64

const (
	RejectionCodeApprovalRequired       RejectionCode = "approval_required"
	RejectionCodeAuthorityExceeded      RejectionCode = "authority_exceeded"
	RejectionCodeToolNotAllowed         RejectionCode = "tool_not_allowed"
	RejectionCodeMissionContextRequired RejectionCode = "mission_context_required"
)

const (
	AuditActionClassToolCall         AuditActionClass = "tool_call"
	AuditActionClassOperatorCommand  AuditActionClass = "operator_command"
	AuditActionClassApprovalDecision AuditActionClass = "approval_decision"
)

const (
	AuditResultAllowed  AuditResult = "allowed"
	AuditResultApplied  AuditResult = "applied"
	AuditResultRejected AuditResult = "rejected"
)

type AuditEvent struct {
	EventID     string           `json:"event_id,omitempty"`
	JobID       string           `json:"job_id"`
	StepID      string           `json:"step_id"`
	ToolName    string           `json:"proposed_action"`
	ActionClass AuditActionClass `json:"action_class,omitempty"`
	Result      AuditResult      `json:"result,omitempty"`
	Allowed     bool             `json:"allowed"`
	Code        RejectionCode    `json:"error_code,omitempty"`
	Reason      string           `json:"reason,omitempty"`
	Timestamp   time.Time        `json:"timestamp"`
}

type AuditEmitter interface {
	EmitAuditEvent(event AuditEvent)
}

func CloneAuditHistory(history []AuditEvent) []AuditEvent {
	if len(history) == 0 {
		return nil
	}
	if len(history) > AuditHistoryCap {
		history = history[len(history)-AuditHistoryCap:]
	}
	cloned := make([]AuditEvent, len(history))
	for i, event := range history {
		cloned[i] = normalizeAuditEvent(event)
	}
	return cloned
}

func AppendAuditHistory(history []AuditEvent, event AuditEvent) []AuditEvent {
	history = append(CloneAuditHistory(history), normalizeAuditEvent(event))
	if len(history) > AuditHistoryCap {
		history = history[len(history)-AuditHistoryCap:]
	}
	return history
}

func normalizeAuditEvent(event AuditEvent) AuditEvent {
	if event.ActionClass == "" {
		event.ActionClass = inferAuditActionClass(event.ToolName)
	}
	if event.Result == "" {
		event.Result = inferAuditResult(event.ActionClass, event.Allowed)
	}
	if event.EventID == "" {
		event.EventID = buildAuditEventID(event)
	}
	return event
}

func inferAuditActionClass(action string) AuditActionClass {
	switch strings.ToLower(strings.TrimSpace(action)) {
	case "pause", "resume", "abort", "status", "set_step":
		return AuditActionClassOperatorCommand
	case "approve", "deny":
		return AuditActionClassApprovalDecision
	default:
		return AuditActionClassToolCall
	}
}

func inferAuditResult(actionClass AuditActionClass, allowed bool) AuditResult {
	if !allowed {
		return AuditResultRejected
	}
	switch actionClass {
	case AuditActionClassOperatorCommand, AuditActionClassApprovalDecision:
		return AuditResultApplied
	default:
		return AuditResultAllowed
	}
}

func buildAuditEventID(event AuditEvent) string {
	sum := sha256.Sum256([]byte(strings.Join([]string{
		event.JobID,
		event.StepID,
		event.ToolName,
		string(event.ActionClass),
		string(event.Result),
		strconv.FormatBool(event.Allowed),
		string(event.Code),
		event.Reason,
		event.Timestamp.UTC().Format(time.RFC3339Nano),
	}, "\x1f")))
	return "ae_" + hex.EncodeToString(sum[:16])
}
