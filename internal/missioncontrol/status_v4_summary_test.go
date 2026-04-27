package missioncontrol

import "testing"

func TestOperatorV4SummaryEmptyStoreIsNotConfigured(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	summary := BuildOperatorStatusSummary(JobRuntimeState{
		JobID: "job-1",
		State: JobStateRunning,
	})
	summary = withOperatorV4IdentitySurfaces(summary, root)
	summary = WithV4Summary(summary)

	if summary.V4Summary == nil {
		t.Fatal("V4Summary = nil, want not_configured summary")
	}
	if summary.V4Summary.State != "not_configured" {
		t.Fatalf("V4Summary.State = %q, want not_configured", summary.V4Summary.State)
	}
	if summary.V4Summary.InvalidIdentityCount != 0 {
		t.Fatalf("V4Summary.InvalidIdentityCount = %d, want 0", summary.V4Summary.InvalidIdentityCount)
	}
	if len(summary.V4Summary.Warnings) != 0 {
		t.Fatalf("V4Summary.Warnings = %#v, want empty", summary.V4Summary.Warnings)
	}
}

func TestOperatorV4SummarySurfacesInvalidIdentityWithoutHidingDetails(t *testing.T) {
	t.Parallel()

	summary := BuildOperatorStatusSummary(JobRuntimeState{
		JobID: "job-1",
		State: JobStateRunning,
	})
	summary.HotUpdateGateIdentity = &OperatorHotUpdateGateIdentityStatus{
		State: "invalid",
		Gates: []OperatorHotUpdateGateStatus{{
			HotUpdateID: "hot-update-bad",
			State:       "invalid",
			Error:       "bad gate",
		}},
	}
	summary = WithV4Summary(summary)

	if summary.V4Summary == nil {
		t.Fatal("V4Summary = nil, want invalid summary")
	}
	if summary.V4Summary.State != "invalid" {
		t.Fatalf("V4Summary.State = %q, want invalid", summary.V4Summary.State)
	}
	if summary.V4Summary.InvalidIdentityCount != 1 {
		t.Fatalf("V4Summary.InvalidIdentityCount = %d, want 1", summary.V4Summary.InvalidIdentityCount)
	}
	if len(summary.V4Summary.Warnings) != 1 || summary.V4Summary.Warnings[0] != "hot_update_gate_identity invalid" {
		t.Fatalf("V4Summary.Warnings = %#v, want hot-update gate warning", summary.V4Summary.Warnings)
	}
	if summary.V4Summary.SelectedHotUpdateID != "hot-update-bad" {
		t.Fatalf("V4Summary.SelectedHotUpdateID = %q, want invalid hot-update id", summary.V4Summary.SelectedHotUpdateID)
	}
	if summary.HotUpdateGateIdentity == nil || len(summary.HotUpdateGateIdentity.Gates) != 1 || summary.HotUpdateGateIdentity.Gates[0].Error != "bad gate" {
		t.Fatalf("HotUpdateGateIdentity details = %#v, want invalid detail preserved", summary.HotUpdateGateIdentity)
	}
}

func withOperatorV4IdentitySurfaces(summary OperatorStatusSummary, root string) OperatorStatusSummary {
	summary = WithRuntimePackIdentity(summary, root)
	summary = WithImprovementCandidateIdentity(summary, root)
	summary = WithEvalSuiteIdentity(summary, root)
	summary = WithPromotionPolicyIdentity(summary, root)
	summary = WithImprovementRunIdentity(summary, root)
	summary = WithCandidateResultIdentity(summary, root)
	summary = WithHotUpdateCanaryRequirementIdentity(summary, root)
	summary = WithHotUpdateCanaryEvidenceIdentity(summary, root)
	summary = WithHotUpdateCanarySatisfactionIdentity(summary, root)
	summary = WithHotUpdateCanarySatisfactionAuthorityIdentity(summary, root)
	summary = WithHotUpdateOwnerApprovalRequestIdentity(summary, root)
	summary = WithHotUpdateOwnerApprovalDecisionIdentity(summary, root)
	summary = WithCandidatePromotionDecisionIdentity(summary, root)
	summary = WithHotUpdateGateIdentity(summary, root)
	summary = WithHotUpdateOutcomeIdentity(summary, root)
	summary = WithPromotionIdentity(summary, root)
	summary = WithRollbackIdentity(summary, root)
	summary = WithRollbackApplyIdentity(summary, root)
	return summary
}
