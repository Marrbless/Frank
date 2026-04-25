package missioncontrol

import (
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestValidateHotUpdateCanaryEvidenceRecordRejectsMissingRequiredFields(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 25, 10, 11, 12, 123456789, time.UTC)
	tests := []struct {
		name string
		edit func(*HotUpdateCanaryEvidenceRecord)
		want string
	}{
		{name: "record version", edit: func(record *HotUpdateCanaryEvidenceRecord) { record.RecordVersion = 0 }, want: "record_version must be positive"},
		{name: "canary evidence id", edit: func(record *HotUpdateCanaryEvidenceRecord) { record.CanaryEvidenceID = "" }, want: "canary_evidence_id is required"},
		{name: "canary requirement id", edit: func(record *HotUpdateCanaryEvidenceRecord) { record.CanaryRequirementID = "" }, want: "canary_requirement_id"},
		{name: "result id", edit: func(record *HotUpdateCanaryEvidenceRecord) { record.ResultID = "" }, want: "result_id"},
		{name: "run id", edit: func(record *HotUpdateCanaryEvidenceRecord) { record.RunID = "" }, want: "run_id"},
		{name: "candidate id", edit: func(record *HotUpdateCanaryEvidenceRecord) { record.CandidateID = "" }, want: "candidate_id"},
		{name: "eval suite id", edit: func(record *HotUpdateCanaryEvidenceRecord) { record.EvalSuiteID = "" }, want: "eval_suite_id"},
		{name: "promotion policy id", edit: func(record *HotUpdateCanaryEvidenceRecord) { record.PromotionPolicyID = "" }, want: "promotion_policy_id"},
		{name: "baseline pack id", edit: func(record *HotUpdateCanaryEvidenceRecord) { record.BaselinePackID = "" }, want: "baseline_pack_id"},
		{name: "candidate pack id", edit: func(record *HotUpdateCanaryEvidenceRecord) { record.CandidatePackID = "" }, want: "candidate_pack_id"},
		{name: "invalid evidence state", edit: func(record *HotUpdateCanaryEvidenceRecord) { record.EvidenceState = "waived" }, want: "evidence_state"},
		{name: "passed true only for passed", edit: func(record *HotUpdateCanaryEvidenceRecord) {
			record.EvidenceState = HotUpdateCanaryEvidenceStateFailed
			record.Passed = true
		}, want: "passed must be true only"},
		{name: "passed required for passed", edit: func(record *HotUpdateCanaryEvidenceRecord) { record.Passed = false }, want: "passed must be true only"},
		{name: "reason", edit: func(record *HotUpdateCanaryEvidenceRecord) { record.Reason = "" }, want: "reason is required"},
		{name: "observed at", edit: func(record *HotUpdateCanaryEvidenceRecord) { record.ObservedAt = time.Time{} }, want: "observed_at is required"},
		{name: "created at", edit: func(record *HotUpdateCanaryEvidenceRecord) { record.CreatedAt = time.Time{} }, want: "created_at is required"},
		{name: "created by", edit: func(record *HotUpdateCanaryEvidenceRecord) { record.CreatedBy = "" }, want: "created_by is required"},
		{name: "deterministic id mismatch", edit: func(record *HotUpdateCanaryEvidenceRecord) {
			record.CanaryEvidenceID = "hot-update-canary-evidence-other"
		}, want: "does not match deterministic canary_evidence_id"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			record := validHotUpdateCanaryEvidenceRecord(now, nil)
			tt.edit(&record)
			err := ValidateHotUpdateCanaryEvidenceRecord(NormalizeHotUpdateCanaryEvidenceRecord(record))
			if err == nil {
				t.Fatal("ValidateHotUpdateCanaryEvidenceRecord() error = nil, want rejection")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("ValidateHotUpdateCanaryEvidenceRecord() error = %q, want substring %q", err.Error(), tt.want)
			}
		})
	}
}

