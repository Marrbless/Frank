package providers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// OpenAIProvider calls an OpenAI-compatible API.
type OpenAIProvider struct {
	APIKey          string
	APIBase         string // e.g. https://api.openai.com/v1
	MaxTokens       int    // 0 means "let the API decide"
	UseResponses    bool
	ReasoningEffort string
	Client          *http.Client

	cooldownMu         sync.Mutex
	rateLimitUntil     time.Time
	rateLimitStatus    string
	rateLimitRequestID string
}

// RateLimitError reports an OpenAI 429 and the local cooldown window that
// Picobot will honor before attempting another provider request.
type RateLimitError struct {
	Status     string
	RequestID  string
	RetryAfter time.Duration
	Until      time.Time
}

func (e *RateLimitError) Error() string {
	status := strings.TrimSpace(e.Status)
	if status == "" {
		status = "429 Too Many Requests"
	}

	parts := []string{"OpenAI API error: " + status}
	if e.RequestID != "" {
		parts = append(parts, fmt.Sprintf("(request id: %s)", e.RequestID))
	}
	if e.RetryAfter > 0 {
		parts = append(parts, fmt.Sprintf("(retry after: %s)", e.RetryAfter.Round(time.Second)))
	}
	return strings.Join(parts, " ")
}

// Backward-compatible constructor.
func NewOpenAIProvider(apiKey, apiBase string, timeoutSecs, maxTokens int) *OpenAIProvider {
	return NewOpenAIProviderWithOptions(apiKey, apiBase, timeoutSecs, maxTokens, false, "")
}

func NewOpenAIProviderWithOptions(apiKey, apiBase string, timeoutSecs, maxTokens int, useResponses bool, reasoningEffort string) *OpenAIProvider {
	if apiBase == "" {
		apiBase = "https://api.openai.com/v1"
	}
	if timeoutSecs <= 1 {
		timeoutSecs = 60
	}
	return &OpenAIProvider{
		APIKey:          apiKey,
		APIBase:         strings.TrimRight(apiBase, "/"),
		MaxTokens:       maxTokens,
		UseResponses:    useResponses,
		ReasoningEffort: strings.TrimSpace(reasoningEffort),
		Client: &http.Client{
			Timeout: time.Duration(timeoutSecs) * time.Second,
		},
	}
}

func (p *OpenAIProvider) GetDefaultModel() string {
	return "gpt-4o-mini"
}

func (p *OpenAIProvider) Chat(ctx context.Context, messages []Message, tools []ToolDefinition, model string) (LLMResponse, error) {
	if model == "" {
		model = p.GetDefaultModel()
	}

	if p.UseResponses {
		return p.chatResponses(ctx, messages, tools, model)
	}
	return p.chatCompletions(ctx, messages, tools, model)
}

/* =========================
   Chat Completions fallback
   ========================= */

type chatCompletionRequest struct {
	Model     string        `json:"model"`
	Messages  []messageJSON `json:"messages"`
	Tools     []toolWrapper `json:"tools,omitempty"`
	MaxTokens int           `json:"max_tokens,omitempty"`
}

type toolWrapper struct {
	Type     string      `json:"type"`
	Function functionDef `json:"function"`
}

type functionDef struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
}

type messageJSON struct {
	Role       string         `json:"role"`
	Content    *string        `json:"content"`
	ToolCallID string         `json:"tool_call_id,omitempty"`
	ToolCalls  []toolCallJSON `json:"tool_calls,omitempty"`
}

type toolCallJSON struct {
	ID       string               `json:"id"`
	Type     string               `json:"type"`
	Function toolCallFunctionJSON `json:"function"`
}

type toolCallFunctionJSON struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type messageResponseJSON struct {
	Role      string         `json:"role"`
	Content   string         `json:"content"`
	ToolCalls []toolCallJSON `json:"tool_calls,omitempty"`
}

type chatCompletionResponse struct {
	Choices []struct {
		Message messageResponseJSON `json:"message"`
	} `json:"choices"`
}

