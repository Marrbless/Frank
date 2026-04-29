package skills

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/local/picobot/internal/missioncontrol"
)

func TestLoaderNoSkillsConfiguredPreservesEmptySelection(t *testing.T) {
	t.Parallel()

	loader := NewLoader(t.TempDir())
	report, err := loader.LoadSelected(nil, missioncontrol.SkillActivationScopeMissionStepPrompt)
	if err != nil {
		t.Fatalf("LoadSelected() error = %v", err)
	}
	if len(report.Active) != 0 || len(report.Skipped) != 0 || len(report.Selected) != 0 {
		t.Fatalf("report = %#v, want empty active/skipped/selected", report)
	}
}

func TestLoaderValidSkillLoadsDeterministically(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	writeSkill(t, workspace, "weather", `---
id: weather
version: v1
description: Weather workflow
allowed_activation_scopes: mission_step_prompt
prompt_only: true
can_affect_tools_or_actions: false
---

# Weather

Use the weather API only when normal tools allow it.
`)
	writeSkill(t, workspace, "calendar", `---
id: calendar
version: v1
description: Calendar workflow
allowed_activation_scopes: [mission_step_prompt]
prompt_only: true
---

# Calendar
`)

	loader := NewLoader(workspace)
	first, err := loader.LoadSelected([]string{"weather", "calendar"}, missioncontrol.SkillActivationScopeMissionStepPrompt)
	if err != nil {
		t.Fatalf("LoadSelected(first) error = %v", err)
	}
	second, err := loader.LoadSelected([]string{"weather", "calendar"}, missioncontrol.SkillActivationScopeMissionStepPrompt)
	if err != nil {
		t.Fatalf("LoadSelected(second) error = %v", err)
	}

	gotIDs := []string{first.Active[0].ID, first.Active[1].ID}
	if !reflect.DeepEqual(gotIDs, []string{"weather", "calendar"}) {
		t.Fatalf("active IDs = %#v, want selected order", gotIDs)
	}
	if !reflect.DeepEqual(first.Status(), second.Status()) {
		t.Fatalf("LoadSelected status not stable:\nfirst=%#v\nsecond=%#v", first.Status(), second.Status())
	}
	if first.Active[0].ContentHash == "" {
		t.Fatal("ContentHash empty, want sha256 hash")
	}
}

func TestLoaderInvalidSkillSkippedWithReason(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	writeSkill(t, workspace, "bad", `---
id: other
description: Bad workflow
allowed_activation_scopes: mission_step_prompt
prompt_only: true
---

# Bad
`)

	report, err := NewLoader(workspace).LoadSelected([]string{"bad"}, missioncontrol.SkillActivationScopeMissionStepPrompt)
	if err != nil {
		t.Fatalf("LoadSelected() error = %v", err)
	}
	if len(report.Active) != 0 {
		t.Fatalf("Active = %#v, want none", report.Active)
	}
	if len(report.Skipped) != 1 || report.Skipped[0].Reason != SkipReasonIDMismatch {
		t.Fatalf("Skipped = %#v, want id mismatch reason", report.Skipped)
	}
}

func TestLoaderUnselectedSkillIsNotIncluded(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	writeGovernedSkill(t, workspace, "selected")
	writeGovernedSkill(t, workspace, "unselected")

	report, err := NewLoader(workspace).LoadSelected([]string{"selected"}, missioncontrol.SkillActivationScopeMissionStepPrompt)
	if err != nil {
		t.Fatalf("LoadSelected() error = %v", err)
	}
	if len(report.Active) != 1 || report.Active[0].ID != "selected" {
		t.Fatalf("Active = %#v, want only selected", report.Active)
	}
}

func TestLoaderDuplicateSelectedSkillSkippedDeterministically(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	writeGovernedSkill(t, workspace, "weather")

	report, err := NewLoader(workspace).LoadSelected([]string{"weather", "weather"}, missioncontrol.SkillActivationScopeMissionStepPrompt)
	if err != nil {
		t.Fatalf("LoadSelected() error = %v", err)
	}
	if len(report.Active) != 1 || report.Active[0].ID != "weather" {
		t.Fatalf("Active = %#v, want one weather skill", report.Active)
	}
	if len(report.Skipped) != 1 || report.Skipped[0].Reason != SkipReasonDuplicateSelection {
		t.Fatalf("Skipped = %#v, want duplicate selection", report.Skipped)
	}
}

func TestLoaderSkillCannotSilentlyAlterToolsOrActions(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	writeSkill(t, workspace, "unsafe", `---
id: unsafe
description: Unsafe workflow
allowed_activation_scopes: mission_step_prompt
prompt_only: false
can_affect_tools_or_actions: true
---

# Unsafe
`)

	report, err := NewLoader(workspace).LoadSelected([]string{"unsafe"}, missioncontrol.SkillActivationScopeMissionStepPrompt)
	if err != nil {
		t.Fatalf("LoadSelected() error = %v", err)
	}
	if len(report.Active) != 0 {
		t.Fatalf("Active = %#v, want none", report.Active)
	}
	if len(report.Skipped) != 1 || report.Skipped[0].Reason != SkipReasonToolActionEffectsBlocked {
		t.Fatalf("Skipped = %#v, want tool/action blocked reason", report.Skipped)
	}
}

func writeGovernedSkill(t *testing.T, workspace string, id string) {
	t.Helper()
	writeSkill(t, workspace, id, `---
id: `+id+`
version: v1
description: Test skill
allowed_activation_scopes: mission_step_prompt
prompt_only: true
can_affect_tools_or_actions: false
---

# `+id+`
`)
}

func writeSkill(t *testing.T, workspace string, id string, content string) {
	t.Helper()
	dir := filepath.Join(workspace, SkillsDirName, id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, SkillFileName), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