func TestHotUpdateCanaryEvidenceIDFromRequirementObservedAt(t *testing.T) {
	t.Parallel()

	observedAt := time.Date(2026, 4, 25, 10, 11, 12, 123456789, time.FixedZone("offset", -4*60*60))
	got := HotUpdateCanaryEvidenceIDFromRequirementObservedAt(" hot-update-canary-requirement-result-1 ", observedAt)
	want := "hot-update-canary-evidence-hot-update-canary-requirement-result-1-20260425T141112123456789Z"
	if got != want {
		t.Fatalf("HotUpdateCanaryEvidenceIDFromRequirementObservedAt() = %q, want %q", got, want)
	}
}

func TestCreateHotUpdateCanaryEvidenceFromRequirementCreatesAllStates(t *testing.T) {
	t.Parallel()

	tests := []struct {
		state  HotUpdateCanaryEvidenceState
		passed bool
	}{
		{state: HotUpdateCanaryEvidenceStatePassed, passed: true},
		{state: HotUpdateCanaryEvidenceStateFailed, passed: false},
		{state: HotUpdateCanaryEvidenceStateBlocked, passed: false},
		{state: HotUpdateCanaryEvidenceStateExpired, passed: false},
	}

	for i, tt := range tests {
		tt := tt
		i := i
		t.Run(string(tt.state), func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			now := time.Date(2026, 4, 25, 11+i, 0, 0, 0, time.UTC)
			requirement := storeCanaryRequirementForEvidence(t, root, now, nil, func(record *CandidateResultRecord) {
				record.ResultID = "result-" + string(tt.state)
			})
			observedAt := now.Add(20 * time.Minute)
			createdAt := now.Add(21 * time.Minute)

			record, changed, err := CreateHotUpdateCanaryEvidenceFromRequirement(root, " "+requirement.CanaryRequirementID+" ", tt.state, observedAt, " operator ", createdAt, " canary observation recorded ")
			if err != nil {
				t.Fatalf("CreateHotUpdateCanaryEvidenceFromRequirement() error = %v", err)
			}
			if !changed {
				t.Fatal("changed = false, want true")
			}
			if record.CanaryRequirementID != requirement.CanaryRequirementID {
				t.Fatalf("CanaryRequirementID = %q, want %q", record.CanaryRequirementID, requirement.CanaryRequirementID)
			}
			if record.ResultID != requirement.ResultID || record.RunID != requirement.RunID || record.CandidateID != requirement.CandidateID ||
				record.EvalSuiteID != requirement.EvalSuiteID || record.PromotionPolicyID != requirement.PromotionPolicyID ||
				record.BaselinePackID != requirement.BaselinePackID || record.CandidatePackID != requirement.CandidatePackID {
				t.Fatalf("evidence refs = %#v, want copied from requirement %#v", record, requirement)
			}
			if record.EvidenceState != tt.state {
				t.Fatalf("EvidenceState = %q, want %q", record.EvidenceState, tt.state)
			}
			if record.Passed != tt.passed {
				t.Fatalf("Passed = %v, want %v", record.Passed, tt.passed)
			}
			if record.Reason != "canary observation recorded" {
				t.Fatalf("Reason = %q, want trimmed reason", record.Reason)
			}
			if !record.ObservedAt.Equal(observedAt) || !record.CreatedAt.Equal(createdAt) {
				t.Fatalf("timestamps = %v/%v, want %v/%v", record.ObservedAt, record.CreatedAt, observedAt, createdAt)
			}
			if record.CreatedBy != "operator" {
				t.Fatalf("CreatedBy = %q, want operator", record.CreatedBy)
			}
			loaded, err := LoadHotUpdateCanaryEvidenceRecord(root, record.CanaryEvidenceID)
			if err != nil {
				t.Fatalf("LoadHotUpdateCanaryEvidenceRecord() error = %v", err)
			}
			if !reflect.DeepEqual(loaded, record) {
				t.Fatalf("loaded = %#v, want %#v", loaded, record)
			}
		})
	}
}

