package missioncontrol

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestMissionModelPolicyJSONRoundTrip(t *testing.T) {
	t.Parallel()

	supportsTools := false
	local := true
	allowFallback := false
	allowCloud := false
	job := testJob([]Step{
		{
			ID:   "route",
			Type: StepTypeDiscussion,
			ModelPolicy: &ModelPolicy{
				AllowedModels: []string{"local_fast"},
				DefaultModel:  "local_fast",
				RequiredCapabilities: ModelPolicyRequiredCapabilities{
					SupportsTools:        &supportsTools,
					Local:                &local,
					AuthorityTierAtLeast: AuthorityTierLow,
				},
				AllowFallback: &allowFallback,
				AllowCloud:    &allowCloud,
			},
		},
		{ID: "final", Type: StepTypeFinalResponse, DependsOn: []string{"route"}},
	})
	job.ModelPolicy = &ModelPolicy{
		AllowedModels: []string{"local_fast", "cloud_reasoning"},
		DefaultModel:  "local_fast",
		RequiredCapabilities: ModelPolicyRequiredCapabilities{
			AuthorityTierAtLeast: AuthorityTierLow,
		},
		AllowFallback: &allowFallback,
		AllowCloud:    &allowCloud,
	}

	data, err := json.Marshal(job)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if !strings.Contains(string(data), `"model_policy"`) {
		t.Fatalf("marshaled job = %s, want model_policy", data)
	}

	var got Job
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if !reflect.DeepEqual(got, job) {
		t.Fatalf("round-trip mismatch: got %#v want %#v", got, job)
	}
	if errors := ValidatePlan(got); len(errors) != 0 {
		t.Fatalf("ValidatePlan() = %#v, want no errors", errors)
	}
}

func TestMissionModelPolicyRejectsUnknownRequiredCapabilityField(t *testing.T) {
	t.Parallel()

	data := []byte(`{
		"id": "job-1",
		"max_authority": "medium",
		"allowed_tools": ["read"],
		"model_policy": {
			"required_capabilities": {
				"supportsTools": false,
				"telepathy": true
			}
		},
		"plan": {
			"id": "plan-1",
			"steps": [
				{"id": "final", "type": "final_response"}
			]
		}
	}`)

	var job Job
	err := json.Unmarshal(data, &job)
	if err == nil {
		t.Fatal("json.Unmarshal() error = nil, want unknown capability field error")
	}
	if !strings.Contains(err.Error(), `unknown field "telepathy"`) {
		t.Fatalf("json.Unmarshal() error = %q, want unknown telepathy field", err.Error())
	}
}

func TestValidatePlanAcceptsMissingModelPolicy(t *testing.T) {
	t.Parallel()

	errors := ValidatePlan(testJob([]Step{
		{ID: "draft", Type: StepTypeDiscussion},
		{ID: "final", Type: StepTypeFinalResponse, DependsOn: []string{"draft"}},
	}))
	if len(errors) != 0 {
		t.Fatalf("ValidatePlan() = %#v, want no errors", errors)
	}
}

func TestValidatePlanRejectsInvalidModelPolicyRefs(t *testing.T) {
	t.Parallel()

	job := testJob([]Step{
		{ID: "draft", Type: StepTypeDiscussion},
		{ID: "final", Type: StepTypeFinalResponse, DependsOn: []string{"draft"}},
	})
	job.ModelPolicy = &ModelPolicy{
		AllowedModels: []string{"bad/ref"},
	}

	errors := ValidatePlan(job)
	want := ValidationError{
		Code:    RejectionCodeInvalidModelPolicy,
		Message: `job model_policy.allowed_models[0]: model_ref "bad/ref" must not contain path separators`,
	}
	if !hasValidationError(errors, want) {
		t.Fatalf("ValidatePlan() = %#v, want to include %#v", errors, want)
	}
}

func TestValidatePlanRejectsInvalidModelPolicyAuthorityTier(t *testing.T) {
	t.Parallel()

	job := testJob([]Step{
		{ID: "draft", Type: StepTypeDiscussion},
		{ID: "final", Type: StepTypeFinalResponse, DependsOn: []string{"draft"}},
	})
	job.ModelPolicy = &ModelPolicy{
		RequiredCapabilities: ModelPolicyRequiredCapabilities{
			AuthorityTierAtLeast: AuthorityTier("root"),
		},
	}

	errors := ValidatePlan(job)
	want := ValidationError{
		Code:    RejectionCodeInvalidModelPolicy,
		Message: `job model_policy.required_capabilities.authorityTierAtLeast "root" is invalid`,
	}
	if !hasValidationError(errors, want) {
		t.Fatalf("ValidatePlan() = %#v, want to include %#v", errors, want)
	}
}

