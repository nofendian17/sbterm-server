package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadDefaults(t *testing.T) {
	cfg, err := Load()
	assert.NoError(t, err)

	assert.Equal(t, "datafeed.running_trade_batch", cfg.Kafka.RunningTradeBatchTopic)
	assert.Equal(t, "ws::addr=localhost:9000;auto_flush_rows=100;", cfg.QuestDB.URL)
}
