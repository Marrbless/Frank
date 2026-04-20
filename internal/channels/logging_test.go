package channels

import (
	"bytes"
	"context"
	"log"
	"strings"
	"testing"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/local/picobot/internal/chat"
	"github.com/slack-go/slack/slackevents"
)

type noopDiscordSender struct{}

func (noopDiscordSender) ChannelMessageSend(channelID, content string, options ...discordgo.RequestOption) (*discordgo.Message, error) {
	return &discordgo.Message{}, nil
}

func (noopDiscordSender) ChannelTyping(channelID string, options ...discordgo.RequestOption) error {
	return nil
}

func captureChannelLogs(t *testing.T, fn func()) string {
	t.Helper()

	var buf bytes.Buffer
	previousWriter := log.Writer()
	log.SetOutput(&buf)
	defer log.SetOutput(previousWriter)

	fn()
	return buf.String()
}

func TestSlackInboundLogsOmitRawContentAndAttachmentURLs(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	hub := chat.NewHub(10)
	client := &slackClient{
		hub:   hub,
		outCh: hub.Subscribe("slack"),
		botID: "UBOT",
		ctx:   ctx,
	}

	logs := captureChannelLogs(t, func() {
		client.handleMessage(&slackevents.MessageEvent{
			User:        "U123",
			Channel:     "D123",
			ChannelType: "im",
			Text:        "private token sk-secret",
			Files: []slackevents.File{
				{URLPrivate: "https://files.example.com/private"},
			},
		})
	})

	if !strings.Contains(logs, "slack: message from U123 in D123 (chars=") {
		t.Fatalf("expected summarized Slack log, got %q", logs)
	}
	if !strings.Contains(logs, "attachments=1") {
		t.Fatalf("expected attachment count in Slack log, got %q", logs)
	}
	if strings.Contains(logs, "private token") || strings.Contains(logs, "sk-secret") || strings.Contains(logs, "files.example.com/private") {
		t.Fatalf("expected Slack log to omit raw content and attachment URLs, got %q", logs)
	}
}

func TestSlackMentionLogsOmitRawContent(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	hub := chat.NewHub(10)
	client := &slackClient{
		hub:   hub,
		outCh: hub.Subscribe("slack"),
		botID: "UBOT",
		ctx:   ctx,
	}

	logs := captureChannelLogs(t, func() {
		client.handleMention(&slackevents.AppMentionEvent{
			User:    "U123",
			Channel: "C123",
			Text:    "<@UBOT> private campaign note",
		})
	})

	if !strings.Contains(logs, "slack: mention from U123 in C123 (chars=") {
		t.Fatalf("expected summarized Slack mention log, got %q", logs)
	}
	if strings.Contains(logs, "private campaign note") {
		t.Fatalf("expected Slack mention log to omit raw content, got %q", logs)
	}
}

func TestDiscordInboundLogsOmitRawContentAndAttachmentURLs(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	hub := chat.NewHub(10)
	client := newDiscordClient(ctx, noopDiscordSender{}, hub, "BOT", nil)

	logs := captureChannelLogs(t, func() {
		client.handleMessage(nil, &discordgo.MessageCreate{
			Message: &discordgo.Message{
				Content:   "private token sk-secret",
				ChannelID: "C123",
				Author: &discordgo.User{
					ID:            "U123",
					Username:      "alice",
					Discriminator: "1234",
				},
				Attachments: []*discordgo.MessageAttachment{
					{URL: "https://cdn.example.com/private"},
				},
			},
		})
		time.Sleep(10 * time.Millisecond)
		client.stopAllTyping()
	})

	if !strings.Contains(logs, "discord: message from alice#1234 (U123) in C123 (chars=") {
		t.Fatalf("expected summarized Discord log, got %q", logs)
	}
	if !strings.Contains(logs, "attachments=1") {
		t.Fatalf("expected attachment count in Discord log, got %q", logs)
	}
	if strings.Contains(logs, "private token") || strings.Contains(logs, "sk-secret") || strings.Contains(logs, "cdn.example.com/private") {
		t.Fatalf("expected Discord log to omit raw content and attachment URLs, got %q", logs)
	}
}
