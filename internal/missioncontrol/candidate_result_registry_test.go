package missioncontrol

import (
	"encoding/json"
	"errors"
	"math"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestCandidateResultRecordRoundTripAndList(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 24, 10, 0, 0, 0, time.FixedZone("offset", -4*60*60))

	storeImprovementRunFixtures(t, root, now)
	if err := StoreImprovementRunRecord(root, validImprovementRunRecord(now.Add(5*time.Minute), func(record *ImprovementRunRecord) {
		record.RunID = "run-result"
		record.CreatedBy = "operator"
	})); err != nil {
		t.Fatalf("StoreImprovementRunRecord() error = %v", err)
	}

	second := validCandidateResultRecord(now.Add(6*time.Minute), func(record *CandidateResultRecord) {
		record.ResultID = "result-b"
		record.RunID = "run-result"
		record.HotUpdateID = ""
		record.BaselineScore = 0.42
		record.TrainScore = 0.63
		record.HoldoutScore = 0.61
		record.Decision = ImprovementRunDecisionDiscard
		record.Notes = "holdout below threshold"
		record.RegressionFlags = []string{"latency_regression"}
		record.CreatedBy = "system"
	})
	if err := StoreCandidateResultRecord(root, second); err != nil {
		t.Fatalf("StoreCandidateResultRecord(result-b) error = %v", err)
	}

	want := validCandidateResultRecord(now.Add(7*time.Minute), func(record *CandidateResultRecord) {
		record.ResultID = " result-a "
		record.RunID = " run-result "
		record.CandidateID = " candidate-1 "
		record.EvalSuiteID = " eval-suite-1 "
		record.BaselinePackID = " pack-base "
		record.CandidatePackID = " pack-candidate "
		record.HotUpdateID = " hot-update-1 "
		record.RegressionFlags = []string{" holdout_warning ", " canary_needed "}
		record.Notes = " keep for next gate "
		record.CreatedBy = " operator "
	})
	if err := StoreCandidateResultRecord(root, want); err != nil {
		t.Fatalf("StoreCandidateResultRecord(result-a) error = %v", err)
	}

	got, err := LoadCandidateResultRecord(root, "result-a")
	if err != nil {
		t.Fatalf("LoadCandidateResultRecord() error = %v", err)
	}

	want.RecordVersion = StoreRecordVersion
	want.ResultID = "result-a"
	want.RunID = "run-result"
	want.CandidateID = "candidate-1"
	want.EvalSuiteID = "eval-suite-1"
	want.BaselinePackID = "pack-base"
	want.CandidatePackID = "pack-candidate"
	want.HotUpdateID = "hot-update-1"
	want.RegressionFlags = []string{"holdout_warning", "canary_needed"}
	want.Notes = "keep for next gate"
	want.CreatedAt = want.CreatedAt.UTC()
	want.CreatedBy = "operator"
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("LoadCandidateResultRecord() = %#v, want %#v", got, want)
	}

	records, err := ListCandidateResultRecords(root)
	if err != nil {
		t.Fatalf("ListCandidateResultRecords() error = %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("ListCandidateResultRecords() len = %d, want 2", len(records))
	}
	if records[0].ResultID != "result-a" || records[1].ResultID != "result-b" {
		t.Fatalf("ListCandidateResultRecords() ids = [%q %q], want [result-a result-b]", records[0].ResultID, records[1].ResultID)
	}
}

