package missioncontrol

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

type AutonomyEligibilityDecision string

const (
	AutonomyEligibilityDecisionEligible   AutonomyEligibilityDecision = "eligible"
	AutonomyEligibilityDecisionIneligible AutonomyEligibilityDecision = "ineligible"
	AutonomyEligibilityDecisionUnknown    AutonomyEligibilityDecision = "unknown"
)

type AutonomyEligibilityReason string

const (
	AutonomyEligibilityReasonOwnerIdentityRequired              AutonomyEligibilityReason = "owner_identity_required"
	AutonomyEligibilityReasonOwnerLegalPersonhoodRequired       AutonomyEligibilityReason = "owner_legal_personhood_required"
	AutonomyEligibilityReasonOwnerPaymentMethodRequired         AutonomyEligibilityReason = "owner_payment_method_required"
	AutonomyEligibilityReasonManualHumanCompletionRequired      AutonomyEligibilityReason = "manual_human_completion_required"
	AutonomyEligibilityReasonHumanGatedKYCOrCustodialOnboarding AutonomyEligibilityReason = "human_gated_kyc_or_custodial_onboarding"
	AutonomyEligibilityReasonHiddenOwnerInfrastructureRequired  AutonomyEligibilityReason = "hidden_owner_infrastructure_required"
	AutonomyEligibilityReasonNotAutonomyCompatible              AutonomyEligibilityReason = "not_autonomy_compatible"
)

type AutonomyEligibilityTargetRef struct {
	Kind       EligibilityTargetKind `json:"kind"`
	RegistryID string                `json:"registry_id"`
}

type AutonomyEligibilityResult struct {
	Target        AutonomyEligibilityTargetRef `json:"target"`
	Decision      AutonomyEligibilityDecision  `json:"decision"`
	RegistryLabel EligibilityLabel             `json:"registry_label,omitempty"`
	CheckID       string                       `json:"check_id,omitempty"`
	CheckedAt     time.Time                    `json:"checked_at,omitempty"`
	Reasons       []AutonomyEligibilityReason  `json:"reasons,omitempty"`
}

var ErrAutonomyEligibleTargetRequired = errors.New("autonomy-eligible target required")

type AutonomyEligibilityError struct {
	Result  AutonomyEligibilityResult
	Message string
}

func (e AutonomyEligibilityError) Error() string {
	if strings.TrimSpace(e.Message) != "" {
		return e.Message
	}
	if e.Result.Decision == AutonomyEligibilityDecisionUnknown {
		return fmt.Sprintf("autonomy eligibility target %q has not been evaluated", e.Result.Target.RegistryID)
	}
	return fmt.Sprintf("autonomy eligibility target %q is not autonomy-compatible", e.Result.Target.RegistryID)
}

func (e AutonomyEligibilityError) Unwrap() error {
	return ErrAutonomyEligibleTargetRequired
}

func EvaluateAutonomyEligibility(root string, target AutonomyEligibilityTargetRef) (AutonomyEligibilityResult, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return AutonomyEligibilityResult{}, err
	}
	if err := validateAutonomyEligibilityTargetRef(target); err != nil {
		return AutonomyEligibilityResult{}, err
	}

	result := AutonomyEligibilityResult{
		Target:   target,
		Decision: AutonomyEligibilityDecisionUnknown,
	}

	record, err := LoadPlatformRecord(root, target.RegistryID)
	if err != nil {
		if errors.Is(err, ErrPlatformRecordNotFound) {
			return result, nil
		}
		return result, err
	}
	if record.TargetClass != target.Kind {
		return result, fmt.Errorf("autonomy eligibility target %q kind %q does not match registry target_class %q", target.RegistryID, target.Kind, record.TargetClass)
	}

	check, err := LoadEligibilityCheckRecord(root, record.LastCheckID)
	if err != nil {
		if errors.Is(err, ErrEligibilityCheckRecordNotFound) {
			return result, fmt.Errorf("autonomy eligibility target %q references missing eligibility check %q", target.RegistryID, record.LastCheckID)
		}
		return result, err
	}
	if check.TargetKind != target.Kind {
		return result, fmt.Errorf("autonomy eligibility target %q kind %q does not match eligibility check target_kind %q", target.RegistryID, target.Kind, check.TargetKind)
	}
	if check.TargetName != record.PlatformName {
		return result, fmt.Errorf("autonomy eligibility target %q registry name %q does not match eligibility check target_name %q", target.RegistryID, record.PlatformName, check.TargetName)
	}
	if check.Label != record.EligibilityLabel {
		return result, fmt.Errorf("autonomy eligibility target %q registry label %q does not match eligibility check label %q", target.RegistryID, record.EligibilityLabel, check.Label)
	}

	reasons, err := autonomyEligibilityReasonsForCheck(check)
	if err != nil {
		return result, err
	}
	result.RegistryLabel = check.Label
	result.CheckID = check.CheckID
	result.CheckedAt = check.CheckedAt.UTC()
	result.Reasons = reasons

	switch check.Label {
	case EligibilityLabelAutonomyCompatible:
		if len(reasons) != 0 {
			return result, fmt.Errorf("autonomy eligibility target %q is labeled autonomy_compatible but fails the autonomy predicate (%s)", target.RegistryID, joinAutonomyEligibilityReasons(reasons))
		}
		result.Decision = AutonomyEligibilityDecisionEligible
	case EligibilityLabelHumanGated, EligibilityLabelIneligible:
		if len(reasons) == 0 {
			return result, fmt.Errorf("autonomy eligibility target %q is labeled %q but has no autonomy-predicate failure reason", target.RegistryID, check.Label)
		}
		result.Decision = AutonomyEligibilityDecisionIneligible
	default:
		return result, fmt.Errorf("autonomy eligibility target %q has unsupported registry label %q", target.RegistryID, check.Label)
	}

	return result, nil
}