func TestValidatePlanEnforcesStepModelPolicyNarrowing(t *testing.T) {
	t.Parallel()

	allowCloud := false
	job := testJob([]Step{
		{
			ID:   "draft",
			Type: StepTypeDiscussion,
			ModelPolicy: &ModelPolicy{
				AllowedModels: []string{"local_fast"},
				DefaultModel:  "local_fast",
				AllowCloud:    &allowCloud,
				RequiredCapabilities: ModelPolicyRequiredCapabilities{
					AuthorityTierAtLeast: AuthorityTierHigh,
				},
			},
		},
		{ID: "final", Type: StepTypeFinalResponse, DependsOn: []string{"draft"}},
	})
	job.ModelPolicy = &ModelPolicy{
		AllowedModels: []string{"local_fast", "cloud_reasoning"},
		AllowCloud:    &allowCloud,
		RequiredCapabilities: ModelPolicyRequiredCapabilities{
			AuthorityTierAtLeast: AuthorityTierMedium,
		},
	}

	if errors := ValidatePlan(job); len(errors) != 0 {
		t.Fatalf("ValidatePlan() = %#v, want no errors", errors)
	}
}

func TestValidatePlanRejectsContradictoryStepModelPolicy(t *testing.T) {
	t.Parallel()

	allowCloudJob := false
	allowCloudStep := true
	supportsToolsJob := true
	supportsToolsStep := false
	job := testJob([]Step{
		{
			ID:   "draft",
			Type: StepTypeDiscussion,
			ModelPolicy: &ModelPolicy{
				AllowedModels: []string{"local_fast"},
				DefaultModel:  "cloud_reasoning",
				AllowCloud:    &allowCloudStep,
				RequiredCapabilities: ModelPolicyRequiredCapabilities{
					SupportsTools:        &supportsToolsStep,
					AuthorityTierAtLeast: AuthorityTierLow,
				},
			},
		},
		{ID: "final", Type: StepTypeFinalResponse, DependsOn: []string{"draft"}},
	})
	job.ModelPolicy = &ModelPolicy{
		AllowedModels: []string{"cloud_reasoning"},
		AllowCloud:    &allowCloudJob,
		RequiredCapabilities: ModelPolicyRequiredCapabilities{
			SupportsTools:        &supportsToolsJob,
			AuthorityTierAtLeast: AuthorityTierMedium,
		},
	}

	errors := ValidatePlan(job)
	wants := []ValidationError{
		{
			Code:    RejectionCodeInvalidModelPolicy,
			StepID:  "draft",
			Message: `step model_policy.default_model "cloud_reasoning" is not in allowed_models`,
		},
		{
			Code:    RejectionCodeInvalidModelPolicy,
			StepID:  "draft",
			Message: `step model_policy.allowed_models includes "local_fast" outside job model_policy.allowed_models`,
		},
		{
			Code:    RejectionCodeInvalidModelPolicy,
			StepID:  "draft",
			Message: "step model_policy.allow_cloud cannot widen job model_policy.allow_cloud=false",
		},
		{
			Code:    RejectionCodeInvalidModelPolicy,
			StepID:  "draft",
			Message: "step model_policy.required_capabilities.supportsTools=false conflicts with job requirement true",
		},
		{
			Code:    RejectionCodeInvalidModelPolicy,
			StepID:  "draft",
			Message: `step model_policy.required_capabilities.authorityTierAtLeast "low" cannot be below job requirement "medium"`,
		},
	}
	for _, want := range wants {
		if !hasValidationError(errors, want) {
			t.Fatalf("ValidatePlan() = %#v, want to include %#v", errors, want)
		}
	}
}

