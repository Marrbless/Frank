package missioncontrol

import (
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestEvalSuiteRecordRoundTripAndList(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 23, 10, 0, 0, 0, time.FixedZone("offset", -4*60*60))

	mustStoreRuntimePack(t, root, validRuntimePackRecord(now, func(record *RuntimePackRecord) {
		record.PackID = "pack-base"
		record.SourceSummary = "baseline pack"
	}))
	mustStoreRuntimePack(t, root, validRuntimePackRecord(now.Add(time.Minute), func(record *RuntimePackRecord) {
		record.PackID = "pack-candidate"
		record.ParentPackID = "pack-base"
		record.RollbackTargetPackID = "pack-base"
		record.SourceSummary = "candidate pack"
	}))
	if err := StoreImprovementCandidateRecord(root, validImprovementCandidateRecord(now.Add(2*time.Minute), func(record *ImprovementCandidateRecord) {
		record.CandidateID = "candidate-1"
		record.BaselinePackID = "pack-base"
		record.CandidatePackID = "pack-candidate"
		record.SourceSummary = "candidate linkage"
		record.SourceWorkspaceRef = ""
	})); err != nil {
		t.Fatalf("StoreImprovementCandidateRecord() error = %v", err)
	}

	second := validEvalSuiteRecord(now.Add(3*time.Minute), func(record *EvalSuiteRecord) {
		record.EvalSuiteID = "eval-suite-b"
		record.RubricRef = "rubric/b"
		record.TrainCorpusRef = "corpus/train-b"
		record.HoldoutCorpusRef = "corpus/holdout-b"
		record.EvaluatorRef = "evaluator/b"
		record.NegativeCaseCount = 2
		record.BoundaryCaseCount = 1
		record.CandidateID = ""
		record.BaselinePackID = ""
		record.CandidatePackID = ""
		record.CreatedBy = "system"
	})
	if err := StoreEvalSuiteRecord(root, second); err != nil {
		t.Fatalf("StoreEvalSuiteRecord(eval-suite-b) error = %v", err)
	}

	want := validEvalSuiteRecord(now.Add(4*time.Minute), func(record *EvalSuiteRecord) {
		record.EvalSuiteID = " eval-suite-a "
		record.RubricRef = " rubric/a "
		record.TrainCorpusRef = " corpus/train-a "
		record.HoldoutCorpusRef = " corpus/holdout-a "
		record.EvaluatorRef = " evaluator/a "
		record.NegativeCaseCount = 5
		record.BoundaryCaseCount = 3
		record.CandidateID = " candidate-1 "
		record.BaselinePackID = " pack-base "
		record.CandidatePackID = " pack-candidate "
		record.CreatedBy = " operator "
	})
	if err := StoreEvalSuiteRecord(root, want); err != nil {
		t.Fatalf("StoreEvalSuiteRecord(eval-suite-a) error = %v", err)
	}

	got, err := LoadEvalSuiteRecord(root, "eval-suite-a")
	if err != nil {
		t.Fatalf("LoadEvalSuiteRecord() error = %v", err)
	}

	want.RecordVersion = StoreRecordVersion
	want.EvalSuiteID = "eval-suite-a"
	want.RubricRef = "rubric/a"
	want.TrainCorpusRef = "corpus/train-a"
	want.HoldoutCorpusRef = "corpus/holdout-a"
	want.EvaluatorRef = "evaluator/a"
	want.CandidateID = "candidate-1"
	want.BaselinePackID = "pack-base"
	want.CandidatePackID = "pack-candidate"
	want.CreatedAt = want.CreatedAt.UTC()
	want.CreatedBy = "operator"
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("LoadEvalSuiteRecord() = %#v, want %#v", got, want)
	}

	records, err := ListEvalSuiteRecords(root)
	if err != nil {
		t.Fatalf("ListEvalSuiteRecords() error = %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("ListEvalSuiteRecords() len = %d, want 2", len(records))
	}
	if records[0].EvalSuiteID != "eval-suite-a" || records[1].EvalSuiteID != "eval-suite-b" {
		t.Fatalf("ListEvalSuiteRecords() ids = [%q %q], want [eval-suite-a eval-suite-b]", records[0].EvalSuiteID, records[1].EvalSuiteID)
	}
}

