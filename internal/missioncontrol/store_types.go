package missioncontrol

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

const (
	StoreRecordVersion      = 1
	StoreSchemaVersion      = 1
	StoreRetentionVersionV1 = 1
)

const (
	storeManifestFilename = "manifest.json"
	storeWriterLockFile   = "writer.lock"
	storeWriterGuardFile  = "writer.lock.guard"
	storeActiveJobFile    = "active_job.json"
	storeLogsDirName      = "logs"
	storeCurrentLogFile   = "current.log"
	storeCurrentMetaFile  = "current.meta.json"
	storeLogPackagesDir   = "log_packages"
	storeGatewayLogFile   = "gateway.log"
)

type StoreState string

const (
	StoreStateInitializing         StoreState = "initializing"
	StoreStateReady                StoreState = "ready"
	StoreStateImportingSnapshot    StoreState = "importing_snapshot"
	StoreStateImportedFromSnapshot StoreState = "imported_from_snapshot"
)

type StoreSnapshotImportMetadata struct {
	Imported                bool      `json:"imported"`
	SourceStatusFile        string    `json:"source_status_file,omitempty"`
	SourceSnapshotUpdatedAt string    `json:"source_snapshot_updated_at,omitempty"`
	SourceJobID             string    `json:"source_job_id,omitempty"`
	ImportedAt              time.Time `json:"imported_at,omitempty"`
}

type StoreManifest struct {
	RecordVersion          int                         `json:"record_version"`
	SchemaVersion          int                         `json:"schema_version"`
	StoreID                string                      `json:"store_id"`
	InitializedAt          time.Time                   `json:"initialized_at"`
	StoreState             StoreState                  `json:"store_state"`
	SnapshotImport         StoreSnapshotImportMetadata `json:"snapshot_import"`
	CurrentWriterEpoch     uint64                      `json:"current_writer_epoch"`
	LastRecoveredAt        time.Time                   `json:"last_recovered_at,omitempty"`
	RetentionPolicyVersion int                         `json:"retention_policy_version"`
}

type WriterLockRecord struct {
	RecordVersion  int       `json:"record_version"`
	WriterEpoch    uint64    `json:"writer_epoch"`
	LeaseHolderID  string    `json:"lease_holder_id"`
	PID            int       `json:"pid,omitempty"`
	Hostname       string    `json:"hostname,omitempty"`
	StartedAt      time.Time `json:"started_at"`
	RenewedAt      time.Time `json:"renewed_at"`
	LeaseExpiresAt time.Time `json:"lease_expires_at"`
	JobID          string    `json:"job_id,omitempty"`
}

type WriterLockLease struct {
	LeaseHolderID string
	PID           int
	Hostname      string
	JobID         string
}

