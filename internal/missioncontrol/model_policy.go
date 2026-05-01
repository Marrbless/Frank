package missioncontrol

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/local/picobot/internal/config"
)

func (c *ModelPolicyRequiredCapabilities) UnmarshalJSON(data []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	for key := range raw {
		switch key {
		case "supportsTools", "local", "offline", "supportsResponsesAPI", "authorityTierAtLeast":
		default:
			return fmt.Errorf("model_policy.required_capabilities contains unknown field %q", key)
		}
	}

	type alias ModelPolicyRequiredCapabilities
	var out alias
	if err := json.Unmarshal(data, &out); err != nil {
		return err
	}
	*c = ModelPolicyRequiredCapabilities(out)
	return nil
}

func validateJobModelPolicies(job Job) []ValidationError {
	errors := make([]ValidationError, 0)
	errors = append(errors, validateModelPolicy("job", "", job.ModelPolicy)...)
	for _, step := range job.Plan.Steps {
		errors = append(errors, validateModelPolicy("step", step.ID, step.ModelPolicy)...)
		errors = append(errors, validateStepModelPolicyNarrowing(job.ModelPolicy, step)...)
	}
	return errors
}

func validateModelPolicy(scope, stepID string, policy *ModelPolicy) []ValidationError {
	if policy == nil {
		return nil
	}

	errors := make([]ValidationError, 0)
	allowedModels := make(map[string]int, len(policy.AllowedModels))
	for i, modelRef := range policy.AllowedModels {
		normalized, err := normalizeModelPolicyModelRef(modelRef)
		if err != nil {
			errors = append(errors, modelPolicyValidationError(stepID, fmt.Sprintf("%s model_policy.allowed_models[%d]: %v", scope, i, err)))
			continue
		}
		if firstIndex, ok := allowedModels[normalized]; ok {
			errors = append(errors, modelPolicyValidationError(stepID, fmt.Sprintf("%s model_policy.allowed_models[%d] duplicates allowed_models[%d]: %s", scope, i, firstIndex, normalized)))
			continue
		}
		allowedModels[normalized] = i
	}

	defaultModel := strings.TrimSpace(policy.DefaultModel)
	if defaultModel != "" {
		normalizedDefault, err := normalizeModelPolicyModelRef(defaultModel)
		if err != nil {
			errors = append(errors, modelPolicyValidationError(stepID, fmt.Sprintf("%s model_policy.default_model: %v", scope, err)))
		} else if len(allowedModels) > 0 {
			if _, ok := allowedModels[normalizedDefault]; !ok {
				errors = append(errors, modelPolicyValidationError(stepID, fmt.Sprintf("%s model_policy.default_model %q is not in allowed_models", scope, normalizedDefault)))
			}
		}
	}

	if tier := strings.TrimSpace(string(policy.RequiredCapabilities.AuthorityTierAtLeast)); tier != "" {
		if _, ok := authorityRank(AuthorityTier(tier)); !ok {
			errors = append(errors, modelPolicyValidationError(stepID, fmt.Sprintf("%s model_policy.required_capabilities.authorityTierAtLeast %q is invalid", scope, tier)))
		}
	}

	return errors
}

