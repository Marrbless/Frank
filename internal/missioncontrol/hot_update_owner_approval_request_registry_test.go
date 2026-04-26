package missioncontrol

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestValidateHotUpdateOwnerApprovalRequestRecordRejectsMissingRequiredFields(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 28, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name string
		edit func(*HotUpdateOwnerApprovalRequestRecord)
		want string
	}{
		{name: "record version", edit: func(record *HotUpdateOwnerApprovalRequestRecord) { record.RecordVersion = 0 }, want: "record_version must be positive"},
		{name: "request id", edit: func(record *HotUpdateOwnerApprovalRequestRecord) { record.OwnerApprovalRequestID = "" }, want: "owner_approval_request_id is required"},
		{name: "authority id", edit: func(record *HotUpdateOwnerApprovalRequestRecord) { record.CanarySatisfactionAuthorityID = "" }, want: "canary_satisfaction_authority_id"},
		{name: "requirement id", edit: func(record *HotUpdateOwnerApprovalRequestRecord) { record.CanaryRequirementID = "" }, want: "canary_requirement_id"},
		{name: "selected evidence id", edit: func(record *HotUpdateOwnerApprovalRequestRecord) { record.SelectedCanaryEvidenceID = "" }, want: "selected_canary_evidence_id"},
		{name: "result id", edit: func(record *HotUpdateOwnerApprovalRequestRecord) { record.ResultID = "" }, want: "result_id"},
		{name: "run id", edit: func(record *HotUpdateOwnerApprovalRequestRecord) { record.RunID = "" }, want: "run_id"},
		{name: "candidate id", edit: func(record *HotUpdateOwnerApprovalRequestRecord) { record.CandidateID = "" }, want: "candidate_id"},
		{name: "eval suite id", edit: func(record *HotUpdateOwnerApprovalRequestRecord) { record.EvalSuiteID = "" }, want: "eval_suite_id"},
		{name: "promotion policy id", edit: func(record *HotUpdateOwnerApprovalRequestRecord) { record.PromotionPolicyID = "" }, want: "promotion_policy_id"},
		{name: "baseline pack id", edit: func(record *HotUpdateOwnerApprovalRequestRecord) { record.BaselinePackID = "" }, want: "baseline_pack_id"},
		{name: "candidate pack id", edit: func(record *HotUpdateOwnerApprovalRequestRecord) { record.CandidatePackID = "" }, want: "candidate_pack_id"},
		{name: "invalid authority state", edit: func(record *HotUpdateOwnerApprovalRequestRecord) { record.AuthorityState = "applied" }, want: "authority_state"},
		{name: "authorized authority state", edit: func(record *HotUpdateOwnerApprovalRequestRecord) {
			record.AuthorityState = HotUpdateCanarySatisfactionAuthorityStateAuthorized
		}, want: "authority_state must be"},
		{name: "invalid satisfaction state", edit: func(record *HotUpdateOwnerApprovalRequestRecord) {
			record.SatisfactionState = HotUpdateCanarySatisfactionStateFailed
		}, want: "satisfaction_state"},
		{name: "satisfied satisfaction state", edit: func(record *HotUpdateOwnerApprovalRequestRecord) {
			record.SatisfactionState = HotUpdateCanarySatisfactionStateSatisfied
		}, want: "satisfaction_state must be"},
		{name: "owner approval required", edit: func(record *HotUpdateOwnerApprovalRequestRecord) { record.OwnerApprovalRequired = false }, want: "owner_approval_required must be true"},
		{name: "invalid request state", edit: func(record *HotUpdateOwnerApprovalRequestRecord) { record.State = "granted" }, want: "state"},
		{name: "reason", edit: func(record *HotUpdateOwnerApprovalRequestRecord) { record.Reason = "" }, want: "reason is required"},
		{name: "created at", edit: func(record *HotUpdateOwnerApprovalRequestRecord) { record.CreatedAt = time.Time{} }, want: "created_at is required"},
		{name: "created by", edit: func(record *HotUpdateOwnerApprovalRequestRecord) { record.CreatedBy = "" }, want: "created_by is required"},
		{name: "deterministic id", edit: func(record *HotUpdateOwnerApprovalRequestRecord) {
			record.OwnerApprovalRequestID = record.OwnerApprovalRequestID + "-other"
		}, want: "does not match deterministic"},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			record := validHotUpdateOwnerApprovalRequestRecord(now, nil)
			tt.edit(&record)
			err := ValidateHotUpdateOwnerApprovalRequestRecord(NormalizeHotUpdateOwnerApprovalRequestRecord(record))
			if err == nil {
				t.Fatal("ValidateHotUpdateOwnerApprovalRequestRecord() error = nil, want rejection")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("ValidateHotUpdateOwnerApprovalRequestRecord() error = %q, want substring %q", err.Error(), tt.want)
			}
		})
	}
}

