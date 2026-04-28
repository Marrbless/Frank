package missioncontrol

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type HotUpdateCanarySatisfactionState string

const (
	HotUpdateCanarySatisfactionStateNotSatisfied         HotUpdateCanarySatisfactionState = "not_satisfied"
	HotUpdateCanarySatisfactionStateSatisfied            HotUpdateCanarySatisfactionState = "satisfied"
	HotUpdateCanarySatisfactionStateWaitingOwnerApproval HotUpdateCanarySatisfactionState = "waiting_owner_approval"
	HotUpdateCanarySatisfactionStateFailed               HotUpdateCanarySatisfactionState = "failed"
	HotUpdateCanarySatisfactionStateBlocked              HotUpdateCanarySatisfactionState = "blocked"
	HotUpdateCanarySatisfactionStateExpired              HotUpdateCanarySatisfactionState = "expired"
	HotUpdateCanarySatisfactionStateInvalid              HotUpdateCanarySatisfactionState = "invalid"
)

type HotUpdateCanarySatisfactionAssessment struct {
	State                     string                           `json:"state"`
	CanaryRequirementID       string                           `json:"canary_requirement_id,omitempty"`
	SelectedCanaryEvidenceID  string                           `json:"selected_canary_evidence_id,omitempty"`
	ResultID                  string                           `json:"result_id,omitempty"`
	RunID                     string                           `json:"run_id,omitempty"`
	CandidateID               string                           `json:"candidate_id,omitempty"`
	EvalSuiteID               string                           `json:"eval_suite_id,omitempty"`
	PromotionPolicyID         string                           `json:"promotion_policy_id,omitempty"`
	BaselinePackID            string                           `json:"baseline_pack_id,omitempty"`
	CandidatePackID           string                           `json:"candidate_pack_id,omitempty"`
	EligibilityState          string                           `json:"eligibility_state,omitempty"`
	CanaryScopeJobRefs        []string                         `json:"canary_scope_job_refs,omitempty"`
	CanaryScopeSurfaces       []string                         `json:"canary_scope_surfaces,omitempty"`
	OwnerApprovalRequired     bool                             `json:"owner_approval_required"`
	SatisfactionState         HotUpdateCanarySatisfactionState `json:"satisfaction_state,omitempty"`
	EvidenceSource            HotUpdateCanaryEvidenceSource    `json:"evidence_source,omitempty"`
	AutomaticTrafficExercised bool                             `json:"automatic_traffic_exercised,omitempty"`
	ExercisedJobRefs          []string                         `json:"exercised_job_refs,omitempty"`
	ExercisedSurfaces         []string                         `json:"exercised_surfaces,omitempty"`
	EvidenceState             HotUpdateCanaryEvidenceState     `json:"evidence_state,omitempty"`
	Passed                    bool                             `json:"passed"`
	ObservedAt                time.Time                        `json:"observed_at,omitempty"`
	Reason                    string                           `json:"reason,omitempty"`
	Error                     string                           `json:"error,omitempty"`
}

type hotUpdateCanarySatisfactionEvidenceCandidate struct {
	record     HotUpdateCanaryEvidenceRecord
	evidenceID string
	err        error
}

func AssessHotUpdateCanarySatisfaction(root, canaryRequirementID string) (HotUpdateCanarySatisfactionAssessment, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return HotUpdateCanarySatisfactionAssessment{}, err
	}
	ref := NormalizeHotUpdateCanaryRequirementRef(HotUpdateCanaryRequirementRef{CanaryRequirementID: canaryRequirementID})
	if err := ValidateHotUpdateCanaryRequirementRef(ref); err != nil {
		return hotUpdateCanaryInvalidSatisfactionAssessment(HotUpdateCanaryRequirementRecord{CanaryRequirementID: ref.CanaryRequirementID}, err), nil
	}
	requirement, err := LoadHotUpdateCanaryRequirementRecord(root, ref.CanaryRequirementID)
	if err != nil {
		return hotUpdateCanaryInvalidSatisfactionAssessment(HotUpdateCanaryRequirementRecord{CanaryRequirementID: ref.CanaryRequirementID}, err), nil
	}
	return assessHotUpdateCanarySatisfactionForRequirement(root, requirement)
}

