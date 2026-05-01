package agent

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/local/picobot/internal/chat"
	"github.com/local/picobot/internal/config"
	"github.com/local/picobot/internal/missioncontrol"
	"github.com/local/picobot/internal/providers"
)

// Fake provider that returns a tool call on first chat, then returns a final message on second chat.
type FakeProvider struct {
	count          int
	firstToolNames []string
}

func (f *FakeProvider) Chat(ctx context.Context, messages []providers.Message, tools []providers.ToolDefinition, model string) (providers.LLMResponse, error) {
	f.count++
	if f.count == 1 {
		f.firstToolNames = make([]string, 0, len(tools))
		for _, tool := range tools {
			f.firstToolNames = append(f.firstToolNames, tool.Name)
		}
	}
	if f.count == 1 {
		// request message tool
		return providers.LLMResponse{
			Content:      "Invoking message tool",
			HasToolCalls: true,
			ToolCalls:    []providers.ToolCall{{ID: "1", Name: "message", Arguments: map[string]interface{}{"content": "hello from tool"}}},
		}, nil
	}
	return providers.LLMResponse{Content: "All done!"}, nil
}
func (f *FakeProvider) GetDefaultModel() string { return "fake" }

type capturingFinalProvider struct {
	firstToolNames []string
}

func (p *capturingFinalProvider) Chat(ctx context.Context, messages []providers.Message, tools []providers.ToolDefinition, model string) (providers.LLMResponse, error) {
	if p.firstToolNames == nil {
		p.firstToolNames = toolDefinitionNames(tools)
	}
	return providers.LLMResponse{Content: "done"}, nil
}

func (p *capturingFinalProvider) GetDefaultModel() string { return "fake" }

type failingMCPTool struct{}

func (t *failingMCPTool) Name() string { return "mcp_demo_lookup" }

func (t *failingMCPTool) Description() string { return "failing MCP tool" }

func (t *failingMCPTool) Parameters() map[string]interface{} {
	return map[string]interface{}{"type": "object"}
}

func (t *failingMCPTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	return "", fmt.Errorf("HTTP 401: {\"token\":\"sk-secret\",\"detail\":\"private note\"}")
}

type failingMCPToolProvider struct {
	calls           int
	lastToolMessage string
}

func (p *failingMCPToolProvider) Chat(ctx context.Context, messages []providers.Message, tools []providers.ToolDefinition, model string) (providers.LLMResponse, error) {
	p.calls++
	if p.calls == 1 {
		return providers.LLMResponse{
			HasToolCalls: true,
			ToolCalls: []providers.ToolCall{
				{
					ID:   "1",
					Name: "mcp_demo_lookup",
					Arguments: map[string]interface{}{
						"authorization": "Bearer sk-secret",
						"query":         "private note",
					},
				},
			},
		}, nil
	}

	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "tool" {
			p.lastToolMessage = messages[i].Content
			break
		}
	}

	return providers.LLMResponse{Content: "Done"}, nil
}

func (p *failingMCPToolProvider) GetDefaultModel() string { return "fake" }

