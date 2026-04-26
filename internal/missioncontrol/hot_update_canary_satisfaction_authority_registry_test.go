package missioncontrol

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestValidateHotUpdateCanarySatisfactionAuthorityRecordRejectsMissingRequiredFields(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 27, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name string
		edit func(*HotUpdateCanarySatisfactionAuthorityRecord)
		want string
	}{
		{name: "record version", edit: func(record *HotUpdateCanarySatisfactionAuthorityRecord) { record.RecordVersion = 0 }, want: "record_version must be positive"},
		{name: "authority id", edit: func(record *HotUpdateCanarySatisfactionAuthorityRecord) { record.CanarySatisfactionAuthorityID = "" }, want: "canary_satisfaction_authority_id is required"},
		{name: "requirement id", edit: func(record *HotUpdateCanarySatisfactionAuthorityRecord) { record.CanaryRequirementID = "" }, want: "canary_requirement_id"},
		{name: "selected evidence id", edit: func(record *HotUpdateCanarySatisfactionAuthorityRecord) { record.SelectedCanaryEvidenceID = "" }, want: "selected_canary_evidence_id"},
		{name: "result id", edit: func(record *HotUpdateCanarySatisfactionAuthorityRecord) { record.ResultID = "" }, want: "result_id"},
		{name: "run id", edit: func(record *HotUpdateCanarySatisfactionAuthorityRecord) { record.RunID = "" }, want: "run_id"},
		{name: "candidate id", edit: func(record *HotUpdateCanarySatisfactionAuthorityRecord) { record.CandidateID = "" }, want: "candidate_id"},
		{name: "eval suite id", edit: func(record *HotUpdateCanarySatisfactionAuthorityRecord) { record.EvalSuiteID = "" }, want: "eval_suite_id"},
		{name: "promotion policy id", edit: func(record *HotUpdateCanarySatisfactionAuthorityRecord) { record.PromotionPolicyID = "" }, want: "promotion_policy_id"},
		{name: "baseline pack id", edit: func(record *HotUpdateCanarySatisfactionAuthorityRecord) { record.BaselinePackID = "" }, want: "baseline_pack_id"},
		{name: "candidate pack id", edit: func(record *HotUpdateCanarySatisfactionAuthorityRecord) { record.CandidatePackID = "" }, want: "candidate_pack_id"},
		{name: "invalid eligibility state", edit: func(record *HotUpdateCanarySatisfactionAuthorityRecord) {
			record.EligibilityState = CandidatePromotionEligibilityStateEligible
		}, want: "eligibility_state"},
		{name: "invalid satisfaction state", edit: func(record *HotUpdateCanarySatisfactionAuthorityRecord) {
			record.SatisfactionState = HotUpdateCanarySatisfactionStateFailed
		}, want: "satisfaction_state"},
		{name: "invalid authority state", edit: func(record *HotUpdateCanarySatisfactionAuthorityRecord) { record.State = "applied" }, want: "state"},
		{name: "satisfied state", edit: func(record *HotUpdateCanarySatisfactionAuthorityRecord) {
			record.State = HotUpdateCanarySatisfactionAuthorityStateWaitingOwnerApproval
		}, want: "state must be \"authorized\""},
		{name: "satisfied owner approval", edit: func(record *HotUpdateCanarySatisfactionAuthorityRecord) { record.OwnerApprovalRequired = true }, want: "owner_approval_required must be false"},
		{name: "waiting owner state", edit: func(record *HotUpdateCanarySatisfactionAuthorityRecord) {
			record.SatisfactionState = HotUpdateCanarySatisfactionStateWaitingOwnerApproval
			record.OwnerApprovalRequired = true
			record.State = HotUpdateCanarySatisfactionAuthorityStateAuthorized
		}, want: "state must be \"waiting_owner_approval\""},
		{name: "waiting owner approval bool", edit: func(record *HotUpdateCanarySatisfactionAuthorityRecord) {
			record.SatisfactionState = HotUpdateCanarySatisfactionStateWaitingOwnerApproval
			record.State = HotUpdateCanarySatisfactionAuthorityStateWaitingOwnerApproval
			record.OwnerApprovalRequired = false
		}, want: "owner_approval_required must be true"},
		{name: "reason", edit: func(record *HotUpdateCanarySatisfactionAuthorityRecord) { record.Reason = "" }, want: "reason is required"},
		{name: "created at", edit: func(record *HotUpdateCanarySatisfactionAuthorityRecord) { record.CreatedAt = time.Time{} }, want: "created_at is required"},
		{name: "created by", edit: func(record *HotUpdateCanarySatisfactionAuthorityRecord) { record.CreatedBy = "" }, want: "created_by is required"},
		{name: "deterministic id", edit: func(record *HotUpdateCanarySatisfactionAuthorityRecord) {
			record.CanarySatisfactionAuthorityID = record.CanarySatisfactionAuthorityID + "-other"
		}, want: "does not match deterministic"},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			record := validHotUpdateCanarySatisfactionAuthorityRecord(now, nil)
			tt.edit(&record)
			err := ValidateHotUpdateCanarySatisfactionAuthorityRecord(NormalizeHotUpdateCanarySatisfactionAuthorityRecord(record))
			if err == nil {
				t.Fatal("ValidateHotUpdateCanarySatisfactionAuthorityRecord() error = nil, want rejection")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("ValidateHotUpdateCanarySatisfactionAuthorityRecord() error = %q, want substring %q", err.Error(), tt.want)
			}
		})
	}
}

