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

type ImprovementCandidateRef struct {
	CandidateID string `json:"candidate_id"`
}

type ImprovementCandidateRecord struct {
	RecordVersion       int       `json:"record_version"`
	CandidateID         string    `json:"candidate_id"`
	BaselinePackID      string    `json:"baseline_pack_id"`
	CandidatePackID     string    `json:"candidate_pack_id"`
	SourceWorkspaceRef  string    `json:"source_workspace_ref,omitempty"`
	SourceSummary       string    `json:"source_summary,omitempty"`
	ValidationBasisRefs []string  `json:"validation_basis_refs"`
	HotUpdateID         string    `json:"hot_update_id,omitempty"`
	CreatedAt           time.Time `json:"created_at"`
	CreatedBy           string    `json:"created_by"`
}

var ErrImprovementCandidateRecordNotFound = errors.New("mission store improvement candidate record not found")

func StoreImprovementCandidatesDir(root string) string {
	return filepath.Join(root, "runtime_packs", "improvement_candidates")
}

func StoreImprovementCandidatePath(root, candidateID string) string {
	return filepath.Join(StoreImprovementCandidatesDir(root), strings.TrimSpace(candidateID)+".json")
}

func NormalizeImprovementCandidateRef(ref ImprovementCandidateRef) ImprovementCandidateRef {
	ref.CandidateID = strings.TrimSpace(ref.CandidateID)
	return ref
}

func NormalizeImprovementCandidateRecord(record ImprovementCandidateRecord) ImprovementCandidateRecord {
	record.CandidateID = strings.TrimSpace(record.CandidateID)
	record.BaselinePackID = strings.TrimSpace(record.BaselinePackID)
	record.CandidatePackID = strings.TrimSpace(record.CandidatePackID)
	record.SourceWorkspaceRef = strings.TrimSpace(record.SourceWorkspaceRef)
	record.SourceSummary = strings.TrimSpace(record.SourceSummary)
	record.ValidationBasisRefs = normalizeImprovementCandidateStrings(record.ValidationBasisRefs)
	record.HotUpdateID = strings.TrimSpace(record.HotUpdateID)
	record.CreatedAt = record.CreatedAt.UTC()
	record.CreatedBy = strings.TrimSpace(record.CreatedBy)
	return record
}

func ImprovementCandidateBaselinePackRef(record ImprovementCandidateRecord) RuntimePackRef {
	return RuntimePackRef{PackID: strings.TrimSpace(record.BaselinePackID)}
}

func ImprovementCandidateProposedPackRef(record ImprovementCandidateRecord) RuntimePackRef {
	return RuntimePackRef{PackID: strings.TrimSpace(record.CandidatePackID)}
}

func ImprovementCandidateHotUpdateGateRef(record ImprovementCandidateRecord) (HotUpdateGateRef, bool) {
	hotUpdateID := strings.TrimSpace(record.HotUpdateID)
	if hotUpdateID == "" {
		return HotUpdateGateRef{}, false
	}
	return HotUpdateGateRef{HotUpdateID: hotUpdateID}, true
}

func ValidateImprovementCandidateRef(ref ImprovementCandidateRef) error {
	return validateImprovementCandidateIdentifierField("improvement candidate ref", "candidate_id", ref.CandidateID)
}

func ValidateImprovementCandidateRecord(record ImprovementCandidateRecord) error {
	if record.RecordVersion <= 0 {
		return fmt.Errorf("mission store improvement candidate record_version must be positive")
	}
	if err := ValidateImprovementCandidateRef(ImprovementCandidateRef{CandidateID: record.CandidateID}); err != nil {
		return err
	}
	if err := ValidateRuntimePackRef(ImprovementCandidateBaselinePackRef(record)); err != nil {
		return fmt.Errorf("mission store improvement candidate baseline_pack_id %q: %w", record.BaselinePackID, err)
	}
	if err := ValidateRuntimePackRef(ImprovementCandidateProposedPackRef(record)); err != nil {
		return fmt.Errorf("mission store improvement candidate candidate_pack_id %q: %w", record.CandidatePackID, err)
	}
	if record.SourceWorkspaceRef == "" && record.SourceSummary == "" {
		return fmt.Errorf("mission store improvement candidate requires source_workspace_ref or source_summary")
	}
	if len(record.ValidationBasisRefs) == 0 {
		return fmt.Errorf("mission store improvement candidate validation_basis_refs are required")
	}
	if gateRef, ok := ImprovementCandidateHotUpdateGateRef(record); ok {
		if err := ValidateHotUpdateGateRef(gateRef); err != nil {
			return fmt.Errorf("mission store improvement candidate hot_update_id %q: %w", record.HotUpdateID, err)
		}
	}
	if record.CreatedAt.IsZero() {
		return fmt.Errorf("mission store improvement candidate created_at is required")
	}
	if record.CreatedBy == "" {
		return fmt.Errorf("mission store improvement candidate created_by is required")
	}
	return nil
}

