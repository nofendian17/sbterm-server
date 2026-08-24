package service

import (
	"context"
	"encoding/json"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/twmb/franz-go/pkg/kgo"
	"google.golang.org/protobuf/proto"

	datafeedv1 "github.com/nofendian17/sbterm/libs/proto/securities/transactional/datafeed/v1"
)

// fakePoller replays queued fetches, then blocks until ctx is cancelled.
type fakePoller struct {
	mu    sync.Mutex
	queue []kgo.Fetches
}

func (f *fakePoller) PollFetches(ctx context.Context) kgo.Fetches {
	f.mu.Lock()
	if len(f.queue) > 0 {
		fs := f.queue[0]
		f.queue = f.queue[1:]
		f.mu.Unlock()
		return fs
	}
	f.mu.Unlock()
	<-ctx.Done()
	var empty kgo.Fetches
	return empty
}

func validBatchRecord(t *testing.T) []byte {
	t.Helper()
	batch := &datafeedv1.RunningTradeBatch{Batch: []*datafeedv1.RunningTrade{
		{Stock: "BBCA", Price: 8250, Volume: 100},
		{Stock: "BBCA", Price: 8260, Volume: 200},
	}}
	value, err := proto.Marshal(batch)
	require.NoError(t, err)
	return value
}

// capturedEnvelope mirrors only the fields the tests dispatch on.
type capturedEnvelope struct {
	Type   string            `json:"type"`
	Symbol string            `json:"symbol"`
	Data   []json.RawMessage `json:"data"`
}

// wireTrade is the DB-shaped projection clients must receive (marketdata.Trade).
type wireTrade struct {
	Stock            string  `json:"stock"`
	Action           string  `json:"action"`
	MarketBoard      string  `json:"market_board"`
	Price            float64 `json:"price"`
	Volume           int64   `json:"volume"`
	IsGlobal         bool    `json:"is_global"`
	ChangeValue      float64 `json:"change_value"`
	ChangePercentage float64 `json:"change_percentage"`
	TradeNumber      int64   `json:"trade_number"`
	Value            float64 `json:"value"`
	WebsocketTS      *string `json:"websocket_ts"`
	TS               string  `json:"ts"`
}

func newCaptureService(topic string) (*Service, *Client, *Client) {
	hub := NewHub(discardLogger())
	subscribed := NewClient(hub, nil)
	subscribed.Subscribe(ChannelRunningTrade, []string{"BBCA"})
	hub.Register(subscribed)
	outsider := NewClient(hub, nil)
	hub.Register(outsider)
	svc := NewService(&fakePoller{}, hub, topic, discardLogger())
	return svc, subscribed, outsider
}

func TestHandleRecordBroadcastsDecodedBatch(t *testing.T) {
	svc, subscribed, outsider := newCaptureService("datafeed.running_trade_batch")

	svc.handleRecord(context.Background(), "datafeed.running_trade_batch", validBatchRecord(t))

	received := receiveAll(subscribed)
	require.Len(t, received, 1)

	var env capturedEnvelope
	require.NoError(t, json.Unmarshal(received[0], &env))
	assert.Equal(t, "running_trade", env.Type)
	assert.Equal(t, "BBCA", env.Symbol)
	require.Len(t, env.Data, 2)

	// The projection must match the QuestDB row shape (marketdata.Trade).
	var trade wireTrade
	require.NoError(t, json.Unmarshal(env.Data[0], &trade))
	assert.Equal(t, "BBCA", trade.Stock)
	assert.Equal(t, 8250.0, trade.Price)
	assert.Equal(t, int64(100), trade.Volume)
	assert.NotZero(t, trade.TS)
	assert.Empty(t, trade.WebsocketTS, "frame without websocket_time omits the field")

	assert.Empty(t, receiveAll(outsider), "client without subscription receives nothing")
}

func TestHandleRecordSkipsBadRecords(t *testing.T) {
	tests := []struct {
		name  string
		topic string
		value []byte
	}{
		{name: "undecodable payload", topic: "datafeed.running_trade_batch", value: []byte{0xff}},
		{name: "unexpected topic", topic: "datafeed.other", value: validBatchRecord(t)},
		{name: "empty batch", topic: "datafeed.running_trade_batch", value: mustMarshal(t, &datafeedv1.RunningTradeBatch{})},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, subscribed, _ := newCaptureService("datafeed.running_trade_batch")

			svc.handleRecord(context.Background(), tt.topic, tt.value)

			assert.Empty(t, receiveAll(subscribed))
		})
	}
}

func mustMarshal(t *testing.T, m proto.Message) []byte {
	t.Helper()
	value, err := proto.Marshal(m)
	require.NoError(t, err)
	return value
}

func TestStartShutdownLifecycle(t *testing.T) {
	hub := NewHub(discardLogger())
	svc := NewService(&fakePoller{}, hub, "datafeed.running_trade_batch", discardLogger())

	svc.Start()
	svc.Start() // idempotent

	require.NoError(t, svc.Shutdown())
	select {
	case <-svc.done:
	default:
		t.Fatal("expected poll loop to have stopped")
	}

	assert.NoError(t, svc.Shutdown()) // second shutdown stays safe
}