func TestHotUpdateCanarySatisfactionAuthorityIDFromRequirementEvidence(t *testing.T) {
	t.Parallel()

	got := HotUpdateCanarySatisfactionAuthorityIDFromRequirementEvidence(" hot-update-canary-requirement-result-1 ", " evidence-1 ")
	want := "hot-update-canary-satisfaction-authority-hot-update-canary-requirement-result-1-evidence-1"
	if got != want {
		t.Fatalf("HotUpdateCanarySatisfactionAuthorityIDFromRequirementEvidence() = %q, want %q", got, want)
	}
}

func TestCreateHotUpdateCanarySatisfactionAuthorityFromRequirementCreatesAuthorizedAndWaitingOwnerApproval(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		policyEdit func(*PromotionPolicyRecord)
		wantState  HotUpdateCanarySatisfactionAuthorityState
		wantSat    HotUpdateCanarySatisfactionState
		wantOwner  bool
	}{
		{name: "authorized", wantState: HotUpdateCanarySatisfactionAuthorityStateAuthorized, wantSat: HotUpdateCanarySatisfactionStateSatisfied},
		{name: "waiting owner approval", policyEdit: func(record *PromotionPolicyRecord) {
			record.RequiresOwnerApproval = true
		}, wantState: HotUpdateCanarySatisfactionAuthorityStateWaitingOwnerApproval, wantSat: HotUpdateCanarySatisfactionStateWaitingOwnerApproval, wantOwner: true},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			now := time.Date(2026, 4, 27, 13, 0, 0, 0, time.UTC)
			requirement := storeCanaryRequirementForEvidence(t, root, now, tt.policyEdit, nil)
			evidence, _, err := CreateHotUpdateCanaryEvidenceFromRequirement(root, requirement.CanaryRequirementID, HotUpdateCanaryEvidenceStatePassed, now.Add(20*time.Minute), "operator", now.Add(21*time.Minute), "canary passed")
			if err != nil {
				t.Fatalf("CreateHotUpdateCanaryEvidenceFromRequirement() error = %v", err)
			}

			record, changed, err := CreateHotUpdateCanarySatisfactionAuthorityFromRequirement(root, " "+requirement.CanaryRequirementID+" ", " operator ", now.Add(22*time.Minute))
			if err != nil {
				t.Fatalf("CreateHotUpdateCanarySatisfactionAuthorityFromRequirement() error = %v", err)
			}
			if !changed {
				t.Fatal("changed = false, want true")
			}
			if record.State != tt.wantState || record.SatisfactionState != tt.wantSat || record.OwnerApprovalRequired != tt.wantOwner {
				t.Fatalf("authority state = %#v, want %s/%s/%v", record, tt.wantState, tt.wantSat, tt.wantOwner)
			}
			if record.SelectedCanaryEvidenceID != evidence.CanaryEvidenceID {
				t.Fatalf("SelectedCanaryEvidenceID = %q, want %q", record.SelectedCanaryEvidenceID, evidence.CanaryEvidenceID)
			}
			if record.CanarySatisfactionAuthorityID != HotUpdateCanarySatisfactionAuthorityIDFromRequirementEvidence(requirement.CanaryRequirementID, evidence.CanaryEvidenceID) {
				t.Fatalf("CanarySatisfactionAuthorityID = %q, want deterministic id", record.CanarySatisfactionAuthorityID)
			}
			if record.ResultID != requirement.ResultID || record.RunID != requirement.RunID || record.CandidateID != requirement.CandidateID ||
				record.EvalSuiteID != requirement.EvalSuiteID || record.PromotionPolicyID != requirement.PromotionPolicyID ||
				record.BaselinePackID != requirement.BaselinePackID || record.CandidatePackID != requirement.CandidatePackID {
				t.Fatalf("authority refs = %#v, want requirement/source refs", record)
			}
			loaded, err := LoadHotUpdateCanarySatisfactionAuthorityRecord(root, record.CanarySatisfactionAuthorityID)
			if err != nil {
				t.Fatalf("LoadHotUpdateCanarySatisfactionAuthorityRecord() error = %v", err)
			}
			if loaded != record {
				t.Fatalf("loaded = %#v, want %#v", loaded, record)
			}
		})
	}
}

