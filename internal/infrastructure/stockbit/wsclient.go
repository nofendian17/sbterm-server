package stockbit

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"google.golang.org/protobuf/proto"

	datafeedv1 "github.com/nofendian17/sbterm-server/internal/infrastructure/stockbit/proto/securities/transactional/datafeed/v1"
	"github.com/nofendian17/sbterm-server/pkg/log"
)

// KeyProvider supplies the websocket key used to authenticate a datafeed
// connection. The key can rotate, so it is fetched again before every dial.
type KeyProvider func(ctx context.Context) (string, error)

// WSSubscription describes one datafeed subscription: the account user id and
// the channels to subscribe the connection to.
type WSSubscription struct {
	UserID  int64
	Channel *datafeedv1.WebsocketChannel
}

// WSHandler receives each decoded server frame. Returning an error is logged
// and the read loop continues; it does not tear down the connection.
type WSHandler func(ctx context.Context, msg *datafeedv1.WebsocketWrapMessageChannel) error

// wsOptions holds the tunable behavior of a WSClient.
type wsOptions struct {
	dialTimeout  time.Duration
	pingInterval time.Duration
	readTimeout  time.Duration
	writeTimeout time.Duration
	backoffInit  time.Duration
	backoffMax   time.Duration
	logger       log.Logger
}

// WSOption configures a WSClient.
type WSOption func(*wsOptions)

// WithWSDialTimeout sets the websocket handshake timeout.
func WithWSDialTimeout(d time.Duration) WSOption {
	return func(o *wsOptions) { o.dialTimeout = d }
}

// WithWSPingInterval sets how often the client sends a keepalive ping frame.
func WithWSPingInterval(d time.Duration) WSOption {
	return func(o *wsOptions) { o.pingInterval = d }
}

// WithWSReadTimeout sets how long the client waits on a read before treating
// the connection as dead and reconnecting.
func WithWSReadTimeout(d time.Duration) WSOption {
	return func(o *wsOptions) { o.readTimeout = d }
}

// WithWSWriteTimeout sets the write deadline for outgoing frames.
func WithWSWriteTimeout(d time.Duration) WSOption {
	return func(o *wsOptions) { o.writeTimeout = d }
}

// WithWSReconnectBackoff sets the initial and maximum delays between reconnect
// attempts; the delay doubles on every consecutive failure.
func WithWSReconnectBackoff(initial, max time.Duration) WSOption {
	return func(o *wsOptions) { o.backoffInit, o.backoffMax = initial, max }
}

// WithWSLogger enables debug logging of connection events.
func WithWSLogger(l log.Logger) WSOption {
	return func(o *wsOptions) { o.logger = l }
}

// WSClient streams the Stockbit datafeed websocket (wss://wssfeed.stockbit.com):
// it dials with a wskey, subscribes to a set of channels, decodes incoming
// protobuf frames, and reconnects with backoff when the connection drops.
type WSClient struct {
	url string
	key KeyProvider

	opts wsOptions

	writeMu sync.Mutex
	connMu  sync.Mutex
	conn    *websocket.Conn
}

// NewWSClient builds a datafeed websocket client. url is the websocket
// endpoint (e.g. "wss://wssfeed.stockbit.com/"); key fetches the wskey used in
// the handshake and in the subscribe frame.
func NewWSClient(url string, key KeyProvider, opts ...WSOption) *WSClient {
	o := wsOptions{
		dialTimeout:  10 * time.Second,
		pingInterval: 30 * time.Second,
		readTimeout:  90 * time.Second,
		writeTimeout: 10 * time.Second,
		backoffInit:  time.Second,
		backoffMax:   30 * time.Second,
	}
	for _, opt := range opts {
		opt(&o)
	}
	return &WSClient{url: url, key: key, opts: o}
}

// Run dials the datafeed, sends the subscription, and dispatches decoded
// frames to handler until ctx is cancelled. On connection failure it fetches a
// fresh key, redials, and resubscribes, backing off exponentially. The returned
// error is nil when Run ended because ctx was cancelled.
func (c *WSClient) Run(ctx context.Context, sub WSSubscription, handler WSHandler) error {
	backoff := c.opts.backoffInit
	for {
		if ctx.Err() != nil {
			return nil
		}

		key, err := c.key(ctx)
		if err != nil {
			c.logWarn("stockbit ws: fetch key failed", "error", err)
			if !sleepCtx(ctx, backoff) {
				return nil
			}
			backoff = c.nextBackoff(backoff)
			continue
		}

		conn, err := c.dial(ctx, key)
		if err != nil {
			c.logWarn("stockbit ws: dial failed", "error", err)
			if !sleepCtx(ctx, backoff) {
				return nil
			}
			backoff = c.nextBackoff(backoff)
			continue
		}
		if err := c.subscribe(conn, key, sub); err != nil {
			conn.Close()
			c.logWarn("stockbit ws: subscribe failed", "error", err)
			if !sleepCtx(ctx, backoff) {
				return nil
			}
			backoff = c.nextBackoff(backoff)
			continue
		}
		backoff = c.opts.backoffInit

		c.setConn(conn)
		c.logInfo("stockbit ws connected")

		pingCtx, pingCancel := context.WithCancel(ctx)
		pingDone := make(chan struct{})
		go c.pingLoop(conn, pingCtx, pingDone)

		// Unblock the pending read when the caller cancels the context.
		connClosed := make(chan struct{})
		go func() {
			select {
			case <-ctx.Done():
				conn.Close()
			case <-connClosed:
			}
		}()

		err = c.readLoop(ctx, conn, handler)
		pingCancel()
		<-pingDone
		c.clearConn(conn)
		conn.Close()
		close(connClosed)

		if ctx.Err() != nil {
			return nil
		}
		c.logWarn("stockbit ws disconnected; reconnecting", "error", err)
		if !sleepCtx(ctx, backoff) {
			return nil
		}
		backoff = c.nextBackoff(backoff)
	}
}

