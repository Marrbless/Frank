package missioncontrol

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"
	"unicode"
)

type RollbackRef struct {
	RollbackID string `json:"rollback_id"`
}

type RollbackRecord struct {
	RecordVersion       int       `json:"record_version"`
	RollbackID          string    `json:"rollback_id"`
	PromotionID         string    `json:"promotion_id,omitempty"`
	HotUpdateID         string    `json:"hot_update_id,omitempty"`
	OutcomeID           string    `json:"outcome_id,omitempty"`
	FromPackID          string    `json:"from_pack_id"`
	TargetPackID        string    `json:"target_pack_id"`
	LastKnownGoodPackID string    `json:"last_known_good_pack_id,omitempty"`
	Reason              string    `json:"reason"`
	Notes               string    `json:"notes,omitempty"`
	RollbackAt          time.Time `json:"rollback_at"`
	CreatedAt           time.Time `json:"created_at"`
	CreatedBy           string    `json:"created_by"`
}

var ErrRollbackRecordNotFound = errors.New("mission store rollback record not found")

func StoreRollbacksDir(root string) string {
	return filepath.Join(root, "runtime_packs", "rollbacks")
}

func StoreRollbackPath(root, rollbackID string) string {
	return filepath.Join(StoreRollbacksDir(root), strings.TrimSpace(rollbackID)+".json")
}

func NormalizeRollbackRef(ref RollbackRef) RollbackRef {
	ref.RollbackID = strings.TrimSpace(ref.RollbackID)
	return ref
}

func NormalizeRollbackRecord(record RollbackRecord) RollbackRecord {
	record.RollbackID = strings.TrimSpace(record.RollbackID)
	record.PromotionID = strings.TrimSpace(record.PromotionID)
	record.HotUpdateID = strings.TrimSpace(record.HotUpdateID)
	record.OutcomeID = strings.TrimSpace(record.OutcomeID)
	record.FromPackID = strings.TrimSpace(record.FromPackID)
	record.TargetPackID = strings.TrimSpace(record.TargetPackID)
	record.LastKnownGoodPackID = strings.TrimSpace(record.LastKnownGoodPackID)
	record.Reason = strings.TrimSpace(record.Reason)
	record.Notes = strings.TrimSpace(record.Notes)
	record.RollbackAt = record.RollbackAt.UTC()
	record.CreatedAt = record.CreatedAt.UTC()
	record.CreatedBy = strings.TrimSpace(record.CreatedBy)
	return record
}

func RollbackPromotionRef(record RollbackRecord) (PromotionRef, bool) {
	promotionID := strings.TrimSpace(record.PromotionID)
	if promotionID == "" {
		return PromotionRef{}, false
	}
	return PromotionRef{PromotionID: promotionID}, true
}

func RollbackHotUpdateGateRef(record RollbackRecord) (HotUpdateGateRef, bool) {
	hotUpdateID := strings.TrimSpace(record.HotUpdateID)
	if hotUpdateID == "" {
		return HotUpdateGateRef{}, false
	}
	return HotUpdateGateRef{HotUpdateID: hotUpdateID}, true
}

func RollbackHotUpdateOutcomeRef(record RollbackRecord) (HotUpdateOutcomeRef, bool) {
	outcomeID := strings.TrimSpace(record.OutcomeID)
	if outcomeID == "" {
		return HotUpdateOutcomeRef{}, false
	}
	return HotUpdateOutcomeRef{OutcomeID: outcomeID}, true
}

func RollbackFromPackRef(record RollbackRecord) RuntimePackRef {
	return RuntimePackRef{PackID: strings.TrimSpace(record.FromPackID)}
}

func RollbackTargetPackRef(record RollbackRecord) RuntimePackRef {
	return RuntimePackRef{PackID: strings.TrimSpace(record.TargetPackID)}
}

func RollbackLastKnownGoodPackRef(record RollbackRecord) (RuntimePackRef, bool) {
	packID := strings.TrimSpace(record.LastKnownGoodPackID)
	if packID == "" {
		return RuntimePackRef{}, false
	}
	return RuntimePackRef{PackID: packID}, true
}

func ValidateRollbackRef(ref RollbackRef) error {
	return validateRollbackIdentifierField("rollback ref", "rollback_id", ref.RollbackID)
}

