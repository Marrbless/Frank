package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/local/picobot/internal/agent"
	agenttools "github.com/local/picobot/internal/agent/tools"
	"github.com/local/picobot/internal/missioncontrol"
	"github.com/spf13/cobra"
)

type missionStatusSnapshot = missioncontrol.MissionStatusSnapshot

// missionStatusFrankZohoSendProofLocator is the dedicated provider-specific
// CLI output contract for `mission status --frank-zoho-send-proof`.
type missionStatusFrankZohoSendProofLocator struct {
	ProviderMessageID  string `json:"provider_message_id"`
	ProviderMailID     string `json:"provider_mail_id,omitempty"`
	MIMEMessageID      string `json:"mime_message_id,omitempty"`
	ProviderAccountID  string `json:"provider_account_id"`
	OriginalMessageURL string `json:"original_message_url"`
}

// missionStatusFrankZohoSendProofVerification is the dedicated provider-specific
// CLI output contract for `mission status --frank-zoho-verify-send-proof`.
type missionStatusFrankZohoSendProofVerification = agenttools.FrankZohoSendProofVerification

type missionStatusFrankZohoSendProofVerifier interface {
	Verify(context.Context, []missioncontrol.OperatorFrankZohoSendProofStatus) ([]missionStatusFrankZohoSendProofVerification, error)
}

type missionStatusFrankZohoSendProofVerifierFunc func(context.Context, []missioncontrol.OperatorFrankZohoSendProofStatus) ([]missionStatusFrankZohoSendProofVerification, error)

func (fn missionStatusFrankZohoSendProofVerifierFunc) Verify(ctx context.Context, proof []missioncontrol.OperatorFrankZohoSendProofStatus) ([]missionStatusFrankZohoSendProofVerification, error) {
	return fn(ctx, proof)
}

type missionStatusAssertionExpectation struct {
	JobID                  *string
	StepID                 *string
	Active                 *bool
	StepType               *string
	RequiredAuthority      *missioncontrol.AuthorityTier
	RequiresApproval       *bool
	NoTools                bool
	HasTools               []string
	ExactAllowedTools      []string
	CheckExactAllowedTools bool
}

var loadGatewayStatusObservation = missioncontrol.LoadGatewayStatusObservation
var loadGatewayStatusObservationFile = missioncontrol.LoadGatewayStatusObservationFile
var loadMissionStatusObservation = missioncontrol.LoadMissionStatusObservation
var writeMissionStatusSnapshotAtomic = missioncontrol.WriteMissionStatusSnapshotAtomic
var newFrankZohoSendProofVerifier = func() missionStatusFrankZohoSendProofVerifier {
	return agenttools.NewFrankZohoSendProofVerifier()
}

func loadMissionStatusFrankZohoSendProofFile(path string) ([]byte, error) {
	snapshot, err := loadMissionStatusObservation(path)
	if err != nil {
		return nil, err
	}

	// This dedicated helper surface consumes only committed
	// runtime_summary.frank_zoho_send_proof and does not fall back to raw runtime
	// receipts.
	proof := make([]missionStatusFrankZohoSendProofLocator, 0)
	if snapshot.RuntimeSummary != nil {
		proof = make([]missionStatusFrankZohoSendProofLocator, 0, len(snapshot.RuntimeSummary.FrankZohoSendProof))
		for _, candidate := range snapshot.RuntimeSummary.FrankZohoSendProof {
			proof = append(proof, missionStatusFrankZohoSendProofLocator{
				ProviderMessageID:  candidate.ProviderMessageID,
				ProviderMailID:     candidate.ProviderMailID,
				MIMEMessageID:      candidate.MIMEMessageID,
				ProviderAccountID:  candidate.ProviderAccountID,
				OriginalMessageURL: candidate.OriginalMessageURL,
			})
		}
	}

	data, err := json.MarshalIndent(proof, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to encode mission status Frank Zoho send proof for %q: %w", path, err)
	}
	data = append(data, '\n')
	return data, nil
}

