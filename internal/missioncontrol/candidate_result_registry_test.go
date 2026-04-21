package missioncontrol

import (
	"errors"
	"math"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestCandidateResultRecordRoundTripAndList(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 24, 10, 0, 0, 0, time.FixedZone("offset", -4*60*60))

	storeImprovementRunFixtures(t, root, now)
	if err := StoreImprovementRunRecord(root, validImprovementRunRecord(now.Add(5*time.Minute), func(record *ImprovementRunRecord) {
		record.RunID = "run-result"
		record.CreatedBy = "operator"
	})); err != nil {
		t.Fatalf("StoreImprovementRunRecord() error = %v", err)
	}

	second := validCandidateResultRecord(now.Add(6*time.Minute), func(record *CandidateResultRecord) {
		record.ResultID = "result-b"
		record.RunID = "run-result"
		record.HotUpdateID = ""
		record.BaselineScore = 0.42
		record.TrainScore = 0.63
		record.HoldoutScore = 0.61
		record.Decision = ImprovementRunDecisionDiscard
		record.Notes = "holdout below threshold"
		record.RegressionFlags = []string{"latency_regression"}
		record.CreatedBy = "system"
	})
	if err := StoreCandidateResultRecord(root, second); err != nil {
		t.Fatalf("StoreCandidateResultRecord(result-b) error = %v", err)
	}

	want := validCandidateResultRecord(now.Add(7*time.Minute), func(record *CandidateResultRecord) {
		record.ResultID = " result-a "
		record.RunID = " run-result "
		record.CandidateID = " candidate-1 "
		record.EvalSuiteID = " eval-suite-1 "
		record.BaselinePackID = " pack-base "
		record.CandidatePackID = " pack-candidate "
		record.HotUpdateID = " hot-update-1 "
		record.RegressionFlags = []string{" holdout_warning ", " canary_needed "}
		record.Notes = " keep for next gate "
		record.CreatedBy = " operator "
	})
	if err := StoreCandidateResultRecord(root, want); err != nil {
		t.Fatalf("StoreCandidateResultRecord(result-a) error = %v", err)
	}

	got, err := LoadCandidateResultRecord(root, "result-a")
	if err != nil {
		t.Fatalf("LoadCandidateResultRecord() error = %v", err)
	}

	want.RecordVersion = StoreRecordVersion
	want.ResultID = "result-a"
	want.RunID = "run-result"
	want.CandidateID = "candidate-1"
	want.EvalSuiteID = "eval-suite-1"
	want.BaselinePackID = "pack-base"
	want.CandidatePackID = "pack-candidate"
	want.HotUpdateID = "hot-update-1"
	want.RegressionFlags = []string{"holdout_warning", "canary_needed"}
	want.Notes = "keep for next gate"
	want.CreatedAt = want.CreatedAt.UTC()
	want.CreatedBy = "operator"
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("LoadCandidateResultRecord() = %#v, want %#v", got, want)
	}

	records, err := ListCandidateResultRecords(root)
	if err != nil {
		t.Fatalf("ListCandidateResultRecords() error = %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("ListCandidateResultRecords() len = %d, want 2", len(records))
	}
	if records[0].ResultID != "result-a" || records[1].ResultID != "result-b" {
		t.Fatalf("ListCandidateResultRecords() ids = [%q %q], want [result-a result-b]", records[0].ResultID, records[1].ResultID)
	}
}

func TestCandidateResultReplayIsIdempotentAndAppendOnly(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 24, 11, 0, 0, 0, time.UTC)

	storeImprovementRunFixtures(t, root, now)
	if err := StoreImprovementRunRecord(root, validImprovementRunRecord(now.Add(5*time.Minute), func(record *ImprovementRunRecord) {
		record.RunID = "run-result"
	})); err != nil {
		t.Fatalf("StoreImprovementRunRecord() error = %v", err)
	}

	record := validCandidateResultRecord(now.Add(6*time.Minute), func(record *CandidateResultRecord) {
		record.ResultID = "result-replay"
		record.RunID = "run-result"
		record.HotUpdateID = ""
		record.Decision = ImprovementRunDecisionKeep
		record.RegressionFlags = nil
		record.Notes = "exact replay"
	})
	if err := StoreCandidateResultRecord(root, record); err != nil {
		t.Fatalf("StoreCandidateResultRecord(first) error = %v", err)
	}
	firstBytes, err := os.ReadFile(StoreCandidateResultPath(root, record.ResultID))
	if err != nil {
		t.Fatalf("ReadFile(first) error = %v", err)
	}

	if err := StoreCandidateResultRecord(root, record); err != nil {
		t.Fatalf("StoreCandidateResultRecord(replay) error = %v", err)
	}
	secondBytes, err := os.ReadFile(StoreCandidateResultPath(root, record.ResultID))
	if err != nil {
		t.Fatalf("ReadFile(second) error = %v", err)
	}
	if string(firstBytes) != string(secondBytes) {
		t.Fatalf("candidate result file changed on idempotent replay\nfirst:\n%s\nsecond:\n%s", string(firstBytes), string(secondBytes))
	}

	err = StoreCandidateResultRecord(root, validCandidateResultRecord(now.Add(7*time.Minute), func(changed *CandidateResultRecord) {
		changed.ResultID = "result-replay"
		changed.RunID = "run-result"
		changed.Notes = "divergent replay"
	}))
	if err == nil {
		t.Fatal("StoreCandidateResultRecord() error = nil, want append-only duplicate rejection")
	}
	if !strings.Contains(err.Error(), `mission store candidate result "result-replay" already exists`) {
		t.Fatalf("StoreCandidateResultRecord() error = %q, want append-only duplicate rejection", err.Error())
	}
}

