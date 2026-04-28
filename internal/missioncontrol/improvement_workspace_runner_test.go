package missioncontrol

import (
	"strings"
	"testing"
	"time"
)

func TestStoreImprovementWorkspaceRunRecordCrashLeavesActivePointerUnchanged(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 26, 10, 0, 0, 0, time.UTC)
	storeImprovementWorkspaceRunFixtures(t, root, now)

	beforeBytes := mustReadFileBytes(t, StoreActiveRuntimePackPointerPath(root))
	beforePointer, err := LoadActiveRuntimePackPointer(root)
	if err != nil {
		t.Fatalf("LoadActiveRuntimePackPointer(before) error = %v", err)
	}
	snapshot := ImprovementWorkspaceActivePointerSnapshotFromPointer(beforePointer)

	record := validImprovementWorkspaceRunRecord(now.Add(7*time.Minute), func(record *ImprovementWorkspaceRunRecord) {
		record.ActivePointerAtStart = snapshot
		record.ActivePointerAtCompletion = snapshot
	})
	stored, created, err := StoreImprovementWorkspaceRunRecord(root, record)
	if err != nil {
		t.Fatalf("StoreImprovementWorkspaceRunRecord() error = %v", err)
	}
	if !created {
		t.Fatal("StoreImprovementWorkspaceRunRecord() created = false, want true")
	}
	assertBytesEqual(t, "active runtime-pack pointer", beforeBytes, mustReadFileBytes(t, StoreActiveRuntimePackPointerPath(root)))

	if stored.Outcome != ImprovementWorkspaceRunOutcomeCrashed {
		t.Fatalf("Outcome = %q, want crashed", stored.Outcome)
	}
	if stored.ActivePointerAtStart.ActivePackID != "pack-base" || stored.ActivePointerAtCompletion.ActivePackID != "pack-base" {
		t.Fatalf("active pointer snapshots = %#v / %#v, want pack-base", stored.ActivePointerAtStart, stored.ActivePointerAtCompletion)
	}

	replayed, replayCreated, err := StoreImprovementWorkspaceRunRecord(root, record)
	if err != nil {
		t.Fatalf("StoreImprovementWorkspaceRunRecord(replay) error = %v", err)
	}
	if replayCreated {
		t.Fatal("StoreImprovementWorkspaceRunRecord(replay) created = true, want false")
	}
	if replayed.WorkspaceRunID != stored.WorkspaceRunID {
		t.Fatalf("replayed WorkspaceRunID = %q, want %q", replayed.WorkspaceRunID, stored.WorkspaceRunID)
	}
	assertBytesEqual(t, "active runtime-pack pointer replay", beforeBytes, mustReadFileBytes(t, StoreActiveRuntimePackPointerPath(root)))
}

func TestStoreImprovementWorkspaceRunRecordRejectsActivePointerDrift(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 26, 11, 0, 0, 0, time.UTC)
	storeImprovementWorkspaceRunFixtures(t, root, now)

	beforePointer, err := LoadActiveRuntimePackPointer(root)
	if err != nil {
		t.Fatalf("LoadActiveRuntimePackPointer(before) error = %v", err)
	}
	staleSnapshot := ImprovementWorkspaceActivePointerSnapshotFromPointer(beforePointer)

	if err := StoreActiveRuntimePackPointer(root, ActiveRuntimePackPointer{
		ActivePackID:         "pack-candidate",
		PreviousActivePackID: "pack-base",
		LastKnownGoodPackID:  "pack-base",
		UpdatedAt:            now.Add(7 * time.Minute),
		UpdatedBy:            "hot-update-gate",
		UpdateRecordRef:      "hot-update-1",
		ReloadGeneration:     1,
	}); err != nil {
		t.Fatalf("StoreActiveRuntimePackPointer(mutated) error = %v", err)
	}
	mutatedBytes := mustReadFileBytes(t, StoreActiveRuntimePackPointerPath(root))

	_, _, err = StoreImprovementWorkspaceRunRecord(root, validImprovementWorkspaceRunRecord(now.Add(8*time.Minute), func(record *ImprovementWorkspaceRunRecord) {
		record.ActivePointerAtStart = staleSnapshot
		record.ActivePointerAtCompletion = staleSnapshot
	}))
	if err == nil {
		t.Fatal("StoreImprovementWorkspaceRunRecord(pointer drift) error = nil, want rejection")
	}
	if !strings.Contains(err.Error(), "active pointer completion snapshot does not match current active pointer") {
		t.Fatalf("StoreImprovementWorkspaceRunRecord(pointer drift) error = %q, want active pointer drift context", err.Error())
	}
	assertBytesEqual(t, "active runtime-pack pointer after drift rejection", mutatedBytes, mustReadFileBytes(t, StoreActiveRuntimePackPointerPath(root)))
}