func TestHotUpdateOwnerApprovalRequestIDFromCanarySatisfactionAuthority(t *testing.T) {
	t.Parallel()

	got := HotUpdateOwnerApprovalRequestIDFromCanarySatisfactionAuthority(" hot-update-canary-satisfaction-authority-a ")
	want := "hot-update-owner-approval-request-hot-update-canary-satisfaction-authority-a"
	if got != want {
		t.Fatalf("HotUpdateOwnerApprovalRequestIDFromCanarySatisfactionAuthority() = %q, want %q", got, want)
	}
}

func TestCreateHotUpdateOwnerApprovalRequestFromCanarySatisfactionAuthorityCreatesRequest(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 28, 13, 0, 0, 0, time.UTC)
	requirement, evidence, authority := storeWaitingOwnerApprovalAuthority(t, root, now)

	record, changed, err := CreateHotUpdateOwnerApprovalRequestFromCanarySatisfactionAuthority(root, " "+authority.CanarySatisfactionAuthorityID+" ", " operator ", now.Add(23*time.Minute))
	if err != nil {
		t.Fatalf("CreateHotUpdateOwnerApprovalRequestFromCanarySatisfactionAuthority() error = %v", err)
	}
	if !changed {
		t.Fatal("changed = false, want true")
	}
	if record.OwnerApprovalRequestID != HotUpdateOwnerApprovalRequestIDFromCanarySatisfactionAuthority(authority.CanarySatisfactionAuthorityID) {
		t.Fatalf("OwnerApprovalRequestID = %q, want deterministic ID", record.OwnerApprovalRequestID)
	}
	if record.CanarySatisfactionAuthorityID != authority.CanarySatisfactionAuthorityID ||
		record.CanaryRequirementID != requirement.CanaryRequirementID ||
		record.SelectedCanaryEvidenceID != evidence.CanaryEvidenceID {
		t.Fatalf("request refs = %#v, want authority/requirement/evidence refs", record)
	}
	if record.ResultID != authority.ResultID || record.RunID != authority.RunID || record.CandidateID != authority.CandidateID ||
		record.EvalSuiteID != authority.EvalSuiteID || record.PromotionPolicyID != authority.PromotionPolicyID ||
		record.BaselinePackID != authority.BaselinePackID || record.CandidatePackID != authority.CandidatePackID {
		t.Fatalf("request source refs = %#v, want authority source refs", record)
	}
	if record.AuthorityState != HotUpdateCanarySatisfactionAuthorityStateWaitingOwnerApproval ||
		record.SatisfactionState != HotUpdateCanarySatisfactionStateWaitingOwnerApproval ||
		!record.OwnerApprovalRequired ||
		record.State != HotUpdateOwnerApprovalRequestStateRequested {
		t.Fatalf("request states = %#v, want waiting_owner_approval/requested", record)
	}
	loaded, err := LoadHotUpdateOwnerApprovalRequestRecord(root, record.OwnerApprovalRequestID)
	if err != nil {
		t.Fatalf("LoadHotUpdateOwnerApprovalRequestRecord() error = %v", err)
	}
	if loaded != record {
		t.Fatalf("loaded = %#v, want %#v", loaded, record)
	}
}

