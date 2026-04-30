package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/local/picobot/internal/channels"
	"github.com/local/picobot/internal/config"
)

var promptSecretIsTerminal = term.IsTerminal
var promptSecretReadPassword = term.ReadPassword
var setupWhatsApp = channels.SetupWhatsApp

func newChannelsCmd() *cobra.Command {
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
				setupWhatsAppInteractive(reader, cfg, cfgPath)
			default:
				fmt.Fprintf(os.Stderr, "invalid choice %q — please enter 1, 2, 3 or 4\n", choice)
			}
		},
	}

	channelsCmd.AddCommand(loginCmd, newChannelsAllowlistCmd())
	return channelsCmd
}

// promptLine prints a prompt and returns the trimmed input line.
func promptLine(reader *bufio.Reader, prompt string) string {
	fmt.Print(prompt)
	line, _ := reader.ReadString('\n')
	return strings.TrimSpace(line)
}

// promptSecret prints a prompt and reads secret input without terminal echo when
// stdin is an interactive terminal. In non-terminal environments, it falls back
// to the existing line-based reader path so scripted use and tests still work.
func promptSecret(reader *bufio.Reader, prompt string) string {
	fmt.Print(prompt)

	fd := int(os.Stdin.Fd())
	if promptSecretIsTerminal(fd) {
		line, err := promptSecretReadPassword(fd)
		fmt.Println()
		if err == nil {
			return strings.TrimSpace(string(line))
		}
		fmt.Fprintln(os.Stderr, "warning: could not disable terminal echo for secret input; falling back to visible input")
	}

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

func promptAllowlistOrOpen(reader *bufio.Reader, prompt, openPrompt string) ([]string, bool, bool) {
	allowFromStr := promptLine(reader, prompt)
	allowFrom := parseAllowFrom(allowFromStr)
	if len(allowFrom) > 0 {
		return allowFrom, false, true
	}
	ack := strings.TrimSpace(strings.ToUpper(promptLine(reader, openPrompt)))
	if ack == "OPEN" {
		return []string{}, true, true
	}
	return []string{}, false, false
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
	fmt.Println("Token input is hidden on supported terminals.")
	fmt.Println()

	token := promptSecret(reader, "Bot token: ")
	if token == "" {
		fmt.Fprintln(os.Stderr, "error: token cannot be empty")
		return
	}

	fmt.Println()
	fmt.Println("To restrict who can message your bot, enter your Telegram user ID.")
	fmt.Println("Find it by messaging @userinfobot on Telegram.")
	fmt.Println("Blank is no longer accepted by default. To allow everyone, confirm explicit open mode.")
	fmt.Println()

	allowFrom, openMode, ok := promptAllowlistOrOpen(
		reader,
		"Allowed user IDs (comma-separated): ",
		"Type OPEN to allow every Telegram user, or press Enter to abort: ",
	)
	if !ok {
		fmt.Fprintln(os.Stderr, "error: Telegram setup requires allowed user IDs or explicit OPEN acknowledgement")
		return
	}

	cfg.Channels.Telegram.Enabled = true
	cfg.Channels.Telegram.Token = token
	cfg.Channels.Telegram.AllowFrom = allowFrom
	cfg.Channels.Telegram.OpenMode = openMode

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
	fmt.Println("Token input is hidden on supported terminals.")
	fmt.Println()

	token := promptSecret(reader, "Bot token: ")
	if token == "" {
		fmt.Fprintln(os.Stderr, "error: token cannot be empty")
		return
	}

	fmt.Println()
	fmt.Println("To restrict who can message your bot, enter Discord user IDs.")
	fmt.Println("Enable Developer Mode (Settings → Advanced) then right-click your name → Copy User ID.")
	fmt.Println("Blank is no longer accepted by default. To allow everyone, confirm explicit open mode.")
	fmt.Println()

	allowFrom, openMode, ok := promptAllowlistOrOpen(
		reader,
		"Allowed user IDs (comma-separated): ",
		"Type OPEN to allow every Discord user, or press Enter to abort: ",
	)
	if !ok {
		fmt.Fprintln(os.Stderr, "error: Discord setup requires allowed user IDs or explicit OPEN acknowledgement")
		return
	}

	cfg.Channels.Discord.Enabled = true
	cfg.Channels.Discord.Token = token
	cfg.Channels.Discord.AllowFrom = allowFrom
	cfg.Channels.Discord.OpenMode = openMode

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
	fmt.Println("Token input is hidden on supported terminals.")
	fmt.Println()

	appToken := promptSecret(reader, "App token (xapp-...): ")
	if appToken == "" {
		fmt.Fprintln(os.Stderr, "error: app token cannot be empty")
		return
	}
	botToken := promptSecret(reader, "Bot token (xoxb-...): ")
	if botToken == "" {
		fmt.Fprintln(os.Stderr, "error: bot token cannot be empty")
		return
	}

	fmt.Println()
	fmt.Println("To restrict who can message your bot, enter Slack user IDs (U...).")
	fmt.Println("Blank is no longer accepted by default. To allow every Slack user, confirm explicit open user mode.")
	fmt.Println()

	allowUsers, openUserMode, ok := promptAllowlistOrOpen(
		reader,
		"Allowed user IDs (comma-separated): ",
		"Type OPEN to allow every Slack user, or press Enter to abort: ",
	)
	if !ok {
		fmt.Fprintln(os.Stderr, "error: Slack setup requires allowed user IDs or explicit OPEN acknowledgement")
		return
	}

	fmt.Println()
	fmt.Println("To restrict which channels the bot listens to, enter Slack channel IDs (C..., G..., D...).")
	fmt.Println("Blank is no longer accepted by default. To allow all channels, confirm explicit open channel mode.")
	fmt.Println()

	allowChannels, openChannelMode, ok := promptAllowlistOrOpen(
		reader,
		"Allowed channel IDs (comma-separated): ",
		"Type OPEN to allow every Slack channel, or press Enter to abort: ",
	)
	if !ok {
		fmt.Fprintln(os.Stderr, "error: Slack setup requires allowed channel IDs or explicit OPEN acknowledgement")
		return
	}

	cfg.Channels.Slack.Enabled = true
	cfg.Channels.Slack.AppToken = appToken
	cfg.Channels.Slack.BotToken = botToken
	cfg.Channels.Slack.AllowUsers = allowUsers
	cfg.Channels.Slack.AllowChannels = allowChannels
	cfg.Channels.Slack.OpenUserMode = openUserMode
	cfg.Channels.Slack.OpenChannelMode = openChannelMode

	if err := config.SaveConfig(cfg, cfgPath); err != nil {
		fmt.Fprintf(os.Stderr, "failed to save config: %v\n", err)
		return
	}

	fmt.Println()
	fmt.Println("Slack configured! Run 'picobot gateway' to start.")
}

func setupWhatsAppInteractive(reader *bufio.Reader, cfg config.Config, cfgPath string) {
	fmt.Println()
	fmt.Println("=== WhatsApp Setup ===")
	fmt.Println()

	fmt.Println("To restrict who can message your bot, enter WhatsApp sender user IDs.")
	fmt.Println("Use the sender user part shown in logs, such as a phone number or LID user.")
	fmt.Println("Blank is no longer accepted by default. To allow everyone, confirm explicit open mode.")
	fmt.Println()

	allowFrom, openMode, ok := promptAllowlistOrOpen(
		reader,
		"Allowed WhatsApp sender IDs (comma-separated): ",
		"Type OPEN to allow every WhatsApp sender, or press Enter to abort: ",
	)
	if !ok {
		fmt.Fprintln(os.Stderr, "error: WhatsApp setup requires allowed sender IDs or explicit OPEN acknowledgement")
		return
	}

	dbPath := cfg.Channels.WhatsApp.DBPath
	if dbPath == "" {
		dbPath = "~/.picobot/whatsapp.db"
	}
	home, _ := os.UserHomeDir()
	if strings.HasPrefix(dbPath, "~/") {
		dbPath = filepath.Join(home, dbPath[2:])
	}

	if err := setupWhatsApp(dbPath); err != nil {
		fmt.Fprintf(os.Stderr, "WhatsApp setup failed: %v\n", err)
		return
	}

	cfg.Channels.WhatsApp.Enabled = true
	cfg.Channels.WhatsApp.DBPath = dbPath
	cfg.Channels.WhatsApp.AllowFrom = allowFrom
	cfg.Channels.WhatsApp.OpenMode = openMode
	if saveErr := config.SaveConfig(cfg, cfgPath); saveErr != nil {
		fmt.Fprintf(os.Stderr, "warning: could not save config: %v\n", saveErr)
	} else {
		fmt.Printf("Config updated: whatsapp enabled, dbPath set to %s\n", dbPath)
	}

	fmt.Println("\nWhatsApp setup complete! Run 'picobot gateway' to start.")
}
