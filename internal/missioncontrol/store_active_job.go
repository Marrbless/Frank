package missioncontrol

import (
	"errors"
	"fmt"
	"os"
	"time"
)

func LoadActiveJobRecord(root string) (ActiveJobRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return ActiveJobRecord{}, err
	}

	var record ActiveJobRecord
	if err := LoadStoreJSON(StoreActiveJobPath(root), &record); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return ActiveJobRecord{}, ErrActiveJobRecordNotFound
		}
		return ActiveJobRecord{}, err
	}
	if err := ValidateActiveJobRecord(record); err != nil {
		return ActiveJobRecord{}, err
	}
	return record, nil
}

func StoreActiveJobRecord(root string, record ActiveJobRecord) error {
	if err := ValidateStoreRoot(root); err != nil {
		return err
	}
	if err := ValidateActiveJobRecord(record); err != nil {
		return err
	}
	return WriteStoreJSONAtomic(StoreActiveJobPath(root), record)
}

func RemoveActiveJobRecord(root string) error {
	if err := ValidateStoreRoot(root); err != nil {
		return err
	}
	path := StoreActiveJobPath(root)
	if err := os.Remove(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	return storeSyncDir(root)
}

func ValidateActiveJobRecord(record ActiveJobRecord) error {
	if record.RecordVersion <= 0 {
		return fmt.Errorf("mission store active job record_version must be positive")
	}
	if record.WriterEpoch == 0 {
		return fmt.Errorf("mission store active job writer_epoch must be positive")
	}
	if record.JobID == "" {
		return fmt.Errorf("mission store active job job_id is required")
	}
	if err := ValidateActiveJobState(record.State); err != nil {
		return err
	}
	if record.LeaseHolderID == "" {
		return fmt.Errorf("mission store active job lease_holder_id is required")
	}
	if record.LeaseExpiresAt.IsZero() {
		return fmt.Errorf("mission store active job lease_expires_at is required")
	}
	if record.UpdatedAt.IsZero() {
		return fmt.Errorf("mission store active job updated_at is required")
	}
	if record.ActivationSeq == 0 {
		return fmt.Errorf("mission store active job activation_seq must be positive")
	}
	return nil
}

func NewActiveJobRecord(writerEpoch uint64, jobID string, state JobState, activeStepID string, leaseHolderID string, leaseExpiresAt time.Time, updatedAt time.Time, activationSeq uint64) (ActiveJobRecord, error) {
	record := ActiveJobRecord{
		RecordVersion:  StoreRecordVersion,
		WriterEpoch:    writerEpoch,
		JobID:          jobID,
		State:          state,
		ActiveStepID:   activeStepID,
		LeaseHolderID:  leaseHolderID,
		LeaseExpiresAt: leaseExpiresAt.UTC(),
		UpdatedAt:      updatedAt.UTC(),
		ActivationSeq:  activationSeq,
	}
	if err := ValidateActiveJobRecord(record); err != nil {
		return ActiveJobRecord{}, err
	}
	return record, nil
}

func ReconcileActiveJobRecord(record ActiveJobRecord, runtimeJobID string, runtimeState JobState, runtimeActiveStepID string, runtimeAppliedSeq uint64) (*ActiveJobRecord, bool) {
	if record.JobID != runtimeJobID {
		return &record, false
	}
	if !HoldsGlobalActiveJobOccupancy(runtimeState) {
		return nil, true
	}
	if runtimeAppliedSeq < record.ActivationSeq {
		return nil, true
	}
	if runtimeActiveStepID == "" {
		return nil, true
	}
	if record.ActiveStepID != "" && record.ActiveStepID != runtimeActiveStepID {
		return nil, true
	}
	return &record, false
}
