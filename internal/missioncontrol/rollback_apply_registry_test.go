package missioncontrol

import (
	"errors"
	"fmt"
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

func TestExecuteRollbackApplyPointerSwitchHappyPathPreservesLastKnownGood(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 30, 15, 0, 0, 0, time.UTC)
	storeRollbackFixtures(t, root, now)
	if err := StoreRollbackRecord(root, validRollbackRecord(now.Add(12*time.Minute), func(record *RollbackRecord) {
		record.RollbackID = "rollback-execute"
		record.CreatedBy = "operator"
	})); err != nil {
		t.Fatalf("StoreRollbackRecord() error = %v", err)
	}
	if _, err := CreateRollbackApplyRecordFromRollback(root, "apply-execute", "rollback-execute", "operator", now.Add(13*time.Minute)); err != nil {
		t.Fatalf("CreateRollbackApplyRecordFromRollback() error = %v", err)
	}
	if _, _, err := AdvanceRollbackApplyPhase(root, "apply-execute", RollbackApplyPhaseValidated, "operator", now.Add(14*time.Minute)); err != nil {
		t.Fatalf("AdvanceRollbackApplyPhase(validated) error = %v", err)
	}
	if _, _, err := AdvanceRollbackApplyPhase(root, "apply-execute", RollbackApplyPhaseReadyToApply, "operator", now.Add(15*time.Minute)); err != nil {
		t.Fatalf("AdvanceRollbackApplyPhase(ready_to_apply) error = %v", err)
	}

	if err := StoreActiveRuntimePackPointer(root, ActiveRuntimePackPointer{
		ActivePackID:         "pack-candidate",
		PreviousActivePackID: "pack-base",
		LastKnownGoodPackID:  "pack-base",
		UpdatedAt:            now.Add(16 * time.Minute),
		UpdatedBy:            "operator",
		UpdateRecordRef:      "promotion:promotion-1",
		ReloadGeneration:     9,
	}); err != nil {
		t.Fatalf("StoreActiveRuntimePackPointer() error = %v", err)
	}
	if err := StoreLastKnownGoodRuntimePackPointer(root, LastKnownGoodRuntimePackPointer{
		PackID:            "pack-base",
		Basis:             "holdout_pass",
		VerifiedAt:        now.Add(17 * time.Minute),
		VerifiedBy:        "operator",
		RollbackRecordRef: "rollback-execute",
	}); err != nil {
		t.Fatalf("StoreLastKnownGoodRuntimePackPointer() error = %v", err)
	}

	beforeLastKnownGoodBytes, err := os.ReadFile(StoreLastKnownGoodRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(last known good before) error = %v", err)
	}

	record, changed, err := ExecuteRollbackApplyPointerSwitch(root, "apply-execute", "operator", now.Add(18*time.Minute))
	if err != nil {
		t.Fatalf("ExecuteRollbackApplyPointerSwitch() error = %v", err)
	}
	if !changed {
		t.Fatal("ExecuteRollbackApplyPointerSwitch() changed = false, want true")
	}
	if record.Phase != RollbackApplyPhasePointerSwitchedReloadPending {
		t.Fatalf("ExecuteRollbackApplyPointerSwitch().Phase = %q, want pointer_switched_reload_pending", record.Phase)
	}
	if record.ActivationState != RollbackApplyActivationStateUnchanged {
		t.Fatalf("ExecuteRollbackApplyPointerSwitch().ActivationState = %q, want unchanged", record.ActivationState)
	}
	if record.PhaseUpdatedBy != "operator" {
		t.Fatalf("ExecuteRollbackApplyPointerSwitch().PhaseUpdatedBy = %q, want operator", record.PhaseUpdatedBy)
	}

	gotPointer, err := LoadActiveRuntimePackPointer(root)
	if err != nil {
		t.Fatalf("LoadActiveRuntimePackPointer() error = %v", err)
	}
	if gotPointer.ActivePackID != "pack-base" {
		t.Fatalf("LoadActiveRuntimePackPointer().ActivePackID = %q, want pack-base", gotPointer.ActivePackID)
	}
	if gotPointer.PreviousActivePackID != "pack-candidate" {
		t.Fatalf("LoadActiveRuntimePackPointer().PreviousActivePackID = %q, want pack-candidate", gotPointer.PreviousActivePackID)
	}
	if gotPointer.LastKnownGoodPackID != "pack-base" {
		t.Fatalf("LoadActiveRuntimePackPointer().LastKnownGoodPackID = %q, want pack-base", gotPointer.LastKnownGoodPackID)
	}
	if gotPointer.UpdateRecordRef != "rollback_apply:apply-execute" {
		t.Fatalf("LoadActiveRuntimePackPointer().UpdateRecordRef = %q, want rollback_apply:apply-execute", gotPointer.UpdateRecordRef)
	}
	if gotPointer.ReloadGeneration != 10 {
		t.Fatalf("LoadActiveRuntimePackPointer().ReloadGeneration = %d, want 10", gotPointer.ReloadGeneration)
	}

	afterLastKnownGoodBytes, err := os.ReadFile(StoreLastKnownGoodRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(last known good after) error = %v", err)
	}
	if string(beforeLastKnownGoodBytes) != string(afterLastKnownGoodBytes) {
		t.Fatalf("last-known-good pointer file changed\nbefore:\n%s\nafter:\n%s", string(beforeLastKnownGoodBytes), string(afterLastKnownGoodBytes))
	}
}

