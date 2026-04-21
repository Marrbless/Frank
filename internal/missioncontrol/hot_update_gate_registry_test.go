package missioncontrol

import (
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestHotUpdateGateRecordRoundTripAndList(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 22, 10, 0, 0, 0, time.UTC)
	mustStoreRuntimePack(t, root, validRuntimePackRecord(now, func(record *RuntimePackRecord) {
		record.PackID = "pack-prev"
	}))
	mustStoreRuntimePack(t, root, validRuntimePackRecord(now.Add(time.Minute), func(record *RuntimePackRecord) {
		record.PackID = "pack-candidate-b"
		record.RollbackTargetPackID = "pack-prev"
	}))
	mustStoreRuntimePack(t, root, validRuntimePackRecord(now.Add(2*time.Minute), func(record *RuntimePackRecord) {
		record.PackID = "pack-candidate-a"
		record.RollbackTargetPackID = "pack-prev"
	}))

	second := validHotUpdateGateRecord(now.Add(3*time.Minute), func(record *HotUpdateGateRecord) {
		record.HotUpdateID = "hot-update-b"
		record.CandidatePackID = "pack-candidate-b"
		record.Objective = "second gate"
	})
	if err := StoreHotUpdateGateRecord(root, second); err != nil {
		t.Fatalf("StoreHotUpdateGateRecord(hot-update-b) error = %v", err)
	}

	want := validHotUpdateGateRecord(now.Add(4*time.Minute), func(record *HotUpdateGateRecord) {
		record.HotUpdateID = " hot-update-a "
		record.Objective = " refresh skills "
		record.CandidatePackID = " pack-candidate-a "
		record.PreviousActivePackID = " pack-prev "
		record.RollbackTargetPackID = " pack-prev "
		record.TargetSurfaces = []string{" skills ", " prompts "}
		record.SurfaceClasses = []string{" class_1 ", " class_2 "}
		record.CompatibilityContractRef = " compat-v2 "
		record.EvalEvidenceRefs = []string{" eval/train ", " eval/holdout "}
		record.SmokeCheckRefs = []string{" smoke/run-1 "}
		record.CanaryRef = " canary-job-1 "
		record.ApprovalRef = " approval-1 "
		record.BudgetRef = " budget-1 "
		record.FailureReason = " staged "
	})
	if err := StoreHotUpdateGateRecord(root, want); err != nil {
		t.Fatalf("StoreHotUpdateGateRecord(hot-update-a) error = %v", err)
	}

	got, err := LoadHotUpdateGateRecord(root, "hot-update-a")
	if err != nil {
		t.Fatalf("LoadHotUpdateGateRecord() error = %v", err)
	}

	want.RecordVersion = StoreRecordVersion
	want.HotUpdateID = "hot-update-a"
	want.Objective = "refresh skills"
	want.CandidatePackID = "pack-candidate-a"
	want.PreviousActivePackID = "pack-prev"
	want.RollbackTargetPackID = "pack-prev"
	want.TargetSurfaces = []string{"skills", "prompts"}
	want.SurfaceClasses = []string{"class_1", "class_2"}
	want.CompatibilityContractRef = "compat-v2"
	want.EvalEvidenceRefs = []string{"eval/train", "eval/holdout"}
	want.SmokeCheckRefs = []string{"smoke/run-1"}
	want.CanaryRef = "canary-job-1"
	want.ApprovalRef = "approval-1"
	want.BudgetRef = "budget-1"
	want.PreparedAt = want.PreparedAt.UTC()
	want.FailureReason = "staged"
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("LoadHotUpdateGateRecord() = %#v, want %#v", got, want)
	}

	records, err := ListHotUpdateGateRecords(root)
	if err != nil {
		t.Fatalf("ListHotUpdateGateRecords() error = %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("ListHotUpdateGateRecords() len = %d, want 2", len(records))
	}
	if records[0].HotUpdateID != "hot-update-a" || records[1].HotUpdateID != "hot-update-b" {
		t.Fatalf("ListHotUpdateGateRecords() ids = [%q %q], want [hot-update-a hot-update-b]", records[0].HotUpdateID, records[1].HotUpdateID)
	}
}

