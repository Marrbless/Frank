package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/local/picobot/internal/agent/memory"
	"github.com/local/picobot/internal/agent/tools"
	"github.com/local/picobot/internal/chat"
	"github.com/local/picobot/internal/cron"
	"github.com/local/picobot/internal/missioncontrol"
	"github.com/local/picobot/internal/providers"
	"github.com/local/picobot/internal/session"
)

var rememberRE = regexp.MustCompile(`(?i)^remember(?:\s+to)?\s+(.+)$`)
var approvalCommandRE = regexp.MustCompile(`(?i)^\s*(approve|deny)\s+(\S+)\s+(\S+)\s*$`)
var runtimeCommandRE = regexp.MustCompile(`(?i)^\s*(pause|resume|abort|status)\s+(\S+)\s*$`)
var inspectCommandRE = regexp.MustCompile(`(?i)^\s*(inspect)\s+(\S+)\s+(\S+)\s*$`)
var setStepCommandRE = regexp.MustCompile(`(?i)^\s*(set_step)\s+(\S+)\s+(\S+)\s*$`)

// sendChannelNotification delivers a non-blocking status message back to the
// originating channel so the user can see tool progress in real time.
// It is a no-op for system channels (heartbeat, cron) that have no user-facing chat.
func sendChannelNotification(hub *chat.Hub, channel, chatID, content string) {
	if isSystemChannel(channel) {
		return
	}
	out := chat.Outbound{Channel: channel, ChatID: chatID, Content: content}
	select {
	case hub.Out <- out:
	default:
		log.Println("sendChannelNotification: outbound channel full, dropping notification")
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

// AgentLoop is the core processing loop; it holds an LLM provider, tools, sessions and context builder.
type AgentLoop struct {
	hub                 *chat.Hub
	provider            providers.LLMProvider
	tools               *tools.Registry
	sessions            *session.SessionManager
	context             *ContextBuilder
	memory              *memory.MemoryStore
	model               string
	maxIterations       int
	running             bool
	taskState           *tools.TaskState
	operatorSetStepHook func(jobID string, stepID string) (string, error)
}

// NewAgentLoop creates a new AgentLoop with the given provider.
func NewAgentLoop(b *chat.Hub, provider providers.LLMProvider, model string, maxIterations int, workspace string, scheduler *cron.Scheduler) *AgentLoop {
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
	reg.Register(tools.NewSpawnTool())
	if scheduler != nil {
		reg.Register(tools.NewCronTool(scheduler))
	}

	sm := session.NewSessionManager(workspace)
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

	return &AgentLoop{
		hub:           b,
		provider:      provider,
		tools:         reg,
		sessions:      sm,
		context:       ctx,
		memory:        mem,
		model:         model,
		maxIterations: maxIterations,
		taskState:     taskState,
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
	a.taskState.SetRuntimeChangeHook(hook)
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

// Run starts processing inbound messages. This is a blocking call until context is canceled.
func (a *AgentLoop) Run(ctx context.Context) {
	a.running = true
	log.Println("Agent loop started")

	for a.running {
		select {
		case <-ctx.Done():
			log.Println("Agent loop received shutdown signal")
			a.running = false
			return

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
						content = err.Error()
					}
					if !isSystemChannel(msg.Channel) {
						sess := a.sessions.GetOrCreate(msg.Channel + ":" + msg.ChatID)
						sess.AddMessage("user", msg.Content)
						sess.AddMessage("assistant", content)
						if saveErr := a.sessions.Save(sess); saveErr != nil {
							log.Printf("error saving session: %v", saveErr)
						}
					}
					out := chat.Outbound{Channel: msg.Channel, ChatID: msg.ChatID, Content: content}
					select {
					case a.hub.Out <- out:
					default:
						log.Println("Outbound channel full, dropping message")
					}
					continue
				}
				if handled, content, err := a.taskState.ApplyNaturalApprovalDecision(msg.Content); handled {
					if err != nil {
						content = err.Error()
					}
					if !isSystemChannel(msg.Channel) {
						sess := a.sessions.GetOrCreate(msg.Channel + ":" + msg.ChatID)
						sess.AddMessage("user", msg.Content)
						sess.AddMessage("assistant", content)
						if saveErr := a.sessions.Save(sess); saveErr != nil {
							log.Printf("error saving session: %v", saveErr)
						}
					}
					out := chat.Outbound{Channel: msg.Channel, ChatID: msg.ChatID, Content: content}
					select {
					case a.hub.Out <- out:
					default:
						log.Println("Outbound channel full, dropping message")
					}
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
				if err := a.memory.AppendToday(note); err != nil {
					log.Printf("error appending to memory: %v", err)
				}
				out := chat.Outbound{Channel: msg.Channel, ChatID: msg.ChatID, Content: "OK, I've remembered that."}
				select {
				case a.hub.Out <- out:
				default:
					log.Println("Outbound channel full, dropping message")
				}

				if !isSystemChannel(msg.Channel) {
					sess := a.sessions.GetOrCreate(msg.Channel + ":" + msg.ChatID)
					sess.AddMessage("user", msg.Content)
					sess.AddMessage("assistant", "OK, I've remembered that.")
					if err := a.sessions.Save(sess); err != nil {
						log.Printf("error saving session: %v", err)
					}
				}
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
					log.Printf("provider error: %v", err)
					finalContent = "Sorry, I encountered an error while processing your request."
					break
				}

				if resp.HasToolCalls {
					messages = append(messages, providers.Message{
						Role:      "assistant",
						Content:   resp.Content,
						ToolCalls: resp.ToolCalls,
					})

					for _, tc := range resp.ToolCalls {
						argsJSON, _ := json.Marshal(tc.Arguments)
						sendChannelNotification(a.hub, msg.Channel, msg.ChatID,
							fmt.Sprintf("🤖 Running: %s %s", tc.Name, argsJSON))

						start := time.Now()
						execCtx := ctx
						if a.taskState != nil {
							if ec, ok := a.taskState.ExecutionContext(); ok {
								execCtx = missioncontrol.WithExecutionContext(ctx, ec)
							}
						}
						res, err := a.tools.Execute(execCtx, tc.Name, tc.Arguments)
						elapsed := time.Since(start).Round(time.Millisecond)

						if err != nil {
							sendChannelNotification(a.hub, msg.Channel, msg.ChatID,
								fmt.Sprintf("📢 %s failed (%s): %v", tc.Name, elapsed, err))
							res = "(tool error) " + err.Error()
						} else {
							successfulTools = append(successfulTools, missioncontrol.RuntimeToolCallEvidence{
								ToolName:  tc.Name,
								Arguments: cloneToolArguments(tc.Arguments),
								Result:    res,
							})
							sendChannelNotification(a.hub, msg.Channel, msg.ChatID,
								fmt.Sprintf("📢 %s done (%s)", tc.Name, elapsed))
						}

						lastToolResult = res
						messages = append(messages, providers.Message{
							Role:       "tool",
							Content:    res,
							ToolCallID: tc.ID,
						})
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
			if a.taskState != nil {
				if err := a.taskState.ApplyStepOutput(finalContent, successfulTools); err != nil {
					log.Printf("mission runtime step completion validation failed: %v", err)
				}
			}

			if !isSystemChannel(msg.Channel) {
				sess.AddMessage("user", msg.Content)
				sess.AddMessage("assistant", finalContent)
				if err := a.sessions.Save(sess); err != nil {
					log.Printf("error saving session: %v", err)
				}
			}

			out := chat.Outbound{Channel: msg.Channel, ChatID: msg.ChatID, Content: finalContent}
			select {
			case a.hub.Out <- out:
			default:
				log.Println("Outbound channel full, dropping message")
			}

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

	if a.taskState != nil {
		a.taskState.BeginTask(fmt.Sprintf("cli:direct:%d", time.Now().UnixNano()))
		a.taskState.SetOperatorSession("cli", "direct")
		if handled, response, err := a.processOperatorCommand(content); handled {
			return response, err
		}
		if handled, response, err := a.taskState.ApplyNaturalApprovalDecision(content); handled {
			return response, err
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
			return "", err
		}

		if !resp.HasToolCalls {
			if resp.Content != "" {
				if a.taskState != nil {
					if err := a.taskState.ApplyStepOutput(resp.Content, successfulTools); err != nil {
						log.Printf("mission runtime step completion validation failed: %v", err)
					}
				}
				return resp.Content, nil
			}
			if lastToolResult != "" {
				if a.taskState != nil {
					if err := a.taskState.ApplyStepOutput(lastToolResult, successfulTools); err != nil {
						log.Printf("mission runtime step completion validation failed: %v", err)
					}
				}
				return lastToolResult, nil
			}
			return resp.Content, nil
		}

		messages = append(messages, providers.Message{
			Role:      "assistant",
			Content:   resp.Content,
			ToolCalls: resp.ToolCalls,
		})

		for _, tc := range resp.ToolCalls {
			execCtx := ctx
			if a.taskState != nil {
				if ec, ok := a.taskState.ExecutionContext(); ok {
					execCtx = missioncontrol.WithExecutionContext(ctx, ec)
				}
			}
			result, err := a.tools.Execute(execCtx, tc.Name, tc.Arguments)
			if err != nil {
				result = "(tool error) " + err.Error()
			} else {
				successfulTools = append(successfulTools, missioncontrol.RuntimeToolCallEvidence{
					ToolName:  tc.Name,
					Arguments: cloneToolArguments(tc.Arguments),
					Result:    result,
				})
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

func (a *AgentLoop) processOperatorCommand(content string) (bool, string, error) {
	if a == nil || a.taskState == nil {
		return false, "", nil
	}

	trimmed := strings.TrimSpace(content)

	setStepMatches := setStepCommandRE.FindStringSubmatch(trimmed)
	if len(setStepMatches) == 4 {
		if a.operatorSetStepHook == nil {
			return true, "", missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
				Message: "SET_STEP requires mission step control configuration",
			}
		}
		response, err := a.operatorSetStepHook(setStepMatches[2], setStepMatches[3])
		return true, response, err
	}

	inspectMatches := inspectCommandRE.FindStringSubmatch(trimmed)
	if len(inspectMatches) == 4 {
		response, err := a.taskState.OperatorInspect(inspectMatches[2], inspectMatches[3])
		return true, response, err
	}

	matches := approvalCommandRE.FindStringSubmatch(trimmed)
	if len(matches) != 4 {
		runtimeMatches := runtimeCommandRE.FindStringSubmatch(trimmed)
		if len(runtimeMatches) != 3 {
			return false, "", nil
		}

		action := strings.ToLower(runtimeMatches[1])
		jobID := runtimeMatches[2]
		var err error
		switch action {
		case "pause":
			err = a.taskState.PauseRuntime(jobID)
		case "resume":
			err = a.taskState.ResumeRuntimeControl(jobID)
		case "abort":
			err = a.taskState.AbortRuntime(jobID)
		case "status":
			response, err := a.taskState.OperatorStatus(jobID)
			return true, response, err
		default:
			return false, "", nil
		}
		if err != nil {
			return true, "", err
		}

		verb := strings.ToUpper(action[:1]) + action[1:] + "d"
		if action == "pause" {
			verb = "Paused"
		} else if action == "abort" {
			verb = "Aborted"
		}
		return true, fmt.Sprintf("%s job=%s.", verb, jobID), nil
	}

	decision := missioncontrol.ApprovalDecision(strings.ToLower(matches[1]))
	jobID := matches[2]
	stepID := matches[3]
	if err := a.taskState.ApplyApprovalDecision(jobID, stepID, decision, missioncontrol.ApprovalGrantedViaOperatorCommand); err != nil {
		return true, "", err
	}

	verb := "Approved"
	if decision == missioncontrol.ApprovalDecisionDeny {
		verb = "Denied"
	}
	return true, fmt.Sprintf("%s job=%s step=%s.", verb, jobID, stepID), nil
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
