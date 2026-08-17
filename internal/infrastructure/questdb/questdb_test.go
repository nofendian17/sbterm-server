package questdb

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/timestamppb"

	datafeedv1 "github.com/nofendian17/sbterm-server/internal/infrastructure/stockbit/proto/securities/transactional/datafeed/v1"
)

func TestChangeAccessors(t *testing.T) {
	assert.Zero(t, changeValue(&datafeedv1.RunningTrade{}))
	assert.Zero(t, changePercentage(&datafeedv1.RunningTrade{}))

	trade := &datafeedv1.RunningTrade{
		Change: &datafeedv1.Change{Value: 25.5, Percentage: 1.25},
	}
	assert.Equal(t, 25.5, changeValue(trade))
	assert.Equal(t, 1.25, changePercentage(trade))
}

func TestTradeTimestamp(t *testing.T) {
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
			assert.Equal(t, tt.want, tradeTimestamp(tt.trade))
		})
	}

	t.Run("falls back to now", func(t *testing.T) {
		before := time.Now()
		got := tradeTimestamp(&datafeedv1.RunningTrade{})
		after := time.Now()
		assert.False(t, got.Before(before.Add(-time.Second)))
		assert.False(t, got.After(after.Add(time.Second)))
	})
}
