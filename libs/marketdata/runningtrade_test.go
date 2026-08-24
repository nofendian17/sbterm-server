package marketdata

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	consumerv1 "github.com/nofendian17/sbterm/libs/proto/securities/transactional/datafeed/consumer/entity/v1"
	datafeedv1 "github.com/nofendian17/sbterm/libs/proto/securities/transactional/datafeed/v1"
)

func TestNewTradeProjectsAllColumns(t *testing.T) {
	tradeTime := timestamppb.New(time.Date(2026, 8, 24, 1, 2, 3, 0, time.UTC))
	wsTime := timestamppb.New(time.Date(2026, 8, 24, 1, 2, 3, 500000000, time.UTC))

	got := NewTrade(&datafeedv1.RunningTrade{
		Stock:         "BBCA",
		Price:         8250,
		Volume:        150.9, // float in proto; DB stores int64
		Action:        datafeedv1.TradeType_TRADE_TYPE_SELL,
		IsGlobal:      true,
		Time:          tradeTime,
		Change:        &datafeedv1.Change{Value: -25, Percentage: -0.3},
		TradeNumber:   42,
		MarketBoard:   consumerv1.BoardType_BOARD_TYPE_RG,
		Value:         1237500,
		WebsocketTime: wsTime,
	})

	assert.Equal(t, "BBCA", got.Stock)
	assert.Equal(t, "TRADE_TYPE_SELL", got.Action)
	assert.Equal(t, "BOARD_TYPE_RG", got.MarketBoard)
	assert.Equal(t, 8250.0, got.Price)
	assert.Equal(t, int64(150), got.Volume) // truncated like the DB sink
	assert.True(t, got.IsGlobal)
	assert.Equal(t, -25.0, got.ChangeValue)
	assert.Equal(t, -0.3, got.ChangePercentage)
	assert.Equal(t, int64(42), got.TradeNumber)
	assert.Equal(t, 1237500.0, got.Value)
	require.NotNil(t, got.WebsocketTS)
	assert.True(t, got.WebsocketTS.Equal(wsTime.AsTime()))
	assert.True(t, got.TS.Equal(tradeTime.AsTime()))
}

func TestNewTradeDefaults(t *testing.T) {
	got := NewTrade(&datafeedv1.RunningTrade{Stock: "ANTM"})

	assert.Equal(t, "TRADE_TYPE_UNSPECIFIED", got.Action)
	assert.Equal(t, "BOARD_TYPE_UNSPECIFIED", got.MarketBoard)
	assert.Zero(t, got.ChangeValue)
	assert.Zero(t, got.ChangePercentage)
	assert.Nil(t, got.WebsocketTS, "absent websocket_time must stay nil")
	assert.False(t, got.TS.IsZero(), "degenerate frame falls back to now")
}

func TestNewTradeTimestampFallback(t *testing.T) {
	wsTime := time.Date(2026, 8, 17, 1, 2, 3, 0, time.UTC)
	tradeTime := wsTime.Add(2 * time.Second)

	tests := []struct {
		name  string
		trade *datafeedv1.RunningTrade
		want  time.Time
	}{
		{
			name:  "trade time wins",
			trade: &datafeedv1.RunningTrade{Time: timestamppb.New(tradeTime), WebsocketTime: timestamppb.New(wsTime)},
			want:  tradeTime,
		},
		{
			name:  "falls back to websocket time",
			trade: &datafeedv1.RunningTrade{WebsocketTime: timestamppb.New(wsTime)},
			want:  wsTime,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.True(t, tt.want.Equal(NewTrade(tt.trade).TS))
		})
	}

	t.Run("falls back to now", func(t *testing.T) {
		before := time.Now()
		got := NewTrade(&datafeedv1.RunningTrade{}).TS
		after := time.Now()
		assert.False(t, got.Before(before.Add(-time.Second)))
		assert.False(t, got.After(after.Add(time.Second)))
	})
}

func TestNewTradesPreservesOrderAndLength(t *testing.T) {
	batch := []*datafeedv1.RunningTrade{{Stock: "A"}, {Stock: "B"}, {Stock: "C"}}

	got := NewTrades(batch)

	require.Len(t, got, 3)
	for i, want := range []string{"A", "B", "C"} {
		assert.Equal(t, want, got[i].Stock)
	}
}
