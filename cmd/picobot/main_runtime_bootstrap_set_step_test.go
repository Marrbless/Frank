package main

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/local/picobot/internal/missioncontrol"
)

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
		Active:       true,
		JobID:        "other-job",
		StepID:       "build",
		StepType:     string(missioncontrol.StepTypeOneShotCode),
		AllowedTools: []string{"read"},
		UpdatedAt:    "2026-03-20T12:00:00Z",
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
		Active:       true,
		JobID:        job.ID,
		StepID:       "final",
		StepType:     string(missioncontrol.StepTypeFinalResponse),
		AllowedTools: []string{"read"},
		UpdatedAt:    "2026-03-20T12:00:01Z",
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
		Active:       true,
		JobID:        "other-job",
		StepID:       "build",
		StepType:     string(missioncontrol.StepTypeOneShotCode),
		AllowedTools: []string{"read"},
		UpdatedAt:    "2026-03-20T12:00:00Z",
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
		Active:       true,
		JobID:        "other-job",
		StepID:       "final",
		StepType:     string(missioncontrol.StepTypeFinalResponse),
		AllowedTools: []string{"read"},
		UpdatedAt:    "2026-03-20T12:00:01Z",
	})

	select {
	case err := <-done:
		t.Fatalf("Execute() returned while job_id was still mismatched: %v", err)
	case <-time.After(60 * time.Millisecond):
	}

	writeMissionStatusSnapshotFile(t, statusPath, missionStatusSnapshot{
		Active:       true,
		JobID:        job.ID,
		StepID:       "final",
		StepType:     string(missioncontrol.StepTypeFinalResponse),
		AllowedTools: []string{"read"},
		UpdatedAt:    "2026-03-20T12:00:02Z",
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
		Active:       true,
		JobID:        "other-job",
		StepID:       "final",
		StepType:     string(missioncontrol.StepTypeFinalResponse),
		AllowedTools: []string{"read"},
		UpdatedAt:    "2026-03-20T12:00:00Z",
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

func TestMissionSetStepCommandConfirmationUsesSharedObservationReader(t *testing.T) {
	original := loadMissionStatusObservation
	t.Cleanup(func() { loadMissionStatusObservation = original })

	controlPath := filepath.Join(t.TempDir(), "control.json")
	called := 0
	loadMissionStatusObservation = func(path string) (missioncontrol.MissionStatusSnapshot, error) {
		called++
		if path != "status.json" {
			t.Fatalf("shared observation path = %q, want %q", path, "status.json")
		}
		switch called {
		case 1:
			return missioncontrol.MissionStatusSnapshot{
				Active:    true,
				StepID:    "build",
				UpdatedAt: "2026-03-20T12:00:00Z",
			}, nil
		default:
			return missioncontrol.MissionStatusSnapshot{
				Active:    true,
				StepID:    "final",
				UpdatedAt: "2026-03-20T12:00:01Z",
			}, nil
		}
	}

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{
		"mission", "set-step",
		"--control-file", controlPath,
		"--step-id", "final",
		"--status-file", "status.json",
		"--wait-timeout", "75ms",
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if called < 2 {
		t.Fatalf("shared observation calls = %d, want at least 2", called)
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
	if !strings.Contains(err.Error(), "E_PLAN_INVALID") {
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
	if !strings.Contains(err.Error(), "E_INVALID_ACTION_FOR_STEP") {
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
	beforeControl := mustReadFile(t, controlFile)

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
	if !bytes.Equal(mustReadFile(t, controlFile), beforeControl) {
		t.Fatalf("mission step control input changed from %q to %q, want unchanged input semantics", string(beforeControl), string(mustReadFile(t, controlFile)))
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

func TestRestoreMissionStepControlFileOnStartupRejectsPreviouslyCompletedStepReplay(t *testing.T) {
	ag := newMissionBootstrapTestLoop()
	cmd := newMissionBootstrapTestCommand()
	job := testMissionBootstrapJob()
	missionFile := writeMissionBootstrapJobFile(t, job)
	controlFile := writeMissionStepControlFile(t, missionStepControlFile{StepID: "build", UpdatedAt: "2026-03-19T12:00:00Z"})
	statusFile := filepath.Join(t.TempDir(), "status.json")
	logBuf, restoreLog := captureStandardLogger(t)
	defer restoreLog()

	writeMissionStatusSnapshotFile(t, statusFile, missionStatusSnapshot{
		Active:         true,
		MissionFile:    missionFile,
		JobID:          job.ID,
		StepID:         "final",
		RuntimeControl: runtimeControlForBootstrapStep(t, job, "final"),
		Runtime: &missioncontrol.JobRuntimeState{
			JobID:        job.ID,
			State:        missioncontrol.JobStatePaused,
			ActiveStepID: "final",
			PausedReason: missioncontrol.RuntimePauseReasonOperatorCommand,
			CompletedSteps: []missioncontrol.RuntimeStepRecord{
				{StepID: "build", At: time.Date(2026, 3, 19, 11, 30, 0, 0, time.UTC)},
			},
		},
		UpdatedAt: "2026-03-19T12:00:00Z",
	})

	if err := cmd.Flags().Set("mission-file", missionFile); err != nil {
		t.Fatalf("Flags().Set(mission-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step", "final"); err != nil {
		t.Fatalf("Flags().Set(mission-step) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-status-file", statusFile); err != nil {
		t.Fatalf("Flags().Set(mission-status-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step-control-file", controlFile); err != nil {
		t.Fatalf("Flags().Set(mission-step-control-file) error = %v", err)
	}

	bootstrappedJob := configureMissionBootstrapJobForStartupTest(t, cmd, ag)
	baseline := restoreMissionStepControlFileOnStartup(cmd, ag, bootstrappedJob)
	if baseline != nil {
		t.Fatalf("restoreMissionStepControlFileOnStartup() baseline = %q, want nil after completed-step replay rejection", string(baseline))
	}

	if _, ok := ag.ActiveMissionStep(); ok {
		t.Fatal("ActiveMissionStep() ok = true, want no live execution context after rehydrated replay rejection")
	}

	runtime, ok := ag.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want persisted runtime state")
	}
	if runtime.ActiveStepID != "final" {
		t.Fatalf("MissionRuntimeState().ActiveStepID = %q, want %q", runtime.ActiveStepID, "final")
	}
	if len(runtime.CompletedSteps) != 1 || runtime.CompletedSteps[0].StepID != "build" {
		t.Fatalf("MissionRuntimeState().CompletedSteps = %#v, want preserved build completion", runtime.CompletedSteps)
	}

	logOutput := logBuf.String()
	if !strings.Contains(logOutput, `step "build" is already recorded as completed in runtime state`) {
		t.Fatalf("log output = %q, want completed-step replay rejection", logOutput)
	}
}

func TestRestoreMissionStepControlFileOnStartupRejectsPreviouslyFailedStepReplay(t *testing.T) {
	ag := newMissionBootstrapTestLoop()
	cmd := newMissionBootstrapTestCommand()
	job := testMissionBootstrapJob()
	missionFile := writeMissionBootstrapJobFile(t, job)
	controlFile := writeMissionStepControlFile(t, missionStepControlFile{StepID: "build", UpdatedAt: "2026-03-19T12:00:00Z"})
	statusFile := filepath.Join(t.TempDir(), "status.json")
	logBuf, restoreLog := captureStandardLogger(t)
	defer restoreLog()

	writeMissionStatusSnapshotFile(t, statusFile, missionStatusSnapshot{
		Active:         true,
		MissionFile:    missionFile,
		JobID:          job.ID,
		StepID:         "final",
		RuntimeControl: runtimeControlForBootstrapStep(t, job, "final"),
		Runtime: &missioncontrol.JobRuntimeState{
			JobID:        job.ID,
			State:        missioncontrol.JobStatePaused,
			ActiveStepID: "final",
			PausedReason: missioncontrol.RuntimePauseReasonOperatorCommand,
			FailedSteps: []missioncontrol.RuntimeStepRecord{
				{StepID: "build", Reason: "validator failed", At: time.Date(2026, 3, 19, 11, 30, 0, 0, time.UTC)},
			},
		},
		UpdatedAt: "2026-03-19T12:00:00Z",
	})

	if err := cmd.Flags().Set("mission-file", missionFile); err != nil {
		t.Fatalf("Flags().Set(mission-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step", "final"); err != nil {
		t.Fatalf("Flags().Set(mission-step) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-status-file", statusFile); err != nil {
		t.Fatalf("Flags().Set(mission-status-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step-control-file", controlFile); err != nil {
		t.Fatalf("Flags().Set(mission-step-control-file) error = %v", err)
	}

	bootstrappedJob := configureMissionBootstrapJobForStartupTest(t, cmd, ag)
	baseline := restoreMissionStepControlFileOnStartup(cmd, ag, bootstrappedJob)
	if baseline != nil {
		t.Fatalf("restoreMissionStepControlFileOnStartup() baseline = %q, want nil after failed-step replay rejection", string(baseline))
	}

	if _, ok := ag.ActiveMissionStep(); ok {
		t.Fatal("ActiveMissionStep() ok = true, want no live execution context after rehydrated replay rejection")
	}

	runtime, ok := ag.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want persisted runtime state")
	}
	if runtime.ActiveStepID != "final" {
		t.Fatalf("MissionRuntimeState().ActiveStepID = %q, want %q", runtime.ActiveStepID, "final")
	}
	if len(runtime.FailedSteps) != 1 || runtime.FailedSteps[0].StepID != "build" {
		t.Fatalf("MissionRuntimeState().FailedSteps = %#v, want preserved build failure", runtime.FailedSteps)
	}

	logOutput := logBuf.String()
	if !strings.Contains(logOutput, `step "build" is already recorded as failed in runtime state`) {
		t.Fatalf("log output = %q, want failed-step replay rejection", logOutput)
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

func TestWatchMissionStepControlFileRejectsPreviouslyCompletedStepReplay(t *testing.T) {
	ag := newMissionBootstrapTestLoop()
	cmd := newMissionBootstrapTestCommand()
	job := testMissionBootstrapJob()
	missionFile := writeMissionBootstrapJobFile(t, job)
	controlFile := writeMissionStepControlFile(t, missionStepControlFile{StepID: "build", UpdatedAt: "2026-03-19T12:00:00Z"})
	statusFile := filepath.Join(t.TempDir(), "status.json")
	logBuf, restoreLog := captureStandardLogger(t)
	defer restoreLog()

	writeMissionStatusSnapshotFile(t, statusFile, missionStatusSnapshot{
		Active:         true,
		MissionFile:    missionFile,
		JobID:          job.ID,
		StepID:         "final",
		RuntimeControl: runtimeControlForBootstrapStep(t, job, "final"),
		Runtime: &missioncontrol.JobRuntimeState{
			JobID:        job.ID,
			State:        missioncontrol.JobStatePaused,
			ActiveStepID: "final",
			PausedReason: missioncontrol.RuntimePauseReasonOperatorCommand,
			CompletedSteps: []missioncontrol.RuntimeStepRecord{
				{StepID: "build", At: time.Date(2026, 3, 19, 11, 30, 0, 0, time.UTC)},
			},
		},
		UpdatedAt: "2026-03-19T12:00:00Z",
	})

	if err := cmd.Flags().Set("mission-file", missionFile); err != nil {
		t.Fatalf("Flags().Set(mission-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step", "final"); err != nil {
		t.Fatalf("Flags().Set(mission-step) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-status-file", statusFile); err != nil {
		t.Fatalf("Flags().Set(mission-status-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step-control-file", controlFile); err != nil {
		t.Fatalf("Flags().Set(mission-step-control-file) error = %v", err)
	}

	bootstrappedJob := configureMissionBootstrapJobForStartupTest(t, cmd, ag)

	ctx, cancel := context.WithCancel(context.Background())
	go watchMissionStepControlFile(ctx, cmd, ag, bootstrappedJob, controlFile, 5*time.Millisecond, nil)
	time.Sleep(25 * time.Millisecond)
	cancel()

	if _, ok := ag.ActiveMissionStep(); ok {
		t.Fatal("ActiveMissionStep() ok = true, want no live execution context after watcher replay rejection")
	}

	runtime, ok := ag.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want persisted runtime state")
	}
	if runtime.ActiveStepID != "final" {
		t.Fatalf("MissionRuntimeState().ActiveStepID = %q, want %q", runtime.ActiveStepID, "final")
	}
	if len(runtime.CompletedSteps) != 1 || runtime.CompletedSteps[0].StepID != "build" {
		t.Fatalf("MissionRuntimeState().CompletedSteps = %#v, want preserved build completion", runtime.CompletedSteps)
	}

	logOutput := logBuf.String()
	if !strings.Contains(logOutput, `mission step control apply failed`) {
		t.Fatalf("log output = %q, want watcher apply failure", logOutput)
	}
	if !strings.Contains(logOutput, `step "build" is already recorded as completed in runtime state`) {
		t.Fatalf("log output = %q, want completed-step replay rejection", logOutput)
	}
	if strings.Contains(logOutput, `mission step control apply succeeded`) {
		t.Fatalf("log output = %q, want no watcher apply success", logOutput)
	}
}

func TestWatchMissionStepControlFileRejectsPreviouslyFailedStepReplay(t *testing.T) {
	ag := newMissionBootstrapTestLoop()
	cmd := newMissionBootstrapTestCommand()
	job := testMissionBootstrapJob()
	missionFile := writeMissionBootstrapJobFile(t, job)
	controlFile := writeMissionStepControlFile(t, missionStepControlFile{StepID: "build", UpdatedAt: "2026-03-19T12:00:00Z"})
	statusFile := filepath.Join(t.TempDir(), "status.json")
	logBuf, restoreLog := captureStandardLogger(t)
	defer restoreLog()

	writeMissionStatusSnapshotFile(t, statusFile, missionStatusSnapshot{
		Active:         true,
		MissionFile:    missionFile,
		JobID:          job.ID,
		StepID:         "final",
		RuntimeControl: runtimeControlForBootstrapStep(t, job, "final"),
		Runtime: &missioncontrol.JobRuntimeState{
			JobID:        job.ID,
			State:        missioncontrol.JobStatePaused,
			ActiveStepID: "final",
			PausedReason: missioncontrol.RuntimePauseReasonOperatorCommand,
			FailedSteps: []missioncontrol.RuntimeStepRecord{
				{StepID: "build", Reason: "validator failed", At: time.Date(2026, 3, 19, 11, 30, 0, 0, time.UTC)},
			},
		},
		UpdatedAt: "2026-03-19T12:00:00Z",
	})

	if err := cmd.Flags().Set("mission-file", missionFile); err != nil {
		t.Fatalf("Flags().Set(mission-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step", "final"); err != nil {
		t.Fatalf("Flags().Set(mission-step) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-status-file", statusFile); err != nil {
		t.Fatalf("Flags().Set(mission-status-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step-control-file", controlFile); err != nil {
		t.Fatalf("Flags().Set(mission-step-control-file) error = %v", err)
	}

	bootstrappedJob := configureMissionBootstrapJobForStartupTest(t, cmd, ag)

	ctx, cancel := context.WithCancel(context.Background())
	go watchMissionStepControlFile(ctx, cmd, ag, bootstrappedJob, controlFile, 5*time.Millisecond, nil)
	time.Sleep(25 * time.Millisecond)
	cancel()

	if _, ok := ag.ActiveMissionStep(); ok {
		t.Fatal("ActiveMissionStep() ok = true, want no live execution context after watcher replay rejection")
	}

	runtime, ok := ag.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want persisted runtime state")
	}
	if runtime.ActiveStepID != "final" {
		t.Fatalf("MissionRuntimeState().ActiveStepID = %q, want %q", runtime.ActiveStepID, "final")
	}
	if len(runtime.FailedSteps) != 1 || runtime.FailedSteps[0].StepID != "build" {
		t.Fatalf("MissionRuntimeState().FailedSteps = %#v, want preserved build failure", runtime.FailedSteps)
	}

	logOutput := logBuf.String()
	if !strings.Contains(logOutput, `mission step control apply failed`) {
		t.Fatalf("log output = %q, want watcher apply failure", logOutput)
	}
	if !strings.Contains(logOutput, `step "build" is already recorded as failed in runtime state`) {
		t.Fatalf("log output = %q, want failed-step replay rejection", logOutput)
	}
	if strings.Contains(logOutput, `mission step control apply succeeded`) {
		t.Fatalf("log output = %q, want no watcher apply success", logOutput)
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

func TestMissionOperatorSetStepCommandActiveJobSucceedsThroughConfirmationPath(t *testing.T) {
	ag := newMissionBootstrapTestLoop()
	cmd := newMissionBootstrapTestCommand()
	job := testMissionBootstrapJob()
	missionFile := writeMissionBootstrapJobFile(t, job)
	statusFile := filepath.Join(t.TempDir(), "status.json")
	controlFile := filepath.Join(t.TempDir(), "control.json")

	if err := cmd.Flags().Set("mission-file", missionFile); err != nil {
		t.Fatalf("Flags().Set(mission-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step", "build"); err != nil {
		t.Fatalf("Flags().Set(mission-step) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-status-file", statusFile); err != nil {
		t.Fatalf("Flags().Set(mission-status-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step-control-file", controlFile); err != nil {
		t.Fatalf("Flags().Set(mission-step-control-file) error = %v", err)
	}

	bootstrappedJob := configureMissionBootstrapJobForStartupTest(t, cmd, ag)
	installMissionOperatorSetStepHook(cmd, ag, &bootstrappedJob, true)
	if err := writeMissionStatusSnapshotFromCommand(cmd, ag, time.Now()); err != nil {
		t.Fatalf("writeMissionStatusSnapshotFromCommand() error = %v", err)
	}

	resp, err := ag.ProcessDirect("SET_STEP job-1 final", 2*time.Second)
	if err != nil {
		t.Fatalf("ProcessDirect(SET_STEP) error = %v", err)
	}
	if resp != "Set step job=job-1 step=final." {
		t.Fatalf("ProcessDirect(SET_STEP) response = %q, want set-step acknowledgement", resp)
	}

	ec, ok := ag.ActiveMissionStep()
	if !ok || ec.Step == nil {
		t.Fatalf("ActiveMissionStep() = %#v, want active final step", ec)
	}
	if ec.Step.ID != "final" {
		t.Fatalf("ActiveMissionStep().Step.ID = %q, want %q", ec.Step.ID, "final")
	}

	control := readMissionStepControlFile(t, controlFile)
	if control.StepID != "final" {
		t.Fatalf("control.StepID = %q, want %q", control.StepID, "final")
	}

	snapshot := readMissionStatusSnapshotFile(t, statusFile)
	if snapshot.StepID != "final" {
		t.Fatalf("snapshot.StepID = %q, want %q", snapshot.StepID, "final")
	}
	if snapshot.JobID != job.ID {
		t.Fatalf("snapshot.JobID = %q, want %q", snapshot.JobID, job.ID)
	}
}

func TestMissionOperatorSetStepCommandWrongJobDoesNotBind(t *testing.T) {
	ag := newMissionBootstrapTestLoop()
	cmd := newMissionBootstrapTestCommand()
	job := testMissionBootstrapJob()
	missionFile := writeMissionBootstrapJobFile(t, job)
	statusFile := filepath.Join(t.TempDir(), "status.json")
	controlFile := filepath.Join(t.TempDir(), "control.json")

	if err := cmd.Flags().Set("mission-file", missionFile); err != nil {
		t.Fatalf("Flags().Set(mission-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step", "build"); err != nil {
		t.Fatalf("Flags().Set(mission-step) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-status-file", statusFile); err != nil {
		t.Fatalf("Flags().Set(mission-status-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step-control-file", controlFile); err != nil {
		t.Fatalf("Flags().Set(mission-step-control-file) error = %v", err)
	}

	bootstrappedJob := configureMissionBootstrapJobForStartupTest(t, cmd, ag)
	installMissionOperatorSetStepHook(cmd, ag, &bootstrappedJob, true)

	_, err := ag.ProcessDirect("SET_STEP other-job final", 2*time.Second)
	if err == nil {
		t.Fatal("ProcessDirect(SET_STEP wrong job) error = nil, want mismatch failure")
	}
	if !strings.Contains(err.Error(), "does not match the active job") {
		t.Fatalf("ProcessDirect(SET_STEP wrong job) error = %q, want job mismatch", err)
	}
	if _, statErr := os.Stat(controlFile); !os.IsNotExist(statErr) {
		t.Fatalf("Stat(controlFile) error = %v, want not exists", statErr)
	}

	ec, ok := ag.ActiveMissionStep()
	if !ok || ec.Step == nil {
		t.Fatalf("ActiveMissionStep() = %#v, want unchanged active step", ec)
	}
	if ec.Step.ID != "build" {
		t.Fatalf("ActiveMissionStep().Step.ID = %q, want %q", ec.Step.ID, "build")
	}
}

func TestMissionOperatorSetStepCommandInvalidStepRejectsDeterministically(t *testing.T) {
	ag := newMissionBootstrapTestLoop()
	cmd := newMissionBootstrapTestCommand()
	job := testMissionBootstrapJob()
	missionFile := writeMissionBootstrapJobFile(t, job)
	statusFile := filepath.Join(t.TempDir(), "status.json")
	controlFile := filepath.Join(t.TempDir(), "control.json")

	if err := cmd.Flags().Set("mission-file", missionFile); err != nil {
		t.Fatalf("Flags().Set(mission-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step", "build"); err != nil {
		t.Fatalf("Flags().Set(mission-step) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-status-file", statusFile); err != nil {
		t.Fatalf("Flags().Set(mission-status-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step-control-file", controlFile); err != nil {
		t.Fatalf("Flags().Set(mission-step-control-file) error = %v", err)
	}

	bootstrappedJob := configureMissionBootstrapJobForStartupTest(t, cmd, ag)
	installMissionOperatorSetStepHook(cmd, ag, &bootstrappedJob, true)

	_, err := ag.ProcessDirect("SET_STEP job-1 missing", 2*time.Second)
	if err == nil {
		t.Fatal("ProcessDirect(SET_STEP missing step) error = nil, want validation failure")
	}
	if !strings.Contains(err.Error(), `step "missing" not found in plan`) {
		t.Fatalf("ProcessDirect(SET_STEP missing step) error = %q, want unknown-step rejection", err)
	}
	if _, statErr := os.Stat(controlFile); !os.IsNotExist(statErr) {
		t.Fatalf("Stat(controlFile) error = %v, want not exists", statErr)
	}
}

func TestMissionOperatorSetStepCommandStaleMatchingStatusSnapshotDoesNotConfirmSuccess(t *testing.T) {
	ag := newMissionBootstrapTestLoop()
	cmd := newMissionBootstrapTestCommand()
	job := testMissionBootstrapJob()
	missionFile := writeMissionBootstrapJobFile(t, job)
	statusFile := filepath.Join(t.TempDir(), "status.json")
	controlFile := filepath.Join(t.TempDir(), "control.json")

	if err := cmd.Flags().Set("mission-file", missionFile); err != nil {
		t.Fatalf("Flags().Set(mission-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step", "build"); err != nil {
		t.Fatalf("Flags().Set(mission-step) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-status-file", statusFile); err != nil {
		t.Fatalf("Flags().Set(mission-status-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step-control-file", controlFile); err != nil {
		t.Fatalf("Flags().Set(mission-step-control-file) error = %v", err)
	}

	bootstrappedJob := configureMissionBootstrapJobForStartupTest(t, cmd, ag)
	ag.SetOperatorSetStepHook(newMissionOperatorSetStepHook(cmd, ag, &bootstrappedJob, false, 150*time.Millisecond))
	writeMissionStatusSnapshotFile(t, statusFile, missionStatusSnapshot{
		Active:       true,
		MissionFile:  missionFile,
		JobID:        job.ID,
		StepID:       "final",
		StepType:     string(missioncontrol.StepTypeFinalResponse),
		AllowedTools: []string{"read"},
		UpdatedAt:    "2026-03-19T12:00:00Z",
	})

	_, err := ag.ProcessDirect("SET_STEP job-1 final", 2*time.Second)
	if err == nil {
		t.Fatal("ProcessDirect(SET_STEP stale status) error = nil, want confirmation timeout")
	}
	if !strings.Contains(err.Error(), "want a fresh matching update") {
		t.Fatalf("ProcessDirect(SET_STEP stale status) error = %q, want stale snapshot rejection", err)
	}
}

func TestMissionOperatorSetStepCommandFreshStatusWithWrongStepTypeDoesNotConfirmSuccess(t *testing.T) {
	ag := newMissionBootstrapTestLoop()
	cmd := newMissionBootstrapTestCommand()
	job := testMissionBootstrapJob()
	missionFile := writeMissionBootstrapJobFile(t, job)
	statusFile := filepath.Join(t.TempDir(), "status.json")
	controlFile := filepath.Join(t.TempDir(), "control.json")

	if err := cmd.Flags().Set("mission-file", missionFile); err != nil {
		t.Fatalf("Flags().Set(mission-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step", "build"); err != nil {
		t.Fatalf("Flags().Set(mission-step) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-status-file", statusFile); err != nil {
		t.Fatalf("Flags().Set(mission-status-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step-control-file", controlFile); err != nil {
		t.Fatalf("Flags().Set(mission-step-control-file) error = %v", err)
	}

	bootstrappedJob := configureMissionBootstrapJobForStartupTest(t, cmd, ag)
	ag.SetOperatorSetStepHook(newMissionOperatorSetStepHook(cmd, ag, &bootstrappedJob, false, 150*time.Millisecond))
	writeMissionStatusSnapshotFile(t, statusFile, missionStatusSnapshot{
		Active:       true,
		MissionFile:  missionFile,
		JobID:        job.ID,
		StepID:       "build",
		StepType:     string(missioncontrol.StepTypeOneShotCode),
		AllowedTools: []string{"read"},
		UpdatedAt:    "2026-03-19T12:00:00Z",
	})

	go func() {
		time.Sleep(50 * time.Millisecond)
		writeMissionStatusSnapshotFile(t, statusFile, missionStatusSnapshot{
			Active:       true,
			MissionFile:  missionFile,
			JobID:        job.ID,
			StepID:       "final",
			StepType:     string(missioncontrol.StepTypeDiscussion),
			AllowedTools: []string{"read"},
			UpdatedAt:    "2026-03-19T12:00:01Z",
		})
	}()

	_, err := ag.ProcessDirect("SET_STEP job-1 final", 2*time.Second)
	if err == nil {
		t.Fatal("ProcessDirect(SET_STEP wrong step type) error = nil, want confirmation failure")
	}
	if !strings.Contains(err.Error(), `has step_type="discussion", want step_type="final_response"`) {
		t.Fatalf("ProcessDirect(SET_STEP wrong step type) error = %q, want step_type mismatch", err)
	}
}

func TestMissionOperatorSetStepCommandFreshStatusWithWrongAllowedToolsDoesNotConfirmSuccess(t *testing.T) {
	ag := newMissionBootstrapTestLoop()
	cmd := newMissionBootstrapTestCommand()
	job := testMissionBootstrapJob()
	missionFile := writeMissionBootstrapJobFile(t, job)
	statusFile := filepath.Join(t.TempDir(), "status.json")
	controlFile := filepath.Join(t.TempDir(), "control.json")

	if err := cmd.Flags().Set("mission-file", missionFile); err != nil {
		t.Fatalf("Flags().Set(mission-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step", "build"); err != nil {
		t.Fatalf("Flags().Set(mission-step) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-status-file", statusFile); err != nil {
		t.Fatalf("Flags().Set(mission-status-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step-control-file", controlFile); err != nil {
		t.Fatalf("Flags().Set(mission-step-control-file) error = %v", err)
	}

	bootstrappedJob := configureMissionBootstrapJobForStartupTest(t, cmd, ag)
	ag.SetOperatorSetStepHook(newMissionOperatorSetStepHook(cmd, ag, &bootstrappedJob, false, 150*time.Millisecond))
	writeMissionStatusSnapshotFile(t, statusFile, missionStatusSnapshot{
		Active:       true,
		MissionFile:  missionFile,
		JobID:        job.ID,
		StepID:       "build",
		StepType:     string(missioncontrol.StepTypeOneShotCode),
		AllowedTools: []string{"read"},
		UpdatedAt:    "2026-03-19T12:00:00Z",
	})

	go func() {
		time.Sleep(50 * time.Millisecond)
		writeMissionStatusSnapshotFile(t, statusFile, missionStatusSnapshot{
			Active:       true,
			MissionFile:  missionFile,
			JobID:        job.ID,
			StepID:       "final",
			StepType:     string(missioncontrol.StepTypeFinalResponse),
			AllowedTools: []string{},
			UpdatedAt:    "2026-03-19T12:00:01Z",
		})
	}()

	_, err := ag.ProcessDirect("SET_STEP job-1 final", 2*time.Second)
	if err == nil {
		t.Fatal("ProcessDirect(SET_STEP wrong allowed tools) error = nil, want confirmation failure")
	}
	if !strings.Contains(err.Error(), `has allowed_tools=[], want allowed_tools=["read"]`) {
		t.Fatalf("ProcessDirect(SET_STEP wrong allowed tools) error = %q, want allowed_tools mismatch", err)
	}
}

func TestMissionOperatorSetStepCommandFreshMatchingStatusSnapshotConfirmsSuccess(t *testing.T) {
	ag := newMissionBootstrapTestLoop()
	cmd := newMissionBootstrapTestCommand()
	job := testMissionBootstrapJob()
	missionFile := writeMissionBootstrapJobFile(t, job)
	statusFile := filepath.Join(t.TempDir(), "status.json")
	controlFile := filepath.Join(t.TempDir(), "control.json")

	if err := cmd.Flags().Set("mission-file", missionFile); err != nil {
		t.Fatalf("Flags().Set(mission-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step", "build"); err != nil {
		t.Fatalf("Flags().Set(mission-step) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-status-file", statusFile); err != nil {
		t.Fatalf("Flags().Set(mission-status-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step-control-file", controlFile); err != nil {
		t.Fatalf("Flags().Set(mission-step-control-file) error = %v", err)
	}

	bootstrappedJob := configureMissionBootstrapJobForStartupTest(t, cmd, ag)
	ag.SetOperatorSetStepHook(newMissionOperatorSetStepHook(cmd, ag, &bootstrappedJob, false, 500*time.Millisecond))
	writeMissionStatusSnapshotFile(t, statusFile, missionStatusSnapshot{
		Active:       true,
		MissionFile:  missionFile,
		JobID:        job.ID,
		StepID:       "build",
		StepType:     string(missioncontrol.StepTypeOneShotCode),
		AllowedTools: []string{"read"},
		UpdatedAt:    "2026-03-19T12:00:00Z",
	})

	go func() {
		time.Sleep(50 * time.Millisecond)
		writeMissionStatusSnapshotFile(t, statusFile, missionStatusSnapshot{
			Active:       true,
			MissionFile:  missionFile,
			JobID:        job.ID,
			StepID:       "final",
			StepType:     string(missioncontrol.StepTypeFinalResponse),
			AllowedTools: []string{"read"},
			UpdatedAt:    "2026-03-19T12:00:01Z",
		})
	}()

	resp, err := ag.ProcessDirect("SET_STEP job-1 final", 2*time.Second)
	if err != nil {
		t.Fatalf("ProcessDirect(SET_STEP fresh status) error = %v", err)
	}
	if resp != "Set step job=job-1 step=final." {
		t.Fatalf("ProcessDirect(SET_STEP fresh status) response = %q, want set-step acknowledgement", resp)
	}
}

func TestMissionOperatorSetStepCommandRehydratedRuntimeSucceedsWhenAppropriate(t *testing.T) {
	ag := newMissionBootstrapTestLoop()
	cmd := newMissionBootstrapTestCommand()
	job := testMissionBootstrapJob()
	missionFile := writeMissionBootstrapJobFile(t, job)
	statusFile := filepath.Join(t.TempDir(), "status.json")
	controlFile := filepath.Join(t.TempDir(), "control.json")
	runtimeControl := runtimeControlForBootstrapStep(t, job, "build")

	writeMissionStatusSnapshotFile(t, statusFile, missionStatusSnapshot{
		Active:         true,
		MissionFile:    missionFile,
		JobID:          job.ID,
		StepID:         "build",
		RuntimeControl: runtimeControl,
		Runtime: &missioncontrol.JobRuntimeState{
			JobID:        job.ID,
			State:        missioncontrol.JobStatePaused,
			ActiveStepID: "build",
			PausedReason: missioncontrol.RuntimePauseReasonOperatorCommand,
		},
		UpdatedAt: "2026-03-19T12:00:00Z",
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
	if err := cmd.Flags().Set("mission-step-control-file", controlFile); err != nil {
		t.Fatalf("Flags().Set(mission-step-control-file) error = %v", err)
	}

	bootstrappedJob := configureMissionBootstrapJobForStartupTest(t, cmd, ag)
	installMissionOperatorSetStepHook(cmd, ag, &bootstrappedJob, true)

	resp, err := ag.ProcessDirect("SET_STEP job-1 final", 2*time.Second)
	if err != nil {
		t.Fatalf("ProcessDirect(SET_STEP rehydrated runtime) error = %v", err)
	}
	if resp != "Set step job=job-1 step=final." {
		t.Fatalf("ProcessDirect(SET_STEP rehydrated runtime) response = %q, want set-step acknowledgement", resp)
	}

	ec, ok := ag.ActiveMissionStep()
	if !ok || ec.Step == nil {
		t.Fatalf("ActiveMissionStep() = %#v, want active final step", ec)
	}
	if ec.Step.ID != "final" {
		t.Fatalf("ActiveMissionStep().Step.ID = %q, want %q", ec.Step.ID, "final")
	}
}

func TestMissionOperatorSetStepCommandRehydratedRuntimeRejectsPreviouslyCompletedStepReplay(t *testing.T) {
	ag := newMissionBootstrapTestLoop()
	cmd := newMissionBootstrapTestCommand()
	job := testMissionBootstrapJob()
	missionFile := writeMissionBootstrapJobFile(t, job)
	statusFile := filepath.Join(t.TempDir(), "status.json")
	controlFile := filepath.Join(t.TempDir(), "control.json")
	runtimeControl := runtimeControlForBootstrapStep(t, job, "final")

	writeMissionStatusSnapshotFile(t, statusFile, missionStatusSnapshot{
		Active:         true,
		MissionFile:    missionFile,
		JobID:          job.ID,
		StepID:         "final",
		RuntimeControl: runtimeControl,
		Runtime: &missioncontrol.JobRuntimeState{
			JobID:        job.ID,
			State:        missioncontrol.JobStatePaused,
			ActiveStepID: "final",
			PausedReason: missioncontrol.RuntimePauseReasonOperatorCommand,
			CompletedSteps: []missioncontrol.RuntimeStepRecord{
				{StepID: "build", At: time.Date(2026, 3, 19, 11, 30, 0, 0, time.UTC)},
			},
		},
		UpdatedAt: "2026-03-19T12:00:00Z",
	})

	if err := cmd.Flags().Set("mission-file", missionFile); err != nil {
		t.Fatalf("Flags().Set(mission-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step", "final"); err != nil {
		t.Fatalf("Flags().Set(mission-step) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-status-file", statusFile); err != nil {
		t.Fatalf("Flags().Set(mission-status-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step-control-file", controlFile); err != nil {
		t.Fatalf("Flags().Set(mission-step-control-file) error = %v", err)
	}

	bootstrappedJob := configureMissionBootstrapJobForStartupTest(t, cmd, ag)
	installMissionOperatorSetStepHook(cmd, ag, &bootstrappedJob, true)

	_, err := ag.ProcessDirect("SET_STEP job-1 build", 2*time.Second)
	if err == nil {
		t.Fatal("ProcessDirect(SET_STEP completed step) error = nil, want replay rejection")
	}
	if !strings.Contains(err.Error(), `step "build" is already recorded as completed in runtime state`) {
		t.Fatalf("ProcessDirect(SET_STEP completed step) error = %q, want completed-step replay rejection", err)
	}

	if _, ok := ag.ActiveMissionStep(); ok {
		t.Fatal("ActiveMissionStep() ok = true, want rehydrated control context without live execution context")
	}
	runtime, ok := ag.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want persisted runtime state")
	}
	if runtime.ActiveStepID != "final" {
		t.Fatalf("MissionRuntimeState().ActiveStepID = %q, want %q", runtime.ActiveStepID, "final")
	}
	if len(runtime.CompletedSteps) != 1 || runtime.CompletedSteps[0].StepID != "build" {
		t.Fatalf("MissionRuntimeState().CompletedSteps = %#v, want preserved build completion", runtime.CompletedSteps)
	}
}

func TestMissionOperatorSetStepCommandRehydratedRuntimeRejectsPreviouslyFailedStepReplay(t *testing.T) {
	ag := newMissionBootstrapTestLoop()
	cmd := newMissionBootstrapTestCommand()
	job := testMissionBootstrapJob()
	missionFile := writeMissionBootstrapJobFile(t, job)
	statusFile := filepath.Join(t.TempDir(), "status.json")
	controlFile := filepath.Join(t.TempDir(), "control.json")
	runtimeControl := runtimeControlForBootstrapStep(t, job, "final")

	writeMissionStatusSnapshotFile(t, statusFile, missionStatusSnapshot{
		Active:         true,
		MissionFile:    missionFile,
		JobID:          job.ID,
		StepID:         "final",
		RuntimeControl: runtimeControl,
		Runtime: &missioncontrol.JobRuntimeState{
			JobID:        job.ID,
			State:        missioncontrol.JobStatePaused,
			ActiveStepID: "final",
			PausedReason: missioncontrol.RuntimePauseReasonOperatorCommand,
			FailedSteps: []missioncontrol.RuntimeStepRecord{
				{StepID: "build", Reason: "validator failed", At: time.Date(2026, 3, 19, 11, 30, 0, 0, time.UTC)},
			},
		},
		UpdatedAt: "2026-03-19T12:00:00Z",
	})

	if err := cmd.Flags().Set("mission-file", missionFile); err != nil {
		t.Fatalf("Flags().Set(mission-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step", "final"); err != nil {
		t.Fatalf("Flags().Set(mission-step) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-status-file", statusFile); err != nil {
		t.Fatalf("Flags().Set(mission-status-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step-control-file", controlFile); err != nil {
		t.Fatalf("Flags().Set(mission-step-control-file) error = %v", err)
	}

	bootstrappedJob := configureMissionBootstrapJobForStartupTest(t, cmd, ag)
	installMissionOperatorSetStepHook(cmd, ag, &bootstrappedJob, true)

	_, err := ag.ProcessDirect("SET_STEP job-1 build", 2*time.Second)
	if err == nil {
		t.Fatal("ProcessDirect(SET_STEP failed step) error = nil, want replay rejection")
	}
	if !strings.Contains(err.Error(), `step "build" is already recorded as failed in runtime state`) {
		t.Fatalf("ProcessDirect(SET_STEP failed step) error = %q, want failed-step replay rejection", err)
	}

	if _, ok := ag.ActiveMissionStep(); ok {
		t.Fatal("ActiveMissionStep() ok = true, want rehydrated control context without live execution context")
	}
	runtime, ok := ag.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want persisted runtime state")
	}
	if runtime.ActiveStepID != "final" {
		t.Fatalf("MissionRuntimeState().ActiveStepID = %q, want %q", runtime.ActiveStepID, "final")
	}
	if len(runtime.FailedSteps) != 1 || runtime.FailedSteps[0].StepID != "build" {
		t.Fatalf("MissionRuntimeState().FailedSteps = %#v, want preserved build failure", runtime.FailedSteps)
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
