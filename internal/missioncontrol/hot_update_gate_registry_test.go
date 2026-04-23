package missioncontrol

import (
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestHotUpdateGateRecordRoundTripAndList(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 22, 10, 0, 0, 0, time.UTC)
	mustStoreRuntimePack(t, root, validRuntimePackRecord(now, func(record *RuntimePackRecord) {
		record.PackID = "pack-prev"
	}))
	mustStoreRuntimePack(t, root, validRuntimePackRecord(now.Add(time.Minute), func(record *RuntimePackRecord) {
		record.PackID = "pack-candidate-b"
		record.RollbackTargetPackID = "pack-prev"
	}))
	mustStoreRuntimePack(t, root, validRuntimePackRecord(now.Add(2*time.Minute), func(record *RuntimePackRecord) {
		record.PackID = "pack-candidate-a"
		record.RollbackTargetPackID = "pack-prev"
	}))

	second := validHotUpdateGateRecord(now.Add(3*time.Minute), func(record *HotUpdateGateRecord) {
		record.HotUpdateID = "hot-update-b"
		record.CandidatePackID = "pack-candidate-b"
		record.Objective = "second gate"
	})
	if err := StoreHotUpdateGateRecord(root, second); err != nil {
		t.Fatalf("StoreHotUpdateGateRecord(hot-update-b) error = %v", err)
	}

	want := validHotUpdateGateRecord(now.Add(4*time.Minute), func(record *HotUpdateGateRecord) {
		record.HotUpdateID = " hot-update-a "
		record.Objective = " refresh skills "
		record.CandidatePackID = " pack-candidate-a "
		record.PreviousActivePackID = " pack-prev "
		record.RollbackTargetPackID = " pack-prev "
		record.TargetSurfaces = []string{" skills ", " prompts "}
		record.SurfaceClasses = []string{" class_1 ", " class_2 "}
		record.CompatibilityContractRef = " compat-v2 "
		record.EvalEvidenceRefs = []string{" eval/train ", " eval/holdout "}
		record.SmokeCheckRefs = []string{" smoke/run-1 "}
		record.CanaryRef = " canary-job-1 "
		record.ApprovalRef = " approval-1 "
		record.BudgetRef = " budget-1 "
		record.FailureReason = " staged "
	})
	if err := StoreHotUpdateGateRecord(root, want); err != nil {
		t.Fatalf("StoreHotUpdateGateRecord(hot-update-a) error = %v", err)
	}

	got, err := LoadHotUpdateGateRecord(root, "hot-update-a")
	if err != nil {
		t.Fatalf("LoadHotUpdateGateRecord() error = %v", err)
	}

	want.RecordVersion = StoreRecordVersion
	want.HotUpdateID = "hot-update-a"
	want.Objective = "refresh skills"
	want.CandidatePackID = "pack-candidate-a"
	want.PreviousActivePackID = "pack-prev"
	want.RollbackTargetPackID = "pack-prev"
	want.TargetSurfaces = []string{"skills", "prompts"}
	want.SurfaceClasses = []string{"class_1", "class_2"}
	want.CompatibilityContractRef = "compat-v2"
	want.EvalEvidenceRefs = []string{"eval/train", "eval/holdout"}
	want.SmokeCheckRefs = []string{"smoke/run-1"}
	want.CanaryRef = "canary-job-1"
	want.ApprovalRef = "approval-1"
	want.BudgetRef = "budget-1"
	want.PreparedAt = want.PreparedAt.UTC()
	want.FailureReason = "staged"
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("LoadHotUpdateGateRecord() = %#v, want %#v", got, want)
	}

	records, err := ListHotUpdateGateRecords(root)
	if err != nil {
		t.Fatalf("ListHotUpdateGateRecords() error = %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("ListHotUpdateGateRecords() len = %d, want 2", len(records))
	}
	if records[0].HotUpdateID != "hot-update-a" || records[1].HotUpdateID != "hot-update-b" {
		t.Fatalf("ListHotUpdateGateRecords() ids = [%q %q], want [hot-update-a hot-update-b]", records[0].HotUpdateID, records[1].HotUpdateID)
	}
}

