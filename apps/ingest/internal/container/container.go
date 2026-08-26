// Package container wires the ingest worker's dependencies with samber/do and
// runs it until a shutdown signal arrives.
package container

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/samber/do/v2"

	"github.com/nofendian17/sbterm/apps/ingest/internal/detection"
	"github.com/nofendian17/sbterm/apps/ingest/internal/infrastructure/config"
	"github.com/nofendian17/sbterm/apps/ingest/internal/infrastructure/hotstate"
	"github.com/nofendian17/sbterm/apps/ingest/internal/infrastructure/kafka"
	"github.com/nofendian17/sbterm/apps/ingest/internal/infrastructure/questdb"
	"github.com/nofendian17/sbterm/apps/ingest/internal/service"
	"github.com/nofendian17/sbterm/libs/pkg/log"
)

// New wires the QuestDB client, the Kafka consumer, and the ingest service
// into a samber/do root scope.
func New(cfg *config.Config, logger log.Logger) *do.RootScope {
	injector := do.New()
	do.ProvideValue(injector, cfg)
	do.ProvideValue(injector, logger)

	do.Provide(injector, func(i do.Injector) (*questdb.Client, error) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		client, err := questdb.New(ctx, cfg.QuestDB.URL, cfg.QuestDB.RunningTradesTable, logger)
		if err != nil {
			return nil, err
		}
		return client.UseOrderBookTable(cfg.QuestDB.OrderBookTable, cfg.QuestDB.BookTTLDays), nil
	})

	do.Provide(injector, func(i do.Injector) (*kafka.Consumer, error) {
		return kafka.NewConsumer(cfg.Kafka.Brokers, cfg.Kafka.Group, kafkaTopicList(cfg.Kafka))
	})

	// Hot-state chain: redis client -> store -> book pipeline. Each layer is
	// nil when bandarmology is disabled, and the pipeline implements
	// do.ShutdownerWithContextAndError so injector.Shutdown() stops its
	// workers automatically.
	do.Provide(injector, func(i do.Injector) (*redis.Client, error) {
		if !cfg.HotState.Enabled {
			return nil, nil
		}
		return redisClient(cfg)
	})

	do.Provide(injector, func(i do.Injector) (*hotstate.Store, error) {
		rdb, err := do.Invoke[*redis.Client](i)
		if err != nil {
			return nil, fmt.Errorf("container: hot state redis: %w", err)
		}
		if rdb == nil {
			return nil, nil
		}
		return hotstate.NewStore(rdb, cfg.HotState.Prefix, cfg.HotState.TTL), nil
	})

	do.Provide(injector, func(i do.Injector) (service.BookPipeline, error) {
		store, err := do.Invoke[*hotstate.Store](i)
		if err != nil {
			return nil, fmt.Errorf("container: hot state store: %w", err)
		}
		if store == nil {
			return nil, nil
		}
		qdbClient, err := do.Invoke[*questdb.Client](i)
		if err != nil {
			return nil, fmt.Errorf("container: construct questdb client: %w", err)
		}

		var bookSink service.BookPersister
		if cfg.QuestDB.OrderBookTable != "" {
			persister, err := qdbClient.NewOrderBookSink(context.Background())
			if err != nil {
				return nil, fmt.Errorf("container: borrow order book sink: %w", err)
			}
			bookSink = persister
		}

		pipe, err := service.NewBookPipeline(service.BookDeps{
			Store:     store,
			Persister: bookSink,
			Logger:    logger,
			EngineFactory: func() *detection.Engine {
				return detection.NewEngine(detection.DefaultConfig(), alertSink(cfg, logger))
			},
			Workers:            6,
			MinPersistInterval: 500 * time.Millisecond,
		})
		if err != nil {
			return nil, fmt.Errorf("container: construct book pipeline: %w", err)
		}
		return pipe, nil
	})

	do.Provide(injector, func(i do.Injector) (*service.Service, error) {
		qdbClient, err := do.Invoke[*questdb.Client](i)
		if err != nil {
			return nil, fmt.Errorf("container: construct questdb client: %w", err)
		}
		consumer, err := do.Invoke[*kafka.Consumer](i)
		if err != nil {
			return nil, fmt.Errorf("container: construct kafka consumer: %w", err)
		}

		runningSink, err := qdbClient.NewRunningTradeBatchSink(context.Background())
		if err != nil {
			return nil, fmt.Errorf("container: borrow running trade sink: %w", err)
		}

		topics := service.Topics{
			RunningTradeBatch: cfg.Kafka.RunningTradeBatchTopic,
			OrderBook:         cfg.Kafka.OrderBookTopic,
			BestBidOffer:      cfg.Kafka.BestBidOfferTopic,
			IepIev:            cfg.Kafka.IepIevTopic,
			LivePrice:         cfg.Kafka.LivePriceTopic,
		}

		var opts []service.HandlerOption
		store, err := do.Invoke[*hotstate.Store](i)
		if err != nil {
			return nil, fmt.Errorf("container: resolve hot state store: %w", err)
		}
		pipe, err := do.Invoke[service.BookPipeline](i)
		if err != nil {
			return nil, fmt.Errorf("container: resolve book pipeline: %w", err)
		}
		if store != nil && pipe != nil {
			opts = append(opts,
				service.WithBookPipeline(pipe),
				service.WithLiveness(store),
				service.WithTradeObserver(pipe),
			)
			logger.Info("bandarmology pipeline enabled (shadow mode)", "prefix", cfg.HotState.Prefix)
		}

		handler := service.NewFrameHandler(runningSink, topics, logger, opts...)
		return service.NewService(consumer, handler, logger), nil
	})

	return injector
}

