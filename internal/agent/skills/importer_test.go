package skills

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestImportGovernedSkillsImportsCodexStyleSkill(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	sourceRoot := t.TempDir()
	writeSourceSkill(t, sourceRoot, "Grill Me", `---
name: Grill Me
description: Stress-test a plan
---

# Grill Me

Ask hard questions.
`)

	report, err := ImportGovernedSkills(ImportOptions{
		WorkspacePath: workspace,
		Sources:       []string{sourceRoot},
	})
	if err != nil {
		t.Fatalf("ImportGovernedSkills() error = %v", err)
	}
	if len(report.Imported) != 1 || report.Imported[0].ID != "grill-me" {
		t.Fatalf("Imported = %#v, want one normalized grill-me import", report.Imported)
	}

	content, err := os.ReadFile(filepath.Join(workspace, "skills", "grill-me", "SKILL.md"))
	if err != nil {
		t.Fatalf("read imported skill: %v", err)
	}
	assertContains(t, string(content), "id: grill-me")
	assertContains(t, string(content), "allowed_activation_scopes: mission_step_prompt")
	assertContains(t, string(content), "prompt_only: true")
	assertContains(t, string(content), "can_affect_tools_or_actions: false")
	assertContains(t, string(content), "Ask hard questions.")
}

func TestImportGovernedSkillsSkipsUnsafeToolAffectingSkill(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	sourceRoot := t.TempDir()
	writeSourceSkill(t, sourceRoot, "unsafe", `---
id: unsafe
description: Unsafe import
prompt_only: false
can_affect_tools_or_actions: true
---

# Unsafe
`)

	report, err := ImportGovernedSkills(ImportOptions{
		WorkspacePath: workspace,
		Sources:       []string{sourceRoot},
	})
	if err != nil {
		t.Fatalf("ImportGovernedSkills() error = %v", err)
	}
	if len(report.Imported) != 0 {
		t.Fatalf("Imported = %#v, want none", report.Imported)
	}
	if len(report.Skipped) != 1 || report.Skipped[0].Reason != ImportSkipReasonToolEffectsBlocked {
		t.Fatalf("Skipped = %#v, want tool/action blocked reason", report.Skipped)
	}
}

func TestImportGovernedSkillsDuplicateIDsResolveDeterministically(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	sourceRoot := t.TempDir()
	writeSourceSkill(t, sourceRoot, "a", `---
id: same
description: First same
---

# A
`)
	writeSourceSkill(t, sourceRoot, "b", `---
id: same
description: Second same
---

# B
`)

	report, err := ImportGovernedSkills(ImportOptions{
		WorkspacePath: workspace,
		Sources:       []string{sourceRoot},
	})
	if err != nil {
		t.Fatalf("ImportGovernedSkills() error = %v", err)
	}
	if len(report.Imported) != 1 || report.Imported[0].ID != "same" {
		t.Fatalf("Imported = %#v, want one same skill", report.Imported)
	}
	if len(report.Skipped) != 1 || report.Skipped[0].Reason != ImportSkipReasonDuplicateCandidate {
		t.Fatalf("Skipped = %#v, want duplicate candidate", report.Skipped)
	}
}

func TestAutoDetectImportSourcesFindsKnownInstallerRoots(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	cwd := t.TempDir()
	home := t.TempDir()
	mustMkdir(t, filepath.Join(workspace, ".agents", "skills"))
	mustMkdir(t, filepath.Join(cwd, ".codex", "skills"))
	mustMkdir(t, filepath.Join(home, ".codex", "skills"))

	got := AutoDetectImportSources(workspace, cwd, home)
	want := []string{
		filepath.Join(workspace, ".agents", "skills"),
		filepath.Join(cwd, ".codex", "skills"),
		filepath.Join(home, ".codex", "skills"),
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("AutoDetectImportSources() = %#v, want %#v", got, want)
	}
}

func writeSourceSkill(t *testing.T, root string, id string, content string) {
	t.Helper()
	dir := filepath.Join(root, id)
	mustMkdir(t, dir)
	if err := os.WriteFile(filepath.Join(dir, SkillFileName), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func mustMkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
}

func assertContains(t *testing.T, got string, want string) {
	t.Helper()
	if !strings.Contains(got, want) {
		t.Fatalf("content missing %q:\n%s", want, got)
	}
}