func TestCandidateRuntimePackPointerRoundTripAndResolve(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 22, 11, 0, 0, 0, time.UTC)
	mustStoreRuntimePack(t, root, validRuntimePackRecord(now, func(record *RuntimePackRecord) {
		record.PackID = "pack-prev"
	}))
	candidate := validRuntimePackRecord(now.Add(time.Minute), func(record *RuntimePackRecord) {
		record.PackID = "pack-candidate"
		record.RollbackTargetPackID = "pack-prev"
	})
	mustStoreRuntimePack(t, root, candidate)

	if err := StoreHotUpdateGateRecord(root, validHotUpdateGateRecord(now.Add(2*time.Minute), func(record *HotUpdateGateRecord) {
		record.HotUpdateID = "hot-update-1"
		record.CandidatePackID = "pack-candidate"
		record.PreviousActivePackID = "pack-prev"
		record.RollbackTargetPackID = "pack-prev"
	})); err != nil {
		t.Fatalf("StoreHotUpdateGateRecord() error = %v", err)
	}

	want := CandidateRuntimePackPointer{
		HotUpdateID:     " hot-update-1 ",
		CandidatePackID: " pack-candidate ",
		UpdatedAt:       now.Add(3 * time.Minute),
		UpdatedBy:       " operator ",
	}
	if err := StoreCandidateRuntimePackPointer(root, want); err != nil {
		t.Fatalf("StoreCandidateRuntimePackPointer() error = %v", err)
	}

	got, err := LoadCandidateRuntimePackPointer(root)
	if err != nil {
		t.Fatalf("LoadCandidateRuntimePackPointer() error = %v", err)
	}

	want.RecordVersion = StoreRecordVersion
	want.HotUpdateID = "hot-update-1"
	want.CandidatePackID = "pack-candidate"
	want.UpdatedAt = want.UpdatedAt.UTC()
	want.UpdatedBy = "operator"
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("LoadCandidateRuntimePackPointer() = %#v, want %#v", got, want)
	}

	resolved, err := ResolveCandidateRuntimePackRecord(root)
	if err != nil {
		t.Fatalf("ResolveCandidateRuntimePackRecord() error = %v", err)
	}
	candidate.RecordVersion = StoreRecordVersion
	candidate = NormalizeRuntimePackRecord(candidate)
	if !reflect.DeepEqual(resolved, candidate) {
		t.Fatalf("ResolveCandidateRuntimePackRecord() = %#v, want %#v", resolved, candidate)
	}
}

func TestHotUpdateGateReplayIsIdempotent(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 22, 12, 0, 0, 0, time.UTC)
	mustStoreRuntimePack(t, root, validRuntimePackRecord(now, func(record *RuntimePackRecord) {
		record.PackID = "pack-prev"
	}))
	mustStoreRuntimePack(t, root, validRuntimePackRecord(now.Add(time.Minute), func(record *RuntimePackRecord) {
		record.PackID = "pack-candidate"
		record.RollbackTargetPackID = "pack-prev"
	}))

	record := validHotUpdateGateRecord(now.Add(2*time.Minute), func(gate *HotUpdateGateRecord) {
		gate.HotUpdateID = "hot-update-replay"
		gate.CandidatePackID = "pack-candidate"
		gate.PreviousActivePackID = "pack-prev"
		gate.RollbackTargetPackID = "pack-prev"
	})
	if err := StoreHotUpdateGateRecord(root, record); err != nil {
		t.Fatalf("StoreHotUpdateGateRecord(first) error = %v", err)
	}
	firstBytes, err := os.ReadFile(StoreHotUpdateGatePath(root, record.HotUpdateID))
	if err != nil {
		t.Fatalf("ReadFile(first) error = %v", err)
	}

	if err := StoreHotUpdateGateRecord(root, record); err != nil {
		t.Fatalf("StoreHotUpdateGateRecord(replay) error = %v", err)
	}
	secondBytes, err := os.ReadFile(StoreHotUpdateGatePath(root, record.HotUpdateID))
	if err != nil {
		t.Fatalf("ReadFile(second) error = %v", err)
	}

	if string(firstBytes) != string(secondBytes) {
		t.Fatalf("hot-update gate file changed on idempotent replay\nfirst:\n%s\nsecond:\n%s", string(firstBytes), string(secondBytes))
	}
}

func TestEnsureHotUpdateGateRecordFromCandidateCreatesOrSelectsExistingMatch(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 22, 12, 30, 0, 0, time.UTC)
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
	if err := StoreActiveRuntimePackPointer(root, ActiveRuntimePackPointer{
		ActivePackID:         "pack-base",
		PreviousActivePackID: "",
		LastKnownGoodPackID:  "pack-base",
		UpdatedAt:            now.Add(2 * time.Minute),
		UpdatedBy:            "operator",
		UpdateRecordRef:      "bootstrap",
		ReloadGeneration:     2,
	}); err != nil {
		t.Fatalf("StoreActiveRuntimePackPointer() error = %v", err)
	}

	first, created, err := EnsureHotUpdateGateRecordFromCandidate(root, "hot-update-select", "pack-candidate", "operator", now.Add(3*time.Minute))
	if err != nil {
		t.Fatalf("EnsureHotUpdateGateRecordFromCandidate(first) error = %v", err)
	}
	if !created {
		t.Fatal("EnsureHotUpdateGateRecordFromCandidate(first) created = false, want true")
	}
	if first.CandidatePackID != "pack-candidate" {
		t.Fatalf("first.CandidatePackID = %q, want pack-candidate", first.CandidatePackID)
	}
	if first.PreviousActivePackID != "pack-base" {
		t.Fatalf("first.PreviousActivePackID = %q, want pack-base", first.PreviousActivePackID)
	}
	if first.RollbackTargetPackID != "pack-base" {
		t.Fatalf("first.RollbackTargetPackID = %q, want pack-base", first.RollbackTargetPackID)
	}
	if got := strings.Join(first.TargetSurfaces, ","); got != "skills" {
		t.Fatalf("first.TargetSurfaces = %#v, want [skills]", first.TargetSurfaces)
	}
	if first.ReloadMode != HotUpdateReloadModeSkillReload {
		t.Fatalf("first.ReloadMode = %q, want skill_reload", first.ReloadMode)
	}
	if first.State != HotUpdateGateStatePrepared {
		t.Fatalf("first.State = %q, want prepared", first.State)
	}
	if first.Decision != HotUpdateGateDecisionKeepStaged {
		t.Fatalf("first.Decision = %q, want keep_staged", first.Decision)
	}

	second, created, err := EnsureHotUpdateGateRecordFromCandidate(root, "hot-update-select", "pack-candidate", "other-operator", now.Add(5*time.Minute))
	if err != nil {
		t.Fatalf("EnsureHotUpdateGateRecordFromCandidate(second) error = %v", err)
	}
	if created {
		t.Fatal("EnsureHotUpdateGateRecordFromCandidate(second) created = true, want false")
	}
	if !reflect.DeepEqual(second, first) {
		t.Fatalf("EnsureHotUpdateGateRecordFromCandidate(second) = %#v, want %#v", second, first)
	}
}

