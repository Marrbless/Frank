package missioncontrol

import (
	"testing"
	"time"
)

func TestDeriveCampaignZohoEmailSendGateDecisionAllowsBelowVerifiedSendStopLimit(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 15, 18, 0, 0, 0, time.UTC)
	campaign := validCampaignRecord(now, func(record *CampaignRecord) {
		record.CampaignID = "campaign-zoho"
		record.StopConditions = []string{"stop after 3 verified sends"}
		record.FailureThreshold = CampaignFailureThreshold{Metric: "rejections", Limit: 3}
	})

	records := []CampaignZohoEmailOutboundActionRecord{
		testCampaignZohoEmailOutboundActionRecord("job-1", 1, mustBuildVerifiedCampaignZohoEmailOutboundAction(t, "step-1", "campaign-zoho", "subject-1", now)),
		testCampaignZohoEmailOutboundActionRecord("job-1", 1, mustBuildVerifiedCampaignZohoEmailOutboundAction(t, "step-2", "campaign-zoho", "subject-2", now.Add(time.Minute))),
	}

	decision, err := DeriveCampaignZohoEmailSendGateDecision(campaign, records)
	if err != nil {
		t.Fatalf("DeriveCampaignZohoEmailSendGateDecision() error = %v", err)
	}
	if !decision.Allowed {
		t.Fatalf("Decision.Allowed = false, want true with two verified sends below limit")
	}
	if decision.Halted {
		t.Fatal("Decision.Halted = true, want false below limit")
	}
	if decision.VerifiedSuccessCount != 2 {
		t.Fatalf("Decision.VerifiedSuccessCount = %d, want 2", decision.VerifiedSuccessCount)
	}
	if decision.AmbiguousOutcomeCount != 0 {
		t.Fatalf("Decision.AmbiguousOutcomeCount = %d, want 0", decision.AmbiguousOutcomeCount)
	}
	if decision.FailureCount != 0 {
		t.Fatalf("Decision.FailureCount = %d, want 0", decision.FailureCount)
	}
}

func TestDeriveCampaignZohoEmailSendGateDecisionHaltsAtVerifiedSendStopLimit(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 15, 18, 30, 0, 0, time.UTC)
	campaign := validCampaignRecord(now, func(record *CampaignRecord) {
		record.CampaignID = "campaign-zoho"
		record.StopConditions = []string{"stop after 2 verified sends"}
		record.FailureThreshold = CampaignFailureThreshold{Metric: "rejections", Limit: 3}
	})

	records := []CampaignZohoEmailOutboundActionRecord{
		testCampaignZohoEmailOutboundActionRecord("job-1", 1, mustBuildVerifiedCampaignZohoEmailOutboundAction(t, "step-1", "campaign-zoho", "subject-1", now)),
		testCampaignZohoEmailOutboundActionRecord("job-1", 1, mustBuildVerifiedCampaignZohoEmailOutboundAction(t, "step-2", "campaign-zoho", "subject-2", now.Add(time.Minute))),
		testCampaignZohoEmailOutboundActionRecord("job-1", 1, mustBuildPreparedCampaignZohoEmailOutboundAction(t, "step-3", "campaign-zoho", "subject-3", now.Add(2*time.Minute))),
	}

	decision, err := DeriveCampaignZohoEmailSendGateDecision(campaign, records)
	if err != nil {
		t.Fatalf("DeriveCampaignZohoEmailSendGateDecision() error = %v", err)
	}
	if decision.Allowed {
		t.Fatal("Decision.Allowed = true, want false once verified-send stop limit is reached")
	}
	if !decision.Halted {
		t.Fatal("Decision.Halted = false, want true once verified-send stop limit is reached")
	}
	if decision.TriggeredStopCondition != "stop after 2 verified sends" {
		t.Fatalf("Decision.TriggeredStopCondition = %q, want configured verified-send stop condition", decision.TriggeredStopCondition)
	}
	if decision.VerifiedSuccessCount != 2 {
		t.Fatalf("Decision.VerifiedSuccessCount = %d, want 2", decision.VerifiedSuccessCount)
	}
	if decision.AmbiguousOutcomeCount != 1 {
		t.Fatalf("Decision.AmbiguousOutcomeCount = %d, want 1", decision.AmbiguousOutcomeCount)
	}
}

func TestDeriveCampaignZohoEmailSendGateDecisionFailsClosedOnUnsupportedStopCondition(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 15, 19, 0, 0, 0, time.UTC)
	campaign := validCampaignRecord(now, func(record *CampaignRecord) {
		record.CampaignID = "campaign-zoho"
		record.StopConditions = []string{"stop after 3 replies"}
		record.FailureThreshold = CampaignFailureThreshold{Metric: "rejections", Limit: 3}
	})

	_, err := DeriveCampaignZohoEmailSendGateDecision(campaign, nil)
	if err == nil {
		t.Fatal("DeriveCampaignZohoEmailSendGateDecision() error = nil, want unsupported stop-condition rejection")
	}
	if got := err.Error(); got != `campaign zoho email stop_condition "stop after 3 replies" is not evaluable from committed outbound action records` {
		t.Fatalf("DeriveCampaignZohoEmailSendGateDecision() error = %q, want unsupported stop-condition rejection", got)
	}
}