func TestCandidateRuntimePackPointerRoundTripAndResolve(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 22, 11, 0, 0, 0, time.UTC)
	mustStoreRuntimePack(t, root, validRuntimePackRecord(now, func(record *RuntimePackRecord) {
		record.PackID = "pack-prev"
	}))
	candidate := validRuntimePackRecord(now.Add(time.Minute), func(record *RuntimePackRecord) {
		record.PackID = "pack-candidate"
		record.RollbackTargetPackID = "pack-prev"
	})
	mustStoreRuntimePack(t, root, candidate)

	if err := StoreHotUpdateGateRecord(root, validHotUpdateGateRecord(now.Add(2*time.Minute), func(record *HotUpdateGateRecord) {
		record.HotUpdateID = "hot-update-1"
		record.CandidatePackID = "pack-candidate"
		record.PreviousActivePackID = "pack-prev"
		record.RollbackTargetPackID = "pack-prev"
	})); err != nil {
		t.Fatalf("StoreHotUpdateGateRecord() error = %v", err)
	}

	want := CandidateRuntimePackPointer{
		HotUpdateID:     " hot-update-1 ",
		CandidatePackID: " pack-candidate ",
		UpdatedAt:       now.Add(3 * time.Minute),
		UpdatedBy:       " operator ",
	}
	if err := StoreCandidateRuntimePackPointer(root, want); err != nil {
		t.Fatalf("StoreCandidateRuntimePackPointer() error = %v", err)
	}

	got, err := LoadCandidateRuntimePackPointer(root)
	if err != nil {
		t.Fatalf("LoadCandidateRuntimePackPointer() error = %v", err)
	}

	want.RecordVersion = StoreRecordVersion
	want.HotUpdateID = "hot-update-1"
	want.CandidatePackID = "pack-candidate"
	want.UpdatedAt = want.UpdatedAt.UTC()
	want.UpdatedBy = "operator"
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("LoadCandidateRuntimePackPointer() = %#v, want %#v", got, want)
	}

	resolved, err := ResolveCandidateRuntimePackRecord(root)
	if err != nil {
		t.Fatalf("ResolveCandidateRuntimePackRecord() error = %v", err)
	}
	candidate.RecordVersion = StoreRecordVersion
	candidate = NormalizeRuntimePackRecord(candidate)
	if !reflect.DeepEqual(resolved, candidate) {
		t.Fatalf("ResolveCandidateRuntimePackRecord() = %#v, want %#v", resolved, candidate)
	}
}

func TestHotUpdateGateReplayIsIdempotent(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 22, 12, 0, 0, 0, time.UTC)
	mustStoreRuntimePack(t, root, validRuntimePackRecord(now, func(record *RuntimePackRecord) {
		record.PackID = "pack-prev"
	}))
	mustStoreRuntimePack(t, root, validRuntimePackRecord(now.Add(time.Minute), func(record *RuntimePackRecord) {
		record.PackID = "pack-candidate"
		record.RollbackTargetPackID = "pack-prev"
	}))

	record := validHotUpdateGateRecord(now.Add(2*time.Minute), func(gate *HotUpdateGateRecord) {
		gate.HotUpdateID = "hot-update-replay"
		gate.CandidatePackID = "pack-candidate"
		gate.PreviousActivePackID = "pack-prev"
		gate.RollbackTargetPackID = "pack-prev"
	})
	if err := StoreHotUpdateGateRecord(root, record); err != nil {
		t.Fatalf("StoreHotUpdateGateRecord(first) error = %v", err)
	}
	firstBytes, err := os.ReadFile(StoreHotUpdateGatePath(root, record.HotUpdateID))
	if err != nil {
		t.Fatalf("ReadFile(first) error = %v", err)
	}

	if err := StoreHotUpdateGateRecord(root, record); err != nil {
		t.Fatalf("StoreHotUpdateGateRecord(replay) error = %v", err)
	}
	secondBytes, err := os.ReadFile(StoreHotUpdateGatePath(root, record.HotUpdateID))
	if err != nil {
		t.Fatalf("ReadFile(second) error = %v", err)
	}

	if string(firstBytes) != string(secondBytes) {
		t.Fatalf("hot-update gate file changed on idempotent replay\nfirst:\n%s\nsecond:\n%s", string(firstBytes), string(secondBytes))
	}
}

func TestHotUpdateGateValidationFailsClosed(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 22, 13, 0, 0, 0, time.UTC)
	mustStoreRuntimePack(t, root, validRuntimePackRecord(now, func(record *RuntimePackRecord) {
		record.PackID = "pack-prev"
	}))
	mustStoreRuntimePack(t, root, validRuntimePackRecord(now.Add(time.Minute), func(record *RuntimePackRecord) {
		record.PackID = "pack-candidate"
		record.RollbackTargetPackID = "pack-prev"
	}))

	tests := []struct {
		name string
		run  func() error
		want string
	}{
		{
			name: "missing hot update id",
			run: func() error {
				return StoreHotUpdateGateRecord(root, validHotUpdateGateRecord(now.Add(2*time.Minute), func(record *HotUpdateGateRecord) {
					record.HotUpdateID = " "
					record.CandidatePackID = "pack-candidate"
					record.PreviousActivePackID = "pack-prev"
					record.RollbackTargetPackID = "pack-prev"
				}))
			},
			want: "mission store hot-update gate hot_update_id is required",
		},
		{
			name: "invalid reload mode",
			run: func() error {
				return StoreHotUpdateGateRecord(root, validHotUpdateGateRecord(now.Add(2*time.Minute), func(record *HotUpdateGateRecord) {
					record.CandidatePackID = "pack-candidate"
					record.PreviousActivePackID = "pack-prev"
					record.RollbackTargetPackID = "pack-prev"
					record.ReloadMode = HotUpdateReloadMode("bad_reload")
				}))
			},
			want: `mission store hot-update gate reload_mode "bad_reload" is invalid`,
		},
		{
			name: "invalid state",
			run: func() error {
				return StoreHotUpdateGateRecord(root, validHotUpdateGateRecord(now.Add(2*time.Minute), func(record *HotUpdateGateRecord) {
					record.CandidatePackID = "pack-candidate"
					record.PreviousActivePackID = "pack-prev"
					record.RollbackTargetPackID = "pack-prev"
					record.State = HotUpdateGateState("bad_state")
				}))
			},
			want: `mission store hot-update gate state "bad_state" is invalid`,
		},
		{
			name: "invalid decision",
			run: func() error {
				return StoreHotUpdateGateRecord(root, validHotUpdateGateRecord(now.Add(2*time.Minute), func(record *HotUpdateGateRecord) {
					record.CandidatePackID = "pack-candidate"
					record.PreviousActivePackID = "pack-prev"
					record.RollbackTargetPackID = "pack-prev"
					record.Decision = HotUpdateGateDecision("bad_decision")
				}))
			},
			want: `mission store hot-update gate decision "bad_decision" is invalid`,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := tc.run()
			if err == nil {
				t.Fatal("StoreHotUpdateGateRecord() error = nil, want fail-closed rejection")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("StoreHotUpdateGateRecord() error = %q, want substring %q", err.Error(), tc.want)
			}
		})
	}
}

