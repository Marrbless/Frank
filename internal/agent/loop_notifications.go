package agent

import (
	"encoding/json"
	"log"
	"strings"
	"sync/atomic"
	"time"

	"github.com/local/picobot/internal/agent/tools"
	"github.com/local/picobot/internal/missioncontrol"
)

func countMissionCheckIns(runtime missioncontrol.JobRuntimeState) int {
	count := 0
	for _, event := range runtime.AuditHistory {
		if runtime.JobID != "" && event.JobID != runtime.JobID {
			continue
		}
		if event.ActionClass != missioncontrol.AuditActionClassRuntime || event.ToolName != "check_in" || !event.Allowed || event.Result != missioncontrol.AuditResultApplied {
			continue
		}
		count++
	}
	return count
}

func countMissionDailySummaries(runtime missioncontrol.JobRuntimeState) int {
	count := 0
	for _, event := range runtime.AuditHistory {
		if runtime.JobID != "" && event.JobID != runtime.JobID {
			continue
		}
		if event.ActionClass != missioncontrol.AuditActionClassRuntime || event.ToolName != "daily_summary" || !event.Allowed || event.Result != missioncontrol.AuditResultApplied {
			continue
		}
		count++
	}
	return count
}

func missionCheckInDue(runtime missioncontrol.JobRuntimeState, now time.Time) bool {
	if runtime.State != missioncontrol.JobStateRunning {
		return false
	}

	anchor := runtime.StartedAt
	if anchor.IsZero() {
		anchor = runtime.CreatedAt
	}
	if anchor.IsZero() {
		return false
	}

	elapsed := now.Sub(anchor)
	if elapsed < missionCheckInInterval {
		return false
	}
	return countMissionCheckIns(runtime) < int(elapsed/missionCheckInInterval)
}

func missionDailySummaryDue(runtime missioncontrol.JobRuntimeState, now time.Time) bool {
	if runtime.State != missioncontrol.JobStateRunning {
		return false
	}

	anchor := runtime.StartedAt
	if anchor.IsZero() {
		anchor = runtime.CreatedAt
	}
	if anchor.IsZero() {
		return false
	}

	elapsed := now.Sub(anchor)
	if elapsed < missionDailySummaryInterval {
		return false
	}
	return countMissionDailySummaries(runtime) < int(elapsed/missionDailySummaryInterval)
}

func buildMissionCheckInContent(taskState *tools.TaskState, runtime missioncontrol.JobRuntimeState) (string, error) {
	ec, ok := currentExecutionContext(taskState)
	if ok && ec.Job != nil {
		allowedTools := missioncontrol.EffectiveAllowedTools(ec.Job, ec.Step)
		campaignPreflight, treasuryPreflight, err := resolveOperatorReadoutCampaignAndTreasuryPreflight(ec)
		if err != nil {
			return "", err
		}
		summary, err := missioncontrol.FormatOperatorStatusSummaryWithAllowedToolsAndCampaignAndTreasuryPreflight(runtime, allowedTools, campaignPreflight, treasuryPreflight)
		if err != nil {
			return "", err
		}
		return missionCheckInContentWithDeferredSchedulerTriggers(summary, ec.MissionStoreRoot)
	}

	summary, err := missioncontrol.FormatOperatorStatusSummary(runtime)
	if err != nil {
		return "", err
	}

	storeRoot := ""
	if taskState != nil {
		_, storeRoot, _ = taskState.MissionJobWithStoreRoot()
	}
	return missionCheckInContentWithDeferredSchedulerTriggers(summary, storeRoot)
}

func missionCheckInContentWithDeferredSchedulerTriggers(summary string, missionStoreRoot string) (string, error) {
	missionStoreRoot = strings.TrimSpace(missionStoreRoot)
	if missionStoreRoot == "" {
		return "Mission check-in:\n" + summary, nil
	}

	deferred, err := missioncontrol.LoadDeferredSchedulerTriggerStatuses(missionStoreRoot)
	if err != nil || len(deferred) == 0 {
		return "Mission check-in:\n" + summary, nil
	}

	var statusSummary missioncontrol.OperatorStatusSummary
	if err := json.Unmarshal([]byte(summary), &statusSummary); err != nil {
		return "", err
	}
	statusSummary = missioncontrol.WithDeferredSchedulerTriggers(statusSummary, deferred)

	data, err := json.MarshalIndent(statusSummary, "", "  ")
	if err != nil {
		return "", err
	}
	return "Mission check-in:\n" + string(append(data, '\n')), nil
}

