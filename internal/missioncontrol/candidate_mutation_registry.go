package missioncontrol

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"
)

type CandidateMutationRef struct {
	MutationID string `json:"mutation_id"`
}

type CandidateMutationRecord struct {
	RecordVersion       int       `json:"record_version"`
	MutationID          string    `json:"mutation_id"`
	RunID               string    `json:"run_id"`
	CandidateID         string    `json:"candidate_id"`
	EvalSuiteID         string    `json:"eval_suite_id"`
	BaselinePackID      string    `json:"baseline_pack_id"`
	CandidatePackID     string    `json:"candidate_pack_id"`
	BaselineResultRef   string    `json:"baseline_result_ref"`
	BaselineCapturedAt  time.Time `json:"baseline_captured_at"`
	MutationStartedAt   time.Time `json:"mutation_started_at"`
	MutationCompletedAt time.Time `json:"mutation_completed_at,omitempty"`
	MutationSummary     string    `json:"mutation_summary"`
	SourceWorkspaceRef  string    `json:"source_workspace_ref,omitempty"`
	CreatedAt           time.Time `json:"created_at"`
	CreatedBy           string    `json:"created_by"`
}

var ErrCandidateMutationRecordNotFound = errors.New("mission store candidate mutation record not found")

func StoreCandidateMutationsDir(root string) string {
	return filepath.Join(root, "runtime_packs", "candidate_mutations")
}

func StoreCandidateMutationPath(root, mutationID string) string {
	return filepath.Join(StoreCandidateMutationsDir(root), strings.TrimSpace(mutationID)+".json")
}

func NormalizeCandidateMutationRef(ref CandidateMutationRef) CandidateMutationRef {
	ref.MutationID = strings.TrimSpace(ref.MutationID)
	return ref
}

func NormalizeCandidateMutationRecord(record CandidateMutationRecord) CandidateMutationRecord {
	record.MutationID = strings.TrimSpace(record.MutationID)
	record.RunID = strings.TrimSpace(record.RunID)
	record.CandidateID = strings.TrimSpace(record.CandidateID)
	record.EvalSuiteID = strings.TrimSpace(record.EvalSuiteID)
	record.BaselinePackID = strings.TrimSpace(record.BaselinePackID)
	record.CandidatePackID = strings.TrimSpace(record.CandidatePackID)
	record.BaselineResultRef = strings.TrimSpace(record.BaselineResultRef)
	record.BaselineCapturedAt = record.BaselineCapturedAt.UTC()
	record.MutationStartedAt = record.MutationStartedAt.UTC()
	record.MutationCompletedAt = record.MutationCompletedAt.UTC()
	record.MutationSummary = strings.TrimSpace(record.MutationSummary)
	record.SourceWorkspaceRef = strings.TrimSpace(record.SourceWorkspaceRef)
	record.CreatedAt = record.CreatedAt.UTC()
	record.CreatedBy = strings.TrimSpace(record.CreatedBy)
	return record
}

func CandidateMutationImprovementRunRef(record CandidateMutationRecord) ImprovementRunRef {
	return ImprovementRunRef{RunID: strings.TrimSpace(record.RunID)}
}

func CandidateMutationImprovementCandidateRef(record CandidateMutationRecord) ImprovementCandidateRef {
	return ImprovementCandidateRef{CandidateID: strings.TrimSpace(record.CandidateID)}
}

func CandidateMutationEvalSuiteRef(record CandidateMutationRecord) EvalSuiteRef {
	return EvalSuiteRef{EvalSuiteID: strings.TrimSpace(record.EvalSuiteID)}
}

func CandidateMutationBaselinePackRef(record CandidateMutationRecord) RuntimePackRef {
	return RuntimePackRef{PackID: strings.TrimSpace(record.BaselinePackID)}
}

func CandidateMutationCandidatePackRef(record CandidateMutationRecord) RuntimePackRef {
	return RuntimePackRef{PackID: strings.TrimSpace(record.CandidatePackID)}
}