func loadMissionStatusFrankZohoVerifiedSendProofFile(ctx context.Context, path string) ([]byte, error) {
	snapshot, err := loadMissionStatusObservation(path)
	if err != nil {
		return nil, err
	}

	// This dedicated helper surface consumes only committed
	// runtime_summary.frank_zoho_send_proof and does not fall back to raw runtime
	// receipts.
	proof := make([]missioncontrol.OperatorFrankZohoSendProofStatus, 0)
	if snapshot.RuntimeSummary != nil {
		proof = append(proof, snapshot.RuntimeSummary.FrankZohoSendProof...)
	}

	verified, err := newFrankZohoSendProofVerifier().Verify(ctx, proof)
	if err != nil {
		return nil, fmt.Errorf("failed to verify mission status Frank Zoho send proof for %q: %w", path, err)
	}

	data, err := json.MarshalIndent(verified, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to encode mission status Frank Zoho send proof verification for %q: %w", path, err)
	}
	data = append(data, '\n')
	return data, nil
}

func writeMissionStatusSnapshotFromCommand(cmd *cobra.Command, ag *agent.AgentLoop, now time.Time) error {
	statusFile, _ := cmd.Flags().GetString("mission-status-file")
	if statusFile == "" {
		return nil
	}

	storeRoot := resolveMissionStoreRoot(cmd)
	if storeRoot == "" {
		return writeMissionStatusSnapshot(statusFile, missionStatusSnapshotMissionFile(cmd), ag, now)
	}

	jobID := ""
	if runtimeState, ok := ag.MissionRuntimeState(); ok {
		jobID = runtimeState.JobID
	}
	if jobID == "" {
		if ec, ok := ag.ActiveMissionStep(); ok && ec.Job != nil {
			jobID = ec.Job.ID
		}
	}
	if jobID == "" {
		return writeMissionStatusSnapshot(statusFile, missionStatusSnapshotMissionFile(cmd), ag, now)
	}
	if _, err := missioncontrol.LoadCommittedJobRuntimeRecord(storeRoot, jobID); err != nil {
		if errors.Is(err, missioncontrol.ErrJobRuntimeRecordNotFound) {
			return writeMissionStatusSnapshot(statusFile, missionStatusSnapshotMissionFile(cmd), ag, now)
		}
		return fmt.Errorf("failed to inspect committed mission runtime for status snapshot in durable store %q: %w", storeRoot, err)
	}

	missionRequired, _ := cmd.Flags().GetBool("mission-required")
	return writeProjectedMissionStatusSnapshot(statusFile, missionStatusSnapshotMissionFile(cmd), storeRoot, missionRequired, jobID, now)
}

func missionStatusSnapshotMissionFile(cmd *cobra.Command) string {
	missionFile, _ := cmd.Flags().GetString("mission-file")
	return missionFile
}