func buildMissionDailySummaryContent(taskState *tools.TaskState, runtime missioncontrol.JobRuntimeState) (string, error) {
	ec, ok := currentExecutionContext(taskState)
	if ok && ec.Job != nil {
		allowedTools := missioncontrol.EffectiveAllowedTools(ec.Job, ec.Step)
		campaignPreflight, treasuryPreflight, err := resolveOperatorReadoutCampaignAndTreasuryPreflight(ec)
		if err != nil {
			return "", err
		}
		summary, err := missioncontrol.FormatOperatorStatusSummaryWithAllowedToolsAndCampaignAndTreasuryPreflight(runtime, allowedTools, campaignPreflight, treasuryPreflight)
		if err != nil {
			return "", err
		}
		return missionDailySummaryContentWithDeferredSchedulerTriggers(summary, ec.MissionStoreRoot)
	}

	summary, err := missioncontrol.FormatOperatorStatusSummary(runtime)
	if err != nil {
		return "", err
	}

	storeRoot := ""
	if taskState != nil {
		_, storeRoot, _ = taskState.MissionJobWithStoreRoot()
	}
	return missionDailySummaryContentWithDeferredSchedulerTriggers(summary, storeRoot)
}

func missionDailySummaryContentWithDeferredSchedulerTriggers(summary string, missionStoreRoot string) (string, error) {
	missionStoreRoot = strings.TrimSpace(missionStoreRoot)
	if missionStoreRoot == "" {
		return "Daily mission summary:\n" + summary, nil
	}

	deferred, err := missioncontrol.LoadDeferredSchedulerTriggerStatuses(missionStoreRoot)
	if err != nil || len(deferred) == 0 {
		return "Daily mission summary:\n" + summary, nil
	}

	var statusSummary missioncontrol.OperatorStatusSummary
	if err := json.Unmarshal([]byte(summary), &statusSummary); err != nil {
		return "", err
	}
	statusSummary = missioncontrol.WithDeferredSchedulerTriggers(statusSummary, deferred)

	data, err := json.MarshalIndent(statusSummary, "", "  ")
	if err != nil {
		return "", err
	}
	return "Daily mission summary:\n" + string(append(data, '\n')), nil
}

func selectPendingApprovalRequest(runtime missioncontrol.JobRuntimeState) (missioncontrol.ApprovalRequest, bool) {
	var fallback *missioncontrol.ApprovalRequest
	for i := len(runtime.ApprovalRequests) - 1; i >= 0; i-- {
		request := runtime.ApprovalRequests[i]
		if runtime.JobID != "" && request.JobID != runtime.JobID {
			continue
		}
		if request.State != missioncontrol.ApprovalStatePending {
			continue
		}
		if runtime.ActiveStepID != "" && request.StepID == runtime.ActiveStepID {
			return request, true
		}
		if fallback == nil {
			candidate := request
			fallback = &candidate
		}
	}
	if fallback == nil {
		return missioncontrol.ApprovalRequest{}, false
	}
	return *fallback, true
}

func approvalRequestNotificationRecorded(runtime missioncontrol.JobRuntimeState, request missioncontrol.ApprovalRequest) bool {
	for _, event := range runtime.AuditHistory {
		if runtime.JobID != "" && event.JobID != runtime.JobID {
			continue
		}
		if event.ActionClass != missioncontrol.AuditActionClassRuntime || event.ToolName != "approval_request" {
			continue
		}
		if !event.Allowed || event.Result != missioncontrol.AuditResultApplied {
			continue
		}
		if event.StepID != request.StepID {
			continue
		}
		if !request.RequestedAt.IsZero() && event.Timestamp.Before(request.RequestedAt) {
			continue
		}
		return true
	}
	return false
}

func approvalRequestNotificationSession(taskState *tools.TaskState, request missioncontrol.ApprovalRequest) (string, string, bool) {
	if channel := strings.TrimSpace(request.SessionChannel); channel != "" || strings.TrimSpace(request.SessionChatID) != "" {
		return channel, strings.TrimSpace(request.SessionChatID), true
	}
	if taskState == nil {
		return "", "", false
	}
	return taskState.OperatorSession()
}

