package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/local/picobot/internal/missioncontrol"
)

func TestMissionStatusCommandWithValidFilePrintsExpectedJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "status.json")
	want := []byte("{\n  \"mission_required\": true,\n  \"active\": true,\n  \"mission_file\": \"mission.json\",\n  \"job_id\": \"job-1\",\n  \"step_id\": \"build\",\n  \"step_type\": \"one_shot_code\",\n  \"required_authority\": \"\",\n  \"requires_approval\": false,\n  \"allowed_tools\": [\n    \"read\"\n  ],\n  \"updated_at\": \"2026-03-20T12:00:00Z\"\n}\n")
	if err := os.WriteFile(path, want, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cmd := NewRootCmd()
	out := &bytes.Buffer{}
	cmd.SetOut(out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mission", "status", "--status-file", path})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if out.String() != string(want) {
		t.Fatalf("stdout = %q, want %q", out.String(), string(want))
	}
}

func TestMissionStatusCommandWithActiveStepFieldsPrintsExpectedJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "status.json")
	wantSnapshot := missionStatusSnapshot{
		MissionRequired:   true,
		Active:            true,
		MissionFile:       "mission.json",
		JobID:             "job-1",
		StepID:            "build",
		StepType:          string(missioncontrol.StepTypeOneShotCode),
		RequiredAuthority: missioncontrol.AuthorityTierMedium,
		RequiresApproval:  true,
		AllowedTools:      []string{"read"},
		UpdatedAt:         "2026-03-20T12:00:00Z",
	}
	want, err := json.MarshalIndent(wantSnapshot, "", "  ")
	if err != nil {
		t.Fatalf("json.MarshalIndent() error = %v", err)
	}
	want = append(want, '\n')
	if err := os.WriteFile(path, want, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cmd := NewRootCmd()
	out := &bytes.Buffer{}
	cmd.SetOut(out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mission", "status", "--status-file", path})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if out.String() != string(want) {
		t.Fatalf("stdout = %q, want %q", out.String(), string(want))
	}
}

func TestMissionStatusCommandUsesSharedObservationReader(t *testing.T) {
	original := loadGatewayStatusObservationFile
	t.Cleanup(func() { loadGatewayStatusObservationFile = original })

	want := []byte("{\"job_id\":\"job-1\"}\n")
	called := 0
	loadGatewayStatusObservationFile = func(path string) ([]byte, error) {
		called++
		if path != "status.json" {
			t.Fatalf("shared observation path = %q, want %q", path, "status.json")
		}
		return want, nil
	}

	cmd := NewRootCmd()
	out := &bytes.Buffer{}
	cmd.SetOut(out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mission", "status", "--status-file", "status.json"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if called != 1 {
		t.Fatalf("shared observation calls = %d, want 1", called)
	}
	if out.String() != string(want) {
		t.Fatalf("stdout = %q, want %q", out.String(), string(want))
	}
}