func TestCandidateResultReplayIsIdempotentAndAppendOnly(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 24, 11, 0, 0, 0, time.UTC)

	storeImprovementRunFixtures(t, root, now)
	if err := StoreImprovementRunRecord(root, validImprovementRunRecord(now.Add(5*time.Minute), func(record *ImprovementRunRecord) {
		record.RunID = "run-result"
	})); err != nil {
		t.Fatalf("StoreImprovementRunRecord() error = %v", err)
	}

	record := validCandidateResultRecord(now.Add(6*time.Minute), func(record *CandidateResultRecord) {
		record.ResultID = "result-replay"
		record.RunID = "run-result"
		record.HotUpdateID = ""
		record.Decision = ImprovementRunDecisionKeep
		record.RegressionFlags = nil
		record.Notes = "exact replay"
	})
	if err := StoreCandidateResultRecord(root, record); err != nil {
		t.Fatalf("StoreCandidateResultRecord(first) error = %v", err)
	}
	firstBytes, err := os.ReadFile(StoreCandidateResultPath(root, record.ResultID))
	if err != nil {
		t.Fatalf("ReadFile(first) error = %v", err)
	}

	if err := StoreCandidateResultRecord(root, record); err != nil {
		t.Fatalf("StoreCandidateResultRecord(replay) error = %v", err)
	}
	secondBytes, err := os.ReadFile(StoreCandidateResultPath(root, record.ResultID))
	if err != nil {
		t.Fatalf("ReadFile(second) error = %v", err)
	}
	if string(firstBytes) != string(secondBytes) {
		t.Fatalf("candidate result file changed on idempotent replay\nfirst:\n%s\nsecond:\n%s", string(firstBytes), string(secondBytes))
	}

	err = StoreCandidateResultRecord(root, validCandidateResultRecord(now.Add(7*time.Minute), func(changed *CandidateResultRecord) {
		changed.ResultID = "result-replay"
		changed.RunID = "run-result"
		changed.Notes = "divergent replay"
	}))
	if err == nil {
		t.Fatal("StoreCandidateResultRecord() error = nil, want append-only duplicate rejection")
	}
	if !strings.Contains(err.Error(), `mission store candidate result "result-replay" already exists`) {
		t.Fatalf("StoreCandidateResultRecord() error = %q, want append-only duplicate rejection", err.Error())
	}
}

func TestCandidateResultValidationFailsClosed(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 24, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name string
		run  func() error
		want string
	}{
		{
			name: "missing result id",
			run: func() error {
				return StoreCandidateResultRecord(root, validCandidateResultRecord(now, func(record *CandidateResultRecord) {
					record.ResultID = " "
				}))
			},
			want: "candidate result ref result_id is required",
		},
		{
			name: "missing decision",
			run: func() error {
				return StoreCandidateResultRecord(root, validCandidateResultRecord(now, func(record *CandidateResultRecord) {
					record.Decision = ""
				}))
			},
			want: "mission store candidate result decision is required",
		},
		{
			name: "invalid decision",
			run: func() error {
				return StoreCandidateResultRecord(root, validCandidateResultRecord(now, func(record *CandidateResultRecord) {
					record.Decision = ImprovementRunDecision("bad_decision")
				}))
			},
			want: `mission store candidate result decision "bad_decision" is invalid`,
		},
		{
			name: "non-finite score",
			run: func() error {
				return StoreCandidateResultRecord(root, validCandidateResultRecord(now, func(record *CandidateResultRecord) {
					record.HoldoutScore = math.Inf(1)
				}))
			},
			want: "mission store candidate result holdout_score must be finite",
		},
		{
			name: "invalid promotion policy id",
			run: func() error {
				return StoreCandidateResultRecord(root, validCandidateResultRecord(now, func(record *CandidateResultRecord) {
					record.PromotionPolicyID = ".bad-policy"
				}))
			},
			want: `mission store candidate result promotion_policy_id ".bad-policy"`,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := tc.run()
			if err == nil {
				t.Fatal("StoreCandidateResultRecord() error = nil, want fail-closed rejection")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("StoreCandidateResultRecord() error = %q, want substring %q", err.Error(), tc.want)
			}
		})
	}
}

