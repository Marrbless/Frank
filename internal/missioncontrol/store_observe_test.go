package missioncontrol

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadMissionStatusObservationFileReturnsExactBytes(t *testing.T) {
	path := filepath.Join(t.TempDir(), "status.json")
	want := []byte("{\n  \"job_id\": \"job-1\"\n}\n")
	if err := os.WriteFile(path, want, 0o644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	got, err := LoadMissionStatusObservationFile(path)
	if err != nil {
		t.Fatalf("LoadMissionStatusObservationFile() error = %v", err)
	}
	if string(got) != string(want) {
		t.Fatalf("LoadMissionStatusObservationFile() = %q, want %q", got, want)
	}
}

func TestLoadMissionStatusObservationFileMissingFilePreservesReadError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.json")

	_, err := LoadMissionStatusObservationFile(path)
	if err == nil {
		t.Fatal("LoadMissionStatusObservationFile() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "failed to read mission status file") {
		t.Fatalf("LoadMissionStatusObservationFile() error = %q, want read failure", err)
	}
}

func TestLoadMissionStatusObservationMissingFileReturnsNotFound(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.json")

	_, err := LoadMissionStatusObservation(path)
	if err == nil {
		t.Fatal("LoadMissionStatusObservation() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "mission status file") || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("LoadMissionStatusObservation() error = %q, want not-found message", err)
	}
}

func TestLoadMissionStatusObservationInvalidJSONReturnsDecodeError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "status.json")
	if err := os.WriteFile(path, []byte("{not-json"), 0o644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	_, err := LoadMissionStatusObservation(path)
	if err == nil {
		t.Fatal("LoadMissionStatusObservation() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "failed to decode mission status file") {
		t.Fatalf("LoadMissionStatusObservation() error = %q, want decode failure", err)
	}
}