// Close closes the current connection, if any. Run resumes with a reconnect
// unless its context was cancelled.
func (c *WSClient) Close() error {
	c.connMu.Lock()
	conn := c.conn
	c.conn = nil
	c.connMu.Unlock()
	if conn != nil {
		return conn.Close()
	}
	return nil
}

func (c *WSClient) dial(ctx context.Context, key string) (*websocket.Conn, error) {
	endpoint := c.url + "?wskey=" + url.QueryEscape(key)
	d := websocket.Dialer{HandshakeTimeout: c.opts.dialTimeout}
	conn, resp, err := d.DialContext(ctx, endpoint, nil)
	if resp != nil && resp.Body != nil {
		resp.Body.Close()
	}
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func (c *WSClient) subscribe(conn *websocket.Conn, key string, sub WSSubscription) error {
	frame, err := proto.Marshal(&datafeedv1.WebsocketRequest{
		UserId:  strconv.FormatInt(sub.UserID, 10),
		Channel: sub.Channel,
		Key:     key,
	})
	if err != nil {
		return fmt.Errorf("stockbit ws: encode subscribe frame: %w", err)
	}
	return c.write(conn, frame)
}

// readLoop consumes frames until the connection fails or ctx is cancelled.
func (c *WSClient) readLoop(ctx context.Context, conn *websocket.Conn, handler WSHandler) error {
	for {
		if ctx.Err() != nil {
			return nil
		}
		if err := conn.SetReadDeadline(time.Now().Add(c.opts.readTimeout)); err != nil {
			return err
		}
		_, payload, err := conn.ReadMessage()
		if err != nil {
			return err
		}

		msg := &datafeedv1.WebsocketWrapMessageChannel{}
		if err := proto.Unmarshal(payload, msg); err != nil {
			c.logDebug("stockbit ws: undecodable frame", "error", err, "raw", truncate(string(payload)))
			continue
		}

		// A server ping is acknowledged with the same keepalive ping frame.
		if msg.GetPing() != nil {
			frame, err := buildPingFrame()
			if err != nil {
				return fmt.Errorf("stockbit ws: encode ping reply: %w", err)
			}
			if err := c.write(conn, frame); err != nil {
				return err
			}
			continue
		}

		if err := handler(ctx, msg); err != nil {
			c.logWarn("stockbit ws: handler error", "error", err)
		}
	}
}

// pingLoop sends keepalive ping frames on the interval until the connection
// drops or the context is cancelled.
func (c *WSClient) pingLoop(conn *websocket.Conn, ctx context.Context, done chan<- struct{}) {
	defer close(done)
	frame, err := buildPingFrame()
	if err != nil {
		c.logWarn("stockbit ws: encode ping frame", "error", err)
		return
	}
	t := time.NewTicker(c.opts.pingInterval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if err := c.write(conn, frame); err != nil {
				c.logDebug("stockbit ws: ping failed", "error", err)
				return
			}
		}
	}
}

// buildPingFrame marshals the byte-exact datafeed ping
// (IgYKBHBpbmc=, verified in wire_compat_test.go).
func buildPingFrame() ([]byte, error) {
	return proto.Marshal(&datafeedv1.WebsocketRequest{
		Ping: &datafeedv1.PingRequest{Message: "ping"},
	})
}

// write serializes writes so the ping loop and the read loop never write
// concurrently.
func (c *WSClient) write(conn *websocket.Conn, frame []byte) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	if err := conn.SetWriteDeadline(time.Now().Add(c.opts.writeTimeout)); err != nil {
		return err
	}
	return conn.WriteMessage(websocket.BinaryMessage, frame)
}

func (c *WSClient) nextBackoff(backoff time.Duration) time.Duration {
	next := backoff * 2
	if next > c.opts.backoffMax {
		return c.opts.backoffMax
	}
	return next
}

func (c *WSClient) setConn(conn *websocket.Conn) {
	c.connMu.Lock()
	c.conn = conn
	c.connMu.Unlock()
}

func (c *WSClient) clearConn(conn *websocket.Conn) {
	c.connMu.Lock()
	if c.conn == conn {
		c.conn = nil
	}
	c.connMu.Unlock()
}

func (c *WSClient) logDebug(msg string, args ...any) {
	if c.opts.logger != nil {
		c.opts.logger.Debug(msg, args...)
	}
}

func (c *WSClient) logInfo(msg string, args ...any) {
	if c.opts.logger != nil {
		c.opts.logger.Info(msg, args...)
	}
}

func (c *WSClient) logWarn(msg string, args ...any) {
	if c.opts.logger != nil {
		c.opts.logger.Warn(msg, args...)
	}
}

// WSChannelAll subscribes the given symbols on every symbol-array datafeed
// channel (watchlist, order book, running trade, live price, best bid offer,
// and their v3 variants). Channels that need typed per-request messages
// (market mover, order queue, tradebook) are left unset.
func WSChannelAll(symbols ...string) *datafeedv1.WebsocketChannel {
	return &datafeedv1.WebsocketChannel{
		Watchlist:         symbols,
		OrderBook:         symbols,
		RunningTrade:      symbols,
		RunningTradeBatch: symbols,
		Liveprice:         symbols,
		Iepiev:            symbols,
		Intraday:          symbols,
		BestBidOffer:      symbols,
		LivepriceV3:       symbols,
		OrderBookV3:       symbols,
		IntradayV3:        symbols,
	}
}