func TestCreateHotUpdateCanaryEvidenceFromRequirementLoadsAndCrossChecksLinkedRecords(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		removePath func(root string, requirement HotUpdateCanaryRequirementRecord) string
		want       string
	}{
		{name: "requirement", removePath: func(root string, requirement HotUpdateCanaryRequirementRecord) string {
			return StoreHotUpdateCanaryRequirementPath(root, requirement.CanaryRequirementID)
		}, want: "hot-update canary requirement record not found"},
		{name: "result", removePath: func(root string, requirement HotUpdateCanaryRequirementRecord) string {
			return StoreCandidateResultPath(root, requirement.ResultID)
		}, want: "candidate result record not found"},
		{name: "run", removePath: func(root string, requirement HotUpdateCanaryRequirementRecord) string {
			return StoreImprovementRunPath(root, requirement.RunID)
		}, want: "run_id"},
		{name: "candidate", removePath: func(root string, requirement HotUpdateCanaryRequirementRecord) string {
			return StoreImprovementCandidatePath(root, requirement.CandidateID)
		}, want: "candidate_id"},
		{name: "eval suite", removePath: func(root string, requirement HotUpdateCanaryRequirementRecord) string {
			return StoreEvalSuitePath(root, requirement.EvalSuiteID)
		}, want: "eval_suite_id"},
		{name: "promotion policy", removePath: func(root string, requirement HotUpdateCanaryRequirementRecord) string {
			return StorePromotionPolicyPath(root, requirement.PromotionPolicyID)
		}, want: "promotion_policy_id"},
		{name: "baseline pack", removePath: func(root string, requirement HotUpdateCanaryRequirementRecord) string {
			return StoreRuntimePackPath(root, requirement.BaselinePackID)
		}, want: "baseline_pack_id"},
		{name: "candidate pack", removePath: func(root string, requirement HotUpdateCanaryRequirementRecord) string {
			return StoreRuntimePackPath(root, requirement.CandidatePackID)
		}, want: "candidate_pack_id"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			now := time.Date(2026, 4, 25, 15, 0, 0, 0, time.UTC)
			requirement := storeCanaryRequirementForEvidence(t, root, now, nil, func(record *CandidateResultRecord) {
				record.ResultID = "result-linked-" + strings.ReplaceAll(tt.name, " ", "-")
			})
			if err := os.Remove(tt.removePath(root, requirement)); err != nil {
				t.Fatalf("Remove() error = %v", err)
			}

			_, changed, err := CreateHotUpdateCanaryEvidenceFromRequirement(root, requirement.CanaryRequirementID, HotUpdateCanaryEvidenceStatePassed, now.Add(20*time.Minute), "operator", now.Add(21*time.Minute), "canary passed")
			if err == nil {
				t.Fatal("CreateHotUpdateCanaryEvidenceFromRequirement() error = nil, want fail-closed linkage rejection")
			}
			if changed {
				t.Fatal("changed = true on linkage rejection, want false")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("CreateHotUpdateCanaryEvidenceFromRequirement() error = %q, want substring %q", err.Error(), tt.want)
			}
		})
	}
}

