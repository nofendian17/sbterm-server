package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// The spec fixes the subscription semantics:
//   - "Broadcast semua batch secara default": a client that never subscribed
//     receives every batch.
//   - "subscribe dengan symbols:[] kembali ke mode broadcast".
//   - "unsubscribe dengan symbols:[] tidak melakukan apa-apa".
//
// Unsubscribing the final symbol must therefore return the client to
// broadcast mode, never deactivate the channel.

func TestWantsDefaultBroadcast(t *testing.T) {
	c := NewClient(nil, nil) // no subscriptions at all

	assert.True(t, c.wants(ChannelRunningTrade, "BBCA"), "never-subscribed client receives all batches")
	assert.True(t, c.wants(ChannelRunningTrade, "ANTM"))
}

func TestWantsAfterSubscribeFilters(t *testing.T) {
	c := NewClient(nil, nil)
	c.Subscribe(ChannelRunningTrade, []string{"BBCA"})

	assert.True(t, c.wants(ChannelRunningTrade, "BBCA"))
	assert.False(t, c.wants(ChannelRunningTrade, "ANTM"), "subscribed clients are filtered to their symbols")
}

func TestUnsubscribeLastSymbolRevertsToBroadcast(t *testing.T) {
	c := NewClient(nil, nil)
	c.Subscribe(ChannelRunningTrade, []string{"BBCA", "BBRI"})

	c.Unsubscribe(ChannelRunningTrade, []string{"BBCA"})
	assert.True(t, c.wants(ChannelRunningTrade, "BBRI"))
	assert.False(t, c.wants(ChannelRunningTrade, "ANTM"))

	c.Unsubscribe(ChannelRunningTrade, []string{"BBRI"})
	assert.True(t, c.wants(ChannelRunningTrade, "BBCA"), "last unsubscribe reverts to receive-all broadcast")
	assert.True(t, c.wants(ChannelRunningTrade, "ANTM"))
}

func TestUnsubscribeEmptySymbolsIsNoop(t *testing.T) {
	c := NewClient(nil, nil)
	c.Subscribe(ChannelRunningTrade, []string{"BBCA"})

	c.Unsubscribe(ChannelRunningTrade, nil)

	assert.True(t, c.wants(ChannelRunningTrade, "BBCA"))
	assert.False(t, c.wants(ChannelRunningTrade, "ANTM"), "empty-symbols unsubscribe changes nothing")

	c.Unsubscribe(Channel("other"), []string{})
	assert.False(t, c.wants(Channel("other"), "BBCA"),
		"channels other than running_trade are strict opt-in")
}

// TestNewChannelsAreStrictOptIn locks the multi-channel rule: the legacy
// running_trade feed stays broadcast-all for untouched clients (original
// spec), but orderbook and alerts must NEVER leak into clients that did not
// explicitly subscribe to them.
func TestNewChannelsAreStrictOptIn(t *testing.T) {
	hub := NewHub(discardLogger())
	rtOnly := NewClient(hub, nil)
	rtOnly.Subscribe(ChannelRunningTrade, []string{"BBCA"})
	hub.Register(rtOnly)

	untouched := NewClient(hub, nil)
	hub.Register(untouched)

	assert.True(t, rtOnly.wants(ChannelRunningTrade, "BBCA"))
	assert.True(t, untouched.wants(ChannelRunningTrade, "BBCA"),
		"legacy running_trade keeps its broadcast-all default")
	assert.False(t, rtOnly.wants(ChannelOrderBook, "BBCA"),
		"orderbook must not leak into a client that only asked for trades")
	assert.False(t, untouched.wants(ChannelAlerts, "BBCA"),
		"alerts are strictly opt-in")

	// A client that subscribed to another channel must not receive the
	// running_trade flood either — legacy default belongs to untouched
	// clients only, otherwise high-volume book frames drown every channel.
	obOnly := NewClient(hub, nil)
	obOnly.Subscribe(ChannelOrderBook, []string{"BBCA"})
	hub.Register(obOnly)
	assert.True(t, obOnly.wants(ChannelOrderBook, "BBCA"))
	assert.False(t, obOnly.wants(ChannelRunningTrade, "GTSI"),
		"subscribing one channel opts you out of the running_trade flood")
}
