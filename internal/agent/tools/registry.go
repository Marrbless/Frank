package tools

import (
	"context"
	"errors"
	"fmt"
	"log"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/local/picobot/internal/missioncontrol"
	"github.com/local/picobot/internal/providers"
)

// Tool is the interface for tools callable by the agent.
type Tool interface {
	Name() string
	Description() string
	// Parameters returns the JSON Schema for tool arguments (nil if no params).
	Parameters() map[string]interface{}
	// Execute performs the tool action and returns a string result.
	Execute(ctx context.Context, args map[string]interface{}) (string, error)
}

// Registry holds registered tools.
type Registry struct {
	mu              sync.RWMutex
	tools           map[string]Tool
	guard           missioncontrol.ToolGuard
	auditEmitter    missioncontrol.AuditEmitter
	missionRequired bool
}

var (
	httpStatusInErrorRE = regexp.MustCompile(`\bHTTP\s+(\d{3})\b`)
	jsonRPCErrorCodeRE  = regexp.MustCompile(`jsonrpc error\s+(-?\d+)`)
)

// NewRegistry constructs a new tool registry.
func NewRegistry() *Registry {
	return &Registry{tools: make(map[string]Tool)}
}

// Register adds a tool to the registry.
func (r *Registry) Register(t Tool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tools[t.Name()] = t
}

// SetGuard attaches an optional tool guard.
func (r *Registry) SetGuard(g missioncontrol.ToolGuard) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.guard = g
}

func (r *Registry) SetAuditEmitter(emitter missioncontrol.AuditEmitter) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.auditEmitter = emitter
}

func (r *Registry) SetMissionRequired(required bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.missionRequired = required
}

func (r *Registry) MissionRequired() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.missionRequired
}

// Get returns a tool by name (or nil if not found).
func (r *Registry) Get(name string) Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.tools[name]
}

// Definitions returns the list of tool definitions to expose to the model.
func (r *Registry) Definitions() []providers.ToolDefinition {
	return r.DefinitionsForExecutionContext(nil)
}

// DefinitionsForExecutionContext returns tool definitions filtered by an optional execution context.
func (r *Registry) DefinitionsForExecutionContext(ec *missioncontrol.ExecutionContext) []providers.ToolDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if ec == nil && r.missionRequired {
		return []providers.ToolDefinition{}
	}

	allowed := allowedToolSetForExecutionContext(ec)
	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		if allowed != nil {
			if _, ok := allowed[name]; !ok {
				continue
			}
		}
		names = append(names, name)
	}
	sort.Strings(names)

	defs := make([]providers.ToolDefinition, 0, len(names))
	for _, name := range names {
		t := r.tools[name]
		defs = append(defs, providers.ToolDefinition{
			Name:        t.Name(),
			Description: t.Description(),
			Parameters:  t.Parameters(),
		})
	}
	return defs
}

func allowedToolSetForExecutionContext(ec *missioncontrol.ExecutionContext) map[string]struct{} {
	if ec == nil {
		return nil
	}

	allowed := make(map[string]struct{})
	if ec.Job != nil {
		for _, toolName := range ec.Job.AllowedTools {
			allowed[toolName] = struct{}{}
		}
	}

	if ec.Step == nil || len(ec.Step.AllowedTools) == 0 {
		return allowed
	}

	filtered := make(map[string]struct{})
	for _, toolName := range ec.Step.AllowedTools {
		if _, ok := allowed[toolName]; ok {
			filtered[toolName] = struct{}{}
		}
	}
	return filtered
}

