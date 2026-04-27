package agent

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/local/picobot/internal/missioncontrol"
)

func TestProcessDirectCanaryRunbookNoOwnerBranchThroughPromotion(t *testing.T) {
	t.Parallel()

	root, resultID := writeLoopHotUpdateCanaryRequirementFixtures(t, func(record *missioncontrol.PromotionPolicyRecord) {
		record.RequiresCanary = true
	}, nil)
	ag := newLoopHotUpdateOutcomeAgent(t, root)

	observedAtRaw := "2026-04-26T04:30:00.123456789Z"
	observedAt, err := time.Parse(time.RFC3339Nano, observedAtRaw)
	if err != nil {
		t.Fatalf("Parse(observedAt) error = %v", err)
	}
	requirementID := missioncontrol.HotUpdateCanaryRequirementIDFromResult(resultID)
	evidenceID := missioncontrol.HotUpdateCanaryEvidenceIDFromRequirementObservedAt(requirementID, observedAt)
	authorityID := missioncontrol.HotUpdateCanarySatisfactionAuthorityIDFromRequirementEvidence(requirementID, evidenceID)
	hotUpdateID := missioncontrol.HotUpdateGateIDFromCanarySatisfactionAuthority(authorityID)
	outcomeID := "hot-update-outcome-" + hotUpdateID
	promotionID := "hot-update-promotion-" + hotUpdateID

	initialPointer := loadCanaryRunbookActivePointer(t, root)
	initialPointerBytes := mustLoopReadFile(t, missioncontrol.StoreActiveRuntimePackPointerPath(root))
	initialLKGBytes, initialLKGFound := readLoopOptionalFile(t, missioncontrol.StoreLastKnownGoodRuntimePackPointerPath(root))

	runCanaryRunbookCommand(t, ag, "HOT_UPDATE_CANARY_REQUIREMENT_CREATE job-1 "+resultID)
	assertCanaryRunbookPointerBytes(t, root, initialPointerBytes)
	assertCanaryRunbookLKG(t, root, initialLKGBytes, initialLKGFound)

	runCanaryRunbookCommand(t, ag, "HOT_UPDATE_CANARY_EVIDENCE_CREATE job-1 "+requirementID+" passed "+observedAtRaw+" canary passed")
	assertCanaryRunbookPointerBytes(t, root, initialPointerBytes)

	runCanaryRunbookCommand(t, ag, "HOT_UPDATE_CANARY_SATISFACTION_AUTHORITY_CREATE job-1 "+requirementID)
	assertCanaryRunbookPointerBytes(t, root, initialPointerBytes)

	runCanaryRunbookCommand(t, ag, "HOT_UPDATE_CANARY_GATE_CREATE job-1 "+authorityID)
	assertCanaryRunbookPointerBytes(t, root, initialPointerBytes)
	assertCanaryRunbookGateStatus(t, ag, hotUpdateID, authorityID, "")

	runCanaryRunbookCommand(t, ag, "HOT_UPDATE_GATE_PHASE job-1 "+hotUpdateID+" validated")
	assertCanaryRunbookPointerBytes(t, root, initialPointerBytes)
	runCanaryRunbookCommand(t, ag, "HOT_UPDATE_GATE_PHASE job-1 "+hotUpdateID+" staged")
	assertCanaryRunbookPointerBytes(t, root, initialPointerBytes)

	runCanaryRunbookCommand(t, ag, "HOT_UPDATE_GATE_EXECUTE job-1 "+hotUpdateID)
	executedPointer := loadCanaryRunbookActivePointer(t, root)
	if executedPointer.ActivePackID != "pack-candidate" {
		t.Fatalf("active pack after execute = %q, want pack-candidate", executedPointer.ActivePackID)
	}
	if executedPointer.ReloadGeneration != initialPointer.ReloadGeneration+1 {
		t.Fatalf("reload_generation after execute = %d, want %d", executedPointer.ReloadGeneration, initialPointer.ReloadGeneration+1)
	}
	executedPointerBytes := mustLoopReadFile(t, missioncontrol.StoreActiveRuntimePackPointerPath(root))

	runCanaryRunbookCommand(t, ag, "HOT_UPDATE_GATE_RELOAD job-1 "+hotUpdateID)
	assertCanaryRunbookPointerBytes(t, root, executedPointerBytes)
	assertCanaryRunbookLKG(t, root, initialLKGBytes, initialLKGFound)

	runCanaryRunbookCommand(t, ag, "HOT_UPDATE_OUTCOME_CREATE job-1 "+hotUpdateID)
	assertCanaryRunbookPointerBytes(t, root, executedPointerBytes)
	assertCanaryRunbookLKG(t, root, initialLKGBytes, initialLKGFound)

	runCanaryRunbookCommand(t, ag, "HOT_UPDATE_PROMOTION_CREATE job-1 "+outcomeID)
	assertCanaryRunbookPointerBytes(t, root, executedPointerBytes)
	assertCanaryRunbookLKG(t, root, initialLKGBytes, initialLKGFound)

	summary := loadCanaryRunbookStatus(t, ag)
	assertCanaryRunbookFinalCommonStatus(t, summary, requirementID, evidenceID, authorityID, hotUpdateID, outcomeID, promotionID, "")
	assertCanaryRunbookNoTerminalSideEffects(t, root)
	assertCanaryRunbookNoRuntimeApprovalRecords(t, root)

	assertCanaryRunbookPointerBytes(t, root, executedPointerBytes)
	assertCanaryRunbookLKG(t, root, initialLKGBytes, initialLKGFound)
}

