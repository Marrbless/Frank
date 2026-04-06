package missioncontrol

import (
	"errors"
	"os"
	"testing"
	"time"
)

func TestPruneStoreRemovesExpiredLogPackageDirsOlderThanNinetyDays(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 6, 12, 0, 0, 0, time.UTC)
	packageID := "20251231T120000.000000000Z-manual"
	writeLogPackageForTest(t, root, packageID, now.AddDate(0, 0, -91))

	result, err := PruneStore(root, WriterLockLease{LeaseHolderID: "holder-1"}, now)
	if err != nil {
		t.Fatalf("PruneStore() error = %v", err)
	}
	if result.PrunedPackageDirs != 1 {
		t.Fatalf("PruneStore().PrunedPackageDirs = %d, want 1", result.PrunedPackageDirs)
	}
	assertPathNotExists(t, StoreLogPackageDir(root, packageID))
}

func TestPruneStoreKeepsRecentLogPackageDirs(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 6, 12, 0, 0, 0, time.UTC)
	packageID := "20260401T120000.000000000Z-daily"
	writeLogPackageForTest(t, root, packageID, now.AddDate(0, 0, -30))

	result, err := PruneStore(root, WriterLockLease{LeaseHolderID: "holder-1"}, now)
	if err != nil {
		t.Fatalf("PruneStore() error = %v", err)
	}
	if result.PrunedPackageDirs != 0 {
		t.Fatalf("PruneStore().PrunedPackageDirs = %d, want 0", result.PrunedPackageDirs)
	}
	if _, err := os.Stat(StoreLogPackageDir(root, packageID)); err != nil {
		t.Fatalf("Stat(log package dir) error = %v", err)
	}
}

func TestPruneStoreRemovesEligibleVersionedAuditFilesOlderThanNinetyDaysOnlyForTerminalJobs(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 6, 12, 0, 0, 0, time.UTC)
	terminalJobID := "job-terminal"
	nonterminalJobID := "job-running"
	writeJobRuntimeForTest(t, root, terminalJobID, JobStateCompleted, now)
	writeJobRuntimeForTest(t, root, nonterminalJobID, JobStateRunning, now)

	oldTerminalAudit := writeAuditVersionForTest(t, root, terminalJobID, 1, "", "terminal-old", now.AddDate(0, 0, -91))
	oldNonterminalAudit := writeAuditVersionForTest(t, root, nonterminalJobID, 1, "", "running-old", now.AddDate(0, 0, -91))

	result, err := PruneStore(root, WriterLockLease{LeaseHolderID: "holder-1"}, now)
	if err != nil {
		t.Fatalf("PruneStore() error = %v", err)
	}
	if result.PrunedAuditFiles != 1 {
		t.Fatalf("PruneStore().PrunedAuditFiles = %d, want 1", result.PrunedAuditFiles)
	}
	if result.SkippedNonterminalJobTrees != 1 {
		t.Fatalf("PruneStore().SkippedNonterminalJobTrees = %d, want 1", result.SkippedNonterminalJobTrees)
	}
	assertPathNotExists(t, oldTerminalAudit)
	if _, err := os.Stat(oldNonterminalAudit); err != nil {
		t.Fatalf("Stat(nonterminal audit) error = %v", err)
	}
}

