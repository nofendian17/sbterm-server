package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	assert.Equal(t, "http://localhost:8080", cfg.Symbols.BaseURL)
	assert.Equal(t, "08:45", cfg.Symbols.RefreshTime)
}

func TestLoadDynamicChannels(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "config.ws.yaml")
	require.NoError(t, os.WriteFile(file, []byte(`
stockbit:
  ws_subscriptions:
    - name: microstructure_ihsg
      dynamic_channels: [order_book, liveprice, iepiev, best_bid_offer]
`), 0o644))

	wd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(dir))
	t.Cleanup(func() { _ = os.Chdir(wd) })

	cfg, err := Load()
	require.NoError(t, err)
	require.Len(t, cfg.Stockbit.WSSubscriptions, 1)
	assert.Equal(t, []string{"order_book", "liveprice", "iepiev", "best_bid_offer"},
		cfg.Stockbit.WSSubscriptions[0].DynamicChannels)
}
