package agent

import (
	"context"
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

func TestProcessDirectRejectsDisallowedToolWhenExecutionContextPresent(t *testing.T) {
	t.Parallel()

	b := chat.NewHub(10)
	prov := &deniedMessageToolProvider{}
	ag := NewAgentLoop(b, prov, prov.GetDefaultModel(), 5, "", nil)
	ag.taskState.SetExecutionContext(missioncontrol.ExecutionContext{
		Job: &missioncontrol.Job{
			ID:           "job-1",
			MaxAuthority: missioncontrol.AuthorityTierHigh,
			AllowedTools: []string{"write_memory"},
		},
		Step: &missioncontrol.Step{
			ID:                "step-1",
			RequiredAuthority: missioncontrol.AuthorityTierLow,
		},
	})

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
}

func contains(s, sub string) bool { return strings.Contains(s, sub) }
