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
		PhaseUpdatedAt:  now.Add(14 * time.Minute).UTC(),
		PhaseUpdatedBy:  "operator",
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
		PhaseUpdatedAt:  now,
		PhaseUpdatedBy:  "operator",
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
		PhaseUpdatedAt:  now.Add(2 * time.Minute),
		PhaseUpdatedBy:  "operator",
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
		PhaseUpdatedAt:  now.Add(13 * time.Minute),
		PhaseUpdatedBy:  "operator",
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
		PhaseUpdatedAt:  now.Add(14 * time.Minute),
		PhaseUpdatedBy:  "operator",
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

func TestLoadRollbackApplyRecordBackfillsLegacyPhaseTransitionMetadata(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 30, 13, 0, 0, 0, time.UTC)
	storeRollbackFixtures(t, root, now)
	if err := StoreRollbackRecord(root, validRollbackRecord(now.Add(12*time.Minute), func(record *RollbackRecord) {
		record.RollbackID = "rollback-legacy"
		record.CreatedBy = "operator"
	})); err != nil {
		t.Fatalf("StoreRollbackRecord() error = %v", err)
	}

	if err := WriteStoreJSONAtomic(StoreRollbackApplyPath(root, "apply-legacy"), struct {
		RecordVersion   int                          `json:"record_version"`
		ApplyID         string                       `json:"apply_id"`
		RollbackID      string                       `json:"rollback_id"`
		Phase           RollbackApplyPhase           `json:"phase"`
		ActivationState RollbackApplyActivationState `json:"activation_state"`
		RequestedAt     time.Time                    `json:"requested_at"`
		CreatedAt       time.Time                    `json:"created_at"`
		CreatedBy       string                       `json:"created_by"`
	}{
		RecordVersion:   StoreRecordVersion,
		ApplyID:         "apply-legacy",
		RollbackID:      "rollback-legacy",
		Phase:           RollbackApplyPhaseRecorded,
		ActivationState: RollbackApplyActivationStateUnchanged,
		RequestedAt:     now.Add(14 * time.Minute),
		CreatedAt:       now.Add(15 * time.Minute),
		CreatedBy:       "operator",
	}); err != nil {
		t.Fatalf("WriteStoreJSONAtomic(apply-legacy) error = %v", err)
	}

	got, err := LoadRollbackApplyRecord(root, "apply-legacy")
	if err != nil {
		t.Fatalf("LoadRollbackApplyRecord() error = %v", err)
	}
	if got.PhaseUpdatedAt != got.CreatedAt {
		t.Fatalf("LoadRollbackApplyRecord().PhaseUpdatedAt = %v, want created_at %v", got.PhaseUpdatedAt, got.CreatedAt)
	}
	if got.PhaseUpdatedBy != got.CreatedBy {
		t.Fatalf("LoadRollbackApplyRecord().PhaseUpdatedBy = %q, want created_by %q", got.PhaseUpdatedBy, got.CreatedBy)
	}
}