func TestCandidateResultRejectsMissingOrMismatchedLinkedRefs(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 24, 13, 0, 0, 0, time.UTC)

	err := StoreCandidateResultRecord(root, validCandidateResultRecord(now, func(record *CandidateResultRecord) {
		record.RunID = "missing-run"
		record.CandidateID = "missing-candidate"
		record.EvalSuiteID = "missing-eval"
		record.BaselinePackID = "missing-base"
		record.CandidatePackID = "missing-pack"
		record.HotUpdateID = "missing-gate"
	}))
	if err == nil {
		t.Fatal("StoreCandidateResultRecord() error = nil, want missing ref rejection")
	}
	if !strings.Contains(err.Error(), ErrImprovementRunRecordNotFound.Error()) {
		t.Fatalf("StoreCandidateResultRecord() error = %q, want missing run rejection", err.Error())
	}

	storeImprovementRunFixtures(t, root, now.Add(time.Minute))
	if err := StoreImprovementRunRecord(root, validImprovementRunRecord(now.Add(6*time.Minute), func(record *ImprovementRunRecord) {
		record.RunID = "run-result"
		record.HotUpdateID = "hot-update-1"
	})); err != nil {
		t.Fatalf("StoreImprovementRunRecord() error = %v", err)
	}

	err = StoreCandidateResultRecord(root, validCandidateResultRecord(now.Add(7*time.Minute), func(record *CandidateResultRecord) {
		record.ResultID = "result-mismatch"
		record.RunID = "run-result"
		record.BaselinePackID = "pack-other"
	}))
	if err == nil {
		t.Fatal("StoreCandidateResultRecord() error = nil, want linkage mismatch rejection")
	}
	if !strings.Contains(err.Error(), `baseline_pack_id "pack-other" does not match run baseline_pack_id "pack-base"`) {
		t.Fatalf("StoreCandidateResultRecord() error = %q, want run linkage mismatch rejection", err.Error())
	}

	err = StoreCandidateResultRecord(root, validCandidateResultRecord(now.Add(8*time.Minute), func(record *CandidateResultRecord) {
		record.ResultID = "result-candidate-pack-mismatch"
		record.RunID = "run-result"
		record.CandidatePackID = "pack-base"
	}))
	if err == nil {
		t.Fatal("StoreCandidateResultRecord() error = nil, want candidate pack mismatch rejection")
	}
	if !strings.Contains(err.Error(), `candidate_pack_id "pack-base" does not match run candidate_pack_id "pack-candidate"`) {
		t.Fatalf("StoreCandidateResultRecord() error = %q, want candidate pack mismatch rejection", err.Error())
	}

	err = StoreCandidateResultRecord(root, validCandidateResultRecord(now.Add(9*time.Minute), func(record *CandidateResultRecord) {
		record.ResultID = "result-eval-suite-mismatch"
		record.RunID = "run-result"
		record.EvalSuiteID = "eval-suite-other"
	}))
	if err == nil {
		t.Fatal("StoreCandidateResultRecord() error = nil, want eval-suite mismatch rejection")
	}
	if !strings.Contains(err.Error(), `eval_suite_id "eval-suite-other" does not match run eval_suite_id "eval-suite-1"`) {
		t.Fatalf("StoreCandidateResultRecord() error = %q, want eval-suite mismatch rejection", err.Error())
	}
}

func TestCandidateResultPromotionPolicyReference(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 24, 13, 30, 0, 0, time.UTC)

	storeImprovementRunFixtures(t, root, now)
	if err := StoreImprovementRunRecord(root, validImprovementRunRecord(now.Add(5*time.Minute), func(record *ImprovementRunRecord) {
		record.RunID = "run-result"
	})); err != nil {
		t.Fatalf("StoreImprovementRunRecord() error = %v", err)
	}
	if err := StorePromotionPolicyRecord(root, validPromotionPolicyRecord(now.Add(6*time.Minute), func(record *PromotionPolicyRecord) {
		record.PromotionPolicyID = "promotion-policy-result"
	})); err != nil {
		t.Fatalf("StorePromotionPolicyRecord() error = %v", err)
	}

	record := validCandidateResultRecord(now.Add(7*time.Minute), func(record *CandidateResultRecord) {
		record.ResultID = "result-policy"
		record.RunID = "run-result"
		record.PromotionPolicyID = " promotion-policy-result "
	})
	if err := StoreCandidateResultRecord(root, record); err != nil {
		t.Fatalf("StoreCandidateResultRecord(with policy) error = %v", err)
	}
	got, err := LoadCandidateResultRecord(root, "result-policy")
	if err != nil {
		t.Fatalf("LoadCandidateResultRecord() error = %v", err)
	}
	if got.PromotionPolicyID != "promotion-policy-result" {
		t.Fatalf("PromotionPolicyID = %q, want promotion-policy-result", got.PromotionPolicyID)
	}

	err = StoreCandidateResultRecord(root, validCandidateResultRecord(now.Add(8*time.Minute), func(record *CandidateResultRecord) {
		record.ResultID = "result-missing-policy"
		record.RunID = "run-result"
		record.PromotionPolicyID = "promotion-policy-missing"
	}))
	if err == nil {
		t.Fatal("StoreCandidateResultRecord() error = nil, want missing promotion policy rejection")
	}
	if !strings.Contains(err.Error(), ErrPromotionPolicyRecordNotFound.Error()) {
		t.Fatalf("StoreCandidateResultRecord() error = %q, want missing promotion policy rejection", err.Error())
	}
}

