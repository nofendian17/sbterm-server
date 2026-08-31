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

	deliveryhttp "github.com/nofendian17/sbterm/apps/core/internal/delivery/http"
	"github.com/nofendian17/sbterm/apps/core/internal/delivery/http/admin"
	authhandler "github.com/nofendian17/sbterm/apps/core/internal/delivery/http/auth"
	"github.com/nofendian17/sbterm/apps/core/internal/delivery/http/health"
	"github.com/nofendian17/sbterm/apps/core/internal/delivery/http/user"
	"github.com/nofendian17/sbterm/apps/core/internal/delivery/http/watchlist"
	"github.com/nofendian17/sbterm/apps/core/internal/infrastructure/cache"
	"github.com/nofendian17/sbterm/apps/core/internal/infrastructure/config"
	"github.com/nofendian17/sbterm/apps/core/internal/infrastructure/database"
	infraRepo "github.com/nofendian17/sbterm/apps/core/internal/infrastructure/repository"
	"github.com/nofendian17/sbterm/apps/core/internal/repository"
	"github.com/nofendian17/sbterm/apps/core/internal/usecase"
	"github.com/nofendian17/sbterm/libs/pkg/log"
)

func New(cfg *config.Config, logger log.Logger) *do.RootScope {
	injector := do.New()

	do.ProvideValue(injector, cfg)
	do.ProvideValue(injector, logger)

	provideInfrastructure(injector)
	provideRepositories(injector)
	provideUsecases(injector)
	provideHandlers(injector)

	return injector
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
	do.Provide(injector, func(i do.Injector) (repository.TxManager, error) {
		return infraRepo.NewTxManager(do.MustInvoke[*database.Postgres](i)), nil
	})

	do.Provide(injector, func(i do.Injector) (repository.HealthRepository, error) {
		return infraRepo.NewHealthRepository(
			do.MustInvoke[*database.Postgres](i),
			do.MustInvoke[*cache.Redis](i),
		), nil
	})

	do.Provide(injector, func(i do.Injector) (repository.UserRepository, error) {
		return infraRepo.NewUserRepository(do.MustInvoke[*database.Postgres](i).Querier()), nil
	})

	do.Provide(injector, func(i do.Injector) (repository.RBACRepository, error) {
		return infraRepo.NewRBACRepository(do.MustInvoke[*database.Postgres](i).Querier()), nil
	})

	do.Provide(injector, func(i do.Injector) (repository.WatchlistRepository, error) {
		return infraRepo.NewWatchlistRepository(do.MustInvoke[*database.Postgres](i).Querier()), nil
	})

	do.Provide(injector, func(i do.Injector) (repository.RefreshStore, error) {
		return infraRepo.NewRedisRefreshStore(do.MustInvoke[*cache.Redis](i).Client()), nil
	})

	do.Provide(injector, func(i do.Injector) (repository.PermissionCache, error) {
		return infraRepo.NewRedisPermissionCache(do.MustInvoke[*cache.Redis](i).Client()), nil
	})
}

