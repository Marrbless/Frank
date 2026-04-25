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

type HotUpdateDeployLockState string

const (
	HotUpdateDeployLockStateUnknown        HotUpdateDeployLockState = "unknown"
	HotUpdateDeployLockStateDeployLocked   HotUpdateDeployLockState = "deploy_locked"
	HotUpdateDeployLockStateDeployUnlocked HotUpdateDeployLockState = "deploy_unlocked"
)

type HotUpdateExecutionSafetyEvidenceRef struct {
	EvidenceID string `json:"evidence_id"`
}

type HotUpdateExecutionSafetyEvidenceRecord struct {
	RecordVersion   int                      `json:"record_version"`
	EvidenceID      string                   `json:"evidence_id"`
	HotUpdateID     string                   `json:"hot_update_id"`
	JobID           string                   `json:"job_id"`
	ActiveStepID    string                   `json:"active_step_id,omitempty"`
	AttemptID       string                   `json:"attempt_id,omitempty"`
	WriterEpoch     uint64                   `json:"writer_epoch,omitempty"`
	ActivationSeq   uint64                   `json:"activation_seq,omitempty"`
	DeployLockState HotUpdateDeployLockState `json:"deploy_lock_state"`
	QuiesceState    HotUpdateQuiesceState    `json:"quiesce_state"`
	Reason          string                   `json:"reason,omitempty"`
	CreatedAt       time.Time                `json:"created_at"`
	CreatedBy       string                   `json:"created_by"`
	ExpiresAt       time.Time                `json:"expires_at,omitempty"`
}

var ErrHotUpdateExecutionSafetyEvidenceRecordNotFound = errors.New("mission store hot-update execution safety evidence record not found")

func StoreHotUpdateExecutionSafetyEvidenceDir(root string) string {
	return filepath.Join(root, "runtime_packs", "hot_update_execution_safety")
}

func StoreHotUpdateExecutionSafetyEvidencePath(root, evidenceID string) string {
	return filepath.Join(StoreHotUpdateExecutionSafetyEvidenceDir(root), strings.TrimSpace(evidenceID)+".json")
}

func HotUpdateExecutionSafetyEvidenceID(hotUpdateID, jobID string) (string, error) {
	hotUpdateID = strings.TrimSpace(hotUpdateID)
	jobID = strings.TrimSpace(jobID)
	if err := ValidateHotUpdateGateRef(HotUpdateGateRef{HotUpdateID: hotUpdateID}); err != nil {
		return "", err
	}
	if err := validateHotUpdateExecutionSafetyJobID(jobID); err != nil {
		return "", err
	}
	return "hot-update-execution-safety-" + hotUpdateID + "-" + jobID, nil
}

func NormalizeHotUpdateExecutionSafetyEvidenceRef(ref HotUpdateExecutionSafetyEvidenceRef) HotUpdateExecutionSafetyEvidenceRef {
	ref.EvidenceID = strings.TrimSpace(ref.EvidenceID)
	return ref
}

func NormalizeHotUpdateExecutionSafetyEvidenceRecord(record HotUpdateExecutionSafetyEvidenceRecord) HotUpdateExecutionSafetyEvidenceRecord {
	record.EvidenceID = strings.TrimSpace(record.EvidenceID)
	record.HotUpdateID = strings.TrimSpace(record.HotUpdateID)
	record.JobID = strings.TrimSpace(record.JobID)
	record.ActiveStepID = strings.TrimSpace(record.ActiveStepID)
	record.AttemptID = strings.TrimSpace(record.AttemptID)
	record.DeployLockState = HotUpdateDeployLockState(strings.TrimSpace(string(record.DeployLockState)))
	record.QuiesceState = HotUpdateQuiesceState(strings.TrimSpace(string(record.QuiesceState)))
	record.Reason = strings.TrimSpace(record.Reason)
	record.CreatedAt = record.CreatedAt.UTC()
	record.CreatedBy = strings.TrimSpace(record.CreatedBy)
	record.ExpiresAt = record.ExpiresAt.UTC()
	return record
}

