package missioncontrol

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"time"
	"unicode"
)

type ImprovementRunState string

const (
	ImprovementRunStateQueued             ImprovementRunState = "queued"
	ImprovementRunStateBaselining         ImprovementRunState = "baselining"
	ImprovementRunStateMutating           ImprovementRunState = "mutating"
	ImprovementRunStateEvaluatingTrain    ImprovementRunState = "evaluating_train"
	ImprovementRunStateEvaluatingHoldout  ImprovementRunState = "evaluating_holdout"
	ImprovementRunStateCandidateReady     ImprovementRunState = "candidate_ready"
	ImprovementRunStateStagedForHotUpdate ImprovementRunState = "staged_for_hot_update"
	ImprovementRunStateCanarying          ImprovementRunState = "canarying"
	ImprovementRunStateHotUpdated         ImprovementRunState = "hot_updated"
	ImprovementRunStatePromoted           ImprovementRunState = "promoted"
	ImprovementRunStateRejected           ImprovementRunState = "rejected"
	ImprovementRunStateRolledBack         ImprovementRunState = "rolled_back"
	ImprovementRunStateFailed             ImprovementRunState = "failed"
	ImprovementRunStateAborted            ImprovementRunState = "aborted"
)

type ImprovementRunDecision string

const (
	ImprovementRunDecisionKeep       ImprovementRunDecision = "keep"
	ImprovementRunDecisionDiscard    ImprovementRunDecision = "discard"
	ImprovementRunDecisionBlocked    ImprovementRunDecision = "blocked"
	ImprovementRunDecisionCrash      ImprovementRunDecision = "crash"
	ImprovementRunDecisionHotUpdated ImprovementRunDecision = "hot_updated"
	ImprovementRunDecisionPromoted   ImprovementRunDecision = "promoted"
	ImprovementRunDecisionRolledBack ImprovementRunDecision = "rolled_back"
)

type ImprovementRunRef struct {
	RunID string `json:"run_id"`
}

type ImprovementRunRecord struct {
	RecordVersion   int                    `json:"record_version"`
	RunID           string                 `json:"run_id"`
	Objective       string                 `json:"objective"`
	ExecutionPlane  string                 `json:"execution_plane"`
	ExecutionHost   string                 `json:"execution_host"`
	MissionFamily   string                 `json:"mission_family"`
	TargetType      string                 `json:"target_type"`
	TargetRef       string                 `json:"target_ref"`
	SurfaceClass    string                 `json:"surface_class"`
	CandidateID     string                 `json:"candidate_id"`
	EvalSuiteID     string                 `json:"eval_suite_id"`
	BaselinePackID  string                 `json:"baseline_pack_id"`
	CandidatePackID string                 `json:"candidate_pack_id"`
	HotUpdateID     string                 `json:"hot_update_id,omitempty"`
	State           ImprovementRunState    `json:"state"`
	Decision        ImprovementRunDecision `json:"decision,omitempty"`
	CreatedAt       time.Time              `json:"created_at"`
	CompletedAt     time.Time              `json:"completed_at,omitempty"`
	StopReason      string                 `json:"stop_reason,omitempty"`
	CreatedBy       string                 `json:"created_by"`
}

var ErrImprovementRunRecordNotFound = errors.New("mission store improvement run record not found")

func StoreImprovementRunsDir(root string) string {
	return filepath.Join(root, "runtime_packs", "improvement_runs")
}

func StoreImprovementRunPath(root, runID string) string {
	return filepath.Join(StoreImprovementRunsDir(root), strings.TrimSpace(runID)+".json")
}

func NormalizeImprovementRunRef(ref ImprovementRunRef) ImprovementRunRef {
	ref.RunID = strings.TrimSpace(ref.RunID)
	return ref
}