func waitForMissionStatusStepConfirmation(path string, stepID string, expectedJobID string, expected *missionStatusAssertionExpectation, previousUpdatedAt string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	var lastErr error

	for {
		snapshot, err := loadMissionStatusSnapshot(path)
		if err != nil {
			lastErr = err
		} else if expected != nil {
			if err := checkMissionStatusAssertion(path, snapshot, *expected); err != nil {
				lastErr = err
			} else if previousUpdatedAt == "" || snapshot.UpdatedAt != previousUpdatedAt {
				return nil
			} else if expected.JobID != nil {
				lastErr = fmt.Errorf("mission status file %q has active=true step_id=%q job_id=%q updated_at=%q, want a fresh matching update with job_id=%q and updated_at different from %q", path, snapshot.StepID, snapshot.JobID, snapshot.UpdatedAt, *expected.JobID, previousUpdatedAt)
			} else {
				lastErr = fmt.Errorf("mission status file %q has active=true step_id=%q updated_at=%q, want a fresh matching update with updated_at different from %q", path, snapshot.StepID, snapshot.UpdatedAt, previousUpdatedAt)
			}
		} else if !snapshot.Active || snapshot.StepID != stepID {
			lastErr = fmt.Errorf("mission status file %q has active=%t step_id=%q, want active=true step_id=%q", path, snapshot.Active, snapshot.StepID, stepID)
		} else if expectedJobID != "" && snapshot.JobID != expectedJobID {
			lastErr = fmt.Errorf("mission status file %q has active=true step_id=%q job_id=%q, want active=true step_id=%q job_id=%q", path, snapshot.StepID, snapshot.JobID, stepID, expectedJobID)
		} else if previousUpdatedAt == "" || snapshot.UpdatedAt != previousUpdatedAt {
			return nil
		} else {
			if expectedJobID != "" {
				lastErr = fmt.Errorf("mission status file %q has active=true step_id=%q job_id=%q updated_at=%q, want a fresh matching update with job_id=%q and updated_at different from %q", path, snapshot.StepID, snapshot.JobID, snapshot.UpdatedAt, expectedJobID, previousUpdatedAt)
			} else {
				lastErr = fmt.Errorf("mission status file %q has active=true step_id=%q updated_at=%q, want a fresh matching update with updated_at different from %q", path, snapshot.StepID, snapshot.UpdatedAt, previousUpdatedAt)
			}
		}

		remaining := time.Until(deadline)
		if remaining <= 0 {
			return fmt.Errorf("timed out waiting up to %s for mission status file %q to confirm step %q: %w", timeout, path, stepID, lastErr)
		}
		sleep := 100 * time.Millisecond
		if remaining < sleep {
			sleep = remaining
		}
		time.Sleep(sleep)
	}
}

func assertMissionStatusSnapshot(path string, expected missionStatusAssertionExpectation) error {
	snapshot, err := loadMissionStatusSnapshot(path)
	if err != nil {
		return err
	}
	return checkMissionStatusAssertion(path, snapshot, expected)
}

func assertMissionGatewayStatusSnapshot(path string, expected missionStatusAssertionExpectation) error {
	snapshot, err := loadGatewayStatusObservation(path)
	if err != nil {
		return err
	}
	return checkMissionStatusAssertion(path, projectGatewayStatusAssertionSnapshot(snapshot), expected)
}

func waitForMissionStatusAssertion(path string, expected missionStatusAssertionExpectation, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	var lastErr error

	for {
		lastErr = assertMissionStatusSnapshot(path, expected)
		if lastErr == nil {
			return nil
		}

		remaining := time.Until(deadline)
		if remaining <= 0 {
			return fmt.Errorf("timed out waiting up to %s for mission status file %q to satisfy assertion: %w", timeout, path, lastErr)
		}
		sleep := 100 * time.Millisecond
		if remaining < sleep {
			sleep = remaining
		}
		time.Sleep(sleep)
	}
}

func waitForMissionGatewayStatusAssertion(path string, expected missionStatusAssertionExpectation, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	var lastErr error

	for {
		lastErr = assertMissionGatewayStatusSnapshot(path, expected)
		if lastErr == nil {
			return nil
		}

		remaining := time.Until(deadline)
		if remaining <= 0 {
			return fmt.Errorf("timed out waiting up to %s for mission status file %q to satisfy assertion: %w", timeout, path, lastErr)
		}
		sleep := 100 * time.Millisecond
		if remaining < sleep {
			sleep = remaining
		}
		time.Sleep(sleep)
	}
}

func projectGatewayStatusAssertionSnapshot(snapshot missioncontrol.GatewayStatusSnapshot) missionStatusSnapshot {
	return missionStatusSnapshot{
		MissionRequired:   snapshot.MissionRequired,
		Active:            snapshot.Active,
		MissionFile:       snapshot.MissionFile,
		JobID:             snapshot.JobID,
		StepID:            snapshot.StepID,
		StepType:          snapshot.StepType,
		RequiredAuthority: snapshot.RequiredAuthority,
		RequiresApproval:  snapshot.RequiresApproval,
		AllowedTools:      append([]string(nil), snapshot.AllowedTools...),
		UpdatedAt:         snapshot.UpdatedAt,
	}
}

