package missioncontrol

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
)

type CandidateResultRef struct {
	ResultID string `json:"result_id"`
}

const (
	CandidatePromotionEligibilityStateEligible                       = "eligible"
	CandidatePromotionEligibilityStateCanaryRequired                 = "canary_required"
	CandidatePromotionEligibilityStateOwnerApprovalRequired          = "owner_approval_required"
	CandidatePromotionEligibilityStateCanaryAndOwnerApprovalRequired = "canary_and_owner_approval_required"
	CandidatePromotionEligibilityStateRejected                       = "rejected"
	CandidatePromotionEligibilityStateUnsupportedPolicy              = "unsupported_policy"
	CandidatePromotionEligibilityStateInvalid                        = "invalid"
)

var candidateResultRequiredScoreKeys = []string{
	"baseline_score",
	"train_score",
	"holdout_score",
	"complexity_score",
	"compatibility_score",
	"resource_score",
}

type CandidatePromotionEligibilityStatus struct {
	State                 string   `json:"state"`
	ResultID              string   `json:"result_id,omitempty"`
	RunID                 string   `json:"run_id,omitempty"`
	CandidateID           string   `json:"candidate_id,omitempty"`
	EvalSuiteID           string   `json:"eval_suite_id,omitempty"`
	PromotionPolicyID     string   `json:"promotion_policy_id,omitempty"`
	BaselinePackID        string   `json:"baseline_pack_id,omitempty"`
	CandidatePackID       string   `json:"candidate_pack_id,omitempty"`
	Decision              string   `json:"decision,omitempty"`
	BlockingReasons       []string `json:"blocking_reasons,omitempty"`
	CanaryRequired        bool     `json:"canary_required,omitempty"`
	OwnerApprovalRequired bool     `json:"owner_approval_required,omitempty"`
	Error                 string   `json:"error,omitempty"`
}

type CandidateResultRecord struct {
	RecordVersion      int                    `json:"record_version"`
	ResultID           string                 `json:"result_id"`
	RunID              string                 `json:"run_id"`
	CandidateID        string                 `json:"candidate_id"`
	EvalSuiteID        string                 `json:"eval_suite_id"`
	PromotionPolicyID  string                 `json:"promotion_policy_id,omitempty"`
	BaselinePackID     string                 `json:"baseline_pack_id"`
	CandidatePackID    string                 `json:"candidate_pack_id"`
	HotUpdateID        string                 `json:"hot_update_id,omitempty"`
	BaselineScore      float64                `json:"baseline_score"`
	TrainScore         float64                `json:"train_score"`
	HoldoutScore       float64                `json:"holdout_score"`
	ComplexityScore    float64                `json:"complexity_score"`
	CompatibilityScore float64                `json:"compatibility_score"`
	ResourceScore      float64                `json:"resource_score"`
	RegressionFlags    []string               `json:"regression_flags,omitempty"`
	Decision           ImprovementRunDecision `json:"decision"`
	Notes              string                 `json:"notes"`
	CreatedAt          time.Time              `json:"created_at"`
	CreatedBy          string                 `json:"created_by"`
}

var ErrCandidateResultRecordNotFound = errors.New("mission store candidate result record not found")

func StoreCandidateResultsDir(root string) string {
	return filepath.Join(root, "runtime_packs", "candidate_results")
}

func StoreCandidateResultPath(root, resultID string) string {
	return filepath.Join(StoreCandidateResultsDir(root), strings.TrimSpace(resultID)+".json")
}

func NormalizeCandidateResultRef(ref CandidateResultRef) CandidateResultRef {
	ref.ResultID = strings.TrimSpace(ref.ResultID)
	return ref
}

func NormalizeCandidateResultRecord(record CandidateResultRecord) CandidateResultRecord {
	record.ResultID = strings.TrimSpace(record.ResultID)
	record.RunID = strings.TrimSpace(record.RunID)
	record.CandidateID = strings.TrimSpace(record.CandidateID)
	record.EvalSuiteID = strings.TrimSpace(record.EvalSuiteID)
	record.PromotionPolicyID = strings.TrimSpace(record.PromotionPolicyID)
	record.BaselinePackID = strings.TrimSpace(record.BaselinePackID)
	record.CandidatePackID = strings.TrimSpace(record.CandidatePackID)
	record.HotUpdateID = strings.TrimSpace(record.HotUpdateID)
	record.RegressionFlags = normalizeCandidateResultStrings(record.RegressionFlags)
	record.Notes = strings.TrimSpace(record.Notes)
	record.CreatedAt = record.CreatedAt.UTC()
	record.CreatedBy = strings.TrimSpace(record.CreatedBy)
	return record
}