func TestEnsureHotUpdateGateRecordFromCandidateRejectsMismatchedExistingCandidate(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 22, 12, 45, 0, 0, time.UTC)
	mustStoreRuntimePack(t, root, validRuntimePackRecord(now, func(record *RuntimePackRecord) {
		record.PackID = "pack-base"
	}))
	mustStoreRuntimePack(t, root, validRuntimePackRecord(now.Add(time.Minute), func(record *RuntimePackRecord) {
		record.PackID = "pack-candidate-a"
		record.ParentPackID = "pack-base"
		record.RollbackTargetPackID = "pack-base"
	}))
	mustStoreRuntimePack(t, root, validRuntimePackRecord(now.Add(2*time.Minute), func(record *RuntimePackRecord) {
		record.PackID = "pack-candidate-b"
		record.ParentPackID = "pack-base"
		record.RollbackTargetPackID = "pack-base"
	}))
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

	first, created, err := EnsureHotUpdateGateRecordFromCandidate(root, "hot-update-mismatch", "pack-candidate-a", "operator", now.Add(4*time.Minute))
	if err != nil {
		t.Fatalf("EnsureHotUpdateGateRecordFromCandidate(first) error = %v", err)
	}
	if !created {
		t.Fatal("EnsureHotUpdateGateRecordFromCandidate(first) created = false, want true")
	}
	if first.CandidatePackID != "pack-candidate-a" {
		t.Fatalf("first.CandidatePackID = %q, want pack-candidate-a", first.CandidatePackID)
	}

	_, created, err = EnsureHotUpdateGateRecordFromCandidate(root, "hot-update-mismatch", "pack-candidate-b", "operator", now.Add(5*time.Minute))
	if err == nil {
		t.Fatal("EnsureHotUpdateGateRecordFromCandidate() error = nil, want mismatched candidate rejection")
	}
	if created {
		t.Fatal("EnsureHotUpdateGateRecordFromCandidate() created = true, want false")
	}
	if !strings.Contains(err.Error(), `candidate_pack_id "pack-candidate-a" does not match requested candidate_pack_id "pack-candidate-b"`) {
		t.Fatalf("EnsureHotUpdateGateRecordFromCandidate() error = %q, want mismatched candidate context", err.Error())
	}
}

func TestAdvanceHotUpdateGatePhaseValidProgressionAndPreservesActiveRuntimePackPointer(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 22, 13, 15, 0, 0, time.UTC)
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
	wantPointer := ActiveRuntimePackPointer{
		ActivePackID:        "pack-base",
		LastKnownGoodPackID: "pack-base",
		UpdatedAt:           now.Add(2 * time.Minute),
		UpdatedBy:           "operator",
		UpdateRecordRef:     "bootstrap",
		ReloadGeneration:    2,
	}
	if err := StoreActiveRuntimePackPointer(root, wantPointer); err != nil {
		t.Fatalf("StoreActiveRuntimePackPointer() error = %v", err)
	}
	wantPointer.RecordVersion = StoreRecordVersion

	if _, _, err := EnsureHotUpdateGateRecordFromCandidate(root, "hot-update-progress", "pack-candidate", "operator", now.Add(3*time.Minute)); err != nil {
		t.Fatalf("EnsureHotUpdateGateRecordFromCandidate() error = %v", err)
	}

	validated, changed, err := AdvanceHotUpdateGatePhase(root, "hot-update-progress", HotUpdateGateStateValidated, "reviewer", now.Add(4*time.Minute))
	if err != nil {
		t.Fatalf("AdvanceHotUpdateGatePhase(validated) error = %v", err)
	}
	if !changed {
		t.Fatal("AdvanceHotUpdateGatePhase(validated) changed = false, want true")
	}
	if validated.State != HotUpdateGateStateValidated {
		t.Fatalf("validated.State = %q, want validated", validated.State)
	}
	if validated.PhaseUpdatedAt != now.Add(4*time.Minute).UTC() {
		t.Fatalf("validated.PhaseUpdatedAt = %v, want %v", validated.PhaseUpdatedAt, now.Add(4*time.Minute).UTC())
	}
	if validated.PhaseUpdatedBy != "reviewer" {
		t.Fatalf("validated.PhaseUpdatedBy = %q, want reviewer", validated.PhaseUpdatedBy)
	}

	staged, changed, err := AdvanceHotUpdateGatePhase(root, "hot-update-progress", HotUpdateGateStateStaged, "operator", now.Add(5*time.Minute))
	if err != nil {
		t.Fatalf("AdvanceHotUpdateGatePhase(staged) error = %v", err)
	}
	if !changed {
		t.Fatal("AdvanceHotUpdateGatePhase(staged) changed = false, want true")
	}
	if staged.State != HotUpdateGateStateStaged {
		t.Fatalf("staged.State = %q, want staged", staged.State)
	}
	if staged.PhaseUpdatedAt != now.Add(5*time.Minute).UTC() {
		t.Fatalf("staged.PhaseUpdatedAt = %v, want %v", staged.PhaseUpdatedAt, now.Add(5*time.Minute).UTC())
	}
	if staged.PhaseUpdatedBy != "operator" {
		t.Fatalf("staged.PhaseUpdatedBy = %q, want operator", staged.PhaseUpdatedBy)
	}

	gotPointer, err := LoadActiveRuntimePackPointer(root)
	if err != nil {
		t.Fatalf("LoadActiveRuntimePackPointer() error = %v", err)
	}
	if gotPointer != wantPointer {
		t.Fatalf("LoadActiveRuntimePackPointer() = %#v, want %#v", gotPointer, wantPointer)
	}
}

