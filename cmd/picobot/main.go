package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/local/picobot/internal/agent"
	"github.com/local/picobot/internal/agent/memory"
	agenttools "github.com/local/picobot/internal/agent/tools"
	"github.com/local/picobot/internal/channels"
	"github.com/local/picobot/internal/chat"
	"github.com/local/picobot/internal/config"
	"github.com/local/picobot/internal/cron"
	"github.com/local/picobot/internal/heartbeat"
	"github.com/local/picobot/internal/missioncontrol"
	"github.com/local/picobot/internal/providers"
)

const version = "0.1.10"

var gatewayLogNow = time.Now
var newGatewayLogTicker = func(interval time.Duration) *time.Ticker {
	return time.NewTicker(interval)
}
var newStoreLogWriter = missioncontrol.NewStoreLogWriter
var packageCurrentLogSegmentOnGatewayStartup = missioncontrol.PackageCurrentLogSegmentOnGatewayStartup
var packageCurrentLogSegmentOnUTCDayRollover = missioncontrol.PackageCurrentLogSegmentOnUTCDayRollover

const missionStoreWriterLeaseHolderAnnotation = "picobot/mission-store-writer-lease-holder"
const scheduledTriggerStepID = "scheduled_trigger"

const (
	scheduledTriggerDeferralRecordVersion = 1
	scheduledTriggerHandleRouted          = "routed"
	scheduledTriggerHandleDeferred        = "deferred"
	scheduledTriggerHandleDuplicate       = "duplicate"
)

type deferredScheduledTriggerRecord struct {
	RecordVersion  int       `json:"record_version"`
	TriggerID      string    `json:"trigger_id"`
	SchedulerJobID string    `json:"scheduler_job_id"`
	Name           string    `json:"name,omitempty"`
	Message        string    `json:"message,omitempty"`
	Channel        string    `json:"channel,omitempty"`
	ChatID         string    `json:"chat_id,omitempty"`
	FireAt         time.Time `json:"fire_at"`
	DeferredAt     time.Time `json:"deferred_at"`
}

func (r deferredScheduledTriggerRecord) cronJob() cron.Job {
	return cron.Job{
		ID:      r.SchedulerJobID,
		Name:    r.Name,
		Message: r.Message,
		Channel: r.Channel,
		ChatID:  r.ChatID,
		FireAt:  r.FireAt,
	}
}

type governedScheduledTriggerDeferrer struct {
	mu        sync.Mutex
	storeRoot string
	inMemory  map[string]deferredScheduledTriggerRecord
	draining  bool
}

func newGovernedScheduledTriggerDeferrer(storeRoot string) *governedScheduledTriggerDeferrer {
	return &governedScheduledTriggerDeferrer{
		storeRoot: strings.TrimSpace(storeRoot),
		inMemory:  make(map[string]deferredScheduledTriggerRecord),
	}
}

func scheduledTriggerFireAt(job cron.Job) time.Time {
	fireAt := job.FireAt.UTC()
	if fireAt.IsZero() {
		fireAt = time.Unix(0, 0).UTC()
	}
	return fireAt
}

func buildGovernedScheduledTriggerJob(job cron.Job) missioncontrol.Job {
	fireAt := scheduledTriggerFireAt(job)
	return missioncontrol.Job{
		ID:           fmt.Sprintf("scheduled-trigger-%s-%s", strings.TrimSpace(job.ID), fireAt.Format("20060102T150405.000000000Z")),
		SpecVersion:  missioncontrol.JobSpecVersionV2,
		State:        missioncontrol.JobStatePending,
		MaxAuthority: missioncontrol.AuthorityTierLow,
		AllowedTools: nil,
		Plan: missioncontrol.Plan{
			ID: fmt.Sprintf("scheduled-trigger-plan-%s", strings.TrimSpace(job.ID)),
			Steps: []missioncontrol.Step{
				{
					ID:                scheduledTriggerStepID,
					Type:              missioncontrol.StepTypeFinalResponse,
					RequiredAuthority: missioncontrol.AuthorityTierLow,
					AllowedTools:      nil,
					RequiresApproval:  false,
					SuccessCriteria: []string{
						"Deliver the scheduled reminder plainly and truthfully to the operator.",
					},
				},
			},
		},
	}
}

func buildGovernedScheduledTriggerContent(job cron.Job) string {
	name := strings.TrimSpace(job.Name)
	message := strings.TrimSpace(job.Message)
	switch {
	case name != "" && message != "":
		return fmt.Sprintf("Scheduled reminder %q fired: %s", name, message)
	case message != "":
		return fmt.Sprintf("Scheduled reminder fired: %s", message)
	case name != "":
		return fmt.Sprintf("Scheduled reminder %q fired.", name)
	default:
		return "Scheduled reminder fired."
	}
}

func deferredScheduledTriggerDir(root string) string {
	return filepath.Join(root, "scheduler", "deferred_triggers")
}

func deferredScheduledTriggerPath(root, triggerID string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(triggerID)))
	return filepath.Join(deferredScheduledTriggerDir(root), hex.EncodeToString(sum[:])+".json")
}

func deferredScheduledTriggerRecordFromJob(trigger cron.Job, now time.Time) deferredScheduledTriggerRecord {
	governedJob := buildGovernedScheduledTriggerJob(trigger)
	return deferredScheduledTriggerRecord{
		RecordVersion:  scheduledTriggerDeferralRecordVersion,
		TriggerID:      governedJob.ID,
		SchedulerJobID: strings.TrimSpace(trigger.ID),
		Name:           strings.TrimSpace(trigger.Name),
		Message:        strings.TrimSpace(trigger.Message),
		Channel:        strings.TrimSpace(trigger.Channel),
		ChatID:         strings.TrimSpace(trigger.ChatID),
		FireAt:         scheduledTriggerFireAt(trigger),
		DeferredAt:     now.UTC(),
	}
}

func deferredScheduledTriggerSortLess(left deferredScheduledTriggerRecord, right deferredScheduledTriggerRecord) bool {
	leftFireAt := left.FireAt.UTC()
	rightFireAt := right.FireAt.UTC()
	if !leftFireAt.Equal(rightFireAt) {
		return leftFireAt.Before(rightFireAt)
	}
	return left.TriggerID < right.TriggerID
}

func (d *governedScheduledTriggerDeferrer) recordDeferredTrigger(trigger cron.Job, now time.Time) error {
	if d == nil {
		return fmt.Errorf("scheduler trigger deferrer is required")
	}

	record := deferredScheduledTriggerRecordFromJob(trigger, now)
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.storeRoot == "" {
		if _, exists := d.inMemory[record.TriggerID]; !exists {
			d.inMemory[record.TriggerID] = record
		}
		return nil
	}

	path := deferredScheduledTriggerPath(d.storeRoot, record.TriggerID)
	var existing deferredScheduledTriggerRecord
	if err := missioncontrol.LoadStoreJSON(path, &existing); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return missioncontrol.WriteStoreJSONAtomic(path, record)
}

