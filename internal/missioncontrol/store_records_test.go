package missioncontrol

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func testJobRuntimeRecord(now time.Time, writerEpoch uint64, appliedSeq uint64) JobRuntimeRecord {
	return JobRuntimeRecord{
		RecordVersion: StoreRecordVersion,
		WriterEpoch:   writerEpoch,
		AppliedSeq:    appliedSeq,
		JobID:         "job-1",
		State:         JobStateRunning,
		ActiveStepID:  "step-1",
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}

func testStepRuntimeRecord(lastSeq uint64) StepRuntimeRecord {
	return StepRuntimeRecord{
		RecordVersion: StoreRecordVersion,
		LastSeq:       lastSeq,
		JobID:         "job-1",
		StepID:        "step-1",
		StepType:      StepTypeDiscussion,
		Status:        StepRuntimeStatusActive,
		Reason:        "step-reason",
	}
}

func testRuntimeControlRecord(lastSeq uint64, writerEpoch uint64) RuntimeControlRecord {
	return RuntimeControlRecord{
		RecordVersion: StoreRecordVersion,
		WriterEpoch:   writerEpoch,
		LastSeq:       lastSeq,
		JobID:         "job-1",
		StepID:        "step-1",
		MaxAuthority:  AuthorityTierHigh,
		AllowedTools:  []string{"exec"},
		Step: Step{
			ID:           "step-1",
			Type:         StepTypeDiscussion,
			AllowedTools: []string{"exec"},
		},
	}
}

func testApprovalRequestRecord(now time.Time, lastSeq uint64, state ApprovalState) ApprovalRequestRecord {
	return ApprovalRequestRecord{
		RecordVersion:   StoreRecordVersion,
		LastSeq:         lastSeq,
		RequestID:       "req-1",
		JobID:           "job-1",
		StepID:          "step-1",
		RequestedAction: "approve",
		Scope:           "mission",
		RequestedVia:    "operator",
		State:           state,
		RequestedAt:     now,
	}
}

func testArtifactRecord(lastSeq uint64, state string, path string) ArtifactRecord {
	return ArtifactRecord{
		RecordVersion: StoreRecordVersion,
		LastSeq:       lastSeq,
		ArtifactID:    "artifact-1",
		JobID:         "job-1",
		StepID:        "step-1",
		StepType:      StepTypeDiscussion,
		Path:          path,
		State:         state,
	}
}

func TestCommittedStepRecordsIgnoreSeqBeyondAppliedSeq(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 4, 15, 0, 0, 0, time.UTC)

	if err := StoreJobRuntimeRecord(root, testJobRuntimeRecord(now, 1, 1)); err != nil {
		t.Fatalf("StoreJobRuntimeRecord() error = %v", err)
	}
	if err := StoreStepRuntimeRecord(root, testStepRuntimeRecord(2)); err != nil {
		t.Fatalf("StoreStepRuntimeRecord() error = %v", err)
	}

	records, err := ListCommittedStepRuntimeRecords(root, "job-1")
	if err != nil {
		t.Fatalf("ListCommittedStepRuntimeRecords() error = %v", err)
	}
	if len(records) != 0 {
		t.Fatalf("ListCommittedStepRuntimeRecords() len = %d, want 0", len(records))
	}
}

