package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/local/picobot/internal/agent"
	"github.com/local/picobot/internal/agent/memory"
	"github.com/local/picobot/internal/channels"
	"github.com/local/picobot/internal/chat"
	"github.com/local/picobot/internal/config"
	"github.com/local/picobot/internal/cron"
	"github.com/local/picobot/internal/heartbeat"
	"github.com/local/picobot/internal/missioncontrol"
	"github.com/local/picobot/internal/providers"
)

const version = "0.1.10"

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
			if err := configureMissionBootstrap(cmd, ag); err != nil {
				return err
			}
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

			// choose model: flag > config > provider default
			modelFlag, _ := cmd.Flags().GetString("model")
			model := modelFlag
			if model == "" && cfg.Agents.Defaults.Model != "" {
				model = cfg.Agents.Defaults.Model
			}
			if model == "" {
				model = provider.GetDefaultModel()
			}

			// create scheduler with fire callback that routes back through the agent loop, so the LLM can process the reminder and respond naturally to the user.
			scheduler := cron.NewScheduler(func(job cron.Job) {
				log.Printf("cron fired: %s — %s", job.Name, job.Message)
				hub.In <- chat.Inbound{
					Channel:  job.Channel,
					SenderID: "cron",
					ChatID:   job.ChatID,
					Content:  fmt.Sprintf("[Scheduled reminder fired] %s — Please relay this to the user in a friendly way.", job.Message),
				}
			})

			maxIter := cfg.Agents.Defaults.MaxToolIterations
			if maxIter <= 0 {
				maxIter = 100
			}
			ag := agent.NewAgentLoop(hub, provider, model, maxIter, cfg.Agents.Defaults.Workspace, scheduler)
			bootstrappedJob, err := configureMissionBootstrapJob(cmd, ag)
			if err != nil {
				return err
			}
			if bootstrappedJob != nil {
				restoreMissionStepControlFileOnStartup(cmd, ag, *bootstrappedJob)
			}
			statusFile, _ := cmd.Flags().GetString("mission-status-file")
			if err := writeMissionStatusSnapshot(statusFile, missionStatusSnapshotMissionFile(cmd), ag, time.Now()); err != nil {
				return err
			}
			if statusFile != "" {
				defer func() {
					if err == nil {
						err = removeMissionStatusSnapshot(statusFile)
					}
				}()
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// start agent loop
			go ag.Run(ctx)

			if bootstrappedJob != nil {
				controlFile, _ := cmd.Flags().GetString("mission-step-control-file")
				if controlFile != "" {
					go watchMissionStepControlFile(ctx, cmd, ag, *bootstrappedJob, controlFile, 500*time.Millisecond)
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

			data, err := os.ReadFile(statusFile)
			if err != nil {
				return fmt.Errorf("failed to read mission status file %q: %w", statusFile, err)
			}

			var snapshot any
			if err := json.Unmarshal(data, &snapshot); err != nil {
				return fmt.Errorf("failed to decode mission status file %q: %w", statusFile, err)
			}

			if _, err := cmd.OutOrStdout().Write(data); err != nil {
				return fmt.Errorf("failed to write mission status output: %w", err)
			}

			return nil
		},
	}
	missionStatusCmd.Flags().String("status-file", "", "Path to a mission status snapshot JSON file")

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

			data, err := os.ReadFile(missionFile)
			if err != nil {
				return fmt.Errorf("failed to read mission file %q: %w", missionFile, err)
			}

			var job missioncontrol.Job
			if err := json.Unmarshal(data, &job); err != nil {
				return fmt.Errorf("failed to decode mission file %q: %w", missionFile, err)
			}

			if err := validateMissionJob(job); err != nil {
				return fmt.Errorf("failed to validate mission file %q: %w", missionFile, err)
			}

			summaryData, err := json.MarshalIndent(newMissionInspectSummary(job), "", "  ")
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
			if missionFile != "" {
				data, err := os.ReadFile(missionFile)
				if err != nil {
					return fmt.Errorf("failed to read mission file %q: %w", missionFile, err)
				}

				var job missioncontrol.Job
				if err := json.Unmarshal(data, &job); err != nil {
					return fmt.Errorf("failed to decode mission file %q: %w", missionFile, err)
				}

				if err := validateMissionStepSelection(job, stepID); err != nil {
					return fmt.Errorf("failed to validate mission file %q: %w", missionFile, err)
				}
			}

			statusFile, _ := cmd.Flags().GetString("status-file")
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

			data, err := json.MarshalIndent(control, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to encode mission step control file %q: %w", controlFile, err)
			}
			data = append(data, '\n')

			if err := os.WriteFile(controlFile, data, 0o644); err != nil {
				return fmt.Errorf("failed to write mission step control file %q: %w", controlFile, err)
			}

			if statusFile == "" {
				return nil
			}

			waitTimeout, _ := cmd.Flags().GetDuration("wait-timeout")
			if !cmd.Flags().Changed("wait-timeout") {
				waitTimeout = 5 * time.Second
			}

			if err := waitForMissionStatusStepConfirmation(statusFile, stepID, previousStatusUpdatedAt, waitTimeout); err != nil {
				return err
			}

			return nil
		},
	}
	missionSetStepCmd.Flags().String("control-file", "", "Path to write a mission step control JSON file")
	missionSetStepCmd.Flags().String("step-id", "", "Mission step ID to activate")
	missionSetStepCmd.Flags().String("mission-file", "", "Path to a mission job JSON file to validate before writing the mission step control file")
	missionSetStepCmd.Flags().String("status-file", "", "Path to a mission status snapshot JSON file to confirm after writing the mission step control file")
	missionSetStepCmd.Flags().Duration("wait-timeout", 0, "How long to wait for mission status confirmation after writing the mission step control file")

	missionCmd.AddCommand(missionStatusCmd)
	missionCmd.AddCommand(missionInspectCmd)
	missionCmd.AddCommand(missionSetStepCmd)
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
	return rootCmd
}