func TestCreateHotUpdateCanaryEvidenceFromRequirementRejectsInvalidRequirementAndStateDrift(t *testing.T) {
	t.Parallel()

	t.Run("invalid requirement", func(t *testing.T) {
		t.Parallel()

		root := t.TempDir()
		now := time.Date(2026, 4, 25, 16, 0, 0, 0, time.UTC)
		requirement := storeCanaryRequirementForEvidence(t, root, now, nil, func(record *CandidateResultRecord) {
			record.ResultID = "result-invalid-requirement"
		})
		requirement.Reason = ""
		if err := WriteStoreJSONAtomic(StoreHotUpdateCanaryRequirementPath(root, requirement.CanaryRequirementID), requirement); err != nil {
			t.Fatalf("WriteStoreJSONAtomic(invalid requirement) error = %v", err)
		}

		_, changed, err := CreateHotUpdateCanaryEvidenceFromRequirement(root, requirement.CanaryRequirementID, HotUpdateCanaryEvidenceStatePassed, now.Add(20*time.Minute), "operator", now.Add(21*time.Minute), "canary passed")
		if err == nil {
			t.Fatal("CreateHotUpdateCanaryEvidenceFromRequirement() error = nil, want invalid requirement rejection")
		}
		if changed {
			t.Fatal("changed = true on invalid requirement, want false")
		}
		if !strings.Contains(err.Error(), "reason is required") {
			t.Fatalf("error = %q, want invalid requirement reason context", err.Error())
		}
	})

	t.Run("requirement not required", func(t *testing.T) {
		t.Parallel()

		root := t.TempDir()
		now := time.Date(2026, 4, 25, 17, 0, 0, 0, time.UTC)
		requirement := storeCanaryRequirementForEvidence(t, root, now, nil, func(record *CandidateResultRecord) {
			record.ResultID = "result-requirement-state"
		})
		requirement.State = HotUpdateCanaryRequirementState("satisfied")
		if err := WriteStoreJSONAtomic(StoreHotUpdateCanaryRequirementPath(root, requirement.CanaryRequirementID), requirement); err != nil {
			t.Fatalf("WriteStoreJSONAtomic(requirement state) error = %v", err)
		}

		_, changed, err := CreateHotUpdateCanaryEvidenceFromRequirement(root, requirement.CanaryRequirementID, HotUpdateCanaryEvidenceStatePassed, now.Add(20*time.Minute), "operator", now.Add(21*time.Minute), "canary passed")
		if err == nil {
			t.Fatal("CreateHotUpdateCanaryEvidenceFromRequirement() error = nil, want state rejection")
		}
		if changed {
			t.Fatal("changed = true on state rejection, want false")
		}
		if !strings.Contains(err.Error(), `state must be "required"`) {
			t.Fatalf("error = %q, want required state context", err.Error())
		}
	})

	t.Run("eligibility drift", func(t *testing.T) {
		t.Parallel()

		root := t.TempDir()
		now := time.Date(2026, 4, 25, 18, 0, 0, 0, time.UTC)
		requirement := storeCanaryRequirementForEvidence(t, root, now, nil, func(record *CandidateResultRecord) {
			record.ResultID = "result-drift"
		})
		if err := WriteStoreJSONAtomic(StorePromotionPolicyPath(root, requirement.PromotionPolicyID), validPromotionPolicyRecord(now.Add(30*time.Minute), func(record *PromotionPolicyRecord) {
			record.RecordVersion = StoreRecordVersion
			record.PromotionPolicyID = requirement.PromotionPolicyID
			record.RequiresCanary = false
			record.RequiresOwnerApproval = false
			record.RequiresHoldoutPass = true
			record.EpsilonRule = "epsilon <= 0.01"
			record.RegressionRule = "no_regression_flags"
			record.CompatibilityRule = "compatibility_score >= 0.90"
			record.ResourceRule = "resource_score >= 0.60"
		})); err != nil {
			t.Fatalf("WriteStoreJSONAtomic(policy drift) error = %v", err)
		}

		_, changed, err := CreateHotUpdateCanaryEvidenceFromRequirement(root, requirement.CanaryRequirementID, HotUpdateCanaryEvidenceStatePassed, now.Add(20*time.Minute), "operator", now.Add(21*time.Minute), "canary passed")
		if err == nil {
			t.Fatal("CreateHotUpdateCanaryEvidenceFromRequirement() error = nil, want eligibility drift rejection")
		}
		if changed {
			t.Fatal("changed = true on eligibility drift, want false")
		}
		if !strings.Contains(err.Error(), `promotion eligibility state "eligible"`) {
			t.Fatalf("error = %q, want eligible drift context", err.Error())
		}
	})
}