func ValidateCandidateMutationRef(ref CandidateMutationRef) error {
	return validateRuntimePackIDField("candidate mutation ref", "mutation_id", ref.MutationID)
}

func ValidateCandidateMutationRecord(record CandidateMutationRecord) error {
	if record.RecordVersion <= 0 {
		return fmt.Errorf("mission store candidate mutation record_version must be positive")
	}
	if err := ValidateCandidateMutationRef(CandidateMutationRef{MutationID: record.MutationID}); err != nil {
		return err
	}
	if err := ValidateImprovementRunRef(CandidateMutationImprovementRunRef(record)); err != nil {
		return fmt.Errorf("mission store candidate mutation run_id %q: %w", record.RunID, err)
	}
	if err := ValidateImprovementCandidateRef(CandidateMutationImprovementCandidateRef(record)); err != nil {
		return fmt.Errorf("mission store candidate mutation candidate_id %q: %w", record.CandidateID, err)
	}
	if err := ValidateEvalSuiteRef(CandidateMutationEvalSuiteRef(record)); err != nil {
		return fmt.Errorf("mission store candidate mutation eval_suite_id %q: %w", record.EvalSuiteID, err)
	}
	if err := ValidateRuntimePackRef(CandidateMutationBaselinePackRef(record)); err != nil {
		return fmt.Errorf("mission store candidate mutation baseline_pack_id %q: %w", record.BaselinePackID, err)
	}
	if err := ValidateRuntimePackRef(CandidateMutationCandidatePackRef(record)); err != nil {
		return fmt.Errorf("mission store candidate mutation candidate_pack_id %q: %w", record.CandidatePackID, err)
	}
	if record.BaselineResultRef == "" {
		return fmt.Errorf("mission store candidate mutation baseline_result_ref is required")
	}
	if record.BaselineCapturedAt.IsZero() {
		return fmt.Errorf("mission store candidate mutation baseline_captured_at is required")
	}
	if record.MutationStartedAt.IsZero() {
		return fmt.Errorf("mission store candidate mutation mutation_started_at is required")
	}
	if record.MutationStartedAt.Before(record.BaselineCapturedAt) {
		return fmt.Errorf("mission store candidate mutation mutation_started_at must not precede baseline_captured_at")
	}
	if !record.MutationCompletedAt.IsZero() && record.MutationCompletedAt.Before(record.MutationStartedAt) {
		return fmt.Errorf("mission store candidate mutation mutation_completed_at must not precede mutation_started_at")
	}
	if record.MutationSummary == "" {
		return fmt.Errorf("mission store candidate mutation mutation_summary is required")
	}
	if record.CreatedAt.IsZero() {
		return fmt.Errorf("mission store candidate mutation created_at is required")
	}
	if record.CreatedBy == "" {
		return fmt.Errorf("mission store candidate mutation created_by is required")
	}
	return nil
}

func StoreCandidateMutationRecord(root string, record CandidateMutationRecord) (CandidateMutationRecord, bool, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return CandidateMutationRecord{}, false, err
	}
	record = NormalizeCandidateMutationRecord(record)
	record.RecordVersion = normalizeRecordVersion(record.RecordVersion)
	if err := ValidateCandidateMutationRecord(record); err != nil {
		return CandidateMutationRecord{}, false, err
	}
	if err := validateCandidateMutationLinkage(root, record); err != nil {
		return CandidateMutationRecord{}, false, err
	}

	path := StoreCandidateMutationPath(root, record.MutationID)
	existing, err := loadCandidateMutationRecordFile(root, path)
	if err == nil {
		if reflect.DeepEqual(existing, record) {
			return existing, false, nil
		}
		return CandidateMutationRecord{}, false, fmt.Errorf("mission store candidate mutation %q already exists", record.MutationID)
	}
	if !errors.Is(err, os.ErrNotExist) {
		return CandidateMutationRecord{}, false, err
	}
	if err := WriteStoreJSONAtomic(path, record); err != nil {
		return CandidateMutationRecord{}, false, err
	}
	stored, err := LoadCandidateMutationRecord(root, record.MutationID)
	if err != nil {
		return CandidateMutationRecord{}, false, err
	}
	return stored, true, nil
}

