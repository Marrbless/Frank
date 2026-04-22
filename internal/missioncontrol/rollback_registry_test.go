package missioncontrol

import (
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestRollbackRecordRoundTripAndList(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 29, 10, 0, 0, 0, time.FixedZone("offset", -4*60*60))

	storeRollbackFixtures(t, root, now)

	second := validRollbackRecord(now.Add(12*time.Minute), func(record *RollbackRecord) {
		record.RollbackID = "rollback-b"
		record.PromotionID = ""
		record.HotUpdateID = ""
		record.OutcomeID = ""
		record.LastKnownGoodPackID = ""
		record.Reason = "manual rollback checkpoint"
		record.Notes = "operator record only"
		record.CreatedBy = "system"
	})
	if err := StoreRollbackRecord(root, second); err != nil {
		t.Fatalf("StoreRollbackRecord(rollback-b) error = %v", err)
	}

	want := validRollbackRecord(now.Add(13*time.Minute), func(record *RollbackRecord) {
		record.RollbackID = " rollback-a "
		record.PromotionID = " promotion-1 "
		record.HotUpdateID = " hot-update-1 "
		record.OutcomeID = " outcome-rollback "
		record.FromPackID = " pack-candidate "
		record.TargetPackID = " pack-base "
		record.LastKnownGoodPackID = " pack-base "
		record.Reason = " operator rollback "
		record.Notes = " durable rollback ledger "
		record.CreatedBy = " operator "
	})
	if err := StoreRollbackRecord(root, want); err != nil {
		t.Fatalf("StoreRollbackRecord(rollback-a) error = %v", err)
	}

	got, err := LoadRollbackRecord(root, "rollback-a")
	if err != nil {
		t.Fatalf("LoadRollbackRecord() error = %v", err)
	}

	want.RecordVersion = StoreRecordVersion
	want.RollbackID = "rollback-a"
	want.PromotionID = "promotion-1"
	want.HotUpdateID = "hot-update-1"
	want.OutcomeID = "outcome-rollback"
	want.FromPackID = "pack-candidate"
	want.TargetPackID = "pack-base"
	want.LastKnownGoodPackID = "pack-base"
	want.Reason = "operator rollback"
	want.Notes = "durable rollback ledger"
	want.RollbackAt = want.RollbackAt.UTC()
	want.CreatedAt = want.CreatedAt.UTC()
	want.CreatedBy = "operator"
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("LoadRollbackRecord() = %#v, want %#v", got, want)
	}

	records, err := ListRollbackRecords(root)
	if err != nil {
		t.Fatalf("ListRollbackRecords() error = %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("ListRollbackRecords() len = %d, want 2", len(records))
	}
	if records[0].RollbackID != "rollback-a" || records[1].RollbackID != "rollback-b" {
		t.Fatalf("ListRollbackRecords() ids = [%q %q], want [rollback-a rollback-b]", records[0].RollbackID, records[1].RollbackID)
	}
}

func TestRollbackReplayIsIdempotentAndImmutable(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 29, 11, 0, 0, 0, time.UTC)

	storeRollbackFixtures(t, root, now)

	record := validRollbackRecord(now.Add(12*time.Minute), func(record *RollbackRecord) {
		record.RollbackID = "rollback-replay"
		record.Reason = "exact replay"
		record.Notes = "same bytes expected"
	})
	if err := StoreRollbackRecord(root, record); err != nil {
		t.Fatalf("StoreRollbackRecord(first) error = %v", err)
	}
	firstBytes, err := os.ReadFile(StoreRollbackPath(root, record.RollbackID))
	if err != nil {
		t.Fatalf("ReadFile(first) error = %v", err)
	}

	if err := StoreRollbackRecord(root, record); err != nil {
		t.Fatalf("StoreRollbackRecord(replay) error = %v", err)
	}
	secondBytes, err := os.ReadFile(StoreRollbackPath(root, record.RollbackID))
	if err != nil {
		t.Fatalf("ReadFile(second) error = %v", err)
	}
	if string(firstBytes) != string(secondBytes) {
		t.Fatalf("rollback file changed on idempotent replay\nfirst:\n%s\nsecond:\n%s", string(firstBytes), string(secondBytes))
	}

	err = StoreRollbackRecord(root, validRollbackRecord(now.Add(13*time.Minute), func(changed *RollbackRecord) {
		changed.RollbackID = "rollback-replay"
		changed.Notes = "divergent replay"
	}))
	if err == nil {
		t.Fatal("StoreRollbackRecord() error = nil, want immutable duplicate rejection")
	}
	if !strings.Contains(err.Error(), `mission store rollback "rollback-replay" already exists`) {
		t.Fatalf("StoreRollbackRecord() error = %q, want immutable duplicate rejection", err.Error())
	}
}