func TestExecuteRollbackApplyPointerSwitchReplayIsIdempotent(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 30, 15, 30, 0, 0, time.UTC)
	storeRollbackFixtures(t, root, now)
	if err := StoreRollbackRecord(root, validRollbackRecord(now.Add(12*time.Minute), func(record *RollbackRecord) {
		record.RollbackID = "rollback-replay-execute"
	})); err != nil {
		t.Fatalf("StoreRollbackRecord() error = %v", err)
	}
	if _, err := CreateRollbackApplyRecordFromRollback(root, "apply-replay-execute", "rollback-replay-execute", "operator", now.Add(13*time.Minute)); err != nil {
		t.Fatalf("CreateRollbackApplyRecordFromRollback() error = %v", err)
	}
	if _, _, err := AdvanceRollbackApplyPhase(root, "apply-replay-execute", RollbackApplyPhaseValidated, "operator", now.Add(14*time.Minute)); err != nil {
		t.Fatalf("AdvanceRollbackApplyPhase(validated) error = %v", err)
	}
	if _, _, err := AdvanceRollbackApplyPhase(root, "apply-replay-execute", RollbackApplyPhaseReadyToApply, "operator", now.Add(15*time.Minute)); err != nil {
		t.Fatalf("AdvanceRollbackApplyPhase(ready_to_apply) error = %v", err)
	}

	if err := StoreActiveRuntimePackPointer(root, ActiveRuntimePackPointer{
		ActivePackID:         "pack-candidate",
		PreviousActivePackID: "pack-base",
		LastKnownGoodPackID:  "pack-base",
		UpdatedAt:            now.Add(16 * time.Minute),
		UpdatedBy:            "operator",
		UpdateRecordRef:      "promotion:promotion-1",
		ReloadGeneration:     4,
	}); err != nil {
		t.Fatalf("StoreActiveRuntimePackPointer() error = %v", err)
	}
	if err := StoreLastKnownGoodRuntimePackPointer(root, LastKnownGoodRuntimePackPointer{
		PackID:            "pack-base",
		Basis:             "holdout_pass",
		VerifiedAt:        now.Add(17 * time.Minute),
		VerifiedBy:        "operator",
		RollbackRecordRef: "rollback-replay-execute",
	}); err != nil {
		t.Fatalf("StoreLastKnownGoodRuntimePackPointer() error = %v", err)
	}

	if _, changed, err := ExecuteRollbackApplyPointerSwitch(root, "apply-replay-execute", "operator", now.Add(18*time.Minute)); err != nil {
		t.Fatalf("ExecuteRollbackApplyPointerSwitch(first) error = %v", err)
	} else if !changed {
		t.Fatal("ExecuteRollbackApplyPointerSwitch(first) changed = false, want true")
	}

	firstPointerBytes, err := os.ReadFile(StoreActiveRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(active pointer first) error = %v", err)
	}
	firstLastKnownGoodBytes, err := os.ReadFile(StoreLastKnownGoodRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(last known good first) error = %v", err)
	}

	record, changed, err := ExecuteRollbackApplyPointerSwitch(root, "apply-replay-execute", "operator", now.Add(19*time.Minute))
	if err != nil {
		t.Fatalf("ExecuteRollbackApplyPointerSwitch(second) error = %v", err)
	}
	if changed {
		t.Fatal("ExecuteRollbackApplyPointerSwitch(second) changed = true, want false")
	}
	if record.Phase != RollbackApplyPhasePointerSwitchedReloadPending {
		t.Fatalf("ExecuteRollbackApplyPointerSwitch(second).Phase = %q, want pointer_switched_reload_pending", record.Phase)
	}

	secondPointerBytes, err := os.ReadFile(StoreActiveRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(active pointer second) error = %v", err)
	}
	if string(firstPointerBytes) != string(secondPointerBytes) {
		t.Fatalf("active runtime pack pointer file changed on idempotent replay\nfirst:\n%s\nsecond:\n%s", string(firstPointerBytes), string(secondPointerBytes))
	}
	secondLastKnownGoodBytes, err := os.ReadFile(StoreLastKnownGoodRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(last known good second) error = %v", err)
	}
	if string(firstLastKnownGoodBytes) != string(secondLastKnownGoodBytes) {
		t.Fatalf("last-known-good pointer file changed on idempotent replay\nfirst:\n%s\nsecond:\n%s", string(firstLastKnownGoodBytes), string(secondLastKnownGoodBytes))
	}

	gotPointer, err := LoadActiveRuntimePackPointer(root)
	if err != nil {
		t.Fatalf("LoadActiveRuntimePackPointer() error = %v", err)
	}
	if gotPointer.ReloadGeneration != 5 {
		t.Fatalf("LoadActiveRuntimePackPointer().ReloadGeneration = %d, want 5", gotPointer.ReloadGeneration)
	}
}