func TestProcessDirectCanaryRunbookOwnerApprovedBranchThroughPromotion(t *testing.T) {
	t.Parallel()

	root, resultID := writeLoopHotUpdateCanaryRequirementFixtures(t, func(record *missioncontrol.PromotionPolicyRecord) {
		record.RequiresCanary = true
		record.RequiresOwnerApproval = true
	}, nil)
	ag := newLoopHotUpdateOutcomeAgent(t, root)

	observedAtRaw := "2026-04-26T04:45:00.123456789Z"
	observedAt, err := time.Parse(time.RFC3339Nano, observedAtRaw)
	if err != nil {
		t.Fatalf("Parse(observedAt) error = %v", err)
	}
	requirementID := missioncontrol.HotUpdateCanaryRequirementIDFromResult(resultID)
	evidenceID := missioncontrol.HotUpdateCanaryEvidenceIDFromRequirementObservedAt(requirementID, observedAt)
	authorityID := missioncontrol.HotUpdateCanarySatisfactionAuthorityIDFromRequirementEvidence(requirementID, evidenceID)
	requestID := missioncontrol.HotUpdateOwnerApprovalRequestIDFromCanarySatisfactionAuthority(authorityID)
	decisionID := missioncontrol.HotUpdateOwnerApprovalDecisionIDFromRequest(requestID)
	hotUpdateID := missioncontrol.HotUpdateGateIDFromCanarySatisfactionAuthority(authorityID)
	outcomeID := "hot-update-outcome-" + hotUpdateID
	promotionID := "hot-update-promotion-" + hotUpdateID

	initialPointer := loadCanaryRunbookActivePointer(t, root)
	initialPointerBytes := mustLoopReadFile(t, missioncontrol.StoreActiveRuntimePackPointerPath(root))
	initialLKGBytes, initialLKGFound := readLoopOptionalFile(t, missioncontrol.StoreLastKnownGoodRuntimePackPointerPath(root))

	runCanaryRunbookCommand(t, ag, "HOT_UPDATE_CANARY_REQUIREMENT_CREATE job-1 "+resultID)
	assertCanaryRunbookPointerBytes(t, root, initialPointerBytes)

	runCanaryRunbookCommand(t, ag, "HOT_UPDATE_CANARY_EVIDENCE_CREATE job-1 "+requirementID+" passed "+observedAtRaw+" canary passed owner approval required")
	assertCanaryRunbookPointerBytes(t, root, initialPointerBytes)

	runCanaryRunbookCommand(t, ag, "HOT_UPDATE_CANARY_SATISFACTION_AUTHORITY_CREATE job-1 "+requirementID)
	assertCanaryRunbookPointerBytes(t, root, initialPointerBytes)

	runCanaryRunbookCommand(t, ag, "HOT_UPDATE_OWNER_APPROVAL_REQUEST_CREATE job-1 "+authorityID)
	assertCanaryRunbookPointerBytes(t, root, initialPointerBytes)

	runCanaryRunbookCommand(t, ag, "HOT_UPDATE_OWNER_APPROVAL_DECISION_CREATE job-1 "+requestID+" granted owner approved")
	assertCanaryRunbookPointerBytes(t, root, initialPointerBytes)

	runCanaryRunbookCommand(t, ag, "HOT_UPDATE_CANARY_GATE_CREATE job-1 "+authorityID+" "+decisionID)
	assertCanaryRunbookPointerBytes(t, root, initialPointerBytes)
	assertCanaryRunbookGateStatus(t, ag, hotUpdateID, authorityID, decisionID)

	runCanaryRunbookCommand(t, ag, "HOT_UPDATE_GATE_PHASE job-1 "+hotUpdateID+" validated")
	assertCanaryRunbookPointerBytes(t, root, initialPointerBytes)
	runCanaryRunbookCommand(t, ag, "HOT_UPDATE_GATE_PHASE job-1 "+hotUpdateID+" staged")
	assertCanaryRunbookPointerBytes(t, root, initialPointerBytes)

	runCanaryRunbookCommand(t, ag, "HOT_UPDATE_GATE_EXECUTE job-1 "+hotUpdateID)
	executedPointer := loadCanaryRunbookActivePointer(t, root)
	if executedPointer.ActivePackID != "pack-candidate" {
		t.Fatalf("active pack after execute = %q, want pack-candidate", executedPointer.ActivePackID)
	}
	if executedPointer.ReloadGeneration != initialPointer.ReloadGeneration+1 {
		t.Fatalf("reload_generation after execute = %d, want %d", executedPointer.ReloadGeneration, initialPointer.ReloadGeneration+1)
	}
	executedPointerBytes := mustLoopReadFile(t, missioncontrol.StoreActiveRuntimePackPointerPath(root))

	runCanaryRunbookCommand(t, ag, "HOT_UPDATE_GATE_RELOAD job-1 "+hotUpdateID)
	assertCanaryRunbookPointerBytes(t, root, executedPointerBytes)
	assertCanaryRunbookLKG(t, root, initialLKGBytes, initialLKGFound)

	runCanaryRunbookCommand(t, ag, "HOT_UPDATE_OUTCOME_CREATE job-1 "+hotUpdateID)
	assertCanaryRunbookPointerBytes(t, root, executedPointerBytes)
	assertCanaryRunbookLKG(t, root, initialLKGBytes, initialLKGFound)

	runCanaryRunbookCommand(t, ag, "HOT_UPDATE_PROMOTION_CREATE job-1 "+outcomeID)
	assertCanaryRunbookPointerBytes(t, root, executedPointerBytes)
	assertCanaryRunbookLKG(t, root, initialLKGBytes, initialLKGFound)

	summary := loadCanaryRunbookStatus(t, ag)
	assertCanaryRunbookOwnerApprovalStatus(t, summary, requestID, decisionID)
	assertCanaryRunbookFinalCommonStatus(t, summary, requirementID, evidenceID, authorityID, hotUpdateID, outcomeID, promotionID, decisionID)
	assertCanaryRunbookNoTerminalSideEffects(t, root)
	assertCanaryRunbookNoRuntimeApprovalRecords(t, root)

	assertCanaryRunbookPointerBytes(t, root, executedPointerBytes)
	assertCanaryRunbookLKG(t, root, initialLKGBytes, initialLKGFound)
}