func addMissionBootstrapFlags(cmd *cobra.Command) {
	cmd.Flags().Bool("mission-required", false, "Require an active mission step before tool execution")
	cmd.Flags().String("mission-file", "", "Path to a mission job JSON file to activate at startup")
	cmd.Flags().String("mission-step", "", "Mission step ID to activate from the mission file")
	cmd.Flags().String("mission-status-file", "", "Path to write a mission status snapshot after startup")
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

	data, err := os.ReadFile(missionFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read mission file %q: %w", missionFile, err)
	}

	var job missioncontrol.Job
	if err := json.Unmarshal(data, &job); err != nil {
		return nil, fmt.Errorf("failed to decode mission file %q: %w", missionFile, err)
	}

	if err := ag.ActivateMissionStep(job, missionStep); err != nil {
		return nil, fmt.Errorf("failed to activate mission step %q from %q: %w", missionStep, missionFile, err)
	}

	return &job, nil
}

type missionStatusSnapshot struct {
	MissionRequired bool     `json:"mission_required"`
	Active          bool     `json:"active"`
	MissionFile     string   `json:"mission_file"`
	JobID           string   `json:"job_id"`
	StepID          string   `json:"step_id"`
	StepType        string   `json:"step_type"`
	AllowedTools    []string `json:"allowed_tools"`
	UpdatedAt       string   `json:"updated_at"`
}

type missionStepControlFile struct {
	StepID    string `json:"step_id"`
	UpdatedAt string `json:"updated_at"`
}

type missionInspectSummary struct {
	JobID        string                       `json:"job_id"`
	MaxAuthority missioncontrol.AuthorityTier `json:"max_authority"`
	AllowedTools []string                     `json:"allowed_tools"`
	Steps        []missionInspectStepSummary  `json:"steps"`
}

type missionInspectStepSummary struct {
	StepID            string                       `json:"step_id"`
	StepType          missioncontrol.StepType      `json:"step_type"`
	DependsOn         []string                     `json:"depends_on"`
	RequiredAuthority missioncontrol.AuthorityTier `json:"required_authority"`
	AllowedTools      []string                     `json:"allowed_tools"`
	RequiresApproval  bool                         `json:"requires_approval"`
}

func validateMissionJob(job missioncontrol.Job) error {
	if validationErrors := missioncontrol.ValidatePlan(job); len(validationErrors) > 0 {
		return validationErrors[0]
	}
	return nil
}