// Execute executes a registered tool by name with args and returns result or error.
func (r *Registry) Execute(ctx context.Context, name string, args map[string]interface{}) (string, error) {
	if name == "" {
		return "", errors.New("tool name is required")
	}
	r.mu.RLock()
	t, ok := r.tools[name]
	guard := r.guard
	auditEmitter := r.auditEmitter
	missionRequired := r.missionRequired
	r.mu.RUnlock()
	if !ok {
		return "", errors.New("tool not found")
	}

	if missionRequired {
		if _, ok := missioncontrol.ExecutionContextFromContext(ctx); !ok {
			event := missioncontrol.AuditEvent{
				ToolName:  name,
				Allowed:   false,
				Code:      missioncontrol.RejectionCodeMissionContextRequired,
				Reason:    "active mission step is required",
				Timestamp: time.Now(),
			}
			emitAuditEvent(auditEmitter, event)
			return "", fmt.Errorf("tool rejected: %s: %s", surfacedToolRejectionCode(missioncontrol.RejectionCodeMissionContextRequired, "active mission step is required"), "active mission step is required")
		}
	}

	if guard != nil {
		if ec, ok := missioncontrol.ExecutionContextFromContext(ctx); ok {
			decision := guard.EvaluateTool(ctx, ec, name, args)
			emitAuditEvent(auditEmitter, decision.Event)
			if !decision.Allowed {
				log.Printf("[tool] ! %s denied: code=%s reason=%s", name, decision.Code, decision.Reason)
				return "", fmt.Errorf("tool rejected: %s: %s", surfacedToolRejectionCode(decision.Code, decision.Reason), decision.Reason)
			}
		}
	}

	log.Printf("[tool] → %s %s", name, SummarizeToolArguments(args))
	start := time.Now()

	result, err := t.Execute(ctx, args)
	elapsed := time.Since(start).Round(time.Millisecond)

	if err != nil {
		log.Printf("[tool] ✗ %s failed after %s: %s", name, elapsed, SurfaceToolExecutionError(name, err))
		return "", err
	}

	log.Printf("[tool] ✓ %s completed in %s (%d bytes)", name, elapsed, len(result))
	return result, nil
}

func logAuditEvent(event missioncontrol.AuditEvent) {
	log.Printf("[tool] audit job=%s step=%s tool=%s allowed=%t code=%s reason=%s timestamp=%s", event.JobID, event.StepID, event.ToolName, event.Allowed, event.Code, event.Reason, event.Timestamp.Format(time.RFC3339Nano))
}

func emitAuditEvent(emitter missioncontrol.AuditEmitter, event missioncontrol.AuditEvent) {
	if emitter != nil {
		emitter.EmitAuditEvent(event)
	}
	logAuditEvent(event)
}

func surfacedToolRejectionCode(code missioncontrol.RejectionCode, reason string) missioncontrol.RejectionCode {
	if code == "" {
		return ""
	}

	raw := string(code)
	if strings.HasPrefix(raw, "E_") {
		return code
	}

	normalizedReason := strings.ToLower(strings.TrimSpace(reason))

	switch code {
	case missioncontrol.RejectionCodeApprovalRequired:
		return missioncontrol.RejectionCode("E_APPROVAL_REQUIRED")
	case missioncontrol.RejectionCodeAuthorityExceeded:
		return missioncontrol.RejectionCode("E_AUTHORITY_EXCEEDED")
	case missioncontrol.RejectionCodeToolNotAllowed, missioncontrol.RejectionCodeUnknownStep:
		return missioncontrol.RejectionCode("E_INVALID_ACTION_FOR_STEP")
	case missioncontrol.RejectionCodeMissionContextRequired:
		return missioncontrol.RejectionCode("E_NO_ACTIVE_STEP")
	case missioncontrol.RejectionCodeWaitingUser:
		return missioncontrol.RejectionCode("E_WAITING_FOR_USER")
	case missioncontrol.RejectionCodeLongRunningStartForbidden:
		return missioncontrol.RejectionCode("E_LONGRUN_START_FORBIDDEN")
	case missioncontrol.RejectionCodeResumeApprovalRequired:
		return missioncontrol.RejectionCode("E_RESUME_REQUIRES_APPROVAL")
	case missioncontrol.RejectionCodeStepValidationFailed,
		missioncontrol.RejectionCodeFalseCompletionClaim,
		missioncontrol.RejectionCodeValidationRequired:
		return missioncontrol.RejectionCode("E_VALIDATION_FAILED")
	case missioncontrol.RejectionCodeInvalidJobTransition:
		return missioncontrol.RejectionCode("E_STEP_OUT_OF_ORDER")
	case missioncontrol.RejectionCodeInvalidRuntimeState:
		switch {
		case strings.Contains(normalizedReason, "requires an active step"), strings.Contains(normalizedReason, "active mission step is required"), strings.Contains(normalizedReason, "no active step"):
			return missioncontrol.RejectionCode("E_NO_ACTIVE_STEP")
		case strings.Contains(normalizedReason, "aborted"):
			return missioncontrol.RejectionCode("E_ABORTED")
		default:
			return missioncontrol.RejectionCode("E_STEP_OUT_OF_ORDER")
		}
	default:
		return code
	}
}

