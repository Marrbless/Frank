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

type HotUpdateSmokeCheckState string

const (
	HotUpdateSmokeCheckStatePassed  HotUpdateSmokeCheckState = "passed"
	HotUpdateSmokeCheckStateFailed  HotUpdateSmokeCheckState = "failed"
	HotUpdateSmokeCheckStateBlocked HotUpdateSmokeCheckState = "blocked"
)

type HotUpdateSmokeCheckRef struct {
	SmokeCheckID string `json:"smoke_check_id"`
}

type HotUpdateSmokeCheckRecord struct {
	RecordVersion   int                      `json:"record_version"`
	SmokeCheckID    string                   `json:"smoke_check_id"`
	HotUpdateID     string                   `json:"hot_update_id"`
	CandidatePackID string                   `json:"candidate_pack_id"`
	EvidenceState   HotUpdateSmokeCheckState `json:"evidence_state"`
	Passed          bool                     `json:"passed"`
	Reason          string                   `json:"reason"`
	ObservedAt      time.Time                `json:"observed_at"`
	CreatedAt       time.Time                `json:"created_at"`
	CreatedBy       string                   `json:"created_by"`
}

type HotUpdateSmokeReadinessAssessment struct {
	State                string                   `json:"state"`
	HotUpdateID          string                   `json:"hot_update_id,omitempty"`
	CandidatePackID      string                   `json:"candidate_pack_id,omitempty"`
	SmokeCheckRefs       []string                 `json:"smoke_check_refs,omitempty"`
	SelectedSmokeCheckID string                   `json:"selected_smoke_check_id,omitempty"`
	EvidenceState        HotUpdateSmokeCheckState `json:"evidence_state,omitempty"`
	Ready                bool                     `json:"ready"`
	Reason               string                   `json:"reason,omitempty"`
	Error                string                   `json:"error,omitempty"`
}

var ErrHotUpdateSmokeCheckRecordNotFound = errors.New("mission store hot-update smoke check record not found")

func StoreHotUpdateSmokeChecksDir(root string) string {
	return filepath.Join(root, "runtime_packs", "hot_update_smoke_checks")
}

func StoreHotUpdateSmokeCheckPath(root, smokeCheckID string) string {
	return filepath.Join(StoreHotUpdateSmokeChecksDir(root), strings.TrimSpace(smokeCheckID)+".json")
}

func HotUpdateSmokeCheckIDFromGateObservedAt(hotUpdateID string, observedAt time.Time) string {
	observedAt = observedAt.UTC()
	observedAtCompact := strings.ReplaceAll(observedAt.Format("20060102T150405.000000000Z"), ".", "")
	return "hot-update-smoke-check-" + strings.TrimSpace(hotUpdateID) + "-" + observedAtCompact
}

func NormalizeHotUpdateSmokeCheckRef(ref HotUpdateSmokeCheckRef) HotUpdateSmokeCheckRef {
	ref.SmokeCheckID = strings.TrimSpace(ref.SmokeCheckID)
	return ref
}

func NormalizeHotUpdateSmokeCheckRecord(record HotUpdateSmokeCheckRecord) HotUpdateSmokeCheckRecord {
	record.SmokeCheckID = strings.TrimSpace(record.SmokeCheckID)
	record.HotUpdateID = strings.TrimSpace(record.HotUpdateID)
	record.CandidatePackID = strings.TrimSpace(record.CandidatePackID)
	record.EvidenceState = HotUpdateSmokeCheckState(strings.TrimSpace(string(record.EvidenceState)))
	record.Reason = strings.TrimSpace(record.Reason)
	record.ObservedAt = record.ObservedAt.UTC()
	record.CreatedAt = record.CreatedAt.UTC()
	record.CreatedBy = strings.TrimSpace(record.CreatedBy)
	return record
}

func ValidateHotUpdateSmokeCheckRef(ref HotUpdateSmokeCheckRef) error {
	return validateHotUpdateIdentifierField("hot-update smoke check ref", "smoke_check_id", ref.SmokeCheckID)
}

