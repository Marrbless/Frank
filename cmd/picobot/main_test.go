package main

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"

	"github.com/local/picobot/internal/agent"
	"github.com/local/picobot/internal/agent/memory"
	"github.com/local/picobot/internal/chat"
	"github.com/local/picobot/internal/config"
	"github.com/local/picobot/internal/missioncontrol"
	"github.com/local/picobot/internal/providers"
)

func TestMemoryCLI_ReadAppendWriteRecent(t *testing.T) {
	// set HOME to a temp dir so onboard writes to temp
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	// create default config + workspace
	if _, _, err := config.Onboard(); err != nil {
		t.Fatalf("onboard failed: %v", err)
	}

	// run: picobot memory append today -c "hello"
	cmd := NewRootCmd()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"memory", "append", "today", "-c", "hello"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("append today failed: %v", err)
	}

	// verify today's file exists
	cfg, _ := config.LoadConfig()
	ws := cfg.Agents.Defaults.Workspace
	if strings.HasPrefix(ws, "~") {
		home, _ := os.UserHomeDir()
		ws = filepath.Join(home, ws[2:])
	}
	memFile := filepath.Join(ws, "memory")
	files, _ := os.ReadDir(memFile)
	found := false
	for _, f := range files {
		if strings.HasSuffix(f.Name(), ".md") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected memory files, none found in %s", memFile)
	}

	// write long-term
	cmd = NewRootCmd()
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"memory", "write", "long", "-c", "LONGCONTENT"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("write long failed: %v", err)
	}

	// read long-term
	cmd = NewRootCmd()
	readBuf := &bytes.Buffer{}
	cmd.SetOut(readBuf)
	cmd.SetArgs([]string{"memory", "read", "long"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("read long failed: %v", err)
	}
	out := readBuf.String()
	if !strings.Contains(out, "LONGCONTENT") {
		t.Fatalf("expected LONGCONTENT in output, got %q", out)
	}

	// recent days
	cmd = NewRootCmd()
	recentBuf := &bytes.Buffer{}
	cmd.SetOut(recentBuf)
	cmd.SetArgs([]string{"memory", "recent", "--days", "1"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("recent failed: %v", err)
	}
	if recentBuf.String() == "" {
		t.Fatalf("expected recent output, got empty")
	}
}

func TestMemoryCLI_Rank(t *testing.T) {
	// set HOME to a temp dir so onboard writes to temp
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	// create default config + workspace
	if _, _, err := config.Onboard(); err != nil {
		t.Fatalf("onboard failed: %v", err)
	}

	// append some memories
	cfg, _ := config.LoadConfig()
	ws := cfg.Agents.Defaults.Workspace
	if strings.HasPrefix(ws, "~") {
		home, _ := os.UserHomeDir()
		ws = filepath.Join(home, ws[2:])
	}
	mem := memory.NewMemoryStoreWithWorkspace(ws, 100)
	_ = mem.AppendToday("buy milk and eggs")
	_ = mem.AppendToday("call mom tomorrow")
	_ = mem.AppendToday("milkshake recipe")

	// run rank command
	cmd := NewRootCmd()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"memory", "rank", "-q", "milk", "-k", "2"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("rank failed: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "buy milk") {
		t.Fatalf("expected 'buy milk' in output, got: %q", out)
	}
	if !strings.Contains(out, "milkshake") && !strings.Contains(out, "Important facts") {
		t.Fatalf("expected either 'milkshake' or 'Important facts' in output, got: %q", out)
	}
}

func TestAgentCLI_ModelFlag(t *testing.T) {
	// set HOME to a temp dir so onboard writes to temp
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	if _, _, err := config.Onboard(); err != nil {
		t.Fatalf("onboard failed: %v", err)
	}
	// remove OpenAI from config so stub provider is used
	cfgPath, _, _ := config.ResolveDefaultPaths()
	cfg2, _ := config.LoadConfig()
	cfg2.Providers.OpenAI = nil
	_ = config.SaveConfig(cfg2, cfgPath)

	cmd := NewRootCmd()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"agent", "--model", "stub-model", "-m", "hello"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("agent failed: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "(stub) Echo") {
		t.Fatalf("expected stub echo output, got: %q", out)
	}
}

func TestMissionStatusCommandWithValidFilePrintsExpectedJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "status.json")
	want := []byte("{\n  \"mission_required\": true,\n  \"active\": true,\n  \"mission_file\": \"mission.json\",\n  \"job_id\": \"job-1\",\n  \"step_id\": \"build\",\n  \"step_type\": \"one_shot_code\",\n  \"allowed_tools\": [\n    \"read\"\n  ],\n  \"updated_at\": \"2026-03-20T12:00:00Z\"\n}\n")
	if err := os.WriteFile(path, want, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cmd := NewRootCmd()
	out := &bytes.Buffer{}
	cmd.SetOut(out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mission", "status", "--status-file", path})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if out.String() != string(want) {
		t.Fatalf("stdout = %q, want %q", out.String(), string(want))
	}
}

func TestMissionStatusCommandWithActiveStepFieldsPrintsExpectedJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "status.json")
	wantSnapshot := missionStatusSnapshot{
		MissionRequired:   true,
		Active:            true,
		MissionFile:       "mission.json",
		JobID:             "job-1",
		StepID:            "build",
		StepType:          string(missioncontrol.StepTypeOneShotCode),
		RequiredAuthority: missioncontrol.AuthorityTierMedium,
		RequiresApproval:  true,
		AllowedTools:      []string{"read"},
		UpdatedAt:         "2026-03-20T12:00:00Z",
	}
	want, err := json.MarshalIndent(wantSnapshot, "", "  ")
	if err != nil {
		t.Fatalf("json.MarshalIndent() error = %v", err)
	}
	want = append(want, '\n')
	if err := os.WriteFile(path, want, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cmd := NewRootCmd()
	out := &bytes.Buffer{}
	cmd.SetOut(out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mission", "status", "--status-file", path})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if out.String() != string(want) {
		t.Fatalf("stdout = %q, want %q", out.String(), string(want))
	}
}