func assessHotUpdateCanarySatisfactionForRequirement(root string, requirement HotUpdateCanaryRequirementRecord) (HotUpdateCanarySatisfactionAssessment, error) {
	requirement = NormalizeHotUpdateCanaryRequirementRecord(requirement)
	if err := ValidateHotUpdateCanaryRequirementRecord(requirement); err != nil {
		return hotUpdateCanaryInvalidSatisfactionAssessment(requirement, err), nil
	}
	if err := validateHotUpdateCanaryRequirementLinkage(root, requirement); err != nil {
		return hotUpdateCanaryInvalidSatisfactionAssessment(requirement, err), nil
	}

	valid, invalid, err := loadHotUpdateCanarySatisfactionEvidenceCandidates(root, requirement)
	if err != nil {
		return HotUpdateCanarySatisfactionAssessment{}, err
	}
	if len(valid) == 0 {
		if len(invalid) > 0 {
			return hotUpdateCanaryInvalidSatisfactionAssessmentFromEvidence(requirement, invalid[0]), nil
		}
		assessment := hotUpdateCanaryBaseSatisfactionAssessment(requirement)
		assessment.State = "configured"
		assessment.SatisfactionState = HotUpdateCanarySatisfactionStateNotSatisfied
		assessment.Reason = "no valid hot-update canary evidence exists for requirement"
		return assessment, nil
	}

	selected := selectHotUpdateCanarySatisfactionEvidence(valid)
	assessment := hotUpdateCanaryBaseSatisfactionAssessment(requirement)
	assessment.State = "configured"
	assessment.SelectedCanaryEvidenceID = selected.CanaryEvidenceID
	assessment.EvidenceSource = selected.EvidenceSource
	assessment.AutomaticTrafficExercised = selected.AutomaticTrafficExercised
	assessment.ExercisedJobRefs = append([]string(nil), selected.ExercisedJobRefs...)
	assessment.ExercisedSurfaces = append([]string(nil), selected.ExercisedSurfaces...)
	assessment.EvidenceState = selected.EvidenceState
	assessment.Passed = selected.Passed
	assessment.ObservedAt = selected.ObservedAt
	assessment.Reason = selected.Reason
	switch selected.EvidenceState {
	case HotUpdateCanaryEvidenceStatePassed:
		if requirement.OwnerApprovalRequired {
			assessment.SatisfactionState = HotUpdateCanarySatisfactionStateWaitingOwnerApproval
		} else {
			assessment.SatisfactionState = HotUpdateCanarySatisfactionStateSatisfied
		}
	case HotUpdateCanaryEvidenceStateFailed:
		assessment.SatisfactionState = HotUpdateCanarySatisfactionStateFailed
	case HotUpdateCanaryEvidenceStateBlocked:
		assessment.SatisfactionState = HotUpdateCanarySatisfactionStateBlocked
	case HotUpdateCanaryEvidenceStateExpired:
		assessment.SatisfactionState = HotUpdateCanarySatisfactionStateExpired
	default:
		assessment.State = "invalid"
		assessment.SatisfactionState = HotUpdateCanarySatisfactionStateInvalid
		assessment.Error = fmt.Sprintf("mission store hot-update canary evidence state %q is invalid", selected.EvidenceState)
	}
	return assessment, nil
}

func hotUpdateCanaryBaseSatisfactionAssessment(requirement HotUpdateCanaryRequirementRecord) HotUpdateCanarySatisfactionAssessment {
	return HotUpdateCanarySatisfactionAssessment{
		CanaryRequirementID:   requirement.CanaryRequirementID,
		ResultID:              requirement.ResultID,
		RunID:                 requirement.RunID,
		CandidateID:           requirement.CandidateID,
		EvalSuiteID:           requirement.EvalSuiteID,
		PromotionPolicyID:     requirement.PromotionPolicyID,
		BaselinePackID:        requirement.BaselinePackID,
		CandidatePackID:       requirement.CandidatePackID,
		EligibilityState:      requirement.EligibilityState,
		CanaryScopeJobRefs:    append([]string(nil), requirement.CanaryScopeJobRefs...),
		CanaryScopeSurfaces:   append([]string(nil), requirement.CanaryScopeSurfaces...),
		OwnerApprovalRequired: requirement.OwnerApprovalRequired,
	}
}

func hotUpdateCanaryInvalidSatisfactionAssessment(requirement HotUpdateCanaryRequirementRecord, err error) HotUpdateCanarySatisfactionAssessment {
	assessment := hotUpdateCanaryBaseSatisfactionAssessment(requirement)
	assessment.State = "invalid"
	assessment.SatisfactionState = HotUpdateCanarySatisfactionStateInvalid
	if err != nil {
		assessment.Error = err.Error()
	}
	return assessment
}

