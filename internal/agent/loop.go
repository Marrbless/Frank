package agent

import (
	"context"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/local/picobot/internal/agent/memory"
	"github.com/local/picobot/internal/agent/tools"
	"github.com/local/picobot/internal/chat"
	"github.com/local/picobot/internal/config"
	"github.com/local/picobot/internal/cron"
	"github.com/local/picobot/internal/mcp"
	"github.com/local/picobot/internal/missioncontrol"
	"github.com/local/picobot/internal/providers"
	"github.com/local/picobot/internal/session"
)

var rememberRE = regexp.MustCompile(`(?i)^remember(?:\s+to)?\s+(.+)$`)
var approvalCommandRE = regexp.MustCompile(`(?i)^\s*(approve|deny)\s+(\S+)\s+(\S+)\s*$`)
var revokeApprovalCommandRE = regexp.MustCompile(`(?i)^\s*(revoke_approval)\s+(\S+)\s+(\S+)\s*$`)
var hotUpdateHelpCommandRE = regexp.MustCompile(`(?i)^\s*(hot_update_help|help\s+hot_update|help\s+v4)\s*$`)
var rollbackRecordCommandRE = regexp.MustCompile(`(?i)^\s*(rollback_record)\s+(\S+)\s+(\S+)\s+(\S+)\s*$`)
var hotUpdateGateRecordCommandRE = regexp.MustCompile(`(?i)^\s*(hot_update_gate_record)\s+(\S+)\s+(\S+)\s+(\S+)\s*$`)
var hotUpdateCanaryRequirementCreateCommandRE = regexp.MustCompile(`(?i)^\s*(hot_update_canary_requirement_create)\s+(\S+)\s+(\S+)\s*$`)
var hotUpdateCanaryEvidenceCreateCommandRE = regexp.MustCompile(`(?i)^\s*(hot_update_canary_evidence_create)\s+(\S+)\s+(\S+)\s+(\S+)\s+(\S+)(?:\s+(.*?))?\s*$`)
var hotUpdateCanarySatisfactionAuthorityCreateCommandRE = regexp.MustCompile(`(?i)^\s*(hot_update_canary_satisfaction_authority_create)\s+(\S+)\s+(\S+)\s*$`)
var hotUpdateOwnerApprovalRequestCreateCommandRE = regexp.MustCompile(`(?i)^\s*(hot_update_owner_approval_request_create)\s+(\S+)\s+(\S+)\s*$`)
var hotUpdateOwnerApprovalDecisionCreateCommandRE = regexp.MustCompile(`(?i)^\s*(hot_update_owner_approval_decision_create)\s+(\S+)\s+(\S+)\s+(\S+)(?:\s+(.*?))?\s*$`)
var hotUpdateCanaryGateCreateCommandRE = regexp.MustCompile(`(?i)^\s*(hot_update_canary_gate_create)\s+(\S+)\s+(\S+)(?:\s+(\S+))?\s*$`)
var hotUpdateGateFromDecisionCommandRE = regexp.MustCompile(`(?i)^\s*(hot_update_gate_from_decision)\s+(\S+)\s+(\S+)\s*$`)
var hotUpdateGatePhaseCommandRE = regexp.MustCompile(`(?i)^\s*(hot_update_gate_phase)\s+(\S+)\s+(\S+)\s+(\S+)\s*$`)
var hotUpdateGateExecuteCommandRE = regexp.MustCompile(`(?i)^\s*(hot_update_gate_execute)\s+(\S+)\s+(\S+)\s*$`)
var hotUpdateGateReloadCommandRE = regexp.MustCompile(`(?i)^\s*(hot_update_gate_reload)\s+(\S+)\s+(\S+)\s*$`)
var hotUpdateGateFailCommandRE = regexp.MustCompile(`(?i)^\s*(hot_update_gate_fail)\s+(\S+)\s+(\S+)(?:\s+(.*?))?\s*$`)
var hotUpdateExecutionReadyCommandRE = regexp.MustCompile(`(?i)^\s*(hot_update_execution_ready)\s+(\S+)\s+(\S+)\s+(\S+)(?:\s+(.*?))?\s*$`)
var hotUpdateOutcomeCreateCommandRE = regexp.MustCompile(`(?i)^\s*(hot_update_outcome_create)\s+(\S+)\s+(\S+)\s*$`)
var hotUpdatePromotionCreateCommandRE = regexp.MustCompile(`(?i)^\s*(hot_update_promotion_create)\s+(\S+)\s+(\S+)\s*$`)
var hotUpdateLKGRecertifyCommandRE = regexp.MustCompile(`(?i)^\s*(hot_update_lkg_recertify)\s+(\S+)\s+(\S+)\s*$`)
var rollbackApplyRecordCommandRE = regexp.MustCompile(`(?i)^\s*(rollback_apply_record)\s+(\S+)\s+(\S+)\s+(\S+)\s*$`)
var rollbackApplyPhaseCommandRE = regexp.MustCompile(`(?i)^\s*(rollback_apply_phase)\s+(\S+)\s+(\S+)\s+(\S+)\s*$`)
var rollbackApplyExecuteCommandRE = regexp.MustCompile(`(?i)^\s*(rollback_apply_execute)\s+(\S+)\s+(\S+)\s*$`)
var rollbackApplyReloadCommandRE = regexp.MustCompile(`(?i)^\s*(rollback_apply_reload)\s+(\S+)\s+(\S+)\s*$`)
var rollbackApplyFailCommandRE = regexp.MustCompile(`(?i)^\s*(rollback_apply_fail)\s+(\S+)\s+(\S+)(?:\s+(.*?))?\s*$`)
var runtimeCommandRE = regexp.MustCompile(`(?i)^\s*(pause|resume|abort|status)\s+(\S+)\s*$`)
var inspectCommandRE = regexp.MustCompile(`(?i)^\s*(inspect)\s+(\S+)\s+(\S+)\s*$`)
var setStepCommandRE = regexp.MustCompile(`(?i)^\s*(set_step)\s+(\S+)\s+(\S+)\s*$`)
var mcpHTTPStatusRE = regexp.MustCompile(`\bHTTP\s+(\d{3})\b`)
var mcpJSONRPCErrorRE = regexp.MustCompile(`jsonrpc error\s+(-?\d+)`)

const (
	missionCheckInInterval      = 30 * time.Minute
	missionDailySummaryInterval = 24 * time.Hour
)

// sendChannelNotification delivers a non-blocking status message back to the
// originating channel so the user can see tool progress in real time.
// It is a no-op for system channels (heartbeat, cron) that have no user-facing chat.
func sendChannelNotification(hub *chat.Hub, channel, chatID, content string) {
	if isSystemChannel(channel) {
		return
	}
	sendAgentReply(hub, channel, chatID, content)
}

func sendAgentReply(hub *chat.Hub, channel, chatID, content string) {
	if isSystemChannel(channel) {
		return
	}
	out := chat.Outbound{Channel: channel, ChatID: chatID, Content: content}
	select {
	case hub.Out <- out:
	default:
		log.Println("Outbound channel full, dropping message")
	}
}

// isSystemChannel reports whether a channel is a background/system trigger
// (heartbeat, cron) rather than an interactive user-facing channel.
// Messages from system channels are processed statelessly: no session history
// is loaded as context and nothing is written back to disk. This prevents the
// heartbeat session file from growing unboundedly and keeps each invocation's
// context window small.
func isSystemChannel(channel string) bool {
	switch channel {
	case "heartbeat", "cron":
		return true
	default:
		return false
	}
}

func activeToolDefinitions(reg *tools.Registry, taskState *tools.TaskState) []providers.ToolDefinition {
	if taskState != nil {
		if ec, ok := taskState.ExecutionContext(); ok {
			return reg.DefinitionsForExecutionContext(&ec)
		}
	}
	return reg.Definitions()
}

func currentExecutionContext(taskState *tools.TaskState) (missioncontrol.ExecutionContext, bool) {
	if taskState == nil {
		return missioncontrol.ExecutionContext{}, false
	}
	return taskState.ExecutionContext()
}

func completeMissionStepOutput(taskState *tools.TaskState, finalContent string, successfulTools []missioncontrol.RuntimeToolCallEvidence) string {
	ec, ok := currentExecutionContext(taskState)
	if taskState == nil || !ok {
		return finalContent
	}

	if err := taskState.ApplyStepOutput(finalContent, successfulTools); err != nil {
		log.Printf("mission runtime step completion validation failed: %v", err)
		return finalContent
	}

	if runtime, ok := taskState.MissionRuntimeState(); ok && runtime.PausedReason == missioncontrol.RuntimePauseReasonBudgetExhausted && runtime.BudgetBlocker != nil {
		return formatBudgetBlockedResponse(ec, runtime)
	}

	return missioncontrol.NormalizeFinalResponse(ec, finalContent)
}

func latestCompletedStepID(runtime missioncontrol.JobRuntimeState) string {
	if len(runtime.CompletedSteps) == 0 {
		return ""
	}
	return runtime.CompletedSteps[len(runtime.CompletedSteps)-1].StepID
}

