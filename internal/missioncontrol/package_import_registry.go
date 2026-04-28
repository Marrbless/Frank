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

type PackageImportActivationState string

const (
	PackageImportActivationStateCandidateOnly PackageImportActivationState = "candidate_only"
)

type PackageAuthorityGrantDeclaration struct {
	AuthorityKind string `json:"authority_kind"`
	TargetRef     string `json:"target_ref,omitempty"`
	Reason        string `json:"reason,omitempty"`
}

type PackageImportRecord struct {
	RecordVersion           int                                `json:"record_version"`
	ImportID                string                             `json:"import_id"`
	CandidateID             string                             `json:"candidate_id"`
	CandidatePackID         string                             `json:"candidate_pack_id"`
	SourcePackageRef        string                             `json:"source_package_ref"`
	SourceProject           string                             `json:"source_project,omitempty"`
	SourceVersion           string                             `json:"source_version,omitempty"`
	ContentRef              string                             `json:"content_ref"`
	ContentSHA256           string                             `json:"content_sha256"`
	ContentKinds            []RuntimePackComponentKind         `json:"content_kinds"`
	SurfaceClass            string                             `json:"surface_class"`
	DeclaredSurfaces        []string                           `json:"declared_surfaces"`
	DeclaredAuthorityGrants []PackageAuthorityGrantDeclaration `json:"declared_authority_grants,omitempty"`
	ActivationState         PackageImportActivationState       `json:"activation_state"`
	ProvenanceRef           string                             `json:"provenance_ref"`
	SourceSummary           string                             `json:"source_summary"`
	CreatedAt               time.Time                          `json:"created_at"`
	CreatedBy               string                             `json:"created_by"`
}

var ErrPackageImportRecordNotFound = errors.New("mission store package import record not found")

func StorePackageImportsDir(root string) string {
	return filepath.Join(root, "runtime_packs", "package_imports")
}

func StorePackageImportPath(root, importID string) string {
	return filepath.Join(StorePackageImportsDir(root), strings.TrimSpace(importID)+".json")
}

func NormalizePackageAuthorityGrantDeclaration(grant PackageAuthorityGrantDeclaration) PackageAuthorityGrantDeclaration {
	grant.AuthorityKind = strings.TrimSpace(grant.AuthorityKind)
	grant.TargetRef = strings.TrimSpace(grant.TargetRef)
	grant.Reason = strings.TrimSpace(grant.Reason)
	return grant
}

func NormalizePackageImportRecord(record PackageImportRecord) PackageImportRecord {
	record.ImportID = strings.TrimSpace(record.ImportID)
	record.CandidateID = strings.TrimSpace(record.CandidateID)
	record.CandidatePackID = strings.TrimSpace(record.CandidatePackID)
	record.SourcePackageRef = strings.TrimSpace(record.SourcePackageRef)
	record.SourceProject = strings.TrimSpace(record.SourceProject)
	record.SourceVersion = strings.TrimSpace(record.SourceVersion)
	record.ContentRef = strings.TrimSpace(record.ContentRef)
	record.ContentSHA256 = strings.ToLower(strings.TrimSpace(record.ContentSHA256))
	record.ContentKinds = normalizePackageImportContentKinds(record.ContentKinds)
	record.SurfaceClass = strings.TrimSpace(record.SurfaceClass)
	record.DeclaredSurfaces = normalizeRuntimePackStrings(record.DeclaredSurfaces)
	record.ActivationState = PackageImportActivationState(strings.TrimSpace(string(record.ActivationState)))
	record.ProvenanceRef = strings.TrimSpace(record.ProvenanceRef)
	record.SourceSummary = strings.TrimSpace(record.SourceSummary)
	record.CreatedAt = record.CreatedAt.UTC()
	record.CreatedBy = strings.TrimSpace(record.CreatedBy)
	if len(record.DeclaredAuthorityGrants) > 0 {
		grants := make([]PackageAuthorityGrantDeclaration, 0, len(record.DeclaredAuthorityGrants))
		for _, grant := range record.DeclaredAuthorityGrants {
			normalized := NormalizePackageAuthorityGrantDeclaration(grant)
			if normalized.AuthorityKind == "" && normalized.TargetRef == "" && normalized.Reason == "" {
				continue
			}
			grants = append(grants, normalized)
		}
		record.DeclaredAuthorityGrants = grants
	}
	return record
}

func ValidatePackageImportRecord(record PackageImportRecord) error {
	if record.RecordVersion <= 0 {
		return fmt.Errorf("mission store package import record_version must be positive")
	}
	if err := validateRuntimePackIDField("mission store package import", "import_id", record.ImportID); err != nil {
		return err
	}
	if err := ValidateImprovementCandidateRef(ImprovementCandidateRef{CandidateID: record.CandidateID}); err != nil {
		return fmt.Errorf("mission store package import candidate_id %q: %w", record.CandidateID, err)
	}
	if err := ValidateRuntimePackRef(RuntimePackRef{PackID: record.CandidatePackID}); err != nil {
		return fmt.Errorf("mission store package import candidate_pack_id %q: %w", record.CandidatePackID, err)
	}
	if record.SourcePackageRef == "" {
		return fmt.Errorf("mission store package import source_package_ref is required")
	}
	if record.ContentRef == "" {
		return fmt.Errorf("mission store package import content_ref is required")
	}
	if err := validateRuntimePackComponentSHA256(record.ContentSHA256); err != nil {
		return err
	}
	if len(record.ContentKinds) == 0 {
		return fmt.Errorf("mission store package import content_kinds are required")
	}
	for _, kind := range record.ContentKinds {
		if err := ValidateRuntimePackComponentKind(kind); err != nil {
			return fmt.Errorf("mission store package import content_kinds: %w", err)
		}
	}
	if record.SurfaceClass == "" {
		return fmt.Errorf("mission store package import surface_class is required")
	}
	if len(record.DeclaredSurfaces) == 0 {
		return fmt.Errorf("mission store package import declared_surfaces are required")
	}
	if record.ActivationState != PackageImportActivationStateCandidateOnly {
		return fmt.Errorf("mission store package import activation_state must be %q", PackageImportActivationStateCandidateOnly)
	}
	if record.ProvenanceRef == "" {
		return fmt.Errorf("mission store package import provenance_ref is required")
	}
	if record.SourceSummary == "" {
		return fmt.Errorf("mission store package import source_summary is required")
	}
	if record.CreatedAt.IsZero() {
		return fmt.Errorf("mission store package import created_at is required")
	}
	if record.CreatedBy == "" {
		return fmt.Errorf("mission store package import created_by is required")
	}
	if len(record.DeclaredAuthorityGrants) > 0 {
		return fmt.Errorf("mission store package import %q rejected: %s", record.ImportID, packageAuthorityGrantBlockerSummary(record.DeclaredAuthorityGrants))
	}
	return nil
}

