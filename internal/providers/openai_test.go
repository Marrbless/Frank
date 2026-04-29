package providers

import (
	"bytes"
	"context"
	"errors"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestOpenAIFunctionCallParsing(t *testing.T) {
	// Build a fake server that returns a tool_calls style response
	h := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write([]byte(`{
		  "choices": [
		    {
		      "message": {
		        "role": "assistant",
		        "content": "",
		        "tool_calls": [
		          {
		            "id": "call_001",
		            "type": "function",
		            "function": {
		              "name": "message",
		              "arguments": "{\"content\": \"Hello from function\"}"
		            }
		          }
		        ]
		      }
		    }
		  ]
		}`))
	}))
	defer h.Close()

	p := NewOpenAIProvider("test-key", h.URL, 60, 0)
	p.Client = &http.Client{Timeout: 5 * time.Second}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	msgs := []Message{{Role: "user", Content: "trigger"}}
	resp, err := p.Chat(ctx, msgs, nil, "model-x")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !resp.HasToolCalls || len(resp.ToolCalls) != 1 {
		t.Fatalf("expected one tool call, got: has=%v len=%d", resp.HasToolCalls, len(resp.ToolCalls))
	}
	if resp.ToolCalls[0].Name != "message" {
		t.Fatalf("expected tool name 'message', got '%s'", resp.ToolCalls[0].Name)
	}
	if resp.ToolCalls[0].Arguments["content"] != "Hello from function" {
		t.Fatalf("unexpected argument content: %v", resp.ToolCalls[0].Arguments)
	}
}

func TestOpenAIDoJSONRedactsNon2xxBody(t *testing.T) {
	h := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Request-Id", "req_123")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"bad auth","token":"sk-secret","prompt":"private note"}`))
	}))
	defer h.Close()

	p := NewOpenAIProvider("test-key", h.URL, 60, 0)
	p.Client = &http.Client{Timeout: 5 * time.Second}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	logs := captureOpenAILogs(t, func() {
		_, err := p.doJSON(ctx, h.URL, []byte(`{"input":"hello"}`))
		if err == nil {
			t.Fatal("doJSON() error = nil, want non-2xx error")
		}
		if got, want := err.Error(), "OpenAI API error: 401 Unauthorized (request id: req_123)"; got != want {
			t.Fatalf("doJSON() error = %q, want %q", got, want)
		}
	})

	if !strings.Contains(logs, "OpenAI API non-2xx: status=401 Unauthorized request_id=req_123") {
		t.Fatalf("expected redacted non-2xx log, got %q", logs)
	}
	if strings.Contains(logs, "sk-secret") || strings.Contains(logs, "private note") {
		t.Fatalf("expected logs to redact provider body, got %q", logs)
	}
}

func TestOpenAIDoJSONRateLimitHonorsRetryAfterCooldown(t *testing.T) {
	var calls int32
	h := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.Header().Set("X-Request-Id", "req_rate_limited")
		w.Header().Set("Retry-After", "3")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":"rate limit"}`))
	}))
	defer h.Close()

	p := NewOpenAIProvider("test-key", h.URL, 60, 0)
	p.Client = &http.Client{Timeout: 5 * time.Second}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := p.doJSON(ctx, h.URL, []byte(`{"input":"hello"}`))
	var firstRateLimit *RateLimitError
	if err == nil || !errors.As(err, &firstRateLimit) {
		t.Fatalf("first doJSON() error = %v, want RateLimitError", err)
	}
	if firstRateLimit.RequestID != "req_rate_limited" {
		t.Fatalf("first RateLimitError.RequestID = %q, want req_rate_limited", firstRateLimit.RequestID)
	}
	if firstRateLimit.RetryAfter <= 0 {
		t.Fatalf("first RateLimitError.RetryAfter = %v, want positive duration", firstRateLimit.RetryAfter)
	}

	_, err = p.doJSON(ctx, h.URL, []byte(`{"input":"hello again"}`))
	var secondRateLimit *RateLimitError
	if err == nil || !errors.As(err, &secondRateLimit) {
		t.Fatalf("second doJSON() error = %v, want RateLimitError", err)
	}
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("provider calls = %d, want 1 because second call should use local cooldown", got)
	}
}

func captureOpenAILogs(t *testing.T, fn func()) string {
	t.Helper()

	var buf bytes.Buffer
	previousWriter := log.Writer()
	log.SetOutput(&buf)
	defer log.SetOutput(previousWriter)

	fn()
	return buf.String()
}