func TestMissionInspectCommandWithValidFilePrintsExpectedSummary(t *testing.T) {
	job := missioncontrol.Job{
		ID:           "job-1",
		MaxAuthority: missioncontrol.AuthorityTierHigh,
		AllowedTools: []string{"read", "write", "write"},
		Plan: missioncontrol.Plan{
			ID: "plan-1",
			Steps: []missioncontrol.Step{
				{
					ID:                "draft",
					Type:              missioncontrol.StepTypeDiscussion,
					RequiredAuthority: missioncontrol.AuthorityTierLow,
					AllowedTools:      []string{"read", "read"},
					RequiresApproval:  true,
					SuccessCriteria:   []string{"share a concise plan"},
				},
				{
					ID:                "build",
					Type:              missioncontrol.StepTypeOneShotCode,
					DependsOn:         []string{"draft", "draft"},
					RequiredAuthority: missioncontrol.AuthorityTierMedium,
					AllowedTools:      []string{"write", "write"},
					SuccessCriteria:   []string{"produce code"},
				},
				{
					ID:                "final",
					Type:              missioncontrol.StepTypeFinalResponse,
					DependsOn:         []string{"build"},
					RequiredAuthority: missioncontrol.AuthorityTierLow,
				},
			},
		},
	}
	path := writeMissionBootstrapJobFile(t, job)

	cmd := NewRootCmd()
	out := &bytes.Buffer{}
	cmd.SetOut(out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mission", "inspect", "--mission-file", path})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	var got missionInspectSummary
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if got.JobID != job.ID {
		t.Fatalf("JobID = %q, want %q", got.JobID, job.ID)
	}
	if got.MaxAuthority != job.MaxAuthority {
		t.Fatalf("MaxAuthority = %q, want %q", got.MaxAuthority, job.MaxAuthority)
	}
	if !reflect.DeepEqual(got.AllowedTools, job.AllowedTools) {
		t.Fatalf("AllowedTools = %v, want %v", got.AllowedTools, job.AllowedTools)
	}
	if len(got.Steps) != len(job.Plan.Steps) {
		t.Fatalf("len(Steps) = %d, want %d", len(got.Steps), len(job.Plan.Steps))
	}
	if got.Steps[0].StepID != "draft" || got.Steps[1].StepID != "build" || got.Steps[2].StepID != "final" {
		t.Fatalf("step order = %#v, want draft/build/final", got.Steps)
	}
	if !reflect.DeepEqual(got.Steps[1].DependsOn, []string{"draft", "draft"}) {
		t.Fatalf("build DependsOn = %v, want duplicate-preserving slice", got.Steps[1].DependsOn)
	}
	if !reflect.DeepEqual(got.Steps[1].AllowedTools, []string{"write", "write"}) {
		t.Fatalf("build AllowedTools = %v, want duplicate-preserving slice", got.Steps[1].AllowedTools)
	}
	if !reflect.DeepEqual(got.Steps[0].SuccessCriteria, []string{"share a concise plan"}) {
		t.Fatalf("draft SuccessCriteria = %v, want [share a concise plan]", got.Steps[0].SuccessCriteria)
	}
	if !reflect.DeepEqual(got.Steps[1].SuccessCriteria, []string{"produce code"}) {
		t.Fatalf("build SuccessCriteria = %v, want [produce code]", got.Steps[1].SuccessCriteria)
	}
	if !reflect.DeepEqual(got.Steps[0].EffectiveAllowedTools, []string{"read"}) {
		t.Fatalf("draft EffectiveAllowedTools = %v, want [read]", got.Steps[0].EffectiveAllowedTools)
	}
	if !reflect.DeepEqual(got.Steps[1].EffectiveAllowedTools, []string{"write"}) {
		t.Fatalf("build EffectiveAllowedTools = %v, want [write]", got.Steps[1].EffectiveAllowedTools)
	}
	if !reflect.DeepEqual(got.Steps[2].EffectiveAllowedTools, []string{"read", "write"}) {
		t.Fatalf("final EffectiveAllowedTools = %v, want [read write]", got.Steps[2].EffectiveAllowedTools)
	}
	if !got.Steps[0].RequiresApproval {
		t.Fatal("draft RequiresApproval = false, want true")
	}
}

func TestMissionInspectCommandWithStepIDReturnsExactlyOneResolvedStep(t *testing.T) {
	job := missioncontrol.Job{
		ID:           "job-1",
		MaxAuthority: missioncontrol.AuthorityTierHigh,
		AllowedTools: []string{"read", "write", "search"},
		Plan: missioncontrol.Plan{
			ID: "plan-1",
			Steps: []missioncontrol.Step{
				{
					ID:                "draft",
					Type:              missioncontrol.StepTypeDiscussion,
					RequiredAuthority: missioncontrol.AuthorityTierLow,
					AllowedTools:      []string{"read"},
				},
				{
					ID:                "build",
					Type:              missioncontrol.StepTypeOneShotCode,
					DependsOn:         []string{"draft", "draft"},
					RequiredAuthority: missioncontrol.AuthorityTierMedium,
					AllowedTools:      []string{"write", "write"},
					SuccessCriteria:   []string{"produce code"},
				},
				{
					ID:        "final",
					Type:      missioncontrol.StepTypeFinalResponse,
					DependsOn: []string{"build"},
				},
			},
		},
	}
	path := writeMissionBootstrapJobFile(t, job)

	cmd := NewRootCmd()
	out := &bytes.Buffer{}
	cmd.SetOut(out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mission", "inspect", "--mission-file", path, "--step-id", "build"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	var got missionInspectSummary
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if got.JobID != job.ID {
		t.Fatalf("JobID = %q, want %q", got.JobID, job.ID)
	}
	if got.MaxAuthority != job.MaxAuthority {
		t.Fatalf("MaxAuthority = %q, want %q", got.MaxAuthority, job.MaxAuthority)
	}
	if !reflect.DeepEqual(got.AllowedTools, job.AllowedTools) {
		t.Fatalf("AllowedTools = %v, want %v", got.AllowedTools, job.AllowedTools)
	}
	if len(got.Steps) != 1 {
		t.Fatalf("len(Steps) = %d, want 1", len(got.Steps))
	}
	if got.Steps[0].StepID != "build" {
		t.Fatalf("StepID = %q, want %q", got.Steps[0].StepID, "build")
	}
	if !reflect.DeepEqual(got.Steps[0].DependsOn, []string{"draft", "draft"}) {
		t.Fatalf("DependsOn = %v, want duplicate-preserving slice", got.Steps[0].DependsOn)
	}
	if !reflect.DeepEqual(got.Steps[0].SuccessCriteria, []string{"produce code"}) {
		t.Fatalf("SuccessCriteria = %v, want [produce code]", got.Steps[0].SuccessCriteria)
	}
}

func TestMissionInspectCommandWithStepIDIncludesResolvedEffectiveAllowedTools(t *testing.T) {
	job := missioncontrol.Job{
		ID:           "job-1",
		MaxAuthority: missioncontrol.AuthorityTierHigh,
		AllowedTools: []string{"read", "write", "search"},
		Plan: missioncontrol.Plan{
			ID: "plan-1",
			Steps: []missioncontrol.Step{
				{
					ID:                "build",
					Type:              missioncontrol.StepTypeOneShotCode,
					RequiredAuthority: missioncontrol.AuthorityTierMedium,
					AllowedTools:      []string{"write", "write"},
				},
				{
					ID:        "final",
					Type:      missioncontrol.StepTypeFinalResponse,
					DependsOn: []string{"build"},
				},
			},
		},
	}
	path := writeMissionBootstrapJobFile(t, job)

	cmd := NewRootCmd()
	out := &bytes.Buffer{}
	cmd.SetOut(out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mission", "inspect", "--mission-file", path, "--step-id", "build"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	var got missionInspectSummary
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if !reflect.DeepEqual(got.Steps[0].EffectiveAllowedTools, []string{"write"}) {
		t.Fatalf("EffectiveAllowedTools = %v, want [write]", got.Steps[0].EffectiveAllowedTools)
	}
}

func TestMissionInspectCommandWithUnknownStepReturnsClearError(t *testing.T) {
	path := writeMissionBootstrapJobFile(t, testMissionBootstrapJob())

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mission", "inspect", "--mission-file", path, "--step-id", "missing"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "failed to resolve mission inspection summary") {
		t.Fatalf("Execute() error = %q, want clear inspect summary failure", err)
	}
	if !strings.Contains(err.Error(), string(missioncontrol.RejectionCodeUnknownStep)) {
		t.Fatalf("Execute() error = %q, want unknown_step code", err)
	}
	if !strings.Contains(err.Error(), `step "missing" not found in plan`) {
		t.Fatalf("Execute() error = %q, want missing step message", err)
	}
}

func TestMissionInspectCommandWithoutStepIDPreservesExistingBehavior(t *testing.T) {
	path := writeMissionBootstrapJobFile(t, testMissionBootstrapJob())

	cmd := NewRootCmd()
	out := &bytes.Buffer{}
	cmd.SetOut(out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mission", "inspect", "--mission-file", path})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	var got missionInspectSummary
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if got.JobID != "job-1" {
		t.Fatalf("JobID = %q, want %q", got.JobID, "job-1")
	}
	if len(got.Steps) != 2 {
		t.Fatalf("len(Steps) = %d, want 2", len(got.Steps))
	}
	if got.Steps[0].StepID != "build" || got.Steps[1].StepID != "final" {
		t.Fatalf("step order = %#v, want build/final", got.Steps)
	}
}

func TestMissionInspectCommandSuccessCriteriaZeroValuePreservesExistingBehavior(t *testing.T) {
	job := missioncontrol.Job{
		ID:           "job-1",
		MaxAuthority: missioncontrol.AuthorityTierLow,
		AllowedTools: []string{"read"},
		Plan: missioncontrol.Plan{
			ID: "plan-1",
			Steps: []missioncontrol.Step{
				{
					ID:                "draft",
					Type:              missioncontrol.StepTypeDiscussion,
					RequiredAuthority: missioncontrol.AuthorityTierLow,
				},
				{
					ID:                "build",
					Type:              missioncontrol.StepTypeOneShotCode,
					RequiredAuthority: missioncontrol.AuthorityTierLow,
					SuccessCriteria:   []string{},
				},
				{
					ID:        "final",
					Type:      missioncontrol.StepTypeFinalResponse,
					DependsOn: []string{"build"},
				},
			},
		},
	}
	path := writeMissionBootstrapJobFile(t, job)

	cmd := NewRootCmd()
	out := &bytes.Buffer{}
	cmd.SetOut(out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mission", "inspect", "--mission-file", path})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	var got missionInspectSummary
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if got.Steps[0].SuccessCriteria != nil {
		t.Fatalf("draft SuccessCriteria = %v, want nil", got.Steps[0].SuccessCriteria)
	}
	if got.Steps[1].SuccessCriteria != nil {
		t.Fatalf("build SuccessCriteria = %v, want nil", got.Steps[1].SuccessCriteria)
	}
	if got.Steps[2].SuccessCriteria != nil {
		t.Fatalf("final SuccessCriteria = %v, want nil", got.Steps[2].SuccessCriteria)
	}
}

func TestMissionInspectCommandWithZeroToolStepPrintsEmptyEffectiveAllowedTools(t *testing.T) {
	job := missioncontrol.Job{
		ID:           "job-1",
		MaxAuthority: missioncontrol.AuthorityTierLow,
		AllowedTools: []string{},
		Plan: missioncontrol.Plan{
			ID: "plan-1",
			Steps: []missioncontrol.Step{
				{
					ID:                "discuss",
					Type:              missioncontrol.StepTypeDiscussion,
					RequiredAuthority: missioncontrol.AuthorityTierLow,
				},
				{
					ID:        "final",
					Type:      missioncontrol.StepTypeFinalResponse,
					DependsOn: []string{"discuss"},
				},
			},
		},
	}
	path := writeMissionBootstrapJobFile(t, job)

	cmd := NewRootCmd()
	out := &bytes.Buffer{}
	cmd.SetOut(out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mission", "inspect", "--mission-file", path})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	var got missionInspectSummary
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if !reflect.DeepEqual(got.Steps[0].EffectiveAllowedTools, []string{}) {
		t.Fatalf("discuss EffectiveAllowedTools = %v, want empty slice", got.Steps[0].EffectiveAllowedTools)
	}
	if !reflect.DeepEqual(got.Steps[1].EffectiveAllowedTools, []string{}) {
		t.Fatalf("final EffectiveAllowedTools = %v, want empty slice", got.Steps[1].EffectiveAllowedTools)
	}
}

func TestMissionInspectCommandWithMissingFileReturnsError(t *testing.T) {
	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mission", "inspect", "--mission-file", filepath.Join(t.TempDir(), "missing.json")})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "failed to read mission file") {
		t.Fatalf("Execute() error = %q, want missing file message", err)
	}
}

func TestMissionInspectCommandWithInvalidJSONReturnsError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mission.json")
	if err := os.WriteFile(path, []byte("{not-json"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mission", "inspect", "--mission-file", path})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "failed to decode mission file") {
		t.Fatalf("Execute() error = %q, want decode failure", err)
	}
}

func TestMissionInspectCommandWithInvalidMissionReturnsValidationError(t *testing.T) {
	path := writeMissionBootstrapJobFile(t, missioncontrol.Job{
		ID:           "job-1",
		MaxAuthority: missioncontrol.AuthorityTierHigh,
		AllowedTools: []string{"read"},
		Plan:         missioncontrol.Plan{ID: "plan-1"},
	})

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mission", "inspect", "--mission-file", path})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "failed to validate mission file") {
		t.Fatalf("Execute() error = %q, want validation failure", err)
	}
	if !strings.Contains(err.Error(), string(missioncontrol.RejectionCodeMissingTerminalFinalStep)) {
		t.Fatalf("Execute() error = %q, want validation error code", err)
	}
}

func TestMissionStatusCommandWithMissingFileReturnsError(t *testing.T) {
	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mission", "status", "--status-file", filepath.Join(t.TempDir(), "missing.json")})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "failed to read mission status file") {
		t.Fatalf("Execute() error = %q, want missing file message", err)
	}
}

func TestMissionStatusCommandWithInvalidFileReturnsError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "status.json")
	if err := os.WriteFile(path, []byte("{not-json"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mission", "status", "--status-file", path})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "failed to decode mission status file") {
		t.Fatalf("Execute() error = %q, want decode failure", err)
	}
}

func TestMissionAssertCommandWithValidStatusFileAndNoConditionsSucceeds(t *testing.T) {
	path := filepath.Join(t.TempDir(), "status.json")
	writeMissionStatusSnapshotFile(t, path, missionStatusSnapshot{
		Active:   true,
		JobID:    "job-1",
		StepID:   "build",
		StepType: string(missioncontrol.StepTypeOneShotCode),
	})

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mission", "assert", "--status-file", path})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
}

func TestMissionAssertCommandOneShotJobIDMatchSucceeds(t *testing.T) {
	path := filepath.Join(t.TempDir(), "status.json")
	writeMissionStatusSnapshotFile(t, path, missionStatusSnapshot{
		Active: true,
		JobID:  "job-1",
		StepID: "build",
	})

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mission", "assert", "--status-file", path, "--job-id", "job-1"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
}

func TestMissionAssertCommandOneShotStepIDMismatchFailsClearly(t *testing.T) {
	path := filepath.Join(t.TempDir(), "status.json")
	writeMissionStatusSnapshotFile(t, path, missionStatusSnapshot{
		Active: true,
		JobID:  "job-1",
		StepID: "build",
	})

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mission", "assert", "--status-file", path, "--step-id", "final"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), `has job_id="job-1" step_id="build" active=true, want step_id="final"`) {
		t.Fatalf("Execute() error = %q, want clear step_id mismatch", err)
	}
}

func TestMissionAssertCommandOneShotActiveMismatchFailsClearly(t *testing.T) {
	path := filepath.Join(t.TempDir(), "status.json")
	writeMissionStatusSnapshotFile(t, path, missionStatusSnapshot{
		Active: false,
		JobID:  "job-1",
		StepID: "build",
	})

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mission", "assert", "--status-file", path, "--active=true"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), `has job_id="job-1" step_id="build" active=false, want active=true`) {
		t.Fatalf("Execute() error = %q, want clear active mismatch", err)
	}
}

func TestMissionAssertCommandOneShotStepTypeMatchSucceeds(t *testing.T) {
	path := filepath.Join(t.TempDir(), "status.json")
	writeMissionStatusSnapshotFile(t, path, missionStatusSnapshot{
		Active:   true,
		JobID:    "job-1",
		StepID:   "build",
		StepType: string(missioncontrol.StepTypeOneShotCode),
	})

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mission", "assert", "--status-file", path, "--step-type", string(missioncontrol.StepTypeOneShotCode)})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
}

func TestMissionAssertCommandOneShotStepTypeMismatchFailsClearly(t *testing.T) {
	path := filepath.Join(t.TempDir(), "status.json")
	writeMissionStatusSnapshotFile(t, path, missionStatusSnapshot{
		Active:   true,
		JobID:    "job-1",
		StepID:   "build",
		StepType: string(missioncontrol.StepTypeDiscussion),
	})

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mission", "assert", "--status-file", path, "--step-type", string(missioncontrol.StepTypeOneShotCode)})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), `has step_type="discussion", want step_type="one_shot_code"`) {
		t.Fatalf("Execute() error = %q, want clear step_type mismatch", err)
	}
}

func TestMissionAssertCommandOneShotRequiredAuthorityMatchSucceeds(t *testing.T) {
	path := filepath.Join(t.TempDir(), "status.json")
	writeMissionStatusSnapshotFile(t, path, missionStatusSnapshot{
		Active:            true,
		JobID:             "job-1",
		StepID:            "build",
		RequiredAuthority: missioncontrol.AuthorityTierMedium,
	})

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mission", "assert", "--status-file", path, "--required-authority", string(missioncontrol.AuthorityTierMedium)})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
}

func TestMissionAssertCommandOneShotRequiredAuthorityMismatchFailsClearly(t *testing.T) {
	path := filepath.Join(t.TempDir(), "status.json")
	writeMissionStatusSnapshotFile(t, path, missionStatusSnapshot{
		Active:            true,
		JobID:             "job-1",
		StepID:            "build",
		RequiredAuthority: missioncontrol.AuthorityTierLow,
	})

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mission", "assert", "--status-file", path, "--required-authority", string(missioncontrol.AuthorityTierMedium)})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), `has required_authority="low", want required_authority="medium"`) {
		t.Fatalf("Execute() error = %q, want clear required_authority mismatch", err)
	}
}

func TestMissionAssertCommandOneShotRequiresApprovalSucceedsWhenTrue(t *testing.T) {
	path := filepath.Join(t.TempDir(), "status.json")
	writeMissionStatusSnapshotFile(t, path, missionStatusSnapshot{
		Active:           true,
		JobID:            "job-1",
		StepID:           "build",
		RequiresApproval: true,
	})

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mission", "assert", "--status-file", path, "--requires-approval"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
}

func TestMissionAssertCommandOneShotRequiresApprovalFailsClearlyWhenFalse(t *testing.T) {
	path := filepath.Join(t.TempDir(), "status.json")
	writeMissionStatusSnapshotFile(t, path, missionStatusSnapshot{
		Active:           true,
		JobID:            "job-1",
		StepID:           "build",
		RequiresApproval: false,
	})

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mission", "assert", "--status-file", path, "--requires-approval"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), `has requires_approval=false, want requires_approval=true`) {
		t.Fatalf("Execute() error = %q, want clear requires_approval mismatch", err)
	}
}

func TestMissionAssertCommandOneShotNoRequiresApprovalSucceedsWhenFalse(t *testing.T) {
	path := filepath.Join(t.TempDir(), "status.json")
	writeMissionStatusSnapshotFile(t, path, missionStatusSnapshot{
		Active:           true,
		JobID:            "job-1",
		StepID:           "build",
		RequiresApproval: false,
	})

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mission", "assert", "--status-file", path, "--no-requires-approval"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
}

func TestMissionAssertCommandOneShotNoToolsSucceedsForEmptyAllowedTools(t *testing.T) {
	path := filepath.Join(t.TempDir(), "status.json")
	writeMissionStatusSnapshotFile(t, path, missionStatusSnapshot{
		Active:       true,
		JobID:        "job-1",
		StepID:       "build",
		AllowedTools: []string{},
	})

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mission", "assert", "--status-file", path, "--no-tools"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
}

