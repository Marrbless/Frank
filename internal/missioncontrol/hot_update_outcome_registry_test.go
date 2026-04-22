package missioncontrol

import (
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestHotUpdateOutcomeRecordRoundTripAndList(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 26, 10, 0, 0, 0, time.FixedZone("offset", -4*60*60))

	storeHotUpdateOutcomeFixtures(t, root, now)

	second := validHotUpdateOutcomeRecord(now.Add(8*time.Minute), func(record *HotUpdateOutcomeRecord) {
		record.OutcomeID = "outcome-b"
		record.CandidateID = ""
		record.RunID = ""
		record.CandidateResultID = ""
		record.CandidatePackID = "pack-candidate"
		record.OutcomeKind = HotUpdateOutcomeKindBlocked
		record.Reason = "policy blocked activation"
		record.Notes = "control-plane block only"
		record.CreatedBy = "system"
	})
	if err := StoreHotUpdateOutcomeRecord(root, second); err != nil {
		t.Fatalf("StoreHotUpdateOutcomeRecord(outcome-b) error = %v", err)
	}

	want := validHotUpdateOutcomeRecord(now.Add(9*time.Minute), func(record *HotUpdateOutcomeRecord) {
		record.OutcomeID = " outcome-a "
		record.HotUpdateID = " hot-update-1 "
		record.CandidateID = " candidate-1 "
		record.RunID = " run-1 "
		record.CandidateResultID = " result-1 "
		record.CandidatePackID = " pack-candidate "
		record.Reason = " operator kept staged "
		record.Notes = " read-only outcome ledger "
		record.CreatedBy = " operator "
	})
	if err := StoreHotUpdateOutcomeRecord(root, want); err != nil {
		t.Fatalf("StoreHotUpdateOutcomeRecord(outcome-a) error = %v", err)
	}

	got, err := LoadHotUpdateOutcomeRecord(root, "outcome-a")
	if err != nil {
		t.Fatalf("LoadHotUpdateOutcomeRecord() error = %v", err)
	}

	want.RecordVersion = StoreRecordVersion
	want.OutcomeID = "outcome-a"
	want.HotUpdateID = "hot-update-1"
	want.CandidateID = "candidate-1"
	want.RunID = "run-1"
	want.CandidateResultID = "result-1"
	want.CandidatePackID = "pack-candidate"
	want.Reason = "operator kept staged"
	want.Notes = "read-only outcome ledger"
	want.OutcomeAt = want.OutcomeAt.UTC()
	want.CreatedAt = want.CreatedAt.UTC()
	want.CreatedBy = "operator"
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("LoadHotUpdateOutcomeRecord() = %#v, want %#v", got, want)
	}

	records, err := ListHotUpdateOutcomeRecords(root)
	if err != nil {
		t.Fatalf("ListHotUpdateOutcomeRecords() error = %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("ListHotUpdateOutcomeRecords() len = %d, want 2", len(records))
	}
	if records[0].OutcomeID != "outcome-a" || records[1].OutcomeID != "outcome-b" {
		t.Fatalf("ListHotUpdateOutcomeRecords() ids = [%q %q], want [outcome-a outcome-b]", records[0].OutcomeID, records[1].OutcomeID)
	}
}

func TestHotUpdateOutcomeReplayIsIdempotentAndAppendOnly(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 26, 11, 0, 0, 0, time.UTC)

	storeHotUpdateOutcomeFixtures(t, root, now)

	record := validHotUpdateOutcomeRecord(now.Add(8*time.Minute), func(record *HotUpdateOutcomeRecord) {
		record.OutcomeID = "outcome-replay"
		record.Reason = "exact replay"
		record.Notes = "same bytes expected"
	})
	if err := StoreHotUpdateOutcomeRecord(root, record); err != nil {
		t.Fatalf("StoreHotUpdateOutcomeRecord(first) error = %v", err)
	}
	firstBytes, err := os.ReadFile(StoreHotUpdateOutcomePath(root, record.OutcomeID))
	if err != nil {
		t.Fatalf("ReadFile(first) error = %v", err)
	}

	if err := StoreHotUpdateOutcomeRecord(root, record); err != nil {
		t.Fatalf("StoreHotUpdateOutcomeRecord(replay) error = %v", err)
	}
	secondBytes, err := os.ReadFile(StoreHotUpdateOutcomePath(root, record.OutcomeID))
	if err != nil {
		t.Fatalf("ReadFile(second) error = %v", err)
	}
	if string(firstBytes) != string(secondBytes) {
		t.Fatalf("hot-update outcome file changed on idempotent replay\nfirst:\n%s\nsecond:\n%s", string(firstBytes), string(secondBytes))
	}

	err = StoreHotUpdateOutcomeRecord(root, validHotUpdateOutcomeRecord(now.Add(9*time.Minute), func(changed *HotUpdateOutcomeRecord) {
		changed.OutcomeID = "outcome-replay"
		changed.Notes = "divergent replay"
	}))
	if err == nil {
		t.Fatal("StoreHotUpdateOutcomeRecord() error = nil, want append-only duplicate rejection")
	}
	if !strings.Contains(err.Error(), `mission store hot-update outcome "outcome-replay" already exists`) {
		t.Fatalf("StoreHotUpdateOutcomeRecord() error = %q, want append-only duplicate rejection", err.Error())
	}
}

