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

func TestCreateHotUpdateGateFromCandidatePromotionDecisionCreatesPreparedGate(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 25, 15, 0, 0, 0, time.UTC)
	decision := storeCandidatePromotionDecisionGateFixture(t, root, now, true)

	got, changed, err := CreateHotUpdateGateFromCandidatePromotionDecision(root, decision.PromotionDecisionID, " operator ", now.Add(10*time.Minute))
	if err != nil {
		t.Fatalf("CreateHotUpdateGateFromCandidatePromotionDecision() error = %v", err)
	}
	if !changed {
		t.Fatal("changed = false, want true")
	}
	if got.HotUpdateID != "hot-update-candidate-promotion-decision-result-eligible" {
		t.Fatalf("HotUpdateID = %q, want deterministic decision-derived id", got.HotUpdateID)
	}
	if got.CandidatePackID != decision.CandidatePackID {
		t.Fatalf("CandidatePackID = %q, want %q", got.CandidatePackID, decision.CandidatePackID)
	}
	if got.PreviousActivePackID != decision.BaselinePackID {
		t.Fatalf("PreviousActivePackID = %q, want %q", got.PreviousActivePackID, decision.BaselinePackID)
	}
	if got.RollbackTargetPackID != decision.BaselinePackID {
		t.Fatalf("RollbackTargetPackID = %q, want %q", got.RollbackTargetPackID, decision.BaselinePackID)
	}
	if got.State != HotUpdateGateStatePrepared {
		t.Fatalf("State = %q, want prepared", got.State)
	}
	if got.Decision != HotUpdateGateDecisionKeepStaged {
		t.Fatalf("Decision = %q, want keep_staged", got.Decision)
	}
	if got.PreparedAt != now.Add(10*time.Minute).UTC() {
		t.Fatalf("PreparedAt = %v, want %v", got.PreparedAt, now.Add(10*time.Minute).UTC())
	}
	if got.PhaseUpdatedBy != "operator" {
		t.Fatalf("PhaseUpdatedBy = %q, want operator", got.PhaseUpdatedBy)
	}
	if got.CompatibilityContractRef != "compat-v1" {
		t.Fatalf("CompatibilityContractRef = %q, want compat-v1", got.CompatibilityContractRef)
	}
	if got.ReloadMode != HotUpdateReloadModeSoftReload {
		t.Fatalf("ReloadMode = %q, want soft_reload", got.ReloadMode)
	}
}

func TestCreateHotUpdateGateFromCandidatePromotionDecisionRejectsMissingOrStaleActivePointer(t *testing.T) {
	t.Parallel()

	t.Run("missing active pointer", func(t *testing.T) {
		t.Parallel()

		root := t.TempDir()
		now := time.Date(2026, 4, 25, 15, 30, 0, 0, time.UTC)
		storeCandidatePromotionEligibilityFixtures(t, root, now, nil, nil)
		decision, _, err := CreateCandidatePromotionDecisionFromEligibleResult(root, "result-eligible", "operator", now.Add(8*time.Minute))
		if err != nil {
			t.Fatalf("CreateCandidatePromotionDecisionFromEligibleResult() error = %v", err)
		}

		_, changed, err := CreateHotUpdateGateFromCandidatePromotionDecision(root, decision.PromotionDecisionID, "operator", now.Add(10*time.Minute))
		if !errors.Is(err, ErrActiveRuntimePackPointerNotFound) {
			t.Fatalf("CreateHotUpdateGateFromCandidatePromotionDecision() error = %v, want %v", err, ErrActiveRuntimePackPointerNotFound)
		}
		if changed {
			t.Fatal("changed = true, want false")
		}
	})

	t.Run("stale active pointer", func(t *testing.T) {
		t.Parallel()

		root := t.TempDir()
		now := time.Date(2026, 4, 25, 15, 45, 0, 0, time.UTC)
		decision := storeCandidatePromotionDecisionGateFixture(t, root, now, false)
		mustStoreRuntimePack(t, root, validRuntimePackRecord(now.Add(9*time.Minute), func(record *RuntimePackRecord) {
			record.PackID = "pack-other-active"
		}))
		if err := StoreActiveRuntimePackPointer(root, ActiveRuntimePackPointer{
			ActivePackID:        "pack-other-active",
			LastKnownGoodPackID: "pack-base",
			UpdatedAt:           now.Add(9 * time.Minute),
			UpdatedBy:           "operator",
			UpdateRecordRef:     "other-hot-update",
			ReloadGeneration:    3,
		}); err != nil {
			t.Fatalf("StoreActiveRuntimePackPointer(stale) error = %v", err)
		}

		_, changed, err := CreateHotUpdateGateFromCandidatePromotionDecision(root, decision.PromotionDecisionID, "operator", now.Add(10*time.Minute))
		if err == nil {
			t.Fatal("CreateHotUpdateGateFromCandidatePromotionDecision() error = nil, want stale active rejection")
		}
		if changed {
			t.Fatal("changed = true, want false")
		}
		if !strings.Contains(err.Error(), `requires active runtime pack pointer active_pack_id "pack-base", found "pack-other-active"`) {
			t.Fatalf("stale active error = %q, want active pointer context", err.Error())
		}
	})
}

func TestCreateHotUpdateGateFromCandidatePromotionDecisionRejectsRollbackTargetProblems(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		editPack func(*RuntimePackRecord)
		want     string
	}{
		{
			name: "missing rollback target field",
			editPack: func(record *RuntimePackRecord) {
				record.RollbackTargetPackID = ""
			},
			want: "rollback_target_pack_id is required",
		},
		{
			name: "missing rollback target pack",
			editPack: func(record *RuntimePackRecord) {
				record.RollbackTargetPackID = "pack-rollback-missing"
			},
			want: "rollback_target_pack_id \"pack-rollback-missing\"",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			now := time.Date(2026, 4, 25, 16, 0, 0, 0, time.UTC)
			decision := storeCandidatePromotionDecisionGateFixture(t, root, now, false)
			mustStoreRuntimePack(t, root, validRuntimePackRecord(now.Add(9*time.Minute), func(record *RuntimePackRecord) {
				record.PackID = decision.CandidatePackID
				record.ParentPackID = decision.BaselinePackID
				tt.editPack(record)
			}))

			_, changed, err := CreateHotUpdateGateFromCandidatePromotionDecision(root, decision.PromotionDecisionID, "operator", now.Add(10*time.Minute))
			if err == nil {
				t.Fatal("CreateHotUpdateGateFromCandidatePromotionDecision() error = nil, want rollback target rejection")
			}
			if changed {
				t.Fatal("changed = true, want false")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("rollback target error = %q, want substring %q", err.Error(), tt.want)
			}
		})
	}
}