func formatBudgetBlockedResponse(ec missioncontrol.ExecutionContext, runtime missioncontrol.JobRuntimeState) string {
	message := "Mission paused: budget exhausted."
	if runtime.BudgetBlocker != nil {
		if blocker := strings.TrimSpace(runtime.BudgetBlocker.Message); blocker != "" {
			message = "Mission paused: " + blocker
			if !strings.HasSuffix(message, ".") {
				message += "."
			}
		}
	}

	updated := ec
	updated.Runtime = &runtime
	return missioncontrol.NormalizeFinalResponse(updated, message)
}

func checkActiveBudgetBeforeToolCall(taskState *tools.TaskState) (string, bool) {
	ec, ok := currentExecutionContext(taskState)
	if taskState == nil || !ok {
		return "", false
	}

	exhausted, err := taskState.EnforceUnattendedWallClockBudget()
	if err != nil {
		log.Printf("mission runtime budget enforcement failed: %v", err)
		return "", false
	}
	if !exhausted {
		return "", false
	}

	runtime, ok := taskState.MissionRuntimeState()
	if !ok {
		return "Mission paused: budget exhausted.", true
	}
	return formatBudgetBlockedResponse(ec, runtime), true
}

func recordFailedToolAction(taskState *tools.TaskState, toolName string, toolErr error) (string, bool) {
	ec, ok := currentExecutionContext(taskState)
	if taskState == nil || !ok || toolErr == nil {
		return "", false
	}
	if strings.HasPrefix(toolErr.Error(), "tool rejected: ") {
		return "", false
	}

	exhausted, err := taskState.RecordFailedToolAction(toolName, tools.SurfaceToolExecutionError(toolName, toolErr))
	if err != nil {
		log.Printf("mission runtime failed-action accounting failed: %v", err)
		return "", false
	}
	if !exhausted {
		return "", false
	}

	runtime, ok := taskState.MissionRuntimeState()
	if !ok {
		return "Mission paused: budget exhausted.", true
	}
	return formatBudgetBlockedResponse(ec, runtime), true
}

func recordOwnerFacingMessage(taskState *tools.TaskState, toolName string) (string, bool) {
	ec, ok := currentExecutionContext(taskState)
	if taskState == nil || !ok || toolName != "message" {
		return "", false
	}

	exhausted, err := taskState.RecordOwnerFacingMessage()
	if err != nil {
		log.Printf("mission runtime owner-message accounting failed: %v", err)
		return "", false
	}
	if !exhausted {
		return "", false
	}

	runtime, ok := taskState.MissionRuntimeState()
	if !ok {
		return "Mission paused: budget exhausted.", true
	}
	return formatBudgetBlockedResponse(ec, runtime), true
}

func recordOperatorDenyAcknowledgement(taskState *tools.TaskState) (string, bool) {
	ec, ok := currentExecutionContext(taskState)
	if taskState == nil || !ok {
		return "", false
	}

	exhausted, err := taskState.RecordOwnerFacingDenyAck()
	if err != nil {
		log.Printf("mission runtime deny acknowledgement accounting failed: %v", err)
		return "", false
	}
	if !exhausted {
		return "", false
	}

	runtime, ok := taskState.MissionRuntimeState()
	if !ok {
		return "Mission paused: budget exhausted.", true
	}
	return formatBudgetBlockedResponse(ec, runtime), true
}

func recordOperatorPauseAcknowledgement(taskState *tools.TaskState) (string, bool) {
	ec, ok := currentExecutionContext(taskState)
	if taskState == nil || !ok {
		return "", false
	}

	exhausted, err := taskState.RecordOwnerFacingPauseAck()
	if err != nil {
		log.Printf("mission runtime pause acknowledgement accounting failed: %v", err)
		return "", false
	}
	if !exhausted {
		return "", false
	}

	runtime, ok := taskState.MissionRuntimeState()
	if !ok {
		return "Mission paused: budget exhausted.", true
	}
	return formatBudgetBlockedResponse(ec, runtime), true
}

func recordOperatorSetStepAcknowledgement(taskState *tools.TaskState) (string, bool) {
	ec, ok := currentExecutionContext(taskState)
	if taskState == nil || !ok {
		return "", false
	}

	exhausted, err := taskState.RecordOwnerFacingSetStepAck()
	if err != nil {
		log.Printf("mission runtime set-step acknowledgement accounting failed: %v", err)
		return "", false
	}
	if !exhausted {
		return "", false
	}

	runtime, ok := taskState.MissionRuntimeState()
	if !ok {
		return "Mission paused: budget exhausted.", true
	}
	return formatBudgetBlockedResponse(ec, runtime), true
}

func recordOperatorRevokeApprovalAcknowledgement(taskState *tools.TaskState) (string, bool) {
	ec, ok := currentExecutionContext(taskState)
	if taskState == nil || !ok {
		return "", false
	}

	exhausted, err := taskState.RecordOwnerFacingRevokeApprovalAck()
	if err != nil {
		log.Printf("mission runtime revoke-approval acknowledgement accounting failed: %v", err)
		return "", false
	}
	if !exhausted {
		return "", false
	}

	runtime, ok := taskState.MissionRuntimeState()
	if !ok {
		return "Mission paused: budget exhausted.", true
	}
	return formatBudgetBlockedResponse(ec, runtime), true
}

func recordOperatorResumeAcknowledgement(taskState *tools.TaskState) (string, bool) {
	ec, ok := currentExecutionContext(taskState)
	if taskState == nil || !ok {
		return "", false
	}

	exhausted, err := taskState.RecordOwnerFacingResumeAck()
	if err != nil {
		log.Printf("mission runtime resume acknowledgement accounting failed: %v", err)
		return "", false
	}
	if !exhausted {
		return "", false
	}

	runtime, ok := taskState.MissionRuntimeState()
	if !ok {
		return "Mission paused: budget exhausted.", true
	}
	return formatBudgetBlockedResponse(ec, runtime), true
}

func (a *AgentLoop) completeMissionStepOutput(finalContent string, successfulTools []missioncontrol.RuntimeToolCallEvidence) string {
	if a == nil {
		return completeMissionStepOutput(nil, finalContent, successfulTools)
	}
	atomic.AddInt32(&a.suppressTerminalNotices, 1)
	defer atomic.AddInt32(&a.suppressTerminalNotices, -1)
	return completeMissionStepOutput(a.taskState, finalContent, successfulTools)
}

// AgentLoop is the core processing loop; it holds an LLM provider, tools, sessions and context builder.
type AgentLoop struct {
	hub                     *chat.Hub
	provider                providers.LLMProvider
	tools                   *tools.Registry
	sessions                *session.SessionManager
	context                 *ContextBuilder
	memory                  *memory.MemoryStore
	model                   string
	maxIterations           int
	running                 bool
	suppressTerminalNotices int32
	taskState               *tools.TaskState
	operatorSetStepHook     func(jobID string, stepID string) (string, error)
	mcpClients              []*mcp.Client
	enableToolActivity      bool
}

// NewAgentLoop creates a new AgentLoop with the given provider.
func NewAgentLoop(b *chat.Hub, provider providers.LLMProvider, model string, maxIterations int, workspace string, scheduler *cron.Scheduler, mcpServers ...map[string]config.MCPServerConfig) *AgentLoop {
	if model == "" {
		model = provider.GetDefaultModel()
	}
	if workspace == "" {
		workspace = "."
	}

	taskState := tools.NewTaskState()

	reg := tools.NewRegistry()
	reg.SetGuard(missioncontrol.NewDefaultToolGuard())
	reg.SetAuditEmitter(taskState)
	reg.Register(tools.NewMessageTool(b))
	reg.Register(tools.NewFrankZohoSendEmailTool())
	reg.Register(tools.NewFrankZohoManageReplyWorkItemToolWithState(taskState))

	// Open an os.Root anchored at the workspace for kernel-enforced sandboxing.
	root, err := os.OpenRoot(workspace)
	if err != nil {
		log.Fatalf("failed to open workspace root %q: %v", workspace, err)
	}

	fsTool, err := tools.NewFilesystemToolWithState(workspace, taskState)
	if err != nil {
		log.Fatalf("failed to create filesystem tool: %v", err)
	}
	reg.Register(fsTool)

	reg.Register(tools.NewExecToolWithWorkspaceAndState(60, workspace, taskState))
	reg.Register(tools.NewWebTool())
	reg.Register(tools.NewWebSearchTool())
	if scheduler != nil {
		reg.Register(tools.NewCronTool(scheduler))
	}

	sm := session.NewSessionManager(workspace)
	if err := sm.LoadAll(); err != nil {
		log.Fatalf("failed to load sessions: %v", err)
	}
	ctx := NewContextBuilder(workspace, memory.NewLLMRanker(provider, model), 5)
	mem := memory.NewMemoryStoreWithWorkspace(workspace, 100)

	reg.Register(tools.NewWriteMemoryTool(mem))
	reg.Register(tools.NewListMemoryTool(mem))
	reg.Register(tools.NewReadMemoryTool(mem))
	reg.Register(tools.NewEditMemoryTool(mem))
	reg.Register(tools.NewDeleteMemoryTool(mem))

	skillMgr := tools.NewSkillManager(root)
	reg.Register(tools.NewCreateSkillTool(skillMgr))
	reg.Register(tools.NewListSkillsTool(skillMgr))
	reg.Register(tools.NewReadSkillTool(skillMgr))
	reg.Register(tools.NewDeleteSkillTool(skillMgr))

	var configuredMCPServers map[string]config.MCPServerConfig
	if len(mcpServers) > 0 {
		configuredMCPServers = mcpServers[0]
	}

	var mcpClients []*mcp.Client
	for name, cfg := range configuredMCPServers {
		var client *mcp.Client
		switch {
		case cfg.Command != "":
			client, err = mcp.NewStdioClient(name, cfg.Command, cfg.Args)
		case cfg.URL != "":
			client, err = mcp.NewHTTPClient(name, cfg.URL, cfg.Headers)
		default:
			log.Printf("MCP server %q: no command or url configured, skipping", name)
			continue
		}
		if err != nil {
			log.Printf("MCP server %q: failed to connect: %s", name, summarizeMCPConnectError(err))
			continue
		}
		mcpClients = append(mcpClients, client)
		for _, tool := range client.Tools() {
			reg.Register(tools.NewMCPTool(client, name, tool))
		}
		log.Printf("MCP server %q: registered %d tools", name, len(client.Tools()))
	}

	return &AgentLoop{
		hub:                b,
		provider:           provider,
		tools:              reg,
		sessions:           sm,
		context:            ctx,
		memory:             mem,
		model:              model,
		maxIterations:      maxIterations,
		taskState:          taskState,
		mcpClients:         mcpClients,
		enableToolActivity: true,
	}
}

