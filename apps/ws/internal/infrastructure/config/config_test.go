package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadDefaults(t *testing.T) {
	cfg, err := Load()
	assert.NoError(t, err)

	topics := cfg.Topics()
	assert.Equal(t, "datafeed.running_trade_batch", topics.RunningTradeBatch)
	assert.Equal(t, "datafeed.order_book", topics.OrderBook)
	assert.Equal(t, "wss://wss-trading.stockbit.com/ws", cfg.Stockbit.WSURL)
}
