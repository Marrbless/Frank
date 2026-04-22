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

type HotUpdateOutcomeKind string

const (
	HotUpdateOutcomeKindKeptStaged        HotUpdateOutcomeKind = "kept_staged"
	HotUpdateOutcomeKindDiscarded         HotUpdateOutcomeKind = "discarded"
	HotUpdateOutcomeKindBlocked           HotUpdateOutcomeKind = "blocked"
	HotUpdateOutcomeKindApprovalRequired  HotUpdateOutcomeKind = "approval_required"
	HotUpdateOutcomeKindColdRestartNeeded HotUpdateOutcomeKind = "cold_restart_required"
	HotUpdateOutcomeKindHotUpdated        HotUpdateOutcomeKind = "hot_updated"
	HotUpdateOutcomeKindCanaryApplied     HotUpdateOutcomeKind = "canary_applied"
	HotUpdateOutcomeKindPromoted          HotUpdateOutcomeKind = "promoted"
	HotUpdateOutcomeKindRolledBack        HotUpdateOutcomeKind = "rolled_back"
	HotUpdateOutcomeKindFailed            HotUpdateOutcomeKind = "failed"
	HotUpdateOutcomeKindAborted           HotUpdateOutcomeKind = "aborted"
)

type HotUpdateOutcomeRef struct {
	OutcomeID string `json:"outcome_id"`
}

type HotUpdateOutcomeRecord struct {
	RecordVersion     int                  `json:"record_version"`
	OutcomeID         string               `json:"outcome_id"`
	HotUpdateID       string               `json:"hot_update_id"`
	CandidateID       string               `json:"candidate_id,omitempty"`
	RunID             string               `json:"run_id,omitempty"`
	CandidateResultID string               `json:"candidate_result_id,omitempty"`
	CandidatePackID   string               `json:"candidate_pack_id,omitempty"`
	OutcomeKind       HotUpdateOutcomeKind `json:"outcome_kind"`
	Reason            string               `json:"reason,omitempty"`
	Notes             string               `json:"notes,omitempty"`
	OutcomeAt         time.Time            `json:"outcome_at"`
	CreatedAt         time.Time            `json:"created_at"`
	CreatedBy         string               `json:"created_by"`
}

var ErrHotUpdateOutcomeRecordNotFound = errors.New("mission store hot-update outcome record not found")

func StoreHotUpdateOutcomesDir(root string) string {
	return filepath.Join(root, "runtime_packs", "hot_update_outcomes")
}

func StoreHotUpdateOutcomePath(root, outcomeID string) string {
	return filepath.Join(StoreHotUpdateOutcomesDir(root), strings.TrimSpace(outcomeID)+".json")
}

func NormalizeHotUpdateOutcomeRef(ref HotUpdateOutcomeRef) HotUpdateOutcomeRef {
	ref.OutcomeID = strings.TrimSpace(ref.OutcomeID)
	return ref
}

func NormalizeHotUpdateOutcomeRecord(record HotUpdateOutcomeRecord) HotUpdateOutcomeRecord {
	record.OutcomeID = strings.TrimSpace(record.OutcomeID)
	record.HotUpdateID = strings.TrimSpace(record.HotUpdateID)
	record.CandidateID = strings.TrimSpace(record.CandidateID)
	record.RunID = strings.TrimSpace(record.RunID)
	record.CandidateResultID = strings.TrimSpace(record.CandidateResultID)
	record.CandidatePackID = strings.TrimSpace(record.CandidatePackID)
	record.Reason = strings.TrimSpace(record.Reason)
	record.Notes = strings.TrimSpace(record.Notes)
	record.OutcomeAt = record.OutcomeAt.UTC()
	record.CreatedAt = record.CreatedAt.UTC()
	record.CreatedBy = strings.TrimSpace(record.CreatedBy)
	return record
}

func HotUpdateOutcomeGateRef(record HotUpdateOutcomeRecord) HotUpdateGateRef {
	return HotUpdateGateRef{HotUpdateID: strings.TrimSpace(record.HotUpdateID)}
}

func HotUpdateOutcomeImprovementCandidateRef(record HotUpdateOutcomeRecord) (ImprovementCandidateRef, bool) {
	candidateID := strings.TrimSpace(record.CandidateID)
	if candidateID == "" {
		return ImprovementCandidateRef{}, false
	}
	return ImprovementCandidateRef{CandidateID: candidateID}, true
}

