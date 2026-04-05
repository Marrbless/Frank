package missioncontrol

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func testHeldWriterLock(t *testing.T, root string, now time.Time) WriterLockRecord {
	t.Helper()

	lock, _, err := AcquireWriterLock(root, now, time.Minute, WriterLockLease{LeaseHolderID: "holder-1"})
	if err != nil {
		t.Fatalf("AcquireWriterLock() error = %v", err)
	}
	return lock
}

func testCommittedStepRuntimeRecord(t *testing.T, root string, lock WriterLockRecord, now time.Time, seq uint64, reason string) {
	t.Helper()

	record := testStepRuntimeRecord(seq)
	record.Reason = reason
	if err := CommitStoreBatch(root, lock, StoreBatch{
		JobRuntime:  testJobRuntimeRecord(now, lock.WriterEpoch, seq),
		StepRecords: []StepRuntimeRecord{record},
	}); err != nil {
		t.Fatalf("CommitStoreBatch(step) error = %v", err)
	}
}

func testCommittedRuntimeControlRecord(t *testing.T, root string, lock WriterLockRecord, now time.Time, seq uint64, authority AuthorityTier) {
	t.Helper()

	record := testRuntimeControlRecord(seq, lock.WriterEpoch)
	record.MaxAuthority = authority
	if err := CommitStoreBatch(root, lock, StoreBatch{
		JobRuntime:     testJobRuntimeRecord(now, lock.WriterEpoch, seq),
		RuntimeControl: &record,
	}); err != nil {
		t.Fatalf("CommitStoreBatch(runtime_control) error = %v", err)
	}
}

func testCommittedApprovalRequest(t *testing.T, root string, lock WriterLockRecord, now time.Time, seq uint64, state ApprovalState) {
	t.Helper()

	record := testApprovalRequestRecord(now, seq, state)
	if err := CommitStoreBatch(root, lock, StoreBatch{
		JobRuntime:       testJobRuntimeRecord(now, lock.WriterEpoch, seq),
		ApprovalRequests: []ApprovalRequestRecord{record},
	}); err != nil {
		t.Fatalf("CommitStoreBatch(approval_request) error = %v", err)
	}
}

func testCommittedArtifact(t *testing.T, root string, lock WriterLockRecord, now time.Time, seq uint64, state string, path string) {
	t.Helper()

	record := testArtifactRecord(seq, state, path)
	if err := CommitStoreBatch(root, lock, StoreBatch{
		JobRuntime: testJobRuntimeRecord(now, lock.WriterEpoch, seq),
		Artifacts:  []ArtifactRecord{record},
	}); err != nil {
		t.Fatalf("CommitStoreBatch(artifact) error = %v", err)
	}
}

func testCommittedAuditEvent(t *testing.T, root string, lock WriterLockRecord, now time.Time, seq uint64, toolName string) {
	t.Helper()

	if err := CommitStoreBatch(root, lock, StoreBatch{
		JobRuntime: testJobRuntimeRecord(now, lock.WriterEpoch, seq),
		AuditEvents: []AuditEventRecord{{
			RecordVersion: StoreRecordVersion,
			Seq:           seq,
			Event: AuditEvent{
				JobID:     "job-1",
				StepID:    "step-1",
				ToolName:  toolName,
				Timestamp: now,
				Allowed:   true,
			},
		}},
	}); err != nil {
		t.Fatalf("CommitStoreBatch(audit) error = %v", err)
	}
}

func testCommittedActiveJob(t *testing.T, root string, lock WriterLockRecord, now time.Time, seq uint64) {
	t.Helper()

	if err := CommitStoreBatch(root, lock, StoreBatch{
		JobRuntime: testJobRuntimeRecord(now, lock.WriterEpoch, seq),
		ActiveJob:  ptrActiveJobRecord(testActiveJobRecord(t, lock.WriterEpoch, now, seq)),
	}); err != nil {
		t.Fatalf("CommitStoreBatch(active_job) error = %v", err)
	}
}

func testActiveJobRecord(t *testing.T, writerEpoch uint64, when time.Time, activationSeq uint64) ActiveJobRecord {
	t.Helper()

	record, err := NewActiveJobRecord(
		writerEpoch,
		"job-1",
		JobStateRunning,
		"step-1",
		"holder-1",
		when.Add(time.Minute),
		when,
		activationSeq,
	)
	if err != nil {
		t.Fatalf("NewActiveJobRecord() error = %v", err)
	}
	return record
}

