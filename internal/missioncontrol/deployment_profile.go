package missioncontrol

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"
)

const (
	DeploymentProfilePhoneResident = "phone_resident"
	DeploymentProfileDesktopDev    = "desktop_dev"
)

type DeploymentProfileAssessment struct {
	ProfileName     string   `json:"profile_name"`
	ExecutionHost   string   `json:"execution_host"`
	StrictPhoneOnly bool     `json:"strict_phone_only"`
	Ready           bool     `json:"ready"`
	Blockers        []string `json:"blockers,omitempty"`
}

type DeploymentProfileRecord struct {
	RecordVersion   int                             `json:"record_version"`
	DeploymentID    string                          `json:"deployment_id"`
	ProfileName     string                          `json:"profile_name"`
	ExecutionHost   string                          `json:"execution_host"`
	StrictPhoneOnly bool                            `json:"strict_phone_only"`
	Capabilities    WorkspaceRunnerHostCapabilities `json:"capabilities"`
	Assessment      DeploymentProfileAssessment     `json:"assessment"`
	CreatedAt       time.Time                       `json:"created_at"`
	CreatedBy       string                          `json:"created_by"`
}

var ErrDeploymentProfileRecordNotFound = errors.New("mission store deployment profile record not found")

func StoreDeploymentProfilesDir(root string) string {
	return filepath.Join(root, "deployment", "profiles")
}

func StoreDeploymentProfilePath(root, deploymentID string) string {
	return filepath.Join(StoreDeploymentProfilesDir(root), strings.TrimSpace(deploymentID)+".json")
}

func DeploymentProfileID(profileName, executionHost string, strictPhoneOnly bool) string {
	mode := "flex"
	if strictPhoneOnly {
		mode = "strict-phone"
	}
	return "deployment-profile-" + strings.TrimSpace(profileName) + "-" + strings.TrimSpace(executionHost) + "-" + mode
}

func NormalizeDeploymentProfileAssessment(assessment DeploymentProfileAssessment) DeploymentProfileAssessment {
	assessment.ProfileName = strings.TrimSpace(assessment.ProfileName)
	assessment.ExecutionHost = strings.TrimSpace(assessment.ExecutionHost)
	assessment.Blockers = normalizeAutonomyStrings(assessment.Blockers)
	return assessment
}

func NormalizeDeploymentProfileRecord(record DeploymentProfileRecord) DeploymentProfileRecord {
	record.DeploymentID = strings.TrimSpace(record.DeploymentID)
	record.ProfileName = strings.TrimSpace(record.ProfileName)
	record.ExecutionHost = strings.TrimSpace(record.ExecutionHost)
	record.Capabilities = NormalizeWorkspaceRunnerHostCapabilities(record.Capabilities)
	record.Assessment = NormalizeDeploymentProfileAssessment(record.Assessment)
	record.CreatedAt = record.CreatedAt.UTC()
	record.CreatedBy = strings.TrimSpace(record.CreatedBy)
	return record
}

func AssessDeploymentProfile(profileName, executionHost string, strictPhoneOnly bool, capabilities WorkspaceRunnerHostCapabilities) DeploymentProfileAssessment {
	profileName = strings.TrimSpace(profileName)
	executionHost = strings.TrimSpace(executionHost)
	capabilities = NormalizeWorkspaceRunnerHostCapabilities(capabilities)
	assessment := DeploymentProfileAssessment{
		ProfileName:     profileName,
		ExecutionHost:   executionHost,
		StrictPhoneOnly: strictPhoneOnly,
	}
	if strictPhoneOnly && executionHost != ExecutionHostPhone {
		assessment.Blockers = append(assessment.Blockers, "strict_phone_only requires execution_host phone")
	}
	switch profileName {
	case DeploymentProfilePhoneResident:
		if executionHost != ExecutionHostPhone {
			assessment.Blockers = append(assessment.Blockers, "phone_resident profile requires execution_host phone")
		}
		workspace := AssessWorkspaceRunnerProfile(WorkspaceRunnerProfilePhone, capabilities)
		assessment.Blockers = append(assessment.Blockers, workspace.Blockers...)
	case DeploymentProfileDesktopDev:
		if strictPhoneOnly {
			assessment.Blockers = append(assessment.Blockers, "desktop_dev profile is not allowed in strict_phone_only mode")
		}
		if executionHost != ExecutionHostDesktopDev {
			assessment.Blockers = append(assessment.Blockers, "desktop_dev profile requires execution_host desktop_dev")
		}
		workspace := AssessWorkspaceRunnerProfile(WorkspaceRunnerProfileDesktopDev, capabilities)
		assessment.Blockers = append(assessment.Blockers, workspace.Blockers...)
	default:
		assessment.Blockers = append(assessment.Blockers, fmt.Sprintf("deployment profile %q is unsupported", profileName))
	}
	assessment.Blockers = normalizeAutonomyStrings(assessment.Blockers)
	assessment.Ready = len(assessment.Blockers) == 0
	return assessment
}

