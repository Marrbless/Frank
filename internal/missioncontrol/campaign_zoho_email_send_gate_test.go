package missioncontrol

import (
	"strings"
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

	decision, err := DeriveCampaignZohoEmailSendGateDecision(campaign, records, nil)
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

	decision, err := DeriveCampaignZohoEmailSendGateDecision(campaign, records, nil)
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

func TestDeriveCampaignZohoEmailSendGateDecisionCountsAttributedReplies(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 15, 19, 0, 0, 0, time.UTC)
	campaign := validCampaignRecord(now, func(record *CampaignRecord) {
		record.CampaignID = "campaign-zoho"
		record.StopConditions = []string{"stop after 3 replies"}
		record.FailureThreshold = CampaignFailureThreshold{Metric: "rejections", Limit: 3}
	})

	records := []CampaignZohoEmailOutboundActionRecord{
		testCampaignZohoEmailOutboundActionRecord("job-1", 1, mustBuildVerifiedCampaignZohoEmailOutboundAction(t, "step-1", "campaign-zoho", "subject-1", now)),
		testCampaignZohoEmailOutboundActionRecord("job-2", 1, mustBuildVerifiedCampaignZohoEmailOutboundAction(t, "step-2", "campaign-other", "subject-2", now.Add(time.Minute))),
	}
	inbound := []FrankZohoInboundReplyRecord{
		testFrankZohoInboundReplyRecord("job-r1", 1, FrankZohoInboundReply{
			Provider:           "zoho_mail",
			ProviderAccountID:  "3323462000000008002",
			ProviderMessageID:  "1711540357880101000",
			ProviderMailID:     "<reply-1@zoho.test>",
			InReplyTo:          "<step-1@example.test>",
			References:         []string{"<step-1@example.test>"},
			ReceivedAt:         now.Add(2 * time.Minute),
			OriginalMessageURL: "https://mail.zoho.test/api/accounts/3323462000000008002/messages/1711540357880101000/originalmessage",
		}),
		testFrankZohoInboundReplyRecord("job-r2", 1, FrankZohoInboundReply{
			Provider:           "zoho_mail",
			ProviderAccountID:  "3323462000000008002",
			ProviderMessageID:  "1711540357880101001",
			ProviderMailID:     "<reply-2@zoho.test>",
			References:         []string{"<step-1@example.test>", "<step-2@example.test>"},
			ReceivedAt:         now.Add(3 * time.Minute),
			OriginalMessageURL: "https://mail.zoho.test/api/accounts/3323462000000008002/messages/1711540357880101001/originalmessage",
		}),
	}

	decision, err := DeriveCampaignZohoEmailSendGateDecision(campaign, records, inbound)
	if err != nil {
		t.Fatalf("DeriveCampaignZohoEmailSendGateDecision() error = %v", err)
	}
	if !decision.Allowed {
		t.Fatalf("Decision.Allowed = false, want true below reply stop limit")
	}
	if decision.AttributedReplyCount != 1 {
		t.Fatalf("Decision.AttributedReplyCount = %d, want 1 uniquely attributed reply", decision.AttributedReplyCount)
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

	decision, err := DeriveCampaignZohoEmailSendGateDecision(campaign, records, nil)
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

func TestDeriveCampaignZohoEmailSendGateDecisionHaltsAtAmbiguousOutcomeThreshold(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 15, 19, 20, 0, 0, time.UTC)
	campaign := validCampaignRecord(now, func(record *CampaignRecord) {
		record.CampaignID = "campaign-zoho"
		record.StopConditions = []string{"stop after 5 verified sends"}
		record.FailureThreshold = CampaignFailureThreshold{Metric: "ambiguous_outcomes", Limit: 2}
	})

	records := []CampaignZohoEmailOutboundActionRecord{
		testCampaignZohoEmailOutboundActionRecord("job-1", 1, mustBuildPreparedCampaignZohoEmailOutboundAction(t, "step-1", "campaign-zoho", "subject-1", now)),
		testCampaignZohoEmailOutboundActionRecord("job-2", 1, mustBuildPreparedCampaignZohoEmailOutboundAction(t, "step-2", "campaign-zoho", "subject-2", now.Add(time.Minute))),
		testCampaignZohoEmailOutboundActionRecord("job-3", 1, mustBuildVerifiedCampaignZohoEmailOutboundAction(t, "step-3", "campaign-zoho", "subject-3", now.Add(2*time.Minute))),
	}

	decision, err := DeriveCampaignZohoEmailSendGateDecision(campaign, records, nil)
	if err != nil {
		t.Fatalf("DeriveCampaignZohoEmailSendGateDecision() error = %v", err)
	}
	if decision.Allowed {
		t.Fatal("Decision.Allowed = true, want false once ambiguous-outcome threshold is reached")
	}
	if !decision.Halted {
		t.Fatal("Decision.Halted = false, want true once ambiguous-outcome threshold is reached")
	}
	if decision.FailureThresholdMetric != "ambiguous_outcomes" {
		t.Fatalf("Decision.FailureThresholdMetric = %q, want ambiguous_outcomes", decision.FailureThresholdMetric)
	}
	if decision.AmbiguousOutcomeCount != 2 {
		t.Fatalf("Decision.AmbiguousOutcomeCount = %d, want 2", decision.AmbiguousOutcomeCount)
	}
	if decision.FailureCount != 0 {
		t.Fatalf("Decision.FailureCount = %d, want 0", decision.FailureCount)
	}
}

func TestDeriveCampaignZohoEmailSendGateDecisionHaltsAtBouncedMessageThreshold(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 17, 19, 30, 0, 0, time.UTC)
	campaign := validCampaignRecord(now, func(record *CampaignRecord) {
		record.CampaignID = "campaign-zoho"
		record.StopConditions = []string{"stop after 5 verified sends"}
		record.FailureThreshold = CampaignFailureThreshold{Metric: "bounced_messages", Limit: 2}
	})

	records := []CampaignZohoEmailOutboundActionRecord{
		testCampaignZohoEmailOutboundActionRecord("job-1", 1, mustBuildVerifiedCampaignZohoEmailOutboundAction(t, "step-1", "campaign-zoho", "subject-1", now)),
		testCampaignZohoEmailOutboundActionRecord("job-2", 1, mustBuildVerifiedCampaignZohoEmailOutboundAction(t, "step-2", "campaign-zoho", "subject-2", now.Add(time.Minute))),
	}
	bounces := []FrankZohoBounceEvidenceRecord{
		testFrankZohoBounceEvidenceRecord(t, "job-b1", 1, NormalizeFrankZohoBounceEvidence(FrankZohoBounceEvidence{
			StepID:             "sync",
			Provider:           "zoho_mail",
			ProviderAccountID:  "3323462000000008002",
			ProviderMessageID:  "1711540357880102001",
			ReceivedAt:         now.Add(2 * time.Minute),
			OriginalMessageURL: "https://mail.zoho.test/api/accounts/3323462000000008002/messages/1711540357880102001/originalmessage",
			CampaignID:         "campaign-zoho",
			OutboundActionID:   records[0].ActionID,
		})),
		testFrankZohoBounceEvidenceRecord(t, "job-b2", 1, NormalizeFrankZohoBounceEvidence(FrankZohoBounceEvidence{
			StepID:             "sync",
			Provider:           "zoho_mail",
			ProviderAccountID:  "3323462000000008002",
			ProviderMessageID:  "1711540357880102002",
			ReceivedAt:         now.Add(3 * time.Minute),
			OriginalMessageURL: "https://mail.zoho.test/api/accounts/3323462000000008002/messages/1711540357880102002/originalmessage",
			CampaignID:         "campaign-zoho",
			OutboundActionID:   records[1].ActionID,
		})),
		testFrankZohoBounceEvidenceRecord(t, "job-b3", 1, NormalizeFrankZohoBounceEvidence(FrankZohoBounceEvidence{
			StepID:             "sync",
			Provider:           "zoho_mail",
			ProviderAccountID:  "3323462000000008002",
			ProviderMessageID:  "1711540357880102003",
			ReceivedAt:         now.Add(4 * time.Minute),
			OriginalMessageURL: "https://mail.zoho.test/api/accounts/3323462000000008002/messages/1711540357880102003/originalmessage",
		})),
	}

	decision, err := DeriveCampaignZohoEmailSendGateDecisionWithBounceEvidence(campaign, records, nil, bounces)
	if err != nil {
		t.Fatalf("DeriveCampaignZohoEmailSendGateDecisionWithBounceEvidence() error = %v", err)
	}
	if decision.Allowed {
		t.Fatal("Decision.Allowed = true, want false once bounced-message threshold is reached")
	}
	if !decision.Halted {
		t.Fatal("Decision.Halted = false, want true once bounced-message threshold is reached")
	}
	if decision.FailureThresholdMetric != "bounced_messages" {
		t.Fatalf("Decision.FailureThresholdMetric = %q, want bounced_messages", decision.FailureThresholdMetric)
	}
	if decision.AttributedBounceCount != 2 {
		t.Fatalf("Decision.AttributedBounceCount = %d, want 2", decision.AttributedBounceCount)
	}
	if decision.Reason != `campaign zoho email failure_threshold "bounced_messages" reached 2/2 counted attributed bounces` {
		t.Fatalf("Decision.Reason = %q, want bounced-message threshold reason", decision.Reason)
	}
}

