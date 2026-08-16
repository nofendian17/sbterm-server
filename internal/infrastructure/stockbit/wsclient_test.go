package stockbit

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	ordertradev1 "github.com/nofendian17/sbterm-server/internal/infrastructure/stockbit/proto/financial/order_trade/entity/v1"
	datafeedv1 "github.com/nofendian17/sbterm-server/internal/infrastructure/stockbit/proto/securities/transactional/datafeed/v1"
)

// wsURL converts an httptest server URL into a ws:// URL.
func wsURL(srv *httptest.Server) string {
	return "ws" + strings.TrimPrefix(srv.URL, "http")
}

// newWSUpgradeServer runs a websocket server that upgrades each connection and
// hands it to handle; returning from handle closes the connection.
func newWSUpgradeServer(t *testing.T, handle func(*websocket.Conn, *http.Request)) *httptest.Server {
	t.Helper()
	upgrader := websocket.Upgrader{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		handle(c, r)
	}))
	t.Cleanup(srv.Close)
	return srv
}

// readBinary reads the next binary message from c, failing on a timeout so the
// test never hangs.
func readBinary(t *testing.T, c *websocket.Conn) []byte {
	t.Helper()
	require.NoError(t, c.SetReadDeadline(time.Now().Add(2*time.Second)))
	_, payload, err := c.ReadMessage()
	require.NoError(t, err, "expected a binary frame from the client")
	return payload
}

// decodeSubscribe decodes a client frame into a datafeed WebsocketRequest.
func decodeSubscribe(t *testing.T, payload []byte) *datafeedv1.WebsocketRequest {
	t.Helper()
	req := &datafeedv1.WebsocketRequest{}
	require.NoError(t, proto.Unmarshal(payload, req))
	return req
}

// drain discards frames on extra connections created by reconnects.
func drain(t *testing.T, c *websocket.Conn) {
	for {
		if _, _, err := c.ReadMessage(); err != nil {
			return
		}
	}
}

// pingFrame is the byte-exact datafeed ping verified in wire_compat_test.go.
var pingFrame = []byte{0x22, 0x06, 0x0A, 0x04, 0x70, 0x69, 0x6E, 0x67}

// readAuthThenSubscribe consumes the authentication frame and the subscription
// frame the client sends on connect, in that order.
func readAuthThenSubscribe(t *testing.T, c *websocket.Conn) (*datafeedv1.WebsocketRequest, *datafeedv1.WebsocketRequest) {
	t.Helper()
	return decodeSubscribe(t, readBinary(t, c)), decodeSubscribe(t, readBinary(t, c))
}