func TestCreateHotUpdateOwnerApprovalRequestFromCanarySatisfactionAuthorityRejectsNonWaitingAuthority(t *testing.T) {
	t.Parallel()

	t.Run("authorized", func(t *testing.T) {
		t.Parallel()

		root := t.TempDir()
		now := time.Date(2026, 4, 28, 14, 0, 0, 0, time.UTC)
		requirement := storeCanaryRequirementForEvidence(t, root, now, nil, nil)
		if _, _, err := CreateHotUpdateCanaryEvidenceFromRequirement(root, requirement.CanaryRequirementID, HotUpdateCanaryEvidenceStatePassed, now.Add(20*time.Minute), "operator", now.Add(21*time.Minute), "canary passed"); err != nil {
			t.Fatalf("CreateHotUpdateCanaryEvidenceFromRequirement() error = %v", err)
		}
		authority, _, err := CreateHotUpdateCanarySatisfactionAuthorityFromRequirement(root, requirement.CanaryRequirementID, "operator", now.Add(22*time.Minute))
		if err != nil {
			t.Fatalf("CreateHotUpdateCanarySatisfactionAuthorityFromRequirement() error = %v", err)
		}

		_, changed, err := CreateHotUpdateOwnerApprovalRequestFromCanarySatisfactionAuthority(root, authority.CanarySatisfactionAuthorityID, "operator", now.Add(23*time.Minute))
		if err == nil {
			t.Fatal("CreateHotUpdateOwnerApprovalRequestFromCanarySatisfactionAuthority() error = nil, want authorized rejection")
		}
		if changed {
			t.Fatal("changed = true, want false")
		}
		if !strings.Contains(err.Error(), "requires canary satisfaction authority state") {
			t.Fatalf("error = %q, want authority state rejection", err.Error())
		}
	})

	t.Run("owner approval false and satisfaction not waiting", func(t *testing.T) {
		t.Parallel()

		root := t.TempDir()
		now := time.Date(2026, 4, 28, 14, 30, 0, 0, time.UTC)
		_, _, authority := storeWaitingOwnerApprovalAuthority(t, root, now)
		authority.OwnerApprovalRequired = false
		authority.SatisfactionState = HotUpdateCanarySatisfactionStateSatisfied
		if err := WriteStoreJSONAtomic(StoreHotUpdateCanarySatisfactionAuthorityPath(root, authority.CanarySatisfactionAuthorityID), authority); err != nil {
			t.Fatalf("WriteStoreJSONAtomic(authority) error = %v", err)
		}

		_, changed, err := CreateHotUpdateOwnerApprovalRequestFromCanarySatisfactionAuthority(root, authority.CanarySatisfactionAuthorityID, "operator", now.Add(23*time.Minute))
		if err == nil {
			t.Fatal("CreateHotUpdateOwnerApprovalRequestFromCanarySatisfactionAuthority() error = nil, want invalid authority rejection")
		}
		if changed {
			t.Fatal("changed = true, want false")
		}
		if !strings.Contains(err.Error(), "satisfaction_state") && !strings.Contains(err.Error(), "owner_approval_required") {
			t.Fatalf("error = %q, want satisfaction/owner rejection", err.Error())
		}
	})
}