func TestMissionAssertCommandOneShotNoToolsFailsClearlyWhenToolsArePresent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "status.json")
	writeMissionStatusSnapshotFile(t, path, missionStatusSnapshot{
		Active:       true,
		JobID:        "job-1",
		StepID:       "build",
		AllowedTools: []string{"read", "write"},
	})

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mission", "assert", "--status-file", path, "--no-tools"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), `has allowed_tools=["read" "write"], want allowed_tools=[]`) {
		t.Fatalf("Execute() error = %q, want clear allowed_tools mismatch", err)
	}
}

func TestMissionAssertCommandOneShotHasToolSucceedsWhenToolIsPresent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "status.json")
	writeMissionStatusSnapshotFile(t, path, missionStatusSnapshot{
		Active:       true,
		JobID:        "job-1",
		StepID:       "build",
		AllowedTools: []string{"read", "write"},
	})

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mission", "assert", "--status-file", path, "--has-tool", "write"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
}

func TestMissionAssertCommandOneShotHasToolFailsClearlyWhenToolIsAbsent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "status.json")
	writeMissionStatusSnapshotFile(t, path, missionStatusSnapshot{
		Active:       true,
		JobID:        "job-1",
		StepID:       "build",
		AllowedTools: []string{"read"},
	})

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mission", "assert", "--status-file", path, "--has-tool", "write"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), `has allowed_tools=["read"], want allowed_tools to include "write"`) {
		t.Fatalf("Execute() error = %q, want clear missing tool message", err)
	}
}

func TestMissionAssertCommandOneShotExactToolSucceedsWhenAllowedToolsExactlyMatch(t *testing.T) {
	path := filepath.Join(t.TempDir(), "status.json")
	writeMissionStatusSnapshotFile(t, path, missionStatusSnapshot{
		Active:       true,
		JobID:        "job-1",
		StepID:       "build",
		AllowedTools: []string{"read", "write"},
	})

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mission", "assert", "--status-file", path, "--exact-tool", "read", "--exact-tool", "write"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
}

func TestMissionAssertCommandOneShotExactToolFailsClearlyWhenAllowedToolsDoNotExactlyMatch(t *testing.T) {
	path := filepath.Join(t.TempDir(), "status.json")
	writeMissionStatusSnapshotFile(t, path, missionStatusSnapshot{
		Active:       true,
		JobID:        "job-1",
		StepID:       "build",
		AllowedTools: []string{"read"},
	})

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mission", "assert", "--status-file", path, "--exact-tool", "read", "--exact-tool", "write"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), `has allowed_tools=["read"], want allowed_tools=["read" "write"]`) {
		t.Fatalf("Execute() error = %q, want clear exact allowed_tools mismatch", err)
	}
}

func TestMissionAssertCommandWaitSucceedsWhenStatusFileChangesBeforeTimeout(t *testing.T) {
	path := filepath.Join(t.TempDir(), "status.json")
	writeMissionStatusSnapshotFile(t, path, missionStatusSnapshot{
		Active: true,
		JobID:  "job-1",
		StepID: "build",
	})

	done := make(chan error, 1)
	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{
		"mission", "assert",
		"--status-file", path,
		"--step-id", "final",
		"--wait-timeout", "250ms",
	})
	go func() {
		done <- cmd.Execute()
	}()

	select {
	case err := <-done:
		t.Fatalf("Execute() returned before matching status update: %v", err)
	case <-time.After(20 * time.Millisecond):
	}

	writeMissionStatusSnapshotFile(t, path, missionStatusSnapshot{
		Active: true,
		JobID:  "job-1",
		StepID: "final",
	})

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Execute() did not return after matching status update")
	}
}

func TestMissionAssertCommandWaitSucceedsWhenAllowedToolsChangeBeforeTimeout(t *testing.T) {
	path := filepath.Join(t.TempDir(), "status.json")
	writeMissionStatusSnapshotFile(t, path, missionStatusSnapshot{
		Active:       true,
		JobID:        "job-1",
		StepID:       "build",
		AllowedTools: []string{"read"},
	})

	done := make(chan error, 1)
	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{
		"mission", "assert",
		"--status-file", path,
		"--has-tool", "write",
		"--wait-timeout", "250ms",
	})
	go func() {
		done <- cmd.Execute()
	}()

	select {
	case err := <-done:
		t.Fatalf("Execute() returned before matching status update: %v", err)
	case <-time.After(20 * time.Millisecond):
	}

	writeMissionStatusSnapshotFile(t, path, missionStatusSnapshot{
		Active:       true,
		JobID:        "job-1",
		StepID:       "build",
		AllowedTools: []string{"read", "write"},
	})

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Execute() did not return after matching status update")
	}
}

func TestMissionAssertCommandWaitSucceedsWhenAllowedToolsExactlyMatchBeforeTimeout(t *testing.T) {
	path := filepath.Join(t.TempDir(), "status.json")
	writeMissionStatusSnapshotFile(t, path, missionStatusSnapshot{
		Active:       true,
		JobID:        "job-1",
		StepID:       "build",
		AllowedTools: []string{"read"},
	})

	done := make(chan error, 1)
	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{
		"mission", "assert",
		"--status-file", path,
		"--exact-tool", "read",
		"--exact-tool", "write",
		"--wait-timeout", "250ms",
	})
	go func() {
		done <- cmd.Execute()
	}()

	select {
	case err := <-done:
		t.Fatalf("Execute() returned before matching status update: %v", err)
	case <-time.After(20 * time.Millisecond):
	}

	writeMissionStatusSnapshotFile(t, path, missionStatusSnapshot{
		Active:       true,
		JobID:        "job-1",
		StepID:       "build",
		AllowedTools: []string{"read", "write"},
	})

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Execute() did not return after matching status update")
	}
}

func TestMissionAssertCommandWaitTimesOutWhenValuesNeverMatch(t *testing.T) {
	path := filepath.Join(t.TempDir(), "status.json")
	writeMissionStatusSnapshotFile(t, path, missionStatusSnapshot{
		Active: true,
		JobID:  "job-1",
		StepID: "build",
	})

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{
		"mission", "assert",
		"--status-file", path,
		"--step-id", "final",
		"--wait-timeout", "75ms",
	})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "timed out waiting up to 75ms") {
		t.Fatalf("Execute() error = %q, want timeout error", err)
	}
	if !strings.Contains(err.Error(), `has job_id="job-1" step_id="build" active=true, want step_id="final"`) {
		t.Fatalf("Execute() error = %q, want observed and expected values", err)
	}
}

func TestMissionAssertCommandWithMissingStatusFileReturnsClearError(t *testing.T) {
	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mission", "assert", "--status-file", filepath.Join(t.TempDir(), "missing.json")})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "mission status file") || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("Execute() error = %q, want missing file message", err)
	}
}

func TestMissionAssertCommandWithInvalidJSONReturnsClearError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "status.json")
	if err := os.WriteFile(path, []byte("{not-json"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mission", "assert", "--status-file", path})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "failed to decode mission status file") {
		t.Fatalf("Execute() error = %q, want decode failure", err)
	}
}

func TestMissionAssertCommandNoToolsAndHasToolReturnsClearArgumentError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "status.json")
	writeMissionStatusSnapshotFile(t, path, missionStatusSnapshot{
		Active:       true,
		JobID:        "job-1",
		StepID:       "build",
		AllowedTools: []string{"read"},
	})

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mission", "assert", "--status-file", path, "--no-tools", "--has-tool", "read"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "--no-tools and --has-tool cannot be used together") {
		t.Fatalf("Execute() error = %q, want clear argument error", err)
	}
}

func TestMissionAssertCommandNoToolsAndExactToolReturnsClearArgumentError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "status.json")
	writeMissionStatusSnapshotFile(t, path, missionStatusSnapshot{
		Active:       true,
		JobID:        "job-1",
		StepID:       "build",
		AllowedTools: []string{"read"},
	})

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mission", "assert", "--status-file", path, "--no-tools", "--exact-tool", "read"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "--no-tools and --exact-tool cannot be used together") {
		t.Fatalf("Execute() error = %q, want clear argument error", err)
	}
}

func TestMissionAssertCommandHasToolAndExactToolReturnsClearArgumentError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "status.json")
	writeMissionStatusSnapshotFile(t, path, missionStatusSnapshot{
		Active:       true,
		JobID:        "job-1",
		StepID:       "build",
		AllowedTools: []string{"read"},
	})

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mission", "assert", "--status-file", path, "--has-tool", "read", "--exact-tool", "read"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "--has-tool and --exact-tool cannot be used together") {
		t.Fatalf("Execute() error = %q, want clear argument error", err)
	}
}

func TestMissionAssertCommandRequiresApprovalAndNoRequiresApprovalReturnsClearArgumentError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "status.json")
	writeMissionStatusSnapshotFile(t, path, missionStatusSnapshot{
		Active: true,
		JobID:  "job-1",
		StepID: "build",
	})

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mission", "assert", "--status-file", path, "--requires-approval", "--no-requires-approval"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "--requires-approval and --no-requires-approval cannot be used together") {
		t.Fatalf("Execute() error = %q, want clear argument error", err)
	}
}

func TestMissionAssertStepCommandSucceedsWhenStatusMatchesMissionStep(t *testing.T) {
	job := missioncontrol.Job{
		ID:           "job-1",
		MaxAuthority: missioncontrol.AuthorityTierHigh,
		AllowedTools: []string{"write", "read", "search"},
		Plan: missioncontrol.Plan{
			ID: "plan-1",
			Steps: []missioncontrol.Step{
				{
					ID:                "build",
					Type:              missioncontrol.StepTypeOneShotCode,
					RequiredAuthority: missioncontrol.AuthorityTierLow,
					AllowedTools:      []string{"write", "read"},
				},
				{
					ID:        "final",
					Type:      missioncontrol.StepTypeFinalResponse,
					DependsOn: []string{"build"},
				},
			},
		},
	}
	missionPath := writeMissionBootstrapJobFile(t, job)
	statusPath := filepath.Join(t.TempDir(), "status.json")
	writeMissionStatusSnapshotFile(t, statusPath, missionStatusSnapshot{
		Active:       true,
		JobID:        "job-1",
		StepID:       "build",
		StepType:     string(missioncontrol.StepTypeOneShotCode),
		AllowedTools: []string{"read", "write"},
	})

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mission", "assert-step", "--mission-file", missionPath, "--status-file", statusPath, "--step-id", "build"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
}

func TestMissionAssertStepCommandSucceedsForZeroToolStepWhenStatusAllowedToolsIsNil(t *testing.T) {
	job := missioncontrol.Job{
		ID:           "phone-discussion-v1",
		MaxAuthority: missioncontrol.AuthorityTierLow,
		AllowedTools: []string{},
		Plan: missioncontrol.Plan{
			ID: "plan-1",
			Steps: []missioncontrol.Step{
				{
					ID:                "discuss",
					Type:              missioncontrol.StepTypeDiscussion,
					RequiredAuthority: missioncontrol.AuthorityTierLow,
				},
				{
					ID:        "final",
					Type:      missioncontrol.StepTypeFinalResponse,
					DependsOn: []string{"discuss"},
				},
			},
		},
	}
	missionPath := writeMissionBootstrapJobFile(t, job)
	statusPath := filepath.Join(t.TempDir(), "status.json")
	writeMissionStatusSnapshotFile(t, statusPath, missionStatusSnapshot{
		Active:    true,
		JobID:     "phone-discussion-v1",
		StepID:    "discuss",
		StepType:  string(missioncontrol.StepTypeDiscussion),
		UpdatedAt: "2026-03-21T10:00:00Z",
	})

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mission", "assert-step", "--mission-file", missionPath, "--status-file", statusPath, "--step-id", "discuss"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
}

func TestMissionAssertStepCommandFailsClearlyWhenAllowedToolsDoNotExactlyMatch(t *testing.T) {
	job := missioncontrol.Job{
		ID:           "job-1",
		MaxAuthority: missioncontrol.AuthorityTierHigh,
		AllowedTools: []string{"read", "write"},
		Plan: missioncontrol.Plan{
			ID: "plan-1",
			Steps: []missioncontrol.Step{
				{
					ID:                "build",
					Type:              missioncontrol.StepTypeOneShotCode,
					RequiredAuthority: missioncontrol.AuthorityTierLow,
					AllowedTools:      []string{"write", "read"},
				},
				{
					ID:        "final",
					Type:      missioncontrol.StepTypeFinalResponse,
					DependsOn: []string{"build"},
				},
			},
		},
	}
	missionPath := writeMissionBootstrapJobFile(t, job)
	statusPath := filepath.Join(t.TempDir(), "status.json")
	writeMissionStatusSnapshotFile(t, statusPath, missionStatusSnapshot{
		Active:       true,
		JobID:        "job-1",
		StepID:       "build",
		StepType:     string(missioncontrol.StepTypeOneShotCode),
		AllowedTools: []string{"read"},
	})

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mission", "assert-step", "--mission-file", missionPath, "--status-file", statusPath, "--step-id", "build"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), `has allowed_tools=["read"], want allowed_tools=["read" "write"]`) {
		t.Fatalf("Execute() error = %q, want exact allowed_tools mismatch", err)
	}
}

