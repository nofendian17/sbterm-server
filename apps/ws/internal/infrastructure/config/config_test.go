package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadDefaults(t *testing.T) {
	cfg, err := Load()
	assert.NoError(t, err)

	assert.Equal(t, "datafeed.running_trade_batch", cfg.Kafka.RunningTradeBatchTopic)
	assert.Equal(t, "datafeed.order_book", cfg.Kafka.OrderBookTopic)
	assert.Equal(t, "datafeed.best_bid_offer", cfg.Kafka.BestBidOfferTopic)
	assert.Equal(t, "datafeed.iepiev", cfg.Kafka.IepIevTopic)
	assert.Equal(t, "datafeed.liveprice", cfg.Kafka.LivePriceTopic)
	assert.Equal(t, "wss://wss-trading.stockbit.com/ws", cfg.Stockbit.WSURL)
}