func TestCreateHotUpdateOwnerApprovalRequestFromCanarySatisfactionAuthorityLoadsAndCrossChecksLinkedRecords(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		mutate func(t *testing.T, root string, requirement HotUpdateCanaryRequirementRecord, evidence HotUpdateCanaryEvidenceRecord, authority HotUpdateCanarySatisfactionAuthorityRecord)
		want   string
	}{
		{name: "missing authority", mutate: func(t *testing.T, root string, _ HotUpdateCanaryRequirementRecord, _ HotUpdateCanaryEvidenceRecord, authority HotUpdateCanarySatisfactionAuthorityRecord) {
			t.Helper()
			if err := os.Remove(StoreHotUpdateCanarySatisfactionAuthorityPath(root, authority.CanarySatisfactionAuthorityID)); err != nil {
				t.Fatalf("Remove(authority) error = %v", err)
			}
		}, want: ErrHotUpdateCanarySatisfactionAuthorityRecordNotFound.Error()},
		{name: "missing requirement", mutate: func(t *testing.T, root string, requirement HotUpdateCanaryRequirementRecord, _ HotUpdateCanaryEvidenceRecord, _ HotUpdateCanarySatisfactionAuthorityRecord) {
			t.Helper()
			if err := os.Remove(StoreHotUpdateCanaryRequirementPath(root, requirement.CanaryRequirementID)); err != nil {
				t.Fatalf("Remove(requirement) error = %v", err)
			}
		}, want: ErrHotUpdateCanaryRequirementRecordNotFound.Error()},
		{name: "missing selected evidence", mutate: func(t *testing.T, root string, _ HotUpdateCanaryRequirementRecord, evidence HotUpdateCanaryEvidenceRecord, _ HotUpdateCanarySatisfactionAuthorityRecord) {
			t.Helper()
			if err := os.Remove(StoreHotUpdateCanaryEvidencePath(root, evidence.CanaryEvidenceID)); err != nil {
				t.Fatalf("Remove(evidence) error = %v", err)
			}
		}, want: ErrHotUpdateCanaryEvidenceRecordNotFound.Error()},
		{name: "missing candidate result", mutate: func(t *testing.T, root string, requirement HotUpdateCanaryRequirementRecord, _ HotUpdateCanaryEvidenceRecord, _ HotUpdateCanarySatisfactionAuthorityRecord) {
			t.Helper()
			if err := os.Remove(StoreCandidateResultPath(root, requirement.ResultID)); err != nil {
				t.Fatalf("Remove(result) error = %v", err)
			}
		}, want: ErrCandidateResultRecordNotFound.Error()},
		{name: "missing run", mutate: func(t *testing.T, root string, requirement HotUpdateCanaryRequirementRecord, _ HotUpdateCanaryEvidenceRecord, _ HotUpdateCanarySatisfactionAuthorityRecord) {
			t.Helper()
			if err := os.Remove(StoreImprovementRunPath(root, requirement.RunID)); err != nil {
				t.Fatalf("Remove(run) error = %v", err)
			}
		}, want: ErrImprovementRunRecordNotFound.Error()},
		{name: "missing candidate", mutate: func(t *testing.T, root string, requirement HotUpdateCanaryRequirementRecord, _ HotUpdateCanaryEvidenceRecord, _ HotUpdateCanarySatisfactionAuthorityRecord) {
			t.Helper()
			if err := os.Remove(StoreImprovementCandidatePath(root, requirement.CandidateID)); err != nil {
				t.Fatalf("Remove(candidate) error = %v", err)
			}
		}, want: ErrImprovementCandidateRecordNotFound.Error()},
		{name: "unfrozen eval suite", mutate: func(t *testing.T, root string, requirement HotUpdateCanaryRequirementRecord, _ HotUpdateCanaryEvidenceRecord, _ HotUpdateCanarySatisfactionAuthorityRecord) {
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
		{name: "missing policy", mutate: func(t *testing.T, root string, requirement HotUpdateCanaryRequirementRecord, _ HotUpdateCanaryEvidenceRecord, _ HotUpdateCanarySatisfactionAuthorityRecord) {
			t.Helper()
			if err := os.Remove(StorePromotionPolicyPath(root, requirement.PromotionPolicyID)); err != nil {
				t.Fatalf("Remove(policy) error = %v", err)
			}
		}, want: ErrPromotionPolicyRecordNotFound.Error()},
		{name: "missing baseline pack", mutate: func(t *testing.T, root string, requirement HotUpdateCanaryRequirementRecord, _ HotUpdateCanaryEvidenceRecord, _ HotUpdateCanarySatisfactionAuthorityRecord) {
			t.Helper()
			if err := os.Remove(StoreRuntimePackPath(root, requirement.BaselinePackID)); err != nil {
				t.Fatalf("Remove(baseline pack) error = %v", err)
			}
		}, want: ErrRuntimePackRecordNotFound.Error()},
		{name: "missing candidate pack", mutate: func(t *testing.T, root string, requirement HotUpdateCanaryRequirementRecord, _ HotUpdateCanaryEvidenceRecord, _ HotUpdateCanarySatisfactionAuthorityRecord) {
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
			now := time.Date(2026, 4, 28, 15, 0, 0, 0, time.UTC)
			requirement, evidence, authority := storeWaitingOwnerApprovalAuthority(t, root, now)
			tt.mutate(t, root, requirement, evidence, authority)

			_, changed, err := CreateHotUpdateOwnerApprovalRequestFromCanarySatisfactionAuthority(root, authority.CanarySatisfactionAuthorityID, "operator", now.Add(23*time.Minute))
			if err == nil {
				t.Fatal("CreateHotUpdateOwnerApprovalRequestFromCanarySatisfactionAuthority() error = nil, want fail-closed rejection")
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

func TestCreateHotUpdateOwnerApprovalRequestFromCanarySatisfactionAuthorityRejectsStaleSatisfactionAndEligibility(t *testing.T) {
	t.Parallel()

	t.Run("fresh canary satisfaction no longer waiting owner approval", func(t *testing.T) {
		t.Parallel()

		root := t.TempDir()
		now := time.Date(2026, 4, 28, 16, 0, 0, 0, time.UTC)
		requirement, _, authority := storeWaitingOwnerApprovalAuthority(t, root, now)
		if _, _, err := CreateHotUpdateCanaryEvidenceFromRequirement(root, requirement.CanaryRequirementID, HotUpdateCanaryEvidenceStateFailed, now.Add(30*time.Minute), "operator", now.Add(31*time.Minute), "canary failed after authority"); err != nil {
			t.Fatalf("CreateHotUpdateCanaryEvidenceFromRequirement(failed) error = %v", err)
		}

		_, changed, err := CreateHotUpdateOwnerApprovalRequestFromCanarySatisfactionAuthority(root, authority.CanarySatisfactionAuthorityID, "operator", now.Add(32*time.Minute))
		if err == nil {
			t.Fatal("CreateHotUpdateOwnerApprovalRequestFromCanarySatisfactionAuthority() error = nil, want stale satisfaction rejection")
		}
		if changed {
			t.Fatal("changed = true, want false")
		}
		if !strings.Contains(err.Error(), "requires canary satisfaction_state") {
			t.Fatalf("error = %q, want stale satisfaction rejection", err.Error())
		}
	})

	t.Run("fresh eligibility no longer canary and owner approval required", func(t *testing.T) {
		t.Parallel()

		root := t.TempDir()
		now := time.Date(2026, 4, 28, 16, 30, 0, 0, time.UTC)
		requirement, _, authority := storeWaitingOwnerApprovalAuthority(t, root, now)
		policy, err := LoadPromotionPolicyRecord(root, requirement.PromotionPolicyID)
		if err != nil {
			t.Fatalf("LoadPromotionPolicyRecord() error = %v", err)
		}
		policy.RequiresOwnerApproval = false
		if err := WriteStoreJSONAtomic(StorePromotionPolicyPath(root, policy.PromotionPolicyID), policy); err != nil {
			t.Fatalf("WriteStoreJSONAtomic(policy) error = %v", err)
		}

		_, changed, err := CreateHotUpdateOwnerApprovalRequestFromCanarySatisfactionAuthority(root, authority.CanarySatisfactionAuthorityID, "operator", now.Add(23*time.Minute))
		if err == nil {
			t.Fatal("CreateHotUpdateOwnerApprovalRequestFromCanarySatisfactionAuthority() error = nil, want stale eligibility rejection")
		}
		if changed {
			t.Fatal("changed = true, want false")
		}
		if !strings.Contains(err.Error(), "derived promotion eligibility") && !strings.Contains(err.Error(), "does not permit hot-update owner approval request") {
			t.Fatalf("error = %q, want stale eligibility rejection", err.Error())
		}
	})
}

func TestHotUpdateOwnerApprovalRequestReplayDuplicatesAndListOrder(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 28, 17, 0, 0, 0, time.UTC)
	_, _, authority := storeWaitingOwnerApprovalAuthority(t, root, now)
	createdAt := now.Add(23 * time.Minute)
	record, changed, err := CreateHotUpdateOwnerApprovalRequestFromCanarySatisfactionAuthority(root, authority.CanarySatisfactionAuthorityID, "operator", createdAt)
	if err != nil {
		t.Fatalf("CreateHotUpdateOwnerApprovalRequestFromCanarySatisfactionAuthority(first) error = %v", err)
	}
	if !changed {
		t.Fatal("changed = false, want true")
	}
	firstBytes, err := os.ReadFile(StoreHotUpdateOwnerApprovalRequestPath(root, record.OwnerApprovalRequestID))
	if err != nil {
		t.Fatalf("ReadFile(first request) error = %v", err)
	}

	replayed, changed, err := CreateHotUpdateOwnerApprovalRequestFromCanarySatisfactionAuthority(root, authority.CanarySatisfactionAuthorityID, " operator ", createdAt)
	if err != nil {
		t.Fatalf("CreateHotUpdateOwnerApprovalRequestFromCanarySatisfactionAuthority(replay) error = %v", err)
	}
	if changed {
		t.Fatal("changed = true, want false replay")
	}
	if replayed != record {
		t.Fatalf("replayed = %#v, want %#v", replayed, record)
	}
	secondBytes, err := os.ReadFile(StoreHotUpdateOwnerApprovalRequestPath(root, record.OwnerApprovalRequestID))
	if err != nil {
		t.Fatalf("ReadFile(replayed request) error = %v", err)
	}
	if string(secondBytes) != string(firstBytes) {
		t.Fatal("request replay changed stored bytes")
	}

	divergent := record
	divergent.Reason = "different reason"
	if err := WriteStoreJSONAtomic(StoreHotUpdateOwnerApprovalRequestPath(root, divergent.OwnerApprovalRequestID), divergent); err != nil {
		t.Fatalf("WriteStoreJSONAtomic(divergent request) error = %v", err)
	}
	_, changed, err = CreateHotUpdateOwnerApprovalRequestFromCanarySatisfactionAuthority(root, authority.CanarySatisfactionAuthorityID, "operator", createdAt)
	if err == nil {
		t.Fatal("CreateHotUpdateOwnerApprovalRequestFromCanarySatisfactionAuthority(divergent) error = nil, want duplicate rejection")
	}
	if changed {
		t.Fatal("changed = true, want false divergent duplicate")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("error = %q, want already exists", err.Error())
	}
	if err := WriteStoreJSONAtomic(StoreHotUpdateOwnerApprovalRequestPath(root, record.OwnerApprovalRequestID), record); err != nil {
		t.Fatalf("WriteStoreJSONAtomic(restore request) error = %v", err)
	}

	otherID := record.OwnerApprovalRequestID + "-other"
	other := record
	other.OwnerApprovalRequestID = otherID
	if err := WriteStoreJSONAtomic(StoreHotUpdateOwnerApprovalRequestPath(root, otherID), other); err != nil {
		t.Fatalf("WriteStoreJSONAtomic(other id request) error = %v", err)
	}
	_, changed, err = CreateHotUpdateOwnerApprovalRequestFromCanarySatisfactionAuthority(root, authority.CanarySatisfactionAuthorityID, "operator", createdAt)
	if err == nil {
		t.Fatal("CreateHotUpdateOwnerApprovalRequestFromCanarySatisfactionAuthority(other id) error = nil, want fail-closed duplicate/id mismatch")
	}
	if changed {
		t.Fatal("changed = true, want false other id duplicate")
	}
	_, err = ListHotUpdateOwnerApprovalRequestRecords(root)
	if err == nil {
		t.Fatal("ListHotUpdateOwnerApprovalRequestRecords() error = nil, want fail-closed duplicate/id mismatch")
	}
	if err := os.Remove(StoreHotUpdateOwnerApprovalRequestPath(root, otherID)); err != nil {
		t.Fatalf("Remove(other id request) error = %v", err)
	}

	_, _, secondAuthority := storeSecondWaitingOwnerApprovalAuthority(t, root, now.Add(time.Hour))
	second, changed, err := CreateHotUpdateOwnerApprovalRequestFromCanarySatisfactionAuthority(root, secondAuthority.CanarySatisfactionAuthorityID, "operator", now.Add(time.Hour+23*time.Minute))
	if err != nil {
		t.Fatalf("CreateHotUpdateOwnerApprovalRequestFromCanarySatisfactionAuthority(second) error = %v", err)
	}
	if !changed {
		t.Fatal("changed = false, want true for second request")
	}
	records, err := ListHotUpdateOwnerApprovalRequestRecords(root)
	if err != nil {
		t.Fatalf("ListHotUpdateOwnerApprovalRequestRecords() error = %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("ListHotUpdateOwnerApprovalRequestRecords() len = %d, want 2", len(records))
	}
	if records[0].OwnerApprovalRequestID > records[1].OwnerApprovalRequestID {
		t.Fatalf("ListHotUpdateOwnerApprovalRequestRecords() order = %#v, want filename order", records)
	}
	var foundFirst, foundSecond bool
	for _, listed := range records {
		if listed.OwnerApprovalRequestID == record.OwnerApprovalRequestID {
			foundFirst = true
		}
		if listed.OwnerApprovalRequestID == second.OwnerApprovalRequestID {
			foundSecond = true
		}
	}
	if !foundFirst || !foundSecond {
		t.Fatalf("ListHotUpdateOwnerApprovalRequestRecords() = %#v, want created requests", records)
	}
}

func TestHotUpdateOwnerApprovalRequestStoreDoesNotMutateSourceOrRuntimeSurfaces(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 28, 18, 0, 0, 0, time.UTC)
	requirement, evidence, authority := storeWaitingOwnerApprovalAuthority(t, root, now)
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
		StoreHotUpdateCanarySatisfactionAuthorityPath(root, authority.CanarySatisfactionAuthorityID),
	} {
		bytes, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ReadFile(%s) error = %v", path, err)
		}
		snapshots[path] = bytes
	}

	if _, _, err := CreateHotUpdateOwnerApprovalRequestFromCanarySatisfactionAuthority(root, authority.CanarySatisfactionAuthorityID, "operator", now.Add(23*time.Minute)); err != nil {
		t.Fatalf("CreateHotUpdateOwnerApprovalRequestFromCanarySatisfactionAuthority() error = %v", err)
	}
	for path, before := range snapshots {
		after, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ReadFile(%s) after request creation error = %v", path, err)
		}
		if string(after) != string(before) {
			t.Fatalf("source record %s changed after owner approval request creation", path)
		}
	}
	assertNoHotUpdateOwnerApprovalRequestDownstreamRecords(t, root)
	for _, path := range []string{
		StoreActiveRuntimePackPointerPath(root),
		StoreLastKnownGoodRuntimePackPointerPath(root),
	} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("path %s exists or errored after request creation: %v", path, err)
		}
	}
}

