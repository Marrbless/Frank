package missioncontrol

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	storePruneLeaseDuration      = time.Minute
	storeLogPackageRetentionDays = 90
	storeAuditRetentionDays      = 90
	storeApprovalRetentionDays   = 180
	storeArtifactRetentionDays   = 180
)

type PruneStoreResult struct {
	StoreRoot                  string
	PrunedPackageDirs          int
	PrunedAuditFiles           int
	PrunedApprovalRequestFiles int
	PrunedApprovalGrantFiles   int
	PrunedArtifactFiles        int
	SkippedNonterminalJobTrees int
}

func PruneStore(root string, lease WriterLockLease, now time.Time) (PruneStoreResult, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return PruneStoreResult{}, err
	}
	now = normalizeStoreLogTime(now)

	lock, err := acquireStorePruneWriterLock(root, lease, now)
	if err != nil {
		return PruneStoreResult{}, err
	}

	guard, err := acquireStoreWriterGuard(root)
	if err != nil {
		return PruneStoreResult{}, err
	}
	defer func() { _ = guard.release() }()

	if err := ValidateHeldWriterLock(root, lock, now); err != nil {
		return PruneStoreResult{}, err
	}

	result := PruneStoreResult{StoreRoot: root}
	logPackageCutoff := now.AddDate(0, 0, -storeLogPackageRetentionDays)
	auditCutoff := now.AddDate(0, 0, -storeAuditRetentionDays)
	approvalCutoff := now.AddDate(0, 0, -storeApprovalRetentionDays)
	artifactCutoff := now.AddDate(0, 0, -storeArtifactRetentionDays)

	prunedPackages, err := pruneExpiredLogPackageDirs(root, logPackageCutoff)
	if err != nil {
		return PruneStoreResult{}, err
	}
	result.PrunedPackageDirs = prunedPackages

	jobIDs, err := listStoreRecordKeys(StoreJobsDir(root))
	if err != nil {
		return PruneStoreResult{}, err
	}
	for _, jobID := range jobIDs {
		hasCandidates, err := jobHasPruneCandidates(root, jobID)
		if err != nil {
			return PruneStoreResult{}, err
		}
		if !hasCandidates {
			continue
		}

		runtime, err := LoadJobRuntimeRecord(root, jobID)
		if err != nil {
			return PruneStoreResult{}, err
		}
		if !IsTerminalJobState(runtime.State) {
			result.SkippedNonterminalJobTrees++
			continue
		}

		prunedAudit, err := pruneExpiredVersionedAuditFiles(root, jobID, auditCutoff)
		if err != nil {
			return PruneStoreResult{}, err
		}
		result.PrunedAuditFiles += prunedAudit

		prunedApprovalRequests, err := pruneExpiredVersionedFilesByKeyDir(StoreApprovalRequestsDir(root, jobID), approvalCutoff)
		if err != nil {
			return PruneStoreResult{}, err
		}
		result.PrunedApprovalRequestFiles += prunedApprovalRequests

		prunedApprovalGrants, err := pruneExpiredVersionedFilesByKeyDir(StoreApprovalGrantsDir(root, jobID), approvalCutoff)
		if err != nil {
			return PruneStoreResult{}, err
		}
		result.PrunedApprovalGrantFiles += prunedApprovalGrants

		prunedArtifacts, err := pruneExpiredVersionedFilesByKeyDir(StoreArtifactsDir(root, jobID), artifactCutoff)
		if err != nil {
			return PruneStoreResult{}, err
		}
		result.PrunedArtifactFiles += prunedArtifacts
	}

	return result, nil
}

func acquireStorePruneWriterLock(root string, lease WriterLockLease, now time.Time) (WriterLockRecord, error) {
	lock, _, err := AcquireWriterLock(root, now, storePruneLeaseDuration, lease)
	if err == nil {
		return lock, nil
	}
	if errors.Is(err, ErrWriterLockExpired) {
		return TakeoverWriterLock(root, now, storePruneLeaseDuration, lease)
	}
	return WriterLockRecord{}, err
}

