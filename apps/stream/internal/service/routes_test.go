package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	consumerv1 "github.com/nofendian17/sbterm/libs/proto/securities/transactional/datafeed/consumer/entity/v1"
)

func TestRoutesOrderBookFrame(t *testing.T) {
	hub := NewHub(discardLogger())
	sub := NewClient(hub, nil)
	sub.Subscribe(ChannelOrderBook, []string{"BBCA"})
	hub.Register(sub)
	svc := NewServiceRoutes(&fakePoller{}, hub, Routes{
		"datafeed.order_book": OrderBookRoute(hub, discardLogger()),
	}, discardLogger())

	ob := &consumerv1.Orderbook{
		StockCode:      "BBCA",
		Body:           "#O|BBCA|BID|7750;12;340000|7745;3;125000",
		SequenceNumber: 41,
		Datetime:       time.Date(2026, 8, 27, 9, 15, 0, 0, time.UTC).Format(time.RFC3339Nano),
	}
	raw, err := proto.Marshal(ob)
	require.NoError(t, err)

	svc.handleRecord(context.Background(), "datafeed.order_book", raw)

	received := receiveAll(sub)
	require.Len(t, received, 1)

	var env struct {
		Type   string `json:"type"`
		Symbol string `json:"symbol"`
		Data   struct {
			Side     string      `json:"side"`
			Seq      int64       `json:"seq"`
			Levels   [][]float64 `json:"levels"`
			Datetime string      `json:"datetime"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(received[0], &env))
	assert.Equal(t, "orderbook", env.Type)
	assert.Equal(t, "BBCA", env.Symbol)
	assert.Equal(t, "BID", env.Data.Side)
	assert.EqualValues(t, 41, env.Data.Seq)
	require.Len(t, env.Data.Levels, 2)
	assert.EqualValues(t, 7750, env.Data.Levels[0][0])
	assert.EqualValues(t, 340000, env.Data.Levels[0][2])

	// A malformed frame must be dropped without panicking or redelivering.
	assert.NotPanics(t, func() {
		_ = svc.handleRecord(context.Background(), "datafeed.order_book", []byte("not-proto"))
	})
}

func TestRoutesAlertPassthrough(t *testing.T) {
	hub := NewHub(discardLogger())
	sub := NewClient(hub, nil)
	sub.Subscribe(ChannelAlerts, []string{"BBCA"})
	hub.Register(sub)
	svc := NewServiceRoutes(&fakePoller{}, hub, Routes{
		"datafeed.alerts": AlertsRoute(hub, discardLogger()),
	}, discardLogger())

	payload := []byte(`{"symbol":"BBCA","type":"PULL_BID","side":"BID","ts":"2026-08-27T09:15:00Z","detail":{"events":2}}`)
	svc.handleRecord(context.Background(), "datafeed.alerts", payload)

	received := receiveAll(sub)
	require.Len(t, received, 1)

	var env struct {
		Type   string `json:"type"`
		Symbol string `json:"symbol"`
		Data   struct {
			Type string `json:"type"`
			Side string `json:"side"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(received[0], &env))
	assert.Equal(t, "alerts", env.Type)
	assert.Equal(t, "BBCA", env.Symbol)
	assert.Equal(t, "PULL_BID", env.Data.Type)
}

func TestUnknownTopicStillDropsQuietly(t *testing.T) {
	svc := NewService(&fakePoller{}, NewHub(discardLogger()), "rt", discardLogger())
	require.NoError(t, svc.handleRecord(context.Background(), "mystery", []byte("{}")))
}