func TestCreateHotUpdateCanarySatisfactionAuthorityFromRequirementRejectsUnsatisfiedStates(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		state       HotUpdateCanaryEvidenceState
		createEvent bool
		want        string
	}{
		{name: "not satisfied", want: "not_satisfied"},
		{name: "failed", state: HotUpdateCanaryEvidenceStateFailed, createEvent: true, want: "failed"},
		{name: "blocked", state: HotUpdateCanaryEvidenceStateBlocked, createEvent: true, want: "blocked"},
		{name: "expired", state: HotUpdateCanaryEvidenceStateExpired, createEvent: true, want: "expired"},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			now := time.Date(2026, 4, 27, 14, 0, 0, 0, time.UTC)
			requirement := storeCanaryRequirementForEvidence(t, root, now, nil, nil)
			if tt.createEvent {
				if _, _, err := CreateHotUpdateCanaryEvidenceFromRequirement(root, requirement.CanaryRequirementID, tt.state, now.Add(20*time.Minute), "operator", now.Add(21*time.Minute), string(tt.state)); err != nil {
					t.Fatalf("CreateHotUpdateCanaryEvidenceFromRequirement() error = %v", err)
				}
			}

			_, changed, err := CreateHotUpdateCanarySatisfactionAuthorityFromRequirement(root, requirement.CanaryRequirementID, "operator", now.Add(22*time.Minute))
			if err == nil {
				t.Fatal("CreateHotUpdateCanarySatisfactionAuthorityFromRequirement() error = nil, want rejection")
			}
			if changed {
				t.Fatal("changed = true, want false")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %q, want substring %q", err.Error(), tt.want)
			}
		})
	}
}