func TestHotUpdateCanaryEvidenceReplayDuplicatesMultipleAttemptsAndListOrder(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 25, 19, 0, 0, 0, time.UTC)
	requirement := storeCanaryRequirementForEvidence(t, root, now, nil, func(record *CandidateResultRecord) {
		record.ResultID = "result-replay"
	})
	observedAt := now.Add(20 * time.Minute)
	createdAt := now.Add(21 * time.Minute)

	record, changed, err := CreateHotUpdateCanaryEvidenceFromRequirement(root, requirement.CanaryRequirementID, HotUpdateCanaryEvidenceStatePassed, observedAt, "operator", createdAt, "canary passed")
	if err != nil {
		t.Fatalf("CreateHotUpdateCanaryEvidenceFromRequirement(first) error = %v", err)
	}
	if !changed {
		t.Fatal("first changed = false, want true")
	}
	firstBytes, err := os.ReadFile(StoreHotUpdateCanaryEvidencePath(root, record.CanaryEvidenceID))
	if err != nil {
		t.Fatalf("ReadFile(first) error = %v", err)
	}

	replayed, changed, err := CreateHotUpdateCanaryEvidenceFromRequirement(root, requirement.CanaryRequirementID, HotUpdateCanaryEvidenceStatePassed, observedAt, "operator", createdAt, "canary passed")
	if err != nil {
		t.Fatalf("CreateHotUpdateCanaryEvidenceFromRequirement(replay) error = %v", err)
	}
	if changed {
		t.Fatal("replay changed = true, want false")
	}
	if !reflect.DeepEqual(replayed, record) {
		t.Fatalf("replayed = %#v, want %#v", replayed, record)
	}
	secondBytes, err := os.ReadFile(StoreHotUpdateCanaryEvidencePath(root, record.CanaryEvidenceID))
	if err != nil {
		t.Fatalf("ReadFile(second) error = %v", err)
	}
	if string(firstBytes) != string(secondBytes) {
		t.Fatal("hot-update canary evidence file changed on idempotent replay")
	}

	_, changed, err = CreateHotUpdateCanaryEvidenceFromRequirement(root, requirement.CanaryRequirementID, HotUpdateCanaryEvidenceStateFailed, observedAt, "operator", createdAt, "canary failed")
	if err == nil {
		t.Fatal("CreateHotUpdateCanaryEvidenceFromRequirement(divergent) error = nil, want duplicate rejection")
	}
	if changed {
		t.Fatal("changed = true on divergent duplicate, want false")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("divergent error = %q, want duplicate context", err.Error())
	}

	later, changed, err := CreateHotUpdateCanaryEvidenceFromRequirement(root, requirement.CanaryRequirementID, HotUpdateCanaryEvidenceStateFailed, observedAt.Add(time.Minute), "operator", createdAt.Add(time.Minute), "canary failed")
	if err != nil {
		t.Fatalf("CreateHotUpdateCanaryEvidenceFromRequirement(second attempt) error = %v", err)
	}
	if !changed {
		t.Fatal("second attempt changed = false, want true")
	}

	records, err := ListHotUpdateCanaryEvidenceRecords(root)
	if err != nil {
		t.Fatalf("ListHotUpdateCanaryEvidenceRecords() error = %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("ListHotUpdateCanaryEvidenceRecords() len = %d, want 2", len(records))
	}
	if records[0].CanaryEvidenceID != record.CanaryEvidenceID || records[1].CanaryEvidenceID != later.CanaryEvidenceID {
		t.Fatalf("ListHotUpdateCanaryEvidenceRecords() order = %#v, want first attempt then later attempt", records)
	}
}