// TestWSClientSendsAuthFrameBeforeSubscribe asserts the client authenticates
// with a channel-less frame before the subscription, exactly like the
// stockbit.com frontend.
func TestWSClientSendsAuthFrameBeforeSubscribe(t *testing.T) {
	const wskey = "l8IDNJKcalsaSZZCOR6A9K5BlPEpeuu542B4Fp6J4vA="
	serverDone := make(chan struct{})
	srv := newWSUpgradeServer(t, func(c *websocket.Conn, r *http.Request) {
		defer close(serverDone)
		auth, sub := readAuthThenSubscribe(t, c)
		assert.Equal(t, "42", auth.GetUserId())
		assert.Equal(t, wskey, auth.GetKey())
		assert.Equal(t, "at-123", auth.GetAccessToken())
		assert.Nil(t, auth.GetChannel(), "auth frame must not carry a channel")
		assert.NotNil(t, sub.GetChannel())
		assert.Equal(t, "42", sub.GetUserId())
		assert.Equal(t, "at-123", sub.GetAccessToken())
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c := NewWSClient(wsURL(srv), func(ctx context.Context) (string, error) { return wskey, nil },
		WithWSAccessTokenProvider(func(ctx context.Context) (string, error) { return "at-123", nil }),
		WithWSReconnectBackoff(time.Millisecond, time.Millisecond))
	done := make(chan error, 1)
	go func() {
		done <- c.Run(ctx, WSSubscription{UserID: 42, Channel: WSChannelWatchlist("BBCA")},
			func(ctx context.Context, m *datafeedv1.WebsocketWrapMessageChannel) error { return nil })
	}()

	select {
	case <-serverDone:
	case <-time.After(2 * time.Second):
		t.Fatal("server did not receive the auth and subscribe frames")
	}
	cancel()
	require.NoError(t, <-done)
}

func TestWSClientSubscribeFrame(t *testing.T) {
	const wskey = "l8IDNJKcalsaSZZCOR6A9K5BlPEpeuu542B4Fp6J4vA="
	serverDone := make(chan struct{})
	srv := newWSUpgradeServer(t, func(c *websocket.Conn, r *http.Request) {
		defer close(serverDone)
		// The wskey must be attached verbatim, exactly like
		// wss://.../?wskey=l8IDNJKcalsaSZZCOR6A9K5BlPEpeuu542B4Fp6J4vA=.
		assert.Equal(t, "wskey="+wskey, r.URL.RawQuery)
		_, req := readAuthThenSubscribe(t, c)
		assert.Equal(t, "42", req.GetUserId())
		assert.Equal(t, wskey, req.GetKey())
		assert.Equal(t, "at-123", req.GetAccessToken())
		assert.Equal(t, []string{"BBCA", "BBRI"}, req.GetChannel().GetWatchlist())
		assert.Equal(t, []string{"BBCA", "BBRI"}, req.GetChannel().GetOrderBook())
		assert.Equal(t, []string{"BBCA", "BBRI"}, req.GetChannel().GetRunningTrade())
		assert.Equal(t, []string{"BBCA", "BBRI"}, req.GetChannel().GetLiveprice())
		assert.Equal(t, []string{"BBCA", "BBRI"}, req.GetChannel().GetBestBidOffer())
		assert.NotNil(t, req.GetChannel().GetLivepriceV3())
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c := NewWSClient(wsURL(srv), func(ctx context.Context) (string, error) {
		return wskey, nil
	}, WithWSAccessTokenProvider(func(ctx context.Context) (string, error) {
		return "at-123", nil
	}), WithWSReconnectBackoff(time.Millisecond, time.Millisecond))
	done := make(chan error, 1)
	go func() {
		done <- c.Run(ctx, WSSubscription{
			UserID: 42,
			Channel: MergeWSChannels(
				WSChannelWatchlist("BBCA", "BBRI"),
				WSChannelOrderBook("BBCA", "BBRI"),
				WSChannelRunningTrade("BBCA", "BBRI"),
				WSChannelLiveprice("BBCA", "BBRI"),
				WSChannelBestBidOffer("BBCA", "BBRI"),
				WSChannelLivepriceV3("BBCA", "BBRI"),
			),
		}, func(ctx context.Context, m *datafeedv1.WebsocketWrapMessageChannel) error { return nil })
	}()

	select {
	case <-serverDone:
	case <-time.After(2 * time.Second):
		t.Fatal("server did not receive a subscribe frame")
	}
	cancel()
	require.NoError(t, <-done)
}

func TestWSClientDispatchesDecodedFrames(t *testing.T) {
	srv := newWSUpgradeServer(t, func(c *websocket.Conn, r *http.Request) {
		_, _ = readAuthThenSubscribe(t, c)
		frame, err := proto.Marshal(&datafeedv1.WebsocketWrapMessageChannel{
			MessageChannel: &datafeedv1.WebsocketWrapMessageChannel_RunningTrade{
				RunningTrade: &datafeedv1.RunningTrade{Stock: "BBCA", Price: 6400, Volume: 100},
			},
		})
		require.NoError(t, err)
		require.NoError(t, c.WriteMessage(websocket.BinaryMessage, frame))
		drain(t, c)
	})

	got := make(chan *datafeedv1.WebsocketWrapMessageChannel, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c := NewWSClient(wsURL(srv), func(ctx context.Context) (string, error) { return "k", nil },
		WithWSReconnectBackoff(time.Millisecond, time.Millisecond))
	done := make(chan error, 1)
	go func() {
		done <- c.Run(ctx, WSSubscription{UserID: 1, Channel: WSChannelWatchlist("BBCA")}, func(ctx context.Context, m *datafeedv1.WebsocketWrapMessageChannel) error {
			got <- m
			return nil
		})
	}()

	select {
	case m := <-got:
		rt := m.GetRunningTrade()
		require.NotNil(t, rt)
		assert.Equal(t, "BBCA", rt.GetStock())
		assert.InDelta(t, 6400, rt.GetPrice(), 0.001)
		assert.InDelta(t, 100, rt.GetVolume(), 0.001)
	case <-time.After(2 * time.Second):
		t.Fatal("handler was not invoked with the decoded frame")
	}
	cancel()
	require.NoError(t, <-done)
}

func TestWSClientKeepalivePing(t *testing.T) {
	var dials atomic.Int32
	serverDone := make(chan struct{})
	srv := newWSUpgradeServer(t, func(c *websocket.Conn, r *http.Request) {
		if dials.Add(1) != 1 {
			drain(t, c)
			return
		}
		defer close(serverDone)
		_, _ = readAuthThenSubscribe(t, c)
		assert.Equal(t, pingFrame, readBinary(t, c), "periodic ping must be byte-exact")

		// A server-initiated ping must be answered with the same ping frame.
		serverPing, err := proto.Marshal(&datafeedv1.WebsocketWrapMessageChannel{
			MessageChannel: &datafeedv1.WebsocketWrapMessageChannel_Ping{
				Ping: &datafeedv1.PingResponse{},
			},
		})
		require.NoError(t, err)
		require.NoError(t, c.WriteMessage(websocket.BinaryMessage, serverPing))
		assert.Equal(t, pingFrame, readBinary(t, c), "server ping must be answered")
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c := NewWSClient(wsURL(srv), func(ctx context.Context) (string, error) { return "k", nil },
		WithWSPingInterval(20*time.Millisecond),
		WithWSReconnectBackoff(time.Millisecond, time.Millisecond))
	done := make(chan error, 1)
	go func() {
		done <- c.Run(ctx, WSSubscription{UserID: 1, Channel: WSChannelWatchlist("BBCA")},
			func(ctx context.Context, m *datafeedv1.WebsocketWrapMessageChannel) error { return nil })
	}()

	select {
	case <-serverDone:
	case <-time.After(2 * time.Second):
		t.Fatal("keepalive handshake was not observed")
	}
	cancel()
	require.NoError(t, <-done)
}

func TestWSClientReconnectsAndResubscribes(t *testing.T) {
	var dials atomic.Int32
	var subscribedKey atomic.Value

	srv := newWSUpgradeServer(t, func(c *websocket.Conn, r *http.Request) {
		n := dials.Add(1)
		subscribedKey.Store(r.URL.Query().Get("wskey"))
		_, req := readAuthThenSubscribe(t, c)
		assert.Equal(t, "1", req.GetUserId())
		if n == 1 {
			require.NoError(t, c.Close()) // drop the first connection; the client must redial
			return
		}
		// Second connection stays up and delivers a frame.
		frame, err := proto.Marshal(&datafeedv1.WebsocketWrapMessageChannel{
			MessageChannel: &datafeedv1.WebsocketWrapMessageChannel_OrderbookBody{
				OrderbookBody: &datafeedv1.OrderBookBody{StockSymbol: "BBCA"},
			},
		})
		require.NoError(t, err)
		require.NoError(t, c.WriteMessage(websocket.BinaryMessage, frame))
		drain(t, c)
	})

	var calls atomic.Int32
	c := NewWSClient(wsURL(srv), func(ctx context.Context) (string, error) {
		if calls.Add(1) == 1 {
			return "ws-key-1", nil
		}
		return "ws-key-2", nil
	}, WithWSReconnectBackoff(time.Millisecond, time.Millisecond))

	got := make(chan *datafeedv1.WebsocketWrapMessageChannel, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() {
		done <- c.Run(ctx, WSSubscription{UserID: 1, Channel: WSChannelWatchlist("BBCA")},
			func(ctx context.Context, m *datafeedv1.WebsocketWrapMessageChannel) error {
				got <- m
				return nil
			})
	}()

	select {
	case m := <-got:
		require.NotNil(t, m.GetOrderbookBody())
		assert.Equal(t, "BBCA", m.GetOrderbookBody().GetStockSymbol())
	case <-time.After(2 * time.Second):
		t.Fatalf("did not receive a frame after reconnect (dials=%d)", dials.Load())
	}

	// The second connection authenticated with a freshly fetched key.
	waitFor(t, func() bool { return dials.Load() >= 2 })
	assert.Equal(t, "ws-key-2", subscribedKey.Load())
	cancel()
	require.NoError(t, <-done)
}

// TestWSZeroOptionsKeepDefaults asserts that zero-valued durations passed via
// options (e.g. from unset config) keep the built-in defaults instead of
// producing a zero/panicking ticker.
func TestWSZeroOptionsKeepDefaults(t *testing.T) {
	c := NewWSClient("wss://example.com/", func(ctx context.Context) (string, error) { return "k", nil },
		WithWSDialTimeout(0),
		WithWSPingInterval(0),
		WithWSReadTimeout(0),
		WithWSWriteTimeout(0),
		WithWSReconnectBackoff(0, 0),
	)
	assert.Equal(t, 10*time.Second, c.opts.dialTimeout)
	assert.Equal(t, 30*time.Second, c.opts.pingInterval)
	assert.Equal(t, 90*time.Second, c.opts.readTimeout)
	assert.Equal(t, 10*time.Second, c.opts.writeTimeout)
	assert.Equal(t, time.Second, c.opts.backoffInit)
	assert.Equal(t, 30*time.Second, c.opts.backoffMax)
}

// TestMergeWSChannelsCopiesSymbolChannels asserts merging combines each
// builder's field without touching typed channels.
func TestMergeWSChannelsCopiesSymbolChannels(t *testing.T) {
	got := MergeWSChannels(
		WSChannelWatchlist("BBCA"),
		WSChannelOrderBook("BBRI"),
		WSChannelRunningTrade("BBCA", "BBRI"),
	)
	assert.Equal(t, []string{"BBCA"}, got.GetWatchlist())
	assert.Equal(t, []string{"BBRI"}, got.GetOrderBook())
	assert.Equal(t, []string{"BBCA", "BBRI"}, got.GetRunningTrade())
	assert.Empty(t, got.GetLiveprice())
	assert.Empty(t, got.GetMarketMover(), "typed channels must stay unset")
	assert.Empty(t, got.GetOrderQueue(), "typed channels must stay unset")
	assert.Empty(t, got.GetTradebook(), "typed channels must stay unset")
}

// TestWSChannelBuildersSetOnlyTheirField asserts each symbol-array builder
// populates exactly its own channel and leaves every other one unset.
func TestWSChannelBuildersSetOnlyTheirField(t *testing.T) {
	symbols := []string{"BBCA"}
	builders := map[string]func(...string) *datafeedv1.WebsocketChannel{
		"watchlist":           WSChannelWatchlist,
		"order book":          WSChannelOrderBook,
		"running trade":       WSChannelRunningTrade,
		"running trade batch": WSChannelRunningTradeBatch,
		"liveprice":           WSChannelLiveprice,
		"iepiev":              WSChannelIepiev,
		"intraday":            WSChannelIntraday,
		"best bid offer":      WSChannelBestBidOffer,
		"liveprice v3":        WSChannelLivepriceV3,
		"order book v3":       WSChannelOrderBookV3,
		"intraday v3":         WSChannelIntradayV3,
	}
	getters := map[string]func(*datafeedv1.WebsocketChannel) []string{
		"watchlist":           (*datafeedv1.WebsocketChannel).GetWatchlist,
		"order book":          (*datafeedv1.WebsocketChannel).GetOrderBook,
		"running trade":       (*datafeedv1.WebsocketChannel).GetRunningTrade,
		"running trade batch": (*datafeedv1.WebsocketChannel).GetRunningTradeBatch,
		"liveprice":           (*datafeedv1.WebsocketChannel).GetLiveprice,
		"iepiev":              (*datafeedv1.WebsocketChannel).GetIepiev,
		"intraday":            (*datafeedv1.WebsocketChannel).GetIntraday,
		"best bid offer":      (*datafeedv1.WebsocketChannel).GetBestBidOffer,
		"liveprice v3":        (*datafeedv1.WebsocketChannel).GetLivepriceV3,
		"order book v3":       (*datafeedv1.WebsocketChannel).GetOrderBookV3,
		"intraday v3":         (*datafeedv1.WebsocketChannel).GetIntradayV3,
	}

	for name, build := range builders {
		t.Run(name, func(t *testing.T) {
			ch := build(symbols...)
			for field, get := range getters {
				if field == name {
					assert.Equal(t, symbols, get(ch), "own field %q must carry the symbols", field)
				} else {
					assert.Empty(t, get(ch), "field %q must stay unset", field)
				}
			}
		})
	}
}

// TestWSChannelTypedBuilders asserts the typed channel builders carry exactly
// the given requests and never touch symbol-array channels.
func TestWSChannelTypedBuilders(t *testing.T) {
	mover := &ordertradev1.MarketMoverWebsocketRequest{CatalogId: 7}
	queue := &ordertradev1.OrderQueueWebsocketRequest{StockCode: "BBCA", Price: 6425}
	book := &ordertradev1.TradebookWebsocketRequest{Symbol: "BBCA", Board: "IDX"}

	moverCh := WSChannelMarketMover(mover)
	assert.Equal(t, []*ordertradev1.MarketMoverWebsocketRequest{mover}, moverCh.GetMarketMover())
	assert.Empty(t, moverCh.GetOrderQueue())
	assert.Empty(t, moverCh.GetTradebook())
	assert.Empty(t, moverCh.GetWatchlist())

	queueCh := WSChannelOrderQueue(queue)
	assert.Equal(t, []*ordertradev1.OrderQueueWebsocketRequest{queue}, queueCh.GetOrderQueue())
	assert.Empty(t, queueCh.GetMarketMover())
	assert.Empty(t, queueCh.GetTradebook())
	assert.Empty(t, queueCh.GetWatchlist())

	bookCh := WSChannelTradebook(book)
	assert.Equal(t, []*ordertradev1.TradebookWebsocketRequest{book}, bookCh.GetTradebook())
	assert.Empty(t, bookCh.GetMarketMover())
	assert.Empty(t, bookCh.GetOrderQueue())
	assert.Empty(t, bookCh.GetWatchlist())
}

// TestWSClientSubscribeFrameRoundTrips asserts that a subscribe frame built by
// the client marshals and decodes back to the reconstructed shape from
// wire_compat_test.go (userId/channels/wskey).
func TestWSClientSubscribeFrameRoundTrips(t *testing.T) {
	syms := []string{"IHSG", "BBRI", "BBCA"}
	req := &datafeedv1.WebsocketRequest{
		UserId:  "667557",
		Channel: WSChannelWatchlist(syms...),
		Key:     "wskey-session",
	}
	b, err := proto.Marshal(req)
	require.NoError(t, err)
	back := decodeSubscribe(t, b)
	assert.Equal(t, syms, back.GetChannel().GetWatchlist())
	assert.Equal(t, "667557", back.GetUserId())
	assert.Equal(t, "wskey-session", back.GetKey())
}
