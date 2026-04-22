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

type PromotionRef struct {
	PromotionID string `json:"promotion_id"`
}

type PromotionRecord struct {
	RecordVersion        int       `json:"record_version"`
	PromotionID          string    `json:"promotion_id"`
	PromotedPackID       string    `json:"promoted_pack_id"`
	PreviousActivePackID string    `json:"previous_active_pack_id"`
	LastKnownGoodPackID  string    `json:"last_known_good_pack_id,omitempty"`
	LastKnownGoodBasis   string    `json:"last_known_good_basis,omitempty"`
	HotUpdateID          string    `json:"hot_update_id"`
	OutcomeID            string    `json:"outcome_id,omitempty"`
	CandidateID          string    `json:"candidate_id,omitempty"`
	RunID                string    `json:"run_id,omitempty"`
	CandidateResultID    string    `json:"candidate_result_id,omitempty"`
	Reason               string    `json:"reason"`
	Notes                string    `json:"notes,omitempty"`
	PromotedAt           time.Time `json:"promoted_at"`
	CreatedAt            time.Time `json:"created_at"`
	CreatedBy            string    `json:"created_by"`
}

var ErrPromotionRecordNotFound = errors.New("mission store promotion record not found")

func StorePromotionsDir(root string) string {
	return filepath.Join(root, "runtime_packs", "promotions")
}

func StorePromotionPath(root, promotionID string) string {
	return filepath.Join(StorePromotionsDir(root), strings.TrimSpace(promotionID)+".json")
}

func NormalizePromotionRef(ref PromotionRef) PromotionRef {
	ref.PromotionID = strings.TrimSpace(ref.PromotionID)
	return ref
}

func NormalizePromotionRecord(record PromotionRecord) PromotionRecord {
	record.PromotionID = strings.TrimSpace(record.PromotionID)
	record.PromotedPackID = strings.TrimSpace(record.PromotedPackID)
	record.PreviousActivePackID = strings.TrimSpace(record.PreviousActivePackID)
	record.LastKnownGoodPackID = strings.TrimSpace(record.LastKnownGoodPackID)
	record.LastKnownGoodBasis = strings.TrimSpace(record.LastKnownGoodBasis)
	record.HotUpdateID = strings.TrimSpace(record.HotUpdateID)
	record.OutcomeID = strings.TrimSpace(record.OutcomeID)
	record.CandidateID = strings.TrimSpace(record.CandidateID)
	record.RunID = strings.TrimSpace(record.RunID)
	record.CandidateResultID = strings.TrimSpace(record.CandidateResultID)
	record.Reason = strings.TrimSpace(record.Reason)
	record.Notes = strings.TrimSpace(record.Notes)
	record.PromotedAt = record.PromotedAt.UTC()
	record.CreatedAt = record.CreatedAt.UTC()
	record.CreatedBy = strings.TrimSpace(record.CreatedBy)
	return record
}

func PromotionPromotedPackRef(record PromotionRecord) RuntimePackRef {
	return RuntimePackRef{PackID: strings.TrimSpace(record.PromotedPackID)}
}

func PromotionPreviousActivePackRef(record PromotionRecord) RuntimePackRef {
	return RuntimePackRef{PackID: strings.TrimSpace(record.PreviousActivePackID)}
}

func PromotionLastKnownGoodPackRef(record PromotionRecord) (RuntimePackRef, bool) {
	packID := strings.TrimSpace(record.LastKnownGoodPackID)
	if packID == "" {
		return RuntimePackRef{}, false
	}
	return RuntimePackRef{PackID: packID}, true
}

func PromotionHotUpdateGateRef(record PromotionRecord) HotUpdateGateRef {
	return HotUpdateGateRef{HotUpdateID: strings.TrimSpace(record.HotUpdateID)}
}