func TestCreateHotUpdateGateFromCandidatePromotionDecisionRejectsStaleDecisionAuthority(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		edit func(t *testing.T, root string, decision CandidatePromotionDecisionRecord)
		want string
	}{
		{
			name: "mismatched decision candidate pack",
			edit: func(t *testing.T, root string, decision CandidatePromotionDecisionRecord) {
				t.Helper()
				mustStoreRuntimePack(t, root, validRuntimePackRecord(time.Date(2026, 4, 25, 16, 30, 0, 0, time.UTC), func(record *RuntimePackRecord) {
					record.PackID = "pack-other-candidate"
					record.ParentPackID = decision.BaselinePackID
					record.RollbackTargetPackID = decision.BaselinePackID
				}))
				decision.CandidatePackID = "pack-other-candidate"
				if err := WriteStoreJSONAtomic(StoreCandidatePromotionDecisionPath(root, decision.PromotionDecisionID), decision); err != nil {
					t.Fatalf("WriteStoreJSONAtomic(decision mismatch) error = %v", err)
				}
			},
			want: "does not match candidate result candidate_pack_id",
		},
		{
			name: "derived eligibility no longer eligible",
			edit: func(t *testing.T, root string, decision CandidatePromotionDecisionRecord) {
				t.Helper()
				result, err := LoadCandidateResultRecord(root, decision.ResultID)
				if err != nil {
					t.Fatalf("LoadCandidateResultRecord() error = %v", err)
				}
				result.HoldoutScore = result.BaselineScore
				if err := WriteStoreJSONAtomic(StoreCandidateResultPath(root, result.ResultID), result); err != nil {
					t.Fatalf("WriteStoreJSONAtomic(result rejected) error = %v", err)
				}
			},
			want: `promotion eligibility state "rejected"`,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			now := time.Date(2026, 4, 25, 16, 30, 0, 0, time.UTC)
			decision := storeCandidatePromotionDecisionGateFixture(t, root, now, false)
			tt.edit(t, root, decision)

			_, changed, err := CreateHotUpdateGateFromCandidatePromotionDecision(root, decision.PromotionDecisionID, "operator", now.Add(10*time.Minute))
			if err == nil {
				t.Fatal("CreateHotUpdateGateFromCandidatePromotionDecision() error = nil, want stale authority rejection")
			}
			if changed {
				t.Fatal("changed = true, want false")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("stale authority error = %q, want substring %q", err.Error(), tt.want)
			}
		})
	}
}

func TestCreateHotUpdateGateFromCandidatePromotionDecisionReplayAndDuplicates(t *testing.T) {
	t.Parallel()

	t.Run("exact replay is byte stable", func(t *testing.T) {
		t.Parallel()

		root := t.TempDir()
		now := time.Date(2026, 4, 25, 17, 0, 0, 0, time.UTC)
		decision := storeCandidatePromotionDecisionGateFixture(t, root, now, false)
		createdAt := now.Add(10 * time.Minute)

		first, changed, err := CreateHotUpdateGateFromCandidatePromotionDecision(root, decision.PromotionDecisionID, "operator", createdAt)
		if err != nil {
			t.Fatalf("CreateHotUpdateGateFromCandidatePromotionDecision(first) error = %v", err)
		}
		if !changed {
			t.Fatal("first changed = false, want true")
		}
		firstBytes, err := os.ReadFile(StoreHotUpdateGatePath(root, first.HotUpdateID))
		if err != nil {
			t.Fatalf("ReadFile(first) error = %v", err)
		}

		second, changed, err := CreateHotUpdateGateFromCandidatePromotionDecision(root, decision.PromotionDecisionID, " operator ", createdAt)
		if err != nil {
			t.Fatalf("CreateHotUpdateGateFromCandidatePromotionDecision(replay) error = %v", err)
		}
		if changed {
			t.Fatal("replay changed = true, want false")
		}
		if !reflect.DeepEqual(second, first) {
			t.Fatalf("replay = %#v, want %#v", second, first)
		}
		secondBytes, err := os.ReadFile(StoreHotUpdateGatePath(root, first.HotUpdateID))
		if err != nil {
			t.Fatalf("ReadFile(second) error = %v", err)
		}
		if string(firstBytes) != string(secondBytes) {
			t.Fatalf("hot-update gate changed on exact replay\nfirst:\n%s\nsecond:\n%s", string(firstBytes), string(secondBytes))
		}
	})

	t.Run("divergent duplicate fails closed", func(t *testing.T) {
		t.Parallel()

		root := t.TempDir()
		now := time.Date(2026, 4, 25, 17, 30, 0, 0, time.UTC)
		decision := storeCandidatePromotionDecisionGateFixture(t, root, now, false)
		if _, _, err := CreateHotUpdateGateFromCandidatePromotionDecision(root, decision.PromotionDecisionID, "operator", now.Add(10*time.Minute)); err != nil {
			t.Fatalf("CreateHotUpdateGateFromCandidatePromotionDecision(first) error = %v", err)
		}

		_, changed, err := CreateHotUpdateGateFromCandidatePromotionDecision(root, decision.PromotionDecisionID, "operator", now.Add(11*time.Minute))
		if err == nil {
			t.Fatal("CreateHotUpdateGateFromCandidatePromotionDecision(divergent) error = nil, want duplicate rejection")
		}
		if changed {
			t.Fatal("changed = true, want false")
		}
		if !strings.Contains(err.Error(), "already exists with divergent candidate promotion decision authority") {
			t.Fatalf("divergent duplicate error = %q, want duplicate context", err.Error())
		}
	})

	t.Run("existing deterministic gate with different candidate fails closed", func(t *testing.T) {
		t.Parallel()

		root := t.TempDir()
		now := time.Date(2026, 4, 25, 18, 0, 0, 0, time.UTC)
		decision := storeCandidatePromotionDecisionGateFixture(t, root, now, false)
		mustStoreRuntimePack(t, root, validRuntimePackRecord(now.Add(9*time.Minute), func(record *RuntimePackRecord) {
			record.PackID = "pack-other-candidate"
			record.ParentPackID = decision.BaselinePackID
			record.RollbackTargetPackID = decision.BaselinePackID
		}))
		if err := StoreHotUpdateGateRecord(root, validHotUpdateGateRecord(now.Add(9*time.Minute), func(record *HotUpdateGateRecord) {
			record.HotUpdateID = "hot-update-" + decision.PromotionDecisionID
			record.CandidatePackID = "pack-other-candidate"
			record.PreviousActivePackID = decision.BaselinePackID
			record.RollbackTargetPackID = decision.BaselinePackID
		})); err != nil {
			t.Fatalf("StoreHotUpdateGateRecord(existing different candidate) error = %v", err)
		}

		_, changed, err := CreateHotUpdateGateFromCandidatePromotionDecision(root, decision.PromotionDecisionID, "operator", now.Add(10*time.Minute))
		if err == nil {
			t.Fatal("CreateHotUpdateGateFromCandidatePromotionDecision() error = nil, want existing candidate mismatch rejection")
		}
		if changed {
			t.Fatal("changed = true, want false")
		}
		if !strings.Contains(err.Error(), `candidate_pack_id "pack-other-candidate" does not match candidate promotion decision candidate_pack_id "pack-candidate"`) {
			t.Fatalf("existing candidate mismatch error = %q, want candidate mismatch context", err.Error())
		}
	})
}