func TestMissionStatusCommandPrintsCanonicalGatewayStatusJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "status.json")
	fullSnapshot := missionStatusSnapshot{
		MissionRequired:   true,
		Active:            true,
		MissionFile:       "mission.json",
		JobID:             "job-1",
		StepID:            "build",
		StepType:          string(missioncontrol.StepTypeOneShotCode),
		RequiredAuthority: missioncontrol.AuthorityTierMedium,
		RequiresApproval:  true,
		AllowedTools:      []string{"read"},
		Runtime: &missioncontrol.JobRuntimeState{
			JobID:        "job-1",
			State:        missioncontrol.JobStatePaused,
			ActiveStepID: "build",
		},
		RuntimeSummary: &missioncontrol.OperatorStatusSummary{
			JobID:        "job-1",
			State:        missioncontrol.JobStatePaused,
			ActiveStepID: "build",
			Artifacts: []missioncontrol.OperatorArtifactStatus{
				{StepID: "build", Path: "/tmp/private.txt"},
			},
		},
		RuntimeControl: &missioncontrol.RuntimeControlContext{
			JobID: "job-1",
			Step:  missioncontrol.Step{ID: "build"},
		},
		UpdatedAt: "2026-04-12T12:00:00Z",
	}
	data, err := json.MarshalIndent(fullSnapshot, "", "  ")
	if err != nil {
		t.Fatalf("json.MarshalIndent() error = %v", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cmd := NewRootCmd()
	out := &bytes.Buffer{}
	cmd.SetOut(out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mission", "status", "--status-file", path})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	got := out.String()
	if strings.Contains(got, `"runtime"`) {
		t.Fatalf("stdout = %q, want canonical gateway status without runtime", got)
	}
	if strings.Contains(got, `"runtime_summary"`) {
		t.Fatalf("stdout = %q, want canonical gateway status without runtime_summary", got)
	}
	if strings.Contains(got, `"runtime_control"`) {
		t.Fatalf("stdout = %q, want canonical gateway status without runtime_control", got)
	}
	if strings.Contains(got, `/tmp/private.txt`) {
		t.Fatalf("stdout = %q, want canonical gateway status without artifact paths", got)
	}
	if !strings.Contains(got, `"job_id": "job-1"`) || !strings.Contains(got, `"step_id": "build"`) {
		t.Fatalf("stdout = %q, want projected gateway status fields", got)
	}
}

func TestMissionStatusCommandReturnsFrankZohoSendProofLocatorsFromRuntimeSummary(t *testing.T) {
	originalGateway := loadGatewayStatusObservationFile
	originalMission := loadMissionStatusObservation
	t.Cleanup(func() {
		loadGatewayStatusObservationFile = originalGateway
		loadMissionStatusObservation = originalMission
	})

	loadGatewayStatusObservationFile = func(path string) ([]byte, error) {
		t.Fatalf("loadGatewayStatusObservationFile(%q) called, want full mission status reader for proof output", path)
		return nil, nil
	}

	loadMissionStatusObservation = func(path string) (missioncontrol.MissionStatusSnapshot, error) {
		if path != "status.json" {
			t.Fatalf("loadMissionStatusObservation path = %q, want %q", path, "status.json")
		}
		return missioncontrol.MissionStatusSnapshot{
			JobID: "job-1",
			Runtime: &missioncontrol.JobRuntimeState{
				JobID: "job-1",
				FrankZohoSendReceipts: []missioncontrol.FrankZohoSendReceipt{
					{
						ProviderMessageID:  "runtime-provider-message",
						ProviderMailID:     "<runtime-mail@zoho.test>",
						MIMEMessageID:      "<runtime-mime@example.test>",
						ProviderAccountID:  "runtime-account",
						OriginalMessageURL: "https://mail.zoho.com/api/accounts/runtime-account/messages/runtime-provider-message/originalmessage",
					},
				},
			},
			RuntimeSummary: &missioncontrol.OperatorStatusSummary{
				FrankZohoSendProof: []missioncontrol.OperatorFrankZohoSendProofStatus{
					{
						StepID:             "send",
						ProviderMessageID:  "1711540357880100000",
						ProviderMailID:     "<mail-1@zoho.test>",
						MIMEMessageID:      "<mime-1@example.test>",
						ProviderAccountID:  "3323462000000008002",
						OriginalMessageURL: "https://mail.zoho.com/api/accounts/3323462000000008002/messages/1711540357880100000/originalmessage",
					},
				},
			},
		}, nil
	}

	cmd := NewRootCmd()
	out := &bytes.Buffer{}
	cmd.SetOut(out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mission", "status", "--status-file", "status.json", "--frank-zoho-send-proof"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	var got []struct {
		ProviderMessageID  string `json:"provider_message_id"`
		ProviderMailID     string `json:"provider_mail_id"`
		MIMEMessageID      string `json:"mime_message_id"`
		ProviderAccountID  string `json:"provider_account_id"`
		OriginalMessageURL string `json:"original_message_url"`
	}
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("json.Unmarshal(stdout) error = %v\nstdout=%s", err, out.String())
	}
	if len(got) != 1 {
		t.Fatalf("len(proof locators) = %d, want 1", len(got))
	}
	var raw []map[string]any
	if err := json.Unmarshal(out.Bytes(), &raw); err != nil {
		t.Fatalf("json.Unmarshal(raw stdout) error = %v\nstdout=%s", err, out.String())
	}
	if len(raw) != 1 {
		t.Fatalf("len(raw proof locators) = %d, want 1", len(raw))
	}
	assertMainJSONObjectKeys(t, raw[0], "mime_message_id", "original_message_url", "provider_account_id", "provider_mail_id", "provider_message_id")
	if got[0].ProviderMessageID != "1711540357880100000" {
		t.Fatalf("ProviderMessageID = %q, want canonical provider message id from runtime_summary", got[0].ProviderMessageID)
	}
	if got[0].ProviderMailID != "<mail-1@zoho.test>" {
		t.Fatalf("ProviderMailID = %q, want secondary provider mail id from runtime_summary", got[0].ProviderMailID)
	}
	if got[0].MIMEMessageID != "<mime-1@example.test>" {
		t.Fatalf("MIMEMessageID = %q, want secondary MIME message id from runtime_summary", got[0].MIMEMessageID)
	}
	if got[0].ProviderAccountID != "3323462000000008002" {
		t.Fatalf("ProviderAccountID = %q, want proof locator account id from runtime_summary", got[0].ProviderAccountID)
	}
	if got[0].OriginalMessageURL != "https://mail.zoho.com/api/accounts/3323462000000008002/messages/1711540357880100000/originalmessage" {
		t.Fatalf("OriginalMessageURL = %q, want later-verification originalmessage URL", got[0].OriginalMessageURL)
	}
	if strings.Contains(out.String(), `"step_id"`) {
		t.Fatalf("stdout = %q, want proof locator output without step_id", out.String())
	}
}

func TestMissionStatusCommandVerifiesFrankZohoSendProofFromRuntimeSummary(t *testing.T) {
	originalGateway := loadGatewayStatusObservationFile
	originalMission := loadMissionStatusObservation
	originalVerifier := newFrankZohoSendProofVerifier
	t.Cleanup(func() {
		loadGatewayStatusObservationFile = originalGateway
		loadMissionStatusObservation = originalMission
		newFrankZohoSendProofVerifier = originalVerifier
	})

	loadGatewayStatusObservationFile = func(path string) ([]byte, error) {
		t.Fatalf("loadGatewayStatusObservationFile(%q) called, want committed mission status reader for verification output", path)
		return nil, nil
	}

	loadMissionStatusObservation = func(path string) (missioncontrol.MissionStatusSnapshot, error) {
		if path != "status.json" {
			t.Fatalf("loadMissionStatusObservation path = %q, want %q", path, "status.json")
		}
		return missioncontrol.MissionStatusSnapshot{
			JobID: "job-1",
			Runtime: &missioncontrol.JobRuntimeState{
				JobID: "job-1",
				FrankZohoSendReceipts: []missioncontrol.FrankZohoSendReceipt{
					{
						ProviderMessageID:  "runtime-provider-message",
						ProviderMailID:     "<runtime-mail@zoho.test>",
						MIMEMessageID:      "<runtime-mime@example.test>",
						ProviderAccountID:  "runtime-account",
						OriginalMessageURL: "https://mail.zoho.com/api/accounts/runtime-account/messages/runtime-provider-message/originalmessage",
					},
				},
			},
			RuntimeSummary: &missioncontrol.OperatorStatusSummary{
				FrankZohoSendProof: []missioncontrol.OperatorFrankZohoSendProofStatus{
					{
						StepID:             "send",
						ProviderMessageID:  "1711540357880100000",
						ProviderMailID:     "<mail-1@zoho.test>",
						MIMEMessageID:      "<mime-1@example.test>",
						ProviderAccountID:  "3323462000000008002",
						OriginalMessageURL: "https://mail.zoho.com/api/accounts/3323462000000008002/messages/1711540357880100000/originalmessage",
					},
				},
			},
		}, nil
	}

	var gotProof []missioncontrol.OperatorFrankZohoSendProofStatus
	newFrankZohoSendProofVerifier = func() missionStatusFrankZohoSendProofVerifier {
		return missionStatusFrankZohoSendProofVerifierFunc(func(ctx context.Context, proof []missioncontrol.OperatorFrankZohoSendProofStatus) ([]missionStatusFrankZohoSendProofVerification, error) {
			gotProof = append([]missioncontrol.OperatorFrankZohoSendProofStatus(nil), proof...)
			return []missionStatusFrankZohoSendProofVerification{
				{
					ProviderMessageID:  proof[0].ProviderMessageID,
					ProviderMailID:     proof[0].ProviderMailID,
					MIMEMessageID:      proof[0].MIMEMessageID,
					ProviderAccountID:  proof[0].ProviderAccountID,
					OriginalMessageURL: proof[0].OriginalMessageURL,
					OriginalMessage:    "From: Frank <frank@omou.online>\r\nSubject: Frank intro\r\n\r\nHello from Frank",
				},
			}, nil
		})
	}

	cmd := NewRootCmd()
	out := &bytes.Buffer{}
	cmd.SetOut(out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mission", "status", "--status-file", "status.json", "--frank-zoho-verify-send-proof"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if len(gotProof) != 1 {
		t.Fatalf("len(verifier proof input) = %d, want 1", len(gotProof))
	}
	if gotProof[0].ProviderMessageID != "1711540357880100000" {
		t.Fatalf("verifier proof ProviderMessageID = %q, want runtime_summary proof and not raw runtime receipt", gotProof[0].ProviderMessageID)
	}
	if gotProof[0].ProviderAccountID != "3323462000000008002" {
		t.Fatalf("verifier proof ProviderAccountID = %q, want committed runtime_summary proof", gotProof[0].ProviderAccountID)
	}
	if gotProof[0].OriginalMessageURL != "https://mail.zoho.com/api/accounts/3323462000000008002/messages/1711540357880100000/originalmessage" {
		t.Fatalf("verifier proof OriginalMessageURL = %q, want committed runtime_summary proof", gotProof[0].OriginalMessageURL)
	}

	var got []struct {
		ProviderMessageID  string `json:"provider_message_id"`
		ProviderMailID     string `json:"provider_mail_id"`
		MIMEMessageID      string `json:"mime_message_id"`
		ProviderAccountID  string `json:"provider_account_id"`
		OriginalMessageURL string `json:"original_message_url"`
		OriginalMessage    string `json:"original_message"`
	}
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("json.Unmarshal(stdout) error = %v\nstdout=%s", err, out.String())
	}
	if len(got) != 1 {
		t.Fatalf("len(verification records) = %d, want 1", len(got))
	}
	var raw []map[string]any
	if err := json.Unmarshal(out.Bytes(), &raw); err != nil {
		t.Fatalf("json.Unmarshal(raw stdout) error = %v\nstdout=%s", err, out.String())
	}
	if len(raw) != 1 {
		t.Fatalf("len(raw verification records) = %d, want 1", len(raw))
	}
	assertMainJSONObjectKeys(t, raw[0], "mime_message_id", "original_message", "original_message_url", "provider_account_id", "provider_mail_id", "provider_message_id")
	if got[0].ProviderMessageID != "1711540357880100000" {
		t.Fatalf("ProviderMessageID = %q, want committed runtime_summary proof locator", got[0].ProviderMessageID)
	}
	if got[0].ProviderMailID != "<mail-1@zoho.test>" {
		t.Fatalf("ProviderMailID = %q, want committed runtime_summary proof locator", got[0].ProviderMailID)
	}
	if got[0].MIMEMessageID != "<mime-1@example.test>" {
		t.Fatalf("MIMEMessageID = %q, want committed runtime_summary proof locator", got[0].MIMEMessageID)
	}
	if got[0].ProviderAccountID != "3323462000000008002" {
		t.Fatalf("ProviderAccountID = %q, want committed runtime_summary proof locator", got[0].ProviderAccountID)
	}
	if got[0].OriginalMessageURL != "https://mail.zoho.com/api/accounts/3323462000000008002/messages/1711540357880100000/originalmessage" {
		t.Fatalf("OriginalMessageURL = %q, want committed runtime_summary proof locator", got[0].OriginalMessageURL)
	}
	if got[0].OriginalMessage != "From: Frank <frank@omou.online>\r\nSubject: Frank intro\r\n\r\nHello from Frank" {
		t.Fatalf("OriginalMessage = %q, want verifier-fetched original message body", got[0].OriginalMessage)
	}
}