func TestEvalSuiteReplayIsIdempotent(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 23, 11, 0, 0, 0, time.UTC)

	record := validEvalSuiteRecord(now, func(record *EvalSuiteRecord) {
		record.EvalSuiteID = "eval-suite-replay"
		record.CandidateID = ""
		record.BaselinePackID = ""
		record.CandidatePackID = ""
	})
	if err := StoreEvalSuiteRecord(root, record); err != nil {
		t.Fatalf("StoreEvalSuiteRecord(first) error = %v", err)
	}
	firstBytes, err := os.ReadFile(StoreEvalSuitePath(root, record.EvalSuiteID))
	if err != nil {
		t.Fatalf("ReadFile(first) error = %v", err)
	}

	if err := StoreEvalSuiteRecord(root, record); err != nil {
		t.Fatalf("StoreEvalSuiteRecord(replay) error = %v", err)
	}
	secondBytes, err := os.ReadFile(StoreEvalSuitePath(root, record.EvalSuiteID))
	if err != nil {
		t.Fatalf("ReadFile(second) error = %v", err)
	}

	if string(firstBytes) != string(secondBytes) {
		t.Fatalf("eval-suite file changed on idempotent replay\nfirst:\n%s\nsecond:\n%s", string(firstBytes), string(secondBytes))
	}
}

func TestEvalSuiteDivergentDuplicateFailsClosed(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 23, 11, 30, 0, 0, time.UTC)
	record := validEvalSuiteRecord(now, func(record *EvalSuiteRecord) {
		record.EvalSuiteID = "eval-suite-immutable"
		record.CandidateID = ""
		record.BaselinePackID = ""
		record.CandidatePackID = ""
	})

	if err := StoreEvalSuiteRecord(root, record); err != nil {
		t.Fatalf("StoreEvalSuiteRecord(first) error = %v", err)
	}

	err := StoreEvalSuiteRecord(root, validEvalSuiteRecord(now, func(record *EvalSuiteRecord) {
		record.EvalSuiteID = "eval-suite-immutable"
		record.HoldoutCorpusRef = "corpus/holdout-mutated"
		record.CandidateID = ""
		record.BaselinePackID = ""
		record.CandidatePackID = ""
	}))
	if err == nil {
		t.Fatal("StoreEvalSuiteRecord(divergent) error = nil, want duplicate rejection")
	}
	if !strings.Contains(err.Error(), `mission store eval-suite "eval-suite-immutable" already exists`) {
		t.Fatalf("StoreEvalSuiteRecord(divergent) error = %q, want duplicate context", err.Error())
	}

	got, err := LoadEvalSuiteRecord(root, "eval-suite-immutable")
	if err != nil {
		t.Fatalf("LoadEvalSuiteRecord() error = %v", err)
	}
	if got.HoldoutCorpusRef != "corpus/holdout-root" {
		t.Fatalf("HoldoutCorpusRef = %q, want original corpus/holdout-root", got.HoldoutCorpusRef)
	}
}

func TestEvalSuiteValidationFailsClosed(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 23, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name string
		run  func() error
		want string
	}{
		{
			name: "missing eval suite id",
			run: func() error {
				return StoreEvalSuiteRecord(root, validEvalSuiteRecord(now, func(record *EvalSuiteRecord) {
					record.EvalSuiteID = " "
				}))
			},
			want: "eval-suite ref eval_suite_id is required",
		},
		{
			name: "missing rubric ref",
			run: func() error {
				return StoreEvalSuiteRecord(root, validEvalSuiteRecord(now, func(record *EvalSuiteRecord) {
					record.RubricRef = " "
				}))
			},
			want: "mission store eval-suite rubric_ref is required",
		},
		{
			name: "negative case count",
			run: func() error {
				return StoreEvalSuiteRecord(root, validEvalSuiteRecord(now, func(record *EvalSuiteRecord) {
					record.NegativeCaseCount = -1
				}))
			},
			want: "mission store eval-suite negative_case_count must be non-negative",
		},
		{
			name: "not frozen",
			run: func() error {
				return StoreEvalSuiteRecord(root, validEvalSuiteRecord(now, func(record *EvalSuiteRecord) {
					record.FrozenForRun = false
				}))
			},
			want: "mission store eval-suite frozen_for_run must be true",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := tc.run()
			if err == nil {
				t.Fatal("StoreEvalSuiteRecord() error = nil, want fail-closed rejection")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("StoreEvalSuiteRecord() error = %q, want substring %q", err.Error(), tc.want)
			}
		})
	}
}

