package config

import (
	"time"

	"errors"

	"github.com/spf13/viper"
)

const (
	ConfigFileName = "config.ingest"
	ConfigFileType = "yaml"
	ConfigFilePath = "."
)

// version is overridable at build time via:
// go build -ldflags "-X github.com/nofendian17/sbterm/apps/ingest/internal/infrastructure/config.version=<tag>"
var version = "dev"

type Config struct {
	QuestDB  QuestDBConfig  `mapstructure:"questdb"`
	Redis    RedisConfig    `mapstructure:"redis"`
	HotState HotStateConfig `mapstructure:"hot_state"`
	Kafka    KafkaConfig    `mapstructure:"kafka"`
	Log      LogConfig      `mapstructure:"log"`
}

// RedisConfig points at the hot-state store.
type RedisConfig struct {
	URL string `mapstructure:"url"`
}

// HotStateConfig gates the live order book mirror in Redis. Disabled by
// default so the rollout can be staged (writer first, evaluator later).
type HotStateConfig struct {
	Enabled bool          `mapstructure:"enabled"`
	Prefix  string        `mapstructure:"prefix"`
	TTL     time.Duration `mapstructure:"ttl"`
}

type QuestDBConfig struct {
	URL                string `mapstructure:"url"`
	RunningTradesTable string `mapstructure:"running_trades_table"`
	OrderBookTable     string `mapstructure:"order_book_table"`
	BookTTLDays        int    `mapstructure:"book_ttl_days"`
}

// KafkaConfig names the Kafka topic for each pipeline explicitly, so the
// mapping does not depend on matching a hardcoded topic string; any topic
// name configured here is used as-is.
type KafkaConfig struct {
	Brokers                []string `mapstructure:"brokers"`
	Group                  string   `mapstructure:"group"`
	RunningTradeBatchTopic string   `mapstructure:"running_trade_batch_topic"`
}

type LogConfig struct {
	Level     string `mapstructure:"level"`
	Format    string `mapstructure:"format"`
	AddSource bool   `mapstructure:"add_source"`
}

// Load reads the config file over defaults, falling back to defaults when the
// file is absent.
func Load() (*Config, error) {
	v := viper.New()
	setDefaults(v)

	v.SetConfigName(ConfigFileName)
	v.SetConfigType(ConfigFileType)
	v.AddConfigPath(ConfigFilePath)

	if err := v.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if !errors.As(err, &notFound) {
			return nil, errors.New("config: read config file: " + err.Error())
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, errors.New("config: unmarshal: " + err.Error())
	}
	return &cfg, nil
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("questdb.url", "ws::addr=localhost:9000;auto_flush_rows=100;")
	v.SetDefault("questdb.running_trades_table", "running_trades")
	v.SetDefault("questdb.order_book_table", "ob_book")
	v.SetDefault("questdb.book_ttl_days", 30)
	v.SetDefault("redis.url", "redis://localhost:6379/0")
	v.SetDefault("hot_state.enabled", false)
	v.SetDefault("hot_state.prefix", "ob")
	v.SetDefault("hot_state.ttl", 24*time.Hour)
	v.SetDefault("kafka.brokers", []string{"localhost:29092"})
	v.SetDefault("kafka.group", "sbterm-ingest")
	v.SetDefault("kafka.running_trade_batch_topic", "datafeed.running_trade_batch")
	v.SetDefault("log.level", "info")
	v.SetDefault("log.format", "text")
	v.SetDefault("log.add_source", false)
}
