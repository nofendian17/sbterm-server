package config

import (
	"errors"
	"time"

	"github.com/spf13/viper"
)

const (
	ConfigFileName = "config.ws"
	ConfigFileType = "yaml"
	ConfigFilePath = "."
)

// version is overridable at build time via:
// go build -ldflags "-X github.com/nofendian17/sbterm/apps/ws/internal/infrastructure/config.version=<tag>"
var version = "dev"

type Config struct {
	Stockbit StockbitConfig `mapstructure:"stockbit"`
	Redis    RedisConfig    `mapstructure:"redis"`
	Kafka    KafkaConfig    `mapstructure:"kafka"`
	Log      LogConfig      `mapstructure:"log"`
}

type StockbitConfig struct {
	BaseURL                   string                 `mapstructure:"base_url"`
	Timeout                   time.Duration          `mapstructure:"timeout"`
	RetryCount                int                    `mapstructure:"retry_count"`
	PlayerID                  string                 `mapstructure:"player_id"`
	Username                  string                 `mapstructure:"username"`
	Password                  string                 `mapstructure:"password"`
	WSURL                     string                 `mapstructure:"ws_url"`
	WSSubscriptions           []WSSubscriptionConfig `mapstructure:"ws_subscriptions"`
	WSPingInterval            time.Duration          `mapstructure:"ws_ping_interval"`
	WSReconnectBackoffInitial time.Duration          `mapstructure:"ws_reconnect_backoff_initial"`
	WSReconnectBackoffMax     time.Duration          `mapstructure:"ws_reconnect_backoff_max"`
}

type WSSubscriptionConfig struct {
	Name     string          `mapstructure:"name"`
	Channels WSChannelConfig `mapstructure:"channels"`
}

type WSChannelConfig struct {
	Watchlist         []string `mapstructure:"watchlist"`
	OrderBook         []string `mapstructure:"order_book"`
	RunningTrade      []string `mapstructure:"running_trade"`
	RunningTradeBatch []string `mapstructure:"running_trade_batch"`
	Liveprice         []string `mapstructure:"liveprice"`
	Iepiev            []string `mapstructure:"iepiev"`
	Intraday          []string `mapstructure:"intraday"`
	BestBidOffer      []string `mapstructure:"best_bid_offer"`
	LivepriceV3       []string `mapstructure:"liveprice_v3"`
	OrderBookV3       []string `mapstructure:"order_book_v3"`
	IntradayV3        []string `mapstructure:"intraday_v3"`
}

type RedisConfig struct {
	URL string `mapstructure:"url"`
}

type KafkaConfig struct {
	Brokers                []string `mapstructure:"brokers"`
	RunningTradeBatchTopic string   `mapstructure:"running_trade_batch_topic"`
	OrderBookTopic         string   `mapstructure:"order_book_topic"`
	BestBidOfferTopic      string   `mapstructure:"best_bid_offer_topic"`
	IepIevTopic            string   `mapstructure:"iepiev_topic"`
	LivePriceTopic         string   `mapstructure:"liveprice_topic"`
}

type LogConfig struct {
	Level     string `mapstructure:"level"`
	Format    string `mapstructure:"format"`
	AddSource bool   `mapstructure:"add_source"`
}

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
	v.SetDefault("stockbit.base_url", "https://exodus.stockbit.com")
	v.SetDefault("stockbit.timeout", 30*time.Second)
	v.SetDefault("stockbit.retry_count", 3)
	v.SetDefault("stockbit.ws_url", "wss://wss-trading.stockbit.com/ws")
	v.SetDefault("stockbit.ws_ping_interval", 30*time.Second)
	v.SetDefault("stockbit.ws_reconnect_backoff_initial", time.Second)
	v.SetDefault("stockbit.ws_reconnect_backoff_max", 30*time.Second)
	v.SetDefault("redis.url", "redis://localhost:6379/0")
	v.SetDefault("kafka.brokers", []string{"localhost:29092"})
	v.SetDefault("kafka.running_trade_batch_topic", "datafeed.running_trade_batch")
	v.SetDefault("kafka.order_book_topic", "datafeed.order_book")
	v.SetDefault("kafka.best_bid_offer_topic", "datafeed.best_bid_offer")
	v.SetDefault("kafka.iepiev_topic", "datafeed.iepiev")
	v.SetDefault("kafka.liveprice_topic", "datafeed.liveprice")
	v.SetDefault("log.level", "info")
	v.SetDefault("log.format", "text")
	v.SetDefault("log.add_source", false)
}