func TestHotUpdateGateRejectsMissingRefs(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 22, 14, 0, 0, 0, time.UTC)

	err := StoreHotUpdateGateRecord(root, validHotUpdateGateRecord(now, func(record *HotUpdateGateRecord) {
		record.CandidatePackID = "missing-candidate"
		record.PreviousActivePackID = "missing-prev"
		record.RollbackTargetPackID = "missing-rollback"
	}))
	if err == nil {
		t.Fatal("StoreHotUpdateGateRecord() error = nil, want missing pack rejection")
	}
	if !strings.Contains(err.Error(), ErrRuntimePackRecordNotFound.Error()) {
		t.Fatalf("StoreHotUpdateGateRecord() error = %q, want missing pack rejection", err.Error())
	}

	mustStoreRuntimePack(t, root, validRuntimePackRecord(now.Add(time.Minute), func(record *RuntimePackRecord) {
		record.PackID = "pack-prev"
	}))
	mustStoreRuntimePack(t, root, validRuntimePackRecord(now.Add(2*time.Minute), func(record *RuntimePackRecord) {
		record.PackID = "pack-candidate"
		record.RollbackTargetPackID = "pack-prev"
	}))

	err = StoreCandidateRuntimePackPointer(root, CandidateRuntimePackPointer{
		HotUpdateID:     "missing-gate",
		CandidatePackID: "pack-candidate",
		UpdatedAt:       now.Add(3 * time.Minute),
		UpdatedBy:       "system",
	})
	if err == nil {
		t.Fatal("StoreCandidateRuntimePackPointer() error = nil, want missing gate rejection")
	}
	if !strings.Contains(err.Error(), ErrHotUpdateGateRecordNotFound.Error()) {
		t.Fatalf("StoreCandidateRuntimePackPointer() error = %q, want missing gate rejection", err.Error())
	}
}

func TestLoadHotUpdateGateAndCandidatePointerNotFound(t *testing.T) {
	t.Parallel()

	root := t.TempDir()

	if _, err := LoadHotUpdateGateRecord(root, "missing-gate"); !errors.Is(err, ErrHotUpdateGateRecordNotFound) {
		t.Fatalf("LoadHotUpdateGateRecord() error = %v, want %v", err, ErrHotUpdateGateRecordNotFound)
	}
	if _, err := LoadCandidateRuntimePackPointer(root); !errors.Is(err, ErrCandidateRuntimePackPointerNotFound) {
		t.Fatalf("LoadCandidateRuntimePackPointer() error = %v, want %v", err, ErrCandidateRuntimePackPointerNotFound)
	}
}

func validHotUpdateGateRecord(now time.Time, mutate func(*HotUpdateGateRecord)) HotUpdateGateRecord {
	record := HotUpdateGateRecord{
		HotUpdateID:              "hot-update-root",
		Objective:                "stage runtime pack candidate",
		CandidatePackID:          "pack-candidate",
		PreviousActivePackID:     "pack-prev",
		RollbackTargetPackID:     "pack-prev",
		TargetSurfaces:           []string{"skills"},
		SurfaceClasses:           []string{"class_1"},
		ReloadMode:               HotUpdateReloadModeSkillReload,
		CompatibilityContractRef: "compat-v1",
		EvalEvidenceRefs:         []string{"eval/train"},
		SmokeCheckRefs:           []string{"smoke/run-1"},
		CanaryRef:                "",
		ApprovalRef:              "",
		BudgetRef:                "",
		PreparedAt:               now,
		State:                    HotUpdateGateStatePrepared,
		Decision:                 HotUpdateGateDecisionKeepStaged,
		FailureReason:            "",
	}
	if mutate != nil {
		mutate(&record)
	}
	return record
}