func TestCreateHotUpdateCanarySatisfactionAuthorityFromRequirementRejectsInvalidAssessment(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 27, 15, 0, 0, 0, time.UTC)
	requirement := storeCanaryRequirementForEvidence(t, root, now, nil, nil)
	bad := validHotUpdateCanaryEvidenceRecord(now.Add(20*time.Minute), func(record *HotUpdateCanaryEvidenceRecord) {
		record.CanaryRequirementID = requirement.CanaryRequirementID
		record.CanaryEvidenceID = HotUpdateCanaryEvidenceIDFromRequirementObservedAt(requirement.CanaryRequirementID, record.ObservedAt)
		record.ResultID = requirement.ResultID
		record.RunID = requirement.RunID
		record.CandidateID = requirement.CandidateID
		record.EvalSuiteID = requirement.EvalSuiteID
		record.PromotionPolicyID = requirement.PromotionPolicyID
		record.BaselinePackID = requirement.BaselinePackID
		record.CandidatePackID = "pack-missing"
	})
	if err := WriteStoreJSONAtomic(StoreHotUpdateCanaryEvidencePath(root, bad.CanaryEvidenceID), bad); err != nil {
		t.Fatalf("WriteStoreJSONAtomic(invalid evidence) error = %v", err)
	}

	_, changed, err := CreateHotUpdateCanarySatisfactionAuthorityFromRequirement(root, requirement.CanaryRequirementID, "operator", now.Add(22*time.Minute))
	if err == nil {
		t.Fatal("CreateHotUpdateCanarySatisfactionAuthorityFromRequirement() error = nil, want invalid assessment rejection")
	}
	if changed {
		t.Fatal("changed = true, want false")
	}
	if !strings.Contains(err.Error(), "requires configured canary satisfaction assessment") {
		t.Fatalf("error = %q, want invalid assessment", err.Error())
	}
}

func TestCreateHotUpdateCanarySatisfactionAuthorityFromRequirementLoadsAndCrossChecksLinkedRecords(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		mutate func(t *testing.T, root string, requirement HotUpdateCanaryRequirementRecord)
		want   string
	}{
		{name: "missing requirement", mutate: func(t *testing.T, root string, requirement HotUpdateCanaryRequirementRecord) {
			t.Helper()
			if err := os.Remove(StoreHotUpdateCanaryRequirementPath(root, requirement.CanaryRequirementID)); err != nil {
				t.Fatalf("Remove(requirement) error = %v", err)
			}
		}, want: ErrHotUpdateCanaryRequirementRecordNotFound.Error()},
		{name: "missing selected evidence", mutate: func(t *testing.T, root string, requirement HotUpdateCanaryRequirementRecord) {
			t.Helper()
			entries, err := os.ReadDir(StoreHotUpdateCanaryEvidenceDir(root))
			if err != nil {
				t.Fatalf("ReadDir(evidence) error = %v", err)
			}
			if err := os.Remove(filepath.Join(StoreHotUpdateCanaryEvidenceDir(root), entries[0].Name())); err != nil {
				t.Fatalf("Remove(evidence) error = %v", err)
			}
		}, want: "not_satisfied"},
		{name: "missing candidate result", mutate: func(t *testing.T, root string, requirement HotUpdateCanaryRequirementRecord) {
			t.Helper()
			if err := os.Remove(StoreCandidateResultPath(root, requirement.ResultID)); err != nil {
				t.Fatalf("Remove(result) error = %v", err)
			}
		}, want: ErrCandidateResultRecordNotFound.Error()},
		{name: "missing run", mutate: func(t *testing.T, root string, requirement HotUpdateCanaryRequirementRecord) {
			t.Helper()
			if err := os.Remove(StoreImprovementRunPath(root, requirement.RunID)); err != nil {
				t.Fatalf("Remove(run) error = %v", err)
			}
		}, want: ErrImprovementRunRecordNotFound.Error()},
		{name: "missing candidate", mutate: func(t *testing.T, root string, requirement HotUpdateCanaryRequirementRecord) {
			t.Helper()
			if err := os.Remove(StoreImprovementCandidatePath(root, requirement.CandidateID)); err != nil {
				t.Fatalf("Remove(candidate) error = %v", err)
			}
		}, want: ErrImprovementCandidateRecordNotFound.Error()},
		{name: "unfrozen eval suite", mutate: func(t *testing.T, root string, requirement HotUpdateCanaryRequirementRecord) {
			t.Helper()
			suite, err := LoadEvalSuiteRecord(root, requirement.EvalSuiteID)
			if err != nil {
				t.Fatalf("LoadEvalSuiteRecord() error = %v", err)
			}
			suite.FrozenForRun = false
			if err := WriteStoreJSONAtomic(StoreEvalSuitePath(root, suite.EvalSuiteID), suite); err != nil {
				t.Fatalf("WriteStoreJSONAtomic(eval suite) error = %v", err)
			}
		}, want: "frozen_for_run must be true"},
		{name: "missing policy", mutate: func(t *testing.T, root string, requirement HotUpdateCanaryRequirementRecord) {
			t.Helper()
			if err := os.Remove(StorePromotionPolicyPath(root, requirement.PromotionPolicyID)); err != nil {
				t.Fatalf("Remove(policy) error = %v", err)
			}
		}, want: ErrPromotionPolicyRecordNotFound.Error()},
		{name: "missing baseline pack", mutate: func(t *testing.T, root string, requirement HotUpdateCanaryRequirementRecord) {
			t.Helper()
			if err := os.Remove(StoreRuntimePackPath(root, requirement.BaselinePackID)); err != nil {
				t.Fatalf("Remove(baseline pack) error = %v", err)
			}
		}, want: ErrRuntimePackRecordNotFound.Error()},
		{name: "missing candidate pack", mutate: func(t *testing.T, root string, requirement HotUpdateCanaryRequirementRecord) {
			t.Helper()
			if err := os.Remove(StoreRuntimePackPath(root, requirement.CandidatePackID)); err != nil {
				t.Fatalf("Remove(candidate pack) error = %v", err)
			}
		}, want: ErrRuntimePackRecordNotFound.Error()},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			now := time.Date(2026, 4, 27, 16, 0, 0, 0, time.UTC)
			requirement := storeCanaryRequirementForEvidence(t, root, now, nil, nil)
			if _, _, err := CreateHotUpdateCanaryEvidenceFromRequirement(root, requirement.CanaryRequirementID, HotUpdateCanaryEvidenceStatePassed, now.Add(20*time.Minute), "operator", now.Add(21*time.Minute), "canary passed"); err != nil {
				t.Fatalf("CreateHotUpdateCanaryEvidenceFromRequirement() error = %v", err)
			}
			tt.mutate(t, root, requirement)

			_, changed, err := CreateHotUpdateCanarySatisfactionAuthorityFromRequirement(root, requirement.CanaryRequirementID, "operator", now.Add(22*time.Minute))
			if err == nil {
				t.Fatal("CreateHotUpdateCanarySatisfactionAuthorityFromRequirement() error = nil, want fail-closed linkage rejection")
			}
			if changed {
				t.Fatal("changed = true, want false")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %q, want substring %q", err.Error(), tt.want)
			}
		})
	}
}