func buildMissionApprovalRequestContent(taskState *tools.TaskState, runtime missioncontrol.JobRuntimeState) (string, error) {
	ec, ok := currentExecutionContext(taskState)
	if ok && ec.Job != nil {
		allowedTools := missioncontrol.EffectiveAllowedTools(ec.Job, ec.Step)
		campaignPreflight, treasuryPreflight, err := resolveOperatorReadoutCampaignAndTreasuryPreflight(ec)
		if err != nil {
			return "", err
		}
		summary, err := missioncontrol.FormatOperatorStatusSummaryWithAllowedToolsAndCampaignAndTreasuryPreflight(runtime, allowedTools, campaignPreflight, treasuryPreflight)
		if err != nil {
			return "", err
		}
		return "Approval required:\n" + summary, nil
	}

	summary, err := missioncontrol.FormatOperatorStatusSummary(runtime)
	if err != nil {
		return "", err
	}
	return "Approval required:\n" + summary, nil
}

func budgetPauseNotificationRecorded(runtime missioncontrol.JobRuntimeState) bool {
	if runtime.BudgetBlocker == nil {
		return false
	}
	for _, event := range runtime.AuditHistory {
		if runtime.JobID != "" && event.JobID != runtime.JobID {
			continue
		}
		if event.ActionClass != missioncontrol.AuditActionClassRuntime || event.ToolName != "budget_pause_notification" {
			continue
		}
		if !event.Allowed || event.Result != missioncontrol.AuditResultApplied {
			continue
		}
		if runtime.ActiveStepID != "" && event.StepID != runtime.ActiveStepID {
			continue
		}
		if !runtime.BudgetBlocker.TriggeredAt.IsZero() && event.Timestamp.Before(runtime.BudgetBlocker.TriggeredAt) {
			continue
		}
		return true
	}
	return false
}

func waitingUserNotificationRecorded(runtime missioncontrol.JobRuntimeState) bool {
	for _, event := range runtime.AuditHistory {
		if runtime.JobID != "" && event.JobID != runtime.JobID {
			continue
		}
		if event.ActionClass != missioncontrol.AuditActionClassRuntime || event.ToolName != "waiting_user_notification" {
			continue
		}
		if !event.Allowed || event.Result != missioncontrol.AuditResultApplied {
			continue
		}
		if runtime.ActiveStepID != "" && event.StepID != runtime.ActiveStepID {
			continue
		}
		if !runtime.WaitingAt.IsZero() && event.Timestamp.Before(runtime.WaitingAt) {
			continue
		}
		return true
	}
	return false
}

func completionNotificationRecorded(runtime missioncontrol.JobRuntimeState) bool {
	stepID := latestCompletedStepID(runtime)
	for _, event := range runtime.AuditHistory {
		if runtime.JobID != "" && event.JobID != runtime.JobID {
			continue
		}
		if event.ActionClass != missioncontrol.AuditActionClassRuntime || event.ToolName != "completion_notification" {
			continue
		}
		if !event.Allowed || event.Result != missioncontrol.AuditResultApplied {
			continue
		}
		if stepID != "" && event.StepID != stepID {
			continue
		}
		if !runtime.CompletedAt.IsZero() && event.Timestamp.Before(runtime.CompletedAt) {
			continue
		}
		return true
	}
	return false
}

func completedStepExecutionContext(taskState *tools.TaskState, runtime missioncontrol.JobRuntimeState) (missioncontrol.ExecutionContext, bool, error) {
	if taskState == nil {
		return missioncontrol.ExecutionContext{}, false, nil
	}

	stepID := latestCompletedStepID(runtime)
	if strings.TrimSpace(stepID) == "" {
		return missioncontrol.ExecutionContext{}, false, nil
	}

	job, storeRoot, ok := taskState.MissionJobWithStoreRoot()
	if !ok {
		return missioncontrol.ExecutionContext{}, false, nil
	}
	if job.ID != "" && runtime.JobID != "" && job.ID != runtime.JobID {
		return missioncontrol.ExecutionContext{}, false, nil
	}

	ec, err := missioncontrol.ResolveExecutionContext(job, stepID)
	if err != nil {
		return missioncontrol.ExecutionContext{}, false, err
	}
	ec.Runtime = missioncontrol.CloneJobRuntimeState(&runtime)
	ec.MissionStoreRoot = storeRoot
	return ec, true, nil
}

