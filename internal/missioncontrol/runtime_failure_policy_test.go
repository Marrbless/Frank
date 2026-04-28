package missioncontrol

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestAssessRepeatedActivePackFailuresRollsBackToLKGAndQuarantines(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 26, 14, 0, 0, 0, time.UTC)
	storeRepeatedFailurePolicyFixtures(t, root, now, true)
	storeRuntimeFailureEventSequence(t, root, now, "pack-candidate", "hot-update-1", 3)

	result, err := AssessRepeatedActivePackFailures(root, RepeatedActivePackFailurePolicySpec{
		AssessmentID: "repeated-candidate",
		CreatedAt:    now.Add(10 * time.Minute),
		CreatedBy:    "watchdog",
	})
	if err != nil {
		t.Fatalf("AssessRepeatedActivePackFailures() error = %v", err)
	}
	if result.Action != RepeatedActivePackFailureActionRollbackTriggered {
		t.Fatalf("Action = %q, want rollback_triggered", result.Action)
	}
	if result.ConsecutiveFailureCount != 3 {
		t.Fatalf("ConsecutiveFailureCount = %d, want 3", result.ConsecutiveFailureCount)
	}
	if !equalStrings(result.FailureEventIDs, []string{"failure-1", "failure-2", "failure-3"}) {
		t.Fatalf("FailureEventIDs = %#v, want failure-1..failure-3", result.FailureEventIDs)
	}

	pointer, err := LoadActiveRuntimePackPointer(root)
	if err != nil {
		t.Fatalf("LoadActiveRuntimePackPointer() error = %v", err)
	}
	if pointer.ActivePackID != "pack-base" {
		t.Fatalf("ActivePackID = %q, want pack-base rollback target", pointer.ActivePackID)
	}
	if pointer.PreviousActivePackID != "pack-candidate" {
		t.Fatalf("PreviousActivePackID = %q, want pack-candidate", pointer.PreviousActivePackID)
	}
	if pointer.UpdateRecordRef != "rollback_apply:apply-repeated-candidate" {
		t.Fatalf("UpdateRecordRef = %q, want rollback apply update ref", pointer.UpdateRecordRef)
	}

	quarantine, err := LoadRuntimePackQuarantineRecord(root, "quarantine-repeated-candidate")
	if err != nil {
		t.Fatalf("LoadRuntimePackQuarantineRecord() error = %v", err)
	}
	if quarantine.PackID != "pack-candidate" {
		t.Fatalf("quarantine PackID = %q, want pack-candidate", quarantine.PackID)
	}
	if quarantine.State != RuntimePackQuarantineStateQuarantined {
		t.Fatalf("quarantine State = %q, want quarantined", quarantine.State)
	}
	if quarantine.RollbackID != "rollback-repeated-candidate" || quarantine.RollbackApplyID != "apply-repeated-candidate" {
		t.Fatalf("quarantine rollback refs = %q/%q, want rollback/apply refs", quarantine.RollbackID, quarantine.RollbackApplyID)
	}

	rollback, err := LoadRollbackRecord(root, "rollback-repeated-candidate")
	if err != nil {
		t.Fatalf("LoadRollbackRecord() error = %v", err)
	}
	if rollback.FromPackID != "pack-candidate" || rollback.TargetPackID != "pack-base" || rollback.LastKnownGoodPackID != "pack-base" {
		t.Fatalf("rollback packs = from %q target %q lkg %q, want candidate -> base", rollback.FromPackID, rollback.TargetPackID, rollback.LastKnownGoodPackID)
	}
	if rollback.HotUpdateID != "hot-update-1" {
		t.Fatalf("rollback HotUpdateID = %q, want hot-update-1", rollback.HotUpdateID)
	}

	apply, err := LoadRollbackApplyRecord(root, "apply-repeated-candidate")
	if err != nil {
		t.Fatalf("LoadRollbackApplyRecord() error = %v", err)
	}
	if apply.Phase != RollbackApplyPhasePointerSwitchedReloadPending {
		t.Fatalf("rollback apply Phase = %q, want pointer_switched_reload_pending", apply.Phase)
	}
}