func TestHotUpdateCanaryEvidenceStoreDoesNotMutateSourceOrRuntimeSurfaces(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 25, 20, 0, 0, 0, time.UTC)
	requirement := storeCanaryRequirementForEvidence(t, root, now, nil, func(record *CandidateResultRecord) {
		record.ResultID = "result-no-mutation"
	})

	snapshots := map[string][]byte{}
	for _, path := range []string{
		StoreRuntimePackPath(root, requirement.BaselinePackID),
		StoreRuntimePackPath(root, requirement.CandidatePackID),
		StoreImprovementCandidatePath(root, requirement.CandidateID),
		StoreEvalSuitePath(root, requirement.EvalSuiteID),
		StoreImprovementRunPath(root, requirement.RunID),
		StoreCandidateResultPath(root, requirement.ResultID),
		StorePromotionPolicyPath(root, requirement.PromotionPolicyID),
		StoreHotUpdateGatePath(root, "hot-update-1"),
		StoreHotUpdateCanaryRequirementPath(root, requirement.CanaryRequirementID),
	} {
		bytes, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ReadFile(%s) error = %v", path, err)
		}
		snapshots[path] = bytes
	}

	if _, _, err := CreateHotUpdateCanaryEvidenceFromRequirement(root, requirement.CanaryRequirementID, HotUpdateCanaryEvidenceStatePassed, now.Add(20*time.Minute), "operator", now.Add(21*time.Minute), "canary passed"); err != nil {
		t.Fatalf("CreateHotUpdateCanaryEvidenceFromRequirement() error = %v", err)
	}

	for path, before := range snapshots {
		after, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ReadFile(%s) after evidence error = %v", path, err)
		}
		if string(after) != string(before) {
			t.Fatalf("source record %s changed after hot-update canary evidence creation", path)
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
			t.Fatalf("path %s exists or errored after hot-update canary evidence creation: %v", path, err)
		}
	}
}

func validHotUpdateCanaryEvidenceRecord(observedAt time.Time, mutate func(*HotUpdateCanaryEvidenceRecord)) HotUpdateCanaryEvidenceRecord {
	requirementID := "hot-update-canary-requirement-result-root"
	record := HotUpdateCanaryEvidenceRecord{
		RecordVersion:       StoreRecordVersion,
		CanaryEvidenceID:    HotUpdateCanaryEvidenceIDFromRequirementObservedAt(requirementID, observedAt),
		CanaryRequirementID: requirementID,
		ResultID:            "result-root",
		RunID:               "run-result",
		CandidateID:         "candidate-1",
		EvalSuiteID:         "eval-suite-1",
		PromotionPolicyID:   "promotion-policy-result",
		BaselinePackID:      "pack-base",
		CandidatePackID:     "pack-candidate",
		EvidenceState:       HotUpdateCanaryEvidenceStatePassed,
		Passed:              true,
		Reason:              "canary passed",
		ObservedAt:          observedAt.UTC(),
		CreatedAt:           observedAt.Add(time.Minute).UTC(),
		CreatedBy:           "operator",
	}
	if mutate != nil {
		mutate(&record)
	}
	return record
}

func storeCanaryRequirementForEvidence(t *testing.T, root string, now time.Time, policyEdit func(*PromotionPolicyRecord), resultEdit func(*CandidateResultRecord)) HotUpdateCanaryRequirementRecord {
	t.Helper()

	storeCandidatePromotionEligibilityFixtures(t, root, now, func(record *PromotionPolicyRecord) {
		record.RequiresCanary = true
		if policyEdit != nil {
			policyEdit(record)
		}
	}, resultEdit)
	resultID := "result-eligible"
	if resultEdit != nil {
		result, err := ListCandidateResultRecords(root)
		if err != nil {
			t.Fatalf("ListCandidateResultRecords() error = %v", err)
		}
		if len(result) != 1 {
			t.Fatalf("ListCandidateResultRecords() len = %d, want 1", len(result))
		}
		resultID = result[0].ResultID
	}
	requirement, _, err := CreateHotUpdateCanaryRequirementFromCandidateResult(root, resultID, "operator", now.Add(10*time.Minute))
	if err != nil {
		t.Fatalf("CreateHotUpdateCanaryRequirementFromCandidateResult() error = %v", err)
	}
	return requirement
}