func TestPruneStoreRemovesEligibleVersionedApprovalRequestAndGrantFilesOlderThanOneHundredEightyDaysOnlyForTerminalJobs(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 6, 12, 0, 0, 0, time.UTC)
	terminalJobID := "job-terminal"
	nonterminalJobID := "job-running"
	writeJobRuntimeForTest(t, root, terminalJobID, JobStateFailed, now)
	writeJobRuntimeForTest(t, root, nonterminalJobID, JobStatePaused, now)

	oldRequest := writeApprovalRequestVersionForTest(t, root, terminalJobID, "req-old", 1, "", now.AddDate(0, 0, -181))
	oldGrant := writeApprovalGrantVersionForTest(t, root, terminalJobID, "grant-old", 1, "", now.AddDate(0, 0, -181))
	oldRunningRequest := writeApprovalRequestVersionForTest(t, root, nonterminalJobID, "req-running", 1, "", now.AddDate(0, 0, -181))
	oldRunningGrant := writeApprovalGrantVersionForTest(t, root, nonterminalJobID, "grant-running", 1, "", now.AddDate(0, 0, -181))

	result, err := PruneStore(root, WriterLockLease{LeaseHolderID: "holder-1"}, now)
	if err != nil {
		t.Fatalf("PruneStore() error = %v", err)
	}
	if result.PrunedApprovalRequestFiles != 1 {
		t.Fatalf("PruneStore().PrunedApprovalRequestFiles = %d, want 1", result.PrunedApprovalRequestFiles)
	}
	if result.PrunedApprovalGrantFiles != 1 {
		t.Fatalf("PruneStore().PrunedApprovalGrantFiles = %d, want 1", result.PrunedApprovalGrantFiles)
	}
	if result.SkippedNonterminalJobTrees != 1 {
		t.Fatalf("PruneStore().SkippedNonterminalJobTrees = %d, want 1", result.SkippedNonterminalJobTrees)
	}
	assertPathNotExists(t, oldRequest)
	assertPathNotExists(t, oldGrant)
	if _, err := os.Stat(oldRunningRequest); err != nil {
		t.Fatalf("Stat(nonterminal approval request) error = %v", err)
	}
	if _, err := os.Stat(oldRunningGrant); err != nil {
		t.Fatalf("Stat(nonterminal approval grant) error = %v", err)
	}
}

func TestPruneStoreRemovesEligibleVersionedArtifactFilesOlderThanOneHundredEightyDaysOnlyForTerminalJobs(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 6, 12, 0, 0, 0, time.UTC)
	terminalJobID := "job-terminal"
	nonterminalJobID := "job-running"
	writeJobRuntimeForTest(t, root, terminalJobID, JobStateRejected, now)
	writeJobRuntimeForTest(t, root, nonterminalJobID, JobStateWaitingUser, now)

	oldArtifact := writeArtifactVersionForTest(t, root, terminalJobID, "artifact-old", 1, "", now.AddDate(0, 0, -181))
	oldRunningArtifact := writeArtifactVersionForTest(t, root, nonterminalJobID, "artifact-running", 1, "", now.AddDate(0, 0, -181))

	result, err := PruneStore(root, WriterLockLease{LeaseHolderID: "holder-1"}, now)
	if err != nil {
		t.Fatalf("PruneStore() error = %v", err)
	}
	if result.PrunedArtifactFiles != 1 {
		t.Fatalf("PruneStore().PrunedArtifactFiles = %d, want 1", result.PrunedArtifactFiles)
	}
	if result.SkippedNonterminalJobTrees != 1 {
		t.Fatalf("PruneStore().SkippedNonterminalJobTrees = %d, want 1", result.SkippedNonterminalJobTrees)
	}
	assertPathNotExists(t, oldArtifact)
	if _, err := os.Stat(oldRunningArtifact); err != nil {
		t.Fatalf("Stat(nonterminal artifact) error = %v", err)
	}
}

func TestPruneStoreSkipsNonterminalJobTreesEntirely(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 6, 12, 0, 0, 0, time.UTC)
	jobID := "job-running"
	writeJobRuntimeForTest(t, root, jobID, JobStateRunning, now)

	auditPath := writeAuditVersionForTest(t, root, jobID, 1, "", "running-audit", now.AddDate(0, 0, -91))
	requestPath := writeApprovalRequestVersionForTest(t, root, jobID, "req-running", 1, "", now.AddDate(0, 0, -181))
	grantPath := writeApprovalGrantVersionForTest(t, root, jobID, "grant-running", 1, "", now.AddDate(0, 0, -181))
	artifactPath := writeArtifactVersionForTest(t, root, jobID, "artifact-running", 1, "", now.AddDate(0, 0, -181))

	result, err := PruneStore(root, WriterLockLease{LeaseHolderID: "holder-1"}, now)
	if err != nil {
		t.Fatalf("PruneStore() error = %v", err)
	}
	if result.SkippedNonterminalJobTrees != 1 {
		t.Fatalf("PruneStore().SkippedNonterminalJobTrees = %d, want 1", result.SkippedNonterminalJobTrees)
	}
	if result.PrunedAuditFiles != 0 || result.PrunedApprovalRequestFiles != 0 || result.PrunedApprovalGrantFiles != 0 || result.PrunedArtifactFiles != 0 {
		t.Fatalf("PruneStore() pruned nonterminal data: %#v", result)
	}
	for _, path := range []string{auditPath, requestPath, grantPath, artifactPath} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("Stat(%q) error = %v", path, err)
		}
	}
}

