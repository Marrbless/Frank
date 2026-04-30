package channels

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/local/picobot/internal/chat"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"github.com/slack-go/slack/socketmode"
)

// slackPoster is the subset of *slack.Client used for posting outbound messages.
// It exists to enable testing without a live Slack connection, mirroring the
// discordSender pattern used by the Discord channel.
type slackPoster interface {
	PostMessageContext(ctx context.Context, channelID string, options ...slack.MsgOption) (string, string, error)
}

// StartSlack starts a Slack bot using Socket Mode.
// allowUsers restricts which Slack user IDs may send messages; empty fails closed.
// allowChannels restricts which Slack channel IDs may send messages; empty fails closed for non-DMs.
func StartSlack(ctx context.Context, hub *chat.Hub, appToken, botToken string, allowUsers, allowChannels []string) error {
	return StartSlackWithOpenMode(ctx, hub, appToken, botToken, allowUsers, allowChannels, false, false)
}

func StartSlackWithOpenMode(ctx context.Context, hub *chat.Hub, appToken, botToken string, allowUsers, allowChannels []string, openUserMode, openChannelMode bool) error {
	if appToken == "" {
		return fmt.Errorf("slack app token not provided")
	}
	if botToken == "" {
		return fmt.Errorf("slack bot token not provided")
	}
	if !strings.HasPrefix(appToken, "xapp-") {
		return fmt.Errorf("slack app token must start with xapp-")
	}
	if !strings.HasPrefix(botToken, "xoxb-") {
		return fmt.Errorf("slack bot token must start with xoxb-")
	}
	if err := requireAllowlistOrExplicitOpen("slack users", allowUsers, openUserMode); err != nil {
		return err
	}
	if err := requireAllowlistOrExplicitOpen("slack channels", allowChannels, openChannelMode); err != nil {
		return err
	}

	api := slack.New(
		botToken,
		slack.OptionAppLevelToken(appToken),
	)

	auth, err := api.AuthTest()
	if err != nil {
		return fmt.Errorf("slack auth test failed: %w", err)
	}
	if auth.UserID == "" {
		return fmt.Errorf("slack auth test returned empty user ID")
	}

	socketClient := socketmode.New(api)
	client := newSlackClient(ctx, socketClient, api, hub, auth.UserID, allowUsers, allowChannels, openUserMode, openChannelMode)

	go client.runOutbound()
	go client.runEvents()

	go func() {
		if err := socketClient.RunContext(ctx); err != nil {
			log.Printf("slack: socket mode error: %s", redactLogError(err))
		}
	}()

	go func() {
		<-ctx.Done()
		log.Println("slack: shutting down")
	}()

	return nil
}

type slackClient struct {
	socket       *socketmode.Client
	poster       slackPoster
	hub          *chat.Hub
	outCh        <-chan chat.Outbound
	botID        string
	allowedUsers map[string]struct{}
	allowedChans map[string]struct{}
	openUserMode bool
	openChanMode bool
	rateLimiter  ChannelRateLimiter
	ctx          context.Context
}

func newSlackClient(ctx context.Context, socket *socketmode.Client, poster slackPoster, hub *chat.Hub, botID string, allowUsers, allowChannels []string, openUserMode, openChannelMode bool) *slackClient {
	allowedUsers := buildAllowedSet(allowUsers)
	allowedChans := buildAllowedSet(allowChannels)

	return &slackClient{
		socket:       socket,
		poster:       poster,
		hub:          hub,
		outCh:        hub.Subscribe("slack"),
		botID:        botID,
		allowedUsers: allowedUsers,
		allowedChans: allowedChans,
		openUserMode: openUserMode,
		openChanMode: openChannelMode,
		rateLimiter:  defaultChannelRateLimiter,
		ctx:          ctx,
	}
}

func (c *slackClient) runEvents() {
	for {
		select {
		case <-c.ctx.Done():
			return
		case evt, ok := <-c.socket.Events:
			if !ok {
				return
			}
			switch evt.Type {
			case socketmode.EventTypeEventsAPI:
				eventsAPIEvent, ok := evt.Data.(slackevents.EventsAPIEvent)
				if !ok {
					log.Printf("slack: unexpected event data: %T", evt.Data)
					continue
				}
				c.socket.Ack(*evt.Request)
				if eventsAPIEvent.Type != slackevents.CallbackEvent {
					continue
				}
				c.handleCallbackEvent(eventsAPIEvent.InnerEvent)
			case socketmode.EventTypeInvalidAuth:
				log.Println("slack: invalid auth")
				return
			}
		}
	}
}

func (c *slackClient) handleCallbackEvent(inner slackevents.EventsAPIInnerEvent) {
	switch ev := inner.Data.(type) {
	case *slackevents.AppMentionEvent:
		c.handleMention(ev)
	case *slackevents.MessageEvent:
		c.handleMessage(ev)
	}
}

