package container

import (
	"context"
	"errors"
	"net/http"
	"os"
	"syscall"
	"time"

	"github.com/samber/do/v2"

	deliveryhttp "github.com/nofendian17/sbterm-server/internal/delivery/http"
	"github.com/nofendian17/sbterm-server/internal/infrastructure/cache"
	"github.com/nofendian17/sbterm-server/internal/infrastructure/config"
	"github.com/nofendian17/sbterm-server/internal/infrastructure/database"
	infraRepo "github.com/nofendian17/sbterm-server/internal/infrastructure/repository"
	"github.com/nofendian17/sbterm-server/internal/repository"
	"github.com/nofendian17/sbterm-server/internal/usecase"
	"github.com/nofendian17/sbterm-server/pkg/log"
)

func New(cfg *config.Config, logger log.Logger) *do.RootScope {
	injector := do.New()

	do.ProvideValue(injector, cfg)
	do.ProvideValue(injector, logger)

	do.Provide(injector, func(i do.Injector) (*database.Postgres, error) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return database.New(ctx, cfg.DatabaseURL,
			database.WithMaxConns(cfg.DBMaxConns),
			database.WithMinConns(cfg.DBMinConns),
			database.WithMaxConnLifetime(cfg.DBMaxConnLifetime),
			database.WithMaxConnIdleTime(cfg.DBMaxConnIdleTime),
		)
	})

	do.Provide(injector, func(i do.Injector) (*cache.Redis, error) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return cache.New(ctx, cfg.RedisURL,
			cache.WithMaxRetries(cfg.RedisMaxRetries),
			cache.WithPoolSize(cfg.RedisPoolSize),
			cache.WithMinIdleConns(cfg.RedisMinIdleConns),
			cache.WithDialTimeout(cfg.RedisDialTimeout),
			cache.WithReadTimeout(cfg.RedisReadTimeout),
			cache.WithWriteTimeout(cfg.RedisWriteTimeout),
		)
	})

	do.Provide(injector, func(i do.Injector) (*infraRepo.HealthRepository, error) {
		return infraRepo.NewHealthRepository(do.MustInvoke[*database.Postgres](i)), nil
	})
	do.MustAs[*infraRepo.HealthRepository, repository.HealthRepository](injector)

	do.Provide(injector, func(i do.Injector) (usecase.HealthUsecase, error) {
		return usecase.NewHealthUsecase(do.MustInvoke[repository.HealthRepository](i)), nil
	})

	do.Provide(injector, func(i do.Injector) (*deliveryhttp.HealthHandler, error) {
		return deliveryhttp.NewHealthHandler(do.MustInvoke[usecase.HealthUsecase](i)), nil
	})

	do.Provide(injector, func(i do.Injector) (*deliveryhttp.Server, error) {
		_ = do.MustInvoke[*cache.Redis](i)
		handler := do.MustInvoke[*deliveryhttp.HealthHandler](i)
		logger := do.MustInvoke[log.Logger](i)
		router := deliveryhttp.NewRouter(handler, logger,
			deliveryhttp.WithRateLimit(cfg.RateLimitRate, cfg.RateLimitBurst),
		)
		return deliveryhttp.NewServer(router,
			deliveryhttp.WithAddr(cfg.Port),
			deliveryhttp.WithReadTimeout(cfg.HTTPReadTimeout),
			deliveryhttp.WithWriteTimeout(cfg.HTTPWriteTimeout),
			deliveryhttp.WithIdleTimeout(cfg.HTTPIdleTimeout),
		), nil
	})

	return injector
}

func Run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	level, err := log.ParseLevel(cfg.LogLevel)
	if err != nil {
		return err
	}
	format, err := log.ParseFormat(cfg.LogFormat)
	if err != nil {
		return err
	}

	logger := log.New(
		log.WithLevel(level),
		log.WithFormat(format),
		log.WithAddSource(cfg.LogAddSource),
	)
	log.SetDefault(logger)

	injector := New(cfg, logger)
	server := do.MustInvoke[*deliveryhttp.Server](injector)

	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("http server failed", "error", err)
		}
	}()
	logger.Info("server started",
		"app", cfg.AppName,
		"version", cfg.AppVersion,
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