func CandidateResultImprovementRunRef(record CandidateResultRecord) ImprovementRunRef {
	return ImprovementRunRef{RunID: strings.TrimSpace(record.RunID)}
}

func CandidateResultImprovementCandidateRef(record CandidateResultRecord) ImprovementCandidateRef {
	return ImprovementCandidateRef{CandidateID: strings.TrimSpace(record.CandidateID)}
}

func CandidateResultEvalSuiteRef(record CandidateResultRecord) EvalSuiteRef {
	return EvalSuiteRef{EvalSuiteID: strings.TrimSpace(record.EvalSuiteID)}
}

func CandidateResultPromotionPolicyRef(record CandidateResultRecord) (PromotionPolicyRef, bool) {
	promotionPolicyID := strings.TrimSpace(record.PromotionPolicyID)
	if promotionPolicyID == "" {
		return PromotionPolicyRef{}, false
	}
	return PromotionPolicyRef{PromotionPolicyID: promotionPolicyID}, true
}

func CandidateResultBaselinePackRef(record CandidateResultRecord) RuntimePackRef {
	return RuntimePackRef{PackID: strings.TrimSpace(record.BaselinePackID)}
}

func CandidateResultCandidatePackRef(record CandidateResultRecord) RuntimePackRef {
	return RuntimePackRef{PackID: strings.TrimSpace(record.CandidatePackID)}
}

func CandidateResultHotUpdateGateRef(record CandidateResultRecord) (HotUpdateGateRef, bool) {
	hotUpdateID := strings.TrimSpace(record.HotUpdateID)
	if hotUpdateID == "" {
		return HotUpdateGateRef{}, false
	}
	return HotUpdateGateRef{HotUpdateID: hotUpdateID}, true
}

func ValidateCandidateResultRef(ref CandidateResultRef) error {
	return validateCandidateResultIdentifierField("candidate result ref", "result_id", ref.ResultID)
}

func ValidateCandidateResultRecord(record CandidateResultRecord) error {
	if record.RecordVersion <= 0 {
		return fmt.Errorf("mission store candidate result record_version must be positive")
	}
	if err := ValidateCandidateResultRef(CandidateResultRef{ResultID: record.ResultID}); err != nil {
		return err
	}
	if err := ValidateImprovementRunRef(CandidateResultImprovementRunRef(record)); err != nil {
		return fmt.Errorf("mission store candidate result run_id %q: %w", record.RunID, err)
	}
	if err := ValidateImprovementCandidateRef(CandidateResultImprovementCandidateRef(record)); err != nil {
		return fmt.Errorf("mission store candidate result candidate_id %q: %w", record.CandidateID, err)
	}
	if err := ValidateEvalSuiteRef(CandidateResultEvalSuiteRef(record)); err != nil {
		return fmt.Errorf("mission store candidate result eval_suite_id %q: %w", record.EvalSuiteID, err)
	}
	if promotionPolicyRef, ok := CandidateResultPromotionPolicyRef(record); ok {
		if err := ValidatePromotionPolicyRef(promotionPolicyRef); err != nil {
			return fmt.Errorf("mission store candidate result promotion_policy_id %q: %w", record.PromotionPolicyID, err)
		}
	}
	if err := ValidateRuntimePackRef(CandidateResultBaselinePackRef(record)); err != nil {
		return fmt.Errorf("mission store candidate result baseline_pack_id %q: %w", record.BaselinePackID, err)
	}
	if err := ValidateRuntimePackRef(CandidateResultCandidatePackRef(record)); err != nil {
		return fmt.Errorf("mission store candidate result candidate_pack_id %q: %w", record.CandidatePackID, err)
	}
	if gateRef, ok := CandidateResultHotUpdateGateRef(record); ok {
		if err := ValidateHotUpdateGateRef(gateRef); err != nil {
			return fmt.Errorf("mission store candidate result hot_update_id %q: %w", record.HotUpdateID, err)
		}
	}
	for fieldName, score := range map[string]float64{
		"baseline_score":      record.BaselineScore,
		"train_score":         record.TrainScore,
		"holdout_score":       record.HoldoutScore,
		"complexity_score":    record.ComplexityScore,
		"compatibility_score": record.CompatibilityScore,
		"resource_score":      record.ResourceScore,
	} {
		if math.IsNaN(score) || math.IsInf(score, 0) {
			return fmt.Errorf("mission store candidate result %s must be finite", fieldName)
		}
	}
	if record.Decision == "" {
		return fmt.Errorf("mission store candidate result decision is required")
	}
	if !isValidImprovementRunDecision(record.Decision) {
		return fmt.Errorf("mission store candidate result decision %q is invalid", record.Decision)
	}
	if record.CreatedAt.IsZero() {
		return fmt.Errorf("mission store candidate result created_at is required")
	}
	if record.CreatedBy == "" {
		return fmt.Errorf("mission store candidate result created_by is required")
	}
	return nil
}