func TestCandidateResultValidationFailsClosed(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 24, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name string
		run  func() error
		want string
	}{
		{
			name: "missing result id",
			run: func() error {
				return StoreCandidateResultRecord(root, validCandidateResultRecord(now, func(record *CandidateResultRecord) {
					record.ResultID = " "
				}))
			},
			want: "candidate result ref result_id is required",
		},
		{
			name: "missing decision",
			run: func() error {
				return StoreCandidateResultRecord(root, validCandidateResultRecord(now, func(record *CandidateResultRecord) {
					record.Decision = ""
				}))
			},
			want: "mission store candidate result decision is required",
		},
		{
			name: "invalid decision",
			run: func() error {
				return StoreCandidateResultRecord(root, validCandidateResultRecord(now, func(record *CandidateResultRecord) {
					record.Decision = ImprovementRunDecision("bad_decision")
				}))
			},
			want: `mission store candidate result decision "bad_decision" is invalid`,
		},
		{
			name: "non-finite score",
			run: func() error {
				return StoreCandidateResultRecord(root, validCandidateResultRecord(now, func(record *CandidateResultRecord) {
					record.HoldoutScore = math.Inf(1)
				}))
			},
			want: "mission store candidate result holdout_score must be finite",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := tc.run()
			if err == nil {
				t.Fatal("StoreCandidateResultRecord() error = nil, want fail-closed rejection")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("StoreCandidateResultRecord() error = %q, want substring %q", err.Error(), tc.want)
			}
		})
	}
}

func TestCandidateResultRejectsMissingOrMismatchedLinkedRefs(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 24, 13, 0, 0, 0, time.UTC)

	err := StoreCandidateResultRecord(root, validCandidateResultRecord(now, func(record *CandidateResultRecord) {
		record.RunID = "missing-run"
		record.CandidateID = "missing-candidate"
		record.EvalSuiteID = "missing-eval"
		record.BaselinePackID = "missing-base"
		record.CandidatePackID = "missing-pack"
		record.HotUpdateID = "missing-gate"
	}))
	if err == nil {
		t.Fatal("StoreCandidateResultRecord() error = nil, want missing ref rejection")
	}
	if !strings.Contains(err.Error(), ErrImprovementRunRecordNotFound.Error()) {
		t.Fatalf("StoreCandidateResultRecord() error = %q, want missing run rejection", err.Error())
	}

	storeImprovementRunFixtures(t, root, now.Add(time.Minute))
	if err := StoreImprovementRunRecord(root, validImprovementRunRecord(now.Add(6*time.Minute), func(record *ImprovementRunRecord) {
		record.RunID = "run-result"
		record.HotUpdateID = "hot-update-1"
	})); err != nil {
		t.Fatalf("StoreImprovementRunRecord() error = %v", err)
	}

	err = StoreCandidateResultRecord(root, validCandidateResultRecord(now.Add(7*time.Minute), func(record *CandidateResultRecord) {
		record.ResultID = "result-mismatch"
		record.RunID = "run-result"
		record.BaselinePackID = "pack-other"
	}))
	if err == nil {
		t.Fatal("StoreCandidateResultRecord() error = nil, want linkage mismatch rejection")
	}
	if !strings.Contains(err.Error(), `baseline_pack_id "pack-other" does not match run baseline_pack_id "pack-base"`) {
		t.Fatalf("StoreCandidateResultRecord() error = %q, want run linkage mismatch rejection", err.Error())
	}
}

func TestLoadCandidateResultRecordNotFound(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if _, err := LoadCandidateResultRecord(root, "missing-result"); !errors.Is(err, ErrCandidateResultRecordNotFound) {
		t.Fatalf("LoadCandidateResultRecord() error = %v, want %v", err, ErrCandidateResultRecordNotFound)
	}
}

func validCandidateResultRecord(now time.Time, mutate func(*CandidateResultRecord)) CandidateResultRecord {
	record := CandidateResultRecord{
		ResultID:           "result-root",
		RunID:              "run-root",
		CandidateID:        "candidate-1",
		EvalSuiteID:        "eval-suite-1",
		BaselinePackID:     "pack-base",
		CandidatePackID:    "pack-candidate",
		HotUpdateID:        "hot-update-1",
		BaselineScore:      0.52,
		TrainScore:         0.78,
		HoldoutScore:       0.74,
		ComplexityScore:    0.21,
		CompatibilityScore: 0.93,
		ResourceScore:      0.67,
		RegressionFlags:    []string{"none"},
		Decision:           ImprovementRunDecisionKeep,
		Notes:              "candidate recorded for later promotion policy",
		CreatedAt:          now,
		CreatedBy:          "system",
	}
	if mutate != nil {
		mutate(&record)
	}
	return record
}