func TestAdvanceHotUpdateGatePhaseIsIdempotentForSamePhase(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 22, 13, 30, 0, 0, time.UTC)
	mustStoreRuntimePack(t, root, validRuntimePackRecord(now, func(record *RuntimePackRecord) {
		record.PackID = "pack-base"
	}))
	mustStoreRuntimePack(t, root, validRuntimePackRecord(now.Add(time.Minute), func(record *RuntimePackRecord) {
		record.PackID = "pack-candidate"
		record.ParentPackID = "pack-base"
		record.RollbackTargetPackID = "pack-base"
	}))
	if err := StoreActiveRuntimePackPointer(root, ActiveRuntimePackPointer{
		ActivePackID:        "pack-base",
		LastKnownGoodPackID: "pack-base",
		UpdatedAt:           now.Add(2 * time.Minute),
		UpdatedBy:           "operator",
		UpdateRecordRef:     "bootstrap",
		ReloadGeneration:    2,
	}); err != nil {
		t.Fatalf("StoreActiveRuntimePackPointer() error = %v", err)
	}

	if _, _, err := EnsureHotUpdateGateRecordFromCandidate(root, "hot-update-idempotent", "pack-candidate", "operator", now.Add(3*time.Minute)); err != nil {
		t.Fatalf("EnsureHotUpdateGateRecordFromCandidate() error = %v", err)
	}

	first, changed, err := AdvanceHotUpdateGatePhase(root, "hot-update-idempotent", HotUpdateGateStateValidated, "reviewer", now.Add(4*time.Minute))
	if err != nil {
		t.Fatalf("AdvanceHotUpdateGatePhase(first) error = %v", err)
	}
	if !changed {
		t.Fatal("AdvanceHotUpdateGatePhase(first) changed = false, want true")
	}

	second, changed, err := AdvanceHotUpdateGatePhase(root, "hot-update-idempotent", HotUpdateGateStateValidated, "later-reviewer", now.Add(5*time.Minute))
	if err != nil {
		t.Fatalf("AdvanceHotUpdateGatePhase(second) error = %v", err)
	}
	if changed {
		t.Fatal("AdvanceHotUpdateGatePhase(second) changed = true, want false")
	}
	if !reflect.DeepEqual(second, first) {
		t.Fatalf("AdvanceHotUpdateGatePhase(second) = %#v, want %#v", second, first)
	}
}

