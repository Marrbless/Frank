package missioncontrol

func LoadMissionStatusObservationFile(path string) ([]byte, error) {
	return loadMissionStatusObservationFile(path, false)
}

func LoadMissionStatusObservation(path string) (MissionStatusSnapshot, error) {
	snapshot, _, err := loadMissionStatusSnapshot(path, true)
	if err != nil {
		return MissionStatusSnapshot{}, err
	}
	return snapshot, nil
}

func loadMissionStatusObservationFile(path string, missingAsNotFound bool) ([]byte, error) {
	_, data, err := loadMissionStatusSnapshot(path, missingAsNotFound)
	if err != nil {
		return nil, err
	}
	return data, nil
}
