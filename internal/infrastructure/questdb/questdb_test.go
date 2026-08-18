package questdb

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	consumerv1 "github.com/nofendian17/sbterm-server/internal/infrastructure/stockbit/proto/securities/transactional/datafeed/consumer/entity/v1"
	datafeedv1 "github.com/nofendian17/sbterm-server/internal/infrastructure/stockbit/proto/securities/transactional/datafeed/v1"
	"github.com/nofendian17/sbterm-server/pkg/log"
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

func TestParseOrderBookBody(t *testing.T) {
	tests := []struct {
		name      string
		body      string
		wantSide  string
		wantLevel []orderBookLevel
		wantErr   bool
	}{
		{
			name:      "offer snapshot with levels and trailing token",
			body:      "#O|BBCA|OFFER|6425;1077;6965000|6450;1545;8579800|1787028158-12811&78061500",
			wantSide:  "OFFER",
			wantLevel: []orderBookLevel{{Price: 6425, Frequency: 1077, Shares: 6965000}, {Price: 6450, Frequency: 1545, Shares: 8579800}},
		},
		{
			name:      "bid snapshot",
			body:      "#O|BBCA|BID|3110;1233;6523600",
			wantSide:  "BID",
			wantLevel: []orderBookLevel{{Price: 3110, Frequency: 1233, Shares: 6523600}},
		},
		{
			name:    "malformed header",
			body:    "garbage",
			wantErr: true,
		},
		{
			name:    "missing side",
			body:    "#O|BBCA||6425;1077;6965000",
			wantErr: true,
		},
		{
			name:      "skips malformed level tokens",
			body:      "#O|BBCA|BID|3110;1233;6523600|broken|3120;1;2",
			wantSide:  "BID",
			wantLevel: []orderBookLevel{{Price: 3110, Frequency: 1233, Shares: 6523600}, {Price: 3120, Frequency: 1, Shares: 2}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			side, levels, err := parseOrderBookBody(tt.body)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantSide, side)
			assert.Equal(t, tt.wantLevel, levels)
		})
	}
}

func TestOrderBookTimestamp(t *testing.T) {
	datetime := "2026-08-18T11:42:38.178723+07:00"
	incoming := "2026-08-18T11:42:38.179680676+07:00"
	wantDateTime, err := time.Parse(time.RFC3339Nano, datetime)
	require.NoError(t, err)
	wantIncoming, err := time.Parse(time.RFC3339Nano, incoming)
	require.NoError(t, err)

	tests := []struct {
		name string
		ob   *consumerv1.Orderbook
		want time.Time
	}{
		{
			name: "datetime wins",
			ob:   &consumerv1.Orderbook{Datetime: datetime, ItchIncomingTime: incoming},
			want: wantDateTime,
		},
		{
			name: "falls back to itch incoming time",
			ob:   &consumerv1.Orderbook{ItchIncomingTime: incoming},
			want: wantIncoming,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, orderBookTimestamp(tt.ob))
		})
	}

	t.Run("falls back to now", func(t *testing.T) {
		before := time.Now()
		got := orderBookTimestamp(&consumerv1.Orderbook{})
		after := time.Now()
		assert.False(t, got.Before(before.Add(-time.Second)))
		assert.False(t, got.After(after.Add(time.Second)))
	})
}

func TestOrderBookSinkStoreRejectsMalformedBody(t *testing.T) {
	c := &Client{
		orderBookTable: DefaultOrderBookTable,
		logger:         log.New(log.WithWriter(io.Discard)),
		schemaOK:       map[string]bool{DefaultOrderBookTable: true},
	}
	s := &orderBookSink{client: c}
	err := s.Store(context.Background(), &consumerv1.Orderbook{Body: "garbage"})
	require.Error(t, err)
	assert.ErrorIs(t, err, errMalformedOrderBook)
}
