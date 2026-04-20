package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/local/picobot/internal/config"
	"github.com/local/picobot/internal/missioncontrol"
)

func writeMalformedTreasuryRecordForMainTest(t *testing.T, root string, treasury missioncontrol.TreasuryRecord) {
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

func writeMissionInspectNotificationsCapabilityFixtures(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	now := time.Date(2026, 4, 18, 18, 0, 0, 0, time.UTC)
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
		State:            missioncontrol.CapabilityOnboardingProposalStateApproved,
	}
	if err := missioncontrol.StoreCapabilityOnboardingProposalRecord(root, record); err != nil {
		t.Fatalf("StoreCapabilityOnboardingProposalRecord() error = %v", err)
	}
	if _, err := missioncontrol.StoreTelegramNotificationsCapabilityExposure(root); err != nil {
		t.Fatalf("StoreTelegramNotificationsCapabilityExposure() error = %v", err)
	}
	return root
}

func writeMissionInspectSharedStorageCapabilityFixtures(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	now := time.Date(2026, 4, 18, 20, 0, 0, 0, time.UTC)
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
		State:            missioncontrol.CapabilityOnboardingProposalStateApproved,
	}
	if err := missioncontrol.StoreCapabilityOnboardingProposalRecord(root, record); err != nil {
		t.Fatalf("StoreCapabilityOnboardingProposalRecord() error = %v", err)
	}
	if _, err := missioncontrol.StoreWorkspaceSharedStorageCapabilityExposure(root, filepath.Join(t.TempDir(), "workspace")); err != nil {
		t.Fatalf("StoreWorkspaceSharedStorageCapabilityExposure() error = %v", err)
	}
	return root
}

func writeMissionInspectContactsCapabilityFixtures(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	home := t.TempDir()
	workspace := filepath.Join(home, "workspace-root")
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.Workspace = workspace
	if err := config.SaveConfig(cfg, filepath.Join(home, ".picobot", "config.json")); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}
	t.Setenv("HOME", home)

	now := time.Date(2026, 4, 18, 23, 0, 0, 0, time.UTC)
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
		State:            missioncontrol.CapabilityOnboardingProposalStateApproved,
	}
	if err := missioncontrol.StoreCapabilityOnboardingProposalRecord(root, record); err != nil {
		t.Fatalf("StoreCapabilityOnboardingProposalRecord() error = %v", err)
	}
	if _, err := missioncontrol.StoreWorkspaceSharedStorageCapabilityExposure(root, workspace); err != nil {
		t.Fatalf("StoreWorkspaceSharedStorageCapabilityExposure() error = %v", err)
	}
	if _, err := missioncontrol.StoreWorkspaceContactsCapabilityExposure(root, workspace); err != nil {
		t.Fatalf("StoreWorkspaceContactsCapabilityExposure() error = %v", err)
	}
	return root
}

func writeMissionInspectLocationCapabilityFixtures(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	home := t.TempDir()
	workspace := filepath.Join(home, "workspace-root")
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.Workspace = workspace
	if err := config.SaveConfig(cfg, filepath.Join(home, ".picobot", "config.json")); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}
	t.Setenv("HOME", home)

	now := time.Date(2026, 4, 19, 0, 0, 0, 0, time.UTC)
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
		State:            missioncontrol.CapabilityOnboardingProposalStateApproved,
	}
	if err := missioncontrol.StoreCapabilityOnboardingProposalRecord(root, record); err != nil {
		t.Fatalf("StoreCapabilityOnboardingProposalRecord() error = %v", err)
	}
	if _, err := missioncontrol.StoreWorkspaceSharedStorageCapabilityExposure(root, workspace); err != nil {
		t.Fatalf("StoreWorkspaceSharedStorageCapabilityExposure() error = %v", err)
	}
	if _, err := missioncontrol.StoreWorkspaceLocationCapabilityExposure(root, workspace); err != nil {
		t.Fatalf("StoreWorkspaceLocationCapabilityExposure() error = %v", err)
	}
	return root
}

func writeMissionInspectCameraCapabilityFixtures(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	home := t.TempDir()
	workspace := filepath.Join(home, "workspace-root")
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.Workspace = workspace
	if err := config.SaveConfig(cfg, filepath.Join(home, ".picobot", "config.json")); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}
	t.Setenv("HOME", home)

	now := time.Date(2026, 4, 19, 3, 0, 0, 0, time.UTC)
	record := missioncontrol.CapabilityOnboardingProposalRecord{
		ProposalID:       "proposal-camera",
		CapabilityName:   missioncontrol.CameraCapabilityName,
		WhyNeeded:        "mission requires local shared camera-image access",
		MissionFamilies:  []string{"workspace"},
		Risks:            []string{"local camera source exposure"},
		Validators:       []string{"shared_storage exposed and committed camera source file exists and is readable"},
		KillSwitch:       "disable camera capability exposure and remove committed camera source reference",
		DataAccessed:     []string{"camera"},
		ApprovalRequired: true,
		CreatedAt:        now,
		State:            missioncontrol.CapabilityOnboardingProposalStateApproved,
	}
	if err := missioncontrol.StoreCapabilityOnboardingProposalRecord(root, record); err != nil {
		t.Fatalf("StoreCapabilityOnboardingProposalRecord() error = %v", err)
	}
	if _, err := missioncontrol.StoreWorkspaceSharedStorageCapabilityExposure(root, workspace); err != nil {
		t.Fatalf("StoreWorkspaceSharedStorageCapabilityExposure() error = %v", err)
	}
	if _, err := missioncontrol.StoreWorkspaceCameraCapabilityExposure(root, workspace); err != nil {
		t.Fatalf("StoreWorkspaceCameraCapabilityExposure() error = %v", err)
	}
	return root
}

func writeMissionInspectMicrophoneCapabilityFixtures(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	home := t.TempDir()
	workspace := filepath.Join(home, "workspace-root")
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.Workspace = workspace
	if err := config.SaveConfig(cfg, filepath.Join(home, ".picobot", "config.json")); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}
	t.Setenv("HOME", home)

	now := time.Date(2026, 4, 19, 6, 0, 0, 0, time.UTC)
	record := missioncontrol.CapabilityOnboardingProposalRecord{
		ProposalID:       "proposal-microphone",
		CapabilityName:   missioncontrol.MicrophoneCapabilityName,
		WhyNeeded:        "mission requires local shared microphone-audio access",
		MissionFamilies:  []string{"workspace"},
		Risks:            []string{"local microphone source exposure"},
		Validators:       []string{"shared_storage exposed and committed microphone source file exists and is readable"},
		KillSwitch:       "disable microphone capability exposure and remove committed microphone source reference",
		DataAccessed:     []string{"microphone"},
		ApprovalRequired: true,
		CreatedAt:        now,
		State:            missioncontrol.CapabilityOnboardingProposalStateApproved,
	}
	if err := missioncontrol.StoreCapabilityOnboardingProposalRecord(root, record); err != nil {
		t.Fatalf("StoreCapabilityOnboardingProposalRecord() error = %v", err)
	}
	if _, err := missioncontrol.StoreWorkspaceSharedStorageCapabilityExposure(root, workspace); err != nil {
		t.Fatalf("StoreWorkspaceSharedStorageCapabilityExposure() error = %v", err)
	}
	if _, err := missioncontrol.StoreWorkspaceMicrophoneCapabilityExposure(root, workspace); err != nil {
		t.Fatalf("StoreWorkspaceMicrophoneCapabilityExposure() error = %v", err)
	}
	return root
}

func writeMissionInspectSMSPhoneCapabilityFixtures(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	home := t.TempDir()
	workspace := filepath.Join(home, "workspace-root")
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.Workspace = workspace
	if err := config.SaveConfig(cfg, filepath.Join(home, ".picobot", "config.json")); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}
	t.Setenv("HOME", home)

	now := time.Date(2026, 4, 19, 9, 0, 0, 0, time.UTC)
	record := missioncontrol.CapabilityOnboardingProposalRecord{
		ProposalID:       "proposal-sms-phone",
		CapabilityName:   missioncontrol.SMSPhoneCapabilityName,
		WhyNeeded:        "mission requires local shared SMS/phone source access",
		MissionFamilies:  []string{"workspace"},
		Risks:            []string{"local SMS/phone source exposure"},
		Validators:       []string{"shared_storage exposed and committed sms_phone source file exists and is readable"},
		KillSwitch:       "disable sms_phone capability exposure and remove committed sms_phone source reference",
		DataAccessed:     []string{"SMS/phone"},
		ApprovalRequired: true,
		CreatedAt:        now,
		State:            missioncontrol.CapabilityOnboardingProposalStateApproved,
	}
	if err := missioncontrol.StoreCapabilityOnboardingProposalRecord(root, record); err != nil {
		t.Fatalf("StoreCapabilityOnboardingProposalRecord() error = %v", err)
	}
	if _, err := missioncontrol.StoreWorkspaceSharedStorageCapabilityExposure(root, workspace); err != nil {
		t.Fatalf("StoreWorkspaceSharedStorageCapabilityExposure() error = %v", err)
	}
	if _, err := missioncontrol.StoreWorkspaceSMSPhoneCapabilityExposure(root, workspace); err != nil {
		t.Fatalf("StoreWorkspaceSMSPhoneCapabilityExposure() error = %v", err)
	}
	return root
}