func TestDeriveCampaignZohoEmailSendGateDecisionHaltsAtReplyStopLimit(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 15, 19, 45, 0, 0, time.UTC)
	campaign := validCampaignRecord(now, func(record *CampaignRecord) {
		record.CampaignID = "campaign-zoho"
		record.StopConditions = []string{"stop after 2 replies"}
		record.FailureThreshold = CampaignFailureThreshold{Metric: "rejections", Limit: 3}
	})

	records := []CampaignZohoEmailOutboundActionRecord{
		testCampaignZohoEmailOutboundActionRecord("job-1", 1, mustBuildVerifiedCampaignZohoEmailOutboundAction(t, "step-1", "campaign-zoho", "subject-1", now)),
		testCampaignZohoEmailOutboundActionRecord("job-2", 1, mustBuildVerifiedCampaignZohoEmailOutboundAction(t, "step-2", "campaign-zoho", "subject-2", now.Add(time.Minute))),
	}
	inbound := []FrankZohoInboundReplyRecord{
		testFrankZohoInboundReplyRecord("job-r1", 1, FrankZohoInboundReply{
			Provider:           "zoho_mail",
			ProviderAccountID:  "3323462000000008002",
			ProviderMessageID:  "1711540357880101000",
			InReplyTo:          "<step-1@example.test>",
			ReceivedAt:         now.Add(2 * time.Minute),
			OriginalMessageURL: "https://mail.zoho.test/api/accounts/3323462000000008002/messages/1711540357880101000/originalmessage",
		}),
		testFrankZohoInboundReplyRecord("job-r2", 1, FrankZohoInboundReply{
			Provider:           "zoho_mail",
			ProviderAccountID:  "3323462000000008002",
			ProviderMessageID:  "1711540357880101001",
			InReplyTo:          "<step-1@example.test>",
			ReceivedAt:         now.Add(3 * time.Minute),
			OriginalMessageURL: "https://mail.zoho.test/api/accounts/3323462000000008002/messages/1711540357880101001/originalmessage",
		}),
	}

	decision, err := DeriveCampaignZohoEmailSendGateDecision(campaign, records, inbound)
	if err != nil {
		t.Fatalf("DeriveCampaignZohoEmailSendGateDecision() error = %v", err)
	}
	if decision.Allowed {
		t.Fatal("Decision.Allowed = true, want false once reply stop limit is reached")
	}
	if !decision.Halted {
		t.Fatal("Decision.Halted = false, want true once reply stop limit is reached")
	}
	if decision.TriggeredStopCondition != "stop after 2 replies" {
		t.Fatalf("Decision.TriggeredStopCondition = %q, want configured reply stop condition", decision.TriggeredStopCondition)
	}
	if decision.AttributedReplyCount != 2 {
		t.Fatalf("Decision.AttributedReplyCount = %d, want 2", decision.AttributedReplyCount)
	}
}

