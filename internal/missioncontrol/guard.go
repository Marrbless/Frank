package missioncontrol

import "context"

type ExecutionContext struct {
	Job  *Job
	Step *Step
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
