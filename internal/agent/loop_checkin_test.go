package agent

import (
	"encoding/json"
	"reflect"
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
		if strings.Contains(out.Content, `"treasury_preflight"`) {
			t.Fatalf("outbound content = %q, want zero-ref check-in path unchanged", out.Content)
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

func TestMaybeEmitMissionCheckInSurfacesResolvedTreasuryPreflight(t *testing.T) {
	t.Parallel()

	root, treasury, container := writeApprovalNotificationTreasuryFixtures(t)
	hub := chat.NewHub(10)
	prov := &finalResponseProvider{content: "unused"}
	ag := NewAgentLoop(hub, prov, prov.GetDefaultModel(), 3, "", nil)

	job := testMissionJob([]string{"read"}, []string{"read"})
	job.Plan.Steps[0].TreasuryRef = &missioncontrol.TreasuryRef{TreasuryID: treasury.TreasuryID}
	if err := ag.ActivateMissionStep(job, "build"); err != nil {
		t.Fatalf("ActivateMissionStep() error = %v", err)
	}
	ag.taskState.SetMissionStoreRoot(root)
	ag.taskState.SetOperatorSession("telegram", "chat-42")

	ec, ok := ag.ActiveMissionStep()
	if !ok || ec.Runtime == nil {
		t.Fatalf("ActiveMissionStep() = (%#v, %t), want active runtime", ec, ok)
	}
	now := time.Date(2026, 4, 8, 22, 0, 0, 0, time.UTC)
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
		summary := decodeMissionCheckInSummary(t, out.Content)
		if summary.TreasuryPreflight == nil {
			t.Fatal("TreasuryPreflight = nil, want resolved treasury/container data")
		}
		if summary.TreasuryPreflight.Treasury == nil {
			t.Fatal("TreasuryPreflight.Treasury = nil, want resolved treasury record")
		}
		if !reflect.DeepEqual(*summary.TreasuryPreflight.Treasury, treasury) {
			t.Fatalf("TreasuryPreflight.Treasury = %#v, want %#v", *summary.TreasuryPreflight.Treasury, treasury)
		}
		if !reflect.DeepEqual(summary.TreasuryPreflight.Containers, []missioncontrol.FrankContainerRecord{container}) {
			t.Fatalf("TreasuryPreflight.Containers = %#v, want [%#v]", summary.TreasuryPreflight.Containers, container)
		}
	default:
		t.Fatal("expected a mission check-in outbound notification")
	}
}

func TestMaybeEmitMissionCheckInInvalidTreasuryStateFailsClosed(t *testing.T) {
	t.Parallel()

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
		CreatedAt: now.UTC(),
		UpdatedAt: now.Add(time.Minute).UTC(),
	}
	if err := missioncontrol.StoreTreasuryRecord(root, treasury); err != nil {
		t.Fatalf("StoreTreasuryRecord() error = %v", err)
	}

	hub := chat.NewHub(10)
	prov := &finalResponseProvider{content: "unused"}
	ag := NewAgentLoop(hub, prov, prov.GetDefaultModel(), 3, "", nil)

	job := testMissionJob([]string{"read"}, []string{"read"})
	job.Plan.Steps[0].TreasuryRef = &missioncontrol.TreasuryRef{TreasuryID: treasury.TreasuryID}
	if err := ag.ActivateMissionStep(job, "build"); err != nil {
		t.Fatalf("ActivateMissionStep() error = %v", err)
	}
	ag.taskState.SetMissionStoreRoot(root)
	ag.taskState.SetOperatorSession("telegram", "chat-42")

	ec, ok := ag.ActiveMissionStep()
	if !ok || ec.Runtime == nil {
		t.Fatalf("ActiveMissionStep() = (%#v, %t), want active runtime", ec, ok)
	}
	checkInAt := now.Add(31 * time.Minute)
	ec.Runtime.CreatedAt = checkInAt.Add(-31 * time.Minute)
	ec.Runtime.UpdatedAt = checkInAt.Add(-31 * time.Minute)
	ec.Runtime.StartedAt = checkInAt.Add(-31 * time.Minute)
	ec.Runtime.ActiveStepAt = checkInAt.Add(-31 * time.Minute)
	ag.taskState.SetExecutionContext(ec)

	ag.maybeEmitMissionCheckIn(checkInAt)

	select {
	case out := <-hub.Out:
		t.Fatalf("unexpected outbound notification for invalid treasury state: %#v", out)
	default:
	}

	runtime, ok := ag.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	_, err := buildMissionCheckInContent(ag.taskState, runtime)
	if err == nil {
		t.Fatal("buildMissionCheckInContent() error = nil, want fail-closed treasury preflight rejection")
	}
	if !strings.Contains(err.Error(), missioncontrol.ErrFrankContainerRecordNotFound.Error()) {
		t.Fatalf("buildMissionCheckInContent() error = %q, want missing container rejection", err)
	}
}

