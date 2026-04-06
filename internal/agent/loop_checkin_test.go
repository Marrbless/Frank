package agent

import (
	"strings"
	"testing"
	"time"

	"github.com/local/picobot/internal/chat"
	"github.com/local/picobot/internal/missioncontrol"
)

func TestMaybeEmitMissionCheckInEmitsOncePerThirtyMinuteBucket(t *testing.T) {
	t.Parallel()

	hub := chat.NewHub(10)
	prov := &finalResponseProvider{content: "unused"}
	ag := NewAgentLoop(hub, prov, prov.GetDefaultModel(), 3, "", nil)
	if err := ag.ActivateMissionStep(testMissionJob([]string{"read"}, []string{"read"}), "build"); err != nil {
		t.Fatalf("ActivateMissionStep() error = %v", err)
	}
	ag.taskState.SetOperatorSession("telegram", "chat-42")

	ec, ok := ag.ActiveMissionStep()
	if !ok || ec.Runtime == nil {
		t.Fatalf("ActiveMissionStep() = (%#v, %t), want active runtime", ec, ok)
	}
	now := time.Date(2026, 3, 28, 12, 30, 0, 0, time.UTC)
	ec.Runtime.CreatedAt = now.Add(-31 * time.Minute)
	ec.Runtime.UpdatedAt = now.Add(-31 * time.Minute)
	ec.Runtime.StartedAt = now.Add(-31 * time.Minute)
	ec.Runtime.ActiveStepAt = now.Add(-31 * time.Minute)
	ag.taskState.SetExecutionContext(ec)

	ag.maybeEmitMissionCheckIn(now)

	select {
	case out := <-hub.Out:
		if out.Channel != "telegram" || out.ChatID != "chat-42" {
			t.Fatalf("outbound session = (%q, %q), want (%q, %q)", out.Channel, out.ChatID, "telegram", "chat-42")
		}
		if !strings.Contains(out.Content, "Mission check-in:") {
			t.Fatalf("outbound content = %q, want mission check-in content", out.Content)
		}
		if !strings.Contains(out.Content, `"job_id": "job-1"`) {
			t.Fatalf("outbound content = %q, want mission status summary", out.Content)
		}
	default:
		t.Fatal("expected a mission check-in outbound notification")
	}

	select {
	case out := <-hub.Out:
		t.Fatalf("unexpected duplicate outbound notification: %#v", out)
	default:
	}

	runtime, ok := ag.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if got := countMissionCheckIns(runtime); got != 1 {
		t.Fatalf("check-in audit count = %d, want 1", got)
	}
	if len(runtime.AuditHistory) == 0 {
		t.Fatal("AuditHistory = empty, want one check-in event")
	}
	if got := runtime.AuditHistory[len(runtime.AuditHistory)-1].ToolName; got != "check_in" {
		t.Fatalf("last audit tool = %q, want %q", got, "check_in")
	}
}

func TestMaybeEmitMissionCheckInSkipsBeforeThirtyMinutes(t *testing.T) {
	t.Parallel()

	hub := chat.NewHub(10)
	prov := &finalResponseProvider{content: "unused"}
	ag := NewAgentLoop(hub, prov, prov.GetDefaultModel(), 3, "", nil)
	if err := ag.ActivateMissionStep(testMissionJob([]string{"read"}, []string{"read"}), "build"); err != nil {
		t.Fatalf("ActivateMissionStep() error = %v", err)
	}
	ag.taskState.SetOperatorSession("telegram", "chat-42")

	ec, ok := ag.ActiveMissionStep()
	if !ok || ec.Runtime == nil {
		t.Fatalf("ActiveMissionStep() = (%#v, %t), want active runtime", ec, ok)
	}
	now := time.Date(2026, 3, 28, 12, 29, 0, 0, time.UTC)
	ec.Runtime.CreatedAt = now.Add(-29 * time.Minute)
	ec.Runtime.UpdatedAt = now.Add(-29 * time.Minute)
	ec.Runtime.StartedAt = now.Add(-29 * time.Minute)
	ec.Runtime.ActiveStepAt = now.Add(-29 * time.Minute)
	ag.taskState.SetExecutionContext(ec)

	ag.maybeEmitMissionCheckIn(now)

	select {
	case out := <-hub.Out:
		t.Fatalf("unexpected outbound notification: %#v", out)
	default:
	}
}