func TestCommittedAuditRecordsIgnoreSeqBeyondAppliedSeq(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 4, 15, 0, 0, 0, time.UTC)

	if err := StoreJobRuntimeRecord(root, testJobRuntimeRecord(now, 1, 1)); err != nil {
		t.Fatalf("StoreJobRuntimeRecord() error = %v", err)
	}

	visible := AuditEventRecord{
		RecordVersion: StoreRecordVersion,
		Seq:           1,
		Event: AuditEvent{
			JobID:     "job-1",
			StepID:    "step-1",
			ToolName:  "echo",
			Timestamp: now,
			Allowed:   true,
		},
	}
	if err := StoreAuditEventRecord(root, visible); err != nil {
		t.Fatalf("StoreAuditEventRecord(visible) error = %v", err)
	}

	invisible := AuditEventRecord{
		RecordVersion: StoreRecordVersion,
		Seq:           2,
		Event: AuditEvent{
			JobID:     "job-1",
			StepID:    "step-1",
			ToolName:  "ls",
			Timestamp: now.Add(time.Second),
			Allowed:   true,
		},
	}
	if err := StoreAuditEventRecord(root, invisible); err != nil {
		t.Fatalf("StoreAuditEventRecord(invisible) error = %v", err)
	}

	records, err := ListCommittedAuditEventRecords(root, "job-1")
	if err != nil {
		t.Fatalf("ListCommittedAuditEventRecords() error = %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("ListCommittedAuditEventRecords() len = %d, want 1", len(records))
	}
	if records[0].Seq != 1 {
		t.Fatalf("ListCommittedAuditEventRecords()[0].Seq = %d, want 1", records[0].Seq)
	}
}

func TestLoadCommittedRuntimeControlRecordInvisibleWhenSeqBeyondAppliedSeq(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 4, 15, 0, 0, 0, time.UTC)

	if err := StoreJobRuntimeRecord(root, testJobRuntimeRecord(now, 1, 1)); err != nil {
		t.Fatalf("StoreJobRuntimeRecord() error = %v", err)
	}

	if err := StoreRuntimeControlRecord(root, testRuntimeControlRecord(2, 1)); err != nil {
		t.Fatalf("StoreRuntimeControlRecord() error = %v", err)
	}

	_, err := LoadCommittedRuntimeControlRecord(root, "job-1")
	if !errors.Is(err, ErrRuntimeControlRecordNotFound) {
		t.Fatalf("LoadCommittedRuntimeControlRecord() error = %v, want %v", err, ErrRuntimeControlRecordNotFound)
	}
}

func TestCommittedReadsIgnoreStrayTempAndNonJSONFiles(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 4, 15, 0, 0, 0, time.UTC)

	if err := StoreJobRuntimeRecord(root, testJobRuntimeRecord(now, 1, 1)); err != nil {
		t.Fatalf("StoreJobRuntimeRecord() error = %v", err)
	}
	if err := StoreStepRuntimeRecord(root, testStepRuntimeRecord(1)); err != nil {
		t.Fatalf("StoreStepRuntimeRecord() error = %v", err)
	}
	if err := StoreRuntimeControlRecord(root, testRuntimeControlRecord(1, 1)); err != nil {
		t.Fatalf("StoreRuntimeControlRecord() error = %v", err)
	}

	stepDir := storeStepRuntimeVersionsDir(root, "job-1", "step-1")
	for _, name := range []string{
		filepath.Join(storeStepsDir(root, "job-1"), "notes.txt"),
		filepath.Join(stepDir, "00000000000000000002.tmp"),
		filepath.Join(stepDir, "garbage.txt"),
	} {
		if err := os.WriteFile(name, []byte("junk"), 0o644); err != nil {
			t.Fatalf("WriteFile(%q) error = %v", name, err)
		}
	}

	controlDir := storeRuntimeControlVersionsDir(root, "job-1")
	for _, name := range []string{
		filepath.Join(controlDir, "runtime_control.tmp-123"),
		filepath.Join(controlDir, "readme.txt"),
	} {
		if err := os.WriteFile(name, []byte("junk"), 0o644); err != nil {
			t.Fatalf("WriteFile(%q) error = %v", name, err)
		}
	}

	stepRecords, err := ListCommittedStepRuntimeRecords(root, "job-1")
	if err != nil {
		t.Fatalf("ListCommittedStepRuntimeRecords() error = %v", err)
	}
	if len(stepRecords) != 1 {
		t.Fatalf("ListCommittedStepRuntimeRecords() len = %d, want 1", len(stepRecords))
	}

	control, err := LoadCommittedRuntimeControlRecord(root, "job-1")
	if err != nil {
		t.Fatalf("LoadCommittedRuntimeControlRecord() error = %v", err)
	}
	if control.LastSeq != 1 {
		t.Fatalf("LoadCommittedRuntimeControlRecord().LastSeq = %d, want 1", control.LastSeq)
	}
}