func TestMissionStatusCommandWithMissingFileReturnsError(t *testing.T) {
	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mission", "status", "--status-file", filepath.Join(t.TempDir(), "missing.json")})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "mission status file") || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("Execute() error = %q, want missing file message", err)
	}
}

func TestMissionStatusCommandWithInvalidFileReturnsError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "status.json")
	if err := os.WriteFile(path, []byte("{not-json"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mission", "status", "--status-file", path})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "failed to decode mission status file") {
		t.Fatalf("Execute() error = %q, want decode failure", err)
	}
}

func TestMissionAssertCommandWithValidStatusFileAndNoConditionsSucceeds(t *testing.T) {
	path := filepath.Join(t.TempDir(), "status.json")
	writeMissionStatusSnapshotFile(t, path, missionStatusSnapshot{
		Active:   true,
		JobID:    "job-1",
		StepID:   "build",
		StepType: string(missioncontrol.StepTypeOneShotCode),
	})

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mission", "assert", "--status-file", path})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
}

func TestMissionAssertCommandOneShotJobIDMatchSucceeds(t *testing.T) {
	path := filepath.Join(t.TempDir(), "status.json")
	writeMissionStatusSnapshotFile(t, path, missionStatusSnapshot{
		Active: true,
		JobID:  "job-1",
		StepID: "build",
	})

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mission", "assert", "--status-file", path, "--job-id", "job-1"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
}