func TestPruneStorePreservesNeverPruneFiles(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 6, 12, 0, 0, 0, time.UTC)
	jobID := "job-terminal"
	writeJobRuntimeForTest(t, root, jobID, JobStateCompleted, now)
	if _, err := EnsureCurrentLogSegment(root, now.AddDate(0, 0, -181)); err != nil {
		t.Fatalf("EnsureCurrentLogSegment() error = %v", err)
	}
	if err := os.WriteFile(StoreCurrentLogPath(root), []byte("active\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(current.log) error = %v", err)
	}
	if err := os.WriteFile(StoreRuntimeControlPath(root, jobID), []byte("{\"fixed\":true}\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(runtime_control.json) error = %v", err)
	}
	stepRecord := StepRuntimeRecord{
		RecordVersion: StoreRecordVersion,
		LastSeq:       1,
		JobID:         jobID,
		StepID:        "step-1",
		StepType:      StepTypeDiscussion,
		Status:        StepRuntimeStatusCompleted,
		CompletedAt:   now.AddDate(0, 0, -181),
	}
	if err := StoreStepRuntimeRecord(root, stepRecord); err != nil {
		t.Fatalf("StoreStepRuntimeRecord() error = %v", err)
	}
	if err := StoreBatchCommitRecord(root, BatchCommitRecord{
		RecordVersion: StoreRecordVersion,
		JobID:         jobID,
		Seq:           1,
		AttemptID:     "attempt-1",
		CommittedAt:   now.AddDate(0, 0, -181),
	}); err != nil {
		t.Fatalf("StoreBatchCommitRecord() error = %v", err)
	}
	if err := WriteStoreJSONAtomic(StoreActiveJobPath(root), ActiveJobRecord{
		RecordVersion:  StoreRecordVersion,
		WriterEpoch:    1,
		JobID:          jobID,
		State:          JobStateRunning,
		LeaseHolderID:  "holder-1",
		LeaseExpiresAt: now.Add(time.Minute),
		UpdatedAt:      now,
		ActivationSeq:  1,
	}); err != nil {
		t.Fatalf("WriteStoreJSONAtomic(active_job.json) error = %v", err)
	}
	oldAudit := writeAuditVersionForTest(t, root, jobID, 2, "", "prunable-audit", now.AddDate(0, 0, -91))

	for _, path := range []string{
		StoreCurrentLogPath(root),
		StoreCurrentLogMetaPath(root),
		StoreRuntimeControlPath(root, jobID),
		storeStepRuntimeVersionPath(root, jobID, stepRecord.StepID, stepRecord.LastSeq, stepRecord.AttemptID),
		storeBatchCommitPath(root, jobID, 1),
		StoreActiveJobPath(root),
	} {
		setFileAgeForTest(t, path, now.AddDate(0, 0, -181))
	}

	if _, err := PruneStore(root, WriterLockLease{LeaseHolderID: "holder-1"}, now); err != nil {
		t.Fatalf("PruneStore() error = %v", err)
	}
	assertPathNotExists(t, oldAudit)
	for _, path := range []string{
		StoreManifestPath(root),
		StoreWriterLockPath(root),
		StoreActiveJobPath(root),
		StoreJobRuntimePath(root, jobID),
		StoreRuntimeControlPath(root, jobID),
		storeStepRuntimeVersionPath(root, jobID, stepRecord.StepID, stepRecord.LastSeq, stepRecord.AttemptID),
		storeBatchCommitPath(root, jobID, 1),
		StoreCurrentLogPath(root),
		StoreCurrentLogMetaPath(root),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("Stat(%q) error = %v", path, err)
		}
	}
}

