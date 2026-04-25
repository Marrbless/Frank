package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/local/picobot/internal/chat"
	"github.com/local/picobot/internal/missioncontrol"
	"github.com/local/picobot/internal/providers"
)

// provider that issues a write_memory tool call on first Chat, and returns a final reply on second
type writeMemoryCallingProvider struct {
	calls int
}

func (p *writeMemoryCallingProvider) Chat(ctx context.Context, messages []providers.Message, tools []providers.ToolDefinition, model string) (providers.LLMResponse, error) {
	p.calls++
	// verify tools include write_memory
	found := false
	for _, t := range tools {
		if t.Name == "write_memory" {
			found = true
			break
		}
	}
	if !found {
		return providers.LLMResponse{}, nil
	}

	if p.calls == 1 {
		args := map[string]interface{}{"target": "today", "content": "Test note", "append": true}
		tc := providers.ToolCall{ID: "1", Name: "write_memory", Arguments: args}
		return providers.LLMResponse{Content: "", HasToolCalls: true, ToolCalls: []providers.ToolCall{tc}}, nil
	}
	return providers.LLMResponse{Content: "Done", HasToolCalls: false}, nil
}
func (p *writeMemoryCallingProvider) GetDefaultModel() string { return "test" }

func TestProcessDirectExecutesToolCall(t *testing.T) {
	b := chat.NewHub(10)
	prov := &writeMemoryCallingProvider{}
	ag := NewAgentLoop(b, prov, prov.GetDefaultModel(), 5, "", nil, nil)

	if ag.MissionRequired() {
		t.Fatal("MissionRequired() = true, want false")
	}

	resp, err := ag.ProcessDirect("please remember Test note", 2*time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp != "Done" {
		t.Fatalf("expected final response 'Done', got '%s'", resp)
	}

	// Verify memory was written to today's note
	mem := ag.memory
	td, err := mem.ReadToday()
	if err != nil {
		t.Fatalf("reading today failed: %v", err)
	}
	if td == "" || !contains(td, "Test note") {
		t.Fatalf("expected today's note to contain Test note, got: %s", td)
	}
}

func TestProcessDirectRedactsProviderErrors(t *testing.T) {
	b := chat.NewHub(10)
	prov := &providerErrorProvider{}
	ag := NewAgentLoop(b, prov, prov.GetDefaultModel(), 5, "", nil, nil)

	_, err := ag.ProcessDirect("hello", 2*time.Second)
	if err == nil {
		t.Fatal("ProcessDirect() error = nil, want provider error")
	}
	if got, want := err.Error(), "OpenAI API error: 401 Unauthorized"; got != want {
		t.Fatalf("ProcessDirect() error = %q, want %q", got, want)
	}
	if strings.Contains(err.Error(), "sk-secret") || strings.Contains(err.Error(), "private note") {
		t.Fatalf("expected ProcessDirect() error to redact provider payload, got %q", err)
	}
}

type deniedMessageToolProvider struct {
	calls int
}

func (p *deniedMessageToolProvider) Chat(ctx context.Context, messages []providers.Message, tools []providers.ToolDefinition, model string) (providers.LLMResponse, error) {
	p.calls++
	if p.calls == 1 {
		return providers.LLMResponse{
			HasToolCalls: true,
			ToolCalls: []providers.ToolCall{
				{
					ID:        "1",
					Name:      "message",
					Arguments: map[string]interface{}{"content": "should not send"},
				},
			},
		}, nil
	}
	return providers.LLMResponse{}, nil
}

func (p *deniedMessageToolProvider) GetDefaultModel() string { return "test" }

type finalResponseProvider struct {
	content string
}

func (p *finalResponseProvider) Chat(ctx context.Context, messages []providers.Message, tools []providers.ToolDefinition, model string) (providers.LLMResponse, error) {
	return providers.LLMResponse{Content: p.content}, nil
}

func (p *finalResponseProvider) GetDefaultModel() string { return "test" }

type providerErrorProvider struct{}

func (p *providerErrorProvider) Chat(ctx context.Context, messages []providers.Message, tools []providers.ToolDefinition, model string) (providers.LLMResponse, error) {
	return providers.LLMResponse{}, fmt.Errorf("OpenAI API error: 401 Unauthorized - {\"token\":\"sk-secret\",\"prompt\":\"private note\"}")
}

func (p *providerErrorProvider) GetDefaultModel() string { return "test" }

type filesystemArtifactProvider struct {
	calls int
}

func (p *filesystemArtifactProvider) Chat(ctx context.Context, messages []providers.Message, tools []providers.ToolDefinition, model string) (providers.LLMResponse, error) {
	p.calls++
	if p.calls == 1 {
		return providers.LLMResponse{
			HasToolCalls: true,
			ToolCalls: []providers.ToolCall{
				{
					ID:   "1",
					Name: "filesystem",
					Arguments: map[string]interface{}{
						"action":  "write",
						"path":    "result.txt",
						"content": "artifact",
					},
				},
				{
					ID:   "2",
					Name: "filesystem",
					Arguments: map[string]interface{}{
						"action": "stat",
						"path":   "result.txt",
					},
				},
			},
		}, nil
	}
	return providers.LLMResponse{Content: "Created result.txt and verified it exists."}, nil
}

func (p *filesystemArtifactProvider) GetDefaultModel() string { return "test" }

type writeMemoryToolCallProvider struct{}

func (p *writeMemoryToolCallProvider) Chat(ctx context.Context, messages []providers.Message, tools []providers.ToolDefinition, model string) (providers.LLMResponse, error) {
	return providers.LLMResponse{
		HasToolCalls: true,
		ToolCalls: []providers.ToolCall{
			{
				ID:        "1",
				Name:      "write_memory",
				Arguments: map[string]interface{}{"target": "today", "content": "budget overrun", "append": true},
			},
		},
	}, nil
}

func (p *writeMemoryToolCallProvider) GetDefaultModel() string { return "test" }

type repeatedFailingMessageToolProvider struct{}

func (p *repeatedFailingMessageToolProvider) Chat(ctx context.Context, messages []providers.Message, tools []providers.ToolDefinition, model string) (providers.LLMResponse, error) {
	return providers.LLMResponse{
		HasToolCalls: true,
		ToolCalls: []providers.ToolCall{
			{ID: "1", Name: "message", Arguments: map[string]interface{}{}},
			{ID: "2", Name: "message", Arguments: map[string]interface{}{}},
			{ID: "3", Name: "message", Arguments: map[string]interface{}{}},
			{ID: "4", Name: "message", Arguments: map[string]interface{}{}},
			{ID: "5", Name: "message", Arguments: map[string]interface{}{}},
		},
	}, nil
}

func (p *repeatedFailingMessageToolProvider) GetDefaultModel() string { return "test" }

type repeatedSuccessfulMessageToolProvider struct {
	calls int
}

func (p *repeatedSuccessfulMessageToolProvider) Chat(ctx context.Context, messages []providers.Message, tools []providers.ToolDefinition, model string) (providers.LLMResponse, error) {
	if p.calls >= 20 {
		return providers.LLMResponse{Content: "Done", HasToolCalls: false}, nil
	}
	p.calls++
	return providers.LLMResponse{
		HasToolCalls: true,
		ToolCalls: []providers.ToolCall{
			{
				ID:   "1",
				Name: "message",
				Arguments: map[string]interface{}{
					"content": "budget check",
				},
			},
		},
	}, nil
}

func (p *repeatedSuccessfulMessageToolProvider) GetDefaultModel() string { return "test" }

type finalResponseAfterNMessagesProvider struct {
	calls    int
	messages int
}

func (p *finalResponseAfterNMessagesProvider) Chat(ctx context.Context, messages []providers.Message, tools []providers.ToolDefinition, model string) (providers.LLMResponse, error) {
	if p.calls >= p.messages {
		return providers.LLMResponse{Content: "Done", HasToolCalls: false}, nil
	}
	p.calls++
	return providers.LLMResponse{
		HasToolCalls: true,
		ToolCalls: []providers.ToolCall{
			{
				ID:   "1",
				Name: "message",
				Arguments: map[string]interface{}{
					"content": "budget check",
				},
			},
		},
	}, nil
}

func (p *finalResponseAfterNMessagesProvider) GetDefaultModel() string { return "test" }

type filesystemStaticArtifactProvider struct {
	calls int
}

func (p *filesystemStaticArtifactProvider) Chat(ctx context.Context, messages []providers.Message, tools []providers.ToolDefinition, model string) (providers.LLMResponse, error) {
	p.calls++
	if p.calls == 1 {
		return providers.LLMResponse{
			HasToolCalls: true,
			ToolCalls: []providers.ToolCall{
				{
					ID:   "1",
					Name: "filesystem",
					Arguments: map[string]interface{}{
						"action":  "write",
						"path":    "report.json",
						"content": "{\n  \"ok\": true\n}\n",
					},
				},
				{
					ID:   "2",
					Name: "filesystem",
					Arguments: map[string]interface{}{
						"action": "stat",
						"path":   "report.json",
					},
				},
				{
					ID:   "3",
					Name: "filesystem",
					Arguments: map[string]interface{}{
						"action": "read",
						"path":   "report.json",
					},
				},
			},
		}, nil
	}
	return providers.LLMResponse{Content: "Created report.json and verified the JSON structure."}, nil
}

func (p *filesystemStaticArtifactProvider) GetDefaultModel() string { return "test" }

func TestProcessDirectRejectsDisallowedToolWhenExecutionContextPresent(t *testing.T) {
	t.Parallel()

	b := chat.NewHub(10)
	prov := &deniedMessageToolProvider{}
	ag := NewAgentLoop(b, prov, prov.GetDefaultModel(), 5, "", nil)
	if err := ag.ActivateMissionStep(testMissionJob([]string{"write_memory"}, []string{"write_memory"}), "build"); err != nil {
		t.Fatalf("ActivateMissionStep() error = %v", err)
	}

	resp, err := ag.ProcessDirect("try to send a message", 2*time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(resp, "E_INVALID_ACTION_FOR_STEP") {
		t.Fatalf("expected canonical rejection code in response, got %q", resp)
	}

	if !strings.Contains(resp, "tool is not allowed by job tool scope") {
		t.Fatalf("expected rejection reason in response, got %q", resp)
	}

	select {
	case out := <-b.Out:
		t.Fatalf("message tool should not have run, but outbound message was published: %#v", out)
	default:
	}

	audits := ag.taskState.AuditEvents()
	if len(audits) != 2 {
		t.Fatalf("AuditEvents() count = %d, want %d", len(audits), 2)
	}
	assertAuditEvent(t, audits[0], "job-1", "build", "message", false, missioncontrol.RejectionCode("E_INVALID_ACTION_FOR_STEP"))
	assertAuditEvent(t, audits[1], "job-1", "build", "step_output", true, "")
}

func TestProcessDirectMissionRequiredWithoutExecutionContextRejectsToolCall(t *testing.T) {
	t.Parallel()

	b := chat.NewHub(10)
	prov := &deniedMessageToolProvider{}
	ag := NewAgentLoop(b, prov, prov.GetDefaultModel(), 5, "", nil)
	ag.SetMissionRequired(true)

	resp, err := ag.ProcessDirect("try to send a message", 2*time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(resp, "E_NO_ACTIVE_STEP") {
		t.Fatalf("expected canonical no-active-step rejection in response, got %q", resp)
	}

	if !strings.Contains(resp, "active mission step is required") {
		t.Fatalf("expected clear mission-required reason in response, got %q", resp)
	}

	select {
	case out := <-b.Out:
		t.Fatalf("message tool should not have run, but outbound message was published: %#v", out)
	default:
	}

	audits := ag.taskState.AuditEvents()
	if len(audits) != 1 {
		t.Fatalf("AuditEvents() count = %d, want %d", len(audits), 1)
	}
	assertAuditEvent(t, audits[0], "", "", "message", false, missioncontrol.RejectionCode("E_NO_ACTIVE_STEP"))
}

func TestProcessDirectMissionRequiredWithActiveMissionStepAllowsTool(t *testing.T) {
	t.Parallel()

	b := chat.NewHub(10)
	prov := &writeMemoryCallingProvider{}
	ag := NewAgentLoop(b, prov, prov.GetDefaultModel(), 5, "", nil)
	ag.SetMissionRequired(true)
	if err := ag.ActivateMissionStep(testMissionJob([]string{"write_memory"}, []string{"write_memory"}), "build"); err != nil {
		t.Fatalf("ActivateMissionStep() error = %v", err)
	}

	resp, err := ag.ProcessDirect("please remember Test note", 2*time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp != "Done" {
		t.Fatalf("expected final response 'Done', got %q", resp)
	}

	td, err := ag.memory.ReadToday()
	if err != nil {
		t.Fatalf("reading today failed: %v", err)
	}
	if td == "" || !contains(td, "Test note") {
		t.Fatalf("expected today's note to contain Test note, got: %s", td)
	}

	ec, ok := ag.ActiveMissionStep()
	if !ok {
		t.Fatal("ActiveMissionStep() ok = false, want true")
	}
	if ec.Runtime == nil {
		t.Fatal("ActiveMissionStep().Runtime = nil, want non-nil")
	}
	if ec.Runtime.State != missioncontrol.JobStateRunning {
		t.Fatalf("ActiveMissionStep().Runtime.State = %q, want %q", ec.Runtime.State, missioncontrol.JobStateRunning)
	}

	audits := ag.taskState.AuditEvents()
	if len(audits) != 2 {
		t.Fatalf("AuditEvents() count = %d, want %d", len(audits), 2)
	}
	assertAuditEvent(t, audits[0], "job-1", "build", "write_memory", true, "")
	assertAuditEvent(t, audits[1], "job-1", "build", "step_output", true, "")
}

func assertAuditEvent(t *testing.T, event missioncontrol.AuditEvent, wantJobID, wantStepID, wantAction string, wantAllowed bool, wantCode missioncontrol.RejectionCode) {
	t.Helper()

	if event.JobID != wantJobID {
		t.Fatalf("AuditEvent.JobID = %q, want %q", event.JobID, wantJobID)
	}
	if event.StepID != wantStepID {
		t.Fatalf("AuditEvent.StepID = %q, want %q", event.StepID, wantStepID)
	}
	if event.ToolName != wantAction {
		t.Fatalf("AuditEvent.ToolName = %q, want %q", event.ToolName, wantAction)
	}
	if event.Allowed != wantAllowed {
		t.Fatalf("AuditEvent.Allowed = %t, want %t", event.Allowed, wantAllowed)
	}
	if event.Code != wantCode {
		t.Fatalf("AuditEvent.Code = %q, want %q", event.Code, wantCode)
	}
	if event.Timestamp.IsZero() {
		t.Fatal("AuditEvent.Timestamp is zero")
	}
}

func assertMissionRuntimeCompletedWithSteps(t *testing.T, ag *AgentLoop, wantCompleted []string) {
	t.Helper()

	if _, ok := ag.ActiveMissionStep(); ok {
		t.Fatal("ActiveMissionStep() ok = true, want false after final completion")
	}

	runtime, ok := ag.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if runtime.State != missioncontrol.JobStateCompleted {
		t.Fatalf("MissionRuntimeState().State = %q, want %q", runtime.State, missioncontrol.JobStateCompleted)
	}
	if runtime.ActiveStepID != "" {
		t.Fatalf("MissionRuntimeState().ActiveStepID = %q, want empty", runtime.ActiveStepID)
	}
	if len(runtime.CompletedSteps) != len(wantCompleted) {
		t.Fatalf("MissionRuntimeState().CompletedSteps = %#v, want %d completed steps", runtime.CompletedSteps, len(wantCompleted))
	}
	for i, want := range wantCompleted {
		if runtime.CompletedSteps[i].StepID != want {
			t.Fatalf("MissionRuntimeState().CompletedSteps[%d].StepID = %q, want %q", i, runtime.CompletedSteps[i].StepID, want)
		}
	}
}

func TestProcessDirectOneShotCodeWithArtifactAndVerificationEvidencePausesStep(t *testing.T) {
	t.Parallel()

	b := chat.NewHub(10)
	prov := &filesystemArtifactProvider{}
	ag := NewAgentLoop(b, prov, prov.GetDefaultModel(), 5, t.TempDir(), nil)
	ag.SetMissionRequired(true)
	if err := ag.ActivateMissionStep(testFilesystemMissionJob(), "build"); err != nil {
		t.Fatalf("ActivateMissionStep() error = %v", err)
	}

	resp, err := ag.ProcessDirect("make the file", 2*time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp != "Created result.txt and verified it exists." {
		t.Fatalf("ProcessDirect() response = %q, want verified artifact response", resp)
	}

	runtime, ok := ag.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if runtime.State != missioncontrol.JobStatePaused {
		t.Fatalf("MissionRuntimeState().State = %q, want %q", runtime.State, missioncontrol.JobStatePaused)
	}
	if len(runtime.CompletedSteps) != 1 || runtime.CompletedSteps[0].StepID != "build" {
		t.Fatalf("MissionRuntimeState().CompletedSteps = %#v, want build completion", runtime.CompletedSteps)
	}
	if _, ok := ag.ActiveMissionStep(); ok {
		t.Fatal("ActiveMissionStep() ok = true, want false after validated one_shot_code completion")
	}
}

func TestProcessDirectLongRunningCodeWithArtifactAndVerificationEvidencePausesStepWithoutStart(t *testing.T) {
	t.Parallel()

	b := chat.NewHub(10)
	prov := &filesystemArtifactProvider{}
	ag := NewAgentLoop(b, prov, prov.GetDefaultModel(), 5, t.TempDir(), nil)
	ag.SetMissionRequired(true)
	if err := ag.ActivateMissionStep(testLongRunningMissionJob(), "build"); err != nil {
		t.Fatalf("ActivateMissionStep() error = %v", err)
	}

	resp, err := ag.ProcessDirect("build the service", 2*time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp != "Created result.txt and verified it exists." {
		t.Fatalf("ProcessDirect() response = %q, want verified artifact response", resp)
	}

	runtime, ok := ag.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if runtime.State != missioncontrol.JobStatePaused {
		t.Fatalf("MissionRuntimeState().State = %q, want %q", runtime.State, missioncontrol.JobStatePaused)
	}
	if len(runtime.CompletedSteps) != 1 || runtime.CompletedSteps[0].StepID != "build" {
		t.Fatalf("MissionRuntimeState().CompletedSteps = %#v, want build completion", runtime.CompletedSteps)
	}
	if _, ok := ag.ActiveMissionStep(); ok {
		t.Fatal("ActiveMissionStep() ok = true, want false after validated long_running_code completion")
	}
}

func TestProcessDirectStaticArtifactWithStructureValidationPausesStep(t *testing.T) {
	t.Parallel()

	b := chat.NewHub(10)
	prov := &filesystemStaticArtifactProvider{}
	ag := NewAgentLoop(b, prov, prov.GetDefaultModel(), 5, t.TempDir(), nil)
	ag.SetMissionRequired(true)
	if err := ag.ActivateMissionStep(testStaticArtifactMissionJob(), "build"); err != nil {
		t.Fatalf("ActivateMissionStep() error = %v", err)
	}

	resp, err := ag.ProcessDirect("make the artifact", 2*time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp != "Created report.json and verified the JSON structure." {
		t.Fatalf("ProcessDirect() response = %q, want verified static artifact response", resp)
	}

	runtime, ok := ag.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if runtime.State != missioncontrol.JobStatePaused {
		t.Fatalf("MissionRuntimeState().State = %q, want %q", runtime.State, missioncontrol.JobStatePaused)
	}
	if len(runtime.CompletedSteps) != 1 || runtime.CompletedSteps[0].StepID != "build" {
		t.Fatalf("MissionRuntimeState().CompletedSteps = %#v, want build completion", runtime.CompletedSteps)
	}
	if _, ok := ag.ActiveMissionStep(); ok {
		t.Fatal("ActiveMissionStep() ok = true, want false after validated static_artifact completion")
	}
}

func TestProcessDirectOneShotCodeThenFinalResponseCompletesJob(t *testing.T) {
	t.Parallel()

	job := testFilesystemMissionJob()

	b := chat.NewHub(10)
	buildProv := &filesystemArtifactProvider{}
	ag := NewAgentLoop(b, buildProv, buildProv.GetDefaultModel(), 5, t.TempDir(), nil)
	ag.SetMissionRequired(true)
	if err := ag.ActivateMissionStep(job, "build"); err != nil {
		t.Fatalf("ActivateMissionStep(build) error = %v", err)
	}

	resp, err := ag.ProcessDirect("make the file", 2*time.Second)
	if err != nil {
		t.Fatalf("build ProcessDirect() error = %v", err)
	}
	if resp != "Created result.txt and verified it exists." {
		t.Fatalf("build ProcessDirect() response = %q, want verified artifact response", resp)
	}

	runtime, ok := ag.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false after build, want true")
	}
	if runtime.State != missioncontrol.JobStatePaused {
		t.Fatalf("MissionRuntimeState().State after build = %q, want %q", runtime.State, missioncontrol.JobStatePaused)
	}
	if len(runtime.CompletedSteps) != 1 || runtime.CompletedSteps[0].StepID != "build" {
		t.Fatalf("MissionRuntimeState().CompletedSteps after build = %#v, want build completion", runtime.CompletedSteps)
	}

	finalProv := &finalResponseProvider{content: "Here is the final answer with the requested artifact completed."}
	ag.provider = finalProv
	if err := ag.ActivateMissionStep(job, "final"); err != nil {
		t.Fatalf("ActivateMissionStep(final) error = %v", err)
	}

	resp, err = ag.ProcessDirect("finish", 2*time.Second)
	if err != nil {
		t.Fatalf("final ProcessDirect() error = %v", err)
	}
	want := finalProv.content + "\n\nMission summary:\nArtifacts: result.txt"
	if resp != want {
		t.Fatalf("final ProcessDirect() response = %q, want %q", resp, want)
	}

	assertMissionRuntimeCompletedWithSteps(t, ag, []string{"build", "final"})
}

func TestProcessDirectStaticArtifactThenFinalResponseCompletesJob(t *testing.T) {
	t.Parallel()

	job := testStaticArtifactMissionJob()

	b := chat.NewHub(10)
	buildProv := &filesystemStaticArtifactProvider{}
	ag := NewAgentLoop(b, buildProv, buildProv.GetDefaultModel(), 5, t.TempDir(), nil)
	ag.SetMissionRequired(true)
	if err := ag.ActivateMissionStep(job, "build"); err != nil {
		t.Fatalf("ActivateMissionStep(build) error = %v", err)
	}

	resp, err := ag.ProcessDirect("make the artifact", 2*time.Second)
	if err != nil {
		t.Fatalf("build ProcessDirect() error = %v", err)
	}
	if resp != "Created report.json and verified the JSON structure." {
		t.Fatalf("build ProcessDirect() response = %q, want verified static artifact response", resp)
	}

	runtime, ok := ag.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false after build, want true")
	}
	if runtime.State != missioncontrol.JobStatePaused {
		t.Fatalf("MissionRuntimeState().State after build = %q, want %q", runtime.State, missioncontrol.JobStatePaused)
	}
	if len(runtime.CompletedSteps) != 1 || runtime.CompletedSteps[0].StepID != "build" {
		t.Fatalf("MissionRuntimeState().CompletedSteps after build = %#v, want build completion", runtime.CompletedSteps)
	}

	finalProv := &finalResponseProvider{content: "Here is the final answer with the requested report delivered."}
	ag.provider = finalProv
	if err := ag.ActivateMissionStep(job, "final"); err != nil {
		t.Fatalf("ActivateMissionStep(final) error = %v", err)
	}

	resp, err = ag.ProcessDirect("finish", 2*time.Second)
	if err != nil {
		t.Fatalf("final ProcessDirect() error = %v", err)
	}
	want := finalProv.content + "\n\nMission summary:\nArtifacts: report.json"
	if resp != want {
		t.Fatalf("final ProcessDirect() response = %q, want %q", resp, want)
	}

	assertMissionRuntimeCompletedWithSteps(t, ag, []string{"build", "final"})
}

func TestProcessDirectFinalResponseTruncatesRawBodyBeforeMissionSummary(t *testing.T) {
	t.Parallel()

	job := testFilesystemMissionJob()

	b := chat.NewHub(10)
	buildProv := &filesystemArtifactProvider{}
	ag := NewAgentLoop(b, buildProv, buildProv.GetDefaultModel(), 5, t.TempDir(), nil)
	ag.SetMissionRequired(true)
	if err := ag.ActivateMissionStep(job, "build"); err != nil {
		t.Fatalf("ActivateMissionStep(build) error = %v", err)
	}

	if _, err := ag.ProcessDirect("make the file", 2*time.Second); err != nil {
		t.Fatalf("build ProcessDirect() error = %v", err)
	}

	finalProv := &finalResponseProvider{content: "Here is the requested artifact output. " + strings.Repeat("A", 5000)}
	ag.provider = finalProv
	if err := ag.ActivateMissionStep(job, "final"); err != nil {
		t.Fatalf("ActivateMissionStep(final) error = %v", err)
	}

	resp, err := ag.ProcessDirect("finish", 2*time.Second)
	if err != nil {
		t.Fatalf("final ProcessDirect() error = %v", err)
	}
	if !strings.Contains(resp, "[final response body truncated; ") {
		t.Fatalf("final ProcessDirect() response = %q, want truncation omission marker", resp)
	}
	if !strings.Contains(resp, "\n\nMission summary:\nArtifacts: result.txt") {
		t.Fatalf("final ProcessDirect() response = %q, want mission summary artifact line", resp)
	}
	if strings.Contains(resp, finalProv.content) {
		t.Fatalf("final ProcessDirect() response unexpectedly preserved the full raw body")
	}

	assertMissionRuntimeCompletedWithSteps(t, ag, []string{"build", "final"})
}

func TestProcessDirectDiscussionSubtypeTransitionsToWaitingUser(t *testing.T) {
	t.Parallel()

	b := chat.NewHub(10)
	prov := &finalResponseProvider{content: "Need approval before continuing."}
	ag := NewAgentLoop(b, prov, prov.GetDefaultModel(), 3, "", nil)
	if err := ag.ActivateMissionStep(testDiscussionMissionJob(), "build"); err != nil {
		t.Fatalf("ActivateMissionStep() error = %v", err)
	}

	resp, err := ag.ProcessDirect("continue", 2*time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp != "Need approval before continuing." {
		t.Fatalf("ProcessDirect() response = %q, want discussion response", resp)
	}

	ec, ok := ag.ActiveMissionStep()
	if !ok {
		t.Fatal("ActiveMissionStep() ok = false, want true")
	}
	if ec.Runtime == nil {
		t.Fatal("ActiveMissionStep().Runtime = nil, want non-nil")
	}
	if ec.Runtime.State != missioncontrol.JobStateWaitingUser {
		t.Fatalf("ActiveMissionStep().Runtime.State = %q, want %q", ec.Runtime.State, missioncontrol.JobStateWaitingUser)
	}
	if ec.Runtime.WaitingReason != "discussion_authorization" {
		t.Fatalf("ActiveMissionStep().Runtime.WaitingReason = %q, want %q", ec.Runtime.WaitingReason, "discussion_authorization")
	}
	if len(ec.Runtime.ApprovalRequests) != 1 {
		t.Fatalf("ActiveMissionStep().Runtime.ApprovalRequests = %#v, want one pending approval", ec.Runtime.ApprovalRequests)
	}
	if ec.Runtime.ApprovalRequests[0].State != missioncontrol.ApprovalStatePending {
		t.Fatalf("ActiveMissionStep().Runtime.ApprovalRequests[0].State = %q, want %q", ec.Runtime.ApprovalRequests[0].State, missioncontrol.ApprovalStatePending)
	}
}

func TestProcessDirectWaitUserSubtypeTransitionsToWaitingUser(t *testing.T) {
	t.Parallel()

	b := chat.NewHub(10)
	prov := &finalResponseProvider{content: "Need approval before continuing."}
	ag := NewAgentLoop(b, prov, prov.GetDefaultModel(), 3, "", nil)
	if err := ag.ActivateMissionStep(testWaitUserMissionJob(), "build"); err != nil {
		t.Fatalf("ActivateMissionStep() error = %v", err)
	}

	resp, err := ag.ProcessDirect("continue", 2*time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp != "Need approval before continuing." {
		t.Fatalf("ProcessDirect() response = %q, want wait_user response", resp)
	}

	ec, ok := ag.ActiveMissionStep()
	if !ok {
		t.Fatal("ActiveMissionStep() ok = false, want true")
	}
	if ec.Runtime == nil {
		t.Fatal("ActiveMissionStep().Runtime = nil, want non-nil")
	}
	if ec.Runtime.State != missioncontrol.JobStateWaitingUser {
		t.Fatalf("ActiveMissionStep().Runtime.State = %q, want %q", ec.Runtime.State, missioncontrol.JobStateWaitingUser)
	}
	if ec.Runtime.WaitingReason != "wait_user_authorization" {
		t.Fatalf("ActiveMissionStep().Runtime.WaitingReason = %q, want %q", ec.Runtime.WaitingReason, "wait_user_authorization")
	}
	if len(ec.Runtime.ApprovalRequests) != 1 {
		t.Fatalf("ActiveMissionStep().Runtime.ApprovalRequests = %#v, want one pending approval", ec.Runtime.ApprovalRequests)
	}
	if ec.Runtime.ApprovalRequests[0].State != missioncontrol.ApprovalStatePending {
		t.Fatalf("ActiveMissionStep().Runtime.ApprovalRequests[0].State = %q, want %q", ec.Runtime.ApprovalRequests[0].State, missioncontrol.ApprovalStatePending)
	}
}

func TestProcessDirectPauseResumeAbortCommandsControlActiveJob(t *testing.T) {
	t.Parallel()

	b := chat.NewHub(10)
	prov := &finalResponseProvider{content: "unused"}
	ag := NewAgentLoop(b, prov, prov.GetDefaultModel(), 3, "", nil)
	if err := ag.ActivateMissionStep(testMissionJob([]string{"read"}, []string{"read"}), "build"); err != nil {
		t.Fatalf("ActivateMissionStep() error = %v", err)
	}

	resp, err := ag.ProcessDirect("PAUSE job-1", 2*time.Second)
	if err != nil {
		t.Fatalf("ProcessDirect(PAUSE) error = %v", err)
	}
	if resp != "Paused job=job-1." {
		t.Fatalf("ProcessDirect(PAUSE) response = %q, want pause acknowledgement", resp)
	}

	ec, ok := ag.ActiveMissionStep()
	if !ok || ec.Runtime == nil {
		t.Fatalf("ActiveMissionStep() = %#v, want paused active step", ec)
	}
	if ec.Runtime.State != missioncontrol.JobStatePaused {
		t.Fatalf("ActiveMissionStep().Runtime.State = %q, want %q", ec.Runtime.State, missioncontrol.JobStatePaused)
	}
	if len(ec.Runtime.CompletedSteps) != 0 {
		t.Fatalf("ActiveMissionStep().Runtime.CompletedSteps = %#v, want empty", ec.Runtime.CompletedSteps)
	}

	resp, err = ag.ProcessDirect("RESUME job-1", 2*time.Second)
	if err != nil {
		t.Fatalf("ProcessDirect(RESUME) error = %v", err)
	}
	if resp != "Resumed job=job-1." {
		t.Fatalf("ProcessDirect(RESUME) response = %q, want resume acknowledgement", resp)
	}

	ec, ok = ag.ActiveMissionStep()
	if !ok || ec.Runtime == nil {
		t.Fatalf("ActiveMissionStep() = %#v, want resumed active step", ec)
	}
	if ec.Runtime.State != missioncontrol.JobStateRunning {
		t.Fatalf("ActiveMissionStep().Runtime.State = %q, want %q", ec.Runtime.State, missioncontrol.JobStateRunning)
	}

	resp, err = ag.ProcessDirect("ABORT job-1", 2*time.Second)
	if err != nil {
		t.Fatalf("ProcessDirect(ABORT) error = %v", err)
	}
	if resp != "Aborted job=job-1." {
		t.Fatalf("ProcessDirect(ABORT) response = %q, want abort acknowledgement", resp)
	}

	if _, ok := ag.ActiveMissionStep(); ok {
		t.Fatal("ActiveMissionStep() ok = true, want false after abort")
	}
	runtime, ok := ag.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if runtime.State != missioncontrol.JobStateAborted {
		t.Fatalf("MissionRuntimeState().State = %q, want %q", runtime.State, missioncontrol.JobStateAborted)
	}
	if len(runtime.AuditHistory) != 5 {
		t.Fatalf("MissionRuntimeState().AuditHistory count = %d, want %d", len(runtime.AuditHistory), 5)
	}
	assertAuditEvent(t, runtime.AuditHistory[0], "job-1", "build", "pause", true, "")
	assertAuditEvent(t, runtime.AuditHistory[1], "job-1", "build", "pause_ack", true, "")
	assertAuditEvent(t, runtime.AuditHistory[2], "job-1", "build", "resume", true, "")
	assertAuditEvent(t, runtime.AuditHistory[3], "job-1", "build", "resume_ack", true, "")
	assertAuditEvent(t, runtime.AuditHistory[4], "job-1", "build", "abort", true, "")

	audits := ag.taskState.AuditEvents()
	if len(audits) != 5 {
		t.Fatalf("AuditEvents() count = %d, want %d", len(audits), 5)
	}
	assertAuditEvent(t, audits[0], "job-1", "build", "pause", true, "")
	assertAuditEvent(t, audits[1], "job-1", "build", "pause_ack", true, "")
	assertAuditEvent(t, audits[2], "job-1", "build", "resume", true, "")
	assertAuditEvent(t, audits[3], "job-1", "build", "resume_ack", true, "")
	assertAuditEvent(t, audits[4], "job-1", "build", "abort", true, "")
}

func TestProcessDirectStatusCommandReturnsDeterministicSummary(t *testing.T) {
	t.Parallel()

	b := chat.NewHub(10)
	prov := &finalResponseProvider{content: "unused"}
	ag := NewAgentLoop(b, prov, prov.GetDefaultModel(), 3, "", nil)
	job := testMissionJob([]string{"read"}, []string{"read"})
	if err := ag.ActivateMissionStep(job, "build"); err != nil {
		t.Fatalf("ActivateMissionStep() error = %v", err)
	}
	ec, ok := ag.ActiveMissionStep()
	if !ok || ec.Runtime == nil {
		t.Fatalf("ActiveMissionStep() = %#v, want active runtime", ec)
	}
	ec.Runtime.ApprovalRequests = []missioncontrol.ApprovalRequest{
		{
			JobID:           job.ID,
			StepID:          "draft",
			RequestedAction: missioncontrol.ApprovalRequestedActionStepComplete,
			Scope:           missioncontrol.ApprovalScopeMissionStep,
			RequestedVia:    missioncontrol.ApprovalRequestedViaRuntime,
			State:           missioncontrol.ApprovalStateDenied,
			RequestedAt:     time.Date(2026, 3, 24, 11, 59, 0, 0, time.UTC),
			ResolvedAt:      time.Date(2026, 3, 24, 12, 0, 0, 0, time.UTC),
		},
		{
			JobID:           job.ID,
			StepID:          "build",
			RequestedAction: missioncontrol.ApprovalRequestedActionStepComplete,
			Scope:           missioncontrol.ApprovalScopeMissionStep,
			RequestedVia:    missioncontrol.ApprovalRequestedViaRuntime,
			State:           missioncontrol.ApprovalStatePending,
			RequestedAt:     time.Date(2026, 3, 24, 12, 1, 0, 0, time.UTC),
			ExpiresAt:       time.Date(2026, 3, 24, 12, 6, 0, 0, time.UTC),
		},
	}
	ag.taskState.SetExecutionContext(ec)
	ag.taskState.EmitAuditEvent(missioncontrol.AuditEvent{
		JobID:     "job-1",
		StepID:    "build",
		ToolName:  "status",
		Allowed:   true,
		Timestamp: time.Date(2026, 3, 24, 12, 0, 0, 0, time.UTC),
	})
	expected := missioncontrol.AppendAuditHistory(nil, missioncontrol.AuditEvent{
		JobID:     "job-1",
		StepID:    "build",
		ToolName:  "status",
		Allowed:   true,
		Timestamp: time.Date(2026, 3, 24, 12, 0, 0, 0, time.UTC),
	})[0]

	resp, err := ag.ProcessDirect("STATUS job-1", 2*time.Second)
	if err != nil {
		t.Fatalf("ProcessDirect(STATUS) error = %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal([]byte(resp), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if got["job_id"] != "job-1" {
		t.Fatalf("job_id = %#v, want %q", got["job_id"], "job-1")
	}
	if got["state"] != string(missioncontrol.JobStateRunning) {
		t.Fatalf("state = %#v, want %q", got["state"], missioncontrol.JobStateRunning)
	}
	if got["active_step_id"] != "build" {
		t.Fatalf("active_step_id = %#v, want %q", got["active_step_id"], "build")
	}
	allowedTools, ok := got["allowed_tools"].([]any)
	if !ok || len(allowedTools) != 1 || allowedTools[0] != "read" {
		t.Fatalf("allowed_tools = %#v, want [%q]", got["allowed_tools"], "read")
	}
	recentAudit, ok := got["recent_audit"].([]any)
	if !ok || len(recentAudit) != 1 {
		t.Fatalf("recent_audit = %#v, want one audit entry", got["recent_audit"])
	}
	entry, ok := recentAudit[0].(map[string]any)
	if !ok {
		t.Fatalf("recent_audit[0] = %#v, want object", recentAudit[0])
	}
	if entry["action"] != "status" {
		t.Fatalf("recent_audit[0].action = %#v, want %q", entry["action"], "status")
	}
	if entry["event_id"] != expected.EventID {
		t.Fatalf("recent_audit[0].event_id = %#v, want %q", entry["event_id"], expected.EventID)
	}
	if entry["action_class"] != string(expected.ActionClass) {
		t.Fatalf("recent_audit[0].action_class = %#v, want %q", entry["action_class"], expected.ActionClass)
	}
	if entry["result"] != string(expected.Result) {
		t.Fatalf("recent_audit[0].result = %#v, want %q", entry["result"], expected.Result)
	}
	approvalHistory, ok := got["approval_history"].([]any)
	if !ok || len(approvalHistory) != 2 {
		t.Fatalf("approval_history = %#v, want two approval entries", got["approval_history"])
	}
	firstHistory, ok := approvalHistory[0].(map[string]any)
	if !ok {
		t.Fatalf("approval_history[0] = %#v, want object", approvalHistory[0])
	}
	if firstHistory["step_id"] != "build" {
		t.Fatalf("approval_history[0].step_id = %#v, want %q", firstHistory["step_id"], "build")
	}
	if firstHistory["requested_at"] != "2026-03-24T12:01:00Z" {
		t.Fatalf("approval_history[0].requested_at = %#v, want %q", firstHistory["requested_at"], "2026-03-24T12:01:00Z")
	}
}

func TestProcessDirectStatusCommandWrongJobDoesNotBind(t *testing.T) {
	t.Parallel()

	b := chat.NewHub(10)
	prov := &finalResponseProvider{content: "unused"}
	ag := NewAgentLoop(b, prov, prov.GetDefaultModel(), 3, "", nil)
	if err := ag.ActivateMissionStep(testMissionJob([]string{"read"}, []string{"read"}), "build"); err != nil {
		t.Fatalf("ActivateMissionStep() error = %v", err)
	}

	_, err := ag.ProcessDirect("STATUS other-job", 2*time.Second)
	if err == nil {
		t.Fatal("ProcessDirect(STATUS other-job) error = nil, want mismatch failure")
	}
	if !strings.Contains(err.Error(), "does not match the active job") {
		t.Fatalf("ProcessDirect(STATUS other-job) error = %q, want job mismatch", err)
	}
}

func TestProcessDirectRollbackRecordCommandCreatesProposalAndPreservesActiveRuntimePackPointer(t *testing.T) {
	t.Parallel()

	root, wantPointer := writeLoopRollbackPromotionFixtures(t)

	b := chat.NewHub(10)
	prov := &finalResponseProvider{content: "unused"}
	ag := NewAgentLoop(b, prov, prov.GetDefaultModel(), 3, "", nil)
	ag.SetMissionStoreRoot(root)
	if err := ag.ActivateMissionStep(testMissionJob([]string{"read"}, []string{"read"}), "build"); err != nil {
		t.Fatalf("ActivateMissionStep() error = %v", err)
	}

	ecBefore, ok := ag.ActiveMissionStep()
	if !ok || ecBefore.Runtime == nil {
		t.Fatalf("ActiveMissionStep() before = %#v, want active runtime", ecBefore)
	}

	resp, err := ag.ProcessDirect("ROLLBACK_RECORD job-1 promotion-1 rollback-1", 2*time.Second)
	if err != nil {
		t.Fatalf("ProcessDirect(ROLLBACK_RECORD) error = %v", err)
	}
	if resp != "Recorded rollback proposal job=job-1 promotion=promotion-1 rollback=rollback-1." {
		t.Fatalf("ProcessDirect(ROLLBACK_RECORD) response = %q, want rollback proposal acknowledgement", resp)
	}

	record, err := missioncontrol.LoadRollbackRecord(root, "rollback-1")
	if err != nil {
		t.Fatalf("LoadRollbackRecord() error = %v", err)
	}
	if record.PromotionID != "promotion-1" {
		t.Fatalf("RollbackRecord.PromotionID = %q, want promotion-1", record.PromotionID)
	}
	if record.HotUpdateID != "hot-update-1" {
		t.Fatalf("RollbackRecord.HotUpdateID = %q, want hot-update-1", record.HotUpdateID)
	}
	if record.OutcomeID != "outcome-1" {
		t.Fatalf("RollbackRecord.OutcomeID = %q, want outcome-1", record.OutcomeID)
	}
	if record.FromPackID != "pack-candidate" {
		t.Fatalf("RollbackRecord.FromPackID = %q, want pack-candidate", record.FromPackID)
	}
	if record.TargetPackID != "pack-base" {
		t.Fatalf("RollbackRecord.TargetPackID = %q, want pack-base", record.TargetPackID)
	}
	if record.LastKnownGoodPackID != "pack-base" {
		t.Fatalf("RollbackRecord.LastKnownGoodPackID = %q, want pack-base", record.LastKnownGoodPackID)
	}
	if record.Reason != "operator requested rollback proposal" {
		t.Fatalf("RollbackRecord.Reason = %q, want operator requested rollback proposal", record.Reason)
	}
	if record.Notes != "derived from promotion promotion-1" {
		t.Fatalf("RollbackRecord.Notes = %q, want derived note", record.Notes)
	}
	if record.CreatedBy != "operator" {
		t.Fatalf("RollbackRecord.CreatedBy = %q, want operator", record.CreatedBy)
	}
	if !record.RollbackAt.Equal(record.CreatedAt) {
		t.Fatalf("RollbackRecord timestamps = (%v, %v), want equal rollback_at and created_at", record.RollbackAt, record.CreatedAt)
	}

	gotPointer, err := missioncontrol.LoadActiveRuntimePackPointer(root)
	if err != nil {
		t.Fatalf("LoadActiveRuntimePackPointer() error = %v", err)
	}
	if gotPointer != wantPointer {
		t.Fatalf("LoadActiveRuntimePackPointer() = %#v, want %#v", gotPointer, wantPointer)
	}

	ecAfter, ok := ag.ActiveMissionStep()
	if !ok || ecAfter.Runtime == nil {
		t.Fatalf("ActiveMissionStep() after = %#v, want active runtime", ecAfter)
	}
	if ecAfter.Runtime.State != ecBefore.Runtime.State {
		t.Fatalf("ActiveMissionStep().Runtime.State = %q, want %q", ecAfter.Runtime.State, ecBefore.Runtime.State)
	}
	if ecAfter.Runtime.ActiveStepID != ecBefore.Runtime.ActiveStepID {
		t.Fatalf("ActiveMissionStep().Runtime.ActiveStepID = %q, want %q", ecAfter.Runtime.ActiveStepID, ecBefore.Runtime.ActiveStepID)
	}

	status, err := ag.ProcessDirect("STATUS job-1", 2*time.Second)
	if err != nil {
		t.Fatalf("ProcessDirect(STATUS) error = %v", err)
	}

	var summary missioncontrol.OperatorStatusSummary
	if err := json.Unmarshal([]byte(status), &summary); err != nil {
		t.Fatalf("json.Unmarshal(status) error = %v", err)
	}
	if summary.RollbackIdentity == nil {
		t.Fatal("RollbackIdentity = nil, want rollback identity block")
	}
	if summary.RollbackIdentity.State != "configured" {
		t.Fatalf("RollbackIdentity.State = %q, want configured", summary.RollbackIdentity.State)
	}
	if len(summary.RollbackIdentity.Rollbacks) != 1 {
		t.Fatalf("RollbackIdentity.Rollbacks len = %d, want 1", len(summary.RollbackIdentity.Rollbacks))
	}
	if summary.RollbackIdentity.Rollbacks[0].RollbackID != "rollback-1" {
		t.Fatalf("RollbackIdentity.Rollbacks[0].RollbackID = %q, want rollback-1", summary.RollbackIdentity.Rollbacks[0].RollbackID)
	}
}

func TestProcessDirectRollbackRecordCommandFailsClosedWhenPromotionIsMissing(t *testing.T) {
	t.Parallel()

	root := t.TempDir()

	b := chat.NewHub(10)
	prov := &finalResponseProvider{content: "unused"}
	ag := NewAgentLoop(b, prov, prov.GetDefaultModel(), 3, "", nil)
	ag.SetMissionStoreRoot(root)
	if err := ag.ActivateMissionStep(testMissionJob([]string{"read"}, []string{"read"}), "build"); err != nil {
		t.Fatalf("ActivateMissionStep() error = %v", err)
	}

	resp, err := ag.ProcessDirect("ROLLBACK_RECORD job-1 missing-promotion rollback-missing", 2*time.Second)
	if err == nil {
		t.Fatal("ProcessDirect(ROLLBACK_RECORD) error = nil, want missing promotion rejection")
	}
	if resp != "" {
		t.Fatalf("ProcessDirect(ROLLBACK_RECORD) response = %q, want empty on rejection", resp)
	}
	if !strings.Contains(err.Error(), missioncontrol.ErrPromotionRecordNotFound.Error()) {
		t.Fatalf("ProcessDirect(ROLLBACK_RECORD) error = %q, want missing promotion rejection", err)
	}
	if _, err := missioncontrol.LoadRollbackRecord(root, "rollback-missing"); err != missioncontrol.ErrRollbackRecordNotFound {
		t.Fatalf("LoadRollbackRecord() error = %v, want %v", err, missioncontrol.ErrRollbackRecordNotFound)
	}
}

func TestProcessDirectHotUpdateGateRecordCommandCreatesOrSelectsGateAndPreservesActiveRuntimePackPointer(t *testing.T) {
	t.Parallel()

	root, wantPointer := writeLoopHotUpdateGateControlFixtures(t)

	beforePointerBytes, err := os.ReadFile(missioncontrol.StoreActiveRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(active pointer before) error = %v", err)
	}

	b := chat.NewHub(10)
	prov := &finalResponseProvider{content: "unused"}
	ag := NewAgentLoop(b, prov, prov.GetDefaultModel(), 3, "", nil)
	ag.SetMissionStoreRoot(root)
	if err := ag.ActivateMissionStep(testMissionJob([]string{"read"}, []string{"read"}), "build"); err != nil {
		t.Fatalf("ActivateMissionStep() error = %v", err)
	}

	resp, err := ag.ProcessDirect("HOT_UPDATE_GATE_RECORD job-1 hot-update-1 pack-candidate", 2*time.Second)
	if err != nil {
		t.Fatalf("ProcessDirect(HOT_UPDATE_GATE_RECORD first) error = %v", err)
	}
	if resp != "Recorded hot-update gate job=job-1 hot_update=hot-update-1 candidate_pack=pack-candidate." {
		t.Fatalf("ProcessDirect(HOT_UPDATE_GATE_RECORD first) response = %q, want create acknowledgement", resp)
	}

	record, err := missioncontrol.LoadHotUpdateGateRecord(root, "hot-update-1")
	if err != nil {
		t.Fatalf("LoadHotUpdateGateRecord() error = %v", err)
	}
	if record.CandidatePackID != "pack-candidate" {
		t.Fatalf("HotUpdateGateRecord.CandidatePackID = %q, want pack-candidate", record.CandidatePackID)
	}
	if record.PreviousActivePackID != "pack-base" {
		t.Fatalf("HotUpdateGateRecord.PreviousActivePackID = %q, want pack-base", record.PreviousActivePackID)
	}
	if record.RollbackTargetPackID != "pack-base" {
		t.Fatalf("HotUpdateGateRecord.RollbackTargetPackID = %q, want pack-base", record.RollbackTargetPackID)
	}
	if !reflect.DeepEqual(record.TargetSurfaces, []string{"skills"}) {
		t.Fatalf("HotUpdateGateRecord.TargetSurfaces = %#v, want [skills]", record.TargetSurfaces)
	}
	if !reflect.DeepEqual(record.SurfaceClasses, []string{"class_1"}) {
		t.Fatalf("HotUpdateGateRecord.SurfaceClasses = %#v, want [class_1]", record.SurfaceClasses)
	}
	if record.ReloadMode != missioncontrol.HotUpdateReloadModeSkillReload {
		t.Fatalf("HotUpdateGateRecord.ReloadMode = %q, want skill_reload", record.ReloadMode)
	}
	if record.CompatibilityContractRef != "compat-v1" {
		t.Fatalf("HotUpdateGateRecord.CompatibilityContractRef = %q, want compat-v1", record.CompatibilityContractRef)
	}
	if record.State != missioncontrol.HotUpdateGateStatePrepared {
		t.Fatalf("HotUpdateGateRecord.State = %q, want prepared", record.State)
	}
	if record.Decision != missioncontrol.HotUpdateGateDecisionKeepStaged {
		t.Fatalf("HotUpdateGateRecord.Decision = %q, want keep_staged", record.Decision)
	}

	firstGateBytes, err := os.ReadFile(missioncontrol.StoreHotUpdateGatePath(root, "hot-update-1"))
	if err != nil {
		t.Fatalf("ReadFile(first gate) error = %v", err)
	}

	resp, err = ag.ProcessDirect("HOT_UPDATE_GATE_RECORD job-1 hot-update-1 pack-candidate", 2*time.Second)
	if err != nil {
		t.Fatalf("ProcessDirect(HOT_UPDATE_GATE_RECORD second) error = %v", err)
	}
	if resp != "Selected hot-update gate job=job-1 hot_update=hot-update-1 candidate_pack=pack-candidate." {
		t.Fatalf("ProcessDirect(HOT_UPDATE_GATE_RECORD second) response = %q, want select acknowledgement", resp)
	}

	secondGateBytes, err := os.ReadFile(missioncontrol.StoreHotUpdateGatePath(root, "hot-update-1"))
	if err != nil {
		t.Fatalf("ReadFile(second gate) error = %v", err)
	}
	if string(firstGateBytes) != string(secondGateBytes) {
		t.Fatalf("hot-update gate file changed on select path\nfirst:\n%s\nsecond:\n%s", string(firstGateBytes), string(secondGateBytes))
	}

	gotPointer, err := missioncontrol.LoadActiveRuntimePackPointer(root)
	if err != nil {
		t.Fatalf("LoadActiveRuntimePackPointer() error = %v", err)
	}
	if gotPointer != wantPointer {
		t.Fatalf("LoadActiveRuntimePackPointer() = %#v, want %#v", gotPointer, wantPointer)
	}

	afterPointerBytes, err := os.ReadFile(missioncontrol.StoreActiveRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(active pointer after) error = %v", err)
	}
	if string(beforePointerBytes) != string(afterPointerBytes) {
		t.Fatalf("active pointer changed on hot-update gate record path\nbefore:\n%s\nafter:\n%s", string(beforePointerBytes), string(afterPointerBytes))
	}

	status, err := ag.ProcessDirect("STATUS job-1", 2*time.Second)
	if err != nil {
		t.Fatalf("ProcessDirect(STATUS) error = %v", err)
	}

	var summary missioncontrol.OperatorStatusSummary
	if err := json.Unmarshal([]byte(status), &summary); err != nil {
		t.Fatalf("json.Unmarshal(status) error = %v", err)
	}
	if summary.HotUpdateGateIdentity == nil {
		t.Fatal("HotUpdateGateIdentity = nil, want hot-update gate identity block")
	}
	if summary.HotUpdateGateIdentity.State != "configured" {
		t.Fatalf("HotUpdateGateIdentity.State = %q, want configured", summary.HotUpdateGateIdentity.State)
	}
	if len(summary.HotUpdateGateIdentity.Gates) != 1 {
		t.Fatalf("HotUpdateGateIdentity.Gates len = %d, want 1", len(summary.HotUpdateGateIdentity.Gates))
	}
	if summary.HotUpdateGateIdentity.Gates[0].HotUpdateID != "hot-update-1" {
		t.Fatalf("HotUpdateGateIdentity.Gates[0].HotUpdateID = %q, want hot-update-1", summary.HotUpdateGateIdentity.Gates[0].HotUpdateID)
	}
	if summary.HotUpdateGateIdentity.Gates[0].CandidatePackID != "pack-candidate" {
		t.Fatalf("HotUpdateGateIdentity.Gates[0].CandidatePackID = %q, want pack-candidate", summary.HotUpdateGateIdentity.Gates[0].CandidatePackID)
	}
}

func TestProcessDirectHotUpdateGateFromDecisionCommandCreatesSelectsAndPreservesSourceRuntimeState(t *testing.T) {
	t.Parallel()

	root, decision := writeLoopCandidatePromotionDecisionGateFixtures(t, true)
	hotUpdateID := "hot-update-" + decision.PromotionDecisionID
	before := snapshotLoopCandidateDecisionGateSideEffects(t, root, decision)

	ag := newLoopHotUpdateOutcomeAgent(t, root)

	resp, err := ag.ProcessDirect("HOT_UPDATE_GATE_FROM_DECISION job-1 "+decision.PromotionDecisionID, 2*time.Second)
	if err != nil {
		t.Fatalf("ProcessDirect(HOT_UPDATE_GATE_FROM_DECISION first) error = %v", err)
	}
	wantCreated := "Created hot-update gate from decision job=job-1 promotion_decision=" + decision.PromotionDecisionID + " hot_update=" + hotUpdateID + "."
	if resp != wantCreated {
		t.Fatalf("ProcessDirect(HOT_UPDATE_GATE_FROM_DECISION first) response = %q, want %q", resp, wantCreated)
	}

	record, err := missioncontrol.LoadHotUpdateGateRecord(root, hotUpdateID)
	if err != nil {
		t.Fatalf("LoadHotUpdateGateRecord() error = %v", err)
	}
	if record.CandidatePackID != decision.CandidatePackID {
		t.Fatalf("HotUpdateGateRecord.CandidatePackID = %q, want %q", record.CandidatePackID, decision.CandidatePackID)
	}
	if record.PreviousActivePackID != decision.BaselinePackID {
		t.Fatalf("HotUpdateGateRecord.PreviousActivePackID = %q, want %q", record.PreviousActivePackID, decision.BaselinePackID)
	}
	if record.RollbackTargetPackID != decision.BaselinePackID {
		t.Fatalf("HotUpdateGateRecord.RollbackTargetPackID = %q, want %q", record.RollbackTargetPackID, decision.BaselinePackID)
	}
	if record.State != missioncontrol.HotUpdateGateStatePrepared {
		t.Fatalf("HotUpdateGateRecord.State = %q, want prepared", record.State)
	}
	if record.Decision != missioncontrol.HotUpdateGateDecisionKeepStaged {
		t.Fatalf("HotUpdateGateRecord.Decision = %q, want keep_staged", record.Decision)
	}

	firstGateBytes, err := os.ReadFile(missioncontrol.StoreHotUpdateGatePath(root, hotUpdateID))
	if err != nil {
		t.Fatalf("ReadFile(first gate) error = %v", err)
	}
	audits := ag.taskState.AuditEvents()
	if len(audits) == 0 {
		t.Fatal("AuditEvents() count = 0, want hot-update gate from decision audit event")
	}
	assertAuditEvent(t, audits[len(audits)-1], "job-1", "build", "hot_update_gate_from_decision", true, "")

	resp, err = ag.ProcessDirect("HOT_UPDATE_GATE_FROM_DECISION job-1 "+decision.PromotionDecisionID, 2*time.Second)
	if err != nil {
		t.Fatalf("ProcessDirect(HOT_UPDATE_GATE_FROM_DECISION second) error = %v", err)
	}
	wantSelected := "Selected hot-update gate from decision job=job-1 promotion_decision=" + decision.PromotionDecisionID + " hot_update=" + hotUpdateID + "."
	if resp != wantSelected {
		t.Fatalf("ProcessDirect(HOT_UPDATE_GATE_FROM_DECISION second) response = %q, want %q", resp, wantSelected)
	}

	secondGateBytes, err := os.ReadFile(missioncontrol.StoreHotUpdateGatePath(root, hotUpdateID))
	if err != nil {
		t.Fatalf("ReadFile(second gate) error = %v", err)
	}
	if string(firstGateBytes) != string(secondGateBytes) {
		t.Fatalf("hot-update gate file changed on decision replay\nfirst:\n%s\nsecond:\n%s", string(firstGateBytes), string(secondGateBytes))
	}
	audits = ag.taskState.AuditEvents()
	if len(audits) == 0 {
		t.Fatal("AuditEvents() count = 0, want selected hot-update gate from decision audit event")
	}
	assertAuditEvent(t, audits[len(audits)-1], "job-1", "build", "hot_update_gate_from_decision", true, "")

	assertLoopCandidateDecisionGateSideEffectsUnchanged(t, root, decision, before)
	assertLoopCandidateDecisionGateNoTerminalRecords(t, root)
}

func TestProcessDirectHotUpdateGateFromDecisionCommandRejectsMalformedArguments(t *testing.T) {
	t.Parallel()

	root, decision := writeLoopCandidatePromotionDecisionGateFixtures(t, false)
	ag := newLoopHotUpdateOutcomeAgent(t, root)

	tests := []string{
		"HOT_UPDATE_GATE_FROM_DECISION job-1",
		"HOT_UPDATE_GATE_FROM_DECISION job-1 " + decision.PromotionDecisionID + " extra",
	}
	for _, command := range tests {
		command := command
		t.Run(command, func(t *testing.T) {
			resp, err := ag.ProcessDirect(command, 2*time.Second)
			if err == nil {
				t.Fatalf("ProcessDirect(%s) error = nil, want malformed argument rejection", command)
			}
			if resp != "" {
				t.Fatalf("ProcessDirect(%s) response = %q, want empty on rejection", command, resp)
			}
			if !strings.Contains(err.Error(), "HOT_UPDATE_GATE_FROM_DECISION requires job_id and promotion_decision_id") {
				t.Fatalf("ProcessDirect(%s) error = %q, want malformed argument context", command, err)
			}
		})
	}
}

func TestProcessDirectHotUpdateGateFromDecisionCommandFailsClosed(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name       string
		setup      func(t *testing.T) (string, string, string)
		wantErr    string
		wantGateID string
	}
	tests := []testCase{
		{
			name: "wrong job id",
			setup: func(t *testing.T) (string, string, string) {
				root, decision := writeLoopCandidatePromotionDecisionGateFixtures(t, false)
				return root, "other-job", decision.PromotionDecisionID
			},
			wantErr: "operator command does not match the active job",
		},
		{
			name: "missing decision",
			setup: func(t *testing.T) (string, string, string) {
				return t.TempDir(), "job-1", "candidate-promotion-decision-missing"
			},
			wantErr:    missioncontrol.ErrCandidatePromotionDecisionRecordNotFound.Error(),
			wantGateID: "hot-update-candidate-promotion-decision-missing",
		},
		{
			name: "non-selected decision",
			setup: func(t *testing.T) (string, string, string) {
				root, decision := writeLoopCandidatePromotionDecisionGateFixtures(t, false)
				decision.Decision = missioncontrol.CandidatePromotionDecision("discarded")
				if err := missioncontrol.WriteStoreJSONAtomic(missioncontrol.StoreCandidatePromotionDecisionPath(root, decision.PromotionDecisionID), decision); err != nil {
					t.Fatalf("WriteStoreJSONAtomic(non-selected decision) error = %v", err)
				}
				return root, "job-1", decision.PromotionDecisionID
			},
			wantErr: `decision must be "selected_for_promotion"`,
		},
		{
			name: "stale derived eligibility",
			setup: func(t *testing.T) (string, string, string) {
				root, decision := writeLoopCandidatePromotionDecisionGateFixtures(t, false)
				result, err := missioncontrol.LoadCandidateResultRecord(root, decision.ResultID)
				if err != nil {
					t.Fatalf("LoadCandidateResultRecord() error = %v", err)
				}
				result.HoldoutScore = result.BaselineScore
				if err := missioncontrol.WriteStoreJSONAtomic(missioncontrol.StoreCandidateResultPath(root, result.ResultID), result); err != nil {
					t.Fatalf("WriteStoreJSONAtomic(stale eligibility) error = %v", err)
				}
				return root, "job-1", decision.PromotionDecisionID
			},
			wantErr: `promotion eligibility state "rejected"`,
		},
		{
			name: "missing linked run",
			setup: func(t *testing.T) (string, string, string) {
				root, decision := writeLoopCandidatePromotionDecisionGateFixtures(t, false)
				if err := os.Remove(missioncontrol.StoreImprovementRunPath(root, decision.RunID)); err != nil {
					t.Fatalf("Remove(improvement run) error = %v", err)
				}
				return root, "job-1", decision.PromotionDecisionID
			},
			wantErr: missioncontrol.ErrImprovementRunRecordNotFound.Error(),
		},
		{
			name: "stale active pointer",
			setup: func(t *testing.T) (string, string, string) {
				root, decision := writeLoopCandidatePromotionDecisionGateFixtures(t, false)
				if err := missioncontrol.StoreRuntimePackRecord(root, missioncontrol.RuntimePackRecord{
					PackID:                   "pack-other-active",
					ParentPackID:             decision.BaselinePackID,
					CreatedAt:                time.Date(2026, 4, 25, 19, 0, 0, 0, time.UTC),
					Channel:                  "phone",
					PromptPackRef:            "prompt-pack-other-active",
					SkillPackRef:             "skill-pack-other-active",
					ManifestRef:              "manifest-other-active",
					ExtensionPackRef:         "extension-other-active",
					PolicyRef:                "policy-other-active",
					SourceSummary:            "other active pack",
					MutableSurfaces:          []string{"skills"},
					ImmutableSurfaces:        []string{"policy", "authority"},
					SurfaceClasses:           []string{"class_1"},
					CompatibilityContractRef: "compat-v1",
					RollbackTargetPackID:     decision.BaselinePackID,
				}); err != nil {
					t.Fatalf("StoreRuntimePackRecord(pack-other-active) error = %v", err)
				}
				if err := missioncontrol.StoreActiveRuntimePackPointer(root, missioncontrol.ActiveRuntimePackPointer{
					ActivePackID:        "pack-other-active",
					LastKnownGoodPackID: "pack-base",
					UpdatedAt:           time.Date(2026, 4, 25, 19, 1, 0, 0, time.UTC),
					UpdatedBy:           "operator",
					UpdateRecordRef:     "other-hot-update",
					ReloadGeneration:    3,
				}); err != nil {
					t.Fatalf("StoreActiveRuntimePackPointer(stale) error = %v", err)
				}
				return root, "job-1", decision.PromotionDecisionID
			},
			wantErr: "requires active runtime pack pointer active_pack_id",
		},
		{
			name: "missing candidate rollback target",
			setup: func(t *testing.T) (string, string, string) {
				root, decision := writeLoopCandidatePromotionDecisionGateFixtures(t, false)
				pack, err := missioncontrol.LoadRuntimePackRecord(root, decision.CandidatePackID)
				if err != nil {
					t.Fatalf("LoadRuntimePackRecord(candidate) error = %v", err)
				}
				pack.RollbackTargetPackID = ""
				if err := missioncontrol.WriteStoreJSONAtomic(missioncontrol.StoreRuntimePackPath(root, pack.PackID), pack); err != nil {
					t.Fatalf("WriteStoreJSONAtomic(candidate without rollback target) error = %v", err)
				}
				return root, "job-1", decision.PromotionDecisionID
			},
			wantErr: "rollback_target_pack_id is required",
		},
		{
			name: "missing rollback target runtime pack",
			setup: func(t *testing.T) (string, string, string) {
				root, decision := writeLoopCandidatePromotionDecisionGateFixtures(t, false)
				pack, err := missioncontrol.LoadRuntimePackRecord(root, decision.CandidatePackID)
				if err != nil {
					t.Fatalf("LoadRuntimePackRecord(candidate) error = %v", err)
				}
				pack.RollbackTargetPackID = "pack-missing-rollback"
				if err := missioncontrol.WriteStoreJSONAtomic(missioncontrol.StoreRuntimePackPath(root, pack.PackID), pack); err != nil {
					t.Fatalf("WriteStoreJSONAtomic(candidate missing rollback pack) error = %v", err)
				}
				return root, "job-1", decision.PromotionDecisionID
			},
			wantErr: "pack-missing-rollback",
		},
		{
			name: "mismatched decision result authority",
			setup: func(t *testing.T) (string, string, string) {
				root, decision := writeLoopCandidatePromotionDecisionGateFixtures(t, false)
				if err := missioncontrol.StoreRuntimePackRecord(root, missioncontrol.RuntimePackRecord{
					PackID:                   "pack-other-candidate",
					ParentPackID:             decision.BaselinePackID,
					CreatedAt:                time.Date(2026, 4, 25, 19, 30, 0, 0, time.UTC),
					Channel:                  "phone",
					PromptPackRef:            "prompt-pack-other-candidate",
					SkillPackRef:             "skill-pack-other-candidate",
					ManifestRef:              "manifest-other-candidate",
					ExtensionPackRef:         "extension-other-candidate",
					PolicyRef:                "policy-other-candidate",
					SourceSummary:            "other candidate pack",
					MutableSurfaces:          []string{"skills"},
					ImmutableSurfaces:        []string{"policy", "authority"},
					SurfaceClasses:           []string{"class_1"},
					CompatibilityContractRef: "compat-v1",
					RollbackTargetPackID:     decision.BaselinePackID,
				}); err != nil {
					t.Fatalf("StoreRuntimePackRecord(pack-other-candidate) error = %v", err)
				}
				decision.CandidatePackID = "pack-other-candidate"
				if err := missioncontrol.WriteStoreJSONAtomic(missioncontrol.StoreCandidatePromotionDecisionPath(root, decision.PromotionDecisionID), decision); err != nil {
					t.Fatalf("WriteStoreJSONAtomic(decision mismatch) error = %v", err)
				}
				return root, "job-1", decision.PromotionDecisionID
			},
			wantErr: "does not match candidate result candidate_pack_id",
		},
		{
			name: "divergent duplicate gate",
			setup: func(t *testing.T) (string, string, string) {
				root, decision := writeLoopCandidatePromotionDecisionGateFixtures(t, false)
				hotUpdateID := "hot-update-" + decision.PromotionDecisionID
				if _, _, err := missioncontrol.CreateHotUpdateGateFromCandidatePromotionDecision(root, decision.PromotionDecisionID, "operator", time.Date(2026, 4, 25, 20, 0, 0, 0, time.UTC)); err != nil {
					t.Fatalf("CreateHotUpdateGateFromCandidatePromotionDecision(setup) error = %v", err)
				}
				gate, err := missioncontrol.LoadHotUpdateGateRecord(root, hotUpdateID)
				if err != nil {
					t.Fatalf("LoadHotUpdateGateRecord(setup) error = %v", err)
				}
				gate.Objective = "manually diverged objective"
				if err := missioncontrol.WriteStoreJSONAtomic(missioncontrol.StoreHotUpdateGatePath(root, hotUpdateID), gate); err != nil {
					t.Fatalf("WriteStoreJSONAtomic(divergent gate) error = %v", err)
				}
				return root, "job-1", decision.PromotionDecisionID
			},
			wantErr: "already exists with divergent candidate promotion decision authority",
		},
		{
			name: "existing deterministic gate with different candidate pack",
			setup: func(t *testing.T) (string, string, string) {
				root, decision := writeLoopCandidatePromotionDecisionGateFixtures(t, false)
				if err := missioncontrol.StoreRuntimePackRecord(root, missioncontrol.RuntimePackRecord{
					PackID:                   "pack-other-candidate",
					ParentPackID:             decision.BaselinePackID,
					CreatedAt:                time.Date(2026, 4, 25, 20, 30, 0, 0, time.UTC),
					Channel:                  "phone",
					PromptPackRef:            "prompt-pack-other-candidate",
					SkillPackRef:             "skill-pack-other-candidate",
					ManifestRef:              "manifest-other-candidate",
					ExtensionPackRef:         "extension-other-candidate",
					PolicyRef:                "policy-other-candidate",
					SourceSummary:            "other candidate pack",
					MutableSurfaces:          []string{"skills"},
					ImmutableSurfaces:        []string{"policy", "authority"},
					SurfaceClasses:           []string{"class_1"},
					CompatibilityContractRef: "compat-v1",
					RollbackTargetPackID:     decision.BaselinePackID,
				}); err != nil {
					t.Fatalf("StoreRuntimePackRecord(pack-other-candidate) error = %v", err)
				}
				if err := missioncontrol.StoreHotUpdateGateRecord(root, missioncontrol.HotUpdateGateRecord{
					HotUpdateID:              "hot-update-" + decision.PromotionDecisionID,
					Objective:                "operator requested hot-update gate for different candidate",
					CandidatePackID:          "pack-other-candidate",
					PreviousActivePackID:     decision.BaselinePackID,
					RollbackTargetPackID:     decision.BaselinePackID,
					TargetSurfaces:           []string{"skills"},
					SurfaceClasses:           []string{"class_1"},
					ReloadMode:               missioncontrol.HotUpdateReloadModeSkillReload,
					CompatibilityContractRef: "compat-v1",
					PreparedAt:               time.Date(2026, 4, 25, 20, 31, 0, 0, time.UTC),
					State:                    missioncontrol.HotUpdateGateStatePrepared,
					Decision:                 missioncontrol.HotUpdateGateDecisionKeepStaged,
				}); err != nil {
					t.Fatalf("StoreHotUpdateGateRecord(existing different candidate) error = %v", err)
				}
				return root, "job-1", decision.PromotionDecisionID
			},
			wantErr: "does not match candidate promotion decision candidate_pack_id",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			root, jobID, promotionDecisionID := tt.setup(t)
			ag := newLoopHotUpdateOutcomeAgent(t, root)

			resp, err := ag.ProcessDirect("HOT_UPDATE_GATE_FROM_DECISION "+jobID+" "+promotionDecisionID, 2*time.Second)
			if err == nil {
				t.Fatal("ProcessDirect(HOT_UPDATE_GATE_FROM_DECISION) error = nil, want fail-closed rejection")
			}
			if resp != "" {
				t.Fatalf("ProcessDirect(HOT_UPDATE_GATE_FROM_DECISION) response = %q, want empty on rejection", resp)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("ProcessDirect(HOT_UPDATE_GATE_FROM_DECISION) error = %q, want substring %q", err, tt.wantErr)
			}
			hotUpdateID := tt.wantGateID
			if hotUpdateID == "" {
				hotUpdateID = "hot-update-" + promotionDecisionID
			}
			if tt.name != "divergent duplicate gate" && tt.name != "existing deterministic gate with different candidate pack" {
				if _, err := missioncontrol.LoadHotUpdateGateRecord(root, hotUpdateID); err != missioncontrol.ErrHotUpdateGateRecordNotFound {
					t.Fatalf("LoadHotUpdateGateRecord(%s) error = %v, want %v", hotUpdateID, err, missioncontrol.ErrHotUpdateGateRecordNotFound)
				}
			}
		})
	}
}

func TestProcessDirectHotUpdateGatePhaseCommandAdvancesGateAndPreservesActiveRuntimePackPointer(t *testing.T) {
	t.Parallel()

	root, wantPointer := writeLoopHotUpdateGateControlFixtures(t)

	beforePointerBytes, err := os.ReadFile(missioncontrol.StoreActiveRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(active pointer before) error = %v", err)
	}

	b := chat.NewHub(10)
	prov := &finalResponseProvider{content: "unused"}
	ag := NewAgentLoop(b, prov, prov.GetDefaultModel(), 3, "", nil)
	ag.SetMissionStoreRoot(root)
	if err := ag.ActivateMissionStep(testMissionJob([]string{"read"}, []string{"read"}), "build"); err != nil {
		t.Fatalf("ActivateMissionStep() error = %v", err)
	}

	if _, err := ag.ProcessDirect("HOT_UPDATE_GATE_RECORD job-1 hot-update-1 pack-candidate", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(HOT_UPDATE_GATE_RECORD) error = %v", err)
	}

	resp, err := ag.ProcessDirect("HOT_UPDATE_GATE_PHASE job-1 hot-update-1 validated", 2*time.Second)
	if err != nil {
		t.Fatalf("ProcessDirect(HOT_UPDATE_GATE_PHASE validated) error = %v", err)
	}
	if resp != "Advanced hot-update gate job=job-1 hot_update=hot-update-1 phase=validated." {
		t.Fatalf("ProcessDirect(HOT_UPDATE_GATE_PHASE validated) response = %q, want validated acknowledgement", resp)
	}

	record, err := missioncontrol.LoadHotUpdateGateRecord(root, "hot-update-1")
	if err != nil {
		t.Fatalf("LoadHotUpdateGateRecord(validated) error = %v", err)
	}
	if record.State != missioncontrol.HotUpdateGateStateValidated {
		t.Fatalf("HotUpdateGateRecord.State = %q, want validated", record.State)
	}
	if record.PhaseUpdatedAt.IsZero() {
		t.Fatal("HotUpdateGateRecord.PhaseUpdatedAt = zero, want populated")
	}
	if record.PhaseUpdatedBy != "operator" {
		t.Fatalf("HotUpdateGateRecord.PhaseUpdatedBy = %q, want operator", record.PhaseUpdatedBy)
	}

	resp, err = ag.ProcessDirect("HOT_UPDATE_GATE_PHASE job-1 hot-update-1 staged", 2*time.Second)
	if err != nil {
		t.Fatalf("ProcessDirect(HOT_UPDATE_GATE_PHASE staged) error = %v", err)
	}
	if resp != "Advanced hot-update gate job=job-1 hot_update=hot-update-1 phase=staged." {
		t.Fatalf("ProcessDirect(HOT_UPDATE_GATE_PHASE staged) response = %q, want staged acknowledgement", resp)
	}

	record, err = missioncontrol.LoadHotUpdateGateRecord(root, "hot-update-1")
	if err != nil {
		t.Fatalf("LoadHotUpdateGateRecord(staged) error = %v", err)
	}
	if record.State != missioncontrol.HotUpdateGateStateStaged {
		t.Fatalf("HotUpdateGateRecord.State = %q, want staged", record.State)
	}

	gotPointer, err := missioncontrol.LoadActiveRuntimePackPointer(root)
	if err != nil {
		t.Fatalf("LoadActiveRuntimePackPointer() error = %v", err)
	}
	if gotPointer != wantPointer {
		t.Fatalf("LoadActiveRuntimePackPointer() = %#v, want %#v", gotPointer, wantPointer)
	}

	afterPointerBytes, err := os.ReadFile(missioncontrol.StoreActiveRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(active pointer after) error = %v", err)
	}
	if string(beforePointerBytes) != string(afterPointerBytes) {
		t.Fatalf("active pointer changed on hot-update gate phase path\nbefore:\n%s\nafter:\n%s", string(beforePointerBytes), string(afterPointerBytes))
	}

	status, err := ag.ProcessDirect("STATUS job-1", 2*time.Second)
	if err != nil {
		t.Fatalf("ProcessDirect(STATUS) error = %v", err)
	}

	var summary missioncontrol.OperatorStatusSummary
	if err := json.Unmarshal([]byte(status), &summary); err != nil {
		t.Fatalf("json.Unmarshal(status) error = %v", err)
	}
	if summary.HotUpdateGateIdentity == nil {
		t.Fatal("HotUpdateGateIdentity = nil, want hot-update gate identity block")
	}
	if len(summary.HotUpdateGateIdentity.Gates) != 1 {
		t.Fatalf("HotUpdateGateIdentity.Gates len = %d, want 1", len(summary.HotUpdateGateIdentity.Gates))
	}
	if summary.HotUpdateGateIdentity.Gates[0].State != "staged" {
		t.Fatalf("HotUpdateGateIdentity.Gates[0].State = %q, want staged", summary.HotUpdateGateIdentity.Gates[0].State)
	}
}

func TestProcessDirectHotUpdateGateExecuteCommandSwitchesPointerAndIsReplaySafe(t *testing.T) {
	t.Parallel()

	root, _ := writeLoopHotUpdateGateControlFixtures(t)
	now := time.Date(2026, 4, 22, 12, 0, 0, 0, time.UTC)

	if err := missioncontrol.StoreLastKnownGoodRuntimePackPointer(root, missioncontrol.LastKnownGoodRuntimePackPointer{
		PackID:            "pack-base",
		Basis:             "holdout_pass",
		VerifiedAt:        now.Add(-30 * time.Second),
		VerifiedBy:        "operator",
		RollbackRecordRef: "bootstrap",
	}); err != nil {
		t.Fatalf("StoreLastKnownGoodRuntimePackPointer() error = %v", err)
	}

	beforeLKGBytes, err := os.ReadFile(missioncontrol.StoreLastKnownGoodRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(last-known-good before) error = %v", err)
	}

	b := chat.NewHub(10)
	prov := &finalResponseProvider{content: "unused"}
	ag := NewAgentLoop(b, prov, prov.GetDefaultModel(), 3, "", nil)
	ag.SetMissionStoreRoot(root)
	if err := ag.ActivateMissionStep(testMissionJob([]string{"read"}, []string{"read"}), "build"); err != nil {
		t.Fatalf("ActivateMissionStep() error = %v", err)
	}

	if _, err := ag.ProcessDirect("HOT_UPDATE_GATE_RECORD job-1 hot-update-1 pack-candidate", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(HOT_UPDATE_GATE_RECORD) error = %v", err)
	}
	if _, err := ag.ProcessDirect("HOT_UPDATE_GATE_PHASE job-1 hot-update-1 validated", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(HOT_UPDATE_GATE_PHASE validated) error = %v", err)
	}
	if _, err := ag.ProcessDirect("HOT_UPDATE_GATE_PHASE job-1 hot-update-1 staged", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(HOT_UPDATE_GATE_PHASE staged) error = %v", err)
	}

	resp, err := ag.ProcessDirect("HOT_UPDATE_GATE_EXECUTE job-1 hot-update-1", 2*time.Second)
	if err != nil {
		t.Fatalf("ProcessDirect(HOT_UPDATE_GATE_EXECUTE first) error = %v", err)
	}
	if resp != "Executed hot-update pointer switch job=job-1 hot_update=hot-update-1." {
		t.Fatalf("ProcessDirect(HOT_UPDATE_GATE_EXECUTE first) response = %q, want execute acknowledgement", resp)
	}

	pointer, err := missioncontrol.LoadActiveRuntimePackPointer(root)
	if err != nil {
		t.Fatalf("LoadActiveRuntimePackPointer() error = %v", err)
	}
	if pointer.ActivePackID != "pack-candidate" {
		t.Fatalf("ActiveRuntimePackPointer.ActivePackID = %q, want pack-candidate", pointer.ActivePackID)
	}
	if pointer.PreviousActivePackID != "pack-base" {
		t.Fatalf("ActiveRuntimePackPointer.PreviousActivePackID = %q, want pack-base", pointer.PreviousActivePackID)
	}
	if pointer.UpdateRecordRef != "hot_update:hot-update-1" {
		t.Fatalf("ActiveRuntimePackPointer.UpdateRecordRef = %q, want hot_update:hot-update-1", pointer.UpdateRecordRef)
	}
	if pointer.ReloadGeneration != 3 {
		t.Fatalf("ActiveRuntimePackPointer.ReloadGeneration = %d, want 3", pointer.ReloadGeneration)
	}

	record, err := missioncontrol.LoadHotUpdateGateRecord(root, "hot-update-1")
	if err != nil {
		t.Fatalf("LoadHotUpdateGateRecord() error = %v", err)
	}
	if record.State != missioncontrol.HotUpdateGateStateReloading {
		t.Fatalf("HotUpdateGateRecord.State = %q, want reloading", record.State)
	}

	resp, err = ag.ProcessDirect("HOT_UPDATE_GATE_EXECUTE job-1 hot-update-1", 2*time.Second)
	if err != nil {
		t.Fatalf("ProcessDirect(HOT_UPDATE_GATE_EXECUTE second) error = %v", err)
	}
	if resp != "Selected hot-update pointer switch job=job-1 hot_update=hot-update-1." {
		t.Fatalf("ProcessDirect(HOT_UPDATE_GATE_EXECUTE second) response = %q, want idempotent acknowledgement", resp)
	}

	replayedPointer, err := missioncontrol.LoadActiveRuntimePackPointer(root)
	if err != nil {
		t.Fatalf("LoadActiveRuntimePackPointer(replayed) error = %v", err)
	}
	if replayedPointer.ReloadGeneration != 3 {
		t.Fatalf("replayedPointer.ReloadGeneration = %d, want 3", replayedPointer.ReloadGeneration)
	}

	afterLKGBytes, err := os.ReadFile(missioncontrol.StoreLastKnownGoodRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(last-known-good after) error = %v", err)
	}
	if string(beforeLKGBytes) != string(afterLKGBytes) {
		t.Fatalf("last-known-good pointer changed on hot-update execute path\nbefore:\n%s\nafter:\n%s", string(beforeLKGBytes), string(afterLKGBytes))
	}

	status, err := ag.ProcessDirect("STATUS job-1", 2*time.Second)
	if err != nil {
		t.Fatalf("ProcessDirect(STATUS) error = %v", err)
	}

	var summary missioncontrol.OperatorStatusSummary
	if err := json.Unmarshal([]byte(status), &summary); err != nil {
		t.Fatalf("json.Unmarshal(status) error = %v", err)
	}
	if summary.HotUpdateGateIdentity == nil {
		t.Fatal("HotUpdateGateIdentity = nil, want hot-update gate identity block")
	}
	if len(summary.HotUpdateGateIdentity.Gates) != 1 {
		t.Fatalf("HotUpdateGateIdentity.Gates len = %d, want 1", len(summary.HotUpdateGateIdentity.Gates))
	}
	if summary.HotUpdateGateIdentity.Gates[0].State != "reloading" {
		t.Fatalf("HotUpdateGateIdentity.Gates[0].State = %q, want reloading", summary.HotUpdateGateIdentity.Gates[0].State)
	}
	if summary.RuntimePackIdentity == nil {
		t.Fatal("RuntimePackIdentity = nil, want runtime pack identity")
	}
	if summary.RuntimePackIdentity.Active.ActivePackID != "pack-candidate" {
		t.Fatalf("RuntimePackIdentity.Active.ActivePackID = %q, want pack-candidate", summary.RuntimePackIdentity.Active.ActivePackID)
	}
}

func TestProcessDirectHotUpdateGateReloadCommandRecordsConvergenceResultWithoutFurtherPointerMutation(t *testing.T) {
	t.Parallel()

	root, _ := writeLoopHotUpdateGateControlFixtures(t)
	now := time.Date(2026, 4, 22, 12, 30, 0, 0, time.UTC)

	if err := missioncontrol.StoreLastKnownGoodRuntimePackPointer(root, missioncontrol.LastKnownGoodRuntimePackPointer{
		PackID:            "pack-base",
		Basis:             "holdout_pass",
		VerifiedAt:        now.Add(-30 * time.Second),
		VerifiedBy:        "operator",
		RollbackRecordRef: "bootstrap",
	}); err != nil {
		t.Fatalf("StoreLastKnownGoodRuntimePackPointer() error = %v", err)
	}

	beforeLKGBytes, err := os.ReadFile(missioncontrol.StoreLastKnownGoodRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(last-known-good before) error = %v", err)
	}

	b := chat.NewHub(10)
	prov := &finalResponseProvider{content: "unused"}
	ag := NewAgentLoop(b, prov, prov.GetDefaultModel(), 3, "", nil)
	ag.SetMissionStoreRoot(root)
	if err := ag.ActivateMissionStep(testMissionJob([]string{"read"}, []string{"read"}), "build"); err != nil {
		t.Fatalf("ActivateMissionStep() error = %v", err)
	}

	if _, err := ag.ProcessDirect("HOT_UPDATE_GATE_RECORD job-1 hot-update-1 pack-candidate", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(HOT_UPDATE_GATE_RECORD) error = %v", err)
	}
	if _, err := ag.ProcessDirect("HOT_UPDATE_GATE_PHASE job-1 hot-update-1 validated", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(HOT_UPDATE_GATE_PHASE validated) error = %v", err)
	}
	if _, err := ag.ProcessDirect("HOT_UPDATE_GATE_PHASE job-1 hot-update-1 staged", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(HOT_UPDATE_GATE_PHASE staged) error = %v", err)
	}
	if _, err := ag.ProcessDirect("HOT_UPDATE_GATE_EXECUTE job-1 hot-update-1", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(HOT_UPDATE_GATE_EXECUTE) error = %v", err)
	}

	beforePointerBytes, err := os.ReadFile(missioncontrol.StoreActiveRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(active pointer before reload) error = %v", err)
	}

	resp, err := ag.ProcessDirect("HOT_UPDATE_GATE_RELOAD job-1 hot-update-1", 2*time.Second)
	if err != nil {
		t.Fatalf("ProcessDirect(HOT_UPDATE_GATE_RELOAD first) error = %v", err)
	}
	if resp != "Executed hot-update reload/apply job=job-1 hot_update=hot-update-1." {
		t.Fatalf("ProcessDirect(HOT_UPDATE_GATE_RELOAD first) response = %q, want execute acknowledgement", resp)
	}

	record, err := missioncontrol.LoadHotUpdateGateRecord(root, "hot-update-1")
	if err != nil {
		t.Fatalf("LoadHotUpdateGateRecord() error = %v", err)
	}
	if record.State != missioncontrol.HotUpdateGateStateReloadApplySucceeded {
		t.Fatalf("HotUpdateGateRecord.State = %q, want reload_apply_succeeded", record.State)
	}

	afterPointerBytes, err := os.ReadFile(missioncontrol.StoreActiveRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(active pointer after reload) error = %v", err)
	}
	if string(beforePointerBytes) != string(afterPointerBytes) {
		t.Fatalf("active runtime pack pointer changed on hot-update reload/apply\nbefore:\n%s\nafter:\n%s", string(beforePointerBytes), string(afterPointerBytes))
	}
	afterLKGBytes, err := os.ReadFile(missioncontrol.StoreLastKnownGoodRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(last-known-good after) error = %v", err)
	}
	if string(beforeLKGBytes) != string(afterLKGBytes) {
		t.Fatalf("last-known-good pointer changed on hot-update reload/apply\nbefore:\n%s\nafter:\n%s", string(beforeLKGBytes), string(afterLKGBytes))
	}

	resp, err = ag.ProcessDirect("HOT_UPDATE_GATE_RELOAD job-1 hot-update-1", 2*time.Second)
	if err != nil {
		t.Fatalf("ProcessDirect(HOT_UPDATE_GATE_RELOAD second) error = %v", err)
	}
	if resp != "Selected hot-update reload/apply job=job-1 hot_update=hot-update-1." {
		t.Fatalf("ProcessDirect(HOT_UPDATE_GATE_RELOAD second) response = %q, want idempotent acknowledgement", resp)
	}

	status, err := ag.ProcessDirect("STATUS job-1", 2*time.Second)
	if err != nil {
		t.Fatalf("ProcessDirect(STATUS) error = %v", err)
	}

	var summary missioncontrol.OperatorStatusSummary
	if err := json.Unmarshal([]byte(status), &summary); err != nil {
		t.Fatalf("json.Unmarshal(status) error = %v", err)
	}
	if summary.HotUpdateGateIdentity == nil {
		t.Fatal("HotUpdateGateIdentity = nil, want hot-update gate identity block")
	}
	if len(summary.HotUpdateGateIdentity.Gates) != 1 {
		t.Fatalf("HotUpdateGateIdentity.Gates len = %d, want 1", len(summary.HotUpdateGateIdentity.Gates))
	}
	if summary.HotUpdateGateIdentity.Gates[0].State != "reload_apply_succeeded" {
		t.Fatalf("HotUpdateGateIdentity.Gates[0].State = %q, want reload_apply_succeeded", summary.HotUpdateGateIdentity.Gates[0].State)
	}
}

func TestProcessDirectHotUpdateGateReloadCommandRetriesFromRecoveryNeeded(t *testing.T) {
	t.Parallel()

	root, _ := writeLoopHotUpdateGateControlFixtures(t)
	now := time.Date(2026, 4, 22, 12, 45, 0, 0, time.UTC)

	if err := missioncontrol.StoreLastKnownGoodRuntimePackPointer(root, missioncontrol.LastKnownGoodRuntimePackPointer{
		PackID:            "pack-base",
		Basis:             "holdout_pass",
		VerifiedAt:        now.Add(-30 * time.Second),
		VerifiedBy:        "operator",
		RollbackRecordRef: "bootstrap",
	}); err != nil {
		t.Fatalf("StoreLastKnownGoodRuntimePackPointer() error = %v", err)
	}

	b := chat.NewHub(10)
	prov := &finalResponseProvider{content: "unused"}
	ag := NewAgentLoop(b, prov, prov.GetDefaultModel(), 3, "", nil)
	ag.SetMissionStoreRoot(root)
	if err := ag.ActivateMissionStep(testMissionJob([]string{"read"}, []string{"read"}), "build"); err != nil {
		t.Fatalf("ActivateMissionStep() error = %v", err)
	}

	if _, err := ag.ProcessDirect("HOT_UPDATE_GATE_RECORD job-1 hot-update-1 pack-candidate", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(HOT_UPDATE_GATE_RECORD) error = %v", err)
	}
	if _, err := ag.ProcessDirect("HOT_UPDATE_GATE_PHASE job-1 hot-update-1 validated", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(HOT_UPDATE_GATE_PHASE validated) error = %v", err)
	}
	if _, err := ag.ProcessDirect("HOT_UPDATE_GATE_PHASE job-1 hot-update-1 staged", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(HOT_UPDATE_GATE_PHASE staged) error = %v", err)
	}
	if _, err := ag.ProcessDirect("HOT_UPDATE_GATE_EXECUTE job-1 hot-update-1", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(HOT_UPDATE_GATE_EXECUTE) error = %v", err)
	}

	record, err := missioncontrol.LoadHotUpdateGateRecord(root, "hot-update-1")
	if err != nil {
		t.Fatalf("LoadHotUpdateGateRecord() error = %v", err)
	}
	recoveryAt := record.PreparedAt.Add(8 * time.Minute)
	record.State = missioncontrol.HotUpdateGateStateReloadApplyInProgress
	record.FailureReason = ""
	record.PhaseUpdatedAt = recoveryAt.UTC()
	record.PhaseUpdatedBy = "operator"
	record = missioncontrol.NormalizeHotUpdateGateRecord(record)
	if err := missioncontrol.ValidateHotUpdateGateRecord(record); err != nil {
		t.Fatalf("ValidateHotUpdateGateRecord() error = %v", err)
	}
	if err := missioncontrol.WriteStoreJSONAtomic(missioncontrol.StoreHotUpdateGatePath(root, "hot-update-1"), record); err != nil {
		t.Fatalf("WriteStoreJSONAtomic(reload_apply_in_progress) error = %v", err)
	}
	if _, changed, err := missioncontrol.ReconcileHotUpdateGateRecoveryNeeded(root, "hot-update-1", "operator", recoveryAt.Add(time.Minute)); err != nil {
		t.Fatalf("ReconcileHotUpdateGateRecoveryNeeded() error = %v", err)
	} else if !changed {
		t.Fatal("ReconcileHotUpdateGateRecoveryNeeded() changed = false, want true")
	}

	record, err = missioncontrol.LoadHotUpdateGateRecord(root, "hot-update-1")
	if err != nil {
		t.Fatalf("LoadHotUpdateGateRecord(recovery-needed) error = %v", err)
	}
	record.FailureReason = "stale retry detail"
	record.PhaseUpdatedAt = recoveryAt.Add(2 * time.Minute).UTC()
	record.PhaseUpdatedBy = "operator"
	record = missioncontrol.NormalizeHotUpdateGateRecord(record)
	if err := missioncontrol.ValidateHotUpdateGateRecord(record); err != nil {
		t.Fatalf("ValidateHotUpdateGateRecord(recovery-needed) error = %v", err)
	}
	if err := missioncontrol.WriteStoreJSONAtomic(missioncontrol.StoreHotUpdateGatePath(root, "hot-update-1"), record); err != nil {
		t.Fatalf("WriteStoreJSONAtomic(reload_apply_recovery_needed) error = %v", err)
	}

	beforePointerBytes, err := os.ReadFile(missioncontrol.StoreActiveRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(active pointer before retry) error = %v", err)
	}
	beforeLastKnownGoodBytes, err := os.ReadFile(missioncontrol.StoreLastKnownGoodRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(last known good before retry) error = %v", err)
	}

	resp, err := ag.ProcessDirect("HOT_UPDATE_GATE_RELOAD job-1 hot-update-1", 2*time.Second)
	if err != nil {
		t.Fatalf("ProcessDirect(HOT_UPDATE_GATE_RELOAD retry) error = %v", err)
	}
	if resp != "Executed hot-update reload/apply job=job-1 hot_update=hot-update-1." {
		t.Fatalf("ProcessDirect(HOT_UPDATE_GATE_RELOAD retry) response = %q, want reload/apply acknowledgement", resp)
	}

	record, err = missioncontrol.LoadHotUpdateGateRecord(root, "hot-update-1")
	if err != nil {
		t.Fatalf("LoadHotUpdateGateRecord(retry result) error = %v", err)
	}
	if record.State != missioncontrol.HotUpdateGateStateReloadApplySucceeded {
		t.Fatalf("HotUpdateGateRecord.State = %q, want reload_apply_succeeded", record.State)
	}
	if record.FailureReason != "" {
		t.Fatalf("HotUpdateGateRecord.FailureReason = %q, want empty", record.FailureReason)
	}

	afterPointerBytes, err := os.ReadFile(missioncontrol.StoreActiveRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(active pointer after retry) error = %v", err)
	}
	if string(beforePointerBytes) != string(afterPointerBytes) {
		t.Fatalf("active runtime pack pointer file changed during retry reload/apply\nbefore:\n%s\nafter:\n%s", string(beforePointerBytes), string(afterPointerBytes))
	}
	afterLastKnownGoodBytes, err := os.ReadFile(missioncontrol.StoreLastKnownGoodRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(last known good after retry) error = %v", err)
	}
	if string(beforeLastKnownGoodBytes) != string(afterLastKnownGoodBytes) {
		t.Fatalf("last-known-good pointer file changed during retry reload/apply\nbefore:\n%s\nafter:\n%s", string(beforeLastKnownGoodBytes), string(afterLastKnownGoodBytes))
	}

	gotPointer, err := missioncontrol.LoadActiveRuntimePackPointer(root)
	if err != nil {
		t.Fatalf("LoadActiveRuntimePackPointer() error = %v", err)
	}
	if gotPointer.ReloadGeneration != 3 {
		t.Fatalf("LoadActiveRuntimePackPointer().ReloadGeneration = %d, want 3", gotPointer.ReloadGeneration)
	}

	status, err := ag.ProcessDirect("STATUS job-1", 2*time.Second)
	if err != nil {
		t.Fatalf("ProcessDirect(STATUS) error = %v", err)
	}

	var summary missioncontrol.OperatorStatusSummary
	if err := json.Unmarshal([]byte(status), &summary); err != nil {
		t.Fatalf("json.Unmarshal(status) error = %v", err)
	}
	if summary.HotUpdateGateIdentity == nil {
		t.Fatal("HotUpdateGateIdentity = nil, want hot-update gate identity block")
	}
	if len(summary.HotUpdateGateIdentity.Gates) != 1 {
		t.Fatalf("HotUpdateGateIdentity.Gates len = %d, want 1", len(summary.HotUpdateGateIdentity.Gates))
	}
	if summary.HotUpdateGateIdentity.Gates[0].State != "reload_apply_succeeded" {
		t.Fatalf("HotUpdateGateIdentity.Gates[0].State = %q, want reload_apply_succeeded", summary.HotUpdateGateIdentity.Gates[0].State)
	}
}

func TestProcessDirectHotUpdateGateFailCommandResolvesRecoveryNeededTerminalFailure(t *testing.T) {
	t.Parallel()

	root, _ := writeLoopHotUpdateGateControlFixtures(t)
	now := time.Date(2026, 4, 22, 13, 0, 0, 0, time.UTC)

	if err := missioncontrol.StoreLastKnownGoodRuntimePackPointer(root, missioncontrol.LastKnownGoodRuntimePackPointer{
		PackID:            "pack-base",
		Basis:             "holdout_pass",
		VerifiedAt:        now.Add(-30 * time.Second),
		VerifiedBy:        "operator",
		RollbackRecordRef: "bootstrap",
	}); err != nil {
		t.Fatalf("StoreLastKnownGoodRuntimePackPointer() error = %v", err)
	}

	b := chat.NewHub(10)
	prov := &finalResponseProvider{content: "unused"}
	ag := NewAgentLoop(b, prov, prov.GetDefaultModel(), 3, "", nil)
	ag.SetMissionStoreRoot(root)
	if err := ag.ActivateMissionStep(testMissionJob([]string{"read"}, []string{"read"}), "build"); err != nil {
		t.Fatalf("ActivateMissionStep() error = %v", err)
	}

	if _, err := ag.ProcessDirect("HOT_UPDATE_GATE_RECORD job-1 hot-update-1 pack-candidate", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(HOT_UPDATE_GATE_RECORD) error = %v", err)
	}
	if _, err := ag.ProcessDirect("HOT_UPDATE_GATE_PHASE job-1 hot-update-1 validated", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(HOT_UPDATE_GATE_PHASE validated) error = %v", err)
	}
	if _, err := ag.ProcessDirect("HOT_UPDATE_GATE_PHASE job-1 hot-update-1 staged", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(HOT_UPDATE_GATE_PHASE staged) error = %v", err)
	}
	if _, err := ag.ProcessDirect("HOT_UPDATE_GATE_EXECUTE job-1 hot-update-1", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(HOT_UPDATE_GATE_EXECUTE) error = %v", err)
	}

	record, err := missioncontrol.LoadHotUpdateGateRecord(root, "hot-update-1")
	if err != nil {
		t.Fatalf("LoadHotUpdateGateRecord() error = %v", err)
	}
	recoveryAt := record.PreparedAt.Add(8 * time.Minute)
	record.State = missioncontrol.HotUpdateGateStateReloadApplyInProgress
	record.FailureReason = ""
	record.PhaseUpdatedAt = recoveryAt.UTC()
	record.PhaseUpdatedBy = "operator"
	record = missioncontrol.NormalizeHotUpdateGateRecord(record)
	if err := missioncontrol.ValidateHotUpdateGateRecord(record); err != nil {
		t.Fatalf("ValidateHotUpdateGateRecord() error = %v", err)
	}
	if err := missioncontrol.WriteStoreJSONAtomic(missioncontrol.StoreHotUpdateGatePath(root, "hot-update-1"), record); err != nil {
		t.Fatalf("WriteStoreJSONAtomic(reload_apply_in_progress) error = %v", err)
	}
	if _, changed, err := missioncontrol.ReconcileHotUpdateGateRecoveryNeeded(root, "hot-update-1", "operator", recoveryAt.Add(time.Minute)); err != nil {
		t.Fatalf("ReconcileHotUpdateGateRecoveryNeeded() error = %v", err)
	} else if !changed {
		t.Fatal("ReconcileHotUpdateGateRecoveryNeeded() changed = false, want true")
	}

	beforePointerBytes, err := os.ReadFile(missioncontrol.StoreActiveRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(active pointer before fail) error = %v", err)
	}
	beforeLastKnownGoodBytes, err := os.ReadFile(missioncontrol.StoreLastKnownGoodRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(last known good before fail) error = %v", err)
	}

	resp, err := ag.ProcessDirect("HOT_UPDATE_GATE_FAIL job-1 hot-update-1 operator requested stop after recovery review", 2*time.Second)
	if err != nil {
		t.Fatalf("ProcessDirect(HOT_UPDATE_GATE_FAIL) error = %v", err)
	}
	if resp != "Resolved hot-update terminal failure job=job-1 hot_update=hot-update-1." {
		t.Fatalf("ProcessDirect(HOT_UPDATE_GATE_FAIL) response = %q, want terminal failure acknowledgement", resp)
	}

	record, err = missioncontrol.LoadHotUpdateGateRecord(root, "hot-update-1")
	if err != nil {
		t.Fatalf("LoadHotUpdateGateRecord(result) error = %v", err)
	}
	if record.State != missioncontrol.HotUpdateGateStateReloadApplyFailed {
		t.Fatalf("HotUpdateGateRecord.State = %q, want reload_apply_failed", record.State)
	}
	if record.FailureReason != "operator_terminal_failure: operator requested stop after recovery review" {
		t.Fatalf("HotUpdateGateRecord.FailureReason = %q, want deterministic terminal failure detail", record.FailureReason)
	}

	afterPointerBytes, err := os.ReadFile(missioncontrol.StoreActiveRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(active pointer after fail) error = %v", err)
	}
	if string(beforePointerBytes) != string(afterPointerBytes) {
		t.Fatalf("active runtime pack pointer file changed during terminal failure resolution\nbefore:\n%s\nafter:\n%s", string(beforePointerBytes), string(afterPointerBytes))
	}
	afterLastKnownGoodBytes, err := os.ReadFile(missioncontrol.StoreLastKnownGoodRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(last known good after fail) error = %v", err)
	}
	if string(beforeLastKnownGoodBytes) != string(afterLastKnownGoodBytes) {
		t.Fatalf("last-known-good pointer file changed during terminal failure resolution\nbefore:\n%s\nafter:\n%s", string(beforeLastKnownGoodBytes), string(afterLastKnownGoodBytes))
	}

	gotPointer, err := missioncontrol.LoadActiveRuntimePackPointer(root)
	if err != nil {
		t.Fatalf("LoadActiveRuntimePackPointer() error = %v", err)
	}
	if gotPointer.ReloadGeneration != 3 {
		t.Fatalf("LoadActiveRuntimePackPointer().ReloadGeneration = %d, want 3", gotPointer.ReloadGeneration)
	}

	outcomes, err := missioncontrol.ListHotUpdateOutcomeRecords(root)
	if err != nil && !os.IsNotExist(err) {
		t.Fatalf("ListHotUpdateOutcomeRecords() error = %v", err)
	}
	if len(outcomes) != 0 {
		t.Fatalf("ListHotUpdateOutcomeRecords() len = %d, want 0", len(outcomes))
	}
	promotions, err := missioncontrol.ListPromotionRecords(root)
	if err != nil && !os.IsNotExist(err) {
		t.Fatalf("ListPromotionRecords() error = %v", err)
	}
	if len(promotions) != 0 {
		t.Fatalf("ListPromotionRecords() len = %d, want 0", len(promotions))
	}

	resp, err = ag.ProcessDirect("HOT_UPDATE_GATE_FAIL job-1 hot-update-1 operator requested stop after recovery review", 2*time.Second)
	if err != nil {
		t.Fatalf("ProcessDirect(HOT_UPDATE_GATE_FAIL replay) error = %v", err)
	}
	if resp != "Selected hot-update terminal failure job=job-1 hot_update=hot-update-1." {
		t.Fatalf("ProcessDirect(HOT_UPDATE_GATE_FAIL replay) response = %q, want idempotent acknowledgement", resp)
	}

	resp, err = ag.ProcessDirect("HOT_UPDATE_GATE_FAIL job-1 hot-update-1 different operator reason", 2*time.Second)
	if err == nil {
		t.Fatal("ProcessDirect(HOT_UPDATE_GATE_FAIL different reason) error = nil, want fail-closed rejection")
	}
	if resp != "" {
		t.Fatalf("ProcessDirect(HOT_UPDATE_GATE_FAIL different reason) response = %q, want empty on rejection", resp)
	}
	if !strings.Contains(err.Error(), "already resolved with failure_reason") {
		t.Fatalf("ProcessDirect(HOT_UPDATE_GATE_FAIL different reason) error = %q, want already-resolved rejection", err)
	}

	status, err := ag.ProcessDirect("STATUS job-1", 2*time.Second)
	if err != nil {
		t.Fatalf("ProcessDirect(STATUS) error = %v", err)
	}
	var summary missioncontrol.OperatorStatusSummary
	if err := json.Unmarshal([]byte(status), &summary); err != nil {
		t.Fatalf("json.Unmarshal(status) error = %v", err)
	}
	if summary.HotUpdateGateIdentity == nil {
		t.Fatal("HotUpdateGateIdentity = nil, want hot-update gate identity block")
	}
	if len(summary.HotUpdateGateIdentity.Gates) != 1 {
		t.Fatalf("HotUpdateGateIdentity.Gates len = %d, want 1", len(summary.HotUpdateGateIdentity.Gates))
	}
	if summary.HotUpdateGateIdentity.Gates[0].State != string(missioncontrol.HotUpdateGateStateReloadApplyFailed) {
		t.Fatalf("HotUpdateGateIdentity.Gates[0].State = %q, want reload_apply_failed", summary.HotUpdateGateIdentity.Gates[0].State)
	}
	if summary.HotUpdateGateIdentity.Gates[0].FailureReason != "operator_terminal_failure: operator requested stop after recovery review" {
		t.Fatalf("HotUpdateGateIdentity.Gates[0].FailureReason = %q, want deterministic terminal failure detail", summary.HotUpdateGateIdentity.Gates[0].FailureReason)
	}
	if summary.HotUpdateGateIdentity.Gates[0].PhaseUpdatedAt == nil {
		t.Fatal("HotUpdateGateIdentity.Gates[0].PhaseUpdatedAt = nil, want transition timestamp")
	}
	if summary.HotUpdateGateIdentity.Gates[0].PhaseUpdatedBy != "operator" {
		t.Fatalf("HotUpdateGateIdentity.Gates[0].PhaseUpdatedBy = %q, want operator", summary.HotUpdateGateIdentity.Gates[0].PhaseUpdatedBy)
	}
}

func TestProcessDirectHotUpdateGateFailCommandRequiresReasonAndRejectsInvalidStartingPhase(t *testing.T) {
	t.Parallel()

	root, _ := writeLoopHotUpdateGateControlFixtures(t)

	b := chat.NewHub(10)
	prov := &finalResponseProvider{content: "unused"}
	ag := NewAgentLoop(b, prov, prov.GetDefaultModel(), 3, "", nil)
	ag.SetMissionStoreRoot(root)
	if err := ag.ActivateMissionStep(testMissionJob([]string{"read"}, []string{"read"}), "build"); err != nil {
		t.Fatalf("ActivateMissionStep() error = %v", err)
	}

	if _, err := ag.ProcessDirect("HOT_UPDATE_GATE_RECORD job-1 hot-update-1 pack-candidate", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(HOT_UPDATE_GATE_RECORD) error = %v", err)
	}

	resp, err := ag.ProcessDirect("HOT_UPDATE_GATE_FAIL job-1 hot-update-1", 2*time.Second)
	if err == nil {
		t.Fatal("ProcessDirect(HOT_UPDATE_GATE_FAIL missing reason) error = nil, want required reason rejection")
	}
	if resp != "" {
		t.Fatalf("ProcessDirect(HOT_UPDATE_GATE_FAIL missing reason) response = %q, want empty on rejection", resp)
	}
	if !strings.Contains(err.Error(), "terminal failure reason is required") {
		t.Fatalf("ProcessDirect(HOT_UPDATE_GATE_FAIL missing reason) error = %q, want required reason rejection", err)
	}

	beforePointerBytes, err := os.ReadFile(missioncontrol.StoreActiveRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(active pointer before) error = %v", err)
	}
	resp, err = ag.ProcessDirect("HOT_UPDATE_GATE_FAIL job-1 hot-update-1 operator requested stop after recovery review", 2*time.Second)
	if err == nil {
		t.Fatal("ProcessDirect(HOT_UPDATE_GATE_FAIL) error = nil, want invalid state rejection")
	}
	if resp != "" {
		t.Fatalf("ProcessDirect(HOT_UPDATE_GATE_FAIL) response = %q, want empty on rejection", resp)
	}
	if !strings.Contains(err.Error(), `state "prepared" does not permit terminal failure resolution`) {
		t.Fatalf("ProcessDirect(HOT_UPDATE_GATE_FAIL) error = %q, want invalid state rejection", err)
	}
	afterPointerBytes, err := os.ReadFile(missioncontrol.StoreActiveRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(active pointer after) error = %v", err)
	}
	if string(beforePointerBytes) != string(afterPointerBytes) {
		t.Fatalf("active runtime pack pointer file changed after invalid terminal failure rejection\nbefore:\n%s\nafter:\n%s", string(beforePointerBytes), string(afterPointerBytes))
	}
}

func TestProcessDirectHotUpdateOutcomeCreateCommandCreatesHotUpdatedOutcomeAndIsReplaySafe(t *testing.T) {
	t.Parallel()

	root, _ := writeLoopHotUpdateGateControlFixtures(t)
	writeLoopHotUpdateLastKnownGoodPointer(t, root)

	ag := newLoopHotUpdateOutcomeAgent(t, root)
	prepareLoopHotUpdateSucceededGate(t, ag)

	before := snapshotLoopHotUpdateOutcomeCreateSideEffects(t, root, "hot-update-1")

	resp, err := ag.ProcessDirect("HOT_UPDATE_OUTCOME_CREATE job-1 hot-update-1", 2*time.Second)
	if err != nil {
		t.Fatalf("ProcessDirect(HOT_UPDATE_OUTCOME_CREATE first) error = %v", err)
	}
	if resp != "Created hot-update outcome job=job-1 hot_update=hot-update-1." {
		t.Fatalf("ProcessDirect(HOT_UPDATE_OUTCOME_CREATE first) response = %q, want create acknowledgement", resp)
	}

	outcome, err := missioncontrol.LoadHotUpdateOutcomeRecord(root, "hot-update-outcome-hot-update-1")
	if err != nil {
		t.Fatalf("LoadHotUpdateOutcomeRecord() error = %v", err)
	}
	if outcome.HotUpdateID != "hot-update-1" {
		t.Fatalf("HotUpdateOutcomeRecord.HotUpdateID = %q, want hot-update-1", outcome.HotUpdateID)
	}
	if outcome.CandidatePackID != "pack-candidate" {
		t.Fatalf("HotUpdateOutcomeRecord.CandidatePackID = %q, want pack-candidate", outcome.CandidatePackID)
	}
	if outcome.OutcomeKind != missioncontrol.HotUpdateOutcomeKindHotUpdated {
		t.Fatalf("HotUpdateOutcomeRecord.OutcomeKind = %q, want hot_updated", outcome.OutcomeKind)
	}
	if outcome.Reason != "hot update reload/apply succeeded" {
		t.Fatalf("HotUpdateOutcomeRecord.Reason = %q, want deterministic success reason", outcome.Reason)
	}
	if outcome.CandidateID != "" || outcome.RunID != "" || outcome.CandidateResultID != "" {
		t.Fatalf("optional outcome refs = %q/%q/%q, want empty", outcome.CandidateID, outcome.RunID, outcome.CandidateResultID)
	}

	firstOutcomeBytes, err := os.ReadFile(missioncontrol.StoreHotUpdateOutcomePath(root, outcome.OutcomeID))
	if err != nil {
		t.Fatalf("ReadFile(first outcome) error = %v", err)
	}

	resp, err = ag.ProcessDirect("HOT_UPDATE_OUTCOME_CREATE job-1 hot-update-1", 2*time.Second)
	if err != nil {
		t.Fatalf("ProcessDirect(HOT_UPDATE_OUTCOME_CREATE replay) error = %v", err)
	}
	if resp != "Selected hot-update outcome job=job-1 hot_update=hot-update-1." {
		t.Fatalf("ProcessDirect(HOT_UPDATE_OUTCOME_CREATE replay) response = %q, want idempotent acknowledgement", resp)
	}
	secondOutcomeBytes, err := os.ReadFile(missioncontrol.StoreHotUpdateOutcomePath(root, outcome.OutcomeID))
	if err != nil {
		t.Fatalf("ReadFile(second outcome) error = %v", err)
	}
	if string(firstOutcomeBytes) != string(secondOutcomeBytes) {
		t.Fatalf("hot-update outcome file changed on idempotent replay\nfirst:\n%s\nsecond:\n%s", string(firstOutcomeBytes), string(secondOutcomeBytes))
	}

	assertLoopHotUpdateOutcomeCreateSideEffectsUnchanged(t, root, "hot-update-1", before)
	assertLoopHotUpdateOutcomeCreateNoPromotions(t, root)

	status, err := ag.ProcessDirect("STATUS job-1", 2*time.Second)
	if err != nil {
		t.Fatalf("ProcessDirect(STATUS) error = %v", err)
	}
	var summary missioncontrol.OperatorStatusSummary
	if err := json.Unmarshal([]byte(status), &summary); err != nil {
		t.Fatalf("json.Unmarshal(status) error = %v", err)
	}
	if summary.HotUpdateOutcomeIdentity == nil {
		t.Fatal("HotUpdateOutcomeIdentity = nil, want created outcome in status")
	}
	if summary.HotUpdateOutcomeIdentity.State != "configured" {
		t.Fatalf("HotUpdateOutcomeIdentity.State = %q, want configured", summary.HotUpdateOutcomeIdentity.State)
	}
	if len(summary.HotUpdateOutcomeIdentity.Outcomes) != 1 {
		t.Fatalf("HotUpdateOutcomeIdentity.Outcomes len = %d, want 1", len(summary.HotUpdateOutcomeIdentity.Outcomes))
	}
	statusOutcome := summary.HotUpdateOutcomeIdentity.Outcomes[0]
	if statusOutcome.OutcomeID != "hot-update-outcome-hot-update-1" {
		t.Fatalf("HotUpdateOutcomeIdentity.Outcomes[0].OutcomeID = %q, want deterministic outcome id", statusOutcome.OutcomeID)
	}
	if statusOutcome.OutcomeKind != string(missioncontrol.HotUpdateOutcomeKindHotUpdated) {
		t.Fatalf("HotUpdateOutcomeIdentity.Outcomes[0].OutcomeKind = %q, want hot_updated", statusOutcome.OutcomeKind)
	}
}

func TestProcessDirectHotUpdateOutcomeCreateCommandCreatesFailedOutcomeWithFailureDetail(t *testing.T) {
	t.Parallel()

	root, _ := writeLoopHotUpdateGateControlFixtures(t)
	writeLoopHotUpdateLastKnownGoodPointer(t, root)
	storeLoopHotUpdateTerminalGate(t, root, "hot-update-1", missioncontrol.HotUpdateGateStateReloadApplyFailed, "operator_terminal_failure: operator requested stop after recovery review")

	ag := newLoopHotUpdateOutcomeAgent(t, root)

	resp, err := ag.ProcessDirect("HOT_UPDATE_OUTCOME_CREATE job-1 hot-update-1", 2*time.Second)
	if err != nil {
		t.Fatalf("ProcessDirect(HOT_UPDATE_OUTCOME_CREATE) error = %v", err)
	}
	if resp != "Created hot-update outcome job=job-1 hot_update=hot-update-1." {
		t.Fatalf("ProcessDirect(HOT_UPDATE_OUTCOME_CREATE) response = %q, want create acknowledgement", resp)
	}

	outcome, err := missioncontrol.LoadHotUpdateOutcomeRecord(root, "hot-update-outcome-hot-update-1")
	if err != nil {
		t.Fatalf("LoadHotUpdateOutcomeRecord() error = %v", err)
	}
	if outcome.OutcomeKind != missioncontrol.HotUpdateOutcomeKindFailed {
		t.Fatalf("HotUpdateOutcomeRecord.OutcomeKind = %q, want failed", outcome.OutcomeKind)
	}
	if outcome.Reason != "operator_terminal_failure: operator requested stop after recovery review" {
		t.Fatalf("HotUpdateOutcomeRecord.Reason = %q, want copied failure detail", outcome.Reason)
	}
}

func TestProcessDirectHotUpdateOutcomeCreateCommandRejectsInvalidSourcesWithoutOutcomeRecord(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		setup     func(t *testing.T, root string)
		command   string
		wantError string
	}{
		{
			name:      "missing gate",
			setup:     func(t *testing.T, root string) {},
			command:   "HOT_UPDATE_OUTCOME_CREATE job-1 hot-update-missing",
			wantError: missioncontrol.ErrHotUpdateGateRecordNotFound.Error(),
		},
		{
			name: "non-terminal gate",
			setup: func(t *testing.T, root string) {
				storeLoopHotUpdateTerminalGate(t, root, "hot-update-1", missioncontrol.HotUpdateGateStatePrepared, "")
			},
			command:   "HOT_UPDATE_OUTCOME_CREATE job-1 hot-update-1",
			wantError: `state "prepared" does not permit outcome creation`,
		},
		{
			name: "failed gate missing failure reason",
			setup: func(t *testing.T, root string) {
				storeLoopHotUpdateTerminalGate(t, root, "hot-update-1", missioncontrol.HotUpdateGateStateReloadApplyFailed, " ")
			},
			command:   "HOT_UPDATE_OUTCOME_CREATE job-1 hot-update-1",
			wantError: "failure_reason is required for outcome creation",
		},
		{
			name: "wrong job",
			setup: func(t *testing.T, root string) {
				storeLoopHotUpdateTerminalGate(t, root, "hot-update-1", missioncontrol.HotUpdateGateStateReloadApplySucceeded, "")
			},
			command:   "HOT_UPDATE_OUTCOME_CREATE other-job hot-update-1",
			wantError: "operator command does not match the active job",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			root, _ := writeLoopHotUpdateGateControlFixtures(t)
			writeLoopHotUpdateLastKnownGoodPointer(t, root)
			tc.setup(t, root)
			ag := newLoopHotUpdateOutcomeAgent(t, root)

			resp, err := ag.ProcessDirect(tc.command, 2*time.Second)
			if err == nil {
				t.Fatalf("ProcessDirect(%s) error = nil, want rejection", tc.command)
			}
			if resp != "" {
				t.Fatalf("ProcessDirect(%s) response = %q, want empty on rejection", tc.command, resp)
			}
			if !strings.Contains(err.Error(), tc.wantError) {
				t.Fatalf("ProcessDirect(%s) error = %q, want substring %q", tc.command, err, tc.wantError)
			}
			assertLoopHotUpdateOutcomeCount(t, root, 0)
		})
	}
}

func TestProcessDirectHotUpdateOutcomeCreateCommandRejectsDuplicateOutcomesFailClosed(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		existingID    string
		existingNotes string
		wantError     string
	}{
		{
			name:          "divergent deterministic outcome",
			existingID:    "hot-update-outcome-hot-update-1",
			existingNotes: "manual divergent duplicate",
			wantError:     `mission store hot-update outcome "hot-update-outcome-hot-update-1" already exists`,
		},
		{
			name:          "different outcome for same hot update",
			existingID:    "legacy-outcome",
			existingNotes: "legacy duplicate",
			wantError:     `hot_update_id "hot-update-1" already exists as "legacy-outcome"`,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			root, _ := writeLoopHotUpdateGateControlFixtures(t)
			writeLoopHotUpdateLastKnownGoodPointer(t, root)
			gate := storeLoopHotUpdateTerminalGate(t, root, "hot-update-1", missioncontrol.HotUpdateGateStateReloadApplySucceeded, "")
			if err := missioncontrol.StoreHotUpdateOutcomeRecord(root, missioncontrol.HotUpdateOutcomeRecord{
				OutcomeID:       tc.existingID,
				HotUpdateID:     "hot-update-1",
				CandidatePackID: "pack-candidate",
				OutcomeKind:     missioncontrol.HotUpdateOutcomeKindHotUpdated,
				Reason:          "hot update reload/apply succeeded",
				Notes:           tc.existingNotes,
				OutcomeAt:       gate.PhaseUpdatedAt,
				CreatedAt:       gate.PhaseUpdatedAt.Add(time.Minute),
				CreatedBy:       "operator",
			}); err != nil {
				t.Fatalf("StoreHotUpdateOutcomeRecord(existing) error = %v", err)
			}
			ag := newLoopHotUpdateOutcomeAgent(t, root)

			resp, err := ag.ProcessDirect("HOT_UPDATE_OUTCOME_CREATE job-1 hot-update-1", 2*time.Second)
			if err == nil {
				t.Fatal("ProcessDirect(HOT_UPDATE_OUTCOME_CREATE) error = nil, want duplicate rejection")
			}
			if resp != "" {
				t.Fatalf("ProcessDirect(HOT_UPDATE_OUTCOME_CREATE) response = %q, want empty on rejection", resp)
			}
			if !strings.Contains(err.Error(), tc.wantError) {
				t.Fatalf("ProcessDirect(HOT_UPDATE_OUTCOME_CREATE) error = %q, want substring %q", err, tc.wantError)
			}
			assertLoopHotUpdateOutcomeCount(t, root, 1)
		})
	}
}

func TestProcessDirectHotUpdatePromotionCreateCommandCreatesPromotionAndIsReplaySafe(t *testing.T) {
	t.Parallel()

	root, _ := writeLoopHotUpdateGateControlFixtures(t)
	writeLoopHotUpdateLastKnownGoodPointer(t, root)

	ag := newLoopHotUpdateOutcomeAgent(t, root)
	prepareLoopHotUpdateSucceededGate(t, ag)
	if _, err := ag.ProcessDirect("HOT_UPDATE_OUTCOME_CREATE job-1 hot-update-1", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(HOT_UPDATE_OUTCOME_CREATE) error = %v", err)
	}

	before := snapshotLoopHotUpdatePromotionCreateSideEffects(t, root, "hot-update-1", "hot-update-outcome-hot-update-1")

	resp, err := ag.ProcessDirect("HOT_UPDATE_PROMOTION_CREATE job-1 hot-update-outcome-hot-update-1", 2*time.Second)
	if err != nil {
		t.Fatalf("ProcessDirect(HOT_UPDATE_PROMOTION_CREATE first) error = %v", err)
	}
	if resp != "Created hot-update promotion job=job-1 outcome=hot-update-outcome-hot-update-1." {
		t.Fatalf("ProcessDirect(HOT_UPDATE_PROMOTION_CREATE first) response = %q, want create acknowledgement", resp)
	}
	audits := ag.taskState.AuditEvents()
	if len(audits) == 0 {
		t.Fatal("AuditEvents() count = 0, want promotion create audit event")
	}
	assertAuditEvent(t, audits[len(audits)-1], "job-1", "build", "hot_update_promotion_create", true, "")

	promotion, err := missioncontrol.LoadPromotionRecord(root, "hot-update-promotion-hot-update-1")
	if err != nil {
		t.Fatalf("LoadPromotionRecord() error = %v", err)
	}
	if promotion.PromotedPackID != "pack-candidate" {
		t.Fatalf("PromotionRecord.PromotedPackID = %q, want pack-candidate", promotion.PromotedPackID)
	}
	if promotion.PreviousActivePackID != "pack-base" {
		t.Fatalf("PromotionRecord.PreviousActivePackID = %q, want pack-base", promotion.PreviousActivePackID)
	}
	if promotion.HotUpdateID != "hot-update-1" {
		t.Fatalf("PromotionRecord.HotUpdateID = %q, want hot-update-1", promotion.HotUpdateID)
	}
	if promotion.OutcomeID != "hot-update-outcome-hot-update-1" {
		t.Fatalf("PromotionRecord.OutcomeID = %q, want deterministic outcome id", promotion.OutcomeID)
	}
	if promotion.Reason != "hot update outcome promoted" {
		t.Fatalf("PromotionRecord.Reason = %q, want deterministic promotion reason", promotion.Reason)
	}
	if promotion.LastKnownGoodPackID != "" || promotion.LastKnownGoodBasis != "" {
		t.Fatalf("PromotionRecord LKG fields = %q/%q, want empty", promotion.LastKnownGoodPackID, promotion.LastKnownGoodBasis)
	}

	firstPromotionBytes, err := os.ReadFile(missioncontrol.StorePromotionPath(root, promotion.PromotionID))
	if err != nil {
		t.Fatalf("ReadFile(first promotion) error = %v", err)
	}

	resp, err = ag.ProcessDirect("HOT_UPDATE_PROMOTION_CREATE job-1 hot-update-outcome-hot-update-1", 2*time.Second)
	if err != nil {
		t.Fatalf("ProcessDirect(HOT_UPDATE_PROMOTION_CREATE replay) error = %v", err)
	}
	if resp != "Selected hot-update promotion job=job-1 outcome=hot-update-outcome-hot-update-1." {
		t.Fatalf("ProcessDirect(HOT_UPDATE_PROMOTION_CREATE replay) response = %q, want idempotent acknowledgement", resp)
	}
	secondPromotionBytes, err := os.ReadFile(missioncontrol.StorePromotionPath(root, promotion.PromotionID))
	if err != nil {
		t.Fatalf("ReadFile(second promotion) error = %v", err)
	}
	if string(firstPromotionBytes) != string(secondPromotionBytes) {
		t.Fatalf("hot-update promotion file changed on idempotent replay\nfirst:\n%s\nsecond:\n%s", string(firstPromotionBytes), string(secondPromotionBytes))
	}

	assertLoopHotUpdatePromotionCreateSideEffectsUnchanged(t, root, "hot-update-1", "hot-update-outcome-hot-update-1", before)

	status, err := ag.ProcessDirect("STATUS job-1", 2*time.Second)
	if err != nil {
		t.Fatalf("ProcessDirect(STATUS) error = %v", err)
	}
	var summary missioncontrol.OperatorStatusSummary
	if err := json.Unmarshal([]byte(status), &summary); err != nil {
		t.Fatalf("json.Unmarshal(status) error = %v", err)
	}
	if summary.PromotionIdentity == nil {
		t.Fatal("PromotionIdentity = nil, want created promotion in status")
	}
	if summary.PromotionIdentity.State != "configured" {
		t.Fatalf("PromotionIdentity.State = %q, want configured", summary.PromotionIdentity.State)
	}
	if len(summary.PromotionIdentity.Promotions) != 1 {
		t.Fatalf("PromotionIdentity.Promotions len = %d, want 1", len(summary.PromotionIdentity.Promotions))
	}
	statusPromotion := summary.PromotionIdentity.Promotions[0]
	if statusPromotion.PromotionID != "hot-update-promotion-hot-update-1" {
		t.Fatalf("PromotionIdentity.Promotions[0].PromotionID = %q, want deterministic promotion id", statusPromotion.PromotionID)
	}
	if statusPromotion.OutcomeID != "hot-update-outcome-hot-update-1" {
		t.Fatalf("PromotionIdentity.Promotions[0].OutcomeID = %q, want source outcome id", statusPromotion.OutcomeID)
	}
}

func TestProcessDirectHotUpdatePromotionCreateCommandRejectsInvalidSourcesWithoutPromotionRecord(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		setup     func(t *testing.T, root string) string
		command   func(outcomeID string) string
		wantError string
	}{
		{
			name:      "missing outcome",
			setup:     func(t *testing.T, root string) string { return "missing-outcome" },
			command:   func(outcomeID string) string { return "HOT_UPDATE_PROMOTION_CREATE job-1 " + outcomeID },
			wantError: missioncontrol.ErrHotUpdateOutcomeRecordNotFound.Error(),
		},
		{
			name: "non-hot-updated outcome",
			setup: func(t *testing.T, root string) string {
				return storeLoopHotUpdateOutcomeForPromotion(t, root, missioncontrol.HotUpdateOutcomeKindBlocked, "blocked by policy")
			},
			command:   func(outcomeID string) string { return "HOT_UPDATE_PROMOTION_CREATE job-1 " + outcomeID },
			wantError: "does not permit promotion creation",
		},
		{
			name: "failed outcome",
			setup: func(t *testing.T, root string) string {
				return storeLoopHotUpdateOutcomeForPromotion(t, root, missioncontrol.HotUpdateOutcomeKindFailed, "operator_terminal_failure: recovery reviewed")
			},
			command:   func(outcomeID string) string { return "HOT_UPDATE_PROMOTION_CREATE job-1 " + outcomeID },
			wantError: "does not permit promotion creation",
		},
		{
			name: "missing originating gate",
			setup: func(t *testing.T, root string) string {
				writeLoopHotUpdateOutcomeRaw(t, root, missioncontrol.HotUpdateOutcomeRecord{
					RecordVersion:   1,
					OutcomeID:       "hot-update-outcome-missing-gate",
					HotUpdateID:     "missing-gate",
					CandidatePackID: "pack-candidate",
					OutcomeKind:     missioncontrol.HotUpdateOutcomeKindHotUpdated,
					Reason:          "hot update reload/apply succeeded",
					OutcomeAt:       time.Date(2026, 4, 22, 12, 2, 0, 0, time.UTC),
					CreatedAt:       time.Date(2026, 4, 22, 12, 3, 0, 0, time.UTC),
					CreatedBy:       "operator",
				})
				return "hot-update-outcome-missing-gate"
			},
			command:   func(outcomeID string) string { return "HOT_UPDATE_PROMOTION_CREATE job-1 " + outcomeID },
			wantError: missioncontrol.ErrHotUpdateGateRecordNotFound.Error(),
		},
		{
			name: "invalid outcome gate linkage",
			setup: func(t *testing.T, root string) string {
				outcomeID := storeLoopHotUpdateOutcomeForPromotion(t, root, missioncontrol.HotUpdateOutcomeKindHotUpdated, "hot update reload/apply succeeded")
				mustStoreLoopRuntimePack(t, root, "pack-other")
				outcome, err := missioncontrol.LoadHotUpdateOutcomeRecord(root, outcomeID)
				if err != nil {
					t.Fatalf("LoadHotUpdateOutcomeRecord() error = %v", err)
				}
				outcome.CandidatePackID = "pack-other"
				writeLoopHotUpdateOutcomeRaw(t, root, outcome)
				return outcomeID
			},
			command:   func(outcomeID string) string { return "HOT_UPDATE_PROMOTION_CREATE job-1 " + outcomeID },
			wantError: `candidate_pack_id "pack-other" does not match hot-update gate candidate_pack_id "pack-candidate"`,
		},
		{
			name: "empty candidate pack",
			setup: func(t *testing.T, root string) string {
				return storeLoopHotUpdateOutcomeForPromotionWithMutation(t, root, func(record *missioncontrol.HotUpdateOutcomeRecord) {
					record.CandidatePackID = ""
				})
			},
			command:   func(outcomeID string) string { return "HOT_UPDATE_PROMOTION_CREATE job-1 " + outcomeID },
			wantError: "candidate_pack_id is required for promotion creation",
		},
		{
			name: "unresolved previous active pack",
			setup: func(t *testing.T, root string) string {
				outcomeID := storeLoopHotUpdateOutcomeForPromotion(t, root, missioncontrol.HotUpdateOutcomeKindHotUpdated, "hot update reload/apply succeeded")
				gate, err := missioncontrol.LoadHotUpdateGateRecord(root, "hot-update-1")
				if err != nil {
					t.Fatalf("LoadHotUpdateGateRecord() error = %v", err)
				}
				gate.PreviousActivePackID = "pack-missing"
				writeLoopHotUpdateGateRaw(t, root, gate)
				return outcomeID
			},
			command:   func(outcomeID string) string { return "HOT_UPDATE_PROMOTION_CREATE job-1 " + outcomeID },
			wantError: `previous_active_pack_id "pack-missing"`,
		},
		{
			name: "wrong job",
			setup: func(t *testing.T, root string) string {
				return storeLoopHotUpdateOutcomeForPromotion(t, root, missioncontrol.HotUpdateOutcomeKindHotUpdated, "hot update reload/apply succeeded")
			},
			command:   func(outcomeID string) string { return "HOT_UPDATE_PROMOTION_CREATE other-job " + outcomeID },
			wantError: "operator command does not match the active job",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			root, _ := writeLoopHotUpdateGateControlFixtures(t)
			writeLoopHotUpdateLastKnownGoodPointer(t, root)
			outcomeID := tc.setup(t, root)
			ag := newLoopHotUpdateOutcomeAgent(t, root)

			resp, err := ag.ProcessDirect(tc.command(outcomeID), 2*time.Second)
			if err == nil {
				t.Fatalf("ProcessDirect(%s) error = nil, want rejection", tc.command(outcomeID))
			}
			if resp != "" {
				t.Fatalf("ProcessDirect(%s) response = %q, want empty on rejection", tc.command(outcomeID), resp)
			}
			if !strings.Contains(err.Error(), tc.wantError) {
				t.Fatalf("ProcessDirect(%s) error = %q, want substring %q", tc.command(outcomeID), err, tc.wantError)
			}
			assertLoopHotUpdatePromotionCount(t, root, 0)
		})
	}
}

func TestProcessDirectHotUpdatePromotionCreateCommandRejectsDuplicatePromotionsFailClosed(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		existingID string
		mutate     func(*missioncontrol.PromotionRecord)
		wantError  string
	}{
		{
			name:       "divergent deterministic promotion",
			existingID: "hot-update-promotion-hot-update-1",
			mutate: func(record *missioncontrol.PromotionRecord) {
				record.Notes = "manual divergent duplicate"
			},
			wantError: `mission store promotion "hot-update-promotion-hot-update-1" already exists`,
		},
		{
			name:       "different promotion for same hot update",
			existingID: "legacy-promotion",
			mutate: func(record *missioncontrol.PromotionRecord) {
				record.OutcomeID = ""
			},
			wantError: `hot_update_id "hot-update-1" already exists as "legacy-promotion"`,
		},
		{
			name:       "different promotion for same outcome",
			existingID: "legacy-promotion",
			mutate:     func(record *missioncontrol.PromotionRecord) {},
			wantError:  `outcome_id "hot-update-outcome-hot-update-1" already exists as "legacy-promotion"`,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			root, _ := writeLoopHotUpdateGateControlFixtures(t)
			writeLoopHotUpdateLastKnownGoodPointer(t, root)
			outcomeID := storeLoopHotUpdateOutcomeForPromotion(t, root, missioncontrol.HotUpdateOutcomeKindHotUpdated, "hot update reload/apply succeeded")
			outcome, err := missioncontrol.LoadHotUpdateOutcomeRecord(root, outcomeID)
			if err != nil {
				t.Fatalf("LoadHotUpdateOutcomeRecord() error = %v", err)
			}
			record := missioncontrol.PromotionRecord{
				PromotionID:          tc.existingID,
				PromotedPackID:       "pack-candidate",
				PreviousActivePackID: "pack-base",
				HotUpdateID:          "hot-update-1",
				OutcomeID:            outcomeID,
				Reason:               "hot update outcome promoted",
				PromotedAt:           outcome.OutcomeAt,
				CreatedAt:            outcome.OutcomeAt.Add(time.Minute),
				CreatedBy:            "operator",
			}
			tc.mutate(&record)
			if err := missioncontrol.StorePromotionRecord(root, record); err != nil {
				t.Fatalf("StorePromotionRecord(existing) error = %v", err)
			}
			ag := newLoopHotUpdateOutcomeAgent(t, root)

			resp, err := ag.ProcessDirect("HOT_UPDATE_PROMOTION_CREATE job-1 "+outcomeID, 2*time.Second)
			if err == nil {
				t.Fatal("ProcessDirect(HOT_UPDATE_PROMOTION_CREATE) error = nil, want duplicate rejection")
			}
			if resp != "" {
				t.Fatalf("ProcessDirect(HOT_UPDATE_PROMOTION_CREATE) response = %q, want empty on rejection", resp)
			}
			if !strings.Contains(err.Error(), tc.wantError) {
				t.Fatalf("ProcessDirect(HOT_UPDATE_PROMOTION_CREATE) error = %q, want substring %q", err, tc.wantError)
			}
			assertLoopHotUpdatePromotionCount(t, root, 1)
		})
	}
}

func TestProcessDirectHotUpdateLKGRecertifyCommandRecertifiesFromPromotionAndIsReplaySafe(t *testing.T) {
	t.Parallel()

	root, _ := writeLoopHotUpdateGateControlFixtures(t)
	writeLoopHotUpdateLastKnownGoodPointer(t, root)

	ag := newLoopHotUpdateOutcomeAgent(t, root)
	prepareLoopHotUpdateSucceededGate(t, ag)
	if _, err := ag.ProcessDirect("HOT_UPDATE_OUTCOME_CREATE job-1 hot-update-1", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(HOT_UPDATE_OUTCOME_CREATE) error = %v", err)
	}
	if _, err := ag.ProcessDirect("HOT_UPDATE_PROMOTION_CREATE job-1 hot-update-outcome-hot-update-1", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(HOT_UPDATE_PROMOTION_CREATE) error = %v", err)
	}

	before := snapshotLoopHotUpdateLKGRecertifySideEffects(t, root, "hot-update-1", "hot-update-outcome-hot-update-1", "hot-update-promotion-hot-update-1")

	resp, err := ag.ProcessDirect("HOT_UPDATE_LKG_RECERTIFY job-1 hot-update-promotion-hot-update-1", 2*time.Second)
	if err != nil {
		t.Fatalf("ProcessDirect(HOT_UPDATE_LKG_RECERTIFY first) error = %v", err)
	}
	if resp != "Recertified hot-update last-known-good job=job-1 promotion=hot-update-promotion-hot-update-1." {
		t.Fatalf("ProcessDirect(HOT_UPDATE_LKG_RECERTIFY first) response = %q, want recertify acknowledgement", resp)
	}
	audits := ag.taskState.AuditEvents()
	if len(audits) == 0 {
		t.Fatal("AuditEvents() count = 0, want lkg recertify audit event")
	}
	assertAuditEvent(t, audits[len(audits)-1], "job-1", "build", "hot_update_lkg_recertify", true, "")

	pointer, err := missioncontrol.LoadLastKnownGoodRuntimePackPointer(root)
	if err != nil {
		t.Fatalf("LoadLastKnownGoodRuntimePackPointer() error = %v", err)
	}
	if pointer.PackID != "pack-candidate" {
		t.Fatalf("LastKnownGoodRuntimePackPointer.PackID = %q, want pack-candidate", pointer.PackID)
	}
	if pointer.Basis != "hot_update_promotion:hot-update-promotion-hot-update-1" {
		t.Fatalf("LastKnownGoodRuntimePackPointer.Basis = %q, want deterministic hot-update promotion basis", pointer.Basis)
	}
	if pointer.VerifiedBy != "operator" {
		t.Fatalf("LastKnownGoodRuntimePackPointer.VerifiedBy = %q, want operator", pointer.VerifiedBy)
	}
	if pointer.RollbackRecordRef != "hot_update_promotion:hot-update-promotion-hot-update-1" {
		t.Fatalf("LastKnownGoodRuntimePackPointer.RollbackRecordRef = %q, want deterministic hot-update promotion ref", pointer.RollbackRecordRef)
	}
	firstLKGBytes, err := os.ReadFile(missioncontrol.StoreLastKnownGoodRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(first last-known-good pointer) error = %v", err)
	}
	if string(firstLKGBytes) == string(before.lastKnownGoodBytes) {
		t.Fatal("last-known-good pointer did not change on first recertification")
	}
	assertLoopHotUpdateLKGRecertifySideEffectsUnchangedExceptLKG(t, root, "hot-update-1", "hot-update-outcome-hot-update-1", "hot-update-promotion-hot-update-1", before)

	beforeReplay := snapshotLoopHotUpdateLKGRecertifySideEffects(t, root, "hot-update-1", "hot-update-outcome-hot-update-1", "hot-update-promotion-hot-update-1")
	resp, err = ag.ProcessDirect("HOT_UPDATE_LKG_RECERTIFY job-1 hot-update-promotion-hot-update-1", 2*time.Second)
	if err != nil {
		t.Fatalf("ProcessDirect(HOT_UPDATE_LKG_RECERTIFY replay) error = %v", err)
	}
	if resp != "Selected hot-update last-known-good job=job-1 promotion=hot-update-promotion-hot-update-1." {
		t.Fatalf("ProcessDirect(HOT_UPDATE_LKG_RECERTIFY replay) response = %q, want selected acknowledgement", resp)
	}
	secondLKGBytes, err := os.ReadFile(missioncontrol.StoreLastKnownGoodRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(second last-known-good pointer) error = %v", err)
	}
	if string(firstLKGBytes) != string(secondLKGBytes) {
		t.Fatalf("last-known-good pointer changed on idempotent replay\nfirst:\n%s\nsecond:\n%s", string(firstLKGBytes), string(secondLKGBytes))
	}
	assertLoopHotUpdateLKGRecertifySideEffectsFullyUnchanged(t, root, "hot-update-1", "hot-update-outcome-hot-update-1", "hot-update-promotion-hot-update-1", beforeReplay)

	status, err := ag.ProcessDirect("STATUS job-1", 2*time.Second)
	if err != nil {
		t.Fatalf("ProcessDirect(STATUS) error = %v", err)
	}
	var summary missioncontrol.OperatorStatusSummary
	if err := json.Unmarshal([]byte(status), &summary); err != nil {
		t.Fatalf("json.Unmarshal(status) error = %v", err)
	}
	if summary.RuntimePackIdentity == nil {
		t.Fatal("RuntimePackIdentity = nil, want recertified LKG in status")
	}
	if summary.RuntimePackIdentity.LastKnownGood.PackID != "pack-candidate" {
		t.Fatalf("RuntimePackIdentity.LastKnownGood.PackID = %q, want pack-candidate", summary.RuntimePackIdentity.LastKnownGood.PackID)
	}
	if summary.RuntimePackIdentity.LastKnownGood.Basis != "hot_update_promotion:hot-update-promotion-hot-update-1" {
		t.Fatalf("RuntimePackIdentity.LastKnownGood.Basis = %q, want hot-update promotion basis", summary.RuntimePackIdentity.LastKnownGood.Basis)
	}
	if summary.RuntimePackIdentity.LastKnownGood.VerifiedAt == nil {
		t.Fatal("RuntimePackIdentity.LastKnownGood.VerifiedAt = nil, want verification timestamp")
	}
}

func TestProcessDirectHotUpdateLKGRecertifyCommandRejectsInvalidSourcesWithoutLKGMutation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		setup     func(t *testing.T, root string) string
		command   func(promotionID string) string
		wantError string
	}{
		{
			name:      "missing promotion",
			setup:     func(t *testing.T, root string) string { return "missing-promotion" },
			command:   func(promotionID string) string { return "HOT_UPDATE_LKG_RECERTIFY job-1 " + promotionID },
			wantError: missioncontrol.ErrPromotionRecordNotFound.Error(),
		},
		{
			name: "promotion without outcome id",
			setup: func(t *testing.T, root string) string {
				return storeLoopHotUpdatePromotionForLKGRecertifyWithMutation(t, root, nil, func(record *missioncontrol.PromotionRecord) {
					record.PromotionID = "promotion-no-outcome"
					record.OutcomeID = ""
				})
			},
			command:   func(promotionID string) string { return "HOT_UPDATE_LKG_RECERTIFY job-1 " + promotionID },
			wantError: `promotion "promotion-no-outcome" outcome_id is required`,
		},
		{
			name: "linked outcome missing",
			setup: func(t *testing.T, root string) string {
				storeLoopHotUpdateTerminalGate(t, root, "hot-update-1", missioncontrol.HotUpdateGateStateReloadApplySucceeded, "")
				writeLoopHotUpdatePromotionRaw(t, root, missioncontrol.PromotionRecord{
					RecordVersion:        1,
					PromotionID:          "promotion-missing-outcome",
					PromotedPackID:       "pack-candidate",
					PreviousActivePackID: "pack-base",
					HotUpdateID:          "hot-update-1",
					OutcomeID:            "missing-outcome",
					Reason:               "hot update outcome promoted",
					PromotedAt:           time.Date(2026, 4, 22, 12, 2, 0, 0, time.UTC),
					CreatedAt:            time.Date(2026, 4, 22, 12, 3, 0, 0, time.UTC),
					CreatedBy:            "operator",
				})
				return "promotion-missing-outcome"
			},
			command:   func(promotionID string) string { return "HOT_UPDATE_LKG_RECERTIFY job-1 " + promotionID },
			wantError: missioncontrol.ErrHotUpdateOutcomeRecordNotFound.Error(),
		},
		{
			name: "linked outcome not hot updated",
			setup: func(t *testing.T, root string) string {
				return storeLoopHotUpdatePromotionForLKGRecertifyWithMutation(t, root, func(record *missioncontrol.HotUpdateOutcomeRecord) {
					record.OutcomeKind = missioncontrol.HotUpdateOutcomeKindFailed
					record.Reason = "operator_terminal_failure: recovery reviewed"
				}, nil)
			},
			command:   func(promotionID string) string { return "HOT_UPDATE_LKG_RECERTIFY job-1 " + promotionID },
			wantError: `outcome_kind "failed" does not permit last-known-good recertification`,
		},
		{
			name: "active pointer missing",
			setup: func(t *testing.T, root string) string {
				promotionID := storeLoopHotUpdatePromotionForLKGRecertifyWithMutation(t, root, nil, nil)
				if err := os.Remove(missioncontrol.StoreActiveRuntimePackPointerPath(root)); err != nil {
					t.Fatalf("Remove(active pointer) error = %v", err)
				}
				return promotionID
			},
			command:   func(promotionID string) string { return "HOT_UPDATE_LKG_RECERTIFY job-1 " + promotionID },
			wantError: missioncontrol.ErrActiveRuntimePackPointerNotFound.Error(),
		},
		{
			name: "active pointer mismatch",
			setup: func(t *testing.T, root string) string {
				promotionID := storeLoopHotUpdatePromotionForLKGRecertifyWithMutation(t, root, nil, nil)
				if err := missioncontrol.StoreActiveRuntimePackPointer(root, missioncontrol.ActiveRuntimePackPointer{
					ActivePackID:        "pack-base",
					LastKnownGoodPackID: "pack-base",
					UpdatedAt:           time.Date(2026, 4, 22, 12, 4, 0, 0, time.UTC),
					UpdatedBy:           "operator",
					UpdateRecordRef:     "manual-active-mismatch",
					ReloadGeneration:    7,
				}); err != nil {
					t.Fatalf("StoreActiveRuntimePackPointer(mismatch) error = %v", err)
				}
				return promotionID
			},
			command:   func(promotionID string) string { return "HOT_UPDATE_LKG_RECERTIFY job-1 " + promotionID },
			wantError: `active_pack_id "pack-base" does not match promotion promoted_pack_id "pack-candidate"`,
		},
		{
			name: "current lkg missing",
			setup: func(t *testing.T, root string) string {
				promotionID := storeLoopHotUpdatePromotionForLKGRecertifyWithMutation(t, root, nil, nil)
				if err := os.Remove(missioncontrol.StoreLastKnownGoodRuntimePackPointerPath(root)); err != nil {
					t.Fatalf("Remove(last-known-good pointer) error = %v", err)
				}
				return promotionID
			},
			command:   func(promotionID string) string { return "HOT_UPDATE_LKG_RECERTIFY job-1 " + promotionID },
			wantError: missioncontrol.ErrLastKnownGoodRuntimePackPointerNotFound.Error(),
		},
		{
			name: "current lkg not previous or promoted pack",
			setup: func(t *testing.T, root string) string {
				promotionID := storeLoopHotUpdatePromotionForLKGRecertifyWithMutation(t, root, nil, nil)
				mustStoreLoopRuntimePack(t, root, "pack-other")
				if err := missioncontrol.StoreLastKnownGoodRuntimePackPointer(root, missioncontrol.LastKnownGoodRuntimePackPointer{
					PackID:            "pack-other",
					Basis:             "external_basis",
					VerifiedAt:        time.Date(2026, 4, 22, 12, 5, 0, 0, time.UTC),
					VerifiedBy:        "operator",
					RollbackRecordRef: "external",
				}); err != nil {
					t.Fatalf("StoreLastKnownGoodRuntimePackPointer(pack-other) error = %v", err)
				}
				return promotionID
			},
			command:   func(promotionID string) string { return "HOT_UPDATE_LKG_RECERTIFY job-1 " + promotionID },
			wantError: `pack_id "pack-other" does not match promotion previous_active_pack_id "pack-base"`,
		},
		{
			name: "divergent existing lkg",
			setup: func(t *testing.T, root string) string {
				promotionID := storeLoopHotUpdatePromotionForLKGRecertifyWithMutation(t, root, nil, nil)
				if err := missioncontrol.StoreLastKnownGoodRuntimePackPointer(root, missioncontrol.LastKnownGoodRuntimePackPointer{
					PackID:            "pack-candidate",
					Basis:             "manual_hot_update_promotion",
					VerifiedAt:        time.Date(2026, 4, 22, 12, 5, 0, 0, time.UTC),
					VerifiedBy:        "operator",
					RollbackRecordRef: "manual",
				}); err != nil {
					t.Fatalf("StoreLastKnownGoodRuntimePackPointer(divergent) error = %v", err)
				}
				return promotionID
			},
			command:   func(promotionID string) string { return "HOT_UPDATE_LKG_RECERTIFY job-1 " + promotionID },
			wantError: `already points to promoted pack but differs from deterministic recertification`,
		},
		{
			name: "wrong job",
			setup: func(t *testing.T, root string) string {
				return storeLoopHotUpdatePromotionForLKGRecertifyWithMutation(t, root, nil, nil)
			},
			command:   func(promotionID string) string { return "HOT_UPDATE_LKG_RECERTIFY other-job " + promotionID },
			wantError: "operator command does not match the active job",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			root, _ := writeLoopHotUpdateGateControlFixtures(t)
			writeLoopHotUpdateLastKnownGoodPointer(t, root)
			promotionID := tc.setup(t, root)
			beforeLKGBytes, beforeLKGFound := readLoopOptionalFile(t, missioncontrol.StoreLastKnownGoodRuntimePackPointerPath(root))
			ag := newLoopHotUpdateOutcomeAgent(t, root)

			resp, err := ag.ProcessDirect(tc.command(promotionID), 2*time.Second)
			if err == nil {
				t.Fatalf("ProcessDirect(%s) error = nil, want rejection", tc.command(promotionID))
			}
			if resp != "" {
				t.Fatalf("ProcessDirect(%s) response = %q, want empty on rejection", tc.command(promotionID), resp)
			}
			if !strings.Contains(err.Error(), tc.wantError) {
				t.Fatalf("ProcessDirect(%s) error = %q, want substring %q", tc.command(promotionID), err, tc.wantError)
			}
			afterLKGBytes, afterLKGFound := readLoopOptionalFile(t, missioncontrol.StoreLastKnownGoodRuntimePackPointerPath(root))
			if beforeLKGFound != afterLKGFound {
				t.Fatalf("last-known-good pointer existence changed from %t to %t", beforeLKGFound, afterLKGFound)
			}
			if string(beforeLKGBytes) != string(afterLKGBytes) {
				t.Fatalf("last-known-good pointer changed on rejected recertify\nbefore:\n%s\nafter:\n%s", string(beforeLKGBytes), string(afterLKGBytes))
			}
		})
	}
}

func TestProcessDirectHotUpdateGateRecordCommandFailsClosedWhenRollbackTargetLinkageIsMissing(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 22, 13, 0, 0, 0, time.UTC)
	if err := missioncontrol.StoreRuntimePackRecord(root, missioncontrol.RuntimePackRecord{
		PackID:                   "pack-base",
		CreatedAt:                now.Add(-2 * time.Minute),
		Channel:                  "phone",
		PromptPackRef:            "prompt-pack-base",
		SkillPackRef:             "skill-pack-base",
		ManifestRef:              "manifest-base",
		ExtensionPackRef:         "extension-base",
		PolicyRef:                "policy-base",
		SourceSummary:            "baseline pack",
		MutableSurfaces:          []string{"skills"},
		ImmutableSurfaces:        []string{"policy", "authority"},
		SurfaceClasses:           []string{"class_1"},
		CompatibilityContractRef: "compat-v1",
	}); err != nil {
		t.Fatalf("StoreRuntimePackRecord(pack-base) error = %v", err)
	}
	if err := missioncontrol.StoreRuntimePackRecord(root, missioncontrol.RuntimePackRecord{
		PackID:                   "pack-candidate-bad",
		ParentPackID:             "pack-base",
		CreatedAt:                now.Add(-time.Minute),
		Channel:                  "phone",
		PromptPackRef:            "prompt-pack-candidate",
		SkillPackRef:             "skill-pack-candidate",
		ManifestRef:              "manifest-candidate",
		ExtensionPackRef:         "extension-candidate",
		PolicyRef:                "policy-candidate",
		SourceSummary:            "candidate pack",
		MutableSurfaces:          []string{"skills"},
		ImmutableSurfaces:        []string{"policy", "authority"},
		SurfaceClasses:           []string{"class_1"},
		CompatibilityContractRef: "compat-v1",
	}); err != nil {
		t.Fatalf("StoreRuntimePackRecord(pack-candidate-bad) error = %v", err)
	}
	wantPointer := missioncontrol.ActiveRuntimePackPointer{
		ActivePackID:        "pack-base",
		LastKnownGoodPackID: "pack-base",
		UpdatedAt:           now,
		UpdatedBy:           "operator",
		UpdateRecordRef:     "bootstrap",
		ReloadGeneration:    2,
	}
	if err := missioncontrol.StoreActiveRuntimePackPointer(root, wantPointer); err != nil {
		t.Fatalf("StoreActiveRuntimePackPointer() error = %v", err)
	}
	wantPointer.RecordVersion = 1
	beforePointerBytes, err := os.ReadFile(missioncontrol.StoreActiveRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(active pointer before) error = %v", err)
	}

	b := chat.NewHub(10)
	prov := &finalResponseProvider{content: "unused"}
	ag := NewAgentLoop(b, prov, prov.GetDefaultModel(), 3, "", nil)
	ag.SetMissionStoreRoot(root)
	if err := ag.ActivateMissionStep(testMissionJob([]string{"read"}, []string{"read"}), "build"); err != nil {
		t.Fatalf("ActivateMissionStep() error = %v", err)
	}

	resp, err := ag.ProcessDirect("HOT_UPDATE_GATE_RECORD job-1 hot-update-bad pack-candidate-bad", 2*time.Second)
	if err == nil {
		t.Fatal("ProcessDirect(HOT_UPDATE_GATE_RECORD) error = nil, want missing rollback target rejection")
	}
	if resp != "" {
		t.Fatalf("ProcessDirect(HOT_UPDATE_GATE_RECORD) response = %q, want empty on rejection", resp)
	}
	if !strings.Contains(err.Error(), "rollback_target_pack_id is required") {
		t.Fatalf("ProcessDirect(HOT_UPDATE_GATE_RECORD) error = %q, want missing rollback_target rejection", err)
	}
	if _, err := missioncontrol.LoadHotUpdateGateRecord(root, "hot-update-bad"); err != missioncontrol.ErrHotUpdateGateRecordNotFound {
		t.Fatalf("LoadHotUpdateGateRecord() error = %v, want %v", err, missioncontrol.ErrHotUpdateGateRecordNotFound)
	}
	gotPointer, err := missioncontrol.LoadActiveRuntimePackPointer(root)
	if err != nil {
		t.Fatalf("LoadActiveRuntimePackPointer() error = %v", err)
	}
	if gotPointer != wantPointer {
		t.Fatalf("LoadActiveRuntimePackPointer() = %#v, want %#v", gotPointer, wantPointer)
	}
	afterPointerBytes, err := os.ReadFile(missioncontrol.StoreActiveRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(active pointer after) error = %v", err)
	}
	if string(beforePointerBytes) != string(afterPointerBytes) {
		t.Fatalf("active pointer changed on rejected hot-update gate path\nbefore:\n%s\nafter:\n%s", string(beforePointerBytes), string(afterPointerBytes))
	}
}

func TestProcessDirectRollbackApplyRecordCommandCreatesOrSelectsWorkflowAndPreservesActiveRuntimePackPointer(t *testing.T) {
	t.Parallel()

	root, wantPointer := writeLoopRollbackPromotionFixtures(t)

	b := chat.NewHub(10)
	prov := &finalResponseProvider{content: "unused"}
	ag := NewAgentLoop(b, prov, prov.GetDefaultModel(), 3, "", nil)
	ag.SetMissionStoreRoot(root)
	if err := ag.ActivateMissionStep(testMissionJob([]string{"read"}, []string{"read"}), "build"); err != nil {
		t.Fatalf("ActivateMissionStep() error = %v", err)
	}

	ecBefore, ok := ag.ActiveMissionStep()
	if !ok || ecBefore.Runtime == nil {
		t.Fatalf("ActiveMissionStep() before = %#v, want active runtime", ecBefore)
	}

	if _, err := ag.ProcessDirect("ROLLBACK_RECORD job-1 promotion-1 rollback-1", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(ROLLBACK_RECORD) error = %v", err)
	}

	resp, err := ag.ProcessDirect("ROLLBACK_APPLY_RECORD job-1 rollback-1 apply-1", 2*time.Second)
	if err != nil {
		t.Fatalf("ProcessDirect(ROLLBACK_APPLY_RECORD first) error = %v", err)
	}
	if resp != "Recorded rollback-apply workflow job=job-1 rollback=rollback-1 apply=apply-1." {
		t.Fatalf("ProcessDirect(ROLLBACK_APPLY_RECORD first) response = %q, want create acknowledgement", resp)
	}

	record, err := missioncontrol.LoadRollbackApplyRecord(root, "apply-1")
	if err != nil {
		t.Fatalf("LoadRollbackApplyRecord() error = %v", err)
	}
	if record.RollbackID != "rollback-1" {
		t.Fatalf("RollbackApplyRecord.RollbackID = %q, want rollback-1", record.RollbackID)
	}
	if record.Phase != missioncontrol.RollbackApplyPhaseRecorded {
		t.Fatalf("RollbackApplyRecord.Phase = %q, want recorded", record.Phase)
	}
	if record.ActivationState != missioncontrol.RollbackApplyActivationStateUnchanged {
		t.Fatalf("RollbackApplyRecord.ActivationState = %q, want unchanged", record.ActivationState)
	}
	if record.CreatedBy != "operator" {
		t.Fatalf("RollbackApplyRecord.CreatedBy = %q, want operator", record.CreatedBy)
	}
	if !record.RequestedAt.Equal(record.CreatedAt) {
		t.Fatalf("RollbackApplyRecord timestamps = (%v, %v), want equal requested_at and created_at", record.RequestedAt, record.CreatedAt)
	}

	firstBytes, err := os.ReadFile(missioncontrol.StoreRollbackApplyPath(root, "apply-1"))
	if err != nil {
		t.Fatalf("ReadFile(first apply) error = %v", err)
	}

	resp, err = ag.ProcessDirect("ROLLBACK_APPLY_RECORD job-1 rollback-1 apply-1", 2*time.Second)
	if err != nil {
		t.Fatalf("ProcessDirect(ROLLBACK_APPLY_RECORD second) error = %v", err)
	}
	if resp != "Selected rollback-apply workflow job=job-1 rollback=rollback-1 apply=apply-1." {
		t.Fatalf("ProcessDirect(ROLLBACK_APPLY_RECORD second) response = %q, want select acknowledgement", resp)
	}

	secondBytes, err := os.ReadFile(missioncontrol.StoreRollbackApplyPath(root, "apply-1"))
	if err != nil {
		t.Fatalf("ReadFile(second apply) error = %v", err)
	}
	if string(firstBytes) != string(secondBytes) {
		t.Fatalf("rollback-apply file changed on select path\nfirst:\n%s\nsecond:\n%s", string(firstBytes), string(secondBytes))
	}

	gotPointer, err := missioncontrol.LoadActiveRuntimePackPointer(root)
	if err != nil {
		t.Fatalf("LoadActiveRuntimePackPointer() error = %v", err)
	}
	if gotPointer != wantPointer {
		t.Fatalf("LoadActiveRuntimePackPointer() = %#v, want %#v", gotPointer, wantPointer)
	}

	ecAfter, ok := ag.ActiveMissionStep()
	if !ok || ecAfter.Runtime == nil {
		t.Fatalf("ActiveMissionStep() after = %#v, want active runtime", ecAfter)
	}
	if ecAfter.Runtime.State != ecBefore.Runtime.State {
		t.Fatalf("ActiveMissionStep().Runtime.State = %q, want %q", ecAfter.Runtime.State, ecBefore.Runtime.State)
	}
	if ecAfter.Runtime.ActiveStepID != ecBefore.Runtime.ActiveStepID {
		t.Fatalf("ActiveMissionStep().Runtime.ActiveStepID = %q, want %q", ecAfter.Runtime.ActiveStepID, ecBefore.Runtime.ActiveStepID)
	}

	status, err := ag.ProcessDirect("STATUS job-1", 2*time.Second)
	if err != nil {
		t.Fatalf("ProcessDirect(STATUS) error = %v", err)
	}

	var summary missioncontrol.OperatorStatusSummary
	if err := json.Unmarshal([]byte(status), &summary); err != nil {
		t.Fatalf("json.Unmarshal(status) error = %v", err)
	}
	if summary.RollbackIdentity == nil {
		t.Fatal("RollbackIdentity = nil, want rollback identity block")
	}
	if summary.RollbackApplyIdentity == nil {
		t.Fatal("RollbackApplyIdentity = nil, want rollback-apply identity block")
	}
	if summary.RollbackIdentity.State != "configured" {
		t.Fatalf("RollbackIdentity.State = %q, want configured", summary.RollbackIdentity.State)
	}
	if summary.RollbackApplyIdentity.State != "configured" {
		t.Fatalf("RollbackApplyIdentity.State = %q, want configured", summary.RollbackApplyIdentity.State)
	}
	if len(summary.RollbackIdentity.Rollbacks) != 1 {
		t.Fatalf("RollbackIdentity.Rollbacks len = %d, want 1", len(summary.RollbackIdentity.Rollbacks))
	}
	if len(summary.RollbackApplyIdentity.Applies) != 1 {
		t.Fatalf("RollbackApplyIdentity.Applies len = %d, want 1", len(summary.RollbackApplyIdentity.Applies))
	}
	if summary.RollbackIdentity.Rollbacks[0].RollbackID != "rollback-1" {
		t.Fatalf("RollbackIdentity.Rollbacks[0].RollbackID = %q, want rollback-1", summary.RollbackIdentity.Rollbacks[0].RollbackID)
	}
	if summary.RollbackApplyIdentity.Applies[0].RollbackApplyID != "apply-1" {
		t.Fatalf("RollbackApplyIdentity.Applies[0].RollbackApplyID = %q, want apply-1", summary.RollbackApplyIdentity.Applies[0].RollbackApplyID)
	}
	if summary.RollbackApplyIdentity.Applies[0].RollbackID != "rollback-1" {
		t.Fatalf("RollbackApplyIdentity.Applies[0].RollbackID = %q, want rollback-1", summary.RollbackApplyIdentity.Applies[0].RollbackID)
	}
}

func TestProcessDirectRollbackApplyRecordCommandFailsClosedWhenRollbackIsMissing(t *testing.T) {
	t.Parallel()

	root := t.TempDir()

	b := chat.NewHub(10)
	prov := &finalResponseProvider{content: "unused"}
	ag := NewAgentLoop(b, prov, prov.GetDefaultModel(), 3, "", nil)
	ag.SetMissionStoreRoot(root)
	if err := ag.ActivateMissionStep(testMissionJob([]string{"read"}, []string{"read"}), "build"); err != nil {
		t.Fatalf("ActivateMissionStep() error = %v", err)
	}

	resp, err := ag.ProcessDirect("ROLLBACK_APPLY_RECORD job-1 missing-rollback apply-missing", 2*time.Second)
	if err == nil {
		t.Fatal("ProcessDirect(ROLLBACK_APPLY_RECORD) error = nil, want missing rollback rejection")
	}
	if resp != "" {
		t.Fatalf("ProcessDirect(ROLLBACK_APPLY_RECORD) response = %q, want empty on rejection", resp)
	}
	if !strings.Contains(err.Error(), missioncontrol.ErrRollbackRecordNotFound.Error()) {
		t.Fatalf("ProcessDirect(ROLLBACK_APPLY_RECORD) error = %q, want missing rollback rejection", err)
	}
	if _, err := missioncontrol.LoadRollbackApplyRecord(root, "apply-missing"); err != missioncontrol.ErrRollbackApplyRecordNotFound {
		t.Fatalf("LoadRollbackApplyRecord() error = %v, want %v", err, missioncontrol.ErrRollbackApplyRecordNotFound)
	}
}

func TestProcessDirectRollbackApplyPhaseCommandAdvancesWorkflowAndPreservesActiveRuntimePackPointer(t *testing.T) {
	t.Parallel()

	root, wantPointer := writeLoopRollbackPromotionFixtures(t)

	b := chat.NewHub(10)
	prov := &finalResponseProvider{content: "unused"}
	ag := NewAgentLoop(b, prov, prov.GetDefaultModel(), 3, "", nil)
	ag.SetMissionStoreRoot(root)
	if err := ag.ActivateMissionStep(testMissionJob([]string{"read"}, []string{"read"}), "build"); err != nil {
		t.Fatalf("ActivateMissionStep() error = %v", err)
	}

	if _, err := ag.ProcessDirect("ROLLBACK_RECORD job-1 promotion-1 rollback-1", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(ROLLBACK_RECORD) error = %v", err)
	}
	if _, err := ag.ProcessDirect("ROLLBACK_APPLY_RECORD job-1 rollback-1 apply-1", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(ROLLBACK_APPLY_RECORD) error = %v", err)
	}

	beforeBytes, err := os.ReadFile(missioncontrol.StoreActiveRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(active pointer before) error = %v", err)
	}

	resp, err := ag.ProcessDirect("ROLLBACK_APPLY_PHASE job-1 apply-1 validated", 2*time.Second)
	if err != nil {
		t.Fatalf("ProcessDirect(ROLLBACK_APPLY_PHASE validated) error = %v", err)
	}
	if resp != "Advanced rollback-apply workflow job=job-1 apply=apply-1 phase=validated." {
		t.Fatalf("ProcessDirect(ROLLBACK_APPLY_PHASE validated) response = %q, want validated acknowledgement", resp)
	}

	resp, err = ag.ProcessDirect("ROLLBACK_APPLY_PHASE job-1 apply-1 ready_to_apply", 2*time.Second)
	if err != nil {
		t.Fatalf("ProcessDirect(ROLLBACK_APPLY_PHASE ready_to_apply) error = %v", err)
	}
	if resp != "Advanced rollback-apply workflow job=job-1 apply=apply-1 phase=ready_to_apply." {
		t.Fatalf("ProcessDirect(ROLLBACK_APPLY_PHASE ready_to_apply) response = %q, want ready acknowledgement", resp)
	}

	record, err := missioncontrol.LoadRollbackApplyRecord(root, "apply-1")
	if err != nil {
		t.Fatalf("LoadRollbackApplyRecord() error = %v", err)
	}
	if record.Phase != missioncontrol.RollbackApplyPhaseReadyToApply {
		t.Fatalf("RollbackApplyRecord.Phase = %q, want ready_to_apply", record.Phase)
	}
	if record.ActivationState != missioncontrol.RollbackApplyActivationStateUnchanged {
		t.Fatalf("RollbackApplyRecord.ActivationState = %q, want unchanged", record.ActivationState)
	}
	if record.PhaseUpdatedBy != "operator" {
		t.Fatalf("RollbackApplyRecord.PhaseUpdatedBy = %q, want operator", record.PhaseUpdatedBy)
	}

	gotPointer, err := missioncontrol.LoadActiveRuntimePackPointer(root)
	if err != nil {
		t.Fatalf("LoadActiveRuntimePackPointer() error = %v", err)
	}
	if gotPointer != wantPointer {
		t.Fatalf("LoadActiveRuntimePackPointer() = %#v, want %#v", gotPointer, wantPointer)
	}
	afterBytes, err := os.ReadFile(missioncontrol.StoreActiveRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(active pointer after) error = %v", err)
	}
	if string(beforeBytes) != string(afterBytes) {
		t.Fatalf("active runtime pack pointer file changed\nbefore:\n%s\nafter:\n%s", string(beforeBytes), string(afterBytes))
	}

	status, err := ag.ProcessDirect("STATUS job-1", 2*time.Second)
	if err != nil {
		t.Fatalf("ProcessDirect(STATUS) error = %v", err)
	}

	var summary missioncontrol.OperatorStatusSummary
	if err := json.Unmarshal([]byte(status), &summary); err != nil {
		t.Fatalf("json.Unmarshal(status) error = %v", err)
	}
	if summary.RollbackApplyIdentity == nil {
		t.Fatal("RollbackApplyIdentity = nil, want rollback-apply identity block")
	}
	if summary.RollbackApplyIdentity.State != "configured" {
		t.Fatalf("RollbackApplyIdentity.State = %q, want configured", summary.RollbackApplyIdentity.State)
	}
	if len(summary.RollbackApplyIdentity.Applies) != 1 {
		t.Fatalf("RollbackApplyIdentity.Applies len = %d, want 1", len(summary.RollbackApplyIdentity.Applies))
	}
	apply := summary.RollbackApplyIdentity.Applies[0]
	if apply.RollbackApplyID != "apply-1" {
		t.Fatalf("RollbackApplyIdentity.Applies[0].RollbackApplyID = %q, want apply-1", apply.RollbackApplyID)
	}
	if apply.Phase != string(missioncontrol.RollbackApplyPhaseReadyToApply) {
		t.Fatalf("RollbackApplyIdentity.Applies[0].Phase = %q, want ready_to_apply", apply.Phase)
	}
	if apply.ActivationState != string(missioncontrol.RollbackApplyActivationStateUnchanged) {
		t.Fatalf("RollbackApplyIdentity.Applies[0].ActivationState = %q, want unchanged", apply.ActivationState)
	}
}

func TestProcessDirectRollbackApplyPhaseCommandRejectsInvalidTransition(t *testing.T) {
	t.Parallel()

	root, wantPointer := writeLoopRollbackPromotionFixtures(t)

	b := chat.NewHub(10)
	prov := &finalResponseProvider{content: "unused"}
	ag := NewAgentLoop(b, prov, prov.GetDefaultModel(), 3, "", nil)
	ag.SetMissionStoreRoot(root)
	if err := ag.ActivateMissionStep(testMissionJob([]string{"read"}, []string{"read"}), "build"); err != nil {
		t.Fatalf("ActivateMissionStep() error = %v", err)
	}

	if _, err := ag.ProcessDirect("ROLLBACK_RECORD job-1 promotion-1 rollback-1", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(ROLLBACK_RECORD) error = %v", err)
	}
	if _, err := ag.ProcessDirect("ROLLBACK_APPLY_RECORD job-1 rollback-1 apply-1", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(ROLLBACK_APPLY_RECORD) error = %v", err)
	}

	beforeBytes, err := os.ReadFile(missioncontrol.StoreActiveRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(active pointer before) error = %v", err)
	}

	resp, err := ag.ProcessDirect("ROLLBACK_APPLY_PHASE job-1 apply-1 ready_to_apply", 2*time.Second)
	if err == nil {
		t.Fatal("ProcessDirect(ROLLBACK_APPLY_PHASE) error = nil, want invalid transition rejection")
	}
	if resp != "" {
		t.Fatalf("ProcessDirect(ROLLBACK_APPLY_PHASE) response = %q, want empty on rejection", resp)
	}
	if !strings.Contains(err.Error(), `phase transition "recorded" -> "ready_to_apply" is invalid`) {
		t.Fatalf("ProcessDirect(ROLLBACK_APPLY_PHASE) error = %q, want invalid transition rejection", err)
	}

	record, err := missioncontrol.LoadRollbackApplyRecord(root, "apply-1")
	if err != nil {
		t.Fatalf("LoadRollbackApplyRecord() error = %v", err)
	}
	if record.Phase != missioncontrol.RollbackApplyPhaseRecorded {
		t.Fatalf("LoadRollbackApplyRecord().Phase = %q, want recorded after rejection", record.Phase)
	}
	if record.ActivationState != missioncontrol.RollbackApplyActivationStateUnchanged {
		t.Fatalf("LoadRollbackApplyRecord().ActivationState = %q, want unchanged after rejection", record.ActivationState)
	}

	gotPointer, err := missioncontrol.LoadActiveRuntimePackPointer(root)
	if err != nil {
		t.Fatalf("LoadActiveRuntimePackPointer() error = %v", err)
	}
	if gotPointer != wantPointer {
		t.Fatalf("LoadActiveRuntimePackPointer() = %#v, want %#v", gotPointer, wantPointer)
	}
	afterBytes, err := os.ReadFile(missioncontrol.StoreActiveRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(active pointer after) error = %v", err)
	}
	if string(beforeBytes) != string(afterBytes) {
		t.Fatalf("active runtime pack pointer file changed\nbefore:\n%s\nafter:\n%s", string(beforeBytes), string(afterBytes))
	}
}

func TestProcessDirectRollbackApplyExecuteCommandSwitchesPointerAndIsReplaySafe(t *testing.T) {
	t.Parallel()

	root, _ := writeLoopRollbackPromotionFixtures(t)
	now := time.Date(2026, 4, 21, 12, 5, 0, 0, time.UTC)
	if err := missioncontrol.StoreLastKnownGoodRuntimePackPointer(root, missioncontrol.LastKnownGoodRuntimePackPointer{
		PackID:            "pack-base",
		Basis:             "holdout_pass",
		VerifiedAt:        now,
		VerifiedBy:        "operator",
		RollbackRecordRef: "promotion:promotion-1",
	}); err != nil {
		t.Fatalf("StoreLastKnownGoodRuntimePackPointer() error = %v", err)
	}

	b := chat.NewHub(10)
	prov := &finalResponseProvider{content: "unused"}
	ag := NewAgentLoop(b, prov, prov.GetDefaultModel(), 3, "", nil)
	ag.SetMissionStoreRoot(root)
	if err := ag.ActivateMissionStep(testMissionJob([]string{"read"}, []string{"read"}), "build"); err != nil {
		t.Fatalf("ActivateMissionStep() error = %v", err)
	}

	if _, err := ag.ProcessDirect("ROLLBACK_RECORD job-1 promotion-1 rollback-1", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(ROLLBACK_RECORD) error = %v", err)
	}
	if _, err := ag.ProcessDirect("ROLLBACK_APPLY_RECORD job-1 rollback-1 apply-1", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(ROLLBACK_APPLY_RECORD) error = %v", err)
	}
	if _, err := ag.ProcessDirect("ROLLBACK_APPLY_PHASE job-1 apply-1 validated", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(ROLLBACK_APPLY_PHASE validated) error = %v", err)
	}
	if _, err := ag.ProcessDirect("ROLLBACK_APPLY_PHASE job-1 apply-1 ready_to_apply", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(ROLLBACK_APPLY_PHASE ready_to_apply) error = %v", err)
	}

	beforeLastKnownGoodBytes, err := os.ReadFile(missioncontrol.StoreLastKnownGoodRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(last known good before) error = %v", err)
	}

	resp, err := ag.ProcessDirect("ROLLBACK_APPLY_EXECUTE job-1 apply-1", 2*time.Second)
	if err != nil {
		t.Fatalf("ProcessDirect(ROLLBACK_APPLY_EXECUTE first) error = %v", err)
	}
	if resp != "Executed rollback-apply pointer switch job=job-1 apply=apply-1." {
		t.Fatalf("ProcessDirect(ROLLBACK_APPLY_EXECUTE first) response = %q, want execute acknowledgement", resp)
	}

	record, err := missioncontrol.LoadRollbackApplyRecord(root, "apply-1")
	if err != nil {
		t.Fatalf("LoadRollbackApplyRecord() error = %v", err)
	}
	if record.Phase != missioncontrol.RollbackApplyPhasePointerSwitchedReloadPending {
		t.Fatalf("RollbackApplyRecord.Phase = %q, want pointer_switched_reload_pending", record.Phase)
	}
	if record.ActivationState != missioncontrol.RollbackApplyActivationStateUnchanged {
		t.Fatalf("RollbackApplyRecord.ActivationState = %q, want unchanged", record.ActivationState)
	}

	gotPointer, err := missioncontrol.LoadActiveRuntimePackPointer(root)
	if err != nil {
		t.Fatalf("LoadActiveRuntimePackPointer() error = %v", err)
	}
	if gotPointer.ActivePackID != "pack-base" {
		t.Fatalf("LoadActiveRuntimePackPointer().ActivePackID = %q, want pack-base", gotPointer.ActivePackID)
	}
	if gotPointer.PreviousActivePackID != "pack-candidate" {
		t.Fatalf("LoadActiveRuntimePackPointer().PreviousActivePackID = %q, want pack-candidate", gotPointer.PreviousActivePackID)
	}
	if gotPointer.LastKnownGoodPackID != "pack-base" {
		t.Fatalf("LoadActiveRuntimePackPointer().LastKnownGoodPackID = %q, want pack-base", gotPointer.LastKnownGoodPackID)
	}
	if gotPointer.UpdateRecordRef != "rollback_apply:apply-1" {
		t.Fatalf("LoadActiveRuntimePackPointer().UpdateRecordRef = %q, want rollback_apply:apply-1", gotPointer.UpdateRecordRef)
	}
	if gotPointer.ReloadGeneration != 8 {
		t.Fatalf("LoadActiveRuntimePackPointer().ReloadGeneration = %d, want 8", gotPointer.ReloadGeneration)
	}

	firstPointerBytes, err := os.ReadFile(missioncontrol.StoreActiveRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(active pointer first) error = %v", err)
	}

	resp, err = ag.ProcessDirect("ROLLBACK_APPLY_EXECUTE job-1 apply-1", 2*time.Second)
	if err != nil {
		t.Fatalf("ProcessDirect(ROLLBACK_APPLY_EXECUTE second) error = %v", err)
	}
	if resp != "Selected rollback-apply pointer switch job=job-1 apply=apply-1." {
		t.Fatalf("ProcessDirect(ROLLBACK_APPLY_EXECUTE second) response = %q, want replay acknowledgement", resp)
	}

	secondPointerBytes, err := os.ReadFile(missioncontrol.StoreActiveRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(active pointer second) error = %v", err)
	}
	if string(firstPointerBytes) != string(secondPointerBytes) {
		t.Fatalf("active runtime pack pointer file changed on replay\nfirst:\n%s\nsecond:\n%s", string(firstPointerBytes), string(secondPointerBytes))
	}

	afterLastKnownGoodBytes, err := os.ReadFile(missioncontrol.StoreLastKnownGoodRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(last known good after) error = %v", err)
	}
	if string(beforeLastKnownGoodBytes) != string(afterLastKnownGoodBytes) {
		t.Fatalf("last-known-good pointer file changed\nbefore:\n%s\nafter:\n%s", string(beforeLastKnownGoodBytes), string(afterLastKnownGoodBytes))
	}

	status, err := ag.ProcessDirect("STATUS job-1", 2*time.Second)
	if err != nil {
		t.Fatalf("ProcessDirect(STATUS) error = %v", err)
	}

	var summary missioncontrol.OperatorStatusSummary
	if err := json.Unmarshal([]byte(status), &summary); err != nil {
		t.Fatalf("json.Unmarshal(status) error = %v", err)
	}
	if summary.RuntimePackIdentity == nil {
		t.Fatal("RuntimePackIdentity = nil, want runtime pack identity block")
	}
	if summary.RollbackApplyIdentity == nil {
		t.Fatal("RollbackApplyIdentity = nil, want rollback-apply identity block")
	}
	if summary.RuntimePackIdentity.Active.ActivePackID != "pack-base" {
		t.Fatalf("RuntimePackIdentity.Active.ActivePackID = %q, want pack-base", summary.RuntimePackIdentity.Active.ActivePackID)
	}
	if summary.RuntimePackIdentity.Active.PreviousActivePackID != "pack-candidate" {
		t.Fatalf("RuntimePackIdentity.Active.PreviousActivePackID = %q, want pack-candidate", summary.RuntimePackIdentity.Active.PreviousActivePackID)
	}
	if summary.RollbackApplyIdentity.Applies[0].Phase != string(missioncontrol.RollbackApplyPhasePointerSwitchedReloadPending) {
		t.Fatalf("RollbackApplyIdentity.Applies[0].Phase = %q, want pointer_switched_reload_pending", summary.RollbackApplyIdentity.Applies[0].Phase)
	}
	if summary.RollbackApplyIdentity.Applies[0].ActivationState != string(missioncontrol.RollbackApplyActivationStateUnchanged) {
		t.Fatalf("RollbackApplyIdentity.Applies[0].ActivationState = %q, want unchanged", summary.RollbackApplyIdentity.Applies[0].ActivationState)
	}
}

func TestProcessDirectRollbackApplyExecuteCommandRejectsInvalidPhaseWithoutPointerMutation(t *testing.T) {
	t.Parallel()

	root, wantPointer := writeLoopRollbackPromotionFixtures(t)

	b := chat.NewHub(10)
	prov := &finalResponseProvider{content: "unused"}
	ag := NewAgentLoop(b, prov, prov.GetDefaultModel(), 3, "", nil)
	ag.SetMissionStoreRoot(root)
	if err := ag.ActivateMissionStep(testMissionJob([]string{"read"}, []string{"read"}), "build"); err != nil {
		t.Fatalf("ActivateMissionStep() error = %v", err)
	}

	if _, err := ag.ProcessDirect("ROLLBACK_RECORD job-1 promotion-1 rollback-1", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(ROLLBACK_RECORD) error = %v", err)
	}
	if _, err := ag.ProcessDirect("ROLLBACK_APPLY_RECORD job-1 rollback-1 apply-1", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(ROLLBACK_APPLY_RECORD) error = %v", err)
	}

	beforePointerBytes, err := os.ReadFile(missioncontrol.StoreActiveRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(active pointer before) error = %v", err)
	}

	resp, err := ag.ProcessDirect("ROLLBACK_APPLY_EXECUTE job-1 apply-1", 2*time.Second)
	if err == nil {
		t.Fatal("ProcessDirect(ROLLBACK_APPLY_EXECUTE) error = nil, want invalid phase rejection")
	}
	if resp != "" {
		t.Fatalf("ProcessDirect(ROLLBACK_APPLY_EXECUTE) response = %q, want empty on rejection", resp)
	}
	if !strings.Contains(err.Error(), `phase "recorded" does not permit pointer switch execution`) {
		t.Fatalf("ProcessDirect(ROLLBACK_APPLY_EXECUTE) error = %q, want invalid phase rejection", err)
	}

	record, err := missioncontrol.LoadRollbackApplyRecord(root, "apply-1")
	if err != nil {
		t.Fatalf("LoadRollbackApplyRecord() error = %v", err)
	}
	if record.Phase != missioncontrol.RollbackApplyPhaseRecorded {
		t.Fatalf("LoadRollbackApplyRecord().Phase = %q, want recorded after rejection", record.Phase)
	}

	gotPointer, err := missioncontrol.LoadActiveRuntimePackPointer(root)
	if err != nil {
		t.Fatalf("LoadActiveRuntimePackPointer() error = %v", err)
	}
	if gotPointer != wantPointer {
		t.Fatalf("LoadActiveRuntimePackPointer() = %#v, want %#v", gotPointer, wantPointer)
	}
	afterPointerBytes, err := os.ReadFile(missioncontrol.StoreActiveRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(active pointer after) error = %v", err)
	}
	if string(beforePointerBytes) != string(afterPointerBytes) {
		t.Fatalf("active runtime pack pointer file changed\nbefore:\n%s\nafter:\n%s", string(beforePointerBytes), string(afterPointerBytes))
	}
}

func TestProcessDirectRollbackApplyReloadCommandSucceedsWithoutSecondPointerMutation(t *testing.T) {
	t.Parallel()

	root, _ := writeLoopRollbackPromotionFixtures(t)
	now := time.Date(2026, 4, 21, 12, 6, 0, 0, time.UTC)
	if err := missioncontrol.StoreLastKnownGoodRuntimePackPointer(root, missioncontrol.LastKnownGoodRuntimePackPointer{
		PackID:            "pack-base",
		Basis:             "holdout_pass",
		VerifiedAt:        now,
		VerifiedBy:        "operator",
		RollbackRecordRef: "promotion:promotion-1",
	}); err != nil {
		t.Fatalf("StoreLastKnownGoodRuntimePackPointer() error = %v", err)
	}

	b := chat.NewHub(10)
	prov := &finalResponseProvider{content: "unused"}
	ag := NewAgentLoop(b, prov, prov.GetDefaultModel(), 3, "", nil)
	ag.SetMissionStoreRoot(root)
	if err := ag.ActivateMissionStep(testMissionJob([]string{"read"}, []string{"read"}), "build"); err != nil {
		t.Fatalf("ActivateMissionStep() error = %v", err)
	}

	if _, err := ag.ProcessDirect("ROLLBACK_RECORD job-1 promotion-1 rollback-1", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(ROLLBACK_RECORD) error = %v", err)
	}
	if _, err := ag.ProcessDirect("ROLLBACK_APPLY_RECORD job-1 rollback-1 apply-1", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(ROLLBACK_APPLY_RECORD) error = %v", err)
	}
	if _, err := ag.ProcessDirect("ROLLBACK_APPLY_PHASE job-1 apply-1 validated", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(ROLLBACK_APPLY_PHASE validated) error = %v", err)
	}
	if _, err := ag.ProcessDirect("ROLLBACK_APPLY_PHASE job-1 apply-1 ready_to_apply", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(ROLLBACK_APPLY_PHASE ready_to_apply) error = %v", err)
	}
	if _, err := ag.ProcessDirect("ROLLBACK_APPLY_EXECUTE job-1 apply-1", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(ROLLBACK_APPLY_EXECUTE) error = %v", err)
	}

	beforePointerBytes, err := os.ReadFile(missioncontrol.StoreActiveRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(active pointer before reload) error = %v", err)
	}
	beforeLastKnownGoodBytes, err := os.ReadFile(missioncontrol.StoreLastKnownGoodRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(last known good before reload) error = %v", err)
	}

	resp, err := ag.ProcessDirect("ROLLBACK_APPLY_RELOAD job-1 apply-1", 2*time.Second)
	if err != nil {
		t.Fatalf("ProcessDirect(ROLLBACK_APPLY_RELOAD first) error = %v", err)
	}
	if resp != "Executed rollback-apply reload/apply job=job-1 apply=apply-1." {
		t.Fatalf("ProcessDirect(ROLLBACK_APPLY_RELOAD first) response = %q, want reload/apply acknowledgement", resp)
	}

	record, err := missioncontrol.LoadRollbackApplyRecord(root, "apply-1")
	if err != nil {
		t.Fatalf("LoadRollbackApplyRecord() error = %v", err)
	}
	if record.Phase != missioncontrol.RollbackApplyPhaseReloadApplySucceeded {
		t.Fatalf("RollbackApplyRecord.Phase = %q, want reload_apply_succeeded", record.Phase)
	}
	if record.ExecutionError != "" {
		t.Fatalf("RollbackApplyRecord.ExecutionError = %q, want empty", record.ExecutionError)
	}

	firstPointerBytes, err := os.ReadFile(missioncontrol.StoreActiveRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(active pointer first) error = %v", err)
	}
	if string(beforePointerBytes) != string(firstPointerBytes) {
		t.Fatalf("active runtime pack pointer file changed during reload/apply\nbefore:\n%s\nafter:\n%s", string(beforePointerBytes), string(firstPointerBytes))
	}

	resp, err = ag.ProcessDirect("ROLLBACK_APPLY_RELOAD job-1 apply-1", 2*time.Second)
	if err != nil {
		t.Fatalf("ProcessDirect(ROLLBACK_APPLY_RELOAD second) error = %v", err)
	}
	if resp != "Selected rollback-apply reload/apply job=job-1 apply=apply-1." {
		t.Fatalf("ProcessDirect(ROLLBACK_APPLY_RELOAD second) response = %q, want replay acknowledgement", resp)
	}

	secondPointerBytes, err := os.ReadFile(missioncontrol.StoreActiveRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(active pointer second) error = %v", err)
	}
	if string(firstPointerBytes) != string(secondPointerBytes) {
		t.Fatalf("active runtime pack pointer file changed on reload/apply replay\nfirst:\n%s\nsecond:\n%s", string(firstPointerBytes), string(secondPointerBytes))
	}
	afterLastKnownGoodBytes, err := os.ReadFile(missioncontrol.StoreLastKnownGoodRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(last known good after reload) error = %v", err)
	}
	if string(beforeLastKnownGoodBytes) != string(afterLastKnownGoodBytes) {
		t.Fatalf("last-known-good pointer file changed during reload/apply\nbefore:\n%s\nafter:\n%s", string(beforeLastKnownGoodBytes), string(afterLastKnownGoodBytes))
	}

	gotPointer, err := missioncontrol.LoadActiveRuntimePackPointer(root)
	if err != nil {
		t.Fatalf("LoadActiveRuntimePackPointer() error = %v", err)
	}
	if gotPointer.ActivePackID != "pack-base" {
		t.Fatalf("LoadActiveRuntimePackPointer().ActivePackID = %q, want pack-base", gotPointer.ActivePackID)
	}
	if gotPointer.ReloadGeneration != 8 {
		t.Fatalf("LoadActiveRuntimePackPointer().ReloadGeneration = %d, want 8", gotPointer.ReloadGeneration)
	}

	status, err := ag.ProcessDirect("STATUS job-1", 2*time.Second)
	if err != nil {
		t.Fatalf("ProcessDirect(STATUS) error = %v", err)
	}

	var summary missioncontrol.OperatorStatusSummary
	if err := json.Unmarshal([]byte(status), &summary); err != nil {
		t.Fatalf("json.Unmarshal(status) error = %v", err)
	}
	if summary.RollbackApplyIdentity == nil {
		t.Fatal("RollbackApplyIdentity = nil, want rollback-apply identity block")
	}
	if summary.RollbackApplyIdentity.Applies[0].Phase != string(missioncontrol.RollbackApplyPhaseReloadApplySucceeded) {
		t.Fatalf("RollbackApplyIdentity.Applies[0].Phase = %q, want reload_apply_succeeded", summary.RollbackApplyIdentity.Applies[0].Phase)
	}
}

func TestProcessDirectRollbackApplyReloadCommandRetriesFromRecoveryNeeded(t *testing.T) {
	t.Parallel()

	root, _ := writeLoopRollbackPromotionFixtures(t)
	now := time.Date(2026, 4, 21, 12, 8, 0, 0, time.UTC)
	if err := missioncontrol.StoreLastKnownGoodRuntimePackPointer(root, missioncontrol.LastKnownGoodRuntimePackPointer{
		PackID:            "pack-base",
		Basis:             "holdout_pass",
		VerifiedAt:        now,
		VerifiedBy:        "operator",
		RollbackRecordRef: "promotion:promotion-1",
	}); err != nil {
		t.Fatalf("StoreLastKnownGoodRuntimePackPointer() error = %v", err)
	}

	b := chat.NewHub(10)
	prov := &finalResponseProvider{content: "unused"}
	ag := NewAgentLoop(b, prov, prov.GetDefaultModel(), 3, "", nil)
	ag.SetMissionStoreRoot(root)
	if err := ag.ActivateMissionStep(testMissionJob([]string{"read"}, []string{"read"}), "build"); err != nil {
		t.Fatalf("ActivateMissionStep() error = %v", err)
	}

	if _, err := ag.ProcessDirect("ROLLBACK_RECORD job-1 promotion-1 rollback-1", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(ROLLBACK_RECORD) error = %v", err)
	}
	if _, err := ag.ProcessDirect("ROLLBACK_APPLY_RECORD job-1 rollback-1 apply-1", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(ROLLBACK_APPLY_RECORD) error = %v", err)
	}
	if _, err := ag.ProcessDirect("ROLLBACK_APPLY_PHASE job-1 apply-1 validated", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(ROLLBACK_APPLY_PHASE validated) error = %v", err)
	}
	if _, err := ag.ProcessDirect("ROLLBACK_APPLY_PHASE job-1 apply-1 ready_to_apply", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(ROLLBACK_APPLY_PHASE ready_to_apply) error = %v", err)
	}
	if _, err := ag.ProcessDirect("ROLLBACK_APPLY_EXECUTE job-1 apply-1", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(ROLLBACK_APPLY_EXECUTE) error = %v", err)
	}

	record, err := missioncontrol.LoadRollbackApplyRecord(root, "apply-1")
	if err != nil {
		t.Fatalf("LoadRollbackApplyRecord() error = %v", err)
	}
	recoveryAt := record.CreatedAt.Add(time.Minute)
	record.Phase = missioncontrol.RollbackApplyPhaseReloadApplyInProgress
	record.ExecutionError = ""
	record.PhaseUpdatedAt = recoveryAt.UTC()
	record.PhaseUpdatedBy = "operator"
	record = missioncontrol.NormalizeRollbackApplyRecord(record)
	if err := missioncontrol.ValidateRollbackApplyRecord(record); err != nil {
		t.Fatalf("ValidateRollbackApplyRecord() error = %v", err)
	}
	if err := missioncontrol.WriteStoreJSONAtomic(missioncontrol.StoreRollbackApplyPath(root, "apply-1"), record); err != nil {
		t.Fatalf("WriteStoreJSONAtomic(reload_apply_in_progress) error = %v", err)
	}
	if _, changed, err := missioncontrol.ReconcileRollbackApplyRecoveryNeeded(root, "apply-1", "operator", recoveryAt.Add(time.Minute)); err != nil {
		t.Fatalf("ReconcileRollbackApplyRecoveryNeeded() error = %v", err)
	} else if !changed {
		t.Fatal("ReconcileRollbackApplyRecoveryNeeded() changed = false, want true")
	}

	beforePointerBytes, err := os.ReadFile(missioncontrol.StoreActiveRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(active pointer before retry) error = %v", err)
	}
	beforeLastKnownGoodBytes, err := os.ReadFile(missioncontrol.StoreLastKnownGoodRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(last known good before retry) error = %v", err)
	}

	resp, err := ag.ProcessDirect("ROLLBACK_APPLY_RELOAD job-1 apply-1", 2*time.Second)
	if err != nil {
		t.Fatalf("ProcessDirect(ROLLBACK_APPLY_RELOAD retry) error = %v", err)
	}
	if resp != "Executed rollback-apply reload/apply job=job-1 apply=apply-1." {
		t.Fatalf("ProcessDirect(ROLLBACK_APPLY_RELOAD retry) response = %q, want reload/apply acknowledgement", resp)
	}

	record, err = missioncontrol.LoadRollbackApplyRecord(root, "apply-1")
	if err != nil {
		t.Fatalf("LoadRollbackApplyRecord(retry result) error = %v", err)
	}
	if record.Phase != missioncontrol.RollbackApplyPhaseReloadApplySucceeded {
		t.Fatalf("RollbackApplyRecord.Phase = %q, want reload_apply_succeeded", record.Phase)
	}
	if record.ExecutionError != "" {
		t.Fatalf("RollbackApplyRecord.ExecutionError = %q, want empty", record.ExecutionError)
	}

	afterPointerBytes, err := os.ReadFile(missioncontrol.StoreActiveRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(active pointer after retry) error = %v", err)
	}
	if string(beforePointerBytes) != string(afterPointerBytes) {
		t.Fatalf("active runtime pack pointer file changed during retry reload/apply\nbefore:\n%s\nafter:\n%s", string(beforePointerBytes), string(afterPointerBytes))
	}
	afterLastKnownGoodBytes, err := os.ReadFile(missioncontrol.StoreLastKnownGoodRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(last known good after retry) error = %v", err)
	}
	if string(beforeLastKnownGoodBytes) != string(afterLastKnownGoodBytes) {
		t.Fatalf("last-known-good pointer file changed during retry reload/apply\nbefore:\n%s\nafter:\n%s", string(beforeLastKnownGoodBytes), string(afterLastKnownGoodBytes))
	}

	gotPointer, err := missioncontrol.LoadActiveRuntimePackPointer(root)
	if err != nil {
		t.Fatalf("LoadActiveRuntimePackPointer() error = %v", err)
	}
	if gotPointer.ReloadGeneration != 8 {
		t.Fatalf("LoadActiveRuntimePackPointer().ReloadGeneration = %d, want 8", gotPointer.ReloadGeneration)
	}
}

func TestProcessDirectRollbackApplyFailCommandResolvesRecoveryNeededAndPreservesStatus(t *testing.T) {
	t.Parallel()

	root, _ := writeLoopRollbackPromotionFixtures(t)
	now := time.Date(2026, 4, 21, 12, 10, 0, 0, time.UTC)
	if err := missioncontrol.StoreLastKnownGoodRuntimePackPointer(root, missioncontrol.LastKnownGoodRuntimePackPointer{
		PackID:            "pack-base",
		Basis:             "holdout_pass",
		VerifiedAt:        now,
		VerifiedBy:        "operator",
		RollbackRecordRef: "promotion:promotion-1",
	}); err != nil {
		t.Fatalf("StoreLastKnownGoodRuntimePackPointer() error = %v", err)
	}

	b := chat.NewHub(10)
	prov := &finalResponseProvider{content: "unused"}
	ag := NewAgentLoop(b, prov, prov.GetDefaultModel(), 3, "", nil)
	ag.SetMissionStoreRoot(root)
	if err := ag.ActivateMissionStep(testMissionJob([]string{"read"}, []string{"read"}), "build"); err != nil {
		t.Fatalf("ActivateMissionStep() error = %v", err)
	}

	if _, err := ag.ProcessDirect("ROLLBACK_RECORD job-1 promotion-1 rollback-1", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(ROLLBACK_RECORD) error = %v", err)
	}
	if _, err := ag.ProcessDirect("ROLLBACK_APPLY_RECORD job-1 rollback-1 apply-1", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(ROLLBACK_APPLY_RECORD) error = %v", err)
	}
	if _, err := ag.ProcessDirect("ROLLBACK_APPLY_PHASE job-1 apply-1 validated", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(ROLLBACK_APPLY_PHASE validated) error = %v", err)
	}
	if _, err := ag.ProcessDirect("ROLLBACK_APPLY_PHASE job-1 apply-1 ready_to_apply", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(ROLLBACK_APPLY_PHASE ready_to_apply) error = %v", err)
	}
	if _, err := ag.ProcessDirect("ROLLBACK_APPLY_EXECUTE job-1 apply-1", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(ROLLBACK_APPLY_EXECUTE) error = %v", err)
	}

	record, err := missioncontrol.LoadRollbackApplyRecord(root, "apply-1")
	if err != nil {
		t.Fatalf("LoadRollbackApplyRecord() error = %v", err)
	}
	recoveryAt := record.CreatedAt.Add(time.Minute)
	record.Phase = missioncontrol.RollbackApplyPhaseReloadApplyInProgress
	record.ExecutionError = ""
	record.PhaseUpdatedAt = recoveryAt.UTC()
	record.PhaseUpdatedBy = "operator"
	record = missioncontrol.NormalizeRollbackApplyRecord(record)
	if err := missioncontrol.ValidateRollbackApplyRecord(record); err != nil {
		t.Fatalf("ValidateRollbackApplyRecord() error = %v", err)
	}
	if err := missioncontrol.WriteStoreJSONAtomic(missioncontrol.StoreRollbackApplyPath(root, "apply-1"), record); err != nil {
		t.Fatalf("WriteStoreJSONAtomic(reload_apply_in_progress) error = %v", err)
	}
	if _, changed, err := missioncontrol.ReconcileRollbackApplyRecoveryNeeded(root, "apply-1", "operator", recoveryAt.Add(time.Minute)); err != nil {
		t.Fatalf("ReconcileRollbackApplyRecoveryNeeded() error = %v", err)
	} else if !changed {
		t.Fatal("ReconcileRollbackApplyRecoveryNeeded() changed = false, want true")
	}

	beforePointerBytes, err := os.ReadFile(missioncontrol.StoreActiveRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(active pointer before fail) error = %v", err)
	}
	beforeLastKnownGoodBytes, err := os.ReadFile(missioncontrol.StoreLastKnownGoodRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(last known good before fail) error = %v", err)
	}

	resp, err := ag.ProcessDirect("ROLLBACK_APPLY_FAIL job-1 apply-1 operator requested stop after recovery review", 2*time.Second)
	if err != nil {
		t.Fatalf("ProcessDirect(ROLLBACK_APPLY_FAIL) error = %v", err)
	}
	if resp != "Resolved rollback-apply terminal failure job=job-1 apply=apply-1." {
		t.Fatalf("ProcessDirect(ROLLBACK_APPLY_FAIL) response = %q, want terminal failure acknowledgement", resp)
	}

	record, err = missioncontrol.LoadRollbackApplyRecord(root, "apply-1")
	if err != nil {
		t.Fatalf("LoadRollbackApplyRecord(result) error = %v", err)
	}
	if record.Phase != missioncontrol.RollbackApplyPhaseReloadApplyFailed {
		t.Fatalf("RollbackApplyRecord.Phase = %q, want reload_apply_failed", record.Phase)
	}
	if record.ExecutionError != "operator_terminal_failure: operator requested stop after recovery review" {
		t.Fatalf("RollbackApplyRecord.ExecutionError = %q, want deterministic terminal failure detail", record.ExecutionError)
	}

	afterPointerBytes, err := os.ReadFile(missioncontrol.StoreActiveRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(active pointer after fail) error = %v", err)
	}
	if string(beforePointerBytes) != string(afterPointerBytes) {
		t.Fatalf("active runtime pack pointer file changed during terminal failure resolution\nbefore:\n%s\nafter:\n%s", string(beforePointerBytes), string(afterPointerBytes))
	}
	afterLastKnownGoodBytes, err := os.ReadFile(missioncontrol.StoreLastKnownGoodRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(last known good after fail) error = %v", err)
	}
	if string(beforeLastKnownGoodBytes) != string(afterLastKnownGoodBytes) {
		t.Fatalf("last-known-good pointer file changed during terminal failure resolution\nbefore:\n%s\nafter:\n%s", string(beforeLastKnownGoodBytes), string(afterLastKnownGoodBytes))
	}

	gotPointer, err := missioncontrol.LoadActiveRuntimePackPointer(root)
	if err != nil {
		t.Fatalf("LoadActiveRuntimePackPointer() error = %v", err)
	}
	if gotPointer.ReloadGeneration != 8 {
		t.Fatalf("LoadActiveRuntimePackPointer().ReloadGeneration = %d, want 8", gotPointer.ReloadGeneration)
	}

	status, err := ag.ProcessDirect("STATUS job-1", 2*time.Second)
	if err != nil {
		t.Fatalf("ProcessDirect(STATUS) error = %v", err)
	}

	var summary missioncontrol.OperatorStatusSummary
	if err := json.Unmarshal([]byte(status), &summary); err != nil {
		t.Fatalf("json.Unmarshal(status) error = %v", err)
	}
	if summary.RollbackApplyIdentity == nil {
		t.Fatal("RollbackApplyIdentity = nil, want rollback-apply identity block")
	}
	if summary.RollbackApplyIdentity.Applies[0].Phase != string(missioncontrol.RollbackApplyPhaseReloadApplyFailed) {
		t.Fatalf("RollbackApplyIdentity.Applies[0].Phase = %q, want reload_apply_failed", summary.RollbackApplyIdentity.Applies[0].Phase)
	}
}

func TestProcessDirectRollbackApplyFailCommandRequiresReasonAndRejectsInvalidStartingPhase(t *testing.T) {
	t.Parallel()

	root, _ := writeLoopRollbackPromotionFixtures(t)

	b := chat.NewHub(10)
	prov := &finalResponseProvider{content: "unused"}
	ag := NewAgentLoop(b, prov, prov.GetDefaultModel(), 3, "", nil)
	ag.SetMissionStoreRoot(root)
	if err := ag.ActivateMissionStep(testMissionJob([]string{"read"}, []string{"read"}), "build"); err != nil {
		t.Fatalf("ActivateMissionStep() error = %v", err)
	}

	if _, err := ag.ProcessDirect("ROLLBACK_RECORD job-1 promotion-1 rollback-1", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(ROLLBACK_RECORD) error = %v", err)
	}
	if _, err := ag.ProcessDirect("ROLLBACK_APPLY_RECORD job-1 rollback-1 apply-1", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(ROLLBACK_APPLY_RECORD) error = %v", err)
	}

	resp, err := ag.ProcessDirect("ROLLBACK_APPLY_FAIL job-1 apply-1", 2*time.Second)
	if err == nil {
		t.Fatal("ProcessDirect(ROLLBACK_APPLY_FAIL missing reason) error = nil, want required reason rejection")
	}
	if resp != "" {
		t.Fatalf("ProcessDirect(ROLLBACK_APPLY_FAIL missing reason) response = %q, want empty on rejection", resp)
	}
	if !strings.Contains(err.Error(), "terminal failure reason is required") {
		t.Fatalf("ProcessDirect(ROLLBACK_APPLY_FAIL missing reason) error = %q, want required reason rejection", err)
	}

	beforePointerBytes, err := os.ReadFile(missioncontrol.StoreActiveRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(active pointer before) error = %v", err)
	}
	resp, err = ag.ProcessDirect("ROLLBACK_APPLY_FAIL job-1 apply-1 operator requested stop after recovery review", 2*time.Second)
	if err == nil {
		t.Fatal("ProcessDirect(ROLLBACK_APPLY_FAIL) error = nil, want invalid phase rejection")
	}
	if resp != "" {
		t.Fatalf("ProcessDirect(ROLLBACK_APPLY_FAIL) response = %q, want empty on rejection", resp)
	}
	if !strings.Contains(err.Error(), `phase "recorded" does not permit terminal failure resolution`) {
		t.Fatalf("ProcessDirect(ROLLBACK_APPLY_FAIL) error = %q, want invalid phase rejection", err)
	}
	afterPointerBytes, err := os.ReadFile(missioncontrol.StoreActiveRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(active pointer after) error = %v", err)
	}
	if string(beforePointerBytes) != string(afterPointerBytes) {
		t.Fatalf("active runtime pack pointer file changed after invalid terminal failure rejection\nbefore:\n%s\nafter:\n%s", string(beforePointerBytes), string(afterPointerBytes))
	}
}

func TestProcessDirectRollbackApplyReloadCommandRejectsInvalidStartingPhase(t *testing.T) {
	t.Parallel()

	root, wantPointer := writeLoopRollbackPromotionFixtures(t)

	b := chat.NewHub(10)
	prov := &finalResponseProvider{content: "unused"}
	ag := NewAgentLoop(b, prov, prov.GetDefaultModel(), 3, "", nil)
	ag.SetMissionStoreRoot(root)
	if err := ag.ActivateMissionStep(testMissionJob([]string{"read"}, []string{"read"}), "build"); err != nil {
		t.Fatalf("ActivateMissionStep() error = %v", err)
	}

	if _, err := ag.ProcessDirect("ROLLBACK_RECORD job-1 promotion-1 rollback-1", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(ROLLBACK_RECORD) error = %v", err)
	}
	if _, err := ag.ProcessDirect("ROLLBACK_APPLY_RECORD job-1 rollback-1 apply-1", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(ROLLBACK_APPLY_RECORD) error = %v", err)
	}

	beforePointerBytes, err := os.ReadFile(missioncontrol.StoreActiveRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(active pointer before) error = %v", err)
	}

	resp, err := ag.ProcessDirect("ROLLBACK_APPLY_RELOAD job-1 apply-1", 2*time.Second)
	if err == nil {
		t.Fatal("ProcessDirect(ROLLBACK_APPLY_RELOAD) error = nil, want invalid phase rejection")
	}
	if resp != "" {
		t.Fatalf("ProcessDirect(ROLLBACK_APPLY_RELOAD) response = %q, want empty on rejection", resp)
	}
	if !strings.Contains(err.Error(), `phase "recorded" does not permit reload/apply execution`) {
		t.Fatalf("ProcessDirect(ROLLBACK_APPLY_RELOAD) error = %q, want invalid phase rejection", err)
	}

	record, err := missioncontrol.LoadRollbackApplyRecord(root, "apply-1")
	if err != nil {
		t.Fatalf("LoadRollbackApplyRecord() error = %v", err)
	}
	if record.Phase != missioncontrol.RollbackApplyPhaseRecorded {
		t.Fatalf("LoadRollbackApplyRecord().Phase = %q, want recorded after rejection", record.Phase)
	}

	gotPointer, err := missioncontrol.LoadActiveRuntimePackPointer(root)
	if err != nil {
		t.Fatalf("LoadActiveRuntimePackPointer() error = %v", err)
	}
	if gotPointer != wantPointer {
		t.Fatalf("LoadActiveRuntimePackPointer() = %#v, want %#v", gotPointer, wantPointer)
	}
	afterPointerBytes, err := os.ReadFile(missioncontrol.StoreActiveRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(active pointer after) error = %v", err)
	}
	if string(beforePointerBytes) != string(afterPointerBytes) {
		t.Fatalf("active runtime pack pointer file changed\nbefore:\n%s\nafter:\n%s", string(beforePointerBytes), string(afterPointerBytes))
	}
}

func TestProcessDirectInspectCommandReturnsDeterministicSummaryForActiveJob(t *testing.T) {
	t.Parallel()

	b := chat.NewHub(10)
	prov := &finalResponseProvider{content: "unused"}
	ag := NewAgentLoop(b, prov, prov.GetDefaultModel(), 3, "", nil)
	job := testMissionJob([]string{"write", "read", "search"}, []string{"read", "read"})
	if err := ag.ActivateMissionStep(job, "build"); err != nil {
		t.Fatalf("ActivateMissionStep() error = %v", err)
	}

	resp, err := ag.ProcessDirect("INSPECT job-1 build", 2*time.Second)
	if err != nil {
		t.Fatalf("ProcessDirect(INSPECT) error = %v", err)
	}

	var got missioncontrol.InspectSummary
	if err := json.Unmarshal([]byte(resp), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if got.JobID != "job-1" {
		t.Fatalf("JobID = %q, want %q", got.JobID, "job-1")
	}
	if len(got.Steps) != 1 {
		t.Fatalf("len(Steps) = %d, want 1", len(got.Steps))
	}
	if got.Steps[0].StepID != "build" {
		t.Fatalf("Steps[0].StepID = %q, want %q", got.Steps[0].StepID, "build")
	}
	if !reflect.DeepEqual(got.Steps[0].EffectiveAllowedTools, []string{"read"}) {
		t.Fatalf("Steps[0].EffectiveAllowedTools = %#v, want %#v", got.Steps[0].EffectiveAllowedTools, []string{"read"})
	}

	runtime, ok := ag.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if runtime.State != missioncontrol.JobStateRunning {
		t.Fatalf("MissionRuntimeState().State = %q, want %q", runtime.State, missioncontrol.JobStateRunning)
	}
}

func TestProcessDirectInspectCommandWrongJobDoesNotBind(t *testing.T) {
	t.Parallel()

	b := chat.NewHub(10)
	prov := &finalResponseProvider{content: "unused"}
	ag := NewAgentLoop(b, prov, prov.GetDefaultModel(), 3, "", nil)
	if err := ag.ActivateMissionStep(testMissionJob([]string{"read"}, []string{"read"}), "build"); err != nil {
		t.Fatalf("ActivateMissionStep() error = %v", err)
	}

	_, err := ag.ProcessDirect("INSPECT other-job build", 2*time.Second)
	if err == nil {
		t.Fatal("ProcessDirect(INSPECT other-job) error = nil, want mismatch failure")
	}
	if !strings.Contains(err.Error(), "does not match the active job") {
		t.Fatalf("ProcessDirect(INSPECT other-job) error = %q, want job mismatch", err)
	}
}

func TestProcessDirectInspectCommandRejectsUnknownStepDeterministically(t *testing.T) {
	t.Parallel()

	b := chat.NewHub(10)
	prov := &finalResponseProvider{content: "unused"}
	ag := NewAgentLoop(b, prov, prov.GetDefaultModel(), 3, "", nil)
	if err := ag.ActivateMissionStep(testMissionJob([]string{"read"}, []string{"read"}), "build"); err != nil {
		t.Fatalf("ActivateMissionStep() error = %v", err)
	}

	_, err := ag.ProcessDirect("INSPECT job-1 missing", 2*time.Second)
	if err == nil {
		t.Fatal("ProcessDirect(INSPECT missing) error = nil, want unknown-step failure")
	}
	if !strings.Contains(err.Error(), "E_INVALID_ACTION_FOR_STEP") {
		t.Fatalf("ProcessDirect(INSPECT missing) error = %q, want unknown_step code", err)
	}
	if !strings.Contains(err.Error(), `step "missing" not found in plan`) {
		t.Fatalf("ProcessDirect(INSPECT missing) error = %q, want missing-step message", err)
	}
}

func TestProcessDirectInspectCommandUsesValidatedPlanAfterRehydration(t *testing.T) {
	t.Parallel()

	b := chat.NewHub(10)
	prov := &finalResponseProvider{content: "unused"}
	ag := NewAgentLoop(b, prov, prov.GetDefaultModel(), 3, "", nil)
	job := testMissionJob([]string{"write", "read", "search"}, []string{"read", "read"})
	control, err := missioncontrol.BuildRuntimeControlContext(job, "build")
	if err != nil {
		t.Fatalf("BuildRuntimeControlContext() error = %v", err)
	}
	if err := ag.HydrateMissionRuntimeControl(job, missioncontrol.JobRuntimeState{
		JobID:        "job-1",
		State:        missioncontrol.JobStatePaused,
		ActiveStepID: "build",
		PausedReason: missioncontrol.RuntimePauseReasonOperatorCommand,
	}, &control); err != nil {
		t.Fatalf("HydrateMissionRuntimeControl() error = %v", err)
	}

	resp, err := ag.ProcessDirect("INSPECT job-1 final", 2*time.Second)
	if err != nil {
		t.Fatalf("ProcessDirect(INSPECT persisted) error = %v", err)
	}

	var got missioncontrol.InspectSummary
	if err := json.Unmarshal([]byte(resp), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if got.JobID != "job-1" {
		t.Fatalf("JobID = %q, want %q", got.JobID, "job-1")
	}
	if len(got.Steps) != 1 || got.Steps[0].StepID != "final" {
		t.Fatalf("Steps = %#v, want one final step", got.Steps)
	}
	if !reflect.DeepEqual(got.Steps[0].EffectiveAllowedTools, []string{"read"}) {
		t.Fatalf("Steps[0].EffectiveAllowedTools = %#v, want %#v", got.Steps[0].EffectiveAllowedTools, []string{"read"})
	}

	if _, ok := ag.ActiveMissionStep(); ok {
		t.Fatal("ActiveMissionStep() ok = true, want false for rehydrated persisted inspect path")
	}
}

func TestProcessDirectInspectCommandUsesValidatedPlanForTerminalRuntime(t *testing.T) {
	t.Parallel()

	b := chat.NewHub(10)
	prov := &finalResponseProvider{content: "unused"}
	ag := NewAgentLoop(b, prov, prov.GetDefaultModel(), 3, "", nil)
	job := testMissionJob([]string{"write", "read", "search"}, []string{"read", "read"})
	if err := ag.HydrateMissionRuntimeControl(job, missioncontrol.JobRuntimeState{
		JobID:       "job-1",
		State:       missioncontrol.JobStateCompleted,
		CompletedAt: time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC),
		CompletedSteps: []missioncontrol.RuntimeStepRecord{
			{StepID: "build", At: time.Date(2026, 3, 25, 11, 59, 0, 0, time.UTC)},
		},
	}, nil); err != nil {
		t.Fatalf("HydrateMissionRuntimeControl() error = %v", err)
	}

	resp, err := ag.ProcessDirect("INSPECT job-1 final", 2*time.Second)
	if err != nil {
		t.Fatalf("ProcessDirect(INSPECT terminal) error = %v", err)
	}

	var got missioncontrol.InspectSummary
	if err := json.Unmarshal([]byte(resp), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if got.JobID != "job-1" {
		t.Fatalf("JobID = %q, want %q", got.JobID, "job-1")
	}
	if len(got.Steps) != 1 || got.Steps[0].StepID != "final" {
		t.Fatalf("Steps = %#v, want one final step", got.Steps)
	}
	if !reflect.DeepEqual(got.Steps[0].EffectiveAllowedTools, []string{"read"}) {
		t.Fatalf("Steps[0].EffectiveAllowedTools = %#v, want %#v", got.Steps[0].EffectiveAllowedTools, []string{"read"})
	}
}

func TestProcessDirectSetStepCommandUsesOperatorHook(t *testing.T) {
	t.Parallel()

	b := chat.NewHub(10)
	prov := &finalResponseProvider{content: "unused"}
	ag := NewAgentLoop(b, prov, prov.GetDefaultModel(), 3, "", nil)

	var gotJobID string
	var gotStepID string
	ag.SetOperatorSetStepHook(func(jobID string, stepID string) (string, error) {
		gotJobID = jobID
		gotStepID = stepID
		return "Set step job=job-1 step=final.", nil
	})

	resp, err := ag.ProcessDirect("SET_STEP job-1 final", 2*time.Second)
	if err != nil {
		t.Fatalf("ProcessDirect(SET_STEP) error = %v", err)
	}
	if resp != "Set step job=job-1 step=final." {
		t.Fatalf("ProcessDirect(SET_STEP) response = %q, want set-step acknowledgement", resp)
	}
	if gotJobID != "job-1" || gotStepID != "final" {
		t.Fatalf("operator hook received job=%q step=%q, want job-1/final", gotJobID, gotStepID)
	}
}

func TestProcessDirectSetStepCommandWithoutHookRejectsDeterministically(t *testing.T) {
	t.Parallel()

	b := chat.NewHub(10)
	prov := &finalResponseProvider{content: "unused"}
	ag := NewAgentLoop(b, prov, prov.GetDefaultModel(), 3, "", nil)

	_, err := ag.ProcessDirect("SET_STEP job-1 final", 2*time.Second)
	if err == nil {
		t.Fatal("ProcessDirect(SET_STEP) error = nil, want deterministic rejection")
	}
	if !strings.Contains(err.Error(), "E_STEP_OUT_OF_ORDER") {
		t.Fatalf("ProcessDirect(SET_STEP) error = %q, want canonical rejection code", err)
	}
	if !strings.Contains(err.Error(), "SET_STEP requires mission step control configuration") {
		t.Fatalf("ProcessDirect(SET_STEP) error = %q, want configuration rejection", err)
	}
}

func TestProcessDirectResumeCommandUsesDurablePausedRuntimeAfterExecutionContextTeardown(t *testing.T) {
	t.Parallel()

	b := chat.NewHub(10)
	prov := &finalResponseProvider{content: "unused"}
	ag := NewAgentLoop(b, prov, prov.GetDefaultModel(), 3, "", nil)
	if err := ag.ActivateMissionStep(testMissionJob([]string{"read"}, []string{"read"}), "build"); err != nil {
		t.Fatalf("ActivateMissionStep() error = %v", err)
	}
	if _, err := ag.ProcessDirect("PAUSE job-1", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(PAUSE) error = %v", err)
	}

	ag.ClearMissionStep()

	if _, ok := ag.ActiveMissionStep(); ok {
		t.Fatal("ActiveMissionStep() ok = true, want false after teardown")
	}

	resp, err := ag.ProcessDirect("RESUME job-1", 2*time.Second)
	if err != nil {
		t.Fatalf("ProcessDirect(RESUME) error = %v", err)
	}
	if resp != "Resumed job=job-1." {
		t.Fatalf("ProcessDirect(RESUME) response = %q, want resume acknowledgement", resp)
	}

	ec, ok := ag.ActiveMissionStep()
	if !ok || ec.Runtime == nil {
		t.Fatalf("ActiveMissionStep() = %#v, want resumed active step", ec)
	}
	if ec.Runtime.State != missioncontrol.JobStateRunning {
		t.Fatalf("ActiveMissionStep().Runtime.State = %q, want %q", ec.Runtime.State, missioncontrol.JobStateRunning)
	}
	if ec.Runtime.ActiveStepID != "build" {
		t.Fatalf("ActiveMissionStep().Runtime.ActiveStepID = %q, want %q", ec.Runtime.ActiveStepID, "build")
	}
}

func TestProcessDirectApproveCommandCompletesPendingApprovalStep(t *testing.T) {
	t.Parallel()

	b := chat.NewHub(10)
	prov := &finalResponseProvider{content: "Need approval before continuing."}
	ag := NewAgentLoop(b, prov, prov.GetDefaultModel(), 3, "", nil)
	if err := ag.ActivateMissionStep(testDiscussionMissionJob(), "build"); err != nil {
		t.Fatalf("ActivateMissionStep() error = %v", err)
	}

	if _, err := ag.ProcessDirect("continue", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(continue) error = %v", err)
	}

	resp, err := ag.ProcessDirect("APPROVE job-1 build", 2*time.Second)
	if err != nil {
		t.Fatalf("ProcessDirect(APPROVE) error = %v", err)
	}
	if resp != "Approved job=job-1 step=build." {
		t.Fatalf("ProcessDirect(APPROVE) response = %q, want approval acknowledgement", resp)
	}

	if _, ok := ag.ActiveMissionStep(); ok {
		t.Fatal("ActiveMissionStep() ok = true, want false after approval completion")
	}
	runtime, ok := ag.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if runtime.State != missioncontrol.JobStatePaused {
		t.Fatalf("MissionRuntimeState().State = %q, want %q", runtime.State, missioncontrol.JobStatePaused)
	}
	if len(runtime.CompletedSteps) != 1 || runtime.CompletedSteps[0].StepID != "build" {
		t.Fatalf("MissionRuntimeState().CompletedSteps = %#v, want build completion", runtime.CompletedSteps)
	}
	if len(runtime.ApprovalGrants) != 1 || runtime.ApprovalGrants[0].State != missioncontrol.ApprovalStateGranted {
		t.Fatalf("MissionRuntimeState().ApprovalGrants = %#v, want one granted approval", runtime.ApprovalGrants)
	}
	if runtime.ApprovalRequests[0].SessionChannel != "cli" || runtime.ApprovalRequests[0].SessionChatID != "direct" {
		t.Fatalf("MissionRuntimeState().ApprovalRequests[0] session = (%q, %q), want (%q, %q)", runtime.ApprovalRequests[0].SessionChannel, runtime.ApprovalRequests[0].SessionChatID, "cli", "direct")
	}
	if runtime.ApprovalGrants[0].SessionChannel != "cli" || runtime.ApprovalGrants[0].SessionChatID != "direct" {
		t.Fatalf("MissionRuntimeState().ApprovalGrants[0] session = (%q, %q), want (%q, %q)", runtime.ApprovalGrants[0].SessionChannel, runtime.ApprovalGrants[0].SessionChatID, "cli", "direct")
	}
}

func TestProcessDirectYesApprovesSinglePendingApprovalStep(t *testing.T) {
	t.Parallel()

	b := chat.NewHub(10)
	prov := &finalResponseProvider{content: "Need approval before continuing."}
	ag := NewAgentLoop(b, prov, prov.GetDefaultModel(), 3, "", nil)
	if err := ag.ActivateMissionStep(testDiscussionMissionJob(), "build"); err != nil {
		t.Fatalf("ActivateMissionStep() error = %v", err)
	}
	if _, err := ag.ProcessDirect("continue", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(continue) error = %v", err)
	}

	resp, err := ag.ProcessDirect("yes", 2*time.Second)
	if err != nil {
		t.Fatalf("ProcessDirect(yes) error = %v", err)
	}
	if resp != "Approved job=job-1 step=build." {
		t.Fatalf("ProcessDirect(yes) response = %q, want approval acknowledgement", resp)
	}

	runtime, ok := ag.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if len(runtime.ApprovalRequests) != 1 || len(runtime.ApprovalGrants) != 1 {
		t.Fatalf("MissionRuntimeState() approvals = requests=%#v grants=%#v, want one bound request and grant", runtime.ApprovalRequests, runtime.ApprovalGrants)
	}
	if runtime.ApprovalRequests[0].SessionChannel != "cli" || runtime.ApprovalRequests[0].SessionChatID != "direct" {
		t.Fatalf("MissionRuntimeState().ApprovalRequests[0] session = (%q, %q), want (%q, %q)", runtime.ApprovalRequests[0].SessionChannel, runtime.ApprovalRequests[0].SessionChatID, "cli", "direct")
	}
	if runtime.ApprovalGrants[0].SessionChannel != "cli" || runtime.ApprovalGrants[0].SessionChatID != "direct" {
		t.Fatalf("MissionRuntimeState().ApprovalGrants[0] session = (%q, %q), want (%q, %q)", runtime.ApprovalGrants[0].SessionChannel, runtime.ApprovalGrants[0].SessionChatID, "cli", "direct")
	}
}

func TestProcessDirectNoDeniesSinglePendingApprovalStep(t *testing.T) {
	t.Parallel()

	b := chat.NewHub(10)
	prov := &finalResponseProvider{content: "Need approval before continuing."}
	ag := NewAgentLoop(b, prov, prov.GetDefaultModel(), 3, "", nil)
	if err := ag.ActivateMissionStep(testDiscussionMissionJob(), "build"); err != nil {
		t.Fatalf("ActivateMissionStep() error = %v", err)
	}
	if _, err := ag.ProcessDirect("continue", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(continue) error = %v", err)
	}

	resp, err := ag.ProcessDirect("no", 2*time.Second)
	if err != nil {
		t.Fatalf("ProcessDirect(no) error = %v", err)
	}
	if resp != "Denied job=job-1 step=build." {
		t.Fatalf("ProcessDirect(no) response = %q, want denial acknowledgement", resp)
	}

	runtime, ok := ag.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if len(runtime.ApprovalRequests) != 1 || runtime.ApprovalRequests[0].State != missioncontrol.ApprovalStateDenied {
		t.Fatalf("MissionRuntimeState().ApprovalRequests = %#v, want one denied request", runtime.ApprovalRequests)
	}
	if runtime.ApprovalRequests[0].SessionChannel != "cli" || runtime.ApprovalRequests[0].SessionChatID != "direct" {
		t.Fatalf("MissionRuntimeState().ApprovalRequests[0] session = (%q, %q), want (%q, %q)", runtime.ApprovalRequests[0].SessionChannel, runtime.ApprovalRequests[0].SessionChatID, "cli", "direct")
	}
}

func TestProcessDirectPauseCommandWrongJobDoesNotBind(t *testing.T) {
	t.Parallel()

	b := chat.NewHub(10)
	prov := &finalResponseProvider{content: "unused"}
	ag := NewAgentLoop(b, prov, prov.GetDefaultModel(), 3, "", nil)
	if err := ag.ActivateMissionStep(testMissionJob([]string{"read"}, []string{"read"}), "build"); err != nil {
		t.Fatalf("ActivateMissionStep() error = %v", err)
	}

	_, err := ag.ProcessDirect("PAUSE other-job", 2*time.Second)
	if err == nil {
		t.Fatal("ProcessDirect(PAUSE wrong job) error = nil, want mismatch failure")
	}

	ec, ok := ag.ActiveMissionStep()
	if !ok || ec.Runtime == nil {
		t.Fatalf("ActiveMissionStep() = %#v, want unchanged active step", ec)
	}
	if ec.Runtime.State != missioncontrol.JobStateRunning {
		t.Fatalf("ActiveMissionStep().Runtime.State = %q, want %q", ec.Runtime.State, missioncontrol.JobStateRunning)
	}

	audits := ag.taskState.AuditEvents()
	if len(audits) != 1 {
		t.Fatalf("AuditEvents() count = %d, want %d", len(audits), 1)
	}
	assertAuditEvent(t, audits[0], "job-1", "build", "pause", false, missioncontrol.RejectionCode("E_VALIDATION_FAILED"))
}

func TestProcessDirectDenyCommandKeepsPendingApprovalStepWaiting(t *testing.T) {
	t.Parallel()

	b := chat.NewHub(10)
	prov := &finalResponseProvider{content: "Need approval before continuing."}
	ag := NewAgentLoop(b, prov, prov.GetDefaultModel(), 3, "", nil)
	if err := ag.ActivateMissionStep(testDiscussionMissionJob(), "build"); err != nil {
		t.Fatalf("ActivateMissionStep() error = %v", err)
	}

	if _, err := ag.ProcessDirect("continue", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(continue) error = %v", err)
	}

	resp, err := ag.ProcessDirect("DENY job-1 build", 2*time.Second)
	if err != nil {
		t.Fatalf("ProcessDirect(DENY) error = %v", err)
	}
	if resp != "Denied job=job-1 step=build." {
		t.Fatalf("ProcessDirect(DENY) response = %q, want denial acknowledgement", resp)
	}

	ec, ok := ag.ActiveMissionStep()
	if !ok {
		t.Fatal("ActiveMissionStep() ok = false, want waiting step")
	}
	if ec.Runtime == nil || ec.Runtime.State != missioncontrol.JobStateWaitingUser {
		t.Fatalf("ActiveMissionStep().Runtime = %#v, want waiting_user runtime", ec.Runtime)
	}
	if len(ec.Runtime.ApprovalRequests) != 1 || ec.Runtime.ApprovalRequests[0].State != missioncontrol.ApprovalStateDenied {
		t.Fatalf("ActiveMissionStep().Runtime.ApprovalRequests = %#v, want one denied approval", ec.Runtime.ApprovalRequests)
	}
}

func TestProcessDirectApproveCommandUsesPersistedWaitingRuntimeAfterExecutionContextTeardown(t *testing.T) {
	t.Parallel()

	b := chat.NewHub(10)
	prov := &finalResponseProvider{content: "Need approval before continuing."}
	ag := NewAgentLoop(b, prov, prov.GetDefaultModel(), 3, "", nil)
	if err := ag.ActivateMissionStep(testDiscussionMissionJob(), "build"); err != nil {
		t.Fatalf("ActivateMissionStep() error = %v", err)
	}

	if _, err := ag.ProcessDirect("continue", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(continue) error = %v", err)
	}

	ag.ClearMissionStep()

	resp, err := ag.ProcessDirect("APPROVE job-1 build", 2*time.Second)
	if err != nil {
		t.Fatalf("ProcessDirect(APPROVE) error = %v", err)
	}
	if resp != "Approved job=job-1 step=build." {
		t.Fatalf("ProcessDirect(APPROVE) response = %q, want approval acknowledgement", resp)
	}

	if _, ok := ag.ActiveMissionStep(); ok {
		t.Fatal("ActiveMissionStep() ok = true, want false after reboot-safe approval completion")
	}
	runtime, ok := ag.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if runtime.State != missioncontrol.JobStatePaused {
		t.Fatalf("MissionRuntimeState().State = %q, want %q", runtime.State, missioncontrol.JobStatePaused)
	}
	if len(runtime.CompletedSteps) != 1 || runtime.CompletedSteps[0].StepID != "build" {
		t.Fatalf("MissionRuntimeState().CompletedSteps = %#v, want build completion", runtime.CompletedSteps)
	}
}

func TestProcessDirectApproveCommandClearsPendingApprovalBudgetPause(t *testing.T) {
	t.Parallel()

	b := chat.NewHub(10)
	prov := &finalResponseProvider{content: "unused"}
	ag := NewAgentLoop(b, prov, prov.GetDefaultModel(), 3, "", nil)
	job, runtimeState, control := testPendingApprovalBudgetPausedDiscussionRuntime(t)
	if err := ag.HydrateMissionRuntimeControl(job, runtimeState, &control); err != nil {
		t.Fatalf("HydrateMissionRuntimeControl() error = %v", err)
	}

	resp, err := ag.ProcessDirect("APPROVE job-1 build", 2*time.Second)
	if err != nil {
		t.Fatalf("ProcessDirect(APPROVE) error = %v", err)
	}
	if resp != "Approved job=job-1 step=build." {
		t.Fatalf("ProcessDirect(APPROVE) response = %q, want approval acknowledgement", resp)
	}

	if _, ok := ag.ActiveMissionStep(); ok {
		t.Fatal("ActiveMissionStep() ok = true, want false after approval completion")
	}
	runtime, ok := ag.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if runtime.State != missioncontrol.JobStatePaused {
		t.Fatalf("MissionRuntimeState().State = %q, want %q", runtime.State, missioncontrol.JobStatePaused)
	}
	if runtime.PausedReason != missioncontrol.RuntimePauseReasonStepComplete {
		t.Fatalf("MissionRuntimeState().PausedReason = %q, want %q", runtime.PausedReason, missioncontrol.RuntimePauseReasonStepComplete)
	}
	if runtime.BudgetBlocker != nil {
		t.Fatalf("MissionRuntimeState().BudgetBlocker = %#v, want nil after approval completion", runtime.BudgetBlocker)
	}
	if len(runtime.CompletedSteps) != 1 || runtime.CompletedSteps[0].StepID != "build" {
		t.Fatalf("MissionRuntimeState().CompletedSteps = %#v, want build completion", runtime.CompletedSteps)
	}
	lastRequest := runtime.ApprovalRequests[len(runtime.ApprovalRequests)-1]
	if lastRequest.StepID != "build" || lastRequest.State != missioncontrol.ApprovalStateGranted {
		t.Fatalf("latest ApprovalRequest = %#v, want granted build approval", lastRequest)
	}
}

func TestProcessDirectDenyCommandClearsPendingApprovalBudgetPause(t *testing.T) {
	t.Parallel()

	b := chat.NewHub(10)
	prov := &finalResponseProvider{content: "unused"}
	ag := NewAgentLoop(b, prov, prov.GetDefaultModel(), 3, "", nil)
	job, runtimeState, control := testPendingApprovalBudgetPausedDiscussionRuntime(t)
	if err := ag.HydrateMissionRuntimeControl(job, runtimeState, &control); err != nil {
		t.Fatalf("HydrateMissionRuntimeControl() error = %v", err)
	}

	resp, err := ag.ProcessDirect("DENY job-1 build", 2*time.Second)
	if err != nil {
		t.Fatalf("ProcessDirect(DENY) error = %v", err)
	}
	if resp != "Denied job=job-1 step=build." {
		t.Fatalf("ProcessDirect(DENY) response = %q, want denial acknowledgement", resp)
	}

	if _, ok := ag.ActiveMissionStep(); ok {
		t.Fatal("ActiveMissionStep() ok = true, want persisted waiting runtime after denial")
	}

	runtime, ok := ag.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if runtime.State != missioncontrol.JobStateWaitingUser {
		t.Fatalf("MissionRuntimeState().State = %q, want %q", runtime.State, missioncontrol.JobStateWaitingUser)
	}
	if runtime.WaitingReason != "discussion_authorization" {
		t.Fatalf("MissionRuntimeState().WaitingReason = %q, want %q", runtime.WaitingReason, "discussion_authorization")
	}
	if runtime.BudgetBlocker != nil {
		t.Fatalf("MissionRuntimeState().BudgetBlocker = %#v, want nil after denial", runtime.BudgetBlocker)
	}
	lastRequest := runtime.ApprovalRequests[len(runtime.ApprovalRequests)-1]
	if lastRequest.StepID != "build" || lastRequest.State != missioncontrol.ApprovalStateDenied {
		t.Fatalf("latest ApprovalRequest = %#v, want denied build approval", lastRequest)
	}
}

func TestProcessDirectRevokeApprovalCommandRevokesMatchingGrant(t *testing.T) {
	t.Parallel()

	b := chat.NewHub(10)
	prov := &finalResponseProvider{content: "unused"}
	ag := NewAgentLoop(b, prov, prov.GetDefaultModel(), 3, "", nil)
	job := testReusableApprovalMissionJob(missioncontrol.ApprovalScopeOneJob)
	now := time.Now().UTC()
	runtimeState := missioncontrol.JobRuntimeState{
		JobID:        job.ID,
		State:        missioncontrol.JobStateRunning,
		ActiveStepID: "authorize-2",
		ApprovalRequests: []missioncontrol.ApprovalRequest{
			{
				JobID:           job.ID,
				StepID:          "authorize-1",
				RequestedAction: missioncontrol.ApprovalRequestedActionStepComplete,
				Scope:           missioncontrol.ApprovalScopeOneJob,
				RequestedVia:    missioncontrol.ApprovalRequestedViaRuntime,
				GrantedVia:      missioncontrol.ApprovalGrantedViaOperatorCommand,
				State:           missioncontrol.ApprovalStateGranted,
				RequestedAt:     now.Add(-2 * time.Minute),
				ResolvedAt:      now.Add(-90 * time.Second),
			},
		},
		ApprovalGrants: []missioncontrol.ApprovalGrant{
			{
				JobID:           job.ID,
				StepID:          "authorize-1",
				RequestedAction: missioncontrol.ApprovalRequestedActionStepComplete,
				Scope:           missioncontrol.ApprovalScopeOneJob,
				GrantedVia:      missioncontrol.ApprovalGrantedViaOperatorCommand,
				State:           missioncontrol.ApprovalStateGranted,
				GrantedAt:       now.Add(-90 * time.Second),
				ExpiresAt:       now.Add(time.Minute),
			},
		},
	}
	if err := ag.taskState.ResumeRuntime(job, runtimeState, nil, true); err != nil {
		t.Fatalf("ResumeRuntime() error = %v", err)
	}

	resp, err := ag.ProcessDirect("REVOKE_APPROVAL job-1 authorize-2", 2*time.Second)
	if err != nil {
		t.Fatalf("ProcessDirect(REVOKE_APPROVAL) error = %v", err)
	}
	if resp != "Revoked approval job=job-1 step=authorize-2." {
		t.Fatalf("ProcessDirect(REVOKE_APPROVAL) response = %q, want revoke acknowledgement", resp)
	}

	runtime, ok := ag.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if len(runtime.ApprovalRequests) != 1 || runtime.ApprovalRequests[0].State != missioncontrol.ApprovalStateRevoked {
		t.Fatalf("MissionRuntimeState().ApprovalRequests = %#v, want one revoked approval request", runtime.ApprovalRequests)
	}
	if runtime.ApprovalRequests[0].RevokedAt.IsZero() {
		t.Fatalf("MissionRuntimeState().ApprovalRequests[0].RevokedAt = %v, want stamped revoke time", runtime.ApprovalRequests[0].RevokedAt)
	}
	if len(runtime.ApprovalGrants) != 1 || runtime.ApprovalGrants[0].State != missioncontrol.ApprovalStateRevoked {
		t.Fatalf("MissionRuntimeState().ApprovalGrants = %#v, want one revoked approval grant", runtime.ApprovalGrants)
	}
	if runtime.ApprovalGrants[0].RevokedAt.IsZero() {
		t.Fatalf("MissionRuntimeState().ApprovalGrants[0].RevokedAt = %v, want stamped revoke time", runtime.ApprovalGrants[0].RevokedAt)
	}
}

func TestProcessDirectNaturalApprovalRejectsAmbiguousPendingRequests(t *testing.T) {
	t.Parallel()

	b := chat.NewHub(10)
	prov := &finalResponseProvider{content: "Need approval before continuing."}
	ag := NewAgentLoop(b, prov, prov.GetDefaultModel(), 3, "", nil)
	if err := ag.ActivateMissionStep(testDiscussionMissionJob(), "build"); err != nil {
		t.Fatalf("ActivateMissionStep() error = %v", err)
	}
	if _, err := ag.ProcessDirect("continue", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(continue) error = %v", err)
	}

	ec, ok := ag.ActiveMissionStep()
	if !ok {
		t.Fatal("ActiveMissionStep() ok = false, want waiting step")
	}
	if ec.Runtime == nil {
		t.Fatal("ActiveMissionStep().Runtime = nil, want waiting_user runtime")
	}
	ec.Runtime.ApprovalRequests = append(ec.Runtime.ApprovalRequests, missioncontrol.ApprovalRequest{
		JobID:           "job-1",
		StepID:          "other-step",
		RequestedAction: missioncontrol.ApprovalRequestedActionStepComplete,
		Scope:           missioncontrol.ApprovalScopeMissionStep,
		State:           missioncontrol.ApprovalStatePending,
	})
	ag.taskState.SetExecutionContext(ec)

	_, err := ag.ProcessDirect("yes", 2*time.Second)
	if err == nil {
		t.Fatal("ProcessDirect(yes) error = nil, want ambiguity failure")
	}
}

func TestProcessDirectDenyCommandUsesPersistedWaitingRuntimeAfterExecutionContextTeardown(t *testing.T) {
	t.Parallel()

	b := chat.NewHub(10)
	prov := &finalResponseProvider{content: "Need approval before continuing."}
	ag := NewAgentLoop(b, prov, prov.GetDefaultModel(), 3, "", nil)
	if err := ag.ActivateMissionStep(testDiscussionMissionJob(), "build"); err != nil {
		t.Fatalf("ActivateMissionStep() error = %v", err)
	}

	if _, err := ag.ProcessDirect("continue", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(continue) error = %v", err)
	}

	ag.ClearMissionStep()

	resp, err := ag.ProcessDirect("DENY job-1 build", 2*time.Second)
	if err != nil {
		t.Fatalf("ProcessDirect(DENY) error = %v", err)
	}
	if resp != "Denied job=job-1 step=build." {
		t.Fatalf("ProcessDirect(DENY) response = %q, want denial acknowledgement", resp)
	}

	if _, ok := ag.ActiveMissionStep(); ok {
		t.Fatal("ActiveMissionStep() ok = true, want false after reboot-safe denial")
	}
	runtime, ok := ag.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if runtime.State != missioncontrol.JobStateWaitingUser {
		t.Fatalf("MissionRuntimeState().State = %q, want %q", runtime.State, missioncontrol.JobStateWaitingUser)
	}
	if len(runtime.ApprovalRequests) != 1 || runtime.ApprovalRequests[0].State != missioncontrol.ApprovalStateDenied {
		t.Fatalf("MissionRuntimeState().ApprovalRequests = %#v, want one denied approval", runtime.ApprovalRequests)
	}
}

func TestProcessDirectNaturalApprovalUsesPersistedWaitingRuntimeAfterExecutionContextTeardown(t *testing.T) {
	t.Parallel()

	b := chat.NewHub(10)
	prov := &finalResponseProvider{content: "Need approval before continuing."}
	ag := NewAgentLoop(b, prov, prov.GetDefaultModel(), 3, "", nil)
	if err := ag.ActivateMissionStep(testDiscussionMissionJob(), "build"); err != nil {
		t.Fatalf("ActivateMissionStep() error = %v", err)
	}
	if _, err := ag.ProcessDirect("continue", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(continue) error = %v", err)
	}

	ag.ClearMissionStep()

	resp, err := ag.ProcessDirect("yes", 2*time.Second)
	if err != nil {
		t.Fatalf("ProcessDirect(yes) error = %v", err)
	}
	if resp != "Approved job=job-1 step=build." {
		t.Fatalf("ProcessDirect(yes) response = %q, want approval acknowledgement", resp)
	}
}

func TestProcessDirectYesDoesNotBindExpiredPendingApprovalStep(t *testing.T) {
	t.Parallel()

	b := chat.NewHub(10)
	prov := &finalResponseProvider{content: "(stub) Echo: yes"}
	ag := NewAgentLoop(b, prov, prov.GetDefaultModel(), 3, "", nil)
	if err := ag.ActivateMissionStep(testDiscussionMissionJob(), "build"); err != nil {
		t.Fatalf("ActivateMissionStep() error = %v", err)
	}
	if _, err := ag.ProcessDirect("continue", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(continue) error = %v", err)
	}

	ec, ok := ag.ActiveMissionStep()
	if !ok || ec.Runtime == nil {
		t.Fatalf("ActiveMissionStep() = %#v, want waiting runtime", ec)
	}
	if ec.Runtime.ApprovalRequests[0].ExpiresAt.IsZero() {
		t.Fatalf("ActiveMissionStep().Runtime.ApprovalRequests = %#v, want stamped expires_at", ec.Runtime.ApprovalRequests)
	}
	expiredAt := time.Now().Add(-1 * time.Minute)
	ec.Runtime.ApprovalRequests[0].ExpiresAt = expiredAt
	ag.taskState.SetExecutionContext(ec)

	resp, err := ag.ProcessDirect("yes", 2*time.Second)
	if err != nil {
		t.Fatalf("ProcessDirect(yes) error = %v", err)
	}
	if resp != "(stub) Echo: yes" {
		t.Fatalf("ProcessDirect(yes) response = %q, want provider fallback", resp)
	}

	runtime, ok := ag.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if runtime.ApprovalRequests[0].State != missioncontrol.ApprovalStateExpired {
		t.Fatalf("MissionRuntimeState().ApprovalRequests = %#v, want expired approval", runtime.ApprovalRequests)
	}
}

func TestProcessDirectApproveCommandBindsOnlyLatestValidRequest(t *testing.T) {
	t.Parallel()

	b := chat.NewHub(10)
	prov := &finalResponseProvider{content: "Need approval before continuing."}
	ag := NewAgentLoop(b, prov, prov.GetDefaultModel(), 3, "", nil)
	if err := ag.ActivateMissionStep(testDiscussionMissionJob(), "build"); err != nil {
		t.Fatalf("ActivateMissionStep() error = %v", err)
	}
	if _, err := ag.ProcessDirect("continue", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(continue) error = %v", err)
	}

	ec, ok := ag.ActiveMissionStep()
	if !ok || ec.Runtime == nil {
		t.Fatalf("ActiveMissionStep() = %#v, want waiting runtime", ec)
	}
	now := time.Now()
	ec.Runtime.ApprovalRequests[0].State = missioncontrol.ApprovalStateSuperseded
	ec.Runtime.ApprovalRequests[0].SupersededAt = now
	ec.Runtime.ApprovalRequests = append(ec.Runtime.ApprovalRequests, missioncontrol.ApprovalRequest{
		JobID:           "job-1",
		StepID:          "build",
		RequestedAction: missioncontrol.ApprovalRequestedActionStepComplete,
		Scope:           missioncontrol.ApprovalScopeMissionStep,
		State:           missioncontrol.ApprovalStatePending,
		RequestedAt:     now.Add(time.Second),
	})
	ag.taskState.SetExecutionContext(ec)

	resp, err := ag.ProcessDirect("APPROVE job-1 build", 2*time.Second)
	if err != nil {
		t.Fatalf("ProcessDirect(APPROVE) error = %v", err)
	}
	if resp != "Approved job=job-1 step=build." {
		t.Fatalf("ProcessDirect(APPROVE) response = %q, want approval acknowledgement", resp)
	}
}

func TestProcessDirectResumeCommandDoesNotBypassWaitingApproval(t *testing.T) {
	t.Parallel()

	b := chat.NewHub(10)
	prov := &finalResponseProvider{content: "Need approval before continuing."}
	ag := NewAgentLoop(b, prov, prov.GetDefaultModel(), 3, "", nil)
	if err := ag.ActivateMissionStep(testDiscussionMissionJob(), "build"); err != nil {
		t.Fatalf("ActivateMissionStep() error = %v", err)
	}

	if _, err := ag.ProcessDirect("continue", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(continue) error = %v", err)
	}

	_, err := ag.ProcessDirect("RESUME job-1", 2*time.Second)
	if err == nil {
		t.Fatal("ProcessDirect(RESUME) error = nil, want waiting_user failure")
	}

	ec, ok := ag.ActiveMissionStep()
	if !ok || ec.Runtime == nil {
		t.Fatalf("ActiveMissionStep() = %#v, want waiting active step", ec)
	}
	if ec.Runtime.State != missioncontrol.JobStateWaitingUser {
		t.Fatalf("ActiveMissionStep().Runtime.State = %q, want %q", ec.Runtime.State, missioncontrol.JobStateWaitingUser)
	}
	if len(ec.Runtime.ApprovalRequests) != 1 || ec.Runtime.ApprovalRequests[0].State != missioncontrol.ApprovalStatePending {
		t.Fatalf("ActiveMissionStep().Runtime.ApprovalRequests = %#v, want one pending approval", ec.Runtime.ApprovalRequests)
	}
}

func TestProcessDirectAbortCommandUsesDurableWaitingRuntimeAfterExecutionContextTeardown(t *testing.T) {
	t.Parallel()

	b := chat.NewHub(10)
	prov := &finalResponseProvider{content: "Need approval before continuing."}
	ag := NewAgentLoop(b, prov, prov.GetDefaultModel(), 3, "", nil)
	if err := ag.ActivateMissionStep(testDiscussionMissionJob(), "build"); err != nil {
		t.Fatalf("ActivateMissionStep() error = %v", err)
	}

	if _, err := ag.ProcessDirect("continue", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(continue) error = %v", err)
	}

	ag.ClearMissionStep()

	if _, ok := ag.ActiveMissionStep(); ok {
		t.Fatal("ActiveMissionStep() ok = true, want false after teardown")
	}

	resp, err := ag.ProcessDirect("ABORT job-1", 2*time.Second)
	if err != nil {
		t.Fatalf("ProcessDirect(ABORT) error = %v", err)
	}
	if resp != "Aborted job=job-1." {
		t.Fatalf("ProcessDirect(ABORT) response = %q, want abort acknowledgement", resp)
	}

	if _, ok := ag.ActiveMissionStep(); ok {
		t.Fatal("ActiveMissionStep() ok = true, want false after abort")
	}
	runtime, ok := ag.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if runtime.State != missioncontrol.JobStateAborted {
		t.Fatalf("MissionRuntimeState().State = %q, want %q", runtime.State, missioncontrol.JobStateAborted)
	}
}

func TestProcessDirectFinalResponseFalseCompletionClaimLeavesRuntimeRunning(t *testing.T) {
	t.Parallel()

	b := chat.NewHub(10)
	prov := &finalResponseProvider{content: "Done"}
	ag := NewAgentLoop(b, prov, prov.GetDefaultModel(), 3, "", nil)
	if err := ag.ActivateMissionStep(testFinalMissionJob(), "final"); err != nil {
		t.Fatalf("ActivateMissionStep() error = %v", err)
	}

	resp, err := ag.ProcessDirect("finish", 2*time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp != "Done" {
		t.Fatalf("ProcessDirect() response = %q, want %q", resp, "Done")
	}

	ec, ok := ag.ActiveMissionStep()
	if !ok {
		t.Fatal("ActiveMissionStep() ok = false, want true")
	}
	if ec.Step == nil || ec.Step.ID != "final" {
		t.Fatalf("ActiveMissionStep().Step = %#v, want final step", ec.Step)
	}
	if ec.Runtime == nil {
		t.Fatal("ActiveMissionStep().Runtime = nil, want non-nil")
	}
	if ec.Runtime.State != missioncontrol.JobStateRunning {
		t.Fatalf("ActiveMissionStep().Runtime.State = %q, want %q", ec.Runtime.State, missioncontrol.JobStateRunning)
	}
}

func TestProcessDirectFinalResponseReturnsBudgetBlockerWhenUnattendedWallClockIsExhausted(t *testing.T) {
	t.Parallel()

	b := chat.NewHub(10)
	prov := &finalResponseProvider{content: "Here is the final answer."}
	ag := NewAgentLoop(b, prov, prov.GetDefaultModel(), 3, "", nil)
	job := testFinalMissionJob()
	now := time.Now().UTC()
	if err := ag.ActivateMissionStep(job, "final"); err != nil {
		t.Fatalf("ActivateMissionStep() error = %v", err)
	}

	ec, ok := ag.ActiveMissionStep()
	if !ok || ec.Runtime == nil {
		t.Fatalf("ActiveMissionStep() = (%#v, %t), want active runtime", ec, ok)
	}
	ec.Runtime.CreatedAt = now.Add(-5 * time.Hour)
	ec.Runtime.UpdatedAt = now.Add(-1 * time.Minute)
	ec.Runtime.StartedAt = now.Add(-5 * time.Hour)
	ec.Runtime.ActiveStepAt = now.Add(-1 * time.Minute)
	ec.Runtime.CompletedSteps = []missioncontrol.RuntimeStepRecord{
		{StepID: "build", At: now.Add(-2 * time.Hour)},
	}
	ag.taskState.SetExecutionContext(ec)

	resp, err := ag.ProcessDirect("finish", 2*time.Second)
	if err != nil {
		t.Fatalf("ProcessDirect() error = %v", err)
	}
	if resp != "Mission paused: unattended wall-clock budget exhausted." {
		t.Fatalf("ProcessDirect() response = %q, want budget pause response", resp)
	}

	runtime, ok := ag.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if runtime.State != missioncontrol.JobStatePaused {
		t.Fatalf("MissionRuntimeState().State = %q, want %q", runtime.State, missioncontrol.JobStatePaused)
	}
	if runtime.BudgetBlocker == nil || runtime.BudgetBlocker.Ceiling != "unattended_wall_clock" {
		t.Fatalf("MissionRuntimeState().BudgetBlocker = %#v, want unattended_wall_clock blocker", runtime.BudgetBlocker)
	}
	if len(runtime.CompletedSteps) != 1 || runtime.CompletedSteps[0].StepID != "build" {
		t.Fatalf("MissionRuntimeState().CompletedSteps = %#v, want only preexisting build completion", runtime.CompletedSteps)
	}
}

func TestProcessDirectDoesNotExecuteToolAfterUnattendedWallClockBudgetIsExhausted(t *testing.T) {
	t.Parallel()

	b := chat.NewHub(10)
	prov := &writeMemoryToolCallProvider{}
	ag := NewAgentLoop(b, prov, prov.GetDefaultModel(), 3, t.TempDir(), nil)
	job := testMissionJob([]string{"write_memory"}, []string{"write_memory"})
	now := time.Now().UTC()
	if err := ag.ActivateMissionStep(job, "build"); err != nil {
		t.Fatalf("ActivateMissionStep() error = %v", err)
	}

	ec, ok := ag.ActiveMissionStep()
	if !ok || ec.Runtime == nil {
		t.Fatalf("ActiveMissionStep() = (%#v, %t), want active runtime", ec, ok)
	}
	ec.Runtime.CreatedAt = now.Add(-5 * time.Hour)
	ec.Runtime.UpdatedAt = now.Add(-1 * time.Minute)
	ec.Runtime.StartedAt = now.Add(-5 * time.Hour)
	ec.Runtime.ActiveStepAt = now.Add(-1 * time.Minute)
	ag.taskState.SetExecutionContext(ec)

	resp, err := ag.ProcessDirect("keep going", 2*time.Second)
	if err != nil {
		t.Fatalf("ProcessDirect() error = %v", err)
	}
	if resp != "Mission paused: unattended wall-clock budget exhausted." {
		t.Fatalf("ProcessDirect() response = %q, want budget pause response", resp)
	}

	td, readErr := ag.memory.ReadToday()
	if readErr != nil {
		t.Fatalf("ReadToday() error = %v", readErr)
	}
	if strings.Contains(td, "budget overrun") {
		t.Fatalf("today memory = %q, want write_memory tool not to run", td)
	}

	runtime, ok := ag.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if runtime.State != missioncontrol.JobStatePaused {
		t.Fatalf("MissionRuntimeState().State = %q, want %q", runtime.State, missioncontrol.JobStatePaused)
	}
	if runtime.BudgetBlocker == nil || runtime.BudgetBlocker.Ceiling != "unattended_wall_clock" {
		t.Fatalf("MissionRuntimeState().BudgetBlocker = %#v, want unattended_wall_clock blocker", runtime.BudgetBlocker)
	}
}

func TestProcessDirectPausesAfterFailedActionBudgetIsExhausted(t *testing.T) {
	t.Parallel()

	b := chat.NewHub(10)
	prov := &repeatedFailingMessageToolProvider{}
	ag := NewAgentLoop(b, prov, prov.GetDefaultModel(), 3, t.TempDir(), nil)
	job := testMissionJob([]string{"message"}, []string{"message"})
	if err := ag.ActivateMissionStep(job, "build"); err != nil {
		t.Fatalf("ActivateMissionStep() error = %v", err)
	}

	resp, err := ag.ProcessDirect("keep going", 2*time.Second)
	if err != nil {
		t.Fatalf("ProcessDirect() error = %v", err)
	}
	if resp != "Mission paused: failed action budget exhausted." {
		t.Fatalf("ProcessDirect() response = %q, want failed-action budget response", resp)
	}

	select {
	case out := <-b.Out:
		t.Fatalf("unexpected outbound message despite failing message tool: %#v", out)
	default:
	}

	runtime, ok := ag.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if runtime.State != missioncontrol.JobStatePaused {
		t.Fatalf("MissionRuntimeState().State = %q, want %q", runtime.State, missioncontrol.JobStatePaused)
	}
	if runtime.BudgetBlocker == nil || runtime.BudgetBlocker.Ceiling != "failed_actions" {
		t.Fatalf("MissionRuntimeState().BudgetBlocker = %#v, want failed_actions blocker", runtime.BudgetBlocker)
	}
	rejectedFailures := 0
	for _, event := range runtime.AuditHistory {
		if event.ActionClass == missioncontrol.AuditActionClassToolCall && event.Allowed && event.Result == missioncontrol.AuditResultRejected {
			rejectedFailures++
		}
	}
	if rejectedFailures != 5 {
		t.Fatalf("MissionRuntimeState().failed rejected tool actions = %d, want 5", rejectedFailures)
	}
	if got := runtime.AuditHistory[len(runtime.AuditHistory)-1].ToolName; got != "budget_exhausted" {
		t.Fatalf("MissionRuntimeState().last audit tool = %q, want budget_exhausted", got)
	}
}

func TestProcessDirectPausesAfterOwnerMessageBudgetIsExhausted(t *testing.T) {
	t.Parallel()

	b := chat.NewHub(32)
	prov := &repeatedSuccessfulMessageToolProvider{}
	ag := NewAgentLoop(b, prov, prov.GetDefaultModel(), 25, t.TempDir(), nil)
	job := testMissionJob([]string{"message"}, []string{"message"})
	if err := ag.ActivateMissionStep(job, "build"); err != nil {
		t.Fatalf("ActivateMissionStep() error = %v", err)
	}

	resp, err := ag.ProcessDirect("keep going", 2*time.Second)
	if err != nil {
		t.Fatalf("ProcessDirect() error = %v", err)
	}
	if resp != "Mission paused: owner-facing message budget exhausted." {
		t.Fatalf("ProcessDirect() response = %q, want owner-message budget response", resp)
	}

	outbound := 0
	for {
		select {
		case out := <-b.Out:
			outbound++
			if out.Content != "budget check" {
				t.Fatalf("unexpected outbound message content: %#v", out)
			}
		default:
			goto drained
		}
	}

drained:
	if outbound != 20 {
		t.Fatalf("outbound message count = %d, want 20", outbound)
	}

	runtime, ok := ag.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if runtime.State != missioncontrol.JobStatePaused {
		t.Fatalf("MissionRuntimeState().State = %q, want %q", runtime.State, missioncontrol.JobStatePaused)
	}
	if runtime.BudgetBlocker == nil || runtime.BudgetBlocker.Ceiling != "owner_messages" {
		t.Fatalf("MissionRuntimeState().BudgetBlocker = %#v, want owner_messages blocker", runtime.BudgetBlocker)
	}
	appliedMessages := 0
	for _, event := range runtime.AuditHistory {
		if event.ToolName == "message" && event.ActionClass == missioncontrol.AuditActionClassToolCall && event.Allowed && event.Result == missioncontrol.AuditResultApplied {
			appliedMessages++
		}
	}
	if appliedMessages != 20 {
		t.Fatalf("MissionRuntimeState().applied message events = %d, want 20", appliedMessages)
	}
	if got := runtime.AuditHistory[len(runtime.AuditHistory)-1].ToolName; got != "budget_exhausted" {
		t.Fatalf("MissionRuntimeState().last audit tool = %q, want budget_exhausted", got)
	}
}

func TestProcessDirectCountsFinalStepOutputTowardOwnerMessageBudget(t *testing.T) {
	t.Parallel()

	b := chat.NewHub(32)
	prov := &finalResponseAfterNMessagesProvider{messages: 19}
	ag := NewAgentLoop(b, prov, prov.GetDefaultModel(), 25, t.TempDir(), nil)
	job := testMissionJob([]string{"message"}, []string{"message"})
	if err := ag.ActivateMissionStep(job, "build"); err != nil {
		t.Fatalf("ActivateMissionStep() error = %v", err)
	}

	resp, err := ag.ProcessDirect("keep going", 2*time.Second)
	if err != nil {
		t.Fatalf("ProcessDirect() error = %v", err)
	}
	if resp != "Mission paused: owner-facing message budget exhausted." {
		t.Fatalf("ProcessDirect() response = %q, want owner-message budget response", resp)
	}

	outbound := 0
	for {
		select {
		case out := <-b.Out:
			outbound++
			if out.Content != "budget check" {
				t.Fatalf("unexpected outbound message content: %#v", out)
			}
		default:
			goto drainedFinalStepOutputBudget
		}
	}

drainedFinalStepOutputBudget:
	if outbound != 19 {
		t.Fatalf("outbound message count = %d, want 19 before final step output budget pause", outbound)
	}

	runtime, ok := ag.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if runtime.State != missioncontrol.JobStatePaused {
		t.Fatalf("MissionRuntimeState().State = %q, want %q", runtime.State, missioncontrol.JobStatePaused)
	}
	if runtime.BudgetBlocker == nil || runtime.BudgetBlocker.Ceiling != "owner_messages" {
		t.Fatalf("MissionRuntimeState().BudgetBlocker = %#v, want owner_messages blocker", runtime.BudgetBlocker)
	}
	appliedMessages := 0
	stepOutputs := 0
	for _, event := range runtime.AuditHistory {
		if event.ToolName == "message" && event.ActionClass == missioncontrol.AuditActionClassToolCall && event.Allowed && event.Result == missioncontrol.AuditResultApplied {
			appliedMessages++
		}
		if event.ToolName == "step_output" && event.ActionClass == missioncontrol.AuditActionClassRuntime && event.Allowed && event.Result == missioncontrol.AuditResultApplied {
			stepOutputs++
		}
	}
	if appliedMessages != 19 {
		t.Fatalf("MissionRuntimeState().applied message events = %d, want 19", appliedMessages)
	}
	if stepOutputs != 1 {
		t.Fatalf("MissionRuntimeState().step output events = %d, want 1", stepOutputs)
	}
	if got := runtime.AuditHistory[len(runtime.AuditHistory)-1].ToolName; got != "budget_exhausted" {
		t.Fatalf("MissionRuntimeState().last audit tool = %q, want budget_exhausted", got)
	}
}

func TestProcessDirectCountsSetStepAcknowledgementTowardOwnerMessageBudget(t *testing.T) {
	t.Parallel()

	b := chat.NewHub(10)
	prov := &finalResponseProvider{content: "unused"}
	ag := NewAgentLoop(b, prov, prov.GetDefaultModel(), 3, "", nil)
	job := testMissionJob([]string{"read"}, []string{"read"})
	if err := ag.ActivateMissionStep(job, "build"); err != nil {
		t.Fatalf("ActivateMissionStep(build) error = %v", err)
	}
	for i := 0; i < 19; i++ {
		exhausted, err := ag.taskState.RecordOwnerFacingMessage()
		if err != nil {
			t.Fatalf("RecordOwnerFacingMessage() step %d error = %v", i, err)
		}
		if exhausted {
			t.Fatalf("RecordOwnerFacingMessage() step %d exhausted = true, want false before set-step acknowledgement", i)
		}
	}

	ag.SetOperatorSetStepHook(func(jobID string, stepID string) (string, error) {
		if err := ag.ActivateMissionStep(job, stepID); err != nil {
			return "", err
		}
		return "Set step job=job-1 step=final.", nil
	})

	resp, err := ag.ProcessDirect("SET_STEP job-1 final", 2*time.Second)
	if err != nil {
		t.Fatalf("ProcessDirect(SET_STEP) error = %v", err)
	}
	if !strings.Contains(resp, "Mission paused: owner-facing message budget exhausted.") {
		t.Fatalf("ProcessDirect(SET_STEP) response = %q, want owner-message budget response", resp)
	}
	if !strings.Contains(resp, "\n\nMission summary:\nPending steps: build") {
		t.Fatalf("ProcessDirect(SET_STEP) response = %q, want deterministic mission summary for final step", resp)
	}

	runtime, ok := ag.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if runtime.State != missioncontrol.JobStatePaused {
		t.Fatalf("MissionRuntimeState().State = %q, want %q", runtime.State, missioncontrol.JobStatePaused)
	}
	if runtime.ActiveStepID != "final" {
		t.Fatalf("MissionRuntimeState().ActiveStepID = %q, want %q", runtime.ActiveStepID, "final")
	}
	if runtime.BudgetBlocker == nil || runtime.BudgetBlocker.Ceiling != "owner_messages" {
		t.Fatalf("MissionRuntimeState().BudgetBlocker = %#v, want owner_messages blocker", runtime.BudgetBlocker)
	}
	if got := runtime.AuditHistory[len(runtime.AuditHistory)-2].ToolName; got != "set_step_ack" {
		t.Fatalf("MissionRuntimeState().penultimate audit tool = %q, want %q", got, "set_step_ack")
	}
	if got := runtime.AuditHistory[len(runtime.AuditHistory)-1].ToolName; got != "budget_exhausted" {
		t.Fatalf("MissionRuntimeState().last audit tool = %q, want %q", got, "budget_exhausted")
	}
}

func TestProcessDirectCountsResumeAcknowledgementTowardOwnerMessageBudget(t *testing.T) {
	t.Parallel()

	b := chat.NewHub(10)
	prov := &finalResponseProvider{content: "unused"}
	ag := NewAgentLoop(b, prov, prov.GetDefaultModel(), 3, "", nil)
	job := testMissionJob([]string{"read"}, []string{"read"})
	if err := ag.ActivateMissionStep(job, "build"); err != nil {
		t.Fatalf("ActivateMissionStep(build) error = %v", err)
	}
	for i := 0; i < 18; i++ {
		exhausted, err := ag.taskState.RecordOwnerFacingMessage()
		if err != nil {
			t.Fatalf("RecordOwnerFacingMessage() step %d error = %v", i, err)
		}
		if exhausted {
			t.Fatalf("RecordOwnerFacingMessage() step %d exhausted = true, want false before resume acknowledgement", i)
		}
	}

	resp, err := ag.ProcessDirect("PAUSE job-1", 2*time.Second)
	if err != nil {
		t.Fatalf("ProcessDirect(PAUSE) error = %v", err)
	}
	if resp != "Paused job=job-1." {
		t.Fatalf("ProcessDirect(PAUSE) response = %q, want pause acknowledgement", resp)
	}

	resp, err = ag.ProcessDirect("RESUME job-1", 2*time.Second)
	if err != nil {
		t.Fatalf("ProcessDirect(RESUME) error = %v", err)
	}
	if resp != "Mission paused: owner-facing message budget exhausted." {
		t.Fatalf("ProcessDirect(RESUME) response = %q, want owner-message budget response", resp)
	}

	runtime, ok := ag.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if runtime.State != missioncontrol.JobStatePaused {
		t.Fatalf("MissionRuntimeState().State = %q, want %q", runtime.State, missioncontrol.JobStatePaused)
	}
	if runtime.ActiveStepID != "build" {
		t.Fatalf("MissionRuntimeState().ActiveStepID = %q, want %q", runtime.ActiveStepID, "build")
	}
	if runtime.BudgetBlocker == nil || runtime.BudgetBlocker.Ceiling != "owner_messages" {
		t.Fatalf("MissionRuntimeState().BudgetBlocker = %#v, want owner_messages blocker", runtime.BudgetBlocker)
	}
	if got := runtime.AuditHistory[len(runtime.AuditHistory)-2].ToolName; got != "resume_ack" {
		t.Fatalf("MissionRuntimeState().penultimate audit tool = %q, want %q", got, "resume_ack")
	}
	if got := runtime.AuditHistory[len(runtime.AuditHistory)-1].ToolName; got != "budget_exhausted" {
		t.Fatalf("MissionRuntimeState().last audit tool = %q, want %q", got, "budget_exhausted")
	}
}

func TestProcessDirectCountsPauseAcknowledgementTowardOwnerMessageBudget(t *testing.T) {
	t.Parallel()

	b := chat.NewHub(10)
	prov := &finalResponseProvider{content: "unused"}
	ag := NewAgentLoop(b, prov, prov.GetDefaultModel(), 3, "", nil)
	job := testMissionJob([]string{"read"}, []string{"read"})
	if err := ag.ActivateMissionStep(job, "build"); err != nil {
		t.Fatalf("ActivateMissionStep(build) error = %v", err)
	}
	for i := 0; i < 19; i++ {
		exhausted, err := ag.taskState.RecordOwnerFacingMessage()
		if err != nil {
			t.Fatalf("RecordOwnerFacingMessage() step %d error = %v", i, err)
		}
		if exhausted {
			t.Fatalf("RecordOwnerFacingMessage() step %d exhausted = true, want false before pause acknowledgement", i)
		}
	}

	resp, err := ag.ProcessDirect("PAUSE job-1", 2*time.Second)
	if err != nil {
		t.Fatalf("ProcessDirect(PAUSE) error = %v", err)
	}
	if resp != "Mission paused: owner-facing message budget exhausted." {
		t.Fatalf("ProcessDirect(PAUSE) response = %q, want owner-message budget response", resp)
	}

	runtime, ok := ag.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if runtime.State != missioncontrol.JobStatePaused {
		t.Fatalf("MissionRuntimeState().State = %q, want %q", runtime.State, missioncontrol.JobStatePaused)
	}
	if runtime.ActiveStepID != "build" {
		t.Fatalf("MissionRuntimeState().ActiveStepID = %q, want %q", runtime.ActiveStepID, "build")
	}
	if runtime.BudgetBlocker == nil || runtime.BudgetBlocker.Ceiling != "owner_messages" {
		t.Fatalf("MissionRuntimeState().BudgetBlocker = %#v, want owner_messages blocker", runtime.BudgetBlocker)
	}
	if got := runtime.AuditHistory[len(runtime.AuditHistory)-2].ToolName; got != "pause_ack" {
		t.Fatalf("MissionRuntimeState().penultimate audit tool = %q, want %q", got, "pause_ack")
	}
	if got := runtime.AuditHistory[len(runtime.AuditHistory)-1].ToolName; got != "budget_exhausted" {
		t.Fatalf("MissionRuntimeState().last audit tool = %q, want %q", got, "budget_exhausted")
	}
}

func TestProcessDirectCountsRevokeApprovalAcknowledgementTowardOwnerMessageBudget(t *testing.T) {
	t.Parallel()

	b := chat.NewHub(10)
	prov := &finalResponseProvider{content: "unused"}
	ag := NewAgentLoop(b, prov, prov.GetDefaultModel(), 3, "", nil)
	job := testReusableApprovalMissionJob(missioncontrol.ApprovalScopeOneJob)
	now := time.Now().UTC()
	runtimeState := missioncontrol.JobRuntimeState{
		JobID:        job.ID,
		State:        missioncontrol.JobStateRunning,
		ActiveStepID: "authorize-2",
		ApprovalRequests: []missioncontrol.ApprovalRequest{
			{
				JobID:           job.ID,
				StepID:          "authorize-1",
				RequestedAction: missioncontrol.ApprovalRequestedActionStepComplete,
				Scope:           missioncontrol.ApprovalScopeOneJob,
				RequestedVia:    missioncontrol.ApprovalRequestedViaRuntime,
				GrantedVia:      missioncontrol.ApprovalGrantedViaOperatorCommand,
				State:           missioncontrol.ApprovalStateGranted,
				RequestedAt:     now.Add(-2 * time.Minute),
				ResolvedAt:      now.Add(-90 * time.Second),
			},
		},
		ApprovalGrants: []missioncontrol.ApprovalGrant{
			{
				JobID:           job.ID,
				StepID:          "authorize-1",
				RequestedAction: missioncontrol.ApprovalRequestedActionStepComplete,
				Scope:           missioncontrol.ApprovalScopeOneJob,
				GrantedVia:      missioncontrol.ApprovalGrantedViaOperatorCommand,
				State:           missioncontrol.ApprovalStateGranted,
				GrantedAt:       now.Add(-90 * time.Second),
				ExpiresAt:       now.Add(time.Minute),
			},
		},
	}
	if err := ag.taskState.ResumeRuntime(job, runtimeState, nil, true); err != nil {
		t.Fatalf("ResumeRuntime() error = %v", err)
	}
	for i := 0; i < 19; i++ {
		exhausted, err := ag.taskState.RecordOwnerFacingMessage()
		if err != nil {
			t.Fatalf("RecordOwnerFacingMessage() step %d error = %v", i, err)
		}
		if exhausted {
			t.Fatalf("RecordOwnerFacingMessage() step %d exhausted = true, want false before revoke acknowledgement", i)
		}
	}

	resp, err := ag.ProcessDirect("REVOKE_APPROVAL job-1 authorize-2", 2*time.Second)
	if err != nil {
		t.Fatalf("ProcessDirect(REVOKE_APPROVAL) error = %v", err)
	}
	if resp != "Mission paused: owner-facing message budget exhausted." {
		t.Fatalf("ProcessDirect(REVOKE_APPROVAL) response = %q, want owner-message budget response", resp)
	}

	runtime, ok := ag.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if runtime.State != missioncontrol.JobStatePaused {
		t.Fatalf("MissionRuntimeState().State = %q, want %q", runtime.State, missioncontrol.JobStatePaused)
	}
	if runtime.ActiveStepID != "authorize-2" {
		t.Fatalf("MissionRuntimeState().ActiveStepID = %q, want %q", runtime.ActiveStepID, "authorize-2")
	}
	if runtime.BudgetBlocker == nil || runtime.BudgetBlocker.Ceiling != "owner_messages" {
		t.Fatalf("MissionRuntimeState().BudgetBlocker = %#v, want owner_messages blocker", runtime.BudgetBlocker)
	}
	if got := runtime.AuditHistory[len(runtime.AuditHistory)-2].ToolName; got != "revoke_approval_ack" {
		t.Fatalf("MissionRuntimeState().penultimate audit tool = %q, want %q", got, "revoke_approval_ack")
	}
	if got := runtime.AuditHistory[len(runtime.AuditHistory)-1].ToolName; got != "budget_exhausted" {
		t.Fatalf("MissionRuntimeState().last audit tool = %q, want %q", got, "budget_exhausted")
	}
}

func TestProcessDirectCountsDenyAcknowledgementTowardOwnerMessageBudget(t *testing.T) {
	t.Parallel()

	b := chat.NewHub(10)
	prov := &finalResponseProvider{content: "unused"}
	ag := NewAgentLoop(b, prov, prov.GetDefaultModel(), 3, "", nil)
	job := testDiscussionMissionJob()
	if err := ag.ActivateMissionStep(job, "build"); err != nil {
		t.Fatalf("ActivateMissionStep(build) error = %v", err)
	}
	for i := 0; i < 18; i++ {
		exhausted, err := ag.taskState.RecordOwnerFacingMessage()
		if err != nil {
			t.Fatalf("RecordOwnerFacingMessage() step %d error = %v", i, err)
		}
		if exhausted {
			t.Fatalf("RecordOwnerFacingMessage() step %d exhausted = true, want false before deny acknowledgement", i)
		}
	}
	if err := ag.taskState.ApplyStepOutput("Waiting for approval.", nil); err != nil {
		t.Fatalf("ApplyStepOutput() error = %v", err)
	}

	resp, err := ag.ProcessDirect("DENY job-1 build", 2*time.Second)
	if err != nil {
		t.Fatalf("ProcessDirect(DENY) error = %v", err)
	}
	if resp != "Mission paused: owner-facing message budget exhausted." {
		t.Fatalf("ProcessDirect(DENY) response = %q, want owner-message budget response", resp)
	}

	runtime, ok := ag.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if runtime.State != missioncontrol.JobStatePaused {
		t.Fatalf("MissionRuntimeState().State = %q, want %q", runtime.State, missioncontrol.JobStatePaused)
	}
	if runtime.ActiveStepID != "build" {
		t.Fatalf("MissionRuntimeState().ActiveStepID = %q, want %q", runtime.ActiveStepID, "build")
	}
	if runtime.BudgetBlocker == nil || runtime.BudgetBlocker.Ceiling != "owner_messages" {
		t.Fatalf("MissionRuntimeState().BudgetBlocker = %#v, want owner_messages blocker", runtime.BudgetBlocker)
	}
	if got := runtime.AuditHistory[len(runtime.AuditHistory)-2].ToolName; got != "deny_ack" {
		t.Fatalf("MissionRuntimeState().penultimate audit tool = %q, want %q", got, "deny_ack")
	}
	if got := runtime.AuditHistory[len(runtime.AuditHistory)-1].ToolName; got != "budget_exhausted" {
		t.Fatalf("MissionRuntimeState().last audit tool = %q, want %q", got, "budget_exhausted")
	}
}

func TestClearMissionStepRestoresNoContextBehavior(t *testing.T) {
	t.Parallel()

	b := chat.NewHub(10)
	prov := &deniedMessageToolProvider{}
	ag := NewAgentLoop(b, prov, prov.GetDefaultModel(), 5, "", nil)
	if err := ag.ActivateMissionStep(testMissionJob([]string{"write_memory"}, []string{"write_memory"}), "build"); err != nil {
		t.Fatalf("ActivateMissionStep() error = %v", err)
	}

	ag.ClearMissionStep()

	if _, ok := ag.ActiveMissionStep(); ok {
		t.Fatal("ActiveMissionStep() ok = true, want false after clear")
	}

	resp, err := ag.ProcessDirect("try to send a message", 2*time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp != "sent" {
		t.Fatalf("expected tool result 'sent', got %q", resp)
	}

	select {
	case out := <-b.Out:
		if out.Content != "should not send" {
			t.Fatalf("unexpected outbound content: %q", out.Content)
		}
	default:
		t.Fatal("expected message tool to run after ClearMissionStep()")
	}
}

func contains(s, sub string) bool { return strings.Contains(s, sub) }

func writeLoopRollbackPromotionFixtures(t *testing.T) (string, missioncontrol.ActiveRuntimePackPointer) {
	t.Helper()

	root := t.TempDir()
	now := time.Date(2026, 4, 21, 12, 0, 0, 0, time.UTC)

	if err := missioncontrol.StoreRuntimePackRecord(root, missioncontrol.RuntimePackRecord{
		PackID:                   "pack-base",
		CreatedAt:                now.Add(-4 * time.Minute),
		Channel:                  "phone",
		PromptPackRef:            "prompt-pack-base",
		SkillPackRef:             "skill-pack-base",
		ManifestRef:              "manifest-base",
		ExtensionPackRef:         "extension-base",
		PolicyRef:                "policy-base",
		SourceSummary:            "baseline pack",
		MutableSurfaces:          []string{"prompts", "skills"},
		ImmutableSurfaces:        []string{"policy", "authority"},
		SurfaceClasses:           []string{"class_1"},
		CompatibilityContractRef: "compat-v1",
	}); err != nil {
		t.Fatalf("StoreRuntimePackRecord(pack-base) error = %v", err)
	}
	if err := missioncontrol.StoreRuntimePackRecord(root, missioncontrol.RuntimePackRecord{
		PackID:                   "pack-candidate",
		ParentPackID:             "pack-base",
		RollbackTargetPackID:     "pack-base",
		CreatedAt:                now.Add(-3 * time.Minute),
		Channel:                  "phone",
		PromptPackRef:            "prompt-pack-candidate",
		SkillPackRef:             "skill-pack-candidate",
		ManifestRef:              "manifest-candidate",
		ExtensionPackRef:         "extension-candidate",
		PolicyRef:                "policy-candidate",
		SourceSummary:            "candidate pack",
		MutableSurfaces:          []string{"prompts", "skills"},
		ImmutableSurfaces:        []string{"policy", "authority"},
		SurfaceClasses:           []string{"class_1"},
		CompatibilityContractRef: "compat-v1",
	}); err != nil {
		t.Fatalf("StoreRuntimePackRecord(pack-candidate) error = %v", err)
	}

	wantPointer := missioncontrol.ActiveRuntimePackPointer{
		ActivePackID:         "pack-candidate",
		PreviousActivePackID: "pack-base",
		LastKnownGoodPackID:  "pack-base",
		UpdatedAt:            now.Add(-30 * time.Second),
		UpdatedBy:            "operator",
		UpdateRecordRef:      "promotion:promotion-1",
		ReloadGeneration:     7,
	}
	if err := missioncontrol.StoreActiveRuntimePackPointer(root, wantPointer); err != nil {
		t.Fatalf("StoreActiveRuntimePackPointer() error = %v", err)
	}
	wantPointer.RecordVersion = 1

	if err := missioncontrol.StoreHotUpdateGateRecord(root, missioncontrol.HotUpdateGateRecord{
		HotUpdateID:              "hot-update-1",
		Objective:                "promote candidate pack",
		CandidatePackID:          "pack-candidate",
		PreviousActivePackID:     "pack-base",
		RollbackTargetPackID:     "pack-base",
		TargetSurfaces:           []string{"prompts", "skills"},
		SurfaceClasses:           []string{"class_1"},
		ReloadMode:               missioncontrol.HotUpdateReloadModeSoftReload,
		CompatibilityContractRef: "compat-v1",
		PreparedAt:               now.Add(-150 * time.Second),
		State:                    missioncontrol.HotUpdateGateStateStaged,
		Decision:                 missioncontrol.HotUpdateGateDecisionKeepStaged,
	}); err != nil {
		t.Fatalf("StoreHotUpdateGateRecord() error = %v", err)
	}
	if err := missioncontrol.StoreHotUpdateOutcomeRecord(root, missioncontrol.HotUpdateOutcomeRecord{
		OutcomeID:       "outcome-1",
		HotUpdateID:     "hot-update-1",
		CandidatePackID: "pack-candidate",
		OutcomeKind:     missioncontrol.HotUpdateOutcomeKindPromoted,
		Reason:          "operator promoted candidate",
		Notes:           "promotion linkage",
		OutcomeAt:       now.Add(-90 * time.Second),
		CreatedAt:       now.Add(-80 * time.Second),
		CreatedBy:       "operator",
	}); err != nil {
		t.Fatalf("StoreHotUpdateOutcomeRecord(outcome-1) error = %v", err)
	}
	if err := missioncontrol.StorePromotionRecord(root, missioncontrol.PromotionRecord{
		PromotionID:          "promotion-1",
		PromotedPackID:       "pack-candidate",
		PreviousActivePackID: "pack-base",
		LastKnownGoodPackID:  "pack-base",
		LastKnownGoodBasis:   "holdout_pass",
		HotUpdateID:          "hot-update-1",
		OutcomeID:            "outcome-1",
		Reason:               "operator approved promotion",
		Notes:                "promotion notes",
		PromotedAt:           now.Add(-70 * time.Second),
		CreatedAt:            now.Add(-60 * time.Second),
		CreatedBy:            "operator",
	}); err != nil {
		t.Fatalf("StorePromotionRecord() error = %v", err)
	}

	return root, wantPointer
}

func writeLoopHotUpdateGateControlFixtures(t *testing.T) (string, missioncontrol.ActiveRuntimePackPointer) {
	t.Helper()

	root := t.TempDir()
	now := time.Date(2026, 4, 22, 12, 0, 0, 0, time.UTC)

	if err := missioncontrol.StoreRuntimePackRecord(root, missioncontrol.RuntimePackRecord{
		PackID:                   "pack-base",
		CreatedAt:                now.Add(-4 * time.Minute),
		Channel:                  "phone",
		PromptPackRef:            "prompt-pack-base",
		SkillPackRef:             "skill-pack-base",
		ManifestRef:              "manifest-base",
		ExtensionPackRef:         "extension-base",
		PolicyRef:                "policy-base",
		SourceSummary:            "baseline pack",
		MutableSurfaces:          []string{"prompts", "skills"},
		ImmutableSurfaces:        []string{"policy", "authority"},
		SurfaceClasses:           []string{"class_1"},
		CompatibilityContractRef: "compat-v1",
	}); err != nil {
		t.Fatalf("StoreRuntimePackRecord(pack-base) error = %v", err)
	}
	if err := missioncontrol.StoreRuntimePackRecord(root, missioncontrol.RuntimePackRecord{
		PackID:                   "pack-candidate",
		ParentPackID:             "pack-base",
		RollbackTargetPackID:     "pack-base",
		CreatedAt:                now.Add(-3 * time.Minute),
		Channel:                  "phone",
		PromptPackRef:            "prompt-pack-candidate",
		SkillPackRef:             "skill-pack-candidate",
		ManifestRef:              "manifest-candidate",
		ExtensionPackRef:         "extension-candidate",
		PolicyRef:                "policy-candidate",
		SourceSummary:            "candidate pack",
		MutableSurfaces:          []string{"skills"},
		ImmutableSurfaces:        []string{"policy", "authority"},
		SurfaceClasses:           []string{"class_1"},
		CompatibilityContractRef: "compat-v1",
	}); err != nil {
		t.Fatalf("StoreRuntimePackRecord(pack-candidate) error = %v", err)
	}

	wantPointer := missioncontrol.ActiveRuntimePackPointer{
		ActivePackID:        "pack-base",
		LastKnownGoodPackID: "pack-base",
		UpdatedAt:           now.Add(-30 * time.Second),
		UpdatedBy:           "operator",
		UpdateRecordRef:     "bootstrap",
		ReloadGeneration:    2,
	}
	if err := missioncontrol.StoreActiveRuntimePackPointer(root, wantPointer); err != nil {
		t.Fatalf("StoreActiveRuntimePackPointer() error = %v", err)
	}
	wantPointer.RecordVersion = 1

	return root, wantPointer
}

func writeLoopCandidatePromotionDecisionGateFixtures(t *testing.T, withLastKnownGood bool) (string, missioncontrol.CandidatePromotionDecisionRecord) {
	t.Helper()

	root, _ := writeLoopHotUpdateGateControlFixtures(t)
	now := time.Date(2026, 4, 25, 18, 45, 0, 0, time.UTC)

	if withLastKnownGood {
		if err := missioncontrol.StoreLastKnownGoodRuntimePackPointer(root, missioncontrol.LastKnownGoodRuntimePackPointer{
			PackID:            "pack-base",
			Basis:             "holdout_pass",
			VerifiedAt:        now.Add(-30 * time.Second),
			VerifiedBy:        "operator",
			RollbackRecordRef: "bootstrap",
		}); err != nil {
			t.Fatalf("StoreLastKnownGoodRuntimePackPointer() error = %v", err)
		}
	}
	if err := missioncontrol.StoreImprovementCandidateRecord(root, missioncontrol.ImprovementCandidateRecord{
		CandidateID:         "candidate-1",
		BaselinePackID:      "pack-base",
		CandidatePackID:     "pack-candidate",
		SourceSummary:       "candidate linkage",
		ValidationBasisRefs: []string{"eval-suite-1"},
		CreatedAt:           now.Add(time.Minute),
		CreatedBy:           "operator",
	}); err != nil {
		t.Fatalf("StoreImprovementCandidateRecord() error = %v", err)
	}
	if err := missioncontrol.StoreEvalSuiteRecord(root, missioncontrol.EvalSuiteRecord{
		EvalSuiteID:       "eval-suite-1",
		RubricRef:         "rubric-v1",
		TrainCorpusRef:    "train-corpus-v1",
		HoldoutCorpusRef:  "holdout-corpus-v1",
		EvaluatorRef:      "evaluator-v1",
		NegativeCaseCount: 1,
		BoundaryCaseCount: 1,
		FrozenForRun:      true,
		CandidateID:       "candidate-1",
		BaselinePackID:    "pack-base",
		CandidatePackID:   "pack-candidate",
		CreatedAt:         now.Add(2 * time.Minute),
		CreatedBy:         "operator",
	}); err != nil {
		t.Fatalf("StoreEvalSuiteRecord() error = %v", err)
	}
	if err := missioncontrol.StoreImprovementRunRecord(root, missioncontrol.ImprovementRunRecord{
		RunID:           "run-result",
		Objective:       "evaluate candidate for promotion",
		ExecutionPlane:  missioncontrol.ExecutionPlaneImprovementWorkspace,
		ExecutionHost:   "phone",
		MissionFamily:   missioncontrol.MissionFamilyEvaluateCandidate,
		TargetType:      "prompt_pack",
		TargetRef:       "prompt-pack://default",
		SurfaceClass:    "class_1",
		CandidateID:     "candidate-1",
		EvalSuiteID:     "eval-suite-1",
		BaselinePackID:  "pack-base",
		CandidatePackID: "pack-candidate",
		State:           missioncontrol.ImprovementRunStateCandidateReady,
		Decision:        missioncontrol.ImprovementRunDecisionKeep,
		CreatedAt:       now.Add(3 * time.Minute),
		CompletedAt:     now.Add(4 * time.Minute),
		StopReason:      "candidate ready for promotion decision",
		CreatedBy:       "operator",
	}); err != nil {
		t.Fatalf("StoreImprovementRunRecord() error = %v", err)
	}
	if err := missioncontrol.StorePromotionPolicyRecord(root, missioncontrol.PromotionPolicyRecord{
		PromotionPolicyID:         "promotion-policy-result",
		RequiresHoldoutPass:       true,
		RequiresCanary:            false,
		RequiresOwnerApproval:     false,
		AllowsAutonomousHotUpdate: true,
		AllowedSurfaceClasses:     []string{"class_1"},
		EpsilonRule:               "epsilon <= 0.01",
		RegressionRule:            "no_regression_flags",
		CompatibilityRule:         "compatibility_score >= 0.90",
		ResourceRule:              "resource_score >= 0.60",
		MaxCanaryDuration:         "15m",
		ForbiddenSurfaceChanges:   []string{"policy", "authority"},
		CreatedAt:                 now.Add(5 * time.Minute),
		CreatedBy:                 "operator",
		Notes:                     "eligible promotion fixture",
	}); err != nil {
		t.Fatalf("StorePromotionPolicyRecord() error = %v", err)
	}
	if err := missioncontrol.StoreCandidateResultRecord(root, missioncontrol.CandidateResultRecord{
		ResultID:           "result-eligible",
		RunID:              "run-result",
		CandidateID:        "candidate-1",
		EvalSuiteID:        "eval-suite-1",
		PromotionPolicyID:  "promotion-policy-result",
		BaselinePackID:     "pack-base",
		CandidatePackID:    "pack-candidate",
		BaselineScore:      0.52,
		TrainScore:         0.78,
		HoldoutScore:       0.74,
		ComplexityScore:    0.21,
		CompatibilityScore: 0.93,
		ResourceScore:      0.67,
		RegressionFlags:    []string{"none"},
		Decision:           missioncontrol.ImprovementRunDecisionKeep,
		Notes:              "eligible candidate result",
		CreatedAt:          now.Add(6 * time.Minute),
		CreatedBy:          "operator",
	}); err != nil {
		t.Fatalf("StoreCandidateResultRecord() error = %v", err)
	}
	decision, changed, err := missioncontrol.CreateCandidatePromotionDecisionFromEligibleResult(root, "result-eligible", "operator", now.Add(7*time.Minute))
	if err != nil {
		t.Fatalf("CreateCandidatePromotionDecisionFromEligibleResult() error = %v", err)
	}
	if !changed {
		t.Fatal("CreateCandidatePromotionDecisionFromEligibleResult() changed = false, want true")
	}
	return root, decision
}

type loopCandidateDecisionGateSideEffects struct {
	files            map[string][]byte
	reloadGeneration uint64
}

func snapshotLoopCandidateDecisionGateSideEffects(t *testing.T, root string, decision missioncontrol.CandidatePromotionDecisionRecord) loopCandidateDecisionGateSideEffects {
	t.Helper()

	paths := []string{
		missioncontrol.StoreCandidatePromotionDecisionPath(root, decision.PromotionDecisionID),
		missioncontrol.StoreCandidateResultPath(root, decision.ResultID),
		missioncontrol.StoreImprovementRunPath(root, decision.RunID),
		missioncontrol.StoreImprovementCandidatePath(root, decision.CandidateID),
		missioncontrol.StoreEvalSuitePath(root, decision.EvalSuiteID),
		missioncontrol.StorePromotionPolicyPath(root, decision.PromotionPolicyID),
		missioncontrol.StoreRuntimePackPath(root, decision.BaselinePackID),
		missioncontrol.StoreRuntimePackPath(root, decision.CandidatePackID),
		missioncontrol.StoreActiveRuntimePackPointerPath(root),
		missioncontrol.StoreLastKnownGoodRuntimePackPointerPath(root),
	}
	files := make(map[string][]byte, len(paths))
	for _, path := range paths {
		bytes, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ReadFile(%s) before error = %v", path, err)
		}
		files[path] = bytes
	}
	pointer, err := missioncontrol.LoadActiveRuntimePackPointer(root)
	if err != nil {
		t.Fatalf("LoadActiveRuntimePackPointer(before) error = %v", err)
	}
	return loopCandidateDecisionGateSideEffects{
		files:            files,
		reloadGeneration: pointer.ReloadGeneration,
	}
}

func assertLoopCandidateDecisionGateSideEffectsUnchanged(t *testing.T, root string, decision missioncontrol.CandidatePromotionDecisionRecord, before loopCandidateDecisionGateSideEffects) {
	t.Helper()

	for path, beforeBytes := range before.files {
		afterBytes, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ReadFile(%s) after error = %v", path, err)
		}
		if string(afterBytes) != string(beforeBytes) {
			t.Fatalf("source/runtime file %s changed after decision-derived hot-update gate\nbefore:\n%s\nafter:\n%s", path, string(beforeBytes), string(afterBytes))
		}
	}
	pointer, err := missioncontrol.LoadActiveRuntimePackPointer(root)
	if err != nil {
		t.Fatalf("LoadActiveRuntimePackPointer(after) error = %v", err)
	}
	if pointer.ReloadGeneration != before.reloadGeneration {
		t.Fatalf("LoadActiveRuntimePackPointer().ReloadGeneration = %d, want %d", pointer.ReloadGeneration, before.reloadGeneration)
	}
	if pointer.ActivePackID != decision.BaselinePackID {
		t.Fatalf("LoadActiveRuntimePackPointer().ActivePackID = %q, want %q", pointer.ActivePackID, decision.BaselinePackID)
	}
}

func assertLoopCandidateDecisionGateNoTerminalRecords(t *testing.T, root string) {
	t.Helper()

	outcomes, err := missioncontrol.ListHotUpdateOutcomeRecords(root)
	if err != nil && !os.IsNotExist(err) {
		t.Fatalf("ListHotUpdateOutcomeRecords() error = %v", err)
	}
	if len(outcomes) != 0 {
		t.Fatalf("ListHotUpdateOutcomeRecords() len = %d, want 0", len(outcomes))
	}
	promotions, err := missioncontrol.ListPromotionRecords(root)
	if err != nil && !os.IsNotExist(err) {
		t.Fatalf("ListPromotionRecords() error = %v", err)
	}
	if len(promotions) != 0 {
		t.Fatalf("ListPromotionRecords() len = %d, want 0", len(promotions))
	}
	rollbacks, err := missioncontrol.ListRollbackRecords(root)
	if err != nil && !os.IsNotExist(err) {
		t.Fatalf("ListRollbackRecords() error = %v", err)
	}
	if len(rollbacks) != 0 {
		t.Fatalf("ListRollbackRecords() len = %d, want 0", len(rollbacks))
	}
	applies, err := missioncontrol.ListRollbackApplyRecords(root)
	if err != nil && !os.IsNotExist(err) {
		t.Fatalf("ListRollbackApplyRecords() error = %v", err)
	}
	if len(applies) != 0 {
		t.Fatalf("ListRollbackApplyRecords() len = %d, want 0", len(applies))
	}
}

func newLoopHotUpdateOutcomeAgent(t *testing.T, root string) *AgentLoop {
	t.Helper()

	b := chat.NewHub(10)
	prov := &finalResponseProvider{content: "unused"}
	ag := NewAgentLoop(b, prov, prov.GetDefaultModel(), 3, "", nil)
	ag.SetMissionStoreRoot(root)
	if err := ag.ActivateMissionStep(testMissionJob([]string{"read"}, []string{"read"}), "build"); err != nil {
		t.Fatalf("ActivateMissionStep() error = %v", err)
	}
	return ag
}

func prepareLoopHotUpdateSucceededGate(t *testing.T, ag *AgentLoop) {
	t.Helper()

	commands := []string{
		"HOT_UPDATE_GATE_RECORD job-1 hot-update-1 pack-candidate",
		"HOT_UPDATE_GATE_PHASE job-1 hot-update-1 validated",
		"HOT_UPDATE_GATE_PHASE job-1 hot-update-1 staged",
		"HOT_UPDATE_GATE_EXECUTE job-1 hot-update-1",
		"HOT_UPDATE_GATE_RELOAD job-1 hot-update-1",
	}
	for _, command := range commands {
		if _, err := ag.ProcessDirect(command, 2*time.Second); err != nil {
			t.Fatalf("ProcessDirect(%s) error = %v", command, err)
		}
	}
}

func writeLoopHotUpdateLastKnownGoodPointer(t *testing.T, root string) {
	t.Helper()

	now := time.Date(2026, 4, 22, 12, 0, 0, 0, time.UTC)
	if err := missioncontrol.StoreLastKnownGoodRuntimePackPointer(root, missioncontrol.LastKnownGoodRuntimePackPointer{
		PackID:            "pack-base",
		Basis:             "holdout_pass",
		VerifiedAt:        now.Add(-time.Minute),
		VerifiedBy:        "operator",
		RollbackRecordRef: "bootstrap",
	}); err != nil {
		t.Fatalf("StoreLastKnownGoodRuntimePackPointer() error = %v", err)
	}
}

func storeLoopHotUpdateTerminalGate(t *testing.T, root string, hotUpdateID string, state missioncontrol.HotUpdateGateState, failureReason string) missioncontrol.HotUpdateGateRecord {
	t.Helper()

	now := time.Date(2026, 4, 22, 12, 0, 0, 0, time.UTC)
	record := missioncontrol.HotUpdateGateRecord{
		HotUpdateID:              hotUpdateID,
		Objective:                "operator requested hot-update gate for candidate pack-candidate",
		CandidatePackID:          "pack-candidate",
		PreviousActivePackID:     "pack-base",
		RollbackTargetPackID:     "pack-base",
		TargetSurfaces:           []string{"skills"},
		SurfaceClasses:           []string{"class_1"},
		ReloadMode:               missioncontrol.HotUpdateReloadModeSkillReload,
		CompatibilityContractRef: "compat-v1",
		PreparedAt:               now.Add(time.Minute),
		PhaseUpdatedAt:           now.Add(2 * time.Minute),
		PhaseUpdatedBy:           "operator",
		State:                    state,
		Decision:                 missioncontrol.HotUpdateGateDecisionKeepStaged,
		FailureReason:            failureReason,
	}
	if err := missioncontrol.StoreHotUpdateGateRecord(root, record); err != nil {
		t.Fatalf("StoreHotUpdateGateRecord() error = %v", err)
	}
	stored, err := missioncontrol.LoadHotUpdateGateRecord(root, hotUpdateID)
	if err != nil {
		t.Fatalf("LoadHotUpdateGateRecord() error = %v", err)
	}
	return stored
}

type loopHotUpdateOutcomeCreateSideEffects struct {
	activePointerBytes   []byte
	lastKnownGoodBytes   []byte
	hotUpdateGateBytes   []byte
	reloadGeneration     uint64
	hotUpdateGateRecords int
}

func snapshotLoopHotUpdateOutcomeCreateSideEffects(t *testing.T, root string, hotUpdateID string) loopHotUpdateOutcomeCreateSideEffects {
	t.Helper()

	activePointerBytes, err := os.ReadFile(missioncontrol.StoreActiveRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(active pointer before) error = %v", err)
	}
	lastKnownGoodBytes, err := os.ReadFile(missioncontrol.StoreLastKnownGoodRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(last-known-good pointer before) error = %v", err)
	}
	hotUpdateGateBytes, err := os.ReadFile(missioncontrol.StoreHotUpdateGatePath(root, hotUpdateID))
	if err != nil {
		t.Fatalf("ReadFile(hot-update gate before) error = %v", err)
	}
	pointer, err := missioncontrol.LoadActiveRuntimePackPointer(root)
	if err != nil {
		t.Fatalf("LoadActiveRuntimePackPointer() error = %v", err)
	}
	gates, err := missioncontrol.ListHotUpdateGateRecords(root)
	if err != nil {
		t.Fatalf("ListHotUpdateGateRecords() error = %v", err)
	}
	return loopHotUpdateOutcomeCreateSideEffects{
		activePointerBytes:   activePointerBytes,
		lastKnownGoodBytes:   lastKnownGoodBytes,
		hotUpdateGateBytes:   hotUpdateGateBytes,
		reloadGeneration:     pointer.ReloadGeneration,
		hotUpdateGateRecords: len(gates),
	}
}

func assertLoopHotUpdateOutcomeCreateSideEffectsUnchanged(t *testing.T, root string, hotUpdateID string, before loopHotUpdateOutcomeCreateSideEffects) {
	t.Helper()

	afterPointerBytes, err := os.ReadFile(missioncontrol.StoreActiveRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(active pointer after) error = %v", err)
	}
	if string(before.activePointerBytes) != string(afterPointerBytes) {
		t.Fatalf("active runtime pack pointer file changed during outcome create\nbefore:\n%s\nafter:\n%s", string(before.activePointerBytes), string(afterPointerBytes))
	}
	pointer, err := missioncontrol.LoadActiveRuntimePackPointer(root)
	if err != nil {
		t.Fatalf("LoadActiveRuntimePackPointer(after) error = %v", err)
	}
	if pointer.ReloadGeneration != before.reloadGeneration {
		t.Fatalf("LoadActiveRuntimePackPointer().ReloadGeneration = %d, want %d", pointer.ReloadGeneration, before.reloadGeneration)
	}

	afterLastKnownGoodBytes, err := os.ReadFile(missioncontrol.StoreLastKnownGoodRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(last-known-good pointer after) error = %v", err)
	}
	if string(before.lastKnownGoodBytes) != string(afterLastKnownGoodBytes) {
		t.Fatalf("last-known-good pointer changed during outcome create\nbefore:\n%s\nafter:\n%s", string(before.lastKnownGoodBytes), string(afterLastKnownGoodBytes))
	}

	afterGateBytes, err := os.ReadFile(missioncontrol.StoreHotUpdateGatePath(root, hotUpdateID))
	if err != nil {
		t.Fatalf("ReadFile(hot-update gate after) error = %v", err)
	}
	if string(before.hotUpdateGateBytes) != string(afterGateBytes) {
		t.Fatalf("hot-update gate changed during outcome create\nbefore:\n%s\nafter:\n%s", string(before.hotUpdateGateBytes), string(afterGateBytes))
	}
	gates, err := missioncontrol.ListHotUpdateGateRecords(root)
	if err != nil {
		t.Fatalf("ListHotUpdateGateRecords(after) error = %v", err)
	}
	if len(gates) != before.hotUpdateGateRecords {
		t.Fatalf("ListHotUpdateGateRecords(after) len = %d, want %d", len(gates), before.hotUpdateGateRecords)
	}
}

func assertLoopHotUpdateOutcomeCreateNoPromotions(t *testing.T, root string) {
	t.Helper()

	promotions, err := missioncontrol.ListPromotionRecords(root)
	if err != nil && !os.IsNotExist(err) {
		t.Fatalf("ListPromotionRecords() error = %v", err)
	}
	if len(promotions) != 0 {
		t.Fatalf("ListPromotionRecords() len = %d, want 0", len(promotions))
	}
}

func assertLoopHotUpdateOutcomeCount(t *testing.T, root string, want int) {
	t.Helper()

	outcomes, err := missioncontrol.ListHotUpdateOutcomeRecords(root)
	if err != nil && !os.IsNotExist(err) {
		t.Fatalf("ListHotUpdateOutcomeRecords() error = %v", err)
	}
	if len(outcomes) != want {
		t.Fatalf("ListHotUpdateOutcomeRecords() len = %d, want %d", len(outcomes), want)
	}
}

func storeLoopHotUpdateOutcomeForPromotion(t *testing.T, root string, kind missioncontrol.HotUpdateOutcomeKind, reason string) string {
	t.Helper()

	return storeLoopHotUpdateOutcomeForPromotionWithMutation(t, root, func(record *missioncontrol.HotUpdateOutcomeRecord) {
		record.OutcomeKind = kind
		record.Reason = reason
	})
}

func storeLoopHotUpdateOutcomeForPromotionWithMutation(t *testing.T, root string, mutate func(*missioncontrol.HotUpdateOutcomeRecord)) string {
	t.Helper()

	gate := storeLoopHotUpdateTerminalGate(t, root, "hot-update-1", missioncontrol.HotUpdateGateStateReloadApplySucceeded, "")
	record := missioncontrol.HotUpdateOutcomeRecord{
		OutcomeID:       "hot-update-outcome-hot-update-1",
		HotUpdateID:     "hot-update-1",
		CandidatePackID: "pack-candidate",
		OutcomeKind:     missioncontrol.HotUpdateOutcomeKindHotUpdated,
		Reason:          "hot update reload/apply succeeded",
		OutcomeAt:       gate.PhaseUpdatedAt,
		CreatedAt:       gate.PhaseUpdatedAt.Add(time.Minute),
		CreatedBy:       "operator",
	}
	if mutate != nil {
		mutate(&record)
	}
	if err := missioncontrol.StoreHotUpdateOutcomeRecord(root, record); err != nil {
		t.Fatalf("StoreHotUpdateOutcomeRecord() error = %v", err)
	}
	return record.OutcomeID
}

func writeLoopHotUpdateOutcomeRaw(t *testing.T, root string, record missioncontrol.HotUpdateOutcomeRecord) {
	t.Helper()

	if record.RecordVersion == 0 {
		record.RecordVersion = 1
	}
	if err := missioncontrol.WriteStoreJSONAtomic(missioncontrol.StoreHotUpdateOutcomePath(root, record.OutcomeID), record); err != nil {
		t.Fatalf("WriteStoreJSONAtomic(hot-update outcome) error = %v", err)
	}
}

func writeLoopHotUpdateGateRaw(t *testing.T, root string, record missioncontrol.HotUpdateGateRecord) {
	t.Helper()

	if record.RecordVersion == 0 {
		record.RecordVersion = 1
	}
	if err := missioncontrol.WriteStoreJSONAtomic(missioncontrol.StoreHotUpdateGatePath(root, record.HotUpdateID), record); err != nil {
		t.Fatalf("WriteStoreJSONAtomic(hot-update gate) error = %v", err)
	}
}

func mustStoreLoopRuntimePack(t *testing.T, root string, packID string) {
	t.Helper()

	now := time.Date(2026, 4, 22, 12, 0, 0, 0, time.UTC)
	if err := missioncontrol.StoreRuntimePackRecord(root, missioncontrol.RuntimePackRecord{
		PackID:                   packID,
		ParentPackID:             "pack-base",
		RollbackTargetPackID:     "pack-base",
		CreatedAt:                now.Add(time.Minute),
		Channel:                  "phone",
		PromptPackRef:            "prompt-pack-" + packID,
		SkillPackRef:             "skill-pack-" + packID,
		ManifestRef:              "manifest-" + packID,
		ExtensionPackRef:         "extension-" + packID,
		PolicyRef:                "policy-" + packID,
		SourceSummary:            "extra pack",
		MutableSurfaces:          []string{"skills"},
		ImmutableSurfaces:        []string{"policy", "authority"},
		SurfaceClasses:           []string{"class_1"},
		CompatibilityContractRef: "compat-v1",
	}); err != nil {
		t.Fatalf("StoreRuntimePackRecord(%s) error = %v", packID, err)
	}
}

type loopHotUpdatePromotionCreateSideEffects struct {
	activePointerBytes    []byte
	lastKnownGoodBytes    []byte
	hotUpdateGateBytes    []byte
	hotUpdateOutcomeBytes []byte
	reloadGeneration      uint64
	hotUpdateGateRecords  int
	hotUpdateOutcomes     int
}

func snapshotLoopHotUpdatePromotionCreateSideEffects(t *testing.T, root string, hotUpdateID string, outcomeID string) loopHotUpdatePromotionCreateSideEffects {
	t.Helper()

	activePointerBytes, err := os.ReadFile(missioncontrol.StoreActiveRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(active pointer before) error = %v", err)
	}
	lastKnownGoodBytes, err := os.ReadFile(missioncontrol.StoreLastKnownGoodRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(last-known-good pointer before) error = %v", err)
	}
	hotUpdateGateBytes, err := os.ReadFile(missioncontrol.StoreHotUpdateGatePath(root, hotUpdateID))
	if err != nil {
		t.Fatalf("ReadFile(hot-update gate before) error = %v", err)
	}
	hotUpdateOutcomeBytes, err := os.ReadFile(missioncontrol.StoreHotUpdateOutcomePath(root, outcomeID))
	if err != nil {
		t.Fatalf("ReadFile(hot-update outcome before) error = %v", err)
	}
	pointer, err := missioncontrol.LoadActiveRuntimePackPointer(root)
	if err != nil {
		t.Fatalf("LoadActiveRuntimePackPointer() error = %v", err)
	}
	gates, err := missioncontrol.ListHotUpdateGateRecords(root)
	if err != nil {
		t.Fatalf("ListHotUpdateGateRecords() error = %v", err)
	}
	outcomes, err := missioncontrol.ListHotUpdateOutcomeRecords(root)
	if err != nil {
		t.Fatalf("ListHotUpdateOutcomeRecords() error = %v", err)
	}
	return loopHotUpdatePromotionCreateSideEffects{
		activePointerBytes:    activePointerBytes,
		lastKnownGoodBytes:    lastKnownGoodBytes,
		hotUpdateGateBytes:    hotUpdateGateBytes,
		hotUpdateOutcomeBytes: hotUpdateOutcomeBytes,
		reloadGeneration:      pointer.ReloadGeneration,
		hotUpdateGateRecords:  len(gates),
		hotUpdateOutcomes:     len(outcomes),
	}
}

func assertLoopHotUpdatePromotionCreateSideEffectsUnchanged(t *testing.T, root string, hotUpdateID string, outcomeID string, before loopHotUpdatePromotionCreateSideEffects) {
	t.Helper()

	after := snapshotLoopHotUpdatePromotionCreateSideEffects(t, root, hotUpdateID, outcomeID)
	if string(before.activePointerBytes) != string(after.activePointerBytes) {
		t.Fatalf("active runtime pack pointer changed during promotion create\nbefore:\n%s\nafter:\n%s", string(before.activePointerBytes), string(after.activePointerBytes))
	}
	if after.reloadGeneration != before.reloadGeneration {
		t.Fatalf("LoadActiveRuntimePackPointer().ReloadGeneration = %d, want %d", after.reloadGeneration, before.reloadGeneration)
	}
	if string(before.lastKnownGoodBytes) != string(after.lastKnownGoodBytes) {
		t.Fatalf("last-known-good pointer changed during promotion create\nbefore:\n%s\nafter:\n%s", string(before.lastKnownGoodBytes), string(after.lastKnownGoodBytes))
	}
	if string(before.hotUpdateGateBytes) != string(after.hotUpdateGateBytes) {
		t.Fatalf("hot-update gate changed during promotion create\nbefore:\n%s\nafter:\n%s", string(before.hotUpdateGateBytes), string(after.hotUpdateGateBytes))
	}
	if string(before.hotUpdateOutcomeBytes) != string(after.hotUpdateOutcomeBytes) {
		t.Fatalf("hot-update outcome changed during promotion create\nbefore:\n%s\nafter:\n%s", string(before.hotUpdateOutcomeBytes), string(after.hotUpdateOutcomeBytes))
	}
	if after.hotUpdateGateRecords != before.hotUpdateGateRecords {
		t.Fatalf("ListHotUpdateGateRecords(after) len = %d, want %d", after.hotUpdateGateRecords, before.hotUpdateGateRecords)
	}
	if after.hotUpdateOutcomes != before.hotUpdateOutcomes {
		t.Fatalf("ListHotUpdateOutcomeRecords(after) len = %d, want %d", after.hotUpdateOutcomes, before.hotUpdateOutcomes)
	}
}

func assertLoopHotUpdatePromotionCount(t *testing.T, root string, want int) {
	t.Helper()

	promotions, err := missioncontrol.ListPromotionRecords(root)
	if err != nil && !os.IsNotExist(err) {
		t.Fatalf("ListPromotionRecords() error = %v", err)
	}
	if len(promotions) != want {
		t.Fatalf("ListPromotionRecords() len = %d, want %d", len(promotions), want)
	}
}

func storeLoopHotUpdatePromotionForLKGRecertifyWithMutation(t *testing.T, root string, mutateOutcome func(*missioncontrol.HotUpdateOutcomeRecord), mutatePromotion func(*missioncontrol.PromotionRecord)) string {
	t.Helper()

	outcomeID := storeLoopHotUpdateOutcomeForPromotionWithMutation(t, root, mutateOutcome)
	outcome, err := missioncontrol.LoadHotUpdateOutcomeRecord(root, outcomeID)
	if err != nil {
		t.Fatalf("LoadHotUpdateOutcomeRecord() error = %v", err)
	}
	record := missioncontrol.PromotionRecord{
		PromotionID:          "hot-update-promotion-hot-update-1",
		PromotedPackID:       "pack-candidate",
		PreviousActivePackID: "pack-base",
		HotUpdateID:          "hot-update-1",
		OutcomeID:            outcomeID,
		Reason:               "hot update outcome promoted",
		PromotedAt:           outcome.OutcomeAt,
		CreatedAt:            outcome.OutcomeAt.Add(time.Minute),
		CreatedBy:            "operator",
	}
	if mutatePromotion != nil {
		mutatePromotion(&record)
	}
	if err := missioncontrol.StorePromotionRecord(root, record); err != nil {
		t.Fatalf("StorePromotionRecord() error = %v", err)
	}
	if err := missioncontrol.StoreActiveRuntimePackPointer(root, missioncontrol.ActiveRuntimePackPointer{
		ActivePackID:         "pack-candidate",
		PreviousActivePackID: "pack-base",
		LastKnownGoodPackID:  "pack-base",
		UpdatedAt:            outcome.OutcomeAt.Add(2 * time.Minute),
		UpdatedBy:            "operator",
		UpdateRecordRef:      "hot_update_gate:hot-update-1",
		ReloadGeneration:     7,
	}); err != nil {
		t.Fatalf("StoreActiveRuntimePackPointer(candidate) error = %v", err)
	}
	return record.PromotionID
}

type loopHotUpdateLKGRecertifySideEffects struct {
	activePointerBytes    []byte
	lastKnownGoodBytes    []byte
	hotUpdateGateBytes    []byte
	hotUpdateOutcomeBytes []byte
	promotionBytes        []byte
	reloadGeneration      uint64
	hotUpdateGateRecords  int
	hotUpdateOutcomes     int
	promotions            int
}

func snapshotLoopHotUpdateLKGRecertifySideEffects(t *testing.T, root string, hotUpdateID string, outcomeID string, promotionID string) loopHotUpdateLKGRecertifySideEffects {
	t.Helper()

	activePointerBytes, err := os.ReadFile(missioncontrol.StoreActiveRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(active pointer) error = %v", err)
	}
	lastKnownGoodBytes, err := os.ReadFile(missioncontrol.StoreLastKnownGoodRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(last-known-good pointer) error = %v", err)
	}
	hotUpdateGateBytes, err := os.ReadFile(missioncontrol.StoreHotUpdateGatePath(root, hotUpdateID))
	if err != nil {
		t.Fatalf("ReadFile(hot-update gate) error = %v", err)
	}
	hotUpdateOutcomeBytes, err := os.ReadFile(missioncontrol.StoreHotUpdateOutcomePath(root, outcomeID))
	if err != nil {
		t.Fatalf("ReadFile(hot-update outcome) error = %v", err)
	}
	promotionBytes, err := os.ReadFile(missioncontrol.StorePromotionPath(root, promotionID))
	if err != nil {
		t.Fatalf("ReadFile(promotion) error = %v", err)
	}
	pointer, err := missioncontrol.LoadActiveRuntimePackPointer(root)
	if err != nil {
		t.Fatalf("LoadActiveRuntimePackPointer() error = %v", err)
	}
	gates, err := missioncontrol.ListHotUpdateGateRecords(root)
	if err != nil {
		t.Fatalf("ListHotUpdateGateRecords() error = %v", err)
	}
	outcomes, err := missioncontrol.ListHotUpdateOutcomeRecords(root)
	if err != nil {
		t.Fatalf("ListHotUpdateOutcomeRecords() error = %v", err)
	}
	promotions, err := missioncontrol.ListPromotionRecords(root)
	if err != nil {
		t.Fatalf("ListPromotionRecords() error = %v", err)
	}
	return loopHotUpdateLKGRecertifySideEffects{
		activePointerBytes:    activePointerBytes,
		lastKnownGoodBytes:    lastKnownGoodBytes,
		hotUpdateGateBytes:    hotUpdateGateBytes,
		hotUpdateOutcomeBytes: hotUpdateOutcomeBytes,
		promotionBytes:        promotionBytes,
		reloadGeneration:      pointer.ReloadGeneration,
		hotUpdateGateRecords:  len(gates),
		hotUpdateOutcomes:     len(outcomes),
		promotions:            len(promotions),
	}
}

func assertLoopHotUpdateLKGRecertifySideEffectsUnchangedExceptLKG(t *testing.T, root string, hotUpdateID string, outcomeID string, promotionID string, before loopHotUpdateLKGRecertifySideEffects) {
	t.Helper()

	after := snapshotLoopHotUpdateLKGRecertifySideEffects(t, root, hotUpdateID, outcomeID, promotionID)
	assertLoopHotUpdateLKGRecertifyStableFields(t, before, after)
}

func assertLoopHotUpdateLKGRecertifySideEffectsFullyUnchanged(t *testing.T, root string, hotUpdateID string, outcomeID string, promotionID string, before loopHotUpdateLKGRecertifySideEffects) {
	t.Helper()

	after := snapshotLoopHotUpdateLKGRecertifySideEffects(t, root, hotUpdateID, outcomeID, promotionID)
	assertLoopHotUpdateLKGRecertifyStableFields(t, before, after)
	if string(before.lastKnownGoodBytes) != string(after.lastKnownGoodBytes) {
		t.Fatalf("last-known-good pointer changed\nbefore:\n%s\nafter:\n%s", string(before.lastKnownGoodBytes), string(after.lastKnownGoodBytes))
	}
}

func assertLoopHotUpdateLKGRecertifyStableFields(t *testing.T, before loopHotUpdateLKGRecertifySideEffects, after loopHotUpdateLKGRecertifySideEffects) {
	t.Helper()

	if string(before.activePointerBytes) != string(after.activePointerBytes) {
		t.Fatalf("active runtime pack pointer changed during LKG recertify\nbefore:\n%s\nafter:\n%s", string(before.activePointerBytes), string(after.activePointerBytes))
	}
	if after.reloadGeneration != before.reloadGeneration {
		t.Fatalf("LoadActiveRuntimePackPointer().ReloadGeneration = %d, want %d", after.reloadGeneration, before.reloadGeneration)
	}
	if string(before.hotUpdateGateBytes) != string(after.hotUpdateGateBytes) {
		t.Fatalf("hot-update gate changed during LKG recertify\nbefore:\n%s\nafter:\n%s", string(before.hotUpdateGateBytes), string(after.hotUpdateGateBytes))
	}
	if string(before.hotUpdateOutcomeBytes) != string(after.hotUpdateOutcomeBytes) {
		t.Fatalf("hot-update outcome changed during LKG recertify\nbefore:\n%s\nafter:\n%s", string(before.hotUpdateOutcomeBytes), string(after.hotUpdateOutcomeBytes))
	}
	if string(before.promotionBytes) != string(after.promotionBytes) {
		t.Fatalf("promotion changed during LKG recertify\nbefore:\n%s\nafter:\n%s", string(before.promotionBytes), string(after.promotionBytes))
	}
	if after.hotUpdateGateRecords != before.hotUpdateGateRecords {
		t.Fatalf("ListHotUpdateGateRecords(after) len = %d, want %d", after.hotUpdateGateRecords, before.hotUpdateGateRecords)
	}
	if after.hotUpdateOutcomes != before.hotUpdateOutcomes {
		t.Fatalf("ListHotUpdateOutcomeRecords(after) len = %d, want %d", after.hotUpdateOutcomes, before.hotUpdateOutcomes)
	}
	if after.promotions != before.promotions {
		t.Fatalf("ListPromotionRecords(after) len = %d, want %d", after.promotions, before.promotions)
	}
}

func writeLoopHotUpdatePromotionRaw(t *testing.T, root string, record missioncontrol.PromotionRecord) {
	t.Helper()

	if record.RecordVersion == 0 {
		record.RecordVersion = 1
	}
	if err := missioncontrol.WriteStoreJSONAtomic(missioncontrol.StorePromotionPath(root, record.PromotionID), record); err != nil {
		t.Fatalf("WriteStoreJSONAtomic(promotion) error = %v", err)
	}
}

func readLoopOptionalFile(t *testing.T, path string) ([]byte, bool) {
	t.Helper()

	bytes, err := os.ReadFile(path)
	if err == nil {
		return bytes, true
	}
	if os.IsNotExist(err) {
		return nil, false
	}
	t.Fatalf("ReadFile(%s) error = %v", path, err)
	return nil, false
}

func testMissionJob(jobAllowedTools, stepAllowedTools []string) missioncontrol.Job {
	return missioncontrol.Job{
		ID:           "job-1",
		MaxAuthority: missioncontrol.AuthorityTierHigh,
		AllowedTools: append([]string(nil), jobAllowedTools...),
		Plan: missioncontrol.Plan{
			ID: "plan-1",
			Steps: []missioncontrol.Step{
				{
					ID:                "build",
					Type:              missioncontrol.StepTypeOneShotCode,
					RequiredAuthority: missioncontrol.AuthorityTierLow,
					AllowedTools:      append([]string(nil), stepAllowedTools...),
					SuccessCriteria:   []string{"produce code"},
				},
				{
					ID:           "final",
					Type:         missioncontrol.StepTypeFinalResponse,
					DependsOn:    []string{"build"},
					AllowedTools: append([]string(nil), stepAllowedTools...),
				},
			},
		},
	}
}

func testDiscussionMissionJob() missioncontrol.Job {
	return missioncontrol.Job{
		ID:           "job-1",
		MaxAuthority: missioncontrol.AuthorityTierHigh,
		AllowedTools: []string{"write_memory"},
		Plan: missioncontrol.Plan{
			ID: "plan-1",
			Steps: []missioncontrol.Step{
				{
					ID:      "build",
					Type:    missioncontrol.StepTypeDiscussion,
					Subtype: missioncontrol.StepSubtypeAuthorization,
				},
				{
					ID:        "final",
					Type:      missioncontrol.StepTypeFinalResponse,
					DependsOn: []string{"build"},
				},
			},
		},
	}
}

func testPendingApprovalBudgetPausedDiscussionRuntime(t *testing.T) (missioncontrol.Job, missioncontrol.JobRuntimeState, missioncontrol.RuntimeControlContext) {
	t.Helper()

	job := testDiscussionMissionJob()
	control, err := missioncontrol.BuildRuntimeControlContext(job, "build")
	if err != nil {
		t.Fatalf("BuildRuntimeControlContext() error = %v", err)
	}

	now := time.Now().UTC()
	runtimeState := missioncontrol.JobRuntimeState{
		JobID:        job.ID,
		State:        missioncontrol.JobStatePaused,
		ActiveStepID: "build",
		PausedReason: missioncontrol.RuntimePauseReasonBudgetExhausted,
		PausedAt:     now,
		BudgetBlocker: &missioncontrol.RuntimeBudgetBlockerRecord{
			Ceiling:     "pending_approvals",
			Limit:       3,
			Observed:    4,
			Message:     "pending approval request budget exhausted",
			TriggeredAt: now,
		},
		ApprovalRequests: []missioncontrol.ApprovalRequest{
			{JobID: job.ID, StepID: "authorize-1", RequestedAction: missioncontrol.ApprovalRequestedActionStepComplete, Scope: missioncontrol.ApprovalScopeMissionStep, State: missioncontrol.ApprovalStatePending, RequestedVia: missioncontrol.ApprovalRequestedViaRuntime, Reason: "discussion_authorization", RequestedAt: now.Add(-3 * time.Minute), ExpiresAt: now.Add(2 * time.Minute)},
			{JobID: job.ID, StepID: "authorize-2", RequestedAction: missioncontrol.ApprovalRequestedActionStepComplete, Scope: missioncontrol.ApprovalScopeMissionStep, State: missioncontrol.ApprovalStatePending, RequestedVia: missioncontrol.ApprovalRequestedViaRuntime, Reason: "discussion_authorization", RequestedAt: now.Add(-2 * time.Minute), ExpiresAt: now.Add(2 * time.Minute)},
			{JobID: job.ID, StepID: "authorize-3", RequestedAction: missioncontrol.ApprovalRequestedActionStepComplete, Scope: missioncontrol.ApprovalScopeMissionStep, State: missioncontrol.ApprovalStatePending, RequestedVia: missioncontrol.ApprovalRequestedViaRuntime, Reason: "discussion_authorization", RequestedAt: now.Add(-time.Minute), ExpiresAt: now.Add(2 * time.Minute)},
			{JobID: job.ID, StepID: "build", RequestedAction: missioncontrol.ApprovalRequestedActionStepComplete, Scope: missioncontrol.ApprovalScopeMissionStep, State: missioncontrol.ApprovalStatePending, RequestedVia: missioncontrol.ApprovalRequestedViaRuntime, Reason: "discussion_authorization", RequestedAt: now, ExpiresAt: now.Add(2 * time.Minute)},
		},
	}

	return job, runtimeState, control
}

func testReusableApprovalMissionJob(scope string) missioncontrol.Job {
	return missioncontrol.Job{
		ID:           "job-1",
		MaxAuthority: missioncontrol.AuthorityTierHigh,
		AllowedTools: []string{"read"},
		Plan: missioncontrol.Plan{
			ID: "plan-1",
			Steps: []missioncontrol.Step{
				{
					ID:            "authorize-1",
					Type:          missioncontrol.StepTypeDiscussion,
					Subtype:       missioncontrol.StepSubtypeAuthorization,
					ApprovalScope: scope,
				},
				{
					ID:            "authorize-2",
					Type:          missioncontrol.StepTypeDiscussion,
					Subtype:       missioncontrol.StepSubtypeAuthorization,
					ApprovalScope: scope,
					DependsOn:     []string{"authorize-1"},
				},
				{
					ID:        "final",
					Type:      missioncontrol.StepTypeFinalResponse,
					DependsOn: []string{"authorize-2"},
				},
			},
		},
	}
}

func testFinalMissionJob() missioncontrol.Job {
	return missioncontrol.Job{
		ID:           "job-1",
		MaxAuthority: missioncontrol.AuthorityTierHigh,
		AllowedTools: []string{"write_memory"},
		Plan: missioncontrol.Plan{
			ID: "plan-1",
			Steps: []missioncontrol.Step{
				{
					ID:   "build",
					Type: missioncontrol.StepTypeOneShotCode,
				},
				{
					ID:        "final",
					Type:      missioncontrol.StepTypeFinalResponse,
					DependsOn: []string{"build"},
				},
			},
		},
	}
}

func testFilesystemMissionJob() missioncontrol.Job {
	return missioncontrol.Job{
		ID:           "job-1",
		MaxAuthority: missioncontrol.AuthorityTierHigh,
		AllowedTools: []string{"filesystem"},
		Plan: missioncontrol.Plan{
			ID: "plan-1",
			Steps: []missioncontrol.Step{
				{
					ID:                  "build",
					Type:                missioncontrol.StepTypeOneShotCode,
					AllowedTools:        []string{"filesystem"},
					OneShotArtifactPath: "result.txt",
				},
				{
					ID:        "final",
					Type:      missioncontrol.StepTypeFinalResponse,
					DependsOn: []string{"build"},
				},
			},
		},
	}
}

func testStaticArtifactMissionJob() missioncontrol.Job {
	return missioncontrol.Job{
		ID:           "job-1",
		MaxAuthority: missioncontrol.AuthorityTierHigh,
		AllowedTools: []string{"filesystem"},
		Plan: missioncontrol.Plan{
			ID: "plan-1",
			Steps: []missioncontrol.Step{
				{
					ID:                   "build",
					Type:                 missioncontrol.StepTypeStaticArtifact,
					AllowedTools:         []string{"filesystem"},
					SuccessCriteria:      []string{"Write `report.json` as valid JSON."},
					StaticArtifactPath:   "report.json",
					StaticArtifactFormat: "json",
				},
				{
					ID:        "final",
					Type:      missioncontrol.StepTypeFinalResponse,
					DependsOn: []string{"build"},
				},
			},
		},
	}
}

func testWaitUserMissionJob() missioncontrol.Job {
	return missioncontrol.Job{
		ID:           "job-1",
		SpecVersion:  missioncontrol.JobSpecVersionV2,
		MaxAuthority: missioncontrol.AuthorityTierHigh,
		AllowedTools: []string{"read"},
		Plan: missioncontrol.Plan{
			ID: "plan-1",
			Steps: []missioncontrol.Step{
				{
					ID:      "build",
					Type:    missioncontrol.StepTypeWaitUser,
					Subtype: missioncontrol.StepSubtypeAuthorization,
				},
				{
					ID:        "final",
					Type:      missioncontrol.StepTypeFinalResponse,
					DependsOn: []string{"build"},
				},
			},
		},
	}
}

func testLongRunningMissionJob() missioncontrol.Job {
	return missioncontrol.Job{
		ID:           "job-1",
		SpecVersion:  missioncontrol.JobSpecVersionV2,
		MaxAuthority: missioncontrol.AuthorityTierHigh,
		AllowedTools: []string{"filesystem"},
		Plan: missioncontrol.Plan{
			ID: "plan-1",
			Steps: []missioncontrol.Step{
				{
					ID:                        "build",
					Type:                      missioncontrol.StepTypeLongRunningCode,
					AllowedTools:              []string{"filesystem"},
					SuccessCriteria:           []string{"Record startup command `npm start` and verify the build artifact exists."},
					LongRunningStartupCommand: []string{"npm", "start"},
					LongRunningArtifactPath:   "result.txt",
				},
				{
					ID:        "final",
					Type:      missioncontrol.StepTypeFinalResponse,
					DependsOn: []string{"build"},
				},
			},
		},
	}
}
