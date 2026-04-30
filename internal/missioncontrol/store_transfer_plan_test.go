package missioncontrol

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestBuildStoreTransferPlanBackupListsCopyableFilesAndSkipsTransientLocks(t *testing.T) {
	root := t.TempDir()
	snapshotRoot := t.TempDir()
	if err := os.MkdirAll(StoreLogsDir(root), 0o755); err != nil {
		t.Fatalf("MkdirAll(source logs) error = %v", err)
	}
	if err := os.MkdirAll(StoreLogsDir(snapshotRoot), 0o755); err != nil {
		t.Fatalf("MkdirAll(destination logs) error = %v", err)
	}
	if err := os.WriteFile(StoreManifestPath(root), []byte("{}\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(source manifest) error = %v", err)
	}
	if err := os.WriteFile(StoreCurrentLogPath(root), []byte("source log\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(source current log) error = %v", err)
	}
	if err := os.WriteFile(StoreCurrentLogPath(snapshotRoot), []byte("old log\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(destination current log) error = %v", err)
	}
	if err := os.WriteFile(StoreWriterLockPath(root), []byte("lock\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(writer lock) error = %v", err)
	}

	plan, err := BuildStoreTransferPlan(root, snapshotRoot, StoreTransferPlanDirectionBackup)
	if err != nil {
		t.Fatalf("BuildStoreTransferPlan() error = %v", err)
	}

	if plan.Direction != StoreTransferPlanDirectionBackup {
		t.Fatalf("plan.Direction = %q, want %q", plan.Direction, StoreTransferPlanDirectionBackup)
	}
	if !plan.DryRun {
		t.Fatal("plan.DryRun = false, want true")
	}
	if plan.SourceRoot != root || plan.DestinationRoot != snapshotRoot {
		t.Fatalf("plan roots = source %q destination %q, want source %q destination %q", plan.SourceRoot, plan.DestinationRoot, root, snapshotRoot)
	}

	entries := storeTransferPlanEntriesByRelPath(plan.Entries)
	assertStoreTransferPlanEntry(t, entries["manifest.json"], "file", "copy_file", true, false, false)
	assertStoreTransferPlanEntry(t, entries[filepath.ToSlash(filepath.Join("logs", "current.log"))], "file", "replace_file", true, true, false)
	assertStoreTransferPlanEntry(t, entries["writer.lock"], "file", "skip_transient_lock", false, false, false)

	if plan.CopyableEntries != 2 {
		t.Fatalf("plan.CopyableEntries = %d, want 2", plan.CopyableEntries)
	}
	if plan.SkippedEntries != 2 {
		t.Fatalf("plan.SkippedEntries = %d, want 2 for existing logs dir and writer lock", plan.SkippedEntries)
	}
	if plan.BlockedEntries != 0 {
		t.Fatalf("plan.BlockedEntries = %d, want 0", plan.BlockedEntries)
	}
	if plan.CopyableBytes != int64(len("{}\n")+len("source log\n")) {
		t.Fatalf("plan.CopyableBytes = %d, want copied file byte count", plan.CopyableBytes)
	}
}

func TestBuildStoreTransferPlanRestoreSwapsSourceAndDestinationWithoutCreatingDestination(t *testing.T) {
	missionRoot := filepath.Join(t.TempDir(), "mission-store")
	snapshotRoot := t.TempDir()
	if err := os.WriteFile(StoreManifestPath(snapshotRoot), []byte("{}\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(snapshot manifest) error = %v", err)
	}

	plan, err := BuildStoreTransferPlan(missionRoot, snapshotRoot, StoreTransferPlanDirectionRestore)
	if err != nil {
		t.Fatalf("BuildStoreTransferPlan() error = %v", err)
	}

	if plan.SourceRoot != snapshotRoot || plan.DestinationRoot != missionRoot {
		t.Fatalf("plan roots = source %q destination %q, want source %q destination %q", plan.SourceRoot, plan.DestinationRoot, snapshotRoot, missionRoot)
	}
	entries := storeTransferPlanEntriesByRelPath(plan.Entries)
	assertStoreTransferPlanEntry(t, entries["manifest.json"], "file", "copy_file", true, false, false)

	if _, err := os.Stat(missionRoot); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("mission destination stat error = %v, want not exist because dry-run must not create it", err)
	}
}

func TestBuildStoreTransferPlanBlocksUnsupportedSourceKinds(t *testing.T) {
	root := t.TempDir()
	snapshotRoot := t.TempDir()
	if err := os.Symlink("missing-target", filepath.Join(root, "linked")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}

	plan, err := BuildStoreTransferPlan(root, snapshotRoot, StoreTransferPlanDirectionBackup)
	if err != nil {
		t.Fatalf("BuildStoreTransferPlan() error = %v", err)
	}

	entries := storeTransferPlanEntriesByRelPath(plan.Entries)
	assertStoreTransferPlanEntry(t, entries["linked"], "symlink", "blocked_unsupported_source", false, false, true)
	if plan.BlockedEntries != 1 {
		t.Fatalf("plan.BlockedEntries = %d, want 1", plan.BlockedEntries)
	}
}

func storeTransferPlanEntriesByRelPath(entries []StoreTransferPlanEntry) map[string]StoreTransferPlanEntry {
	byRelPath := make(map[string]StoreTransferPlanEntry, len(entries))
	for _, entry := range entries {
		byRelPath[entry.RelPath] = entry
	}
	return byRelPath
}

func assertStoreTransferPlanEntry(t *testing.T, entry StoreTransferPlanEntry, kind string, action string, wouldWrite bool, wouldReplace bool, blocked bool) {
	t.Helper()

	if entry.RelPath == "" {
		t.Fatal("entry missing from plan")
	}
	if entry.Kind != kind {
		t.Fatalf("entry %q Kind = %q, want %q", entry.RelPath, entry.Kind, kind)
	}
	if entry.Action != action {
		t.Fatalf("entry %q Action = %q, want %q", entry.RelPath, entry.Action, action)
	}
	if entry.WouldWrite != wouldWrite {
		t.Fatalf("entry %q WouldWrite = %t, want %t", entry.RelPath, entry.WouldWrite, wouldWrite)
	}
	if entry.WouldReplace != wouldReplace {
		t.Fatalf("entry %q WouldReplace = %t, want %t", entry.RelPath, entry.WouldReplace, wouldReplace)
	}
	if entry.Blocked != blocked {
		t.Fatalf("entry %q Blocked = %t, want %t", entry.RelPath, entry.Blocked, blocked)
	}
}
