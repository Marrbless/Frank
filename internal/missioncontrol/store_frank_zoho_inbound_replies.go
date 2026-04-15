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

type FrankZohoInboundReplyRecord struct {
	RecordVersion      int       `json:"record_version"`
	LastSeq            uint64    `json:"last_seq"`
	ReplyID            string    `json:"reply_id"`
	JobID              string    `json:"job_id"`
	StepID             string    `json:"step_id"`
	AttemptID          string    `json:"attempt_id,omitempty"`
	Provider           string    `json:"provider"`
	ProviderAccountID  string    `json:"provider_account_id"`
	ProviderMessageID  string    `json:"provider_message_id"`
	ProviderMailID     string    `json:"provider_mail_id,omitempty"`
	MIMEMessageID      string    `json:"mime_message_id,omitempty"`
	InReplyTo          string    `json:"in_reply_to,omitempty"`
	References         []string  `json:"references,omitempty"`
	FromAddress        string    `json:"from_address,omitempty"`
	FromDisplayName    string    `json:"from_display_name,omitempty"`
	Subject            string    `json:"subject,omitempty"`
	ReceivedAt         time.Time `json:"received_at"`
	OriginalMessageURL string    `json:"original_message_url"`
}

func ValidateFrankZohoInboundReplyRecord(record FrankZohoInboundReplyRecord) error {
	if record.RecordVersion <= 0 {
		return fmt.Errorf("mission store frank zoho inbound reply record_version must be positive")
	}
	if record.LastSeq == 0 {
		return fmt.Errorf("mission store frank zoho inbound reply last_seq must be positive")
	}
	if strings.TrimSpace(record.ReplyID) == "" {
		return fmt.Errorf("mission store frank zoho inbound reply reply_id is required")
	}
	if strings.TrimSpace(record.JobID) == "" {
		return fmt.Errorf("mission store frank zoho inbound reply job_id is required")
	}
	reply := FrankZohoInboundReply{
		ReplyID:            record.ReplyID,
		StepID:             record.StepID,
		Provider:           record.Provider,
		ProviderAccountID:  record.ProviderAccountID,
		ProviderMessageID:  record.ProviderMessageID,
		ProviderMailID:     record.ProviderMailID,
		MIMEMessageID:      record.MIMEMessageID,
		InReplyTo:          record.InReplyTo,
		References:         append([]string(nil), record.References...),
		FromAddress:        record.FromAddress,
		FromDisplayName:    record.FromDisplayName,
		Subject:            record.Subject,
		ReceivedAt:         record.ReceivedAt,
		OriginalMessageURL: record.OriginalMessageURL,
	}
	if err := ValidateFrankZohoInboundReply(reply); err != nil {
		return err
	}
	return nil
}

func StoreFrankZohoInboundReplyRecord(root string, record FrankZohoInboundReplyRecord) error {
	if err := ValidateStoreRoot(root); err != nil {
		return err
	}
	if err := ValidateFrankZohoInboundReplyRecord(record); err != nil {
		return err
	}
	return WriteStoreJSONAtomic(StoreFrankZohoInboundReplyPath(root, record.JobID, record.ReplyID), record)
}

func LoadFrankZohoInboundReplyRecord(root, jobID, replyID string) (FrankZohoInboundReplyRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return FrankZohoInboundReplyRecord{}, err
	}
	var record FrankZohoInboundReplyRecord
	if err := LoadStoreJSON(StoreFrankZohoInboundReplyPath(root, jobID, replyID), &record); err != nil {
		return FrankZohoInboundReplyRecord{}, err
	}
	if err := ValidateFrankZohoInboundReplyRecord(record); err != nil {
		return FrankZohoInboundReplyRecord{}, err
	}
	return record, nil
}

func ListCommittedFrankZohoInboundReplyRecords(root, jobID string) ([]FrankZohoInboundReplyRecord, error) {
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
	replyIDs, err := listStoreRecordKeys(StoreFrankZohoInboundRepliesDir(root, jobID))
	if err != nil {
		return nil, err
	}
	records := make([]FrankZohoInboundReplyRecord, 0, len(replyIDs))
	for _, replyID := range replyIDs {
		record, err := loadLatestVisibleVersionedJSONRecordAtOrBefore(
			storeFrankZohoInboundReplyVersionsDir(root, jobID, replyID),
			jobRuntime.AppliedSeq,
			resolver,
			loadFrankZohoInboundReplyRecordFile,
			func(record FrankZohoInboundReplyRecord) uint64 { return record.LastSeq },
			func(record FrankZohoInboundReplyRecord) string { return record.AttemptID },
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
		return records[i].ReplyID < records[j].ReplyID
	})
	return records, nil
}

func loadFrankZohoInboundReplyRecordFile(path string) (FrankZohoInboundReplyRecord, error) {
	var record FrankZohoInboundReplyRecord
	if err := LoadStoreJSON(path, &record); err != nil {
		return FrankZohoInboundReplyRecord{}, err
	}
	if err := ValidateFrankZohoInboundReplyRecord(record); err != nil {
		return FrankZohoInboundReplyRecord{}, err
	}
	return record, nil
}

func storeFrankZohoInboundReplyVersionsDir(root, jobID, replyID string) string {
	return filepath.Join(StoreFrankZohoInboundRepliesDir(root, jobID), replyID)
}

func storeFrankZohoInboundReplyVersionPath(root, jobID, replyID string, seq uint64, attemptID string) string {
	return filepath.Join(storeFrankZohoInboundReplyVersionsDir(root, jobID, replyID), storeVersionFilename(seq, attemptID))
}