func TestStoreImprovementWorkspaceRunRecordValidationFailsClosed(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
	storeImprovementWorkspaceRunFixtures(t, root, now)
	pointer, err := LoadActiveRuntimePackPointer(root)
	if err != nil {
		t.Fatalf("LoadActiveRuntimePackPointer() error = %v", err)
	}
	snapshot := ImprovementWorkspaceActivePointerSnapshotFromPointer(pointer)

	tests := []struct {
		name string
		edit func(*ImprovementWorkspaceRunRecord)
		want string
	}{
		{
			name: "missing failure reason",
			edit: func(record *ImprovementWorkspaceRunRecord) {
				record.FailureReason = " "
			},
			want: "failure_reason is required",
		},
		{
			name: "completion before start",
			edit: func(record *ImprovementWorkspaceRunRecord) {
				record.CompletedAt = record.StartedAt.Add(-time.Second)
			},
			want: "completed_at must not precede started_at",
		},
		{
			name: "completion pointer differs from start",
			edit: func(record *ImprovementWorkspaceRunRecord) {
				record.ActivePointerAtCompletion.ReloadGeneration++
			},
			want: "active pointer changed between start and completion",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			record := validImprovementWorkspaceRunRecord(now.Add(9*time.Minute), func(record *ImprovementWorkspaceRunRecord) {
				record.WorkspaceRunID = "workspace-run-" + strings.ReplaceAll(tc.name, " ", "-")
				record.ActivePointerAtStart = snapshot
				record.ActivePointerAtCompletion = snapshot
				tc.edit(record)
			})
			_, _, err := StoreImprovementWorkspaceRunRecord(root, record)
			if err == nil {
				t.Fatalf("StoreImprovementWorkspaceRunRecord() error = nil, want %q", tc.want)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("StoreImprovementWorkspaceRunRecord() error = %q, want %q", err.Error(), tc.want)
			}
		})
	}
}

func validImprovementWorkspaceRunRecord(now time.Time, mutate func(*ImprovementWorkspaceRunRecord)) ImprovementWorkspaceRunRecord {
	record := ImprovementWorkspaceRunRecord{
		WorkspaceRunID: "workspace-run-1",
		RunID:          "run-root",
		CandidateID:    "candidate-1",
		ExecutionHost:  "phone",
		Outcome:        ImprovementWorkspaceRunOutcomeCrashed,
		FailureReason:  "local deterministic workspace crash fixture",
		StartedAt:      now,
		CompletedAt:    now.Add(time.Minute),
		CreatedAt:      now.Add(2 * time.Minute),
		CreatedBy:      "system",
	}
	if mutate != nil {
		mutate(&record)
	}
	return record
}

func storeImprovementWorkspaceRunFixtures(t *testing.T, root string, now time.Time) {
	t.Helper()

	storeImprovementRunFixtures(t, root, now)
	if err := StoreImprovementRunRecord(root, validImprovementRunRecord(now.Add(5*time.Minute), func(record *ImprovementRunRecord) {
		record.State = ImprovementRunStateMutating
		record.Decision = ""
		record.CompletedAt = time.Time{}
		record.StopReason = ""
	})); err != nil {
		t.Fatalf("StoreImprovementRunRecord() error = %v", err)
	}
	if err := StoreActiveRuntimePackPointer(root, ActiveRuntimePackPointer{
		ActivePackID:        "pack-base",
		LastKnownGoodPackID: "pack-base",
		UpdatedAt:           now.Add(6 * time.Minute),
		UpdatedBy:           "bootstrap",
		UpdateRecordRef:     "bootstrap-active",
		ReloadGeneration:    0,
	}); err != nil {
		t.Fatalf("StoreActiveRuntimePackPointer() error = %v", err)
	}
}