func NormalizeImprovementRunRecord(record ImprovementRunRecord) ImprovementRunRecord {
	record.RunID = strings.TrimSpace(record.RunID)
	record.Objective = strings.TrimSpace(record.Objective)
	record.ExecutionPlane = strings.TrimSpace(record.ExecutionPlane)
	record.ExecutionHost = strings.TrimSpace(record.ExecutionHost)
	record.MissionFamily = strings.TrimSpace(record.MissionFamily)
	record.TargetType = strings.TrimSpace(record.TargetType)
	record.TargetRef = strings.TrimSpace(record.TargetRef)
	record.SurfaceClass = strings.TrimSpace(record.SurfaceClass)
	record.CandidateID = strings.TrimSpace(record.CandidateID)
	record.EvalSuiteID = strings.TrimSpace(record.EvalSuiteID)
	record.BaselinePackID = strings.TrimSpace(record.BaselinePackID)
	record.CandidatePackID = strings.TrimSpace(record.CandidatePackID)
	record.HotUpdateID = strings.TrimSpace(record.HotUpdateID)
	record.CreatedAt = record.CreatedAt.UTC()
	record.CompletedAt = record.CompletedAt.UTC()
	record.StopReason = strings.TrimSpace(record.StopReason)
	record.CreatedBy = strings.TrimSpace(record.CreatedBy)
	return record
}

func ImprovementRunCandidateRef(record ImprovementRunRecord) ImprovementCandidateRef {
	return ImprovementCandidateRef{CandidateID: strings.TrimSpace(record.CandidateID)}
}

func ImprovementRunEvalSuiteRef(record ImprovementRunRecord) EvalSuiteRef {
	return EvalSuiteRef{EvalSuiteID: strings.TrimSpace(record.EvalSuiteID)}
}

func ImprovementRunBaselinePackRef(record ImprovementRunRecord) RuntimePackRef {
	return RuntimePackRef{PackID: strings.TrimSpace(record.BaselinePackID)}
}

func ImprovementRunCandidatePackRef(record ImprovementRunRecord) RuntimePackRef {
	return RuntimePackRef{PackID: strings.TrimSpace(record.CandidatePackID)}
}

func ImprovementRunHotUpdateGateRef(record ImprovementRunRecord) (HotUpdateGateRef, bool) {
	hotUpdateID := strings.TrimSpace(record.HotUpdateID)
	if hotUpdateID == "" {
		return HotUpdateGateRef{}, false
	}
	return HotUpdateGateRef{HotUpdateID: hotUpdateID}, true
}

func ValidateImprovementRunRef(ref ImprovementRunRef) error {
	return validateImprovementRunIdentifierField("improvement run ref", "run_id", ref.RunID)
}

func ValidateImprovementRunRecord(record ImprovementRunRecord) error {
	if record.RecordVersion <= 0 {
		return fmt.Errorf("mission store improvement run record_version must be positive")
	}
	if err := ValidateImprovementRunRef(ImprovementRunRef{RunID: record.RunID}); err != nil {
		return err
	}
	if record.Objective == "" {
		return fmt.Errorf("mission store improvement run objective is required")
	}
	if record.ExecutionPlane == "" {
		return fmt.Errorf("mission store improvement run execution_plane is required")
	}
	if record.ExecutionHost == "" {
		return fmt.Errorf("mission store improvement run execution_host is required")
	}
	if record.MissionFamily == "" {
		return fmt.Errorf("mission store improvement run mission_family is required")
	}
	if record.TargetType == "" {
		return fmt.Errorf("mission store improvement run target_type is required")
	}
	if record.TargetRef == "" {
		return fmt.Errorf("mission store improvement run target_ref is required")
	}
	if record.SurfaceClass == "" {
		return fmt.Errorf("mission store improvement run surface_class is required")
	}
	if err := ValidateImprovementCandidateRef(ImprovementRunCandidateRef(record)); err != nil {
		return fmt.Errorf("mission store improvement run candidate_id %q: %w", record.CandidateID, err)
	}
	if err := ValidateEvalSuiteRef(ImprovementRunEvalSuiteRef(record)); err != nil {
		return fmt.Errorf("mission store improvement run eval_suite_id %q: %w", record.EvalSuiteID, err)
	}
	if err := ValidateRuntimePackRef(ImprovementRunBaselinePackRef(record)); err != nil {
		return fmt.Errorf("mission store improvement run baseline_pack_id %q: %w", record.BaselinePackID, err)
	}
	if err := ValidateRuntimePackRef(ImprovementRunCandidatePackRef(record)); err != nil {
		return fmt.Errorf("mission store improvement run candidate_pack_id %q: %w", record.CandidatePackID, err)
	}
	if gateRef, ok := ImprovementRunHotUpdateGateRef(record); ok {
		if err := ValidateHotUpdateGateRef(gateRef); err != nil {
			return fmt.Errorf("mission store improvement run hot_update_id %q: %w", record.HotUpdateID, err)
		}
	}
	if !isValidImprovementRunState(record.State) {
		return fmt.Errorf("mission store improvement run state %q is invalid", record.State)
	}
	if record.Decision != "" && !isValidImprovementRunDecision(record.Decision) {
		return fmt.Errorf("mission store improvement run decision %q is invalid", record.Decision)
	}
	if record.CreatedAt.IsZero() {
		return fmt.Errorf("mission store improvement run created_at is required")
	}
	if record.CreatedBy == "" {
		return fmt.Errorf("mission store improvement run created_by is required")
	}
	if !record.CompletedAt.IsZero() && record.Decision == "" {
		return fmt.Errorf("mission store improvement run completed_at requires decision")
	}
	return nil
}