func TestDeriveCampaignZohoEmailSendGateDecisionHaltsAtFailureThreshold(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 15, 19, 15, 0, 0, time.UTC)
	campaign := validCampaignRecord(now, func(record *CampaignRecord) {
		record.CampaignID = "campaign-zoho"
		record.StopConditions = []string{"stop after 5 verified sends"}
		record.FailureThreshold = CampaignFailureThreshold{Metric: "rejections", Limit: 2}
	})

	records := []CampaignZohoEmailOutboundActionRecord{
		testCampaignZohoEmailOutboundActionRecord("job-1", 1, mustBuildFailedCampaignZohoEmailOutboundAction(t, "step-1", "campaign-zoho", "subject-1", now)),
		testCampaignZohoEmailOutboundActionRecord("job-2", 1, mustBuildFailedCampaignZohoEmailOutboundAction(t, "step-2", "campaign-zoho", "subject-2", now.Add(time.Minute))),
		testCampaignZohoEmailOutboundActionRecord("job-3", 1, mustBuildPreparedCampaignZohoEmailOutboundAction(t, "step-3", "campaign-zoho", "subject-3", now.Add(2*time.Minute))),
	}

	decision, err := DeriveCampaignZohoEmailSendGateDecision(campaign, records)
	if err != nil {
		t.Fatalf("DeriveCampaignZohoEmailSendGateDecision() error = %v", err)
	}
	if decision.Allowed {
		t.Fatal("Decision.Allowed = true, want false once failure threshold is reached")
	}
	if !decision.Halted {
		t.Fatal("Decision.Halted = false, want true once failure threshold is reached")
	}
	if decision.FailureCount != 2 {
		t.Fatalf("Decision.FailureCount = %d, want 2", decision.FailureCount)
	}
	if decision.AmbiguousOutcomeCount != 1 {
		t.Fatalf("Decision.AmbiguousOutcomeCount = %d, want 1", decision.AmbiguousOutcomeCount)
	}
}

func TestListCommittedCampaignZohoEmailOutboundActionRecordsByCampaignFiltersAndPrefersMostTerminalState(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 15, 19, 30, 0, 0, time.UTC)

	for _, jobID := range []string{"job-1", "job-2", "job-3"} {
		if err := StoreJobRuntimeRecord(root, JobRuntimeRecord{
			RecordVersion: StoreRecordVersion,
			WriterEpoch:   1,
			AppliedSeq:    1,
			JobID:         jobID,
			State:         JobStateRunning,
			ActiveStepID:  "send-outbound-email",
			CreatedAt:     now,
			UpdatedAt:     now,
		}); err != nil {
			t.Fatalf("StoreJobRuntimeRecord(%q) error = %v", jobID, err)
		}
	}

	prepared := mustBuildPreparedCampaignZohoEmailOutboundAction(t, "send-outbound-email", "campaign-zoho", "Frank intro", now)
	verified := mustBuildVerifiedCampaignZohoEmailOutboundAction(t, "send-outbound-email", "campaign-zoho", "Frank intro", now)
	otherCampaign := mustBuildVerifiedCampaignZohoEmailOutboundAction(t, "send-other", "campaign-other", "Other subject", now.Add(time.Minute))

	if err := StoreCampaignZohoEmailOutboundActionRecord(root, testCampaignZohoEmailOutboundActionRecord("job-1", 1, prepared)); err != nil {
		t.Fatalf("StoreCampaignZohoEmailOutboundActionRecord(prepared) error = %v", err)
	}
	if err := StoreCampaignZohoEmailOutboundActionRecord(root, testCampaignZohoEmailOutboundActionRecord("job-2", 1, verified)); err != nil {
		t.Fatalf("StoreCampaignZohoEmailOutboundActionRecord(verified) error = %v", err)
	}
	if err := StoreCampaignZohoEmailOutboundActionRecord(root, testCampaignZohoEmailOutboundActionRecord("job-3", 1, otherCampaign)); err != nil {
		t.Fatalf("StoreCampaignZohoEmailOutboundActionRecord(other) error = %v", err)
	}

	records, err := ListCommittedCampaignZohoEmailOutboundActionRecordsByCampaign(root, "campaign-zoho")
	if err != nil {
		t.Fatalf("ListCommittedCampaignZohoEmailOutboundActionRecordsByCampaign() error = %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("ListCommittedCampaignZohoEmailOutboundActionRecordsByCampaign() len = %d, want 1 deduped record", len(records))
	}
	if records[0].JobID != "job-2" {
		t.Fatalf("ListCommittedCampaignZohoEmailOutboundActionRecordsByCampaign()[0].JobID = %q, want terminal verified job record", records[0].JobID)
	}
	if records[0].State != string(CampaignZohoEmailOutboundActionStateVerified) {
		t.Fatalf("ListCommittedCampaignZohoEmailOutboundActionRecordsByCampaign()[0].State = %q, want verified", records[0].State)
	}
}

