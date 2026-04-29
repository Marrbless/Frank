package main

import (
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
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/local/picobot/internal/agent"
	"github.com/local/picobot/internal/channels"
	"github.com/local/picobot/internal/chat"
	"github.com/local/picobot/internal/config"
	"github.com/local/picobot/internal/cron"
	"github.com/local/picobot/internal/heartbeat"
	"github.com/local/picobot/internal/missioncontrol"
	"github.com/local/picobot/internal/providers"
)

const version = "1.0.1"

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

	rootCmd.AddCommand(newChannelsCmd())

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
			ag := agent.NewAgentLoop(hub, provider, model, maxIter, cfg.Agents.Defaults.Workspace, nil, cfg.MCPServers)
			defer ag.Close()
			if cfg.Agents.Defaults.EnableToolActivityIndicator != nil && !*cfg.Agents.Defaults.EnableToolActivityIndicator {
				ag.SetToolActivityIndicator(false)
			}
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
			ag = agent.NewAgentLoop(hub, provider, model, maxIter, cfg.Agents.Defaults.Workspace, scheduler, cfg.MCPServers)
			defer ag.Close()
			if cfg.Agents.Defaults.EnableToolActivityIndicator != nil && !*cfg.Agents.Defaults.EnableToolActivityIndicator {
				ag.SetToolActivityIndicator(false)
			}
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

	rootCmd.AddCommand(newMemoryCmd())
	rootCmd.AddCommand(newSkillsCmd())

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

	if err := missioncontrol.WriteStoreFileAtomic(path, data); err != nil {
		return fmt.Errorf("%s %q: %w", writeErrPrefix, path, err)
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

type missionStepControlFile struct {
	StepID    string `json:"step_id"`
	UpdatedAt string `json:"updated_at"`
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

var loadValidatedLegacyMissionStatusSnapshot = missioncontrol.LoadValidatedLegacyMissionStatusSnapshot
var loadMissionStatusObservationFile = missioncontrol.LoadMissionStatusObservationFile

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

func main() {
	rootCmd := NewRootCmd()
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
