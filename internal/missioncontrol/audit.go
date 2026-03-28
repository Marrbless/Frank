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
	AuditActionClassRuntime          AuditActionClass = "runtime"
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
	event.Code = canonicalizeAuditErrorCode(event.Code, event.Reason)
	if event.EventID == "" {
		event.EventID = buildAuditEventID(event)
	}
	return event
}

func canonicalizeAuditErrorCode(code RejectionCode, reason string) RejectionCode {
	if code == "" {
		return ""
	}

	raw := string(code)
	if strings.HasPrefix(raw, "E_") {
		return code
	}

	normalizedReason := strings.ToLower(strings.TrimSpace(reason))

	switch code {
	case RejectionCodeApprovalRequired:
		return RejectionCode("E_APPROVAL_REQUIRED")
	case RejectionCodeAuthorityExceeded:
		return RejectionCode("E_AUTHORITY_EXCEEDED")
	case RejectionCodeToolNotAllowed, RejectionCodeUnknownStep:
		return RejectionCode("E_INVALID_ACTION_FOR_STEP")
	case RejectionCodeMissionContextRequired:
		return RejectionCode("E_NO_ACTIVE_STEP")
	case RejectionCodeWaitingUser:
		return RejectionCode("E_WAITING_FOR_USER")
	case RejectionCodeLongRunningStartForbidden:
		return RejectionCode("E_LONGRUN_START_FORBIDDEN")
	case RejectionCodeResumeApprovalRequired:
		return RejectionCode("E_RESUME_REQUIRES_APPROVAL")
	case RejectionCodeDuplicateStepID,
		RejectionCodeMissingDependencyTarget,
		RejectionCodeDependencyCycle,
		RejectionCodeMissingTerminalFinalStep,
		RejectionCodeInvalidStepType:
		return RejectionCode("E_PLAN_INVALID")
	case RejectionCodeStepValidationFailed,
		RejectionCodeFalseCompletionClaim,
		RejectionCodeValidationRequired:
		return RejectionCode("E_VALIDATION_FAILED")
	case RejectionCodeInvalidJobTransition:
		return RejectionCode("E_STEP_OUT_OF_ORDER")
	case RejectionCodeInvalidRuntimeState:
		switch {
		case strings.Contains(normalizedReason, "requires an active step"), strings.Contains(normalizedReason, "no active step"):
			return RejectionCode("E_NO_ACTIVE_STEP")
		case strings.Contains(normalizedReason, "aborted"):
			return RejectionCode("E_ABORTED")
		default:
			return RejectionCode("E_STEP_OUT_OF_ORDER")
		}
	default:
		return code
	}
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
	case AuditActionClassOperatorCommand, AuditActionClassApprovalDecision, AuditActionClassRuntime:
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
