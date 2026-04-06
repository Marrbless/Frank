package missioncontrol

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const storeLogPackageLeaseDuration = time.Minute

type CurrentLogSegmentState struct {
	Exists    bool
	Empty     bool
	ByteCount int64
}

type PackageCurrentLogResult struct {
	Reason             LogPackageReason
	NoOp               bool
	NoOpCause          string
	PackageID          string
	LogRelPath         string
	ByteCount          int64
	CurrentLogRelPath  string
	CurrentMetaRelPath string
}

func ValidateLogPackageReason(reason LogPackageReason) error {
	switch reason {
	case LogPackageReasonManual, LogPackageReasonDaily, LogPackageReasonReboot:
		return nil
	default:
		return fmt.Errorf("mission store log package reason %q is invalid", reason)
	}
}

func ValidateLogSegmentMeta(meta LogSegmentMeta) error {
	if meta.RecordVersion <= 0 {
		return fmt.Errorf("mission store log segment meta record_version must be positive")
	}
	if meta.OpenedAt.IsZero() {
		return fmt.Errorf("mission store log segment meta opened_at is required")
	}
	return nil
}

func ValidateLogPackageManifest(manifest LogPackageManifest) error {
	if manifest.RecordVersion <= 0 {
		return fmt.Errorf("mission store log package manifest record_version must be positive")
	}
	if manifest.PackageID == "" {
		return fmt.Errorf("mission store log package manifest package_id is required")
	}
	if err := ValidateLogPackageReason(manifest.Reason); err != nil {
		return err
	}
	if manifest.CreatedAt.IsZero() {
		return fmt.Errorf("mission store log package manifest created_at is required")
	}
	if manifest.SegmentOpenedAt.IsZero() {
		return fmt.Errorf("mission store log package manifest segment_opened_at is required")
	}
	if manifest.SegmentClosedAt.IsZero() {
		return fmt.Errorf("mission store log package manifest segment_closed_at is required")
	}
	if manifest.SegmentClosedAt.Before(manifest.SegmentOpenedAt) {
		return fmt.Errorf("mission store log package manifest segment_closed_at must be >= segment_opened_at")
	}
	if manifest.LogRelPath == "" {
		return fmt.Errorf("mission store log package manifest log_relpath is required")
	}
	if manifest.ByteCount <= 0 {
		return fmt.Errorf("mission store log package manifest byte_count must be positive")
	}
	return nil
}

func LoadCurrentLogSegmentMeta(root string) (LogSegmentMeta, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return LogSegmentMeta{}, err
	}

	var meta LogSegmentMeta
	if err := LoadStoreJSON(StoreCurrentLogMetaPath(root), &meta); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return LogSegmentMeta{}, ErrLogSegmentMetaNotFound
		}
		return LogSegmentMeta{}, err
	}
	if err := ValidateLogSegmentMeta(meta); err != nil {
		return LogSegmentMeta{}, err
	}
	return meta, nil
}

func StoreCurrentLogSegmentMeta(root string, meta LogSegmentMeta) error {
	if err := ValidateStoreRoot(root); err != nil {
		return err
	}
	if err := ValidateLogSegmentMeta(meta); err != nil {
		return err
	}
	return WriteStoreJSONAtomic(StoreCurrentLogMetaPath(root), meta)
}

func LoadLogPackageManifest(root string, packageID string) (LogPackageManifest, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return LogPackageManifest{}, err
	}
	if packageID == "" {
		return LogPackageManifest{}, fmt.Errorf("mission store log package manifest package_id is required")
	}

	var manifest LogPackageManifest
	if err := LoadStoreJSON(StoreLogPackageManifestPath(root, packageID), &manifest); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return LogPackageManifest{}, ErrLogPackageManifestNotFound
		}
		return LogPackageManifest{}, err
	}
	if err := ValidateLogPackageManifest(manifest); err != nil {
		return LogPackageManifest{}, err
	}
	return manifest, nil
}

func StoreLogPackageManifestRecord(root string, manifest LogPackageManifest) error {
	if err := ValidateStoreRoot(root); err != nil {
		return err
	}
	if err := ValidateLogPackageManifest(manifest); err != nil {
		return err
	}
	return WriteStoreJSONAtomic(StoreLogPackageManifestPath(root, manifest.PackageID), manifest)
}