func (a *AgentLoop) ActivateMissionStep(job missioncontrol.Job, stepID string) error {
	if a == nil || a.taskState == nil {
		return nil
	}
	return a.taskState.ActivateStep(job, stepID)
}

func (a *AgentLoop) ClearMissionStep() {
	if a == nil || a.taskState == nil {
		return
	}
	a.taskState.ClearExecutionContext()
}

func (a *AgentLoop) ActiveMissionStep() (missioncontrol.ExecutionContext, bool) {
	if a == nil || a.taskState == nil {
		return missioncontrol.ExecutionContext{}, false
	}
	return a.taskState.ExecutionContext()
}

func (a *AgentLoop) MissionRuntimeState() (missioncontrol.JobRuntimeState, bool) {
	if a == nil || a.taskState == nil {
		return missioncontrol.JobRuntimeState{}, false
	}
	return a.taskState.MissionRuntimeState()
}

func (a *AgentLoop) MissionRuntimeControl() (missioncontrol.RuntimeControlContext, bool) {
	if a == nil || a.taskState == nil {
		return missioncontrol.RuntimeControlContext{}, false
	}
	return a.taskState.MissionRuntimeControl()
}

func (a *AgentLoop) GovernedSkillStatus() missioncontrol.GovernedSkillSelectionStatus {
	if a == nil || a.context == nil || a.context.skillsLoader == nil {
		return missioncontrol.GovernedSkillSelectionStatus{}
	}
	var selected []string
	if ec, ok := a.ActiveMissionStep(); ok {
		selected = missioncontrol.EffectiveSelectedSkills(ec.Job, ec.Step)
	}
	report, err := a.context.skillsLoader.LoadSelected(selected, missioncontrol.SkillActivationScopeMissionStepPrompt)
	if err != nil {
		return missioncontrol.GovernedSkillSelectionStatus{
			Root:     a.context.skillsLoader.SkillsRoot(),
			Scope:    missioncontrol.SkillActivationScopeMissionStepPrompt,
			Selected: selected,
			Skipped: []missioncontrol.GovernedSkillSkippedStatus{{
				Reason: "skill_loader_error",
			}},
		}
	}
	return report.Status()
}

func (a *AgentLoop) ResumeMissionRuntime(job missioncontrol.Job, runtimeState missioncontrol.JobRuntimeState, control *missioncontrol.RuntimeControlContext, resumeApproved bool) error {
	if a == nil || a.taskState == nil {
		return nil
	}
	return a.taskState.ResumeRuntime(job, runtimeState, control, resumeApproved)
}

func (a *AgentLoop) HydrateMissionRuntimeControl(job missioncontrol.Job, runtimeState missioncontrol.JobRuntimeState, control *missioncontrol.RuntimeControlContext) error {
	if a == nil || a.taskState == nil {
		return nil
	}
	return a.taskState.HydrateRuntimeControl(job, runtimeState, control)
}

func (a *AgentLoop) SetMissionRuntimeChangeHook(hook func()) {
	if a == nil || a.taskState == nil {
		return
	}
	a.taskState.SetRuntimeChangeHook(a.composeMissionRuntimeChangeHook(hook))
}

func (a *AgentLoop) SetMissionRuntimePersistHook(hook func(*missioncontrol.Job, missioncontrol.JobRuntimeState, *missioncontrol.RuntimeControlContext) error) {
	if a == nil || a.taskState == nil {
		return
	}
	a.taskState.SetRuntimePersistHook(hook)
}

func (a *AgentLoop) SetMissionRuntimeProjectionHook(hook func(*missioncontrol.Job, missioncontrol.JobRuntimeState, *missioncontrol.RuntimeControlContext) error) {
	if a == nil || a.taskState == nil {
		return
	}
	a.taskState.SetRuntimeProjectionHook(hook)
}

func (a *AgentLoop) SetMissionStoreRoot(root string) {
	if a == nil || a.taskState == nil {
		return
	}
	a.taskState.SetMissionStoreRoot(root)
}

func (a *AgentLoop) SetMissionRequired(required bool) {
	if a == nil || a.tools == nil {
		return
	}
	a.tools.SetMissionRequired(required)
}

func (a *AgentLoop) SetOperatorSetStepHook(hook func(jobID string, stepID string) (string, error)) {
	if a == nil {
		return
	}
	a.operatorSetStepHook = hook
}

func (a *AgentLoop) MissionRequired() bool {
	if a == nil || a.tools == nil {
		return false
	}
	return a.tools.MissionRequired()
}

// SetToolActivityIndicator controls whether the feedback of tool progress
func (a *AgentLoop) SetToolActivityIndicator(enabled bool) {
	a.enableToolActivity = enabled
}

// Close shuts down all MCP server connections.
func (a *AgentLoop) Close() {
	for _, c := range a.mcpClients {
		_ = c.Close()
	}
}