func (c *slackClient) handleMention(ev *slackevents.AppMentionEvent) {
	if ev.User == "" || ev.User == c.botID || ev.BotID != "" {
		return
	}

	if !c.isAllowed(ev.User, ev.Channel, false) {
		c.logUnauthorized(ev.User, ev.Channel, false)
		return
	}

	content := stripSlackMention(ev.Text, c.botID)
	content = strings.TrimSpace(content)
	if content == "" {
		return
	}
	if !c.allowInboundByRateLimit(ev.User, ev.Channel) {
		return
	}

	threadTS := ev.ThreadTimeStamp
	chatID := formatSlackChatID(ev.Channel, threadTS)
	teamID := firstNonEmpty(ev.SourceTeam, ev.UserTeam)

	log.Printf("slack: mention from %s in %s (%s)", redactLogID(ev.User), redactLogID(ev.Channel), summarizeInboundContent(content, 0))

	c.hub.In <- chat.Inbound{
		Channel:   "slack",
		SenderID:  ev.User,
		ChatID:    chatID,
		Content:   content,
		Timestamp: time.Now(),
		Metadata: map[string]interface{}{
			"channel_id": ev.Channel,
			"team_id":    teamID,
			"thread_ts":  threadTS,
			"is_dm":      false,
		},
	}
}

func (c *slackClient) handleMessage(ev *slackevents.MessageEvent) {
	if ev.User == "" || ev.User == c.botID || ev.BotID != "" {
		return
	}
	if ev.SubType != "" {
		return
	}

	isDM := ev.ChannelType == "im"
	if !isDM {
		return
	}

	if !c.isAllowed(ev.User, ev.Channel, isDM) {
		c.logUnauthorized(ev.User, ev.Channel, isDM)
		return
	}

	content := strings.TrimSpace(ev.Text)
	content = appendSlackAttachments(content, ev.Files)
	content = strings.TrimSpace(content)
	if content == "" {
		return
	}
	if !c.allowInboundByRateLimit(ev.User, ev.Channel) {
		return
	}

	threadTS := ev.ThreadTimeStamp
	chatID := formatSlackChatID(ev.Channel, threadTS)
	teamID := firstNonEmpty(ev.SourceTeam, ev.UserTeam)

	log.Printf("slack: message from %s in %s (%s)", redactLogID(ev.User), redactLogID(ev.Channel), summarizeInboundContent(content, len(ev.Files)))

	c.hub.In <- chat.Inbound{
		Channel:   "slack",
		SenderID:  ev.User,
		ChatID:    chatID,
		Content:   content,
		Timestamp: time.Now(),
		Metadata: map[string]interface{}{
			"channel_id": ev.Channel,
			"team_id":    teamID,
			"thread_ts":  threadTS,
			"is_dm":      isDM,
		},
	}
}

func (c *slackClient) runOutbound() {
	for {
		select {
		case <-c.ctx.Done():
			log.Println("slack: stopping outbound sender")
			return
		case out := <-c.outCh:
			channelID, threadTS := splitSlackChatID(out.ChatID)
			if channelID == "" {
				log.Printf("slack: invalid chat ID %s", redactLogID(out.ChatID))
				continue
			}
			for _, chunk := range splitMessage(out.Content, 4000) {
				opts := []slack.MsgOption{slack.MsgOptionText(chunk, false)}
				if threadTS != "" {
					opts = append(opts, slack.MsgOptionTS(threadTS))
				}
				if _, _, err := c.poster.PostMessageContext(c.ctx, channelID, opts...); err != nil {
					log.Printf("slack: send error: %s", redactLogError(err))
				}
			}
		}
	}
}

func (c *slackClient) isAllowed(userID, channelID string, isDM bool) bool {
	if !allowedBySingleAllowlist(c.allowedUsers, c.openUserMode, userID) {
		return false
	}
	if isDM {
		return true
	}
	return allowedBySingleAllowlist(c.allowedChans, c.openChanMode, channelID)
}

func (c *slackClient) allowInboundByRateLimit(userID, channelID string) bool {
	limiter := c.rateLimiter
	if limiter == nil {
		limiter = defaultChannelRateLimiter
	}
	if limiter.AllowInbound("slack", userID, channelID, time.Now()) {
		return true
	}
	log.Printf("slack: rate limited inbound message user=%s channel=%s", redactLogID(userID), redactLogID(channelID))
	return false
}

func (c *slackClient) logUnauthorized(userID, channelID string, isDM bool) {
	userAllowed := c.openUserMode
	channelAllowed := true
	if len(c.allowedUsers) > 0 {
		_, userAllowed = c.allowedUsers[userID]
	}
	if !isDM {
		channelAllowed = c.openChanMode
		if len(c.allowedChans) > 0 {
			_, channelAllowed = c.allowedChans[channelID]
		}
	}
	log.Printf("slack: dropped message: user allowed=%t channel allowed=%t user=%s channel=%s", userAllowed, channelAllowed, redactLogID(userID), redactLogID(channelID))
}

func stripSlackMention(text, botID string) string {
	if botID == "" {
		return text
	}
	return strings.ReplaceAll(text, "<@"+botID+">", "")
}

func appendSlackAttachments(content string, files []slackevents.File) string {
	for _, file := range files {
		url := file.URLPrivate
		if url == "" {
			url = file.URLPrivateDownload
		}
		if url == "" {
			url = file.Permalink
		}
		if url == "" {
			continue
		}
		if content != "" {
			content += "\n"
		}
		content += fmt.Sprintf("[attachment: %s]", url)
	}
	return content
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func formatSlackChatID(channelID, threadTS string) string {
	if threadTS == "" {
		return channelID
	}
	return channelID + "::" + threadTS
}

func splitSlackChatID(chatID string) (string, string) {
	parts := strings.SplitN(chatID, "::", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return chatID, ""
}