func TestCandidateResultStoreDoesNotMutateLinkedRecordsOrRuntimePointers(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 24, 13, 45, 0, 0, time.UTC)

	storeImprovementRunFixtures(t, root, now)
	if err := StoreImprovementRunRecord(root, validImprovementRunRecord(now.Add(5*time.Minute), func(record *ImprovementRunRecord) {
		record.RunID = "run-result"
	})); err != nil {
		t.Fatalf("StoreImprovementRunRecord() error = %v", err)
	}
	if err := StorePromotionPolicyRecord(root, validPromotionPolicyRecord(now.Add(6*time.Minute), func(record *PromotionPolicyRecord) {
		record.PromotionPolicyID = "promotion-policy-result"
	})); err != nil {
		t.Fatalf("StorePromotionPolicyRecord() error = %v", err)
	}

	snapshots := map[string][]byte{}
	for _, path := range []string{
		StoreRuntimePackPath(root, "pack-base"),
		StoreRuntimePackPath(root, "pack-candidate"),
		StoreImprovementCandidatePath(root, "candidate-1"),
		StoreEvalSuitePath(root, "eval-suite-1"),
		StoreImprovementRunPath(root, "run-result"),
		StoreHotUpdateGatePath(root, "hot-update-1"),
		StorePromotionPolicyPath(root, "promotion-policy-result"),
	} {
		bytes, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ReadFile(%s) error = %v", path, err)
		}
		snapshots[path] = bytes
	}

	if err := StoreCandidateResultRecord(root, validCandidateResultRecord(now.Add(7*time.Minute), func(record *CandidateResultRecord) {
		record.ResultID = "result-no-mutation"
		record.RunID = "run-result"
		record.PromotionPolicyID = "promotion-policy-result"
	})); err != nil {
		t.Fatalf("StoreCandidateResultRecord() error = %v", err)
	}

	for path, before := range snapshots {
		after, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ReadFile(%s) after store error = %v", path, err)
		}
		if string(after) != string(before) {
			t.Fatalf("linked record %s changed after candidate-result store", path)
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
			t.Fatalf("path %s exists or errored after candidate-result store: %v", path, err)
		}
	}
}

func TestEvaluateCandidateResultPromotionEligibilityFailClosed(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 24, 14, 0, 0, 0, time.UTC)

	status, err := EvaluateCandidateResultPromotionEligibility(root, "missing-result")
	if err != nil {
		t.Fatalf("EvaluateCandidateResultPromotionEligibility() error = %v, want nil status error", err)
	}
	if status.State != CandidatePromotionEligibilityStateInvalid || !strings.Contains(status.Error, ErrCandidateResultRecordNotFound.Error()) {
		t.Fatalf("missing result status = %#v, want invalid missing-result status", status)
	}

	storeCandidatePromotionEligibilityFixtures(t, root, now, nil, func(record *CandidateResultRecord) {
		record.ResultID = "result-missing-policy-id"
		record.PromotionPolicyID = ""
	})
	status, err = EvaluateCandidateResultPromotionEligibility(root, "result-missing-policy-id")
	if err != nil {
		t.Fatalf("EvaluateCandidateResultPromotionEligibility(missing policy id) error = %v", err)
	}
	if status.State != CandidatePromotionEligibilityStateInvalid || !strings.Contains(status.Error, "promotion_policy_id is required") {
		t.Fatalf("missing promotion_policy_id status = %#v, want invalid", status)
	}

	record := validCandidateResultRecord(now.Add(10*time.Minute), func(record *CandidateResultRecord) {
		record.RecordVersion = StoreRecordVersion
		record.ResultID = "result-missing-policy"
		record.RunID = "run-result"
		record.PromotionPolicyID = "promotion-policy-missing"
	})
	writeRawCandidateResultRecord(t, root, record, "")
	status, err = EvaluateCandidateResultPromotionEligibility(root, "result-missing-policy")
	if err != nil {
		t.Fatalf("EvaluateCandidateResultPromotionEligibility(missing policy) error = %v", err)
	}
	if status.State != CandidatePromotionEligibilityStateInvalid || !strings.Contains(status.Error, ErrPromotionPolicyRecordNotFound.Error()) {
		t.Fatalf("missing policy status = %#v, want invalid missing-policy status", status)
	}
}