func newMissionStatusAssertionForStep(job missioncontrol.Job, stepID string) (missionStatusAssertionExpectation, error) {
	ec, err := missioncontrol.ResolveExecutionContext(job, stepID)
	if err != nil {
		return missionStatusAssertionExpectation{}, err
	}

	expected := missionStatusAssertionExpectation{
		ExactAllowedTools:      intersectAllowedTools(ec),
		CheckExactAllowedTools: true,
	}
	if ec.Job != nil {
		expected.JobID = valueOrNilString(ec.Job.ID)
	}
	if ec.Step != nil {
		expected.StepID = valueOrNilString(ec.Step.ID)
		stepType := string(ec.Step.Type)
		expected.StepType = &stepType
	}
	active := true
	expected.Active = &active

	return expected, nil
}

func checkMissionStatusAssertion(path string, snapshot missionStatusSnapshot, expected missionStatusAssertionExpectation) error {
	if expected.JobID != nil && snapshot.JobID != *expected.JobID {
		return fmt.Errorf("mission status file %q has job_id=%q step_id=%q active=%t, want job_id=%q", path, snapshot.JobID, snapshot.StepID, snapshot.Active, *expected.JobID)
	}
	if expected.StepID != nil && snapshot.StepID != *expected.StepID {
		return fmt.Errorf("mission status file %q has job_id=%q step_id=%q active=%t, want step_id=%q", path, snapshot.JobID, snapshot.StepID, snapshot.Active, *expected.StepID)
	}
	if expected.Active != nil && snapshot.Active != *expected.Active {
		return fmt.Errorf("mission status file %q has job_id=%q step_id=%q active=%t, want active=%t", path, snapshot.JobID, snapshot.StepID, snapshot.Active, *expected.Active)
	}
	if expected.StepType != nil && snapshot.StepType != *expected.StepType {
		return fmt.Errorf("mission status file %q has step_type=%q, want step_type=%q", path, snapshot.StepType, *expected.StepType)
	}
	if expected.RequiredAuthority != nil && snapshot.RequiredAuthority != *expected.RequiredAuthority {
		return fmt.Errorf("mission status file %q has required_authority=%q, want required_authority=%q", path, snapshot.RequiredAuthority, *expected.RequiredAuthority)
	}
	if expected.RequiresApproval != nil && snapshot.RequiresApproval != *expected.RequiresApproval {
		return fmt.Errorf("mission status file %q has requires_approval=%t, want requires_approval=%t", path, snapshot.RequiresApproval, *expected.RequiresApproval)
	}
	if expected.NoTools && len(snapshot.AllowedTools) != 0 {
		return fmt.Errorf("mission status file %q has allowed_tools=%q, want allowed_tools=[]", path, snapshot.AllowedTools)
	}
	for _, toolName := range expected.HasTools {
		if !containsString(snapshot.AllowedTools, toolName) {
			return fmt.Errorf("mission status file %q has allowed_tools=%q, want allowed_tools to include %q", path, snapshot.AllowedTools, toolName)
		}
	}
	if expected.CheckExactAllowedTools && !equalAllowedToolsExact(snapshot.AllowedTools, expected.ExactAllowedTools) {
		return fmt.Errorf("mission status file %q has allowed_tools=%q, want allowed_tools=%q", path, snapshot.AllowedTools, expected.ExactAllowedTools)
	}
	return nil
}

func equalAllowedToolsExact(got []string, want []string) bool {
	if len(got) == 0 && len(want) == 0 {
		return true
	}
	return reflect.DeepEqual(got, want)
}

func loadMissionStatusSnapshot(path string) (missionStatusSnapshot, error) {
	return loadMissionStatusObservation(path)
}

