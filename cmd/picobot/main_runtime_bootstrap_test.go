package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"

	"github.com/local/picobot/internal/agent"
	"github.com/local/picobot/internal/chat"
	"github.com/local/picobot/internal/missioncontrol"
	"github.com/local/picobot/internal/providers"
)

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
	if got.RuntimeSummary == nil {
		t.Fatal("RuntimeSummary = nil, want non-nil")
	}
	if got.RuntimeSummary.State != missioncontrol.JobStateRunning {
		t.Fatalf("RuntimeSummary.State = %q, want %q", got.RuntimeSummary.State, missioncontrol.JobStateRunning)
	}
	if got.RuntimeSummary.ActiveStepID != "build" {
		t.Fatalf("RuntimeSummary.ActiveStepID = %q, want %q", got.RuntimeSummary.ActiveStepID, "build")
	}
	if !reflect.DeepEqual(got.RuntimeSummary.AllowedTools, []string{"read"}) {
		t.Fatalf("RuntimeSummary.AllowedTools = %#v, want %#v", got.RuntimeSummary.AllowedTools, []string{"read"})
	}
	if got.Runtime.InspectablePlan == nil {
		t.Fatal("Runtime.InspectablePlan = nil, want non-nil")
	}
	if len(got.Runtime.InspectablePlan.Steps) != len(testMissionBootstrapJob().Plan.Steps) {
		t.Fatalf("len(Runtime.InspectablePlan.Steps) = %d, want %d", len(got.Runtime.InspectablePlan.Steps), len(testMissionBootstrapJob().Plan.Steps))
	}
	if got.RuntimeControl == nil {
		t.Fatal("RuntimeControl = nil, want non-nil")
	}
	if got.RuntimeControl.JobID != "job-1" {
		t.Fatalf("RuntimeControl.JobID = %q, want %q", got.RuntimeControl.JobID, "job-1")
	}
	if got.RuntimeControl.Step.ID != "build" {
		t.Fatalf("RuntimeControl.Step.ID = %q, want %q", got.RuntimeControl.Step.ID, "build")
	}
	if got.RuntimeControl.Step.Type != missioncontrol.StepTypeOneShotCode {
		t.Fatalf("RuntimeControl.Step.Type = %q, want %q", got.RuntimeControl.Step.Type, missioncontrol.StepTypeOneShotCode)
	}
}