func TestAdvanceRollbackApplyPhaseValidProgressionAndPreservesActiveRuntimePackPointer(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 30, 13, 30, 0, 0, time.UTC)
	storeRollbackFixtures(t, root, now)
	if err := StoreRollbackRecord(root, validRollbackRecord(now.Add(12*time.Minute), func(record *RollbackRecord) {
		record.RollbackID = "rollback-progress"
		record.CreatedBy = "operator"
	})); err != nil {
		t.Fatalf("StoreRollbackRecord() error = %v", err)
	}
	if _, err := CreateRollbackApplyRecordFromRollback(root, "apply-progress", "rollback-progress", "operator", now.Add(13*time.Minute)); err != nil {
		t.Fatalf("CreateRollbackApplyRecordFromRollback() error = %v", err)
	}

	wantPointer := ActiveRuntimePackPointer{
		ActivePackID:         "pack-candidate",
		PreviousActivePackID: "pack-base",
		LastKnownGoodPackID:  "pack-base",
		UpdatedAt:            now.Add(14 * time.Minute),
		UpdatedBy:            "operator",
		UpdateRecordRef:      "promotion:promotion-1",
		ReloadGeneration:     9,
	}
	if err := StoreActiveRuntimePackPointer(root, wantPointer); err != nil {
		t.Fatalf("StoreActiveRuntimePackPointer() error = %v", err)
	}
	beforeBytes, err := os.ReadFile(StoreActiveRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(active pointer before) error = %v", err)
	}

	validated, changed, err := AdvanceRollbackApplyPhase(root, "apply-progress", RollbackApplyPhaseValidated, "reviewer", now.Add(15*time.Minute))
	if err != nil {
		t.Fatalf("AdvanceRollbackApplyPhase(validated) error = %v", err)
	}
	if !changed {
		t.Fatal("AdvanceRollbackApplyPhase(validated) changed = false, want true")
	}
	if validated.Phase != RollbackApplyPhaseValidated {
		t.Fatalf("AdvanceRollbackApplyPhase(validated).Phase = %q, want validated", validated.Phase)
	}
	if validated.PhaseUpdatedAt != now.Add(15*time.Minute).UTC() {
		t.Fatalf("AdvanceRollbackApplyPhase(validated).PhaseUpdatedAt = %v, want %v", validated.PhaseUpdatedAt, now.Add(15*time.Minute).UTC())
	}
	if validated.PhaseUpdatedBy != "reviewer" {
		t.Fatalf("AdvanceRollbackApplyPhase(validated).PhaseUpdatedBy = %q, want reviewer", validated.PhaseUpdatedBy)
	}
	if validated.ActivationState != RollbackApplyActivationStateUnchanged {
		t.Fatalf("AdvanceRollbackApplyPhase(validated).ActivationState = %q, want unchanged", validated.ActivationState)
	}

	ready, changed, err := AdvanceRollbackApplyPhase(root, "apply-progress", RollbackApplyPhaseReadyToApply, "operator", now.Add(16*time.Minute))
	if err != nil {
		t.Fatalf("AdvanceRollbackApplyPhase(ready_to_apply) error = %v", err)
	}
	if !changed {
		t.Fatal("AdvanceRollbackApplyPhase(ready_to_apply) changed = false, want true")
	}
	if ready.Phase != RollbackApplyPhaseReadyToApply {
		t.Fatalf("AdvanceRollbackApplyPhase(ready_to_apply).Phase = %q, want ready_to_apply", ready.Phase)
	}
	if ready.PhaseUpdatedAt != now.Add(16*time.Minute).UTC() {
		t.Fatalf("AdvanceRollbackApplyPhase(ready_to_apply).PhaseUpdatedAt = %v, want %v", ready.PhaseUpdatedAt, now.Add(16*time.Minute).UTC())
	}
	if ready.PhaseUpdatedBy != "operator" {
		t.Fatalf("AdvanceRollbackApplyPhase(ready_to_apply).PhaseUpdatedBy = %q, want operator", ready.PhaseUpdatedBy)
	}
	if ready.ActivationState != RollbackApplyActivationStateUnchanged {
		t.Fatalf("AdvanceRollbackApplyPhase(ready_to_apply).ActivationState = %q, want unchanged", ready.ActivationState)
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

func TestAdvanceRollbackApplyPhaseRejectsInvalidTransition(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 30, 14, 0, 0, 0, time.UTC)
	storeRollbackFixtures(t, root, now)
	if err := StoreRollbackRecord(root, validRollbackRecord(now.Add(12*time.Minute), func(record *RollbackRecord) {
		record.RollbackID = "rollback-invalid"
	})); err != nil {
		t.Fatalf("StoreRollbackRecord() error = %v", err)
	}
	if _, err := CreateRollbackApplyRecordFromRollback(root, "apply-invalid", "rollback-invalid", "operator", now.Add(13*time.Minute)); err != nil {
		t.Fatalf("CreateRollbackApplyRecordFromRollback() error = %v", err)
	}

	got, changed, err := AdvanceRollbackApplyPhase(root, "apply-invalid", RollbackApplyPhaseReadyToApply, "operator", now.Add(14*time.Minute))
	if err == nil {
		t.Fatal("AdvanceRollbackApplyPhase() error = nil, want invalid transition rejection")
	}
	if changed {
		t.Fatal("AdvanceRollbackApplyPhase() changed = true, want false")
	}
	if got != (RollbackApplyRecord{}) {
		t.Fatalf("AdvanceRollbackApplyPhase() record = %#v, want zero value on rejection", got)
	}
	if !strings.Contains(err.Error(), `phase transition "recorded" -> "ready_to_apply" is invalid`) {
		t.Fatalf("AdvanceRollbackApplyPhase() error = %q, want invalid transition context", err.Error())
	}
}

func TestAdvanceRollbackApplyPhaseIsIdempotentForSamePhase(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 30, 14, 30, 0, 0, time.UTC)
	storeRollbackFixtures(t, root, now)
	if err := StoreRollbackRecord(root, validRollbackRecord(now.Add(12*time.Minute), func(record *RollbackRecord) {
		record.RollbackID = "rollback-idempotent"
	})); err != nil {
		t.Fatalf("StoreRollbackRecord() error = %v", err)
	}
	if _, err := CreateRollbackApplyRecordFromRollback(root, "apply-idempotent", "rollback-idempotent", "operator", now.Add(13*time.Minute)); err != nil {
		t.Fatalf("CreateRollbackApplyRecordFromRollback() error = %v", err)
	}
	first, changed, err := AdvanceRollbackApplyPhase(root, "apply-idempotent", RollbackApplyPhaseValidated, "reviewer", now.Add(14*time.Minute))
	if err != nil {
		t.Fatalf("AdvanceRollbackApplyPhase(first) error = %v", err)
	}
	if !changed {
		t.Fatal("AdvanceRollbackApplyPhase(first) changed = false, want true")
	}
	firstBytes, err := os.ReadFile(StoreRollbackApplyPath(root, "apply-idempotent"))
	if err != nil {
		t.Fatalf("ReadFile(first) error = %v", err)
	}

	second, changed, err := AdvanceRollbackApplyPhase(root, "apply-idempotent", RollbackApplyPhaseValidated, "reviewer", now.Add(14*time.Minute))
	if err != nil {
		t.Fatalf("AdvanceRollbackApplyPhase(second) error = %v", err)
	}
	if changed {
		t.Fatal("AdvanceRollbackApplyPhase(second) changed = true, want false")
	}
	if !reflect.DeepEqual(second, first) {
		t.Fatalf("AdvanceRollbackApplyPhase(second) = %#v, want %#v", second, first)
	}
	secondBytes, err := os.ReadFile(StoreRollbackApplyPath(root, "apply-idempotent"))
	if err != nil {
		t.Fatalf("ReadFile(second) error = %v", err)
	}
	if string(firstBytes) != string(secondBytes) {
		t.Fatalf("rollback apply file changed on idempotent phase replay\nfirst:\n%s\nsecond:\n%s", string(firstBytes), string(secondBytes))
	}
}

func TestLoadRollbackApplyRecordNotFound(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if _, err := LoadRollbackApplyRecord(root, "missing-apply"); !errors.Is(err, ErrRollbackApplyRecordNotFound) {
		t.Fatalf("LoadRollbackApplyRecord() error = %v, want %v", err, ErrRollbackApplyRecordNotFound)
	}
}
