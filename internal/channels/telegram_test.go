package channels

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/local/picobot/internal/chat"
)

func TestStartTelegramWithBase(t *testing.T) {
	token := "testtoken"
	// channel to capture sendMessage posts
	sent := make(chan url.Values, 4)

	// simple stateful handler: first getUpdates returns one update, subsequent return empty
	first := true
	h := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if strings.HasSuffix(path, "/getUpdates") {
			w.Header().Set("Content-Type", "application/json")
			if first {
				first = false
				w.Write([]byte(`{"ok":true,"result":[{"update_id":1,"message":{"message_id":1,"from":{"id":123},"chat":{"id":456,"type":"private"},"text":"hello"}}]}`))
				return
			}
			w.Write([]byte(`{"ok":true,"result":[]}`))
			return
		}
		if strings.HasSuffix(path, "/sendMessage") {
			if err := r.ParseForm(); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			sent <- r.PostForm
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"ok":true,"result":{}}`))
			return
		}
		w.WriteHeader(404)
	}))
	defer h.Close()

	base := h.URL + "/bot" + token
	b := chat.NewHub(10)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := StartTelegramWithBaseOpenMode(ctx, b, token, base, nil, true); err != nil {
		t.Fatalf("StartTelegramWithBase failed: %v", err)
	}
	// Start the hub router so outbound messages sent to b.Out are dispatched
	// to each channel's subscription (telegram in this test).
	b.StartRouter(ctx)

	// Wait for inbound from getUpdates
	select {
	case msg := <-b.In:
		if msg.Content != "hello" {
			t.Fatalf("unexpected inbound content: %s", msg.Content)
		}
		if msg.ChatID != "456" {
			t.Fatalf("unexpected chat id: %s", msg.ChatID)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for inbound message")
	}

	// send an outbound message and ensure server receives it
	out := chat.Outbound{Channel: "telegram", ChatID: "456", Content: "reply"}
	b.Out <- out

	select {
	case v := <-sent:
		if v.Get("chat_id") != "456" || v.Get("text") != "reply" {
			t.Fatalf("unexpected sendMessage form: %v", v)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for sendMessage to be posted")
	}

	// cancel and allow goroutines to stop
	cancel()
	// give a small grace period
	time.Sleep(50 * time.Millisecond)
}

func TestStartTelegramWithBaseFailsClosedWithoutAllowlist(t *testing.T) {
	err := StartTelegramWithBase(context.Background(), chat.NewHub(10), "token", "https://example.com/bottoken", nil)
	if err == nil {
		t.Fatal("expected empty allowlist to fail closed")
	}
	if !strings.Contains(err.Error(), "allowlist is empty") {
		t.Fatalf("expected allowlist error, got %v", err)
	}
}

func TestReadTelegramBotIdentityWithBase(t *testing.T) {
	t.Parallel()

	token := "testtoken"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/getMe") {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ok":true,"result":{"id":123456789,"is_bot":true,"username":"frank_owner_bot"}}`))
	}))
	defer server.Close()

	got, err := ReadTelegramBotIdentityWithBase(context.Background(), token, server.URL+"/bot"+token)
	if err != nil {
		t.Fatalf("ReadTelegramBotIdentityWithBase() error = %v", err)
	}
	if got.BotUserID != "123456789" {
		t.Fatalf("BotUserID = %q, want %q", got.BotUserID, "123456789")
	}
	if got.Username != "frank_owner_bot" {
		t.Fatalf("Username = %q, want %q", got.Username, "frank_owner_bot")
	}
}
