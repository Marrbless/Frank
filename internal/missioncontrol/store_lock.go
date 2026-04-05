package missioncontrol

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"
)

var (
	storeWriterGuardRetryDelay = 2 * time.Millisecond
	storeWriterGuardMaxWait    = time.Second
	storeWriterGuardStaleAfter = 5 * time.Second
	storeWriterGuardSleep      = time.Sleep
	storeWriterGuardNow        = time.Now
)

type storeWriterGuard struct {
	root string
	path string
}

type writerGuardRecord struct {
	RecordVersion int       `json:"record_version"`
	LeaseHolderID string    `json:"lease_holder_id,omitempty"`
	PID           int       `json:"pid,omitempty"`
	Hostname      string    `json:"hostname,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

func LoadWriterLock(root string) (WriterLockRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return WriterLockRecord{}, err
	}

	var record WriterLockRecord
	if err := LoadStoreJSON(StoreWriterLockPath(root), &record); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return WriterLockRecord{}, ErrWriterLockNotFound
		}
		return WriterLockRecord{}, err
	}
	if err := ValidateWriterLockRecord(record); err != nil {
		return WriterLockRecord{}, err
	}
	return record, nil
}

func ValidateWriterLockRecord(record WriterLockRecord) error {
	if record.RecordVersion <= 0 {
		return fmt.Errorf("mission store writer lock record_version must be positive")
	}
	if record.WriterEpoch == 0 {
		return fmt.Errorf("mission store writer lock writer_epoch must be positive")
	}
	if record.LeaseHolderID == "" {
		return fmt.Errorf("mission store writer lock lease_holder_id is required")
	}
	if record.StartedAt.IsZero() {
		return fmt.Errorf("mission store writer lock started_at is required")
	}
	if record.RenewedAt.IsZero() {
		return fmt.Errorf("mission store writer lock renewed_at is required")
	}
	if record.LeaseExpiresAt.IsZero() {
		return fmt.Errorf("mission store writer lock lease_expires_at is required")
	}
	if record.LeaseExpiresAt.Before(record.RenewedAt) {
		return fmt.Errorf("mission store writer lock lease_expires_at must be >= renewed_at")
	}
	return nil
}

func IsWriterLockStale(record WriterLockRecord, now time.Time) bool {
	if err := ValidateWriterLockRecord(record); err != nil {
		return true
	}
	return !record.LeaseExpiresAt.After(now)
}

func AcquireWriterLock(root string, now time.Time, leaseDuration time.Duration, lease WriterLockLease) (WriterLockRecord, bool, error) {
	return acquireWriterLock(root, now, leaseDuration, lease, false)
}

func TakeoverWriterLock(root string, now time.Time, leaseDuration time.Duration, lease WriterLockLease) (WriterLockRecord, error) {
	record, _, err := acquireWriterLock(root, now, leaseDuration, lease, true)
	return record, err
}

func acquireWriterLock(root string, now time.Time, leaseDuration time.Duration, lease WriterLockLease, allowTakeover bool) (WriterLockRecord, bool, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return WriterLockRecord{}, false, err
	}
	if leaseDuration <= 0 {
		return WriterLockRecord{}, false, fmt.Errorf("mission store writer lock lease duration must be positive")
	}
	if lease.LeaseHolderID == "" {
		return WriterLockRecord{}, false, fmt.Errorf("mission store writer lock lease_holder_id is required")
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return WriterLockRecord{}, false, err
	}

	guard, err := acquireStoreWriterGuard(root)
	if err != nil {
		return WriterLockRecord{}, false, err
	}
	defer func() { _ = guard.release() }()

	manifest, err := InitStoreManifest(root, now)
	if err != nil {
		return WriterLockRecord{}, false, err
	}

	current, err := LoadWriterLock(root)
	switch {
	case err == nil:
		if current.WriterEpoch != manifest.CurrentWriterEpoch {
			if !allowTakeover || !IsWriterLockStale(current, now.UTC()) {
				return WriterLockRecord{}, false, newWriterEpochIncoherentError(manifest.CurrentWriterEpoch, current.WriterEpoch)
			}

			nextEpoch := maxWriterEpoch(current.WriterEpoch, manifest.CurrentWriterEpoch) + 1
			takenOver := newWriterLockRecord(nextEpoch, now, leaseDuration, lease)
			if err := StoreWriterLockRecord(root, takenOver); err != nil {
				return WriterLockRecord{}, false, err
			}
			manifest.CurrentWriterEpoch = nextEpoch
			manifest.LastRecoveredAt = now.UTC()
			if err := StoreManifestRecord(root, manifest); err != nil {
				return WriterLockRecord{}, false, err
			}
			if err := ValidateWriterEpochCoherence(root); err != nil {
				return WriterLockRecord{}, false, err
			}
			return takenOver, true, nil
		}
		if current.LeaseHolderID == lease.LeaseHolderID {
			if IsWriterLockStale(current, now.UTC()) {
				if !allowTakeover {
					return WriterLockRecord{}, false, ErrWriterLockExpired
				}
			} else {
				renewed := current
				renewed.RenewedAt = now.UTC()
				renewed.LeaseExpiresAt = now.UTC().Add(leaseDuration)
				renewed.JobID = lease.JobID
				if err := StoreWriterLockRecord(root, renewed); err != nil {
					return WriterLockRecord{}, false, err
				}
				if err := ValidateWriterEpochCoherence(root); err != nil {
					return WriterLockRecord{}, false, err
				}
				return renewed, false, nil
			}
		}
		if !IsWriterLockStale(current, now.UTC()) {
			return WriterLockRecord{}, false, ErrWriterLockHeld
		}
		if !allowTakeover {
			return WriterLockRecord{}, false, ErrWriterLockHeld
		}

		nextEpoch := manifest.CurrentWriterEpoch + 1
		takenOver := newWriterLockRecord(nextEpoch, now, leaseDuration, lease)
		if err := StoreWriterLockRecord(root, takenOver); err != nil {
			return WriterLockRecord{}, false, err
		}
		manifest.CurrentWriterEpoch = nextEpoch
		manifest.LastRecoveredAt = now.UTC()
		if err := StoreManifestRecord(root, manifest); err != nil {
			return WriterLockRecord{}, false, err
		}
		if err := ValidateWriterEpochCoherence(root); err != nil {
			return WriterLockRecord{}, false, err
		}
		return takenOver, true, nil
	case errors.Is(err, ErrWriterLockNotFound):
		nextEpoch := manifest.CurrentWriterEpoch + 1
		record := newWriterLockRecord(nextEpoch, now, leaseDuration, lease)
		if err := StoreWriterLockRecord(root, record); err != nil {
			return WriterLockRecord{}, false, err
		}
		manifest.CurrentWriterEpoch = nextEpoch
		if err := StoreManifestRecord(root, manifest); err != nil {
			return WriterLockRecord{}, false, err
		}
		if err := ValidateWriterEpochCoherence(root); err != nil {
			return WriterLockRecord{}, false, err
		}
		return record, false, nil
	default:
		if !allowTakeover {
			return WriterLockRecord{}, false, err
		}

		nextEpoch := manifest.CurrentWriterEpoch + 1
		record := newWriterLockRecord(nextEpoch, now, leaseDuration, lease)
		if err := StoreWriterLockRecord(root, record); err != nil {
			return WriterLockRecord{}, false, err
		}
		manifest.CurrentWriterEpoch = nextEpoch
		manifest.LastRecoveredAt = now.UTC()
		if err := StoreManifestRecord(root, manifest); err != nil {
			return WriterLockRecord{}, false, err
		}
		if err := ValidateWriterEpochCoherence(root); err != nil {
			return WriterLockRecord{}, false, err
		}
		return record, true, nil
	}
}

func RenewWriterLock(root string, current WriterLockRecord, now time.Time, leaseDuration time.Duration) (WriterLockRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return WriterLockRecord{}, err
	}
	if leaseDuration <= 0 {
		return WriterLockRecord{}, fmt.Errorf("mission store writer lock lease duration must be positive")
	}

	guard, err := acquireStoreWriterGuard(root)
	if err != nil {
		return WriterLockRecord{}, err
	}
	defer func() { _ = guard.release() }()

	stored, err := LoadWriterLock(root)
	if err != nil {
		return WriterLockRecord{}, err
	}
	manifest, err := LoadStoreManifest(root)
	if err != nil {
		return WriterLockRecord{}, err
	}
	if stored.WriterEpoch != manifest.CurrentWriterEpoch {
		return WriterLockRecord{}, newWriterEpochIncoherentError(manifest.CurrentWriterEpoch, stored.WriterEpoch)
	}
	if stored.LeaseHolderID != current.LeaseHolderID || stored.WriterEpoch != current.WriterEpoch {
		return WriterLockRecord{}, ErrWriterLockHeld
	}
	if !stored.LeaseExpiresAt.After(now.UTC()) {
		return WriterLockRecord{}, ErrWriterLockExpired
	}

	stored.RenewedAt = now.UTC()
	stored.LeaseExpiresAt = now.UTC().Add(leaseDuration)
	if err := StoreWriterLockRecord(root, stored); err != nil {
		return WriterLockRecord{}, err
	}
	if err := ValidateWriterEpochCoherence(root); err != nil {
		return WriterLockRecord{}, err
	}
	return stored, nil
}

func StoreWriterLockRecord(root string, record WriterLockRecord) error {
	if err := ValidateStoreRoot(root); err != nil {
		return err
	}
	if err := ValidateWriterLockRecord(record); err != nil {
		return err
	}
	return WriteStoreJSONAtomic(StoreWriterLockPath(root), record)
}

func newWriterLockRecord(epoch uint64, now time.Time, leaseDuration time.Duration, lease WriterLockLease) WriterLockRecord {
	now = now.UTC()
	return WriterLockRecord{
		RecordVersion:  StoreRecordVersion,
		WriterEpoch:    epoch,
		LeaseHolderID:  lease.LeaseHolderID,
		PID:            lease.PID,
		Hostname:       lease.Hostname,
		StartedAt:      now,
		RenewedAt:      now,
		LeaseExpiresAt: now.Add(leaseDuration),
		JobID:          lease.JobID,
	}
}

func ValidateWriterEpochCoherence(root string) error {
	manifest, err := LoadStoreManifest(root)
	if err != nil {
		return err
	}
	lock, err := LoadWriterLock(root)
	if err != nil {
		return err
	}
	if manifest.CurrentWriterEpoch != lock.WriterEpoch {
		return newWriterEpochIncoherentError(manifest.CurrentWriterEpoch, lock.WriterEpoch)
	}
	return nil
}

func newWriterEpochIncoherentError(manifestEpoch uint64, lockEpoch uint64) error {
	return fmt.Errorf("%w: manifest=%d lock=%d", ErrWriterEpochIncoherent, manifestEpoch, lockEpoch)
}

func maxWriterEpoch(left uint64, right uint64) uint64 {
	if left > right {
		return left
	}
	return right
}

func acquireStoreWriterGuard(root string) (*storeWriterGuard, error) {
	deadline := storeWriterGuardNow().Add(storeWriterGuardMaxWait)
	path := StoreWriterGuardPath(root)
	for {
		file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err == nil {
			record := writerGuardRecord{
				RecordVersion: StoreRecordVersion,
				CreatedAt:     storeWriterGuardNow().UTC(),
			}
			data, marshalErr := json.Marshal(record)
			if marshalErr != nil {
				_ = file.Close()
				_ = os.Remove(path)
				_ = storeSyncDir(root)
				return nil, marshalErr
			}
			if _, writeErr := file.Write(append(data, '\n')); writeErr != nil {
				_ = file.Close()
				_ = os.Remove(path)
				_ = storeSyncDir(root)
				return nil, writeErr
			}
			if syncErr := storeSyncFile(file); syncErr != nil {
				_ = file.Close()
				_ = os.Remove(path)
				_ = storeSyncDir(root)
				return nil, syncErr
			}
			if closeErr := file.Close(); closeErr != nil {
				_ = os.Remove(path)
				_ = storeSyncDir(root)
				return nil, closeErr
			}
			if err := storeSyncDir(root); err != nil {
				_ = os.Remove(path)
				_ = storeSyncDir(root)
				return nil, err
			}
			return &storeWriterGuard{root: root, path: path}, nil
		}
		if !errors.Is(err, os.ErrExist) {
			return nil, err
		}
		recovered, recoverErr := recoverStaleStoreWriterGuard(root, storeWriterGuardNow())
		if recoverErr != nil {
			return nil, recoverErr
		}
		if recovered {
			continue
		}
		if !storeWriterGuardNow().Before(deadline) {
			return nil, ErrWriterLockHeld
		}
		storeWriterGuardSleep(storeWriterGuardRetryDelay)
	}
}

func (g *storeWriterGuard) release() error {
	if g == nil || g.path == "" {
		return nil
	}
	if err := os.Remove(g.path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return storeSyncDir(g.root)
}

func recoverStaleStoreWriterGuard(root string, now time.Time) (bool, error) {
	path := StoreWriterGuardPath(root)
	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	if !guardFileIsProvablyStale(info, now) {
		return false, nil
	}
	if err := os.Remove(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	if err := storeSyncDir(root); err != nil {
		return false, err
	}
	return true, nil
}

func guardFileIsProvablyStale(info os.FileInfo, now time.Time) bool {
	if info == nil {
		return false
	}
	return !info.ModTime().Add(storeWriterGuardStaleAfter).After(now)
}
