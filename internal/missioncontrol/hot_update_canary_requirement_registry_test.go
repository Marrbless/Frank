package missioncontrol

import (
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestValidateHotUpdateCanaryRequirementRecordRejectsMissingRequiredFields(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 25, 10, 0, 0, 0, time.UTC)
	tests := []struct {
		name string
		edit func(*HotUpdateCanaryRequirementRecord)
		want string
	}{
		{name: "record version", edit: func(record *HotUpdateCanaryRequirementRecord) { record.RecordVersion = 0 }, want: "record_version must be positive"},
		{name: "canary requirement id", edit: func(record *HotUpdateCanaryRequirementRecord) { record.CanaryRequirementID = "" }, want: "canary_requirement_id is required"},
		{name: "result id", edit: func(record *HotUpdateCanaryRequirementRecord) { record.ResultID = "" }, want: "result_id"},
		{name: "run id", edit: func(record *HotUpdateCanaryRequirementRecord) { record.RunID = "" }, want: "run_id"},
		{name: "candidate id", edit: func(record *HotUpdateCanaryRequirementRecord) { record.CandidateID = "" }, want: "candidate_id"},
		{name: "eval suite id", edit: func(record *HotUpdateCanaryRequirementRecord) { record.EvalSuiteID = "" }, want: "eval_suite_id"},
		{name: "promotion policy id", edit: func(record *HotUpdateCanaryRequirementRecord) { record.PromotionPolicyID = "" }, want: "promotion_policy_id"},
		{name: "baseline pack id", edit: func(record *HotUpdateCanaryRequirementRecord) { record.BaselinePackID = "" }, want: "baseline_pack_id"},
		{name: "candidate pack id", edit: func(record *HotUpdateCanaryRequirementRecord) { record.CandidatePackID = "" }, want: "candidate_pack_id"},
		{name: "invalid eligibility state", edit: func(record *HotUpdateCanaryRequirementRecord) {
			record.EligibilityState = CandidatePromotionEligibilityStateEligible
		}, want: "eligibility_state"},
		{name: "required by policy", edit: func(record *HotUpdateCanaryRequirementRecord) { record.RequiredByPolicy = false }, want: "required_by_policy must be true"},
		{name: "state required", edit: func(record *HotUpdateCanaryRequirementRecord) { record.State = "satisfied" }, want: `state must be "required"`},
		{name: "reason", edit: func(record *HotUpdateCanaryRequirementRecord) { record.Reason = "" }, want: "reason is required"},
		{name: "created at", edit: func(record *HotUpdateCanaryRequirementRecord) { record.CreatedAt = time.Time{} }, want: "created_at is required"},
		{name: "created by", edit: func(record *HotUpdateCanaryRequirementRecord) { record.CreatedBy = "" }, want: "created_by is required"},
		{name: "deterministic id mismatch", edit: func(record *HotUpdateCanaryRequirementRecord) {
			record.CanaryRequirementID = "hot-update-canary-requirement-other"
		}, want: "does not match deterministic canary_requirement_id"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			record := validHotUpdateCanaryRequirementRecord(now, nil)
			tt.edit(&record)
			err := ValidateHotUpdateCanaryRequirementRecord(NormalizeHotUpdateCanaryRequirementRecord(record))
			if err == nil {
				t.Fatal("ValidateHotUpdateCanaryRequirementRecord() error = nil, want rejection")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("ValidateHotUpdateCanaryRequirementRecord() error = %q, want substring %q", err.Error(), tt.want)
			}
		})
	}
}

func TestHotUpdateCanaryRequirementIDFromResult(t *testing.T) {
	t.Parallel()

	if got := HotUpdateCanaryRequirementIDFromResult(" result-1 "); got != "hot-update-canary-requirement-result-1" {
		t.Fatalf("HotUpdateCanaryRequirementIDFromResult() = %q, want hot-update-canary-requirement-result-1", got)
	}
}