func ValidateHotUpdateExecutionSafetyEvidenceRef(ref HotUpdateExecutionSafetyEvidenceRef) error {
	return validateHotUpdateIdentifierField("hot-update execution safety evidence ref", "evidence_id", ref.EvidenceID)
}

func ValidateHotUpdateExecutionSafetyEvidenceRecord(record HotUpdateExecutionSafetyEvidenceRecord) error {
	if record.RecordVersion <= 0 {
		return fmt.Errorf("mission store hot-update execution safety evidence record_version must be positive")
	}
	if err := ValidateHotUpdateExecutionSafetyEvidenceRef(HotUpdateExecutionSafetyEvidenceRef{EvidenceID: record.EvidenceID}); err != nil {
		return err
	}
	if err := ValidateHotUpdateGateRef(HotUpdateGateRef{HotUpdateID: record.HotUpdateID}); err != nil {
		return fmt.Errorf("mission store hot-update execution safety evidence hot_update_id %q: %w", record.HotUpdateID, err)
	}
	if err := validateHotUpdateExecutionSafetyJobID(record.JobID); err != nil {
		return err
	}
	wantEvidenceID, err := HotUpdateExecutionSafetyEvidenceID(record.HotUpdateID, record.JobID)
	if err != nil {
		return err
	}
	if record.EvidenceID != wantEvidenceID {
		return fmt.Errorf("mission store hot-update execution safety evidence evidence_id %q does not match deterministic evidence_id %q", record.EvidenceID, wantEvidenceID)
	}
	if !isValidHotUpdateDeployLockState(record.DeployLockState) {
		return fmt.Errorf("mission store hot-update execution safety evidence deploy_lock_state %q is invalid", record.DeployLockState)
	}
	if !isValidHotUpdateExecutionSafetyQuiesceState(record.QuiesceState) {
		return fmt.Errorf("mission store hot-update execution safety evidence quiesce_state %q is invalid", record.QuiesceState)
	}
	if record.CreatedAt.IsZero() {
		return fmt.Errorf("mission store hot-update execution safety evidence created_at is required")
	}
	if record.CreatedBy == "" {
		return fmt.Errorf("mission store hot-update execution safety evidence created_by is required")
	}
	if record.DeployLockState == HotUpdateDeployLockStateDeployUnlocked && record.QuiesceState == HotUpdateQuiesceStateReady && record.ExpiresAt.IsZero() {
		return fmt.Errorf("mission store hot-update execution safety evidence expires_at is required for deploy_unlocked ready evidence")
	}
	return nil
}

func StoreHotUpdateExecutionSafetyEvidenceRecord(root string, record HotUpdateExecutionSafetyEvidenceRecord) (HotUpdateExecutionSafetyEvidenceRecord, bool, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return HotUpdateExecutionSafetyEvidenceRecord{}, false, err
	}
	record = NormalizeHotUpdateExecutionSafetyEvidenceRecord(record)
	record.RecordVersion = normalizeRecordVersion(record.RecordVersion)
	if err := ValidateHotUpdateExecutionSafetyEvidenceRecord(record); err != nil {
		return HotUpdateExecutionSafetyEvidenceRecord{}, false, err
	}

	path := StoreHotUpdateExecutionSafetyEvidencePath(root, record.EvidenceID)
	existing, err := loadHotUpdateExecutionSafetyEvidenceRecordFile(path)
	if err == nil {
		if reflect.DeepEqual(existing, record) {
			return existing, false, nil
		}
		return HotUpdateExecutionSafetyEvidenceRecord{}, false, fmt.Errorf("mission store hot-update execution safety evidence %q already exists", record.EvidenceID)
	}
	if !errors.Is(err, os.ErrNotExist) {
		return HotUpdateExecutionSafetyEvidenceRecord{}, false, err
	}
	if err := WriteStoreJSONAtomic(path, record); err != nil {
		return HotUpdateExecutionSafetyEvidenceRecord{}, false, err
	}
	stored, err := LoadHotUpdateExecutionSafetyEvidenceRecord(root, record.EvidenceID)
	if err != nil {
		return HotUpdateExecutionSafetyEvidenceRecord{}, false, err
	}
	return stored, true, nil
}