func StoreImprovementRunRecord(root string, record ImprovementRunRecord) error {
	if err := ValidateStoreRoot(root); err != nil {
		return err
	}
	record = NormalizeImprovementRunRecord(record)
	record.RecordVersion = normalizeRecordVersion(record.RecordVersion)
	if err := ValidateImprovementRunRecord(record); err != nil {
		return err
	}
	if err := validateImprovementRunLinkage(root, record); err != nil {
		return err
	}

	path := StoreImprovementRunPath(root, record.RunID)
	if existing, err := loadImprovementRunRecordFile(root, path); err == nil {
		if reflect.DeepEqual(existing, record) {
			return nil
		}
		return fmt.Errorf("mission store improvement run %q already exists", record.RunID)
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	return WriteStoreJSONAtomic(path, record)
}

func LoadImprovementRunRecord(root, runID string) (ImprovementRunRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return ImprovementRunRecord{}, err
	}
	ref := NormalizeImprovementRunRef(ImprovementRunRef{RunID: runID})
	if err := ValidateImprovementRunRef(ref); err != nil {
		return ImprovementRunRecord{}, err
	}
	record, err := loadImprovementRunRecordFile(root, StoreImprovementRunPath(root, ref.RunID))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return ImprovementRunRecord{}, ErrImprovementRunRecordNotFound
		}
		return ImprovementRunRecord{}, err
	}
	return record, nil
}