func TestMissionAssertStepCommandUnknownStepReturnsClearError(t *testing.T) {
	missionPath := writeMissionBootstrapJobFile(t, testMissionBootstrapJob())
	statusPath := filepath.Join(t.TempDir(), "status.json")
	writeMissionStatusSnapshotFile(t, statusPath, missionStatusSnapshot{
		Active: true,
		JobID:  "job-1",
		StepID: "build",
	})

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mission", "assert-step", "--mission-file", missionPath, "--status-file", statusPath, "--step-id", "missing"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "failed to validate mission file") {
		t.Fatalf("Execute() error = %q, want validation failure", err)
	}
	if !strings.Contains(err.Error(), string(missioncontrol.RejectionCodeUnknownStep)) {
		t.Fatalf("Execute() error = %q, want unknown_step code", err)
	}
	if !strings.Contains(err.Error(), `step "missing" not found in plan`) {
		t.Fatalf("Execute() error = %q, want missing step message", err)
	}
}

func TestMissionAssertStepCommandWaitSucceedsWhenStatusChangesBeforeTimeout(t *testing.T) {
	job := missioncontrol.Job{
		ID:           "job-1",
		MaxAuthority: missioncontrol.AuthorityTierHigh,
		AllowedTools: []string{"read", "write"},
		Plan: missioncontrol.Plan{
			ID: "plan-1",
			Steps: []missioncontrol.Step{
				{
					ID:                "build",
					Type:              missioncontrol.StepTypeOneShotCode,
					RequiredAuthority: missioncontrol.AuthorityTierLow,
					AllowedTools:      []string{"write", "read"},
				},
				{
					ID:        "final",
					Type:      missioncontrol.StepTypeFinalResponse,
					DependsOn: []string{"build"},
				},
			},
		},
	}
	missionPath := writeMissionBootstrapJobFile(t, job)
	statusPath := filepath.Join(t.TempDir(), "status.json")
	writeMissionStatusSnapshotFile(t, statusPath, missionStatusSnapshot{
		Active:       true,
		JobID:        "job-1",
		StepID:       "build",
		StepType:     string(missioncontrol.StepTypeOneShotCode),
		AllowedTools: []string{"read"},
	})

	done := make(chan error, 1)
	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{
		"mission", "assert-step",
		"--mission-file", missionPath,
		"--status-file", statusPath,
		"--step-id", "build",
		"--wait-timeout", "250ms",
	})
	go func() {
		done <- cmd.Execute()
	}()

	select {
	case err := <-done:
		t.Fatalf("Execute() returned before matching status update: %v", err)
	case <-time.After(20 * time.Millisecond):
	}

	writeMissionStatusSnapshotFile(t, statusPath, missionStatusSnapshot{
		Active:       true,
		JobID:        "job-1",
		StepID:       "build",
		StepType:     string(missioncontrol.StepTypeOneShotCode),
		AllowedTools: []string{"read", "write"},
	})

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Execute() did not return after matching status update")
	}
}

func TestMissionAssertStepCommandWithInvalidMissionReturnsValidationError(t *testing.T) {
	missionPath := writeMissionBootstrapJobFile(t, missioncontrol.Job{
		ID:           "job-1",
		MaxAuthority: missioncontrol.AuthorityTierHigh,
		AllowedTools: []string{"read"},
		Plan:         missioncontrol.Plan{ID: "plan-1"},
	})
	statusPath := filepath.Join(t.TempDir(), "status.json")
	writeMissionStatusSnapshotFile(t, statusPath, missionStatusSnapshot{
		Active: true,
		JobID:  "job-1",
		StepID: "build",
	})

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mission", "assert-step", "--mission-file", missionPath, "--status-file", statusPath, "--step-id", "build"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "failed to validate mission file") {
		t.Fatalf("Execute() error = %q, want validation failure", err)
	}
	if !strings.Contains(err.Error(), string(missioncontrol.RejectionCodeMissingTerminalFinalStep)) {
		t.Fatalf("Execute() error = %q, want validation error code", err)
	}
}

func TestMissionSetStepCommandInvalidControlPathReturnsClearError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "control-dir")
	if err := os.Mkdir(path, 0o755); err != nil {
		t.Fatalf("Mkdir() error = %v", err)
	}

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mission", "set-step", "--control-file", path, "--step-id", "final"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "failed to write mission step control file") {
		t.Fatalf("Execute() error = %q, want write failure", err)
	}
	assertNoAtomicTempFiles(t, dir, filepath.Base(path))
}

func TestMissionSetStepCommandWithoutStatusFilePreservesCurrentBehavior(t *testing.T) {
	path := filepath.Join(t.TempDir(), "control.json")

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mission", "set-step", "--control-file", path, "--step-id", "final"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	control := readMissionStepControlFile(t, path)
	if control.StepID != "final" {
		t.Fatalf("StepID = %q, want %q", control.StepID, "final")
	}
}

func TestMissionSetStepCommandLeavesNoTempFileOnSuccess(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "control.json")

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mission", "set-step", "--control-file", path, "--step-id", "final"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	control := readMissionStepControlFile(t, path)
	if control.StepID != "final" {
		t.Fatalf("StepID = %q, want %q", control.StepID, "final")
	}
	assertNoAtomicTempFiles(t, dir, filepath.Base(path))
}

func TestMissionSetStepCommandWritesUpdatedAt(t *testing.T) {
	path := filepath.Join(t.TempDir(), "control.json")

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mission", "set-step", "--control-file", path, "--step-id", "final"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	control := readMissionStepControlFile(t, path)
	if control.UpdatedAt == "" {
		t.Fatal("UpdatedAt = empty string, want RFC3339Nano timestamp")
	}

	parsed, err := time.Parse(time.RFC3339Nano, control.UpdatedAt)
	if err != nil {
		t.Fatalf("time.Parse() error = %v", err)
	}
	if _, offset := parsed.Zone(); offset != 0 {
		t.Fatalf("UpdatedAt offset = %d, want 0", offset)
	}
}

func TestMissionSetStepCommandWithStatusFileWaitsWhenMatchingSnapshotUpdatedAtIsUnchanged(t *testing.T) {
	controlPath := filepath.Join(t.TempDir(), "control.json")
	statusPath := filepath.Join(t.TempDir(), "status.json")
	writeMissionStatusSnapshotFile(t, statusPath, missionStatusSnapshot{
		Active:    true,
		StepID:    "final",
		UpdatedAt: "2026-03-20T12:00:00Z",
	})

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{
		"mission", "set-step",
		"--control-file", controlPath,
		"--step-id", "final",
		"--status-file", statusPath,
		"--wait-timeout", "250ms",
	})

	done := make(chan error, 1)
	go func() {
		done <- cmd.Execute()
	}()

	select {
	case err := <-done:
		t.Fatalf("Execute() returned before fresh status update: %v", err)
	case <-time.After(20 * time.Millisecond):
	}

	writeMissionStatusSnapshotFile(t, statusPath, missionStatusSnapshot{
		Active:    true,
		StepID:    "final",
		UpdatedAt: "2026-03-20T12:00:01Z",
	})

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Execute() did not return after fresh status update")
	}

	control := readMissionStepControlFile(t, controlPath)
	if control.StepID != "final" {
		t.Fatalf("StepID = %q, want %q", control.StepID, "final")
	}
}

func TestMissionSetStepCommandWithStatusFileWithoutMissionFilePreservesCurrentBehavior(t *testing.T) {
	controlPath := filepath.Join(t.TempDir(), "control.json")
	statusPath := filepath.Join(t.TempDir(), "status.json")
	writeMissionStatusSnapshotFile(t, statusPath, missionStatusSnapshot{
		Active:    true,
		JobID:     "other-job",
		StepID:    "final",
		UpdatedAt: "2026-03-20T12:00:00Z",
	})

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{
		"mission", "set-step",
		"--control-file", controlPath,
		"--step-id", "final",
		"--status-file", statusPath,
		"--wait-timeout", "250ms",
	})

	done := make(chan error, 1)
	go func() {
		done <- cmd.Execute()
	}()

	select {
	case err := <-done:
		t.Fatalf("Execute() returned before fresh status update: %v", err)
	case <-time.After(20 * time.Millisecond):
	}

	writeMissionStatusSnapshotFile(t, statusPath, missionStatusSnapshot{
		Active:    true,
		JobID:     "different-job",
		StepID:    "final",
		UpdatedAt: "2026-03-20T12:00:01Z",
	})

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Execute() did not return after fresh status update")
	}
}

func TestMissionSetStepCommandWithStatusFileSucceedsWhenSnapshotChangesBeforeTimeout(t *testing.T) {
	controlPath := filepath.Join(t.TempDir(), "control.json")
	statusPath := filepath.Join(t.TempDir(), "status.json")
	writeMissionStatusSnapshotFile(t, statusPath, missionStatusSnapshot{
		Active:    true,
		StepID:    "build",
		UpdatedAt: "2026-03-20T12:00:00Z",
	})

	errCh := make(chan error, 1)
	go func() {
		time.Sleep(25 * time.Millisecond)
		data, err := json.Marshal(missionStatusSnapshot{
			Active:    true,
			StepID:    "final",
			UpdatedAt: "2026-03-20T12:00:01Z",
		})
		if err != nil {
			errCh <- err
			return
		}
		errCh <- os.WriteFile(statusPath, data, 0o644)
	}()

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{
		"mission", "set-step",
		"--control-file", controlPath,
		"--step-id", "final",
		"--status-file", statusPath,
		"--wait-timeout", "250ms",
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if err := <-errCh; err != nil {
		t.Fatalf("status update error = %v", err)
	}

	control := readMissionStepControlFile(t, controlPath)
	if control.StepID != "final" {
		t.Fatalf("StepID = %q, want %q", control.StepID, "final")
	}
}

func TestMissionSetStepCommandWithMissionFileAndStatusFileSucceedsWhenFreshSnapshotMatchesStepAndJob(t *testing.T) {
	controlPath := filepath.Join(t.TempDir(), "control.json")
	statusPath := filepath.Join(t.TempDir(), "status.json")
	job := testMissionBootstrapJob()
	missionPath := writeMissionBootstrapJobFile(t, job)
	logBuf, restoreLog := captureStandardLogger(t)
	defer restoreLog()
	writeMissionStatusSnapshotFile(t, statusPath, missionStatusSnapshot{
		Active:    true,
		JobID:     "other-job",
		StepID:    "build",
		UpdatedAt: "2026-03-20T12:00:00Z",
	})

	done := make(chan error, 1)
	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{
		"mission", "set-step",
		"--control-file", controlPath,
		"--step-id", "final",
		"--mission-file", missionPath,
		"--status-file", statusPath,
		"--wait-timeout", "250ms",
	})
	go func() {
		done <- cmd.Execute()
	}()

	select {
	case err := <-done:
		t.Fatalf("Execute() returned before matching status update: %v", err)
	case <-time.After(20 * time.Millisecond):
	}

	writeMissionStatusSnapshotFile(t, statusPath, missionStatusSnapshot{
		Active:    true,
		JobID:     job.ID,
		StepID:    "final",
		UpdatedAt: "2026-03-20T12:00:01Z",
	})

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Execute() did not return after matching status update")
	}

	logOutput := logBuf.String()
	if !strings.Contains(logOutput, "mission set-step status confirmation succeeded") {
		t.Fatalf("log output = %q, want set-step success log", logOutput)
	}
	if !strings.Contains(logOutput, `job_id="`+job.ID+`"`) {
		t.Fatalf("log output = %q, want job_id", logOutput)
	}
	if !strings.Contains(logOutput, `step_id="final"`) {
		t.Fatalf("log output = %q, want step_id", logOutput)
	}
	if !strings.Contains(logOutput, `control_file="`+controlPath+`"`) {
		t.Fatalf("log output = %q, want control file path", logOutput)
	}
	if !strings.Contains(logOutput, `status_file="`+statusPath+`"`) {
		t.Fatalf("log output = %q, want status file path", logOutput)
	}
}

func TestMissionSetStepCommandWithMissionFileAndStatusFileWaitsWhenStepMatchesButJobDoesNot(t *testing.T) {
	controlPath := filepath.Join(t.TempDir(), "control.json")
	statusPath := filepath.Join(t.TempDir(), "status.json")
	job := testMissionBootstrapJob()
	missionPath := writeMissionBootstrapJobFile(t, job)
	writeMissionStatusSnapshotFile(t, statusPath, missionStatusSnapshot{
		Active:    true,
		JobID:     "other-job",
		StepID:    "build",
		UpdatedAt: "2026-03-20T12:00:00Z",
	})

	done := make(chan error, 1)
	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{
		"mission", "set-step",
		"--control-file", controlPath,
		"--step-id", "final",
		"--mission-file", missionPath,
		"--status-file", statusPath,
		"--wait-timeout", "250ms",
	})
	go func() {
		done <- cmd.Execute()
	}()

	select {
	case err := <-done:
		t.Fatalf("Execute() returned before status update: %v", err)
	case <-time.After(20 * time.Millisecond):
	}

	writeMissionStatusSnapshotFile(t, statusPath, missionStatusSnapshot{
		Active:    true,
		JobID:     "other-job",
		StepID:    "final",
		UpdatedAt: "2026-03-20T12:00:01Z",
	})

	select {
	case err := <-done:
		t.Fatalf("Execute() returned while job_id was still mismatched: %v", err)
	case <-time.After(60 * time.Millisecond):
	}

	writeMissionStatusSnapshotFile(t, statusPath, missionStatusSnapshot{
		Active:    true,
		JobID:     job.ID,
		StepID:    "final",
		UpdatedAt: "2026-03-20T12:00:02Z",
	})

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Execute() did not return after matching job_id update")
	}
}