func (d *governedScheduledTriggerDeferrer) listDeferredTriggers() ([]deferredScheduledTriggerRecord, error) {
	if d == nil {
		return nil, nil
	}

	d.mu.Lock()
	storeRoot := d.storeRoot
	if storeRoot == "" {
		records := make([]deferredScheduledTriggerRecord, 0, len(d.inMemory))
		for _, record := range d.inMemory {
			records = append(records, record)
		}
		d.mu.Unlock()
		sort.Slice(records, func(i, j int) bool {
			return deferredScheduledTriggerSortLess(records[i], records[j])
		})
		return records, nil
	}
	d.mu.Unlock()

	entries, err := os.ReadDir(deferredScheduledTriggerDir(storeRoot))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	records := make([]deferredScheduledTriggerRecord, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		var record deferredScheduledTriggerRecord
		if err := missioncontrol.LoadStoreJSON(filepath.Join(deferredScheduledTriggerDir(storeRoot), entry.Name()), &record); err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	sort.Slice(records, func(i, j int) bool {
		return deferredScheduledTriggerSortLess(records[i], records[j])
	})
	return records, nil
}

func (d *governedScheduledTriggerDeferrer) removeDeferredTrigger(triggerID string) error {
	if d == nil {
		return nil
	}

	triggerID = strings.TrimSpace(triggerID)
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.storeRoot == "" {
		delete(d.inMemory, triggerID)
		return nil
	}

	path := deferredScheduledTriggerPath(d.storeRoot, triggerID)
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func (d *governedScheduledTriggerDeferrer) blockingMissionJobID(ag *agent.AgentLoop) (string, bool, error) {
	if ag != nil {
		if runtime, ok := ag.MissionRuntimeState(); ok && runtime.JobID != "" && !missioncontrol.IsTerminalJobState(runtime.State) {
			return runtime.JobID, true, nil
		}
	}
	if d == nil || d.storeRoot == "" {
		return "", false, nil
	}

	record, err := missioncontrol.LoadActiveJobRecord(d.storeRoot)
	if err != nil {
		if errors.Is(err, missioncontrol.ErrActiveJobRecordNotFound) {
			return "", false, nil
		}
		return "", false, err
	}
	if record.JobID == "" || !missioncontrol.HoldsGlobalActiveJobOccupancy(record.State) {
		return "", false, nil
	}
	return record.JobID, true, nil
}

func (d *governedScheduledTriggerDeferrer) triggerAlreadyGoverned(ag *agent.AgentLoop, triggerID string) (bool, error) {
	triggerID = strings.TrimSpace(triggerID)
	if ag != nil {
		if runtime, ok := ag.MissionRuntimeState(); ok && runtime.JobID == triggerID {
			return true, nil
		}
	}
	if d == nil || d.storeRoot == "" {
		return false, nil
	}
	_, err := missioncontrol.LoadCommittedJobRuntimeRecord(d.storeRoot, triggerID)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, missioncontrol.ErrJobRuntimeRecordNotFound) {
		return false, nil
	}
	return false, err
}

func (d *governedScheduledTriggerDeferrer) routeOrDefer(ag *agent.AgentLoop, hub *chat.Hub, trigger cron.Job) (string, error) {
	record := deferredScheduledTriggerRecordFromJob(trigger, time.Now().UTC())
	alreadyGoverned, err := d.triggerAlreadyGoverned(ag, record.TriggerID)
	if err != nil {
		return "", err
	}
	if alreadyGoverned {
		if err := d.removeDeferredTrigger(record.TriggerID); err != nil {
			return "", err
		}
		return scheduledTriggerHandleDuplicate, nil
	}

	if blockerJobID, blocked, err := d.blockingMissionJobID(ag); err != nil {
		return "", err
	} else if blocked && blockerJobID != "" && blockerJobID != record.TriggerID {
		if err := d.recordDeferredTrigger(trigger, time.Now().UTC()); err != nil {
			return "", err
		}
		return scheduledTriggerHandleDeferred, nil
	}

	if err := routeScheduledTriggerThroughGovernedJob(ag, hub, trigger); err != nil {
		if blockerJobID, blocked, blockErr := d.blockingMissionJobID(ag); blockErr != nil {
			return "", blockErr
		} else if blocked && blockerJobID != "" && blockerJobID != record.TriggerID {
			if deferErr := d.recordDeferredTrigger(trigger, time.Now().UTC()); deferErr != nil {
				return "", deferErr
			}
			return scheduledTriggerHandleDeferred, nil
		}
		return "", err
	}

	return scheduledTriggerHandleRouted, nil
}

func (d *governedScheduledTriggerDeferrer) drainReady(ag *agent.AgentLoop, hub *chat.Hub) error {
	if d == nil {
		return nil
	}

	d.mu.Lock()
	if d.draining {
		d.mu.Unlock()
		return nil
	}
	d.draining = true
	d.mu.Unlock()
	defer func() {
		d.mu.Lock()
		d.draining = false
		d.mu.Unlock()
	}()

	for {
		if _, blocked, err := d.blockingMissionJobID(ag); err != nil {
			return err
		} else if blocked {
			return nil
		}

		records, err := d.listDeferredTriggers()
		if err != nil {
			return err
		}
		if len(records) == 0 {
			return nil
		}

		record := records[0]
		alreadyGoverned, err := d.triggerAlreadyGoverned(ag, record.TriggerID)
		if err != nil {
			return err
		}
		if alreadyGoverned {
			if err := d.removeDeferredTrigger(record.TriggerID); err != nil {
				return err
			}
			continue
		}

		if err := routeScheduledTriggerThroughGovernedJob(ag, hub, record.cronJob()); err != nil {
			if _, blocked, blockErr := d.blockingMissionJobID(ag); blockErr != nil {
				return blockErr
			} else if blocked {
				return nil
			}
			return err
		}
		return d.removeDeferredTrigger(record.TriggerID)
	}
}

func routeScheduledTriggerThroughGovernedJob(ag *agent.AgentLoop, hub *chat.Hub, trigger cron.Job) error {
	if ag == nil {
		return fmt.Errorf("scheduler trigger requires an active agent loop")
	}
	if hub == nil {
		return fmt.Errorf("scheduler trigger requires a chat hub")
	}

	job := buildGovernedScheduledTriggerJob(trigger)
	if err := ag.ActivateMissionStep(job, scheduledTriggerStepID); err != nil {
		return err
	}

	hub.In <- chat.Inbound{
		Channel:  trigger.Channel,
		SenderID: "cron",
		ChatID:   trigger.ChatID,
		Content:  buildGovernedScheduledTriggerContent(trigger),
	}
	return nil
}

func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "picobot",
		Short: "picobot — lightweight clawbot in Go",
	}

	rootCmd.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Print version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("🤖 picobot v%s\n", version)
		},
	})

	onboardCmd := &cobra.Command{
		Use:   "onboard",
		Short: "Create default config and workspace",
		Run: func(cmd *cobra.Command, args []string) {
			cfgPath, workspacePath, err := config.Onboard()
			if err != nil {
				fmt.Fprintf(os.Stderr, "onboard failed: %v\n", err)
				return
			}
			fmt.Printf("Wrote config to %s\nInitialized workspace at %s\n", cfgPath, workspacePath)
		},
	}

	rootCmd.AddCommand(onboardCmd)

	// channels command — connect and configure messaging channels interactively.
	channelsCmd := &cobra.Command{
		Use:   "channels",
		Short: "Manage channel connections (Telegram, Discord, Slack, WhatsApp)",
	}

	loginCmd := &cobra.Command{
		Use:   "login",
		Short: "Interactively connect a channel (Telegram, Discord, Slack, or WhatsApp)",
		Run: func(cmd *cobra.Command, args []string) {
			reader := bufio.NewReader(os.Stdin)

			fmt.Println("Which channel would you like to connect?")
			fmt.Println()
			fmt.Println("  1) Telegram")
			fmt.Println("  2) Discord")
			fmt.Println("  3) Slack")
			fmt.Println("  4) WhatsApp")
			fmt.Println()
			fmt.Print("Enter 1, 2, 3 or 4: ")

			choice, _ := reader.ReadString('\n')
			choice = strings.TrimSpace(strings.ToLower(choice))

			cfg, err := config.LoadConfig()
			if err != nil {
				fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
				return
			}
			cfgPath, _, err := config.ResolveDefaultPaths()
			if err != nil {
				fmt.Fprintf(os.Stderr, "failed to resolve config path: %v\n", err)
				return
			}

			switch choice {
			case "1", "telegram":
				setupTelegramInteractive(reader, cfg, cfgPath)
			case "2", "discord":
				setupDiscordInteractive(reader, cfg, cfgPath)
			case "3", "slack":
				setupSlackInteractive(reader, cfg, cfgPath)
			case "4", "whatsapp":
				setupWhatsAppInteractive(cfg, cfgPath)
			default:
				fmt.Fprintf(os.Stderr, "invalid choice %q — please enter 1, 2, 3 or 4\n", choice)
			}
		},
	}

	channelsCmd.AddCommand(loginCmd)
	rootCmd.AddCommand(channelsCmd)

	agentCmd := &cobra.Command{
		Use:   "agent",
		Short: "Run a single-shot agent query (use -m)",
		RunE: func(cmd *cobra.Command, args []string) error {
			msg, _ := cmd.Flags().GetString("message")
			modelFlag, _ := cmd.Flags().GetString("model")
			if msg == "" {
				fmt.Println("Specify a message with -m \"your message\"")
				return nil
			}

			hub := chat.NewHub(100)
			cfg, _ := config.LoadConfig()
			provider := providers.NewProviderFromConfig(cfg)

			// choose model: flag > config default > provider default
			model := modelFlag
			if model == "" && cfg.Agents.Defaults.Model != "" {
				model = cfg.Agents.Defaults.Model
			}
			if model == "" {
				model = provider.GetDefaultModel()
			}

			maxIter := cfg.Agents.Defaults.MaxToolIterations
			if maxIter <= 0 {
				maxIter = 100
			}
			ag := agent.NewAgentLoop(hub, provider, model, maxIter, cfg.Agents.Defaults.Workspace, nil)
			installMissionRuntimeChangeHook(cmd, ag)
			bootstrappedJob, err := configureMissionBootstrapJob(cmd, ag)
			if err != nil {
				return err
			}
			installMissionOperatorSetStepHook(cmd, ag, bootstrappedJob, true)
			if err := writeMissionStatusSnapshotFromCommand(cmd, ag, time.Now()); err != nil {
				return err
			}

			resp, err := ag.ProcessDirect(msg, 60*time.Second)
			if err != nil {
				fmt.Fprintln(cmd.ErrOrStderr(), "error:", err)
				return nil
			}
			fmt.Fprintln(cmd.OutOrStdout(), resp)
			return nil
		},
	}
	agentCmd.Flags().StringP("message", "m", "", "Message to send to the agent")
	agentCmd.Flags().StringP("model", "M", "", "Model to use (overrides config/provider default)")
	addMissionBootstrapFlags(agentCmd)
	rootCmd.AddCommand(agentCmd)

	gatewayCmd := &cobra.Command{
		Use:   "gateway",
		Short: "Start long-running gateway (agent, channels, heartbeat)",
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			hub := chat.NewHub(200)
			cfg, _ := config.LoadConfig()
			provider := providers.NewProviderFromConfig(cfg)
			var ag *agent.AgentLoop

			// choose model: flag > config > provider default
			modelFlag, _ := cmd.Flags().GetString("model")
			model := modelFlag
			if model == "" && cfg.Agents.Defaults.Model != "" {
				model = cfg.Agents.Defaults.Model
			}
			if model == "" {
				model = provider.GetDefaultModel()
			}

			storeRoot, logLease, restoreGatewayLogger, err := configureGatewayMissionStoreLogging(cmd)
			if err != nil {
				return err
			}
			defer restoreGatewayLogger()

			deferredScheduledTriggers := newGovernedScheduledTriggerDeferrer(resolveMissionStoreRoot(cmd))

			// Route fired schedules through a governed mission step before they re-enter the ordinary agent loop.
			scheduler := cron.NewScheduler(func(job cron.Job) {
				log.Printf("cron fired: %s — %s", job.Name, job.Message)
				result, err := deferredScheduledTriggers.routeOrDefer(ag, hub, job)
				if err != nil {
					log.Printf("cron governed trigger failed for %s: %v", job.ID, err)
					return
				}
				switch result {
				case scheduledTriggerHandleDeferred:
					log.Printf("cron governed trigger deferred for %s", job.ID)
				case scheduledTriggerHandleDuplicate:
					log.Printf("cron governed trigger already governed for %s", job.ID)
				}
			})

			maxIter := cfg.Agents.Defaults.MaxToolIterations
			if maxIter <= 0 {
				maxIter = 100
			}
			ag = agent.NewAgentLoop(hub, provider, model, maxIter, cfg.Agents.Defaults.Workspace, scheduler)
			installMissionRuntimeChangeHookWithExtension(cmd, ag, func() {
				if err := deferredScheduledTriggers.drainReady(ag, hub); err != nil {
					log.Printf("cron governed deferred trigger drain failed: %v", err)
				}
			})
			bootstrappedJob, err := configureMissionBootstrapJob(cmd, ag)
			if err != nil {
				return err
			}
			installMissionOperatorSetStepHook(cmd, ag, bootstrappedJob, false)
			var missionStepControlBaseline []byte
			if bootstrappedJob != nil {
				missionStepControlBaseline = restoreMissionStepControlFileOnStartup(cmd, ag, *bootstrappedJob)
			}
			if err := deferredScheduledTriggers.drainReady(ag, hub); err != nil {
				return err
			}
			if err := writeMissionStatusSnapshotFromCommand(cmd, ag, time.Now()); err != nil {
				return err
			}
			statusFile, _ := cmd.Flags().GetString("mission-status-file")
			if statusFile != "" {
				defer func() {
					if err == nil {
						err = removeMissionStatusSnapshot(statusFile)
					}
				}()
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			if storeRoot != "" {
				go watchGatewayLogDayRollover(ctx, storeRoot, logLease, time.Minute)
			}

			// start agent loop
			go ag.Run(ctx)

			if bootstrappedJob != nil {
				controlFile, _ := cmd.Flags().GetString("mission-step-control-file")
				if controlFile != "" {
					go watchMissionStepControlFile(ctx, cmd, ag, *bootstrappedJob, controlFile, 500*time.Millisecond, missionStepControlBaseline)
				}
			}

			// start cron scheduler
			go scheduler.Start(ctx.Done())

			// start heartbeat
			hbInterval := time.Duration(cfg.Agents.Defaults.HeartbeatIntervalS) * time.Second
			if hbInterval <= 0 {
				hbInterval = 60 * time.Second
			}
			heartbeat.StartHeartbeat(ctx, cfg.Agents.Defaults.Workspace, hbInterval, hub)

			// start telegram if enabled
			if cfg.Channels.Telegram.Enabled {
				if err := channels.StartTelegram(ctx, hub, cfg.Channels.Telegram.Token, cfg.Channels.Telegram.AllowFrom); err != nil {
					fmt.Fprintf(os.Stderr, "failed to start telegram: %v\n", err)
				}
			}

			// start discord if enabled
			if cfg.Channels.Discord.Enabled {
				if err := channels.StartDiscord(ctx, hub, cfg.Channels.Discord.Token, cfg.Channels.Discord.AllowFrom); err != nil {
					fmt.Fprintf(os.Stderr, "failed to start discord: %v\n", err)
				}
			}

			// start slack if enabled
			if cfg.Channels.Slack.Enabled {
				if err := channels.StartSlack(ctx, hub, cfg.Channels.Slack.AppToken, cfg.Channels.Slack.BotToken, cfg.Channels.Slack.AllowUsers, cfg.Channels.Slack.AllowChannels); err != nil {
					fmt.Fprintf(os.Stderr, "failed to start slack: %v\n", err)
				}
			}

			// start whatsapp if enabled
			if cfg.Channels.WhatsApp.Enabled {
				dbPath := cfg.Channels.WhatsApp.DBPath
				if dbPath == "" {
					dbPath = "~/.picobot/whatsapp.db"
				}
				// Expand home directory
				if strings.HasPrefix(dbPath, "~/") {
					home, _ := os.UserHomeDir()
					dbPath = filepath.Join(home, dbPath[2:])
				}
				if err := channels.StartWhatsApp(ctx, hub, dbPath, cfg.Channels.WhatsApp.AllowFrom); err != nil {
					fmt.Fprintf(os.Stderr, "failed to start whatsapp: %v\n", err)
				}
			}

			// start hub router after all channels have subscribed.
			// This routes outbound messages from hub.Out to each channel's
			// dedicated queue, preventing competing reads when multiple channels
			// are active simultaneously.
			hub.StartRouter(ctx)

			// wait for signal
			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
			<-sigCh
			fmt.Println("shutting down gateway")
			cancel()
			return nil
		},
	}
	gatewayCmd.Flags().StringP("model", "M", "", "Model to use (overrides model in config.json)")
	addMissionBootstrapFlags(gatewayCmd)
	gatewayCmd.Flags().String("mission-step-control-file", "", "Path to a mission step control JSON file for gateway step switching")
	rootCmd.AddCommand(gatewayCmd)

	missionCmd := &cobra.Command{
		Use:   "mission",
		Short: "Read mission status and switch mission steps",
	}

	missionStatusCmd := &cobra.Command{
		Use:          "status",
		Short:        "Print a mission status snapshot JSON file",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			statusFile, _ := cmd.Flags().GetString("status-file")
			if statusFile == "" {
				return fmt.Errorf("--status-file is required")
			}

			frankZohoSendProofOnly, _ := cmd.Flags().GetBool("frank-zoho-send-proof")
			frankZohoVerifySendProof, _ := cmd.Flags().GetBool("frank-zoho-verify-send-proof")
			if frankZohoSendProofOnly && frankZohoVerifySendProof {
				return fmt.Errorf("--frank-zoho-send-proof and --frank-zoho-verify-send-proof are mutually exclusive")
			}

			var (
				data []byte
				err  error
			)
			if frankZohoVerifySendProof {
				data, err = loadMissionStatusFrankZohoVerifiedSendProofFile(cmd.Context(), statusFile)
			} else if frankZohoSendProofOnly {
				data, err = loadMissionStatusFrankZohoSendProofFile(statusFile)
			} else {
				data, err = loadGatewayStatusObservationFile(statusFile)
			}
			if err != nil {
				return err
			}

			if _, err := cmd.OutOrStdout().Write(data); err != nil {
				return fmt.Errorf("failed to write mission status output: %w", err)
			}

			return nil
		},
	}
	missionStatusCmd.Flags().String("status-file", "", "Path to a mission status snapshot JSON file")
	missionStatusCmd.Flags().Bool("frank-zoho-send-proof", false, "Print provider-specific Frank Zoho send proof locators from committed runtime_summary.frank_zoho_send_proof")
	missionStatusCmd.Flags().Bool("frank-zoho-verify-send-proof", false, "Fetch originalmessage bodies for provider-specific Frank Zoho send proof locators from committed runtime_summary.frank_zoho_send_proof")

	missionInspectCmd := &cobra.Command{
		Use:          "inspect",
		Short:        "Inspect a mission JSON file and list valid steps",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			missionFile, _ := cmd.Flags().GetString("mission-file")
			if missionFile == "" {
				return fmt.Errorf("--mission-file is required")
			}

			job, err := loadMissionJobFile(missionFile)
			if err != nil {
				return err
			}
			if err := validateMissionJob(job, resolveMissionStoreRoot(cmd)); err != nil {
				return fmt.Errorf("failed to validate mission file %q: %w", missionFile, err)
			}

			stepID, _ := cmd.Flags().GetString("step-id")
			storeRoot := resolveMissionStoreRoot(cmd)
			notificationsCapabilityOnly, _ := cmd.Flags().GetBool("notifications-capability")
			sharedStorageCapabilityOnly, _ := cmd.Flags().GetBool("shared-storage-capability")
			contactsCapabilityOnly, _ := cmd.Flags().GetBool("contacts-capability")
			locationCapabilityOnly, _ := cmd.Flags().GetBool("location-capability")
			cameraCapabilityOnly, _ := cmd.Flags().GetBool("camera-capability")
			microphoneCapabilityOnly, _ := cmd.Flags().GetBool("microphone-capability")
			smsPhoneCapabilityOnly, _ := cmd.Flags().GetBool("sms-phone-capability")
			bluetoothNFCCapabilityOnly, _ := cmd.Flags().GetBool("bluetooth-nfc-capability")
			broadAppControlCapabilityOnly, _ := cmd.Flags().GetBool("broad-app-control-capability")

			if notificationsCapabilityOnly {
				if storeRoot == "" {
					return fmt.Errorf("--mission-store-root is required with --notifications-capability")
				}
				record, err := newMissionInspectNotificationsCapability(job, stepID, storeRoot)
				if err != nil {
					return fmt.Errorf("failed to resolve notifications capability inspection for %q: %w", missionFile, err)
				}
				recordData, err := json.MarshalIndent(record, "", "  ")
				if err != nil {
					return fmt.Errorf("failed to encode notifications capability inspection output: %w", err)
				}
				recordData = append(recordData, '\n')
				if _, err := cmd.OutOrStdout().Write(recordData); err != nil {
					return fmt.Errorf("failed to write notifications capability inspection output: %w", err)
				}
				return nil
			}
			if sharedStorageCapabilityOnly {
				if storeRoot == "" {
					return fmt.Errorf("--mission-store-root is required with --shared-storage-capability")
				}
				record, err := newMissionInspectSharedStorageCapability(job, stepID, storeRoot)
				if err != nil {
					return fmt.Errorf("failed to resolve shared_storage capability inspection for %q: %w", missionFile, err)
				}
				recordData, err := json.MarshalIndent(record, "", "  ")
				if err != nil {
					return fmt.Errorf("failed to encode shared_storage capability inspection output: %w", err)
				}
				recordData = append(recordData, '\n')
				if _, err := cmd.OutOrStdout().Write(recordData); err != nil {
					return fmt.Errorf("failed to write shared_storage capability inspection output: %w", err)
				}
				return nil
			}
			if contactsCapabilityOnly {
				if storeRoot == "" {
					return fmt.Errorf("--mission-store-root is required with --contacts-capability")
				}
				record, err := newMissionInspectContactsCapability(job, stepID, storeRoot)
				if err != nil {
					return fmt.Errorf("failed to resolve contacts capability inspection for %q: %w", missionFile, err)
				}
				recordData, err := json.MarshalIndent(record, "", "  ")
				if err != nil {
					return fmt.Errorf("failed to encode contacts capability inspection output: %w", err)
				}
				recordData = append(recordData, '\n')
				if _, err := cmd.OutOrStdout().Write(recordData); err != nil {
					return fmt.Errorf("failed to write contacts capability inspection output: %w", err)
				}
				return nil
			}
			if locationCapabilityOnly {
				if storeRoot == "" {
					return fmt.Errorf("--mission-store-root is required with --location-capability")
				}
				record, err := newMissionInspectLocationCapability(job, stepID, storeRoot)
				if err != nil {
					return fmt.Errorf("failed to resolve location capability inspection for %q: %w", missionFile, err)
				}
				recordData, err := json.MarshalIndent(record, "", "  ")
				if err != nil {
					return fmt.Errorf("failed to encode location capability inspection output: %w", err)
				}
				recordData = append(recordData, '\n')
				if _, err := cmd.OutOrStdout().Write(recordData); err != nil {
					return fmt.Errorf("failed to write location capability inspection output: %w", err)
				}
				return nil
			}
			if cameraCapabilityOnly {
				if storeRoot == "" {
					return fmt.Errorf("--mission-store-root is required with --camera-capability")
				}
				record, err := newMissionInspectCameraCapability(job, stepID, storeRoot)
				if err != nil {
					return fmt.Errorf("failed to resolve camera capability inspection for %q: %w", missionFile, err)
				}
				recordData, err := json.MarshalIndent(record, "", "  ")
				if err != nil {
					return fmt.Errorf("failed to encode camera capability inspection output: %w", err)
				}
				recordData = append(recordData, '\n')
				if _, err := cmd.OutOrStdout().Write(recordData); err != nil {
					return fmt.Errorf("failed to write camera capability inspection output: %w", err)
				}
				return nil
			}
			if microphoneCapabilityOnly {
				if storeRoot == "" {
					return fmt.Errorf("--mission-store-root is required with --microphone-capability")
				}
				record, err := newMissionInspectMicrophoneCapability(job, stepID, storeRoot)
				if err != nil {
					return fmt.Errorf("failed to resolve microphone capability inspection for %q: %w", missionFile, err)
				}
				recordData, err := json.MarshalIndent(record, "", "  ")
				if err != nil {
					return fmt.Errorf("failed to encode microphone capability inspection output: %w", err)
				}
				recordData = append(recordData, '\n')
				if _, err := cmd.OutOrStdout().Write(recordData); err != nil {
					return fmt.Errorf("failed to write microphone capability inspection output: %w", err)
				}
				return nil
			}
			if smsPhoneCapabilityOnly {
				if storeRoot == "" {
					return fmt.Errorf("--mission-store-root is required with --sms-phone-capability")
				}
				record, err := newMissionInspectSMSPhoneCapability(job, stepID, storeRoot)
				if err != nil {
					return fmt.Errorf("failed to resolve sms_phone capability inspection for %q: %w", missionFile, err)
				}
				recordData, err := json.MarshalIndent(record, "", "  ")
				if err != nil {
					return fmt.Errorf("failed to encode sms_phone capability inspection output: %w", err)
				}
				recordData = append(recordData, '\n')
				if _, err := cmd.OutOrStdout().Write(recordData); err != nil {
					return fmt.Errorf("failed to write sms_phone capability inspection output: %w", err)
				}
				return nil
			}
			if bluetoothNFCCapabilityOnly {
				if storeRoot == "" {
					return fmt.Errorf("--mission-store-root is required with --bluetooth-nfc-capability")
				}
				record, err := newMissionInspectBluetoothNFCCapability(job, stepID, storeRoot)
				if err != nil {
					return fmt.Errorf("failed to resolve bluetooth_nfc capability inspection for %q: %w", missionFile, err)
				}
				recordData, err := json.MarshalIndent(record, "", "  ")
				if err != nil {
					return fmt.Errorf("failed to encode bluetooth_nfc capability inspection output: %w", err)
				}
				recordData = append(recordData, '\n')
				if _, err := cmd.OutOrStdout().Write(recordData); err != nil {
					return fmt.Errorf("failed to write bluetooth_nfc capability inspection output: %w", err)
				}
				return nil
			}
			if broadAppControlCapabilityOnly {
				if storeRoot == "" {
					return fmt.Errorf("--mission-store-root is required with --broad-app-control-capability")
				}
				record, err := newMissionInspectBroadAppControlCapability(job, stepID, storeRoot)
				if err != nil {
					return fmt.Errorf("failed to resolve broad_app_control capability inspection for %q: %w", missionFile, err)
				}
				recordData, err := json.MarshalIndent(record, "", "  ")
				if err != nil {
					return fmt.Errorf("failed to encode broad_app_control capability inspection output: %w", err)
				}
				recordData = append(recordData, '\n')
				if _, err := cmd.OutOrStdout().Write(recordData); err != nil {
					return fmt.Errorf("failed to write broad_app_control capability inspection output: %w", err)
				}
				return nil
			}

			summary, err := newMissionInspectSummary(job, stepID, storeRoot)
			if err != nil {
				return fmt.Errorf("failed to resolve mission inspection summary for %q: %w", missionFile, err)
			}

			summaryData, err := json.MarshalIndent(summary, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to encode mission inspection output: %w", err)
			}
			summaryData = append(summaryData, '\n')

			if _, err := cmd.OutOrStdout().Write(summaryData); err != nil {
				return fmt.Errorf("failed to write mission inspection output: %w", err)
			}

			return nil
		},
	}
	missionInspectCmd.Flags().String("mission-file", "", "Path to a mission job JSON file")
	missionInspectCmd.Flags().String("mission-store-root", "", "Path to the durable mission store root")
	missionInspectCmd.Flags().String("step-id", "", "Optional mission step ID to filter the inspection summary to one step")
	missionInspectCmd.Flags().Bool("notifications-capability", false, "Print the committed notifications capability record from the mission store")
	missionInspectCmd.Flags().Bool("shared-storage-capability", false, "Print the committed shared_storage capability record from the mission store")
	missionInspectCmd.Flags().Bool("contacts-capability", false, "Print the committed contacts capability record and source from the mission store")
	missionInspectCmd.Flags().Bool("location-capability", false, "Print the committed location capability record and source from the mission store")
	missionInspectCmd.Flags().Bool("camera-capability", false, "Print the committed camera capability record and source from the mission store")
	missionInspectCmd.Flags().Bool("microphone-capability", false, "Print the committed microphone capability record and source from the mission store")
	missionInspectCmd.Flags().Bool("sms-phone-capability", false, "Print the committed sms_phone capability record and source from the mission store")
	missionInspectCmd.Flags().Bool("bluetooth-nfc-capability", false, "Print the committed bluetooth_nfc capability record and source from the mission store")
	missionInspectCmd.Flags().Bool("broad-app-control-capability", false, "Print the committed broad_app_control capability record and source from the mission store")

	missionAssertCmd := &cobra.Command{
		Use:          "assert",
		Short:        "Assert mission status conditions from a snapshot JSON file",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			statusFile, _ := cmd.Flags().GetString("status-file")
			if statusFile == "" {
				return fmt.Errorf("--status-file is required")
			}

			expected := missionStatusAssertionExpectation{}
			if cmd.Flags().Changed("job-id") {
				expected.JobID = valueOrNilString(cmd.Flags().Lookup("job-id").Value.String())
			}
			if cmd.Flags().Changed("step-id") {
				expected.StepID = valueOrNilString(cmd.Flags().Lookup("step-id").Value.String())
			}
			if cmd.Flags().Changed("active") {
				active, _ := cmd.Flags().GetBool("active")
				expected.Active = &active
			}
			if cmd.Flags().Changed("step-type") {
				expected.StepType = valueOrNilString(cmd.Flags().Lookup("step-type").Value.String())
			}
			if cmd.Flags().Changed("required-authority") {
				requiredAuthority := missioncontrol.AuthorityTier(cmd.Flags().Lookup("required-authority").Value.String())
				expected.RequiredAuthority = &requiredAuthority
			}
			if cmd.Flags().Changed("requires-approval") {
				requiresApproval := true
				expected.RequiresApproval = &requiresApproval
			}
			if cmd.Flags().Changed("no-requires-approval") {
				requiresApproval := false
				expected.RequiresApproval = &requiresApproval
			}
			expected.NoTools, _ = cmd.Flags().GetBool("no-tools")
			expected.HasTools, _ = cmd.Flags().GetStringArray("has-tool")
			if cmd.Flags().Changed("exact-tool") {
				expected.ExactAllowedTools, _ = cmd.Flags().GetStringArray("exact-tool")
				expected.CheckExactAllowedTools = true
			}
			if cmd.Flags().Changed("requires-approval") && cmd.Flags().Changed("no-requires-approval") {
				return fmt.Errorf("--requires-approval and --no-requires-approval cannot be used together")
			}
			if expected.NoTools && len(expected.HasTools) > 0 {
				return fmt.Errorf("--no-tools and --has-tool cannot be used together")
			}
			if expected.NoTools && expected.CheckExactAllowedTools {
				return fmt.Errorf("--no-tools and --exact-tool cannot be used together")
			}
			if len(expected.HasTools) > 0 && expected.CheckExactAllowedTools {
				return fmt.Errorf("--has-tool and --exact-tool cannot be used together")
			}

			waitTimeout, _ := cmd.Flags().GetDuration("wait-timeout")
			if waitTimeout <= 0 {
				return assertMissionGatewayStatusSnapshot(statusFile, expected)
			}
			return waitForMissionGatewayStatusAssertion(statusFile, expected, waitTimeout)
		},
	}
	missionAssertCmd.Flags().String("status-file", "", "Path to a mission status snapshot JSON file")
	missionAssertCmd.Flags().String("job-id", "", "Expected mission job ID")
	missionAssertCmd.Flags().String("step-id", "", "Expected mission step ID")
	missionAssertCmd.Flags().Bool("active", false, "Expected mission active state")
	missionAssertCmd.Flags().String("step-type", "", "Expected mission step type")
	missionAssertCmd.Flags().String("required-authority", "", "Expected mission required authority tier")
	missionAssertCmd.Flags().Bool("requires-approval", false, "Require mission requires_approval=true")
	missionAssertCmd.Flags().Bool("no-requires-approval", false, "Require mission requires_approval=false")
	missionAssertCmd.Flags().Bool("no-tools", false, "Require allowed_tools to be empty")
	missionAssertCmd.Flags().StringArray("has-tool", nil, "Require a named tool to appear in allowed_tools; repeat to require multiple tools")
	missionAssertCmd.Flags().StringArray("exact-tool", nil, "Require allowed_tools to exactly match the named tools in order; repeat to require multiple tools")
	missionAssertCmd.Flags().Duration("wait-timeout", 0, "How long to wait for mission status assertion success")

	missionAssertStepCmd := &cobra.Command{
		Use:          "assert-step",
		Short:        "Assert mission status matches a specific step from a mission JSON file",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			statusFile, _ := cmd.Flags().GetString("status-file")
			if statusFile == "" {
				return fmt.Errorf("--status-file is required")
			}

			missionFile, _ := cmd.Flags().GetString("mission-file")
			if missionFile == "" {
				return fmt.Errorf("--mission-file is required")
			}

			stepID, _ := cmd.Flags().GetString("step-id")
			if stepID == "" {
				return fmt.Errorf("--step-id is required")
			}

			job, err := loadMissionJobFile(missionFile)
			if err != nil {
				return err
			}

			expected, err := newMissionStatusAssertionForStep(job, stepID)
			if err != nil {
				return fmt.Errorf("failed to validate mission file %q: %w", missionFile, err)
			}

			waitTimeout, _ := cmd.Flags().GetDuration("wait-timeout")
			if waitTimeout <= 0 {
				return assertMissionGatewayStatusSnapshot(statusFile, expected)
			}
			return waitForMissionGatewayStatusAssertion(statusFile, expected, waitTimeout)
		},
	}
	missionAssertStepCmd.Flags().String("status-file", "", "Path to a mission status snapshot JSON file")
	missionAssertStepCmd.Flags().String("mission-file", "", "Path to a mission job JSON file")
	missionAssertStepCmd.Flags().String("step-id", "", "Mission step ID to assert against the live status snapshot")
	missionAssertStepCmd.Flags().Duration("wait-timeout", 0, "How long to wait for mission status to match the mission step")

	missionSetStepCmd := &cobra.Command{
		Use:          "set-step",
		Short:        "Write a mission step control JSON file",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			controlFile, _ := cmd.Flags().GetString("control-file")
			if controlFile == "" {
				return fmt.Errorf("--control-file is required")
			}

			stepID, _ := cmd.Flags().GetString("step-id")
			if stepID == "" {
				return fmt.Errorf("--step-id is required")
			}

			missionFile, _ := cmd.Flags().GetString("mission-file")
			var expectedJobID string
			var confirmationExpectation *missionStatusAssertionExpectation
			if missionFile != "" {
				data, err := os.ReadFile(missionFile)
				if err != nil {
					return fmt.Errorf("failed to read mission file %q: %w", missionFile, err)
				}

				var job missioncontrol.Job
				if err := json.Unmarshal(data, &job); err != nil {
					return fmt.Errorf("failed to decode mission file %q: %w", missionFile, err)
				}

				if err := validateMissionStepSelection(job, stepID, resolveMissionStoreRoot(cmd)); err != nil {
					return fmt.Errorf("failed to validate mission file %q: %w", missionFile, err)
				}

				expectedJobID = job.ID
				expected, err := newMissionStatusAssertionForStep(job, stepID)
				if err != nil {
					return fmt.Errorf("failed to build mission status confirmation for %q: %w", missionFile, err)
				}
				confirmationExpectation = &expected
			}

			statusFile, _ := cmd.Flags().GetString("status-file")
			waitTimeout, _ := cmd.Flags().GetDuration("wait-timeout")
			if err := writeMissionStepControlAndConfirm(controlFile, statusFile, stepID, expectedJobID, confirmationExpectation, waitTimeout, cmd.Flags().Changed("wait-timeout"), nil); err != nil {
				return err
			}

			if statusFile != "" {
				log.Printf("mission set-step status confirmation succeeded job_id=%q step_id=%q control_file=%q status_file=%q", expectedJobID, stepID, controlFile, statusFile)
			}

			return nil
		},
	}
	missionSetStepCmd.Flags().String("control-file", "", "Path to write a mission step control JSON file")
	missionSetStepCmd.Flags().String("step-id", "", "Mission step ID to activate")
	missionSetStepCmd.Flags().String("mission-file", "", "Path to a mission job JSON file to validate before writing the mission step control file")
	missionSetStepCmd.Flags().String("status-file", "", "Path to a mission status snapshot JSON file to confirm after writing the mission step control file")
	missionSetStepCmd.Flags().Duration("wait-timeout", 0, "How long to wait for mission status confirmation after writing the mission step control file")

	missionPackageLogsCmd := &cobra.Command{
		Use:          "package-logs",
		Short:        "Package the active mission log segment under the durable store",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			storeRoot, _ := cmd.Flags().GetString("mission-store-root")
			if storeRoot == "" {
				return fmt.Errorf("--mission-store-root is required")
			}

			reasonValue, _ := cmd.Flags().GetString("reason")
			reason := missioncontrol.LogPackageReason(reasonValue)
			if err := missioncontrol.ValidateLogPackageReason(reason); err != nil {
				return err
			}

			hostname, _ := os.Hostname()
			result, err := missioncontrol.PackageCurrentLogSegment(
				storeRoot,
				reason,
				missioncontrol.WriterLockLease{
					LeaseHolderID: "picobot-mission-package-logs",
					PID:           os.Getpid(),
					Hostname:      hostname,
				},
				time.Now().UTC(),
			)
			if err != nil {
				return err
			}

			summary := missionPackageLogsSummary{
				Action:             "packaged",
				Reason:             result.Reason,
				PackageID:          result.PackageID,
				LogRelPath:         result.LogRelPath,
				ByteCount:          result.ByteCount,
				CurrentLogRelPath:  result.CurrentLogRelPath,
				CurrentMetaRelPath: result.CurrentMetaRelPath,
			}
			if result.NoOp {
				summary.Action = "noop"
				summary.NoOpCause = result.NoOpCause
			}

			data, err := json.MarshalIndent(summary, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to encode log packaging output: %w", err)
			}
			data = append(data, '\n')
			if _, err := cmd.OutOrStdout().Write(data); err != nil {
				return fmt.Errorf("failed to write log packaging output: %w", err)
			}
			return nil
		},
	}
	missionPackageLogsCmd.Flags().String("mission-store-root", "", "Path to the durable mission store root")
	missionPackageLogsCmd.Flags().String("reason", string(missioncontrol.LogPackageReasonManual), "Packaging reason: manual, daily, or reboot")

	missionPruneStoreCmd := &cobra.Command{
		Use:          "prune-store",
		Short:        "Prune expired retained mission store data",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			storeRoot, _ := cmd.Flags().GetString("mission-store-root")
			if storeRoot == "" {
				return fmt.Errorf("--mission-store-root is required")
			}

			hostname, _ := os.Hostname()
			result, err := missioncontrol.PruneStore(
				storeRoot,
				missioncontrol.WriterLockLease{
					LeaseHolderID: fmt.Sprintf("picobot-mission-prune-store-%d", os.Getpid()),
					PID:           os.Getpid(),
					Hostname:      hostname,
				},
				time.Now().UTC(),
			)
			if err != nil {
				return err
			}

			summary := missionPruneStoreSummary{
				Action:                     "pruned",
				StoreRoot:                  result.StoreRoot,
				PrunedPackageDirs:          result.PrunedPackageDirs,
				PrunedAuditFiles:           result.PrunedAuditFiles,
				PrunedApprovalRequestFiles: result.PrunedApprovalRequestFiles,
				PrunedApprovalGrantFiles:   result.PrunedApprovalGrantFiles,
				PrunedArtifactFiles:        result.PrunedArtifactFiles,
				SkippedNonterminalJobTrees: result.SkippedNonterminalJobTrees,
			}
			if result.PrunedPackageDirs == 0 &&
				result.PrunedAuditFiles == 0 &&
				result.PrunedApprovalRequestFiles == 0 &&
				result.PrunedApprovalGrantFiles == 0 &&
				result.PrunedArtifactFiles == 0 {
				summary.Action = "noop"
			}

			data, err := json.MarshalIndent(summary, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to encode prune-store output: %w", err)
			}
			data = append(data, '\n')
			if _, err := cmd.OutOrStdout().Write(data); err != nil {
				return fmt.Errorf("failed to write prune-store output: %w", err)
			}
			return nil
		},
	}
	missionPruneStoreCmd.Flags().String("mission-store-root", "", "Path to the durable mission store root")

	missionCmd.AddCommand(missionStatusCmd)
	missionCmd.AddCommand(missionInspectCmd)
	missionCmd.AddCommand(missionAssertCmd)
	missionCmd.AddCommand(missionAssertStepCmd)
	missionCmd.AddCommand(missionSetStepCmd)
	missionCmd.AddCommand(missionPackageLogsCmd)
	missionCmd.AddCommand(missionPruneStoreCmd)
	rootCmd.AddCommand(missionCmd)

	// memory subcommands: read, append, write, recent
	memoryCmd := &cobra.Command{
		Use:   "memory",
		Short: "Inspect or modify workspace memory files",
	}

	readCmd := &cobra.Command{
		Use:   "read [today|long]",
		Short: "Read memory (today or long-term)",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			target := args[0]
			cfg, _ := config.LoadConfig()
			ws := cfg.Agents.Defaults.Workspace
			if ws == "" {
				ws = "~/.picobot/workspace"
			}
			home, _ := os.UserHomeDir()
			if strings.HasPrefix(ws, "~/") {
				ws = filepath.Join(home, ws[2:])
			}
			mem := memory.NewMemoryStoreWithWorkspace(ws, 100)
			switch target {
			case "today":
				out, _ := mem.ReadToday()
				fmt.Fprintln(cmd.OutOrStdout(), out)
			case "long":
				out, _ := mem.ReadLongTerm()
				fmt.Fprintln(cmd.OutOrStdout(), out)
			default:
				fmt.Fprintln(cmd.ErrOrStderr(), "unknown target: "+target)
			}
		},
	}

	appendCmd := &cobra.Command{
		Use:   "append [today|long] -c <content>",
		Short: "Append content to today's note or long-term memory",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			target := args[0]
			content, _ := cmd.Flags().GetString("content")
			if content == "" {
				fmt.Fprintln(cmd.ErrOrStderr(), "-c content required")
				return
			}
			cfg, _ := config.LoadConfig()
			ws := cfg.Agents.Defaults.Workspace
			if ws == "" {
				ws = "~/.picobot/workspace"
			}
			home, _ := os.UserHomeDir()
			if strings.HasPrefix(ws, "~/") {
				ws = filepath.Join(home, ws[2:])
			}
			mem := memory.NewMemoryStoreWithWorkspace(ws, 100)
			switch target {
			case "today":
				if err := mem.AppendToday(content); err != nil {
					fmt.Fprintln(cmd.ErrOrStderr(), "append failed:", err)
					return
				}
				fmt.Fprintln(cmd.OutOrStdout(), "appended to today")
			case "long":
				lt, err := mem.ReadLongTerm()
				if err != nil {
					fmt.Fprintln(cmd.ErrOrStderr(), "append long failed:", err)
					return
				}
				if err := mem.WriteLongTerm(lt + "\n" + content); err != nil {
					fmt.Fprintln(cmd.ErrOrStderr(), "append long failed:", err)
					return
				}
				fmt.Fprintln(cmd.OutOrStdout(), "appended to long-term memory")
			default:
				fmt.Fprintln(cmd.ErrOrStderr(), "unknown target:", target)
			}
		},
	}
	appendCmd.Flags().StringP("content", "c", "", "Content to append")

	writeCmd := &cobra.Command{
		Use:   "write long -c <content>",
		Short: "Write (overwrite) long-term MEMORY.md",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if args[0] != "long" {
				fmt.Fprintln(os.Stderr, "write currently only supports 'long'")
				return
			}
			content, _ := cmd.Flags().GetString("content")
			if content == "" {
				fmt.Fprintln(cmd.ErrOrStderr(), "-c content required")
				return
			}
			cfg, _ := config.LoadConfig()
			ws := cfg.Agents.Defaults.Workspace
			if ws == "" {
				ws = "~/.picobot/workspace"
			}
			home, _ := os.UserHomeDir()
			if strings.HasPrefix(ws, "~/") {
				ws = filepath.Join(home, ws[2:])
			}
			mem := memory.NewMemoryStoreWithWorkspace(ws, 100)
			if err := mem.WriteLongTerm(content); err != nil {
				fmt.Fprintln(cmd.ErrOrStderr(), "write failed:", err)
				return
			}
			fmt.Fprintln(cmd.OutOrStdout(), "wrote long-term memory")
		},
	}
	writeCmd.Flags().StringP("content", "c", "", "Content to write")

	recentCmd := &cobra.Command{
		Use:   "recent -days N",
		Short: "Show recent N days' notes",
		Run: func(cmd *cobra.Command, args []string) {
			days, _ := cmd.Flags().GetInt("days")
			cfg, _ := config.LoadConfig()
			ws := cfg.Agents.Defaults.Workspace
			if ws == "" {
				ws = "~/.picobot/workspace"
			}
			home, _ := os.UserHomeDir()
			if strings.HasPrefix(ws, "~/") {
				ws = filepath.Join(home, ws[2:])
			}
			mem := memory.NewMemoryStoreWithWorkspace(ws, 100)
			out, _ := mem.GetRecentMemories(days)
			fmt.Fprintln(cmd.OutOrStdout(), out)
		},
	}
	recentCmd.Flags().IntP("days", "d", 1, "Number of days to include")

	memoryCmd.AddCommand(readCmd)
	memoryCmd.AddCommand(appendCmd)
	memoryCmd.AddCommand(writeCmd)
	memoryCmd.AddCommand(recentCmd)

	// rank subcommand: rank recent memories by relevance to a query
	rankCmd := &cobra.Command{
		Use:   "rank -q <query>",
		Short: "Rank recent memories relative to a query",
		Run: func(cmd *cobra.Command, args []string) {
			q, _ := cmd.Flags().GetString("query")
			if q == "" {
				fmt.Fprintln(cmd.ErrOrStderr(), "-q query required")
				return
			}
			top, _ := cmd.Flags().GetInt("top")
			verbose, _ := cmd.Flags().GetBool("verbose")
			cfg, _ := config.LoadConfig()
			ws := cfg.Agents.Defaults.Workspace
			if ws == "" {
				ws = "~/.picobot/workspace"
			}
			home, _ := os.UserHomeDir()
			if strings.HasPrefix(ws, "~/") {
				ws = filepath.Join(home, ws[2:])
			}
			mem := memory.NewMemoryStoreWithWorkspace(ws, 100)
			// Build memory items from today's file (split into lines) and long-term memory
			items := make([]memory.MemoryItem, 0)
			if td, err := mem.ReadToday(); err == nil && td != "" {
				for _, line := range strings.Split(td, "\n") {
					line = strings.TrimSpace(line)
					if line == "" {
						continue
					}
					// strip leading timestamp [2026-02-07...] if present
					if idx := strings.Index(line, "] "); idx != -1 && strings.HasPrefix(line, "[") {
						line = strings.TrimSpace(line[idx+2:])
					}
					items = append(items, memory.MemoryItem{Kind: "today", Text: line})
				}
			}
			if lt, err := mem.ReadLongTerm(); err == nil && lt != "" {
				for _, line := range strings.Split(lt, "\n") {
					line = strings.TrimSpace(line)
					if line == "" {
						continue
					}
					items = append(items, memory.MemoryItem{Kind: "long", Text: line})
				}
			}
			provider := providers.NewProviderFromConfig(cfg)
			var logger *log.Logger
			if verbose {
				logger = log.New(cmd.OutOrStdout(), "ranker: ", 0)
			}
			ranker := memory.NewLLMRankerWithLogger(provider, provider.GetDefaultModel(), logger)
			res := ranker.Rank(q, items, top)
			for i, m := range res {
				fmt.Fprintf(cmd.OutOrStdout(), "%d: %s (%s)\n", i+1, m.Text, m.Kind)
			}
		},
	}
	rankCmd.Flags().StringP("query", "q", "", "Query to rank memories against")
	rankCmd.Flags().IntP("top", "k", 5, "Number of top memories to show")
	rankCmd.Flags().BoolP("verbose", "v", false, "Enable verbose diagnostic logging (to stdout)")
	memoryCmd.AddCommand(rankCmd)

	rootCmd.AddCommand(memoryCmd)

	wrapCommandRunEWithSurfacedValidationErrors(rootCmd)
	return rootCmd
}