func ListImprovementRunRecords(root string) ([]ImprovementRunRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(StoreImprovementRunsDir(root))
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

	records := make([]ImprovementRunRecord, 0, len(names))
	for _, name := range names {
		record, err := loadImprovementRunRecordFile(root, filepath.Join(StoreImprovementRunsDir(root), name))
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, nil
}

func loadImprovementRunRecordFile(root, path string) (ImprovementRunRecord, error) {
	var record ImprovementRunRecord
	if err := LoadStoreJSON(path, &record); err != nil {
		return ImprovementRunRecord{}, err
	}
	record = NormalizeImprovementRunRecord(record)
	if err := ValidateImprovementRunRecord(record); err != nil {
		return ImprovementRunRecord{}, err
	}
	if err := validateImprovementRunLinkage(root, record); err != nil {
		return ImprovementRunRecord{}, err
	}
	return record, nil
}

func validateImprovementRunLinkage(root string, record ImprovementRunRecord) error {
	candidate, err := LoadImprovementCandidateRecord(root, record.CandidateID)
	if err != nil {
		return fmt.Errorf("mission store improvement run candidate_id %q: %w", record.CandidateID, err)
	}
	if candidate.BaselinePackID != record.BaselinePackID {
		return fmt.Errorf(
			"mission store improvement run baseline_pack_id %q does not match candidate baseline_pack_id %q",
			record.BaselinePackID,
			candidate.BaselinePackID,
		)
	}
	if candidate.CandidatePackID != record.CandidatePackID {
		return fmt.Errorf(
			"mission store improvement run candidate_pack_id %q does not match candidate candidate_pack_id %q",
			record.CandidatePackID,
			candidate.CandidatePackID,
		)
	}

	evalSuite, err := LoadEvalSuiteRecord(root, record.EvalSuiteID)
	if err != nil {
		return fmt.Errorf("mission store improvement run eval_suite_id %q: %w", record.EvalSuiteID, err)
	}
	if evalSuite.CandidateID != "" && evalSuite.CandidateID != record.CandidateID {
		return fmt.Errorf(
			"mission store improvement run eval_suite_id %q candidate_id %q does not match run candidate_id %q",
			record.EvalSuiteID,
			evalSuite.CandidateID,
			record.CandidateID,
		)
	}
	if evalSuite.BaselinePackID != "" && evalSuite.BaselinePackID != record.BaselinePackID {
		return fmt.Errorf(
			"mission store improvement run eval_suite_id %q baseline_pack_id %q does not match run baseline_pack_id %q",
			record.EvalSuiteID,
			evalSuite.BaselinePackID,
			record.BaselinePackID,
		)
	}
	if evalSuite.CandidatePackID != "" && evalSuite.CandidatePackID != record.CandidatePackID {
		return fmt.Errorf(
			"mission store improvement run eval_suite_id %q candidate_pack_id %q does not match run candidate_pack_id %q",
			record.EvalSuiteID,
			evalSuite.CandidatePackID,
			record.CandidatePackID,
		)
	}

	if _, err := LoadRuntimePackRecord(root, record.BaselinePackID); err != nil {
		return fmt.Errorf("mission store improvement run baseline_pack_id %q: %w", record.BaselinePackID, err)
	}
	if _, err := LoadRuntimePackRecord(root, record.CandidatePackID); err != nil {
		return fmt.Errorf("mission store improvement run candidate_pack_id %q: %w", record.CandidatePackID, err)
	}

	if gateRef, ok := ImprovementRunHotUpdateGateRef(record); ok {
		gate, err := LoadHotUpdateGateRecord(root, gateRef.HotUpdateID)
		if err != nil {
			return fmt.Errorf("mission store improvement run hot_update_id %q: %w", gateRef.HotUpdateID, err)
		}
		if gate.CandidatePackID != record.CandidatePackID {
			return fmt.Errorf(
				"mission store improvement run hot_update_id %q candidate_pack_id %q does not match run candidate_pack_id %q",
				gateRef.HotUpdateID,
				gate.CandidatePackID,
				record.CandidatePackID,
			)
		}
		if gate.PreviousActivePackID != record.BaselinePackID {
			return fmt.Errorf(
				"mission store improvement run hot_update_id %q previous_active_pack_id %q does not match run baseline_pack_id %q",
				gateRef.HotUpdateID,
				gate.PreviousActivePackID,
				record.BaselinePackID,
			)
		}
	}

	return nil
}

func validateImprovementRunIdentifierField(surface, fieldName, value string) error {
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

func isValidImprovementRunState(state ImprovementRunState) bool {
	switch state {
	case ImprovementRunStateQueued,
		ImprovementRunStateBaselining,
		ImprovementRunStateMutating,
		ImprovementRunStateEvaluatingTrain,
		ImprovementRunStateEvaluatingHoldout,
		ImprovementRunStateCandidateReady,
		ImprovementRunStateStagedForHotUpdate,
		ImprovementRunStateCanarying,
		ImprovementRunStateHotUpdated,
		ImprovementRunStatePromoted,
		ImprovementRunStateRejected,
		ImprovementRunStateRolledBack,
		ImprovementRunStateFailed,
		ImprovementRunStateAborted:
		return true
	default:
		return false
	}
}

func isValidImprovementRunDecision(decision ImprovementRunDecision) bool {
	switch decision {
	case ImprovementRunDecisionKeep,
		ImprovementRunDecisionDiscard,
		ImprovementRunDecisionBlocked,
		ImprovementRunDecisionCrash,
		ImprovementRunDecisionHotUpdated,
		ImprovementRunDecisionPromoted,
		ImprovementRunDecisionRolledBack:
		return true
	default:
		return false
	}
}
