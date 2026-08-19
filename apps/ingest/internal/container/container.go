// Package container wires the ingest worker's dependencies with samber/do and
// runs it until a shutdown signal arrives.
package container

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/samber/do/v2"

	"github.com/nofendian17/sbterm/apps/ingest/internal/infrastructure/config"
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
		return questdb.New(ctx, cfg.QuestDB.URL, cfg.QuestDB.RunningTradesTable, logger)
	})

	do.Provide(injector, func(i do.Injector) (*kafka.Consumer, error) {
		return kafka.NewConsumer(cfg.Kafka.Brokers, cfg.Kafka.Group, kafkaTopicList(cfg.Kafka))
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
		}
		handler := service.NewFrameHandler(runningSink, topics, logger)
		return service.NewService(consumer, handler, logger), nil
	})

	return injector
}

// kafkaTopicList collects the configured topic names into the slice the
// Kafka consumer subscribes to.
func kafkaTopicList(kafkaCfg config.KafkaConfig) []string {
	return []string{kafkaCfg.RunningTradeBatchTopic}
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