func TestCreateHotUpdateCanarySatisfactionAuthorityFromRequirementRejectsStaleEligibilityAndLatestNonPassed(t *testing.T) {
	t.Parallel()

	t.Run("stale eligibility", func(t *testing.T) {
		t.Parallel()

		root := t.TempDir()
		now := time.Date(2026, 4, 27, 17, 0, 0, 0, time.UTC)
		requirement := storeCanaryRequirementForEvidence(t, root, now, nil, nil)
		if _, _, err := CreateHotUpdateCanaryEvidenceFromRequirement(root, requirement.CanaryRequirementID, HotUpdateCanaryEvidenceStatePassed, now.Add(20*time.Minute), "operator", now.Add(21*time.Minute), "canary passed"); err != nil {
			t.Fatalf("CreateHotUpdateCanaryEvidenceFromRequirement() error = %v", err)
		}
		policy, err := LoadPromotionPolicyRecord(root, requirement.PromotionPolicyID)
		if err != nil {
			t.Fatalf("LoadPromotionPolicyRecord() error = %v", err)
		}
		policy.RequiresCanary = false
		if err := WriteStoreJSONAtomic(StorePromotionPolicyPath(root, policy.PromotionPolicyID), policy); err != nil {
			t.Fatalf("WriteStoreJSONAtomic(policy) error = %v", err)
		}

		_, changed, err := CreateHotUpdateCanarySatisfactionAuthorityFromRequirement(root, requirement.CanaryRequirementID, "operator", now.Add(22*time.Minute))
		if err == nil {
			t.Fatal("CreateHotUpdateCanarySatisfactionAuthorityFromRequirement() error = nil, want stale eligibility rejection")
		}
		if changed {
			t.Fatal("changed = true, want false")
		}
		if !strings.Contains(err.Error(), "does not permit hot-update canary requirement") {
			t.Fatalf("error = %q, want stale eligibility", err.Error())
		}
	})

	t.Run("latest non passed", func(t *testing.T) {
		t.Parallel()

		root := t.TempDir()
		now := time.Date(2026, 4, 27, 17, 30, 0, 0, time.UTC)
		requirement := storeCanaryRequirementForEvidence(t, root, now, nil, nil)
		if _, _, err := CreateHotUpdateCanaryEvidenceFromRequirement(root, requirement.CanaryRequirementID, HotUpdateCanaryEvidenceStatePassed, now.Add(20*time.Minute), "operator", now.Add(21*time.Minute), "canary passed"); err != nil {
			t.Fatalf("CreateHotUpdateCanaryEvidenceFromRequirement(passed) error = %v", err)
		}
		if _, _, err := CreateHotUpdateCanaryEvidenceFromRequirement(root, requirement.CanaryRequirementID, HotUpdateCanaryEvidenceStateFailed, now.Add(30*time.Minute), "operator", now.Add(31*time.Minute), "canary failed"); err != nil {
			t.Fatalf("CreateHotUpdateCanaryEvidenceFromRequirement(failed) error = %v", err)
		}

		_, changed, err := CreateHotUpdateCanarySatisfactionAuthorityFromRequirement(root, requirement.CanaryRequirementID, "operator", now.Add(32*time.Minute))
		if err == nil {
			t.Fatal("CreateHotUpdateCanarySatisfactionAuthorityFromRequirement() error = nil, want latest failed rejection")
		}
		if changed {
			t.Fatal("changed = true, want false")
		}
		if !strings.Contains(err.Error(), "failed") {
			t.Fatalf("error = %q, want latest failed state", err.Error())
		}
	})
}