func TestAgentExecutesToolCall(t *testing.T) {
	b := chat.NewHub(10)
	p := &FakeProvider{}
	ag := NewAgentLoop(b, p, p.GetDefaultModel(), 3, "", nil, nil)

	if _, ok := ag.ActiveMissionStep(); ok {
		t.Fatal("taskState.ExecutionContext() ok = true, want false")
	}

	if ag.MissionRequired() {
		t.Fatal("MissionRequired() = true, want false")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	go ag.Run(ctx)

	// send inbound
	in := chat.Inbound{Channel: "cli", SenderID: "user", ChatID: "one", Content: "trigger"}
	select {
	case b.In <- in:
	default:
		t.Fatalf("couldn't send inbound")
	}

	// expect outbound
	deadline := time.After(1 * time.Second)
	for {
		select {
		case out := <-b.Out:
			if out.Content == "All done!" {
				wantTools := toolDefinitionNames(ag.tools.Definitions())
				if !reflect.DeepEqual(p.firstToolNames, wantTools) {
					t.Fatalf("provider tools = %#v, want %#v", p.firstToolNames, wantTools)
				}
				return
			}
			// otherwise continue waiting until timeout
		case <-deadline:
			t.Fatalf("timeout waiting for final outbound message")
		}
	}
}

func TestAgentLoopRedactsToolActivityAndToolMessageErrors(t *testing.T) {
	b := chat.NewHub(10)
	p := &failingMCPToolProvider{}
	ag := NewAgentLoop(b, p, p.GetDefaultModel(), 3, "", nil, nil)
	ag.tools.Register(&failingMCPTool{})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	go ag.Run(ctx)

	in := chat.Inbound{Channel: "cli", SenderID: "user", ChatID: "one", Content: "trigger"}
	select {
	case b.In <- in:
	default:
		t.Fatalf("couldn't send inbound")
	}

	deadline := time.After(1 * time.Second)
	var outputs []string
	for {
		select {
		case out := <-b.Out:
			outputs = append(outputs, out.Content)
			if out.Content == "Done" {
				joined := strings.Join(outputs, "\n")
				if !strings.Contains(joined, "🤖 Running: mcp_demo_lookup (arg_count=2 arg_types=string:2)") {
					t.Fatalf("expected redacted running notification, got %q", joined)
				}
				if !strings.Contains(joined, "📢 mcp_demo_lookup failed (") || !strings.Contains(joined, "MCP tool failed (HTTP 401)") {
					t.Fatalf("expected redacted failure notification, got %q", joined)
				}
				if strings.Contains(joined, "authorization") || strings.Contains(joined, "query") || strings.Contains(joined, "sk-secret") || strings.Contains(joined, "private note") {
					t.Fatalf("expected outbound notifications to redact secrets, got %q", joined)
				}
				if got, want := p.lastToolMessage, "(tool error) MCP tool failed (HTTP 401)"; got != want {
					t.Fatalf("provider saw tool message %q, want %q", got, want)
				}
				return
			}
		case <-deadline:
			t.Fatalf("timeout waiting for final outbound message; outputs=%q", outputs)
		}
	}
}

func TestAgentLoopDoesNotExposeSpawnTool(t *testing.T) {
	t.Parallel()

	b := chat.NewHub(10)
	p := &FakeProvider{}
	ag := NewAgentLoop(b, p, p.GetDefaultModel(), 3, "", nil)

	for _, name := range toolDefinitionNames(ag.tools.Definitions()) {
		if name == "spawn" {
			t.Fatal("Definitions() unexpectedly exposed spawn")
		}
	}
}

func TestAgentLoopMissionRequiredWithoutContextProviderSeesNoTools(t *testing.T) {
	t.Parallel()

	b := chat.NewHub(10)
	p := &FakeProvider{}
	ag := NewAgentLoop(b, p, p.GetDefaultModel(), 3, "", nil)
	ag.SetMissionRequired(true)

	if !ag.MissionRequired() {
		t.Fatal("MissionRequired() = false, want true")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	go ag.Run(ctx)

	in := chat.Inbound{Channel: "cli", SenderID: "user", ChatID: "one", Content: "trigger"}
	select {
	case b.In <- in:
	default:
		t.Fatalf("couldn't send inbound")
	}

	deadline := time.After(1 * time.Second)
	for {
		select {
		case out := <-b.Out:
			if out.Content == "All done!" {
				if len(p.firstToolNames) != 0 {
					t.Fatalf("provider tools = %#v, want empty list", p.firstToolNames)
				}
				return
			}
		case <-deadline:
			t.Fatalf("timeout waiting for final outbound message")
		}
	}
}

func TestAgentLoopModelToolSuppressionProviderSeesNoTools(t *testing.T) {
	t.Parallel()

	b := chat.NewHub(10)
	p := &capturingFinalProvider{}
	ag := NewAgentLoop(b, p, p.GetDefaultModel(), 3, "", nil)
	ag.SetModelToolDefinitionsAllowed(false)

	out, err := ag.ProcessDirect("trigger", time.Second)
	if err != nil {
		t.Fatalf("ProcessDirect() error = %v", err)
	}
	if out != "done" {
		t.Fatalf("ProcessDirect() = %q, want done", out)
	}
	if len(p.firstToolNames) != 0 {
		t.Fatalf("provider tools = %#v, want empty list", p.firstToolNames)
	}
}

func TestAgentLoopModelToolAllowancePreservesMissionAllowedTools(t *testing.T) {
	t.Parallel()

	b := chat.NewHub(10)
	p := &capturingFinalProvider{}
	ag := NewAgentLoop(b, p, p.GetDefaultModel(), 3, "", nil)
	ag.SetModelToolDefinitionsAllowed(true)
	if err := ag.ActivateMissionStep(testMissionJob([]string{"filesystem", "message"}, []string{"filesystem"}), "build"); err != nil {
		t.Fatalf("ActivateMissionStep() error = %v", err)
	}

	if _, err := ag.ProcessDirect("trigger", time.Second); err != nil {
		t.Fatalf("ProcessDirect() error = %v", err)
	}
	if !reflect.DeepEqual(p.firstToolNames, []string{"filesystem"}) {
		t.Fatalf("provider tools = %#v, want filesystem only", p.firstToolNames)
	}
}

func TestAgentLoopStoresLegacyRawModelRouteRecord(t *testing.T) {
	t.Parallel()

	cfg := config.Config{}
	cfg.Providers.OpenAI = &config.ProviderConfig{APIKey: "legacy-secret", APIBase: "https://legacy.example/v1"}
	reg, err := config.BuildModelRegistry(cfg)
	if err != nil {
		t.Fatalf("BuildModelRegistry() error = %v", err)
	}
	route, err := reg.Route(config.ModelRouteOptions{
		ExplicitModel:          "raw-provider-model",
		AllowRawProviderModel:  true,
		RawProviderModelSource: config.RouteReasonCLIOverride,
	})
	if err != nil {
		t.Fatalf("Route() error = %v", err)
	}

	ag := NewAgentLoop(chat.NewHub(10), &capturingFinalProvider{}, "initial", 3, "", nil)
	ag.SetModelRoute(route)
	got, ok := ag.ModelRoute()
	if !ok {
		t.Fatal("ModelRoute() ok = false, want true")
	}
	if got.SelectedModelRef != config.LegacyModelRef || got.ProviderRef != config.LegacyProviderRef || got.ProviderModel != "raw-provider-model" {
		t.Fatalf("ModelRoute() = %#v, want legacy raw provider model", got)
	}
	if !got.ToolDefinitionsAllowed || got.ToolDefinitionsSuppressed {
		t.Fatalf("tool schema flags = allowed %t suppressed %t, want allowed", got.ToolDefinitionsAllowed, got.ToolDefinitionsSuppressed)
	}
}

func TestAgentLoopStoresAliasRouteRecord(t *testing.T) {
	t.Parallel()

	cfg := routeRecordTestConfig()
	reg, err := config.BuildModelRegistry(cfg)
	if err != nil {
		t.Fatalf("BuildModelRegistry() error = %v", err)
	}
	route, err := reg.Route(config.ModelRouteOptions{ExplicitModel: "best"})
	if err != nil {
		t.Fatalf("Route() error = %v", err)
	}

	ag := NewAgentLoop(chat.NewHub(10), &capturingFinalProvider{}, "initial", 3, "", nil)
	ag.SetModelRoute(route)
	got, ok := ag.ModelRoute()
	if !ok {
		t.Fatal("ModelRoute() ok = false, want true")
	}
	if got.SelectedModelRef != "cloud_reasoning" || got.ProviderRef != "openrouter" || got.ProviderModel != "google/gemini-test" {
		t.Fatalf("ModelRoute() = %#v, want alias-resolved cloud route", got)
	}
	if got.SelectionReason != config.RouteReasonCLIOverride {
		t.Fatalf("selection reason = %q, want cli_override", got.SelectionReason)
	}
	if got.Request.MaxTokens != 8192 || got.Request.TimeoutS != 120 || got.Request.Temperature == nil || *got.Request.Temperature != 0.5 {
		t.Fatalf("request override = %#v, want cloud request values", got.Request)
	}
}

func TestAgentLoopStoresLocalSuppressedRouteRecord(t *testing.T) {
	t.Parallel()

	cfg := routeRecordTestConfig()
	reg, err := config.BuildModelRegistry(cfg)
	if err != nil {
		t.Fatalf("BuildModelRegistry() error = %v", err)
	}
	route, err := reg.Route(config.ModelRouteOptions{ExplicitModel: "phone"})
	if err != nil {
		t.Fatalf("Route() error = %v", err)
	}

	ag := NewAgentLoop(chat.NewHub(10), &capturingFinalProvider{}, "initial", 3, "", nil)
	ag.SetModelRoute(route)
	got, ok := ag.ModelRoute()
	if !ok {
		t.Fatal("ModelRoute() ok = false, want true")
	}
	if got.SelectedModelRef != "local_fast" || got.ProviderRef != "llamacpp_phone" || got.ProviderModel != "qwen3-test-local" {
		t.Fatalf("ModelRoute() = %#v, want local route", got)
	}
	if !got.Capabilities.Local || got.Capabilities.AuthorityTier != config.ModelAuthorityLow {
		t.Fatalf("capabilities = %#v, want local low-authority route", got.Capabilities)
	}
	if got.ToolDefinitionsAllowed || !got.ToolDefinitionsSuppressed {
		t.Fatalf("tool schema flags = allowed %t suppressed %t, want suppressed", got.ToolDefinitionsAllowed, got.ToolDefinitionsSuppressed)
	}

	out, err := ag.ProcessDirect("trigger", time.Second)
	if err != nil {
		t.Fatalf("ProcessDirect() error = %v", err)
	}
	if out != "done" {
		t.Fatalf("ProcessDirect() = %q, want done", out)
	}
	provider := ag.provider.(*capturingFinalProvider)
	if len(provider.firstToolNames) != 0 {
		t.Fatalf("provider tools = %#v, want empty list", provider.firstToolNames)
	}
}

func TestAgentLoopGovernedSkillStatusExposesSelectedActiveAndSkipped(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	writeLoopTestSkill(t, workspace, "good", true)
	writeLoopTestSkill(t, workspace, "unsafe", false)

	b := chat.NewHub(10)
	p := &FakeProvider{}
	ag := NewAgentLoop(b, p, p.GetDefaultModel(), 3, workspace, nil)
	job := missioncontrol.Job{
		ID:           "job-1",
		SpecVersion:  missioncontrol.JobSpecVersionV2,
		MaxAuthority: missioncontrol.AuthorityTierHigh,
		AllowedTools: []string{"filesystem"},
		Plan: missioncontrol.Plan{Steps: []missioncontrol.Step{
			{
				ID:                  "build",
				Type:                missioncontrol.StepTypeOneShotCode,
				AllowedTools:        []string{"filesystem"},
				OneShotArtifactPath: "out.txt",
				SelectedSkills:      []string{"good", "unsafe"},
			},
			{ID: "final", Type: missioncontrol.StepTypeFinalResponse, DependsOn: []string{"build"}},
		}},
	}
	if err := ag.ActivateMissionStep(job, "build"); err != nil {
		t.Fatalf("ActivateMissionStep() error = %v", err)
	}

	status := ag.GovernedSkillStatus()
	if !reflect.DeepEqual(status.Selected, []string{"good", "unsafe"}) {
		t.Fatalf("Selected = %#v, want good and unsafe", status.Selected)
	}
	if len(status.Active) != 1 || status.Active[0].ID != "good" {
		t.Fatalf("Active = %#v, want good", status.Active)
	}
	if len(status.Skipped) != 1 || status.Skipped[0].ID != "unsafe" {
		t.Fatalf("Skipped = %#v, want unsafe", status.Skipped)
	}
	if got := toolDefinitionNames(ag.tools.DefinitionsForExecutionContext(mustActiveMissionStep(t, ag))); !reflect.DeepEqual(got, []string{"filesystem"}) {
		t.Fatalf("exposed tools = %#v, want unchanged allowed tools", got)
	}
}

func TestAgentLoopActivateMissionStepAndActiveMissionStep(t *testing.T) {
	t.Parallel()

	b := chat.NewHub(10)
	p := &FakeProvider{}
	ag := NewAgentLoop(b, p, p.GetDefaultModel(), 3, "", nil)

	job := testMissionJob([]string{"read"}, []string{"read"})
	if err := ag.ActivateMissionStep(job, "build"); err != nil {
		t.Fatalf("ActivateMissionStep() error = %v", err)
	}

	ec, ok := ag.ActiveMissionStep()
	if !ok {
		t.Fatal("ActiveMissionStep() ok = false, want true")
	}

	if ec.Job == nil {
		t.Fatal("ActiveMissionStep().Job = nil, want non-nil")
	}

	if ec.Step == nil {
		t.Fatal("ActiveMissionStep().Step = nil, want non-nil")
	}

	if ec.Job.ID != job.ID {
		t.Fatalf("ActiveMissionStep().Job.ID = %q, want %q", ec.Job.ID, job.ID)
	}

	if ec.Step.ID != "build" {
		t.Fatalf("ActiveMissionStep().Step.ID = %q, want %q", ec.Step.ID, "build")
	}

	job.AllowedTools[0] = "mutated"
	if ec.Job.AllowedTools[0] != "read" {
		t.Fatalf("ActiveMissionStep().Job.AllowedTools[0] = %q, want %q", ec.Job.AllowedTools[0], "read")
	}

	if ec.Step.RequiredAuthority != missioncontrol.AuthorityTierLow {
		t.Fatalf("ActiveMissionStep().Step.RequiredAuthority = %q, want %q", ec.Step.RequiredAuthority, missioncontrol.AuthorityTierLow)
	}

	ec.Job.AllowedTools[0] = "mutated-from-snapshot"
	ec.Step.AllowedTools[0] = "mutated-step-tool"

	stored, ok := ag.ActiveMissionStep()
	if !ok {
		t.Fatal("ActiveMissionStep() ok = false on second read, want true")
	}

	if stored.Job.AllowedTools[0] != "read" {
		t.Fatalf("stored ActiveMissionStep().Job.AllowedTools[0] = %q, want %q", stored.Job.AllowedTools[0], "read")
	}

	if stored.Step.AllowedTools[0] != "read" {
		t.Fatalf("stored ActiveMissionStep().Step.AllowedTools[0] = %q, want %q", stored.Step.AllowedTools[0], "read")
	}
}

func TestAgentLoopLowAuthorityModelReceivesNoDangerousToolSchemas(t *testing.T) {
	b := chat.NewHub(10)
	p := &FakeProvider{}
	ag := NewAgentLoop(b, p, p.GetDefaultModel(), 3, "", nil)
	ag.tools.Register(&failingMCPTool{})
	ag.SetModelRoute(testAuthorityModelRoute(config.ModelAuthorityLow))

	allowed := []string{"message", "read_memory", "list_memory", "web_search", "exec", "filesystem", "mcp_demo_lookup"}
	if err := ag.ActivateMissionStep(testMissionJob(allowed, allowed), "build"); err != nil {
		t.Fatalf("ActivateMissionStep() error = %v", err)
	}

	if got := toolDefinitionNames(ag.activeToolDefinitions()); len(got) != 0 {
		t.Fatalf("activeToolDefinitions() = %#v, want no low-authority dangerous tools", got)
	}
}

func TestAgentLoopMediumAuthorityModelReceivesReadOnlyLocalToolSchemas(t *testing.T) {
	b := chat.NewHub(10)
	p := &FakeProvider{}
	ag := NewAgentLoop(b, p, p.GetDefaultModel(), 3, "", nil)
	ag.SetModelRoute(testAuthorityModelRoute(config.ModelAuthorityMedium))

	allowed := []string{"read_memory", "list_memory", "read_skill", "list_skills", "web_search", "message"}
	if err := ag.ActivateMissionStep(testMissionJob(allowed, allowed), "build"); err != nil {
		t.Fatalf("ActivateMissionStep() error = %v", err)
	}

	got := toolDefinitionNames(ag.activeToolDefinitions())
	want := []string{"list_memory", "list_skills", "read_memory", "read_skill"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("activeToolDefinitions() = %#v, want medium read-only local tools %#v", got, want)
	}
}

func TestAgentLoopMediumAuthorityModelSuppressesUnclassifiedToolSchemas(t *testing.T) {
	b := chat.NewHub(10)
	p := &FakeProvider{}
	ag := NewAgentLoop(b, p, p.GetDefaultModel(), 3, "", nil)
	ag.tools.Register(&failingMCPTool{})
	ag.SetModelRoute(testAuthorityModelRoute(config.ModelAuthorityMedium))

	allowed := []string{"mcp_demo_lookup"}
	if err := ag.ActivateMissionStep(testMissionJob(allowed, allowed), "build"); err != nil {
		t.Fatalf("ActivateMissionStep() error = %v", err)
	}

	if got := toolDefinitionNames(ag.activeToolDefinitions()); len(got) != 0 {
		t.Fatalf("activeToolDefinitions() = %#v, want unclassified MCP suppressed", got)
	}
}

func TestAgentLoopHighAuthorityModelKeepsMissionApprovedToolSchemas(t *testing.T) {
	b := chat.NewHub(10)
	p := &FakeProvider{}
	ag := NewAgentLoop(b, p, p.GetDefaultModel(), 3, "", nil)
	ag.tools.Register(&failingMCPTool{})
	ag.SetModelRoute(testAuthorityModelRoute(config.ModelAuthorityHigh))

	allowed := []string{"exec", "filesystem", "message", "web_search", "mcp_demo_lookup"}
	if err := ag.ActivateMissionStep(testMissionJob(allowed, allowed), "build"); err != nil {
		t.Fatalf("ActivateMissionStep() error = %v", err)
	}

	got := toolDefinitionNames(ag.activeToolDefinitions())
	want := []string{"exec", "filesystem", "mcp_demo_lookup", "message", "web_search"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("activeToolDefinitions() = %#v, want high authority unchanged %#v", got, want)
	}
}

func TestAgentLoopRejectsMissionStepAboveSelectedModelAuthority(t *testing.T) {
	b := chat.NewHub(10)
	p := &FakeProvider{}
	ag := NewAgentLoop(b, p, p.GetDefaultModel(), 3, "", nil)
	ag.SetModelRoute(testAuthorityModelRoute(config.ModelAuthorityLow))
	job := testMissionJob([]string{"read_memory"}, []string{"read_memory"})
	job.Plan.Steps[0].RequiredAuthority = missioncontrol.AuthorityTierHigh

	err := ag.ActivateMissionStep(job, "build")
	if err == nil {
		t.Fatal("ActivateMissionStep() error = nil, want model authority rejection")
	}
	var validationErr missioncontrol.ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("ActivateMissionStep() error = %T %[1]v, want ValidationError", err)
	}
	if validationErr.Code != missioncontrol.RejectionCodeAuthorityExceeded {
		t.Fatalf("ValidationError.Code = %q, want authority_exceeded", validationErr.Code)
	}
	if !strings.Contains(validationErr.Message, "selected model authority tier") {
		t.Fatalf("ValidationError.Message = %q, want selected model authority message", validationErr.Message)
	}
}

func TestAgentLoopModelControlMetricsRecordRouteSuccessAndFallback(t *testing.T) {
	b := chat.NewHub(10)
	p := &FakeProvider{}
	ag := NewAgentLoop(b, p, p.GetDefaultModel(), 3, "", nil)

	route := testAuthorityModelRoute(config.ModelAuthorityHigh)
	route.FallbackDepth = 1
	ag.SetModelRoute(route)

	metrics := ag.ModelControlMetrics()
	if metrics.RouteAttemptCount != 1 || metrics.RouteSuccessCount != 1 || metrics.FallbackCount != 1 {
		t.Fatalf("ModelControlMetrics() = %#v, want one attempt, success, and fallback", metrics)
	}
	if metrics.RouteFailureCount != 0 || metrics.ToolSchemaSuppressedCount != 0 {
		t.Fatalf("ModelControlMetrics() = %#v, want no failure or suppression counts", metrics)
	}
}

func TestAgentLoopModelControlMetricsRecordToolSchemaSuppression(t *testing.T) {
	b := chat.NewHub(10)
	p := &FakeProvider{}
	ag := NewAgentLoop(b, p, p.GetDefaultModel(), 3, "", nil)

	route := testAuthorityModelRoute(config.ModelAuthorityHigh)
	route.ToolDefinitionsAllowed = false
	route.ToolDefinitionsSuppressed = true
	ag.SetModelRoute(route)
	allowed := []string{"message"}
	if err := ag.ActivateMissionStep(testMissionJob(allowed, allowed), "build"); err != nil {
		t.Fatalf("ActivateMissionStep() error = %v", err)
	}
	if got := ag.activeToolDefinitions(); len(got) != 0 {
		t.Fatalf("activeToolDefinitions() = %#v, want no tools", toolDefinitionNames(got))
	}

	metrics := ag.ModelControlMetrics()
	if metrics.ToolSchemaSuppressedCount != 1 {
		t.Fatalf("ModelControlMetrics().ToolSchemaSuppressedCount = %d, want 1; metrics=%#v", metrics.ToolSchemaSuppressedCount, metrics)
	}
}

func TestAgentLoopModelControlMetricsRecordAuthorityFilteringSuppression(t *testing.T) {
	b := chat.NewHub(10)
	p := &FakeProvider{}
	ag := NewAgentLoop(b, p, p.GetDefaultModel(), 3, "", nil)
	ag.SetModelRoute(testAuthorityModelRoute(config.ModelAuthorityMedium))

	allowed := []string{"read_memory", "exec"}
	if err := ag.ActivateMissionStep(testMissionJob(allowed, allowed), "build"); err != nil {
		t.Fatalf("ActivateMissionStep() error = %v", err)
	}
	got := toolDefinitionNames(ag.activeToolDefinitions())
	want := []string{"read_memory"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("activeToolDefinitions() = %#v, want %#v", got, want)
	}

	metrics := ag.ModelControlMetrics()
	if metrics.ToolSchemaSuppressedCount != 1 {
		t.Fatalf("ModelControlMetrics().ToolSchemaSuppressedCount = %d, want 1; metrics=%#v", metrics.ToolSchemaSuppressedCount, metrics)
	}
}

func TestAgentLoopModelControlMetricsRecordAuthorityDenial(t *testing.T) {
	b := chat.NewHub(10)
	p := &FakeProvider{}
	ag := NewAgentLoop(b, p, p.GetDefaultModel(), 3, "", nil)
	ag.SetModelRoute(testAuthorityModelRoute(config.ModelAuthorityLow))
	job := testMissionJob([]string{"read_memory"}, []string{"read_memory"})
	job.Plan.Steps[0].RequiredAuthority = missioncontrol.AuthorityTierHigh

	err := ag.ActivateMissionStep(job, "build")
	if err == nil {
		t.Fatal("ActivateMissionStep() error = nil, want authority denial")
	}
	metrics := ag.ModelControlMetrics()
	if metrics.RouteFailureCount != 1 || metrics.AuthorityDenialCount != 1 {
		t.Fatalf("ModelControlMetrics() = %#v, want one route failure and authority denial", metrics)
	}
}

func TestAgentLoopModelControlMetricsRecordModelPolicyDenial(t *testing.T) {
	b := chat.NewHub(10)
	p := &FakeProvider{}
	ag := NewAgentLoop(b, p, p.GetDefaultModel(), 3, "", nil)
	ag.SetMissionModelRouter(func(job missioncontrol.Job, stepID string) (config.ModelRoute, providers.LLMProvider, missioncontrol.OperatorModelControlMetricsStatus, bool, error) {
		return config.ModelRoute{}, nil, missioncontrol.OperatorModelControlMetricsStatus{}, false, missioncontrol.ValidationError{
			Code:    missioncontrol.RejectionCodeInvalidModelPolicy,
			StepID:  stepID,
			Message: "model_policy denied route",
		}
	})

	err := ag.ActivateMissionStep(testMissionJob([]string{"read_memory"}, []string{"read_memory"}), "build")
	if err == nil {
		t.Fatal("ActivateMissionStep() error = nil, want policy denial")
	}
	metrics := ag.ModelControlMetrics()
	if metrics.RouteAttemptCount != 1 || metrics.RouteFailureCount != 1 || metrics.ModelPolicyDenialCount != 1 {
		t.Fatalf("ModelControlMetrics() = %#v, want one route attempt, failure, and model policy denial", metrics)
	}
}

func TestAgentLoopModelControlMetricsIncludeRouterProviderHealthFailures(t *testing.T) {
	b := chat.NewHub(10)
	p := &FakeProvider{}
	ag := NewAgentLoop(b, p, p.GetDefaultModel(), 3, "", nil)
	ag.SetMissionModelRouter(func(job missioncontrol.Job, stepID string) (config.ModelRoute, providers.LLMProvider, missioncontrol.OperatorModelControlMetricsStatus, bool, error) {
		metrics := missioncontrol.OperatorModelControlMetricsStatus{ProviderHealthFailureCount: 2}
		return testAuthorityModelRoute(config.ModelAuthorityHigh), p, metrics, true, nil
	})

	if err := ag.ActivateMissionStep(testMissionJob([]string{"read_memory"}, []string{"read_memory"}), "build"); err != nil {
		t.Fatalf("ActivateMissionStep() error = %v", err)
	}
	metrics := ag.ModelControlMetrics()
	if metrics.ProviderHealthFailureCount != 2 || metrics.RouteAttemptCount != 1 || metrics.RouteSuccessCount != 1 {
		t.Fatalf("ModelControlMetrics() = %#v, want health failures plus successful route", metrics)
	}
}

func testAuthorityModelRoute(tier config.ModelAuthorityTier) config.ModelRoute {
	return config.ModelRoute{
		SelectedModelRef:       "test_model",
		ProviderRef:            "test_provider",
		ProviderModel:          "test-provider-model",
		ToolDefinitionsAllowed: true,
		Capabilities: config.ModelCapabilities{
			SupportsTools: true,
			AuthorityTier: tier,
		},
	}
}

func toolDefinitionNames(defs []providers.ToolDefinition) []string {
	names := make([]string, 0, len(defs))
	for _, def := range defs {
		names = append(names, def.Name)
	}
	return names
}

func mustActiveMissionStep(t *testing.T, ag *AgentLoop) *missioncontrol.ExecutionContext {
	t.Helper()
	ec, ok := ag.ActiveMissionStep()
	if !ok {
		t.Fatal("ActiveMissionStep() ok = false, want true")
	}
	return &ec
}

func routeRecordTestConfig() config.Config {
	tempCloud := 0.5
	tempLocal := 0.3
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.Model = "cloud_reasoning"
	cfg.Providers.Named = map[string]config.ProviderConfig{
		"openrouter":     {Type: config.ProviderTypeOpenAICompatible, APIKey: "router-secret", APIBase: "https://openrouter.example/v1"},
		"llamacpp_phone": {Type: config.ProviderTypeOpenAICompatible, APIKey: "not-needed", APIBase: "http://127.0.0.1:8080/v1"},
	}
	cfg.Models = map[string]config.ModelProfileConfig{
		"cloud_reasoning": {
			Provider:      "openrouter",
			ProviderModel: "google/gemini-test",
			Capabilities: config.ModelCapabilities{
				SupportsTools: true,
				AuthorityTier: config.ModelAuthorityHigh,
				CostTier:      config.ModelCostStandard,
				LatencyTier:   config.ModelLatencyNormal,
			},
			Request: config.ModelRequestConfig{MaxTokens: 8192, Temperature: &tempCloud, TimeoutS: 120},
		},
		"local_fast": {
			Provider:      "llamacpp_phone",
			ProviderModel: "qwen3-test-local",
			Capabilities: config.ModelCapabilities{
				Local:         true,
				Offline:       true,
				AuthorityTier: config.ModelAuthorityLow,
				CostTier:      config.ModelCostFree,
				LatencyTier:   config.ModelLatencySlow,
			},
			Request: config.ModelRequestConfig{MaxTokens: 1024, Temperature: &tempLocal, TimeoutS: 300},
		},
	}
	cfg.ModelAliases = map[string]string{"best": "cloud_reasoning", "phone": "local_fast"}
	cfg.ModelRouting = config.ModelRoutingConfig{DefaultModel: "cloud_reasoning", LocalPreferredModel: "local_fast"}
	return cfg
}

func writeLoopTestSkill(t *testing.T, workspace string, id string, promptOnly bool) {
	t.Helper()
	dir := filepath.Join(workspace, "skills", id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	content := fmt.Sprintf(`---
id: %s
version: v1
description: Test skill
allowed_activation_scopes: mission_step_prompt
prompt_only: %t
can_affect_tools_or_actions: %t
---

# %s
`, id, promptOnly, !promptOnly, id)
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