func TestExecuteRollbackApplyPointerSwitchRejectsInvalidPhaseAndMissingRollbackWithoutPointerMutation(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 30, 16, 0, 0, 0, time.UTC)

	t.Run("invalid phase", func(t *testing.T) {
		t.Parallel()

		root := t.TempDir()
		storeRollbackFixtures(t, root, now)
		if err := StoreRollbackRecord(root, validRollbackRecord(now.Add(12*time.Minute), func(record *RollbackRecord) {
			record.RollbackID = "rollback-invalid-execute"
		})); err != nil {
			t.Fatalf("StoreRollbackRecord() error = %v", err)
		}
		if _, err := CreateRollbackApplyRecordFromRollback(root, "apply-invalid-execute", "rollback-invalid-execute", "operator", now.Add(13*time.Minute)); err != nil {
			t.Fatalf("CreateRollbackApplyRecordFromRollback() error = %v", err)
		}
		if err := StoreActiveRuntimePackPointer(root, ActiveRuntimePackPointer{
			ActivePackID:         "pack-candidate",
			PreviousActivePackID: "pack-base",
			LastKnownGoodPackID:  "pack-base",
			UpdatedAt:            now.Add(14 * time.Minute),
			UpdatedBy:            "operator",
			UpdateRecordRef:      "promotion:promotion-1",
			ReloadGeneration:     4,
		}); err != nil {
			t.Fatalf("StoreActiveRuntimePackPointer() error = %v", err)
		}

		beforePointerBytes, err := os.ReadFile(StoreActiveRuntimePackPointerPath(root))
		if err != nil {
			t.Fatalf("ReadFile(active pointer before) error = %v", err)
		}

		got, changed, err := ExecuteRollbackApplyPointerSwitch(root, "apply-invalid-execute", "operator", now.Add(15*time.Minute))
		if err == nil {
			t.Fatal("ExecuteRollbackApplyPointerSwitch() error = nil, want invalid phase rejection")
		}
		if changed {
			t.Fatal("ExecuteRollbackApplyPointerSwitch() changed = true, want false")
		}
		if got != (RollbackApplyRecord{}) {
			t.Fatalf("ExecuteRollbackApplyPointerSwitch() record = %#v, want zero value", got)
		}
		if !strings.Contains(err.Error(), `phase "recorded" does not permit pointer switch execution`) {
			t.Fatalf("ExecuteRollbackApplyPointerSwitch() error = %q, want invalid phase context", err.Error())
		}

		afterPointerBytes, err := os.ReadFile(StoreActiveRuntimePackPointerPath(root))
		if err != nil {
			t.Fatalf("ReadFile(active pointer after) error = %v", err)
		}
		if string(beforePointerBytes) != string(afterPointerBytes) {
			t.Fatalf("active runtime pack pointer file changed on invalid phase rejection\nbefore:\n%s\nafter:\n%s", string(beforePointerBytes), string(afterPointerBytes))
		}
	})

	t.Run("missing rollback", func(t *testing.T) {
		t.Parallel()

		root := t.TempDir()
		storeRollbackFixtures(t, root, now)
		if err := StoreRollbackRecord(root, validRollbackRecord(now.Add(12*time.Minute), func(record *RollbackRecord) {
			record.RollbackID = "rollback-missing-execute"
		})); err != nil {
			t.Fatalf("StoreRollbackRecord() error = %v", err)
		}
		if _, err := CreateRollbackApplyRecordFromRollback(root, "apply-missing-execute", "rollback-missing-execute", "operator", now.Add(13*time.Minute)); err != nil {
			t.Fatalf("CreateRollbackApplyRecordFromRollback() error = %v", err)
		}
		if _, _, err := AdvanceRollbackApplyPhase(root, "apply-missing-execute", RollbackApplyPhaseValidated, "operator", now.Add(14*time.Minute)); err != nil {
			t.Fatalf("AdvanceRollbackApplyPhase(validated) error = %v", err)
		}
		if _, _, err := AdvanceRollbackApplyPhase(root, "apply-missing-execute", RollbackApplyPhaseReadyToApply, "operator", now.Add(15*time.Minute)); err != nil {
			t.Fatalf("AdvanceRollbackApplyPhase(ready_to_apply) error = %v", err)
		}
		if err := StoreActiveRuntimePackPointer(root, ActiveRuntimePackPointer{
			ActivePackID:         "pack-candidate",
			PreviousActivePackID: "pack-base",
			LastKnownGoodPackID:  "pack-base",
			UpdatedAt:            now.Add(16 * time.Minute),
			UpdatedBy:            "operator",
			UpdateRecordRef:      "promotion:promotion-1",
			ReloadGeneration:     7,
		}); err != nil {
			t.Fatalf("StoreActiveRuntimePackPointer() error = %v", err)
		}

		beforePointerBytes, err := os.ReadFile(StoreActiveRuntimePackPointerPath(root))
		if err != nil {
			t.Fatalf("ReadFile(active pointer before) error = %v", err)
		}
		if err := os.Remove(StoreRollbackPath(root, "rollback-missing-execute")); err != nil {
			t.Fatalf("Remove(rollback) error = %v", err)
		}

		got, changed, err := ExecuteRollbackApplyPointerSwitch(root, "apply-missing-execute", "operator", now.Add(17*time.Minute))
		if err == nil {
			t.Fatal("ExecuteRollbackApplyPointerSwitch() error = nil, want missing rollback rejection")
		}
		if changed {
			t.Fatal("ExecuteRollbackApplyPointerSwitch() changed = true, want false")
		}
		if got != (RollbackApplyRecord{}) {
			t.Fatalf("ExecuteRollbackApplyPointerSwitch() record = %#v, want zero value", got)
		}
		if !strings.Contains(err.Error(), ErrRollbackRecordNotFound.Error()) {
			t.Fatalf("ExecuteRollbackApplyPointerSwitch() error = %q, want missing rollback rejection", err.Error())
		}

		afterPointerBytes, err := os.ReadFile(StoreActiveRuntimePackPointerPath(root))
		if err != nil {
			t.Fatalf("ReadFile(active pointer after) error = %v", err)
		}
		if string(beforePointerBytes) != string(afterPointerBytes) {
			t.Fatalf("active runtime pack pointer file changed on missing rollback rejection\nbefore:\n%s\nafter:\n%s", string(beforePointerBytes), string(afterPointerBytes))
		}
	})
}

func TestExecuteRollbackApplyReloadApplyHappyPathPreservesPointerAndLastKnownGood(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 30, 16, 30, 0, 0, time.UTC)
	storeRollbackFixtures(t, root, now)
	if err := StoreRollbackRecord(root, validRollbackRecord(now.Add(12*time.Minute), func(record *RollbackRecord) {
		record.RollbackID = "rollback-reload-success"
		record.CreatedBy = "operator"
	})); err != nil {
		t.Fatalf("StoreRollbackRecord() error = %v", err)
	}
	if _, err := CreateRollbackApplyRecordFromRollback(root, "apply-reload-success", "rollback-reload-success", "operator", now.Add(13*time.Minute)); err != nil {
		t.Fatalf("CreateRollbackApplyRecordFromRollback() error = %v", err)
	}
	if _, _, err := AdvanceRollbackApplyPhase(root, "apply-reload-success", RollbackApplyPhaseValidated, "operator", now.Add(14*time.Minute)); err != nil {
		t.Fatalf("AdvanceRollbackApplyPhase(validated) error = %v", err)
	}
	if _, _, err := AdvanceRollbackApplyPhase(root, "apply-reload-success", RollbackApplyPhaseReadyToApply, "operator", now.Add(15*time.Minute)); err != nil {
		t.Fatalf("AdvanceRollbackApplyPhase(ready_to_apply) error = %v", err)
	}
	if err := StoreActiveRuntimePackPointer(root, ActiveRuntimePackPointer{
		ActivePackID:         "pack-candidate",
		PreviousActivePackID: "pack-base",
		LastKnownGoodPackID:  "pack-base",
		UpdatedAt:            now.Add(15 * time.Minute),
		UpdatedBy:            "operator",
		UpdateRecordRef:      "promotion:promotion-1",
		ReloadGeneration:     7,
	}); err != nil {
		t.Fatalf("StoreActiveRuntimePackPointer() error = %v", err)
	}
	if _, _, err := ExecuteRollbackApplyPointerSwitch(root, "apply-reload-success", "operator", now.Add(16*time.Minute)); err != nil {
		t.Fatalf("ExecuteRollbackApplyPointerSwitch() error = %v", err)
	}
	if err := StoreLastKnownGoodRuntimePackPointer(root, LastKnownGoodRuntimePackPointer{
		PackID:            "pack-base",
		Basis:             "holdout_pass",
		VerifiedAt:        now.Add(17 * time.Minute),
		VerifiedBy:        "operator",
		RollbackRecordRef: "rollback-reload-success",
	}); err != nil {
		t.Fatalf("StoreLastKnownGoodRuntimePackPointer() error = %v", err)
	}

	beforePointerBytes, err := os.ReadFile(StoreActiveRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(active pointer before) error = %v", err)
	}
	beforeLastKnownGoodBytes, err := os.ReadFile(StoreLastKnownGoodRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(last known good before) error = %v", err)
	}

	record, changed, err := ExecuteRollbackApplyReloadApply(root, "apply-reload-success", "operator", now.Add(18*time.Minute))
	if err != nil {
		t.Fatalf("ExecuteRollbackApplyReloadApply() error = %v", err)
	}
	if !changed {
		t.Fatal("ExecuteRollbackApplyReloadApply() changed = false, want true")
	}
	if record.Phase != RollbackApplyPhaseReloadApplySucceeded {
		t.Fatalf("ExecuteRollbackApplyReloadApply().Phase = %q, want reload_apply_succeeded", record.Phase)
	}
	if record.ExecutionError != "" {
		t.Fatalf("ExecuteRollbackApplyReloadApply().ExecutionError = %q, want empty", record.ExecutionError)
	}
	if record.ActivationState != RollbackApplyActivationStateUnchanged {
		t.Fatalf("ExecuteRollbackApplyReloadApply().ActivationState = %q, want unchanged", record.ActivationState)
	}

	afterPointerBytes, err := os.ReadFile(StoreActiveRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(active pointer after) error = %v", err)
	}
	if string(beforePointerBytes) != string(afterPointerBytes) {
		t.Fatalf("active runtime pack pointer file changed during reload/apply\nbefore:\n%s\nafter:\n%s", string(beforePointerBytes), string(afterPointerBytes))
	}
	afterLastKnownGoodBytes, err := os.ReadFile(StoreLastKnownGoodRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(last known good after) error = %v", err)
	}
	if string(beforeLastKnownGoodBytes) != string(afterLastKnownGoodBytes) {
		t.Fatalf("last-known-good pointer file changed during reload/apply\nbefore:\n%s\nafter:\n%s", string(beforeLastKnownGoodBytes), string(afterLastKnownGoodBytes))
	}

	gotPointer, err := LoadActiveRuntimePackPointer(root)
	if err != nil {
		t.Fatalf("LoadActiveRuntimePackPointer() error = %v", err)
	}
	if gotPointer.ActivePackID != "pack-base" {
		t.Fatalf("LoadActiveRuntimePackPointer().ActivePackID = %q, want pack-base", gotPointer.ActivePackID)
	}
	if gotPointer.ReloadGeneration != 8 {
		t.Fatalf("LoadActiveRuntimePackPointer().ReloadGeneration = %d, want 8", gotPointer.ReloadGeneration)
	}
}