func TestCreateHotUpdateCanaryRequirementFromCandidateResultCanaryRequired(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 25, 11, 0, 0, 0, time.UTC)
	storeCandidatePromotionEligibilityFixtures(t, root, now, func(record *PromotionPolicyRecord) {
		record.RequiresCanary = true
	}, func(record *CandidateResultRecord) {
		record.ResultID = "result-canary"
	})

	record, changed, err := CreateHotUpdateCanaryRequirementFromCandidateResult(root, " result-canary ", " operator ", now.Add(10*time.Minute))
	if err != nil {
		t.Fatalf("CreateHotUpdateCanaryRequirementFromCandidateResult() error = %v", err)
	}
	if !changed {
		t.Fatal("changed = false, want true")
	}

	want := validHotUpdateCanaryRequirementRecord(now.Add(10*time.Minute), func(record *HotUpdateCanaryRequirementRecord) {
		record.ResultID = "result-canary"
		record.CanaryRequirementID = "hot-update-canary-requirement-result-canary"
		record.CreatedBy = "operator"
	})
	if !reflect.DeepEqual(record, want) {
		t.Fatalf("canary requirement = %#v, want %#v", record, want)
	}

	loaded, err := LoadHotUpdateCanaryRequirementRecord(root, "hot-update-canary-requirement-result-canary")
	if err != nil {
		t.Fatalf("LoadHotUpdateCanaryRequirementRecord() error = %v", err)
	}
	if !reflect.DeepEqual(loaded, want) {
		t.Fatalf("loaded = %#v, want %#v", loaded, want)
	}
}

func TestCreateHotUpdateCanaryRequirementFromCandidateResultCanaryAndOwnerApprovalRequired(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 25, 12, 0, 0, 0, time.UTC)
	storeCandidatePromotionEligibilityFixtures(t, root, now, func(record *PromotionPolicyRecord) {
		record.RequiresCanary = true
		record.RequiresOwnerApproval = true
	}, func(record *CandidateResultRecord) {
		record.ResultID = "result-canary-owner"
	})

	record, changed, err := CreateHotUpdateCanaryRequirementFromCandidateResult(root, "result-canary-owner", "operator", now.Add(10*time.Minute))
	if err != nil {
		t.Fatalf("CreateHotUpdateCanaryRequirementFromCandidateResult() error = %v", err)
	}
	if !changed {
		t.Fatal("changed = false, want true")
	}
	if record.EligibilityState != CandidatePromotionEligibilityStateCanaryAndOwnerApprovalRequired {
		t.Fatalf("EligibilityState = %q, want canary_and_owner_approval_required", record.EligibilityState)
	}
	if !record.OwnerApprovalRequired {
		t.Fatal("OwnerApprovalRequired = false, want true")
	}
}

func TestCreateHotUpdateCanaryRequirementFromCandidateResultRejectsNonCanaryStates(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		resultID   string
		policyEdit func(*PromotionPolicyRecord)
		resultEdit func(*CandidateResultRecord)
		want       string
	}{
		{name: "eligible", resultID: "result-eligible", want: `promotion eligibility state "eligible"`},
		{name: "owner approval required", resultID: "result-owner", policyEdit: func(record *PromotionPolicyRecord) { record.RequiresOwnerApproval = true }, want: `promotion eligibility state "owner_approval_required"`},
		{name: "rejected", resultID: "result-rejected", resultEdit: func(record *CandidateResultRecord) { record.HoldoutScore = record.BaselineScore }, want: `promotion eligibility state "rejected"`},
		{name: "unsupported policy", resultID: "result-unsupported", policyEdit: func(record *PromotionPolicyRecord) { record.EpsilonRule = "epsilon maybe small" }, want: `promotion eligibility state "unsupported_policy"`},
		{name: "invalid", resultID: "result-invalid", resultEdit: func(record *CandidateResultRecord) { record.PromotionPolicyID = "" }, want: `promotion eligibility state "invalid"`},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			now := time.Date(2026, 4, 25, 13, 0, 0, 0, time.UTC)
			storeCandidatePromotionEligibilityFixtures(t, root, now, tt.policyEdit, func(record *CandidateResultRecord) {
				record.ResultID = tt.resultID
				if tt.resultEdit != nil {
					tt.resultEdit(record)
				}
			})

			_, changed, err := CreateHotUpdateCanaryRequirementFromCandidateResult(root, tt.resultID, "operator", now.Add(10*time.Minute))
			if err == nil {
				t.Fatal("CreateHotUpdateCanaryRequirementFromCandidateResult() error = nil, want fail-closed rejection")
			}
			if changed {
				t.Fatal("changed = true on rejected state, want false")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("CreateHotUpdateCanaryRequirementFromCandidateResult() error = %q, want substring %q", err.Error(), tt.want)
			}
			if _, statErr := os.Stat(StoreHotUpdateCanaryRequirementsDir(root)); !os.IsNotExist(statErr) {
				t.Fatalf("canary requirement dir exists or errored after rejection: %v", statErr)
			}
		})
	}
}