func EnsureCurrentLogSegment(root string, now time.Time) (LogSegmentMeta, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return LogSegmentMeta{}, err
	}
	now = normalizeStoreLogTime(now)

	if err := ensureStoreChildDir(root, StoreLogsDir(root)); err != nil {
		return LogSegmentMeta{}, err
	}
	if _, err := ensureStoreFileExists(StoreCurrentLogPath(root)); err != nil {
		return LogSegmentMeta{}, err
	}

	meta, err := LoadCurrentLogSegmentMeta(root)
	switch {
	case err == nil:
		return meta, nil
	case errors.Is(err, ErrLogSegmentMetaNotFound):
		meta = LogSegmentMeta{
			RecordVersion: StoreRecordVersion,
			OpenedAt:      now,
		}
		if err := StoreCurrentLogSegmentMeta(root, meta); err != nil {
			return LogSegmentMeta{}, err
		}
		return meta, nil
	default:
		return LogSegmentMeta{}, err
	}
}

func InspectCurrentLogSegment(root string) (CurrentLogSegmentState, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return CurrentLogSegmentState{}, err
	}

	info, err := os.Stat(StoreCurrentLogPath(root))
	if errors.Is(err, os.ErrNotExist) {
		return CurrentLogSegmentState{}, nil
	}
	if err != nil {
		return CurrentLogSegmentState{}, err
	}
	if info.IsDir() {
		return CurrentLogSegmentState{}, fmt.Errorf("mission store current log path %q must be a file", StoreCurrentLogPath(root))
	}

	return CurrentLogSegmentState{
		Exists:    true,
		Empty:     info.Size() == 0,
		ByteCount: info.Size(),
	}, nil
}

func PackageCurrentLogSegment(root string, reason LogPackageReason, lease WriterLockLease, now time.Time) (PackageCurrentLogResult, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return PackageCurrentLogResult{}, err
	}
	if err := ValidateLogPackageReason(reason); err != nil {
		return PackageCurrentLogResult{}, err
	}
	now = normalizeStoreLogTime(now)

	lock, err := acquireLogPackagingWriterLock(root, lease, now)
	if err != nil {
		return PackageCurrentLogResult{}, err
	}

	guard, err := acquireStoreWriterGuard(root)
	if err != nil {
		return PackageCurrentLogResult{}, err
	}
	defer func() { _ = guard.release() }()

	if err := ValidateHeldWriterLock(root, lock, now); err != nil {
		return PackageCurrentLogResult{}, err
	}

	state, err := InspectCurrentLogSegment(root)
	if err != nil {
		return PackageCurrentLogResult{}, err
	}

	result := PackageCurrentLogResult{
		Reason:             reason,
		CurrentLogRelPath:  storeCurrentLogRelPath(),
		CurrentMetaRelPath: storeCurrentLogMetaRelPath(),
	}
	if !state.Exists {
		if _, err := EnsureCurrentLogSegment(root, now); err != nil {
			return PackageCurrentLogResult{}, err
		}
		result.NoOp = true
		result.NoOpCause = "absent"
		return result, nil
	}

	meta, err := EnsureCurrentLogSegment(root, now)
	if err != nil {
		return PackageCurrentLogResult{}, err
	}
	if state.Empty {
		result.NoOp = true
		result.NoOpCause = "empty"
		return result, nil
	}

	currentLogPath := StoreCurrentLogPath(root)
	currentFile, err := os.OpenFile(currentLogPath, os.O_RDWR, 0)
	if err != nil {
		return PackageCurrentLogResult{}, err
	}
	info, err := currentFile.Stat()
	if err != nil {
		_ = currentFile.Close()
		return PackageCurrentLogResult{}, err
	}
	if info.Size() == 0 {
		_ = currentFile.Close()
		result.NoOp = true
		result.NoOpCause = "empty"
		return result, nil
	}
	if err := storeSyncFile(currentFile); err != nil {
		_ = currentFile.Close()
		return PackageCurrentLogResult{}, err
	}
	if err := currentFile.Close(); err != nil {
		return PackageCurrentLogResult{}, err
	}

	packageID := newLogPackageID(now, reason)
	logPackagesDir := StoreLogPackagesDir(root)
	if err := ensureStoreChildDir(root, logPackagesDir); err != nil {
		return PackageCurrentLogResult{}, err
	}
	packageDir := StoreLogPackageDir(root, packageID)
	if err := ensureStoreChildDir(logPackagesDir, packageDir); err != nil {
		return PackageCurrentLogResult{}, err
	}

	gatewayLogPath := StoreLogPackageGatewayLogPath(root, packageID)
	if err := os.Rename(currentLogPath, gatewayLogPath); err != nil {
		return PackageCurrentLogResult{}, err
	}
	if err := storeSyncDir(StoreLogsDir(root)); err != nil {
		return PackageCurrentLogResult{}, err
	}
	if err := storeSyncDir(packageDir); err != nil {
		return PackageCurrentLogResult{}, err
	}

	manifest := LogPackageManifest{
		RecordVersion:   StoreRecordVersion,
		PackageID:       packageID,
		Reason:          reason,
		CreatedAt:       now,
		SegmentOpenedAt: meta.OpenedAt.UTC(),
		SegmentClosedAt: now,
		LogRelPath:      storeGatewayLogRelPath(packageID),
		ByteCount:       info.Size(),
	}
	if err := StoreLogPackageManifestRecord(root, manifest); err != nil {
		return PackageCurrentLogResult{}, err
	}
	if _, err := reopenCurrentLogSegment(root, now); err != nil {
		return PackageCurrentLogResult{}, err
	}

	result.PackageID = packageID
	result.LogRelPath = manifest.LogRelPath
	result.ByteCount = manifest.ByteCount
	return result, nil
}