// Run starts processing inbound messages. This is a blocking call until context is canceled.
func (a *AgentLoop) Run(ctx context.Context) {
	a.running = true
	log.Println("Agent loop started")
	checkInTicker := time.NewTicker(missionCheckInInterval)
	defer checkInTicker.Stop()

	for a.running {
		select {
		case <-ctx.Done():
			log.Println("Agent loop received shutdown signal")
			a.running = false
			return
		case now := <-checkInTicker.C:
			a.maybeEmitPeriodicMissionNotifications(now)

		case msg, ok := <-a.hub.In:
			if !ok {
				log.Println("Inbound channel closed, stopping agent loop")
				a.running = false
				return
			}

			log.Printf("Processing message from %s:%s\n", msg.Channel, msg.SenderID)

			if a.taskState != nil {
				a.taskState.BeginTask(fmt.Sprintf("%s:%s:%d", msg.Channel, msg.ChatID, time.Now().UnixNano()))
				a.taskState.SetOperatorSession(msg.Channel, msg.ChatID)
				if handled, content, err := a.processOperatorCommand(msg.Content); handled {
					if err != nil {
						content = missioncontrol.SurfacedValidationErrorString(err)
					}
					if !isSystemChannel(msg.Channel) {
						sess := a.sessions.GetOrCreate(msg.Channel + ":" + msg.ChatID)
						sess.AddMessage("user", msg.Content)
						sess.AddMessage("assistant", content)
						if saveErr := a.sessions.Save(sess); saveErr != nil {
							log.Printf("error saving session: %v", saveErr)
						}
					}
					sendAgentReply(a.hub, msg.Channel, msg.ChatID, content)
					continue
				}
				if handled, content, err := a.taskState.ApplyNaturalApprovalDecision(msg.Content); handled {
					if err != nil {
						content = missioncontrol.SurfacedValidationErrorString(err)
					}
					if !isSystemChannel(msg.Channel) {
						sess := a.sessions.GetOrCreate(msg.Channel + ":" + msg.ChatID)
						sess.AddMessage("user", msg.Content)
						sess.AddMessage("assistant", content)
						if saveErr := a.sessions.Save(sess); saveErr != nil {
							log.Printf("error saving session: %v", saveErr)
						}
					}
					sendAgentReply(a.hub, msg.Channel, msg.ChatID, content)
					continue
				}
				if inputKind, err := a.taskState.ApplyWaitingUserInput(msg.Content); err != nil {
					log.Printf("mission runtime waiting_user input validation failed: %v", err)
				} else if inputKind != missioncontrol.WaitingUserInputNone {
					log.Printf("mission runtime waiting_user input accepted: kind=%s", inputKind)
				}
			}

			// Quick heuristic: if user asks the agent to remember something explicitly,
			// store it in today's note and reply immediately without calling the LLM.
			trimmed := strings.TrimSpace(msg.Content)
			rememberRe := rememberRE
			if matches := rememberRe.FindStringSubmatch(trimmed); len(matches) == 2 {
				note := matches[1]
				response := "OK, I've remembered that."
				if err := a.memory.AppendToday(note); err != nil {
					log.Printf("error appending to memory: %v", err)
					response = "I couldn't remember that because saving memory failed."
				}

				if !isSystemChannel(msg.Channel) {
					sess := a.sessions.GetOrCreate(msg.Channel + ":" + msg.ChatID)
					sess.AddMessage("user", msg.Content)
					sess.AddMessage("assistant", response)
					if err := a.sessions.Save(sess); err != nil {
						log.Printf("error saving session: %v", err)
					}
				}
				sendAgentReply(a.hub, msg.Channel, msg.ChatID, response)
				continue
			}

			if mt := a.tools.Get("message"); mt != nil {
				if mtool, ok := mt.(interface{ SetContext(string, string) }); ok {
					mtool.SetContext(msg.Channel, msg.ChatID)
				}
			}
			if ct := a.tools.Get("cron"); ct != nil {
				if ctool, ok := ct.(interface{ SetContext(string, string) }); ok {
					ctool.SetContext(msg.Channel, msg.ChatID)
				}
			}

			var sess *session.Session
			if isSystemChannel(msg.Channel) {
				sess = &session.Session{Key: msg.Channel + ":" + msg.ChatID}
			} else {
				sess = a.sessions.GetOrCreate(msg.Channel + ":" + msg.ChatID)
			}

			memCtx, _ := a.memory.GetMemoryContext()
			memories := a.memory.Recent(5)
			toolDefs := activeToolDefinitions(a.tools, a.taskState)
			var activeStep *missioncontrol.ExecutionContext
			if a.taskState != nil {
				if ec, ok := a.taskState.ExecutionContext(); ok {
					activeStep = &ec
				}
			}
			messages := a.context.BuildMessages(sess.GetHistory(), msg.Content, msg.Channel, msg.ChatID, memCtx, memories, activeStep, toolDefs)

			iteration := 0
			finalContent := ""
			lastToolResult := ""
			successfulTools := make([]missioncontrol.RuntimeToolCallEvidence, 0)

			for iteration < a.maxIterations {
				iteration++

				resp, err := a.provider.Chat(ctx, messages, toolDefs, a.model)
				if err != nil {
					log.Printf("provider error: %s", summarizeProviderError(err))
					if rateLimitContent, ok := providerRateLimitResponse(err); ok {
						finalContent = rateLimitContent
					} else {
						finalContent = "Sorry, I encountered an error while processing your request."
					}
					break
				}

				if resp.HasToolCalls {
					messages = append(messages, providers.Message{
						Role:      "assistant",
						Content:   resp.Content,
						ToolCalls: resp.ToolCalls,
					})

					for _, tc := range resp.ToolCalls {
						if budgetResponse, blocked := checkActiveBudgetBeforeToolCall(a.taskState); blocked {
							finalContent = budgetResponse
							break
						}

						if a.enableToolActivity {
							sendChannelNotification(a.hub, msg.Channel, msg.ChatID,
								fmt.Sprintf("🤖 Running: %s (%s)", tc.Name, tools.SummarizeToolArguments(tc.Arguments)))
						}

						start := time.Now()
						execCtx := ctx
						if a.taskState != nil {
							if ec, ok := a.taskState.ExecutionContext(); ok {
								execCtx = missioncontrol.WithExecutionContext(ctx, ec)
							}
						}
						var (
							res             string
							err             error
							skipToolExecute bool
						)
						if tc.Name == tools.FrankZohoSendEmailToolName && a.taskState != nil {
							res, skipToolExecute, err = a.taskState.PrepareFrankZohoCampaignSend(tc.Arguments)
						} else if tc.Name == tools.FrankZohoManageReplyWorkItemToolName && a.taskState != nil {
							res, skipToolExecute, err = a.taskState.ManageFrankZohoCampaignReplyWorkItem(tc.Arguments)
						}
						if err == nil && !skipToolExecute {
							res, err = a.tools.Execute(execCtx, tc.Name, tc.Arguments)
						}
						elapsed := time.Since(start).Round(time.Millisecond)

						if err == nil && tc.Name == tools.FrankZohoSendEmailToolName && a.taskState != nil && !skipToolExecute {
							if persistErr := a.taskState.RecordFrankZohoCampaignSend(tc.Arguments, res); persistErr != nil {
								err = persistErr
							}
						}
						if err != nil {
							if tc.Name == tools.FrankZohoSendEmailToolName && a.taskState != nil {
								if persistErr := a.taskState.RecordFrankZohoCampaignSendFailure(tc.Arguments, err); persistErr != nil {
									err = persistErr
								}
							}
							if budgetResponse, blocked := recordFailedToolAction(a.taskState, tc.Name, err); blocked {
								finalContent = budgetResponse
								break
							}
							surfacedToolErr := tools.SurfaceToolExecutionError(tc.Name, err)
							if a.enableToolActivity {
								sendChannelNotification(a.hub, msg.Channel, msg.ChatID,
									fmt.Sprintf("📢 %s failed (%s): %s", tc.Name, elapsed, surfacedToolErr))
							}
							res = "(tool error) " + surfacedToolErr
						} else {
							successfulTools = append(successfulTools, missioncontrol.RuntimeToolCallEvidence{
								ToolName:  tc.Name,
								Arguments: cloneToolArguments(tc.Arguments),
								Result:    res,
							})
							if budgetResponse, blocked := recordOwnerFacingMessage(a.taskState, tc.Name); blocked {
								finalContent = budgetResponse
								break
							}
							if a.enableToolActivity {
								sendChannelNotification(a.hub, msg.Channel, msg.ChatID,
									fmt.Sprintf("📢 %s done (%s)", tc.Name, elapsed))
							}
						}

						lastToolResult = res
						messages = append(messages, providers.Message{
							Role:       "tool",
							Content:    res,
							ToolCallID: tc.ID,
						})
					}
					if finalContent != "" {
						break
					}
					continue
				}

				finalContent = resp.Content
				break
			}

			if finalContent == "" && lastToolResult != "" {
				finalContent = lastToolResult
			} else if finalContent == "" {
				finalContent = "I've completed processing but have no response to give."
			}
			finalContent = a.completeMissionStepOutput(finalContent, successfulTools)

			if !isSystemChannel(msg.Channel) {
				sess.AddMessage("user", msg.Content)
				sess.AddMessage("assistant", finalContent)
				if err := a.sessions.Save(sess); err != nil {
					log.Printf("error saving session: %v", err)
				}
			}

			sendAgentReply(a.hub, msg.Channel, msg.ChatID, finalContent)

		default:
			time.Sleep(100 * time.Millisecond)
		}
	}
}