func HotUpdateOutcomeImprovementRunRef(record HotUpdateOutcomeRecord) (ImprovementRunRef, bool) {
	runID := strings.TrimSpace(record.RunID)
	if runID == "" {
		return ImprovementRunRef{}, false
	}
	return ImprovementRunRef{RunID: runID}, true
}

func HotUpdateOutcomeCandidateResultRef(record HotUpdateOutcomeRecord) (CandidateResultRef, bool) {
	resultID := strings.TrimSpace(record.CandidateResultID)
	if resultID == "" {
		return CandidateResultRef{}, false
	}
	return CandidateResultRef{ResultID: resultID}, true
}

func HotUpdateOutcomeCandidatePackRef(record HotUpdateOutcomeRecord) (RuntimePackRef, bool) {
	packID := strings.TrimSpace(record.CandidatePackID)
	if packID == "" {
		return RuntimePackRef{}, false
	}
	return RuntimePackRef{PackID: packID}, true
}

func ValidateHotUpdateOutcomeRef(ref HotUpdateOutcomeRef) error {
	return validateHotUpdateOutcomeIdentifierField("hot-update outcome ref", "outcome_id", ref.OutcomeID)
}

func ValidateHotUpdateOutcomeRecord(record HotUpdateOutcomeRecord) error {
	if record.RecordVersion <= 0 {
		return fmt.Errorf("mission store hot-update outcome record_version must be positive")
	}
	if err := ValidateHotUpdateOutcomeRef(HotUpdateOutcomeRef{OutcomeID: record.OutcomeID}); err != nil {
		return err
	}
	if err := ValidateHotUpdateGateRef(HotUpdateOutcomeGateRef(record)); err != nil {
		return fmt.Errorf("mission store hot-update outcome hot_update_id %q: %w", record.HotUpdateID, err)
	}
	if candidateRef, ok := HotUpdateOutcomeImprovementCandidateRef(record); ok {
		if err := ValidateImprovementCandidateRef(candidateRef); err != nil {
			return fmt.Errorf("mission store hot-update outcome candidate_id %q: %w", record.CandidateID, err)
		}
	}
	if runRef, ok := HotUpdateOutcomeImprovementRunRef(record); ok {
		if err := ValidateImprovementRunRef(runRef); err != nil {
			return fmt.Errorf("mission store hot-update outcome run_id %q: %w", record.RunID, err)
		}
	}
	if resultRef, ok := HotUpdateOutcomeCandidateResultRef(record); ok {
		if err := ValidateCandidateResultRef(resultRef); err != nil {
			return fmt.Errorf("mission store hot-update outcome candidate_result_id %q: %w", record.CandidateResultID, err)
		}
	}
	if packRef, ok := HotUpdateOutcomeCandidatePackRef(record); ok {
		if err := ValidateRuntimePackRef(packRef); err != nil {
			return fmt.Errorf("mission store hot-update outcome candidate_pack_id %q: %w", record.CandidatePackID, err)
		}
	}
	if !isValidHotUpdateOutcomeKind(record.OutcomeKind) {
		return fmt.Errorf("mission store hot-update outcome outcome_kind %q is invalid", record.OutcomeKind)
	}
	if record.OutcomeAt.IsZero() {
		return fmt.Errorf("mission store hot-update outcome outcome_at is required")
	}
	if record.CreatedAt.IsZero() {
		return fmt.Errorf("mission store hot-update outcome created_at is required")
	}
	if record.CreatedBy == "" {
		return fmt.Errorf("mission store hot-update outcome created_by is required")
	}
	return nil
}

func StoreHotUpdateOutcomeRecord(root string, record HotUpdateOutcomeRecord) error {
	if err := ValidateStoreRoot(root); err != nil {
		return err
	}
	record = NormalizeHotUpdateOutcomeRecord(record)
	record.RecordVersion = normalizeRecordVersion(record.RecordVersion)
	if err := ValidateHotUpdateOutcomeRecord(record); err != nil {
		return err
	}
	if err := validateHotUpdateOutcomeLinkage(root, record); err != nil {
		return err
	}

	path := StoreHotUpdateOutcomePath(root, record.OutcomeID)
	if existing, err := loadHotUpdateOutcomeRecordFile(root, path); err == nil {
		if reflect.DeepEqual(existing, record) {
			return nil
		}
		return fmt.Errorf("mission store hot-update outcome %q already exists", record.OutcomeID)
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	return WriteStoreJSONAtomic(path, record)
}

func LoadHotUpdateOutcomeRecord(root, outcomeID string) (HotUpdateOutcomeRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return HotUpdateOutcomeRecord{}, err
	}
	ref := NormalizeHotUpdateOutcomeRef(HotUpdateOutcomeRef{OutcomeID: outcomeID})
	if err := ValidateHotUpdateOutcomeRef(ref); err != nil {
		return HotUpdateOutcomeRecord{}, err
	}
	record, err := loadHotUpdateOutcomeRecordFile(root, StoreHotUpdateOutcomePath(root, ref.OutcomeID))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return HotUpdateOutcomeRecord{}, ErrHotUpdateOutcomeRecordNotFound
		}
		return HotUpdateOutcomeRecord{}, err
	}
	return record, nil
}