func TestEvaluateCandidateResultPromotionEligibilityScorePresenceAndFiniteChecks(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 24, 14, 30, 0, 0, time.UTC)

	storeCandidatePromotionEligibilityFixtures(t, root, now, nil, func(record *CandidateResultRecord) {
		record.ResultID = "result-missing-score"
	})
	removeCandidateResultJSONKey(t, root, "result-missing-score", "holdout_score")
	status, err := EvaluateCandidateResultPromotionEligibility(root, "result-missing-score")
	if err != nil {
		t.Fatalf("EvaluateCandidateResultPromotionEligibility(missing score) error = %v", err)
	}
	if status.State != CandidatePromotionEligibilityStateInvalid || !strings.Contains(status.Error, "holdout_score is required") {
		t.Fatalf("missing score status = %#v, want invalid missing holdout_score", status)
	}

	nonFiniteRoot := t.TempDir()
	storeCandidatePromotionEligibilityFixtures(t, nonFiniteRoot, now.Add(time.Hour), nil, func(record *CandidateResultRecord) {
		record.ResultID = "result-non-finite-score"
	})
	replaceCandidateResultJSONKey(t, nonFiniteRoot, "result-non-finite-score", "holdout_score", json.RawMessage(`1e999`))
	status, err = EvaluateCandidateResultPromotionEligibility(nonFiniteRoot, "result-non-finite-score")
	if err != nil {
		t.Fatalf("EvaluateCandidateResultPromotionEligibility(non-finite score) error = %v", err)
	}
	if status.State != CandidatePromotionEligibilityStateInvalid {
		t.Fatalf("non-finite score status = %#v, want invalid", status)
	}
}

func TestEvaluateCandidateResultPromotionEligibilityUnsupportedPolicyRules(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		policyEdit func(*PromotionPolicyRecord)
	}{
		{
			name: "epsilon",
			policyEdit: func(record *PromotionPolicyRecord) {
				record.EpsilonRule = "epsilon around 0.01"
			},
		},
		{
			name: "regression",
			policyEdit: func(record *PromotionPolicyRecord) {
				record.RegressionRule = "regressions maybe ok"
			},
		},
		{
			name: "compatibility",
			policyEdit: func(record *PromotionPolicyRecord) {
				record.CompatibilityRule = "compatibility_contract_passed"
			},
		},
		{
			name: "resource",
			policyEdit: func(record *PromotionPolicyRecord) {
				record.ResourceRule = "resource_budget_within_limit"
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			now := time.Date(2026, 4, 24, 15, 0, 0, 0, time.UTC)
			storeCandidatePromotionEligibilityFixtures(t, root, now, tt.policyEdit, func(record *CandidateResultRecord) {
				record.ResultID = "result-" + tt.name
			})

			status, err := EvaluateCandidateResultPromotionEligibility(root, "result-"+tt.name)
			if err != nil {
				t.Fatalf("EvaluateCandidateResultPromotionEligibility() error = %v", err)
			}
			if status.State != CandidatePromotionEligibilityStateUnsupportedPolicy {
				t.Fatalf("State = %q, want unsupported_policy; status = %#v", status.State, status)
			}
		})
	}
}

