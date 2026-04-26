package missioncontrol

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestValidateHotUpdateOwnerApprovalDecisionRecordRejectsMissingRequiredFields(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 29, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name string
		edit func(*HotUpdateOwnerApprovalDecisionRecord)
		want string
	}{
		{name: "record version", edit: func(record *HotUpdateOwnerApprovalDecisionRecord) { record.RecordVersion = 0 }, want: "record_version must be positive"},
		{name: "decision id", edit: func(record *HotUpdateOwnerApprovalDecisionRecord) { record.OwnerApprovalDecisionID = "" }, want: "owner_approval_decision_id is required"},
		{name: "request id", edit: func(record *HotUpdateOwnerApprovalDecisionRecord) { record.OwnerApprovalRequestID = "" }, want: "owner_approval_request_id"},
		{name: "authority id", edit: func(record *HotUpdateOwnerApprovalDecisionRecord) { record.CanarySatisfactionAuthorityID = "" }, want: "canary_satisfaction_authority_id"},
		{name: "requirement id", edit: func(record *HotUpdateOwnerApprovalDecisionRecord) { record.CanaryRequirementID = "" }, want: "canary_requirement_id"},
		{name: "selected evidence id", edit: func(record *HotUpdateOwnerApprovalDecisionRecord) { record.SelectedCanaryEvidenceID = "" }, want: "selected_canary_evidence_id"},
		{name: "result id", edit: func(record *HotUpdateOwnerApprovalDecisionRecord) { record.ResultID = "" }, want: "result_id"},
		{name: "run id", edit: func(record *HotUpdateOwnerApprovalDecisionRecord) { record.RunID = "" }, want: "run_id"},
		{name: "candidate id", edit: func(record *HotUpdateOwnerApprovalDecisionRecord) { record.CandidateID = "" }, want: "candidate_id"},
		{name: "eval suite id", edit: func(record *HotUpdateOwnerApprovalDecisionRecord) { record.EvalSuiteID = "" }, want: "eval_suite_id"},
		{name: "promotion policy id", edit: func(record *HotUpdateOwnerApprovalDecisionRecord) { record.PromotionPolicyID = "" }, want: "promotion_policy_id"},
		{name: "baseline pack id", edit: func(record *HotUpdateOwnerApprovalDecisionRecord) { record.BaselinePackID = "" }, want: "baseline_pack_id"},
		{name: "candidate pack id", edit: func(record *HotUpdateOwnerApprovalDecisionRecord) { record.CandidatePackID = "" }, want: "candidate_pack_id"},
		{name: "invalid request state", edit: func(record *HotUpdateOwnerApprovalDecisionRecord) { record.RequestState = "granted" }, want: "request_state"},
		{name: "invalid authority state", edit: func(record *HotUpdateOwnerApprovalDecisionRecord) { record.AuthorityState = "applied" }, want: "authority_state"},
		{name: "authorized authority state", edit: func(record *HotUpdateOwnerApprovalDecisionRecord) {
			record.AuthorityState = HotUpdateCanarySatisfactionAuthorityStateAuthorized
		}, want: "authority_state must be"},
		{name: "invalid satisfaction state", edit: func(record *HotUpdateOwnerApprovalDecisionRecord) {
			record.SatisfactionState = HotUpdateCanarySatisfactionStateFailed
		}, want: "satisfaction_state"},
		{name: "satisfied satisfaction state", edit: func(record *HotUpdateOwnerApprovalDecisionRecord) {
			record.SatisfactionState = HotUpdateCanarySatisfactionStateSatisfied
		}, want: "satisfaction_state must be"},
		{name: "owner approval required", edit: func(record *HotUpdateOwnerApprovalDecisionRecord) { record.OwnerApprovalRequired = false }, want: "owner_approval_required must be true"},
		{name: "invalid decision", edit: func(record *HotUpdateOwnerApprovalDecisionRecord) { record.Decision = "expired" }, want: "decision"},
		{name: "reason", edit: func(record *HotUpdateOwnerApprovalDecisionRecord) { record.Reason = "" }, want: "reason is required"},
		{name: "decided at", edit: func(record *HotUpdateOwnerApprovalDecisionRecord) { record.DecidedAt = time.Time{} }, want: "decided_at is required"},
		{name: "decided by", edit: func(record *HotUpdateOwnerApprovalDecisionRecord) { record.DecidedBy = "" }, want: "decided_by is required"},
		{name: "deterministic id", edit: func(record *HotUpdateOwnerApprovalDecisionRecord) {
			record.OwnerApprovalDecisionID = record.OwnerApprovalDecisionID + "-other"
		}, want: "does not match deterministic"},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			record := validHotUpdateOwnerApprovalDecisionRecord(now, nil)
			tt.edit(&record)
			err := ValidateHotUpdateOwnerApprovalDecisionRecord(NormalizeHotUpdateOwnerApprovalDecisionRecord(record))
			if err == nil {
				t.Fatal("ValidateHotUpdateOwnerApprovalDecisionRecord() error = nil, want rejection")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("ValidateHotUpdateOwnerApprovalDecisionRecord() error = %q, want substring %q", err.Error(), tt.want)
			}
		})
	}
}

