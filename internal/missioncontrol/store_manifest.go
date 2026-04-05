package missioncontrol

import (
	"errors"
	"fmt"
	"os"
	"time"
)

var storeManifestWriteJSONAtomic = WriteStoreJSONAtomic

func LoadStoreManifest(root string) (StoreManifest, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return StoreManifest{}, err
	}

	path := StoreManifestPath(root)
	var manifest StoreManifest
	if err := LoadStoreJSON(path, &manifest); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return StoreManifest{}, ErrStoreManifestNotFound
		}
		return StoreManifest{}, err
	}
	if err := ValidateStoreManifest(manifest); err != nil {
		return StoreManifest{}, err
	}
	return manifest, nil
}

func InitStoreManifest(root string, now time.Time) (StoreManifest, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return StoreManifest{}, err
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return StoreManifest{}, err
	}

	if manifest, err := LoadStoreManifest(root); err == nil {
		return manifest, nil
	} else if !errors.Is(err, ErrStoreManifestNotFound) {
		return StoreManifest{}, err
	}

	manifest := StoreManifest{
		RecordVersion:          StoreRecordVersion,
		SchemaVersion:          StoreSchemaVersion,
		StoreID:                fmt.Sprintf("store-%d", now.UTC().UnixNano()),
		InitializedAt:          now.UTC(),
		StoreState:             StoreStateReady,
		CurrentWriterEpoch:     0,
		RetentionPolicyVersion: StoreRetentionVersionV1,
	}
	if err := StoreManifestRecord(root, manifest); err != nil {
		return StoreManifest{}, err
	}
	return manifest, nil
}

func StoreManifestRecord(root string, manifest StoreManifest) error {
	if err := ValidateStoreRoot(root); err != nil {
		return err
	}
	if err := ValidateStoreManifest(manifest); err != nil {
		return err
	}
	return storeManifestWriteJSONAtomic(StoreManifestPath(root), manifest)
}

func ValidateStoreManifest(manifest StoreManifest) error {
	if manifest.RecordVersion <= 0 {
		return fmt.Errorf("mission store manifest record_version must be positive")
	}
	if manifest.SchemaVersion <= 0 {
		return fmt.Errorf("mission store manifest schema_version must be positive")
	}
	if manifest.StoreID == "" {
		return fmt.Errorf("mission store manifest store_id is required")
	}
	if manifest.InitializedAt.IsZero() {
		return fmt.Errorf("mission store manifest initialized_at is required")
	}
	if err := ValidateStoreState(manifest.StoreState); err != nil {
		return err
	}
	if manifest.RetentionPolicyVersion <= 0 {
		return fmt.Errorf("mission store manifest retention_policy_version must be positive")
	}
	if manifest.SnapshotImport.Imported {
		if manifest.SnapshotImport.ImportedAt.IsZero() {
			return fmt.Errorf("mission store manifest imported snapshot requires imported_at")
		}
		if manifest.StoreState != StoreStateImportedFromSnapshot {
			return fmt.Errorf("mission store manifest imported snapshot requires store_state %q", StoreStateImportedFromSnapshot)
		}
	}
	if manifest.StoreState == StoreStateImportedFromSnapshot && !manifest.SnapshotImport.Imported {
		return fmt.Errorf("mission store manifest store_state %q requires snapshot_import.imported=true", StoreStateImportedFromSnapshot)
	}
	return nil
}
