// Package container wires the stream service's dependencies with samber/do
// and runs it until a shutdown signal arrives.
package container

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/samber/do/v2"

	"github.com/nofendian17/sbterm/apps/stream/internal/delivery/ws"
	"github.com/nofendian17/sbterm/apps/stream/internal/infrastructure/config"
	"github.com/nofendian17/sbterm/apps/stream/internal/infrastructure/kafka"
	"github.com/nofendian17/sbterm/apps/stream/internal/service"
	"github.com/nofendian17/sbterm/libs/pkg/log"
)

// New wires the Kafka consumer, the fan-out hub and loop, and the HTTP server
// into a samber/do root scope. Nothing dials at construction time: franz-go
// reconnects lazily, so a broker that is down does not block startup.
func New(cfg *config.Config, logger log.Logger) *do.RootScope {
	injector := do.New()
	do.ProvideValue(injector, cfg)
	do.ProvideValue(injector, logger)

	do.Provide(injector, func(i do.Injector) (*kafka.Consumer, error) {
		conf := do.MustInvoke[*config.Config](i)
		topics := []string{conf.Kafka.RunningTradeBatchTopic}
		return kafka.NewConsumer(conf.Kafka.Brokers, conf.Kafka.Group, topics)
	})

	do.Provide(injector, func(i do.Injector) (*service.Hub, error) {
		return service.NewHub(do.MustInvoke[log.Logger](i)), nil
	})

	do.Provide(injector, func(i do.Injector) (*service.Service, error) {
		conf := do.MustInvoke[*config.Config](i)
		consumer, err := do.Invoke[*kafka.Consumer](i)
		if err != nil {
			return nil, fmt.Errorf("container: construct consumer: %w", err)
		}
		hub := do.MustInvoke[*service.Hub](i)
		return service.NewService(consumer, hub, conf.Kafka.RunningTradeBatchTopic, do.MustInvoke[log.Logger](i)), nil
	})

	do.Provide(injector, func(i do.Injector) (*http.Server, error) {
		conf := do.MustInvoke[*config.Config](i)
		hub := do.MustInvoke[*service.Hub](i)
		return &http.Server{
			Addr:    conf.Port,
			Handler: ws.NewServer(hub).Router(),
		}, nil
	})

	return injector
}

// Run loads the config, wires the container, starts the fan-out loop and the
// HTTP server, and blocks until a shutdown signal arrives. Shutdown follows
// the spec order: stop the poll loop first (no more broadcasts), drain HTTP
// gracefully, release every remaining client writer, then close Kafka.
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

	consumer, err := do.Invoke[*kafka.Consumer](injector)
	if err != nil {
		return fmt.Errorf("container: construct kafka consumer: %w", err)
	}
	hub, err := do.Invoke[*service.Hub](injector)
	if err != nil {
		return fmt.Errorf("container: construct hub: %w", err)
	}
	svc, err := do.Invoke[*service.Service](injector)
	if err != nil {
		return fmt.Errorf("container: construct stream service: %w", err)
	}
	httpSrv, err := do.Invoke[*http.Server](injector)
	if err != nil {
		return fmt.Errorf("container: construct http server: %w", err)
	}

	svc.Start()

	serveErr := make(chan error, 1)
	go func() {
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serveErr <- fmt.Errorf("container: http server: %w", err)
		}
	}()
	logger.Info("stream started",
		"port", cfg.Port,
		"group", cfg.Kafka.Group,
		"topic", cfg.Kafka.RunningTradeBatchTopic,
	)

	if err := waitInterruptOrServeFailure(logger, serveErr); err != nil {
		return err
	}
	// One shared deadline covers loop stop + HTTP drain + hub close: the
	// spec bounds TOTAL shutdown at ≤5 seconds, not per step.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := svc.Shutdown(ctx); err != nil {
		logger.Warn("stream: fan-out loop shutdown", "error", err)
	}
	if err := httpSrv.Shutdown(ctx); err != nil {
		logger.Warn("stream: http server shutdown", "error", err)
	}
	hub.Close()
	consumer.Close()
	if report := injector.Shutdown(); !report.Succeed {
		logger.Error("container shutdown failed", "error", report)
	}
	logger.Info("stream stopped")

	select {
	case err := <-serveErr:
		return err
	default:
		return nil
	}
}

// waitInterruptOrServeFailure blocks until SIGTERM/SIGINT arrives, or returns
// immediately when the HTTP server reports it cannot serve: a broken port must
// fail the process now instead of idling invisibly until someone sends a
// signal.
func waitInterruptOrServeFailure(logger log.Logger, serveErr <-chan error) error {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, os.Interrupt)
	defer signal.Stop(sigChan)

	select {
	case sig := <-sigChan:
		logger.Info("received shutdown signal", "signal", sig.String())
		return nil
	case err := <-serveErr:
		logger.Error("stream: http server failed", "error", err)
		return err
	}
}