func EnsureHotUpdateExecutionReadyEvidence(root string, hotUpdateID string, activeJob ActiveJobRecord, createdBy string, createdAt time.Time, expiresAt time.Time, reason string) (HotUpdateExecutionSafetyEvidenceRecord, bool, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return HotUpdateExecutionSafetyEvidenceRecord{}, false, err
	}
	if err := ValidateActiveJobRecord(activeJob); err != nil {
		return HotUpdateExecutionSafetyEvidenceRecord{}, false, err
	}
	hotUpdateID = strings.TrimSpace(hotUpdateID)
	if err := ValidateHotUpdateGateRef(HotUpdateGateRef{HotUpdateID: hotUpdateID}); err != nil {
		return HotUpdateExecutionSafetyEvidenceRecord{}, false, err
	}
	createdBy = strings.TrimSpace(createdBy)
	if createdBy == "" {
		return HotUpdateExecutionSafetyEvidenceRecord{}, false, fmt.Errorf("mission store hot-update execution safety evidence created_by is required")
	}
	createdAt = createdAt.UTC()
	if createdAt.IsZero() {
		return HotUpdateExecutionSafetyEvidenceRecord{}, false, fmt.Errorf("mission store hot-update execution safety evidence created_at is required")
	}
	expiresAt = expiresAt.UTC()
	if expiresAt.IsZero() {
		return HotUpdateExecutionSafetyEvidenceRecord{}, false, fmt.Errorf("mission store hot-update execution safety evidence expires_at is required")
	}
	if !expiresAt.After(createdAt) {
		return HotUpdateExecutionSafetyEvidenceRecord{}, false, fmt.Errorf("mission store hot-update execution safety evidence expires_at must be after created_at")
	}

	evidenceID, err := HotUpdateExecutionSafetyEvidenceID(hotUpdateID, activeJob.JobID)
	if err != nil {
		return HotUpdateExecutionSafetyEvidenceRecord{}, false, err
	}
	record := HotUpdateExecutionSafetyEvidenceRecord{
		RecordVersion:   StoreRecordVersion,
		EvidenceID:      evidenceID,
		HotUpdateID:     hotUpdateID,
		JobID:           strings.TrimSpace(activeJob.JobID),
		ActiveStepID:    strings.TrimSpace(activeJob.ActiveStepID),
		AttemptID:       strings.TrimSpace(activeJob.AttemptID),
		WriterEpoch:     activeJob.WriterEpoch,
		ActivationSeq:   activeJob.ActivationSeq,
		DeployLockState: HotUpdateDeployLockStateDeployUnlocked,
		QuiesceState:    HotUpdateQuiesceStateReady,
		Reason:          strings.TrimSpace(reason),
		CreatedAt:       createdAt,
		CreatedBy:       createdBy,
		ExpiresAt:       expiresAt,
	}
	record = NormalizeHotUpdateExecutionSafetyEvidenceRecord(record)
	if err := ValidateHotUpdateExecutionSafetyEvidenceRecord(record); err != nil {
		return HotUpdateExecutionSafetyEvidenceRecord{}, false, err
	}

	path := StoreHotUpdateExecutionSafetyEvidencePath(root, record.EvidenceID)
	existing, err := loadHotUpdateExecutionSafetyEvidenceRecordFile(path)
	if err == nil {
		if reflect.DeepEqual(existing, record) {
			return existing, false, nil
		}
		if hotUpdateExecutionSafetyEvidenceStale(existing, activeJob) {
			return HotUpdateExecutionSafetyEvidenceRecord{}, false, fmt.Errorf("mission store hot-update execution safety evidence %q is stale for current active job", record.EvidenceID)
		}
		if !hotUpdateExecutionSafetyEvidenceExpired(existing, createdAt) {
			return HotUpdateExecutionSafetyEvidenceRecord{}, false, fmt.Errorf("mission store hot-update execution safety evidence %q divergent duplicate already exists", record.EvidenceID)
		}
		if existing.HotUpdateID != record.HotUpdateID || existing.JobID != record.JobID {
			return HotUpdateExecutionSafetyEvidenceRecord{}, false, fmt.Errorf("mission store hot-update execution safety evidence %q does not match requested hot update and job", record.EvidenceID)
		}
		if err := WriteStoreJSONAtomic(path, record); err != nil {
			return HotUpdateExecutionSafetyEvidenceRecord{}, false, err
		}
		stored, err := LoadHotUpdateExecutionSafetyEvidenceRecord(root, record.EvidenceID)
		if err != nil {
			return HotUpdateExecutionSafetyEvidenceRecord{}, false, err
		}
		return stored, true, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return HotUpdateExecutionSafetyEvidenceRecord{}, false, err
	}
	if err := WriteStoreJSONAtomic(path, record); err != nil {
		return HotUpdateExecutionSafetyEvidenceRecord{}, false, err
	}
	stored, err := LoadHotUpdateExecutionSafetyEvidenceRecord(root, record.EvidenceID)
	if err != nil {
		return HotUpdateExecutionSafetyEvidenceRecord{}, false, err
	}
	return stored, true, nil
}

