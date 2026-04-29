package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSkillsImportCommandImportsSourceRoot(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	source := t.TempDir()
	writeCommandTestSkill(t, source, "grill-me", `---
name: grill-me
description: Stress-test a plan
---

# Grill Me
`)

	cmd := newSkillsCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"import", "--workspace", workspace, source})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	var report struct {
		Imported []struct {
			ID string `json:"id"`
		} `json:"imported"`
	}
	if err := json.Unmarshal(out.Bytes(), &report); err != nil {
		t.Fatalf("json.Unmarshal(report) error = %v\n%s", err, out.String())
	}
	if len(report.Imported) != 1 || report.Imported[0].ID != "grill-me" {
		t.Fatalf("Imported = %#v, want grill-me", report.Imported)
	}
	if _, err := os.Stat(filepath.Join(workspace, "skills", "grill-me", "SKILL.md")); err != nil {
		t.Fatalf("imported skill stat: %v", err)
	}
}

func TestSkillsImportCommandAutoDetectsAgentSkillRoot(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	source := filepath.Join(workspace, ".agents", "skills")
	writeCommandTestSkill(t, source, "interview", `---
name: Interview
description: Ask structured questions
---

# Interview
`)

	cmd := newSkillsCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"import", "--workspace", workspace})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	var report struct {
		Sources  []string `json:"sources"`
		Imported []struct {
			ID string `json:"id"`
		} `json:"imported"`
	}
	if err := json.Unmarshal(out.Bytes(), &report); err != nil {
		t.Fatalf("json.Unmarshal(report) error = %v\n%s", err, out.String())
	}
	if !containsSource(report.Sources, source) {
		t.Fatalf("Sources = %#v, want auto-detected .agents/skills", report.Sources)
	}
	if !containsImportedSkill(report.Imported, "interview") {
		t.Fatalf("Imported = %#v, want interview", report.Imported)
	}
}

func writeCommandTestSkill(t *testing.T, root string, id string, content string) {
	t.Helper()
	dir := filepath.Join(root, id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func containsSource(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func containsImportedSkill(values []struct {
	ID string `json:"id"`
}, want string) bool {
	for _, value := range values {
		if value.ID == want {
			return true
		}
	}
	return false
}
