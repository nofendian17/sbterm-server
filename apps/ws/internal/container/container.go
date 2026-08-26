// Package container wires the ws worker's dependencies with samber/do and
// runs it until a shutdown signal arrives.
package container

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/redis/go-redis/v9"
	"github.com/samber/do/v2"

	"github.com/nofendian17/sbterm/apps/ws/internal/infrastructure/config"
	"github.com/nofendian17/sbterm/apps/ws/internal/infrastructure/kafka"
	stockbitws "github.com/nofendian17/sbterm/apps/ws/internal/infrastructure/stockbit"
	"github.com/nofendian17/sbterm/apps/ws/internal/infrastructure/symbols"
	service "github.com/nofendian17/sbterm/apps/ws/internal/service"
	"github.com/nofendian17/sbterm/libs/pkg/log"
	datafeedv1 "github.com/nofendian17/sbterm/libs/proto/securities/transactional/datafeed/v1"
	"github.com/nofendian17/sbterm/libs/stockbit"
)

// New wires the refresher, the Kafka producer, and the datafeed websocket
// service into a samber/do root scope.
func New(cfg *config.Config, logger log.Logger) *do.RootScope {
	injector := do.New()
	do.ProvideValue(injector, cfg)
	do.ProvideValue(injector, logger)

	do.Provide(injector, func(i do.Injector) (*stockbit.Refresher, error) {
		opts := []stockbit.Option{
			stockbit.WithTimeout(cfg.Stockbit.Timeout),
			stockbit.WithRetryCount(cfg.Stockbit.RetryCount),
			stockbit.WithLogger(logger),
		}
		if cfg.Stockbit.BaseURL != "" {
			opts = append(opts, stockbit.WithBaseURL(cfg.Stockbit.BaseURL))
		}
		client := stockbit.New(opts...)

		rdb, err := redisClient(cfg)
		if err != nil {
			return nil, err
		}
		store := stockbit.NewRedisTokenStore(rdb)
		refresher := stockbit.NewRefresher(client, store, stockbit.Credentials{
			PlayerID: cfg.Stockbit.PlayerID,
			Username: cfg.Stockbit.Username,
			Password: cfg.Stockbit.Password,
		}, logger)
		client.SetAuthenticator(refresher)
		return refresher, nil
	})

	do.Provide(injector, func(i do.Injector) (*kafka.Producer, error) {
		return kafka.NewProducer(cfg.Kafka.Brokers, logger)
	})

	do.Provide(injector, func(i do.Injector) (*service.Service, error) {
		refresher, err := do.Invoke[*stockbit.Refresher](i)
		if err != nil {
			return nil, err
		}
		publisher, err := do.Invoke[*kafka.Producer](i)
		if err != nil {
			return nil, err
		}

		subs := make([]*service.Subscription, 0, len(cfg.Stockbit.WSSubscriptions))
		var provider *symbols.Provider
		var staticChans []*datafeedv1.WebsocketChannel
		type dynamicEntry struct {
			sub     config.WSSubscriptionConfig
			channel stockbitws.ChannelProvider
		}
		var dynamics []dynamicEntry
		for _, sub := range cfg.Stockbit.WSSubscriptions {
			if len(sub.DynamicChannels) > 0 {
				if provider == nil {
					provider = symbols.New(cfg.Symbols.BaseURL, &http.Client{Timeout: cfg.Symbols.Timeout}, cfg.Symbols.CacheTTL)
				}
				dynamic := sub
				dynamics = append(dynamics, dynamicEntry{
					sub: dynamic,
					channel: func(ctx context.Context) (*datafeedv1.WebsocketChannel, error) {
						universe, err := provider.Symbols(ctx)
						if err != nil {
							return nil, fmt.Errorf("ws: resolve dynamic symbols: %w", err)
						}
						return service.BuildMicrostructureChannel(dynamic.DynamicChannels, universe)
					},
				})
				continue
			}
			staticChans = append(staticChans, service.BuildChannel(sub.Channels))
		}

		// Upstream allows ONE datafeed connection per account: parallel
		// connections kick each other off. All subscriptions therefore share
		// a single client; each becomes its own subscribe frame (auth once,
		// subscribe many — the frontend pattern), and the dynamic symbol set
		// is re-resolved on every reconnect plus the daily refresh.
		if len(dynamics) > 0 {
			first := dynamics[0]
			entry := &service.Subscription{
				Name:      first.sub.Name,
				Client:    wsClientFor(refresher, cfg, logger),
				ChannelFn: first.channel,
				RefreshAt: cfg.Symbols.RefreshTime,
			}
			if len(staticChans) > 0 {
				base := stockbitws.MergeWSChannels(staticChans...)
				entry.ExtraFns = append(entry.ExtraFns,
					func(context.Context) (*datafeedv1.WebsocketChannel, error) { return base, nil })
			}
			for _, d := range dynamics[1:] {
				entry.ExtraFns = append(entry.ExtraFns, d.channel)
			}
			subs = append(subs, entry)
		} else {
			for i, ch := range staticChans {
				name := ""
				if i < len(cfg.Stockbit.WSSubscriptions) {
					name = cfg.Stockbit.WSSubscriptions[i].Name
				}
				subs = append(subs, &service.Subscription{
					Name:    name,
					Client:  wsClientFor(refresher, cfg, logger),
					Channel: ch,
				})
			}
		}

		router := service.NewFrameRouter(publisher, service.Topics{
			RunningTradeBatch: cfg.Kafka.RunningTradeBatchTopic,
			OrderBook:         cfg.Kafka.OrderBookTopic,
			BestBidOffer:      cfg.Kafka.BestBidOfferTopic,
			IepIev:            cfg.Kafka.IepIevTopic,
			LivePrice:         cfg.Kafka.LivePriceTopic,
		})
		return service.New(subs, refresher, router, logger), nil
	})

	return injector
}