func StoreImprovementCandidateRecord(root string, record ImprovementCandidateRecord) error {
	if err := ValidateStoreRoot(root); err != nil {
		return err
	}
	record = NormalizeImprovementCandidateRecord(record)
	record.RecordVersion = normalizeRecordVersion(record.RecordVersion)
	if err := ValidateImprovementCandidateRecord(record); err != nil {
		return err
	}
	if err := validateImprovementCandidateLinkage(root, record); err != nil {
		return err
	}
	return WriteStoreJSONAtomic(StoreImprovementCandidatePath(root, record.CandidateID), record)
}

func LoadImprovementCandidateRecord(root, candidateID string) (ImprovementCandidateRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return ImprovementCandidateRecord{}, err
	}
	ref := NormalizeImprovementCandidateRef(ImprovementCandidateRef{CandidateID: candidateID})
	if err := ValidateImprovementCandidateRef(ref); err != nil {
		return ImprovementCandidateRecord{}, err
	}
	record, err := loadImprovementCandidateRecordFile(root, StoreImprovementCandidatePath(root, ref.CandidateID))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return ImprovementCandidateRecord{}, ErrImprovementCandidateRecordNotFound
		}
		return ImprovementCandidateRecord{}, err
	}
	return record, nil
}

func ListImprovementCandidateRecords(root string) ([]ImprovementCandidateRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(StoreImprovementCandidatesDir(root))
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

	records := make([]ImprovementCandidateRecord, 0, len(names))
	for _, name := range names {
		record, err := loadImprovementCandidateRecordFile(root, filepath.Join(StoreImprovementCandidatesDir(root), name))
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, nil
}

func loadImprovementCandidateRecordFile(root, path string) (ImprovementCandidateRecord, error) {
	var record ImprovementCandidateRecord
	if err := LoadStoreJSON(path, &record); err != nil {
		return ImprovementCandidateRecord{}, err
	}
	record = NormalizeImprovementCandidateRecord(record)
	if err := ValidateImprovementCandidateRecord(record); err != nil {
		return ImprovementCandidateRecord{}, err
	}
	if err := validateImprovementCandidateLinkage(root, record); err != nil {
		return ImprovementCandidateRecord{}, err
	}
	return record, nil
}

func validateImprovementCandidateLinkage(root string, record ImprovementCandidateRecord) error {
	baselineRef := ImprovementCandidateBaselinePackRef(record)
	if _, err := LoadRuntimePackRecord(root, baselineRef.PackID); err != nil {
		return fmt.Errorf("mission store improvement candidate baseline_pack_id %q: %w", baselineRef.PackID, err)
	}

	candidateRef := ImprovementCandidateProposedPackRef(record)
	candidatePack, err := LoadRuntimePackRecord(root, candidateRef.PackID)
	if err != nil {
		return fmt.Errorf("mission store improvement candidate candidate_pack_id %q: %w", candidateRef.PackID, err)
	}
	if candidatePack.ParentPackID != "" && candidatePack.ParentPackID != baselineRef.PackID {
		return fmt.Errorf(
			"mission store improvement candidate candidate_pack_id %q parent_pack_id %q does not match baseline_pack_id %q",
			candidateRef.PackID,
			candidatePack.ParentPackID,
			baselineRef.PackID,
		)
	}

	gateRef, ok := ImprovementCandidateHotUpdateGateRef(record)
	if !ok {
		return nil
	}
	gate, err := LoadHotUpdateGateRecord(root, gateRef.HotUpdateID)
	if err != nil {
		return fmt.Errorf("mission store improvement candidate hot_update_id %q: %w", gateRef.HotUpdateID, err)
	}
	if gate.CandidatePackID != candidateRef.PackID {
		return fmt.Errorf(
			"mission store improvement candidate hot_update_id %q candidate_pack_id %q does not match candidate_pack_id %q",
			gateRef.HotUpdateID,
			gate.CandidatePackID,
			candidateRef.PackID,
		)
	}
	if gate.PreviousActivePackID != baselineRef.PackID {
		return fmt.Errorf(
			"mission store improvement candidate hot_update_id %q previous_active_pack_id %q does not match baseline_pack_id %q",
			gateRef.HotUpdateID,
			gate.PreviousActivePackID,
			baselineRef.PackID,
		)
	}
	return nil
}

func normalizeImprovementCandidateStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	normalized := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		normalized = append(normalized, trimmed)
	}
	if len(normalized) == 0 {
		return nil
	}
	return normalized
}

func validateImprovementCandidateIdentifierField(surface, fieldName, value string) error {
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