func TestProcessDirectCanaryRunbookRejectedOwnerApprovalBlocksGate(t *testing.T) {
	t.Parallel()

	root, resultID := writeLoopHotUpdateCanaryRequirementFixtures(t, func(record *missioncontrol.PromotionPolicyRecord) {
		record.RequiresCanary = true
		record.RequiresOwnerApproval = true
	}, nil)
	ag := newLoopHotUpdateOutcomeAgent(t, root)

	observedAtRaw := "2026-04-26T05:00:00.123456789Z"
	observedAt, err := time.Parse(time.RFC3339Nano, observedAtRaw)
	if err != nil {
		t.Fatalf("Parse(observedAt) error = %v", err)
	}
	requirementID := missioncontrol.HotUpdateCanaryRequirementIDFromResult(resultID)
	evidenceID := missioncontrol.HotUpdateCanaryEvidenceIDFromRequirementObservedAt(requirementID, observedAt)
	authorityID := missioncontrol.HotUpdateCanarySatisfactionAuthorityIDFromRequirementEvidence(requirementID, evidenceID)
	requestID := missioncontrol.HotUpdateOwnerApprovalRequestIDFromCanarySatisfactionAuthority(authorityID)
	decisionID := missioncontrol.HotUpdateOwnerApprovalDecisionIDFromRequest(requestID)

	initialPointerBytes := mustLoopReadFile(t, missioncontrol.StoreActiveRuntimePackPointerPath(root))
	initialPointer := loadCanaryRunbookActivePointer(t, root)
	initialLKGBytes, initialLKGFound := readLoopOptionalFile(t, missioncontrol.StoreLastKnownGoodRuntimePackPointerPath(root))

	runCanaryRunbookCommand(t, ag, "HOT_UPDATE_CANARY_REQUIREMENT_CREATE job-1 "+resultID)
	runCanaryRunbookCommand(t, ag, "HOT_UPDATE_CANARY_EVIDENCE_CREATE job-1 "+requirementID+" passed "+observedAtRaw+" canary passed but owner rejected")
	runCanaryRunbookCommand(t, ag, "HOT_UPDATE_CANARY_SATISFACTION_AUTHORITY_CREATE job-1 "+requirementID)
	runCanaryRunbookCommand(t, ag, "HOT_UPDATE_OWNER_APPROVAL_REQUEST_CREATE job-1 "+authorityID)
	runCanaryRunbookCommand(t, ag, "HOT_UPDATE_OWNER_APPROVAL_DECISION_CREATE job-1 "+requestID+" rejected owner rejected")

	decision, err := missioncontrol.LoadHotUpdateOwnerApprovalDecisionRecord(root, decisionID)
	if err != nil {
		t.Fatalf("LoadHotUpdateOwnerApprovalDecisionRecord() error = %v", err)
	}
	if decision.Decision != missioncontrol.HotUpdateOwnerApprovalDecisionRejected {
		t.Fatalf("owner approval decision = %q, want rejected", decision.Decision)
	}

	resp, err := ag.ProcessDirect("HOT_UPDATE_CANARY_GATE_CREATE job-1 "+authorityID+" "+decisionID, 2*time.Second)
	if err == nil {
		t.Fatal("ProcessDirect(HOT_UPDATE_CANARY_GATE_CREATE rejected) error = nil, want fail-closed rejection")
	}
	if resp != "" {
		t.Fatalf("ProcessDirect(HOT_UPDATE_CANARY_GATE_CREATE rejected) response = %q, want empty", resp)
	}
	if !strings.Contains(err.Error(), "does not permit hot-update canary gate creation") {
		t.Fatalf("ProcessDirect(HOT_UPDATE_CANARY_GATE_CREATE rejected) error = %q, want rejected decision context", err)
	}

	assertCanaryRunbookPointerBytes(t, root, initialPointerBytes)
	afterPointer := loadCanaryRunbookActivePointer(t, root)
	if afterPointer.ReloadGeneration != initialPointer.ReloadGeneration {
		t.Fatalf("reload_generation after rejected owner gate = %d, want %d", afterPointer.ReloadGeneration, initialPointer.ReloadGeneration)
	}
	assertCanaryRunbookLKG(t, root, initialLKGBytes, initialLKGFound)
	assertCanaryRunbookNoGateTerminalOrRuntimeApprovalRecords(t, root)
}