func TestExecuteRollbackApplyReloadApplyRecordsFailureWithoutMutatingPointer(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 30, 17, 0, 0, 0, time.UTC)
	storeRollbackFixtures(t, root, now)
	if err := StoreRollbackRecord(root, validRollbackRecord(now.Add(12*time.Minute), func(record *RollbackRecord) {
		record.RollbackID = "rollback-reload-failed"
		record.CreatedBy = "operator"
	})); err != nil {
		t.Fatalf("StoreRollbackRecord() error = %v", err)
	}
	if _, err := CreateRollbackApplyRecordFromRollback(root, "apply-reload-failed", "rollback-reload-failed", "operator", now.Add(13*time.Minute)); err != nil {
		t.Fatalf("CreateRollbackApplyRecordFromRollback() error = %v", err)
	}
	if _, _, err := AdvanceRollbackApplyPhase(root, "apply-reload-failed", RollbackApplyPhaseValidated, "operator", now.Add(14*time.Minute)); err != nil {
		t.Fatalf("AdvanceRollbackApplyPhase(validated) error = %v", err)
	}
	if _, _, err := AdvanceRollbackApplyPhase(root, "apply-reload-failed", RollbackApplyPhaseReadyToApply, "operator", now.Add(15*time.Minute)); err != nil {
		t.Fatalf("AdvanceRollbackApplyPhase(ready_to_apply) error = %v", err)
	}
	if err := StoreActiveRuntimePackPointer(root, ActiveRuntimePackPointer{
		ActivePackID:         "pack-candidate",
		PreviousActivePackID: "pack-base",
		LastKnownGoodPackID:  "pack-base",
		UpdatedAt:            now.Add(15 * time.Minute),
		UpdatedBy:            "operator",
		UpdateRecordRef:      "promotion:promotion-1",
		ReloadGeneration:     7,
	}); err != nil {
		t.Fatalf("StoreActiveRuntimePackPointer() error = %v", err)
	}
	if _, _, err := ExecuteRollbackApplyPointerSwitch(root, "apply-reload-failed", "operator", now.Add(16*time.Minute)); err != nil {
		t.Fatalf("ExecuteRollbackApplyPointerSwitch() error = %v", err)
	}
	if err := StoreLastKnownGoodRuntimePackPointer(root, LastKnownGoodRuntimePackPointer{
		PackID:            "pack-base",
		Basis:             "holdout_pass",
		VerifiedAt:        now.Add(17 * time.Minute),
		VerifiedBy:        "operator",
		RollbackRecordRef: "rollback-reload-failed",
	}); err != nil {
		t.Fatalf("StoreLastKnownGoodRuntimePackPointer() error = %v", err)
	}

	beforePointerBytes, err := os.ReadFile(StoreActiveRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(active pointer before) error = %v", err)
	}
	beforeLastKnownGoodBytes, err := os.ReadFile(StoreLastKnownGoodRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(last known good before) error = %v", err)
	}

	record, changed, err := executeRollbackApplyReloadApplyWithConvergence(root, "apply-reload-failed", "operator", now.Add(18*time.Minute), func(root string, record RollbackApplyRecord, rollback RollbackRecord) error {
		return fmt.Errorf("simulated restart-style convergence failure")
	})
	if err == nil {
		t.Fatal("executeRollbackApplyReloadApplyWithConvergence() error = nil, want failure")
	}
	if !changed {
		t.Fatal("executeRollbackApplyReloadApplyWithConvergence() changed = false, want true")
	}
	if record.Phase != RollbackApplyPhaseReloadApplyFailed {
		t.Fatalf("executeRollbackApplyReloadApplyWithConvergence().Phase = %q, want reload_apply_failed", record.Phase)
	}
	if record.ExecutionError != "simulated restart-style convergence failure" {
		t.Fatalf("executeRollbackApplyReloadApplyWithConvergence().ExecutionError = %q, want simulated failure", record.ExecutionError)
	}

	afterPointerBytes, err := os.ReadFile(StoreActiveRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(active pointer after) error = %v", err)
	}
	if string(beforePointerBytes) != string(afterPointerBytes) {
		t.Fatalf("active runtime pack pointer file changed on reload/apply failure\nbefore:\n%s\nafter:\n%s", string(beforePointerBytes), string(afterPointerBytes))
	}
	afterLastKnownGoodBytes, err := os.ReadFile(StoreLastKnownGoodRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(last known good after) error = %v", err)
	}
	if string(beforeLastKnownGoodBytes) != string(afterLastKnownGoodBytes) {
		t.Fatalf("last-known-good pointer file changed on reload/apply failure\nbefore:\n%s\nafter:\n%s", string(beforeLastKnownGoodBytes), string(afterLastKnownGoodBytes))
	}

	gotPointer, err := LoadActiveRuntimePackPointer(root)
	if err != nil {
		t.Fatalf("LoadActiveRuntimePackPointer() error = %v", err)
	}
	if gotPointer.ReloadGeneration != 8 {
		t.Fatalf("LoadActiveRuntimePackPointer().ReloadGeneration = %d, want 8", gotPointer.ReloadGeneration)
	}
}

