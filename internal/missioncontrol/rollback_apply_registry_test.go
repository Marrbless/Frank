package missioncontrol

import (
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestCreateRollbackApplyRecordFromCommittedRollback(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 30, 10, 0, 0, 0, time.UTC)
	storeRollbackFixtures(t, root, now)

	if err := StoreRollbackRecord(root, validRollbackRecord(now.Add(12*time.Minute), func(record *RollbackRecord) {
		record.RollbackID = "rollback-apply-root"
		record.CreatedBy = "operator"
	})); err != nil {
		t.Fatalf("StoreRollbackRecord() error = %v", err)
	}

	wantPointer := ActiveRuntimePackPointer{
		ActivePackID:         "pack-candidate",
		PreviousActivePackID: "pack-base",
		LastKnownGoodPackID:  "pack-base",
		UpdatedAt:            now.Add(13 * time.Minute),
		UpdatedBy:            "operator",
		UpdateRecordRef:      "promotion:promotion-1",
		ReloadGeneration:     4,
	}
	if err := StoreActiveRuntimePackPointer(root, wantPointer); err != nil {
		t.Fatalf("StoreActiveRuntimePackPointer() error = %v", err)
	}
	beforeBytes, err := os.ReadFile(StoreActiveRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(active pointer before) error = %v", err)
	}

	got, err := CreateRollbackApplyRecordFromRollback(root, "apply-1", "rollback-apply-root", " operator ", now.Add(14*time.Minute))
	if err != nil {
		t.Fatalf("CreateRollbackApplyRecordFromRollback() error = %v", err)
	}

	want := RollbackApplyRecord{
		RecordVersion:   StoreRecordVersion,
		ApplyID:         "apply-1",
		RollbackID:      "rollback-apply-root",
		Phase:           RollbackApplyPhaseRecorded,
		ActivationState: RollbackApplyActivationStateUnchanged,
		RequestedAt:     now.Add(14 * time.Minute).UTC(),
		CreatedAt:       now.Add(14 * time.Minute).UTC(),
		CreatedBy:       "operator",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("CreateRollbackApplyRecordFromRollback() = %#v, want %#v", got, want)
	}

	loaded, err := LoadRollbackApplyRecord(root, "apply-1")
	if err != nil {
		t.Fatalf("LoadRollbackApplyRecord() error = %v", err)
	}
	if !reflect.DeepEqual(loaded, want) {
		t.Fatalf("LoadRollbackApplyRecord() = %#v, want %#v", loaded, want)
	}

	records, err := ListRollbackApplyRecords(root)
	if err != nil {
		t.Fatalf("ListRollbackApplyRecords() error = %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("ListRollbackApplyRecords() len = %d, want 1", len(records))
	}
	if !reflect.DeepEqual(records[0], want) {
		t.Fatalf("ListRollbackApplyRecords()[0] = %#v, want %#v", records[0], want)
	}

	gotPointer, err := LoadActiveRuntimePackPointer(root)
	if err != nil {
		t.Fatalf("LoadActiveRuntimePackPointer() error = %v", err)
	}
	wantPointer.RecordVersion = StoreRecordVersion
	wantPointer = NormalizeActiveRuntimePackPointer(wantPointer)
	if !reflect.DeepEqual(gotPointer, wantPointer) {
		t.Fatalf("LoadActiveRuntimePackPointer() = %#v, want %#v", gotPointer, wantPointer)
	}
	afterBytes, err := os.ReadFile(StoreActiveRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(active pointer after) error = %v", err)
	}
	if string(beforeBytes) != string(afterBytes) {
		t.Fatalf("active runtime pack pointer file changed\nbefore:\n%s\nafter:\n%s", string(beforeBytes), string(afterBytes))
	}
}

func TestRollbackApplyRecordRejectsMissingOrInvalidRollbackRefs(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 30, 11, 0, 0, 0, time.UTC)

	err := StoreRollbackApplyRecord(root, RollbackApplyRecord{
		ApplyID:         "apply-missing",
		RollbackID:      "missing-rollback",
		Phase:           RollbackApplyPhaseRecorded,
		ActivationState: RollbackApplyActivationStateUnchanged,
		RequestedAt:     now,
		CreatedAt:       now,
		CreatedBy:       "operator",
	})
	if err == nil {
		t.Fatal("StoreRollbackApplyRecord() error = nil, want missing rollback rejection")
	}
	if !strings.Contains(err.Error(), ErrRollbackRecordNotFound.Error()) {
		t.Fatalf("StoreRollbackApplyRecord() error = %q, want missing rollback rejection", err.Error())
	}

	storeRollbackFixtures(t, root, now.Add(time.Minute))
	if err := WriteStoreJSONAtomic(StoreRollbackApplyPath(root, "apply-orphan"), RollbackApplyRecord{
		RecordVersion:   StoreRecordVersion,
		ApplyID:         "apply-orphan",
		RollbackID:      "rollback-orphan",
		Phase:           RollbackApplyPhaseRecorded,
		ActivationState: RollbackApplyActivationStateUnchanged,
		RequestedAt:     now.Add(2 * time.Minute),
		CreatedAt:       now.Add(2 * time.Minute),
		CreatedBy:       "operator",
	}); err != nil {
		t.Fatalf("WriteStoreJSONAtomic(apply-orphan) error = %v", err)
	}

	if _, err := LoadRollbackApplyRecord(root, "apply-orphan"); err == nil {
		t.Fatal("LoadRollbackApplyRecord() error = nil, want fail-closed orphan rollback rejection")
	} else if !strings.Contains(err.Error(), `rollback_id "rollback-orphan"`) {
		t.Fatalf("LoadRollbackApplyRecord() error = %q, want orphan rollback context", err.Error())
	}
}