func PromotionHotUpdateOutcomeRef(record PromotionRecord) (HotUpdateOutcomeRef, bool) {
	outcomeID := strings.TrimSpace(record.OutcomeID)
	if outcomeID == "" {
		return HotUpdateOutcomeRef{}, false
	}
	return HotUpdateOutcomeRef{OutcomeID: outcomeID}, true
}

func PromotionImprovementCandidateRef(record PromotionRecord) (ImprovementCandidateRef, bool) {
	candidateID := strings.TrimSpace(record.CandidateID)
	if candidateID == "" {
		return ImprovementCandidateRef{}, false
	}
	return ImprovementCandidateRef{CandidateID: candidateID}, true
}

func PromotionImprovementRunRef(record PromotionRecord) (ImprovementRunRef, bool) {
	runID := strings.TrimSpace(record.RunID)
	if runID == "" {
		return ImprovementRunRef{}, false
	}
	return ImprovementRunRef{RunID: runID}, true
}

func PromotionCandidateResultRef(record PromotionRecord) (CandidateResultRef, bool) {
	resultID := strings.TrimSpace(record.CandidateResultID)
	if resultID == "" {
		return CandidateResultRef{}, false
	}
	return CandidateResultRef{ResultID: resultID}, true
}

func ValidatePromotionRef(ref PromotionRef) error {
	return validatePromotionIdentifierField("promotion ref", "promotion_id", ref.PromotionID)
}

func ValidatePromotionRecord(record PromotionRecord) error {
	if record.RecordVersion <= 0 {
		return fmt.Errorf("mission store promotion record_version must be positive")
	}
	if err := ValidatePromotionRef(PromotionRef{PromotionID: record.PromotionID}); err != nil {
		return err
	}
	if err := ValidateRuntimePackRef(PromotionPromotedPackRef(record)); err != nil {
		return fmt.Errorf("mission store promotion promoted_pack_id %q: %w", record.PromotedPackID, err)
	}
	if err := ValidateRuntimePackRef(PromotionPreviousActivePackRef(record)); err != nil {
		return fmt.Errorf("mission store promotion previous_active_pack_id %q: %w", record.PreviousActivePackID, err)
	}
	if packRef, ok := PromotionLastKnownGoodPackRef(record); ok {
		if err := ValidateRuntimePackRef(packRef); err != nil {
			return fmt.Errorf("mission store promotion last_known_good_pack_id %q: %w", record.LastKnownGoodPackID, err)
		}
		if record.LastKnownGoodBasis == "" {
			return fmt.Errorf("mission store promotion last_known_good_basis is required with last_known_good_pack_id")
		}
	} else if record.LastKnownGoodBasis != "" {
		return fmt.Errorf("mission store promotion last_known_good_pack_id is required with last_known_good_basis")
	}
	if err := ValidateHotUpdateGateRef(PromotionHotUpdateGateRef(record)); err != nil {
		return fmt.Errorf("mission store promotion hot_update_id %q: %w", record.HotUpdateID, err)
	}
	if outcomeRef, ok := PromotionHotUpdateOutcomeRef(record); ok {
		if err := ValidateHotUpdateOutcomeRef(outcomeRef); err != nil {
			return fmt.Errorf("mission store promotion outcome_id %q: %w", record.OutcomeID, err)
		}
	}
	if candidateRef, ok := PromotionImprovementCandidateRef(record); ok {
		if err := ValidateImprovementCandidateRef(candidateRef); err != nil {
			return fmt.Errorf("mission store promotion candidate_id %q: %w", record.CandidateID, err)
		}
	}
	if runRef, ok := PromotionImprovementRunRef(record); ok {
		if err := ValidateImprovementRunRef(runRef); err != nil {
			return fmt.Errorf("mission store promotion run_id %q: %w", record.RunID, err)
		}
	}
	if resultRef, ok := PromotionCandidateResultRef(record); ok {
		if err := ValidateCandidateResultRef(resultRef); err != nil {
			return fmt.Errorf("mission store promotion candidate_result_id %q: %w", record.CandidateResultID, err)
		}
	}
	if record.Reason == "" {
		return fmt.Errorf("mission store promotion reason is required")
	}
	if record.PromotedAt.IsZero() {
		return fmt.Errorf("mission store promotion promoted_at is required")
	}
	if record.CreatedAt.IsZero() {
		return fmt.Errorf("mission store promotion created_at is required")
	}
	if record.CreatedBy == "" {
		return fmt.Errorf("mission store promotion created_by is required")
	}
	return nil
}