func buildMissionCompletionContent(taskState *tools.TaskState, runtime missioncontrol.JobRuntimeState) (string, error) {
	ec, ok, err := completedStepExecutionContext(taskState, runtime)
	if err != nil {
		return "", err
	}
	if ok && ec.Step != nil && (ec.Step.CampaignRef != nil || ec.Step.TreasuryRef != nil) {
		campaignPreflight, treasuryPreflight, err := resolveOperatorReadoutCampaignAndTreasuryPreflight(ec)
		if err != nil {
			return "", err
		}
		summary, err := missioncontrol.FormatOperatorStatusSummaryWithAllowedToolsAndCampaignAndTreasuryPreflight(
			runtime,
			missioncontrol.EffectiveAllowedTools(ec.Job, ec.Step),
			campaignPreflight,
			treasuryPreflight,
		)
		if err != nil {
			return "", err
		}
		return "Mission completed:\n" + summary, nil
	}

	summary, err := missioncontrol.FormatOperatorStatusSummary(runtime)
	if err != nil {
		return "", err
	}
	return "Mission completed:\n" + summary, nil
}

func buildMissionWaitingUserContent(taskState *tools.TaskState, runtime missioncontrol.JobRuntimeState) (string, error) {
	ec, ok := currentExecutionContext(taskState)
	if ok && ec.Job != nil {
		allowedTools := missioncontrol.EffectiveAllowedTools(ec.Job, ec.Step)
		campaignPreflight, treasuryPreflight, err := resolveOperatorReadoutCampaignAndTreasuryPreflight(ec)
		if err != nil {
			return "", err
		}
		summary, err := missioncontrol.FormatOperatorStatusSummaryWithAllowedToolsAndCampaignAndTreasuryPreflight(runtime, allowedTools, campaignPreflight, treasuryPreflight)
		if err != nil {
			return "", err
		}
		return "Waiting for user:\n" + summary, nil
	}

	summary, err := missioncontrol.FormatOperatorStatusSummary(runtime)
	if err != nil {
		return "", err
	}
	return "Waiting for user:\n" + summary, nil
}

func buildMissionBudgetPauseContent(taskState *tools.TaskState, runtime missioncontrol.JobRuntimeState) (string, error) {
	ec, ok := currentExecutionContext(taskState)
	if ok && ec.Job != nil {
		allowedTools := missioncontrol.EffectiveAllowedTools(ec.Job, ec.Step)
		campaignPreflight, treasuryPreflight, err := resolveOperatorReadoutCampaignAndTreasuryPreflight(ec)
		if err != nil {
			return "", err
		}
		summary, err := missioncontrol.FormatOperatorStatusSummaryWithAllowedToolsAndCampaignAndTreasuryPreflight(runtime, allowedTools, campaignPreflight, treasuryPreflight)
		if err != nil {
			return "", err
		}
		return "Mission paused:\n" + summary, nil
	}

	summary, err := missioncontrol.FormatOperatorStatusSummary(runtime)
	if err != nil {
		return "", err
	}
	return "Mission paused:\n" + summary, nil
}

func resolveOperatorReadoutCampaignAndTreasuryPreflight(ec missioncontrol.ExecutionContext) (*missioncontrol.ResolvedExecutionContextCampaignPreflight, *missioncontrol.ResolvedExecutionContextTreasuryPreflight, error) {
	var campaignPreflight *missioncontrol.ResolvedExecutionContextCampaignPreflight
	if ec.Step != nil && ec.Step.CampaignRef != nil {
		resolved, err := missioncontrol.ResolveExecutionContextCampaignPreflight(ec)
		if err != nil {
			return nil, nil, err
		}
		campaignPreflight = &resolved
	}

	var treasuryPreflight *missioncontrol.ResolvedExecutionContextTreasuryPreflight
	if ec.Step != nil && ec.Step.TreasuryRef != nil {
		resolved, err := missioncontrol.ResolveExecutionContextTreasuryPreflight(ec)
		if err != nil {
			return nil, nil, err
		}
		treasuryPreflight = &resolved
	}

	return campaignPreflight, treasuryPreflight, nil
}

