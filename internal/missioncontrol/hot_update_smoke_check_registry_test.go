package missioncontrol

import (
	"strings"
	"testing"
	"time"
)

func TestHotUpdateSmokeCheckRecordRoundTripAndReadiness(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 5, 1, 9, 0, 0, 0, time.UTC)
	storeSmokeCheckGateFixture(t, root, now, "hot-update-smoke")

	missing, err := AssessHotUpdateSmokeReadiness(root, "hot-update-smoke")
	if err != nil {
		t.Fatalf("AssessHotUpdateSmokeReadiness(missing) error = %v", err)
	}
	if missing.Ready || missing.State != "missing" || !strings.Contains(missing.Reason, string(RejectionCodeV4SmokeCheckRequired)) {
		t.Fatalf("missing smoke assessment = %#v, want E_SMOKE_CHECK_REQUIRED not-ready", missing)
	}

	failed, changed, err := CreateHotUpdateSmokeCheckFromGate(root, " hot-update-smoke ", HotUpdateSmokeCheckStateFailed, now.Add(4*time.Minute), " operator ", now.Add(5*time.Minute), " deterministic smoke failed ")
	if err != nil {
		t.Fatalf("CreateHotUpdateSmokeCheckFromGate(failed) error = %v", err)
	}
	if !changed {
		t.Fatal("CreateHotUpdateSmokeCheckFromGate(failed) changed = false, want true")
	}
	if !strings.HasPrefix(failed.SmokeCheckID, "hot-update-smoke-check-hot-update-smoke-") {
		t.Fatalf("failed.SmokeCheckID = %q, want deterministic hot-update prefix", failed.SmokeCheckID)
	}
	if failed.Passed {
		t.Fatal("failed.Passed = true, want false")
	}

	failedAssessment, err := AssessHotUpdateSmokeReadiness(root, "hot-update-smoke")
	if err != nil {
		t.Fatalf("AssessHotUpdateSmokeReadiness(failed) error = %v", err)
	}
	if failedAssessment.Ready || failedAssessment.State != "failed" || failedAssessment.SelectedSmokeCheckID != failed.SmokeCheckID || !strings.Contains(failedAssessment.Reason, string(RejectionCodeV4SmokeCheckFailed)) {
		t.Fatalf("failed smoke assessment = %#v, want selected failed smoke blocker", failedAssessment)
	}

	passed, changed, err := CreateHotUpdateSmokeCheckFromGate(root, "hot-update-smoke", HotUpdateSmokeCheckStatePassed, now.Add(6*time.Minute), "operator", now.Add(7*time.Minute), "deterministic smoke passed")
	if err != nil {
		t.Fatalf("CreateHotUpdateSmokeCheckFromGate(passed) error = %v", err)
	}
	if !changed {
		t.Fatal("CreateHotUpdateSmokeCheckFromGate(passed) changed = false, want true")
	}
	if !passed.Passed {
		t.Fatal("passed.Passed = false, want true")
	}

	ready, err := AssessHotUpdateSmokeReadiness(root, "hot-update-smoke")
	if err != nil {
		t.Fatalf("AssessHotUpdateSmokeReadiness(ready) error = %v", err)
	}
	if !ready.Ready || ready.State != "ready" || ready.SelectedSmokeCheckID != passed.SmokeCheckID {
		t.Fatalf("ready smoke assessment = %#v, want selected passed smoke ready", ready)
	}

	replayed, changed, err := CreateHotUpdateSmokeCheckFromGate(root, "hot-update-smoke", HotUpdateSmokeCheckStatePassed, now.Add(6*time.Minute), "operator", now.Add(7*time.Minute), "deterministic smoke passed")
	if err != nil {
		t.Fatalf("CreateHotUpdateSmokeCheckFromGate(replay) error = %v", err)
	}
	if changed {
		t.Fatal("CreateHotUpdateSmokeCheckFromGate(replay) changed = true, want false")
	}
	if replayed.SmokeCheckID != passed.SmokeCheckID {
		t.Fatalf("replayed.SmokeCheckID = %q, want %q", replayed.SmokeCheckID, passed.SmokeCheckID)
	}

	gate, err := LoadHotUpdateGateRecord(root, "hot-update-smoke")
	if err != nil {
		t.Fatalf("LoadHotUpdateGateRecord() error = %v", err)
	}
	if got := strings.Join(gate.SmokeCheckRefs, ","); got != failed.SmokeCheckID+","+passed.SmokeCheckID {
		t.Fatalf("gate.SmokeCheckRefs = %#v, want failed then passed refs", gate.SmokeCheckRefs)
	}
}