func TestExecuteRollbackApplyReloadApplyReplayAfterSuccessIsIdempotent(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 30, 17, 30, 0, 0, time.UTC)
	storeRollbackFixtures(t, root, now)
	if err := StoreRollbackRecord(root, validRollbackRecord(now.Add(12*time.Minute), func(record *RollbackRecord) {
		record.RollbackID = "rollback-reload-replay"
	})); err != nil {
		t.Fatalf("StoreRollbackRecord() error = %v", err)
	}
	if _, err := CreateRollbackApplyRecordFromRollback(root, "apply-reload-replay", "rollback-reload-replay", "operator", now.Add(13*time.Minute)); err != nil {
		t.Fatalf("CreateRollbackApplyRecordFromRollback() error = %v", err)
	}
	if _, _, err := AdvanceRollbackApplyPhase(root, "apply-reload-replay", RollbackApplyPhaseValidated, "operator", now.Add(14*time.Minute)); err != nil {
		t.Fatalf("AdvanceRollbackApplyPhase(validated) error = %v", err)
	}
	if _, _, err := AdvanceRollbackApplyPhase(root, "apply-reload-replay", RollbackApplyPhaseReadyToApply, "operator", now.Add(15*time.Minute)); err != nil {
		t.Fatalf("AdvanceRollbackApplyPhase(ready_to_apply) error = %v", err)
	}
	if err := StoreActiveRuntimePackPointer(root, ActiveRuntimePackPointer{
		ActivePackID:         "pack-candidate",
		PreviousActivePackID: "pack-base",
		LastKnownGoodPackID:  "pack-base",
		UpdatedAt:            now.Add(15 * time.Minute),
		UpdatedBy:            "operator",
		UpdateRecordRef:      "promotion:promotion-1",
		ReloadGeneration:     7,
	}); err != nil {
		t.Fatalf("StoreActiveRuntimePackPointer() error = %v", err)
	}
	if _, _, err := ExecuteRollbackApplyPointerSwitch(root, "apply-reload-replay", "operator", now.Add(16*time.Minute)); err != nil {
		t.Fatalf("ExecuteRollbackApplyPointerSwitch() error = %v", err)
	}

	if _, changed, err := ExecuteRollbackApplyReloadApply(root, "apply-reload-replay", "operator", now.Add(17*time.Minute)); err != nil {
		t.Fatalf("ExecuteRollbackApplyReloadApply(first) error = %v", err)
	} else if !changed {
		t.Fatal("ExecuteRollbackApplyReloadApply(first) changed = false, want true")
	}

	firstPointerBytes, err := os.ReadFile(StoreActiveRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(active pointer first) error = %v", err)
	}

	record, changed, err := ExecuteRollbackApplyReloadApply(root, "apply-reload-replay", "operator", now.Add(18*time.Minute))
	if err != nil {
		t.Fatalf("ExecuteRollbackApplyReloadApply(second) error = %v", err)
	}
	if changed {
		t.Fatal("ExecuteRollbackApplyReloadApply(second) changed = true, want false")
	}
	if record.Phase != RollbackApplyPhaseReloadApplySucceeded {
		t.Fatalf("ExecuteRollbackApplyReloadApply(second).Phase = %q, want reload_apply_succeeded", record.Phase)
	}

	secondPointerBytes, err := os.ReadFile(StoreActiveRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(active pointer second) error = %v", err)
	}
	if string(firstPointerBytes) != string(secondPointerBytes) {
		t.Fatalf("active runtime pack pointer file changed on reload/apply success replay\nfirst:\n%s\nsecond:\n%s", string(firstPointerBytes), string(secondPointerBytes))
	}

	gotPointer, err := LoadActiveRuntimePackPointer(root)
	if err != nil {
		t.Fatalf("LoadActiveRuntimePackPointer() error = %v", err)
	}
	if gotPointer.ReloadGeneration != 8 {
		t.Fatalf("LoadActiveRuntimePackPointer().ReloadGeneration = %d, want 8", gotPointer.ReloadGeneration)
	}
}

