package missioncontrol

import (
	"testing"
	"time"
)

func TestAttributedCampaignZohoEmailOutboundActionForBounceMatchesUniqueOriginalMIMEIdentity(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 17, 18, 0, 0, 0, time.UTC)
	action := testCampaignZohoEmailOutboundActionRecord("job-a", 1, NormalizeCampaignZohoEmailOutboundAction(CampaignZohoEmailOutboundAction{
		ActionID:          "action-1",
		StepID:            "send",
		CampaignID:        "campaign-mail",
		State:             CampaignZohoEmailOutboundActionStateVerified,
		Provider:          "zoho_mail",
		ProviderAccountID: "3323462000000008002",
		FromAddress:       "frank@example.com",
		Addressing: CampaignZohoEmailAddressing{
			To: []string{"person@example.com"},
		},
		Subject:            "Frank intro",
		BodyFormat:         "plaintext",
		BodySHA256:         "body-sha",
		PreparedAt:         now.Add(-2 * time.Minute),
		SentAt:             now.Add(-90 * time.Second),
		VerifiedAt:         now.Add(-80 * time.Second),
		ProviderMessageID:  "1711540357880100001",
		ProviderMailID:     "<sent-1@zoho.test>",
		MIMEMessageID:      "<mime-1@example.test>",
		OriginalMessageURL: "https://mail.zoho.test/api/accounts/3323462000000008002/messages/1711540357880100001/originalmessage",
	}))

	evidence := NormalizeFrankZohoBounceEvidence(FrankZohoBounceEvidence{
		StepID:                "send",
		Provider:              "zoho_mail",
		ProviderAccountID:     "3323462000000008002",
		ProviderMessageID:     "1711540357880102000",
		OriginalMIMEMessageID: "<mime-1@example.test>",
		ReceivedAt:            now,
		OriginalMessageURL:    "https://mail.zoho.test/api/accounts/3323462000000008002/messages/1711540357880102000/originalmessage",
	})
	evidence.BounceID = normalizedFrankZohoBounceEvidenceID(evidence)

	got, ok := attributedCampaignZohoEmailOutboundActionForBounce(evidence, []CampaignZohoEmailOutboundActionRecord{action})
	if !ok {
		t.Fatal("attributedCampaignZohoEmailOutboundActionForBounce() ok = false, want unique attribution")
	}
	if got.ActionID != "action-1" {
		t.Fatalf("attributedCampaignZohoEmailOutboundActionForBounce().ActionID = %q, want action-1", got.ActionID)
	}
}

func TestAttributedCampaignZohoEmailOutboundActionForBounceFailsClosedOnAmbiguousMatches(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 17, 18, 0, 0, 0, time.UTC)
	outbound := []CampaignZohoEmailOutboundActionRecord{
		testCampaignZohoEmailOutboundActionRecord("job-a", 1, NormalizeCampaignZohoEmailOutboundAction(CampaignZohoEmailOutboundAction{
			ActionID:          "action-1",
			StepID:            "send",
			CampaignID:        "campaign-mail-a",
			State:             CampaignZohoEmailOutboundActionStateSent,
			Provider:          "zoho_mail",
			ProviderAccountID: "3323462000000008002",
			FromAddress:       "frank@example.com",
			Addressing:        CampaignZohoEmailAddressing{To: []string{"person@example.com"}},
			Subject:           "First",
			BodyFormat:        "plaintext",
			BodySHA256:        "body-a",
			PreparedAt:        now.Add(-3 * time.Minute),
			SentAt:            now.Add(-2 * time.Minute),
			MIMEMessageID:     "<mime-shared@example.test>",
		})),
		testCampaignZohoEmailOutboundActionRecord("job-b", 1, NormalizeCampaignZohoEmailOutboundAction(CampaignZohoEmailOutboundAction{
			ActionID:          "action-2",
			StepID:            "send",
			CampaignID:        "campaign-mail-b",
			State:             CampaignZohoEmailOutboundActionStateSent,
			Provider:          "zoho_mail",
			ProviderAccountID: "3323462000000008002",
			FromAddress:       "frank@example.com",
			Addressing:        CampaignZohoEmailAddressing{To: []string{"second@example.com"}},
			Subject:           "Second",
			BodyFormat:        "plaintext",
			BodySHA256:        "body-b",
			PreparedAt:        now.Add(-150 * time.Second),
			SentAt:            now.Add(-140 * time.Second),
			MIMEMessageID:     "<mime-shared@example.test>",
		})),
	}

	evidence := NormalizeFrankZohoBounceEvidence(FrankZohoBounceEvidence{
		StepID:                "send",
		Provider:              "zoho_mail",
		ProviderAccountID:     "3323462000000008002",
		ProviderMessageID:     "1711540357880102000",
		OriginalMIMEMessageID: "<mime-shared@example.test>",
		ReceivedAt:            now,
		OriginalMessageURL:    "https://mail.zoho.test/api/accounts/3323462000000008002/messages/1711540357880102000/originalmessage",
	})
	evidence.BounceID = normalizedFrankZohoBounceEvidenceID(evidence)

	if _, ok := attributedCampaignZohoEmailOutboundActionForBounce(evidence, outbound); ok {
		t.Fatal("attributedCampaignZohoEmailOutboundActionForBounce() ok = true, want fail-closed ambiguous attribution")
	}
}
