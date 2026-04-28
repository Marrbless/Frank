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

type ImprovementWorkspaceRunOutcome string

const (
	ImprovementWorkspaceRunOutcomeCrashed ImprovementWorkspaceRunOutcome = "crashed"
	ImprovementWorkspaceRunOutcomeFailed  ImprovementWorkspaceRunOutcome = "failed"
)

type ImprovementWorkspaceActivePointerSnapshot struct {
	ActivePackID         string    `json:"active_pack_id"`
	PreviousActivePackID string    `json:"previous_active_pack_id,omitempty"`
	LastKnownGoodPackID  string    `json:"last_known_good_pack_id,omitempty"`
	UpdateRecordRef      string    `json:"update_record_ref"`
	ReloadGeneration     uint64    `json:"reload_generation"`
	UpdatedAt            time.Time `json:"updated_at"`
	UpdatedBy            string    `json:"updated_by"`
}

type ImprovementWorkspaceRunRecord struct {
	RecordVersion             int                                       `json:"record_version"`
	WorkspaceRunID            string                                    `json:"workspace_run_id"`
	RunID                     string                                    `json:"run_id"`
	CandidateID               string                                    `json:"candidate_id"`
	ExecutionHost             string                                    `json:"execution_host"`
	Outcome                   ImprovementWorkspaceRunOutcome            `json:"outcome"`
	FailureReason             string                                    `json:"failure_reason"`
	StartedAt                 time.Time                                 `json:"started_at"`
	CompletedAt               time.Time                                 `json:"completed_at"`
	ActivePointerAtStart      ImprovementWorkspaceActivePointerSnapshot `json:"active_pointer_at_start"`
	ActivePointerAtCompletion ImprovementWorkspaceActivePointerSnapshot `json:"active_pointer_at_completion"`
	CreatedAt                 time.Time                                 `json:"created_at"`
	CreatedBy                 string                                    `json:"created_by"`
}

var ErrImprovementWorkspaceRunRecordNotFound = errors.New("mission store improvement workspace run record not found")

func StoreImprovementWorkspaceRunsDir(root string) string {
	return filepath.Join(root, "runtime_packs", "improvement_workspace_runs")
}

func StoreImprovementWorkspaceRunPath(root, workspaceRunID string) string {
	return filepath.Join(StoreImprovementWorkspaceRunsDir(root), strings.TrimSpace(workspaceRunID)+".json")
}

func ImprovementWorkspaceActivePointerSnapshotFromPointer(pointer ActiveRuntimePackPointer) ImprovementWorkspaceActivePointerSnapshot {
	pointer = NormalizeActiveRuntimePackPointer(pointer)
	return ImprovementWorkspaceActivePointerSnapshot{
		ActivePackID:         pointer.ActivePackID,
		PreviousActivePackID: pointer.PreviousActivePackID,
		LastKnownGoodPackID:  pointer.LastKnownGoodPackID,
		UpdateRecordRef:      pointer.UpdateRecordRef,
		ReloadGeneration:     pointer.ReloadGeneration,
		UpdatedAt:            pointer.UpdatedAt,
		UpdatedBy:            pointer.UpdatedBy,
	}
}

func NormalizeImprovementWorkspaceActivePointerSnapshot(snapshot ImprovementWorkspaceActivePointerSnapshot) ImprovementWorkspaceActivePointerSnapshot {
	snapshot.ActivePackID = strings.TrimSpace(snapshot.ActivePackID)
	snapshot.PreviousActivePackID = strings.TrimSpace(snapshot.PreviousActivePackID)
	snapshot.LastKnownGoodPackID = strings.TrimSpace(snapshot.LastKnownGoodPackID)
	snapshot.UpdateRecordRef = strings.TrimSpace(snapshot.UpdateRecordRef)
	snapshot.UpdatedAt = snapshot.UpdatedAt.UTC()
	snapshot.UpdatedBy = strings.TrimSpace(snapshot.UpdatedBy)
	return snapshot
}

func NormalizeImprovementWorkspaceRunRecord(record ImprovementWorkspaceRunRecord) ImprovementWorkspaceRunRecord {
	record.WorkspaceRunID = strings.TrimSpace(record.WorkspaceRunID)
	record.RunID = strings.TrimSpace(record.RunID)
	record.CandidateID = strings.TrimSpace(record.CandidateID)
	record.ExecutionHost = strings.TrimSpace(record.ExecutionHost)
	record.Outcome = ImprovementWorkspaceRunOutcome(strings.TrimSpace(string(record.Outcome)))
	record.FailureReason = strings.TrimSpace(record.FailureReason)
	record.StartedAt = record.StartedAt.UTC()
	record.CompletedAt = record.CompletedAt.UTC()
	record.ActivePointerAtStart = NormalizeImprovementWorkspaceActivePointerSnapshot(record.ActivePointerAtStart)
	record.ActivePointerAtCompletion = NormalizeImprovementWorkspaceActivePointerSnapshot(record.ActivePointerAtCompletion)
	record.CreatedAt = record.CreatedAt.UTC()
	record.CreatedBy = strings.TrimSpace(record.CreatedBy)
	return record
}