func TestCreateHotUpdateGateFromCandidatePromotionDecisionPreservesSourceAndRuntimeState(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 25, 18, 30, 0, 0, time.UTC)
	decision := storeCandidatePromotionDecisionGateFixture(t, root, now, true)

	snapshots := map[string][]byte{}
	for _, path := range []string{
		StoreCandidatePromotionDecisionPath(root, decision.PromotionDecisionID),
		StoreCandidateResultPath(root, decision.ResultID),
		StoreImprovementRunPath(root, decision.RunID),
		StoreImprovementCandidatePath(root, decision.CandidateID),
		StoreEvalSuitePath(root, decision.EvalSuiteID),
		StorePromotionPolicyPath(root, decision.PromotionPolicyID),
		StoreRuntimePackPath(root, decision.BaselinePackID),
		StoreRuntimePackPath(root, decision.CandidatePackID),
		StoreActiveRuntimePackPointerPath(root),
		StoreLastKnownGoodRuntimePackPointerPath(root),
		StoreHotUpdateGatePath(root, "hot-update-1"),
	} {
		bytes, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ReadFile(%s) before helper error = %v", path, err)
		}
		snapshots[path] = bytes
	}
	beforePointer, err := LoadActiveRuntimePackPointer(root)
	if err != nil {
		t.Fatalf("LoadActiveRuntimePackPointer(before) error = %v", err)
	}

	if _, _, err := CreateHotUpdateGateFromCandidatePromotionDecision(root, decision.PromotionDecisionID, "operator", now.Add(10*time.Minute)); err != nil {
		t.Fatalf("CreateHotUpdateGateFromCandidatePromotionDecision() error = %v", err)
	}

	for path, before := range snapshots {
		after, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ReadFile(%s) after helper error = %v", path, err)
		}
		if string(after) != string(before) {
			t.Fatalf("source/runtime file %s changed after hot-update gate helper", path)
		}
	}
	afterPointer, err := LoadActiveRuntimePackPointer(root)
	if err != nil {
		t.Fatalf("LoadActiveRuntimePackPointer(after) error = %v", err)
	}
	if afterPointer.ReloadGeneration != beforePointer.ReloadGeneration {
		t.Fatalf("ReloadGeneration = %d, want unchanged %d", afterPointer.ReloadGeneration, beforePointer.ReloadGeneration)
	}

	outcomes, err := ListHotUpdateOutcomeRecords(root)
	if err != nil {
		t.Fatalf("ListHotUpdateOutcomeRecords() error = %v", err)
	}
	if len(outcomes) != 0 {
		t.Fatalf("ListHotUpdateOutcomeRecords() len = %d, want 0", len(outcomes))
	}
	promotions, err := ListPromotionRecords(root)
	if err != nil {
		t.Fatalf("ListPromotionRecords() error = %v", err)
	}
	if len(promotions) != 0 {
		t.Fatalf("ListPromotionRecords() len = %d, want 0", len(promotions))
	}
	rollbacks, err := ListRollbackRecords(root)
	if err != nil {
		t.Fatalf("ListRollbackRecords() error = %v", err)
	}
	if len(rollbacks) != 0 {
		t.Fatalf("ListRollbackRecords() len = %d, want 0", len(rollbacks))
	}
	applies, err := ListRollbackApplyRecords(root)
	if err != nil {
		t.Fatalf("ListRollbackApplyRecords() error = %v", err)
	}
	if len(applies) != 0 {
		t.Fatalf("ListRollbackApplyRecords() len = %d, want 0", len(applies))
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

func TestExecuteHotUpdateGateReloadApplyHappyPathPreservesPointerAndLastKnownGood(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 22, 15, 0, 0, 0, time.UTC)
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

	if _, _, err := EnsureHotUpdateGateRecordFromCandidate(root, "hot-update-reload-success", "pack-candidate", "operator", now.Add(3*time.Minute)); err != nil {
		t.Fatalf("EnsureHotUpdateGateRecordFromCandidate() error = %v", err)
	}
	if _, _, err := AdvanceHotUpdateGatePhase(root, "hot-update-reload-success", HotUpdateGateStateValidated, "operator", now.Add(4*time.Minute)); err != nil {
		t.Fatalf("AdvanceHotUpdateGatePhase(validated) error = %v", err)
	}
	if _, _, err := AdvanceHotUpdateGatePhase(root, "hot-update-reload-success", HotUpdateGateStateStaged, "operator", now.Add(5*time.Minute)); err != nil {
		t.Fatalf("AdvanceHotUpdateGatePhase(staged) error = %v", err)
	}
	if _, _, err := ExecuteHotUpdateGatePointerSwitch(root, "hot-update-reload-success", "operator", now.Add(6*time.Minute)); err != nil {
		t.Fatalf("ExecuteHotUpdateGatePointerSwitch() error = %v", err)
	}

	beforePointerBytes, err := os.ReadFile(StoreActiveRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(active pointer before) error = %v", err)
	}
	beforeLKGBytes, err := os.ReadFile(StoreLastKnownGoodRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(last-known-good before) error = %v", err)
	}

	record, changed, err := ExecuteHotUpdateGateReloadApply(root, "hot-update-reload-success", "operator", now.Add(7*time.Minute))
	if err != nil {
		t.Fatalf("ExecuteHotUpdateGateReloadApply() error = %v", err)
	}
	if !changed {
		t.Fatal("ExecuteHotUpdateGateReloadApply() changed = false, want true")
	}
	if record.State != HotUpdateGateStateReloadApplySucceeded {
		t.Fatalf("ExecuteHotUpdateGateReloadApply().State = %q, want reload_apply_succeeded", record.State)
	}
	if record.FailureReason != "" {
		t.Fatalf("ExecuteHotUpdateGateReloadApply().FailureReason = %q, want empty", record.FailureReason)
	}

	afterPointerBytes, err := os.ReadFile(StoreActiveRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(active pointer after) error = %v", err)
	}
	if string(beforePointerBytes) != string(afterPointerBytes) {
		t.Fatalf("active runtime pack pointer file changed during hot-update reload/apply\nbefore:\n%s\nafter:\n%s", string(beforePointerBytes), string(afterPointerBytes))
	}
	afterLKGBytes, err := os.ReadFile(StoreLastKnownGoodRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(last-known-good after) error = %v", err)
	}
	if string(beforeLKGBytes) != string(afterLKGBytes) {
		t.Fatalf("last-known-good pointer file changed during hot-update reload/apply\nbefore:\n%s\nafter:\n%s", string(beforeLKGBytes), string(afterLKGBytes))
	}

	gotPointer, err := LoadActiveRuntimePackPointer(root)
	if err != nil {
		t.Fatalf("LoadActiveRuntimePackPointer() error = %v", err)
	}
	if gotPointer.ActivePackID != "pack-candidate" {
		t.Fatalf("LoadActiveRuntimePackPointer().ActivePackID = %q, want pack-candidate", gotPointer.ActivePackID)
	}
	if gotPointer.ReloadGeneration != 3 {
		t.Fatalf("LoadActiveRuntimePackPointer().ReloadGeneration = %d, want 3", gotPointer.ReloadGeneration)
	}

	second, changed, err := ExecuteHotUpdateGateReloadApply(root, "hot-update-reload-success", "operator", now.Add(8*time.Minute))
	if err != nil {
		t.Fatalf("ExecuteHotUpdateGateReloadApply(replay) error = %v", err)
	}
	if changed {
		t.Fatal("ExecuteHotUpdateGateReloadApply(replay) changed = true, want false")
	}
	if !reflect.DeepEqual(second, record) {
		t.Fatalf("ExecuteHotUpdateGateReloadApply(replay) = %#v, want %#v", second, record)
	}

	outcomes, err := ListHotUpdateOutcomeRecords(root)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("ListHotUpdateOutcomeRecords() error = %v", err)
	}
	if len(outcomes) != 0 {
		t.Fatalf("ListHotUpdateOutcomeRecords() len = %d, want 0", len(outcomes))
	}
	promotions, err := ListPromotionRecords(root)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("ListPromotionRecords() error = %v", err)
	}
	if len(promotions) != 0 {
		t.Fatalf("ListPromotionRecords() len = %d, want 0", len(promotions))
	}
}

