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
	"reflect"
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
			installMissionRuntimeChangeHook(cmd, ag)
			bootstrappedJob, err := configureMissionBootstrapJob(cmd, ag)
			if err != nil {
				return err
			}
			installMissionOperatorSetStepHook(cmd, ag, bootstrappedJob, false)
			var missionStepControlBaseline []byte
			if bootstrappedJob != nil {
				missionStepControlBaseline = restoreMissionStepControlFileOnStartup(cmd, ag, *bootstrappedJob)
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

			job, err := loadMissionJobFile(missionFile)
			if err != nil {
				return err
			}
			if err := validateMissionJob(job); err != nil {
				return fmt.Errorf("failed to validate mission file %q: %w", missionFile, err)
			}

			stepID, _ := cmd.Flags().GetString("step-id")

			summary, err := newMissionInspectSummary(job, stepID)
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
	missionInspectCmd.Flags().String("step-id", "", "Optional mission step ID to filter the inspection summary to one step")

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
				return assertMissionStatusSnapshot(statusFile, expected)
			}
			return waitForMissionStatusAssertion(statusFile, expected, waitTimeout)
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
				return assertMissionStatusSnapshot(statusFile, expected)
			}
			return waitForMissionStatusAssertion(statusFile, expected, waitTimeout)
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

				expectedJobID = job.ID
			}

			statusFile, _ := cmd.Flags().GetString("status-file")
			waitTimeout, _ := cmd.Flags().GetDuration("wait-timeout")
			if err := writeMissionStepControlAndConfirm(controlFile, statusFile, stepID, expectedJobID, waitTimeout, cmd.Flags().Changed("wait-timeout"), nil); err != nil {
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

	missionCmd.AddCommand(missionStatusCmd)
	missionCmd.AddCommand(missionInspectCmd)
	missionCmd.AddCommand(missionAssertCmd)
	missionCmd.AddCommand(missionAssertStepCmd)
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
	cmd.Flags().Bool("mission-resume-approved", false, "Approve resuming a persisted mission runtime after reboot")
}