func ValidateHotUpdateSmokeCheckRecord(record HotUpdateSmokeCheckRecord) error {
	if record.RecordVersion <= 0 {
		return fmt.Errorf("mission store hot-update smoke check record_version must be positive")
	}
	if err := ValidateHotUpdateSmokeCheckRef(HotUpdateSmokeCheckRef{SmokeCheckID: record.SmokeCheckID}); err != nil {
		return err
	}
	if err := ValidateHotUpdateGateRef(HotUpdateGateRef{HotUpdateID: record.HotUpdateID}); err != nil {
		return fmt.Errorf("mission store hot-update smoke check hot_update_id %q: %w", record.HotUpdateID, err)
	}
	if err := ValidateRuntimePackRef(RuntimePackRef{PackID: record.CandidatePackID}); err != nil {
		return fmt.Errorf("mission store hot-update smoke check candidate_pack_id %q: %w", record.CandidatePackID, err)
	}
	if record.ObservedAt.IsZero() {
		return fmt.Errorf("mission store hot-update smoke check observed_at is required")
	}
	if record.SmokeCheckID != HotUpdateSmokeCheckIDFromGateObservedAt(record.HotUpdateID, record.ObservedAt) {
		return fmt.Errorf("mission store hot-update smoke check smoke_check_id %q does not match deterministic smoke_check_id %q", record.SmokeCheckID, HotUpdateSmokeCheckIDFromGateObservedAt(record.HotUpdateID, record.ObservedAt))
	}
	if !isValidHotUpdateSmokeCheckState(record.EvidenceState) {
		return fmt.Errorf("mission store hot-update smoke check evidence_state %q is invalid", record.EvidenceState)
	}
	if record.Passed != (record.EvidenceState == HotUpdateSmokeCheckStatePassed) {
		return fmt.Errorf("mission store hot-update smoke check passed must be true only when evidence_state is %q", HotUpdateSmokeCheckStatePassed)
	}
	if record.Reason == "" {
		return fmt.Errorf("mission store hot-update smoke check reason is required")
	}
	if record.CreatedAt.IsZero() {
		return fmt.Errorf("mission store hot-update smoke check created_at is required")
	}
	if record.CreatedBy == "" {
		return fmt.Errorf("mission store hot-update smoke check created_by is required")
	}
	return nil
}

func StoreHotUpdateSmokeCheckRecord(root string, record HotUpdateSmokeCheckRecord) (HotUpdateSmokeCheckRecord, bool, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return HotUpdateSmokeCheckRecord{}, false, err
	}
	record = NormalizeHotUpdateSmokeCheckRecord(record)
	record.RecordVersion = normalizeRecordVersion(record.RecordVersion)
	if err := ValidateHotUpdateSmokeCheckRecord(record); err != nil {
		return HotUpdateSmokeCheckRecord{}, false, err
	}
	if err := validateHotUpdateSmokeCheckLinkage(root, record); err != nil {
		return HotUpdateSmokeCheckRecord{}, false, err
	}

	path := StoreHotUpdateSmokeCheckPath(root, record.SmokeCheckID)
	existing, err := loadHotUpdateSmokeCheckRecordFile(root, path)
	if err == nil {
		if reflect.DeepEqual(existing, record) {
			return existing, false, nil
		}
		return HotUpdateSmokeCheckRecord{}, false, fmt.Errorf("mission store hot-update smoke check %q already exists", record.SmokeCheckID)
	}
	if !errors.Is(err, os.ErrNotExist) {
		return HotUpdateSmokeCheckRecord{}, false, err
	}
	if err := WriteStoreJSONAtomic(path, record); err != nil {
		return HotUpdateSmokeCheckRecord{}, false, err
	}
	stored, err := LoadHotUpdateSmokeCheckRecord(root, record.SmokeCheckID)
	if err != nil {
		return HotUpdateSmokeCheckRecord{}, false, err
	}
	return stored, true, nil
}

func LoadHotUpdateSmokeCheckRecord(root, smokeCheckID string) (HotUpdateSmokeCheckRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return HotUpdateSmokeCheckRecord{}, err
	}
	ref := NormalizeHotUpdateSmokeCheckRef(HotUpdateSmokeCheckRef{SmokeCheckID: smokeCheckID})
	if err := ValidateHotUpdateSmokeCheckRef(ref); err != nil {
		return HotUpdateSmokeCheckRecord{}, err
	}
	record, err := loadHotUpdateSmokeCheckRecordFile(root, StoreHotUpdateSmokeCheckPath(root, ref.SmokeCheckID))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return HotUpdateSmokeCheckRecord{}, ErrHotUpdateSmokeCheckRecordNotFound
		}
		return HotUpdateSmokeCheckRecord{}, err
	}
	return record, nil
}

func ListHotUpdateSmokeCheckRecords(root string) ([]HotUpdateSmokeCheckRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return nil, err
	}
	return listStoreJSONRecords(StoreHotUpdateSmokeChecksDir(root), func(path string) (HotUpdateSmokeCheckRecord, error) {
		return loadHotUpdateSmokeCheckRecordFile(root, path)
	})
}