// alertSink returns the detector's output sink. When hot_state.alerts_topic
// is set, alerts are published to Kafka (for the stream fan-out) in addition
// to the always-on shadow log; otherwise this is log-only observation mode.
func alertSink(cfg *config.Config, logger log.Logger) detection.Sink {
	shadow := shadowSink{logger: logger}
	topic := cfg.HotState.AlertsTopic
	if topic == "" {
		return shadow
	}
	pub, err := kafka.NewPublisher(cfg.Kafka.Brokers, logger)
	if err != nil {
		logger.Error("container: alert publisher disabled, falling back to log-only", "error", err)
		return shadow
	}
	return alertKafkaSink{pub: pub, topic: topic, inner: shadow}
}

// alertKafkaSink publishes each alert as JSON and keeps the shadow log.
type alertKafkaSink struct {
	pub   *kafka.Publisher
	topic string
	inner detection.Sink
}

func (s alertKafkaSink) Emit(ctx context.Context, a detection.Alert) error {
	if err := s.inner.Emit(ctx, a); err != nil {
		return err
	}
	value, err := json.Marshal(a)
	if err != nil {
		return fmt.Errorf("container: marshal alert: %w", err)
	}
	return s.pub.Publish(ctx, s.topic, a.Symbol, value)
}

// shadowSink logs every signal instead of publishing it: the evaluator runs
// in observation mode until its thresholds are calibrated on real sessions.
type shadowSink struct {
	logger log.Logger
}

func (s shadowSink) Emit(_ context.Context, a detection.Alert) error {
	s.logger.Info("bandarmology signal",
		"symbol", a.Symbol, "type", string(a.Type), "side", a.Side,
		"ts", a.TS.Format(time.RFC3339Nano), "detail", a.Detail)
	return nil
}

// kafkaTopicList collects the configured topic names into the slice the
// Kafka consumer subscribes to. Auxiliary topics are always subscribed: when
// the hot state is disabled their frames are dropped at the handler.
func kafkaTopicList(kafkaCfg config.KafkaConfig) []string {
	return []string{
		kafkaCfg.RunningTradeBatchTopic,
		kafkaCfg.OrderBookTopic,
		kafkaCfg.BestBidOfferTopic,
		kafkaCfg.IepIevTopic,
		kafkaCfg.LivePriceTopic,
	}
}

// redisClient builds the go-redis client backing the hot-state store.
func redisClient(cfg *config.Config) (*redis.Client, error) {
	opt, err := redis.ParseURL(cfg.Redis.URL)
	if err != nil {
		return nil, fmt.Errorf("container: parse redis url: %w", err)
	}
	return redis.NewClient(opt), nil
}

// Run loads the config, wires the container, starts the ingest loop, and
// blocks until a shutdown signal arrives.
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
	defer injector.Shutdown()

	svc, err := do.Invoke[*service.Service](injector)
	if err != nil {
		return fmt.Errorf("container: construct ingest service: %w", err)
	}
	svc.Start()
	logger.Info("ingest started",
		"group", cfg.Kafka.Group,
		"topics", kafkaTopicList(cfg.Kafka),
		"questdb", cfg.QuestDB.URL,
	)

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
	logger.Info("ingest stopped")
	return nil
}