func writeMissionInspectBluetoothNFCCapabilityFixtures(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	home := t.TempDir()
	workspace := filepath.Join(home, "workspace-root")
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.Workspace = workspace
	if err := config.SaveConfig(cfg, filepath.Join(home, ".picobot", "config.json")); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}
	t.Setenv("HOME", home)

	now := time.Date(2026, 4, 19, 12, 0, 0, 0, time.UTC)
	record := missioncontrol.CapabilityOnboardingProposalRecord{
		ProposalID:       "proposal-bluetooth-nfc",
		CapabilityName:   missioncontrol.BluetoothNFCCapabilityName,
		WhyNeeded:        "mission requires local shared Bluetooth/NFC source access",
		MissionFamilies:  []string{"workspace"},
		Risks:            []string{"local Bluetooth/NFC source exposure"},
		Validators:       []string{"shared_storage exposed and committed bluetooth_nfc source file exists and is readable"},
		KillSwitch:       "disable bluetooth_nfc capability exposure and remove committed bluetooth_nfc source reference",
		DataAccessed:     []string{"Bluetooth/NFC"},
		ApprovalRequired: true,
		CreatedAt:        now,
		State:            missioncontrol.CapabilityOnboardingProposalStateApproved,
	}
	if err := missioncontrol.StoreCapabilityOnboardingProposalRecord(root, record); err != nil {
		t.Fatalf("StoreCapabilityOnboardingProposalRecord() error = %v", err)
	}
	if _, err := missioncontrol.StoreWorkspaceSharedStorageCapabilityExposure(root, workspace); err != nil {
		t.Fatalf("StoreWorkspaceSharedStorageCapabilityExposure() error = %v", err)
	}
	if _, err := missioncontrol.StoreWorkspaceBluetoothNFCCapabilityExposure(root, workspace); err != nil {
		t.Fatalf("StoreWorkspaceBluetoothNFCCapabilityExposure() error = %v", err)
	}
	return root
}

func writeMissionInspectBroadAppControlCapabilityFixtures(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	home := t.TempDir()
	workspace := filepath.Join(home, "workspace-root")
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.Workspace = workspace
	if err := config.SaveConfig(cfg, filepath.Join(home, ".picobot", "config.json")); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}
	t.Setenv("HOME", home)

	now := time.Date(2026, 4, 19, 12, 0, 0, 0, time.UTC)
	record := missioncontrol.CapabilityOnboardingProposalRecord{
		ProposalID:       "proposal-broad-app-control",
		CapabilityName:   missioncontrol.BroadAppControlCapabilityName,
		WhyNeeded:        "mission requires local shared broad app control source access",
		MissionFamilies:  []string{"workspace"},
		Risks:            []string{"local broad app control source exposure"},
		Validators:       []string{"shared_storage exposed and committed broad_app_control source file exists and is readable"},
		KillSwitch:       "disable broad_app_control capability exposure and remove committed broad_app_control source reference",
		DataAccessed:     []string{"broad app control"},
		ApprovalRequired: true,
		CreatedAt:        now,
		State:            missioncontrol.CapabilityOnboardingProposalStateApproved,
	}
	if err := missioncontrol.StoreCapabilityOnboardingProposalRecord(root, record); err != nil {
		t.Fatalf("StoreCapabilityOnboardingProposalRecord() error = %v", err)
	}
	if _, err := missioncontrol.StoreWorkspaceSharedStorageCapabilityExposure(root, workspace); err != nil {
		t.Fatalf("StoreWorkspaceSharedStorageCapabilityExposure() error = %v", err)
	}
	if _, err := missioncontrol.StoreWorkspaceBroadAppControlCapabilityExposure(root, workspace); err != nil {
		t.Fatalf("StoreWorkspaceBroadAppControlCapabilityExposure() error = %v", err)
	}
	return root
}