func TestCloneJobCopiesModelPolicy(t *testing.T) {
	t.Parallel()

	allowFallback := false
	supportsTools := true
	job := testJob([]Step{
		{
			ID:   "draft",
			Type: StepTypeDiscussion,
			ModelPolicy: &ModelPolicy{
				AllowedModels: []string{"cloud_reasoning"},
				RequiredCapabilities: ModelPolicyRequiredCapabilities{
					SupportsTools: &supportsTools,
				},
			},
		},
		{ID: "final", Type: StepTypeFinalResponse, DependsOn: []string{"draft"}},
	})
	job.ModelPolicy = &ModelPolicy{
		AllowedModels: []string{"cloud_reasoning"},
		AllowFallback: &allowFallback,
	}

	cloned := CloneJob(&job)
	cloned.ModelPolicy.AllowedModels[0] = "mutated"
	*cloned.ModelPolicy.AllowFallback = true
	cloned.Plan.Steps[0].ModelPolicy.AllowedModels[0] = "mutated-step"
	*cloned.Plan.Steps[0].ModelPolicy.RequiredCapabilities.SupportsTools = false

	if job.ModelPolicy.AllowedModels[0] != "cloud_reasoning" {
		t.Fatalf("job model policy allowed_models mutated to %q", job.ModelPolicy.AllowedModels[0])
	}
	if *job.ModelPolicy.AllowFallback {
		t.Fatal("job model policy allow_fallback mutated to true")
	}
	if job.Plan.Steps[0].ModelPolicy.AllowedModels[0] != "cloud_reasoning" {
		t.Fatalf("step model policy allowed_models mutated to %q", job.Plan.Steps[0].ModelPolicy.AllowedModels[0])
	}
	if !*job.Plan.Steps[0].ModelPolicy.RequiredCapabilities.SupportsTools {
		t.Fatal("step model policy supportsTools mutated to false")
	}
}

func TestEffectiveModelPolicyMergesJobAndStepPolicy(t *testing.T) {
	t.Parallel()

	allowFallbackJob := true
	allowCloudJob := false
	allowFallbackStep := false
	supportsTools := true
	local := true
	job := testJob([]Step{
		{
			ID:   "draft",
			Type: StepTypeDiscussion,
			ModelPolicy: &ModelPolicy{
				AllowedModels: []string{"local_fast"},
				DefaultModel:  "local_fast",
				RequiredCapabilities: ModelPolicyRequiredCapabilities{
					Local: &local,
				},
				AllowFallback: &allowFallbackStep,
			},
		},
		{ID: "final", Type: StepTypeFinalResponse, DependsOn: []string{"draft"}},
	})
	job.ModelPolicy = &ModelPolicy{
		AllowedModels: []string{"local_fast", "cloud_reasoning"},
		DefaultModel:  "cloud_reasoning",
		RequiredCapabilities: ModelPolicyRequiredCapabilities{
			SupportsTools:        &supportsTools,
			AuthorityTierAtLeast: AuthorityTierMedium,
		},
		AllowFallback: &allowFallbackJob,
		AllowCloud:    &allowCloudJob,
	}

	policy, policyID := EffectiveModelPolicy(&job, &job.Plan.Steps[0])
	if policyID != "step:model_policy" {
		t.Fatalf("policyID = %q, want step:model_policy", policyID)
	}
	if !reflect.DeepEqual(policy.AllowedModels, []string{"local_fast"}) {
		t.Fatalf("AllowedModels = %#v, want step allowed_models", policy.AllowedModels)
	}
	if policy.DefaultModel != "local_fast" {
		t.Fatalf("DefaultModel = %q, want local_fast", policy.DefaultModel)
	}
	if policy.RequiredCapabilities.SupportsTools == nil || !*policy.RequiredCapabilities.SupportsTools {
		t.Fatal("SupportsTools not inherited from job policy")
	}
	if policy.RequiredCapabilities.Local == nil || !*policy.RequiredCapabilities.Local {
		t.Fatal("Local not applied from step policy")
	}
	if policy.RequiredCapabilities.AuthorityTierAtLeast != AuthorityTierMedium {
		t.Fatalf("AuthorityTierAtLeast = %q, want medium", policy.RequiredCapabilities.AuthorityTierAtLeast)
	}
	if policy.AllowFallback == nil || *policy.AllowFallback {
		t.Fatal("AllowFallback did not use explicit step false")
	}
	if policy.AllowCloud == nil || *policy.AllowCloud {
		t.Fatal("AllowCloud did not inherit explicit job false")
	}
}
