package missioncontrol

import (
	"errors"
	"testing"
	"time"
)

func TestInitStoreManifestCreatesReadyManifest(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 4, 15, 0, 0, 0, time.UTC)

	manifest, err := InitStoreManifest(root, now)
	if err != nil {
		t.Fatalf("InitStoreManifest() error = %v", err)
	}
	if manifest.StoreState != StoreStateReady {
		t.Fatalf("StoreState = %q, want %q", manifest.StoreState, StoreStateReady)
	}
	if manifest.SchemaVersion != StoreSchemaVersion {
		t.Fatalf("SchemaVersion = %d, want %d", manifest.SchemaVersion, StoreSchemaVersion)
	}

	loaded, err := LoadStoreManifest(root)
	if err != nil {
		t.Fatalf("LoadStoreManifest() error = %v", err)
	}
	if loaded.StoreID != manifest.StoreID {
		t.Fatalf("LoadStoreManifest().StoreID = %q, want %q", loaded.StoreID, manifest.StoreID)
	}
}

func TestLoadStoreManifestMissingReturnsSentinel(t *testing.T) {
	t.Parallel()

	_, err := LoadStoreManifest(t.TempDir())
	if !errors.Is(err, ErrStoreManifestNotFound) {
		t.Fatalf("LoadStoreManifest() error = %v, want %v", err, ErrStoreManifestNotFound)
	}
}

func TestStoreManifestRecordRejectsImportedStateWithoutImportMetadata(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	manifest := StoreManifest{
		RecordVersion:          StoreRecordVersion,
		SchemaVersion:          StoreSchemaVersion,
		StoreID:                "store-1",
		InitializedAt:          time.Date(2026, 4, 4, 15, 0, 0, 0, time.UTC),
		StoreState:             StoreStateImportedFromSnapshot,
		RetentionPolicyVersion: StoreRetentionVersionV1,
	}

	err := StoreManifestRecord(root, manifest)
	if err == nil {
		t.Fatal("StoreManifestRecord() error = nil, want validation failure")
	}
}

func TestStoreManifestRecordAcceptsImportedStateWithImportMetadata(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 4, 15, 0, 0, 0, time.UTC)
	manifest := StoreManifest{
		RecordVersion:          StoreRecordVersion,
		SchemaVersion:          StoreSchemaVersion,
		StoreID:                "store-1",
		InitializedAt:          now,
		StoreState:             StoreStateImportedFromSnapshot,
		RetentionPolicyVersion: StoreRetentionVersionV1,
		SnapshotImport: StoreSnapshotImportMetadata{
			Imported:                true,
			SourceStatusFile:        "status.json",
			SourceSnapshotUpdatedAt: "2026-04-04T15:00:00Z",
			SourceJobID:             "job-1",
			ImportedAt:              now,
		},
	}

	if err := StoreManifestRecord(root, manifest); err != nil {
		t.Fatalf("StoreManifestRecord() error = %v", err)
	}
}
