package ws_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestWSStartsWithSubscriptionAndStops(t *testing.T) {
	// Fake Stockbit REST API: answers the websocket key used in the handshake.
	keyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/auth/websocket/key", r.URL.Path)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message":"ok","data":{"key":"ws-key"}}`))
	}))
	defer keyServer.Close()

	serverDone := make(chan struct{})
	upgrader := websocket.Upgrader{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := upgrader.Upgrade(w, r, nil)
		require.NoError(t, err)
		defer close(serverDone)
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
		assert.Equal(t, []string{"BBCA", "BBRI"}, req.GetChannel().GetWatchlist())
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
	cfg.Stockbit.WSSymbols = []string{"BBCA", "BBRI"}

	logger := log.New(log.WithWriter(io.Discard))
	injector := container.New(cfg, logger)

	svc, err := do.Invoke[*ws.Service](injector)
	require.NoError(t, err)

	svc.Start()
	defer func() { injector.ShutdownWithContext(context.Background()) }()

	select {
	case <-serverDone:
	case <-time.After(2 * time.Second):
		t.Fatal("ws server did not receive the boot subscription")
	}

	require.NoError(t, svc.Shutdown())
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