func TestExecuteRollbackApplyReloadApplyRejectsInvalidStartingPhase(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 30, 18, 0, 0, 0, time.UTC)
	storeRollbackFixtures(t, root, now)
	if err := StoreRollbackRecord(root, validRollbackRecord(now.Add(12*time.Minute), func(record *RollbackRecord) {
		record.RollbackID = "rollback-reload-invalid"
	})); err != nil {
		t.Fatalf("StoreRollbackRecord() error = %v", err)
	}
	if _, err := CreateRollbackApplyRecordFromRollback(root, "apply-reload-invalid", "rollback-reload-invalid", "operator", now.Add(13*time.Minute)); err != nil {
		t.Fatalf("CreateRollbackApplyRecordFromRollback() error = %v", err)
	}
	if _, _, err := AdvanceRollbackApplyPhase(root, "apply-reload-invalid", RollbackApplyPhaseValidated, "operator", now.Add(14*time.Minute)); err != nil {
		t.Fatalf("AdvanceRollbackApplyPhase(validated) error = %v", err)
	}
	if _, _, err := AdvanceRollbackApplyPhase(root, "apply-reload-invalid", RollbackApplyPhaseReadyToApply, "operator", now.Add(15*time.Minute)); err != nil {
		t.Fatalf("AdvanceRollbackApplyPhase(ready_to_apply) error = %v", err)
	}
	if err := StoreActiveRuntimePackPointer(root, ActiveRuntimePackPointer{
		ActivePackID:         "pack-candidate",
		PreviousActivePackID: "pack-base",
		LastKnownGoodPackID:  "pack-base",
		UpdatedAt:            now.Add(15 * time.Minute),
		UpdatedBy:            "operator",
		UpdateRecordRef:      "promotion:promotion-1",
		ReloadGeneration:     7,
	}); err != nil {
		t.Fatalf("StoreActiveRuntimePackPointer() error = %v", err)
	}
	if _, _, err := ExecuteRollbackApplyPointerSwitch(root, "apply-reload-invalid", "operator", now.Add(16*time.Minute)); err != nil {
		t.Fatalf("ExecuteRollbackApplyPointerSwitch() error = %v", err)
	}
	if _, _, err := executeRollbackApplyReloadApplyWithConvergence(root, "apply-reload-invalid", "operator", now.Add(17*time.Minute), func(root string, record RollbackApplyRecord, rollback RollbackRecord) error {
		return fmt.Errorf("simulated failure")
	}); err == nil {
		t.Fatal("executeRollbackApplyReloadApplyWithConvergence(first) error = nil, want failure")
	}

	beforePointerBytes, err := os.ReadFile(StoreActiveRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(active pointer before) error = %v", err)
	}

	got, changed, err := ExecuteRollbackApplyReloadApply(root, "apply-reload-invalid", "operator", now.Add(18*time.Minute))
	if err == nil {
		t.Fatal("ExecuteRollbackApplyReloadApply() error = nil, want invalid phase rejection")
	}
	if changed {
		t.Fatal("ExecuteRollbackApplyReloadApply() changed = true, want false")
	}
	if got != (RollbackApplyRecord{}) {
		t.Fatalf("ExecuteRollbackApplyReloadApply() record = %#v, want zero value", got)
	}
	if !strings.Contains(err.Error(), `phase "reload_apply_failed" does not permit reload/apply retry`) {
		t.Fatalf("ExecuteRollbackApplyReloadApply() error = %q, want invalid phase rejection", err.Error())
	}

	afterPointerBytes, err := os.ReadFile(StoreActiveRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(active pointer after) error = %v", err)
	}
	if string(beforePointerBytes) != string(afterPointerBytes) {
		t.Fatalf("active runtime pack pointer file changed after invalid starting phase rejection\nbefore:\n%s\nafter:\n%s", string(beforePointerBytes), string(afterPointerBytes))
	}
}

func TestReconcileRollbackApplyRecoveryNeededNormalizesInProgressWithoutMutatingPointerState(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 30, 18, 30, 0, 0, time.UTC)
	storeRollbackApplyReloadInProgressFixture(t, root, now, "rollback-recovery", "apply-recovery")

	beforePointerBytes, err := os.ReadFile(StoreActiveRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(active pointer before) error = %v", err)
	}
	beforeLastKnownGoodBytes, err := os.ReadFile(StoreLastKnownGoodRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(last known good before) error = %v", err)
	}

	record, changed, err := ReconcileRollbackApplyRecoveryNeeded(root, "apply-recovery", "operator", now.Add(19*time.Minute))
	if err != nil {
		t.Fatalf("ReconcileRollbackApplyRecoveryNeeded() error = %v", err)
	}
	if !changed {
		t.Fatal("ReconcileRollbackApplyRecoveryNeeded() changed = false, want true")
	}
	if record.Phase != RollbackApplyPhaseReloadApplyRecoveryNeeded {
		t.Fatalf("ReconcileRollbackApplyRecoveryNeeded().Phase = %q, want reload_apply_recovery_needed", record.Phase)
	}
	if record.ExecutionError != "" {
		t.Fatalf("ReconcileRollbackApplyRecoveryNeeded().ExecutionError = %q, want empty", record.ExecutionError)
	}
	if record.ActivationState != RollbackApplyActivationStateUnchanged {
		t.Fatalf("ReconcileRollbackApplyRecoveryNeeded().ActivationState = %q, want unchanged", record.ActivationState)
	}
	if record.PhaseUpdatedAt != now.Add(19*time.Minute).UTC() {
		t.Fatalf("ReconcileRollbackApplyRecoveryNeeded().PhaseUpdatedAt = %v, want %v", record.PhaseUpdatedAt, now.Add(19*time.Minute).UTC())
	}
	if record.PhaseUpdatedBy != "operator" {
		t.Fatalf("ReconcileRollbackApplyRecoveryNeeded().PhaseUpdatedBy = %q, want operator", record.PhaseUpdatedBy)
	}

	afterPointerBytes, err := os.ReadFile(StoreActiveRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(active pointer after) error = %v", err)
	}
	if string(beforePointerBytes) != string(afterPointerBytes) {
		t.Fatalf("active runtime pack pointer file changed during recovery normalization\nbefore:\n%s\nafter:\n%s", string(beforePointerBytes), string(afterPointerBytes))
	}
	afterLastKnownGoodBytes, err := os.ReadFile(StoreLastKnownGoodRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(last known good after) error = %v", err)
	}
	if string(beforeLastKnownGoodBytes) != string(afterLastKnownGoodBytes) {
		t.Fatalf("last-known-good pointer file changed during recovery normalization\nbefore:\n%s\nafter:\n%s", string(beforeLastKnownGoodBytes), string(afterLastKnownGoodBytes))
	}

	gotPointer, err := LoadActiveRuntimePackPointer(root)
	if err != nil {
		t.Fatalf("LoadActiveRuntimePackPointer() error = %v", err)
	}
	if gotPointer.ReloadGeneration != 8 {
		t.Fatalf("LoadActiveRuntimePackPointer().ReloadGeneration = %d, want 8", gotPointer.ReloadGeneration)
	}
}