func TestMaybeEmitMissionDailySummaryEmitsOncePerTwentyFourHourBucket(t *testing.T) {
	t.Parallel()

	hub := chat.NewHub(10)
	prov := &finalResponseProvider{content: "unused"}
	ag := NewAgentLoop(hub, prov, prov.GetDefaultModel(), 3, "", nil)
	if err := ag.ActivateMissionStep(testMissionJob([]string{"read"}, []string{"read"}), "build"); err != nil {
		t.Fatalf("ActivateMissionStep() error = %v", err)
	}
	ag.taskState.SetOperatorSession("telegram", "chat-42")

	ec, ok := ag.ActiveMissionStep()
	if !ok || ec.Runtime == nil {
		t.Fatalf("ActiveMissionStep() = (%#v, %t), want active runtime", ec, ok)
	}
	now := time.Date(2026, 3, 29, 12, 0, 0, 0, time.UTC)
	ec.Runtime.CreatedAt = now.Add(-24 * time.Hour)
	ec.Runtime.UpdatedAt = now.Add(-24 * time.Hour)
	ec.Runtime.StartedAt = now.Add(-24 * time.Hour)
	ec.Runtime.ActiveStepAt = now.Add(-24 * time.Hour)
	ag.taskState.SetExecutionContext(ec)

	ag.maybeEmitMissionDailySummary(now)

	select {
	case out := <-hub.Out:
		if out.Channel != "telegram" || out.ChatID != "chat-42" {
			t.Fatalf("outbound session = (%q, %q), want (%q, %q)", out.Channel, out.ChatID, "telegram", "chat-42")
		}
		if !strings.Contains(out.Content, "Daily mission summary:") {
			t.Fatalf("outbound content = %q, want daily mission summary content", out.Content)
		}
		if !strings.Contains(out.Content, `"job_id": "job-1"`) {
			t.Fatalf("outbound content = %q, want mission status summary", out.Content)
		}
	default:
		t.Fatal("expected a daily mission summary outbound notification")
	}

	runtime, ok := ag.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if got := countMissionDailySummaries(runtime); got != 1 {
		t.Fatalf("daily summary audit count = %d, want 1", got)
	}
	if len(runtime.AuditHistory) == 0 {
		t.Fatal("AuditHistory = empty, want one daily summary event")
	}
	if got := runtime.AuditHistory[len(runtime.AuditHistory)-1].ToolName; got != "daily_summary" {
		t.Fatalf("last audit tool = %q, want %q", got, "daily_summary")
	}
}

func TestMaybeEmitMissionDailySummarySkipsBeforeTwentyFourHours(t *testing.T) {
	t.Parallel()

	hub := chat.NewHub(10)
	prov := &finalResponseProvider{content: "unused"}
	ag := NewAgentLoop(hub, prov, prov.GetDefaultModel(), 3, "", nil)
	if err := ag.ActivateMissionStep(testMissionJob([]string{"read"}, []string{"read"}), "build"); err != nil {
		t.Fatalf("ActivateMissionStep() error = %v", err)
	}
	ag.taskState.SetOperatorSession("telegram", "chat-42")

	ec, ok := ag.ActiveMissionStep()
	if !ok || ec.Runtime == nil {
		t.Fatalf("ActiveMissionStep() = (%#v, %t), want active runtime", ec, ok)
	}
	now := time.Date(2026, 3, 29, 11, 59, 0, 0, time.UTC)
	ec.Runtime.CreatedAt = now.Add(-(24*time.Hour - time.Minute))
	ec.Runtime.UpdatedAt = now.Add(-(24*time.Hour - time.Minute))
	ec.Runtime.StartedAt = now.Add(-(24*time.Hour - time.Minute))
	ec.Runtime.ActiveStepAt = now.Add(-(24*time.Hour - time.Minute))
	ag.taskState.SetExecutionContext(ec)

	ag.maybeEmitMissionDailySummary(now)

	select {
	case out := <-hub.Out:
		t.Fatalf("unexpected outbound notification: %#v", out)
	default:
	}
}