func TestMissionInspectCommandWithValidFilePrintsExpectedSummary(t *testing.T) {
	job := missioncontrol.Job{
		ID:           "job-1",
		MaxAuthority: missioncontrol.AuthorityTierHigh,
		AllowedTools: []string{"read", "write", "write"},
		Plan: missioncontrol.Plan{
			ID: "plan-1",
			Steps: []missioncontrol.Step{
				{
					ID:                "draft",
					Type:              missioncontrol.StepTypeDiscussion,
					RequiredAuthority: missioncontrol.AuthorityTierLow,
					AllowedTools:      []string{"read", "read"},
					RequiresApproval:  true,
					SuccessCriteria:   []string{"share a concise plan"},
				},
				{
					ID:                "build",
					Type:              missioncontrol.StepTypeOneShotCode,
					DependsOn:         []string{"draft", "draft"},
					RequiredAuthority: missioncontrol.AuthorityTierMedium,
					AllowedTools:      []string{"write", "write"},
					SuccessCriteria:   []string{"produce code"},
				},
				{
					ID:                "final",
					Type:              missioncontrol.StepTypeFinalResponse,
					DependsOn:         []string{"build"},
					RequiredAuthority: missioncontrol.AuthorityTierLow,
				},
			},
		},
	}
	path := writeMissionBootstrapJobFile(t, job)

	cmd := NewRootCmd()
	out := &bytes.Buffer{}
	cmd.SetOut(out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mission", "inspect", "--mission-file", path})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	var got missionInspectSummary
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if got.JobID != job.ID {
		t.Fatalf("JobID = %q, want %q", got.JobID, job.ID)
	}
	if got.MaxAuthority != job.MaxAuthority {
		t.Fatalf("MaxAuthority = %q, want %q", got.MaxAuthority, job.MaxAuthority)
	}
	if !reflect.DeepEqual(got.AllowedTools, job.AllowedTools) {
		t.Fatalf("AllowedTools = %v, want %v", got.AllowedTools, job.AllowedTools)
	}
	if len(got.Steps) != len(job.Plan.Steps) {
		t.Fatalf("len(Steps) = %d, want %d", len(got.Steps), len(job.Plan.Steps))
	}
	if got.Steps[0].StepID != "draft" || got.Steps[1].StepID != "build" || got.Steps[2].StepID != "final" {
		t.Fatalf("step order = %#v, want draft/build/final", got.Steps)
	}
	if !reflect.DeepEqual(got.Steps[1].DependsOn, []string{"draft", "draft"}) {
		t.Fatalf("build DependsOn = %v, want duplicate-preserving slice", got.Steps[1].DependsOn)
	}
	if !reflect.DeepEqual(got.Steps[1].AllowedTools, []string{"write", "write"}) {
		t.Fatalf("build AllowedTools = %v, want duplicate-preserving slice", got.Steps[1].AllowedTools)
	}
	if !reflect.DeepEqual(got.Steps[0].SuccessCriteria, []string{"share a concise plan"}) {
		t.Fatalf("draft SuccessCriteria = %v, want [share a concise plan]", got.Steps[0].SuccessCriteria)
	}
	if !reflect.DeepEqual(got.Steps[1].SuccessCriteria, []string{"produce code"}) {
		t.Fatalf("build SuccessCriteria = %v, want [produce code]", got.Steps[1].SuccessCriteria)
	}
	if !reflect.DeepEqual(got.Steps[0].EffectiveAllowedTools, []string{"read"}) {
		t.Fatalf("draft EffectiveAllowedTools = %v, want [read]", got.Steps[0].EffectiveAllowedTools)
	}
	if !reflect.DeepEqual(got.Steps[1].EffectiveAllowedTools, []string{"write"}) {
		t.Fatalf("build EffectiveAllowedTools = %v, want [write]", got.Steps[1].EffectiveAllowedTools)
	}
	if !reflect.DeepEqual(got.Steps[2].EffectiveAllowedTools, []string{"read", "write"}) {
		t.Fatalf("final EffectiveAllowedTools = %v, want [read write]", got.Steps[2].EffectiveAllowedTools)
	}
	if !got.Steps[0].RequiresApproval {
		t.Fatal("draft RequiresApproval = false, want true")
	}
}

func TestMissionInspectCommandWithStepIDReturnsExactlyOneResolvedStep(t *testing.T) {
	job := missioncontrol.Job{
		ID:           "job-1",
		MaxAuthority: missioncontrol.AuthorityTierHigh,
		AllowedTools: []string{"read", "write", "search"},
		Plan: missioncontrol.Plan{
			ID: "plan-1",
			Steps: []missioncontrol.Step{
				{
					ID:                "draft",
					Type:              missioncontrol.StepTypeDiscussion,
					RequiredAuthority: missioncontrol.AuthorityTierLow,
					AllowedTools:      []string{"read"},
				},
				{
					ID:                "build",
					Type:              missioncontrol.StepTypeOneShotCode,
					DependsOn:         []string{"draft", "draft"},
					RequiredAuthority: missioncontrol.AuthorityTierMedium,
					AllowedTools:      []string{"write", "write"},
					SuccessCriteria:   []string{"produce code"},
				},
				{
					ID:        "final",
					Type:      missioncontrol.StepTypeFinalResponse,
					DependsOn: []string{"build"},
				},
			},
		},
	}
	path := writeMissionBootstrapJobFile(t, job)

	cmd := NewRootCmd()
	out := &bytes.Buffer{}
	cmd.SetOut(out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mission", "inspect", "--mission-file", path, "--step-id", "build"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	var got missionInspectSummary
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if got.JobID != job.ID {
		t.Fatalf("JobID = %q, want %q", got.JobID, job.ID)
	}
	if got.MaxAuthority != job.MaxAuthority {
		t.Fatalf("MaxAuthority = %q, want %q", got.MaxAuthority, job.MaxAuthority)
	}
	if !reflect.DeepEqual(got.AllowedTools, job.AllowedTools) {
		t.Fatalf("AllowedTools = %v, want %v", got.AllowedTools, job.AllowedTools)
	}
	if len(got.Steps) != 1 {
		t.Fatalf("len(Steps) = %d, want 1", len(got.Steps))
	}
	if got.Steps[0].StepID != "build" {
		t.Fatalf("StepID = %q, want %q", got.Steps[0].StepID, "build")
	}
	if !reflect.DeepEqual(got.Steps[0].DependsOn, []string{"draft", "draft"}) {
		t.Fatalf("DependsOn = %v, want duplicate-preserving slice", got.Steps[0].DependsOn)
	}
	if !reflect.DeepEqual(got.Steps[0].SuccessCriteria, []string{"produce code"}) {
		t.Fatalf("SuccessCriteria = %v, want [produce code]", got.Steps[0].SuccessCriteria)
	}
}

func TestMissionInspectCommandWithStepIDIncludesResolvedEffectiveAllowedTools(t *testing.T) {
	job := missioncontrol.Job{
		ID:           "job-1",
		MaxAuthority: missioncontrol.AuthorityTierHigh,
		AllowedTools: []string{"read", "write", "search"},
		Plan: missioncontrol.Plan{
			ID: "plan-1",
			Steps: []missioncontrol.Step{
				{
					ID:                "build",
					Type:              missioncontrol.StepTypeOneShotCode,
					RequiredAuthority: missioncontrol.AuthorityTierMedium,
					AllowedTools:      []string{"write", "write"},
				},
				{
					ID:        "final",
					Type:      missioncontrol.StepTypeFinalResponse,
					DependsOn: []string{"build"},
				},
			},
		},
	}
	path := writeMissionBootstrapJobFile(t, job)

	cmd := NewRootCmd()
	out := &bytes.Buffer{}
	cmd.SetOut(out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mission", "inspect", "--mission-file", path, "--step-id", "build"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	var got missionInspectSummary
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if !reflect.DeepEqual(got.Steps[0].EffectiveAllowedTools, []string{"write"}) {
		t.Fatalf("EffectiveAllowedTools = %v, want [write]", got.Steps[0].EffectiveAllowedTools)
	}
}

func TestMissionInspectCommandTreasuryPreflightZeroRefPathUnchanged(t *testing.T) {
	job := testMissionBootstrapJob()
	path := writeMissionBootstrapJobFile(t, job)

	cmd := NewRootCmd()
	out := &bytes.Buffer{}
	cmd.SetOut(out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mission", "inspect", "--mission-file", path, "--step-id", "build"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	var got missionInspectSummary
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if len(got.Steps) != 1 || got.Steps[0].StepID != "build" {
		t.Fatalf("Steps = %#v, want one build step", got.Steps)
	}
	if got.Steps[0].CampaignPreflight != nil {
		t.Fatalf("CampaignPreflight = %#v, want nil for zero-ref path", got.Steps[0].CampaignPreflight)
	}
	if got.Steps[0].TreasuryPreflight != nil {
		t.Fatalf("TreasuryPreflight = %#v, want nil for zero-ref path", got.Steps[0].TreasuryPreflight)
	}
	if !reflect.DeepEqual(got.Steps[0].EffectiveAllowedTools, []string{"read"}) {
		t.Fatalf("EffectiveAllowedTools = %v, want [read]", got.Steps[0].EffectiveAllowedTools)
	}
}

func TestMissionInspectCommandTreasuryStepSurfacesResolvedTreasuryPreflight(t *testing.T) {
	root, treasury, container := writeMissionInspectTreasuryFixtures(t)
	job := testMissionBootstrapJob()
	job.Plan.Steps[0].TreasuryRef = &missioncontrol.TreasuryRef{TreasuryID: treasury.TreasuryID}
	path := writeMissionBootstrapJobFile(t, job)

	cmd := NewRootCmd()
	out := &bytes.Buffer{}
	cmd.SetOut(out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mission", "inspect", "--mission-store-root", root, "--mission-file", path, "--step-id", "build"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	var got missionInspectSummary
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if len(got.Steps) != 1 || got.Steps[0].StepID != "build" {
		t.Fatalf("Steps = %#v, want one build step", got.Steps)
	}
	if got.Steps[0].TreasuryPreflight == nil {
		t.Fatal("TreasuryPreflight = nil, want resolved treasury/container data")
	}
	if got.Steps[0].TreasuryPreflight.Treasury == nil {
		t.Fatal("TreasuryPreflight.Treasury = nil, want resolved treasury record")
	}
	if !reflect.DeepEqual(*got.Steps[0].TreasuryPreflight.Treasury, treasury) {
		t.Fatalf("TreasuryPreflight.Treasury = %#v, want %#v", *got.Steps[0].TreasuryPreflight.Treasury, treasury)
	}
	if !reflect.DeepEqual(got.Steps[0].TreasuryPreflight.Containers, []missioncontrol.FrankContainerRecord{container}) {
		t.Fatalf("TreasuryPreflight.Containers = %#v, want [%#v]", got.Steps[0].TreasuryPreflight.Containers, container)
	}
}

func TestMissionInspectCommandCampaignStepSurfacesResolvedCampaignPreflight(t *testing.T) {
	root, _, container := writeMissionInspectTreasuryFixtures(t)
	campaign := mustStoreMissionInspectCampaignFixture(t, root, container)
	job := testMissionBootstrapJob()
	job.Plan.Steps[0].CampaignRef = &missioncontrol.CampaignRef{CampaignID: campaign.CampaignID}
	path := writeMissionBootstrapJobFile(t, job)

	cmd := NewRootCmd()
	out := &bytes.Buffer{}
	cmd.SetOut(out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mission", "inspect", "--mission-store-root", root, "--mission-file", path, "--step-id", "build"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	var got missionInspectSummary
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if len(got.Steps) != 1 || got.Steps[0].StepID != "build" {
		t.Fatalf("Steps = %#v, want one build step", got.Steps)
	}
	if got.Steps[0].CampaignPreflight == nil || got.Steps[0].CampaignPreflight.Campaign == nil {
		t.Fatalf("CampaignPreflight = %#v, want resolved campaign preflight", got.Steps[0].CampaignPreflight)
	}
	if got.Steps[0].CampaignPreflight.Campaign.CampaignID != campaign.CampaignID {
		t.Fatalf("CampaignPreflight.Campaign.CampaignID = %q, want %q", got.Steps[0].CampaignPreflight.Campaign.CampaignID, campaign.CampaignID)
	}
	if len(got.Steps[0].CampaignPreflight.Identities) != 1 || len(got.Steps[0].CampaignPreflight.Accounts) != 1 || len(got.Steps[0].CampaignPreflight.Containers) != 1 {
		t.Fatalf("CampaignPreflight = %#v, want one identity/account/container", got.Steps[0].CampaignPreflight)
	}
	if got.Steps[0].TreasuryPreflight != nil {
		t.Fatalf("TreasuryPreflight = %#v, want nil on campaign-only path", got.Steps[0].TreasuryPreflight)
	}
}

func TestMissionInspectCommandZohoMailboxBootstrapStepSurfacesResolvedPreflight(t *testing.T) {
	root, identity, account := writeMissionInspectZohoMailboxBootstrapFixtures(t)
	job := testMissionBootstrapJob()
	job.Plan.Steps[0].GovernedExternalTargets = []missioncontrol.AutonomyEligibilityTargetRef{
		{Kind: missioncontrol.EligibilityTargetKindProvider, RegistryID: "provider-mail"},
		{Kind: missioncontrol.EligibilityTargetKindAccountClass, RegistryID: "account-class-mailbox"},
	}
	job.Plan.Steps[0].FrankObjectRefs = []missioncontrol.FrankRegistryObjectRef{
		{Kind: missioncontrol.FrankRegistryObjectKindIdentity, ObjectID: identity.IdentityID},
		{Kind: missioncontrol.FrankRegistryObjectKindAccount, ObjectID: account.AccountID},
	}
	path := writeMissionBootstrapJobFile(t, job)

	cmd := NewRootCmd()
	out := &bytes.Buffer{}
	cmd.SetOut(out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mission", "inspect", "--mission-store-root", root, "--mission-file", path, "--step-id", "build"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	var got missionInspectSummary
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if len(got.Steps) != 1 || got.Steps[0].StepID != "build" {
		t.Fatalf("Steps = %#v, want one build step", got.Steps)
	}
	if got.Steps[0].FrankZohoMailboxBootstrapPreflight == nil {
		t.Fatal("FrankZohoMailboxBootstrapPreflight = nil, want resolved bootstrap pair")
	}
	if got.Steps[0].FrankZohoMailboxBootstrapPreflight.Identity == nil || !reflect.DeepEqual(*got.Steps[0].FrankZohoMailboxBootstrapPreflight.Identity, identity) {
		t.Fatalf("FrankZohoMailboxBootstrapPreflight.Identity = %#v, want %#v", got.Steps[0].FrankZohoMailboxBootstrapPreflight.Identity, identity)
	}
	if got.Steps[0].FrankZohoMailboxBootstrapPreflight.Account == nil || !reflect.DeepEqual(*got.Steps[0].FrankZohoMailboxBootstrapPreflight.Account, account) {
		t.Fatalf("FrankZohoMailboxBootstrapPreflight.Account = %#v, want %#v", got.Steps[0].FrankZohoMailboxBootstrapPreflight.Account, account)
	}
	if got.Steps[0].CampaignPreflight != nil {
		t.Fatalf("CampaignPreflight = %#v, want nil on bootstrap-only path", got.Steps[0].CampaignPreflight)
	}
	if got.Steps[0].TreasuryPreflight != nil {
		t.Fatalf("TreasuryPreflight = %#v, want nil on bootstrap-only path", got.Steps[0].TreasuryPreflight)
	}
}

func TestMissionInspectCommandTelegramOwnerControlStepSurfacesResolvedPreflight(t *testing.T) {
	root, identity, account := writeMissionInspectTelegramOwnerControlFixtures(t)
	job := testMissionBootstrapJob()
	job.Plan.Steps[0].IdentityMode = missioncontrol.IdentityModeOwnerOnlyControl
	job.Plan.Steps[0].FrankObjectRefs = []missioncontrol.FrankRegistryObjectRef{
		{Kind: missioncontrol.FrankRegistryObjectKindIdentity, ObjectID: identity.IdentityID},
		{Kind: missioncontrol.FrankRegistryObjectKindAccount, ObjectID: account.AccountID},
	}
	path := writeMissionBootstrapJobFile(t, job)

	cmd := NewRootCmd()
	out := &bytes.Buffer{}
	cmd.SetOut(out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mission", "inspect", "--mission-store-root", root, "--mission-file", path, "--step-id", "build"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	var got missionInspectSummary
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if len(got.Steps) != 1 || got.Steps[0].StepID != "build" {
		t.Fatalf("Steps = %#v, want one build step", got.Steps)
	}
	if got.Steps[0].FrankTelegramOwnerControlOnboardingPreflight == nil {
		t.Fatal("FrankTelegramOwnerControlOnboardingPreflight = nil, want resolved onboarding bundle")
	}
	if got.Steps[0].FrankTelegramOwnerControlOnboardingPreflight.Identity == nil || !reflect.DeepEqual(*got.Steps[0].FrankTelegramOwnerControlOnboardingPreflight.Identity, identity) {
		t.Fatalf("FrankTelegramOwnerControlOnboardingPreflight.Identity = %#v, want %#v", got.Steps[0].FrankTelegramOwnerControlOnboardingPreflight.Identity, identity)
	}
	if got.Steps[0].FrankTelegramOwnerControlOnboardingPreflight.Account == nil || !reflect.DeepEqual(*got.Steps[0].FrankTelegramOwnerControlOnboardingPreflight.Account, account) {
		t.Fatalf("FrankTelegramOwnerControlOnboardingPreflight.Account = %#v, want %#v", got.Steps[0].FrankTelegramOwnerControlOnboardingPreflight.Account, account)
	}
	if got.Steps[0].CampaignPreflight != nil {
		t.Fatalf("CampaignPreflight = %#v, want nil on telegram owner-control-only path", got.Steps[0].CampaignPreflight)
	}
	if got.Steps[0].TreasuryPreflight != nil {
		t.Fatalf("TreasuryPreflight = %#v, want nil on telegram owner-control-only path", got.Steps[0].TreasuryPreflight)
	}
}

func TestMissionInspectCommandTreasuryPreflightInvalidContainerStateFailsClosed(t *testing.T) {
	root := t.TempDir()
	now := time.Date(2026, 4, 8, 21, 15, 0, 0, time.UTC)
	treasury := missioncontrol.TreasuryRecord{
		RecordVersion:  missioncontrol.StoreRecordVersion,
		TreasuryID:     "treasury-missing-container",
		DisplayName:    "Frank Treasury",
		State:          missioncontrol.TreasuryStateBootstrap,
		ZeroSeedPolicy: missioncontrol.TreasuryZeroSeedPolicyOwnerSeedForbidden,
		ContainerRefs: []missioncontrol.FrankRegistryObjectRef{
			{
				Kind:     missioncontrol.FrankRegistryObjectKindContainer,
				ObjectID: "missing-container",
			},
		},
		CreatedAt: now,
		UpdatedAt: now.Add(time.Minute),
	}
	writeMalformedTreasuryRecordForMainTest(t, root, treasury)

	job := testMissionBootstrapJob()
	job.Plan.Steps[0].TreasuryRef = &missioncontrol.TreasuryRef{TreasuryID: treasury.TreasuryID}
	path := writeMissionBootstrapJobFile(t, job)

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mission", "inspect", "--mission-store-root", root, "--mission-file", path, "--step-id", "build"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want fail-closed inspection failure")
	}
	if !strings.Contains(err.Error(), "failed to resolve mission inspection summary") {
		t.Fatalf("Execute() error = %q, want inspection summary failure", err)
	}
	if !strings.Contains(err.Error(), missioncontrol.ErrFrankContainerRecordNotFound.Error()) {
		t.Fatalf("Execute() error = %q, want missing container rejection", err)
	}
}

func TestMissionInspectCommandMissingCampaignFailsClosed(t *testing.T) {
	job := testMissionBootstrapJob()
	job.Plan.Steps[0].CampaignRef = &missioncontrol.CampaignRef{CampaignID: "campaign-missing"}
	path := writeMissionBootstrapJobFile(t, job)

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mission", "inspect", "--mission-store-root", t.TempDir(), "--mission-file", path, "--step-id", "build"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want missing campaign rejection")
	}
	if !strings.Contains(err.Error(), "failed to resolve mission inspection summary") {
		t.Fatalf("Execute() error = %q, want inspection summary failure", err)
	}
	if !strings.Contains(err.Error(), missioncontrol.ErrCampaignRecordNotFound.Error()) {
		t.Fatalf("Execute() error = %q, want missing campaign rejection", err)
	}
}

func TestMissionInspectCommandWithUnknownStepReturnsClearError(t *testing.T) {
	path := writeMissionBootstrapJobFile(t, testMissionBootstrapJob())

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mission", "inspect", "--mission-file", path, "--step-id", "missing"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "failed to resolve mission inspection summary") {
		t.Fatalf("Execute() error = %q, want clear inspect summary failure", err)
	}
	if !strings.Contains(err.Error(), "E_INVALID_ACTION_FOR_STEP") {
		t.Fatalf("Execute() error = %q, want unknown_step code", err)
	}
	if !strings.Contains(err.Error(), `step "missing" not found in plan`) {
		t.Fatalf("Execute() error = %q, want missing step message", err)
	}
}

func TestMissionInspectCommandWithoutStepIDPreservesExistingBehavior(t *testing.T) {
	path := writeMissionBootstrapJobFile(t, testMissionBootstrapJob())

	cmd := NewRootCmd()
	out := &bytes.Buffer{}
	cmd.SetOut(out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mission", "inspect", "--mission-file", path})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	var got missionInspectSummary
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if got.JobID != "job-1" {
		t.Fatalf("JobID = %q, want %q", got.JobID, "job-1")
	}
	if len(got.Steps) != 2 {
		t.Fatalf("len(Steps) = %d, want 2", len(got.Steps))
	}
	if got.Steps[0].StepID != "build" || got.Steps[1].StepID != "final" {
		t.Fatalf("step order = %#v, want build/final", got.Steps)
	}
}

func TestMissionInspectCommandNotificationsCapabilityReturnsCommittedRecord(t *testing.T) {
	root := writeMissionInspectNotificationsCapabilityFixtures(t)
	job := testMissionBootstrapJob()
	job.Plan.Steps[0].RequiredCapabilities = []string{missioncontrol.NotificationsCapabilityName}
	job.Plan.Steps[0].CapabilityOnboardingProposalRef = &missioncontrol.CapabilityOnboardingProposalRef{
		ProposalID: "proposal-notifications",
	}
	path := writeMissionBootstrapJobFile(t, job)

	cmd := NewRootCmd()
	out := &bytes.Buffer{}
	cmd.SetOut(out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{
		"mission", "inspect",
		"--mission-file", path,
		"--mission-store-root", root,
		"--step-id", "build",
		"--notifications-capability",
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	var got missionInspectNotificationsCapability
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if got.CapabilityID != missioncontrol.NotificationsTelegramCapabilityID {
		t.Fatalf("CapabilityID = %q, want %q", got.CapabilityID, missioncontrol.NotificationsTelegramCapabilityID)
	}
	if !got.Exposed {
		t.Fatal("Exposed = false, want true")
	}
}

func TestMissionInspectCommandNotificationsCapabilityRequiresStoreRoot(t *testing.T) {
	path := writeMissionBootstrapJobFile(t, testMissionBootstrapJob())

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mission", "inspect", "--mission-file", path, "--notifications-capability"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want store-root requirement")
	}
	if !strings.Contains(err.Error(), "--mission-store-root is required with --notifications-capability") {
		t.Fatalf("Execute() error = %q, want store-root requirement", err)
	}
}

func TestMissionInspectCommandNotificationsCapabilityRejectsStepWithoutRequirement(t *testing.T) {
	root := writeMissionInspectNotificationsCapabilityFixtures(t)
	path := writeMissionBootstrapJobFile(t, testMissionBootstrapJob())

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{
		"mission", "inspect",
		"--mission-file", path,
		"--mission-store-root", root,
		"--step-id", "build",
		"--notifications-capability",
	})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want notifications requirement rejection")
	}
	if !strings.Contains(err.Error(), `step "build" does not require notifications capability`) {
		t.Fatalf("Execute() error = %q, want notifications requirement rejection", err)
	}
}

func TestMissionInspectCommandSharedStorageCapabilityReturnsCommittedRecord(t *testing.T) {
	root := writeMissionInspectSharedStorageCapabilityFixtures(t)
	job := testMissionBootstrapJob()
	job.Plan.Steps[0].RequiredCapabilities = []string{missioncontrol.SharedStorageCapabilityName}
	job.Plan.Steps[0].CapabilityOnboardingProposalRef = &missioncontrol.CapabilityOnboardingProposalRef{
		ProposalID: "proposal-shared-storage",
	}
	path := writeMissionBootstrapJobFile(t, job)

	cmd := NewRootCmd()
	out := &bytes.Buffer{}
	cmd.SetOut(out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{
		"mission", "inspect",
		"--mission-file", path,
		"--mission-store-root", root,
		"--step-id", "build",
		"--shared-storage-capability",
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	var got missionInspectSharedStorageCapability
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if got.CapabilityID != missioncontrol.SharedStorageWorkspaceCapabilityID {
		t.Fatalf("CapabilityID = %q, want %q", got.CapabilityID, missioncontrol.SharedStorageWorkspaceCapabilityID)
	}
	if !got.Exposed {
		t.Fatal("Exposed = false, want true")
	}
}

func TestMissionInspectCommandSharedStorageCapabilityRequiresStoreRoot(t *testing.T) {
	path := writeMissionBootstrapJobFile(t, testMissionBootstrapJob())

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mission", "inspect", "--mission-file", path, "--shared-storage-capability"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want store-root requirement")
	}
	if !strings.Contains(err.Error(), "--mission-store-root is required with --shared-storage-capability") {
		t.Fatalf("Execute() error = %q, want store-root requirement", err)
	}
}

func TestMissionInspectCommandSharedStorageCapabilityRejectsStepWithoutRequirement(t *testing.T) {
	root := writeMissionInspectSharedStorageCapabilityFixtures(t)
	path := writeMissionBootstrapJobFile(t, testMissionBootstrapJob())

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{
		"mission", "inspect",
		"--mission-file", path,
		"--mission-store-root", root,
		"--step-id", "build",
		"--shared-storage-capability",
	})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want shared_storage requirement rejection")
	}
	if !strings.Contains(err.Error(), `step "build" does not require shared_storage capability`) {
		t.Fatalf("Execute() error = %q, want shared_storage requirement rejection", err)
	}
}