func provideUsecases(injector *do.RootScope) {
	do.Provide(injector, func(i do.Injector) (*usecase.TokenService, error) {
		cfg := do.MustInvoke[*config.Config](i)
		store := do.MustInvoke[repository.RefreshStore](i)
		return usecase.NewTokenService(
			cfg.Auth.JWTSecret,
			cfg.Auth.AccessTokenTTL,
			cfg.Auth.RefreshTokenTTL,
			store,
		), nil
	})

	do.Provide(injector, func(i do.Injector) (usecase.HealthUsecase, error) {
		return usecase.NewHealthUsecase(do.MustInvoke[repository.HealthRepository](i)), nil
	})

	do.Provide(injector, func(i do.Injector) (usecase.AuthUsecase, error) {
		cfg := do.MustInvoke[*config.Config](i)
		return usecase.NewAuthUsecase(
			do.MustInvoke[repository.UserRepository](i),
			do.MustInvoke[*usecase.TokenService](i),
			do.MustInvoke[repository.TxManager](i),
			usecase.AuthConfig{
				DefaultUserTTL: cfg.Auth.DefaultUserTTL,
				BcryptCost:     cfg.Auth.BcryptCost,
			},
		), nil
	})

	do.Provide(injector, func(i do.Injector) (usecase.UserUsecase, error) {
		return usecase.NewUserUsecase(do.MustInvoke[repository.UserRepository](i)), nil
	})

	do.Provide(injector, func(i do.Injector) (usecase.WatchlistUsecase, error) {
		return usecase.NewWatchlistUsecase(do.MustInvoke[repository.WatchlistRepository](i)), nil
	})

	do.Provide(injector, func(i do.Injector) (usecase.RBACUsecase, error) {
		return usecase.NewRBACUsecase(
			do.MustInvoke[repository.RBACRepository](i),
			do.MustInvoke[repository.PermissionCache](i),
		), nil
	})

	do.Provide(injector, func(i do.Injector) (usecase.AdminUsecase, error) {
		return usecase.NewAdminUsecase(
			do.MustInvoke[repository.UserRepository](i),
			do.MustInvoke[usecase.RBACUsecase](i),
		), nil
	})
}

func provideHandlers(injector *do.RootScope) {
	do.Provide(injector, func(i do.Injector) (*health.HealthHandler, error) {
		return health.NewHealthHandler(do.MustInvoke[usecase.HealthUsecase](i)), nil
	})

	do.Provide(injector, func(i do.Injector) (*authhandler.AuthHandler, error) {
		return authhandler.NewAuthHandler(do.MustInvoke[usecase.AuthUsecase](i)), nil
	})

	do.Provide(injector, func(i do.Injector) (*user.UserHandler, error) {
		return user.NewUserHandler(do.MustInvoke[usecase.UserUsecase](i)), nil
	})

	do.Provide(injector, func(i do.Injector) (*watchlist.WatchlistHandler, error) {
		return watchlist.NewWatchlistHandler(do.MustInvoke[usecase.WatchlistUsecase](i)), nil
	})

	do.Provide(injector, func(i do.Injector) (*admin.AdminHandler, error) {
		return admin.NewAdminHandler(do.MustInvoke[usecase.AdminUsecase](i)), nil
	})

	// Server
	do.Provide(injector, func(i do.Injector) (*deliveryhttp.Server, error) {
		cfg := do.MustInvoke[*config.Config](i)
		logger := do.MustInvoke[log.Logger](i)

		tokenService := do.MustInvoke[*usecase.TokenService](i)
		userRepo := do.MustInvoke[repository.UserRepository](i)
		rbacUc := do.MustInvoke[usecase.RBACUsecase](i)

		router := deliveryhttp.NewRouter(deliveryhttp.Handlers{
			Health:    do.MustInvoke[*health.HealthHandler](i),
			Auth:      do.MustInvoke[*authhandler.AuthHandler](i),
			User:      do.MustInvoke[*user.UserHandler](i),
			Watchlist: do.MustInvoke[*watchlist.WatchlistHandler](i),
			Admin:     do.MustInvoke[*admin.AdminHandler](i),
		}, deliveryhttp.AuthDeps{
			Verifier: tokenService,
			Loader:   userRepo,
			Checker:  rbacUc,
		}, logger, do.MustInvoke[usecase.AuthUsecase](i),
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

	// Run migrations
	if err := database.RunMigrations(context.Background(), cfg.Database.URL); err != nil {
		logger.Error("migrations failed", "error", err)
		return err
	}
	logger.Info("migrations applied")

	injector := New(cfg, logger)

	if err := pingInfra(injector); err != nil {
		return err
	}

	server, err := do.Invoke[*deliveryhttp.Server](injector)
	if err != nil {
		return fmt.Errorf("container: construct server: %w", err)
	}

	logger.Info("server started",
		"app", cfg.App.Name,
		"version", cfg.App.Version,
		"addr", cfg.Port,
	)

	return awaitShutdown(server, injector, logger)
}
