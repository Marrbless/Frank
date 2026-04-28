package missioncontrol

import (
	"fmt"
	"strings"
	"time"
)

const (
	WorkspaceRunnerProfilePhone      = "phone"
	WorkspaceRunnerProfileDesktopDev = "desktop_dev"
)

type WorkspaceRunnerHostCapabilities struct {
	HostProfile               string `json:"host_profile"`
	LocalWorkspaceAvailable   bool   `json:"local_workspace_available"`
	FakePhoneProfileAvailable bool   `json:"fake_phone_profile_available"`
	NetworkDisabled           bool   `json:"network_disabled"`
	ExternalServicesDisabled  bool   `json:"external_services_disabled"`
}

type WorkspaceRunnerProfileAssessment struct {
	ProfileName   string   `json:"profile_name"`
	ExecutionHost string   `json:"execution_host"`
	Ready         bool     `json:"ready"`
	Blockers      []string `json:"blockers,omitempty"`
}

type DeterministicWorkspaceRunRequest struct {
	RunID        string                          `json:"run_id"`
	ProfileName  string                          `json:"profile_name"`
	Capabilities WorkspaceRunnerHostCapabilities `json:"capabilities"`
	StartedAt    time.Time                       `json:"started_at"`
	CreatedBy    string                          `json:"created_by"`
}

func AssessWorkspaceRunnerProfile(profileName string, capabilities WorkspaceRunnerHostCapabilities) WorkspaceRunnerProfileAssessment {
	profileName = strings.TrimSpace(profileName)
	capabilities = NormalizeWorkspaceRunnerHostCapabilities(capabilities)
	assessment := WorkspaceRunnerProfileAssessment{
		ProfileName: profileName,
	}
	switch profileName {
	case WorkspaceRunnerProfilePhone:
		assessment.ExecutionHost = ExecutionHostPhone
		if !capabilities.FakePhoneProfileAvailable {
			assessment.Blockers = append(assessment.Blockers, "fake_phone_profile_available is required")
		}
	case WorkspaceRunnerProfileDesktopDev:
		assessment.ExecutionHost = ExecutionHostDesktopDev
	default:
		assessment.Blockers = append(assessment.Blockers, fmt.Sprintf("workspace runner profile %q is unsupported", profileName))
	}
	if !capabilities.LocalWorkspaceAvailable {
		assessment.Blockers = append(assessment.Blockers, "local_workspace_available is required")
	}
	if !capabilities.NetworkDisabled {
		assessment.Blockers = append(assessment.Blockers, "network_disabled is required")
	}
	if !capabilities.ExternalServicesDisabled {
		assessment.Blockers = append(assessment.Blockers, "external_services_disabled is required")
	}
	assessment.Ready = len(assessment.Blockers) == 0
	return assessment
}

func NormalizeWorkspaceRunnerHostCapabilities(capabilities WorkspaceRunnerHostCapabilities) WorkspaceRunnerHostCapabilities {
	capabilities.HostProfile = strings.TrimSpace(capabilities.HostProfile)
	return capabilities
}

func DeterministicWorkspaceRunID(runID, profileName string, startedAt time.Time) string {
	startedAt = startedAt.UTC()
	startedAtCompact := strings.ReplaceAll(startedAt.Format("20060102T150405.000000000Z"), ".", "")
	return "workspace-run-" + strings.TrimSpace(runID) + "-" + strings.TrimSpace(profileName) + "-" + startedAtCompact
}

func RunDeterministicImprovementWorkspace(root string, request DeterministicWorkspaceRunRequest) (ImprovementWorkspaceRunRecord, WorkspaceRunnerProfileAssessment, bool, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return ImprovementWorkspaceRunRecord{}, WorkspaceRunnerProfileAssessment{}, false, err
	}
	request.RunID = strings.TrimSpace(request.RunID)
	request.ProfileName = strings.TrimSpace(request.ProfileName)
	request.CreatedBy = strings.TrimSpace(request.CreatedBy)
	request.StartedAt = request.StartedAt.UTC()
	if request.StartedAt.IsZero() {
		return ImprovementWorkspaceRunRecord{}, WorkspaceRunnerProfileAssessment{}, false, fmt.Errorf("deterministic workspace run started_at is required")
	}
	if request.CreatedBy == "" {
		return ImprovementWorkspaceRunRecord{}, WorkspaceRunnerProfileAssessment{}, false, fmt.Errorf("deterministic workspace run created_by is required")
	}
	assessment := AssessWorkspaceRunnerProfile(request.ProfileName, request.Capabilities)
	if !assessment.Ready {
		return ImprovementWorkspaceRunRecord{}, assessment, false, fmt.Errorf("workspace runner profile %q is not ready: %s", request.ProfileName, strings.Join(assessment.Blockers, "; "))
	}
	run, err := LoadImprovementRunRecord(root, request.RunID)
	if err != nil {
		return ImprovementWorkspaceRunRecord{}, assessment, false, err
	}
	if run.ExecutionHost != assessment.ExecutionHost {
		return ImprovementWorkspaceRunRecord{}, assessment, false, fmt.Errorf("improvement run %q execution_host %q does not match workspace runner profile host %q", run.RunID, run.ExecutionHost, assessment.ExecutionHost)
	}
	pointer, err := LoadActiveRuntimePackPointer(root)
	if err != nil {
		return ImprovementWorkspaceRunRecord{}, assessment, false, err
	}
	snapshot := ImprovementWorkspaceActivePointerSnapshotFromPointer(pointer)
	record := ImprovementWorkspaceRunRecord{
		WorkspaceRunID:            DeterministicWorkspaceRunID(run.RunID, request.ProfileName, request.StartedAt),
		RunID:                     run.RunID,
		CandidateID:               run.CandidateID,
		ExecutionHost:             assessment.ExecutionHost,
		Outcome:                   ImprovementWorkspaceRunOutcomeSucceeded,
		StartedAt:                 request.StartedAt,
		CompletedAt:               request.StartedAt,
		ActivePointerAtStart:      snapshot,
		ActivePointerAtCompletion: snapshot,
		CreatedAt:                 request.StartedAt,
		CreatedBy:                 request.CreatedBy,
	}
	stored, changed, err := StoreImprovementWorkspaceRunRecord(root, record)
	if err != nil {
		return ImprovementWorkspaceRunRecord{}, assessment, false, err
	}
	return stored, assessment, changed, nil
}
