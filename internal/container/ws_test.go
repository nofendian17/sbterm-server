package container

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

	"github.com/nofendian17/sbterm-server/internal/infrastructure/stockbit"
	datafeedv1 "github.com/nofendian17/sbterm-server/internal/infrastructure/stockbit/proto/securities/transactional/datafeed/v1"
	"github.com/nofendian17/sbterm-server/pkg/log"
)

func TestWSMessageJSON(t *testing.T) {
	msg := &datafeedv1.WebsocketWrapMessageChannel{
		MessageChannel: &datafeedv1.WebsocketWrapMessageChannel_RunningTrade{
			RunningTrade: &datafeedv1.RunningTrade{Stock: "BBCA", Price: 6400},
		},
	}
	out := wsMessageJSON(msg)
	assert.Contains(t, out, `"stock":"BBCA"`)
	assert.Contains(t, out, "6400")
}

func TestWSMessageJSONNotTruncated(t *testing.T) {
	batch := make([]*datafeedv1.RunningTrade, 0, 500)
	for i := 0; i < 500; i++ {
		batch = append(batch, &datafeedv1.RunningTrade{Stock: "BBCA", Price: 6350, Volume: 100})
	}
	msg := &datafeedv1.WebsocketWrapMessageChannel{
		MessageChannel: &datafeedv1.WebsocketWrapMessageChannel_RunningTradeBatch{
			RunningTradeBatch: &datafeedv1.RunningTradeBatch{Batch: batch},
		},
	}
	out := wsMessageJSON(msg)
	assert.Greater(t, len(out), 4096)
	assert.NotContains(t, out, "truncated")
	assert.Contains(t, out, `"batch"`)
}

func TestWSEnabledStartsWithSubscriptionAndStops(t *testing.T) {
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

	cfg := testConfig("postgres://user:pass@127.0.0.1:1/db?sslmode=disable&connect_timeout=1", redisURL)
	cfg.Stockbit.BaseURL = keyServer.URL
	cfg.Stockbit.WSURL = "ws" + strings.TrimPrefix(srv.URL, "http")
	cfg.Stockbit.WSSymbols = []string{"BBCA", "BBRI"}

	logger := log.New(log.WithWriter(io.Discard))
	injector := New(cfg, logger)

	svc, err := do.Invoke[*wsService](injector)
	require.NoError(t, err)

	svc.start()
	defer func() { injector.ShutdownWithContext(context.Background()) }()

	select {
	case <-serverDone:
	case <-time.After(2 * time.Second):
		t.Fatal("ws server did not receive the boot subscription")
	}

	require.NoError(t, svc.Shutdown())
}
