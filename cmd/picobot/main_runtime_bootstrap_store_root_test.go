package main

import (
	"testing"

	"github.com/local/picobot/internal/missioncontrol"
)

func TestResolveMissionStoreRootPrefersExplicitFlag(t *testing.T) {
	cmd := newMissionBootstrapTestCommand()
	if err := cmd.Flags().Set("mission-store-root", "/tmp/store-root"); err != nil {
		t.Fatalf("Flags().Set(mission-store-root) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-status-file", "/tmp/status.json"); err != nil {
		t.Fatalf("Flags().Set(mission-status-file) error = %v", err)
	}

	got := resolveMissionStoreRoot(cmd)
	if got != "/tmp/store-root" {
		t.Fatalf("resolveMissionStoreRoot() = %q, want %q", got, "/tmp/store-root")
	}
	if got != missioncontrol.ResolveStoreRoot("/tmp/store-root", "/tmp/status.json") {
		t.Fatalf("resolveMissionStoreRoot() = %q, want missioncontrol.ResolveStoreRoot parity", got)
	}
}

func TestResolveMissionStoreRootFallsBackToStatusFile(t *testing.T) {
	cmd := newMissionBootstrapTestCommand()
	if err := cmd.Flags().Set("mission-status-file", "/tmp/status.json"); err != nil {
		t.Fatalf("Flags().Set(mission-status-file) error = %v", err)
	}

	got := resolveMissionStoreRoot(cmd)
	if got != "/tmp/status.json.store" {
		t.Fatalf("resolveMissionStoreRoot() = %q, want %q", got, "/tmp/status.json.store")
	}
	if got != missioncontrol.ResolveStoreRoot("", "/tmp/status.json") {
		t.Fatalf("resolveMissionStoreRoot() = %q, want missioncontrol.ResolveStoreRoot parity", got)
	}
}

func TestResolveMissionStoreRootReturnsEmptyWithoutInputs(t *testing.T) {
	cmd := newMissionBootstrapTestCommand()

	got := resolveMissionStoreRoot(cmd)
	if got != "" {
		t.Fatalf("resolveMissionStoreRoot() = %q, want empty string", got)
	}
	if got != missioncontrol.ResolveStoreRoot("", "") {
		t.Fatalf("resolveMissionStoreRoot() = %q, want missioncontrol.ResolveStoreRoot parity", got)
	}
}
