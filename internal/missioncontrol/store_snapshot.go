package missioncontrol

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

var missionStatusSnapshotReadFile = os.ReadFile
var missionStatusSnapshotDecode = decodeMissionStatusSnapshotJSON
var missionStatusSnapshotWriteFileAtomic = writeMissionStatusSnapshotFileAtomic

func WriteMissionStatusSnapshotAtomic(path string, snapshot MissionStatusSnapshot) error {
	return writeMissionStatusSnapshotJSONAtomic(path, snapshot, "failed to encode mission status snapshot", "failed to write mission status snapshot")
}

func WriteProjectedMissionStatusSnapshot(path, root, jobID string, opts MissionStatusSnapshotOptions) error {
	snapshot, err := BuildCommittedMissionStatusSnapshot(root, jobID, opts)
	if err != nil {
		return err
	}
	return WriteMissionStatusSnapshotAtomic(path, snapshot)
}

func writeMissionStatusSnapshotJSONAtomic(path string, value any, encodeErrPrefix string, writeErrPrefix string) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("%s %q: %w", encodeErrPrefix, path, err)
	}
	data = append(data, '\n')

	if err := missionStatusSnapshotWriteFileAtomic(path, data); err != nil {
		return fmt.Errorf("%s %q: %w", writeErrPrefix, path, err)
	}

	return nil
}

func writeMissionStatusSnapshotFileAtomic(path string, data []byte) (err error) {
	dir := filepath.Dir(path)
	tempFile, err := os.CreateTemp(dir, filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}

	tempPath := tempFile.Name()
	defer func() {
		if err == nil {
			return
		}
		if closeErr := tempFile.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
		_ = os.Remove(tempPath)
	}()

	if _, err = tempFile.Write(data); err != nil {
		return err
	}
	if err = tempFile.Close(); err != nil {
		return err
	}
	if err = os.Rename(tempPath, path); err != nil {
		return err
	}

	return nil
}

func loadMissionStatusSnapshot(path string, missingAsNotFound bool) (MissionStatusSnapshot, []byte, error) {
	data, err := missionStatusSnapshotReadFile(path)
	if err != nil {
		if missingAsNotFound && os.IsNotExist(err) {
			return MissionStatusSnapshot{}, nil, fmt.Errorf("mission status file %q not found", path)
		}
		return MissionStatusSnapshot{}, nil, fmt.Errorf("failed to read mission status file %q: %w", path, err)
	}

	snapshot, err := missionStatusSnapshotDecode(path, data)
	if err != nil {
		return MissionStatusSnapshot{}, nil, err
	}

	return snapshot, data, nil
}

func decodeMissionStatusSnapshotJSON(path string, data []byte) (MissionStatusSnapshot, error) {
	var snapshot MissionStatusSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return MissionStatusSnapshot{}, fmt.Errorf("failed to decode mission status file %q: %w", path, err)
	}
	return snapshot, nil
}