func StoreCandidateResultRecord(root string, record CandidateResultRecord) error {
	if err := ValidateStoreRoot(root); err != nil {
		return err
	}
	record = NormalizeCandidateResultRecord(record)
	record.RecordVersion = normalizeRecordVersion(record.RecordVersion)
	if err := ValidateCandidateResultRecord(record); err != nil {
		return err
	}
	if err := validateCandidateResultLinkage(root, record); err != nil {
		return err
	}

	path := StoreCandidateResultPath(root, record.ResultID)
	if existing, err := loadCandidateResultRecordFile(root, path); err == nil {
		if reflect.DeepEqual(existing, record) {
			return nil
		}
		return fmt.Errorf("mission store candidate result %q already exists", record.ResultID)
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	return WriteStoreJSONAtomic(path, record)
}

func LoadCandidateResultRecord(root, resultID string) (CandidateResultRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return CandidateResultRecord{}, err
	}
	ref := NormalizeCandidateResultRef(CandidateResultRef{ResultID: resultID})
	if err := ValidateCandidateResultRef(ref); err != nil {
		return CandidateResultRecord{}, err
	}
	record, err := loadCandidateResultRecordFile(root, StoreCandidateResultPath(root, ref.ResultID))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return CandidateResultRecord{}, ErrCandidateResultRecordNotFound
		}
		return CandidateResultRecord{}, err
	}
	return record, nil
}

func EvaluateCandidateResultPromotionEligibility(root, resultID string) (CandidatePromotionEligibilityStatus, error) {
	status := CandidatePromotionEligibilityStatus{
		State:    CandidatePromotionEligibilityStateInvalid,
		ResultID: strings.TrimSpace(resultID),
	}
	if err := ValidateStoreRoot(root); err != nil {
		status.Error = err.Error()
		return status, err
	}

	record, err := LoadCandidateResultRecord(root, resultID)
	if err != nil {
		status.Error = err.Error()
		return status, nil
	}
	status = candidatePromotionEligibilityStatusFromRecord(record)

	if record.PromotionPolicyID == "" {
		status.State = CandidatePromotionEligibilityStateInvalid
		status.BlockingReasons = []string{"promotion_policy_id is required"}
		status.Error = "mission store candidate result promotion_policy_id is required for promotion eligibility"
		return status, nil
	}
	if err := validateCandidateResultScoreKeyPresence(root, record.ResultID); err != nil {
		status.State = CandidatePromotionEligibilityStateInvalid
		status.BlockingReasons = []string{err.Error()}
		status.Error = err.Error()
		return status, nil
	}
	if err := validateCandidateResultFiniteScores(record); err != nil {
		status.State = CandidatePromotionEligibilityStateInvalid
		status.BlockingReasons = []string{err.Error()}
		status.Error = err.Error()
		return status, nil
	}

	policy, err := LoadPromotionPolicyRecord(root, record.PromotionPolicyID)
	if err != nil {
		status.State = CandidatePromotionEligibilityStateInvalid
		status.BlockingReasons = []string{err.Error()}
		status.Error = err.Error()
		return status, nil
	}

	epsilon, err := parseCandidateEligibilityEpsilonRule(policy.EpsilonRule)
	if err != nil {
		status.State = CandidatePromotionEligibilityStateUnsupportedPolicy
		status.BlockingReasons = []string{err.Error()}
		return status, nil
	}
	regressionMode, err := parseCandidateEligibilityRegressionRule(policy.RegressionRule)
	if err != nil {
		status.State = CandidatePromotionEligibilityStateUnsupportedPolicy
		status.BlockingReasons = []string{err.Error()}
		return status, nil
	}
	compatibilityThreshold, err := parseCandidateEligibilityThresholdRule(policy.CompatibilityRule, "compatibility_score")
	if err != nil {
		status.State = CandidatePromotionEligibilityStateUnsupportedPolicy
		status.BlockingReasons = []string{err.Error()}
		return status, nil
	}
	resourceThreshold, err := parseCandidateEligibilityThresholdRule(policy.ResourceRule, "resource_score")
	if err != nil {
		status.State = CandidatePromotionEligibilityStateUnsupportedPolicy
		status.BlockingReasons = []string{err.Error()}
		return status, nil
	}

	var blocking []string
	if record.Decision != ImprovementRunDecisionKeep {
		blocking = append(blocking, fmt.Sprintf("decision %q is not keep", record.Decision))
	}

	holdoutPass := record.HoldoutScore > record.BaselineScore+epsilon
	trainPass := record.TrainScore > record.BaselineScore+epsilon
	if trainPass && !holdoutPass {
		blocking = append(blocking, "train-only improvement is not promotable")
	}
	if policy.RequiresHoldoutPass && !holdoutPass {
		blocking = append(blocking, "holdout score does not satisfy epsilon rule")
	}
	if !holdoutPass {
		blocking = append(blocking, "holdout improvement is required for promotion eligibility")
	}
	if !candidateResultRegressionFlagsPass(record.RegressionFlags, regressionMode) {
		blocking = append(blocking, "regression flags are not allowed by policy")
	}
	if record.CompatibilityScore < compatibilityThreshold {
		blocking = append(blocking, "compatibility score is below policy threshold")
	}
	if record.ResourceScore < resourceThreshold {
		blocking = append(blocking, "resource score is below policy threshold")
	}
	if len(blocking) > 0 {
		status.State = CandidatePromotionEligibilityStateRejected
		status.BlockingReasons = blocking
		return status, nil
	}

	status.CanaryRequired = policy.RequiresCanary
	status.OwnerApprovalRequired = policy.RequiresOwnerApproval
	switch {
	case status.CanaryRequired && status.OwnerApprovalRequired:
		status.State = CandidatePromotionEligibilityStateCanaryAndOwnerApprovalRequired
	case status.CanaryRequired:
		status.State = CandidatePromotionEligibilityStateCanaryRequired
	case status.OwnerApprovalRequired:
		status.State = CandidatePromotionEligibilityStateOwnerApprovalRequired
	default:
		status.State = CandidatePromotionEligibilityStateEligible
	}
	return status, nil
}