func pruneExpiredLogPackageDirs(root string, cutoff time.Time) (int, error) {
	entries, err := os.ReadDir(StoreLogPackagesDir(root))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return 0, nil
		}
		return 0, err
	}

	pruned := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		manifest, err := LoadLogPackageManifest(root, entry.Name())
		if err != nil {
			return 0, err
		}
		if manifest.PackageID != entry.Name() {
			return 0, fmt.Errorf("mission store log package manifest package_id %q does not match directory %q", manifest.PackageID, entry.Name())
		}
		if !manifest.CreatedAt.UTC().Before(cutoff) {
			continue
		}

		if err := os.RemoveAll(StoreLogPackageDir(root, entry.Name())); err != nil {
			return 0, err
		}
		if err := storeSyncDir(StoreLogPackagesDir(root)); err != nil {
			return 0, err
		}
		pruned++
	}
	return pruned, nil
}

func jobHasPruneCandidates(root string, jobID string) (bool, error) {
	paths := []string{
		StoreAuditDir(root, jobID),
		StoreApprovalRequestsDir(root, jobID),
		StoreApprovalGrantsDir(root, jobID),
		StoreArtifactsDir(root, jobID),
	}
	for _, path := range paths {
		info, err := os.Stat(path)
		switch {
		case err == nil:
			if !info.IsDir() {
				return false, fmt.Errorf("mission store prune candidate path %q must be a directory", path)
			}
			return true, nil
		case errors.Is(err, os.ErrNotExist):
			continue
		default:
			return false, err
		}
	}
	return false, nil
}

func pruneExpiredVersionedAuditFiles(root, jobID string, cutoff time.Time) (int, error) {
	return pruneExpiredVersionedFilesInDir(StoreAuditDir(root, jobID), cutoff, isStoreVersionedAuditFilename)
}

func pruneExpiredVersionedFilesByKeyDir(root string, cutoff time.Time) (int, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return 0, nil
		}
		return 0, err
	}

	pruned := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		count, err := pruneExpiredVersionedFilesInDir(filepath.Join(root, entry.Name()), cutoff, isStoreVersionFilename)
		if err != nil {
			return 0, err
		}
		pruned += count
	}
	return pruned, nil
}

func pruneExpiredVersionedFilesInDir(dir string, cutoff time.Time, matches func(string) bool) (int, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return 0, nil
		}
		return 0, err
	}

	pruned := 0
	for _, entry := range entries {
		if entry.IsDir() || !matches(entry.Name()) {
			continue
		}

		path := filepath.Join(dir, entry.Name())
		info, err := entry.Info()
		if err != nil {
			return 0, err
		}
		if !info.ModTime().UTC().Before(cutoff) {
			continue
		}
		if err := os.Remove(path); err != nil {
			return 0, err
		}
		if err := storeSyncDir(dir); err != nil {
			return 0, err
		}
		pruned++
	}
	return pruned, nil
}

func isStoreVersionedAuditFilename(name string) bool {
	if !strings.HasSuffix(name, ".json") {
		return false
	}
	base := strings.TrimSuffix(name, ".json")
	if !strings.Contains(base, "--") {
		return false
	}
	parts := strings.Split(base, "--")
	if len(parts) < 2 || len(parts) > 3 {
		return false
	}
	return isStoreSeqComponent(parts[0])
}

func isStoreVersionFilename(name string) bool {
	if !strings.HasSuffix(name, ".json") {
		return false
	}
	base := strings.TrimSuffix(name, ".json")
	parts := strings.Split(base, "--")
	if len(parts) < 1 || len(parts) > 2 {
		return false
	}
	return isStoreSeqComponent(parts[0])
}

func isStoreSeqComponent(value string) bool {
	if len(value) != 20 {
		return false
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