func TestMissionAssertCommandOneShotStepIDMismatchFailsClearly(t *testing.T) {
	path := filepath.Join(t.TempDir(), "status.json")
	writeMissionStatusSnapshotFile(t, path, missionStatusSnapshot{
		Active: true,
		JobID:  "job-1",
		StepID: "build",
	})

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mission", "assert", "--status-file", path, "--step-id", "final"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), `has job_id="job-1" step_id="build" active=true, want step_id="final"`) {
		t.Fatalf("Execute() error = %q, want clear step_id mismatch", err)
	}
}

func TestMissionAssertCommandOneShotActiveMismatchFailsClearly(t *testing.T) {
	path := filepath.Join(t.TempDir(), "status.json")
	writeMissionStatusSnapshotFile(t, path, missionStatusSnapshot{
		Active: false,
		JobID:  "job-1",
		StepID: "build",
	})

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mission", "assert", "--status-file", path, "--active=true"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), `has job_id="job-1" step_id="build" active=false, want active=true`) {
		t.Fatalf("Execute() error = %q, want clear active mismatch", err)
	}
}

func TestMissionAssertCommandOneShotStepTypeMatchSucceeds(t *testing.T) {
	path := filepath.Join(t.TempDir(), "status.json")
	writeMissionStatusSnapshotFile(t, path, missionStatusSnapshot{
		Active:   true,
		JobID:    "job-1",
		StepID:   "build",
		StepType: string(missioncontrol.StepTypeOneShotCode),
	})

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mission", "assert", "--status-file", path, "--step-type", string(missioncontrol.StepTypeOneShotCode)})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
}

func TestMissionAssertCommandOneShotStepTypeMismatchFailsClearly(t *testing.T) {
	path := filepath.Join(t.TempDir(), "status.json")
	writeMissionStatusSnapshotFile(t, path, missionStatusSnapshot{
		Active:   true,
		JobID:    "job-1",
		StepID:   "build",
		StepType: string(missioncontrol.StepTypeDiscussion),
	})

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mission", "assert", "--status-file", path, "--step-type", string(missioncontrol.StepTypeOneShotCode)})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), `has step_type="discussion", want step_type="one_shot_code"`) {
		t.Fatalf("Execute() error = %q, want clear step_type mismatch", err)
	}
}

func TestMissionAssertCommandOneShotRequiredAuthorityMatchSucceeds(t *testing.T) {
	path := filepath.Join(t.TempDir(), "status.json")
	writeMissionStatusSnapshotFile(t, path, missionStatusSnapshot{
		Active:            true,
		JobID:             "job-1",
		StepID:            "build",
		RequiredAuthority: missioncontrol.AuthorityTierMedium,
	})

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mission", "assert", "--status-file", path, "--required-authority", string(missioncontrol.AuthorityTierMedium)})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
}

func TestMissionAssertCommandOneShotRequiredAuthorityMismatchFailsClearly(t *testing.T) {
	path := filepath.Join(t.TempDir(), "status.json")
	writeMissionStatusSnapshotFile(t, path, missionStatusSnapshot{
		Active:            true,
		JobID:             "job-1",
		StepID:            "build",
		RequiredAuthority: missioncontrol.AuthorityTierLow,
	})

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mission", "assert", "--status-file", path, "--required-authority", string(missioncontrol.AuthorityTierMedium)})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), `has required_authority="low", want required_authority="medium"`) {
		t.Fatalf("Execute() error = %q, want clear required_authority mismatch", err)
	}
}

func TestMissionAssertCommandOneShotRequiresApprovalSucceedsWhenTrue(t *testing.T) {
	path := filepath.Join(t.TempDir(), "status.json")
	writeMissionStatusSnapshotFile(t, path, missionStatusSnapshot{
		Active:           true,
		JobID:            "job-1",
		StepID:           "build",
		RequiresApproval: true,
	})

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mission", "assert", "--status-file", path, "--requires-approval"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
}