// ProcessDirect sends a message directly to the provider and returns the response.
// It supports tool calling - if the model requests tools, they will be executed.
func (a *AgentLoop) ProcessDirect(content string, timeout time.Duration) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if parsed, ok := parseStaticOperatorCommand(content); ok && parsed.Kind == operatorCommandHotUpdateHelp {
		return hotUpdateOperatorHelpText(), nil
	}

	if a.taskState != nil {
		a.taskState.BeginTask(fmt.Sprintf("cli:direct:%d", time.Now().UnixNano()))
		a.taskState.SetOperatorSession("cli", "direct")
		if handled, response, err := a.processOperatorCommand(content); handled {
			return response, missioncontrol.SurfaceValidationError(err)
		}
		if handled, response, err := a.taskState.ApplyNaturalApprovalDecision(content); handled {
			return response, missioncontrol.SurfaceValidationError(err)
		}
		if inputKind, err := a.taskState.ApplyWaitingUserInput(content); err != nil {
			log.Printf("mission runtime waiting_user input validation failed: %v", err)
		} else if inputKind != missioncontrol.WaitingUserInputNone {
			log.Printf("mission runtime waiting_user input accepted: kind=%s", inputKind)
		}
	}

	if mt := a.tools.Get("message"); mt != nil {
		if mtool, ok := mt.(interface{ SetContext(string, string) }); ok {
			mtool.SetContext("cli", "direct")
		}
	}
	if ct := a.tools.Get("cron"); ct != nil {
		if ctool, ok := ct.(interface{ SetContext(string, string) }); ok {
			ctool.SetContext("cli", "direct")
		}
	}

	memCtx, _ := a.memory.GetMemoryContext()
	memories := a.memory.Recent(5)
	toolDefs := activeToolDefinitions(a.tools, a.taskState)
	var activeStep *missioncontrol.ExecutionContext
	if a.taskState != nil {
		if ec, ok := a.taskState.ExecutionContext(); ok {
			activeStep = &ec
		}
	}
	messages := a.context.BuildMessages(nil, content, "cli", "direct", memCtx, memories, activeStep, toolDefs)

	var lastToolResult string
	successfulTools := make([]missioncontrol.RuntimeToolCallEvidence, 0)
	for iteration := 0; iteration < a.maxIterations; iteration++ {
		resp, err := a.provider.Chat(ctx, messages, toolDefs, a.model)
		if err != nil {
			return "", fmt.Errorf("%s", summarizeProviderError(err))
		}

		if !resp.HasToolCalls {
			if resp.Content != "" {
				return a.completeMissionStepOutput(resp.Content, successfulTools), nil
			}
			if lastToolResult != "" {
				return a.completeMissionStepOutput(lastToolResult, successfulTools), nil
			}
			return resp.Content, nil
		}

		messages = append(messages, providers.Message{
			Role:      "assistant",
			Content:   resp.Content,
			ToolCalls: resp.ToolCalls,
		})

		for _, tc := range resp.ToolCalls {
			if budgetResponse, blocked := checkActiveBudgetBeforeToolCall(a.taskState); blocked {
				return budgetResponse, nil
			}

			execCtx := ctx
			if a.taskState != nil {
				if ec, ok := a.taskState.ExecutionContext(); ok {
					execCtx = missioncontrol.WithExecutionContext(ctx, ec)
				}
			}
			var (
				result          string
				err             error
				skipToolExecute bool
			)
			if tc.Name == tools.FrankZohoSendEmailToolName && a.taskState != nil {
				result, skipToolExecute, err = a.taskState.PrepareFrankZohoCampaignSend(tc.Arguments)
			} else if tc.Name == tools.FrankZohoManageReplyWorkItemToolName && a.taskState != nil {
				result, skipToolExecute, err = a.taskState.ManageFrankZohoCampaignReplyWorkItem(tc.Arguments)
			}
			if err == nil && !skipToolExecute {
				result, err = a.tools.Execute(execCtx, tc.Name, tc.Arguments)
			}
			if err == nil && tc.Name == tools.FrankZohoSendEmailToolName && a.taskState != nil && !skipToolExecute {
				if persistErr := a.taskState.RecordFrankZohoCampaignSend(tc.Arguments, result); persistErr != nil {
					err = persistErr
				}
			}
			if err != nil {
				if tc.Name == tools.FrankZohoSendEmailToolName && a.taskState != nil {
					if persistErr := a.taskState.RecordFrankZohoCampaignSendFailure(tc.Arguments, err); persistErr != nil {
						err = persistErr
					}
				}
				if budgetResponse, blocked := recordFailedToolAction(a.taskState, tc.Name, err); blocked {
					return budgetResponse, nil
				}
				result = "(tool error) " + tools.SurfaceToolExecutionError(tc.Name, err)
			} else {
				successfulTools = append(successfulTools, missioncontrol.RuntimeToolCallEvidence{
					ToolName:  tc.Name,
					Arguments: cloneToolArguments(tc.Arguments),
					Result:    result,
				})
				if budgetResponse, blocked := recordOwnerFacingMessage(a.taskState, tc.Name); blocked {
					return budgetResponse, nil
				}
			}
			lastToolResult = result
			messages = append(messages, providers.Message{
				Role:       "tool",
				Content:    result,
				ToolCallID: tc.ID,
			})
		}
	}

	return "Max iterations reached without final response", nil
}

func hotUpdateOperatorHelpText() string {
	return strings.TrimSpace(`Frank V4 hot-update direct-command help

Runbook:
  docs/HOT_UPDATE_OPERATOR_RUNBOOK.md

Help aliases:
` + formatStaticOperatorCommandAliases(staticHotUpdateHelpAliases()) + `

Inspect state:
  STATUS <job_id>
  v4_summary gives compact V4 lifecycle state; detailed identity sections remain audit authority.

Eligible-only hot-update path:
  HOT_UPDATE_GATE_FROM_DECISION <job_id> <promotion_decision_id>
  HOT_UPDATE_GATE_RECORD <job_id> <hot_update_id> <candidate_pack_id>
  HOT_UPDATE_EXECUTION_READY <job_id> <hot_update_id> <ttl_seconds> [reason...]
  HOT_UPDATE_GATE_PHASE <job_id> <hot_update_id> validated
  HOT_UPDATE_GATE_PHASE <job_id> <hot_update_id> staged
  HOT_UPDATE_GATE_EXECUTE <job_id> <hot_update_id>
  HOT_UPDATE_GATE_RELOAD <job_id> <hot_update_id>
  HOT_UPDATE_GATE_FAIL <job_id> <hot_update_id> [reason...]
  HOT_UPDATE_OUTCOME_CREATE <job_id> <hot_update_id>
  HOT_UPDATE_PROMOTION_CREATE <job_id> <outcome_id>
  HOT_UPDATE_LKG_RECERTIFY <job_id> <promotion_id>

Canary-required hot-update path:
  HOT_UPDATE_CANARY_REQUIREMENT_CREATE <job_id> <result_id>
  HOT_UPDATE_CANARY_EVIDENCE_CREATE <job_id> <canary_requirement_id> <evidence_state> <observed_at> [reason...]
  HOT_UPDATE_CANARY_SATISFACTION_AUTHORITY_CREATE <job_id> <canary_requirement_id>
  HOT_UPDATE_OWNER_APPROVAL_REQUEST_CREATE <job_id> <canary_satisfaction_authority_id>
  HOT_UPDATE_OWNER_APPROVAL_DECISION_CREATE <job_id> <owner_approval_request_id> <decision> [reason...]
  HOT_UPDATE_CANARY_GATE_CREATE <job_id> <canary_satisfaction_authority_id> [owner_approval_decision_id]
  HOT_UPDATE_GATE_PHASE <job_id> <hot_update_id> validated
  HOT_UPDATE_GATE_PHASE <job_id> <hot_update_id> staged
  HOT_UPDATE_GATE_EXECUTE <job_id> <hot_update_id>
  HOT_UPDATE_GATE_RELOAD <job_id> <hot_update_id>
  HOT_UPDATE_OUTCOME_CREATE <job_id> <hot_update_id>
  HOT_UPDATE_PROMOTION_CREATE <job_id> <outcome_id>

Rollback, rollback-apply, and LKG recovery:
  ROLLBACK_RECORD <job_id> <promotion_id> <rollback_id>
  ROLLBACK_APPLY_RECORD <job_id> <rollback_id> <apply_id>
  ROLLBACK_APPLY_PHASE <job_id> <apply_id> <phase>
  ROLLBACK_APPLY_EXECUTE <job_id> <apply_id>
  ROLLBACK_APPLY_RELOAD <job_id> <apply_id>
  ROLLBACK_APPLY_FAIL <job_id> <apply_id> [reason...]
  HOT_UPDATE_LKG_RECERTIFY <job_id> <promotion_id>

Notes:
  CandidatePromotionDecisionRecord remains eligible-only.
  Canary owner approval uses exact granted or rejected decisions; natural-language yes/no/approve/deny aliases are not canary owner approval authority.
  Canary-derived gates are guarded before phase advancement, pointer switch, and reload/apply.
  Outcomes and promotions preserve canary_ref and approval_ref audit lineage.
  Rollback, rollback-apply, and LKG recertification remain generic recovery flows.`)
}