func LoadCandidateMutationRecord(root, mutationID string) (CandidateMutationRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return CandidateMutationRecord{}, err
	}
	ref := NormalizeCandidateMutationRef(CandidateMutationRef{MutationID: mutationID})
	if err := ValidateCandidateMutationRef(ref); err != nil {
		return CandidateMutationRecord{}, err
	}
	record, err := loadCandidateMutationRecordFile(root, StoreCandidateMutationPath(root, ref.MutationID))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return CandidateMutationRecord{}, ErrCandidateMutationRecordNotFound
		}
		return CandidateMutationRecord{}, err
	}
	return record, nil
}

func ListCandidateMutationRecords(root string) ([]CandidateMutationRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return nil, err
	}
	return listStoreJSONRecords(StoreCandidateMutationsDir(root), func(path string) (CandidateMutationRecord, error) {
		return loadCandidateMutationRecordFile(root, path)
	})
}

func loadCandidateMutationRecordFile(root, path string) (CandidateMutationRecord, error) {
	var record CandidateMutationRecord
	if err := LoadStoreJSON(path, &record); err != nil {
		return CandidateMutationRecord{}, err
	}
	record = NormalizeCandidateMutationRecord(record)
	if err := ValidateCandidateMutationRecord(record); err != nil {
		return CandidateMutationRecord{}, err
	}
	if err := validateCandidateMutationLinkage(root, record); err != nil {
		return CandidateMutationRecord{}, err
	}
	return record, nil
}

func validateCandidateMutationLinkage(root string, record CandidateMutationRecord) error {
	run, err := LoadImprovementRunRecord(root, record.RunID)
	if err != nil {
		return fmt.Errorf("mission store candidate mutation run_id %q: %w", record.RunID, err)
	}
	candidate, err := LoadImprovementCandidateRecord(root, record.CandidateID)
	if err != nil {
		return fmt.Errorf("mission store candidate mutation candidate_id %q: %w", record.CandidateID, err)
	}
	if run.CandidateID != record.CandidateID {
		return fmt.Errorf("mission store candidate mutation candidate_id %q does not match run candidate_id %q", record.CandidateID, run.CandidateID)
	}
	if run.EvalSuiteID != record.EvalSuiteID {
		return fmt.Errorf("mission store candidate mutation eval_suite_id %q does not match run eval_suite_id %q", record.EvalSuiteID, run.EvalSuiteID)
	}
	if run.BaselinePackID != record.BaselinePackID {
		return fmt.Errorf("mission store candidate mutation baseline_pack_id %q does not match run baseline_pack_id %q", record.BaselinePackID, run.BaselinePackID)
	}
	if run.CandidatePackID != record.CandidatePackID {
		return fmt.Errorf("mission store candidate mutation candidate_pack_id %q does not match run candidate_pack_id %q", record.CandidatePackID, run.CandidatePackID)
	}
	if candidate.BaselinePackID != record.BaselinePackID {
		return fmt.Errorf("mission store candidate mutation baseline_pack_id %q does not match candidate baseline_pack_id %q", record.BaselinePackID, candidate.BaselinePackID)
	}
	if candidate.CandidatePackID != record.CandidatePackID {
		return fmt.Errorf("mission store candidate mutation candidate_pack_id %q does not match candidate candidate_pack_id %q", record.CandidatePackID, candidate.CandidatePackID)
	}
	if !candidateMutationContainsString(candidate.ValidationBasisRefs, record.BaselineResultRef) {
		return fmt.Errorf("mission store candidate mutation baseline_result_ref %q is not present in candidate validation_basis_refs", record.BaselineResultRef)
	}
	return nil
}

func candidateMutationContainsString(values []string, value string) bool {
	value = strings.TrimSpace(value)
	for _, candidate := range values {
		if strings.TrimSpace(candidate) == value {
			return true
		}
	}
	return false
}