func wrapCommandRunEWithSurfacedValidationErrors(cmd *cobra.Command) {
	if cmd == nil {
		return
	}

	if cmd.RunE != nil {
		originalRunE := cmd.RunE
		cmd.RunE = func(cmd *cobra.Command, args []string) error {
			return missioncontrol.SurfaceValidationError(originalRunE(cmd, args))
		}
	}

	for _, subcommand := range cmd.Commands() {
		wrapCommandRunEWithSurfacedValidationErrors(subcommand)
	}
}

func addMissionBootstrapFlags(cmd *cobra.Command) {
	cmd.Flags().Bool("mission-required", false, "Require an active mission step before tool execution")
	cmd.Flags().String("mission-file", "", "Path to a mission job JSON file to activate at startup")
	cmd.Flags().String("mission-step", "", "Mission step ID to activate from the mission file")
	cmd.Flags().String("mission-store-root", "", "Path to the durable mission store root")
	cmd.Flags().String("mission-status-file", "", "Path to write a mission status snapshot after startup")
	cmd.Flags().Bool("mission-resume-approved", false, "Approve resuming a persisted mission runtime after reboot")
}

func resolveMissionStoreRoot(cmd *cobra.Command) string {
	if cmd == nil {
		return ""
	}

	storeRoot, _ := cmd.Flags().GetString("mission-store-root")
	statusFile, _ := cmd.Flags().GetString("mission-status-file")
	return missioncontrol.ResolveStoreRoot(storeRoot, statusFile)
}