func TestRollbackValidationFailsClosed(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 29, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name string
		run  func() error
		want string
	}{
		{
			name: "missing rollback id",
			run: func() error {
				return StoreRollbackRecord(root, validRollbackRecord(now, func(record *RollbackRecord) {
					record.RollbackID = " "
				}))
			},
			want: "rollback ref rollback_id is required",
		},
		{
			name: "missing reason",
			run: func() error {
				return StoreRollbackRecord(root, validRollbackRecord(now, func(record *RollbackRecord) {
					record.Reason = " "
				}))
			},
			want: "mission store rollback reason is required",
		},
		{
			name: "missing rollback time",
			run: func() error {
				return StoreRollbackRecord(root, validRollbackRecord(now, func(record *RollbackRecord) {
					record.RollbackAt = time.Time{}
				}))
			},
			want: "mission store rollback rollback_at is required",
		},
		{
			name: "missing created by",
			run: func() error {
				return StoreRollbackRecord(root, validRollbackRecord(now, func(record *RollbackRecord) {
					record.CreatedBy = " "
				}))
			},
			want: "mission store rollback created_by is required",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := tc.run()
			if err == nil {
				t.Fatal("StoreRollbackRecord() error = nil, want fail-closed rejection")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("StoreRollbackRecord() error = %q, want substring %q", err.Error(), tc.want)
			}
		})
	}
}

func TestRollbackRejectsMissingAndMismatchedLinkedRefs(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 29, 13, 0, 0, 0, time.UTC)

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

	err := StoreRollbackRecord(root, validRollbackRecord(now, func(record *RollbackRecord) {
		record.PromotionID = ""
		record.HotUpdateID = "missing-gate"
		record.OutcomeID = ""
		record.LastKnownGoodPackID = ""
	}))
	if err == nil {
		t.Fatal("StoreRollbackRecord() error = nil, want missing gate rejection")
	}
	if !strings.Contains(err.Error(), ErrHotUpdateGateRecordNotFound.Error()) {
		t.Fatalf("StoreRollbackRecord() error = %q, want missing gate rejection", err.Error())
	}

	storeRollbackFixtures(t, root, now.Add(time.Minute))

	err = StoreRollbackRecord(root, validRollbackRecord(now.Add(12*time.Minute), func(record *RollbackRecord) {
		record.RollbackID = "rollback-mismatch"
		record.TargetPackID = "pack-other"
	}))
	if err == nil {
		t.Fatal("StoreRollbackRecord() error = nil, want target pack linkage mismatch rejection")
	}
	if !strings.Contains(err.Error(), `target_pack_id "pack-other" does not match promotion previous_active_pack_id "pack-base" or last_known_good_pack_id "pack-base"`) {
		t.Fatalf("StoreRollbackRecord() error = %q, want target pack linkage mismatch rejection", err.Error())
	}

	err = StoreRollbackRecord(root, validRollbackRecord(now.Add(13*time.Minute), func(record *RollbackRecord) {
		record.RollbackID = "rollback-missing-promotion"
		record.PromotionID = "promotion-missing"
		record.OutcomeID = ""
	}))
	if err == nil {
		t.Fatal("StoreRollbackRecord() error = nil, want missing promotion rejection")
	}
	if !strings.Contains(err.Error(), ErrPromotionRecordNotFound.Error()) {
		t.Fatalf("StoreRollbackRecord() error = %q, want missing promotion rejection", err.Error())
	}
}

func TestLoadRollbackRecordNotFound(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if _, err := LoadRollbackRecord(root, "missing-rollback"); !errors.Is(err, ErrRollbackRecordNotFound) {
		t.Fatalf("LoadRollbackRecord() error = %v, want %v", err, ErrRollbackRecordNotFound)
	}
}

func validRollbackRecord(now time.Time, mutate func(*RollbackRecord)) RollbackRecord {
	record := RollbackRecord{
		RollbackID:          "rollback-root",
		PromotionID:         "promotion-1",
		HotUpdateID:         "hot-update-1",
		OutcomeID:           "outcome-rollback",
		FromPackID:          "pack-candidate",
		TargetPackID:        "pack-base",
		LastKnownGoodPackID: "pack-base",
		Reason:              "rollback approved",
		Notes:               "persisted without applying runtime changes",
		RollbackAt:          now,
		CreatedAt:           now.Add(time.Minute),
		CreatedBy:           "system",
	}
	if mutate != nil {
		mutate(&record)
	}
	return record
}

func storeRollbackFixtures(t *testing.T, root string, now time.Time) {
	t.Helper()

	storePromotionFixtures(t, root, now)
	if err := StorePromotionRecord(root, validPromotionRecord(now.Add(9*time.Minute), func(record *PromotionRecord) {
		record.PromotionID = "promotion-1"
		record.OutcomeID = "outcome-1"
		record.CreatedBy = "operator"
	})); err != nil {
		t.Fatalf("StorePromotionRecord() error = %v", err)
	}
	if err := StoreHotUpdateOutcomeRecord(root, validHotUpdateOutcomeRecord(now.Add(10*time.Minute), func(record *HotUpdateOutcomeRecord) {
		record.OutcomeID = "outcome-rollback"
		record.HotUpdateID = "hot-update-1"
		record.CandidateID = "candidate-1"
		record.RunID = "run-1"
		record.CandidateResultID = "result-1"
		record.CandidatePackID = "pack-candidate"
		record.OutcomeKind = HotUpdateOutcomeKindRolledBack
		record.Reason = "rollback selected"
		record.Notes = "recorded before rollback ledger"
		record.CreatedBy = "operator"
	})); err != nil {
		t.Fatalf("StoreHotUpdateOutcomeRecord(outcome-rollback) error = %v", err)
	}
	mustStoreRuntimePack(t, root, validRuntimePackRecord(now.Add(11*time.Minute), func(record *RuntimePackRecord) {
		record.PackID = "pack-other"
		record.ParentPackID = "pack-base"
		record.RollbackTargetPackID = "pack-base"
		record.SourceSummary = "alternate rollback target"
	}))
}