func TestReconcileRollbackApplyRecoveryNeededRejectsInvalidLinkageWithoutPointerMutation(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 30, 19, 0, 0, 0, time.UTC)

	t.Run("missing rollback", func(t *testing.T) {
		t.Parallel()

		root := t.TempDir()
		storeRollbackApplyReloadInProgressFixture(t, root, now, "rollback-recovery-missing", "apply-recovery-missing")

		beforePointerBytes, err := os.ReadFile(StoreActiveRuntimePackPointerPath(root))
		if err != nil {
			t.Fatalf("ReadFile(active pointer before) error = %v", err)
		}
		beforeLastKnownGoodBytes, err := os.ReadFile(StoreLastKnownGoodRuntimePackPointerPath(root))
		if err != nil {
			t.Fatalf("ReadFile(last known good before) error = %v", err)
		}
		if err := os.Remove(StoreRollbackPath(root, "rollback-recovery-missing")); err != nil {
			t.Fatalf("Remove(rollback) error = %v", err)
		}

		got, changed, err := ReconcileRollbackApplyRecoveryNeeded(root, "apply-recovery-missing", "operator", now.Add(19*time.Minute))
		if err == nil {
			t.Fatal("ReconcileRollbackApplyRecoveryNeeded() error = nil, want missing rollback rejection")
		}
		if changed {
			t.Fatal("ReconcileRollbackApplyRecoveryNeeded() changed = true, want false")
		}
		if got != (RollbackApplyRecord{}) {
			t.Fatalf("ReconcileRollbackApplyRecoveryNeeded() record = %#v, want zero value", got)
		}
		if !strings.Contains(err.Error(), ErrRollbackRecordNotFound.Error()) {
			t.Fatalf("ReconcileRollbackApplyRecoveryNeeded() error = %q, want missing rollback rejection", err.Error())
		}

		afterPointerBytes, err := os.ReadFile(StoreActiveRuntimePackPointerPath(root))
		if err != nil {
			t.Fatalf("ReadFile(active pointer after) error = %v", err)
		}
		if string(beforePointerBytes) != string(afterPointerBytes) {
			t.Fatalf("active runtime pack pointer file changed on missing rollback rejection\nbefore:\n%s\nafter:\n%s", string(beforePointerBytes), string(afterPointerBytes))
		}
		afterLastKnownGoodBytes, err := os.ReadFile(StoreLastKnownGoodRuntimePackPointerPath(root))
		if err != nil {
			t.Fatalf("ReadFile(last known good after) error = %v", err)
		}
		if string(beforeLastKnownGoodBytes) != string(afterLastKnownGoodBytes) {
			t.Fatalf("last-known-good pointer file changed on missing rollback rejection\nbefore:\n%s\nafter:\n%s", string(beforeLastKnownGoodBytes), string(afterLastKnownGoodBytes))
		}
	})

	t.Run("invalid active pointer linkage", func(t *testing.T) {
		t.Parallel()

		root := t.TempDir()
		storeRollbackApplyReloadInProgressFixture(t, root, now, "rollback-recovery-pointer", "apply-recovery-pointer")

		beforePointerBytes, err := os.ReadFile(StoreActiveRuntimePackPointerPath(root))
		if err != nil {
			t.Fatalf("ReadFile(active pointer before) error = %v", err)
		}
		beforeLastKnownGoodBytes, err := os.ReadFile(StoreLastKnownGoodRuntimePackPointerPath(root))
		if err != nil {
			t.Fatalf("ReadFile(last known good before) error = %v", err)
		}

		pointer, err := LoadActiveRuntimePackPointer(root)
		if err != nil {
			t.Fatalf("LoadActiveRuntimePackPointer() error = %v", err)
		}
		pointer.UpdateRecordRef = "promotion:promotion-2"
		if err := StoreActiveRuntimePackPointer(root, pointer); err != nil {
			t.Fatalf("StoreActiveRuntimePackPointer() error = %v", err)
		}
		mutatedPointerBytes, err := os.ReadFile(StoreActiveRuntimePackPointerPath(root))
		if err != nil {
			t.Fatalf("ReadFile(active pointer mutated) error = %v", err)
		}

		got, changed, err := ReconcileRollbackApplyRecoveryNeeded(root, "apply-recovery-pointer", "operator", now.Add(20*time.Minute))
		if err == nil {
			t.Fatal("ReconcileRollbackApplyRecoveryNeeded() error = nil, want invalid active pointer rejection")
		}
		if changed {
			t.Fatal("ReconcileRollbackApplyRecoveryNeeded() changed = true, want false")
		}
		if got != (RollbackApplyRecord{}) {
			t.Fatalf("ReconcileRollbackApplyRecoveryNeeded() record = %#v, want zero value", got)
		}
		if !strings.Contains(err.Error(), `update_record_ref "rollback_apply:apply-recovery-pointer"`) {
			t.Fatalf("ReconcileRollbackApplyRecoveryNeeded() error = %q, want invalid active pointer context", err.Error())
		}

		afterLastKnownGoodBytes, err := os.ReadFile(StoreLastKnownGoodRuntimePackPointerPath(root))
		if err != nil {
			t.Fatalf("ReadFile(last known good after) error = %v", err)
		}
		if string(beforeLastKnownGoodBytes) != string(afterLastKnownGoodBytes) {
			t.Fatalf("last-known-good pointer file changed on invalid active pointer rejection\nbefore:\n%s\nafter:\n%s", string(beforeLastKnownGoodBytes), string(afterLastKnownGoodBytes))
		}

		afterPointerBytes, err := os.ReadFile(StoreActiveRuntimePackPointerPath(root))
		if err != nil {
			t.Fatalf("ReadFile(active pointer after) error = %v", err)
		}
		if string(beforePointerBytes) == string(mutatedPointerBytes) {
			t.Fatal("active runtime pack pointer file did not reflect the intended invalid linkage setup")
		}
		if string(mutatedPointerBytes) != string(afterPointerBytes) {
			t.Fatalf("active runtime pack pointer file changed during invalid active pointer rejection\nbefore:\n%s\nmutated:\n%s\nafter:\n%s", string(beforePointerBytes), string(mutatedPointerBytes), string(afterPointerBytes))
		}

		gotPointer, err := LoadActiveRuntimePackPointer(root)
		if err != nil {
			t.Fatalf("LoadActiveRuntimePackPointer() error = %v", err)
		}
		if gotPointer.ReloadGeneration != 8 {
			t.Fatalf("LoadActiveRuntimePackPointer().ReloadGeneration = %d, want 8", gotPointer.ReloadGeneration)
		}
	})
}

