package missioncontrol

import (
	"errors"
	"testing"
	"time"
)

func TestStoreAndLoadActiveJobRecord(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	record, err := NewActiveJobRecord(
		1,
		"job-1",
		JobStateRunning,
		"step-1",
		"holder-1",
		time.Date(2026, 4, 4, 15, 5, 0, 0, time.UTC),
		time.Date(2026, 4, 4, 15, 0, 0, 0, time.UTC),
		7,
	)
	if err != nil {
		t.Fatalf("NewActiveJobRecord() error = %v", err)
	}

	if err := StoreActiveJobRecord(root, record); err != nil {
		t.Fatalf("StoreActiveJobRecord() error = %v", err)
	}

	loaded, err := LoadActiveJobRecord(root)
	if err != nil {
		t.Fatalf("LoadActiveJobRecord() error = %v", err)
	}
	if loaded.JobID != record.JobID {
		t.Fatalf("LoadActiveJobRecord().JobID = %q, want %q", loaded.JobID, record.JobID)
	}
	if loaded.ActivationSeq != record.ActivationSeq {
		t.Fatalf("LoadActiveJobRecord().ActivationSeq = %d, want %d", loaded.ActivationSeq, record.ActivationSeq)
	}
}

func TestLoadActiveJobRecordMissingReturnsSentinel(t *testing.T) {
	t.Parallel()

	_, err := LoadActiveJobRecord(t.TempDir())
	if !errors.Is(err, ErrActiveJobRecordNotFound) {
		t.Fatalf("LoadActiveJobRecord() error = %v, want %v", err, ErrActiveJobRecordNotFound)
	}
}

func TestReconcileActiveJobRecordClearsTerminalOrStaleOccupancy(t *testing.T) {
	t.Parallel()

	record, err := NewActiveJobRecord(
		2,
		"job-1",
		JobStatePaused,
		"step-1",
		"holder-1",
		time.Date(2026, 4, 4, 15, 5, 0, 0, time.UTC),
		time.Date(2026, 4, 4, 15, 0, 0, 0, time.UTC),
		9,
	)
	if err != nil {
		t.Fatalf("NewActiveJobRecord() error = %v", err)
	}

	next, changed := ReconcileActiveJobRecord(record, "job-1", JobStateCompleted, "", 9)
	if !changed {
		t.Fatal("ReconcileActiveJobRecord(terminal) changed = false, want true")
	}
	if next != nil {
		t.Fatalf("ReconcileActiveJobRecord(terminal) = %#v, want nil", next)
	}

	next, changed = ReconcileActiveJobRecord(record, "job-1", JobStatePaused, "step-1", 8)
	if !changed {
		t.Fatal("ReconcileActiveJobRecord(stale seq) changed = false, want true")
	}
	if next != nil {
		t.Fatalf("ReconcileActiveJobRecord(stale seq) = %#v, want nil", next)
	}

	next, changed = ReconcileActiveJobRecord(record, "job-1", JobStateFailed, "step-1", 9)
	if !changed {
		t.Fatal("ReconcileActiveJobRecord(non-occupancy) changed = false, want true")
	}
	if next != nil {
		t.Fatalf("ReconcileActiveJobRecord(non-occupancy) = %#v, want nil", next)
	}
}