func TestMissionInspectCommandContactsCapabilityReturnsCommittedRecordAndSource(t *testing.T) {
	root := writeMissionInspectContactsCapabilityFixtures(t)
	job := testMissionBootstrapJob()
	job.Plan.Steps[0].RequiredCapabilities = []string{missioncontrol.ContactsCapabilityName}
	job.Plan.Steps[0].CapabilityOnboardingProposalRef = &missioncontrol.CapabilityOnboardingProposalRef{
		ProposalID: "proposal-contacts",
	}
	path := writeMissionBootstrapJobFile(t, job)

	cmd := NewRootCmd()
	out := &bytes.Buffer{}
	cmd.SetOut(out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{
		"mission", "inspect",
		"--mission-file", path,
		"--mission-store-root", root,
		"--step-id", "build",
		"--contacts-capability",
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	var got missionInspectContactsCapability
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if got.Capability.CapabilityID != missioncontrol.ContactsLocalFileCapabilityID {
		t.Fatalf("Capability.CapabilityID = %q, want %q", got.Capability.CapabilityID, missioncontrol.ContactsLocalFileCapabilityID)
	}
	if !got.Capability.Exposed {
		t.Fatal("Capability.Exposed = false, want true")
	}
	if got.Source.Path != missioncontrol.ContactsLocalFileDefaultPath {
		t.Fatalf("Source.Path = %q, want %q", got.Source.Path, missioncontrol.ContactsLocalFileDefaultPath)
	}
}

func TestMissionInspectCommandContactsCapabilityRequiresStoreRoot(t *testing.T) {
	path := writeMissionBootstrapJobFile(t, testMissionBootstrapJob())

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mission", "inspect", "--mission-file", path, "--contacts-capability"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want store-root requirement")
	}
	if !strings.Contains(err.Error(), "--mission-store-root is required with --contacts-capability") {
		t.Fatalf("Execute() error = %q, want store-root requirement", err)
	}
}