func storeWaitingOwnerApprovalAuthority(t *testing.T, root string, now time.Time) (HotUpdateCanaryRequirementRecord, HotUpdateCanaryEvidenceRecord, HotUpdateCanarySatisfactionAuthorityRecord) {
	t.Helper()

	requirement := storeCanaryRequirementForEvidence(t, root, now, func(record *PromotionPolicyRecord) {
		record.RequiresOwnerApproval = true
	}, nil)
	evidence, _, err := CreateHotUpdateCanaryEvidenceFromRequirement(root, requirement.CanaryRequirementID, HotUpdateCanaryEvidenceStatePassed, now.Add(20*time.Minute), "operator", now.Add(21*time.Minute), "canary passed")
	if err != nil {
		t.Fatalf("CreateHotUpdateCanaryEvidenceFromRequirement() error = %v", err)
	}
	authority, _, err := CreateHotUpdateCanarySatisfactionAuthorityFromRequirement(root, requirement.CanaryRequirementID, "operator", now.Add(22*time.Minute))
	if err != nil {
		t.Fatalf("CreateHotUpdateCanarySatisfactionAuthorityFromRequirement() error = %v", err)
	}
	return requirement, evidence, authority
}

func storeSecondWaitingOwnerApprovalAuthority(t *testing.T, root string, now time.Time) (HotUpdateCanaryRequirementRecord, HotUpdateCanaryEvidenceRecord, HotUpdateCanarySatisfactionAuthorityRecord) {
	t.Helper()

	policyID := "promotion-policy-result-b"
	resultID := "result-eligible-b"
	if err := StorePromotionPolicyRecord(root, validPromotionPolicyRecord(now.Add(6*time.Minute), func(record *PromotionPolicyRecord) {
		record.PromotionPolicyID = policyID
		record.RequiresCanary = true
		record.RequiresOwnerApproval = true
		record.RequiresHoldoutPass = true
		record.EpsilonRule = "epsilon <= 0.01"
		record.RegressionRule = "no_regression_flags"
		record.CompatibilityRule = "compatibility_score >= 0.90"
		record.ResourceRule = "resource_score >= 0.60"
	})); err != nil {
		t.Fatalf("StorePromotionPolicyRecord(second) error = %v", err)
	}
	if err := StoreCandidateResultRecord(root, validCandidateResultRecord(now.Add(7*time.Minute), func(record *CandidateResultRecord) {
		record.ResultID = resultID
		record.RunID = "run-result"
		record.CandidateID = "candidate-1"
		record.EvalSuiteID = "eval-suite-1"
		record.PromotionPolicyID = policyID
		record.BaselinePackID = "pack-base"
		record.CandidatePackID = "pack-candidate"
		record.BaselineScore = 0.52
		record.TrainScore = 0.78
		record.HoldoutScore = 0.74
		record.CompatibilityScore = 0.93
		record.ResourceScore = 0.67
		record.RegressionFlags = []string{"none"}
		record.Decision = ImprovementRunDecisionKeep
	})); err != nil {
		t.Fatalf("StoreCandidateResultRecord(second) error = %v", err)
	}
	requirement, _, err := CreateHotUpdateCanaryRequirementFromCandidateResult(root, resultID, "operator", now.Add(10*time.Minute))
	if err != nil {
		t.Fatalf("CreateHotUpdateCanaryRequirementFromCandidateResult(second) error = %v", err)
	}
	evidence, _, err := CreateHotUpdateCanaryEvidenceFromRequirement(root, requirement.CanaryRequirementID, HotUpdateCanaryEvidenceStatePassed, now.Add(20*time.Minute), "operator", now.Add(21*time.Minute), "canary passed")
	if err != nil {
		t.Fatalf("CreateHotUpdateCanaryEvidenceFromRequirement(second) error = %v", err)
	}
	authority, _, err := CreateHotUpdateCanarySatisfactionAuthorityFromRequirement(root, requirement.CanaryRequirementID, "operator", now.Add(22*time.Minute))
	if err != nil {
		t.Fatalf("CreateHotUpdateCanarySatisfactionAuthorityFromRequirement(second) error = %v", err)
	}
	return requirement, evidence, authority
}

