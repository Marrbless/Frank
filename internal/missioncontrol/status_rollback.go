package missioncontrol

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type OperatorRollbackIdentityStatus struct {
	State     string                   `json:"state"`
	Rollbacks []OperatorRollbackStatus `json:"rollbacks,omitempty"`
}

type OperatorRollbackApplyIdentityStatus struct {
	State   string                        `json:"state"`
	Applies []OperatorRollbackApplyStatus `json:"applies,omitempty"`
}

type OperatorRollbackStatus struct {
	State               string  `json:"state"`
	RollbackID          string  `json:"rollback_id,omitempty"`
	PromotionID         string  `json:"promotion_id,omitempty"`
	HotUpdateID         string  `json:"hot_update_id,omitempty"`
	OutcomeID           string  `json:"outcome_id,omitempty"`
	FromPackID          string  `json:"from_pack_id,omitempty"`
	TargetPackID        string  `json:"target_pack_id,omitempty"`
	LastKnownGoodPackID string  `json:"last_known_good_pack_id,omitempty"`
	Reason              string  `json:"reason,omitempty"`
	Notes               string  `json:"notes,omitempty"`
	RollbackAt          *string `json:"rollback_at,omitempty"`
	CreatedAt           *string `json:"created_at,omitempty"`
	CreatedBy           string  `json:"created_by,omitempty"`
	Error               string  `json:"error,omitempty"`
}

type OperatorRollbackApplyStatus struct {
	State           string  `json:"state"`
	RollbackApplyID string  `json:"rollback_apply_id,omitempty"`
	RollbackID      string  `json:"rollback_id,omitempty"`
	Phase           string  `json:"phase,omitempty"`
	ActivationState string  `json:"activation_state,omitempty"`
	CreatedAt       *string `json:"created_at,omitempty"`
	CreatedBy       string  `json:"created_by,omitempty"`
	Error           string  `json:"error,omitempty"`
}

func WithRollbackIdentity(summary OperatorStatusSummary, root string) OperatorStatusSummary {
	root = strings.TrimSpace(root)
	if root == "" {
		return summary
	}
	status := LoadOperatorRollbackIdentityStatus(root)
	summary.RollbackIdentity = &status
	return summary
}

func WithRollbackApplyIdentity(summary OperatorStatusSummary, root string) OperatorStatusSummary {
	root = strings.TrimSpace(root)
	if root == "" {
		return summary
	}
	status := LoadOperatorRollbackApplyIdentityStatus(root)
	summary.RollbackApplyIdentity = &status
	return summary
}

func LoadOperatorRollbackIdentityStatus(root string) OperatorRollbackIdentityStatus {
	rollbacks, found, invalid, err := loadOperatorRollbackStatuses(root)
	if !found {
		return OperatorRollbackIdentityStatus{State: "not_configured"}
	}
	if err != nil {
		return OperatorRollbackIdentityStatus{State: "invalid"}
	}
	state := "configured"
	if invalid {
		state = "invalid"
	}
	return OperatorRollbackIdentityStatus{
		State:     state,
		Rollbacks: rollbacks,
	}
}

func LoadOperatorRollbackApplyIdentityStatus(root string) OperatorRollbackApplyIdentityStatus {
	applies, found, invalid, err := loadOperatorRollbackApplyStatuses(root)
	if !found {
		return OperatorRollbackApplyIdentityStatus{State: "not_configured"}
	}
	if err != nil {
		return OperatorRollbackApplyIdentityStatus{State: "invalid"}
	}
	state := "configured"
	if invalid {
		state = "invalid"
	}
	return OperatorRollbackApplyIdentityStatus{
		State:   state,
		Applies: applies,
	}
}

