package missioncontrol

import (
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestCreateCandidatePromotionDecisionFromEligibleResultStoresLoadsAndLists(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 25, 10, 0, 0, 0, time.FixedZone("offset", -4*60*60))
	storeCandidatePromotionEligibilityFixtures(t, root, now, nil, func(record *CandidateResultRecord) {
		record.ResultID = "result-eligible"
	})

	record, changed, err := CreateCandidatePromotionDecisionFromEligibleResult(root, "result-eligible", " operator ", now.Add(10*time.Minute))
	if err != nil {
		t.Fatalf("CreateCandidatePromotionDecisionFromEligibleResult() error = %v", err)
	}
	if !changed {
		t.Fatal("changed = false, want true")
	}

	want := CandidatePromotionDecisionRecord{
		RecordVersion:       StoreRecordVersion,
		PromotionDecisionID: "candidate-promotion-decision-result-eligible",
		ResultID:            "result-eligible",
		RunID:               "run-result",
		CandidateID:         "candidate-1",
		EvalSuiteID:         "eval-suite-1",
		PromotionPolicyID:   "promotion-policy-result",
		BaselinePackID:      "pack-base",
		CandidatePackID:     "pack-candidate",
		EligibilityState:    CandidatePromotionEligibilityStateEligible,
		Decision:            CandidatePromotionDecisionSelectedForPromotion,
		Reason:              "candidate result eligible for promotion",
		CreatedAt:           now.Add(10 * time.Minute).UTC(),
		CreatedBy:           "operator",
	}
	if !reflect.DeepEqual(record, want) {
		t.Fatalf("decision record = %#v, want %#v", record, want)
	}

	loaded, err := LoadCandidatePromotionDecisionRecord(root, "candidate-promotion-decision-result-eligible")
	if err != nil {
		t.Fatalf("LoadCandidatePromotionDecisionRecord() error = %v", err)
	}
	if !reflect.DeepEqual(loaded, want) {
		t.Fatalf("loaded record = %#v, want %#v", loaded, want)
	}

	records, err := ListCandidatePromotionDecisionRecords(root)
	if err != nil {
		t.Fatalf("ListCandidatePromotionDecisionRecords() error = %v", err)
	}
	if len(records) != 1 || records[0].PromotionDecisionID != "candidate-promotion-decision-result-eligible" {
		t.Fatalf("ListCandidatePromotionDecisionRecords() = %#v, want one deterministic decision", records)
	}

	encoded, err := json.Marshal(loaded)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	var decoded CandidatePromotionDecisionRecord
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if !reflect.DeepEqual(decoded, loaded) {
		t.Fatalf("JSON round-trip = %#v, want %#v", decoded, loaded)
	}
}