func hotUpdateCanaryInvalidSatisfactionAssessmentFromEvidence(requirement HotUpdateCanaryRequirementRecord, candidate hotUpdateCanarySatisfactionEvidenceCandidate) HotUpdateCanarySatisfactionAssessment {
	assessment := hotUpdateCanaryBaseSatisfactionAssessment(requirement)
	record := candidate.record
	if record.CanaryRequirementID != "" {
		assessment.CanaryRequirementID = record.CanaryRequirementID
	}
	assessment.SelectedCanaryEvidenceID = candidate.evidenceID
	if record.CanaryEvidenceID != "" {
		assessment.SelectedCanaryEvidenceID = record.CanaryEvidenceID
	}
	if record.ResultID != "" {
		assessment.ResultID = record.ResultID
	}
	if record.RunID != "" {
		assessment.RunID = record.RunID
	}
	if record.CandidateID != "" {
		assessment.CandidateID = record.CandidateID
	}
	if record.EvalSuiteID != "" {
		assessment.EvalSuiteID = record.EvalSuiteID
	}
	if record.PromotionPolicyID != "" {
		assessment.PromotionPolicyID = record.PromotionPolicyID
	}
	if record.BaselinePackID != "" {
		assessment.BaselinePackID = record.BaselinePackID
	}
	if record.CandidatePackID != "" {
		assessment.CandidatePackID = record.CandidatePackID
	}
	assessment.EvidenceState = record.EvidenceState
	assessment.EvidenceSource = record.EvidenceSource
	assessment.AutomaticTrafficExercised = record.AutomaticTrafficExercised
	assessment.ExercisedJobRefs = append([]string(nil), record.ExercisedJobRefs...)
	assessment.ExercisedSurfaces = append([]string(nil), record.ExercisedSurfaces...)
	assessment.Passed = record.Passed
	assessment.ObservedAt = record.ObservedAt
	assessment.Reason = record.Reason
	assessment.State = "invalid"
	assessment.SatisfactionState = HotUpdateCanarySatisfactionStateInvalid
	if candidate.err != nil {
		assessment.Error = candidate.err.Error()
	}
	return assessment
}

func selectHotUpdateCanarySatisfactionEvidence(records []HotUpdateCanaryEvidenceRecord) HotUpdateCanaryEvidenceRecord {
	selected := records[0]
	for _, record := range records[1:] {
		if record.ObservedAt.After(selected.ObservedAt) ||
			(record.ObservedAt.Equal(selected.ObservedAt) && record.CanaryEvidenceID > selected.CanaryEvidenceID) {
			selected = record
		}
	}
	return selected
}

func loadHotUpdateCanarySatisfactionEvidenceCandidates(root string, requirement HotUpdateCanaryRequirementRecord) ([]HotUpdateCanaryEvidenceRecord, []hotUpdateCanarySatisfactionEvidenceCandidate, error) {
	entries, err := os.ReadDir(StoreHotUpdateCanaryEvidenceDir(root))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil, nil
		}
		return nil, nil, err
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !isStoreJSONDataFile(entry.Name()) {
			continue
		}
		names = append(names, entry.Name())
	}
	sort.Strings(names)

	valid := make([]HotUpdateCanaryEvidenceRecord, 0, len(names))
	invalid := make([]hotUpdateCanarySatisfactionEvidenceCandidate, 0)
	for _, name := range names {
		path := filepath.Join(StoreHotUpdateCanaryEvidenceDir(root), name)
		candidate, ok := loadHotUpdateCanarySatisfactionEvidenceCandidate(root, requirement, path)
		if !ok {
			continue
		}
		if candidate.err != nil {
			invalid = append(invalid, candidate)
			continue
		}
		valid = append(valid, candidate.record)
	}
	return valid, invalid, nil
}

func loadHotUpdateCanarySatisfactionEvidenceCandidate(root string, requirement HotUpdateCanaryRequirementRecord, path string) (hotUpdateCanarySatisfactionEvidenceCandidate, bool) {
	evidenceID := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	candidate := hotUpdateCanarySatisfactionEvidenceCandidate{evidenceID: evidenceID}
	filenameMatchesRequirement := strings.HasPrefix(evidenceID, "hot-update-canary-evidence-"+requirement.CanaryRequirementID+"-")

	var record HotUpdateCanaryEvidenceRecord
	if err := LoadStoreJSON(path, &record); err != nil {
		if !filenameMatchesRequirement {
			return candidate, false
		}
		candidate.err = err
		return candidate, true
	}
	record = NormalizeHotUpdateCanaryEvidenceRecord(record)
	candidate.record = record
	if record.CanaryRequirementID != requirement.CanaryRequirementID {
		if record.CanaryRequirementID == "" && filenameMatchesRequirement {
			candidate.err = fmt.Errorf("mission store hot-update canary evidence canary_requirement_id is required")
			return candidate, true
		}
		return candidate, false
	}
	if err := ValidateHotUpdateCanaryEvidenceRecord(record); err != nil {
		candidate.err = err
		return candidate, true
	}
	if err := validateHotUpdateCanaryEvidenceLinkage(root, record); err != nil {
		candidate.err = err
		return candidate, true
	}
	return candidate, true
}