func TestAssessRepeatedActivePackFailuresRecordsTerminalBlockerWithoutLKG(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 26, 15, 0, 0, 0, time.UTC)
	storeRepeatedFailurePolicyFixtures(t, root, now, false)
	storeRuntimeFailureEventSequence(t, root, now, "pack-candidate", "hot-update-1", 3)
	beforePointerBytes := mustReadFileBytes(t, StoreActiveRuntimePackPointerPath(root))

	result, err := AssessRepeatedActivePackFailures(root, RepeatedActivePackFailurePolicySpec{
		AssessmentID: "no-lkg",
		CreatedAt:    now.Add(10 * time.Minute),
		CreatedBy:    "watchdog",
	})
	if err != nil {
		t.Fatalf("AssessRepeatedActivePackFailures() error = %v", err)
	}
	if result.Action != RepeatedActivePackFailureActionTerminalBlocked {
		t.Fatalf("Action = %q, want terminal_blocked", result.Action)
	}
	if result.TerminalBlockerID != "blocker-no-lkg" {
		t.Fatalf("TerminalBlockerID = %q, want blocker-no-lkg", result.TerminalBlockerID)
	}
	assertBytesEqual(t, "active runtime-pack pointer", beforePointerBytes, mustReadFileBytes(t, StoreActiveRuntimePackPointerPath(root)))

	blocker, err := LoadRepeatedFailureTerminalBlockerRecord(root, "blocker-no-lkg")
	if err != nil {
		t.Fatalf("LoadRepeatedFailureTerminalBlockerRecord() error = %v", err)
	}
	if blocker.PackID != "pack-candidate" {
		t.Fatalf("blocker PackID = %q, want pack-candidate", blocker.PackID)
	}
	if !strings.Contains(blocker.Reason, "no distinct last-known-good recovery target") {
		t.Fatalf("blocker Reason = %q, want no-LKG context", blocker.Reason)
	}
	if _, err := LoadRuntimePackQuarantineRecord(root, "quarantine-no-lkg"); !errors.Is(err, ErrRuntimePackQuarantineRecordNotFound) {
		t.Fatalf("LoadRuntimePackQuarantineRecord(quarantine-no-lkg) error = %v, want not found", err)
	}
}

func TestAssessRepeatedActivePackFailuresBelowThresholdNoop(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 26, 16, 0, 0, 0, time.UTC)
	storeRepeatedFailurePolicyFixtures(t, root, now, true)
	storeRuntimeFailureEventSequence(t, root, now, "pack-candidate", "hot-update-1", 2)
	beforePointerBytes := mustReadFileBytes(t, StoreActiveRuntimePackPointerPath(root))

	result, err := AssessRepeatedActivePackFailures(root, RepeatedActivePackFailurePolicySpec{
		AssessmentID: "below-threshold",
		CreatedAt:    now.Add(10 * time.Minute),
		CreatedBy:    "watchdog",
	})
	if err != nil {
		t.Fatalf("AssessRepeatedActivePackFailures() error = %v", err)
	}
	if result.Action != RepeatedActivePackFailureActionNone {
		t.Fatalf("Action = %q, want none", result.Action)
	}
	if result.ConsecutiveFailureCount != 2 {
		t.Fatalf("ConsecutiveFailureCount = %d, want 2", result.ConsecutiveFailureCount)
	}
	assertBytesEqual(t, "active runtime-pack pointer", beforePointerBytes, mustReadFileBytes(t, StoreActiveRuntimePackPointerPath(root)))
}

func TestRuntimeFailureEventReplayAndDivergentDuplicate(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 26, 17, 0, 0, 0, time.UTC)
	storeRepeatedFailurePolicyFixtures(t, root, now, true)

	record := validRuntimeFailureEventRecord(now.Add(8*time.Minute), "failure-replay", "pack-candidate", "hot-update-1", RuntimeFailureKindRuntime)
	stored, created, err := StoreRuntimeFailureEventRecord(root, record)
	if err != nil {
		t.Fatalf("StoreRuntimeFailureEventRecord(first) error = %v", err)
	}
	if !created {
		t.Fatal("StoreRuntimeFailureEventRecord(first) created = false, want true")
	}
	replayed, created, err := StoreRuntimeFailureEventRecord(root, record)
	if err != nil {
		t.Fatalf("StoreRuntimeFailureEventRecord(replay) error = %v", err)
	}
	if created {
		t.Fatal("StoreRuntimeFailureEventRecord(replay) created = true, want false")
	}
	if replayed.EventID != stored.EventID {
		t.Fatalf("replayed EventID = %q, want %q", replayed.EventID, stored.EventID)
	}

	divergent := record
	divergent.FailureReason = "different reason"
	if _, _, err := StoreRuntimeFailureEventRecord(root, divergent); err == nil {
		t.Fatal("StoreRuntimeFailureEventRecord(divergent) error = nil, want duplicate rejection")
	} else if !strings.Contains(err.Error(), `mission store runtime failure event "failure-replay" already exists`) {
		t.Fatalf("StoreRuntimeFailureEventRecord(divergent) error = %q, want duplicate context", err.Error())
	}
}