func valueOrNilString(value string) *string {
	return &value
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func writeMissionStatusSnapshot(path string, missionFile string, ag *agent.AgentLoop, now time.Time) error {
	if path == "" {
		return nil
	}

	var runtimeState *missioncontrol.JobRuntimeState
	if currentRuntime, ok := ag.MissionRuntimeState(); ok {
		runtimeState = missioncontrol.CloneJobRuntimeState(&currentRuntime)
	}
	var runtimeControl *missioncontrol.RuntimeControlContext
	if currentControl, ok := ag.MissionRuntimeControl(); ok {
		runtimeControl = missioncontrol.CloneRuntimeControlContext(&currentControl)
	}

	snapshot := missionStatusSnapshot{
		MissionRequired: ag.MissionRequired(),
		MissionFile:     missionFile,
		JobID:           "",
		StepID:          "",
		StepType:        "",
		AllowedTools:    []string{},
		Runtime:         runtimeState,
		RuntimeControl:  runtimeControl,
		UpdatedAt:       now.UTC().Format(time.RFC3339Nano),
	}

	if ec, ok := ag.ActiveMissionStep(); ok {
		snapshot.Active = true
		if ec.Job != nil {
			snapshot.JobID = ec.Job.ID
		}
		if ec.Step != nil {
			snapshot.StepID = ec.Step.ID
			snapshot.StepType = string(ec.Step.Type)
			snapshot.RequiredAuthority = ec.Step.RequiredAuthority
			snapshot.RequiresApproval = ec.Step.RequiresApproval
		}
		snapshot.AllowedTools = intersectAllowedTools(ec)
	} else if runtimeState != nil {
		snapshot.JobID = runtimeState.JobID
		snapshot.StepID = runtimeState.ActiveStepID
	}

	if runtimeState != nil {
		var summaryAllowedTools []string
		if snapshot.Active {
			summaryAllowedTools = append([]string(nil), snapshot.AllowedTools...)
		} else if runtimeControl != nil {
			summaryAllowedTools = missioncontrol.EffectiveAllowedTools(
				&missioncontrol.Job{AllowedTools: append([]string(nil), runtimeControl.AllowedTools...)},
				&runtimeControl.Step,
			)
		}
		summary := missioncontrol.BuildOperatorStatusSummaryWithAllowedTools(*runtimeState, summaryAllowedTools)
		snapshot.RuntimeSummary = &summary
	}

	return writeMissionStatusSnapshotAtomic(path, snapshot)
}

func writeProjectedMissionStatusSnapshot(path string, missionFile string, storeRoot string, missionRequired bool, jobID string, now time.Time) error {
	if path == "" {
		return nil
	}
	if strings.TrimSpace(storeRoot) == "" {
		return fmt.Errorf("mission status snapshot projection requires a durable store root")
	}
	if strings.TrimSpace(jobID) == "" {
		return fmt.Errorf("mission status snapshot projection requires a job_id")
	}
	if now.IsZero() {
		now = time.Now().UTC()
	} else {
		now = now.UTC()
	}

	snapshot, err := missioncontrol.BuildCommittedMissionStatusSnapshot(storeRoot, jobID, missioncontrol.MissionStatusSnapshotOptions{
		MissionRequired: missionRequired,
		MissionFile:     missionFile,
		UpdatedAt:       now,
	})
	if err != nil {
		return fmt.Errorf("failed to build committed mission status snapshot from durable store %q: %w", storeRoot, err)
	}
	return writeMissionStatusSnapshotAtomic(path, snapshot)
}

func intersectAllowedTools(ec missioncontrol.ExecutionContext) []string {
	if ec.Job == nil {
		return []string{}
	}

	jobTools := make(map[string]struct{}, len(ec.Job.AllowedTools))
	for _, toolName := range ec.Job.AllowedTools {
		jobTools[toolName] = struct{}{}
	}

	allowed := make([]string, 0, len(jobTools))
	if ec.Step == nil || len(ec.Step.AllowedTools) == 0 {
		for toolName := range jobTools {
			allowed = append(allowed, toolName)
		}
		sort.Strings(allowed)
		return allowed
	}

	for _, toolName := range ec.Step.AllowedTools {
		if _, ok := jobTools[toolName]; ok {
			allowed = append(allowed, toolName)
			delete(jobTools, toolName)
		}
	}
	sort.Strings(allowed)
	return allowed
}

func removeMissionStatusSnapshot(path string) error {
	if path == "" {
		return nil
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove mission status snapshot %q: %w", path, err)
	}
	return nil
}