func runCanaryRunbookCommand(t *testing.T, ag *AgentLoop, command string) string {
	t.Helper()

	resp, err := ag.ProcessDirect(command, 2*time.Second)
	if err != nil {
		t.Fatalf("ProcessDirect(%s) error = %v", command, err)
	}
	if strings.TrimSpace(resp) == "" {
		t.Fatalf("ProcessDirect(%s) response is empty, want success acknowledgement", command)
	}
	return resp
}

func loadCanaryRunbookStatus(t *testing.T, ag *AgentLoop) missioncontrol.OperatorStatusSummary {
	t.Helper()

	status, err := ag.ProcessDirect("STATUS job-1", 2*time.Second)
	if err != nil {
		t.Fatalf("ProcessDirect(STATUS) error = %v", err)
	}
	var summary missioncontrol.OperatorStatusSummary
	if err := json.Unmarshal([]byte(status), &summary); err != nil {
		t.Fatalf("json.Unmarshal(STATUS) error = %v\nstatus:\n%s", err, status)
	}
	return summary
}

func assertCanaryRunbookGateStatus(t *testing.T, ag *AgentLoop, hotUpdateID, canaryRef, approvalRef string) {
	t.Helper()

	summary := loadCanaryRunbookStatus(t, ag)
	if summary.HotUpdateGateIdentity == nil || summary.HotUpdateGateIdentity.State != "configured" {
		t.Fatalf("HotUpdateGateIdentity = %#v, want configured", summary.HotUpdateGateIdentity)
	}
	gate := findCanaryRunbookGateStatus(t, summary, hotUpdateID)
	if gate.CanaryRef != canaryRef || gate.ApprovalRef != approvalRef {
		t.Fatalf("gate refs = %q/%q, want %q/%q", gate.CanaryRef, gate.ApprovalRef, canaryRef, approvalRef)
	}
}