func (a *AgentLoop) maybeEmitApprovalRequestNotification() {
	if a == nil || a.taskState == nil || a.hub == nil {
		return
	}

	runtime, ok := a.taskState.MissionRuntimeState()
	if !ok || runtime.State != missioncontrol.JobStateWaitingUser {
		return
	}

	request, ok := selectPendingApprovalRequest(runtime)
	if !ok || approvalRequestNotificationRecorded(runtime, request) {
		return
	}
	channel, chatID, ok := approvalRequestNotificationSession(a.taskState, request)
	if !ok {
		return
	}

	exhausted, err := a.taskState.RecordOwnerFacingApprovalRequest()
	if err != nil {
		log.Printf("mission runtime approval notification accounting failed: %v", err)
		return
	}

	runtime, ok = a.taskState.MissionRuntimeState()
	if !ok {
		return
	}
	if exhausted {
		ec, ok := a.taskState.ExecutionContext()
		if !ok || ec.Job == nil {
			return
		}
		sendChannelNotification(a.hub, channel, chatID, formatBudgetBlockedResponse(ec, runtime))
		return
	}

	content, err := buildMissionApprovalRequestContent(a.taskState, runtime)
	if err != nil {
		log.Printf("mission runtime approval notification formatting failed: %v", err)
		return
	}
	sendChannelNotification(a.hub, channel, chatID, content)
}

func (a *AgentLoop) maybeEmitBudgetPauseNotification() {
	if a == nil || a.taskState == nil || a.hub == nil {
		return
	}

	runtime, ok := a.taskState.MissionRuntimeState()
	if !ok || runtime.State != missioncontrol.JobStatePaused || runtime.PausedReason != missioncontrol.RuntimePauseReasonBudgetExhausted || runtime.BudgetBlocker == nil {
		return
	}
	if budgetPauseNotificationRecorded(runtime) {
		return
	}

	channel, chatID, ok := a.taskState.OperatorSession()
	if !ok {
		return
	}

	_, err := a.taskState.RecordOwnerFacingBudgetPause()
	if err != nil {
		log.Printf("mission runtime budget-pause notification accounting failed: %v", err)
		return
	}

	runtime, ok = a.taskState.MissionRuntimeState()
	if !ok {
		return
	}
	content, err := buildMissionBudgetPauseContent(a.taskState, runtime)
	if err != nil {
		log.Printf("mission runtime budget-pause notification formatting failed: %v", err)
		return
	}
	sendChannelNotification(a.hub, channel, chatID, content)
}

func (a *AgentLoop) maybeEmitCompletionNotification() {
	if a == nil || a.taskState == nil || a.hub == nil || atomic.LoadInt32(&a.suppressTerminalNotices) > 0 {
		return
	}

	runtime, ok := a.taskState.MissionRuntimeState()
	if !ok || runtime.State != missioncontrol.JobStateCompleted || len(runtime.CompletedSteps) == 0 {
		return
	}
	if completionNotificationRecorded(runtime) {
		return
	}

	channel, chatID, ok := a.taskState.OperatorSession()
	if !ok {
		return
	}

	exhausted, err := a.taskState.RecordOwnerFacingCompletion()
	if err != nil {
		log.Printf("mission runtime completion notification accounting failed: %v", err)
		return
	}
	if exhausted {
		return
	}

	runtime, ok = a.taskState.MissionRuntimeState()
	if !ok {
		return
	}
	content, err := buildMissionCompletionContent(a.taskState, runtime)
	if err != nil {
		log.Printf("mission runtime completion notification formatting failed: %v", err)
		return
	}
	sendChannelNotification(a.hub, channel, chatID, content)
}

