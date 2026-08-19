package service_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"

	"github.com/nofendian17/sbterm/apps/ws/internal/infrastructure/config"
	stockbitws "github.com/nofendian17/sbterm/apps/ws/internal/infrastructure/stockbit"
	"github.com/nofendian17/sbterm/apps/ws/internal/service"
	datafeedv1 "github.com/nofendian17/sbterm/libs/proto/securities/transactional/datafeed/v1"
)

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
			want: stockbitws.WSChannelRunningTradeBatch("*"),
		},
		{
			name: "selected order book v3",
			ch:   config.WSChannelConfig{OrderBookV3: []string{"BBCA", "BBRI"}},
			want: stockbitws.WSChannelOrderBookV3("BBCA", "BBRI"),
		},
		{
			name: "multiple channels in one subscription",
			ch: config.WSChannelConfig{
				RunningTradeBatch: []string{"*"},
				OrderBookV3:       []string{"BBCA"},
				LivepriceV3:       []string{"BBCA"},
			},
			want: stockbitws.MergeWSChannels(
				stockbitws.WSChannelRunningTradeBatch("*"),
				stockbitws.WSChannelOrderBookV3("BBCA"),
				stockbitws.WSChannelLivepriceV3("BBCA"),
			),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.True(t, proto.Equal(tt.want, service.BuildChannel(tt.ch)))
		})
	}
}
