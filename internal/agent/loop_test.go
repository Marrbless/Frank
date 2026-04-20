package agent

import (
	"testing"
	"time"

	"github.com/local/picobot/internal/chat"
	"github.com/local/picobot/internal/providers"
	"github.com/local/picobot/internal/session"
)

func TestProcessDirectWithStub(t *testing.T) {
	b := chat.NewHub(10)
	p := providers.NewStubProvider()

	ag := NewAgentLoop(b, p, p.GetDefaultModel(), 5, "", nil, nil)

	resp, err := ag.ProcessDirect("hello", 1*time.Second)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp == "" {
		t.Fatalf("expected response, got empty string")
	}
}

func TestNewAgentLoopRehydratesSavedSessionsOnStartup(t *testing.T) {
	workspace := t.TempDir()
	sm := session.NewSessionManager(workspace)
	sess := sm.GetOrCreate("telegram-chat-42")
	sess.AddMessage("user", "saved before restart")
	if err := sm.Save(sess); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	b := chat.NewHub(10)
	p := providers.NewStubProvider()
	ag := NewAgentLoop(b, p, p.GetDefaultModel(), 5, workspace, nil, nil)

	got := ag.sessions.GetOrCreate("telegram-chat-42")
	if len(got.History) != 1 {
		t.Fatalf("len(got.History) = %d, want 1", len(got.History))
	}
	if got.History[0] != "user: saved before restart" {
		t.Fatalf("got.History = %#v, want rehydrated session history", got.History)
	}
}
