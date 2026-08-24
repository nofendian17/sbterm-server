package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSubscriptionSemantics(t *testing.T) {
	tests := []struct {
		name   string
		setup  func(c *Client)
		subs   map[Channel][]string
		checks []struct {
			ch     Channel
			symbol string
			want   bool
		}
	}{
		{
			name: "subscribe adds symbols to a channel",
			setup: func(c *Client) {
				c.Subscribe(ChannelRunningTrade, []string{"BBCA"})
			},
			checks: []struct {
				ch     Channel
				symbol string
				want   bool
			}{
				{ChannelRunningTrade, "BBCA", true},
				{ChannelRunningTrade, "ANTM", false},
			},
		},
		{
			name: "re-subscribing merges symbol sets",
			setup: func(c *Client) {
				c.Subscribe(ChannelRunningTrade, []string{"BBCA"})
				c.Subscribe(ChannelRunningTrade, []string{"ANTM", "BBRI"})
			},
			checks: []struct {
				ch     Channel
				symbol string
				want   bool
			}{
				{ChannelRunningTrade, "BBCA", true},
				{ChannelRunningTrade, "ANTM", true},
				{ChannelRunningTrade, "BBRI", true},
				{ChannelRunningTrade, "TLKM", false},
			},
		},
		{
			name: "unsubscribe removes single symbol and keeps the rest",
			setup: func(c *Client) {
				c.Subscribe(ChannelRunningTrade, []string{"BBCA", "ANTM"})
				c.Unsubscribe(ChannelRunningTrade, []string{"BBCA"})
			},
			checks: []struct {
				ch     Channel
				symbol string
				want   bool
			}{
				{ChannelRunningTrade, "BBCA", false},
				{ChannelRunningTrade, "ANTM", true},
			},
		},
		{
			name: "empty subscribe switches channel to broadcast mode",
			setup: func(c *Client) {
				c.Subscribe(ChannelRunningTrade, []string{"BBCA"})
				c.Subscribe(ChannelRunningTrade, nil)
			},
			checks: []struct {
				ch     Channel
				symbol string
				want   bool
			}{
				{ChannelRunningTrade, "ANYTHING", true},
			},
		},
		{
			name: "unsubscribe on broadcast mode is a no-op",
			setup: func(c *Client) {
				c.Subscribe(ChannelRunningTrade, nil)
				c.Unsubscribe(ChannelRunningTrade, []string{"BBCA"})
			},
			checks: []struct {
				ch     Channel
				symbol string
				want   bool
			}{
				{ChannelRunningTrade, "BBCA", true},
			},
		},
		{
			name: "channels are independent",
			setup: func(c *Client) {
				c.Subscribe(ChannelRunningTrade, []string{"BBCA"})
				c.Subscribe(Channel("liveprice"), nil)
			},
			checks: []struct {
				ch     Channel
				symbol string
				want   bool
			}{
				{ChannelRunningTrade, "BBCA", true},
				{ChannelRunningTrade, "BBRI", false},
				{Channel("liveprice"), "BBCA", true},
				{Channel("liveprice"), "BBRI", true},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewClient(nil, nil)
			tt.setup(c)
			for _, check := range tt.checks {
				assert.Equal(t, check.want, c.wants(check.ch, check.symbol),
					"channel %q symbol %q", check.ch, check.symbol)
			}
		})
	}
}

func TestDeliverOverflowWithoutHubReleasesWriter(t *testing.T) {
	c := NewClient(nil, nil)
	payload := []byte(`{"k":1}`)

	for i := 0; i < sendBufferSize; i++ {
		c.Deliver(payload)
	}
	c.Deliver(payload)

	select {
	case <-c.done:
	default:
		t.Fatal("expected done closed after buffer overflow")
	}
}

func TestKnownChannelRegistry(t *testing.T) {
	assert.True(t, KnownChannel(ChannelRunningTrade))
	assert.False(t, KnownChannel(Channel("liveprice")))
}