func configureGatewayMissionStoreLogging(cmd *cobra.Command) (string, missioncontrol.WriterLockLease, func(), error) {
	storeRoot := resolveMissionStoreRoot(cmd)
	if storeRoot == "" {
		return "", missioncontrol.WriterLockLease{}, func() {}, nil
	}

	lease := ensureMissionStoreWriterLease(cmd)
	if _, err := packageCurrentLogSegmentOnGatewayStartup(storeRoot, lease, gatewayLogNow().UTC()); err != nil {
		return "", missioncontrol.WriterLockLease{}, nil, err
	}

	writer, err := newStoreLogWriter(storeRoot, gatewayLogNow().UTC())
	if err != nil {
		return "", missioncontrol.WriterLockLease{}, nil, err
	}

	originalWriter := log.Writer()
	log.SetOutput(writer)
	return storeRoot, lease, func() { log.SetOutput(originalWriter) }, nil
}

func ensureMissionStoreWriterLease(cmd *cobra.Command) missioncontrol.WriterLockLease {
	hostname, _ := os.Hostname()
	if cmd != nil {
		if cmd.Annotations == nil {
			cmd.Annotations = make(map[string]string)
		}
		if holder := strings.TrimSpace(cmd.Annotations[missionStoreWriterLeaseHolderAnnotation]); holder != "" {
			return missioncontrol.WriterLockLease{
				LeaseHolderID: holder,
				PID:           os.Getpid(),
				Hostname:      hostname,
			}
		}
		holder := fmt.Sprintf("picobot-%d-%d", os.Getpid(), time.Now().UTC().UnixNano())
		cmd.Annotations[missionStoreWriterLeaseHolderAnnotation] = holder
		return missioncontrol.WriterLockLease{
			LeaseHolderID: holder,
			PID:           os.Getpid(),
			Hostname:      hostname,
		}
	}
	return missioncontrol.WriterLockLease{
		LeaseHolderID: fmt.Sprintf("picobot-%d-%d", os.Getpid(), time.Now().UTC().UnixNano()),
		PID:           os.Getpid(),
		Hostname:      hostname,
	}
}

func watchGatewayLogDayRollover(ctx context.Context, storeRoot string, lease missioncontrol.WriterLockLease, interval time.Duration) {
	if strings.TrimSpace(storeRoot) == "" || interval <= 0 {
		return
	}

	ticker := newGatewayLogTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if _, err := packageCurrentLogSegmentOnUTCDayRollover(storeRoot, lease, gatewayLogNow().UTC()); err != nil {
				log.Printf("gateway log day rollover packaging failed for %q: %v", storeRoot, err)
			}
		}
	}
}

func installMissionRuntimeChangeHook(cmd *cobra.Command, ag *agent.AgentLoop) {
	installMissionRuntimeChangeHookWithExtension(cmd, ag, nil)
}

