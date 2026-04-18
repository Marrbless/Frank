package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/local/picobot/internal/missioncontrol"
)

func writeMalformedTreasuryRecordForTaskStateTest(t *testing.T, root string, treasury missioncontrol.TreasuryRecord) {
	t.Helper()

	if err := missioncontrol.WriteStoreJSONAtomic(missioncontrol.StoreTreasuryPath(root, treasury.TreasuryID), map[string]interface{}{
		"record_version":   treasury.RecordVersion,
		"treasury_id":      treasury.TreasuryID,
		"display_name":     treasury.DisplayName,
		"state":            string(treasury.State),
		"zero_seed_policy": string(treasury.ZeroSeedPolicy),
		"container_refs": []map[string]interface{}{
			{
				"kind":      string(treasury.ContainerRefs[0].Kind),
				"object_id": treasury.ContainerRefs[0].ObjectID,
			},
		},
		"created_at": treasury.CreatedAt,
		"updated_at": treasury.UpdatedAt,
	}); err != nil {
		t.Fatalf("WriteStoreJSONAtomic() error = %v", err)
	}
}

func TestTaskStateActivateStepStoresValidExecutionContext(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	job := testTaskStateJob()

	if err := state.ActivateStep(job, "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}

	ec, ok := state.ExecutionContext()
	if !ok {
		t.Fatal("ExecutionContext() ok = false, want true")
	}

	if ec.Job == nil {
		t.Fatal("ExecutionContext().Job = nil, want non-nil")
	}

	if ec.Step == nil {
		t.Fatal("ExecutionContext().Step = nil, want non-nil")
	}

	if ec.Job.ID != job.ID {
		t.Fatalf("ExecutionContext().Job.ID = %q, want %q", ec.Job.ID, job.ID)
	}

	if ec.Step.ID != "build" {
		t.Fatalf("ExecutionContext().Step.ID = %q, want %q", ec.Step.ID, "build")
	}
	if ec.Runtime == nil {
		t.Fatal("ExecutionContext().Runtime = nil, want non-nil")
	}
	if ec.Runtime.State != missioncontrol.JobStateRunning {
		t.Fatalf("ExecutionContext().Runtime.State = %q, want %q", ec.Runtime.State, missioncontrol.JobStateRunning)
	}
	if ec.Runtime.ActiveStepID != "build" {
		t.Fatalf("ExecutionContext().Runtime.ActiveStepID = %q, want %q", ec.Runtime.ActiveStepID, "build")
	}
}

func TestTaskStateActivateStepCarriesMissionStoreRoot(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	state.SetMissionStoreRoot("/tmp/mission-store")

	if err := state.ActivateStep(testTaskStateJob(), "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}

	ec, ok := state.ExecutionContext()
	if !ok {
		t.Fatal("ExecutionContext() ok = false, want true")
	}
	if ec.MissionStoreRoot != "/tmp/mission-store" {
		t.Fatalf("ExecutionContext().MissionStoreRoot = %q, want %q", ec.MissionStoreRoot, "/tmp/mission-store")
	}
}

func TestTaskStateActivateStepCarriesMissionStoreRootNormalizedIdentityModeDeclaredTargetsAndFrankObjectRefs(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	state := NewTaskState()
	state.SetMissionStoreRoot(root)

	job := testTaskStateJob()
	target := missioncontrol.AutonomyEligibilityTargetRef{
		Kind:       missioncontrol.EligibilityTargetKindProvider,
		RegistryID: "provider-mail",
	}
	job.Plan.Steps[0].GovernedExternalTargets = []missioncontrol.AutonomyEligibilityTargetRef{target}
	job.Plan.Steps[0].FrankObjectRefs = []missioncontrol.FrankRegistryObjectRef{
		{
			Kind:     missioncontrol.FrankRegistryObjectKind(" identity "),
			ObjectID: " identity-1 ",
		},
	}

	if err := state.ActivateStep(job, "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}

	ec, ok := state.ExecutionContext()
	if !ok {
		t.Fatal("ExecutionContext() ok = false, want true")
	}
	if ec.MissionStoreRoot != root {
		t.Fatalf("ExecutionContext().MissionStoreRoot = %q, want %q", ec.MissionStoreRoot, root)
	}
	if ec.Step == nil {
		t.Fatal("ExecutionContext().Step = nil, want non-nil")
	}
	if ec.Step.IdentityMode != missioncontrol.IdentityModeAgentAlias {
		t.Fatalf("ExecutionContext().Step.IdentityMode = %q, want %q", ec.Step.IdentityMode, missioncontrol.IdentityModeAgentAlias)
	}
	wantTargets := []missioncontrol.AutonomyEligibilityTargetRef{target}
	if !reflect.DeepEqual(ec.GovernedExternalTargets, wantTargets) {
		t.Fatalf("ExecutionContext().GovernedExternalTargets = %#v, want %#v", ec.GovernedExternalTargets, wantTargets)
	}
	wantRefs := []missioncontrol.FrankRegistryObjectRef{
		{
			Kind:     missioncontrol.FrankRegistryObjectKindIdentity,
			ObjectID: "identity-1",
		},
	}
	if !reflect.DeepEqual(ec.Step.FrankObjectRefs, wantRefs) {
		t.Fatalf("ExecutionContext().Step.FrankObjectRefs = %#v, want %#v", ec.Step.FrankObjectRefs, wantRefs)
	}
}

func TestTaskStateActivateStepInvalidPlanDoesNotOverwriteExistingContext(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	original := missioncontrol.ExecutionContext{
		Job:  &missioncontrol.Job{ID: "existing-job"},
		Step: &missioncontrol.Step{ID: "existing-step"},
	}
	state.SetExecutionContext(original)

	err := state.ActivateStep(missioncontrol.Job{
		ID:           "job-1",
		MaxAuthority: missioncontrol.AuthorityTierHigh,
		AllowedTools: []string{"read"},
		Plan:         missioncontrol.Plan{ID: "plan-1"},
	}, "build")
	if err == nil {
		t.Fatal("ActivateStep() error = nil, want validation error")
	}

	got, ok := state.ExecutionContext()
	if !ok {
		t.Fatal("ExecutionContext() ok = false, want true")
	}

	if !reflect.DeepEqual(got, original) {
		t.Fatalf("ExecutionContext() = %#v, want original %#v", got, original)
	}
}

func TestTaskStateActivateStepUnknownStepDoesNotOverwriteExistingContext(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	original := missioncontrol.ExecutionContext{
		Job:  &missioncontrol.Job{ID: "existing-job"},
		Step: &missioncontrol.Step{ID: "existing-step"},
	}
	state.SetExecutionContext(original)

	err := state.ActivateStep(testTaskStateJob(), "missing")
	if err == nil {
		t.Fatal("ActivateStep() error = nil, want unknown step error")
	}

	validationErr, ok := err.(missioncontrol.ValidationError)
	if !ok {
		t.Fatalf("ActivateStep() error type = %T, want ValidationError", err)
	}

	if validationErr.Code != missioncontrol.RejectionCodeUnknownStep {
		t.Fatalf("ValidationError.Code = %q, want %q", validationErr.Code, missioncontrol.RejectionCodeUnknownStep)
	}

	got, ok := state.ExecutionContext()
	if !ok {
		t.Fatal("ExecutionContext() ok = false, want true")
	}

	if !reflect.DeepEqual(got, original) {
		t.Fatalf("ExecutionContext() = %#v, want original %#v", got, original)
	}
}

func TestTaskStateExecutionContextReturnsIndependentSnapshot(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	if err := state.ActivateStep(testTaskStateJob(), "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}

	ec, ok := state.ExecutionContext()
	if !ok {
		t.Fatal("ExecutionContext() ok = false, want true")
	}

	ec.Job.AllowedTools[0] = "mutated-job-tool"
	ec.Job.Plan.Steps[0].AllowedTools[0] = "mutated-plan-step-tool"
	ec.Step.AllowedTools[0] = "mutated-step-tool"
	ec.Runtime.ActiveStepID = "mutated-runtime-step"

	stored, ok := state.ExecutionContext()
	if !ok {
		t.Fatal("ExecutionContext() ok = false, want true")
	}

	if stored.Job.AllowedTools[0] != "read" {
		t.Fatalf("stored Job.AllowedTools[0] = %q, want %q", stored.Job.AllowedTools[0], "read")
	}

	if stored.Job.Plan.Steps[0].AllowedTools[0] != "read" {
		t.Fatalf("stored Job.Plan.Steps[0].AllowedTools[0] = %q, want %q", stored.Job.Plan.Steps[0].AllowedTools[0], "read")
	}

	if stored.Step.AllowedTools[0] != "read" {
		t.Fatalf("stored Step.AllowedTools[0] = %q, want %q", stored.Step.AllowedTools[0], "read")
	}
	if stored.Runtime == nil {
		t.Fatal("stored Runtime = nil, want non-nil")
	}
	if stored.Runtime.ActiveStepID != "build" {
		t.Fatalf("stored Runtime.ActiveStepID = %q, want %q", stored.Runtime.ActiveStepID, "build")
	}
}

func TestTaskStateClearExecutionContextPreservesDurableRuntimeState(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	if err := state.ActivateStep(testTaskStateJob(), "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}

	state.ClearExecutionContext()

	got, ok := state.ExecutionContext()
	if ok {
		t.Fatalf("ExecutionContext() ok = true, want false with context %#v", got)
	}

	if !reflect.DeepEqual(got, missioncontrol.ExecutionContext{}) {
		t.Fatalf("ExecutionContext() = %#v, want zero value", got)
	}
	runtime, ok := state.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want durable runtime after clear")
	}
	if runtime.State != missioncontrol.JobStateRunning {
		t.Fatalf("MissionRuntimeState().State = %q, want %q", runtime.State, missioncontrol.JobStateRunning)
	}
	if runtime.ActiveStepID != "build" {
		t.Fatalf("MissionRuntimeState().ActiveStepID = %q, want %q", runtime.ActiveStepID, "build")
	}
}

func TestTaskStateResumeRuntimeStoresExecutionContext(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	job := testTaskStateJob()
	runtimeState := missioncontrol.JobRuntimeState{
		JobID:        job.ID,
		State:        missioncontrol.JobStateRunning,
		ActiveStepID: "build",
	}

	if err := state.ResumeRuntime(job, runtimeState, nil, true); err != nil {
		t.Fatalf("ResumeRuntime() error = %v", err)
	}

	ec, ok := state.ExecutionContext()
	if !ok {
		t.Fatal("ExecutionContext() ok = false, want true")
	}
	if ec.Runtime == nil {
		t.Fatal("ExecutionContext().Runtime = nil, want non-nil")
	}
	if ec.Runtime.ActiveStepID != "build" {
		t.Fatalf("ExecutionContext().Runtime.ActiveStepID = %q, want %q", ec.Runtime.ActiveStepID, "build")
	}
}

func TestTaskStateResumeRuntimeCarriesMissionStoreRoot(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	state.SetMissionStoreRoot("/tmp/mission-store")
	job := testTaskStateJob()
	runtimeState := missioncontrol.JobRuntimeState{
		JobID:        job.ID,
		State:        missioncontrol.JobStateRunning,
		ActiveStepID: "build",
	}

	if err := state.ResumeRuntime(job, runtimeState, nil, true); err != nil {
		t.Fatalf("ResumeRuntime() error = %v", err)
	}

	ec, ok := state.ExecutionContext()
	if !ok {
		t.Fatal("ExecutionContext() ok = false, want true")
	}
	if ec.MissionStoreRoot != "/tmp/mission-store" {
		t.Fatalf("ExecutionContext().MissionStoreRoot = %q, want %q", ec.MissionStoreRoot, "/tmp/mission-store")
	}
}

func TestTaskStateZeroTargetExecutionWithoutMissionStoreRootPreservesV2Behavior(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	if err := state.ActivateStep(testTaskStateJob(), "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}

	ec, ok := state.ExecutionContext()
	if !ok {
		t.Fatal("ExecutionContext() ok = false, want true")
	}
	if ec.MissionStoreRoot != "" {
		t.Fatalf("ExecutionContext().MissionStoreRoot = %q, want empty", ec.MissionStoreRoot)
	}
	if ec.Step == nil {
		t.Fatal("ExecutionContext().Step = nil, want non-nil")
	}
	if ec.Step.IdentityMode != missioncontrol.IdentityModeAgentAlias {
		t.Fatalf("ExecutionContext().Step.IdentityMode = %q, want %q", ec.Step.IdentityMode, missioncontrol.IdentityModeAgentAlias)
	}
	if ec.GovernedExternalTargets != nil {
		t.Fatalf("ExecutionContext().GovernedExternalTargets = %#v, want nil", ec.GovernedExternalTargets)
	}
	if ec.Step.FrankObjectRefs != nil {
		t.Fatalf("ExecutionContext().Step.FrankObjectRefs = %#v, want nil", ec.Step.FrankObjectRefs)
	}

	decision := missioncontrol.NewDefaultToolGuard().EvaluateTool(context.Background(), ec, "read", nil)
	if !decision.Allowed {
		t.Fatalf("EvaluateTool().Allowed = false, want true: %#v", decision)
	}
	if decision.Code != "" {
		t.Fatalf("EvaluateTool().Code = %q, want empty", decision.Code)
	}
	if decision.Reason != "" {
		t.Fatalf("EvaluateTool().Reason = %q, want empty", decision.Reason)
	}
}

func TestTaskStateOrdinaryRuntimePathProvidesStoreRootToAutonomyGuard(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 7, 15, 0, 0, 0, time.UTC)
	target := missioncontrol.AutonomyEligibilityTargetRef{
		Kind:       missioncontrol.EligibilityTargetKindProvider,
		RegistryID: "provider-human-id",
	}
	writeTaskStateAutonomyEligibilityFixture(t, root, target, missioncontrol.PlatformRecord{
		PlatformID:       target.RegistryID,
		PlatformName:     "human-id.example",
		TargetClass:      target.Kind,
		EligibilityLabel: missioncontrol.EligibilityLabelIneligible,
		LastCheckID:      "check-provider-human-id",
		Notes:            []string{"registry note"},
		UpdatedAt:        now,
	}, missioncontrol.EligibilityCheckRecord{
		CheckID:                     "check-provider-human-id",
		TargetKind:                  target.Kind,
		TargetName:                  "human-id.example",
		CanCreateWithoutOwner:       false,
		CanOnboardWithoutOwner:      false,
		CanControlAsAgent:           false,
		CanRecoverAsAgent:           false,
		RequiresHumanOnlyStep:       true,
		RequiresOwnerOnlySecretOrID: true,
		RulesAsObservedOK:           false,
		Label:                       missioncontrol.EligibilityLabelIneligible,
		Reasons:                     []string{string(missioncontrol.AutonomyEligibilityReasonOwnerIdentityRequired)},
		CheckedAt:                   now,
	})

	state := NewTaskState()
	state.SetMissionStoreRoot(root)
	job := testTaskStateJob()
	job.Plan.Steps[0].GovernedExternalTargets = []missioncontrol.AutonomyEligibilityTargetRef{target}

	if err := state.ActivateStep(job, "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}

	ec, ok := state.ExecutionContext()
	if !ok {
		t.Fatal("ExecutionContext() ok = false, want true")
	}
	_, wantErr := missioncontrol.RequireAutonomyEligibleTarget(root, target)
	if wantErr == nil {
		t.Fatal("RequireAutonomyEligibleTarget() error = nil, want fail-closed error")
	}

	decision := missioncontrol.NewDefaultToolGuard().EvaluateTool(context.Background(), ec, "read", nil)
	if decision.Allowed {
		t.Fatalf("EvaluateTool().Allowed = true, want false: %#v", decision)
	}
	if decision.Code != missioncontrol.RejectionCodeInvalidRuntimeState {
		t.Fatalf("EvaluateTool().Code = %q, want %q", decision.Code, missioncontrol.RejectionCodeInvalidRuntimeState)
	}
	if decision.Reason != wantErr.Error() {
		t.Fatalf("EvaluateTool().Reason = %q, want %q", decision.Reason, wantErr.Error())
	}
}

func TestTaskStateResumeRuntimeRejectsCompletedActiveStepReplay(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	job := testTaskStateJob()
	runtimeState := missioncontrol.JobRuntimeState{
		JobID:        job.ID,
		State:        missioncontrol.JobStatePaused,
		ActiveStepID: "build",
		CompletedSteps: []missioncontrol.RuntimeStepRecord{
			{StepID: "build", At: time.Date(2026, 3, 27, 12, 0, 0, 0, time.UTC)},
		},
	}

	err := state.ResumeRuntime(job, runtimeState, nil, true)
	if err == nil {
		t.Fatal("ResumeRuntime() error = nil, want completed-step replay rejection")
	}

	validationErr, ok := err.(missioncontrol.ValidationError)
	if !ok {
		t.Fatalf("ResumeRuntime() error type = %T, want ValidationError", err)
	}
	if validationErr.Code != missioncontrol.RejectionCodeInvalidRuntimeState {
		t.Fatalf("ValidationError.Code = %q, want %q", validationErr.Code, missioncontrol.RejectionCodeInvalidRuntimeState)
	}
	if _, ok := state.ExecutionContext(); ok {
		t.Fatal("ExecutionContext() ok = true, want false after rejected replay resume")
	}
}

func TestTaskStateActivateStepRejectsPreviouslyCompletedStepReplay(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	job := testTaskStateJob()
	if err := state.HydrateRuntimeControl(job, missioncontrol.JobRuntimeState{
		JobID:        job.ID,
		State:        missioncontrol.JobStatePaused,
		ActiveStepID: "final",
		PausedReason: missioncontrol.RuntimePauseReasonOperatorCommand,
		CompletedSteps: []missioncontrol.RuntimeStepRecord{
			{StepID: "build", At: time.Date(2026, 3, 27, 12, 0, 0, 0, time.UTC)},
		},
	}, nil); err != nil {
		t.Fatalf("HydrateRuntimeControl() setup error = %v", err)
	}

	err := state.ActivateStep(job, "build")
	if err == nil {
		t.Fatal("ActivateStep() error = nil, want completed-step replay rejection")
	}

	validationErr, ok := err.(missioncontrol.ValidationError)
	if !ok {
		t.Fatalf("ActivateStep() error type = %T, want ValidationError", err)
	}
	if validationErr.Code != missioncontrol.RejectionCodeInvalidRuntimeState {
		t.Fatalf("ValidationError.Code = %q, want %q", validationErr.Code, missioncontrol.RejectionCodeInvalidRuntimeState)
	}
}

func TestTaskStateActivateStepRejectsPreviouslyFailedStepReplay(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	job := testTaskStateJob()
	if err := state.HydrateRuntimeControl(job, missioncontrol.JobRuntimeState{
		JobID:        job.ID,
		State:        missioncontrol.JobStatePaused,
		ActiveStepID: "final",
		PausedReason: missioncontrol.RuntimePauseReasonOperatorCommand,
		FailedSteps: []missioncontrol.RuntimeStepRecord{
			{StepID: "build", Reason: "validator failed", At: time.Date(2026, 3, 27, 12, 0, 0, 0, time.UTC)},
		},
	}, nil); err != nil {
		t.Fatalf("HydrateRuntimeControl() setup error = %v", err)
	}

	err := state.ActivateStep(job, "build")
	if err == nil {
		t.Fatal("ActivateStep() error = nil, want failed-step replay rejection")
	}

	validationErr, ok := err.(missioncontrol.ValidationError)
	if !ok {
		t.Fatalf("ActivateStep() error type = %T, want ValidationError", err)
	}
	if validationErr.Code != missioncontrol.RejectionCodeInvalidRuntimeState {
		t.Fatalf("ValidationError.Code = %q, want %q", validationErr.Code, missioncontrol.RejectionCodeInvalidRuntimeState)
	}
}

func TestTaskStateActivateStepAllowsDifferentJobAfterTerminalRuntime(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	previous := testTaskStateJob()
	previous.ID = "previous-job"
	if err := state.HydrateRuntimeControl(previous, missioncontrol.JobRuntimeState{
		JobID:       previous.ID,
		State:       missioncontrol.JobStateCompleted,
		CompletedAt: time.Date(2026, 4, 12, 15, 0, 0, 0, time.UTC),
		CompletedSteps: []missioncontrol.RuntimeStepRecord{
			{StepID: "build", At: time.Date(2026, 4, 12, 14, 59, 0, 0, time.UTC)},
		},
	}, nil); err != nil {
		t.Fatalf("HydrateRuntimeControl() setup error = %v", err)
	}

	next := testTaskStateJob()
	next.ID = "next-job"
	if err := state.ActivateStep(next, "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}

	ec, ok := state.ExecutionContext()
	if !ok {
		t.Fatal("ExecutionContext() ok = false, want true")
	}
	if ec.Job == nil || ec.Job.ID != "next-job" {
		t.Fatalf("ExecutionContext().Job = %+v, want next-job", ec.Job)
	}
	if ec.Runtime == nil {
		t.Fatal("ExecutionContext().Runtime = nil, want running runtime")
	}
	if ec.Runtime.JobID != "next-job" {
		t.Fatalf("ExecutionContext().Runtime.JobID = %q, want %q", ec.Runtime.JobID, "next-job")
	}
	if ec.Runtime.State != missioncontrol.JobStateRunning {
		t.Fatalf("ExecutionContext().Runtime.State = %q, want %q", ec.Runtime.State, missioncontrol.JobStateRunning)
	}
}

func TestTaskStateActivateStepRejectsDifferentJobWhileAnotherRuntimeRunning(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	current := testTaskStateJob()
	current.ID = "current-job"
	if err := state.ActivateStep(current, "build"); err != nil {
		t.Fatalf("ActivateStep(current) error = %v", err)
	}

	next := testTaskStateJob()
	next.ID = "next-job"
	err := state.ActivateStep(next, "build")
	if err == nil {
		t.Fatal("ActivateStep(next) error = nil, want running-job mismatch rejection")
	}
	if !strings.Contains(err.Error(), `runtime job "current-job" does not match mission job "next-job"`) {
		t.Fatalf("ActivateStep(next) error = %q, want running-job mismatch rejection", err)
	}

	ec, ok := state.ExecutionContext()
	if !ok {
		t.Fatal("ExecutionContext() ok = false, want true")
	}
	if ec.Job == nil || ec.Job.ID != "current-job" {
		t.Fatalf("ExecutionContext().Job = %+v, want current-job", ec.Job)
	}
	if ec.Runtime == nil || ec.Runtime.JobID != "current-job" {
		t.Fatalf("ExecutionContext().Runtime = %+v, want current running job", ec.Runtime)
	}
}

func TestTaskStateActivateStepTreasuryPathCallsActivationProducerOnce(t *testing.T) {
	t.Parallel()

	root, treasury, _ := writeTaskStateTreasuryFixtures(t)
	job := testTaskStateJob()
	job.Plan.Steps[0].TreasuryRef = &missioncontrol.TreasuryRef{TreasuryID: treasury.TreasuryID}

	state := NewTaskState()
	state.SetMissionStoreRoot(root)

	calls := 0
	var gotRoot string
	var gotLease missioncontrol.WriterLockLease
	var gotInput missioncontrol.DefaultTreasuryActivationPolicyInput
	var gotAt time.Time
	state.treasuryActivationProducerHook = func(root string, lease missioncontrol.WriterLockLease, input missioncontrol.DefaultTreasuryActivationPolicyInput, now time.Time) error {
		calls++
		gotRoot = root
		gotLease = lease
		gotInput = input
		gotAt = now
		return nil
	}

	if err := state.ActivateStep(job, "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}

	if calls != 1 {
		t.Fatalf("treasuryActivationProducerHook calls = %d, want 1", calls)
	}
	if gotRoot != root {
		t.Fatalf("treasuryActivationProducerHook root = %q, want %q", gotRoot, root)
	}
	if gotLease.LeaseHolderID != taskStateTreasuryExecutionLeaseHolderID {
		t.Fatalf("treasuryActivationProducerHook lease = %#v, want %q", gotLease, taskStateTreasuryExecutionLeaseHolderID)
	}
	if !reflect.DeepEqual(gotInput, missioncontrol.DefaultTreasuryActivationPolicyInput{
		TreasuryRef: missioncontrol.TreasuryRef{TreasuryID: treasury.TreasuryID},
	}) {
		t.Fatalf("treasuryActivationProducerHook input = %#v, want treasury ref %q", gotInput, treasury.TreasuryID)
	}
	if gotAt.IsZero() {
		t.Fatal("treasuryActivationProducerHook now = zero, want activation timestamp")
	}

	runtime, ok := state.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want stored runtime")
	}
	if runtime.State != missioncontrol.JobStateRunning || runtime.ActiveStepID != "build" {
		t.Fatalf("MissionRuntimeState() = %#v, want running build runtime", runtime)
	}
}

func TestTaskStateActivateStepTreasuryBootstrapPathCallsMutationThenProducerOnce(t *testing.T) {
	t.Parallel()

	root, treasury, _ := writeTaskStateTreasuryBootstrapFixtures(t)
	job := testTaskStateJob()
	job.Plan.Steps[0].TreasuryRef = &missioncontrol.TreasuryRef{TreasuryID: treasury.TreasuryID}

	state := NewTaskState()
	state.SetMissionStoreRoot(root)

	acquisitionCalls := 0
	bootstrapCalls := 0
	activationCalls := 0

	var gotAcquisitionRoot string
	var gotAcquisitionLease missioncontrol.WriterLockLease
	var gotAcquisitionInput missioncontrol.FirstTreasuryAcquisitionInput
	var gotBootstrapRoot string
	var gotBootstrapLease missioncontrol.WriterLockLease
	var gotBootstrapInput missioncontrol.FirstValueTreasuryBootstrapInput

	state.treasuryFirstAcquisitionHook = func(root string, lease missioncontrol.WriterLockLease, input missioncontrol.FirstTreasuryAcquisitionInput, now time.Time) error {
		acquisitionCalls++
		gotAcquisitionRoot = root
		gotAcquisitionLease = lease
		gotAcquisitionInput = input
		return nil
	}
	state.treasuryBootstrapProducerHook = func(root string, lease missioncontrol.WriterLockLease, input missioncontrol.FirstValueTreasuryBootstrapInput, now time.Time) error {
		bootstrapCalls++
		gotBootstrapRoot = root
		gotBootstrapLease = lease
		gotBootstrapInput = input
		return nil
	}
	state.treasuryActivationProducerHook = func(root string, lease missioncontrol.WriterLockLease, input missioncontrol.DefaultTreasuryActivationPolicyInput, now time.Time) error {
		activationCalls++
		return nil
	}

	if err := state.ActivateStep(job, "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}

	if acquisitionCalls != 1 {
		t.Fatalf("treasuryFirstAcquisitionHook calls = %d, want 1", acquisitionCalls)
	}
	if bootstrapCalls != 1 {
		t.Fatalf("treasuryBootstrapProducerHook calls = %d, want 1", bootstrapCalls)
	}
	if activationCalls != 0 {
		t.Fatalf("treasuryActivationProducerHook calls = %d, want 0 on bootstrap path", activationCalls)
	}
	if gotAcquisitionRoot != root {
		t.Fatalf("treasuryFirstAcquisitionHook root = %q, want %q", gotAcquisitionRoot, root)
	}
	if gotAcquisitionLease.LeaseHolderID != taskStateTreasuryExecutionLeaseHolderID {
		t.Fatalf("treasuryFirstAcquisitionHook lease = %#v, want %q", gotAcquisitionLease, taskStateTreasuryExecutionLeaseHolderID)
	}
	if !reflect.DeepEqual(gotAcquisitionInput, missioncontrol.FirstTreasuryAcquisitionInput{
		TreasuryID: treasury.TreasuryID,
		EntryID:    treasury.BootstrapAcquisition.EntryID,
		AssetCode:  treasury.BootstrapAcquisition.AssetCode,
		Amount:     treasury.BootstrapAcquisition.Amount,
		SourceRef:  treasury.BootstrapAcquisition.SourceRef,
	}) {
		t.Fatalf("treasuryFirstAcquisitionHook input = %#v, want committed bootstrap acquisition payload", gotAcquisitionInput)
	}
	if gotBootstrapRoot != root {
		t.Fatalf("treasuryBootstrapProducerHook root = %q, want %q", gotBootstrapRoot, root)
	}
	if gotBootstrapLease.LeaseHolderID != taskStateTreasuryExecutionLeaseHolderID {
		t.Fatalf("treasuryBootstrapProducerHook lease = %#v, want %q", gotBootstrapLease, taskStateTreasuryExecutionLeaseHolderID)
	}
	if !reflect.DeepEqual(gotBootstrapInput, missioncontrol.FirstValueTreasuryBootstrapInput{
		TreasuryRef: missioncontrol.TreasuryRef{TreasuryID: treasury.TreasuryID},
		EntryID:     treasury.BootstrapAcquisition.EntryID,
	}) {
		t.Fatalf("treasuryBootstrapProducerHook input = %#v, want committed first-entry bootstrap input", gotBootstrapInput)
	}
}

func TestTaskStateActivateStepTreasuryBootstrapPathInvokesRealMutationAndProducer(t *testing.T) {
	t.Parallel()

	root, treasury, _ := writeTaskStateTreasuryBootstrapFixtures(t)
	job := testTaskStateJob()
	job.Plan.Steps[0].TreasuryRef = &missioncontrol.TreasuryRef{TreasuryID: treasury.TreasuryID}

	state := NewTaskState()
	state.SetMissionStoreRoot(root)

	if err := state.ActivateStep(job, "build"); err != nil {
		t.Fatalf("ActivateStep(first) error = %v", err)
	}

	firstRuntime, ok := state.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want stored runtime after bootstrap activation")
	}
	firstTreasury, err := missioncontrol.LoadTreasuryRecord(root, treasury.TreasuryID)
	if err != nil {
		t.Fatalf("LoadTreasuryRecord(first) error = %v", err)
	}
	if firstTreasury.State != missioncontrol.TreasuryStateActive {
		t.Fatalf("LoadTreasuryRecord(first).State = %q, want %q", firstTreasury.State, missioncontrol.TreasuryStateActive)
	}
	entries, err := missioncontrol.ListTreasuryLedgerEntries(root, treasury.TreasuryID)
	if err != nil {
		t.Fatalf("ListTreasuryLedgerEntries(first) error = %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("ListTreasuryLedgerEntries(first) len = %d, want 1", len(entries))
	}
	if entries[0].EntryID != treasury.BootstrapAcquisition.EntryID || entries[0].EntryKind != missioncontrol.TreasuryLedgerEntryKindAcquisition {
		t.Fatalf("ListTreasuryLedgerEntries(first) = %#v, want one committed acquisition entry", entries)
	}

	err = state.ActivateStep(job, "build")
	if err == nil {
		t.Fatal("ActivateStep(replay) error = nil, want deterministic treasury bootstrap rejection")
	}
	if !strings.Contains(err.Error(), `execution context treasury "treasury-wallet" requires committed treasury.post_bootstrap_acquisition for additional acquisition`) {
		t.Fatalf("ActivateStep(replay) error = %q, want missing post-bootstrap acquisition rejection", err.Error())
	}

	secondRuntime, ok := state.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want unchanged runtime after replay rejection")
	}
	if !reflect.DeepEqual(secondRuntime, firstRuntime) {
		t.Fatalf("MissionRuntimeState() = %#v, want unchanged %#v", secondRuntime, firstRuntime)
	}
	secondTreasury, err := missioncontrol.LoadTreasuryRecord(root, treasury.TreasuryID)
	if err != nil {
		t.Fatalf("LoadTreasuryRecord(second) error = %v", err)
	}
	if !reflect.DeepEqual(secondTreasury, firstTreasury) {
		t.Fatalf("LoadTreasuryRecord(second) = %#v, want unchanged %#v", secondTreasury, firstTreasury)
	}
	secondEntries, err := missioncontrol.ListTreasuryLedgerEntries(root, treasury.TreasuryID)
	if err != nil {
		t.Fatalf("ListTreasuryLedgerEntries(second) error = %v", err)
	}
	if !reflect.DeepEqual(secondEntries, entries) {
		t.Fatalf("ListTreasuryLedgerEntries(second) = %#v, want unchanged %#v", secondEntries, entries)
	}
}

func TestTaskStateActivateStepActiveTreasuryPathCallsPostAcquisitionHookOnce(t *testing.T) {
	t.Parallel()

	root, treasury, _ := writeTaskStateActiveTreasuryAcquisitionFixtures(t)
	job := testTaskStateJob()
	job.Plan.Steps[0].TreasuryRef = &missioncontrol.TreasuryRef{TreasuryID: treasury.TreasuryID}

	state := NewTaskState()
	state.SetMissionStoreRoot(root)

	postCalls := 0
	activationCalls := 0
	var gotRoot string
	var gotLease missioncontrol.WriterLockLease
	var gotInput missioncontrol.PostBootstrapTreasuryAcquisitionInput
	var gotAt time.Time
	state.treasuryPostAcquisitionHook = func(root string, lease missioncontrol.WriterLockLease, input missioncontrol.PostBootstrapTreasuryAcquisitionInput, now time.Time) error {
		postCalls++
		gotRoot = root
		gotLease = lease
		gotInput = input
		gotAt = now
		return nil
	}
	state.treasuryActivationProducerHook = func(root string, lease missioncontrol.WriterLockLease, input missioncontrol.DefaultTreasuryActivationPolicyInput, now time.Time) error {
		activationCalls++
		return nil
	}

	if err := state.ActivateStep(job, "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}

	if postCalls != 1 {
		t.Fatalf("treasuryPostAcquisitionHook calls = %d, want 1", postCalls)
	}
	if activationCalls != 0 {
		t.Fatalf("treasuryActivationProducerHook calls = %d, want 0 on active acquisition path", activationCalls)
	}
	if gotRoot != root {
		t.Fatalf("treasuryPostAcquisitionHook root = %q, want %q", gotRoot, root)
	}
	if gotLease.LeaseHolderID != taskStateTreasuryExecutionLeaseHolderID {
		t.Fatalf("treasuryPostAcquisitionHook lease = %#v, want %q", gotLease, taskStateTreasuryExecutionLeaseHolderID)
	}
	if !reflect.DeepEqual(gotInput, missioncontrol.PostBootstrapTreasuryAcquisitionInput{
		TreasuryID: treasury.TreasuryID,
	}) {
		t.Fatalf("treasuryPostAcquisitionHook input = %#v, want treasury id %q", gotInput, treasury.TreasuryID)
	}
	if gotAt.IsZero() {
		t.Fatal("treasuryPostAcquisitionHook now = zero, want activation timestamp")
	}
}

func TestTaskStateActivateStepActiveTreasuryPathInvokesRealMutation(t *testing.T) {
	t.Parallel()

	root, treasury, _ := writeTaskStateActiveTreasuryAcquisitionFixtures(t)
	job := testTaskStateJob()
	job.Plan.Steps[0].TreasuryRef = &missioncontrol.TreasuryRef{TreasuryID: treasury.TreasuryID}

	state := NewTaskState()
	state.SetMissionStoreRoot(root)

	if err := state.ActivateStep(job, "build"); err != nil {
		t.Fatalf("ActivateStep(first) error = %v", err)
	}

	firstTreasury, err := missioncontrol.LoadTreasuryRecord(root, treasury.TreasuryID)
	if err != nil {
		t.Fatalf("LoadTreasuryRecord(first) error = %v", err)
	}
	if firstTreasury.PostBootstrapAcquisition == nil || firstTreasury.PostBootstrapAcquisition.ConsumedEntryID == "" {
		t.Fatalf("LoadTreasuryRecord(first).PostBootstrapAcquisition = %#v, want consumed entry linkage", firstTreasury.PostBootstrapAcquisition)
	}
	entries, err := missioncontrol.ListTreasuryLedgerEntries(root, treasury.TreasuryID)
	if err != nil {
		t.Fatalf("ListTreasuryLedgerEntries(first) error = %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("ListTreasuryLedgerEntries(first) len = %d, want 1", len(entries))
	}
	if entries[0].EntryID != firstTreasury.PostBootstrapAcquisition.ConsumedEntryID || entries[0].EntryKind != missioncontrol.TreasuryLedgerEntryKindAcquisition {
		t.Fatalf("ListTreasuryLedgerEntries(first) = %#v, want one committed post-bootstrap acquisition entry", entries)
	}

	err = state.ActivateStep(job, "build")
	if err == nil {
		t.Fatal("ActivateStep(replay) error = nil, want deterministic consumed post-bootstrap rejection")
	}
	if !strings.Contains(err.Error(), `execution context treasury "treasury-wallet" treasury.post_bootstrap_acquisition is already consumed by entry "`) {
		t.Fatalf("ActivateStep(replay) error = %q, want consumed post-bootstrap rejection", err.Error())
	}

	secondTreasury, err := missioncontrol.LoadTreasuryRecord(root, treasury.TreasuryID)
	if err != nil {
		t.Fatalf("LoadTreasuryRecord(second) error = %v", err)
	}
	if !reflect.DeepEqual(secondTreasury, firstTreasury) {
		t.Fatalf("LoadTreasuryRecord(second) = %#v, want unchanged %#v", secondTreasury, firstTreasury)
	}
	secondEntries, err := missioncontrol.ListTreasuryLedgerEntries(root, treasury.TreasuryID)
	if err != nil {
		t.Fatalf("ListTreasuryLedgerEntries(second) error = %v", err)
	}
	if !reflect.DeepEqual(secondEntries, entries) {
		t.Fatalf("ListTreasuryLedgerEntries(second) = %#v, want unchanged %#v", secondEntries, entries)
	}
}

func TestTaskStateActivateStepActiveTreasurySuspendPathCallsSuspendProducerOnce(t *testing.T) {
	t.Parallel()

	root, treasury := writeTaskStateActiveTreasurySuspendFixtures(t)
	job := testTaskStateJob()
	job.Plan.Steps[0].TreasuryRef = &missioncontrol.TreasuryRef{TreasuryID: treasury.TreasuryID}

	state := NewTaskState()
	state.SetMissionStoreRoot(root)

	suspendCalls := 0
	allocateCalls := 0
	var gotRoot string
	var gotLease missioncontrol.WriterLockLease
	var gotInput missioncontrol.PostActiveTreasurySuspendInput
	var gotAt time.Time
	state.treasuryPostActiveSuspendHook = func(root string, lease missioncontrol.WriterLockLease, input missioncontrol.PostActiveTreasurySuspendInput, now time.Time) error {
		suspendCalls++
		gotRoot = root
		gotLease = lease
		gotInput = input
		gotAt = now
		return nil
	}
	state.treasuryPostActiveAllocateHook = func(root string, lease missioncontrol.WriterLockLease, input missioncontrol.PostActiveTreasuryAllocateInput, now time.Time) error {
		allocateCalls++
		return nil
	}

	if err := state.ActivateStep(job, "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}

	if suspendCalls != 1 {
		t.Fatalf("treasuryPostActiveSuspendHook calls = %d, want 1", suspendCalls)
	}
	if allocateCalls != 0 {
		t.Fatalf("treasuryPostActiveAllocateHook calls = %d, want 0 on active suspend path", allocateCalls)
	}
	if gotRoot != root {
		t.Fatalf("treasuryPostActiveSuspendHook root = %q, want %q", gotRoot, root)
	}
	if gotLease.LeaseHolderID != taskStateTreasuryExecutionLeaseHolderID {
		t.Fatalf("treasuryPostActiveSuspendHook lease = %#v, want %q", gotLease, taskStateTreasuryExecutionLeaseHolderID)
	}
	if !reflect.DeepEqual(gotInput, missioncontrol.PostActiveTreasurySuspendInput{
		TreasuryRef: missioncontrol.TreasuryRef{TreasuryID: treasury.TreasuryID},
	}) {
		t.Fatalf("treasuryPostActiveSuspendHook input = %#v, want treasury ref %q", gotInput, treasury.TreasuryID)
	}
	if gotAt.IsZero() {
		t.Fatal("treasuryPostActiveSuspendHook now = zero, want activation timestamp")
	}
}

func TestTaskStateActivateStepActiveTreasurySuspendPathInvokesRealProducer(t *testing.T) {
	t.Parallel()

	root, treasury := writeTaskStateActiveTreasurySuspendFixtures(t)
	job := testTaskStateJob()
	job.Plan.Steps[0].TreasuryRef = &missioncontrol.TreasuryRef{TreasuryID: treasury.TreasuryID}

	state := NewTaskState()
	state.SetMissionStoreRoot(root)

	if err := state.ActivateStep(job, "build"); err != nil {
		t.Fatalf("ActivateStep(first) error = %v", err)
	}

	firstTreasury, err := missioncontrol.LoadTreasuryRecord(root, treasury.TreasuryID)
	if err != nil {
		t.Fatalf("LoadTreasuryRecord(first) error = %v", err)
	}
	if firstTreasury.State != missioncontrol.TreasuryStateSuspended {
		t.Fatalf("LoadTreasuryRecord(first).State = %q, want %q", firstTreasury.State, missioncontrol.TreasuryStateSuspended)
	}
	if firstTreasury.PostActiveSuspend == nil || firstTreasury.PostActiveSuspend.ConsumedTransitionID == "" {
		t.Fatalf("LoadTreasuryRecord(first).PostActiveSuspend = %#v, want consumed transition linkage", firstTreasury.PostActiveSuspend)
	}

	if err := state.ActivateStep(job, "build"); err != nil {
		t.Fatalf("ActivateStep(second) error = %v, want suspended no-op", err)
	}

	secondTreasury, err := missioncontrol.LoadTreasuryRecord(root, treasury.TreasuryID)
	if err != nil {
		t.Fatalf("LoadTreasuryRecord(second) error = %v", err)
	}
	if !reflect.DeepEqual(secondTreasury, firstTreasury) {
		t.Fatalf("LoadTreasuryRecord(second) = %#v, want unchanged %#v", secondTreasury, firstTreasury)
	}
}

func TestTaskStateActivateStepSuspendedTreasuryResumePathCallsResumeProducerOnce(t *testing.T) {
	t.Parallel()

	root, treasury := writeTaskStateSuspendedTreasuryResumeFixtures(t)
	job := testTaskStateJob()
	job.Plan.Steps[0].TreasuryRef = &missioncontrol.TreasuryRef{TreasuryID: treasury.TreasuryID}

	state := NewTaskState()
	state.SetMissionStoreRoot(root)

	resumeCalls := 0
	suspendCalls := 0
	var gotRoot string
	var gotLease missioncontrol.WriterLockLease
	var gotInput missioncontrol.PostSuspendTreasuryResumeInput
	var gotAt time.Time
	state.treasuryPostSuspendResumeHook = func(root string, lease missioncontrol.WriterLockLease, input missioncontrol.PostSuspendTreasuryResumeInput, now time.Time) error {
		resumeCalls++
		gotRoot = root
		gotLease = lease
		gotInput = input
		gotAt = now
		return nil
	}
	state.treasuryPostActiveSuspendHook = func(root string, lease missioncontrol.WriterLockLease, input missioncontrol.PostActiveTreasurySuspendInput, now time.Time) error {
		suspendCalls++
		return nil
	}

	if err := state.ActivateStep(job, "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}

	if resumeCalls != 1 {
		t.Fatalf("treasuryPostSuspendResumeHook calls = %d, want 1", resumeCalls)
	}
	if suspendCalls != 0 {
		t.Fatalf("treasuryPostActiveSuspendHook calls = %d, want 0 on suspended resume path", suspendCalls)
	}
	if gotRoot != root {
		t.Fatalf("treasuryPostSuspendResumeHook root = %q, want %q", gotRoot, root)
	}
	if gotLease.LeaseHolderID != taskStateTreasuryExecutionLeaseHolderID {
		t.Fatalf("treasuryPostSuspendResumeHook lease = %#v, want %q", gotLease, taskStateTreasuryExecutionLeaseHolderID)
	}
	if !reflect.DeepEqual(gotInput, missioncontrol.PostSuspendTreasuryResumeInput{
		TreasuryRef: missioncontrol.TreasuryRef{TreasuryID: treasury.TreasuryID},
	}) {
		t.Fatalf("treasuryPostSuspendResumeHook input = %#v, want treasury ref %q", gotInput, treasury.TreasuryID)
	}
	if gotAt.IsZero() {
		t.Fatal("treasuryPostSuspendResumeHook now = zero, want activation timestamp")
	}
}

func TestTaskStateActivateStepSuspendedTreasuryResumePathInvokesRealProducer(t *testing.T) {
	t.Parallel()

	root, treasury := writeTaskStateSuspendedTreasuryResumeFixtures(t)
	job := testTaskStateJob()
	job.Plan.Steps[0].TreasuryRef = &missioncontrol.TreasuryRef{TreasuryID: treasury.TreasuryID}

	state := NewTaskState()
	state.SetMissionStoreRoot(root)

	if err := state.ActivateStep(job, "build"); err != nil {
		t.Fatalf("ActivateStep(first) error = %v", err)
	}

	firstTreasury, err := missioncontrol.LoadTreasuryRecord(root, treasury.TreasuryID)
	if err != nil {
		t.Fatalf("LoadTreasuryRecord(first) error = %v", err)
	}
	if firstTreasury.State != missioncontrol.TreasuryStateActive {
		t.Fatalf("LoadTreasuryRecord(first).State = %q, want %q", firstTreasury.State, missioncontrol.TreasuryStateActive)
	}
	if firstTreasury.PostSuspendResume == nil || firstTreasury.PostSuspendResume.ConsumedTransitionID == "" {
		t.Fatalf("LoadTreasuryRecord(first).PostSuspendResume = %#v, want consumed transition linkage", firstTreasury.PostSuspendResume)
	}
	if firstTreasury.PostActiveSuspend != nil {
		t.Fatalf("LoadTreasuryRecord(first).PostActiveSuspend = %#v, want cleared consumed suspend block", firstTreasury.PostActiveSuspend)
	}

	state.treasuryPostAcquisitionHook = nil
	if err := state.ActivateStep(job, "build"); err != nil {
		t.Fatalf("ActivateStep(second) error = %v, want active no-op", err)
	}

	secondTreasury, err := missioncontrol.LoadTreasuryRecord(root, treasury.TreasuryID)
	if err != nil {
		t.Fatalf("LoadTreasuryRecord(second) error = %v", err)
	}
	if !reflect.DeepEqual(secondTreasury, firstTreasury) {
		t.Fatalf("LoadTreasuryRecord(second) = %#v, want unchanged %#v", secondTreasury, firstTreasury)
	}
}

func TestTaskStateActivateStepActiveTreasuryAllocatePathCallsAllocateProducerOnce(t *testing.T) {
	t.Parallel()

	root, treasury, sourceContainer := writeTaskStateActiveTreasuryAllocateFixtures(t)
	job := testTaskStateJob()
	job.Plan.Steps[0].TreasuryRef = &missioncontrol.TreasuryRef{TreasuryID: treasury.TreasuryID}

	state := NewTaskState()
	state.SetMissionStoreRoot(root)

	allocateCalls := 0
	reinvestCalls := 0
	var gotRoot string
	var gotLease missioncontrol.WriterLockLease
	var gotInput missioncontrol.PostActiveTreasuryAllocateInput
	var gotAt time.Time
	state.treasuryPostActiveAllocateHook = func(root string, lease missioncontrol.WriterLockLease, input missioncontrol.PostActiveTreasuryAllocateInput, now time.Time) error {
		allocateCalls++
		gotRoot = root
		gotLease = lease
		gotInput = input
		gotAt = now
		return nil
	}
	state.treasuryPostActiveReinvestHook = func(root string, lease missioncontrol.WriterLockLease, input missioncontrol.PostActiveTreasuryReinvestInput, now time.Time) error {
		reinvestCalls++
		return nil
	}

	if err := state.ActivateStep(job, "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}

	if allocateCalls != 1 {
		t.Fatalf("treasuryPostActiveAllocateHook calls = %d, want 1", allocateCalls)
	}
	if reinvestCalls != 0 {
		t.Fatalf("treasuryPostActiveReinvestHook calls = %d, want 0 on active allocate path", reinvestCalls)
	}
	if gotRoot != root {
		t.Fatalf("treasuryPostActiveAllocateHook root = %q, want %q", gotRoot, root)
	}
	if gotLease.LeaseHolderID != taskStateTreasuryExecutionLeaseHolderID {
		t.Fatalf("treasuryPostActiveAllocateHook lease = %#v, want %q", gotLease, taskStateTreasuryExecutionLeaseHolderID)
	}
	if !reflect.DeepEqual(gotInput, missioncontrol.PostActiveTreasuryAllocateInput{
		TreasuryRef: missioncontrol.TreasuryRef{TreasuryID: treasury.TreasuryID},
	}) {
		t.Fatalf("treasuryPostActiveAllocateHook input = %#v, want treasury ref %q", gotInput, treasury.TreasuryID)
	}
	if gotAt.IsZero() {
		t.Fatal("treasuryPostActiveAllocateHook now = zero, want activation timestamp")
	}
	if gotInput.TreasuryRef.TreasuryID == sourceContainer.ContainerID {
		t.Fatalf("treasuryPostActiveAllocateHook input treasury ref = %#v, want step treasury ref only", gotInput)
	}
}

func TestTaskStateActivateStepActiveTreasuryAllocatePathInvokesRealProducer(t *testing.T) {
	t.Parallel()

	root, treasury, sourceContainer := writeTaskStateActiveTreasuryAllocateFixtures(t)
	job := testTaskStateJob()
	job.Plan.Steps[0].TreasuryRef = &missioncontrol.TreasuryRef{TreasuryID: treasury.TreasuryID}

	state := NewTaskState()
	state.SetMissionStoreRoot(root)

	if err := state.ActivateStep(job, "build"); err != nil {
		t.Fatalf("ActivateStep(first) error = %v", err)
	}

	firstTreasury, err := missioncontrol.LoadTreasuryRecord(root, treasury.TreasuryID)
	if err != nil {
		t.Fatalf("LoadTreasuryRecord(first) error = %v", err)
	}
	if firstTreasury.PostActiveAllocate == nil || firstTreasury.PostActiveAllocate.ConsumedEntryID == "" {
		t.Fatalf("LoadTreasuryRecord(first).PostActiveAllocate = %#v, want consumed entry linkage", firstTreasury.PostActiveAllocate)
	}
	if firstTreasury.PostActiveAllocate.SourceContainerRef.ObjectID != sourceContainer.ContainerID {
		t.Fatalf("LoadTreasuryRecord(first).PostActiveAllocate.SourceContainerRef = %#v, want %q", firstTreasury.PostActiveAllocate.SourceContainerRef, sourceContainer.ContainerID)
	}
	entries, err := missioncontrol.ListTreasuryLedgerEntries(root, treasury.TreasuryID)
	if err != nil {
		t.Fatalf("ListTreasuryLedgerEntries(first) error = %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("ListTreasuryLedgerEntries(first) len = %d, want 1", len(entries))
	}
	if entries[0].EntryID != firstTreasury.PostActiveAllocate.ConsumedEntryID || entries[0].EntryKind != missioncontrol.TreasuryLedgerEntryKindMovement {
		t.Fatalf("ListTreasuryLedgerEntries(first) = %#v, want one committed post-active allocate movement entry", entries)
	}

	err = state.ActivateStep(job, "build")
	if err == nil {
		t.Fatal("ActivateStep(replay) error = nil, want deterministic consumed post-active allocate rejection")
	}
	if !strings.Contains(err.Error(), `execution context treasury "treasury-wallet" treasury.post_active_allocate is already consumed by entry "`) {
		t.Fatalf("ActivateStep(replay) error = %q, want consumed post-active allocate rejection", err.Error())
	}

	secondTreasury, err := missioncontrol.LoadTreasuryRecord(root, treasury.TreasuryID)
	if err != nil {
		t.Fatalf("LoadTreasuryRecord(second) error = %v", err)
	}
	if !reflect.DeepEqual(secondTreasury, firstTreasury) {
		t.Fatalf("LoadTreasuryRecord(second) = %#v, want unchanged %#v", secondTreasury, firstTreasury)
	}
	secondEntries, err := missioncontrol.ListTreasuryLedgerEntries(root, treasury.TreasuryID)
	if err != nil {
		t.Fatalf("ListTreasuryLedgerEntries(second) error = %v", err)
	}
	if !reflect.DeepEqual(secondEntries, entries) {
		t.Fatalf("ListTreasuryLedgerEntries(second) = %#v, want unchanged %#v", secondEntries, entries)
	}
}

func TestTaskStateActivateStepActiveTreasuryReinvestPathCallsReinvestProducerOnce(t *testing.T) {
	t.Parallel()

	root, treasury, sourceContainer, targetContainer := writeTaskStateActiveTreasuryReinvestFixtures(t)
	job := testTaskStateJob()
	job.Plan.Steps[0].TreasuryRef = &missioncontrol.TreasuryRef{TreasuryID: treasury.TreasuryID}

	state := NewTaskState()
	state.SetMissionStoreRoot(root)

	reinvestCalls := 0
	spendCalls := 0
	var gotRoot string
	var gotLease missioncontrol.WriterLockLease
	var gotInput missioncontrol.PostActiveTreasuryReinvestInput
	var gotAt time.Time
	state.treasuryPostActiveReinvestHook = func(root string, lease missioncontrol.WriterLockLease, input missioncontrol.PostActiveTreasuryReinvestInput, now time.Time) error {
		reinvestCalls++
		gotRoot = root
		gotLease = lease
		gotInput = input
		gotAt = now
		return nil
	}
	state.treasuryPostActiveSpendHook = func(root string, lease missioncontrol.WriterLockLease, input missioncontrol.PostActiveTreasurySpendInput, now time.Time) error {
		spendCalls++
		return nil
	}

	if err := state.ActivateStep(job, "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}

	if reinvestCalls != 1 {
		t.Fatalf("treasuryPostActiveReinvestHook calls = %d, want 1", reinvestCalls)
	}
	if spendCalls != 0 {
		t.Fatalf("treasuryPostActiveSpendHook calls = %d, want 0 on active reinvest path", spendCalls)
	}
	if gotRoot != root {
		t.Fatalf("treasuryPostActiveReinvestHook root = %q, want %q", gotRoot, root)
	}
	if gotLease.LeaseHolderID != taskStateTreasuryExecutionLeaseHolderID {
		t.Fatalf("treasuryPostActiveReinvestHook lease = %#v, want %q", gotLease, taskStateTreasuryExecutionLeaseHolderID)
	}
	if !reflect.DeepEqual(gotInput, missioncontrol.PostActiveTreasuryReinvestInput{
		TreasuryRef: missioncontrol.TreasuryRef{TreasuryID: treasury.TreasuryID},
	}) {
		t.Fatalf("treasuryPostActiveReinvestHook input = %#v, want treasury ref %q", gotInput, treasury.TreasuryID)
	}
	if gotAt.IsZero() {
		t.Fatal("treasuryPostActiveReinvestHook now = zero, want activation timestamp")
	}
	if gotInput.TreasuryRef.TreasuryID == sourceContainer.ContainerID || gotInput.TreasuryRef.TreasuryID == targetContainer.ContainerID {
		t.Fatalf("treasuryPostActiveReinvestHook input treasury ref = %#v, want step treasury ref only", gotInput)
	}
}

func TestTaskStateActivateStepActiveTreasuryReinvestPathInvokesRealProducer(t *testing.T) {
	t.Parallel()

	root, treasury, _, targetContainer := writeTaskStateActiveTreasuryReinvestFixtures(t)
	job := testTaskStateJob()
	job.Plan.Steps[0].TreasuryRef = &missioncontrol.TreasuryRef{TreasuryID: treasury.TreasuryID}

	state := NewTaskState()
	state.SetMissionStoreRoot(root)

	if err := state.ActivateStep(job, "build"); err != nil {
		t.Fatalf("ActivateStep(first) error = %v", err)
	}

	firstTreasury, err := missioncontrol.LoadTreasuryRecord(root, treasury.TreasuryID)
	if err != nil {
		t.Fatalf("LoadTreasuryRecord(first) error = %v", err)
	}
	if firstTreasury.PostActiveReinvest == nil || firstTreasury.PostActiveReinvest.ConsumedEntryID == "" {
		t.Fatalf("LoadTreasuryRecord(first).PostActiveReinvest = %#v, want consumed entry linkage", firstTreasury.PostActiveReinvest)
	}
	if firstTreasury.PostActiveReinvest.TargetContainerRef.ObjectID != targetContainer.ContainerID {
		t.Fatalf("LoadTreasuryRecord(first).PostActiveReinvest.TargetContainerRef = %#v, want %q", firstTreasury.PostActiveReinvest.TargetContainerRef, targetContainer.ContainerID)
	}
	entries, err := missioncontrol.ListTreasuryLedgerEntries(root, treasury.TreasuryID)
	if err != nil {
		t.Fatalf("ListTreasuryLedgerEntries(first) error = %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("ListTreasuryLedgerEntries(first) len = %d, want 2", len(entries))
	}
	var sawDisposition, sawAcquisition bool
	for _, entry := range entries {
		switch entry.EntryKind {
		case missioncontrol.TreasuryLedgerEntryKindDisposition:
			sawDisposition = true
		case missioncontrol.TreasuryLedgerEntryKindAcquisition:
			sawAcquisition = true
			if entry.EntryID != firstTreasury.PostActiveReinvest.ConsumedEntryID {
				t.Fatalf("Acquisition entry id = %q, want consumed_entry_id %q", entry.EntryID, firstTreasury.PostActiveReinvest.ConsumedEntryID)
			}
		}
	}
	if !sawDisposition || !sawAcquisition {
		t.Fatalf("ListTreasuryLedgerEntries(first) = %#v, want paired reinvest entries", entries)
	}

	err = state.ActivateStep(job, "build")
	if err == nil {
		t.Fatal("ActivateStep(replay) error = nil, want deterministic consumed post-active reinvest rejection")
	}
	if !strings.Contains(err.Error(), `execution context treasury "treasury-wallet" treasury.post_active_reinvest is already consumed by entry "`) {
		t.Fatalf("ActivateStep(replay) error = %q, want consumed post-active reinvest rejection", err.Error())
	}

	secondTreasury, err := missioncontrol.LoadTreasuryRecord(root, treasury.TreasuryID)
	if err != nil {
		t.Fatalf("LoadTreasuryRecord(second) error = %v", err)
	}
	if !reflect.DeepEqual(secondTreasury, firstTreasury) {
		t.Fatalf("LoadTreasuryRecord(second) = %#v, want unchanged %#v", secondTreasury, firstTreasury)
	}
	secondEntries, err := missioncontrol.ListTreasuryLedgerEntries(root, treasury.TreasuryID)
	if err != nil {
		t.Fatalf("ListTreasuryLedgerEntries(second) error = %v", err)
	}
	if !reflect.DeepEqual(secondEntries, entries) {
		t.Fatalf("ListTreasuryLedgerEntries(second) = %#v, want unchanged %#v", secondEntries, entries)
	}
}

func TestTaskStateActivateStepActiveTreasurySpendPathCallsSpendProducerOnce(t *testing.T) {
	t.Parallel()

	root, treasury, sourceContainer := writeTaskStateActiveTreasurySpendFixtures(t)
	job := testTaskStateJob()
	job.Plan.Steps[0].TreasuryRef = &missioncontrol.TreasuryRef{TreasuryID: treasury.TreasuryID}

	state := NewTaskState()
	state.SetMissionStoreRoot(root)

	spendCalls := 0
	transferCalls := 0
	var gotRoot string
	var gotLease missioncontrol.WriterLockLease
	var gotInput missioncontrol.PostActiveTreasurySpendInput
	var gotAt time.Time
	state.treasuryPostActiveSpendHook = func(root string, lease missioncontrol.WriterLockLease, input missioncontrol.PostActiveTreasurySpendInput, now time.Time) error {
		spendCalls++
		gotRoot = root
		gotLease = lease
		gotInput = input
		gotAt = now
		return nil
	}
	state.treasuryPostActiveTransferHook = func(root string, lease missioncontrol.WriterLockLease, input missioncontrol.PostActiveTreasuryTransferInput, now time.Time) error {
		transferCalls++
		return nil
	}

	if err := state.ActivateStep(job, "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}

	if spendCalls != 1 {
		t.Fatalf("treasuryPostActiveSpendHook calls = %d, want 1", spendCalls)
	}
	if transferCalls != 0 {
		t.Fatalf("treasuryPostActiveTransferHook calls = %d, want 0 on active spend path", transferCalls)
	}
	if gotRoot != root {
		t.Fatalf("treasuryPostActiveSpendHook root = %q, want %q", gotRoot, root)
	}
	if gotLease.LeaseHolderID != taskStateTreasuryExecutionLeaseHolderID {
		t.Fatalf("treasuryPostActiveSpendHook lease = %#v, want %q", gotLease, taskStateTreasuryExecutionLeaseHolderID)
	}
	if !reflect.DeepEqual(gotInput, missioncontrol.PostActiveTreasurySpendInput{
		TreasuryRef: missioncontrol.TreasuryRef{TreasuryID: treasury.TreasuryID},
	}) {
		t.Fatalf("treasuryPostActiveSpendHook input = %#v, want treasury ref %q", gotInput, treasury.TreasuryID)
	}
	if gotAt.IsZero() {
		t.Fatal("treasuryPostActiveSpendHook now = zero, want activation timestamp")
	}
	if gotInput.TreasuryRef.TreasuryID == sourceContainer.ContainerID {
		t.Fatalf("treasuryPostActiveSpendHook input treasury ref = %#v, want step treasury ref only", gotInput)
	}
}

func TestTaskStateActivateStepActiveTreasurySpendPathInvokesRealProducer(t *testing.T) {
	t.Parallel()

	root, treasury, sourceContainer := writeTaskStateActiveTreasurySpendFixtures(t)
	job := testTaskStateJob()
	job.Plan.Steps[0].TreasuryRef = &missioncontrol.TreasuryRef{TreasuryID: treasury.TreasuryID}

	state := NewTaskState()
	state.SetMissionStoreRoot(root)

	if err := state.ActivateStep(job, "build"); err != nil {
		t.Fatalf("ActivateStep(first) error = %v", err)
	}

	firstTreasury, err := missioncontrol.LoadTreasuryRecord(root, treasury.TreasuryID)
	if err != nil {
		t.Fatalf("LoadTreasuryRecord(first) error = %v", err)
	}
	if firstTreasury.PostActiveSpend == nil || firstTreasury.PostActiveSpend.ConsumedEntryID == "" {
		t.Fatalf("LoadTreasuryRecord(first).PostActiveSpend = %#v, want consumed entry linkage", firstTreasury.PostActiveSpend)
	}
	if firstTreasury.PostActiveSpend.TargetRef != "vendor:domain-renewal" {
		t.Fatalf("LoadTreasuryRecord(first).PostActiveSpend.TargetRef = %q, want %q", firstTreasury.PostActiveSpend.TargetRef, "vendor:domain-renewal")
	}
	entries, err := missioncontrol.ListTreasuryLedgerEntries(root, treasury.TreasuryID)
	if err != nil {
		t.Fatalf("ListTreasuryLedgerEntries(first) error = %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("ListTreasuryLedgerEntries(first) len = %d, want 1", len(entries))
	}
	if entries[0].EntryID != firstTreasury.PostActiveSpend.ConsumedEntryID || entries[0].EntryKind != missioncontrol.TreasuryLedgerEntryKindDisposition {
		t.Fatalf("ListTreasuryLedgerEntries(first) = %#v, want one committed post-active spend disposition entry", entries)
	}

	err = state.ActivateStep(job, "build")
	if err == nil {
		t.Fatal("ActivateStep(replay) error = nil, want deterministic consumed post-active spend rejection")
	}
	if !strings.Contains(err.Error(), `execution context treasury "treasury-wallet" treasury.post_active_spend is already consumed by entry "`) {
		t.Fatalf("ActivateStep(replay) error = %q, want consumed post-active spend rejection", err.Error())
	}

	secondTreasury, err := missioncontrol.LoadTreasuryRecord(root, treasury.TreasuryID)
	if err != nil {
		t.Fatalf("LoadTreasuryRecord(second) error = %v", err)
	}
	if !reflect.DeepEqual(secondTreasury, firstTreasury) {
		t.Fatalf("LoadTreasuryRecord(second) = %#v, want unchanged %#v", secondTreasury, firstTreasury)
	}
	secondEntries, err := missioncontrol.ListTreasuryLedgerEntries(root, treasury.TreasuryID)
	if err != nil {
		t.Fatalf("ListTreasuryLedgerEntries(second) error = %v", err)
	}
	if !reflect.DeepEqual(secondEntries, entries) {
		t.Fatalf("ListTreasuryLedgerEntries(second) = %#v, want unchanged %#v", secondEntries, entries)
	}
	if sourceContainer.ContainerID != "container-wallet" {
		t.Fatalf("source container = %#v, want treasury source container fixture", sourceContainer)
	}
}

func TestTaskStateActivateStepActiveTreasuryTransferPathCallsTransferProducerOnce(t *testing.T) {
	t.Parallel()

	root, treasury, _, targetContainer := writeTaskStateActiveTreasuryTransferFixtures(t)
	job := testTaskStateJob()
	job.Plan.Steps[0].TreasuryRef = &missioncontrol.TreasuryRef{TreasuryID: treasury.TreasuryID}

	state := NewTaskState()
	state.SetMissionStoreRoot(root)

	transferCalls := 0
	saveCalls := 0
	var gotRoot string
	var gotLease missioncontrol.WriterLockLease
	var gotInput missioncontrol.PostActiveTreasuryTransferInput
	var gotAt time.Time
	state.treasuryPostActiveTransferHook = func(root string, lease missioncontrol.WriterLockLease, input missioncontrol.PostActiveTreasuryTransferInput, now time.Time) error {
		transferCalls++
		gotRoot = root
		gotLease = lease
		gotInput = input
		gotAt = now
		return nil
	}
	state.treasuryPostActiveSaveHook = func(root string, lease missioncontrol.WriterLockLease, input missioncontrol.PostActiveTreasurySaveInput, now time.Time) error {
		saveCalls++
		return nil
	}

	if err := state.ActivateStep(job, "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}

	if transferCalls != 1 {
		t.Fatalf("treasuryPostActiveTransferHook calls = %d, want 1", transferCalls)
	}
	if saveCalls != 0 {
		t.Fatalf("treasuryPostActiveSaveHook calls = %d, want 0 on active transfer path", saveCalls)
	}
	if gotRoot != root {
		t.Fatalf("treasuryPostActiveTransferHook root = %q, want %q", gotRoot, root)
	}
	if gotLease.LeaseHolderID != taskStateTreasuryExecutionLeaseHolderID {
		t.Fatalf("treasuryPostActiveTransferHook lease = %#v, want %q", gotLease, taskStateTreasuryExecutionLeaseHolderID)
	}
	if !reflect.DeepEqual(gotInput, missioncontrol.PostActiveTreasuryTransferInput{
		TreasuryRef: missioncontrol.TreasuryRef{TreasuryID: treasury.TreasuryID},
	}) {
		t.Fatalf("treasuryPostActiveTransferHook input = %#v, want treasury ref %q", gotInput, treasury.TreasuryID)
	}
	if gotAt.IsZero() {
		t.Fatal("treasuryPostActiveTransferHook now = zero, want activation timestamp")
	}
	if gotInput.TreasuryRef.TreasuryID == targetContainer.ContainerID {
		t.Fatalf("treasuryPostActiveTransferHook input treasury ref = %#v, want step treasury ref only", gotInput)
	}
}

func TestTaskStateActivateStepActiveTreasuryTransferPathInvokesRealProducer(t *testing.T) {
	t.Parallel()

	root, treasury, _, targetContainer := writeTaskStateActiveTreasuryTransferFixtures(t)
	job := testTaskStateJob()
	job.Plan.Steps[0].TreasuryRef = &missioncontrol.TreasuryRef{TreasuryID: treasury.TreasuryID}

	state := NewTaskState()
	state.SetMissionStoreRoot(root)

	if err := state.ActivateStep(job, "build"); err != nil {
		t.Fatalf("ActivateStep(first) error = %v", err)
	}

	firstTreasury, err := missioncontrol.LoadTreasuryRecord(root, treasury.TreasuryID)
	if err != nil {
		t.Fatalf("LoadTreasuryRecord(first) error = %v", err)
	}
	if firstTreasury.PostActiveTransfer == nil || firstTreasury.PostActiveTransfer.ConsumedEntryID == "" {
		t.Fatalf("LoadTreasuryRecord(first).PostActiveTransfer = %#v, want consumed entry linkage", firstTreasury.PostActiveTransfer)
	}
	if firstTreasury.PostActiveTransfer.TargetContainerRef.ObjectID != targetContainer.ContainerID {
		t.Fatalf("LoadTreasuryRecord(first).PostActiveTransfer.TargetContainerRef = %#v, want %q", firstTreasury.PostActiveTransfer.TargetContainerRef, targetContainer.ContainerID)
	}
	entries, err := missioncontrol.ListTreasuryLedgerEntries(root, treasury.TreasuryID)
	if err != nil {
		t.Fatalf("ListTreasuryLedgerEntries(first) error = %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("ListTreasuryLedgerEntries(first) len = %d, want 1", len(entries))
	}
	if entries[0].EntryID != firstTreasury.PostActiveTransfer.ConsumedEntryID || entries[0].EntryKind != missioncontrol.TreasuryLedgerEntryKindMovement {
		t.Fatalf("ListTreasuryLedgerEntries(first) = %#v, want one committed post-active transfer movement entry", entries)
	}

	err = state.ActivateStep(job, "build")
	if err == nil {
		t.Fatal("ActivateStep(replay) error = nil, want deterministic consumed post-active transfer rejection")
	}
	if !strings.Contains(err.Error(), `execution context treasury "treasury-wallet" treasury.post_active_transfer is already consumed by entry "`) {
		t.Fatalf("ActivateStep(replay) error = %q, want consumed post-active transfer rejection", err.Error())
	}

	secondTreasury, err := missioncontrol.LoadTreasuryRecord(root, treasury.TreasuryID)
	if err != nil {
		t.Fatalf("LoadTreasuryRecord(second) error = %v", err)
	}
	if !reflect.DeepEqual(secondTreasury, firstTreasury) {
		t.Fatalf("LoadTreasuryRecord(second) = %#v, want unchanged %#v", secondTreasury, firstTreasury)
	}
	secondEntries, err := missioncontrol.ListTreasuryLedgerEntries(root, treasury.TreasuryID)
	if err != nil {
		t.Fatalf("ListTreasuryLedgerEntries(second) error = %v", err)
	}
	if !reflect.DeepEqual(secondEntries, entries) {
		t.Fatalf("ListTreasuryLedgerEntries(second) = %#v, want unchanged %#v", secondEntries, entries)
	}
}

func TestTaskStateActivateStepActiveTreasurySavePathCallsSaveProducerOnce(t *testing.T) {
	t.Parallel()

	root, treasury, _, targetContainer := writeTaskStateActiveTreasurySaveFixtures(t)
	job := testTaskStateJob()
	job.Plan.Steps[0].TreasuryRef = &missioncontrol.TreasuryRef{TreasuryID: treasury.TreasuryID}

	state := NewTaskState()
	state.SetMissionStoreRoot(root)

	saveCalls := 0
	acquisitionCalls := 0
	var gotRoot string
	var gotLease missioncontrol.WriterLockLease
	var gotInput missioncontrol.PostActiveTreasurySaveInput
	var gotAt time.Time
	state.treasuryPostActiveSaveHook = func(root string, lease missioncontrol.WriterLockLease, input missioncontrol.PostActiveTreasurySaveInput, now time.Time) error {
		saveCalls++
		gotRoot = root
		gotLease = lease
		gotInput = input
		gotAt = now
		return nil
	}
	state.treasuryPostAcquisitionHook = func(root string, lease missioncontrol.WriterLockLease, input missioncontrol.PostBootstrapTreasuryAcquisitionInput, now time.Time) error {
		acquisitionCalls++
		return nil
	}

	if err := state.ActivateStep(job, "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}

	if saveCalls != 1 {
		t.Fatalf("treasuryPostActiveSaveHook calls = %d, want 1", saveCalls)
	}
	if acquisitionCalls != 0 {
		t.Fatalf("treasuryPostAcquisitionHook calls = %d, want 0 on active save path", acquisitionCalls)
	}
	if gotRoot != root {
		t.Fatalf("treasuryPostActiveSaveHook root = %q, want %q", gotRoot, root)
	}
	if gotLease.LeaseHolderID != taskStateTreasuryExecutionLeaseHolderID {
		t.Fatalf("treasuryPostActiveSaveHook lease = %#v, want %q", gotLease, taskStateTreasuryExecutionLeaseHolderID)
	}
	if !reflect.DeepEqual(gotInput, missioncontrol.PostActiveTreasurySaveInput{
		TreasuryRef: missioncontrol.TreasuryRef{TreasuryID: treasury.TreasuryID},
	}) {
		t.Fatalf("treasuryPostActiveSaveHook input = %#v, want treasury ref %q", gotInput, treasury.TreasuryID)
	}
	if gotAt.IsZero() {
		t.Fatal("treasuryPostActiveSaveHook now = zero, want activation timestamp")
	}
	if gotInput.TreasuryRef.TreasuryID == targetContainer.ContainerID {
		t.Fatalf("treasuryPostActiveSaveHook input treasury ref = %#v, want step treasury ref only", gotInput)
	}
}

func TestTaskStateActivateStepActiveTreasurySavePathInvokesRealProducer(t *testing.T) {
	t.Parallel()

	root, treasury, _, targetContainer := writeTaskStateActiveTreasurySaveFixtures(t)
	job := testTaskStateJob()
	job.Plan.Steps[0].TreasuryRef = &missioncontrol.TreasuryRef{TreasuryID: treasury.TreasuryID}

	state := NewTaskState()
	state.SetMissionStoreRoot(root)

	if err := state.ActivateStep(job, "build"); err != nil {
		t.Fatalf("ActivateStep(first) error = %v", err)
	}

	firstTreasury, err := missioncontrol.LoadTreasuryRecord(root, treasury.TreasuryID)
	if err != nil {
		t.Fatalf("LoadTreasuryRecord(first) error = %v", err)
	}
	if firstTreasury.PostActiveSave == nil || firstTreasury.PostActiveSave.ConsumedEntryID == "" {
		t.Fatalf("LoadTreasuryRecord(first).PostActiveSave = %#v, want consumed entry linkage", firstTreasury.PostActiveSave)
	}
	if firstTreasury.PostActiveSave.TargetContainerID != targetContainer.ContainerID {
		t.Fatalf("LoadTreasuryRecord(first).PostActiveSave.TargetContainerID = %q, want %q", firstTreasury.PostActiveSave.TargetContainerID, targetContainer.ContainerID)
	}
	entries, err := missioncontrol.ListTreasuryLedgerEntries(root, treasury.TreasuryID)
	if err != nil {
		t.Fatalf("ListTreasuryLedgerEntries(first) error = %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("ListTreasuryLedgerEntries(first) len = %d, want 1", len(entries))
	}
	if entries[0].EntryID != firstTreasury.PostActiveSave.ConsumedEntryID || entries[0].EntryKind != missioncontrol.TreasuryLedgerEntryKindMovement {
		t.Fatalf("ListTreasuryLedgerEntries(first) = %#v, want one committed post-active save movement entry", entries)
	}

	err = state.ActivateStep(job, "build")
	if err == nil {
		t.Fatal("ActivateStep(replay) error = nil, want deterministic consumed post-active save rejection")
	}
	if !strings.Contains(err.Error(), `execution context treasury "treasury-wallet" treasury.post_active_save is already consumed by entry "`) {
		t.Fatalf("ActivateStep(replay) error = %q, want consumed post-active save rejection", err.Error())
	}

	secondTreasury, err := missioncontrol.LoadTreasuryRecord(root, treasury.TreasuryID)
	if err != nil {
		t.Fatalf("LoadTreasuryRecord(second) error = %v", err)
	}
	if !reflect.DeepEqual(secondTreasury, firstTreasury) {
		t.Fatalf("LoadTreasuryRecord(second) = %#v, want unchanged %#v", secondTreasury, firstTreasury)
	}
	secondEntries, err := missioncontrol.ListTreasuryLedgerEntries(root, treasury.TreasuryID)
	if err != nil {
		t.Fatalf("ListTreasuryLedgerEntries(second) error = %v", err)
	}
	if !reflect.DeepEqual(secondEntries, entries) {
		t.Fatalf("ListTreasuryLedgerEntries(second) = %#v, want unchanged %#v", secondEntries, entries)
	}
}

func TestTaskStateActivateStepTreasuryPolicyDisallowedFailsClosed(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 11, 16, 0, 0, 0, time.UTC)
	target := missioncontrol.AutonomyEligibilityTargetRef{
		Kind:       missioncontrol.EligibilityTargetKindTreasuryContainerClass,
		RegistryID: "container-class-human-wallet",
	}
	writeTaskStateAutonomyEligibilityFixture(t, root, target, missioncontrol.PlatformRecord{
		PlatformID:       target.RegistryID,
		PlatformName:     "container-class-human-wallet",
		TargetClass:      target.Kind,
		EligibilityLabel: missioncontrol.EligibilityLabelIneligible,
		LastCheckID:      "check-container-class-human-wallet",
		Notes:            []string{"registry note"},
		UpdatedAt:        now,
	}, missioncontrol.EligibilityCheckRecord{
		CheckID:                     "check-container-class-human-wallet",
		TargetKind:                  target.Kind,
		TargetName:                  "container-class-human-wallet",
		CanCreateWithoutOwner:       false,
		CanOnboardWithoutOwner:      false,
		CanControlAsAgent:           false,
		CanRecoverAsAgent:           false,
		RequiresHumanOnlyStep:       true,
		RequiresOwnerOnlySecretOrID: true,
		RulesAsObservedOK:           false,
		Label:                       missioncontrol.EligibilityLabelIneligible,
		Reasons:                     []string{string(missioncontrol.AutonomyEligibilityReasonOwnerIdentityRequired)},
		CheckedAt:                   now,
	})

	container := missioncontrol.FrankContainerRecord{
		RecordVersion:        missioncontrol.StoreRecordVersion,
		ContainerID:          "container-human-wallet",
		ContainerKind:        "wallet",
		Label:                "Human Wallet",
		ContainerClassID:     target.RegistryID,
		State:                "candidate",
		EligibilityTargetRef: target,
		CreatedAt:            now.Add(time.Minute).UTC(),
		UpdatedAt:            now.Add(2 * time.Minute).UTC(),
	}
	if err := missioncontrol.StoreFrankContainerRecord(root, container); err != nil {
		t.Fatalf("StoreFrankContainerRecord() error = %v", err)
	}

	treasury := missioncontrol.TreasuryRecord{
		RecordVersion:  missioncontrol.StoreRecordVersion,
		TreasuryID:     "treasury-policy-disallowed",
		DisplayName:    "Frank Treasury",
		State:          missioncontrol.TreasuryStateFunded,
		ZeroSeedPolicy: missioncontrol.TreasuryZeroSeedPolicyOwnerSeedForbidden,
		ContainerRefs: []missioncontrol.FrankRegistryObjectRef{{
			Kind:     missioncontrol.FrankRegistryObjectKindContainer,
			ObjectID: container.ContainerID,
		}},
		CreatedAt: now.UTC(),
		UpdatedAt: now.Add(3 * time.Minute).UTC(),
	}
	if err := missioncontrol.StoreTreasuryRecord(root, treasury); err != nil {
		t.Fatalf("StoreTreasuryRecord() error = %v", err)
	}

	state := NewTaskState()
	state.SetMissionStoreRoot(root)
	job := testTaskStateJob()
	job.Plan.Steps[0].TreasuryRef = &missioncontrol.TreasuryRef{TreasuryID: treasury.TreasuryID}

	_, wantErr := missioncontrol.RequireAutonomyEligibleTarget(root, target)
	if wantErr == nil {
		t.Fatal("RequireAutonomyEligibleTarget() error = nil, want policy rejection")
	}

	err := state.ActivateStep(job, "build")
	if err == nil {
		t.Fatal("ActivateStep() error = nil, want treasury policy rejection")
	}
	if err.Error() != wantErr.Error() {
		t.Fatalf("ActivateStep() error = %q, want %q", err.Error(), wantErr.Error())
	}
	if _, ok := state.ExecutionContext(); ok {
		t.Fatal("ExecutionContext() ok = true, want no active context after policy rejection")
	}
	if _, ok := state.MissionRuntimeState(); ok {
		t.Fatal("MissionRuntimeState() ok = true, want no stored runtime after policy rejection")
	}

	loadedTreasury, loadErr := missioncontrol.LoadTreasuryRecord(root, treasury.TreasuryID)
	if loadErr != nil {
		t.Fatalf("LoadTreasuryRecord() error = %v", loadErr)
	}
	if loadedTreasury.State != missioncontrol.TreasuryStateFunded {
		t.Fatalf("LoadTreasuryRecord().State = %q, want %q", loadedTreasury.State, missioncontrol.TreasuryStateFunded)
	}
}

func TestTaskStateActivateStepTreasuryReplayStaysDeterministic(t *testing.T) {
	t.Parallel()

	root, treasury, _ := writeTaskStateTreasuryFixtures(t)
	job := testTaskStateJob()
	job.Plan.Steps[0].TreasuryRef = &missioncontrol.TreasuryRef{TreasuryID: treasury.TreasuryID}

	state := NewTaskState()
	state.SetMissionStoreRoot(root)
	if err := state.ActivateStep(job, "build"); err != nil {
		t.Fatalf("ActivateStep(first) error = %v", err)
	}

	firstRuntime, ok := state.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want stored runtime after first activation")
	}
	firstTreasury, err := missioncontrol.LoadTreasuryRecord(root, treasury.TreasuryID)
	if err != nil {
		t.Fatalf("LoadTreasuryRecord(first) error = %v", err)
	}

	err = state.ActivateStep(job, "build")
	if err == nil {
		t.Fatal("ActivateStep(replay) error = nil, want deterministic treasury activation rejection")
	}
	if !strings.Contains(err.Error(), `execution context treasury "treasury-wallet" requires committed treasury.post_bootstrap_acquisition for additional acquisition`) {
		t.Fatalf("ActivateStep(replay) error = %q, want missing post-bootstrap acquisition rejection", err.Error())
	}

	secondRuntime, ok := state.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want unchanged runtime after replay rejection")
	}
	if !reflect.DeepEqual(secondRuntime, firstRuntime) {
		t.Fatalf("MissionRuntimeState() = %#v, want unchanged %#v", secondRuntime, firstRuntime)
	}
	secondTreasury, err := missioncontrol.LoadTreasuryRecord(root, treasury.TreasuryID)
	if err != nil {
		t.Fatalf("LoadTreasuryRecord(second) error = %v", err)
	}
	if !reflect.DeepEqual(secondTreasury, firstTreasury) {
		t.Fatalf("LoadTreasuryRecord() = %#v, want unchanged %#v", secondTreasury, firstTreasury)
	}
}

func TestTaskStateActivateStepCampaignZeroRefPathUnchanged(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	calls := 0
	state.campaignReadinessGuardHook = func(missioncontrol.ExecutionContext) error {
		calls++
		return nil
	}

	if err := state.ActivateStep(testTaskStateJob(), "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}
	if calls != 0 {
		t.Fatalf("campaignReadinessGuardHook calls = %d, want 0 for zero-campaign-ref path", calls)
	}
}

func TestTaskStateActivateStepZohoMailboxBootstrapZeroRefPathUnchanged(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	calls := 0
	state.zohoMailboxBootstrapHook = func(string, missioncontrol.ResolvedExecutionContextFrankZohoMailboxBootstrapPair, time.Time) error {
		calls++
		return nil
	}

	if err := state.ActivateStep(testTaskStateJob(), "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}
	if calls != 0 {
		t.Fatalf("zohoMailboxBootstrapHook calls = %d, want 0 for zero-frank-object-ref path", calls)
	}
}

func TestTaskStateActivateStepZohoMailboxBootstrapPathCallsProducerOnce(t *testing.T) {
	t.Parallel()

	root, identity, account := writeTaskStateZohoMailboxBootstrapFixtures(t)
	job := testTaskStateJob()
	job.Plan.Steps[0].GovernedExternalTargets = []missioncontrol.AutonomyEligibilityTargetRef{
		{Kind: missioncontrol.EligibilityTargetKindProvider, RegistryID: "provider-mail"},
		{Kind: missioncontrol.EligibilityTargetKindAccountClass, RegistryID: "account-class-mailbox"},
	}
	job.Plan.Steps[0].FrankObjectRefs = []missioncontrol.FrankRegistryObjectRef{
		{Kind: missioncontrol.FrankRegistryObjectKindIdentity, ObjectID: identity.IdentityID},
		{Kind: missioncontrol.FrankRegistryObjectKindAccount, ObjectID: account.AccountID},
	}

	state := NewTaskState()
	state.SetMissionStoreRoot(root)

	calls := 0
	var gotRoot string
	var gotPair missioncontrol.ResolvedExecutionContextFrankZohoMailboxBootstrapPair
	var gotAt time.Time
	state.zohoMailboxBootstrapHook = func(root string, pair missioncontrol.ResolvedExecutionContextFrankZohoMailboxBootstrapPair, now time.Time) error {
		calls++
		gotRoot = root
		gotPair = pair
		gotAt = now
		return nil
	}

	if err := state.ActivateStep(job, "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}

	if calls != 1 {
		t.Fatalf("zohoMailboxBootstrapHook calls = %d, want 1", calls)
	}
	if gotRoot != root {
		t.Fatalf("zohoMailboxBootstrapHook root = %q, want %q", gotRoot, root)
	}
	wantPair := missioncontrol.ResolvedExecutionContextFrankZohoMailboxBootstrapPair{
		Identity: identity,
		Account:  account,
	}
	if !reflect.DeepEqual(gotPair, wantPair) {
		t.Fatalf("zohoMailboxBootstrapHook pair = %#v, want %#v", gotPair, wantPair)
	}
	if gotAt.IsZero() {
		t.Fatal("zohoMailboxBootstrapHook now = zero, want activation timestamp")
	}
}

func TestTaskStateActivateStepZohoMailboxBootstrapPathInvokesRealProducer(t *testing.T) {
	t.Parallel()

	root, identity, account := writeTaskStateZohoMailboxBootstrapFixtures(t)
	job := testTaskStateJob()
	job.Plan.Steps[0].GovernedExternalTargets = []missioncontrol.AutonomyEligibilityTargetRef{
		{Kind: missioncontrol.EligibilityTargetKindProvider, RegistryID: "provider-mail"},
		{Kind: missioncontrol.EligibilityTargetKindAccountClass, RegistryID: "account-class-mailbox"},
	}
	job.Plan.Steps[0].FrankObjectRefs = []missioncontrol.FrankRegistryObjectRef{
		{Kind: missioncontrol.FrankRegistryObjectKindIdentity, ObjectID: identity.IdentityID},
		{Kind: missioncontrol.FrankRegistryObjectKindAccount, ObjectID: account.AccountID},
	}

	state := NewTaskState()
	state.SetMissionStoreRoot(root)

	calls := 0
	state.zohoMailboxBootstrapHook = func(root string, pair missioncontrol.ResolvedExecutionContextFrankZohoMailboxBootstrapPair, now time.Time) error {
		calls++
		return missioncontrol.ProduceFrankZohoMailboxBootstrap(root, pair, now)
	}

	if err := state.ActivateStep(job, "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}
	if calls != 1 {
		t.Fatalf("zohoMailboxBootstrapHook calls = %d, want 1 real-producer invocation", calls)
	}

	storedAccount, err := missioncontrol.LoadFrankAccountRecord(root, account.AccountID)
	if err != nil {
		t.Fatalf("LoadFrankAccountRecord() error = %v", err)
	}
	if !reflect.DeepEqual(storedAccount, account) {
		t.Fatalf("LoadFrankAccountRecord() = %#v, want replay-safe unchanged committed account %#v", storedAccount, account)
	}
}

func TestTaskStateActivateStepZohoMailboxBootstrapInvalidPairFailsClosed(t *testing.T) {
	t.Parallel()

	root, identity, _ := writeTaskStateZohoMailboxBootstrapFixtures(t)
	job := testTaskStateJob()
	job.Plan.Steps[0].GovernedExternalTargets = []missioncontrol.AutonomyEligibilityTargetRef{
		{Kind: missioncontrol.EligibilityTargetKindProvider, RegistryID: "provider-mail"},
		{Kind: missioncontrol.EligibilityTargetKindAccountClass, RegistryID: "account-class-mailbox"},
	}
	job.Plan.Steps[0].FrankObjectRefs = []missioncontrol.FrankRegistryObjectRef{
		{Kind: missioncontrol.FrankRegistryObjectKindIdentity, ObjectID: identity.IdentityID},
		{Kind: missioncontrol.FrankRegistryObjectKindAccount, ObjectID: "account-missing"},
	}

	state := NewTaskState()
	state.SetMissionStoreRoot(root)

	calls := 0
	state.zohoMailboxBootstrapHook = func(string, missioncontrol.ResolvedExecutionContextFrankZohoMailboxBootstrapPair, time.Time) error {
		calls++
		return nil
	}

	err := state.ActivateStep(job, "build")
	if err == nil {
		t.Fatal("ActivateStep() error = nil, want invalid zoho mailbox pair rejection")
	}
	if !strings.Contains(err.Error(), `resolve Frank object ref kind "account" object_id "account-missing": mission store Frank account record not found`) {
		t.Fatalf("ActivateStep() error = %q, want invalid zoho mailbox pair rejection", err.Error())
	}
	if calls != 0 {
		t.Fatalf("zohoMailboxBootstrapHook calls = %d, want 0 when pair resolution fails", calls)
	}
	if _, ok := state.ExecutionContext(); ok {
		t.Fatal("ExecutionContext() ok = true, want no active context after zoho mailbox pair rejection")
	}
	if _, ok := state.MissionRuntimeState(); ok {
		t.Fatal("MissionRuntimeState() ok = true, want no stored runtime after zoho mailbox pair rejection")
	}
}

func TestTaskStateActivateStepCampaignPathCallsReadinessGuardOnce(t *testing.T) {
	t.Parallel()

	root, _, container := writeTaskStateTreasuryFixtures(t)
	campaign := mustStoreTaskStateCampaignFixture(t, root, container)
	job := testTaskStateJob()
	job.Plan.Steps[0].CampaignRef = &missioncontrol.CampaignRef{CampaignID: campaign.CampaignID}

	state := NewTaskState()
	state.SetMissionStoreRoot(root)

	calls := 0
	var gotEC missioncontrol.ExecutionContext
	state.campaignReadinessGuardHook = func(ec missioncontrol.ExecutionContext) error {
		calls++
		gotEC = missioncontrol.CloneExecutionContext(ec)
		return nil
	}

	if err := state.ActivateStep(job, "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}
	if calls != 1 {
		t.Fatalf("campaignReadinessGuardHook calls = %d, want 1", calls)
	}
	if gotEC.Step == nil || gotEC.Step.CampaignRef == nil {
		t.Fatalf("campaignReadinessGuardHook execution context = %#v, want campaign-aware step", gotEC)
	}
	if gotEC.Step.CampaignRef.CampaignID != campaign.CampaignID {
		t.Fatalf("campaignReadinessGuardHook campaign_id = %q, want %q", gotEC.Step.CampaignRef.CampaignID, campaign.CampaignID)
	}
	if gotEC.MissionStoreRoot != root {
		t.Fatalf("campaignReadinessGuardHook mission_store_root = %q, want %q", gotEC.MissionStoreRoot, root)
	}
}

func TestTaskStateActivateStepCampaignReadinessDisallowedFailsClosed(t *testing.T) {
	t.Parallel()

	root, _, container := writeTaskStateTreasuryFixtures(t)
	campaign := mustStoreTaskStateCampaignFixture(t, root, container)
	campaign.State = missioncontrol.CampaignStateDraft
	if err := missioncontrol.StoreCampaignRecord(root, campaign); err != nil {
		t.Fatalf("StoreCampaignRecord() error = %v", err)
	}

	job := testTaskStateJob()
	job.Plan.Steps[0].CampaignRef = &missioncontrol.CampaignRef{CampaignID: campaign.CampaignID}

	state := NewTaskState()
	state.SetMissionStoreRoot(root)

	err := state.ActivateStep(job, "build")
	if err == nil {
		t.Fatal("ActivateStep() error = nil, want campaign readiness rejection")
	}
	if !strings.Contains(err.Error(), `campaign readiness requires state "active"; got "draft"`) {
		t.Fatalf("ActivateStep() error = %q, want campaign readiness rejection", err.Error())
	}
	if _, ok := state.ExecutionContext(); ok {
		t.Fatal("ExecutionContext() ok = true, want no active context after campaign rejection")
	}
	if _, ok := state.MissionRuntimeState(); ok {
		t.Fatal("MissionRuntimeState() ok = true, want no stored runtime after campaign rejection")
	}
}

func TestTaskStateApplyStepOutputPausesCompletedOneShotCodeStep(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	if err := state.ActivateStep(testTaskStateJob(), "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}

	if err := state.ApplyStepOutput("Implemented the change.", []missioncontrol.RuntimeToolCallEvidence{
		{ToolName: "filesystem", Arguments: map[string]interface{}{"action": "write", "path": "result.txt"}},
		{ToolName: "filesystem", Arguments: map[string]interface{}{"action": "stat", "path": "result.txt"}},
	}); err != nil {
		t.Fatalf("ApplyStepOutput() error = %v", err)
	}

	if _, ok := state.ExecutionContext(); ok {
		t.Fatal("ExecutionContext() ok = true, want false after completed step pause")
	}

	runtime, ok := state.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if runtime.State != missioncontrol.JobStatePaused {
		t.Fatalf("MissionRuntimeState().State = %q, want %q", runtime.State, missioncontrol.JobStatePaused)
	}
	if len(runtime.CompletedSteps) != 1 || runtime.CompletedSteps[0].StepID != "build" {
		t.Fatalf("MissionRuntimeState().CompletedSteps = %#v, want build completion", runtime.CompletedSteps)
	}
}

func TestTaskStateApplyStepOutputPausesForUnattendedWallClockBudget(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	job := testTaskStateJob()
	now := time.Now().UTC()
	if err := state.ActivateStep(job, "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}

	ec, ok := state.ExecutionContext()
	if !ok || ec.Runtime == nil {
		t.Fatalf("ExecutionContext() = (%#v, %t), want active runtime", ec, ok)
	}
	ec.Runtime.CreatedAt = now.Add(-5 * time.Hour)
	ec.Runtime.UpdatedAt = now.Add(-2 * time.Minute)
	ec.Runtime.StartedAt = now.Add(-5 * time.Hour)
	ec.Runtime.ActiveStepAt = now.Add(-2 * time.Minute)
	state.SetExecutionContext(ec)

	if err := state.ApplyStepOutput("Implemented the change.", []missioncontrol.RuntimeToolCallEvidence{
		{ToolName: "filesystem", Arguments: map[string]interface{}{"action": "write", "path": "result.txt"}},
	}); err != nil {
		t.Fatalf("ApplyStepOutput() error = %v", err)
	}

	ec, ok = state.ExecutionContext()
	if !ok {
		t.Fatal("ExecutionContext() ok = false, want paused execution context")
	}
	if ec.Runtime == nil || ec.Runtime.State != missioncontrol.JobStatePaused {
		t.Fatalf("ExecutionContext().Runtime = %#v, want paused runtime", ec.Runtime)
	}

	runtime, ok := state.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if runtime.PausedReason != missioncontrol.RuntimePauseReasonBudgetExhausted {
		t.Fatalf("MissionRuntimeState().PausedReason = %q, want %q", runtime.PausedReason, missioncontrol.RuntimePauseReasonBudgetExhausted)
	}
	if runtime.BudgetBlocker == nil {
		t.Fatal("MissionRuntimeState().BudgetBlocker = nil, want blocker")
	}
	if runtime.BudgetBlocker.Ceiling != "unattended_wall_clock" {
		t.Fatalf("MissionRuntimeState().BudgetBlocker.Ceiling = %q, want %q", runtime.BudgetBlocker.Ceiling, "unattended_wall_clock")
	}
	if len(runtime.CompletedSteps) != 0 {
		t.Fatalf("MissionRuntimeState().CompletedSteps = %#v, want empty after budget pause", runtime.CompletedSteps)
	}
	if len(runtime.AuditHistory) != 1 || runtime.AuditHistory[0].ToolName != "budget_exhausted" {
		t.Fatalf("MissionRuntimeState().AuditHistory = %#v, want one budget_exhausted event", runtime.AuditHistory)
	}
}

func TestTaskStateEnforceUnattendedWallClockBudgetPausesRunningExecution(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	job := testTaskStateJob()
	now := time.Now().UTC()
	if err := state.ActivateStep(job, "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}

	ec, ok := state.ExecutionContext()
	if !ok || ec.Runtime == nil {
		t.Fatalf("ExecutionContext() = (%#v, %t), want active runtime", ec, ok)
	}
	ec.Runtime.CreatedAt = now.Add(-5 * time.Hour)
	ec.Runtime.UpdatedAt = now.Add(-1 * time.Minute)
	ec.Runtime.StartedAt = now.Add(-5 * time.Hour)
	ec.Runtime.ActiveStepAt = now.Add(-1 * time.Minute)
	state.SetExecutionContext(ec)

	exhausted, err := state.EnforceUnattendedWallClockBudget()
	if err != nil {
		t.Fatalf("EnforceUnattendedWallClockBudget() error = %v", err)
	}
	if !exhausted {
		t.Fatal("EnforceUnattendedWallClockBudget() exhausted = false, want true")
	}

	runtime, ok := state.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if runtime.State != missioncontrol.JobStatePaused {
		t.Fatalf("MissionRuntimeState().State = %q, want %q", runtime.State, missioncontrol.JobStatePaused)
	}
	if runtime.BudgetBlocker == nil || runtime.BudgetBlocker.Ceiling != "unattended_wall_clock" {
		t.Fatalf("MissionRuntimeState().BudgetBlocker = %#v, want unattended wall-clock blocker", runtime.BudgetBlocker)
	}
}

func TestTaskStateRecordFailedToolActionPausesAtBudget(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	job := testTaskStateJob()
	if err := state.ActivateStep(job, "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}

	var exhausted bool
	var err error
	for i := 0; i < 5; i++ {
		exhausted, err = state.RecordFailedToolAction("message", "message tool: 'content' argument required")
		if err != nil {
			t.Fatalf("RecordFailedToolAction() step %d error = %v", i, err)
		}
	}

	if !exhausted {
		t.Fatal("RecordFailedToolAction() exhausted = false, want true on threshold")
	}

	runtime, ok := state.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if runtime.State != missioncontrol.JobStatePaused {
		t.Fatalf("MissionRuntimeState().State = %q, want %q", runtime.State, missioncontrol.JobStatePaused)
	}
	if runtime.BudgetBlocker == nil || runtime.BudgetBlocker.Ceiling != "failed_actions" {
		t.Fatalf("MissionRuntimeState().BudgetBlocker = %#v, want failed_actions blocker", runtime.BudgetBlocker)
	}
	if len(runtime.AuditHistory) != 6 {
		t.Fatalf("MissionRuntimeState().AuditHistory count = %d, want 6", len(runtime.AuditHistory))
	}
}

func TestTaskStateRecordOwnerFacingMessagePausesAtBudget(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	job := testTaskStateJob()
	if err := state.ActivateStep(job, "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}

	var exhausted bool
	var err error
	for i := 0; i < 20; i++ {
		exhausted, err = state.RecordOwnerFacingMessage()
		if err != nil {
			t.Fatalf("RecordOwnerFacingMessage() step %d error = %v", i, err)
		}
	}

	if !exhausted {
		t.Fatal("RecordOwnerFacingMessage() exhausted = false, want true on threshold")
	}

	runtime, ok := state.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if runtime.State != missioncontrol.JobStatePaused {
		t.Fatalf("MissionRuntimeState().State = %q, want %q", runtime.State, missioncontrol.JobStatePaused)
	}
	if runtime.BudgetBlocker == nil || runtime.BudgetBlocker.Ceiling != "owner_messages" {
		t.Fatalf("MissionRuntimeState().BudgetBlocker = %#v, want owner_messages blocker", runtime.BudgetBlocker)
	}
	if len(runtime.AuditHistory) != 21 {
		t.Fatalf("MissionRuntimeState().AuditHistory count = %d, want 21", len(runtime.AuditHistory))
	}
}

func TestTaskStateApplyStepOutputPausesAtOwnerMessageBudget(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	job := testTaskStateJob()
	if err := state.ActivateStep(job, "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}

	for i := 0; i < 19; i++ {
		exhausted, err := state.RecordOwnerFacingMessage()
		if err != nil {
			t.Fatalf("RecordOwnerFacingMessage() step %d error = %v", i, err)
		}
		if exhausted {
			t.Fatalf("RecordOwnerFacingMessage() step %d exhausted = true, want false before final output", i)
		}
	}

	if err := state.ApplyStepOutput("Implemented the change.", nil); err != nil {
		t.Fatalf("ApplyStepOutput() error = %v", err)
	}

	runtime, ok := state.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if runtime.State != missioncontrol.JobStatePaused {
		t.Fatalf("MissionRuntimeState().State = %q, want %q", runtime.State, missioncontrol.JobStatePaused)
	}
	if runtime.BudgetBlocker == nil || runtime.BudgetBlocker.Ceiling != "owner_messages" {
		t.Fatalf("MissionRuntimeState().BudgetBlocker = %#v, want owner_messages blocker", runtime.BudgetBlocker)
	}
	if len(runtime.CompletedSteps) != 0 {
		t.Fatalf("MissionRuntimeState().CompletedSteps = %#v, want empty after budget pause", runtime.CompletedSteps)
	}
	if len(runtime.AuditHistory) != 21 {
		t.Fatalf("MissionRuntimeState().AuditHistory count = %d, want 21", len(runtime.AuditHistory))
	}
	if runtime.AuditHistory[19].ToolName != "step_output" {
		t.Fatalf("MissionRuntimeState().step output audit tool = %q, want %q", runtime.AuditHistory[19].ToolName, "step_output")
	}
	if runtime.AuditHistory[19].ActionClass != missioncontrol.AuditActionClassRuntime {
		t.Fatalf("MissionRuntimeState().step output audit class = %q, want %q", runtime.AuditHistory[19].ActionClass, missioncontrol.AuditActionClassRuntime)
	}
	if runtime.AuditHistory[20].ToolName != "budget_exhausted" {
		t.Fatalf("MissionRuntimeState().last audit tool = %q, want %q", runtime.AuditHistory[20].ToolName, "budget_exhausted")
	}
}

func TestTaskStateRecordOwnerFacingSetStepAckPausesAtOwnerMessageBudget(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	job := testTaskStateJob()
	if err := state.ActivateStep(job, "build"); err != nil {
		t.Fatalf("ActivateStep(build) error = %v", err)
	}

	for i := 0; i < 19; i++ {
		exhausted, err := state.RecordOwnerFacingMessage()
		if err != nil {
			t.Fatalf("RecordOwnerFacingMessage() step %d error = %v", i, err)
		}
		if exhausted {
			t.Fatalf("RecordOwnerFacingMessage() step %d exhausted = true, want false before set-step acknowledgement", i)
		}
	}

	if err := state.ActivateStep(job, "final"); err != nil {
		t.Fatalf("ActivateStep(final) error = %v", err)
	}

	exhausted, err := state.RecordOwnerFacingSetStepAck()
	if err != nil {
		t.Fatalf("RecordOwnerFacingSetStepAck() error = %v", err)
	}
	if !exhausted {
		t.Fatal("RecordOwnerFacingSetStepAck() exhausted = false, want true at threshold")
	}

	runtime, ok := state.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if runtime.State != missioncontrol.JobStatePaused {
		t.Fatalf("MissionRuntimeState().State = %q, want %q", runtime.State, missioncontrol.JobStatePaused)
	}
	if runtime.ActiveStepID != "final" {
		t.Fatalf("MissionRuntimeState().ActiveStepID = %q, want %q", runtime.ActiveStepID, "final")
	}
	if runtime.BudgetBlocker == nil || runtime.BudgetBlocker.Ceiling != "owner_messages" {
		t.Fatalf("MissionRuntimeState().BudgetBlocker = %#v, want owner_messages blocker", runtime.BudgetBlocker)
	}
	if got := runtime.AuditHistory[len(runtime.AuditHistory)-2].ToolName; got != "set_step_ack" {
		t.Fatalf("MissionRuntimeState().penultimate audit tool = %q, want %q", got, "set_step_ack")
	}
	if got := runtime.AuditHistory[len(runtime.AuditHistory)-1].ToolName; got != "budget_exhausted" {
		t.Fatalf("MissionRuntimeState().last audit tool = %q, want %q", got, "budget_exhausted")
	}
}

func TestTaskStateRecordOwnerFacingResumeAckPausesAtOwnerMessageBudget(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	job := testTaskStateJob()
	if err := state.ActivateStep(job, "build"); err != nil {
		t.Fatalf("ActivateStep(build) error = %v", err)
	}

	for i := 0; i < 19; i++ {
		exhausted, err := state.RecordOwnerFacingMessage()
		if err != nil {
			t.Fatalf("RecordOwnerFacingMessage() step %d error = %v", i, err)
		}
		if exhausted {
			t.Fatalf("RecordOwnerFacingMessage() step %d exhausted = true, want false before resume acknowledgement", i)
		}
	}

	if err := state.PauseRuntime(job.ID); err != nil {
		t.Fatalf("PauseRuntime() error = %v", err)
	}
	if err := state.ResumeRuntimeControl(job.ID); err != nil {
		t.Fatalf("ResumeRuntimeControl() error = %v", err)
	}

	exhausted, err := state.RecordOwnerFacingResumeAck()
	if err != nil {
		t.Fatalf("RecordOwnerFacingResumeAck() error = %v", err)
	}
	if !exhausted {
		t.Fatal("RecordOwnerFacingResumeAck() exhausted = false, want true at threshold")
	}

	runtime, ok := state.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if runtime.State != missioncontrol.JobStatePaused {
		t.Fatalf("MissionRuntimeState().State = %q, want %q", runtime.State, missioncontrol.JobStatePaused)
	}
	if runtime.ActiveStepID != "build" {
		t.Fatalf("MissionRuntimeState().ActiveStepID = %q, want %q", runtime.ActiveStepID, "build")
	}
	if runtime.BudgetBlocker == nil || runtime.BudgetBlocker.Ceiling != "owner_messages" {
		t.Fatalf("MissionRuntimeState().BudgetBlocker = %#v, want owner_messages blocker", runtime.BudgetBlocker)
	}
	if got := runtime.AuditHistory[len(runtime.AuditHistory)-2].ToolName; got != "resume_ack" {
		t.Fatalf("MissionRuntimeState().penultimate audit tool = %q, want %q", got, "resume_ack")
	}
	if got := runtime.AuditHistory[len(runtime.AuditHistory)-1].ToolName; got != "budget_exhausted" {
		t.Fatalf("MissionRuntimeState().last audit tool = %q, want %q", got, "budget_exhausted")
	}
}

func TestTaskStateRecordOwnerFacingRevokeApprovalAckPausesAtOwnerMessageBudget(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	job := testReusableApprovalJob(missioncontrol.ApprovalScopeOneJob)
	now := time.Now().UTC()
	runtimeState := missioncontrol.JobRuntimeState{
		JobID:        job.ID,
		State:        missioncontrol.JobStateRunning,
		ActiveStepID: "authorize-2",
		ApprovalRequests: []missioncontrol.ApprovalRequest{
			{
				JobID:           job.ID,
				StepID:          "authorize-1",
				RequestedAction: missioncontrol.ApprovalRequestedActionStepComplete,
				Scope:           missioncontrol.ApprovalScopeOneJob,
				RequestedVia:    missioncontrol.ApprovalRequestedViaRuntime,
				GrantedVia:      missioncontrol.ApprovalGrantedViaOperatorCommand,
				State:           missioncontrol.ApprovalStateGranted,
				RequestedAt:     now.Add(-2 * time.Minute),
				ResolvedAt:      now.Add(-90 * time.Second),
			},
		},
		ApprovalGrants: []missioncontrol.ApprovalGrant{
			{
				JobID:           job.ID,
				StepID:          "authorize-1",
				RequestedAction: missioncontrol.ApprovalRequestedActionStepComplete,
				Scope:           missioncontrol.ApprovalScopeOneJob,
				GrantedVia:      missioncontrol.ApprovalGrantedViaOperatorCommand,
				State:           missioncontrol.ApprovalStateGranted,
				GrantedAt:       now.Add(-90 * time.Second),
				ExpiresAt:       now.Add(time.Minute),
			},
		},
	}

	if err := state.ResumeRuntime(job, runtimeState, nil, true); err != nil {
		t.Fatalf("ResumeRuntime() error = %v", err)
	}
	for i := 0; i < 19; i++ {
		exhausted, err := state.RecordOwnerFacingMessage()
		if err != nil {
			t.Fatalf("RecordOwnerFacingMessage() step %d error = %v", i, err)
		}
		if exhausted {
			t.Fatalf("RecordOwnerFacingMessage() step %d exhausted = true, want false before revoke acknowledgement", i)
		}
	}
	if err := state.RevokeApproval(job.ID, "authorize-2"); err != nil {
		t.Fatalf("RevokeApproval() error = %v", err)
	}

	exhausted, err := state.RecordOwnerFacingRevokeApprovalAck()
	if err != nil {
		t.Fatalf("RecordOwnerFacingRevokeApprovalAck() error = %v", err)
	}
	if !exhausted {
		t.Fatal("RecordOwnerFacingRevokeApprovalAck() exhausted = false, want true at threshold")
	}

	runtime, ok := state.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if runtime.State != missioncontrol.JobStatePaused {
		t.Fatalf("MissionRuntimeState().State = %q, want %q", runtime.State, missioncontrol.JobStatePaused)
	}
	if runtime.ActiveStepID != "authorize-2" {
		t.Fatalf("MissionRuntimeState().ActiveStepID = %q, want %q", runtime.ActiveStepID, "authorize-2")
	}
	if runtime.BudgetBlocker == nil || runtime.BudgetBlocker.Ceiling != "owner_messages" {
		t.Fatalf("MissionRuntimeState().BudgetBlocker = %#v, want owner_messages blocker", runtime.BudgetBlocker)
	}
	if got := runtime.AuditHistory[len(runtime.AuditHistory)-2].ToolName; got != "revoke_approval_ack" {
		t.Fatalf("MissionRuntimeState().penultimate audit tool = %q, want %q", got, "revoke_approval_ack")
	}
	if got := runtime.AuditHistory[len(runtime.AuditHistory)-1].ToolName; got != "budget_exhausted" {
		t.Fatalf("MissionRuntimeState().last audit tool = %q, want %q", got, "budget_exhausted")
	}
}

func TestTaskStateRecordOwnerFacingDenyAckPausesAtOwnerMessageBudget(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	job := testTaskStateJob()
	job.Plan.Steps[0] = missioncontrol.Step{
		ID:      "build",
		Type:    missioncontrol.StepTypeDiscussion,
		Subtype: missioncontrol.StepSubtypeAuthorization,
	}

	if err := state.ActivateStep(job, "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}
	for i := 0; i < 18; i++ {
		exhausted, err := state.RecordOwnerFacingMessage()
		if err != nil {
			t.Fatalf("RecordOwnerFacingMessage() step %d error = %v", i, err)
		}
		if exhausted {
			t.Fatalf("RecordOwnerFacingMessage() step %d exhausted = true, want false before deny acknowledgement", i)
		}
	}
	if err := state.ApplyStepOutput("Waiting for approval.", nil); err != nil {
		t.Fatalf("ApplyStepOutput() error = %v", err)
	}
	if err := state.ApplyApprovalDecision("job-1", "build", missioncontrol.ApprovalDecisionDeny, missioncontrol.ApprovalGrantedViaOperatorCommand); err != nil {
		t.Fatalf("ApplyApprovalDecision() error = %v", err)
	}

	exhausted, err := state.RecordOwnerFacingDenyAck()
	if err != nil {
		t.Fatalf("RecordOwnerFacingDenyAck() error = %v", err)
	}
	if !exhausted {
		t.Fatal("RecordOwnerFacingDenyAck() exhausted = false, want true at threshold")
	}

	runtime, ok := state.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if runtime.State != missioncontrol.JobStatePaused {
		t.Fatalf("MissionRuntimeState().State = %q, want %q", runtime.State, missioncontrol.JobStatePaused)
	}
	if runtime.ActiveStepID != "build" {
		t.Fatalf("MissionRuntimeState().ActiveStepID = %q, want %q", runtime.ActiveStepID, "build")
	}
	if runtime.BudgetBlocker == nil || runtime.BudgetBlocker.Ceiling != "owner_messages" {
		t.Fatalf("MissionRuntimeState().BudgetBlocker = %#v, want owner_messages blocker", runtime.BudgetBlocker)
	}
	if got := runtime.AuditHistory[len(runtime.AuditHistory)-2].ToolName; got != "deny_ack" {
		t.Fatalf("MissionRuntimeState().penultimate audit tool = %q, want %q", got, "deny_ack")
	}
	if got := runtime.AuditHistory[len(runtime.AuditHistory)-1].ToolName; got != "budget_exhausted" {
		t.Fatalf("MissionRuntimeState().last audit tool = %q, want %q", got, "budget_exhausted")
	}
}

func TestTaskStateRecordOwnerFacingPauseAckPausesAtOwnerMessageBudget(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	job := testTaskStateJob()
	if err := state.ActivateStep(job, "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}

	for i := 0; i < 19; i++ {
		exhausted, err := state.RecordOwnerFacingMessage()
		if err != nil {
			t.Fatalf("RecordOwnerFacingMessage() step %d error = %v", i, err)
		}
		if exhausted {
			t.Fatalf("RecordOwnerFacingMessage() step %d exhausted = true, want false before pause acknowledgement", i)
		}
	}
	if err := state.PauseRuntime(job.ID); err != nil {
		t.Fatalf("PauseRuntime() error = %v", err)
	}

	exhausted, err := state.RecordOwnerFacingPauseAck()
	if err != nil {
		t.Fatalf("RecordOwnerFacingPauseAck() error = %v", err)
	}
	if !exhausted {
		t.Fatal("RecordOwnerFacingPauseAck() exhausted = false, want true at threshold")
	}

	runtime, ok := state.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if runtime.State != missioncontrol.JobStatePaused {
		t.Fatalf("MissionRuntimeState().State = %q, want %q", runtime.State, missioncontrol.JobStatePaused)
	}
	if runtime.ActiveStepID != "build" {
		t.Fatalf("MissionRuntimeState().ActiveStepID = %q, want %q", runtime.ActiveStepID, "build")
	}
	if runtime.BudgetBlocker == nil || runtime.BudgetBlocker.Ceiling != "owner_messages" {
		t.Fatalf("MissionRuntimeState().BudgetBlocker = %#v, want owner_messages blocker", runtime.BudgetBlocker)
	}
	if got := runtime.AuditHistory[len(runtime.AuditHistory)-2].ToolName; got != "pause_ack" {
		t.Fatalf("MissionRuntimeState().penultimate audit tool = %q, want %q", got, "pause_ack")
	}
	if got := runtime.AuditHistory[len(runtime.AuditHistory)-1].ToolName; got != "budget_exhausted" {
		t.Fatalf("MissionRuntimeState().last audit tool = %q, want %q", got, "budget_exhausted")
	}
}

func TestTaskStateApplyStepOutputPausesCompletedStaticArtifactStep(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	job := testTaskStateJob()
	job.AllowedTools = []string{"filesystem", "read"}
	job.Plan.Steps[0] = missioncontrol.Step{
		ID:              "build",
		Type:            missioncontrol.StepTypeStaticArtifact,
		AllowedTools:    []string{"filesystem"},
		SuccessCriteria: []string{"Write `report.json` as valid JSON."},
	}

	if err := state.ActivateStep(job, "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}

	if err := state.ApplyStepOutput("Created report.json.", []missioncontrol.RuntimeToolCallEvidence{
		{ToolName: "filesystem", Arguments: map[string]interface{}{"action": "write", "path": "report.json"}, Result: "written"},
		{ToolName: "filesystem", Arguments: map[string]interface{}{"action": "stat", "path": "report.json"}, Result: "exists=true\nkind=file\nname=report.json\nsize=17\n"},
		{ToolName: "filesystem", Arguments: map[string]interface{}{"action": "read", "path": "report.json"}, Result: "{\n  \"ok\": true\n}\n"},
	}); err != nil {
		t.Fatalf("ApplyStepOutput() error = %v", err)
	}

	if _, ok := state.ExecutionContext(); ok {
		t.Fatal("ExecutionContext() ok = true, want false after completed static_artifact pause")
	}

	runtime, ok := state.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if runtime.State != missioncontrol.JobStatePaused {
		t.Fatalf("MissionRuntimeState().State = %q, want %q", runtime.State, missioncontrol.JobStatePaused)
	}
	if len(runtime.CompletedSteps) != 1 || runtime.CompletedSteps[0].StepID != "build" {
		t.Fatalf("MissionRuntimeState().CompletedSteps = %#v, want build completion", runtime.CompletedSteps)
	}
}

func TestTaskStateApplyStepOutputTransitionsDiscussionSubtypeToWaitingUser(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	job := testTaskStateJob()
	job.Plan.Steps[0] = missioncontrol.Step{
		ID:      "build",
		Type:    missioncontrol.StepTypeDiscussion,
		Subtype: missioncontrol.StepSubtypeAuthorization,
	}
	job.Plan.Steps[1] = missioncontrol.Step{
		ID:        "final",
		Type:      missioncontrol.StepTypeFinalResponse,
		DependsOn: []string{"build"},
	}

	if err := state.ActivateStep(job, "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}

	if err := state.ApplyStepOutput("Need approval before continuing.", nil); err != nil {
		t.Fatalf("ApplyStepOutput() error = %v", err)
	}

	ec, ok := state.ExecutionContext()
	if !ok {
		t.Fatal("ExecutionContext() ok = false, want true")
	}
	if ec.Runtime == nil {
		t.Fatal("ExecutionContext().Runtime = nil, want non-nil")
	}
	if ec.Runtime.State != missioncontrol.JobStateWaitingUser {
		t.Fatalf("ExecutionContext().Runtime.State = %q, want %q", ec.Runtime.State, missioncontrol.JobStateWaitingUser)
	}
}

func TestTaskStatePauseRuntimePausesActiveStepWithoutCompletion(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	if err := state.ActivateStep(testTaskStateJob(), "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}

	if err := state.PauseRuntime("job-1"); err != nil {
		t.Fatalf("PauseRuntime() error = %v", err)
	}

	ec, ok := state.ExecutionContext()
	if !ok {
		t.Fatal("ExecutionContext() ok = false, want paused execution context")
	}
	if ec.Runtime == nil {
		t.Fatal("ExecutionContext().Runtime = nil, want non-nil")
	}
	if ec.Runtime.State != missioncontrol.JobStatePaused {
		t.Fatalf("ExecutionContext().Runtime.State = %q, want %q", ec.Runtime.State, missioncontrol.JobStatePaused)
	}
	if ec.Runtime.ActiveStepID != "build" {
		t.Fatalf("ExecutionContext().Runtime.ActiveStepID = %q, want %q", ec.Runtime.ActiveStepID, "build")
	}
	if len(ec.Runtime.CompletedSteps) != 0 {
		t.Fatalf("ExecutionContext().Runtime.CompletedSteps = %#v, want empty", ec.Runtime.CompletedSteps)
	}
}

func TestTaskStatePauseRuntimeRequiresActiveExecutionContextAfterTeardown(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	if err := state.ActivateStep(testTaskStateJob(), "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}

	state.ClearExecutionContext()

	err := state.PauseRuntime("job-1")
	if err == nil {
		t.Fatal("PauseRuntime() error = nil, want active-step failure")
	}

	validationErr, ok := err.(missioncontrol.ValidationError)
	if !ok {
		t.Fatalf("PauseRuntime() error type = %T, want ValidationError", err)
	}
	if validationErr.Code != missioncontrol.RejectionCodeInvalidRuntimeState {
		t.Fatalf("ValidationError.Code = %q, want %q", validationErr.Code, missioncontrol.RejectionCodeInvalidRuntimeState)
	}

	runtime, ok := state.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want preserved runtime")
	}
	if runtime.State != missioncontrol.JobStateRunning {
		t.Fatalf("MissionRuntimeState().State = %q, want %q", runtime.State, missioncontrol.JobStateRunning)
	}
	if runtime.ActiveStepID != "build" {
		t.Fatalf("MissionRuntimeState().ActiveStepID = %q, want %q", runtime.ActiveStepID, "build")
	}
}

func TestTaskStateResumeRuntimeControlRequiresPausedState(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	if err := state.ActivateStep(testTaskStateJob(), "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}

	err := state.ResumeRuntimeControl("job-1")
	if err == nil {
		t.Fatal("ResumeRuntimeControl() error = nil, want paused-state failure")
	}

	validationErr, ok := err.(missioncontrol.ValidationError)
	if !ok {
		t.Fatalf("ResumeRuntimeControl() error type = %T, want ValidationError", err)
	}
	if validationErr.Code != missioncontrol.RejectionCodeInvalidRuntimeState {
		t.Fatalf("ValidationError.Code = %q, want %q", validationErr.Code, missioncontrol.RejectionCodeInvalidRuntimeState)
	}
}

func TestTaskStateHydrateRuntimeControlResumesPausedRuntimeAfterRehydration(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	job := testTaskStateJob()
	runtime := missioncontrol.JobRuntimeState{
		JobID:        job.ID,
		State:        missioncontrol.JobStatePaused,
		ActiveStepID: "build",
		PausedReason: missioncontrol.RuntimePauseReasonOperatorCommand,
	}

	if err := state.HydrateRuntimeControl(job, runtime, nil); err != nil {
		t.Fatalf("HydrateRuntimeControl() error = %v", err)
	}
	if _, ok := state.ExecutionContext(); ok {
		t.Fatal("ExecutionContext() ok = true, want false after rehydration")
	}
	if err := state.ResumeRuntimeControl("job-1"); err != nil {
		t.Fatalf("ResumeRuntimeControl() error = %v", err)
	}

	ec, ok := state.ExecutionContext()
	if !ok {
		t.Fatal("ExecutionContext() ok = false, want restored context")
	}
	if ec.Runtime == nil || ec.Runtime.State != missioncontrol.JobStateRunning {
		t.Fatalf("ExecutionContext().Runtime = %#v, want running runtime", ec.Runtime)
	}
	if ec.Step == nil || ec.Step.ID != "build" {
		t.Fatalf("ExecutionContext().Step = %#v, want build", ec.Step)
	}
}

func TestTaskStateResumeRuntimeControlRejectsCompletedActiveStepReplayAfterRehydration(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	job := testTaskStateJob()
	runtime := missioncontrol.JobRuntimeState{
		JobID:        job.ID,
		State:        missioncontrol.JobStatePaused,
		ActiveStepID: "build",
		PausedReason: missioncontrol.RuntimePauseReasonOperatorCommand,
		CompletedSteps: []missioncontrol.RuntimeStepRecord{
			{StepID: "build", At: time.Date(2026, 3, 27, 12, 0, 0, 0, time.UTC)},
		},
	}

	if err := state.HydrateRuntimeControl(job, runtime, nil); err != nil {
		t.Fatalf("HydrateRuntimeControl() error = %v", err)
	}

	err := state.ResumeRuntimeControl(job.ID)
	if err == nil {
		t.Fatal("ResumeRuntimeControl() error = nil, want completed-step replay rejection")
	}

	validationErr, ok := err.(missioncontrol.ValidationError)
	if !ok {
		t.Fatalf("ResumeRuntimeControl() error type = %T, want ValidationError", err)
	}
	if validationErr.Code != missioncontrol.RejectionCodeInvalidRuntimeState {
		t.Fatalf("ValidationError.Code = %q, want %q", validationErr.Code, missioncontrol.RejectionCodeInvalidRuntimeState)
	}
	if _, ok := state.ExecutionContext(); ok {
		t.Fatal("ExecutionContext() ok = true, want no active context after rejected replay resume")
	}
}

func TestTaskStateResumeRuntimePreservesReusableOneJobApprovalAfterReboot(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	job := missioncontrol.Job{
		ID:           "job-1",
		MaxAuthority: missioncontrol.AuthorityTierHigh,
		AllowedTools: []string{"read"},
		Plan: missioncontrol.Plan{
			ID: "plan-1",
			Steps: []missioncontrol.Step{
				{
					ID:            "authorize-1",
					Type:          missioncontrol.StepTypeDiscussion,
					Subtype:       missioncontrol.StepSubtypeAuthorization,
					ApprovalScope: missioncontrol.ApprovalScopeOneJob,
				},
				{
					ID:            "authorize-2",
					Type:          missioncontrol.StepTypeDiscussion,
					Subtype:       missioncontrol.StepSubtypeAuthorization,
					ApprovalScope: missioncontrol.ApprovalScopeOneJob,
					DependsOn:     []string{"authorize-1"},
				},
				{
					ID:        "final",
					Type:      missioncontrol.StepTypeFinalResponse,
					DependsOn: []string{"authorize-2"},
				},
			},
		},
	}
	now := time.Now().UTC()
	runtimeState := missioncontrol.JobRuntimeState{
		JobID:        job.ID,
		State:        missioncontrol.JobStateRunning,
		ActiveStepID: "authorize-2",
		ApprovalRequests: []missioncontrol.ApprovalRequest{
			{
				JobID:           job.ID,
				StepID:          "authorize-1",
				RequestedAction: missioncontrol.ApprovalRequestedActionStepComplete,
				Scope:           missioncontrol.ApprovalScopeOneJob,
				State:           missioncontrol.ApprovalStateGranted,
				RequestedAt:     now.Add(-2 * time.Minute),
				ResolvedAt:      now.Add(-90 * time.Second),
			},
		},
		ApprovalGrants: []missioncontrol.ApprovalGrant{
			{
				JobID:           job.ID,
				StepID:          "authorize-1",
				RequestedAction: missioncontrol.ApprovalRequestedActionStepComplete,
				Scope:           missioncontrol.ApprovalScopeOneJob,
				GrantedVia:      missioncontrol.ApprovalGrantedViaOperatorCommand,
				State:           missioncontrol.ApprovalStateGranted,
				GrantedAt:       now.Add(-90 * time.Second),
				ExpiresAt:       now.Add(time.Minute),
			},
		},
	}

	if err := state.ResumeRuntime(job, runtimeState, nil, true); err != nil {
		t.Fatalf("ResumeRuntime() error = %v", err)
	}
	if err := state.ApplyStepOutput("Need approval before continuing.", nil); err != nil {
		t.Fatalf("ApplyStepOutput() error = %v", err)
	}

	if _, ok := state.ExecutionContext(); ok {
		t.Fatal("ExecutionContext() ok = true, want false after completion")
	}
	runtime, ok := state.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if runtime.State != missioncontrol.JobStatePaused {
		t.Fatalf("MissionRuntimeState().State = %q, want %q", runtime.State, missioncontrol.JobStatePaused)
	}
	if len(runtime.CompletedSteps) != 1 || runtime.CompletedSteps[0].StepID != "authorize-2" {
		t.Fatalf("MissionRuntimeState().CompletedSteps = %#v, want authorize-2 completion", runtime.CompletedSteps)
	}
	if len(runtime.ApprovalRequests) != 2 || runtime.ApprovalRequests[1].State != missioncontrol.ApprovalStateGranted {
		t.Fatalf("MissionRuntimeState().ApprovalRequests = %#v, want reused one_job approval recorded", runtime.ApprovalRequests)
	}
}

func TestTaskStateResumeRuntimePreservesReusableOneSessionApprovalAfterReboot(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	job := missioncontrol.Job{
		ID:           "job-1",
		MaxAuthority: missioncontrol.AuthorityTierHigh,
		AllowedTools: []string{"read"},
		Plan: missioncontrol.Plan{
			ID: "plan-1",
			Steps: []missioncontrol.Step{
				{
					ID:            "authorize-1",
					Type:          missioncontrol.StepTypeDiscussion,
					Subtype:       missioncontrol.StepSubtypeAuthorization,
					ApprovalScope: missioncontrol.ApprovalScopeOneSession,
				},
				{
					ID:            "authorize-2",
					Type:          missioncontrol.StepTypeDiscussion,
					Subtype:       missioncontrol.StepSubtypeAuthorization,
					ApprovalScope: missioncontrol.ApprovalScopeOneSession,
					DependsOn:     []string{"authorize-1"},
				},
				{
					ID:        "final",
					Type:      missioncontrol.StepTypeFinalResponse,
					DependsOn: []string{"authorize-2"},
				},
			},
		},
	}
	now := time.Now().UTC()
	runtimeState := missioncontrol.JobRuntimeState{
		JobID:        job.ID,
		State:        missioncontrol.JobStateRunning,
		ActiveStepID: "authorize-2",
		ApprovalRequests: []missioncontrol.ApprovalRequest{
			{
				JobID:           job.ID,
				StepID:          "authorize-1",
				RequestedAction: missioncontrol.ApprovalRequestedActionStepComplete,
				Scope:           missioncontrol.ApprovalScopeOneSession,
				SessionChannel:  "telegram",
				SessionChatID:   "chat-42",
				State:           missioncontrol.ApprovalStateGranted,
				RequestedAt:     now.Add(-2 * time.Minute),
				ResolvedAt:      now.Add(-90 * time.Second),
			},
		},
		ApprovalGrants: []missioncontrol.ApprovalGrant{
			{
				JobID:           job.ID,
				StepID:          "authorize-1",
				RequestedAction: missioncontrol.ApprovalRequestedActionStepComplete,
				Scope:           missioncontrol.ApprovalScopeOneSession,
				GrantedVia:      missioncontrol.ApprovalGrantedViaOperatorCommand,
				SessionChannel:  "telegram",
				SessionChatID:   "chat-42",
				State:           missioncontrol.ApprovalStateGranted,
				GrantedAt:       now.Add(-90 * time.Second),
				ExpiresAt:       now.Add(time.Minute),
			},
		},
	}

	state.SetOperatorSession("telegram", "chat-42")
	if err := state.ResumeRuntime(job, runtimeState, nil, true); err != nil {
		t.Fatalf("ResumeRuntime() error = %v", err)
	}
	if err := state.ApplyStepOutput("Need approval before continuing.", nil); err != nil {
		t.Fatalf("ApplyStepOutput() error = %v", err)
	}

	if _, ok := state.ExecutionContext(); ok {
		t.Fatal("ExecutionContext() ok = true, want false after completion")
	}
	runtime, ok := state.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if runtime.State != missioncontrol.JobStatePaused {
		t.Fatalf("MissionRuntimeState().State = %q, want %q", runtime.State, missioncontrol.JobStatePaused)
	}
	if len(runtime.CompletedSteps) != 1 || runtime.CompletedSteps[0].StepID != "authorize-2" {
		t.Fatalf("MissionRuntimeState().CompletedSteps = %#v, want authorize-2 completion", runtime.CompletedSteps)
	}
	if len(runtime.ApprovalRequests) != 2 || runtime.ApprovalRequests[1].State != missioncontrol.ApprovalStateGranted {
		t.Fatalf("MissionRuntimeState().ApprovalRequests = %#v, want reused one_session approval recorded", runtime.ApprovalRequests)
	}
	if runtime.ApprovalRequests[1].SessionChannel != "telegram" || runtime.ApprovalRequests[1].SessionChatID != "chat-42" {
		t.Fatalf("MissionRuntimeState().ApprovalRequests[1] session = (%q, %q), want (%q, %q)", runtime.ApprovalRequests[1].SessionChannel, runtime.ApprovalRequests[1].SessionChatID, "telegram", "chat-42")
	}
}

func TestTaskStateResumeRuntimeUsesDeterministicLatestReusableApprovalAfterReboot(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	job := missioncontrol.Job{
		ID:           "job-1",
		MaxAuthority: missioncontrol.AuthorityTierHigh,
		AllowedTools: []string{"read"},
		Plan: missioncontrol.Plan{
			ID: "plan-1",
			Steps: []missioncontrol.Step{
				{
					ID:            "authorize-a",
					Type:          missioncontrol.StepTypeDiscussion,
					Subtype:       missioncontrol.StepSubtypeAuthorization,
					ApprovalScope: missioncontrol.ApprovalScopeOneJob,
				},
				{
					ID:            "authorize-b",
					Type:          missioncontrol.StepTypeDiscussion,
					Subtype:       missioncontrol.StepSubtypeAuthorization,
					ApprovalScope: missioncontrol.ApprovalScopeOneJob,
					DependsOn:     []string{"authorize-a"},
				},
				{
					ID:            "authorize-c",
					Type:          missioncontrol.StepTypeDiscussion,
					Subtype:       missioncontrol.StepSubtypeAuthorization,
					ApprovalScope: missioncontrol.ApprovalScopeOneJob,
					DependsOn:     []string{"authorize-b"},
				},
				{
					ID:        "final",
					Type:      missioncontrol.StepTypeFinalResponse,
					DependsOn: []string{"authorize-c"},
				},
			},
		},
	}
	now := time.Now().UTC()
	runtimeState := missioncontrol.JobRuntimeState{
		JobID:        job.ID,
		State:        missioncontrol.JobStateRunning,
		ActiveStepID: "authorize-c",
		ApprovalRequests: []missioncontrol.ApprovalRequest{
			{
				JobID:           job.ID,
				StepID:          "authorize-b",
				RequestedAction: missioncontrol.ApprovalRequestedActionStepComplete,
				Scope:           missioncontrol.ApprovalScopeOneJob,
				State:           missioncontrol.ApprovalStateGranted,
				RequestedAt:     now.Add(-90 * time.Second),
				ResolvedAt:      now.Add(-time.Minute),
			},
			{
				JobID:           job.ID,
				StepID:          "authorize-a",
				RequestedAction: missioncontrol.ApprovalRequestedActionStepComplete,
				Scope:           missioncontrol.ApprovalScopeOneJob,
				State:           missioncontrol.ApprovalStateGranted,
				RequestedAt:     now.Add(-3 * time.Minute),
				ResolvedAt:      now.Add(-2 * time.Minute),
			},
		},
		ApprovalGrants: []missioncontrol.ApprovalGrant{
			{
				JobID:           job.ID,
				StepID:          "authorize-b",
				RequestedAction: missioncontrol.ApprovalRequestedActionStepComplete,
				Scope:           missioncontrol.ApprovalScopeOneJob,
				GrantedVia:      missioncontrol.ApprovalGrantedViaOperatorReply,
				State:           missioncontrol.ApprovalStateGranted,
				GrantedAt:       now.Add(-time.Minute),
				ExpiresAt:       now.Add(time.Minute),
			},
			{
				JobID:           job.ID,
				StepID:          "authorize-a",
				RequestedAction: missioncontrol.ApprovalRequestedActionStepComplete,
				Scope:           missioncontrol.ApprovalScopeOneJob,
				GrantedVia:      missioncontrol.ApprovalGrantedViaOperatorCommand,
				State:           missioncontrol.ApprovalStateGranted,
				GrantedAt:       now.Add(-2 * time.Minute),
				ExpiresAt:       now.Add(time.Minute),
			},
		},
	}

	if err := state.ResumeRuntime(job, runtimeState, nil, true); err != nil {
		t.Fatalf("ResumeRuntime() error = %v", err)
	}
	if err := state.ApplyStepOutput("Need approval before continuing.", nil); err != nil {
		t.Fatalf("ApplyStepOutput() error = %v", err)
	}

	runtime, ok := state.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if len(runtime.ApprovalRequests) != 3 {
		t.Fatalf("MissionRuntimeState().ApprovalRequests = %#v, want three approval records", runtime.ApprovalRequests)
	}
	if runtime.ApprovalRequests[2].StepID != "authorize-c" {
		t.Fatalf("MissionRuntimeState().ApprovalRequests[2].StepID = %q, want %q", runtime.ApprovalRequests[2].StepID, "authorize-c")
	}
	if runtime.ApprovalRequests[2].GrantedVia != missioncontrol.ApprovalGrantedViaOperatorReply {
		t.Fatalf("MissionRuntimeState().ApprovalRequests[2].GrantedVia = %q, want %q", runtime.ApprovalRequests[2].GrantedVia, missioncontrol.ApprovalGrantedViaOperatorReply)
	}
}

func TestTaskStateRevokeApprovalPreventsOneJobReuse(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	job := testReusableApprovalJob(missioncontrol.ApprovalScopeOneJob)
	now := time.Now().UTC()
	runtimeState := missioncontrol.JobRuntimeState{
		JobID:        job.ID,
		State:        missioncontrol.JobStateRunning,
		ActiveStepID: "authorize-2",
		ApprovalRequests: []missioncontrol.ApprovalRequest{
			{
				JobID:           job.ID,
				StepID:          "authorize-1",
				RequestedAction: missioncontrol.ApprovalRequestedActionStepComplete,
				Scope:           missioncontrol.ApprovalScopeOneJob,
				RequestedVia:    missioncontrol.ApprovalRequestedViaRuntime,
				GrantedVia:      missioncontrol.ApprovalGrantedViaOperatorCommand,
				State:           missioncontrol.ApprovalStateGranted,
				RequestedAt:     now.Add(-2 * time.Minute),
				ResolvedAt:      now.Add(-90 * time.Second),
			},
		},
		ApprovalGrants: []missioncontrol.ApprovalGrant{
			{
				JobID:           job.ID,
				StepID:          "authorize-1",
				RequestedAction: missioncontrol.ApprovalRequestedActionStepComplete,
				Scope:           missioncontrol.ApprovalScopeOneJob,
				GrantedVia:      missioncontrol.ApprovalGrantedViaOperatorCommand,
				State:           missioncontrol.ApprovalStateGranted,
				GrantedAt:       now.Add(-90 * time.Second),
				ExpiresAt:       now.Add(time.Minute),
			},
		},
	}

	if err := state.ResumeRuntime(job, runtimeState, nil, true); err != nil {
		t.Fatalf("ResumeRuntime() error = %v", err)
	}
	if err := state.RevokeApproval(job.ID, "authorize-2"); err != nil {
		t.Fatalf("RevokeApproval() error = %v", err)
	}
	if err := state.ApplyStepOutput("Need approval before continuing.", nil); err != nil {
		t.Fatalf("ApplyStepOutput() error = %v", err)
	}

	runtime, ok := state.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if runtime.State != missioncontrol.JobStateWaitingUser {
		t.Fatalf("MissionRuntimeState().State = %q, want %q", runtime.State, missioncontrol.JobStateWaitingUser)
	}
	if len(runtime.ApprovalRequests) != 2 || runtime.ApprovalRequests[0].State != missioncontrol.ApprovalStateRevoked || runtime.ApprovalRequests[1].State != missioncontrol.ApprovalStatePending {
		t.Fatalf("MissionRuntimeState().ApprovalRequests = %#v, want revoked then pending approvals", runtime.ApprovalRequests)
	}
	if runtime.ApprovalRequests[0].RevokedAt.IsZero() {
		t.Fatalf("MissionRuntimeState().ApprovalRequests[0].RevokedAt = %v, want stamped revoke time", runtime.ApprovalRequests[0].RevokedAt)
	}
	if len(runtime.ApprovalGrants) != 1 || runtime.ApprovalGrants[0].State != missioncontrol.ApprovalStateRevoked {
		t.Fatalf("MissionRuntimeState().ApprovalGrants = %#v, want one revoked approval grant", runtime.ApprovalGrants)
	}
}

func TestTaskStateRevokeApprovalPreventsOneSessionReuse(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	job := testReusableApprovalJob(missioncontrol.ApprovalScopeOneSession)
	now := time.Now().UTC()
	runtimeState := missioncontrol.JobRuntimeState{
		JobID:        job.ID,
		State:        missioncontrol.JobStateRunning,
		ActiveStepID: "authorize-2",
		ApprovalRequests: []missioncontrol.ApprovalRequest{
			{
				JobID:           job.ID,
				StepID:          "authorize-1",
				RequestedAction: missioncontrol.ApprovalRequestedActionStepComplete,
				Scope:           missioncontrol.ApprovalScopeOneSession,
				RequestedVia:    missioncontrol.ApprovalRequestedViaRuntime,
				GrantedVia:      missioncontrol.ApprovalGrantedViaOperatorCommand,
				SessionChannel:  "telegram",
				SessionChatID:   "chat-42",
				State:           missioncontrol.ApprovalStateGranted,
				RequestedAt:     now.Add(-2 * time.Minute),
				ResolvedAt:      now.Add(-90 * time.Second),
			},
		},
		ApprovalGrants: []missioncontrol.ApprovalGrant{
			{
				JobID:           job.ID,
				StepID:          "authorize-1",
				RequestedAction: missioncontrol.ApprovalRequestedActionStepComplete,
				Scope:           missioncontrol.ApprovalScopeOneSession,
				GrantedVia:      missioncontrol.ApprovalGrantedViaOperatorCommand,
				SessionChannel:  "telegram",
				SessionChatID:   "chat-42",
				State:           missioncontrol.ApprovalStateGranted,
				GrantedAt:       now.Add(-90 * time.Second),
				ExpiresAt:       now.Add(time.Minute),
			},
		},
	}

	state.SetOperatorSession("telegram", "chat-42")
	if err := state.ResumeRuntime(job, runtimeState, nil, true); err != nil {
		t.Fatalf("ResumeRuntime() error = %v", err)
	}
	if err := state.RevokeApproval(job.ID, "authorize-2"); err != nil {
		t.Fatalf("RevokeApproval() error = %v", err)
	}
	if err := state.ApplyStepOutput("Need approval before continuing.", nil); err != nil {
		t.Fatalf("ApplyStepOutput() error = %v", err)
	}

	runtime, ok := state.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if len(runtime.ApprovalRequests) != 2 || runtime.ApprovalRequests[0].State != missioncontrol.ApprovalStateRevoked || runtime.ApprovalRequests[1].State != missioncontrol.ApprovalStatePending {
		t.Fatalf("MissionRuntimeState().ApprovalRequests = %#v, want revoked then pending approvals", runtime.ApprovalRequests)
	}
	if runtime.ApprovalRequests[0].RevokedAt.IsZero() {
		t.Fatalf("MissionRuntimeState().ApprovalRequests[0].RevokedAt = %v, want stamped revoke time", runtime.ApprovalRequests[0].RevokedAt)
	}
	if len(runtime.ApprovalGrants) != 1 || runtime.ApprovalGrants[0].State != missioncontrol.ApprovalStateRevoked {
		t.Fatalf("MissionRuntimeState().ApprovalGrants = %#v, want one revoked approval grant", runtime.ApprovalGrants)
	}
}

func TestTaskStateRevokeApprovalDoesNotAffectDifferentSession(t *testing.T) {
	t.Parallel()

	job := testReusableApprovalJob(missioncontrol.ApprovalScopeOneSession)
	now := time.Now().UTC()
	runtimeState := missioncontrol.JobRuntimeState{
		JobID:        job.ID,
		State:        missioncontrol.JobStateRunning,
		ActiveStepID: "authorize-2",
		ApprovalRequests: []missioncontrol.ApprovalRequest{
			{
				JobID:           job.ID,
				StepID:          "authorize-1",
				RequestedAction: missioncontrol.ApprovalRequestedActionStepComplete,
				Scope:           missioncontrol.ApprovalScopeOneSession,
				RequestedVia:    missioncontrol.ApprovalRequestedViaRuntime,
				GrantedVia:      missioncontrol.ApprovalGrantedViaOperatorCommand,
				SessionChannel:  "telegram",
				SessionChatID:   "chat-42",
				State:           missioncontrol.ApprovalStateGranted,
				RequestedAt:     now.Add(-2 * time.Minute),
				ResolvedAt:      now.Add(-90 * time.Second),
			},
		},
		ApprovalGrants: []missioncontrol.ApprovalGrant{
			{
				JobID:           job.ID,
				StepID:          "authorize-1",
				RequestedAction: missioncontrol.ApprovalRequestedActionStepComplete,
				Scope:           missioncontrol.ApprovalScopeOneSession,
				GrantedVia:      missioncontrol.ApprovalGrantedViaOperatorCommand,
				SessionChannel:  "telegram",
				SessionChatID:   "chat-42",
				State:           missioncontrol.ApprovalStateGranted,
				GrantedAt:       now.Add(-90 * time.Second),
				ExpiresAt:       now.Add(time.Minute),
			},
		},
	}

	state := NewTaskState()
	state.SetOperatorSession("slack", "C123::171234")
	if err := state.ResumeRuntime(job, runtimeState, nil, true); err != nil {
		t.Fatalf("ResumeRuntime() error = %v", err)
	}
	if err := state.RevokeApproval(job.ID, "authorize-2"); err == nil {
		t.Fatal("RevokeApproval() error = nil, want session mismatch failure")
	}

	reuseState := NewTaskState()
	reuseState.SetOperatorSession("telegram", "chat-42")
	if err := reuseState.ResumeRuntime(job, runtimeState, nil, true); err != nil {
		t.Fatalf("ResumeRuntime() error = %v", err)
	}
	if err := reuseState.ApplyStepOutput("Need approval before continuing.", nil); err != nil {
		t.Fatalf("ApplyStepOutput() error = %v", err)
	}

	runtime, ok := reuseState.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if runtime.State != missioncontrol.JobStatePaused {
		t.Fatalf("MissionRuntimeState().State = %q, want %q", runtime.State, missioncontrol.JobStatePaused)
	}
	if len(runtime.ApprovalRequests) != 2 || runtime.ApprovalRequests[1].State != missioncontrol.ApprovalStateGranted {
		t.Fatalf("MissionRuntimeState().ApprovalRequests = %#v, want reused one_session approval recorded", runtime.ApprovalRequests)
	}
}

func TestTaskStateRevokeApprovalWrongJobOrStepDoesNotBind(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	job := testReusableApprovalJob(missioncontrol.ApprovalScopeOneJob)
	now := time.Now().UTC()
	runtimeState := missioncontrol.JobRuntimeState{
		JobID:        job.ID,
		State:        missioncontrol.JobStateRunning,
		ActiveStepID: "authorize-2",
		ApprovalRequests: []missioncontrol.ApprovalRequest{
			{
				JobID:           job.ID,
				StepID:          "authorize-1",
				RequestedAction: missioncontrol.ApprovalRequestedActionStepComplete,
				Scope:           missioncontrol.ApprovalScopeOneJob,
				RequestedVia:    missioncontrol.ApprovalRequestedViaRuntime,
				GrantedVia:      missioncontrol.ApprovalGrantedViaOperatorCommand,
				State:           missioncontrol.ApprovalStateGranted,
				RequestedAt:     now.Add(-2 * time.Minute),
				ResolvedAt:      now.Add(-90 * time.Second),
			},
		},
		ApprovalGrants: []missioncontrol.ApprovalGrant{
			{
				JobID:           job.ID,
				StepID:          "authorize-1",
				RequestedAction: missioncontrol.ApprovalRequestedActionStepComplete,
				Scope:           missioncontrol.ApprovalScopeOneJob,
				GrantedVia:      missioncontrol.ApprovalGrantedViaOperatorCommand,
				State:           missioncontrol.ApprovalStateGranted,
				GrantedAt:       now.Add(-90 * time.Second),
				ExpiresAt:       now.Add(time.Minute),
			},
		},
	}

	if err := state.ResumeRuntime(job, runtimeState, nil, true); err != nil {
		t.Fatalf("ResumeRuntime() error = %v", err)
	}
	if err := state.RevokeApproval("other-job", "authorize-2"); err == nil {
		t.Fatal("RevokeApproval(wrong job) error = nil, want mismatch failure")
	}
	if err := state.RevokeApproval(job.ID, "authorize-1"); err == nil {
		t.Fatal("RevokeApproval(wrong step) error = nil, want mismatch failure")
	}

	runtime, ok := state.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if len(runtime.ApprovalRequests) != 1 || runtime.ApprovalRequests[0].State != missioncontrol.ApprovalStateGranted {
		t.Fatalf("MissionRuntimeState().ApprovalRequests = %#v, want unchanged granted approval", runtime.ApprovalRequests)
	}
	if len(runtime.ApprovalGrants) != 1 || runtime.ApprovalGrants[0].State != missioncontrol.ApprovalStateGranted {
		t.Fatalf("MissionRuntimeState().ApprovalGrants = %#v, want unchanged granted approval", runtime.ApprovalGrants)
	}
}

func TestTaskStatePersistedRuntimeRevocationHonorsRevokedApprovalState(t *testing.T) {
	t.Parallel()

	job := missioncontrol.Job{
		ID:           "job-1",
		MaxAuthority: missioncontrol.AuthorityTierHigh,
		AllowedTools: []string{"read"},
		Plan: missioncontrol.Plan{
			ID: "plan-1",
			Steps: []missioncontrol.Step{
				{
					ID:            "authorize-1",
					Type:          missioncontrol.StepTypeDiscussion,
					Subtype:       missioncontrol.StepSubtypeAuthorization,
					ApprovalScope: missioncontrol.ApprovalScopeOneJob,
				},
				{
					ID:            "authorize-2",
					Type:          missioncontrol.StepTypeDiscussion,
					Subtype:       missioncontrol.StepSubtypeAuthorization,
					ApprovalScope: missioncontrol.ApprovalScopeOneJob,
					DependsOn:     []string{"authorize-1"},
				},
				{
					ID:        "final",
					Type:      missioncontrol.StepTypeFinalResponse,
					DependsOn: []string{"authorize-2"},
				},
			},
		},
	}
	control, err := missioncontrol.BuildRuntimeControlContext(job, "authorize-2")
	if err != nil {
		t.Fatalf("BuildRuntimeControlContext() error = %v", err)
	}
	now := time.Now().UTC()
	runtimeState := missioncontrol.JobRuntimeState{
		JobID:        job.ID,
		State:        missioncontrol.JobStateRunning,
		ActiveStepID: "authorize-2",
		ApprovalRequests: []missioncontrol.ApprovalRequest{
			{
				JobID:           job.ID,
				StepID:          "authorize-1",
				RequestedAction: missioncontrol.ApprovalRequestedActionStepComplete,
				Scope:           missioncontrol.ApprovalScopeOneJob,
				RequestedVia:    missioncontrol.ApprovalRequestedViaRuntime,
				GrantedVia:      missioncontrol.ApprovalGrantedViaOperatorCommand,
				State:           missioncontrol.ApprovalStateGranted,
				RequestedAt:     now.Add(-2 * time.Minute),
				ResolvedAt:      now.Add(-90 * time.Second),
			},
		},
		ApprovalGrants: []missioncontrol.ApprovalGrant{
			{
				JobID:           job.ID,
				StepID:          "authorize-1",
				RequestedAction: missioncontrol.ApprovalRequestedActionStepComplete,
				Scope:           missioncontrol.ApprovalScopeOneJob,
				GrantedVia:      missioncontrol.ApprovalGrantedViaOperatorCommand,
				State:           missioncontrol.ApprovalStateGranted,
				GrantedAt:       now.Add(-90 * time.Second),
				ExpiresAt:       now.Add(time.Minute),
			},
		},
	}

	state := NewTaskState()
	if err := state.HydrateRuntimeControl(job, runtimeState, &control); err != nil {
		t.Fatalf("HydrateRuntimeControl() error = %v", err)
	}
	if err := state.RevokeApproval(job.ID, "authorize-2"); err != nil {
		t.Fatalf("RevokeApproval() error = %v", err)
	}

	revokedRuntime, ok := state.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if len(revokedRuntime.ApprovalGrants) != 1 || revokedRuntime.ApprovalGrants[0].State != missioncontrol.ApprovalStateRevoked {
		t.Fatalf("MissionRuntimeState().ApprovalGrants = %#v, want one revoked approval grant", revokedRuntime.ApprovalGrants)
	}

	resumed := NewTaskState()
	if err := resumed.ResumeRuntime(job, revokedRuntime, nil, true); err != nil {
		t.Fatalf("ResumeRuntime() error = %v", err)
	}
	if err := resumed.ApplyStepOutput("Need approval before continuing.", nil); err != nil {
		t.Fatalf("ApplyStepOutput() error = %v", err)
	}

	runtime, ok := resumed.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if runtime.State != missioncontrol.JobStateWaitingUser {
		t.Fatalf("MissionRuntimeState().State = %q, want %q", runtime.State, missioncontrol.JobStateWaitingUser)
	}
	if len(runtime.ApprovalRequests) != 2 || runtime.ApprovalRequests[1].State != missioncontrol.ApprovalStatePending {
		t.Fatalf("MissionRuntimeState().ApprovalRequests = %#v, want revoked then pending approvals after reboot-safe resume", runtime.ApprovalRequests)
	}
}

func TestTaskStateResumeRuntimeControlResumesPausedRuntimeAfterExecutionContextTeardown(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	if err := state.ActivateStep(testTaskStateJob(), "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}
	if err := state.PauseRuntime("job-1"); err != nil {
		t.Fatalf("PauseRuntime() error = %v", err)
	}

	state.ClearExecutionContext()

	if _, ok := state.ExecutionContext(); ok {
		t.Fatal("ExecutionContext() ok = true, want false after teardown")
	}
	if err := state.ResumeRuntimeControl("job-1"); err != nil {
		t.Fatalf("ResumeRuntimeControl() error = %v", err)
	}

	ec, ok := state.ExecutionContext()
	if !ok {
		t.Fatal("ExecutionContext() ok = false, want restored active context")
	}
	if ec.Runtime == nil {
		t.Fatal("ExecutionContext().Runtime = nil, want non-nil")
	}
	if ec.Runtime.State != missioncontrol.JobStateRunning {
		t.Fatalf("ExecutionContext().Runtime.State = %q, want %q", ec.Runtime.State, missioncontrol.JobStateRunning)
	}
	if ec.Runtime.ActiveStepID != "build" {
		t.Fatalf("ExecutionContext().Runtime.ActiveStepID = %q, want %q", ec.Runtime.ActiveStepID, "build")
	}
}

func TestTaskStateResumeRuntimeControlDoesNotBypassPendingApproval(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	job := testTaskStateJob()
	job.Plan.Steps[0] = missioncontrol.Step{
		ID:      "build",
		Type:    missioncontrol.StepTypeDiscussion,
		Subtype: missioncontrol.StepSubtypeAuthorization,
	}

	if err := state.ActivateStep(job, "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}
	if err := state.ApplyStepOutput("Waiting for approval.", nil); err != nil {
		t.Fatalf("ApplyStepOutput() error = %v", err)
	}

	err := state.ResumeRuntimeControl("job-1")
	if err == nil {
		t.Fatal("ResumeRuntimeControl() error = nil, want waiting_user failure")
	}

	ec, ok := state.ExecutionContext()
	if !ok {
		t.Fatal("ExecutionContext() ok = false, want waiting execution context")
	}
	if ec.Runtime == nil || ec.Runtime.State != missioncontrol.JobStateWaitingUser {
		t.Fatalf("ExecutionContext().Runtime = %#v, want waiting_user runtime", ec.Runtime)
	}
	if len(ec.Runtime.ApprovalRequests) != 1 || ec.Runtime.ApprovalRequests[0].State != missioncontrol.ApprovalStatePending {
		t.Fatalf("ExecutionContext().Runtime.ApprovalRequests = %#v, want one pending approval", ec.Runtime.ApprovalRequests)
	}
}

func TestTaskStateResumeRuntimeControlDoesNotBypassDeniedApproval(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	job := testTaskStateJob()
	job.Plan.Steps[0] = missioncontrol.Step{
		ID:      "build",
		Type:    missioncontrol.StepTypeDiscussion,
		Subtype: missioncontrol.StepSubtypeAuthorization,
	}

	if err := state.ActivateStep(job, "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}
	if err := state.ApplyStepOutput("Waiting for approval.", nil); err != nil {
		t.Fatalf("ApplyStepOutput() error = %v", err)
	}
	if err := state.ApplyApprovalDecision("job-1", "build", missioncontrol.ApprovalDecisionDeny, missioncontrol.ApprovalGrantedViaOperatorCommand); err != nil {
		t.Fatalf("ApplyApprovalDecision() error = %v", err)
	}

	err := state.ResumeRuntimeControl("job-1")
	if err == nil {
		t.Fatal("ResumeRuntimeControl() error = nil, want waiting_user failure")
	}

	ec, ok := state.ExecutionContext()
	if !ok {
		t.Fatal("ExecutionContext() ok = false, want waiting execution context")
	}
	if ec.Runtime == nil || ec.Runtime.State != missioncontrol.JobStateWaitingUser {
		t.Fatalf("ExecutionContext().Runtime = %#v, want waiting_user runtime", ec.Runtime)
	}
	if len(ec.Runtime.ApprovalRequests) != 1 || ec.Runtime.ApprovalRequests[0].State != missioncontrol.ApprovalStateDenied {
		t.Fatalf("ExecutionContext().Runtime.ApprovalRequests = %#v, want one denied approval", ec.Runtime.ApprovalRequests)
	}
}

func TestTaskStateResumeRuntimeControlWrongJobDoesNotBindAfterExecutionContextTeardown(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	if err := state.ActivateStep(testTaskStateJob(), "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}
	if err := state.PauseRuntime("job-1"); err != nil {
		t.Fatalf("PauseRuntime() error = %v", err)
	}

	state.ClearExecutionContext()

	err := state.ResumeRuntimeControl("other-job")
	if err == nil {
		t.Fatal("ResumeRuntimeControl() error = nil, want job mismatch failure")
	}

	validationErr, ok := err.(missioncontrol.ValidationError)
	if !ok {
		t.Fatalf("ResumeRuntimeControl() error type = %T, want ValidationError", err)
	}
	if validationErr.Code != missioncontrol.RejectionCodeStepValidationFailed {
		t.Fatalf("ValidationError.Code = %q, want %q", validationErr.Code, missioncontrol.RejectionCodeStepValidationFailed)
	}

	if _, ok := state.ExecutionContext(); ok {
		t.Fatal("ExecutionContext() ok = true, want false after wrong-job rejection")
	}
	runtime, ok := state.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want durable paused runtime")
	}
	if runtime.State != missioncontrol.JobStatePaused {
		t.Fatalf("MissionRuntimeState().State = %q, want %q", runtime.State, missioncontrol.JobStatePaused)
	}
	if runtime.ActiveStepID != "build" {
		t.Fatalf("MissionRuntimeState().ActiveStepID = %q, want %q", runtime.ActiveStepID, "build")
	}
}

func TestTaskStateHydrateRuntimeControlWrongJobDoesNotBindAfterRehydration(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	job := testTaskStateJob()
	runtime := missioncontrol.JobRuntimeState{
		JobID:        job.ID,
		State:        missioncontrol.JobStatePaused,
		ActiveStepID: "build",
		PausedReason: missioncontrol.RuntimePauseReasonOperatorCommand,
	}

	if err := state.HydrateRuntimeControl(job, runtime, nil); err != nil {
		t.Fatalf("HydrateRuntimeControl() error = %v", err)
	}

	err := state.ResumeRuntimeControl("other-job")
	if err == nil {
		t.Fatal("ResumeRuntimeControl() error = nil, want job mismatch failure")
	}

	validationErr, ok := err.(missioncontrol.ValidationError)
	if !ok {
		t.Fatalf("ResumeRuntimeControl() error type = %T, want ValidationError", err)
	}
	if validationErr.Code != missioncontrol.RejectionCodeStepValidationFailed {
		t.Fatalf("ValidationError.Code = %q, want %q", validationErr.Code, missioncontrol.RejectionCodeStepValidationFailed)
	}
}

func TestTaskStateAbortRuntimeTransitionsToAborted(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	if err := state.ActivateStep(testTaskStateJob(), "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}
	if err := state.PauseRuntime("job-1"); err != nil {
		t.Fatalf("PauseRuntime() error = %v", err)
	}

	if err := state.AbortRuntime("job-1"); err != nil {
		t.Fatalf("AbortRuntime() error = %v", err)
	}

	if _, ok := state.ExecutionContext(); ok {
		t.Fatal("ExecutionContext() ok = true, want false after abort")
	}

	runtime, ok := state.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if runtime.State != missioncontrol.JobStateAborted {
		t.Fatalf("MissionRuntimeState().State = %q, want %q", runtime.State, missioncontrol.JobStateAborted)
	}
	if runtime.AbortedReason != missioncontrol.RuntimeAbortReasonOperatorCommand {
		t.Fatalf("MissionRuntimeState().AbortedReason = %q, want %q", runtime.AbortedReason, missioncontrol.RuntimeAbortReasonOperatorCommand)
	}
}

func TestTaskStateHydrateRuntimeControlAbortsWaitingUserRuntimeAfterRehydration(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	job := testTaskStateJob()
	job.Plan.Steps[0] = missioncontrol.Step{
		ID:      "build",
		Type:    missioncontrol.StepTypeDiscussion,
		Subtype: missioncontrol.StepSubtypeAuthorization,
	}
	runtime := missioncontrol.JobRuntimeState{
		JobID:         job.ID,
		State:         missioncontrol.JobStateWaitingUser,
		ActiveStepID:  "build",
		WaitingReason: "awaiting operator input",
		ApprovalRequests: []missioncontrol.ApprovalRequest{
			{
				JobID:           job.ID,
				StepID:          "build",
				RequestedAction: missioncontrol.ApprovalRequestedActionStepComplete,
				Scope:           missioncontrol.ApprovalScopeMissionStep,
				State:           missioncontrol.ApprovalStatePending,
			},
		},
	}

	if err := state.HydrateRuntimeControl(job, runtime, nil); err != nil {
		t.Fatalf("HydrateRuntimeControl() error = %v", err)
	}
	if err := state.AbortRuntime("job-1"); err != nil {
		t.Fatalf("AbortRuntime() error = %v", err)
	}

	if _, ok := state.ExecutionContext(); ok {
		t.Fatal("ExecutionContext() ok = true, want false after abort")
	}
	got, ok := state.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if got.State != missioncontrol.JobStateAborted {
		t.Fatalf("MissionRuntimeState().State = %q, want %q", got.State, missioncontrol.JobStateAborted)
	}
}

func TestTaskStateAbortRuntimeAbortsWaitingUserRuntimeAfterExecutionContextTeardown(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	job := testTaskStateJob()
	job.Plan.Steps[0] = missioncontrol.Step{
		ID:      "build",
		Type:    missioncontrol.StepTypeDiscussion,
		Subtype: missioncontrol.StepSubtypeAuthorization,
	}

	if err := state.ActivateStep(job, "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}
	if err := state.ApplyStepOutput("Waiting for approval.", nil); err != nil {
		t.Fatalf("ApplyStepOutput() error = %v", err)
	}

	state.ClearExecutionContext()

	if err := state.AbortRuntime("job-1"); err != nil {
		t.Fatalf("AbortRuntime() error = %v", err)
	}

	if _, ok := state.ExecutionContext(); ok {
		t.Fatal("ExecutionContext() ok = true, want false after abort")
	}
	runtime, ok := state.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if runtime.State != missioncontrol.JobStateAborted {
		t.Fatalf("MissionRuntimeState().State = %q, want %q", runtime.State, missioncontrol.JobStateAborted)
	}
	if runtime.AbortedReason != missioncontrol.RuntimeAbortReasonOperatorCommand {
		t.Fatalf("MissionRuntimeState().AbortedReason = %q, want %q", runtime.AbortedReason, missioncontrol.RuntimeAbortReasonOperatorCommand)
	}
}

func TestTaskStateHydrateRuntimeControlRejectsTerminalOperatorCommands(t *testing.T) {
	t.Parallel()

	for _, stateValue := range []missioncontrol.JobState{
		missioncontrol.JobStateCompleted,
		missioncontrol.JobStateFailed,
		missioncontrol.JobStateAborted,
	} {
		stateValue := stateValue
		t.Run(string(stateValue), func(t *testing.T) {
			t.Parallel()

			state := NewTaskState()
			job := testTaskStateJob()
			runtime := missioncontrol.JobRuntimeState{
				JobID: job.ID,
				State: stateValue,
			}

			if err := state.HydrateRuntimeControl(job, runtime, nil); err != nil {
				t.Fatalf("HydrateRuntimeControl() error = %v", err)
			}

			for _, run := range []struct {
				name string
				fn   func() error
			}{
				{name: "resume", fn: func() error { return state.ResumeRuntimeControl(job.ID) }},
				{name: "abort", fn: func() error { return state.AbortRuntime(job.ID) }},
			} {
				err := run.fn()
				if err == nil {
					t.Fatalf("%s error = nil, want invalid runtime state", run.name)
				}
				validationErr, ok := err.(missioncontrol.ValidationError)
				if !ok {
					t.Fatalf("%s error type = %T, want ValidationError", run.name, err)
				}
				if validationErr.Code != missioncontrol.RejectionCodeInvalidRuntimeState {
					t.Fatalf("%s ValidationError.Code = %q, want %q", run.name, validationErr.Code, missioncontrol.RejectionCodeInvalidRuntimeState)
				}
			}
		})
	}
}

func TestTaskStateApplyApprovalDecisionPausesCompletedStep(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	job := testTaskStateJob()
	job.Plan.Steps[0] = missioncontrol.Step{
		ID:      "build",
		Type:    missioncontrol.StepTypeDiscussion,
		Subtype: missioncontrol.StepSubtypeAuthorization,
	}
	job.Plan.Steps[1] = missioncontrol.Step{
		ID:        "final",
		Type:      missioncontrol.StepTypeFinalResponse,
		DependsOn: []string{"build"},
	}

	if err := state.ActivateStep(job, "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}
	if err := state.ApplyStepOutput("Waiting for approval.", nil); err != nil {
		t.Fatalf("ApplyStepOutput() error = %v", err)
	}

	if err := state.ApplyApprovalDecision("job-1", "build", missioncontrol.ApprovalDecisionApprove, missioncontrol.ApprovalGrantedViaOperatorCommand); err != nil {
		t.Fatalf("ApplyApprovalDecision() error = %v", err)
	}
	inputKind, err := state.ApplyWaitingUserInput("approved")
	if err != nil {
		t.Fatalf("ApplyWaitingUserInput() error = %v", err)
	}
	if inputKind != missioncontrol.WaitingUserInputNone {
		t.Fatalf("ApplyWaitingUserInput() kind = %q, want %q after approval completion", inputKind, missioncontrol.WaitingUserInputNone)
	}
	if _, ok := state.ExecutionContext(); ok {
		t.Fatal("ExecutionContext() ok = true, want false after approval completion")
	}

	runtime, ok := state.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if runtime.State != missioncontrol.JobStatePaused {
		t.Fatalf("MissionRuntimeState().State = %q, want %q", runtime.State, missioncontrol.JobStatePaused)
	}
	if len(runtime.CompletedSteps) != 1 || runtime.CompletedSteps[0].StepID != "build" {
		t.Fatalf("MissionRuntimeState().CompletedSteps = %#v, want build completion", runtime.CompletedSteps)
	}
	if len(runtime.ApprovalGrants) != 1 || runtime.ApprovalGrants[0].State != missioncontrol.ApprovalStateGranted {
		t.Fatalf("MissionRuntimeState().ApprovalGrants = %#v, want one granted approval", runtime.ApprovalGrants)
	}
}

func TestTaskStateApplyApprovalDecisionUsesPersistedRuntimeControlAfterExecutionContextTeardown(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	job := testTaskStateJob()
	job.Plan.Steps[0] = missioncontrol.Step{
		ID:      "build",
		Type:    missioncontrol.StepTypeDiscussion,
		Subtype: missioncontrol.StepSubtypeAuthorization,
	}
	job.Plan.Steps[1] = missioncontrol.Step{
		ID:        "final",
		Type:      missioncontrol.StepTypeFinalResponse,
		DependsOn: []string{"build"},
	}

	if err := state.ActivateStep(job, "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}
	if err := state.ApplyStepOutput("Waiting for approval.", nil); err != nil {
		t.Fatalf("ApplyStepOutput() error = %v", err)
	}

	state.SetOperatorSession("telegram", "chat-42")
	state.ClearExecutionContext()

	if err := state.ApplyApprovalDecision("job-1", "build", missioncontrol.ApprovalDecisionApprove, missioncontrol.ApprovalGrantedViaOperatorCommand); err != nil {
		t.Fatalf("ApplyApprovalDecision() error = %v", err)
	}

	if _, ok := state.ExecutionContext(); ok {
		t.Fatal("ExecutionContext() ok = true, want false after reboot-safe approval completion")
	}

	runtime, ok := state.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if runtime.State != missioncontrol.JobStatePaused {
		t.Fatalf("MissionRuntimeState().State = %q, want %q", runtime.State, missioncontrol.JobStatePaused)
	}
	if len(runtime.CompletedSteps) != 1 || runtime.CompletedSteps[0].StepID != "build" {
		t.Fatalf("MissionRuntimeState().CompletedSteps = %#v, want build completion", runtime.CompletedSteps)
	}
	if len(runtime.ApprovalGrants) != 1 || runtime.ApprovalGrants[0].State != missioncontrol.ApprovalStateGranted {
		t.Fatalf("MissionRuntimeState().ApprovalGrants = %#v, want one granted approval", runtime.ApprovalGrants)
	}
	if runtime.ApprovalRequests[0].SessionChannel != "telegram" || runtime.ApprovalRequests[0].SessionChatID != "chat-42" {
		t.Fatalf("MissionRuntimeState().ApprovalRequests[0] session = (%q, %q), want (%q, %q)", runtime.ApprovalRequests[0].SessionChannel, runtime.ApprovalRequests[0].SessionChatID, "telegram", "chat-42")
	}
	if runtime.ApprovalGrants[0].SessionChannel != "telegram" || runtime.ApprovalGrants[0].SessionChatID != "chat-42" {
		t.Fatalf("MissionRuntimeState().ApprovalGrants[0] session = (%q, %q), want (%q, %q)", runtime.ApprovalGrants[0].SessionChannel, runtime.ApprovalGrants[0].SessionChatID, "telegram", "chat-42")
	}
}

func TestTaskStateApplyNaturalApprovalDecisionApprovesSinglePendingRequest(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	job := testTaskStateJob()
	job.Plan.Steps[0] = missioncontrol.Step{
		ID:      "build",
		Type:    missioncontrol.StepTypeDiscussion,
		Subtype: missioncontrol.StepSubtypeAuthorization,
	}

	if err := state.ActivateStep(job, "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}
	if err := state.ApplyStepOutput("Waiting for approval.", nil); err != nil {
		t.Fatalf("ApplyStepOutput() error = %v", err)
	}

	handled, resp, err := state.ApplyNaturalApprovalDecision("yes")
	if err != nil {
		t.Fatalf("ApplyNaturalApprovalDecision(yes) error = %v", err)
	}
	if !handled {
		t.Fatal("ApplyNaturalApprovalDecision(yes) handled = false, want true")
	}
	if resp != "Approved job=job-1 step=build." {
		t.Fatalf("ApplyNaturalApprovalDecision(yes) response = %q, want approval acknowledgement", resp)
	}

	runtime, ok := state.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if runtime.State != missioncontrol.JobStatePaused {
		t.Fatalf("MissionRuntimeState().State = %q, want %q", runtime.State, missioncontrol.JobStatePaused)
	}
}

func TestTaskStateApplyNaturalApprovalDecisionRejectsAmbiguousPendingRequests(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	job := testTaskStateJob()
	job.Plan.Steps[0] = missioncontrol.Step{
		ID:      "build",
		Type:    missioncontrol.StepTypeDiscussion,
		Subtype: missioncontrol.StepSubtypeAuthorization,
	}

	if err := state.ActivateStep(job, "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}
	if err := state.ApplyStepOutput("Waiting for approval.", nil); err != nil {
		t.Fatalf("ApplyStepOutput() error = %v", err)
	}

	state.mu.Lock()
	state.executionContext.Runtime.ApprovalRequests = append(state.executionContext.Runtime.ApprovalRequests, missioncontrol.ApprovalRequest{
		JobID:           "job-1",
		StepID:          "other-step",
		RequestedAction: missioncontrol.ApprovalRequestedActionStepComplete,
		Scope:           missioncontrol.ApprovalScopeMissionStep,
		State:           missioncontrol.ApprovalStatePending,
	})
	state.runtimeState.ApprovalRequests = append(state.runtimeState.ApprovalRequests, missioncontrol.ApprovalRequest{
		JobID:           "job-1",
		StepID:          "other-step",
		RequestedAction: missioncontrol.ApprovalRequestedActionStepComplete,
		Scope:           missioncontrol.ApprovalScopeMissionStep,
		State:           missioncontrol.ApprovalStatePending,
	})
	state.mu.Unlock()

	handled, _, err := state.ApplyNaturalApprovalDecision("yes")
	if err == nil {
		t.Fatal("ApplyNaturalApprovalDecision(yes) error = nil, want ambiguity failure")
	}
	if !handled {
		t.Fatal("ApplyNaturalApprovalDecision(yes) handled = false, want true")
	}

	runtime, ok := state.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if runtime.State != missioncontrol.JobStateWaitingUser {
		t.Fatalf("MissionRuntimeState().State = %q, want %q", runtime.State, missioncontrol.JobStateWaitingUser)
	}
	if len(runtime.ApprovalRequests) != 2 {
		t.Fatalf("MissionRuntimeState().ApprovalRequests = %#v, want two pending approvals", runtime.ApprovalRequests)
	}
}

func TestTaskStateApplyNaturalApprovalDecisionUsesPersistedRuntimeControlAfterExecutionContextTeardown(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	job := testTaskStateJob()
	job.Plan.Steps[0] = missioncontrol.Step{
		ID:      "build",
		Type:    missioncontrol.StepTypeDiscussion,
		Subtype: missioncontrol.StepSubtypeAuthorization,
	}

	if err := state.ActivateStep(job, "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}
	if err := state.ApplyStepOutput("Waiting for approval.", nil); err != nil {
		t.Fatalf("ApplyStepOutput() error = %v", err)
	}

	state.SetOperatorSession("slack", "C123::171234")
	state.ClearExecutionContext()

	handled, resp, err := state.ApplyNaturalApprovalDecision("yes")
	if err != nil {
		t.Fatalf("ApplyNaturalApprovalDecision(yes) error = %v", err)
	}
	if !handled {
		t.Fatal("ApplyNaturalApprovalDecision(yes) handled = false, want true")
	}
	if resp != "Approved job=job-1 step=build." {
		t.Fatalf("ApplyNaturalApprovalDecision(yes) response = %q, want approval acknowledgement", resp)
	}

	runtime, ok := state.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if runtime.State != missioncontrol.JobStatePaused {
		t.Fatalf("MissionRuntimeState().State = %q, want %q", runtime.State, missioncontrol.JobStatePaused)
	}
	if len(runtime.ApprovalRequests) != 1 || runtime.ApprovalRequests[0].State != missioncontrol.ApprovalStateGranted {
		t.Fatalf("MissionRuntimeState().ApprovalRequests = %#v, want one granted approval", runtime.ApprovalRequests)
	}
	if len(runtime.ApprovalGrants) != 1 || runtime.ApprovalGrants[0].State != missioncontrol.ApprovalStateGranted {
		t.Fatalf("MissionRuntimeState().ApprovalGrants = %#v, want one granted approval", runtime.ApprovalGrants)
	}
	if runtime.ApprovalRequests[0].SessionChannel != "slack" || runtime.ApprovalRequests[0].SessionChatID != "C123::171234" {
		t.Fatalf("MissionRuntimeState().ApprovalRequests[0] session = (%q, %q), want (%q, %q)", runtime.ApprovalRequests[0].SessionChannel, runtime.ApprovalRequests[0].SessionChatID, "slack", "C123::171234")
	}
	if runtime.ApprovalGrants[0].SessionChannel != "slack" || runtime.ApprovalGrants[0].SessionChatID != "C123::171234" {
		t.Fatalf("MissionRuntimeState().ApprovalGrants[0] session = (%q, %q), want (%q, %q)", runtime.ApprovalGrants[0].SessionChannel, runtime.ApprovalGrants[0].SessionChatID, "slack", "C123::171234")
	}
}

func TestTaskStateApplyNaturalApprovalDecisionDoesNotBindExpiredPendingRequest(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	job := testTaskStateJob()
	job.Plan.Steps[0] = missioncontrol.Step{
		ID:      "build",
		Type:    missioncontrol.StepTypeDiscussion,
		Subtype: missioncontrol.StepSubtypeAuthorization,
	}

	if err := state.ActivateStep(job, "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}
	if err := state.ApplyStepOutput("Waiting for approval.", nil); err != nil {
		t.Fatalf("ApplyStepOutput() error = %v", err)
	}

	state.mu.Lock()
	expiredAt := time.Now().Add(-1 * time.Minute)
	state.executionContext.Runtime.ApprovalRequests[0].ExpiresAt = expiredAt
	state.runtimeState.ApprovalRequests[0].ExpiresAt = expiredAt
	state.mu.Unlock()

	for _, input := range []string{"yes", "no"} {
		handled, _, err := state.ApplyNaturalApprovalDecision(input)
		if err != nil {
			t.Fatalf("ApplyNaturalApprovalDecision(%q) error = %v", input, err)
		}
		if handled {
			t.Fatalf("ApplyNaturalApprovalDecision(%q) handled = true, want false", input)
		}
	}

	runtime, ok := state.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if len(runtime.ApprovalRequests) != 1 || runtime.ApprovalRequests[0].State != missioncontrol.ApprovalStateExpired {
		t.Fatalf("MissionRuntimeState().ApprovalRequests = %#v, want one expired approval", runtime.ApprovalRequests)
	}
}

func TestTaskStateApplyNaturalApprovalDecisionDoesNotBindSupersededRequest(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	job := testTaskStateJob()
	job.Plan.Steps[0] = missioncontrol.Step{
		ID:      "build",
		Type:    missioncontrol.StepTypeDiscussion,
		Subtype: missioncontrol.StepSubtypeAuthorization,
	}

	if err := state.ActivateStep(job, "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}
	if err := state.ApplyStepOutput("Waiting for approval.", nil); err != nil {
		t.Fatalf("ApplyStepOutput() error = %v", err)
	}

	now := time.Now()
	state.mu.Lock()
	state.executionContext.Runtime.ApprovalRequests = append(state.executionContext.Runtime.ApprovalRequests, missioncontrol.ApprovalRequest{
		JobID:           "job-1",
		StepID:          "build",
		RequestedAction: missioncontrol.ApprovalRequestedActionStepComplete,
		Scope:           missioncontrol.ApprovalScopeMissionStep,
		State:           missioncontrol.ApprovalStatePending,
		RequestedAt:     now,
	})
	state.runtimeState.ApprovalRequests = append(state.runtimeState.ApprovalRequests, missioncontrol.ApprovalRequest{
		JobID:           "job-1",
		StepID:          "build",
		RequestedAction: missioncontrol.ApprovalRequestedActionStepComplete,
		Scope:           missioncontrol.ApprovalScopeMissionStep,
		State:           missioncontrol.ApprovalStatePending,
		RequestedAt:     now,
	})
	state.executionContext.Runtime.ApprovalRequests[0].State = missioncontrol.ApprovalStateSuperseded
	state.executionContext.Runtime.ApprovalRequests[0].SupersededAt = now
	state.runtimeState.ApprovalRequests[0].State = missioncontrol.ApprovalStateSuperseded
	state.runtimeState.ApprovalRequests[0].SupersededAt = now
	state.mu.Unlock()

	handled, resp, err := state.ApplyNaturalApprovalDecision("yes")
	if err != nil {
		t.Fatalf("ApplyNaturalApprovalDecision(yes) error = %v", err)
	}
	if !handled {
		t.Fatal("ApplyNaturalApprovalDecision(yes) handled = false, want true")
	}
	if resp != "Approved job=job-1 step=build." {
		t.Fatalf("ApplyNaturalApprovalDecision(yes) response = %q, want approval acknowledgement", resp)
	}

	runtime, ok := state.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if len(runtime.ApprovalRequests) != 2 || runtime.ApprovalRequests[0].State != missioncontrol.ApprovalStateSuperseded {
		t.Fatalf("MissionRuntimeState().ApprovalRequests = %#v, want leading superseded request", runtime.ApprovalRequests)
	}
}

func TestTaskStateApplyApprovalDecisionBindsOnlyLatestValidRequest(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	job := testTaskStateJob()
	job.Plan.Steps[0] = missioncontrol.Step{
		ID:      "build",
		Type:    missioncontrol.StepTypeDiscussion,
		Subtype: missioncontrol.StepSubtypeAuthorization,
	}

	if err := state.ActivateStep(job, "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}
	if err := state.ApplyStepOutput("Waiting for approval.", nil); err != nil {
		t.Fatalf("ApplyStepOutput() error = %v", err)
	}

	now := time.Now()
	if state.executionContext.Runtime.ApprovalRequests[0].ExpiresAt.IsZero() {
		t.Fatalf("executionContext approval request = %#v, want stamped expires_at", state.executionContext.Runtime.ApprovalRequests)
	}
	if state.runtimeState.ApprovalRequests[0].ExpiresAt.IsZero() {
		t.Fatalf("runtimeState approval request = %#v, want stamped expires_at", state.runtimeState.ApprovalRequests)
	}
	state.mu.Lock()
	state.executionContext.Runtime.ApprovalRequests[0].State = missioncontrol.ApprovalStateSuperseded
	state.executionContext.Runtime.ApprovalRequests[0].SupersededAt = now
	state.runtimeState.ApprovalRequests[0].State = missioncontrol.ApprovalStateSuperseded
	state.runtimeState.ApprovalRequests[0].SupersededAt = now
	newRequest := missioncontrol.ApprovalRequest{
		JobID:           "job-1",
		StepID:          "build",
		RequestedAction: missioncontrol.ApprovalRequestedActionStepComplete,
		Scope:           missioncontrol.ApprovalScopeMissionStep,
		State:           missioncontrol.ApprovalStatePending,
		RequestedAt:     now.Add(time.Second),
	}
	state.executionContext.Runtime.ApprovalRequests = append(state.executionContext.Runtime.ApprovalRequests, newRequest)
	state.runtimeState.ApprovalRequests = append(state.runtimeState.ApprovalRequests, newRequest)
	state.mu.Unlock()

	if err := state.ApplyApprovalDecision("job-1", "build", missioncontrol.ApprovalDecisionApprove, missioncontrol.ApprovalGrantedViaOperatorCommand); err != nil {
		t.Fatalf("ApplyApprovalDecision() error = %v", err)
	}

	runtime, ok := state.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if len(runtime.ApprovalRequests) != 2 {
		t.Fatalf("len(ApprovalRequests) = %d, want 2", len(runtime.ApprovalRequests))
	}
	if runtime.ApprovalRequests[0].State != missioncontrol.ApprovalStateSuperseded {
		t.Fatalf("ApprovalRequests[0].State = %q, want %q", runtime.ApprovalRequests[0].State, missioncontrol.ApprovalStateSuperseded)
	}
	if runtime.ApprovalRequests[1].State != missioncontrol.ApprovalStateGranted {
		t.Fatalf("ApprovalRequests[1].State = %q, want %q", runtime.ApprovalRequests[1].State, missioncontrol.ApprovalStateGranted)
	}
}

func TestTaskStateApplyNaturalApprovalDecisionDoesNotBindWrongStep(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	job := testTaskStateJob()
	job.Plan.Steps[0] = missioncontrol.Step{
		ID:      "build",
		Type:    missioncontrol.StepTypeDiscussion,
		Subtype: missioncontrol.StepSubtypeAuthorization,
	}

	if err := state.ActivateStep(job, "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}
	if err := state.ApplyStepOutput("Waiting for approval.", nil); err != nil {
		t.Fatalf("ApplyStepOutput() error = %v", err)
	}

	state.mu.Lock()
	state.executionContext.Runtime.ApprovalRequests[0].StepID = "other-step"
	state.runtimeState.ApprovalRequests[0].StepID = "other-step"
	state.mu.Unlock()

	handled, _, err := state.ApplyNaturalApprovalDecision("yes")
	if err == nil {
		t.Fatal("ApplyNaturalApprovalDecision(yes) error = nil, want mismatch failure")
	}
	if !handled {
		t.Fatal("ApplyNaturalApprovalDecision(yes) handled = false, want true")
	}

	runtime, ok := state.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if runtime.State != missioncontrol.JobStateWaitingUser {
		t.Fatalf("MissionRuntimeState().State = %q, want %q", runtime.State, missioncontrol.JobStateWaitingUser)
	}
	if len(runtime.CompletedSteps) != 0 {
		t.Fatalf("MissionRuntimeState().CompletedSteps = %#v, want empty", runtime.CompletedSteps)
	}
}

func TestTaskStateApplyNaturalApprovalDecisionRejectsTerminalPersistedRuntime(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	if err := state.HydrateRuntimeControl(missioncontrol.Job{ID: "job-1"}, missioncontrol.JobRuntimeState{
		JobID: "job-1",
		State: missioncontrol.JobStateCompleted,
	}, nil); err != nil {
		t.Fatalf("HydrateRuntimeControl() error = %v", err)
	}

	handled, _, err := state.ApplyNaturalApprovalDecision("yes")
	if err == nil {
		t.Fatal("ApplyNaturalApprovalDecision(yes) error = nil, want terminal-state rejection")
	}
	if !handled {
		t.Fatal("ApplyNaturalApprovalDecision(yes) handled = false, want true")
	}
}

func TestTaskStateHydrateRuntimeControlExpiresElapsedApprovalImmediately(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	hookCalls := 0
	state.SetRuntimeChangeHook(func() {
		hookCalls++
	})

	expiredAt := time.Now().Add(-1 * time.Minute)
	if err := state.HydrateRuntimeControl(missioncontrol.Job{ID: "job-1"}, missioncontrol.JobRuntimeState{
		JobID:         "job-1",
		State:         missioncontrol.JobStateWaitingUser,
		ActiveStepID:  "build",
		WaitingReason: "awaiting operator input",
		ApprovalRequests: []missioncontrol.ApprovalRequest{
			{
				JobID:           "job-1",
				StepID:          "build",
				RequestedAction: missioncontrol.ApprovalRequestedActionStepComplete,
				Scope:           missioncontrol.ApprovalScopeMissionStep,
				RequestedVia:    missioncontrol.ApprovalRequestedViaRuntime,
				State:           missioncontrol.ApprovalStatePending,
				RequestedAt:     expiredAt.Add(-1 * time.Minute),
				ExpiresAt:       expiredAt,
			},
		},
	}, &missioncontrol.RuntimeControlContext{
		JobID: "job-1",
		Step: missioncontrol.Step{
			ID:      "build",
			Type:    missioncontrol.StepTypeDiscussion,
			Subtype: missioncontrol.StepSubtypeAuthorization,
		},
	}); err != nil {
		t.Fatalf("HydrateRuntimeControl() error = %v", err)
	}

	runtime, ok := state.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if len(runtime.ApprovalRequests) != 1 || runtime.ApprovalRequests[0].State != missioncontrol.ApprovalStateExpired {
		t.Fatalf("MissionRuntimeState().ApprovalRequests = %#v, want one expired approval", runtime.ApprovalRequests)
	}
	if runtime.ApprovalRequests[0].ResolvedAt != expiredAt {
		t.Fatalf("MissionRuntimeState().ApprovalRequests[0].ResolvedAt = %v, want %v", runtime.ApprovalRequests[0].ResolvedAt, expiredAt)
	}
	if hookCalls != 1 {
		t.Fatalf("runtime change hook calls = %d, want 1", hookCalls)
	}
}

func TestTaskStateHydrateRuntimeControlLeavesTerminalRuntimeUnchanged(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	hookCalls := 0
	state.SetRuntimeChangeHook(func() {
		hookCalls++
	})

	expiredAt := time.Now().Add(-1 * time.Minute)
	if err := state.HydrateRuntimeControl(missioncontrol.Job{ID: "job-1"}, missioncontrol.JobRuntimeState{
		JobID: "job-1",
		State: missioncontrol.JobStateCompleted,
		ApprovalRequests: []missioncontrol.ApprovalRequest{
			{
				JobID:           "job-1",
				StepID:          "build",
				RequestedAction: missioncontrol.ApprovalRequestedActionStepComplete,
				Scope:           missioncontrol.ApprovalScopeMissionStep,
				State:           missioncontrol.ApprovalStatePending,
				ExpiresAt:       expiredAt,
			},
		},
	}, nil); err != nil {
		t.Fatalf("HydrateRuntimeControl() error = %v", err)
	}

	runtime, ok := state.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if len(runtime.ApprovalRequests) != 1 || runtime.ApprovalRequests[0].State != missioncontrol.ApprovalStatePending {
		t.Fatalf("MissionRuntimeState().ApprovalRequests = %#v, want unchanged terminal approval request", runtime.ApprovalRequests)
	}
	if hookCalls != 0 {
		t.Fatalf("runtime change hook calls = %d, want 0", hookCalls)
	}
}

func TestTaskStateEmitAuditEventPersistsIntoRuntimeHistoryAndTruncatesDeterministically(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	if err := state.ActivateStep(testTaskStateJob(), "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}

	total := missioncontrol.AuditHistoryCap + 2
	for i := 0; i < total; i++ {
		state.EmitAuditEvent(missioncontrol.AuditEvent{
			JobID:     "job-1",
			StepID:    "build",
			ToolName:  fmt.Sprintf("command-%02d", i),
			Allowed:   true,
			Timestamp: time.Date(2026, 3, 24, 12, 0, i, 0, time.UTC),
		})
	}

	audits := state.AuditEvents()
	if len(audits) != missioncontrol.AuditHistoryCap {
		t.Fatalf("AuditEvents() count = %d, want %d", len(audits), missioncontrol.AuditHistoryCap)
	}

	runtime, ok := state.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if len(runtime.AuditHistory) != missioncontrol.AuditHistoryCap {
		t.Fatalf("MissionRuntimeState().AuditHistory count = %d, want %d", len(runtime.AuditHistory), missioncontrol.AuditHistoryCap)
	}

	for i := 0; i < missioncontrol.AuditHistoryCap; i++ {
		want := fmt.Sprintf("command-%02d", i+2)
		if audits[i].ToolName != want {
			t.Fatalf("AuditEvents()[%d].ToolName = %q, want %q", i, audits[i].ToolName, want)
		}
		if audits[i].EventID == "" {
			t.Fatalf("AuditEvents()[%d].EventID = empty, want deterministic id", i)
		}
		if audits[i].ActionClass != missioncontrol.AuditActionClassToolCall {
			t.Fatalf("AuditEvents()[%d].ActionClass = %q, want %q", i, audits[i].ActionClass, missioncontrol.AuditActionClassToolCall)
		}
		if audits[i].Result != missioncontrol.AuditResultAllowed {
			t.Fatalf("AuditEvents()[%d].Result = %q, want %q", i, audits[i].Result, missioncontrol.AuditResultAllowed)
		}
		if runtime.AuditHistory[i].ToolName != want {
			t.Fatalf("MissionRuntimeState().AuditHistory[%d].ToolName = %q, want %q", i, runtime.AuditHistory[i].ToolName, want)
		}
	}
}

func TestTaskStateHydrateRuntimeControlRestoresAuditHistoryWithoutDuplication(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	job := testTaskStateJob()
	runtime := missioncontrol.JobRuntimeState{
		JobID:        "job-1",
		State:        missioncontrol.JobStatePaused,
		ActiveStepID: "build",
		PausedReason: missioncontrol.RuntimePauseReasonOperatorCommand,
		AuditHistory: []missioncontrol.AuditEvent{
			{
				JobID:     "job-1",
				StepID:    "build",
				ToolName:  "pause",
				Allowed:   true,
				Timestamp: time.Date(2026, 3, 24, 12, 0, 0, 0, time.UTC),
			},
		},
	}

	if err := state.HydrateRuntimeControl(job, runtime, nil); err != nil {
		t.Fatalf("HydrateRuntimeControl() error = %v", err)
	}
	state.ClearExecutionContext()
	if err := state.HydrateRuntimeControl(job, runtime, nil); err != nil {
		t.Fatalf("HydrateRuntimeControl() second error = %v", err)
	}

	audits := state.AuditEvents()
	if len(audits) != 1 {
		t.Fatalf("AuditEvents() count = %d, want 1", len(audits))
	}
	if audits[0].ToolName != "pause" {
		t.Fatalf("AuditEvents()[0].ToolName = %q, want %q", audits[0].ToolName, "pause")
	}

	gotRuntime, ok := state.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if len(gotRuntime.AuditHistory) != 1 {
		t.Fatalf("MissionRuntimeState().AuditHistory count = %d, want 1", len(gotRuntime.AuditHistory))
	}
	expectedAudit := missioncontrol.AppendAuditHistory(nil, missioncontrol.AuditEvent{
		JobID:     "job-1",
		StepID:    "build",
		ToolName:  "pause",
		Allowed:   true,
		Timestamp: time.Date(2026, 3, 24, 12, 0, 0, 0, time.UTC),
	})[0]
	if gotRuntime.AuditHistory[0] != expectedAudit {
		t.Fatalf("MissionRuntimeState().AuditHistory[0] = %#v, want %#v", gotRuntime.AuditHistory[0], expectedAudit)
	}
}

func TestTaskStateHydrateRuntimeControlNormalizesLegacyRevokedAtOnlyOnce(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	hookCalls := 0
	state.SetRuntimeChangeHook(func() {
		hookCalls++
	})

	job := testTaskStateJob()
	revokedAt := time.Date(2026, 3, 24, 12, 1, 0, 0, time.UTC)
	runtime := missioncontrol.JobRuntimeState{
		JobID:        "job-1",
		State:        missioncontrol.JobStatePaused,
		ActiveStepID: "build",
		ApprovalRequests: []missioncontrol.ApprovalRequest{
			{
				JobID:           "job-1",
				StepID:          "build",
				RequestedAction: missioncontrol.ApprovalRequestedActionStepComplete,
				Scope:           missioncontrol.ApprovalScopeOneJob,
				RequestedVia:    missioncontrol.ApprovalRequestedViaRuntime,
				GrantedVia:      missioncontrol.ApprovalGrantedViaOperatorCommand,
				State:           missioncontrol.ApprovalStateRevoked,
			},
		},
		ApprovalGrants: []missioncontrol.ApprovalGrant{
			{
				JobID:           "job-1",
				StepID:          "build",
				RequestedAction: missioncontrol.ApprovalRequestedActionStepComplete,
				Scope:           missioncontrol.ApprovalScopeOneJob,
				GrantedVia:      missioncontrol.ApprovalGrantedViaOperatorCommand,
				State:           missioncontrol.ApprovalStateRevoked,
				RevokedAt:       revokedAt,
			},
		},
	}

	if err := state.HydrateRuntimeControl(job, runtime, nil); err != nil {
		t.Fatalf("HydrateRuntimeControl() error = %v", err)
	}
	gotRuntime, ok := state.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if gotRuntime.ApprovalRequests[0].RevokedAt != revokedAt {
		t.Fatalf("MissionRuntimeState().ApprovalRequests[0].RevokedAt = %v, want %v", gotRuntime.ApprovalRequests[0].RevokedAt, revokedAt)
	}
	if hookCalls != 1 {
		t.Fatalf("runtime change hook calls after first hydrate = %d, want 1", hookCalls)
	}

	state.ClearExecutionContext()
	if hookCalls != 2 {
		t.Fatalf("runtime change hook calls after ClearExecutionContext() = %d, want 2", hookCalls)
	}
	hookCallsAfterClear := hookCalls
	if err := state.HydrateRuntimeControl(job, gotRuntime, nil); err != nil {
		t.Fatalf("HydrateRuntimeControl() second error = %v", err)
	}
	if hookCalls != hookCallsAfterClear {
		t.Fatalf("runtime change hook calls after second hydrate = %d, want unchanged %d", hookCalls, hookCallsAfterClear)
	}
}

func TestTaskStateHydrateRuntimeControlApplyApprovalDecisionRejectsExpiredRequest(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	expiredAt := time.Now().Add(-1 * time.Minute)
	if err := state.HydrateRuntimeControl(missioncontrol.Job{ID: "job-1"}, missioncontrol.JobRuntimeState{
		JobID:         "job-1",
		State:         missioncontrol.JobStateWaitingUser,
		ActiveStepID:  "build",
		WaitingReason: "awaiting operator input",
		ApprovalRequests: []missioncontrol.ApprovalRequest{
			{
				JobID:           "job-1",
				StepID:          "build",
				RequestedAction: missioncontrol.ApprovalRequestedActionStepComplete,
				Scope:           missioncontrol.ApprovalScopeMissionStep,
				RequestedVia:    missioncontrol.ApprovalRequestedViaRuntime,
				State:           missioncontrol.ApprovalStatePending,
				ExpiresAt:       expiredAt,
			},
		},
	}, &missioncontrol.RuntimeControlContext{
		JobID: "job-1",
		Step: missioncontrol.Step{
			ID:      "build",
			Type:    missioncontrol.StepTypeDiscussion,
			Subtype: missioncontrol.StepSubtypeAuthorization,
		},
	}); err != nil {
		t.Fatalf("HydrateRuntimeControl() error = %v", err)
	}

	err := state.ApplyApprovalDecision("job-1", "build", missioncontrol.ApprovalDecisionApprove, missioncontrol.ApprovalGrantedViaOperatorCommand)
	if err == nil {
		t.Fatal("ApplyApprovalDecision() error = nil, want expired approval failure")
	}

	runtime, ok := state.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if len(runtime.ApprovalRequests) != 1 || runtime.ApprovalRequests[0].State != missioncontrol.ApprovalStateExpired {
		t.Fatalf("MissionRuntimeState().ApprovalRequests = %#v, want one expired approval", runtime.ApprovalRequests)
	}
}

func TestTaskStateApplyApprovalDecisionDenyAfterExecutionContextTeardownBlocksLaterFreeFormInput(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	job := testTaskStateJob()
	job.Plan.Steps[0] = missioncontrol.Step{
		ID:      "build",
		Type:    missioncontrol.StepTypeDiscussion,
		Subtype: missioncontrol.StepSubtypeAuthorization,
	}
	job.Plan.Steps[1] = missioncontrol.Step{
		ID:        "final",
		Type:      missioncontrol.StepTypeFinalResponse,
		DependsOn: []string{"build"},
	}

	if err := state.ActivateStep(job, "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}
	if err := state.ApplyStepOutput("Waiting for approval.", nil); err != nil {
		t.Fatalf("ApplyStepOutput() error = %v", err)
	}

	state.ClearExecutionContext()

	if err := state.ApplyApprovalDecision("job-1", "build", missioncontrol.ApprovalDecisionDeny, missioncontrol.ApprovalGrantedViaOperatorCommand); err != nil {
		t.Fatalf("ApplyApprovalDecision() error = %v", err)
	}

	inputKind, err := state.ApplyWaitingUserInput("approved")
	if err != nil {
		t.Fatalf("ApplyWaitingUserInput() error = %v", err)
	}
	if inputKind != missioncontrol.WaitingUserInputNone {
		t.Fatalf("ApplyWaitingUserInput() kind = %q, want %q without execution context", inputKind, missioncontrol.WaitingUserInputNone)
	}

	runtime, ok := state.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if runtime.State != missioncontrol.JobStateWaitingUser {
		t.Fatalf("MissionRuntimeState().State = %q, want %q", runtime.State, missioncontrol.JobStateWaitingUser)
	}
	if len(runtime.CompletedSteps) != 0 {
		t.Fatalf("MissionRuntimeState().CompletedSteps = %#v, want empty", runtime.CompletedSteps)
	}
	if len(runtime.ApprovalRequests) != 1 || runtime.ApprovalRequests[0].State != missioncontrol.ApprovalStateDenied {
		t.Fatalf("MissionRuntimeState().ApprovalRequests = %#v, want one denied approval", runtime.ApprovalRequests)
	}
}

func TestTaskStateApplyWaitingUserInputDoesNotBindPendingApproval(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	job := testTaskStateJob()
	job.Plan.Steps[0] = missioncontrol.Step{
		ID:      "build",
		Type:    missioncontrol.StepTypeDiscussion,
		Subtype: missioncontrol.StepSubtypeAuthorization,
	}
	job.Plan.Steps[1] = missioncontrol.Step{
		ID:        "final",
		Type:      missioncontrol.StepTypeFinalResponse,
		DependsOn: []string{"build"},
	}

	if err := state.ActivateStep(job, "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}
	if err := state.ApplyStepOutput("Waiting for approval.", nil); err != nil {
		t.Fatalf("ApplyStepOutput() error = %v", err)
	}

	_, err := state.ApplyWaitingUserInput("approved")
	if err == nil {
		t.Fatal("ApplyWaitingUserInput() error = nil, want explicit operator approval failure")
	}

	ec, ok := state.ExecutionContext()
	if !ok {
		t.Fatal("ExecutionContext() ok = false, want waiting execution context")
	}
	if ec.Runtime == nil || ec.Runtime.State != missioncontrol.JobStateWaitingUser {
		t.Fatalf("ExecutionContext().Runtime = %#v, want waiting_user runtime", ec.Runtime)
	}
	if len(ec.Runtime.ApprovalRequests) != 1 || ec.Runtime.ApprovalRequests[0].State != missioncontrol.ApprovalStatePending {
		t.Fatalf("ExecutionContext().Runtime.ApprovalRequests = %#v, want one pending approval", ec.Runtime.ApprovalRequests)
	}
}

func TestTaskStateApplyApprovalDecisionWrongBindingAfterExecutionContextTeardownDoesNotMutateRuntime(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	job := testTaskStateJob()
	job.Plan.Steps[0] = missioncontrol.Step{
		ID:      "build",
		Type:    missioncontrol.StepTypeDiscussion,
		Subtype: missioncontrol.StepSubtypeAuthorization,
	}
	job.Plan.Steps[1] = missioncontrol.Step{
		ID:        "final",
		Type:      missioncontrol.StepTypeFinalResponse,
		DependsOn: []string{"build"},
	}

	if err := state.ActivateStep(job, "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}
	if err := state.ApplyStepOutput("Waiting for approval.", nil); err != nil {
		t.Fatalf("ApplyStepOutput() error = %v", err)
	}

	state.ClearExecutionContext()
	state.SetOperatorSession("telegram", "chat-99")

	for _, tc := range []struct {
		name   string
		jobID  string
		stepID string
	}{
		{name: "wrong job", jobID: "other-job", stepID: "build"},
		{name: "wrong step", jobID: "job-1", stepID: "other-step"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := state.ApplyApprovalDecision(tc.jobID, tc.stepID, missioncontrol.ApprovalDecisionApprove, missioncontrol.ApprovalGrantedViaOperatorCommand)
			if err == nil {
				t.Fatal("ApplyApprovalDecision() error = nil, want mismatch failure")
			}
		})
	}

	runtime, ok := state.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if runtime.State != missioncontrol.JobStateWaitingUser {
		t.Fatalf("MissionRuntimeState().State = %q, want %q", runtime.State, missioncontrol.JobStateWaitingUser)
	}
	if len(runtime.ApprovalRequests) != 1 || runtime.ApprovalRequests[0].State != missioncontrol.ApprovalStatePending {
		t.Fatalf("MissionRuntimeState().ApprovalRequests = %#v, want one pending approval", runtime.ApprovalRequests)
	}
	if runtime.ApprovalRequests[0].SessionChannel != "" || runtime.ApprovalRequests[0].SessionChatID != "" {
		t.Fatalf("MissionRuntimeState().ApprovalRequests[0] session = (%q, %q), want empty session on non-binding mismatch", runtime.ApprovalRequests[0].SessionChannel, runtime.ApprovalRequests[0].SessionChatID)
	}
}

func TestTaskStateApplyWaitingUserInputDoesNotCompleteDeniedApproval(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	job := testTaskStateJob()
	job.Plan.Steps[0] = missioncontrol.Step{
		ID:      "build",
		Type:    missioncontrol.StepTypeDiscussion,
		Subtype: missioncontrol.StepSubtypeAuthorization,
	}
	job.Plan.Steps[1] = missioncontrol.Step{
		ID:        "final",
		Type:      missioncontrol.StepTypeFinalResponse,
		DependsOn: []string{"build"},
	}

	if err := state.ActivateStep(job, "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}
	if err := state.ApplyStepOutput("Waiting for approval.", nil); err != nil {
		t.Fatalf("ApplyStepOutput() error = %v", err)
	}
	if err := state.ApplyApprovalDecision("job-1", "build", missioncontrol.ApprovalDecisionDeny, missioncontrol.ApprovalGrantedViaOperatorCommand); err != nil {
		t.Fatalf("ApplyApprovalDecision() error = %v", err)
	}

	_, err := state.ApplyWaitingUserInput("go ahead")
	if err == nil {
		t.Fatal("ApplyWaitingUserInput() error = nil, want denied approval failure")
	}

	ec, ok := state.ExecutionContext()
	if !ok {
		t.Fatal("ExecutionContext() ok = false, want waiting execution context")
	}
	if ec.Runtime == nil || ec.Runtime.State != missioncontrol.JobStateWaitingUser {
		t.Fatalf("ExecutionContext().Runtime = %#v, want waiting_user runtime", ec.Runtime)
	}
	if len(ec.Runtime.CompletedSteps) != 0 {
		t.Fatalf("ExecutionContext().Runtime.CompletedSteps = %#v, want empty", ec.Runtime.CompletedSteps)
	}
	if len(ec.Runtime.ApprovalRequests) != 1 || ec.Runtime.ApprovalRequests[0].State != missioncontrol.ApprovalStateDenied {
		t.Fatalf("ExecutionContext().Runtime.ApprovalRequests = %#v, want one denied approval", ec.Runtime.ApprovalRequests)
	}
}

func TestTaskStateOperatorInspectWithoutValidatedPlanReturnsDeterministicError(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	control, err := missioncontrol.BuildRuntimeControlContext(testTaskStateJob(), "build")
	if err != nil {
		t.Fatalf("BuildRuntimeControlContext() error = %v", err)
	}

	state.runtimeState = missioncontrol.JobRuntimeState{
		JobID:        "job-1",
		State:        missioncontrol.JobStatePaused,
		ActiveStepID: "build",
		PausedReason: missioncontrol.RuntimePauseReasonOperatorCommand,
	}
	state.hasRuntimeState = true
	state.runtimeControl = control
	state.hasRuntimeControl = true

	_, err = state.OperatorInspect("job-1", "build")
	if err == nil {
		t.Fatal("OperatorInspect() error = nil, want missing-plan failure")
	}
	if !strings.Contains(err.Error(), string(missioncontrol.RejectionCodeInvalidRuntimeState)) {
		t.Fatalf("OperatorInspect() error = %q, want invalid_runtime_state code", err)
	}
	if !strings.Contains(err.Error(), "inspect command requires validated mission plan") {
		t.Fatalf("OperatorInspect() error = %q, want missing validated plan message", err)
	}
}

func TestTaskStateOperatorInspectActiveExecutionContextZeroTreasuryRefPathUnchanged(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	if err := state.ActivateStep(testTaskStateJob(), "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}

	got, err := state.OperatorInspect("job-1", "final")
	if err != nil {
		t.Fatalf("OperatorInspect() error = %v", err)
	}

	var summary missioncontrol.InspectSummary
	if err := json.Unmarshal([]byte(got), &summary); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if len(summary.Steps) != 1 || summary.Steps[0].StepID != "final" {
		t.Fatalf("Steps = %#v, want one final step", summary.Steps)
	}
	if summary.Steps[0].CampaignPreflight != nil {
		t.Fatalf("CampaignPreflight = %#v, want nil for zero-ref path", summary.Steps[0].CampaignPreflight)
	}
	if summary.Steps[0].TreasuryPreflight != nil {
		t.Fatalf("TreasuryPreflight = %#v, want nil for zero-ref path", summary.Steps[0].TreasuryPreflight)
	}
}

func TestTaskStateOperatorInspectActiveExecutionContextSurfacesResolvedTreasuryPreflight(t *testing.T) {
	t.Parallel()

	root, treasury, container := writeTaskStateTreasuryFixtures(t)
	job := testTaskStateJob()
	job.Plan.Steps[0].TreasuryRef = &missioncontrol.TreasuryRef{TreasuryID: treasury.TreasuryID}

	state := NewTaskState()
	state.SetMissionStoreRoot(root)
	if err := state.ActivateStep(job, "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}

	got, err := state.OperatorInspect("job-1", "build")
	if err != nil {
		t.Fatalf("OperatorInspect() error = %v", err)
	}

	var summary missioncontrol.InspectSummary
	if err := json.Unmarshal([]byte(got), &summary); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if len(summary.Steps) != 1 || summary.Steps[0].StepID != "build" {
		t.Fatalf("Steps = %#v, want one build step", summary.Steps)
	}
	if summary.Steps[0].CampaignPreflight != nil {
		t.Fatalf("CampaignPreflight = %#v, want nil on treasury-only path", summary.Steps[0].CampaignPreflight)
	}
	if summary.Steps[0].TreasuryPreflight == nil {
		t.Fatal("TreasuryPreflight = nil, want resolved treasury/container data")
	}
	if summary.Steps[0].TreasuryPreflight.Treasury == nil {
		t.Fatal("TreasuryPreflight.Treasury = nil, want resolved treasury record")
	}
	if summary.Steps[0].TreasuryPreflight.Treasury.TreasuryID != treasury.TreasuryID {
		t.Fatalf("TreasuryPreflight.Treasury.TreasuryID = %q, want %q", summary.Steps[0].TreasuryPreflight.Treasury.TreasuryID, treasury.TreasuryID)
	}
	if summary.Steps[0].TreasuryPreflight.Treasury.State != missioncontrol.TreasuryStateActive {
		t.Fatalf("TreasuryPreflight.Treasury.State = %q, want %q", summary.Steps[0].TreasuryPreflight.Treasury.State, missioncontrol.TreasuryStateActive)
	}
	if !reflect.DeepEqual(summary.Steps[0].TreasuryPreflight.Containers, []missioncontrol.FrankContainerRecord{container}) {
		t.Fatalf("TreasuryPreflight.Containers = %#v, want [%#v]", summary.Steps[0].TreasuryPreflight.Containers, container)
	}
}

func TestTaskStateOperatorInspectActiveExecutionContextSurfacesResolvedCampaignPreflight(t *testing.T) {
	t.Parallel()

	root, _, container := writeTaskStateTreasuryFixtures(t)
	campaign := mustStoreTaskStateCampaignFixture(t, root, container)
	job := testTaskStateJob()
	job.Plan.Steps[0].CampaignRef = &missioncontrol.CampaignRef{CampaignID: campaign.CampaignID}

	state := NewTaskState()
	state.SetMissionStoreRoot(root)
	if err := state.ActivateStep(job, "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}

	got, err := state.OperatorInspect("job-1", "build")
	if err != nil {
		t.Fatalf("OperatorInspect() error = %v", err)
	}

	var summary missioncontrol.InspectSummary
	if err := json.Unmarshal([]byte(got), &summary); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if len(summary.Steps) != 1 || summary.Steps[0].StepID != "build" {
		t.Fatalf("Steps = %#v, want one build step", summary.Steps)
	}
	if summary.Steps[0].CampaignPreflight == nil || summary.Steps[0].CampaignPreflight.Campaign == nil {
		t.Fatalf("CampaignPreflight = %#v, want resolved campaign preflight", summary.Steps[0].CampaignPreflight)
	}
	if summary.Steps[0].CampaignPreflight.Campaign.CampaignID != campaign.CampaignID {
		t.Fatalf("CampaignPreflight.Campaign.CampaignID = %q, want %q", summary.Steps[0].CampaignPreflight.Campaign.CampaignID, campaign.CampaignID)
	}
	if len(summary.Steps[0].CampaignPreflight.Identities) != 1 || len(summary.Steps[0].CampaignPreflight.Accounts) != 1 || len(summary.Steps[0].CampaignPreflight.Containers) != 1 {
		t.Fatalf("CampaignPreflight = %#v, want one identity/account/container", summary.Steps[0].CampaignPreflight)
	}
	if summary.Steps[0].TreasuryPreflight != nil {
		t.Fatalf("TreasuryPreflight = %#v, want nil on campaign-only path", summary.Steps[0].TreasuryPreflight)
	}
}

func TestTaskStateOperatorInspectSurfacesCampaignZohoEmailAddressing(t *testing.T) {
	t.Parallel()

	root, _, container := writeTaskStateTreasuryFixtures(t)
	campaign := mustStoreFrankZohoAddressedCampaignFixture(t, root, container)
	job := testTaskStateJob()
	job.Plan.Steps[0].CampaignRef = &missioncontrol.CampaignRef{CampaignID: campaign.CampaignID}

	state := NewTaskState()
	state.SetMissionStoreRoot(root)
	if err := state.ActivateStep(job, "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}

	got, err := state.OperatorInspect("job-1", "build")
	if err != nil {
		t.Fatalf("OperatorInspect() error = %v", err)
	}

	envelope := mustTaskStateJSONObject(t, got)
	steps := mustTaskStateJSONArray(t, envelope["steps"], "inspect.steps")
	step := steps[0].(map[string]any)
	preflight := step["campaign_preflight"].(map[string]any)
	campaignJSON := preflight["campaign"].(map[string]any)
	addressingJSON, ok := campaignJSON["zoho_email_addressing"].(map[string]any)
	if !ok {
		t.Fatalf("steps[0].campaign_preflight.campaign.zoho_email_addressing = %#v, want object", campaignJSON["zoho_email_addressing"])
	}
	assertTaskStateJSONObjectKeys(t, campaignJSON, "campaign_id", "campaign_kind", "compliance_checks", "created_at", "display_name", "failure_threshold", "frank_object_refs", "governed_external_targets", "identity_mode", "objective", "record_version", "state", "stop_conditions", "updated_at", "zoho_email_addressing")
	if !reflect.DeepEqual(mustTaskStateJSONArray(t, addressingJSON["to"], "steps[0].campaign_preflight.campaign.zoho_email_addressing.to"), []any{"person@example.com", "team@example.com"}) {
		t.Fatalf("steps[0].campaign_preflight.campaign.zoho_email_addressing.to = %#v, want [person@example.com team@example.com]", addressingJSON["to"])
	}
	if !reflect.DeepEqual(mustTaskStateJSONArray(t, addressingJSON["cc"], "steps[0].campaign_preflight.campaign.zoho_email_addressing.cc"), []any{"copy@example.com"}) {
		t.Fatalf("steps[0].campaign_preflight.campaign.zoho_email_addressing.cc = %#v, want [copy@example.com]", addressingJSON["cc"])
	}
	if !reflect.DeepEqual(mustTaskStateJSONArray(t, addressingJSON["bcc"], "steps[0].campaign_preflight.campaign.zoho_email_addressing.bcc"), []any{"blind@example.com"}) {
		t.Fatalf("steps[0].campaign_preflight.campaign.zoho_email_addressing.bcc = %#v, want [blind@example.com]", addressingJSON["bcc"])
	}

	var summary missioncontrol.InspectSummary
	if err := json.Unmarshal([]byte(got), &summary); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if len(summary.Steps) != 1 {
		t.Fatalf("Steps len = %d, want 1", len(summary.Steps))
	}
	if summary.Steps[0].CampaignPreflight == nil || summary.Steps[0].CampaignPreflight.Campaign == nil {
		t.Fatalf("CampaignPreflight = %#v, want resolved campaign preflight", summary.Steps[0].CampaignPreflight)
	}
	addressing := summary.Steps[0].CampaignPreflight.Campaign.ZohoEmailAddressing
	if addressing == nil {
		t.Fatalf("CampaignPreflight.Campaign.ZohoEmailAddressing = nil, want campaign-owned Zoho addressing")
	}
	if !reflect.DeepEqual(addressing.To, []string{"person@example.com", "team@example.com"}) {
		t.Fatalf("CampaignPreflight.Campaign.ZohoEmailAddressing.To = %#v, want [person@example.com team@example.com]", addressing.To)
	}
	if !reflect.DeepEqual(addressing.CC, []string{"copy@example.com"}) {
		t.Fatalf("CampaignPreflight.Campaign.ZohoEmailAddressing.CC = %#v, want [copy@example.com]", addressing.CC)
	}
	if !reflect.DeepEqual(addressing.BCC, []string{"blind@example.com"}) {
		t.Fatalf("CampaignPreflight.Campaign.ZohoEmailAddressing.BCC = %#v, want [blind@example.com]", addressing.BCC)
	}
}

func TestTaskStateOperatorInspectActiveAndPersistedPathsPreserveAdapterBoundaryContract(t *testing.T) {
	t.Parallel()

	root, treasury, container := writeTaskStateTreasuryFixtures(t)
	campaign := mustStoreTaskStateCampaignFixture(t, root, container)
	job := testTaskStateJob()
	job.Plan.Steps[0].CampaignRef = &missioncontrol.CampaignRef{CampaignID: campaign.CampaignID}
	job.Plan.Steps[0].TreasuryRef = &missioncontrol.TreasuryRef{TreasuryID: treasury.TreasuryID}

	t.Run("active", func(t *testing.T) {
		t.Parallel()

		state := NewTaskState()
		state.SetMissionStoreRoot(root)
		if err := state.ActivateStep(job, "build"); err != nil {
			t.Fatalf("ActivateStep() error = %v", err)
		}

		got, err := state.OperatorInspect("job-1", "build")
		if err != nil {
			t.Fatalf("OperatorInspect() error = %v", err)
		}
		assertTaskStateReadoutAdapterBoundary(t, got, true, true)

		summary := mustTaskStateReadoutJSON[missioncontrol.InspectSummary](t, got)
		envelope := mustTaskStateJSONObject(t, got)
		assertTaskStateJSONObjectKeys(t, envelope, "allowed_tools", "job_id", "max_authority", "steps")
		steps := mustTaskStateJSONArray(t, envelope["steps"], "inspect.steps")
		if len(steps) != 1 {
			t.Fatalf("steps len = %d, want 1", len(steps))
		}
		step, ok := steps[0].(map[string]any)
		if !ok {
			t.Fatalf("steps[0] = %#v, want object", steps[0])
		}
		assertTaskStateJSONObjectKeys(t, step, "allowed_tools", "campaign_preflight", "depends_on", "effective_allowed_tools", "required_authority", "requires_approval", "step_id", "step_type", "success_criteria", "treasury_preflight")
		assertTaskStateResolvedCampaignPreflightJSONEnvelope(t, step["campaign_preflight"])
		assertTaskStateResolvedTreasuryPreflightJSONEnvelope(t, step["treasury_preflight"])
		if len(summary.Steps) != 1 || summary.Steps[0].StepID != "build" {
			t.Fatalf("Steps = %#v, want one build step", summary.Steps)
		}
		if summary.Steps[0].CampaignPreflight == nil || summary.Steps[0].CampaignPreflight.Campaign == nil {
			t.Fatalf("CampaignPreflight = %#v, want resolved campaign preflight on active path", summary.Steps[0].CampaignPreflight)
		}
		if summary.Steps[0].CampaignPreflight.Campaign.CampaignID != campaign.CampaignID {
			t.Fatalf("CampaignPreflight.Campaign.CampaignID = %q, want %q", summary.Steps[0].CampaignPreflight.Campaign.CampaignID, campaign.CampaignID)
		}
		if summary.Steps[0].TreasuryPreflight == nil || summary.Steps[0].TreasuryPreflight.Treasury == nil {
			t.Fatalf("TreasuryPreflight = %#v, want resolved treasury preflight on active path", summary.Steps[0].TreasuryPreflight)
		}
		if summary.Steps[0].TreasuryPreflight.Treasury.TreasuryID != treasury.TreasuryID {
			t.Fatalf("TreasuryPreflight.Treasury.TreasuryID = %q, want %q", summary.Steps[0].TreasuryPreflight.Treasury.TreasuryID, treasury.TreasuryID)
		}
		if summary.Steps[0].TreasuryPreflight.Treasury.State != missioncontrol.TreasuryStateActive {
			t.Fatalf("TreasuryPreflight.Treasury.State = %q, want %q", summary.Steps[0].TreasuryPreflight.Treasury.State, missioncontrol.TreasuryStateActive)
		}
		if !reflect.DeepEqual(summary.Steps[0].TreasuryPreflight.Containers, []missioncontrol.FrankContainerRecord{container}) {
			t.Fatalf("TreasuryPreflight.Containers = %#v, want [%#v]", summary.Steps[0].TreasuryPreflight.Containers, container)
		}
	})

	t.Run("persisted", func(t *testing.T) {
		t.Parallel()

		state := NewTaskState()
		inspectablePlan, err := missioncontrol.BuildInspectablePlanContext(job)
		if err != nil {
			t.Fatalf("BuildInspectablePlanContext() error = %v", err)
		}
		control, err := missioncontrol.BuildRuntimeControlContext(job, "build")
		if err != nil {
			t.Fatalf("BuildRuntimeControlContext() error = %v", err)
		}

		state.runtimeState = missioncontrol.JobRuntimeState{
			JobID:           "job-1",
			State:           missioncontrol.JobStatePaused,
			ActiveStepID:    "build",
			InspectablePlan: &inspectablePlan,
			PausedReason:    missioncontrol.RuntimePauseReasonOperatorCommand,
		}
		state.hasRuntimeState = true
		state.runtimeControl = control
		state.hasRuntimeControl = true

		got, err := state.OperatorInspect("job-1", "build")
		if err != nil {
			t.Fatalf("OperatorInspect() error = %v", err)
		}
		assertTaskStateReadoutAdapterBoundary(t, got, false, false)

		summary := mustTaskStateReadoutJSON[missioncontrol.InspectSummary](t, got)
		envelope := mustTaskStateJSONObject(t, got)
		assertTaskStateJSONObjectKeys(t, envelope, "allowed_tools", "job_id", "max_authority", "steps")
		steps := mustTaskStateJSONArray(t, envelope["steps"], "inspect.steps")
		if len(steps) != 1 {
			t.Fatalf("steps len = %d, want 1", len(steps))
		}
		step, ok := steps[0].(map[string]any)
		if !ok {
			t.Fatalf("steps[0] = %#v, want object", steps[0])
		}
		assertTaskStateJSONObjectKeys(t, step, "allowed_tools", "depends_on", "effective_allowed_tools", "required_authority", "requires_approval", "step_id", "step_type", "success_criteria")
		if len(summary.Steps) != 1 || summary.Steps[0].StepID != "build" {
			t.Fatalf("Steps = %#v, want one build step", summary.Steps)
		}
		if summary.Steps[0].CampaignPreflight != nil {
			t.Fatalf("CampaignPreflight = %#v, want nil for persisted inspectable-plan path", summary.Steps[0].CampaignPreflight)
		}
		if summary.Steps[0].TreasuryPreflight != nil {
			t.Fatalf("TreasuryPreflight = %#v, want nil for persisted inspectable-plan path", summary.Steps[0].TreasuryPreflight)
		}
	})
}

func TestTaskStateActivateStepMissingCampaignFailsClosed(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	job := testTaskStateJob()
	job.Plan.Steps[0].CampaignRef = &missioncontrol.CampaignRef{CampaignID: "campaign-missing"}
	state.SetMissionStoreRoot(t.TempDir())
	err := state.ActivateStep(job, "build")
	if err == nil {
		t.Fatal("ActivateStep() error = nil, want missing campaign rejection")
	}
	if !strings.Contains(err.Error(), missioncontrol.ErrCampaignRecordNotFound.Error()) {
		t.Fatalf("ActivateStep() error = %q, want missing campaign rejection", err)
	}
}

func TestTaskStateActivateStepBootstrapTreasuryWithoutCommittedAcquisitionFailsClosed(t *testing.T) {
	t.Parallel()

	root, treasury, _ := writeTaskStateTreasuryFixtures(t)
	treasury.State = missioncontrol.TreasuryStateBootstrap
	treasury.BootstrapAcquisition = nil
	if err := missioncontrol.StoreTreasuryRecord(root, treasury); err != nil {
		t.Fatalf("StoreTreasuryRecord() error = %v", err)
	}

	job := testTaskStateJob()
	job.Plan.Steps[0].TreasuryRef = &missioncontrol.TreasuryRef{TreasuryID: treasury.TreasuryID}

	state := NewTaskState()
	state.SetMissionStoreRoot(root)
	err := state.ActivateStep(job, "build")
	if err == nil {
		t.Fatal("ActivateStep() error = nil, want fail-closed missing bootstrap acquisition rejection")
	}
	if !strings.Contains(err.Error(), `execution context treasury "treasury-wallet" requires committed treasury.bootstrap_acquisition for first-value acquisition`) {
		t.Fatalf("ActivateStep() error = %q, want missing bootstrap acquisition rejection", err)
	}
}

func TestTaskStateActivateStepNotificationsCapabilityPathCallsHookOnce(t *testing.T) {
	t.Parallel()

	root := writeTaskStateNotificationsCapabilityProposalFixture(t, missioncontrol.CapabilityOnboardingProposalStateApproved)
	job := testTaskStateJob()
	job.Plan.Steps[0].RequiredCapabilities = []string{missioncontrol.NotificationsCapabilityName}
	job.Plan.Steps[0].CapabilityOnboardingProposalRef = &missioncontrol.CapabilityOnboardingProposalRef{
		ProposalID: "proposal-notifications",
	}

	state := NewTaskState()
	state.SetMissionStoreRoot(root)
	calls := 0
	state.notificationsCapabilityHook = func(root string, ec missioncontrol.ExecutionContext, now time.Time) error {
		calls++
		if _, err := missioncontrol.StoreTelegramNotificationsCapabilityExposure(root); err != nil {
			return err
		}
		return nil
	}

	if err := state.ActivateStep(job, "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}
	if calls != 1 {
		t.Fatalf("notificationsCapabilityHook calls = %d, want 1", calls)
	}

	record, err := missioncontrol.RequireExposedNotificationsCapabilityRecord(root)
	if err != nil {
		t.Fatalf("RequireExposedNotificationsCapabilityRecord() error = %v", err)
	}
	if record.CapabilityID != missioncontrol.NotificationsTelegramCapabilityID {
		t.Fatalf("CapabilityID = %q, want %q", record.CapabilityID, missioncontrol.NotificationsTelegramCapabilityID)
	}
}

func TestTaskStateActivateStepNotificationsCapabilityPathInvokesRealMutation(t *testing.T) {
	root := writeTaskStateNotificationsCapabilityProposalFixture(t, missioncontrol.CapabilityOnboardingProposalStateApproved)
	home := t.TempDir()
	configDir := filepath.Join(home, ".picobot")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	configPath := filepath.Join(configDir, "config.json")
	configJSON := `{"channels":{"telegram":{"enabled":true,"token":"telegram-token","allowFrom":["12345"]}}}`
	if err := os.WriteFile(configPath, []byte(configJSON), 0o644); err != nil {
		t.Fatalf("WriteFile(config.json) error = %v", err)
	}
	t.Setenv("HOME", home)

	job := testTaskStateJob()
	job.Plan.Steps[0].RequiredCapabilities = []string{missioncontrol.NotificationsCapabilityName}
	job.Plan.Steps[0].CapabilityOnboardingProposalRef = &missioncontrol.CapabilityOnboardingProposalRef{
		ProposalID: "proposal-notifications",
	}

	state := NewTaskState()
	state.SetMissionStoreRoot(root)

	if err := state.ActivateStep(job, "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}

	record, err := missioncontrol.RequireExposedNotificationsCapabilityRecord(root)
	if err != nil {
		t.Fatalf("RequireExposedNotificationsCapabilityRecord() error = %v", err)
	}
	if record.Validator != missioncontrol.NotificationsTelegramCapabilityValidator {
		t.Fatalf("Validator = %q, want %q", record.Validator, missioncontrol.NotificationsTelegramCapabilityValidator)
	}
}

func TestTaskStateActivateStepNotificationsCapabilityRequiresApprovedProposal(t *testing.T) {
	t.Parallel()

	root := writeTaskStateNotificationsCapabilityProposalFixture(t, missioncontrol.CapabilityOnboardingProposalStateProposed)
	job := testTaskStateJob()
	job.Plan.Steps[0].RequiredCapabilities = []string{missioncontrol.NotificationsCapabilityName}
	job.Plan.Steps[0].CapabilityOnboardingProposalRef = &missioncontrol.CapabilityOnboardingProposalRef{
		ProposalID: "proposal-notifications",
	}

	state := NewTaskState()
	state.SetMissionStoreRoot(root)
	state.notificationsCapabilityHook = func(root string, ec missioncontrol.ExecutionContext, now time.Time) error {
		t.Fatal("notificationsCapabilityHook() called for unapproved proposal")
		return nil
	}

	err := state.ActivateStep(job, "build")
	if err == nil {
		t.Fatal("ActivateStep() error = nil, want approved-proposal rejection")
	}
	if !strings.Contains(err.Error(), `requires approved capability onboarding proposal "proposal-notifications", got state "proposed"`) {
		t.Fatalf("ActivateStep() error = %q, want approved-proposal rejection", err)
	}
}

func TestTaskStateActivateStepNotificationsCapabilityFailsClosedWithoutExposedRecord(t *testing.T) {
	t.Parallel()

	root := writeTaskStateNotificationsCapabilityProposalFixture(t, missioncontrol.CapabilityOnboardingProposalStateApproved)
	job := testTaskStateJob()
	job.Plan.Steps[0].RequiredCapabilities = []string{missioncontrol.NotificationsCapabilityName}
	job.Plan.Steps[0].CapabilityOnboardingProposalRef = &missioncontrol.CapabilityOnboardingProposalRef{
		ProposalID: "proposal-notifications",
	}

	state := NewTaskState()
	state.SetMissionStoreRoot(root)
	state.notificationsCapabilityHook = nil

	err := state.ActivateStep(job, "build")
	if err == nil {
		t.Fatal("ActivateStep() error = nil, want fail-closed notifications exposure rejection")
	}
	if !strings.Contains(err.Error(), `notifications capability requires one committed capability record named "notifications"`) {
		t.Fatalf("ActivateStep() error = %q, want missing capability record rejection", err)
	}
}

func TestTaskStateActivateStepSharedStorageCapabilityPathCallsHookOnce(t *testing.T) {
	t.Parallel()

	root := writeTaskStateSharedStorageCapabilityProposalFixture(t, missioncontrol.CapabilityOnboardingProposalStateApproved)
	job := testTaskStateJob()
	job.Plan.Steps[0].RequiredCapabilities = []string{missioncontrol.SharedStorageCapabilityName}
	job.Plan.Steps[0].CapabilityOnboardingProposalRef = &missioncontrol.CapabilityOnboardingProposalRef{
		ProposalID: "proposal-shared-storage",
	}

	state := NewTaskState()
	state.SetMissionStoreRoot(root)
	calls := 0
	state.sharedStorageCapabilityHook = func(root string, ec missioncontrol.ExecutionContext, now time.Time) error {
		calls++
		if _, err := missioncontrol.StoreWorkspaceSharedStorageCapabilityExposure(root, filepath.Join(t.TempDir(), "workspace")); err != nil {
			return err
		}
		return nil
	}

	if err := state.ActivateStep(job, "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}
	if calls != 1 {
		t.Fatalf("sharedStorageCapabilityHook calls = %d, want 1", calls)
	}

	record, err := missioncontrol.RequireExposedSharedStorageCapabilityRecord(root)
	if err != nil {
		t.Fatalf("RequireExposedSharedStorageCapabilityRecord() error = %v", err)
	}
	if record.CapabilityID != missioncontrol.SharedStorageWorkspaceCapabilityID {
		t.Fatalf("CapabilityID = %q, want %q", record.CapabilityID, missioncontrol.SharedStorageWorkspaceCapabilityID)
	}
}

func TestTaskStateActivateStepSharedStorageCapabilityPathInvokesRealMutation(t *testing.T) {
	root := writeTaskStateSharedStorageCapabilityProposalFixture(t, missioncontrol.CapabilityOnboardingProposalStateApproved)
	home := t.TempDir()
	configDir := filepath.Join(home, ".picobot")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	workspace := filepath.Join(home, "workspace-root")
	configPath := filepath.Join(configDir, "config.json")
	configJSON := fmt.Sprintf(`{"agents":{"defaults":{"workspace":%q}}}`, workspace)
	if err := os.WriteFile(configPath, []byte(configJSON), 0o644); err != nil {
		t.Fatalf("WriteFile(config.json) error = %v", err)
	}
	t.Setenv("HOME", home)

	job := testTaskStateJob()
	job.Plan.Steps[0].RequiredCapabilities = []string{missioncontrol.SharedStorageCapabilityName}
	job.Plan.Steps[0].CapabilityOnboardingProposalRef = &missioncontrol.CapabilityOnboardingProposalRef{
		ProposalID: "proposal-shared-storage",
	}

	state := NewTaskState()
	state.SetMissionStoreRoot(root)

	if err := state.ActivateStep(job, "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}

	record, err := missioncontrol.RequireExposedSharedStorageCapabilityRecord(root)
	if err != nil {
		t.Fatalf("RequireExposedSharedStorageCapabilityRecord() error = %v", err)
	}
	if record.Validator != missioncontrol.SharedStorageWorkspaceCapabilityValidator {
		t.Fatalf("Validator = %q, want %q", record.Validator, missioncontrol.SharedStorageWorkspaceCapabilityValidator)
	}
	if _, err := os.Stat(filepath.Join(workspace, "SOUL.md")); err != nil {
		t.Fatalf("Stat(SOUL.md) error = %v", err)
	}
}

func TestTaskStateActivateStepSharedStorageCapabilityRequiresApprovedProposal(t *testing.T) {
	t.Parallel()

	root := writeTaskStateSharedStorageCapabilityProposalFixture(t, missioncontrol.CapabilityOnboardingProposalStateProposed)
	job := testTaskStateJob()
	job.Plan.Steps[0].RequiredCapabilities = []string{missioncontrol.SharedStorageCapabilityName}
	job.Plan.Steps[0].CapabilityOnboardingProposalRef = &missioncontrol.CapabilityOnboardingProposalRef{
		ProposalID: "proposal-shared-storage",
	}

	state := NewTaskState()
	state.SetMissionStoreRoot(root)
	state.sharedStorageCapabilityHook = func(root string, ec missioncontrol.ExecutionContext, now time.Time) error {
		t.Fatal("sharedStorageCapabilityHook() called for unapproved proposal")
		return nil
	}

	err := state.ActivateStep(job, "build")
	if err == nil {
		t.Fatal("ActivateStep() error = nil, want approved-proposal rejection")
	}
	if !strings.Contains(err.Error(), `requires approved capability onboarding proposal "proposal-shared-storage", got state "proposed"`) {
		t.Fatalf("ActivateStep() error = %q, want approved-proposal rejection", err)
	}
}

func TestTaskStateActivateStepSharedStorageCapabilityFailsClosedWithoutExposedRecord(t *testing.T) {
	t.Parallel()

	root := writeTaskStateSharedStorageCapabilityProposalFixture(t, missioncontrol.CapabilityOnboardingProposalStateApproved)
	job := testTaskStateJob()
	job.Plan.Steps[0].RequiredCapabilities = []string{missioncontrol.SharedStorageCapabilityName}
	job.Plan.Steps[0].CapabilityOnboardingProposalRef = &missioncontrol.CapabilityOnboardingProposalRef{
		ProposalID: "proposal-shared-storage",
	}

	state := NewTaskState()
	state.SetMissionStoreRoot(root)
	state.sharedStorageCapabilityHook = nil

	err := state.ActivateStep(job, "build")
	if err == nil {
		t.Fatal("ActivateStep() error = nil, want fail-closed shared_storage exposure rejection")
	}
	if !strings.Contains(err.Error(), `shared_storage capability requires one committed capability record named "shared_storage"`) {
		t.Fatalf("ActivateStep() error = %q, want missing capability record rejection", err)
	}
}

func TestTaskStateActivateStepContactsCapabilityPathCallsHookOnce(t *testing.T) {
	root := writeTaskStateContactsCapabilityProposalFixture(t, missioncontrol.CapabilityOnboardingProposalStateApproved)
	workspace := writeTaskStateContactsCapabilityConfigFixture(t)
	if _, err := missioncontrol.StoreWorkspaceSharedStorageCapabilityExposure(root, workspace); err != nil {
		t.Fatalf("StoreWorkspaceSharedStorageCapabilityExposure() error = %v", err)
	}

	job := testTaskStateJob()
	job.Plan.Steps[0].RequiredCapabilities = []string{missioncontrol.ContactsCapabilityName}
	job.Plan.Steps[0].CapabilityOnboardingProposalRef = &missioncontrol.CapabilityOnboardingProposalRef{
		ProposalID: "proposal-contacts",
	}

	state := NewTaskState()
	state.SetMissionStoreRoot(root)
	calls := 0
	state.contactsCapabilityHook = func(root string, ec missioncontrol.ExecutionContext, now time.Time) error {
		calls++
		if _, err := missioncontrol.StoreWorkspaceContactsCapabilityExposure(root, workspace); err != nil {
			return err
		}
		return nil
	}

	if err := state.ActivateStep(job, "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}
	if calls != 1 {
		t.Fatalf("contactsCapabilityHook calls = %d, want 1", calls)
	}

	record, err := missioncontrol.RequireExposedContactsCapabilityRecord(root)
	if err != nil {
		t.Fatalf("RequireExposedContactsCapabilityRecord() error = %v", err)
	}
	if record.CapabilityID != missioncontrol.ContactsLocalFileCapabilityID {
		t.Fatalf("CapabilityID = %q, want %q", record.CapabilityID, missioncontrol.ContactsLocalFileCapabilityID)
	}
	if _, err := missioncontrol.RequireReadableContactsSourceRecord(root, workspace); err != nil {
		t.Fatalf("RequireReadableContactsSourceRecord() error = %v", err)
	}
}

func TestTaskStateActivateStepContactsCapabilityPathInvokesRealMutation(t *testing.T) {
	root := writeTaskStateContactsCapabilityProposalFixture(t, missioncontrol.CapabilityOnboardingProposalStateApproved)
	workspace := writeTaskStateContactsCapabilityConfigFixture(t)
	if _, err := missioncontrol.StoreWorkspaceSharedStorageCapabilityExposure(root, workspace); err != nil {
		t.Fatalf("StoreWorkspaceSharedStorageCapabilityExposure() error = %v", err)
	}

	job := testTaskStateJob()
	job.Plan.Steps[0].RequiredCapabilities = []string{missioncontrol.ContactsCapabilityName}
	job.Plan.Steps[0].CapabilityOnboardingProposalRef = &missioncontrol.CapabilityOnboardingProposalRef{
		ProposalID: "proposal-contacts",
	}

	state := NewTaskState()
	state.SetMissionStoreRoot(root)

	if err := state.ActivateStep(job, "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}

	record, err := missioncontrol.RequireExposedContactsCapabilityRecord(root)
	if err != nil {
		t.Fatalf("RequireExposedContactsCapabilityRecord() error = %v", err)
	}
	if record.Validator != missioncontrol.ContactsLocalFileCapabilityValidator {
		t.Fatalf("Validator = %q, want %q", record.Validator, missioncontrol.ContactsLocalFileCapabilityValidator)
	}
	source, err := missioncontrol.RequireReadableContactsSourceRecord(root, workspace)
	if err != nil {
		t.Fatalf("RequireReadableContactsSourceRecord() error = %v", err)
	}
	if source.Path != missioncontrol.ContactsLocalFileDefaultPath {
		t.Fatalf("Path = %q, want %q", source.Path, missioncontrol.ContactsLocalFileDefaultPath)
	}
}

func TestTaskStateActivateStepContactsCapabilityRequiresApprovedProposal(t *testing.T) {
	t.Parallel()

	root := writeTaskStateContactsCapabilityProposalFixture(t, missioncontrol.CapabilityOnboardingProposalStateProposed)
	job := testTaskStateJob()
	job.Plan.Steps[0].RequiredCapabilities = []string{missioncontrol.ContactsCapabilityName}
	job.Plan.Steps[0].CapabilityOnboardingProposalRef = &missioncontrol.CapabilityOnboardingProposalRef{
		ProposalID: "proposal-contacts",
	}

	state := NewTaskState()
	state.SetMissionStoreRoot(root)
	state.contactsCapabilityHook = func(root string, ec missioncontrol.ExecutionContext, now time.Time) error {
		t.Fatal("contactsCapabilityHook() called for unapproved proposal")
		return nil
	}

	err := state.ActivateStep(job, "build")
	if err == nil {
		t.Fatal("ActivateStep() error = nil, want approved-proposal rejection")
	}
	if !strings.Contains(err.Error(), `requires approved capability onboarding proposal "proposal-contacts", got state "proposed"`) {
		t.Fatalf("ActivateStep() error = %q, want approved-proposal rejection", err)
	}
}

func TestTaskStateActivateStepContactsCapabilityFailsClosedWithoutExposedRecord(t *testing.T) {
	root := writeTaskStateContactsCapabilityProposalFixture(t, missioncontrol.CapabilityOnboardingProposalStateApproved)
	workspace := writeTaskStateContactsCapabilityConfigFixture(t)
	if _, err := missioncontrol.StoreWorkspaceSharedStorageCapabilityExposure(root, workspace); err != nil {
		t.Fatalf("StoreWorkspaceSharedStorageCapabilityExposure() error = %v", err)
	}

	job := testTaskStateJob()
	job.Plan.Steps[0].RequiredCapabilities = []string{missioncontrol.ContactsCapabilityName}
	job.Plan.Steps[0].CapabilityOnboardingProposalRef = &missioncontrol.CapabilityOnboardingProposalRef{
		ProposalID: "proposal-contacts",
	}

	state := NewTaskState()
	state.SetMissionStoreRoot(root)
	state.contactsCapabilityHook = nil

	err := state.ActivateStep(job, "build")
	if err == nil {
		t.Fatal("ActivateStep() error = nil, want fail-closed contacts exposure rejection")
	}
	if !strings.Contains(err.Error(), `contacts capability requires one committed capability record named "contacts"`) {
		t.Fatalf("ActivateStep() error = %q, want missing capability record rejection", err)
	}
}

func TestTaskStateActivateStepContactsCapabilityFailsClosedWithoutSharedStorageExposure(t *testing.T) {
	t.Parallel()

	root := writeTaskStateContactsCapabilityProposalFixture(t, missioncontrol.CapabilityOnboardingProposalStateApproved)
	job := testTaskStateJob()
	job.Plan.Steps[0].RequiredCapabilities = []string{missioncontrol.ContactsCapabilityName}
	job.Plan.Steps[0].CapabilityOnboardingProposalRef = &missioncontrol.CapabilityOnboardingProposalRef{
		ProposalID: "proposal-contacts",
	}

	state := NewTaskState()
	state.SetMissionStoreRoot(root)
	state.contactsCapabilityHook = func(root string, ec missioncontrol.ExecutionContext, now time.Time) error {
		t.Fatal("contactsCapabilityHook() called without shared_storage exposure")
		return nil
	}

	err := state.ActivateStep(job, "build")
	if err == nil {
		t.Fatal("ActivateStep() error = nil, want shared_storage rejection")
	}
	if !strings.Contains(err.Error(), `contacts capability requires shared_storage exposure: shared_storage capability requires one committed capability record named "shared_storage"`) {
		t.Fatalf("ActivateStep() error = %q, want shared_storage rejection", err)
	}
}

func TestTaskStateActivateStepLocationCapabilityPathCallsHookOnce(t *testing.T) {
	root := writeTaskStateLocationCapabilityProposalFixture(t, missioncontrol.CapabilityOnboardingProposalStateApproved)
	workspace := writeTaskStateLocationCapabilityConfigFixture(t)
	if _, err := missioncontrol.StoreWorkspaceSharedStorageCapabilityExposure(root, workspace); err != nil {
		t.Fatalf("StoreWorkspaceSharedStorageCapabilityExposure() error = %v", err)
	}

	job := testTaskStateJob()
	job.Plan.Steps[0].RequiredCapabilities = []string{missioncontrol.LocationCapabilityName}
	job.Plan.Steps[0].CapabilityOnboardingProposalRef = &missioncontrol.CapabilityOnboardingProposalRef{
		ProposalID: "proposal-location",
	}

	state := NewTaskState()
	state.SetMissionStoreRoot(root)
	calls := 0
	state.locationCapabilityHook = func(root string, ec missioncontrol.ExecutionContext, now time.Time) error {
		calls++
		if _, err := missioncontrol.StoreWorkspaceLocationCapabilityExposure(root, workspace); err != nil {
			return err
		}
		return nil
	}

	if err := state.ActivateStep(job, "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}
	if calls != 1 {
		t.Fatalf("locationCapabilityHook calls = %d, want 1", calls)
	}

	record, err := missioncontrol.RequireExposedLocationCapabilityRecord(root)
	if err != nil {
		t.Fatalf("RequireExposedLocationCapabilityRecord() error = %v", err)
	}
	if record.CapabilityID != missioncontrol.LocationLocalFileCapabilityID {
		t.Fatalf("CapabilityID = %q, want %q", record.CapabilityID, missioncontrol.LocationLocalFileCapabilityID)
	}
	if _, err := missioncontrol.RequireReadableLocationSourceRecord(root, workspace); err != nil {
		t.Fatalf("RequireReadableLocationSourceRecord() error = %v", err)
	}
}

func TestTaskStateActivateStepLocationCapabilityPathInvokesRealMutation(t *testing.T) {
	root := writeTaskStateLocationCapabilityProposalFixture(t, missioncontrol.CapabilityOnboardingProposalStateApproved)
	workspace := writeTaskStateLocationCapabilityConfigFixture(t)
	if _, err := missioncontrol.StoreWorkspaceSharedStorageCapabilityExposure(root, workspace); err != nil {
		t.Fatalf("StoreWorkspaceSharedStorageCapabilityExposure() error = %v", err)
	}

	job := testTaskStateJob()
	job.Plan.Steps[0].RequiredCapabilities = []string{missioncontrol.LocationCapabilityName}
	job.Plan.Steps[0].CapabilityOnboardingProposalRef = &missioncontrol.CapabilityOnboardingProposalRef{
		ProposalID: "proposal-location",
	}

	state := NewTaskState()
	state.SetMissionStoreRoot(root)

	if err := state.ActivateStep(job, "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}

	record, err := missioncontrol.RequireExposedLocationCapabilityRecord(root)
	if err != nil {
		t.Fatalf("RequireExposedLocationCapabilityRecord() error = %v", err)
	}
	if record.Validator != missioncontrol.LocationLocalFileCapabilityValidator {
		t.Fatalf("Validator = %q, want %q", record.Validator, missioncontrol.LocationLocalFileCapabilityValidator)
	}
	source, err := missioncontrol.RequireReadableLocationSourceRecord(root, workspace)
	if err != nil {
		t.Fatalf("RequireReadableLocationSourceRecord() error = %v", err)
	}
	if source.Path != missioncontrol.LocationLocalFileDefaultPath {
		t.Fatalf("Path = %q, want %q", source.Path, missioncontrol.LocationLocalFileDefaultPath)
	}
}

func TestTaskStateActivateStepLocationCapabilityRequiresApprovedProposal(t *testing.T) {
	t.Parallel()

	root := writeTaskStateLocationCapabilityProposalFixture(t, missioncontrol.CapabilityOnboardingProposalStateProposed)
	job := testTaskStateJob()
	job.Plan.Steps[0].RequiredCapabilities = []string{missioncontrol.LocationCapabilityName}
	job.Plan.Steps[0].CapabilityOnboardingProposalRef = &missioncontrol.CapabilityOnboardingProposalRef{
		ProposalID: "proposal-location",
	}

	state := NewTaskState()
	state.SetMissionStoreRoot(root)
	state.locationCapabilityHook = func(root string, ec missioncontrol.ExecutionContext, now time.Time) error {
		t.Fatal("locationCapabilityHook() called for unapproved proposal")
		return nil
	}

	err := state.ActivateStep(job, "build")
	if err == nil {
		t.Fatal("ActivateStep() error = nil, want approved-proposal rejection")
	}
	if !strings.Contains(err.Error(), `requires approved capability onboarding proposal "proposal-location", got state "proposed"`) {
		t.Fatalf("ActivateStep() error = %q, want approved-proposal rejection", err)
	}
}

func TestTaskStateActivateStepLocationCapabilityFailsClosedWithoutExposedRecord(t *testing.T) {
	root := writeTaskStateLocationCapabilityProposalFixture(t, missioncontrol.CapabilityOnboardingProposalStateApproved)
	workspace := writeTaskStateLocationCapabilityConfigFixture(t)
	if _, err := missioncontrol.StoreWorkspaceSharedStorageCapabilityExposure(root, workspace); err != nil {
		t.Fatalf("StoreWorkspaceSharedStorageCapabilityExposure() error = %v", err)
	}

	job := testTaskStateJob()
	job.Plan.Steps[0].RequiredCapabilities = []string{missioncontrol.LocationCapabilityName}
	job.Plan.Steps[0].CapabilityOnboardingProposalRef = &missioncontrol.CapabilityOnboardingProposalRef{
		ProposalID: "proposal-location",
	}

	state := NewTaskState()
	state.SetMissionStoreRoot(root)
	state.locationCapabilityHook = nil

	err := state.ActivateStep(job, "build")
	if err == nil {
		t.Fatal("ActivateStep() error = nil, want fail-closed location exposure rejection")
	}
	if !strings.Contains(err.Error(), `location capability requires one committed capability record named "location"`) {
		t.Fatalf("ActivateStep() error = %q, want missing capability record rejection", err)
	}
}

func TestTaskStateActivateStepLocationCapabilityFailsClosedWithoutSharedStorageExposure(t *testing.T) {
	t.Parallel()

	root := writeTaskStateLocationCapabilityProposalFixture(t, missioncontrol.CapabilityOnboardingProposalStateApproved)
	job := testTaskStateJob()
	job.Plan.Steps[0].RequiredCapabilities = []string{missioncontrol.LocationCapabilityName}
	job.Plan.Steps[0].CapabilityOnboardingProposalRef = &missioncontrol.CapabilityOnboardingProposalRef{
		ProposalID: "proposal-location",
	}

	state := NewTaskState()
	state.SetMissionStoreRoot(root)
	state.locationCapabilityHook = func(root string, ec missioncontrol.ExecutionContext, now time.Time) error {
		t.Fatal("locationCapabilityHook() called without shared_storage exposure")
		return nil
	}

	err := state.ActivateStep(job, "build")
	if err == nil {
		t.Fatal("ActivateStep() error = nil, want shared_storage rejection")
	}
	if !strings.Contains(err.Error(), `location capability requires shared_storage exposure: shared_storage capability requires one committed capability record named "shared_storage"`) {
		t.Fatalf("ActivateStep() error = %q, want shared_storage rejection", err)
	}
}

func TestTaskStateOperatorInspectUsesPersistedInspectablePlanWithoutMissionJob(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	job := testTaskStateJob()
	inspectablePlan, err := missioncontrol.BuildInspectablePlanContext(job)
	if err != nil {
		t.Fatalf("BuildInspectablePlanContext() error = %v", err)
	}
	control, err := missioncontrol.BuildRuntimeControlContext(job, "build")
	if err != nil {
		t.Fatalf("BuildRuntimeControlContext() error = %v", err)
	}

	state.runtimeState = missioncontrol.JobRuntimeState{
		JobID:           "job-1",
		State:           missioncontrol.JobStatePaused,
		ActiveStepID:    "build",
		InspectablePlan: &inspectablePlan,
		PausedReason:    missioncontrol.RuntimePauseReasonOperatorCommand,
	}
	state.hasRuntimeState = true
	state.runtimeControl = control
	state.hasRuntimeControl = true

	got, err := state.OperatorInspect("job-1", "final")
	if err != nil {
		t.Fatalf("OperatorInspect() error = %v", err)
	}

	var summary missioncontrol.InspectSummary
	if err := json.Unmarshal([]byte(got), &summary); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if summary.JobID != "job-1" {
		t.Fatalf("JobID = %q, want %q", summary.JobID, "job-1")
	}
	if len(summary.Steps) != 1 || summary.Steps[0].StepID != "final" {
		t.Fatalf("Steps = %#v, want one final step", summary.Steps)
	}
	if !reflect.DeepEqual(summary.Steps[0].EffectiveAllowedTools, []string{"read"}) {
		t.Fatalf("EffectiveAllowedTools = %#v, want %#v", summary.Steps[0].EffectiveAllowedTools, []string{"read"})
	}
}

func TestTaskStateOperatorInspectPersistedInspectablePlanPathUnchangedForTreasurySteps(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	job := testTaskStateJob()
	job.Plan.Steps[0].TreasuryRef = &missioncontrol.TreasuryRef{TreasuryID: "treasury-wallet"}
	inspectablePlan, err := missioncontrol.BuildInspectablePlanContext(job)
	if err != nil {
		t.Fatalf("BuildInspectablePlanContext() error = %v", err)
	}
	control, err := missioncontrol.BuildRuntimeControlContext(job, "build")
	if err != nil {
		t.Fatalf("BuildRuntimeControlContext() error = %v", err)
	}

	state.runtimeState = missioncontrol.JobRuntimeState{
		JobID:           "job-1",
		State:           missioncontrol.JobStatePaused,
		ActiveStepID:    "build",
		InspectablePlan: &inspectablePlan,
		PausedReason:    missioncontrol.RuntimePauseReasonOperatorCommand,
	}
	state.hasRuntimeState = true
	state.runtimeControl = control
	state.hasRuntimeControl = true

	got, err := state.OperatorInspect("job-1", "build")
	if err != nil {
		t.Fatalf("OperatorInspect() error = %v", err)
	}

	var summary missioncontrol.InspectSummary
	if err := json.Unmarshal([]byte(got), &summary); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if len(summary.Steps) != 1 || summary.Steps[0].StepID != "build" {
		t.Fatalf("Steps = %#v, want one build step", summary.Steps)
	}
	if summary.Steps[0].TreasuryPreflight != nil {
		t.Fatalf("TreasuryPreflight = %#v, want nil for persisted inspectable-plan path", summary.Steps[0].TreasuryPreflight)
	}
}

func TestTaskStateOperatorInspectPersistedInspectablePlanWrongJobDoesNotBind(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	inspectablePlan, err := missioncontrol.BuildInspectablePlanContext(testTaskStateJob())
	if err != nil {
		t.Fatalf("BuildInspectablePlanContext() error = %v", err)
	}

	state.runtimeState = missioncontrol.JobRuntimeState{
		JobID:           "job-1",
		State:           missioncontrol.JobStatePaused,
		ActiveStepID:    "build",
		InspectablePlan: &inspectablePlan,
		PausedReason:    missioncontrol.RuntimePauseReasonOperatorCommand,
	}
	state.hasRuntimeState = true

	_, err = state.OperatorInspect("other-job", "final")
	if err == nil {
		t.Fatal("OperatorInspect() error = nil, want mismatch failure")
	}
	if !strings.Contains(err.Error(), "does not match the active job") {
		t.Fatalf("OperatorInspect() error = %q, want job mismatch", err)
	}
}

func TestTaskStateOperatorInspectPersistedInspectablePlanRejectsInvalidStep(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	inspectablePlan, err := missioncontrol.BuildInspectablePlanContext(testTaskStateJob())
	if err != nil {
		t.Fatalf("BuildInspectablePlanContext() error = %v", err)
	}

	state.runtimeState = missioncontrol.JobRuntimeState{
		JobID:           "job-1",
		State:           missioncontrol.JobStatePaused,
		ActiveStepID:    "build",
		InspectablePlan: &inspectablePlan,
		PausedReason:    missioncontrol.RuntimePauseReasonOperatorCommand,
	}
	state.hasRuntimeState = true

	_, err = state.OperatorInspect("job-1", "missing")
	if err == nil {
		t.Fatal("OperatorInspect() error = nil, want unknown-step failure")
	}
	if !strings.Contains(err.Error(), string(missioncontrol.RejectionCodeUnknownStep)) {
		t.Fatalf("OperatorInspect() error = %q, want unknown_step code", err)
	}
	if !strings.Contains(err.Error(), `step "missing" not found in plan`) {
		t.Fatalf("OperatorInspect() error = %q, want missing-step message", err)
	}
}

func TestTaskStateOperatorInspectTerminalRuntimeUsesPersistedInspectablePlanWithoutMissionJob(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	inspectablePlan, err := missioncontrol.BuildInspectablePlanContext(testTaskStateJob())
	if err != nil {
		t.Fatalf("BuildInspectablePlanContext() error = %v", err)
	}

	state.runtimeState = missioncontrol.JobRuntimeState{
		JobID:           "job-1",
		State:           missioncontrol.JobStateCompleted,
		InspectablePlan: &inspectablePlan,
		CompletedAt:     time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC),
	}
	state.hasRuntimeState = true

	got, err := state.OperatorInspect("job-1", "final")
	if err != nil {
		t.Fatalf("OperatorInspect() error = %v", err)
	}

	var summary missioncontrol.InspectSummary
	if err := json.Unmarshal([]byte(got), &summary); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if len(summary.Steps) != 1 || summary.Steps[0].StepID != "final" {
		t.Fatalf("Steps = %#v, want one final step", summary.Steps)
	}
}

func writeTaskStateNotificationsCapabilityProposalFixture(t *testing.T, state missioncontrol.CapabilityOnboardingProposalState) string {
	t.Helper()

	root := t.TempDir()
	now := time.Date(2026, 4, 18, 16, 0, 0, 0, time.UTC)
	record := missioncontrol.CapabilityOnboardingProposalRecord{
		ProposalID:       "proposal-notifications",
		CapabilityName:   missioncontrol.NotificationsCapabilityName,
		WhyNeeded:        "mission requires operator-facing notifications",
		MissionFamilies:  []string{"outreach"},
		Risks:            []string{"notification spam"},
		Validators:       []string{"telegram owner-control channel confirmed"},
		KillSwitch:       "disable telegram channel and revoke proposal",
		DataAccessed:     []string{"notifications"},
		ApprovalRequired: true,
		CreatedAt:        now,
		State:            state,
	}
	if err := missioncontrol.StoreCapabilityOnboardingProposalRecord(root, record); err != nil {
		t.Fatalf("StoreCapabilityOnboardingProposalRecord() error = %v", err)
	}
	return root
}

func writeTaskStateSharedStorageCapabilityProposalFixture(t *testing.T, state missioncontrol.CapabilityOnboardingProposalState) string {
	t.Helper()

	root := t.TempDir()
	now := time.Date(2026, 4, 18, 17, 0, 0, 0, time.UTC)
	record := missioncontrol.CapabilityOnboardingProposalRecord{
		ProposalID:       "proposal-shared-storage",
		CapabilityName:   missioncontrol.SharedStorageCapabilityName,
		WhyNeeded:        "mission requires shared workspace storage",
		MissionFamilies:  []string{"workspace"},
		Risks:            []string{"workspace data exposure"},
		Validators:       []string{"configured workspace root initialized and writable"},
		KillSwitch:       "disable workspace-backed shared_storage exposure and revoke proposal",
		DataAccessed:     []string{"shared storage"},
		ApprovalRequired: true,
		CreatedAt:        now,
		State:            state,
	}
	if err := missioncontrol.StoreCapabilityOnboardingProposalRecord(root, record); err != nil {
		t.Fatalf("StoreCapabilityOnboardingProposalRecord() error = %v", err)
	}
	return root
}

func writeTaskStateContactsCapabilityProposalFixture(t *testing.T, state missioncontrol.CapabilityOnboardingProposalState) string {
	t.Helper()

	root := t.TempDir()
	now := time.Date(2026, 4, 18, 22, 0, 0, 0, time.UTC)
	record := missioncontrol.CapabilityOnboardingProposalRecord{
		ProposalID:       "proposal-contacts",
		CapabilityName:   missioncontrol.ContactsCapabilityName,
		WhyNeeded:        "mission requires local shared contacts access",
		MissionFamilies:  []string{"workspace"},
		Risks:            []string{"local contacts exposure"},
		Validators:       []string{"shared_storage exposed and committed contacts source file exists and is readable"},
		KillSwitch:       "disable contacts capability exposure and remove committed contacts source reference",
		DataAccessed:     []string{"contacts"},
		ApprovalRequired: true,
		CreatedAt:        now,
		State:            state,
	}
	if err := missioncontrol.StoreCapabilityOnboardingProposalRecord(root, record); err != nil {
		t.Fatalf("StoreCapabilityOnboardingProposalRecord() error = %v", err)
	}
	return root
}

func writeTaskStateContactsCapabilityConfigFixture(t *testing.T) string {
	t.Helper()

	home := t.TempDir()
	configDir := filepath.Join(home, ".picobot")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	workspace := filepath.Join(home, "workspace-root")
	configPath := filepath.Join(configDir, "config.json")
	configJSON := fmt.Sprintf(`{"agents":{"defaults":{"workspace":%q}}}`, workspace)
	if err := os.WriteFile(configPath, []byte(configJSON), 0o644); err != nil {
		t.Fatalf("WriteFile(config.json) error = %v", err)
	}
	t.Setenv("HOME", home)
	return workspace
}

func writeTaskStateLocationCapabilityProposalFixture(t *testing.T, state missioncontrol.CapabilityOnboardingProposalState) string {
	t.Helper()

	root := t.TempDir()
	now := time.Date(2026, 4, 19, 1, 0, 0, 0, time.UTC)
	record := missioncontrol.CapabilityOnboardingProposalRecord{
		ProposalID:       "proposal-location",
		CapabilityName:   missioncontrol.LocationCapabilityName,
		WhyNeeded:        "mission requires local shared location access",
		MissionFamilies:  []string{"workspace"},
		Risks:            []string{"local location exposure"},
		Validators:       []string{"shared_storage exposed and committed location source file exists and is readable"},
		KillSwitch:       "disable location capability exposure and remove committed location source reference",
		DataAccessed:     []string{"location"},
		ApprovalRequired: true,
		CreatedAt:        now,
		State:            state,
	}
	if err := missioncontrol.StoreCapabilityOnboardingProposalRecord(root, record); err != nil {
		t.Fatalf("StoreCapabilityOnboardingProposalRecord() error = %v", err)
	}
	return root
}

func writeTaskStateLocationCapabilityConfigFixture(t *testing.T) string {
	t.Helper()

	home := t.TempDir()
	configDir := filepath.Join(home, ".picobot")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	workspace := filepath.Join(home, "workspace-root")
	configPath := filepath.Join(configDir, "config.json")
	configJSON := fmt.Sprintf(`{"agents":{"defaults":{"workspace":%q}}}`, workspace)
	if err := os.WriteFile(configPath, []byte(configJSON), 0o644); err != nil {
		t.Fatalf("WriteFile(config.json) error = %v", err)
	}
	t.Setenv("HOME", home)
	return workspace
}

func testTaskStateJob() missioncontrol.Job {
	return missioncontrol.Job{
		ID:           "job-1",
		MaxAuthority: missioncontrol.AuthorityTierHigh,
		AllowedTools: []string{"read", "write"},
		Plan: missioncontrol.Plan{
			ID: "plan-1",
			Steps: []missioncontrol.Step{
				{
					ID:                "build",
					Type:              missioncontrol.StepTypeOneShotCode,
					RequiredAuthority: missioncontrol.AuthorityTierMedium,
					AllowedTools:      []string{"read"},
					SuccessCriteria:   []string{"produce code"},
				},
				{
					ID:           "final",
					Type:         missioncontrol.StepTypeFinalResponse,
					DependsOn:    []string{"build"},
					AllowedTools: []string{"read"},
				},
			},
		},
	}
}

func testReusableApprovalJob(scope string) missioncontrol.Job {
	return missioncontrol.Job{
		ID:           "job-1",
		MaxAuthority: missioncontrol.AuthorityTierHigh,
		AllowedTools: []string{"read"},
		Plan: missioncontrol.Plan{
			ID: "plan-1",
			Steps: []missioncontrol.Step{
				{
					ID:            "authorize-1",
					Type:          missioncontrol.StepTypeDiscussion,
					Subtype:       missioncontrol.StepSubtypeAuthorization,
					ApprovalScope: scope,
				},
				{
					ID:            "authorize-2",
					Type:          missioncontrol.StepTypeDiscussion,
					Subtype:       missioncontrol.StepSubtypeAuthorization,
					ApprovalScope: scope,
					DependsOn:     []string{"authorize-1"},
				},
				{
					ID:        "final",
					Type:      missioncontrol.StepTypeFinalResponse,
					DependsOn: []string{"authorize-2"},
				},
			},
		},
	}
}

func writeTaskStateAutonomyEligibilityFixture(t *testing.T, root string, target missioncontrol.AutonomyEligibilityTargetRef, record missioncontrol.PlatformRecord, check missioncontrol.EligibilityCheckRecord) {
	t.Helper()

	if err := missioncontrol.StorePlatformRecord(root, record); err != nil {
		t.Fatalf("StorePlatformRecord(%s) error = %v", target.RegistryID, err)
	}
	if err := missioncontrol.StoreEligibilityCheckRecord(root, check); err != nil {
		t.Fatalf("StoreEligibilityCheckRecord(%s) error = %v", check.CheckID, err)
	}
}

func writeTaskStateTreasuryFixtures(t *testing.T) (string, missioncontrol.TreasuryRecord, missioncontrol.FrankContainerRecord) {
	t.Helper()

	root := t.TempDir()
	now := time.Date(2026, 4, 8, 21, 0, 0, 0, time.UTC)
	target := missioncontrol.AutonomyEligibilityTargetRef{
		Kind:       missioncontrol.EligibilityTargetKindTreasuryContainerClass,
		RegistryID: "container-class-wallet",
	}
	writeTaskStateAutonomyEligibilityFixture(t, root, target, missioncontrol.PlatformRecord{
		PlatformID:       target.RegistryID,
		PlatformName:     "container-class-wallet",
		TargetClass:      target.Kind,
		EligibilityLabel: missioncontrol.EligibilityLabelAutonomyCompatible,
		LastCheckID:      "check-container-class-wallet",
		Notes:            []string{"registry note"},
		UpdatedAt:        now,
	}, missioncontrol.EligibilityCheckRecord{
		CheckID:                "check-container-class-wallet",
		TargetKind:             target.Kind,
		TargetName:             "container-class-wallet",
		CanCreateWithoutOwner:  true,
		CanOnboardWithoutOwner: true,
		CanControlAsAgent:      true,
		CanRecoverAsAgent:      true,
		RulesAsObservedOK:      true,
		Label:                  missioncontrol.EligibilityLabelAutonomyCompatible,
		Reasons:                []string{"autonomy_compatible"},
		CheckedAt:              now,
	})

	container := missioncontrol.FrankContainerRecord{
		RecordVersion:        missioncontrol.StoreRecordVersion,
		ContainerID:          "container-wallet",
		ContainerKind:        "wallet",
		Label:                "Primary Wallet",
		ContainerClassID:     "container-class-wallet",
		State:                "active",
		EligibilityTargetRef: target,
		CreatedAt:            now.Add(time.Minute).UTC(),
		UpdatedAt:            now.Add(2 * time.Minute).UTC(),
	}
	if err := missioncontrol.StoreFrankContainerRecord(root, container); err != nil {
		t.Fatalf("StoreFrankContainerRecord() error = %v", err)
	}

	treasury := missioncontrol.TreasuryRecord{
		RecordVersion:  missioncontrol.StoreRecordVersion,
		TreasuryID:     "treasury-wallet",
		DisplayName:    "Frank Treasury",
		State:          missioncontrol.TreasuryStateFunded,
		ZeroSeedPolicy: missioncontrol.TreasuryZeroSeedPolicyOwnerSeedForbidden,
		ContainerRefs: []missioncontrol.FrankRegistryObjectRef{
			{
				Kind:     missioncontrol.FrankRegistryObjectKindContainer,
				ObjectID: container.ContainerID,
			},
		},
		CreatedAt: now.UTC(),
		UpdatedAt: now.Add(3 * time.Minute).UTC(),
	}
	if err := missioncontrol.StoreTreasuryRecord(root, treasury); err != nil {
		t.Fatalf("StoreTreasuryRecord() error = %v", err)
	}

	return root, treasury, container
}

func writeTaskStateTreasuryBootstrapFixtures(t *testing.T) (string, missioncontrol.TreasuryRecord, missioncontrol.FrankContainerRecord) {
	t.Helper()

	root, treasury, container := writeTaskStateTreasuryFixtures(t)
	treasury.State = missioncontrol.TreasuryStateBootstrap
	treasury.BootstrapAcquisition = &missioncontrol.TreasuryBootstrapAcquisition{
		EntryID:         "entry-first-value",
		AssetCode:       "USD",
		Amount:          "10.00",
		SourceRef:       "payout:listing-1",
		EvidenceLocator: "https://evidence.example/payout-1",
		ConfirmedAt:     treasury.UpdatedAt.Add(time.Minute),
	}
	treasury.UpdatedAt = treasury.UpdatedAt.Add(2 * time.Minute)
	if err := missioncontrol.StoreTreasuryRecord(root, treasury); err != nil {
		t.Fatalf("StoreTreasuryRecord() error = %v", err)
	}
	return root, treasury, container
}

func writeTaskStateActiveTreasuryAcquisitionFixtures(t *testing.T) (string, missioncontrol.TreasuryRecord, missioncontrol.FrankContainerRecord) {
	t.Helper()

	root, treasury, container := writeTaskStateTreasuryFixtures(t)
	treasury.State = missioncontrol.TreasuryStateActive
	treasury.PostBootstrapAcquisition = &missioncontrol.TreasuryPostBootstrapAcquisition{
		AssetCode:       "USD",
		Amount:          "2.25",
		SourceRef:       "payout:listing-2",
		EvidenceLocator: "https://evidence.example/payout-2",
		ConfirmedAt:     treasury.UpdatedAt.Add(time.Minute),
	}
	treasury.UpdatedAt = treasury.UpdatedAt.Add(2 * time.Minute)
	if err := missioncontrol.StoreTreasuryRecord(root, treasury); err != nil {
		t.Fatalf("StoreTreasuryRecord() error = %v", err)
	}
	return root, treasury, container
}

func writeTaskStateActiveTreasurySuspendFixtures(t *testing.T) (string, missioncontrol.TreasuryRecord) {
	t.Helper()

	root, treasury, _ := writeTaskStateTreasuryFixtures(t)

	treasury.State = missioncontrol.TreasuryStateActive
	treasury.PostActiveSuspend = &missioncontrol.TreasuryPostActiveSuspend{
		Reason:    "risk:manual-review-required",
		SourceRef: "suspend:risk-review-a",
	}
	treasury.UpdatedAt = treasury.UpdatedAt.Add(3 * time.Minute)
	if err := missioncontrol.StoreTreasuryRecord(root, treasury); err != nil {
		t.Fatalf("StoreTreasuryRecord() error = %v", err)
	}
	return root, treasury
}

func writeTaskStateSuspendedTreasuryResumeFixtures(t *testing.T) (string, missioncontrol.TreasuryRecord) {
	t.Helper()

	root, treasury, _ := writeTaskStateTreasuryFixtures(t)

	treasury.State = missioncontrol.TreasuryStateSuspended
	treasury.PostActiveSuspend = &missioncontrol.TreasuryPostActiveSuspend{
		Reason:               "risk:manual-review-required",
		SourceRef:            "suspend:risk-review-a",
		ConsumedTransitionID: "transition-suspend-a",
	}
	treasury.PostSuspendResume = &missioncontrol.TreasuryPostSuspendResume{
		Reason:    "ops:manual-clear",
		SourceRef: "resume:manual-clear-a",
	}
	treasury.UpdatedAt = treasury.UpdatedAt.Add(4 * time.Minute)
	if err := missioncontrol.StoreTreasuryRecord(root, treasury); err != nil {
		t.Fatalf("StoreTreasuryRecord() error = %v", err)
	}
	return root, treasury
}

func writeTaskStateActiveTreasuryAllocateFixtures(t *testing.T) (string, missioncontrol.TreasuryRecord, missioncontrol.FrankContainerRecord) {
	t.Helper()

	root, treasury, container := writeTaskStateTreasuryFixtures(t)

	treasury.State = missioncontrol.TreasuryStateActive
	treasury.PostActiveAllocate = &missioncontrol.TreasuryPostActiveAllocate{
		AssetCode: "USD",
		Amount:    "1.10",
		SourceContainerRef: missioncontrol.FrankRegistryObjectRef{
			Kind:     missioncontrol.FrankRegistryObjectKindContainer,
			ObjectID: container.ContainerID,
		},
		AllocationTargetRef: "allocation:ops-reserve",
		SourceRef:           "allocate:ops-reserve-a",
	}
	treasury.UpdatedAt = treasury.UpdatedAt.Add(3 * time.Minute)
	if err := missioncontrol.StoreTreasuryRecord(root, treasury); err != nil {
		t.Fatalf("StoreTreasuryRecord() error = %v", err)
	}
	return root, treasury, container
}

func writeTaskStateActiveTreasuryReinvestFixtures(t *testing.T) (string, missioncontrol.TreasuryRecord, missioncontrol.FrankContainerRecord, missioncontrol.FrankContainerRecord) {
	t.Helper()

	root, treasury, container := writeTaskStateTreasuryFixtures(t)
	now := treasury.UpdatedAt
	target := missioncontrol.AutonomyEligibilityTargetRef{
		Kind:       missioncontrol.EligibilityTargetKindTreasuryContainerClass,
		RegistryID: "container-class-investment",
	}
	writeTaskStateAutonomyEligibilityFixture(t, root, target, missioncontrol.PlatformRecord{
		PlatformID:       target.RegistryID,
		PlatformName:     "container-class-investment",
		TargetClass:      target.Kind,
		EligibilityLabel: missioncontrol.EligibilityLabelAutonomyCompatible,
		LastCheckID:      "check-container-class-investment",
		Notes:            []string{"registry note"},
		UpdatedAt:        now,
	}, missioncontrol.EligibilityCheckRecord{
		CheckID:                "check-container-class-investment",
		TargetKind:             target.Kind,
		TargetName:             "container-class-investment",
		CanCreateWithoutOwner:  true,
		CanOnboardWithoutOwner: true,
		CanControlAsAgent:      true,
		CanRecoverAsAgent:      true,
		RulesAsObservedOK:      true,
		Label:                  missioncontrol.EligibilityLabelAutonomyCompatible,
		Reasons:                []string{"autonomy_compatible"},
		CheckedAt:              now,
	})
	targetContainer := missioncontrol.FrankContainerRecord{
		RecordVersion:        missioncontrol.StoreRecordVersion,
		ContainerID:          "container-investment",
		ContainerKind:        "wallet",
		Label:                "Investment Wallet",
		ContainerClassID:     target.RegistryID,
		State:                "active",
		EligibilityTargetRef: target,
		CreatedAt:            now.Add(time.Minute).UTC(),
		UpdatedAt:            now.Add(2 * time.Minute).UTC(),
	}
	if err := missioncontrol.StoreFrankContainerRecord(root, targetContainer); err != nil {
		t.Fatalf("StoreFrankContainerRecord() error = %v", err)
	}

	treasury.State = missioncontrol.TreasuryStateActive
	treasury.PostActiveReinvest = &missioncontrol.TreasuryPostActiveReinvest{
		SourceAssetCode: "USD",
		SourceAmount:    "0.75",
		TargetAssetCode: "BTC",
		TargetAmount:    "0.00001000",
		SourceContainerRef: missioncontrol.FrankRegistryObjectRef{
			Kind:     missioncontrol.FrankRegistryObjectKindContainer,
			ObjectID: container.ContainerID,
		},
		TargetContainerRef: missioncontrol.FrankRegistryObjectRef{
			Kind:     missioncontrol.FrankRegistryObjectKindContainer,
			ObjectID: targetContainer.ContainerID,
		},
		SourceRef:       "trade:reinvest-a",
		EvidenceLocator: "https://evidence.example/reinvest-a",
		ConfirmedAt:     now.Add(90 * time.Second),
	}
	treasury.UpdatedAt = treasury.UpdatedAt.Add(3 * time.Minute)
	if err := missioncontrol.StoreTreasuryRecord(root, treasury); err != nil {
		t.Fatalf("StoreTreasuryRecord() error = %v", err)
	}
	return root, treasury, container, targetContainer
}

func writeTaskStateActiveTreasurySpendFixtures(t *testing.T) (string, missioncontrol.TreasuryRecord, missioncontrol.FrankContainerRecord) {
	t.Helper()

	root, treasury, container := writeTaskStateTreasuryFixtures(t)

	treasury.State = missioncontrol.TreasuryStateActive
	treasury.PostActiveSpend = &missioncontrol.TreasuryPostActiveSpend{
		AssetCode: "USD",
		Amount:    "0.75",
		SourceContainerRef: missioncontrol.FrankRegistryObjectRef{
			Kind:     missioncontrol.FrankRegistryObjectKindContainer,
			ObjectID: container.ContainerID,
		},
		TargetRef:       "vendor:domain-renewal",
		SourceRef:       "spend:domain-renewal-a",
		EvidenceLocator: "https://evidence.example/spend-a",
	}
	treasury.UpdatedAt = treasury.UpdatedAt.Add(3 * time.Minute)
	if err := missioncontrol.StoreTreasuryRecord(root, treasury); err != nil {
		t.Fatalf("StoreTreasuryRecord() error = %v", err)
	}
	return root, treasury, container
}

func writeTaskStateActiveTreasuryTransferFixtures(t *testing.T) (string, missioncontrol.TreasuryRecord, missioncontrol.FrankContainerRecord, missioncontrol.FrankContainerRecord) {
	t.Helper()

	root, treasury, container := writeTaskStateTreasuryFixtures(t)
	now := treasury.UpdatedAt
	target := missioncontrol.AutonomyEligibilityTargetRef{
		Kind:       missioncontrol.EligibilityTargetKindTreasuryContainerClass,
		RegistryID: "container-class-vault",
	}
	writeTaskStateAutonomyEligibilityFixture(t, root, target, missioncontrol.PlatformRecord{
		PlatformID:       target.RegistryID,
		PlatformName:     "container-class-vault",
		TargetClass:      target.Kind,
		EligibilityLabel: missioncontrol.EligibilityLabelAutonomyCompatible,
		LastCheckID:      "check-container-class-vault",
		Notes:            []string{"registry note"},
		UpdatedAt:        now,
	}, missioncontrol.EligibilityCheckRecord{
		CheckID:                "check-container-class-vault",
		TargetKind:             target.Kind,
		TargetName:             "container-class-vault",
		CanCreateWithoutOwner:  true,
		CanOnboardWithoutOwner: true,
		CanControlAsAgent:      true,
		CanRecoverAsAgent:      true,
		RulesAsObservedOK:      true,
		Label:                  missioncontrol.EligibilityLabelAutonomyCompatible,
		Reasons:                []string{"autonomy_compatible"},
		CheckedAt:              now,
	})
	targetContainer := missioncontrol.FrankContainerRecord{
		RecordVersion:        missioncontrol.StoreRecordVersion,
		ContainerID:          "container-vault",
		ContainerKind:        "wallet",
		Label:                "Vault Wallet",
		ContainerClassID:     target.RegistryID,
		State:                "active",
		EligibilityTargetRef: target,
		CreatedAt:            now.Add(time.Minute).UTC(),
		UpdatedAt:            now.Add(2 * time.Minute).UTC(),
	}
	if err := missioncontrol.StoreFrankContainerRecord(root, targetContainer); err != nil {
		t.Fatalf("StoreFrankContainerRecord() error = %v", err)
	}

	treasury.State = missioncontrol.TreasuryStateActive
	treasury.PostActiveTransfer = &missioncontrol.TreasuryPostActiveTransfer{
		AssetCode: "USD",
		Amount:    "1.25",
		SourceContainerRef: missioncontrol.FrankRegistryObjectRef{
			Kind:     missioncontrol.FrankRegistryObjectKindContainer,
			ObjectID: container.ContainerID,
		},
		TargetContainerRef: missioncontrol.FrankRegistryObjectRef{
			Kind:     missioncontrol.FrankRegistryObjectKindContainer,
			ObjectID: targetContainer.ContainerID,
		},
		SourceRef:       "transfer:rebalance-a",
		EvidenceLocator: "https://evidence.example/transfer-a",
	}
	treasury.UpdatedAt = treasury.UpdatedAt.Add(3 * time.Minute)
	if err := missioncontrol.StoreTreasuryRecord(root, treasury); err != nil {
		t.Fatalf("StoreTreasuryRecord() error = %v", err)
	}
	return root, treasury, container, targetContainer
}

func writeTaskStateActiveTreasurySaveFixtures(t *testing.T) (string, missioncontrol.TreasuryRecord, missioncontrol.FrankContainerRecord, missioncontrol.FrankContainerRecord) {
	t.Helper()

	root, treasury, container := writeTaskStateTreasuryFixtures(t)
	now := treasury.UpdatedAt
	target := missioncontrol.AutonomyEligibilityTargetRef{
		Kind:       missioncontrol.EligibilityTargetKindTreasuryContainerClass,
		RegistryID: "container-class-savings",
	}
	writeTaskStateAutonomyEligibilityFixture(t, root, target, missioncontrol.PlatformRecord{
		PlatformID:       target.RegistryID,
		PlatformName:     "container-class-savings",
		TargetClass:      target.Kind,
		EligibilityLabel: missioncontrol.EligibilityLabelAutonomyCompatible,
		LastCheckID:      "check-container-class-savings",
		Notes:            []string{"registry note"},
		UpdatedAt:        now,
	}, missioncontrol.EligibilityCheckRecord{
		CheckID:                "check-container-class-savings",
		TargetKind:             target.Kind,
		TargetName:             "container-class-savings",
		CanCreateWithoutOwner:  true,
		CanOnboardWithoutOwner: true,
		CanControlAsAgent:      true,
		CanRecoverAsAgent:      true,
		RulesAsObservedOK:      true,
		Label:                  missioncontrol.EligibilityLabelAutonomyCompatible,
		Reasons:                []string{"autonomy_compatible"},
		CheckedAt:              now,
	})
	targetContainer := missioncontrol.FrankContainerRecord{
		RecordVersion:        missioncontrol.StoreRecordVersion,
		ContainerID:          "container-savings",
		ContainerKind:        "wallet",
		Label:                "Savings Wallet",
		ContainerClassID:     target.RegistryID,
		State:                "active",
		EligibilityTargetRef: target,
		CreatedAt:            now.Add(time.Minute).UTC(),
		UpdatedAt:            now.Add(2 * time.Minute).UTC(),
	}
	if err := missioncontrol.StoreFrankContainerRecord(root, targetContainer); err != nil {
		t.Fatalf("StoreFrankContainerRecord() error = %v", err)
	}

	treasury.State = missioncontrol.TreasuryStateActive
	treasury.PostActiveSave = &missioncontrol.TreasuryPostActiveSave{
		AssetCode:         "USD",
		Amount:            "1.25",
		TargetContainerID: targetContainer.ContainerID,
		SourceRef:         "transfer:reserve-a",
		EvidenceLocator:   "https://evidence.example/save-a",
	}
	treasury.UpdatedAt = treasury.UpdatedAt.Add(3 * time.Minute)
	if err := missioncontrol.StoreTreasuryRecord(root, treasury); err != nil {
		t.Fatalf("StoreTreasuryRecord() error = %v", err)
	}
	return root, treasury, container, targetContainer
}

func writeTaskStateZohoMailboxBootstrapFixtures(t *testing.T) (string, missioncontrol.FrankIdentityRecord, missioncontrol.FrankAccountRecord) {
	t.Helper()

	root := t.TempDir()
	now := time.Date(2026, 4, 16, 15, 0, 0, 0, time.UTC)
	providerTarget := missioncontrol.AutonomyEligibilityTargetRef{
		Kind:       missioncontrol.EligibilityTargetKindProvider,
		RegistryID: "provider-mail",
	}
	accountTarget := missioncontrol.AutonomyEligibilityTargetRef{
		Kind:       missioncontrol.EligibilityTargetKindAccountClass,
		RegistryID: "account-class-mailbox",
	}

	writeTaskStateAutonomyEligibilityFixture(t, root, providerTarget, missioncontrol.PlatformRecord{
		PlatformID:       providerTarget.RegistryID,
		PlatformName:     "provider-mail.example",
		TargetClass:      providerTarget.Kind,
		EligibilityLabel: missioncontrol.EligibilityLabelAutonomyCompatible,
		LastCheckID:      "check-provider-mail",
		Notes:            []string{"registry note"},
		UpdatedAt:        now,
	}, missioncontrol.EligibilityCheckRecord{
		CheckID:                "check-provider-mail",
		TargetKind:             providerTarget.Kind,
		TargetName:             "provider-mail.example",
		CanCreateWithoutOwner:  true,
		CanOnboardWithoutOwner: true,
		CanControlAsAgent:      true,
		CanRecoverAsAgent:      true,
		RulesAsObservedOK:      true,
		Label:                  missioncontrol.EligibilityLabelAutonomyCompatible,
		Reasons:                []string{"autonomy_compatible"},
		CheckedAt:              now,
	})
	writeTaskStateAutonomyEligibilityFixture(t, root, accountTarget, missioncontrol.PlatformRecord{
		PlatformID:       accountTarget.RegistryID,
		PlatformName:     "account-class-mailbox",
		TargetClass:      accountTarget.Kind,
		EligibilityLabel: missioncontrol.EligibilityLabelAutonomyCompatible,
		LastCheckID:      "check-account-class-mailbox",
		Notes:            []string{"registry note"},
		UpdatedAt:        now.Add(time.Minute),
	}, missioncontrol.EligibilityCheckRecord{
		CheckID:                "check-account-class-mailbox",
		TargetKind:             accountTarget.Kind,
		TargetName:             "account-class-mailbox",
		CanCreateWithoutOwner:  true,
		CanOnboardWithoutOwner: true,
		CanControlAsAgent:      true,
		CanRecoverAsAgent:      true,
		RulesAsObservedOK:      true,
		Label:                  missioncontrol.EligibilityLabelAutonomyCompatible,
		Reasons:                []string{"autonomy_compatible"},
		CheckedAt:              now.Add(time.Minute),
	})

	identity := missioncontrol.FrankIdentityRecord{
		RecordVersion:        missioncontrol.StoreRecordVersion,
		IdentityID:           "identity-mail",
		IdentityKind:         "email",
		DisplayName:          "Frank Mail",
		ProviderOrPlatformID: providerTarget.RegistryID,
		ZohoMailbox: &missioncontrol.FrankZohoMailboxIdentity{
			FromAddress:     "frank@example.com",
			FromDisplayName: "Frank",
		},
		IdentityMode:         missioncontrol.IdentityModeAgentAlias,
		State:                "candidate",
		EligibilityTargetRef: providerTarget,
		CreatedAt:            now.UTC(),
		UpdatedAt:            now.Add(time.Minute).UTC(),
	}
	if err := missioncontrol.StoreFrankIdentityRecord(root, identity); err != nil {
		t.Fatalf("StoreFrankIdentityRecord() error = %v", err)
	}

	account := missioncontrol.FrankAccountRecord{
		RecordVersion:        missioncontrol.StoreRecordVersion,
		AccountID:            "account-mail",
		AccountKind:          "mailbox",
		Label:                "Inbox",
		ProviderOrPlatformID: providerTarget.RegistryID,
		ZohoMailbox: &missioncontrol.FrankZohoMailboxAccount{
			OrganizationID:             "zoid-123",
			AdminOAuthTokenEnvVarRef:   "PICOBOT_ZOHO_MAIL_ADMIN_TOKEN",
			BootstrapPasswordEnvVarRef: "PICOBOT_ZOHO_MAIL_BOOTSTRAP_PASSWORD",
			ProviderAccountID:          "3323462000000008002",
			ConfirmedCreated:           true,
		},
		IdentityID:           identity.IdentityID,
		ControlModel:         "agent_managed",
		RecoveryModel:        "agent_recoverable",
		State:                "candidate",
		EligibilityTargetRef: accountTarget,
		CreatedAt:            now.Add(2 * time.Minute).UTC(),
		UpdatedAt:            now.Add(3 * time.Minute).UTC(),
	}
	if err := missioncontrol.StoreFrankAccountRecord(root, account); err != nil {
		t.Fatalf("StoreFrankAccountRecord() error = %v", err)
	}

	return root, identity, account
}

func mustStoreTaskStateCampaignFixture(t *testing.T, root string, container missioncontrol.FrankContainerRecord) missioncontrol.CampaignRecord {
	t.Helper()

	now := time.Date(2026, 4, 8, 20, 45, 0, 0, time.UTC)
	providerTarget := missioncontrol.AutonomyEligibilityTargetRef{
		Kind:       missioncontrol.EligibilityTargetKindProvider,
		RegistryID: "provider-mail",
	}
	accountTarget := missioncontrol.AutonomyEligibilityTargetRef{
		Kind:       missioncontrol.EligibilityTargetKindAccountClass,
		RegistryID: "account-class-mailbox",
	}
	writeTaskStateAutonomyEligibilityFixture(t, root, providerTarget, missioncontrol.PlatformRecord{
		PlatformID:       providerTarget.RegistryID,
		PlatformName:     "provider-mail.example",
		TargetClass:      providerTarget.Kind,
		EligibilityLabel: missioncontrol.EligibilityLabelAutonomyCompatible,
		LastCheckID:      "check-provider-mail",
		Notes:            []string{"registry note"},
		UpdatedAt:        now,
	}, missioncontrol.EligibilityCheckRecord{
		CheckID:                "check-provider-mail",
		TargetKind:             providerTarget.Kind,
		TargetName:             "provider-mail.example",
		CanCreateWithoutOwner:  true,
		CanOnboardWithoutOwner: true,
		CanControlAsAgent:      true,
		CanRecoverAsAgent:      true,
		RulesAsObservedOK:      true,
		Label:                  missioncontrol.EligibilityLabelAutonomyCompatible,
		Reasons:                []string{"autonomy_compatible"},
		CheckedAt:              now,
	})
	writeTaskStateAutonomyEligibilityFixture(t, root, accountTarget, missioncontrol.PlatformRecord{
		PlatformID:       accountTarget.RegistryID,
		PlatformName:     "account-class-mailbox",
		TargetClass:      accountTarget.Kind,
		EligibilityLabel: missioncontrol.EligibilityLabelAutonomyCompatible,
		LastCheckID:      "check-account-class-mailbox",
		Notes:            []string{"registry note"},
		UpdatedAt:        now.Add(time.Minute),
	}, missioncontrol.EligibilityCheckRecord{
		CheckID:                "check-account-class-mailbox",
		TargetKind:             accountTarget.Kind,
		TargetName:             "account-class-mailbox",
		CanCreateWithoutOwner:  true,
		CanOnboardWithoutOwner: true,
		CanControlAsAgent:      true,
		CanRecoverAsAgent:      true,
		RulesAsObservedOK:      true,
		Label:                  missioncontrol.EligibilityLabelAutonomyCompatible,
		Reasons:                []string{"autonomy_compatible"},
		CheckedAt:              now.Add(time.Minute),
	})

	identity := missioncontrol.FrankIdentityRecord{
		RecordVersion:        missioncontrol.StoreRecordVersion,
		IdentityID:           "identity-mail",
		IdentityKind:         "email",
		DisplayName:          "Frank Mail",
		ProviderOrPlatformID: providerTarget.RegistryID,
		IdentityMode:         missioncontrol.IdentityModeAgentAlias,
		State:                "active",
		EligibilityTargetRef: providerTarget,
		CreatedAt:            now.UTC(),
		UpdatedAt:            now.Add(time.Minute).UTC(),
	}
	if err := missioncontrol.StoreFrankIdentityRecord(root, identity); err != nil {
		t.Fatalf("StoreFrankIdentityRecord() error = %v", err)
	}

	account := missioncontrol.FrankAccountRecord{
		RecordVersion:        missioncontrol.StoreRecordVersion,
		AccountID:            "account-mail",
		AccountKind:          "mailbox",
		Label:                "Inbox",
		ProviderOrPlatformID: providerTarget.RegistryID,
		IdentityID:           identity.IdentityID,
		ControlModel:         "agent_managed",
		RecoveryModel:        "agent_recoverable",
		State:                "active",
		EligibilityTargetRef: accountTarget,
		CreatedAt:            now.Add(2 * time.Minute).UTC(),
		UpdatedAt:            now.Add(3 * time.Minute).UTC(),
	}
	if err := missioncontrol.StoreFrankAccountRecord(root, account); err != nil {
		t.Fatalf("StoreFrankAccountRecord() error = %v", err)
	}

	campaign := missioncontrol.CampaignRecord{
		RecordVersion:           missioncontrol.StoreRecordVersion,
		CampaignID:              "campaign-mail",
		CampaignKind:            missioncontrol.CampaignKindOutreach,
		DisplayName:             "Frank Outreach",
		State:                   missioncontrol.CampaignStateActive,
		Objective:               "Reach aligned operators",
		GovernedExternalTargets: []missioncontrol.AutonomyEligibilityTargetRef{providerTarget},
		FrankObjectRefs: []missioncontrol.FrankRegistryObjectRef{
			{Kind: missioncontrol.FrankRegistryObjectKindIdentity, ObjectID: identity.IdentityID},
			{Kind: missioncontrol.FrankRegistryObjectKindAccount, ObjectID: account.AccountID},
			{Kind: missioncontrol.FrankRegistryObjectKindContainer, ObjectID: container.ContainerID},
		},
		IdentityMode:     missioncontrol.IdentityModeAgentAlias,
		StopConditions:   []string{"stop after 3 replies"},
		FailureThreshold: missioncontrol.CampaignFailureThreshold{Metric: "rejections", Limit: 3},
		ComplianceChecks: []string{"can-spam-reviewed"},
		CreatedAt:        now.Add(4 * time.Minute).UTC(),
		UpdatedAt:        now.Add(5 * time.Minute).UTC(),
	}
	if err := missioncontrol.StoreCampaignRecord(root, campaign); err != nil {
		t.Fatalf("StoreCampaignRecord() error = %v", err)
	}
	return campaign
}