func TestEvalSuiteStoreDoesNotMutateRuntimeSurfaces(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 23, 13, 30, 0, 0, time.UTC)
	if err := StoreEvalSuiteRecord(root, validEvalSuiteRecord(now, func(record *EvalSuiteRecord) {
		record.CandidateID = ""
		record.BaselinePackID = ""
		record.CandidatePackID = ""
	})); err != nil {
		t.Fatalf("StoreEvalSuiteRecord() error = %v", err)
	}

	absentPaths := []string{
		StoreRuntimePacksDir(root),
		StoreImprovementCandidatesDir(root),
		StoreImprovementRunsDir(root),
		StoreCandidateResultsDir(root),
		StoreHotUpdateGatesDir(root),
		StoreHotUpdateOutcomesDir(root),
		StorePromotionsDir(root),
		StoreRollbacksDir(root),
		StoreRollbackAppliesDir(root),
		StoreActiveRuntimePackPointerPath(root),
		StoreLastKnownGoodRuntimePackPointerPath(root),
	}
	for _, path := range absentPaths {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("path %s exists or errored after eval-suite store: %v", path, err)
		}
	}
}

func TestEvalSuiteRejectsMissingRefs(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 23, 13, 0, 0, 0, time.UTC)

	err := StoreEvalSuiteRecord(root, validEvalSuiteRecord(now, func(record *EvalSuiteRecord) {
		record.CandidateID = "missing-candidate"
		record.BaselinePackID = "missing-base"
		record.CandidatePackID = "missing-pack"
	}))
	if err == nil {
		t.Fatal("StoreEvalSuiteRecord() error = nil, want missing ref rejection")
	}
	if !strings.Contains(err.Error(), ErrImprovementCandidateRecordNotFound.Error()) && !strings.Contains(err.Error(), ErrRuntimePackRecordNotFound.Error()) {
		t.Fatalf("StoreEvalSuiteRecord() error = %q, want missing ref rejection", err.Error())
	}

	mustStoreRuntimePack(t, root, validRuntimePackRecord(now.Add(time.Minute), func(record *RuntimePackRecord) {
		record.PackID = "pack-base"
	}))
	mustStoreRuntimePack(t, root, validRuntimePackRecord(now.Add(2*time.Minute), func(record *RuntimePackRecord) {
		record.PackID = "pack-candidate"
		record.ParentPackID = "pack-base"
		record.RollbackTargetPackID = "pack-base"
	}))
	if err := StoreImprovementCandidateRecord(root, validImprovementCandidateRecord(now.Add(3*time.Minute), func(record *ImprovementCandidateRecord) {
		record.CandidateID = "candidate-1"
		record.BaselinePackID = "pack-base"
		record.CandidatePackID = "pack-candidate"
		record.SourceSummary = "candidate linkage"
		record.SourceWorkspaceRef = ""
	})); err != nil {
		t.Fatalf("StoreImprovementCandidateRecord() error = %v", err)
	}

	err = StoreEvalSuiteRecord(root, validEvalSuiteRecord(now.Add(4*time.Minute), func(record *EvalSuiteRecord) {
		record.CandidateID = "candidate-1"
		record.BaselinePackID = "pack-other"
		record.CandidatePackID = "pack-candidate"
	}))
	if err == nil {
		t.Fatal("StoreEvalSuiteRecord() error = nil, want candidate linkage mismatch rejection")
	}
	if !strings.Contains(err.Error(), `baseline_pack_id "pack-other" does not match candidate baseline_pack_id "pack-base"`) {
		t.Fatalf("StoreEvalSuiteRecord() error = %q, want candidate linkage mismatch rejection", err.Error())
	}
}

func TestLoadEvalSuiteRecordNotFound(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if _, err := LoadEvalSuiteRecord(root, "missing-eval-suite"); !errors.Is(err, ErrEvalSuiteRecordNotFound) {
		t.Fatalf("LoadEvalSuiteRecord() error = %v, want %v", err, ErrEvalSuiteRecordNotFound)
	}
}

func validEvalSuiteRecord(now time.Time, mutate func(*EvalSuiteRecord)) EvalSuiteRecord {
	record := EvalSuiteRecord{
		EvalSuiteID:       "eval-suite-root",
		RubricRef:         "rubric/root",
		TrainCorpusRef:    "corpus/train-root",
		HoldoutCorpusRef:  "corpus/holdout-root",
		EvaluatorRef:      "evaluator/root",
		NegativeCaseCount: 1,
		BoundaryCaseCount: 1,
		FrozenForRun:      true,
		CandidateID:       "",
		BaselinePackID:    "",
		CandidatePackID:   "",
		CreatedAt:         now,
		CreatedBy:         "system",
	}
	if mutate != nil {
		mutate(&record)
	}
	return record
}