func TestDeriveCampaignZohoEmailSendGateDecisionFailsClosedOnUnsupportedStopCondition(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 15, 19, 50, 0, 0, time.UTC)
	campaign := validCampaignRecord(now, func(record *CampaignRecord) {
		record.CampaignID = "campaign-zoho"
		record.StopConditions = []string{"stop after 3 opens"}
		record.FailureThreshold = CampaignFailureThreshold{Metric: "rejections", Limit: 3}
	})

	_, err := DeriveCampaignZohoEmailSendGateDecision(campaign, nil, nil)
	if err == nil {
		t.Fatal("DeriveCampaignZohoEmailSendGateDecision() error = nil, want unsupported stop-condition rejection")
	}
	if got := err.Error(); got != `campaign zoho email stop_condition "stop after 3 opens" is not evaluable from committed outbound and inbound reply records` {
		t.Fatalf("DeriveCampaignZohoEmailSendGateDecision() error = %q, want unsupported stop-condition rejection", got)
	}
}

func TestDeriveCampaignZohoEmailSendGateDecisionFailsClosedOnUnsupportedOpenFailureThresholdMetric(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 15, 19, 55, 0, 0, time.UTC)
	campaign := validCampaignRecord(now, func(record *CampaignRecord) {
		record.CampaignID = "campaign-zoho"
		record.StopConditions = []string{"stop after 3 verified sends"}
		record.FailureThreshold = CampaignFailureThreshold{Metric: "opens", Limit: 3}
	})

	_, err := DeriveCampaignZohoEmailSendGateDecision(campaign, nil, nil)
	if err == nil {
		t.Fatal("DeriveCampaignZohoEmailSendGateDecision() error = nil, want unsupported failure-threshold rejection")
	}
	if got := err.Error(); got != `campaign zoho email failure_threshold.metric "opens" is not evaluable from committed outbound action records` {
		t.Fatalf("DeriveCampaignZohoEmailSendGateDecision() error = %q, want unsupported failure-threshold rejection", got)
	}
}