func TestCreateHotUpdateCanaryRequirementFromCandidateResultRejectsMissingMetadata(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 25, 14, 0, 0, 0, time.UTC)
	storeCandidatePromotionEligibilityFixtures(t, root, now, func(record *PromotionPolicyRecord) {
		record.RequiresCanary = true
	}, func(record *CandidateResultRecord) {
		record.ResultID = "result-metadata"
	})

	_, changed, err := CreateHotUpdateCanaryRequirementFromCandidateResult(root, "result-metadata", " ", now.Add(10*time.Minute))
	if err == nil {
		t.Fatal("CreateHotUpdateCanaryRequirementFromCandidateResult(missing created_by) error = nil, want rejection")
	}
	if changed {
		t.Fatal("changed = true on missing created_by, want false")
	}
	if !strings.Contains(err.Error(), "created_by is required") {
		t.Fatalf("missing created_by error = %q, want created_by context", err.Error())
	}

	_, changed, err = CreateHotUpdateCanaryRequirementFromCandidateResult(root, "result-metadata", "operator", time.Time{})
	if err == nil {
		t.Fatal("CreateHotUpdateCanaryRequirementFromCandidateResult(zero created_at) error = nil, want rejection")
	}
	if changed {
		t.Fatal("changed = true on zero created_at, want false")
	}
	if !strings.Contains(err.Error(), "created_at is required") {
		t.Fatalf("zero created_at error = %q, want created_at context", err.Error())
	}
}

func TestHotUpdateCanaryRequirementReplayDuplicatesAndListOrder(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 25, 15, 0, 0, 0, time.UTC)
	storeCandidatePromotionEligibilityFixtures(t, root, now, func(record *PromotionPolicyRecord) {
		record.RequiresCanary = true
	}, func(record *CandidateResultRecord) {
		record.ResultID = "result-b"
	})

	record, changed, err := CreateHotUpdateCanaryRequirementFromCandidateResult(root, "result-b", "operator", now.Add(10*time.Minute))
	if err != nil {
		t.Fatalf("CreateHotUpdateCanaryRequirementFromCandidateResult(first) error = %v", err)
	}
	if !changed {
		t.Fatal("first changed = false, want true")
	}
	firstBytes, err := os.ReadFile(StoreHotUpdateCanaryRequirementPath(root, record.CanaryRequirementID))
	if err != nil {
		t.Fatalf("ReadFile(first) error = %v", err)
	}

	replayed, changed, err := CreateHotUpdateCanaryRequirementFromCandidateResult(root, "result-b", "operator", now.Add(10*time.Minute))
	if err != nil {
		t.Fatalf("CreateHotUpdateCanaryRequirementFromCandidateResult(replay) error = %v", err)
	}
	if changed {
		t.Fatal("replay changed = true, want false")
	}
	if !reflect.DeepEqual(replayed, record) {
		t.Fatalf("replayed = %#v, want %#v", replayed, record)
	}
	secondBytes, err := os.ReadFile(StoreHotUpdateCanaryRequirementPath(root, record.CanaryRequirementID))
	if err != nil {
		t.Fatalf("ReadFile(second) error = %v", err)
	}
	if string(firstBytes) != string(secondBytes) {
		t.Fatalf("hot-update canary requirement file changed on idempotent replay")
	}

	divergent := record
	divergent.Reason = "different reason"
	err = StoreHotUpdateCanaryRequirementRecord(root, divergent)
	if err == nil {
		t.Fatal("StoreHotUpdateCanaryRequirementRecord(divergent) error = nil, want duplicate rejection")
	}
	if !strings.Contains(err.Error(), `hot-update canary requirement "hot-update-canary-requirement-result-b" already exists`) {
		t.Fatalf("StoreHotUpdateCanaryRequirementRecord(divergent) error = %q, want duplicate context", err.Error())
	}

	otherID := record
	otherID.CanaryRequirementID = "hot-update-canary-requirement-other"
	err = StoreHotUpdateCanaryRequirementRecord(root, otherID)
	if err == nil {
		t.Fatal("StoreHotUpdateCanaryRequirementRecord(second result requirement) error = nil, want same-result rejection")
	}

	storeSecondCanaryResult(t, root, now)
	if _, _, err := CreateHotUpdateCanaryRequirementFromCandidateResult(root, "result-a", "operator", now.Add(11*time.Minute)); err != nil {
		t.Fatalf("CreateHotUpdateCanaryRequirementFromCandidateResult(result-a) error = %v", err)
	}
	records, err := ListHotUpdateCanaryRequirementRecords(root)
	if err != nil {
		t.Fatalf("ListHotUpdateCanaryRequirementRecords() error = %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("ListHotUpdateCanaryRequirementRecords() len = %d, want 2", len(records))
	}
	if records[0].CanaryRequirementID != "hot-update-canary-requirement-result-a" || records[1].CanaryRequirementID != "hot-update-canary-requirement-result-b" {
		t.Fatalf("ListHotUpdateCanaryRequirementRecords() order = %#v, want result-a then result-b", records)
	}
}