func TestRollbackApplyReplayIsIdempotentAndImmutable(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 30, 12, 0, 0, 0, time.UTC)
	storeRollbackFixtures(t, root, now)
	if err := StoreRollbackRecord(root, validRollbackRecord(now.Add(12*time.Minute), func(record *RollbackRecord) {
		record.RollbackID = "rollback-replay"
	})); err != nil {
		t.Fatalf("StoreRollbackRecord() error = %v", err)
	}

	record := RollbackApplyRecord{
		ApplyID:         "apply-replay",
		RollbackID:      "rollback-replay",
		Phase:           RollbackApplyPhaseRecorded,
		ActivationState: RollbackApplyActivationStateUnchanged,
		RequestedAt:     now.Add(13 * time.Minute),
		CreatedAt:       now.Add(13 * time.Minute),
		CreatedBy:       "operator",
	}
	if err := StoreRollbackApplyRecord(root, record); err != nil {
		t.Fatalf("StoreRollbackApplyRecord(first) error = %v", err)
	}
	firstBytes, err := os.ReadFile(StoreRollbackApplyPath(root, record.ApplyID))
	if err != nil {
		t.Fatalf("ReadFile(first) error = %v", err)
	}

	if err := StoreRollbackApplyRecord(root, record); err != nil {
		t.Fatalf("StoreRollbackApplyRecord(replay) error = %v", err)
	}
	secondBytes, err := os.ReadFile(StoreRollbackApplyPath(root, record.ApplyID))
	if err != nil {
		t.Fatalf("ReadFile(second) error = %v", err)
	}
	if string(firstBytes) != string(secondBytes) {
		t.Fatalf("rollback apply file changed on idempotent replay\nfirst:\n%s\nsecond:\n%s", string(firstBytes), string(secondBytes))
	}

	err = StoreRollbackApplyRecord(root, RollbackApplyRecord{
		ApplyID:         "apply-replay",
		RollbackID:      "rollback-replay",
		Phase:           RollbackApplyPhaseValidated,
		ActivationState: RollbackApplyActivationStateUnchanged,
		RequestedAt:     now.Add(13 * time.Minute),
		CreatedAt:       now.Add(14 * time.Minute),
		CreatedBy:       "operator",
	})
	if err == nil {
		t.Fatal("StoreRollbackApplyRecord() error = nil, want immutable duplicate rejection")
	}
	if !strings.Contains(err.Error(), `mission store rollback apply "apply-replay" already exists`) {
		t.Fatalf("StoreRollbackApplyRecord() error = %q, want immutable duplicate rejection", err.Error())
	}
}

