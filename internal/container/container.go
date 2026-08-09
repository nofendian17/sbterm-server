package container

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/samber/do/v2"
	"golang.org/x/sync/errgroup"

	deliveryhttp "github.com/nofendian17/sbterm-server/internal/delivery/http"
	"github.com/nofendian17/sbterm-server/internal/delivery/http/health"
	"github.com/nofendian17/sbterm-server/internal/delivery/http/mover"
	"github.com/nofendian17/sbterm-server/internal/delivery/http/trending"
	"github.com/nofendian17/sbterm-server/internal/infrastructure/cache"
	"github.com/nofendian17/sbterm-server/internal/infrastructure/config"
	"github.com/nofendian17/sbterm-server/internal/infrastructure/database"
	infraRepo "github.com/nofendian17/sbterm-server/internal/infrastructure/repository"
	"github.com/nofendian17/sbterm-server/internal/infrastructure/stockbit"
	"github.com/nofendian17/sbterm-server/internal/repository"
	"github.com/nofendian17/sbterm-server/internal/usecase"
	"github.com/nofendian17/sbterm-server/pkg/log"
	"github.com/nofendian17/sbterm-server/pkg/validator"
)

func New(cfg *config.Config, logger log.Logger) *do.RootScope {
	injector := do.New()

	do.ProvideValue(injector, cfg)
	do.ProvideValue(injector, logger)

	provideCommon(injector)
	provideInfrastructure(injector)
	provideStockbit(injector)
	provideRepositories(injector)
	provideUsecases(injector)
	provideHandlers(injector)

	return injector
}

// provideCommon registers shared cross-cutting providers such as the request
// validator used by HTTP handlers.
func provideCommon(injector *do.RootScope) {
	do.Provide(injector, func(i do.Injector) (validator.Validator, error) {
		return validator.New(), nil
	})
}

func provideInfrastructure(injector *do.RootScope) {
	do.Provide(injector, func(i do.Injector) (*database.Postgres, error) {
		cfg := do.MustInvoke[*config.Config](i)
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
		cfg := do.MustInvoke[*config.Config](i)
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
}

func provideRepositories(injector *do.RootScope) {
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

	do.Provide(injector, func(i do.Injector) (*infraRepo.TrendingRepository, error) {
		client, err := do.Invoke[*stockbit.Client](i)
		if err != nil {
			return nil, err
		}
		return infraRepo.NewTrendingRepository(client), nil
	})
	do.MustAs[*infraRepo.TrendingRepository, repository.TrendingRepository](injector)

	do.Provide(injector, func(i do.Injector) (*infraRepo.MarketMoverRepository, error) {
		client, err := do.Invoke[*stockbit.Client](i)
		if err != nil {
			return nil, err
		}
		return infraRepo.NewMarketMoverRepository(client), nil
	})
	do.MustAs[*infraRepo.MarketMoverRepository, repository.MarketMoverRepository](injector)
}

func provideStockbit(injector *do.RootScope) {
	do.Provide(injector, func(i do.Injector) (*stockbit.Refresher, error) {
		cfg := do.MustInvoke[*config.Config](i)
		logger := do.MustInvoke[log.Logger](i)

		opts := []stockbit.Option{
			stockbit.WithTimeout(cfg.Stockbit.Timeout),
			stockbit.WithRetryCount(cfg.Stockbit.RetryCount),
			stockbit.WithLogger(logger),
		}
		if cfg.Stockbit.BaseURL != "" {
			opts = append(opts, stockbit.WithBaseURL(cfg.Stockbit.BaseURL))
		}
		client := stockbit.New(opts...)

		cmd := do.MustInvoke[*cache.Redis](i).Cmdable()
		if cmd == nil {
			return nil, fmt.Errorf("container: redis client unavailable for token store")
		}
		store := stockbit.NewRedisTokenStore(cmd)
		refresher := stockbit.NewRefresher(client, store, stockbit.Credentials{
			PlayerID: cfg.Stockbit.PlayerID,
			Username: cfg.Stockbit.Username,
			Password: cfg.Stockbit.Password,
		}, logger)
		client.SetAuthenticator(refresher)
		return refresher, nil
	})

	do.Provide(injector, func(i do.Injector) (*stockbit.Client, error) {
		refresher, err := do.Invoke[*stockbit.Refresher](i)
		if err != nil {
			return nil, err
		}
		return refresher.Client(), nil
	})
}

func provideUsecases(injector *do.RootScope) {
	do.Provide(injector, func(i do.Injector) (usecase.HealthUsecase, error) {
		return usecase.NewHealthUsecase(do.MustInvoke[repository.HealthRepository](i)), nil
	})

	do.Provide(injector, func(i do.Injector) (usecase.TrendingUsecase, error) {
		return usecase.NewTrendingUsecase(do.MustInvoke[repository.TrendingRepository](i)), nil
	})

	do.Provide(injector, func(i do.Injector) (usecase.MarketMoverUsecase, error) {
		return usecase.NewMarketMoverUsecase(do.MustInvoke[repository.MarketMoverRepository](i)), nil
	})
}

func provideHandlers(injector *do.RootScope) {
	do.Provide(injector, func(i do.Injector) (*health.HealthHandler, error) {
		return health.NewHealthHandler(do.MustInvoke[usecase.HealthUsecase](i)), nil
	})

	do.Provide(injector, func(i do.Injector) (*trending.TrendingHandler, error) {
		return trending.NewTrendingHandler(do.MustInvoke[usecase.TrendingUsecase](i)), nil
	})

	do.Provide(injector, func(i do.Injector) (*mover.MarketMoverHandler, error) {
		return mover.NewMarketMoverHandler(do.MustInvoke[usecase.MarketMoverUsecase](i), do.MustInvoke[validator.Validator](i)), nil
	})

	do.Provide(injector, func(i do.Injector) (*deliveryhttp.Server, error) {
		cfg := do.MustInvoke[*config.Config](i)
		logger := do.MustInvoke[log.Logger](i)
		handler := do.MustInvoke[*health.HealthHandler](i)
		trendingHandler := do.MustInvoke[*trending.TrendingHandler](i)
		moverHandler := do.MustInvoke[*mover.MarketMoverHandler](i)

		router := deliveryhttp.NewRouter(handler, trendingHandler, moverHandler, logger,
			deliveryhttp.WithRateLimit(cfg.RateLimit.Rate, cfg.RateLimit.Burst),
		)
		return deliveryhttp.NewServer(router,
			deliveryhttp.WithAddr(cfg.Port),
			deliveryhttp.WithReadTimeout(cfg.HTTP.ReadTimeout),
			deliveryhttp.WithWriteTimeout(cfg.HTTP.WriteTimeout),
			deliveryhttp.WithIdleTimeout(cfg.HTTP.IdleTimeout),
		), nil
	})
}

type pinger interface {
	Ping(context.Context) error
}

func pingService[T pinger](ctx context.Context, injector *do.RootScope, name string) error {
	svc, err := do.Invoke[T](injector)
	if err != nil {
		return fmt.Errorf("container: resolve %s: %w", name, err)
	}
	if err := svc.Ping(ctx); err != nil {
		return fmt.Errorf("container: %s unreachable: %w", name, err)
	}
	return nil
}

// pingInfra verifies database and Redis connectivity before the server starts.
func pingInfra(injector *do.RootScope) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		return pingService[*database.Postgres](ctx, injector, "postgres")
	})
	g.Go(func() error {
		return pingService[*cache.Redis](ctx, injector, "redis")
	})

	return g.Wait()
}

