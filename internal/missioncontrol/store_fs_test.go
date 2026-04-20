package missioncontrol

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestWriteStoreFileAtomicWritesFileAndSyncsParentDir(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "child", "record.json")

	originalSyncDir := storeSyncDir
	t.Cleanup(func() { storeSyncDir = originalSyncDir })

	called := false
	storeSyncDir = func(dir string) error {
		called = true
		if dir != filepath.Dir(path) {
			t.Fatalf("storeSyncDir() dir = %q, want %q", dir, filepath.Dir(path))
		}
		return nil
	}

	if err := WriteStoreFileAtomic(path, []byte("{\"ok\":true}\n")); err != nil {
		t.Fatalf("WriteStoreFileAtomic() error = %v", err)
	}
	if !called {
		t.Fatal("storeSyncDir() called = false, want true")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(data) != "{\"ok\":true}\n" {
		t.Fatalf("ReadFile() = %q, want %q", string(data), "{\"ok\":true}\n")
	}
}

func TestWriteStoreFileAtomicFailsClosedWhenParentDirSyncFails(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "record.json")

	originalSyncDir := storeSyncDir
	t.Cleanup(func() { storeSyncDir = originalSyncDir })

	wantErr := errors.New("sync dir failed")
	storeSyncDir = func(string) error { return wantErr }

	err := WriteStoreFileAtomic(path, []byte("{}\n"))
	if !errors.Is(err, wantErr) {
		t.Fatalf("WriteStoreFileAtomic() error = %v, want %v", err, wantErr)
	}
}

func TestWriteStoreFileAtomicModePreservesRequestedFileMode(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "record.json")

	if err := WriteStoreFileAtomicMode(path, []byte("{}\n"), 0o644); err != nil {
		t.Fatalf("WriteStoreFileAtomicMode() error = %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	if got := info.Mode().Perm(); got != 0o644 {
		t.Fatalf("file mode = %#o, want %#o", got, 0o644)
	}
}