func RequireAutonomyEligibleTarget(root string, target AutonomyEligibilityTargetRef) (AutonomyEligibilityResult, error) {
	result, err := EvaluateAutonomyEligibility(root, target)
	if err != nil {
		return result, err
	}
	if result.Decision == AutonomyEligibilityDecisionEligible {
		return result, nil
	}
	if result.Decision == AutonomyEligibilityDecisionUnknown {
		return result, AutonomyEligibilityError{
			Result:  result,
			Message: fmt.Sprintf("autonomy eligibility target %q has no autonomy-compatible registry record", target.RegistryID),
		}
	}
	return result, AutonomyEligibilityError{
		Result:  result,
		Message: fmt.Sprintf("autonomy eligibility target %q is not autonomy-compatible (%s)", target.RegistryID, joinAutonomyEligibilityReasons(result.Reasons)),
	}
}

func validateAutonomyEligibilityTargetRef(target AutonomyEligibilityTargetRef) error {
	if !isValidEligibilityTargetKind(target.Kind) {
		return fmt.Errorf("autonomy eligibility target kind %q is invalid", target.Kind)
	}
	if strings.TrimSpace(target.RegistryID) == "" {
		return fmt.Errorf("autonomy eligibility target registry_id is required")
	}
	return nil
}

func autonomyEligibilityReasonsForCheck(check EligibilityCheckRecord) ([]AutonomyEligibilityReason, error) {
	explicit, err := parseExplicitAutonomyEligibilityReasons(check)
	if err != nil {
		return nil, err
	}
	if len(explicit) > 0 {
		return explicit, nil
	}

	var inferred []AutonomyEligibilityReason
	if check.RequiresHumanOnlyStep {
		if check.Label == EligibilityLabelHumanGated {
			inferred = append(inferred, AutonomyEligibilityReasonHumanGatedKYCOrCustodialOnboarding)
		} else {
			inferred = append(inferred, AutonomyEligibilityReasonManualHumanCompletionRequired)
		}
	}
	if !check.CanOnboardWithoutOwner {
		inferred = append(inferred, AutonomyEligibilityReasonManualHumanCompletionRequired)
	}
	if check.RequiresOwnerOnlySecretOrID {
		inferred = append(inferred, AutonomyEligibilityReasonHiddenOwnerInfrastructureRequired)
	}
	if !check.CanCreateWithoutOwner || !check.CanControlAsAgent || !check.CanRecoverAsAgent || !check.RulesAsObservedOK {
		inferred = append(inferred, AutonomyEligibilityReasonNotAutonomyCompatible)
	}

	return uniqueAutonomyEligibilityReasons(inferred), nil
}

func parseExplicitAutonomyEligibilityReasons(check EligibilityCheckRecord) ([]AutonomyEligibilityReason, error) {
	reasons := make([]AutonomyEligibilityReason, 0, len(check.Reasons))
	for _, raw := range check.Reasons {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		reason := AutonomyEligibilityReason(raw)
		if isValidAutonomyEligibilityReason(reason) {
			reasons = append(reasons, reason)
			continue
		}
		if check.Label != EligibilityLabelAutonomyCompatible {
			return nil, fmt.Errorf("mission store eligibility check %q reason %q is not a canonical autonomy reason", check.CheckID, raw)
		}
	}
	return uniqueAutonomyEligibilityReasons(reasons), nil
}

func uniqueAutonomyEligibilityReasons(reasons []AutonomyEligibilityReason) []AutonomyEligibilityReason {
	if len(reasons) == 0 {
		return nil
	}
	seen := make(map[AutonomyEligibilityReason]struct{}, len(reasons))
	out := make([]AutonomyEligibilityReason, 0, len(reasons))
	for _, reason := range reasons {
		if !isValidAutonomyEligibilityReason(reason) {
			continue
		}
		if _, ok := seen[reason]; ok {
			continue
		}
		seen[reason] = struct{}{}
		out = append(out, reason)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i] < out[j]
	})
	if len(out) == 0 {
		return nil
	}
	return out
}

func joinAutonomyEligibilityReasons(reasons []AutonomyEligibilityReason) string {
	if len(reasons) == 0 {
		return ""
	}
	parts := make([]string, len(reasons))
	for i, reason := range reasons {
		parts[i] = string(reason)
	}
	return strings.Join(parts, ", ")
}

func isValidAutonomyEligibilityReason(reason AutonomyEligibilityReason) bool {
	switch reason {
	case AutonomyEligibilityReasonOwnerIdentityRequired,
		AutonomyEligibilityReasonOwnerLegalPersonhoodRequired,
		AutonomyEligibilityReasonOwnerPaymentMethodRequired,
		AutonomyEligibilityReasonManualHumanCompletionRequired,
		AutonomyEligibilityReasonHumanGatedKYCOrCustodialOnboarding,
		AutonomyEligibilityReasonHiddenOwnerInfrastructureRequired,
		AutonomyEligibilityReasonNotAutonomyCompatible:
		return true
	default:
		return false
	}
}