func StorePackageImportRecord(root string, record PackageImportRecord) (PackageImportRecord, bool, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return PackageImportRecord{}, false, err
	}
	record = NormalizePackageImportRecord(record)
	record.RecordVersion = normalizeRecordVersion(record.RecordVersion)
	if err := ValidatePackageImportRecord(record); err != nil {
		return PackageImportRecord{}, false, err
	}
	if err := validatePackageImportCandidateLinkage(root, record); err != nil {
		return PackageImportRecord{}, false, err
	}

	path := StorePackageImportPath(root, record.ImportID)
	existing, err := loadPackageImportRecordFile(root, path)
	if err == nil {
		if reflect.DeepEqual(existing, record) {
			return existing, false, nil
		}
		return PackageImportRecord{}, false, fmt.Errorf("mission store package import %q already exists", record.ImportID)
	}
	if !errors.Is(err, os.ErrNotExist) {
		return PackageImportRecord{}, false, err
	}
	if err := WriteStoreJSONAtomic(path, record); err != nil {
		return PackageImportRecord{}, false, err
	}
	stored, err := LoadPackageImportRecord(root, record.ImportID)
	if err != nil {
		return PackageImportRecord{}, false, err
	}
	return stored, true, nil
}

func LoadPackageImportRecord(root, importID string) (PackageImportRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return PackageImportRecord{}, err
	}
	normalizedImportID := strings.TrimSpace(importID)
	if err := validateRuntimePackIDField("mission store package import", "import_id", normalizedImportID); err != nil {
		return PackageImportRecord{}, err
	}
	record, err := loadPackageImportRecordFile(root, StorePackageImportPath(root, normalizedImportID))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return PackageImportRecord{}, ErrPackageImportRecordNotFound
		}
		return PackageImportRecord{}, err
	}
	return record, nil
}

func ListPackageImportRecords(root string) ([]PackageImportRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return nil, err
	}
	return listStoreJSONRecords(StorePackageImportsDir(root), func(path string) (PackageImportRecord, error) {
		return loadPackageImportRecordFile(root, path)
	})
}

func loadPackageImportRecordFile(root, path string) (PackageImportRecord, error) {
	var record PackageImportRecord
	if err := LoadStoreJSON(path, &record); err != nil {
		return PackageImportRecord{}, err
	}
	record = NormalizePackageImportRecord(record)
	if err := ValidatePackageImportRecord(record); err != nil {
		return PackageImportRecord{}, err
	}
	if err := validatePackageImportCandidateLinkage(root, record); err != nil {
		return PackageImportRecord{}, err
	}
	return record, nil
}

func validatePackageImportCandidateLinkage(root string, record PackageImportRecord) error {
	candidate, err := LoadImprovementCandidateRecord(root, record.CandidateID)
	if err != nil {
		return fmt.Errorf("mission store package import candidate_id %q: %w", record.CandidateID, err)
	}
	if candidate.CandidatePackID != record.CandidatePackID {
		return fmt.Errorf("mission store package import candidate_pack_id %q does not match candidate %q candidate_pack_id %q", record.CandidatePackID, record.CandidateID, candidate.CandidatePackID)
	}
	return nil
}

func normalizePackageImportContentKinds(values []RuntimePackComponentKind) []RuntimePackComponentKind {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[RuntimePackComponentKind]struct{}, len(values))
	normalized := make([]RuntimePackComponentKind, 0, len(values))
	for _, value := range values {
		kind := RuntimePackComponentKind(strings.TrimSpace(string(value)))
		if kind == "" {
			continue
		}
		if _, ok := seen[kind]; ok {
			continue
		}
		seen[kind] = struct{}{}
		normalized = append(normalized, kind)
	}
	if len(normalized) == 0 {
		return nil
	}
	return normalized
}

func packageAuthorityGrantBlockerSummary(grants []PackageAuthorityGrantDeclaration) string {
	parts := make([]string, 0, len(grants))
	for _, grant := range grants {
		grant = NormalizePackageAuthorityGrantDeclaration(grant)
		if grant.AuthorityKind == "" {
			grant.AuthorityKind = "unspecified_authority"
		}
		reason := grant.Reason
		if reason == "" {
			reason = "package content may not grant authority"
		}
		parts = append(parts, fmt.Sprintf("%s: %s", grant.AuthorityKind, reason))
	}
	if len(parts) == 0 {
		return string(RejectionCodeV4PackageAuthorityGrantForbidden) + ": package content may not grant authority"
	}
	return string(RejectionCodeV4PackageAuthorityGrantForbidden) + ": " + strings.Join(parts, "; ")
}