func TestHotUpdateCanarySatisfactionAuthorityReplayDuplicatesNewEvidenceAndListOrder(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 27, 18, 0, 0, 0, time.UTC)
	requirement := storeCanaryRequirementForEvidence(t, root, now, nil, nil)
	firstEvidence, _, err := CreateHotUpdateCanaryEvidenceFromRequirement(root, requirement.CanaryRequirementID, HotUpdateCanaryEvidenceStatePassed, now.Add(20*time.Minute), "operator", now.Add(21*time.Minute), "first canary passed")
	if err != nil {
		t.Fatalf("CreateHotUpdateCanaryEvidenceFromRequirement(first) error = %v", err)
	}
	createdAt := now.Add(22 * time.Minute)
	record, changed, err := CreateHotUpdateCanarySatisfactionAuthorityFromRequirement(root, requirement.CanaryRequirementID, "operator", createdAt)
	if err != nil {
		t.Fatalf("CreateHotUpdateCanarySatisfactionAuthorityFromRequirement(first) error = %v", err)
	}
	if !changed {
		t.Fatal("changed = false, want true")
	}
	if record.SelectedCanaryEvidenceID != firstEvidence.CanaryEvidenceID {
		t.Fatalf("SelectedCanaryEvidenceID = %q, want first evidence %q", record.SelectedCanaryEvidenceID, firstEvidence.CanaryEvidenceID)
	}
	firstBytes, err := os.ReadFile(StoreHotUpdateCanarySatisfactionAuthorityPath(root, record.CanarySatisfactionAuthorityID))
	if err != nil {
		t.Fatalf("ReadFile(first authority) error = %v", err)
	}

	replayed, changed, err := CreateHotUpdateCanarySatisfactionAuthorityFromRequirement(root, requirement.CanaryRequirementID, "operator", createdAt)
	if err != nil {
		t.Fatalf("CreateHotUpdateCanarySatisfactionAuthorityFromRequirement(replay) error = %v", err)
	}
	if changed {
		t.Fatal("changed = true, want false replay")
	}
	if replayed != record {
		t.Fatalf("replayed = %#v, want %#v", replayed, record)
	}
	secondBytes, err := os.ReadFile(StoreHotUpdateCanarySatisfactionAuthorityPath(root, record.CanarySatisfactionAuthorityID))
	if err != nil {
		t.Fatalf("ReadFile(replayed authority) error = %v", err)
	}
	if string(secondBytes) != string(firstBytes) {
		t.Fatal("authority replay changed stored bytes")
	}

	divergent := record
	divergent.Reason = "different reason"
	if err := WriteStoreJSONAtomic(StoreHotUpdateCanarySatisfactionAuthorityPath(root, divergent.CanarySatisfactionAuthorityID), divergent); err != nil {
		t.Fatalf("WriteStoreJSONAtomic(divergent authority) error = %v", err)
	}
	_, changed, err = CreateHotUpdateCanarySatisfactionAuthorityFromRequirement(root, requirement.CanaryRequirementID, "operator", createdAt)
	if err == nil {
		t.Fatal("CreateHotUpdateCanarySatisfactionAuthorityFromRequirement(divergent) error = nil, want duplicate rejection")
	}
	if changed {
		t.Fatal("changed = true, want false divergent duplicate")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("error = %q, want already exists", err.Error())
	}
	if err := WriteStoreJSONAtomic(StoreHotUpdateCanarySatisfactionAuthorityPath(root, record.CanarySatisfactionAuthorityID), record); err != nil {
		t.Fatalf("WriteStoreJSONAtomic(restore authority) error = %v", err)
	}

	otherID := record.CanarySatisfactionAuthorityID + "-other"
	other := record
	other.CanarySatisfactionAuthorityID = otherID
	if err := WriteStoreJSONAtomic(StoreHotUpdateCanarySatisfactionAuthorityPath(root, otherID), other); err != nil {
		t.Fatalf("WriteStoreJSONAtomic(other id authority) error = %v", err)
	}
	_, changed, err = CreateHotUpdateCanarySatisfactionAuthorityFromRequirement(root, requirement.CanaryRequirementID, "operator", createdAt)
	if err == nil {
		t.Fatal("CreateHotUpdateCanarySatisfactionAuthorityFromRequirement(other id) error = nil, want fail-closed duplicate/id mismatch")
	}
	if changed {
		t.Fatal("changed = true, want false other id duplicate")
	}
	_, err = ListHotUpdateCanarySatisfactionAuthorityRecords(root)
	if err == nil {
		t.Fatal("ListHotUpdateCanarySatisfactionAuthorityRecords() error = nil, want fail-closed duplicate/id mismatch")
	}
	if err := os.Remove(StoreHotUpdateCanarySatisfactionAuthorityPath(root, otherID)); err != nil {
		t.Fatalf("Remove(other id authority) error = %v", err)
	}

	secondEvidence, _, err := CreateHotUpdateCanaryEvidenceFromRequirement(root, requirement.CanaryRequirementID, HotUpdateCanaryEvidenceStatePassed, now.Add(40*time.Minute), "operator", now.Add(41*time.Minute), "second canary passed")
	if err != nil {
		t.Fatalf("CreateHotUpdateCanaryEvidenceFromRequirement(second) error = %v", err)
	}
	later, changed, err := CreateHotUpdateCanarySatisfactionAuthorityFromRequirement(root, requirement.CanaryRequirementID, "operator", now.Add(42*time.Minute))
	if err != nil {
		t.Fatalf("CreateHotUpdateCanarySatisfactionAuthorityFromRequirement(new evidence) error = %v", err)
	}
	if !changed {
		t.Fatal("changed = false, want true for newer passed evidence")
	}
	if later.SelectedCanaryEvidenceID != secondEvidence.CanaryEvidenceID || later.CanarySatisfactionAuthorityID == record.CanarySatisfactionAuthorityID {
		t.Fatalf("later authority = %#v, want distinct authority for second evidence %q", later, secondEvidence.CanaryEvidenceID)
	}

	records, err := ListHotUpdateCanarySatisfactionAuthorityRecords(root)
	if err != nil {
		t.Fatalf("ListHotUpdateCanarySatisfactionAuthorityRecords() error = %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("ListHotUpdateCanarySatisfactionAuthorityRecords() len = %d, want 2", len(records))
	}
	if records[0].CanarySatisfactionAuthorityID > records[1].CanarySatisfactionAuthorityID {
		t.Fatalf("ListHotUpdateCanarySatisfactionAuthorityRecords() order = %#v, want filename order", records)
	}
}