func TestAdvanceHotUpdateGatePhaseRejectsSkippedRegressiveAndInvalidStartingTransitions(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 22, 13, 45, 0, 0, time.UTC)
	mustStoreRuntimePack(t, root, validRuntimePackRecord(now, func(record *RuntimePackRecord) {
		record.PackID = "pack-base"
	}))
	mustStoreRuntimePack(t, root, validRuntimePackRecord(now.Add(time.Minute), func(record *RuntimePackRecord) {
		record.PackID = "pack-candidate"
		record.ParentPackID = "pack-base"
		record.RollbackTargetPackID = "pack-base"
	}))
	if err := StoreActiveRuntimePackPointer(root, ActiveRuntimePackPointer{
		ActivePackID:        "pack-base",
		LastKnownGoodPackID: "pack-base",
		UpdatedAt:           now.Add(2 * time.Minute),
		UpdatedBy:           "operator",
		UpdateRecordRef:     "bootstrap",
		ReloadGeneration:    2,
	}); err != nil {
		t.Fatalf("StoreActiveRuntimePackPointer() error = %v", err)
	}

	if _, _, err := EnsureHotUpdateGateRecordFromCandidate(root, "hot-update-invalid", "pack-candidate", "operator", now.Add(3*time.Minute)); err != nil {
		t.Fatalf("EnsureHotUpdateGateRecordFromCandidate() error = %v", err)
	}

	got, changed, err := AdvanceHotUpdateGatePhase(root, "hot-update-invalid", HotUpdateGateStateStaged, "operator", now.Add(4*time.Minute))
	if err == nil {
		t.Fatal("AdvanceHotUpdateGatePhase(prepared->staged) error = nil, want skipped transition rejection")
	}
	if changed {
		t.Fatal("AdvanceHotUpdateGatePhase(prepared->staged) changed = true, want false")
	}
	if !reflect.DeepEqual(got, HotUpdateGateRecord{}) {
		t.Fatalf("AdvanceHotUpdateGatePhase(prepared->staged) record = %#v, want zero value", got)
	}
	if !strings.Contains(err.Error(), `transition "prepared" -> "staged" is invalid`) {
		t.Fatalf("AdvanceHotUpdateGatePhase(prepared->staged) error = %q, want invalid transition context", err.Error())
	}

	if _, _, err := AdvanceHotUpdateGatePhase(root, "hot-update-invalid", HotUpdateGateStateValidated, "operator", now.Add(5*time.Minute)); err != nil {
		t.Fatalf("AdvanceHotUpdateGatePhase(validated) error = %v", err)
	}
	if _, _, err := AdvanceHotUpdateGatePhase(root, "hot-update-invalid", HotUpdateGateStateStaged, "operator", now.Add(6*time.Minute)); err != nil {
		t.Fatalf("AdvanceHotUpdateGatePhase(staged) error = %v", err)
	}
	got, changed, err = AdvanceHotUpdateGatePhase(root, "hot-update-invalid", HotUpdateGateStateValidated, "operator", now.Add(7*time.Minute))
	if err == nil {
		t.Fatal("AdvanceHotUpdateGatePhase(staged->validated) error = nil, want regressive transition rejection")
	}
	if changed {
		t.Fatal("AdvanceHotUpdateGatePhase(staged->validated) changed = true, want false")
	}
	if !reflect.DeepEqual(got, HotUpdateGateRecord{}) {
		t.Fatalf("AdvanceHotUpdateGatePhase(staged->validated) record = %#v, want zero value", got)
	}
	if !strings.Contains(err.Error(), `transition "staged" -> "validated" is invalid`) {
		t.Fatalf("AdvanceHotUpdateGatePhase(staged->validated) error = %q, want invalid transition context", err.Error())
	}

	if err := StoreHotUpdateGateRecord(root, validHotUpdateGateRecord(now.Add(8*time.Minute), func(record *HotUpdateGateRecord) {
		record.HotUpdateID = "hot-update-start-invalid"
		record.CandidatePackID = "pack-candidate"
		record.PreviousActivePackID = "pack-base"
		record.RollbackTargetPackID = "pack-base"
		record.State = HotUpdateGateStateQuiescing
		record.PhaseUpdatedAt = now.Add(8 * time.Minute)
		record.PhaseUpdatedBy = "operator"
	})); err != nil {
		t.Fatalf("StoreHotUpdateGateRecord(quiescing) error = %v", err)
	}

	got, changed, err = AdvanceHotUpdateGatePhase(root, "hot-update-start-invalid", HotUpdateGateStateValidated, "operator", now.Add(9*time.Minute))
	if err == nil {
		t.Fatal("AdvanceHotUpdateGatePhase(quiescing->validated) error = nil, want invalid starting state rejection")
	}
	if changed {
		t.Fatal("AdvanceHotUpdateGatePhase(quiescing->validated) changed = true, want false")
	}
	if !reflect.DeepEqual(got, HotUpdateGateRecord{}) {
		t.Fatalf("AdvanceHotUpdateGatePhase(quiescing->validated) record = %#v, want zero value", got)
	}
	if !strings.Contains(err.Error(), `state "quiescing" cannot advance via phase control`) {
		t.Fatalf("AdvanceHotUpdateGatePhase(quiescing->validated) error = %q, want invalid starting state context", err.Error())
	}
}