func TestWriteMissionStatusSnapshotFromCommandUsesCommittedDurableProjectionWhenPresent(t *testing.T) {
	ag := newMissionBootstrapTestLoop()
	cmd := newMissionBootstrapTestCommand()
	job := testMissionBootstrapJob()
	missionFile := writeMissionBootstrapJobFile(t, job)
	statusFile := filepath.Join(t.TempDir(), "status.json")
	storeRoot := missioncontrol.ResolveStoreRoot("", statusFile)
	now := time.Date(2026, 4, 5, 19, 0, 0, 0, time.UTC)

	if err := cmd.Flags().Set("mission-file", missionFile); err != nil {
		t.Fatalf("Flags().Set(mission-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step", "build"); err != nil {
		t.Fatalf("Flags().Set(mission-step) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-status-file", statusFile); err != nil {
		t.Fatalf("Flags().Set(mission-status-file) error = %v", err)
	}

	writeCommittedMissionBootstrapJobRuntimeRecord(t, storeRoot, job.ID, 4, 1, now, missioncontrol.JobStatePaused, "final")
	writeCommittedMissionBootstrapRuntimeControlRecord(t, storeRoot, 4, 1, runtimeControlForBootstrapStep(t, job, "final"))

	liveControl := runtimeControlForBootstrapStep(t, job, "build")
	liveRuntime := missioncontrol.JobRuntimeState{
		JobID:        job.ID,
		State:        missioncontrol.JobStateRunning,
		ActiveStepID: "build",
		ActiveStepAt: now.Add(time.Minute),
	}
	if err := ag.HydrateMissionRuntimeControl(job, liveRuntime, liveControl); err != nil {
		t.Fatalf("HydrateMissionRuntimeControl() error = %v", err)
	}

	if err := writeMissionStatusSnapshotFromCommand(cmd, ag, now.Add(2*time.Minute)); err != nil {
		t.Fatalf("writeMissionStatusSnapshotFromCommand() error = %v", err)
	}

	got := readMissionStatusSnapshotFile(t, statusFile)
	if got.Active {
		t.Fatal("Active = true, want false from committed paused durable projection")
	}
	if got.JobID != job.ID {
		t.Fatalf("JobID = %q, want %q", got.JobID, job.ID)
	}
	if got.StepID != "final" {
		t.Fatalf("StepID = %q, want %q", got.StepID, "final")
	}
	if got.StepType != string(missioncontrol.StepTypeFinalResponse) {
		t.Fatalf("StepType = %q, want %q", got.StepType, missioncontrol.StepTypeFinalResponse)
	}
	assertMissionStatusSnapshotMatchesCommittedDurableState(t, got, storeRoot, job.ID, now.Add(2*time.Minute))
}

func TestStartupAndRuntimeChangeDurableProjectionUseSameSharedBuilder(t *testing.T) {
	job := missioncontrol.Job{
		ID:           "job-1",
		SpecVersion:  missioncontrol.JobSpecVersionV2,
		MaxAuthority: missioncontrol.AuthorityTierHigh,
		AllowedTools: []string{"read"},
		Plan: missioncontrol.Plan{
			ID: "plan-1",
			Steps: []missioncontrol.Step{
				{ID: "gamma", Type: missioncontrol.StepTypeOneShotCode, OneShotArtifactPath: "zeta.txt"},
				{ID: "alpha", Type: missioncontrol.StepTypeStaticArtifact, StaticArtifactPath: "alpha.json", StaticArtifactFormat: "json"},
				{ID: "beta", Type: missioncontrol.StepTypeLongRunningCode, LongRunningArtifactPath: "service.bin", LongRunningStartupCommand: []string{"go", "build", "./cmd/service"}},
				{ID: "delta", Type: missioncontrol.StepTypeStaticArtifact, StaticArtifactPath: "delta.md", StaticArtifactFormat: "markdown"},
				{ID: "epsilon", Type: missioncontrol.StepTypeOneShotCode, OneShotArtifactPath: "epsilon.go"},
				{ID: "zeta", Type: missioncontrol.StepTypeStaticArtifact, StaticArtifactPath: "zeta.yaml", StaticArtifactFormat: "yaml"},
				{ID: "final", Type: missioncontrol.StepTypeFinalResponse, DependsOn: []string{"zeta"}},
			},
		},
	}
	control, err := missioncontrol.BuildRuntimeControlContext(job, "final")
	if err != nil {
		t.Fatalf("BuildRuntimeControlContext() error = %v", err)
	}
	plan, err := missioncontrol.BuildInspectablePlanContext(job)
	if err != nil {
		t.Fatalf("BuildInspectablePlanContext() error = %v", err)
	}
	requests := make([]missioncontrol.ApprovalRequest, 0, missioncontrol.OperatorStatusApprovalHistoryLimit+2)
	for i := 0; i < missioncontrol.OperatorStatusApprovalHistoryLimit+2; i++ {
		requests = append(requests, missioncontrol.ApprovalRequest{
			JobID:           job.ID,
			StepID:          "step-" + string(rune('a'+i)),
			RequestedAction: missioncontrol.ApprovalRequestedActionStepComplete,
			Scope:           missioncontrol.ApprovalScopeMissionStep,
			State:           missioncontrol.ApprovalStatePending,
			RequestedAt:     time.Date(2026, 3, 24, 12, i, 0, 0, time.UTC),
		})
	}
	history := make([]missioncontrol.AuditEvent, 0, missioncontrol.OperatorStatusRecentAuditLimit+1)
	for i := 0; i < missioncontrol.OperatorStatusRecentAuditLimit+1; i++ {
		history = append(history, missioncontrol.AuditEvent{
			JobID:     job.ID,
			StepID:    "build",
			ToolName:  "status",
			Allowed:   true,
			Timestamp: time.Date(2026, 3, 24, 13, i, 0, 0, time.UTC),
		})
	}
	now := time.Now().UTC().Truncate(time.Second)
	requestBase := now.Add(-8 * time.Hour)
	auditBase := now.Add(-7 * time.Hour)
	pausedAt := now.Add(-6 * time.Hour)
	for i := range requests {
		requests[i].RequestedAt = requestBase.Add(time.Duration(i) * time.Minute)
	}
	for i := range history {
		history[i].Timestamp = auditBase.Add(time.Duration(i) * time.Minute)
	}
	runtime := missioncontrol.JobRuntimeState{
		JobID:            job.ID,
		State:            missioncontrol.JobStatePaused,
		ActiveStepID:     "final",
		InspectablePlan:  &plan,
		PausedReason:     missioncontrol.RuntimePauseReasonOperatorCommand,
		PausedAt:         pausedAt,
		ApprovalRequests: requests,
		AuditHistory:     history,
		CompletedSteps: []missioncontrol.RuntimeStepRecord{
			{StepID: "zeta"},
			{StepID: "gamma"},
			{StepID: "beta", ResultingState: &missioncontrol.RuntimeResultingStateRecord{Kind: string(missioncontrol.StepTypeLongRunningCode), Target: "service.bin", State: "already_present"}},
			{StepID: "alpha"},
			{StepID: "epsilon"},
			{StepID: "delta"},
		},
	}

	statusFile := filepath.Join(t.TempDir(), "status.json")
	runtimePath := filepath.Join(t.TempDir(), "runtime.json")
	storeRoot := missioncontrol.ResolveStoreRoot("", statusFile)
	if err := missioncontrol.PersistProjectedRuntimeState(storeRoot, missioncontrol.WriterLockLease{LeaseHolderID: "holder-1"}, &job, runtime, &control, now); err != nil {
		t.Fatalf("PersistProjectedRuntimeState() error = %v", err)
	}

	cmd := newMissionBootstrapTestCommand()
	missionFile := writeMissionBootstrapJobFile(t, job)
	if err := cmd.Flags().Set("mission-file", missionFile); err != nil {
		t.Fatalf("Flags().Set(mission-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step", "build"); err != nil {
		t.Fatalf("Flags().Set(mission-step) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-status-file", statusFile); err != nil {
		t.Fatalf("Flags().Set(mission-status-file) error = %v", err)
	}

	ag := newMissionBootstrapTestLoop()
	liveControl := runtimeControlForBootstrapStep(t, testMissionBootstrapJob(), "build")
	liveRuntime := missioncontrol.JobRuntimeState{
		JobID:        job.ID,
		State:        missioncontrol.JobStateRunning,
		ActiveStepID: "build",
		ActiveStepAt: now.Add(time.Minute),
	}
	if err := ag.HydrateMissionRuntimeControl(testMissionBootstrapJob(), liveRuntime, liveControl); err != nil {
		t.Fatalf("HydrateMissionRuntimeControl() error = %v", err)
	}

	if err := writeMissionStatusSnapshotFromCommand(cmd, ag, now); err != nil {
		t.Fatalf("writeMissionStatusSnapshotFromCommand() error = %v", err)
	}
	if err := writeProjectedMissionStatusSnapshot(runtimePath, missionFile, storeRoot, false, job.ID, now); err != nil {
		t.Fatalf("writeProjectedMissionStatusSnapshot() error = %v", err)
	}

	if !bytes.Equal(mustReadFile(t, statusFile), mustReadFile(t, runtimePath)) {
		t.Fatalf("durable startup/runtime projection bytes differ:\nstartup=%s\nruntime=%s", string(mustReadFile(t, statusFile)), string(mustReadFile(t, runtimePath)))
	}
}

func TestWriteMissionStatusSnapshotFromCommandFallsBackToLiveWhenDurableStoreEmptyForJob(t *testing.T) {
	ag := newMissionBootstrapTestLoop()
	cmd := newMissionBootstrapTestCommand()
	job := testMissionBootstrapJob()
	missionFile := writeMissionBootstrapJobFile(t, job)
	statusFile := filepath.Join(t.TempDir(), "status.json")
	now := time.Date(2026, 4, 5, 19, 45, 0, 0, time.UTC)

	if err := cmd.Flags().Set("mission-file", missionFile); err != nil {
		t.Fatalf("Flags().Set(mission-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step", "build"); err != nil {
		t.Fatalf("Flags().Set(mission-step) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-status-file", statusFile); err != nil {
		t.Fatalf("Flags().Set(mission-status-file) error = %v", err)
	}

	control := runtimeControlForBootstrapStep(t, job, "build")
	runtime := missioncontrol.JobRuntimeState{
		JobID:        job.ID,
		State:        missioncontrol.JobStatePaused,
		ActiveStepID: "build",
		PausedReason: missioncontrol.RuntimePauseReasonOperatorCommand,
		PausedAt:     now.Add(-time.Minute),
	}
	if err := ag.HydrateMissionRuntimeControl(job, runtime, control); err != nil {
		t.Fatalf("HydrateMissionRuntimeControl() error = %v", err)
	}

	if err := writeMissionStatusSnapshotFromCommand(cmd, ag, now); err != nil {
		t.Fatalf("writeMissionStatusSnapshotFromCommand() error = %v", err)
	}

	got := readMissionStatusSnapshotFile(t, statusFile)
	if got.StepID != "build" {
		t.Fatalf("StepID = %q, want %q", got.StepID, "build")
	}
	if got.StepType != "" {
		t.Fatalf("StepType = %q, want empty live fallback step metadata", got.StepType)
	}
	if got.Runtime == nil || got.Runtime.State != missioncontrol.JobStatePaused {
		t.Fatalf("Runtime = %#v, want live paused runtime", got.Runtime)
	}
	if got.RuntimeControl == nil || got.RuntimeControl.Step.ID != "build" {
		t.Fatalf("RuntimeControl = %#v, want live build control", got.RuntimeControl)
	}
}

func TestWriteMissionStatusSnapshotIncludesRuntimeSummaryTruncationForPersistedRuntime(t *testing.T) {
	ag := newMissionBootstrapTestLoop()
	job := missioncontrol.Job{
		ID:           "job-1",
		SpecVersion:  missioncontrol.JobSpecVersionV2,
		MaxAuthority: missioncontrol.AuthorityTierHigh,
		AllowedTools: []string{"read"},
		Plan: missioncontrol.Plan{
			ID: "plan-1",
			Steps: []missioncontrol.Step{
				{ID: "gamma", Type: missioncontrol.StepTypeOneShotCode, OneShotArtifactPath: "zeta.txt"},
				{ID: "alpha", Type: missioncontrol.StepTypeStaticArtifact, StaticArtifactPath: "alpha.json", StaticArtifactFormat: "json"},
				{ID: "beta", Type: missioncontrol.StepTypeLongRunningCode, LongRunningArtifactPath: "service.bin", LongRunningStartupCommand: []string{"go", "build", "./cmd/service"}},
				{ID: "delta", Type: missioncontrol.StepTypeStaticArtifact, StaticArtifactPath: "delta.md", StaticArtifactFormat: "markdown"},
				{ID: "epsilon", Type: missioncontrol.StepTypeOneShotCode, OneShotArtifactPath: "epsilon.go"},
				{ID: "zeta", Type: missioncontrol.StepTypeStaticArtifact, StaticArtifactPath: "zeta.yaml", StaticArtifactFormat: "yaml"},
				{ID: "final", Type: missioncontrol.StepTypeFinalResponse, DependsOn: []string{"zeta"}},
			},
		},
	}
	control, err := missioncontrol.BuildRuntimeControlContext(job, "final")
	if err != nil {
		t.Fatalf("BuildRuntimeControlContext() error = %v", err)
	}
	plan, err := missioncontrol.BuildInspectablePlanContext(job)
	if err != nil {
		t.Fatalf("BuildInspectablePlanContext() error = %v", err)
	}
	requests := make([]missioncontrol.ApprovalRequest, 0, missioncontrol.OperatorStatusApprovalHistoryLimit+2)
	for i := 0; i < missioncontrol.OperatorStatusApprovalHistoryLimit+2; i++ {
		requests = append(requests, missioncontrol.ApprovalRequest{
			JobID:           job.ID,
			StepID:          "step-" + string(rune('a'+i)),
			RequestedAction: missioncontrol.ApprovalRequestedActionStepComplete,
			Scope:           missioncontrol.ApprovalScopeMissionStep,
			State:           missioncontrol.ApprovalStatePending,
			RequestedAt:     time.Date(2026, 3, 24, 12, i, 0, 0, time.UTC),
		})
	}
	history := make([]missioncontrol.AuditEvent, 0, missioncontrol.OperatorStatusRecentAuditLimit+1)
	for i := 0; i < missioncontrol.OperatorStatusRecentAuditLimit+1; i++ {
		history = append(history, missioncontrol.AuditEvent{
			JobID:     job.ID,
			StepID:    "build",
			ToolName:  "status",
			Allowed:   true,
			Timestamp: time.Date(2026, 3, 24, 13, i, 0, 0, time.UTC),
		})
	}
	runtime := missioncontrol.JobRuntimeState{
		JobID:            job.ID,
		State:            missioncontrol.JobStatePaused,
		ActiveStepID:     "final",
		InspectablePlan:  &plan,
		PausedReason:     missioncontrol.RuntimePauseReasonOperatorCommand,
		PausedAt:         time.Date(2026, 3, 24, 13, 30, 0, 0, time.UTC),
		ApprovalRequests: requests,
		AuditHistory:     history,
		CompletedSteps: []missioncontrol.RuntimeStepRecord{
			{StepID: "zeta"},
			{StepID: "gamma"},
			{StepID: "beta", ResultingState: &missioncontrol.RuntimeResultingStateRecord{Kind: string(missioncontrol.StepTypeLongRunningCode), Target: "service.bin", State: "already_present"}},
			{StepID: "alpha"},
			{StepID: "epsilon"},
			{StepID: "delta"},
		},
	}
	if err := ag.HydrateMissionRuntimeControl(job, runtime, &control); err != nil {
		t.Fatalf("HydrateMissionRuntimeControl() error = %v", err)
	}

	path := filepath.Join(t.TempDir(), "status.json")
	if err := writeMissionStatusSnapshot(path, "mission.json", ag, time.Date(2026, 3, 27, 12, 0, 0, 0, time.UTC)); err != nil {
		t.Fatalf("writeMissionStatusSnapshot() error = %v", err)
	}

	got := readMissionStatusSnapshotFile(t, path)
	if got.RuntimeSummary == nil {
		t.Fatal("RuntimeSummary = nil, want persisted runtime summary")
	}
	if got.Active {
		t.Fatal("Active = true, want false for persisted-only runtime snapshot")
	}
	if got.RuntimeSummary.State != missioncontrol.JobStatePaused {
		t.Fatalf("RuntimeSummary.State = %q, want %q", got.RuntimeSummary.State, missioncontrol.JobStatePaused)
	}
	if got.RuntimeSummary.PausedReason != missioncontrol.RuntimePauseReasonOperatorCommand {
		t.Fatalf("RuntimeSummary.PausedReason = %q, want %q", got.RuntimeSummary.PausedReason, missioncontrol.RuntimePauseReasonOperatorCommand)
	}
	if got.RuntimeSummary.PausedAt == nil || *got.RuntimeSummary.PausedAt != "2026-03-24T13:30:00Z" {
		t.Fatalf("RuntimeSummary.PausedAt = %#v, want RFC3339 pause time", got.RuntimeSummary.PausedAt)
	}
	if !reflect.DeepEqual(got.RuntimeSummary.AllowedTools, []string{"read"}) {
		t.Fatalf("RuntimeSummary.AllowedTools = %#v, want %#v", got.RuntimeSummary.AllowedTools, []string{"read"})
	}
	if len(got.RuntimeSummary.Artifacts) != missioncontrol.OperatorStatusArtifactLimit {
		t.Fatalf("RuntimeSummary.Artifacts = %#v, want %d deterministic entries", got.RuntimeSummary.Artifacts, missioncontrol.OperatorStatusArtifactLimit)
	}
	if got.RuntimeSummary.Artifacts[0].StepID != "gamma" || got.RuntimeSummary.Artifacts[0].Path != "zeta.txt" {
		t.Fatalf("RuntimeSummary.Artifacts[0] = %#v, want step_id=%q path=%q", got.RuntimeSummary.Artifacts[0], "gamma", "zeta.txt")
	}
	if got.RuntimeSummary.Artifacts[2].StepID != "beta" || got.RuntimeSummary.Artifacts[2].State != "already_present" {
		t.Fatalf("RuntimeSummary.Artifacts[2] = %#v, want step_id=%q state=%q", got.RuntimeSummary.Artifacts[2], "beta", "already_present")
	}
	if got.RuntimeSummary.Truncation == nil {
		t.Fatal("RuntimeSummary.Truncation = nil, want truncation metadata")
	}
	if got.RuntimeSummary.Truncation.ApprovalHistoryOmitted != 2 {
		t.Fatalf("RuntimeSummary.Truncation.ApprovalHistoryOmitted = %d, want 2", got.RuntimeSummary.Truncation.ApprovalHistoryOmitted)
	}
	if got.RuntimeSummary.Truncation.RecentAuditOmitted != 1 {
		t.Fatalf("RuntimeSummary.Truncation.RecentAuditOmitted = %d, want 1", got.RuntimeSummary.Truncation.RecentAuditOmitted)
	}
	if got.RuntimeSummary.Truncation.ArtifactsOmitted != 1 {
		t.Fatalf("RuntimeSummary.Truncation.ArtifactsOmitted = %d, want 1", got.RuntimeSummary.Truncation.ArtifactsOmitted)
	}
}

func TestWriteProjectedMissionStatusSnapshotIncludesCommittedRuntimeSummaryTruncation(t *testing.T) {
	job := missioncontrol.Job{
		ID:           "job-1",
		SpecVersion:  missioncontrol.JobSpecVersionV2,
		MaxAuthority: missioncontrol.AuthorityTierHigh,
		AllowedTools: []string{"read"},
		Plan: missioncontrol.Plan{
			ID: "plan-1",
			Steps: []missioncontrol.Step{
				{ID: "gamma", Type: missioncontrol.StepTypeOneShotCode, OneShotArtifactPath: "zeta.txt"},
				{ID: "alpha", Type: missioncontrol.StepTypeStaticArtifact, StaticArtifactPath: "alpha.json", StaticArtifactFormat: "json"},
				{ID: "beta", Type: missioncontrol.StepTypeLongRunningCode, LongRunningArtifactPath: "service.bin", LongRunningStartupCommand: []string{"go", "build", "./cmd/service"}},
				{ID: "delta", Type: missioncontrol.StepTypeStaticArtifact, StaticArtifactPath: "delta.md", StaticArtifactFormat: "markdown"},
				{ID: "epsilon", Type: missioncontrol.StepTypeOneShotCode, OneShotArtifactPath: "epsilon.go"},
				{ID: "zeta", Type: missioncontrol.StepTypeStaticArtifact, StaticArtifactPath: "zeta.yaml", StaticArtifactFormat: "yaml"},
				{ID: "final", Type: missioncontrol.StepTypeFinalResponse, DependsOn: []string{"zeta"}},
			},
		},
	}
	control, err := missioncontrol.BuildRuntimeControlContext(job, "final")
	if err != nil {
		t.Fatalf("BuildRuntimeControlContext() error = %v", err)
	}
	plan, err := missioncontrol.BuildInspectablePlanContext(job)
	if err != nil {
		t.Fatalf("BuildInspectablePlanContext() error = %v", err)
	}
	requests := make([]missioncontrol.ApprovalRequest, 0, missioncontrol.OperatorStatusApprovalHistoryLimit+2)
	for i := 0; i < missioncontrol.OperatorStatusApprovalHistoryLimit+2; i++ {
		requests = append(requests, missioncontrol.ApprovalRequest{
			JobID:           job.ID,
			StepID:          "step-" + string(rune('a'+i)),
			RequestedAction: missioncontrol.ApprovalRequestedActionStepComplete,
			Scope:           missioncontrol.ApprovalScopeMissionStep,
			State:           missioncontrol.ApprovalStatePending,
			RequestedAt:     time.Date(2026, 3, 24, 12, i, 0, 0, time.UTC),
		})
	}
	history := make([]missioncontrol.AuditEvent, 0, missioncontrol.OperatorStatusRecentAuditLimit+1)
	for i := 0; i < missioncontrol.OperatorStatusRecentAuditLimit+1; i++ {
		history = append(history, missioncontrol.AuditEvent{
			JobID:     job.ID,
			StepID:    "build",
			ToolName:  "status",
			Allowed:   true,
			Timestamp: time.Date(2026, 3, 24, 13, i, 0, 0, time.UTC),
		})
	}
	now := time.Now().UTC().Truncate(time.Second)
	requestBase := now.Add(-8 * time.Hour)
	auditBase := now.Add(-7 * time.Hour)
	pausedAt := now.Add(-6 * time.Hour)
	for i := range requests {
		requests[i].RequestedAt = requestBase.Add(time.Duration(i) * time.Minute)
	}
	for i := range history {
		history[i].Timestamp = auditBase.Add(time.Duration(i) * time.Minute)
	}
	runtime := missioncontrol.JobRuntimeState{
		JobID:            job.ID,
		State:            missioncontrol.JobStatePaused,
		ActiveStepID:     "final",
		InspectablePlan:  &plan,
		PausedReason:     missioncontrol.RuntimePauseReasonOperatorCommand,
		PausedAt:         pausedAt,
		ApprovalRequests: requests,
		AuditHistory:     history,
		CompletedSteps: []missioncontrol.RuntimeStepRecord{
			{StepID: "zeta"},
			{StepID: "gamma"},
			{StepID: "beta", ResultingState: &missioncontrol.RuntimeResultingStateRecord{Kind: string(missioncontrol.StepTypeLongRunningCode), Target: "service.bin", State: "already_present"}},
			{StepID: "alpha"},
			{StepID: "epsilon"},
			{StepID: "delta"},
		},
	}
	storeRoot := filepath.Join(t.TempDir(), "status.store")
	if err := missioncontrol.PersistProjectedRuntimeState(storeRoot, missioncontrol.WriterLockLease{LeaseHolderID: "holder-1"}, &job, runtime, &control, now); err != nil {
		t.Fatalf("PersistProjectedRuntimeState() error = %v", err)
	}

	path := filepath.Join(t.TempDir(), "status.json")
	if err := writeProjectedMissionStatusSnapshot(path, "mission.json", storeRoot, false, job.ID, now); err != nil {
		t.Fatalf("writeProjectedMissionStatusSnapshot() error = %v", err)
	}

	got := readMissionStatusSnapshotFile(t, path)
	if got.RuntimeSummary == nil {
		t.Fatal("RuntimeSummary = nil, want persisted runtime summary")
	}
	if got.RuntimeSummary.State != missioncontrol.JobStatePaused {
		t.Fatalf("RuntimeSummary.State = %q, want %q", got.RuntimeSummary.State, missioncontrol.JobStatePaused)
	}
	if got.RuntimeSummary.PausedReason != missioncontrol.RuntimePauseReasonOperatorCommand {
		t.Fatalf("RuntimeSummary.PausedReason = %q, want %q", got.RuntimeSummary.PausedReason, missioncontrol.RuntimePauseReasonOperatorCommand)
	}
	if got.RuntimeSummary.PausedAt == nil || *got.RuntimeSummary.PausedAt != pausedAt.Format(time.RFC3339) {
		t.Fatalf("RuntimeSummary.PausedAt = %#v, want RFC3339 pause time", got.RuntimeSummary.PausedAt)
	}
	if !reflect.DeepEqual(got.AllowedTools, []string{"read"}) {
		t.Fatalf("AllowedTools = %#v, want %#v", got.AllowedTools, []string{"read"})
	}
	if !reflect.DeepEqual(got.RuntimeSummary.AllowedTools, []string{"read"}) {
		t.Fatalf("RuntimeSummary.AllowedTools = %#v, want %#v", got.RuntimeSummary.AllowedTools, []string{"read"})
	}
	if len(got.RuntimeSummary.Artifacts) != missioncontrol.OperatorStatusArtifactLimit {
		t.Fatalf("RuntimeSummary.Artifacts = %#v, want %d deterministic entries", got.RuntimeSummary.Artifacts, missioncontrol.OperatorStatusArtifactLimit)
	}
	if got.RuntimeSummary.Artifacts[0].StepID != "gamma" || got.RuntimeSummary.Artifacts[0].Path != "zeta.txt" {
		t.Fatalf("RuntimeSummary.Artifacts[0] = %#v, want step_id=%q path=%q", got.RuntimeSummary.Artifacts[0], "gamma", "zeta.txt")
	}
	if got.RuntimeSummary.Artifacts[2].StepID != "beta" || got.RuntimeSummary.Artifacts[2].State != "already_present" {
		t.Fatalf("RuntimeSummary.Artifacts[2] = %#v, want step_id=%q state=%q", got.RuntimeSummary.Artifacts[2], "beta", "already_present")
	}
	if got.RuntimeSummary.Truncation == nil {
		t.Fatal("RuntimeSummary.Truncation = nil, want truncation metadata")
	}
	if got.RuntimeSummary.Truncation.ApprovalHistoryOmitted != 2 {
		t.Fatalf("RuntimeSummary.Truncation.ApprovalHistoryOmitted = %d, want 2", got.RuntimeSummary.Truncation.ApprovalHistoryOmitted)
	}
	if got.RuntimeSummary.Truncation.RecentAuditOmitted != 1 {
		t.Fatalf("RuntimeSummary.Truncation.RecentAuditOmitted = %d, want 1", got.RuntimeSummary.Truncation.RecentAuditOmitted)
	}
	if got.RuntimeSummary.Truncation.ArtifactsOmitted != 1 {
		t.Fatalf("RuntimeSummary.Truncation.ArtifactsOmitted = %d, want 1", got.RuntimeSummary.Truncation.ArtifactsOmitted)
	}
	assertMissionStatusSnapshotMatchesCommittedDurableState(t, got, storeRoot, job.ID, now)
}

func TestMissionStatusSnapshotWritePersistsAuditHistory(t *testing.T) {
	statusFile := filepath.Join(t.TempDir(), "status.json")
	cmd := newMissionBootstrapTestCommand()
	if err := cmd.Flags().Set("mission-status-file", statusFile); err != nil {
		t.Fatalf("Flags().Set(mission-status-file) error = %v", err)
	}

	hub := chat.NewHub(10)
	provider := &missionStatusFixedResponseProvider{content: "unused"}
	ag := agent.NewAgentLoop(hub, provider, provider.GetDefaultModel(), 3, "", nil)
	installMissionRuntimeChangeHook(cmd, ag)

	if err := ag.ActivateMissionStep(testMissionBootstrapJob(), "build"); err != nil {
		t.Fatalf("ActivateMissionStep() error = %v", err)
	}
	if _, err := ag.ProcessDirect("PAUSE job-1", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(PAUSE) error = %v", err)
	}

	got := readMissionStatusSnapshotFile(t, statusFile)
	if got.Runtime == nil {
		t.Fatal("Runtime = nil, want non-nil")
	}
	if got.Runtime.State != missioncontrol.JobStatePaused {
		t.Fatalf("Runtime.State = %q, want %q", got.Runtime.State, missioncontrol.JobStatePaused)
	}
	if len(got.Runtime.AuditHistory) != 2 {
		t.Fatalf("Runtime.AuditHistory count = %d, want 2", len(got.Runtime.AuditHistory))
	}
	expectedAudit := missioncontrol.AppendAuditHistory(nil, missioncontrol.AuditEvent{
		JobID:       "job-1",
		StepID:      "build",
		ToolName:    "pause",
		ActionClass: missioncontrol.AuditActionClassOperatorCommand,
		Result:      missioncontrol.AuditResultApplied,
		Allowed:     true,
		Timestamp:   got.Runtime.AuditHistory[0].Timestamp,
	})
	expectedAudit = missioncontrol.AppendAuditHistory(expectedAudit, missioncontrol.AuditEvent{
		JobID:       "job-1",
		StepID:      "build",
		ToolName:    "pause_ack",
		ActionClass: missioncontrol.AuditActionClassRuntime,
		Result:      missioncontrol.AuditResultApplied,
		Allowed:     true,
		Timestamp:   got.Runtime.AuditHistory[1].Timestamp,
	})
	if !reflect.DeepEqual(got.Runtime.AuditHistory, expectedAudit) {
		t.Fatalf("Runtime.AuditHistory = %#v, want persisted pause and pause_ack audits", got.Runtime.AuditHistory)
	}
}

func TestMissionStatusBootstrapRehydratesAuditHistoryWithoutDuplication(t *testing.T) {
	statusFile := filepath.Join(t.TempDir(), "status.json")
	cmd := newMissionBootstrapTestCommand()
	if err := cmd.Flags().Set("mission-status-file", statusFile); err != nil {
		t.Fatalf("Flags().Set(mission-status-file) error = %v", err)
	}

	job := testMissionBootstrapJob()
	missionFile := writeMissionBootstrapJobFile(t, job)
	persistedAudit := missioncontrol.AppendAuditHistory(nil, missioncontrol.AuditEvent{
		JobID:     job.ID,
		StepID:    "build",
		ToolName:  "pause",
		Allowed:   true,
		Timestamp: time.Date(2026, 3, 24, 12, 0, 0, 0, time.UTC),
	})[0]
	writeMissionStatusSnapshotFile(t, statusFile, missionStatusSnapshot{
		MissionFile: missionFile,
		JobID:       job.ID,
		StepID:      "build",
		Runtime: &missioncontrol.JobRuntimeState{
			JobID:        job.ID,
			State:        missioncontrol.JobStatePaused,
			ActiveStepID: "build",
			PausedReason: missioncontrol.RuntimePauseReasonOperatorCommand,
			AuditHistory: []missioncontrol.AuditEvent{persistedAudit},
		},
		RuntimeControl: runtimeControlForBootstrapStep(t, job, "build"),
	})
	if err := cmd.Flags().Set("mission-file", missionFile); err != nil {
		t.Fatalf("Flags().Set(mission-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step", "build"); err != nil {
		t.Fatalf("Flags().Set(mission-step) error = %v", err)
	}

	ag := newMissionBootstrapTestLoop()
	installMissionRuntimeChangeHook(cmd, ag)

	if err := configureMissionBootstrap(cmd, ag); err != nil {
		t.Fatalf("configureMissionBootstrap() error = %v", err)
	}

	runtime, ok := ag.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if !reflect.DeepEqual(runtime.AuditHistory, []missioncontrol.AuditEvent{persistedAudit}) {
		t.Fatalf("MissionRuntimeState().AuditHistory = %#v, want persisted history %#v", runtime.AuditHistory, []missioncontrol.AuditEvent{persistedAudit})
	}

	now := time.Date(2026, 3, 24, 12, 1, 0, 0, time.UTC)
	if err := writeMissionStatusSnapshotFromCommand(cmd, ag, now); err != nil {
		t.Fatalf("writeMissionStatusSnapshotFromCommand() error = %v", err)
	}

	snapshot := readMissionStatusSnapshotFile(t, statusFile)
	if snapshot.Runtime == nil {
		t.Fatal("snapshot.Runtime = nil, want non-nil")
	}
	if !reflect.DeepEqual(snapshot.Runtime.AuditHistory, []missioncontrol.AuditEvent{persistedAudit}) {
		t.Fatalf("snapshot.Runtime.AuditHistory = %#v, want persisted history %#v", snapshot.Runtime.AuditHistory, []missioncontrol.AuditEvent{persistedAudit})
	}
}

func TestMissionStatusRuntimeChangeHookPersistsApprovalLifecycle(t *testing.T) {
	statusFile := filepath.Join(t.TempDir(), "status.json")
	cmd := newMissionBootstrapTestCommand()
	if err := cmd.Flags().Set("mission-status-file", statusFile); err != nil {
		t.Fatalf("Flags().Set(mission-status-file) error = %v", err)
	}

	hub := chat.NewHub(10)
	provider := &missionStatusFixedResponseProvider{content: "Need approval before continuing."}
	ag := agent.NewAgentLoop(hub, provider, provider.GetDefaultModel(), 3, "", nil)
	installMissionRuntimeChangeHook(cmd, ag)

	job := missioncontrol.Job{
		ID:           "job-1",
		MaxAuthority: missioncontrol.AuthorityTierHigh,
		AllowedTools: []string{"read"},
		Plan: missioncontrol.Plan{
			ID: "plan-1",
			Steps: []missioncontrol.Step{
				{
					ID:      "build",
					Type:    missioncontrol.StepTypeDiscussion,
					Subtype: missioncontrol.StepSubtypeAuthorization,
				},
				{
					ID:        "final",
					Type:      missioncontrol.StepTypeFinalResponse,
					DependsOn: []string{"build"},
				},
			},
		},
	}

	if err := ag.ActivateMissionStep(job, "build"); err != nil {
		t.Fatalf("ActivateMissionStep() error = %v", err)
	}
	if _, err := ag.ProcessDirect("continue", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(continue) error = %v", err)
	}

	waitingSnapshot := readMissionStatusSnapshotFile(t, statusFile)
	if waitingSnapshot.Runtime == nil {
		t.Fatal("waitingSnapshot.Runtime = nil, want non-nil")
	}
	if waitingSnapshot.Runtime.State != missioncontrol.JobStateWaitingUser {
		t.Fatalf("waitingSnapshot.Runtime.State = %q, want %q", waitingSnapshot.Runtime.State, missioncontrol.JobStateWaitingUser)
	}
	if len(waitingSnapshot.Runtime.ApprovalRequests) != 1 || waitingSnapshot.Runtime.ApprovalRequests[0].State != missioncontrol.ApprovalStatePending {
		t.Fatalf("waitingSnapshot.Runtime.ApprovalRequests = %#v, want one pending approval", waitingSnapshot.Runtime.ApprovalRequests)
	}

	if _, err := ag.ProcessDirect("APPROVE job-1 build", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(APPROVE) error = %v", err)
	}

	approvedSnapshot := readMissionStatusSnapshotFile(t, statusFile)
	if approvedSnapshot.Runtime == nil {
		t.Fatal("approvedSnapshot.Runtime = nil, want non-nil")
	}
	if approvedSnapshot.Runtime.State != missioncontrol.JobStatePaused {
		t.Fatalf("approvedSnapshot.Runtime.State = %q, want %q", approvedSnapshot.Runtime.State, missioncontrol.JobStatePaused)
	}
	if len(approvedSnapshot.Runtime.CompletedSteps) != 1 || approvedSnapshot.Runtime.CompletedSteps[0].StepID != "build" {
		t.Fatalf("approvedSnapshot.Runtime.CompletedSteps = %#v, want build completion", approvedSnapshot.Runtime.CompletedSteps)
	}
	if len(approvedSnapshot.Runtime.ApprovalGrants) != 1 || approvedSnapshot.Runtime.ApprovalGrants[0].State != missioncontrol.ApprovalStateGranted {
		t.Fatalf("approvedSnapshot.Runtime.ApprovalGrants = %#v, want one granted approval", approvedSnapshot.Runtime.ApprovalGrants)
	}
	if approvedSnapshot.Runtime.ApprovalRequests[0].SessionChannel != "cli" || approvedSnapshot.Runtime.ApprovalRequests[0].SessionChatID != "direct" {
		t.Fatalf("approvedSnapshot.Runtime.ApprovalRequests[0] session = (%q, %q), want (%q, %q)", approvedSnapshot.Runtime.ApprovalRequests[0].SessionChannel, approvedSnapshot.Runtime.ApprovalRequests[0].SessionChatID, "cli", "direct")
	}
	if approvedSnapshot.Runtime.ApprovalGrants[0].SessionChannel != "cli" || approvedSnapshot.Runtime.ApprovalGrants[0].SessionChatID != "direct" {
		t.Fatalf("approvedSnapshot.Runtime.ApprovalGrants[0] session = (%q, %q), want (%q, %q)", approvedSnapshot.Runtime.ApprovalGrants[0].SessionChannel, approvedSnapshot.Runtime.ApprovalGrants[0].SessionChatID, "cli", "direct")
	}
}

func TestMissionStatusRuntimeChangeHookPersistsNaturalApprovalLifecycle(t *testing.T) {
	statusFile := filepath.Join(t.TempDir(), "status.json")
	cmd := newMissionBootstrapTestCommand()
	if err := cmd.Flags().Set("mission-status-file", statusFile); err != nil {
		t.Fatalf("Flags().Set(mission-status-file) error = %v", err)
	}

	hub := chat.NewHub(10)
	provider := &missionStatusFixedResponseProvider{content: "Need approval before continuing."}
	ag := agent.NewAgentLoop(hub, provider, provider.GetDefaultModel(), 3, "", nil)
	installMissionRuntimeChangeHook(cmd, ag)

	job := missioncontrol.Job{
		ID:           "job-1",
		MaxAuthority: missioncontrol.AuthorityTierHigh,
		AllowedTools: []string{"read"},
		Plan: missioncontrol.Plan{
			ID: "plan-1",
			Steps: []missioncontrol.Step{
				{
					ID:      "build",
					Type:    missioncontrol.StepTypeDiscussion,
					Subtype: missioncontrol.StepSubtypeAuthorization,
				},
				{
					ID:        "final",
					Type:      missioncontrol.StepTypeFinalResponse,
					DependsOn: []string{"build"},
				},
			},
		},
	}

	if err := ag.ActivateMissionStep(job, "build"); err != nil {
		t.Fatalf("ActivateMissionStep() error = %v", err)
	}
	if _, err := ag.ProcessDirect("continue", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(continue) error = %v", err)
	}
	if _, err := ag.ProcessDirect("yes", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(yes) error = %v", err)
	}

	snapshot := readMissionStatusSnapshotFile(t, statusFile)
	if snapshot.Runtime == nil {
		t.Fatal("snapshot.Runtime = nil, want non-nil")
	}
	if snapshot.Runtime.State != missioncontrol.JobStatePaused {
		t.Fatalf("snapshot.Runtime.State = %q, want %q", snapshot.Runtime.State, missioncontrol.JobStatePaused)
	}
	if len(snapshot.Runtime.ApprovalGrants) != 1 || snapshot.Runtime.ApprovalGrants[0].GrantedVia != missioncontrol.ApprovalGrantedViaOperatorReply {
		t.Fatalf("snapshot.Runtime.ApprovalGrants = %#v, want one natural-language approval grant", snapshot.Runtime.ApprovalGrants)
	}
	if snapshot.Runtime.ApprovalRequests[0].SessionChannel != "cli" || snapshot.Runtime.ApprovalRequests[0].SessionChatID != "direct" {
		t.Fatalf("snapshot.Runtime.ApprovalRequests[0] session = (%q, %q), want (%q, %q)", snapshot.Runtime.ApprovalRequests[0].SessionChannel, snapshot.Runtime.ApprovalRequests[0].SessionChatID, "cli", "direct")
	}
	if snapshot.Runtime.ApprovalGrants[0].SessionChannel != "cli" || snapshot.Runtime.ApprovalGrants[0].SessionChatID != "direct" {
		t.Fatalf("snapshot.Runtime.ApprovalGrants[0] session = (%q, %q), want (%q, %q)", snapshot.Runtime.ApprovalGrants[0].SessionChannel, snapshot.Runtime.ApprovalGrants[0].SessionChatID, "cli", "direct")
	}
}

func TestMissionStatusRuntimeChangeHookPersistsRehydratedApprovalLifecycle(t *testing.T) {
	statusFile := filepath.Join(t.TempDir(), "status.json")
	cmd := newMissionBootstrapTestCommand()
	if err := cmd.Flags().Set("mission-status-file", statusFile); err != nil {
		t.Fatalf("Flags().Set(mission-status-file) error = %v", err)
	}

	job := missioncontrol.Job{
		ID:           "job-1",
		MaxAuthority: missioncontrol.AuthorityTierHigh,
		AllowedTools: []string{"read"},
		Plan: missioncontrol.Plan{
			ID: "plan-1",
			Steps: []missioncontrol.Step{
				{
					ID:      "build",
					Type:    missioncontrol.StepTypeDiscussion,
					Subtype: missioncontrol.StepSubtypeAuthorization,
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
	content := expectedAuthorizationApprovalContent(job.MaxAuthority)
	initialSnapshot := missionStatusSnapshot{
		MissionFile: missionFile,
		JobID:       job.ID,
		StepID:      "build",
		Runtime: &missioncontrol.JobRuntimeState{
			JobID:         job.ID,
			State:         missioncontrol.JobStateWaitingUser,
			ActiveStepID:  "build",
			WaitingReason: "awaiting operator input",
			ApprovalRequests: []missioncontrol.ApprovalRequest{
				{
					JobID:           job.ID,
					StepID:          "build",
					RequestedAction: missioncontrol.ApprovalRequestedActionStepComplete,
					Scope:           missioncontrol.ApprovalScopeMissionStep,
					Content:         &content,
					SessionChannel:  "telegram",
					SessionChatID:   "chat-42",
					State:           missioncontrol.ApprovalStatePending,
				},
			},
		},
	}
	writeMissionStatusSnapshotFile(t, statusFile, initialSnapshot)
	if err := cmd.Flags().Set("mission-file", missionFile); err != nil {
		t.Fatalf("Flags().Set(mission-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step", "build"); err != nil {
		t.Fatalf("Flags().Set(mission-step) error = %v", err)
	}

	hub := chat.NewHub(10)
	provider := &missionStatusFixedResponseProvider{content: "unused"}
	ag := agent.NewAgentLoop(hub, provider, provider.GetDefaultModel(), 3, "", nil)
	installMissionRuntimeChangeHook(cmd, ag)

	if err := configureMissionBootstrap(cmd, ag); err != nil {
		t.Fatalf("configureMissionBootstrap() error = %v", err)
	}
	if runtime, ok := ag.MissionRuntimeState(); !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	} else if len(runtime.ApprovalRequests) != 1 || runtime.ApprovalRequests[0].SessionChannel != "telegram" || runtime.ApprovalRequests[0].SessionChatID != "chat-42" {
		t.Fatalf("MissionRuntimeState().ApprovalRequests = %#v, want preserved rehydrated session binding", runtime.ApprovalRequests)
	}

	if _, err := ag.ProcessDirect("DENY job-1 build", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(DENY) error = %v", err)
	}

	deniedSnapshot := readMissionStatusSnapshotFile(t, statusFile)
	if deniedSnapshot.Runtime == nil {
		t.Fatal("deniedSnapshot.Runtime = nil, want non-nil")
	}
	if deniedSnapshot.Runtime.State != missioncontrol.JobStateWaitingUser {
		t.Fatalf("deniedSnapshot.Runtime.State = %q, want %q", deniedSnapshot.Runtime.State, missioncontrol.JobStateWaitingUser)
	}
	if len(deniedSnapshot.Runtime.ApprovalRequests) != 1 || deniedSnapshot.Runtime.ApprovalRequests[0].State != missioncontrol.ApprovalStateDenied {
		t.Fatalf("deniedSnapshot.Runtime.ApprovalRequests = %#v, want one denied approval", deniedSnapshot.Runtime.ApprovalRequests)
	}
	if deniedSnapshot.Runtime.ApprovalRequests[0].Content == nil || *deniedSnapshot.Runtime.ApprovalRequests[0].Content != content {
		t.Fatalf("deniedSnapshot.Runtime.ApprovalRequests[0].Content = %#v, want %#v", deniedSnapshot.Runtime.ApprovalRequests[0].Content, content)
	}
	if deniedSnapshot.Runtime.ApprovalRequests[0].SessionChannel != "cli" || deniedSnapshot.Runtime.ApprovalRequests[0].SessionChatID != "direct" {
		t.Fatalf("deniedSnapshot.Runtime.ApprovalRequests[0] session = (%q, %q), want (%q, %q)", deniedSnapshot.Runtime.ApprovalRequests[0].SessionChannel, deniedSnapshot.Runtime.ApprovalRequests[0].SessionChatID, "cli", "direct")
	}
	durableDenied, err := missioncontrol.HydrateCommittedJobRuntimeState(resolveMissionStoreRoot(cmd), job.ID, time.Now().UTC())
	if err != nil {
		t.Fatalf("HydrateCommittedJobRuntimeState(denied) error = %v", err)
	}
	if durableDenied.State != missioncontrol.JobStateWaitingUser {
		t.Fatalf("HydrateCommittedJobRuntimeState(denied).State = %q, want %q", durableDenied.State, missioncontrol.JobStateWaitingUser)
	}
	if len(durableDenied.ApprovalRequests) != 1 || durableDenied.ApprovalRequests[0].State != missioncontrol.ApprovalStateDenied {
		t.Fatalf("HydrateCommittedJobRuntimeState(denied).ApprovalRequests = %#v, want one denied approval", durableDenied.ApprovalRequests)
	}
	assertMissionStatusSnapshotMatchesCommittedDurableState(t, deniedSnapshot, resolveMissionStoreRoot(cmd), job.ID, time.Now().UTC())

	statusFile = filepath.Join(t.TempDir(), "status.json")
	cmd = newMissionBootstrapTestCommand()
	if err := cmd.Flags().Set("mission-status-file", statusFile); err != nil {
		t.Fatalf("Flags().Set(second mission-status-file) error = %v", err)
	}
	writeMissionStatusSnapshotFile(t, statusFile, initialSnapshot)
	if err := cmd.Flags().Set("mission-file", missionFile); err != nil {
		t.Fatalf("Flags().Set(second mission-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step", "build"); err != nil {
		t.Fatalf("Flags().Set(second mission-step) error = %v", err)
	}
	ag = agent.NewAgentLoop(hub, provider, provider.GetDefaultModel(), 3, "", nil)
	installMissionRuntimeChangeHook(cmd, ag)
	if err := configureMissionBootstrap(cmd, ag); err != nil {
		t.Fatalf("configureMissionBootstrap() second boot error = %v", err)
	}

	if _, err := ag.ProcessDirect("APPROVE job-1 build", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(APPROVE) error = %v", err)
	}

	approvedSnapshot := readMissionStatusSnapshotFile(t, statusFile)
	if approvedSnapshot.Runtime == nil {
		t.Fatal("approvedSnapshot.Runtime = nil, want non-nil")
	}
	if approvedSnapshot.Runtime.State != missioncontrol.JobStatePaused {
		t.Fatalf("approvedSnapshot.Runtime.State = %q, want %q", approvedSnapshot.Runtime.State, missioncontrol.JobStatePaused)
	}
	if len(approvedSnapshot.Runtime.CompletedSteps) != 1 || approvedSnapshot.Runtime.CompletedSteps[0].StepID != "build" {
		t.Fatalf("approvedSnapshot.Runtime.CompletedSteps = %#v, want build completion", approvedSnapshot.Runtime.CompletedSteps)
	}
	if len(approvedSnapshot.Runtime.ApprovalGrants) != 1 || approvedSnapshot.Runtime.ApprovalGrants[0].State != missioncontrol.ApprovalStateGranted {
		t.Fatalf("approvedSnapshot.Runtime.ApprovalGrants = %#v, want one granted approval", approvedSnapshot.Runtime.ApprovalGrants)
	}
	if len(approvedSnapshot.Runtime.ApprovalRequests) != 1 || approvedSnapshot.Runtime.ApprovalRequests[0].Content == nil || *approvedSnapshot.Runtime.ApprovalRequests[0].Content != content {
		t.Fatalf("approvedSnapshot.Runtime.ApprovalRequests = %#v, want persisted enriched request content", approvedSnapshot.Runtime.ApprovalRequests)
	}
	if approvedSnapshot.Runtime.ApprovalRequests[0].SessionChannel != "cli" || approvedSnapshot.Runtime.ApprovalRequests[0].SessionChatID != "direct" {
		t.Fatalf("approvedSnapshot.Runtime.ApprovalRequests[0] session = (%q, %q), want (%q, %q)", approvedSnapshot.Runtime.ApprovalRequests[0].SessionChannel, approvedSnapshot.Runtime.ApprovalRequests[0].SessionChatID, "cli", "direct")
	}
	if approvedSnapshot.Runtime.ApprovalGrants[0].SessionChannel != "cli" || approvedSnapshot.Runtime.ApprovalGrants[0].SessionChatID != "direct" {
		t.Fatalf("approvedSnapshot.Runtime.ApprovalGrants[0] session = (%q, %q), want (%q, %q)", approvedSnapshot.Runtime.ApprovalGrants[0].SessionChannel, approvedSnapshot.Runtime.ApprovalGrants[0].SessionChatID, "cli", "direct")
	}
	durableApproved, err := missioncontrol.HydrateCommittedJobRuntimeState(resolveMissionStoreRoot(cmd), job.ID, time.Now().UTC())
	if err != nil {
		t.Fatalf("HydrateCommittedJobRuntimeState(approved) error = %v", err)
	}
	if durableApproved.State != missioncontrol.JobStatePaused {
		t.Fatalf("HydrateCommittedJobRuntimeState(approved).State = %q, want %q", durableApproved.State, missioncontrol.JobStatePaused)
	}
	if len(durableApproved.CompletedSteps) != 1 || durableApproved.CompletedSteps[0].StepID != "build" {
		t.Fatalf("HydrateCommittedJobRuntimeState(approved).CompletedSteps = %#v, want build completion", durableApproved.CompletedSteps)
	}
	if len(durableApproved.ApprovalGrants) != 1 || durableApproved.ApprovalGrants[0].State != missioncontrol.ApprovalStateGranted {
		t.Fatalf("HydrateCommittedJobRuntimeState(approved).ApprovalGrants = %#v, want one granted approval", durableApproved.ApprovalGrants)
	}
	assertMissionStatusSnapshotMatchesCommittedDurableState(t, approvedSnapshot, resolveMissionStoreRoot(cmd), job.ID, time.Now().UTC())
}

func TestMissionStatusRuntimeChangeHookPersistsRehydratedNaturalApprovalLifecycle(t *testing.T) {
	statusFile := filepath.Join(t.TempDir(), "status.json")
	cmd := newMissionBootstrapTestCommand()
	if err := cmd.Flags().Set("mission-status-file", statusFile); err != nil {
		t.Fatalf("Flags().Set(mission-status-file) error = %v", err)
	}

	job := missioncontrol.Job{
		ID:           "job-1",
		MaxAuthority: missioncontrol.AuthorityTierHigh,
		AllowedTools: []string{"read"},
		Plan: missioncontrol.Plan{
			ID: "plan-1",
			Steps: []missioncontrol.Step{
				{
					ID:      "build",
					Type:    missioncontrol.StepTypeDiscussion,
					Subtype: missioncontrol.StepSubtypeAuthorization,
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
	content := expectedAuthorizationApprovalContent(job.MaxAuthority)
	initialSnapshot := missionStatusSnapshot{
		MissionFile:    missionFile,
		JobID:          job.ID,
		StepID:         "build",
		RuntimeControl: runtimeControlForBootstrapStep(t, job, "build"),
		Runtime: &missioncontrol.JobRuntimeState{
			JobID:         job.ID,
			State:         missioncontrol.JobStateWaitingUser,
			ActiveStepID:  "build",
			WaitingReason: "awaiting operator input",
			ApprovalRequests: []missioncontrol.ApprovalRequest{
				{
					JobID:           job.ID,
					StepID:          "build",
					RequestedAction: missioncontrol.ApprovalRequestedActionStepComplete,
					Scope:           missioncontrol.ApprovalScopeMissionStep,
					Content:         &content,
					SessionChannel:  "telegram",
					SessionChatID:   "chat-42",
					State:           missioncontrol.ApprovalStatePending,
				},
			},
		},
	}
	writeMissionStatusSnapshotFile(t, statusFile, initialSnapshot)
	if err := cmd.Flags().Set("mission-file", missionFile); err != nil {
		t.Fatalf("Flags().Set(mission-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step", "build"); err != nil {
		t.Fatalf("Flags().Set(mission-step) error = %v", err)
	}

	hub := chat.NewHub(10)
	provider := &missionStatusFixedResponseProvider{content: "unused"}
	ag := agent.NewAgentLoop(hub, provider, provider.GetDefaultModel(), 3, "", nil)
	installMissionRuntimeChangeHook(cmd, ag)

	if err := configureMissionBootstrap(cmd, ag); err != nil {
		t.Fatalf("configureMissionBootstrap() error = %v", err)
	}
	if runtime, ok := ag.MissionRuntimeState(); !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	} else if len(runtime.ApprovalRequests) != 1 || runtime.ApprovalRequests[0].Content == nil || *runtime.ApprovalRequests[0].Content != content || runtime.ApprovalRequests[0].SessionChannel != "telegram" || runtime.ApprovalRequests[0].SessionChatID != "chat-42" {
		t.Fatalf("MissionRuntimeState().ApprovalRequests = %#v, want preserved enriched request content and session binding after rehydration", runtime.ApprovalRequests)
	}
	if _, err := ag.ProcessDirect("no", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(no) error = %v", err)
	}

	deniedSnapshot := readMissionStatusSnapshotFile(t, statusFile)
	if deniedSnapshot.Runtime == nil {
		t.Fatal("deniedSnapshot.Runtime = nil, want non-nil")
	}
	if deniedSnapshot.Runtime.State != missioncontrol.JobStateWaitingUser {
		t.Fatalf("deniedSnapshot.Runtime.State = %q, want %q", deniedSnapshot.Runtime.State, missioncontrol.JobStateWaitingUser)
	}
	if len(deniedSnapshot.Runtime.ApprovalRequests) != 1 || deniedSnapshot.Runtime.ApprovalRequests[0].State != missioncontrol.ApprovalStateDenied {
		t.Fatalf("deniedSnapshot.Runtime.ApprovalRequests = %#v, want one denied approval", deniedSnapshot.Runtime.ApprovalRequests)
	}
	if deniedSnapshot.Runtime.ApprovalRequests[0].Content == nil || *deniedSnapshot.Runtime.ApprovalRequests[0].Content != content {
		t.Fatalf("deniedSnapshot.Runtime.ApprovalRequests[0].Content = %#v, want %#v", deniedSnapshot.Runtime.ApprovalRequests[0].Content, content)
	}
	if deniedSnapshot.Runtime.ApprovalRequests[0].SessionChannel != "cli" || deniedSnapshot.Runtime.ApprovalRequests[0].SessionChatID != "direct" {
		t.Fatalf("deniedSnapshot.Runtime.ApprovalRequests[0] session = (%q, %q), want (%q, %q)", deniedSnapshot.Runtime.ApprovalRequests[0].SessionChannel, deniedSnapshot.Runtime.ApprovalRequests[0].SessionChatID, "cli", "direct")
	}
	assertMissionStatusSnapshotMatchesCommittedDurableState(t, deniedSnapshot, resolveMissionStoreRoot(cmd), job.ID, time.Now().UTC())

	statusFile = filepath.Join(t.TempDir(), "status.json")
	cmd = newMissionBootstrapTestCommand()
	if err := cmd.Flags().Set("mission-status-file", statusFile); err != nil {
		t.Fatalf("Flags().Set(second mission-status-file) error = %v", err)
	}
	writeMissionStatusSnapshotFile(t, statusFile, initialSnapshot)
	if err := cmd.Flags().Set("mission-file", missionFile); err != nil {
		t.Fatalf("Flags().Set(second mission-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step", "build"); err != nil {
		t.Fatalf("Flags().Set(second mission-step) error = %v", err)
	}
	ag = agent.NewAgentLoop(hub, provider, provider.GetDefaultModel(), 3, "", nil)
	installMissionRuntimeChangeHook(cmd, ag)
	if err := configureMissionBootstrap(cmd, ag); err != nil {
		t.Fatalf("configureMissionBootstrap() second boot error = %v", err)
	}
	if _, err := ag.ProcessDirect("yes", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(yes) error = %v", err)
	}

	approvedSnapshot := readMissionStatusSnapshotFile(t, statusFile)
	if approvedSnapshot.Runtime == nil {
		t.Fatal("approvedSnapshot.Runtime = nil, want non-nil")
	}
	if approvedSnapshot.Runtime.State != missioncontrol.JobStatePaused {
		t.Fatalf("approvedSnapshot.Runtime.State = %q, want %q", approvedSnapshot.Runtime.State, missioncontrol.JobStatePaused)
	}
	if len(approvedSnapshot.Runtime.ApprovalGrants) != 1 || approvedSnapshot.Runtime.ApprovalGrants[0].GrantedVia != missioncontrol.ApprovalGrantedViaOperatorReply {
		t.Fatalf("approvedSnapshot.Runtime.ApprovalGrants = %#v, want one natural-language approval grant", approvedSnapshot.Runtime.ApprovalGrants)
	}
	if len(approvedSnapshot.Runtime.ApprovalRequests) != 1 || approvedSnapshot.Runtime.ApprovalRequests[0].Content == nil || *approvedSnapshot.Runtime.ApprovalRequests[0].Content != content {
		t.Fatalf("approvedSnapshot.Runtime.ApprovalRequests = %#v, want persisted enriched request content", approvedSnapshot.Runtime.ApprovalRequests)
	}
	assertMissionStatusSnapshotMatchesCommittedDurableState(t, approvedSnapshot, resolveMissionStoreRoot(cmd), job.ID, time.Now().UTC())
	if approvedSnapshot.Runtime.ApprovalRequests[0].SessionChannel != "cli" || approvedSnapshot.Runtime.ApprovalRequests[0].SessionChatID != "direct" {
		t.Fatalf("approvedSnapshot.Runtime.ApprovalRequests[0] session = (%q, %q), want (%q, %q)", approvedSnapshot.Runtime.ApprovalRequests[0].SessionChannel, approvedSnapshot.Runtime.ApprovalRequests[0].SessionChatID, "cli", "direct")
	}
	if approvedSnapshot.Runtime.ApprovalGrants[0].SessionChannel != "cli" || approvedSnapshot.Runtime.ApprovalGrants[0].SessionChatID != "direct" {
		t.Fatalf("approvedSnapshot.Runtime.ApprovalGrants[0] session = (%q, %q), want (%q, %q)", approvedSnapshot.Runtime.ApprovalGrants[0].SessionChannel, approvedSnapshot.Runtime.ApprovalGrants[0].SessionChatID, "cli", "direct")
	}
}

func TestMissionStatusRuntimeChangeHookPersistsApprovalExpiryLifecycle(t *testing.T) {
	statusFile := filepath.Join(t.TempDir(), "status.json")
	cmd := newMissionBootstrapTestCommand()
	if err := cmd.Flags().Set("mission-status-file", statusFile); err != nil {
		t.Fatalf("Flags().Set(mission-status-file) error = %v", err)
	}

	hub := chat.NewHub(10)
	provider := &missionStatusFixedResponseProvider{content: "Need approval before continuing."}
	ag := agent.NewAgentLoop(hub, provider, provider.GetDefaultModel(), 3, "", nil)
	installMissionRuntimeChangeHook(cmd, ag)

	job := missioncontrol.Job{
		ID:           "job-1",
		MaxAuthority: missioncontrol.AuthorityTierHigh,
		AllowedTools: []string{"read"},
		Plan: missioncontrol.Plan{
			ID: "plan-1",
			Steps: []missioncontrol.Step{
				{
					ID:      "build",
					Type:    missioncontrol.StepTypeDiscussion,
					Subtype: missioncontrol.StepSubtypeAuthorization,
				},
				{
					ID:        "final",
					Type:      missioncontrol.StepTypeFinalResponse,
					DependsOn: []string{"build"},
				},
			},
		},
	}

	if err := ag.ActivateMissionStep(job, "build"); err != nil {
		t.Fatalf("ActivateMissionStep() error = %v", err)
	}
	if _, err := ag.ProcessDirect("continue", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(continue) error = %v", err)
	}
	if _, err := ag.ProcessDirect("timeout", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(timeout) error = %v", err)
	}

	snapshot := readMissionStatusSnapshotFile(t, statusFile)
	if snapshot.Runtime == nil {
		t.Fatal("snapshot.Runtime = nil, want non-nil")
	}
	if len(snapshot.Runtime.ApprovalRequests) != 1 || snapshot.Runtime.ApprovalRequests[0].ExpiresAt.IsZero() {
		t.Fatalf("snapshot.Runtime.ApprovalRequests = %#v, want one stamped approval request", snapshot.Runtime.ApprovalRequests)
	}
	if len(snapshot.Runtime.ApprovalRequests) != 1 || snapshot.Runtime.ApprovalRequests[0].State != missioncontrol.ApprovalStateExpired {
		t.Fatalf("snapshot.Runtime.ApprovalRequests = %#v, want one expired approval", snapshot.Runtime.ApprovalRequests)
	}
	if snapshot.Runtime.ApprovalRequests[0].ExpiresAt.IsZero() {
		t.Fatalf("snapshot.Runtime.ApprovalRequests[0].ExpiresAt = %v, want non-zero", snapshot.Runtime.ApprovalRequests[0].ExpiresAt)
	}
}

func TestMissionStatusBootstrapRehydratedYesDoesNotBindExpiredApproval(t *testing.T) {
	statusFile := filepath.Join(t.TempDir(), "status.json")
	cmd := newMissionBootstrapTestCommand()
	if err := cmd.Flags().Set("mission-status-file", statusFile); err != nil {
		t.Fatalf("Flags().Set(mission-status-file) error = %v", err)
	}

	job := missioncontrol.Job{
		ID:           "job-1",
		MaxAuthority: missioncontrol.AuthorityTierHigh,
		AllowedTools: []string{"read"},
		Plan: missioncontrol.Plan{
			ID: "plan-1",
			Steps: []missioncontrol.Step{
				{
					ID:      "build",
					Type:    missioncontrol.StepTypeDiscussion,
					Subtype: missioncontrol.StepSubtypeAuthorization,
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
	expiredAt := time.Now().Add(-1 * time.Minute)
	writeMissionStatusSnapshotFile(t, statusFile, missionStatusSnapshot{
		MissionFile: missionFile,
		JobID:       job.ID,
		StepID:      "build",
		Runtime: &missioncontrol.JobRuntimeState{
			JobID:         job.ID,
			State:         missioncontrol.JobStateWaitingUser,
			ActiveStepID:  "build",
			WaitingReason: "awaiting operator input",
			ApprovalRequests: []missioncontrol.ApprovalRequest{
				{
					JobID:           job.ID,
					StepID:          "build",
					RequestedAction: missioncontrol.ApprovalRequestedActionStepComplete,
					Scope:           missioncontrol.ApprovalScopeMissionStep,
					State:           missioncontrol.ApprovalStatePending,
					ExpiresAt:       expiredAt,
				},
			},
		},
	})
	if err := cmd.Flags().Set("mission-file", missionFile); err != nil {
		t.Fatalf("Flags().Set(mission-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step", "build"); err != nil {
		t.Fatalf("Flags().Set(mission-step) error = %v", err)
	}

	ag := newMissionBootstrapTestLoop()
	installMissionRuntimeChangeHook(cmd, ag)
	if err := configureMissionBootstrap(cmd, ag); err != nil {
		t.Fatalf("configureMissionBootstrap() error = %v", err)
	}

	snapshot := readMissionStatusSnapshotFile(t, statusFile)
	if snapshot.Runtime == nil {
		t.Fatal("snapshot.Runtime = nil, want non-nil")
	}
	if len(snapshot.Runtime.ApprovalRequests) != 1 || snapshot.Runtime.ApprovalRequests[0].State != missioncontrol.ApprovalStateExpired {
		t.Fatalf("snapshot.Runtime.ApprovalRequests = %#v, want one expired approval immediately after bootstrap", snapshot.Runtime.ApprovalRequests)
	}

	resp, err := ag.ProcessDirect("yes", 2*time.Second)
	if err != nil {
		t.Fatalf("ProcessDirect(yes) error = %v", err)
	}
	if !strings.Contains(resp, "(stub) Echo") {
		t.Fatalf("ProcessDirect(yes) response = %q, want provider fallback", resp)
	}

	runtime, ok := ag.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if len(runtime.ApprovalRequests) != 1 || runtime.ApprovalRequests[0].State != missioncontrol.ApprovalStateExpired {
		t.Fatalf("MissionRuntimeState().ApprovalRequests = %#v, want one expired approval", runtime.ApprovalRequests)
	}

	snapshot = readMissionStatusSnapshotFile(t, statusFile)
	if snapshot.Runtime == nil {
		t.Fatal("snapshot.Runtime = nil, want non-nil")
	}
	if len(snapshot.Runtime.ApprovalRequests) != 1 || snapshot.Runtime.ApprovalRequests[0].State != missioncontrol.ApprovalStateExpired {
		t.Fatalf("snapshot.Runtime.ApprovalRequests = %#v, want one expired approval", snapshot.Runtime.ApprovalRequests)
	}
}

func TestMissionStatusBootstrapRehydratedApproveDoesNotBindExpiredApproval(t *testing.T) {
	statusFile := filepath.Join(t.TempDir(), "status.json")
	cmd := newMissionBootstrapTestCommand()
	if err := cmd.Flags().Set("mission-status-file", statusFile); err != nil {
		t.Fatalf("Flags().Set(mission-status-file) error = %v", err)
	}

	job := missioncontrol.Job{
		ID:           "job-1",
		MaxAuthority: missioncontrol.AuthorityTierHigh,
		AllowedTools: []string{"read"},
		Plan: missioncontrol.Plan{
			ID: "plan-1",
			Steps: []missioncontrol.Step{
				{
					ID:      "build",
					Type:    missioncontrol.StepTypeDiscussion,
					Subtype: missioncontrol.StepSubtypeAuthorization,
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
	expiredAt := time.Now().Add(-1 * time.Minute)
	writeMissionStatusSnapshotFile(t, statusFile, missionStatusSnapshot{
		MissionFile: missionFile,
		JobID:       job.ID,
		StepID:      "build",
		Runtime: &missioncontrol.JobRuntimeState{
			JobID:         job.ID,
			State:         missioncontrol.JobStateWaitingUser,
			ActiveStepID:  "build",
			WaitingReason: "awaiting operator input",
			ApprovalRequests: []missioncontrol.ApprovalRequest{
				{
					JobID:           job.ID,
					StepID:          "build",
					RequestedAction: missioncontrol.ApprovalRequestedActionStepComplete,
					Scope:           missioncontrol.ApprovalScopeMissionStep,
					State:           missioncontrol.ApprovalStatePending,
					ExpiresAt:       expiredAt,
				},
			},
			UpdatedAt: expiredAt.Add(-1 * time.Minute),
		},
		RuntimeControl: &missioncontrol.RuntimeControlContext{
			JobID: job.ID,
			Step: missioncontrol.Step{
				ID:      "build",
				Type:    missioncontrol.StepTypeDiscussion,
				Subtype: missioncontrol.StepSubtypeAuthorization,
			},
		},
	})
	if err := cmd.Flags().Set("mission-file", missionFile); err != nil {
		t.Fatalf("Flags().Set(mission-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step", "build"); err != nil {
		t.Fatalf("Flags().Set(mission-step) error = %v", err)
	}

	ag := newMissionBootstrapTestLoop()
	installMissionRuntimeChangeHook(cmd, ag)
	if err := configureMissionBootstrap(cmd, ag); err != nil {
		t.Fatalf("configureMissionBootstrap() error = %v", err)
	}

	_, err := ag.ProcessDirect("APPROVE job-1 build", 2*time.Second)
	if err == nil {
		t.Fatal("ProcessDirect(APPROVE) error = nil, want expired approval rejection")
	}
	if !strings.Contains(err.Error(), "no pending approval request matches the active job and step") {
		t.Fatalf("ProcessDirect(APPROVE) error = %q, want expired approval rejection", err)
	}

	runtime, ok := ag.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if len(runtime.ApprovalRequests) != 1 || runtime.ApprovalRequests[0].State != missioncontrol.ApprovalStateExpired {
		t.Fatalf("MissionRuntimeState().ApprovalRequests = %#v, want one expired approval", runtime.ApprovalRequests)
	}
}

func TestMissionStatusBootstrapRehydratedApproveUsesLatestNonSupersededApproval(t *testing.T) {
	statusFile := filepath.Join(t.TempDir(), "status.json")
	cmd := newMissionBootstrapTestCommand()
	if err := cmd.Flags().Set("mission-status-file", statusFile); err != nil {
		t.Fatalf("Flags().Set(mission-status-file) error = %v", err)
	}

	job := missioncontrol.Job{
		ID:           "job-1",
		MaxAuthority: missioncontrol.AuthorityTierHigh,
		AllowedTools: []string{"read"},
		Plan: missioncontrol.Plan{
			ID: "plan-1",
			Steps: []missioncontrol.Step{
				{
					ID:      "build",
					Type:    missioncontrol.StepTypeDiscussion,
					Subtype: missioncontrol.StepSubtypeAuthorization,
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
	now := time.Now()
	writeMissionStatusSnapshotFile(t, statusFile, missionStatusSnapshot{
		MissionFile: missionFile,
		JobID:       job.ID,
		StepID:      "build",
		Runtime: &missioncontrol.JobRuntimeState{
			JobID:         job.ID,
			State:         missioncontrol.JobStateWaitingUser,
			ActiveStepID:  "build",
			WaitingReason: "awaiting operator input",
			ApprovalRequests: []missioncontrol.ApprovalRequest{
				{
					JobID:           job.ID,
					StepID:          "build",
					RequestedAction: missioncontrol.ApprovalRequestedActionStepComplete,
					Scope:           missioncontrol.ApprovalScopeMissionStep,
					State:           missioncontrol.ApprovalStateSuperseded,
					RequestedAt:     now.Add(-2 * time.Minute),
					ResolvedAt:      now.Add(-1 * time.Minute),
					SupersededAt:    now.Add(-1 * time.Minute),
				},
				{
					JobID:           job.ID,
					StepID:          "build",
					RequestedAction: missioncontrol.ApprovalRequestedActionStepComplete,
					Scope:           missioncontrol.ApprovalScopeMissionStep,
					State:           missioncontrol.ApprovalStatePending,
					RequestedAt:     now,
				},
			},
		},
	})
	if err := cmd.Flags().Set("mission-file", missionFile); err != nil {
		t.Fatalf("Flags().Set(mission-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step", "build"); err != nil {
		t.Fatalf("Flags().Set(mission-step) error = %v", err)
	}

	ag := newMissionBootstrapTestLoop()
	if err := configureMissionBootstrap(cmd, ag); err != nil {
		t.Fatalf("configureMissionBootstrap() error = %v", err)
	}

	if _, err := ag.ProcessDirect("APPROVE job-1 build", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(APPROVE) error = %v", err)
	}

	runtime, ok := ag.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if len(runtime.ApprovalRequests) != 2 || runtime.ApprovalRequests[0].State != missioncontrol.ApprovalStateSuperseded || runtime.ApprovalRequests[1].State != missioncontrol.ApprovalStateGranted {
		t.Fatalf("MissionRuntimeState().ApprovalRequests = %#v, want superseded then granted approvals", runtime.ApprovalRequests)
	}
}

func TestMissionStatusRuntimeChangeHookPersistsSupersededApprovalLifecycle(t *testing.T) {
	statusFile := filepath.Join(t.TempDir(), "status.json")
	cmd := newMissionBootstrapTestCommand()
	if err := cmd.Flags().Set("mission-status-file", statusFile); err != nil {
		t.Fatalf("Flags().Set(mission-status-file) error = %v", err)
	}

	job := missioncontrol.Job{
		ID:           "job-1",
		MaxAuthority: missioncontrol.AuthorityTierHigh,
		AllowedTools: []string{"read"},
		Plan: missioncontrol.Plan{
			ID: "plan-1",
			Steps: []missioncontrol.Step{
				{
					ID:      "build",
					Type:    missioncontrol.StepTypeDiscussion,
					Subtype: missioncontrol.StepSubtypeAuthorization,
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
	now := time.Now()
	writeMissionStatusSnapshotFile(t, statusFile, missionStatusSnapshot{
		MissionFile:    missionFile,
		JobID:          job.ID,
		StepID:         "build",
		RuntimeControl: runtimeControlForBootstrapStep(t, job, "build"),
		Runtime: &missioncontrol.JobRuntimeState{
			JobID:         job.ID,
			State:         missioncontrol.JobStateWaitingUser,
			ActiveStepID:  "build",
			WaitingReason: "awaiting operator input",
			ApprovalRequests: []missioncontrol.ApprovalRequest{
				{
					JobID:           job.ID,
					StepID:          "build",
					RequestedAction: missioncontrol.ApprovalRequestedActionStepComplete,
					Scope:           missioncontrol.ApprovalScopeMissionStep,
					State:           missioncontrol.ApprovalStateSuperseded,
					RequestedAt:     now.Add(-2 * time.Minute),
					ResolvedAt:      now.Add(-1 * time.Minute),
					SupersededAt:    now.Add(-1 * time.Minute),
				},
				{
					JobID:           job.ID,
					StepID:          "build",
					RequestedAction: missioncontrol.ApprovalRequestedActionStepComplete,
					Scope:           missioncontrol.ApprovalScopeMissionStep,
					State:           missioncontrol.ApprovalStatePending,
					RequestedAt:     now,
				},
			},
		},
	})
	if err := cmd.Flags().Set("mission-file", missionFile); err != nil {
		t.Fatalf("Flags().Set(mission-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step", "build"); err != nil {
		t.Fatalf("Flags().Set(mission-step) error = %v", err)
	}

	hub := chat.NewHub(10)
	provider := &missionStatusFixedResponseProvider{content: "unused"}
	ag := agent.NewAgentLoop(hub, provider, provider.GetDefaultModel(), 3, "", nil)
	installMissionRuntimeChangeHook(cmd, ag)
	if err := configureMissionBootstrap(cmd, ag); err != nil {
		t.Fatalf("configureMissionBootstrap() error = %v", err)
	}

	if _, err := ag.ProcessDirect("APPROVE job-1 build", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(APPROVE) error = %v", err)
	}

	snapshot := readMissionStatusSnapshotFile(t, statusFile)
	if snapshot.Runtime == nil {
		t.Fatal("snapshot.Runtime = nil, want non-nil")
	}
	if len(snapshot.Runtime.ApprovalRequests) != 2 || snapshot.Runtime.ApprovalRequests[0].State != missioncontrol.ApprovalStateSuperseded || snapshot.Runtime.ApprovalRequests[1].State != missioncontrol.ApprovalStateGranted {
		t.Fatalf("snapshot.Runtime.ApprovalRequests = %#v, want superseded then granted approvals", snapshot.Runtime.ApprovalRequests)
	}
}

func TestMissionStatusRuntimeChangeHookPersistsPauseResumeAbortLifecycle(t *testing.T) {
	statusFile := filepath.Join(t.TempDir(), "status.json")
	cmd := newMissionBootstrapTestCommand()
	if err := cmd.Flags().Set("mission-status-file", statusFile); err != nil {
		t.Fatalf("Flags().Set(mission-status-file) error = %v", err)
	}

	hub := chat.NewHub(10)
	provider := &missionStatusFixedResponseProvider{content: "unused"}
	ag := agent.NewAgentLoop(hub, provider, provider.GetDefaultModel(), 3, "", nil)
	installMissionRuntimeChangeHook(cmd, ag)

	if err := ag.ActivateMissionStep(testMissionBootstrapJob(), "build"); err != nil {
		t.Fatalf("ActivateMissionStep() error = %v", err)
	}
	if _, err := ag.ProcessDirect("PAUSE job-1", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(PAUSE) error = %v", err)
	}

	pausedSnapshot := readMissionStatusSnapshotFile(t, statusFile)
	if pausedSnapshot.Runtime == nil {
		t.Fatal("pausedSnapshot.Runtime = nil, want non-nil")
	}
	if pausedSnapshot.Runtime.State != missioncontrol.JobStatePaused {
		t.Fatalf("pausedSnapshot.Runtime.State = %q, want %q", pausedSnapshot.Runtime.State, missioncontrol.JobStatePaused)
	}
	if pausedSnapshot.Runtime.ActiveStepID != "build" {
		t.Fatalf("pausedSnapshot.Runtime.ActiveStepID = %q, want %q", pausedSnapshot.Runtime.ActiveStepID, "build")
	}
	if len(pausedSnapshot.Runtime.CompletedSteps) != 0 {
		t.Fatalf("pausedSnapshot.Runtime.CompletedSteps = %#v, want empty", pausedSnapshot.Runtime.CompletedSteps)
	}
	if pausedSnapshot.RuntimeControl == nil || pausedSnapshot.RuntimeControl.Step.ID != "build" {
		t.Fatalf("pausedSnapshot.RuntimeControl = %#v, want persisted build control", pausedSnapshot.RuntimeControl)
	}
	assertMissionStatusSnapshotMatchesCommittedDurableState(t, pausedSnapshot, resolveMissionStoreRoot(cmd), "job-1", time.Now().UTC())

	ag.ClearMissionStep()

	tornDownSnapshot := readMissionStatusSnapshotFile(t, statusFile)
	if tornDownSnapshot.Active {
		t.Fatal("tornDownSnapshot.Active = true, want false after teardown")
	}
	if tornDownSnapshot.Runtime == nil {
		t.Fatal("tornDownSnapshot.Runtime = nil, want non-nil")
	}
	if tornDownSnapshot.Runtime.State != missioncontrol.JobStatePaused {
		t.Fatalf("tornDownSnapshot.Runtime.State = %q, want %q", tornDownSnapshot.Runtime.State, missioncontrol.JobStatePaused)
	}
	if tornDownSnapshot.Runtime.ActiveStepID != "build" {
		t.Fatalf("tornDownSnapshot.Runtime.ActiveStepID = %q, want %q", tornDownSnapshot.Runtime.ActiveStepID, "build")
	}
	if tornDownSnapshot.RuntimeControl == nil || tornDownSnapshot.RuntimeControl.Step.ID != "build" {
		t.Fatalf("tornDownSnapshot.RuntimeControl = %#v, want persisted build control", tornDownSnapshot.RuntimeControl)
	}
	assertMissionStatusSnapshotMatchesCommittedDurableState(t, tornDownSnapshot, resolveMissionStoreRoot(cmd), "job-1", time.Now().UTC())

	if _, err := ag.ProcessDirect("RESUME job-1", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(RESUME) error = %v", err)
	}

	resumedSnapshot := readMissionStatusSnapshotFile(t, statusFile)
	if resumedSnapshot.Runtime == nil {
		t.Fatal("resumedSnapshot.Runtime = nil, want non-nil")
	}
	if resumedSnapshot.Runtime.State != missioncontrol.JobStateRunning {
		t.Fatalf("resumedSnapshot.Runtime.State = %q, want %q", resumedSnapshot.Runtime.State, missioncontrol.JobStateRunning)
	}
	if resumedSnapshot.Runtime.ActiveStepID != "build" {
		t.Fatalf("resumedSnapshot.Runtime.ActiveStepID = %q, want %q", resumedSnapshot.Runtime.ActiveStepID, "build")
	}
	if resumedSnapshot.RuntimeControl == nil || resumedSnapshot.RuntimeControl.Step.ID != "build" {
		t.Fatalf("resumedSnapshot.RuntimeControl = %#v, want persisted build control", resumedSnapshot.RuntimeControl)
	}
	assertMissionStatusSnapshotMatchesCommittedDurableState(t, resumedSnapshot, resolveMissionStoreRoot(cmd), "job-1", time.Now().UTC())
	durableResumed, err := missioncontrol.HydrateCommittedJobRuntimeState(resolveMissionStoreRoot(cmd), "job-1", time.Now().UTC())
	if err != nil {
		t.Fatalf("HydrateCommittedJobRuntimeState(resumed) error = %v", err)
	}
	if durableResumed.State != missioncontrol.JobStateRunning || durableResumed.ActiveStepID != "build" {
		t.Fatalf("HydrateCommittedJobRuntimeState(resumed) = %#v, want running build runtime", durableResumed)
	}

	ag.ClearMissionStep()

	if _, err := ag.ProcessDirect("ABORT job-1", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(ABORT) error = %v", err)
	}

	abortedSnapshot := readMissionStatusSnapshotFile(t, statusFile)
	if abortedSnapshot.Runtime == nil {
		t.Fatal("abortedSnapshot.Runtime = nil, want non-nil")
	}
	if abortedSnapshot.Runtime.State != missioncontrol.JobStateAborted {
		t.Fatalf("abortedSnapshot.Runtime.State = %q, want %q", abortedSnapshot.Runtime.State, missioncontrol.JobStateAborted)
	}
	if abortedSnapshot.Runtime.AbortedReason != missioncontrol.RuntimeAbortReasonOperatorCommand {
		t.Fatalf("abortedSnapshot.Runtime.AbortedReason = %q, want %q", abortedSnapshot.Runtime.AbortedReason, missioncontrol.RuntimeAbortReasonOperatorCommand)
	}
	if abortedSnapshot.Active {
		t.Fatal("abortedSnapshot.Active = true, want false")
	}
	if abortedSnapshot.Runtime.InspectablePlan == nil {
		t.Fatal("abortedSnapshot.Runtime.InspectablePlan = nil, want persisted inspectable plan")
	}
	if len(abortedSnapshot.Runtime.InspectablePlan.Steps) != len(testMissionBootstrapJob().Plan.Steps) {
		t.Fatalf("len(abortedSnapshot.Runtime.InspectablePlan.Steps) = %d, want %d", len(abortedSnapshot.Runtime.InspectablePlan.Steps), len(testMissionBootstrapJob().Plan.Steps))
	}
	if abortedSnapshot.RuntimeControl != nil {
		t.Fatalf("abortedSnapshot.RuntimeControl = %#v, want nil for terminal aborted snapshot", abortedSnapshot.RuntimeControl)
	}
	assertMissionStatusSnapshotMatchesCommittedDurableState(t, abortedSnapshot, resolveMissionStoreRoot(cmd), "job-1", time.Now().UTC())
	durableAborted, err := missioncontrol.HydrateCommittedJobRuntimeState(resolveMissionStoreRoot(cmd), "job-1", time.Now().UTC())
	if err != nil {
		t.Fatalf("HydrateCommittedJobRuntimeState(aborted) error = %v", err)
	}
	if durableAborted.State != missioncontrol.JobStateAborted {
		t.Fatalf("HydrateCommittedJobRuntimeState(aborted).State = %q, want %q", durableAborted.State, missioncontrol.JobStateAborted)
	}
}

func TestMissionStatusRuntimeChangeHookPersistsDurableAbortFromWaitingUserAfterTeardown(t *testing.T) {
	statusFile := filepath.Join(t.TempDir(), "status.json")
	cmd := newMissionBootstrapTestCommand()
	if err := cmd.Flags().Set("mission-status-file", statusFile); err != nil {
		t.Fatalf("Flags().Set(mission-status-file) error = %v", err)
	}

	hub := chat.NewHub(10)
	provider := &missionStatusFixedResponseProvider{content: "Need approval before continuing."}
	ag := agent.NewAgentLoop(hub, provider, provider.GetDefaultModel(), 3, "", nil)
	installMissionRuntimeChangeHook(cmd, ag)

	job := missioncontrol.Job{
		ID:           "job-1",
		MaxAuthority: missioncontrol.AuthorityTierHigh,
		AllowedTools: []string{"read"},
		Plan: missioncontrol.Plan{
			ID: "plan-1",
			Steps: []missioncontrol.Step{
				{
					ID:      "build",
					Type:    missioncontrol.StepTypeDiscussion,
					Subtype: missioncontrol.StepSubtypeAuthorization,
				},
				{
					ID:        "final",
					Type:      missioncontrol.StepTypeFinalResponse,
					DependsOn: []string{"build"},
				},
			},
		},
	}

	if err := ag.ActivateMissionStep(job, "build"); err != nil {
		t.Fatalf("ActivateMissionStep() error = %v", err)
	}
	if _, err := ag.ProcessDirect("continue", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(continue) error = %v", err)
	}

	ag.ClearMissionStep()

	tornDownSnapshot := readMissionStatusSnapshotFile(t, statusFile)
	if tornDownSnapshot.Active {
		t.Fatal("tornDownSnapshot.Active = true, want false after teardown")
	}
	if tornDownSnapshot.Runtime == nil {
		t.Fatal("tornDownSnapshot.Runtime = nil, want non-nil")
	}
	if tornDownSnapshot.Runtime.State != missioncontrol.JobStateWaitingUser {
		t.Fatalf("tornDownSnapshot.Runtime.State = %q, want %q", tornDownSnapshot.Runtime.State, missioncontrol.JobStateWaitingUser)
	}
	if len(tornDownSnapshot.Runtime.ApprovalRequests) != 1 || tornDownSnapshot.Runtime.ApprovalRequests[0].State != missioncontrol.ApprovalStatePending {
		t.Fatalf("tornDownSnapshot.Runtime.ApprovalRequests = %#v, want one pending approval", tornDownSnapshot.Runtime.ApprovalRequests)
	}
	if tornDownSnapshot.RuntimeControl == nil || tornDownSnapshot.RuntimeControl.Step.ID != "build" {
		t.Fatalf("tornDownSnapshot.RuntimeControl = %#v, want persisted build control", tornDownSnapshot.RuntimeControl)
	}

	if _, err := ag.ProcessDirect("ABORT job-1", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(ABORT) error = %v", err)
	}

	abortedSnapshot := readMissionStatusSnapshotFile(t, statusFile)
	if abortedSnapshot.Runtime == nil {
		t.Fatal("abortedSnapshot.Runtime = nil, want non-nil")
	}
	if abortedSnapshot.Runtime.State != missioncontrol.JobStateAborted {
		t.Fatalf("abortedSnapshot.Runtime.State = %q, want %q", abortedSnapshot.Runtime.State, missioncontrol.JobStateAborted)
	}
	if abortedSnapshot.Runtime.AbortedReason != missioncontrol.RuntimeAbortReasonOperatorCommand {
		t.Fatalf("abortedSnapshot.Runtime.AbortedReason = %q, want %q", abortedSnapshot.Runtime.AbortedReason, missioncontrol.RuntimeAbortReasonOperatorCommand)
	}
	if abortedSnapshot.Active {
		t.Fatal("abortedSnapshot.Active = true, want false")
	}
	if abortedSnapshot.Runtime.InspectablePlan == nil {
		t.Fatal("abortedSnapshot.Runtime.InspectablePlan = nil, want persisted inspectable plan")
	}
	if len(abortedSnapshot.Runtime.InspectablePlan.Steps) != 2 {
		t.Fatalf("len(abortedSnapshot.Runtime.InspectablePlan.Steps) = %d, want %d", len(abortedSnapshot.Runtime.InspectablePlan.Steps), 2)
	}
	if abortedSnapshot.RuntimeControl != nil {
		t.Fatalf("abortedSnapshot.RuntimeControl = %#v, want nil for terminal aborted snapshot", abortedSnapshot.RuntimeControl)
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

func TestMissionStatusBootstrapRejectsInconsistentPersistedRuntimeStepEnvelope(t *testing.T) {
	ag := newMissionBootstrapTestLoop()
	cmd := newMissionBootstrapTestCommand()
	job := testMissionBootstrapJob()
	missionFile := writeMissionBootstrapJobFile(t, job)
	statusFile := filepath.Join(t.TempDir(), "status.json")
	writeMissionStatusSnapshotFile(t, statusFile, missionStatusSnapshot{
		MissionFile: missionFile,
		JobID:       job.ID,
		StepID:      "final",
		Runtime: &missioncontrol.JobRuntimeState{
			JobID:        job.ID,
			State:        missioncontrol.JobStatePaused,
			ActiveStepID: "build",
			PausedReason: missioncontrol.RuntimePauseReasonOperatorCommand,
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
		t.Fatal("configureMissionBootstrap() error = nil, want persisted-runtime mismatch failure")
	}
	if !strings.Contains(err.Error(), `snapshot step_id "final" does not match runtime active_step_id "build"`) {
		t.Fatalf("configureMissionBootstrap() error = %q, want step envelope mismatch", err)
	}
}

func TestMissionStatusBootstrapRejectsInconsistentPersistedRuntimeControlStep(t *testing.T) {
	ag := newMissionBootstrapTestLoop()
	cmd := newMissionBootstrapTestCommand()
	job := testMissionBootstrapJob()
	missionFile := writeMissionBootstrapJobFile(t, job)
	statusFile := filepath.Join(t.TempDir(), "status.json")
	writeMissionStatusSnapshotFile(t, statusFile, missionStatusSnapshot{
		MissionFile: missionFile,
		JobID:       job.ID,
		StepID:      "build",
		RuntimeControl: &missioncontrol.RuntimeControlContext{
			JobID:        job.ID,
			MaxAuthority: job.MaxAuthority,
			AllowedTools: []string{"read"},
			Step: missioncontrol.Step{
				ID:   "final",
				Type: missioncontrol.StepTypeFinalResponse,
			},
		},
		Runtime: &missioncontrol.JobRuntimeState{
			JobID:        job.ID,
			State:        missioncontrol.JobStatePaused,
			ActiveStepID: "build",
			PausedReason: missioncontrol.RuntimePauseReasonOperatorCommand,
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
		t.Fatal("configureMissionBootstrap() error = nil, want persisted-control mismatch failure")
	}
	if !strings.Contains(err.Error(), `runtime control step_id "final" does not match runtime active_step_id "build"`) {
		t.Fatalf("configureMissionBootstrap() error = %q, want runtime-control mismatch", err)
	}
}

func TestMissionStatusBootstrapRejectsPersistedRuntimeWithActiveCompletedStepRecord(t *testing.T) {
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
			State:        missioncontrol.JobStatePaused,
			ActiveStepID: "build",
			PausedReason: missioncontrol.RuntimePauseReasonOperatorCommand,
			CompletedSteps: []missioncontrol.RuntimeStepRecord{
				{StepID: "build", At: time.Date(2026, 3, 27, 10, 0, 0, 0, time.UTC)},
			},
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
		t.Fatal("configureMissionBootstrap() error = nil, want completed-step replay marker failure")
	}
	if !strings.Contains(err.Error(), `active_step_id "build" is already recorded in completed_steps`) {
		t.Fatalf("configureMissionBootstrap() error = %q, want completed-step replay marker mismatch", err)
	}
}

func TestMissionStatusBootstrapRejectsPersistedRuntimeWithActiveFailedStepRecord(t *testing.T) {
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
			State:        missioncontrol.JobStatePaused,
			ActiveStepID: "build",
			PausedReason: missioncontrol.RuntimePauseReasonOperatorCommand,
			FailedSteps: []missioncontrol.RuntimeStepRecord{
				{StepID: "build", Reason: "validator failed", At: time.Date(2026, 3, 27, 10, 0, 0, 0, time.UTC)},
			},
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
		t.Fatal("configureMissionBootstrap() error = nil, want failed-step replay marker failure")
	}
	if !strings.Contains(err.Error(), `active_step_id "build" is already recorded in failed_steps`) {
		t.Fatalf("configureMissionBootstrap() error = %q, want failed-step replay marker mismatch", err)
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

func TestMissionStatusBootstrapApprovedResumeUsesPersistedRuntimeControlWhenMissionFileChanges(t *testing.T) {
	ag := newMissionBootstrapTestLoop()
	cmd := newMissionBootstrapTestCommand()
	job := testMissionBootstrapJob()
	missionFile := writeMissionBootstrapJobFile(t, job)
	statusFile := filepath.Join(t.TempDir(), "status.json")
	writeMissionStatusSnapshotFile(t, statusFile, missionStatusSnapshot{
		MissionFile:    missionFile,
		JobID:          job.ID,
		StepID:         "build",
		RuntimeControl: runtimeControlForBootstrapStep(t, job, "build"),
		Runtime: &missioncontrol.JobRuntimeState{
			JobID:        job.ID,
			State:        missioncontrol.JobStatePaused,
			ActiveStepID: "build",
			PausedReason: missioncontrol.RuntimePauseReasonOperatorCommand,
		},
	})
	writeMissionBootstrapJobFile(t, missioncontrol.Job{
		ID:           job.ID,
		MaxAuthority: missioncontrol.AuthorityTierLow,
		AllowedTools: []string{"shell"},
		Plan: missioncontrol.Plan{
			ID: job.Plan.ID,
			Steps: []missioncontrol.Step{
				{
					ID:   "final",
					Type: missioncontrol.StepTypeFinalResponse,
				},
			},
		},
	}, missionFile)

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
	if ec.Step == nil {
		t.Fatal("ActiveMissionStep().Step = nil, want non-nil")
	}
	if ec.Step.Type != missioncontrol.StepTypeOneShotCode {
		t.Fatalf("ActiveMissionStep().Step.Type = %q, want persisted %q", ec.Step.Type, missioncontrol.StepTypeOneShotCode)
	}
	if ec.Step.RequiredAuthority != missioncontrol.AuthorityTierLow {
		t.Fatalf("ActiveMissionStep().Step.RequiredAuthority = %q, want persisted %q", ec.Step.RequiredAuthority, missioncontrol.AuthorityTierLow)
	}
	if !reflect.DeepEqual(ec.Step.AllowedTools, []string{"read"}) {
		t.Fatalf("ActiveMissionStep().Step.AllowedTools = %#v, want persisted %#v", ec.Step.AllowedTools, []string{"read"})
	}
}

func TestMissionStatusBootstrapRehydratesPausedRuntimeControlAfterReboot(t *testing.T) {
	statusFile := filepath.Join(t.TempDir(), "status.json")
	cmd := newMissionBootstrapTestCommand()
	if err := cmd.Flags().Set("mission-status-file", statusFile); err != nil {
		t.Fatalf("Flags().Set(mission-status-file) error = %v", err)
	}

	job := testMissionBootstrapJob()
	missionFile := writeMissionBootstrapJobFile(t, job)
	writeMissionStatusSnapshotFile(t, statusFile, missionStatusSnapshot{
		MissionFile: missionFile,
		JobID:       job.ID,
		StepID:      "build",
		Runtime: &missioncontrol.JobRuntimeState{
			JobID:        job.ID,
			State:        missioncontrol.JobStatePaused,
			ActiveStepID: "build",
			PausedReason: missioncontrol.RuntimePauseReasonOperatorCommand,
		},
	})
	if err := cmd.Flags().Set("mission-file", missionFile); err != nil {
		t.Fatalf("Flags().Set(mission-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step", "build"); err != nil {
		t.Fatalf("Flags().Set(mission-step) error = %v", err)
	}

	hub := chat.NewHub(10)
	provider := &missionStatusFixedResponseProvider{content: "unused"}
	ag := agent.NewAgentLoop(hub, provider, provider.GetDefaultModel(), 3, "", nil)
	installMissionRuntimeChangeHook(cmd, ag)

	if err := configureMissionBootstrap(cmd, ag); err != nil {
		t.Fatalf("configureMissionBootstrap() error = %v", err)
	}
	if _, ok := ag.ActiveMissionStep(); ok {
		t.Fatal("ActiveMissionStep() ok = true, want false after control rehydration")
	}

	runtime, ok := ag.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if runtime.State != missioncontrol.JobStatePaused {
		t.Fatalf("MissionRuntimeState().State = %q, want %q", runtime.State, missioncontrol.JobStatePaused)
	}

	if _, err := ag.ProcessDirect("RESUME job-1", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(RESUME) error = %v", err)
	}

	ec, ok := ag.ActiveMissionStep()
	if !ok || ec.Runtime == nil {
		t.Fatalf("ActiveMissionStep() = %#v, want running runtime", ec)
	}
	if ec.Runtime.State != missioncontrol.JobStateRunning {
		t.Fatalf("ActiveMissionStep().Runtime.State = %q, want %q", ec.Runtime.State, missioncontrol.JobStateRunning)
	}

	snapshot := readMissionStatusSnapshotFile(t, statusFile)
	if snapshot.Runtime == nil || snapshot.Runtime.State != missioncontrol.JobStateRunning {
		t.Fatalf("snapshot.Runtime = %#v, want running runtime", snapshot.Runtime)
	}
	if !snapshot.Active {
		t.Fatal("snapshot.Active = false, want true after resume")
	}
}

func TestMissionStatusBootstrapRehydratesPausedRuntimeControlUsesFallbackWithoutPersistedControl(t *testing.T) {
	statusFile := filepath.Join(t.TempDir(), "status.json")
	cmd := newMissionBootstrapTestCommand()
	if err := cmd.Flags().Set("mission-status-file", statusFile); err != nil {
		t.Fatalf("Flags().Set(mission-status-file) error = %v", err)
	}

	job := testMissionBootstrapJob()
	missionFile := writeMissionBootstrapJobFile(t, job)
	writeMissionStatusSnapshotFile(t, statusFile, missionStatusSnapshot{
		MissionFile: missionFile,
		JobID:       job.ID,
		StepID:      "build",
		Runtime: &missioncontrol.JobRuntimeState{
			JobID:        job.ID,
			State:        missioncontrol.JobStatePaused,
			ActiveStepID: "build",
			PausedReason: missioncontrol.RuntimePauseReasonOperatorCommand,
		},
	})
	if err := cmd.Flags().Set("mission-file", missionFile); err != nil {
		t.Fatalf("Flags().Set(mission-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step", "build"); err != nil {
		t.Fatalf("Flags().Set(mission-step) error = %v", err)
	}

	ag := newMissionBootstrapTestLoop()
	if err := configureMissionBootstrap(cmd, ag); err != nil {
		t.Fatalf("configureMissionBootstrap() error = %v", err)
	}
	if _, ok := ag.ActiveMissionStep(); ok {
		t.Fatal("ActiveMissionStep() ok = true, want false after control rehydration")
	}

	if _, err := ag.ProcessDirect("RESUME job-1", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(RESUME) error = %v", err)
	}

	ec, ok := ag.ActiveMissionStep()
	if !ok || ec.Step == nil {
		t.Fatalf("ActiveMissionStep() = %#v, want resumed active step", ec)
	}
	if ec.Step.Type != missioncontrol.StepTypeOneShotCode {
		t.Fatalf("ActiveMissionStep().Step.Type = %q, want %q", ec.Step.Type, missioncontrol.StepTypeOneShotCode)
	}
}

func TestMissionStatusBootstrapRehydratedApproveFromWaitingUserUsesPersistedRuntimeControlWhenMissionFileChanges(t *testing.T) {
	statusFile := filepath.Join(t.TempDir(), "status.json")
	cmd := newMissionBootstrapTestCommand()
	if err := cmd.Flags().Set("mission-status-file", statusFile); err != nil {
		t.Fatalf("Flags().Set(mission-status-file) error = %v", err)
	}

	job := missioncontrol.Job{
		ID:           "job-1",
		MaxAuthority: missioncontrol.AuthorityTierHigh,
		AllowedTools: []string{"read"},
		Plan: missioncontrol.Plan{
			ID: "plan-1",
			Steps: []missioncontrol.Step{
				{
					ID:      "build",
					Type:    missioncontrol.StepTypeDiscussion,
					Subtype: missioncontrol.StepSubtypeAuthorization,
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
	writeMissionStatusSnapshotFile(t, statusFile, missionStatusSnapshot{
		MissionFile:    missionFile,
		JobID:          job.ID,
		StepID:         "build",
		RuntimeControl: runtimeControlForBootstrapStep(t, job, "build"),
		Runtime: &missioncontrol.JobRuntimeState{
			JobID:         job.ID,
			State:         missioncontrol.JobStateWaitingUser,
			ActiveStepID:  "build",
			WaitingReason: "awaiting operator input",
			ApprovalRequests: []missioncontrol.ApprovalRequest{
				{
					JobID:           job.ID,
					StepID:          "build",
					RequestedAction: missioncontrol.ApprovalRequestedActionStepComplete,
					Scope:           missioncontrol.ApprovalScopeMissionStep,
					State:           missioncontrol.ApprovalStatePending,
				},
			},
		},
	})
	writeMissionBootstrapJobFile(t, missioncontrol.Job{
		ID:           job.ID,
		MaxAuthority: missioncontrol.AuthorityTierLow,
		AllowedTools: []string{"shell"},
		Plan: missioncontrol.Plan{
			ID: job.Plan.ID,
			Steps: []missioncontrol.Step{
				{
					ID:   "final",
					Type: missioncontrol.StepTypeFinalResponse,
				},
			},
		},
	}, missionFile)

	if err := cmd.Flags().Set("mission-file", missionFile); err != nil {
		t.Fatalf("Flags().Set(mission-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step", "build"); err != nil {
		t.Fatalf("Flags().Set(mission-step) error = %v", err)
	}

	ag := newMissionBootstrapTestLoop()
	if err := configureMissionBootstrap(cmd, ag); err != nil {
		t.Fatalf("configureMissionBootstrap() error = %v", err)
	}
	if _, ok := ag.ActiveMissionStep(); ok {
		t.Fatal("ActiveMissionStep() ok = true, want false after waiting_user rehydration")
	}

	if _, err := ag.ProcessDirect("APPROVE job-1 build", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(APPROVE) error = %v", err)
	}

	runtime, ok := ag.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if runtime.State != missioncontrol.JobStatePaused {
		t.Fatalf("MissionRuntimeState().State = %q, want %q", runtime.State, missioncontrol.JobStatePaused)
	}
	if len(runtime.CompletedSteps) != 1 || runtime.CompletedSteps[0].StepID != "build" {
		t.Fatalf("MissionRuntimeState().CompletedSteps = %#v, want build completion", runtime.CompletedSteps)
	}
	if len(runtime.ApprovalGrants) != 1 || runtime.ApprovalGrants[0].State != missioncontrol.ApprovalStateGranted {
		t.Fatalf("MissionRuntimeState().ApprovalGrants = %#v, want one granted approval", runtime.ApprovalGrants)
	}
}

func TestMissionStatusBootstrapRehydratedApproveUsesFallbackWithoutPersistedControl(t *testing.T) {
	statusFile := filepath.Join(t.TempDir(), "status.json")
	cmd := newMissionBootstrapTestCommand()
	if err := cmd.Flags().Set("mission-status-file", statusFile); err != nil {
		t.Fatalf("Flags().Set(mission-status-file) error = %v", err)
	}

	job := missioncontrol.Job{
		ID:           "job-1",
		MaxAuthority: missioncontrol.AuthorityTierHigh,
		AllowedTools: []string{"read"},
		Plan: missioncontrol.Plan{
			ID: "plan-1",
			Steps: []missioncontrol.Step{
				{
					ID:      "build",
					Type:    missioncontrol.StepTypeDiscussion,
					Subtype: missioncontrol.StepSubtypeAuthorization,
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
	writeMissionStatusSnapshotFile(t, statusFile, missionStatusSnapshot{
		MissionFile: missionFile,
		JobID:       job.ID,
		StepID:      "build",
		Runtime: &missioncontrol.JobRuntimeState{
			JobID:         job.ID,
			State:         missioncontrol.JobStateWaitingUser,
			ActiveStepID:  "build",
			WaitingReason: "awaiting operator input",
			ApprovalRequests: []missioncontrol.ApprovalRequest{
				{
					JobID:           job.ID,
					StepID:          "build",
					RequestedAction: missioncontrol.ApprovalRequestedActionStepComplete,
					Scope:           missioncontrol.ApprovalScopeMissionStep,
					State:           missioncontrol.ApprovalStatePending,
				},
			},
		},
	})

	if err := cmd.Flags().Set("mission-file", missionFile); err != nil {
		t.Fatalf("Flags().Set(mission-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step", "build"); err != nil {
		t.Fatalf("Flags().Set(mission-step) error = %v", err)
	}

	ag := newMissionBootstrapTestLoop()
	if err := configureMissionBootstrap(cmd, ag); err != nil {
		t.Fatalf("configureMissionBootstrap() error = %v", err)
	}

	if _, err := ag.ProcessDirect("APPROVE job-1 build", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(APPROVE) error = %v", err)
	}

	runtime, ok := ag.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if runtime.State != missioncontrol.JobStatePaused {
		t.Fatalf("MissionRuntimeState().State = %q, want %q", runtime.State, missioncontrol.JobStatePaused)
	}
}

func TestMissionStatusBootstrapRehydratedDenyBlocksLaterFreeFormCompletion(t *testing.T) {
	statusFile := filepath.Join(t.TempDir(), "status.json")
	cmd := newMissionBootstrapTestCommand()
	if err := cmd.Flags().Set("mission-status-file", statusFile); err != nil {
		t.Fatalf("Flags().Set(mission-status-file) error = %v", err)
	}

	job := missioncontrol.Job{
		ID:           "job-1",
		MaxAuthority: missioncontrol.AuthorityTierHigh,
		AllowedTools: []string{"read"},
		Plan: missioncontrol.Plan{
			ID: "plan-1",
			Steps: []missioncontrol.Step{
				{
					ID:      "build",
					Type:    missioncontrol.StepTypeDiscussion,
					Subtype: missioncontrol.StepSubtypeAuthorization,
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
	writeMissionStatusSnapshotFile(t, statusFile, missionStatusSnapshot{
		MissionFile: missionFile,
		JobID:       job.ID,
		StepID:      "build",
		Runtime: &missioncontrol.JobRuntimeState{
			JobID:         job.ID,
			State:         missioncontrol.JobStateWaitingUser,
			ActiveStepID:  "build",
			WaitingReason: "awaiting operator input",
			ApprovalRequests: []missioncontrol.ApprovalRequest{
				{
					JobID:           job.ID,
					StepID:          "build",
					RequestedAction: missioncontrol.ApprovalRequestedActionStepComplete,
					Scope:           missioncontrol.ApprovalScopeMissionStep,
					State:           missioncontrol.ApprovalStatePending,
				},
			},
		},
	})
	if err := cmd.Flags().Set("mission-file", missionFile); err != nil {
		t.Fatalf("Flags().Set(mission-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step", "build"); err != nil {
		t.Fatalf("Flags().Set(mission-step) error = %v", err)
	}

	ag := newMissionBootstrapTestLoop()
	if err := configureMissionBootstrap(cmd, ag); err != nil {
		t.Fatalf("configureMissionBootstrap() error = %v", err)
	}

	if _, err := ag.ProcessDirect("DENY job-1 build", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(DENY) error = %v", err)
	}
	resp, err := ag.ProcessDirect("approved", 2*time.Second)
	if err != nil {
		t.Fatalf("ProcessDirect(approved) error = %v", err)
	}
	if !strings.Contains(resp, "(stub) Echo") {
		t.Fatalf("ProcessDirect(approved) response = %q, want stub provider fallback after denied reboot-safe path", resp)
	}

	runtime, ok := ag.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if runtime.State != missioncontrol.JobStateWaitingUser {
		t.Fatalf("MissionRuntimeState().State = %q, want %q", runtime.State, missioncontrol.JobStateWaitingUser)
	}
	if len(runtime.CompletedSteps) != 0 {
		t.Fatalf("MissionRuntimeState().CompletedSteps = %#v, want empty", runtime.CompletedSteps)
	}
	if len(runtime.ApprovalRequests) != 1 || runtime.ApprovalRequests[0].State != missioncontrol.ApprovalStateDenied {
		t.Fatalf("MissionRuntimeState().ApprovalRequests = %#v, want one denied approval", runtime.ApprovalRequests)
	}
}

func TestMissionStatusBootstrapRehydratedWrongJobDoesNotBind(t *testing.T) {
	statusFile := filepath.Join(t.TempDir(), "status.json")
	cmd := newMissionBootstrapTestCommand()
	if err := cmd.Flags().Set("mission-status-file", statusFile); err != nil {
		t.Fatalf("Flags().Set(mission-status-file) error = %v", err)
	}

	job := testMissionBootstrapJob()
	missionFile := writeMissionBootstrapJobFile(t, job)
	writeMissionStatusSnapshotFile(t, statusFile, missionStatusSnapshot{
		MissionFile: missionFile,
		JobID:       job.ID,
		StepID:      "build",
		Runtime: &missioncontrol.JobRuntimeState{
			JobID:        job.ID,
			State:        missioncontrol.JobStatePaused,
			ActiveStepID: "build",
			PausedReason: missioncontrol.RuntimePauseReasonOperatorCommand,
		},
	})
	if err := cmd.Flags().Set("mission-file", missionFile); err != nil {
		t.Fatalf("Flags().Set(mission-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step", "build"); err != nil {
		t.Fatalf("Flags().Set(mission-step) error = %v", err)
	}

	ag := newMissionBootstrapTestLoop()
	if err := configureMissionBootstrap(cmd, ag); err != nil {
		t.Fatalf("configureMissionBootstrap() error = %v", err)
	}

	_, err := ag.ProcessDirect("RESUME other-job", 2*time.Second)
	if err == nil {
		t.Fatal("ProcessDirect(RESUME wrong job) error = nil, want mismatch failure")
	}
	if !strings.Contains(err.Error(), "does not match the active job") {
		t.Fatalf("ProcessDirect(RESUME wrong job) error = %q, want job mismatch", err)
	}
	if _, ok := ag.ActiveMissionStep(); ok {
		t.Fatal("ActiveMissionStep() ok = true, want false after wrong-job rejection")
	}
}

func TestMissionStatusBootstrapNormalizesLegacyRevokedApprovalRequestAndPersistsSnapshot(t *testing.T) {
	statusFile := filepath.Join(t.TempDir(), "status.json")
	cmd := newMissionBootstrapTestCommand()
	if err := cmd.Flags().Set("mission-status-file", statusFile); err != nil {
		t.Fatalf("Flags().Set(mission-status-file) error = %v", err)
	}

	job := missioncontrol.Job{
		ID:           "job-1",
		MaxAuthority: missioncontrol.AuthorityTierHigh,
		AllowedTools: []string{"read"},
		Plan: missioncontrol.Plan{
			ID: "plan-1",
			Steps: []missioncontrol.Step{
				{
					ID:                "build",
					Type:              missioncontrol.StepTypeDiscussion,
					Subtype:           missioncontrol.StepSubtypeAuthorization,
					ApprovalScope:     missioncontrol.ApprovalScopeOneJob,
					AllowedTools:      []string{"read"},
					RequiredAuthority: missioncontrol.AuthorityTierLow,
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
	revokedAt := time.Date(2026, 3, 24, 12, 1, 0, 0, time.UTC)
	writeMissionStatusSnapshotFile(t, statusFile, missionStatusSnapshot{
		MissionFile:    missionFile,
		JobID:          job.ID,
		StepID:         "build",
		RuntimeControl: runtimeControlForBootstrapStep(t, job, "build"),
		Runtime: &missioncontrol.JobRuntimeState{
			JobID:        job.ID,
			State:        missioncontrol.JobStatePaused,
			ActiveStepID: "build",
			PausedReason: missioncontrol.RuntimePauseReasonOperatorCommand,
			ApprovalRequests: []missioncontrol.ApprovalRequest{
				{
					JobID:           job.ID,
					StepID:          "build",
					RequestedAction: missioncontrol.ApprovalRequestedActionStepComplete,
					Scope:           missioncontrol.ApprovalScopeOneJob,
					RequestedVia:    missioncontrol.ApprovalRequestedViaRuntime,
					GrantedVia:      missioncontrol.ApprovalGrantedViaOperatorCommand,
					State:           missioncontrol.ApprovalStateRevoked,
				},
			},
			ApprovalGrants: []missioncontrol.ApprovalGrant{
				{
					JobID:           job.ID,
					StepID:          "build",
					RequestedAction: missioncontrol.ApprovalRequestedActionStepComplete,
					Scope:           missioncontrol.ApprovalScopeOneJob,
					GrantedVia:      missioncontrol.ApprovalGrantedViaOperatorCommand,
					State:           missioncontrol.ApprovalStateRevoked,
					RevokedAt:       revokedAt,
				},
			},
		},
	})
	if err := cmd.Flags().Set("mission-file", missionFile); err != nil {
		t.Fatalf("Flags().Set(mission-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step", "build"); err != nil {
		t.Fatalf("Flags().Set(mission-step) error = %v", err)
	}

	ag := newMissionBootstrapTestLoop()
	installMissionRuntimeChangeHook(cmd, ag)
	if err := configureMissionBootstrap(cmd, ag); err != nil {
		t.Fatalf("configureMissionBootstrap() error = %v", err)
	}

	runtime, ok := ag.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if runtime.ApprovalRequests[0].RevokedAt != revokedAt {
		t.Fatalf("MissionRuntimeState().ApprovalRequests[0].RevokedAt = %v, want %v", runtime.ApprovalRequests[0].RevokedAt, revokedAt)
	}

	snapshot := readMissionStatusSnapshotFile(t, statusFile)
	if snapshot.Runtime == nil {
		t.Fatal("snapshot.Runtime = nil, want persisted runtime")
	}
	if snapshot.Runtime.ApprovalRequests[0].RevokedAt != revokedAt {
		t.Fatalf("snapshot.Runtime.ApprovalRequests[0].RevokedAt = %v, want %v", snapshot.Runtime.ApprovalRequests[0].RevokedAt, revokedAt)
	}
}

func TestMissionStatusBootstrapRehydratedWrongStepDoesNotBind(t *testing.T) {
	statusFile := filepath.Join(t.TempDir(), "status.json")
	cmd := newMissionBootstrapTestCommand()
	if err := cmd.Flags().Set("mission-status-file", statusFile); err != nil {
		t.Fatalf("Flags().Set(mission-status-file) error = %v", err)
	}

	job := missioncontrol.Job{
		ID:           "job-1",
		MaxAuthority: missioncontrol.AuthorityTierHigh,
		AllowedTools: []string{"read"},
		Plan: missioncontrol.Plan{
			ID: "plan-1",
			Steps: []missioncontrol.Step{
				{
					ID:      "build",
					Type:    missioncontrol.StepTypeDiscussion,
					Subtype: missioncontrol.StepSubtypeAuthorization,
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
	writeMissionStatusSnapshotFile(t, statusFile, missionStatusSnapshot{
		MissionFile: missionFile,
		JobID:       job.ID,
		StepID:      "build",
		Runtime: &missioncontrol.JobRuntimeState{
			JobID:         job.ID,
			State:         missioncontrol.JobStateWaitingUser,
			ActiveStepID:  "build",
			WaitingReason: "awaiting operator input",
			ApprovalRequests: []missioncontrol.ApprovalRequest{
				{
					JobID:           job.ID,
					StepID:          "build",
					RequestedAction: missioncontrol.ApprovalRequestedActionStepComplete,
					Scope:           missioncontrol.ApprovalScopeMissionStep,
					State:           missioncontrol.ApprovalStatePending,
				},
			},
		},
	})
	if err := cmd.Flags().Set("mission-file", missionFile); err != nil {
		t.Fatalf("Flags().Set(mission-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step", "build"); err != nil {
		t.Fatalf("Flags().Set(mission-step) error = %v", err)
	}

	ag := newMissionBootstrapTestLoop()
	if err := configureMissionBootstrap(cmd, ag); err != nil {
		t.Fatalf("configureMissionBootstrap() error = %v", err)
	}

	_, err := ag.ProcessDirect("APPROVE job-1 other-step", 2*time.Second)
	if err == nil {
		t.Fatal("ProcessDirect(APPROVE wrong step) error = nil, want mismatch failure")
	}
	if !strings.Contains(err.Error(), "does not match the active job and step") {
		t.Fatalf("ProcessDirect(APPROVE wrong step) error = %q, want step mismatch", err)
	}

	runtime, ok := ag.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if runtime.State != missioncontrol.JobStateWaitingUser {
		t.Fatalf("MissionRuntimeState().State = %q, want %q", runtime.State, missioncontrol.JobStateWaitingUser)
	}
	if len(runtime.ApprovalRequests) != 1 || runtime.ApprovalRequests[0].State != missioncontrol.ApprovalStatePending {
		t.Fatalf("MissionRuntimeState().ApprovalRequests = %#v, want one pending approval", runtime.ApprovalRequests)
	}
}

func TestMissionStatusBootstrapRehydratedAbortUsesPersistedRuntimeControlWhenMissionFileChanges(t *testing.T) {
	statusFile := filepath.Join(t.TempDir(), "status.json")
	cmd := newMissionBootstrapTestCommand()
	if err := cmd.Flags().Set("mission-status-file", statusFile); err != nil {
		t.Fatalf("Flags().Set(mission-status-file) error = %v", err)
	}

	job := missioncontrol.Job{
		ID:           "job-1",
		MaxAuthority: missioncontrol.AuthorityTierHigh,
		AllowedTools: []string{"read"},
		Plan: missioncontrol.Plan{
			ID: "plan-1",
			Steps: []missioncontrol.Step{
				{
					ID:      "build",
					Type:    missioncontrol.StepTypeDiscussion,
					Subtype: missioncontrol.StepSubtypeAuthorization,
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
	writeMissionStatusSnapshotFile(t, statusFile, missionStatusSnapshot{
		MissionFile:    missionFile,
		JobID:          job.ID,
		StepID:         "build",
		RuntimeControl: runtimeControlForBootstrapStep(t, job, "build"),
		Runtime: &missioncontrol.JobRuntimeState{
			JobID:         job.ID,
			State:         missioncontrol.JobStateWaitingUser,
			ActiveStepID:  "build",
			WaitingReason: "awaiting operator input",
			ApprovalRequests: []missioncontrol.ApprovalRequest{
				{
					JobID:           job.ID,
					StepID:          "build",
					RequestedAction: missioncontrol.ApprovalRequestedActionStepComplete,
					Scope:           missioncontrol.ApprovalScopeMissionStep,
					State:           missioncontrol.ApprovalStatePending,
				},
			},
		},
	})
	writeMissionBootstrapJobFile(t, missioncontrol.Job{
		ID:           job.ID,
		MaxAuthority: missioncontrol.AuthorityTierLow,
		AllowedTools: []string{"shell"},
		Plan: missioncontrol.Plan{
			ID: job.Plan.ID,
			Steps: []missioncontrol.Step{
				{
					ID:   "final",
					Type: missioncontrol.StepTypeFinalResponse,
				},
			},
		},
	}, missionFile)

	if err := cmd.Flags().Set("mission-file", missionFile); err != nil {
		t.Fatalf("Flags().Set(mission-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step", "build"); err != nil {
		t.Fatalf("Flags().Set(mission-step) error = %v", err)
	}

	ag := newMissionBootstrapTestLoop()
	if err := configureMissionBootstrap(cmd, ag); err != nil {
		t.Fatalf("configureMissionBootstrap() error = %v", err)
	}
	if _, ok := ag.ActiveMissionStep(); ok {
		t.Fatal("ActiveMissionStep() ok = true, want false after waiting_user rehydration")
	}

	if _, err := ag.ProcessDirect("ABORT job-1", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(ABORT) error = %v", err)
	}

	runtime, ok := ag.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if runtime.State != missioncontrol.JobStateAborted {
		t.Fatalf("MissionRuntimeState().State = %q, want %q", runtime.State, missioncontrol.JobStateAborted)
	}
}

func TestMissionStatusBootstrapRehydratedAbortFromWaitingUserPersistsLifecycle(t *testing.T) {
	statusFile := filepath.Join(t.TempDir(), "status.json")
	cmd := newMissionBootstrapTestCommand()
	if err := cmd.Flags().Set("mission-status-file", statusFile); err != nil {
		t.Fatalf("Flags().Set(mission-status-file) error = %v", err)
	}

	job := missioncontrol.Job{
		ID:           "job-1",
		MaxAuthority: missioncontrol.AuthorityTierHigh,
		AllowedTools: []string{"read"},
		Plan: missioncontrol.Plan{
			ID: "plan-1",
			Steps: []missioncontrol.Step{
				{
					ID:      "build",
					Type:    missioncontrol.StepTypeDiscussion,
					Subtype: missioncontrol.StepSubtypeAuthorization,
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
	writeMissionStatusSnapshotFile(t, statusFile, missionStatusSnapshot{
		MissionFile: missionFile,
		JobID:       job.ID,
		StepID:      "build",
		Runtime: &missioncontrol.JobRuntimeState{
			JobID:         job.ID,
			State:         missioncontrol.JobStateWaitingUser,
			ActiveStepID:  "build",
			WaitingReason: "awaiting operator input",
			ApprovalRequests: []missioncontrol.ApprovalRequest{
				{
					JobID:           job.ID,
					StepID:          "build",
					RequestedAction: missioncontrol.ApprovalRequestedActionStepComplete,
					Scope:           missioncontrol.ApprovalScopeMissionStep,
					State:           missioncontrol.ApprovalStatePending,
				},
			},
		},
	})
	if err := cmd.Flags().Set("mission-file", missionFile); err != nil {
		t.Fatalf("Flags().Set(mission-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step", "build"); err != nil {
		t.Fatalf("Flags().Set(mission-step) error = %v", err)
	}

	hub := chat.NewHub(10)
	provider := &missionStatusFixedResponseProvider{content: "unused"}
	ag := agent.NewAgentLoop(hub, provider, provider.GetDefaultModel(), 3, "", nil)
	installMissionRuntimeChangeHook(cmd, ag)

	if err := configureMissionBootstrap(cmd, ag); err != nil {
		t.Fatalf("configureMissionBootstrap() error = %v", err)
	}
	if _, ok := ag.ActiveMissionStep(); ok {
		t.Fatal("ActiveMissionStep() ok = true, want false after waiting_user rehydration")
	}

	if _, err := ag.ProcessDirect("ABORT job-1", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(ABORT) error = %v", err)
	}

	snapshot := readMissionStatusSnapshotFile(t, statusFile)
	if snapshot.Runtime == nil || snapshot.Runtime.State != missioncontrol.JobStateAborted {
		t.Fatalf("snapshot.Runtime = %#v, want aborted runtime", snapshot.Runtime)
	}
	if snapshot.Active {
		t.Fatal("snapshot.Active = true, want false after abort")
	}
}

func TestMissionStatusBootstrapRehydratedTerminalRuntimeRejectsOperatorControl(t *testing.T) {
	for _, runtimeState := range []missioncontrol.JobState{
		missioncontrol.JobStateCompleted,
		missioncontrol.JobStateFailed,
		missioncontrol.JobStateAborted,
	} {
		runtimeState := runtimeState
		t.Run(string(runtimeState), func(t *testing.T) {
			ag := newMissionBootstrapTestLoop()
			cmd := newMissionBootstrapTestCommand()
			job := testMissionBootstrapJob()
			missionFile := writeMissionBootstrapJobFile(t, job)
			statusFile := filepath.Join(t.TempDir(), "status.json")
			writeMissionStatusSnapshotFile(t, statusFile, missionStatusSnapshot{
				MissionFile: missionFile,
				JobID:       job.ID,
				Runtime: &missioncontrol.JobRuntimeState{
					JobID: job.ID,
					State: runtimeState,
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

			if err := configureMissionBootstrap(cmd, ag); err != nil {
				t.Fatalf("configureMissionBootstrap() error = %v", err)
			}
			if _, ok := ag.ActiveMissionStep(); ok {
				t.Fatal("ActiveMissionStep() ok = true, want false for terminal rehydration")
			}

			for _, command := range []string{"RESUME job-1", "ABORT job-1"} {
				_, err := ag.ProcessDirect(command, 2*time.Second)
				if err == nil {
					t.Fatalf("ProcessDirect(%s) error = nil, want invalid runtime state", command)
				}
				wantCode := "E_STEP_OUT_OF_ORDER"
				if runtimeState == missioncontrol.JobStateAborted {
					wantCode = "E_ABORTED"
				}
				if !strings.Contains(err.Error(), wantCode) {
					t.Fatalf("ProcessDirect(%s) error = %q, want canonical rejection code %q", command, err, wantCode)
				}
			}
		})
	}
}

func TestMissionStatusBootstrapRehydratedTerminalRuntimeRejectsApprovalDecisions(t *testing.T) {
	for _, runtimeState := range []missioncontrol.JobState{
		missioncontrol.JobStateCompleted,
		missioncontrol.JobStateFailed,
		missioncontrol.JobStateAborted,
	} {
		runtimeState := runtimeState
		t.Run(string(runtimeState), func(t *testing.T) {
			ag := newMissionBootstrapTestLoop()
			cmd := newMissionBootstrapTestCommand()
			job := testMissionBootstrapJob()
			missionFile := writeMissionBootstrapJobFile(t, job)
			statusFile := filepath.Join(t.TempDir(), "status.json")
			writeMissionStatusSnapshotFile(t, statusFile, missionStatusSnapshot{
				MissionFile: missionFile,
				JobID:       job.ID,
				Runtime: &missioncontrol.JobRuntimeState{
					JobID: job.ID,
					State: runtimeState,
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

			if err := configureMissionBootstrap(cmd, ag); err != nil {
				t.Fatalf("configureMissionBootstrap() error = %v", err)
			}

			_, err := ag.ProcessDirect("APPROVE job-1 build", 2*time.Second)
			if err == nil {
				t.Fatal("ProcessDirect(APPROVE) error = nil, want terminal-state rejection")
			}
			if !strings.Contains(err.Error(), string(runtimeState)) {
				t.Fatalf("ProcessDirect(APPROVE) error = %q, want state-specific rejection", err)
			}
		})
	}
}

func TestMissionStatusBootstrapRehydratedTerminalRuntimeRejectsNaturalApprovalDecisions(t *testing.T) {
	for _, runtimeState := range []missioncontrol.JobState{
		missioncontrol.JobStateCompleted,
		missioncontrol.JobStateFailed,
		missioncontrol.JobStateAborted,
	} {
		runtimeState := runtimeState
		t.Run(string(runtimeState), func(t *testing.T) {
			ag := newMissionBootstrapTestLoop()
			cmd := newMissionBootstrapTestCommand()
			job := testMissionBootstrapJob()
			missionFile := writeMissionBootstrapJobFile(t, job)
			statusFile := filepath.Join(t.TempDir(), "status.json")
			writeMissionStatusSnapshotFile(t, statusFile, missionStatusSnapshot{
				MissionFile: missionFile,
				JobID:       job.ID,
				Runtime: &missioncontrol.JobRuntimeState{
					JobID: job.ID,
					State: runtimeState,
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

			if err := configureMissionBootstrap(cmd, ag); err != nil {
				t.Fatalf("configureMissionBootstrap() error = %v", err)
			}

			_, err := ag.ProcessDirect("yes", 2*time.Second)
			if err == nil {
				t.Fatal("ProcessDirect(yes) error = nil, want terminal-state rejection")
			}
			if !strings.Contains(err.Error(), string(runtimeState)) {
				t.Fatalf("ProcessDirect(yes) error = %q, want state-specific rejection", err)
			}
		})
	}
}

func newMissionBootstrapTestCommand() *cobra.Command {
	cmd := &cobra.Command{Use: "test"}
	addMissionBootstrapFlags(cmd)
	cmd.Flags().String("mission-step-control-file", "", "")
	return cmd
}

func TestMissionStatusBootstrapUsesCommittedDurableRuntimeWhenPresent(t *testing.T) {
	statusFile := filepath.Join(t.TempDir(), "status.json")
	cmd := newMissionBootstrapTestCommand()
	job := testMissionBootstrapJob()
	missionFile := writeMissionBootstrapJobFile(t, job)
	storeRoot := missioncontrol.ResolveStoreRoot("", statusFile)
	now := time.Date(2026, 4, 5, 12, 0, 0, 0, time.UTC)

	writeCommittedMissionBootstrapJobRuntimeRecord(t, storeRoot, job.ID, 7, 1, now, missioncontrol.JobStatePaused, "build")
	writeCommittedMissionBootstrapRuntimeControlRecord(t, storeRoot, 7, 1, runtimeControlForBootstrapStep(t, job, "build"))

	if err := cmd.Flags().Set("mission-file", missionFile); err != nil {
		t.Fatalf("Flags().Set(mission-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step", "build"); err != nil {
		t.Fatalf("Flags().Set(mission-step) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-status-file", statusFile); err != nil {
		t.Fatalf("Flags().Set(mission-status-file) error = %v", err)
	}

	ag := newMissionBootstrapTestLoop()
	if err := configureMissionBootstrap(cmd, ag); err != nil {
		t.Fatalf("configureMissionBootstrap() error = %v", err)
	}
	if _, ok := ag.ActiveMissionStep(); ok {
		t.Fatal("ActiveMissionStep() ok = true, want false after durable paused-runtime rehydration")
	}

	runtime, ok := ag.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if runtime.State != missioncontrol.JobStatePaused {
		t.Fatalf("MissionRuntimeState().State = %q, want %q", runtime.State, missioncontrol.JobStatePaused)
	}
	if runtime.ActiveStepID != "build" {
		t.Fatalf("MissionRuntimeState().ActiveStepID = %q, want %q", runtime.ActiveStepID, "build")
	}
}

func TestMissionStatusBootstrapFallsBackToSnapshotWhenDurableStoreAbsent(t *testing.T) {
	ag := newMissionBootstrapTestLoop()
	cmd := newMissionBootstrapTestCommand()
	job := testMissionBootstrapJob()
	missionFile := writeMissionBootstrapJobFile(t, job)
	statusFile := filepath.Join(t.TempDir(), "status.json")

	writeMissionStatusSnapshotFile(t, statusFile, missionStatusSnapshot{
		MissionFile:    missionFile,
		JobID:          job.ID,
		StepID:         "build",
		RuntimeControl: runtimeControlForBootstrapStep(t, job, "build"),
		Runtime: &missioncontrol.JobRuntimeState{
			JobID:        job.ID,
			State:        missioncontrol.JobStatePaused,
			ActiveStepID: "build",
			PausedReason: missioncontrol.RuntimePauseReasonOperatorCommand,
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

	if err := configureMissionBootstrap(cmd, ag); err != nil {
		t.Fatalf("configureMissionBootstrap() error = %v", err)
	}

	runtime, ok := ag.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if runtime.State != missioncontrol.JobStatePaused {
		t.Fatalf("MissionRuntimeState().State = %q, want %q", runtime.State, missioncontrol.JobStatePaused)
	}
	if runtime.ActiveStepID != "build" {
		t.Fatalf("MissionRuntimeState().ActiveStepID = %q, want %q", runtime.ActiveStepID, "build")
	}
}

func TestLoadPersistedMissionRuntimeUsesSnapshotWhenStoreRootUnconfigured(t *testing.T) {
	job := testMissionBootstrapJob()
	statusFile := filepath.Join(t.TempDir(), "status.json")
	writeMissionStatusSnapshotFile(t, statusFile, missionStatusSnapshot{
		JobID:  job.ID,
		StepID: "build",
		Runtime: &missioncontrol.JobRuntimeState{
			JobID:        job.ID,
			State:        missioncontrol.JobStatePaused,
			ActiveStepID: "build",
		},
		RuntimeControl: runtimeControlForBootstrapStep(t, job, "build"),
	})

	runtime, control, source, ok, err := loadPersistedMissionRuntime(statusFile, "", job, time.Now().UTC())
	if err != nil {
		t.Fatalf("loadPersistedMissionRuntime() error = %v", err)
	}
	if !ok {
		t.Fatal("loadPersistedMissionRuntime() ok = false, want true")
	}
	if source != statusFile {
		t.Fatalf("loadPersistedMissionRuntime() source = %q, want %q", source, statusFile)
	}
	if runtime.State != missioncontrol.JobStatePaused || runtime.ActiveStepID != "build" {
		t.Fatalf("loadPersistedMissionRuntime() runtime = %#v, want paused build runtime", runtime)
	}
	if control == nil || control.Step.ID != "build" {
		t.Fatalf("loadPersistedMissionRuntime() control = %#v, want build control", control)
	}
}

func TestLoadPersistedMissionRuntimeSnapshotUsesSharedLegacyHelper(t *testing.T) {
	job := testMissionBootstrapJob()
	statusFile := filepath.Join(t.TempDir(), "status.json")

	original := loadValidatedLegacyMissionStatusSnapshot
	t.Cleanup(func() { loadValidatedLegacyMissionStatusSnapshot = original })
	called := 0
	loadValidatedLegacyMissionStatusSnapshot = func(path string, jobID string) (missioncontrol.MissionStatusSnapshot, bool, error) {
		called++
		if path != statusFile {
			t.Fatalf("legacy helper path = %q, want %q", path, statusFile)
		}
		if jobID != job.ID {
			t.Fatalf("legacy helper jobID = %q, want %q", jobID, job.ID)
		}
		return missioncontrol.MissionStatusSnapshot{
			JobID:  job.ID,
			StepID: "build",
			Runtime: &missioncontrol.JobRuntimeState{
				JobID:        job.ID,
				State:        missioncontrol.JobStatePaused,
				ActiveStepID: "build",
			},
			RuntimeControl: runtimeControlForBootstrapStep(t, job, "build"),
		}, true, nil
	}

	runtime, control, ok, err := loadPersistedMissionRuntimeSnapshot(statusFile, job)
	if err != nil {
		t.Fatalf("loadPersistedMissionRuntimeSnapshot() error = %v", err)
	}
	if !ok {
		t.Fatal("loadPersistedMissionRuntimeSnapshot() ok = false, want true")
	}
	if called != 1 {
		t.Fatalf("legacy helper calls = %d, want 1", called)
	}
	if runtime.State != missioncontrol.JobStatePaused || runtime.ActiveStepID != "build" {
		t.Fatalf("loadPersistedMissionRuntimeSnapshot() runtime = %#v, want paused build runtime", runtime)
	}
	if control == nil || control.Step.ID != "build" {
		t.Fatalf("loadPersistedMissionRuntimeSnapshot() control = %#v, want build control", control)
	}
}

func TestLoadPersistedMissionRuntimeUsesSharedFallbackWhenStoreRootUnconfigured(t *testing.T) {
	job := testMissionBootstrapJob()
	statusFile := filepath.Join(t.TempDir(), "status.json")

	original := loadValidatedLegacyMissionStatusSnapshot
	t.Cleanup(func() { loadValidatedLegacyMissionStatusSnapshot = original })
	called := 0
	loadValidatedLegacyMissionStatusSnapshot = func(path string, jobID string) (missioncontrol.MissionStatusSnapshot, bool, error) {
		called++
		return missioncontrol.MissionStatusSnapshot{
			JobID:  job.ID,
			StepID: "build",
			Runtime: &missioncontrol.JobRuntimeState{
				JobID:        job.ID,
				State:        missioncontrol.JobStatePaused,
				ActiveStepID: "build",
			},
			RuntimeControl: runtimeControlForBootstrapStep(t, job, "build"),
		}, true, nil
	}

	runtime, control, source, ok, err := loadPersistedMissionRuntime(statusFile, "", job, time.Now().UTC())
	if err != nil {
		t.Fatalf("loadPersistedMissionRuntime() error = %v", err)
	}
	if !ok {
		t.Fatal("loadPersistedMissionRuntime() ok = false, want true")
	}
	if source != statusFile {
		t.Fatalf("loadPersistedMissionRuntime() source = %q, want %q", source, statusFile)
	}
	if called != 1 {
		t.Fatalf("legacy helper calls = %d, want 1", called)
	}
	if runtime.State != missioncontrol.JobStatePaused || runtime.ActiveStepID != "build" {
		t.Fatalf("loadPersistedMissionRuntime() runtime = %#v, want paused build runtime", runtime)
	}
	if control == nil || control.Step.ID != "build" {
		t.Fatalf("loadPersistedMissionRuntime() control = %#v, want build control", control)
	}
}

func TestMissionStatusBootstrapFallsBackToSnapshotWhenDurableStoreEmptyForJob(t *testing.T) {
	ag := newMissionBootstrapTestLoop()
	cmd := newMissionBootstrapTestCommand()
	job := testMissionBootstrapJob()
	missionFile := writeMissionBootstrapJobFile(t, job)
	statusFile := filepath.Join(t.TempDir(), "status.json")
	storeRoot := missioncontrol.ResolveStoreRoot("", statusFile)
	now := time.Date(2026, 4, 5, 13, 0, 0, 0, time.UTC)

	writeCommittedMissionBootstrapJobRuntimeRecord(t, storeRoot, "other-job", 3, 1, now, missioncontrol.JobStatePaused, "build")
	writeMissionStatusSnapshotFile(t, statusFile, missionStatusSnapshot{
		MissionFile:    missionFile,
		JobID:          job.ID,
		StepID:         "build",
		RuntimeControl: runtimeControlForBootstrapStep(t, job, "build"),
		Runtime: &missioncontrol.JobRuntimeState{
			JobID:        job.ID,
			State:        missioncontrol.JobStatePaused,
			ActiveStepID: "build",
			PausedReason: missioncontrol.RuntimePauseReasonOperatorCommand,
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

	if err := configureMissionBootstrap(cmd, ag); err != nil {
		t.Fatalf("configureMissionBootstrap() error = %v", err)
	}

	runtime, ok := ag.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if runtime.State != missioncontrol.JobStatePaused {
		t.Fatalf("MissionRuntimeState().State = %q, want %q", runtime.State, missioncontrol.JobStatePaused)
	}
}

func TestLoadPersistedMissionRuntimeUsesSharedFallbackWhenDurableStoreEmptyForJob(t *testing.T) {
	job := testMissionBootstrapJob()
	statusFile := filepath.Join(t.TempDir(), "status.json")
	storeRoot := missioncontrol.ResolveStoreRoot("", statusFile)
	now := time.Date(2026, 4, 5, 13, 0, 0, 0, time.UTC)

	writeCommittedMissionBootstrapJobRuntimeRecord(t, storeRoot, "other-job", 3, 1, now, missioncontrol.JobStatePaused, "build")

	original := loadValidatedLegacyMissionStatusSnapshot
	t.Cleanup(func() { loadValidatedLegacyMissionStatusSnapshot = original })
	called := 0
	loadValidatedLegacyMissionStatusSnapshot = func(path string, jobID string) (missioncontrol.MissionStatusSnapshot, bool, error) {
		called++
		return missioncontrol.MissionStatusSnapshot{
			JobID:  job.ID,
			StepID: "build",
			Runtime: &missioncontrol.JobRuntimeState{
				JobID:        job.ID,
				State:        missioncontrol.JobStatePaused,
				ActiveStepID: "build",
			},
			RuntimeControl: runtimeControlForBootstrapStep(t, job, "build"),
		}, true, nil
	}

	runtime, control, source, ok, err := loadPersistedMissionRuntime(statusFile, storeRoot, job, now)
	if err != nil {
		t.Fatalf("loadPersistedMissionRuntime() error = %v", err)
	}
	if !ok {
		t.Fatal("loadPersistedMissionRuntime() ok = false, want true")
	}
	if source != statusFile {
		t.Fatalf("loadPersistedMissionRuntime() source = %q, want %q", source, statusFile)
	}
	if called != 1 {
		t.Fatalf("legacy helper calls = %d, want 1", called)
	}
	if runtime.State != missioncontrol.JobStatePaused || runtime.ActiveStepID != "build" {
		t.Fatalf("loadPersistedMissionRuntime() runtime = %#v, want paused build runtime", runtime)
	}
	if control == nil || control.Step.ID != "build" {
		t.Fatalf("loadPersistedMissionRuntime() control = %#v, want build control", control)
	}
}

func TestMissionStatusBootstrapPrefersDurableRuntimeOverConflictingSnapshot(t *testing.T) {
	ag := newMissionBootstrapTestLoop()
	cmd := newMissionBootstrapTestCommand()
	job := testMissionBootstrapJob()
	missionFile := writeMissionBootstrapJobFile(t, job)
	statusFile := filepath.Join(t.TempDir(), "status.json")
	storeRoot := missioncontrol.ResolveStoreRoot("", statusFile)
	now := time.Date(2026, 4, 5, 14, 0, 0, 0, time.UTC)

	writeCommittedMissionBootstrapJobRuntimeRecord(t, storeRoot, job.ID, 5, 1, now, missioncontrol.JobStatePaused, "build")
	writeCommittedMissionBootstrapRuntimeControlRecord(t, storeRoot, 5, 1, runtimeControlForBootstrapStep(t, job, "build"))
	writeMissionStatusSnapshotFile(t, statusFile, missionStatusSnapshot{
		MissionFile:    missionFile,
		JobID:          job.ID,
		StepID:         "final",
		RuntimeControl: runtimeControlForBootstrapStep(t, job, "final"),
		Runtime: &missioncontrol.JobRuntimeState{
			JobID:        job.ID,
			State:        missioncontrol.JobStatePaused,
			ActiveStepID: "final",
			PausedReason: missioncontrol.RuntimePauseReasonOperatorCommand,
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

	if err := configureMissionBootstrap(cmd, ag); err != nil {
		t.Fatalf("configureMissionBootstrap() error = %v", err)
	}

	runtime, ok := ag.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if runtime.ActiveStepID != "build" {
		t.Fatalf("MissionRuntimeState().ActiveStepID = %q, want durable %q", runtime.ActiveStepID, "build")
	}
}

func TestLoadPersistedMissionRuntimeDoesNotFallbackWhenDurableHydrationFails(t *testing.T) {
	job := testMissionBootstrapJob()
	statusFile := filepath.Join(t.TempDir(), "status.json")
	storeRoot := missioncontrol.ResolveStoreRoot("", statusFile)
	now := time.Date(2026, 4, 5, 15, 0, 0, 0, time.UTC)

	writeCommittedMissionBootstrapJobRuntimeRecord(t, storeRoot, job.ID, 6, 1, now, missioncontrol.JobStatePaused, "build")
	writeCommittedMissionBootstrapRuntimeControlRecord(t, storeRoot, 6, 1, runtimeControlForBootstrapStep(t, job, "final"))

	original := loadValidatedLegacyMissionStatusSnapshot
	t.Cleanup(func() { loadValidatedLegacyMissionStatusSnapshot = original })
	called := 0
	loadValidatedLegacyMissionStatusSnapshot = func(path string, jobID string) (missioncontrol.MissionStatusSnapshot, bool, error) {
		called++
		return missioncontrol.MissionStatusSnapshot{}, false, nil
	}

	_, _, _, ok, err := loadPersistedMissionRuntime(statusFile, storeRoot, job, now)
	if err == nil {
		t.Fatal("loadPersistedMissionRuntime() error = nil, want durable hydration failure")
	}
	if ok {
		t.Fatal("loadPersistedMissionRuntime() ok = true, want false")
	}
	if called != 0 {
		t.Fatalf("legacy helper calls = %d, want 0 on durable failure", called)
	}
}

func TestMissionStatusBootstrapFailsClosedWhenDurableHydrationFails(t *testing.T) {
	ag := newMissionBootstrapTestLoop()
	cmd := newMissionBootstrapTestCommand()
	job := testMissionBootstrapJob()
	missionFile := writeMissionBootstrapJobFile(t, job)
	statusFile := filepath.Join(t.TempDir(), "status.json")
	storeRoot := missioncontrol.ResolveStoreRoot("", statusFile)
	now := time.Date(2026, 4, 5, 15, 0, 0, 0, time.UTC)

	writeCommittedMissionBootstrapJobRuntimeRecord(t, storeRoot, job.ID, 6, 1, now, missioncontrol.JobStatePaused, "build")
	writeCommittedMissionBootstrapRuntimeControlRecord(t, storeRoot, 6, 1, runtimeControlForBootstrapStep(t, job, "final"))
	writeMissionStatusSnapshotFile(t, statusFile, missionStatusSnapshot{
		MissionFile:    missionFile,
		JobID:          job.ID,
		StepID:         "build",
		RuntimeControl: runtimeControlForBootstrapStep(t, job, "build"),
		Runtime: &missioncontrol.JobRuntimeState{
			JobID:        job.ID,
			State:        missioncontrol.JobStatePaused,
			ActiveStepID: "build",
			PausedReason: missioncontrol.RuntimePauseReasonOperatorCommand,
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
		t.Fatal("configureMissionBootstrap() error = nil, want durable hydration failure")
	}
	if !strings.Contains(err.Error(), "durable store") {
		t.Fatalf("configureMissionBootstrap() error = %q, want durable-store failure", err)
	}
}

func TestMissionStatusBootstrapDurableTerminalStateDoesNotRestoreActiveControl(t *testing.T) {
	ag := newMissionBootstrapTestLoop()
	cmd := newMissionBootstrapTestCommand()
	job := testMissionBootstrapJob()
	missionFile := writeMissionBootstrapJobFile(t, job)
	statusFile := filepath.Join(t.TempDir(), "status.json")
	storeRoot := missioncontrol.ResolveStoreRoot("", statusFile)
	now := time.Date(2026, 4, 5, 16, 0, 0, 0, time.UTC)

	writeCommittedMissionBootstrapJobRuntimeRecord(t, storeRoot, job.ID, 8, 2, now, missioncontrol.JobStateCompleted, "")
	writeCommittedMissionBootstrapRuntimeControlRecord(t, storeRoot, 8, 1, runtimeControlForBootstrapStep(t, job, "build"))
	writeCommittedMissionBootstrapActiveJobRecord(t, storeRoot, 8, missioncontrol.JobStateRunning, "build", now, 1)

	if err := cmd.Flags().Set("mission-file", missionFile); err != nil {
		t.Fatalf("Flags().Set(mission-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step", "build"); err != nil {
		t.Fatalf("Flags().Set(mission-step) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-status-file", statusFile); err != nil {
		t.Fatalf("Flags().Set(mission-status-file) error = %v", err)
	}

	if err := configureMissionBootstrap(cmd, ag); err != nil {
		t.Fatalf("configureMissionBootstrap() error = %v", err)
	}
	if _, ok := ag.ActiveMissionStep(); ok {
		t.Fatal("ActiveMissionStep() ok = true, want false for durable terminal state")
	}
	if _, ok := ag.MissionRuntimeControl(); ok {
		t.Fatal("MissionRuntimeControl() ok = true, want false for durable terminal state")
	}
	runtime, ok := ag.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if runtime.State != missioncontrol.JobStateCompleted {
		t.Fatalf("MissionRuntimeState().State = %q, want %q", runtime.State, missioncontrol.JobStateCompleted)
	}
	if runtime.ActiveStepID != "" {
		t.Fatalf("MissionRuntimeState().ActiveStepID = %q, want empty", runtime.ActiveStepID)
	}
}

func TestMissionStatusBootstrapDurableRuntimeStillRequiresResumeApproval(t *testing.T) {
	ag := newMissionBootstrapTestLoop()
	cmd := newMissionBootstrapTestCommand()
	job := testMissionBootstrapJob()
	missionFile := writeMissionBootstrapJobFile(t, job)
	statusFile := filepath.Join(t.TempDir(), "status.json")
	storeRoot := missioncontrol.ResolveStoreRoot("", statusFile)
	now := time.Date(2026, 4, 5, 17, 0, 0, 0, time.UTC)

	writeCommittedMissionBootstrapJobRuntimeRecord(t, storeRoot, job.ID, 9, 1, now, missioncontrol.JobStateRunning, "build")
	writeCommittedMissionBootstrapRuntimeControlRecord(t, storeRoot, 9, 1, runtimeControlForBootstrapStep(t, job, "build"))

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

func TestMissionStatusRuntimePersistenceUpdatesDurableStoreAndSnapshotTogether(t *testing.T) {
	statusFile := filepath.Join(t.TempDir(), "status.json")
	cmd := newMissionBootstrapTestCommand()
	job := testMissionBootstrapJob()
	missionFile := writeMissionBootstrapJobFile(t, job)
	if err := cmd.Flags().Set("mission-status-file", statusFile); err != nil {
		t.Fatalf("Flags().Set(mission-status-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-file", missionFile); err != nil {
		t.Fatalf("Flags().Set(mission-file) error = %v", err)
	}

	hub := chat.NewHub(10)
	provider := &missionStatusFixedResponseProvider{content: "unused"}
	ag := agent.NewAgentLoop(hub, provider, provider.GetDefaultModel(), 3, "", nil)
	installMissionRuntimeChangeHook(cmd, ag)

	if err := ag.ActivateMissionStep(job, "build"); err != nil {
		t.Fatalf("ActivateMissionStep() error = %v", err)
	}
	if _, err := ag.ProcessDirect("PAUSE job-1", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(PAUSE) error = %v", err)
	}

	snapshot := readMissionStatusSnapshotFile(t, statusFile)
	if snapshot.Runtime == nil || snapshot.Runtime.State != missioncontrol.JobStatePaused {
		t.Fatalf("snapshot.Runtime = %#v, want paused runtime", snapshot.Runtime)
	}

	storeRoot := resolveMissionStoreRoot(cmd)
	assertMissionStatusSnapshotMatchesCommittedDurableState(t, snapshot, storeRoot, job.ID, time.Now().UTC())

	runtime, err := missioncontrol.HydrateCommittedJobRuntimeState(storeRoot, job.ID, time.Now().UTC())
	if err != nil {
		t.Fatalf("HydrateCommittedJobRuntimeState() error = %v", err)
	}
	if runtime.State != missioncontrol.JobStatePaused {
		t.Fatalf("HydrateCommittedJobRuntimeState().State = %q, want %q", runtime.State, missioncontrol.JobStatePaused)
	}
	control, err := missioncontrol.HydrateCommittedRuntimeControlContext(storeRoot, job.ID, time.Now().UTC())
	if err != nil {
		t.Fatalf("HydrateCommittedRuntimeControlContext() error = %v", err)
	}
	if control == nil || control.Step.ID != "build" {
		t.Fatalf("HydrateCommittedRuntimeControlContext() = %#v, want build control", control)
	}
}

func TestMissionStatusRuntimePersistenceDurableWriteFailureLeavesSnapshotUnchanged(t *testing.T) {
	statusFile := filepath.Join(t.TempDir(), "status.json")
	cmd := newMissionBootstrapTestCommand()
	job := testMissionBootstrapJob()
	missionFile := writeMissionBootstrapJobFile(t, job)
	if err := cmd.Flags().Set("mission-status-file", statusFile); err != nil {
		t.Fatalf("Flags().Set(mission-status-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-file", missionFile); err != nil {
		t.Fatalf("Flags().Set(mission-file) error = %v", err)
	}

	storeRoot := resolveMissionStoreRoot(cmd)
	seedIncoherentMissionStore(t, storeRoot, time.Date(2026, 4, 5, 18, 0, 0, 0, time.UTC))

	hub := chat.NewHub(10)
	provider := &missionStatusFixedResponseProvider{content: "unused"}
	ag := agent.NewAgentLoop(hub, provider, provider.GetDefaultModel(), 3, "", nil)
	installMissionRuntimeChangeHook(cmd, ag)

	err := ag.ActivateMissionStep(job, "build")
	if err == nil {
		t.Fatal("ActivateMissionStep() error = nil, want durable write failure")
	}
	if _, statErr := os.Stat(statusFile); !os.IsNotExist(statErr) {
		t.Fatalf("status file stat error = %v, want not-exist", statErr)
	}
	if _, ok := ag.MissionRuntimeState(); ok {
		t.Fatal("MissionRuntimeState() ok = true, want false after failed durable persist")
	}
}

func TestMissionStatusRuntimePersistenceProjectionFailureLeavesSnapshotUnchanged(t *testing.T) {
	statusFile := filepath.Join(t.TempDir(), "status.json")
	cmd := newMissionBootstrapTestCommand()
	job := testMissionBootstrapJob()
	missionFile := writeMissionBootstrapJobFile(t, job)
	if err := cmd.Flags().Set("mission-status-file", statusFile); err != nil {
		t.Fatalf("Flags().Set(mission-status-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-file", missionFile); err != nil {
		t.Fatalf("Flags().Set(mission-file) error = %v", err)
	}

	previous := missionStatusSnapshot{
		MissionFile: missionFile,
		JobID:       "previous-job",
		StepID:      "previous-step",
		UpdatedAt:   "2026-04-05T18:00:00Z",
	}
	writeMissionStatusSnapshotFile(t, statusFile, previous)

	originalWrite := writeMissionStatusSnapshotAtomic
	t.Cleanup(func() { writeMissionStatusSnapshotAtomic = originalWrite })
	writeMissionStatusSnapshotAtomic = func(path string, snapshot missionStatusSnapshot) error {
		if path == statusFile {
			return errors.New("forced projection write failure")
		}
		return originalWrite(path, snapshot)
	}

	hub := chat.NewHub(10)
	provider := &missionStatusFixedResponseProvider{content: "unused"}
	ag := agent.NewAgentLoop(hub, provider, provider.GetDefaultModel(), 3, "", nil)
	installMissionRuntimeChangeHook(cmd, ag)

	err := ag.ActivateMissionStep(job, "build")
	if err == nil {
		t.Fatal("ActivateMissionStep() error = nil, want projection failure")
	}
	if !strings.Contains(err.Error(), "forced projection write failure") {
		t.Fatalf("ActivateMissionStep() error = %q, want projection failure", err)
	}

	if got := readMissionStatusSnapshotFile(t, statusFile); !reflect.DeepEqual(got, previous) {
		t.Fatalf("status snapshot = %#v, want unchanged %#v", got, previous)
	}

	storeRoot := resolveMissionStoreRoot(cmd)
	durableRuntime, err := missioncontrol.HydrateCommittedJobRuntimeState(storeRoot, job.ID, time.Now().UTC())
	if err != nil {
		t.Fatalf("HydrateCommittedJobRuntimeState() error = %v", err)
	}
	if durableRuntime.State != missioncontrol.JobStateRunning || durableRuntime.ActiveStepID != "build" {
		t.Fatalf("HydrateCommittedJobRuntimeState() = %#v, want committed running build runtime", durableRuntime)
	}

	runtime, ok := ag.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want in-memory runtime preserved")
	}
	if runtime.State != missioncontrol.JobStateRunning || runtime.ActiveStepID != "build" {
		t.Fatalf("MissionRuntimeState() = %#v, want running build runtime", runtime)
	}
}

func TestMissionStatusBootstrapPrefersLatestDurableStateAfterMutation(t *testing.T) {
	statusFile := filepath.Join(t.TempDir(), "status.json")
	cmd := newMissionBootstrapTestCommand()
	job := testMissionBootstrapJob()
	missionFile := writeMissionBootstrapJobFile(t, job)
	if err := cmd.Flags().Set("mission-status-file", statusFile); err != nil {
		t.Fatalf("Flags().Set(mission-status-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-file", missionFile); err != nil {
		t.Fatalf("Flags().Set(mission-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step", "build"); err != nil {
		t.Fatalf("Flags().Set(mission-step) error = %v", err)
	}

	hub := chat.NewHub(10)
	provider := &missionStatusFixedResponseProvider{content: "unused"}
	ag := agent.NewAgentLoop(hub, provider, provider.GetDefaultModel(), 3, "", nil)
	installMissionRuntimeChangeHook(cmd, ag)

	if err := configureMissionBootstrap(cmd, ag); err != nil {
		t.Fatalf("configureMissionBootstrap() error = %v", err)
	}
	if _, err := ag.ProcessDirect("PAUSE job-1", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(PAUSE) error = %v", err)
	}

	writeMissionStatusSnapshotFile(t, statusFile, missionStatusSnapshot{
		MissionFile:    missionFile,
		JobID:          job.ID,
		StepID:         "build",
		RuntimeControl: runtimeControlForBootstrapStep(t, job, "build"),
		Runtime: &missioncontrol.JobRuntimeState{
			JobID:        job.ID,
			State:        missioncontrol.JobStateRunning,
			ActiveStepID: "build",
		},
	})

	ag2 := newMissionBootstrapTestLoop()
	installMissionRuntimeChangeHook(cmd, ag2)
	if err := configureMissionBootstrap(cmd, ag2); err != nil {
		t.Fatalf("configureMissionBootstrap(second boot) error = %v", err)
	}

	runtime, ok := ag2.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if runtime.State != missioncontrol.JobStatePaused {
		t.Fatalf("MissionRuntimeState().State = %q, want durable %q", runtime.State, missioncontrol.JobStatePaused)
	}
}

func newMissionBootstrapTestLoop() *agent.AgentLoop {
	hub := chat.NewHub(10)
	provider := providers.NewStubProvider()
	return agent.NewAgentLoop(hub, provider, provider.GetDefaultModel(), 3, "", nil)
}

func writeCommittedMissionBootstrapJobRuntimeRecord(t *testing.T, root string, jobID string, writerEpoch, appliedSeq uint64, at time.Time, state missioncontrol.JobState, activeStepID string) {
	t.Helper()

	record := missioncontrol.JobRuntimeRecord{
		RecordVersion: missioncontrol.StoreRecordVersion,
		WriterEpoch:   writerEpoch,
		AppliedSeq:    appliedSeq,
		JobID:         jobID,
		State:         state,
		ActiveStepID:  activeStepID,
		CreatedAt:     at.Add(-time.Minute).UTC(),
		UpdatedAt:     at.UTC(),
		StartedAt:     at.Add(-time.Minute).UTC(),
	}
	switch state {
	case missioncontrol.JobStateRunning:
		record.ActiveStepAt = at.UTC()
	case missioncontrol.JobStateWaitingUser:
		record.WaitingAt = at.UTC()
		record.WaitingReason = "awaiting operator confirmation"
	case missioncontrol.JobStatePaused:
		record.PausedAt = at.UTC()
		record.PausedReason = missioncontrol.RuntimePauseReasonOperatorCommand
	case missioncontrol.JobStateAborted:
		record.AbortedAt = at.UTC()
		record.AbortedReason = "operator aborted"
	case missioncontrol.JobStateCompleted:
		record.CompletedAt = at.UTC()
	case missioncontrol.JobStateFailed:
		record.FailedAt = at.UTC()
	}
	if err := missioncontrol.StoreJobRuntimeRecord(root, record); err != nil {
		t.Fatalf("StoreJobRuntimeRecord() error = %v", err)
	}
}

func writeCommittedMissionBootstrapRuntimeControlRecord(t *testing.T, root string, writerEpoch, seq uint64, control *missioncontrol.RuntimeControlContext) {
	t.Helper()
	if control == nil {
		return
	}

	record := missioncontrol.RuntimeControlRecord{
		RecordVersion: missioncontrol.StoreRecordVersion,
		WriterEpoch:   writerEpoch,
		LastSeq:       seq,
		JobID:         control.JobID,
		StepID:        control.Step.ID,
		MaxAuthority:  control.MaxAuthority,
		AllowedTools:  append([]string(nil), control.AllowedTools...),
		Step:          cloneMissionBootstrapStep(control.Step),
	}
	if err := missioncontrol.StoreRuntimeControlRecord(root, record); err != nil {
		t.Fatalf("StoreRuntimeControlRecord() error = %v", err)
	}
}

func writeCommittedMissionBootstrapActiveJobRecord(t *testing.T, root string, writerEpoch uint64, state missioncontrol.JobState, activeStepID string, at time.Time, activationSeq uint64) {
	t.Helper()

	record, err := missioncontrol.NewActiveJobRecord(
		writerEpoch,
		"job-1",
		state,
		activeStepID,
		"holder-1",
		at.Add(time.Minute),
		at,
		activationSeq,
	)
	if err != nil {
		t.Fatalf("NewActiveJobRecord() error = %v", err)
	}
	if err := missioncontrol.StoreActiveJobRecord(root, record); err != nil {
		t.Fatalf("StoreActiveJobRecord() error = %v", err)
	}
}

func seedIncoherentMissionStore(t *testing.T, root string, now time.Time) {
	t.Helper()

	manifest, err := missioncontrol.InitStoreManifest(root, now)
	if err != nil {
		t.Fatalf("InitStoreManifest() error = %v", err)
	}
	manifest.CurrentWriterEpoch = 2
	if err := missioncontrol.StoreManifestRecord(root, manifest); err != nil {
		t.Fatalf("StoreManifestRecord() error = %v", err)
	}
	if err := missioncontrol.StoreWriterLockRecord(root, missioncontrol.WriterLockRecord{
		RecordVersion:  missioncontrol.StoreRecordVersion,
		WriterEpoch:    1,
		LeaseHolderID:  "other-holder",
		StartedAt:      now,
		RenewedAt:      now,
		LeaseExpiresAt: now.Add(time.Minute),
		JobID:          "job-1",
	}); err != nil {
		t.Fatalf("StoreWriterLockRecord() error = %v", err)
	}
}

func assertMissionStatusSnapshotMatchesCommittedDurableState(t *testing.T, snapshot missionStatusSnapshot, storeRoot string, jobID string, now time.Time) {
	t.Helper()

	updatedAt, err := time.Parse(time.RFC3339Nano, snapshot.UpdatedAt)
	if err != nil {
		t.Fatalf("time.Parse(snapshot.UpdatedAt=%q) error = %v", snapshot.UpdatedAt, err)
	}
	expected, err := missioncontrol.BuildCommittedMissionStatusSnapshot(
		storeRoot,
		jobID,
		missioncontrol.MissionStatusSnapshotOptions{
			MissionRequired: snapshot.MissionRequired,
			MissionFile:     snapshot.MissionFile,
			UpdatedAt:       updatedAt,
		},
	)
	if err != nil {
		t.Fatalf("BuildCommittedMissionStatusSnapshot(%q) error = %v", jobID, err)
	}
	if !reflect.DeepEqual(snapshot, expected) {
		t.Fatalf("snapshot = %#v, want durable %#v", snapshot, expected)
	}
}

func cloneMissionBootstrapStep(step missioncontrol.Step) missioncontrol.Step {
	cloned := step
	cloned.DependsOn = append([]string(nil), step.DependsOn...)
	cloned.AllowedTools = append([]string(nil), step.AllowedTools...)
	cloned.SuccessCriteria = append([]string(nil), step.SuccessCriteria...)
	cloned.LongRunningStartupCommand = append([]string(nil), step.LongRunningStartupCommand...)
	return cloned
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

func TestConfigureMissionBootstrapJobAcceptsV2LongRunningCodeMissionFile(t *testing.T) {
	cmd := newMissionBootstrapTestCommand()
	ag := newMissionBootstrapTestLoop()
	missionPath := writeMissionBootstrapJobFile(t, missioncontrol.Job{
		ID:           "job-1",
		SpecVersion:  missioncontrol.JobSpecVersionV2,
		MaxAuthority: missioncontrol.AuthorityTierHigh,
		AllowedTools: []string{"filesystem"},
		Plan: missioncontrol.Plan{
			ID: "plan-1",
			Steps: []missioncontrol.Step{
				{
					ID:                        "build",
					Type:                      missioncontrol.StepTypeLongRunningCode,
					RequiredAuthority:         missioncontrol.AuthorityTierLow,
					AllowedTools:              []string{"filesystem"},
					SuccessCriteria:           []string{"Build the service artifact and record the startup command."},
					LongRunningStartupCommand: []string{"npm", "start"},
					LongRunningArtifactPath:   "dist/service.js",
				},
				{
					ID:        "final",
					Type:      missioncontrol.StepTypeFinalResponse,
					DependsOn: []string{"build"},
				},
			},
		},
	})
	if err := cmd.Flags().Set("mission-file", missionPath); err != nil {
		t.Fatalf("Flags().Set(mission-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step", "build"); err != nil {
		t.Fatalf("Flags().Set(mission-step) error = %v", err)
	}

	job := configureMissionBootstrapJobForStartupTest(t, cmd, ag)
	if job.SpecVersion != missioncontrol.JobSpecVersionV2 {
		t.Fatalf("Job.SpecVersion = %q, want %q", job.SpecVersion, missioncontrol.JobSpecVersionV2)
	}

	ec, ok := ag.ActiveMissionStep()
	if !ok {
		t.Fatal("ActiveMissionStep() ok = false, want activated long_running_code step")
	}
	if ec.Step == nil {
		t.Fatal("ActiveMissionStep().Step = nil, want non-nil")
	}
	if ec.Step.Type != missioncontrol.StepTypeLongRunningCode {
		t.Fatalf("ActiveMissionStep().Step.Type = %q, want %q", ec.Step.Type, missioncontrol.StepTypeLongRunningCode)
	}
	if !reflect.DeepEqual(ec.Step.LongRunningStartupCommand, []string{"npm", "start"}) {
		t.Fatalf("ActiveMissionStep().Step.LongRunningStartupCommand = %#v, want %#v", ec.Step.LongRunningStartupCommand, []string{"npm", "start"})
	}
	if ec.Step.LongRunningArtifactPath != "dist/service.js" {
		t.Fatalf("ActiveMissionStep().Step.LongRunningArtifactPath = %q, want %q", ec.Step.LongRunningArtifactPath, "dist/service.js")
	}
}

type missionStatusFixedResponseProvider struct {
	content string
}

func (p *missionStatusFixedResponseProvider) Chat(ctx context.Context, messages []providers.Message, tools []providers.ToolDefinition, model string) (providers.LLMResponse, error) {
	return providers.LLMResponse{Content: p.content}, nil
}

func (p *missionStatusFixedResponseProvider) GetDefaultModel() string {
	return "stub"
}

func writeMissionBootstrapJobFile(t *testing.T, job missioncontrol.Job, paths ...string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "mission.json")
	if len(paths) > 0 {
		path = paths[0]
	}
	data, err := json.Marshal(job)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	return path
}

func runtimeControlForBootstrapStep(t *testing.T, job missioncontrol.Job, stepID string) *missioncontrol.RuntimeControlContext {
	t.Helper()

	control, err := missioncontrol.BuildRuntimeControlContext(job, stepID)
	if err != nil {
		t.Fatalf("BuildRuntimeControlContext() error = %v", err)
	}
	return &control
}

func expectedAuthorizationApprovalContent(authority missioncontrol.AuthorityTier) missioncontrol.ApprovalRequestContent {
	return missioncontrol.ApprovalRequestContent{
		ProposedAction:   "Complete the authorization discussion step and continue to the next mission step.",
		WhyNeeded:        "This step asks the operator to explicitly approve continuation before the mission can proceed.",
		AuthorityTier:    authority,
		IdentityScope:    missioncontrol.ApprovalScopeNone,
		PublicScope:      missioncontrol.ApprovalScopeNone,
		FilesystemEffect: missioncontrol.ApprovalEffectNone,
		ProcessEffect:    missioncontrol.ApprovalEffectNone,
		NetworkEffect:    missioncontrol.ApprovalEffectNone,
		FallbackIfDenied: "Keep the mission in waiting_user and require an explicit follow-up decision before proceeding.",
	}
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