func TestHotUpdateCanarySatisfactionAuthorityStoreDoesNotMutateSourceOrRuntimeSurfaces(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 27, 19, 0, 0, 0, time.UTC)
	requirement := storeCanaryRequirementForEvidence(t, root, now, nil, nil)
	evidence, _, err := CreateHotUpdateCanaryEvidenceFromRequirement(root, requirement.CanaryRequirementID, HotUpdateCanaryEvidenceStatePassed, now.Add(20*time.Minute), "operator", now.Add(21*time.Minute), "canary passed")
	if err != nil {
		t.Fatalf("CreateHotUpdateCanaryEvidenceFromRequirement() error = %v", err)
	}
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
		StoreHotUpdateCanaryEvidencePath(root, evidence.CanaryEvidenceID),
	} {
		bytes, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ReadFile(%s) error = %v", path, err)
		}
		snapshots[path] = bytes
	}

	if _, _, err := CreateHotUpdateCanarySatisfactionAuthorityFromRequirement(root, requirement.CanaryRequirementID, "operator", now.Add(22*time.Minute)); err != nil {
		t.Fatalf("CreateHotUpdateCanarySatisfactionAuthorityFromRequirement() error = %v", err)
	}
	for path, before := range snapshots {
		after, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ReadFile(%s) after authority creation error = %v", path, err)
		}
		if string(after) != string(before) {
			t.Fatalf("source record %s changed after canary satisfaction authority creation", path)
		}
	}
	for _, path := range []string{
		StoreCandidatePromotionDecisionsDir(root),
		StoreHotUpdateOutcomesDir(root),
		StorePromotionsDir(root),
		StoreRollbacksDir(root),
		StoreRollbackAppliesDir(root),
		StoreActiveRuntimePackPointerPath(root),
		StoreLastKnownGoodRuntimePackPointerPath(root),
	} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("path %s exists or errored after authority creation: %v", path, err)
		}
	}
}

