package missioncontrol

import (
	"strings"
	"testing"
	"time"
)

func TestAppendFrankZohoBounceEvidenceNormalizesAndAssignsStableID(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 17, 13, 15, 0, 0, time.UTC)
	runtime, changed, err := AppendFrankZohoBounceEvidence(JobRuntimeState{}, FrankZohoBounceEvidence{
		StepID:                    " build ",
		Provider:                  " zoho_mail ",
		ProviderAccountID:         " 3323462000000008002 ",
		ProviderMessageID:         " 1711540357880102000 ",
		ProviderMailID:            " <bounce-1@zoho.test> ",
		MIMEMessageID:             " <bounce-1@example.test> ",
		InReplyTo:                 " <mime-1@example.test> ",
		References:                []string{"<seed@example.test>", " <mime-1@example.test> ", "<seed@example.test>"},
		OriginalProviderMessageID: " 1711540357880100001 ",
		OriginalProviderMailID:    " <sent-1@zoho.test> ",
		OriginalMIMEMessageID:     " <mime-1@example.test> ",
		FinalRecipient:            " person@example.com ",
		DiagnosticCode:            " smtp; 550 5.1.1 mailbox unavailable ",
		ReceivedAt:                now,
		OriginalMessageURL:        " https://mail.zoho.test/api/accounts/3323462000000008002/messages/1711540357880102000/originalmessage ",
		CampaignID:                " campaign-mail ",
		OutboundActionID:          " action-1 ",
	})
	if err != nil {
		t.Fatalf("AppendFrankZohoBounceEvidence() error = %v", err)
	}
	if !changed || len(runtime.FrankZohoBounceEvidence) != 1 {
		t.Fatalf("AppendFrankZohoBounceEvidence() changed = %v len = %d, want one normalized evidence record", changed, len(runtime.FrankZohoBounceEvidence))
	}

	got := runtime.FrankZohoBounceEvidence[0]
	if got.BounceID == "" {
		t.Fatal("BounceID = empty, want stable normalized id")
	}
	if got.BounceID != normalizedFrankZohoBounceEvidenceID(got) {
		t.Fatalf("BounceID = %q, want normalized id %q", got.BounceID, normalizedFrankZohoBounceEvidenceID(got))
	}
	if got.StepID != "build" {
		t.Fatalf("StepID = %q, want build", got.StepID)
	}
	if got.ProviderMessageID != "1711540357880102000" {
		t.Fatalf("ProviderMessageID = %q, want trimmed provider id", got.ProviderMessageID)
	}
	if len(got.References) != 2 || got.References[1] != "<mime-1@example.test>" {
		t.Fatalf("References = %#v, want normalized unique reference chain", got.References)
	}
	if got.OriginalMIMEMessageID != "<mime-1@example.test>" {
		t.Fatalf("OriginalMIMEMessageID = %q, want outbound MIME linkage", got.OriginalMIMEMessageID)
	}
	if got.CampaignID != "campaign-mail" || got.OutboundActionID != "action-1" {
		t.Fatalf("attribution = (%q, %q), want committed campaign/action linkage", got.CampaignID, got.OutboundActionID)
	}

	runtime, changed, err = AppendFrankZohoBounceEvidence(runtime, got)
	if err != nil {
		t.Fatalf("AppendFrankZohoBounceEvidence(duplicate) error = %v", err)
	}
	if changed {
		t.Fatal("AppendFrankZohoBounceEvidence(duplicate) changed = true, want deterministic no-op")
	}
}

func TestAppendFrankZohoBounceEvidenceRejectsPartialAttribution(t *testing.T) {
	t.Parallel()

	_, _, err := AppendFrankZohoBounceEvidence(JobRuntimeState{}, FrankZohoBounceEvidence{
		StepID:             "build",
		Provider:           "zoho_mail",
		ProviderAccountID:  "3323462000000008002",
		ProviderMessageID:  "1711540357880102000",
		ReceivedAt:         time.Date(2026, 4, 17, 13, 20, 0, 0, time.UTC),
		OriginalMessageURL: "https://mail.zoho.test/api/accounts/3323462000000008002/messages/1711540357880102000/originalmessage",
		CampaignID:         "campaign-mail",
	})
	if err == nil {
		t.Fatal("AppendFrankZohoBounceEvidence() error = nil, want partial-attribution rejection")
	}
	if !strings.Contains(err.Error(), "campaign_id and outbound_action_id must either both be set or both be empty") {
		t.Fatalf("AppendFrankZohoBounceEvidence() error = %q, want partial-attribution rejection", err.Error())
	}
}