func TestExecuteHotUpdateGateReloadApplyRecordsFailureWithoutMutatingPointer(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 22, 15, 30, 0, 0, time.UTC)
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
	if err := StoreLastKnownGoodRuntimePackPointer(root, LastKnownGoodRuntimePackPointer{
		PackID:            "pack-base",
		Basis:             "holdout_pass",
		VerifiedAt:        now.Add(2 * time.Minute),
		VerifiedBy:        "operator",
		RollbackRecordRef: "bootstrap",
	}); err != nil {
		t.Fatalf("StoreLastKnownGoodRuntimePackPointer() error = %v", err)
	}

	if _, _, err := EnsureHotUpdateGateRecordFromCandidate(root, "hot-update-reload-failed", "pack-candidate", "operator", now.Add(3*time.Minute)); err != nil {
		t.Fatalf("EnsureHotUpdateGateRecordFromCandidate() error = %v", err)
	}
	if _, _, err := AdvanceHotUpdateGatePhase(root, "hot-update-reload-failed", HotUpdateGateStateValidated, "operator", now.Add(4*time.Minute)); err != nil {
		t.Fatalf("AdvanceHotUpdateGatePhase(validated) error = %v", err)
	}
	if _, _, err := AdvanceHotUpdateGatePhase(root, "hot-update-reload-failed", HotUpdateGateStateStaged, "operator", now.Add(5*time.Minute)); err != nil {
		t.Fatalf("AdvanceHotUpdateGatePhase(staged) error = %v", err)
	}
	if _, _, err := ExecuteHotUpdateGatePointerSwitch(root, "hot-update-reload-failed", "operator", now.Add(6*time.Minute)); err != nil {
		t.Fatalf("ExecuteHotUpdateGatePointerSwitch() error = %v", err)
	}

	beforePointerBytes, err := os.ReadFile(StoreActiveRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(active pointer before) error = %v", err)
	}
	beforeLKGBytes, err := os.ReadFile(StoreLastKnownGoodRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(last-known-good before) error = %v", err)
	}

	record, changed, err := executeHotUpdateGateReloadApplyWithConvergence(root, "hot-update-reload-failed", "operator", now.Add(7*time.Minute), func(root string, record HotUpdateGateRecord) error {
		return errors.New("simulated hot-update convergence failure")
	})
	if err == nil {
		t.Fatal("executeHotUpdateGateReloadApplyWithConvergence() error = nil, want failure")
	}
	if !changed {
		t.Fatal("executeHotUpdateGateReloadApplyWithConvergence() changed = false, want true")
	}
	if record.State != HotUpdateGateStateReloadApplyFailed {
		t.Fatalf("executeHotUpdateGateReloadApplyWithConvergence().State = %q, want reload_apply_failed", record.State)
	}
	if record.FailureReason != "simulated hot-update convergence failure" {
		t.Fatalf("executeHotUpdateGateReloadApplyWithConvergence().FailureReason = %q, want simulated failure", record.FailureReason)
	}

	afterPointerBytes, err := os.ReadFile(StoreActiveRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(active pointer after) error = %v", err)
	}
	if string(beforePointerBytes) != string(afterPointerBytes) {
		t.Fatalf("active runtime pack pointer file changed on hot-update convergence failure\nbefore:\n%s\nafter:\n%s", string(beforePointerBytes), string(afterPointerBytes))
	}
	afterLKGBytes, err := os.ReadFile(StoreLastKnownGoodRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(last-known-good after) error = %v", err)
	}
	if string(beforeLKGBytes) != string(afterLKGBytes) {
		t.Fatalf("last-known-good pointer file changed on hot-update convergence failure\nbefore:\n%s\nafter:\n%s", string(beforeLKGBytes), string(afterLKGBytes))
	}
}

func TestExecuteHotUpdateGateReloadApplyRejectsInvalidStateAndBadAttribution(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 22, 16, 0, 0, 0, time.UTC)
	mustStoreRuntimePack(t, root, validRuntimePackRecord(now, func(record *RuntimePackRecord) {
		record.PackID = "pack-base"
	}))
	mustStoreRuntimePack(t, root, validRuntimePackRecord(now.Add(time.Minute), func(record *RuntimePackRecord) {
		record.PackID = "pack-candidate"
		record.ParentPackID = "pack-base"
		record.RollbackTargetPackID = "pack-base"
	}))
	if err := StoreActiveRuntimePackPointer(root, ActiveRuntimePackPointer{
		ActivePackID:         "pack-candidate",
		PreviousActivePackID: "pack-base",
		LastKnownGoodPackID:  "pack-base",
		UpdatedAt:            now.Add(2 * time.Minute),
		UpdatedBy:            "operator",
		UpdateRecordRef:      "hot_update:hot-update-good",
		ReloadGeneration:     3,
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

	beforePointerBytes, err := os.ReadFile(StoreActiveRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(active pointer before) error = %v", err)
	}

	if err := StoreHotUpdateGateRecord(root, validHotUpdateGateRecord(now.Add(3*time.Minute), func(record *HotUpdateGateRecord) {
		record.HotUpdateID = "hot-update-not-reloading"
		record.CandidatePackID = "pack-candidate"
		record.PreviousActivePackID = "pack-base"
		record.RollbackTargetPackID = "pack-base"
		record.State = HotUpdateGateStateStaged
		record.PhaseUpdatedAt = now.Add(3 * time.Minute)
		record.PhaseUpdatedBy = "operator"
	})); err != nil {
		t.Fatalf("StoreHotUpdateGateRecord(staged) error = %v", err)
	}

	got, changed, err := ExecuteHotUpdateGateReloadApply(root, "hot-update-not-reloading", "operator", now.Add(4*time.Minute))
	if err == nil {
		t.Fatal("ExecuteHotUpdateGateReloadApply(staged) error = nil, want invalid state rejection")
	}
	if changed {
		t.Fatal("ExecuteHotUpdateGateReloadApply(staged) changed = true, want false")
	}
	if !reflect.DeepEqual(got, HotUpdateGateRecord{}) {
		t.Fatalf("ExecuteHotUpdateGateReloadApply(staged) record = %#v, want zero value", got)
	}
	if !strings.Contains(err.Error(), `state "staged" does not permit reload/apply execution`) {
		t.Fatalf("ExecuteHotUpdateGateReloadApply(staged) error = %q, want invalid state context", err.Error())
	}

	if err := StoreHotUpdateGateRecord(root, validHotUpdateGateRecord(now.Add(5*time.Minute), func(record *HotUpdateGateRecord) {
		record.HotUpdateID = "hot-update-bad-attribution"
		record.CandidatePackID = "pack-candidate"
		record.PreviousActivePackID = "pack-base"
		record.RollbackTargetPackID = "pack-base"
		record.State = HotUpdateGateStateReloading
		record.PhaseUpdatedAt = now.Add(5 * time.Minute)
		record.PhaseUpdatedBy = "operator"
	})); err != nil {
		t.Fatalf("StoreHotUpdateGateRecord(reloading) error = %v", err)
	}

	got, changed, err = ExecuteHotUpdateGateReloadApply(root, "hot-update-bad-attribution", "operator", now.Add(6*time.Minute))
	if err == nil {
		t.Fatal("ExecuteHotUpdateGateReloadApply(bad attribution) error = nil, want attribution rejection")
	}
	if changed {
		t.Fatal("ExecuteHotUpdateGateReloadApply(bad attribution) changed = true, want false")
	}
	if !reflect.DeepEqual(got, HotUpdateGateRecord{}) {
		t.Fatalf("ExecuteHotUpdateGateReloadApply(bad attribution) record = %#v, want zero value", got)
	}
	if !strings.Contains(err.Error(), `update_record_ref "hot_update:hot-update-bad-attribution"`) {
		t.Fatalf("ExecuteHotUpdateGateReloadApply(bad attribution) error = %q, want update_record_ref context", err.Error())
	}

	afterPointerBytes, err := os.ReadFile(StoreActiveRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(active pointer after) error = %v", err)
	}
	if string(beforePointerBytes) != string(afterPointerBytes) {
		t.Fatalf("active runtime pack pointer file changed after rejected reload/apply\nbefore:\n%s\nafter:\n%s", string(beforePointerBytes), string(afterPointerBytes))
	}
}

func TestReconcileHotUpdateGateRecoveryNeededNormalizesInProgressWithoutMutatingPointerState(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 23, 10, 0, 0, 0, time.UTC)
	storeHotUpdateReloadInProgressFixture(t, root, now, "hot-update-recovery")

	beforePointerBytes, err := os.ReadFile(StoreActiveRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(active pointer before) error = %v", err)
	}
	beforeLastKnownGoodBytes, err := os.ReadFile(StoreLastKnownGoodRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(last known good before) error = %v", err)
	}

	record, changed, err := ReconcileHotUpdateGateRecoveryNeeded(root, "hot-update-recovery", "operator", now.Add(8*time.Minute))
	if err != nil {
		t.Fatalf("ReconcileHotUpdateGateRecoveryNeeded() error = %v", err)
	}
	if !changed {
		t.Fatal("ReconcileHotUpdateGateRecoveryNeeded() changed = false, want true")
	}
	if record.State != HotUpdateGateStateReloadApplyRecoveryNeeded {
		t.Fatalf("ReconcileHotUpdateGateRecoveryNeeded().State = %q, want reload_apply_recovery_needed", record.State)
	}
	if record.FailureReason != "" {
		t.Fatalf("ReconcileHotUpdateGateRecoveryNeeded().FailureReason = %q, want empty", record.FailureReason)
	}
	if record.PhaseUpdatedAt != now.Add(8*time.Minute).UTC() {
		t.Fatalf("ReconcileHotUpdateGateRecoveryNeeded().PhaseUpdatedAt = %v, want %v", record.PhaseUpdatedAt, now.Add(8*time.Minute).UTC())
	}
	if record.PhaseUpdatedBy != "operator" {
		t.Fatalf("ReconcileHotUpdateGateRecoveryNeeded().PhaseUpdatedBy = %q, want operator", record.PhaseUpdatedBy)
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
	if gotPointer.ActivePackID != "pack-candidate" {
		t.Fatalf("LoadActiveRuntimePackPointer().ActivePackID = %q, want pack-candidate", gotPointer.ActivePackID)
	}
	if gotPointer.ReloadGeneration != 3 {
		t.Fatalf("LoadActiveRuntimePackPointer().ReloadGeneration = %d, want 3", gotPointer.ReloadGeneration)
	}

	outcomes, err := ListHotUpdateOutcomeRecords(root)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("ListHotUpdateOutcomeRecords() error = %v", err)
	}
	if len(outcomes) != 0 {
		t.Fatalf("ListHotUpdateOutcomeRecords() len = %d, want 0", len(outcomes))
	}
	promotions, err := ListPromotionRecords(root)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("ListPromotionRecords() error = %v", err)
	}
	if len(promotions) != 0 {
		t.Fatalf("ListPromotionRecords() len = %d, want 0", len(promotions))
	}
}