func TestMaybeEmitMissionDailySummaryDoesNotEmitTwiceInSameBucket(t *testing.T) {
	t.Parallel()

	hub := chat.NewHub(10)
	prov := &finalResponseProvider{content: "unused"}
	ag := NewAgentLoop(hub, prov, prov.GetDefaultModel(), 3, "", nil)
	if err := ag.ActivateMissionStep(testMissionJob([]string{"read"}, []string{"read"}), "build"); err != nil {
		t.Fatalf("ActivateMissionStep() error = %v", err)
	}
	ag.taskState.SetOperatorSession("telegram", "chat-42")

	ec, ok := ag.ActiveMissionStep()
	if !ok || ec.Runtime == nil {
		t.Fatalf("ActiveMissionStep() = (%#v, %t), want active runtime", ec, ok)
	}
	anchor := time.Date(2026, 3, 28, 12, 0, 0, 0, time.UTC)
	ec.Runtime.CreatedAt = anchor
	ec.Runtime.UpdatedAt = anchor
	ec.Runtime.StartedAt = anchor
	ec.Runtime.ActiveStepAt = anchor
	ag.taskState.SetExecutionContext(ec)

	first := anchor.Add(24 * time.Hour)
	ag.maybeEmitMissionDailySummary(first)

	select {
	case <-hub.Out:
	default:
		t.Fatal("expected the first daily mission summary outbound notification")
	}

	second := first.Add(30 * time.Minute)
	ag.maybeEmitMissionDailySummary(second)

	select {
	case out := <-hub.Out:
		t.Fatalf("unexpected duplicate daily summary notification: %#v", out)
	default:
	}

	runtime, ok := ag.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if got := countMissionDailySummaries(runtime); got != 1 {
		t.Fatalf("daily summary audit count = %d, want 1", got)
	}
}

func TestMaybeEmitMissionDailySummaryRestartSafeDuplicateSuppression(t *testing.T) {
	t.Parallel()

	hub := chat.NewHub(10)
	prov := &finalResponseProvider{content: "unused"}
	firstLoop := NewAgentLoop(hub, prov, prov.GetDefaultModel(), 3, "", nil)
	if err := firstLoop.ActivateMissionStep(testMissionJob([]string{"read"}, []string{"read"}), "build"); err != nil {
		t.Fatalf("ActivateMissionStep() error = %v", err)
	}
	firstLoop.taskState.SetOperatorSession("telegram", "chat-42")

	ec, ok := firstLoop.ActiveMissionStep()
	if !ok || ec.Runtime == nil {
		t.Fatalf("ActiveMissionStep() = (%#v, %t), want active runtime", ec, ok)
	}
	anchor := time.Date(2026, 3, 28, 12, 0, 0, 0, time.UTC)
	ec.Runtime.CreatedAt = anchor
	ec.Runtime.UpdatedAt = anchor
	ec.Runtime.StartedAt = anchor
	ec.Runtime.ActiveStepAt = anchor
	firstLoop.taskState.SetExecutionContext(ec)

	now := anchor.Add(24 * time.Hour)
	firstLoop.maybeEmitMissionDailySummary(now)

	select {
	case <-hub.Out:
	default:
		t.Fatal("expected an initial daily mission summary outbound notification")
	}

	runtime, ok := firstLoop.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	job := testMissionJob([]string{"read"}, []string{"read"})
	restartedLoop := NewAgentLoop(chat.NewHub(10), prov, prov.GetDefaultModel(), 3, "", nil)
	restartedLoop.taskState.SetOperatorSession("telegram", "chat-42")
	restartedLoop.taskState.SetExecutionContext(missioncontrol.ExecutionContext{
		Job:  &job,
		Step: &job.Plan.Steps[0],
		Runtime: &missioncontrol.JobRuntimeState{
			JobID:        runtime.JobID,
			State:        runtime.State,
			ActiveStepID: runtime.ActiveStepID,
			CreatedAt:    runtime.CreatedAt,
			UpdatedAt:    runtime.UpdatedAt,
			StartedAt:    runtime.StartedAt,
			ActiveStepAt: runtime.ActiveStepAt,
			AuditHistory: missioncontrol.CloneAuditHistory(runtime.AuditHistory),
		},
	})

	restartedLoop.maybeEmitMissionDailySummary(now.Add(30 * time.Minute))

	select {
	case out := <-restartedLoop.hub.Out:
		t.Fatalf("unexpected restart duplicate daily summary notification: %#v", out)
	default:
	}
}

