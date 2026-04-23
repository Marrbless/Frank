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
