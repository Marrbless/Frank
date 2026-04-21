package missioncontrol

import (
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestImprovementCandidateRecordRoundTripAndList(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 22, 15, 0, 0, 0, time.FixedZone("offset", -4*60*60))

	mustStoreRuntimePack(t, root, validRuntimePackRecord(now, func(record *RuntimePackRecord) {
		record.PackID = "pack-base"
		record.SourceSummary = "baseline pack"
	}))
	mustStoreRuntimePack(t, root, validRuntimePackRecord(now.Add(time.Minute), func(record *RuntimePackRecord) {
		record.PackID = "pack-candidate-b"
		record.ParentPackID = "pack-base"
		record.RollbackTargetPackID = "pack-base"
		record.SourceSummary = "second candidate pack"
	}))
	mustStoreRuntimePack(t, root, validRuntimePackRecord(now.Add(2*time.Minute), func(record *RuntimePackRecord) {
		record.PackID = "pack-candidate-a"
		record.ParentPackID = "pack-base"
		record.RollbackTargetPackID = "pack-base"
		record.SourceSummary = "first candidate pack"
	}))

	if err := StoreHotUpdateGateRecord(root, validHotUpdateGateRecord(now.Add(3*time.Minute), func(record *HotUpdateGateRecord) {
		record.HotUpdateID = "hot-update-a"
		record.CandidatePackID = "pack-candidate-a"
		record.PreviousActivePackID = "pack-base"
		record.RollbackTargetPackID = "pack-base"
	})); err != nil {
		t.Fatalf("StoreHotUpdateGateRecord(hot-update-a) error = %v", err)
	}

	second := validImprovementCandidateRecord(now.Add(4*time.Minute), func(record *ImprovementCandidateRecord) {
		record.CandidateID = "candidate-b"
		record.CandidatePackID = "pack-candidate-b"
		record.SourceWorkspaceRef = ""
		record.SourceSummary = "second candidate summary"
		record.ValidationBasisRefs = []string{"eval/baseline"}
		record.HotUpdateID = ""
	})
	if err := StoreImprovementCandidateRecord(root, second); err != nil {
		t.Fatalf("StoreImprovementCandidateRecord(candidate-b) error = %v", err)
	}

	want := validImprovementCandidateRecord(now.Add(5*time.Minute), func(record *ImprovementCandidateRecord) {
		record.CandidateID = " candidate-a "
		record.BaselinePackID = " pack-base "
		record.CandidatePackID = " pack-candidate-a "
		record.SourceWorkspaceRef = " workspace/runs/run-1 "
		record.SourceSummary = " candidate seeded from workspace "
		record.ValidationBasisRefs = []string{" eval/baseline ", " eval/holdout "}
		record.HotUpdateID = " hot-update-a "
		record.CreatedBy = " operator "
	})
	if err := StoreImprovementCandidateRecord(root, want); err != nil {
		t.Fatalf("StoreImprovementCandidateRecord(candidate-a) error = %v", err)
	}

	got, err := LoadImprovementCandidateRecord(root, "candidate-a")
	if err != nil {
		t.Fatalf("LoadImprovementCandidateRecord() error = %v", err)
	}

	want.RecordVersion = StoreRecordVersion
	want.CandidateID = "candidate-a"
	want.BaselinePackID = "pack-base"
	want.CandidatePackID = "pack-candidate-a"
	want.SourceWorkspaceRef = "workspace/runs/run-1"
	want.SourceSummary = "candidate seeded from workspace"
	want.ValidationBasisRefs = []string{"eval/baseline", "eval/holdout"}
	want.HotUpdateID = "hot-update-a"
	want.CreatedAt = want.CreatedAt.UTC()
	want.CreatedBy = "operator"
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("LoadImprovementCandidateRecord() = %#v, want %#v", got, want)
	}

	records, err := ListImprovementCandidateRecords(root)
	if err != nil {
		t.Fatalf("ListImprovementCandidateRecords() error = %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("ListImprovementCandidateRecords() len = %d, want 2", len(records))
	}
	if records[0].CandidateID != "candidate-a" || records[1].CandidateID != "candidate-b" {
		t.Fatalf("ListImprovementCandidateRecords() ids = [%q %q], want [candidate-a candidate-b]", records[0].CandidateID, records[1].CandidateID)
	}
}