func TestMissionInspectCommandContactsCapabilityRejectsStepWithoutRequirement(t *testing.T) {
	root := writeMissionInspectContactsCapabilityFixtures(t)
	path := writeMissionBootstrapJobFile(t, testMissionBootstrapJob())

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{
		"mission", "inspect",
		"--mission-file", path,
		"--mission-store-root", root,
		"--step-id", "build",
		"--contacts-capability",
	})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want contacts requirement rejection")
	}
	if !strings.Contains(err.Error(), `step "build" does not require contacts capability`) {
		t.Fatalf("Execute() error = %q, want contacts requirement rejection", err)
	}
}

func TestMissionInspectCommandLocationCapabilityReturnsCommittedRecordAndSource(t *testing.T) {
	root := writeMissionInspectLocationCapabilityFixtures(t)
	job := testMissionBootstrapJob()
	job.Plan.Steps[0].RequiredCapabilities = []string{missioncontrol.LocationCapabilityName}
	job.Plan.Steps[0].CapabilityOnboardingProposalRef = &missioncontrol.CapabilityOnboardingProposalRef{
		ProposalID: "proposal-location",
	}
	path := writeMissionBootstrapJobFile(t, job)

	cmd := NewRootCmd()
	out := &bytes.Buffer{}
	cmd.SetOut(out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{
		"mission", "inspect",
		"--mission-file", path,
		"--mission-store-root", root,
		"--step-id", "build",
		"--location-capability",
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	var got missionInspectLocationCapability
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if got.Capability.CapabilityID != missioncontrol.LocationLocalFileCapabilityID {
		t.Fatalf("Capability.CapabilityID = %q, want %q", got.Capability.CapabilityID, missioncontrol.LocationLocalFileCapabilityID)
	}
	if !got.Capability.Exposed {
		t.Fatal("Capability.Exposed = false, want true")
	}
	if got.Source.Path != missioncontrol.LocationLocalFileDefaultPath {
		t.Fatalf("Source.Path = %q, want %q", got.Source.Path, missioncontrol.LocationLocalFileDefaultPath)
	}
}

func TestMissionInspectCommandLocationCapabilityRequiresStoreRoot(t *testing.T) {
	path := writeMissionBootstrapJobFile(t, testMissionBootstrapJob())

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mission", "inspect", "--mission-file", path, "--location-capability"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want store-root requirement")
	}
	if !strings.Contains(err.Error(), "--mission-store-root is required with --location-capability") {
		t.Fatalf("Execute() error = %q, want store-root requirement", err)
	}
}

