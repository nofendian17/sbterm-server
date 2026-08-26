package config

import (
	"errors"

	"github.com/spf13/viper"
)

const (
	ConfigFileName = "config.stream"
	ConfigFileType = "yaml"
	ConfigFilePath = "."
)

// version is overridable at build time via:
// go build -ldflags "-X github.com/nofendian17/sbterm/apps/stream/internal/infrastructure/config.version=<tag>"
var version = "dev"

type Config struct {
	Port  string      `mapstructure:"port"`
	Kafka KafkaConfig `mapstructure:"kafka"`
	Log   LogConfig   `mapstructure:"log"`
}

type KafkaConfig struct {
	Brokers                []string `mapstructure:"brokers"`
	Group                  string   `mapstructure:"group"`
	RunningTradeBatchTopic string   `mapstructure:"running_trade_batch_topic"`
	OrderBookTopic         string   `mapstructure:"order_book_topic"`
	AlertsTopic            string   `mapstructure:"alerts_topic"`
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
	v.SetDefault("app.name", "sbterm-server")
	v.SetDefault("app.version", version)
	v.SetDefault("port", ":8081")
	v.SetDefault("kafka.brokers", []string{"localhost:29092"})
	v.SetDefault("kafka.group", "sbterm-stream")
	v.SetDefault("kafka.running_trade_batch_topic", "datafeed.running_trade_batch")
	v.SetDefault("kafka.order_book_topic", "datafeed.order_book")
	v.SetDefault("kafka.alerts_topic", "datafeed.alerts")
	v.SetDefault("log.level", "info")
	v.SetDefault("log.format", "text")
	v.SetDefault("log.add_source", false)
}
