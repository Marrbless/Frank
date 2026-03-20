package main

import (
	"bytes"
	"encoding/json"
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
	missionFile := writeMissionBootstrapJobFile(t, testMissionBootstrapJob())
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
	err := writeMissionStatusSnapshot(t.TempDir(), "", ag, time.Date(2026, 3, 19, 12, 0, 0, 0, time.UTC))
	if err == nil {
		t.Fatal("writeMissionStatusSnapshot() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "failed to write mission status snapshot") {
		t.Fatalf("writeMissionStatusSnapshot() error = %q, want write failure", err)
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
	return cmd
}

func newMissionBootstrapTestLoop() *agent.AgentLoop {
	hub := chat.NewHub(10)
	provider := providers.NewStubProvider()
	return agent.NewAgentLoop(hub, provider, provider.GetDefaultModel(), 3, "", nil)
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