func TestExecuteHotUpdateGatePointerSwitchSwitchesActivePointerAndIsReplaySafe(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 22, 14, 15, 0, 0, time.UTC)
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
	if err := StoreActiveRuntimePackPointer(root, ActiveRuntimePackPointer{
		ActivePackID:        "pack-base",
		LastKnownGoodPackID: "pack-base",
		UpdatedAt:           now.Add(2 * time.Minute),
		UpdatedBy:           "operator",
		UpdateRecordRef:     "bootstrap",
		ReloadGeneration:    2,
	}); err != nil {
		t.Fatalf("StoreActiveRuntimePackPointer() error = %v", err)
	}
	if err := StoreLastKnownGoodRuntimePackPointer(root, LastKnownGoodRuntimePackPointer{
		PackID:            "pack-base",
		Basis:             "holdout_pass",
		VerifiedAt:        now.Add(2 * time.Minute),
		VerifiedBy:        "operator",
		RollbackRecordRef: "bootstrap",
	}); err != nil {
		t.Fatalf("StoreLastKnownGoodRuntimePackPointer() error = %v", err)
	}

	if _, _, err := EnsureHotUpdateGateRecordFromCandidate(root, "hot-update-execute", "pack-candidate", "operator", now.Add(3*time.Minute)); err != nil {
		t.Fatalf("EnsureHotUpdateGateRecordFromCandidate() error = %v", err)
	}
	if _, _, err := AdvanceHotUpdateGatePhase(root, "hot-update-execute", HotUpdateGateStateValidated, "operator", now.Add(4*time.Minute)); err != nil {
		t.Fatalf("AdvanceHotUpdateGatePhase(validated) error = %v", err)
	}
	if _, _, err := AdvanceHotUpdateGatePhase(root, "hot-update-execute", HotUpdateGateStateStaged, "operator", now.Add(5*time.Minute)); err != nil {
		t.Fatalf("AdvanceHotUpdateGatePhase(staged) error = %v", err)
	}

	beforeLKGBytes, err := os.ReadFile(StoreLastKnownGoodRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(last-known-good before) error = %v", err)
	}

	executed, changed, err := ExecuteHotUpdateGatePointerSwitch(root, "hot-update-execute", "operator", now.Add(6*time.Minute))
	if err != nil {
		t.Fatalf("ExecuteHotUpdateGatePointerSwitch(first) error = %v", err)
	}
	if !changed {
		t.Fatal("ExecuteHotUpdateGatePointerSwitch(first) changed = false, want true")
	}
	if executed.State != HotUpdateGateStateReloading {
		t.Fatalf("executed.State = %q, want reloading", executed.State)
	}
	if executed.PhaseUpdatedAt != now.Add(6*time.Minute).UTC() {
		t.Fatalf("executed.PhaseUpdatedAt = %v, want %v", executed.PhaseUpdatedAt, now.Add(6*time.Minute).UTC())
	}
	if executed.PhaseUpdatedBy != "operator" {
		t.Fatalf("executed.PhaseUpdatedBy = %q, want operator", executed.PhaseUpdatedBy)
	}

	activePointer, err := LoadActiveRuntimePackPointer(root)
	if err != nil {
		t.Fatalf("LoadActiveRuntimePackPointer() error = %v", err)
	}
	if activePointer.ActivePackID != "pack-candidate" {
		t.Fatalf("activePointer.ActivePackID = %q, want pack-candidate", activePointer.ActivePackID)
	}
	if activePointer.PreviousActivePackID != "pack-base" {
		t.Fatalf("activePointer.PreviousActivePackID = %q, want pack-base", activePointer.PreviousActivePackID)
	}
	if activePointer.UpdateRecordRef != "hot_update:hot-update-execute" {
		t.Fatalf("activePointer.UpdateRecordRef = %q, want hot_update:hot-update-execute", activePointer.UpdateRecordRef)
	}
	if activePointer.ReloadGeneration != 3 {
		t.Fatalf("activePointer.ReloadGeneration = %d, want 3", activePointer.ReloadGeneration)
	}

	second, changed, err := ExecuteHotUpdateGatePointerSwitch(root, "hot-update-execute", "operator", now.Add(7*time.Minute))
	if err != nil {
		t.Fatalf("ExecuteHotUpdateGatePointerSwitch(second) error = %v", err)
	}
	if changed {
		t.Fatal("ExecuteHotUpdateGatePointerSwitch(second) changed = true, want false")
	}
	if !reflect.DeepEqual(second, executed) {
		t.Fatalf("ExecuteHotUpdateGatePointerSwitch(second) = %#v, want %#v", second, executed)
	}

	replayedPointer, err := LoadActiveRuntimePackPointer(root)
	if err != nil {
		t.Fatalf("LoadActiveRuntimePackPointer(replayed) error = %v", err)
	}
	if replayedPointer.ReloadGeneration != 3 {
		t.Fatalf("replayedPointer.ReloadGeneration = %d, want 3", replayedPointer.ReloadGeneration)
	}

	afterLKGBytes, err := os.ReadFile(StoreLastKnownGoodRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(last-known-good after) error = %v", err)
	}
	if string(beforeLKGBytes) != string(afterLKGBytes) {
		t.Fatalf("last-known-good pointer changed during hot-update execute\nbefore:\n%s\nafter:\n%s", string(beforeLKGBytes), string(afterLKGBytes))
	}
}