func (a *AgentLoop) processOperatorCommand(content string) (bool, string, error) {
	if a == nil || a.taskState == nil {
		return false, "", nil
	}

	trimmed := strings.TrimSpace(content)

	if parsed, ok := parseStaticOperatorCommand(trimmed); ok && parsed.Kind == operatorCommandStatus {
		response, err := a.taskState.OperatorStatus(parsed.JobID)
		return true, response, err
	}
	if parsed, ok := parseStaticOperatorCommand(trimmed); ok {
		switch parsed.Kind {
		case operatorCommandPause, operatorCommandResume, operatorCommandAbort:
			return a.applyRuntimeOperatorCommand(parsed)
		case operatorCommandSetStep:
			return a.applySetStepOperatorCommand(parsed)
		}
	}

	inspectMatches := inspectCommandRE.FindStringSubmatch(trimmed)
	if len(inspectMatches) == 4 {
		response, err := a.taskState.OperatorInspect(inspectMatches[2], inspectMatches[3])
		return true, response, err
	}

	revokeMatches := revokeApprovalCommandRE.FindStringSubmatch(trimmed)
	if len(revokeMatches) == 4 {
		jobID := revokeMatches[2]
		stepID := revokeMatches[3]
		if err := a.taskState.RevokeApproval(jobID, stepID); err != nil {
			return true, "", err
		}
		if budgetResponse, blocked := recordOperatorRevokeApprovalAcknowledgement(a.taskState); blocked {
			return true, budgetResponse, nil
		}
		return true, fmt.Sprintf("Revoked approval job=%s step=%s.", jobID, stepID), nil
	}

	rollbackMatches := rollbackRecordCommandRE.FindStringSubmatch(trimmed)
	if len(rollbackMatches) == 5 {
		jobID := rollbackMatches[2]
		promotionID := rollbackMatches[3]
		rollbackID := rollbackMatches[4]
		if err := a.taskState.RecordRollbackFromPromotion(jobID, promotionID, rollbackID); err != nil {
			return true, "", err
		}
		return true, fmt.Sprintf("Recorded rollback proposal job=%s promotion=%s rollback=%s.", jobID, promotionID, rollbackID), nil
	}

	hotUpdateGateMatches := hotUpdateGateRecordCommandRE.FindStringSubmatch(trimmed)
	if len(hotUpdateGateMatches) == 5 {
		jobID := hotUpdateGateMatches[2]
		hotUpdateID := hotUpdateGateMatches[3]
		candidatePackID := hotUpdateGateMatches[4]
		created, err := a.taskState.EnsureHotUpdateGateRecord(jobID, hotUpdateID, candidatePackID)
		if err != nil {
			return true, "", err
		}
		if created {
			return true, fmt.Sprintf("Recorded hot-update gate job=%s hot_update=%s candidate_pack=%s.", jobID, hotUpdateID, candidatePackID), nil
		}
		return true, fmt.Sprintf("Selected hot-update gate job=%s hot_update=%s candidate_pack=%s.", jobID, hotUpdateID, candidatePackID), nil
	}

	hotUpdateCanaryRequirementCreateMatches := hotUpdateCanaryRequirementCreateCommandRE.FindStringSubmatch(trimmed)
	if len(hotUpdateCanaryRequirementCreateMatches) == 4 {
		jobID := hotUpdateCanaryRequirementCreateMatches[2]
		resultID := hotUpdateCanaryRequirementCreateMatches[3]
		record, changed, err := a.taskState.CreateHotUpdateCanaryRequirementFromCandidateResult(jobID, resultID)
		if err != nil {
			return true, "", err
		}
		if changed {
			return true, fmt.Sprintf("Created hot-update canary requirement job=%s result=%s canary_requirement=%s owner_approval_required=%t.", jobID, resultID, record.CanaryRequirementID, record.OwnerApprovalRequired), nil
		}
		return true, fmt.Sprintf("Selected hot-update canary requirement job=%s result=%s canary_requirement=%s owner_approval_required=%t.", jobID, resultID, record.CanaryRequirementID, record.OwnerApprovalRequired), nil
	}
	if isMalformedHotUpdateCanaryRequirementCreateCommand(trimmed) {
		return true, "", fmt.Errorf("HOT_UPDATE_CANARY_REQUIREMENT_CREATE requires job_id and result_id")
	}

	hotUpdateCanaryEvidenceCreateMatches := hotUpdateCanaryEvidenceCreateCommandRE.FindStringSubmatch(trimmed)
	if len(hotUpdateCanaryEvidenceCreateMatches) == 7 {
		jobID := hotUpdateCanaryEvidenceCreateMatches[2]
		canaryRequirementID := hotUpdateCanaryEvidenceCreateMatches[3]
		evidenceState := missioncontrol.HotUpdateCanaryEvidenceState(hotUpdateCanaryEvidenceCreateMatches[4])
		observedAt, err := parseHotUpdateCanaryEvidenceObservedAt(hotUpdateCanaryEvidenceCreateMatches[5])
		if err != nil {
			return true, "", err
		}
		reason := hotUpdateCanaryEvidenceCreateMatches[6]
		record, changed, err := a.taskState.CreateHotUpdateCanaryEvidenceFromRequirement(jobID, canaryRequirementID, evidenceState, observedAt, reason)
		if err != nil {
			return true, "", err
		}
		if changed {
			return true, fmt.Sprintf("Created hot-update canary evidence job=%s canary_requirement=%s canary_evidence=%s evidence_state=%s passed=%t.", jobID, canaryRequirementID, record.CanaryEvidenceID, record.EvidenceState, record.Passed), nil
		}
		return true, fmt.Sprintf("Selected hot-update canary evidence job=%s canary_requirement=%s canary_evidence=%s evidence_state=%s passed=%t.", jobID, canaryRequirementID, record.CanaryEvidenceID, record.EvidenceState, record.Passed), nil
	}
	if isMalformedHotUpdateCanaryEvidenceCreateCommand(trimmed) {
		return true, "", fmt.Errorf("HOT_UPDATE_CANARY_EVIDENCE_CREATE requires job_id, canary_requirement_id, evidence_state, observed_at, and optional reason")
	}

	hotUpdateCanarySatisfactionAuthorityCreateMatches := hotUpdateCanarySatisfactionAuthorityCreateCommandRE.FindStringSubmatch(trimmed)
	if len(hotUpdateCanarySatisfactionAuthorityCreateMatches) == 4 {
		jobID := hotUpdateCanarySatisfactionAuthorityCreateMatches[2]
		canaryRequirementID := hotUpdateCanarySatisfactionAuthorityCreateMatches[3]
		record, changed, err := a.taskState.CreateHotUpdateCanarySatisfactionAuthorityFromRequirement(jobID, canaryRequirementID)
		if err != nil {
			return true, "", err
		}
		if changed {
			return true, fmt.Sprintf("Created hot-update canary satisfaction authority job=%s canary_requirement=%s authority=%s authority_state=%s owner_approval_required=%t.", jobID, canaryRequirementID, record.CanarySatisfactionAuthorityID, record.State, record.OwnerApprovalRequired), nil
		}
		return true, fmt.Sprintf("Selected hot-update canary satisfaction authority job=%s canary_requirement=%s authority=%s authority_state=%s owner_approval_required=%t.", jobID, canaryRequirementID, record.CanarySatisfactionAuthorityID, record.State, record.OwnerApprovalRequired), nil
	}
	if isMalformedHotUpdateCanarySatisfactionAuthorityCreateCommand(trimmed) {
		return true, "", fmt.Errorf("HOT_UPDATE_CANARY_SATISFACTION_AUTHORITY_CREATE requires job_id and canary_requirement_id")
	}

	hotUpdateOwnerApprovalRequestCreateMatches := hotUpdateOwnerApprovalRequestCreateCommandRE.FindStringSubmatch(trimmed)
	if len(hotUpdateOwnerApprovalRequestCreateMatches) == 4 {
		jobID := hotUpdateOwnerApprovalRequestCreateMatches[2]
		canarySatisfactionAuthorityID := hotUpdateOwnerApprovalRequestCreateMatches[3]
		record, changed, err := a.taskState.CreateHotUpdateOwnerApprovalRequestFromCanarySatisfactionAuthority(jobID, canarySatisfactionAuthorityID)
		if err != nil {
			return true, "", err
		}
		if changed {
			return true, fmt.Sprintf("Created hot-update owner approval request job=%s canary_satisfaction_authority=%s owner_approval_request=%s request_state=%s authority_state=%s owner_approval_required=%t.", jobID, canarySatisfactionAuthorityID, record.OwnerApprovalRequestID, record.State, record.AuthorityState, record.OwnerApprovalRequired), nil
		}
		return true, fmt.Sprintf("Selected hot-update owner approval request job=%s canary_satisfaction_authority=%s owner_approval_request=%s request_state=%s authority_state=%s owner_approval_required=%t.", jobID, canarySatisfactionAuthorityID, record.OwnerApprovalRequestID, record.State, record.AuthorityState, record.OwnerApprovalRequired), nil
	}
	if isMalformedHotUpdateOwnerApprovalRequestCreateCommand(trimmed) {
		return true, "", fmt.Errorf("HOT_UPDATE_OWNER_APPROVAL_REQUEST_CREATE requires job_id and canary_satisfaction_authority_id")
	}

	hotUpdateOwnerApprovalDecisionCreateMatches := hotUpdateOwnerApprovalDecisionCreateCommandRE.FindStringSubmatch(trimmed)
	if len(hotUpdateOwnerApprovalDecisionCreateMatches) == 6 {
		jobID := hotUpdateOwnerApprovalDecisionCreateMatches[2]
		ownerApprovalRequestID := hotUpdateOwnerApprovalDecisionCreateMatches[3]
		decision := missioncontrol.HotUpdateOwnerApprovalDecision(hotUpdateOwnerApprovalDecisionCreateMatches[4])
		reason := hotUpdateOwnerApprovalDecisionCreateMatches[5]
		record, changed, err := a.taskState.CreateHotUpdateOwnerApprovalDecisionFromRequest(jobID, ownerApprovalRequestID, decision, reason)
		if err != nil {
			return true, "", err
		}
		if changed {
			return true, fmt.Sprintf("Created hot-update owner approval decision job=%s owner_approval_request=%s owner_approval_decision=%s decision=%s.", jobID, ownerApprovalRequestID, record.OwnerApprovalDecisionID, record.Decision), nil
		}
		return true, fmt.Sprintf("Selected hot-update owner approval decision job=%s owner_approval_request=%s owner_approval_decision=%s decision=%s.", jobID, ownerApprovalRequestID, record.OwnerApprovalDecisionID, record.Decision), nil
	}
	if isMalformedHotUpdateOwnerApprovalDecisionCreateCommand(trimmed) {
		return true, "", fmt.Errorf("HOT_UPDATE_OWNER_APPROVAL_DECISION_CREATE requires job_id, owner_approval_request_id, decision, and optional reason")
	}

	hotUpdateCanaryGateCreateMatches := hotUpdateCanaryGateCreateCommandRE.FindStringSubmatch(trimmed)
	if len(hotUpdateCanaryGateCreateMatches) == 5 {
		jobID := hotUpdateCanaryGateCreateMatches[2]
		canarySatisfactionAuthorityID := hotUpdateCanaryGateCreateMatches[3]
		ownerApprovalDecisionID := hotUpdateCanaryGateCreateMatches[4]
		record, changed, err := a.taskState.CreateHotUpdateGateFromCanarySatisfactionAuthority(jobID, canarySatisfactionAuthorityID, ownerApprovalDecisionID)
		if err != nil {
			return true, "", err
		}
		if changed {
			return true, fmt.Sprintf("Created hot-update canary gate job=%s canary_satisfaction_authority=%s hot_update=%s canary_ref=%s approval_ref=%s.", jobID, canarySatisfactionAuthorityID, record.HotUpdateID, record.CanaryRef, record.ApprovalRef), nil
		}
		return true, fmt.Sprintf("Selected hot-update canary gate job=%s canary_satisfaction_authority=%s hot_update=%s canary_ref=%s approval_ref=%s.", jobID, canarySatisfactionAuthorityID, record.HotUpdateID, record.CanaryRef, record.ApprovalRef), nil
	}
	if isMalformedHotUpdateCanaryGateCreateCommand(trimmed) {
		return true, "", fmt.Errorf("HOT_UPDATE_CANARY_GATE_CREATE requires job_id, canary_satisfaction_authority_id, and optional owner_approval_decision_id")
	}

	hotUpdateGateFromDecisionMatches := hotUpdateGateFromDecisionCommandRE.FindStringSubmatch(trimmed)
	if len(hotUpdateGateFromDecisionMatches) == 4 {
		jobID := hotUpdateGateFromDecisionMatches[2]
		promotionDecisionID := hotUpdateGateFromDecisionMatches[3]
		changed, err := a.taskState.CreateHotUpdateGateFromCandidatePromotionDecision(jobID, promotionDecisionID)
		if err != nil {
			return true, "", err
		}
		hotUpdateID := "hot-update-" + strings.TrimSpace(promotionDecisionID)
		if changed {
			return true, fmt.Sprintf("Created hot-update gate from decision job=%s promotion_decision=%s hot_update=%s.", jobID, promotionDecisionID, hotUpdateID), nil
		}
		return true, fmt.Sprintf("Selected hot-update gate from decision job=%s promotion_decision=%s hot_update=%s.", jobID, promotionDecisionID, hotUpdateID), nil
	}
	if isMalformedHotUpdateGateFromDecisionCommand(trimmed) {
		return true, "", fmt.Errorf("HOT_UPDATE_GATE_FROM_DECISION requires job_id and promotion_decision_id")
	}

	hotUpdateGatePhaseMatches := hotUpdateGatePhaseCommandRE.FindStringSubmatch(trimmed)
	if len(hotUpdateGatePhaseMatches) == 5 {
		jobID := hotUpdateGatePhaseMatches[2]
		hotUpdateID := hotUpdateGatePhaseMatches[3]
		phase := hotUpdateGatePhaseMatches[4]
		changed, err := a.taskState.AdvanceHotUpdateGatePhase(jobID, hotUpdateID, phase)
		if err != nil {
			return true, "", err
		}
		if changed {
			return true, fmt.Sprintf("Advanced hot-update gate job=%s hot_update=%s phase=%s.", jobID, hotUpdateID, phase), nil
		}
		return true, fmt.Sprintf("Selected hot-update gate job=%s hot_update=%s phase=%s.", jobID, hotUpdateID, phase), nil
	}

	hotUpdateGateExecuteMatches := hotUpdateGateExecuteCommandRE.FindStringSubmatch(trimmed)
	if len(hotUpdateGateExecuteMatches) == 4 {
		jobID := hotUpdateGateExecuteMatches[2]
		hotUpdateID := hotUpdateGateExecuteMatches[3]
		changed, err := a.taskState.ExecuteHotUpdateGatePointerSwitch(jobID, hotUpdateID)
		if err != nil {
			return true, "", err
		}
		if changed {
			return true, fmt.Sprintf("Executed hot-update pointer switch job=%s hot_update=%s.", jobID, hotUpdateID), nil
		}
		return true, fmt.Sprintf("Selected hot-update pointer switch job=%s hot_update=%s.", jobID, hotUpdateID), nil
	}

	hotUpdateGateReloadMatches := hotUpdateGateReloadCommandRE.FindStringSubmatch(trimmed)
	if len(hotUpdateGateReloadMatches) == 4 {
		jobID := hotUpdateGateReloadMatches[2]
		hotUpdateID := hotUpdateGateReloadMatches[3]
		changed, err := a.taskState.ExecuteHotUpdateGateReloadApply(jobID, hotUpdateID)
		if err != nil {
			return true, "", err
		}
		if changed {
			return true, fmt.Sprintf("Executed hot-update reload/apply job=%s hot_update=%s.", jobID, hotUpdateID), nil
		}
		return true, fmt.Sprintf("Selected hot-update reload/apply job=%s hot_update=%s.", jobID, hotUpdateID), nil
	}

	hotUpdateGateFailMatches := hotUpdateGateFailCommandRE.FindStringSubmatch(trimmed)
	if len(hotUpdateGateFailMatches) == 5 {
		jobID := hotUpdateGateFailMatches[2]
		hotUpdateID := hotUpdateGateFailMatches[3]
		reason := hotUpdateGateFailMatches[4]
		changed, err := a.taskState.ResolveHotUpdateGateTerminalFailure(jobID, hotUpdateID, reason)
		if err != nil {
			return true, "", err
		}
		if changed {
			return true, fmt.Sprintf("Resolved hot-update terminal failure job=%s hot_update=%s.", jobID, hotUpdateID), nil
		}
		return true, fmt.Sprintf("Selected hot-update terminal failure job=%s hot_update=%s.", jobID, hotUpdateID), nil
	}

	hotUpdateExecutionReadyMatches := hotUpdateExecutionReadyCommandRE.FindStringSubmatch(trimmed)
	if len(hotUpdateExecutionReadyMatches) == 6 {
		jobID := hotUpdateExecutionReadyMatches[2]
		hotUpdateID := hotUpdateExecutionReadyMatches[3]
		ttlSeconds, err := strconv.Atoi(hotUpdateExecutionReadyMatches[4])
		if err != nil {
			return true, "", fmt.Errorf("HOT_UPDATE_EXECUTION_READY ttl_seconds must be an integer")
		}
		reason := hotUpdateExecutionReadyMatches[5]
		record, changed, err := a.taskState.RecordHotUpdateExecutionReady(jobID, hotUpdateID, ttlSeconds, reason)
		if err != nil {
			return true, "", err
		}
		expiresAt := record.ExpiresAt.UTC().Format(time.RFC3339)
		if changed {
			return true, fmt.Sprintf("Recorded hot-update execution readiness job=%s hot_update=%s expires_at=%s.", jobID, hotUpdateID, expiresAt), nil
		}
		return true, fmt.Sprintf("Selected hot-update execution readiness job=%s hot_update=%s expires_at=%s.", jobID, hotUpdateID, expiresAt), nil
	}
	if isMalformedHotUpdateExecutionReadyCommand(trimmed) {
		return true, "", fmt.Errorf("HOT_UPDATE_EXECUTION_READY requires job_id, hot_update_id, ttl_seconds, and optional reason")
	}

	hotUpdateOutcomeCreateMatches := hotUpdateOutcomeCreateCommandRE.FindStringSubmatch(trimmed)
	if len(hotUpdateOutcomeCreateMatches) == 4 {
		jobID := hotUpdateOutcomeCreateMatches[2]
		hotUpdateID := hotUpdateOutcomeCreateMatches[3]
		changed, err := a.taskState.CreateHotUpdateOutcomeFromTerminalGate(jobID, hotUpdateID)
		if err != nil {
			return true, "", err
		}
		if changed {
			return true, fmt.Sprintf("Created hot-update outcome job=%s hot_update=%s.", jobID, hotUpdateID), nil
		}
		return true, fmt.Sprintf("Selected hot-update outcome job=%s hot_update=%s.", jobID, hotUpdateID), nil
	}

	hotUpdatePromotionCreateMatches := hotUpdatePromotionCreateCommandRE.FindStringSubmatch(trimmed)
	if len(hotUpdatePromotionCreateMatches) == 4 {
		jobID := hotUpdatePromotionCreateMatches[2]
		outcomeID := hotUpdatePromotionCreateMatches[3]
		changed, err := a.taskState.CreatePromotionFromSuccessfulHotUpdateOutcome(jobID, outcomeID)
		if err != nil {
			return true, "", err
		}
		if changed {
			return true, fmt.Sprintf("Created hot-update promotion job=%s outcome=%s.", jobID, outcomeID), nil
		}
		return true, fmt.Sprintf("Selected hot-update promotion job=%s outcome=%s.", jobID, outcomeID), nil
	}

	hotUpdateLKGRecertifyMatches := hotUpdateLKGRecertifyCommandRE.FindStringSubmatch(trimmed)
	if len(hotUpdateLKGRecertifyMatches) == 4 {
		jobID := hotUpdateLKGRecertifyMatches[2]
		promotionID := hotUpdateLKGRecertifyMatches[3]
		changed, err := a.taskState.RecertifyLastKnownGoodFromPromotion(jobID, promotionID)
		if err != nil {
			return true, "", err
		}
		if changed {
			return true, fmt.Sprintf("Recertified hot-update last-known-good job=%s promotion=%s.", jobID, promotionID), nil
		}
		return true, fmt.Sprintf("Selected hot-update last-known-good job=%s promotion=%s.", jobID, promotionID), nil
	}

	rollbackApplyMatches := rollbackApplyRecordCommandRE.FindStringSubmatch(trimmed)
	if len(rollbackApplyMatches) == 5 {
		jobID := rollbackApplyMatches[2]
		rollbackID := rollbackApplyMatches[3]
		applyID := rollbackApplyMatches[4]
		created, err := a.taskState.EnsureRollbackApplyRecord(jobID, rollbackID, applyID)
		if err != nil {
			return true, "", err
		}
		if created {
			return true, fmt.Sprintf("Recorded rollback-apply workflow job=%s rollback=%s apply=%s.", jobID, rollbackID, applyID), nil
		}
		return true, fmt.Sprintf("Selected rollback-apply workflow job=%s rollback=%s apply=%s.", jobID, rollbackID, applyID), nil
	}

	rollbackApplyPhaseMatches := rollbackApplyPhaseCommandRE.FindStringSubmatch(trimmed)
	if len(rollbackApplyPhaseMatches) == 5 {
		jobID := rollbackApplyPhaseMatches[2]
		applyID := rollbackApplyPhaseMatches[3]
		phase := rollbackApplyPhaseMatches[4]
		changed, err := a.taskState.AdvanceRollbackApplyPhase(jobID, applyID, phase)
		if err != nil {
			return true, "", err
		}
		if changed {
			return true, fmt.Sprintf("Advanced rollback-apply workflow job=%s apply=%s phase=%s.", jobID, applyID, phase), nil
		}
		return true, fmt.Sprintf("Selected rollback-apply workflow job=%s apply=%s phase=%s.", jobID, applyID, phase), nil
	}

	rollbackApplyExecuteMatches := rollbackApplyExecuteCommandRE.FindStringSubmatch(trimmed)
	if len(rollbackApplyExecuteMatches) == 4 {
		jobID := rollbackApplyExecuteMatches[2]
		applyID := rollbackApplyExecuteMatches[3]
		changed, err := a.taskState.ExecuteRollbackApplyPointerSwitch(jobID, applyID)
		if err != nil {
			return true, "", err
		}
		if changed {
			return true, fmt.Sprintf("Executed rollback-apply pointer switch job=%s apply=%s.", jobID, applyID), nil
		}
		return true, fmt.Sprintf("Selected rollback-apply pointer switch job=%s apply=%s.", jobID, applyID), nil
	}

	rollbackApplyReloadMatches := rollbackApplyReloadCommandRE.FindStringSubmatch(trimmed)
	if len(rollbackApplyReloadMatches) == 4 {
		jobID := rollbackApplyReloadMatches[2]
		applyID := rollbackApplyReloadMatches[3]
		changed, err := a.taskState.ExecuteRollbackApplyReloadApply(jobID, applyID)
		if err != nil {
			return true, "", err
		}
		if changed {
			return true, fmt.Sprintf("Executed rollback-apply reload/apply job=%s apply=%s.", jobID, applyID), nil
		}
		return true, fmt.Sprintf("Selected rollback-apply reload/apply job=%s apply=%s.", jobID, applyID), nil
	}

	rollbackApplyFailMatches := rollbackApplyFailCommandRE.FindStringSubmatch(trimmed)
	if len(rollbackApplyFailMatches) == 5 {
		jobID := rollbackApplyFailMatches[2]
		applyID := rollbackApplyFailMatches[3]
		reason := rollbackApplyFailMatches[4]
		changed, err := a.taskState.ResolveRollbackApplyTerminalFailure(jobID, applyID, reason)
		if err != nil {
			return true, "", err
		}
		if changed {
			return true, fmt.Sprintf("Resolved rollback-apply terminal failure job=%s apply=%s.", jobID, applyID), nil
		}
		return true, fmt.Sprintf("Selected rollback-apply terminal failure job=%s apply=%s.", jobID, applyID), nil
	}

	if parsed, ok := parseStaticOperatorCommand(trimmed); ok {
		switch parsed.Kind {
		case operatorCommandApprove, operatorCommandDeny:
			return a.applyApprovalOperatorCommand(parsed)
		}
	}

	return false, "", nil
}

