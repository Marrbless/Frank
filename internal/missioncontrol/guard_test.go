package missioncontrol

import (
	"context"
	"testing"
	"time"
)

func TestExecutionContextRoundTrip(t *testing.T) {
	t.Parallel()

	job := &Job{ID: "job-1"}
	step := &Step{ID: "step-1"}

	ctx := WithExecutionContext(context.Background(), ExecutionContext{
		Job:  job,
		Step: step,
	})

	got, ok := ExecutionContextFromContext(ctx)
	if !ok {
		t.Fatal("ExecutionContextFromContext() ok = false, want true")
	}

	if got.Job != job {
		t.Fatalf("ExecutionContextFromContext().Job = %p, want %p", got.Job, job)
	}

	if got.Step != step {
		t.Fatalf("ExecutionContextFromContext().Step = %p, want %p", got.Step, step)
	}
}

func TestExecutionContextFromContextMissing(t *testing.T) {
	t.Parallel()

	if _, ok := ExecutionContextFromContext(context.Background()); ok {
		t.Fatal("ExecutionContextFromContext() ok = true, want false")
	}
}

func TestDefaultToolGuardAllow(t *testing.T) {
	t.Parallel()

	decision := NewDefaultToolGuard().EvaluateTool(context.Background(), testExecutionContext(), "read", nil)

	if !decision.Allowed {
		t.Fatalf("EvaluateTool().Allowed = false, want true: %#v", decision)
	}

	if decision.Code != "" {
		t.Fatalf("EvaluateTool().Code = %q, want empty", decision.Code)
	}

	if decision.Reason != "" {
		t.Fatalf("EvaluateTool().Reason = %q, want empty", decision.Reason)
	}
}

func TestDefaultToolGuardApprovalRequired(t *testing.T) {
	t.Parallel()

	ec := testExecutionContext()
	ec.Step.RequiresApproval = true

	decision := NewDefaultToolGuard().EvaluateTool(context.Background(), ec, "read", nil)

	assertDenied(t, decision, RejectionCodeApprovalRequired, "step requires approval")
}

func TestDefaultToolGuardAuthorityExceeded(t *testing.T) {
	t.Parallel()

	ec := testExecutionContext()
	ec.Job.MaxAuthority = AuthorityTierLow
	ec.Step.RequiredAuthority = AuthorityTierHigh

	decision := NewDefaultToolGuard().EvaluateTool(context.Background(), ec, "read", nil)

	assertDenied(t, decision, RejectionCodeAuthorityExceeded, "step required authority exceeds job max authority")
}

func TestDefaultToolGuardDeniedByJobToolScope(t *testing.T) {
	t.Parallel()

	ec := testExecutionContext()
	ec.Job.AllowedTools = []string{"write"}

	decision := NewDefaultToolGuard().EvaluateTool(context.Background(), ec, "read", nil)

	assertDenied(t, decision, RejectionCodeToolNotAllowed, "tool is not allowed by job tool scope")
}

func TestDefaultToolGuardDeniedByStepToolScope(t *testing.T) {
	t.Parallel()

	ec := testExecutionContext()
	ec.Step.AllowedTools = []string{"write"}

	decision := NewDefaultToolGuard().EvaluateTool(context.Background(), ec, "read", nil)

	assertDenied(t, decision, RejectionCodeToolNotAllowed, "tool is not allowed by step tool scope")
}

func TestDefaultToolGuardMissingJobOrStepContext(t *testing.T) {
	t.Parallel()

	decision := NewDefaultToolGuard().EvaluateTool(context.Background(), ExecutionContext{}, "read", nil)

	assertDenied(t, decision, RejectionCodeToolNotAllowed, "missing job or step context")
}

func TestDefaultToolGuardEventFieldsPopulated(t *testing.T) {
	t.Parallel()

	ec := testExecutionContext()

	decision := NewDefaultToolGuard().EvaluateTool(context.Background(), ec, "read", nil)

	if decision.Event.JobID != ec.Job.ID {
		t.Fatalf("Event.JobID = %q, want %q", decision.Event.JobID, ec.Job.ID)
	}

	if decision.Event.StepID != ec.Step.ID {
		t.Fatalf("Event.StepID = %q, want %q", decision.Event.StepID, ec.Step.ID)
	}

	if decision.Event.ToolName != "read" {
		t.Fatalf("Event.ToolName = %q, want %q", decision.Event.ToolName, "read")
	}

	if !decision.Event.Allowed {
		t.Fatalf("Event.Allowed = false, want true")
	}

	if decision.Event.Code != decision.Code {
		t.Fatalf("Event.Code = %q, want %q", decision.Event.Code, decision.Code)
	}

	if decision.Event.Reason != decision.Reason {
		t.Fatalf("Event.Reason = %q, want %q", decision.Event.Reason, decision.Reason)
	}
}

func TestDefaultToolGuardEventTimestampNonZero(t *testing.T) {
	t.Parallel()

	decision := NewDefaultToolGuard().EvaluateTool(context.Background(), testExecutionContext(), "read", nil)

	if decision.Event.Timestamp.IsZero() {
		t.Fatal("Event.Timestamp is zero")
	}

	if decision.Event.Timestamp.After(time.Now().Add(time.Second)) {
		t.Fatalf("Event.Timestamp = %v, looks invalid", decision.Event.Timestamp)
	}
}

func assertDenied(t *testing.T, decision GuardDecision, wantCode RejectionCode, wantReason string) {
	t.Helper()

	if decision.Allowed {
		t.Fatalf("EvaluateTool().Allowed = true, want false: %#v", decision)
	}

	if decision.Code != wantCode {
		t.Fatalf("EvaluateTool().Code = %q, want %q", decision.Code, wantCode)
	}

	if decision.Reason != wantReason {
		t.Fatalf("EvaluateTool().Reason = %q, want %q", decision.Reason, wantReason)
	}
}

func testExecutionContext() ExecutionContext {
	job := &Job{
		ID:           "job-1",
		MaxAuthority: AuthorityTierMedium,
		AllowedTools: []string{"read", "write"},
	}
	step := &Step{
		ID:                "step-1",
		RequiredAuthority: AuthorityTierLow,
		AllowedTools:      []string{"read"},
	}
	return ExecutionContext{
		Job:  job,
		Step: step,
	}
}
