package missioncontrol

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"
)

type EvalSuiteRef struct {
	EvalSuiteID string `json:"eval_suite_id"`
}

type EvalSuiteRecord struct {
	RecordVersion     int       `json:"record_version"`
	EvalSuiteID       string    `json:"eval_suite_id"`
	RubricRef         string    `json:"rubric_ref"`
	TrainCorpusRef    string    `json:"train_corpus_ref"`
	HoldoutCorpusRef  string    `json:"holdout_corpus_ref"`
	EvaluatorRef      string    `json:"evaluator_ref"`
	NegativeCaseCount int       `json:"negative_case_count"`
	BoundaryCaseCount int       `json:"boundary_case_count"`
	FrozenForRun      bool      `json:"frozen_for_run"`
	CandidateID       string    `json:"candidate_id,omitempty"`
	BaselinePackID    string    `json:"baseline_pack_id,omitempty"`
	CandidatePackID   string    `json:"candidate_pack_id,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
	CreatedBy         string    `json:"created_by"`
}

var ErrEvalSuiteRecordNotFound = errors.New("mission store eval-suite record not found")

func StoreEvalSuitesDir(root string) string {
	return filepath.Join(root, "runtime_packs", "eval_suites")
}

func StoreEvalSuitePath(root, evalSuiteID string) string {
	return filepath.Join(StoreEvalSuitesDir(root), strings.TrimSpace(evalSuiteID)+".json")
}

func NormalizeEvalSuiteRef(ref EvalSuiteRef) EvalSuiteRef {
	ref.EvalSuiteID = strings.TrimSpace(ref.EvalSuiteID)
	return ref
}

func NormalizeEvalSuiteRecord(record EvalSuiteRecord) EvalSuiteRecord {
	record.EvalSuiteID = strings.TrimSpace(record.EvalSuiteID)
	record.RubricRef = strings.TrimSpace(record.RubricRef)
	record.TrainCorpusRef = strings.TrimSpace(record.TrainCorpusRef)
	record.HoldoutCorpusRef = strings.TrimSpace(record.HoldoutCorpusRef)
	record.EvaluatorRef = strings.TrimSpace(record.EvaluatorRef)
	record.CandidateID = strings.TrimSpace(record.CandidateID)
	record.BaselinePackID = strings.TrimSpace(record.BaselinePackID)
	record.CandidatePackID = strings.TrimSpace(record.CandidatePackID)
	record.CreatedAt = record.CreatedAt.UTC()
	record.CreatedBy = strings.TrimSpace(record.CreatedBy)
	return record
}

func EvalSuiteImprovementCandidateRef(record EvalSuiteRecord) (ImprovementCandidateRef, bool) {
	candidateID := strings.TrimSpace(record.CandidateID)
	if candidateID == "" {
		return ImprovementCandidateRef{}, false
	}
	return ImprovementCandidateRef{CandidateID: candidateID}, true
}

func EvalSuiteBaselinePackRef(record EvalSuiteRecord) (RuntimePackRef, bool) {
	packID := strings.TrimSpace(record.BaselinePackID)
	if packID == "" {
		return RuntimePackRef{}, false
	}
	return RuntimePackRef{PackID: packID}, true
}

func EvalSuiteCandidatePackRef(record EvalSuiteRecord) (RuntimePackRef, bool) {
	packID := strings.TrimSpace(record.CandidatePackID)
	if packID == "" {
		return RuntimePackRef{}, false
	}
	return RuntimePackRef{PackID: packID}, true
}

func ValidateEvalSuiteRef(ref EvalSuiteRef) error {
	return validateEvalSuiteIdentifierField("eval-suite ref", "eval_suite_id", ref.EvalSuiteID)
}

func ValidateEvalSuiteRecord(record EvalSuiteRecord) error {
	if record.RecordVersion <= 0 {
		return fmt.Errorf("mission store eval-suite record_version must be positive")
	}
	if err := ValidateEvalSuiteRef(EvalSuiteRef{EvalSuiteID: record.EvalSuiteID}); err != nil {
		return err
	}
	if record.RubricRef == "" {
		return fmt.Errorf("mission store eval-suite rubric_ref is required")
	}
	if record.TrainCorpusRef == "" {
		return fmt.Errorf("mission store eval-suite train_corpus_ref is required")
	}
	if record.HoldoutCorpusRef == "" {
		return fmt.Errorf("mission store eval-suite holdout_corpus_ref is required")
	}
	if record.EvaluatorRef == "" {
		return fmt.Errorf("mission store eval-suite evaluator_ref is required")
	}
	if record.NegativeCaseCount < 0 {
		return fmt.Errorf("mission store eval-suite negative_case_count must be non-negative")
	}
	if record.BoundaryCaseCount < 0 {
		return fmt.Errorf("mission store eval-suite boundary_case_count must be non-negative")
	}
	if !record.FrozenForRun {
		return fmt.Errorf("mission store eval-suite frozen_for_run must be true")
	}
	if candidateRef, ok := EvalSuiteImprovementCandidateRef(record); ok {
		if err := ValidateImprovementCandidateRef(candidateRef); err != nil {
			return fmt.Errorf("mission store eval-suite candidate_id %q: %w", record.CandidateID, err)
		}
	}
	if baselineRef, ok := EvalSuiteBaselinePackRef(record); ok {
		if err := ValidateRuntimePackRef(baselineRef); err != nil {
			return fmt.Errorf("mission store eval-suite baseline_pack_id %q: %w", record.BaselinePackID, err)
		}
	}
	if candidatePackRef, ok := EvalSuiteCandidatePackRef(record); ok {
		if err := ValidateRuntimePackRef(candidatePackRef); err != nil {
			return fmt.Errorf("mission store eval-suite candidate_pack_id %q: %w", record.CandidatePackID, err)
		}
	}
	if record.CreatedAt.IsZero() {
		return fmt.Errorf("mission store eval-suite created_at is required")
	}
	if record.CreatedBy == "" {
		return fmt.Errorf("mission store eval-suite created_by is required")
	}
	return nil
}

func StoreEvalSuiteRecord(root string, record EvalSuiteRecord) error {
	if err := ValidateStoreRoot(root); err != nil {
		return err
	}
	record = NormalizeEvalSuiteRecord(record)
	record.RecordVersion = normalizeRecordVersion(record.RecordVersion)
	if err := ValidateEvalSuiteRecord(record); err != nil {
		return err
	}
	if err := validateEvalSuiteLinkage(root, record); err != nil {
		return err
	}
	return WriteStoreJSONAtomic(StoreEvalSuitePath(root, record.EvalSuiteID), record)
}

func LoadEvalSuiteRecord(root, evalSuiteID string) (EvalSuiteRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return EvalSuiteRecord{}, err
	}
	ref := NormalizeEvalSuiteRef(EvalSuiteRef{EvalSuiteID: evalSuiteID})
	if err := ValidateEvalSuiteRef(ref); err != nil {
		return EvalSuiteRecord{}, err
	}
	record, err := loadEvalSuiteRecordFile(root, StoreEvalSuitePath(root, ref.EvalSuiteID))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return EvalSuiteRecord{}, ErrEvalSuiteRecordNotFound
		}
		return EvalSuiteRecord{}, err
	}
	return record, nil
}

func ListEvalSuiteRecords(root string) ([]EvalSuiteRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(StoreEvalSuitesDir(root))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !isStoreJSONDataFile(entry.Name()) {
			continue
		}
		names = append(names, entry.Name())
	}
	sort.Strings(names)

	records := make([]EvalSuiteRecord, 0, len(names))
	for _, name := range names {
		record, err := loadEvalSuiteRecordFile(root, filepath.Join(StoreEvalSuitesDir(root), name))
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, nil
}

func loadEvalSuiteRecordFile(root, path string) (EvalSuiteRecord, error) {
	var record EvalSuiteRecord
	if err := LoadStoreJSON(path, &record); err != nil {
		return EvalSuiteRecord{}, err
	}
	record = NormalizeEvalSuiteRecord(record)
	if err := ValidateEvalSuiteRecord(record); err != nil {
		return EvalSuiteRecord{}, err
	}
	if err := validateEvalSuiteLinkage(root, record); err != nil {
		return EvalSuiteRecord{}, err
	}
	return record, nil
}

func validateEvalSuiteLinkage(root string, record EvalSuiteRecord) error {
	if candidateRef, ok := EvalSuiteImprovementCandidateRef(record); ok {
		candidate, err := LoadImprovementCandidateRecord(root, candidateRef.CandidateID)
		if err != nil {
			return fmt.Errorf("mission store eval-suite candidate_id %q: %w", candidateRef.CandidateID, err)
		}
		if baselineRef, ok := EvalSuiteBaselinePackRef(record); ok && candidate.BaselinePackID != baselineRef.PackID {
			return fmt.Errorf(
				"mission store eval-suite baseline_pack_id %q does not match candidate baseline_pack_id %q",
				baselineRef.PackID,
				candidate.BaselinePackID,
			)
		}
		if candidatePackRef, ok := EvalSuiteCandidatePackRef(record); ok && candidate.CandidatePackID != candidatePackRef.PackID {
			return fmt.Errorf(
				"mission store eval-suite candidate_pack_id %q does not match candidate candidate_pack_id %q",
				candidatePackRef.PackID,
				candidate.CandidatePackID,
			)
		}
	}
	if baselineRef, ok := EvalSuiteBaselinePackRef(record); ok {
		if _, err := LoadRuntimePackRecord(root, baselineRef.PackID); err != nil {
			return fmt.Errorf("mission store eval-suite baseline_pack_id %q: %w", baselineRef.PackID, err)
		}
	}
	if candidatePackRef, ok := EvalSuiteCandidatePackRef(record); ok {
		if _, err := LoadRuntimePackRecord(root, candidatePackRef.PackID); err != nil {
			return fmt.Errorf("mission store eval-suite candidate_pack_id %q: %w", candidatePackRef.PackID, err)
		}
	}
	return nil
}

func validateEvalSuiteIdentifierField(surface, fieldName, value string) error {
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