func TestDeriveCampaignZohoEmailSendGateDecisionInvariantTable(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 18, 10, 0, 0, 0, time.UTC)
	first := testCampaignZohoEmailOutboundActionRecord("job-1", 1, mustBuildVerifiedCampaignZohoEmailOutboundAction(t, "step-1", "campaign-zoho", "subject-1", now))
	second := testCampaignZohoEmailOutboundActionRecord("job-2", 1, mustBuildVerifiedCampaignZohoEmailOutboundAction(t, "step-2", "campaign-zoho", "subject-2", now.Add(time.Minute)))
	attributedReply := testFrankZohoInboundReplyRecord("job-r1", 1, FrankZohoInboundReply{
		StepID:             "sync-replies",
		Provider:           "zoho_mail",
		ProviderAccountID:  "3323462000000008002",
		ProviderMessageID:  "reply-gate-1",
		InReplyTo:          first.MIMEMessageID,
		FromAddress:        "person@example.com",
		FromAddressCount:   1,
		ReceivedAt:         now.Add(2 * time.Minute),
		OriginalMessageURL: "https://mail.zoho.test/api/accounts/3323462000000008002/messages/reply-gate-1/originalmessage",
	})
	attributedBounce := testFrankZohoBounceEvidenceRecord(t, "job-b1", 1, NormalizeFrankZohoBounceEvidence(FrankZohoBounceEvidence{
		StepID:             "sync-bounces",
		Provider:           "zoho_mail",
		ProviderAccountID:  "3323462000000008002",
		ProviderMessageID:  "bounce-gate-1",
		ReceivedAt:         now.Add(3 * time.Minute),
		OriginalMessageURL: "https://mail.zoho.test/api/accounts/3323462000000008002/messages/bounce-gate-1/originalmessage",
		CampaignID:         "campaign-zoho",
		OutboundActionID:   first.ActionID,
	}))
	unattributedBounce := testFrankZohoBounceEvidenceRecord(t, "job-b2", 1, NormalizeFrankZohoBounceEvidence(FrankZohoBounceEvidence{
		StepID:             "sync-bounces",
		Provider:           "zoho_mail",
		ProviderAccountID:  "3323462000000008002",
		ProviderMessageID:  "bounce-gate-2",
		ReceivedAt:         now.Add(4 * time.Minute),
		OriginalMessageURL: "https://mail.zoho.test/api/accounts/3323462000000008002/messages/bounce-gate-2/originalmessage",
	}))

	tests := []struct {
		name               string
		stopConditions     []string
		failureThreshold   CampaignFailureThreshold
		outbound           []CampaignZohoEmailOutboundActionRecord
		replies            []FrankZohoInboundReplyRecord
		bounces            []FrankZohoBounceEvidenceRecord
		wantAllowed        bool
		wantHalted         bool
		wantReplyCount     int
		wantBounceCount    int
		wantTriggeredStop  string
		wantReasonContains string
	}{
		{
			name:             "attributed reply and unattributed bounce stay below limits",
			stopConditions:   []string{"stop after 2 replies"},
			failureThreshold: CampaignFailureThreshold{Metric: "bounced_messages", Limit: 1},
			outbound:         []CampaignZohoEmailOutboundActionRecord{first},
			replies:          []FrankZohoInboundReplyRecord{attributedReply},
			bounces:          []FrankZohoBounceEvidenceRecord{unattributedBounce},
			wantAllowed:      true,
			wantReplyCount:   1,
			wantBounceCount:  0,
		},
		{
			name:               "attributed reply trips reply stop condition",
			stopConditions:     []string{"stop after first reply"},
			failureThreshold:   CampaignFailureThreshold{Metric: "bounced_messages", Limit: 2},
			outbound:           []CampaignZohoEmailOutboundActionRecord{first},
			replies:            []FrankZohoInboundReplyRecord{attributedReply},
			bounces:            []FrankZohoBounceEvidenceRecord{unattributedBounce},
			wantHalted:         true,
			wantReplyCount:     1,
			wantBounceCount:    0,
			wantTriggeredStop:  "stop after first reply",
			wantReasonContains: "triggered after 1 attributed replies",
		},
		{
			name:               "attributed bounce trips bounce threshold",
			stopConditions:     []string{"stop after 3 replies"},
			failureThreshold:   CampaignFailureThreshold{Metric: "bounced_messages", Limit: 1},
			outbound:           []CampaignZohoEmailOutboundActionRecord{first, second},
			replies:            nil,
			bounces:            []FrankZohoBounceEvidenceRecord{attributedBounce, unattributedBounce},
			wantHalted:         true,
			wantReplyCount:     0,
			wantBounceCount:    1,
			wantReasonContains: `failure_threshold "bounced_messages" reached 1/1 counted attributed bounces`,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			campaign := validCampaignRecord(now, func(record *CampaignRecord) {
				record.CampaignID = "campaign-zoho"
				record.StopConditions = tc.stopConditions
				record.FailureThreshold = tc.failureThreshold
			})
			decision, err := DeriveCampaignZohoEmailSendGateDecisionWithBounceEvidence(campaign, tc.outbound, tc.replies, tc.bounces)
			if err != nil {
				t.Fatalf("DeriveCampaignZohoEmailSendGateDecisionWithBounceEvidence() error = %v", err)
			}
			if decision.Allowed != tc.wantAllowed {
				t.Fatalf("Allowed = %v, want %v: %#v", decision.Allowed, tc.wantAllowed, decision)
			}
			if decision.Halted != tc.wantHalted {
				t.Fatalf("Halted = %v, want %v: %#v", decision.Halted, tc.wantHalted, decision)
			}
			if decision.AttributedReplyCount != tc.wantReplyCount {
				t.Fatalf("AttributedReplyCount = %d, want %d", decision.AttributedReplyCount, tc.wantReplyCount)
			}
			if decision.AttributedBounceCount != tc.wantBounceCount {
				t.Fatalf("AttributedBounceCount = %d, want %d", decision.AttributedBounceCount, tc.wantBounceCount)
			}
			if decision.TriggeredStopCondition != tc.wantTriggeredStop {
				t.Fatalf("TriggeredStopCondition = %q, want %q", decision.TriggeredStopCondition, tc.wantTriggeredStop)
			}
			if tc.wantReasonContains != "" && !strings.Contains(decision.Reason, tc.wantReasonContains) {
				t.Fatalf("Reason = %q, want substring %q", decision.Reason, tc.wantReasonContains)
			}
		})
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
		ProviderMessageID:  "1711540357880100000-" + stepID,
		ProviderMailID:     "<mail-" + stepID + "@zoho.test>",
		MIMEMessageID:      "<" + stepID + "@example.test>",
		OriginalMessageURL: "https://mail.zoho.test/api/accounts/3323462000000008002/messages/1711540357880100000-" + stepID + "/originalmessage",
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