func TestPruneStoreFailsClosedWhenLiveWriterLockHeldByAnotherProcess(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 6, 12, 0, 0, 0, time.UTC)
	if _, _, err := AcquireWriterLock(root, now, time.Minute, WriterLockLease{LeaseHolderID: "holder-1"}); err != nil {
		t.Fatalf("AcquireWriterLock() error = %v", err)
	}

	_, err := PruneStore(root, WriterLockLease{LeaseHolderID: "holder-2"}, now.Add(10*time.Second))
	if !errors.Is(err, ErrWriterLockHeld) {
		t.Fatalf("PruneStore() error = %v, want %v", err, ErrWriterLockHeld)
	}
}

func TestPruneStoreNoOpWhenNothingEligible(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 6, 12, 0, 0, 0, time.UTC)
	jobID := "job-terminal"
	writeJobRuntimeForTest(t, root, jobID, JobStateAborted, now)
	recentPackageID := "20260331T120000.000000000Z-daily"
	writeLogPackageForTest(t, root, recentPackageID, now.AddDate(0, 0, -5))
	recentAudit := writeAuditVersionForTest(t, root, jobID, 1, "", "recent-audit", now.AddDate(0, 0, -5))
	recentRequest := writeApprovalRequestVersionForTest(t, root, jobID, "req-recent", 1, "", now.AddDate(0, 0, -5))
	recentGrant := writeApprovalGrantVersionForTest(t, root, jobID, "grant-recent", 1, "", now.AddDate(0, 0, -5))
	recentArtifact := writeArtifactVersionForTest(t, root, jobID, "artifact-recent", 1, "", now.AddDate(0, 0, -5))

	result, err := PruneStore(root, WriterLockLease{LeaseHolderID: "holder-1"}, now)
	if err != nil {
		t.Fatalf("PruneStore() error = %v", err)
	}
	if result.PrunedPackageDirs != 0 ||
		result.PrunedAuditFiles != 0 ||
		result.PrunedApprovalRequestFiles != 0 ||
		result.PrunedApprovalGrantFiles != 0 ||
		result.PrunedArtifactFiles != 0 ||
		result.SkippedNonterminalJobTrees != 0 {
		t.Fatalf("PruneStore() = %#v, want zero-count no-op", result)
	}
	for _, path := range []string{
		StoreLogPackageDir(root, recentPackageID),
		recentAudit,
		recentRequest,
		recentGrant,
		recentArtifact,
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("Stat(%q) error = %v", path, err)
		}
	}
}

func writeLogPackageForTest(t *testing.T, root string, packageID string, createdAt time.Time) {
	t.Helper()

	if err := ensureStoreChildDir(root, StoreLogPackagesDir(root)); err != nil {
		t.Fatalf("ensureStoreChildDir(log_packages) error = %v", err)
	}
	if err := ensureStoreChildDir(StoreLogPackagesDir(root), StoreLogPackageDir(root, packageID)); err != nil {
		t.Fatalf("ensureStoreChildDir(package dir) error = %v", err)
	}
	if err := os.WriteFile(StoreLogPackageGatewayLogPath(root, packageID), []byte("gateway\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(gateway.log) error = %v", err)
	}
	if err := StoreLogPackageManifestRecord(root, LogPackageManifest{
		RecordVersion:   StoreRecordVersion,
		PackageID:       packageID,
		Reason:          LogPackageReasonManual,
		CreatedAt:       createdAt,
		SegmentOpenedAt: createdAt.Add(-time.Hour),
		SegmentClosedAt: createdAt,
		LogRelPath:      storeGatewayLogRelPath(packageID),
		ByteCount:       int64(len("gateway\n")),
	}); err != nil {
		t.Fatalf("StoreLogPackageManifestRecord() error = %v", err)
	}
}

func writeJobRuntimeForTest(t *testing.T, root string, jobID string, state JobState, now time.Time) {
	t.Helper()

	if err := StoreJobRuntimeRecord(root, JobRuntimeRecord{
		RecordVersion: StoreRecordVersion,
		WriterEpoch:   1,
		AppliedSeq:    1,
		JobID:         jobID,
		State:         state,
		CreatedAt:     now.Add(-time.Hour),
		UpdatedAt:     now,
	}); err != nil {
		t.Fatalf("StoreJobRuntimeRecord() error = %v", err)
	}
}