func (a *AgentLoop) applyApprovalOperatorCommand(parsed parsedOperatorCommand) (bool, string, error) {
	decision := missioncontrol.ApprovalDecisionApprove
	if parsed.Kind == operatorCommandDeny {
		decision = missioncontrol.ApprovalDecisionDeny
	}
	jobID := parsed.JobID
	stepID := parsed.StepID
	if err := a.taskState.ApplyApprovalDecision(jobID, stepID, decision, missioncontrol.ApprovalGrantedViaOperatorCommand); err != nil {
		return true, "", err
	}

	verb := "Approved"
	if decision == missioncontrol.ApprovalDecisionDeny {
		if budgetResponse, blocked := recordOperatorDenyAcknowledgement(a.taskState); blocked {
			return true, budgetResponse, nil
		}
		verb = "Denied"
	}
	return true, fmt.Sprintf("%s job=%s step=%s.", verb, jobID, stepID), nil
}

func (a *AgentLoop) applySetStepOperatorCommand(parsed parsedOperatorCommand) (bool, string, error) {
	if a.operatorSetStepHook == nil {
		return true, "", missioncontrol.ValidationError{
			Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
			Message: "SET_STEP requires mission step control configuration",
		}
	}
	response, err := a.operatorSetStepHook(parsed.JobID, parsed.StepID)
	if err != nil {
		return true, response, err
	}
	if budgetResponse, blocked := recordOperatorSetStepAcknowledgement(a.taskState); blocked {
		return true, budgetResponse, nil
	}
	return true, response, nil
}