func TestMissionInspectCommandLocationCapabilityRejectsStepWithoutRequirement(t *testing.T) {
	root := writeMissionInspectLocationCapabilityFixtures(t)
	path := writeMissionBootstrapJobFile(t, testMissionBootstrapJob())

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{
		"mission", "inspect",
		"--mission-file", path,
		"--mission-store-root", root,
		"--step-id", "build",
		"--location-capability",
	})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want location requirement rejection")
	}
	if !strings.Contains(err.Error(), `step "build" does not require location capability`) {
		t.Fatalf("Execute() error = %q, want location requirement rejection", err)
	}
}

func TestMissionInspectCommandCameraCapabilityReturnsCommittedRecordAndSource(t *testing.T) {
	root := writeMissionInspectCameraCapabilityFixtures(t)
	job := testMissionBootstrapJob()
	job.Plan.Steps[0].RequiredCapabilities = []string{missioncontrol.CameraCapabilityName}
	job.Plan.Steps[0].CapabilityOnboardingProposalRef = &missioncontrol.CapabilityOnboardingProposalRef{
		ProposalID: "proposal-camera",
	}
	path := writeMissionBootstrapJobFile(t, job)

	cmd := NewRootCmd()
	out := &bytes.Buffer{}
	cmd.SetOut(out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{
		"mission", "inspect",
		"--mission-file", path,
		"--mission-store-root", root,
		"--step-id", "build",
		"--camera-capability",
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	var got missionInspectCameraCapability
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if got.Capability.CapabilityID != missioncontrol.CameraLocalFileCapabilityID {
		t.Fatalf("Capability.CapabilityID = %q, want %q", got.Capability.CapabilityID, missioncontrol.CameraLocalFileCapabilityID)
	}
	if !got.Capability.Exposed {
		t.Fatal("Capability.Exposed = false, want true")
	}
	if got.Source.Path != missioncontrol.CameraLocalFileDefaultPath {
		t.Fatalf("Source.Path = %q, want %q", got.Source.Path, missioncontrol.CameraLocalFileDefaultPath)
	}
}

func TestMissionInspectCommandCameraCapabilityRequiresStoreRoot(t *testing.T) {
	path := writeMissionBootstrapJobFile(t, testMissionBootstrapJob())

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mission", "inspect", "--mission-file", path, "--camera-capability"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want store-root requirement")
	}
	if !strings.Contains(err.Error(), "--mission-store-root is required with --camera-capability") {
		t.Fatalf("Execute() error = %q, want store-root requirement", err)
	}
}

func TestMissionInspectCommandCameraCapabilityRejectsStepWithoutRequirement(t *testing.T) {
	root := writeMissionInspectCameraCapabilityFixtures(t)
	path := writeMissionBootstrapJobFile(t, testMissionBootstrapJob())

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{
		"mission", "inspect",
		"--mission-file", path,
		"--mission-store-root", root,
		"--step-id", "build",
		"--camera-capability",
	})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want camera requirement rejection")
	}
	if !strings.Contains(err.Error(), `step "build" does not require camera capability`) {
		t.Fatalf("Execute() error = %q, want camera requirement rejection", err)
	}
}

func TestMissionInspectCommandMicrophoneCapabilityReturnsCommittedRecordAndSource(t *testing.T) {
	root := writeMissionInspectMicrophoneCapabilityFixtures(t)
	job := testMissionBootstrapJob()
	job.Plan.Steps[0].RequiredCapabilities = []string{missioncontrol.MicrophoneCapabilityName}
	job.Plan.Steps[0].CapabilityOnboardingProposalRef = &missioncontrol.CapabilityOnboardingProposalRef{
		ProposalID: "proposal-microphone",
	}
	path := writeMissionBootstrapJobFile(t, job)

	cmd := NewRootCmd()
	out := &bytes.Buffer{}
	cmd.SetOut(out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{
		"mission", "inspect",
		"--mission-file", path,
		"--mission-store-root", root,
		"--step-id", "build",
		"--microphone-capability",
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	var got missionInspectMicrophoneCapability
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if got.Capability.CapabilityID != missioncontrol.MicrophoneLocalFileCapabilityID {
		t.Fatalf("Capability.CapabilityID = %q, want %q", got.Capability.CapabilityID, missioncontrol.MicrophoneLocalFileCapabilityID)
	}
	if !got.Capability.Exposed {
		t.Fatal("Capability.Exposed = false, want true")
	}
	if got.Source.Path != missioncontrol.MicrophoneLocalFileDefaultPath {
		t.Fatalf("Source.Path = %q, want %q", got.Source.Path, missioncontrol.MicrophoneLocalFileDefaultPath)
	}
}

func TestMissionInspectCommandMicrophoneCapabilityRequiresStoreRoot(t *testing.T) {
	path := writeMissionBootstrapJobFile(t, testMissionBootstrapJob())

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mission", "inspect", "--mission-file", path, "--microphone-capability"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want store-root requirement")
	}
	if !strings.Contains(err.Error(), "--mission-store-root is required with --microphone-capability") {
		t.Fatalf("Execute() error = %q, want store-root requirement", err)
	}
}

func TestMissionInspectCommandMicrophoneCapabilityRejectsStepWithoutRequirement(t *testing.T) {
	root := writeMissionInspectMicrophoneCapabilityFixtures(t)
	path := writeMissionBootstrapJobFile(t, testMissionBootstrapJob())

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{
		"mission", "inspect",
		"--mission-file", path,
		"--mission-store-root", root,
		"--step-id", "build",
		"--microphone-capability",
	})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want microphone requirement rejection")
	}
	if !strings.Contains(err.Error(), `step "build" does not require microphone capability`) {
		t.Fatalf("Execute() error = %q, want microphone requirement rejection", err)
	}
}

func TestMissionInspectCommandSMSPhoneCapabilityReturnsCommittedRecordAndSource(t *testing.T) {
	root := writeMissionInspectSMSPhoneCapabilityFixtures(t)
	job := testMissionBootstrapJob()
	job.Plan.Steps[0].RequiredCapabilities = []string{missioncontrol.SMSPhoneCapabilityName}
	job.Plan.Steps[0].CapabilityOnboardingProposalRef = &missioncontrol.CapabilityOnboardingProposalRef{
		ProposalID: "proposal-sms-phone",
	}
	path := writeMissionBootstrapJobFile(t, job)

	cmd := NewRootCmd()
	out := &bytes.Buffer{}
	cmd.SetOut(out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{
		"mission", "inspect",
		"--mission-file", path,
		"--mission-store-root", root,
		"--step-id", "build",
		"--sms-phone-capability",
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	var got missionInspectSMSPhoneCapability
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if got.Capability.CapabilityID != missioncontrol.SMSPhoneLocalFileCapabilityID {
		t.Fatalf("Capability.CapabilityID = %q, want %q", got.Capability.CapabilityID, missioncontrol.SMSPhoneLocalFileCapabilityID)
	}
	if !got.Capability.Exposed {
		t.Fatal("Capability.Exposed = false, want true")
	}
	if got.Source.Path != missioncontrol.SMSPhoneLocalFileDefaultPath {
		t.Fatalf("Source.Path = %q, want %q", got.Source.Path, missioncontrol.SMSPhoneLocalFileDefaultPath)
	}
}

func TestMissionInspectCommandSMSPhoneCapabilityRequiresStoreRoot(t *testing.T) {
	path := writeMissionBootstrapJobFile(t, testMissionBootstrapJob())

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mission", "inspect", "--mission-file", path, "--sms-phone-capability"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want store-root requirement")
	}
	if !strings.Contains(err.Error(), "--mission-store-root is required with --sms-phone-capability") {
		t.Fatalf("Execute() error = %q, want store-root requirement", err)
	}
}