func validHotUpdateOwnerApprovalRequestRecord(createdAt time.Time, mutate func(*HotUpdateOwnerApprovalRequestRecord)) HotUpdateOwnerApprovalRequestRecord {
	authorityID := "hot-update-canary-satisfaction-authority-hot-update-canary-requirement-result-root-hot-update-canary-evidence-hot-update-canary-requirement-result-root-20260428T115900Z"
	record := HotUpdateOwnerApprovalRequestRecord{
		RecordVersion:                 StoreRecordVersion,
		OwnerApprovalRequestID:        HotUpdateOwnerApprovalRequestIDFromCanarySatisfactionAuthority(authorityID),
		CanarySatisfactionAuthorityID: authorityID,
		CanaryRequirementID:           "hot-update-canary-requirement-result-root",
		SelectedCanaryEvidenceID:      "hot-update-canary-evidence-hot-update-canary-requirement-result-root-20260428T115900Z",
		ResultID:                      "result-root",
		RunID:                         "run-result",
		CandidateID:                   "candidate-1",
		EvalSuiteID:                   "eval-suite-1",
		PromotionPolicyID:             "promotion-policy-result",
		BaselinePackID:                "pack-base",
		CandidatePackID:               "pack-candidate",
		AuthorityState:                HotUpdateCanarySatisfactionAuthorityStateWaitingOwnerApproval,
		SatisfactionState:             HotUpdateCanarySatisfactionStateWaitingOwnerApproval,
		OwnerApprovalRequired:         true,
		State:                         HotUpdateOwnerApprovalRequestStateRequested,
		Reason:                        "hot-update owner approval requested after canary satisfaction",
		CreatedAt:                     createdAt.UTC(),
		CreatedBy:                     "operator",
	}
	if mutate != nil {
		mutate(&record)
	}
	return record
}

func assertNoHotUpdateOwnerApprovalRequestDownstreamRecords(t *testing.T, root string) {
	t.Helper()

	decisions, err := ListCandidatePromotionDecisionRecords(root)
	if err != nil {
		t.Fatalf("ListCandidatePromotionDecisionRecords() error = %v", err)
	}
	if len(decisions) != 0 {
		t.Fatalf("ListCandidatePromotionDecisionRecords() len = %d, want 0", len(decisions))
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
	rollbackApplies, err := ListRollbackApplyRecords(root)
	if err != nil {
		t.Fatalf("ListRollbackApplyRecords() error = %v", err)
	}
	if len(rollbackApplies) != 0 {
		t.Fatalf("ListRollbackApplyRecords() len = %d, want 0", len(rollbackApplies))
	}
}
