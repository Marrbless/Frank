package agent

import (
	"os"
	"testing"
	"time"

	"github.com/local/picobot/internal/missioncontrol"
)

func TestProcessDirectEligibleRunbookFromDecisionThroughLastKnownGood(t *testing.T) {
	t.Parallel()

	root, decision := writeLoopCandidatePromotionDecisionGateFixtures(t, true)
	now := time.Date(2026, 4, 26, 6, 0, 0, 0, time.UTC)
	writeLoopActiveJobEvidence(t, root, now, "job-1", missioncontrol.JobStateRunning, missioncontrol.ExecutionPlaneLiveRuntime, missioncontrol.MissionFamilyBootstrapRevenue)
	ag := newLoopHotUpdateOutcomeAgent(t, root)

	hotUpdateID := "hot-update-" + decision.PromotionDecisionID
	outcomeID := "hot-update-outcome-" + hotUpdateID
	promotionID := "hot-update-promotion-" + hotUpdateID

	initialPointer := loadCanaryRunbookActivePointer(t, root)
	initialPointerBytes := mustLoopReadFile(t, missioncontrol.StoreActiveRuntimePackPointerPath(root))
	initialLKGBytes := mustLoopReadFile(t, missioncontrol.StoreLastKnownGoodRuntimePackPointerPath(root))

	runCanaryRunbookCommand(t, ag, "HOT_UPDATE_GATE_FROM_DECISION job-1 "+decision.PromotionDecisionID)
	assertCanaryRunbookPointerBytes(t, root, initialPointerBytes)
	assertEligibleRunbookLKGBytes(t, root, initialLKGBytes)

	runCanaryRunbookCommand(t, ag, "HOT_UPDATE_GATE_PHASE job-1 "+hotUpdateID+" validated")
	assertCanaryRunbookPointerBytes(t, root, initialPointerBytes)
	assertEligibleRunbookLKGBytes(t, root, initialLKGBytes)

	runCanaryRunbookCommand(t, ag, "HOT_UPDATE_GATE_PHASE job-1 "+hotUpdateID+" staged")
	assertCanaryRunbookPointerBytes(t, root, initialPointerBytes)
	assertEligibleRunbookLKGBytes(t, root, initialLKGBytes)

	runCanaryRunbookCommand(t, ag, "HOT_UPDATE_EXECUTION_READY job-1 "+hotUpdateID+" 300 operator checked quiesce")
	assertCanaryRunbookPointerBytes(t, root, initialPointerBytes)
	assertEligibleRunbookLKGBytes(t, root, initialLKGBytes)

	runCanaryRunbookCommand(t, ag, "HOT_UPDATE_GATE_EXECUTE job-1 "+hotUpdateID)
	executedPointer := loadCanaryRunbookActivePointer(t, root)
	if executedPointer.ActivePackID != decision.CandidatePackID {
		t.Fatalf("active pack after execute = %q, want %q", executedPointer.ActivePackID, decision.CandidatePackID)
	}
	if executedPointer.ReloadGeneration != initialPointer.ReloadGeneration+1 {
		t.Fatalf("reload_generation after execute = %d, want %d", executedPointer.ReloadGeneration, initialPointer.ReloadGeneration+1)
	}
	executedPointerBytes := mustLoopReadFile(t, missioncontrol.StoreActiveRuntimePackPointerPath(root))
	assertEligibleRunbookLKGBytes(t, root, initialLKGBytes)

	recordLoopHotUpdateSmokeCheck(t, root, hotUpdateID, now.Add(7*time.Minute))
	runCanaryRunbookCommand(t, ag, "HOT_UPDATE_GATE_RELOAD job-1 "+hotUpdateID)
	assertCanaryRunbookPointerBytes(t, root, executedPointerBytes)
	assertEligibleRunbookLKGBytes(t, root, initialLKGBytes)

	runCanaryRunbookCommand(t, ag, "HOT_UPDATE_OUTCOME_CREATE job-1 "+hotUpdateID)
	assertCanaryRunbookPointerBytes(t, root, executedPointerBytes)
	assertEligibleRunbookLKGBytes(t, root, initialLKGBytes)

	runCanaryRunbookCommand(t, ag, "HOT_UPDATE_PROMOTION_CREATE job-1 "+outcomeID)
	assertCanaryRunbookPointerBytes(t, root, executedPointerBytes)
	assertEligibleRunbookLKGBytes(t, root, initialLKGBytes)

	runCanaryRunbookCommand(t, ag, "HOT_UPDATE_LKG_RECERTIFY job-1 "+promotionID)
	assertCanaryRunbookPointerBytes(t, root, executedPointerBytes)
	recertifiedLKG := loadEligibleRunbookLastKnownGood(t, root)
	if recertifiedLKG.PackID != decision.CandidatePackID {
		t.Fatalf("last-known-good pack after recertify = %q, want %q", recertifiedLKG.PackID, decision.CandidatePackID)
	}
	if recertifiedLKG.Basis != "hot_update_promotion:"+promotionID {
		t.Fatalf("last-known-good basis = %q, want hot-update promotion basis", recertifiedLKG.Basis)
	}

	afterPointer := loadCanaryRunbookActivePointer(t, root)
	if afterPointer.ReloadGeneration != executedPointer.ReloadGeneration {
		t.Fatalf("reload_generation after reload/outcome/promotion/LKG = %d, want %d", afterPointer.ReloadGeneration, executedPointer.ReloadGeneration)
	}

	summary := loadCanaryRunbookStatus(t, ag)
	assertEligibleRunbookCandidatePromotionDecisionLedger(t, root, decision)
	assertEligibleRunbookFinalStatus(t, summary, decision, hotUpdateID, outcomeID, promotionID)
	assertEligibleRunbookNoCanaryAuthorityRecords(t, root)
	assertEligibleRunbookNoRollbackRecords(t, root)
	assertCanaryRunbookNoRuntimeApprovalRecords(t, root)
}