func TestEnsureRollbackApplyRecordFromRollbackCreatesOrSelectsExistingMatch(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 30, 12, 30, 0, 0, time.UTC)
	storeRollbackFixtures(t, root, now)
	if err := StoreRollbackRecord(root, validRollbackRecord(now.Add(12*time.Minute), func(record *RollbackRecord) {
		record.RollbackID = "rollback-select"
		record.CreatedBy = "operator"
	})); err != nil {
		t.Fatalf("StoreRollbackRecord() error = %v", err)
	}

	first, created, err := EnsureRollbackApplyRecordFromRollback(root, "apply-select", "rollback-select", "operator", now.Add(13*time.Minute))
	if err != nil {
		t.Fatalf("EnsureRollbackApplyRecordFromRollback(first) error = %v", err)
	}
	if !created {
		t.Fatal("EnsureRollbackApplyRecordFromRollback(first) created = false, want true")
	}

	firstBytes, err := os.ReadFile(StoreRollbackApplyPath(root, "apply-select"))
	if err != nil {
		t.Fatalf("ReadFile(first) error = %v", err)
	}

	second, created, err := EnsureRollbackApplyRecordFromRollback(root, "apply-select", "rollback-select", "other-operator", now.Add(17*time.Minute))
	if err != nil {
		t.Fatalf("EnsureRollbackApplyRecordFromRollback(second) error = %v", err)
	}
	if created {
		t.Fatal("EnsureRollbackApplyRecordFromRollback(second) created = true, want false")
	}
	if !reflect.DeepEqual(second, first) {
		t.Fatalf("EnsureRollbackApplyRecordFromRollback(second) = %#v, want %#v", second, first)
	}

	secondBytes, err := os.ReadFile(StoreRollbackApplyPath(root, "apply-select"))
	if err != nil {
		t.Fatalf("ReadFile(second) error = %v", err)
	}
	if string(firstBytes) != string(secondBytes) {
		t.Fatalf("rollback apply file changed on select path\nfirst:\n%s\nsecond:\n%s", string(firstBytes), string(secondBytes))
	}
}

func TestEnsureRollbackApplyRecordFromRollbackRejectsMismatchedExistingRollback(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 30, 12, 45, 0, 0, time.UTC)
	storeRollbackFixtures(t, root, now)
	if err := StoreRollbackRecord(root, validRollbackRecord(now.Add(12*time.Minute), func(record *RollbackRecord) {
		record.RollbackID = "rollback-a"
		record.CreatedBy = "operator"
	})); err != nil {
		t.Fatalf("StoreRollbackRecord(rollback-a) error = %v", err)
	}
	if err := StoreRollbackRecord(root, validRollbackRecord(now.Add(13*time.Minute), func(record *RollbackRecord) {
		record.RollbackID = "rollback-b"
		record.CreatedBy = "operator"
	})); err != nil {
		t.Fatalf("StoreRollbackRecord(rollback-b) error = %v", err)
	}
	if _, err := CreateRollbackApplyRecordFromRollback(root, "apply-mismatch", "rollback-a", "operator", now.Add(14*time.Minute)); err != nil {
		t.Fatalf("CreateRollbackApplyRecordFromRollback() error = %v", err)
	}

	_, created, err := EnsureRollbackApplyRecordFromRollback(root, "apply-mismatch", "rollback-b", "operator", now.Add(15*time.Minute))
	if err == nil {
		t.Fatal("EnsureRollbackApplyRecordFromRollback() error = nil, want mismatched rollback rejection")
	}
	if created {
		t.Fatal("EnsureRollbackApplyRecordFromRollback() created = true, want false")
	}
	if !strings.Contains(err.Error(), `rollback_id "rollback-a" does not match requested rollback_id "rollback-b"`) {
		t.Fatalf("EnsureRollbackApplyRecordFromRollback() error = %q, want mismatched rollback context", err.Error())
	}
}

func TestLoadRollbackApplyRecordNotFound(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if _, err := LoadRollbackApplyRecord(root, "missing-apply"); !errors.Is(err, ErrRollbackApplyRecordNotFound) {
		t.Fatalf("LoadRollbackApplyRecord() error = %v, want %v", err, ErrRollbackApplyRecordNotFound)
	}
}