func TestHotUpdateOutcomeValidationFailsClosed(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name string
		run  func() error
		want string
	}{
		{
			name: "missing outcome id",
			run: func() error {
				return StoreHotUpdateOutcomeRecord(root, validHotUpdateOutcomeRecord(now, func(record *HotUpdateOutcomeRecord) {
					record.OutcomeID = " "
				}))
			},
			want: "hot-update outcome ref outcome_id is required",
		},
		{
			name: "invalid outcome kind",
			run: func() error {
				return StoreHotUpdateOutcomeRecord(root, validHotUpdateOutcomeRecord(now, func(record *HotUpdateOutcomeRecord) {
					record.OutcomeKind = HotUpdateOutcomeKind("bad_kind")
				}))
			},
			want: `mission store hot-update outcome outcome_kind "bad_kind" is invalid`,
		},
		{
			name: "missing outcome time",
			run: func() error {
				return StoreHotUpdateOutcomeRecord(root, validHotUpdateOutcomeRecord(now, func(record *HotUpdateOutcomeRecord) {
					record.OutcomeAt = time.Time{}
				}))
			},
			want: "mission store hot-update outcome outcome_at is required",
		},
		{
			name: "missing created by",
			run: func() error {
				return StoreHotUpdateOutcomeRecord(root, validHotUpdateOutcomeRecord(now, func(record *HotUpdateOutcomeRecord) {
					record.CreatedBy = " "
				}))
			},
			want: "mission store hot-update outcome created_by is required",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := tc.run()
			if err == nil {
				t.Fatal("StoreHotUpdateOutcomeRecord() error = nil, want fail-closed rejection")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("StoreHotUpdateOutcomeRecord() error = %q, want substring %q", err.Error(), tc.want)
			}
		})
	}
}

func TestHotUpdateOutcomeRejectsMissingAndMismatchedLinkedRefs(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 26, 13, 0, 0, 0, time.UTC)

	err := StoreHotUpdateOutcomeRecord(root, validHotUpdateOutcomeRecord(now, func(record *HotUpdateOutcomeRecord) {
		record.HotUpdateID = "missing-gate"
		record.CandidateID = ""
		record.RunID = ""
		record.CandidateResultID = ""
		record.CandidatePackID = ""
	}))
	if err == nil {
		t.Fatal("StoreHotUpdateOutcomeRecord() error = nil, want missing gate rejection")
	}
	if !strings.Contains(err.Error(), ErrHotUpdateGateRecordNotFound.Error()) {
		t.Fatalf("StoreHotUpdateOutcomeRecord() error = %q, want missing gate rejection", err.Error())
	}

	storeHotUpdateOutcomeFixtures(t, root, now.Add(time.Minute))

	err = StoreHotUpdateOutcomeRecord(root, validHotUpdateOutcomeRecord(now.Add(9*time.Minute), func(record *HotUpdateOutcomeRecord) {
		record.OutcomeID = "outcome-mismatch"
		record.CandidatePackID = "pack-other"
	}))
	if err == nil {
		t.Fatal("StoreHotUpdateOutcomeRecord() error = nil, want linkage mismatch rejection")
	}
	if !strings.Contains(err.Error(), `candidate_pack_id "pack-other" does not match hot-update gate candidate_pack_id "pack-candidate"`) {
		t.Fatalf("StoreHotUpdateOutcomeRecord() error = %q, want candidate_pack_id linkage mismatch rejection", err.Error())
	}
}

func TestLoadHotUpdateOutcomeRecordNotFound(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if _, err := LoadHotUpdateOutcomeRecord(root, "missing-outcome"); !errors.Is(err, ErrHotUpdateOutcomeRecordNotFound) {
		t.Fatalf("LoadHotUpdateOutcomeRecord() error = %v, want %v", err, ErrHotUpdateOutcomeRecordNotFound)
	}
}

func validHotUpdateOutcomeRecord(now time.Time, mutate func(*HotUpdateOutcomeRecord)) HotUpdateOutcomeRecord {
	record := HotUpdateOutcomeRecord{
		OutcomeID:         "outcome-root",
		HotUpdateID:       "hot-update-1",
		CandidateID:       "candidate-1",
		RunID:             "run-1",
		CandidateResultID: "result-1",
		CandidatePackID:   "pack-candidate",
		OutcomeKind:       HotUpdateOutcomeKindKeptStaged,
		Reason:            "candidate kept staged",
		Notes:             "recorded for later operator control",
		OutcomeAt:         now,
		CreatedAt:         now.Add(time.Minute),
		CreatedBy:         "system",
	}
	if mutate != nil {
		mutate(&record)
	}
	return record
}

func storeHotUpdateOutcomeFixtures(t *testing.T, root string, now time.Time) {
	t.Helper()

	storeImprovementRunFixtures(t, root, now)
	if err := StoreImprovementRunRecord(root, validImprovementRunRecord(now.Add(5*time.Minute), func(record *ImprovementRunRecord) {
		record.RunID = "run-1"
		record.HotUpdateID = "hot-update-1"
		record.CreatedBy = "operator"
	})); err != nil {
		t.Fatalf("StoreImprovementRunRecord() error = %v", err)
	}
	if err := StoreCandidateResultRecord(root, validCandidateResultRecord(now.Add(6*time.Minute), func(record *CandidateResultRecord) {
		record.ResultID = "result-1"
		record.RunID = "run-1"
		record.HotUpdateID = "hot-update-1"
		record.CreatedBy = "operator"
	})); err != nil {
		t.Fatalf("StoreCandidateResultRecord() error = %v", err)
	}
	mustStoreRuntimePack(t, root, validRuntimePackRecord(now.Add(7*time.Minute), func(record *RuntimePackRecord) {
		record.PackID = "pack-other"
		record.ParentPackID = "pack-base"
		record.RollbackTargetPackID = "pack-base"
	}))
}
