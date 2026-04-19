package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"

	"github.com/local/picobot/internal/agent"
	"github.com/local/picobot/internal/agent/memory"
	"github.com/local/picobot/internal/chat"
	"github.com/local/picobot/internal/config"
	"github.com/local/picobot/internal/cron"
	"github.com/local/picobot/internal/missioncontrol"
	"github.com/local/picobot/internal/providers"
)

type scheduledTriggerTestProvider struct {
	content         string
	responses       []string
	lastUserMessage string
	userMessages    []string
	lastToolNames   []string
	calls           int
}

func (p *scheduledTriggerTestProvider) Chat(ctx context.Context, messages []providers.Message, tools []providers.ToolDefinition, model string) (providers.LLMResponse, error) {
	p.calls++
	p.lastToolNames = p.lastToolNames[:0]
	for _, tool := range tools {
		p.lastToolNames = append(p.lastToolNames, tool.Name)
	}
	p.lastUserMessage = ""
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "user" {
			p.lastUserMessage = messages[i].Content
			break
		}
	}
	p.userMessages = append(p.userMessages, p.lastUserMessage)
	if len(p.responses) > 0 {
		content := p.responses[0]
		p.responses = p.responses[1:]
		return providers.LLMResponse{Content: content}, nil
	}
	return providers.LLMResponse{Content: p.content}, nil
}

func (p *scheduledTriggerTestProvider) GetDefaultModel() string { return "scheduled-trigger-test" }

func installScheduledTriggerTestPersistence(root string, ag *agent.AgentLoop) {
	if ag == nil {
		return
	}
	ag.SetMissionStoreRoot(root)
	ag.SetMissionRuntimePersistHook(func(job *missioncontrol.Job, runtime missioncontrol.JobRuntimeState, control *missioncontrol.RuntimeControlContext) error {
		return missioncontrol.PersistProjectedRuntimeState(root, missioncontrol.WriterLockLease{LeaseHolderID: "scheduled-trigger-test"}, job, runtime, control, time.Now().UTC())
	})
}

func assertMainJSONObjectKeys(t *testing.T, object map[string]any, want ...string) {
	t.Helper()

	got := make([]string, 0, len(object))
	for key := range object {
		got = append(got, key)
	}
	sort.Strings(got)

	wantKeys := append([]string(nil), want...)
	sort.Strings(wantKeys)

	if !reflect.DeepEqual(got, wantKeys) {
		t.Fatalf("JSON keys = %#v, want %#v", got, wantKeys)
	}
}

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

func TestMemoryCLI_ReadAppendWriteRecent(t *testing.T) {
	// set HOME to a temp dir so onboard writes to temp
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	// create default config + workspace
	if _, _, err := config.Onboard(); err != nil {
		t.Fatalf("onboard failed: %v", err)
	}

	// run: picobot memory append today -c "hello"
	cmd := NewRootCmd()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"memory", "append", "today", "-c", "hello"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("append today failed: %v", err)
	}

	// verify today's file exists
	cfg, _ := config.LoadConfig()
	ws := cfg.Agents.Defaults.Workspace
	if strings.HasPrefix(ws, "~") {
		home, _ := os.UserHomeDir()
		ws = filepath.Join(home, ws[2:])
	}
	memFile := filepath.Join(ws, "memory")
	files, _ := os.ReadDir(memFile)
	found := false
	for _, f := range files {
		if strings.HasSuffix(f.Name(), ".md") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected memory files, none found in %s", memFile)
	}

	// write long-term
	cmd = NewRootCmd()
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"memory", "write", "long", "-c", "LONGCONTENT"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("write long failed: %v", err)
	}

	// read long-term
	cmd = NewRootCmd()
	readBuf := &bytes.Buffer{}
	cmd.SetOut(readBuf)
	cmd.SetArgs([]string{"memory", "read", "long"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("read long failed: %v", err)
	}
	out := readBuf.String()
	if !strings.Contains(out, "LONGCONTENT") {
		t.Fatalf("expected LONGCONTENT in output, got %q", out)
	}

	// recent days
	cmd = NewRootCmd()
	recentBuf := &bytes.Buffer{}
	cmd.SetOut(recentBuf)
	cmd.SetArgs([]string{"memory", "recent", "--days", "1"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("recent failed: %v", err)
	}
	if recentBuf.String() == "" {
		t.Fatalf("expected recent output, got empty")
	}
}

func TestRouteScheduledTriggerThroughGovernedJobCompletesMissionBoundReminder(t *testing.T) {
	hub := chat.NewHub(10)
	prov := &scheduledTriggerTestProvider{content: "Reminder: stand up and stretch."}
	ag := agent.NewAgentLoop(hub, prov, prov.GetDefaultModel(), 3, t.TempDir(), nil)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	go func() {
		ag.Run(ctx)
		close(done)
	}()
	defer func() {
		cancel()
		<-done
	}()

	trigger := cron.Job{
		ID:      "job-7",
		Name:    "stretch",
		Message: "stand up and stretch",
		FireAt:  time.Date(2026, 4, 12, 15, 4, 5, 123456789, time.UTC),
		Channel: "telegram",
		ChatID:  "chat-123",
	}

	if err := routeScheduledTriggerThroughGovernedJob(ag, hub, trigger); err != nil {
		t.Fatalf("routeScheduledTriggerThroughGovernedJob() error = %v", err)
	}

	select {
	case out := <-hub.Out:
		if out.Channel != "telegram" {
			t.Fatalf("Outbound.Channel = %q, want %q", out.Channel, "telegram")
		}
		if out.ChatID != "chat-123" {
			t.Fatalf("Outbound.ChatID = %q, want %q", out.ChatID, "chat-123")
		}
		if out.Content != "Reminder: stand up and stretch." {
			t.Fatalf("Outbound.Content = %q, want %q", out.Content, "Reminder: stand up and stretch.")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for governed scheduled reminder output")
	}

	wantPrompt := `Scheduled reminder "stretch" fired: stand up and stretch`
	if prov.lastUserMessage != wantPrompt {
		t.Fatalf("provider last user message = %q, want %q", prov.lastUserMessage, wantPrompt)
	}
	if strings.Contains(prov.lastUserMessage, "[Scheduled reminder fired]") {
		t.Fatalf("provider last user message = %q, want governed prompt without legacy raw reinjection wrapper", prov.lastUserMessage)
	}
	if strings.Contains(prov.lastUserMessage, "Please relay this to the user in a friendly way.") {
		t.Fatalf("provider last user message = %q, want governed prompt without legacy relay wrapper", prov.lastUserMessage)
	}
	if len(prov.lastToolNames) != 0 {
		t.Fatalf("provider tool exposure = %v, want no tools for governed scheduled trigger", prov.lastToolNames)
	}

	runtime, ok := ag.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if runtime.JobID != "scheduled-trigger-job-7-20260412T150405.123456789Z" {
		t.Fatalf("MissionRuntimeState().JobID = %q, want deterministic governed scheduler job id", runtime.JobID)
	}
	if runtime.State != missioncontrol.JobStateCompleted {
		t.Fatalf("MissionRuntimeState().State = %q, want %q", runtime.State, missioncontrol.JobStateCompleted)
	}
	if runtime.ActiveStepID != "" {
		t.Fatalf("MissionRuntimeState().ActiveStepID = %q, want empty after completion", runtime.ActiveStepID)
	}
	if len(runtime.CompletedSteps) != 1 || runtime.CompletedSteps[0].StepID != scheduledTriggerStepID {
		t.Fatalf("MissionRuntimeState().CompletedSteps = %+v, want completed scheduled trigger step", runtime.CompletedSteps)
	}
	if runtime.InspectablePlan == nil {
		t.Fatal("MissionRuntimeState().InspectablePlan = nil, want governed plan context")
	}
	if runtime.InspectablePlan.MaxAuthority != missioncontrol.AuthorityTierLow {
		t.Fatalf("MissionRuntimeState().InspectablePlan.MaxAuthority = %q, want %q", runtime.InspectablePlan.MaxAuthority, missioncontrol.AuthorityTierLow)
	}
	if len(runtime.InspectablePlan.AllowedTools) != 0 {
		t.Fatalf("MissionRuntimeState().InspectablePlan.AllowedTools = %v, want empty", runtime.InspectablePlan.AllowedTools)
	}
	foundStepOutputAudit := false
	for _, event := range runtime.AuditHistory {
		if event.ActionClass == missioncontrol.AuditActionClassRuntime && event.ToolName == "step_output" && event.Allowed && event.Result == missioncontrol.AuditResultApplied {
			foundStepOutputAudit = true
			break
		}
	}
	if !foundStepOutputAudit {
		t.Fatalf("MissionRuntimeState().AuditHistory = %+v, want runtime step_output audit evidence", runtime.AuditHistory)
	}
	if _, ok := ag.ActiveMissionStep(); ok {
		t.Fatal("ActiveMissionStep() ok = true, want false after terminal scheduled trigger completion")
	}
}

func TestRouteScheduledTriggerThroughGovernedJobRejectsWhileAnotherMissionIsRunning(t *testing.T) {
	hub := chat.NewHub(10)
	prov := &scheduledTriggerTestProvider{content: "unused"}
	ag := agent.NewAgentLoop(hub, prov, prov.GetDefaultModel(), 3, t.TempDir(), nil)

	existingJob := missioncontrol.Job{
		ID:           "existing-job",
		SpecVersion:  missioncontrol.JobSpecVersionV2,
		State:        missioncontrol.JobStatePending,
		MaxAuthority: missioncontrol.AuthorityTierLow,
		AllowedTools: nil,
		Plan: missioncontrol.Plan{
			ID: "existing-plan",
			Steps: []missioncontrol.Step{
				{
					ID:                "existing-step",
					Type:              missioncontrol.StepTypeFinalResponse,
					RequiredAuthority: missioncontrol.AuthorityTierLow,
				},
			},
		},
	}
	if err := ag.ActivateMissionStep(existingJob, "existing-step"); err != nil {
		t.Fatalf("ActivateMissionStep(existing) error = %v", err)
	}

	trigger := cron.Job{
		ID:      "job-8",
		Name:    "blocked",
		Message: "do not stack",
		FireAt:  time.Date(2026, 4, 12, 16, 0, 0, 0, time.UTC),
		Channel: "telegram",
		ChatID:  "chat-456",
	}
	err := routeScheduledTriggerThroughGovernedJob(ag, hub, trigger)
	if err == nil {
		t.Fatal("routeScheduledTriggerThroughGovernedJob() error = nil, want running-job rejection")
	}
	if !strings.Contains(err.Error(), `runtime job "existing-job" does not match mission job "scheduled-trigger-job-8-20260412T160000.000000000Z"`) {
		t.Fatalf("routeScheduledTriggerThroughGovernedJob() error = %q, want running-job mismatch rejection", err)
	}

	select {
	case msg := <-hub.In:
		t.Fatalf("hub.In unexpectedly queued message = %+v", msg)
	default:
	}

	runtime, ok := ag.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if runtime.JobID != "existing-job" {
		t.Fatalf("MissionRuntimeState().JobID = %q, want %q", runtime.JobID, "existing-job")
	}
	if runtime.State != missioncontrol.JobStateRunning {
		t.Fatalf("MissionRuntimeState().State = %q, want %q", runtime.State, missioncontrol.JobStateRunning)
	}
}

func testBlockingMissionJob() missioncontrol.Job {
	return missioncontrol.Job{
		ID:           "existing-job",
		SpecVersion:  missioncontrol.JobSpecVersionV2,
		State:        missioncontrol.JobStatePending,
		MaxAuthority: missioncontrol.AuthorityTierLow,
		AllowedTools: nil,
		Plan: missioncontrol.Plan{
			ID: "existing-plan",
			Steps: []missioncontrol.Step{
				{
					ID:                "existing-step",
					Type:              missioncontrol.StepTypeFinalResponse,
					RequiredAuthority: missioncontrol.AuthorityTierLow,
				},
			},
		},
	}
}

func TestGovernedScheduledTriggerDeferrerRecordsBlockedTriggerOnce(t *testing.T) {
	hub := chat.NewHub(10)
	prov := &scheduledTriggerTestProvider{content: "unused"}
	ag := agent.NewAgentLoop(hub, prov, prov.GetDefaultModel(), 3, t.TempDir(), nil)
	storeRoot := t.TempDir()
	installScheduledTriggerTestPersistence(storeRoot, ag)

	existingJob := testBlockingMissionJob()
	if err := ag.ActivateMissionStep(existingJob, "existing-step"); err != nil {
		t.Fatalf("ActivateMissionStep(existing) error = %v", err)
	}

	deferrer := newGovernedScheduledTriggerDeferrer(storeRoot)
	trigger := cron.Job{
		ID:      "job-8",
		Name:    "blocked",
		Message: "do not stack",
		FireAt:  time.Date(2026, 4, 12, 16, 0, 0, 0, time.UTC),
		Channel: "telegram",
		ChatID:  "chat-456",
	}

	result, err := deferrer.routeOrDefer(ag, hub, trigger)
	if err != nil {
		t.Fatalf("routeOrDefer() error = %v", err)
	}
	if result != scheduledTriggerHandleDeferred {
		t.Fatalf("routeOrDefer() result = %q, want %q", result, scheduledTriggerHandleDeferred)
	}

	records, err := deferrer.listDeferredTriggers()
	if err != nil {
		t.Fatalf("listDeferredTriggers() error = %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("len(listDeferredTriggers()) = %d, want 1", len(records))
	}
	if records[0].TriggerID != "scheduled-trigger-job-8-20260412T160000.000000000Z" {
		t.Fatalf("deferred TriggerID = %q, want deterministic governed job id", records[0].TriggerID)
	}

	select {
	case msg := <-hub.In:
		t.Fatalf("hub.In unexpectedly queued message = %+v", msg)
	default:
	}

	runtime, ok := ag.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if runtime.JobID != "existing-job" {
		t.Fatalf("MissionRuntimeState().JobID = %q, want %q", runtime.JobID, "existing-job")
	}
	if runtime.State != missioncontrol.JobStateRunning {
		t.Fatalf("MissionRuntimeState().State = %q, want %q", runtime.State, missioncontrol.JobStateRunning)
	}
}

func TestGovernedScheduledTriggerDeferrerDeduplicatesReplay(t *testing.T) {
	hub := chat.NewHub(10)
	prov := &scheduledTriggerTestProvider{content: "unused"}
	ag := agent.NewAgentLoop(hub, prov, prov.GetDefaultModel(), 3, t.TempDir(), nil)
	storeRoot := t.TempDir()
	installScheduledTriggerTestPersistence(storeRoot, ag)

	existingJob := testBlockingMissionJob()
	if err := ag.ActivateMissionStep(existingJob, "existing-step"); err != nil {
		t.Fatalf("ActivateMissionStep(existing) error = %v", err)
	}

	deferrer := newGovernedScheduledTriggerDeferrer(storeRoot)
	trigger := cron.Job{
		ID:      "job-8",
		Name:    "blocked",
		Message: "do not stack",
		FireAt:  time.Date(2026, 4, 12, 16, 0, 0, 0, time.UTC),
		Channel: "telegram",
		ChatID:  "chat-456",
	}

	for i := 0; i < 2; i++ {
		result, err := deferrer.routeOrDefer(ag, hub, trigger)
		if err != nil {
			t.Fatalf("routeOrDefer(replay %d) error = %v", i+1, err)
		}
		if result != scheduledTriggerHandleDeferred {
			t.Fatalf("routeOrDefer(replay %d) result = %q, want %q", i+1, result, scheduledTriggerHandleDeferred)
		}
	}

	records, err := deferrer.listDeferredTriggers()
	if err != nil {
		t.Fatalf("listDeferredTriggers() error = %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("len(listDeferredTriggers()) = %d, want 1 deferred record after replay", len(records))
	}
}

func TestGovernedScheduledTriggerDeferrerDrainsDeferredTriggerThroughOrdinaryGovernedPath(t *testing.T) {
	hub := chat.NewHub(20)
	prov := &scheduledTriggerTestProvider{
		responses: []string{
			"The existing blocker is now cleared.",
			"Reminder: stand up and stretch.",
		},
	}
	ag := agent.NewAgentLoop(hub, prov, prov.GetDefaultModel(), 3, t.TempDir(), nil)
	storeRoot := t.TempDir()
	installScheduledTriggerTestPersistence(storeRoot, ag)

	deferrer := newGovernedScheduledTriggerDeferrer(storeRoot)
	drainErrCh := make(chan error, 1)
	ag.SetMissionRuntimeChangeHook(func() {
		if err := deferrer.drainReady(ag, hub); err != nil {
			select {
			case drainErrCh <- err:
			default:
			}
		}
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	go func() {
		ag.Run(ctx)
		close(done)
	}()
	defer func() {
		cancel()
		<-done
	}()

	existingJob := testBlockingMissionJob()
	if err := ag.ActivateMissionStep(existingJob, "existing-step"); err != nil {
		t.Fatalf("ActivateMissionStep(existing) error = %v", err)
	}

	trigger := cron.Job{
		ID:      "job-9",
		Name:    "stretch",
		Message: "stand up and stretch",
		FireAt:  time.Date(2026, 4, 12, 17, 0, 0, 0, time.UTC),
		Channel: "telegram",
		ChatID:  "chat-789",
	}
	result, err := deferrer.routeOrDefer(ag, hub, trigger)
	if err != nil {
		t.Fatalf("routeOrDefer() error = %v", err)
	}
	if result != scheduledTriggerHandleDeferred {
		t.Fatalf("routeOrDefer() result = %q, want %q", result, scheduledTriggerHandleDeferred)
	}

	hub.In <- chat.Inbound{
		Channel:  "telegram",
		SenderID: "user",
		ChatID:   "chat-789",
		Content:  "finish the existing job",
	}

	select {
	case out := <-hub.Out:
		if out.Content != "The existing blocker is now cleared." {
			t.Fatalf("first Outbound.Content = %q, want %q", out.Content, "The existing blocker is now cleared.")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for existing mission completion output")
	}

	select {
	case out := <-hub.Out:
		if out.Channel != "telegram" {
			t.Fatalf("second Outbound.Channel = %q, want %q", out.Channel, "telegram")
		}
		if out.ChatID != "chat-789" {
			t.Fatalf("second Outbound.ChatID = %q, want %q", out.ChatID, "chat-789")
		}
		if out.Content != "Reminder: stand up and stretch." {
			t.Fatalf("second Outbound.Content = %q, want %q", out.Content, "Reminder: stand up and stretch.")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for deferred governed scheduled reminder output")
	}

	select {
	case err := <-drainErrCh:
		t.Fatalf("drainReady() error = %v", err)
	default:
	}

	records, err := deferrer.listDeferredTriggers()
	if err != nil {
		t.Fatalf("listDeferredTriggers() error = %v", err)
	}
	if len(records) != 0 {
		t.Fatalf("len(listDeferredTriggers()) = %d, want 0 after drain", len(records))
	}

	if len(prov.userMessages) < 2 {
		t.Fatalf("provider userMessages = %v, want existing mission then deferred reminder", prov.userMessages)
	}
	wantDeferredPrompt := `Scheduled reminder "stretch" fired: stand up and stretch`
	if prov.userMessages[1] != wantDeferredPrompt {
		t.Fatalf("provider deferred user message = %q, want %q", prov.userMessages[1], wantDeferredPrompt)
	}
	if len(prov.lastToolNames) != 0 {
		t.Fatalf("provider tool exposure = %v, want no tools for deferred governed scheduled trigger", prov.lastToolNames)
	}

	runtime, ok := ag.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if runtime.JobID != "scheduled-trigger-job-9-20260412T170000.000000000Z" {
		t.Fatalf("MissionRuntimeState().JobID = %q, want deterministic deferred governed job id", runtime.JobID)
	}
	if runtime.State != missioncontrol.JobStateCompleted {
		t.Fatalf("MissionRuntimeState().State = %q, want %q", runtime.State, missioncontrol.JobStateCompleted)
	}
	foundStepOutputAudit := false
	for _, event := range runtime.AuditHistory {
		if event.ActionClass == missioncontrol.AuditActionClassRuntime && event.ToolName == "step_output" && event.Allowed && event.Result == missioncontrol.AuditResultApplied {
			foundStepOutputAudit = true
			break
		}
	}
	if !foundStepOutputAudit {
		t.Fatalf("MissionRuntimeState().AuditHistory = %+v, want runtime step_output audit evidence", runtime.AuditHistory)
	}
}

func TestMemoryCLI_Rank(t *testing.T) {
	// set HOME to a temp dir so onboard writes to temp
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	// create default config + workspace
	if _, _, err := config.Onboard(); err != nil {
		t.Fatalf("onboard failed: %v", err)
	}

	// append some memories
	cfg, _ := config.LoadConfig()
	ws := cfg.Agents.Defaults.Workspace
	if strings.HasPrefix(ws, "~") {
		home, _ := os.UserHomeDir()
		ws = filepath.Join(home, ws[2:])
	}
	mem := memory.NewMemoryStoreWithWorkspace(ws, 100)
	_ = mem.AppendToday("buy milk and eggs")
	_ = mem.AppendToday("call mom tomorrow")
	_ = mem.AppendToday("milkshake recipe")

	// run rank command
	cmd := NewRootCmd()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"memory", "rank", "-q", "milk", "-k", "2"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("rank failed: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "buy milk") {
		t.Fatalf("expected 'buy milk' in output, got: %q", out)
	}
	if !strings.Contains(out, "milkshake") && !strings.Contains(out, "Important facts") {
		t.Fatalf("expected either 'milkshake' or 'Important facts' in output, got: %q", out)
	}
}

func TestAgentCLI_ModelFlag(t *testing.T) {
	// set HOME to a temp dir so onboard writes to temp
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	if _, _, err := config.Onboard(); err != nil {
		t.Fatalf("onboard failed: %v", err)
	}
	// remove OpenAI from config so stub provider is used
	cfgPath, _, _ := config.ResolveDefaultPaths()
	cfg2, _ := config.LoadConfig()
	cfg2.Providers.OpenAI = nil
	_ = config.SaveConfig(cfg2, cfgPath)

	cmd := NewRootCmd()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"agent", "--model", "stub-model", "-m", "hello"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("agent failed: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "(stub) Echo") {
		t.Fatalf("expected stub echo output, got: %q", out)
	}
}

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

func TestMissionPackageLogsCommandReturnsStableSummary(t *testing.T) {
	root := t.TempDir()
	openedAt := time.Date(2026, 4, 5, 9, 0, 0, 0, time.UTC)
	if _, err := missioncontrol.EnsureCurrentLogSegment(root, openedAt); err != nil {
		t.Fatalf("EnsureCurrentLogSegment() error = %v", err)
	}
	if err := os.WriteFile(missioncontrol.StoreCurrentLogPath(root), []byte("gateway line\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(current.log) error = %v", err)
	}

	cmd := NewRootCmd()
	out := &bytes.Buffer{}
	cmd.SetOut(out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mission", "package-logs", "--mission-store-root", root, "--reason", "manual"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	var summary missionPackageLogsSummary
	if err := json.Unmarshal(out.Bytes(), &summary); err != nil {
		t.Fatalf("json.Unmarshal(stdout) error = %v", err)
	}
	if summary.Action != "packaged" {
		t.Fatalf("summary.Action = %q, want %q", summary.Action, "packaged")
	}
	if summary.Reason != missioncontrol.LogPackageReasonManual {
		t.Fatalf("summary.Reason = %q, want %q", summary.Reason, missioncontrol.LogPackageReasonManual)
	}
	if summary.PackageID == "" {
		t.Fatal("summary.PackageID = empty, want package ID")
	}
	if summary.LogRelPath != filepath.ToSlash(filepath.Join("log_packages", summary.PackageID, "gateway.log")) {
		t.Fatalf("summary.LogRelPath = %q, want gateway log relpath", summary.LogRelPath)
	}
	if summary.CurrentLogRelPath != filepath.ToSlash(filepath.Join("logs", "current.log")) {
		t.Fatalf("summary.CurrentLogRelPath = %q, want %q", summary.CurrentLogRelPath, filepath.ToSlash(filepath.Join("logs", "current.log")))
	}
	if summary.CurrentMetaRelPath != filepath.ToSlash(filepath.Join("logs", "current.meta.json")) {
		t.Fatalf("summary.CurrentMetaRelPath = %q, want %q", summary.CurrentMetaRelPath, filepath.ToSlash(filepath.Join("logs", "current.meta.json")))
	}
	if summary.ByteCount == 0 {
		t.Fatal("summary.ByteCount = 0, want packaged byte count")
	}
}

func TestMissionPackageLogsCommandPrunesExpiredPackagesAfterSuccessfulPackaging(t *testing.T) {
	root := t.TempDir()
	now := time.Now().UTC()
	oldPackageID := "20251230T120000.000000000Z-manual"
	if err := writeCommandLogPackageForTest(root, oldPackageID, now.AddDate(0, 0, -91)); err != nil {
		t.Fatalf("writeCommandLogPackageForTest() error = %v", err)
	}
	if _, err := missioncontrol.EnsureCurrentLogSegment(root, now.Add(-time.Hour)); err != nil {
		t.Fatalf("EnsureCurrentLogSegment() error = %v", err)
	}
	if err := os.WriteFile(missioncontrol.StoreCurrentLogPath(root), []byte("gateway line\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(current.log) error = %v", err)
	}

	cmd := NewRootCmd()
	out := &bytes.Buffer{}
	cmd.SetOut(out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mission", "package-logs", "--mission-store-root", root, "--reason", "manual"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	assertPathNotExists(t, missioncontrol.StoreLogPackageDir(root, oldPackageID))
}

func TestMissionPruneStoreCommandReturnsStableSummary(t *testing.T) {
	root := t.TempDir()
	now := time.Now().UTC()
	packageID := "20251231T120000.000000000Z-manual"
	if err := os.MkdirAll(missioncontrol.StoreLogPackageDir(root, packageID), 0o755); err != nil {
		t.Fatalf("MkdirAll(package dir) error = %v", err)
	}
	if err := os.WriteFile(missioncontrol.StoreLogPackageGatewayLogPath(root, packageID), []byte("gateway\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(gateway.log) error = %v", err)
	}
	if err := missioncontrol.StoreLogPackageManifestRecord(root, missioncontrol.LogPackageManifest{
		RecordVersion:   missioncontrol.StoreRecordVersion,
		PackageID:       packageID,
		Reason:          missioncontrol.LogPackageReasonManual,
		CreatedAt:       now.AddDate(0, 0, -91),
		SegmentOpenedAt: now.AddDate(0, 0, -91).Add(-time.Hour),
		SegmentClosedAt: now.AddDate(0, 0, -91),
		LogRelPath:      filepath.ToSlash(filepath.Join("log_packages", packageID, "gateway.log")),
		ByteCount:       int64(len("gateway\n")),
	}); err != nil {
		t.Fatalf("StoreLogPackageManifestRecord() error = %v", err)
	}

	cmd := NewRootCmd()
	out := &bytes.Buffer{}
	cmd.SetOut(out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mission", "prune-store", "--mission-store-root", root})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	var summary missionPruneStoreSummary
	if err := json.Unmarshal(out.Bytes(), &summary); err != nil {
		t.Fatalf("json.Unmarshal(stdout) error = %v", err)
	}
	if summary.Action != "pruned" {
		t.Fatalf("summary.Action = %q, want %q", summary.Action, "pruned")
	}
	if summary.StoreRoot != root {
		t.Fatalf("summary.StoreRoot = %q, want %q", summary.StoreRoot, root)
	}
	if summary.PrunedPackageDirs != 1 {
		t.Fatalf("summary.PrunedPackageDirs = %d, want 1", summary.PrunedPackageDirs)
	}
	if summary.PrunedAuditFiles != 0 || summary.PrunedApprovalRequestFiles != 0 || summary.PrunedApprovalGrantFiles != 0 || summary.PrunedArtifactFiles != 0 || summary.SkippedNonterminalJobTrees != 0 {
		t.Fatalf("summary = %#v, want only packaged dir count", summary)
	}
}

func TestConfigureGatewayMissionStoreLoggingPrunesExpiredPackagesAfterStartupPackaging(t *testing.T) {
	originalGatewayLogNow := gatewayLogNow
	t.Cleanup(func() { gatewayLogNow = originalGatewayLogNow })

	now := time.Date(2026, 4, 6, 0, 1, 0, 0, time.UTC)
	gatewayLogNow = func() time.Time { return now }

	root := t.TempDir()
	oldPackageID := "20251230T120000.000000000Z-reboot"
	if err := writeCommandLogPackageForTest(root, oldPackageID, now.AddDate(0, 0, -91)); err != nil {
		t.Fatalf("writeCommandLogPackageForTest() error = %v", err)
	}
	if _, err := missioncontrol.EnsureCurrentLogSegment(root, now.Add(-time.Hour)); err != nil {
		t.Fatalf("EnsureCurrentLogSegment() error = %v", err)
	}
	if err := os.WriteFile(missioncontrol.StoreCurrentLogPath(root), []byte("startup line\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(current.log) error = %v", err)
	}

	cmd := &cobra.Command{Use: "gateway"}
	addMissionBootstrapFlags(cmd)
	if err := cmd.Flags().Set("mission-store-root", root); err != nil {
		t.Fatalf("Flags().Set(mission-store-root) error = %v", err)
	}

	storeRoot, lease, restore, err := configureGatewayMissionStoreLogging(cmd)
	if err != nil {
		t.Fatalf("configureGatewayMissionStoreLogging() error = %v", err)
	}
	defer restore()

	if storeRoot != root {
		t.Fatalf("configureGatewayMissionStoreLogging().storeRoot = %q, want %q", storeRoot, root)
	}
	if lease.LeaseHolderID == "" {
		t.Fatal("configureGatewayMissionStoreLogging().lease.LeaseHolderID = empty, want gateway lease holder")
	}
	assertPathNotExists(t, missioncontrol.StoreLogPackageDir(root, oldPackageID))
}

func TestMissionPruneStoreCommandReturnsStableNoOpSummary(t *testing.T) {
	root := t.TempDir()

	cmd := NewRootCmd()
	out := &bytes.Buffer{}
	cmd.SetOut(out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mission", "prune-store", "--mission-store-root", root})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	var summary missionPruneStoreSummary
	if err := json.Unmarshal(out.Bytes(), &summary); err != nil {
		t.Fatalf("json.Unmarshal(stdout) error = %v", err)
	}
	if summary.Action != "noop" {
		t.Fatalf("summary.Action = %q, want %q", summary.Action, "noop")
	}
	if summary.StoreRoot != root {
		t.Fatalf("summary.StoreRoot = %q, want %q", summary.StoreRoot, root)
	}
	if summary.PrunedPackageDirs != 0 ||
		summary.PrunedAuditFiles != 0 ||
		summary.PrunedApprovalRequestFiles != 0 ||
		summary.PrunedApprovalGrantFiles != 0 ||
		summary.PrunedArtifactFiles != 0 ||
		summary.SkippedNonterminalJobTrees != 0 {
		t.Fatalf("summary = %#v, want zero-count no-op", summary)
	}
}

func TestConfigureGatewayMissionStoreLoggingRoutesStdlibLoggerIntoActiveSegment(t *testing.T) {
	root := t.TempDir()
	cmd := &cobra.Command{Use: "gateway"}
	addMissionBootstrapFlags(cmd)
	if err := cmd.Flags().Set("mission-store-root", root); err != nil {
		t.Fatalf("Flags().Set(mission-store-root) error = %v", err)
	}

	storeRoot, lease, restore, err := configureGatewayMissionStoreLogging(cmd)
	if err != nil {
		t.Fatalf("configureGatewayMissionStoreLogging() error = %v", err)
	}
	defer restore()

	if storeRoot != root {
		t.Fatalf("configureGatewayMissionStoreLogging().storeRoot = %q, want %q", storeRoot, root)
	}
	if lease.LeaseHolderID == "" {
		t.Fatal("configureGatewayMissionStoreLogging().lease.LeaseHolderID = empty, want gateway lease holder")
	}

	log.Printf("gateway logger line")

	data, err := os.ReadFile(missioncontrol.StoreCurrentLogPath(root))
	if err != nil {
		t.Fatalf("ReadFile(current.log) error = %v", err)
	}
	if !strings.Contains(string(data), "gateway logger line") {
		t.Fatalf("ReadFile(current.log) = %q, want logger line", string(data))
	}
}

func writeCommandLogPackageForTest(root string, packageID string, createdAt time.Time) error {
	if err := os.MkdirAll(missioncontrol.StoreLogPackageDir(root, packageID), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(missioncontrol.StoreLogPackageGatewayLogPath(root, packageID), []byte("gateway\n"), 0o644); err != nil {
		return err
	}
	return missioncontrol.StoreLogPackageManifestRecord(root, missioncontrol.LogPackageManifest{
		RecordVersion:   missioncontrol.StoreRecordVersion,
		PackageID:       packageID,
		Reason:          missioncontrol.LogPackageReasonManual,
		CreatedAt:       createdAt,
		SegmentOpenedAt: createdAt.Add(-time.Hour),
		SegmentClosedAt: createdAt,
		LogRelPath:      filepath.ToSlash(filepath.Join("log_packages", packageID, "gateway.log")),
		ByteCount:       int64(len("gateway\n")),
	})
}

func assertPathNotExists(t *testing.T, path string) {
	t.Helper()

	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return
	} else if err != nil {
		t.Fatalf("Stat(%q) error = %v, want os.ErrNotExist", path, err)
	}
	t.Fatalf("Stat(%q) error = nil, want os.ErrNotExist", path)
}

func TestConfigureGatewayMissionStoreLoggingWithoutStoreRootPreservesExistingLoggerBehavior(t *testing.T) {
	logBuf, restoreStandardLogger := captureStandardLogger(t)
	defer restoreStandardLogger()

	cmd := &cobra.Command{Use: "gateway"}
	addMissionBootstrapFlags(cmd)
	previousWriter := log.Writer()

	storeRoot, lease, restore, err := configureGatewayMissionStoreLogging(cmd)
	if err != nil {
		t.Fatalf("configureGatewayMissionStoreLogging() error = %v", err)
	}
	defer restore()

	if storeRoot != "" {
		t.Fatalf("configureGatewayMissionStoreLogging().storeRoot = %q, want empty", storeRoot)
	}
	if lease.LeaseHolderID != "" {
		t.Fatalf("configureGatewayMissionStoreLogging().lease = %#v, want zero lease", lease)
	}
	if log.Writer() != previousWriter {
		t.Fatal("configureGatewayMissionStoreLogging() changed logger output without a store root")
	}

	log.Printf("fallback logger line")
	if !strings.Contains(logBuf.String(), "fallback logger line") {
		t.Fatalf("log output = %q, want fallback logger line", logBuf.String())
	}
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

func TestMissionInspectCommandSlackOwnerControlStepSurfacesResolvedPreflight(t *testing.T) {
	root, identity, account := writeMissionInspectSlackOwnerControlFixtures(t)
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
	if got.Steps[0].FrankSlackOwnerControlOnboardingPreflight == nil {
		t.Fatal("FrankSlackOwnerControlOnboardingPreflight = nil, want resolved onboarding bundle")
	}
	if got.Steps[0].FrankSlackOwnerControlOnboardingPreflight.Identity == nil || !reflect.DeepEqual(*got.Steps[0].FrankSlackOwnerControlOnboardingPreflight.Identity, identity) {
		t.Fatalf("FrankSlackOwnerControlOnboardingPreflight.Identity = %#v, want %#v", got.Steps[0].FrankSlackOwnerControlOnboardingPreflight.Identity, identity)
	}
	if got.Steps[0].FrankSlackOwnerControlOnboardingPreflight.Account == nil || !reflect.DeepEqual(*got.Steps[0].FrankSlackOwnerControlOnboardingPreflight.Account, account) {
		t.Fatalf("FrankSlackOwnerControlOnboardingPreflight.Account = %#v, want %#v", got.Steps[0].FrankSlackOwnerControlOnboardingPreflight.Account, account)
	}
	if got.Steps[0].CampaignPreflight != nil {
		t.Fatalf("CampaignPreflight = %#v, want nil on slack owner-control-only path", got.Steps[0].CampaignPreflight)
	}
	if got.Steps[0].TreasuryPreflight != nil {
		t.Fatalf("TreasuryPreflight = %#v, want nil on slack owner-control-only path", got.Steps[0].TreasuryPreflight)
	}
}

func TestMissionInspectCommandDiscordOwnerControlStepSurfacesResolvedPreflight(t *testing.T) {
	root, identity, account := writeMissionInspectDiscordOwnerControlFixtures(t)
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
	if got.Steps[0].FrankDiscordOwnerControlOnboardingPreflight == nil {
		t.Fatal("FrankDiscordOwnerControlOnboardingPreflight = nil, want resolved onboarding bundle")
	}
	if got.Steps[0].FrankDiscordOwnerControlOnboardingPreflight.Identity == nil || !reflect.DeepEqual(*got.Steps[0].FrankDiscordOwnerControlOnboardingPreflight.Identity, identity) {
		t.Fatalf("FrankDiscordOwnerControlOnboardingPreflight.Identity = %#v, want %#v", got.Steps[0].FrankDiscordOwnerControlOnboardingPreflight.Identity, identity)
	}
	if got.Steps[0].FrankDiscordOwnerControlOnboardingPreflight.Account == nil || !reflect.DeepEqual(*got.Steps[0].FrankDiscordOwnerControlOnboardingPreflight.Account, account) {
		t.Fatalf("FrankDiscordOwnerControlOnboardingPreflight.Account = %#v, want %#v", got.Steps[0].FrankDiscordOwnerControlOnboardingPreflight.Account, account)
	}
	if got.Steps[0].CampaignPreflight != nil {
		t.Fatalf("CampaignPreflight = %#v, want nil on discord owner-control-only path", got.Steps[0].CampaignPreflight)
	}
	if got.Steps[0].TreasuryPreflight != nil {
		t.Fatalf("TreasuryPreflight = %#v, want nil on discord owner-control-only path", got.Steps[0].TreasuryPreflight)
	}
}

func TestMissionInspectCommandWhatsAppOwnerControlStepSurfacesResolvedPreflight(t *testing.T) {
	root, identity, account := writeMissionInspectWhatsAppOwnerControlFixtures(t)
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
	if got.Steps[0].FrankWhatsAppOwnerControlOnboardingPreflight == nil {
		t.Fatal("FrankWhatsAppOwnerControlOnboardingPreflight = nil, want resolved onboarding bundle")
	}
	if got.Steps[0].FrankWhatsAppOwnerControlOnboardingPreflight.Identity == nil || !reflect.DeepEqual(*got.Steps[0].FrankWhatsAppOwnerControlOnboardingPreflight.Identity, identity) {
		t.Fatalf("FrankWhatsAppOwnerControlOnboardingPreflight.Identity = %#v, want %#v", got.Steps[0].FrankWhatsAppOwnerControlOnboardingPreflight.Identity, identity)
	}
	if got.Steps[0].FrankWhatsAppOwnerControlOnboardingPreflight.Account == nil || !reflect.DeepEqual(*got.Steps[0].FrankWhatsAppOwnerControlOnboardingPreflight.Account, account) {
		t.Fatalf("FrankWhatsAppOwnerControlOnboardingPreflight.Account = %#v, want %#v", got.Steps[0].FrankWhatsAppOwnerControlOnboardingPreflight.Account, account)
	}
	if got.Steps[0].CampaignPreflight != nil {
		t.Fatalf("CampaignPreflight = %#v, want nil on whatsapp owner-control-only path", got.Steps[0].CampaignPreflight)
	}
	if got.Steps[0].TreasuryPreflight != nil {
		t.Fatalf("TreasuryPreflight = %#v, want nil on whatsapp owner-control-only path", got.Steps[0].TreasuryPreflight)
	}
}

func TestMissionInspectCommandGitHubOnboardingStepSurfacesResolvedPreflight(t *testing.T) {
	root, identity, account := writeMissionInspectGitHubFixtures(t)
	job := testMissionBootstrapJob()
	job.Plan.Steps[0].IdentityMode = missioncontrol.IdentityModeAgentAlias
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
	if got.Steps[0].FrankGitHubOnboardingPreflight == nil {
		t.Fatal("FrankGitHubOnboardingPreflight = nil, want resolved onboarding bundle")
	}
	if got.Steps[0].FrankGitHubOnboardingPreflight.Identity == nil || !reflect.DeepEqual(*got.Steps[0].FrankGitHubOnboardingPreflight.Identity, identity) {
		t.Fatalf("FrankGitHubOnboardingPreflight.Identity = %#v, want %#v", got.Steps[0].FrankGitHubOnboardingPreflight.Identity, identity)
	}
	if got.Steps[0].FrankGitHubOnboardingPreflight.Account == nil || !reflect.DeepEqual(*got.Steps[0].FrankGitHubOnboardingPreflight.Account, account) {
		t.Fatalf("FrankGitHubOnboardingPreflight.Account = %#v, want %#v", got.Steps[0].FrankGitHubOnboardingPreflight.Account, account)
	}
	if got.Steps[0].CampaignPreflight != nil {
		t.Fatalf("CampaignPreflight = %#v, want nil on github onboarding-only path", got.Steps[0].CampaignPreflight)
	}
	if got.Steps[0].TreasuryPreflight != nil {
		t.Fatalf("TreasuryPreflight = %#v, want nil on github onboarding-only path", got.Steps[0].TreasuryPreflight)
	}
}

func TestMissionInspectCommandStripeOnboardingStepSurfacesResolvedPreflight(t *testing.T) {
	root, identity, account := writeMissionInspectStripeFixtures(t)
	job := testMissionBootstrapJob()
	job.Plan.Steps[0].IdentityMode = missioncontrol.IdentityModeAgentAlias
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
	if got.Steps[0].FrankStripeOnboardingPreflight == nil {
		t.Fatal("FrankStripeOnboardingPreflight = nil, want resolved onboarding bundle")
	}
	if got.Steps[0].FrankStripeOnboardingPreflight.Identity == nil || !reflect.DeepEqual(*got.Steps[0].FrankStripeOnboardingPreflight.Identity, identity) {
		t.Fatalf("FrankStripeOnboardingPreflight.Identity = %#v, want %#v", got.Steps[0].FrankStripeOnboardingPreflight.Identity, identity)
	}
	if got.Steps[0].FrankStripeOnboardingPreflight.Account == nil || !reflect.DeepEqual(*got.Steps[0].FrankStripeOnboardingPreflight.Account, account) {
		t.Fatalf("FrankStripeOnboardingPreflight.Account = %#v, want %#v", got.Steps[0].FrankStripeOnboardingPreflight.Account, account)
	}
	if got.Steps[0].CampaignPreflight != nil {
		t.Fatalf("CampaignPreflight = %#v, want nil on stripe onboarding-only path", got.Steps[0].CampaignPreflight)
	}
	if got.Steps[0].TreasuryPreflight != nil {
		t.Fatalf("TreasuryPreflight = %#v, want nil on stripe onboarding-only path", got.Steps[0].TreasuryPreflight)
	}
}

func TestMissionInspectCommandPayPalOnboardingStepSurfacesResolvedPreflight(t *testing.T) {
	root, identity, account := writeMissionInspectPayPalFixtures(t)
	job := testMissionBootstrapJob()
	job.Plan.Steps[0].IdentityMode = missioncontrol.IdentityModeAgentAlias
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
	if got.Steps[0].FrankPayPalOnboardingPreflight == nil {
		t.Fatal("FrankPayPalOnboardingPreflight = nil, want resolved onboarding bundle")
	}
	if got.Steps[0].FrankPayPalOnboardingPreflight.Identity == nil || !reflect.DeepEqual(*got.Steps[0].FrankPayPalOnboardingPreflight.Identity, identity) {
		t.Fatalf("FrankPayPalOnboardingPreflight.Identity = %#v, want %#v", got.Steps[0].FrankPayPalOnboardingPreflight.Identity, identity)
	}
	if got.Steps[0].FrankPayPalOnboardingPreflight.Account == nil || !reflect.DeepEqual(*got.Steps[0].FrankPayPalOnboardingPreflight.Account, account) {
		t.Fatalf("FrankPayPalOnboardingPreflight.Account = %#v, want %#v", got.Steps[0].FrankPayPalOnboardingPreflight.Account, account)
	}
	if got.Steps[0].CampaignPreflight != nil {
		t.Fatalf("CampaignPreflight = %#v, want nil on paypal onboarding-only path", got.Steps[0].CampaignPreflight)
	}
	if got.Steps[0].TreasuryPreflight != nil {
		t.Fatalf("TreasuryPreflight = %#v, want nil on paypal onboarding-only path", got.Steps[0].TreasuryPreflight)
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

func TestMissionSetStepCommandInvalidControlPathReturnsClearError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "control-dir")
	if err := os.Mkdir(path, 0o755); err != nil {
		t.Fatalf("Mkdir() error = %v", err)
	}

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mission", "set-step", "--control-file", path, "--step-id", "final"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "failed to write mission step control file") {
		t.Fatalf("Execute() error = %q, want write failure", err)
	}
	assertNoAtomicTempFiles(t, dir, filepath.Base(path))
}

func TestMissionSetStepCommandWithoutStatusFilePreservesCurrentBehavior(t *testing.T) {
	path := filepath.Join(t.TempDir(), "control.json")

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mission", "set-step", "--control-file", path, "--step-id", "final"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	control := readMissionStepControlFile(t, path)
	if control.StepID != "final" {
		t.Fatalf("StepID = %q, want %q", control.StepID, "final")
	}
}

func TestMissionSetStepCommandLeavesNoTempFileOnSuccess(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "control.json")

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mission", "set-step", "--control-file", path, "--step-id", "final"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	control := readMissionStepControlFile(t, path)
	if control.StepID != "final" {
		t.Fatalf("StepID = %q, want %q", control.StepID, "final")
	}
	assertNoAtomicTempFiles(t, dir, filepath.Base(path))
}

func TestMissionSetStepCommandWritesUpdatedAt(t *testing.T) {
	path := filepath.Join(t.TempDir(), "control.json")

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mission", "set-step", "--control-file", path, "--step-id", "final"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	control := readMissionStepControlFile(t, path)
	if control.UpdatedAt == "" {
		t.Fatal("UpdatedAt = empty string, want RFC3339Nano timestamp")
	}

	parsed, err := time.Parse(time.RFC3339Nano, control.UpdatedAt)
	if err != nil {
		t.Fatalf("time.Parse() error = %v", err)
	}
	if _, offset := parsed.Zone(); offset != 0 {
		t.Fatalf("UpdatedAt offset = %d, want 0", offset)
	}
}

func TestMissionSetStepCommandWithStatusFileWaitsWhenMatchingSnapshotUpdatedAtIsUnchanged(t *testing.T) {
	controlPath := filepath.Join(t.TempDir(), "control.json")
	statusPath := filepath.Join(t.TempDir(), "status.json")
	writeMissionStatusSnapshotFile(t, statusPath, missionStatusSnapshot{
		Active:    true,
		StepID:    "final",
		UpdatedAt: "2026-03-20T12:00:00Z",
	})

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{
		"mission", "set-step",
		"--control-file", controlPath,
		"--step-id", "final",
		"--status-file", statusPath,
		"--wait-timeout", "250ms",
	})

	done := make(chan error, 1)
	go func() {
		done <- cmd.Execute()
	}()

	select {
	case err := <-done:
		t.Fatalf("Execute() returned before fresh status update: %v", err)
	case <-time.After(20 * time.Millisecond):
	}

	writeMissionStatusSnapshotFile(t, statusPath, missionStatusSnapshot{
		Active:    true,
		StepID:    "final",
		UpdatedAt: "2026-03-20T12:00:01Z",
	})

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Execute() did not return after fresh status update")
	}

	control := readMissionStepControlFile(t, controlPath)
	if control.StepID != "final" {
		t.Fatalf("StepID = %q, want %q", control.StepID, "final")
	}
}

func TestMissionSetStepCommandWithStatusFileWithoutMissionFilePreservesCurrentBehavior(t *testing.T) {
	controlPath := filepath.Join(t.TempDir(), "control.json")
	statusPath := filepath.Join(t.TempDir(), "status.json")
	writeMissionStatusSnapshotFile(t, statusPath, missionStatusSnapshot{
		Active:    true,
		JobID:     "other-job",
		StepID:    "final",
		UpdatedAt: "2026-03-20T12:00:00Z",
	})

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{
		"mission", "set-step",
		"--control-file", controlPath,
		"--step-id", "final",
		"--status-file", statusPath,
		"--wait-timeout", "250ms",
	})

	done := make(chan error, 1)
	go func() {
		done <- cmd.Execute()
	}()

	select {
	case err := <-done:
		t.Fatalf("Execute() returned before fresh status update: %v", err)
	case <-time.After(20 * time.Millisecond):
	}

	writeMissionStatusSnapshotFile(t, statusPath, missionStatusSnapshot{
		Active:    true,
		JobID:     "different-job",
		StepID:    "final",
		UpdatedAt: "2026-03-20T12:00:01Z",
	})

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Execute() did not return after fresh status update")
	}
}

func TestMissionSetStepCommandWithStatusFileSucceedsWhenSnapshotChangesBeforeTimeout(t *testing.T) {
	controlPath := filepath.Join(t.TempDir(), "control.json")
	statusPath := filepath.Join(t.TempDir(), "status.json")
	writeMissionStatusSnapshotFile(t, statusPath, missionStatusSnapshot{
		Active:    true,
		StepID:    "build",
		UpdatedAt: "2026-03-20T12:00:00Z",
	})

	errCh := make(chan error, 1)
	go func() {
		time.Sleep(25 * time.Millisecond)
		data, err := json.Marshal(missionStatusSnapshot{
			Active:    true,
			StepID:    "final",
			UpdatedAt: "2026-03-20T12:00:01Z",
		})
		if err != nil {
			errCh <- err
			return
		}
		errCh <- os.WriteFile(statusPath, data, 0o644)
	}()

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{
		"mission", "set-step",
		"--control-file", controlPath,
		"--step-id", "final",
		"--status-file", statusPath,
		"--wait-timeout", "250ms",
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if err := <-errCh; err != nil {
		t.Fatalf("status update error = %v", err)
	}

	control := readMissionStepControlFile(t, controlPath)
	if control.StepID != "final" {
		t.Fatalf("StepID = %q, want %q", control.StepID, "final")
	}
}

func TestMissionSetStepCommandWithMissionFileAndStatusFileSucceedsWhenFreshSnapshotMatchesStepAndJob(t *testing.T) {
	controlPath := filepath.Join(t.TempDir(), "control.json")
	statusPath := filepath.Join(t.TempDir(), "status.json")
	job := testMissionBootstrapJob()
	missionPath := writeMissionBootstrapJobFile(t, job)
	logBuf, restoreLog := captureStandardLogger(t)
	defer restoreLog()
	writeMissionStatusSnapshotFile(t, statusPath, missionStatusSnapshot{
		Active:       true,
		JobID:        "other-job",
		StepID:       "build",
		StepType:     string(missioncontrol.StepTypeOneShotCode),
		AllowedTools: []string{"read"},
		UpdatedAt:    "2026-03-20T12:00:00Z",
	})

	done := make(chan error, 1)
	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{
		"mission", "set-step",
		"--control-file", controlPath,
		"--step-id", "final",
		"--mission-file", missionPath,
		"--status-file", statusPath,
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
		JobID:        job.ID,
		StepID:       "final",
		StepType:     string(missioncontrol.StepTypeFinalResponse),
		AllowedTools: []string{"read"},
		UpdatedAt:    "2026-03-20T12:00:01Z",
	})

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Execute() did not return after matching status update")
	}

	logOutput := logBuf.String()
	if !strings.Contains(logOutput, "mission set-step status confirmation succeeded") {
		t.Fatalf("log output = %q, want set-step success log", logOutput)
	}
	if !strings.Contains(logOutput, `job_id="`+job.ID+`"`) {
		t.Fatalf("log output = %q, want job_id", logOutput)
	}
	if !strings.Contains(logOutput, `step_id="final"`) {
		t.Fatalf("log output = %q, want step_id", logOutput)
	}
	if !strings.Contains(logOutput, `control_file="`+controlPath+`"`) {
		t.Fatalf("log output = %q, want control file path", logOutput)
	}
	if !strings.Contains(logOutput, `status_file="`+statusPath+`"`) {
		t.Fatalf("log output = %q, want status file path", logOutput)
	}
}

func TestMissionSetStepCommandWithMissionFileAndStatusFileWaitsWhenStepMatchesButJobDoesNot(t *testing.T) {
	controlPath := filepath.Join(t.TempDir(), "control.json")
	statusPath := filepath.Join(t.TempDir(), "status.json")
	job := testMissionBootstrapJob()
	missionPath := writeMissionBootstrapJobFile(t, job)
	writeMissionStatusSnapshotFile(t, statusPath, missionStatusSnapshot{
		Active:       true,
		JobID:        "other-job",
		StepID:       "build",
		StepType:     string(missioncontrol.StepTypeOneShotCode),
		AllowedTools: []string{"read"},
		UpdatedAt:    "2026-03-20T12:00:00Z",
	})

	done := make(chan error, 1)
	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{
		"mission", "set-step",
		"--control-file", controlPath,
		"--step-id", "final",
		"--mission-file", missionPath,
		"--status-file", statusPath,
		"--wait-timeout", "250ms",
	})
	go func() {
		done <- cmd.Execute()
	}()

	select {
	case err := <-done:
		t.Fatalf("Execute() returned before status update: %v", err)
	case <-time.After(20 * time.Millisecond):
	}

	writeMissionStatusSnapshotFile(t, statusPath, missionStatusSnapshot{
		Active:       true,
		JobID:        "other-job",
		StepID:       "final",
		StepType:     string(missioncontrol.StepTypeFinalResponse),
		AllowedTools: []string{"read"},
		UpdatedAt:    "2026-03-20T12:00:01Z",
	})

	select {
	case err := <-done:
		t.Fatalf("Execute() returned while job_id was still mismatched: %v", err)
	case <-time.After(60 * time.Millisecond):
	}

	writeMissionStatusSnapshotFile(t, statusPath, missionStatusSnapshot{
		Active:       true,
		JobID:        job.ID,
		StepID:       "final",
		StepType:     string(missioncontrol.StepTypeFinalResponse),
		AllowedTools: []string{"read"},
		UpdatedAt:    "2026-03-20T12:00:02Z",
	})

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Execute() did not return after matching job_id update")
	}
}

func TestMissionSetStepCommandWithMissionFileAndStatusFileTimesOutWhenJobIDNeverMatches(t *testing.T) {
	controlPath := filepath.Join(t.TempDir(), "control.json")
	statusPath := filepath.Join(t.TempDir(), "status.json")
	job := testMissionBootstrapJob()
	missionPath := writeMissionBootstrapJobFile(t, job)
	writeMissionStatusSnapshotFile(t, statusPath, missionStatusSnapshot{
		Active:       true,
		JobID:        "other-job",
		StepID:       "final",
		StepType:     string(missioncontrol.StepTypeFinalResponse),
		AllowedTools: []string{"read"},
		UpdatedAt:    "2026-03-20T12:00:00Z",
	})

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{
		"mission", "set-step",
		"--control-file", controlPath,
		"--step-id", "final",
		"--mission-file", missionPath,
		"--status-file", statusPath,
		"--wait-timeout", "75ms",
	})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "timed out waiting") {
		t.Fatalf("Execute() error = %q, want timeout message", err)
	}
	if !strings.Contains(err.Error(), `job_id="other-job"`) {
		t.Fatalf("Execute() error = %q, want observed job_id", err)
	}
	if !strings.Contains(err.Error(), `job_id="job-1"`) {
		t.Fatalf("Execute() error = %q, want expected job_id", err)
	}
}

func TestMissionSetStepCommandWithStatusFileTimesOutWhenStepNeverMatches(t *testing.T) {
	controlPath := filepath.Join(t.TempDir(), "control.json")
	statusPath := filepath.Join(t.TempDir(), "status.json")
	writeMissionStatusSnapshotFile(t, statusPath, missionStatusSnapshot{
		Active:    true,
		StepID:    "build",
		UpdatedAt: "2026-03-20T12:00:00Z",
	})

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{
		"mission", "set-step",
		"--control-file", controlPath,
		"--step-id", "final",
		"--status-file", statusPath,
		"--wait-timeout", "75ms",
	})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "timed out waiting") {
		t.Fatalf("Execute() error = %q, want timeout message", err)
	}
	if !strings.Contains(err.Error(), `want active=true step_id="final"`) {
		t.Fatalf("Execute() error = %q, want requested step confirmation message", err)
	}

	control := readMissionStepControlFile(t, controlPath)
	if control.StepID != "final" {
		t.Fatalf("StepID = %q, want %q", control.StepID, "final")
	}
}

func TestMissionSetStepCommandWithStatusFileTimesOutWhenMatchingSnapshotIsNotFresh(t *testing.T) {
	controlPath := filepath.Join(t.TempDir(), "control.json")
	statusPath := filepath.Join(t.TempDir(), "status.json")
	writeMissionStatusSnapshotFile(t, statusPath, missionStatusSnapshot{
		Active:    true,
		StepID:    "final",
		UpdatedAt: "2026-03-20T12:00:00Z",
	})

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{
		"mission", "set-step",
		"--control-file", controlPath,
		"--step-id", "final",
		"--status-file", statusPath,
		"--wait-timeout", "75ms",
	})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "timed out waiting") {
		t.Fatalf("Execute() error = %q, want timeout message", err)
	}
	if !strings.Contains(err.Error(), "fresh matching update") {
		t.Fatalf("Execute() error = %q, want freshness message", err)
	}

	control := readMissionStepControlFile(t, controlPath)
	if control.StepID != "final" {
		t.Fatalf("StepID = %q, want %q", control.StepID, "final")
	}
}

func TestMissionSetStepCommandWithInvalidStatusJSONReturnsError(t *testing.T) {
	controlPath := filepath.Join(t.TempDir(), "control.json")
	statusPath := filepath.Join(t.TempDir(), "status.json")
	if err := os.WriteFile(statusPath, []byte("{"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{
		"mission", "set-step",
		"--control-file", controlPath,
		"--step-id", "final",
		"--status-file", statusPath,
		"--wait-timeout", "75ms",
	})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "timed out waiting") {
		t.Fatalf("Execute() error = %q, want timeout message", err)
	}
	if !strings.Contains(err.Error(), "failed to decode mission status file") {
		t.Fatalf("Execute() error = %q, want status decode failure", err)
	}
}

func TestMissionSetStepCommandWithNoPriorValidStatusSnapshotSucceedsWhenMatchingSnapshotAppears(t *testing.T) {
	controlPath := filepath.Join(t.TempDir(), "control.json")
	statusPath := filepath.Join(t.TempDir(), "status.json")
	if err := os.WriteFile(statusPath, []byte("{"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	errCh := make(chan error, 1)
	go func() {
		time.Sleep(25 * time.Millisecond)
		data, err := json.Marshal(missionStatusSnapshot{
			Active:    true,
			StepID:    "final",
			UpdatedAt: "2026-03-20T12:00:01Z",
		})
		if err != nil {
			errCh <- err
			return
		}
		errCh <- os.WriteFile(statusPath, data, 0o644)
	}()

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{
		"mission", "set-step",
		"--control-file", controlPath,
		"--step-id", "final",
		"--status-file", statusPath,
		"--wait-timeout", "250ms",
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if err := <-errCh; err != nil {
		t.Fatalf("status update error = %v", err)
	}

	control := readMissionStepControlFile(t, controlPath)
	if control.StepID != "final" {
		t.Fatalf("StepID = %q, want %q", control.StepID, "final")
	}
}

func TestMissionSetStepCommandConfirmationUsesSharedObservationReader(t *testing.T) {
	original := loadMissionStatusObservation
	t.Cleanup(func() { loadMissionStatusObservation = original })

	controlPath := filepath.Join(t.TempDir(), "control.json")
	called := 0
	loadMissionStatusObservation = func(path string) (missioncontrol.MissionStatusSnapshot, error) {
		called++
		if path != "status.json" {
			t.Fatalf("shared observation path = %q, want %q", path, "status.json")
		}
		switch called {
		case 1:
			return missioncontrol.MissionStatusSnapshot{
				Active:    true,
				StepID:    "build",
				UpdatedAt: "2026-03-20T12:00:00Z",
			}, nil
		default:
			return missioncontrol.MissionStatusSnapshot{
				Active:    true,
				StepID:    "final",
				UpdatedAt: "2026-03-20T12:00:01Z",
			}, nil
		}
	}

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{
		"mission", "set-step",
		"--control-file", controlPath,
		"--step-id", "final",
		"--status-file", "status.json",
		"--wait-timeout", "75ms",
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if called < 2 {
		t.Fatalf("shared observation calls = %d, want at least 2", called)
	}

	control := readMissionStepControlFile(t, controlPath)
	if control.StepID != "final" {
		t.Fatalf("StepID = %q, want %q", control.StepID, "final")
	}
}

func TestMissionSetStepCommandWithMissingStatusFileReturnsErrorAfterWaiting(t *testing.T) {
	controlPath := filepath.Join(t.TempDir(), "control.json")
	statusPath := filepath.Join(t.TempDir(), "missing-status.json")

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{
		"mission", "set-step",
		"--control-file", controlPath,
		"--step-id", "final",
		"--status-file", statusPath,
		"--wait-timeout", "75ms",
	})

	start := time.Now()
	err := cmd.Execute()
	elapsed := time.Since(start)
	if err == nil {
		t.Fatal("Execute() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "timed out waiting") {
		t.Fatalf("Execute() error = %q, want timeout message", err)
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Fatalf("Execute() error = %q, want missing status file message", err)
	}
	if elapsed < 50*time.Millisecond {
		t.Fatalf("Execute() elapsed = %v, want wait before timeout", elapsed)
	}

	control := readMissionStepControlFile(t, controlPath)
	if control.StepID != "final" {
		t.Fatalf("StepID = %q, want %q", control.StepID, "final")
	}
}

func TestMissionSetStepCommandWithMissionFileWritesControlFile(t *testing.T) {
	controlPath := filepath.Join(t.TempDir(), "control.json")
	missionPath := writeMissionBootstrapJobFile(t, testMissionBootstrapJob())

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{
		"mission", "set-step",
		"--control-file", controlPath,
		"--step-id", "final",
		"--mission-file", missionPath,
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	control := readMissionStepControlFile(t, controlPath)
	if control.StepID != "final" {
		t.Fatalf("StepID = %q, want %q", control.StepID, "final")
	}
	if control.UpdatedAt == "" {
		t.Fatal("UpdatedAt = empty string, want RFC3339Nano timestamp")
	}
}

func TestMissionSetStepCommandWithMissingMissionFileReturnsError(t *testing.T) {
	controlPath := filepath.Join(t.TempDir(), "control.json")
	missionPath := filepath.Join(t.TempDir(), "missing-mission.json")

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{
		"mission", "set-step",
		"--control-file", controlPath,
		"--step-id", "final",
		"--mission-file", missionPath,
	})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "failed to read mission file") {
		t.Fatalf("Execute() error = %q, want mission read failure", err)
	}

	assertMissionStepControlFileMissing(t, controlPath)
}

func TestMissionSetStepCommandWithInvalidMissionJSONReturnsError(t *testing.T) {
	controlPath := filepath.Join(t.TempDir(), "control.json")
	missionPath := filepath.Join(t.TempDir(), "mission.json")
	if err := os.WriteFile(missionPath, []byte("{"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{
		"mission", "set-step",
		"--control-file", controlPath,
		"--step-id", "final",
		"--mission-file", missionPath,
	})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "failed to decode mission file") {
		t.Fatalf("Execute() error = %q, want mission decode failure", err)
	}

	assertMissionStepControlFileMissing(t, controlPath)
}

func TestMissionSetStepCommandWithInvalidMissionReturnsError(t *testing.T) {
	controlPath := filepath.Join(t.TempDir(), "control.json")
	missionPath := writeMissionBootstrapJobFile(t, missioncontrol.Job{
		ID:           "job-1",
		MaxAuthority: missioncontrol.AuthorityTierHigh,
		AllowedTools: []string{"read"},
		Plan: missioncontrol.Plan{
			ID: "plan-1",
			Steps: []missioncontrol.Step{
				{
					ID:                "draft",
					Type:              missioncontrol.StepTypeDiscussion,
					RequiredAuthority: missioncontrol.AuthorityTierLow,
					AllowedTools:      []string{"read"},
				},
			},
		},
	})

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{
		"mission", "set-step",
		"--control-file", controlPath,
		"--step-id", "draft",
		"--mission-file", missionPath,
	})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "failed to validate mission file") {
		t.Fatalf("Execute() error = %q, want mission validation failure", err)
	}
	if !strings.Contains(err.Error(), "E_PLAN_INVALID") {
		t.Fatalf("Execute() error = %q, want validation error code", err)
	}

	assertMissionStepControlFileMissing(t, controlPath)
}

func TestMissionSetStepCommandWithUnknownMissionStepReturnsError(t *testing.T) {
	controlPath := filepath.Join(t.TempDir(), "control.json")
	missionPath := writeMissionBootstrapJobFile(t, testMissionBootstrapJob())

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{
		"mission", "set-step",
		"--control-file", controlPath,
		"--step-id", "missing",
		"--mission-file", missionPath,
	})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "failed to validate mission file") {
		t.Fatalf("Execute() error = %q, want mission validation failure", err)
	}
	if !strings.Contains(err.Error(), "E_INVALID_ACTION_FOR_STEP") {
		t.Fatalf("Execute() error = %q, want unknown step error code", err)
	}

	assertMissionStepControlFileMissing(t, controlPath)
}

func TestMissionSetStepCommandWithoutRequiredFlagsReturnsError(t *testing.T) {
	t.Run("missing control file", func(t *testing.T) {
		cmd := NewRootCmd()
		cmd.SetOut(&bytes.Buffer{})
		cmd.SetErr(&bytes.Buffer{})
		cmd.SetArgs([]string{"mission", "set-step"})

		err := cmd.Execute()
		if err == nil {
			t.Fatal("Execute() error = nil, want non-nil")
		}
		if !strings.Contains(err.Error(), "--control-file is required") {
			t.Fatalf("Execute() error = %q, want missing control-file message", err)
		}
	})

	t.Run("missing step id", func(t *testing.T) {
		cmd := NewRootCmd()
		cmd.SetOut(&bytes.Buffer{})
		cmd.SetErr(&bytes.Buffer{})
		cmd.SetArgs([]string{"mission", "set-step", "--control-file", filepath.Join(t.TempDir(), "control.json")})

		err := cmd.Execute()
		if err == nil {
			t.Fatal("Execute() error = nil, want non-nil")
		}
		if !strings.Contains(err.Error(), "--step-id is required") {
			t.Fatalf("Execute() error = %q, want missing step-id message", err)
		}
	})
}

func TestConfigureMissionBootstrapDefaultUnchanged(t *testing.T) {
	ag := newMissionBootstrapTestLoop()
	cmd := newMissionBootstrapTestCommand()

	if err := configureMissionBootstrap(cmd, ag); err != nil {
		t.Fatalf("configureMissionBootstrap() error = %v", err)
	}

	if ag.MissionRequired() {
		t.Fatal("MissionRequired() = true, want false")
	}

	if _, ok := ag.ActiveMissionStep(); ok {
		t.Fatal("ActiveMissionStep() ok = true, want false")
	}
}

func TestConfigureMissionBootstrapMissionRequiredEnablesMode(t *testing.T) {
	ag := newMissionBootstrapTestLoop()
	cmd := newMissionBootstrapTestCommand()
	if err := cmd.Flags().Set("mission-required", "true"); err != nil {
		t.Fatalf("Flags().Set() error = %v", err)
	}

	if err := configureMissionBootstrap(cmd, ag); err != nil {
		t.Fatalf("configureMissionBootstrap() error = %v", err)
	}

	if !ag.MissionRequired() {
		t.Fatal("MissionRequired() = false, want true")
	}
}

func TestConfigureMissionBootstrapMissionFileActivatesStep(t *testing.T) {
	ag := newMissionBootstrapTestLoop()
	cmd := newMissionBootstrapTestCommand()
	missionFile := writeMissionBootstrapJobFile(t, testMissionBootstrapJob())
	if err := cmd.Flags().Set("mission-file", missionFile); err != nil {
		t.Fatalf("Flags().Set(mission-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step", "build"); err != nil {
		t.Fatalf("Flags().Set(mission-step) error = %v", err)
	}

	if err := configureMissionBootstrap(cmd, ag); err != nil {
		t.Fatalf("configureMissionBootstrap() error = %v", err)
	}

	ec, ok := ag.ActiveMissionStep()
	if !ok {
		t.Fatal("ActiveMissionStep() ok = false, want true")
	}

	if ec.Job == nil || ec.Step == nil {
		t.Fatalf("ActiveMissionStep() = %#v, want non-nil job and step", ec)
	}

	if ec.Job.ID != "job-1" {
		t.Fatalf("ActiveMissionStep().Job.ID = %q, want %q", ec.Job.ID, "job-1")
	}

	if ec.Step.ID != "build" {
		t.Fatalf("ActiveMissionStep().Step.ID = %q, want %q", ec.Step.ID, "build")
	}
}

func TestConfigureMissionBootstrapInvalidMissionFileFailsStartup(t *testing.T) {
	ag := newMissionBootstrapTestLoop()
	cmd := newMissionBootstrapTestCommand()
	missionFile := filepath.Join(t.TempDir(), "mission.json")
	if err := os.WriteFile(missionFile, []byte("{not-json"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if err := cmd.Flags().Set("mission-file", missionFile); err != nil {
		t.Fatalf("Flags().Set(mission-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step", "build"); err != nil {
		t.Fatalf("Flags().Set(mission-step) error = %v", err)
	}

	err := configureMissionBootstrap(cmd, ag)
	if err == nil {
		t.Fatal("configureMissionBootstrap() error = nil, want decode error")
	}

	if !strings.Contains(err.Error(), "failed to decode mission file") {
		t.Fatalf("configureMissionBootstrap() error = %q, want decode failure", err)
	}
}

func TestConfigureMissionBootstrapMissionFileRequiresMissionStep(t *testing.T) {
	ag := newMissionBootstrapTestLoop()
	cmd := newMissionBootstrapTestCommand()
	missionFile := writeMissionBootstrapJobFile(t, testMissionBootstrapJob())
	if err := cmd.Flags().Set("mission-file", missionFile); err != nil {
		t.Fatalf("Flags().Set(mission-file) error = %v", err)
	}

	err := configureMissionBootstrap(cmd, ag)
	if err == nil {
		t.Fatal("configureMissionBootstrap() error = nil, want missing mission-step error")
	}

	if !strings.Contains(err.Error(), "--mission-file requires --mission-step") {
		t.Fatalf("configureMissionBootstrap() error = %q, want missing mission-step message", err)
	}
}

func TestConfigureMissionBootstrapMissionStepRequiresMissionFile(t *testing.T) {
	ag := newMissionBootstrapTestLoop()
	cmd := newMissionBootstrapTestCommand()
	if err := cmd.Flags().Set("mission-step", "build"); err != nil {
		t.Fatalf("Flags().Set(mission-step) error = %v", err)
	}

	err := configureMissionBootstrap(cmd, ag)
	if err == nil {
		t.Fatal("configureMissionBootstrap() error = nil, want missing mission-file error")
	}

	if !strings.Contains(err.Error(), "--mission-step requires --mission-file") {
		t.Fatalf("configureMissionBootstrap() error = %q, want missing mission-file message", err)
	}
}

func TestConfigureMissionBootstrapMissionStepControlFileRequiresMissionFile(t *testing.T) {
	ag := newMissionBootstrapTestLoop()
	cmd := newMissionBootstrapTestCommand()
	if err := cmd.Flags().Set("mission-step-control-file", filepath.Join(t.TempDir(), "control.json")); err != nil {
		t.Fatalf("Flags().Set(mission-step-control-file) error = %v", err)
	}

	err := configureMissionBootstrap(cmd, ag)
	if err == nil {
		t.Fatal("configureMissionBootstrap() error = nil, want missing mission-file error")
	}

	if !strings.Contains(err.Error(), "--mission-step-control-file requires --mission-file") {
		t.Fatalf("configureMissionBootstrap() error = %q, want missing mission-file message", err)
	}
}

func TestWriteMissionStatusSnapshotFromCommandDefaultPathUnchanged(t *testing.T) {
	ag := newMissionBootstrapTestLoop()
	cmd := newMissionBootstrapTestCommand()
	missionFile := writeMissionBootstrapJobFile(t, testMissionBootstrapJob())

	if err := cmd.Flags().Set("mission-file", missionFile); err != nil {
		t.Fatalf("Flags().Set(mission-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step", "build"); err != nil {
		t.Fatalf("Flags().Set(mission-step) error = %v", err)
	}
	if err := configureMissionBootstrap(cmd, ag); err != nil {
		t.Fatalf("configureMissionBootstrap() error = %v", err)
	}

	if err := writeMissionStatusSnapshotFromCommand(cmd, ag, time.Date(2026, 3, 19, 12, 0, 0, 0, time.UTC)); err != nil {
		t.Fatalf("writeMissionStatusSnapshotFromCommand() error = %v", err)
	}

	entries, err := os.ReadDir(filepath.Dir(missionFile))
	if err != nil {
		t.Fatalf("ReadDir() error = %v", err)
	}
	if len(entries) != 1 || entries[0].Name() != filepath.Base(missionFile) {
		t.Fatalf("ReadDir() = %v, want only %q", entries, filepath.Base(missionFile))
	}
}

func TestWriteMissionStatusSnapshotNoActiveMissionWritesInactiveSnapshot(t *testing.T) {
	ag := newMissionBootstrapTestLoop()
	path := filepath.Join(t.TempDir(), "status.json")
	now := time.Date(2026, 3, 19, 12, 0, 0, 123, time.UTC)

	if err := writeMissionStatusSnapshot(path, "", ag, now); err != nil {
		t.Fatalf("writeMissionStatusSnapshot() error = %v", err)
	}

	got := readMissionStatusSnapshotFile(t, path)
	if got.MissionRequired {
		t.Fatal("MissionRequired = true, want false")
	}
	if got.Active {
		t.Fatal("Active = true, want false")
	}
	if got.MissionFile != "" {
		t.Fatalf("MissionFile = %q, want empty", got.MissionFile)
	}
	if got.JobID != "" || got.StepID != "" || got.StepType != "" {
		t.Fatalf("snapshot IDs = (%q, %q, %q), want empty strings", got.JobID, got.StepID, got.StepType)
	}
	if got.RequiredAuthority != "" {
		t.Fatalf("RequiredAuthority = %q, want empty", got.RequiredAuthority)
	}
	if got.RequiresApproval {
		t.Fatal("RequiresApproval = true, want false")
	}
	if len(got.AllowedTools) != 0 {
		t.Fatalf("AllowedTools = %v, want empty", got.AllowedTools)
	}
	if got.UpdatedAt != now.Format(time.RFC3339Nano) {
		t.Fatalf("UpdatedAt = %q, want %q", got.UpdatedAt, now.Format(time.RFC3339Nano))
	}
}

func TestWriteMissionStatusSnapshotActiveMissionWritesExpectedFields(t *testing.T) {
	ag := newMissionBootstrapTestLoop()
	cmd := newMissionBootstrapTestCommand()
	job := missioncontrol.Job{
		ID:           "job-1",
		MaxAuthority: missioncontrol.AuthorityTierHigh,
		AllowedTools: []string{"read"},
		Plan: missioncontrol.Plan{
			ID: "plan-1",
			Steps: []missioncontrol.Step{
				{
					ID:                "build",
					Type:              missioncontrol.StepTypeOneShotCode,
					RequiredAuthority: missioncontrol.AuthorityTierMedium,
					RequiresApproval:  true,
					AllowedTools:      []string{"read"},
				},
				{
					ID:        "final",
					Type:      missioncontrol.StepTypeFinalResponse,
					DependsOn: []string{"build"},
				},
			},
		},
	}
	missionFile := writeMissionBootstrapJobFile(t, job)
	path := filepath.Join(t.TempDir(), "status.json")
	now := time.Date(2026, 3, 19, 12, 0, 0, 456, time.UTC)

	if err := cmd.Flags().Set("mission-required", "true"); err != nil {
		t.Fatalf("Flags().Set(mission-required) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-file", missionFile); err != nil {
		t.Fatalf("Flags().Set(mission-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step", "build"); err != nil {
		t.Fatalf("Flags().Set(mission-step) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-status-file", path); err != nil {
		t.Fatalf("Flags().Set(mission-status-file) error = %v", err)
	}

	if err := configureMissionBootstrap(cmd, ag); err != nil {
		t.Fatalf("configureMissionBootstrap() error = %v", err)
	}
	if err := writeMissionStatusSnapshotFromCommand(cmd, ag, now); err != nil {
		t.Fatalf("writeMissionStatusSnapshotFromCommand() error = %v", err)
	}

	got := readMissionStatusSnapshotFile(t, path)
	if !got.MissionRequired {
		t.Fatal("MissionRequired = false, want true")
	}
	if !got.Active {
		t.Fatal("Active = false, want true")
	}
	if got.MissionFile != missionFile {
		t.Fatalf("MissionFile = %q, want %q", got.MissionFile, missionFile)
	}
	if got.JobID != "job-1" {
		t.Fatalf("JobID = %q, want %q", got.JobID, "job-1")
	}
	if got.StepID != "build" {
		t.Fatalf("StepID = %q, want %q", got.StepID, "build")
	}
	if got.StepType != string(missioncontrol.StepTypeOneShotCode) {
		t.Fatalf("StepType = %q, want %q", got.StepType, missioncontrol.StepTypeOneShotCode)
	}
	if got.RequiredAuthority != missioncontrol.AuthorityTierMedium {
		t.Fatalf("RequiredAuthority = %q, want %q", got.RequiredAuthority, missioncontrol.AuthorityTierMedium)
	}
	if !got.RequiresApproval {
		t.Fatal("RequiresApproval = false, want true")
	}
	if got.Runtime == nil {
		t.Fatal("Runtime = nil, want non-nil")
	}
	if got.Runtime.State != missioncontrol.JobStateRunning {
		t.Fatalf("Runtime.State = %q, want %q", got.Runtime.State, missioncontrol.JobStateRunning)
	}
	if got.Runtime.ActiveStepID != "build" {
		t.Fatalf("Runtime.ActiveStepID = %q, want %q", got.Runtime.ActiveStepID, "build")
	}
	if got.RuntimeSummary == nil {
		t.Fatal("RuntimeSummary = nil, want non-nil")
	}
	if got.RuntimeSummary.State != missioncontrol.JobStateRunning {
		t.Fatalf("RuntimeSummary.State = %q, want %q", got.RuntimeSummary.State, missioncontrol.JobStateRunning)
	}
	if got.RuntimeSummary.ActiveStepID != "build" {
		t.Fatalf("RuntimeSummary.ActiveStepID = %q, want %q", got.RuntimeSummary.ActiveStepID, "build")
	}
	if !reflect.DeepEqual(got.RuntimeSummary.AllowedTools, []string{"read"}) {
		t.Fatalf("RuntimeSummary.AllowedTools = %#v, want %#v", got.RuntimeSummary.AllowedTools, []string{"read"})
	}
	if got.Runtime.InspectablePlan == nil {
		t.Fatal("Runtime.InspectablePlan = nil, want non-nil")
	}
	if len(got.Runtime.InspectablePlan.Steps) != len(testMissionBootstrapJob().Plan.Steps) {
		t.Fatalf("len(Runtime.InspectablePlan.Steps) = %d, want %d", len(got.Runtime.InspectablePlan.Steps), len(testMissionBootstrapJob().Plan.Steps))
	}
	if got.RuntimeControl == nil {
		t.Fatal("RuntimeControl = nil, want non-nil")
	}
	if got.RuntimeControl.JobID != "job-1" {
		t.Fatalf("RuntimeControl.JobID = %q, want %q", got.RuntimeControl.JobID, "job-1")
	}
	if got.RuntimeControl.Step.ID != "build" {
		t.Fatalf("RuntimeControl.Step.ID = %q, want %q", got.RuntimeControl.Step.ID, "build")
	}
	if got.RuntimeControl.Step.Type != missioncontrol.StepTypeOneShotCode {
		t.Fatalf("RuntimeControl.Step.Type = %q, want %q", got.RuntimeControl.Step.Type, missioncontrol.StepTypeOneShotCode)
	}
}

func TestWriteMissionStatusSnapshotFromCommandUsesCommittedDurableProjectionWhenPresent(t *testing.T) {
	ag := newMissionBootstrapTestLoop()
	cmd := newMissionBootstrapTestCommand()
	job := testMissionBootstrapJob()
	missionFile := writeMissionBootstrapJobFile(t, job)
	statusFile := filepath.Join(t.TempDir(), "status.json")
	storeRoot := missioncontrol.ResolveStoreRoot("", statusFile)
	now := time.Date(2026, 4, 5, 19, 0, 0, 0, time.UTC)

	if err := cmd.Flags().Set("mission-file", missionFile); err != nil {
		t.Fatalf("Flags().Set(mission-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step", "build"); err != nil {
		t.Fatalf("Flags().Set(mission-step) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-status-file", statusFile); err != nil {
		t.Fatalf("Flags().Set(mission-status-file) error = %v", err)
	}

	writeCommittedMissionBootstrapJobRuntimeRecord(t, storeRoot, job.ID, 4, 1, now, missioncontrol.JobStatePaused, "final")
	writeCommittedMissionBootstrapRuntimeControlRecord(t, storeRoot, 4, 1, runtimeControlForBootstrapStep(t, job, "final"))

	liveControl := runtimeControlForBootstrapStep(t, job, "build")
	liveRuntime := missioncontrol.JobRuntimeState{
		JobID:        job.ID,
		State:        missioncontrol.JobStateRunning,
		ActiveStepID: "build",
		ActiveStepAt: now.Add(time.Minute),
	}
	if err := ag.HydrateMissionRuntimeControl(job, liveRuntime, liveControl); err != nil {
		t.Fatalf("HydrateMissionRuntimeControl() error = %v", err)
	}

	if err := writeMissionStatusSnapshotFromCommand(cmd, ag, now.Add(2*time.Minute)); err != nil {
		t.Fatalf("writeMissionStatusSnapshotFromCommand() error = %v", err)
	}

	got := readMissionStatusSnapshotFile(t, statusFile)
	if got.Active {
		t.Fatal("Active = true, want false from committed paused durable projection")
	}
	if got.JobID != job.ID {
		t.Fatalf("JobID = %q, want %q", got.JobID, job.ID)
	}
	if got.StepID != "final" {
		t.Fatalf("StepID = %q, want %q", got.StepID, "final")
	}
	if got.StepType != string(missioncontrol.StepTypeFinalResponse) {
		t.Fatalf("StepType = %q, want %q", got.StepType, missioncontrol.StepTypeFinalResponse)
	}
	assertMissionStatusSnapshotMatchesCommittedDurableState(t, got, storeRoot, job.ID, now.Add(2*time.Minute))
}

func TestStartupAndRuntimeChangeDurableProjectionUseSameSharedBuilder(t *testing.T) {
	job := missioncontrol.Job{
		ID:           "job-1",
		SpecVersion:  missioncontrol.JobSpecVersionV2,
		MaxAuthority: missioncontrol.AuthorityTierHigh,
		AllowedTools: []string{"read"},
		Plan: missioncontrol.Plan{
			ID: "plan-1",
			Steps: []missioncontrol.Step{
				{ID: "gamma", Type: missioncontrol.StepTypeOneShotCode, OneShotArtifactPath: "zeta.txt"},
				{ID: "alpha", Type: missioncontrol.StepTypeStaticArtifact, StaticArtifactPath: "alpha.json", StaticArtifactFormat: "json"},
				{ID: "beta", Type: missioncontrol.StepTypeLongRunningCode, LongRunningArtifactPath: "service.bin", LongRunningStartupCommand: []string{"go", "build", "./cmd/service"}},
				{ID: "delta", Type: missioncontrol.StepTypeStaticArtifact, StaticArtifactPath: "delta.md", StaticArtifactFormat: "markdown"},
				{ID: "epsilon", Type: missioncontrol.StepTypeOneShotCode, OneShotArtifactPath: "epsilon.go"},
				{ID: "zeta", Type: missioncontrol.StepTypeStaticArtifact, StaticArtifactPath: "zeta.yaml", StaticArtifactFormat: "yaml"},
				{ID: "final", Type: missioncontrol.StepTypeFinalResponse, DependsOn: []string{"zeta"}},
			},
		},
	}
	control, err := missioncontrol.BuildRuntimeControlContext(job, "final")
	if err != nil {
		t.Fatalf("BuildRuntimeControlContext() error = %v", err)
	}
	plan, err := missioncontrol.BuildInspectablePlanContext(job)
	if err != nil {
		t.Fatalf("BuildInspectablePlanContext() error = %v", err)
	}
	requests := make([]missioncontrol.ApprovalRequest, 0, missioncontrol.OperatorStatusApprovalHistoryLimit+2)
	for i := 0; i < missioncontrol.OperatorStatusApprovalHistoryLimit+2; i++ {
		requests = append(requests, missioncontrol.ApprovalRequest{
			JobID:           job.ID,
			StepID:          "step-" + string(rune('a'+i)),
			RequestedAction: missioncontrol.ApprovalRequestedActionStepComplete,
			Scope:           missioncontrol.ApprovalScopeMissionStep,
			State:           missioncontrol.ApprovalStatePending,
			RequestedAt:     time.Date(2026, 3, 24, 12, i, 0, 0, time.UTC),
		})
	}
	history := make([]missioncontrol.AuditEvent, 0, missioncontrol.OperatorStatusRecentAuditLimit+1)
	for i := 0; i < missioncontrol.OperatorStatusRecentAuditLimit+1; i++ {
		history = append(history, missioncontrol.AuditEvent{
			JobID:     job.ID,
			StepID:    "build",
			ToolName:  "status",
			Allowed:   true,
			Timestamp: time.Date(2026, 3, 24, 13, i, 0, 0, time.UTC),
		})
	}
	now := time.Now().UTC().Truncate(time.Second)
	requestBase := now.Add(-8 * time.Hour)
	auditBase := now.Add(-7 * time.Hour)
	pausedAt := now.Add(-6 * time.Hour)
	for i := range requests {
		requests[i].RequestedAt = requestBase.Add(time.Duration(i) * time.Minute)
	}
	for i := range history {
		history[i].Timestamp = auditBase.Add(time.Duration(i) * time.Minute)
	}
	runtime := missioncontrol.JobRuntimeState{
		JobID:            job.ID,
		State:            missioncontrol.JobStatePaused,
		ActiveStepID:     "final",
		InspectablePlan:  &plan,
		PausedReason:     missioncontrol.RuntimePauseReasonOperatorCommand,
		PausedAt:         pausedAt,
		ApprovalRequests: requests,
		AuditHistory:     history,
		CompletedSteps: []missioncontrol.RuntimeStepRecord{
			{StepID: "zeta"},
			{StepID: "gamma"},
			{StepID: "beta", ResultingState: &missioncontrol.RuntimeResultingStateRecord{Kind: string(missioncontrol.StepTypeLongRunningCode), Target: "service.bin", State: "already_present"}},
			{StepID: "alpha"},
			{StepID: "epsilon"},
			{StepID: "delta"},
		},
	}

	statusFile := filepath.Join(t.TempDir(), "status.json")
	runtimePath := filepath.Join(t.TempDir(), "runtime.json")
	storeRoot := missioncontrol.ResolveStoreRoot("", statusFile)
	if err := missioncontrol.PersistProjectedRuntimeState(storeRoot, missioncontrol.WriterLockLease{LeaseHolderID: "holder-1"}, &job, runtime, &control, now); err != nil {
		t.Fatalf("PersistProjectedRuntimeState() error = %v", err)
	}

	cmd := newMissionBootstrapTestCommand()
	missionFile := writeMissionBootstrapJobFile(t, job)
	if err := cmd.Flags().Set("mission-file", missionFile); err != nil {
		t.Fatalf("Flags().Set(mission-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step", "build"); err != nil {
		t.Fatalf("Flags().Set(mission-step) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-status-file", statusFile); err != nil {
		t.Fatalf("Flags().Set(mission-status-file) error = %v", err)
	}

	ag := newMissionBootstrapTestLoop()
	liveControl := runtimeControlForBootstrapStep(t, testMissionBootstrapJob(), "build")
	liveRuntime := missioncontrol.JobRuntimeState{
		JobID:        job.ID,
		State:        missioncontrol.JobStateRunning,
		ActiveStepID: "build",
		ActiveStepAt: now.Add(time.Minute),
	}
	if err := ag.HydrateMissionRuntimeControl(testMissionBootstrapJob(), liveRuntime, liveControl); err != nil {
		t.Fatalf("HydrateMissionRuntimeControl() error = %v", err)
	}

	if err := writeMissionStatusSnapshotFromCommand(cmd, ag, now); err != nil {
		t.Fatalf("writeMissionStatusSnapshotFromCommand() error = %v", err)
	}
	if err := writeProjectedMissionStatusSnapshot(runtimePath, missionFile, storeRoot, false, job.ID, now); err != nil {
		t.Fatalf("writeProjectedMissionStatusSnapshot() error = %v", err)
	}

	if !bytes.Equal(mustReadFile(t, statusFile), mustReadFile(t, runtimePath)) {
		t.Fatalf("durable startup/runtime projection bytes differ:\nstartup=%s\nruntime=%s", string(mustReadFile(t, statusFile)), string(mustReadFile(t, runtimePath)))
	}
}

func TestWriteMissionStatusSnapshotFromCommandFallsBackToLiveWhenDurableStoreEmptyForJob(t *testing.T) {
	ag := newMissionBootstrapTestLoop()
	cmd := newMissionBootstrapTestCommand()
	job := testMissionBootstrapJob()
	missionFile := writeMissionBootstrapJobFile(t, job)
	statusFile := filepath.Join(t.TempDir(), "status.json")
	now := time.Date(2026, 4, 5, 19, 45, 0, 0, time.UTC)

	if err := cmd.Flags().Set("mission-file", missionFile); err != nil {
		t.Fatalf("Flags().Set(mission-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step", "build"); err != nil {
		t.Fatalf("Flags().Set(mission-step) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-status-file", statusFile); err != nil {
		t.Fatalf("Flags().Set(mission-status-file) error = %v", err)
	}

	control := runtimeControlForBootstrapStep(t, job, "build")
	runtime := missioncontrol.JobRuntimeState{
		JobID:        job.ID,
		State:        missioncontrol.JobStatePaused,
		ActiveStepID: "build",
		PausedReason: missioncontrol.RuntimePauseReasonOperatorCommand,
		PausedAt:     now.Add(-time.Minute),
	}
	if err := ag.HydrateMissionRuntimeControl(job, runtime, control); err != nil {
		t.Fatalf("HydrateMissionRuntimeControl() error = %v", err)
	}

	if err := writeMissionStatusSnapshotFromCommand(cmd, ag, now); err != nil {
		t.Fatalf("writeMissionStatusSnapshotFromCommand() error = %v", err)
	}

	got := readMissionStatusSnapshotFile(t, statusFile)
	if got.StepID != "build" {
		t.Fatalf("StepID = %q, want %q", got.StepID, "build")
	}
	if got.StepType != "" {
		t.Fatalf("StepType = %q, want empty live fallback step metadata", got.StepType)
	}
	if got.Runtime == nil || got.Runtime.State != missioncontrol.JobStatePaused {
		t.Fatalf("Runtime = %#v, want live paused runtime", got.Runtime)
	}
	if got.RuntimeControl == nil || got.RuntimeControl.Step.ID != "build" {
		t.Fatalf("RuntimeControl = %#v, want live build control", got.RuntimeControl)
	}
}

func TestWriteMissionStatusSnapshotIncludesRuntimeSummaryTruncationForPersistedRuntime(t *testing.T) {
	ag := newMissionBootstrapTestLoop()
	job := missioncontrol.Job{
		ID:           "job-1",
		SpecVersion:  missioncontrol.JobSpecVersionV2,
		MaxAuthority: missioncontrol.AuthorityTierHigh,
		AllowedTools: []string{"read"},
		Plan: missioncontrol.Plan{
			ID: "plan-1",
			Steps: []missioncontrol.Step{
				{ID: "gamma", Type: missioncontrol.StepTypeOneShotCode, OneShotArtifactPath: "zeta.txt"},
				{ID: "alpha", Type: missioncontrol.StepTypeStaticArtifact, StaticArtifactPath: "alpha.json", StaticArtifactFormat: "json"},
				{ID: "beta", Type: missioncontrol.StepTypeLongRunningCode, LongRunningArtifactPath: "service.bin", LongRunningStartupCommand: []string{"go", "build", "./cmd/service"}},
				{ID: "delta", Type: missioncontrol.StepTypeStaticArtifact, StaticArtifactPath: "delta.md", StaticArtifactFormat: "markdown"},
				{ID: "epsilon", Type: missioncontrol.StepTypeOneShotCode, OneShotArtifactPath: "epsilon.go"},
				{ID: "zeta", Type: missioncontrol.StepTypeStaticArtifact, StaticArtifactPath: "zeta.yaml", StaticArtifactFormat: "yaml"},
				{ID: "final", Type: missioncontrol.StepTypeFinalResponse, DependsOn: []string{"zeta"}},
			},
		},
	}
	control, err := missioncontrol.BuildRuntimeControlContext(job, "final")
	if err != nil {
		t.Fatalf("BuildRuntimeControlContext() error = %v", err)
	}
	plan, err := missioncontrol.BuildInspectablePlanContext(job)
	if err != nil {
		t.Fatalf("BuildInspectablePlanContext() error = %v", err)
	}
	requests := make([]missioncontrol.ApprovalRequest, 0, missioncontrol.OperatorStatusApprovalHistoryLimit+2)
	for i := 0; i < missioncontrol.OperatorStatusApprovalHistoryLimit+2; i++ {
		requests = append(requests, missioncontrol.ApprovalRequest{
			JobID:           job.ID,
			StepID:          "step-" + string(rune('a'+i)),
			RequestedAction: missioncontrol.ApprovalRequestedActionStepComplete,
			Scope:           missioncontrol.ApprovalScopeMissionStep,
			State:           missioncontrol.ApprovalStatePending,
			RequestedAt:     time.Date(2026, 3, 24, 12, i, 0, 0, time.UTC),
		})
	}
	history := make([]missioncontrol.AuditEvent, 0, missioncontrol.OperatorStatusRecentAuditLimit+1)
	for i := 0; i < missioncontrol.OperatorStatusRecentAuditLimit+1; i++ {
		history = append(history, missioncontrol.AuditEvent{
			JobID:     job.ID,
			StepID:    "build",
			ToolName:  "status",
			Allowed:   true,
			Timestamp: time.Date(2026, 3, 24, 13, i, 0, 0, time.UTC),
		})
	}
	runtime := missioncontrol.JobRuntimeState{
		JobID:            job.ID,
		State:            missioncontrol.JobStatePaused,
		ActiveStepID:     "final",
		InspectablePlan:  &plan,
		PausedReason:     missioncontrol.RuntimePauseReasonOperatorCommand,
		PausedAt:         time.Date(2026, 3, 24, 13, 30, 0, 0, time.UTC),
		ApprovalRequests: requests,
		AuditHistory:     history,
		CompletedSteps: []missioncontrol.RuntimeStepRecord{
			{StepID: "zeta"},
			{StepID: "gamma"},
			{StepID: "beta", ResultingState: &missioncontrol.RuntimeResultingStateRecord{Kind: string(missioncontrol.StepTypeLongRunningCode), Target: "service.bin", State: "already_present"}},
			{StepID: "alpha"},
			{StepID: "epsilon"},
			{StepID: "delta"},
		},
	}
	if err := ag.HydrateMissionRuntimeControl(job, runtime, &control); err != nil {
		t.Fatalf("HydrateMissionRuntimeControl() error = %v", err)
	}

	path := filepath.Join(t.TempDir(), "status.json")
	if err := writeMissionStatusSnapshot(path, "mission.json", ag, time.Date(2026, 3, 27, 12, 0, 0, 0, time.UTC)); err != nil {
		t.Fatalf("writeMissionStatusSnapshot() error = %v", err)
	}

	got := readMissionStatusSnapshotFile(t, path)
	if got.RuntimeSummary == nil {
		t.Fatal("RuntimeSummary = nil, want persisted runtime summary")
	}
	if got.Active {
		t.Fatal("Active = true, want false for persisted-only runtime snapshot")
	}
	if got.RuntimeSummary.State != missioncontrol.JobStatePaused {
		t.Fatalf("RuntimeSummary.State = %q, want %q", got.RuntimeSummary.State, missioncontrol.JobStatePaused)
	}
	if got.RuntimeSummary.PausedReason != missioncontrol.RuntimePauseReasonOperatorCommand {
		t.Fatalf("RuntimeSummary.PausedReason = %q, want %q", got.RuntimeSummary.PausedReason, missioncontrol.RuntimePauseReasonOperatorCommand)
	}
	if got.RuntimeSummary.PausedAt == nil || *got.RuntimeSummary.PausedAt != "2026-03-24T13:30:00Z" {
		t.Fatalf("RuntimeSummary.PausedAt = %#v, want RFC3339 pause time", got.RuntimeSummary.PausedAt)
	}
	if !reflect.DeepEqual(got.RuntimeSummary.AllowedTools, []string{"read"}) {
		t.Fatalf("RuntimeSummary.AllowedTools = %#v, want %#v", got.RuntimeSummary.AllowedTools, []string{"read"})
	}
	if len(got.RuntimeSummary.Artifacts) != missioncontrol.OperatorStatusArtifactLimit {
		t.Fatalf("RuntimeSummary.Artifacts = %#v, want %d deterministic entries", got.RuntimeSummary.Artifacts, missioncontrol.OperatorStatusArtifactLimit)
	}
	if got.RuntimeSummary.Artifacts[0].StepID != "gamma" || got.RuntimeSummary.Artifacts[0].Path != "zeta.txt" {
		t.Fatalf("RuntimeSummary.Artifacts[0] = %#v, want step_id=%q path=%q", got.RuntimeSummary.Artifacts[0], "gamma", "zeta.txt")
	}
	if got.RuntimeSummary.Artifacts[2].StepID != "beta" || got.RuntimeSummary.Artifacts[2].State != "already_present" {
		t.Fatalf("RuntimeSummary.Artifacts[2] = %#v, want step_id=%q state=%q", got.RuntimeSummary.Artifacts[2], "beta", "already_present")
	}
	if got.RuntimeSummary.Truncation == nil {
		t.Fatal("RuntimeSummary.Truncation = nil, want truncation metadata")
	}
	if got.RuntimeSummary.Truncation.ApprovalHistoryOmitted != 2 {
		t.Fatalf("RuntimeSummary.Truncation.ApprovalHistoryOmitted = %d, want 2", got.RuntimeSummary.Truncation.ApprovalHistoryOmitted)
	}
	if got.RuntimeSummary.Truncation.RecentAuditOmitted != 1 {
		t.Fatalf("RuntimeSummary.Truncation.RecentAuditOmitted = %d, want 1", got.RuntimeSummary.Truncation.RecentAuditOmitted)
	}
	if got.RuntimeSummary.Truncation.ArtifactsOmitted != 1 {
		t.Fatalf("RuntimeSummary.Truncation.ArtifactsOmitted = %d, want 1", got.RuntimeSummary.Truncation.ArtifactsOmitted)
	}
}

func TestWriteProjectedMissionStatusSnapshotIncludesCommittedRuntimeSummaryTruncation(t *testing.T) {
	job := missioncontrol.Job{
		ID:           "job-1",
		SpecVersion:  missioncontrol.JobSpecVersionV2,
		MaxAuthority: missioncontrol.AuthorityTierHigh,
		AllowedTools: []string{"read"},
		Plan: missioncontrol.Plan{
			ID: "plan-1",
			Steps: []missioncontrol.Step{
				{ID: "gamma", Type: missioncontrol.StepTypeOneShotCode, OneShotArtifactPath: "zeta.txt"},
				{ID: "alpha", Type: missioncontrol.StepTypeStaticArtifact, StaticArtifactPath: "alpha.json", StaticArtifactFormat: "json"},
				{ID: "beta", Type: missioncontrol.StepTypeLongRunningCode, LongRunningArtifactPath: "service.bin", LongRunningStartupCommand: []string{"go", "build", "./cmd/service"}},
				{ID: "delta", Type: missioncontrol.StepTypeStaticArtifact, StaticArtifactPath: "delta.md", StaticArtifactFormat: "markdown"},
				{ID: "epsilon", Type: missioncontrol.StepTypeOneShotCode, OneShotArtifactPath: "epsilon.go"},
				{ID: "zeta", Type: missioncontrol.StepTypeStaticArtifact, StaticArtifactPath: "zeta.yaml", StaticArtifactFormat: "yaml"},
				{ID: "final", Type: missioncontrol.StepTypeFinalResponse, DependsOn: []string{"zeta"}},
			},
		},
	}
	control, err := missioncontrol.BuildRuntimeControlContext(job, "final")
	if err != nil {
		t.Fatalf("BuildRuntimeControlContext() error = %v", err)
	}
	plan, err := missioncontrol.BuildInspectablePlanContext(job)
	if err != nil {
		t.Fatalf("BuildInspectablePlanContext() error = %v", err)
	}
	requests := make([]missioncontrol.ApprovalRequest, 0, missioncontrol.OperatorStatusApprovalHistoryLimit+2)
	for i := 0; i < missioncontrol.OperatorStatusApprovalHistoryLimit+2; i++ {
		requests = append(requests, missioncontrol.ApprovalRequest{
			JobID:           job.ID,
			StepID:          "step-" + string(rune('a'+i)),
			RequestedAction: missioncontrol.ApprovalRequestedActionStepComplete,
			Scope:           missioncontrol.ApprovalScopeMissionStep,
			State:           missioncontrol.ApprovalStatePending,
			RequestedAt:     time.Date(2026, 3, 24, 12, i, 0, 0, time.UTC),
		})
	}
	history := make([]missioncontrol.AuditEvent, 0, missioncontrol.OperatorStatusRecentAuditLimit+1)
	for i := 0; i < missioncontrol.OperatorStatusRecentAuditLimit+1; i++ {
		history = append(history, missioncontrol.AuditEvent{
			JobID:     job.ID,
			StepID:    "build",
			ToolName:  "status",
			Allowed:   true,
			Timestamp: time.Date(2026, 3, 24, 13, i, 0, 0, time.UTC),
		})
	}
	now := time.Now().UTC().Truncate(time.Second)
	requestBase := now.Add(-8 * time.Hour)
	auditBase := now.Add(-7 * time.Hour)
	pausedAt := now.Add(-6 * time.Hour)
	for i := range requests {
		requests[i].RequestedAt = requestBase.Add(time.Duration(i) * time.Minute)
	}
	for i := range history {
		history[i].Timestamp = auditBase.Add(time.Duration(i) * time.Minute)
	}
	runtime := missioncontrol.JobRuntimeState{
		JobID:            job.ID,
		State:            missioncontrol.JobStatePaused,
		ActiveStepID:     "final",
		InspectablePlan:  &plan,
		PausedReason:     missioncontrol.RuntimePauseReasonOperatorCommand,
		PausedAt:         pausedAt,
		ApprovalRequests: requests,
		AuditHistory:     history,
		CompletedSteps: []missioncontrol.RuntimeStepRecord{
			{StepID: "zeta"},
			{StepID: "gamma"},
			{StepID: "beta", ResultingState: &missioncontrol.RuntimeResultingStateRecord{Kind: string(missioncontrol.StepTypeLongRunningCode), Target: "service.bin", State: "already_present"}},
			{StepID: "alpha"},
			{StepID: "epsilon"},
			{StepID: "delta"},
		},
	}
	storeRoot := filepath.Join(t.TempDir(), "status.store")
	if err := missioncontrol.PersistProjectedRuntimeState(storeRoot, missioncontrol.WriterLockLease{LeaseHolderID: "holder-1"}, &job, runtime, &control, now); err != nil {
		t.Fatalf("PersistProjectedRuntimeState() error = %v", err)
	}

	path := filepath.Join(t.TempDir(), "status.json")
	if err := writeProjectedMissionStatusSnapshot(path, "mission.json", storeRoot, false, job.ID, now); err != nil {
		t.Fatalf("writeProjectedMissionStatusSnapshot() error = %v", err)
	}

	got := readMissionStatusSnapshotFile(t, path)
	if got.RuntimeSummary == nil {
		t.Fatal("RuntimeSummary = nil, want persisted runtime summary")
	}
	if got.RuntimeSummary.State != missioncontrol.JobStatePaused {
		t.Fatalf("RuntimeSummary.State = %q, want %q", got.RuntimeSummary.State, missioncontrol.JobStatePaused)
	}
	if got.RuntimeSummary.PausedReason != missioncontrol.RuntimePauseReasonOperatorCommand {
		t.Fatalf("RuntimeSummary.PausedReason = %q, want %q", got.RuntimeSummary.PausedReason, missioncontrol.RuntimePauseReasonOperatorCommand)
	}
	if got.RuntimeSummary.PausedAt == nil || *got.RuntimeSummary.PausedAt != pausedAt.Format(time.RFC3339) {
		t.Fatalf("RuntimeSummary.PausedAt = %#v, want RFC3339 pause time", got.RuntimeSummary.PausedAt)
	}
	if !reflect.DeepEqual(got.AllowedTools, []string{"read"}) {
		t.Fatalf("AllowedTools = %#v, want %#v", got.AllowedTools, []string{"read"})
	}
	if !reflect.DeepEqual(got.RuntimeSummary.AllowedTools, []string{"read"}) {
		t.Fatalf("RuntimeSummary.AllowedTools = %#v, want %#v", got.RuntimeSummary.AllowedTools, []string{"read"})
	}
	if len(got.RuntimeSummary.Artifacts) != missioncontrol.OperatorStatusArtifactLimit {
		t.Fatalf("RuntimeSummary.Artifacts = %#v, want %d deterministic entries", got.RuntimeSummary.Artifacts, missioncontrol.OperatorStatusArtifactLimit)
	}
	if got.RuntimeSummary.Artifacts[0].StepID != "gamma" || got.RuntimeSummary.Artifacts[0].Path != "zeta.txt" {
		t.Fatalf("RuntimeSummary.Artifacts[0] = %#v, want step_id=%q path=%q", got.RuntimeSummary.Artifacts[0], "gamma", "zeta.txt")
	}
	if got.RuntimeSummary.Artifacts[2].StepID != "beta" || got.RuntimeSummary.Artifacts[2].State != "already_present" {
		t.Fatalf("RuntimeSummary.Artifacts[2] = %#v, want step_id=%q state=%q", got.RuntimeSummary.Artifacts[2], "beta", "already_present")
	}
	if got.RuntimeSummary.Truncation == nil {
		t.Fatal("RuntimeSummary.Truncation = nil, want truncation metadata")
	}
	if got.RuntimeSummary.Truncation.ApprovalHistoryOmitted != 2 {
		t.Fatalf("RuntimeSummary.Truncation.ApprovalHistoryOmitted = %d, want 2", got.RuntimeSummary.Truncation.ApprovalHistoryOmitted)
	}
	if got.RuntimeSummary.Truncation.RecentAuditOmitted != 1 {
		t.Fatalf("RuntimeSummary.Truncation.RecentAuditOmitted = %d, want 1", got.RuntimeSummary.Truncation.RecentAuditOmitted)
	}
	if got.RuntimeSummary.Truncation.ArtifactsOmitted != 1 {
		t.Fatalf("RuntimeSummary.Truncation.ArtifactsOmitted = %d, want 1", got.RuntimeSummary.Truncation.ArtifactsOmitted)
	}
	assertMissionStatusSnapshotMatchesCommittedDurableState(t, got, storeRoot, job.ID, now)
}

func TestMissionStatusSnapshotWritePersistsAuditHistory(t *testing.T) {
	statusFile := filepath.Join(t.TempDir(), "status.json")
	cmd := newMissionBootstrapTestCommand()
	if err := cmd.Flags().Set("mission-status-file", statusFile); err != nil {
		t.Fatalf("Flags().Set(mission-status-file) error = %v", err)
	}

	hub := chat.NewHub(10)
	provider := &missionStatusFixedResponseProvider{content: "unused"}
	ag := agent.NewAgentLoop(hub, provider, provider.GetDefaultModel(), 3, "", nil)
	installMissionRuntimeChangeHook(cmd, ag)

	if err := ag.ActivateMissionStep(testMissionBootstrapJob(), "build"); err != nil {
		t.Fatalf("ActivateMissionStep() error = %v", err)
	}
	if _, err := ag.ProcessDirect("PAUSE job-1", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(PAUSE) error = %v", err)
	}

	got := readMissionStatusSnapshotFile(t, statusFile)
	if got.Runtime == nil {
		t.Fatal("Runtime = nil, want non-nil")
	}
	if got.Runtime.State != missioncontrol.JobStatePaused {
		t.Fatalf("Runtime.State = %q, want %q", got.Runtime.State, missioncontrol.JobStatePaused)
	}
	if len(got.Runtime.AuditHistory) != 2 {
		t.Fatalf("Runtime.AuditHistory count = %d, want 2", len(got.Runtime.AuditHistory))
	}
	expectedAudit := missioncontrol.AppendAuditHistory(nil, missioncontrol.AuditEvent{
		JobID:       "job-1",
		StepID:      "build",
		ToolName:    "pause",
		ActionClass: missioncontrol.AuditActionClassOperatorCommand,
		Result:      missioncontrol.AuditResultApplied,
		Allowed:     true,
		Timestamp:   got.Runtime.AuditHistory[0].Timestamp,
	})
	expectedAudit = missioncontrol.AppendAuditHistory(expectedAudit, missioncontrol.AuditEvent{
		JobID:       "job-1",
		StepID:      "build",
		ToolName:    "pause_ack",
		ActionClass: missioncontrol.AuditActionClassRuntime,
		Result:      missioncontrol.AuditResultApplied,
		Allowed:     true,
		Timestamp:   got.Runtime.AuditHistory[1].Timestamp,
	})
	if !reflect.DeepEqual(got.Runtime.AuditHistory, expectedAudit) {
		t.Fatalf("Runtime.AuditHistory = %#v, want persisted pause and pause_ack audits", got.Runtime.AuditHistory)
	}
}

func TestMissionStatusBootstrapRehydratesAuditHistoryWithoutDuplication(t *testing.T) {
	statusFile := filepath.Join(t.TempDir(), "status.json")
	cmd := newMissionBootstrapTestCommand()
	if err := cmd.Flags().Set("mission-status-file", statusFile); err != nil {
		t.Fatalf("Flags().Set(mission-status-file) error = %v", err)
	}

	job := testMissionBootstrapJob()
	missionFile := writeMissionBootstrapJobFile(t, job)
	persistedAudit := missioncontrol.AppendAuditHistory(nil, missioncontrol.AuditEvent{
		JobID:     job.ID,
		StepID:    "build",
		ToolName:  "pause",
		Allowed:   true,
		Timestamp: time.Date(2026, 3, 24, 12, 0, 0, 0, time.UTC),
	})[0]
	writeMissionStatusSnapshotFile(t, statusFile, missionStatusSnapshot{
		MissionFile: missionFile,
		JobID:       job.ID,
		StepID:      "build",
		Runtime: &missioncontrol.JobRuntimeState{
			JobID:        job.ID,
			State:        missioncontrol.JobStatePaused,
			ActiveStepID: "build",
			PausedReason: missioncontrol.RuntimePauseReasonOperatorCommand,
			AuditHistory: []missioncontrol.AuditEvent{persistedAudit},
		},
		RuntimeControl: runtimeControlForBootstrapStep(t, job, "build"),
	})
	if err := cmd.Flags().Set("mission-file", missionFile); err != nil {
		t.Fatalf("Flags().Set(mission-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step", "build"); err != nil {
		t.Fatalf("Flags().Set(mission-step) error = %v", err)
	}

	ag := newMissionBootstrapTestLoop()
	installMissionRuntimeChangeHook(cmd, ag)

	if err := configureMissionBootstrap(cmd, ag); err != nil {
		t.Fatalf("configureMissionBootstrap() error = %v", err)
	}

	runtime, ok := ag.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if !reflect.DeepEqual(runtime.AuditHistory, []missioncontrol.AuditEvent{persistedAudit}) {
		t.Fatalf("MissionRuntimeState().AuditHistory = %#v, want persisted history %#v", runtime.AuditHistory, []missioncontrol.AuditEvent{persistedAudit})
	}

	now := time.Date(2026, 3, 24, 12, 1, 0, 0, time.UTC)
	if err := writeMissionStatusSnapshotFromCommand(cmd, ag, now); err != nil {
		t.Fatalf("writeMissionStatusSnapshotFromCommand() error = %v", err)
	}

	snapshot := readMissionStatusSnapshotFile(t, statusFile)
	if snapshot.Runtime == nil {
		t.Fatal("snapshot.Runtime = nil, want non-nil")
	}
	if !reflect.DeepEqual(snapshot.Runtime.AuditHistory, []missioncontrol.AuditEvent{persistedAudit}) {
		t.Fatalf("snapshot.Runtime.AuditHistory = %#v, want persisted history %#v", snapshot.Runtime.AuditHistory, []missioncontrol.AuditEvent{persistedAudit})
	}
}

func TestMissionStatusRuntimeChangeHookPersistsApprovalLifecycle(t *testing.T) {
	statusFile := filepath.Join(t.TempDir(), "status.json")
	cmd := newMissionBootstrapTestCommand()
	if err := cmd.Flags().Set("mission-status-file", statusFile); err != nil {
		t.Fatalf("Flags().Set(mission-status-file) error = %v", err)
	}

	hub := chat.NewHub(10)
	provider := &missionStatusFixedResponseProvider{content: "Need approval before continuing."}
	ag := agent.NewAgentLoop(hub, provider, provider.GetDefaultModel(), 3, "", nil)
	installMissionRuntimeChangeHook(cmd, ag)

	job := missioncontrol.Job{
		ID:           "job-1",
		MaxAuthority: missioncontrol.AuthorityTierHigh,
		AllowedTools: []string{"read"},
		Plan: missioncontrol.Plan{
			ID: "plan-1",
			Steps: []missioncontrol.Step{
				{
					ID:      "build",
					Type:    missioncontrol.StepTypeDiscussion,
					Subtype: missioncontrol.StepSubtypeAuthorization,
				},
				{
					ID:        "final",
					Type:      missioncontrol.StepTypeFinalResponse,
					DependsOn: []string{"build"},
				},
			},
		},
	}

	if err := ag.ActivateMissionStep(job, "build"); err != nil {
		t.Fatalf("ActivateMissionStep() error = %v", err)
	}
	if _, err := ag.ProcessDirect("continue", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(continue) error = %v", err)
	}

	waitingSnapshot := readMissionStatusSnapshotFile(t, statusFile)
	if waitingSnapshot.Runtime == nil {
		t.Fatal("waitingSnapshot.Runtime = nil, want non-nil")
	}
	if waitingSnapshot.Runtime.State != missioncontrol.JobStateWaitingUser {
		t.Fatalf("waitingSnapshot.Runtime.State = %q, want %q", waitingSnapshot.Runtime.State, missioncontrol.JobStateWaitingUser)
	}
	if len(waitingSnapshot.Runtime.ApprovalRequests) != 1 || waitingSnapshot.Runtime.ApprovalRequests[0].State != missioncontrol.ApprovalStatePending {
		t.Fatalf("waitingSnapshot.Runtime.ApprovalRequests = %#v, want one pending approval", waitingSnapshot.Runtime.ApprovalRequests)
	}

	if _, err := ag.ProcessDirect("APPROVE job-1 build", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(APPROVE) error = %v", err)
	}

	approvedSnapshot := readMissionStatusSnapshotFile(t, statusFile)
	if approvedSnapshot.Runtime == nil {
		t.Fatal("approvedSnapshot.Runtime = nil, want non-nil")
	}
	if approvedSnapshot.Runtime.State != missioncontrol.JobStatePaused {
		t.Fatalf("approvedSnapshot.Runtime.State = %q, want %q", approvedSnapshot.Runtime.State, missioncontrol.JobStatePaused)
	}
	if len(approvedSnapshot.Runtime.CompletedSteps) != 1 || approvedSnapshot.Runtime.CompletedSteps[0].StepID != "build" {
		t.Fatalf("approvedSnapshot.Runtime.CompletedSteps = %#v, want build completion", approvedSnapshot.Runtime.CompletedSteps)
	}
	if len(approvedSnapshot.Runtime.ApprovalGrants) != 1 || approvedSnapshot.Runtime.ApprovalGrants[0].State != missioncontrol.ApprovalStateGranted {
		t.Fatalf("approvedSnapshot.Runtime.ApprovalGrants = %#v, want one granted approval", approvedSnapshot.Runtime.ApprovalGrants)
	}
	if approvedSnapshot.Runtime.ApprovalRequests[0].SessionChannel != "cli" || approvedSnapshot.Runtime.ApprovalRequests[0].SessionChatID != "direct" {
		t.Fatalf("approvedSnapshot.Runtime.ApprovalRequests[0] session = (%q, %q), want (%q, %q)", approvedSnapshot.Runtime.ApprovalRequests[0].SessionChannel, approvedSnapshot.Runtime.ApprovalRequests[0].SessionChatID, "cli", "direct")
	}
	if approvedSnapshot.Runtime.ApprovalGrants[0].SessionChannel != "cli" || approvedSnapshot.Runtime.ApprovalGrants[0].SessionChatID != "direct" {
		t.Fatalf("approvedSnapshot.Runtime.ApprovalGrants[0] session = (%q, %q), want (%q, %q)", approvedSnapshot.Runtime.ApprovalGrants[0].SessionChannel, approvedSnapshot.Runtime.ApprovalGrants[0].SessionChatID, "cli", "direct")
	}
}

func TestMissionStatusRuntimeChangeHookPersistsNaturalApprovalLifecycle(t *testing.T) {
	statusFile := filepath.Join(t.TempDir(), "status.json")
	cmd := newMissionBootstrapTestCommand()
	if err := cmd.Flags().Set("mission-status-file", statusFile); err != nil {
		t.Fatalf("Flags().Set(mission-status-file) error = %v", err)
	}

	hub := chat.NewHub(10)
	provider := &missionStatusFixedResponseProvider{content: "Need approval before continuing."}
	ag := agent.NewAgentLoop(hub, provider, provider.GetDefaultModel(), 3, "", nil)
	installMissionRuntimeChangeHook(cmd, ag)

	job := missioncontrol.Job{
		ID:           "job-1",
		MaxAuthority: missioncontrol.AuthorityTierHigh,
		AllowedTools: []string{"read"},
		Plan: missioncontrol.Plan{
			ID: "plan-1",
			Steps: []missioncontrol.Step{
				{
					ID:      "build",
					Type:    missioncontrol.StepTypeDiscussion,
					Subtype: missioncontrol.StepSubtypeAuthorization,
				},
				{
					ID:        "final",
					Type:      missioncontrol.StepTypeFinalResponse,
					DependsOn: []string{"build"},
				},
			},
		},
	}

	if err := ag.ActivateMissionStep(job, "build"); err != nil {
		t.Fatalf("ActivateMissionStep() error = %v", err)
	}
	if _, err := ag.ProcessDirect("continue", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(continue) error = %v", err)
	}
	if _, err := ag.ProcessDirect("yes", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(yes) error = %v", err)
	}

	snapshot := readMissionStatusSnapshotFile(t, statusFile)
	if snapshot.Runtime == nil {
		t.Fatal("snapshot.Runtime = nil, want non-nil")
	}
	if snapshot.Runtime.State != missioncontrol.JobStatePaused {
		t.Fatalf("snapshot.Runtime.State = %q, want %q", snapshot.Runtime.State, missioncontrol.JobStatePaused)
	}
	if len(snapshot.Runtime.ApprovalGrants) != 1 || snapshot.Runtime.ApprovalGrants[0].GrantedVia != missioncontrol.ApprovalGrantedViaOperatorReply {
		t.Fatalf("snapshot.Runtime.ApprovalGrants = %#v, want one natural-language approval grant", snapshot.Runtime.ApprovalGrants)
	}
	if snapshot.Runtime.ApprovalRequests[0].SessionChannel != "cli" || snapshot.Runtime.ApprovalRequests[0].SessionChatID != "direct" {
		t.Fatalf("snapshot.Runtime.ApprovalRequests[0] session = (%q, %q), want (%q, %q)", snapshot.Runtime.ApprovalRequests[0].SessionChannel, snapshot.Runtime.ApprovalRequests[0].SessionChatID, "cli", "direct")
	}
	if snapshot.Runtime.ApprovalGrants[0].SessionChannel != "cli" || snapshot.Runtime.ApprovalGrants[0].SessionChatID != "direct" {
		t.Fatalf("snapshot.Runtime.ApprovalGrants[0] session = (%q, %q), want (%q, %q)", snapshot.Runtime.ApprovalGrants[0].SessionChannel, snapshot.Runtime.ApprovalGrants[0].SessionChatID, "cli", "direct")
	}
}

func TestMissionStatusRuntimeChangeHookPersistsRehydratedApprovalLifecycle(t *testing.T) {
	statusFile := filepath.Join(t.TempDir(), "status.json")
	cmd := newMissionBootstrapTestCommand()
	if err := cmd.Flags().Set("mission-status-file", statusFile); err != nil {
		t.Fatalf("Flags().Set(mission-status-file) error = %v", err)
	}

	job := missioncontrol.Job{
		ID:           "job-1",
		MaxAuthority: missioncontrol.AuthorityTierHigh,
		AllowedTools: []string{"read"},
		Plan: missioncontrol.Plan{
			ID: "plan-1",
			Steps: []missioncontrol.Step{
				{
					ID:      "build",
					Type:    missioncontrol.StepTypeDiscussion,
					Subtype: missioncontrol.StepSubtypeAuthorization,
				},
				{
					ID:        "final",
					Type:      missioncontrol.StepTypeFinalResponse,
					DependsOn: []string{"build"},
				},
			},
		},
	}
	missionFile := writeMissionBootstrapJobFile(t, job)
	content := expectedAuthorizationApprovalContent(job.MaxAuthority)
	initialSnapshot := missionStatusSnapshot{
		MissionFile: missionFile,
		JobID:       job.ID,
		StepID:      "build",
		Runtime: &missioncontrol.JobRuntimeState{
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
					Content:         &content,
					SessionChannel:  "telegram",
					SessionChatID:   "chat-42",
					State:           missioncontrol.ApprovalStatePending,
				},
			},
		},
	}
	writeMissionStatusSnapshotFile(t, statusFile, initialSnapshot)
	if err := cmd.Flags().Set("mission-file", missionFile); err != nil {
		t.Fatalf("Flags().Set(mission-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step", "build"); err != nil {
		t.Fatalf("Flags().Set(mission-step) error = %v", err)
	}

	hub := chat.NewHub(10)
	provider := &missionStatusFixedResponseProvider{content: "unused"}
	ag := agent.NewAgentLoop(hub, provider, provider.GetDefaultModel(), 3, "", nil)
	installMissionRuntimeChangeHook(cmd, ag)

	if err := configureMissionBootstrap(cmd, ag); err != nil {
		t.Fatalf("configureMissionBootstrap() error = %v", err)
	}
	if runtime, ok := ag.MissionRuntimeState(); !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	} else if len(runtime.ApprovalRequests) != 1 || runtime.ApprovalRequests[0].SessionChannel != "telegram" || runtime.ApprovalRequests[0].SessionChatID != "chat-42" {
		t.Fatalf("MissionRuntimeState().ApprovalRequests = %#v, want preserved rehydrated session binding", runtime.ApprovalRequests)
	}

	if _, err := ag.ProcessDirect("DENY job-1 build", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(DENY) error = %v", err)
	}

	deniedSnapshot := readMissionStatusSnapshotFile(t, statusFile)
	if deniedSnapshot.Runtime == nil {
		t.Fatal("deniedSnapshot.Runtime = nil, want non-nil")
	}
	if deniedSnapshot.Runtime.State != missioncontrol.JobStateWaitingUser {
		t.Fatalf("deniedSnapshot.Runtime.State = %q, want %q", deniedSnapshot.Runtime.State, missioncontrol.JobStateWaitingUser)
	}
	if len(deniedSnapshot.Runtime.ApprovalRequests) != 1 || deniedSnapshot.Runtime.ApprovalRequests[0].State != missioncontrol.ApprovalStateDenied {
		t.Fatalf("deniedSnapshot.Runtime.ApprovalRequests = %#v, want one denied approval", deniedSnapshot.Runtime.ApprovalRequests)
	}
	if deniedSnapshot.Runtime.ApprovalRequests[0].Content == nil || *deniedSnapshot.Runtime.ApprovalRequests[0].Content != content {
		t.Fatalf("deniedSnapshot.Runtime.ApprovalRequests[0].Content = %#v, want %#v", deniedSnapshot.Runtime.ApprovalRequests[0].Content, content)
	}
	if deniedSnapshot.Runtime.ApprovalRequests[0].SessionChannel != "cli" || deniedSnapshot.Runtime.ApprovalRequests[0].SessionChatID != "direct" {
		t.Fatalf("deniedSnapshot.Runtime.ApprovalRequests[0] session = (%q, %q), want (%q, %q)", deniedSnapshot.Runtime.ApprovalRequests[0].SessionChannel, deniedSnapshot.Runtime.ApprovalRequests[0].SessionChatID, "cli", "direct")
	}
	durableDenied, err := missioncontrol.HydrateCommittedJobRuntimeState(resolveMissionStoreRoot(cmd), job.ID, time.Now().UTC())
	if err != nil {
		t.Fatalf("HydrateCommittedJobRuntimeState(denied) error = %v", err)
	}
	if durableDenied.State != missioncontrol.JobStateWaitingUser {
		t.Fatalf("HydrateCommittedJobRuntimeState(denied).State = %q, want %q", durableDenied.State, missioncontrol.JobStateWaitingUser)
	}
	if len(durableDenied.ApprovalRequests) != 1 || durableDenied.ApprovalRequests[0].State != missioncontrol.ApprovalStateDenied {
		t.Fatalf("HydrateCommittedJobRuntimeState(denied).ApprovalRequests = %#v, want one denied approval", durableDenied.ApprovalRequests)
	}
	assertMissionStatusSnapshotMatchesCommittedDurableState(t, deniedSnapshot, resolveMissionStoreRoot(cmd), job.ID, time.Now().UTC())

	statusFile = filepath.Join(t.TempDir(), "status.json")
	cmd = newMissionBootstrapTestCommand()
	if err := cmd.Flags().Set("mission-status-file", statusFile); err != nil {
		t.Fatalf("Flags().Set(second mission-status-file) error = %v", err)
	}
	writeMissionStatusSnapshotFile(t, statusFile, initialSnapshot)
	if err := cmd.Flags().Set("mission-file", missionFile); err != nil {
		t.Fatalf("Flags().Set(second mission-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step", "build"); err != nil {
		t.Fatalf("Flags().Set(second mission-step) error = %v", err)
	}
	ag = agent.NewAgentLoop(hub, provider, provider.GetDefaultModel(), 3, "", nil)
	installMissionRuntimeChangeHook(cmd, ag)
	if err := configureMissionBootstrap(cmd, ag); err != nil {
		t.Fatalf("configureMissionBootstrap() second boot error = %v", err)
	}

	if _, err := ag.ProcessDirect("APPROVE job-1 build", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(APPROVE) error = %v", err)
	}

	approvedSnapshot := readMissionStatusSnapshotFile(t, statusFile)
	if approvedSnapshot.Runtime == nil {
		t.Fatal("approvedSnapshot.Runtime = nil, want non-nil")
	}
	if approvedSnapshot.Runtime.State != missioncontrol.JobStatePaused {
		t.Fatalf("approvedSnapshot.Runtime.State = %q, want %q", approvedSnapshot.Runtime.State, missioncontrol.JobStatePaused)
	}
	if len(approvedSnapshot.Runtime.CompletedSteps) != 1 || approvedSnapshot.Runtime.CompletedSteps[0].StepID != "build" {
		t.Fatalf("approvedSnapshot.Runtime.CompletedSteps = %#v, want build completion", approvedSnapshot.Runtime.CompletedSteps)
	}
	if len(approvedSnapshot.Runtime.ApprovalGrants) != 1 || approvedSnapshot.Runtime.ApprovalGrants[0].State != missioncontrol.ApprovalStateGranted {
		t.Fatalf("approvedSnapshot.Runtime.ApprovalGrants = %#v, want one granted approval", approvedSnapshot.Runtime.ApprovalGrants)
	}
	if len(approvedSnapshot.Runtime.ApprovalRequests) != 1 || approvedSnapshot.Runtime.ApprovalRequests[0].Content == nil || *approvedSnapshot.Runtime.ApprovalRequests[0].Content != content {
		t.Fatalf("approvedSnapshot.Runtime.ApprovalRequests = %#v, want persisted enriched request content", approvedSnapshot.Runtime.ApprovalRequests)
	}
	if approvedSnapshot.Runtime.ApprovalRequests[0].SessionChannel != "cli" || approvedSnapshot.Runtime.ApprovalRequests[0].SessionChatID != "direct" {
		t.Fatalf("approvedSnapshot.Runtime.ApprovalRequests[0] session = (%q, %q), want (%q, %q)", approvedSnapshot.Runtime.ApprovalRequests[0].SessionChannel, approvedSnapshot.Runtime.ApprovalRequests[0].SessionChatID, "cli", "direct")
	}
	if approvedSnapshot.Runtime.ApprovalGrants[0].SessionChannel != "cli" || approvedSnapshot.Runtime.ApprovalGrants[0].SessionChatID != "direct" {
		t.Fatalf("approvedSnapshot.Runtime.ApprovalGrants[0] session = (%q, %q), want (%q, %q)", approvedSnapshot.Runtime.ApprovalGrants[0].SessionChannel, approvedSnapshot.Runtime.ApprovalGrants[0].SessionChatID, "cli", "direct")
	}
	durableApproved, err := missioncontrol.HydrateCommittedJobRuntimeState(resolveMissionStoreRoot(cmd), job.ID, time.Now().UTC())
	if err != nil {
		t.Fatalf("HydrateCommittedJobRuntimeState(approved) error = %v", err)
	}
	if durableApproved.State != missioncontrol.JobStatePaused {
		t.Fatalf("HydrateCommittedJobRuntimeState(approved).State = %q, want %q", durableApproved.State, missioncontrol.JobStatePaused)
	}
	if len(durableApproved.CompletedSteps) != 1 || durableApproved.CompletedSteps[0].StepID != "build" {
		t.Fatalf("HydrateCommittedJobRuntimeState(approved).CompletedSteps = %#v, want build completion", durableApproved.CompletedSteps)
	}
	if len(durableApproved.ApprovalGrants) != 1 || durableApproved.ApprovalGrants[0].State != missioncontrol.ApprovalStateGranted {
		t.Fatalf("HydrateCommittedJobRuntimeState(approved).ApprovalGrants = %#v, want one granted approval", durableApproved.ApprovalGrants)
	}
	assertMissionStatusSnapshotMatchesCommittedDurableState(t, approvedSnapshot, resolveMissionStoreRoot(cmd), job.ID, time.Now().UTC())
}

func TestMissionStatusRuntimeChangeHookPersistsRehydratedNaturalApprovalLifecycle(t *testing.T) {
	statusFile := filepath.Join(t.TempDir(), "status.json")
	cmd := newMissionBootstrapTestCommand()
	if err := cmd.Flags().Set("mission-status-file", statusFile); err != nil {
		t.Fatalf("Flags().Set(mission-status-file) error = %v", err)
	}

	job := missioncontrol.Job{
		ID:           "job-1",
		MaxAuthority: missioncontrol.AuthorityTierHigh,
		AllowedTools: []string{"read"},
		Plan: missioncontrol.Plan{
			ID: "plan-1",
			Steps: []missioncontrol.Step{
				{
					ID:      "build",
					Type:    missioncontrol.StepTypeDiscussion,
					Subtype: missioncontrol.StepSubtypeAuthorization,
				},
				{
					ID:        "final",
					Type:      missioncontrol.StepTypeFinalResponse,
					DependsOn: []string{"build"},
				},
			},
		},
	}
	missionFile := writeMissionBootstrapJobFile(t, job)
	content := expectedAuthorizationApprovalContent(job.MaxAuthority)
	initialSnapshot := missionStatusSnapshot{
		MissionFile:    missionFile,
		JobID:          job.ID,
		StepID:         "build",
		RuntimeControl: runtimeControlForBootstrapStep(t, job, "build"),
		Runtime: &missioncontrol.JobRuntimeState{
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
					Content:         &content,
					SessionChannel:  "telegram",
					SessionChatID:   "chat-42",
					State:           missioncontrol.ApprovalStatePending,
				},
			},
		},
	}
	writeMissionStatusSnapshotFile(t, statusFile, initialSnapshot)
	if err := cmd.Flags().Set("mission-file", missionFile); err != nil {
		t.Fatalf("Flags().Set(mission-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step", "build"); err != nil {
		t.Fatalf("Flags().Set(mission-step) error = %v", err)
	}

	hub := chat.NewHub(10)
	provider := &missionStatusFixedResponseProvider{content: "unused"}
	ag := agent.NewAgentLoop(hub, provider, provider.GetDefaultModel(), 3, "", nil)
	installMissionRuntimeChangeHook(cmd, ag)

	if err := configureMissionBootstrap(cmd, ag); err != nil {
		t.Fatalf("configureMissionBootstrap() error = %v", err)
	}
	if runtime, ok := ag.MissionRuntimeState(); !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	} else if len(runtime.ApprovalRequests) != 1 || runtime.ApprovalRequests[0].Content == nil || *runtime.ApprovalRequests[0].Content != content || runtime.ApprovalRequests[0].SessionChannel != "telegram" || runtime.ApprovalRequests[0].SessionChatID != "chat-42" {
		t.Fatalf("MissionRuntimeState().ApprovalRequests = %#v, want preserved enriched request content and session binding after rehydration", runtime.ApprovalRequests)
	}
	if _, err := ag.ProcessDirect("no", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(no) error = %v", err)
	}

	deniedSnapshot := readMissionStatusSnapshotFile(t, statusFile)
	if deniedSnapshot.Runtime == nil {
		t.Fatal("deniedSnapshot.Runtime = nil, want non-nil")
	}
	if deniedSnapshot.Runtime.State != missioncontrol.JobStateWaitingUser {
		t.Fatalf("deniedSnapshot.Runtime.State = %q, want %q", deniedSnapshot.Runtime.State, missioncontrol.JobStateWaitingUser)
	}
	if len(deniedSnapshot.Runtime.ApprovalRequests) != 1 || deniedSnapshot.Runtime.ApprovalRequests[0].State != missioncontrol.ApprovalStateDenied {
		t.Fatalf("deniedSnapshot.Runtime.ApprovalRequests = %#v, want one denied approval", deniedSnapshot.Runtime.ApprovalRequests)
	}
	if deniedSnapshot.Runtime.ApprovalRequests[0].Content == nil || *deniedSnapshot.Runtime.ApprovalRequests[0].Content != content {
		t.Fatalf("deniedSnapshot.Runtime.ApprovalRequests[0].Content = %#v, want %#v", deniedSnapshot.Runtime.ApprovalRequests[0].Content, content)
	}
	if deniedSnapshot.Runtime.ApprovalRequests[0].SessionChannel != "cli" || deniedSnapshot.Runtime.ApprovalRequests[0].SessionChatID != "direct" {
		t.Fatalf("deniedSnapshot.Runtime.ApprovalRequests[0] session = (%q, %q), want (%q, %q)", deniedSnapshot.Runtime.ApprovalRequests[0].SessionChannel, deniedSnapshot.Runtime.ApprovalRequests[0].SessionChatID, "cli", "direct")
	}
	assertMissionStatusSnapshotMatchesCommittedDurableState(t, deniedSnapshot, resolveMissionStoreRoot(cmd), job.ID, time.Now().UTC())

	statusFile = filepath.Join(t.TempDir(), "status.json")
	cmd = newMissionBootstrapTestCommand()
	if err := cmd.Flags().Set("mission-status-file", statusFile); err != nil {
		t.Fatalf("Flags().Set(second mission-status-file) error = %v", err)
	}
	writeMissionStatusSnapshotFile(t, statusFile, initialSnapshot)
	if err := cmd.Flags().Set("mission-file", missionFile); err != nil {
		t.Fatalf("Flags().Set(second mission-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step", "build"); err != nil {
		t.Fatalf("Flags().Set(second mission-step) error = %v", err)
	}
	ag = agent.NewAgentLoop(hub, provider, provider.GetDefaultModel(), 3, "", nil)
	installMissionRuntimeChangeHook(cmd, ag)
	if err := configureMissionBootstrap(cmd, ag); err != nil {
		t.Fatalf("configureMissionBootstrap() second boot error = %v", err)
	}
	if _, err := ag.ProcessDirect("yes", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(yes) error = %v", err)
	}

	approvedSnapshot := readMissionStatusSnapshotFile(t, statusFile)
	if approvedSnapshot.Runtime == nil {
		t.Fatal("approvedSnapshot.Runtime = nil, want non-nil")
	}
	if approvedSnapshot.Runtime.State != missioncontrol.JobStatePaused {
		t.Fatalf("approvedSnapshot.Runtime.State = %q, want %q", approvedSnapshot.Runtime.State, missioncontrol.JobStatePaused)
	}
	if len(approvedSnapshot.Runtime.ApprovalGrants) != 1 || approvedSnapshot.Runtime.ApprovalGrants[0].GrantedVia != missioncontrol.ApprovalGrantedViaOperatorReply {
		t.Fatalf("approvedSnapshot.Runtime.ApprovalGrants = %#v, want one natural-language approval grant", approvedSnapshot.Runtime.ApprovalGrants)
	}
	if len(approvedSnapshot.Runtime.ApprovalRequests) != 1 || approvedSnapshot.Runtime.ApprovalRequests[0].Content == nil || *approvedSnapshot.Runtime.ApprovalRequests[0].Content != content {
		t.Fatalf("approvedSnapshot.Runtime.ApprovalRequests = %#v, want persisted enriched request content", approvedSnapshot.Runtime.ApprovalRequests)
	}
	assertMissionStatusSnapshotMatchesCommittedDurableState(t, approvedSnapshot, resolveMissionStoreRoot(cmd), job.ID, time.Now().UTC())
	if approvedSnapshot.Runtime.ApprovalRequests[0].SessionChannel != "cli" || approvedSnapshot.Runtime.ApprovalRequests[0].SessionChatID != "direct" {
		t.Fatalf("approvedSnapshot.Runtime.ApprovalRequests[0] session = (%q, %q), want (%q, %q)", approvedSnapshot.Runtime.ApprovalRequests[0].SessionChannel, approvedSnapshot.Runtime.ApprovalRequests[0].SessionChatID, "cli", "direct")
	}
	if approvedSnapshot.Runtime.ApprovalGrants[0].SessionChannel != "cli" || approvedSnapshot.Runtime.ApprovalGrants[0].SessionChatID != "direct" {
		t.Fatalf("approvedSnapshot.Runtime.ApprovalGrants[0] session = (%q, %q), want (%q, %q)", approvedSnapshot.Runtime.ApprovalGrants[0].SessionChannel, approvedSnapshot.Runtime.ApprovalGrants[0].SessionChatID, "cli", "direct")
	}
}

func TestMissionStatusRuntimeChangeHookPersistsApprovalExpiryLifecycle(t *testing.T) {
	statusFile := filepath.Join(t.TempDir(), "status.json")
	cmd := newMissionBootstrapTestCommand()
	if err := cmd.Flags().Set("mission-status-file", statusFile); err != nil {
		t.Fatalf("Flags().Set(mission-status-file) error = %v", err)
	}

	hub := chat.NewHub(10)
	provider := &missionStatusFixedResponseProvider{content: "Need approval before continuing."}
	ag := agent.NewAgentLoop(hub, provider, provider.GetDefaultModel(), 3, "", nil)
	installMissionRuntimeChangeHook(cmd, ag)

	job := missioncontrol.Job{
		ID:           "job-1",
		MaxAuthority: missioncontrol.AuthorityTierHigh,
		AllowedTools: []string{"read"},
		Plan: missioncontrol.Plan{
			ID: "plan-1",
			Steps: []missioncontrol.Step{
				{
					ID:      "build",
					Type:    missioncontrol.StepTypeDiscussion,
					Subtype: missioncontrol.StepSubtypeAuthorization,
				},
				{
					ID:        "final",
					Type:      missioncontrol.StepTypeFinalResponse,
					DependsOn: []string{"build"},
				},
			},
		},
	}

	if err := ag.ActivateMissionStep(job, "build"); err != nil {
		t.Fatalf("ActivateMissionStep() error = %v", err)
	}
	if _, err := ag.ProcessDirect("continue", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(continue) error = %v", err)
	}
	if _, err := ag.ProcessDirect("timeout", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(timeout) error = %v", err)
	}

	snapshot := readMissionStatusSnapshotFile(t, statusFile)
	if snapshot.Runtime == nil {
		t.Fatal("snapshot.Runtime = nil, want non-nil")
	}
	if len(snapshot.Runtime.ApprovalRequests) != 1 || snapshot.Runtime.ApprovalRequests[0].ExpiresAt.IsZero() {
		t.Fatalf("snapshot.Runtime.ApprovalRequests = %#v, want one stamped approval request", snapshot.Runtime.ApprovalRequests)
	}
	if len(snapshot.Runtime.ApprovalRequests) != 1 || snapshot.Runtime.ApprovalRequests[0].State != missioncontrol.ApprovalStateExpired {
		t.Fatalf("snapshot.Runtime.ApprovalRequests = %#v, want one expired approval", snapshot.Runtime.ApprovalRequests)
	}
	if snapshot.Runtime.ApprovalRequests[0].ExpiresAt.IsZero() {
		t.Fatalf("snapshot.Runtime.ApprovalRequests[0].ExpiresAt = %v, want non-zero", snapshot.Runtime.ApprovalRequests[0].ExpiresAt)
	}
}

func TestMissionStatusBootstrapRehydratedYesDoesNotBindExpiredApproval(t *testing.T) {
	statusFile := filepath.Join(t.TempDir(), "status.json")
	cmd := newMissionBootstrapTestCommand()
	if err := cmd.Flags().Set("mission-status-file", statusFile); err != nil {
		t.Fatalf("Flags().Set(mission-status-file) error = %v", err)
	}

	job := missioncontrol.Job{
		ID:           "job-1",
		MaxAuthority: missioncontrol.AuthorityTierHigh,
		AllowedTools: []string{"read"},
		Plan: missioncontrol.Plan{
			ID: "plan-1",
			Steps: []missioncontrol.Step{
				{
					ID:      "build",
					Type:    missioncontrol.StepTypeDiscussion,
					Subtype: missioncontrol.StepSubtypeAuthorization,
				},
				{
					ID:        "final",
					Type:      missioncontrol.StepTypeFinalResponse,
					DependsOn: []string{"build"},
				},
			},
		},
	}
	missionFile := writeMissionBootstrapJobFile(t, job)
	expiredAt := time.Now().Add(-1 * time.Minute)
	writeMissionStatusSnapshotFile(t, statusFile, missionStatusSnapshot{
		MissionFile: missionFile,
		JobID:       job.ID,
		StepID:      "build",
		Runtime: &missioncontrol.JobRuntimeState{
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
					ExpiresAt:       expiredAt,
				},
			},
		},
	})
	if err := cmd.Flags().Set("mission-file", missionFile); err != nil {
		t.Fatalf("Flags().Set(mission-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step", "build"); err != nil {
		t.Fatalf("Flags().Set(mission-step) error = %v", err)
	}

	ag := newMissionBootstrapTestLoop()
	installMissionRuntimeChangeHook(cmd, ag)
	if err := configureMissionBootstrap(cmd, ag); err != nil {
		t.Fatalf("configureMissionBootstrap() error = %v", err)
	}

	snapshot := readMissionStatusSnapshotFile(t, statusFile)
	if snapshot.Runtime == nil {
		t.Fatal("snapshot.Runtime = nil, want non-nil")
	}
	if len(snapshot.Runtime.ApprovalRequests) != 1 || snapshot.Runtime.ApprovalRequests[0].State != missioncontrol.ApprovalStateExpired {
		t.Fatalf("snapshot.Runtime.ApprovalRequests = %#v, want one expired approval immediately after bootstrap", snapshot.Runtime.ApprovalRequests)
	}

	resp, err := ag.ProcessDirect("yes", 2*time.Second)
	if err != nil {
		t.Fatalf("ProcessDirect(yes) error = %v", err)
	}
	if !strings.Contains(resp, "(stub) Echo") {
		t.Fatalf("ProcessDirect(yes) response = %q, want provider fallback", resp)
	}

	runtime, ok := ag.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if len(runtime.ApprovalRequests) != 1 || runtime.ApprovalRequests[0].State != missioncontrol.ApprovalStateExpired {
		t.Fatalf("MissionRuntimeState().ApprovalRequests = %#v, want one expired approval", runtime.ApprovalRequests)
	}

	snapshot = readMissionStatusSnapshotFile(t, statusFile)
	if snapshot.Runtime == nil {
		t.Fatal("snapshot.Runtime = nil, want non-nil")
	}
	if len(snapshot.Runtime.ApprovalRequests) != 1 || snapshot.Runtime.ApprovalRequests[0].State != missioncontrol.ApprovalStateExpired {
		t.Fatalf("snapshot.Runtime.ApprovalRequests = %#v, want one expired approval", snapshot.Runtime.ApprovalRequests)
	}
}

func TestMissionStatusBootstrapRehydratedApproveDoesNotBindExpiredApproval(t *testing.T) {
	statusFile := filepath.Join(t.TempDir(), "status.json")
	cmd := newMissionBootstrapTestCommand()
	if err := cmd.Flags().Set("mission-status-file", statusFile); err != nil {
		t.Fatalf("Flags().Set(mission-status-file) error = %v", err)
	}

	job := missioncontrol.Job{
		ID:           "job-1",
		MaxAuthority: missioncontrol.AuthorityTierHigh,
		AllowedTools: []string{"read"},
		Plan: missioncontrol.Plan{
			ID: "plan-1",
			Steps: []missioncontrol.Step{
				{
					ID:      "build",
					Type:    missioncontrol.StepTypeDiscussion,
					Subtype: missioncontrol.StepSubtypeAuthorization,
				},
				{
					ID:        "final",
					Type:      missioncontrol.StepTypeFinalResponse,
					DependsOn: []string{"build"},
				},
			},
		},
	}
	missionFile := writeMissionBootstrapJobFile(t, job)
	expiredAt := time.Now().Add(-1 * time.Minute)
	writeMissionStatusSnapshotFile(t, statusFile, missionStatusSnapshot{
		MissionFile: missionFile,
		JobID:       job.ID,
		StepID:      "build",
		Runtime: &missioncontrol.JobRuntimeState{
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
					ExpiresAt:       expiredAt,
				},
			},
			UpdatedAt: expiredAt.Add(-1 * time.Minute),
		},
		RuntimeControl: &missioncontrol.RuntimeControlContext{
			JobID: job.ID,
			Step: missioncontrol.Step{
				ID:      "build",
				Type:    missioncontrol.StepTypeDiscussion,
				Subtype: missioncontrol.StepSubtypeAuthorization,
			},
		},
	})
	if err := cmd.Flags().Set("mission-file", missionFile); err != nil {
		t.Fatalf("Flags().Set(mission-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step", "build"); err != nil {
		t.Fatalf("Flags().Set(mission-step) error = %v", err)
	}

	ag := newMissionBootstrapTestLoop()
	installMissionRuntimeChangeHook(cmd, ag)
	if err := configureMissionBootstrap(cmd, ag); err != nil {
		t.Fatalf("configureMissionBootstrap() error = %v", err)
	}

	_, err := ag.ProcessDirect("APPROVE job-1 build", 2*time.Second)
	if err == nil {
		t.Fatal("ProcessDirect(APPROVE) error = nil, want expired approval rejection")
	}
	if !strings.Contains(err.Error(), "no pending approval request matches the active job and step") {
		t.Fatalf("ProcessDirect(APPROVE) error = %q, want expired approval rejection", err)
	}

	runtime, ok := ag.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if len(runtime.ApprovalRequests) != 1 || runtime.ApprovalRequests[0].State != missioncontrol.ApprovalStateExpired {
		t.Fatalf("MissionRuntimeState().ApprovalRequests = %#v, want one expired approval", runtime.ApprovalRequests)
	}
}

func TestMissionStatusBootstrapRehydratedApproveUsesLatestNonSupersededApproval(t *testing.T) {
	statusFile := filepath.Join(t.TempDir(), "status.json")
	cmd := newMissionBootstrapTestCommand()
	if err := cmd.Flags().Set("mission-status-file", statusFile); err != nil {
		t.Fatalf("Flags().Set(mission-status-file) error = %v", err)
	}

	job := missioncontrol.Job{
		ID:           "job-1",
		MaxAuthority: missioncontrol.AuthorityTierHigh,
		AllowedTools: []string{"read"},
		Plan: missioncontrol.Plan{
			ID: "plan-1",
			Steps: []missioncontrol.Step{
				{
					ID:      "build",
					Type:    missioncontrol.StepTypeDiscussion,
					Subtype: missioncontrol.StepSubtypeAuthorization,
				},
				{
					ID:        "final",
					Type:      missioncontrol.StepTypeFinalResponse,
					DependsOn: []string{"build"},
				},
			},
		},
	}
	missionFile := writeMissionBootstrapJobFile(t, job)
	now := time.Now()
	writeMissionStatusSnapshotFile(t, statusFile, missionStatusSnapshot{
		MissionFile: missionFile,
		JobID:       job.ID,
		StepID:      "build",
		Runtime: &missioncontrol.JobRuntimeState{
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
					State:           missioncontrol.ApprovalStateSuperseded,
					RequestedAt:     now.Add(-2 * time.Minute),
					ResolvedAt:      now.Add(-1 * time.Minute),
					SupersededAt:    now.Add(-1 * time.Minute),
				},
				{
					JobID:           job.ID,
					StepID:          "build",
					RequestedAction: missioncontrol.ApprovalRequestedActionStepComplete,
					Scope:           missioncontrol.ApprovalScopeMissionStep,
					State:           missioncontrol.ApprovalStatePending,
					RequestedAt:     now,
				},
			},
		},
	})
	if err := cmd.Flags().Set("mission-file", missionFile); err != nil {
		t.Fatalf("Flags().Set(mission-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step", "build"); err != nil {
		t.Fatalf("Flags().Set(mission-step) error = %v", err)
	}

	ag := newMissionBootstrapTestLoop()
	if err := configureMissionBootstrap(cmd, ag); err != nil {
		t.Fatalf("configureMissionBootstrap() error = %v", err)
	}

	if _, err := ag.ProcessDirect("APPROVE job-1 build", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(APPROVE) error = %v", err)
	}

	runtime, ok := ag.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if len(runtime.ApprovalRequests) != 2 || runtime.ApprovalRequests[0].State != missioncontrol.ApprovalStateSuperseded || runtime.ApprovalRequests[1].State != missioncontrol.ApprovalStateGranted {
		t.Fatalf("MissionRuntimeState().ApprovalRequests = %#v, want superseded then granted approvals", runtime.ApprovalRequests)
	}
}

func TestMissionStatusRuntimeChangeHookPersistsSupersededApprovalLifecycle(t *testing.T) {
	statusFile := filepath.Join(t.TempDir(), "status.json")
	cmd := newMissionBootstrapTestCommand()
	if err := cmd.Flags().Set("mission-status-file", statusFile); err != nil {
		t.Fatalf("Flags().Set(mission-status-file) error = %v", err)
	}

	job := missioncontrol.Job{
		ID:           "job-1",
		MaxAuthority: missioncontrol.AuthorityTierHigh,
		AllowedTools: []string{"read"},
		Plan: missioncontrol.Plan{
			ID: "plan-1",
			Steps: []missioncontrol.Step{
				{
					ID:      "build",
					Type:    missioncontrol.StepTypeDiscussion,
					Subtype: missioncontrol.StepSubtypeAuthorization,
				},
				{
					ID:        "final",
					Type:      missioncontrol.StepTypeFinalResponse,
					DependsOn: []string{"build"},
				},
			},
		},
	}
	missionFile := writeMissionBootstrapJobFile(t, job)
	now := time.Now()
	writeMissionStatusSnapshotFile(t, statusFile, missionStatusSnapshot{
		MissionFile:    missionFile,
		JobID:          job.ID,
		StepID:         "build",
		RuntimeControl: runtimeControlForBootstrapStep(t, job, "build"),
		Runtime: &missioncontrol.JobRuntimeState{
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
					State:           missioncontrol.ApprovalStateSuperseded,
					RequestedAt:     now.Add(-2 * time.Minute),
					ResolvedAt:      now.Add(-1 * time.Minute),
					SupersededAt:    now.Add(-1 * time.Minute),
				},
				{
					JobID:           job.ID,
					StepID:          "build",
					RequestedAction: missioncontrol.ApprovalRequestedActionStepComplete,
					Scope:           missioncontrol.ApprovalScopeMissionStep,
					State:           missioncontrol.ApprovalStatePending,
					RequestedAt:     now,
				},
			},
		},
	})
	if err := cmd.Flags().Set("mission-file", missionFile); err != nil {
		t.Fatalf("Flags().Set(mission-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step", "build"); err != nil {
		t.Fatalf("Flags().Set(mission-step) error = %v", err)
	}

	hub := chat.NewHub(10)
	provider := &missionStatusFixedResponseProvider{content: "unused"}
	ag := agent.NewAgentLoop(hub, provider, provider.GetDefaultModel(), 3, "", nil)
	installMissionRuntimeChangeHook(cmd, ag)
	if err := configureMissionBootstrap(cmd, ag); err != nil {
		t.Fatalf("configureMissionBootstrap() error = %v", err)
	}

	if _, err := ag.ProcessDirect("APPROVE job-1 build", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(APPROVE) error = %v", err)
	}

	snapshot := readMissionStatusSnapshotFile(t, statusFile)
	if snapshot.Runtime == nil {
		t.Fatal("snapshot.Runtime = nil, want non-nil")
	}
	if len(snapshot.Runtime.ApprovalRequests) != 2 || snapshot.Runtime.ApprovalRequests[0].State != missioncontrol.ApprovalStateSuperseded || snapshot.Runtime.ApprovalRequests[1].State != missioncontrol.ApprovalStateGranted {
		t.Fatalf("snapshot.Runtime.ApprovalRequests = %#v, want superseded then granted approvals", snapshot.Runtime.ApprovalRequests)
	}
}

func TestMissionStatusRuntimeChangeHookPersistsPauseResumeAbortLifecycle(t *testing.T) {
	statusFile := filepath.Join(t.TempDir(), "status.json")
	cmd := newMissionBootstrapTestCommand()
	if err := cmd.Flags().Set("mission-status-file", statusFile); err != nil {
		t.Fatalf("Flags().Set(mission-status-file) error = %v", err)
	}

	hub := chat.NewHub(10)
	provider := &missionStatusFixedResponseProvider{content: "unused"}
	ag := agent.NewAgentLoop(hub, provider, provider.GetDefaultModel(), 3, "", nil)
	installMissionRuntimeChangeHook(cmd, ag)

	if err := ag.ActivateMissionStep(testMissionBootstrapJob(), "build"); err != nil {
		t.Fatalf("ActivateMissionStep() error = %v", err)
	}
	if _, err := ag.ProcessDirect("PAUSE job-1", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(PAUSE) error = %v", err)
	}

	pausedSnapshot := readMissionStatusSnapshotFile(t, statusFile)
	if pausedSnapshot.Runtime == nil {
		t.Fatal("pausedSnapshot.Runtime = nil, want non-nil")
	}
	if pausedSnapshot.Runtime.State != missioncontrol.JobStatePaused {
		t.Fatalf("pausedSnapshot.Runtime.State = %q, want %q", pausedSnapshot.Runtime.State, missioncontrol.JobStatePaused)
	}
	if pausedSnapshot.Runtime.ActiveStepID != "build" {
		t.Fatalf("pausedSnapshot.Runtime.ActiveStepID = %q, want %q", pausedSnapshot.Runtime.ActiveStepID, "build")
	}
	if len(pausedSnapshot.Runtime.CompletedSteps) != 0 {
		t.Fatalf("pausedSnapshot.Runtime.CompletedSteps = %#v, want empty", pausedSnapshot.Runtime.CompletedSteps)
	}
	if pausedSnapshot.RuntimeControl == nil || pausedSnapshot.RuntimeControl.Step.ID != "build" {
		t.Fatalf("pausedSnapshot.RuntimeControl = %#v, want persisted build control", pausedSnapshot.RuntimeControl)
	}
	assertMissionStatusSnapshotMatchesCommittedDurableState(t, pausedSnapshot, resolveMissionStoreRoot(cmd), "job-1", time.Now().UTC())

	ag.ClearMissionStep()

	tornDownSnapshot := readMissionStatusSnapshotFile(t, statusFile)
	if tornDownSnapshot.Active {
		t.Fatal("tornDownSnapshot.Active = true, want false after teardown")
	}
	if tornDownSnapshot.Runtime == nil {
		t.Fatal("tornDownSnapshot.Runtime = nil, want non-nil")
	}
	if tornDownSnapshot.Runtime.State != missioncontrol.JobStatePaused {
		t.Fatalf("tornDownSnapshot.Runtime.State = %q, want %q", tornDownSnapshot.Runtime.State, missioncontrol.JobStatePaused)
	}
	if tornDownSnapshot.Runtime.ActiveStepID != "build" {
		t.Fatalf("tornDownSnapshot.Runtime.ActiveStepID = %q, want %q", tornDownSnapshot.Runtime.ActiveStepID, "build")
	}
	if tornDownSnapshot.RuntimeControl == nil || tornDownSnapshot.RuntimeControl.Step.ID != "build" {
		t.Fatalf("tornDownSnapshot.RuntimeControl = %#v, want persisted build control", tornDownSnapshot.RuntimeControl)
	}
	assertMissionStatusSnapshotMatchesCommittedDurableState(t, tornDownSnapshot, resolveMissionStoreRoot(cmd), "job-1", time.Now().UTC())

	if _, err := ag.ProcessDirect("RESUME job-1", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(RESUME) error = %v", err)
	}

	resumedSnapshot := readMissionStatusSnapshotFile(t, statusFile)
	if resumedSnapshot.Runtime == nil {
		t.Fatal("resumedSnapshot.Runtime = nil, want non-nil")
	}
	if resumedSnapshot.Runtime.State != missioncontrol.JobStateRunning {
		t.Fatalf("resumedSnapshot.Runtime.State = %q, want %q", resumedSnapshot.Runtime.State, missioncontrol.JobStateRunning)
	}
	if resumedSnapshot.Runtime.ActiveStepID != "build" {
		t.Fatalf("resumedSnapshot.Runtime.ActiveStepID = %q, want %q", resumedSnapshot.Runtime.ActiveStepID, "build")
	}
	if resumedSnapshot.RuntimeControl == nil || resumedSnapshot.RuntimeControl.Step.ID != "build" {
		t.Fatalf("resumedSnapshot.RuntimeControl = %#v, want persisted build control", resumedSnapshot.RuntimeControl)
	}
	assertMissionStatusSnapshotMatchesCommittedDurableState(t, resumedSnapshot, resolveMissionStoreRoot(cmd), "job-1", time.Now().UTC())
	durableResumed, err := missioncontrol.HydrateCommittedJobRuntimeState(resolveMissionStoreRoot(cmd), "job-1", time.Now().UTC())
	if err != nil {
		t.Fatalf("HydrateCommittedJobRuntimeState(resumed) error = %v", err)
	}
	if durableResumed.State != missioncontrol.JobStateRunning || durableResumed.ActiveStepID != "build" {
		t.Fatalf("HydrateCommittedJobRuntimeState(resumed) = %#v, want running build runtime", durableResumed)
	}

	ag.ClearMissionStep()

	if _, err := ag.ProcessDirect("ABORT job-1", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(ABORT) error = %v", err)
	}

	abortedSnapshot := readMissionStatusSnapshotFile(t, statusFile)
	if abortedSnapshot.Runtime == nil {
		t.Fatal("abortedSnapshot.Runtime = nil, want non-nil")
	}
	if abortedSnapshot.Runtime.State != missioncontrol.JobStateAborted {
		t.Fatalf("abortedSnapshot.Runtime.State = %q, want %q", abortedSnapshot.Runtime.State, missioncontrol.JobStateAborted)
	}
	if abortedSnapshot.Runtime.AbortedReason != missioncontrol.RuntimeAbortReasonOperatorCommand {
		t.Fatalf("abortedSnapshot.Runtime.AbortedReason = %q, want %q", abortedSnapshot.Runtime.AbortedReason, missioncontrol.RuntimeAbortReasonOperatorCommand)
	}
	if abortedSnapshot.Active {
		t.Fatal("abortedSnapshot.Active = true, want false")
	}
	if abortedSnapshot.Runtime.InspectablePlan == nil {
		t.Fatal("abortedSnapshot.Runtime.InspectablePlan = nil, want persisted inspectable plan")
	}
	if len(abortedSnapshot.Runtime.InspectablePlan.Steps) != len(testMissionBootstrapJob().Plan.Steps) {
		t.Fatalf("len(abortedSnapshot.Runtime.InspectablePlan.Steps) = %d, want %d", len(abortedSnapshot.Runtime.InspectablePlan.Steps), len(testMissionBootstrapJob().Plan.Steps))
	}
	if abortedSnapshot.RuntimeControl != nil {
		t.Fatalf("abortedSnapshot.RuntimeControl = %#v, want nil for terminal aborted snapshot", abortedSnapshot.RuntimeControl)
	}
	assertMissionStatusSnapshotMatchesCommittedDurableState(t, abortedSnapshot, resolveMissionStoreRoot(cmd), "job-1", time.Now().UTC())
	durableAborted, err := missioncontrol.HydrateCommittedJobRuntimeState(resolveMissionStoreRoot(cmd), "job-1", time.Now().UTC())
	if err != nil {
		t.Fatalf("HydrateCommittedJobRuntimeState(aborted) error = %v", err)
	}
	if durableAborted.State != missioncontrol.JobStateAborted {
		t.Fatalf("HydrateCommittedJobRuntimeState(aborted).State = %q, want %q", durableAborted.State, missioncontrol.JobStateAborted)
	}
}

func TestMissionStatusRuntimeChangeHookPersistsDurableAbortFromWaitingUserAfterTeardown(t *testing.T) {
	statusFile := filepath.Join(t.TempDir(), "status.json")
	cmd := newMissionBootstrapTestCommand()
	if err := cmd.Flags().Set("mission-status-file", statusFile); err != nil {
		t.Fatalf("Flags().Set(mission-status-file) error = %v", err)
	}

	hub := chat.NewHub(10)
	provider := &missionStatusFixedResponseProvider{content: "Need approval before continuing."}
	ag := agent.NewAgentLoop(hub, provider, provider.GetDefaultModel(), 3, "", nil)
	installMissionRuntimeChangeHook(cmd, ag)

	job := missioncontrol.Job{
		ID:           "job-1",
		MaxAuthority: missioncontrol.AuthorityTierHigh,
		AllowedTools: []string{"read"},
		Plan: missioncontrol.Plan{
			ID: "plan-1",
			Steps: []missioncontrol.Step{
				{
					ID:      "build",
					Type:    missioncontrol.StepTypeDiscussion,
					Subtype: missioncontrol.StepSubtypeAuthorization,
				},
				{
					ID:        "final",
					Type:      missioncontrol.StepTypeFinalResponse,
					DependsOn: []string{"build"},
				},
			},
		},
	}

	if err := ag.ActivateMissionStep(job, "build"); err != nil {
		t.Fatalf("ActivateMissionStep() error = %v", err)
	}
	if _, err := ag.ProcessDirect("continue", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(continue) error = %v", err)
	}

	ag.ClearMissionStep()

	tornDownSnapshot := readMissionStatusSnapshotFile(t, statusFile)
	if tornDownSnapshot.Active {
		t.Fatal("tornDownSnapshot.Active = true, want false after teardown")
	}
	if tornDownSnapshot.Runtime == nil {
		t.Fatal("tornDownSnapshot.Runtime = nil, want non-nil")
	}
	if tornDownSnapshot.Runtime.State != missioncontrol.JobStateWaitingUser {
		t.Fatalf("tornDownSnapshot.Runtime.State = %q, want %q", tornDownSnapshot.Runtime.State, missioncontrol.JobStateWaitingUser)
	}
	if len(tornDownSnapshot.Runtime.ApprovalRequests) != 1 || tornDownSnapshot.Runtime.ApprovalRequests[0].State != missioncontrol.ApprovalStatePending {
		t.Fatalf("tornDownSnapshot.Runtime.ApprovalRequests = %#v, want one pending approval", tornDownSnapshot.Runtime.ApprovalRequests)
	}
	if tornDownSnapshot.RuntimeControl == nil || tornDownSnapshot.RuntimeControl.Step.ID != "build" {
		t.Fatalf("tornDownSnapshot.RuntimeControl = %#v, want persisted build control", tornDownSnapshot.RuntimeControl)
	}

	if _, err := ag.ProcessDirect("ABORT job-1", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(ABORT) error = %v", err)
	}

	abortedSnapshot := readMissionStatusSnapshotFile(t, statusFile)
	if abortedSnapshot.Runtime == nil {
		t.Fatal("abortedSnapshot.Runtime = nil, want non-nil")
	}
	if abortedSnapshot.Runtime.State != missioncontrol.JobStateAborted {
		t.Fatalf("abortedSnapshot.Runtime.State = %q, want %q", abortedSnapshot.Runtime.State, missioncontrol.JobStateAborted)
	}
	if abortedSnapshot.Runtime.AbortedReason != missioncontrol.RuntimeAbortReasonOperatorCommand {
		t.Fatalf("abortedSnapshot.Runtime.AbortedReason = %q, want %q", abortedSnapshot.Runtime.AbortedReason, missioncontrol.RuntimeAbortReasonOperatorCommand)
	}
	if abortedSnapshot.Active {
		t.Fatal("abortedSnapshot.Active = true, want false")
	}
	if abortedSnapshot.Runtime.InspectablePlan == nil {
		t.Fatal("abortedSnapshot.Runtime.InspectablePlan = nil, want persisted inspectable plan")
	}
	if len(abortedSnapshot.Runtime.InspectablePlan.Steps) != 2 {
		t.Fatalf("len(abortedSnapshot.Runtime.InspectablePlan.Steps) = %d, want %d", len(abortedSnapshot.Runtime.InspectablePlan.Steps), 2)
	}
	if abortedSnapshot.RuntimeControl != nil {
		t.Fatalf("abortedSnapshot.RuntimeControl = %#v, want nil for terminal aborted snapshot", abortedSnapshot.RuntimeControl)
	}
}

func TestWriteMissionStatusSnapshotLeavesNoTempFileOnSuccess(t *testing.T) {
	ag := newMissionBootstrapTestLoop()
	dir := t.TempDir()
	path := filepath.Join(dir, "status.json")
	now := time.Date(2026, 3, 19, 12, 0, 0, 456, time.UTC)

	if err := writeMissionStatusSnapshot(path, "", ag, now); err != nil {
		t.Fatalf("writeMissionStatusSnapshot() error = %v", err)
	}

	got := readMissionStatusSnapshotFile(t, path)
	if got.UpdatedAt != now.Format(time.RFC3339Nano) {
		t.Fatalf("UpdatedAt = %q, want %q", got.UpdatedAt, now.Format(time.RFC3339Nano))
	}
	assertNoAtomicTempFiles(t, dir, filepath.Base(path))
}

func TestWriteMissionStatusSnapshotAllowedToolsIntersectedAndSorted(t *testing.T) {
	ag := newMissionBootstrapTestLoop()
	job := missioncontrol.Job{
		ID:           "job-1",
		AllowedTools: []string{"zeta", "alpha", "beta"},
		Plan: missioncontrol.Plan{
			ID: "plan-1",
			Steps: []missioncontrol.Step{
				{
					ID:           "build",
					Type:         missioncontrol.StepTypeOneShotCode,
					AllowedTools: []string{"zeta", "beta", "beta"},
				},
				{
					ID:        "respond",
					Type:      missioncontrol.StepTypeFinalResponse,
					DependsOn: []string{"build"},
				},
			},
		},
	}
	path := filepath.Join(t.TempDir(), "status.json")

	if err := ag.ActivateMissionStep(job, "build"); err != nil {
		t.Fatalf("ActivateMissionStep() error = %v", err)
	}
	if err := writeMissionStatusSnapshot(path, "mission.json", ag, time.Date(2026, 3, 19, 12, 0, 0, 789, time.UTC)); err != nil {
		t.Fatalf("writeMissionStatusSnapshot() error = %v", err)
	}

	got := readMissionStatusSnapshotFile(t, path)
	want := []string{"beta", "zeta"}
	if !reflect.DeepEqual(got.AllowedTools, want) {
		t.Fatalf("AllowedTools = %v, want %v", got.AllowedTools, want)
	}
}

func TestWriteMissionStatusSnapshotInvalidOutputPathReturnsError(t *testing.T) {
	ag := newMissionBootstrapTestLoop()
	dir := t.TempDir()
	path := filepath.Join(dir, "status-dir")
	if err := os.Mkdir(path, 0o755); err != nil {
		t.Fatalf("Mkdir() error = %v", err)
	}

	err := writeMissionStatusSnapshot(path, "", ag, time.Date(2026, 3, 19, 12, 0, 0, 0, time.UTC))
	if err == nil {
		t.Fatal("writeMissionStatusSnapshot() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "failed to write mission status snapshot") {
		t.Fatalf("writeMissionStatusSnapshot() error = %q, want write failure", err)
	}
	assertNoAtomicTempFiles(t, dir, filepath.Base(path))
}

func TestApplyMissionStepControlFileSwitchesActiveStep(t *testing.T) {
	ag := newMissionBootstrapTestLoop()
	cmd := newMissionBootstrapTestCommand()
	job := testMissionBootstrapJob()
	controlFile := writeMissionStepControlFile(t, missionStepControlFile{StepID: "final", UpdatedAt: "2026-03-19T12:00:00Z"})

	if err := ag.ActivateMissionStep(job, "build"); err != nil {
		t.Fatalf("ActivateMissionStep() error = %v", err)
	}

	stepID, changed, err := applyMissionStepControlFile(cmd, ag, job, controlFile)
	if err != nil {
		t.Fatalf("applyMissionStepControlFile() error = %v", err)
	}
	if !changed {
		t.Fatal("applyMissionStepControlFile() changed = false, want true")
	}
	if stepID != "final" {
		t.Fatalf("applyMissionStepControlFile() stepID = %q, want %q", stepID, "final")
	}

	ec, ok := ag.ActiveMissionStep()
	if !ok || ec.Step == nil {
		t.Fatalf("ActiveMissionStep() = %#v, want active step", ec)
	}
	if ec.Step.ID != "final" {
		t.Fatalf("ActiveMissionStep().Step.ID = %q, want %q", ec.Step.ID, "final")
	}
}

func TestApplyMissionStepControlFileInvalidStepPreservesActiveStep(t *testing.T) {
	ag := newMissionBootstrapTestLoop()
	cmd := newMissionBootstrapTestCommand()
	job := testMissionBootstrapJob()
	controlFile := writeMissionStepControlFile(t, missionStepControlFile{StepID: "missing"})

	if err := ag.ActivateMissionStep(job, "build"); err != nil {
		t.Fatalf("ActivateMissionStep() error = %v", err)
	}

	stepID, changed, err := applyMissionStepControlFile(cmd, ag, job, controlFile)
	if err == nil {
		t.Fatal("applyMissionStepControlFile() error = nil, want invalid step error")
	}
	if changed {
		t.Fatal("applyMissionStepControlFile() changed = true, want false")
	}
	if stepID != "" {
		t.Fatalf("applyMissionStepControlFile() stepID = %q, want empty", stepID)
	}

	ec, ok := ag.ActiveMissionStep()
	if !ok || ec.Step == nil {
		t.Fatalf("ActiveMissionStep() = %#v, want active step", ec)
	}
	if ec.Step.ID != "build" {
		t.Fatalf("ActiveMissionStep().Step.ID = %q, want %q", ec.Step.ID, "build")
	}
}

func TestApplyMissionStepControlFileRewritesStatusSnapshotOnSuccess(t *testing.T) {
	ag := newMissionBootstrapTestLoop()
	cmd := newMissionBootstrapTestCommand()
	job := testMissionBootstrapJob()
	missionFile := writeMissionBootstrapJobFile(t, job)
	statusFile := filepath.Join(t.TempDir(), "status.json")
	controlFile := writeMissionStepControlFile(t, missionStepControlFile{StepID: "final"})

	if err := cmd.Flags().Set("mission-file", missionFile); err != nil {
		t.Fatalf("Flags().Set(mission-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step", "build"); err != nil {
		t.Fatalf("Flags().Set(mission-step) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-status-file", statusFile); err != nil {
		t.Fatalf("Flags().Set(mission-status-file) error = %v", err)
	}

	if err := ag.ActivateMissionStep(job, "build"); err != nil {
		t.Fatalf("ActivateMissionStep() error = %v", err)
	}
	if err := writeMissionStatusSnapshotFromCommand(cmd, ag, time.Date(2026, 3, 19, 12, 0, 0, 0, time.UTC)); err != nil {
		t.Fatalf("writeMissionStatusSnapshotFromCommand() error = %v", err)
	}
	beforeControl := mustReadFile(t, controlFile)

	before := readMissionStatusSnapshotFile(t, statusFile)
	if before.StepID != "build" {
		t.Fatalf("initial snapshot StepID = %q, want %q", before.StepID, "build")
	}

	stepID, changed, err := applyMissionStepControlFile(cmd, ag, job, controlFile)
	if err != nil {
		t.Fatalf("applyMissionStepControlFile() error = %v", err)
	}
	if !changed {
		t.Fatal("applyMissionStepControlFile() changed = false, want true")
	}
	if stepID != "final" {
		t.Fatalf("applyMissionStepControlFile() stepID = %q, want %q", stepID, "final")
	}

	after := readMissionStatusSnapshotFile(t, statusFile)
	if after.StepID != "final" {
		t.Fatalf("rewritten snapshot StepID = %q, want %q", after.StepID, "final")
	}
	if after.UpdatedAt == before.UpdatedAt {
		t.Fatalf("rewritten snapshot UpdatedAt = %q, want changed timestamp", after.UpdatedAt)
	}
	if !bytes.Equal(mustReadFile(t, controlFile), beforeControl) {
		t.Fatalf("mission step control input changed from %q to %q, want unchanged input semantics", string(beforeControl), string(mustReadFile(t, controlFile)))
	}
}

func TestApplyMissionStepControlFileAbsentFileIsNoOp(t *testing.T) {
	ag := newMissionBootstrapTestLoop()
	cmd := newMissionBootstrapTestCommand()
	job := testMissionBootstrapJob()

	if err := ag.ActivateMissionStep(job, "build"); err != nil {
		t.Fatalf("ActivateMissionStep() error = %v", err)
	}

	stepID, changed, err := applyMissionStepControlFile(cmd, ag, job, filepath.Join(t.TempDir(), "missing.json"))
	if err != nil {
		t.Fatalf("applyMissionStepControlFile() error = %v", err)
	}
	if changed {
		t.Fatal("applyMissionStepControlFile() changed = true, want false")
	}
	if stepID != "" {
		t.Fatalf("applyMissionStepControlFile() stepID = %q, want empty", stepID)
	}

	ec, ok := ag.ActiveMissionStep()
	if !ok || ec.Step == nil {
		t.Fatalf("ActiveMissionStep() = %#v, want active step", ec)
	}
	if ec.Step.ID != "build" {
		t.Fatalf("ActiveMissionStep().Step.ID = %q, want %q", ec.Step.ID, "build")
	}
}

func TestRestoreMissionStepControlFileOnStartupAbsentFileIsNoOp(t *testing.T) {
	ag := newMissionBootstrapTestLoop()
	cmd := newMissionBootstrapTestCommand()
	missionFile := writeMissionBootstrapJobFile(t, testMissionBootstrapJob())

	if err := cmd.Flags().Set("mission-file", missionFile); err != nil {
		t.Fatalf("Flags().Set(mission-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step", "build"); err != nil {
		t.Fatalf("Flags().Set(mission-step) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step-control-file", filepath.Join(t.TempDir(), "missing.json")); err != nil {
		t.Fatalf("Flags().Set(mission-step-control-file) error = %v", err)
	}

	job := configureMissionBootstrapJobForStartupTest(t, cmd, ag)
	baseline := restoreMissionStepControlFileOnStartup(cmd, ag, job)
	if baseline != nil {
		t.Fatalf("restoreMissionStepControlFileOnStartup() baseline = %q, want nil", string(baseline))
	}

	ec, ok := ag.ActiveMissionStep()
	if !ok || ec.Step == nil {
		t.Fatalf("ActiveMissionStep() = %#v, want active step", ec)
	}
	if ec.Step.ID != "build" {
		t.Fatalf("ActiveMissionStep().Step.ID = %q, want %q", ec.Step.ID, "build")
	}
}

func TestRestoreMissionStepControlFileOnStartupAbsentFileLeavesWatcherBaselineAsNoOp(t *testing.T) {
	ag := newMissionBootstrapTestLoop()
	cmd := newMissionBootstrapTestCommand()
	job := testMissionBootstrapJob()
	controlFile := filepath.Join(t.TempDir(), "missing.json")
	logBuf, restoreLog := captureStandardLogger(t)
	defer restoreLog()

	if err := ag.ActivateMissionStep(job, "build"); err != nil {
		t.Fatalf("ActivateMissionStep() error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step-control-file", controlFile); err != nil {
		t.Fatalf("Flags().Set(mission-step-control-file) error = %v", err)
	}

	baseline := restoreMissionStepControlFileOnStartup(cmd, ag, job)
	if baseline != nil {
		t.Fatalf("restoreMissionStepControlFileOnStartup() baseline = %q, want nil", string(baseline))
	}

	ctx, cancel := context.WithCancel(context.Background())
	go watchMissionStepControlFile(ctx, cmd, ag, job, controlFile, 5*time.Millisecond, baseline)
	time.Sleep(25 * time.Millisecond)
	cancel()

	ec, ok := ag.ActiveMissionStep()
	if !ok || ec.Step == nil {
		t.Fatalf("ActiveMissionStep() = %#v, want active step", ec)
	}
	if ec.Step.ID != "build" {
		t.Fatalf("ActiveMissionStep().Step.ID = %q, want %q", ec.Step.ID, "build")
	}
	if strings.Contains(logBuf.String(), "mission step control apply succeeded") {
		t.Fatalf("log output = %q, want no watcher apply success", logBuf.String())
	}
}

func TestRestoreMissionStepControlFileOnStartupValidFileOverridesBootstrappedStep(t *testing.T) {
	ag := newMissionBootstrapTestLoop()
	cmd := newMissionBootstrapTestCommand()
	missionFile := writeMissionBootstrapJobFile(t, testMissionBootstrapJob())
	controlFile := writeMissionStepControlFile(t, missionStepControlFile{StepID: "final", UpdatedAt: "2026-03-19T12:00:00Z"})
	logBuf, restoreLog := captureStandardLogger(t)
	defer restoreLog()

	if err := cmd.Flags().Set("mission-file", missionFile); err != nil {
		t.Fatalf("Flags().Set(mission-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step", "build"); err != nil {
		t.Fatalf("Flags().Set(mission-step) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step-control-file", controlFile); err != nil {
		t.Fatalf("Flags().Set(mission-step-control-file) error = %v", err)
	}

	job := configureMissionBootstrapJobForStartupTest(t, cmd, ag)
	baseline := restoreMissionStepControlFileOnStartup(cmd, ag, job)

	ec, ok := ag.ActiveMissionStep()
	if !ok || ec.Step == nil {
		t.Fatalf("ActiveMissionStep() = %#v, want active step", ec)
	}
	if ec.Step.ID != "final" {
		t.Fatalf("ActiveMissionStep().Step.ID = %q, want %q", ec.Step.ID, "final")
	}
	logOutput := logBuf.String()
	if !strings.Contains(logOutput, "mission step control startup apply succeeded") {
		t.Fatalf("log output = %q, want startup apply success", logOutput)
	}
	if !strings.Contains(logOutput, `job_id="job-1"`) {
		t.Fatalf("log output = %q, want job_id", logOutput)
	}
	if !strings.Contains(logOutput, `step_id="final"`) {
		t.Fatalf("log output = %q, want step_id", logOutput)
	}
	if !strings.Contains(logOutput, `control_file="`+controlFile+`"`) {
		t.Fatalf("log output = %q, want control file path", logOutput)
	}
	if !bytes.Equal(baseline, mustReadFile(t, controlFile)) {
		t.Fatalf("restoreMissionStepControlFileOnStartup() baseline = %q, want control file contents", string(baseline))
	}
}

func TestRestoreMissionStepControlFileOnStartupRejectsPreviouslyCompletedStepReplay(t *testing.T) {
	ag := newMissionBootstrapTestLoop()
	cmd := newMissionBootstrapTestCommand()
	job := testMissionBootstrapJob()
	missionFile := writeMissionBootstrapJobFile(t, job)
	controlFile := writeMissionStepControlFile(t, missionStepControlFile{StepID: "build", UpdatedAt: "2026-03-19T12:00:00Z"})
	statusFile := filepath.Join(t.TempDir(), "status.json")
	logBuf, restoreLog := captureStandardLogger(t)
	defer restoreLog()

	writeMissionStatusSnapshotFile(t, statusFile, missionStatusSnapshot{
		Active:         true,
		MissionFile:    missionFile,
		JobID:          job.ID,
		StepID:         "final",
		RuntimeControl: runtimeControlForBootstrapStep(t, job, "final"),
		Runtime: &missioncontrol.JobRuntimeState{
			JobID:        job.ID,
			State:        missioncontrol.JobStatePaused,
			ActiveStepID: "final",
			PausedReason: missioncontrol.RuntimePauseReasonOperatorCommand,
			CompletedSteps: []missioncontrol.RuntimeStepRecord{
				{StepID: "build", At: time.Date(2026, 3, 19, 11, 30, 0, 0, time.UTC)},
			},
		},
		UpdatedAt: "2026-03-19T12:00:00Z",
	})

	if err := cmd.Flags().Set("mission-file", missionFile); err != nil {
		t.Fatalf("Flags().Set(mission-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step", "final"); err != nil {
		t.Fatalf("Flags().Set(mission-step) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-status-file", statusFile); err != nil {
		t.Fatalf("Flags().Set(mission-status-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step-control-file", controlFile); err != nil {
		t.Fatalf("Flags().Set(mission-step-control-file) error = %v", err)
	}

	bootstrappedJob := configureMissionBootstrapJobForStartupTest(t, cmd, ag)
	baseline := restoreMissionStepControlFileOnStartup(cmd, ag, bootstrappedJob)
	if baseline != nil {
		t.Fatalf("restoreMissionStepControlFileOnStartup() baseline = %q, want nil after completed-step replay rejection", string(baseline))
	}

	if _, ok := ag.ActiveMissionStep(); ok {
		t.Fatal("ActiveMissionStep() ok = true, want no live execution context after rehydrated replay rejection")
	}

	runtime, ok := ag.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want persisted runtime state")
	}
	if runtime.ActiveStepID != "final" {
		t.Fatalf("MissionRuntimeState().ActiveStepID = %q, want %q", runtime.ActiveStepID, "final")
	}
	if len(runtime.CompletedSteps) != 1 || runtime.CompletedSteps[0].StepID != "build" {
		t.Fatalf("MissionRuntimeState().CompletedSteps = %#v, want preserved build completion", runtime.CompletedSteps)
	}

	logOutput := logBuf.String()
	if !strings.Contains(logOutput, `step "build" is already recorded as completed in runtime state`) {
		t.Fatalf("log output = %q, want completed-step replay rejection", logOutput)
	}
}

func TestRestoreMissionStepControlFileOnStartupRejectsPreviouslyFailedStepReplay(t *testing.T) {
	ag := newMissionBootstrapTestLoop()
	cmd := newMissionBootstrapTestCommand()
	job := testMissionBootstrapJob()
	missionFile := writeMissionBootstrapJobFile(t, job)
	controlFile := writeMissionStepControlFile(t, missionStepControlFile{StepID: "build", UpdatedAt: "2026-03-19T12:00:00Z"})
	statusFile := filepath.Join(t.TempDir(), "status.json")
	logBuf, restoreLog := captureStandardLogger(t)
	defer restoreLog()

	writeMissionStatusSnapshotFile(t, statusFile, missionStatusSnapshot{
		Active:         true,
		MissionFile:    missionFile,
		JobID:          job.ID,
		StepID:         "final",
		RuntimeControl: runtimeControlForBootstrapStep(t, job, "final"),
		Runtime: &missioncontrol.JobRuntimeState{
			JobID:        job.ID,
			State:        missioncontrol.JobStatePaused,
			ActiveStepID: "final",
			PausedReason: missioncontrol.RuntimePauseReasonOperatorCommand,
			FailedSteps: []missioncontrol.RuntimeStepRecord{
				{StepID: "build", Reason: "validator failed", At: time.Date(2026, 3, 19, 11, 30, 0, 0, time.UTC)},
			},
		},
		UpdatedAt: "2026-03-19T12:00:00Z",
	})

	if err := cmd.Flags().Set("mission-file", missionFile); err != nil {
		t.Fatalf("Flags().Set(mission-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step", "final"); err != nil {
		t.Fatalf("Flags().Set(mission-step) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-status-file", statusFile); err != nil {
		t.Fatalf("Flags().Set(mission-status-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step-control-file", controlFile); err != nil {
		t.Fatalf("Flags().Set(mission-step-control-file) error = %v", err)
	}

	bootstrappedJob := configureMissionBootstrapJobForStartupTest(t, cmd, ag)
	baseline := restoreMissionStepControlFileOnStartup(cmd, ag, bootstrappedJob)
	if baseline != nil {
		t.Fatalf("restoreMissionStepControlFileOnStartup() baseline = %q, want nil after failed-step replay rejection", string(baseline))
	}

	if _, ok := ag.ActiveMissionStep(); ok {
		t.Fatal("ActiveMissionStep() ok = true, want no live execution context after rehydrated replay rejection")
	}

	runtime, ok := ag.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want persisted runtime state")
	}
	if runtime.ActiveStepID != "final" {
		t.Fatalf("MissionRuntimeState().ActiveStepID = %q, want %q", runtime.ActiveStepID, "final")
	}
	if len(runtime.FailedSteps) != 1 || runtime.FailedSteps[0].StepID != "build" {
		t.Fatalf("MissionRuntimeState().FailedSteps = %#v, want preserved build failure", runtime.FailedSteps)
	}

	logOutput := logBuf.String()
	if !strings.Contains(logOutput, `step "build" is already recorded as failed in runtime state`) {
		t.Fatalf("log output = %q, want failed-step replay rejection", logOutput)
	}
}

func TestRestoreMissionStepControlFileOnStartupThenWatcherDoesNotDuplicateUnchangedApply(t *testing.T) {
	ag := newMissionBootstrapTestLoop()
	cmd := newMissionBootstrapTestCommand()
	job := testMissionBootstrapJob()
	controlFile := writeMissionStepControlFile(t, missionStepControlFile{StepID: "final", UpdatedAt: "2026-03-19T12:00:00Z"})
	logBuf, restoreLog := captureStandardLogger(t)
	defer restoreLog()

	if err := ag.ActivateMissionStep(job, "build"); err != nil {
		t.Fatalf("ActivateMissionStep() error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step-control-file", controlFile); err != nil {
		t.Fatalf("Flags().Set(mission-step-control-file) error = %v", err)
	}
	baseline := restoreMissionStepControlFileOnStartup(cmd, ag, job)
	if !bytes.Equal(baseline, mustReadFile(t, controlFile)) {
		t.Fatalf("restoreMissionStepControlFileOnStartup() baseline = %q, want control file contents", string(baseline))
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go watchMissionStepControlFile(ctx, cmd, ag, job, controlFile, 5*time.Millisecond, baseline)

	deadline := time.Now().Add(500 * time.Millisecond)
	for {
		ec, ok := ag.ActiveMissionStep()
		if ok && ec.Step != nil && ec.Step.ID == "final" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("ActiveMissionStep() did not update to final")
		}
		time.Sleep(5 * time.Millisecond)
	}

	time.Sleep(50 * time.Millisecond)
	logOutput := logBuf.String()
	if strings.Count(logOutput, "mission step control startup apply succeeded") != 1 {
		t.Fatalf("log output = %q, want one startup apply success", logOutput)
	}
	if strings.Contains(logOutput, "mission step control apply succeeded") {
		t.Fatalf("log output = %q, want no duplicate watcher apply success", logOutput)
	}
}

func TestWatchMissionStepControlFileRejectsPreviouslyCompletedStepReplay(t *testing.T) {
	ag := newMissionBootstrapTestLoop()
	cmd := newMissionBootstrapTestCommand()
	job := testMissionBootstrapJob()
	missionFile := writeMissionBootstrapJobFile(t, job)
	controlFile := writeMissionStepControlFile(t, missionStepControlFile{StepID: "build", UpdatedAt: "2026-03-19T12:00:00Z"})
	statusFile := filepath.Join(t.TempDir(), "status.json")
	logBuf, restoreLog := captureStandardLogger(t)
	defer restoreLog()

	writeMissionStatusSnapshotFile(t, statusFile, missionStatusSnapshot{
		Active:         true,
		MissionFile:    missionFile,
		JobID:          job.ID,
		StepID:         "final",
		RuntimeControl: runtimeControlForBootstrapStep(t, job, "final"),
		Runtime: &missioncontrol.JobRuntimeState{
			JobID:        job.ID,
			State:        missioncontrol.JobStatePaused,
			ActiveStepID: "final",
			PausedReason: missioncontrol.RuntimePauseReasonOperatorCommand,
			CompletedSteps: []missioncontrol.RuntimeStepRecord{
				{StepID: "build", At: time.Date(2026, 3, 19, 11, 30, 0, 0, time.UTC)},
			},
		},
		UpdatedAt: "2026-03-19T12:00:00Z",
	})

	if err := cmd.Flags().Set("mission-file", missionFile); err != nil {
		t.Fatalf("Flags().Set(mission-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step", "final"); err != nil {
		t.Fatalf("Flags().Set(mission-step) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-status-file", statusFile); err != nil {
		t.Fatalf("Flags().Set(mission-status-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step-control-file", controlFile); err != nil {
		t.Fatalf("Flags().Set(mission-step-control-file) error = %v", err)
	}

	bootstrappedJob := configureMissionBootstrapJobForStartupTest(t, cmd, ag)

	ctx, cancel := context.WithCancel(context.Background())
	go watchMissionStepControlFile(ctx, cmd, ag, bootstrappedJob, controlFile, 5*time.Millisecond, nil)
	time.Sleep(25 * time.Millisecond)
	cancel()

	if _, ok := ag.ActiveMissionStep(); ok {
		t.Fatal("ActiveMissionStep() ok = true, want no live execution context after watcher replay rejection")
	}

	runtime, ok := ag.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want persisted runtime state")
	}
	if runtime.ActiveStepID != "final" {
		t.Fatalf("MissionRuntimeState().ActiveStepID = %q, want %q", runtime.ActiveStepID, "final")
	}
	if len(runtime.CompletedSteps) != 1 || runtime.CompletedSteps[0].StepID != "build" {
		t.Fatalf("MissionRuntimeState().CompletedSteps = %#v, want preserved build completion", runtime.CompletedSteps)
	}

	logOutput := logBuf.String()
	if !strings.Contains(logOutput, `mission step control apply failed`) {
		t.Fatalf("log output = %q, want watcher apply failure", logOutput)
	}
	if !strings.Contains(logOutput, `step "build" is already recorded as completed in runtime state`) {
		t.Fatalf("log output = %q, want completed-step replay rejection", logOutput)
	}
	if strings.Contains(logOutput, `mission step control apply succeeded`) {
		t.Fatalf("log output = %q, want no watcher apply success", logOutput)
	}
}

func TestWatchMissionStepControlFileRejectsPreviouslyFailedStepReplay(t *testing.T) {
	ag := newMissionBootstrapTestLoop()
	cmd := newMissionBootstrapTestCommand()
	job := testMissionBootstrapJob()
	missionFile := writeMissionBootstrapJobFile(t, job)
	controlFile := writeMissionStepControlFile(t, missionStepControlFile{StepID: "build", UpdatedAt: "2026-03-19T12:00:00Z"})
	statusFile := filepath.Join(t.TempDir(), "status.json")
	logBuf, restoreLog := captureStandardLogger(t)
	defer restoreLog()

	writeMissionStatusSnapshotFile(t, statusFile, missionStatusSnapshot{
		Active:         true,
		MissionFile:    missionFile,
		JobID:          job.ID,
		StepID:         "final",
		RuntimeControl: runtimeControlForBootstrapStep(t, job, "final"),
		Runtime: &missioncontrol.JobRuntimeState{
			JobID:        job.ID,
			State:        missioncontrol.JobStatePaused,
			ActiveStepID: "final",
			PausedReason: missioncontrol.RuntimePauseReasonOperatorCommand,
			FailedSteps: []missioncontrol.RuntimeStepRecord{
				{StepID: "build", Reason: "validator failed", At: time.Date(2026, 3, 19, 11, 30, 0, 0, time.UTC)},
			},
		},
		UpdatedAt: "2026-03-19T12:00:00Z",
	})

	if err := cmd.Flags().Set("mission-file", missionFile); err != nil {
		t.Fatalf("Flags().Set(mission-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step", "final"); err != nil {
		t.Fatalf("Flags().Set(mission-step) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-status-file", statusFile); err != nil {
		t.Fatalf("Flags().Set(mission-status-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step-control-file", controlFile); err != nil {
		t.Fatalf("Flags().Set(mission-step-control-file) error = %v", err)
	}

	bootstrappedJob := configureMissionBootstrapJobForStartupTest(t, cmd, ag)

	ctx, cancel := context.WithCancel(context.Background())
	go watchMissionStepControlFile(ctx, cmd, ag, bootstrappedJob, controlFile, 5*time.Millisecond, nil)
	time.Sleep(25 * time.Millisecond)
	cancel()

	if _, ok := ag.ActiveMissionStep(); ok {
		t.Fatal("ActiveMissionStep() ok = true, want no live execution context after watcher replay rejection")
	}

	runtime, ok := ag.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want persisted runtime state")
	}
	if runtime.ActiveStepID != "final" {
		t.Fatalf("MissionRuntimeState().ActiveStepID = %q, want %q", runtime.ActiveStepID, "final")
	}
	if len(runtime.FailedSteps) != 1 || runtime.FailedSteps[0].StepID != "build" {
		t.Fatalf("MissionRuntimeState().FailedSteps = %#v, want preserved build failure", runtime.FailedSteps)
	}

	logOutput := logBuf.String()
	if !strings.Contains(logOutput, `mission step control apply failed`) {
		t.Fatalf("log output = %q, want watcher apply failure", logOutput)
	}
	if !strings.Contains(logOutput, `step "build" is already recorded as failed in runtime state`) {
		t.Fatalf("log output = %q, want failed-step replay rejection", logOutput)
	}
	if strings.Contains(logOutput, `mission step control apply succeeded`) {
		t.Fatalf("log output = %q, want no watcher apply success", logOutput)
	}
}

func TestRestoreMissionStepControlFileOnStartupInvalidFilePreservesBootstrappedStep(t *testing.T) {
	ag := newMissionBootstrapTestLoop()
	cmd := newMissionBootstrapTestCommand()
	missionFile := writeMissionBootstrapJobFile(t, testMissionBootstrapJob())
	controlFile := writeMissionStepControlFile(t, missionStepControlFile{StepID: "missing"})
	logBuf := &bytes.Buffer{}
	logWriter := log.Writer()
	logFlags := log.Flags()
	logPrefix := log.Prefix()
	log.SetOutput(logBuf)
	log.SetFlags(0)
	log.SetPrefix("")
	defer func() {
		log.SetOutput(logWriter)
		log.SetFlags(logFlags)
		log.SetPrefix(logPrefix)
	}()

	if err := cmd.Flags().Set("mission-file", missionFile); err != nil {
		t.Fatalf("Flags().Set(mission-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step", "build"); err != nil {
		t.Fatalf("Flags().Set(mission-step) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step-control-file", controlFile); err != nil {
		t.Fatalf("Flags().Set(mission-step-control-file) error = %v", err)
	}

	job := configureMissionBootstrapJobForStartupTest(t, cmd, ag)
	baseline := restoreMissionStepControlFileOnStartup(cmd, ag, job)

	ec, ok := ag.ActiveMissionStep()
	if !ok || ec.Step == nil {
		t.Fatalf("ActiveMissionStep() = %#v, want active step", ec)
	}
	if ec.Step.ID != "build" {
		t.Fatalf("ActiveMissionStep().Step.ID = %q, want %q", ec.Step.ID, "build")
	}
	if !strings.Contains(logBuf.String(), "mission step control startup apply failed") {
		t.Fatalf("log output = %q, want startup apply failure", logBuf.String())
	}
	if baseline != nil {
		t.Fatalf("restoreMissionStepControlFileOnStartup() baseline = %q, want nil", string(baseline))
	}
}

func TestRestoreMissionStepControlFileOnStartupInitialSnapshotReflectsRestoredStep(t *testing.T) {
	ag := newMissionBootstrapTestLoop()
	cmd := newMissionBootstrapTestCommand()
	job := testMissionBootstrapJob()
	missionFile := writeMissionBootstrapJobFile(t, job)
	statusFile := filepath.Join(t.TempDir(), "status.json")
	controlFile := writeMissionStepControlFile(t, missionStepControlFile{StepID: "final", UpdatedAt: "2026-03-19T12:00:00Z"})

	if err := cmd.Flags().Set("mission-file", missionFile); err != nil {
		t.Fatalf("Flags().Set(mission-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step", "build"); err != nil {
		t.Fatalf("Flags().Set(mission-step) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step-control-file", controlFile); err != nil {
		t.Fatalf("Flags().Set(mission-step-control-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-status-file", statusFile); err != nil {
		t.Fatalf("Flags().Set(mission-status-file) error = %v", err)
	}

	bootstrappedJob := configureMissionBootstrapJobForStartupTest(t, cmd, ag)
	restoreMissionStepControlFileOnStartup(cmd, ag, bootstrappedJob)
	if err := writeMissionStatusSnapshotFromCommand(cmd, ag, time.Date(2026, 3, 19, 12, 0, 0, 0, time.UTC)); err != nil {
		t.Fatalf("writeMissionStatusSnapshotFromCommand() error = %v", err)
	}

	snapshot := readMissionStatusSnapshotFile(t, statusFile)
	if snapshot.StepID != "final" {
		t.Fatalf("initial snapshot StepID = %q, want %q", snapshot.StepID, "final")
	}
	if snapshot.Runtime == nil {
		t.Fatal("initial snapshot Runtime = nil, want non-nil")
	}
	if snapshot.Runtime.ActiveStepID != "final" {
		t.Fatalf("initial snapshot Runtime.ActiveStepID = %q, want %q", snapshot.Runtime.ActiveStepID, "final")
	}
}

func TestMissionStatusBootstrapRequiresResumeApprovalAfterReboot(t *testing.T) {
	ag := newMissionBootstrapTestLoop()
	cmd := newMissionBootstrapTestCommand()
	job := testMissionBootstrapJob()
	missionFile := writeMissionBootstrapJobFile(t, job)
	statusFile := filepath.Join(t.TempDir(), "status.json")
	writeMissionStatusSnapshotFile(t, statusFile, missionStatusSnapshot{
		MissionFile: missionFile,
		JobID:       job.ID,
		StepID:      "build",
		Runtime: &missioncontrol.JobRuntimeState{
			JobID:        job.ID,
			State:        missioncontrol.JobStateRunning,
			ActiveStepID: "build",
		},
	})

	if err := cmd.Flags().Set("mission-file", missionFile); err != nil {
		t.Fatalf("Flags().Set(mission-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step", "build"); err != nil {
		t.Fatalf("Flags().Set(mission-step) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-status-file", statusFile); err != nil {
		t.Fatalf("Flags().Set(mission-status-file) error = %v", err)
	}

	err := configureMissionBootstrap(cmd, ag)
	if err == nil {
		t.Fatal("configureMissionBootstrap() error = nil, want resume approval failure")
	}
	if !strings.Contains(err.Error(), "--mission-resume-approved") {
		t.Fatalf("configureMissionBootstrap() error = %q, want resume approval message", err)
	}
}

func TestMissionStatusBootstrapRejectsInconsistentPersistedRuntimeStepEnvelope(t *testing.T) {
	ag := newMissionBootstrapTestLoop()
	cmd := newMissionBootstrapTestCommand()
	job := testMissionBootstrapJob()
	missionFile := writeMissionBootstrapJobFile(t, job)
	statusFile := filepath.Join(t.TempDir(), "status.json")
	writeMissionStatusSnapshotFile(t, statusFile, missionStatusSnapshot{
		MissionFile: missionFile,
		JobID:       job.ID,
		StepID:      "final",
		Runtime: &missioncontrol.JobRuntimeState{
			JobID:        job.ID,
			State:        missioncontrol.JobStatePaused,
			ActiveStepID: "build",
			PausedReason: missioncontrol.RuntimePauseReasonOperatorCommand,
		},
	})

	if err := cmd.Flags().Set("mission-file", missionFile); err != nil {
		t.Fatalf("Flags().Set(mission-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step", "build"); err != nil {
		t.Fatalf("Flags().Set(mission-step) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-status-file", statusFile); err != nil {
		t.Fatalf("Flags().Set(mission-status-file) error = %v", err)
	}

	err := configureMissionBootstrap(cmd, ag)
	if err == nil {
		t.Fatal("configureMissionBootstrap() error = nil, want persisted-runtime mismatch failure")
	}
	if !strings.Contains(err.Error(), `snapshot step_id "final" does not match runtime active_step_id "build"`) {
		t.Fatalf("configureMissionBootstrap() error = %q, want step envelope mismatch", err)
	}
}

func TestMissionStatusBootstrapRejectsInconsistentPersistedRuntimeControlStep(t *testing.T) {
	ag := newMissionBootstrapTestLoop()
	cmd := newMissionBootstrapTestCommand()
	job := testMissionBootstrapJob()
	missionFile := writeMissionBootstrapJobFile(t, job)
	statusFile := filepath.Join(t.TempDir(), "status.json")
	writeMissionStatusSnapshotFile(t, statusFile, missionStatusSnapshot{
		MissionFile: missionFile,
		JobID:       job.ID,
		StepID:      "build",
		RuntimeControl: &missioncontrol.RuntimeControlContext{
			JobID:        job.ID,
			MaxAuthority: job.MaxAuthority,
			AllowedTools: []string{"read"},
			Step: missioncontrol.Step{
				ID:   "final",
				Type: missioncontrol.StepTypeFinalResponse,
			},
		},
		Runtime: &missioncontrol.JobRuntimeState{
			JobID:        job.ID,
			State:        missioncontrol.JobStatePaused,
			ActiveStepID: "build",
			PausedReason: missioncontrol.RuntimePauseReasonOperatorCommand,
		},
	})

	if err := cmd.Flags().Set("mission-file", missionFile); err != nil {
		t.Fatalf("Flags().Set(mission-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step", "build"); err != nil {
		t.Fatalf("Flags().Set(mission-step) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-status-file", statusFile); err != nil {
		t.Fatalf("Flags().Set(mission-status-file) error = %v", err)
	}

	err := configureMissionBootstrap(cmd, ag)
	if err == nil {
		t.Fatal("configureMissionBootstrap() error = nil, want persisted-control mismatch failure")
	}
	if !strings.Contains(err.Error(), `runtime control step_id "final" does not match runtime active_step_id "build"`) {
		t.Fatalf("configureMissionBootstrap() error = %q, want runtime-control mismatch", err)
	}
}

func TestMissionStatusBootstrapRejectsPersistedRuntimeWithActiveCompletedStepRecord(t *testing.T) {
	ag := newMissionBootstrapTestLoop()
	cmd := newMissionBootstrapTestCommand()
	job := testMissionBootstrapJob()
	missionFile := writeMissionBootstrapJobFile(t, job)
	statusFile := filepath.Join(t.TempDir(), "status.json")
	writeMissionStatusSnapshotFile(t, statusFile, missionStatusSnapshot{
		MissionFile: missionFile,
		JobID:       job.ID,
		StepID:      "build",
		Runtime: &missioncontrol.JobRuntimeState{
			JobID:        job.ID,
			State:        missioncontrol.JobStatePaused,
			ActiveStepID: "build",
			PausedReason: missioncontrol.RuntimePauseReasonOperatorCommand,
			CompletedSteps: []missioncontrol.RuntimeStepRecord{
				{StepID: "build", At: time.Date(2026, 3, 27, 10, 0, 0, 0, time.UTC)},
			},
		},
	})

	if err := cmd.Flags().Set("mission-file", missionFile); err != nil {
		t.Fatalf("Flags().Set(mission-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step", "build"); err != nil {
		t.Fatalf("Flags().Set(mission-step) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-status-file", statusFile); err != nil {
		t.Fatalf("Flags().Set(mission-status-file) error = %v", err)
	}

	err := configureMissionBootstrap(cmd, ag)
	if err == nil {
		t.Fatal("configureMissionBootstrap() error = nil, want completed-step replay marker failure")
	}
	if !strings.Contains(err.Error(), `active_step_id "build" is already recorded in completed_steps`) {
		t.Fatalf("configureMissionBootstrap() error = %q, want completed-step replay marker mismatch", err)
	}
}

func TestMissionStatusBootstrapRejectsPersistedRuntimeWithActiveFailedStepRecord(t *testing.T) {
	ag := newMissionBootstrapTestLoop()
	cmd := newMissionBootstrapTestCommand()
	job := testMissionBootstrapJob()
	missionFile := writeMissionBootstrapJobFile(t, job)
	statusFile := filepath.Join(t.TempDir(), "status.json")
	writeMissionStatusSnapshotFile(t, statusFile, missionStatusSnapshot{
		MissionFile: missionFile,
		JobID:       job.ID,
		StepID:      "build",
		Runtime: &missioncontrol.JobRuntimeState{
			JobID:        job.ID,
			State:        missioncontrol.JobStatePaused,
			ActiveStepID: "build",
			PausedReason: missioncontrol.RuntimePauseReasonOperatorCommand,
			FailedSteps: []missioncontrol.RuntimeStepRecord{
				{StepID: "build", Reason: "validator failed", At: time.Date(2026, 3, 27, 10, 0, 0, 0, time.UTC)},
			},
		},
	})

	if err := cmd.Flags().Set("mission-file", missionFile); err != nil {
		t.Fatalf("Flags().Set(mission-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step", "build"); err != nil {
		t.Fatalf("Flags().Set(mission-step) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-status-file", statusFile); err != nil {
		t.Fatalf("Flags().Set(mission-status-file) error = %v", err)
	}

	err := configureMissionBootstrap(cmd, ag)
	if err == nil {
		t.Fatal("configureMissionBootstrap() error = nil, want failed-step replay marker failure")
	}
	if !strings.Contains(err.Error(), `active_step_id "build" is already recorded in failed_steps`) {
		t.Fatalf("configureMissionBootstrap() error = %q, want failed-step replay marker mismatch", err)
	}
}

func TestMissionStatusBootstrapApprovedResumeUsesPersistedRuntimeStep(t *testing.T) {
	ag := newMissionBootstrapTestLoop()
	cmd := newMissionBootstrapTestCommand()
	job := testMissionBootstrapJob()
	missionFile := writeMissionBootstrapJobFile(t, job)
	statusFile := filepath.Join(t.TempDir(), "status.json")
	writeMissionStatusSnapshotFile(t, statusFile, missionStatusSnapshot{
		MissionFile: missionFile,
		JobID:       job.ID,
		StepID:      "build",
		Runtime: &missioncontrol.JobRuntimeState{
			JobID:          job.ID,
			State:          missioncontrol.JobStateWaitingUser,
			ActiveStepID:   "build",
			WaitingReason:  "awaiting operator confirmation",
			CompletedSteps: []missioncontrol.RuntimeStepRecord{{StepID: "draft", At: time.Date(2026, 3, 23, 10, 0, 0, 0, time.UTC)}},
		},
	})

	if err := cmd.Flags().Set("mission-file", missionFile); err != nil {
		t.Fatalf("Flags().Set(mission-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step", "build"); err != nil {
		t.Fatalf("Flags().Set(mission-step) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-status-file", statusFile); err != nil {
		t.Fatalf("Flags().Set(mission-status-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-resume-approved", "true"); err != nil {
		t.Fatalf("Flags().Set(mission-resume-approved) error = %v", err)
	}

	if err := configureMissionBootstrap(cmd, ag); err != nil {
		t.Fatalf("configureMissionBootstrap() error = %v", err)
	}

	ec, ok := ag.ActiveMissionStep()
	if !ok {
		t.Fatal("ActiveMissionStep() ok = false, want true")
	}
	if ec.Runtime == nil {
		t.Fatal("ActiveMissionStep().Runtime = nil, want non-nil")
	}
	if ec.Runtime.State != missioncontrol.JobStateRunning {
		t.Fatalf("ActiveMissionStep().Runtime.State = %q, want %q", ec.Runtime.State, missioncontrol.JobStateRunning)
	}
	if len(ec.Runtime.CompletedSteps) != 1 || ec.Runtime.CompletedSteps[0].StepID != "draft" {
		t.Fatalf("ActiveMissionStep().Runtime.CompletedSteps = %#v, want preserved draft completion", ec.Runtime.CompletedSteps)
	}
}

func TestMissionStatusBootstrapApprovedResumeUsesPersistedRuntimeControlWhenMissionFileChanges(t *testing.T) {
	ag := newMissionBootstrapTestLoop()
	cmd := newMissionBootstrapTestCommand()
	job := testMissionBootstrapJob()
	missionFile := writeMissionBootstrapJobFile(t, job)
	statusFile := filepath.Join(t.TempDir(), "status.json")
	writeMissionStatusSnapshotFile(t, statusFile, missionStatusSnapshot{
		MissionFile:    missionFile,
		JobID:          job.ID,
		StepID:         "build",
		RuntimeControl: runtimeControlForBootstrapStep(t, job, "build"),
		Runtime: &missioncontrol.JobRuntimeState{
			JobID:        job.ID,
			State:        missioncontrol.JobStatePaused,
			ActiveStepID: "build",
			PausedReason: missioncontrol.RuntimePauseReasonOperatorCommand,
		},
	})
	writeMissionBootstrapJobFile(t, missioncontrol.Job{
		ID:           job.ID,
		MaxAuthority: missioncontrol.AuthorityTierLow,
		AllowedTools: []string{"shell"},
		Plan: missioncontrol.Plan{
			ID: job.Plan.ID,
			Steps: []missioncontrol.Step{
				{
					ID:   "final",
					Type: missioncontrol.StepTypeFinalResponse,
				},
			},
		},
	}, missionFile)

	if err := cmd.Flags().Set("mission-file", missionFile); err != nil {
		t.Fatalf("Flags().Set(mission-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step", "build"); err != nil {
		t.Fatalf("Flags().Set(mission-step) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-status-file", statusFile); err != nil {
		t.Fatalf("Flags().Set(mission-status-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-resume-approved", "true"); err != nil {
		t.Fatalf("Flags().Set(mission-resume-approved) error = %v", err)
	}

	if err := configureMissionBootstrap(cmd, ag); err != nil {
		t.Fatalf("configureMissionBootstrap() error = %v", err)
	}

	ec, ok := ag.ActiveMissionStep()
	if !ok {
		t.Fatal("ActiveMissionStep() ok = false, want true")
	}
	if ec.Step == nil {
		t.Fatal("ActiveMissionStep().Step = nil, want non-nil")
	}
	if ec.Step.Type != missioncontrol.StepTypeOneShotCode {
		t.Fatalf("ActiveMissionStep().Step.Type = %q, want persisted %q", ec.Step.Type, missioncontrol.StepTypeOneShotCode)
	}
	if ec.Step.RequiredAuthority != missioncontrol.AuthorityTierLow {
		t.Fatalf("ActiveMissionStep().Step.RequiredAuthority = %q, want persisted %q", ec.Step.RequiredAuthority, missioncontrol.AuthorityTierLow)
	}
	if !reflect.DeepEqual(ec.Step.AllowedTools, []string{"read"}) {
		t.Fatalf("ActiveMissionStep().Step.AllowedTools = %#v, want persisted %#v", ec.Step.AllowedTools, []string{"read"})
	}
}

func TestMissionStatusBootstrapRehydratesPausedRuntimeControlAfterReboot(t *testing.T) {
	statusFile := filepath.Join(t.TempDir(), "status.json")
	cmd := newMissionBootstrapTestCommand()
	if err := cmd.Flags().Set("mission-status-file", statusFile); err != nil {
		t.Fatalf("Flags().Set(mission-status-file) error = %v", err)
	}

	job := testMissionBootstrapJob()
	missionFile := writeMissionBootstrapJobFile(t, job)
	writeMissionStatusSnapshotFile(t, statusFile, missionStatusSnapshot{
		MissionFile: missionFile,
		JobID:       job.ID,
		StepID:      "build",
		Runtime: &missioncontrol.JobRuntimeState{
			JobID:        job.ID,
			State:        missioncontrol.JobStatePaused,
			ActiveStepID: "build",
			PausedReason: missioncontrol.RuntimePauseReasonOperatorCommand,
		},
	})
	if err := cmd.Flags().Set("mission-file", missionFile); err != nil {
		t.Fatalf("Flags().Set(mission-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step", "build"); err != nil {
		t.Fatalf("Flags().Set(mission-step) error = %v", err)
	}

	hub := chat.NewHub(10)
	provider := &missionStatusFixedResponseProvider{content: "unused"}
	ag := agent.NewAgentLoop(hub, provider, provider.GetDefaultModel(), 3, "", nil)
	installMissionRuntimeChangeHook(cmd, ag)

	if err := configureMissionBootstrap(cmd, ag); err != nil {
		t.Fatalf("configureMissionBootstrap() error = %v", err)
	}
	if _, ok := ag.ActiveMissionStep(); ok {
		t.Fatal("ActiveMissionStep() ok = true, want false after control rehydration")
	}

	runtime, ok := ag.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if runtime.State != missioncontrol.JobStatePaused {
		t.Fatalf("MissionRuntimeState().State = %q, want %q", runtime.State, missioncontrol.JobStatePaused)
	}

	if _, err := ag.ProcessDirect("RESUME job-1", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(RESUME) error = %v", err)
	}

	ec, ok := ag.ActiveMissionStep()
	if !ok || ec.Runtime == nil {
		t.Fatalf("ActiveMissionStep() = %#v, want running runtime", ec)
	}
	if ec.Runtime.State != missioncontrol.JobStateRunning {
		t.Fatalf("ActiveMissionStep().Runtime.State = %q, want %q", ec.Runtime.State, missioncontrol.JobStateRunning)
	}

	snapshot := readMissionStatusSnapshotFile(t, statusFile)
	if snapshot.Runtime == nil || snapshot.Runtime.State != missioncontrol.JobStateRunning {
		t.Fatalf("snapshot.Runtime = %#v, want running runtime", snapshot.Runtime)
	}
	if !snapshot.Active {
		t.Fatal("snapshot.Active = false, want true after resume")
	}
}

func TestMissionStatusBootstrapRehydratesPausedRuntimeControlUsesFallbackWithoutPersistedControl(t *testing.T) {
	statusFile := filepath.Join(t.TempDir(), "status.json")
	cmd := newMissionBootstrapTestCommand()
	if err := cmd.Flags().Set("mission-status-file", statusFile); err != nil {
		t.Fatalf("Flags().Set(mission-status-file) error = %v", err)
	}

	job := testMissionBootstrapJob()
	missionFile := writeMissionBootstrapJobFile(t, job)
	writeMissionStatusSnapshotFile(t, statusFile, missionStatusSnapshot{
		MissionFile: missionFile,
		JobID:       job.ID,
		StepID:      "build",
		Runtime: &missioncontrol.JobRuntimeState{
			JobID:        job.ID,
			State:        missioncontrol.JobStatePaused,
			ActiveStepID: "build",
			PausedReason: missioncontrol.RuntimePauseReasonOperatorCommand,
		},
	})
	if err := cmd.Flags().Set("mission-file", missionFile); err != nil {
		t.Fatalf("Flags().Set(mission-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step", "build"); err != nil {
		t.Fatalf("Flags().Set(mission-step) error = %v", err)
	}

	ag := newMissionBootstrapTestLoop()
	if err := configureMissionBootstrap(cmd, ag); err != nil {
		t.Fatalf("configureMissionBootstrap() error = %v", err)
	}
	if _, ok := ag.ActiveMissionStep(); ok {
		t.Fatal("ActiveMissionStep() ok = true, want false after control rehydration")
	}

	if _, err := ag.ProcessDirect("RESUME job-1", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(RESUME) error = %v", err)
	}

	ec, ok := ag.ActiveMissionStep()
	if !ok || ec.Step == nil {
		t.Fatalf("ActiveMissionStep() = %#v, want resumed active step", ec)
	}
	if ec.Step.Type != missioncontrol.StepTypeOneShotCode {
		t.Fatalf("ActiveMissionStep().Step.Type = %q, want %q", ec.Step.Type, missioncontrol.StepTypeOneShotCode)
	}
}

func TestMissionStatusBootstrapRehydratedApproveFromWaitingUserUsesPersistedRuntimeControlWhenMissionFileChanges(t *testing.T) {
	statusFile := filepath.Join(t.TempDir(), "status.json")
	cmd := newMissionBootstrapTestCommand()
	if err := cmd.Flags().Set("mission-status-file", statusFile); err != nil {
		t.Fatalf("Flags().Set(mission-status-file) error = %v", err)
	}

	job := missioncontrol.Job{
		ID:           "job-1",
		MaxAuthority: missioncontrol.AuthorityTierHigh,
		AllowedTools: []string{"read"},
		Plan: missioncontrol.Plan{
			ID: "plan-1",
			Steps: []missioncontrol.Step{
				{
					ID:      "build",
					Type:    missioncontrol.StepTypeDiscussion,
					Subtype: missioncontrol.StepSubtypeAuthorization,
				},
				{
					ID:        "final",
					Type:      missioncontrol.StepTypeFinalResponse,
					DependsOn: []string{"build"},
				},
			},
		},
	}
	missionFile := writeMissionBootstrapJobFile(t, job)
	writeMissionStatusSnapshotFile(t, statusFile, missionStatusSnapshot{
		MissionFile:    missionFile,
		JobID:          job.ID,
		StepID:         "build",
		RuntimeControl: runtimeControlForBootstrapStep(t, job, "build"),
		Runtime: &missioncontrol.JobRuntimeState{
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
		},
	})
	writeMissionBootstrapJobFile(t, missioncontrol.Job{
		ID:           job.ID,
		MaxAuthority: missioncontrol.AuthorityTierLow,
		AllowedTools: []string{"shell"},
		Plan: missioncontrol.Plan{
			ID: job.Plan.ID,
			Steps: []missioncontrol.Step{
				{
					ID:   "final",
					Type: missioncontrol.StepTypeFinalResponse,
				},
			},
		},
	}, missionFile)

	if err := cmd.Flags().Set("mission-file", missionFile); err != nil {
		t.Fatalf("Flags().Set(mission-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step", "build"); err != nil {
		t.Fatalf("Flags().Set(mission-step) error = %v", err)
	}

	ag := newMissionBootstrapTestLoop()
	if err := configureMissionBootstrap(cmd, ag); err != nil {
		t.Fatalf("configureMissionBootstrap() error = %v", err)
	}
	if _, ok := ag.ActiveMissionStep(); ok {
		t.Fatal("ActiveMissionStep() ok = true, want false after waiting_user rehydration")
	}

	if _, err := ag.ProcessDirect("APPROVE job-1 build", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(APPROVE) error = %v", err)
	}

	runtime, ok := ag.MissionRuntimeState()
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

func TestMissionStatusBootstrapRehydratedApproveUsesFallbackWithoutPersistedControl(t *testing.T) {
	statusFile := filepath.Join(t.TempDir(), "status.json")
	cmd := newMissionBootstrapTestCommand()
	if err := cmd.Flags().Set("mission-status-file", statusFile); err != nil {
		t.Fatalf("Flags().Set(mission-status-file) error = %v", err)
	}

	job := missioncontrol.Job{
		ID:           "job-1",
		MaxAuthority: missioncontrol.AuthorityTierHigh,
		AllowedTools: []string{"read"},
		Plan: missioncontrol.Plan{
			ID: "plan-1",
			Steps: []missioncontrol.Step{
				{
					ID:      "build",
					Type:    missioncontrol.StepTypeDiscussion,
					Subtype: missioncontrol.StepSubtypeAuthorization,
				},
				{
					ID:        "final",
					Type:      missioncontrol.StepTypeFinalResponse,
					DependsOn: []string{"build"},
				},
			},
		},
	}
	missionFile := writeMissionBootstrapJobFile(t, job)
	writeMissionStatusSnapshotFile(t, statusFile, missionStatusSnapshot{
		MissionFile: missionFile,
		JobID:       job.ID,
		StepID:      "build",
		Runtime: &missioncontrol.JobRuntimeState{
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
		},
	})

	if err := cmd.Flags().Set("mission-file", missionFile); err != nil {
		t.Fatalf("Flags().Set(mission-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step", "build"); err != nil {
		t.Fatalf("Flags().Set(mission-step) error = %v", err)
	}

	ag := newMissionBootstrapTestLoop()
	if err := configureMissionBootstrap(cmd, ag); err != nil {
		t.Fatalf("configureMissionBootstrap() error = %v", err)
	}

	if _, err := ag.ProcessDirect("APPROVE job-1 build", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(APPROVE) error = %v", err)
	}

	runtime, ok := ag.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if runtime.State != missioncontrol.JobStatePaused {
		t.Fatalf("MissionRuntimeState().State = %q, want %q", runtime.State, missioncontrol.JobStatePaused)
	}
}

func TestMissionStatusBootstrapRehydratedDenyBlocksLaterFreeFormCompletion(t *testing.T) {
	statusFile := filepath.Join(t.TempDir(), "status.json")
	cmd := newMissionBootstrapTestCommand()
	if err := cmd.Flags().Set("mission-status-file", statusFile); err != nil {
		t.Fatalf("Flags().Set(mission-status-file) error = %v", err)
	}

	job := missioncontrol.Job{
		ID:           "job-1",
		MaxAuthority: missioncontrol.AuthorityTierHigh,
		AllowedTools: []string{"read"},
		Plan: missioncontrol.Plan{
			ID: "plan-1",
			Steps: []missioncontrol.Step{
				{
					ID:      "build",
					Type:    missioncontrol.StepTypeDiscussion,
					Subtype: missioncontrol.StepSubtypeAuthorization,
				},
				{
					ID:        "final",
					Type:      missioncontrol.StepTypeFinalResponse,
					DependsOn: []string{"build"},
				},
			},
		},
	}
	missionFile := writeMissionBootstrapJobFile(t, job)
	writeMissionStatusSnapshotFile(t, statusFile, missionStatusSnapshot{
		MissionFile: missionFile,
		JobID:       job.ID,
		StepID:      "build",
		Runtime: &missioncontrol.JobRuntimeState{
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
		},
	})
	if err := cmd.Flags().Set("mission-file", missionFile); err != nil {
		t.Fatalf("Flags().Set(mission-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step", "build"); err != nil {
		t.Fatalf("Flags().Set(mission-step) error = %v", err)
	}

	ag := newMissionBootstrapTestLoop()
	if err := configureMissionBootstrap(cmd, ag); err != nil {
		t.Fatalf("configureMissionBootstrap() error = %v", err)
	}

	if _, err := ag.ProcessDirect("DENY job-1 build", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(DENY) error = %v", err)
	}
	resp, err := ag.ProcessDirect("approved", 2*time.Second)
	if err != nil {
		t.Fatalf("ProcessDirect(approved) error = %v", err)
	}
	if !strings.Contains(resp, "(stub) Echo") {
		t.Fatalf("ProcessDirect(approved) response = %q, want stub provider fallback after denied reboot-safe path", resp)
	}

	runtime, ok := ag.MissionRuntimeState()
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

func TestMissionStatusBootstrapRehydratedWrongJobDoesNotBind(t *testing.T) {
	statusFile := filepath.Join(t.TempDir(), "status.json")
	cmd := newMissionBootstrapTestCommand()
	if err := cmd.Flags().Set("mission-status-file", statusFile); err != nil {
		t.Fatalf("Flags().Set(mission-status-file) error = %v", err)
	}

	job := testMissionBootstrapJob()
	missionFile := writeMissionBootstrapJobFile(t, job)
	writeMissionStatusSnapshotFile(t, statusFile, missionStatusSnapshot{
		MissionFile: missionFile,
		JobID:       job.ID,
		StepID:      "build",
		Runtime: &missioncontrol.JobRuntimeState{
			JobID:        job.ID,
			State:        missioncontrol.JobStatePaused,
			ActiveStepID: "build",
			PausedReason: missioncontrol.RuntimePauseReasonOperatorCommand,
		},
	})
	if err := cmd.Flags().Set("mission-file", missionFile); err != nil {
		t.Fatalf("Flags().Set(mission-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step", "build"); err != nil {
		t.Fatalf("Flags().Set(mission-step) error = %v", err)
	}

	ag := newMissionBootstrapTestLoop()
	if err := configureMissionBootstrap(cmd, ag); err != nil {
		t.Fatalf("configureMissionBootstrap() error = %v", err)
	}

	_, err := ag.ProcessDirect("RESUME other-job", 2*time.Second)
	if err == nil {
		t.Fatal("ProcessDirect(RESUME wrong job) error = nil, want mismatch failure")
	}
	if !strings.Contains(err.Error(), "does not match the active job") {
		t.Fatalf("ProcessDirect(RESUME wrong job) error = %q, want job mismatch", err)
	}
	if _, ok := ag.ActiveMissionStep(); ok {
		t.Fatal("ActiveMissionStep() ok = true, want false after wrong-job rejection")
	}
}

func TestMissionStatusBootstrapNormalizesLegacyRevokedApprovalRequestAndPersistsSnapshot(t *testing.T) {
	statusFile := filepath.Join(t.TempDir(), "status.json")
	cmd := newMissionBootstrapTestCommand()
	if err := cmd.Flags().Set("mission-status-file", statusFile); err != nil {
		t.Fatalf("Flags().Set(mission-status-file) error = %v", err)
	}

	job := missioncontrol.Job{
		ID:           "job-1",
		MaxAuthority: missioncontrol.AuthorityTierHigh,
		AllowedTools: []string{"read"},
		Plan: missioncontrol.Plan{
			ID: "plan-1",
			Steps: []missioncontrol.Step{
				{
					ID:                "build",
					Type:              missioncontrol.StepTypeDiscussion,
					Subtype:           missioncontrol.StepSubtypeAuthorization,
					ApprovalScope:     missioncontrol.ApprovalScopeOneJob,
					AllowedTools:      []string{"read"},
					RequiredAuthority: missioncontrol.AuthorityTierLow,
				},
				{
					ID:        "final",
					Type:      missioncontrol.StepTypeFinalResponse,
					DependsOn: []string{"build"},
				},
			},
		},
	}
	missionFile := writeMissionBootstrapJobFile(t, job)
	revokedAt := time.Date(2026, 3, 24, 12, 1, 0, 0, time.UTC)
	writeMissionStatusSnapshotFile(t, statusFile, missionStatusSnapshot{
		MissionFile:    missionFile,
		JobID:          job.ID,
		StepID:         "build",
		RuntimeControl: runtimeControlForBootstrapStep(t, job, "build"),
		Runtime: &missioncontrol.JobRuntimeState{
			JobID:        job.ID,
			State:        missioncontrol.JobStatePaused,
			ActiveStepID: "build",
			PausedReason: missioncontrol.RuntimePauseReasonOperatorCommand,
			ApprovalRequests: []missioncontrol.ApprovalRequest{
				{
					JobID:           job.ID,
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
					JobID:           job.ID,
					StepID:          "build",
					RequestedAction: missioncontrol.ApprovalRequestedActionStepComplete,
					Scope:           missioncontrol.ApprovalScopeOneJob,
					GrantedVia:      missioncontrol.ApprovalGrantedViaOperatorCommand,
					State:           missioncontrol.ApprovalStateRevoked,
					RevokedAt:       revokedAt,
				},
			},
		},
	})
	if err := cmd.Flags().Set("mission-file", missionFile); err != nil {
		t.Fatalf("Flags().Set(mission-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step", "build"); err != nil {
		t.Fatalf("Flags().Set(mission-step) error = %v", err)
	}

	ag := newMissionBootstrapTestLoop()
	installMissionRuntimeChangeHook(cmd, ag)
	if err := configureMissionBootstrap(cmd, ag); err != nil {
		t.Fatalf("configureMissionBootstrap() error = %v", err)
	}

	runtime, ok := ag.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if runtime.ApprovalRequests[0].RevokedAt != revokedAt {
		t.Fatalf("MissionRuntimeState().ApprovalRequests[0].RevokedAt = %v, want %v", runtime.ApprovalRequests[0].RevokedAt, revokedAt)
	}

	snapshot := readMissionStatusSnapshotFile(t, statusFile)
	if snapshot.Runtime == nil {
		t.Fatal("snapshot.Runtime = nil, want persisted runtime")
	}
	if snapshot.Runtime.ApprovalRequests[0].RevokedAt != revokedAt {
		t.Fatalf("snapshot.Runtime.ApprovalRequests[0].RevokedAt = %v, want %v", snapshot.Runtime.ApprovalRequests[0].RevokedAt, revokedAt)
	}
}

func TestMissionOperatorSetStepCommandActiveJobSucceedsThroughConfirmationPath(t *testing.T) {
	ag := newMissionBootstrapTestLoop()
	cmd := newMissionBootstrapTestCommand()
	job := testMissionBootstrapJob()
	missionFile := writeMissionBootstrapJobFile(t, job)
	statusFile := filepath.Join(t.TempDir(), "status.json")
	controlFile := filepath.Join(t.TempDir(), "control.json")

	if err := cmd.Flags().Set("mission-file", missionFile); err != nil {
		t.Fatalf("Flags().Set(mission-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step", "build"); err != nil {
		t.Fatalf("Flags().Set(mission-step) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-status-file", statusFile); err != nil {
		t.Fatalf("Flags().Set(mission-status-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step-control-file", controlFile); err != nil {
		t.Fatalf("Flags().Set(mission-step-control-file) error = %v", err)
	}

	bootstrappedJob := configureMissionBootstrapJobForStartupTest(t, cmd, ag)
	installMissionOperatorSetStepHook(cmd, ag, &bootstrappedJob, true)
	if err := writeMissionStatusSnapshotFromCommand(cmd, ag, time.Now()); err != nil {
		t.Fatalf("writeMissionStatusSnapshotFromCommand() error = %v", err)
	}

	resp, err := ag.ProcessDirect("SET_STEP job-1 final", 2*time.Second)
	if err != nil {
		t.Fatalf("ProcessDirect(SET_STEP) error = %v", err)
	}
	if resp != "Set step job=job-1 step=final." {
		t.Fatalf("ProcessDirect(SET_STEP) response = %q, want set-step acknowledgement", resp)
	}

	ec, ok := ag.ActiveMissionStep()
	if !ok || ec.Step == nil {
		t.Fatalf("ActiveMissionStep() = %#v, want active final step", ec)
	}
	if ec.Step.ID != "final" {
		t.Fatalf("ActiveMissionStep().Step.ID = %q, want %q", ec.Step.ID, "final")
	}

	control := readMissionStepControlFile(t, controlFile)
	if control.StepID != "final" {
		t.Fatalf("control.StepID = %q, want %q", control.StepID, "final")
	}

	snapshot := readMissionStatusSnapshotFile(t, statusFile)
	if snapshot.StepID != "final" {
		t.Fatalf("snapshot.StepID = %q, want %q", snapshot.StepID, "final")
	}
	if snapshot.JobID != job.ID {
		t.Fatalf("snapshot.JobID = %q, want %q", snapshot.JobID, job.ID)
	}
}

func TestMissionOperatorSetStepCommandWrongJobDoesNotBind(t *testing.T) {
	ag := newMissionBootstrapTestLoop()
	cmd := newMissionBootstrapTestCommand()
	job := testMissionBootstrapJob()
	missionFile := writeMissionBootstrapJobFile(t, job)
	statusFile := filepath.Join(t.TempDir(), "status.json")
	controlFile := filepath.Join(t.TempDir(), "control.json")

	if err := cmd.Flags().Set("mission-file", missionFile); err != nil {
		t.Fatalf("Flags().Set(mission-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step", "build"); err != nil {
		t.Fatalf("Flags().Set(mission-step) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-status-file", statusFile); err != nil {
		t.Fatalf("Flags().Set(mission-status-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step-control-file", controlFile); err != nil {
		t.Fatalf("Flags().Set(mission-step-control-file) error = %v", err)
	}

	bootstrappedJob := configureMissionBootstrapJobForStartupTest(t, cmd, ag)
	installMissionOperatorSetStepHook(cmd, ag, &bootstrappedJob, true)

	_, err := ag.ProcessDirect("SET_STEP other-job final", 2*time.Second)
	if err == nil {
		t.Fatal("ProcessDirect(SET_STEP wrong job) error = nil, want mismatch failure")
	}
	if !strings.Contains(err.Error(), "does not match the active job") {
		t.Fatalf("ProcessDirect(SET_STEP wrong job) error = %q, want job mismatch", err)
	}
	if _, statErr := os.Stat(controlFile); !os.IsNotExist(statErr) {
		t.Fatalf("Stat(controlFile) error = %v, want not exists", statErr)
	}

	ec, ok := ag.ActiveMissionStep()
	if !ok || ec.Step == nil {
		t.Fatalf("ActiveMissionStep() = %#v, want unchanged active step", ec)
	}
	if ec.Step.ID != "build" {
		t.Fatalf("ActiveMissionStep().Step.ID = %q, want %q", ec.Step.ID, "build")
	}
}

func TestMissionOperatorSetStepCommandInvalidStepRejectsDeterministically(t *testing.T) {
	ag := newMissionBootstrapTestLoop()
	cmd := newMissionBootstrapTestCommand()
	job := testMissionBootstrapJob()
	missionFile := writeMissionBootstrapJobFile(t, job)
	statusFile := filepath.Join(t.TempDir(), "status.json")
	controlFile := filepath.Join(t.TempDir(), "control.json")

	if err := cmd.Flags().Set("mission-file", missionFile); err != nil {
		t.Fatalf("Flags().Set(mission-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step", "build"); err != nil {
		t.Fatalf("Flags().Set(mission-step) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-status-file", statusFile); err != nil {
		t.Fatalf("Flags().Set(mission-status-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step-control-file", controlFile); err != nil {
		t.Fatalf("Flags().Set(mission-step-control-file) error = %v", err)
	}

	bootstrappedJob := configureMissionBootstrapJobForStartupTest(t, cmd, ag)
	installMissionOperatorSetStepHook(cmd, ag, &bootstrappedJob, true)

	_, err := ag.ProcessDirect("SET_STEP job-1 missing", 2*time.Second)
	if err == nil {
		t.Fatal("ProcessDirect(SET_STEP missing step) error = nil, want validation failure")
	}
	if !strings.Contains(err.Error(), `step "missing" not found in plan`) {
		t.Fatalf("ProcessDirect(SET_STEP missing step) error = %q, want unknown-step rejection", err)
	}
	if _, statErr := os.Stat(controlFile); !os.IsNotExist(statErr) {
		t.Fatalf("Stat(controlFile) error = %v, want not exists", statErr)
	}
}

func TestMissionOperatorSetStepCommandStaleMatchingStatusSnapshotDoesNotConfirmSuccess(t *testing.T) {
	ag := newMissionBootstrapTestLoop()
	cmd := newMissionBootstrapTestCommand()
	job := testMissionBootstrapJob()
	missionFile := writeMissionBootstrapJobFile(t, job)
	statusFile := filepath.Join(t.TempDir(), "status.json")
	controlFile := filepath.Join(t.TempDir(), "control.json")

	if err := cmd.Flags().Set("mission-file", missionFile); err != nil {
		t.Fatalf("Flags().Set(mission-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step", "build"); err != nil {
		t.Fatalf("Flags().Set(mission-step) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-status-file", statusFile); err != nil {
		t.Fatalf("Flags().Set(mission-status-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step-control-file", controlFile); err != nil {
		t.Fatalf("Flags().Set(mission-step-control-file) error = %v", err)
	}

	bootstrappedJob := configureMissionBootstrapJobForStartupTest(t, cmd, ag)
	ag.SetOperatorSetStepHook(newMissionOperatorSetStepHook(cmd, ag, &bootstrappedJob, false, 150*time.Millisecond))
	writeMissionStatusSnapshotFile(t, statusFile, missionStatusSnapshot{
		Active:       true,
		MissionFile:  missionFile,
		JobID:        job.ID,
		StepID:       "final",
		StepType:     string(missioncontrol.StepTypeFinalResponse),
		AllowedTools: []string{"read"},
		UpdatedAt:    "2026-03-19T12:00:00Z",
	})

	_, err := ag.ProcessDirect("SET_STEP job-1 final", 2*time.Second)
	if err == nil {
		t.Fatal("ProcessDirect(SET_STEP stale status) error = nil, want confirmation timeout")
	}
	if !strings.Contains(err.Error(), "want a fresh matching update") {
		t.Fatalf("ProcessDirect(SET_STEP stale status) error = %q, want stale snapshot rejection", err)
	}
}

func TestMissionOperatorSetStepCommandFreshStatusWithWrongStepTypeDoesNotConfirmSuccess(t *testing.T) {
	ag := newMissionBootstrapTestLoop()
	cmd := newMissionBootstrapTestCommand()
	job := testMissionBootstrapJob()
	missionFile := writeMissionBootstrapJobFile(t, job)
	statusFile := filepath.Join(t.TempDir(), "status.json")
	controlFile := filepath.Join(t.TempDir(), "control.json")

	if err := cmd.Flags().Set("mission-file", missionFile); err != nil {
		t.Fatalf("Flags().Set(mission-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step", "build"); err != nil {
		t.Fatalf("Flags().Set(mission-step) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-status-file", statusFile); err != nil {
		t.Fatalf("Flags().Set(mission-status-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step-control-file", controlFile); err != nil {
		t.Fatalf("Flags().Set(mission-step-control-file) error = %v", err)
	}

	bootstrappedJob := configureMissionBootstrapJobForStartupTest(t, cmd, ag)
	ag.SetOperatorSetStepHook(newMissionOperatorSetStepHook(cmd, ag, &bootstrappedJob, false, 150*time.Millisecond))
	writeMissionStatusSnapshotFile(t, statusFile, missionStatusSnapshot{
		Active:       true,
		MissionFile:  missionFile,
		JobID:        job.ID,
		StepID:       "build",
		StepType:     string(missioncontrol.StepTypeOneShotCode),
		AllowedTools: []string{"read"},
		UpdatedAt:    "2026-03-19T12:00:00Z",
	})

	go func() {
		time.Sleep(50 * time.Millisecond)
		writeMissionStatusSnapshotFile(t, statusFile, missionStatusSnapshot{
			Active:       true,
			MissionFile:  missionFile,
			JobID:        job.ID,
			StepID:       "final",
			StepType:     string(missioncontrol.StepTypeDiscussion),
			AllowedTools: []string{"read"},
			UpdatedAt:    "2026-03-19T12:00:01Z",
		})
	}()

	_, err := ag.ProcessDirect("SET_STEP job-1 final", 2*time.Second)
	if err == nil {
		t.Fatal("ProcessDirect(SET_STEP wrong step type) error = nil, want confirmation failure")
	}
	if !strings.Contains(err.Error(), `has step_type="discussion", want step_type="final_response"`) {
		t.Fatalf("ProcessDirect(SET_STEP wrong step type) error = %q, want step_type mismatch", err)
	}
}

func TestMissionOperatorSetStepCommandFreshStatusWithWrongAllowedToolsDoesNotConfirmSuccess(t *testing.T) {
	ag := newMissionBootstrapTestLoop()
	cmd := newMissionBootstrapTestCommand()
	job := testMissionBootstrapJob()
	missionFile := writeMissionBootstrapJobFile(t, job)
	statusFile := filepath.Join(t.TempDir(), "status.json")
	controlFile := filepath.Join(t.TempDir(), "control.json")

	if err := cmd.Flags().Set("mission-file", missionFile); err != nil {
		t.Fatalf("Flags().Set(mission-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step", "build"); err != nil {
		t.Fatalf("Flags().Set(mission-step) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-status-file", statusFile); err != nil {
		t.Fatalf("Flags().Set(mission-status-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step-control-file", controlFile); err != nil {
		t.Fatalf("Flags().Set(mission-step-control-file) error = %v", err)
	}

	bootstrappedJob := configureMissionBootstrapJobForStartupTest(t, cmd, ag)
	ag.SetOperatorSetStepHook(newMissionOperatorSetStepHook(cmd, ag, &bootstrappedJob, false, 150*time.Millisecond))
	writeMissionStatusSnapshotFile(t, statusFile, missionStatusSnapshot{
		Active:       true,
		MissionFile:  missionFile,
		JobID:        job.ID,
		StepID:       "build",
		StepType:     string(missioncontrol.StepTypeOneShotCode),
		AllowedTools: []string{"read"},
		UpdatedAt:    "2026-03-19T12:00:00Z",
	})

	go func() {
		time.Sleep(50 * time.Millisecond)
		writeMissionStatusSnapshotFile(t, statusFile, missionStatusSnapshot{
			Active:       true,
			MissionFile:  missionFile,
			JobID:        job.ID,
			StepID:       "final",
			StepType:     string(missioncontrol.StepTypeFinalResponse),
			AllowedTools: []string{},
			UpdatedAt:    "2026-03-19T12:00:01Z",
		})
	}()

	_, err := ag.ProcessDirect("SET_STEP job-1 final", 2*time.Second)
	if err == nil {
		t.Fatal("ProcessDirect(SET_STEP wrong allowed tools) error = nil, want confirmation failure")
	}
	if !strings.Contains(err.Error(), `has allowed_tools=[], want allowed_tools=["read"]`) {
		t.Fatalf("ProcessDirect(SET_STEP wrong allowed tools) error = %q, want allowed_tools mismatch", err)
	}
}

func TestMissionOperatorSetStepCommandFreshMatchingStatusSnapshotConfirmsSuccess(t *testing.T) {
	ag := newMissionBootstrapTestLoop()
	cmd := newMissionBootstrapTestCommand()
	job := testMissionBootstrapJob()
	missionFile := writeMissionBootstrapJobFile(t, job)
	statusFile := filepath.Join(t.TempDir(), "status.json")
	controlFile := filepath.Join(t.TempDir(), "control.json")

	if err := cmd.Flags().Set("mission-file", missionFile); err != nil {
		t.Fatalf("Flags().Set(mission-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step", "build"); err != nil {
		t.Fatalf("Flags().Set(mission-step) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-status-file", statusFile); err != nil {
		t.Fatalf("Flags().Set(mission-status-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step-control-file", controlFile); err != nil {
		t.Fatalf("Flags().Set(mission-step-control-file) error = %v", err)
	}

	bootstrappedJob := configureMissionBootstrapJobForStartupTest(t, cmd, ag)
	ag.SetOperatorSetStepHook(newMissionOperatorSetStepHook(cmd, ag, &bootstrappedJob, false, 500*time.Millisecond))
	writeMissionStatusSnapshotFile(t, statusFile, missionStatusSnapshot{
		Active:       true,
		MissionFile:  missionFile,
		JobID:        job.ID,
		StepID:       "build",
		StepType:     string(missioncontrol.StepTypeOneShotCode),
		AllowedTools: []string{"read"},
		UpdatedAt:    "2026-03-19T12:00:00Z",
	})

	go func() {
		time.Sleep(50 * time.Millisecond)
		writeMissionStatusSnapshotFile(t, statusFile, missionStatusSnapshot{
			Active:       true,
			MissionFile:  missionFile,
			JobID:        job.ID,
			StepID:       "final",
			StepType:     string(missioncontrol.StepTypeFinalResponse),
			AllowedTools: []string{"read"},
			UpdatedAt:    "2026-03-19T12:00:01Z",
		})
	}()

	resp, err := ag.ProcessDirect("SET_STEP job-1 final", 2*time.Second)
	if err != nil {
		t.Fatalf("ProcessDirect(SET_STEP fresh status) error = %v", err)
	}
	if resp != "Set step job=job-1 step=final." {
		t.Fatalf("ProcessDirect(SET_STEP fresh status) response = %q, want set-step acknowledgement", resp)
	}
}

func TestMissionOperatorSetStepCommandRehydratedRuntimeSucceedsWhenAppropriate(t *testing.T) {
	ag := newMissionBootstrapTestLoop()
	cmd := newMissionBootstrapTestCommand()
	job := testMissionBootstrapJob()
	missionFile := writeMissionBootstrapJobFile(t, job)
	statusFile := filepath.Join(t.TempDir(), "status.json")
	controlFile := filepath.Join(t.TempDir(), "control.json")
	runtimeControl := runtimeControlForBootstrapStep(t, job, "build")

	writeMissionStatusSnapshotFile(t, statusFile, missionStatusSnapshot{
		Active:         true,
		MissionFile:    missionFile,
		JobID:          job.ID,
		StepID:         "build",
		RuntimeControl: runtimeControl,
		Runtime: &missioncontrol.JobRuntimeState{
			JobID:        job.ID,
			State:        missioncontrol.JobStatePaused,
			ActiveStepID: "build",
			PausedReason: missioncontrol.RuntimePauseReasonOperatorCommand,
		},
		UpdatedAt: "2026-03-19T12:00:00Z",
	})

	if err := cmd.Flags().Set("mission-file", missionFile); err != nil {
		t.Fatalf("Flags().Set(mission-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step", "build"); err != nil {
		t.Fatalf("Flags().Set(mission-step) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-status-file", statusFile); err != nil {
		t.Fatalf("Flags().Set(mission-status-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step-control-file", controlFile); err != nil {
		t.Fatalf("Flags().Set(mission-step-control-file) error = %v", err)
	}

	bootstrappedJob := configureMissionBootstrapJobForStartupTest(t, cmd, ag)
	installMissionOperatorSetStepHook(cmd, ag, &bootstrappedJob, true)

	resp, err := ag.ProcessDirect("SET_STEP job-1 final", 2*time.Second)
	if err != nil {
		t.Fatalf("ProcessDirect(SET_STEP rehydrated runtime) error = %v", err)
	}
	if resp != "Set step job=job-1 step=final." {
		t.Fatalf("ProcessDirect(SET_STEP rehydrated runtime) response = %q, want set-step acknowledgement", resp)
	}

	ec, ok := ag.ActiveMissionStep()
	if !ok || ec.Step == nil {
		t.Fatalf("ActiveMissionStep() = %#v, want active final step", ec)
	}
	if ec.Step.ID != "final" {
		t.Fatalf("ActiveMissionStep().Step.ID = %q, want %q", ec.Step.ID, "final")
	}
}

func TestMissionOperatorSetStepCommandRehydratedRuntimeRejectsPreviouslyCompletedStepReplay(t *testing.T) {
	ag := newMissionBootstrapTestLoop()
	cmd := newMissionBootstrapTestCommand()
	job := testMissionBootstrapJob()
	missionFile := writeMissionBootstrapJobFile(t, job)
	statusFile := filepath.Join(t.TempDir(), "status.json")
	controlFile := filepath.Join(t.TempDir(), "control.json")
	runtimeControl := runtimeControlForBootstrapStep(t, job, "final")

	writeMissionStatusSnapshotFile(t, statusFile, missionStatusSnapshot{
		Active:         true,
		MissionFile:    missionFile,
		JobID:          job.ID,
		StepID:         "final",
		RuntimeControl: runtimeControl,
		Runtime: &missioncontrol.JobRuntimeState{
			JobID:        job.ID,
			State:        missioncontrol.JobStatePaused,
			ActiveStepID: "final",
			PausedReason: missioncontrol.RuntimePauseReasonOperatorCommand,
			CompletedSteps: []missioncontrol.RuntimeStepRecord{
				{StepID: "build", At: time.Date(2026, 3, 19, 11, 30, 0, 0, time.UTC)},
			},
		},
		UpdatedAt: "2026-03-19T12:00:00Z",
	})

	if err := cmd.Flags().Set("mission-file", missionFile); err != nil {
		t.Fatalf("Flags().Set(mission-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step", "final"); err != nil {
		t.Fatalf("Flags().Set(mission-step) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-status-file", statusFile); err != nil {
		t.Fatalf("Flags().Set(mission-status-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step-control-file", controlFile); err != nil {
		t.Fatalf("Flags().Set(mission-step-control-file) error = %v", err)
	}

	bootstrappedJob := configureMissionBootstrapJobForStartupTest(t, cmd, ag)
	installMissionOperatorSetStepHook(cmd, ag, &bootstrappedJob, true)

	_, err := ag.ProcessDirect("SET_STEP job-1 build", 2*time.Second)
	if err == nil {
		t.Fatal("ProcessDirect(SET_STEP completed step) error = nil, want replay rejection")
	}
	if !strings.Contains(err.Error(), `step "build" is already recorded as completed in runtime state`) {
		t.Fatalf("ProcessDirect(SET_STEP completed step) error = %q, want completed-step replay rejection", err)
	}

	if _, ok := ag.ActiveMissionStep(); ok {
		t.Fatal("ActiveMissionStep() ok = true, want rehydrated control context without live execution context")
	}
	runtime, ok := ag.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want persisted runtime state")
	}
	if runtime.ActiveStepID != "final" {
		t.Fatalf("MissionRuntimeState().ActiveStepID = %q, want %q", runtime.ActiveStepID, "final")
	}
	if len(runtime.CompletedSteps) != 1 || runtime.CompletedSteps[0].StepID != "build" {
		t.Fatalf("MissionRuntimeState().CompletedSteps = %#v, want preserved build completion", runtime.CompletedSteps)
	}
}

func TestMissionOperatorSetStepCommandRehydratedRuntimeRejectsPreviouslyFailedStepReplay(t *testing.T) {
	ag := newMissionBootstrapTestLoop()
	cmd := newMissionBootstrapTestCommand()
	job := testMissionBootstrapJob()
	missionFile := writeMissionBootstrapJobFile(t, job)
	statusFile := filepath.Join(t.TempDir(), "status.json")
	controlFile := filepath.Join(t.TempDir(), "control.json")
	runtimeControl := runtimeControlForBootstrapStep(t, job, "final")

	writeMissionStatusSnapshotFile(t, statusFile, missionStatusSnapshot{
		Active:         true,
		MissionFile:    missionFile,
		JobID:          job.ID,
		StepID:         "final",
		RuntimeControl: runtimeControl,
		Runtime: &missioncontrol.JobRuntimeState{
			JobID:        job.ID,
			State:        missioncontrol.JobStatePaused,
			ActiveStepID: "final",
			PausedReason: missioncontrol.RuntimePauseReasonOperatorCommand,
			FailedSteps: []missioncontrol.RuntimeStepRecord{
				{StepID: "build", Reason: "validator failed", At: time.Date(2026, 3, 19, 11, 30, 0, 0, time.UTC)},
			},
		},
		UpdatedAt: "2026-03-19T12:00:00Z",
	})

	if err := cmd.Flags().Set("mission-file", missionFile); err != nil {
		t.Fatalf("Flags().Set(mission-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step", "final"); err != nil {
		t.Fatalf("Flags().Set(mission-step) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-status-file", statusFile); err != nil {
		t.Fatalf("Flags().Set(mission-status-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step-control-file", controlFile); err != nil {
		t.Fatalf("Flags().Set(mission-step-control-file) error = %v", err)
	}

	bootstrappedJob := configureMissionBootstrapJobForStartupTest(t, cmd, ag)
	installMissionOperatorSetStepHook(cmd, ag, &bootstrappedJob, true)

	_, err := ag.ProcessDirect("SET_STEP job-1 build", 2*time.Second)
	if err == nil {
		t.Fatal("ProcessDirect(SET_STEP failed step) error = nil, want replay rejection")
	}
	if !strings.Contains(err.Error(), `step "build" is already recorded as failed in runtime state`) {
		t.Fatalf("ProcessDirect(SET_STEP failed step) error = %q, want failed-step replay rejection", err)
	}

	if _, ok := ag.ActiveMissionStep(); ok {
		t.Fatal("ActiveMissionStep() ok = true, want rehydrated control context without live execution context")
	}
	runtime, ok := ag.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want persisted runtime state")
	}
	if runtime.ActiveStepID != "final" {
		t.Fatalf("MissionRuntimeState().ActiveStepID = %q, want %q", runtime.ActiveStepID, "final")
	}
	if len(runtime.FailedSteps) != 1 || runtime.FailedSteps[0].StepID != "build" {
		t.Fatalf("MissionRuntimeState().FailedSteps = %#v, want preserved build failure", runtime.FailedSteps)
	}
}

func TestMissionStatusBootstrapRehydratedWrongStepDoesNotBind(t *testing.T) {
	statusFile := filepath.Join(t.TempDir(), "status.json")
	cmd := newMissionBootstrapTestCommand()
	if err := cmd.Flags().Set("mission-status-file", statusFile); err != nil {
		t.Fatalf("Flags().Set(mission-status-file) error = %v", err)
	}

	job := missioncontrol.Job{
		ID:           "job-1",
		MaxAuthority: missioncontrol.AuthorityTierHigh,
		AllowedTools: []string{"read"},
		Plan: missioncontrol.Plan{
			ID: "plan-1",
			Steps: []missioncontrol.Step{
				{
					ID:      "build",
					Type:    missioncontrol.StepTypeDiscussion,
					Subtype: missioncontrol.StepSubtypeAuthorization,
				},
				{
					ID:        "final",
					Type:      missioncontrol.StepTypeFinalResponse,
					DependsOn: []string{"build"},
				},
			},
		},
	}
	missionFile := writeMissionBootstrapJobFile(t, job)
	writeMissionStatusSnapshotFile(t, statusFile, missionStatusSnapshot{
		MissionFile: missionFile,
		JobID:       job.ID,
		StepID:      "build",
		Runtime: &missioncontrol.JobRuntimeState{
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
		},
	})
	if err := cmd.Flags().Set("mission-file", missionFile); err != nil {
		t.Fatalf("Flags().Set(mission-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step", "build"); err != nil {
		t.Fatalf("Flags().Set(mission-step) error = %v", err)
	}

	ag := newMissionBootstrapTestLoop()
	if err := configureMissionBootstrap(cmd, ag); err != nil {
		t.Fatalf("configureMissionBootstrap() error = %v", err)
	}

	_, err := ag.ProcessDirect("APPROVE job-1 other-step", 2*time.Second)
	if err == nil {
		t.Fatal("ProcessDirect(APPROVE wrong step) error = nil, want mismatch failure")
	}
	if !strings.Contains(err.Error(), "does not match the active job and step") {
		t.Fatalf("ProcessDirect(APPROVE wrong step) error = %q, want step mismatch", err)
	}

	runtime, ok := ag.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if runtime.State != missioncontrol.JobStateWaitingUser {
		t.Fatalf("MissionRuntimeState().State = %q, want %q", runtime.State, missioncontrol.JobStateWaitingUser)
	}
	if len(runtime.ApprovalRequests) != 1 || runtime.ApprovalRequests[0].State != missioncontrol.ApprovalStatePending {
		t.Fatalf("MissionRuntimeState().ApprovalRequests = %#v, want one pending approval", runtime.ApprovalRequests)
	}
}

func TestMissionStatusBootstrapRehydratedAbortUsesPersistedRuntimeControlWhenMissionFileChanges(t *testing.T) {
	statusFile := filepath.Join(t.TempDir(), "status.json")
	cmd := newMissionBootstrapTestCommand()
	if err := cmd.Flags().Set("mission-status-file", statusFile); err != nil {
		t.Fatalf("Flags().Set(mission-status-file) error = %v", err)
	}

	job := missioncontrol.Job{
		ID:           "job-1",
		MaxAuthority: missioncontrol.AuthorityTierHigh,
		AllowedTools: []string{"read"},
		Plan: missioncontrol.Plan{
			ID: "plan-1",
			Steps: []missioncontrol.Step{
				{
					ID:      "build",
					Type:    missioncontrol.StepTypeDiscussion,
					Subtype: missioncontrol.StepSubtypeAuthorization,
				},
				{
					ID:        "final",
					Type:      missioncontrol.StepTypeFinalResponse,
					DependsOn: []string{"build"},
				},
			},
		},
	}
	missionFile := writeMissionBootstrapJobFile(t, job)
	writeMissionStatusSnapshotFile(t, statusFile, missionStatusSnapshot{
		MissionFile:    missionFile,
		JobID:          job.ID,
		StepID:         "build",
		RuntimeControl: runtimeControlForBootstrapStep(t, job, "build"),
		Runtime: &missioncontrol.JobRuntimeState{
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
		},
	})
	writeMissionBootstrapJobFile(t, missioncontrol.Job{
		ID:           job.ID,
		MaxAuthority: missioncontrol.AuthorityTierLow,
		AllowedTools: []string{"shell"},
		Plan: missioncontrol.Plan{
			ID: job.Plan.ID,
			Steps: []missioncontrol.Step{
				{
					ID:   "final",
					Type: missioncontrol.StepTypeFinalResponse,
				},
			},
		},
	}, missionFile)

	if err := cmd.Flags().Set("mission-file", missionFile); err != nil {
		t.Fatalf("Flags().Set(mission-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step", "build"); err != nil {
		t.Fatalf("Flags().Set(mission-step) error = %v", err)
	}

	ag := newMissionBootstrapTestLoop()
	if err := configureMissionBootstrap(cmd, ag); err != nil {
		t.Fatalf("configureMissionBootstrap() error = %v", err)
	}
	if _, ok := ag.ActiveMissionStep(); ok {
		t.Fatal("ActiveMissionStep() ok = true, want false after waiting_user rehydration")
	}

	if _, err := ag.ProcessDirect("ABORT job-1", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(ABORT) error = %v", err)
	}

	runtime, ok := ag.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if runtime.State != missioncontrol.JobStateAborted {
		t.Fatalf("MissionRuntimeState().State = %q, want %q", runtime.State, missioncontrol.JobStateAborted)
	}
}

func TestMissionStatusBootstrapRehydratedAbortFromWaitingUserPersistsLifecycle(t *testing.T) {
	statusFile := filepath.Join(t.TempDir(), "status.json")
	cmd := newMissionBootstrapTestCommand()
	if err := cmd.Flags().Set("mission-status-file", statusFile); err != nil {
		t.Fatalf("Flags().Set(mission-status-file) error = %v", err)
	}

	job := missioncontrol.Job{
		ID:           "job-1",
		MaxAuthority: missioncontrol.AuthorityTierHigh,
		AllowedTools: []string{"read"},
		Plan: missioncontrol.Plan{
			ID: "plan-1",
			Steps: []missioncontrol.Step{
				{
					ID:      "build",
					Type:    missioncontrol.StepTypeDiscussion,
					Subtype: missioncontrol.StepSubtypeAuthorization,
				},
				{
					ID:        "final",
					Type:      missioncontrol.StepTypeFinalResponse,
					DependsOn: []string{"build"},
				},
			},
		},
	}
	missionFile := writeMissionBootstrapJobFile(t, job)
	writeMissionStatusSnapshotFile(t, statusFile, missionStatusSnapshot{
		MissionFile: missionFile,
		JobID:       job.ID,
		StepID:      "build",
		Runtime: &missioncontrol.JobRuntimeState{
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
		},
	})
	if err := cmd.Flags().Set("mission-file", missionFile); err != nil {
		t.Fatalf("Flags().Set(mission-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step", "build"); err != nil {
		t.Fatalf("Flags().Set(mission-step) error = %v", err)
	}

	hub := chat.NewHub(10)
	provider := &missionStatusFixedResponseProvider{content: "unused"}
	ag := agent.NewAgentLoop(hub, provider, provider.GetDefaultModel(), 3, "", nil)
	installMissionRuntimeChangeHook(cmd, ag)

	if err := configureMissionBootstrap(cmd, ag); err != nil {
		t.Fatalf("configureMissionBootstrap() error = %v", err)
	}
	if _, ok := ag.ActiveMissionStep(); ok {
		t.Fatal("ActiveMissionStep() ok = true, want false after waiting_user rehydration")
	}

	if _, err := ag.ProcessDirect("ABORT job-1", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(ABORT) error = %v", err)
	}

	snapshot := readMissionStatusSnapshotFile(t, statusFile)
	if snapshot.Runtime == nil || snapshot.Runtime.State != missioncontrol.JobStateAborted {
		t.Fatalf("snapshot.Runtime = %#v, want aborted runtime", snapshot.Runtime)
	}
	if snapshot.Active {
		t.Fatal("snapshot.Active = true, want false after abort")
	}
}

func TestMissionStatusBootstrapRehydratedTerminalRuntimeRejectsOperatorControl(t *testing.T) {
	for _, runtimeState := range []missioncontrol.JobState{
		missioncontrol.JobStateCompleted,
		missioncontrol.JobStateFailed,
		missioncontrol.JobStateAborted,
	} {
		runtimeState := runtimeState
		t.Run(string(runtimeState), func(t *testing.T) {
			ag := newMissionBootstrapTestLoop()
			cmd := newMissionBootstrapTestCommand()
			job := testMissionBootstrapJob()
			missionFile := writeMissionBootstrapJobFile(t, job)
			statusFile := filepath.Join(t.TempDir(), "status.json")
			writeMissionStatusSnapshotFile(t, statusFile, missionStatusSnapshot{
				MissionFile: missionFile,
				JobID:       job.ID,
				Runtime: &missioncontrol.JobRuntimeState{
					JobID: job.ID,
					State: runtimeState,
				},
			})

			if err := cmd.Flags().Set("mission-file", missionFile); err != nil {
				t.Fatalf("Flags().Set(mission-file) error = %v", err)
			}
			if err := cmd.Flags().Set("mission-step", "build"); err != nil {
				t.Fatalf("Flags().Set(mission-step) error = %v", err)
			}
			if err := cmd.Flags().Set("mission-status-file", statusFile); err != nil {
				t.Fatalf("Flags().Set(mission-status-file) error = %v", err)
			}

			if err := configureMissionBootstrap(cmd, ag); err != nil {
				t.Fatalf("configureMissionBootstrap() error = %v", err)
			}
			if _, ok := ag.ActiveMissionStep(); ok {
				t.Fatal("ActiveMissionStep() ok = true, want false for terminal rehydration")
			}

			for _, command := range []string{"RESUME job-1", "ABORT job-1"} {
				_, err := ag.ProcessDirect(command, 2*time.Second)
				if err == nil {
					t.Fatalf("ProcessDirect(%s) error = nil, want invalid runtime state", command)
				}
				wantCode := "E_STEP_OUT_OF_ORDER"
				if runtimeState == missioncontrol.JobStateAborted {
					wantCode = "E_ABORTED"
				}
				if !strings.Contains(err.Error(), wantCode) {
					t.Fatalf("ProcessDirect(%s) error = %q, want canonical rejection code %q", command, err, wantCode)
				}
			}
		})
	}
}

func TestMissionStatusBootstrapRehydratedTerminalRuntimeRejectsApprovalDecisions(t *testing.T) {
	for _, runtimeState := range []missioncontrol.JobState{
		missioncontrol.JobStateCompleted,
		missioncontrol.JobStateFailed,
		missioncontrol.JobStateAborted,
	} {
		runtimeState := runtimeState
		t.Run(string(runtimeState), func(t *testing.T) {
			ag := newMissionBootstrapTestLoop()
			cmd := newMissionBootstrapTestCommand()
			job := testMissionBootstrapJob()
			missionFile := writeMissionBootstrapJobFile(t, job)
			statusFile := filepath.Join(t.TempDir(), "status.json")
			writeMissionStatusSnapshotFile(t, statusFile, missionStatusSnapshot{
				MissionFile: missionFile,
				JobID:       job.ID,
				Runtime: &missioncontrol.JobRuntimeState{
					JobID: job.ID,
					State: runtimeState,
				},
			})

			if err := cmd.Flags().Set("mission-file", missionFile); err != nil {
				t.Fatalf("Flags().Set(mission-file) error = %v", err)
			}
			if err := cmd.Flags().Set("mission-step", "build"); err != nil {
				t.Fatalf("Flags().Set(mission-step) error = %v", err)
			}
			if err := cmd.Flags().Set("mission-status-file", statusFile); err != nil {
				t.Fatalf("Flags().Set(mission-status-file) error = %v", err)
			}

			if err := configureMissionBootstrap(cmd, ag); err != nil {
				t.Fatalf("configureMissionBootstrap() error = %v", err)
			}

			_, err := ag.ProcessDirect("APPROVE job-1 build", 2*time.Second)
			if err == nil {
				t.Fatal("ProcessDirect(APPROVE) error = nil, want terminal-state rejection")
			}
			if !strings.Contains(err.Error(), string(runtimeState)) {
				t.Fatalf("ProcessDirect(APPROVE) error = %q, want state-specific rejection", err)
			}
		})
	}
}

func TestMissionStatusBootstrapRehydratedTerminalRuntimeRejectsNaturalApprovalDecisions(t *testing.T) {
	for _, runtimeState := range []missioncontrol.JobState{
		missioncontrol.JobStateCompleted,
		missioncontrol.JobStateFailed,
		missioncontrol.JobStateAborted,
	} {
		runtimeState := runtimeState
		t.Run(string(runtimeState), func(t *testing.T) {
			ag := newMissionBootstrapTestLoop()
			cmd := newMissionBootstrapTestCommand()
			job := testMissionBootstrapJob()
			missionFile := writeMissionBootstrapJobFile(t, job)
			statusFile := filepath.Join(t.TempDir(), "status.json")
			writeMissionStatusSnapshotFile(t, statusFile, missionStatusSnapshot{
				MissionFile: missionFile,
				JobID:       job.ID,
				Runtime: &missioncontrol.JobRuntimeState{
					JobID: job.ID,
					State: runtimeState,
				},
			})

			if err := cmd.Flags().Set("mission-file", missionFile); err != nil {
				t.Fatalf("Flags().Set(mission-file) error = %v", err)
			}
			if err := cmd.Flags().Set("mission-step", "build"); err != nil {
				t.Fatalf("Flags().Set(mission-step) error = %v", err)
			}
			if err := cmd.Flags().Set("mission-status-file", statusFile); err != nil {
				t.Fatalf("Flags().Set(mission-status-file) error = %v", err)
			}

			if err := configureMissionBootstrap(cmd, ag); err != nil {
				t.Fatalf("configureMissionBootstrap() error = %v", err)
			}

			_, err := ag.ProcessDirect("yes", 2*time.Second)
			if err == nil {
				t.Fatal("ProcessDirect(yes) error = nil, want terminal-state rejection")
			}
			if !strings.Contains(err.Error(), string(runtimeState)) {
				t.Fatalf("ProcessDirect(yes) error = %q, want state-specific rejection", err)
			}
		})
	}
}

func TestWatchMissionStepControlFileChangedFileAppliesOnceAfterStartup(t *testing.T) {
	ag := newMissionBootstrapTestLoop()
	cmd := newMissionBootstrapTestCommand()
	job := testMissionBootstrapJob()
	controlFile := writeMissionStepControlFile(t, missionStepControlFile{StepID: "build", UpdatedAt: "2026-03-19T12:00:00Z"})
	logBuf, restoreLog := captureStandardLogger(t)
	defer restoreLog()

	if err := ag.ActivateMissionStep(job, "build"); err != nil {
		t.Fatalf("ActivateMissionStep() error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step-control-file", controlFile); err != nil {
		t.Fatalf("Flags().Set(mission-step-control-file) error = %v", err)
	}
	baseline := restoreMissionStepControlFileOnStartup(cmd, ag, job)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go watchMissionStepControlFile(ctx, cmd, ag, job, controlFile, 5*time.Millisecond, baseline)

	time.Sleep(25 * time.Millisecond)
	writeMissionStepControlFile(t, missionStepControlFile{StepID: "final", UpdatedAt: "2026-03-19T12:00:01Z"}, controlFile)

	deadline := time.Now().Add(500 * time.Millisecond)
	for {
		ec, ok := ag.ActiveMissionStep()
		if ok && ec.Step != nil && ec.Step.ID == "final" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("ActiveMissionStep() did not update to final")
		}
		time.Sleep(5 * time.Millisecond)
	}

	deadline = time.Now().Add(500 * time.Millisecond)
	for {
		logOutput := logBuf.String()
		if strings.Count(logOutput, "mission step control apply succeeded") == 1 {
			if !strings.Contains(logOutput, `job_id="`+job.ID+`"`) {
				t.Fatalf("log output = %q, want job_id", logOutput)
			}
			if !strings.Contains(logOutput, `step_id="final"`) {
				t.Fatalf("log output = %q, want step_id", logOutput)
			}
			if !strings.Contains(logOutput, `control_file="`+controlFile+`"`) {
				t.Fatalf("log output = %q, want control file path", logOutput)
			}
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("log output = %q, want one watcher apply success", logOutput)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func TestRemoveMissionStatusSnapshotRemovesFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "status.json")
	if err := os.WriteFile(path, []byte("{}\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if err := removeMissionStatusSnapshot(path); err != nil {
		t.Fatalf("removeMissionStatusSnapshot() error = %v", err)
	}

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("Stat() error = %v, want not exists", err)
	}
}

func newMissionBootstrapTestCommand() *cobra.Command {
	cmd := &cobra.Command{Use: "test"}
	addMissionBootstrapFlags(cmd)
	cmd.Flags().String("mission-step-control-file", "", "")
	return cmd
}

func TestResolveMissionStoreRootPrefersExplicitFlag(t *testing.T) {
	cmd := newMissionBootstrapTestCommand()
	if err := cmd.Flags().Set("mission-store-root", "/tmp/store-root"); err != nil {
		t.Fatalf("Flags().Set(mission-store-root) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-status-file", "/tmp/status.json"); err != nil {
		t.Fatalf("Flags().Set(mission-status-file) error = %v", err)
	}

	got := resolveMissionStoreRoot(cmd)
	if got != "/tmp/store-root" {
		t.Fatalf("resolveMissionStoreRoot() = %q, want %q", got, "/tmp/store-root")
	}
	if got != missioncontrol.ResolveStoreRoot("/tmp/store-root", "/tmp/status.json") {
		t.Fatalf("resolveMissionStoreRoot() = %q, want missioncontrol.ResolveStoreRoot parity", got)
	}
}

func TestResolveMissionStoreRootFallsBackToStatusFile(t *testing.T) {
	cmd := newMissionBootstrapTestCommand()
	if err := cmd.Flags().Set("mission-status-file", "/tmp/status.json"); err != nil {
		t.Fatalf("Flags().Set(mission-status-file) error = %v", err)
	}

	got := resolveMissionStoreRoot(cmd)
	if got != "/tmp/status.json.store" {
		t.Fatalf("resolveMissionStoreRoot() = %q, want %q", got, "/tmp/status.json.store")
	}
	if got != missioncontrol.ResolveStoreRoot("", "/tmp/status.json") {
		t.Fatalf("resolveMissionStoreRoot() = %q, want missioncontrol.ResolveStoreRoot parity", got)
	}
}

func TestResolveMissionStoreRootReturnsEmptyWithoutInputs(t *testing.T) {
	cmd := newMissionBootstrapTestCommand()

	got := resolveMissionStoreRoot(cmd)
	if got != "" {
		t.Fatalf("resolveMissionStoreRoot() = %q, want empty string", got)
	}
	if got != missioncontrol.ResolveStoreRoot("", "") {
		t.Fatalf("resolveMissionStoreRoot() = %q, want missioncontrol.ResolveStoreRoot parity", got)
	}
}

func TestMissionStatusBootstrapUsesCommittedDurableRuntimeWhenPresent(t *testing.T) {
	statusFile := filepath.Join(t.TempDir(), "status.json")
	cmd := newMissionBootstrapTestCommand()
	job := testMissionBootstrapJob()
	missionFile := writeMissionBootstrapJobFile(t, job)
	storeRoot := missioncontrol.ResolveStoreRoot("", statusFile)
	now := time.Date(2026, 4, 5, 12, 0, 0, 0, time.UTC)

	writeCommittedMissionBootstrapJobRuntimeRecord(t, storeRoot, job.ID, 7, 1, now, missioncontrol.JobStatePaused, "build")
	writeCommittedMissionBootstrapRuntimeControlRecord(t, storeRoot, 7, 1, runtimeControlForBootstrapStep(t, job, "build"))

	if err := cmd.Flags().Set("mission-file", missionFile); err != nil {
		t.Fatalf("Flags().Set(mission-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step", "build"); err != nil {
		t.Fatalf("Flags().Set(mission-step) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-status-file", statusFile); err != nil {
		t.Fatalf("Flags().Set(mission-status-file) error = %v", err)
	}

	ag := newMissionBootstrapTestLoop()
	if err := configureMissionBootstrap(cmd, ag); err != nil {
		t.Fatalf("configureMissionBootstrap() error = %v", err)
	}
	if _, ok := ag.ActiveMissionStep(); ok {
		t.Fatal("ActiveMissionStep() ok = true, want false after durable paused-runtime rehydration")
	}

	runtime, ok := ag.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if runtime.State != missioncontrol.JobStatePaused {
		t.Fatalf("MissionRuntimeState().State = %q, want %q", runtime.State, missioncontrol.JobStatePaused)
	}
	if runtime.ActiveStepID != "build" {
		t.Fatalf("MissionRuntimeState().ActiveStepID = %q, want %q", runtime.ActiveStepID, "build")
	}
}

func TestMissionStatusBootstrapFallsBackToSnapshotWhenDurableStoreAbsent(t *testing.T) {
	ag := newMissionBootstrapTestLoop()
	cmd := newMissionBootstrapTestCommand()
	job := testMissionBootstrapJob()
	missionFile := writeMissionBootstrapJobFile(t, job)
	statusFile := filepath.Join(t.TempDir(), "status.json")

	writeMissionStatusSnapshotFile(t, statusFile, missionStatusSnapshot{
		MissionFile:    missionFile,
		JobID:          job.ID,
		StepID:         "build",
		RuntimeControl: runtimeControlForBootstrapStep(t, job, "build"),
		Runtime: &missioncontrol.JobRuntimeState{
			JobID:        job.ID,
			State:        missioncontrol.JobStatePaused,
			ActiveStepID: "build",
			PausedReason: missioncontrol.RuntimePauseReasonOperatorCommand,
		},
	})

	if err := cmd.Flags().Set("mission-file", missionFile); err != nil {
		t.Fatalf("Flags().Set(mission-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step", "build"); err != nil {
		t.Fatalf("Flags().Set(mission-step) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-status-file", statusFile); err != nil {
		t.Fatalf("Flags().Set(mission-status-file) error = %v", err)
	}

	if err := configureMissionBootstrap(cmd, ag); err != nil {
		t.Fatalf("configureMissionBootstrap() error = %v", err)
	}

	runtime, ok := ag.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if runtime.State != missioncontrol.JobStatePaused {
		t.Fatalf("MissionRuntimeState().State = %q, want %q", runtime.State, missioncontrol.JobStatePaused)
	}
	if runtime.ActiveStepID != "build" {
		t.Fatalf("MissionRuntimeState().ActiveStepID = %q, want %q", runtime.ActiveStepID, "build")
	}
}

func TestLoadPersistedMissionRuntimeUsesSnapshotWhenStoreRootUnconfigured(t *testing.T) {
	job := testMissionBootstrapJob()
	statusFile := filepath.Join(t.TempDir(), "status.json")
	writeMissionStatusSnapshotFile(t, statusFile, missionStatusSnapshot{
		JobID:  job.ID,
		StepID: "build",
		Runtime: &missioncontrol.JobRuntimeState{
			JobID:        job.ID,
			State:        missioncontrol.JobStatePaused,
			ActiveStepID: "build",
		},
		RuntimeControl: runtimeControlForBootstrapStep(t, job, "build"),
	})

	runtime, control, source, ok, err := loadPersistedMissionRuntime(statusFile, "", job, time.Now().UTC())
	if err != nil {
		t.Fatalf("loadPersistedMissionRuntime() error = %v", err)
	}
	if !ok {
		t.Fatal("loadPersistedMissionRuntime() ok = false, want true")
	}
	if source != statusFile {
		t.Fatalf("loadPersistedMissionRuntime() source = %q, want %q", source, statusFile)
	}
	if runtime.State != missioncontrol.JobStatePaused || runtime.ActiveStepID != "build" {
		t.Fatalf("loadPersistedMissionRuntime() runtime = %#v, want paused build runtime", runtime)
	}
	if control == nil || control.Step.ID != "build" {
		t.Fatalf("loadPersistedMissionRuntime() control = %#v, want build control", control)
	}
}

func TestLoadPersistedMissionRuntimeSnapshotUsesSharedLegacyHelper(t *testing.T) {
	job := testMissionBootstrapJob()
	statusFile := filepath.Join(t.TempDir(), "status.json")

	original := loadValidatedLegacyMissionStatusSnapshot
	t.Cleanup(func() { loadValidatedLegacyMissionStatusSnapshot = original })
	called := 0
	loadValidatedLegacyMissionStatusSnapshot = func(path string, jobID string) (missioncontrol.MissionStatusSnapshot, bool, error) {
		called++
		if path != statusFile {
			t.Fatalf("legacy helper path = %q, want %q", path, statusFile)
		}
		if jobID != job.ID {
			t.Fatalf("legacy helper jobID = %q, want %q", jobID, job.ID)
		}
		return missioncontrol.MissionStatusSnapshot{
			JobID:  job.ID,
			StepID: "build",
			Runtime: &missioncontrol.JobRuntimeState{
				JobID:        job.ID,
				State:        missioncontrol.JobStatePaused,
				ActiveStepID: "build",
			},
			RuntimeControl: runtimeControlForBootstrapStep(t, job, "build"),
		}, true, nil
	}

	runtime, control, ok, err := loadPersistedMissionRuntimeSnapshot(statusFile, job)
	if err != nil {
		t.Fatalf("loadPersistedMissionRuntimeSnapshot() error = %v", err)
	}
	if !ok {
		t.Fatal("loadPersistedMissionRuntimeSnapshot() ok = false, want true")
	}
	if called != 1 {
		t.Fatalf("legacy helper calls = %d, want 1", called)
	}
	if runtime.State != missioncontrol.JobStatePaused || runtime.ActiveStepID != "build" {
		t.Fatalf("loadPersistedMissionRuntimeSnapshot() runtime = %#v, want paused build runtime", runtime)
	}
	if control == nil || control.Step.ID != "build" {
		t.Fatalf("loadPersistedMissionRuntimeSnapshot() control = %#v, want build control", control)
	}
}

func TestLoadPersistedMissionRuntimeUsesSharedFallbackWhenStoreRootUnconfigured(t *testing.T) {
	job := testMissionBootstrapJob()
	statusFile := filepath.Join(t.TempDir(), "status.json")

	original := loadValidatedLegacyMissionStatusSnapshot
	t.Cleanup(func() { loadValidatedLegacyMissionStatusSnapshot = original })
	called := 0
	loadValidatedLegacyMissionStatusSnapshot = func(path string, jobID string) (missioncontrol.MissionStatusSnapshot, bool, error) {
		called++
		return missioncontrol.MissionStatusSnapshot{
			JobID:  job.ID,
			StepID: "build",
			Runtime: &missioncontrol.JobRuntimeState{
				JobID:        job.ID,
				State:        missioncontrol.JobStatePaused,
				ActiveStepID: "build",
			},
			RuntimeControl: runtimeControlForBootstrapStep(t, job, "build"),
		}, true, nil
	}

	runtime, control, source, ok, err := loadPersistedMissionRuntime(statusFile, "", job, time.Now().UTC())
	if err != nil {
		t.Fatalf("loadPersistedMissionRuntime() error = %v", err)
	}
	if !ok {
		t.Fatal("loadPersistedMissionRuntime() ok = false, want true")
	}
	if source != statusFile {
		t.Fatalf("loadPersistedMissionRuntime() source = %q, want %q", source, statusFile)
	}
	if called != 1 {
		t.Fatalf("legacy helper calls = %d, want 1", called)
	}
	if runtime.State != missioncontrol.JobStatePaused || runtime.ActiveStepID != "build" {
		t.Fatalf("loadPersistedMissionRuntime() runtime = %#v, want paused build runtime", runtime)
	}
	if control == nil || control.Step.ID != "build" {
		t.Fatalf("loadPersistedMissionRuntime() control = %#v, want build control", control)
	}
}

func TestMissionStatusBootstrapFallsBackToSnapshotWhenDurableStoreEmptyForJob(t *testing.T) {
	ag := newMissionBootstrapTestLoop()
	cmd := newMissionBootstrapTestCommand()
	job := testMissionBootstrapJob()
	missionFile := writeMissionBootstrapJobFile(t, job)
	statusFile := filepath.Join(t.TempDir(), "status.json")
	storeRoot := missioncontrol.ResolveStoreRoot("", statusFile)
	now := time.Date(2026, 4, 5, 13, 0, 0, 0, time.UTC)

	writeCommittedMissionBootstrapJobRuntimeRecord(t, storeRoot, "other-job", 3, 1, now, missioncontrol.JobStatePaused, "build")
	writeMissionStatusSnapshotFile(t, statusFile, missionStatusSnapshot{
		MissionFile:    missionFile,
		JobID:          job.ID,
		StepID:         "build",
		RuntimeControl: runtimeControlForBootstrapStep(t, job, "build"),
		Runtime: &missioncontrol.JobRuntimeState{
			JobID:        job.ID,
			State:        missioncontrol.JobStatePaused,
			ActiveStepID: "build",
			PausedReason: missioncontrol.RuntimePauseReasonOperatorCommand,
		},
	})

	if err := cmd.Flags().Set("mission-file", missionFile); err != nil {
		t.Fatalf("Flags().Set(mission-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step", "build"); err != nil {
		t.Fatalf("Flags().Set(mission-step) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-status-file", statusFile); err != nil {
		t.Fatalf("Flags().Set(mission-status-file) error = %v", err)
	}

	if err := configureMissionBootstrap(cmd, ag); err != nil {
		t.Fatalf("configureMissionBootstrap() error = %v", err)
	}

	runtime, ok := ag.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if runtime.State != missioncontrol.JobStatePaused {
		t.Fatalf("MissionRuntimeState().State = %q, want %q", runtime.State, missioncontrol.JobStatePaused)
	}
}

func TestLoadPersistedMissionRuntimeUsesSharedFallbackWhenDurableStoreEmptyForJob(t *testing.T) {
	job := testMissionBootstrapJob()
	statusFile := filepath.Join(t.TempDir(), "status.json")
	storeRoot := missioncontrol.ResolveStoreRoot("", statusFile)
	now := time.Date(2026, 4, 5, 13, 0, 0, 0, time.UTC)

	writeCommittedMissionBootstrapJobRuntimeRecord(t, storeRoot, "other-job", 3, 1, now, missioncontrol.JobStatePaused, "build")

	original := loadValidatedLegacyMissionStatusSnapshot
	t.Cleanup(func() { loadValidatedLegacyMissionStatusSnapshot = original })
	called := 0
	loadValidatedLegacyMissionStatusSnapshot = func(path string, jobID string) (missioncontrol.MissionStatusSnapshot, bool, error) {
		called++
		return missioncontrol.MissionStatusSnapshot{
			JobID:  job.ID,
			StepID: "build",
			Runtime: &missioncontrol.JobRuntimeState{
				JobID:        job.ID,
				State:        missioncontrol.JobStatePaused,
				ActiveStepID: "build",
			},
			RuntimeControl: runtimeControlForBootstrapStep(t, job, "build"),
		}, true, nil
	}

	runtime, control, source, ok, err := loadPersistedMissionRuntime(statusFile, storeRoot, job, now)
	if err != nil {
		t.Fatalf("loadPersistedMissionRuntime() error = %v", err)
	}
	if !ok {
		t.Fatal("loadPersistedMissionRuntime() ok = false, want true")
	}
	if source != statusFile {
		t.Fatalf("loadPersistedMissionRuntime() source = %q, want %q", source, statusFile)
	}
	if called != 1 {
		t.Fatalf("legacy helper calls = %d, want 1", called)
	}
	if runtime.State != missioncontrol.JobStatePaused || runtime.ActiveStepID != "build" {
		t.Fatalf("loadPersistedMissionRuntime() runtime = %#v, want paused build runtime", runtime)
	}
	if control == nil || control.Step.ID != "build" {
		t.Fatalf("loadPersistedMissionRuntime() control = %#v, want build control", control)
	}
}

func TestMissionStatusBootstrapPrefersDurableRuntimeOverConflictingSnapshot(t *testing.T) {
	ag := newMissionBootstrapTestLoop()
	cmd := newMissionBootstrapTestCommand()
	job := testMissionBootstrapJob()
	missionFile := writeMissionBootstrapJobFile(t, job)
	statusFile := filepath.Join(t.TempDir(), "status.json")
	storeRoot := missioncontrol.ResolveStoreRoot("", statusFile)
	now := time.Date(2026, 4, 5, 14, 0, 0, 0, time.UTC)

	writeCommittedMissionBootstrapJobRuntimeRecord(t, storeRoot, job.ID, 5, 1, now, missioncontrol.JobStatePaused, "build")
	writeCommittedMissionBootstrapRuntimeControlRecord(t, storeRoot, 5, 1, runtimeControlForBootstrapStep(t, job, "build"))
	writeMissionStatusSnapshotFile(t, statusFile, missionStatusSnapshot{
		MissionFile:    missionFile,
		JobID:          job.ID,
		StepID:         "final",
		RuntimeControl: runtimeControlForBootstrapStep(t, job, "final"),
		Runtime: &missioncontrol.JobRuntimeState{
			JobID:        job.ID,
			State:        missioncontrol.JobStatePaused,
			ActiveStepID: "final",
			PausedReason: missioncontrol.RuntimePauseReasonOperatorCommand,
		},
	})

	if err := cmd.Flags().Set("mission-file", missionFile); err != nil {
		t.Fatalf("Flags().Set(mission-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step", "build"); err != nil {
		t.Fatalf("Flags().Set(mission-step) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-status-file", statusFile); err != nil {
		t.Fatalf("Flags().Set(mission-status-file) error = %v", err)
	}

	if err := configureMissionBootstrap(cmd, ag); err != nil {
		t.Fatalf("configureMissionBootstrap() error = %v", err)
	}

	runtime, ok := ag.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if runtime.ActiveStepID != "build" {
		t.Fatalf("MissionRuntimeState().ActiveStepID = %q, want durable %q", runtime.ActiveStepID, "build")
	}
}

func TestLoadPersistedMissionRuntimeDoesNotFallbackWhenDurableHydrationFails(t *testing.T) {
	job := testMissionBootstrapJob()
	statusFile := filepath.Join(t.TempDir(), "status.json")
	storeRoot := missioncontrol.ResolveStoreRoot("", statusFile)
	now := time.Date(2026, 4, 5, 15, 0, 0, 0, time.UTC)

	writeCommittedMissionBootstrapJobRuntimeRecord(t, storeRoot, job.ID, 6, 1, now, missioncontrol.JobStatePaused, "build")
	writeCommittedMissionBootstrapRuntimeControlRecord(t, storeRoot, 6, 1, runtimeControlForBootstrapStep(t, job, "final"))

	original := loadValidatedLegacyMissionStatusSnapshot
	t.Cleanup(func() { loadValidatedLegacyMissionStatusSnapshot = original })
	called := 0
	loadValidatedLegacyMissionStatusSnapshot = func(path string, jobID string) (missioncontrol.MissionStatusSnapshot, bool, error) {
		called++
		return missioncontrol.MissionStatusSnapshot{}, false, nil
	}

	_, _, _, ok, err := loadPersistedMissionRuntime(statusFile, storeRoot, job, now)
	if err == nil {
		t.Fatal("loadPersistedMissionRuntime() error = nil, want durable hydration failure")
	}
	if ok {
		t.Fatal("loadPersistedMissionRuntime() ok = true, want false")
	}
	if called != 0 {
		t.Fatalf("legacy helper calls = %d, want 0 on durable failure", called)
	}
}

func TestMissionStatusBootstrapFailsClosedWhenDurableHydrationFails(t *testing.T) {
	ag := newMissionBootstrapTestLoop()
	cmd := newMissionBootstrapTestCommand()
	job := testMissionBootstrapJob()
	missionFile := writeMissionBootstrapJobFile(t, job)
	statusFile := filepath.Join(t.TempDir(), "status.json")
	storeRoot := missioncontrol.ResolveStoreRoot("", statusFile)
	now := time.Date(2026, 4, 5, 15, 0, 0, 0, time.UTC)

	writeCommittedMissionBootstrapJobRuntimeRecord(t, storeRoot, job.ID, 6, 1, now, missioncontrol.JobStatePaused, "build")
	writeCommittedMissionBootstrapRuntimeControlRecord(t, storeRoot, 6, 1, runtimeControlForBootstrapStep(t, job, "final"))
	writeMissionStatusSnapshotFile(t, statusFile, missionStatusSnapshot{
		MissionFile:    missionFile,
		JobID:          job.ID,
		StepID:         "build",
		RuntimeControl: runtimeControlForBootstrapStep(t, job, "build"),
		Runtime: &missioncontrol.JobRuntimeState{
			JobID:        job.ID,
			State:        missioncontrol.JobStatePaused,
			ActiveStepID: "build",
			PausedReason: missioncontrol.RuntimePauseReasonOperatorCommand,
		},
	})

	if err := cmd.Flags().Set("mission-file", missionFile); err != nil {
		t.Fatalf("Flags().Set(mission-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step", "build"); err != nil {
		t.Fatalf("Flags().Set(mission-step) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-status-file", statusFile); err != nil {
		t.Fatalf("Flags().Set(mission-status-file) error = %v", err)
	}

	err := configureMissionBootstrap(cmd, ag)
	if err == nil {
		t.Fatal("configureMissionBootstrap() error = nil, want durable hydration failure")
	}
	if !strings.Contains(err.Error(), "durable store") {
		t.Fatalf("configureMissionBootstrap() error = %q, want durable-store failure", err)
	}
}

func TestMissionStatusBootstrapDurableTerminalStateDoesNotRestoreActiveControl(t *testing.T) {
	ag := newMissionBootstrapTestLoop()
	cmd := newMissionBootstrapTestCommand()
	job := testMissionBootstrapJob()
	missionFile := writeMissionBootstrapJobFile(t, job)
	statusFile := filepath.Join(t.TempDir(), "status.json")
	storeRoot := missioncontrol.ResolveStoreRoot("", statusFile)
	now := time.Date(2026, 4, 5, 16, 0, 0, 0, time.UTC)

	writeCommittedMissionBootstrapJobRuntimeRecord(t, storeRoot, job.ID, 8, 2, now, missioncontrol.JobStateCompleted, "")
	writeCommittedMissionBootstrapRuntimeControlRecord(t, storeRoot, 8, 1, runtimeControlForBootstrapStep(t, job, "build"))
	writeCommittedMissionBootstrapActiveJobRecord(t, storeRoot, 8, missioncontrol.JobStateRunning, "build", now, 1)

	if err := cmd.Flags().Set("mission-file", missionFile); err != nil {
		t.Fatalf("Flags().Set(mission-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step", "build"); err != nil {
		t.Fatalf("Flags().Set(mission-step) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-status-file", statusFile); err != nil {
		t.Fatalf("Flags().Set(mission-status-file) error = %v", err)
	}

	if err := configureMissionBootstrap(cmd, ag); err != nil {
		t.Fatalf("configureMissionBootstrap() error = %v", err)
	}
	if _, ok := ag.ActiveMissionStep(); ok {
		t.Fatal("ActiveMissionStep() ok = true, want false for durable terminal state")
	}
	if _, ok := ag.MissionRuntimeControl(); ok {
		t.Fatal("MissionRuntimeControl() ok = true, want false for durable terminal state")
	}
	runtime, ok := ag.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if runtime.State != missioncontrol.JobStateCompleted {
		t.Fatalf("MissionRuntimeState().State = %q, want %q", runtime.State, missioncontrol.JobStateCompleted)
	}
	if runtime.ActiveStepID != "" {
		t.Fatalf("MissionRuntimeState().ActiveStepID = %q, want empty", runtime.ActiveStepID)
	}
}

func TestMissionStatusBootstrapDurableRuntimeStillRequiresResumeApproval(t *testing.T) {
	ag := newMissionBootstrapTestLoop()
	cmd := newMissionBootstrapTestCommand()
	job := testMissionBootstrapJob()
	missionFile := writeMissionBootstrapJobFile(t, job)
	statusFile := filepath.Join(t.TempDir(), "status.json")
	storeRoot := missioncontrol.ResolveStoreRoot("", statusFile)
	now := time.Date(2026, 4, 5, 17, 0, 0, 0, time.UTC)

	writeCommittedMissionBootstrapJobRuntimeRecord(t, storeRoot, job.ID, 9, 1, now, missioncontrol.JobStateRunning, "build")
	writeCommittedMissionBootstrapRuntimeControlRecord(t, storeRoot, 9, 1, runtimeControlForBootstrapStep(t, job, "build"))

	if err := cmd.Flags().Set("mission-file", missionFile); err != nil {
		t.Fatalf("Flags().Set(mission-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step", "build"); err != nil {
		t.Fatalf("Flags().Set(mission-step) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-status-file", statusFile); err != nil {
		t.Fatalf("Flags().Set(mission-status-file) error = %v", err)
	}

	err := configureMissionBootstrap(cmd, ag)
	if err == nil {
		t.Fatal("configureMissionBootstrap() error = nil, want resume approval failure")
	}
	if !strings.Contains(err.Error(), "--mission-resume-approved") {
		t.Fatalf("configureMissionBootstrap() error = %q, want resume approval message", err)
	}
}

func TestMissionStatusRuntimePersistenceUpdatesDurableStoreAndSnapshotTogether(t *testing.T) {
	statusFile := filepath.Join(t.TempDir(), "status.json")
	cmd := newMissionBootstrapTestCommand()
	job := testMissionBootstrapJob()
	missionFile := writeMissionBootstrapJobFile(t, job)
	if err := cmd.Flags().Set("mission-status-file", statusFile); err != nil {
		t.Fatalf("Flags().Set(mission-status-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-file", missionFile); err != nil {
		t.Fatalf("Flags().Set(mission-file) error = %v", err)
	}

	hub := chat.NewHub(10)
	provider := &missionStatusFixedResponseProvider{content: "unused"}
	ag := agent.NewAgentLoop(hub, provider, provider.GetDefaultModel(), 3, "", nil)
	installMissionRuntimeChangeHook(cmd, ag)

	if err := ag.ActivateMissionStep(job, "build"); err != nil {
		t.Fatalf("ActivateMissionStep() error = %v", err)
	}
	if _, err := ag.ProcessDirect("PAUSE job-1", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(PAUSE) error = %v", err)
	}

	snapshot := readMissionStatusSnapshotFile(t, statusFile)
	if snapshot.Runtime == nil || snapshot.Runtime.State != missioncontrol.JobStatePaused {
		t.Fatalf("snapshot.Runtime = %#v, want paused runtime", snapshot.Runtime)
	}

	storeRoot := resolveMissionStoreRoot(cmd)
	assertMissionStatusSnapshotMatchesCommittedDurableState(t, snapshot, storeRoot, job.ID, time.Now().UTC())

	runtime, err := missioncontrol.HydrateCommittedJobRuntimeState(storeRoot, job.ID, time.Now().UTC())
	if err != nil {
		t.Fatalf("HydrateCommittedJobRuntimeState() error = %v", err)
	}
	if runtime.State != missioncontrol.JobStatePaused {
		t.Fatalf("HydrateCommittedJobRuntimeState().State = %q, want %q", runtime.State, missioncontrol.JobStatePaused)
	}
	control, err := missioncontrol.HydrateCommittedRuntimeControlContext(storeRoot, job.ID, time.Now().UTC())
	if err != nil {
		t.Fatalf("HydrateCommittedRuntimeControlContext() error = %v", err)
	}
	if control == nil || control.Step.ID != "build" {
		t.Fatalf("HydrateCommittedRuntimeControlContext() = %#v, want build control", control)
	}
}

func TestMissionStatusRuntimePersistenceDurableWriteFailureLeavesSnapshotUnchanged(t *testing.T) {
	statusFile := filepath.Join(t.TempDir(), "status.json")
	cmd := newMissionBootstrapTestCommand()
	job := testMissionBootstrapJob()
	missionFile := writeMissionBootstrapJobFile(t, job)
	if err := cmd.Flags().Set("mission-status-file", statusFile); err != nil {
		t.Fatalf("Flags().Set(mission-status-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-file", missionFile); err != nil {
		t.Fatalf("Flags().Set(mission-file) error = %v", err)
	}

	storeRoot := resolveMissionStoreRoot(cmd)
	seedIncoherentMissionStore(t, storeRoot, time.Date(2026, 4, 5, 18, 0, 0, 0, time.UTC))

	hub := chat.NewHub(10)
	provider := &missionStatusFixedResponseProvider{content: "unused"}
	ag := agent.NewAgentLoop(hub, provider, provider.GetDefaultModel(), 3, "", nil)
	installMissionRuntimeChangeHook(cmd, ag)

	err := ag.ActivateMissionStep(job, "build")
	if err == nil {
		t.Fatal("ActivateMissionStep() error = nil, want durable write failure")
	}
	if _, statErr := os.Stat(statusFile); !os.IsNotExist(statErr) {
		t.Fatalf("status file stat error = %v, want not-exist", statErr)
	}
	if _, ok := ag.MissionRuntimeState(); ok {
		t.Fatal("MissionRuntimeState() ok = true, want false after failed durable persist")
	}
}

func TestMissionStatusRuntimePersistenceProjectionFailureLeavesSnapshotUnchanged(t *testing.T) {
	statusFile := filepath.Join(t.TempDir(), "status.json")
	cmd := newMissionBootstrapTestCommand()
	job := testMissionBootstrapJob()
	missionFile := writeMissionBootstrapJobFile(t, job)
	if err := cmd.Flags().Set("mission-status-file", statusFile); err != nil {
		t.Fatalf("Flags().Set(mission-status-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-file", missionFile); err != nil {
		t.Fatalf("Flags().Set(mission-file) error = %v", err)
	}

	previous := missionStatusSnapshot{
		MissionFile: missionFile,
		JobID:       "previous-job",
		StepID:      "previous-step",
		UpdatedAt:   "2026-04-05T18:00:00Z",
	}
	writeMissionStatusSnapshotFile(t, statusFile, previous)

	originalWrite := writeMissionStatusSnapshotAtomic
	t.Cleanup(func() { writeMissionStatusSnapshotAtomic = originalWrite })
	writeMissionStatusSnapshotAtomic = func(path string, snapshot missionStatusSnapshot) error {
		if path == statusFile {
			return errors.New("forced projection write failure")
		}
		return originalWrite(path, snapshot)
	}

	hub := chat.NewHub(10)
	provider := &missionStatusFixedResponseProvider{content: "unused"}
	ag := agent.NewAgentLoop(hub, provider, provider.GetDefaultModel(), 3, "", nil)
	installMissionRuntimeChangeHook(cmd, ag)

	err := ag.ActivateMissionStep(job, "build")
	if err == nil {
		t.Fatal("ActivateMissionStep() error = nil, want projection failure")
	}
	if !strings.Contains(err.Error(), "forced projection write failure") {
		t.Fatalf("ActivateMissionStep() error = %q, want projection failure", err)
	}

	if got := readMissionStatusSnapshotFile(t, statusFile); !reflect.DeepEqual(got, previous) {
		t.Fatalf("status snapshot = %#v, want unchanged %#v", got, previous)
	}

	storeRoot := resolveMissionStoreRoot(cmd)
	durableRuntime, err := missioncontrol.HydrateCommittedJobRuntimeState(storeRoot, job.ID, time.Now().UTC())
	if err != nil {
		t.Fatalf("HydrateCommittedJobRuntimeState() error = %v", err)
	}
	if durableRuntime.State != missioncontrol.JobStateRunning || durableRuntime.ActiveStepID != "build" {
		t.Fatalf("HydrateCommittedJobRuntimeState() = %#v, want committed running build runtime", durableRuntime)
	}

	runtime, ok := ag.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want in-memory runtime preserved")
	}
	if runtime.State != missioncontrol.JobStateRunning || runtime.ActiveStepID != "build" {
		t.Fatalf("MissionRuntimeState() = %#v, want running build runtime", runtime)
	}
}

func TestMissionStatusBootstrapPrefersLatestDurableStateAfterMutation(t *testing.T) {
	statusFile := filepath.Join(t.TempDir(), "status.json")
	cmd := newMissionBootstrapTestCommand()
	job := testMissionBootstrapJob()
	missionFile := writeMissionBootstrapJobFile(t, job)
	if err := cmd.Flags().Set("mission-status-file", statusFile); err != nil {
		t.Fatalf("Flags().Set(mission-status-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-file", missionFile); err != nil {
		t.Fatalf("Flags().Set(mission-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step", "build"); err != nil {
		t.Fatalf("Flags().Set(mission-step) error = %v", err)
	}

	hub := chat.NewHub(10)
	provider := &missionStatusFixedResponseProvider{content: "unused"}
	ag := agent.NewAgentLoop(hub, provider, provider.GetDefaultModel(), 3, "", nil)
	installMissionRuntimeChangeHook(cmd, ag)

	if err := configureMissionBootstrap(cmd, ag); err != nil {
		t.Fatalf("configureMissionBootstrap() error = %v", err)
	}
	if _, err := ag.ProcessDirect("PAUSE job-1", 2*time.Second); err != nil {
		t.Fatalf("ProcessDirect(PAUSE) error = %v", err)
	}

	writeMissionStatusSnapshotFile(t, statusFile, missionStatusSnapshot{
		MissionFile:    missionFile,
		JobID:          job.ID,
		StepID:         "build",
		RuntimeControl: runtimeControlForBootstrapStep(t, job, "build"),
		Runtime: &missioncontrol.JobRuntimeState{
			JobID:        job.ID,
			State:        missioncontrol.JobStateRunning,
			ActiveStepID: "build",
		},
	})

	ag2 := newMissionBootstrapTestLoop()
	installMissionRuntimeChangeHook(cmd, ag2)
	if err := configureMissionBootstrap(cmd, ag2); err != nil {
		t.Fatalf("configureMissionBootstrap(second boot) error = %v", err)
	}

	runtime, ok := ag2.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if runtime.State != missioncontrol.JobStatePaused {
		t.Fatalf("MissionRuntimeState().State = %q, want durable %q", runtime.State, missioncontrol.JobStatePaused)
	}
}

func newMissionBootstrapTestLoop() *agent.AgentLoop {
	hub := chat.NewHub(10)
	provider := providers.NewStubProvider()
	return agent.NewAgentLoop(hub, provider, provider.GetDefaultModel(), 3, "", nil)
}

func writeCommittedMissionBootstrapJobRuntimeRecord(t *testing.T, root string, jobID string, writerEpoch, appliedSeq uint64, at time.Time, state missioncontrol.JobState, activeStepID string) {
	t.Helper()

	record := missioncontrol.JobRuntimeRecord{
		RecordVersion: missioncontrol.StoreRecordVersion,
		WriterEpoch:   writerEpoch,
		AppliedSeq:    appliedSeq,
		JobID:         jobID,
		State:         state,
		ActiveStepID:  activeStepID,
		CreatedAt:     at.Add(-time.Minute).UTC(),
		UpdatedAt:     at.UTC(),
		StartedAt:     at.Add(-time.Minute).UTC(),
	}
	switch state {
	case missioncontrol.JobStateRunning:
		record.ActiveStepAt = at.UTC()
	case missioncontrol.JobStateWaitingUser:
		record.WaitingAt = at.UTC()
		record.WaitingReason = "awaiting operator confirmation"
	case missioncontrol.JobStatePaused:
		record.PausedAt = at.UTC()
		record.PausedReason = missioncontrol.RuntimePauseReasonOperatorCommand
	case missioncontrol.JobStateAborted:
		record.AbortedAt = at.UTC()
		record.AbortedReason = "operator aborted"
	case missioncontrol.JobStateCompleted:
		record.CompletedAt = at.UTC()
	case missioncontrol.JobStateFailed:
		record.FailedAt = at.UTC()
	}
	if err := missioncontrol.StoreJobRuntimeRecord(root, record); err != nil {
		t.Fatalf("StoreJobRuntimeRecord() error = %v", err)
	}
}

func writeCommittedMissionBootstrapRuntimeControlRecord(t *testing.T, root string, writerEpoch, seq uint64, control *missioncontrol.RuntimeControlContext) {
	t.Helper()
	if control == nil {
		return
	}

	record := missioncontrol.RuntimeControlRecord{
		RecordVersion: missioncontrol.StoreRecordVersion,
		WriterEpoch:   writerEpoch,
		LastSeq:       seq,
		JobID:         control.JobID,
		StepID:        control.Step.ID,
		MaxAuthority:  control.MaxAuthority,
		AllowedTools:  append([]string(nil), control.AllowedTools...),
		Step:          cloneMissionBootstrapStep(control.Step),
	}
	if err := missioncontrol.StoreRuntimeControlRecord(root, record); err != nil {
		t.Fatalf("StoreRuntimeControlRecord() error = %v", err)
	}
}

func writeCommittedMissionBootstrapActiveJobRecord(t *testing.T, root string, writerEpoch uint64, state missioncontrol.JobState, activeStepID string, at time.Time, activationSeq uint64) {
	t.Helper()

	record, err := missioncontrol.NewActiveJobRecord(
		writerEpoch,
		"job-1",
		state,
		activeStepID,
		"holder-1",
		at.Add(time.Minute),
		at,
		activationSeq,
	)
	if err != nil {
		t.Fatalf("NewActiveJobRecord() error = %v", err)
	}
	if err := missioncontrol.StoreActiveJobRecord(root, record); err != nil {
		t.Fatalf("StoreActiveJobRecord() error = %v", err)
	}
}

func seedIncoherentMissionStore(t *testing.T, root string, now time.Time) {
	t.Helper()

	manifest, err := missioncontrol.InitStoreManifest(root, now)
	if err != nil {
		t.Fatalf("InitStoreManifest() error = %v", err)
	}
	manifest.CurrentWriterEpoch = 2
	if err := missioncontrol.StoreManifestRecord(root, manifest); err != nil {
		t.Fatalf("StoreManifestRecord() error = %v", err)
	}
	if err := missioncontrol.StoreWriterLockRecord(root, missioncontrol.WriterLockRecord{
		RecordVersion:  missioncontrol.StoreRecordVersion,
		WriterEpoch:    1,
		LeaseHolderID:  "other-holder",
		StartedAt:      now,
		RenewedAt:      now,
		LeaseExpiresAt: now.Add(time.Minute),
		JobID:          "job-1",
	}); err != nil {
		t.Fatalf("StoreWriterLockRecord() error = %v", err)
	}
}

func assertMissionStatusSnapshotMatchesCommittedDurableState(t *testing.T, snapshot missionStatusSnapshot, storeRoot string, jobID string, now time.Time) {
	t.Helper()

	updatedAt, err := time.Parse(time.RFC3339Nano, snapshot.UpdatedAt)
	if err != nil {
		t.Fatalf("time.Parse(snapshot.UpdatedAt=%q) error = %v", snapshot.UpdatedAt, err)
	}
	expected, err := missioncontrol.BuildCommittedMissionStatusSnapshot(
		storeRoot,
		jobID,
		missioncontrol.MissionStatusSnapshotOptions{
			MissionRequired: snapshot.MissionRequired,
			MissionFile:     snapshot.MissionFile,
			UpdatedAt:       updatedAt,
		},
	)
	if err != nil {
		t.Fatalf("BuildCommittedMissionStatusSnapshot(%q) error = %v", jobID, err)
	}
	if !reflect.DeepEqual(snapshot, expected) {
		t.Fatalf("snapshot = %#v, want durable %#v", snapshot, expected)
	}
}

func cloneMissionBootstrapStep(step missioncontrol.Step) missioncontrol.Step {
	cloned := step
	cloned.DependsOn = append([]string(nil), step.DependsOn...)
	cloned.AllowedTools = append([]string(nil), step.AllowedTools...)
	cloned.SuccessCriteria = append([]string(nil), step.SuccessCriteria...)
	cloned.LongRunningStartupCommand = append([]string(nil), step.LongRunningStartupCommand...)
	return cloned
}

func configureMissionBootstrapJobForStartupTest(t *testing.T, cmd *cobra.Command, ag *agent.AgentLoop) missioncontrol.Job {
	t.Helper()

	job, err := configureMissionBootstrapJob(cmd, ag)
	if err != nil {
		t.Fatalf("configureMissionBootstrapJob() error = %v", err)
	}
	if job == nil {
		t.Fatal("configureMissionBootstrapJob() job = nil, want bootstrapped job")
	}

	return *job
}

func TestConfigureMissionBootstrapJobAcceptsV2LongRunningCodeMissionFile(t *testing.T) {
	cmd := newMissionBootstrapTestCommand()
	ag := newMissionBootstrapTestLoop()
	missionPath := writeMissionBootstrapJobFile(t, missioncontrol.Job{
		ID:           "job-1",
		SpecVersion:  missioncontrol.JobSpecVersionV2,
		MaxAuthority: missioncontrol.AuthorityTierHigh,
		AllowedTools: []string{"filesystem"},
		Plan: missioncontrol.Plan{
			ID: "plan-1",
			Steps: []missioncontrol.Step{
				{
					ID:                        "build",
					Type:                      missioncontrol.StepTypeLongRunningCode,
					RequiredAuthority:         missioncontrol.AuthorityTierLow,
					AllowedTools:              []string{"filesystem"},
					SuccessCriteria:           []string{"Build the service artifact and record the startup command."},
					LongRunningStartupCommand: []string{"npm", "start"},
					LongRunningArtifactPath:   "dist/service.js",
				},
				{
					ID:        "final",
					Type:      missioncontrol.StepTypeFinalResponse,
					DependsOn: []string{"build"},
				},
			},
		},
	})
	if err := cmd.Flags().Set("mission-file", missionPath); err != nil {
		t.Fatalf("Flags().Set(mission-file) error = %v", err)
	}
	if err := cmd.Flags().Set("mission-step", "build"); err != nil {
		t.Fatalf("Flags().Set(mission-step) error = %v", err)
	}

	job := configureMissionBootstrapJobForStartupTest(t, cmd, ag)
	if job.SpecVersion != missioncontrol.JobSpecVersionV2 {
		t.Fatalf("Job.SpecVersion = %q, want %q", job.SpecVersion, missioncontrol.JobSpecVersionV2)
	}

	ec, ok := ag.ActiveMissionStep()
	if !ok {
		t.Fatal("ActiveMissionStep() ok = false, want activated long_running_code step")
	}
	if ec.Step == nil {
		t.Fatal("ActiveMissionStep().Step = nil, want non-nil")
	}
	if ec.Step.Type != missioncontrol.StepTypeLongRunningCode {
		t.Fatalf("ActiveMissionStep().Step.Type = %q, want %q", ec.Step.Type, missioncontrol.StepTypeLongRunningCode)
	}
	if !reflect.DeepEqual(ec.Step.LongRunningStartupCommand, []string{"npm", "start"}) {
		t.Fatalf("ActiveMissionStep().Step.LongRunningStartupCommand = %#v, want %#v", ec.Step.LongRunningStartupCommand, []string{"npm", "start"})
	}
	if ec.Step.LongRunningArtifactPath != "dist/service.js" {
		t.Fatalf("ActiveMissionStep().Step.LongRunningArtifactPath = %q, want %q", ec.Step.LongRunningArtifactPath, "dist/service.js")
	}
}

type missionStatusFixedResponseProvider struct {
	content string
}

func (p *missionStatusFixedResponseProvider) Chat(ctx context.Context, messages []providers.Message, tools []providers.ToolDefinition, model string) (providers.LLMResponse, error) {
	return providers.LLMResponse{Content: p.content}, nil
}

func (p *missionStatusFixedResponseProvider) GetDefaultModel() string {
	return "stub"
}

func writeMissionBootstrapJobFile(t *testing.T, job missioncontrol.Job, paths ...string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "mission.json")
	if len(paths) > 0 {
		path = paths[0]
	}
	data, err := json.Marshal(job)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	return path
}

func runtimeControlForBootstrapStep(t *testing.T, job missioncontrol.Job, stepID string) *missioncontrol.RuntimeControlContext {
	t.Helper()

	control, err := missioncontrol.BuildRuntimeControlContext(job, stepID)
	if err != nil {
		t.Fatalf("BuildRuntimeControlContext() error = %v", err)
	}
	return &control
}

func expectedAuthorizationApprovalContent(authority missioncontrol.AuthorityTier) missioncontrol.ApprovalRequestContent {
	return missioncontrol.ApprovalRequestContent{
		ProposedAction:   "Complete the authorization discussion step and continue to the next mission step.",
		WhyNeeded:        "This step asks the operator to explicitly approve continuation before the mission can proceed.",
		AuthorityTier:    authority,
		IdentityScope:    missioncontrol.ApprovalScopeNone,
		PublicScope:      missioncontrol.ApprovalScopeNone,
		FilesystemEffect: missioncontrol.ApprovalEffectNone,
		ProcessEffect:    missioncontrol.ApprovalEffectNone,
		NetworkEffect:    missioncontrol.ApprovalEffectNone,
		FallbackIfDenied: "Keep the mission in waiting_user and require an explicit follow-up decision before proceeding.",
	}
}

func mustReadFile(t *testing.T, path string) []byte {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", path, err)
	}
	return data
}

func writeMissionStepControlFile(t *testing.T, control missionStepControlFile, paths ...string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "control.json")
	if len(paths) > 0 {
		path = paths[0]
	}
	data, err := json.Marshal(control)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	return path
}

func writeMissionStatusSnapshotFile(t *testing.T, path string, snapshot missionStatusSnapshot) {
	t.Helper()

	data, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
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

func writeMissionInspectSlackOwnerControlFixtures(t *testing.T) (string, missioncontrol.FrankIdentityRecord, missioncontrol.FrankAccountRecord) {
	t.Helper()

	root := t.TempDir()
	now := time.Date(2026, 4, 18, 18, 0, 0, 0, time.UTC)
	providerTarget := missioncontrol.AutonomyEligibilityTargetRef{
		Kind:       missioncontrol.EligibilityTargetKindProvider,
		RegistryID: "provider-slack-owner-control",
	}
	accountTarget := missioncontrol.AutonomyEligibilityTargetRef{
		Kind:       missioncontrol.EligibilityTargetKindAccountClass,
		RegistryID: "account-class-slack-owner-control",
	}
	writeMissionInspectEligibilityFixture(t, root, providerTarget, missioncontrol.EligibilityLabelAutonomyCompatible, "slack owner-control", "check-provider-slack", now)
	writeMissionInspectEligibilityFixture(t, root, accountTarget, missioncontrol.EligibilityLabelAutonomyCompatible, "slack owner-control account", "check-account-class-slack", now.Add(time.Minute))

	identity := missioncontrol.FrankIdentityRecord{
		RecordVersion:        missioncontrol.StoreRecordVersion,
		IdentityID:           "identity-slack-owner-control",
		IdentityKind:         "owner_control_channel",
		DisplayName:          "Slack Owner Control",
		ProviderOrPlatformID: providerTarget.RegistryID,
		SlackOwnerControl:    &missioncontrol.FrankSlackOwnerControlIdentity{},
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
		AccountID:            "account-slack-owner-control",
		AccountKind:          "owner_control_channel",
		Label:                "Slack Owner Control",
		ProviderOrPlatformID: providerTarget.RegistryID,
		SlackOwnerControl:    &missioncontrol.FrankSlackOwnerControlAccount{},
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

func writeMissionInspectDiscordOwnerControlFixtures(t *testing.T) (string, missioncontrol.FrankIdentityRecord, missioncontrol.FrankAccountRecord) {
	t.Helper()

	root := t.TempDir()
	now := time.Date(2026, 4, 18, 18, 0, 0, 0, time.UTC)
	providerTarget := missioncontrol.AutonomyEligibilityTargetRef{
		Kind:       missioncontrol.EligibilityTargetKindProvider,
		RegistryID: "provider-discord-owner-control",
	}
	accountTarget := missioncontrol.AutonomyEligibilityTargetRef{
		Kind:       missioncontrol.EligibilityTargetKindAccountClass,
		RegistryID: "account-class-discord-owner-control",
	}
	writeMissionInspectEligibilityFixture(t, root, providerTarget, missioncontrol.EligibilityLabelAutonomyCompatible, "discord owner-control", "check-provider-discord", now)
	writeMissionInspectEligibilityFixture(t, root, accountTarget, missioncontrol.EligibilityLabelAutonomyCompatible, "discord owner-control account", "check-account-class-discord", now.Add(time.Minute))

	identity := missioncontrol.FrankIdentityRecord{
		RecordVersion:        missioncontrol.StoreRecordVersion,
		IdentityID:           "identity-discord-owner-control",
		IdentityKind:         "owner_control_channel",
		DisplayName:          "Discord Owner Control",
		ProviderOrPlatformID: providerTarget.RegistryID,
		DiscordOwnerControl:  &missioncontrol.FrankDiscordOwnerControlIdentity{},
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
		AccountID:            "account-discord-owner-control",
		AccountKind:          "owner_control_channel",
		Label:                "Discord Owner Control",
		ProviderOrPlatformID: providerTarget.RegistryID,
		DiscordOwnerControl:  &missioncontrol.FrankDiscordOwnerControlAccount{},
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

func writeMissionInspectWhatsAppOwnerControlFixtures(t *testing.T) (string, missioncontrol.FrankIdentityRecord, missioncontrol.FrankAccountRecord) {
	t.Helper()

	root := t.TempDir()
	now := time.Date(2026, 4, 18, 18, 0, 0, 0, time.UTC)
	providerTarget := missioncontrol.AutonomyEligibilityTargetRef{
		Kind:       missioncontrol.EligibilityTargetKindProvider,
		RegistryID: "provider-whatsapp-owner-control",
	}
	accountTarget := missioncontrol.AutonomyEligibilityTargetRef{
		Kind:       missioncontrol.EligibilityTargetKindAccountClass,
		RegistryID: "account-class-whatsapp-owner-control",
	}
	writeMissionInspectEligibilityFixture(t, root, providerTarget, missioncontrol.EligibilityLabelAutonomyCompatible, "whatsapp owner-control", "check-provider-whatsapp", now)
	writeMissionInspectEligibilityFixture(t, root, accountTarget, missioncontrol.EligibilityLabelAutonomyCompatible, "whatsapp owner-control account", "check-account-class-whatsapp", now.Add(time.Minute))

	identity := missioncontrol.FrankIdentityRecord{
		RecordVersion:        missioncontrol.StoreRecordVersion,
		IdentityID:           "identity-whatsapp-owner-control",
		IdentityKind:         "owner_control_channel",
		DisplayName:          "WhatsApp Owner Control",
		ProviderOrPlatformID: providerTarget.RegistryID,
		WhatsAppOwnerControl: &missioncontrol.FrankWhatsAppOwnerControlIdentity{},
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
		AccountID:            "account-whatsapp-owner-control",
		AccountKind:          "owner_control_channel",
		Label:                "WhatsApp Owner Control",
		ProviderOrPlatformID: providerTarget.RegistryID,
		WhatsAppOwnerControl: &missioncontrol.FrankWhatsAppOwnerControlAccount{},
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

func writeMissionInspectGitHubFixtures(t *testing.T) (string, missioncontrol.FrankIdentityRecord, missioncontrol.FrankAccountRecord) {
	t.Helper()

	root := t.TempDir()
	now := time.Date(2026, 4, 18, 18, 0, 0, 0, time.UTC)
	providerTarget := missioncontrol.AutonomyEligibilityTargetRef{
		Kind:       missioncontrol.EligibilityTargetKindProvider,
		RegistryID: "provider-github",
	}
	accountTarget := missioncontrol.AutonomyEligibilityTargetRef{
		Kind:       missioncontrol.EligibilityTargetKindAccountClass,
		RegistryID: "account-class-github",
	}
	writeMissionInspectEligibilityFixture(t, root, providerTarget, missioncontrol.EligibilityLabelAutonomyCompatible, "github", "check-provider-github", now)
	writeMissionInspectEligibilityFixture(t, root, accountTarget, missioncontrol.EligibilityLabelAutonomyCompatible, "github account", "check-account-class-github", now.Add(time.Minute))

	identity := missioncontrol.FrankIdentityRecord{
		RecordVersion:        missioncontrol.StoreRecordVersion,
		IdentityID:           "identity-github",
		IdentityKind:         "platform_identity",
		DisplayName:          "Frank GitHub",
		ProviderOrPlatformID: providerTarget.RegistryID,
		GitHub:               &missioncontrol.FrankGitHubIdentity{},
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
		AccountID:            "account-github",
		AccountKind:          "platform_account",
		Label:                "GitHub Account",
		ProviderOrPlatformID: providerTarget.RegistryID,
		GitHub: &missioncontrol.FrankGitHubAccount{
			TokenEnvVarRef: "PICOBOT_GITHUB_TOKEN",
		},
		IdentityID:           identity.IdentityID,
		ControlModel:         "agent_managed",
		RecoveryModel:        "env_ref_recoverable",
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

func writeMissionInspectStripeFixtures(t *testing.T) (string, missioncontrol.FrankIdentityRecord, missioncontrol.FrankAccountRecord) {
	t.Helper()

	root := t.TempDir()
	now := time.Date(2026, 4, 18, 20, 0, 0, 0, time.UTC)
	providerTarget := missioncontrol.AutonomyEligibilityTargetRef{
		Kind:       missioncontrol.EligibilityTargetKindProvider,
		RegistryID: "provider-stripe",
	}
	accountTarget := missioncontrol.AutonomyEligibilityTargetRef{
		Kind:       missioncontrol.EligibilityTargetKindAccountClass,
		RegistryID: "account-class-stripe",
	}
	writeMissionInspectEligibilityFixture(t, root, providerTarget, missioncontrol.EligibilityLabelAutonomyCompatible, "stripe", "check-provider-stripe", now)
	writeMissionInspectEligibilityFixture(t, root, accountTarget, missioncontrol.EligibilityLabelAutonomyCompatible, "stripe account", "check-account-class-stripe", now.Add(time.Minute))

	identity := missioncontrol.FrankIdentityRecord{
		RecordVersion:        missioncontrol.StoreRecordVersion,
		IdentityID:           "identity-stripe",
		IdentityKind:         "platform_identity",
		DisplayName:          "Frank Stripe",
		ProviderOrPlatformID: providerTarget.RegistryID,
		Stripe:               &missioncontrol.FrankStripeIdentity{},
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
		AccountID:            "account-stripe",
		AccountKind:          "platform_account",
		Label:                "Stripe Account",
		ProviderOrPlatformID: providerTarget.RegistryID,
		Stripe: &missioncontrol.FrankStripeAccount{
			SecretKeyEnvVarRef: "PICOBOT_STRIPE_SECRET_KEY",
		},
		IdentityID:           identity.IdentityID,
		ControlModel:         "agent_managed",
		RecoveryModel:        "env_ref_recoverable",
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

func writeMissionInspectPayPalFixtures(t *testing.T) (string, missioncontrol.FrankIdentityRecord, missioncontrol.FrankAccountRecord) {
	t.Helper()

	root := t.TempDir()
	now := time.Date(2026, 4, 18, 22, 0, 0, 0, time.UTC)
	providerTarget := missioncontrol.AutonomyEligibilityTargetRef{
		Kind:       missioncontrol.EligibilityTargetKindProvider,
		RegistryID: "provider-paypal",
	}
	accountTarget := missioncontrol.AutonomyEligibilityTargetRef{
		Kind:       missioncontrol.EligibilityTargetKindAccountClass,
		RegistryID: "account-class-paypal",
	}
	writeMissionInspectEligibilityFixture(t, root, providerTarget, missioncontrol.EligibilityLabelAutonomyCompatible, "paypal", "check-provider-paypal", now)
	writeMissionInspectEligibilityFixture(t, root, accountTarget, missioncontrol.EligibilityLabelAutonomyCompatible, "paypal account", "check-account-class-paypal", now.Add(time.Minute))

	identity := missioncontrol.FrankIdentityRecord{
		RecordVersion:        missioncontrol.StoreRecordVersion,
		IdentityID:           "identity-paypal",
		IdentityKind:         "platform_identity",
		DisplayName:          "Frank PayPal",
		ProviderOrPlatformID: providerTarget.RegistryID,
		PayPal:               &missioncontrol.FrankPayPalIdentity{},
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
		AccountID:            "account-paypal",
		AccountKind:          "platform_account",
		Label:                "PayPal Account",
		ProviderOrPlatformID: providerTarget.RegistryID,
		PayPal: &missioncontrol.FrankPayPalAccount{
			ClientIDEnvVarRef:     "PICOBOT_PAYPAL_CLIENT_ID",
			ClientSecretEnvVarRef: "PICOBOT_PAYPAL_CLIENT_SECRET",
			Environment:           "sandbox",
		},
		IdentityID:           identity.IdentityID,
		ControlModel:         "agent_managed",
		RecoveryModel:        "env_ref_recoverable",
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

func testMissionBootstrapJob() missioncontrol.Job {
	return missioncontrol.Job{
		ID:           "job-1",
		MaxAuthority: missioncontrol.AuthorityTierHigh,
		AllowedTools: []string{"read"},
		Plan: missioncontrol.Plan{
			ID: "plan-1",
			Steps: []missioncontrol.Step{
				{
					ID:                "build",
					Type:              missioncontrol.StepTypeOneShotCode,
					RequiredAuthority: missioncontrol.AuthorityTierLow,
					AllowedTools:      []string{"read"},
				},
				{
					ID:        "final",
					Type:      missioncontrol.StepTypeFinalResponse,
					DependsOn: []string{"build"},
				},
			},
		},
	}
}

func readMissionStatusSnapshotFile(t *testing.T, path string) missionStatusSnapshot {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	var snapshot missionStatusSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	return snapshot
}

func readMissionStepControlFile(t *testing.T, path string) missionStepControlFile {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	var control missionStepControlFile
	if err := json.Unmarshal(data, &control); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	return control
}

func assertMissionStepControlFileMissing(t *testing.T, path string) {
	t.Helper()

	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return
	}
	if err != nil {
		t.Fatalf("Stat() error = %v, want os.ErrNotExist", err)
	}
	t.Fatalf("Stat() error = nil, want os.ErrNotExist for %q", path)
}

func assertNoAtomicTempFiles(t *testing.T, dir string, targetBase string) {
	t.Helper()

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir() error = %v", err)
	}

	prefix := targetBase + ".tmp-"
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), prefix) {
			t.Fatalf("unexpected temp file %q left in %q", entry.Name(), dir)
		}
	}
}

func captureStandardLogger(t *testing.T) (*bytes.Buffer, func()) {
	t.Helper()

	logBuf := &bytes.Buffer{}
	logWriter := log.Writer()
	logFlags := log.Flags()
	logPrefix := log.Prefix()
	log.SetOutput(logBuf)
	log.SetFlags(0)
	log.SetPrefix("")

	return logBuf, func() {
		log.SetOutput(logWriter)
		log.SetFlags(logFlags)
		log.SetPrefix(logPrefix)
	}
}