func assertCanaryRunbookFinalCommonStatus(t *testing.T, summary missioncontrol.OperatorStatusSummary, requirementID, evidenceID, authorityID, hotUpdateID, outcomeID, promotionID, approvalRef string) {
	t.Helper()

	if summary.HotUpdateCanaryRequirementIdentity == nil || summary.HotUpdateCanaryRequirementIdentity.State != "configured" || len(summary.HotUpdateCanaryRequirementIdentity.Requirements) != 1 {
		t.Fatalf("HotUpdateCanaryRequirementIdentity = %#v, want one configured requirement", summary.HotUpdateCanaryRequirementIdentity)
	}
	if summary.HotUpdateCanaryRequirementIdentity.Requirements[0].CanaryRequirementID != requirementID {
		t.Fatalf("requirement id = %q, want %q", summary.HotUpdateCanaryRequirementIdentity.Requirements[0].CanaryRequirementID, requirementID)
	}
	if summary.HotUpdateCanaryRequirementIdentity.Requirements[0].RequirementState != string(missioncontrol.HotUpdateCanaryRequirementStateRequired) {
		t.Fatalf("requirement state = %q, want required", summary.HotUpdateCanaryRequirementIdentity.Requirements[0].RequirementState)
	}

	if summary.HotUpdateCanaryEvidenceIdentity == nil || summary.HotUpdateCanaryEvidenceIdentity.State != "configured" || len(summary.HotUpdateCanaryEvidenceIdentity.Evidence) != 1 {
		t.Fatalf("HotUpdateCanaryEvidenceIdentity = %#v, want one configured evidence", summary.HotUpdateCanaryEvidenceIdentity)
	}
	if summary.HotUpdateCanaryEvidenceIdentity.Evidence[0].CanaryEvidenceID != evidenceID || summary.HotUpdateCanaryEvidenceIdentity.Evidence[0].EvidenceState != string(missioncontrol.HotUpdateCanaryEvidenceStatePassed) {
		t.Fatalf("evidence status = %#v, want id %q passed", summary.HotUpdateCanaryEvidenceIdentity.Evidence[0], evidenceID)
	}

	if summary.HotUpdateCanarySatisfactionIdentity == nil || summary.HotUpdateCanarySatisfactionIdentity.State != "configured" || len(summary.HotUpdateCanarySatisfactionIdentity.Assessments) == 0 {
		t.Fatalf("HotUpdateCanarySatisfactionIdentity = %#v, want configured assessment", summary.HotUpdateCanarySatisfactionIdentity)
	}
	expectedSatisfactionState := string(missioncontrol.HotUpdateCanarySatisfactionStateSatisfied)
	expectedAuthorityState := string(missioncontrol.HotUpdateCanarySatisfactionAuthorityStateAuthorized)
	if approvalRef != "" {
		expectedSatisfactionState = string(missioncontrol.HotUpdateCanarySatisfactionStateWaitingOwnerApproval)
		expectedAuthorityState = string(missioncontrol.HotUpdateCanarySatisfactionAuthorityStateWaitingOwnerApproval)
	}
	assessment := summary.HotUpdateCanarySatisfactionIdentity.Assessments[0]
	if assessment.SelectedCanaryEvidenceID != evidenceID || assessment.SatisfactionState != expectedSatisfactionState {
		t.Fatalf("canary satisfaction = %#v, want evidence %q state %q", assessment, evidenceID, expectedSatisfactionState)
	}

	if summary.HotUpdateCanarySatisfactionAuthorityIdentity == nil || summary.HotUpdateCanarySatisfactionAuthorityIdentity.State != "configured" || len(summary.HotUpdateCanarySatisfactionAuthorityIdentity.Authorities) != 1 {
		t.Fatalf("HotUpdateCanarySatisfactionAuthorityIdentity = %#v, want one configured authority", summary.HotUpdateCanarySatisfactionAuthorityIdentity)
	}
	if summary.HotUpdateCanarySatisfactionAuthorityIdentity.Authorities[0].CanarySatisfactionAuthorityID != authorityID {
		t.Fatalf("authority id = %q, want %q", summary.HotUpdateCanarySatisfactionAuthorityIdentity.Authorities[0].CanarySatisfactionAuthorityID, authorityID)
	}
	if summary.HotUpdateCanarySatisfactionAuthorityIdentity.Authorities[0].AuthorityState != expectedAuthorityState ||
		summary.HotUpdateCanarySatisfactionAuthorityIdentity.Authorities[0].SatisfactionState != expectedSatisfactionState {
		t.Fatalf("authority status = %#v, want authority %q satisfaction %q", summary.HotUpdateCanarySatisfactionAuthorityIdentity.Authorities[0], expectedAuthorityState, expectedSatisfactionState)
	}

	if summary.HotUpdateGateIdentity == nil || summary.HotUpdateGateIdentity.State != "configured" {
		t.Fatalf("HotUpdateGateIdentity = %#v, want configured", summary.HotUpdateGateIdentity)
	}
	gate := findCanaryRunbookGateStatus(t, summary, hotUpdateID)
	if gate.CanaryRef != authorityID || gate.ApprovalRef != approvalRef {
		t.Fatalf("gate refs = %q/%q, want %q/%q", gate.CanaryRef, gate.ApprovalRef, authorityID, approvalRef)
	}
	if gate.State != string(missioncontrol.HotUpdateGateStateReloadApplySucceeded) {
		t.Fatalf("gate state = %q, want reload_apply_succeeded", gate.State)
	}

	if summary.HotUpdateOutcomeIdentity == nil || summary.HotUpdateOutcomeIdentity.State != "configured" || len(summary.HotUpdateOutcomeIdentity.Outcomes) != 1 {
		t.Fatalf("HotUpdateOutcomeIdentity = %#v, want one configured outcome", summary.HotUpdateOutcomeIdentity)
	}
	outcome := summary.HotUpdateOutcomeIdentity.Outcomes[0]
	if outcome.OutcomeID != outcomeID || outcome.HotUpdateID != hotUpdateID {
		t.Fatalf("outcome refs = %q/%q, want %q/%q", outcome.OutcomeID, outcome.HotUpdateID, outcomeID, hotUpdateID)
	}
	if outcome.CanaryRef != authorityID || outcome.ApprovalRef != approvalRef {
		t.Fatalf("outcome lineage = %q/%q, want %q/%q", outcome.CanaryRef, outcome.ApprovalRef, authorityID, approvalRef)
	}
	if outcome.OutcomeKind != string(missioncontrol.HotUpdateOutcomeKindHotUpdated) {
		t.Fatalf("outcome kind = %q, want hot_updated", outcome.OutcomeKind)
	}

	if summary.PromotionIdentity == nil || summary.PromotionIdentity.State != "configured" || len(summary.PromotionIdentity.Promotions) != 1 {
		t.Fatalf("PromotionIdentity = %#v, want one configured promotion", summary.PromotionIdentity)
	}
	promotion := summary.PromotionIdentity.Promotions[0]
	if promotion.PromotionID != promotionID || promotion.OutcomeID != outcomeID || promotion.HotUpdateID != hotUpdateID {
		t.Fatalf("promotion refs = %#v, want promotion=%q outcome=%q hot_update=%q", promotion, promotionID, outcomeID, hotUpdateID)
	}
	if promotion.CanaryRef != authorityID || promotion.ApprovalRef != approvalRef {
		t.Fatalf("promotion lineage = %q/%q, want %q/%q", promotion.CanaryRef, promotion.ApprovalRef, authorityID, approvalRef)
	}

	if summary.V4Summary == nil {
		t.Fatal("V4Summary = nil, want compact V4 status summary")
	}
	if summary.V4Summary.State != "promoted" {
		t.Fatalf("V4Summary.State = %q, want promoted", summary.V4Summary.State)
	}
	if summary.V4Summary.SelectedHotUpdateID != hotUpdateID || summary.V4Summary.SelectedOutcomeID != outcomeID || summary.V4Summary.SelectedPromotionID != promotionID {
		t.Fatalf("V4Summary selected refs = %#v, want hot_update=%q outcome=%q promotion=%q", summary.V4Summary, hotUpdateID, outcomeID, promotionID)
	}
	if !summary.V4Summary.HasCanaryAuthority {
		t.Fatalf("V4Summary.HasCanaryAuthority = false, want true")
	}
	if summary.V4Summary.HasOwnerApprovalDecision != (approvalRef != "") {
		t.Fatalf("V4Summary.HasOwnerApprovalDecision = %t, want %t", summary.V4Summary.HasOwnerApprovalDecision, approvalRef != "")
	}
	if summary.V4Summary.HasCandidatePromotionDecision || summary.V4Summary.HasRollback || summary.V4Summary.HasRollbackApply {
		t.Fatalf("V4Summary = %#v, want canary path without candidate decision or rollback signals", summary.V4Summary)
	}
	if summary.V4Summary.InvalidIdentityCount != 0 || len(summary.V4Summary.Warnings) != 0 {
		t.Fatalf("V4Summary invalid fields = count %d warnings %#v, want clean summary", summary.V4Summary.InvalidIdentityCount, summary.V4Summary.Warnings)
	}
}