func TestMissionRuntimeChangeHookEmitsApprovalRequiredNotificationOncePerPendingRequest(t *testing.T) {
	t.Parallel()

	hub := chat.NewHub(10)
	prov := &finalResponseProvider{content: "Need approval before continuing."}
	ag := NewAgentLoop(hub, prov, prov.GetDefaultModel(), 3, "", nil)
	ag.SetMissionRuntimeChangeHook(nil)
	if err := ag.ActivateMissionStep(testDiscussionMissionJob(), "build"); err != nil {
		t.Fatalf("ActivateMissionStep() error = %v", err)
	}

	resp, err := ag.ProcessDirect("continue", 2*time.Second)
	if err != nil {
		t.Fatalf("ProcessDirect() error = %v", err)
	}
	if resp != "Need approval before continuing." {
		t.Fatalf("ProcessDirect() response = %q, want discussion response", resp)
	}

	select {
	case out := <-hub.Out:
		if out.Channel != "cli" || out.ChatID != "direct" {
			t.Fatalf("outbound session = (%q, %q), want (%q, %q)", out.Channel, out.ChatID, "cli", "direct")
		}
		if !strings.Contains(out.Content, "Approval required:") {
			t.Fatalf("outbound content = %q, want approval notification prefix", out.Content)
		}
		if !strings.Contains(out.Content, `"step_id": "build"`) {
			t.Fatalf("outbound content = %q, want active approval step", out.Content)
		}
	default:
		t.Fatal("expected an approval-required outbound notification")
	}

	ag.taskState.EmitAuditEvent(missioncontrol.AuditEvent{
		JobID:       "job-1",
		StepID:      "build",
		ToolName:    "status",
		ActionClass: missioncontrol.AuditActionClassRuntime,
		Result:      missioncontrol.AuditResultApplied,
		Allowed:     true,
		Timestamp:   time.Date(2026, 4, 6, 12, 0, 0, 0, time.UTC),
	})

	select {
	case out := <-hub.Out:
		t.Fatalf("unexpected duplicate approval notification: %#v", out)
	default:
	}
}

func TestMissionRuntimeChangeHookEmitsWaitingUserNotificationOncePerWaitingState(t *testing.T) {
	t.Parallel()

	hub := chat.NewHub(10)
	prov := &finalResponseProvider{content: "Waiting for your answer."}
	ag := NewAgentLoop(hub, prov, prov.GetDefaultModel(), 3, "", nil)
	ag.SetMissionRuntimeChangeHook(nil)
	if err := ag.ActivateMissionStep(testWaitingUserNotificationMissionJob(), "build"); err != nil {
		t.Fatalf("ActivateMissionStep() error = %v", err)
	}

	resp, err := ag.ProcessDirect("continue", 2*time.Second)
	if err != nil {
		t.Fatalf("ProcessDirect() error = %v", err)
	}
	if resp != "Waiting for your answer." {
		t.Fatalf("ProcessDirect() response = %q, want waiting-user response", resp)
	}

	select {
	case out := <-hub.Out:
		if out.Channel != "cli" || out.ChatID != "direct" {
			t.Fatalf("outbound session = (%q, %q), want (%q, %q)", out.Channel, out.ChatID, "cli", "direct")
		}
		if !strings.Contains(out.Content, "Waiting for user:") {
			t.Fatalf("outbound content = %q, want waiting-user notification prefix", out.Content)
		}
		if !strings.Contains(out.Content, `"waiting_reason": "discussion_blocker"`) {
			t.Fatalf("outbound content = %q, want waiting reason", out.Content)
		}
	default:
		t.Fatal("expected a waiting-user outbound notification")
	}

	ag.maybeEmitWaitingUserNotification()

	select {
	case out := <-hub.Out:
		t.Fatalf("unexpected duplicate waiting-user notification: %#v", out)
	default:
	}

	runtime, ok := ag.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	last := runtime.AuditHistory[len(runtime.AuditHistory)-1]
	if last.ToolName != "waiting_user_notification" {
		t.Fatalf("last audit tool = %q, want %q", last.ToolName, "waiting_user_notification")
	}
}