func TestExecuteHotUpdateGatePointerSwitchRejectsInvalidStateAndBrokenLinkageWithoutPointerMutation(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 22, 14, 30, 0, 0, time.UTC)
	mustStoreRuntimePack(t, root, validRuntimePackRecord(now, func(record *RuntimePackRecord) {
		record.PackID = "pack-base"
	}))
	mustStoreRuntimePack(t, root, validRuntimePackRecord(now.Add(time.Minute), func(record *RuntimePackRecord) {
		record.PackID = "pack-candidate"
		record.ParentPackID = "pack-base"
		record.RollbackTargetPackID = "pack-base"
	}))
	if err := StoreActiveRuntimePackPointer(root, ActiveRuntimePackPointer{
		ActivePackID:        "pack-base",
		LastKnownGoodPackID: "pack-base",
		UpdatedAt:           now.Add(2 * time.Minute),
		UpdatedBy:           "operator",
		UpdateRecordRef:     "bootstrap",
		ReloadGeneration:    2,
	}); err != nil {
		t.Fatalf("StoreActiveRuntimePackPointer() error = %v", err)
	}

	beforePointerBytes, err := os.ReadFile(StoreActiveRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(active pointer before) error = %v", err)
	}

	if err := StoreHotUpdateGateRecord(root, validHotUpdateGateRecord(now.Add(3*time.Minute), func(record *HotUpdateGateRecord) {
		record.HotUpdateID = "hot-update-not-staged"
		record.CandidatePackID = "pack-candidate"
		record.PreviousActivePackID = "pack-base"
		record.RollbackTargetPackID = "pack-base"
		record.State = HotUpdateGateStateValidated
		record.PhaseUpdatedAt = now.Add(3 * time.Minute)
		record.PhaseUpdatedBy = "operator"
	})); err != nil {
		t.Fatalf("StoreHotUpdateGateRecord(validated) error = %v", err)
	}

	got, changed, err := ExecuteHotUpdateGatePointerSwitch(root, "hot-update-not-staged", "operator", now.Add(4*time.Minute))
	if err == nil {
		t.Fatal("ExecuteHotUpdateGatePointerSwitch(validated) error = nil, want invalid state rejection")
	}
	if changed {
		t.Fatal("ExecuteHotUpdateGatePointerSwitch(validated) changed = true, want false")
	}
	if !reflect.DeepEqual(got, HotUpdateGateRecord{}) {
		t.Fatalf("ExecuteHotUpdateGatePointerSwitch(validated) record = %#v, want zero value", got)
	}
	if !strings.Contains(err.Error(), `state "validated" does not permit pointer switch execution`) {
		t.Fatalf("ExecuteHotUpdateGatePointerSwitch(validated) error = %q, want invalid state context", err.Error())
	}

	if err := StoreHotUpdateGateRecord(root, validHotUpdateGateRecord(now.Add(5*time.Minute), func(record *HotUpdateGateRecord) {
		record.HotUpdateID = "hot-update-bad-linkage"
		record.CandidatePackID = "pack-candidate"
		record.PreviousActivePackID = "pack-base"
		record.RollbackTargetPackID = "pack-other"
		record.State = HotUpdateGateStateStaged
		record.PhaseUpdatedAt = now.Add(5 * time.Minute)
		record.PhaseUpdatedBy = "operator"
	})); err == nil {
		t.Fatal("StoreHotUpdateGateRecord(bad linkage) error = nil, want missing rollback target rejection")
	}

	mustStoreRuntimePack(t, root, validRuntimePackRecord(now.Add(6*time.Minute), func(record *RuntimePackRecord) {
		record.PackID = "pack-other"
	}))
	if err := StoreHotUpdateGateRecord(root, validHotUpdateGateRecord(now.Add(7*time.Minute), func(record *HotUpdateGateRecord) {
		record.HotUpdateID = "hot-update-mismatch"
		record.CandidatePackID = "pack-candidate"
		record.PreviousActivePackID = "pack-base"
		record.RollbackTargetPackID = "pack-other"
		record.State = HotUpdateGateStateStaged
		record.PhaseUpdatedAt = now.Add(7 * time.Minute)
		record.PhaseUpdatedBy = "operator"
	})); err != nil {
		t.Fatalf("StoreHotUpdateGateRecord(mismatch) error = %v", err)
	}

	got, changed, err = ExecuteHotUpdateGatePointerSwitch(root, "hot-update-mismatch", "operator", now.Add(8*time.Minute))
	if err == nil {
		t.Fatal("ExecuteHotUpdateGatePointerSwitch(mismatch) error = nil, want linkage rejection")
	}
	if changed {
		t.Fatal("ExecuteHotUpdateGatePointerSwitch(mismatch) changed = true, want false")
	}
	if !reflect.DeepEqual(got, HotUpdateGateRecord{}) {
		t.Fatalf("ExecuteHotUpdateGatePointerSwitch(mismatch) record = %#v, want zero value", got)
	}
	if !strings.Contains(err.Error(), `rollback_target_pack_id "pack-other" does not match candidate rollback_target_pack_id "pack-base"`) {
		t.Fatalf("ExecuteHotUpdateGatePointerSwitch(mismatch) error = %q, want rollback linkage context", err.Error())
	}

	afterPointerBytes, err := os.ReadFile(StoreActiveRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(active pointer after) error = %v", err)
	}
	if string(beforePointerBytes) != string(afterPointerBytes) {
		t.Fatalf("active pointer changed on rejected hot-update execute\nbefore:\n%s\nafter:\n%s", string(beforePointerBytes), string(afterPointerBytes))
	}
}

