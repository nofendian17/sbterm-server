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

	assert.Equal(t, ":8081", cfg.Port)
	assert.Equal(t, []string{"localhost:29092"}, cfg.Kafka.Brokers)
	assert.Equal(t, "sbterm-stream", cfg.Kafka.Group)
	assert.Equal(t, "datafeed.running_trade_batch", cfg.Kafka.RunningTradeBatchTopic)
}

func TestLoadReadsOverrideFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.stream.yaml")
	content := "port: \":9999\"\nkafka:\n  brokers: [\"broker-a:9092\", \"broker-b:9092\"]\n  group: override-group\n  running_trade_batch_topic: other.topic\nlog:\n  level: debug\n  format: json\n  add_source: true\n"
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))

	wd, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(wd) })
	require.NoError(t, os.Chdir(dir))

	cfg, err := Load()
	require.NoError(t, err)

	assert.Equal(t, ":9999", cfg.Port)
	assert.Equal(t, []string{"broker-a:9092", "broker-b:9092"}, cfg.Kafka.Brokers)
	assert.Equal(t, "override-group", cfg.Kafka.Group)
	assert.Equal(t, "other.topic", cfg.Kafka.RunningTradeBatchTopic)
	assert.Equal(t, "debug", cfg.Log.Level)
	assert.Equal(t, "json", cfg.Log.Format)
	assert.True(t, cfg.Log.AddSource)
}