func TestMissionAssertCommandOneShotRequiresApprovalFailsClearlyWhenFalse(t *testing.T) {
	path := filepath.Join(t.TempDir(), "status.json")
	writeMissionStatusSnapshotFile(t, path, missionStatusSnapshot{
		Active:           true,
		JobID:            "job-1",
		StepID:           "build",
		RequiresApproval: false,
	})

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mission", "assert", "--status-file", path, "--requires-approval"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), `has requires_approval=false, want requires_approval=true`) {
		t.Fatalf("Execute() error = %q, want clear requires_approval mismatch", err)
	}
}

func TestMissionAssertCommandOneShotNoRequiresApprovalSucceedsWhenFalse(t *testing.T) {
	path := filepath.Join(t.TempDir(), "status.json")
	writeMissionStatusSnapshotFile(t, path, missionStatusSnapshot{
		Active:           true,
		JobID:            "job-1",
		StepID:           "build",
		RequiresApproval: false,
	})

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mission", "assert", "--status-file", path, "--no-requires-approval"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
}

func TestMissionAssertCommandOneShotNoToolsSucceedsForEmptyAllowedTools(t *testing.T) {
	path := filepath.Join(t.TempDir(), "status.json")
	writeMissionStatusSnapshotFile(t, path, missionStatusSnapshot{
		Active:       true,
		JobID:        "job-1",
		StepID:       "build",
		AllowedTools: []string{},
	})

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mission", "assert", "--status-file", path, "--no-tools"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
}

func TestMissionAssertCommandOneShotNoToolsFailsClearlyWhenToolsArePresent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "status.json")
	writeMissionStatusSnapshotFile(t, path, missionStatusSnapshot{
		Active:       true,
		JobID:        "job-1",
		StepID:       "build",
		AllowedTools: []string{"read", "write"},
	})

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mission", "assert", "--status-file", path, "--no-tools"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), `has allowed_tools=["read" "write"], want allowed_tools=[]`) {
		t.Fatalf("Execute() error = %q, want clear allowed_tools mismatch", err)
	}
}

func TestMissionAssertCommandOneShotHasToolSucceedsWhenToolIsPresent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "status.json")
	writeMissionStatusSnapshotFile(t, path, missionStatusSnapshot{
		Active:       true,
		JobID:        "job-1",
		StepID:       "build",
		AllowedTools: []string{"read", "write"},
	})

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mission", "assert", "--status-file", path, "--has-tool", "write"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
}

func TestMissionAssertCommandOneShotHasToolFailsClearlyWhenToolIsAbsent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "status.json")
	writeMissionStatusSnapshotFile(t, path, missionStatusSnapshot{
		Active:       true,
		JobID:        "job-1",
		StepID:       "build",
		AllowedTools: []string{"read"},
	})

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mission", "assert", "--status-file", path, "--has-tool", "write"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), `has allowed_tools=["read"], want allowed_tools to include "write"`) {
		t.Fatalf("Execute() error = %q, want clear missing tool message", err)
	}
}

func TestMissionAssertCommandOneShotExactToolSucceedsWhenAllowedToolsExactlyMatch(t *testing.T) {
	path := filepath.Join(t.TempDir(), "status.json")
	writeMissionStatusSnapshotFile(t, path, missionStatusSnapshot{
		Active:       true,
		JobID:        "job-1",
		StepID:       "build",
		AllowedTools: []string{"read", "write"},
	})

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mission", "assert", "--status-file", path, "--exact-tool", "read", "--exact-tool", "write"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
}

func TestMissionAssertCommandOneShotExactToolFailsClearlyWhenAllowedToolsDoNotExactlyMatch(t *testing.T) {
	path := filepath.Join(t.TempDir(), "status.json")
	writeMissionStatusSnapshotFile(t, path, missionStatusSnapshot{
		Active:       true,
		JobID:        "job-1",
		StepID:       "build",
		AllowedTools: []string{"read"},
	})

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mission", "assert", "--status-file", path, "--exact-tool", "read", "--exact-tool", "write"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), `has allowed_tools=["read"], want allowed_tools=["read" "write"]`) {
		t.Fatalf("Execute() error = %q, want clear exact allowed_tools mismatch", err)
	}
}

func TestMissionAssertCommandWaitSucceedsWhenStatusFileChangesBeforeTimeout(t *testing.T) {
	path := filepath.Join(t.TempDir(), "status.json")
	writeMissionStatusSnapshotFile(t, path, missionStatusSnapshot{
		Active: true,
		JobID:  "job-1",
		StepID: "build",
	})

	done := make(chan error, 1)
	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{
		"mission", "assert",
		"--status-file", path,
		"--step-id", "final",
		"--wait-timeout", "250ms",
	})
	go func() {
		done <- cmd.Execute()
	}()

	select {
	case err := <-done:
		t.Fatalf("Execute() returned before matching status update: %v", err)
	case <-time.After(20 * time.Millisecond):
	}

	writeMissionStatusSnapshotFile(t, path, missionStatusSnapshot{
		Active: true,
		JobID:  "job-1",
		StepID: "final",
	})

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Execute() did not return after matching status update")
	}
}