func acquireLogPackagingWriterLock(root string, lease WriterLockLease, now time.Time) (WriterLockRecord, error) {
	lock, _, err := AcquireWriterLock(root, now, storeLogPackageLeaseDuration, lease)
	if err == nil {
		return lock, nil
	}
	if errors.Is(err, ErrWriterLockExpired) {
		return TakeoverWriterLock(root, now, storeLogPackageLeaseDuration, lease)
	}
	return WriterLockRecord{}, err
}

func reopenCurrentLogSegment(root string, now time.Time) (LogSegmentMeta, error) {
	if err := ensureStoreChildDir(root, StoreLogsDir(root)); err != nil {
		return LogSegmentMeta{}, err
	}
	if created, err := ensureStoreFileExists(StoreCurrentLogPath(root)); err != nil {
		return LogSegmentMeta{}, err
	} else if !created {
		return LogSegmentMeta{}, fmt.Errorf("mission store current log path %q already exists during reopen", StoreCurrentLogPath(root))
	}

	meta := LogSegmentMeta{
		RecordVersion: StoreRecordVersion,
		OpenedAt:      normalizeStoreLogTime(now),
	}
	if err := StoreCurrentLogSegmentMeta(root, meta); err != nil {
		return LogSegmentMeta{}, err
	}
	return meta, nil
}

func normalizeStoreLogTime(now time.Time) time.Time {
	if now.IsZero() {
		return time.Now().UTC()
	}
	return now.UTC()
}

func newLogPackageID(now time.Time, reason LogPackageReason) string {
	return fmt.Sprintf("%s-%s", normalizeStoreLogTime(now).Format("20060102T150405.000000000Z"), reason)
}

func ensureStoreChildDir(parent string, path string) error {
	info, err := os.Stat(path)
	switch {
	case err == nil:
		if !info.IsDir() {
			return fmt.Errorf("mission store path %q must be a directory", path)
		}
		return nil
	case !errors.Is(err, os.ErrNotExist):
		return err
	}

	if err := os.Mkdir(path, 0o755); err != nil {
		if errors.Is(err, os.ErrExist) {
			return nil
		}
		return err
	}
	return storeSyncDir(parent)
}

func ensureStoreFileExists(path string) (bool, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0o644)
	if err == nil {
		if err := storeSyncFile(file); err != nil {
			_ = file.Close()
			_ = os.Remove(path)
			_ = storeSyncDir(filepath.Dir(path))
			return false, err
		}
		if err := file.Close(); err != nil {
			_ = os.Remove(path)
			_ = storeSyncDir(filepath.Dir(path))
			return false, err
		}
		if err := storeSyncDir(filepath.Dir(path)); err != nil {
			return false, err
		}
		return true, nil
	}
	if !errors.Is(err, os.ErrExist) {
		return false, err
	}

	info, statErr := os.Stat(path)
	if statErr != nil {
		return false, statErr
	}
	if info.IsDir() {
		return false, fmt.Errorf("mission store path %q must be a file", path)
	}
	return false, nil
}

func storeCurrentLogRelPath() string {
	return filepath.ToSlash(filepath.Join(storeLogsDirName, storeCurrentLogFile))
}

func storeCurrentLogMetaRelPath() string {
	return filepath.ToSlash(filepath.Join(storeLogsDirName, storeCurrentMetaFile))
}

func storeGatewayLogRelPath(packageID string) string {
	return filepath.ToSlash(filepath.Join(storeLogPackagesDir, packageID, storeGatewayLogFile))
}