func TestCreateHotUpdateCanaryRequirementFromCandidateResultFailsClosedForMissingLinkedRecords(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		removePath func(root string) string
		want       string
	}{
		{name: "result", removePath: func(root string) string { return StoreCandidateResultPath(root, "result-canary") }, want: "candidate result record not found"},
		{name: "run", removePath: func(root string) string { return StoreImprovementRunPath(root, "run-result") }, want: "run_id"},
		{name: "candidate", removePath: func(root string) string { return StoreImprovementCandidatePath(root, "candidate-1") }, want: "candidate_id"},
		{name: "eval suite", removePath: func(root string) string { return StoreEvalSuitePath(root, "eval-suite-1") }, want: "eval_suite_id"},
		{name: "promotion policy", removePath: func(root string) string { return StorePromotionPolicyPath(root, "promotion-policy-result") }, want: "promotion_policy_id"},
		{name: "baseline pack", removePath: func(root string) string { return StoreRuntimePackPath(root, "pack-base") }, want: "baseline_pack_id"},
		{name: "candidate pack", removePath: func(root string) string { return StoreRuntimePackPath(root, "pack-candidate") }, want: "candidate_pack_id"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			now := time.Date(2026, 4, 25, 16, 0, 0, 0, time.UTC)
			storeCandidatePromotionEligibilityFixtures(t, root, now, func(record *PromotionPolicyRecord) {
				record.RequiresCanary = true
			}, func(record *CandidateResultRecord) {
				record.ResultID = "result-canary"
			})
			if err := os.Remove(tt.removePath(root)); err != nil {
				t.Fatalf("Remove() error = %v", err)
			}

			_, changed, err := CreateHotUpdateCanaryRequirementFromCandidateResult(root, "result-canary", "operator", now.Add(10*time.Minute))
			if err == nil {
				t.Fatal("CreateHotUpdateCanaryRequirementFromCandidateResult() error = nil, want missing linkage rejection")
			}
			if changed {
				t.Fatal("changed = true on missing linkage, want false")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("CreateHotUpdateCanaryRequirementFromCandidateResult() error = %q, want substring %q", err.Error(), tt.want)
			}
		})
	}
}

func TestHotUpdateCanaryRequirementReplayFailsClosedWhenEligibilityChanges(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 25, 17, 0, 0, 0, time.UTC)
	storeCandidatePromotionEligibilityFixtures(t, root, now, func(record *PromotionPolicyRecord) {
		record.RequiresCanary = true
	}, func(record *CandidateResultRecord) {
		record.ResultID = "result-canary"
	})
	if _, _, err := CreateHotUpdateCanaryRequirementFromCandidateResult(root, "result-canary", "operator", now.Add(10*time.Minute)); err != nil {
		t.Fatalf("CreateHotUpdateCanaryRequirementFromCandidateResult(first) error = %v", err)
	}

	if err := WriteStoreJSONAtomic(StorePromotionPolicyPath(root, "promotion-policy-result"), validPromotionPolicyRecord(now.Add(6*time.Minute), func(record *PromotionPolicyRecord) {
		record.RecordVersion = StoreRecordVersion
		record.PromotionPolicyID = "promotion-policy-result"
		record.RequiresCanary = false
		record.RequiresOwnerApproval = false
		record.RequiresHoldoutPass = true
		record.EpsilonRule = "epsilon <= 0.01"
		record.RegressionRule = "no_regression_flags"
		record.CompatibilityRule = "compatibility_score >= 0.90"
		record.ResourceRule = "resource_score >= 0.60"
	})); err != nil {
		t.Fatalf("WriteStoreJSONAtomic(policy) error = %v", err)
	}

	_, changed, err := CreateHotUpdateCanaryRequirementFromCandidateResult(root, "result-canary", "operator", now.Add(10*time.Minute))
	if err == nil {
		t.Fatal("CreateHotUpdateCanaryRequirementFromCandidateResult(replay after policy drift) error = nil, want fail-closed rejection")
	}
	if changed {
		t.Fatal("changed = true after eligibility drift, want false")
	}
	if !strings.Contains(err.Error(), `promotion eligibility state "eligible"`) {
		t.Fatalf("CreateHotUpdateCanaryRequirementFromCandidateResult() error = %q, want eligible drift context", err.Error())
	}
}