func validHotUpdateCanarySatisfactionAuthorityRecord(createdAt time.Time, mutate func(*HotUpdateCanarySatisfactionAuthorityRecord)) HotUpdateCanarySatisfactionAuthorityRecord {
	requirementID := "hot-update-canary-requirement-result-root"
	evidenceID := HotUpdateCanaryEvidenceIDFromRequirementObservedAt(requirementID, createdAt.Add(-time.Minute))
	record := HotUpdateCanarySatisfactionAuthorityRecord{
		RecordVersion:                 StoreRecordVersion,
		CanarySatisfactionAuthorityID: HotUpdateCanarySatisfactionAuthorityIDFromRequirementEvidence(requirementID, evidenceID),
		CanaryRequirementID:           requirementID,
		SelectedCanaryEvidenceID:      evidenceID,
		ResultID:                      "result-root",
		RunID:                         "run-result",
		CandidateID:                   "candidate-1",
		EvalSuiteID:                   "eval-suite-1",
		PromotionPolicyID:             "promotion-policy-result",
		BaselinePackID:                "pack-base",
		CandidatePackID:               "pack-candidate",
		EligibilityState:              CandidatePromotionEligibilityStateCanaryRequired,
		OwnerApprovalRequired:         false,
		SatisfactionState:             HotUpdateCanarySatisfactionStateSatisfied,
		State:                         HotUpdateCanarySatisfactionAuthorityStateAuthorized,
		Reason:                        "hot-update canary satisfaction authority recorded from passed canary evidence",
		CreatedAt:                     createdAt.UTC(),
		CreatedBy:                     "operator",
	}
	if mutate != nil {
		mutate(&record)
	}
	return record
}