func newLogger(cfg *config.Config) (log.Logger, error) {
	level, err := log.ParseLevel(cfg.Log.Level)
	if err != nil {
		return nil, err
	}
	format, err := log.ParseFormat(cfg.Log.Format)
	if err != nil {
		return nil, err
	}

	logger := log.New(
		log.WithLevel(level),
		log.WithFormat(format),
		log.WithAddSource(cfg.Log.AddSource),
	)
	log.SetDefault(logger)

	return logger, nil
}

func awaitShutdown(server *deliveryhttp.Server, injector *do.RootScope, logger log.Logger) error {
	errChan := make(chan error, 1)
	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errChan <- fmt.Errorf("http server failed: %w", err)
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, os.Interrupt)
	defer signal.Stop(sigChan)

	select {
	case err := <-errChan:
		logger.Error("server startup failed", "error", err)
		if report := injector.Shutdown(); !report.Succeed {
			logger.Error("container shutdown failed", "error", report)
		}
		return err
	case sig := <-sigChan:
		logger.Info("received shutdown signal", "signal", sig.String())
		if report := injector.Shutdown(); !report.Succeed {
			logger.Error("container shutdown failed", "error", report)
			return report
		}
	}

	logger.Info("server stopped")
	return nil
}

func Run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	logger, err := newLogger(cfg)
	if err != nil {
		return err
	}

	injector := New(cfg, logger)

	if err := pingInfra(injector); err != nil {
		return err
	}

	refresher, err := do.Invoke[*stockbit.Refresher](injector)
	if err != nil {
		return fmt.Errorf("container: construct stockbit refresher: %w", err)
	}
	refresher.Start()

	server, err := do.Invoke[*deliveryhttp.Server](injector)
	if err != nil {
		return fmt.Errorf("container: construct server: %w", err)
	}

	logger.Info("server started",
		"app", cfg.App.Name,
		"version", cfg.App.Version,
		"addr", cfg.Port,
	)

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		statuses := injector.HealthCheckWithContext(ctx)
		failed := 0
		for name, err := range statuses {
			if err != nil {
				failed++
				logger.Warn("service health check failed", "service", name, "error", err)
			}
		}
		logger.Info("health check complete", "services", len(statuses), "failed", failed)
	}()

	return awaitShutdown(server, injector, logger)
}
