package container

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"syscall"
	"time"

	"github.com/samber/do/v2"

	deliveryhttp "github.com/nofendian17/sbterm-server/internal/delivery/http"
	"github.com/nofendian17/sbterm-server/internal/delivery/http/health"
	"github.com/nofendian17/sbterm-server/internal/infrastructure/cache"
	"github.com/nofendian17/sbterm-server/internal/infrastructure/config"
	"github.com/nofendian17/sbterm-server/internal/infrastructure/database"
	infraRepo "github.com/nofendian17/sbterm-server/internal/infrastructure/repository"
	"github.com/nofendian17/sbterm-server/internal/repository"
	"github.com/nofendian17/sbterm-server/internal/usecase"
	"github.com/nofendian17/sbterm-server/pkg/log"
)

func New(cfg *config.Config, logger log.Logger) (*do.RootScope, error) {
	injector := do.New()

	do.ProvideValue(injector, cfg)
	do.ProvideValue(injector, logger)

	do.Provide(injector, func(i do.Injector) (*database.Postgres, error) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return database.New(ctx, cfg.Database.URL,
			database.WithMaxConns(cfg.Database.MaxConns),
			database.WithMinConns(cfg.Database.MinConns),
			database.WithMaxConnLifetime(cfg.Database.MaxConnLifetime),
			database.WithMaxConnIdleTime(cfg.Database.MaxConnIdleTime),
		)
	})

	do.Provide(injector, func(i do.Injector) (*cache.Redis, error) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return cache.New(ctx, cfg.Redis.URL,
			cache.WithMaxRetries(cfg.Redis.MaxRetries),
			cache.WithPoolSize(cfg.Redis.PoolSize),
			cache.WithMinIdleConns(cfg.Redis.MinIdleConns),
			cache.WithDialTimeout(cfg.Redis.DialTimeout),
			cache.WithReadTimeout(cfg.Redis.ReadTimeout),
			cache.WithWriteTimeout(cfg.Redis.WriteTimeout),
		)
	})

	// Construct infrastructure eagerly so that a malformed database or Redis
	// URL fails fast at startup instead of surfacing lazily on first use.
	// Both services are registered before this point, so the invoke below
	// materializes them and registers their health check / shutdown hooks.
	if _, err := do.Invoke[*database.Postgres](injector); err != nil {
		return nil, fmt.Errorf("container: construct postgres: %w", err)
	}
	if _, err := do.Invoke[*cache.Redis](injector); err != nil {
		return nil, fmt.Errorf("container: construct redis: %w", err)
	}

	do.Provide(injector, func(i do.Injector) (*infraRepo.TxManagerImpl, error) {
		return infraRepo.NewTxManager(do.MustInvoke[*database.Postgres](i)), nil
	})
	do.MustAs[*infraRepo.TxManagerImpl, repository.TxManager](injector)

	do.Provide(injector, func(i do.Injector) (*infraRepo.HealthRepository, error) {
		return infraRepo.NewHealthRepository(
			do.MustInvoke[*database.Postgres](i),
			do.MustInvoke[*cache.Redis](i),
		), nil
	})
	do.MustAs[*infraRepo.HealthRepository, repository.HealthRepository](injector)

	do.Provide(injector, func(i do.Injector) (usecase.HealthUsecase, error) {
		return usecase.NewHealthUsecase(do.MustInvoke[repository.HealthRepository](i)), nil
	})

	do.Provide(injector, func(i do.Injector) (*health.HealthHandler, error) {
		return health.NewHealthHandler(do.MustInvoke[usecase.HealthUsecase](i)), nil
	})

	do.Provide(injector, func(i do.Injector) (*deliveryhttp.Server, error) {
		handler := do.MustInvoke[*health.HealthHandler](i)
		logger := do.MustInvoke[log.Logger](i)
		router := deliveryhttp.NewRouter(handler, logger,
			deliveryhttp.WithRateLimit(cfg.RateLimit.Rate, cfg.RateLimit.Burst),
		)
		return deliveryhttp.NewServer(router,
			deliveryhttp.WithAddr(cfg.Port),
			deliveryhttp.WithReadTimeout(cfg.HTTP.ReadTimeout),
			deliveryhttp.WithWriteTimeout(cfg.HTTP.WriteTimeout),
			deliveryhttp.WithIdleTimeout(cfg.HTTP.IdleTimeout),
		), nil
	})

	return injector, nil
}

func Run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	level, err := log.ParseLevel(cfg.Log.Level)
	if err != nil {
		return err
	}
	format, err := log.ParseFormat(cfg.Log.Format)
	if err != nil {
		return err
	}

	logger := log.New(
		log.WithLevel(level),
		log.WithFormat(format),
		log.WithAddSource(cfg.Log.AddSource),
	)
	log.SetDefault(logger)

	injector, err := New(cfg, logger)
	if err != nil {
		return err
	}
	server := do.MustInvoke[*deliveryhttp.Server](injector)

	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("http server failed", "error", err)
		}
	}()
	logger.Info("server started",
		"app", cfg.App.Name,
		"version", cfg.App.Version,
		"addr", cfg.Port,
	)

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		statuses := injector.HealthCheckWithContext(ctx)
		for name, err := range statuses {
			if err != nil {
				logger.Warn("service health check failed", "service", name, "error", err)
			} else {
				logger.Info("service healthy", "service", name)
			}
		}
	}()

	injector.ShutdownOnSignals(syscall.SIGTERM, os.Interrupt)
	logger.Info("server stopped")

	return nil
}