func installMissionRuntimeChangeHook(cmd *cobra.Command, ag *agent.AgentLoop) {
	if cmd == nil || ag == nil {
		return
	}

	statusFile, _ := cmd.Flags().GetString("mission-status-file")
	if statusFile == "" {
		return
	}

	ag.SetMissionRuntimeChangeHook(func() {
		if err := writeMissionStatusSnapshot(statusFile, missionStatusSnapshotMissionFile(cmd), ag, time.Now()); err != nil {
			log.Printf("mission runtime status snapshot update failed for %q: %v", statusFile, err)
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
		if err := validateMissionStepSelection(*job, stepID); err != nil {
			return "", fmt.Errorf("failed to validate mission file %q: %w", missionFile, err)
		}

		var apply func() error
		if applySynchronously {
			apply = func() error {
				_, _, err := applyMissionStepControlFile(cmd, ag, *job, controlFile)
				return err
			}
		}

		if err := writeMissionStepControlAndConfirm(controlFile, statusFile, stepID, jobID, waitTimeout, false, apply); err != nil {
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

func writeMissionStepControlAndConfirm(controlFile string, statusFile string, stepID string, expectedJobID string, waitTimeout time.Duration, waitTimeoutExplicit bool, apply func() error) error {
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

	return waitForMissionStatusStepConfirmation(statusFile, stepID, expectedJobID, previousStatusUpdatedAt, waitTimeout)
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
	resumeApproved, _ := cmd.Flags().GetBool("mission-resume-approved")
	job, err := loadMissionJobFile(missionFile)
	if err != nil {
		return nil, err
	}

	if runtimeState, runtimeControl, ok, err := loadPersistedMissionRuntime(statusFile, job); err != nil {
		return nil, err
	} else if ok {
		if runtimeState.ActiveStepID != "" && runtimeState.ActiveStepID != missionStep {
			return nil, fmt.Errorf("persisted mission runtime step %q does not match --mission-step %q", runtimeState.ActiveStepID, missionStep)
		}
		if resumeApproved {
			if err := ag.ResumeMissionRuntime(job, runtimeState, runtimeControl, true); err != nil {
				return nil, fmt.Errorf("failed to resume persisted mission runtime from %q: %w", statusFile, err)
			}
			return &job, nil
		}
		switch runtimeState.State {
		case missioncontrol.JobStatePaused, missioncontrol.JobStateWaitingUser:
			if err := ag.HydrateMissionRuntimeControl(job, runtimeState, runtimeControl); err != nil {
				return nil, fmt.Errorf("failed to rehydrate persisted mission runtime control from %q: %w", statusFile, err)
			}
			return &job, nil
		case missioncontrol.JobStateCompleted, missioncontrol.JobStateFailed, missioncontrol.JobStateRejected, missioncontrol.JobStateAborted:
			if err := ag.HydrateMissionRuntimeControl(job, runtimeState, runtimeControl); err != nil {
				return nil, fmt.Errorf("failed to rehydrate persisted mission runtime terminal state from %q: %w", statusFile, err)
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

type missionStatusSnapshot struct {
	MissionRequired   bool                                  `json:"mission_required"`
	Active            bool                                  `json:"active"`
	MissionFile       string                                `json:"mission_file"`
	JobID             string                                `json:"job_id"`
	StepID            string                                `json:"step_id"`
	StepType          string                                `json:"step_type"`
	RequiredAuthority missioncontrol.AuthorityTier          `json:"required_authority"`
	RequiresApproval  bool                                  `json:"requires_approval"`
	AllowedTools      []string                              `json:"allowed_tools"`
	Runtime           *missioncontrol.JobRuntimeState       `json:"runtime,omitempty"`
	RuntimeControl    *missioncontrol.RuntimeControlContext `json:"runtime_control,omitempty"`
	UpdatedAt         string                                `json:"updated_at"`
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

type missionInspectSummary = missioncontrol.InspectSummary
type missionInspectStepSummary = missioncontrol.InspectStep

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

func loadPersistedMissionRuntime(path string, job missioncontrol.Job) (missioncontrol.JobRuntimeState, *missioncontrol.RuntimeControlContext, bool, error) {
	if path == "" {
		return missioncontrol.JobRuntimeState{}, nil, false, nil
	}

	snapshot, err := loadMissionStatusSnapshot(path)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return missioncontrol.JobRuntimeState{}, nil, false, nil
		}
		return missioncontrol.JobRuntimeState{}, nil, false, err
	}
	if snapshot.Runtime == nil {
		return missioncontrol.JobRuntimeState{}, nil, false, nil
	}
	if snapshot.Runtime.JobID != "" && snapshot.Runtime.JobID != job.ID {
		return missioncontrol.JobRuntimeState{}, nil, false, nil
	}
	var runtimeControl *missioncontrol.RuntimeControlContext
	if snapshot.RuntimeControl != nil {
		if snapshot.RuntimeControl.JobID != "" && snapshot.RuntimeControl.JobID != job.ID {
			return missioncontrol.JobRuntimeState{}, nil, false, nil
		}
		runtimeControl = missioncontrol.CloneRuntimeControlContext(snapshot.RuntimeControl)
	}
	return *missioncontrol.CloneJobRuntimeState(snapshot.Runtime), runtimeControl, true, nil
}

func newMissionInspectSummary(job missioncontrol.Job, stepID string) (missionInspectSummary, error) {
	return missioncontrol.NewInspectSummary(job, stepID)
}

func activateMissionStepFromControlData(ag *agent.AgentLoop, job missioncontrol.Job, path string, data []byte) (string, bool, error) {
	var control missionStepControlFile
	if err := json.Unmarshal(data, &control); err != nil {
		return "", false, fmt.Errorf("failed to decode mission step control file %q: %w", path, err)
	}
	if control.StepID == "" {
		return "", false, fmt.Errorf("mission step control file %q is missing step_id", path)
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
	return writeMissionStatusSnapshot(statusFile, missionStatusSnapshotMissionFile(cmd), ag, now)
}

func missionStatusSnapshotMissionFile(cmd *cobra.Command) string {
	missionFile, _ := cmd.Flags().GetString("mission-file")
	return missionFile
}

func waitForMissionStatusStepConfirmation(path string, stepID string, expectedJobID string, previousUpdatedAt string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	var lastErr error

	for {
		snapshot, err := loadMissionStatusSnapshot(path)
		if err != nil {
			lastErr = err
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

	return writeJSONAtomic(path, snapshot, "failed to encode mission status snapshot", "failed to write mission status snapshot")
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
