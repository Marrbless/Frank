package agent

import (
	"context"
	"encoding/json"
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
	ag := NewAgentLoop(b, prov, prov.GetDefaultModel(), 5, "", nil)

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

	if !strings.Contains(resp, string(missioncontrol.RejectionCodeToolNotAllowed)) {
		t.Fatalf("expected rejection code in response, got %q", resp)
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
	if len(audits) != 1 {
		t.Fatalf("AuditEvents() count = %d, want %d", len(audits), 1)
	}
	assertAuditEvent(t, audits[0], "job-1", "build", "message", false, missioncontrol.RejectionCode("E_INVALID_ACTION_FOR_STEP"))
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

	if !strings.Contains(resp, string(missioncontrol.RejectionCodeMissionContextRequired)) {
		t.Fatalf("expected mission_context_required in response, got %q", resp)
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
	if len(audits) != 1 {
		t.Fatalf("AuditEvents() count = %d, want %d", len(audits), 1)
	}
	assertAuditEvent(t, audits[0], "job-1", "build", "write_memory", true, "")
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
	if resp != finalProv.content {
		t.Fatalf("final ProcessDirect() response = %q, want %q", resp, finalProv.content)
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
	if resp != finalProv.content {
		t.Fatalf("final ProcessDirect() response = %q, want %q", resp, finalProv.content)
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
	if len(runtime.AuditHistory) != 3 {
		t.Fatalf("MissionRuntimeState().AuditHistory count = %d, want %d", len(runtime.AuditHistory), 3)
	}
	assertAuditEvent(t, runtime.AuditHistory[0], "job-1", "build", "pause", true, "")
	assertAuditEvent(t, runtime.AuditHistory[1], "job-1", "build", "resume", true, "")
	assertAuditEvent(t, runtime.AuditHistory[2], "job-1", "build", "abort", true, "")

	audits := ag.taskState.AuditEvents()
	if len(audits) != 3 {
		t.Fatalf("AuditEvents() count = %d, want %d", len(audits), 3)
	}
	assertAuditEvent(t, audits[0], "job-1", "build", "pause", true, "")
	assertAuditEvent(t, audits[1], "job-1", "build", "resume", true, "")
	assertAuditEvent(t, audits[2], "job-1", "build", "abort", true, "")
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
	if !strings.Contains(err.Error(), string(missioncontrol.RejectionCodeUnknownStep)) {
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
					ID:           "build",
					Type:         missioncontrol.StepTypeOneShotCode,
					AllowedTools: []string{"filesystem"},
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
					ID:              "build",
					Type:            missioncontrol.StepTypeStaticArtifact,
					AllowedTools:    []string{"filesystem"},
					SuccessCriteria: []string{"Write `report.json` as valid JSON."},
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
