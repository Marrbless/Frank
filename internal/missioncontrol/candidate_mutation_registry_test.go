package missioncontrol

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestCandidateMutationRecordRoundTripListReplayAndDivergentDuplicate(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 5, 8, 10, 0, 0, 0, time.FixedZone("offset", -4*60*60))
	storeCandidateMutationFixtures(t, root, now)

	second := validCandidateMutationRecord(now.Add(10*time.Minute), func(record *CandidateMutationRecord) {
		record.MutationID = "mutation-b"
		record.MutationSummary = "second bounded mutation"
	})
	if _, changed, err := StoreCandidateMutationRecord(root, second); err != nil {
		t.Fatalf("StoreCandidateMutationRecord(mutation-b) error = %v", err)
	} else if !changed {
		t.Fatal("StoreCandidateMutationRecord(mutation-b) changed = false, want true")
	}

	want := validCandidateMutationRecord(now.Add(11*time.Minute), func(record *CandidateMutationRecord) {
		record.MutationID = " mutation-a "
		record.RunID = " run-1 "
		record.CandidateID = " candidate-1 "
		record.EvalSuiteID = " eval-suite-1 "
		record.BaselinePackID = " pack-base "
		record.CandidatePackID = " pack-candidate "
		record.BaselineResultRef = " eval/baseline "
		record.MutationSummary = " prompt pack mutation "
		record.SourceWorkspaceRef = " workspace/runs/run-1 "
		record.CreatedBy = " operator "
	})
	got, changed, err := StoreCandidateMutationRecord(root, want)
	if err != nil {
		t.Fatalf("StoreCandidateMutationRecord(mutation-a) error = %v", err)
	}
	if !changed {
		t.Fatal("StoreCandidateMutationRecord(mutation-a) changed = false, want true")
	}

	want.RecordVersion = StoreRecordVersion
	want = NormalizeCandidateMutationRecord(want)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("StoreCandidateMutationRecord() = %#v, want %#v", got, want)
	}

	loaded, err := LoadCandidateMutationRecord(root, "mutation-a")
	if err != nil {
		t.Fatalf("LoadCandidateMutationRecord() error = %v", err)
	}
	if !reflect.DeepEqual(loaded, want) {
		t.Fatalf("LoadCandidateMutationRecord() = %#v, want %#v", loaded, want)
	}

	records, err := ListCandidateMutationRecords(root)
	if err != nil {
		t.Fatalf("ListCandidateMutationRecords() error = %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("ListCandidateMutationRecords() len = %d, want 2", len(records))
	}
	if records[0].MutationID != "mutation-a" || records[1].MutationID != "mutation-b" {
		t.Fatalf("ListCandidateMutationRecords() ids = [%q %q], want [mutation-a mutation-b]", records[0].MutationID, records[1].MutationID)
	}

	replayed, changed, err := StoreCandidateMutationRecord(root, want)
	if err != nil {
		t.Fatalf("StoreCandidateMutationRecord(replay) error = %v", err)
	}
	if changed {
		t.Fatal("StoreCandidateMutationRecord(replay) changed = true, want false")
	}
	if !reflect.DeepEqual(replayed, want) {
		t.Fatalf("StoreCandidateMutationRecord(replay) = %#v, want %#v", replayed, want)
	}

	divergent := want
	divergent.MutationSummary = "different mutation"
	if _, _, err := StoreCandidateMutationRecord(root, divergent); err == nil {
		t.Fatal("StoreCandidateMutationRecord(divergent) error = nil, want duplicate rejection")
	} else if !strings.Contains(err.Error(), `mission store candidate mutation "mutation-a" already exists`) {
		t.Fatalf("StoreCandidateMutationRecord(divergent) error = %q, want duplicate context", err.Error())
	}
}