func ValidateRollbackRecord(record RollbackRecord) error {
	if record.RecordVersion <= 0 {
		return fmt.Errorf("mission store rollback record_version must be positive")
	}
	if err := ValidateRollbackRef(RollbackRef{RollbackID: record.RollbackID}); err != nil {
		return err
	}
	if promotionRef, ok := RollbackPromotionRef(record); ok {
		if err := ValidatePromotionRef(promotionRef); err != nil {
			return fmt.Errorf("mission store rollback promotion_id %q: %w", record.PromotionID, err)
		}
	}
	if gateRef, ok := RollbackHotUpdateGateRef(record); ok {
		if err := ValidateHotUpdateGateRef(gateRef); err != nil {
			return fmt.Errorf("mission store rollback hot_update_id %q: %w", record.HotUpdateID, err)
		}
	}
	if outcomeRef, ok := RollbackHotUpdateOutcomeRef(record); ok {
		if err := ValidateHotUpdateOutcomeRef(outcomeRef); err != nil {
			return fmt.Errorf("mission store rollback outcome_id %q: %w", record.OutcomeID, err)
		}
	}
	if err := ValidateRuntimePackRef(RollbackFromPackRef(record)); err != nil {
		return fmt.Errorf("mission store rollback from_pack_id %q: %w", record.FromPackID, err)
	}
	if err := ValidateRuntimePackRef(RollbackTargetPackRef(record)); err != nil {
		return fmt.Errorf("mission store rollback target_pack_id %q: %w", record.TargetPackID, err)
	}
	if packRef, ok := RollbackLastKnownGoodPackRef(record); ok {
		if err := ValidateRuntimePackRef(packRef); err != nil {
			return fmt.Errorf("mission store rollback last_known_good_pack_id %q: %w", record.LastKnownGoodPackID, err)
		}
	}
	if record.Reason == "" {
		return fmt.Errorf("mission store rollback reason is required")
	}
	if record.RollbackAt.IsZero() {
		return fmt.Errorf("mission store rollback rollback_at is required")
	}
	if record.CreatedAt.IsZero() {
		return fmt.Errorf("mission store rollback created_at is required")
	}
	if record.CreatedBy == "" {
		return fmt.Errorf("mission store rollback created_by is required")
	}
	return nil
}

func StoreRollbackRecord(root string, record RollbackRecord) error {
	if err := ValidateStoreRoot(root); err != nil {
		return err
	}
	record = NormalizeRollbackRecord(record)
	record.RecordVersion = normalizeRecordVersion(record.RecordVersion)
	if err := ValidateRollbackRecord(record); err != nil {
		return err
	}
	if err := validateRollbackLinkage(root, record); err != nil {
		return err
	}

	path := StoreRollbackPath(root, record.RollbackID)
	if existing, err := loadRollbackRecordFile(root, path); err == nil {
		if reflect.DeepEqual(existing, record) {
			return nil
		}
		return fmt.Errorf("mission store rollback %q already exists", record.RollbackID)
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	return WriteStoreJSONAtomic(path, record)
}

func LoadRollbackRecord(root, rollbackID string) (RollbackRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return RollbackRecord{}, err
	}
	ref := NormalizeRollbackRef(RollbackRef{RollbackID: rollbackID})
	if err := ValidateRollbackRef(ref); err != nil {
		return RollbackRecord{}, err
	}
	record, err := loadRollbackRecordFile(root, StoreRollbackPath(root, ref.RollbackID))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return RollbackRecord{}, ErrRollbackRecordNotFound
		}
		return RollbackRecord{}, err
	}
	return record, nil
}

func ListRollbackRecords(root string) ([]RollbackRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return nil, err
	}
	return listStoreJSONRecords(StoreRollbacksDir(root), func(path string) (RollbackRecord, error) {
		return loadRollbackRecordFile(root, path)
	})
}

func loadRollbackRecordFile(root, path string) (RollbackRecord, error) {
	var record RollbackRecord
	if err := LoadStoreJSON(path, &record); err != nil {
		return RollbackRecord{}, err
	}
	record = NormalizeRollbackRecord(record)
	if err := ValidateRollbackRecord(record); err != nil {
		return RollbackRecord{}, err
	}
	if err := validateRollbackLinkage(root, record); err != nil {
		return RollbackRecord{}, err
	}
	return record, nil
}