func assertCanaryRunbookOwnerApprovalStatus(t *testing.T, summary missioncontrol.OperatorStatusSummary, requestID, decisionID string) {
	t.Helper()

	if summary.HotUpdateOwnerApprovalRequestIdentity == nil || summary.HotUpdateOwnerApprovalRequestIdentity.State != "configured" || len(summary.HotUpdateOwnerApprovalRequestIdentity.Requests) != 1 {
		t.Fatalf("HotUpdateOwnerApprovalRequestIdentity = %#v, want one configured request", summary.HotUpdateOwnerApprovalRequestIdentity)
	}
	if summary.HotUpdateOwnerApprovalRequestIdentity.Requests[0].OwnerApprovalRequestID != requestID {
		t.Fatalf("owner approval request id = %q, want %q", summary.HotUpdateOwnerApprovalRequestIdentity.Requests[0].OwnerApprovalRequestID, requestID)
	}

	if summary.HotUpdateOwnerApprovalDecisionIdentity == nil || summary.HotUpdateOwnerApprovalDecisionIdentity.State != "configured" || len(summary.HotUpdateOwnerApprovalDecisionIdentity.Decisions) != 1 {
		t.Fatalf("HotUpdateOwnerApprovalDecisionIdentity = %#v, want one configured decision", summary.HotUpdateOwnerApprovalDecisionIdentity)
	}
	decision := summary.HotUpdateOwnerApprovalDecisionIdentity.Decisions[0]
	if decision.OwnerApprovalDecisionID != decisionID {
		t.Fatalf("owner approval decision id = %q, want %q", decision.OwnerApprovalDecisionID, decisionID)
	}
	if decision.Decision != string(missioncontrol.HotUpdateOwnerApprovalDecisionGranted) {
		t.Fatalf("owner approval decision = %q, want granted", decision.Decision)
	}
}

