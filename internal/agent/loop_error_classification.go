package agent

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/local/picobot/internal/providers"
)

func summarizeProviderError(err error) string {
	if err == nil {
		return "provider request failed"
	}
	var rateLimitErr *providers.RateLimitError
	if errors.As(err, &rateLimitErr) {
		if rateLimitErr.RetryAfter > 0 {
			return fmt.Sprintf("OpenAI API rate limited; retry after %s", rateLimitErr.RetryAfter.Round(time.Second))
		}
		return "OpenAI API rate limited"
	}

	msg := strings.TrimSpace(err.Error())
	if msg == "" {
		return "provider request failed"
	}
	if strings.HasPrefix(msg, "OpenAI API error:") {
		trimmed := strings.TrimSpace(strings.TrimPrefix(msg, "OpenAI API error:"))
		if idx := strings.Index(trimmed, " - "); idx >= 0 {
			trimmed = strings.TrimSpace(trimmed[:idx])
		}
		if trimmed == "" {
			return "OpenAI API error"
		}
		return "OpenAI API error: " + trimmed
	}

	lower := strings.ToLower(msg)
	switch {
	case strings.Contains(lower, "context deadline exceeded"):
		return "provider request timed out"
	case strings.Contains(lower, "connection refused"), strings.Contains(lower, "no such host"):
		return "provider connection failed"
	default:
		return "provider request failed"
	}
}

func providerRateLimitResponse(err error) (string, bool) {
	var rateLimitErr *providers.RateLimitError
	if !errors.As(err, &rateLimitErr) {
		return "", false
	}
	if rateLimitErr.RetryAfter > 0 {
		return fmt.Sprintf("OpenAI is rate limited right now. Try again after about %s.", rateLimitErr.RetryAfter.Round(time.Second)), true
	}
	return "OpenAI is rate limited right now. Try again shortly.", true
}

func summarizeMCPConnectError(err error) string {
	if err == nil {
		return "connection failed"
	}

	msg := strings.TrimSpace(err.Error())
	if msg == "" {
		return "connection failed"
	}
	if matches := mcpHTTPStatusRE.FindStringSubmatch(msg); len(matches) == 2 {
		return fmt.Sprintf("connection failed (HTTP %s)", matches[1])
	}
	if matches := mcpJSONRPCErrorRE.FindStringSubmatch(msg); len(matches) == 2 {
		return fmt.Sprintf("connection failed (jsonrpc error %s)", matches[1])
	}

	switch {
	case strings.Contains(msg, "initialize:"):
		return "initialize failed"
	case strings.Contains(msg, "tools/list:"):
		return "tools/list failed"
	case strings.Contains(msg, "unexpected EOF"):
		return "unexpected EOF"
	case strings.Contains(msg, "no response in SSE stream"):
		return "no response in SSE stream"
	default:
		return "connection failed"
	}
}