func TestMissionAssertCommandWaitSucceedsWhenAllowedToolsChangeBeforeTimeout(t *testing.T) {
	path := filepath.Join(t.TempDir(), "status.json")
	writeMissionStatusSnapshotFile(t, path, missionStatusSnapshot{
		Active:       true,
		JobID:        "job-1",
		StepID:       "build",
		AllowedTools: []string{"read"},
	})

	done := make(chan error, 1)
	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{
		"mission", "assert",
		"--status-file", path,
		"--has-tool", "write",
		"--wait-timeout", "250ms",
	})
	go func() {
		done <- cmd.Execute()
	}()

	select {
	case err := <-done:
		t.Fatalf("Execute() returned before matching status update: %v", err)
	case <-time.After(20 * time.Millisecond):
	}

	writeMissionStatusSnapshotFile(t, path, missionStatusSnapshot{
		Active:       true,
		JobID:        "job-1",
		StepID:       "build",
		AllowedTools: []string{"read", "write"},
	})

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Execute() did not return after matching status update")
	}
}

func TestMissionAssertCommandWaitSucceedsWhenAllowedToolsExactlyMatchBeforeTimeout(t *testing.T) {
	path := filepath.Join(t.TempDir(), "status.json")
	writeMissionStatusSnapshotFile(t, path, missionStatusSnapshot{
		Active:       true,
		JobID:        "job-1",
		StepID:       "build",
		AllowedTools: []string{"read"},
	})

	done := make(chan error, 1)
	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{
		"mission", "assert",
		"--status-file", path,
		"--exact-tool", "read",
		"--exact-tool", "write",
		"--wait-timeout", "250ms",
	})
	go func() {
		done <- cmd.Execute()
	}()

	select {
	case err := <-done:
		t.Fatalf("Execute() returned before matching status update: %v", err)
	case <-time.After(20 * time.Millisecond):
	}

	writeMissionStatusSnapshotFile(t, path, missionStatusSnapshot{
		Active:       true,
		JobID:        "job-1",
		StepID:       "build",
		AllowedTools: []string{"read", "write"},
	})

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Execute() did not return after matching status update")
	}
}

func TestMissionAssertCommandWaitTimesOutWhenValuesNeverMatch(t *testing.T) {
	path := filepath.Join(t.TempDir(), "status.json")
	writeMissionStatusSnapshotFile(t, path, missionStatusSnapshot{
		Active: true,
		JobID:  "job-1",
		StepID: "build",
	})

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{
		"mission", "assert",
		"--status-file", path,
		"--step-id", "final",
		"--wait-timeout", "75ms",
	})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "timed out waiting up to 75ms") {
		t.Fatalf("Execute() error = %q, want timeout error", err)
	}
	if !strings.Contains(err.Error(), `has job_id="job-1" step_id="build" active=true, want step_id="final"`) {
		t.Fatalf("Execute() error = %q, want observed and expected values", err)
	}
}

func TestMissionAssertCommandWithMissingStatusFileReturnsClearError(t *testing.T) {
	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mission", "assert", "--status-file", filepath.Join(t.TempDir(), "missing.json")})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "mission status file") || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("Execute() error = %q, want missing file message", err)
	}
}

func TestMissionAssertCommandWithInvalidJSONReturnsClearError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "status.json")
	if err := os.WriteFile(path, []byte("{not-json"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mission", "assert", "--status-file", path})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "failed to decode mission status file") {
		t.Fatalf("Execute() error = %q, want decode failure", err)
	}
}

func TestMissionAssertCommandUsesSharedGatewayObservationReader(t *testing.T) {
	original := loadGatewayStatusObservation
	t.Cleanup(func() { loadGatewayStatusObservation = original })

	called := 0
	loadGatewayStatusObservation = func(path string) (missioncontrol.GatewayStatusSnapshot, error) {
		called++
		if path != "status.json" {
			t.Fatalf("shared gateway observation path = %q, want %q", path, "status.json")
		}
		return missioncontrol.GatewayStatusSnapshot{
			Active:            true,
			JobID:             "job-1",
			StepID:            "build",
			StepType:          string(missioncontrol.StepTypeOneShotCode),
			RequiredAuthority: missioncontrol.AuthorityTierMedium,
			RequiresApproval:  true,
			AllowedTools:      []string{"read", "write"},
		}, nil
	}

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{
		"mission", "assert",
		"--status-file", "status.json",
		"--job-id", "job-1",
		"--step-id", "build",
		"--active=true",
		"--step-type", string(missioncontrol.StepTypeOneShotCode),
		"--required-authority", string(missioncontrol.AuthorityTierMedium),
		"--requires-approval",
		"--exact-tool", "read",
		"--exact-tool", "write",
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if called != 1 {
		t.Fatalf("shared gateway observation calls = %d, want 1", called)
	}
}

func TestMissionAssertStepCommandUsesSharedGatewayObservationReader(t *testing.T) {
	job := missioncontrol.Job{
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
					RequiresApproval:  true,
					AllowedTools:      []string{"write", "read"},
				},
				{
					ID:        "final",
					Type:      missioncontrol.StepTypeFinalResponse,
					DependsOn: []string{"build"},
				},
			},
		},
	}
	missionPath := writeMissionBootstrapJobFile(t, job)

	original := loadGatewayStatusObservation
	t.Cleanup(func() { loadGatewayStatusObservation = original })

	called := 0
	loadGatewayStatusObservation = func(path string) (missioncontrol.GatewayStatusSnapshot, error) {
		called++
		if path != "status.json" {
			t.Fatalf("shared gateway observation path = %q, want %q", path, "status.json")
		}
		return missioncontrol.GatewayStatusSnapshot{
			Active:            true,
			JobID:             "job-1",
			StepID:            "build",
			StepType:          string(missioncontrol.StepTypeOneShotCode),
			RequiredAuthority: missioncontrol.AuthorityTierMedium,
			RequiresApproval:  true,
			AllowedTools:      []string{"read", "write"},
		}, nil
	}

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mission", "assert-step", "--mission-file", missionPath, "--status-file", "status.json", "--step-id", "build"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if called != 1 {
		t.Fatalf("shared gateway observation calls = %d, want 1", called)
	}
}