func TestReconcileHotUpdateGateRecoveryNeededRejectsInvalidLinkageWithoutPointerMutation(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 23, 10, 30, 0, 0, time.UTC)

	t.Run("missing rollback target pack", func(t *testing.T) {
		t.Parallel()

		root := t.TempDir()
		storeHotUpdateReloadInProgressFixture(t, root, now, "hot-update-recovery-missing")

		beforePointerBytes, err := os.ReadFile(StoreActiveRuntimePackPointerPath(root))
		if err != nil {
			t.Fatalf("ReadFile(active pointer before) error = %v", err)
		}
		beforeLastKnownGoodBytes, err := os.ReadFile(StoreLastKnownGoodRuntimePackPointerPath(root))
		if err != nil {
			t.Fatalf("ReadFile(last known good before) error = %v", err)
		}
		if err := os.Remove(StoreRuntimePackPath(root, "pack-base")); err != nil {
			t.Fatalf("Remove(runtime pack) error = %v", err)
		}

		got, changed, err := ReconcileHotUpdateGateRecoveryNeeded(root, "hot-update-recovery-missing", "operator", now.Add(8*time.Minute))
		if err == nil {
			t.Fatal("ReconcileHotUpdateGateRecoveryNeeded() error = nil, want missing rollback-target rejection")
		}
		if changed {
			t.Fatal("ReconcileHotUpdateGateRecoveryNeeded() changed = true, want false")
		}
		if !reflect.DeepEqual(got, HotUpdateGateRecord{}) {
			t.Fatalf("ReconcileHotUpdateGateRecoveryNeeded() record = %#v, want zero value", got)
		}
		if !strings.Contains(err.Error(), `mission store runtime pack record not found`) {
			t.Fatalf("ReconcileHotUpdateGateRecoveryNeeded() error = %q, want broken linkage rejection", err.Error())
		}

		afterPointerBytes, err := os.ReadFile(StoreActiveRuntimePackPointerPath(root))
		if err != nil {
			t.Fatalf("ReadFile(active pointer after) error = %v", err)
		}
		if string(beforePointerBytes) != string(afterPointerBytes) {
			t.Fatalf("active runtime pack pointer file changed on missing rollback-target rejection\nbefore:\n%s\nafter:\n%s", string(beforePointerBytes), string(afterPointerBytes))
		}
		afterLastKnownGoodBytes, err := os.ReadFile(StoreLastKnownGoodRuntimePackPointerPath(root))
		if err != nil {
			t.Fatalf("ReadFile(last known good after) error = %v", err)
		}
		if string(beforeLastKnownGoodBytes) != string(afterLastKnownGoodBytes) {
			t.Fatalf("last-known-good pointer file changed on missing rollback-target rejection\nbefore:\n%s\nafter:\n%s", string(beforeLastKnownGoodBytes), string(afterLastKnownGoodBytes))
		}
	})

	t.Run("invalid active pointer attribution", func(t *testing.T) {
		t.Parallel()

		root := t.TempDir()
		storeHotUpdateReloadInProgressFixture(t, root, now, "hot-update-recovery-pointer")

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

		got, changed, err := ReconcileHotUpdateGateRecoveryNeeded(root, "hot-update-recovery-pointer", "operator", now.Add(9*time.Minute))
		if err == nil {
			t.Fatal("ReconcileHotUpdateGateRecoveryNeeded() error = nil, want invalid active pointer rejection")
		}
		if changed {
			t.Fatal("ReconcileHotUpdateGateRecoveryNeeded() changed = true, want false")
		}
		if !reflect.DeepEqual(got, HotUpdateGateRecord{}) {
			t.Fatalf("ReconcileHotUpdateGateRecoveryNeeded() record = %#v, want zero value", got)
		}
		if !strings.Contains(err.Error(), `update_record_ref "hot_update:hot-update-recovery-pointer"`) {
			t.Fatalf("ReconcileHotUpdateGateRecoveryNeeded() error = %q, want invalid active pointer context", err.Error())
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
	})
}

