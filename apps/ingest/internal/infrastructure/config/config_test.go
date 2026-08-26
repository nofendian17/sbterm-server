package config

import (
	"strings"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadDefaults(t *testing.T) {
	cfg, err := Load()
	assert.NoError(t, err)

	assert.Equal(t, "datafeed.running_trade_batch", cfg.Kafka.RunningTradeBatchTopic)
	assert.Equal(t, "ws::addr=localhost:9000;auto_flush_rows=100;connect_timeout=10000;", cfg.QuestDB.URL)
}

// TestLoadDetectionOverrides confirms the detection section parses from YAML
// and accepts both plain numbers and duration strings.
func TestLoadDetectionOverrides(t *testing.T) {
	v := viper.New()
	v.SetConfigType("yaml")
	const doc = `
detection:
  top_levels: 8
  cooldown: 30m
  pull:
    min_qty: 1000
    repeat_k: 3
    window: 15m
  iceberg:
    min_qty: 500
    n: 4
    uniformity_pct: 70
  accum:
    net_min: 50000000
    window: 10m
    mid_drift_max: 0.02
    confirm_for: 2m
    support_gamma: 0.8
  distrib:
    support_gamma: 0.9
`
	require.NoError(t, v.ReadConfig(strings.NewReader(doc)))

	var cfg Config
	require.NoError(t, v.Unmarshal(&cfg))

	assert.Equal(t, 8, cfg.Detection.TopLevels)
	assert.Equal(t, 30*time.Minute, cfg.Detection.Cooldown)
	assert.Equal(t, int64(1000), cfg.Detection.Pull.MinQty)
	assert.Equal(t, 3, cfg.Detection.Pull.RepeatK)
	assert.Equal(t, 15*time.Minute, cfg.Detection.Pull.Window)
	assert.Equal(t, int64(500), cfg.Detection.Iceberg.MinQty)
	assert.Equal(t, 4, cfg.Detection.Iceberg.N)
	assert.Equal(t, 70.0, cfg.Detection.Iceberg.UniformityPct)
	assert.Equal(t, 50000000.0, cfg.Detection.Accum.NetMin)
	assert.Equal(t, 10*time.Minute, cfg.Detection.Accum.Window)
	assert.Equal(t, 0.02, cfg.Detection.Accum.MidDriftMax)
	assert.Equal(t, 2*time.Minute, cfg.Detection.Accum.ConfirmFor)
	assert.Equal(t, 0.8, cfg.Detection.Accum.SupportGamma)
	assert.Equal(t, 0.9, cfg.Detection.Distrib.SupportGamma)
}
