package missioncontrol

import (
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestPromotionRecordRoundTripAndList(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 27, 10, 0, 0, 0, time.FixedZone("offset", -4*60*60))

	storePromotionFixtures(t, root, now)

	second := validPromotionRecord(now.Add(10*time.Minute), func(record *PromotionRecord) {
		record.PromotionID = "promotion-b"
		record.OutcomeID = ""
		record.CandidateID = ""
		record.RunID = ""
		record.CandidateResultID = ""
		record.LastKnownGoodPackID = ""
		record.LastKnownGoodBasis = ""
		record.Reason = "manual promotion checkpoint"
		record.Notes = "operator record only"
		record.CreatedBy = "system"
	})
	if err := StorePromotionRecord(root, second); err != nil {
		t.Fatalf("StorePromotionRecord(promotion-b) error = %v", err)
	}

	want := validPromotionRecord(now.Add(11*time.Minute), func(record *PromotionRecord) {
		record.PromotionID = " promotion-a "
		record.PromotedPackID = " pack-candidate "
		record.PreviousActivePackID = " pack-base "
		record.LastKnownGoodPackID = " pack-base "
		record.LastKnownGoodBasis = " holdout_pass "
		record.HotUpdateID = " hot-update-1 "
		record.OutcomeID = " outcome-1 "
		record.CandidateID = " candidate-1 "
		record.RunID = " run-1 "
		record.CandidateResultID = " result-1 "
		record.Reason = " operator promotion "
		record.Notes = " durable promotion ledger "
		record.CreatedBy = " operator "
	})
	if err := StorePromotionRecord(root, want); err != nil {
		t.Fatalf("StorePromotionRecord(promotion-a) error = %v", err)
	}

	got, err := LoadPromotionRecord(root, "promotion-a")
	if err != nil {
		t.Fatalf("LoadPromotionRecord() error = %v", err)
	}

	want.RecordVersion = StoreRecordVersion
	want.PromotionID = "promotion-a"
	want.PromotedPackID = "pack-candidate"
	want.PreviousActivePackID = "pack-base"
	want.LastKnownGoodPackID = "pack-base"
	want.LastKnownGoodBasis = "holdout_pass"
	want.HotUpdateID = "hot-update-1"
	want.OutcomeID = "outcome-1"
	want.CandidateID = "candidate-1"
	want.RunID = "run-1"
	want.CandidateResultID = "result-1"
	want.Reason = "operator promotion"
	want.Notes = "durable promotion ledger"
	want.PromotedAt = want.PromotedAt.UTC()
	want.CreatedAt = want.CreatedAt.UTC()
	want.CreatedBy = "operator"
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("LoadPromotionRecord() = %#v, want %#v", got, want)
	}

	records, err := ListPromotionRecords(root)
	if err != nil {
		t.Fatalf("ListPromotionRecords() error = %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("ListPromotionRecords() len = %d, want 2", len(records))
	}
	if records[0].PromotionID != "promotion-a" || records[1].PromotionID != "promotion-b" {
		t.Fatalf("ListPromotionRecords() ids = [%q %q], want [promotion-a promotion-b]", records[0].PromotionID, records[1].PromotionID)
	}
}

func TestPromotionReplayIsIdempotentAndImmutable(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 27, 11, 0, 0, 0, time.UTC)

	storePromotionFixtures(t, root, now)

	record := validPromotionRecord(now.Add(10*time.Minute), func(record *PromotionRecord) {
		record.PromotionID = "promotion-replay"
		record.Reason = "exact replay"
		record.Notes = "same bytes expected"
	})
	if err := StorePromotionRecord(root, record); err != nil {
		t.Fatalf("StorePromotionRecord(first) error = %v", err)
	}
	firstBytes, err := os.ReadFile(StorePromotionPath(root, record.PromotionID))
	if err != nil {
		t.Fatalf("ReadFile(first) error = %v", err)
	}

	if err := StorePromotionRecord(root, record); err != nil {
		t.Fatalf("StorePromotionRecord(replay) error = %v", err)
	}
	secondBytes, err := os.ReadFile(StorePromotionPath(root, record.PromotionID))
	if err != nil {
		t.Fatalf("ReadFile(second) error = %v", err)
	}
	if string(firstBytes) != string(secondBytes) {
		t.Fatalf("promotion file changed on idempotent replay\nfirst:\n%s\nsecond:\n%s", string(firstBytes), string(secondBytes))
	}

	err = StorePromotionRecord(root, validPromotionRecord(now.Add(11*time.Minute), func(changed *PromotionRecord) {
		changed.PromotionID = "promotion-replay"
		changed.Notes = "divergent replay"
	}))
	if err == nil {
		t.Fatal("StorePromotionRecord() error = nil, want immutable duplicate rejection")
	}
	if !strings.Contains(err.Error(), `mission store promotion "promotion-replay" already exists`) {
		t.Fatalf("StorePromotionRecord() error = %q, want immutable duplicate rejection", err.Error())
	}
}