func installMissionRuntimeChangeHookWithExtension(cmd *cobra.Command, ag *agent.AgentLoop, after func()) {
	if cmd == nil || ag == nil {
		return
	}

	statusFile, _ := cmd.Flags().GetString("mission-status-file")
	storeRoot := resolveMissionStoreRoot(cmd)
	ag.SetMissionStoreRoot(storeRoot)

	if storeRoot != "" {
		lease := ensureMissionStoreWriterLease(cmd)
		ag.SetMissionRuntimePersistHook(func(job *missioncontrol.Job, runtime missioncontrol.JobRuntimeState, control *missioncontrol.RuntimeControlContext) error {
			return missioncontrol.PersistProjectedRuntimeState(storeRoot, lease, job, runtime, control, time.Now().UTC())
		})
	} else {
		ag.SetMissionRuntimePersistHook(nil)
	}

	if statusFile == "" {
		ag.SetMissionRuntimeProjectionHook(nil)
		ag.SetMissionRuntimeChangeHook(after)
		return
	}

	if storeRoot != "" {
		missionRequired, _ := cmd.Flags().GetBool("mission-required")
		ag.SetMissionRuntimeProjectionHook(func(job *missioncontrol.Job, runtime missioncontrol.JobRuntimeState, _ *missioncontrol.RuntimeControlContext) error {
			jobID := runtime.JobID
			if strings.TrimSpace(jobID) == "" && job != nil {
				jobID = job.ID
			}
			return writeProjectedMissionStatusSnapshot(statusFile, missionStatusSnapshotMissionFile(cmd), storeRoot, missionRequired, jobID, time.Now())
		})
		ag.SetMissionRuntimeChangeHook(func() {
			defer func() {
				if after != nil {
					after()
				}
			}()

			jobID := ""
			if runtimeState, ok := ag.MissionRuntimeState(); ok {
				jobID = runtimeState.JobID
			}
			if jobID == "" {
				if ec, ok := ag.ActiveMissionStep(); ok && ec.Job != nil {
					jobID = ec.Job.ID
				}
			}
			if jobID != "" {
				if _, err := missioncontrol.LoadCommittedJobRuntimeRecord(storeRoot, jobID); err == nil {
					return
				} else if !errors.Is(err, missioncontrol.ErrJobRuntimeRecordNotFound) {
					log.Printf("mission runtime status snapshot durable availability check failed for %q: %v", statusFile, err)
					return
				}
			}
			if err := writeMissionStatusSnapshot(statusFile, missionStatusSnapshotMissionFile(cmd), ag, time.Now()); err != nil {
				log.Printf("mission runtime status snapshot update failed for %q: %v", statusFile, err)
			}
		})
		return
	}

	ag.SetMissionRuntimeProjectionHook(nil)
	ag.SetMissionRuntimeChangeHook(func() {
		if err := writeMissionStatusSnapshot(statusFile, missionStatusSnapshotMissionFile(cmd), ag, time.Now()); err != nil {
			log.Printf("mission runtime status snapshot update failed for %q: %v", statusFile, err)
		}
		if after != nil {
			after()
		}
	})
}

func installMissionOperatorSetStepHook(cmd *cobra.Command, ag *agent.AgentLoop, job *missioncontrol.Job, applySynchronously bool) {
	if ag == nil {
		return
	}
	ag.SetOperatorSetStepHook(newMissionOperatorSetStepHook(cmd, ag, job, applySynchronously, 0))
}

func newMissionOperatorSetStepHook(cmd *cobra.Command, ag *agent.AgentLoop, job *missioncontrol.Job, applySynchronously bool, waitTimeout time.Duration) func(string, string) (string, error) {
	return func(jobID string, stepID string) (string, error) {
		if cmd == nil || ag == nil || job == nil {
			return "", missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
				Message: "SET_STEP requires an active bootstrapped mission",
			}
		}

		controlFile, _ := cmd.Flags().GetString("mission-step-control-file")
		if controlFile == "" {
			return "", missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
				Message: "SET_STEP requires --mission-step-control-file",
			}
		}

		statusFile, _ := cmd.Flags().GetString("mission-status-file")
		if statusFile == "" {
			return "", missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
				Message: "SET_STEP requires --mission-status-file",
			}
		}

		if err := validateMissionOperatorSetStepBinding(ag, *job, jobID); err != nil {
			return "", err
		}

		missionFile := missionStatusSnapshotMissionFile(cmd)
		if err := validateMissionStepSelection(*job, stepID, resolveMissionStoreRoot(cmd)); err != nil {
			return "", fmt.Errorf("failed to validate mission file %q: %w", missionFile, err)
		}

		var apply func() error
		if applySynchronously {
			apply = func() error {
				_, _, err := applyMissionStepControlFile(cmd, ag, *job, controlFile)
				return err
			}
		}

		expected, err := newMissionStatusAssertionForStep(*job, stepID)
		if err != nil {
			return "", fmt.Errorf("failed to build mission status confirmation for %q: %w", missionStatusSnapshotMissionFile(cmd), err)
		}

		if err := writeMissionStepControlAndConfirm(controlFile, statusFile, stepID, jobID, &expected, waitTimeout, false, apply); err != nil {
			return "", err
		}

		return fmt.Sprintf("Set step job=%s step=%s.", jobID, stepID), nil
	}
}

func validateMissionOperatorSetStepBinding(ag *agent.AgentLoop, job missioncontrol.Job, jobID string) error {
	if ag == nil {
		return missioncontrol.ValidationError{
			Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
			Message: "operator command requires an active mission step",
		}
	}

	if ec, ok := ag.ActiveMissionStep(); ok && ec.Job != nil && ec.Runtime != nil {
		if ec.Job.ID != jobID {
			return missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeStepValidationFailed,
				Message: "operator command does not match the active job",
			}
		}
		return nil
	}

	runtimeState, ok := ag.MissionRuntimeState()
	if !ok {
		return missioncontrol.ValidationError{
			Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
			Message: "operator command requires an active mission step",
		}
	}
	if runtimeState.JobID != "" && runtimeState.JobID != jobID {
		return missioncontrol.ValidationError{
			Code:    missioncontrol.RejectionCodeStepValidationFailed,
			Message: "operator command does not match the active job",
		}
	}
	if runtimeState.JobID == "" && job.ID != jobID {
		return missioncontrol.ValidationError{
			Code:    missioncontrol.RejectionCodeStepValidationFailed,
			Message: "operator command does not match the active job",
		}
	}
	if missioncontrol.IsTerminalJobState(runtimeState.State) {
		return missioncontrol.ValidationError{
			Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
			Message: fmt.Sprintf("cannot activate a new step while job is %q", runtimeState.State),
		}
	}
	if runtimeState.ActiveStepID == "" {
		return missioncontrol.ValidationError{
			Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
			Message: "operator command requires an active mission step",
		}
	}
	if control, hasControl := ag.MissionRuntimeControl(); hasControl && control.JobID != "" && control.JobID != jobID {
		return missioncontrol.ValidationError{
			Code:    missioncontrol.RejectionCodeStepValidationFailed,
			Message: "operator command does not match the active job",
		}
	}
	return nil
}

func writeMissionStepControlAndConfirm(controlFile string, statusFile string, stepID string, expectedJobID string, expected *missionStatusAssertionExpectation, waitTimeout time.Duration, waitTimeoutExplicit bool, apply func() error) error {
	var previousStatusUpdatedAt string
	if statusFile != "" {
		if snapshot, err := loadMissionStatusSnapshot(statusFile); err == nil {
			previousStatusUpdatedAt = snapshot.UpdatedAt
		}
	}

	control := missionStepControlFile{
		StepID:    stepID,
		UpdatedAt: time.Now().UTC().Format(time.RFC3339Nano),
	}

	if err := writeJSONAtomic(controlFile, control, "failed to encode mission step control file", "failed to write mission step control file"); err != nil {
		return err
	}
	if apply != nil {
		if err := apply(); err != nil {
			return err
		}
	}

	if statusFile == "" {
		return nil
	}
	if !waitTimeoutExplicit && waitTimeout <= 0 {
		waitTimeout = 5 * time.Second
	}

	return waitForMissionStatusStepConfirmation(statusFile, stepID, expectedJobID, expected, previousStatusUpdatedAt, waitTimeout)
}

func writeJSONAtomic(path string, value any, encodeErrPrefix string, writeErrPrefix string) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("%s %q: %w", encodeErrPrefix, path, err)
	}
	data = append(data, '\n')

	if err := writeJSONBytesAtomic(path, data); err != nil {
		return fmt.Errorf("%s %q: %w", writeErrPrefix, path, err)
	}

	return nil
}

func writeJSONBytesAtomic(path string, data []byte) (err error) {
	dir := filepath.Dir(path)
	tempFile, err := os.CreateTemp(dir, filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}

	tempPath := tempFile.Name()
	defer func() {
		if err == nil {
			return
		}
		if closeErr := tempFile.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
		_ = os.Remove(tempPath)
	}()

	if _, err = tempFile.Write(data); err != nil {
		return err
	}
	if err = tempFile.Close(); err != nil {
		return err
	}
	if err = os.Rename(tempPath, path); err != nil {
		return err
	}

	return nil
}

func configureMissionBootstrap(cmd *cobra.Command, ag *agent.AgentLoop) error {
	_, err := configureMissionBootstrapJob(cmd, ag)
	return err
}

func configureMissionBootstrapJob(cmd *cobra.Command, ag *agent.AgentLoop) (*missioncontrol.Job, error) {
	missionRequired, _ := cmd.Flags().GetBool("mission-required")
	missionFile, _ := cmd.Flags().GetString("mission-file")
	missionStep, _ := cmd.Flags().GetString("mission-step")

	if missionRequired {
		ag.SetMissionRequired(true)
	}

	if missionStep != "" && missionFile == "" {
		return nil, fmt.Errorf("--mission-step requires --mission-file")
	}

	controlFile, _ := cmd.Flags().GetString("mission-step-control-file")
	if controlFile != "" && missionFile == "" {
		return nil, fmt.Errorf("--mission-step-control-file requires --mission-file")
	}

	if missionFile == "" {
		return nil, nil
	}

	if missionStep == "" {
		return nil, fmt.Errorf("--mission-file requires --mission-step")
	}

	statusFile, _ := cmd.Flags().GetString("mission-status-file")
	storeRoot := resolveMissionStoreRoot(cmd)
	resumeApproved, _ := cmd.Flags().GetBool("mission-resume-approved")
	job, err := loadMissionJobFile(missionFile)
	if err != nil {
		return nil, err
	}

	if runtimeState, runtimeControl, sourcePath, ok, err := loadPersistedMissionRuntime(statusFile, storeRoot, job, time.Now().UTC()); err != nil {
		return nil, err
	} else if ok {
		if runtimeState.ActiveStepID != "" && runtimeState.ActiveStepID != missionStep {
			return nil, fmt.Errorf("persisted mission runtime step %q does not match --mission-step %q", runtimeState.ActiveStepID, missionStep)
		}
		if resumeApproved {
			if err := ag.ResumeMissionRuntime(job, runtimeState, runtimeControl, true); err != nil {
				return nil, fmt.Errorf("failed to resume persisted mission runtime from %q: %w", sourcePath, err)
			}
			return &job, nil
		}
		switch runtimeState.State {
		case missioncontrol.JobStatePaused, missioncontrol.JobStateWaitingUser:
			if err := ag.HydrateMissionRuntimeControl(job, runtimeState, runtimeControl); err != nil {
				return nil, fmt.Errorf("failed to rehydrate persisted mission runtime control from %q: %w", sourcePath, err)
			}
			return &job, nil
		case missioncontrol.JobStateCompleted, missioncontrol.JobStateFailed, missioncontrol.JobStateRejected, missioncontrol.JobStateAborted:
			if err := ag.HydrateMissionRuntimeControl(job, runtimeState, runtimeControl); err != nil {
				return nil, fmt.Errorf("failed to rehydrate persisted mission runtime terminal state from %q: %w", sourcePath, err)
			}
			return &job, nil
		default:
			return nil, missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeResumeApprovalRequired,
				Message: "persisted mission runtime requires --mission-resume-approved before resuming after reboot",
			}
		}
	}

	if err := ag.ActivateMissionStep(job, missionStep); err != nil {
		return nil, fmt.Errorf("failed to activate mission step %q from %q: %w", missionStep, missionFile, err)
	}

	return &job, nil
}

type missionStatusSnapshot = missioncontrol.MissionStatusSnapshot

// missionStatusFrankZohoSendProofLocator is the dedicated provider-specific
// CLI output contract for `mission status --frank-zoho-send-proof`.
type missionStatusFrankZohoSendProofLocator struct {
	ProviderMessageID  string `json:"provider_message_id"`
	ProviderMailID     string `json:"provider_mail_id,omitempty"`
	MIMEMessageID      string `json:"mime_message_id,omitempty"`
	ProviderAccountID  string `json:"provider_account_id"`
	OriginalMessageURL string `json:"original_message_url"`
}

// missionStatusFrankZohoSendProofVerification is the dedicated provider-specific
// CLI output contract for `mission status --frank-zoho-verify-send-proof`.
type missionStatusFrankZohoSendProofVerification = agenttools.FrankZohoSendProofVerification

type missionStatusFrankZohoSendProofVerifier interface {
	Verify(context.Context, []missioncontrol.OperatorFrankZohoSendProofStatus) ([]missionStatusFrankZohoSendProofVerification, error)
}

type missionStatusFrankZohoSendProofVerifierFunc func(context.Context, []missioncontrol.OperatorFrankZohoSendProofStatus) ([]missionStatusFrankZohoSendProofVerification, error)

func (fn missionStatusFrankZohoSendProofVerifierFunc) Verify(ctx context.Context, proof []missioncontrol.OperatorFrankZohoSendProofStatus) ([]missionStatusFrankZohoSendProofVerification, error) {
	return fn(ctx, proof)
}

type missionStepControlFile struct {
	StepID    string `json:"step_id"`
	UpdatedAt string `json:"updated_at"`
}

type missionStatusAssertionExpectation struct {
	JobID                  *string
	StepID                 *string
	Active                 *bool
	StepType               *string
	RequiredAuthority      *missioncontrol.AuthorityTier
	RequiresApproval       *bool
	NoTools                bool
	HasTools               []string
	ExactAllowedTools      []string
	CheckExactAllowedTools bool
}

type missionPackageLogsSummary struct {
	Action             string                          `json:"action"`
	Reason             missioncontrol.LogPackageReason `json:"reason"`
	NoOpCause          string                          `json:"noop_cause,omitempty"`
	PackageID          string                          `json:"package_id,omitempty"`
	LogRelPath         string                          `json:"log_relpath,omitempty"`
	ByteCount          int64                           `json:"byte_count,omitempty"`
	CurrentLogRelPath  string                          `json:"current_log_relpath"`
	CurrentMetaRelPath string                          `json:"current_meta_relpath"`
}

type missionPruneStoreSummary struct {
	Action                     string `json:"action"`
	StoreRoot                  string `json:"store_root"`
	PrunedPackageDirs          int    `json:"pruned_package_dirs"`
	PrunedAuditFiles           int    `json:"pruned_audit_files"`
	PrunedApprovalRequestFiles int    `json:"pruned_approval_request_files"`
	PrunedApprovalGrantFiles   int    `json:"pruned_approval_grant_files"`
	PrunedArtifactFiles        int    `json:"pruned_artifact_files"`
	SkippedNonterminalJobTrees int    `json:"skipped_nonterminal_job_trees"`
}

type missionInspectSummary = missioncontrol.InspectSummary
type missionInspectNotificationsCapability = missioncontrol.CapabilityRecord
type missionInspectSharedStorageCapability = missioncontrol.CapabilityRecord
type missionInspectContactsCapability struct {
	Capability missioncontrol.CapabilityRecord     `json:"capability"`
	Source     missioncontrol.ContactsSourceRecord `json:"source"`
}
type missionInspectLocationCapability struct {
	Capability missioncontrol.CapabilityRecord     `json:"capability"`
	Source     missioncontrol.LocationSourceRecord `json:"source"`
}
type missionInspectCameraCapability struct {
	Capability missioncontrol.CapabilityRecord   `json:"capability"`
	Source     missioncontrol.CameraSourceRecord `json:"source"`
}
type missionInspectMicrophoneCapability struct {
	Capability missioncontrol.CapabilityRecord       `json:"capability"`
	Source     missioncontrol.MicrophoneSourceRecord `json:"source"`
}
type missionInspectSMSPhoneCapability struct {
	Capability missioncontrol.CapabilityRecord     `json:"capability"`
	Source     missioncontrol.SMSPhoneSourceRecord `json:"source"`
}
type missionInspectBluetoothNFCCapability struct {
	Capability missioncontrol.CapabilityRecord         `json:"capability"`
	Source     missioncontrol.BluetoothNFCSourceRecord `json:"source"`
}
type missionInspectBroadAppControlCapability struct {
	Capability missioncontrol.CapabilityRecord            `json:"capability"`
	Source     missioncontrol.BroadAppControlSourceRecord `json:"source"`
}