func TestHotUpdateGateValidationFailsClosed(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 22, 13, 0, 0, 0, time.UTC)
	mustStoreRuntimePack(t, root, validRuntimePackRecord(now, func(record *RuntimePackRecord) {
		record.PackID = "pack-prev"
	}))
	mustStoreRuntimePack(t, root, validRuntimePackRecord(now.Add(time.Minute), func(record *RuntimePackRecord) {
		record.PackID = "pack-candidate"
		record.RollbackTargetPackID = "pack-prev"
	}))

	tests := []struct {
		name string
		run  func() error
		want string
	}{
		{
			name: "missing hot update id",
			run: func() error {
				return StoreHotUpdateGateRecord(root, validHotUpdateGateRecord(now.Add(2*time.Minute), func(record *HotUpdateGateRecord) {
					record.HotUpdateID = " "
					record.CandidatePackID = "pack-candidate"
					record.PreviousActivePackID = "pack-prev"
					record.RollbackTargetPackID = "pack-prev"
				}))
			},
			want: "mission store hot-update gate hot_update_id is required",
		},
		{
			name: "invalid reload mode",
			run: func() error {
				return StoreHotUpdateGateRecord(root, validHotUpdateGateRecord(now.Add(2*time.Minute), func(record *HotUpdateGateRecord) {
					record.CandidatePackID = "pack-candidate"
					record.PreviousActivePackID = "pack-prev"
					record.RollbackTargetPackID = "pack-prev"
					record.ReloadMode = HotUpdateReloadMode("bad_reload")
				}))
			},
			want: `mission store hot-update gate reload_mode "bad_reload" is invalid`,
		},
		{
			name: "invalid state",
			run: func() error {
				return StoreHotUpdateGateRecord(root, validHotUpdateGateRecord(now.Add(2*time.Minute), func(record *HotUpdateGateRecord) {
					record.CandidatePackID = "pack-candidate"
					record.PreviousActivePackID = "pack-prev"
					record.RollbackTargetPackID = "pack-prev"
					record.State = HotUpdateGateState("bad_state")
				}))
			},
			want: `mission store hot-update gate state "bad_state" is invalid`,
		},
		{
			name: "invalid decision",
			run: func() error {
				return StoreHotUpdateGateRecord(root, validHotUpdateGateRecord(now.Add(2*time.Minute), func(record *HotUpdateGateRecord) {
					record.CandidatePackID = "pack-candidate"
					record.PreviousActivePackID = "pack-prev"
					record.RollbackTargetPackID = "pack-prev"
					record.Decision = HotUpdateGateDecision("bad_decision")
				}))
			},
			want: `mission store hot-update gate decision "bad_decision" is invalid`,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := tc.run()
			if err == nil {
				t.Fatal("StoreHotUpdateGateRecord() error = nil, want fail-closed rejection")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("StoreHotUpdateGateRecord() error = %q, want substring %q", err.Error(), tc.want)
			}
		})
	}
}

func TestHotUpdateGateRejectsMissingRefs(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 22, 14, 0, 0, 0, time.UTC)

	err := StoreHotUpdateGateRecord(root, validHotUpdateGateRecord(now, func(record *HotUpdateGateRecord) {
		record.CandidatePackID = "missing-candidate"
		record.PreviousActivePackID = "missing-prev"
		record.RollbackTargetPackID = "missing-rollback"
	}))
	if err == nil {
		t.Fatal("StoreHotUpdateGateRecord() error = nil, want missing pack rejection")
	}
	if !strings.Contains(err.Error(), ErrRuntimePackRecordNotFound.Error()) {
		t.Fatalf("StoreHotUpdateGateRecord() error = %q, want missing pack rejection", err.Error())
	}

	mustStoreRuntimePack(t, root, validRuntimePackRecord(now.Add(time.Minute), func(record *RuntimePackRecord) {
		record.PackID = "pack-prev"
	}))
	mustStoreRuntimePack(t, root, validRuntimePackRecord(now.Add(2*time.Minute), func(record *RuntimePackRecord) {
		record.PackID = "pack-candidate"
		record.RollbackTargetPackID = "pack-prev"
	}))

	err = StoreCandidateRuntimePackPointer(root, CandidateRuntimePackPointer{
		HotUpdateID:     "missing-gate",
		CandidatePackID: "pack-candidate",
		UpdatedAt:       now.Add(3 * time.Minute),
		UpdatedBy:       "system",
	})
	if err == nil {
		t.Fatal("StoreCandidateRuntimePackPointer() error = nil, want missing gate rejection")
	}
	if !strings.Contains(err.Error(), ErrHotUpdateGateRecordNotFound.Error()) {
		t.Fatalf("StoreCandidateRuntimePackPointer() error = %q, want missing gate rejection", err.Error())
	}
}

func TestLoadHotUpdateGateAndCandidatePointerNotFound(t *testing.T) {
	t.Parallel()

	root := t.TempDir()

	if _, err := LoadHotUpdateGateRecord(root, "missing-gate"); !errors.Is(err, ErrHotUpdateGateRecordNotFound) {
		t.Fatalf("LoadHotUpdateGateRecord() error = %v, want %v", err, ErrHotUpdateGateRecordNotFound)
	}
	if _, err := LoadCandidateRuntimePackPointer(root); !errors.Is(err, ErrCandidateRuntimePackPointerNotFound) {
		t.Fatalf("LoadCandidateRuntimePackPointer() error = %v, want %v", err, ErrCandidateRuntimePackPointerNotFound)
	}
}

func validHotUpdateGateRecord(now time.Time, mutate func(*HotUpdateGateRecord)) HotUpdateGateRecord {
	record := HotUpdateGateRecord{
		HotUpdateID:              "hot-update-root",
		Objective:                "stage runtime pack candidate",
		CandidatePackID:          "pack-candidate",
		PreviousActivePackID:     "pack-prev",
		RollbackTargetPackID:     "pack-prev",
		TargetSurfaces:           []string{"skills"},
		SurfaceClasses:           []string{"class_1"},
		ReloadMode:               HotUpdateReloadModeSkillReload,
		CompatibilityContractRef: "compat-v1",
		EvalEvidenceRefs:         []string{"eval/train"},
		SmokeCheckRefs:           []string{"smoke/run-1"},
		CanaryRef:                "",
		ApprovalRef:              "",
		BudgetRef:                "",
		PreparedAt:               now,
		PhaseUpdatedAt:           now,
		PhaseUpdatedBy:           "operator",
		State:                    HotUpdateGateStatePrepared,
		Decision:                 HotUpdateGateDecisionKeepStaged,
		FailureReason:            "",
	}
	if mutate != nil {
		mutate(&record)
	}
	return record
}