func TestPromotionValidationFailsClosed(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 27, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name string
		run  func() error
		want string
	}{
		{
			name: "missing promotion id",
			run: func() error {
				return StorePromotionRecord(root, validPromotionRecord(now, func(record *PromotionRecord) {
					record.PromotionID = " "
				}))
			},
			want: "promotion ref promotion_id is required",
		},
		{
			name: "missing reason",
			run: func() error {
				return StorePromotionRecord(root, validPromotionRecord(now, func(record *PromotionRecord) {
					record.Reason = " "
				}))
			},
			want: "mission store promotion reason is required",
		},
		{
			name: "missing promoted at",
			run: func() error {
				return StorePromotionRecord(root, validPromotionRecord(now, func(record *PromotionRecord) {
					record.PromotedAt = time.Time{}
				}))
			},
			want: "mission store promotion promoted_at is required",
		},
		{
			name: "basis without last known good pack",
			run: func() error {
				return StorePromotionRecord(root, validPromotionRecord(now, func(record *PromotionRecord) {
					record.LastKnownGoodPackID = ""
					record.LastKnownGoodBasis = "holdout_pass"
				}))
			},
			want: "mission store promotion last_known_good_pack_id is required with last_known_good_basis",
		},
		{
			name: "missing created by",
			run: func() error {
				return StorePromotionRecord(root, validPromotionRecord(now, func(record *PromotionRecord) {
					record.CreatedBy = " "
				}))
			},
			want: "mission store promotion created_by is required",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := tc.run()
			if err == nil {
				t.Fatal("StorePromotionRecord() error = nil, want fail-closed rejection")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("StorePromotionRecord() error = %q, want substring %q", err.Error(), tc.want)
			}
		})
	}
}

func TestPromotionRejectsMissingAndMismatchedLinkedRefs(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 27, 13, 0, 0, 0, time.UTC)

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

	err := StorePromotionRecord(root, validPromotionRecord(now, func(record *PromotionRecord) {
		record.HotUpdateID = "missing-gate"
		record.OutcomeID = ""
		record.CandidateID = ""
		record.RunID = ""
		record.CandidateResultID = ""
		record.LastKnownGoodPackID = ""
		record.LastKnownGoodBasis = ""
	}))
	if err == nil {
		t.Fatal("StorePromotionRecord() error = nil, want missing gate rejection")
	}
	if !strings.Contains(err.Error(), ErrHotUpdateGateRecordNotFound.Error()) {
		t.Fatalf("StorePromotionRecord() error = %q, want missing gate rejection", err.Error())
	}

	storePromotionFixtures(t, root, now.Add(time.Minute))

	err = StorePromotionRecord(root, validPromotionRecord(now.Add(10*time.Minute), func(record *PromotionRecord) {
		record.PromotionID = "promotion-mismatch"
		record.PromotedPackID = "pack-other"
	}))
	if err == nil {
		t.Fatal("StorePromotionRecord() error = nil, want promoted pack linkage mismatch rejection")
	}
	if !strings.Contains(err.Error(), `promoted_pack_id "pack-other" does not match hot-update gate candidate_pack_id "pack-candidate"`) {
		t.Fatalf("StorePromotionRecord() error = %q, want promoted pack linkage mismatch rejection", err.Error())
	}

	err = StorePromotionRecord(root, validPromotionRecord(now.Add(11*time.Minute), func(record *PromotionRecord) {
		record.PromotionID = "promotion-missing-outcome"
		record.OutcomeID = "missing-outcome"
	}))
	if err == nil {
		t.Fatal("StorePromotionRecord() error = nil, want missing outcome rejection")
	}
	if !strings.Contains(err.Error(), ErrHotUpdateOutcomeRecordNotFound.Error()) {
		t.Fatalf("StorePromotionRecord() error = %q, want missing outcome rejection", err.Error())
	}
}

func TestLoadPromotionRecordNotFound(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if _, err := LoadPromotionRecord(root, "missing-promotion"); !errors.Is(err, ErrPromotionRecordNotFound) {
		t.Fatalf("LoadPromotionRecord() error = %v, want %v", err, ErrPromotionRecordNotFound)
	}
}

func validPromotionRecord(now time.Time, mutate func(*PromotionRecord)) PromotionRecord {
	record := PromotionRecord{
		PromotionID:          "promotion-root",
		PromotedPackID:       "pack-candidate",
		PreviousActivePackID: "pack-base",
		LastKnownGoodPackID:  "pack-base",
		LastKnownGoodBasis:   "holdout_pass",
		HotUpdateID:          "hot-update-1",
		OutcomeID:            "outcome-1",
		CandidateID:          "candidate-1",
		RunID:                "run-1",
		CandidateResultID:    "result-1",
		Reason:               "promotion approved",
		Notes:                "persisted without applying control-plane changes",
		PromotedAt:           now,
		CreatedAt:            now.Add(time.Minute),
		CreatedBy:            "system",
	}
	if mutate != nil {
		mutate(&record)
	}
	return record
}

func storePromotionFixtures(t *testing.T, root string, now time.Time) {
	t.Helper()

	storeHotUpdateOutcomeFixtures(t, root, now)
	if err := StoreHotUpdateOutcomeRecord(root, validHotUpdateOutcomeRecord(now.Add(8*time.Minute), func(record *HotUpdateOutcomeRecord) {
		record.OutcomeID = "outcome-1"
		record.HotUpdateID = "hot-update-1"
		record.CandidateID = "candidate-1"
		record.RunID = "run-1"
		record.CandidateResultID = "result-1"
		record.CandidatePackID = "pack-candidate"
		record.OutcomeKind = HotUpdateOutcomeKindPromoted
		record.Reason = "promotion candidate selected"
		record.Notes = "recorded before promotion ledger"
		record.CreatedBy = "operator"
	})); err != nil {
		t.Fatalf("StoreHotUpdateOutcomeRecord() error = %v", err)
	}
}
