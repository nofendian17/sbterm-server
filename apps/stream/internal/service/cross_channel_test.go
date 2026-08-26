package service

import (
	"context"
	"encoding/json"
	"testing"

	datafeedv1 "github.com/nofendian17/sbterm/libs/proto/securities/transactional/datafeed/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	consumerv1 "github.com/nofendian17/sbterm/libs/proto/securities/transactional/datafeed/consumer/entity/v1"
)

// Reproduksi persis skenario live: client hanya subscribe orderbook,
// lalu kedua route di-broadcast. running_trade tidak boleh lolos.
func TestNoCrossChannelLeak(t *testing.T) {
	hub := NewHub(discardLogger())
	obSub := NewClient(hub, nil)
	obSub.Subscribe(ChannelOrderBook, []string{"BBCA", "BBRI", "BUMI"})
	hub.Register(obSub)

	rtRoute := RunningTradeRoute(hub, discardLogger())
	obRoute := OrderBookRoute(hub, discardLogger())

	batch := &datafeedv1.RunningTradeBatch{Batch: []*datafeedv1.RunningTrade{
		{Stock: "GTSI", Price: 100, Volume: 100},
	}}
	raw, err := proto.Marshal(batch)
	require.NoError(t, err)
	require.NoError(t, rtRoute(context.Background(), raw))

	ob := &consumerv1.Orderbook{StockCode: "BUMI", Body: "#O|BUMI|BID|190;1;100"}
	raw2, err := proto.Marshal(ob)
	require.NoError(t, err)
	require.NoError(t, obRoute(context.Background(), raw2))

	received := receiveAll(obSub)
	types := map[string]int{}
	for _, m := range received {
		var p struct {
			Type string `json:"type"`
		}
		_ = jsonUnmarshalHelper(m, &p)
		types[p.Type]++
	}
	assert.Equal(t, 0, types["running_trade"], "running_trade leaked into orderbook subscriber")
	assert.Equal(t, 1, types["orderbook"])
}

func jsonUnmarshalHelper(data []byte, v any) error {
	return json.Unmarshal(data, v)
}