func ListHotUpdateOutcomeRecords(root string) ([]HotUpdateOutcomeRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return nil, err
	}
	return listStoreJSONRecords(StoreHotUpdateOutcomesDir(root), func(path string) (HotUpdateOutcomeRecord, error) {
		return loadHotUpdateOutcomeRecordFile(root, path)
	})
}

func loadHotUpdateOutcomeRecordFile(root, path string) (HotUpdateOutcomeRecord, error) {
	var record HotUpdateOutcomeRecord
	if err := LoadStoreJSON(path, &record); err != nil {
		return HotUpdateOutcomeRecord{}, err
	}
	record = NormalizeHotUpdateOutcomeRecord(record)
	if err := ValidateHotUpdateOutcomeRecord(record); err != nil {
		return HotUpdateOutcomeRecord{}, err
	}
	if err := validateHotUpdateOutcomeLinkage(root, record); err != nil {
		return HotUpdateOutcomeRecord{}, err
	}
	return record, nil
}

func validateHotUpdateOutcomeLinkage(root string, record HotUpdateOutcomeRecord) error {
	gate := HotUpdateOutcomeGateRef(record)
	gateRecord, err := LoadHotUpdateGateRecord(root, gate.HotUpdateID)
	if err != nil {
		return fmt.Errorf("mission store hot-update outcome hot_update_id %q: %w", gate.HotUpdateID, err)
	}

	if packRef, ok := HotUpdateOutcomeCandidatePackRef(record); ok {
		if _, err := LoadRuntimePackRecord(root, packRef.PackID); err != nil {
			return fmt.Errorf("mission store hot-update outcome candidate_pack_id %q: %w", packRef.PackID, err)
		}
		if gateRecord.CandidatePackID != packRef.PackID {
			return fmt.Errorf(
				"mission store hot-update outcome candidate_pack_id %q does not match hot-update gate candidate_pack_id %q",
				packRef.PackID,
				gateRecord.CandidatePackID,
			)
		}
	}

	if candidateRef, ok := HotUpdateOutcomeImprovementCandidateRef(record); ok {
		candidate, err := LoadImprovementCandidateRecord(root, candidateRef.CandidateID)
		if err != nil {
			return fmt.Errorf("mission store hot-update outcome candidate_id %q: %w", candidateRef.CandidateID, err)
		}
		if candidate.CandidatePackID != gateRecord.CandidatePackID {
			return fmt.Errorf(
				"mission store hot-update outcome candidate_id %q candidate_pack_id %q does not match hot-update gate candidate_pack_id %q",
				candidateRef.CandidateID,
				candidate.CandidatePackID,
				gateRecord.CandidatePackID,
			)
		}
		if candidate.HotUpdateID != "" && candidate.HotUpdateID != gate.HotUpdateID {
			return fmt.Errorf(
				"mission store hot-update outcome candidate_id %q hot_update_id %q does not match hot-update outcome hot_update_id %q",
				candidateRef.CandidateID,
				candidate.HotUpdateID,
				gate.HotUpdateID,
			)
		}
		if packRef, ok := HotUpdateOutcomeCandidatePackRef(record); ok && candidate.CandidatePackID != packRef.PackID {
			return fmt.Errorf(
				"mission store hot-update outcome candidate_id %q candidate_pack_id %q does not match outcome candidate_pack_id %q",
				candidateRef.CandidateID,
				candidate.CandidatePackID,
				packRef.PackID,
			)
		}
	}

	if runRef, ok := HotUpdateOutcomeImprovementRunRef(record); ok {
		run, err := LoadImprovementRunRecord(root, runRef.RunID)
		if err != nil {
			return fmt.Errorf("mission store hot-update outcome run_id %q: %w", runRef.RunID, err)
		}
		if run.HotUpdateID == "" {
			return fmt.Errorf("mission store hot-update outcome run_id %q requires run hot_update_id", runRef.RunID)
		}
		if run.HotUpdateID != gate.HotUpdateID {
			return fmt.Errorf(
				"mission store hot-update outcome run_id %q hot_update_id %q does not match hot-update outcome hot_update_id %q",
				runRef.RunID,
				run.HotUpdateID,
				gate.HotUpdateID,
			)
		}
		if run.CandidatePackID != gateRecord.CandidatePackID {
			return fmt.Errorf(
				"mission store hot-update outcome run_id %q candidate_pack_id %q does not match hot-update gate candidate_pack_id %q",
				runRef.RunID,
				run.CandidatePackID,
				gateRecord.CandidatePackID,
			)
		}
		if candidateRef, ok := HotUpdateOutcomeImprovementCandidateRef(record); ok && run.CandidateID != candidateRef.CandidateID {
			return fmt.Errorf(
				"mission store hot-update outcome run_id %q candidate_id %q does not match outcome candidate_id %q",
				runRef.RunID,
				run.CandidateID,
				candidateRef.CandidateID,
			)
		}
		if packRef, ok := HotUpdateOutcomeCandidatePackRef(record); ok && run.CandidatePackID != packRef.PackID {
			return fmt.Errorf(
				"mission store hot-update outcome run_id %q candidate_pack_id %q does not match outcome candidate_pack_id %q",
				runRef.RunID,
				run.CandidatePackID,
				packRef.PackID,
			)
		}
	}

	if resultRef, ok := HotUpdateOutcomeCandidateResultRef(record); ok {
		result, err := LoadCandidateResultRecord(root, resultRef.ResultID)
		if err != nil {
			return fmt.Errorf("mission store hot-update outcome candidate_result_id %q: %w", resultRef.ResultID, err)
		}
		if result.HotUpdateID == "" {
			return fmt.Errorf("mission store hot-update outcome candidate_result_id %q requires candidate result hot_update_id", resultRef.ResultID)
		}
		if result.HotUpdateID != gate.HotUpdateID {
			return fmt.Errorf(
				"mission store hot-update outcome candidate_result_id %q hot_update_id %q does not match hot-update outcome hot_update_id %q",
				resultRef.ResultID,
				result.HotUpdateID,
				gate.HotUpdateID,
			)
		}
		if result.CandidatePackID != gateRecord.CandidatePackID {
			return fmt.Errorf(
				"mission store hot-update outcome candidate_result_id %q candidate_pack_id %q does not match hot-update gate candidate_pack_id %q",
				resultRef.ResultID,
				result.CandidatePackID,
				gateRecord.CandidatePackID,
			)
		}
		if candidateRef, ok := HotUpdateOutcomeImprovementCandidateRef(record); ok && result.CandidateID != candidateRef.CandidateID {
			return fmt.Errorf(
				"mission store hot-update outcome candidate_result_id %q candidate_id %q does not match outcome candidate_id %q",
				resultRef.ResultID,
				result.CandidateID,
				candidateRef.CandidateID,
			)
		}
		if runRef, ok := HotUpdateOutcomeImprovementRunRef(record); ok && result.RunID != runRef.RunID {
			return fmt.Errorf(
				"mission store hot-update outcome candidate_result_id %q run_id %q does not match outcome run_id %q",
				resultRef.ResultID,
				result.RunID,
				runRef.RunID,
			)
		}
		if packRef, ok := HotUpdateOutcomeCandidatePackRef(record); ok && result.CandidatePackID != packRef.PackID {
			return fmt.Errorf(
				"mission store hot-update outcome candidate_result_id %q candidate_pack_id %q does not match outcome candidate_pack_id %q",
				resultRef.ResultID,
				result.CandidatePackID,
				packRef.PackID,
			)
		}
	}

	return nil
}

func validateHotUpdateOutcomeIdentifierField(surface, fieldName, value string) error {
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

func isValidHotUpdateOutcomeKind(kind HotUpdateOutcomeKind) bool {
	switch kind {
	case HotUpdateOutcomeKindKeptStaged,
		HotUpdateOutcomeKindDiscarded,
		HotUpdateOutcomeKindBlocked,
		HotUpdateOutcomeKindApprovalRequired,
		HotUpdateOutcomeKindColdRestartNeeded,
		HotUpdateOutcomeKindHotUpdated,
		HotUpdateOutcomeKindCanaryApplied,
		HotUpdateOutcomeKindPromoted,
		HotUpdateOutcomeKindRolledBack,
		HotUpdateOutcomeKindFailed,
		HotUpdateOutcomeKindAborted:
		return true
	default:
		return false
	}
}