func TestReconcileHotUpdateGateRecoveryNeededReplayIsIdempotent(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 23, 11, 0, 0, 0, time.UTC)
	storeHotUpdateReloadInProgressFixture(t, root, now, "hot-update-recovery-replay")

	if _, changed, err := ReconcileHotUpdateGateRecoveryNeeded(root, "hot-update-recovery-replay", "operator", now.Add(8*time.Minute)); err != nil {
		t.Fatalf("ReconcileHotUpdateGateRecoveryNeeded(first) error = %v", err)
	} else if !changed {
		t.Fatal("ReconcileHotUpdateGateRecoveryNeeded(first) changed = false, want true")
	}

	firstPointerBytes, err := os.ReadFile(StoreActiveRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(active pointer first) error = %v", err)
	}
	firstLastKnownGoodBytes, err := os.ReadFile(StoreLastKnownGoodRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(last known good first) error = %v", err)
	}

	record, changed, err := ReconcileHotUpdateGateRecoveryNeeded(root, "hot-update-recovery-replay", "operator", now.Add(9*time.Minute))
	if err != nil {
		t.Fatalf("ReconcileHotUpdateGateRecoveryNeeded(second) error = %v", err)
	}
	if changed {
		t.Fatal("ReconcileHotUpdateGateRecoveryNeeded(second) changed = true, want false")
	}
	if record.State != HotUpdateGateStateReloadApplyRecoveryNeeded {
		t.Fatalf("ReconcileHotUpdateGateRecoveryNeeded(second).State = %q, want reload_apply_recovery_needed", record.State)
	}

	secondPointerBytes, err := os.ReadFile(StoreActiveRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(active pointer second) error = %v", err)
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

func TestExecuteHotUpdateGateReloadApplyRetryFromRecoveryNeededSucceedsWithoutSecondPointerMutation(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 23, 11, 30, 0, 0, time.UTC)
	storeHotUpdateRecoveryNeededFixture(t, root, now, "hot-update-retry-success")

	beforePointerBytes, err := os.ReadFile(StoreActiveRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(active pointer before) error = %v", err)
	}
	beforeLastKnownGoodBytes, err := os.ReadFile(StoreLastKnownGoodRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(last known good before) error = %v", err)
	}

	record, changed, err := ExecuteHotUpdateGateReloadApply(root, "hot-update-retry-success", "operator", now.Add(10*time.Minute))
	if err != nil {
		t.Fatalf("ExecuteHotUpdateGateReloadApply() error = %v", err)
	}
	if !changed {
		t.Fatal("ExecuteHotUpdateGateReloadApply() changed = false, want true")
	}
	if record.State != HotUpdateGateStateReloadApplySucceeded {
		t.Fatalf("ExecuteHotUpdateGateReloadApply().State = %q, want reload_apply_succeeded", record.State)
	}
	if record.FailureReason != "" {
		t.Fatalf("ExecuteHotUpdateGateReloadApply().FailureReason = %q, want empty", record.FailureReason)
	}

	afterPointerBytes, err := os.ReadFile(StoreActiveRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(active pointer after) error = %v", err)
	}
	if string(beforePointerBytes) != string(afterPointerBytes) {
		t.Fatalf("active runtime pack pointer file changed during retry reload/apply\nbefore:\n%s\nafter:\n%s", string(beforePointerBytes), string(afterPointerBytes))
	}
	afterLastKnownGoodBytes, err := os.ReadFile(StoreLastKnownGoodRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(last known good after) error = %v", err)
	}
	if string(beforeLastKnownGoodBytes) != string(afterLastKnownGoodBytes) {
		t.Fatalf("last-known-good pointer file changed during retry reload/apply\nbefore:\n%s\nafter:\n%s", string(beforeLastKnownGoodBytes), string(afterLastKnownGoodBytes))
	}

	gotPointer, err := LoadActiveRuntimePackPointer(root)
	if err != nil {
		t.Fatalf("LoadActiveRuntimePackPointer() error = %v", err)
	}
	if gotPointer.ActivePackID != "pack-candidate" {
		t.Fatalf("LoadActiveRuntimePackPointer().ActivePackID = %q, want pack-candidate", gotPointer.ActivePackID)
	}
	if gotPointer.ReloadGeneration != 3 {
		t.Fatalf("LoadActiveRuntimePackPointer().ReloadGeneration = %d, want 3", gotPointer.ReloadGeneration)
	}

	gates, err := ListHotUpdateGateRecords(root)
	if err != nil {
		t.Fatalf("ListHotUpdateGateRecords() error = %v", err)
	}
	if len(gates) != 1 {
		t.Fatalf("ListHotUpdateGateRecords() len = %d, want 1", len(gates))
	}
	outcomes, err := ListHotUpdateOutcomeRecords(root)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("ListHotUpdateOutcomeRecords() error = %v", err)
	}
	if len(outcomes) != 0 {
		t.Fatalf("ListHotUpdateOutcomeRecords() len = %d, want 0", len(outcomes))
	}
	promotions, err := ListPromotionRecords(root)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("ListPromotionRecords() error = %v", err)
	}
	if len(promotions) != 0 {
		t.Fatalf("ListPromotionRecords() len = %d, want 0", len(promotions))
	}
}

func TestExecuteHotUpdateGateReloadApplyRetryFromRecoveryNeededRecordsFailureAndClearsFailureReasonOnStart(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 23, 11, 45, 0, 0, time.UTC)
	storeHotUpdateRecoveryNeededFixture(t, root, now, "hot-update-retry-failure")

	beforePointerBytes, err := os.ReadFile(StoreActiveRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(active pointer before) error = %v", err)
	}
	beforeLastKnownGoodBytes, err := os.ReadFile(StoreLastKnownGoodRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(last known good before) error = %v", err)
	}

	observedInProgress := false
	record, changed, err := executeHotUpdateGateReloadApplyWithConvergence(root, "hot-update-retry-failure", "operator", now.Add(10*time.Minute), func(root string, record HotUpdateGateRecord) error {
		current, err := LoadHotUpdateGateRecord(root, record.HotUpdateID)
		if err != nil {
			return err
		}
		if current.State != HotUpdateGateStateReloadApplyInProgress {
			return errors.New("retry did not persist reload_apply_in_progress before convergence")
		}
		if current.FailureReason != "" {
			return errors.New("retry did not clear stale failure reason before convergence")
		}
		observedInProgress = true
		return errors.New("simulated retry convergence failure")
	})
	if err == nil {
		t.Fatal("executeHotUpdateGateReloadApplyWithConvergence() error = nil, want failure")
	}
	if !changed {
		t.Fatal("executeHotUpdateGateReloadApplyWithConvergence() changed = false, want true")
	}
	if !observedInProgress {
		t.Fatal("executeHotUpdateGateReloadApplyWithConvergence() did not persist reload_apply_in_progress with cleared failure_reason before failure")
	}
	if record.State != HotUpdateGateStateReloadApplyFailed {
		t.Fatalf("executeHotUpdateGateReloadApplyWithConvergence().State = %q, want reload_apply_failed", record.State)
	}
	if record.FailureReason != "simulated retry convergence failure" {
		t.Fatalf("executeHotUpdateGateReloadApplyWithConvergence().FailureReason = %q, want simulated retry failure", record.FailureReason)
	}

	afterPointerBytes, err := os.ReadFile(StoreActiveRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(active pointer after) error = %v", err)
	}
	if string(beforePointerBytes) != string(afterPointerBytes) {
		t.Fatalf("active runtime pack pointer file changed during retry failure\nbefore:\n%s\nafter:\n%s", string(beforePointerBytes), string(afterPointerBytes))
	}
	afterLastKnownGoodBytes, err := os.ReadFile(StoreLastKnownGoodRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(last known good after) error = %v", err)
	}
	if string(beforeLastKnownGoodBytes) != string(afterLastKnownGoodBytes) {
		t.Fatalf("last-known-good pointer file changed during retry failure\nbefore:\n%s\nafter:\n%s", string(beforeLastKnownGoodBytes), string(afterLastKnownGoodBytes))
	}

	gotPointer, err := LoadActiveRuntimePackPointer(root)
	if err != nil {
		t.Fatalf("LoadActiveRuntimePackPointer() error = %v", err)
	}
	if gotPointer.ReloadGeneration != 3 {
		t.Fatalf("LoadActiveRuntimePackPointer().ReloadGeneration = %d, want 3", gotPointer.ReloadGeneration)
	}

	gates, err := ListHotUpdateGateRecords(root)
	if err != nil {
		t.Fatalf("ListHotUpdateGateRecords() error = %v", err)
	}
	if len(gates) != 1 {
		t.Fatalf("ListHotUpdateGateRecords() len = %d, want 1", len(gates))
	}
	outcomes, err := ListHotUpdateOutcomeRecords(root)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("ListHotUpdateOutcomeRecords() error = %v", err)
	}
	if len(outcomes) != 0 {
		t.Fatalf("ListHotUpdateOutcomeRecords() len = %d, want 0", len(outcomes))
	}
	promotions, err := ListPromotionRecords(root)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("ListPromotionRecords() error = %v", err)
	}
	if len(promotions) != 0 {
		t.Fatalf("ListPromotionRecords() len = %d, want 0", len(promotions))
	}
}