func TestEvaluateCandidateResultPromotionEligibilityStates(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		policyEdit func(*PromotionPolicyRecord)
		resultEdit func(*CandidateResultRecord)
		wantState  string
	}{
		{
			name:      "eligible",
			wantState: CandidatePromotionEligibilityStateEligible,
		},
		{
			name: "train-only rejected",
			resultEdit: func(record *CandidateResultRecord) {
				record.HoldoutScore = record.BaselineScore
			},
			wantState: CandidatePromotionEligibilityStateRejected,
		},
		{
			name: "discard decision rejected",
			resultEdit: func(record *CandidateResultRecord) {
				record.Decision = ImprovementRunDecisionDiscard
			},
			wantState: CandidatePromotionEligibilityStateRejected,
		},
		{
			name: "regression flags rejected",
			resultEdit: func(record *CandidateResultRecord) {
				record.RegressionFlags = []string{"latency_regression"}
			},
			wantState: CandidatePromotionEligibilityStateRejected,
		},
		{
			name: "canary required",
			policyEdit: func(record *PromotionPolicyRecord) {
				record.RequiresCanary = true
			},
			wantState: CandidatePromotionEligibilityStateCanaryRequired,
		},
		{
			name: "owner approval required",
			policyEdit: func(record *PromotionPolicyRecord) {
				record.RequiresOwnerApproval = true
			},
			wantState: CandidatePromotionEligibilityStateOwnerApprovalRequired,
		},
		{
			name: "canary and owner approval required",
			policyEdit: func(record *PromotionPolicyRecord) {
				record.RequiresCanary = true
				record.RequiresOwnerApproval = true
			},
			wantState: CandidatePromotionEligibilityStateCanaryAndOwnerApprovalRequired,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			now := time.Date(2026, 4, 24, 15, 30, 0, 0, time.UTC)
			storeCandidatePromotionEligibilityFixtures(t, root, now, tt.policyEdit, func(record *CandidateResultRecord) {
				record.ResultID = "result-" + strings.ReplaceAll(tt.name, " ", "-")
				if tt.resultEdit != nil {
					tt.resultEdit(record)
				}
			})

			status, err := EvaluateCandidateResultPromotionEligibility(root, "result-"+strings.ReplaceAll(tt.name, " ", "-"))
			if err != nil {
				t.Fatalf("EvaluateCandidateResultPromotionEligibility() error = %v", err)
			}
			if status.State != tt.wantState {
				t.Fatalf("State = %q, want %q; status = %#v", status.State, tt.wantState, status)
			}
		})
	}
}

func TestEvaluateCandidateResultPromotionEligibilityDoesNotMutateLinkedRecordsOrRuntimePointers(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 24, 16, 0, 0, 0, time.UTC)
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

	status, err := EvaluateCandidateResultPromotionEligibility(root, "result-no-mutation")
	if err != nil {
		t.Fatalf("EvaluateCandidateResultPromotionEligibility() error = %v", err)
	}
	if status.State != CandidatePromotionEligibilityStateEligible {
		t.Fatalf("State = %q, want eligible; status = %#v", status.State, status)
	}

	for path, before := range snapshots {
		after, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ReadFile(%s) after eligibility error = %v", path, err)
		}
		if string(after) != string(before) {
			t.Fatalf("linked record %s changed after eligibility evaluation", path)
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
			t.Fatalf("path %s exists or errored after eligibility evaluation: %v", path, err)
		}
	}
}

func TestLoadCandidateResultRecordNotFound(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if _, err := LoadCandidateResultRecord(root, "missing-result"); !errors.Is(err, ErrCandidateResultRecordNotFound) {
		t.Fatalf("LoadCandidateResultRecord() error = %v, want %v", err, ErrCandidateResultRecordNotFound)
	}
}

func storeCandidatePromotionEligibilityFixtures(t *testing.T, root string, now time.Time, policyEdit func(*PromotionPolicyRecord), resultEdit func(*CandidateResultRecord)) {
	t.Helper()

	storeImprovementRunFixtures(t, root, now)
	if err := StoreImprovementRunRecord(root, validImprovementRunRecord(now.Add(5*time.Minute), func(record *ImprovementRunRecord) {
		record.RunID = "run-result"
	})); err != nil {
		t.Fatalf("StoreImprovementRunRecord() error = %v", err)
	}
	if err := StorePromotionPolicyRecord(root, validPromotionPolicyRecord(now.Add(6*time.Minute), func(record *PromotionPolicyRecord) {
		record.PromotionPolicyID = "promotion-policy-result"
		record.RequiresCanary = false
		record.RequiresOwnerApproval = false
		record.RequiresHoldoutPass = true
		record.EpsilonRule = "epsilon <= 0.01"
		record.RegressionRule = "no_regression_flags"
		record.CompatibilityRule = "compatibility_score >= 0.90"
		record.ResourceRule = "resource_score >= 0.60"
		if policyEdit != nil {
			policyEdit(record)
		}
	})); err != nil {
		t.Fatalf("StorePromotionPolicyRecord() error = %v", err)
	}
	if err := StoreCandidateResultRecord(root, validCandidateResultRecord(now.Add(7*time.Minute), func(record *CandidateResultRecord) {
		record.ResultID = "result-eligible"
		record.RunID = "run-result"
		record.PromotionPolicyID = "promotion-policy-result"
		record.BaselineScore = 0.52
		record.TrainScore = 0.78
		record.HoldoutScore = 0.74
		record.CompatibilityScore = 0.93
		record.ResourceScore = 0.67
		record.RegressionFlags = []string{"none"}
		record.Decision = ImprovementRunDecisionKeep
		if resultEdit != nil {
			resultEdit(record)
		}
	})); err != nil {
		t.Fatalf("StoreCandidateResultRecord() error = %v", err)
	}
}

