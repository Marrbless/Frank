package tools

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/local/picobot/internal/chat"
	"github.com/local/picobot/internal/missioncontrol"
	"github.com/local/picobot/internal/providers"
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
	reg.SetMissionRequired(false)
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

	if reg.MissionRequired() {
		t.Fatal("MissionRequired() = true, want false")
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

	if !strings.Contains(err.Error(), "E_INVALID_ACTION_FOR_STEP") {
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

func TestRegistryExecuteGuardDeniesEmitsAuditEvent(t *testing.T) {
	reg := NewRegistry()
	tool := &stubTool{name: "stub", result: "ok"}
	audits := &stubAuditEmitter{}
	guard := &stubGuard{
		decision: missioncontrol.GuardDecision{
			Allowed: false,
			Code:    missioncontrol.RejectionCodeToolNotAllowed,
			Reason:  "tool is outside the step scope",
			Event: missioncontrol.AuditEvent{
				JobID:     "job-1",
				StepID:    "step-1",
				ToolName:  "stub",
				Allowed:   false,
				Code:      missioncontrol.RejectionCodeToolNotAllowed,
				Reason:    "tool is outside the step scope",
				Timestamp: time.Unix(123, 0).UTC(),
			},
		},
	}
	reg.Register(tool)
	reg.SetGuard(guard)
	reg.SetAuditEmitter(audits)

	ctx := missioncontrol.WithExecutionContext(context.Background(), missioncontrol.ExecutionContext{
		Job:  &missioncontrol.Job{ID: "job-1"},
		Step: &missioncontrol.Step{ID: "step-1"},
	})

	logs := captureRegistryLogs(t, func() {
		_, _ = reg.Execute(ctx, "stub", map[string]interface{}{"x": 1})
	})

	if tool.executeCalls != 0 {
		t.Fatalf("tool executeCalls = %d, want %d", tool.executeCalls, 0)
	}

	if len(audits.events) != 1 {
		t.Fatalf("audit event count = %d, want %d", len(audits.events), 1)
	}

	if !reflect.DeepEqual(audits.events[0], guard.decision.Event) {
		t.Fatalf("emitted audit event = %#v, want %#v", audits.events[0], guard.decision.Event)
	}

	if !strings.Contains(logs, "[tool] audit job=job-1 step=step-1 tool=stub allowed=false code=tool_not_allowed reason=tool is outside the step scope") {
		t.Fatalf("expected denied audit log, got %q", logs)
	}

	if got := countAuditLogLines(logs); got != 1 {
		t.Fatalf("audit log line count = %d, want %d in logs %q", got, 1, logs)
	}

	if strings.Contains(logs, "event={") {
		t.Fatalf("expected denied log not to repeat audit event payload, got %q", logs)
	}
}

func TestRegistryExecuteGuardAllowsEmitsAuditEvent(t *testing.T) {
	reg := NewRegistry()
	tool := &stubTool{name: "stub", result: "ok"}
	audits := &stubAuditEmitter{}
	guard := &stubGuard{
		decision: missioncontrol.GuardDecision{
			Allowed: true,
			Event: missioncontrol.AuditEvent{
				JobID:     "job-1",
				StepID:    "step-1",
				ToolName:  "stub",
				Allowed:   true,
				Timestamp: time.Unix(456, 0).UTC(),
			},
		},
	}
	reg.Register(tool)
	reg.SetGuard(guard)
	reg.SetAuditEmitter(audits)

	ctx := missioncontrol.WithExecutionContext(context.Background(), missioncontrol.ExecutionContext{
		Job:  &missioncontrol.Job{ID: "job-1"},
		Step: &missioncontrol.Step{ID: "step-1"},
	})

	logs := captureRegistryLogs(t, func() {
		result, err := reg.Execute(ctx, "stub", map[string]interface{}{"x": 1})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if result != "ok" {
			t.Fatalf("Execute() result = %q, want %q", result, "ok")
		}
	})

	if tool.executeCalls != 1 {
		t.Fatalf("tool executeCalls = %d, want %d", tool.executeCalls, 1)
	}

	if len(audits.events) != 1 {
		t.Fatalf("audit event count = %d, want %d", len(audits.events), 1)
	}

	if !reflect.DeepEqual(audits.events[0], guard.decision.Event) {
		t.Fatalf("emitted audit event = %#v, want %#v", audits.events[0], guard.decision.Event)
	}

	if !strings.Contains(logs, "[tool] audit job=job-1 step=step-1 tool=stub allowed=true") {
		t.Fatalf("expected allowed audit log, got %q", logs)
	}

	if got := countAuditLogLines(logs); got != 1 {
		t.Fatalf("audit log line count = %d, want %d in logs %q", got, 1, logs)
	}
}

func TestRegistryExecuteWithoutExecutionContextPreservesAuditBehavior(t *testing.T) {
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

	logs := captureRegistryLogs(t, func() {
		result, err := reg.Execute(context.Background(), "stub", map[string]interface{}{"x": 1})
		if err != nil {
			t.Fatalf("Execute() error = %v, want nil", err)
		}
		if result != "ok" {
			t.Fatalf("Execute() result = %q, want %q", result, "ok")
		}
	})

	if tool.executeCalls != 1 {
		t.Fatalf("tool executeCalls = %d, want %d", tool.executeCalls, 1)
	}

	if guard.calls != 0 {
		t.Fatalf("guard calls = %d, want %d", guard.calls, 0)
	}

	if strings.Contains(logs, "[tool] audit") {
		t.Fatalf("expected no audit log without execution context, got %q", logs)
	}
}

func TestRegistryExecuteRedactsSensitiveArgsAndRemoteErrorsInLogs(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&stubTool{
		name: "mcp_demo_lookup",
		err:  fmt.Errorf("HTTP 401: {\"token\":\"sk-secret\",\"query\":\"private note\"}"),
	})

	logs := captureRegistryLogs(t, func() {
		_, err := reg.Execute(context.Background(), "mcp_demo_lookup", map[string]interface{}{
			"authorization": "Bearer sk-secret",
			"query":         "private note",
		})
		if err == nil {
			t.Fatal("Execute() error = nil, want tool failure")
		}
	})

	if !strings.Contains(logs, "[tool] → mcp_demo_lookup arg_keys=[authorization query] arg_count=2") {
		t.Fatalf("expected redacted arg summary, got %q", logs)
	}
	if !strings.Contains(logs, "[tool] ✗ mcp_demo_lookup failed after ") || !strings.Contains(logs, "MCP tool failed (HTTP 401)") {
		t.Fatalf("expected redacted MCP failure summary, got %q", logs)
	}
	if strings.Contains(logs, "sk-secret") || strings.Contains(logs, "private note") || strings.Contains(logs, "{\"token\"") {
		t.Fatalf("expected logs to exclude raw args and payloads, got %q", logs)
	}
}

func TestRegistryDefinitionsMissionRequiredWithoutExecutionContextReturnsEmpty(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	reg.Register(&stubTool{name: "alpha"})
	reg.Register(&stubTool{name: "beta"})
	reg.SetMissionRequired(true)

	got := definitionNames(reg.DefinitionsForExecutionContext(nil))
	if len(got) != 0 {
		t.Fatalf("DefinitionsForExecutionContext(nil) = %#v, want empty list", got)
	}
}

func TestRegistryExecuteMissionRequiredWithoutExecutionContextRejectsAndLogsAudit(t *testing.T) {
	reg := NewRegistry()
	reg.SetMissionRequired(true)
	audits := &stubAuditEmitter{}
	tool := &stubTool{name: "stub", result: "ok"}
	reg.Register(tool)
	reg.SetAuditEmitter(audits)

	var err error
	logs := captureRegistryLogs(t, func() {
		_, err = reg.Execute(context.Background(), "stub", map[string]interface{}{"x": 1})
	})

	if err == nil {
		t.Fatal("Execute() error = nil, want mission context rejection")
	}

	if tool.executeCalls != 0 {
		t.Fatalf("tool executeCalls = %d, want %d", tool.executeCalls, 0)
	}

	if !strings.Contains(err.Error(), "E_NO_ACTIVE_STEP") {
		t.Fatalf("error %q does not contain canonical no-active-step code", err)
	}

	if !strings.Contains(err.Error(), "active mission step is required") {
		t.Fatalf("error %q does not contain clear reason", err)
	}

	if !strings.Contains(logs, "[tool] audit job= step= tool=stub allowed=false code=mission_context_required reason=active mission step is required") {
		t.Fatalf("expected mission-required audit log, got %q", logs)
	}

	if got := countAuditLogLines(logs); got != 1 {
		t.Fatalf("audit log line count = %d, want %d in logs %q", got, 1, logs)
	}

	if len(audits.events) != 1 {
		t.Fatalf("audit event count = %d, want %d", len(audits.events), 1)
	}

	if audits.events[0].ToolName != "stub" {
		t.Fatalf("AuditEvent.ToolName = %q, want %q", audits.events[0].ToolName, "stub")
	}

	if audits.events[0].Code != missioncontrol.RejectionCodeMissionContextRequired {
		t.Fatalf("AuditEvent.Code = %q, want %q", audits.events[0].Code, missioncontrol.RejectionCodeMissionContextRequired)
	}

	if audits.events[0].Timestamp.IsZero() {
		t.Fatal("AuditEvent.Timestamp is zero")
	}
}

func TestRegistryDefinitionsWithoutExecutionContextReturnsFullToolSet(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	reg.Register(&stubTool{name: "zeta"})
	reg.Register(&stubTool{name: "alpha"})
	reg.Register(&stubTool{name: "beta"})

	got := definitionNames(reg.DefinitionsForExecutionContext(nil))
	want := []string{"alpha", "beta", "zeta"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("DefinitionsForExecutionContext(nil) = %#v, want %#v", got, want)
	}

	if !reflect.DeepEqual(definitionNames(reg.Definitions()), want) {
		t.Fatalf("Definitions() did not preserve full tool set")
	}
}

func TestRegistryDefinitionsJobLevelFiltering(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	reg.Register(&stubTool{name: "alpha"})
	reg.Register(&stubTool{name: "beta"})
	reg.Register(&stubTool{name: "zeta"})

	ec := &missioncontrol.ExecutionContext{
		Job: &missioncontrol.Job{
			AllowedTools: []string{"zeta", "alpha"},
		},
	}

	got := definitionNames(reg.DefinitionsForExecutionContext(ec))
	want := []string{"alpha", "zeta"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("DefinitionsForExecutionContext(job) = %#v, want %#v", got, want)
	}
}

func TestRegistryDefinitionsStepLevelIntersectionFiltering(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	reg.Register(&stubTool{name: "alpha"})
	reg.Register(&stubTool{name: "beta"})
	reg.Register(&stubTool{name: "zeta"})

	ec := &missioncontrol.ExecutionContext{
		Job: &missioncontrol.Job{
			AllowedTools: []string{"alpha", "beta"},
		},
		Step: &missioncontrol.Step{
			AllowedTools: []string{"beta", "zeta"},
		},
	}

	got := definitionNames(reg.DefinitionsForExecutionContext(ec))
	want := []string{"beta"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("DefinitionsForExecutionContext(job+step) = %#v, want %#v", got, want)
	}
}

func TestSurfacedToolRejectionCodeCanonicalizesFrozenV2Codes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		code   missioncontrol.RejectionCode
		reason string
		want   missioncontrol.RejectionCode
	}{
		{name: "already canonical", code: missioncontrol.RejectionCode("E_WAITING_FOR_USER"), want: missioncontrol.RejectionCode("E_WAITING_FOR_USER")},
		{name: "tool not allowed", code: missioncontrol.RejectionCodeToolNotAllowed, reason: "tool is outside the step scope", want: missioncontrol.RejectionCode("E_INVALID_ACTION_FOR_STEP")},
		{name: "mission context", code: missioncontrol.RejectionCodeMissionContextRequired, reason: "active mission step is required", want: missioncontrol.RejectionCode("E_NO_ACTIVE_STEP")},
		{name: "approval required", code: missioncontrol.RejectionCodeApprovalRequired, reason: "step requires approval", want: missioncontrol.RejectionCode("E_APPROVAL_REQUIRED")},
		{name: "waiting user", code: missioncontrol.RejectionCodeWaitingUser, reason: "job is waiting for user input", want: missioncontrol.RejectionCode("E_WAITING_FOR_USER")},
		{name: "invalid runtime state", code: missioncontrol.RejectionCodeInvalidRuntimeState, reason: "job is not executable while in \"paused\" state", want: missioncontrol.RejectionCode("E_STEP_OUT_OF_ORDER")},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			if got := surfacedToolRejectionCode(tc.code, tc.reason); got != tc.want {
				t.Fatalf("surfacedToolRejectionCode(%q, %q) = %q, want %q", tc.code, tc.reason, got, tc.want)
			}
		})
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

type stubAuditEmitter struct {
	events []missioncontrol.AuditEvent
}

func (s *stubAuditEmitter) EmitAuditEvent(event missioncontrol.AuditEvent) {
	s.events = append(s.events, event)
}

func definitionNames(defs []providers.ToolDefinition) []string {
	names := make([]string, 0, len(defs))
	for _, def := range defs {
		names = append(names, def.Name)
	}
	return names
}

func captureRegistryLogs(t *testing.T, fn func()) string {
	t.Helper()

	var buf bytes.Buffer
	previousWriter := log.Writer()
	log.SetOutput(&buf)
	defer log.SetOutput(previousWriter)

	fn()
	return buf.String()
}

func countAuditLogLines(logs string) int {
	count := 0
	for _, line := range strings.Split(logs, "\n") {
		if strings.Contains(line, "[tool] audit ") {
			count++
		}
	}
	return count
}