func TestImprovementCandidateReplayIsIdempotent(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 22, 16, 0, 0, 0, time.UTC)

	mustStoreRuntimePack(t, root, validRuntimePackRecord(now, func(record *RuntimePackRecord) {
		record.PackID = "pack-base"
	}))
	mustStoreRuntimePack(t, root, validRuntimePackRecord(now.Add(time.Minute), func(record *RuntimePackRecord) {
		record.PackID = "pack-candidate"
		record.ParentPackID = "pack-base"
		record.RollbackTargetPackID = "pack-base"
	}))

	record := validImprovementCandidateRecord(now.Add(2*time.Minute), func(candidate *ImprovementCandidateRecord) {
		candidate.CandidateID = "candidate-replay"
		candidate.BaselinePackID = "pack-base"
		candidate.CandidatePackID = "pack-candidate"
	})
	if err := StoreImprovementCandidateRecord(root, record); err != nil {
		t.Fatalf("StoreImprovementCandidateRecord(first) error = %v", err)
	}
	firstBytes, err := os.ReadFile(StoreImprovementCandidatePath(root, record.CandidateID))
	if err != nil {
		t.Fatalf("ReadFile(first) error = %v", err)
	}

	if err := StoreImprovementCandidateRecord(root, record); err != nil {
		t.Fatalf("StoreImprovementCandidateRecord(replay) error = %v", err)
	}
	secondBytes, err := os.ReadFile(StoreImprovementCandidatePath(root, record.CandidateID))
	if err != nil {
		t.Fatalf("ReadFile(second) error = %v", err)
	}

	if string(firstBytes) != string(secondBytes) {
		t.Fatalf("improvement candidate file changed on idempotent replay\nfirst:\n%s\nsecond:\n%s", string(firstBytes), string(secondBytes))
	}
}

func TestImprovementCandidateValidationFailsClosed(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 22, 17, 0, 0, 0, time.UTC)

	mustStoreRuntimePack(t, root, validRuntimePackRecord(now, func(record *RuntimePackRecord) {
		record.PackID = "pack-base"
	}))
	mustStoreRuntimePack(t, root, validRuntimePackRecord(now.Add(time.Minute), func(record *RuntimePackRecord) {
		record.PackID = "pack-candidate"
		record.ParentPackID = "pack-base"
		record.RollbackTargetPackID = "pack-base"
	}))

	tests := []struct {
		name string
		run  func() error
		want string
	}{
		{
			name: "missing candidate id",
			run: func() error {
				return StoreImprovementCandidateRecord(root, validImprovementCandidateRecord(now.Add(2*time.Minute), func(record *ImprovementCandidateRecord) {
					record.CandidateID = " "
				}))
			},
			want: "improvement candidate ref candidate_id is required",
		},
		{
			name: "missing source fields",
			run: func() error {
				return StoreImprovementCandidateRecord(root, validImprovementCandidateRecord(now.Add(2*time.Minute), func(record *ImprovementCandidateRecord) {
					record.SourceWorkspaceRef = ""
					record.SourceSummary = ""
				}))
			},
			want: "mission store improvement candidate requires source_workspace_ref or source_summary",
		},
		{
			name: "missing validation basis",
			run: func() error {
				return StoreImprovementCandidateRecord(root, validImprovementCandidateRecord(now.Add(2*time.Minute), func(record *ImprovementCandidateRecord) {
					record.ValidationBasisRefs = nil
				}))
			},
			want: "mission store improvement candidate validation_basis_refs are required",
		},
		{
			name: "invalid hot update id",
			run: func() error {
				return StoreImprovementCandidateRecord(root, validImprovementCandidateRecord(now.Add(2*time.Minute), func(record *ImprovementCandidateRecord) {
					record.HotUpdateID = "bad/id"
				}))
			},
			want: `mission store improvement candidate hot_update_id "bad/id"`,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := tc.run()
			if err == nil {
				t.Fatal("StoreImprovementCandidateRecord() error = nil, want fail-closed rejection")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("StoreImprovementCandidateRecord() error = %q, want substring %q", err.Error(), tc.want)
			}
		})
	}
}