func (p *OpenAIProvider) chatCompletions(ctx context.Context, messages []Message, tools []ToolDefinition, model string) (LLMResponse, error) {
	reqBody := chatCompletionRequest{
		Model:     model,
		Messages:  make([]messageJSON, 0, len(messages)),
		MaxTokens: p.MaxTokens,
	}

	for _, m := range messages {
		mj := messageJSON{Role: m.Role, ToolCallID: m.ToolCallID}
		if len(m.ToolCalls) > 0 && m.Content == "" {
			mj.Content = nil
		} else {
			c := m.Content
			mj.Content = &c
		}
		for _, tc := range m.ToolCalls {
			argsBytes, _ := json.Marshal(tc.Arguments)
			mj.ToolCalls = append(mj.ToolCalls, toolCallJSON{
				ID:   tc.ID,
				Type: "function",
				Function: toolCallFunctionJSON{
					Name:      tc.Name,
					Arguments: string(argsBytes),
				},
			})
		}
		reqBody.Messages = append(reqBody.Messages, mj)
	}

	if len(tools) > 0 {
		reqBody.Tools = buildChatCompletionTools(tools)
	}

	b, err := json.Marshal(reqBody)
	if err != nil {
		return LLMResponse{}, err
	}

	url := fmt.Sprintf("%s/chat/completions", p.APIBase)
	body, err := p.doJSON(ctx, url, b)
	if err != nil {
		return LLMResponse{}, err
	}

	var out chatCompletionResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return LLMResponse{}, err
	}
	if len(out.Choices) == 0 {
		return LLMResponse{}, errors.New("OpenAI API returned no choices")
	}

	msg := out.Choices[0].Message
	if len(msg.ToolCalls) > 0 {
		var tcs []ToolCall
		for _, tc := range msg.ToolCalls {
			var parsed map[string]interface{}
			if err := json.Unmarshal([]byte(tc.Function.Arguments), &parsed); err != nil {
				continue
			}
			tcs = append(tcs, ToolCall{
				ID:        tc.ID,
				Name:      tc.Function.Name,
				Arguments: parsed,
			})
		}
		if len(tcs) > 0 {
			return LLMResponse{
				Content:      strings.TrimSpace(msg.Content),
				HasToolCalls: true,
				ToolCalls:    tcs,
			}, nil
		}
	}

	return LLMResponse{
		Content:      strings.TrimSpace(msg.Content),
		HasToolCalls: false,
	}, nil
}

func buildChatCompletionTools(tools []ToolDefinition) []toolWrapper {
	out := make([]toolWrapper, 0, len(tools))
	for _, t := range tools {
		params := t.Parameters
		if params == nil {
			params = map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			}
		}
		out = append(out, toolWrapper{
			Type: "function",
			Function: functionDef{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  params,
			},
		})
	}
	return out
}

/* ==================
   Responses API path
   ================== */

type responsesAPIResponse struct {
	OutputText string            `json:"output_text"`
	Output     []json.RawMessage `json:"output"`
}

type responsesItemType struct {
	Type string `json:"type"`
}

type responsesFunctionCall struct {
	ID        string      `json:"id"`
	CallID    string      `json:"call_id"`
	Name      string      `json:"name"`
	Arguments interface{} `json:"arguments"`
	Type      string      `json:"type"`
}

type responsesMessage struct {
	Type    string                    `json:"type"`
	Role    string                    `json:"role"`
	Content []responsesMessageContent `json:"content"`
}

type responsesMessageContent struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

func (p *OpenAIProvider) chatResponses(ctx context.Context, messages []Message, tools []ToolDefinition, model string) (LLMResponse, error) {
	reqBody := map[string]interface{}{
		"model": model,
		"input": buildResponsesInput(messages),
	}

	if len(tools) > 0 {
		reqBody["tools"] = buildResponsesTools(tools)
	}
	if p.MaxTokens > 0 {
		reqBody["max_output_tokens"] = p.MaxTokens
	}
	if p.ReasoningEffort != "" {
		reqBody["reasoning"] = map[string]interface{}{
			"effort": p.ReasoningEffort,
		}
	}

	b, err := json.Marshal(reqBody)
	if err != nil {
		return LLMResponse{}, err
	}

	url := fmt.Sprintf("%s/responses", p.APIBase)
	body, err := p.doJSON(ctx, url, b)
	if err != nil {
		return LLMResponse{}, err
	}

	var out responsesAPIResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return LLMResponse{}, err
	}

	var finalText strings.Builder
	var tcs []ToolCall

	for _, raw := range out.Output {
		var kind responsesItemType
		if err := json.Unmarshal(raw, &kind); err != nil {
			continue
		}

		switch kind.Type {
		case "function_call":
			var fc responsesFunctionCall
			if err := json.Unmarshal(raw, &fc); err != nil {
				continue
			}
			args := map[string]interface{}{}
			switch v := fc.Arguments.(type) {
			case string:
				_ = json.Unmarshal([]byte(v), &args)
			case map[string]interface{}:
				args = v
			}
			id := fc.CallID
			if id == "" {
				id = fc.ID
			}
			tcs = append(tcs, ToolCall{
				ID:        id,
				Name:      fc.Name,
				Arguments: args,
			})

		case "message":
			var msg responsesMessage
			if err := json.Unmarshal(raw, &msg); err != nil {
				continue
			}
			for _, c := range msg.Content {
				if strings.TrimSpace(c.Text) == "" {
					continue
				}
				if finalText.Len() > 0 {
					finalText.WriteString("\n")
				}
				finalText.WriteString(c.Text)
			}
		}
	}

	if len(tcs) > 0 {
		content := strings.TrimSpace(out.OutputText)
		if content == "" {
			content = strings.TrimSpace(finalText.String())
		}
		return LLMResponse{
			Content:      content,
			HasToolCalls: true,
			ToolCalls:    tcs,
		}, nil
	}

	content := strings.TrimSpace(out.OutputText)
	if content == "" {
		content = strings.TrimSpace(finalText.String())
	}
	return LLMResponse{
		Content:      content,
		HasToolCalls: false,
	}, nil
}