type ActiveJobRecord struct {
	RecordVersion  int       `json:"record_version"`
	WriterEpoch    uint64    `json:"writer_epoch"`
	JobID          string    `json:"job_id"`
	State          JobState  `json:"state"`
	ActiveStepID   string    `json:"active_step_id,omitempty"`
	AttemptID      string    `json:"attempt_id,omitempty"`
	LeaseHolderID  string    `json:"lease_holder_id"`
	LeaseExpiresAt time.Time `json:"lease_expires_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	ActivationSeq  uint64    `json:"activation_seq"`
}

type LogPackageReason string

const (
	LogPackageReasonManual LogPackageReason = "manual"
	LogPackageReasonDaily  LogPackageReason = "daily"
	LogPackageReasonReboot LogPackageReason = "reboot"
)

type LogSegmentMeta struct {
	RecordVersion int       `json:"record_version"`
	OpenedAt      time.Time `json:"opened_at"`
}

type LogPackageManifest struct {
	RecordVersion   int              `json:"record_version"`
	PackageID       string           `json:"package_id"`
	Reason          LogPackageReason `json:"reason"`
	CreatedAt       time.Time        `json:"created_at"`
	SegmentOpenedAt time.Time        `json:"segment_opened_at"`
	SegmentClosedAt time.Time        `json:"segment_closed_at"`
	LogRelPath      string           `json:"log_relpath"`
	ByteCount       int64            `json:"byte_count"`
}

var (
	ErrActiveJobRecordNotFound    = errors.New("mission store active job record not found")
	ErrLogPackageManifestNotFound = errors.New("mission store log package manifest not found")
	ErrLogSegmentMetaNotFound     = errors.New("mission store log segment meta not found")
	ErrStoreManifestNotFound      = errors.New("mission store manifest not found")
	ErrWriterEpochIncoherent      = errors.New("mission store writer epoch is incoherent")
	ErrWriterLockExpired          = errors.New("mission store writer lock lease has expired")
	ErrWriterLockHeld             = errors.New("mission store writer lock is already held")
	ErrWriterLockNotFound         = errors.New("mission store writer lock not found")
)

func ResolveStoreRoot(explicitRoot string, statusFile string) string {
	if strings.TrimSpace(explicitRoot) != "" {
		return explicitRoot
	}
	if strings.TrimSpace(statusFile) == "" {
		return ""
	}
	return statusFile + ".store"
}

func StoreManifestPath(root string) string {
	return filepath.Join(root, storeManifestFilename)
}

func StoreWriterLockPath(root string) string {
	return filepath.Join(root, storeWriterLockFile)
}

func StoreWriterGuardPath(root string) string {
	return filepath.Join(root, storeWriterGuardFile)
}

func StoreActiveJobPath(root string) string {
	return filepath.Join(root, storeActiveJobFile)
}

func StoreLogsDir(root string) string {
	return filepath.Join(root, storeLogsDirName)
}

func StoreCurrentLogPath(root string) string {
	return filepath.Join(StoreLogsDir(root), storeCurrentLogFile)
}

func StoreCurrentLogMetaPath(root string) string {
	return filepath.Join(StoreLogsDir(root), storeCurrentMetaFile)
}

func StoreLogPackagesDir(root string) string {
	return filepath.Join(root, storeLogPackagesDir)
}

func StoreLogPackageDir(root string, packageID string) string {
	return filepath.Join(StoreLogPackagesDir(root), packageID)
}

func StoreLogPackageManifestPath(root string, packageID string) string {
	return filepath.Join(StoreLogPackageDir(root, packageID), storeManifestFilename)
}

func StoreLogPackageGatewayLogPath(root string, packageID string) string {
	return filepath.Join(StoreLogPackageDir(root, packageID), storeGatewayLogFile)
}

func StoreJobsDir(root string) string {
	return filepath.Join(root, "jobs")
}

func StoreJobDir(root string, jobID string) string {
	return filepath.Join(StoreJobsDir(root), jobID)
}

func StoreJobRuntimePath(root string, jobID string) string {
	return filepath.Join(StoreJobDir(root, jobID), "job_runtime.json")
}

func StoreRuntimeControlPath(root string, jobID string) string {
	return filepath.Join(StoreJobDir(root, jobID), "runtime_control.json")
}

func StoreStepRuntimePath(root string, jobID string, stepID string) string {
	return filepath.Join(StoreJobDir(root, jobID), "steps", stepID+".json")
}

func StoreApprovalRequestsDir(root string, jobID string) string {
	return filepath.Join(StoreJobDir(root, jobID), "approvals", "requests")
}

func StoreApprovalRequestPath(root string, jobID string, requestID string) string {
	return filepath.Join(StoreApprovalRequestsDir(root, jobID), requestID+".json")
}

func StoreApprovalGrantsDir(root string, jobID string) string {
	return filepath.Join(StoreJobDir(root, jobID), "approvals", "grants")
}

func StoreApprovalGrantPath(root string, jobID string, grantID string) string {
	return filepath.Join(StoreApprovalGrantsDir(root, jobID), grantID+".json")
}

func StoreAuditDir(root string, jobID string) string {
	return filepath.Join(StoreJobDir(root, jobID), "audit")
}

func StoreAuditEventPath(root string, jobID string, eventID string) string {
	return filepath.Join(StoreAuditDir(root, jobID), eventID+".json")
}

func StoreArtifactsDir(root string, jobID string) string {
	return filepath.Join(StoreJobDir(root, jobID), "artifacts")
}

func StoreArtifactPath(root string, jobID string, artifactID string) string {
	return filepath.Join(StoreArtifactsDir(root, jobID), artifactID+".json")
}

func ValidateStoreRoot(root string) error {
	if strings.TrimSpace(root) == "" {
		return fmt.Errorf("mission store root is required")
	}
	return nil
}

func ValidateStoreState(state StoreState) error {
	switch state {
	case StoreStateInitializing, StoreStateReady, StoreStateImportingSnapshot, StoreStateImportedFromSnapshot:
		return nil
	default:
		return fmt.Errorf("invalid mission store state %q", state)
	}
}

func ValidateActiveJobState(state JobState) error {
	if HoldsGlobalActiveJobOccupancy(state) {
		return nil
	}
	return fmt.Errorf("active job state %q does not hold global occupancy", state)
}

func HoldsGlobalActiveJobOccupancy(state JobState) bool {
	switch state {
	case JobStateRunning, JobStateWaitingUser, JobStatePaused:
		return true
	default:
		return false
	}
}
