package service

import (
	"io"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/nofendian17/sbterm/libs/pkg/log"
)

func discardLogger() log.Logger {
	return log.New(log.WithWriter(io.Discard))
}

// receiveAll drains the client's send buffer without blocking.
func receiveAll(c *Client) [][]byte {
	var out [][]byte
	for {
		select {
		case p := <-c.send:
			out = append(out, p)
		default:
			return out
		}
	}
}

// newRegisteredClient builds a wire-free client and registers it.
func newRegisteredClient(h *Hub, subs map[Channel][]string) *Client {
	c := NewClient(h, nil)
	for ch, symbols := range subs {
		c.Subscribe(ch, symbols)
	}
	h.Register(c)
	return c
}

func TestBroadcastFilter(t *testing.T) {
	payload := []byte(`{"k":1}`)

	tests := []struct {
		name    string
		subs    map[Channel][]string
		channel Channel
		symbol  string
		want    int
	}{
		{
			name:    "no subscription receives everything (spec default)",
			channel: ChannelRunningTrade,
			symbol:  "BBCA",
			want:    1,
		},
		{
			name:    "matching symbol receives",
			subs:    map[Channel][]string{ChannelRunningTrade: {"BBCA", "ANTM"}},
			channel: ChannelRunningTrade,
			symbol:  "BBCA",
			want:    1,
		},
		{
			name:    "non-matching symbol filtered",
			subs:    map[Channel][]string{ChannelRunningTrade: {"BBCA"}},
			channel: ChannelRunningTrade,
			symbol:  "ANTM",
			want:    0,
		},
		{
			name:    "broadcast mode receives every symbol",
			subs:    map[Channel][]string{ChannelRunningTrade: {}},
			channel: ChannelRunningTrade,
			symbol:  "ANTM",
			want:    1,
		},
		{
			name:    "other channel with explicit symbols filters unknown channel",
			subs:    map[Channel][]string{ChannelRunningTrade: {"BBCA"}},
			channel: Channel("liveprice"),
			symbol:  "BBCA",
			want:    1, // never-subscribed channel = receive-all default
		},
		{
			name:    "subscribed other channel filters its non-matching symbols",
			subs:    map[Channel][]string{ChannelRunningTrade: {"BBCA"}, Channel("liveprice"): {"LIVE"}},
			channel: Channel("liveprice"),
			symbol:  "BBCA",
			want:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hub := NewHub(discardLogger())
			c := newRegisteredClient(hub, tt.subs)

			hub.Broadcast(tt.channel, tt.symbol, payload)

			assert.Len(t, receiveAll(c), tt.want)
		})
	}
}

func TestHubUnsubscribeLastSymbolRevertsToBroadcast(t *testing.T) {
	hub := NewHub(discardLogger())
	c := newRegisteredClient(hub, map[Channel][]string{ChannelRunningTrade: {"BBCA"}})

	c.Unsubscribe(ChannelRunningTrade, []string{"BBCA"})

	assert.True(t, c.wants(ChannelRunningTrade, "BBCA"), "last unsubscribe reverts to receive-all")
	assert.True(t, c.wants(ChannelRunningTrade, "ANY"))
}

func TestUnsubscribeBroadcastModeIsNoop(t *testing.T) {
	hub := NewHub(discardLogger())
	c := newRegisteredClient(hub, map[Channel][]string{ChannelRunningTrade: {}})

	c.Unsubscribe(ChannelRunningTrade, []string{"BBCA"})

	assert.True(t, c.wants(ChannelRunningTrade, "ANY"))
}

func TestSlowClientDisconnectedWhenBufferFull(t *testing.T) {
	hub := NewHub(discardLogger())
	c := newRegisteredClient(hub, map[Channel][]string{ChannelRunningTrade: {"BBCA"}})

	payload := []byte(`{"k":1}`)
	for i := 0; i < sendBufferSize; i++ {
		hub.Broadcast(ChannelRunningTrade, "BBCA", payload)
	}
	assert.Len(t, hub.clients, 1, "a full buffer alone must not disconnect the client")

	hub.Broadcast(ChannelRunningTrade, "BBCA", payload)

	assert.Empty(t, hub.clients, "slow client must be unregistered")
	select {
	case <-c.done:
	default:
		t.Fatal("expected writer stop signal to be closed")
	}
}

func TestUnregisterIdempotent(t *testing.T) {
	hub := NewHub(discardLogger())
	c := newRegisteredClient(hub, nil)

	hub.Unregister(c)
	hub.Unregister(c)

	assert.Empty(t, hub.clients)
	select {
	case <-c.done:
	default:
		t.Fatal("expected done closed exactly once")
	}
}

func TestCloseReleasesEveryClient(t *testing.T) {
	hub := NewHub(discardLogger())
	a := newRegisteredClient(hub, nil)
	b := newRegisteredClient(hub, nil)

	hub.Close()

	assert.Empty(t, hub.clients)
	for _, c := range []*Client{a, b} {
		select {
		case <-c.done:
		default:
			t.Fatal("expected all writers released")
		}
	}
}

func TestConcurrentRegisterBroadcastUnregister(t *testing.T) {
	hub := NewHub(discardLogger())
	payload := []byte(`{"k":1}`)

	var wg sync.WaitGroup
	wg.Add(3)
	go func() {
		defer wg.Done()
		for i := 0; i < 200; i++ {
			c := NewClient(hub, nil)
			c.Subscribe(ChannelRunningTrade, []string{"BBCA"})
			hub.Register(c)
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < 200; i++ {
			hub.Broadcast(ChannelRunningTrade, "BBCA", payload)
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < 200; i++ {
			time.Sleep(time.Millisecond)
			hub.mu.RLock()
			targets := make([]*Client, 0, len(hub.clients))
			for c := range hub.clients {
				targets = append(targets, c)
			}
			hub.mu.RUnlock()
			for _, c := range targets {
				if i%2 == 0 {
					hub.Unregister(c)
				}
			}
		}
	}()
	wg.Wait()

	hub.Close()
}