func ListCandidateResultRecords(root string) ([]CandidateResultRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(StoreCandidateResultsDir(root))
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

	records := make([]CandidateResultRecord, 0, len(names))
	for _, name := range names {
		record, err := loadCandidateResultRecordFile(root, filepath.Join(StoreCandidateResultsDir(root), name))
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, nil
}

func candidatePromotionEligibilityStatusFromRecord(record CandidateResultRecord) CandidatePromotionEligibilityStatus {
	return CandidatePromotionEligibilityStatus{
		State:             CandidatePromotionEligibilityStateInvalid,
		ResultID:          record.ResultID,
		RunID:             record.RunID,
		CandidateID:       record.CandidateID,
		EvalSuiteID:       record.EvalSuiteID,
		PromotionPolicyID: record.PromotionPolicyID,
		BaselinePackID:    record.BaselinePackID,
		CandidatePackID:   record.CandidatePackID,
		Decision:          string(record.Decision),
	}
}

func validateCandidateResultScoreKeyPresence(root, resultID string) error {
	bytes, err := os.ReadFile(StoreCandidateResultPath(root, resultID))
	if err != nil {
		return err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(bytes, &fields); err != nil {
		return err
	}
	for _, key := range candidateResultRequiredScoreKeys {
		if _, ok := fields[key]; !ok {
			return fmt.Errorf("mission store candidate result %s is required for promotion eligibility", key)
		}
	}
	return nil
}

func validateCandidateResultFiniteScores(record CandidateResultRecord) error {
	for fieldName, score := range map[string]float64{
		"baseline_score":      record.BaselineScore,
		"train_score":         record.TrainScore,
		"holdout_score":       record.HoldoutScore,
		"complexity_score":    record.ComplexityScore,
		"compatibility_score": record.CompatibilityScore,
		"resource_score":      record.ResourceScore,
	} {
		if math.IsNaN(score) || math.IsInf(score, 0) {
			return fmt.Errorf("mission store candidate result %s must be finite", fieldName)
		}
	}
	return nil
}

func parseCandidateEligibilityEpsilonRule(rule string) (float64, error) {
	field, threshold, err := parseCandidateEligibilityComparisonRule(rule)
	if err != nil {
		return 0, fmt.Errorf("unsupported epsilon_rule %q", rule)
	}
	if field != "epsilon" || threshold < 0 {
		return 0, fmt.Errorf("unsupported epsilon_rule %q", rule)
	}
	return threshold, nil
}

func parseCandidateEligibilityRegressionRule(rule string) (string, error) {
	rule = strings.TrimSpace(rule)
	switch rule {
	case "no_regression_flags", "holdout_regression <= 0":
		return rule, nil
	default:
		return "", fmt.Errorf("unsupported regression_rule %q", rule)
	}
}

func parseCandidateEligibilityThresholdRule(rule, wantField string) (float64, error) {
	field, threshold, err := parseCandidateEligibilityComparisonRule(rule)
	if err != nil {
		return 0, fmt.Errorf("unsupported %s rule %q", wantField, rule)
	}
	if field != wantField || threshold < 0 || threshold > 1 {
		return 0, fmt.Errorf("unsupported %s rule %q", wantField, rule)
	}
	return threshold, nil
}

func parseCandidateEligibilityComparisonRule(rule string) (string, float64, error) {
	fields := strings.Fields(strings.TrimSpace(rule))
	if len(fields) != 3 || fields[1] != "<=" && fields[1] != ">=" {
		return "", 0, fmt.Errorf("unsupported comparison rule")
	}
	if fields[1] != "<=" && fields[0] == "epsilon" {
		return "", 0, fmt.Errorf("unsupported comparison rule")
	}
	if fields[1] != ">=" && fields[0] != "epsilon" {
		return "", 0, fmt.Errorf("unsupported comparison rule")
	}
	threshold, err := strconv.ParseFloat(fields[2], 64)
	if err != nil || math.IsNaN(threshold) || math.IsInf(threshold, 0) {
		return "", 0, fmt.Errorf("unsupported comparison rule")
	}
	return fields[0], threshold, nil
}

func candidateResultRegressionFlagsPass(flags []string, mode string) bool {
	switch mode {
	case "no_regression_flags", "holdout_regression <= 0":
		for _, flag := range flags {
			if strings.TrimSpace(strings.ToLower(flag)) == "none" {
				continue
			}
			return false
		}
		return true
	default:
		return false
	}
}

func loadCandidateResultRecordFile(root, path string) (CandidateResultRecord, error) {
	var record CandidateResultRecord
	if err := LoadStoreJSON(path, &record); err != nil {
		return CandidateResultRecord{}, err
	}
	record = NormalizeCandidateResultRecord(record)
	if err := ValidateCandidateResultRecord(record); err != nil {
		return CandidateResultRecord{}, err
	}
	if err := validateCandidateResultLinkage(root, record); err != nil {
		return CandidateResultRecord{}, err
	}
	return record, nil
}

func validateCandidateResultLinkage(root string, record CandidateResultRecord) error {
	run, err := LoadImprovementRunRecord(root, record.RunID)
	if err != nil {
		return fmt.Errorf("mission store candidate result run_id %q: %w", record.RunID, err)
	}
	if run.CandidateID != record.CandidateID {
		return fmt.Errorf(
			"mission store candidate result candidate_id %q does not match run candidate_id %q",
			record.CandidateID,
			run.CandidateID,
		)
	}
	if run.EvalSuiteID != record.EvalSuiteID {
		return fmt.Errorf(
			"mission store candidate result eval_suite_id %q does not match run eval_suite_id %q",
			record.EvalSuiteID,
			run.EvalSuiteID,
		)
	}
	if run.BaselinePackID != record.BaselinePackID {
		return fmt.Errorf(
			"mission store candidate result baseline_pack_id %q does not match run baseline_pack_id %q",
			record.BaselinePackID,
			run.BaselinePackID,
		)
	}
	if run.CandidatePackID != record.CandidatePackID {
		return fmt.Errorf(
			"mission store candidate result candidate_pack_id %q does not match run candidate_pack_id %q",
			record.CandidatePackID,
			run.CandidatePackID,
		)
	}
	if gateRef, ok := CandidateResultHotUpdateGateRef(record); ok {
		if run.HotUpdateID == "" {
			return fmt.Errorf("mission store candidate result hot_update_id %q requires run hot_update_id", gateRef.HotUpdateID)
		}
		if run.HotUpdateID != gateRef.HotUpdateID {
			return fmt.Errorf(
				"mission store candidate result hot_update_id %q does not match run hot_update_id %q",
				gateRef.HotUpdateID,
				run.HotUpdateID,
			)
		}
	}

	candidate, err := LoadImprovementCandidateRecord(root, record.CandidateID)
	if err != nil {
		return fmt.Errorf("mission store candidate result candidate_id %q: %w", record.CandidateID, err)
	}
	if candidate.BaselinePackID != record.BaselinePackID {
		return fmt.Errorf(
			"mission store candidate result baseline_pack_id %q does not match candidate baseline_pack_id %q",
			record.BaselinePackID,
			candidate.BaselinePackID,
		)
	}
	if candidate.CandidatePackID != record.CandidatePackID {
		return fmt.Errorf(
			"mission store candidate result candidate_pack_id %q does not match candidate candidate_pack_id %q",
			record.CandidatePackID,
			candidate.CandidatePackID,
		)
	}
	if gateRef, ok := CandidateResultHotUpdateGateRef(record); ok && candidate.HotUpdateID != "" && candidate.HotUpdateID != gateRef.HotUpdateID {
		return fmt.Errorf(
			"mission store candidate result hot_update_id %q does not match candidate hot_update_id %q",
			gateRef.HotUpdateID,
			candidate.HotUpdateID,
		)
	}

	evalSuite, err := LoadEvalSuiteRecord(root, record.EvalSuiteID)
	if err != nil {
		return fmt.Errorf("mission store candidate result eval_suite_id %q: %w", record.EvalSuiteID, err)
	}
	if evalSuite.CandidateID != "" && evalSuite.CandidateID != record.CandidateID {
		return fmt.Errorf(
			"mission store candidate result eval_suite_id %q candidate_id %q does not match result candidate_id %q",
			record.EvalSuiteID,
			evalSuite.CandidateID,
			record.CandidateID,
		)
	}
	if evalSuite.BaselinePackID != "" && evalSuite.BaselinePackID != record.BaselinePackID {
		return fmt.Errorf(
			"mission store candidate result eval_suite_id %q baseline_pack_id %q does not match result baseline_pack_id %q",
			record.EvalSuiteID,
			evalSuite.BaselinePackID,
			record.BaselinePackID,
		)
	}
	if evalSuite.CandidatePackID != "" && evalSuite.CandidatePackID != record.CandidatePackID {
		return fmt.Errorf(
			"mission store candidate result eval_suite_id %q candidate_pack_id %q does not match result candidate_pack_id %q",
			record.EvalSuiteID,
			evalSuite.CandidatePackID,
			record.CandidatePackID,
		)
	}

	if promotionPolicyRef, ok := CandidateResultPromotionPolicyRef(record); ok {
		if _, err := LoadPromotionPolicyRecord(root, promotionPolicyRef.PromotionPolicyID); err != nil {
			return fmt.Errorf("mission store candidate result promotion_policy_id %q: %w", promotionPolicyRef.PromotionPolicyID, err)
		}
	}

	if _, err := LoadRuntimePackRecord(root, record.BaselinePackID); err != nil {
		return fmt.Errorf("mission store candidate result baseline_pack_id %q: %w", record.BaselinePackID, err)
	}
	if _, err := LoadRuntimePackRecord(root, record.CandidatePackID); err != nil {
		return fmt.Errorf("mission store candidate result candidate_pack_id %q: %w", record.CandidatePackID, err)
	}

	if gateRef, ok := CandidateResultHotUpdateGateRef(record); ok {
		gate, err := LoadHotUpdateGateRecord(root, gateRef.HotUpdateID)
		if err != nil {
			return fmt.Errorf("mission store candidate result hot_update_id %q: %w", gateRef.HotUpdateID, err)
		}
		if gate.CandidatePackID != record.CandidatePackID {
			return fmt.Errorf(
				"mission store candidate result hot_update_id %q candidate_pack_id %q does not match result candidate_pack_id %q",
				gateRef.HotUpdateID,
				gate.CandidatePackID,
				record.CandidatePackID,
			)
		}
		if gate.PreviousActivePackID != record.BaselinePackID {
			return fmt.Errorf(
				"mission store candidate result hot_update_id %q previous_active_pack_id %q does not match result baseline_pack_id %q",
				gateRef.HotUpdateID,
				gate.PreviousActivePackID,
				record.BaselinePackID,
			)
		}
	}

	return nil
}

func validateCandidateResultIdentifierField(surface, fieldName, value string) error {
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

func normalizeCandidateResultStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		normalized = append(normalized, value)
	}
	if len(normalized) == 0 {
		return nil
	}
	return normalized
}