var loadValidatedLegacyMissionStatusSnapshot = missioncontrol.LoadValidatedLegacyMissionStatusSnapshot
var loadGatewayStatusObservation = missioncontrol.LoadGatewayStatusObservation
var loadGatewayStatusObservationFile = missioncontrol.LoadGatewayStatusObservationFile
var loadMissionStatusObservation = missioncontrol.LoadMissionStatusObservation
var loadMissionStatusObservationFile = missioncontrol.LoadMissionStatusObservationFile
var writeMissionStatusSnapshotAtomic = missioncontrol.WriteMissionStatusSnapshotAtomic
var newFrankZohoSendProofVerifier = func() missionStatusFrankZohoSendProofVerifier {
	return agenttools.NewFrankZohoSendProofVerifier()
}

func loadMissionStatusFrankZohoSendProofFile(path string) ([]byte, error) {
	snapshot, err := loadMissionStatusObservation(path)
	if err != nil {
		return nil, err
	}

	// This dedicated helper surface consumes only committed
	// runtime_summary.frank_zoho_send_proof and does not fall back to raw runtime
	// receipts.
	proof := make([]missionStatusFrankZohoSendProofLocator, 0)
	if snapshot.RuntimeSummary != nil {
		proof = make([]missionStatusFrankZohoSendProofLocator, 0, len(snapshot.RuntimeSummary.FrankZohoSendProof))
		for _, candidate := range snapshot.RuntimeSummary.FrankZohoSendProof {
			proof = append(proof, missionStatusFrankZohoSendProofLocator{
				ProviderMessageID:  candidate.ProviderMessageID,
				ProviderMailID:     candidate.ProviderMailID,
				MIMEMessageID:      candidate.MIMEMessageID,
				ProviderAccountID:  candidate.ProviderAccountID,
				OriginalMessageURL: candidate.OriginalMessageURL,
			})
		}
	}

	data, err := json.MarshalIndent(proof, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to encode mission status Frank Zoho send proof for %q: %w", path, err)
	}
	data = append(data, '\n')
	return data, nil
}

func loadMissionStatusFrankZohoVerifiedSendProofFile(ctx context.Context, path string) ([]byte, error) {
	snapshot, err := loadMissionStatusObservation(path)
	if err != nil {
		return nil, err
	}

	// This dedicated helper surface consumes only committed
	// runtime_summary.frank_zoho_send_proof and does not fall back to raw runtime
	// receipts.
	proof := make([]missioncontrol.OperatorFrankZohoSendProofStatus, 0)
	if snapshot.RuntimeSummary != nil {
		proof = append(proof, snapshot.RuntimeSummary.FrankZohoSendProof...)
	}

	verified, err := newFrankZohoSendProofVerifier().Verify(ctx, proof)
	if err != nil {
		return nil, fmt.Errorf("failed to verify mission status Frank Zoho send proof for %q: %w", path, err)
	}

	data, err := json.MarshalIndent(verified, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to encode mission status Frank Zoho send proof verification for %q: %w", path, err)
	}
	data = append(data, '\n')
	return data, nil
}

func validateMissionJob(job missioncontrol.Job, storeRoot string) error {
	job.MissionStoreRoot = storeRoot
	if validationErrors := missioncontrol.ValidatePlan(job); len(validationErrors) > 0 {
		return validationErrors[0]
	}
	return nil
}

func validateMissionStepSelection(job missioncontrol.Job, stepID string, storeRoot string) error {
	if err := validateMissionJob(job, storeRoot); err != nil {
		return err
	}

	for _, step := range job.Plan.Steps {
		if step.ID == stepID {
			return nil
		}
	}

	return missioncontrol.ValidationError{
		Code:    missioncontrol.RejectionCodeUnknownStep,
		StepID:  stepID,
		Message: fmt.Sprintf(`step %q not found in plan`, stepID),
	}
}

func loadMissionJobFile(path string) (missioncontrol.Job, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return missioncontrol.Job{}, fmt.Errorf("failed to read mission file %q: %w", path, err)
	}

	var job missioncontrol.Job
	if err := json.Unmarshal(data, &job); err != nil {
		return missioncontrol.Job{}, fmt.Errorf("failed to decode mission file %q: %w", path, err)
	}

	return job, nil
}

func loadPersistedMissionRuntime(path string, storeRoot string, job missioncontrol.Job, now time.Time) (missioncontrol.JobRuntimeState, *missioncontrol.RuntimeControlContext, string, bool, error) {
	if runtimeState, runtimeControl, ok, err := loadCommittedMissionRuntime(storeRoot, job, now); err != nil {
		return missioncontrol.JobRuntimeState{}, nil, "", false, err
	} else if ok {
		return runtimeState, runtimeControl, storeRoot, true, nil
	}

	if runtimeState, runtimeControl, ok, err := loadPersistedMissionRuntimeSnapshot(path, job); err != nil {
		return missioncontrol.JobRuntimeState{}, nil, "", false, err
	} else if ok {
		return runtimeState, runtimeControl, path, true, nil
	}

	return missioncontrol.JobRuntimeState{}, nil, "", false, nil
}

func loadCommittedMissionRuntime(storeRoot string, job missioncontrol.Job, now time.Time) (missioncontrol.JobRuntimeState, *missioncontrol.RuntimeControlContext, bool, error) {
	if strings.TrimSpace(storeRoot) == "" {
		return missioncontrol.JobRuntimeState{}, nil, false, nil
	}

	runtimeState, err := missioncontrol.HydrateCommittedJobRuntimeState(storeRoot, job.ID, now)
	if err != nil {
		if errors.Is(err, missioncontrol.ErrJobRuntimeRecordNotFound) {
			return missioncontrol.JobRuntimeState{}, nil, false, nil
		}
		return missioncontrol.JobRuntimeState{}, nil, false, fmt.Errorf("failed to hydrate committed mission runtime from durable store %q: %w", storeRoot, err)
	}

	runtimeControl, err := missioncontrol.HydrateCommittedRuntimeControlContext(storeRoot, job.ID, now)
	if err != nil {
		return missioncontrol.JobRuntimeState{}, nil, false, fmt.Errorf("failed to hydrate committed mission runtime control from durable store %q: %w", storeRoot, err)
	}

	return runtimeState, runtimeControl, true, nil
}

func loadPersistedMissionRuntimeSnapshot(path string, job missioncontrol.Job) (missioncontrol.JobRuntimeState, *missioncontrol.RuntimeControlContext, bool, error) {
	snapshot, ok, err := loadValidatedLegacyMissionStatusSnapshot(path, job.ID)
	if err != nil {
		return missioncontrol.JobRuntimeState{}, nil, false, err
	}
	if !ok {
		return missioncontrol.JobRuntimeState{}, nil, false, nil
	}
	var runtimeControl *missioncontrol.RuntimeControlContext
	if snapshot.RuntimeControl != nil {
		runtimeControl = missioncontrol.CloneRuntimeControlContext(snapshot.RuntimeControl)
	}
	return *missioncontrol.CloneJobRuntimeState(snapshot.Runtime), runtimeControl, true, nil
}

func newMissionInspectSummary(job missioncontrol.Job, stepID string, storeRoot string) (missionInspectSummary, error) {
	return missioncontrol.NewInspectSummaryWithCampaignAndTreasuryPreflight(job, stepID, storeRoot)
}

func newMissionInspectNotificationsCapability(job missioncontrol.Job, stepID string, storeRoot string) (missionInspectNotificationsCapability, error) {
	job.MissionStoreRoot = strings.TrimSpace(storeRoot)
	if job.MissionStoreRoot == "" {
		return missionInspectNotificationsCapability{}, fmt.Errorf("mission store root is required")
	}

	if stepID != "" {
		ec, err := missioncontrol.ResolveExecutionContext(job, stepID)
		if err != nil {
			return missionInspectNotificationsCapability{}, err
		}
		if ec.Step == nil || !missioncontrol.StepRequiresNotificationsCapability(*ec.Step) {
			return missionInspectNotificationsCapability{}, fmt.Errorf("step %q does not require notifications capability", stepID)
		}
	}

	record, err := missioncontrol.ResolveNotificationsCapabilityRecord(job.MissionStoreRoot)
	if err != nil {
		return missionInspectNotificationsCapability{}, err
	}
	return *record, nil
}

func newMissionInspectSharedStorageCapability(job missioncontrol.Job, stepID string, storeRoot string) (missionInspectSharedStorageCapability, error) {
	job.MissionStoreRoot = strings.TrimSpace(storeRoot)
	if job.MissionStoreRoot == "" {
		return missionInspectSharedStorageCapability{}, fmt.Errorf("mission store root is required")
	}

	if stepID != "" {
		ec, err := missioncontrol.ResolveExecutionContext(job, stepID)
		if err != nil {
			return missionInspectSharedStorageCapability{}, err
		}
		if ec.Step == nil || !missioncontrol.StepRequiresSharedStorageCapability(*ec.Step) {
			return missionInspectSharedStorageCapability{}, fmt.Errorf("step %q does not require shared_storage capability", stepID)
		}
	}

	record, err := missioncontrol.ResolveSharedStorageCapabilityRecord(job.MissionStoreRoot)
	if err != nil {
		return missionInspectSharedStorageCapability{}, err
	}
	return *record, nil
}

func newMissionInspectContactsCapability(job missioncontrol.Job, stepID string, storeRoot string) (missionInspectContactsCapability, error) {
	job.MissionStoreRoot = strings.TrimSpace(storeRoot)
	if job.MissionStoreRoot == "" {
		return missionInspectContactsCapability{}, fmt.Errorf("mission store root is required")
	}

	if stepID != "" {
		ec, err := missioncontrol.ResolveExecutionContext(job, stepID)
		if err != nil {
			return missionInspectContactsCapability{}, err
		}
		if ec.Step == nil || !missioncontrol.StepRequiresContactsCapability(*ec.Step) {
			return missionInspectContactsCapability{}, fmt.Errorf("step %q does not require contacts capability", stepID)
		}
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		return missionInspectContactsCapability{}, fmt.Errorf("contacts capability inspection requires readable config: %w", err)
	}

	capability, err := missioncontrol.ResolveContactsCapabilityRecord(job.MissionStoreRoot)
	if err != nil {
		return missionInspectContactsCapability{}, err
	}
	source, err := missioncontrol.RequireReadableContactsSourceRecord(job.MissionStoreRoot, cfg.Agents.Defaults.Workspace)
	if err != nil {
		return missionInspectContactsCapability{}, err
	}
	return missionInspectContactsCapability{
		Capability: *capability,
		Source:     *source,
	}, nil
}

func newMissionInspectLocationCapability(job missioncontrol.Job, stepID string, storeRoot string) (missionInspectLocationCapability, error) {
	job.MissionStoreRoot = strings.TrimSpace(storeRoot)
	if job.MissionStoreRoot == "" {
		return missionInspectLocationCapability{}, fmt.Errorf("mission store root is required")
	}

	if stepID != "" {
		ec, err := missioncontrol.ResolveExecutionContext(job, stepID)
		if err != nil {
			return missionInspectLocationCapability{}, err
		}
		if ec.Step == nil || !missioncontrol.StepRequiresLocationCapability(*ec.Step) {
			return missionInspectLocationCapability{}, fmt.Errorf("step %q does not require location capability", stepID)
		}
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		return missionInspectLocationCapability{}, fmt.Errorf("location capability inspection requires readable config: %w", err)
	}

	capability, err := missioncontrol.ResolveLocationCapabilityRecord(job.MissionStoreRoot)
	if err != nil {
		return missionInspectLocationCapability{}, err
	}
	source, err := missioncontrol.RequireReadableLocationSourceRecord(job.MissionStoreRoot, cfg.Agents.Defaults.Workspace)
	if err != nil {
		return missionInspectLocationCapability{}, err
	}
	return missionInspectLocationCapability{
		Capability: *capability,
		Source:     *source,
	}, nil
}

func newMissionInspectCameraCapability(job missioncontrol.Job, stepID string, storeRoot string) (missionInspectCameraCapability, error) {
	job.MissionStoreRoot = strings.TrimSpace(storeRoot)
	if job.MissionStoreRoot == "" {
		return missionInspectCameraCapability{}, fmt.Errorf("mission store root is required")
	}

	if stepID != "" {
		ec, err := missioncontrol.ResolveExecutionContext(job, stepID)
		if err != nil {
			return missionInspectCameraCapability{}, err
		}
		if ec.Step == nil || !missioncontrol.StepRequiresCameraCapability(*ec.Step) {
			return missionInspectCameraCapability{}, fmt.Errorf("step %q does not require camera capability", stepID)
		}
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		return missionInspectCameraCapability{}, fmt.Errorf("camera capability inspection requires readable config: %w", err)
	}

	capability, err := missioncontrol.ResolveCameraCapabilityRecord(job.MissionStoreRoot)
	if err != nil {
		return missionInspectCameraCapability{}, err
	}
	source, err := missioncontrol.RequireReadableCameraSourceRecord(job.MissionStoreRoot, cfg.Agents.Defaults.Workspace)
	if err != nil {
		return missionInspectCameraCapability{}, err
	}
	return missionInspectCameraCapability{
		Capability: *capability,
		Source:     *source,
	}, nil
}

func newMissionInspectMicrophoneCapability(job missioncontrol.Job, stepID string, storeRoot string) (missionInspectMicrophoneCapability, error) {
	job.MissionStoreRoot = strings.TrimSpace(storeRoot)
	if job.MissionStoreRoot == "" {
		return missionInspectMicrophoneCapability{}, fmt.Errorf("mission store root is required")
	}

	if stepID != "" {
		ec, err := missioncontrol.ResolveExecutionContext(job, stepID)
		if err != nil {
			return missionInspectMicrophoneCapability{}, err
		}
		if ec.Step == nil || !missioncontrol.StepRequiresMicrophoneCapability(*ec.Step) {
			return missionInspectMicrophoneCapability{}, fmt.Errorf("step %q does not require microphone capability", stepID)
		}
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		return missionInspectMicrophoneCapability{}, fmt.Errorf("microphone capability inspection requires readable config: %w", err)
	}

	capability, err := missioncontrol.ResolveMicrophoneCapabilityRecord(job.MissionStoreRoot)
	if err != nil {
		return missionInspectMicrophoneCapability{}, err
	}
	source, err := missioncontrol.RequireReadableMicrophoneSourceRecord(job.MissionStoreRoot, cfg.Agents.Defaults.Workspace)
	if err != nil {
		return missionInspectMicrophoneCapability{}, err
	}
	return missionInspectMicrophoneCapability{
		Capability: *capability,
		Source:     *source,
	}, nil
}

