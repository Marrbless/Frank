package missioncontrol

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func StoreCampaignZohoEmailReplyWorkItemRecord(root string, record CampaignZohoEmailReplyWorkItemRecord) error {
	if err := ValidateStoreRoot(root); err != nil {
		return err
	}
	if err := ValidateCampaignZohoEmailReplyWorkItemRecord(record); err != nil {
		return err
	}
	return WriteStoreJSONAtomic(StoreCampaignZohoEmailReplyWorkItemPath(root, record.JobID, record.ReplyWorkItemID), record)
}

func LoadCampaignZohoEmailReplyWorkItemRecord(root, jobID, replyWorkItemID string) (CampaignZohoEmailReplyWorkItemRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return CampaignZohoEmailReplyWorkItemRecord{}, err
	}
	var record CampaignZohoEmailReplyWorkItemRecord
	if err := LoadStoreJSON(StoreCampaignZohoEmailReplyWorkItemPath(root, jobID, replyWorkItemID), &record); err != nil {
		return CampaignZohoEmailReplyWorkItemRecord{}, err
	}
	if err := ValidateCampaignZohoEmailReplyWorkItemRecord(record); err != nil {
		return CampaignZohoEmailReplyWorkItemRecord{}, err
	}
	return record, nil
}

func ListCommittedCampaignZohoEmailReplyWorkItemRecords(root, jobID string) ([]CampaignZohoEmailReplyWorkItemRecord, error) {
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
	replyWorkItemIDs, err := listStoreRecordKeys(StoreCampaignZohoEmailReplyWorkItemsDir(root, jobID))
	if err != nil {
		return nil, err
	}
	records := make([]CampaignZohoEmailReplyWorkItemRecord, 0, len(replyWorkItemIDs))
	for _, replyWorkItemID := range replyWorkItemIDs {
		record, err := loadLatestVisibleVersionedJSONRecordAtOrBefore(
			storeCampaignZohoEmailReplyWorkItemVersionsDir(root, jobID, replyWorkItemID),
			jobRuntime.AppliedSeq,
			resolver,
			loadCampaignZohoEmailReplyWorkItemRecordFile,
			func(record CampaignZohoEmailReplyWorkItemRecord) uint64 { return record.LastSeq },
			func(record CampaignZohoEmailReplyWorkItemRecord) string { return record.AttemptID },
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
		if !records[i].CreatedAt.Equal(records[j].CreatedAt) {
			return records[i].CreatedAt.Before(records[j].CreatedAt)
		}
		return records[i].ReplyWorkItemID < records[j].ReplyWorkItemID
	})
	return records, nil
}

func ListCommittedAllCampaignZohoEmailReplyWorkItemRecords(root string) ([]CampaignZohoEmailReplyWorkItemRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(StoreJobsDir(root))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	byReplyWorkItemID := make(map[string]CampaignZohoEmailReplyWorkItemRecord)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		jobID := strings.TrimSpace(entry.Name())
		if jobID == "" {
			continue
		}
		records, err := ListCommittedCampaignZohoEmailReplyWorkItemRecords(root, jobID)
		if err != nil {
			return nil, err
		}
		for _, record := range records {
			existing, ok := byReplyWorkItemID[record.ReplyWorkItemID]
			if !ok || campaignZohoEmailReplyWorkItemRecordPreferred(record, existing) {
				byReplyWorkItemID[record.ReplyWorkItemID] = record
			}
		}
	}
	if len(byReplyWorkItemID) == 0 {
		return nil, nil
	}
	records := make([]CampaignZohoEmailReplyWorkItemRecord, 0, len(byReplyWorkItemID))
	for _, record := range byReplyWorkItemID {
		records = append(records, record)
	}
	sort.SliceStable(records, func(i, j int) bool {
		if !records[i].CreatedAt.Equal(records[j].CreatedAt) {
			return records[i].CreatedAt.Before(records[j].CreatedAt)
		}
		return records[i].ReplyWorkItemID < records[j].ReplyWorkItemID
	})
	return records, nil
}

func loadCampaignZohoEmailReplyWorkItemRecordFile(path string) (CampaignZohoEmailReplyWorkItemRecord, error) {
	var record CampaignZohoEmailReplyWorkItemRecord
	if err := LoadStoreJSON(path, &record); err != nil {
		return CampaignZohoEmailReplyWorkItemRecord{}, err
	}
	if err := ValidateCampaignZohoEmailReplyWorkItemRecord(record); err != nil {
		return CampaignZohoEmailReplyWorkItemRecord{}, err
	}
	return record, nil
}

func storeCampaignZohoEmailReplyWorkItemVersionsDir(root, jobID, replyWorkItemID string) string {
	return filepath.Join(StoreCampaignZohoEmailReplyWorkItemsDir(root, jobID), replyWorkItemID)
}

func storeCampaignZohoEmailReplyWorkItemVersionPath(root, jobID, replyWorkItemID string, seq uint64, attemptID string) string {
	return filepath.Join(storeCampaignZohoEmailReplyWorkItemVersionsDir(root, jobID, replyWorkItemID), storeVersionFilename(seq, attemptID))
}

func campaignZohoEmailReplyWorkItemRecordPreferred(candidate, existing CampaignZohoEmailReplyWorkItemRecord) bool {
	if !candidate.UpdatedAt.Equal(existing.UpdatedAt) {
		return candidate.UpdatedAt.After(existing.UpdatedAt)
	}
	if candidate.LastSeq != existing.LastSeq {
		return candidate.LastSeq > existing.LastSeq
	}
	leftPath := filepath.Join(strings.TrimSpace(candidate.JobID), strings.TrimSpace(candidate.ReplyWorkItemID))
	rightPath := filepath.Join(strings.TrimSpace(existing.JobID), strings.TrimSpace(existing.ReplyWorkItemID))
	return leftPath > rightPath
}