func TestBuildMissionCheckInContentPersistedRuntimePathUnchangedForTreasurySteps(t *testing.T) {
	t.Parallel()

	hub := chat.NewHub(10)
	prov := &finalResponseProvider{content: "unused"}
	ag := NewAgentLoop(hub, prov, prov.GetDefaultModel(), 3, "", nil)

	summary, err := buildMissionCheckInContent(ag.taskState, missioncontrol.JobRuntimeState{
		JobID:        "job-1",
		State:        missioncontrol.JobStateRunning,
		ActiveStepID: "build",
		CreatedAt:    time.Date(2026, 4, 8, 21, 29, 0, 0, time.UTC),
		UpdatedAt:    time.Date(2026, 4, 8, 21, 29, 0, 0, time.UTC),
		StartedAt:    time.Date(2026, 4, 8, 21, 29, 0, 0, time.UTC),
		ActiveStepAt: time.Date(2026, 4, 8, 21, 29, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("buildMissionCheckInContent() error = %v", err)
	}
	if !strings.HasPrefix(summary, "Mission check-in:\n") {
		t.Fatalf("summary = %q, want mission check-in prefix", summary)
	}
	if strings.Contains(summary, `"treasury_preflight"`) {
		t.Fatalf("summary = %q, want persisted runtime path unchanged", summary)
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

func TestMaybeEmitPeriodicMissionNotificationsPrefersDailySummaryOverCheckInAtTwentyFourHours(t *testing.T) {
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

	ag.maybeEmitPeriodicMissionNotifications(anchor.Add(24 * time.Hour))

	select {
	case out := <-hub.Out:
		if !strings.Contains(out.Content, "Daily mission summary:") {
			t.Fatalf("outbound content = %q, want daily mission summary content", out.Content)
		}
		if strings.Contains(out.Content, "Mission check-in:") {
			t.Fatalf("outbound content = %q, want daily summary without check-in prefix", out.Content)
		}
	default:
		t.Fatal("expected a daily mission summary outbound notification")
	}

	select {
	case out := <-hub.Out:
		t.Fatalf("unexpected extra outbound notification: %#v", out)
	default:
	}

	runtime, ok := ag.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if got := countMissionDailySummaries(runtime); got != 1 {
		t.Fatalf("daily summary audit count = %d, want 1", got)
	}
	if got := countMissionCheckIns(runtime); got != 0 {
		t.Fatalf("check-in audit count = %d, want 0 when daily summary suppresses same-tick check-in", got)
	}
	last := runtime.AuditHistory[len(runtime.AuditHistory)-1]
	if last.ToolName != "daily_summary" {
		t.Fatalf("last audit tool = %q, want %q", last.ToolName, "daily_summary")
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
		if strings.Contains(out.Content, `"treasury_preflight"`) {
			t.Fatalf("outbound content = %q, want zero-ref approval notification path unchanged", out.Content)
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

func TestMissionRuntimeChangeHookApprovalNotificationSurfacesResolvedTreasuryPreflight(t *testing.T) {
	t.Parallel()

	root, treasury, container := writeApprovalNotificationTreasuryFixtures(t)
	hub := chat.NewHub(10)
	prov := &finalResponseProvider{content: "Need approval before continuing."}
	ag := NewAgentLoop(hub, prov, prov.GetDefaultModel(), 3, "", nil)
	ag.SetMissionRuntimeChangeHook(nil)

	job := testDiscussionMissionJob()
	job.Plan.Steps[0].TreasuryRef = &missioncontrol.TreasuryRef{TreasuryID: treasury.TreasuryID}
	ag.taskState.SetMissionStoreRoot(root)
	if err := ag.ActivateMissionStep(job, "build"); err != nil {
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
		summary := decodeApprovalNotificationSummary(t, out.Content)
		if summary.TreasuryPreflight == nil {
			t.Fatal("TreasuryPreflight = nil, want resolved treasury/container data")
		}
		if summary.TreasuryPreflight.Treasury == nil {
			t.Fatal("TreasuryPreflight.Treasury = nil, want resolved treasury record")
		}
		if !reflect.DeepEqual(*summary.TreasuryPreflight.Treasury, treasury) {
			t.Fatalf("TreasuryPreflight.Treasury = %#v, want %#v", *summary.TreasuryPreflight.Treasury, treasury)
		}
		if !reflect.DeepEqual(summary.TreasuryPreflight.Containers, []missioncontrol.FrankContainerRecord{container}) {
			t.Fatalf("TreasuryPreflight.Containers = %#v, want [%#v]", summary.TreasuryPreflight.Containers, container)
		}
	default:
		t.Fatal("expected an approval-required outbound notification")
	}
}

func TestMissionRuntimeChangeHookApprovalNotificationInvalidTreasuryStateFailsClosed(t *testing.T) {
	t.Parallel()

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
		CreatedAt: now.UTC(),
		UpdatedAt: now.Add(time.Minute).UTC(),
	}
	if err := missioncontrol.StoreTreasuryRecord(root, treasury); err != nil {
		t.Fatalf("StoreTreasuryRecord() error = %v", err)
	}

	hub := chat.NewHub(10)
	prov := &finalResponseProvider{content: "Need approval before continuing."}
	ag := NewAgentLoop(hub, prov, prov.GetDefaultModel(), 3, "", nil)
	ag.SetMissionRuntimeChangeHook(nil)

	job := testDiscussionMissionJob()
	job.Plan.Steps[0].TreasuryRef = &missioncontrol.TreasuryRef{TreasuryID: treasury.TreasuryID}
	ag.taskState.SetMissionStoreRoot(root)
	if err := ag.ActivateMissionStep(job, "build"); err != nil {
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
		t.Fatalf("unexpected outbound notification for invalid treasury state: %#v", out)
	default:
	}

	runtime, ok := ag.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	_, err = buildMissionApprovalRequestContent(ag.taskState, runtime)
	if err == nil {
		t.Fatal("buildMissionApprovalRequestContent() error = nil, want fail-closed treasury preflight rejection")
	}
	if !strings.Contains(err.Error(), missioncontrol.ErrFrankContainerRecordNotFound.Error()) {
		t.Fatalf("buildMissionApprovalRequestContent() error = %q, want missing container rejection", err)
	}
}

func TestBuildMissionApprovalRequestContentPersistedRuntimePathUnchangedForTreasurySteps(t *testing.T) {
	t.Parallel()

	hub := chat.NewHub(10)
	prov := &finalResponseProvider{content: "unused"}
	ag := NewAgentLoop(hub, prov, prov.GetDefaultModel(), 3, "", nil)

	summary, err := buildMissionApprovalRequestContent(ag.taskState, missioncontrol.JobRuntimeState{
		JobID:         "job-1",
		State:         missioncontrol.JobStateWaitingUser,
		ActiveStepID:  "build",
		WaitingReason: "discussion_authorization",
		WaitingAt:     time.Date(2026, 4, 8, 21, 30, 0, 0, time.UTC),
		ApprovalRequests: []missioncontrol.ApprovalRequest{
			{
				JobID:           "job-1",
				StepID:          "build",
				RequestedAction: missioncontrol.ApprovalRequestedActionStepComplete,
				Scope:           missioncontrol.ApprovalScopeMissionStep,
				RequestedVia:    missioncontrol.ApprovalRequestedViaRuntime,
				State:           missioncontrol.ApprovalStatePending,
				RequestedAt:     time.Date(2026, 4, 8, 21, 29, 0, 0, time.UTC),
			},
		},
	})
	if err != nil {
		t.Fatalf("buildMissionApprovalRequestContent() error = %v", err)
	}
	if !strings.HasPrefix(summary, "Approval required:\n") {
		t.Fatalf("summary = %q, want approval notification prefix", summary)
	}
	if strings.Contains(summary, `"treasury_preflight"`) {
		t.Fatalf("summary = %q, want persisted runtime path unchanged", summary)
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

func decodeApprovalNotificationSummary(t *testing.T, content string) missioncontrol.OperatorStatusSummary {
	t.Helper()

	const prefix = "Approval required:\n"
	if !strings.HasPrefix(content, prefix) {
		t.Fatalf("content = %q, want prefix %q", content, prefix)
	}

	var summary missioncontrol.OperatorStatusSummary
	if err := json.Unmarshal([]byte(strings.TrimPrefix(content, prefix)), &summary); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	return summary
}

func decodeMissionCheckInSummary(t *testing.T, content string) missioncontrol.OperatorStatusSummary {
	t.Helper()

	const prefix = "Mission check-in:\n"
	if !strings.HasPrefix(content, prefix) {
		t.Fatalf("content = %q, want prefix %q", content, prefix)
	}

	var summary missioncontrol.OperatorStatusSummary
	if err := json.Unmarshal([]byte(strings.TrimPrefix(content, prefix)), &summary); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	return summary
}

func writeApprovalNotificationTreasuryFixtures(t *testing.T) (string, missioncontrol.TreasuryRecord, missioncontrol.FrankContainerRecord) {
	t.Helper()

	root := t.TempDir()
	now := time.Date(2026, 4, 8, 21, 0, 0, 0, time.UTC)
	target := missioncontrol.AutonomyEligibilityTargetRef{
		Kind:       missioncontrol.EligibilityTargetKindTreasuryContainerClass,
		RegistryID: "container-class-wallet",
	}
	writeApprovalNotificationEligibilityFixture(t, root, missioncontrol.PlatformRecord{
		PlatformID:       target.RegistryID,
		PlatformName:     "container-class-wallet",
		TargetClass:      target.Kind,
		EligibilityLabel: missioncontrol.EligibilityLabelAutonomyCompatible,
		LastCheckID:      "check-container-class-wallet",
		Notes:            []string{"registry note"},
		UpdatedAt:        now.UTC(),
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
		CheckedAt:              now.UTC(),
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

func writeApprovalNotificationEligibilityFixture(t *testing.T, root string, platform missioncontrol.PlatformRecord, check missioncontrol.EligibilityCheckRecord) {
	t.Helper()

	if err := missioncontrol.StorePlatformRecord(root, platform); err != nil {
		t.Fatalf("StorePlatformRecord() error = %v", err)
	}
	if err := missioncontrol.StoreEligibilityCheckRecord(root, check); err != nil {
		t.Fatalf("StoreEligibilityCheckRecord() error = %v", err)
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