func TestImprovementCandidateRejectsMissingRefsAndInvalidLinkage(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 22, 18, 0, 0, 0, time.UTC)

	err := StoreImprovementCandidateRecord(root, validImprovementCandidateRecord(now, func(record *ImprovementCandidateRecord) {
		record.BaselinePackID = "missing-base"
		record.CandidatePackID = "missing-candidate"
	}))
	if err == nil {
		t.Fatal("StoreImprovementCandidateRecord() error = nil, want missing pack rejection")
	}
	if !strings.Contains(err.Error(), ErrRuntimePackRecordNotFound.Error()) {
		t.Fatalf("StoreImprovementCandidateRecord() error = %q, want missing pack rejection", err.Error())
	}

	mustStoreRuntimePack(t, root, validRuntimePackRecord(now.Add(time.Minute), func(record *RuntimePackRecord) {
		record.PackID = "pack-base"
	}))
	mustStoreRuntimePack(t, root, validRuntimePackRecord(now.Add(2*time.Minute), func(record *RuntimePackRecord) {
		record.PackID = "pack-candidate-parent-mismatch"
		record.ParentPackID = "pack-other"
		record.RollbackTargetPackID = "pack-base"
	}))
	mustStoreRuntimePack(t, root, validRuntimePackRecord(now.Add(3*time.Minute), func(record *RuntimePackRecord) {
		record.PackID = "pack-candidate-linked"
		record.ParentPackID = "pack-base"
		record.RollbackTargetPackID = "pack-base"
	}))

	err = StoreImprovementCandidateRecord(root, validImprovementCandidateRecord(now.Add(4*time.Minute), func(record *ImprovementCandidateRecord) {
		record.CandidateID = "candidate-parent-mismatch"
		record.BaselinePackID = "pack-base"
		record.CandidatePackID = "pack-candidate-parent-mismatch"
	}))
	if err == nil {
		t.Fatal("StoreImprovementCandidateRecord() error = nil, want parent linkage rejection")
	}
	if !strings.Contains(err.Error(), `parent_pack_id "pack-other" does not match baseline_pack_id "pack-base"`) {
		t.Fatalf("StoreImprovementCandidateRecord() error = %q, want parent linkage rejection", err.Error())
	}

	err = StoreImprovementCandidateRecord(root, validImprovementCandidateRecord(now.Add(5*time.Minute), func(record *ImprovementCandidateRecord) {
		record.CandidateID = "candidate-missing-gate"
		record.BaselinePackID = "pack-base"
		record.CandidatePackID = "pack-candidate-linked"
		record.HotUpdateID = "missing-gate"
	}))
	if err == nil {
		t.Fatal("StoreImprovementCandidateRecord() error = nil, want missing gate rejection")
	}
	if !strings.Contains(err.Error(), ErrHotUpdateGateRecordNotFound.Error()) {
		t.Fatalf("StoreImprovementCandidateRecord() error = %q, want missing gate rejection", err.Error())
	}

	if err := StoreHotUpdateGateRecord(root, validHotUpdateGateRecord(now.Add(6*time.Minute), func(record *HotUpdateGateRecord) {
		record.HotUpdateID = "hot-update-mismatch"
		record.CandidatePackID = "pack-candidate-linked"
		record.PreviousActivePackID = "pack-other-base"
		record.RollbackTargetPackID = "pack-base"
	})); err == nil {
		t.Fatal("StoreHotUpdateGateRecord() error = nil, want missing previous-active rejection")
	}

	mustStoreRuntimePack(t, root, validRuntimePackRecord(now.Add(7*time.Minute), func(record *RuntimePackRecord) {
		record.PackID = "pack-other-base"
	}))
	if err := StoreHotUpdateGateRecord(root, validHotUpdateGateRecord(now.Add(8*time.Minute), func(record *HotUpdateGateRecord) {
		record.HotUpdateID = "hot-update-mismatch"
		record.CandidatePackID = "pack-candidate-linked"
		record.PreviousActivePackID = "pack-other-base"
		record.RollbackTargetPackID = "pack-base"
	})); err != nil {
		t.Fatalf("StoreHotUpdateGateRecord(hot-update-mismatch) error = %v", err)
	}

	err = StoreImprovementCandidateRecord(root, validImprovementCandidateRecord(now.Add(9*time.Minute), func(record *ImprovementCandidateRecord) {
		record.CandidateID = "candidate-gate-mismatch"
		record.BaselinePackID = "pack-base"
		record.CandidatePackID = "pack-candidate-linked"
		record.HotUpdateID = "hot-update-mismatch"
	}))
	if err == nil {
		t.Fatal("StoreImprovementCandidateRecord() error = nil, want gate linkage rejection")
	}
	if !strings.Contains(err.Error(), `previous_active_pack_id "pack-other-base" does not match baseline_pack_id "pack-base"`) {
		t.Fatalf("StoreImprovementCandidateRecord() error = %q, want gate linkage rejection", err.Error())
	}
}

func validImprovementCandidateRecord(now time.Time, mutate func(*ImprovementCandidateRecord)) ImprovementCandidateRecord {
	record := ImprovementCandidateRecord{
		CandidateID:        "candidate-root",
		BaselinePackID:     "pack-base",
		CandidatePackID:    "pack-candidate",
		SourceWorkspaceRef: "workspace/runs/root",
		SourceSummary:      "seeded candidate",
		ValidationBasisRefs: []string{
			"eval/baseline",
			"eval/train",
		},
		HotUpdateID: "",
		CreatedAt:   now,
		CreatedBy:   "system",
	}
	if mutate != nil {
		mutate(&record)
	}
	return record
}