func validateStepModelPolicyNarrowing(jobPolicy *ModelPolicy, step Step) []ValidationError {
	if jobPolicy == nil || step.ModelPolicy == nil {
		return nil
	}

	stepID := step.ID
	errors := make([]ValidationError, 0)

	jobAllowed, jobAllowedOK := normalizedModelPolicyAllowedSet(jobPolicy.AllowedModels)
	stepAllowed, stepAllowedOK := normalizedModelPolicyAllowedSet(step.ModelPolicy.AllowedModels)
	if jobAllowedOK && stepAllowedOK && len(jobAllowed) > 0 {
		for modelRef := range stepAllowed {
			if _, ok := jobAllowed[modelRef]; !ok {
				errors = append(errors, modelPolicyValidationError(stepID, fmt.Sprintf("step model_policy.allowed_models includes %q outside job model_policy.allowed_models", modelRef)))
			}
		}
	}

	stepDefault, stepDefaultOK := normalizedOptionalModelPolicyModelRef(step.ModelPolicy.DefaultModel)
	if stepDefaultOK && stepDefault != "" && jobAllowedOK && len(jobAllowed) > 0 {
		if _, ok := jobAllowed[stepDefault]; !ok {
			errors = append(errors, modelPolicyValidationError(stepID, fmt.Sprintf("step model_policy.default_model %q is outside job model_policy.allowed_models", stepDefault)))
		}
	}

	if jobPolicy.AllowFallback != nil && !*jobPolicy.AllowFallback && step.ModelPolicy.AllowFallback != nil && *step.ModelPolicy.AllowFallback {
		errors = append(errors, modelPolicyValidationError(stepID, "step model_policy.allow_fallback cannot widen job model_policy.allow_fallback=false"))
	}
	if jobPolicy.AllowCloud != nil && !*jobPolicy.AllowCloud && step.ModelPolicy.AllowCloud != nil && *step.ModelPolicy.AllowCloud {
		errors = append(errors, modelPolicyValidationError(stepID, "step model_policy.allow_cloud cannot widen job model_policy.allow_cloud=false"))
	}

	errors = append(errors, validateStepModelRequiredCapabilityNarrowing(jobPolicy.RequiredCapabilities, step.ModelPolicy.RequiredCapabilities, stepID)...)
	return errors
}

func validateStepModelRequiredCapabilityNarrowing(jobRequired, stepRequired ModelPolicyRequiredCapabilities, stepID string) []ValidationError {
	errors := make([]ValidationError, 0)
	errors = append(errors, validateBoolModelCapabilityNarrowing("supportsTools", jobRequired.SupportsTools, stepRequired.SupportsTools, stepID)...)
	errors = append(errors, validateBoolModelCapabilityNarrowing("local", jobRequired.Local, stepRequired.Local, stepID)...)
	errors = append(errors, validateBoolModelCapabilityNarrowing("offline", jobRequired.Offline, stepRequired.Offline, stepID)...)
	errors = append(errors, validateBoolModelCapabilityNarrowing("supportsResponsesAPI", jobRequired.SupportsResponsesAPI, stepRequired.SupportsResponsesAPI, stepID)...)

	jobTier := AuthorityTier(strings.TrimSpace(string(jobRequired.AuthorityTierAtLeast)))
	stepTier := AuthorityTier(strings.TrimSpace(string(stepRequired.AuthorityTierAtLeast)))
	if jobTier != "" && stepTier != "" {
		jobRank, jobOK := authorityRank(jobTier)
		stepRank, stepOK := authorityRank(stepTier)
		if jobOK && stepOK && stepRank < jobRank {
			errors = append(errors, modelPolicyValidationError(stepID, fmt.Sprintf("step model_policy.required_capabilities.authorityTierAtLeast %q cannot be below job requirement %q", stepTier, jobTier)))
		}
	}
	return errors
}

func validateBoolModelCapabilityNarrowing(name string, jobValue, stepValue *bool, stepID string) []ValidationError {
	if jobValue == nil || stepValue == nil || *jobValue == *stepValue {
		return nil
	}
	return []ValidationError{
		modelPolicyValidationError(stepID, fmt.Sprintf("step model_policy.required_capabilities.%s=%t conflicts with job requirement %t", name, *stepValue, *jobValue)),
	}
}

func normalizedModelPolicyAllowedSet(values []string) (map[string]struct{}, bool) {
	out := make(map[string]struct{}, len(values))
	for _, value := range values {
		normalized, err := normalizeModelPolicyModelRef(value)
		if err != nil {
			return nil, false
		}
		out[normalized] = struct{}{}
	}
	return out, true
}

func normalizedOptionalModelPolicyModelRef(value string) (string, bool) {
	if strings.TrimSpace(value) == "" {
		return "", true
	}
	normalized, err := normalizeModelPolicyModelRef(value)
	if err != nil {
		return "", false
	}
	return normalized, true
}

func normalizeModelPolicyModelRef(value string) (string, error) {
	return config.NormalizeModelRef(value)
}

func modelPolicyValidationError(stepID, message string) ValidationError {
	return ValidationError{
		Code:    RejectionCodeInvalidModelPolicy,
		StepID:  stepID,
		Message: message,
	}
}