func TestReconcileRollbackApplyRecoveryNeededReplayIsIdempotent(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 30, 19, 30, 0, 0, time.UTC)
	storeRollbackApplyReloadInProgressFixture(t, root, now, "rollback-recovery-replay", "apply-recovery-replay")

	if _, changed, err := ReconcileRollbackApplyRecoveryNeeded(root, "apply-recovery-replay", "operator", now.Add(19*time.Minute)); err != nil {
		t.Fatalf("ReconcileRollbackApplyRecoveryNeeded(first) error = %v", err)
	} else if !changed {
		t.Fatal("ReconcileRollbackApplyRecoveryNeeded(first) changed = false, want true")
	}

	firstRecordBytes, err := os.ReadFile(StoreRollbackApplyPath(root, "apply-recovery-replay"))
	if err != nil {
		t.Fatalf("ReadFile(record first) error = %v", err)
	}
	firstPointerBytes, err := os.ReadFile(StoreActiveRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(pointer first) error = %v", err)
	}
	firstLastKnownGoodBytes, err := os.ReadFile(StoreLastKnownGoodRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(last known good first) error = %v", err)
	}

	record, changed, err := ReconcileRollbackApplyRecoveryNeeded(root, "apply-recovery-replay", "operator", now.Add(20*time.Minute))
	if err != nil {
		t.Fatalf("ReconcileRollbackApplyRecoveryNeeded(second) error = %v", err)
	}
	if changed {
		t.Fatal("ReconcileRollbackApplyRecoveryNeeded(second) changed = true, want false")
	}
	if record.Phase != RollbackApplyPhaseReloadApplyRecoveryNeeded {
		t.Fatalf("ReconcileRollbackApplyRecoveryNeeded(second).Phase = %q, want reload_apply_recovery_needed", record.Phase)
	}

	secondRecordBytes, err := os.ReadFile(StoreRollbackApplyPath(root, "apply-recovery-replay"))
	if err != nil {
		t.Fatalf("ReadFile(record second) error = %v", err)
	}
	if string(firstRecordBytes) != string(secondRecordBytes) {
		t.Fatalf("rollback apply record file changed on idempotent recovery replay\nfirst:\n%s\nsecond:\n%s", string(firstRecordBytes), string(secondRecordBytes))
	}
	secondPointerBytes, err := os.ReadFile(StoreActiveRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(pointer second) error = %v", err)
	}
	if string(firstPointerBytes) != string(secondPointerBytes) {
		t.Fatalf("active runtime pack pointer file changed on idempotent recovery replay\nfirst:\n%s\nsecond:\n%s", string(firstPointerBytes), string(secondPointerBytes))
	}
	secondLastKnownGoodBytes, err := os.ReadFile(StoreLastKnownGoodRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(last known good second) error = %v", err)
	}
	if string(firstLastKnownGoodBytes) != string(secondLastKnownGoodBytes) {
		t.Fatalf("last-known-good pointer file changed on idempotent recovery replay\nfirst:\n%s\nsecond:\n%s", string(firstLastKnownGoodBytes), string(secondLastKnownGoodBytes))
	}
}

func storeRollbackApplyReloadInProgressFixture(t *testing.T, root string, now time.Time, rollbackID string, applyID string) {
	t.Helper()

	storeRollbackFixtures(t, root, now)
	if err := StoreRollbackRecord(root, validRollbackRecord(now.Add(12*time.Minute), func(record *RollbackRecord) {
		record.RollbackID = rollbackID
		record.CreatedBy = "operator"
	})); err != nil {
		t.Fatalf("StoreRollbackRecord() error = %v", err)
	}
	if _, err := CreateRollbackApplyRecordFromRollback(root, applyID, rollbackID, "operator", now.Add(13*time.Minute)); err != nil {
		t.Fatalf("CreateRollbackApplyRecordFromRollback() error = %v", err)
	}
	if _, _, err := AdvanceRollbackApplyPhase(root, applyID, RollbackApplyPhaseValidated, "operator", now.Add(14*time.Minute)); err != nil {
		t.Fatalf("AdvanceRollbackApplyPhase(validated) error = %v", err)
	}
	if _, _, err := AdvanceRollbackApplyPhase(root, applyID, RollbackApplyPhaseReadyToApply, "operator", now.Add(15*time.Minute)); err != nil {
		t.Fatalf("AdvanceRollbackApplyPhase(ready_to_apply) error = %v", err)
	}
	if err := StoreActiveRuntimePackPointer(root, ActiveRuntimePackPointer{
		ActivePackID:         "pack-candidate",
		PreviousActivePackID: "pack-base",
		LastKnownGoodPackID:  "pack-base",
		UpdatedAt:            now.Add(15 * time.Minute),
		UpdatedBy:            "operator",
		UpdateRecordRef:      "promotion:promotion-1",
		ReloadGeneration:     7,
	}); err != nil {
		t.Fatalf("StoreActiveRuntimePackPointer() error = %v", err)
	}
	if _, _, err := ExecuteRollbackApplyPointerSwitch(root, applyID, "operator", now.Add(16*time.Minute)); err != nil {
		t.Fatalf("ExecuteRollbackApplyPointerSwitch() error = %v", err)
	}
	if err := StoreLastKnownGoodRuntimePackPointer(root, LastKnownGoodRuntimePackPointer{
		PackID:            "pack-base",
		Basis:             "holdout_pass",
		VerifiedAt:        now.Add(17 * time.Minute),
		VerifiedBy:        "operator",
		RollbackRecordRef: rollbackID,
	}); err != nil {
		t.Fatalf("StoreLastKnownGoodRuntimePackPointer() error = %v", err)
	}

	record, err := LoadRollbackApplyRecord(root, applyID)
	if err != nil {
		t.Fatalf("LoadRollbackApplyRecord() error = %v", err)
	}
	record.Phase = RollbackApplyPhaseReloadApplyInProgress
	record.ExecutionError = ""
	record.PhaseUpdatedAt = now.Add(18 * time.Minute).UTC()
	record.PhaseUpdatedBy = "operator"
	record = NormalizeRollbackApplyRecord(record)
	if err := ValidateRollbackApplyRecord(record); err != nil {
		t.Fatalf("ValidateRollbackApplyRecord() error = %v", err)
	}
	if err := WriteStoreJSONAtomic(StoreRollbackApplyPath(root, applyID), record); err != nil {
		t.Fatalf("WriteStoreJSONAtomic(reload_apply_in_progress) error = %v", err)
	}
}

func TestLoadRollbackApplyRecordNotFound(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if _, err := LoadRollbackApplyRecord(root, "missing-apply"); !errors.Is(err, ErrRollbackApplyRecordNotFound) {
		t.Fatalf("LoadRollbackApplyRecord() error = %v, want %v", err, ErrRollbackApplyRecordNotFound)
	}
}