func newMissionInspectSMSPhoneCapability(job missioncontrol.Job, stepID string, storeRoot string) (missionInspectSMSPhoneCapability, error) {
	job.MissionStoreRoot = strings.TrimSpace(storeRoot)
	if job.MissionStoreRoot == "" {
		return missionInspectSMSPhoneCapability{}, fmt.Errorf("mission store root is required")
	}

	if stepID != "" {
		ec, err := missioncontrol.ResolveExecutionContext(job, stepID)
		if err != nil {
			return missionInspectSMSPhoneCapability{}, err
		}
		if ec.Step == nil || !missioncontrol.StepRequiresSMSPhoneCapability(*ec.Step) {
			return missionInspectSMSPhoneCapability{}, fmt.Errorf("step %q does not require sms_phone capability", stepID)
		}
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		return missionInspectSMSPhoneCapability{}, fmt.Errorf("sms_phone capability inspection requires readable config: %w", err)
	}

	capability, err := missioncontrol.ResolveSMSPhoneCapabilityRecord(job.MissionStoreRoot)
	if err != nil {
		return missionInspectSMSPhoneCapability{}, err
	}
	source, err := missioncontrol.RequireReadableSMSPhoneSourceRecord(job.MissionStoreRoot, cfg.Agents.Defaults.Workspace)
	if err != nil {
		return missionInspectSMSPhoneCapability{}, err
	}
	return missionInspectSMSPhoneCapability{
		Capability: *capability,
		Source:     *source,
	}, nil
}

func newMissionInspectBluetoothNFCCapability(job missioncontrol.Job, stepID string, storeRoot string) (missionInspectBluetoothNFCCapability, error) {
	job.MissionStoreRoot = strings.TrimSpace(storeRoot)
	if job.MissionStoreRoot == "" {
		return missionInspectBluetoothNFCCapability{}, fmt.Errorf("mission store root is required")
	}

	if stepID != "" {
		ec, err := missioncontrol.ResolveExecutionContext(job, stepID)
		if err != nil {
			return missionInspectBluetoothNFCCapability{}, err
		}
		if ec.Step == nil || !missioncontrol.StepRequiresBluetoothNFCCapability(*ec.Step) {
			return missionInspectBluetoothNFCCapability{}, fmt.Errorf("step %q does not require bluetooth_nfc capability", stepID)
		}
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		return missionInspectBluetoothNFCCapability{}, fmt.Errorf("bluetooth_nfc capability inspection requires readable config: %w", err)
	}

	capability, err := missioncontrol.ResolveBluetoothNFCCapabilityRecord(job.MissionStoreRoot)
	if err != nil {
		return missionInspectBluetoothNFCCapability{}, err
	}
	source, err := missioncontrol.RequireReadableBluetoothNFCSourceRecord(job.MissionStoreRoot, cfg.Agents.Defaults.Workspace)
	if err != nil {
		return missionInspectBluetoothNFCCapability{}, err
	}
	return missionInspectBluetoothNFCCapability{
		Capability: *capability,
		Source:     *source,
	}, nil
}

func newMissionInspectBroadAppControlCapability(job missioncontrol.Job, stepID string, storeRoot string) (missionInspectBroadAppControlCapability, error) {
	job.MissionStoreRoot = strings.TrimSpace(storeRoot)
	if job.MissionStoreRoot == "" {
		return missionInspectBroadAppControlCapability{}, fmt.Errorf("mission store root is required")
	}

	if stepID != "" {
		ec, err := missioncontrol.ResolveExecutionContext(job, stepID)
		if err != nil {
			return missionInspectBroadAppControlCapability{}, err
		}
		if ec.Step == nil || !missioncontrol.StepRequiresBroadAppControlCapability(*ec.Step) {
			return missionInspectBroadAppControlCapability{}, fmt.Errorf("step %q does not require broad_app_control capability", stepID)
		}
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		return missionInspectBroadAppControlCapability{}, fmt.Errorf("broad_app_control capability inspection requires readable config: %w", err)
	}

	capability, err := missioncontrol.ResolveBroadAppControlCapabilityRecord(job.MissionStoreRoot)
	if err != nil {
		return missionInspectBroadAppControlCapability{}, err
	}
	source, err := missioncontrol.RequireReadableBroadAppControlSourceRecord(job.MissionStoreRoot, cfg.Agents.Defaults.Workspace)
	if err != nil {
		return missionInspectBroadAppControlCapability{}, err
	}
	return missionInspectBroadAppControlCapability{
		Capability: *capability,
		Source:     *source,
	}, nil
}

func activateMissionStepFromControlData(ag *agent.AgentLoop, job missioncontrol.Job, path string, data []byte) (string, bool, error) {
	var control missionStepControlFile
	if err := json.Unmarshal(data, &control); err != nil {
		return "", false, fmt.Errorf("failed to decode mission step control file %q: %w", path, err)
	}
	if control.StepID == "" {
		return "", false, fmt.Errorf("mission step control file %q is missing step_id", path)
	}
	if runtimeState, ok := ag.MissionRuntimeState(); ok {
		if runtimeState.JobID != "" && runtimeState.JobID != job.ID {
			return "", false, missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeStepValidationFailed,
				StepID:  control.StepID,
				Message: "operator command does not match the active job",
			}
		}
		if missioncontrol.HasCompletedRuntimeStep(runtimeState, control.StepID) {
			return "", false, missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
				StepID:  control.StepID,
				Message: fmt.Sprintf("step %q is already recorded as completed in runtime state", control.StepID),
			}
		}
		if missioncontrol.HasFailedRuntimeStep(runtimeState, control.StepID) {
			return "", false, missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
				StepID:  control.StepID,
				Message: fmt.Sprintf("step %q is already recorded as failed in runtime state", control.StepID),
			}
		}
	}

	if err := ag.ActivateMissionStep(job, control.StepID); err != nil {
		return "", false, fmt.Errorf("failed to activate mission step %q from control file %q: %w", control.StepID, path, err)
	}

	return control.StepID, true, nil
}

func activateMissionStepFromControlFile(ag *agent.AgentLoop, job missioncontrol.Job, path string) (string, bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", false, nil
		}
		return "", false, fmt.Errorf("failed to read mission step control file %q: %w", path, err)
	}

	return activateMissionStepFromControlData(ag, job, path, data)
}

func restoreMissionStepControlFileOnStartup(cmd *cobra.Command, ag *agent.AgentLoop, job missioncontrol.Job) []byte {
	controlFile, _ := cmd.Flags().GetString("mission-step-control-file")
	if controlFile == "" {
		return nil
	}

	data, err := os.ReadFile(controlFile)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("mission step control startup apply failed for %q: failed to read mission step control file %q: %v", controlFile, controlFile, err)
		}
		return nil
	}

	stepID, changed, err := activateMissionStepFromControlData(ag, job, controlFile, data)
	if err != nil {
		log.Printf("mission step control startup apply failed for %q: %v", controlFile, err)
		return nil
	}
	if changed {
		log.Printf("mission step control startup apply succeeded job_id=%q step_id=%q control_file=%q", job.ID, stepID, controlFile)
	}
	return data
}

// mission-step-control-file is an operator control input surface, not a derived
// runtime projection. Slice 3c intentionally leaves its file semantics unchanged.
func applyMissionStepControlFile(cmd *cobra.Command, ag *agent.AgentLoop, job missioncontrol.Job, path string) (string, bool, error) {
	stepID, changed, err := activateMissionStepFromControlFile(ag, job, path)
	if err != nil || !changed {
		return stepID, changed, err
	}

	if err := writeMissionStatusSnapshotFromCommand(cmd, ag, time.Now()); err != nil {
		return "", false, err
	}

	return stepID, true, nil
}

func watchMissionStepControlFile(ctx context.Context, cmd *cobra.Command, ag *agent.AgentLoop, job missioncontrol.Job, path string, interval time.Duration, baseline []byte) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	lastContent := append([]byte(nil), baseline...)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			data, err := os.ReadFile(path)
			if err != nil {
				if !os.IsNotExist(err) {
					log.Printf("mission step control watch failed for %q: %v", path, err)
				}
				lastContent = nil
				continue
			}
			if bytes.Equal(lastContent, data) {
				continue
			}
			lastContent = append(lastContent[:0], data...)
			stepID, changed, err := applyMissionStepControlFile(cmd, ag, job, path)
			if err != nil {
				log.Printf("mission step control apply failed for %q: %v", path, err)
				continue
			}
			if changed {
				log.Printf("mission step control apply succeeded job_id=%q step_id=%q control_file=%q", job.ID, stepID, path)
			}
		}
	}
}

func writeMissionStatusSnapshotFromCommand(cmd *cobra.Command, ag *agent.AgentLoop, now time.Time) error {
	statusFile, _ := cmd.Flags().GetString("mission-status-file")
	if statusFile == "" {
		return nil
	}

	storeRoot := resolveMissionStoreRoot(cmd)
	if storeRoot == "" {
		return writeMissionStatusSnapshot(statusFile, missionStatusSnapshotMissionFile(cmd), ag, now)
	}

	jobID := ""
	if runtimeState, ok := ag.MissionRuntimeState(); ok {
		jobID = runtimeState.JobID
	}
	if jobID == "" {
		if ec, ok := ag.ActiveMissionStep(); ok && ec.Job != nil {
			jobID = ec.Job.ID
		}
	}
	if jobID == "" {
		return writeMissionStatusSnapshot(statusFile, missionStatusSnapshotMissionFile(cmd), ag, now)
	}
	if _, err := missioncontrol.LoadCommittedJobRuntimeRecord(storeRoot, jobID); err != nil {
		if errors.Is(err, missioncontrol.ErrJobRuntimeRecordNotFound) {
			return writeMissionStatusSnapshot(statusFile, missionStatusSnapshotMissionFile(cmd), ag, now)
		}
		return fmt.Errorf("failed to inspect committed mission runtime for status snapshot in durable store %q: %w", storeRoot, err)
	}

	missionRequired, _ := cmd.Flags().GetBool("mission-required")
	return writeProjectedMissionStatusSnapshot(statusFile, missionStatusSnapshotMissionFile(cmd), storeRoot, missionRequired, jobID, now)
}

func missionStatusSnapshotMissionFile(cmd *cobra.Command) string {
	missionFile, _ := cmd.Flags().GetString("mission-file")
	return missionFile
}

func waitForMissionStatusStepConfirmation(path string, stepID string, expectedJobID string, expected *missionStatusAssertionExpectation, previousUpdatedAt string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	var lastErr error

	for {
		snapshot, err := loadMissionStatusSnapshot(path)
		if err != nil {
			lastErr = err
		} else if expected != nil {
			if err := checkMissionStatusAssertion(path, snapshot, *expected); err != nil {
				lastErr = err
			} else if previousUpdatedAt == "" || snapshot.UpdatedAt != previousUpdatedAt {
				return nil
			} else if expected.JobID != nil {
				lastErr = fmt.Errorf("mission status file %q has active=true step_id=%q job_id=%q updated_at=%q, want a fresh matching update with job_id=%q and updated_at different from %q", path, snapshot.StepID, snapshot.JobID, snapshot.UpdatedAt, *expected.JobID, previousUpdatedAt)
			} else {
				lastErr = fmt.Errorf("mission status file %q has active=true step_id=%q updated_at=%q, want a fresh matching update with updated_at different from %q", path, snapshot.StepID, snapshot.UpdatedAt, previousUpdatedAt)
			}
		} else if !snapshot.Active || snapshot.StepID != stepID {
			lastErr = fmt.Errorf("mission status file %q has active=%t step_id=%q, want active=true step_id=%q", path, snapshot.Active, snapshot.StepID, stepID)
		} else if expectedJobID != "" && snapshot.JobID != expectedJobID {
			lastErr = fmt.Errorf("mission status file %q has active=true step_id=%q job_id=%q, want active=true step_id=%q job_id=%q", path, snapshot.StepID, snapshot.JobID, stepID, expectedJobID)
		} else if previousUpdatedAt == "" || snapshot.UpdatedAt != previousUpdatedAt {
			return nil
		} else {
			if expectedJobID != "" {
				lastErr = fmt.Errorf("mission status file %q has active=true step_id=%q job_id=%q updated_at=%q, want a fresh matching update with job_id=%q and updated_at different from %q", path, snapshot.StepID, snapshot.JobID, snapshot.UpdatedAt, expectedJobID, previousUpdatedAt)
			} else {
				lastErr = fmt.Errorf("mission status file %q has active=true step_id=%q updated_at=%q, want a fresh matching update with updated_at different from %q", path, snapshot.StepID, snapshot.UpdatedAt, previousUpdatedAt)
			}
		}

		if remaining := time.Until(deadline); remaining <= 0 {
			return fmt.Errorf("timed out waiting up to %s for mission status file %q to confirm step %q: %w", timeout, path, stepID, lastErr)
		} else {
			sleep := 100 * time.Millisecond
			if remaining < sleep {
				sleep = remaining
			}
			time.Sleep(sleep)
		}
	}
}

func assertMissionStatusSnapshot(path string, expected missionStatusAssertionExpectation) error {
	snapshot, err := loadMissionStatusSnapshot(path)
	if err != nil {
		return err
	}
	return checkMissionStatusAssertion(path, snapshot, expected)
}

func assertMissionGatewayStatusSnapshot(path string, expected missionStatusAssertionExpectation) error {
	snapshot, err := loadGatewayStatusObservation(path)
	if err != nil {
		return err
	}
	return checkMissionStatusAssertion(path, projectGatewayStatusAssertionSnapshot(snapshot), expected)
}

func waitForMissionStatusAssertion(path string, expected missionStatusAssertionExpectation, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	var lastErr error

	for {
		lastErr = assertMissionStatusSnapshot(path, expected)
		if lastErr == nil {
			return nil
		}

		if remaining := time.Until(deadline); remaining <= 0 {
			return fmt.Errorf("timed out waiting up to %s for mission status file %q to satisfy assertion: %w", timeout, path, lastErr)
		} else {
			sleep := 100 * time.Millisecond
			if remaining < sleep {
				sleep = remaining
			}
			time.Sleep(sleep)
		}
	}
}

func waitForMissionGatewayStatusAssertion(path string, expected missionStatusAssertionExpectation, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	var lastErr error

	for {
		lastErr = assertMissionGatewayStatusSnapshot(path, expected)
		if lastErr == nil {
			return nil
		}

		if remaining := time.Until(deadline); remaining <= 0 {
			return fmt.Errorf("timed out waiting up to %s for mission status file %q to satisfy assertion: %w", timeout, path, lastErr)
		} else {
			sleep := 100 * time.Millisecond
			if remaining < sleep {
				sleep = remaining
			}
			time.Sleep(sleep)
		}
	}
}

func projectGatewayStatusAssertionSnapshot(snapshot missioncontrol.GatewayStatusSnapshot) missionStatusSnapshot {
	return missionStatusSnapshot{
		MissionRequired:   snapshot.MissionRequired,
		Active:            snapshot.Active,
		MissionFile:       snapshot.MissionFile,
		JobID:             snapshot.JobID,
		StepID:            snapshot.StepID,
		StepType:          snapshot.StepType,
		RequiredAuthority: snapshot.RequiredAuthority,
		RequiresApproval:  snapshot.RequiresApproval,
		AllowedTools:      append([]string(nil), snapshot.AllowedTools...),
		UpdatedAt:         snapshot.UpdatedAt,
	}
}

