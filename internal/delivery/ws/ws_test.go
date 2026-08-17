package ws_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
	"github.com/samber/do/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/nofendian17/sbterm-server/internal/container"
	"github.com/nofendian17/sbterm-server/internal/delivery/ws"
	"github.com/nofendian17/sbterm-server/internal/infrastructure/config"
	"github.com/nofendian17/sbterm-server/internal/infrastructure/stockbit"
	datafeedv1 "github.com/nofendian17/sbterm-server/internal/infrastructure/stockbit/proto/securities/transactional/datafeed/v1"
	"github.com/nofendian17/sbterm-server/pkg/log"
)

func TestWSStartsWithSubscriptionsAndStops(t *testing.T) {
	// Fake Stockbit REST API: answers the websocket key used in the handshake.
	keyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/auth/websocket/key", r.URL.Path)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message":"ok","data":{"key":"ws-key"}}`))
	}))
	defer keyServer.Close()

	expected := []*datafeedv1.WebsocketChannel{
		stockbit.WSChannelRunningTradeBatch("*"),
		stockbit.WSChannelOrderBookV3("BBCA", "BBRI"),
	}

	var connections sync.WaitGroup
	upgrader := websocket.Upgrader{}
	subscribeFrames := make(chan *datafeedv1.WebsocketRequest, len(expected))
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := upgrader.Upgrade(w, r, nil)
		require.NoError(t, err)
		connections.Add(1)
		defer connections.Done()
		defer c.Close()
		assert.Equal(t, "ws-key", r.URL.Query().Get("wskey"))
		require.NoError(t, c.SetReadDeadline(time.Now().Add(2*time.Second)))
		_, payload, err := c.ReadMessage()
		require.NoError(t, err)
		auth := &datafeedv1.WebsocketRequest{}
		require.NoError(t, proto.Unmarshal(payload, auth))
		assert.Equal(t, "667557", auth.GetUserId())
		assert.Equal(t, "ws-key", auth.GetKey())
		assert.Nil(t, auth.GetChannel(), "first frame must be the channel-less auth frame")
		_, payload, err = c.ReadMessage()
		require.NoError(t, err)
		req := &datafeedv1.WebsocketRequest{}
		require.NoError(t, proto.Unmarshal(payload, req))
		assert.Equal(t, "667557", req.GetUserId())
		assert.Equal(t, "ws-key", req.GetKey())
		subscribeFrames <- req
		// Hold the connection open until the client shuts it down.
		for {
			if _, _, err := c.ReadMessage(); err != nil {
				return
			}
		}
	}))
	defer srv.Close()

	rdb := miniredis.RunT(t)
	redisURL := fmt.Sprintf("redis://%s/0", rdb.Addr())
	rawClient := redis.NewClient(&redis.Options{Addr: rdb.Addr()})
	defer rawClient.Close()
	store := stockbit.NewRedisTokenStore(rawClient)
	require.NoError(t, store.Set(context.Background(), &stockbit.TokenData{
		Access:  stockbit.TokenPair{Token: "at", ExpiredAt: "2030-01-01T00:00:00Z"},
		Refresh: stockbit.TokenPair{Token: "rt", ExpiredAt: "2030-01-01T00:00:00Z"},
		UserID:  667557,
	}))

	cfg := newTestConfig("postgres://user:pass@127.0.0.1:1/db?sslmode=disable&connect_timeout=1", redisURL)
	cfg.Stockbit.BaseURL = keyServer.URL
	cfg.Stockbit.WSURL = "ws" + strings.TrimPrefix(srv.URL, "http")
	cfg.Stockbit.WSSubscriptions = []config.WSSubscriptionConfig{
		{
			Name:     "running_trade_batch_all",
			Channels: config.WSChannelConfig{RunningTradeBatch: []string{"*"}},
		},
		{
			Name:     "order_book_v3_selected",
			Channels: config.WSChannelConfig{OrderBookV3: []string{"BBCA", "BBRI"}},
		},
	}

	logger := log.New(log.WithWriter(io.Discard))
	injector := container.New(cfg, logger)

	svc, err := do.Invoke[*ws.Service](injector)
	require.NoError(t, err)

	svc.Start()
	defer func() { injector.ShutdownWithContext(context.Background()) }()

	received := make([]*datafeedv1.WebsocketChannel, 0, len(expected))
	for range expected {
		select {
		case req := <-subscribeFrames:
			received = append(received, req.GetChannel())
		case <-time.After(2 * time.Second):
			t.Fatal("ws server did not receive all boot subscriptions")
		}
	}
	for _, want := range expected {
		found := false
		for _, got := range received {
			if proto.Equal(want, got) {
				found = true
				break
			}
		}
		require.True(t, found, "expected subscription %v was not received", want)
	}

	require.NoError(t, svc.Shutdown())

	closed := make(chan struct{})
	go func() {
		connections.Wait()
		close(closed)
	}()
	select {
	case <-closed:
	case <-time.After(2 * time.Second):
		t.Fatal("ws clients did not close their connections on shutdown")
	}
}

func TestBuildChannel(t *testing.T) {
	tests := []struct {
		name string
		ch   config.WSChannelConfig
		want *datafeedv1.WebsocketChannel
	}{
		{
			name: "empty config subscribes nothing",
			ch:   config.WSChannelConfig{},
			want: &datafeedv1.WebsocketChannel{},
		},
		{
			name: "wildcard running trade batch",
			ch:   config.WSChannelConfig{RunningTradeBatch: []string{"*"}},
			want: stockbit.WSChannelRunningTradeBatch("*"),
		},
		{
			name: "selected order book v3",
			ch:   config.WSChannelConfig{OrderBookV3: []string{"BBCA", "BBRI"}},
			want: stockbit.WSChannelOrderBookV3("BBCA", "BBRI"),
		},
		{
			name: "multiple channels in one subscription",
			ch: config.WSChannelConfig{
				RunningTradeBatch: []string{"*"},
				OrderBookV3:       []string{"BBCA"},
				LivepriceV3:       []string{"BBCA"},
			},
			want: stockbit.MergeWSChannels(
				stockbit.WSChannelRunningTradeBatch("*"),
				stockbit.WSChannelOrderBookV3("BBCA"),
				stockbit.WSChannelLivepriceV3("BBCA"),
			),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.True(t, proto.Equal(tt.want, ws.BuildChannel(tt.ch)))
		})
	}
}

func newTestConfig(databaseURL, redisURL string) *config.Config {
	return &config.Config{
		App: config.AppConfig{
			Name:    "test-app",
			Version: "1.0.0",
		},
		Port: ":9999",
		Database: config.DatabaseConfig{
			URL:             databaseURL,
			MaxConns:        10,
			MinConns:        0,
			MaxConnLifetime: 30 * time.Minute,
			MaxConnIdleTime: 5 * time.Minute,
		},
		Redis: config.RedisConfig{
			URL:          redisURL,
			MaxRetries:   1,
			PoolSize:     1,
			MinIdleConns: 0,
			DialTimeout:  time.Second,
			ReadTimeout:  time.Second,
			WriteTimeout: time.Second,
		},
		Log: config.LogConfig{
			Level:     "info",
			Format:    "text",
			AddSource: false,
		},
		RateLimit: config.RateLimitConfig{
			Rate:  100,
			Burst: 200,
		},
		HTTP: config.HTTPConfig{
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
	}
}
