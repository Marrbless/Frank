package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sort"
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
	mu    sync.RWMutex
	tools map[string]Tool
	guard missioncontrol.ToolGuard
}

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
	r.mu.RUnlock()
	if !ok {
		return "", errors.New("tool not found")
	}

	if guard != nil {
		if ec, ok := missioncontrol.ExecutionContextFromContext(ctx); ok {
			decision := guard.EvaluateTool(ctx, ec, name, args)
			log.Printf("[tool] audit job=%s step=%s tool=%s allowed=%t code=%s reason=%s timestamp=%s", decision.Event.JobID, decision.Event.StepID, decision.Event.ToolName, decision.Event.Allowed, decision.Event.Code, decision.Event.Reason, decision.Event.Timestamp.Format(time.RFC3339Nano))
			if !decision.Allowed {
				log.Printf("[tool] ! %s denied: code=%s reason=%s event=%+v", name, decision.Code, decision.Reason, decision.Event)
				return "", fmt.Errorf("tool rejected: %s: %s", decision.Code, decision.Reason)
			}
		}
	}

	// Log tool execution start
	argsJSON, _ := json.Marshal(args)
	log.Printf("[tool] → %s %s", name, argsJSON)
	start := time.Now()

	result, err := t.Execute(ctx, args)
	elapsed := time.Since(start).Round(time.Millisecond)

	if err != nil {
		log.Printf("[tool] ✗ %s failed after %s: %v", name, elapsed, err)
		return "", err
	}

	log.Printf("[tool] ✓ %s completed in %s (%d bytes)", name, elapsed, len(result))
	return result, nil
}
