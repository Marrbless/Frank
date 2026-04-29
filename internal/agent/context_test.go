package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/local/picobot/internal/agent/memory"
	"github.com/local/picobot/internal/missioncontrol"
	"github.com/local/picobot/internal/providers"
)

func TestBuildMessagesIncludesMemories(t *testing.T) {
	cb := NewContextBuilder(".", memory.NewSimpleRanker(), 5)
	history := []string{"user: hi"}
	mems := []memory.MemoryItem{{Kind: "short", Text: "remember this"}, {Kind: "long", Text: "big fact"}}
	memCtx := "Long-term memory: important fact"
	msgs := cb.BuildMessages(history, "hello", "telegram", "123", memCtx, mems, nil, nil)

	// Expect at least 1 system message + 1 user history + 1 current user message
	if len(msgs) < 3 {
		t.Fatalf("expected at least 3 messages, got %d", len(msgs))
	}
	if msgs[0].Role != "system" {
		t.Fatalf("expected first message to be system prompt, got %s", msgs[0].Role)
	}
	// find a system message containing the memory context
	foundMemCtx := false
	foundSummary := false
	for _, m := range msgs {
		if m.Role == "system" && strings.Contains(m.Content, "Long-term memory: important fact") {
			foundMemCtx = true
		}
		if m.Role == "system" && strings.Contains(m.Content, "remember this") && strings.Contains(m.Content, "big fact") {
			foundSummary = true
		}
	}
	if !foundMemCtx {
		t.Fatalf("expected memory context system message to be present in messages: %v", msgs)
	}
	if !foundSummary {
		t.Fatalf("expected memory summary to be present in messages: %v", msgs)
	}
}

func TestBuildMessagesWithoutActiveMissionStepPreservesCurrentBehavior(t *testing.T) {
	cb := NewContextBuilder(".", memory.NewSimpleRanker(), 5)

	msgs := cb.BuildMessages(nil, "hello", "telegram", "123", "", nil, nil, []providers.ToolDefinition{{Name: "write_memory"}})

	if len(msgs) == 0 || msgs[0].Role != "system" {
		t.Fatalf("expected first message to be system prompt, got %#v", msgs)
	}

	sys := msgs[0].Content
	if !strings.Contains(sys, `You are operating on channel="telegram" chatID="123". Tool use is constrained by the current job, step, and policy.`) {
		t.Fatalf("expected base tool policy text in system prompt, got %q", sys)
	}
	if strings.Contains(sys, "An active mission step is in effect") {
		t.Fatalf("did not expect mission-step restriction text without active step, got %q", sys)
	}
}

func TestBuildMessagesWithActiveMissionStepAndNoToolsIncludesNoToolsRestriction(t *testing.T) {
	cb := NewContextBuilder(".", memory.NewSimpleRanker(), 5)
	ec := missioncontrol.ExecutionContext{
		Job:  &missioncontrol.Job{ID: "job-1"},
		Step: &missioncontrol.Step{ID: "build"},
	}

	msgs := cb.BuildMessages(nil, "hello", "telegram", "123", "", nil, &ec, nil)
	sys := msgs[0].Content

	for _, want := range []string{
		`An active mission step is in effect: step="build".`,
		"Tool use is limited to the currently exposed tool set for this mission step.",
		"No tools are available in the current mission step.",
		"Do not claim that you can create files, run commands, browse, write memory, or initiate projects.",
		"Do not simulate tool invocations or print raw tool or command names as if they executed.",
		"say plainly that you cannot perform that action in the current mission",
	} {
		if !strings.Contains(sys, want) {
			t.Fatalf("expected system prompt to contain %q, got %q", want, sys)
		}
	}
}

func TestBuildMessagesWithActiveMissionStepRestrictsToExposedTools(t *testing.T) {
	cb := NewContextBuilder(".", memory.NewSimpleRanker(), 5)
	ec := missioncontrol.ExecutionContext{
		Job:  &missioncontrol.Job{ID: "job-1"},
		Step: &missioncontrol.Step{ID: "build"},
	}

	msgs := cb.BuildMessages(nil, "hello", "telegram", "123", "", nil, &ec, []providers.ToolDefinition{
		{Name: "read_memory"},
		{Name: "write_memory"},
	})
	sys := msgs[0].Content

	if !strings.Contains(sys, "Tool use is limited to the currently exposed tool set for this mission step.") {
		t.Fatalf("expected restricted-tool text in system prompt, got %q", sys)
	}
	if !strings.Contains(sys, "Only the following tools are available in the current mission step: read_memory, write_memory.") {
		t.Fatalf("expected exposed tool list in system prompt, got %q", sys)
	}
	if !strings.Contains(sys, "Do not claim access to any other tools.") {
		t.Fatalf("expected no-other-tools instruction in system prompt, got %q", sys)
	}
	if strings.Contains(sys, "No tools are available in the current mission step.") {
		t.Fatalf("did not expect no-tools text in system prompt, got %q", sys)
	}
}

func TestBuildMessagesIncludesOnlySelectedGovernedSkills(t *testing.T) {
	workspace := t.TempDir()
	writeContextTestSkill(t, workspace, "selected", "Selected instructions")
	writeContextTestSkill(t, workspace, "unselected", "Unselected instructions")

	cb := NewContextBuilder(workspace, memory.NewSimpleRanker(), 5)
	ec := missioncontrol.ExecutionContext{
		Job:  &missioncontrol.Job{ID: "job-1"},
		Step: &missioncontrol.Step{ID: "build", SelectedSkills: []string{"selected"}},
	}

	msgs := cb.BuildMessages(nil, "hello", "telegram", "123", "", nil, &ec, nil)
	sys := msgs[0].Content

	if !strings.Contains(sys, "Active Governed Skills:") || !strings.Contains(sys, "Selected instructions") {
		t.Fatalf("expected selected skill in system prompt, got %q", sys)
	}
	if strings.Contains(sys, "Unselected instructions") {
		t.Fatalf("did not expect unselected skill in system prompt, got %q", sys)
	}
}

func TestBuildMessagesSkipsSelectedInvalidSkill(t *testing.T) {
	workspace := t.TempDir()
	dir := filepath.Join(workspace, "skills", "unsafe")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(`---
id: unsafe
description: Unsafe instructions
allowed_activation_scopes: mission_step_prompt
prompt_only: false
can_affect_tools_or_actions: true
---

Unsafe instructions
`), 0o644); err != nil {
		t.Fatal(err)
	}

	cb := NewContextBuilder(workspace, memory.NewSimpleRanker(), 5)
	ec := missioncontrol.ExecutionContext{
		Job:  &missioncontrol.Job{ID: "job-1"},
		Step: &missioncontrol.Step{ID: "build", SelectedSkills: []string{"unsafe"}},
	}

	msgs := cb.BuildMessages(nil, "hello", "telegram", "123", "", nil, &ec, nil)
	if strings.Contains(msgs[0].Content, "Unsafe instructions") {
		t.Fatalf("invalid skill was included in system prompt: %q", msgs[0].Content)
	}
}

func writeContextTestSkill(t *testing.T, workspace string, id string, body string) {
	t.Helper()
	dir := filepath.Join(workspace, "skills", id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	content := `---
id: ` + id + `
version: v1
description: Test skill
allowed_activation_scopes: mission_step_prompt
prompt_only: true
can_affect_tools_or_actions: false
---

` + body + `
`
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