func TestHotUpdateCanaryRequirementStoreDoesNotMutateRuntimeSurfaces(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 25, 18, 0, 0, 0, time.UTC)
	storeCandidatePromotionEligibilityFixtures(t, root, now, func(record *PromotionPolicyRecord) {
		record.RequiresCanary = true
	}, func(record *CandidateResultRecord) {
		record.ResultID = "result-no-mutation"
	})

	snapshots := map[string][]byte{}
	for _, path := range []string{
		StoreRuntimePackPath(root, "pack-base"),
		StoreRuntimePackPath(root, "pack-candidate"),
		StoreImprovementCandidatePath(root, "candidate-1"),
		StoreEvalSuitePath(root, "eval-suite-1"),
		StoreImprovementRunPath(root, "run-result"),
		StoreCandidateResultPath(root, "result-no-mutation"),
		StorePromotionPolicyPath(root, "promotion-policy-result"),
		StoreHotUpdateGatePath(root, "hot-update-1"),
	} {
		bytes, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ReadFile(%s) error = %v", path, err)
		}
		snapshots[path] = bytes
	}

	if _, _, err := CreateHotUpdateCanaryRequirementFromCandidateResult(root, "result-no-mutation", "operator", now.Add(10*time.Minute)); err != nil {
		t.Fatalf("CreateHotUpdateCanaryRequirementFromCandidateResult() error = %v", err)
	}

	for path, before := range snapshots {
		after, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ReadFile(%s) after canary requirement error = %v", path, err)
		}
		if string(after) != string(before) {
			t.Fatalf("linked record %s changed after hot-update canary requirement creation", path)
		}
	}

	absentPaths := []string{
		StoreCandidatePromotionDecisionsDir(root),
		StoreHotUpdateOutcomesDir(root),
		StorePromotionsDir(root),
		StoreRollbacksDir(root),
		StoreRollbackAppliesDir(root),
		StoreActiveRuntimePackPointerPath(root),
		StoreLastKnownGoodRuntimePackPointerPath(root),
	}
	for _, path := range absentPaths {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("path %s exists or errored after hot-update canary requirement creation: %v", path, err)
		}
	}
}

func validHotUpdateCanaryRequirementRecord(now time.Time, mutate func(*HotUpdateCanaryRequirementRecord)) HotUpdateCanaryRequirementRecord {
	record := HotUpdateCanaryRequirementRecord{
		RecordVersion:         StoreRecordVersion,
		CanaryRequirementID:   "hot-update-canary-requirement-result-root",
		ResultID:              "result-root",
		RunID:                 "run-result",
		CandidateID:           "candidate-1",
		EvalSuiteID:           "eval-suite-1",
		PromotionPolicyID:     "promotion-policy-result",
		BaselinePackID:        "pack-base",
		CandidatePackID:       "pack-candidate",
		EligibilityState:      CandidatePromotionEligibilityStateCanaryRequired,
		RequiredByPolicy:      true,
		OwnerApprovalRequired: false,
		State:                 HotUpdateCanaryRequirementStateRequired,
		Reason:                "candidate result requires canary before promotion",
		CreatedAt:             now.UTC(),
		CreatedBy:             "operator",
	}
	if mutate != nil {
		mutate(&record)
	}
	return record
}

func storeSecondCanaryResult(t *testing.T, root string, now time.Time) {
	t.Helper()

	if err := StoreImprovementRunRecord(root, validImprovementRunRecord(now.Add(12*time.Minute), func(record *ImprovementRunRecord) {
		record.RunID = "run-result-a"
	})); err != nil {
		t.Fatalf("StoreImprovementRunRecord(run-result-a) error = %v", err)
	}
	if err := StoreCandidateResultRecord(root, validCandidateResultRecord(now.Add(13*time.Minute), func(record *CandidateResultRecord) {
		record.ResultID = "result-a"
		record.RunID = "run-result-a"
		record.PromotionPolicyID = "promotion-policy-result"
		record.BaselineScore = 0.52
		record.TrainScore = 0.78
		record.HoldoutScore = 0.74
		record.CompatibilityScore = 0.93
		record.ResourceScore = 0.67
		record.RegressionFlags = []string{"none"}
		record.Decision = ImprovementRunDecisionKeep
	})); err != nil {
		t.Fatalf("StoreCandidateResultRecord(result-a) error = %v", err)
	}
}