func TestMaybeEmitCompletionNotificationEmitsOncePerCompletedRuntime(t *testing.T) {
	t.Parallel()

	hub := chat.NewHub(10)
	prov := &finalResponseProvider{content: "unused"}
	ag := NewAgentLoop(hub, prov, prov.GetDefaultModel(), 3, "", nil)
	ag.taskState.SetOperatorSession("telegram", "chat-42")

	job := testCompletionNotificationMissionJob()
	ag.taskState.SetExecutionContext(missioncontrol.ExecutionContext{
		Job: &job,
		Runtime: &missioncontrol.JobRuntimeState{
			JobID:       "job-1",
			State:       missioncontrol.JobStateCompleted,
			CompletedAt: time.Date(2026, 4, 6, 13, 30, 0, 0, time.UTC),
			CompletedSteps: []missioncontrol.RuntimeStepRecord{
				{StepID: "final", At: time.Date(2026, 4, 6, 13, 30, 0, 0, time.UTC)},
			},
		},
	})

	ag.maybeEmitCompletionNotification()

	select {
	case out := <-hub.Out:
		if out.Channel != "telegram" || out.ChatID != "chat-42" {
			t.Fatalf("outbound session = (%q, %q), want (%q, %q)", out.Channel, out.ChatID, "telegram", "chat-42")
		}
		if !strings.Contains(out.Content, "Mission completed:") {
			t.Fatalf("outbound content = %q, want completion notification prefix", out.Content)
		}
		if !strings.Contains(out.Content, `"state": "completed"`) {
			t.Fatalf("outbound content = %q, want completed status summary", out.Content)
		}
	default:
		t.Fatal("expected a completion outbound notification")
	}

	runtime, ok := ag.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	last := runtime.AuditHistory[len(runtime.AuditHistory)-1]
	if last.ToolName != "completion_notification" {
		t.Fatalf("last audit tool = %q, want %q", last.ToolName, "completion_notification")
	}

	ag.maybeEmitCompletionNotification()

	select {
	case out := <-hub.Out:
		t.Fatalf("unexpected duplicate completion notification: %#v", out)
	default:
	}
}

func TestMissionRuntimeChangeHookSuppressesCompletionNotificationDuringDirectResponse(t *testing.T) {
	t.Parallel()

	hub := chat.NewHub(10)
	prov := &finalResponseProvider{content: "Here is the requested result."}
	ag := NewAgentLoop(hub, prov, prov.GetDefaultModel(), 3, "", nil)
	ag.SetMissionRuntimeChangeHook(nil)
	if err := ag.ActivateMissionStep(testCompletionNotificationMissionJob(), "final"); err != nil {
		t.Fatalf("ActivateMissionStep() error = %v", err)
	}

	resp, err := ag.ProcessDirect("finish", 2*time.Second)
	if err != nil {
		t.Fatalf("ProcessDirect() error = %v", err)
	}
	if !strings.Contains(resp, "Here is the requested result.") {
		t.Fatalf("ProcessDirect() response = %q, want truthful final response content", resp)
	}

	select {
	case out := <-hub.Out:
		t.Fatalf("unexpected completion outbound notification during direct response: %#v", out)
	default:
	}

	runtime, ok := ag.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if runtime.State != missioncontrol.JobStateCompleted {
		t.Fatalf("MissionRuntimeState().State = %q, want %q", runtime.State, missioncontrol.JobStateCompleted)
	}
	for _, event := range runtime.AuditHistory {
		if event.ToolName == "completion_notification" {
			t.Fatalf("AuditHistory unexpectedly contains completion notification: %#v", runtime.AuditHistory)
		}
	}
}