func TestMissionSetStepCommandWithMissionFileAndStatusFileTimesOutWhenJobIDNeverMatches(t *testing.T) {
	controlPath := filepath.Join(t.TempDir(), "control.json")
	statusPath := filepath.Join(t.TempDir(), "status.json")
	job := testMissionBootstrapJob()
	missionPath := writeMissionBootstrapJobFile(t, job)
	writeMissionStatusSnapshotFile(t, statusPath, missionStatusSnapshot{
		Active:    true,
		JobID:     "other-job",
		StepID:    "final",
		UpdatedAt: "2026-03-20T12:00:00Z",
	})

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{
		"mission", "set-step",
		"--control-file", controlPath,
		"--step-id", "final",
		"--mission-file", missionPath,
		"--status-file", statusPath,
		"--wait-timeout", "75ms",
	})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "timed out waiting") {
		t.Fatalf("Execute() error = %q, want timeout message", err)
	}
	if !strings.Contains(err.Error(), `job_id="other-job"`) {
		t.Fatalf("Execute() error = %q, want observed job_id", err)
	}
	if !strings.Contains(err.Error(), `job_id="job-1"`) {
		t.Fatalf("Execute() error = %q, want expected job_id", err)
	}
}

func TestMissionSetStepCommandWithStatusFileTimesOutWhenStepNeverMatches(t *testing.T) {
	controlPath := filepath.Join(t.TempDir(), "control.json")
	statusPath := filepath.Join(t.TempDir(), "status.json")
	writeMissionStatusSnapshotFile(t, statusPath, missionStatusSnapshot{
		Active:    true,
		StepID:    "build",
		UpdatedAt: "2026-03-20T12:00:00Z",
	})

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{
		"mission", "set-step",
		"--control-file", controlPath,
		"--step-id", "final",
		"--status-file", statusPath,
		"--wait-timeout", "75ms",
	})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "timed out waiting") {
		t.Fatalf("Execute() error = %q, want timeout message", err)
	}
	if !strings.Contains(err.Error(), `want active=true step_id="final"`) {
		t.Fatalf("Execute() error = %q, want requested step confirmation message", err)
	}

	control := readMissionStepControlFile(t, controlPath)
	if control.StepID != "final" {
		t.Fatalf("StepID = %q, want %q", control.StepID, "final")
	}
}

func TestMissionSetStepCommandWithStatusFileTimesOutWhenMatchingSnapshotIsNotFresh(t *testing.T) {
	controlPath := filepath.Join(t.TempDir(), "control.json")
	statusPath := filepath.Join(t.TempDir(), "status.json")
	writeMissionStatusSnapshotFile(t, statusPath, missionStatusSnapshot{
		Active:    true,
		StepID:    "final",
		UpdatedAt: "2026-03-20T12:00:00Z",
	})

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{
		"mission", "set-step",
		"--control-file", controlPath,
		"--step-id", "final",
		"--status-file", statusPath,
		"--wait-timeout", "75ms",
	})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "timed out waiting") {
		t.Fatalf("Execute() error = %q, want timeout message", err)
	}
	if !strings.Contains(err.Error(), "fresh matching update") {
		t.Fatalf("Execute() error = %q, want freshness message", err)
	}

	control := readMissionStepControlFile(t, controlPath)
	if control.StepID != "final" {
		t.Fatalf("StepID = %q, want %q", control.StepID, "final")
	}
}

func TestMissionSetStepCommandWithInvalidStatusJSONReturnsError(t *testing.T) {
	controlPath := filepath.Join(t.TempDir(), "control.json")
	statusPath := filepath.Join(t.TempDir(), "status.json")
	if err := os.WriteFile(statusPath, []byte("{"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{
		"mission", "set-step",
		"--control-file", controlPath,
		"--step-id", "final",
		"--status-file", statusPath,
		"--wait-timeout", "75ms",
	})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "timed out waiting") {
		t.Fatalf("Execute() error = %q, want timeout message", err)
	}
	if !strings.Contains(err.Error(), "failed to decode mission status file") {
		t.Fatalf("Execute() error = %q, want status decode failure", err)
	}
}

func TestMissionSetStepCommandWithNoPriorValidStatusSnapshotSucceedsWhenMatchingSnapshotAppears(t *testing.T) {
	controlPath := filepath.Join(t.TempDir(), "control.json")
	statusPath := filepath.Join(t.TempDir(), "status.json")
	if err := os.WriteFile(statusPath, []byte("{"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	errCh := make(chan error, 1)
	go func() {
		time.Sleep(25 * time.Millisecond)
		data, err := json.Marshal(missionStatusSnapshot{
			Active:    true,
			StepID:    "final",
			UpdatedAt: "2026-03-20T12:00:01Z",
		})
		if err != nil {
			errCh <- err
			return
		}
		errCh <- os.WriteFile(statusPath, data, 0o644)
	}()

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{
		"mission", "set-step",
		"--control-file", controlPath,
		"--step-id", "final",
		"--status-file", statusPath,
		"--wait-timeout", "250ms",
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if err := <-errCh; err != nil {
		t.Fatalf("status update error = %v", err)
	}

	control := readMissionStepControlFile(t, controlPath)
	if control.StepID != "final" {
		t.Fatalf("StepID = %q, want %q", control.StepID, "final")
	}
}

func TestMissionSetStepCommandWithMissingStatusFileReturnsErrorAfterWaiting(t *testing.T) {
	controlPath := filepath.Join(t.TempDir(), "control.json")
	statusPath := filepath.Join(t.TempDir(), "missing-status.json")

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{
		"mission", "set-step",
		"--control-file", controlPath,
		"--step-id", "final",
		"--status-file", statusPath,
		"--wait-timeout", "75ms",
	})

	start := time.Now()
	err := cmd.Execute()
	elapsed := time.Since(start)
	if err == nil {
		t.Fatal("Execute() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "timed out waiting") {
		t.Fatalf("Execute() error = %q, want timeout message", err)
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Fatalf("Execute() error = %q, want missing status file message", err)
	}
	if elapsed < 50*time.Millisecond {
		t.Fatalf("Execute() elapsed = %v, want wait before timeout", elapsed)
	}

	control := readMissionStepControlFile(t, controlPath)
	if control.StepID != "final" {
		t.Fatalf("StepID = %q, want %q", control.StepID, "final")
	}
}

func TestMissionSetStepCommandWithMissionFileWritesControlFile(t *testing.T) {
	controlPath := filepath.Join(t.TempDir(), "control.json")
	missionPath := writeMissionBootstrapJobFile(t, testMissionBootstrapJob())

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{
		"mission", "set-step",
		"--control-file", controlPath,
		"--step-id", "final",
		"--mission-file", missionPath,
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	control := readMissionStepControlFile(t, controlPath)
	if control.StepID != "final" {
		t.Fatalf("StepID = %q, want %q", control.StepID, "final")
	}
	if control.UpdatedAt == "" {
		t.Fatal("UpdatedAt = empty string, want RFC3339Nano timestamp")
	}
}

func TestMissionSetStepCommandWithMissingMissionFileReturnsError(t *testing.T) {
	controlPath := filepath.Join(t.TempDir(), "control.json")
	missionPath := filepath.Join(t.TempDir(), "missing-mission.json")

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{
		"mission", "set-step",
		"--control-file", controlPath,
		"--step-id", "final",
		"--mission-file", missionPath,
	})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "failed to read mission file") {
		t.Fatalf("Execute() error = %q, want mission read failure", err)
	}

	assertMissionStepControlFileMissing(t, controlPath)
}

func TestMissionSetStepCommandWithInvalidMissionJSONReturnsError(t *testing.T) {
	controlPath := filepath.Join(t.TempDir(), "control.json")
	missionPath := filepath.Join(t.TempDir(), "mission.json")
	if err := os.WriteFile(missionPath, []byte("{"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{
		"mission", "set-step",
		"--control-file", controlPath,
		"--step-id", "final",
		"--mission-file", missionPath,
	})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "failed to decode mission file") {
		t.Fatalf("Execute() error = %q, want mission decode failure", err)
	}

	assertMissionStepControlFileMissing(t, controlPath)
}

func TestMissionSetStepCommandWithInvalidMissionReturnsError(t *testing.T) {
	controlPath := filepath.Join(t.TempDir(), "control.json")
	missionPath := writeMissionBootstrapJobFile(t, missioncontrol.Job{
		ID:           "job-1",
		MaxAuthority: missioncontrol.AuthorityTierHigh,
		AllowedTools: []string{"read"},
		Plan: missioncontrol.Plan{
			ID: "plan-1",
			Steps: []missioncontrol.Step{
				{
					ID:                "draft",
					Type:              missioncontrol.StepTypeDiscussion,
					RequiredAuthority: missioncontrol.AuthorityTierLow,
					AllowedTools:      []string{"read"},
				},
			},
		},
	})

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{
		"mission", "set-step",
		"--control-file", controlPath,
		"--step-id", "draft",
		"--mission-file", missionPath,
	})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "failed to validate mission file") {
		t.Fatalf("Execute() error = %q, want mission validation failure", err)
	}
	if !strings.Contains(err.Error(), string(missioncontrol.RejectionCodeMissingTerminalFinalStep)) {
		t.Fatalf("Execute() error = %q, want validation error code", err)
	}

	assertMissionStepControlFileMissing(t, controlPath)
}

func TestMissionSetStepCommandWithUnknownMissionStepReturnsError(t *testing.T) {
	controlPath := filepath.Join(t.TempDir(), "control.json")
	missionPath := writeMissionBootstrapJobFile(t, testMissionBootstrapJob())

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{
		"mission", "set-step",
		"--control-file", controlPath,
		"--step-id", "missing",
		"--mission-file", missionPath,
	})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "failed to validate mission file") {
		t.Fatalf("Execute() error = %q, want mission validation failure", err)
	}
	if !strings.Contains(err.Error(), string(missioncontrol.RejectionCodeUnknownStep)) {
		t.Fatalf("Execute() error = %q, want unknown step error code", err)
	}

	assertMissionStepControlFileMissing(t, controlPath)
}

func TestMissionSetStepCommandWithoutRequiredFlagsReturnsError(t *testing.T) {
	t.Run("missing control file", func(t *testing.T) {
		cmd := NewRootCmd()
		cmd.SetOut(&bytes.Buffer{})
		cmd.SetErr(&bytes.Buffer{})
		cmd.SetArgs([]string{"mission", "set-step"})

		err := cmd.Execute()
		if err == nil {
			t.Fatal("Execute() error = nil, want non-nil")
		}
		if !strings.Contains(err.Error(), "--control-file is required") {
			t.Fatalf("Execute() error = %q, want missing control-file message", err)
		}
	})

	t.Run("missing step id", func(t *testing.T) {
		cmd := NewRootCmd()
		cmd.SetOut(&bytes.Buffer{})
		cmd.SetErr(&bytes.Buffer{})
		cmd.SetArgs([]string{"mission", "set-step", "--control-file", filepath.Join(t.TempDir(), "control.json")})

		err := cmd.Execute()
		if err == nil {
			t.Fatal("Execute() error = nil, want non-nil")
		}
		if !strings.Contains(err.Error(), "--step-id is required") {
			t.Fatalf("Execute() error = %q, want missing step-id message", err)
		}
	})
}

func TestConfigureMissionBootstrapDefaultUnchanged(t *testing.T) {
	ag := newMissionBootstrapTestLoop()
	cmd := newMissionBootstrapTestCommand()

	if err := configureMissionBootstrap(cmd, ag); err != nil {
		t.Fatalf("configureMissionBootstrap() error = %v", err)
	}

	if ag.MissionRequired() {
		t.Fatal("MissionRequired() = true, want false")
	}

	if _, ok := ag.ActiveMissionStep(); ok {
		t.Fatal("ActiveMissionStep() ok = true, want false")
	}
}

func TestConfigureMissionBootstrapMissionRequiredEnablesMode(t *testing.T) {
	ag := newMissionBootstrapTestLoop()
	cmd := newMissionBootstrapTestCommand()
	if err := cmd.Flags().Set("mission-required", "true"); err != nil {
		t.Fatalf("Flags().Set() error = %v", err)
	}

	if err := configureMissionBootstrap(cmd, ag); err != nil {
		t.Fatalf("configureMissionBootstrap() error = %v", err)
	}

	if !ag.MissionRequired() {
		t.Fatal("MissionRequired() = false, want true")
	}
}