func CreateHotUpdateSmokeCheckFromGate(root, hotUpdateID string, state HotUpdateSmokeCheckState, observedAt time.Time, createdBy string, createdAt time.Time, reason string) (HotUpdateSmokeCheckRecord, bool, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return HotUpdateSmokeCheckRecord{}, false, err
	}
	ref := NormalizeHotUpdateGateRef(HotUpdateGateRef{HotUpdateID: hotUpdateID})
	if err := ValidateHotUpdateGateRef(ref); err != nil {
		return HotUpdateSmokeCheckRecord{}, false, err
	}
	state = HotUpdateSmokeCheckState(strings.TrimSpace(string(state)))
	if !isValidHotUpdateSmokeCheckState(state) {
		return HotUpdateSmokeCheckRecord{}, false, fmt.Errorf("mission store hot-update smoke check evidence_state %q is invalid", state)
	}
	observedAt = observedAt.UTC()
	if observedAt.IsZero() {
		return HotUpdateSmokeCheckRecord{}, false, fmt.Errorf("mission store hot-update smoke check observed_at is required")
	}
	createdBy = strings.TrimSpace(createdBy)
	if createdBy == "" {
		return HotUpdateSmokeCheckRecord{}, false, fmt.Errorf("mission store hot-update smoke check created_by is required")
	}
	createdAt = createdAt.UTC()
	if createdAt.IsZero() {
		return HotUpdateSmokeCheckRecord{}, false, fmt.Errorf("mission store hot-update smoke check created_at is required")
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return HotUpdateSmokeCheckRecord{}, false, fmt.Errorf("mission store hot-update smoke check reason is required")
	}

	gate, err := LoadHotUpdateGateRecord(root, ref.HotUpdateID)
	if err != nil {
		return HotUpdateSmokeCheckRecord{}, false, err
	}
	record := NormalizeHotUpdateSmokeCheckRecord(HotUpdateSmokeCheckRecord{
		RecordVersion:   StoreRecordVersion,
		SmokeCheckID:    HotUpdateSmokeCheckIDFromGateObservedAt(gate.HotUpdateID, observedAt),
		HotUpdateID:     gate.HotUpdateID,
		CandidatePackID: gate.CandidatePackID,
		EvidenceState:   state,
		Passed:          state == HotUpdateSmokeCheckStatePassed,
		Reason:          reason,
		ObservedAt:      observedAt,
		CreatedAt:       createdAt,
		CreatedBy:       createdBy,
	})
	stored, changed, err := StoreHotUpdateSmokeCheckRecord(root, record)
	if err != nil {
		return HotUpdateSmokeCheckRecord{}, false, err
	}

	gate, err = LoadHotUpdateGateRecord(root, gate.HotUpdateID)
	if err != nil {
		return HotUpdateSmokeCheckRecord{}, false, err
	}
	if !containsHotUpdateString(gate.SmokeCheckRefs, stored.SmokeCheckID) {
		gate.SmokeCheckRefs = append(gate.SmokeCheckRefs, stored.SmokeCheckID)
		if err := StoreHotUpdateGateRecord(root, gate); err != nil {
			return HotUpdateSmokeCheckRecord{}, false, err
		}
		changed = true
	}
	return stored, changed, nil
}

func AssessHotUpdateSmokeReadiness(root string, hotUpdateID string) (HotUpdateSmokeReadinessAssessment, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return HotUpdateSmokeReadinessAssessment{}, err
	}
	ref := NormalizeHotUpdateGateRef(HotUpdateGateRef{HotUpdateID: hotUpdateID})
	if err := ValidateHotUpdateGateRef(ref); err != nil {
		return HotUpdateSmokeReadinessAssessment{}, err
	}
	gate, err := LoadHotUpdateGateRecord(root, ref.HotUpdateID)
	if err != nil {
		return HotUpdateSmokeReadinessAssessment{}, err
	}
	assessment := hotUpdateSmokeReadinessAssessmentFromGate(gate)
	if !hotUpdateGateRequiresSmoke(gate) {
		assessment.State = "not_required"
		assessment.Ready = true
		assessment.Reason = "hot-update gate surface classes do not require smoke evidence"
		return assessment, nil
	}
	if len(gate.SmokeCheckRefs) == 0 {
		assessment.State = "missing"
		assessment.Ready = false
		assessment.Reason = string(RejectionCodeV4SmokeCheckRequired) + ": hot-update gate has no smoke_check_refs"
		return assessment, nil
	}

	selectedSmokeCheckID := gate.SmokeCheckRefs[len(gate.SmokeCheckRefs)-1]
	smoke, err := LoadHotUpdateSmokeCheckRecord(root, selectedSmokeCheckID)
	if err != nil {
		assessment.State = "invalid"
		assessment.Ready = false
		assessment.Reason = string(RejectionCodeV4SmokeCheckRequired) + ": selected smoke check ref is invalid"
		assessment.Error = err.Error()
		return assessment, err
	}
	assessment.SelectedSmokeCheckID = smoke.SmokeCheckID
	assessment.EvidenceState = smoke.EvidenceState
	if smoke.HotUpdateID != gate.HotUpdateID || smoke.CandidatePackID != gate.CandidatePackID {
		err := fmt.Errorf("mission store hot-update smoke check %q does not match hot-update gate %q", smoke.SmokeCheckID, gate.HotUpdateID)
		assessment.State = "invalid"
		assessment.Ready = false
		assessment.Reason = string(RejectionCodeV4SmokeCheckRequired) + ": selected smoke check does not match gate"
		assessment.Error = err.Error()
		return assessment, err
	}
	if !smoke.Passed {
		assessment.State = "failed"
		assessment.Ready = false
		assessment.Reason = string(RejectionCodeV4SmokeCheckFailed) + ": " + smoke.Reason
		return assessment, nil
	}
	assessment.State = "ready"
	assessment.Ready = true
	assessment.Reason = "selected smoke check passed"
	return assessment, nil
}

