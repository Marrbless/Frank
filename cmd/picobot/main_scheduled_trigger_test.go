package main

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/local/picobot/internal/agent"
	"github.com/local/picobot/internal/chat"
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