func StorePromotionRecord(root string, record PromotionRecord) error {
	if err := ValidateStoreRoot(root); err != nil {
		return err
	}
	record = NormalizePromotionRecord(record)
	record.RecordVersion = normalizeRecordVersion(record.RecordVersion)
	if err := ValidatePromotionRecord(record); err != nil {
		return err
	}
	if err := validatePromotionLinkage(root, record); err != nil {
		return err
	}

	path := StorePromotionPath(root, record.PromotionID)
	if existing, err := loadPromotionRecordFile(root, path); err == nil {
		if reflect.DeepEqual(existing, record) {
			return nil
		}
		return fmt.Errorf("mission store promotion %q already exists", record.PromotionID)
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	return WriteStoreJSONAtomic(path, record)
}

func LoadPromotionRecord(root, promotionID string) (PromotionRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return PromotionRecord{}, err
	}
	ref := NormalizePromotionRef(PromotionRef{PromotionID: promotionID})
	if err := ValidatePromotionRef(ref); err != nil {
		return PromotionRecord{}, err
	}
	record, err := loadPromotionRecordFile(root, StorePromotionPath(root, ref.PromotionID))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return PromotionRecord{}, ErrPromotionRecordNotFound
		}
		return PromotionRecord{}, err
	}
	return record, nil
}

func ListPromotionRecords(root string) ([]PromotionRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return nil, err
	}
	return listStoreJSONRecords(StorePromotionsDir(root), func(path string) (PromotionRecord, error) {
		return loadPromotionRecordFile(root, path)
	})
}

func loadPromotionRecordFile(root, path string) (PromotionRecord, error) {
	var record PromotionRecord
	if err := LoadStoreJSON(path, &record); err != nil {
		return PromotionRecord{}, err
	}
	record = NormalizePromotionRecord(record)
	if err := ValidatePromotionRecord(record); err != nil {
		return PromotionRecord{}, err
	}
	if err := validatePromotionLinkage(root, record); err != nil {
		return PromotionRecord{}, err
	}
	return record, nil
}