func storeRepeatedFailurePolicyFixtures(t *testing.T, root string, now time.Time, withLKG bool) {
	t.Helper()

	mustStoreRuntimePack(t, root, validRuntimePackRecord(now, func(record *RuntimePackRecord) {
		record.PackID = "pack-base"
		record.SourceSummary = "last known good pack"
	}))
	mustStoreRuntimePack(t, root, validRuntimePackRecord(now.Add(time.Minute), func(record *RuntimePackRecord) {
		record.PackID = "pack-candidate"
		record.ParentPackID = "pack-base"
		record.RollbackTargetPackID = "pack-base"
		record.SourceSummary = "hot-updated candidate pack"
	}))
	if err := StoreActiveRuntimePackPointer(root, ActiveRuntimePackPointer{
		ActivePackID:     "pack-base",
		UpdatedAt:        now.Add(2 * time.Minute),
		UpdatedBy:        "bootstrap",
		UpdateRecordRef:  "bootstrap-active",
		ReloadGeneration: 0,
	}); err != nil {
		t.Fatalf("StoreActiveRuntimePackPointer(base) error = %v", err)
	}
	if err := StoreHotUpdateGateRecord(root, validHotUpdateGateRecord(now.Add(3*time.Minute), func(record *HotUpdateGateRecord) {
		record.HotUpdateID = "hot-update-1"
		record.CandidatePackID = "pack-candidate"
		record.PreviousActivePackID = "pack-base"
		record.RollbackTargetPackID = "pack-base"
	})); err != nil {
		t.Fatalf("StoreHotUpdateGateRecord() error = %v", err)
	}
	if withLKG {
		if err := StoreLastKnownGoodRuntimePackPointer(root, LastKnownGoodRuntimePackPointer{
			PackID:            "pack-base",
			Basis:             "smoke_check",
			VerifiedAt:        now.Add(4 * time.Minute),
			VerifiedBy:        "operator",
			RollbackRecordRef: "bootstrap-lkg",
		}); err != nil {
			t.Fatalf("StoreLastKnownGoodRuntimePackPointer() error = %v", err)
		}
	}
	if err := StoreActiveRuntimePackPointer(root, ActiveRuntimePackPointer{
		ActivePackID:         "pack-candidate",
		PreviousActivePackID: "pack-base",
		LastKnownGoodPackID:  "pack-base",
		UpdatedAt:            now.Add(5 * time.Minute),
		UpdatedBy:            "hot-update-gate",
		UpdateRecordRef:      "hot_update:hot-update-1",
		ReloadGeneration:     1,
	}); err != nil {
		t.Fatalf("StoreActiveRuntimePackPointer(candidate) error = %v", err)
	}
}

func storeRuntimeFailureEventSequence(t *testing.T, root string, now time.Time, packID string, hotUpdateID string, count int) {
	t.Helper()

	for i := 1; i <= count; i++ {
		kind := RuntimeFailureKindRuntime
		if i%2 == 0 {
			kind = RuntimeFailureKindSmoke
		}
		if _, _, err := StoreRuntimeFailureEventRecord(root, validRuntimeFailureEventRecord(
			now.Add(time.Duration(5+i)*time.Minute),
			"failure-"+string(rune('0'+i)),
			packID,
			hotUpdateID,
			kind,
		)); err != nil {
			t.Fatalf("StoreRuntimeFailureEventRecord(%d) error = %v", i, err)
		}
	}
}

func validRuntimeFailureEventRecord(now time.Time, eventID string, packID string, hotUpdateID string, kind RuntimeFailureKind) RuntimeFailureEventRecord {
	return RuntimeFailureEventRecord{
		EventID:       eventID,
		PackID:        packID,
		HotUpdateID:   hotUpdateID,
		FailureKind:   kind,
		FailureReason: "deterministic fake failure",
		ObservedAt:    now,
		CreatedAt:     now,
		CreatedBy:     "test",
	}
}

func equalStrings(a []string, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