func requireHotUpdateSmokeReadiness(root string, record HotUpdateGateRecord) error {
	assessment, err := AssessHotUpdateSmokeReadiness(root, record.HotUpdateID)
	if err != nil {
		return err
	}
	if !assessment.Ready {
		return fmt.Errorf("mission store hot-update gate %q smoke readiness is not ready: %s", record.HotUpdateID, assessment.Reason)
	}
	return nil
}

func loadHotUpdateSmokeCheckRecordFile(root, path string) (HotUpdateSmokeCheckRecord, error) {
	var record HotUpdateSmokeCheckRecord
	if err := LoadStoreJSON(path, &record); err != nil {
		return HotUpdateSmokeCheckRecord{}, err
	}
	record = NormalizeHotUpdateSmokeCheckRecord(record)
	if err := ValidateHotUpdateSmokeCheckRecord(record); err != nil {
		return HotUpdateSmokeCheckRecord{}, err
	}
	if err := validateHotUpdateSmokeCheckLinkage(root, record); err != nil {
		return HotUpdateSmokeCheckRecord{}, err
	}
	return record, nil
}

func validateHotUpdateSmokeCheckLinkage(root string, record HotUpdateSmokeCheckRecord) error {
	gate, err := LoadHotUpdateGateRecord(root, record.HotUpdateID)
	if err != nil {
		return fmt.Errorf("mission store hot-update smoke check hot_update_id %q: %w", record.HotUpdateID, err)
	}
	if gate.CandidatePackID != record.CandidatePackID {
		return fmt.Errorf("mission store hot-update smoke check %q candidate_pack_id %q does not match hot-update gate candidate_pack_id %q", record.SmokeCheckID, record.CandidatePackID, gate.CandidatePackID)
	}
	if _, err := LoadRuntimePackRecord(root, record.CandidatePackID); err != nil {
		return fmt.Errorf("mission store hot-update smoke check candidate_pack_id %q: %w", record.CandidatePackID, err)
	}
	return nil
}

func hotUpdateSmokeReadinessAssessmentFromGate(gate HotUpdateGateRecord) HotUpdateSmokeReadinessAssessment {
	return HotUpdateSmokeReadinessAssessment{
		HotUpdateID:     gate.HotUpdateID,
		CandidatePackID: gate.CandidatePackID,
		SmokeCheckRefs:  append([]string(nil), gate.SmokeCheckRefs...),
	}
}

func hotUpdateGateRequiresSmoke(gate HotUpdateGateRecord) bool {
	for _, class := range gate.SurfaceClasses {
		if strings.EqualFold(strings.TrimSpace(class), "class_0") {
			continue
		}
		if strings.TrimSpace(class) != "" {
			return true
		}
	}
	return false
}

func containsHotUpdateString(values []string, value string) bool {
	value = strings.TrimSpace(value)
	for _, existing := range values {
		if strings.TrimSpace(existing) == value {
			return true
		}
	}
	return false
}

func isValidHotUpdateSmokeCheckState(state HotUpdateSmokeCheckState) bool {
	switch state {
	case HotUpdateSmokeCheckStatePassed,
		HotUpdateSmokeCheckStateFailed,
		HotUpdateSmokeCheckStateBlocked:
		return true
	default:
		return false
	}
}
