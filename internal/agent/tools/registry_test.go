package tools

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/local/picobot/internal/chat"
	"github.com/local/picobot/internal/missioncontrol"
)

func TestMessageToolPublishesOutbound(t *testing.T) {
	b := chat.NewHub(10)
	mt := NewMessageTool(b)
	mt.SetContext("cli", "test-chat")

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	res, err := mt.Execute(ctx, map[string]interface{}{"content": "hello world"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res != "sent" {
		t.Fatalf("expected 'sent' result, got: %s", res)
	}

	select {
	case out := <-b.Out:
		if out.Content != "hello world" {
			t.Fatalf("unexpected content: %s", out.Content)
		}
	default:
		t.Fatalf("no outbound message published")
	}
}

func TestRegistryExecuteWithoutGuardPreservesBehavior(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	tool := &stubTool{name: "stub", result: "ok"}
	reg.Register(tool)

	result, err := reg.Execute(context.Background(), "stub", map[string]interface{}{"x": 1})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if result != "ok" {
		t.Fatalf("Execute() result = %q, want %q", result, "ok")
	}

	if tool.executeCalls != 1 {
		t.Fatalf("tool executeCalls = %d, want %d", tool.executeCalls, 1)
	}
}

func TestRegistryExecuteGuardAllows(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	tool := &stubTool{name: "stub", result: "ok"}
	guard := &stubGuard{
		decision: missioncontrol.GuardDecision{
			Allowed: true,
		},
	}
	reg.Register(tool)
	reg.SetGuard(guard)

	job := &missioncontrol.Job{ID: "job-1"}
	step := &missioncontrol.Step{ID: "step-1"}
	ctx := missioncontrol.WithExecutionContext(context.Background(), missioncontrol.ExecutionContext{
		Job:  job,
		Step: step,
	})

	result, err := reg.Execute(ctx, "stub", map[string]interface{}{"x": 1})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if result != "ok" {
		t.Fatalf("Execute() result = %q, want %q", result, "ok")
	}

	if tool.executeCalls != 1 {
		t.Fatalf("tool executeCalls = %d, want %d", tool.executeCalls, 1)
	}

	if guard.calls != 1 {
		t.Fatalf("guard calls = %d, want %d", guard.calls, 1)
	}

	if guard.lastExecutionContext.Job != job {
		t.Fatalf("guard job = %p, want %p", guard.lastExecutionContext.Job, job)
	}

	if guard.lastExecutionContext.Step != step {
		t.Fatalf("guard step = %p, want %p", guard.lastExecutionContext.Step, step)
	}
}

func TestRegistryExecuteGuardDenies(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	tool := &stubTool{name: "stub", result: "ok"}
	guard := &stubGuard{
		decision: missioncontrol.GuardDecision{
			Allowed: false,
			Code:    missioncontrol.RejectionCodeToolNotAllowed,
			Reason:  "tool is outside the step scope",
			Event: missioncontrol.AuditEvent{
				JobID:    "job-1",
				StepID:   "step-1",
				ToolName: "stub",
				Allowed:  false,
				Code:     missioncontrol.RejectionCodeToolNotAllowed,
				Reason:   "tool is outside the step scope",
			},
		},
	}
	reg.Register(tool)
	reg.SetGuard(guard)

	ctx := missioncontrol.WithExecutionContext(context.Background(), missioncontrol.ExecutionContext{
		Job:  &missioncontrol.Job{ID: "job-1"},
		Step: &missioncontrol.Step{ID: "step-1"},
	})

	result, err := reg.Execute(ctx, "stub", map[string]interface{}{"x": 1})
	if err == nil {
		t.Fatal("Execute() error = nil, want rejection error")
	}

	if result != "" {
		t.Fatalf("Execute() result = %q, want empty string", result)
	}

	if tool.executeCalls != 0 {
		t.Fatalf("tool executeCalls = %d, want %d", tool.executeCalls, 0)
	}

	if guard.calls != 1 {
		t.Fatalf("guard calls = %d, want %d", guard.calls, 1)
	}

	if !strings.Contains(err.Error(), string(missioncontrol.RejectionCodeToolNotAllowed)) {
		t.Fatalf("error %q does not contain rejection code", err)
	}

	if !strings.Contains(err.Error(), "tool is outside the step scope") {
		t.Fatalf("error %q does not contain rejection reason", err)
	}
}

func TestRegistryExecuteWithoutExecutionContextSkipsGuard(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	tool := &stubTool{name: "stub", result: "ok"}
	guard := &stubGuard{
		decision: missioncontrol.GuardDecision{
			Allowed: false,
			Code:    missioncontrol.RejectionCodeToolNotAllowed,
			Reason:  "should not be used",
		},
	}
	reg.Register(tool)
	reg.SetGuard(guard)

	result, err := reg.Execute(context.Background(), "stub", map[string]interface{}{"x": 1})
	if err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}

	if result != "ok" {
		t.Fatalf("Execute() result = %q, want %q", result, "ok")
	}

	if tool.executeCalls != 1 {
		t.Fatalf("tool executeCalls = %d, want %d", tool.executeCalls, 1)
	}

	if guard.calls != 0 {
		t.Fatalf("guard calls = %d, want %d", guard.calls, 0)
	}
}

type stubTool struct {
	name         string
	result       string
	err          error
	executeCalls int
}

func (t *stubTool) Name() string {
	return t.name
}

func (t *stubTool) Description() string {
	return "stub tool"
}

func (t *stubTool) Parameters() map[string]interface{} {
	return nil
}

func (t *stubTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	t.executeCalls++
	return t.result, t.err
}

type stubGuard struct {
	decision             missioncontrol.GuardDecision
	calls                int
	lastExecutionContext missioncontrol.ExecutionContext
	lastToolName         string
	lastArgs             map[string]interface{}
}

func (g *stubGuard) EvaluateTool(ctx context.Context, ec missioncontrol.ExecutionContext, toolName string, args map[string]interface{}) missioncontrol.GuardDecision {
	g.calls++
	g.lastExecutionContext = ec
	g.lastToolName = toolName
	g.lastArgs = args
	return g.decision
}
