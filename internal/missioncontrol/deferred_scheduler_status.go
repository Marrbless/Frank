package missioncontrol

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type deferredScheduledTriggerStoreRecord struct {
	RecordVersion  int       `json:"record_version"`
	TriggerID      string    `json:"trigger_id"`
	SchedulerJobID string    `json:"scheduler_job_id"`
	Name           string    `json:"name,omitempty"`
	Message        string    `json:"message,omitempty"`
	Channel        string    `json:"channel,omitempty"`
	ChatID         string    `json:"chat_id,omitempty"`
	FireAt         time.Time `json:"fire_at"`
	DeferredAt     time.Time `json:"deferred_at"`
}

func deferredSchedulerTriggersDir(root string) string {
	return filepath.Join(root, "scheduler", "deferred_triggers")
}

func LoadDeferredSchedulerTriggerStatuses(root string) ([]OperatorDeferredSchedulerTriggerStatus, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return nil, nil
	}

	entries, err := os.ReadDir(deferredSchedulerTriggersDir(root))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	records := make([]deferredScheduledTriggerStoreRecord, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		var record deferredScheduledTriggerStoreRecord
		if err := LoadStoreJSON(filepath.Join(deferredSchedulerTriggersDir(root), entry.Name()), &record); err != nil {
			return nil, err
		}
		records = append(records, record)
	}

	sort.Slice(records, func(i, j int) bool {
		leftFireAt := records[i].FireAt.UTC()
		rightFireAt := records[j].FireAt.UTC()
		if !leftFireAt.Equal(rightFireAt) {
			return leftFireAt.Before(rightFireAt)
		}
		return records[i].TriggerID < records[j].TriggerID
	})

	statuses := make([]OperatorDeferredSchedulerTriggerStatus, len(records))
	for i, record := range records {
		statuses[i] = OperatorDeferredSchedulerTriggerStatus{
			TriggerID:      strings.TrimSpace(record.TriggerID),
			SchedulerJobID: strings.TrimSpace(record.SchedulerJobID),
			Name:           strings.TrimSpace(record.Name),
			Message:        strings.TrimSpace(record.Message),
			FireAt:         record.FireAt.UTC().Format(time.RFC3339Nano),
			DeferredAt:     record.DeferredAt.UTC().Format(time.RFC3339Nano),
		}
	}

	return statuses, nil
}