// wsClientFor builds one datafeed websocket client with the standard
// credential, keepalive, and backoff options.
func wsClientFor(refresher *stockbit.Refresher, cfg *config.Config, logger log.Logger) *stockbitws.WSClient {
	return stockbitws.NewWSClient(cfg.Stockbit.WSURL, func(ctx context.Context) (string, error) {
		key, err := refresher.Client().GetWebSocketKey(ctx)
		if err != nil {
			return "", fmt.Errorf("ws: fetch websocket key: %w", err)
		}
		return key.Data.Key, nil
	},
		stockbitws.WithWSAccessTokenProvider(func(ctx context.Context) (string, error) {
			return refresher.EnsureToken(ctx)
		}),
		stockbitws.WithWSPingInterval(cfg.Stockbit.WSPingInterval),
		stockbitws.WithWSReconnectBackoff(cfg.Stockbit.WSReconnectBackoffInitial, cfg.Stockbit.WSReconnectBackoffMax),
		stockbitws.WithWSLogger(logger),
	)
}

// redisClient builds the go-redis client backing the token store. The store
// needs only the connection; connectivity failures surface lazily on refresh.
func redisClient(cfg *config.Config) (*redis.Client, error) {
	opt, err := redis.ParseURL(cfg.Redis.URL)
	if err != nil {
		return nil, fmt.Errorf("ws: parse redis url: %w", err)
	}
	return redis.NewClient(opt), nil
}

// Run loads the config, wires the container, starts the token refresher and
// the datafeed worker, and blocks until a shutdown signal arrives.
func Run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	level, lerr := log.ParseLevel(cfg.Log.Level)
	if lerr != nil {
		return lerr
	}
	format, ferr := log.ParseFormat(cfg.Log.Format)
	if ferr != nil {
		return ferr
	}
	logger := log.New(log.WithLevel(level), log.WithFormat(format), log.WithAddSource(cfg.Log.AddSource))
	log.SetDefault(logger)

	injector := New(cfg, logger)

	refresher, err := do.Invoke[*stockbit.Refresher](injector)
	if err != nil {
		return fmt.Errorf("container: construct stockbit refresher: %w", err)
	}
	refresher.Start()
	defer injector.Shutdown()

	if len(cfg.Stockbit.WSSubscriptions) == 0 {
		logger.Warn("stockbit ws_subscriptions is empty; no datafeed subscriptions")
	} else {
		wsSvc, err := do.Invoke[*service.Service](injector)
		if err != nil {
			return fmt.Errorf("container: construct stockbit ws service: %w", err)
		}
		wsSvc.Start()
		logger.Info("ws worker started", "subscriptions", len(cfg.Stockbit.WSSubscriptions))
	}

	return awaitSignal(injector, logger)
}

// awaitSignal blocks until SIGTERM or SIGINT, then shuts the container down
// gracefully. It reports a non-nil error when the container shutdown fails.
func awaitSignal(injector *do.RootScope, logger log.Logger) error {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, os.Interrupt)
	defer signal.Stop(sigChan)

	sig := <-sigChan
	logger.Info("received shutdown signal", "signal", sig.String())
	report := injector.Shutdown()
	if !report.Succeed {
		logger.Error("container shutdown failed", "error", report)
		return report
	}
	logger.Info("ws worker stopped")
	return nil
}
