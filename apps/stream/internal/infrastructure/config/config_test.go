package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadDefaults(t *testing.T) {
	cfg, err := Load()
	assert.NoError(t, err)

	assert.Equal(t, ":8081", cfg.Port)
	assert.Equal(t, []string{"localhost:29092"}, cfg.Kafka.Brokers)
	assert.Equal(t, "sbterm-stream", cfg.Kafka.Group)
	assert.Equal(t, "datafeed.running_trade_batch", cfg.Kafka.RunningTradeBatchTopic)
}
