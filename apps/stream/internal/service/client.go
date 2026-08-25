package service

import (
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait = 10 * time.Second
	// PongWait bounds how long the connection may stay silent between pongs;
	// the read side renews it through its pong handler.
	PongWait       = 60 * time.Second
	pingPeriod     = 45 * time.Second
	sendBufferSize = 256
)

// PingPeriod reports the keepalive ping ticker interval (spec: ±45s).
func PingPeriod() time.Duration { return pingPeriod }

// Client is one connected WebSocket consumer: its subscription state, its
// outbound buffer, and its write pump. Subscription state changes come from
// the read goroutine (delivery), deliveries come from the poll loop through
// Hub.Broadcast — hence the mutex around subs.
type Client struct {
	hub  *Hub
	conn *websocket.Conn
	send chan []byte

	mu   sync.RWMutex
	subs map[Channel]map[string]struct{}

	done        chan struct{}
	releaseOnce sync.Once
}

// NewClient builds a client over an upgraded connection. The caller registers
// it with the hub and starts WritePump; conn may be nil in unit tests that
// never touch the wire.
func NewClient(hub *Hub, conn *websocket.Conn) *Client {
	return &Client{
		hub:  hub,
		conn: conn,
		send: make(chan []byte, sendBufferSize),
		subs: make(map[Channel]map[string]struct{}),
		done: make(chan struct{}),
	}
}

// Subscribe records interest in a channel. An empty symbols slice switches the
// client to broadcast mode for that channel (receives every symbol); a nil
// inner set marks broadcast mode.
func (c *Client) Subscribe(ch Channel, symbols []string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	set, ok := c.subs[ch]
	switch {
	case len(symbols) == 0:
		c.subs[ch] = nil
	case !ok:
		set = make(map[string]struct{}, len(symbols))
		for _, s := range symbols {
			set[s] = struct{}{}
		}
		c.subs[ch] = set
	default:
		for _, s := range symbols {
			set[s] = struct{}{}
		}
	}
}

// Unsubscribe removes symbols from one channel. Removing the final symbol
// returns the channel to broadcast (receive-all) mode, per the design spec;
// unsubscribing from an inactive channel or with no symbols is a no-op.
func (c *Client) Unsubscribe(ch Channel, symbols []string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	set, ok := c.subs[ch]
	if !ok || len(symbols) == 0 {
		return // inactive channel or nothing to remove: no-op
	}
	if set == nil {
		return // already broadcast mode: no-op
	}
	for _, s := range symbols {
		delete(set, s)
	}
	if len(set) == 0 {
		c.subs[ch] = nil // last symbol gone: back to receive-all
	}
}

// wants reports whether a record on channel/symbol matches the subscription.
// Clients that never subscribed receive everything: the spec's default is
// broadcast-all ("klien tanpa subscribe terima semua batch").
func (c *Client) wants(channel Channel, symbol string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if set, ok := c.subs[channel]; ok && set != nil {
		_, wants := set[symbol]
		return wants
	}
	return true // never subscribed, or explicit broadcast mode
}

// Deliver enqueues one pre-marshaled payload without blocking. A full buffer
// means the reader cannot keep up, so the client is unregistered as slow.
func (c *Client) Deliver(payload []byte) {
	select {
	case c.send <- payload:
	default:
		c.releaseSlow()
	}
}

// WritePump drains the send buffer onto the connection until the hub releases
// the client or a write fails. A failing writer unregisters itself so the
// read loop observes the dead connection and tears down the socket.
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		if c.hub != nil {
			c.hub.Unregister(c)
		}
	}()

	for {
		select {
		case payload := <-c.send:
			if err := c.write(payload); err != nil {
				return
			}
		case <-ticker.C:
			if err := c.conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(writeWait)); err != nil {
				return
			}
		case <-c.done:
			return
		}
	}
}

func (c *Client) write(payload []byte) error {
	if err := c.conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
		return err
	}
	return c.conn.WriteMessage(websocket.TextMessage, payload)
}

// release closes the writer's stop signal exactly once, safely even when two
// goroutines hit it concurrently (slow-client eviction + hub unregister).
func (c *Client) release() {
	c.releaseOnce.Do(func() { close(c.done) })
}

func (c *Client) releaseSlow() {
	if c.hub != nil {
		c.hub.Unregister(c)
		return
	}
	c.release()
}