func mustBuildPreparedCampaignZohoEmailOutboundAction(t *testing.T, stepID, campaignID, subject string, preparedAt time.Time) CampaignZohoEmailOutboundAction {
	t.Helper()

	action, err := BuildCampaignZohoEmailOutboundPreparedAction(
		stepID,
		campaignID,
		"3323462000000008002",
		"frank@omou.online",
		"Frank",
		CampaignZohoEmailAddressing{
			To:  []string{"person@example.com"},
			CC:  []string{"copy@example.com"},
			BCC: []string{"blind@example.com"},
		},
		subject,
		"plaintext",
		"Hello from Frank",
		preparedAt,
	)
	if err != nil {
		t.Fatalf("BuildCampaignZohoEmailOutboundPreparedAction() error = %v", err)
	}
	return action
}

func mustBuildVerifiedCampaignZohoEmailOutboundAction(t *testing.T, stepID, campaignID, subject string, preparedAt time.Time) CampaignZohoEmailOutboundAction {
	t.Helper()

	prepared := mustBuildPreparedCampaignZohoEmailOutboundAction(t, stepID, campaignID, subject, preparedAt)
	sent, err := BuildCampaignZohoEmailOutboundSentAction(prepared, FrankZohoSendReceipt{
		Provider:           "zoho_mail",
		ProviderAccountID:  "3323462000000008002",
		FromAddress:        "frank@omou.online",
		FromDisplayName:    "Frank",
		ProviderMessageID:  "1711540357880100000",
		ProviderMailID:     "<mail-1@zoho.test>",
		MIMEMessageID:      "<mime-1@example.test>",
		OriginalMessageURL: "https://mail.zoho.test/api/accounts/3323462000000008002/messages/1711540357880100000/originalmessage",
	}, preparedAt.Add(10*time.Second))
	if err != nil {
		t.Fatalf("BuildCampaignZohoEmailOutboundSentAction() error = %v", err)
	}
	sent.State = CampaignZohoEmailOutboundActionStateVerified
	sent.VerifiedAt = preparedAt.Add(20 * time.Second)
	if err := ValidateCampaignZohoEmailOutboundAction(sent); err != nil {
		t.Fatalf("ValidateCampaignZohoEmailOutboundAction(verified) error = %v", err)
	}
	return sent
}

func mustBuildFailedCampaignZohoEmailOutboundAction(t *testing.T, stepID, campaignID, subject string, preparedAt time.Time) CampaignZohoEmailOutboundAction {
	t.Helper()

	prepared := mustBuildPreparedCampaignZohoEmailOutboundAction(t, stepID, campaignID, subject, preparedAt)
	failed, err := BuildCampaignZohoEmailOutboundFailedAction(prepared, CampaignZohoEmailOutboundFailure{
		HTTPStatus:                400,
		ProviderStatusCode:        1510,
		ProviderStatusDescription: "recipient rejected",
	}, preparedAt.Add(5*time.Second))
	if err != nil {
		t.Fatalf("BuildCampaignZohoEmailOutboundFailedAction() error = %v", err)
	}
	return failed
}

func testCampaignZohoEmailOutboundActionRecord(jobID string, lastSeq uint64, action CampaignZohoEmailOutboundAction) CampaignZohoEmailOutboundActionRecord {
	normalized := NormalizeCampaignZohoEmailOutboundAction(action)
	return CampaignZohoEmailOutboundActionRecord{
		RecordVersion:      StoreRecordVersion,
		LastSeq:            lastSeq,
		ActionID:           normalized.ActionID,
		JobID:              jobID,
		StepID:             normalized.StepID,
		CampaignID:         normalized.CampaignID,
		State:              string(normalized.State),
		Provider:           normalized.Provider,
		ProviderAccountID:  normalized.ProviderAccountID,
		FromAddress:        normalized.FromAddress,
		FromDisplayName:    normalized.FromDisplayName,
		Addressing:         normalized.Addressing,
		Subject:            normalized.Subject,
		BodyFormat:         normalized.BodyFormat,
		BodySHA256:         normalized.BodySHA256,
		PreparedAt:         normalized.PreparedAt,
		SentAt:             normalized.SentAt,
		VerifiedAt:         normalized.VerifiedAt,
		FailedAt:           normalized.FailedAt,
		ProviderMessageID:  normalized.ProviderMessageID,
		ProviderMailID:     normalized.ProviderMailID,
		MIMEMessageID:      normalized.MIMEMessageID,
		OriginalMessageURL: normalized.OriginalMessageURL,
		Failure:            normalized.Failure,
	}
}