func SummarizeToolArguments(args map[string]interface{}) string {
	if len(args) == 0 {
		return "arg_count=0"
	}

	typeCounts := make(map[string]int)
	for _, value := range args {
		typeCounts[summarizeToolArgumentType(value)]++
	}

	typeNames := make([]string, 0, len(typeCounts))
	for typeName := range typeCounts {
		typeNames = append(typeNames, typeName)
	}
	sort.Strings(typeNames)

	parts := make([]string, 0, len(typeNames))
	for _, typeName := range typeNames {
		parts = append(parts, fmt.Sprintf("%s:%d", typeName, typeCounts[typeName]))
	}

	return fmt.Sprintf("arg_count=%d arg_types=%s", len(args), strings.Join(parts, ","))
}

func summarizeToolArgumentType(value interface{}) string {
	if value == nil {
		return "null"
	}

	switch value.(type) {
	case bool:
		return "bool"
	case string:
		return "string"
	}

	kind := reflect.TypeOf(value).Kind()
	switch kind {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return "number"
	case reflect.Array, reflect.Slice:
		return "array"
	case reflect.Map, reflect.Struct:
		return "object"
	default:
		return "other"
	}
}

func SurfaceToolExecutionError(name string, err error) string {
	if err == nil {
		return ""
	}

	msg := strings.TrimSpace(err.Error())
	if msg == "" {
		return "tool execution failed"
	}
	if strings.HasPrefix(msg, "tool rejected:") {
		return msg
	}
	if strings.HasPrefix(msg, "OpenAI API error:") {
		return summarizeOpenAIProviderError(msg)
	}
	if strings.HasPrefix(name, "mcp_") {
		if status := extractHTTPStatus(msg); status != "" {
			return fmt.Sprintf("MCP tool failed (%s)", status)
		}
		if code := extractJSONRPCErrorCode(msg); code != "" {
			return fmt.Sprintf("MCP tool failed (%s)", code)
		}
		return "MCP tool failed"
	}
	if isRemoteSensitiveTool(name, msg) {
		if status := extractHTTPStatus(msg); status != "" {
			return fmt.Sprintf("remote tool error (%s)", status)
		}
		if code := extractJSONRPCErrorCode(msg); code != "" {
			return fmt.Sprintf("remote tool error (%s)", code)
		}
		return "remote tool error"
	}
	return msg
}

func isRemoteSensitiveTool(name, msg string) bool {
	switch name {
	case "web", "web_search", FrankZohoSendEmailToolName, FrankZohoManageReplyWorkItemToolName:
		return true
	}

	lower := strings.ToLower(msg)
	return strings.Contains(lower, "http ") ||
		strings.Contains(lower, "jsonrpc error") ||
		strings.HasPrefix(lower, "tool error:")
}

func summarizeOpenAIProviderError(msg string) string {
	trimmed := strings.TrimSpace(strings.TrimPrefix(msg, "OpenAI API error:"))
	if idx := strings.Index(trimmed, " - "); idx >= 0 {
		trimmed = strings.TrimSpace(trimmed[:idx])
	}
	if trimmed == "" {
		return "OpenAI API error"
	}
	return "OpenAI API error: " + trimmed
}

func extractHTTPStatus(msg string) string {
	matches := httpStatusInErrorRE.FindStringSubmatch(msg)
	if len(matches) != 2 {
		return ""
	}
	return "HTTP " + matches[1]
}

func extractJSONRPCErrorCode(msg string) string {
	matches := jsonRPCErrorCodeRE.FindStringSubmatch(msg)
	if len(matches) != 2 {
		return ""
	}
	return "jsonrpc error " + matches[1]
}