func TestCandidatePromotionDecisionReplayAndDuplicates(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 25, 11, 0, 0, 0, time.UTC)
	storeCandidatePromotionEligibilityFixtures(t, root, now, nil, func(record *CandidateResultRecord) {
		record.ResultID = "result-replay"
	})

	record, changed, err := CreateCandidatePromotionDecisionFromEligibleResult(root, "result-replay", "operator", now.Add(10*time.Minute))
	if err != nil {
		t.Fatalf("CreateCandidatePromotionDecisionFromEligibleResult(first) error = %v", err)
	}
	if !changed {
		t.Fatal("first changed = false, want true")
	}
	firstBytes, err := os.ReadFile(StoreCandidatePromotionDecisionPath(root, record.PromotionDecisionID))
	if err != nil {
		t.Fatalf("ReadFile(first) error = %v", err)
	}

	replayed, changed, err := CreateCandidatePromotionDecisionFromEligibleResult(root, "result-replay", "operator", now.Add(10*time.Minute))
	if err != nil {
		t.Fatalf("CreateCandidatePromotionDecisionFromEligibleResult(replay) error = %v", err)
	}
	if changed {
		t.Fatal("replay changed = true, want false")
	}
	if !reflect.DeepEqual(replayed, record) {
		t.Fatalf("replayed = %#v, want %#v", replayed, record)
	}
	secondBytes, err := os.ReadFile(StoreCandidatePromotionDecisionPath(root, record.PromotionDecisionID))
	if err != nil {
		t.Fatalf("ReadFile(second) error = %v", err)
	}
	if string(firstBytes) != string(secondBytes) {
		t.Fatalf("candidate promotion decision file changed on idempotent replay\nfirst:\n%s\nsecond:\n%s", string(firstBytes), string(secondBytes))
	}

	divergent := record
	divergent.Notes = "divergent replay"
	err = StoreCandidatePromotionDecisionRecord(root, divergent)
	if err == nil {
		t.Fatal("StoreCandidatePromotionDecisionRecord(divergent) error = nil, want duplicate rejection")
	}
	if !strings.Contains(err.Error(), `mission store candidate promotion decision "candidate-promotion-decision-result-replay" already exists`) {
		t.Fatalf("StoreCandidatePromotionDecisionRecord(divergent) error = %q, want duplicate context", err.Error())
	}

	otherID := record
	otherID.PromotionDecisionID = "candidate-promotion-decision-other"
	err = StoreCandidatePromotionDecisionRecord(root, otherID)
	if err == nil {
		t.Fatal("StoreCandidatePromotionDecisionRecord(second result decision) error = nil, want same-result rejection")
	}
	if !strings.Contains(err.Error(), `mission store candidate promotion decision for result_id "result-replay" already exists`) {
		t.Fatalf("StoreCandidatePromotionDecisionRecord(second result decision) error = %q, want same-result context", err.Error())
	}
}

func TestCreateCandidatePromotionDecisionFromEligibleResultFailsClosedForNonEligibleStates(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		resultID   string
		policyEdit func(*PromotionPolicyRecord)
		resultEdit func(*CandidateResultRecord)
		want       string
	}{
		{
			name:     "missing candidate result",
			resultID: "missing-result",
			want:     `promotion eligibility state "invalid"`,
		},
		{
			name:     "invalid missing promotion policy id",
			resultID: "result-invalid",
			resultEdit: func(record *CandidateResultRecord) {
				record.PromotionPolicyID = ""
			},
			want: `promotion eligibility state "invalid"`,
		},
		{
			name:     "rejected",
			resultID: "result-rejected",
			resultEdit: func(record *CandidateResultRecord) {
				record.HoldoutScore = record.BaselineScore
			},
			want: `promotion eligibility state "rejected"`,
		},
		{
			name:     "unsupported policy",
			resultID: "result-unsupported",
			policyEdit: func(record *PromotionPolicyRecord) {
				record.EpsilonRule = "epsilon maybe small"
			},
			want: `promotion eligibility state "unsupported_policy"`,
		},
		{
			name:     "canary required",
			resultID: "result-canary",
			policyEdit: func(record *PromotionPolicyRecord) {
				record.RequiresCanary = true
			},
			want: `promotion eligibility state "canary_required"`,
		},
		{
			name:     "owner approval required",
			resultID: "result-owner",
			policyEdit: func(record *PromotionPolicyRecord) {
				record.RequiresOwnerApproval = true
			},
			want: `promotion eligibility state "owner_approval_required"`,
		},
		{
			name:     "canary and owner approval required",
			resultID: "result-canary-owner",
			policyEdit: func(record *PromotionPolicyRecord) {
				record.RequiresCanary = true
				record.RequiresOwnerApproval = true
			},
			want: `promotion eligibility state "canary_and_owner_approval_required"`,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			now := time.Date(2026, 4, 25, 12, 0, 0, 0, time.UTC)
			if tt.resultID != "missing-result" {
				storeCandidatePromotionEligibilityFixtures(t, root, now, tt.policyEdit, func(record *CandidateResultRecord) {
					record.ResultID = tt.resultID
					if tt.resultEdit != nil {
						tt.resultEdit(record)
					}
				})
			}

			_, changed, err := CreateCandidatePromotionDecisionFromEligibleResult(root, tt.resultID, "operator", now.Add(10*time.Minute))
			if err == nil {
				t.Fatal("CreateCandidatePromotionDecisionFromEligibleResult() error = nil, want fail-closed rejection")
			}
			if changed {
				t.Fatal("changed = true on rejected decision, want false")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("CreateCandidatePromotionDecisionFromEligibleResult() error = %q, want substring %q", err.Error(), tt.want)
			}
			if _, statErr := os.Stat(StoreCandidatePromotionDecisionsDir(root)); !os.IsNotExist(statErr) {
				t.Fatalf("candidate promotion decision dir exists or errored after rejection: %v", statErr)
			}
		})
	}
}