func TestValidateHotUpdateOwnerApprovalDecisionRecordAcceptsGrantedAndRejected(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 29, 12, 30, 0, 0, time.UTC)
	for _, decision := range []HotUpdateOwnerApprovalDecision{
		HotUpdateOwnerApprovalDecisionGranted,
		HotUpdateOwnerApprovalDecisionRejected,
	} {
		decision := decision
		t.Run(string(decision), func(t *testing.T) {
			t.Parallel()

			record := validHotUpdateOwnerApprovalDecisionRecord(now, func(record *HotUpdateOwnerApprovalDecisionRecord) {
				record.Decision = decision
			})
			if err := ValidateHotUpdateOwnerApprovalDecisionRecord(record); err != nil {
				t.Fatalf("ValidateHotUpdateOwnerApprovalDecisionRecord(%s) error = %v", decision, err)
			}
		})
	}
}

func TestHotUpdateOwnerApprovalDecisionIDFromRequest(t *testing.T) {
	t.Parallel()

	got := HotUpdateOwnerApprovalDecisionIDFromRequest(" hot-update-owner-approval-request-a ")
	want := "hot-update-owner-approval-decision-hot-update-owner-approval-request-a"
	if got != want {
		t.Fatalf("HotUpdateOwnerApprovalDecisionIDFromRequest() = %q, want %q", got, want)
	}
}

func TestCreateHotUpdateOwnerApprovalDecisionFromRequestCreatesGrantedAndRejected(t *testing.T) {
	t.Parallel()

	for _, decision := range []HotUpdateOwnerApprovalDecision{
		HotUpdateOwnerApprovalDecisionGranted,
		HotUpdateOwnerApprovalDecisionRejected,
	} {
		decision := decision
		t.Run(string(decision), func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			now := time.Date(2026, 4, 29, 13, 0, 0, 0, time.UTC)
			requirement, evidence, authority := storeWaitingOwnerApprovalAuthority(t, root, now)
			request, _, err := CreateHotUpdateOwnerApprovalRequestFromCanarySatisfactionAuthority(root, authority.CanarySatisfactionAuthorityID, "operator", now.Add(23*time.Minute))
			if err != nil {
				t.Fatalf("CreateHotUpdateOwnerApprovalRequestFromCanarySatisfactionAuthority() error = %v", err)
			}

			record, changed, err := CreateHotUpdateOwnerApprovalDecisionFromRequest(root, " "+request.OwnerApprovalRequestID+" ", decision, " operator ", now.Add(24*time.Minute), " owner decision ")
			if err != nil {
				t.Fatalf("CreateHotUpdateOwnerApprovalDecisionFromRequest() error = %v", err)
			}
			if !changed {
				t.Fatal("changed = false, want true")
			}
			if record.OwnerApprovalDecisionID != HotUpdateOwnerApprovalDecisionIDFromRequest(request.OwnerApprovalRequestID) {
				t.Fatalf("OwnerApprovalDecisionID = %q, want deterministic ID", record.OwnerApprovalDecisionID)
			}
			if record.OwnerApprovalRequestID != request.OwnerApprovalRequestID ||
				record.CanarySatisfactionAuthorityID != authority.CanarySatisfactionAuthorityID ||
				record.CanaryRequirementID != requirement.CanaryRequirementID ||
				record.SelectedCanaryEvidenceID != evidence.CanaryEvidenceID {
				t.Fatalf("decision refs = %#v, want request/authority/requirement/evidence refs", record)
			}
			if record.ResultID != request.ResultID || record.RunID != request.RunID || record.CandidateID != request.CandidateID ||
				record.EvalSuiteID != request.EvalSuiteID || record.PromotionPolicyID != request.PromotionPolicyID ||
				record.BaselinePackID != request.BaselinePackID || record.CandidatePackID != request.CandidatePackID {
				t.Fatalf("decision source refs = %#v, want request source refs", record)
			}
			if record.RequestState != HotUpdateOwnerApprovalRequestStateRequested ||
				record.AuthorityState != HotUpdateCanarySatisfactionAuthorityStateWaitingOwnerApproval ||
				record.SatisfactionState != HotUpdateCanarySatisfactionStateWaitingOwnerApproval ||
				!record.OwnerApprovalRequired ||
				record.Decision != decision ||
				record.Reason != "owner decision" ||
				record.DecidedBy != "operator" {
				t.Fatalf("decision state/provenance = %#v, want requested/waiting/%s/operator", record, decision)
			}
			loaded, err := LoadHotUpdateOwnerApprovalDecisionRecord(root, record.OwnerApprovalDecisionID)
			if err != nil {
				t.Fatalf("LoadHotUpdateOwnerApprovalDecisionRecord() error = %v", err)
			}
			if loaded != record {
				t.Fatalf("loaded = %#v, want %#v", loaded, record)
			}
		})
	}
}