func LoadHotUpdateExecutionSafetyEvidenceRecord(root, evidenceID string) (HotUpdateExecutionSafetyEvidenceRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return HotUpdateExecutionSafetyEvidenceRecord{}, err
	}
	ref := NormalizeHotUpdateExecutionSafetyEvidenceRef(HotUpdateExecutionSafetyEvidenceRef{EvidenceID: evidenceID})
	if err := ValidateHotUpdateExecutionSafetyEvidenceRef(ref); err != nil {
		return HotUpdateExecutionSafetyEvidenceRecord{}, err
	}
	record, err := loadHotUpdateExecutionSafetyEvidenceRecordFile(StoreHotUpdateExecutionSafetyEvidencePath(root, ref.EvidenceID))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return HotUpdateExecutionSafetyEvidenceRecord{}, ErrHotUpdateExecutionSafetyEvidenceRecordNotFound
		}
		return HotUpdateExecutionSafetyEvidenceRecord{}, err
	}
	return record, nil
}

func LoadCurrentHotUpdateExecutionSafetyEvidenceRecord(root, hotUpdateID, jobID string) (HotUpdateExecutionSafetyEvidenceRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return HotUpdateExecutionSafetyEvidenceRecord{}, err
	}
	evidenceID, err := HotUpdateExecutionSafetyEvidenceID(hotUpdateID, jobID)
	if err != nil {
		return HotUpdateExecutionSafetyEvidenceRecord{}, err
	}
	return LoadHotUpdateExecutionSafetyEvidenceRecord(root, evidenceID)
}

func ListHotUpdateExecutionSafetyEvidenceRecords(root string) ([]HotUpdateExecutionSafetyEvidenceRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return nil, err
	}
	return listStoreJSONRecords(StoreHotUpdateExecutionSafetyEvidenceDir(root), loadHotUpdateExecutionSafetyEvidenceRecordFile)
}

func loadHotUpdateExecutionSafetyEvidenceRecordFile(path string) (HotUpdateExecutionSafetyEvidenceRecord, error) {
	var record HotUpdateExecutionSafetyEvidenceRecord
	if err := LoadStoreJSON(path, &record); err != nil {
		return HotUpdateExecutionSafetyEvidenceRecord{}, err
	}
	record = NormalizeHotUpdateExecutionSafetyEvidenceRecord(record)
	if err := ValidateHotUpdateExecutionSafetyEvidenceRecord(record); err != nil {
		return HotUpdateExecutionSafetyEvidenceRecord{}, err
	}
	return record, nil
}

func isValidHotUpdateDeployLockState(state HotUpdateDeployLockState) bool {
	switch state {
	case HotUpdateDeployLockStateUnknown, HotUpdateDeployLockStateDeployLocked, HotUpdateDeployLockStateDeployUnlocked:
		return true
	default:
		return false
	}
}

func isValidHotUpdateExecutionSafetyQuiesceState(state HotUpdateQuiesceState) bool {
	switch state {
	case HotUpdateQuiesceStateUnknown, HotUpdateQuiesceStateReady, HotUpdateQuiesceStateFailed:
		return true
	default:
		return false
	}
}

func validateHotUpdateExecutionSafetyJobID(jobID string) error {
	return validateHotUpdateIdentifierField("mission store hot-update execution safety evidence", "job_id", jobID)
}