func findCanaryRunbookGateStatus(t *testing.T, summary missioncontrol.OperatorStatusSummary, hotUpdateID string) missioncontrol.OperatorHotUpdateGateStatus {
	t.Helper()

	for _, gate := range summary.HotUpdateGateIdentity.Gates {
		if gate.HotUpdateID == hotUpdateID {
			return gate
		}
	}
	t.Fatalf("hot-update gate %q not found in status: %#v", hotUpdateID, summary.HotUpdateGateIdentity.Gates)
	return missioncontrol.OperatorHotUpdateGateStatus{}
}

func loadCanaryRunbookActivePointer(t *testing.T, root string) missioncontrol.ActiveRuntimePackPointer {
	t.Helper()

	pointer, err := missioncontrol.LoadActiveRuntimePackPointer(root)
	if err != nil {
		t.Fatalf("LoadActiveRuntimePackPointer() error = %v", err)
	}
	return pointer
}

func assertCanaryRunbookPointerBytes(t *testing.T, root string, want []byte) {
	t.Helper()

	got := mustLoopReadFile(t, missioncontrol.StoreActiveRuntimePackPointerPath(root))
	if string(got) != string(want) {
		t.Fatalf("active runtime pack pointer changed unexpectedly\nwant:\n%s\ngot:\n%s", string(want), string(got))
	}
}