func TestMissionInspectCommandSMSPhoneCapabilityRejectsStepWithoutRequirement(t *testing.T) {
	root := writeMissionInspectSMSPhoneCapabilityFixtures(t)
	path := writeMissionBootstrapJobFile(t, testMissionBootstrapJob())

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{
		"mission", "inspect",
		"--mission-file", path,
		"--mission-store-root", root,
		"--step-id", "build",
		"--sms-phone-capability",
	})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want sms_phone requirement rejection")
	}
	if !strings.Contains(err.Error(), `step "build" does not require sms_phone capability`) {
		t.Fatalf("Execute() error = %q, want sms_phone requirement rejection", err)
	}
}

func TestMissionInspectCommandBluetoothNFCCapabilityReturnsCommittedRecordAndSource(t *testing.T) {
	root := writeMissionInspectBluetoothNFCCapabilityFixtures(t)
	job := testMissionBootstrapJob()
	job.Plan.Steps[0].RequiredCapabilities = []string{missioncontrol.BluetoothNFCCapabilityName}
	job.Plan.Steps[0].CapabilityOnboardingProposalRef = &missioncontrol.CapabilityOnboardingProposalRef{
		ProposalID: "proposal-bluetooth-nfc",
	}
	path := writeMissionBootstrapJobFile(t, job)

	cmd := NewRootCmd()
	out := &bytes.Buffer{}
	cmd.SetOut(out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{
		"mission", "inspect",
		"--mission-file", path,
		"--mission-store-root", root,
		"--step-id", "build",
		"--bluetooth-nfc-capability",
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	var got missionInspectBluetoothNFCCapability
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if got.Capability.CapabilityID != missioncontrol.BluetoothNFCLocalFileCapabilityID {
		t.Fatalf("Capability.CapabilityID = %q, want %q", got.Capability.CapabilityID, missioncontrol.BluetoothNFCLocalFileCapabilityID)
	}
	if !got.Capability.Exposed {
		t.Fatal("Capability.Exposed = false, want true")
	}
	if got.Source.Path != missioncontrol.BluetoothNFCLocalFileDefaultPath {
		t.Fatalf("Source.Path = %q, want %q", got.Source.Path, missioncontrol.BluetoothNFCLocalFileDefaultPath)
	}
}

func TestMissionInspectCommandBluetoothNFCCapabilityRequiresStoreRoot(t *testing.T) {
	path := writeMissionBootstrapJobFile(t, testMissionBootstrapJob())

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mission", "inspect", "--mission-file", path, "--bluetooth-nfc-capability"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want store-root requirement")
	}
	if !strings.Contains(err.Error(), "--mission-store-root is required with --bluetooth-nfc-capability") {
		t.Fatalf("Execute() error = %q, want store-root requirement", err)
	}
}

func TestMissionInspectCommandBluetoothNFCCapabilityRejectsStepWithoutRequirement(t *testing.T) {
	root := writeMissionInspectBluetoothNFCCapabilityFixtures(t)
	path := writeMissionBootstrapJobFile(t, testMissionBootstrapJob())

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{
		"mission", "inspect",
		"--mission-file", path,
		"--mission-store-root", root,
		"--step-id", "build",
		"--bluetooth-nfc-capability",
	})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want bluetooth_nfc requirement rejection")
	}
	if !strings.Contains(err.Error(), `step "build" does not require bluetooth_nfc capability`) {
		t.Fatalf("Execute() error = %q, want bluetooth_nfc requirement rejection", err)
	}
}

func TestMissionInspectCommandBroadAppControlCapabilityReturnsCommittedRecordAndSource(t *testing.T) {
	root := writeMissionInspectBroadAppControlCapabilityFixtures(t)
	job := testMissionBootstrapJob()
	job.Plan.Steps[0].RequiredCapabilities = []string{missioncontrol.BroadAppControlCapabilityName}
	job.Plan.Steps[0].CapabilityOnboardingProposalRef = &missioncontrol.CapabilityOnboardingProposalRef{
		ProposalID: "proposal-broad-app-control",
	}
	path := writeMissionBootstrapJobFile(t, job)

	cmd := NewRootCmd()
	out := &bytes.Buffer{}
	cmd.SetOut(out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{
		"mission", "inspect",
		"--mission-file", path,
		"--mission-store-root", root,
		"--step-id", "build",
		"--broad-app-control-capability",
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	var got missionInspectBroadAppControlCapability
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if got.Capability.CapabilityID != missioncontrol.BroadAppControlLocalFileCapabilityID {
		t.Fatalf("Capability.CapabilityID = %q, want %q", got.Capability.CapabilityID, missioncontrol.BroadAppControlLocalFileCapabilityID)
	}
	if !got.Capability.Exposed {
		t.Fatal("Capability.Exposed = false, want true")
	}
	if got.Source.Path != missioncontrol.BroadAppControlLocalFileDefaultPath {
		t.Fatalf("Source.Path = %q, want %q", got.Source.Path, missioncontrol.BroadAppControlLocalFileDefaultPath)
	}
}

func TestMissionInspectCommandBroadAppControlCapabilityRequiresStoreRoot(t *testing.T) {
	path := writeMissionBootstrapJobFile(t, testMissionBootstrapJob())

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mission", "inspect", "--mission-file", path, "--broad-app-control-capability"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want store-root requirement")
	}
	if !strings.Contains(err.Error(), "--mission-store-root is required with --broad-app-control-capability") {
		t.Fatalf("Execute() error = %q, want store-root requirement", err)
	}
}

func TestMissionInspectCommandBroadAppControlCapabilityRejectsStepWithoutRequirement(t *testing.T) {
	root := writeMissionInspectBroadAppControlCapabilityFixtures(t)
	path := writeMissionBootstrapJobFile(t, testMissionBootstrapJob())

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{
		"mission", "inspect",
		"--mission-file", path,
		"--mission-store-root", root,
		"--step-id", "build",
		"--broad-app-control-capability",
	})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want broad_app_control requirement rejection")
	}
	if !strings.Contains(err.Error(), `step "build" does not require broad_app_control capability`) {
		t.Fatalf("Execute() error = %q, want broad_app_control requirement rejection", err)
	}
}