func TestExecuteHotUpdateGateReloadApplyRequiresPassingSmokeEvidence(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC)
	storeSmokeCheckReloadingGateFixture(t, root, now, "hot-update-smoke-enforced")

	got, changed, err := ExecuteHotUpdateGateReloadApply(root, "hot-update-smoke-enforced", "operator", now.Add(7*time.Minute))
	if err == nil {
		t.Fatal("ExecuteHotUpdateGateReloadApply(missing smoke) error = nil, want blocker")
	}
	if changed || got.HotUpdateID != "" {
		t.Fatalf("ExecuteHotUpdateGateReloadApply(missing smoke) = %#v changed %t, want zero/false", got, changed)
	}
	if !strings.Contains(err.Error(), string(RejectionCodeV4SmokeCheckRequired)) {
		t.Fatalf("missing smoke error = %q, want E_SMOKE_CHECK_REQUIRED", err.Error())
	}

	if _, _, err := CreateHotUpdateSmokeCheckFromGate(root, "hot-update-smoke-enforced", HotUpdateSmokeCheckStateFailed, now.Add(8*time.Minute), "operator", now.Add(9*time.Minute), "post-reload smoke failed"); err != nil {
		t.Fatalf("CreateHotUpdateSmokeCheckFromGate(failed) error = %v", err)
	}
	got, changed, err = ExecuteHotUpdateGateReloadApply(root, "hot-update-smoke-enforced", "operator", now.Add(10*time.Minute))
	if err == nil {
		t.Fatal("ExecuteHotUpdateGateReloadApply(failed smoke) error = nil, want blocker")
	}
	if changed || got.HotUpdateID != "" {
		t.Fatalf("ExecuteHotUpdateGateReloadApply(failed smoke) = %#v changed %t, want zero/false", got, changed)
	}
	if !strings.Contains(err.Error(), string(RejectionCodeV4SmokeCheckFailed)) {
		t.Fatalf("failed smoke error = %q, want E_SMOKE_CHECK_FAILED", err.Error())
	}

	if _, _, err := CreateHotUpdateSmokeCheckFromGate(root, "hot-update-smoke-enforced", HotUpdateSmokeCheckStatePassed, now.Add(11*time.Minute), "operator", now.Add(12*time.Minute), "post-reload smoke passed"); err != nil {
		t.Fatalf("CreateHotUpdateSmokeCheckFromGate(passed) error = %v", err)
	}
	got, changed, err = ExecuteHotUpdateGateReloadApply(root, "hot-update-smoke-enforced", "operator", now.Add(13*time.Minute))
	if err != nil {
		t.Fatalf("ExecuteHotUpdateGateReloadApply(passed smoke) error = %v", err)
	}
	if !changed || got.State != HotUpdateGateStateReloadApplySucceeded {
		t.Fatalf("ExecuteHotUpdateGateReloadApply(passed smoke) = %#v changed %t, want succeeded true", got, changed)
	}
}

func TestCreateHotUpdateOutcomeFromSuccessfulGateRequiresSmokeEvidence(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 5, 1, 11, 0, 0, 0, time.UTC)
	storeSmokeCheckGateFixture(t, root, now, "hot-update-outcome-smoke")
	if err := StoreActiveRuntimePackPointer(root, ActiveRuntimePackPointer{
		ActivePackID:         "pack-candidate",
		PreviousActivePackID: "pack-base",
		LastKnownGoodPackID:  "pack-base",
		UpdatedAt:            now.Add(2 * time.Minute),
		UpdatedBy:            "operator",
		UpdateRecordRef:      hotUpdateGatePointerUpdateRecordRef("hot-update-outcome-smoke"),
		ReloadGeneration:     3,
	}); err != nil {
		t.Fatalf("StoreActiveRuntimePackPointer() error = %v", err)
	}
	gate, err := LoadHotUpdateGateRecord(root, "hot-update-outcome-smoke")
	if err != nil {
		t.Fatalf("LoadHotUpdateGateRecord() error = %v", err)
	}
	gate.State = HotUpdateGateStateReloadApplySucceeded
	gate.PhaseUpdatedAt = now.Add(3 * time.Minute)
	if err := StoreHotUpdateGateRecord(root, gate); err != nil {
		t.Fatalf("StoreHotUpdateGateRecord(success missing smoke) error = %v", err)
	}

	if _, changed, err := CreateHotUpdateOutcomeFromTerminalGate(root, "hot-update-outcome-smoke", "operator", now.Add(4*time.Minute)); err == nil {
		t.Fatal("CreateHotUpdateOutcomeFromTerminalGate(missing smoke) error = nil, want blocker")
	} else if changed {
		t.Fatal("CreateHotUpdateOutcomeFromTerminalGate(missing smoke) changed = true, want false")
	} else if !strings.Contains(err.Error(), string(RejectionCodeV4SmokeCheckRequired)) {
		t.Fatalf("missing outcome smoke error = %q, want E_SMOKE_CHECK_REQUIRED", err.Error())
	}
}