func TestResolveHotUpdateGateTerminalFailureFromRecoveryNeededPreservesCommittedState(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 23, 12, 0, 0, 0, time.UTC)
	storeHotUpdateRecoveryNeededFixture(t, root, now, "hot-update-terminal-failure")

	beforePointerBytes, err := os.ReadFile(StoreActiveRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(active pointer before) error = %v", err)
	}
	beforeLastKnownGoodBytes, err := os.ReadFile(StoreLastKnownGoodRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(last known good before) error = %v", err)
	}

	record, changed, err := ResolveHotUpdateGateTerminalFailure(root, "hot-update-terminal-failure", "operator requested stop after recovery review", "operator", now.Add(10*time.Minute))
	if err != nil {
		t.Fatalf("ResolveHotUpdateGateTerminalFailure() error = %v", err)
	}
	if !changed {
		t.Fatal("ResolveHotUpdateGateTerminalFailure() changed = false, want true")
	}
	if record.State != HotUpdateGateStateReloadApplyFailed {
		t.Fatalf("ResolveHotUpdateGateTerminalFailure().State = %q, want reload_apply_failed", record.State)
	}
	if record.FailureReason != "operator_terminal_failure: operator requested stop after recovery review" {
		t.Fatalf("ResolveHotUpdateGateTerminalFailure().FailureReason = %q, want deterministic operator failure detail", record.FailureReason)
	}

	afterPointerBytes, err := os.ReadFile(StoreActiveRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(active pointer after) error = %v", err)
	}
	if string(beforePointerBytes) != string(afterPointerBytes) {
		t.Fatalf("active runtime pack pointer file changed during terminal failure resolution\nbefore:\n%s\nafter:\n%s", string(beforePointerBytes), string(afterPointerBytes))
	}
	afterLastKnownGoodBytes, err := os.ReadFile(StoreLastKnownGoodRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(last known good after) error = %v", err)
	}
	if string(beforeLastKnownGoodBytes) != string(afterLastKnownGoodBytes) {
		t.Fatalf("last-known-good pointer file changed during terminal failure resolution\nbefore:\n%s\nafter:\n%s", string(beforeLastKnownGoodBytes), string(afterLastKnownGoodBytes))
	}

	gotPointer, err := LoadActiveRuntimePackPointer(root)
	if err != nil {
		t.Fatalf("LoadActiveRuntimePackPointer() error = %v", err)
	}
	if gotPointer.ReloadGeneration != 3 {
		t.Fatalf("LoadActiveRuntimePackPointer().ReloadGeneration = %d, want 3", gotPointer.ReloadGeneration)
	}

	gates, err := ListHotUpdateGateRecords(root)
	if err != nil {
		t.Fatalf("ListHotUpdateGateRecords() error = %v", err)
	}
	if len(gates) != 1 {
		t.Fatalf("ListHotUpdateGateRecords() len = %d, want 1", len(gates))
	}
	outcomes, err := ListHotUpdateOutcomeRecords(root)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("ListHotUpdateOutcomeRecords() error = %v", err)
	}
	if len(outcomes) != 0 {
		t.Fatalf("ListHotUpdateOutcomeRecords() len = %d, want 0", len(outcomes))
	}
	promotions, err := ListPromotionRecords(root)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("ListPromotionRecords() error = %v", err)
	}
	if len(promotions) != 0 {
		t.Fatalf("ListPromotionRecords() len = %d, want 0", len(promotions))
	}
	applies, err := ListRollbackApplyRecords(root)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("ListRollbackApplyRecords() error = %v", err)
	}
	if len(applies) != 0 {
		t.Fatalf("ListRollbackApplyRecords() len = %d, want 0", len(applies))
	}
}

func TestResolveHotUpdateGateTerminalFailureRequiresReasonAndReplayIsIdempotent(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 23, 12, 15, 0, 0, time.UTC)
	storeHotUpdateRecoveryNeededFixture(t, root, now, "hot-update-terminal-replay")

	got, changed, err := ResolveHotUpdateGateTerminalFailure(root, "hot-update-terminal-replay", "   ", "operator", now.Add(10*time.Minute))
	if err == nil {
		t.Fatal("ResolveHotUpdateGateTerminalFailure() error = nil, want required reason rejection")
	}
	if changed {
		t.Fatal("ResolveHotUpdateGateTerminalFailure() changed = true, want false")
	}
	if !reflect.DeepEqual(got, HotUpdateGateRecord{}) {
		t.Fatalf("ResolveHotUpdateGateTerminalFailure() record = %#v, want zero value", got)
	}
	if !strings.Contains(err.Error(), "terminal failure reason is required") {
		t.Fatalf("ResolveHotUpdateGateTerminalFailure() error = %q, want required reason rejection", err.Error())
	}

	first, changed, err := ResolveHotUpdateGateTerminalFailure(root, "hot-update-terminal-replay", "operator requested stop after recovery review", "operator", now.Add(11*time.Minute))
	if err != nil {
		t.Fatalf("ResolveHotUpdateGateTerminalFailure(first) error = %v", err)
	}
	if !changed {
		t.Fatal("ResolveHotUpdateGateTerminalFailure(first) changed = false, want true")
	}
	firstBytes, err := os.ReadFile(StoreHotUpdateGatePath(root, "hot-update-terminal-replay"))
	if err != nil {
		t.Fatalf("ReadFile(first) error = %v", err)
	}

	second, changed, err := ResolveHotUpdateGateTerminalFailure(root, "hot-update-terminal-replay", "operator requested stop after recovery review", "operator", now.Add(12*time.Minute))
	if err != nil {
		t.Fatalf("ResolveHotUpdateGateTerminalFailure(second) error = %v", err)
	}
	if changed {
		t.Fatal("ResolveHotUpdateGateTerminalFailure(second) changed = true, want false")
	}
	if !reflect.DeepEqual(second, first) {
		t.Fatalf("ResolveHotUpdateGateTerminalFailure(second) = %#v, want %#v", second, first)
	}
	secondBytes, err := os.ReadFile(StoreHotUpdateGatePath(root, "hot-update-terminal-replay"))
	if err != nil {
		t.Fatalf("ReadFile(second) error = %v", err)
	}
	if string(firstBytes) != string(secondBytes) {
		t.Fatalf("hot-update gate record file changed on idempotent terminal failure replay\nfirst:\n%s\nsecond:\n%s", string(firstBytes), string(secondBytes))
	}

	_, changed, err = ResolveHotUpdateGateTerminalFailure(root, "hot-update-terminal-replay", "different operator reason", "operator", now.Add(13*time.Minute))
	if err == nil {
		t.Fatal("ResolveHotUpdateGateTerminalFailure(different reason) error = nil, want fail-closed rejection")
	}
	if changed {
		t.Fatal("ResolveHotUpdateGateTerminalFailure(different reason) changed = true, want false")
	}
	if !strings.Contains(err.Error(), "already resolved with failure_reason") {
		t.Fatalf("ResolveHotUpdateGateTerminalFailure(different reason) error = %q, want already-resolved rejection", err.Error())
	}
}

