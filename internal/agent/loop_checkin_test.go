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