func newMissionStatusAssertionForStep(job missioncontrol.Job, stepID string) (missionStatusAssertionExpectation, error) {
	ec, err := missioncontrol.ResolveExecutionContext(job, stepID)
	if err != nil {
		return missionStatusAssertionExpectation{}, err
	}

	expected := missionStatusAssertionExpectation{
		ExactAllowedTools:      intersectAllowedTools(ec),
		CheckExactAllowedTools: true,
	}
	if ec.Job != nil {
		expected.JobID = valueOrNilString(ec.Job.ID)
	}
	if ec.Step != nil {
		expected.StepID = valueOrNilString(ec.Step.ID)
		stepType := string(ec.Step.Type)
		expected.StepType = &stepType
	}
	active := true
	expected.Active = &active

	return expected, nil
}

func checkMissionStatusAssertion(path string, snapshot missionStatusSnapshot, expected missionStatusAssertionExpectation) error {
	if expected.JobID != nil && snapshot.JobID != *expected.JobID {
		return fmt.Errorf("mission status file %q has job_id=%q step_id=%q active=%t, want job_id=%q", path, snapshot.JobID, snapshot.StepID, snapshot.Active, *expected.JobID)
	}
	if expected.StepID != nil && snapshot.StepID != *expected.StepID {
		return fmt.Errorf("mission status file %q has job_id=%q step_id=%q active=%t, want step_id=%q", path, snapshot.JobID, snapshot.StepID, snapshot.Active, *expected.StepID)
	}
	if expected.Active != nil && snapshot.Active != *expected.Active {
		return fmt.Errorf("mission status file %q has job_id=%q step_id=%q active=%t, want active=%t", path, snapshot.JobID, snapshot.StepID, snapshot.Active, *expected.Active)
	}
	if expected.StepType != nil && snapshot.StepType != *expected.StepType {
		return fmt.Errorf("mission status file %q has step_type=%q, want step_type=%q", path, snapshot.StepType, *expected.StepType)
	}
	if expected.RequiredAuthority != nil && snapshot.RequiredAuthority != *expected.RequiredAuthority {
		return fmt.Errorf("mission status file %q has required_authority=%q, want required_authority=%q", path, snapshot.RequiredAuthority, *expected.RequiredAuthority)
	}
	if expected.RequiresApproval != nil && snapshot.RequiresApproval != *expected.RequiresApproval {
		return fmt.Errorf("mission status file %q has requires_approval=%t, want requires_approval=%t", path, snapshot.RequiresApproval, *expected.RequiresApproval)
	}
	if expected.NoTools && len(snapshot.AllowedTools) != 0 {
		return fmt.Errorf("mission status file %q has allowed_tools=%q, want allowed_tools=[]", path, snapshot.AllowedTools)
	}
	for _, toolName := range expected.HasTools {
		if !containsString(snapshot.AllowedTools, toolName) {
			return fmt.Errorf("mission status file %q has allowed_tools=%q, want allowed_tools to include %q", path, snapshot.AllowedTools, toolName)
		}
	}
	if expected.CheckExactAllowedTools && !equalAllowedToolsExact(snapshot.AllowedTools, expected.ExactAllowedTools) {
		return fmt.Errorf("mission status file %q has allowed_tools=%q, want allowed_tools=%q", path, snapshot.AllowedTools, expected.ExactAllowedTools)
	}
	return nil
}

func equalAllowedToolsExact(got []string, want []string) bool {
	if len(got) == 0 && len(want) == 0 {
		return true
	}
	return reflect.DeepEqual(got, want)
}

func loadMissionStatusSnapshot(path string) (missionStatusSnapshot, error) {
	return loadMissionStatusObservation(path)
}

func valueOrNilString(value string) *string {
	return &value
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func writeMissionStatusSnapshot(path string, missionFile string, ag *agent.AgentLoop, now time.Time) error {
	if path == "" {
		return nil
	}

	var runtimeState *missioncontrol.JobRuntimeState
	if currentRuntime, ok := ag.MissionRuntimeState(); ok {
		runtimeState = missioncontrol.CloneJobRuntimeState(&currentRuntime)
	}
	var runtimeControl *missioncontrol.RuntimeControlContext
	if currentControl, ok := ag.MissionRuntimeControl(); ok {
		runtimeControl = missioncontrol.CloneRuntimeControlContext(&currentControl)
	}

	snapshot := missionStatusSnapshot{
		MissionRequired: ag.MissionRequired(),
		MissionFile:     missionFile,
		JobID:           "",
		StepID:          "",
		StepType:        "",
		AllowedTools:    []string{},
		Runtime:         runtimeState,
		RuntimeControl:  runtimeControl,
		UpdatedAt:       now.UTC().Format(time.RFC3339Nano),
	}

	if ec, ok := ag.ActiveMissionStep(); ok {
		snapshot.Active = true
		if ec.Job != nil {
			snapshot.JobID = ec.Job.ID
		}
		if ec.Step != nil {
			snapshot.StepID = ec.Step.ID
			snapshot.StepType = string(ec.Step.Type)
			snapshot.RequiredAuthority = ec.Step.RequiredAuthority
			snapshot.RequiresApproval = ec.Step.RequiresApproval
		}
		snapshot.AllowedTools = intersectAllowedTools(ec)
	} else if runtimeState != nil {
		snapshot.JobID = runtimeState.JobID
		snapshot.StepID = runtimeState.ActiveStepID
	}

	if runtimeState != nil {
		var summaryAllowedTools []string
		if snapshot.Active {
			summaryAllowedTools = append([]string(nil), snapshot.AllowedTools...)
		} else if runtimeControl != nil {
			summaryAllowedTools = missioncontrol.EffectiveAllowedTools(
				&missioncontrol.Job{AllowedTools: append([]string(nil), runtimeControl.AllowedTools...)},
				&runtimeControl.Step,
			)
		}
		summary := missioncontrol.BuildOperatorStatusSummaryWithAllowedTools(*runtimeState, summaryAllowedTools)
		snapshot.RuntimeSummary = &summary
	}

	return writeMissionStatusSnapshotAtomic(path, snapshot)
}

func writeProjectedMissionStatusSnapshot(path string, missionFile string, storeRoot string, missionRequired bool, jobID string, now time.Time) error {
	if path == "" {
		return nil
	}
	if strings.TrimSpace(storeRoot) == "" {
		return fmt.Errorf("mission status snapshot projection requires a durable store root")
	}
	if strings.TrimSpace(jobID) == "" {
		return fmt.Errorf("mission status snapshot projection requires a job_id")
	}
	if now.IsZero() {
		now = time.Now().UTC()
	} else {
		now = now.UTC()
	}

	snapshot, err := missioncontrol.BuildCommittedMissionStatusSnapshot(storeRoot, jobID, missioncontrol.MissionStatusSnapshotOptions{
		MissionRequired: missionRequired,
		MissionFile:     missionFile,
		UpdatedAt:       now,
	})
	if err != nil {
		return fmt.Errorf("failed to build committed mission status snapshot from durable store %q: %w", storeRoot, err)
	}
	return writeMissionStatusSnapshotAtomic(path, snapshot)
}

func intersectAllowedTools(ec missioncontrol.ExecutionContext) []string {
	if ec.Job == nil {
		return []string{}
	}

	jobTools := make(map[string]struct{}, len(ec.Job.AllowedTools))
	for _, toolName := range ec.Job.AllowedTools {
		jobTools[toolName] = struct{}{}
	}

	allowed := make([]string, 0, len(jobTools))
	if ec.Step == nil || len(ec.Step.AllowedTools) == 0 {
		for toolName := range jobTools {
			allowed = append(allowed, toolName)
		}
		sort.Strings(allowed)
		return allowed
	}

	for _, toolName := range ec.Step.AllowedTools {
		if _, ok := jobTools[toolName]; ok {
			allowed = append(allowed, toolName)
			delete(jobTools, toolName)
		}
	}
	sort.Strings(allowed)
	return allowed
}

func removeMissionStatusSnapshot(path string) error {
	if path == "" {
		return nil
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove mission status snapshot %q: %w", path, err)
	}
	return nil
}

func main() {
	rootCmd := NewRootCmd()
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// promptLine prints a prompt and returns the trimmed input line.
func promptLine(reader *bufio.Reader, prompt string) string {
	fmt.Print(prompt)
	line, _ := reader.ReadString('\n')
	return strings.TrimSpace(line)
}

// parseAllowFrom splits a comma-separated string into a trimmed slice.
// Returns an empty slice (not nil) if the input is blank.
func parseAllowFrom(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	if out == nil {
		return []string{}
	}
	return out
}

func setupTelegramInteractive(reader *bufio.Reader, cfg config.Config, cfgPath string) {
	fmt.Println()
	fmt.Println("=== Telegram Setup ===")
	fmt.Println()
	fmt.Println("You need a bot token from @BotFather on Telegram:")
	fmt.Println("  1. Message @BotFather on Telegram")
	fmt.Println("  2. Send /newbot and follow the prompts")
	fmt.Println("  3. Copy the token it gives you")
	fmt.Println()

	token := promptLine(reader, "Bot token: ")
	if token == "" {
		fmt.Fprintln(os.Stderr, "error: token cannot be empty")
		return
	}

	fmt.Println()
	fmt.Println("To restrict who can message your bot, enter your Telegram user ID.")
	fmt.Println("Find it by messaging @userinfobot on Telegram.")
	fmt.Println("Leave blank to allow everyone.")
	fmt.Println()

	allowFromStr := promptLine(reader, "Allowed user IDs (comma-separated, blank = everyone): ")
	allowFrom := parseAllowFrom(allowFromStr)

	cfg.Channels.Telegram.Enabled = true
	cfg.Channels.Telegram.Token = token
	cfg.Channels.Telegram.AllowFrom = allowFrom

	if err := config.SaveConfig(cfg, cfgPath); err != nil {
		fmt.Fprintf(os.Stderr, "failed to save config: %v\n", err)
		return
	}

	fmt.Println()
	fmt.Println("Telegram configured! Run 'picobot gateway' to start.")
}

func setupDiscordInteractive(reader *bufio.Reader, cfg config.Config, cfgPath string) {
	fmt.Println()
	fmt.Println("=== Discord Setup ===")
	fmt.Println()
	fmt.Println("You need a bot token from the Discord Developer Portal:")
	fmt.Println("  1. Go to https://discord.com/developers/applications")
	fmt.Println("  2. Create an application → Bot → Reset Token")
	fmt.Println("  3. Enable \"Message Content Intent\" under Privileged Gateway Intents")
	fmt.Println("  4. Invite the bot to your server via OAuth2 → URL Generator")
	fmt.Println("  5. Copy the token and paste it below")
	fmt.Println()

	token := promptLine(reader, "Bot token: ")
	if token == "" {
		fmt.Fprintln(os.Stderr, "error: token cannot be empty")
		return
	}

	fmt.Println()
	fmt.Println("To restrict who can message your bot, enter Discord user IDs.")
	fmt.Println("Enable Developer Mode (Settings → Advanced) then right-click your name → Copy User ID.")
	fmt.Println("Leave blank to allow everyone.")
	fmt.Println()

	allowFromStr := promptLine(reader, "Allowed user IDs (comma-separated, blank = everyone): ")
	allowFrom := parseAllowFrom(allowFromStr)

	cfg.Channels.Discord.Enabled = true
	cfg.Channels.Discord.Token = token
	cfg.Channels.Discord.AllowFrom = allowFrom

	if err := config.SaveConfig(cfg, cfgPath); err != nil {
		fmt.Fprintf(os.Stderr, "failed to save config: %v\n", err)
		return
	}

	fmt.Println()
	fmt.Println("Discord configured! Run 'picobot gateway' to start.")
}

func setupSlackInteractive(reader *bufio.Reader, cfg config.Config, cfgPath string) {
	fmt.Println()
	fmt.Println("=== Slack Setup ===")
	fmt.Println()
	fmt.Println("You need a Slack App with Socket Mode enabled:")
	fmt.Println("  1. Create or select an app in https://api.slack.com/apps")
	fmt.Println("  2. Go to Settings → Socket Mode and enable it")
	fmt.Println("  3. Go to Settings → Socket Mode → App Level Token")
	fmt.Println("  4. Generate an App-Level Token (xapp-...) with connections:write scope and save it down first")
	fmt.Println("  5. Go to Features → OAuth & Permissions → Bot Token Scopes and add:")
	fmt.Println("     - app_mentions:read")
	fmt.Println("     - chat:write")
	fmt.Println("     - channels:history")
	fmt.Println("     - groups:history")
	fmt.Println("     - im:history")
	fmt.Println("     - mpim:history")
	fmt.Println("     - files:read")
	fmt.Println("  6. Go to Features → Event Subscriptions → enable Events")
	fmt.Println("  7. Go to Subscribe to bot events and add:")
	fmt.Println("     - app_mention")
	fmt.Println("     - message.im")
	fmt.Println("  8. Click Install to Workspace and save the Bot User OAuth Token (xoxb-...) first")
	fmt.Println()

	appToken := promptLine(reader, "App token (xapp-...): ")
	if appToken == "" {
		fmt.Fprintln(os.Stderr, "error: app token cannot be empty")
		return
	}
	botToken := promptLine(reader, "Bot token (xoxb-...): ")
	if botToken == "" {
		fmt.Fprintln(os.Stderr, "error: bot token cannot be empty")
		return
	}

	fmt.Println()
	fmt.Println("To restrict who can message your bot, enter Slack user IDs (U...).")
	fmt.Println("Leave blank to allow everyone.")
	fmt.Println()

	allowUsersStr := promptLine(reader, "Allowed user IDs (comma-separated, blank = everyone): ")
	allowUsers := parseAllowFrom(allowUsersStr)

	fmt.Println()
	fmt.Println("To restrict which channels the bot listens to, enter Slack channel IDs (C..., G..., D...).")
	fmt.Println("Leave blank to allow all channels.")
	fmt.Println()

	allowChannelsStr := promptLine(reader, "Allowed channel IDs (comma-separated, blank = all): ")
	allowChannels := parseAllowFrom(allowChannelsStr)

	cfg.Channels.Slack.Enabled = true
	cfg.Channels.Slack.AppToken = appToken
	cfg.Channels.Slack.BotToken = botToken
	cfg.Channels.Slack.AllowUsers = allowUsers
	cfg.Channels.Slack.AllowChannels = allowChannels

	if err := config.SaveConfig(cfg, cfgPath); err != nil {
		fmt.Fprintf(os.Stderr, "failed to save config: %v\n", err)
		return
	}

	fmt.Println()
	fmt.Println("Slack configured! Run 'picobot gateway' to start.")
}

func setupWhatsAppInteractive(cfg config.Config, cfgPath string) {
	fmt.Println()
	fmt.Println("=== WhatsApp Setup ===")
	fmt.Println()

	dbPath := cfg.Channels.WhatsApp.DBPath
	if dbPath == "" {
		dbPath = "~/.picobot/whatsapp.db"
	}
	home, _ := os.UserHomeDir()
	if strings.HasPrefix(dbPath, "~/") {
		dbPath = filepath.Join(home, dbPath[2:])
	}

	if err := channels.SetupWhatsApp(dbPath); err != nil {
		fmt.Fprintf(os.Stderr, "WhatsApp setup failed: %v\n", err)
		return
	}

	cfg.Channels.WhatsApp.Enabled = true
	cfg.Channels.WhatsApp.DBPath = dbPath
	if saveErr := config.SaveConfig(cfg, cfgPath); saveErr != nil {
		fmt.Fprintf(os.Stderr, "warning: could not save config: %v\n", saveErr)
	} else {
		fmt.Printf("Config updated: whatsapp enabled, dbPath set to %s\n", dbPath)
	}

	fmt.Println("\nWhatsApp setup complete! Run 'picobot gateway' to start.")
}