func writeAuditVersionForTest(t *testing.T, root, jobID string, seq uint64, attemptID string, eventID string, agedAt time.Time) string {
	t.Helper()

	record := AuditEventRecord{
		RecordVersion: StoreRecordVersion,
		Seq:           seq,
		AttemptID:     attemptID,
		Event: AuditEvent{
			EventID:   eventID,
			JobID:     jobID,
			StepID:    "step-1",
			ToolName:  "status",
			Allowed:   true,
			Timestamp: agedAt,
		},
	}
	if err := StoreAuditEventRecord(root, record); err != nil {
		t.Fatalf("StoreAuditEventRecord() error = %v", err)
	}
	path := storeAuditEventVersionPath(root, jobID, seq, attemptID, record.Event.EventID)
	setFileAgeForTest(t, path, agedAt)
	return path
}

func writeApprovalRequestVersionForTest(t *testing.T, root, jobID, requestID string, seq uint64, attemptID string, agedAt time.Time) string {
	t.Helper()

	record := ApprovalRequestRecord{
		RecordVersion:   StoreRecordVersion,
		LastSeq:         seq,
		RequestID:       requestID,
		JobID:           jobID,
		StepID:          "step-1",
		RequestedAction: ApprovalRequestedActionStepComplete,
		Scope:           ApprovalScopeOneStep,
		RequestedVia:    ApprovalRequestedViaRuntime,
		State:           ApprovalStatePending,
		RequestedAt:     agedAt,
	}
	if err := StoreApprovalRequestRecord(root, record); err != nil {
		t.Fatalf("StoreApprovalRequestRecord() error = %v", err)
	}
	path := storeApprovalRequestVersionPath(root, jobID, requestID, seq, attemptID)
	setFileAgeForTest(t, path, agedAt)
	return path
}

func writeApprovalGrantVersionForTest(t *testing.T, root, jobID, grantID string, seq uint64, attemptID string, agedAt time.Time) string {
	t.Helper()

	record := ApprovalGrantRecord{
		RecordVersion:   StoreRecordVersion,
		LastSeq:         seq,
		GrantID:         grantID,
		RequestID:       "request-" + grantID,
		JobID:           jobID,
		StepID:          "step-1",
		RequestedAction: ApprovalRequestedActionStepComplete,
		Scope:           ApprovalScopeOneStep,
		GrantedVia:      ApprovalGrantedViaOperatorCommand,
		State:           ApprovalStateGranted,
		GrantedAt:       agedAt,
	}
	if err := StoreApprovalGrantRecord(root, record); err != nil {
		t.Fatalf("StoreApprovalGrantRecord() error = %v", err)
	}
	path := storeApprovalGrantVersionPath(root, jobID, grantID, seq, attemptID)
	setFileAgeForTest(t, path, agedAt)
	return path
}

func writeArtifactVersionForTest(t *testing.T, root, jobID, artifactID string, seq uint64, attemptID string, agedAt time.Time) string {
	t.Helper()

	record := ArtifactRecord{
		RecordVersion: StoreRecordVersion,
		LastSeq:       seq,
		ArtifactID:    artifactID,
		JobID:         jobID,
		StepID:        "step-1",
		StepType:      StepTypeDiscussion,
		Path:          "/tmp/" + artifactID,
		State:         "verified",
		VerifiedAt:    agedAt,
	}
	if err := StoreArtifactRecord(root, record); err != nil {
		t.Fatalf("StoreArtifactRecord() error = %v", err)
	}
	path := storeArtifactVersionPath(root, jobID, artifactID, seq, attemptID)
	setFileAgeForTest(t, path, agedAt)
	return path
}

func setFileAgeForTest(t *testing.T, path string, at time.Time) {
	t.Helper()

	if err := os.Chtimes(path, at, at); err != nil {
		t.Fatalf("Chtimes(%q) error = %v", path, err)
	}
}

func assertPathNotExists(t *testing.T, path string) {
	t.Helper()

	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return
	} else if err != nil {
		t.Fatalf("Stat(%q) error = %v, want os.ErrNotExist", path, err)
	}
	t.Fatalf("Stat(%q) error = nil, want os.ErrNotExist", path)
}