func writeRawCandidateResultRecord(t *testing.T, root string, record CandidateResultRecord, deleteKey string) {
	t.Helper()

	record = NormalizeCandidateResultRecord(record)
	fields := candidateResultRecordJSONFields(t, record)
	if deleteKey != "" {
		delete(fields, deleteKey)
	}
	if err := WriteStoreJSONAtomic(StoreCandidateResultPath(root, record.ResultID), fields); err != nil {
		t.Fatalf("WriteStoreJSONAtomic(%s) error = %v", record.ResultID, err)
	}
}

func removeCandidateResultJSONKey(t *testing.T, root, resultID, key string) {
	t.Helper()

	fields := readCandidateResultJSONFields(t, root, resultID)
	delete(fields, key)
	if err := WriteStoreJSONAtomic(StoreCandidateResultPath(root, resultID), fields); err != nil {
		t.Fatalf("WriteStoreJSONAtomic(remove %s) error = %v", key, err)
	}
}

func replaceCandidateResultJSONKey(t *testing.T, root, resultID, key string, value json.RawMessage) {
	t.Helper()

	fields := readCandidateResultJSONFields(t, root, resultID)
	fields[key] = value
	if err := WriteStoreJSONAtomic(StoreCandidateResultPath(root, resultID), fields); err != nil {
		t.Fatalf("WriteStoreJSONAtomic(replace %s) error = %v", key, err)
	}
}

func readCandidateResultJSONFields(t *testing.T, root, resultID string) map[string]json.RawMessage {
	t.Helper()

	bytes, err := os.ReadFile(StoreCandidateResultPath(root, resultID))
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", resultID, err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(bytes, &fields); err != nil {
		t.Fatalf("Unmarshal(%s) error = %v", resultID, err)
	}
	return fields
}

func candidateResultRecordJSONFields(t *testing.T, record CandidateResultRecord) map[string]json.RawMessage {
	t.Helper()

	bytes, err := json.Marshal(record)
	if err != nil {
		t.Fatalf("Marshal(%s) error = %v", record.ResultID, err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(bytes, &fields); err != nil {
		t.Fatalf("Unmarshal marshaled record %s error = %v", record.ResultID, err)
	}
	return fields
}

func validCandidateResultRecord(now time.Time, mutate func(*CandidateResultRecord)) CandidateResultRecord {
	record := CandidateResultRecord{
		ResultID:           "result-root",
		RunID:              "run-root",
		CandidateID:        "candidate-1",
		EvalSuiteID:        "eval-suite-1",
		PromotionPolicyID:  "",
		BaselinePackID:     "pack-base",
		CandidatePackID:    "pack-candidate",
		HotUpdateID:        "hot-update-1",
		BaselineScore:      0.52,
		TrainScore:         0.78,
		HoldoutScore:       0.74,
		ComplexityScore:    0.21,
		CompatibilityScore: 0.93,
		ResourceScore:      0.67,
		RegressionFlags:    []string{"none"},
		Decision:           ImprovementRunDecisionKeep,
		Notes:              "candidate recorded for later promotion policy",
		CreatedAt:          now,
		CreatedBy:          "system",
	}
	if mutate != nil {
		mutate(&record)
	}
	return record
}