func TestMissionAssertCommandNoToolsAndHasToolReturnsClearArgumentError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "status.json")
	writeMissionStatusSnapshotFile(t, path, missionStatusSnapshot{
		Active:       true,
		JobID:        "job-1",
		StepID:       "build",
		AllowedTools: []string{"read"},
	})

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mission", "assert", "--status-file", path, "--no-tools", "--has-tool", "read"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "--no-tools and --has-tool cannot be used together") {
		t.Fatalf("Execute() error = %q, want clear argument error", err)
	}
}

func TestMissionAssertCommandNoToolsAndExactToolReturnsClearArgumentError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "status.json")
	writeMissionStatusSnapshotFile(t, path, missionStatusSnapshot{
		Active:       true,
		JobID:        "job-1",
		StepID:       "build",
		AllowedTools: []string{"read"},
	})

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mission", "assert", "--status-file", path, "--no-tools", "--exact-tool", "read"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "--no-tools and --exact-tool cannot be used together") {
		t.Fatalf("Execute() error = %q, want clear argument error", err)
	}
}

func TestMissionAssertCommandHasToolAndExactToolReturnsClearArgumentError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "status.json")
	writeMissionStatusSnapshotFile(t, path, missionStatusSnapshot{
		Active:       true,
		JobID:        "job-1",
		StepID:       "build",
		AllowedTools: []string{"read"},
	})

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mission", "assert", "--status-file", path, "--has-tool", "read", "--exact-tool", "read"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "--has-tool and --exact-tool cannot be used together") {
		t.Fatalf("Execute() error = %q, want clear argument error", err)
	}
}

func TestMissionAssertCommandRequiresApprovalAndNoRequiresApprovalReturnsClearArgumentError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "status.json")
	writeMissionStatusSnapshotFile(t, path, missionStatusSnapshot{
		Active: true,
		JobID:  "job-1",
		StepID: "build",
	})

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mission", "assert", "--status-file", path, "--requires-approval", "--no-requires-approval"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "--requires-approval and --no-requires-approval cannot be used together") {
		t.Fatalf("Execute() error = %q, want clear argument error", err)
	}
}

func TestMissionAssertStepCommandSucceedsWhenStatusMatchesMissionStep(t *testing.T) {
	job := missioncontrol.Job{
		ID:           "job-1",
		MaxAuthority: missioncontrol.AuthorityTierHigh,
		AllowedTools: []string{"write", "read", "search"},
		Plan: missioncontrol.Plan{
			ID: "plan-1",
			Steps: []missioncontrol.Step{
				{
					ID:                "build",
					Type:              missioncontrol.StepTypeOneShotCode,
					RequiredAuthority: missioncontrol.AuthorityTierLow,
					AllowedTools:      []string{"write", "read"},
				},
				{
					ID:        "final",
					Type:      missioncontrol.StepTypeFinalResponse,
					DependsOn: []string{"build"},
				},
			},
		},
	}
	missionPath := writeMissionBootstrapJobFile(t, job)
	statusPath := filepath.Join(t.TempDir(), "status.json")
	writeMissionStatusSnapshotFile(t, statusPath, missionStatusSnapshot{
		Active:       true,
		JobID:        "job-1",
		StepID:       "build",
		StepType:     string(missioncontrol.StepTypeOneShotCode),
		AllowedTools: []string{"read", "write"},
	})

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mission", "assert-step", "--mission-file", missionPath, "--status-file", statusPath, "--step-id", "build"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
}

func TestMissionAssertStepCommandSucceedsForZeroToolStepWhenStatusAllowedToolsIsNil(t *testing.T) {
	job := missioncontrol.Job{
		ID:           "phone-discussion-v1",
		MaxAuthority: missioncontrol.AuthorityTierLow,
		AllowedTools: []string{},
		Plan: missioncontrol.Plan{
			ID: "plan-1",
			Steps: []missioncontrol.Step{
				{
					ID:                "discuss",
					Type:              missioncontrol.StepTypeDiscussion,
					RequiredAuthority: missioncontrol.AuthorityTierLow,
				},
				{
					ID:        "final",
					Type:      missioncontrol.StepTypeFinalResponse,
					DependsOn: []string{"discuss"},
				},
			},
		},
	}
	missionPath := writeMissionBootstrapJobFile(t, job)
	statusPath := filepath.Join(t.TempDir(), "status.json")
	writeMissionStatusSnapshotFile(t, statusPath, missionStatusSnapshot{
		Active:    true,
		JobID:     "phone-discussion-v1",
		StepID:    "discuss",
		StepType:  string(missioncontrol.StepTypeDiscussion),
		UpdatedAt: "2026-03-21T10:00:00Z",
	})

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mission", "assert-step", "--mission-file", missionPath, "--status-file", statusPath, "--step-id", "discuss"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
}