func TestCommitStoreBatchWritesJobRuntimeLast(t *testing.T) {
	root := t.TempDir()
	now := time.Now().UTC()
	lock := testHeldWriterLock(t, root, now)

	originalHook := storeBatchBeforeMutation
	t.Cleanup(func() { storeBatchBeforeMutation = originalHook })

	var order []string
	storeBatchBeforeMutation = func(path string) error {
		order = append(order, path)
		return nil
	}

	batch := StoreBatch{
		JobRuntime:     testJobRuntimeRecord(now, lock.WriterEpoch, 1),
		RuntimeControl: ptrRuntimeControlRecord(testRuntimeControlRecord(1, lock.WriterEpoch)),
		StepRecords:    []StepRuntimeRecord{testStepRuntimeRecord(1)},
		ActiveJob:      ptrActiveJobRecord(testActiveJobRecord(t, lock.WriterEpoch, now, 1)),
	}
	if err := CommitStoreBatch(root, lock, batch); err != nil {
		t.Fatalf("CommitStoreBatch() error = %v", err)
	}

	if len(order) < 3 {
		t.Fatalf("mutation order len = %d, want >= 3", len(order))
	}
	if order[len(order)-1] != StoreJobRuntimePath(root, "job-1") {
		t.Fatalf("last mutation path = %q, want %q", order[len(order)-1], StoreJobRuntimePath(root, "job-1"))
	}
	if indexOfPath(order, StoreActiveJobPath(root)) >= len(order)-1 {
		t.Fatalf("active_job mutation order = %#v, want before job runtime", order)
	}
}