func buildResponsesTools(tools []ToolDefinition) []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(tools))
	for _, t := range tools {
		params := t.Parameters
		if params == nil {
			params = map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			}
		}
		out = append(out, map[string]interface{}{
			"type":        "function",
			"name":        t.Name,
			"description": t.Description,
			"parameters":  params,
		})
	}
	return out
}

func buildResponsesInput(messages []Message) []map[string]interface{} {
	input := make([]map[string]interface{}, 0, len(messages)*2)

	for _, m := range messages {
		if m.Role == "tool" {
			if m.ToolCallID == "" {
				continue
			}
			input = append(input, map[string]interface{}{
				"type":    "function_call_output",
				"call_id": m.ToolCallID,
				"output":  m.Content,
			})
			continue
		}

		if len(m.ToolCalls) > 0 {
			if strings.TrimSpace(m.Content) != "" {
				input = append(input, map[string]interface{}{
					"type":    "message",
					"role":    m.Role,
					"content": m.Content,
				})
			}
			for _, tc := range m.ToolCalls {
				argsBytes, _ := json.Marshal(tc.Arguments)
				input = append(input, map[string]interface{}{
					"type":      "function_call",
					"call_id":   tc.ID,
					"name":      tc.Name,
					"arguments": string(argsBytes),
				})
			}
			continue
		}

		input = append(input, map[string]interface{}{
			"type":    "message",
			"role":    m.Role,
			"content": m.Content,
		})
	}

	return input
}

/* ===========
   HTTP helper
   =========== */

func (p *OpenAIProvider) doJSON(ctx context.Context, url string, body []byte) ([]byte, error) {
	if err := p.activeRateLimitError(); err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if p.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.APIKey)
	}

	resp, err := p.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		requestID := strings.TrimSpace(resp.Header.Get("X-Request-Id"))
		if requestID == "" {
			requestID = strings.TrimSpace(resp.Header.Get("Request-Id"))
		}
		if resp.StatusCode == http.StatusTooManyRequests {
			retryAfter := parseRetryAfter(resp.Header.Get("Retry-After"), time.Now())
			until := p.setRateLimitCooldown(resp.Status, requestID, retryAfter)
			if requestID == "" {
				log.Printf("OpenAI API non-2xx: status=%s body_bytes=%d retry_after=%s", resp.Status, len(respBody), retryAfter.Round(time.Second))
			} else {
				log.Printf("OpenAI API non-2xx: status=%s request_id=%s body_bytes=%d retry_after=%s", resp.Status, requestID, len(respBody), retryAfter.Round(time.Second))
			}
			return nil, &RateLimitError{
				Status:     resp.Status,
				RequestID:  requestID,
				RetryAfter: time.Until(until),
				Until:      until,
			}
		}
		if requestID == "" {
			log.Printf("OpenAI API non-2xx: status=%s body_bytes=%d", resp.Status, len(respBody))
		} else {
			log.Printf("OpenAI API non-2xx: status=%s request_id=%s body_bytes=%d", resp.Status, requestID, len(respBody))
		}
		if requestID == "" {
			return nil, fmt.Errorf("OpenAI API error: %s", resp.Status)
		}
		return nil, fmt.Errorf("OpenAI API error: %s (request id: %s)", resp.Status, requestID)
	}
	return respBody, nil
}

func (p *OpenAIProvider) activeRateLimitError() error {
	if p == nil {
		return nil
	}

	p.cooldownMu.Lock()
	defer p.cooldownMu.Unlock()

	now := time.Now()
	if p.rateLimitUntil.IsZero() || !now.Before(p.rateLimitUntil) {
		return nil
	}

	status := p.rateLimitStatus
	if status == "" {
		status = "429 Too Many Requests"
	}
	return &RateLimitError{
		Status:     status,
		RequestID:  p.rateLimitRequestID,
		RetryAfter: time.Until(p.rateLimitUntil),
		Until:      p.rateLimitUntil,
	}
}

func (p *OpenAIProvider) setRateLimitCooldown(status, requestID string, retryAfter time.Duration) time.Time {
	if retryAfter <= 0 {
		retryAfter = time.Minute
	}
	until := time.Now().Add(retryAfter)

	p.cooldownMu.Lock()
	defer p.cooldownMu.Unlock()

	if until.After(p.rateLimitUntil) {
		p.rateLimitUntil = until
		p.rateLimitStatus = status
		p.rateLimitRequestID = requestID
	}
	return p.rateLimitUntil
}

func parseRetryAfter(value string, now time.Time) time.Duration {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Minute
	}

	if seconds, err := strconv.Atoi(value); err == nil {
		if seconds <= 0 {
			return time.Minute
		}
		return time.Duration(seconds) * time.Second
	}

	if at, err := http.ParseTime(value); err == nil {
		if now.Before(at) {
			return at.Sub(now)
		}
	}

	return time.Minute
}
