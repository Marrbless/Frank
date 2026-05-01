package missioncontrol

import (
	"encoding/json"
	"fmt"
)

// GatewayStatusSnapshot is the canonical read-only gateway-status.json envelope.
// It is a sanitized projection of the mission status snapshot contract and does
// not expose raw runtime state or runtime control records.
type GatewayStatusSnapshot struct {
	MissionRequired   bool                               `json:"mission_required"`
	Active            bool                               `json:"active"`
	MissionFile       string                             `json:"mission_file"`
	JobID             string                             `json:"job_id"`
	StepID            string                             `json:"step_id"`
	StepType          string                             `json:"step_type"`
	RequiredAuthority AuthorityTier                      `json:"required_authority"`
	RequiresApproval  bool                               `json:"requires_approval"`
	AllowedTools      []string                           `json:"allowed_tools"`
	Model             *OperatorModelRouteStatus          `json:"model,omitempty"`
	ModelMetrics      *OperatorModelControlMetricsStatus `json:"model_metrics,omitempty"`
	UpdatedAt         string                             `json:"updated_at"`
}

type GatewayStatusSnapshotOptions = MissionStatusSnapshotOptions

func ProjectGatewayStatusSnapshot(snapshot MissionStatusSnapshot) GatewayStatusSnapshot {
	return GatewayStatusSnapshot{
		MissionRequired:   snapshot.MissionRequired,
		Active:            snapshot.Active,
		MissionFile:       snapshot.MissionFile,
		JobID:             snapshot.JobID,
		StepID:            snapshot.StepID,
		StepType:          snapshot.StepType,
		RequiredAuthority: snapshot.RequiredAuthority,
		RequiresApproval:  snapshot.RequiresApproval,
		AllowedTools:      append([]string(nil), snapshot.AllowedTools...),
		Model:             CloneOperatorModelRouteStatus(snapshot.Model),
		ModelMetrics:      CloneOperatorModelControlMetricsStatus(snapshot.ModelMetrics),
		UpdatedAt:         snapshot.UpdatedAt,
	}
}

func BuildCommittedGatewayStatusSnapshot(root, jobID string, opts GatewayStatusSnapshotOptions) (GatewayStatusSnapshot, error) {
	snapshot, err := BuildCommittedMissionStatusSnapshot(root, jobID, MissionStatusSnapshotOptions(opts))
	if err != nil {
		return GatewayStatusSnapshot{}, err
	}
	return ProjectGatewayStatusSnapshot(snapshot), nil
}

func WriteGatewayStatusSnapshotAtomic(path string, snapshot GatewayStatusSnapshot) error {
	return writeMissionStatusSnapshotJSONAtomic(path, snapshot, "failed to encode gateway status snapshot", "failed to write gateway status snapshot")
}

func WriteProjectedGatewayStatusSnapshot(path, root, jobID string, opts GatewayStatusSnapshotOptions) error {
	snapshot, err := BuildCommittedGatewayStatusSnapshot(root, jobID, opts)
	if err != nil {
		return err
	}
	return WriteGatewayStatusSnapshotAtomic(path, snapshot)
}

func LoadGatewayStatusObservation(path string) (GatewayStatusSnapshot, error) {
	snapshot, err := LoadMissionStatusObservation(path)
	if err != nil {
		return GatewayStatusSnapshot{}, err
	}
	return ProjectGatewayStatusSnapshot(snapshot), nil
}

func LoadGatewayStatusObservationFile(path string) ([]byte, error) {
	snapshot, err := LoadGatewayStatusObservation(path)
	if err != nil {
		return nil, err
	}
	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to encode gateway status file %q: %w", path, err)
	}
	data = append(data, '\n')
	return data, nil
}
