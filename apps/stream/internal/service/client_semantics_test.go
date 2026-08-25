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
	assert.True(t, c.wants(Channel("other"), "BBCA"), "an inactive channel falls back to receive-all default")
}