func ValidateImprovementWorkspaceActivePointerSnapshot(context string, snapshot ImprovementWorkspaceActivePointerSnapshot) error {
	pointer := ActiveRuntimePackPointer{
		RecordVersion:        StoreRecordVersion,
		ActivePackID:         snapshot.ActivePackID,
		PreviousActivePackID: snapshot.PreviousActivePackID,
		LastKnownGoodPackID:  snapshot.LastKnownGoodPackID,
		UpdatedAt:            snapshot.UpdatedAt,
		UpdatedBy:            snapshot.UpdatedBy,
		UpdateRecordRef:      snapshot.UpdateRecordRef,
		ReloadGeneration:     snapshot.ReloadGeneration,
	}
	if err := ValidateActiveRuntimePackPointer(pointer); err != nil {
		return fmt.Errorf("%s: %w", context, err)
	}
	return nil
}

func ValidateImprovementWorkspaceRunRecord(record ImprovementWorkspaceRunRecord) error {
	if record.RecordVersion <= 0 {
		return fmt.Errorf("mission store improvement workspace run record_version must be positive")
	}
	if err := validateRuntimePackIDField("mission store improvement workspace run", "workspace_run_id", record.WorkspaceRunID); err != nil {
		return err
	}
	if err := ValidateImprovementRunRef(ImprovementRunRef{RunID: record.RunID}); err != nil {
		return fmt.Errorf("mission store improvement workspace run run_id %q: %w", record.RunID, err)
	}
	if err := ValidateImprovementCandidateRef(ImprovementCandidateRef{CandidateID: record.CandidateID}); err != nil {
		return fmt.Errorf("mission store improvement workspace run candidate_id %q: %w", record.CandidateID, err)
	}
	if record.ExecutionHost == "" {
		return fmt.Errorf("mission store improvement workspace run execution_host is required")
	}
	if !isValidImprovementWorkspaceRunOutcome(record.Outcome) {
		return fmt.Errorf("mission store improvement workspace run outcome %q is invalid", record.Outcome)
	}
	if record.FailureReason == "" {
		return fmt.Errorf("mission store improvement workspace run failure_reason is required")
	}
	if record.StartedAt.IsZero() {
		return fmt.Errorf("mission store improvement workspace run started_at is required")
	}
	if record.CompletedAt.IsZero() {
		return fmt.Errorf("mission store improvement workspace run completed_at is required")
	}
	if record.CompletedAt.Before(record.StartedAt) {
		return fmt.Errorf("mission store improvement workspace run completed_at must not precede started_at")
	}
	if err := ValidateImprovementWorkspaceActivePointerSnapshot("mission store improvement workspace run active_pointer_at_start", record.ActivePointerAtStart); err != nil {
		return err
	}
	if err := ValidateImprovementWorkspaceActivePointerSnapshot("mission store improvement workspace run active_pointer_at_completion", record.ActivePointerAtCompletion); err != nil {
		return err
	}
	if !reflect.DeepEqual(record.ActivePointerAtStart, record.ActivePointerAtCompletion) {
		return fmt.Errorf("mission store improvement workspace run active pointer changed between start and completion")
	}
	if record.CreatedAt.IsZero() {
		return fmt.Errorf("mission store improvement workspace run created_at is required")
	}
	if record.CreatedBy == "" {
		return fmt.Errorf("mission store improvement workspace run created_by is required")
	}
	return nil
}

func StoreImprovementWorkspaceRunRecord(root string, record ImprovementWorkspaceRunRecord) (ImprovementWorkspaceRunRecord, bool, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return ImprovementWorkspaceRunRecord{}, false, err
	}
	record = NormalizeImprovementWorkspaceRunRecord(record)
	record.RecordVersion = normalizeRecordVersion(record.RecordVersion)
	if err := ValidateImprovementWorkspaceRunRecord(record); err != nil {
		return ImprovementWorkspaceRunRecord{}, false, err
	}
	if err := validateImprovementWorkspaceRunLinkage(root, record, true); err != nil {
		return ImprovementWorkspaceRunRecord{}, false, err
	}

	path := StoreImprovementWorkspaceRunPath(root, record.WorkspaceRunID)
	existing, err := loadImprovementWorkspaceRunRecordFile(root, path)
	if err == nil {
		if reflect.DeepEqual(existing, record) {
			return existing, false, nil
		}
		return ImprovementWorkspaceRunRecord{}, false, fmt.Errorf("mission store improvement workspace run %q already exists", record.WorkspaceRunID)
	}
	if !errors.Is(err, os.ErrNotExist) {
		return ImprovementWorkspaceRunRecord{}, false, err
	}
	if err := WriteStoreJSONAtomic(path, record); err != nil {
		return ImprovementWorkspaceRunRecord{}, false, err
	}
	stored, err := LoadImprovementWorkspaceRunRecord(root, record.WorkspaceRunID)
	if err != nil {
		return ImprovementWorkspaceRunRecord{}, false, err
	}
	return stored, true, nil
}