func validateRollbackLinkage(root string, record RollbackRecord) error {
	fromRef := RollbackFromPackRef(record)
	if _, err := LoadRuntimePackRecord(root, fromRef.PackID); err != nil {
		return fmt.Errorf("mission store rollback from_pack_id %q: %w", fromRef.PackID, err)
	}

	targetRef := RollbackTargetPackRef(record)
	if _, err := LoadRuntimePackRecord(root, targetRef.PackID); err != nil {
		return fmt.Errorf("mission store rollback target_pack_id %q: %w", targetRef.PackID, err)
	}

	if packRef, ok := RollbackLastKnownGoodPackRef(record); ok {
		if _, err := LoadRuntimePackRecord(root, packRef.PackID); err != nil {
			return fmt.Errorf("mission store rollback last_known_good_pack_id %q: %w", packRef.PackID, err)
		}
	}

	if promotionRef, ok := RollbackPromotionRef(record); ok {
		promotion, err := LoadPromotionRecord(root, promotionRef.PromotionID)
		if err != nil {
			return fmt.Errorf("mission store rollback promotion_id %q: %w", promotionRef.PromotionID, err)
		}
		if promotion.PromotedPackID != fromRef.PackID {
			return fmt.Errorf(
				"mission store rollback promotion_id %q promoted_pack_id %q does not match from_pack_id %q",
				promotionRef.PromotionID,
				promotion.PromotedPackID,
				fromRef.PackID,
			)
		}
		if promotion.HotUpdateID != "" {
			if gateRef, ok := RollbackHotUpdateGateRef(record); ok && promotion.HotUpdateID != gateRef.HotUpdateID {
				return fmt.Errorf(
					"mission store rollback promotion_id %q hot_update_id %q does not match rollback hot_update_id %q",
					promotionRef.PromotionID,
					promotion.HotUpdateID,
					gateRef.HotUpdateID,
				)
			}
		}
		if packRef, ok := RollbackLastKnownGoodPackRef(record); ok && promotion.LastKnownGoodPackID != "" && promotion.LastKnownGoodPackID != packRef.PackID {
			return fmt.Errorf(
				"mission store rollback promotion_id %q last_known_good_pack_id %q does not match rollback last_known_good_pack_id %q",
				promotionRef.PromotionID,
				promotion.LastKnownGoodPackID,
				packRef.PackID,
			)
		}
		if targetRef.PackID != promotion.PreviousActivePackID && (promotion.LastKnownGoodPackID == "" || targetRef.PackID != promotion.LastKnownGoodPackID) {
			return fmt.Errorf(
				"mission store rollback target_pack_id %q does not match promotion previous_active_pack_id %q or last_known_good_pack_id %q",
				targetRef.PackID,
				promotion.PreviousActivePackID,
				promotion.LastKnownGoodPackID,
			)
		}
	}

	if gateRef, ok := RollbackHotUpdateGateRef(record); ok {
		gate, err := LoadHotUpdateGateRecord(root, gateRef.HotUpdateID)
		if err != nil {
			return fmt.Errorf("mission store rollback hot_update_id %q: %w", gateRef.HotUpdateID, err)
		}
		if gate.CandidatePackID != fromRef.PackID {
			return fmt.Errorf(
				"mission store rollback hot_update_id %q candidate_pack_id %q does not match from_pack_id %q",
				gateRef.HotUpdateID,
				gate.CandidatePackID,
				fromRef.PackID,
			)
		}
		if targetRef.PackID != gate.PreviousActivePackID && targetRef.PackID != gate.RollbackTargetPackID {
			return fmt.Errorf(
				"mission store rollback target_pack_id %q does not match hot-update gate previous_active_pack_id %q or rollback_target_pack_id %q",
				targetRef.PackID,
				gate.PreviousActivePackID,
				gate.RollbackTargetPackID,
			)
		}
	}

	if outcomeRef, ok := RollbackHotUpdateOutcomeRef(record); ok {
		outcome, err := LoadHotUpdateOutcomeRecord(root, outcomeRef.OutcomeID)
		if err != nil {
			return fmt.Errorf("mission store rollback outcome_id %q: %w", outcomeRef.OutcomeID, err)
		}
		if gateRef, ok := RollbackHotUpdateGateRef(record); ok && outcome.HotUpdateID != gateRef.HotUpdateID {
			return fmt.Errorf(
				"mission store rollback outcome_id %q hot_update_id %q does not match rollback hot_update_id %q",
				outcomeRef.OutcomeID,
				outcome.HotUpdateID,
				gateRef.HotUpdateID,
			)
		}
		if outcome.CandidatePackID != "" && outcome.CandidatePackID != fromRef.PackID {
			return fmt.Errorf(
				"mission store rollback outcome_id %q candidate_pack_id %q does not match from_pack_id %q",
				outcomeRef.OutcomeID,
				outcome.CandidatePackID,
				fromRef.PackID,
			)
		}
	}

	return nil
}

func validateRollbackIdentifierField(surface, fieldName, value string) error {
	normalized := strings.TrimSpace(value)
	if normalized == "" {
		return fmt.Errorf("%s %s is required", surface, fieldName)
	}
	if strings.HasPrefix(normalized, ".") || strings.HasSuffix(normalized, ".") {
		return fmt.Errorf("%s %s %q is invalid", surface, fieldName, normalized)
	}
	for _, r := range normalized {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			continue
		}
		switch r {
		case '-', '_', '.':
			continue
		default:
			return fmt.Errorf("%s %s %q is invalid", surface, fieldName, normalized)
		}
	}
	return nil
}