func TestCreateCandidatePromotionDecisionFromEligibleResultRejectsMissingMetadata(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 25, 13, 0, 0, 0, time.UTC)
	storeCandidatePromotionEligibilityFixtures(t, root, now, nil, func(record *CandidateResultRecord) {
		record.ResultID = "result-metadata"
	})

	_, changed, err := CreateCandidatePromotionDecisionFromEligibleResult(root, "result-metadata", " ", now.Add(10*time.Minute))
	if err == nil {
		t.Fatal("CreateCandidatePromotionDecisionFromEligibleResult(missing created_by) error = nil, want rejection")
	}
	if changed {
		t.Fatal("changed = true on missing created_by, want false")
	}
	if !strings.Contains(err.Error(), "created_by is required") {
		t.Fatalf("missing created_by error = %q, want created_by context", err.Error())
	}

	_, changed, err = CreateCandidatePromotionDecisionFromEligibleResult(root, "result-metadata", "operator", time.Time{})
	if err == nil {
		t.Fatal("CreateCandidatePromotionDecisionFromEligibleResult(zero created_at) error = nil, want rejection")
	}
	if changed {
		t.Fatal("changed = true on zero created_at, want false")
	}
	if !strings.Contains(err.Error(), "created_at is required") {
		t.Fatalf("zero created_at error = %q, want created_at context", err.Error())
	}
}

func TestCandidatePromotionDecisionStoreDoesNotMutateRuntimeSurfaces(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 25, 14, 0, 0, 0, time.UTC)
	storeCandidatePromotionEligibilityFixtures(t, root, now, nil, func(record *CandidateResultRecord) {
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

	if _, _, err := CreateCandidatePromotionDecisionFromEligibleResult(root, "result-no-mutation", "operator", now.Add(10*time.Minute)); err != nil {
		t.Fatalf("CreateCandidatePromotionDecisionFromEligibleResult() error = %v", err)
	}

	for path, before := range snapshots {
		after, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ReadFile(%s) after decision error = %v", path, err)
		}
		if string(after) != string(before) {
			t.Fatalf("linked record %s changed after candidate promotion decision creation", path)
		}
	}

	absentPaths := []string{
		StoreHotUpdateOutcomesDir(root),
		StorePromotionsDir(root),
		StoreRollbacksDir(root),
		StoreRollbackAppliesDir(root),
		StoreActiveRuntimePackPointerPath(root),
		StoreLastKnownGoodRuntimePackPointerPath(root),
	}
	for _, path := range absentPaths {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("path %s exists or errored after candidate promotion decision creation: %v", path, err)
		}
	}
}

func validCandidatePromotionDecisionRecord(now time.Time, mutate func(*CandidatePromotionDecisionRecord)) CandidatePromotionDecisionRecord {
	record := CandidatePromotionDecisionRecord{
		PromotionDecisionID: "candidate-promotion-decision-result-root",
		ResultID:            "result-root",
		RunID:               "run-result",
		CandidateID:         "candidate-1",
		EvalSuiteID:         "eval-suite-1",
		PromotionPolicyID:   "promotion-policy-result",
		BaselinePackID:      "pack-base",
		CandidatePackID:     "pack-candidate",
		EligibilityState:    CandidatePromotionEligibilityStateEligible,
		Decision:            CandidatePromotionDecisionSelectedForPromotion,
		Reason:              "candidate result eligible for promotion",
		CreatedAt:           now,
		CreatedBy:           "operator",
	}
	if mutate != nil {
		mutate(&record)
	}
	return record
}