func storeSmokeCheckGateFixture(t *testing.T, root string, now time.Time, hotUpdateID string) {
	t.Helper()

	mustStoreRuntimePack(t, root, validRuntimePackRecord(now, func(record *RuntimePackRecord) {
		record.PackID = "pack-base"
	}))
	mustStoreRuntimePack(t, root, validRuntimePackRecord(now.Add(time.Minute), func(record *RuntimePackRecord) {
		record.PackID = "pack-candidate"
		record.ParentPackID = "pack-base"
		record.RollbackTargetPackID = "pack-base"
		record.MutableSurfaces = []string{"skills"}
		record.SurfaceClasses = []string{"class_1"}
		record.CompatibilityContractRef = "compat-v1"
	}))
	if err := StoreHotUpdateGateRecord(root, validHotUpdateGateRecord(now.Add(2*time.Minute), func(record *HotUpdateGateRecord) {
		record.HotUpdateID = hotUpdateID
		record.CandidatePackID = "pack-candidate"
		record.PreviousActivePackID = "pack-base"
		record.RollbackTargetPackID = "pack-base"
		record.SmokeCheckRefs = nil
	})); err != nil {
		t.Fatalf("StoreHotUpdateGateRecord() error = %v", err)
	}
}

func storeSmokeCheckReloadingGateFixture(t *testing.T, root string, now time.Time, hotUpdateID string) {
	t.Helper()

	storeSmokeCheckGateFixture(t, root, now, hotUpdateID)
	if err := StoreActiveRuntimePackPointer(root, ActiveRuntimePackPointer{
		ActivePackID:        "pack-base",
		LastKnownGoodPackID: "pack-base",
		UpdatedAt:           now.Add(3 * time.Minute),
		UpdatedBy:           "operator",
		UpdateRecordRef:     "bootstrap",
		ReloadGeneration:    2,
	}); err != nil {
		t.Fatalf("StoreActiveRuntimePackPointer() error = %v", err)
	}
	if err := StoreLastKnownGoodRuntimePackPointer(root, LastKnownGoodRuntimePackPointer{
		PackID:            "pack-base",
		Basis:             "holdout_pass",
		VerifiedAt:        now.Add(3 * time.Minute),
		VerifiedBy:        "operator",
		RollbackRecordRef: "bootstrap",
	}); err != nil {
		t.Fatalf("StoreLastKnownGoodRuntimePackPointer() error = %v", err)
	}
	if _, _, err := AdvanceHotUpdateGatePhase(root, hotUpdateID, HotUpdateGateStateValidated, "operator", now.Add(4*time.Minute)); err != nil {
		t.Fatalf("AdvanceHotUpdateGatePhase(validated) error = %v", err)
	}
	if _, _, err := AdvanceHotUpdateGatePhase(root, hotUpdateID, HotUpdateGateStateStaged, "operator", now.Add(5*time.Minute)); err != nil {
		t.Fatalf("AdvanceHotUpdateGatePhase(staged) error = %v", err)
	}
	if _, _, err := ExecuteHotUpdateGatePointerSwitch(root, hotUpdateID, "operator", now.Add(6*time.Minute)); err != nil {
		t.Fatalf("ExecuteHotUpdateGatePointerSwitch() error = %v", err)
	}
}
