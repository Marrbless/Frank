package channels

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/local/picobot/internal/chat"
	"github.com/slack-go/slack/slackevents"
)

func TestTelegramAuthorizationMatrix(t *testing.T) {
	tests := []struct {
		name       string
		allowFrom  []string
		openMode   bool
		fromID     int64
		wantAccept bool
	}{
		{name: "allowed user", allowFrom: []string{"123"}, fromID: 123, wantAccept: true},
		{name: "denied user", allowFrom: []string{"999"}, fromID: 123, wantAccept: false},
		{name: "explicit open mode", openMode: true, fromID: 123, wantAccept: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sentUpdate := false
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if !strings.HasSuffix(r.URL.Path, "/getUpdates") {
					w.WriteHeader(http.StatusNotFound)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				if sentUpdate {
					_, _ = w.Write([]byte(`{"ok":true,"result":[]}`))
					return
				}
				sentUpdate = true
				_, _ = fmt.Fprintf(w, `{"ok":true,"result":[{"update_id":1,"message":{"message_id":1,"from":{"id":%d},"chat":{"id":456,"type":"private"},"text":"hello"}}]}`, tt.fromID)
			}))
			defer server.Close()

			hub := chat.NewHub(10)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			if err := StartTelegramWithBaseOpenMode(ctx, hub, "token", server.URL+"/bottoken", tt.allowFrom, tt.openMode); err != nil {
				t.Fatalf("StartTelegramWithBaseOpenMode() error = %v", err)
			}

			if tt.wantAccept {
				msg := waitForInbound(t, hub)
				if msg.SenderID != strconv.FormatInt(tt.fromID, 10) || msg.Content != "hello" {
					t.Fatalf("inbound = %#v, want sender %d content hello", msg, tt.fromID)
				}
				return
			}
			assertNoInbound(t, hub)
		})
	}
}

func TestDiscordAuthorizationMatrix(t *testing.T) {
	tests := []struct {
		name              string
		allowFrom         []string
		openMode          bool
		guildID           string
		mentionsBot       bool
		referencesBot     bool
		content           string
		wantAccept        bool
		wantInboundText   string
		wantInboundChatID string
	}{
		{name: "dm allowed user", allowFrom: []string{"U1"}, content: "dm hello", wantAccept: true, wantInboundText: "dm hello", wantInboundChatID: "C1"},
		{name: "dm denied user", allowFrom: []string{"U2"}, content: "blocked", wantAccept: false},
		{name: "dm explicit open mode", openMode: true, content: "open hello", wantAccept: true, wantInboundText: "open hello", wantInboundChatID: "C1"},
		{name: "guild mention allowed", allowFrom: []string{"U1"}, guildID: "G1", mentionsBot: true, content: "<@BOT> guild hello", wantAccept: true, wantInboundText: "guild hello", wantInboundChatID: "C1"},
		{name: "guild no mention ignored", allowFrom: []string{"U1"}, guildID: "G1", content: "ambient", wantAccept: false},
		{name: "guild reply to bot allowed", allowFrom: []string{"U1"}, guildID: "G1", referencesBot: true, content: "reply hello", wantAccept: true, wantInboundText: "reply hello", wantInboundChatID: "C1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hub := chat.NewHub(10)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			client := newDiscordClient(ctx, noopDiscordSender{}, hub, "BOT", tt.allowFrom, tt.openMode)
			defer client.stopAllTyping()

			message := &discordgo.Message{
				Content:   tt.content,
				ChannelID: "C1",
				GuildID:   tt.guildID,
				Author:    &discordgo.User{ID: "U1", Username: "alice"},
			}
			if tt.mentionsBot {
				message.Mentions = []*discordgo.User{{ID: "BOT", Username: "picobot"}}
			}
			if tt.referencesBot {
				message.ReferencedMessage = &discordgo.Message{Author: &discordgo.User{ID: "BOT", Username: "picobot"}}
			}

			client.handleMessage(nil, &discordgo.MessageCreate{Message: message})
			if tt.wantAccept {
				msg := waitForInbound(t, hub)
				if msg.Content != tt.wantInboundText || msg.ChatID != tt.wantInboundChatID {
					t.Fatalf("inbound = %#v, want content %q chat %q", msg, tt.wantInboundText, tt.wantInboundChatID)
				}
				return
			}
			assertNoInbound(t, hub)
		})
	}
}