func validateMissionStepSelection(job missioncontrol.Job, stepID string) error {
	if err := validateMissionJob(job); err != nil {
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

func newMissionInspectSummary(job missioncontrol.Job) missionInspectSummary {
	steps := make([]missionInspectStepSummary, 0, len(job.Plan.Steps))
	for _, step := range job.Plan.Steps {
		steps = append(steps, missionInspectStepSummary{
			StepID:            step.ID,
			StepType:          step.Type,
			DependsOn:         append([]string(nil), step.DependsOn...),
			RequiredAuthority: step.RequiredAuthority,
			AllowedTools:      append([]string(nil), step.AllowedTools...),
			RequiresApproval:  step.RequiresApproval,
		})
	}

	return missionInspectSummary{
		JobID:        job.ID,
		MaxAuthority: job.MaxAuthority,
		AllowedTools: append([]string(nil), job.AllowedTools...),
		Steps:        steps,
	}
}

func activateMissionStepFromControlFile(ag *agent.AgentLoop, job missioncontrol.Job, path string) (bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to read mission step control file %q: %w", path, err)
	}

	var control missionStepControlFile
	if err := json.Unmarshal(data, &control); err != nil {
		return false, fmt.Errorf("failed to decode mission step control file %q: %w", path, err)
	}
	if control.StepID == "" {
		return false, fmt.Errorf("mission step control file %q is missing step_id", path)
	}

	if err := ag.ActivateMissionStep(job, control.StepID); err != nil {
		return false, fmt.Errorf("failed to activate mission step %q from control file %q: %w", control.StepID, path, err)
	}

	return true, nil
}

func restoreMissionStepControlFileOnStartup(cmd *cobra.Command, ag *agent.AgentLoop, job missioncontrol.Job) {
	controlFile, _ := cmd.Flags().GetString("mission-step-control-file")
	if controlFile == "" {
		return
	}

	if _, err := activateMissionStepFromControlFile(ag, job, controlFile); err != nil {
		log.Printf("mission step control startup apply failed for %q: %v", controlFile, err)
	}
}

func applyMissionStepControlFile(cmd *cobra.Command, ag *agent.AgentLoop, job missioncontrol.Job, path string) (bool, error) {
	changed, err := activateMissionStepFromControlFile(ag, job, path)
	if err != nil || !changed {
		return changed, err
	}

	if err := writeMissionStatusSnapshotFromCommand(cmd, ag, time.Now()); err != nil {
		return false, err
	}

	return true, nil
}

func watchMissionStepControlFile(ctx context.Context, cmd *cobra.Command, ag *agent.AgentLoop, job missioncontrol.Job, path string, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	var lastContent []byte
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
			if _, err := applyMissionStepControlFile(cmd, ag, job, path); err != nil {
				log.Printf("mission step control apply failed for %q: %v", path, err)
			}
		}
	}
}

func writeMissionStatusSnapshotFromCommand(cmd *cobra.Command, ag *agent.AgentLoop, now time.Time) error {
	statusFile, _ := cmd.Flags().GetString("mission-status-file")
	return writeMissionStatusSnapshot(statusFile, missionStatusSnapshotMissionFile(cmd), ag, now)
}

func missionStatusSnapshotMissionFile(cmd *cobra.Command) string {
	missionFile, _ := cmd.Flags().GetString("mission-file")
	return missionFile
}

func waitForMissionStatusStepConfirmation(path string, stepID string, previousUpdatedAt string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	var lastErr error

	for {
		snapshot, err := loadMissionStatusSnapshot(path)
		if err != nil {
			lastErr = err
		} else if !snapshot.Active || snapshot.StepID != stepID {
			lastErr = fmt.Errorf("mission status file %q has active=%t step_id=%q, want active=true step_id=%q", path, snapshot.Active, snapshot.StepID, stepID)
		} else if previousUpdatedAt == "" || snapshot.UpdatedAt != previousUpdatedAt {
			return nil
		} else {
			lastErr = fmt.Errorf("mission status file %q has active=true step_id=%q updated_at=%q, want a fresh matching update with updated_at different from %q", path, snapshot.StepID, snapshot.UpdatedAt, previousUpdatedAt)
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

func loadMissionStatusSnapshot(path string) (missionStatusSnapshot, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return missionStatusSnapshot{}, fmt.Errorf("mission status file %q not found", path)
		}
		return missionStatusSnapshot{}, fmt.Errorf("failed to read mission status file %q: %w", path, err)
	}

	var snapshot missionStatusSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return missionStatusSnapshot{}, fmt.Errorf("failed to decode mission status file %q: %w", path, err)
	}

	return snapshot, nil
}

func writeMissionStatusSnapshot(path string, missionFile string, ag *agent.AgentLoop, now time.Time) error {
	if path == "" {
		return nil
	}

	snapshot := missionStatusSnapshot{
		MissionRequired: ag.MissionRequired(),
		MissionFile:     missionFile,
		JobID:           "",
		StepID:          "",
		StepType:        "",
		AllowedTools:    []string{},
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
		}
		snapshot.AllowedTools = intersectAllowedTools(ec)
	}

	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to encode mission status snapshot %q: %w", path, err)
	}
	data = append(data, '\n')

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("failed to write mission status snapshot %q: %w", path, err)
	}

	return nil
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