func testFrankZohoInboundReplyRecord(jobID string, lastSeq uint64, reply FrankZohoInboundReply) FrankZohoInboundReplyRecord {
	normalized := NormalizeFrankZohoInboundReply(reply)
	if normalized.ReplyID == "" {
		normalized.ReplyID = normalizedFrankZohoInboundReplyID(normalized)
	}
	stepID := normalized.StepID
	if stepID == "" {
		stepID = "sync-replies"
	}
	return FrankZohoInboundReplyRecord{
		RecordVersion:      StoreRecordVersion,
		LastSeq:            lastSeq,
		ReplyID:            normalized.ReplyID,
		JobID:              jobID,
		StepID:             stepID,
		Provider:           normalized.Provider,
		ProviderAccountID:  normalized.ProviderAccountID,
		ProviderMessageID:  normalized.ProviderMessageID,
		ProviderMailID:     normalized.ProviderMailID,
		MIMEMessageID:      normalized.MIMEMessageID,
		InReplyTo:          normalized.InReplyTo,
		References:         append([]string(nil), normalized.References...),
		FromAddress:        normalized.FromAddress,
		FromDisplayName:    normalized.FromDisplayName,
		FromAddressCount:   normalized.FromAddressCount,
		Subject:            normalized.Subject,
		ReceivedAt:         normalized.ReceivedAt,
		OriginalMessageURL: normalized.OriginalMessageURL,
	}
}