func loadOperatorRollbackStatuses(root string) ([]OperatorRollbackStatus, bool, bool, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return nil, true, false, err
	}

	entries, err := os.ReadDir(StoreRollbacksDir(root))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, false, false, nil
		}
		return nil, true, false, err
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !isStoreJSONDataFile(entry.Name()) {
			continue
		}
		names = append(names, entry.Name())
	}
	if len(names) == 0 {
		return nil, false, false, nil
	}
	sort.Strings(names)

	rollbacks := make([]OperatorRollbackStatus, 0, len(names))
	invalid := false
	for _, name := range names {
		status := loadOperatorRollbackStatus(root, filepath.Join(StoreRollbacksDir(root), name))
		if status.State == "invalid" {
			invalid = true
		}
		rollbacks = append(rollbacks, status)
	}
	return rollbacks, true, invalid, nil
}

func loadOperatorRollbackStatus(root, path string) OperatorRollbackStatus {
	status := OperatorRollbackStatus{
		State:      "invalid",
		RollbackID: strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)),
	}

	var record RollbackRecord
	if err := LoadStoreJSON(path, &record); err != nil {
		status.Error = err.Error()
		return status
	}

	record = NormalizeRollbackRecord(record)
	status = operatorRollbackStatusFromRecord(record)
	if err := ValidateRollbackRecord(record); err != nil {
		status.State = "invalid"
		status.Error = err.Error()
		return status
	}
	if err := validateRollbackLinkage(root, record); err != nil {
		status.State = "invalid"
		status.Error = err.Error()
		return status
	}
	status.State = "configured"
	return status
}

func loadOperatorRollbackApplyStatuses(root string) ([]OperatorRollbackApplyStatus, bool, bool, error) {
	entries, err := os.ReadDir(StoreRollbackAppliesDir(root))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, false, false, nil
		}
		return nil, true, false, err
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		names = append(names, entry.Name())
	}
	sort.Strings(names)

	applies := make([]OperatorRollbackApplyStatus, 0, len(names))
	invalid := false
	for _, name := range names {
		status := loadOperatorRollbackApplyStatus(root, filepath.Join(StoreRollbackAppliesDir(root), name))
		if status.State == "invalid" {
			invalid = true
		}
		applies = append(applies, status)
	}
	return applies, true, invalid, nil
}

func operatorRollbackStatusFromRecord(record RollbackRecord) OperatorRollbackStatus {
	return OperatorRollbackStatus{
		RollbackID:          record.RollbackID,
		PromotionID:         record.PromotionID,
		HotUpdateID:         record.HotUpdateID,
		OutcomeID:           record.OutcomeID,
		FromPackID:          record.FromPackID,
		TargetPackID:        record.TargetPackID,
		LastKnownGoodPackID: record.LastKnownGoodPackID,
		Reason:              record.Reason,
		Notes:               record.Notes,
		RollbackAt:          formatOperatorStatusTime(record.RollbackAt),
		CreatedAt:           formatOperatorStatusTime(record.CreatedAt),
		CreatedBy:           record.CreatedBy,
	}
}

func loadOperatorRollbackApplyStatus(root, path string) OperatorRollbackApplyStatus {
	status := OperatorRollbackApplyStatus{
		State:           "invalid",
		RollbackApplyID: strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)),
	}

	var record RollbackApplyRecord
	if err := LoadStoreJSON(path, &record); err != nil {
		status.Error = err.Error()
		return status
	}

	record = NormalizeRollbackApplyRecord(record)
	status = operatorRollbackApplyStatusFromRecord(record)
	if err := ValidateRollbackApplyRecord(record); err != nil {
		status.State = "invalid"
		status.Error = err.Error()
		return status
	}
	if err := validateRollbackApplyLinkage(root, record); err != nil {
		status.State = "invalid"
		status.Error = err.Error()
		return status
	}
	status.State = "configured"
	return status
}

func operatorRollbackApplyStatusFromRecord(record RollbackApplyRecord) OperatorRollbackApplyStatus {
	return OperatorRollbackApplyStatus{
		RollbackApplyID: record.ApplyID,
		RollbackID:      record.RollbackID,
		Phase:           string(record.Phase),
		ActivationState: string(record.ActivationState),
		CreatedAt:       formatOperatorStatusTime(record.CreatedAt),
		CreatedBy:       record.CreatedBy,
	}
}