func TestCommitStoreBatchExistingCommittedStepSurvivesFailedOverwriteBatch(t *testing.T) {
	root := t.TempDir()
	now := time.Now().UTC()
	lock := testHeldWriterLock(t, root, now)

	testCommittedStepRuntimeRecord(t, root, lock, now, 1, "committed-seq-1")

	originalHook := storeBatchBeforeMutation
	t.Cleanup(func() { storeBatchBeforeMutation = originalHook })
	storeBatchBeforeMutation = func(path string) error {
		if path == StoreJobRuntimePath(root, "job-1") {
			return fmt.Errorf("forced job runtime write failure")
		}
		return nil
	}

	next := testStepRuntimeRecord(2)
	next.Reason = "orphan-seq-2"
	if err := CommitStoreBatch(root, lock, StoreBatch{
		JobRuntime:  testJobRuntimeRecord(now.Add(time.Minute), lock.WriterEpoch, 2),
		StepRecords: []StepRuntimeRecord{next},
	}); err == nil {
		t.Fatal("CommitStoreBatch() error = nil, want forced failure")
	}

	records, err := ListCommittedStepRuntimeRecords(root, "job-1")
	if err != nil {
		t.Fatalf("ListCommittedStepRuntimeRecords() error = %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("ListCommittedStepRuntimeRecords() len = %d, want 1", len(records))
	}
	if records[0].Reason != "committed-seq-1" {
		t.Fatalf("ListCommittedStepRuntimeRecords()[0].Reason = %q, want %q", records[0].Reason, "committed-seq-1")
	}
}

func TestCommitStoreBatchFailedSeqOneStepRetryDoesNotSurfaceOrphan(t *testing.T) {
	root := t.TempDir()
	now := time.Now().UTC()
	lock := testHeldWriterLock(t, root, now)

	originalHook := storeBatchBeforeMutation
	t.Cleanup(func() { storeBatchBeforeMutation = originalHook })
	storeBatchBeforeMutation = func(path string) error {
		if path == StoreJobRuntimePath(root, "job-1") {
			return fmt.Errorf("forced job runtime write failure")
		}
		return nil
	}

	if err := CommitStoreBatch(root, lock, StoreBatch{
		JobRuntime:  testJobRuntimeRecord(now, lock.WriterEpoch, 1),
		StepRecords: []StepRuntimeRecord{testStepRuntimeRecord(1)},
	}); err == nil {
		t.Fatal("CommitStoreBatch(first failed seq1) error = nil, want forced failure")
	}

	storeBatchBeforeMutation = func(string) error { return nil }
	if err := CommitStoreBatch(root, lock, StoreBatch{
		JobRuntime: testJobRuntimeRecord(now.Add(time.Minute), lock.WriterEpoch, 1),
	}); err != nil {
		t.Fatalf("CommitStoreBatch(successful retry seq1) error = %v", err)
	}

	records, err := ListCommittedStepRuntimeRecords(root, "job-1")
	if err != nil {
		t.Fatalf("ListCommittedStepRuntimeRecords() error = %v", err)
	}
	if len(records) != 0 {
		t.Fatalf("ListCommittedStepRuntimeRecords() len = %d, want 0", len(records))
	}
}

func TestCommitStoreBatchExistingCommittedRuntimeControlSurvivesFailedOverwriteBatch(t *testing.T) {
	root := t.TempDir()
	now := time.Now().UTC()
	lock := testHeldWriterLock(t, root, now)

	testCommittedRuntimeControlRecord(t, root, lock, now, 1, AuthorityTierLow)

	originalHook := storeBatchBeforeMutation
	t.Cleanup(func() { storeBatchBeforeMutation = originalHook })
	storeBatchBeforeMutation = func(path string) error {
		if path == StoreJobRuntimePath(root, "job-1") {
			return fmt.Errorf("forced job runtime write failure")
		}
		return nil
	}

	next := testRuntimeControlRecord(2, lock.WriterEpoch)
	next.MaxAuthority = AuthorityTierHigh
	if err := CommitStoreBatch(root, lock, StoreBatch{
		JobRuntime:     testJobRuntimeRecord(now.Add(time.Minute), lock.WriterEpoch, 2),
		RuntimeControl: &next,
	}); err == nil {
		t.Fatal("CommitStoreBatch() error = nil, want forced failure")
	}

	record, err := LoadCommittedRuntimeControlRecord(root, "job-1")
	if err != nil {
		t.Fatalf("LoadCommittedRuntimeControlRecord() error = %v", err)
	}
	if record.MaxAuthority != AuthorityTierLow {
		t.Fatalf("LoadCommittedRuntimeControlRecord().MaxAuthority = %q, want %q", record.MaxAuthority, AuthorityTierLow)
	}
}

func TestCommitStoreBatchFailedSeqOneRuntimeControlRetryDoesNotSurfaceOrphan(t *testing.T) {
	root := t.TempDir()
	now := time.Now().UTC()
	lock := testHeldWriterLock(t, root, now)

	originalHook := storeBatchBeforeMutation
	t.Cleanup(func() { storeBatchBeforeMutation = originalHook })
	storeBatchBeforeMutation = func(path string) error {
		if path == StoreJobRuntimePath(root, "job-1") {
			return fmt.Errorf("forced job runtime write failure")
		}
		return nil
	}

	if err := CommitStoreBatch(root, lock, StoreBatch{
		JobRuntime:     testJobRuntimeRecord(now, lock.WriterEpoch, 1),
		RuntimeControl: ptrRuntimeControlRecord(testRuntimeControlRecord(1, lock.WriterEpoch)),
	}); err == nil {
		t.Fatal("CommitStoreBatch(first failed runtime_control seq1) error = nil, want forced failure")
	}

	storeBatchBeforeMutation = func(string) error { return nil }
	if err := CommitStoreBatch(root, lock, StoreBatch{
		JobRuntime: testJobRuntimeRecord(now.Add(time.Minute), lock.WriterEpoch, 1),
	}); err != nil {
		t.Fatalf("CommitStoreBatch(successful retry seq1) error = %v", err)
	}

	_, err := LoadCommittedRuntimeControlRecord(root, "job-1")
	if !errors.Is(err, ErrRuntimeControlRecordNotFound) {
		t.Fatalf("LoadCommittedRuntimeControlRecord() error = %v, want %v", err, ErrRuntimeControlRecordNotFound)
	}
}

func TestCommitStoreBatchExistingCommittedApprovalRecordSurvivesFailedOverwriteBatch(t *testing.T) {
	root := t.TempDir()
	now := time.Now().UTC()
	lock := testHeldWriterLock(t, root, now)

	testCommittedApprovalRequest(t, root, lock, now, 1, ApprovalStatePending)

	originalHook := storeBatchBeforeMutation
	t.Cleanup(func() { storeBatchBeforeMutation = originalHook })
	storeBatchBeforeMutation = func(path string) error {
		if path == StoreJobRuntimePath(root, "job-1") {
			return fmt.Errorf("forced job runtime write failure")
		}
		return nil
	}

	next := testApprovalRequestRecord(now.Add(time.Minute), 2, ApprovalStateGranted)
	if err := CommitStoreBatch(root, lock, StoreBatch{
		JobRuntime:       testJobRuntimeRecord(now.Add(time.Minute), lock.WriterEpoch, 2),
		ApprovalRequests: []ApprovalRequestRecord{next},
	}); err == nil {
		t.Fatal("CommitStoreBatch() error = nil, want forced failure")
	}

	records, err := ListCommittedApprovalRequestRecords(root, "job-1")
	if err != nil {
		t.Fatalf("ListCommittedApprovalRequestRecords() error = %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("ListCommittedApprovalRequestRecords() len = %d, want 1", len(records))
	}
	if records[0].State != ApprovalStatePending {
		t.Fatalf("ListCommittedApprovalRequestRecords()[0].State = %q, want %q", records[0].State, ApprovalStatePending)
	}
}

func TestCommitStoreBatchFailedSeqOneApprovalRetryDoesNotSurfaceOrphan(t *testing.T) {
	root := t.TempDir()
	now := time.Now().UTC()
	lock := testHeldWriterLock(t, root, now)

	originalHook := storeBatchBeforeMutation
	t.Cleanup(func() { storeBatchBeforeMutation = originalHook })
	storeBatchBeforeMutation = func(path string) error {
		if path == StoreJobRuntimePath(root, "job-1") {
			return fmt.Errorf("forced job runtime write failure")
		}
		return nil
	}

	if err := CommitStoreBatch(root, lock, StoreBatch{
		JobRuntime:       testJobRuntimeRecord(now, lock.WriterEpoch, 1),
		ApprovalRequests: []ApprovalRequestRecord{testApprovalRequestRecord(now, 1, ApprovalStatePending)},
	}); err == nil {
		t.Fatal("CommitStoreBatch(first failed approval seq1) error = nil, want forced failure")
	}

	storeBatchBeforeMutation = func(string) error { return nil }
	if err := CommitStoreBatch(root, lock, StoreBatch{
		JobRuntime: testJobRuntimeRecord(now.Add(time.Minute), lock.WriterEpoch, 1),
	}); err != nil {
		t.Fatalf("CommitStoreBatch(successful retry seq1) error = %v", err)
	}

	records, err := ListCommittedApprovalRequestRecords(root, "job-1")
	if err != nil {
		t.Fatalf("ListCommittedApprovalRequestRecords() error = %v", err)
	}
	if len(records) != 0 {
		t.Fatalf("ListCommittedApprovalRequestRecords() len = %d, want 0", len(records))
	}
}

func TestCommitStoreBatchExistingCommittedArtifactSurvivesFailedOverwriteBatch(t *testing.T) {
	root := t.TempDir()
	now := time.Now().UTC()
	lock := testHeldWriterLock(t, root, now)

	testCommittedArtifact(t, root, lock, now, 1, "verified", "artifacts/out-1.txt")

	originalHook := storeBatchBeforeMutation
	t.Cleanup(func() { storeBatchBeforeMutation = originalHook })
	storeBatchBeforeMutation = func(path string) error {
		if path == StoreJobRuntimePath(root, "job-1") {
			return fmt.Errorf("forced job runtime write failure")
		}
		return nil
	}

	next := testArtifactRecord(2, "failed", "artifacts/out-2.txt")
	if err := CommitStoreBatch(root, lock, StoreBatch{
		JobRuntime: testJobRuntimeRecord(now.Add(time.Minute), lock.WriterEpoch, 2),
		Artifacts:  []ArtifactRecord{next},
	}); err == nil {
		t.Fatal("CommitStoreBatch() error = nil, want forced failure")
	}

	records, err := ListCommittedArtifactRecords(root, "job-1")
	if err != nil {
		t.Fatalf("ListCommittedArtifactRecords() error = %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("ListCommittedArtifactRecords() len = %d, want 1", len(records))
	}
	if records[0].Path != "artifacts/out-1.txt" {
		t.Fatalf("ListCommittedArtifactRecords()[0].Path = %q, want %q", records[0].Path, "artifacts/out-1.txt")
	}
}

func TestCommitStoreBatchFailedSeqOneAuditRetryDoesNotSurfaceOrphan(t *testing.T) {
	root := t.TempDir()
	now := time.Now().UTC()
	lock := testHeldWriterLock(t, root, now)

	originalHook := storeBatchBeforeMutation
	t.Cleanup(func() { storeBatchBeforeMutation = originalHook })
	storeBatchBeforeMutation = func(path string) error {
		if path == StoreJobRuntimePath(root, "job-1") {
			return fmt.Errorf("forced job runtime write failure")
		}
		return nil
	}

	if err := CommitStoreBatch(root, lock, StoreBatch{
		JobRuntime: testJobRuntimeRecord(now, lock.WriterEpoch, 1),
		AuditEvents: []AuditEventRecord{{
			RecordVersion: StoreRecordVersion,
			Seq:           1,
			Event: AuditEvent{
				JobID:     "job-1",
				StepID:    "step-1",
				ToolName:  "orphan-audit",
				Timestamp: now,
				Allowed:   true,
			},
		}},
	}); err == nil {
		t.Fatal("CommitStoreBatch(first failed audit seq1) error = nil, want forced failure")
	}

	storeBatchBeforeMutation = func(string) error { return nil }
	if err := CommitStoreBatch(root, lock, StoreBatch{
		JobRuntime: testJobRuntimeRecord(now.Add(time.Minute), lock.WriterEpoch, 1),
	}); err != nil {
		t.Fatalf("CommitStoreBatch(successful retry seq1) error = %v", err)
	}

	records, err := ListCommittedAuditEventRecords(root, "job-1")
	if err != nil {
		t.Fatalf("ListCommittedAuditEventRecords() error = %v", err)
	}
	if len(records) != 0 {
		t.Fatalf("ListCommittedAuditEventRecords() len = %d, want 0", len(records))
	}
}

func TestCommitStoreBatchFailedSeqOneArtifactRetryDoesNotSurfaceOrphan(t *testing.T) {
	root := t.TempDir()
	now := time.Now().UTC()
	lock := testHeldWriterLock(t, root, now)

	originalHook := storeBatchBeforeMutation
	t.Cleanup(func() { storeBatchBeforeMutation = originalHook })
	storeBatchBeforeMutation = func(path string) error {
		if path == StoreJobRuntimePath(root, "job-1") {
			return fmt.Errorf("forced job runtime write failure")
		}
		return nil
	}

	if err := CommitStoreBatch(root, lock, StoreBatch{
		JobRuntime: testJobRuntimeRecord(now, lock.WriterEpoch, 1),
		Artifacts:  []ArtifactRecord{testArtifactRecord(1, "verified", "artifacts/orphan.txt")},
	}); err == nil {
		t.Fatal("CommitStoreBatch(first failed artifact seq1) error = nil, want forced failure")
	}

	storeBatchBeforeMutation = func(string) error { return nil }
	if err := CommitStoreBatch(root, lock, StoreBatch{
		JobRuntime: testJobRuntimeRecord(now.Add(time.Minute), lock.WriterEpoch, 1),
	}); err != nil {
		t.Fatalf("CommitStoreBatch(successful retry seq1) error = %v", err)
	}

	records, err := ListCommittedArtifactRecords(root, "job-1")
	if err != nil {
		t.Fatalf("ListCommittedArtifactRecords() error = %v", err)
	}
	if len(records) != 0 {
		t.Fatalf("ListCommittedArtifactRecords() len = %d, want 0", len(records))
	}
}

func TestCommitStoreBatchRefusesWhenManifestLockCoherenceBroken(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Now().UTC()
	lock := testHeldWriterLock(t, root, now)

	manifest, err := LoadStoreManifest(root)
	if err != nil {
		t.Fatalf("LoadStoreManifest() error = %v", err)
	}
	manifest.CurrentWriterEpoch++
	if err := StoreManifestRecord(root, manifest); err != nil {
		t.Fatalf("StoreManifestRecord() error = %v", err)
	}

	err = CommitStoreBatch(root, lock, StoreBatch{JobRuntime: testJobRuntimeRecord(now, lock.WriterEpoch, 1)})
	if !errors.Is(err, ErrWriterEpochIncoherent) {
		t.Fatalf("CommitStoreBatch() error = %v, want %v", err, ErrWriterEpochIncoherent)
	}
}

func TestCommitStoreBatchRejectsRepeatedOrRegressedAppliedSeq(t *testing.T) {
	root := t.TempDir()
	now := time.Now().UTC()
	lock := testHeldWriterLock(t, root, now)

	if err := CommitStoreBatch(root, lock, StoreBatch{
		JobRuntime: testJobRuntimeRecord(now, lock.WriterEpoch, 1),
	}); err != nil {
		t.Fatalf("CommitStoreBatch(seq1) error = %v", err)
	}

	repeatedErr := CommitStoreBatch(root, lock, StoreBatch{
		JobRuntime: testJobRuntimeRecord(now.Add(time.Minute), lock.WriterEpoch, 1),
	})
	if repeatedErr == nil || !strings.Contains(repeatedErr.Error(), "must advance from committed seq 1") {
		t.Fatalf("CommitStoreBatch(repeated) error = %v, want applied_seq advance failure", repeatedErr)
	}

	if err := CommitStoreBatch(root, lock, StoreBatch{
		JobRuntime: testJobRuntimeRecord(now.Add(2*time.Minute), lock.WriterEpoch, 2),
	}); err != nil {
		t.Fatalf("CommitStoreBatch(seq2) error = %v", err)
	}

	regressedErr := CommitStoreBatch(root, lock, StoreBatch{
		JobRuntime: testJobRuntimeRecord(now.Add(3*time.Minute), lock.WriterEpoch, 1),
	})
	if regressedErr == nil || !strings.Contains(regressedErr.Error(), "must advance from committed seq 2") {
		t.Fatalf("CommitStoreBatch(regressed) error = %v, want applied_seq advance failure", regressedErr)
	}
}

func TestCommitStoreBatchRejectsInvalidActiveJobMetadata(t *testing.T) {
	root := t.TempDir()
	now := time.Now().UTC()
	lock := testHeldWriterLock(t, root, now)

	tests := []struct {
		name      string
		mutate    func(*StoreBatch)
		wantError string
	}{
		{
			name: "epoch mismatch",
			mutate: func(batch *StoreBatch) {
				batch.ActiveJob.WriterEpoch = lock.WriterEpoch + 1
			},
			wantError: "active job writer_epoch",
		},
		{
			name: "activation seq mismatch",
			mutate: func(batch *StoreBatch) {
				batch.ActiveJob.ActivationSeq = batch.JobRuntime.AppliedSeq + 1
			},
			wantError: "active job activation_seq",
		},
		{
			name: "non occupancy state",
			mutate: func(batch *StoreBatch) {
				batch.JobRuntime.State = JobStateCompleted
			},
			wantError: "requires occupancy state",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			batch := StoreBatch{
				JobRuntime: testJobRuntimeRecord(now, lock.WriterEpoch, 1),
				ActiveJob:  ptrActiveJobRecord(testActiveJobRecord(t, lock.WriterEpoch, now, 1)),
			}
			tc.mutate(&batch)
			err := CommitStoreBatch(root, lock, batch)
			if err == nil || !strings.Contains(err.Error(), tc.wantError) {
				t.Fatalf("CommitStoreBatch() error = %v, want substring %q", err, tc.wantError)
			}
		})
	}
}

func TestCommitStoreBatchFailedSeqTwoActiveJobRetryDoesNotResurrectStaleActiveJob(t *testing.T) {
	root := t.TempDir()
	now := time.Now().UTC()
	lock := testHeldWriterLock(t, root, now)

	testCommittedActiveJob(t, root, lock, now, 1)

	originalHook := storeBatchBeforeMutation
	t.Cleanup(func() { storeBatchBeforeMutation = originalHook })
	storeBatchBeforeMutation = func(path string) error {
		if path == StoreJobRuntimePath(root, "job-1") {
			return fmt.Errorf("forced job runtime write failure")
		}
		return nil
	}

	if err := CommitStoreBatch(root, lock, StoreBatch{
		JobRuntime: testJobRuntimeRecord(now.Add(time.Minute), lock.WriterEpoch, 2),
		ActiveJob:  ptrActiveJobRecord(testActiveJobRecord(t, lock.WriterEpoch, now.Add(time.Minute), 2)),
	}); err == nil {
		t.Fatal("CommitStoreBatch(failed seq2 active_job) error = nil, want forced failure")
	}

	storeBatchBeforeMutation = func(string) error { return nil }
	jobRuntime := testJobRuntimeRecord(now.Add(2*time.Minute), lock.WriterEpoch, 2)
	jobRuntime.State = JobStateCompleted
	if err := CommitStoreBatch(root, lock, StoreBatch{
		JobRuntime: jobRuntime,
	}); err != nil {
		t.Fatalf("CommitStoreBatch(successful seq2 non-occupancy) error = %v", err)
	}

	_, err := LoadCommittedActiveJobRecord(root, "job-1")
	if !errors.Is(err, ErrActiveJobRecordNotFound) {
		t.Fatalf("LoadCommittedActiveJobRecord() error = %v, want %v", err, ErrActiveJobRecordNotFound)
	}
}

func TestCommitStoreBatchSameSeqRetriesShowOnlyWinningAttempt(t *testing.T) {
	root := t.TempDir()
	now := time.Now().UTC()
	lock := testHeldWriterLock(t, root, now)

	originalHook := storeBatchBeforeMutation
	t.Cleanup(func() { storeBatchBeforeMutation = originalHook })

	failCount := 0
	storeBatchBeforeMutation = func(path string) error {
		if path == StoreJobRuntimePath(root, "job-1") && failCount < 2 {
			failCount++
			return fmt.Errorf("forced job runtime write failure")
		}
		return nil
	}

	first := testStepRuntimeRecord(1)
	first.Reason = "first-attempt"
	if err := CommitStoreBatch(root, lock, StoreBatch{
		JobRuntime:  testJobRuntimeRecord(now, lock.WriterEpoch, 1),
		StepRecords: []StepRuntimeRecord{first},
	}); err == nil {
		t.Fatal("CommitStoreBatch(first retry) error = nil, want forced failure")
	}

	second := testStepRuntimeRecord(1)
	second.Reason = "second-attempt"
	if err := CommitStoreBatch(root, lock, StoreBatch{
		JobRuntime:  testJobRuntimeRecord(now.Add(time.Minute), lock.WriterEpoch, 1),
		StepRecords: []StepRuntimeRecord{second},
	}); err == nil {
		t.Fatal("CommitStoreBatch(second retry) error = nil, want forced failure")
	}

	storeBatchBeforeMutation = func(string) error { return nil }
	winning := testStepRuntimeRecord(1)
	winning.Reason = "winning-attempt"
	if err := CommitStoreBatch(root, lock, StoreBatch{
		JobRuntime:  testJobRuntimeRecord(now.Add(2*time.Minute), lock.WriterEpoch, 1),
		StepRecords: []StepRuntimeRecord{winning},
	}); err != nil {
		t.Fatalf("CommitStoreBatch(winning retry) error = %v", err)
	}

	records, err := ListCommittedStepRuntimeRecords(root, "job-1")
	if err != nil {
		t.Fatalf("ListCommittedStepRuntimeRecords() error = %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("ListCommittedStepRuntimeRecords() len = %d, want 1", len(records))
	}
	if records[0].Reason != "winning-attempt" {
		t.Fatalf("ListCommittedStepRuntimeRecords()[0].Reason = %q, want %q", records[0].Reason, "winning-attempt")
	}
}

func TestCommitStoreBatchSameSeqAuditRetriesShowOnlyWinningAttempt(t *testing.T) {
	root := t.TempDir()
	now := time.Now().UTC()
	lock := testHeldWriterLock(t, root, now)

	originalHook := storeBatchBeforeMutation
	t.Cleanup(func() { storeBatchBeforeMutation = originalHook })

	failCount := 0
	storeBatchBeforeMutation = func(path string) error {
		if path == StoreJobRuntimePath(root, "job-1") && failCount < 2 {
			failCount++
			return fmt.Errorf("forced job runtime write failure")
		}
		return nil
	}

	firstAudit := AuditEventRecord{
		RecordVersion: StoreRecordVersion,
		Seq:           1,
		Event: AuditEvent{
			JobID:     "job-1",
			StepID:    "step-1",
			ToolName:  "first-audit",
			Timestamp: now,
			Allowed:   true,
		},
	}
	if err := CommitStoreBatch(root, lock, StoreBatch{
		JobRuntime: testJobRuntimeRecord(now, lock.WriterEpoch, 1),
		AuditEvents: []AuditEventRecord{
			firstAudit,
		},
	}); err == nil {
		t.Fatal("CommitStoreBatch(first audit retry) error = nil, want forced failure")
	}

	secondAudit := AuditEventRecord{
		RecordVersion: StoreRecordVersion,
		Seq:           1,
		Event: AuditEvent{
			JobID:     "job-1",
			StepID:    "step-1",
			ToolName:  "second-audit",
			Timestamp: now.Add(time.Second),
			Allowed:   true,
		},
	}
	if err := CommitStoreBatch(root, lock, StoreBatch{
		JobRuntime: testJobRuntimeRecord(now.Add(time.Minute), lock.WriterEpoch, 1),
		AuditEvents: []AuditEventRecord{
			secondAudit,
		},
	}); err == nil {
		t.Fatal("CommitStoreBatch(second audit retry) error = nil, want forced failure")
	}

	storeBatchBeforeMutation = func(string) error { return nil }
	winningAudit := AuditEventRecord{
		RecordVersion: StoreRecordVersion,
		Seq:           1,
		Event: AuditEvent{
			JobID:     "job-1",
			StepID:    "step-1",
			ToolName:  "winning-audit",
			Timestamp: now.Add(2 * time.Second),
			Allowed:   true,
		},
	}
	if err := CommitStoreBatch(root, lock, StoreBatch{
		JobRuntime: testJobRuntimeRecord(now.Add(2*time.Minute), lock.WriterEpoch, 1),
		AuditEvents: []AuditEventRecord{
			winningAudit,
		},
	}); err != nil {
		t.Fatalf("CommitStoreBatch(winning audit retry) error = %v", err)
	}

	records, err := ListCommittedAuditEventRecords(root, "job-1")
	if err != nil {
		t.Fatalf("ListCommittedAuditEventRecords() error = %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("ListCommittedAuditEventRecords() len = %d, want 1", len(records))
	}
	if records[0].Event.ToolName != "winning-audit" {
		t.Fatalf("ListCommittedAuditEventRecords()[0].Event.ToolName = %q, want %q", records[0].Event.ToolName, "winning-audit")
	}
}

func TestCommitStoreBatchPriorCommittedAuditSeqSurvivesFailedOverwriteAtNextSeq(t *testing.T) {
	root := t.TempDir()
	now := time.Now().UTC()
	lock := testHeldWriterLock(t, root, now)

	testCommittedAuditEvent(t, root, lock, now, 1, "committed-audit-seq1")

	originalHook := storeBatchBeforeMutation
	t.Cleanup(func() { storeBatchBeforeMutation = originalHook })
	storeBatchBeforeMutation = func(path string) error {
		if path == StoreJobRuntimePath(root, "job-1") {
			return fmt.Errorf("forced job runtime write failure")
		}
		return nil
	}

	if err := CommitStoreBatch(root, lock, StoreBatch{
		JobRuntime: testJobRuntimeRecord(now.Add(time.Minute), lock.WriterEpoch, 2),
		AuditEvents: []AuditEventRecord{{
			RecordVersion: StoreRecordVersion,
			Seq:           2,
			Event: AuditEvent{
				JobID:     "job-1",
				StepID:    "step-1",
				ToolName:  "orphan-audit-seq2",
				Timestamp: now.Add(time.Minute),
				Allowed:   true,
			},
		}},
	}); err == nil {
		t.Fatal("CommitStoreBatch(failed audit seq2) error = nil, want forced failure")
	}

	records, err := ListCommittedAuditEventRecords(root, "job-1")
	if err != nil {
		t.Fatalf("ListCommittedAuditEventRecords() error = %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("ListCommittedAuditEventRecords() len = %d, want 1", len(records))
	}
	if records[0].Event.ToolName != "committed-audit-seq1" {
		t.Fatalf("ListCommittedAuditEventRecords()[0].Event.ToolName = %q, want %q", records[0].Event.ToolName, "committed-audit-seq1")
	}
}

func TestCommitStoreBatchActiveJobRemovalParticipatesInOrdering(t *testing.T) {
	root := t.TempDir()
	now := time.Now().UTC()
	lock := testHeldWriterLock(t, root, now)

	if err := CommitStoreBatch(root, lock, StoreBatch{
		JobRuntime: testJobRuntimeRecord(now, lock.WriterEpoch, 1),
		ActiveJob:  ptrActiveJobRecord(testActiveJobRecord(t, lock.WriterEpoch, now, 1)),
	}); err != nil {
		t.Fatalf("CommitStoreBatch(initial active job) error = %v", err)
	}

	originalHook := storeBatchBeforeMutation
	t.Cleanup(func() { storeBatchBeforeMutation = originalHook })

	var order []string
	storeBatchBeforeMutation = func(path string) error {
		order = append(order, path)
		return nil
	}

	batch := StoreBatch{
		JobRuntime:      testJobRuntimeRecord(now.Add(time.Minute), lock.WriterEpoch, 2),
		RemoveActiveJob: true,
	}
	batch.JobRuntime.State = JobStateCompleted
	if err := CommitStoreBatch(root, lock, batch); err != nil {
		t.Fatalf("CommitStoreBatch() error = %v", err)
	}

	if indexOfPath(order, StoreActiveJobPath(root)) == -1 {
		t.Fatalf("mutation order = %#v, want active_job removal path", order)
	}
	if order[len(order)-1] != StoreJobRuntimePath(root, "job-1") {
		t.Fatalf("last mutation path = %q, want %q", order[len(order)-1], StoreJobRuntimePath(root, "job-1"))
	}
	if _, err := LoadActiveJobRecord(root); !errors.Is(err, ErrActiveJobRecordNotFound) {
		t.Fatalf("LoadActiveJobRecord() error = %v, want %v", err, ErrActiveJobRecordNotFound)
	}
}

func TestCommittedAuditRecordsIgnoreOrphanBatchWrites(t *testing.T) {
	root := t.TempDir()
	now := time.Now().UTC()
	lock := testHeldWriterLock(t, root, now)

	originalHook := storeBatchBeforeMutation
	t.Cleanup(func() { storeBatchBeforeMutation = originalHook })
	storeBatchBeforeMutation = func(path string) error {
		if path == StoreJobRuntimePath(root, "job-1") {
			return fmt.Errorf("forced job runtime write failure")
		}
		return nil
	}

	batch := StoreBatch{
		JobRuntime: testJobRuntimeRecord(now, lock.WriterEpoch, 1),
		AuditEvents: []AuditEventRecord{{
			RecordVersion: StoreRecordVersion,
			Seq:           1,
			Event: AuditEvent{
				JobID:     "job-1",
				StepID:    "step-1",
				ToolName:  "echo",
				Timestamp: now,
				Allowed:   true,
			},
		}},
	}
	if err := CommitStoreBatch(root, lock, batch); err == nil {
		t.Fatal("CommitStoreBatch() error = nil, want forced failure")
	}

	records, err := ListCommittedAuditEventRecords(root, "job-1")
	if err != nil {
		t.Fatalf("ListCommittedAuditEventRecords() error = %v", err)
	}
	if len(records) != 0 {
		t.Fatalf("ListCommittedAuditEventRecords() len = %d, want 0", len(records))
	}
}

func ptrRuntimeControlRecord(record RuntimeControlRecord) *RuntimeControlRecord {
	return &record
}

func ptrActiveJobRecord(record ActiveJobRecord) *ActiveJobRecord {
	return &record
}

func indexOfPath(paths []string, want string) int {
	for i, path := range paths {
		if filepath.Clean(path) == filepath.Clean(want) {
			return i
		}
	}
	return -1
}
