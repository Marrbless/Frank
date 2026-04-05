package missioncontrol

import (
	"encoding/json"
	"fmt"
	"os"
)

func LoadMissionStatusObservationFile(path string) ([]byte, error) {
	return loadMissionStatusObservationFile(path, false)
}

func LoadMissionStatusObservation(path string) (MissionStatusSnapshot, error) {
	data, err := loadMissionStatusObservationFile(path, true)
	if err != nil {
		return MissionStatusSnapshot{}, err
	}

	var snapshot MissionStatusSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return MissionStatusSnapshot{}, fmt.Errorf("failed to decode mission status file %q: %w", path, err)
	}

	return snapshot, nil
}

func loadMissionStatusObservationFile(path string, missingAsNotFound bool) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if missingAsNotFound && os.IsNotExist(err) {
			return nil, fmt.Errorf("mission status file %q not found", path)
		}
		return nil, fmt.Errorf("failed to read mission status file %q: %w", path, err)
	}

	var snapshot any
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return nil, fmt.Errorf("failed to decode mission status file %q: %w", path, err)
	}

	return data, nil
}
