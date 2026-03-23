package missioncontrol

import (
	"context"
	"time"
)

type ExecutionContext struct {
	Job     *Job
	Step    *Step
	Runtime *JobRuntimeState
}

type executionContextKey struct{}

func WithExecutionContext(ctx context.Context, ec ExecutionContext) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, executionContextKey{}, ec)
}

func ExecutionContextFromContext(ctx context.Context) (ExecutionContext, bool) {
	if ctx == nil {
		return ExecutionContext{}, false
	}

	ec, ok := ctx.Value(executionContextKey{}).(ExecutionContext)
	return ec, ok
}

type GuardDecision struct {
	Allowed bool
	Code    RejectionCode
	Reason  string
	Event   AuditEvent
}

type ToolGuard interface {
	EvaluateTool(ctx context.Context, ec ExecutionContext, toolName string, args map[string]interface{}) GuardDecision
}

type defaultToolGuard struct{}

func NewDefaultToolGuard() ToolGuard {
	return defaultToolGuard{}
}

func (defaultToolGuard) EvaluateTool(ctx context.Context, ec ExecutionContext, toolName string, args map[string]interface{}) GuardDecision {
	if ec.Job == nil || ec.Step == nil {
		return newGuardDecision(ec, toolName, false, RejectionCodeToolNotAllowed, "missing job or step context")
	}
	if err := ValidateRuntimeExecution(ec); err != nil {
		validationErr, ok := err.(ValidationError)
		if !ok {
			return newGuardDecision(ec, toolName, false, RejectionCodeInvalidRuntimeState, err.Error())
		}
		return newGuardDecision(ec, toolName, false, validationErr.Code, validationErr.Message)
	}

	if ec.Step.RequiresApproval {
		return newGuardDecision(ec, toolName, false, RejectionCodeApprovalRequired, "step requires approval")
	}

	jobAuthority, jobAuthorityOK := guardAuthorityRank(ec.Job.MaxAuthority)
	stepAuthority, stepAuthorityOK := guardAuthorityRank(ec.Step.RequiredAuthority)
	if stepAuthorityOK && jobAuthorityOK && stepAuthority > jobAuthority {
		return newGuardDecision(ec, toolName, false, RejectionCodeAuthorityExceeded, "step required authority exceeds job max authority")
	}

	if !containsTool(ec.Job.AllowedTools, toolName) {
		return newGuardDecision(ec, toolName, false, RejectionCodeToolNotAllowed, "tool is not allowed by job tool scope")
	}

	if len(ec.Step.AllowedTools) > 0 && !containsTool(ec.Step.AllowedTools, toolName) {
		return newGuardDecision(ec, toolName, false, RejectionCodeToolNotAllowed, "tool is not allowed by step tool scope")
	}

	return newGuardDecision(ec, toolName, true, "", "")
}

func newGuardDecision(ec ExecutionContext, toolName string, allowed bool, code RejectionCode, reason string) GuardDecision {
	event := AuditEvent{
		ToolName:  toolName,
		Allowed:   allowed,
		Code:      code,
		Reason:    reason,
		Timestamp: time.Now(),
	}
	if ec.Job != nil {
		event.JobID = ec.Job.ID
	}
	if ec.Step != nil {
		event.StepID = ec.Step.ID
	}

	return GuardDecision{
		Allowed: allowed,
		Code:    code,
		Reason:  reason,
		Event:   event,
	}
}

func containsTool(allowedTools []string, toolName string) bool {
	for _, allowedTool := range allowedTools {
		if allowedTool == toolName {
			return true
		}
	}
	return false
}

func guardAuthorityRank(tier AuthorityTier) (int, bool) {
	switch tier {
	case AuthorityTierLow:
		return 0, true
	case AuthorityTierMedium:
		return 1, true
	case AuthorityTierHigh:
		return 2, true
	default:
		return 0, false
	}
}