func TestMissionInspectCommandSuccessCriteriaZeroValuePreservesExistingBehavior(t *testing.T) {
	job := missioncontrol.Job{
		ID:           "job-1",
		MaxAuthority: missioncontrol.AuthorityTierLow,
		AllowedTools: []string{"read"},
		Plan: missioncontrol.Plan{
			ID: "plan-1",
			Steps: []missioncontrol.Step{
				{
					ID:                "draft",
					Type:              missioncontrol.StepTypeDiscussion,
					RequiredAuthority: missioncontrol.AuthorityTierLow,
				},
				{
					ID:                "build",
					Type:              missioncontrol.StepTypeOneShotCode,
					RequiredAuthority: missioncontrol.AuthorityTierLow,
					SuccessCriteria:   []string{},
				},
				{
					ID:        "final",
					Type:      missioncontrol.StepTypeFinalResponse,
					DependsOn: []string{"build"},
				},
			},
		},
	}
	path := writeMissionBootstrapJobFile(t, job)

	cmd := NewRootCmd()
	out := &bytes.Buffer{}
	cmd.SetOut(out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mission", "inspect", "--mission-file", path})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	var got missionInspectSummary
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if got.Steps[0].SuccessCriteria != nil {
		t.Fatalf("draft SuccessCriteria = %v, want nil", got.Steps[0].SuccessCriteria)
	}
	if got.Steps[1].SuccessCriteria != nil {
		t.Fatalf("build SuccessCriteria = %v, want nil", got.Steps[1].SuccessCriteria)
	}
	if got.Steps[2].SuccessCriteria != nil {
		t.Fatalf("final SuccessCriteria = %v, want nil", got.Steps[2].SuccessCriteria)
	}
}

func TestMissionInspectCommandWithZeroToolStepPrintsEmptyEffectiveAllowedTools(t *testing.T) {
	job := missioncontrol.Job{
		ID:           "job-1",
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
	path := writeMissionBootstrapJobFile(t, job)

	cmd := NewRootCmd()
	out := &bytes.Buffer{}
	cmd.SetOut(out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mission", "inspect", "--mission-file", path})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	var got missionInspectSummary
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if !reflect.DeepEqual(got.Steps[0].EffectiveAllowedTools, []string{}) {
		t.Fatalf("discuss EffectiveAllowedTools = %v, want empty slice", got.Steps[0].EffectiveAllowedTools)
	}
	if !reflect.DeepEqual(got.Steps[1].EffectiveAllowedTools, []string{}) {
		t.Fatalf("final EffectiveAllowedTools = %v, want empty slice", got.Steps[1].EffectiveAllowedTools)
	}
}

func TestMissionInspectCommandWithMissingFileReturnsError(t *testing.T) {
	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mission", "inspect", "--mission-file", filepath.Join(t.TempDir(), "missing.json")})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "failed to read mission file") {
		t.Fatalf("Execute() error = %q, want missing file message", err)
	}
}

func TestMissionInspectCommandWithInvalidJSONReturnsError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mission.json")
	if err := os.WriteFile(path, []byte("{not-json"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mission", "inspect", "--mission-file", path})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "failed to decode mission file") {
		t.Fatalf("Execute() error = %q, want decode failure", err)
	}
}

func TestMissionInspectCommandWithInvalidMissionReturnsValidationError(t *testing.T) {
	path := writeMissionBootstrapJobFile(t, missioncontrol.Job{
		ID:           "job-1",
		MaxAuthority: missioncontrol.AuthorityTierHigh,
		AllowedTools: []string{"read"},
		Plan:         missioncontrol.Plan{ID: "plan-1"},
	})

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mission", "inspect", "--mission-file", path})

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

func writeMissionInspectTreasuryFixtures(t *testing.T) (string, missioncontrol.TreasuryRecord, missioncontrol.FrankContainerRecord) {
	t.Helper()

	root := t.TempDir()
	now := time.Date(2026, 4, 8, 21, 0, 0, 0, time.UTC)
	target := missioncontrol.AutonomyEligibilityTargetRef{
		Kind:       missioncontrol.EligibilityTargetKindTreasuryContainerClass,
		RegistryID: "container-class-wallet",
	}
	writeMissionInspectEligibilityFixture(t, root, target, missioncontrol.EligibilityLabelAutonomyCompatible, "container-class-wallet", "check-container-class-wallet", now)

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
		State:          missioncontrol.TreasuryStateBootstrap,
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

func writeMissionInspectZohoMailboxBootstrapFixtures(t *testing.T) (string, missioncontrol.FrankIdentityRecord, missioncontrol.FrankAccountRecord) {
	t.Helper()

	root := t.TempDir()
	now := time.Date(2026, 4, 8, 20, 45, 0, 0, time.UTC)
	providerTarget := missioncontrol.AutonomyEligibilityTargetRef{
		Kind:       missioncontrol.EligibilityTargetKindProvider,
		RegistryID: "provider-mail",
	}
	accountTarget := missioncontrol.AutonomyEligibilityTargetRef{
		Kind:       missioncontrol.EligibilityTargetKindAccountClass,
		RegistryID: "account-class-mailbox",
	}
	writeMissionInspectEligibilityFixture(t, root, providerTarget, missioncontrol.EligibilityLabelAutonomyCompatible, "provider-mail.example", "check-provider-mail", now)
	writeMissionInspectEligibilityFixture(t, root, accountTarget, missioncontrol.EligibilityLabelAutonomyCompatible, "account-class-mailbox", "check-account-class-mailbox", now.Add(time.Minute))

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
		State:                "active",
		EligibilityTargetRef: accountTarget,
		CreatedAt:            now.Add(2 * time.Minute).UTC(),
		UpdatedAt:            now.Add(3 * time.Minute).UTC(),
	}
	if err := missioncontrol.StoreFrankAccountRecord(root, account); err != nil {
		t.Fatalf("StoreFrankAccountRecord() error = %v", err)
	}

	return root, identity, account
}

func writeMissionInspectTelegramOwnerControlFixtures(t *testing.T) (string, missioncontrol.FrankIdentityRecord, missioncontrol.FrankAccountRecord) {
	t.Helper()

	root := t.TempDir()
	now := time.Date(2026, 4, 18, 18, 0, 0, 0, time.UTC)
	providerTarget := missioncontrol.AutonomyEligibilityTargetRef{
		Kind:       missioncontrol.EligibilityTargetKindProvider,
		RegistryID: "provider-telegram-owner-control",
	}
	accountTarget := missioncontrol.AutonomyEligibilityTargetRef{
		Kind:       missioncontrol.EligibilityTargetKindAccountClass,
		RegistryID: "account-class-telegram-owner-control",
	}
	writeMissionInspectEligibilityFixture(t, root, providerTarget, missioncontrol.EligibilityLabelAutonomyCompatible, "telegram owner-control", "check-provider-telegram", now)
	writeMissionInspectEligibilityFixture(t, root, accountTarget, missioncontrol.EligibilityLabelAutonomyCompatible, "telegram owner-control account", "check-account-class-telegram", now.Add(time.Minute))

	identity := missioncontrol.FrankIdentityRecord{
		RecordVersion:        missioncontrol.StoreRecordVersion,
		IdentityID:           "identity-telegram-owner-control",
		IdentityKind:         "owner_control_channel",
		DisplayName:          "Telegram Owner Control",
		ProviderOrPlatformID: providerTarget.RegistryID,
		TelegramOwnerControl: &missioncontrol.FrankTelegramOwnerControlIdentity{},
		IdentityMode:         missioncontrol.IdentityModeOwnerOnlyControl,
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
		AccountID:            "account-telegram-owner-control",
		AccountKind:          "owner_control_channel",
		Label:                "Telegram Owner Control",
		ProviderOrPlatformID: providerTarget.RegistryID,
		TelegramOwnerControl: &missioncontrol.FrankTelegramOwnerControlAccount{},
		IdentityID:           identity.IdentityID,
		ControlModel:         "owner_controlled",
		RecoveryModel:        "owner_recoverable",
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

func mustStoreMissionInspectCampaignFixture(t *testing.T, root string, container missioncontrol.FrankContainerRecord) missioncontrol.CampaignRecord {
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
	writeMissionInspectEligibilityFixture(t, root, providerTarget, missioncontrol.EligibilityLabelAutonomyCompatible, "provider-mail.example", "check-provider-mail", now)
	writeMissionInspectEligibilityFixture(t, root, accountTarget, missioncontrol.EligibilityLabelAutonomyCompatible, "account-class-mailbox", "check-account-class-mailbox", now.Add(time.Minute))

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
		State:                   missioncontrol.CampaignStateDraft,
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

func writeMissionInspectEligibilityFixture(t *testing.T, root string, target missioncontrol.AutonomyEligibilityTargetRef, label missioncontrol.EligibilityLabel, targetName string, checkID string, checkedAt time.Time) {
	t.Helper()

	check := missioncontrol.EligibilityCheckRecord{
		CheckID:    checkID,
		TargetKind: target.Kind,
		TargetName: targetName,
		Label:      label,
		CheckedAt:  checkedAt,
	}
	platform := missioncontrol.PlatformRecord{
		PlatformID:       target.RegistryID,
		PlatformName:     targetName,
		TargetClass:      target.Kind,
		EligibilityLabel: label,
		LastCheckID:      checkID,
		Notes:            []string{"registry note"},
		UpdatedAt:        checkedAt.UTC(),
	}

	switch label {
	case missioncontrol.EligibilityLabelAutonomyCompatible:
		check.CanCreateWithoutOwner = true
		check.CanOnboardWithoutOwner = true
		check.CanControlAsAgent = true
		check.CanRecoverAsAgent = true
		check.RulesAsObservedOK = true
		check.Reasons = []string{"autonomy_compatible"}
	case missioncontrol.EligibilityLabelHumanGated:
		check.CanCreateWithoutOwner = false
		check.CanOnboardWithoutOwner = false
		check.CanControlAsAgent = false
		check.CanRecoverAsAgent = false
		check.RequiresHumanOnlyStep = true
		check.RulesAsObservedOK = false
		check.Reasons = []string{string(missioncontrol.AutonomyEligibilityReasonHumanGatedKYCOrCustodialOnboarding)}
	case missioncontrol.EligibilityLabelIneligible:
		check.CanCreateWithoutOwner = false
		check.CanOnboardWithoutOwner = false
		check.CanControlAsAgent = false
		check.CanRecoverAsAgent = false
		check.RequiresOwnerOnlySecretOrID = true
		check.RulesAsObservedOK = false
		check.Reasons = []string{string(missioncontrol.AutonomyEligibilityReasonOwnerIdentityRequired)}
	default:
		t.Fatalf("unsupported eligibility label %q", label)
	}

	if err := missioncontrol.StorePlatformRecord(root, platform); err != nil {
		t.Fatalf("StorePlatformRecord(%s) error = %v", target.RegistryID, err)
	}
	if err := missioncontrol.StoreEligibilityCheckRecord(root, check); err != nil {
		t.Fatalf("StoreEligibilityCheckRecord(%s) error = %v", checkID, err)
	}
}