func TestResolveHotUpdateGateTerminalFailureRejectsNonRecoveryNeededStates(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 23, 12, 30, 0, 0, time.UTC)
	tests := []struct {
		name  string
		state HotUpdateGateState
	}{
		{name: "prepared", state: HotUpdateGateStatePrepared},
		{name: "reloading", state: HotUpdateGateStateReloading},
		{name: "reload_apply_in_progress", state: HotUpdateGateStateReloadApplyInProgress},
		{name: "reload_apply_succeeded", state: HotUpdateGateStateReloadApplySucceeded},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			hotUpdateID := "hot-update-terminal-invalid-" + tc.name
			storeHotUpdateRecoveryNeededFixture(t, root, now, hotUpdateID)
			record, err := LoadHotUpdateGateRecord(root, hotUpdateID)
			if err != nil {
				t.Fatalf("LoadHotUpdateGateRecord() error = %v", err)
			}
			record.State = tc.state
			record.FailureReason = ""
			record.PhaseUpdatedAt = now.Add(10 * time.Minute)
			record.PhaseUpdatedBy = "operator"
			if err := StoreHotUpdateGateRecord(root, record); err != nil {
				t.Fatalf("StoreHotUpdateGateRecord(%s) error = %v", tc.state, err)
			}
			beforePointerBytes, err := os.ReadFile(StoreActiveRuntimePackPointerPath(root))
			if err != nil {
				t.Fatalf("ReadFile(active pointer before) error = %v", err)
			}

			got, changed, err := ResolveHotUpdateGateTerminalFailure(root, hotUpdateID, "operator requested stop after recovery review", "operator", now.Add(11*time.Minute))
			if err == nil {
				t.Fatal("ResolveHotUpdateGateTerminalFailure() error = nil, want invalid state rejection")
			}
			if changed {
				t.Fatal("ResolveHotUpdateGateTerminalFailure() changed = true, want false")
			}
			if !reflect.DeepEqual(got, HotUpdateGateRecord{}) {
				t.Fatalf("ResolveHotUpdateGateTerminalFailure() record = %#v, want zero value", got)
			}
			if !strings.Contains(err.Error(), "does not permit terminal failure resolution") {
				t.Fatalf("ResolveHotUpdateGateTerminalFailure() error = %q, want invalid state rejection", err.Error())
			}

			afterPointerBytes, err := os.ReadFile(StoreActiveRuntimePackPointerPath(root))
			if err != nil {
				t.Fatalf("ReadFile(active pointer after) error = %v", err)
			}
			if string(beforePointerBytes) != string(afterPointerBytes) {
				t.Fatalf("active runtime pack pointer file changed after invalid terminal failure rejection\nbefore:\n%s\nafter:\n%s", string(beforePointerBytes), string(afterPointerBytes))
			}
		})
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

func storeHotUpdateReloadInProgressFixture(t *testing.T, root string, now time.Time, hotUpdateID string) {
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
	if _, _, err := EnsureHotUpdateGateRecordFromCandidate(root, hotUpdateID, "pack-candidate", "operator", now.Add(3*time.Minute)); err != nil {
		t.Fatalf("EnsureHotUpdateGateRecordFromCandidate() error = %v", err)
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

	record, err := LoadHotUpdateGateRecord(root, hotUpdateID)
	if err != nil {
		t.Fatalf("LoadHotUpdateGateRecord() error = %v", err)
	}
	record.State = HotUpdateGateStateReloadApplyInProgress
	record.FailureReason = ""
	record.PhaseUpdatedAt = now.Add(7 * time.Minute)
	record.PhaseUpdatedBy = "operator"
	if err := StoreHotUpdateGateRecord(root, record); err != nil {
		t.Fatalf("StoreHotUpdateGateRecord(reload_apply_in_progress) error = %v", err)
	}
}

func storeCandidatePromotionDecisionGateFixture(t *testing.T, root string, now time.Time, withLastKnownGood bool) CandidatePromotionDecisionRecord {
	t.Helper()

	storeCandidatePromotionEligibilityFixtures(t, root, now, nil, nil)
	if err := StoreActiveRuntimePackPointer(root, ActiveRuntimePackPointer{
		ActivePackID:        "pack-base",
		LastKnownGoodPackID: "pack-base",
		UpdatedAt:           now.Add(8 * time.Minute),
		UpdatedBy:           "operator",
		UpdateRecordRef:     "bootstrap",
		ReloadGeneration:    2,
	}); err != nil {
		t.Fatalf("StoreActiveRuntimePackPointer() error = %v", err)
	}
	if withLastKnownGood {
		if err := StoreLastKnownGoodRuntimePackPointer(root, LastKnownGoodRuntimePackPointer{
			PackID:            "pack-base",
			Basis:             "holdout_pass",
			VerifiedAt:        now.Add(8 * time.Minute),
			VerifiedBy:        "operator",
			RollbackRecordRef: "bootstrap",
		}); err != nil {
			t.Fatalf("StoreLastKnownGoodRuntimePackPointer() error = %v", err)
		}
	}
	decision, changed, err := CreateCandidatePromotionDecisionFromEligibleResult(root, "result-eligible", "operator", now.Add(9*time.Minute))
	if err != nil {
		t.Fatalf("CreateCandidatePromotionDecisionFromEligibleResult() error = %v", err)
	}
	if !changed {
		t.Fatal("CreateCandidatePromotionDecisionFromEligibleResult() changed = false, want true")
	}
	return decision
}

func storeHotUpdateRecoveryNeededFixture(t *testing.T, root string, now time.Time, hotUpdateID string) {
	t.Helper()

	storeHotUpdateReloadInProgressFixture(t, root, now, hotUpdateID)
	if _, changed, err := ReconcileHotUpdateGateRecoveryNeeded(root, hotUpdateID, "operator", now.Add(8*time.Minute)); err != nil {
		t.Fatalf("ReconcileHotUpdateGateRecoveryNeeded() error = %v", err)
	} else if !changed {
		t.Fatal("ReconcileHotUpdateGateRecoveryNeeded() changed = false, want true")
	}

	record, err := LoadHotUpdateGateRecord(root, hotUpdateID)
	if err != nil {
		t.Fatalf("LoadHotUpdateGateRecord(recovery-needed) error = %v", err)
	}
	record.FailureReason = "stale retry detail"
	record.PhaseUpdatedAt = now.Add(9 * time.Minute)
	record.PhaseUpdatedBy = "operator"
	if err := StoreHotUpdateGateRecord(root, record); err != nil {
		t.Fatalf("StoreHotUpdateGateRecord(reload_apply_recovery_needed) error = %v", err)
	}
}