func assertCanaryRunbookLKG(t *testing.T, root string, wantBytes []byte, wantFound bool) {
	t.Helper()

	gotBytes, gotFound := readLoopOptionalFile(t, missioncontrol.StoreLastKnownGoodRuntimePackPointerPath(root))
	if gotFound != wantFound {
		t.Fatalf("last-known-good pointer found = %t, want %t", gotFound, wantFound)
	}
	if gotFound && string(gotBytes) != string(wantBytes) {
		t.Fatalf("last-known-good pointer changed unexpectedly\nwant:\n%s\ngot:\n%s", string(wantBytes), string(gotBytes))
	}
}

func assertCanaryRunbookNoTerminalSideEffects(t *testing.T, root string) {
	t.Helper()

	assertCanaryRunbookNoCandidatePromotionDecision(t, root)
	assertCanaryRunbookNoRollbacks(t, root)
	assertCanaryRunbookNoRollbackApplies(t, root)
}

func assertCanaryRunbookNoGateTerminalOrRuntimeApprovalRecords(t *testing.T, root string) {
	t.Helper()

	gates, err := missioncontrol.ListHotUpdateGateRecords(root)
	if err != nil && !os.IsNotExist(err) {
		t.Fatalf("ListHotUpdateGateRecords() error = %v", err)
	}
	if len(gates) != 0 {
		t.Fatalf("ListHotUpdateGateRecords() len = %d, want 0", len(gates))
	}
	outcomes, err := missioncontrol.ListHotUpdateOutcomeRecords(root)
	if err != nil && !os.IsNotExist(err) {
		t.Fatalf("ListHotUpdateOutcomeRecords() error = %v", err)
	}
	if len(outcomes) != 0 {
		t.Fatalf("ListHotUpdateOutcomeRecords() len = %d, want 0", len(outcomes))
	}
	promotions, err := missioncontrol.ListPromotionRecords(root)
	if err != nil && !os.IsNotExist(err) {
		t.Fatalf("ListPromotionRecords() error = %v", err)
	}
	if len(promotions) != 0 {
		t.Fatalf("ListPromotionRecords() len = %d, want 0", len(promotions))
	}
	assertCanaryRunbookNoTerminalSideEffects(t, root)
	assertCanaryRunbookNoRuntimeApprovalRecords(t, root)
}

func assertCanaryRunbookNoRuntimeApprovalRecords(t *testing.T, root string) {
	t.Helper()

	requests, err := missioncontrol.ListCommittedApprovalRequestRecords(root, "job-1")
	if err != nil && !os.IsNotExist(err) {
		t.Fatalf("ListCommittedApprovalRequestRecords() error = %v", err)
	}
	if len(requests) != 0 {
		t.Fatalf("ListCommittedApprovalRequestRecords() len = %d, want 0", len(requests))
	}
	grants, err := missioncontrol.ListCommittedApprovalGrantRecords(root, "job-1")
	if err != nil && !os.IsNotExist(err) {
		t.Fatalf("ListCommittedApprovalGrantRecords() error = %v", err)
	}
	if len(grants) != 0 {
		t.Fatalf("ListCommittedApprovalGrantRecords() len = %d, want 0", len(grants))
	}
}

func assertCanaryRunbookNoCandidatePromotionDecision(t *testing.T, root string) {
	t.Helper()

	decisions, err := missioncontrol.ListCandidatePromotionDecisionRecords(root)
	if err != nil && !os.IsNotExist(err) {
		t.Fatalf("ListCandidatePromotionDecisionRecords() error = %v", err)
	}
	if len(decisions) != 0 {
		t.Fatalf("ListCandidatePromotionDecisionRecords() len = %d, want 0", len(decisions))
	}
}

func assertCanaryRunbookNoRollbacks(t *testing.T, root string) {
	t.Helper()

	rollbacks, err := missioncontrol.ListRollbackRecords(root)
	if err != nil && !os.IsNotExist(err) {
		t.Fatalf("ListRollbackRecords() error = %v", err)
	}
	if len(rollbacks) != 0 {
		t.Fatalf("ListRollbackRecords() len = %d, want 0", len(rollbacks))
	}
}

func assertCanaryRunbookNoRollbackApplies(t *testing.T, root string) {
	t.Helper()

	applies, err := missioncontrol.ListRollbackApplyRecords(root)
	if err != nil && !os.IsNotExist(err) {
		t.Fatalf("ListRollbackApplyRecords() error = %v", err)
	}
	if len(applies) != 0 {
		t.Fatalf("ListRollbackApplyRecords() len = %d, want 0", len(applies))
	}
}