func (a *AgentLoop) applyRuntimeOperatorCommand(parsed parsedOperatorCommand) (bool, string, error) {
	jobID := parsed.JobID
	var err error
	switch parsed.Kind {
	case operatorCommandPause:
		err = a.taskState.PauseRuntime(jobID)
	case operatorCommandResume:
		err = a.taskState.ResumeRuntimeControl(jobID)
	case operatorCommandAbort:
		err = a.taskState.AbortRuntime(jobID)
	default:
		return false, "", nil
	}
	if err != nil {
		return true, "", err
	}
	switch parsed.Kind {
	case operatorCommandPause:
		if budgetResponse, blocked := recordOperatorPauseAcknowledgement(a.taskState); blocked {
			return true, budgetResponse, nil
		}
	case operatorCommandResume:
		if budgetResponse, blocked := recordOperatorResumeAcknowledgement(a.taskState); blocked {
			return true, budgetResponse, nil
		}
	}

	verb := "Resumed"
	switch parsed.Kind {
	case operatorCommandPause:
		verb = "Paused"
	case operatorCommandAbort:
		verb = "Aborted"
	}
	return true, fmt.Sprintf("%s job=%s.", verb, jobID), nil
}

func isMalformedHotUpdateGateFromDecisionCommand(content string) bool {
	fields := strings.Fields(content)
	return len(fields) > 0 && strings.EqualFold(fields[0], "hot_update_gate_from_decision")
}

func isMalformedHotUpdateCanaryRequirementCreateCommand(content string) bool {
	fields := strings.Fields(content)
	return len(fields) > 0 && strings.EqualFold(fields[0], "hot_update_canary_requirement_create")
}

func isMalformedHotUpdateCanaryEvidenceCreateCommand(content string) bool {
	fields := strings.Fields(content)
	return len(fields) > 0 && strings.EqualFold(fields[0], "hot_update_canary_evidence_create")
}

func isMalformedHotUpdateCanarySatisfactionAuthorityCreateCommand(content string) bool {
	fields := strings.Fields(content)
	return len(fields) > 0 && strings.EqualFold(fields[0], "hot_update_canary_satisfaction_authority_create")
}

func isMalformedHotUpdateOwnerApprovalRequestCreateCommand(content string) bool {
	fields := strings.Fields(content)
	return len(fields) > 0 && strings.EqualFold(fields[0], "hot_update_owner_approval_request_create")
}

func isMalformedHotUpdateOwnerApprovalDecisionCreateCommand(content string) bool {
	fields := strings.Fields(content)
	return len(fields) > 0 && strings.EqualFold(fields[0], "hot_update_owner_approval_decision_create")
}

func isMalformedHotUpdateCanaryGateCreateCommand(content string) bool {
	fields := strings.Fields(content)
	return len(fields) > 0 && strings.EqualFold(fields[0], "hot_update_canary_gate_create")
}

func parseHotUpdateCanaryEvidenceObservedAt(value string) (time.Time, error) {
	observedAt, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(value))
	if err != nil {
		return time.Time{}, fmt.Errorf("HOT_UPDATE_CANARY_EVIDENCE_CREATE observed_at must be RFC3339 or RFC3339Nano: %w", err)
	}
	return observedAt, nil
}

func isMalformedHotUpdateExecutionReadyCommand(content string) bool {
	fields := strings.Fields(content)
	return len(fields) > 0 && strings.EqualFold(fields[0], "hot_update_execution_ready")
}

func cloneToolArguments(args map[string]interface{}) map[string]interface{} {
	if args == nil {
		return nil
	}

	cloned := make(map[string]interface{}, len(args))
	for key, value := range args {
		cloned[key] = value
	}
	return cloned
}
