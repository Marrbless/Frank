package missioncontrol

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type FrankZohoBounceEvidenceRecord struct {
	RecordVersion             int       `json:"record_version"`
	LastSeq                   uint64    `json:"last_seq"`
	BounceID                  string    `json:"bounce_id"`
	JobID                     string    `json:"job_id"`
	StepID                    string    `json:"step_id"`
	AttemptID                 string    `json:"attempt_id,omitempty"`
	Provider                  string    `json:"provider"`
	ProviderAccountID         string    `json:"provider_account_id"`
	ProviderMessageID         string    `json:"provider_message_id"`
	ProviderMailID            string    `json:"provider_mail_id,omitempty"`
	MIMEMessageID             string    `json:"mime_message_id,omitempty"`
	InReplyTo                 string    `json:"in_reply_to,omitempty"`
	References                []string  `json:"references,omitempty"`
	OriginalProviderMessageID string    `json:"original_provider_message_id,omitempty"`
	OriginalProviderMailID    string    `json:"original_provider_mail_id,omitempty"`
	OriginalMIMEMessageID     string    `json:"original_mime_message_id,omitempty"`
	FinalRecipient            string    `json:"final_recipient,omitempty"`
	DiagnosticCode            string    `json:"diagnostic_code,omitempty"`
	ReceivedAt                time.Time `json:"received_at"`
	OriginalMessageURL        string    `json:"original_message_url"`
	CampaignID                string    `json:"campaign_id,omitempty"`
	OutboundActionID          string    `json:"outbound_action_id,omitempty"`
}

func ValidateFrankZohoBounceEvidenceRecord(record FrankZohoBounceEvidenceRecord) error {
	if record.RecordVersion <= 0 {
		return fmt.Errorf("mission store frank zoho bounce evidence record_version must be positive")
	}
	if record.LastSeq == 0 {
		return fmt.Errorf("mission store frank zoho bounce evidence last_seq must be positive")
	}
	if strings.TrimSpace(record.BounceID) == "" {
		return fmt.Errorf("mission store frank zoho bounce evidence bounce_id is required")
	}
	if strings.TrimSpace(record.JobID) == "" {
		return fmt.Errorf("mission store frank zoho bounce evidence job_id is required")
	}
	evidence := FrankZohoBounceEvidence{
		BounceID:                  record.BounceID,
		StepID:                    record.StepID,
		Provider:                  record.Provider,
		ProviderAccountID:         record.ProviderAccountID,
		ProviderMessageID:         record.ProviderMessageID,
		ProviderMailID:            record.ProviderMailID,
		MIMEMessageID:             record.MIMEMessageID,
		InReplyTo:                 record.InReplyTo,
		References:                append([]string(nil), record.References...),
		OriginalProviderMessageID: record.OriginalProviderMessageID,
		OriginalProviderMailID:    record.OriginalProviderMailID,
		OriginalMIMEMessageID:     record.OriginalMIMEMessageID,
		FinalRecipient:            record.FinalRecipient,
		DiagnosticCode:            record.DiagnosticCode,
		ReceivedAt:                record.ReceivedAt,
		OriginalMessageURL:        record.OriginalMessageURL,
		CampaignID:                record.CampaignID,
		OutboundActionID:          record.OutboundActionID,
	}
	return ValidateFrankZohoBounceEvidence(evidence)
}

func StoreFrankZohoBounceEvidenceRecord(root string, record FrankZohoBounceEvidenceRecord) error {
	if err := ValidateStoreRoot(root); err != nil {
		return err
	}
	if err := ValidateFrankZohoBounceEvidenceRecord(record); err != nil {
		return err
	}
	return WriteStoreJSONAtomic(StoreFrankZohoBounceEvidencePath(root, record.JobID, record.BounceID), record)
}

func LoadFrankZohoBounceEvidenceRecord(root, jobID, bounceID string) (FrankZohoBounceEvidenceRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return FrankZohoBounceEvidenceRecord{}, err
	}
	var record FrankZohoBounceEvidenceRecord
	if err := LoadStoreJSON(StoreFrankZohoBounceEvidencePath(root, jobID, bounceID), &record); err != nil {
		return FrankZohoBounceEvidenceRecord{}, err
	}
	if err := ValidateFrankZohoBounceEvidenceRecord(record); err != nil {
		return FrankZohoBounceEvidenceRecord{}, err
	}
	return record, nil
}

func ListCommittedFrankZohoBounceEvidenceRecords(root, jobID string) ([]FrankZohoBounceEvidenceRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return nil, err
	}
	jobRuntime, err := LoadCommittedJobRuntimeRecord(root, jobID)
	if err != nil {
		if errors.Is(err, ErrJobRuntimeRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	resolver := newStoreCommittedAttemptResolver(root, jobID)
	bounceIDs, err := listStoreRecordKeys(StoreFrankZohoBounceEvidenceDir(root, jobID))
	if err != nil {
		return nil, err
	}
	records := make([]FrankZohoBounceEvidenceRecord, 0, len(bounceIDs))
	for _, bounceID := range bounceIDs {
		record, err := loadLatestVisibleVersionedJSONRecordAtOrBefore(
			storeFrankZohoBounceEvidenceVersionsDir(root, jobID, bounceID),
			jobRuntime.AppliedSeq,
			resolver,
			loadFrankZohoBounceEvidenceRecordFile,
			func(record FrankZohoBounceEvidenceRecord) uint64 { return record.LastSeq },
			func(record FrankZohoBounceEvidenceRecord) string { return record.AttemptID },
		)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	sort.SliceStable(records, func(i, j int) bool {
		if !records[i].ReceivedAt.Equal(records[j].ReceivedAt) {
			return records[i].ReceivedAt.Before(records[j].ReceivedAt)
		}
		return records[i].BounceID < records[j].BounceID
	})
	return records, nil
}

func loadFrankZohoBounceEvidenceRecordFile(path string) (FrankZohoBounceEvidenceRecord, error) {
	var record FrankZohoBounceEvidenceRecord
	if err := LoadStoreJSON(path, &record); err != nil {
		return FrankZohoBounceEvidenceRecord{}, err
	}
	if err := ValidateFrankZohoBounceEvidenceRecord(record); err != nil {
		return FrankZohoBounceEvidenceRecord{}, err
	}
	return record, nil
}

func storeFrankZohoBounceEvidenceVersionsDir(root, jobID, bounceID string) string {
	return filepath.Join(StoreFrankZohoBounceEvidenceDir(root, jobID), bounceID)
}

func storeFrankZohoBounceEvidenceVersionPath(root, jobID, bounceID string, seq uint64, attemptID string) string {
	return filepath.Join(storeFrankZohoBounceEvidenceVersionsDir(root, jobID, bounceID), storeVersionFilename(seq, attemptID))
}
