package agent

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"strings"

	"github.com/local/picobot/internal/chat"
	"github.com/local/picobot/internal/providers"
)

// Provider that fails the test if called (ensures remember shortcut skips provider)
type FailingProvider struct{}

func (f *FailingProvider) Chat(ctx context.Context, messages []providers.Message, tools []providers.ToolDefinition, model string) (providers.LLMResponse, error) {
	panic("Chat should not be called when handling remember messages")
}
func (f *FailingProvider) GetDefaultModel() string { return "fail" }

func TestAgentRemembersToday(t *testing.T) {
	b := chat.NewHub(10)
	p := &FailingProvider{}
	ag := NewAgentLoop(b, p, p.GetDefaultModel(), 5, "", nil, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	go ag.Run(ctx)

	in := chat.Inbound{Channel: "cli", SenderID: "user", ChatID: "one", Content: "Remember to buy milk"}
	select {
	case b.In <- in:
	default:
		t.Fatalf("couldn't send inbound")
	}

	deadline := time.After(1 * time.Second)
	for {
		select {
		case out := <-b.Out:
			if out.Content == "OK, I've remembered that." {
				// success; verify today's file contains the note
				memCtx, _ := ag.memory.ReadToday()
				if memCtx == "" || !strings.Contains(memCtx, "buy milk") {
					t.Fatalf("expected today's memory to contain 'buy milk', got %q", memCtx)
				}
				return
			}
		case <-deadline:
			t.Fatalf("timeout waiting for remember confirmation")
		}
	}
}

func TestAgentRememberFailsClosedWhenAppendTodayFails(t *testing.T) {
	workspace := t.TempDir()
	blockingPath := filepath.Join(workspace, "memory")
	if err := os.WriteFile(blockingPath, []byte("not-a-directory"), 0o644); err != nil {
		t.Fatalf("WriteFile(memory blocker) error = %v", err)
	}

	b := chat.NewHub(10)
	p := &FailingProvider{}
	ag := NewAgentLoop(b, p, p.GetDefaultModel(), 5, workspace, nil, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	go ag.Run(ctx)

	in := chat.Inbound{Channel: "cli", SenderID: "user", ChatID: "one", Content: "Remember to buy milk"}
	select {
	case b.In <- in:
	default:
		t.Fatalf("couldn't send inbound")
	}

	deadline := time.After(1 * time.Second)
	for {
		select {
		case out := <-b.Out:
			if out.Content == "I couldn't remember that because saving memory failed." {
				if out.Content == "OK, I've remembered that." {
					t.Fatal("remember shortcut reported success, want fail-closed response")
				}
				sess := ag.sessions.GetOrCreate("cli:one")
				if len(sess.History) != 2 {
					t.Fatalf("len(sess.History) = %d, want 2", len(sess.History))
				}
				if sess.History[1] != "assistant: I couldn't remember that because saving memory failed." {
					t.Fatalf("sess.History[1] = %q, want failure response recorded", sess.History[1])
				}
				return
			}
		case <-deadline:
			t.Fatalf("timeout waiting for remember failure confirmation")
		}
	}
}