func TestConfigureMissionBootstrapMissionFileActivatesStep(t *testing.T) {
	ag := newMissionBootstrapTestLoop()
	cmd := newMissionBootstrapTestCommand()
	missionFile := writeMissionBootstrapJobFile(t, testMissionBootstrapJob())
	if err := cmd.Flags().Set("mission-file", missionFile); err != nil {
		t.Fatalf("Flags().Set(mission-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step", "build"); err != nil {
		t.Fatalf("Flags().Set(mission-step) error = %v", err)
	}

	if err := configureMissionBootstrap(cmd, ag); err != nil {
		t.Fatalf("configureMissionBootstrap() error = %v", err)
	}

	ec, ok := ag.ActiveMissionStep()
	if !ok {
		t.Fatal("ActiveMissionStep() ok = false, want true")
	}

	if ec.Job == nil || ec.Step == nil {
		t.Fatalf("ActiveMissionStep() = %#v, want non-nil job and step", ec)
	}

	if ec.Job.ID != "job-1" {
		t.Fatalf("ActiveMissionStep().Job.ID = %q, want %q", ec.Job.ID, "job-1")
	}

	if ec.Step.ID != "build" {
		t.Fatalf("ActiveMissionStep().Step.ID = %q, want %q", ec.Step.ID, "build")
	}
}

func TestConfigureMissionBootstrapInvalidMissionFileFailsStartup(t *testing.T) {
	ag := newMissionBootstrapTestLoop()
	cmd := newMissionBootstrapTestCommand()
	missionFile := filepath.Join(t.TempDir(), "mission.json")
	if err := os.WriteFile(missionFile, []byte("{not-json"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if err := cmd.Flags().Set("mission-file", missionFile); err != nil {
		t.Fatalf("Flags().Set(mission-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step", "build"); err != nil {
		t.Fatalf("Flags().Set(mission-step) error = %v", err)
	}

	err := configureMissionBootstrap(cmd, ag)
	if err == nil {
		t.Fatal("configureMissionBootstrap() error = nil, want decode error")
	}

	if !strings.Contains(err.Error(), "failed to decode mission file") {
		t.Fatalf("configureMissionBootstrap() error = %q, want decode failure", err)
	}
}

func TestConfigureMissionBootstrapMissionFileRequiresMissionStep(t *testing.T) {
	ag := newMissionBootstrapTestLoop()
	cmd := newMissionBootstrapTestCommand()
	missionFile := writeMissionBootstrapJobFile(t, testMissionBootstrapJob())
	if err := cmd.Flags().Set("mission-file", missionFile); err != nil {
		t.Fatalf("Flags().Set(mission-file) error = %v", err)
	}

	err := configureMissionBootstrap(cmd, ag)
	if err == nil {
		t.Fatal("configureMissionBootstrap() error = nil, want missing mission-step error")
	}

	if !strings.Contains(err.Error(), "--mission-file requires --mission-step") {
		t.Fatalf("configureMissionBootstrap() error = %q, want missing mission-step message", err)
	}
}

func TestConfigureMissionBootstrapMissionStepRequiresMissionFile(t *testing.T) {
	ag := newMissionBootstrapTestLoop()
	cmd := newMissionBootstrapTestCommand()
	if err := cmd.Flags().Set("mission-step", "build"); err != nil {
		t.Fatalf("Flags().Set(mission-step) error = %v", err)
	}

	err := configureMissionBootstrap(cmd, ag)
	if err == nil {
		t.Fatal("configureMissionBootstrap() error = nil, want missing mission-file error")
	}

	if !strings.Contains(err.Error(), "--mission-step requires --mission-file") {
		t.Fatalf("configureMissionBootstrap() error = %q, want missing mission-file message", err)
	}
}

func TestConfigureMissionBootstrapMissionStepControlFileRequiresMissionFile(t *testing.T) {
	ag := newMissionBootstrapTestLoop()
	cmd := newMissionBootstrapTestCommand()
	if err := cmd.Flags().Set("mission-step-control-file", filepath.Join(t.TempDir(), "control.json")); err != nil {
		t.Fatalf("Flags().Set(mission-step-control-file) error = %v", err)
	}

	err := configureMissionBootstrap(cmd, ag)
	if err == nil {
		t.Fatal("configureMissionBootstrap() error = nil, want missing mission-file error")
	}

	if !strings.Contains(err.Error(), "--mission-step-control-file requires --mission-file") {
		t.Fatalf("configureMissionBootstrap() error = %q, want missing mission-file message", err)
	}
}

func TestWriteMissionStatusSnapshotFromCommandDefaultPathUnchanged(t *testing.T) {
	ag := newMissionBootstrapTestLoop()
	cmd := newMissionBootstrapTestCommand()
	missionFile := writeMissionBootstrapJobFile(t, testMissionBootstrapJob())

	if err := cmd.Flags().Set("mission-file", missionFile); err != nil {
		t.Fatalf("Flags().Set(mission-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step", "build"); err != nil {
		t.Fatalf("Flags().Set(mission-step) error = %v", err)
	}
	if err := configureMissionBootstrap(cmd, ag); err != nil {
		t.Fatalf("configureMissionBootstrap() error = %v", err)
	}

	if err := writeMissionStatusSnapshotFromCommand(cmd, ag, time.Date(2026, 3, 19, 12, 0, 0, 0, time.UTC)); err != nil {
		t.Fatalf("writeMissionStatusSnapshotFromCommand() error = %v", err)
	}

	entries, err := os.ReadDir(filepath.Dir(missionFile))
	if err != nil {
		t.Fatalf("ReadDir() error = %v", err)
	}
	if len(entries) != 1 || entries[0].Name() != filepath.Base(missionFile) {
		t.Fatalf("ReadDir() = %v, want only %q", entries, filepath.Base(missionFile))
	}
}

func TestWriteMissionStatusSnapshotNoActiveMissionWritesInactiveSnapshot(t *testing.T) {
	ag := newMissionBootstrapTestLoop()
	path := filepath.Join(t.TempDir(), "status.json")
	now := time.Date(2026, 3, 19, 12, 0, 0, 123, time.UTC)

	if err := writeMissionStatusSnapshot(path, "", ag, now); err != nil {
		t.Fatalf("writeMissionStatusSnapshot() error = %v", err)
	}

	got := readMissionStatusSnapshotFile(t, path)
	if got.MissionRequired {
		t.Fatal("MissionRequired = true, want false")
	}
	if got.Active {
		t.Fatal("Active = true, want false")
	}
	if got.MissionFile != "" {
		t.Fatalf("MissionFile = %q, want empty", got.MissionFile)
	}
	if got.JobID != "" || got.StepID != "" || got.StepType != "" {
		t.Fatalf("snapshot IDs = (%q, %q, %q), want empty strings", got.JobID, got.StepID, got.StepType)
	}
	if got.RequiredAuthority != "" {
		t.Fatalf("RequiredAuthority = %q, want empty", got.RequiredAuthority)
	}
	if got.RequiresApproval {
		t.Fatal("RequiresApproval = true, want false")
	}
	if len(got.AllowedTools) != 0 {
		t.Fatalf("AllowedTools = %v, want empty", got.AllowedTools)
	}
	if got.UpdatedAt != now.Format(time.RFC3339Nano) {
		t.Fatalf("UpdatedAt = %q, want %q", got.UpdatedAt, now.Format(time.RFC3339Nano))
	}
}

func TestWriteMissionStatusSnapshotActiveMissionWritesExpectedFields(t *testing.T) {
	ag := newMissionBootstrapTestLoop()
	cmd := newMissionBootstrapTestCommand()
	job := missioncontrol.Job{
		ID:           "job-1",
		MaxAuthority: missioncontrol.AuthorityTierHigh,
		AllowedTools: []string{"read"},
		Plan: missioncontrol.Plan{
			ID: "plan-1",
			Steps: []missioncontrol.Step{
				{
					ID:                "build",
					Type:              missioncontrol.StepTypeOneShotCode,
					RequiredAuthority: missioncontrol.AuthorityTierMedium,
					RequiresApproval:  true,
					AllowedTools:      []string{"read"},
				},
				{
					ID:        "final",
					Type:      missioncontrol.StepTypeFinalResponse,
					DependsOn: []string{"build"},
				},
			},
		},
	}
	missionFile := writeMissionBootstrapJobFile(t, job)
	path := filepath.Join(t.TempDir(), "status.json")
	now := time.Date(2026, 3, 19, 12, 0, 0, 456, time.UTC)

	if err := cmd.Flags().Set("mission-required", "true"); err != nil {
		t.Fatalf("Flags().Set(mission-required) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-file", missionFile); err != nil {
		t.Fatalf("Flags().Set(mission-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step", "build"); err != nil {
		t.Fatalf("Flags().Set(mission-step) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-status-file", path); err != nil {
		t.Fatalf("Flags().Set(mission-status-file) error = %v", err)
	}

	if err := configureMissionBootstrap(cmd, ag); err != nil {
		t.Fatalf("configureMissionBootstrap() error = %v", err)
	}
	if err := writeMissionStatusSnapshotFromCommand(cmd, ag, now); err != nil {
		t.Fatalf("writeMissionStatusSnapshotFromCommand() error = %v", err)
	}

	got := readMissionStatusSnapshotFile(t, path)
	if !got.MissionRequired {
		t.Fatal("MissionRequired = false, want true")
	}
	if !got.Active {
		t.Fatal("Active = false, want true")
	}
	if got.MissionFile != missionFile {
		t.Fatalf("MissionFile = %q, want %q", got.MissionFile, missionFile)
	}
	if got.JobID != "job-1" {
		t.Fatalf("JobID = %q, want %q", got.JobID, "job-1")
	}
	if got.StepID != "build" {
		t.Fatalf("StepID = %q, want %q", got.StepID, "build")
	}
	if got.StepType != string(missioncontrol.StepTypeOneShotCode) {
		t.Fatalf("StepType = %q, want %q", got.StepType, missioncontrol.StepTypeOneShotCode)
	}
	if got.RequiredAuthority != missioncontrol.AuthorityTierMedium {
		t.Fatalf("RequiredAuthority = %q, want %q", got.RequiredAuthority, missioncontrol.AuthorityTierMedium)
	}
	if !got.RequiresApproval {
		t.Fatal("RequiresApproval = false, want true")
	}
	if got.Runtime == nil {
		t.Fatal("Runtime = nil, want non-nil")
	}
	if got.Runtime.State != missioncontrol.JobStateRunning {
		t.Fatalf("Runtime.State = %q, want %q", got.Runtime.State, missioncontrol.JobStateRunning)
	}
	if got.Runtime.ActiveStepID != "build" {
		t.Fatalf("Runtime.ActiveStepID = %q, want %q", got.Runtime.ActiveStepID, "build")
	}
}

func TestWriteMissionStatusSnapshotLeavesNoTempFileOnSuccess(t *testing.T) {
	ag := newMissionBootstrapTestLoop()
	dir := t.TempDir()
	path := filepath.Join(dir, "status.json")
	now := time.Date(2026, 3, 19, 12, 0, 0, 456, time.UTC)

	if err := writeMissionStatusSnapshot(path, "", ag, now); err != nil {
		t.Fatalf("writeMissionStatusSnapshot() error = %v", err)
	}

	got := readMissionStatusSnapshotFile(t, path)
	if got.UpdatedAt != now.Format(time.RFC3339Nano) {
		t.Fatalf("UpdatedAt = %q, want %q", got.UpdatedAt, now.Format(time.RFC3339Nano))
	}
	assertNoAtomicTempFiles(t, dir, filepath.Base(path))
}

func TestWriteMissionStatusSnapshotAllowedToolsIntersectedAndSorted(t *testing.T) {
	ag := newMissionBootstrapTestLoop()
	job := missioncontrol.Job{
		ID:           "job-1",
		AllowedTools: []string{"zeta", "alpha", "beta"},
		Plan: missioncontrol.Plan{
			ID: "plan-1",
			Steps: []missioncontrol.Step{
				{
					ID:           "build",
					Type:         missioncontrol.StepTypeOneShotCode,
					AllowedTools: []string{"zeta", "beta", "beta"},
				},
				{
					ID:        "respond",
					Type:      missioncontrol.StepTypeFinalResponse,
					DependsOn: []string{"build"},
				},
			},
		},
	}
	path := filepath.Join(t.TempDir(), "status.json")

	if err := ag.ActivateMissionStep(job, "build"); err != nil {
		t.Fatalf("ActivateMissionStep() error = %v", err)
	}
	if err := writeMissionStatusSnapshot(path, "mission.json", ag, time.Date(2026, 3, 19, 12, 0, 0, 789, time.UTC)); err != nil {
		t.Fatalf("writeMissionStatusSnapshot() error = %v", err)
	}

	got := readMissionStatusSnapshotFile(t, path)
	want := []string{"beta", "zeta"}
	if !reflect.DeepEqual(got.AllowedTools, want) {
		t.Fatalf("AllowedTools = %v, want %v", got.AllowedTools, want)
	}
}

func TestWriteMissionStatusSnapshotInvalidOutputPathReturnsError(t *testing.T) {
	ag := newMissionBootstrapTestLoop()
	dir := t.TempDir()
	path := filepath.Join(dir, "status-dir")
	if err := os.Mkdir(path, 0o755); err != nil {
		t.Fatalf("Mkdir() error = %v", err)
	}

	err := writeMissionStatusSnapshot(path, "", ag, time.Date(2026, 3, 19, 12, 0, 0, 0, time.UTC))
	if err == nil {
		t.Fatal("writeMissionStatusSnapshot() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "failed to write mission status snapshot") {
		t.Fatalf("writeMissionStatusSnapshot() error = %q, want write failure", err)
	}
	assertNoAtomicTempFiles(t, dir, filepath.Base(path))
}

func TestApplyMissionStepControlFileSwitchesActiveStep(t *testing.T) {
	ag := newMissionBootstrapTestLoop()
	cmd := newMissionBootstrapTestCommand()
	job := testMissionBootstrapJob()
	controlFile := writeMissionStepControlFile(t, missionStepControlFile{StepID: "final", UpdatedAt: "2026-03-19T12:00:00Z"})

	if err := ag.ActivateMissionStep(job, "build"); err != nil {
		t.Fatalf("ActivateMissionStep() error = %v", err)
	}

	stepID, changed, err := applyMissionStepControlFile(cmd, ag, job, controlFile)
	if err != nil {
		t.Fatalf("applyMissionStepControlFile() error = %v", err)
	}
	if !changed {
		t.Fatal("applyMissionStepControlFile() changed = false, want true")
	}
	if stepID != "final" {
		t.Fatalf("applyMissionStepControlFile() stepID = %q, want %q", stepID, "final")
	}

	ec, ok := ag.ActiveMissionStep()
	if !ok || ec.Step == nil {
		t.Fatalf("ActiveMissionStep() = %#v, want active step", ec)
	}
	if ec.Step.ID != "final" {
		t.Fatalf("ActiveMissionStep().Step.ID = %q, want %q", ec.Step.ID, "final")
	}
}

func TestApplyMissionStepControlFileInvalidStepPreservesActiveStep(t *testing.T) {
	ag := newMissionBootstrapTestLoop()
	cmd := newMissionBootstrapTestCommand()
	job := testMissionBootstrapJob()
	controlFile := writeMissionStepControlFile(t, missionStepControlFile{StepID: "missing"})

	if err := ag.ActivateMissionStep(job, "build"); err != nil {
		t.Fatalf("ActivateMissionStep() error = %v", err)
	}

	stepID, changed, err := applyMissionStepControlFile(cmd, ag, job, controlFile)
	if err == nil {
		t.Fatal("applyMissionStepControlFile() error = nil, want invalid step error")
	}
	if changed {
		t.Fatal("applyMissionStepControlFile() changed = true, want false")
	}
	if stepID != "" {
		t.Fatalf("applyMissionStepControlFile() stepID = %q, want empty", stepID)
	}

	ec, ok := ag.ActiveMissionStep()
	if !ok || ec.Step == nil {
		t.Fatalf("ActiveMissionStep() = %#v, want active step", ec)
	}
	if ec.Step.ID != "build" {
		t.Fatalf("ActiveMissionStep().Step.ID = %q, want %q", ec.Step.ID, "build")
	}
}

func TestApplyMissionStepControlFileRewritesStatusSnapshotOnSuccess(t *testing.T) {
	ag := newMissionBootstrapTestLoop()
	cmd := newMissionBootstrapTestCommand()
	job := testMissionBootstrapJob()
	missionFile := writeMissionBootstrapJobFile(t, job)
	statusFile := filepath.Join(t.TempDir(), "status.json")
	controlFile := writeMissionStepControlFile(t, missionStepControlFile{StepID: "final"})

	if err := cmd.Flags().Set("mission-file", missionFile); err != nil {
		t.Fatalf("Flags().Set(mission-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step", "build"); err != nil {
		t.Fatalf("Flags().Set(mission-step) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-status-file", statusFile); err != nil {
		t.Fatalf("Flags().Set(mission-status-file) error = %v", err)
	}

	if err := ag.ActivateMissionStep(job, "build"); err != nil {
		t.Fatalf("ActivateMissionStep() error = %v", err)
	}
	if err := writeMissionStatusSnapshotFromCommand(cmd, ag, time.Date(2026, 3, 19, 12, 0, 0, 0, time.UTC)); err != nil {
		t.Fatalf("writeMissionStatusSnapshotFromCommand() error = %v", err)
	}

	before := readMissionStatusSnapshotFile(t, statusFile)
	if before.StepID != "build" {
		t.Fatalf("initial snapshot StepID = %q, want %q", before.StepID, "build")
	}

	stepID, changed, err := applyMissionStepControlFile(cmd, ag, job, controlFile)
	if err != nil {
		t.Fatalf("applyMissionStepControlFile() error = %v", err)
	}
	if !changed {
		t.Fatal("applyMissionStepControlFile() changed = false, want true")
	}
	if stepID != "final" {
		t.Fatalf("applyMissionStepControlFile() stepID = %q, want %q", stepID, "final")
	}

	after := readMissionStatusSnapshotFile(t, statusFile)
	if after.StepID != "final" {
		t.Fatalf("rewritten snapshot StepID = %q, want %q", after.StepID, "final")
	}
	if after.UpdatedAt == before.UpdatedAt {
		t.Fatalf("rewritten snapshot UpdatedAt = %q, want changed timestamp", after.UpdatedAt)
	}
}

func TestApplyMissionStepControlFileAbsentFileIsNoOp(t *testing.T) {
	ag := newMissionBootstrapTestLoop()
	cmd := newMissionBootstrapTestCommand()
	job := testMissionBootstrapJob()

	if err := ag.ActivateMissionStep(job, "build"); err != nil {
		t.Fatalf("ActivateMissionStep() error = %v", err)
	}

	stepID, changed, err := applyMissionStepControlFile(cmd, ag, job, filepath.Join(t.TempDir(), "missing.json"))
	if err != nil {
		t.Fatalf("applyMissionStepControlFile() error = %v", err)
	}
	if changed {
		t.Fatal("applyMissionStepControlFile() changed = true, want false")
	}
	if stepID != "" {
		t.Fatalf("applyMissionStepControlFile() stepID = %q, want empty", stepID)
	}

	ec, ok := ag.ActiveMissionStep()
	if !ok || ec.Step == nil {
		t.Fatalf("ActiveMissionStep() = %#v, want active step", ec)
	}
	if ec.Step.ID != "build" {
		t.Fatalf("ActiveMissionStep().Step.ID = %q, want %q", ec.Step.ID, "build")
	}
}

func TestRestoreMissionStepControlFileOnStartupAbsentFileIsNoOp(t *testing.T) {
	ag := newMissionBootstrapTestLoop()
	cmd := newMissionBootstrapTestCommand()
	missionFile := writeMissionBootstrapJobFile(t, testMissionBootstrapJob())

	if err := cmd.Flags().Set("mission-file", missionFile); err != nil {
		t.Fatalf("Flags().Set(mission-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step", "build"); err != nil {
		t.Fatalf("Flags().Set(mission-step) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step-control-file", filepath.Join(t.TempDir(), "missing.json")); err != nil {
		t.Fatalf("Flags().Set(mission-step-control-file) error = %v", err)
	}

	job := configureMissionBootstrapJobForStartupTest(t, cmd, ag)
	baseline := restoreMissionStepControlFileOnStartup(cmd, ag, job)
	if baseline != nil {
		t.Fatalf("restoreMissionStepControlFileOnStartup() baseline = %q, want nil", string(baseline))
	}

	ec, ok := ag.ActiveMissionStep()
	if !ok || ec.Step == nil {
		t.Fatalf("ActiveMissionStep() = %#v, want active step", ec)
	}
	if ec.Step.ID != "build" {
		t.Fatalf("ActiveMissionStep().Step.ID = %q, want %q", ec.Step.ID, "build")
	}
}

func TestRestoreMissionStepControlFileOnStartupAbsentFileLeavesWatcherBaselineAsNoOp(t *testing.T) {
	ag := newMissionBootstrapTestLoop()
	cmd := newMissionBootstrapTestCommand()
	job := testMissionBootstrapJob()
	controlFile := filepath.Join(t.TempDir(), "missing.json")
	logBuf, restoreLog := captureStandardLogger(t)
	defer restoreLog()

	if err := ag.ActivateMissionStep(job, "build"); err != nil {
		t.Fatalf("ActivateMissionStep() error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step-control-file", controlFile); err != nil {
		t.Fatalf("Flags().Set(mission-step-control-file) error = %v", err)
	}

	baseline := restoreMissionStepControlFileOnStartup(cmd, ag, job)
	if baseline != nil {
		t.Fatalf("restoreMissionStepControlFileOnStartup() baseline = %q, want nil", string(baseline))
	}

	ctx, cancel := context.WithCancel(context.Background())
	go watchMissionStepControlFile(ctx, cmd, ag, job, controlFile, 5*time.Millisecond, baseline)
	time.Sleep(25 * time.Millisecond)
	cancel()

	ec, ok := ag.ActiveMissionStep()
	if !ok || ec.Step == nil {
		t.Fatalf("ActiveMissionStep() = %#v, want active step", ec)
	}
	if ec.Step.ID != "build" {
		t.Fatalf("ActiveMissionStep().Step.ID = %q, want %q", ec.Step.ID, "build")
	}
	if strings.Contains(logBuf.String(), "mission step control apply succeeded") {
		t.Fatalf("log output = %q, want no watcher apply success", logBuf.String())
	}
}

func TestRestoreMissionStepControlFileOnStartupValidFileOverridesBootstrappedStep(t *testing.T) {
	ag := newMissionBootstrapTestLoop()
	cmd := newMissionBootstrapTestCommand()
	missionFile := writeMissionBootstrapJobFile(t, testMissionBootstrapJob())
	controlFile := writeMissionStepControlFile(t, missionStepControlFile{StepID: "final", UpdatedAt: "2026-03-19T12:00:00Z"})
	logBuf, restoreLog := captureStandardLogger(t)
	defer restoreLog()

	if err := cmd.Flags().Set("mission-file", missionFile); err != nil {
		t.Fatalf("Flags().Set(mission-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step", "build"); err != nil {
		t.Fatalf("Flags().Set(mission-step) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step-control-file", controlFile); err != nil {
		t.Fatalf("Flags().Set(mission-step-control-file) error = %v", err)
	}

	job := configureMissionBootstrapJobForStartupTest(t, cmd, ag)
	baseline := restoreMissionStepControlFileOnStartup(cmd, ag, job)

	ec, ok := ag.ActiveMissionStep()
	if !ok || ec.Step == nil {
		t.Fatalf("ActiveMissionStep() = %#v, want active step", ec)
	}
	if ec.Step.ID != "final" {
		t.Fatalf("ActiveMissionStep().Step.ID = %q, want %q", ec.Step.ID, "final")
	}
	logOutput := logBuf.String()
	if !strings.Contains(logOutput, "mission step control startup apply succeeded") {
		t.Fatalf("log output = %q, want startup apply success", logOutput)
	}
	if !strings.Contains(logOutput, `job_id="job-1"`) {
		t.Fatalf("log output = %q, want job_id", logOutput)
	}
	if !strings.Contains(logOutput, `step_id="final"`) {
		t.Fatalf("log output = %q, want step_id", logOutput)
	}
	if !strings.Contains(logOutput, `control_file="`+controlFile+`"`) {
		t.Fatalf("log output = %q, want control file path", logOutput)
	}
	if !bytes.Equal(baseline, mustReadFile(t, controlFile)) {
		t.Fatalf("restoreMissionStepControlFileOnStartup() baseline = %q, want control file contents", string(baseline))
	}
}

func TestRestoreMissionStepControlFileOnStartupThenWatcherDoesNotDuplicateUnchangedApply(t *testing.T) {
	ag := newMissionBootstrapTestLoop()
	cmd := newMissionBootstrapTestCommand()
	job := testMissionBootstrapJob()
	controlFile := writeMissionStepControlFile(t, missionStepControlFile{StepID: "final", UpdatedAt: "2026-03-19T12:00:00Z"})
	logBuf, restoreLog := captureStandardLogger(t)
	defer restoreLog()

	if err := ag.ActivateMissionStep(job, "build"); err != nil {
		t.Fatalf("ActivateMissionStep() error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step-control-file", controlFile); err != nil {
		t.Fatalf("Flags().Set(mission-step-control-file) error = %v", err)
	}
	baseline := restoreMissionStepControlFileOnStartup(cmd, ag, job)
	if !bytes.Equal(baseline, mustReadFile(t, controlFile)) {
		t.Fatalf("restoreMissionStepControlFileOnStartup() baseline = %q, want control file contents", string(baseline))
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go watchMissionStepControlFile(ctx, cmd, ag, job, controlFile, 5*time.Millisecond, baseline)

	deadline := time.Now().Add(500 * time.Millisecond)
	for {
		ec, ok := ag.ActiveMissionStep()
		if ok && ec.Step != nil && ec.Step.ID == "final" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("ActiveMissionStep() did not update to final")
		}
		time.Sleep(5 * time.Millisecond)
	}

	time.Sleep(50 * time.Millisecond)
	logOutput := logBuf.String()
	if strings.Count(logOutput, "mission step control startup apply succeeded") != 1 {
		t.Fatalf("log output = %q, want one startup apply success", logOutput)
	}
	if strings.Contains(logOutput, "mission step control apply succeeded") {
		t.Fatalf("log output = %q, want no duplicate watcher apply success", logOutput)
	}
}

func TestRestoreMissionStepControlFileOnStartupInvalidFilePreservesBootstrappedStep(t *testing.T) {
	ag := newMissionBootstrapTestLoop()
	cmd := newMissionBootstrapTestCommand()
	missionFile := writeMissionBootstrapJobFile(t, testMissionBootstrapJob())
	controlFile := writeMissionStepControlFile(t, missionStepControlFile{StepID: "missing"})
	logBuf := &bytes.Buffer{}
	logWriter := log.Writer()
	logFlags := log.Flags()
	logPrefix := log.Prefix()
	log.SetOutput(logBuf)
	log.SetFlags(0)
	log.SetPrefix("")
	defer func() {
		log.SetOutput(logWriter)
		log.SetFlags(logFlags)
		log.SetPrefix(logPrefix)
	}()

	if err := cmd.Flags().Set("mission-file", missionFile); err != nil {
		t.Fatalf("Flags().Set(mission-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step", "build"); err != nil {
		t.Fatalf("Flags().Set(mission-step) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step-control-file", controlFile); err != nil {
		t.Fatalf("Flags().Set(mission-step-control-file) error = %v", err)
	}

	job := configureMissionBootstrapJobForStartupTest(t, cmd, ag)
	baseline := restoreMissionStepControlFileOnStartup(cmd, ag, job)

	ec, ok := ag.ActiveMissionStep()
	if !ok || ec.Step == nil {
		t.Fatalf("ActiveMissionStep() = %#v, want active step", ec)
	}
	if ec.Step.ID != "build" {
		t.Fatalf("ActiveMissionStep().Step.ID = %q, want %q", ec.Step.ID, "build")
	}
	if !strings.Contains(logBuf.String(), "mission step control startup apply failed") {
		t.Fatalf("log output = %q, want startup apply failure", logBuf.String())
	}
	if baseline != nil {
		t.Fatalf("restoreMissionStepControlFileOnStartup() baseline = %q, want nil", string(baseline))
	}
}

func TestRestoreMissionStepControlFileOnStartupInitialSnapshotReflectsRestoredStep(t *testing.T) {
	ag := newMissionBootstrapTestLoop()
	cmd := newMissionBootstrapTestCommand()
	job := testMissionBootstrapJob()
	missionFile := writeMissionBootstrapJobFile(t, job)
	statusFile := filepath.Join(t.TempDir(), "status.json")
	controlFile := writeMissionStepControlFile(t, missionStepControlFile{StepID: "final", UpdatedAt: "2026-03-19T12:00:00Z"})

	if err := cmd.Flags().Set("mission-file", missionFile); err != nil {
		t.Fatalf("Flags().Set(mission-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step", "build"); err != nil {
		t.Fatalf("Flags().Set(mission-step) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step-control-file", controlFile); err != nil {
		t.Fatalf("Flags().Set(mission-step-control-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-status-file", statusFile); err != nil {
		t.Fatalf("Flags().Set(mission-status-file) error = %v", err)
	}

	bootstrappedJob := configureMissionBootstrapJobForStartupTest(t, cmd, ag)
	restoreMissionStepControlFileOnStartup(cmd, ag, bootstrappedJob)
	if err := writeMissionStatusSnapshotFromCommand(cmd, ag, time.Date(2026, 3, 19, 12, 0, 0, 0, time.UTC)); err != nil {
		t.Fatalf("writeMissionStatusSnapshotFromCommand() error = %v", err)
	}

	snapshot := readMissionStatusSnapshotFile(t, statusFile)
	if snapshot.StepID != "final" {
		t.Fatalf("initial snapshot StepID = %q, want %q", snapshot.StepID, "final")
	}
	if snapshot.Runtime == nil {
		t.Fatal("initial snapshot Runtime = nil, want non-nil")
	}
	if snapshot.Runtime.ActiveStepID != "final" {
		t.Fatalf("initial snapshot Runtime.ActiveStepID = %q, want %q", snapshot.Runtime.ActiveStepID, "final")
	}
}

func TestMissionStatusBootstrapRequiresResumeApprovalAfterReboot(t *testing.T) {
	ag := newMissionBootstrapTestLoop()
	cmd := newMissionBootstrapTestCommand()
	job := testMissionBootstrapJob()
	missionFile := writeMissionBootstrapJobFile(t, job)
	statusFile := filepath.Join(t.TempDir(), "status.json")
	writeMissionStatusSnapshotFile(t, statusFile, missionStatusSnapshot{
		MissionFile: missionFile,
		JobID:       job.ID,
		StepID:      "build",
		Runtime: &missioncontrol.JobRuntimeState{
			JobID:        job.ID,
			State:        missioncontrol.JobStateRunning,
			ActiveStepID: "build",
		},
	})

	if err := cmd.Flags().Set("mission-file", missionFile); err != nil {
		t.Fatalf("Flags().Set(mission-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step", "build"); err != nil {
		t.Fatalf("Flags().Set(mission-step) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-status-file", statusFile); err != nil {
		t.Fatalf("Flags().Set(mission-status-file) error = %v", err)
	}

	err := configureMissionBootstrap(cmd, ag)
	if err == nil {
		t.Fatal("configureMissionBootstrap() error = nil, want resume approval failure")
	}
	if !strings.Contains(err.Error(), "--mission-resume-approved") {
		t.Fatalf("configureMissionBootstrap() error = %q, want resume approval message", err)
	}
}

func TestMissionStatusBootstrapApprovedResumeUsesPersistedRuntimeStep(t *testing.T) {
	ag := newMissionBootstrapTestLoop()
	cmd := newMissionBootstrapTestCommand()
	job := testMissionBootstrapJob()
	missionFile := writeMissionBootstrapJobFile(t, job)
	statusFile := filepath.Join(t.TempDir(), "status.json")
	writeMissionStatusSnapshotFile(t, statusFile, missionStatusSnapshot{
		MissionFile: missionFile,
		JobID:       job.ID,
		StepID:      "build",
		Runtime: &missioncontrol.JobRuntimeState{
			JobID:          job.ID,
			State:          missioncontrol.JobStateWaitingUser,
			ActiveStepID:   "build",
			WaitingReason:  "awaiting operator confirmation",
			CompletedSteps: []missioncontrol.RuntimeStepRecord{{StepID: "draft", At: time.Date(2026, 3, 23, 10, 0, 0, 0, time.UTC)}},
		},
	})

	if err := cmd.Flags().Set("mission-file", missionFile); err != nil {
		t.Fatalf("Flags().Set(mission-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step", "build"); err != nil {
		t.Fatalf("Flags().Set(mission-step) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-status-file", statusFile); err != nil {
		t.Fatalf("Flags().Set(mission-status-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-resume-approved", "true"); err != nil {
		t.Fatalf("Flags().Set(mission-resume-approved) error = %v", err)
	}

	if err := configureMissionBootstrap(cmd, ag); err != nil {
		t.Fatalf("configureMissionBootstrap() error = %v", err)
	}

	ec, ok := ag.ActiveMissionStep()
	if !ok {
		t.Fatal("ActiveMissionStep() ok = false, want true")
	}
	if ec.Runtime == nil {
		t.Fatal("ActiveMissionStep().Runtime = nil, want non-nil")
	}
	if ec.Runtime.State != missioncontrol.JobStateRunning {
		t.Fatalf("ActiveMissionStep().Runtime.State = %q, want %q", ec.Runtime.State, missioncontrol.JobStateRunning)
	}
	if len(ec.Runtime.CompletedSteps) != 1 || ec.Runtime.CompletedSteps[0].StepID != "draft" {
		t.Fatalf("ActiveMissionStep().Runtime.CompletedSteps = %#v, want preserved draft completion", ec.Runtime.CompletedSteps)
	}
}

func TestWatchMissionStepControlFileChangedFileAppliesOnceAfterStartup(t *testing.T) {
	ag := newMissionBootstrapTestLoop()
	cmd := newMissionBootstrapTestCommand()
	job := testMissionBootstrapJob()
	controlFile := writeMissionStepControlFile(t, missionStepControlFile{StepID: "build", UpdatedAt: "2026-03-19T12:00:00Z"})
	logBuf, restoreLog := captureStandardLogger(t)
	defer restoreLog()

	if err := ag.ActivateMissionStep(job, "build"); err != nil {
		t.Fatalf("ActivateMissionStep() error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step-control-file", controlFile); err != nil {
		t.Fatalf("Flags().Set(mission-step-control-file) error = %v", err)
	}
	baseline := restoreMissionStepControlFileOnStartup(cmd, ag, job)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go watchMissionStepControlFile(ctx, cmd, ag, job, controlFile, 5*time.Millisecond, baseline)

	time.Sleep(25 * time.Millisecond)
	writeMissionStepControlFile(t, missionStepControlFile{StepID: "final", UpdatedAt: "2026-03-19T12:00:01Z"}, controlFile)

	deadline := time.Now().Add(500 * time.Millisecond)
	for {
		ec, ok := ag.ActiveMissionStep()
		if ok && ec.Step != nil && ec.Step.ID == "final" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("ActiveMissionStep() did not update to final")
		}
		time.Sleep(5 * time.Millisecond)
	}

	deadline = time.Now().Add(500 * time.Millisecond)
	for {
		logOutput := logBuf.String()
		if strings.Count(logOutput, "mission step control apply succeeded") == 1 {
			if !strings.Contains(logOutput, `job_id="`+job.ID+`"`) {
				t.Fatalf("log output = %q, want job_id", logOutput)
			}
			if !strings.Contains(logOutput, `step_id="final"`) {
				t.Fatalf("log output = %q, want step_id", logOutput)
			}
			if !strings.Contains(logOutput, `control_file="`+controlFile+`"`) {
				t.Fatalf("log output = %q, want control file path", logOutput)
			}
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("log output = %q, want one watcher apply success", logOutput)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func TestRemoveMissionStatusSnapshotRemovesFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "status.json")
	if err := os.WriteFile(path, []byte("{}\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if err := removeMissionStatusSnapshot(path); err != nil {
		t.Fatalf("removeMissionStatusSnapshot() error = %v", err)
	}

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("Stat() error = %v, want not exists", err)
	}
}

func newMissionBootstrapTestCommand() *cobra.Command {
	cmd := &cobra.Command{Use: "test"}
	addMissionBootstrapFlags(cmd)
	cmd.Flags().String("mission-step-control-file", "", "")
	return cmd
}

func newMissionBootstrapTestLoop() *agent.AgentLoop {
	hub := chat.NewHub(10)
	provider := providers.NewStubProvider()
	return agent.NewAgentLoop(hub, provider, provider.GetDefaultModel(), 3, "", nil)
}

func configureMissionBootstrapJobForStartupTest(t *testing.T, cmd *cobra.Command, ag *agent.AgentLoop) missioncontrol.Job {
	t.Helper()

	job, err := configureMissionBootstrapJob(cmd, ag)
	if err != nil {
		t.Fatalf("configureMissionBootstrapJob() error = %v", err)
	}
	if job == nil {
		t.Fatal("configureMissionBootstrapJob() job = nil, want bootstrapped job")
	}

	return *job
}

func writeMissionBootstrapJobFile(t *testing.T, job missioncontrol.Job) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "mission.json")
	data, err := json.Marshal(job)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	return path
}

func mustReadFile(t *testing.T, path string) []byte {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", path, err)
	}
	return data
}

func writeMissionStepControlFile(t *testing.T, control missionStepControlFile, paths ...string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "control.json")
	if len(paths) > 0 {
		path = paths[0]
	}
	data, err := json.Marshal(control)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	return path
}

func writeMissionStatusSnapshotFile(t *testing.T, path string, snapshot missionStatusSnapshot) {
	t.Helper()

	data, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
}

func testMissionBootstrapJob() missioncontrol.Job {
	return missioncontrol.Job{
		ID:           "job-1",
		MaxAuthority: missioncontrol.AuthorityTierHigh,
		AllowedTools: []string{"read"},
		Plan: missioncontrol.Plan{
			ID: "plan-1",
			Steps: []missioncontrol.Step{
				{
					ID:                "build",
					Type:              missioncontrol.StepTypeOneShotCode,
					RequiredAuthority: missioncontrol.AuthorityTierLow,
					AllowedTools:      []string{"read"},
				},
				{
					ID:        "final",
					Type:      missioncontrol.StepTypeFinalResponse,
					DependsOn: []string{"build"},
				},
			},
		},
	}
}

func readMissionStatusSnapshotFile(t *testing.T, path string) missionStatusSnapshot {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	var snapshot missionStatusSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	return snapshot
}

func readMissionStepControlFile(t *testing.T, path string) missionStepControlFile {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	var control missionStepControlFile
	if err := json.Unmarshal(data, &control); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	return control
}

func assertMissionStepControlFileMissing(t *testing.T, path string) {
	t.Helper()

	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return
	}
	if err != nil {
		t.Fatalf("Stat() error = %v, want os.ErrNotExist", err)
	}
	t.Fatalf("Stat() error = nil, want os.ErrNotExist for %q", path)
}

func assertNoAtomicTempFiles(t *testing.T, dir string, targetBase string) {
	t.Helper()

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir() error = %v", err)
	}

	prefix := targetBase + ".tmp-"
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), prefix) {
			t.Fatalf("unexpected temp file %q left in %q", entry.Name(), dir)
		}
	}
}

func captureStandardLogger(t *testing.T) (*bytes.Buffer, func()) {
	t.Helper()

	logBuf := &bytes.Buffer{}
	logWriter := log.Writer()
	logFlags := log.Flags()
	logPrefix := log.Prefix()
	log.SetOutput(logBuf)
	log.SetFlags(0)
	log.SetPrefix("")

	return logBuf, func() {
		log.SetOutput(logWriter)
		log.SetFlags(logFlags)
		log.SetPrefix(logPrefix)
	}
}