func TestMaybeEmitBudgetPauseNotificationEmitsOncePerBudgetBlocker(t *testing.T) {
	t.Parallel()

	hub := chat.NewHub(10)
	prov := &finalResponseProvider{content: "unused"}
	ag := NewAgentLoop(hub, prov, prov.GetDefaultModel(), 3, "", nil)
	ag.taskState.SetOperatorSession("telegram", "chat-42")

	job := testMissionJob([]string{"read"}, []string{"read"})
	ec := missioncontrol.ExecutionContext{
		Job: &job,
		Step: &missioncontrol.Step{
			ID:   "build",
			Type: missioncontrol.StepTypeDiscussion,
		},
		Runtime: &missioncontrol.JobRuntimeState{
			JobID:        "job-1",
			State:        missioncontrol.JobStatePaused,
			ActiveStepID: "build",
			PausedReason: missioncontrol.RuntimePauseReasonBudgetExhausted,
			BudgetBlocker: &missioncontrol.RuntimeBudgetBlockerRecord{
				Ceiling:     "owner_messages",
				Limit:       20,
				Observed:    20,
				Message:     "owner-facing message budget exhausted",
				TriggeredAt: time.Date(2026, 4, 6, 12, 0, 0, 0, time.UTC),
			},
		},
	}
	ag.taskState.SetExecutionContext(ec)

	ag.maybeEmitBudgetPauseNotification()

	select {
	case out := <-hub.Out:
		if out.Channel != "telegram" || out.ChatID != "chat-42" {
			t.Fatalf("outbound session = (%q, %q), want (%q, %q)", out.Channel, out.ChatID, "telegram", "chat-42")
		}
		if !strings.Contains(out.Content, "Mission paused:") {
			t.Fatalf("outbound content = %q, want mission paused prefix", out.Content)
		}
		if !strings.Contains(out.Content, `"budget_blocker": {`) {
			t.Fatalf("outbound content = %q, want budget blocker summary", out.Content)
		}
	default:
		t.Fatal("expected a budget pause outbound notification")
	}

	runtime, ok := ag.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	last := runtime.AuditHistory[len(runtime.AuditHistory)-1]
	if last.ToolName != "budget_pause_notification" {
		t.Fatalf("last audit tool = %q, want %q", last.ToolName, "budget_pause_notification")
	}

	ag.maybeEmitBudgetPauseNotification()

	select {
	case out := <-hub.Out:
		t.Fatalf("unexpected duplicate budget pause notification: %#v", out)
	default:
	}
}

func testWaitingUserNotificationMissionJob() missioncontrol.Job {
	return missioncontrol.Job{
		ID:           "job-1",
		SpecVersion:  missioncontrol.JobSpecVersionV2,
		MaxAuthority: missioncontrol.AuthorityTierHigh,
		AllowedTools: []string{"read"},
		Plan: missioncontrol.Plan{
			ID: "plan-1",
			Steps: []missioncontrol.Step{
				{
					ID:      "build",
					Type:    missioncontrol.StepTypeDiscussion,
					Subtype: missioncontrol.StepSubtypeBlocker,
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

func testCompletionNotificationMissionJob() missioncontrol.Job {
	return missioncontrol.Job{
		ID:           "job-1",
		SpecVersion:  missioncontrol.JobSpecVersionV2,
		MaxAuthority: missioncontrol.AuthorityTierHigh,
		AllowedTools: []string{"read"},
		Plan: missioncontrol.Plan{
			ID: "plan-1",
			Steps: []missioncontrol.Step{
				{
					ID:   "final",
					Type: missioncontrol.StepTypeFinalResponse,
				},
			},
		},
	}
}
