package agent

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/local/picobot/internal/chat"
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

func TestAgentExecutesToolCall(t *testing.T) {
	b := chat.NewHub(10)
	p := &FakeProvider{}
	ag := NewAgentLoop(b, p, p.GetDefaultModel(), 3, "", nil)

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

func toolDefinitionNames(defs []providers.ToolDefinition) []string {
	names := make([]string, 0, len(defs))
	for _, def := range defs {
		names = append(names, def.Name)
	}
	return names
}