func TestCandidateMutationRecordValidationFailsClosed(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 5, 8, 11, 0, 0, 0, time.UTC)
	storeCandidateMutationFixtures(t, root, now)

	tests := []struct {
		name string
		run  func() error
		want string
	}{
		{
			name: "missing mutation id",
			run: func() error {
				_, _, err := StoreCandidateMutationRecord(root, validCandidateMutationRecord(now.Add(10*time.Minute), func(record *CandidateMutationRecord) {
					record.MutationID = " "
				}))
				return err
			},
			want: "candidate mutation ref mutation_id is required",
		},
		{
			name: "missing baseline result ref",
			run: func() error {
				_, _, err := StoreCandidateMutationRecord(root, validCandidateMutationRecord(now.Add(10*time.Minute), func(record *CandidateMutationRecord) {
					record.BaselineResultRef = " "
				}))
				return err
			},
			want: "mission store candidate mutation baseline_result_ref is required",
		},
		{
			name: "mutation started before baseline",
			run: func() error {
				_, _, err := StoreCandidateMutationRecord(root, validCandidateMutationRecord(now.Add(10*time.Minute), func(record *CandidateMutationRecord) {
					record.MutationStartedAt = record.BaselineCapturedAt.Add(-time.Minute)
				}))
				return err
			},
			want: "mission store candidate mutation mutation_started_at must not precede baseline_captured_at",
		},
		{
			name: "mutation completed before start",
			run: func() error {
				_, _, err := StoreCandidateMutationRecord(root, validCandidateMutationRecord(now.Add(10*time.Minute), func(record *CandidateMutationRecord) {
					record.MutationCompletedAt = record.MutationStartedAt.Add(-time.Minute)
				}))
				return err
			},
			want: "mission store candidate mutation mutation_completed_at must not precede mutation_started_at",
		},
		{
			name: "missing summary",
			run: func() error {
				_, _, err := StoreCandidateMutationRecord(root, validCandidateMutationRecord(now.Add(10*time.Minute), func(record *CandidateMutationRecord) {
					record.MutationSummary = " "
				}))
				return err
			},
			want: "mission store candidate mutation mutation_summary is required",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := tc.run()
			if err == nil {
				t.Fatal("StoreCandidateMutationRecord() error = nil, want fail-closed rejection")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("StoreCandidateMutationRecord() error = %q, want substring %q", err.Error(), tc.want)
			}
		})
	}
}

func TestCandidateMutationRecordRejectsLinkageGaps(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		mutate func(*CandidateMutationRecord)
		want   string
	}{
		{
			name: "baseline ref not recorded on candidate",
			mutate: func(record *CandidateMutationRecord) {
				record.BaselineResultRef = "eval/missing-baseline"
			},
			want: `baseline_result_ref "eval/missing-baseline" is not present in candidate validation_basis_refs`,
		},
		{
			name: "run candidate mismatch",
			mutate: func(record *CandidateMutationRecord) {
				record.CandidateID = "candidate-other"
			},
			want: ErrImprovementCandidateRecordNotFound.Error(),
		},
		{
			name: "run pack mismatch",
			mutate: func(record *CandidateMutationRecord) {
				record.CandidatePackID = "pack-base"
			},
			want: `candidate_pack_id "pack-base" does not match run candidate_pack_id "pack-candidate"`,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			now := time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC)
			storeCandidateMutationFixtures(t, root, now)
			_, _, err := StoreCandidateMutationRecord(root, validCandidateMutationRecord(now.Add(10*time.Minute), tc.mutate))
			if err == nil {
				t.Fatal("StoreCandidateMutationRecord() error = nil, want linkage rejection")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("StoreCandidateMutationRecord() error = %q, want substring %q", err.Error(), tc.want)
			}
		})
	}
}

func TestLoadCandidateMutationRecordNotFound(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if _, err := LoadCandidateMutationRecord(root, "missing-mutation"); !errors.Is(err, ErrCandidateMutationRecordNotFound) {
		t.Fatalf("LoadCandidateMutationRecord() error = %v, want %v", err, ErrCandidateMutationRecordNotFound)
	}
}

func storeCandidateMutationFixtures(t *testing.T, root string, now time.Time) {
	t.Helper()

	storeImprovementRunFixtures(t, root, now)
	if err := StoreImprovementRunRecord(root, validImprovementRunRecord(now.Add(5*time.Minute), func(record *ImprovementRunRecord) {
		record.RunID = "run-1"
		record.CandidateID = "candidate-1"
		record.EvalSuiteID = "eval-suite-1"
		record.BaselinePackID = "pack-base"
		record.CandidatePackID = "pack-candidate"
		record.HotUpdateID = "hot-update-1"
		record.CreatedBy = "operator"
	})); err != nil {
		t.Fatalf("StoreImprovementRunRecord() error = %v", err)
	}
}

func validCandidateMutationRecord(now time.Time, mutate func(*CandidateMutationRecord)) CandidateMutationRecord {
	record := CandidateMutationRecord{
		MutationID:          "mutation-root",
		RunID:               "run-1",
		CandidateID:         "candidate-1",
		EvalSuiteID:         "eval-suite-1",
		BaselinePackID:      "pack-base",
		CandidatePackID:     "pack-candidate",
		BaselineResultRef:   "eval/baseline",
		BaselineCapturedAt:  now,
		MutationStartedAt:   now.Add(time.Minute),
		MutationCompletedAt: now.Add(2 * time.Minute),
		MutationSummary:     "bounded prompt pack mutation",
		SourceWorkspaceRef:  "workspace/runs/run-1",
		CreatedAt:           now.Add(3 * time.Minute),
		CreatedBy:           "system",
	}
	if mutate != nil {
		mutate(&record)
	}
	return record
}
