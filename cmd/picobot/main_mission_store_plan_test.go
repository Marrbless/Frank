package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/local/picobot/internal/missioncontrol"
)

func TestMissionStoreTransferPlanCommandIsDryRun(t *testing.T) {
	root := t.TempDir()
	snapshotRoot := filepath.Join(t.TempDir(), "snapshot")
	if err := os.WriteFile(missioncontrol.StoreManifestPath(root), []byte("{}\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(manifest) error = %v", err)
	}

	cmd := NewRootCmd()
	out := &bytes.Buffer{}
	cmd.SetOut(out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{
		"mission",
		"store-transfer-plan",
		"--direction",
		"backup",
		"--mission-store-root",
		root,
		"--snapshot-root",
		snapshotRoot,
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	var plan missioncontrol.StoreTransferPlan
	if err := json.Unmarshal(out.Bytes(), &plan); err != nil {
		t.Fatalf("json.Unmarshal(stdout) error = %v", err)
	}
	if plan.Direction != missioncontrol.StoreTransferPlanDirectionBackup {
		t.Fatalf("plan.Direction = %q, want backup", plan.Direction)
	}
	if !plan.DryRun {
		t.Fatal("plan.DryRun = false, want true")
	}
	if plan.SourceRoot != root || plan.DestinationRoot != snapshotRoot {
		t.Fatalf("plan roots = source %q destination %q, want source %q destination %q", plan.SourceRoot, plan.DestinationRoot, root, snapshotRoot)
	}
	if plan.CopyableEntries != 1 || plan.BlockedEntries != 0 {
		t.Fatalf("plan counts = copyable %d blocked %d, want copyable 1 blocked 0", plan.CopyableEntries, plan.BlockedEntries)
	}
	if _, err := os.Stat(snapshotRoot); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("snapshot root stat error = %v, want not exist because command is dry-run", err)
	}
}

func TestMissionStoreTransferPlanCommandRequiresDirection(t *testing.T) {
	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{
		"mission",
		"store-transfer-plan",
		"--mission-store-root",
		t.TempDir(),
		"--snapshot-root",
		t.TempDir(),
	})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want required direction error")
	}
	if err.Error() != "--direction is required" {
		t.Fatalf("Execute() error = %q, want required direction error", err.Error())
	}
}