func LoadImprovementWorkspaceRunRecord(root, workspaceRunID string) (ImprovementWorkspaceRunRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return ImprovementWorkspaceRunRecord{}, err
	}
	normalizedWorkspaceRunID := strings.TrimSpace(workspaceRunID)
	if err := validateRuntimePackIDField("mission store improvement workspace run", "workspace_run_id", normalizedWorkspaceRunID); err != nil {
		return ImprovementWorkspaceRunRecord{}, err
	}
	record, err := loadImprovementWorkspaceRunRecordFile(root, StoreImprovementWorkspaceRunPath(root, normalizedWorkspaceRunID))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return ImprovementWorkspaceRunRecord{}, ErrImprovementWorkspaceRunRecordNotFound
		}
		return ImprovementWorkspaceRunRecord{}, err
	}
	return record, nil
}

func ListImprovementWorkspaceRunRecords(root string) ([]ImprovementWorkspaceRunRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return nil, err
	}
	return listStoreJSONRecords(StoreImprovementWorkspaceRunsDir(root), func(path string) (ImprovementWorkspaceRunRecord, error) {
		return loadImprovementWorkspaceRunRecordFile(root, path)
	})
}

func loadImprovementWorkspaceRunRecordFile(root, path string) (ImprovementWorkspaceRunRecord, error) {
	var record ImprovementWorkspaceRunRecord
	if err := LoadStoreJSON(path, &record); err != nil {
		return ImprovementWorkspaceRunRecord{}, err
	}
	record = NormalizeImprovementWorkspaceRunRecord(record)
	if err := ValidateImprovementWorkspaceRunRecord(record); err != nil {
		return ImprovementWorkspaceRunRecord{}, err
	}
	if err := validateImprovementWorkspaceRunLinkage(root, record, false); err != nil {
		return ImprovementWorkspaceRunRecord{}, err
	}
	return record, nil
}

func validateImprovementWorkspaceRunLinkage(root string, record ImprovementWorkspaceRunRecord, requireCurrentCompletionPointer bool) error {
	run, err := LoadImprovementRunRecord(root, record.RunID)
	if err != nil {
		return fmt.Errorf("mission store improvement workspace run run_id %q: %w", record.RunID, err)
	}
	if run.ExecutionPlane != ExecutionPlaneImprovementWorkspace {
		return fmt.Errorf("mission store improvement workspace run run_id %q execution_plane must be %q", run.RunID, ExecutionPlaneImprovementWorkspace)
	}
	if run.CandidateID != record.CandidateID {
		return fmt.Errorf("mission store improvement workspace run candidate_id %q does not match run candidate_id %q", record.CandidateID, run.CandidateID)
	}
	if run.ExecutionHost != record.ExecutionHost {
		return fmt.Errorf("mission store improvement workspace run execution_host %q does not match run execution_host %q", record.ExecutionHost, run.ExecutionHost)
	}
	if _, err := LoadImprovementCandidateRecord(root, record.CandidateID); err != nil {
		return fmt.Errorf("mission store improvement workspace run candidate_id %q: %w", record.CandidateID, err)
	}
	if requireCurrentCompletionPointer {
		currentPointer, err := LoadActiveRuntimePackPointer(root)
		if err != nil {
			return fmt.Errorf("mission store improvement workspace run active pointer: %w", err)
		}
		currentSnapshot := ImprovementWorkspaceActivePointerSnapshotFromPointer(currentPointer)
		if !reflect.DeepEqual(currentSnapshot, record.ActivePointerAtCompletion) {
			return fmt.Errorf("mission store improvement workspace run active pointer completion snapshot does not match current active pointer")
		}
	}
	return nil
}

func isValidImprovementWorkspaceRunOutcome(outcome ImprovementWorkspaceRunOutcome) bool {
	switch outcome {
	case ImprovementWorkspaceRunOutcomeCrashed, ImprovementWorkspaceRunOutcomeFailed:
		return true
	default:
		return false
	}
}