func TestMissionAssertStepCommandFailsClearlyWhenAllowedToolsDoNotExactlyMatch(t *testing.T) {
	job := missioncontrol.Job{
		ID:           "job-1",
		MaxAuthority: missioncontrol.AuthorityTierHigh,
		AllowedTools: []string{"read", "write"},
		Plan: missioncontrol.Plan{
			ID: "plan-1",
			Steps: []missioncontrol.Step{
				{
					ID:                "build",
					Type:              missioncontrol.StepTypeOneShotCode,
					RequiredAuthority: missioncontrol.AuthorityTierLow,
					AllowedTools:      []string{"write", "read"},
				},
				{
					ID:        "final",
					Type:      missioncontrol.StepTypeFinalResponse,
					DependsOn: []string{"build"},
				},
			},
		},
	}
	missionPath := writeMissionBootstrapJobFile(t, job)
	statusPath := filepath.Join(t.TempDir(), "status.json")
	writeMissionStatusSnapshotFile(t, statusPath, missionStatusSnapshot{
		Active:       true,
		JobID:        "job-1",
		StepID:       "build",
		StepType:     string(missioncontrol.StepTypeOneShotCode),
		AllowedTools: []string{"read"},
	})

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mission", "assert-step", "--mission-file", missionPath, "--status-file", statusPath, "--step-id", "build"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), `has allowed_tools=["read"], want allowed_tools=["read" "write"]`) {
		t.Fatalf("Execute() error = %q, want exact allowed_tools mismatch", err)
	}
}

func TestMissionAssertStepCommandUnknownStepReturnsClearError(t *testing.T) {
	missionPath := writeMissionBootstrapJobFile(t, testMissionBootstrapJob())
	statusPath := filepath.Join(t.TempDir(), "status.json")
	writeMissionStatusSnapshotFile(t, statusPath, missionStatusSnapshot{
		Active: true,
		JobID:  "job-1",
		StepID: "build",
	})

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mission", "assert-step", "--mission-file", missionPath, "--status-file", statusPath, "--step-id", "missing"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "failed to validate mission file") {
		t.Fatalf("Execute() error = %q, want validation failure", err)
	}
	if !strings.Contains(err.Error(), "E_INVALID_ACTION_FOR_STEP") {
		t.Fatalf("Execute() error = %q, want unknown_step code", err)
	}
	if !strings.Contains(err.Error(), `step "missing" not found in plan`) {
		t.Fatalf("Execute() error = %q, want missing step message", err)
	}
}

func TestMissionAssertStepCommandWaitSucceedsWhenStatusChangesBeforeTimeout(t *testing.T) {
	job := missioncontrol.Job{
		ID:           "job-1",
		MaxAuthority: missioncontrol.AuthorityTierHigh,
		AllowedTools: []string{"read", "write"},
		Plan: missioncontrol.Plan{
			ID: "plan-1",
			Steps: []missioncontrol.Step{
				{
					ID:                "build",
					Type:              missioncontrol.StepTypeOneShotCode,
					RequiredAuthority: missioncontrol.AuthorityTierLow,
					AllowedTools:      []string{"write", "read"},
				},
				{
					ID:        "final",
					Type:      missioncontrol.StepTypeFinalResponse,
					DependsOn: []string{"build"},
				},
			},
		},
	}
	missionPath := writeMissionBootstrapJobFile(t, job)
	statusPath := filepath.Join(t.TempDir(), "status.json")
	writeMissionStatusSnapshotFile(t, statusPath, missionStatusSnapshot{
		Active:       true,
		JobID:        "job-1",
		StepID:       "build",
		StepType:     string(missioncontrol.StepTypeOneShotCode),
		AllowedTools: []string{"read"},
	})

	done := make(chan error, 1)
	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{
		"mission", "assert-step",
		"--mission-file", missionPath,
		"--status-file", statusPath,
		"--step-id", "build",
		"--wait-timeout", "250ms",
	})
	go func() {
		done <- cmd.Execute()
	}()

	select {
	case err := <-done:
		t.Fatalf("Execute() returned before matching status update: %v", err)
	case <-time.After(20 * time.Millisecond):
	}

	writeMissionStatusSnapshotFile(t, statusPath, missionStatusSnapshot{
		Active:       true,
		JobID:        "job-1",
		StepID:       "build",
		StepType:     string(missioncontrol.StepTypeOneShotCode),
		AllowedTools: []string{"read", "write"},
	})

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Execute() did not return after matching status update")
	}
}

func TestMissionAssertStepCommandWithInvalidMissionReturnsValidationError(t *testing.T) {
	missionPath := writeMissionBootstrapJobFile(t, missioncontrol.Job{
		ID:           "job-1",
		MaxAuthority: missioncontrol.AuthorityTierHigh,
		AllowedTools: []string{"read"},
		Plan:         missioncontrol.Plan{ID: "plan-1"},
	})
	statusPath := filepath.Join(t.TempDir(), "status.json")
	writeMissionStatusSnapshotFile(t, statusPath, missionStatusSnapshot{
		Active: true,
		JobID:  "job-1",
		StepID: "build",
	})

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mission", "assert-step", "--mission-file", missionPath, "--status-file", statusPath, "--step-id", "build"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "failed to validate mission file") {
		t.Fatalf("Execute() error = %q, want validation failure", err)
	}
	if !strings.Contains(err.Error(), "E_PLAN_INVALID") {
		t.Fatalf("Execute() error = %q, want validation error code", err)
	}
}