func testFrankZohoBounceEvidenceRecord(t *testing.T, jobID string, lastSeq uint64, evidence FrankZohoBounceEvidence) FrankZohoBounceEvidenceRecord {
	t.Helper()
	normalized := NormalizeFrankZohoBounceEvidence(evidence)
	if normalized.BounceID == "" {
		runtime, changed, err := AppendFrankZohoBounceEvidence(JobRuntimeState{}, normalized)
		if err != nil {
			t.Fatalf("AppendFrankZohoBounceEvidence() error = %v", err)
		}
		if !changed || len(runtime.FrankZohoBounceEvidence) != 1 {
			t.Fatalf("AppendFrankZohoBounceEvidence() changed = %v len = %d, want one normalized bounce evidence", changed, len(runtime.FrankZohoBounceEvidence))
		}
		normalized = runtime.FrankZohoBounceEvidence[0]
	}
	stepID := normalized.StepID
	if stepID == "" {
		stepID = "sync-bounces"
	}
	return FrankZohoBounceEvidenceRecord{
		RecordVersion:             StoreRecordVersion,
		LastSeq:                   lastSeq,
		BounceID:                  normalized.BounceID,
		JobID:                     jobID,
		StepID:                    stepID,
		Provider:                  normalized.Provider,
		ProviderAccountID:         normalized.ProviderAccountID,
		ProviderMessageID:         normalized.ProviderMessageID,
		ProviderMailID:            normalized.ProviderMailID,
		MIMEMessageID:             normalized.MIMEMessageID,
		InReplyTo:                 normalized.InReplyTo,
		References:                append([]string(nil), normalized.References...),
		OriginalProviderMessageID: normalized.OriginalProviderMessageID,
		OriginalProviderMailID:    normalized.OriginalProviderMailID,
		OriginalMIMEMessageID:     normalized.OriginalMIMEMessageID,
		FinalRecipient:            normalized.FinalRecipient,
		DiagnosticCode:            normalized.DiagnosticCode,
		ReceivedAt:                normalized.ReceivedAt,
		OriginalMessageURL:        normalized.OriginalMessageURL,
		CampaignID:                normalized.CampaignID,
		OutboundActionID:          normalized.OutboundActionID,
	}
}
