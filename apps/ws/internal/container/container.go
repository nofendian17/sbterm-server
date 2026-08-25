// Package container wires the ws worker's dependencies with samber/do and
// runs it until a shutdown signal arrives.
package container

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/redis/go-redis/v9"
	"github.com/samber/do/v2"

	"github.com/nofendian17/sbterm/apps/ws/internal/infrastructure/config"
	"github.com/nofendian17/sbterm/apps/ws/internal/infrastructure/kafka"
	stockbitws "github.com/nofendian17/sbterm/apps/ws/internal/infrastructure/stockbit"
	service "github.com/nofendian17/sbterm/apps/ws/internal/service"
	"github.com/nofendian17/sbterm/libs/pkg/log"
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
		for _, sub := range cfg.Stockbit.WSSubscriptions {
			ws := stockbitws.NewWSClient(cfg.Stockbit.WSURL, func(ctx context.Context) (string, error) {
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
			subs = append(subs, &service.Subscription{
				Name:    sub.Name,
				Client:  ws,
				Channel: service.BuildChannel(sub.Channels),
			})
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
