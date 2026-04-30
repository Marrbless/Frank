package channels

import "time"

// ChannelRateLimiter is a channel-level hook for inbound rate decisions.
// The default implementation allows all messages; production rate policy can
// be wired later without changing channel handler control flow.
type ChannelRateLimiter interface {
	AllowInbound(channel, senderID, chatID string, now time.Time) bool
}

var defaultChannelRateLimiter ChannelRateLimiter = allowAllChannelRateLimiter{}

type allowAllChannelRateLimiter struct{}

func (allowAllChannelRateLimiter) AllowInbound(channel, senderID, chatID string, now time.Time) bool {
	return true
}