func ValidateDeploymentProfileRecord(record DeploymentProfileRecord) error {
	if record.RecordVersion <= 0 {
		return fmt.Errorf("mission store deployment profile record_version must be positive")
	}
	if err := validateAutonomyIdentifierField("mission store deployment profile", "deployment_id", record.DeploymentID); err != nil {
		return err
	}
	if record.ProfileName == "" {
		return fmt.Errorf("mission store deployment profile profile_name is required")
	}
	if !isKnownExecutionHost(record.ExecutionHost) {
		return fmt.Errorf("mission store deployment profile execution_host %q is invalid", record.ExecutionHost)
	}
	if record.DeploymentID != DeploymentProfileID(record.ProfileName, record.ExecutionHost, record.StrictPhoneOnly) {
		return fmt.Errorf("mission store deployment profile deployment_id %q does not match deterministic deployment_id %q", record.DeploymentID, DeploymentProfileID(record.ProfileName, record.ExecutionHost, record.StrictPhoneOnly))
	}
	assessment := AssessDeploymentProfile(record.ProfileName, record.ExecutionHost, record.StrictPhoneOnly, record.Capabilities)
	if !reflect.DeepEqual(record.Assessment, assessment) {
		return fmt.Errorf("mission store deployment profile assessment does not match deterministic assessment")
	}
	if !record.Assessment.Ready {
		return fmt.Errorf("mission store deployment profile assessment is not ready: %s", strings.Join(record.Assessment.Blockers, "; "))
	}
	if record.CreatedAt.IsZero() {
		return fmt.Errorf("mission store deployment profile created_at is required")
	}
	if record.CreatedBy == "" {
		return fmt.Errorf("mission store deployment profile created_by is required")
	}
	return nil
}

func StoreDeploymentProfileRecord(root string, record DeploymentProfileRecord) (DeploymentProfileRecord, bool, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return DeploymentProfileRecord{}, false, err
	}
	record = NormalizeDeploymentProfileRecord(record)
	record.RecordVersion = normalizeRecordVersion(record.RecordVersion)
	if err := ValidateDeploymentProfileRecord(record); err != nil {
		return DeploymentProfileRecord{}, false, err
	}
	path := StoreDeploymentProfilePath(root, record.DeploymentID)
	if existing, err := loadDeploymentProfileRecordFile(path); err == nil {
		if reflect.DeepEqual(existing, record) {
			return existing, false, nil
		}
		return DeploymentProfileRecord{}, false, fmt.Errorf("mission store deployment profile %q already exists", record.DeploymentID)
	} else if !errors.Is(err, os.ErrNotExist) {
		return DeploymentProfileRecord{}, false, err
	}
	if err := WriteStoreJSONAtomic(path, record); err != nil {
		return DeploymentProfileRecord{}, false, err
	}
	return record, true, nil
}

func LoadDeploymentProfileRecord(root, deploymentID string) (DeploymentProfileRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return DeploymentProfileRecord{}, err
	}
	deploymentID = strings.TrimSpace(deploymentID)
	if err := validateAutonomyIdentifierField("deployment profile ref", "deployment_id", deploymentID); err != nil {
		return DeploymentProfileRecord{}, err
	}
	record, err := loadDeploymentProfileRecordFile(StoreDeploymentProfilePath(root, deploymentID))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return DeploymentProfileRecord{}, ErrDeploymentProfileRecordNotFound
		}
		return DeploymentProfileRecord{}, err
	}
	return record, nil
}

func ListDeploymentProfileRecords(root string) ([]DeploymentProfileRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return nil, err
	}
	return listStoreJSONRecords(StoreDeploymentProfilesDir(root), loadDeploymentProfileRecordFile)
}

func loadDeploymentProfileRecordFile(path string) (DeploymentProfileRecord, error) {
	var record DeploymentProfileRecord
	if err := LoadStoreJSON(path, &record); err != nil {
		return DeploymentProfileRecord{}, err
	}
	record = NormalizeDeploymentProfileRecord(record)
	if err := ValidateDeploymentProfileRecord(record); err != nil {
		return DeploymentProfileRecord{}, err
	}
	return record, nil
}