func assertEligibleRunbookFinalStatus(t *testing.T, summary missioncontrol.OperatorStatusSummary, decision missioncontrol.CandidatePromotionDecisionRecord, hotUpdateID, outcomeID, promotionID string) {
	t.Helper()

	if summary.HotUpdateGateIdentity == nil || summary.HotUpdateGateIdentity.State != "configured" || len(summary.HotUpdateGateIdentity.Gates) != 1 {
		t.Fatalf("HotUpdateGateIdentity = %#v, want one configured gate", summary.HotUpdateGateIdentity)
	}
	gate := summary.HotUpdateGateIdentity.Gates[0]
	if gate.HotUpdateID != hotUpdateID || gate.CandidatePackID != decision.CandidatePackID || gate.PreviousActivePackID != decision.BaselinePackID {
		t.Fatalf("gate status = %#v, want hot_update=%q candidate=%q previous=%q", gate, hotUpdateID, decision.CandidatePackID, decision.BaselinePackID)
	}
	if gate.CanaryRef != "" || gate.ApprovalRef != "" {
		t.Fatalf("eligible-only gate lineage = %q/%q, want empty canary/approval refs", gate.CanaryRef, gate.ApprovalRef)
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
	if outcome.CanaryRef != "" || outcome.ApprovalRef != "" {
		t.Fatalf("eligible-only outcome lineage = %q/%q, want empty canary/approval refs", outcome.CanaryRef, outcome.ApprovalRef)
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
	if promotion.CanaryRef != "" || promotion.ApprovalRef != "" {
		t.Fatalf("eligible-only promotion lineage = %q/%q, want empty canary/approval refs", promotion.CanaryRef, promotion.ApprovalRef)
	}

	if summary.RuntimePackIdentity == nil {
		t.Fatal("RuntimePackIdentity = nil, want active and last-known-good status")
	}
	if summary.RuntimePackIdentity.Active.ActivePackID != decision.CandidatePackID {
		t.Fatalf("active pack status = %q, want %q", summary.RuntimePackIdentity.Active.ActivePackID, decision.CandidatePackID)
	}
	if summary.RuntimePackIdentity.LastKnownGood.PackID != decision.CandidatePackID {
		t.Fatalf("last-known-good pack status = %q, want %q", summary.RuntimePackIdentity.LastKnownGood.PackID, decision.CandidatePackID)
	}
	if summary.RuntimePackIdentity.LastKnownGood.Basis != "hot_update_promotion:"+promotionID {
		t.Fatalf("last-known-good basis status = %q, want hot-update promotion basis", summary.RuntimePackIdentity.LastKnownGood.Basis)
	}

	if summary.V4Summary == nil {
		t.Fatal("V4Summary = nil, want compact V4 status summary")
	}
	if summary.V4Summary.State != "last_known_good_recertified" {
		t.Fatalf("V4Summary.State = %q, want last_known_good_recertified", summary.V4Summary.State)
	}
	if summary.V4Summary.ActivePackID != decision.CandidatePackID || summary.V4Summary.LastKnownGoodPackID != decision.CandidatePackID {
		t.Fatalf("V4Summary packs = active %q lkg %q, want %q", summary.V4Summary.ActivePackID, summary.V4Summary.LastKnownGoodPackID, decision.CandidatePackID)
	}
	if summary.V4Summary.SelectedHotUpdateID != hotUpdateID || summary.V4Summary.SelectedOutcomeID != outcomeID || summary.V4Summary.SelectedPromotionID != promotionID {
		t.Fatalf("V4Summary selected refs = %#v, want hot_update=%q outcome=%q promotion=%q", summary.V4Summary, hotUpdateID, outcomeID, promotionID)
	}
	if !summary.V4Summary.HasCandidatePromotionDecision {
		t.Fatalf("V4Summary.HasCandidatePromotionDecision = false, want true")
	}
	if summary.V4Summary.HasCanaryAuthority || summary.V4Summary.HasOwnerApprovalDecision || summary.V4Summary.HasRollback || summary.V4Summary.HasRollbackApply {
		t.Fatalf("V4Summary = %#v, want eligible-only path without canary/approval/rollback signals", summary.V4Summary)
	}
	if summary.V4Summary.InvalidIdentityCount != 0 || len(summary.V4Summary.Warnings) != 0 {
		t.Fatalf("V4Summary invalid fields = count %d warnings %#v, want clean summary", summary.V4Summary.InvalidIdentityCount, summary.V4Summary.Warnings)
	}
}

func assertEligibleRunbookCandidatePromotionDecisionLedger(t *testing.T, root string, decision missioncontrol.CandidatePromotionDecisionRecord) {
	t.Helper()

	decisions, err := missioncontrol.ListCandidatePromotionDecisionRecords(root)
	if err != nil {
		t.Fatalf("ListCandidatePromotionDecisionRecords() error = %v", err)
	}
	if len(decisions) != 1 {
		t.Fatalf("ListCandidatePromotionDecisionRecords() len = %d, want 1", len(decisions))
	}
	if decisions[0].PromotionDecisionID != decision.PromotionDecisionID || decisions[0].ResultID != decision.ResultID {
		t.Fatalf("candidate promotion decision ledger = %#v, want decision=%q result=%q", decisions[0], decision.PromotionDecisionID, decision.ResultID)
	}
	if decisions[0].EligibilityState != missioncontrol.CandidatePromotionEligibilityStateEligible {
		t.Fatalf("candidate promotion decision eligibility = %q, want eligible", decisions[0].EligibilityState)
	}
}

func assertEligibleRunbookLKGBytes(t *testing.T, root string, want []byte) {
	t.Helper()

	got := mustLoopReadFile(t, missioncontrol.StoreLastKnownGoodRuntimePackPointerPath(root))
	if string(got) != string(want) {
		t.Fatalf("last-known-good pointer changed unexpectedly\nwant:\n%s\ngot:\n%s", string(want), string(got))
	}
}

func loadEligibleRunbookLastKnownGood(t *testing.T, root string) missioncontrol.LastKnownGoodRuntimePackPointer {
	t.Helper()

	pointer, err := missioncontrol.LoadLastKnownGoodRuntimePackPointer(root)
	if err != nil {
		t.Fatalf("LoadLastKnownGoodRuntimePackPointer() error = %v", err)
	}
	return pointer
}

func assertEligibleRunbookNoRollbackRecords(t *testing.T, root string) {
	t.Helper()

	rollbacks, err := missioncontrol.ListRollbackRecords(root)
	if err != nil && !os.IsNotExist(err) {
		t.Fatalf("ListRollbackRecords() error = %v", err)
	}
	if len(rollbacks) != 0 {
		t.Fatalf("ListRollbackRecords() len = %d, want 0", len(rollbacks))
	}
	applies, err := missioncontrol.ListRollbackApplyRecords(root)
	if err != nil && !os.IsNotExist(err) {
		t.Fatalf("ListRollbackApplyRecords() error = %v", err)
	}
	if len(applies) != 0 {
		t.Fatalf("ListRollbackApplyRecords() len = %d, want 0", len(applies))
	}
}

func assertEligibleRunbookNoCanaryAuthorityRecords(t *testing.T, root string) {
	t.Helper()

	requirements, err := missioncontrol.ListHotUpdateCanaryRequirementRecords(root)
	if err != nil && !os.IsNotExist(err) {
		t.Fatalf("ListHotUpdateCanaryRequirementRecords() error = %v", err)
	}
	if len(requirements) != 0 {
		t.Fatalf("ListHotUpdateCanaryRequirementRecords() len = %d, want 0", len(requirements))
	}
	evidence, err := missioncontrol.ListHotUpdateCanaryEvidenceRecords(root)
	if err != nil && !os.IsNotExist(err) {
		t.Fatalf("ListHotUpdateCanaryEvidenceRecords() error = %v", err)
	}
	if len(evidence) != 0 {
		t.Fatalf("ListHotUpdateCanaryEvidenceRecords() len = %d, want 0", len(evidence))
	}
	authorities, err := missioncontrol.ListHotUpdateCanarySatisfactionAuthorityRecords(root)
	if err != nil && !os.IsNotExist(err) {
		t.Fatalf("ListHotUpdateCanarySatisfactionAuthorityRecords() error = %v", err)
	}
	if len(authorities) != 0 {
		t.Fatalf("ListHotUpdateCanarySatisfactionAuthorityRecords() len = %d, want 0", len(authorities))
	}
	requests, err := missioncontrol.ListHotUpdateOwnerApprovalRequestRecords(root)
	if err != nil && !os.IsNotExist(err) {
		t.Fatalf("ListHotUpdateOwnerApprovalRequestRecords() error = %v", err)
	}
	if len(requests) != 0 {
		t.Fatalf("ListHotUpdateOwnerApprovalRequestRecords() len = %d, want 0", len(requests))
	}
	decisions, err := missioncontrol.ListHotUpdateOwnerApprovalDecisionRecords(root)
	if err != nil && !os.IsNotExist(err) {
		t.Fatalf("ListHotUpdateOwnerApprovalDecisionRecords() error = %v", err)
	}
	if len(decisions) != 0 {
		t.Fatalf("ListHotUpdateOwnerApprovalDecisionRecords() len = %d, want 0", len(decisions))
	}
}