func TestSlackAuthorizationMatrix(t *testing.T) {
	t.Run("dm allowed user", func(t *testing.T) {
		hub, client, cancel := newSlackAuthTestClient([]string{"U1"}, nil, false, false)
		defer cancel()
		client.handleMessage(&slackevents.MessageEvent{User: "U1", Channel: "D1", ChannelType: "im", Text: "dm hello"})
		msg := waitForInbound(t, hub)
		if msg.Content != "dm hello" || msg.ChatID != "D1" {
			t.Fatalf("inbound = %#v, want dm hello in D1", msg)
		}
	})

	t.Run("dm denied user", func(t *testing.T) {
		hub, client, cancel := newSlackAuthTestClient([]string{"U2"}, nil, false, false)
		defer cancel()
		client.handleMessage(&slackevents.MessageEvent{User: "U1", Channel: "D1", ChannelType: "im", Text: "blocked"})
		assertNoInbound(t, hub)
	})

	t.Run("dm explicit open user mode", func(t *testing.T) {
		hub, client, cancel := newSlackAuthTestClient(nil, nil, true, false)
		defer cancel()
		client.handleMessage(&slackevents.MessageEvent{User: "U1", Channel: "D1", ChannelType: "im", Text: "open dm"})
		msg := waitForInbound(t, hub)
		if msg.Content != "open dm" {
			t.Fatalf("inbound = %#v, want open dm", msg)
		}
	})

	t.Run("channel message ignored without mention", func(t *testing.T) {
		hub, client, cancel := newSlackAuthTestClient([]string{"U1"}, []string{"C1"}, false, false)
		defer cancel()
		client.handleMessage(&slackevents.MessageEvent{User: "U1", Channel: "C1", ChannelType: "channel", Text: "ambient"})
		assertNoInbound(t, hub)
	})

	t.Run("mention allowed", func(t *testing.T) {
		hub, client, cancel := newSlackAuthTestClient([]string{"U1"}, []string{"C1"}, false, false)
		defer cancel()
		client.handleMention(&slackevents.AppMentionEvent{User: "U1", Channel: "C1", Text: "<@UBOT> hello"})
		msg := waitForInbound(t, hub)
		if msg.Content != "hello" || msg.ChatID != "C1" {
			t.Fatalf("inbound = %#v, want hello in C1", msg)
		}
	})

	t.Run("mention denied channel", func(t *testing.T) {
		hub, client, cancel := newSlackAuthTestClient([]string{"U1"}, []string{"C2"}, false, false)
		defer cancel()
		client.handleMention(&slackevents.AppMentionEvent{User: "U1", Channel: "C1", Text: "<@UBOT> blocked"})
		assertNoInbound(t, hub)
	})

	t.Run("mention explicit open modes with thread reply", func(t *testing.T) {
		hub, client, cancel := newSlackAuthTestClient(nil, nil, true, true)
		defer cancel()
		client.handleMention(&slackevents.AppMentionEvent{User: "U1", Channel: "C1", ThreadTimeStamp: "1699999999.000001", Text: "<@UBOT> threaded"})
		msg := waitForInbound(t, hub)
		if msg.Content != "threaded" || msg.ChatID != "C1::1699999999.000001" {
			t.Fatalf("inbound = %#v, want threaded reply chat id", msg)
		}
	})
}

func newSlackAuthTestClient(allowUsers, allowChannels []string, openUserMode, openChannelMode bool) (*chat.Hub, *slackClient, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	hub := chat.NewHub(10)
	client := newSlackClient(ctx, nil, nil, hub, "UBOT", allowUsers, allowChannels, openUserMode, openChannelMode)
	return hub, client, cancel
}

func waitForInbound(t *testing.T, hub *chat.Hub) chat.Inbound {
	t.Helper()
	select {
	case msg := <-hub.In:
		return msg
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for inbound message")
		return chat.Inbound{}
	}
}

func assertNoInbound(t *testing.T, hub *chat.Hub) {
	t.Helper()
	select {
	case msg := <-hub.In:
		t.Fatalf("unexpected inbound message: %#v", msg)
	case <-time.After(50 * time.Millisecond):
	}
}