func (a *AgentLoop) maybeEmitWaitingUserNotification() {
	if a == nil || a.taskState == nil || a.hub == nil {
		return
	}

	runtime, ok := a.taskState.MissionRuntimeState()
	if !ok || runtime.State != missioncontrol.JobStateWaitingUser {
		return
	}
	if _, hasPendingApproval := selectPendingApprovalRequest(runtime); hasPendingApproval {
		return
	}
	if waitingUserNotificationRecorded(runtime) {
		return
	}

	channel, chatID, ok := a.taskState.OperatorSession()
	if !ok {
		return
	}

	exhausted, err := a.taskState.RecordOwnerFacingWaitingUser()
	if err != nil {
		log.Printf("mission runtime waiting-user notification accounting failed: %v", err)
		return
	}
	if exhausted {
		return
	}

	runtime, ok = a.taskState.MissionRuntimeState()
	if !ok {
		return
	}
	content, err := buildMissionWaitingUserContent(a.taskState, runtime)
	if err != nil {
		log.Printf("mission runtime waiting-user notification formatting failed: %v", err)
		return
	}
	sendChannelNotification(a.hub, channel, chatID, content)
}

func (a *AgentLoop) composeMissionRuntimeChangeHook(hook func()) func() {
	return func() {
		if hook != nil {
			hook()
		}
		a.maybeEmitCompletionNotification()
		a.maybeEmitApprovalRequestNotification()
		a.maybeEmitWaitingUserNotification()
		a.maybeEmitBudgetPauseNotification()
	}
}

func (a *AgentLoop) maybeEmitMissionCheckIn(now time.Time) {
	if a == nil || a.taskState == nil {
		return
	}

	runtime, ok := a.taskState.MissionRuntimeState()
	if !ok || !missionCheckInDue(runtime, now) {
		return
	}

	if exhausted, err := a.taskState.RecordOwnerFacingCheckIn(); err != nil {
		log.Printf("mission runtime check-in accounting failed: %v", err)
		return
	} else if exhausted {
		runtime, ok = a.taskState.MissionRuntimeState()
		if !ok {
			return
		}
		ec, ok := a.taskState.ExecutionContext()
		if !ok || ec.Job == nil {
			return
		}
		channel, chatID, ok := a.taskState.OperatorSession()
		if !ok {
			return
		}
		sendChannelNotification(a.hub, channel, chatID, formatBudgetBlockedResponse(ec, runtime))
		return
	}

	runtime, ok = a.taskState.MissionRuntimeState()
	if !ok {
		return
	}
	content, err := buildMissionCheckInContent(a.taskState, runtime)
	if err != nil {
		log.Printf("mission runtime check-in formatting failed: %v", err)
		return
	}
	channel, chatID, ok := a.taskState.OperatorSession()
	if !ok {
		return
	}
	sendChannelNotification(a.hub, channel, chatID, content)
}

func (a *AgentLoop) maybeEmitMissionDailySummary(now time.Time) {
	if a == nil || a.taskState == nil {
		return
	}

	runtime, ok := a.taskState.MissionRuntimeState()
	if !ok || !missionDailySummaryDue(runtime, now) {
		return
	}

	if exhausted, err := a.taskState.RecordOwnerFacingDailySummary(); err != nil {
		log.Printf("mission runtime daily summary accounting failed: %v", err)
		return
	} else if exhausted {
		runtime, ok = a.taskState.MissionRuntimeState()
		if !ok {
			return
		}
		ec, ok := a.taskState.ExecutionContext()
		if !ok || ec.Job == nil {
			return
		}
		channel, chatID, ok := a.taskState.OperatorSession()
		if !ok {
			return
		}
		sendChannelNotification(a.hub, channel, chatID, formatBudgetBlockedResponse(ec, runtime))
		return
	}

	runtime, ok = a.taskState.MissionRuntimeState()
	if !ok {
		return
	}
	content, err := buildMissionDailySummaryContent(a.taskState, runtime)
	if err != nil {
		log.Printf("mission runtime daily summary formatting failed: %v", err)
		return
	}
	channel, chatID, ok := a.taskState.OperatorSession()
	if !ok {
		return
	}
	sendChannelNotification(a.hub, channel, chatID, content)
}

func (a *AgentLoop) maybeEmitPeriodicMissionNotifications(now time.Time) {
	if a == nil || a.taskState == nil {
		return
	}

	runtime, ok := a.taskState.MissionRuntimeState()
	if !ok {
		return
	}
	if missionDailySummaryDue(runtime, now) {
		a.maybeEmitMissionDailySummary(now)
		return
	}
	if missionCheckInDue(runtime, now) {
		a.maybeEmitMissionCheckIn(now)
	}
}