func TestCreateHotUpdateOwnerApprovalDecisionFromRequestRejectsInvalidInputsAndRequest(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 29, 14, 0, 0, 0, time.UTC)
	_, _, authority := storeWaitingOwnerApprovalAuthority(t, root, now)
	request, _, err := CreateHotUpdateOwnerApprovalRequestFromCanarySatisfactionAuthority(root, authority.CanarySatisfactionAuthorityID, "operator", now.Add(23*time.Minute))
	if err != nil {
		t.Fatalf("CreateHotUpdateOwnerApprovalRequestFromCanarySatisfactionAuthority() error = %v", err)
	}

	tests := []struct {
		name      string
		requestID string
		decision  HotUpdateOwnerApprovalDecision
		decidedBy string
		decidedAt time.Time
		reason    string
		want      string
	}{
		{name: "missing request", requestID: "", decision: HotUpdateOwnerApprovalDecisionGranted, decidedBy: "operator", decidedAt: now, reason: "approved", want: "owner_approval_request_id"},
		{name: "invalid decision", requestID: request.OwnerApprovalRequestID, decision: "expired", decidedBy: "operator", decidedAt: now, reason: "approved", want: "decision"},
		{name: "missing decided by", requestID: request.OwnerApprovalRequestID, decision: HotUpdateOwnerApprovalDecisionGranted, decidedAt: now, reason: "approved", want: "decided_by"},
		{name: "zero decided at", requestID: request.OwnerApprovalRequestID, decision: HotUpdateOwnerApprovalDecisionGranted, decidedBy: "operator", reason: "approved", want: "decided_at"},
		{name: "missing reason", requestID: request.OwnerApprovalRequestID, decision: HotUpdateOwnerApprovalDecisionGranted, decidedBy: "operator", decidedAt: now, want: "reason"},
		{name: "missing request record", requestID: "hot-update-owner-approval-request-missing", decision: HotUpdateOwnerApprovalDecisionGranted, decidedBy: "operator", decidedAt: now, reason: "approved", want: ErrHotUpdateOwnerApprovalRequestRecordNotFound.Error()},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, changed, err := CreateHotUpdateOwnerApprovalDecisionFromRequest(root, tt.requestID, tt.decision, tt.decidedBy, tt.decidedAt, tt.reason)
			if err == nil {
				t.Fatal("CreateHotUpdateOwnerApprovalDecisionFromRequest() error = nil, want rejection")
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

func TestCreateHotUpdateOwnerApprovalDecisionFromRequestRejectsInvalidRequestState(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 29, 14, 30, 0, 0, time.UTC)
	_, _, authority := storeWaitingOwnerApprovalAuthority(t, root, now)
	request, _, err := CreateHotUpdateOwnerApprovalRequestFromCanarySatisfactionAuthority(root, authority.CanarySatisfactionAuthorityID, "operator", now.Add(23*time.Minute))
	if err != nil {
		t.Fatalf("CreateHotUpdateOwnerApprovalRequestFromCanarySatisfactionAuthority() error = %v", err)
	}
	request.State = "granted"
	if err := WriteStoreJSONAtomic(StoreHotUpdateOwnerApprovalRequestPath(root, request.OwnerApprovalRequestID), request); err != nil {
		t.Fatalf("WriteStoreJSONAtomic(request) error = %v", err)
	}

	_, changed, err := CreateHotUpdateOwnerApprovalDecisionFromRequest(root, request.OwnerApprovalRequestID, HotUpdateOwnerApprovalDecisionGranted, "operator", now.Add(24*time.Minute), "approved")
	if err == nil {
		t.Fatal("CreateHotUpdateOwnerApprovalDecisionFromRequest() error = nil, want request state rejection")
	}
	if changed {
		t.Fatal("changed = true, want false")
	}
	if !strings.Contains(err.Error(), "state") {
		t.Fatalf("error = %q, want request state rejection", err.Error())
	}
}

func TestCreateHotUpdateOwnerApprovalDecisionFromRequestRejectsStaleSatisfactionAndEligibility(t *testing.T) {
	t.Parallel()

	t.Run("fresh canary satisfaction no longer waiting owner approval", func(t *testing.T) {
		t.Parallel()

		root := t.TempDir()
		now := time.Date(2026, 4, 29, 15, 0, 0, 0, time.UTC)
		requirement, _, authority := storeWaitingOwnerApprovalAuthority(t, root, now)
		request, _, err := CreateHotUpdateOwnerApprovalRequestFromCanarySatisfactionAuthority(root, authority.CanarySatisfactionAuthorityID, "operator", now.Add(23*time.Minute))
		if err != nil {
			t.Fatalf("CreateHotUpdateOwnerApprovalRequestFromCanarySatisfactionAuthority() error = %v", err)
		}
		if _, _, err := CreateHotUpdateCanaryEvidenceFromRequirement(root, requirement.CanaryRequirementID, HotUpdateCanaryEvidenceStateFailed, now.Add(30*time.Minute), "operator", now.Add(31*time.Minute), "canary failed after request"); err != nil {
			t.Fatalf("CreateHotUpdateCanaryEvidenceFromRequirement(failed) error = %v", err)
		}

		_, changed, err := CreateHotUpdateOwnerApprovalDecisionFromRequest(root, request.OwnerApprovalRequestID, HotUpdateOwnerApprovalDecisionGranted, "operator", now.Add(32*time.Minute), "approved")
		if err == nil {
			t.Fatal("CreateHotUpdateOwnerApprovalDecisionFromRequest() error = nil, want stale satisfaction rejection")
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
		now := time.Date(2026, 4, 29, 15, 30, 0, 0, time.UTC)
		requirement, _, authority := storeWaitingOwnerApprovalAuthority(t, root, now)
		request, _, err := CreateHotUpdateOwnerApprovalRequestFromCanarySatisfactionAuthority(root, authority.CanarySatisfactionAuthorityID, "operator", now.Add(23*time.Minute))
		if err != nil {
			t.Fatalf("CreateHotUpdateOwnerApprovalRequestFromCanarySatisfactionAuthority() error = %v", err)
		}
		policy, err := LoadPromotionPolicyRecord(root, requirement.PromotionPolicyID)
		if err != nil {
			t.Fatalf("LoadPromotionPolicyRecord() error = %v", err)
		}
		policy.RequiresOwnerApproval = false
		if err := WriteStoreJSONAtomic(StorePromotionPolicyPath(root, policy.PromotionPolicyID), policy); err != nil {
			t.Fatalf("WriteStoreJSONAtomic(policy) error = %v", err)
		}

		_, changed, err := CreateHotUpdateOwnerApprovalDecisionFromRequest(root, request.OwnerApprovalRequestID, HotUpdateOwnerApprovalDecisionGranted, "operator", now.Add(24*time.Minute), "approved")
		if err == nil {
			t.Fatal("CreateHotUpdateOwnerApprovalDecisionFromRequest() error = nil, want stale eligibility rejection")
		}
		if changed {
			t.Fatal("changed = true, want false")
		}
		if !strings.Contains(err.Error(), "derived promotion eligibility") && !strings.Contains(err.Error(), "does not permit hot-update owner approval request") {
			t.Fatalf("error = %q, want stale eligibility rejection", err.Error())
		}
	})
}

func TestCreateHotUpdateOwnerApprovalDecisionFromRequestLoadsAndCrossChecksLinkedRecords(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		mutate func(t *testing.T, root string, request HotUpdateOwnerApprovalRequestRecord, requirement HotUpdateCanaryRequirementRecord, evidence HotUpdateCanaryEvidenceRecord, authority HotUpdateCanarySatisfactionAuthorityRecord)
		want   string
	}{
		{name: "missing request", mutate: func(t *testing.T, root string, request HotUpdateOwnerApprovalRequestRecord, _ HotUpdateCanaryRequirementRecord, _ HotUpdateCanaryEvidenceRecord, _ HotUpdateCanarySatisfactionAuthorityRecord) {
			t.Helper()
			if err := os.Remove(StoreHotUpdateOwnerApprovalRequestPath(root, request.OwnerApprovalRequestID)); err != nil {
				t.Fatalf("Remove(request) error = %v", err)
			}
		}, want: ErrHotUpdateOwnerApprovalRequestRecordNotFound.Error()},
		{name: "missing authority", mutate: func(t *testing.T, root string, _ HotUpdateOwnerApprovalRequestRecord, _ HotUpdateCanaryRequirementRecord, _ HotUpdateCanaryEvidenceRecord, authority HotUpdateCanarySatisfactionAuthorityRecord) {
			t.Helper()
			if err := os.Remove(StoreHotUpdateCanarySatisfactionAuthorityPath(root, authority.CanarySatisfactionAuthorityID)); err != nil {
				t.Fatalf("Remove(authority) error = %v", err)
			}
		}, want: ErrHotUpdateCanarySatisfactionAuthorityRecordNotFound.Error()},
		{name: "missing requirement", mutate: func(t *testing.T, root string, _ HotUpdateOwnerApprovalRequestRecord, requirement HotUpdateCanaryRequirementRecord, _ HotUpdateCanaryEvidenceRecord, _ HotUpdateCanarySatisfactionAuthorityRecord) {
			t.Helper()
			if err := os.Remove(StoreHotUpdateCanaryRequirementPath(root, requirement.CanaryRequirementID)); err != nil {
				t.Fatalf("Remove(requirement) error = %v", err)
			}
		}, want: ErrHotUpdateCanaryRequirementRecordNotFound.Error()},
		{name: "missing selected evidence", mutate: func(t *testing.T, root string, _ HotUpdateOwnerApprovalRequestRecord, _ HotUpdateCanaryRequirementRecord, evidence HotUpdateCanaryEvidenceRecord, _ HotUpdateCanarySatisfactionAuthorityRecord) {
			t.Helper()
			if err := os.Remove(StoreHotUpdateCanaryEvidencePath(root, evidence.CanaryEvidenceID)); err != nil {
				t.Fatalf("Remove(evidence) error = %v", err)
			}
		}, want: ErrHotUpdateCanaryEvidenceRecordNotFound.Error()},
		{name: "missing candidate result", mutate: func(t *testing.T, root string, _ HotUpdateOwnerApprovalRequestRecord, requirement HotUpdateCanaryRequirementRecord, _ HotUpdateCanaryEvidenceRecord, _ HotUpdateCanarySatisfactionAuthorityRecord) {
			t.Helper()
			if err := os.Remove(StoreCandidateResultPath(root, requirement.ResultID)); err != nil {
				t.Fatalf("Remove(result) error = %v", err)
			}
		}, want: ErrCandidateResultRecordNotFound.Error()},
		{name: "missing run", mutate: func(t *testing.T, root string, _ HotUpdateOwnerApprovalRequestRecord, requirement HotUpdateCanaryRequirementRecord, _ HotUpdateCanaryEvidenceRecord, _ HotUpdateCanarySatisfactionAuthorityRecord) {
			t.Helper()
			if err := os.Remove(StoreImprovementRunPath(root, requirement.RunID)); err != nil {
				t.Fatalf("Remove(run) error = %v", err)
			}
		}, want: ErrImprovementRunRecordNotFound.Error()},
		{name: "missing candidate", mutate: func(t *testing.T, root string, _ HotUpdateOwnerApprovalRequestRecord, requirement HotUpdateCanaryRequirementRecord, _ HotUpdateCanaryEvidenceRecord, _ HotUpdateCanarySatisfactionAuthorityRecord) {
			t.Helper()
			if err := os.Remove(StoreImprovementCandidatePath(root, requirement.CandidateID)); err != nil {
				t.Fatalf("Remove(candidate) error = %v", err)
			}
		}, want: ErrImprovementCandidateRecordNotFound.Error()},
		{name: "unfrozen eval suite", mutate: func(t *testing.T, root string, _ HotUpdateOwnerApprovalRequestRecord, requirement HotUpdateCanaryRequirementRecord, _ HotUpdateCanaryEvidenceRecord, _ HotUpdateCanarySatisfactionAuthorityRecord) {
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
		{name: "missing policy", mutate: func(t *testing.T, root string, _ HotUpdateOwnerApprovalRequestRecord, requirement HotUpdateCanaryRequirementRecord, _ HotUpdateCanaryEvidenceRecord, _ HotUpdateCanarySatisfactionAuthorityRecord) {
			t.Helper()
			if err := os.Remove(StorePromotionPolicyPath(root, requirement.PromotionPolicyID)); err != nil {
				t.Fatalf("Remove(policy) error = %v", err)
			}
		}, want: ErrPromotionPolicyRecordNotFound.Error()},
		{name: "missing baseline pack", mutate: func(t *testing.T, root string, _ HotUpdateOwnerApprovalRequestRecord, requirement HotUpdateCanaryRequirementRecord, _ HotUpdateCanaryEvidenceRecord, _ HotUpdateCanarySatisfactionAuthorityRecord) {
			t.Helper()
			if err := os.Remove(StoreRuntimePackPath(root, requirement.BaselinePackID)); err != nil {
				t.Fatalf("Remove(baseline pack) error = %v", err)
			}
		}, want: ErrRuntimePackRecordNotFound.Error()},
		{name: "missing candidate pack", mutate: func(t *testing.T, root string, _ HotUpdateOwnerApprovalRequestRecord, requirement HotUpdateCanaryRequirementRecord, _ HotUpdateCanaryEvidenceRecord, _ HotUpdateCanarySatisfactionAuthorityRecord) {
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
			now := time.Date(2026, 4, 29, 16, 0, 0, 0, time.UTC)
			requirement, evidence, authority := storeWaitingOwnerApprovalAuthority(t, root, now)
			request, _, err := CreateHotUpdateOwnerApprovalRequestFromCanarySatisfactionAuthority(root, authority.CanarySatisfactionAuthorityID, "operator", now.Add(23*time.Minute))
			if err != nil {
				t.Fatalf("CreateHotUpdateOwnerApprovalRequestFromCanarySatisfactionAuthority() error = %v", err)
			}
			tt.mutate(t, root, request, requirement, evidence, authority)

			_, changed, err := CreateHotUpdateOwnerApprovalDecisionFromRequest(root, request.OwnerApprovalRequestID, HotUpdateOwnerApprovalDecisionGranted, "operator", now.Add(24*time.Minute), "approved")
			if err == nil {
				t.Fatal("CreateHotUpdateOwnerApprovalDecisionFromRequest() error = nil, want fail-closed rejection")
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

func TestHotUpdateOwnerApprovalDecisionReplayDuplicatesAndListOrder(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 29, 17, 0, 0, 0, time.UTC)
	_, _, authority := storeWaitingOwnerApprovalAuthority(t, root, now)
	request, _, err := CreateHotUpdateOwnerApprovalRequestFromCanarySatisfactionAuthority(root, authority.CanarySatisfactionAuthorityID, "operator", now.Add(23*time.Minute))
	if err != nil {
		t.Fatalf("CreateHotUpdateOwnerApprovalRequestFromCanarySatisfactionAuthority() error = %v", err)
	}
	decidedAt := now.Add(24 * time.Minute)
	record, changed, err := CreateHotUpdateOwnerApprovalDecisionFromRequest(root, request.OwnerApprovalRequestID, HotUpdateOwnerApprovalDecisionGranted, "operator", decidedAt, "approved")
	if err != nil {
		t.Fatalf("CreateHotUpdateOwnerApprovalDecisionFromRequest(first) error = %v", err)
	}
	if !changed {
		t.Fatal("changed = false, want true")
	}
	firstBytes, err := os.ReadFile(StoreHotUpdateOwnerApprovalDecisionPath(root, record.OwnerApprovalDecisionID))
	if err != nil {
		t.Fatalf("ReadFile(first decision) error = %v", err)
	}

	replayed, changed, err := CreateHotUpdateOwnerApprovalDecisionFromRequest(root, request.OwnerApprovalRequestID, HotUpdateOwnerApprovalDecisionGranted, " operator ", decidedAt, " approved ")
	if err != nil {
		t.Fatalf("CreateHotUpdateOwnerApprovalDecisionFromRequest(replay) error = %v", err)
	}
	if changed {
		t.Fatal("changed = true, want false replay")
	}
	if replayed != record {
		t.Fatalf("replayed = %#v, want %#v", replayed, record)
	}
	secondBytes, err := os.ReadFile(StoreHotUpdateOwnerApprovalDecisionPath(root, record.OwnerApprovalDecisionID))
	if err != nil {
		t.Fatalf("ReadFile(replayed decision) error = %v", err)
	}
	if string(secondBytes) != string(firstBytes) {
		t.Fatal("decision replay changed stored bytes")
	}

	_, changed, err = CreateHotUpdateOwnerApprovalDecisionFromRequest(root, request.OwnerApprovalRequestID, HotUpdateOwnerApprovalDecisionRejected, "operator", decidedAt, "rejected")
	if err == nil {
		t.Fatal("CreateHotUpdateOwnerApprovalDecisionFromRequest(different decision) error = nil, want duplicate rejection")
	}
	if changed {
		t.Fatal("changed = true, want false different decision")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("error = %q, want already exists", err.Error())
	}

	divergent := record
	divergent.Reason = "different reason"
	if err := WriteStoreJSONAtomic(StoreHotUpdateOwnerApprovalDecisionPath(root, divergent.OwnerApprovalDecisionID), divergent); err != nil {
		t.Fatalf("WriteStoreJSONAtomic(divergent decision) error = %v", err)
	}
	_, changed, err = CreateHotUpdateOwnerApprovalDecisionFromRequest(root, request.OwnerApprovalRequestID, HotUpdateOwnerApprovalDecisionGranted, "operator", decidedAt, "approved")
	if err == nil {
		t.Fatal("CreateHotUpdateOwnerApprovalDecisionFromRequest(divergent) error = nil, want duplicate rejection")
	}
	if changed {
		t.Fatal("changed = true, want false divergent duplicate")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("error = %q, want already exists", err.Error())
	}
	if err := WriteStoreJSONAtomic(StoreHotUpdateOwnerApprovalDecisionPath(root, record.OwnerApprovalDecisionID), record); err != nil {
		t.Fatalf("WriteStoreJSONAtomic(restore decision) error = %v", err)
	}

	otherID := "hot-update-owner-approval-decision-other"
	other := record
	other.OwnerApprovalDecisionID = otherID
	if err := WriteStoreJSONAtomic(StoreHotUpdateOwnerApprovalDecisionPath(root, otherID), other); err != nil {
		t.Fatalf("WriteStoreJSONAtomic(other id decision) error = %v", err)
	}
	_, changed, err = CreateHotUpdateOwnerApprovalDecisionFromRequest(root, request.OwnerApprovalRequestID, HotUpdateOwnerApprovalDecisionGranted, "operator", decidedAt, "approved")
	if err == nil {
		t.Fatal("CreateHotUpdateOwnerApprovalDecisionFromRequest(other id) error = nil, want fail-closed duplicate/id mismatch")
	}
	if changed {
		t.Fatal("changed = true, want false other id duplicate")
	}
	_, err = ListHotUpdateOwnerApprovalDecisionRecords(root)
	if err == nil {
		t.Fatal("ListHotUpdateOwnerApprovalDecisionRecords() error = nil, want fail-closed duplicate/id mismatch")
	}
	if err := os.RemoveAll(StoreHotUpdateOwnerApprovalDecisionDir(root, otherID)); err != nil {
		t.Fatalf("RemoveAll(other id decision) error = %v", err)
	}

	_, _, secondAuthority := storeSecondWaitingOwnerApprovalAuthority(t, root, now.Add(time.Hour))
	secondRequest, _, err := CreateHotUpdateOwnerApprovalRequestFromCanarySatisfactionAuthority(root, secondAuthority.CanarySatisfactionAuthorityID, "operator", now.Add(time.Hour+23*time.Minute))
	if err != nil {
		t.Fatalf("CreateHotUpdateOwnerApprovalRequestFromCanarySatisfactionAuthority(second) error = %v", err)
	}
	second, changed, err := CreateHotUpdateOwnerApprovalDecisionFromRequest(root, secondRequest.OwnerApprovalRequestID, HotUpdateOwnerApprovalDecisionRejected, "operator", now.Add(time.Hour+24*time.Minute), "rejected")
	if err != nil {
		t.Fatalf("CreateHotUpdateOwnerApprovalDecisionFromRequest(second) error = %v", err)
	}
	if !changed {
		t.Fatal("changed = false, want true for second decision")
	}
	records, err := ListHotUpdateOwnerApprovalDecisionRecords(root)
	if err != nil {
		t.Fatalf("ListHotUpdateOwnerApprovalDecisionRecords() error = %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("ListHotUpdateOwnerApprovalDecisionRecords() len = %d, want 2", len(records))
	}
	relisted, err := ListHotUpdateOwnerApprovalDecisionRecords(root)
	if err != nil {
		t.Fatalf("ListHotUpdateOwnerApprovalDecisionRecords(relisted) error = %v", err)
	}
	if len(relisted) != len(records) {
		t.Fatalf("ListHotUpdateOwnerApprovalDecisionRecords(relisted) len = %d, want %d", len(relisted), len(records))
	}
	for i := range records {
		if relisted[i].OwnerApprovalDecisionID != records[i].OwnerApprovalDecisionID {
			t.Fatalf("ListHotUpdateOwnerApprovalDecisionRecords() order changed from %#v to %#v", records, relisted)
		}
	}
	var foundFirst, foundSecond bool
	for _, listed := range records {
		if listed.OwnerApprovalDecisionID == record.OwnerApprovalDecisionID {
			foundFirst = true
		}
		if listed.OwnerApprovalDecisionID == second.OwnerApprovalDecisionID {
			foundSecond = true
		}
	}
	if !foundFirst || !foundSecond {
		t.Fatalf("ListHotUpdateOwnerApprovalDecisionRecords() = %#v, want created decisions", records)
	}
}

func TestHotUpdateOwnerApprovalDecisionStoreDoesNotMutateSourceOrRuntimeSurfaces(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 29, 18, 0, 0, 0, time.UTC)
	requirement, evidence, authority := storeWaitingOwnerApprovalAuthority(t, root, now)
	request, _, err := CreateHotUpdateOwnerApprovalRequestFromCanarySatisfactionAuthority(root, authority.CanarySatisfactionAuthorityID, "operator", now.Add(23*time.Minute))
	if err != nil {
		t.Fatalf("CreateHotUpdateOwnerApprovalRequestFromCanarySatisfactionAuthority() error = %v", err)
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
		StoreHotUpdateCanarySatisfactionAuthorityPath(root, authority.CanarySatisfactionAuthorityID),
		StoreHotUpdateOwnerApprovalRequestPath(root, request.OwnerApprovalRequestID),
	} {
		bytes, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ReadFile(%s) error = %v", path, err)
		}
		snapshots[path] = bytes
	}

	if _, _, err := CreateHotUpdateOwnerApprovalDecisionFromRequest(root, request.OwnerApprovalRequestID, HotUpdateOwnerApprovalDecisionGranted, "operator", now.Add(24*time.Minute), "approved"); err != nil {
		t.Fatalf("CreateHotUpdateOwnerApprovalDecisionFromRequest() error = %v", err)
	}
	for path, before := range snapshots {
		after, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ReadFile(%s) after decision creation error = %v", path, err)
		}
		if string(after) != string(before) {
			t.Fatalf("source record %s changed after owner approval decision creation", path)
		}
	}
	assertNoHotUpdateOwnerApprovalRequestDownstreamRecords(t, root)
	for _, path := range []string{
		StoreActiveRuntimePackPointerPath(root),
		StoreLastKnownGoodRuntimePackPointerPath(root),
	} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("path %s exists or errored after decision creation: %v", path, err)
		}
	}
}

func validHotUpdateOwnerApprovalDecisionRecord(decidedAt time.Time, mutate func(*HotUpdateOwnerApprovalDecisionRecord)) HotUpdateOwnerApprovalDecisionRecord {
	request := validHotUpdateOwnerApprovalRequestRecord(decidedAt.Add(-time.Minute), nil)
	record := HotUpdateOwnerApprovalDecisionRecord{
		RecordVersion:                 StoreRecordVersion,
		OwnerApprovalDecisionID:       HotUpdateOwnerApprovalDecisionIDFromRequest(request.OwnerApprovalRequestID),
		OwnerApprovalRequestID:        request.OwnerApprovalRequestID,
		CanarySatisfactionAuthorityID: request.CanarySatisfactionAuthorityID,
		CanaryRequirementID:           request.CanaryRequirementID,
		SelectedCanaryEvidenceID:      request.SelectedCanaryEvidenceID,
		ResultID:                      request.ResultID,
		RunID:                         request.RunID,
		CandidateID:                   request.CandidateID,
		EvalSuiteID:                   request.EvalSuiteID,
		PromotionPolicyID:             request.PromotionPolicyID,
		BaselinePackID:                request.BaselinePackID,
		CandidatePackID:               request.CandidatePackID,
		RequestState:                  request.State,
		AuthorityState:                request.AuthorityState,
		SatisfactionState:             request.SatisfactionState,
		OwnerApprovalRequired:         request.OwnerApprovalRequired,
		Decision:                      HotUpdateOwnerApprovalDecisionGranted,
		Reason:                        "approved",
		DecidedAt:                     decidedAt.UTC(),
		DecidedBy:                     "operator",
	}
	if mutate != nil {
		mutate(&record)
	}
	return record
}