func validatePromotionLinkage(root string, record PromotionRecord) error {
	promotedRef := PromotionPromotedPackRef(record)
	if _, err := LoadRuntimePackRecord(root, promotedRef.PackID); err != nil {
		return fmt.Errorf("mission store promotion promoted_pack_id %q: %w", promotedRef.PackID, err)
	}

	previousRef := PromotionPreviousActivePackRef(record)
	if _, err := LoadRuntimePackRecord(root, previousRef.PackID); err != nil {
		return fmt.Errorf("mission store promotion previous_active_pack_id %q: %w", previousRef.PackID, err)
	}

	if packRef, ok := PromotionLastKnownGoodPackRef(record); ok {
		if _, err := LoadRuntimePackRecord(root, packRef.PackID); err != nil {
			return fmt.Errorf("mission store promotion last_known_good_pack_id %q: %w", packRef.PackID, err)
		}
	}

	gateRef := PromotionHotUpdateGateRef(record)
	gate, err := LoadHotUpdateGateRecord(root, gateRef.HotUpdateID)
	if err != nil {
		return fmt.Errorf("mission store promotion hot_update_id %q: %w", gateRef.HotUpdateID, err)
	}
	if gate.CandidatePackID != promotedRef.PackID {
		return fmt.Errorf(
			"mission store promotion promoted_pack_id %q does not match hot-update gate candidate_pack_id %q",
			promotedRef.PackID,
			gate.CandidatePackID,
		)
	}
	if gate.PreviousActivePackID != previousRef.PackID {
		return fmt.Errorf(
			"mission store promotion previous_active_pack_id %q does not match hot-update gate previous_active_pack_id %q",
			previousRef.PackID,
			gate.PreviousActivePackID,
		)
	}

	if outcomeRef, ok := PromotionHotUpdateOutcomeRef(record); ok {
		outcome, err := LoadHotUpdateOutcomeRecord(root, outcomeRef.OutcomeID)
		if err != nil {
			return fmt.Errorf("mission store promotion outcome_id %q: %w", outcomeRef.OutcomeID, err)
		}
		if outcome.HotUpdateID != gateRef.HotUpdateID {
			return fmt.Errorf(
				"mission store promotion outcome_id %q hot_update_id %q does not match promotion hot_update_id %q",
				outcomeRef.OutcomeID,
				outcome.HotUpdateID,
				gateRef.HotUpdateID,
			)
		}
		if outcome.CandidatePackID != "" && outcome.CandidatePackID != promotedRef.PackID {
			return fmt.Errorf(
				"mission store promotion outcome_id %q candidate_pack_id %q does not match promoted_pack_id %q",
				outcomeRef.OutcomeID,
				outcome.CandidatePackID,
				promotedRef.PackID,
			)
		}
		if candidateRef, ok := PromotionImprovementCandidateRef(record); ok && outcome.CandidateID != "" && outcome.CandidateID != candidateRef.CandidateID {
			return fmt.Errorf(
				"mission store promotion outcome_id %q candidate_id %q does not match promotion candidate_id %q",
				outcomeRef.OutcomeID,
				outcome.CandidateID,
				candidateRef.CandidateID,
			)
		}
		if runRef, ok := PromotionImprovementRunRef(record); ok && outcome.RunID != "" && outcome.RunID != runRef.RunID {
			return fmt.Errorf(
				"mission store promotion outcome_id %q run_id %q does not match promotion run_id %q",
				outcomeRef.OutcomeID,
				outcome.RunID,
				runRef.RunID,
			)
		}
		if resultRef, ok := PromotionCandidateResultRef(record); ok && outcome.CandidateResultID != "" && outcome.CandidateResultID != resultRef.ResultID {
			return fmt.Errorf(
				"mission store promotion outcome_id %q candidate_result_id %q does not match promotion candidate_result_id %q",
				outcomeRef.OutcomeID,
				outcome.CandidateResultID,
				resultRef.ResultID,
			)
		}
	}

	if candidateRef, ok := PromotionImprovementCandidateRef(record); ok {
		candidate, err := LoadImprovementCandidateRecord(root, candidateRef.CandidateID)
		if err != nil {
			return fmt.Errorf("mission store promotion candidate_id %q: %w", candidateRef.CandidateID, err)
		}
		if candidate.BaselinePackID != previousRef.PackID {
			return fmt.Errorf(
				"mission store promotion candidate_id %q baseline_pack_id %q does not match previous_active_pack_id %q",
				candidateRef.CandidateID,
				candidate.BaselinePackID,
				previousRef.PackID,
			)
		}
		if candidate.CandidatePackID != promotedRef.PackID {
			return fmt.Errorf(
				"mission store promotion candidate_id %q candidate_pack_id %q does not match promoted_pack_id %q",
				candidateRef.CandidateID,
				candidate.CandidatePackID,
				promotedRef.PackID,
			)
		}
		if candidate.HotUpdateID != "" && candidate.HotUpdateID != gateRef.HotUpdateID {
			return fmt.Errorf(
				"mission store promotion candidate_id %q hot_update_id %q does not match promotion hot_update_id %q",
				candidateRef.CandidateID,
				candidate.HotUpdateID,
				gateRef.HotUpdateID,
			)
		}
	}

	if runRef, ok := PromotionImprovementRunRef(record); ok {
		run, err := LoadImprovementRunRecord(root, runRef.RunID)
		if err != nil {
			return fmt.Errorf("mission store promotion run_id %q: %w", runRef.RunID, err)
		}
		if run.HotUpdateID == "" {
			return fmt.Errorf("mission store promotion run_id %q requires run hot_update_id", runRef.RunID)
		}
		if run.HotUpdateID != gateRef.HotUpdateID {
			return fmt.Errorf(
				"mission store promotion run_id %q hot_update_id %q does not match promotion hot_update_id %q",
				runRef.RunID,
				run.HotUpdateID,
				gateRef.HotUpdateID,
			)
		}
		if run.BaselinePackID != previousRef.PackID {
			return fmt.Errorf(
				"mission store promotion run_id %q baseline_pack_id %q does not match previous_active_pack_id %q",
				runRef.RunID,
				run.BaselinePackID,
				previousRef.PackID,
			)
		}
		if run.CandidatePackID != promotedRef.PackID {
			return fmt.Errorf(
				"mission store promotion run_id %q candidate_pack_id %q does not match promoted_pack_id %q",
				runRef.RunID,
				run.CandidatePackID,
				promotedRef.PackID,
			)
		}
		if candidateRef, ok := PromotionImprovementCandidateRef(record); ok && run.CandidateID != candidateRef.CandidateID {
			return fmt.Errorf(
				"mission store promotion run_id %q candidate_id %q does not match promotion candidate_id %q",
				runRef.RunID,
				run.CandidateID,
				candidateRef.CandidateID,
			)
		}
	}

	if resultRef, ok := PromotionCandidateResultRef(record); ok {
		result, err := LoadCandidateResultRecord(root, resultRef.ResultID)
		if err != nil {
			return fmt.Errorf("mission store promotion candidate_result_id %q: %w", resultRef.ResultID, err)
		}
		if result.HotUpdateID == "" {
			return fmt.Errorf("mission store promotion candidate_result_id %q requires candidate result hot_update_id", resultRef.ResultID)
		}
		if result.HotUpdateID != gateRef.HotUpdateID {
			return fmt.Errorf(
				"mission store promotion candidate_result_id %q hot_update_id %q does not match promotion hot_update_id %q",
				resultRef.ResultID,
				result.HotUpdateID,
				gateRef.HotUpdateID,
			)
		}
		if result.BaselinePackID != previousRef.PackID {
			return fmt.Errorf(
				"mission store promotion candidate_result_id %q baseline_pack_id %q does not match previous_active_pack_id %q",
				resultRef.ResultID,
				result.BaselinePackID,
				previousRef.PackID,
			)
		}
		if result.CandidatePackID != promotedRef.PackID {
			return fmt.Errorf(
				"mission store promotion candidate_result_id %q candidate_pack_id %q does not match promoted_pack_id %q",
				resultRef.ResultID,
				result.CandidatePackID,
				promotedRef.PackID,
			)
		}
		if candidateRef, ok := PromotionImprovementCandidateRef(record); ok && result.CandidateID != candidateRef.CandidateID {
			return fmt.Errorf(
				"mission store promotion candidate_result_id %q candidate_id %q does not match promotion candidate_id %q",
				resultRef.ResultID,
				result.CandidateID,
				candidateRef.CandidateID,
			)
		}
		if runRef, ok := PromotionImprovementRunRef(record); ok && result.RunID != runRef.RunID {
			return fmt.Errorf(
				"mission store promotion candidate_result_id %q run_id %q does not match promotion run_id %q",
				resultRef.ResultID,
				result.RunID,
				runRef.RunID,
			)
		}
	}

	return nil
}

func validatePromotionIdentifierField(surface, fieldName, value string) error {
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
